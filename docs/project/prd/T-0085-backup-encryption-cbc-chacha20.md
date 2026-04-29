# PRD: backup AES-256-CBC + ChaCha20-Poly1305 加密算法实施（T-0085）

> **关联**: Backlog T-0085 / Sprint-08 / Domain=F06/backup+security / Type=feat / Prio=P3
> **作者**: Claude（代 Owner=电信+SecOps）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施
> **关键决策**: CBC + HMAC-SHA256 (encrypt-then-MAC) 防 padding-oracle；ChaCha20-Poly1305 走标准 AEAD；OENC v1 envelope 加 algo byte 'C'/'P'；既有 AES-GCM 路径完全不动（背向兼容）

---

## 1. 业务背景

T-0075 落 AES-256-GCM 信封加密 + 把 AES-CBC / ChaCha20-Poly1305 标记为 stub："请选择 AES-256-GCM 或关闭加密"。validatePolicy 拒绝 EnableEncryption=true + 这两个 algo。

业务驱动：
- **CBC**：部分企业合规（等保 2.0 / 中国密码法）要求支持国密以外的对称分组算法；CBC 是 AES 的"块模式"代表
- **ChaCha20-Poly1305**：移动端部署友好（无 AES 硬件加速时性能优于 AES-GCM）；TLS 1.3 标配
- **运维选择权**：T-0075 框架已就位（Encryptor 接口 + factory），实施这两个算法是 mirror T-0077 lz4/bzip2 的结构演化

---

## 2. ULTRATHINK 决策

### 2.1 CBC 必须配 HMAC-SHA256（encrypt-then-MAC）

CBC alone **是已知不安全的**（padding-oracle attack 可逐字节解密）。安全 CBC 实施必须 encrypt-then-MAC：
1. 生成 IV (16B random)
2. PKCS7 pad plaintext → AES-256-CBC encrypt → ciphertext
3. HMAC-SHA256 over (algo_byte ‖ IV ‖ AAD ‖ ciphertext) using **separate** MAC key derived from DEK via HKDF

**Decrypt 顺序严格**：HMAC verify FIRST（constant-time `hmac.Equal`） → 仅 verify 通过才 decrypt。否则 padding-oracle 漏洞重新出现。

### 2.2 HKDF 派生 MAC key 不扩 wrapped_DEK

DEK 仍为 32B（AES-256 key）。CBC 路径多需 32B MAC key，方案：
- A. DEK 扩到 64B（[encryption_key 32B][mac_key 32B]）→ wrapped_DEK 增至 80B → 文件头变长
- B. **HKDF-SHA256(DEK, info="backup-cbc-mac")** → 32B MAC key，wrapped_DEK 不变

采纳 B。HKDF 是 RFC 5869 标准；Go `golang.org/x/crypto/hkdf` 已是 indirect dep（提升到 direct）。

### 2.3 ChaCha20-Poly1305 是 AEAD 标准

`golang.org/x/crypto/chacha20poly1305` 暴露 `cipher.AEAD` 接口，与 GCM 同 shape：
- 32B key, 12B nonce, 16B tag, 内置 AAD
- envelope 与 GCM 完全对称，仅 algo_byte = 'P'

dependency：`golang.org/x/crypto` 已在 go.mod（间接）；提升到 direct dep。

### 2.4 文件格式 — OENC v1 不变

| Algo | algoByte | extension() |
|------|----------|-------------|
| AES-256-GCM | 'G' | "enc" |
| **AES-256-CBC** | **'C'** | "enc" |
| **ChaCha20-Poly1305** | **'P'** | "enc" |

下载方按 magic+ver+algoByte 路由到对应解密分支。同 .enc 文件名兼容三算法。

### 2.5 KEK wrap DEK 仍 AES-256-GCM

KEK 是 32B 通用对称密钥，wrap DEK 不必复杂化。所有算法共享 KEK + AES-256-GCM wrap，wrapped_DEK 长度恒定 48B（32B + 16B GCM tag）。

### 2.6 backwards compatibility

既有 GCM 加密文件 100% 兼容 — 文件 algo_byte 'G' 走原 path。本任务仅加新分支，不改既有 paths.

### 2.7 KeyProvider 共用

EnableEncryption=true 任意 algo 都需 KEK 可用。validatePolicy 把 GCM 的 keyProvider.Available() 检查推广到所有三算法。

### 2.8 测试矩阵

mirror TestAESGCM_* 模式：
- TestAESCBC_RoundTrip (sizes 0/1/16/1023/1024/65536/1MB)
- TestAESCBC_TamperDetected (HMAC catches bit flip)
- TestAESCBC_WrongAAD (HMAC includes AAD)
- TestAESCBC_WrongKey
- TestChaCha20Poly1305_RoundTrip (同 size set)
- TestChaCha20Poly1305_TamperDetected
- TestChaCha20Poly1305_WrongAAD
- TestChaCha20Poly1305_WrongKey
- TestNonceUniqueness 扩到三算法

policy_service_test 翻 2 case（CBC + ChaCha20 reject → accept），policy_service.go validatePolicy 删 "not yet implemented" 分支。

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| 等保合规运维 | OMC 提供分组密码 AES-CBC 选项满足合规要求，配置 + KEK 设置后 backup 走 CBC+HMAC 路径 |
| 移动端运维 | ARM 服务器无 AES-NI，ChaCha20 性能更好；选 ChaCha20-Poly1305 |
| 后端开发 | Encryptor 接口已抽象，加 2 实现 + factory 路由即可；OENC v1 envelope 不变 |
| QA | T-0075 16 既有 case 零回归；新算法 mirror 同 5 维度（roundtrip / tamper / AAD / key / nonce uniqueness）|

---

## 4. 验收标准（GWT）

### V1 — AES-256-CBC roundtrip 全 size 通过
- **Given** KP with valid 32B KEK；plaintext sizes [0, 1, 16, 1023, 1024, 65536, 1MB]
- **When** Encrypt → Decrypt 同 AAD
- **Then** plaintext bit-identical recover；blob[5] == 'C'

### V2 — AES-256-CBC tamper 检测
- **Given** valid CBC blob
- **When** flip 1 byte 在 ciphertext 区段 → Decrypt
- **Then** ErrEncryptionAuthFailed（HMAC mismatch）

### V3 — AES-256-CBC AAD mismatch 检测
- **Given** Encrypt with AAD="A"
- **When** Decrypt with AAD="B"
- **Then** ErrEncryptionAuthFailed

### V4 — ChaCha20-Poly1305 roundtrip
- mirror V1 但 algo='P'

### V5 — ChaCha20-Poly1305 tamper 检测
- mirror V2

### V6 — ChaCha20-Poly1305 AAD mismatch 检测
- mirror V3

### V7 — Nonce uniqueness 三算法都通过
- 500 次 Encrypt 同 plaintext+AAD，全 ciphertext 不重复

### V8 — validatePolicy 接受三算法
- **Given** EnableEncryption=true + (CBC OR ChaCha20) + KP available
- **When** PUT /backup/policy
- **Then** 200 OK（既有 reject 分支已删除）

### V9 — KEK 不可用时三算法都 reject
- **Given** EnableEncryption=true + 任一 algo + KP not available
- **When** PUT /backup/policy
- **Then** 400 ErrInvalidInput

### V10 — backwards compat — GCM 文件未改
- **Given** 既有 16 TestAESGCM_* case
- **When** go test -race ./internal/backup/...
- **Then** 全 PASS（GCM path 零改动）

---

## 5. 运营商差异矩阵

无差异。三家运营商对加密算法选择无强制要求；运维侧按部署需求选。

---

## 6. 非目标

| # | 非目标 | 原因 |
|---|--------|------|
| N1 | 国密算法 (SM4 / SM2) | 当前不在合规要求；future 任务（如有需求） |
| N2 | RSA 等非对称加密 | KEK + per-file DEK 信封模式不需要 |
| N3 | encryption_algorithm 联动 backup compression（如 CBC 强制不压缩）| 算法选择正交 |
| N4 | UI alert_severity 默认 critical 当 EnableEncryption=true | T-0084 已 settled severity 字段 |

---

## 7. 依赖

| 依赖 | 状态 |
|------|------|
| T-0075 加密框架 | ✅ done — 复用 Encryptor 接口 + factory + KEK |
| `golang.org/x/crypto` | ✅ indirect 已在 go.mod；提升到 direct |
| `golang.org/x/crypto/chacha20poly1305` | ✅ 同上 |
| `golang.org/x/crypto/hkdf` | ✅ 同上 |

---

## 8. 度量

无新增 metric — 共用 T-0075 既有 `omc_backup_encrypted_total` + `omc_backup_encryption_errors_total{reason}`。三算法都贡献到同一 collector（与 algo 无关）。

---

## 9. 设计备忘（S2）

### 9.1 encryption.go 新增结构

```go
// cbcChainEncryptor: AES-256-CBC + HMAC-SHA256 (encrypt-then-MAC).
// HMAC key 由 DEK 经 HKDF-SHA256(info="backup-cbc-mac") 派生，避免扩 wrapped_DEK.
type cbcChainEncryptor struct{ kp KeyProvider }

func (c *cbcChainEncryptor) Encrypt(plaintext, aad []byte) ([]byte, error) { ... }
func (c *cbcChainEncryptor) Decrypt(blob, aad []byte) ([]byte, error)      { ... }
func (c *cbcChainEncryptor) Format() string                                 { return "AES-256-CBC" }
func (c *cbcChainEncryptor) Extension() string                              { return "enc" }

// chaCha20Encryptor: ChaCha20-Poly1305 AEAD.
type chaCha20Encryptor struct{ kp KeyProvider }
// 同 4 method 实施
```

### 9.2 NewEncryptor factory 扩展

```go
func NewEncryptor(algo string, kp KeyProvider) (Encryptor, error) {
    if kp == nil || !kp.Available() {
        return nil, ErrEncryptionKeyUnavailable
    }
    switch algo {
    case "AES-256-GCM":
        return &gcmEncryptor{kp: kp}, nil
    case "AES-256-CBC":
        return &cbcChainEncryptor{kp: kp}, nil   // T-0085
    case "ChaCha20-Poly1305":
        return &chaCha20Encryptor{kp: kp}, nil   // T-0085
    default:
        return nil, ErrEncryptionAlgorithmInvalid
    }
}
```

### 9.3 Algo byte 常量

```go
const (
    encAlgoGCM         byte = 'G'
    encAlgoCBC         byte = 'C'  // T-0085
    encAlgoChaCha20    byte = 'P'  // T-0085 (Poly1305)
)
```

### 9.4 policy_service.go 修改

删除 stub-reject 分支：
```go
case "AES-256-CBC", "ChaCha20-Poly1305":
    return fmt.Errorf("encryption_algorithm=%s not yet implemented...", ...)
```

改为：
```go
case "AES-256-GCM", "AES-256-CBC", "ChaCha20-Poly1305":
    if s.keyProvider != nil && !s.keyProvider.Available() {
        return fmt.Errorf("encryption_algorithm=%s enabled but encryption key not configured...", ...)
    }
```

### 9.5 PKCS7 padding helper

```go
func pkcs7Pad(data []byte, blockSize int) []byte {
    pad := blockSize - len(data)%blockSize
    return append(data, bytes.Repeat([]byte{byte(pad)}, pad)...)
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
    n := len(data)
    if n == 0 || n%blockSize != 0 {
        return nil, ErrEncryptionFormatInvalid
    }
    pad := int(data[n-1])
    if pad == 0 || pad > blockSize {
        return nil, ErrEncryptionFormatInvalid
    }
    for i := n - pad; i < n; i++ {
        if int(data[i]) != pad {
            return nil, ErrEncryptionFormatInvalid
        }
    }
    return data[:n-pad], nil
}
```

### 9.6 文件清单

修改：
- `omcgo/internal/backup/encryption.go` — +cbcChainEncryptor + chaCha20Encryptor + factory 扩展 + algo byte 常量 + PKCS7 helper
- `omcgo/internal/backup/encryption_test.go` — +mirror TestAESCBC_* / TestChaCha20_* + extend TestAESGCM_NonceUniqueness 到三算法
- `omcgo/internal/backup/policy_service.go` — 删除 stub-reject 分支 + 推广 KEK availability check
- `omcgo/internal/backup/policy_service_test.go` — 翻转 2 case (CBC + ChaCha20 reject → accept)
- `omcgo/go.mod` — `golang.org/x/crypto` indirect → direct (chacha20poly1305 + hkdf)

无新增文件。

### 9.7 安全自查清单

- [ ] HMAC verify 严格在 decrypt 之前（防 padding oracle）
- [ ] `hmac.Equal` constant-time（防 timing side-channel）
- [ ] HKDF info 与算法绑定（"backup-cbc-mac" 不与其他用途混用）
- [ ] PKCS7 unpad 验证 padding 合法性（防 malformed padding）
- [ ] 0 加密路径 log key bytes
- [ ] fail-closed — 加密失败不 fall through plaintext
- [ ] AAD 进入 HMAC（CBC 必需，AEAD 内置）
- [ ] DEK 即生即用，KEK 仅 wrap

---

## 10. 实施要点

预计工作量：M（约 1.5 人日）— 2 algorithm impl + 8 case 新测 + 2 case 翻转 + factory 路由

预计涉及模块：`omcgo/internal/backup/`（4 文件）+ `go.mod`

预计新增端点：无；预计新迁移：无；FE 影响：无（algo 已 select 字段）

---

## 11. 审批

| 角色 | 占位 | 日期 | 备注 |
|------|------|------|------|
| 产品 / 架构 / 数据 / 电信 / 安全 / QA | Claude | 2026-04-29 | mirror T-0077 模式；security-sensitive 但非新威胁面（既有 GCM 同等 |

---

## 12. 变更记录

| 日期 | 版本 | 摘要 | 作者 |
|------|------|------|------|
| 2026-04-29 | v1.0 | 初稿；ULTRATHINK 8 决策 | Claude |
