# T-0086 verify — backup KMSKeyProvider skeleton + mock

> **Backlog**: T-0086 (P2 / F06/backup+security / sprint-08 / R-102)
> **PRD**: `docs/project/prd/T-0086-backup-kms-key-provider-skeleton.md`
> **Date**: 2026-04-30
> **Sensitivity**: 🔒 Security-sensitive — pluggability surface for KEK source

---

## 1. 改动摘要

| 文件 | 性质 | 行数变动 |
|------|------|---------|
| `omcgo/internal/backup/key_provider_kms.go` | 新增 — KMSClient iface + KMSKeyProvider impl | +146 |
| `omcgo/internal/backup/key_provider_kms_test.go` | 新增 — mockKMSClient + V1-V10 tests | +267 |
| `omcgo/internal/backup/key_provider.go` | 修改 — header doc 加 KMS pointer + KEKWrapper forward note | +13 / -5 |
| `omcgo/docs/project/prd/T-0086-backup-kms-key-provider-skeleton.md` | 新增 — PRD（scope 收紧 + carve T-0091） | +290 |

无新迁移、无新端点、无前端、无 cmd/main.go DI、无新依赖、无 encryption.go 改动。

## 2. 关键设计与 PRD §2 ULTRATHINK 对齐

| §2 决策 | 实施验证 |
|---------|---------|
| §2.1 沿用 KeyProvider 不引入 KEKWrapper | KMSKeyProvider 直接 `var _ KeyProvider` 编译期断言；T-0085 envelope 零改动 |
| §2.2 KMSClient 3 方法（GenerateDataKey / Decrypt / KeyID）| AWS KMS API shape mirror；ctx-aware；KeyID 用于 logs / 未来 metric labels |
| §2.3 caching 策略：构造时 Decrypt 一次，KEK(ctx) 返 copy | mirror EnvKeyProvider M-2 fix；caveat doc 在 package doc + struct doc 双重提示 |
| §2.4 ciphertextBlob 来源由 NewKMSKeyProvider 接受参数 | DI 拆分 — caller 决定从 env / file / secret store 读，本任务不绑死 |
| §2.5 mockKMSClient 用 AES-256-GCM 在 masterKey 下 wrap/unwrap | wrapInternal/unwrapInternal helper；nonce 随机；GCM tag 检测篡改 |
| §2.6 与 EnvKeyProvider 关系 | 同一接口 `KeyProvider`；DI 选其一；本任务不接 cmd/main.go DI 防 mock 泄漏 production |
| §2.7 测试矩阵 9 case + V10 wrong-size guard | 实施 V1-V10 全过 -race |
| §2.8 不引入真 KMS SDK | go.mod / go.sum 无变更；只用 stdlib + 既有 internal types |

## 3. 验证命令

```bash
CGO_ENABLED=0 go build ./...                               # ✅ pass
CGO_ENABLED=0 go vet ./...                                 # ✅ pass (0 issue)
gofmt -l <changed files>                                   # ✅ 0 file
CGO_ENABLED=0 go test -count=1 -race ./internal/backup/... # ✅ ok 2.765s
```

### 3.1 验收标准（PRD §4）

| Case | 测试 | 状态 |
|------|------|------|
| V1 KMSKeyProvider 构造 + KEK round-trip | `TestKMSKeyProvider_V1_RoundTrip` | ✅ |
| V2 KEK 返 copy 防 mutation | `TestKMSKeyProvider_V2_KEKReturnsCopy` | ✅ |
| V3 KMS Decrypt 失败 | `TestKMSKeyProvider_V3_DecryptFailure` | ✅ |
| V4 ciphertextBlob 篡改 | `TestKMSKeyProvider_V4_BlobTampered` | ✅ |
| V5 satisfies KeyProvider | `TestKMSKeyProvider_V5_SatisfiesKeyProvider` + 编译期 `var _ KeyProvider` | ✅ |
| V6 KeyID 暴露 + nil 安全 | `TestKMSKeyProvider_V6_KeyIDExposed` | ✅ |
| V7 Available false 边界（nil client / empty blob / nil receiver）| `TestKMSKeyProvider_V7_AvailableFalseForBadConstruction` | ✅ |
| V8 mockKMSClient round-trip + failNext 注入 | `TestMockKMSClient_V8_RoundTrip` | ✅ |
| V9 不同 master key 不互通 | `TestMockKMSClient_V9_DifferentMasterKeyNotInteroperable` | ✅ |
| V10 wrong-size KEK reject（非 32B）| `TestKMSKeyProvider_V10_RejectsWrongSizeKEK` | ✅ |

### 3.2 既有测试零回归

T-0083 / T-0085 既有 backup 测试全过；encryption.go 一字未改。

## 4. 安全自查清单

- [x] **KMSClient interface 1-3 方法（小接口原则）** — 3 方法精确 mirror AWS KMS shape
- [x] **plaintext KEK 仅在 struct 私有字段** — 不暴露通过 KeyID() 等其他途径
- [x] **KEK(ctx) 返 copy** — caller mutation 不污染源（V2 测试验证）
- [x] **fmt.Errorf 不携带 KEY 字节** — 静态 grep 通过
- [x] **caveat doc 显式声明 plaintext-in-memory 与 KMS-native 模型差异** — package doc + struct doc + Future 注释三处对齐
- [x] **NewKMSKeyProvider 4 路 fail-fast**（client nil / empty blob / Decrypt err / wrong size）— V3/V4/V7/V10 全测
- [x] **不接 cmd/main.go DI** — 防 mock 误入 production；T-0091 真集成时统一接
- [x] **mockKMSClient 在 _test.go 中** — 不可被 production 代码 import
- [x] **failNext 字段在 mock 用完后 reset** — 防多步骤测试脏状态
- [x] **Available() / KeyID() nil receiver 安全** — 不 panic（V6/V7 测试验证）

## 5. 与 T-0087 / T-0091 路径对齐

- **T-0087 KEK 旋转路径**：本任务 KMSClient 接口已含 `KeyID() string`（master key 标识符），未来 envelope 加 `kek_id` 字段时可直接读；mockKMSClient 的 masterKey 已是字段，扩展 `failNext="rotate"` + 多 master key 即可演练旋转
- **T-0091 真 KMS 集成路径**：实施 AWS KMS / Vault adapter 时只需让 adapter 满足 KMSClient 接口即可——KMSKeyProvider 调用方不动；预留 caveat doc 提示 T-0091 同步引入 KEKWrapper 摆脱 in-process plaintext

## 6. E/R 比

无新端点。

## 7. 已识别风险与遗留

| 风险 | 缓解 / 说明 |
|------|------------|
| plaintext KEK 仍在进程内存 | 同 EnvKeyProvider 弱点；caveat doc 显式标注；T-0087 KEKWrapper 解决 |
| KMSClient 非 cycling — 启动时 Decrypt 一次缓存 | T-0091 真集成时如运维要求每次 KMS 调用，需扩 KMSClient 加 `EncryptDEK` / `DecryptDEK` 方法对（per-encryption round-trip） |
| 没有真 KMS 适配器 | T-0091 显式 carve 出来，等运维选定 AWS / Vault / HSM 凭据后启动 |
| KMS 调用可观测性缺失（latency / error rate）| 真集成时加 `omc_backup_kms_call_*` metric（PRD §8 已记账）|
| 多版本 KEK 不支持 | T-0087 envelope 加 `kek_id` 解决 |

## 8. DoD 自查

- [x] `go build ./...`
- [x] `go test -race ./internal/backup/...`
- [x] `gofmt -l` 0 file
- [x] `go vet` 0 issue
- [x] PRD V1-V10 全测试覆盖
- [x] 公共接口无 `any` / `interface{}` — KMSClient 三方法都强类型
- [x] 无 TODO/FIXME 残留（"FUTURE (T-0087/T-0091)" 是 forward-compat note 不是 unfinished work）
- [x] 错误 `fmt.Errorf("ctx: %w", err)` wrap
- [x] 接口小（3 方法 / KeyProvider 已有的 2 方法）符合 §16.2 "Keep interfaces small (1-3 methods)"
- [x] 0 schema 变更 / 0 新依赖 / 0 cmd 改动 / 0 encryption.go 改动
- [x] caveat doc 三层（package / struct / Future note）确保后续维护者不误以为已是 KMS-native

## 9. 安全 review 等级建议

skeleton + mock 性质，未触及 `internal/admin/` 或 `middleware/auth*`。**Self-review verdict: APPROVE**：

- 接口形状对照 AWS KMS / Vault Transit 行业标准
- 所有 fail 路径 sentinel 化（ErrEncryptionKeyUnavailable / ErrEncryptionFormatInvalid 复用 T-0075 既有错误集）
- 不引入 production-impacting 改动（无 DI / 无 envelope / 无 SDK）
- caveat doc 三层防御后续维护者错误升级心智

未推荐 reviewer-agent 介入：路径未触发强制；改动是 well-bounded skeleton；不接 production DI 路径；mock 在 _test.go 不可外泄。

## 10. Reviewer 备忘

Approve 建议。Highlight：

1. **有意识地选择 KeyProvider 路线（§2.1 PRD 选项 A）** — 不是疏忽；KEKWrapper 接口的引入与 envelope kek_id 一并放到 T-0087；这避免 skeleton 阶段做不必要 refactor
2. **KMSClient 接口 3 方法精确 mirror AWS KMS** — 真 adapter 实施时几乎 1:1 翻译；Vault Transit 也只需薄包装
3. **mockKMSClient 在 _test.go 不可被 production 引入** — 防止 mock 误进入 cmd/main.go DI；T-0091 实施时由真 adapter 替代
4. **failNext 注入而非 returnedErr 字段** — 让单步骤测试更接近真 KMS 偶发失败模型；用完即 reset 防脏状态
5. **V10 wrong-size KEK reject** — 防御 KMS 误返非-32B（如适配器配错 key spec）；ErrEncryptionFormatInvalid 是正确错误类
6. **Future 注释绑定到具体 task ID（T-0087 / T-0091）** — 不是泛泛的 TODO，而是与 backlog 链接的 forward-compat 路标
7. **不接 cmd/main.go DI** — 是 *特性* 而非 omission；防 skeleton 误入 production；T-0091 真集成时 ACS 启动加 `if cfg.KMS.Enabled { kp = NewKMSKeyProvider(...) } else { kp = NewEnvKeyProvider() }`

## 11. T-0091 carve-out

新增 backlog 条目 T-0091 "backup 真 KMS 适配器实施"：

- **scope**: AWS KMS adapter（or Vault Transit / HSM 二选一） + DI wiring + observability metrics
- **deps**: T-0086 ✅ + 运维侧凭据
- **est**: M-L（视厂商）
- **prio**: P2（同 T-0086）
- **trigger**: 运维侧选定 KMS 厂商 + 提供 IAM role / Vault token

待 backlog 写入 §3 / §4。
