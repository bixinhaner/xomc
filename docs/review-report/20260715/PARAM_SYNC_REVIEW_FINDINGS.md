# 参数同步新方案本地修改 Review Findings

日期：2026-07-15
分支：`feat/param-sync-reliable-data-plane`
范围：当前工作区全部未提交修改，包括 `paramsync`、ACS、Provisioning、Task、配置、数据库迁移和前端接入
结论：本轮识别的 3 个 P1 可靠性问题均已按低风险方案修复并补回归测试；仍建议先灰度验证，不直接按生产配置中的 100% durable 模式切入生产。

## P1：高风险

### 1. 9005 返回私有 path 时，full coverage 不会被标记为 incomplete

修复状态：已解决。

修复结果：ResultProcessor 现在只使用 run 冻结 mapping 解析 private/standard path，支持对象前缀和运行时实例；无法解析时先按本 task 的 requested names 缩小范围，仍无法定位则保守将整轮 coverage 标为 incomplete，保证未知 9005 不会触发误删：`omcgo/internal/paramsync/result_processor.go:447-554`。回归测试见 `omcgo/internal/paramsync/coverage_delete_test.go:27-90`。

证据：

- planner 将 coverage 的 `Path` 和冻结 mapping 的 `StandardPath` 保存为标准路径，同时另存 `PrivatePath`：`omcgo/internal/paramsync/planner.go:151-176`。
- ACS 将设备 fault 中的 `bad_path`/`bad_paths` 写入 recovered task result；这些值来自设备侧，因此 standard/private mapping 不同时是私有 path：`omcgo/internal/acs/handler.go:1488-1512`。
- ResultProcessor 将上述 path 原样交给 `markCoverageIncomplete`：`omcgo/internal/paramsync/result_processor.go:97-110`。
- `markCoverageIncomplete` 只用 bad path 与 coverage 的标准 `Path` 做正反方向的文本/前缀匹配，没有使用 `CoverageScope.Mappings[].PrivatePath` 反向翻译：`omcgo/internal/paramsync/result_processor.go:451-482`。
- 当前 coverage 测试只验证删除 predicate；没有覆盖“私有 bad path 与标准 coverage path 不同”的 incomplete 标记：`omcgo/internal/paramsync/coverage_delete_test.go:10-25`。

影响：

- standard/private path 不同的参数收到 tolerated 9005 后，对应 coverage 仍保持 `complete=true`。
- full finalize 会继续执行该 coverage 的缺失参数删除，旧值可能被错误删除。
- run 仍可能以 `succeeded/OK` 结束，使该数据损坏不易从状态和摘要中识别。

修复要求：使用 run 冻结 mapping 将 `bad_path`/`bad_paths` 从 private path 反向定位到 standard coverage，或同时按冻结 mapping 的 standard/private 模板匹配；增加叶子、对象前缀和运行时实例三类 standard/private 不同的回归测试，并验证 incomplete coverage 不执行删除。

### 2. PostgreSQL send fence 成功后的任一 Redis 错误，会留下从未下发的 `sent` task

修复状态：已解决。

修复结果：新增按 `task_id + cwmp_id + status='sent'` 限定的 `ReleaseSentClaimIfUnwritten`，只补偿 ACS 明确尚未写出 RPC 的窗口；CAS 未命中时不会覆盖 terminal 或新 claim。补偿后以 PG pending 状态修复 Redis，Redis 仍不可用时交给既有 reconciler；同时删除了 fence 成功后的多余 Redis 回读和 PG 二次更新：`omcgo/internal/task/pg_repository.go:187-200`、`omcgo/internal/task/service.go:493-557`。故障与终态防复活测试见 `omcgo/internal/task/service_pg_test.go:350-405`。

证据：

- `MarkTaskSent` 先以 PostgreSQL CAS 将 task 从 `pending` 改为 `sent`，随后才更新 Redis：`omcgo/internal/task/service.go:495-512`。
- Redis `MarkTaskSent` 失败时函数直接返回，没有把 PostgreSQL 中本次 CAS 写入的 `sent/cwmp_id/sent_at` 补偿回 `pending`：`omcgo/internal/task/service.go:512-514`。
- 即使 Redis 标记成功，随后的 `GetByID` 失败也会返回错误，同样没有补偿 PostgreSQL fence：`omcgo/internal/task/service.go:517-520`。
- ACS 只有在 `MarkTaskSent` 整体成功后才返回待发送 RPC；上述错误会结束本次 dispatch，不向设备发送该 task：`omcgo/internal/acs/handler.go:1016-1034`。
- 同一会话内再次处理时，PostgreSQL fence 只接受 `pending`，因此该 task 会被拒绝并删除 Redis 副本：`omcgo/internal/task/pg_repository.go:168-187`、`omcgo/internal/task/service.go:497-509`。
- 新 Inform 会调用 `RecoverPendingTasks`，以 PostgreSQL 中的 `sent` 记录为准将任务恢复为 `pending`；因此该问题不是永久不可恢复，但恢复依赖设备再次发起 Inform，并会消耗一次 retry：`omcgo/internal/acs/handler.go:469-477`、`omcgo/internal/task/service.go:782-851`。
- 现有 PG 测试覆盖 fence 成功和 fence 拒绝后的 stale Redis 清理，但没有覆盖“PG CAS 成功后，Redis Mark/Get 失败”：`omcgo/internal/task/service_pg_test.go:316-369`。

影响：

- 短暂 Redis 错误会把一个实际未发给设备的 task 留在 PostgreSQL `sent` 状态并中止当前 CWMP dispatch。
- task 只能等下一次 Inform 才会恢复；设备 Inform 周期较长时，单次瞬时故障会被放大为明显延迟，并额外消耗 retry budget。重复发生时可提前耗尽重试并使 durable run 失败。
- 该问题影响所有走 `TaskService.MarkTaskSent` 的 task，不仅参数同步。

修复要求：为 PG fence 成功后的、且明确发生在 HTTP RPC 写出前的失败分支增加带 `task_id + cwmp_id + status='sent'` 条件的补偿，并恢复 Redis 可执行索引；保留 `RecoverPendingTasks` 处理进程崩溃等发送结果不确定的场景。补故障注入测试，覆盖 Redis Mark 失败、补偿写 Redis 失败、补偿与取消/过期并发，证明 Redis 恢复后 task 可再次安全发送且 terminal/cancelled task 不会复活。

### 3. completion projection 补偿存在两层永久短路

修复状态：已解决。

修复结果：maintenance 与 projector 使用独立超时预算并最终聚合错误；批量 projection 逐条隔离失败。新增 `000023_parameter_sync_projection_lease.sql`，通过 lease token、过期时间和 next-attempt 退避实现单 worker claim、崩溃回收及失败防热循环；事件和 maintenance 共用同一执行路径：`omcgo/cmd/app/provider/paramsync.go:83-97`、`omcgo/internal/paramsync/completion_projector.go:54-171`。批量容错、并发 claim、旧 NULL lease 回收和退避测试见 `omcgo/internal/paramsync/completion_projector_test.go`。

证据：

- `runParamSyncMaintenance` 会执行所有 repair，并用 `errors.Join` 返回任一子任务错误：`omcgo/cmd/app/provider/paramsync.go:44-79`。
- 定时循环仅在上述聚合错误为 nil 时才调用 `projector.ReconcilePending`：`omcgo/cmd/app/provider/paramsync.go:415-421`。
- 即使进入 `ReconcilePending`，它也会按 `completed_at` 顺序逐条处理，并在第一条 projection 错误时立即返回：`omcgo/internal/paramsync/completion_projector.go:83-116`。一条永久失败的旧 run 会在每轮都排在前面，后续 pending run 永远得不到处理。
- `projectRun` 会把失败记录重新置为 `failed`，因此该毒性记录会持续满足下一轮查询条件：`omcgo/internal/paramsync/completion_projector.go:56-72`。
- 现有 maintenance 测试只证明 maintainer 内部 repair 不互相短路，projector 不在接口和测试调用序列中；也没有多条 projection 中首条失败、后续仍继续的测试：`omcgo/cmd/app/provider/paramsync_maintenance_test.go:53-61`。

影响：

- 任一持续性 maintenance 错误都会阻止全部 pending/failed completion projection 重试。
- 即便其他 maintenance 全部成功，一条历史坏 projection 也会形成队头阻塞，后续所有 full run 的 `device_info` 和设备名可能永久不刷新。
- 数据库中的 `projection_status/projection_attempts/projection_error` 只能记录失败，不能形成面向整批记录的补偿闭环。

修复要求：无论其他 maintenance repair 是否失败，都独立执行 `ReconcilePending`，最后统一聚合错误；`ReconcilePending` 对每条 run 独立尝试、收集错误后继续，并通过 CAS/claim 避免并发重复执行；增加测试证明 maintainer 返回错误时 projector 仍被调用，以及首条 projection 永久失败时后续 run 仍能完成。

## 其他审查结论

- 复核了 request/run 状态机、task/outbox、9005 recovery、result/finalize、取消与超时补偿、provisioning binding、completion projection、配置开关、迁移 `000021/000022`、ACS/APP 接线和前端 request/run 轮询契约。
- 未发现新的 P0；上述 3 项已修复，本轮没有遗留需要阻断合入的已知 finding。
- `projectTaskValues` 保留冻结 mapping 之外的设备返回参数是当前测试明确固定的行为（`omcgo/internal/paramsync/planner_test.go:158-175`），本报告不将其作为缺陷；若产品要求 durable full snapshot 只允许已映射 storable path，需另行收紧契约和测试。

## 验证结果

- `go test ./internal/paramsync ./internal/task ./internal/acs ./internal/device ./internal/provision ./cmd/app/provider ./internal/core/appconfig`：通过。
- `go test -race ./internal/paramsync`、`go test -race ./internal/task`、`go test -race ./cmd/app/provider`：串行执行并通过。
- `go vet ./internal/paramsync ./internal/task ./cmd/app/provider ./internal/acs`：通过。
- `cd omcmb && npm run typecheck`：通过。
- `go build ./...`：通过。
- `go test ./...`：通过（包含 `test/e2e` 和 `test/integration`）。
- `git diff --check`：通过。
- 已新增 recovered 9005 私有路径映射与保守降级、发送 fence Redis 故障补偿及 terminal CAS、projection 批量隔离/并发 claim/lease 回收/失败退避的回归测试；上述 3 条失败路径均有直接测试覆盖。
- migration 按约定合入 `000023_parameter_sync_projection_lease.sql`，未修改既有 `000021/000022`。

## 已完成的修复顺序

1. 已修复私有 `bad_path` 到冻结 coverage 的匹配和保守降级，阻止 full reconcile 误删数据。
2. 已修复发送 fence 的 PG/Redis 条件补偿，避免未下发 task 停留为 `sent`。
3. 已移除 completion projection 的两层短路，并加入 lease、退避和批量容错测试。

## 低风险修复方案

### A. Coverage：只依赖 run 冻结数据，无法定位时保守禁止删除

1. 新增纯函数，将 `storedTaskResult` 中的 `bad_path`、`bad_paths` 和 `RequestedNames` 映射为需要置为 incomplete 的 coverage 下标；不要查询当前 ParamRegistry，避免运行中 mapping 变化影响历史 run。
2. 匹配顺序固定为：冻结 mapping 的 private→standard 精确翻译；对象前缀/运行时实例的分段模板匹配；使用本 task 的 `RequestedNames` 定位 coverage。
3. 路径比较必须按 `.` 分段并将纯数字实例归一化为 `{i}`，不能使用无边界的字符串前缀，避免 `Device.Radio.1` 错配 `Device.Radio.10`。
4. recovered 9005 若仍无法定位，full run 必须采取保守策略：将该 task 请求所覆盖的 scope 置为 incomplete；连请求 scope 也无法解析时，将本 run 全部 coverage 置为 incomplete。允许保留旧值，不允许继续把未知失败解释为“参数已不存在”。
5. `markCoverageIncomplete` 只负责持久化已解析下标，并保持一次事务内更新；不要在这里重新读取 registry 或设备产品信息。

回归测试至少覆盖：standard/private 不同的叶子、对象前缀、单/多级 `{i}`、批次整体跳过、相似数字实例不串扰、未知路径保守降级，以及 incomplete scope 不执行 delete、无关 scope 仍可正常 reconcile。

### B. Send fence：仅补偿“确定未写出 RPC”的窗口，不猜测未知发送结果

1. 保留 PostgreSQL `MarkSentIfPending` 作为发送权威 fence；删除 fence 成功后多余的 Redis `GetByID → repo.Update`，因为 PG 已经持久化 `sent/cwmp_id/sent_at`，该二次读取只扩大失败窗口。
2. 增加仓库条件 CAS，例如 `ReleaseSentClaim(taskID, cwmpID)`：仅当当前仍为同一 `sent + cwmp_id` 时恢复 `pending` 并清空 `cwmp_id/sent_at`。CAS 未命中时不得覆盖 completed/failed/expired/cancelled 或新的发送 claim。
3. Redis `MarkTaskSent` 在 RPC 写出前失败时调用上述补偿；CAS 成功后删除旧 CWMP 映射并用 PG 返回的权威 task 重建 pending 队列。Redis 仍不可用时保留 PG pending，由既有 PG/Redis reconciler 修复，不再次把 PG 改成 sent。
4. 只有代码能证明 RPC 尚未写入 HTTP response 时才允许补偿。进程崩溃、连接中断或写出结果不确定时继续保留 sent，由新 Inform 的 `RecoverPendingTasks` 恢复，避免重复下发一个设备可能已经执行的 RPC。
5. 补偿失败要 `errors.Join` 原始错误并增加独立指标；不能吞掉错误，也不能在 CAS 未命中时强行 requeue。

故障注入测试至少覆盖：Redis Mark 完全失败/部分成功、补偿 Redis 仍失败、CAS 前并发 completed/expired/cancelled、同一 task 获得新 cwmp_id、进程崩溃仍由新 Inform 恢复，以及任何 terminal task 都不会回到队列。

### C. Projection：独立执行、逐条隔离、使用可恢复 lease

1. reconcile tick 中无条件分别调用 maintainer 和 projector，最后 `errors.Join`；两者任何一方失败都不能阻止另一方执行。
2. `ReconcilePending` 对整批 run 逐条尝试并收集错误，不能在首条失败时返回；成功计数只统计真正 completed 的 projection。
3. 不使用永久 `processing` CAS。新增增量迁移（建议 `000023`，不要重写可能已经应用的 `000021`）加入 projection lease/next-attempt 字段；claim 只接受 `pending/failed` 或 lease 已过期的 `processing`，并以条件 UPDATE RETURNING 保证同一时刻只有一个 worker 执行。
4. 成功时清 lease 并置 `completed`；失败时置 `failed`、记录错误、设置有上限的指数退避。进程在 Refresh 中崩溃时由 lease 到期自动恢复。
5. 事件实时处理和 maintenance 补偿必须复用同一个 claim/project/complete 方法，避免形成两套并发语义。

回归测试至少覆盖：maintenance 失败仍执行 projector、首条永久失败不阻塞后续、两个实例并发只有一个拿到 claim、worker 崩溃后 lease 可回收、失败退避不会热循环、重复 projection 保持幂等。

### D. 交付与回滚约束

1. 先写上述失败测试，再按 A→B→C 顺序实现，每个工作包独立通过 targeted tests、`go test ./...`、`go vet` 和前端 typecheck。
2. 修复完成前，生产配置保持 durable data plane 关闭并保留 legacy fallback；不要直接使用当前 `canary_percent: 100`。
3. 上线按 5%→25%→50%→100% 逐级观察；重点监控 coverage incomplete、send-claim compensation、task retry exhaustion、projection failed/lease recovery 和 outbox backlog。
4. 每一级只通过配置回退，不回滚迁移、不删除 request/run/outbox 数据；发现参数误删、terminal task requeue 或 projection backlog 持续增长时立即停止扩量。
