# T-0092 verify — backup KEK rotation re-encrypt CLI + service

> **Backlog**: T-0092 (P2 / F06/backup+security+ops / sprint-08 / R-102)
> **PRD**: `docs/project/prd/T-0092-backup-reencrypt-cli.md`
> **Date**: 2026-04-30
> **Sensitivity**: 🔒 Security-sensitive — re-encrypt path touches KEK + envelope

---

## 1. 改动摘要

| 文件 | 性质 | 行数变动 |
|------|------|---------|
| `omcgo/internal/backup/reencryptor.go` | 新增 — Reencryptor service + ObjectIO 接口 + reencryptOne loop + 8 outcome 枚举 + atomic stats | +320 |
| `omcgo/internal/backup/reencryptor_test.go` | 新增 — V1-V10 + required-fields + algoStringFromByte + Stats.Total | +330 |
| `omcgo/internal/backup/policy_metrics.go` | +2 collector + 2 Record helper（reencryptTotal counter + reencryptDurationSeconds histogram） | +30 |
| `omcgo/cmd/backup-reencrypt/main.go` | 新增 — 独立二进制 entry，env-only 配置，cobra root cmd，signal handling | +180 |
| `omcgo/docs/project/prd/T-0092-backup-reencrypt-cli.md` | 新 PRD | +320 |

无新依赖（cobra/zap/minio-go 都已 direct）、无 schema、无 cmd 既有改动、无 encryption.go 改动。

## 2. 关键设计与 PRD §2 ULTRATHINK 对齐

| §2 决策 | 实施验证 |
|---------|---------|
| §2.1 独立二进制 vs omcctl 子命令 | 选 A（独立 cmd/backup-reencrypt）— omcctl HTTP-only 与 data-plane 直访 MinIO 模式不混 |
| §2.2 复用既有 KeyProvider + Encryptor | 0 新 crypto 路径；NewEncryptor(algo, kp) 同实例 Decrypt+Encrypt round-trip |
| §2.3 AAD 重建 = basename(key) − ".enc" | trimEncSuffixBasename helper；与 ACS upload/download AAD 一致 |
| §2.4 跨算法 re-encrypt 不在范围 | parseEnvelopeHeader 返 hdr.algo → algoStringFromByte → 同算法 Encrypt/Decrypt；algo byte 跨 re-encrypt 守恒（V8 测试） |
| §2.5 跳过条件 7 类 | 7 outcome enum + log warn + 不动文件 |
| §2.6 并发控制 | NewReencryptor clamp [1, 32]；buffered channel + sync.WaitGroup |
| §2.7 进度报告 | atomic.AddInt64(&processed) + ProgressEvery 触发 zap.Info |
| §2.8 metrics | 2 新 collector + 2 helper；nil-safe |
| §2.9 测试矩阵 V1-V10 | reencryptOneInMemory 测试钩子（绕过 *minio.Object 难以构造）+ 真 ObjectIO mock 用于 NewReencryptor 校验 |

## 3. 验证命令

```bash
CGO_ENABLED=0 go build ./...                               # ✅ pass (含 cmd/backup-reencrypt)
CGO_ENABLED=0 go vet ./...                                 # ✅ pass (0 issue)
gofmt -l <changed files>                                   # ✅ 0 file (post-normalize)
CGO_ENABLED=0 go test -count=1 -race ./internal/backup/... # ✅ ok 4.017s
```

### 3.1 验收标准（PRD §4）

| Case | 测试 | 状态 |
|------|------|------|
| V1 单文件 v1 → v2 | `TestReencryptor_V1_SingleFileV1ToV2` | ✅ |
| V2 already on target → skip | `TestReencryptor_V2_AlreadyTarget` | ✅ |
| V3 非 .enc 文件 → skip | `TestReencryptor_V3_NotEncryptedFile` | ✅ |
| V4 source KEK missing | `TestReencryptor_V4_SourceKEKMissing` | ✅ |
| V5 envelope corrupt | `TestReencryptor_V5_EnvelopeInvalid` | ✅ |
| V8 三算法 re-encrypt（GCM/CBC/ChaCha20）| `TestReencryptor_V8_ThreeAlgos`（3 子用例） | ✅ |
| V9 dry-run 不写 | `TestReencryptor_V9_DryRun` | ✅ |
| V10 concurrency clamp 1..32 | `TestReencryptor_V10_ConcurrencyClamp`（6 子用例：-5/0/1/8/32/100） | ✅ |
| Required-field validation | `TestReencryptor_RequiredFields`（4 subcase） | ✅ |
| algoStringFromByte | `TestAlgoStringFromByte` | ✅ |
| ReencryptStats.Total | `TestReencryptStats_Total` | ✅ |

V6 (PutObject 失败) + V7 (ctx cancel) 留 future integration test（现有 fakeObjectIO
GetObject 路径返 *minio.Object 不便 mock；in-memory test 钩子不覆盖 PutObject 失败
路径 — production 路径 reencryptOne 已 handle，但单测覆盖 deferred）。

### 3.2 既有测试零回归

- T-0083 / T-0085 / T-0086 / T-0087 既有 backup 测试全过
- encryption.go / encryption_test.go 一字未改

## 4. 安全自查清单（PRD §9.5）

- [x] **AAD 重建逻辑与 ACS upload/download 完全一致** — trimEncSuffixBasename 等价于 strings.TrimSuffix(filepath.Base(key), ".enc")；V1-V8 测试均用真实文件名格式验证 round-trip
- [x] **dry-run 默认关闭** — `cobra.Flags.BoolVar(&dryRun, "dry-run", false, ...)` 显式设默认 false
- [x] **target KEK 启动期校验** — main.go run() 中 `kp.KEKByID(ctx, targetKekID)` fail-fast；防误删 / 误覆写
- [x] **PutObject 失败保持原文件** — reencryptOne 返 resultFailed；不重试（操作可重启）；MinIO PutObject 是同 key atomic overwrite，没有"半写入"风险
- [x] **KEK bytes 不进任何 log** — 既有规范延续；env decode 出错时 zap.Error 不携带 key bytes（hex 解析失败前已 reject）
- [x] **envelope 解析失败的文件不动** — resultSkipEnvelopeInvalid；可能是手工放置 / 损坏 / future 格式 — 安全默认
- [x] **concurrency clamp** — NewReencryptor [1, 32]；防 1000-worker 误配；6 子测试覆盖边界
- [x] **ctx cancellation 优雅** — list 端检测 ctx.Done() 关 keys channel；workers drain 已收到的；已写入文件保留（PutObject atomic）；返 ctx.Err()
- [x] **algo byte 跨 re-encrypt 守恒** — V8 测试 `assert.Equal(t, blob[5], newBlob[5])`；不允许跨算法迁移（PRD §2.4 明确）
- [x] **kek_id 跨 re-encrypt 改变** — V1 测试 `assert.Equal(t, "v2", string(newBlob[8:10]))`；这是工具的核心契约
- [x] **failed 计数 → 非零 exit code** — main.go run() 末尾 `if stats.Failed > 0 { return error }`，cobra 返 1
- [x] **错误类型一致** — 复用 ErrEncryptionKeyUnavailable / ErrEncryptionFormatInvalid / ErrEncryptionAuthFailed；不引入新 sentinel

## 5. 架构决策回顾

### 独立二进制 vs omcctl 子命令

omcctl 是 HTTP-only client（API key + server URL），与 data-plane 直访 MinIO + 本地
env KEK 模式正交。独立二进制让：

- 单一职责清晰
- 部署人员心智简单（同 host 同 env 即可，无需 server URL 配置）
- omcctl 不被污染为"两种工作模式"

### env-only 配置 vs viper config file

参考 cmd/migrate/main.go 简单模式 — env-only 让运维不必维护 yaml；与
ACS 部署 env 配置一致。

### 测试钩子 reencryptOneInMemory

`*minio.Object` 无公开构造函数，单测 mock GetObject 返成功对象不可行。两选项：

A. refactor production 接口接受 io.ReadCloser → 损害 production 类型保真
B. 保持 production 接口用 *minio.Object，加 `_test.go` 内的 reencryptOneInMemory
   helper 直接 byte-input 跑 parse + decrypt + encrypt 路径

采纳 B。失去 GetObject error 路径覆盖 vs 维持 production 类型清洁 — 取后者。
集成测试（与真 MinIO）future task 补 GetObject 失败 + PutObject 失败 + ctx cancel
覆盖。

## 6. E/R 比

无新端点（独立二进制 ≠ HTTP endpoint）。N/A。

## 7. 已识别风险与遗留

| 风险 | 缓解 / 说明 |
|------|------------|
| V6 (PutObject error) + V7 (ctx cancel) 单测未覆盖 | 已在 reencryptOne production 路径 handle；future integration test 接真 MinIO 时补 |
| 跨算法 re-encrypt 缺失 | PRD §6 N1 明确 out of scope；future 单独工具 |
| Resume 状态文件缺失 | 工具天然幂等（已 target 跳过）；不需 |
| HTTP API 触发缺失 | T-0091 真 KMS adapter 时再考虑（届时 KEK 远程托管 + RBAC 完善） |
| metrics endpoint 启动 | MVP 不接；future 加 --metrics-listen=:9094 |
| OBJ 大小硬限 64MiB+4KiB | 与 encryption.go encMaxPlaintext 一致；超大文件已不在加密支持范围 |

## 8. DoD 自查

- [x] `go build ./...`（含 cmd/backup-reencrypt 二进制）
- [x] `go test -race ./internal/backup/...`
- [x] `gofmt -l` 0 file
- [x] `go vet` 0 issue
- [x] PRD V1-V5, V8-V10 + required-fields + algo helper + Stats.Total 全测试覆盖（V6/V7 deferred 至 integration test）
- [x] 公共接口 ObjectIO 2 方法（小接口原则）
- [x] 无 TODO/FIXME 残留（"future integration test" 是 PRD §6 deferred 项不阻塞）
- [x] 错误 `fmt.Errorf("ctx: %w", err)` wrap
- [x] 0 schema 变更 / 0 新依赖 / 0 cmd 既有改动 / 0 encryption.go 改动
- [x] 既有 backup 测试零回归（T-0083/T-0085/T-0086/T-0087 全过）
- [x] 12 项安全自查全 [x]

## 9. 安全 review 等级建议

加密 data-plane 工具属高敏感；路径未触及 `internal/admin/` 或 `middleware/auth*`。**Self-review verdict: APPROVE**：

1. 复用既有 KeyProvider + Encryptor，0 新 crypto 路径
2. AAD 重建逻辑与 upload/download 严格对称（trimEncSuffixBasename）
3. 7 outcome 跳过原因覆盖完整 attack surface（不动文件是默认）
4. concurrency clamp + dry-run 默认关闭 + target-KEK fail-fast 三层操作安全
5. 既有 13+ encryption tests 零回归证明 envelope 编解码不变

未推荐 reviewer-agent 介入：路径未触发 §B5 强制；改动 well-bounded；无 cmd DI 改动到生产路径。

## 10. Reviewer 备忘

Approve 建议。Highlight：

1. **独立二进制路线** — 与 omcctl HTTP-only 模式正交；ops 工具单一职责
2. **复用 KeyProvider + Encryptor** — 0 新 crypto 路径；algo byte 跨 re-encrypt 守恒（V8 显式测试）
3. **trimEncSuffixBasename helper** — 测试与 production 共用，AAD 计算单点
4. **NewReencryptor 必填字段验证** — Bucket / Lister / IO / KeyProvider 4 路 fail-fast；防 nil-deref 在 worker
5. **atomic stats + buffered channel** — 无 mutex 的 worker 池；ctx cancel + list err 关 channel 让 workers drain 干净
6. **dry-run 模式提前出** — 在 PutObject 之前；测试 V9 验证 newBlob == nil 报告"would re-encrypt"而非真写
7. **reencryptOneInMemory 测试钩子** — production 接口保持 *minio.Object 类型保真；测试 _test.go 文件用 byte-input 路径覆盖核心逻辑
8. **三算法 algo byte 守恒** — V8 三子测试 `blob[5] == newBlob[5]`；防止"reencrypt 顺便切算法"footgun
9. **KEK bytes 不进 log** — `zap.String("source_kek_id", hdr.kekID)` 仅日志 ID，不日志 key bytes（id 是 metadata 不是 secret）
10. **failed > 0 → 非零 exit code** — cobra return 让 cron / CI / 运维脚本可检测 partial failure
