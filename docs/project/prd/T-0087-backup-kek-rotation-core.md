# PRD: backup KEK 旋转 — envelope kek_id + multi-version KeyProvider（T-0087）

> **关联**: Backlog T-0087 / Sprint-08（提前 1 sprint，原 sprint-09）/ Domain=F06/backup+security / Type=feat / Prio=P2
> **作者**: Claude（代 Owner=电信+SecOps）
> **创建**: 2026-04-30
> **状态**: 草案 → 实施
> **关键决策**: T-0087-core = envelope OENC v2 加 kek_id + KeyProvider 多版本接口 + 多版本 EnvKeyProvider；CLI 批量 re-encrypt 工具 carve T-0092

---

## 1. 业务背景

T-0075 落 AES-256-GCM 信封加密，T-0085 加 CBC + ChaCha20-Poly1305 — 三算法
矩阵 ship。但**所有文件都用单一 KEK**（OMC_BACKUP_ENCRYPTION_KEY），没有
版本概念。生产场景下：

- **KEK 泄漏**：env var 被运维误打印 / 镜像意外推到公网 → 必须立即切换 KEK
- **合规要求**：等保 2.0 / PCI-DSS 要求加密密钥定期轮换（90 天 / 1 年）
- **运维窗口**：硬切换需要服务停机 + 全量解密重加密 — 不可接受

T-0086 落 KMSKeyProvider skeleton 时承诺"KEK 旋转 + envelope kek_id 留 T-0087
一并做"。本任务兑现承诺。

re-encrypt CLI 工具是真实运维所需，但属于"ops layer"独立可交付，carve T-0092。

---

## 2. ULTRATHINK 决策

### 2.1 envelope 加 kek_id 而非引入 KEKWrapper 接口

T-0086 caveat doc 提到"proper KMS-native flow = KEKWrapper"。本任务**仍**沿用
现有 `wrapDEKWithKEK` / `unwrapDEKWithKEK` 模式（local AES-256-GCM 在 plaintext
KEK 下 wrap DEK），原因：

| 方案 | 优 | 劣 |
|------|---|---|
| A. envelope 加 kek_id + 多版本 KeyProvider 找 KEK | 既有 wrap 逻辑零改动；3 算法 encode/decode 改动小；测试矩阵可控 | KEK 仍在进程内存（与 EnvKeyProvider/T-0086 KMSKeyProvider 同等弱点） |
| B. KEKWrapper 接口（每次 Wrap/Unwrap 远程调 KMS）| KEK 永不暴露明文 — proper KMS-native | encryption.go 大 refactor；3 算法每个 Encrypt/Decrypt 加远程调用；测试需要网络 mock；离线测试困难 |

**采纳 A**。理由：
- 真 KMS adapter 的接入面在 T-0091（trigger 运维凭据）— 那时再考虑 B 更合适
- T-0087 是密钥**旋转**而非密钥**外部化**；这两件事正交
- envelope 加 kek_id 同样让 B 路线在未来可平滑落地（kek_id 可作 KMSClient.Decrypt 的 keyId 参数）

KEKWrapper 仍是 T-0091 + T-0086 改造一并落地的目标。本任务不做 refactor。

### 2.2 OENC v2 envelope 格式

v1 layout 占用 byte 6-7 为 reserved（0x0000）。v2 重用 byte 6 作 `kek_id_len`：

```
v2 layout:
[magic "OENC" 4B][version=2 1B][algo 1B][kek_id_len 1B][reserved 1B 0x00]
[kek_id NB (0..255)]
[outer_nonce 12B]
[wrapped_DEK_len 4B][wrapped_DEK 48B]
[body algo-specific (G/C/P 同 v1)]
```

字节级布局：
- 0..3: magic "OENC"
- 4: version (1 = legacy / 2 = with kek_id)
- 5: algo (G/C/P)
- 6: **kek_id_len (u8)** — 0..255
- 7: reserved (0x00)
- 8..8+N-1: kek_id bytes (UTF-8 / ASCII，**not in any authenticated scope**)
- 之后同 v1

**version 显式 bump 到 2**：v1 解码器读 v2 文件应该报错（"version unsupported"），不应该静默误解析。当前 T-0085 decryptor 已有 version check，自动 reject v2 — 安全。

**kek_id_len = 0 时**：envelope 与 v1 格式仅 1 字节不同（byte 6 v1=0 reserved，v2=0 kek_id_len，相同值）。语义一致：empty kek_id → fall back 到 KEKByID("")。

### 2.3 kek_id NOT in AAD/HMAC 认证范围

风险评估：
- 攻击者篡改 envelope 的 kek_id 字段 → 解密器查不存在的 KEK → ErrEncryptionKeyUnavailable，拒绝
- 攻击者把 kek_id 改为他们已知的 KEK ID（如 v_old 已泄漏） → 用 v_old 解密 wrapped_DEK：
  - 但 wrapped_DEK 是用原 KEK 加密的，v_old 无法 GCM open → ErrEncryptionAuthFailed
  - 攻击者无法构造 v_old 下合法的 wrapped_DEK（除非完全重 forge envelope，那已经是 known-key compromise game）
- 结论：kek_id 不需要进 authenticated scope；现有 wrappedDEK 的 KEK-绑定 GCM tag 已经检测所有相关攻击

如果把 kek_id 进 AAD：
- v1 文件没有 kek_id → AAD = filename
- v2 文件有 kek_id → AAD = filename + kek_id
- decrypt 必须按 version 分支算 AAD，复杂度增加，价值有限

采纳：kek_id NOT in AAD。AAD 维持 filename（含 cmp.ext）。

### 2.4 KeyProvider 接口扩展

加两个方法：

```go
type KeyProvider interface {
    KEK(ctx context.Context) ([]byte, error)                          // legacy / 兼容路径
    Available() bool

    // T-0087 additions:
    ActiveKEKID() string                                              // current active key ID, "" for legacy single-key
    KEKByID(ctx context.Context, kekID string) ([]byte, error)        // lookup by ID
}
```

向后兼容：
- 既有调用方继续用 `KEK(ctx)`（语义 = "当前 active KEK"）
- 新调用方 encryption.go 改用 `KEKByID(ctx, envelope.kek_id)` 实现 multi-version 查询
- KEK(ctx) 实现 = `KEKByID(ctx, ActiveKEKID())`

3 个 KeyProvider impl 都实现新方法：
- **EnvKeyProvider**：见 §2.5 多版本支持
- **KMSKeyProvider**（T-0086）：ActiveKEKID() = client.KeyID()；KEKByID(ctx, id) 仅匹配自身 ID 时返回 cached key
- **staticKeyProvider**（test）：单键，ID = ""

### 2.5 EnvKeyProvider 多版本支持

新增两个 env var（向后兼容，单键部署不感知）：

| Env var | 含义 | 必填？ |
|---------|------|--------|
| `OMC_BACKUP_ENCRYPTION_KEY` | active key（64-hex） | 单键模式必填；多键模式 = active KEK |
| `OMC_BACKUP_ENCRYPTION_KEY_ID` | active key 的 ID（自由 string，最大 255 char）| 默认 ""；多键模式建议设（如 "v2"） |
| `OMC_BACKUP_ENCRYPTION_KEY_HISTORY` | 历史 KEK 列表，格式 `id1=hex;id2=hex;...` | 仅多键模式需要 |

行为：
- 单键模式（HISTORY 未设）：
  - keys = {KEY_ID: KEY}
  - ActiveKEKID() = KEY_ID（默认 ""）
  - KEKByID("") + KEKByID(KEY_ID) 都返回 KEY
  - 兼容 T-0075 部署：env 只设 KEY → ID = "" → keys[""] = KEY → 现有所有调用工作

- 多键模式（HISTORY 设了）：
  - keys = {KEY_ID: KEY, ...HISTORY}
  - ActiveKEKID() = KEY_ID
  - KEKByID(id) 查 keys 表
  - 历史 KEK 仅用于 read 旧文件；写永远用 active

- 校验：所有 hex 必须 64-char；HISTORY 解析失败构造返 error；KEY_ID 不能重复出现（active 与 history 互斥）

### 2.6 backwards compatibility

- T-0075 / T-0085 写的 v1 文件：decrypt 走 v1 路径（不解析 kek_id），用 `KEKByID(ctx, "")` 查 KEK
- 单键模式 EnvKeyProvider：keys[""] = single KEK → 兼容 100%
- 多键模式 EnvKeyProvider：旧文件假定属于 ""（legacy），但 keys[""] 可能不存在（如运维明确设了 KEY_ID="v1"）
  - **解决**：多键模式下，如果 KEY_ID 不为空，自动同时把 active key 注册到 keys[""]（让历史文件继续可读）OR
  - **更好**：构造时如果 HISTORY 不含 ""，且 KEY_ID 不为空，明确报错"legacy v1 files unreadable; set KEY_HISTORY to include legacy key with id=''"
  - 采纳：autopopulate keys[""] = active key 当 KEY_ID 不为空且 HISTORY 不含 ""（运维心智轻松；可读老文件）

### 2.7 测试矩阵

V1-V8（mirror T-0085 模式）：

- V1: encrypt+decrypt round-trip with kek_id="v2"（GCM/CBC/ChaCha 三算法）
- V2: backwards compat — 用 v1 envelope decrypt（empty kek_id → KEKByID("")）
- V3: rotation 场景 — encrypt under "v1"，switch active 到 "v2"，read v1 file 仍 OK
- V4: unknown kek_id → ErrEncryptionKeyUnavailable
- V5: kek_id 篡改 → 走 V4 路径或 wrong-DEK GCM 失败
- V6: empty kek_id（v2 with kek_id_len=0）round-trip
- V7: long kek_id（255 char）round-trip
- V8: EnvKeyProvider 单键 + 多键模式 env 解析正确

### 2.8 carve out: T-0092 re-encrypt CLI 工具

scope：
- CLI 入口 `omcctl backup re-encrypt --target-kek-id=v2 [--concurrency=4]`
- walk MinIO bucket（list + filter `.enc` files）
- 对每个文件：parse envelope，识别 kek_id；若 != target → decrypt + re-encrypt + replace
- 并发控制（mirror T-0089 semaphore）
- 进度报告 + metrics（`omc_backup_reencrypt_total{result}` + `omc_backup_reencrypt_duration_seconds`）
- 优雅取消（ctx）

est：M。trigger：T-0087 done 后即可启动；不需运维凭据（local KEK 多版本支持已就绪）。

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| SecOps（KEK 泄漏处理）| KEK 泄漏后 30 分钟内：生成 KEK_v2，配置 KEY=v2 + KEY_ID="v2" + HISTORY="v1=<v1-hex>"，重启 ACS — 新写入用 v2，旧文件仍可读 |
| 合规审计 | 90 天密钥轮换流程化：每季度生成新 KEK + 加入 active；旧 KEK 移到 HISTORY；CLI 工具批量 re-encrypt（T-0092）|
| 后端开发 | encryption.go envelope v2 升级后，对接业务无感（KeyProvider 接口扩展向后兼容）|
| QA | 既有 T-0075/T-0085 测试零回归；mixed bucket（v1+v2 文件）解密都通过 |
| 运维（首次部署）| 不设 HISTORY 时退化为 T-0075 单键模式 — 完全无感升级 |

---

## 4. 验收标准（GWT）

### V1 — kek_id="v2" round-trip 三算法

- **Given** EnvKeyProvider with active=v2，HISTORY=""，algos ∈ {GCM, CBC, ChaCha20}
- **When** Encrypt(plaintext, AAD) → Decrypt(blob, AAD)
- **Then** plaintext bit-identical recover；blob[4]=2，blob[6]=2 (kek_id_len)，blob[8..9]="v2"

### V2 — v1 backwards compat

- **Given** v1 envelope（既有 T-0075 写文件，blob[4]=1，byte 6=0）+ 单键 EnvKeyProvider
- **When** Decrypt(blob, AAD)
- **Then** plaintext recover；走 KEKByID("") 路径

### V3 — rotation scenario

- **Given** 步骤 1：active=v1 写文件 A；步骤 2：active=v2 + HISTORY="v1=<hex>"
- **When** decrypt A
- **Then** A 用 v1 KEK 解密成功；新文件 B 写入用 v2

### V4 — unknown kek_id

- **Given** envelope with kek_id="v999"，KeyProvider 无 v999
- **When** Decrypt
- **Then** ErrEncryptionKeyUnavailable

### V5 — kek_id 篡改不通过

- **Given** v2 envelope，flip 1 byte in kek_id → 变成不存在的 ID
- **When** Decrypt
- **Then** ErrEncryptionKeyUnavailable（或 ErrEncryptionAuthFailed 如果碰巧匹配另一 KEK 但 wrappedDEK 是原 KEK 的）

### V6 — empty kek_id v2 envelope

- **Given** v2 envelope written with active="" (legacy single-key new-style)
- **When** Decrypt
- **Then** plaintext recover；KEKByID("") 找到唯一键

### V7 — long kek_id (255B)

- **Given** kek_id 长度 255B
- **When** Encrypt → Decrypt
- **Then** round-trip OK；envelope kek_id_len=0xFF

### V8 — EnvKeyProvider env 解析

- **Given** env var 三组合（仅 KEY；KEY+KEY_ID；KEY+KEY_ID+HISTORY）
- **When** NewEnvKeyProvider
- **Then** keys map 大小 = 1 / 1 / N+1；ActiveKEKID 返回 KEY_ID（"" / "v2" / "v2"）

### V9 — HISTORY 解析失败

- **Given** HISTORY="v1==invalid hex"
- **When** NewEnvKeyProvider
- **Then** error；no provider；不破坏单键模式

### V10 — 既有 16 GCM + 8 CBC + 4 ChaCha20 测试零回归

- **Given** T-0075 + T-0085 既有测试套
- **When** go test -race ./internal/backup/...
- **Then** 全 PASS

---

## 5. 运营商差异矩阵

无差异。KEK 旋转策略由运维侧合规要求决定，OMC 不绑定运营商。

---

## 6. 非目标

| # | 非目标 | 原因 |
|---|--------|------|
| N1 | CLI 批量 re-encrypt 工具 | T-0092 carve（独立 ops layer）|
| N2 | 真 KMS（AWS/Vault/HSM）多版本适配器 | T-0091 真集成时一并做 |
| N3 | KEKWrapper 接口（每次 KMS round-trip）| T-0091 / 未来 refactor 时引入；本任务沿用 wrap helper |
| N4 | KEK 自动旋转（cron + KMS API 触发）| 运维侧自行决定；OMC 仅提供"重启即生效"机制 |
| N5 | kek_id 进 AAD/HMAC 认证范围 | §2.3 风险评估后不需要 |
| N6 | DB 持久化的 kek 元数据（版本时间戳 / 用途等）| 未来观测增强；不阻塞旋转能力 |

---

## 7. 依赖

| 依赖 | 状态 |
|------|------|
| T-0075 加密框架 | ✅ done |
| T-0085 三算法 | ✅ done |
| T-0086 KMSKeyProvider skeleton | ✅ done |

不引入第三方 dep。

---

## 8. 度量

T-0087-core 阶段无新 metric。T-0092（re-encrypt CLI）时计划加：

- `omc_backup_reencrypt_total{result}` — 文件级 re-encrypt 计数
- `omc_backup_reencrypt_duration_seconds` — 单文件 re-encrypt 延迟

---

## 9. 设计备忘（S2）

### 9.1 文件清单

修改：

- `omcgo/internal/backup/encryption.go` — bump version=2；解析 kek_id；3 算法 Encrypt/Decrypt 全升级
- `omcgo/internal/backup/key_provider.go` — KeyProvider 接口 +2 方法；EnvKeyProvider 多版本支持
- `omcgo/internal/backup/key_provider_kms.go` — KMSKeyProvider 实施新方法
- `omcgo/internal/backup/testhelpers_keyprovider_test.go` — staticKeyProvider 实施新方法
- `omcgo/internal/backup/encryption_test.go` — V1-V7 + V10 backwards compat
- `omcgo/internal/backup/key_provider_test.go` — V8/V9 多版本 env tests
- `omcgo/internal/backup/key_provider_kms_test.go` — minor 加 ActiveKEKID/KEKByID 检查

无新文件、无新 dep、无 schema、无 cmd DI 改动。

### 9.2 OENC v2 envelope encode/decode

```go
const encVersion2 = byte(0x02)

// Encrypt path (3 algos):
out = append(out, encMagic...)
out = append(out, encVersion2, algoByte, byte(len(kekID)), 0x00) // 4B
out = append(out, []byte(kekID)...)                              // N bytes
// ... rest same as v1

// Decrypt path:
ver := blob[4]
algo := blob[5]
var kekID string
var off int
switch ver {
case encVersion: // v1 = 0x01 — backwards compat
    if blob[6] != 0x00 || blob[7] != 0x00 {
        return nil, fmt.Errorf("v1 reserved bytes nonzero: %w", ErrEncryptionFormatInvalid)
    }
    kekID = ""
    off = 8
case encVersion2:
    kekIDLen := int(blob[6])
    if blob[7] != 0x00 {
        return nil, fmt.Errorf("v2 reserved byte nonzero: %w", ErrEncryptionFormatInvalid)
    }
    if 8+kekIDLen > len(blob) {
        return nil, fmt.Errorf("truncated kek_id: %w", ErrEncryptionFormatInvalid)
    }
    kekID = string(blob[8 : 8+kekIDLen])
    off = 8 + kekIDLen
default:
    return nil, fmt.Errorf("version=%d unsupported: %w", ver, ErrEncryptionFormatInvalid)
}

kek, err := e.kp.KEKByID(nil, kekID)
if err != nil { return nil, err }
// ... rest same as v1 (parse outer_nonce, wrapped_DEK, body)
```

3 算法各自的 Encrypt 把 ActiveKEKID() 写入；Decrypt 通用路径解析 kek_id 后用 KEKByID 查 KEK。

为减少代码重复，把 envelope 头解析提到 helper：

```go
type parsedHeader struct {
    algo  byte
    kekID string
    off   int  // body start
}

func parseEnvelopeHeader(blob []byte) (*parsedHeader, error) { ... }
```

### 9.3 EnvKeyProvider 多版本

```go
type EnvKeyProvider struct {
    activeID string
    keys     map[string][]byte // id → 32B AES-256 key
}

func NewEnvKeyProvider() (*EnvKeyProvider, error) {
    activeKey := os.Getenv(EnvBackupEncryptionKey)
    if activeKey == "" {
        return &EnvKeyProvider{keys: map[string][]byte{}}, nil // unavailable
    }
    activeID := os.Getenv(EnvBackupEncryptionKeyID)
    keys := map[string][]byte{}
    
    decoded, err := decodeKey(activeKey)
    if err != nil { return nil, err }
    keys[activeID] = decoded
    
    // History parsing
    history := os.Getenv(EnvBackupEncryptionKeyHistory)
    if history != "" {
        for _, entry := range strings.Split(history, ";") { ... }
    }
    
    // Backcompat: if activeID != "" and "" not in keys, autopopulate
    // keys[""] = decoded so legacy v1 files remain readable.
    if activeID != "" {
        if _, has := keys[""]; !has {
            keys[""] = decoded
        }
    }
    
    return &EnvKeyProvider{activeID: activeID, keys: keys}, nil
}

func (p *EnvKeyProvider) KEK(ctx context.Context) ([]byte, error) {
    return p.KEKByID(ctx, p.ActiveKEKID())
}

func (p *EnvKeyProvider) Available() bool {
    return p != nil && len(p.keys) > 0
}

func (p *EnvKeyProvider) ActiveKEKID() string {
    if p == nil { return "" }
    return p.activeID
}

func (p *EnvKeyProvider) KEKByID(_ context.Context, kekID string) ([]byte, error) {
    if !p.Available() {
        return nil, ErrEncryptionKeyUnavailable
    }
    key, ok := p.keys[kekID]
    if !ok {
        return nil, fmt.Errorf("kek id=%q not found: %w", kekID, ErrEncryptionKeyUnavailable)
    }
    out := make([]byte, len(key))
    copy(out, key)
    return out, nil
}
```

### 9.4 KMSKeyProvider 实施新方法

```go
func (p *KMSKeyProvider) ActiveKEKID() string {
    return p.KeyID() // delegate to existing
}

func (p *KMSKeyProvider) KEKByID(ctx context.Context, kekID string) ([]byte, error) {
    if !p.Available() {
        return nil, ErrEncryptionKeyUnavailable
    }
    if kekID != "" && kekID != p.keyID {
        return nil, fmt.Errorf("kms key provider single-key, got id=%q expected %q: %w",
            kekID, p.keyID, ErrEncryptionKeyUnavailable)
    }
    return p.KEK(ctx) // delegate to existing copy-returning impl
}
```

T-0091 真 KMS adapter 会扩展 KMSKeyProvider 为多版本（cache map），届时此处方法
路由到 multi-key cache。

### 9.5 迁移路径

部署升级流程（T-0087 ship 后运维操作）：

1. **T-0087 升级**：部署新 ACS 二进制；env 维持仅 OMC_BACKUP_ENCRYPTION_KEY → 单键模式 → 0 行为变化
2. **加 KEY_ID**（可选）：env 加 OMC_BACKUP_ENCRYPTION_KEY_ID="v1"；重启 — 新写入 envelope kek_id="v1"，老文件仍 v1 envelope 可读（KEKByID("") 自动 fallback）
3. **轮换准备**：生成新 KEK_v2 hex；env 改 KEY=v2_hex + KEY_ID="v2" + KEY_HISTORY="v1=<v1_hex>"；重启
4. **轮换生效**：新写入用 v2；老 v1 文件读取走 HISTORY 中的 v1 KEK
5. **批量 re-encrypt**（T-0092）：CLI 工具走 bucket 重加密所有 v1 → v2
6. **退役 v1**：所有文件 re-encrypted 后；env 移除 KEY_HISTORY；重启

### 9.6 安全自查清单

- [ ] kek_id 不进 authenticated scope（§2.3 评估）
- [ ] EnvKeyProvider 多版本 hex 解析 fail-fast
- [ ] active KEK 不能与 history KEK 同 ID（构造时 reject）
- [ ] keys map 不被外部 mutate（KEKByID 返 copy）
- [ ] kek_id_len 边界（u8 max 255）+ envelope truncate 检查
- [ ] 0 log key bytes（既有规范延续）
- [ ] reserved 字节 0x00 校验防 envelope 字段 misalignment
- [ ] 既有 wrapDEKWithKEK / unwrapDEKWithKEK helper 零改动 — KEK→DEK wrap 逻辑稳定

---

## 10. 实施要点

预计工作量：M-L（约 1.5-2 人日）— envelope 3 算法 × Encrypt/Decrypt 升级 +
KeyProvider 接口扩展 + EnvKeyProvider 多版本 env 解析 + 测试矩阵 V1-V10

预计涉及模块：`omcgo/internal/backup/`（6 文件改）+ 0 cmd 改

预计新增端点：无；预计新迁移：无；FE 影响：无；DI 改动：无

---

## 11. 审批

| 角色 | 占位 | 日期 | 备注 |
|------|------|------|------|
| 产品 / 架构 / 安全 / 运维 / QA | Claude | 2026-04-30 | scope 收紧 — re-encrypt CLI carve T-0092；envelope v2 显式 version bump；EnvKeyProvider 多版本 env 向后兼容；rotation 路径文档化 |

---

## 12. 变更记录

| 日期 | 版本 | 摘要 | 作者 |
|------|------|------|------|
| 2026-04-30 | v1.0 | 初稿；ULTRATHINK 8 决策；T-0087-core scope；carve T-0092 | Claude |
