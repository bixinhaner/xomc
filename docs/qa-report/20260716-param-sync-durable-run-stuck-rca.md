# 参数页“同步中”无法结束 - durable paramsync 收口卡住 RCA

- 报告日期：2026-07-16
- 现场环境：`http://172.24.224.78:8081`
- 前端版本：`v100.0.0-20260716-1405`
- 复现页面：设备详情 -> 参数树
- 复现设备：`120200054822CHB0004`
- device_id：`55d680e6-eba4-4ecc-89b2-5fa90b79fb23`
- run_id：`dbb9f001-bde0-46eb-9f22-e88ce0fff225`
- request_id：`5c44a3eb-29b3-4f92-9fa5-99e2f70b99fd`

## 1. 现象

用户进入设备参数页后，页面顶部持续显示“同步中”，`同步参数`按钮禁用，等待数十分钟仍不恢复。

前端持续轮询：

```text
GET /api/v1/devices/55d680e6-eba4-4ecc-89b2-5fa90b79fb23/parameter-sync/active
GET /api/v1/devices/55d680e6-eba4-4ecc-89b2-5fa90b79fb23/parameters/sync-status
```

其中 `/parameter-sync/active` 一直返回 `active_run`，所以前端按后端状态持续展示“同步中”。

## 2. 已核实现场证据

### 2.1 当前 run/request 状态

现场 PostgreSQL 查询结果：

```text
parameter_sync_runs:
id = dbb9f001-bde0-46eb-9f22-e88ce0fff225
status = waiting_device
expected_task_count = 38
terminal_task_count = 38
processed_task_count = 0
failed_task_count = 0
completed_at = NULL
version = 16

parameter_sync_requests:
id = 5c44a3eb-29b3-4f92-9fa5-99e2f70b99fd
status = running
run_id = dbb9f001-bde0-46eb-9f22-e88ce0fff225
active_run_id = dbb9f001-bde0-46eb-9f22-e88ce0fff225
completed_at = NULL
```

### 2.2 设备任务已经全部终态

```text
device_tasks where source='param_sync' and source_id=run_id:
completed = 38
```

但 durable paramsync 结果表为空：

```text
parameter_sync_task_results where run_id=run_id:
0 rows

missing_results = 38
```

### 2.3 sync-status 已经显示无待处理命令

页面接口返回的同步状态摘要：

```text
parameters/sync-status:
status = idle
pending_commands = 0
last_param_sync_at = 2026-07-16T06:40:22.779859Z
last_sync_gpv.task_count = 38
last_sync_gpv.successful_commands = 38
last_sync_gpv.last_completed_at = 2026-07-16T06:41:49.351445Z
```

这说明南向 GPV 任务已经完成；卡住的是 durable paramsync run/request 的官方收口链路。

### 2.4 不是单设备个案

现场全局统计：

```text
parameter_sync_runs:
waiting_device = 2766
executing = 166
succeeded = 19

param_sync terminal device_tasks = 56144
missing parameter_sync_task_results = 51071
```

这说明 durable paramsync 结果处理大面积落后，不是某一台设备的单点问题。

### 2.5 NATS result consumer 严重积压

现场 NATS `PARAM_SYNC` stream：

```text
PARAM_SYNC stream:
messages = 109484

consumer param-sync-results-pull:
delivered.consumer_seq = 10243
ack_floor.consumer_seq = 10217
num_ack_pending = 26
num_pending = 109458
```

含义：

- `param_sync.task_result` 消息大量积压在 NATS 中。
- `param-sync-results-pull` 只处理到约 1 万条，后面还有约 10.9 万条待消费。
- 当前设备这 38 条 result 没有进入 `parameter_sync_task_results`，符合“排在积压队列后面/消费端追不上”的形态。

### 2.6 app 日志显示 DB 写入持续超时

现场 `omcgo-app-1` 日志持续出现：

```text
parameter sync outbox dispatch failed:
mark parameter sync outbox delivered: context deadline exceeded

parameter sync outbox dispatch failed:
mark parameter sync outbox failed: context deadline exceeded
```

同一时间段还出现设备侧查询超时：

```text
auto group-assign failed (heartbeat path): context deadline exceeded
lookup disconnected alarm on device online failed: context deadline exceeded
```

这说明 PostgreSQL 在现场压力下已经出现普遍超时。result consumer 每处理一条结果都要执行较重的 DB 事务，因此吞吐进一步下降。

## 3. 直接原因

durable paramsync run 的收口条件是：

```go
ReadyToFinalize =
  terminal_task_count == expected_task_count &&
  processed_task_count == expected_task_count
```

现场为：

```text
expected_task_count = 38
terminal_task_count = 38
processed_task_count = 0
```

所以状态机认为：

```text
设备任务都已经终态，但这些任务结果还没有被 durable paramsync 官方结果处理器处理。
```

因此 run 不能从 `waiting_device` 推进到 `succeeded`，request 也不能清空 `active_run_id`，前端持续看到 `active_run`，最终表现为“同步中”无法结束。

## 4. 根因链路

当前实时路径为：

```text
device_task 终态
  -> param_sync.task_result 事件
  -> NATS PARAM_SYNC stream
  -> param-sync-results-pull consumer
  -> PGResultProcessor.Process()
  -> parameter_sync_task_results.status = processed/failed
  -> processed_task_count 推进
  -> run/request finalize
```

现场卡在：

```text
NATS PARAM_SYNC stream
  -> param-sync-results-pull consumer
```

具体表现：

```text
result 消息积压约 10.9 万条
PGResultProcessor 没能及时处理
parameter_sync_task_results 大量缺失
processed_task_count 不推进
run/request 不收口
```

所以根因不是：

- 不是前端 loading 状态忘记关闭。
- 不是这台设备 38 个 GPV 没完成。
- 不是 `sync-status` 本身错误。

根因是：

```text
durable paramsync 的实时 result 消费链路在高负载/DB 超时下吞吐崩塌，且当前补偿机制仍依赖同一条拥堵事件链路，导致已完成的 device_tasks 无法及时转化为 processed results。
```

## 5. 为什么之前这样设计

之前把 `terminal_task_count` 和 `processed_task_count` 分开，是有意为之。

```text
terminal_task_count:
  device_tasks 是否进入 completed/failed/expired/cancelled。

processed_task_count:
  这些终态 task 是否已经经过 durable paramsync 官方结果处理器处理。
```

这个分离的价值是避免“看见 device_tasks completed 就提前宣布同步成功”。因为 `device_tasks.completed` 只说明 ACS 收到了 GPV 响应，不说明 durable paramsync 已经完成：

- 解析 GPV raw response。
- 投影参数到 staging/device_parameters。
- 处理 9005 recovery。
- 标记 coverage incomplete。
- 更新 `parameter_sync_task_results`。
- 合并 full sync staging。
- 写 `last_param_sync_at`。
- 发 terminal outbox。
- 清理 request active_run_id。

原设计让所有路径最终回到 `PGResultProcessor`，是为了保持单一业务语义，避免多个地方各自判断成功造成双写、早收口和竞态。

## 6. 设计盲点

已有补偿 `Reconciler.RepublishMissingResults()` 能发现：

```text
device_tasks 已终态，但 parameter_sync_task_results 缺失
```

但它当前做的是“重新发布 result 事件”，也就是：

```text
missing result
  -> republish param_sync.task_result
  -> param-sync-results-pull consumer
  -> PGResultProcessor
```

当 `param-sync-results-pull` 本身已经积压 10 万级时，补偿继续把消息发回同一条拥堵队列，无法快速收口，甚至会加重积压。

这是本次暴露出的核心设计盲点：

```text
实时路径被设计成唯一收口路径；补偿路径也依赖实时路径。
```

## 7. 修改方案

目标不是删除实时路径，也不是只做补偿。更合理的顺序是：

```text
先优化正常路径，降低积压概率；
再补齐 DB recovery，保证积压后仍然最终收口。
```

最终架构应调整为：

```text
正常路径：ACS/task terminal -> NATS -> ResultConsumer -> PGResultProcessor
兜底路径：DB Reconciler -----------------------> PGResultProcessor
```

### 7.1 正常路径正向优化

实时路径仍然有价值：

- 正常负载下秒级完成同步。
- 解耦 ACS 与 APP，避免 ACS 在南向会话里同步等待重 DB 写入。
- 用 NATS 承接瞬时峰值。
- 保持设备协议处理链路短。

因此不建议把所有结果处理都改成不走 NATS。正向优化应优先降低 `PARAM_SYNC` 积压概率，并避免系统继续制造超过 result consumer/PG 能力的 backlog。

#### 7.1.1 ResultConsumer 分片并发

- 按 `device_id` 或 `device_sn` hash 分片。
- 同一设备进入同一 shard，保证单设备结果有序。
- 不同设备并行处理。
- 每个 shard 使用有界队列；队列满时 NAK/redelivery，不落入无界内存。

这里建议复用 `provision` GPV worker 的 shard worker 思路，而不是采用 PM 文件入库的多 `QueueSubscribe` 并发模式。

原因是 paramsync result 不是独立文件处理，而是设备/run 状态机收口：

- 同一设备、同一 run 的多个 batch result 并发写入会增加 run lock、finalize 抢占和事务等待。
- NATS 多 queue subscribe 会把消息负载均衡到任意订阅，业务侧不能保证同设备进入同一处理线程。
- shard worker 可以做到同设备串行、不同设备并行，既提高吞吐，又降低单 run 内竞态。
- shard 队列水位还能暴露具体是哪类设备或 shard 拖慢处理，背压粒度比单纯依赖 consumer pending 更清晰。

PM 文件入库适合多 `QueueSubscribe`，因为文件之间基本独立，目标是吃满 CPU/IO；参数同步 result 更适合单 pull consumer + shard worker pool。

#### 7.1.2 拉取、ACK 与错误分类

- Pull consumer 批量拉取，减少单消息调度开销。
- 只有 `PGResultProcessor.Process()` 事务 commit 成功后才 ACK。
- DB timeout、deadlock、连接池耗尽等临时错误返回 error，让 NATS redelivery。
- payload 格式错误、task/run/device 不匹配等永久错误进入 DLQ 或显式 fail run，避免无限重投。

#### 7.1.3 DB 事务瘦身

`PGResultProcessor` 事务内只保留必须强一致的步骤：

```text
insert task result
  -> 写 staging/正式参数
  -> 更新 processed/failed count
  -> 判断并执行 finalize
```

以下副作用尽量通过 outbox 或后置 projector 异步执行：

- `device_info` refresh。
- device name sync hook。
- terminal notification/binding projection。
- 其他非本轮结果处理强一致要求的统计或投影。

#### 7.1.4 SQL 与索引优化

优先检查和补强这些查询路径：

- `device_tasks(source, source_id, status)`。
- `parameter_sync_task_results(run_id, task_id)`。
- `parameter_sync_runs(status, updated_at/started_at)`。
- `parameter_sync_outbox(status, next_attempt_at, created_at)`。
- active run 查询相关索引。

同时对以下重 SQL 做 `EXPLAIN ANALYZE`：

- `loadAuthoritativeRunCounts`。
- staging 写入。
- full finalize 的 staging -> `device_parameters` merge。
- request/run active 查询。

#### 7.1.5 批量写入

- full sync staging 使用批量 insert/upsert。
- full finalize 使用集合 SQL merge，禁止按参数逐行 upsert。
- 大对象参数多的设备单独压测，确认单 run finalizer P95 不随参数量线性恶化到不可接受范围。

#### 7.1.6 背压与限流

出现以下任一情况时，应降低或暂停新的 full sync：

- `PARAM_SYNC` consumer pending 超阈值。
- `terminal_task_count - processed_task_count` 持续增长。
- PG 写超时率升高。
- outbox pending/failed 持续增长。

背压优先级建议：

```text
先降周期同步、设备上线自动同步；
再降批量 campaign；
手工同步保留较高优先级，但仍受单设备 active run 和全局水位限制。
```

outbox dispatch、request dispatch、result consumer、DB recovery 必须使用独立 ticker、timeout 和限流参数，避免一个链路阻塞后拖死其他修复链路。

### 7.2 兜底路径直接调用同一个 PGResultProcessor

正常路径优化只能减少积压概率，不能保证积压发生后一定收口。兜底路径要解决的是：

```text
device_tasks 已终态
parameter_sync_task_results 缺失
processed_task_count 不推进
```

将 `RepublishMissingResults()` 升级为直接 DB recovery：

```text
扫描 DB 中 missing terminal task
  -> 构造 ParamSyncTaskResultPayload
  -> 直接调用 PGResultProcessor.Process()
  -> 写 parameter_sync_task_results
  -> 推进 processed_task_count
  -> 满足条件后 finalize run/request
```

关键点：

```text
补偿路径不能直接 update run.status='succeeded'。
补偿路径必须复用 PGResultProcessor，保持和实时路径完全一致的业务语义。
```

这样既不绕过 staging、9005、coverage、失败收敛、terminal outbox，也不再把补偿排到 NATS 积压队列后面。

### 7.3 按 run 聚合优先恢复

优先处理这种 run：

```text
run.status in ('waiting_device','executing','processing')
terminal_task_count = expected_task_count
processed_task_count < expected_task_count
```

按 run 内任务顺序处理：

```text
ORDER BY command_index, created_at, id
```

虽然状态机理论上依赖幂等和计数重算，不要求严格顺序，但按原任务顺序恢复更接近实时路径，降低引入新竞态的风险。

### 7.4 独立 recovery loop

将 result recovery 从 outbox dispatch 的高频循环中拆出来：

- 独立 ticker。
- 独立 timeout。
- 独立限流。
- 独立日志与指标。

推荐初始参数：

```text
每轮最多 20 个 run
每个 run 最多 200 个 missing task
单轮 timeout 20s
```

避免在 PG 已经高压时继续制造更大的写入洪峰。

### 7.5 outbox 和 result recovery 分离限流

当前日志显示 outbox dispatch 自身也在 DB timeout。后续需要：

- 限制 `param_sync.task.enqueue` dispatch 速率。
- result recovery 不应被 enqueue backlog 阻塞。
- 当 result backlog 超阈值时，暂停或降低新的 full sync enqueue。

### 7.6 API/前端体验兜底

后端 `/parameter-sync/active` 可以增加诊断字段：

```text
stalled_finalizing = true
reason = "terminal tasks completed but results are not processed"
```

前端可以将这种状态展示为“同步结果收口中”，而不是普通“同步中”。但这只是体验兜底，不能替代后端 DB recovery。

## 8. 建议代码改动点

### 8.1 正常路径优化改动点

#### `omcgo/internal/paramsync/result_consumer.go`

- 将单 handler 同步处理扩展为固定 shard worker pool。
- shard key 使用 `device_id` 或 `device_sn`，保证同设备保序。
- 每个 shard 使用有界队列；队列满时返回 error，让 NATS redelivery。
- 保持 ACK-after-commit：只有 `PGResultProcessor.Process()` 成功返回后，handler 才返回 nil。
- 对永久错误增加明确分类，避免同一 poison message 无限占用消费能力。

建议处理流程：

```text
PullSubscribe handler 收到消息
  -> decode ParamSyncTaskResultPayload
  -> 按 device_id/device_sn 选择 shard
  -> 构造 work item，携带 done channel
  -> 投递到 shard queue
  -> handler 等待 done channel
  -> worker 调用 PGResultProcessor.Process()
  -> worker 将处理 error 写回 done channel
  -> handler 根据 error 返回 nil/error
  -> EventBus 按返回值 ACK/NAK/Term
```

禁止 handler 只把消息放入 shard queue 就返回 nil；否则进程在 worker 写库前崩溃时，NATS 已经 ACK，result 会静默丢失。

初始参数建议：

```text
shard_count = 8
queue_depth_per_shard = 256
```

后续根据 PG 连接池、单 result 事务耗时和 NATS pending 水位调优。`shard_count` 不应盲目超过 PG 可承受并发，否则会把 NATS 积压转移成 PG 锁等待和连接池等待。

#### `omcgo/internal/paramsync/result_processor.go`

- 收窄事务内工作，只保留 result 落库、参数/staging 写入、计数推进和 finalize 判断。
- `device_info` refresh、device name sync、terminal binding projection 等非强一致副作用优先迁到 outbox/projector。
- 对 `loadAuthoritativeRunCounts`、staging 写入、full finalize merge 做 `EXPLAIN ANALYZE`，根据实际执行计划补索引或改集合 SQL。
- full staging 写入和正式参数 merge 优先批量化，避免按参数逐行 upsert。

#### `omcgo/internal/paramsync/outbox.go` 和 request dispatch

- result backlog 高时降低 `param_sync.task.enqueue` dispatch 速率。
- request dispatch、enqueue outbox、result consumer 使用独立限流，避免周期同步或上线自动同步继续放大 backlog。
- 对手工同步保留较高优先级，但仍受单设备 active run 和全局水位保护。

#### 数据库与指标

- 检查并补强 `device_tasks(source, source_id, status)`、`parameter_sync_runs(status, updated_at/started_at)`、`parameter_sync_outbox(status, next_attempt_at, created_at)` 等索引。
- 增加 result consumer TPS、处理耗时、redelivery、DLQ、`terminal_task_count - processed_task_count`、PG timeout/connection wait 指标。

### 8.2 `omcgo/internal/paramsync/reconciler.go`

新增或改造：

```go
type Reconciler struct {
    pool      *pgxpool.Pool
    bus       event.EventBus
    processor ResultProcessor
    metrics   *Metrics
}
```

新增方法：

```text
RecoverMissingResults(ctx, runLimit, taskLimit int) (int, error)
```

行为：

1. 找 active 且 `terminal=expected`、`processed<expected` 的 run。
2. 查询该 run 下缺失 `parameter_sync_task_results` 的 terminal tasks。
3. 对每个 task 构造 `ParamSyncTaskResultPayload`。
4. 调用 `processor.Process(ctx, payload)`。
5. 幂等跳过 duplicate。

保留原 `RepublishMissingResults()` 作为没有 processor 时的兼容路径，或者逐步废弃。

### 8.3 `omcgo/cmd/app/provider/paramsync.go`

初始化时复用同一个 processor：

```go
processor := paramsync.NewPGResultProcessor(c.PgPool).WithMetrics(metrics)
reconciler := paramsync.NewReconciler(c.PgPool, c.EventBus, metrics).WithResultProcessor(processor)
resultConsumer := paramsync.NewResultConsumer(c.EventBus, processor)
```

维护循环顺序建议：

```text
ReconcileRunCounts
RecoverMissingResults
ReconcileTerminalBindings
ReconcilePending projections
CleanStaging
CollectMetrics
```

`ReconcileRunCounts` 放在 `RecoverMissingResults` 前面，是为了先用 `device_tasks` 和已有 result 表重建权威计数，再选择 `terminal=expected` 且 `processed<expected` 的 run 做直接恢复。

### 8.4 `omcgo/internal/paramsync/result_processor.go`

确认并补足幂等：

- 同一 `run_id + task_id` 重复处理不重复写。
- 已 terminal run 不被重新处理。
- reconciliation event id 与实时 event id 不冲突。

### 8.5 测试

新增集成测试：

```text
given:
  run.status = waiting_device
  expected_task_count = 1
  terminal_task_count = 1
  processed_task_count = 0
  device_tasks.status = completed
  parameter_sync_task_results 缺失

when:
  RecoverMissingResults()

then:
  parameter_sync_task_results.status = processed
  processed_task_count = 1
  run.status = succeeded
  request.status = succeeded
  request.active_run_id = NULL
```

再加一个失败场景：

```text
device_tasks.status = failed
RecoverMissingResults()
run.status = failed
request.status = failed
last_param_sync_failed_at 被写入
```

## 9. 观测与告警

需要新增/补齐指标：

```text
param_sync_results_consumer_pending
param_sync_results_consumer_tps
param_sync_results_consumer_duration_seconds
param_sync_result_dlq_total
param_sync_missing_results_total
param_sync_active_runs_terminal_unprocessed
param_sync_recovery_processed_total
param_sync_recovery_failed_total
param_sync_outbox_pending
param_sync_outbox_dispatch_timeout_total
```

建议告警：

```text
terminal_task_count = expected_task_count
AND processed_task_count < expected_task_count
持续超过 2 分钟
```

以及：

```text
param-sync-results-pull num_pending > 10000
持续超过 5 分钟
```

## 10. 总结

本次“同步中”无法结束的本质是：

```text
设备任务已经完成，但 durable paramsync 的 result consumer 积压严重，导致 parameter_sync_task_results 大面积缺失，processed_task_count 不推进，run/request 无法 finalize。
```

彻底修复不是取消实时路径，而是先优化正常路径吞吐和背压，再补齐不依赖 NATS 积压队列的 DB recovery：

```text
NATS 实时消费负责低延迟；
DB reconciliation 负责最终正确性；
两条路径共用 PGResultProcessor，保证业务语义一致。
```

这样即使 NATS backlog、consumer 重启或 PostgreSQL 短时高压，已经完成的 `device_tasks` 也不会让 durable run 永久卡在 `waiting_device`。
