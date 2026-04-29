# PRD: backup 加密执行 — AES-256-GCM 信封加密（T-0075 / R-102 final core）

> **关联**: Backlog T-0075 / Sprint-07..08 / Domain=F06/backup+security / Type=feat
> **作者**: Claude（代 Owner=电信+SecOps）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施
> **关键决策**: 信封加密 (KEK + 每文件 DEK) + EnvKeyProvider MVP + AES-256-GCM 单算法（CBC/ChaCha20 stub）+ buffer-then-encrypt 64MB 上限 + AAD=filename + 文件扩展 `.enc`
> **安全敏感**：本任务密钥处理 / 加密算法 / 文件格式涉及商用级运营商 OMC 数据保护；S5 阶段必须强制追加 `/security-review`

---

## 1. 业务背景

T-0071 持久化 19 字段含 `EnableEncryption / EncryptionAlgorithm`，UI 端打"⚠ 加密功能仅持久化偏好；executor 尚未实施加密"橙色 Alert 横幅 + PersistedOnlyTag。R-102 enforcement 路径上加密是**最后一块 core capability**：

- T-0073 cleanup ✅ / T-0074+T-0077 压缩 ✅ / T-0072+T-0078 restore ✅ / T-0079 task↔path 链路 ✅ / T-0076 物理删除 ✅
- 仅剩本任务（T-0075）。完成后 R-102 7 项 enforcement 100% 闭环

任务 backlog 标 **安全敏感** + Owner 含 SecOps — 设计与实施全程按生产级密码学规范执行。

---

## 2. ULTRATHINK 决策（8 关键问题）

### 2.1 Q1 — 加密层在压缩之后

industry 共识：encrypted data 高熵 → 压缩失效 → **compress-then-encrypt** 是正解。后续流水线：

```
plaintext (config XML) → compress (gzip/zstd/lz4/bzip2 via T-0074/T-0077)
                       → encrypt (AES-256-GCM, T-0075)
                       → MinIO PutObject
```

文件扩展 chain: `cfg.xml.gz.enc`（先 gzip 后 encrypt）。Restore 反向 peel：`.enc` 解密 → `.gz` 解压 → 服 XML。

### 2.2 Q2 — 信封加密 (KEK + 每文件 DEK)

| 方案 | 复杂度 | 旋转支持 | MVP 适配性 |
|------|--------|---------|----------|
| A 单 env-key | 低 | 否 | 简单但密钥泄露 = 全部历史泄露 |
| **B 信封加密 (KEK + DEK)** | 中 | 是（旋转 KEK，DEK 不动）| **采纳**：行业标准 + 后续 KMS 切换零业务侵入 |
| C 直接 KMS API call per file | 高 | 是 | OMC 部署不一定有 KMS infra |

**MVP 选 B**：
- KEK 32 字节（256-bit）由 `EnvKeyProvider` 从 `OMC_BACKUP_ENCRYPTION_KEY` 环境变量读取（hex string 64 字符）
- 每个 backup 文件生成新 32 字节 DEK（`crypto/rand`）
- DEK 用 KEK 通过 AES-256-GCM 包装为 wrapped_DEK
- Plaintext 用 DEK 通过 AES-256-GCM 加密为 ciphertext
- 文件包格式见 §2.5

**KeyProvider 接口预留 KMS 扩展点**：T-0086 (新登记) 可加 `KMSKeyProvider` 实现，业务代码无须改动。

### 2.3 Q3 — Buffer-then-encrypt（不是流式）

AES-GCM 单次 `Seal()` 一次性产生 auth tag — 不友好流式。备份文件实际大小：

- 典型 vendor config XML：< 1MB
- 极端：< 10MB
- 64MB 是合理上限（包含极端 cases + 防 OOM）

**MVP 决策**：upload handler 加 64MB cap（reject 超限，明确 error message 给运维）；< 64MB 全 buffer 后单次加密。后续若需 > 64MB 支持，T-0085 可加 chunked GCM 框架格式（chunk-by-chunk auth tag）。

### 2.4 Q4 — 算法范围（mirror T-0074/T-0077 stub 模式）

| 算法 | T-0075 实施 | 理由 |
|------|------------|------|
| **AES-256-GCM** | ✅ 完整 | AEAD; AES-NI 加速; stdlib `crypto/cipher` 内置；行业默认 |
| AES-256-CBC | ⏸ stub | 老旧；无 auth；padding-oracle 攻击风险；**不推荐使用** — 警告但拒绝 |
| ChaCha20-Poly1305 | ⏸ stub | 优秀 alternative；需 `golang.org/x/crypto/chacha20poly1305` 新 dep；defer T-0085 |

EnableEncryption=true + algo ∈ {CBC, ChaCha20} → service-layer reject（与 T-0074 lz4/bzip2 同模式）

### 2.5 Q5 — 文件格式

```
[magic "OENC" 4B][version 1B = 0x01][algo 1B = 'G'][reserved 2B = 0x0000]   ← header 8B
[outer_nonce 12B]                                                            ← KEK→DEK GCM nonce
[wrapped_DEK_len 4B uint32 LE]                                               ← 几乎总是 60
[wrapped_DEK ?B = AES-GCM(KEK, outer_nonce, DEK) = 32 + 16 = 48B]           ← actually 48B
[inner_nonce 12B]                                                            ← DEK→data GCM nonce
[ciphertext + 16B GCM tag]                                                   ← AAD bound
```

**总开销** = 8 + 12 + 4 + 48 + 12 + 16 = 100 字节 / 文件。对 1MB+ 备份可忽略。

magic `OENC` 让 download handler 检测加密文件（即使没有 `.enc` 扩展也能识别 — 防 ext 与 metadata 分歧；最终决策仍以扩展为权威）。

**Version 字节** = 0x01：今天的格式版本。后续若改为 chunked 流式，bumps to 0x02；解码前先看 version 决定路由。

### 2.6 Q6 — AAD 绑定 filename 防替换

AES-GCM 支持 Associated Data — 不加密但参与 auth 计算。设 **AAD = filename**（not full path — bucket 信息在 MinIO 上下文中）：

- 攻击者用 backup_file_A 替换 backup_file_B（同 KEK）→ 解密失败（filename 不匹配）
- 替换攻击在恶意 MinIO 访问场景重要（运营商 OMC 部署 MinIO 通常 trusted，但纵深防御）

Encrypt 端：filename 来自 upload URL `?filename=`；Decrypt 端：filename 来自 download object path basename — 两端一致即可解密。

**Edge case**：如果运维 manual 把 `cfg-A.xml.gz.enc` rename 为 `cfg-B.xml.gz.enc`（手 mv），decrypt 失败。文档化为已知行为（不应 manual rename 加密备份）。

### 2.7 Q7 — 威胁模型 + MVP 缓解

| 威胁 | 缓解 | 状态 |
|------|------|------|
| 密钥侧泄露 (env var 在 `/proc/<pid>/environ`) | OS-level ACL；env var 是 Linux 标准；运营商部署可用 systemd `EnvironmentFile=` + chmod 600 | accept |
| Nonce 重用（GCM 致命） | crypto/rand 96-bit nonce per file；2^48 后 50% 冲突概率 — 单 KEK 生命周期实际 < 10^6 文件 | safe |
| Tamper（位翻） | AES-GCM 16B auth tag；decrypt 失败 | safe |
| 替换攻击（A 换 B 文件名） | AAD=filename | mitigated |
| KEK 泄露 → 全部 backup 暴露 | KEK rotation = T-0087；MVP accept | doc |
| 历史 plaintext backup 仍在 MinIO | forward-only；老文件不强制重加密；T-0087 包含 backfill | doc |
| 关闭加密 → 老 .enc 不可解 | KEK 还在则可解；KEK 删则不可解；不在 EnableCompression toggle 范围内做强制 | doc |
| Side-channel timing | Go stdlib AES-GCM 常数时间（`crypto/subtle` for tag compare） | safe |
| Random 弱 | `crypto/rand` = `/dev/urandom` (Linux) | safe |

### 2.8 Q8 — UI 收尾

T-0071 留的 "⚠ 加密尚未生效" 橙色 Alert + PersistedOnlyTag(severity=warning)，本任务关闭：

- `EnableEncryption=false`：tag 隐藏（之前也是）
- `EnableEncryption=true && algorithm="AES-256-GCM"`：tag 改为 `severity="info"`（绿色）含 "已生效" 文案；移除 Alert 横幅
- `EnableEncryption=true && algorithm ∈ {CBC, ChaCha20}`：tag 保持 warning + Alert 含 "暂未实施 — 选 GCM 或关闭加密"

**多皮肤**：仅 `omcmb/webcode/src/pages/backup/BackupPolicy/index.tsx` 改；v2/v3 不引用本页面（T-0071 已确认）。

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| 网管运维 | 我希望 backup 文件能在 MinIO 桶里以加密形式存储；OMC 部署用 env var 配 32B 主密钥；选 AES-256-GCM 后 UI 告警提示消失 |
| SecOps | 加密文件格式必须是行业标准（信封 + AEAD）；nonce 不重用；密钥不写代码；老 plaintext 文件 forward-only |
| Go 开发者 | KeyProvider 是接口；Env 实现是 MVP；KMS 实现可后续 plug-in 不改业务 |
| QA | UI 切换"开/关加密"后下次 backup 落 MinIO 文件名带 `.enc`；restore 自动 decrypt；e2e 验证 service 校验路径 |

---

## 4. 验收标准（GWT）

### V1 — Encryptor round-trip (AES-256-GCM)
- **Given** 32 字节 KEK；任意 plaintext (1KB / 1MB / 10MB)
- **When** `Encrypt(plaintext, AAD=filename) → ciphertext`；`Decrypt(ciphertext, AAD=filename) → recovered`
- **Then** recovered == plaintext；ciphertext 长度 = plaintext + 100B；前 4 字节是 `OENC`

### V2 — Tamper detection
- **Given** 已加密文件
- **When** 任意一字节位翻 → Decrypt
- **Then** 返回 error；不输出部分明文

### V3 — Nonce uniqueness（统计）
- **Given** 同 KEK 加密 1000 个相同 plaintext 1000 次
- **Then** 1000 个 ciphertext 全互不相同（outer + inner nonce 都随机）

### V4 — Wrong AAD rejection
- **Given** 用 `filename=A.xml` 加密的文件
- **When** 用 `filename=B.xml` AAD 解密
- **Then** error；不输出明文

### V5 — Service-layer reject CBC / ChaCha20 when EnableEncryption=true
- **Given** PUT /backup/policy body `{enable_encryption:true, encryption_algorithm:"AES-256-CBC"}`
- **Then** 400；error 含 "not yet implemented; choose AES-256-GCM"

### V6 — Service-layer reject AES-256-GCM when KEK unavailable
- **Given** `OMC_BACKUP_ENCRYPTION_KEY` 未设置
- **When** PUT body `{enable_encryption:true, encryption_algorithm:"AES-256-GCM"}`
- **Then** 400；error 含 "encryption key not configured (set OMC_BACKUP_ENCRYPTION_KEY)"

### V7 — Upload handler 加密路径生效
- **Given** policy 启用 + KEK 配；CPE 上传 backup file size < 64MB
- **When** PUT 命中 `acs/upload/handler.go:ServeHTTP`
- **Then** MinIO 对象路径以 `.enc` 结尾；body 前 4 字节 `OENC`；metric `omc_backup_encrypted_total` +1

### V8 — Upload 64MB 上限
- **Given** EnableEncryption=true；上传 body Content-Length > 64MB
- **When** ServeHTTP 检查
- **Then** 400 / 413；error 明确指明加密分支限制

### V9 — Download handler 解密路径
- **Given** MinIO 对象 `cfg.xml.gz.enc`
- **When** CPE 调 download URL
- **Then** response body 解密 + 解压后 = 原 plaintext XML；Content-Disposition filename = `cfg.xml`（去 `.enc` 和 `.gz`）

### V10 — Frontend warning Tag 移除（AES-256-GCM 路径）
- **Given** EnableEncryption=true + algorithm=AES-256-GCM + KEK 已配
- **When** 加载 BackupPolicy 页
- **Then** encryption section Tag 显示 "已生效" info-color；不再有橙色 Alert 横幅

---

## 5. 运营商差异矩阵

无差异。加密属 OMC 内部存储保护；CPE / 运营商无感知。**本条整体一致。**

---

## 6. 非目标

| # | 非目标 | 后续承接 |
|---|--------|---------|
| N1 | AES-256-CBC + ChaCha20-Poly1305 算法实施 | T-0085（新登记，与 T-0086 解耦） |
| N2 | KMS / HSM KeyProvider | T-0086（新登记） |
| N3 | KEK rotation + 历史文件批量 re-encrypt | T-0087（新登记） |
| N4 | > 64MB backup 流式分块加密格式 | T-0085 candidate |
| N5 | KEK 多版本支持（KEK1 加密的老文件解密 + KEK2 加密新文件） | T-0087 |
| N6 | 加密失败 fallback 到 plaintext upload | **不做** — fail closed 是安全要求 |

---

## 7. 依赖

| 依赖 | 状态 |
|------|------|
| T-0071 (BackupPolicy schema) | ✅ done — 19 字段含 enable_encryption + encryption_algorithm |
| T-0074 + T-0077 (压缩管道) | ✅ done — 加密层位于其后 |
| T-0072 (download handler 解压) | ✅ done — 加 decrypt-then-decompress 链 |
| `crypto/aes` + `crypto/cipher` + `crypto/rand` | ✅ stdlib |

无新外部 dep。

---

## 8. 度量

| metric | 含义 |
|--------|------|
| `omc_backup_encrypted_total` | counter — 加密成功上传数 |
| `omc_backup_encryption_errors_total{reason}` | counter — reason ∈ {key_unavailable, oversize, encrypt_fail, format_invalid} |
| `omc_backup_decryption_errors_total{reason}` | counter — restore 路径上的 reason ∈ {wrong_aad, tamper, key_unavailable, format_invalid} |

复用 PolicyMetrics 命名空间，新增 3 collector + 3 Record helper。

---

## 9. 设计备忘（S2）

### 9.1 接口契约

`internal/backup/encryption.go`：
```go
var (
    ErrEncryptionKeyUnavailable        = errors.New("encryption key not configured")
    ErrEncryptionAlgorithmInvalid      = errors.New("encryption algorithm must be AES-256-GCM|AES-256-CBC|ChaCha20-Poly1305")
    ErrEncryptionAlgorithmNotImplemented = errors.New("encryption algorithm not implemented in current build")
    ErrEncryptionInputTooLarge         = errors.New("input exceeds encryption buffer ceiling")
    ErrEncryptionFormatInvalid         = errors.New("invalid encryption file format")
    ErrEncryptionAuthFailed            = errors.New("encryption auth tag verification failed (tampered or wrong key/AAD)")
)

type Encryptor interface {
    Encrypt(plaintext, aad []byte) ([]byte, error)
    Decrypt(ciphertext, aad []byte) ([]byte, error)
    Format() string  // "aes-256-gcm"
    Extension() string // ".enc"
}

func NewEncryptor(format string, kp KeyProvider) (Encryptor, error)
```

`internal/backup/key_provider.go`：
```go
type KeyProvider interface {
    KEK(ctx context.Context) ([]byte, error)
    Available() bool
}

type EnvKeyProvider struct { key []byte }
func NewEnvKeyProvider() *EnvKeyProvider // reads OMC_BACKUP_ENCRYPTION_KEY env var; absence → Available()=false
```

### 9.2 文件格式编码

```
const (
    encMagic     = "OENC"
    encVersion   = 0x01
    encAlgoGCM   = 'G'
    encNonceSize = 12  // GCM standard
    encDEKSize   = 32  // AES-256
    encTagSize   = 16  // GCM standard
    encMaxPlaintext = 64 * 1024 * 1024  // 64 MB cap
)
```

### 9.3 Upload handler 改动

`acs/upload/handler.go`：在 `maybeWrapForCompression` 之后加新阶段 `maybeEncrypt`：
```go
body, contentLength := r.Body, r.ContentLength
... compression ...

// T-0075: encryption layer (after compression). Buffers fully into memory
// up to 64MB ceiling; backup files in production are <10MB.
if cmp.applied || ft == tr069.FileTypeConfig {
    if h.encryptor != nil && h.policyEncryptionEnabled() {
        encryptedBytes, err := h.encryptThenBuffer(body, filename, contentLength)
        if err != nil {
            // metrics + http error path
        }
        body = bytes.NewReader(encryptedBytes)
        contentLength = int64(len(encryptedBytes))
        objectPath += ".enc"
        uploadOpts.ContentEncoding = uploadOpts.ContentEncoding + "+aes-256-gcm" // or set new field
    }
}
PutObject(ctx, bucket, objectPath, body, contentLength, uploadOpts)
```

注意：上述伪代码简化；实际需要 nil-safe + 64MB 上限检查在 buffer 前。

### 9.4 Download handler 改动

`acs/download/handler.go`：在 `detectCompression` 之前先 `detectEncryption`（按 `.enc` 后缀剥层）：

```go
// detectEncryption peels ".enc" suffix; returns Decryptor + cleanName + ok.
// Decryptor.Wrap reads cipher bytes fully → decrypts → returns plaintext reader.
```

链路：`MinIO obj → decrypt (buffered) → decompress (streaming) → ResponseWriter`。

### 9.5 PolicyService validate 加 2 路径

```go
if p.EnableEncryption {
    switch p.EncryptionAlgorithm {
    case "AES-256-GCM":
        if !s.keyProvider.Available() {
            return fmt.Errorf("encryption key not configured ...: %w", ErrInvalidInput)
        }
    case "AES-256-CBC", "ChaCha20-Poly1305":
        return fmt.Errorf("encryption_algorithm=%s not yet implemented (暂未支持) ...: %w", ...)
    }
}
```

注意：PolicyService 现在依赖 KeyProvider — 增加构造器参数 OR 后置 setter（与 backupHandler.SetPolicyService 同模式）。**采纳 setter** 避免破坏 NewPolicyService 既有签名。

### 9.6 DI 改动

`cmd/acs/main.go`：
```go
keyProvider := backup.NewEnvKeyProvider()
encryptor, _ := backup.NewEncryptor("AES-256-GCM", keyProvider)  // nil if KEK unavailable; handler nil-safe
uploadHandler.SetEncryption(encryptor)
downloadHandler.SetEncryption(encryptor) // download 复用同 KeyProvider
```

`cmd/app/provider/modules.go`：policyService.SetKeyProvider(keyProvider)（新 setter）。

注意：app 进程 **也需要** EnvKeyProvider（policyService validate at PUT time），但 ACS 进程才真正用于 encrypt/decrypt。两进程读同一 env var。

### 9.7 测试矩阵

| Layer | Test | 目标 GWT |
|-------|------|---------|
| encryption.go | TestAESGCM_RoundTrip 多 size | V1 |
| encryption.go | TestAESGCM_TamperDetected | V2 |
| encryption.go | TestAESGCM_NonceUnique（500 次循环检测唯一性） | V3 |
| encryption.go | TestAESGCM_AADBindsContext（A 加密 B 解密失败） | V4 |
| encryption.go | TestNewEncryptor_StubAlgorithms | (algo not impl) |
| encryption.go | TestNewEncryptor_KeyUnavailable | (key missing) |
| encryption.go | TestEncrypt_InputTooLarge_64MB | V8 |
| key_provider.go | TestEnvKeyProvider_HexParse + Available | (parse + nil-safe) |
| policy_service | TestValidatePolicy_CBCRejected | V5 |
| policy_service | TestValidatePolicy_KEKUnavailable | V6 |
| upload/handler | TestMaybeEncrypt_AppliedFlow | V7 |
| upload/handler | TestMaybeEncrypt_NoKeySkipped | (back-compat) |
| download/handler | TestDecryptThenDecompress | V9 |

### 9.8 Frontend 改动

`BackupPolicy/index.tsx`：
- `PersistedOnlyTag severity="warning"` → 条件化：`{algorithm === "AES-256-GCM" ? null : <PersistedOnlyTag severity="warning" />}`
- Alert 横幅同条件化
- 新 i18n key `backup.policy.encryptionRequiresKEK` (说明 KEK env var 配置)

### 9.9 文件清单

新增：
- `omcgo/internal/backup/encryption.go` (~200 行) — Encryptor 接口 + AES-GCM impl + 2 stub + 文件格式 encode/decode + 6 sentinel error
- `omcgo/internal/backup/encryption_test.go` (~200 行) — 7 case
- `omcgo/internal/backup/key_provider.go` (~70 行) — KeyProvider + EnvKeyProvider
- `omcgo/internal/backup/key_provider_test.go` (~50 行) — 3 case

修改：
- `omcgo/internal/backup/policy_service.go` — +keyProvider field + SetKeyProvider + validatePolicy 加 2 检查
- `omcgo/internal/backup/policy_service_test.go` — +V5 + V6 case
- `omcgo/internal/backup/policy_metrics.go` — +3 collector + 3 Record helper
- `omcgo/internal/acs/upload/handler.go` — +SetEncryption + maybeEncrypt + 64MB cap
- `omcgo/internal/acs/upload/handler_test.go` — +TestMaybeEncrypt 2 case
- `omcgo/internal/acs/download/handler.go` — +SetEncryption + decrypt-before-decompress
- `omcgo/internal/acs/download/decompress_test.go`（或新文件）— +decrypt round-trip
- `omcgo/cmd/acs/main.go` — DI EnvKeyProvider + Encryptor → upload/download handlers
- `omcgo/cmd/app/provider/modules.go` — DI keyProvider → policyService
- `omcgo/scripts/e2e_verify.sh` — +bk-12 (PUT policy CBC reject) + bk-13 (PUT GCM no-key reject)
- `omcmb/webcode/src/pages/backup/BackupPolicy/index.tsx` — 条件化 PersistedOnlyTag + Alert
- `omcmb/frontend-core/src/i18n/zh-CN/index.ts` + `en-US/index.ts` — +1-2 key

### 9.10 待定点

| 待定 | 决策 |
|------|------|
| KMS provider 接口预留点 | KeyProvider 已是接口 — KMSKeyProvider 未来 plug-in 即可 |
| 加密失败时 metric reason 粒度 | 4 类（key_unavailable / oversize / encrypt_fail / format_invalid）足够 dashboard |
| AAD 编码 — full path vs basename | basename 即可（同一文件 upload+download 都看到 basename） |
