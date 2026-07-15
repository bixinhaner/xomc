# 参数同步功能详细实现计划

- 日期：2026-07-15
- 关联架构：`PARAM_SYNC_REMEDIATION_PLAN.md`
- 目标：先交付可靠数据面，再在不修改核心业务契约的前提下扩展至 10 万级设备控制面
- 当前状态：待评审、待拆 Issue

## 1. 实施目标与边界

本计划分为两个可独立验收的交付：

### 交付 A：可靠数据面

解决当前参数同步的正确性和可靠性问题：

- 每次同步触发都有 request ID 和最终结果。
- 每轮设备执行都有 run ID。
- 同设备最多一个 active full run。
- task 执行完成与结果落库完成分别计数。
- GPV 同步结果只有一个写入者。
- 结果处理成功后才 ACK。
- 重复事件不会重复写入、计数或 finalize。
- full sync 使用整轮 staging，成功后一次性提交。
- partial/readback 不更新全量同步时间，不执行 diff/reconcile。
- Provisioning 能随 run 进入明确成功或失败终态。
- PG、Redis、NATS 短时故障不会造成请求或结果静默丢失。

### 交付 B：10 万设备控制面

在交付 A 的稳定接口上增加：

- SyncCampaign 渐进生成设备请求。
- 分布式 Scheduler。
- 全局、分组和 ACS 实例级配额。
- 公平调度和多级背压。
- 固定分片 worker pool。
- Request/Run/Result 补偿器。
- 批量写入、数据分区、结果对象存储。
- 容量压测和动态速率控制。

### 非目标

- 本次不重构非参数同步类 device task。
- 本次不改变 CWMP 协议和设备侧行为。
- 第一阶段不实现通用的 Full+Partial 请求合并。
- 第一阶段不建立同设备无限 FIFO 队列；用户冲突继续返回 409，自动重复触发返回 deduplicated。
- 第一阶段不迁移历史已终态 device task，仅处理上线后的新参数同步请求。

## 2. 关键实施原则

1. PostgreSQL 是 request/run/task/result 的权威状态。
2. Redis 只作为可重建的 task 执行索引。
3. NATS 使用 At-Least-Once，业务层通过数据库幂等实现 Exactly-Once Effect。
4. 同设备串行，不同设备并行。
5. 不在设备执行期间持有 PG advisory transaction lock。
6. 所有数据库变更使用增量迁移，当前建议从 `000021` 开始；施工前必须再次检查最大迁移号。
7. 新旧链路通过功能开关和设备哈希灰度切换，不能让两个结果消费者同时写同一参数。
8. 每个工作包必须包含测试、指标和回滚路径。

## 3. 目标包结构

新增后端包：

```text
omcgo/internal/paramsync/
├─ model.go                 # Request/Run/Result/Binding 领域模型
├─ state_machine.go         # request/run 状态转换
├─ repository.go            # 仓库接口
├─ pg_repository.go         # PG 实现
├─ service.go               # 请求接收与查询
├─ planner.go               # MappingSet → GPV 批次计划
├─ dispatcher.go            # request → run → task/outbox
├─ result_consumer.go       # 唯一参数同步结果消费者
├─ result_processor.go      # 翻译、过滤、staging 写入
├─ finalizer.go             # full/partial 收口
├─ outbox.go                # task/result outbox dispatcher
├─ reconciler.go            # Request/Run/Result 补偿
├─ scheduler.go             # 交付 B 分布式调度
├─ campaign.go              # 交付 B Campaign
├─ metrics.go
└─ *_test.go
```

现有模块职责调整：

- `provision`：保留触发适配和 provisioning 状态联动，移出参数落库与收尾判断。
- `device`：保留设备 API 和普通 GPV 投影；参数同步 GPV 不再由 `RPCResponseSubscriber` 写库。
- `acs`：保留协议执行、SOAP 解析、task terminal 和结果引用发布。
- `task`：提供批量 task/outbox 和按 run 查询能力，不感知参数同步业务状态机。

## 4. 交付 A 工作包

## A0. 契约冻结与功能开关

### 实现内容

定义并冻结：

- RequestStatus：`accepted/queued/running/succeeded/failed/timed_out/cancelled/deduplicated/rejected`。
- RunStatus：`planning/enqueuing/waiting_device/executing/processing/succeeded/failed/cancelled`。
- SyncScope：`full/partial/readback/policy_probe`。
- TriggerReason：`bootstrap/model_upload/device_online/firmware_changed/periodic/manual/config_pull/license/spv_readback/add_object_readback/inform_period_probe`。
- ResultCode：`OK/ACTIVE_SYNC_EXISTS/PATH_B_UNAVAILABLE/NO_STORABLE_PATH/TASK_FAILED/TASK_EXPIRED/RESULT_PROCESSING_FAILED/DEADLINE_EXCEEDED` 等。

新增运行时开关：

```text
param_sync.run_enabled
param_sync.result_consumer_enabled
param_sync.staging_enabled
param_sync.canary_percent
param_sync.legacy_fallback_enabled
```

设备灰度函数必须确定性：

```text
hash(device_id) % 100 < canary_percent
```

### 修改位置

- 新增 `internal/paramsync/model.go`、`state_machine.go`。
- `internal/core/event/subjects.go` 增加参数同步结果和终态主题。
- `internal/core/appconfig` 或 `sys_configs` 增加功能开关读取。

### 验收

- 所有非法状态转换返回明确错误。
- 相同 device ID 在所有实例上得到相同灰度结果。
- 开关关闭时现有行为不变。

## A1. 数据库迁移与仓库

### 迁移文件

当前主库迁移最大号为 `000020`，建议新增：

```text
omcgo/migrations/000021_parameter_sync_runtime.sql
```

施工前重新执行最大号和撞号检查；不要依据迁移 README 中已滞后的“仅 000001”描述复用编号。

### 新增表

- `parameter_sync_requests`
- `parameter_sync_runs`
- `parameter_sync_request_bindings`
- `parameter_sync_task_results`
- `parameter_sync_staging_values`
- `parameter_sync_outbox`

### 关键字段

Request：

```text
id, device_id, device_sn, trigger_reason, sync_scope
requested_paths, status, run_id, active_run_id
priority, next_attempt_at, deadline_at
idempotency_key, result_code, result_summary, error_message
campaign_id（第一阶段允许为空）
created_at, started_at, completed_at, updated_at
```

Run：

```text
id, request_id, device_id, device_sn
trigger_reason, sync_scope, mapping_source, mapping_version
status, expected_task_count, terminal_task_count
processed_task_count, failed_task_count
error_message, started_at, completed_at, version
```

### 索引和约束

- 同设备 active full run 部分唯一索引。
- `(caller_type, idempotency_key)` 唯一索引，允许 idempotency key 为空。
- Request 调度索引：`(status, priority, next_attempt_at, created_at)`。
- Run 设备历史索引：`(device_id, created_at DESC)`。
- Task result 主键：`(run_id, task_id)`。
- Staging 主键：`(run_id, parameter_path)`。
- Outbox 索引：`(status, next_attempt_at, created_at)`。

### 仓库接口

```go
type Repository interface {
    CreateRequest(ctx context.Context, req *SyncRequest) error
    GetRequest(ctx context.Context, id uuid.UUID) (*SyncRequest, error)
    CreateOrDeduplicateRun(ctx context.Context, req *SyncRequest) (*StartResult, error)
    GetRun(ctx context.Context, id uuid.UUID) (*SyncRun, error)
    InsertTaskResultIfAbsent(ctx context.Context, result *TaskResult) (bool, error)
    ClaimRunForFinalize(ctx context.Context, runID uuid.UUID) (bool, error)
}
```

涉及 staging/finalize 的操作必须支持传入 `pgx.Tx`，保证结果记录、参数暂存和计数更新在同一事务内。

### 测试

- 迁移 Up/Down。
- 同设备并发创建 100 个 full run，只有一个成功成为 active。
- idempotency key 并发插入只产生一条 request。
- task result 重复插入返回 already processed。
- CAS finalize 只能被一个实例领取。

## A2. Request API 与触发适配

### 新增服务接口

```go
type Service interface {
    Submit(ctx context.Context, cmd SubmitCommand) (*SubmitResult, error)
    GetRequest(ctx context.Context, requestID uuid.UUID) (*RequestView, error)
    GetActiveRun(ctx context.Context, deviceID uuid.UUID) (*RunView, error)
}
```

`SubmitResult` 至少包含：

```text
request_id, run_id, status, result_code
scope, trigger_reason, task_count, active_run_id
```

### 触发入口迁移顺序

1. 手工 `/devices/:id/sync-params`。
2. `device.online`。
3. `firmware.changed`。
4. Bootstrap Path B。
5. ModelUpload 完成。
6. PeriodicSyncer。
7. Config Pull / License partial。

### 处理规则

- 用户请求碰到 active run：Request 终态 `rejected`，HTTP 返回 409。
- 自动请求碰到 active run：Request 终态 `deduplicated`，记录 active run ID。
- Provisioning 碰到兼容 active full run：创建 binding，等待该 run 终态。
- MappingSet 不可用：Request 终态 `rejected/PATH_B_UNAVAILABLE`。
- 无 storable path：Request 直接 `succeeded/NO_STORABLE_PATH`，Provisioning 同步完成，不创建 task。

### 兼容响应

现有前端依赖 `source_id`，过渡期返回：

```text
source_id = request_id
request_id = request_id
run_id = run_id
```

前端完成迁移后再废弃 `source_id` 别名。

### 测试

- 参数树和快速设置并发提交，后到请求得到 409 和 request ID。
- 自动同步重复触发得到 deduplicated 终态。
- 无路径和 Mapping 不可用均有明确结果。
- 相同 idempotency key 重试返回原 request。

## A3. Planner 与 full/partial/readback 分类

### Planner 输入

```go
type PlanCommand struct {
    RunID          uuid.UUID
    Device         *model.Device
    Scope          SyncScope
    RequestedPaths []string
    TriggerReason  TriggerReason
}
```

### Planner 输出

```go
type Plan struct {
    MappingSource string
    MappingVersion string
    RequestedPaths []string
    Batches []TaskBatch
    Coverage []CoverageScope
}
```

### 改造内容

- 将 `extractStorablePrefixes`、大对象展开和 `buildGPVBatches` 封装到 Planner。
- `command_key` 只用于诊断，不再决定 full/partial。
- task 必须从 run 读取结构化 scope。
- 大对象展开阈值改为 `gpvNATSPayloadBudgetBytes`。
- `readback/policy_probe` 不创建 full run，可作为轻量 request 或普通 device task，但必须带结构化用途标签。

### 测试

- full、partial、readback 生成不同计划语义。
- scoped manual/config/license 绝不标记 full。
- 大对象临界值始终低于 NATS 安全预算。
- MappingSet 相同输入生成确定性批次顺序。

## A4. 原子 task 计划与 Outbox

### 新增 task 能力

避免修改通用 `Enqueuer` 的所有调用者，新增窄接口：

```go
type BatchPlanner interface {
    CreatePlannedTasks(ctx context.Context, tx pgx.Tx, reqs []CreateTaskRequest) ([]*Task, error)
}
```

同一事务内：

1. Run 转为 `enqueuing`。
2. 创建全部 device_tasks，`source=param_sync`、`source_id=run_id`。
3. 为全部 GPV 批次写 task enqueue outbox。
4. 回写 `expected_task_count`。
5. Run 转为 `waiting_device` 或 `executing`。

Outbox dispatcher：

- `FOR UPDATE SKIP LOCKED` 分批领取。
- Redis Push 成功后标记 delivered。
- 失败指数退避并记录 last_error。
- 达到上限不丢弃，转 dead 并由告警/补偿器处理。
- Redis Push 必须以 task ID 幂等。
- 同一 run 的全部批次按 `command_index/priority` 放入同一设备队列；ACS 在一个 CWMP 会话内按“请求 → 响应 → 下一请求”严格串行，不允许批次并发。
- 不能等待上一批 result consumer 完成后才释放下一批，否则设备会先结束会话，退化为每批一次 Connection Request。
- Outbox 每轮派发先完成该轮所有 task 的无唤醒入队，再按设备去重触发一次逻辑唤醒；禁止每个批次各发一次 Connection Request。普通单任务入口仍保持即时唤醒。
- 手动同步 API 只创建 durable request/run，不再保留旧 Path B 的直接 Connection Request；否则会与 Outbox 唤醒重复。

### 现有代码调整

- `provision.EnqueueGPVBatches` 过渡为 Planner/Dispatcher 适配器。
- `TaskService.CreateTask` 保留给非参数同步任务。
- 参数同步禁止继续逐条 `CreateTask`。

### 测试

- 第 N 个 task 构造失败时事务整体回滚。
- PG 提交后进程退出，Outbox 重启后仍能推送全部 task。
- Redis 重复 Push 不产生重复可执行 task。
- Outbox backlog 指标和告警可用。

## A5. ACS 结果持久化与轻量事件

### 新事件

```go
type TaskResultEvent struct {
    EventID   string
    RequestID uuid.UUID
    RunID     uuid.UUID
    TaskID    string
    DeviceID  uuid.UUID
    DeviceSN  string
    Success   bool
    ResultRef string
}
```

新增主题建议：

```text
param_sync.task.result
param_sync.run.completed
param_sync.run.failed
```

### ACS 改造

- task terminal 前后保存解析结果和 raw response 引用。
- 不在 task.completed 事件内携带完整 raw SOAP。
- 对 `source=param_sync` 的 task 发布专用 TaskResultEvent。
- failed/expired/cancelled 也必须产生 canonical result event。
- GPV 9005 作为设备不支持该路径的正常能力差异：记录原 task 和 bad path，不创建 `-r` task，不阻断同一设备队列中的后续批次。
- 非 9005 的通信、协议或系统错误进入 cancelling，并通过取消 Outbox 驱逐尚未执行的批次。

### ResultRef 第一阶段实现

第一阶段可使用主库 `device_task_result_payloads`/现有 result JSON 的数据库引用，不强制立即接入对象存储；事件必须只携带引用，保证后续无协议变更迁移到 MinIO。

### 测试

- 成功、Fault、expired、cancelled 都产生一次 canonical event。
- 事件体不包含 raw response。
- 9005 不增加 expected count，后续预入队批次继续执行。
- NATS publish 失败时 Result Reconciler 可从 PG 补发。

## A6. 唯一 ResultConsumer 与 ACK-after-commit

### 消费模型

- 使用 durable pull consumer。
- 按 device ID 固定分片。
- handler 将事件提交 worker 后等待处理结果，不能立即返回。
- worker 成功提交事务后 handler 返回 nil，EventBus 才 ACK。
- 数据库错误返回 error，触发 NAK/redelivery。

### 单条结果事务

1. 校验 task/run/device 关联。
2. `INSERT task_result ON CONFLICT DO NOTHING`。
3. 已存在则幂等成功。
4. 读取 ResultRef。
5. 使用 MappingValidator 翻译和过滤。
6. full 写 staging，partial/readback 按各自策略写正式参数。
7. 更新 terminal/processed/failed 计数。
8. 判断是否可领取 Finalizer。
9. 提交后 ACK。

### 错误分类

- 临时 DB/NATS/对象存储错误：返回 error，重投。
- 永久数据格式错误：写 result failed、run failed，并将原事件 Term/DLQ。
- task/run 不匹配：安全拒绝写参数，记录安全审计和指标。

### 现有消费者切换

- `device.RPCResponseSubscriber` 遇到 `source=param_sync` 必须直接跳过。
- `ProvisioningEngine` 删除参数同步 GPV 的落库订阅。
- 普通 MML、SPV readback、AddObject readback 可暂时保留原 subscriber，随后按 scope 逐步收敛。

### 测试

- 数据库失败时消息 NAK，恢复后重投成功。
- 相同事件投递五次只处理一次。
- worker 收到事件后进程退出，消息重新投递。
- 两个 consumer 实例竞争同一事件不会重复落库。
- 同设备结果按 shard 保序，不同设备可并行。

## A7. Full staging 与 Finalizer

### Full 结果写入

- 每批结果只写 `parameter_sync_staging_values`。
- staging 保存 standard path、private path、value、type、writable、coverage scope。
- 不在批次处理阶段更新 `device_parameters`、`device_info` 或 `last_param_sync_at`。

### Finalize 条件

```text
terminal_task_count = expected_task_count
processed_task_count = expected_task_count
failed_task_count = 0
```

通过 `UPDATE ... WHERE status IN (...) RETURNING` CAS 领取。

### Full Finalizer 事务

1. Run 转 `processing`。
2. 批量 merge staging → `device_parameters`。
3. 基于整轮 seen paths 计算 missing diff。
4. 仅对计划声明完整覆盖且成功的对象执行 reconcile。
5. 刷新 `device_info`。
6. 执行 device name sync hook。
7. 更新 `last_param_sync_at` 并清空失败字段。
8. Request/Run 转 succeeded。
9. 发布 run completed outbox。
10. 清理 staging，或标记待异步清理。

### Partial/readback Finalizer

- partial 只更新请求路径和 request/run 终态。
- readback 只更新回读参数。
- 二者都禁止更新全量时间、执行 missing diff/reconcile 和名称全量 hook。

### 失败收口

- 任一不可恢复 task 失败：Run/Request failed。
- full staging 不提交。
- 写 `last_param_sync_failed_at/error`。
- 取消该 run 剩余 pending task。
- 发布 run failed outbox。

### 测试

- 多批 full 只 finalize 一次。
- worker backlog 下第一批结果不能提前 finalize。
- 最后一批结果使用整轮 staging 做 diff。
- partial/readback 不触发任何 full 副作用。
- Finalizer 中途失败事务整体回滚，可安全重试。

## A8. Provisioning 状态联动

### Binding

Provisioning 触发 full run 时创建：

```text
request_id → run_id → provisioning_task_id
```

规则：

- 新建 run：Provisioning 在 run 建立成功后转 `syncing`。
- 加入已有兼容 full run：创建 binding，再转 `syncing`。
- Run succeeded：所有未终态 binding 转 completed。
- Run failed/timed_out：所有未终态 binding 转 failed。
- No storable path：Request 和 Provisioning 直接 completed。
- 不再依赖普通 task.completed 数量推进 Provisioning。

### 删除旧逻辑

- `shouldFinalizePathBSync`。
- Redis pending batches。
- recovered GPV 特殊 finalize callback。
- `handleAutoSync` 对 `(true, 0, nil)` 的模糊处理。

### 测试

- 正常 single/multi batch 均完成 Provisioning。
- 防重加入已有 run 后随 run 完成。
- run 失败、超时和无路径均进入明确终态。
- 重复 run terminal event 不重复发布 provisioning completed/failed。

## A9. 状态 API、前端与统计迁移

### 后端 API

新增：

```text
GET /parameter-sync/requests/:request_id
GET /devices/:id/parameter-sync/active
GET /devices/:id/parameter-sync/history
```

现有参数树 sync status 改为从 SyncRun/Request 读取，不再：

- CountOpenSyncGPVByDevice 猜测状态。
- LatestSyncGPVSummaryByDevice 使用 30 秒时间窗口聚合。
- 从 command key 推断同步类型。

### 前端

- `syncDeviceParams` 保存 request ID/run ID。
- 参数树和快速设置继续共享设备级 busy 状态。
- 完成判断查询 request 终态，不再用 `last_param_sync_at >= startedAt` 猜测。
- 展示 request 级成功/失败路径统计。
- 409 展示当前 active request/run 信息。

### 兼容期

- 现有 `/sync-status` DTO 保持字段兼容。
- 新增 `active_request`、`active_run`、`last_request`。
- 前端切换完成后再删除 `last_sync_gpv` 旧 SQL 聚合。

### 测试

- 参数树和快速设置看到同一 active run。
- scoped sync 完成后按 request 终态退出等待。
- 失败路径统计：失败 command 的全部 requested paths 都失败，param_faults 只补详情。
- 页面刷新和换浏览器后仍能恢复当前同步状态。

## A10. 补偿器、指标与运维接口

### 第一阶段补偿器

- Request Sweeper：accepted/queued/running 超 deadline → timed_out。
- Run Reconciler：从 device_tasks 重建 expected/terminal/failed。
- Result Reconciler：task 已终态但 task result 缺失 → 补发/补处理。
- Staging Cleaner：清理超保留期 terminal run staging。
- Outbox Dispatcher：重试 pending/failed outbox。

### 指标

```text
param_sync_requests_total{scope,reason,status}
param_sync_runs_active{status}
param_sync_request_duration_seconds
param_sync_run_duration_seconds
param_sync_tasks_expected/terminal/processed/failed
param_sync_result_redelivery_total
param_sync_finalize_total{result}
param_sync_outbox_backlog
param_sync_staging_rows
param_sync_reconcile_repairs_total
```

### 告警

- Request queued age 超阈值。
- Run processing 长时间不结束。
- task terminal 与 processed 差值持续增长。
- Outbox backlog 持续增长。
- Result redelivery/Term/DLQ 激增。
- 单产品型号失败率异常。

### 运维接口

- 查询 request/run/task 关联。
- 重放指定 task result。
- 重试指定 failed request。
- 取消 queued/running request。
- 查看并重投 dead outbox。

## A11. 灰度切换与旧链路清理

### 上线顺序

1. 发布纯增量数据库迁移。
2. 发布新代码，所有新开关关闭。
3. 启用 Outbox/补偿器但不接管参数同步。
4. `canary_percent=1`，仅哈希命中设备走新链路。
5. 确认新 task `source=param_sync`，旧 subscribers 对其跳过。
6. 逐步扩大到 5%、20%、50%、100%。
7. 100% 稳定后停止创建 legacy sync-gpv run。
8. 等待 legacy open task 自然完成或过期。
9. 删除旧 pending/reason Redis key 和旧 summary SQL。

### 灰度不变量

- 单个 task 只能被新或旧结果写入者之一处理。
- 单个设备同一时刻不能同时存在 legacy 和 new full sync。
- 切换新链路前必须检查该设备没有 legacy open sync-GPV task。

### 回滚

- 停止接收新链路 request。
- 已创建的新 run 允许继续收口；若新 consumer 故障，则由 Result Reconciler 保留待恢复结果。
- 新请求重新走 legacy fallback。
- 数据库对象全部保留，不执行破坏性 Down。
- 回滚不得重新启用旧 subscriber 去消费已经属于新 run 的 task。

## 5. 交付 A 验收门槛

必须全部满足才能进入 100%：

1. 连续运行期内 request accepted 与 terminal 数量最终守恒。
2. 同设备 active full run 数永远不超过 1。
3. succeeded full run 满足 expected=terminal=processed 且 failed=0。
4. 数据库、Redis、NATS 分别注入短时故障后能自动恢复。
5. 重复事件不会产生重复参数、重复计数或重复 hook。
6. 多批 full 的 diff/reconcile 使用整轮快照。
7. partial/readback 不更新 `last_param_sync_at`。
8. Provisioning 所有路径均能进入终态。
9. 参数树和快速设置使用 request 状态完成闭环。
10. 旧 subscriber 不再写 `source=param_sync` 的结果。
11. `go build ./...`、`go test ./...`、前端 typecheck 全部通过。
12. 关键链路具备 dashboard 和告警。

## 6. 交付 B：10 万设备控制面

## B1. SyncCampaign

### 实现内容

- 新增 `parameter_sync_campaigns`。
- 周期同步从直接扫描并调用 StartPathBSync 改为创建 Campaign。
- 使用 `(last_device_id, limit)` 游标分页选择设备。
- 每轮按 backlog 水位生成 500～2000 个 request。
- Campaign 汇总 total/generated/succeeded/failed/deduplicated/timed_out。
- 支持 pause/resume/cancel。

### 修改位置

- 新增 `paramsync/campaign.go`。
- 替换 `provision/periodic_syncer.go` 的直接入队逻辑。
- 系统配置增加同步窗口、目标速率和 Campaign batch size。

### 测试

- 10 万模拟设备不会一次创建全部 task。
- Campaign 重启后从游标恢复且不重复生成 request。
- pause/resume/cancel 保持计数正确。

## B2. 分布式 Scheduler

### 实现内容

- 多实例 `FOR UPDATE SKIP LOCKED` 领取 queued request。
- 按 priority、created_at 和分组权重调度。
- 同设备 active run 唯一索引作为最终防线。
- 请求获得执行令牌后才创建 run/task。
- 区分 waiting_device、executing、processing 资源计数。

### 配额

- global start rate。
- global executing devices。
- per ACS instance。
- per carrier/region/product。
- per device queue depth。
- PG/NATS/Redis 水位保护。

### 测试

- 多 scheduler 实例不重复领取 request。
- 一个故障分组不占满所有额度。
- 高优先级不导致周期任务永久饥饿。
- 动态调整配额无需重启。

## B3. 固定分片 Result Worker

### 实现内容

- `hash(device_id) % shard_count` 路由。
- 每个 shard 固定 worker 和有界队列。
- 同设备事件保序，不同设备并行。
- shard 满时 consumer NAK，不能丢到无界内存。
- poison message 达最大投递后进入 DLQ，并失败对应 request/run。

### 测试

- 单设备慢处理不阻塞其他 shard。
- 不创建每设备常驻 goroutine。
- 队列满、进程退出、消息重投行为正确。

## B4. 批量写入与数据生命周期

### 实现内容

- staging 使用 `pgx.CopyFrom` 或 `unnest` 批量写。
- 正式参数 merge 使用集合 SQL，禁止逐行 upsert。
- 评估 `device_parameters` 按 device hash 分区。
- request/run/task result 按月分区或设置归档策略。
- staging 只保留 active/recent failed run。
- raw result 迁移 MinIO，PG 保存 result_ref 和摘要。

### 基准目标

- 参数写入吞吐高于目标 task/result 吞吐两倍。
- Finalizer P95 不随单设备参数数线性恶化到不可接受范围。
- 清理任务不长时间锁正式参数表。

## B5. 容量控制与动态速率

### 计算公式

```text
start_rate = device_count / sync_window_seconds
task_rate = start_rate * avg_tasks_per_device
executing_concurrency = start_rate * avg_execution_duration
```

以 10 万设备 6 小时完成、平均 30 task、平均 24 秒为初始模型：

```text
start_rate ≈ 4.63 device/s
task_rate ≈ 139 task/s
executing ≈ 111 devices
```

初始设计按两倍余量：

```text
10 device/s
300 task/s
250 executing devices
```

Scheduler 根据以下反馈降速：

- PG transaction latency/connection saturation。
- NATS ack pending/redelivery。
- Redis queue depth。
- ACS active sessions/RPC latency。
- Result worker queue depth。

## B6. 10 万设备压测与故障演练

### 数据模型

- 10 万设备记录。
- 多产品型号和不同 MappingSet。
- 混合在线、离线、慢设备、Fault 设备。
- 参数规模分布覆盖小型、典型和 BSC 大对象。

### 压测场景

1. 10 万设备 24 小时平滑同步。
2. 10 万设备 6 小时窗口。
3. 大量设备同时上线形成突发。
4. 5% 慢设备、1% Fault、1% 长期离线。
5. Scheduler/Consumer/ACS 多实例扩缩容。
6. PG、Redis、NATS 单点短时不可用。
7. ResultConsumer 处理中重启。
8. Outbox 大量积压后恢复。

### 通过标准

- 每个 request 最终进入可解释终态。
- 单设备失败不降低其他设备成功率或阻塞其执行。
- 无重复 finalize、无参数跨设备写入。
- 全局吞吐达到配置同步窗口要求。
- queued age、result latency 和 PG/NATS/Redis 水位处于预设阈值。
- 停止流量后 backlog 能在可预测时间内归零。

## 7. 工作包依赖关系

```text
A0 契约/开关
 └─ A1 数据库/仓库
     ├─ A2 Request API
     ├─ A3 Planner
     └─ A4 Task Outbox
          └─ A5 ACS Result Event
              └─ A6 ResultConsumer
                  └─ A7 Finalizer
                      ├─ A8 Provisioning 联动
                      ├─ A9 API/前端
                      └─ A10 补偿/指标
                           └─ A11 灰度切换
                                └─ 交付 A 完成
                                     ├─ B1 Campaign
                                     ├─ B2 Scheduler
                                     ├─ B3 分片 Worker
                                     └─ B4 批量写/分区
                                          └─ B5 动态容量
                                              └─ B6 10 万压测
```

可并行项：

- A2 与 A3 可在 A1 接口冻结后并行。
- A5 事件契约与 A4 Outbox 可并行开发。
- A8、A9、A10 可在 A7 稳定后并行。
- B1、B2、B3、B4 可基于交付 A 接口并行。

## 8. 建议拆分的 Issue

建议一个 Issue 只覆盖一个可独立验收的纵向切片：

1. `feat(paramsync): 建立 Request/Run 领域模型和迁移`
2. `feat(paramsync): 提供手工同步 Request API 和设备级互斥`
3. `refactor(paramsync): 抽取 Path B Planner 和结构化 scope`
4. `feat(task): 支持参数同步批量 task 与 Outbox`
5. `feat(acs): 发布轻量参数同步结果事件`
6. `feat(paramsync): 实现幂等 ResultConsumer 和 ACK-after-commit`
7. `feat(paramsync): 实现 full staging 与唯一 Finalizer`
8. `fix(provision): 使用 SyncRun 终态收口 provisioning`
9. `refactor(device): 参数同步结果退出通用 RPCResponseSubscriber`
10. `feat(paramsync): 迁移状态 API 和前端请求跟踪`
11. `feat(paramsync): 增加 Outbox/Sweeper/Reconciler 和指标`
12. `chore(paramsync): 灰度切换并删除 legacy sync 状态推断`
13. `feat(paramsync): 增加 SyncCampaign 渐进调度`
14. `feat(paramsync): 增加分布式配额 Scheduler`
15. `perf(paramsync): 分片 Result Worker 和批量参数写入`
16. `perf(paramsync): 数据分区与大结果对象存储`
17. `test(paramsync): 10 万设备容量压测和故障演练`

## 9. Definition of Done

每个实现 Issue 必须满足：

- 代码遵循 handler → service → repository/model 分层。
- SQL 使用 Squirrel、参数化 SQL 或静态集合 SQL，禁止字符串拼接用户输入。
- 有单元测试和必要的 PG/Redis/NATS 集成测试。
- 新状态、失败和补偿路径有指标。
- 有功能开关或兼容路径，不要求破坏性一次切换。
- 文档同步更新 API、状态机和运维说明。
- `git diff --check` 通过。
- `cd omcgo && go build ./... && go test ./...` 通过。
- 涉及前端时 `cd omcmb && npm run typecheck` 通过。
- 涉及 compose/NATS 时相应 compose config 校验通过。

## 10. 2026-07-15 执行面闭环补充（已落地）

交付 A 的可靠数据面保留原有 `SyncRequest / SyncRun / Result / Outbox / Staging` 契约，执行面补充以下强制约束：

1. 参数同步完整批次计划及全部 enqueue Outbox 一次性写入；Outbox 将同一 run 的批次按顺序放入设备 Redis 队列，ACS 在同一个 CWMP 会话内严格执行 `GPV批次1 → 响应 → GPV批次2 → 响应 → … → 会话结束`。禁止使用“Redis 放行窗口固定为 1”的逐批异步释放方式。
2. Redis 入队、容量检查、唤醒和驱逐统一由 `TaskService` 承担，PARAM_SYNC Outbox 不再直接操作 Redis 队列；批量 Outbox 使用“全部入队后按设备唤醒一次”的接口，不能形成批次数量级的唤醒风暴。
3. 9005 记为设备不支持且不重试，继续后续批次；首次非 9005 真实失败不直接写 `failed`，而是进入 `cancelling`：未发送批次转为 `cancelled`、生成规范化结果并写取消 Outbox；已发送批次等待响应或过期。
4. `expected = terminal = processed` 后才允许从 `cancelling` 进入 `failed/cancelled`，运行计数始终从 `device_tasks + parameter_sync_task_results` 推导。
5. Enqueue Outbox 在 Redis Push 前后都校验运行状态；取消与入队并发时，失效任务会被驱逐，避免终态运行继续下发。
6. `device_online / periodic / firmware_changed` 使用 PostgreSQL 持久化退避：`1m → 5m → 15m → 1h → 6h`，成功后重置。
7. 活动运行、任务计数、Outbox backlog 和 staging rows 指标由数据库快照刷新；失败运行收敛后立即清理 staging。
8. 性能验收必须区分 GPV RPC 时间与批次间等待；正常可唤醒设备不应为每个批次创建新 Inform/Connection Request 会话。
9. 滚动升级期间，旧版本已创建且只含首批 enqueue Outbox 的 active run 允许由 ResultConsumer 幂等补齐下一批；新 run 因全部 Outbox 已存在，该兼容语句必须为零操作，待旧 run 排空后可删除。

执行面控制结构已合并进初始迁移：`000021_parameter_sync_runtime.sql`。
