# PRD: backup KMSKeyProvider skeleton + mock（T-0086）

> **关联**: Backlog T-0086 / Sprint-08 / Domain=F06/backup+security / Type=feat / Prio=P2
> **作者**: Claude（代 Owner=电信+SecOps+运维）
> **创建**: 2026-04-30
> **状态**: 草案 → 实施
> **关键决策**: 仅做 skeleton + mock；真 AWS/Vault SDK 集成 carve out 到 T-0091（运维选定厂商凭据后再启动）

---

## 1. 业务背景

T-0075 落 AES-256-GCM 信封加密时把 KEK 来源抽象成 `KeyProvider` 接口，目前仅
`EnvKeyProvider`（OMC_BACKUP_ENCRYPTION_KEY 64-hex env var）一种实施。生产部署
中合规 / 等保 / 多租户场景常要求把 KEK 托管到外部 KMS（AWS KMS / 阿里 KMS / 华为
HSM / HashiCorp Vault Transit）—— 这是 T-0086 的最终目标。

实操难点：

- AI 端无运维凭据（KMS endpoint / IAM role / Vault token）—— 真集成必须等运维侧选定厂商
- 提前引入 AWS / Vault SDK 直接依赖会让 go.mod 长出几兆字节的间接依赖且无法测试
- T-0087（KEK 旋转）需要一个可插入的 KMS-style 后端来开发对接逻辑

**结论**：本任务交付 **skeleton + mock**，让 T-0087 + 任何真 KMS 厂商集成都可
独立推进。真 KMS 适配器（adapter）拆出 T-0091。

---

## 2. ULTRATHINK 决策

### 2.1 接口形状选择 — 沿用 KeyProvider vs 新增 KEKWrapper

候选方案：

| 方案 | 优 | 劣 |
|------|---|---|
| A. KMSKeyProvider 实现既有 `KeyProvider`（startup 时 cache 一份 KEK 到 RAM） | 0 既有代码改动；T-0085 envelope 不动 | KEK 在进程内存的语义不变 — 不是真 KMS 的"KEK 永不暴露明文"模式 |
| B. 引入 `KEKWrapper { Wrap, Unwrap }` 接口，refactor encryption.go 使用 wrap 而非直接 KEK | 安全模型正确；KEK 仅在 KMS 侧可见 | 大幅 refactor；envelope 需加 kek_id；测试矩阵翻倍 |

**采纳 A**。理由：

- 本任务定位是 skeleton；B 的 refactor 应该集中在 T-0087（KEK 旋转）一并做
- A 的 caching trade-off 与既有 EnvKeyProvider 无差别（env var 也驻留进程内存）
- 后续真 KMS 集成（T-0091）可通过厂商 adapter 实现 KMSClient → KMSKeyProvider 暴露
  既有 KEK 接口；选 B 路线时再独立做 envelope refactor

**T-0087 / T-0091 路径**：T-0087 需要 envelope 加 `kek_id` 才能做版本切换 —
那时再升级到 KEKWrapper 接口；本任务保留 caveat doc 在 KMSKeyProvider header。

### 2.2 KMSClient 接口最小集

参照 AWS KMS / Vault Transit 共同形态，定义最小 3 方法：

```go
type KMSClient interface {
    // GenerateDataKey returns a fresh plaintext+ciphertextBlob pair generated
    // by the KMS using its master key. Plaintext is the 32B AES key for
    // local use; ciphertextBlob is the opaque KMS-encrypted form for
    // persistent storage. Mirror of AWS KMS GenerateDataKey shape.
    GenerateDataKey(ctx context.Context) (plaintext, ciphertextBlob []byte, err error)
    // Decrypt returns the plaintext for a previously-generated ciphertextBlob.
    Decrypt(ctx context.Context, ciphertextBlob []byte) ([]byte, error)
    // KeyID returns the master key identifier for logging / observability.
    // Real impls return AWS ARN / Vault key path; mock returns "mock-kms".
    KeyID() string
}
```

3 方法足以：

- 启动时一次 `GenerateDataKey` 取得 plaintext KEK（cache）+ ciphertextBlob（持久化以备 T-0091 envelope 使用）
- 后续启动 `Decrypt(blob)` 还原 KEK（避免每次都 GenerateDataKey 产生新 key）
- KeyID 提供观测点（logs/metrics 标签）

不引入 Wrap/Unwrap 因 KEY-LEVEL wrap 是 envelope 范畴，本任务不动 envelope。

### 2.3 KMSKeyProvider caching 策略

Provider 在 `NewKMSKeyProvider(client, ciphertextBlob)` 构造时调用
`client.Decrypt(ciphertextBlob)` 一次，缓存 plaintext KEK 到 `provider.kek`。
后续 `KEK(ctx)` 从 cache 返回 copy（mirror EnvKeyProvider §M-2 fix 模式）。

trade-off：

- 优：与 EnvKeyProvider 行为完全一致 — encryption.go / cmd/acs/main.go 无须改
- 劣：plaintext KEK 驻留进程内存（与 EnvKeyProvider 同等弱点）
- 缓解：caveat doc + 真 KMS 用户走 T-0091 时同步引入 KEKWrapper 摆脱此模型

### 2.4 ciphertextBlob 来源

启动期 ciphertextBlob 从哪来？两选项：

- **option a**：env var `OMC_BACKUP_KMS_CIPHERTEXT_BLOB`（base64） — 简单；运维侧首次部署运行 `aws kms generate-data-key` 拿 blob 写入 env
- option b：从配置文件 / secret store 读

skeleton 阶段两个都不实施 DI（未接 main.go）；**接口只接受 ciphertextBlob []byte**。
真集成（T-0091）选 a 或 b 由运维选。

### 2.5 mockKMSClient 行为

测试用 mock 决定性能：

```go
type mockKMSClient struct {
    keys      map[string][]byte // ciphertextBlob hex → plaintext
    masterKey []byte            // for blob round-trip simulation
    keyID     string
    failNext  string            // "generate" | "decrypt" | "" inject failure
}
```

- `GenerateDataKey` 用 AES-256-GCM 在 masterKey 下 wrap 一个新 32B 随机 key（mirror AWS KMS algorithm）
- `Decrypt` GCM-open；mismatch → AuthFailed
- `KeyID` 返常量 "mock-kms-master-1"
- 支持 failNext 注入失败（覆盖 unavailable / blob tampered / context cancel 路径）

### 2.6 与 EnvKeyProvider 的关系

两 provider 满足同一 `KeyProvider` 接口；运维侧 DI 选其一（env-key OR kms-key）。
本任务 KMSKeyProvider 暴露但**不**接 main.go 的 cmd/acs DI（保留给 T-0091 真集成
时一并加）—— 避免 mock 误进入 production 路径。

### 2.7 测试矩阵

mirror TestEnvKeyProvider_* 模式：

- TestKMSKeyProvider_RoundTrip — Generate + Decrypt 圆环
- TestKMSKeyProvider_KEKReturnsCopy — 防 caller mutation
- TestKMSKeyProvider_AvailableTrue / False — 含 nil client 与失败构造
- TestKMSKeyProvider_BlobTampered — 检测 ciphertextBlob 被改后构造失败
- TestKMSKeyProvider_ConformsToKeyProvider — 编译期断言 `var _ KeyProvider = (*KMSKeyProvider)(nil)`
- TestMockKMSClient_GenerateDecrypt — mock 自身 round-trip
- TestMockKMSClient_failNextInjection — 失败注入路径
- TestMockKMSClient_RoundTripWithDifferentMasterKey — 不同 master key 不互通

### 2.8 不引入真 KMS SDK

PRD §1 明示原因：运维凭据 / 厂商选择尚未确定。skeleton 阶段：

- 不 import `github.com/aws/aws-sdk-go-v2/service/kms`
- 不 import `github.com/hashicorp/vault/api`
- 不修改 go.mod / go.sum

这两 SDK 任意一个会让 go.sum 长 30+ 条 transitive deps。等 T-0091 真集成时再加。

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| SecOps 工程师 | KMSKeyProvider skeleton 让我能开始评估 AWS KMS / Vault adapter 实施方案，无须等 OMC backend 完工 |
| 后端开发 | T-0087 KEK 旋转可在 mockKMSClient 上演练版本切换逻辑，无须真 KMS |
| 运维 | 看到代码里 KMSClient 接口存在，对未来切外部 KMS 有信心；当前先用 EnvKeyProvider 不阻塞 |
| 合规审计 | KMS 替换路径已就绪 — 等保认证时不必紧急写新代码 |

---

## 4. 验收标准（GWT）

### V1 — KMSKeyProvider 构造成功
- **Given** mockKMSClient + GenerateDataKey 取得 blob
- **When** NewKMSKeyProvider(client, blob) 调用
- **Then** 返回 non-nil provider；Available()=true；KEK(ctx) 返回 32B copy

### V2 — KEK 返回 copy 防 caller mutation
- **Given** 已构造 provider
- **When** 连续两次 KEK(ctx) 取出，对第一次返回的 slice 做 mutation
- **Then** 第二次 KEK 内容不受影响（mirror EnvKeyProvider M-2 fix）

### V3 — KMS Decrypt 失败路径
- **Given** mockKMSClient.failNext="decrypt"
- **When** NewKMSKeyProvider 构造
- **Then** 返回 error；no provider；error wrap KMS 失败原因

### V4 — ciphertextBlob 篡改检测
- **Given** 合法 blob，flip 1 byte
- **When** NewKMSKeyProvider 构造
- **Then** 返回 ErrEncryptionAuthFailed（GCM 验证失败）

### V5 — KMSClient 接口实现 KeyProvider
- **Given** *KMSKeyProvider 类型
- **When** 编译期 `var _ KeyProvider = (*KMSKeyProvider)(nil)`
- **Then** 编译通过（接口符合性）

### V6 — KeyID 暴露
- **Given** mockKMSClient.KeyID()="mock-kms-master-1"
- **When** provider.KeyID() 调用（如新增方法）OR provider 内部记录 KeyID 用于 logs
- **Then** KeyID 字符串可读

### V7 — Available 在 nil client 下返 false
- **Given** NewKMSKeyProvider(nil, ...) 调用
- **When** 构造失败，OR 构造成功但 client=nil
- **Then** Available() 返 false

### V8 — mockKMSClient round-trip
- **Given** mockKMSClient with masterKey
- **When** GenerateDataKey 后立即 Decrypt 同 blob
- **Then** plaintext 与 GenerateDataKey 返回的 plaintext 一致

### V9 — 不同 mockKMSClient masterKey 不互通
- **Given** 两个 mockKMSClient(masterKeyA / masterKeyB)，A 生成 blob
- **When** B Decrypt(blob_from_A)
- **Then** 失败

---

## 5. 运营商差异矩阵

无差异。KMS 选择由运维侧根据合规与现有基础设施决定，OMC 不绑定特定运营商。

---

## 6. 非目标

| # | 非目标 | 原因 |
|---|--------|------|
| N1 | AWS KMS / Vault / 华为 HSM 真 SDK 集成 | T-0091 followup（运维选厂商后） |
| N2 | KEK 旋转（多版本 key） | T-0087 followup |
| N3 | KEKWrapper 接口（envelope-style 替代 KeyProvider）| T-0087 联合做 |
| N4 | DI wiring 到 cmd/acs/main.go | skeleton 阶段不接 — 防 mock 误入 production；T-0091 一并接 |
| N5 | KMS 调用观测点（call latency histogram / error counter）| T-0091 真集成时加 |
| N6 | KEK 缓存 TTL（"每隔 N 小时 re-Decrypt"）| T-0087 联合 — 旋转时机自然刷新 |

---

## 7. 依赖

| 依赖 | 状态 |
|------|------|
| T-0075 加密框架 | ✅ done — KeyProvider 接口预留点 |
| `crypto/aes` + `crypto/cipher` | ✅ stdlib |

不引入第三方 dep。

---

## 8. 度量

skeleton 阶段无新 metric。真集成（T-0091）时计划加：

- `omc_backup_kms_call_total{op,result}` — KMS API 调用计数
- `omc_backup_kms_call_duration_seconds{op}` — 延迟直方图

---

## 9. 设计备忘（S2）

### 9.1 文件清单

新增：

- `omcgo/internal/backup/key_provider_kms.go` — KMSClient 接口 + KMSKeyProvider 实施 + caveat doc
- `omcgo/internal/backup/key_provider_kms_test.go` — mockKMSClient + V1-V9 测试

修改：

- `omcgo/internal/backup/key_provider.go` — header doc 加 "see key_provider_kms.go for KMS variant" 指引

无 cmd/main.go DI 改动；无 encryption.go 改动；无 go.mod 改动。

### 9.2 KMSKeyProvider 结构

```go
// KMSKeyProvider satisfies KeyProvider by fetching the plaintext KEK from
// a KMSClient at construction time and caching it. The cached KEK is
// returned (as a copy) by KEK(ctx) for the lifetime of the process.
//
// Security caveat: this design keeps the plaintext KEK in process memory,
// matching the EnvKeyProvider model. It is NOT the KMS-native pattern
// where every Encrypt/Decrypt round-trips to the KMS service. The
// correct architecture for that flow is KEKWrapper (T-0087+); the
// KMSKeyProvider here is a pluggability surface for testing and for
// future real KMS adapters that operate under the same constraint
// (e.g., shared-cluster envelope).
type KMSKeyProvider struct {
    client KMSClient
    kek    []byte // 32B; nil until constructed; never logged
    keyID  string
}
```

### 9.3 KMSClient 接口

```go
type KMSClient interface {
    GenerateDataKey(ctx context.Context) (plaintext, ciphertextBlob []byte, err error)
    Decrypt(ctx context.Context, ciphertextBlob []byte) ([]byte, error)
    KeyID() string
}
```

### 9.4 mockKMSClient

```go
type mockKMSClient struct {
    masterKey []byte // 32B AES-256
    keyID     string
    // failNext: when non-empty, the next matching call returns an error.
    // valid values: "generate", "decrypt"
    failNext string
}
```

mock 内部用 AES-256-GCM 在 masterKey 下 seal/open 模拟真 KMS — 行为决定性 +
覆盖 wrong-master-key / tampered-blob 路径。

### 9.5 文件位置

key_provider_kms.go 与 key_provider.go 同包同目录；命名一致 `key_provider_kms.go`
让 grep `key_provider` 一并捞到所有 provider 实现。

### 9.6 forward-compat note

KMSKeyProvider doc 显式说明 T-0087 / T-0091 路径：

```go
// FUTURE (T-0087): when KEK rotation lands, this provider should expose
// a KEKVersion() string and the encryption envelope should record the
// version so unwrap can pick the right key.
//
// FUTURE (T-0091): real KMS adapters (AWS KMS / Vault / HSM) implement
// KMSClient with the matching service SDK. The KMSKeyProvider host
// stays unchanged.
```

---

## 10. 实施要点

预计工作量：M（约 1 人日）— 1 接口 + 1 struct + 1 mock + 7-9 case；不动现有代码

预计涉及模块：`omcgo/internal/backup/`（2 新文件 + 1 doc-only 修改）

预计新增端点：无；预计新迁移：无；FE 影响：无；DI 改动：无

---

## 11. 审批

| 角色 | 占位 | 日期 | 备注 |
|------|------|------|------|
| 产品 / 架构 / 安全 / 运维 / QA | Claude | 2026-04-30 | skeleton + mock；真集成 T-0091 拆 followup；KEK 旋转 T-0087 联合 envelope refactor |

---

## 12. 变更记录

| 日期 | 版本 | 摘要 | 作者 |
|------|------|------|------|
| 2026-04-30 | v1.0 | 初稿；ULTRATHINK 8 决策；scope = skeleton+mock；carve T-0091 真集成 | Claude |
