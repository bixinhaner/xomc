# S4 Verify Report — T-0089 backup decrypt semaphore

> 任务：T-0089 / `feat(acs): backup decrypt 并发 semaphore 防 DoS`
> Sprint：sprint-08（R-102 followup / T-0075 review M-4 闭环）
> 基线 commit：`a2cb305f`（T-0084 S7 之上）
> 验证日期：2026-04-29

---

## 1. 验证范围

PRD §4 列 10 条 GWT（V1-V10）。新增 DecryptSemaphore type + Handler 集成 + 3 metrics + env var DI。无新端点，无 schema migration，纯并发控制。

## 2. S3 出口门复核

| 命令 | 结果 |
|------|------|
| `go build ./...` | ✅ 通过 |
| `go vet ./...` | ✅ 通过 |
| `gofmt -l <my-files>` | ✅ 无输出（5 文件全 clean） |
| `go test -race ./internal/acs/download/...` | ✅ 通过（10 case + V5 3 sub） |
| `go test -race ./internal/backup/...` | ✅ 通过（既有 + 3 新 metric 注册） |
| `go test -race ./internal/acs/...` | ⚠️ stun 包 `TestServer_Integration_NonStandardENB` race，**pre-existing**（baseline 不含 T-0089 改动也复现 — 已 stash 验证） |
| 新增 TODO/FIXME/panic | 0 |
| 新增 carrier 硬编码 | 0 |
| 新增 `any` / `interface{}` | 0 |

## 3. S4 验证清单

### 3.1 metric 名 grep（3/3 命中）

```
omc_backup_decrypt_in_flight       → policy_metrics.go:144
omc_backup_decrypt_wait_seconds    → policy_metrics.go:148
omc_backup_decrypt_rejected_total  → policy_metrics.go:153
```

### 3.2 迁移演练

**N/A** — 本任务无新迁移。纯进程内并发控制，不触 DB。

### 3.3 端点 E/R ≥ 1

**N/A** — 无新增 HTTP 端点（仅扩 maybeDecrypt 内部流程 + ServeHTTP 错误映射加分支）。

### 3.4 累计型依赖核销

**N/A** — Deps `T-0075 ✅`（commit `715a756a` + S7 `a1c54231`）是普通 done 依赖。

### 3.5 GWT 覆盖矩阵

| GWT | Test | Result |
|-----|------|--------|
| V1 under-limit no block | TestDecryptSemaphore_UnderLimit_NoBlock | ✅ |
| V2 at-limit timeout | TestDecryptSemaphore_AtLimit_Timeout | ✅ |
| V3 release unblocks waiter | TestDecryptSemaphore_ReleaseUnblocksWaiter | ✅ |
| V4 ctx cancel no slot leak | TestDecryptSemaphore_CtxCancel_NoSlotLeak | ✅ |
| V5 limit≤0 disable | TestDecryptSemaphore_DisabledByZeroLimit (3 sub: 0, -1, -100) | ✅ |
| V6 503+Retry-After in HTTP | handler.go:172-185 ServeHTTP 错误映射 — 设计 verify（无 e2e mock minio 测试）| ✅ code-level |
| V7 env var fallback / clamp | TestDecryptSemaphore_HardCapClamp + decryptDefaultLimit code-level | ✅ |
| V8 nil metrics safe | TestDecryptSemaphore_NilMetricsSafe | ✅ |
| V9 wait duration returned | TestDecryptSemaphore_AcquireReturnsWaitDuration | ✅ |
| V10 sentinel comparable | TestDecryptSemaphore_ErrSentinelIsComparable | ✅ |
| Bonus 并发 fairness | TestDecryptSemaphore_ConcurrentFairness（10 acquirer / limit=2） | ✅ |

**总计**：10 顶层 + 3 sub + 1 bonus = 14 case，全 PASS / -race 全过。

### 3.6 新代码覆盖率（per-function）

| 函数 | 覆盖率 |
|------|--------|
| `NewDecryptSemaphore` | **100%** |
| `Acquire` | **100%** |
| `Release` | **100%** |
| `SetDecryptSemaphore` (handler) | 0%（既有 handler 集成测试无；与 T-0075 maybeDecrypt 同 baseline 0%）|
| `maybeDecrypt` (handler) | 0%（既有 baseline）|
| 整包 download | 26.2%（无回退；T-0089 新增 100% coverage 拉高基线）|

### 3.7 既有 T-0075 测试零回归

`internal/acs/download/...` 全 case PASS / -race 全过 — 既有 T-0072 解压 + T-0075 解密路径在 SetDecryptSemaphore 未调用时（h.decryptSem == nil）走原 maybeDecrypt nil-safe 分支。

`internal/backup/...` 全 case PASS / -race 全过 — 3 新 collector 注册不影响既有 T-0073/0074/0075/0076/0082/0084 metric 行为。

### 3.8 文件清单

新增（2）：
- `omcgo/internal/acs/download/decrypt_semaphore.go`（~125 行 — DecryptSemaphore type + ErrDecryptSemaphoreTimeout sentinel + DecryptSemaphoreHardCap const + NewDecryptSemaphore + Acquire + Release，全部 nil-safe）
- `omcgo/internal/acs/download/decrypt_semaphore_test.go`（~210 行 — 10 顶层 V1-V10 + 3 sub + 1 bonus）

修改（3）：
- `omcgo/internal/acs/download/handler.go` — Handler +decryptSem 字段；+SetDecryptSemaphore setter；maybeDecrypt 加 ctx 参数 + decryptSem.Acquire/defer Release；ServeHTTP 错误映射加 4 路 switch（ctx cancel / semaphore timeout / auth-fail / default）
- `omcgo/internal/backup/policy_metrics.go` — +3 collectors（gauge / histogram / counter-vec）+ 3 nil-safe Record/Set/Observe 方法 + 顶部 doc 更新
- `omcgo/cmd/acs/main.go` — `time` + `strconv` import；env var `OMC_BACKUP_DECRYPT_CONCURRENCY` 解析（默认 8）+ `decryptDefaultTimeout = 30s` const + DI 调用 `SetDecryptSemaphore` 仅当 `backupEncryptor != nil`（保持 T-0075 启用条件）

文档（1）：
- `docs/project/prd/T-0089-backup-decrypt-semaphore.md`（PRD ~310 行）

Backlog：
- `docs/project/backlog.md`（T-0089 状态 triaged → in_design + PRD 路径回写）

## 4. 风险与已知限制

| # | 项 | 状态 |
|---|----|------|
| K1 | encrypt 上传侧 64MB×N 同样放大 | PRD §2.6 不同威胁模型；upload-side 拆 followup |
| K2 | 进程重启清空 in-flight gauge | 接受（gauge 反映当前 process 状态，重启即新世代） |
| K3 | DecryptSemaphoreHardCap 64 是经验值 | 64 × 64MB = 4GB 是合理的内存上限；future 调整需 const 修改 |
| K4 | timeout 30s 不可配置 | PRD §9.7 接受；如有需求可再加 env var |
| K5 | x/sync/semaphore 替代品 | PRD §2.4 buffered channel 足够；future 若需 weighted 可换 |

## 5. S4 出口门结论

| 门 | 状态 |
|----|------|
| 所有命令绿（build/vet/test/race） | ✅（stun pre-existing race 已记账） |
| E/R ≥ 1（无新端点则 N/A） | ✅ N/A |
| 迁移双向演练（无迁移则 N/A） | ✅ N/A |
| 3 metric 名全部能 grep 找到 | ✅ |
| 累计型依赖阈值（无则 N/A） | ✅ N/A |

**S4 通过**，进入 S5 review。

---

## 6. S5 Review 落实记录

### 6.1 Review-agent — PASS-WITH-FIXES（0 P0 / 0 HIGH / 3 MED / 4 LOW）

| Severity | Finding | 处置 |
|----------|---------|------|
| MED-1 | 负 env var 静默 disable 是误关 footgun（运维想"无限"反而无 cap） | ✅ decryptDefaultLimit 添加 WARN log "≤ 0 disables semaphore — UNBOUNDED" |
| MED-2 | NewDecryptSemaphore "oversize_config" 计入 rejection 计数器 mislead Grafana 告警（构造时一次性脉冲非真 acquire 拒绝） | ✅ 移除 metric record；新增 `ClampedLimit()` 方法；caller `cmd/acs/main.go` 在 clamp 时 WARN log + Info log effective limit |
| MED-3 | `Acquire` 在 `size > maxEncryptedSize` 检查前 — 攻击者可短周期 churn slot 耗 legit acquirer | ✅ 顺序翻转：size pre-check 在 Acquire 之前；oversize 请求直接 422 不占 slot |
| LOW-1 | size-check vs I/O error wrap 路径分流 | ⏸ cleared — 当前行为正确（auth/format → 422，I/O → 500） |
| LOW-2 | TestDecryptSemaphore_HardCapClamp 不释放 64 slots | ✅ 添加 `t.Cleanup(release-loop)` |
| LOW-3 | V10 errors.Join 测试与 handler.go 实际 fmt.Errorf %w 不一致 | ✅ 加 `errorsWrapWithFmt` helper + assert |
| LOW-4 | gauge `omc_backup_decrypt_in_flight` 在 Set 与 channel-op 间 racy 快照 | ⏸ leave — Prometheus Set 线程安全；Grafana 端 eventual consistency 可接受 |
| LOW-5 | `decryptDefaultTimeout` 不可配置 | ⏸ leave — PRD §9.7 已 settled；future env var 如有需求再加 |

**Reviewer 11 项 cleared 验证**：slot leak on ctx cancel（select 三路分支语义）/ timer leak after fire（defer Stop no-op safe）/ Prometheus gauge 并发安全 / nil-receiver semaphore 全路径 / nil PolicyMetrics 全 Record 路径 / ServeHTTP ctx-cancel return 安全（无 header staged）/ hard-cap 严格 64（V7 测试证明 64 success + 第 65 timeout）/ 错误映射顺序最特定优先 / 既有 T-0075 AAD 计算未触动 / nil metrics 在 success+timeout 双路径 / decryptDefaultLimit bad-parse fallback。

### 6.2 DoD（按 `docs/project/dod.md` 通用项）

- [x] `go build ./...` 通过
- [x] `go test -race ./internal/acs/download/...` 全绿（11 case + 3 sub）
- [x] `go test -race ./internal/backup/...` 全绿（既有 + 3 新 metric 注册）
- [x] `gofmt -l <my-files>` 无输出
- [x] `go vet ./...` 通过
- [x] 新代码无 TODO/FIXME/panic
- [x] 无新增 carrier 硬编码
- [x] 无新增 `any`/`interface{}`
- [x] 新功能含成功 + 失败 + 边界路径测试（V1-V10 + 1 bonus）
- [x] 新代码 per-function coverage：NewDecryptSemaphore + Acquire + Release 各 100%
- [x] 既有 T-0075 + T-0072 测试零回归
- [x] 关键 review finding 全部落实或显式 leave 并记账

**S5 通过**，进入 S6 commit。
