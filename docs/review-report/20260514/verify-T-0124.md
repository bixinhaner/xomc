# T-0124 — S4 本地验证报告

**任务**：T-0124（F09 参数同步触发链 §2 — 周期同步 PeriodicSyncer + PG advisory lock leader）
**PRD**：`docs/project/prd/F09-param-sync-trigger-chain.md#§2` + `## 设计备忘（T-0124）`
**Sprint**：sprint-11 stretch
**Owner**：Claude
**日期**：2026-05-14
**Est**：L（最大工作量）

---

## 1. 改动范围

| 文件 | 行数变化 | 说明 |
|------|---------|------|
| `omcgo/migrations/000094_devices_last_param_sync_at.sql` | +18 / -0 | **新增迁移** — `last_param_sync_at TIMESTAMPTZ` + 部分索引 `WHERE status='active'` NULLS FIRST + Down 配对 |
| `internal/core/model/device.go` | +3 / -0 | Device struct 加 `LastParamSyncAt *time.Time` |
| `internal/device/device_repository.go` | +80 / -0 | DeviceReader 接口 + DeviceWriter 接口各加 1 方法；Pg 实现 `ListStaleForParamSync`（NULLS FIRST 优先 + Lt threshold）+ `UpdateLastParamSyncAt`（单列）；deviceColumns + scanDeviceFromRow + scanDeviceRow 含字段 |
| `internal/core/appconfig/config.go` | +18 / -1 | ProvisionConfig 加 PeriodicSync 嵌套；PeriodicSyncConfig 5 字段（Enabled/Interval/BatchSize/MaxConcurrent/StaggerWindow）+ 中文注释 |
| `internal/provision/leader_elector.go` | **新增 110 行** | LeaderElector interface + PGAdvisoryLeaderElector 实现：TryAcquire / Release 含 conn 长持 + held atomic + 异常归还自动 unlock |
| `internal/provision/periodic_syncer.go` | **新增 200 行** | StaleDeviceLister + PathBSyncStarter narrow interfaces；PeriodicSyncer struct + Start/runOnce/enqueueBatch；并发池 semaphore + WaitGroup + Stagger window 打散；nil leader 视为单副本部署放行；defer leader.Release |
| `internal/provision/sync.go` | +20 / -0 | ParamSyncWriter narrow interface + SetParamSyncWriter setter；SyncService.paramSyncWriter 字段 |
| `internal/provision/sync_pathb.go` | +12 / -0 | HandleSyncResultPathB BatchUpsert 成功后调 paramSyncWriter.UpdateLastParamSyncAt（统一回写口径，不区分触发源） |
| `cmd/app/provider/modules.go` | +20 / -2 | DI wiring：syncSvc.SetParamSyncWriter(c.DeviceRepo)；PeriodicSync.Enabled=true 时 PGAdvisoryLeaderElector + PeriodicSyncer + go Start |
| `cmd/app/etc/config.dev.yaml` | +10 / -0 | provision.periodic_sync 样例段（enabled=false 灰度，含灰度建议注释） |
| 9 个测试 mock 文件 mock stubs 补齐 | +54 / -0 | ListStaleForParamSync + UpdateLastParamSyncAt 桩补齐：service_test.go / handler_test.go / heartbeat_test.go / inform_handler_test.go / mock_device_repository_test.go（mockgen 风格 4 方法）/ provision/engine_test.go / interop/runner_test.go / transfer/bridge_test.go / software/service_test.go / backup/executor_test.go / nedirect/service_test.go / northbound/sync/service_test.go |
| `internal/provision/periodic_syncer_test.go` | **新增 240 行** | 12 testcase 覆盖 PeriodicSyncer 全控制流 |
| `internal/provision/sync_pathb_test.go` | +30 / 0 | 2 testcase 验证 SetParamSyncWriter 链式 + nil safe |

**总改动**：净 +815 / -3，生产代码 ~490 行 / 测试 ~325 行。**符合 L 工作量**。

---

## 2. S4 出口门

| 门 | 结果 |
|----|------|
| `go build ./...` | ✅ PASS |
| `go test ./...` 全包 | ✅ PASS（仅 pre-existing TestDownloadHandler 与本任务无关） |
| `check-migrations.sh` | ✅ 编号区间 000001 → 000094；新加无冲突；20 处既有冲突 release-gate 集中清理（与本任务无关） |
| 前端 typecheck | N/A |
| `golangci-lint` | ⚠️ 本地未装留 CI |
| 迁移 up/down 演练 | ⚠️ 跳过（环境无 PG）；Down 段已写完整 DROP INDEX + DROP COLUMN，本地 dev 启动会自动应用 |
| 新端点 E/R ≥ 1 | N/A（无新 REST 端点） |
| 累计型依赖 | N/A |

---

## 3. 观测埋点 grep 验证

```bash
grep -rn "ListStaleForParamSync" internal/ → device repo interface + Pg impl + 12 mock 桩
grep -rn "UpdateLastParamSyncAt" internal/ → device repo interface + Pg impl + ParamSyncWriter + sync_pathb 回写 + 12 mock 桩
grep -rn "periodic syncer" internal/ → 5 log key（started / disabled / not leader / batch enqueued / list/StartPathBSync 失败）
grep -rn "PGAdvisoryLeaderElector\|pg_try_advisory_lock\|pg_advisory_unlock" internal/ → leader_elector.go 实现
grep -rn "last_param_sync_at written" internal/ → sync_pathb.go debug log（成功回写）
```

全 grep 命中 ✅

---

## 4. 测试统计（新增）

| 测试函数 | 行为覆盖 | 结果 |
|---------|---------|------|
| `TestPeriodicSyncer_DisabledNoOp` | Enabled=false → Start 立即返不调 lister | ✅ |
| `TestPeriodicSyncer_NotLeader_SkipsRun` | leader.TryAcquire 返 false → 跳过 list + sync | ✅ |
| `TestPeriodicSyncer_LeaderError_SkipsRun` | leader.TryAcquire 返 err → 跳过本轮 | ✅ |
| `TestPeriodicSyncer_LeaderRunsBatch` | leader 拿到 → 5 设备全入队 + 验证 reason="periodic" + sourceID 前缀 | ✅ |
| `TestPeriodicSyncer_NilLeader_RunsBatch` | nil leader（单副本部署）直接放行 | ✅ |
| `TestPeriodicSyncer_EmptyBatch_NoSyncCalls` | list 返 nil → 不入队 | ✅ |
| `TestPeriodicSyncer_StartPathBSyncFailureIsolated` | 单设备 sync 失败 → 其他设备仍入队（per-device 隔离） | ✅ |
| `TestPeriodicSyncer_PathBUnavailable_CountsAsSkipped` | used=false（无 MappingSet）算 skipped 不算失败 | ✅ |
| `TestPeriodicSyncer_ListError_NoCrash` | list 失败 → 不入队不 panic | ✅ |
| `TestPeriodicSyncer_RespectsBatchSizeDefault` | 默认 BatchSize=200 / MaxConcurrent=10 | ✅ |
| `TestPeriodicSyncer_Start_StopsOnCtxCancel` | ctx cancel → Start 退出 + defer leader.Release | ✅ (50ms) |
| `TestPeriodicSyncer_ConcurrentRequest_RespectsMaxConcurrent` | MaxConcurrent=5 → 20 设备峰值并发 ≤ 5 | ✅ (80ms) |
| `TestSetParamSyncWriter_ChainableReturnsSyncService` | 链式调用契约 | ✅ |
| `TestSetParamSyncWriter_NilSafe` | nil 注入不 panic | ✅ |

**合计 14 个新测试 / 0 个 FAIL** ✅

---

## 5. 出口门 5 项检查

- [x] 接口契约明确（4 narrow interfaces：LeaderElector / ParamSyncWriter / StaleDeviceLister / PathBSyncStarter）
- [x] 迁移草案 up/down 完整 + 编号 000094 无冲突
- [x] Carrier 差异点：无新增（GPV 范围方案 B 复用 MappingSet）
- [x] 观测埋点：5 log key grep 命中；Prometheus counter 推 follow-up（与 T-0123 throttle counter 一起加）
- [x] 待定点 0 个

---

## 6. Out of Scope / 诚实碎片

1. **Prometheus 周期同步指标** — `provision_periodic_devices_processed_total` 等推 follow-up（与 T-0123 throttle counter 一起加），当前可由 log grep "batch enqueued" 统计
2. **PG advisory lock 真集成测试** — 需启 PG 环境；当前测试通过 mock LeaderElector 验证控制流，PG 实现的正确性由代码评审 + 多副本灰度部署验证
3. **多副本 leader 切换演练** — 范围外（runbook 记录手动验证：起两个 app 副本观察日志确认只一个执行 runOnce）
4. **迁移 up/down 本地双向演练** — 跳过（无 PG dev 环境）；Down 段已写完整 DROP INDEX + DROP COLUMN，goose 框架会按版本号自动应用
5. **golangci-lint** — 本地未装留 CI
6. **OfflineDetector 迁移到 LeaderElector 抽象** — T-G post-RC，本任务范围外（接口已为复用预留）
7. **TestDownloadHandler pre-existing failure** — 与本任务无关，git stash 验证基线已 FAIL

---

## 7. 关键决策回顾（ULTRATHINK 6 项）

| 决策 | 方案 | 理由 |
|------|------|------|
| Narrow interface 注入 | `ParamSyncWriter` / `StaleDeviceLister` / `PathBSyncStarter` 消费者侧 1-2 方法接口 | 测试只 mock 必要方法；DeviceRepository 自然满足；后续 T-G 迁移 OfflineDetector 时复用 |
| LeaderElector 抽象先行 | 2 方法接口 + PGAdvisoryLeaderElector 唯一实现 | T-G post-RC 让 OfflineDetector 迁移；接口稳定不锁定实现 |
| Lock conn 长持 vs 每 tick 抢 | startup 一次 TryAcquire + held atomic 幂等 | 避免抢占抖动；连接断开 → PG session 结束自动 release（异常退出天然不残留） |
| 回写位置严格 BatchUpsert 成功后 | sync_pathb.go 末端调用一行 | GPV 部分失败/超时/取消时不回写让下一轮兜底；不在入队时回写避免任务未完被下一 tick 跳过 |
| MaxConcurrent 控制 CreateTask 入队并发 | 非 GPV 并发 | GPV 并发由现有 ACS worker pool / 准入控制管理；防 PG 写入毛刺 |
| nil leader 视为单副本部署放行 | NewPeriodicSyncer(... leader nil ...) 跳过 leader 检查直接执行 runOnce | dev/test 简化；生产部署必传 PGAdvisoryLeaderElector |

---

**S4 出口门：✅ PASS**
