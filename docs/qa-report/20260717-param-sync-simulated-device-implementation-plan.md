# 参数同步堆积问题：基于当前代码的可落地实施方案

- 方案日期：2026-07-17
- 方案状态：P0 前置约束已补齐；P0-1/P0-2 已开始，当前为 expand/gate 实现阶段；本文仍不表示完整方案已经完成
- 适用版本基线：当前仓库工作树及现场版本 `v100.0.0-20260717-0708`
- 关联 RCA：[20260717-param-sync-simulated-device-backlog-rca.md](</Users/wangyong/OBJECT/Codex/xomc/docs/qa-report/20260717-param-sync-simulated-device-backlog-rca.md>)
- 目标：停止生命周期入口继续制造参数同步堆积，恢复 durable result 收口能力，并保证真实设备、ModelUpload、Manual 和历史 backlog 都有可追踪的终态

## 1. 执行摘要

本次问题不是 UI 周期同步开关开启导致。现场 `periodicSyncEnabled=false`，但历史代码仍会在 Bootstrap、DeviceOnline 和 FirmwareChanged 路径中直接调用 Path B；这些入口创建了大量 GPV task 和 durable run。当前现场的主要卡点是：`device_tasks` 已进入终态，而 `parameter_sync_task_results` 没有及时登记，导致 `processed_task_count` 不推进，run/request 不 finalize，设备级 active run 继续占用。

方案采用五个原则：

1. **先封口，再排空**：先停止 Bootstrap、DeviceOnline、FirmwareChanged 直接创建参数同步；不直接清空 active run，不直接把 run 改成 succeeded。
2. **统一 durable 链路**：所有“参数同步”语义统一进入 `parameter_sync_requests -> parameter_sync_runs -> device_tasks(source=param_sync) -> result/recovery -> finalize`；旧 Path B 只保留可复用的 GPV 路径规划算法，不再物化 `sync-gpv` 业务 task。
3. **ModelUpload 终态触发一次**：上传成功并完成模型解析，或明确判定不支持且已有可用模型/映射时，创建一次 `model_upload` durable request。上传请求本身不触发同步；临时失败不伪装成不支持。
4. **结果处理与补偿同一幂等边界**：实时 consumer 和 recovery 都以 `(run_id, task_id)` 为幂等边界；单个坏 task、DB timeout 或 NATS 重投不能阻塞其它 task。
5. **所有状态可查询、所有失败可恢复**：admission、deduplicate、DLQ、lease、replay、人工处理和 API 状态都写入 durable 状态，不依赖日志或 Redis 临时键。

## 2. 当前代码基线

当前代码仍是“durable 优先、legacy fallback 并存”的过渡实现，本节是实施前必须冻结的事实基线。

| 当前入口 | 当前代码位置 | 当前行为 | 本方案目标 |
|---|---|---|---|
| Bootstrap | `omcgo/internal/provision/engine.go` 的 `handleAutoSync` | 直接调用 `StartPathBSync(..., reason=bootstrap)` | Bootstrap 不创建参数同步；只做入网、绑定、生命周期和模型准备 |
| DeviceOnline | `engine.go` 的 `HandleDeviceOnline` | 直接调用 `StartPathBSync(..., reason=device_online)` | 只保留设备状态消费者，不触发参数同步 |
| FirmwareChanged | `engine.go` 的 `HandleFirmwareChanged` | ModelUpload 未入队或失败时直接回退 Path B；上传完成后也由文件事件触发 Path B | 不直接触发同步；允许触发 ModelUpload，ModelUpload 终态再触发 `model_upload` |
| ModelUpload 请求 | `internal/provision/model_upload.go` | 创建 discovery log 和 Upload task；无 durable intent | 增加 upload intent、任务身份、终态状态和一次性 durable trigger |
| ModelUpload 文件完成 | `engine.go` 的 `handleDataModelFileReceived` | 解析模型后按 `AutoSync.Enabled` 调 Path B | 模型解析/映射成功后提交 `model_upload` durable request |
| ModelUpload 不支持 | `ModelUploadService.HandleUploadFailed` | 只更新 discovery log；调用链不完整 | 明确 `not_supported` 终态，完成 provisioning task，再触发一次 durable request |
| Manual | `device_service.go -> ParamSyncStarter -> provider/paramsync.go` | durable 不可用时仍可调用 legacy starter | 直接 durable；不可用时明确 `unavailable/rejected`，不 fallback |
| Config Pull | `internal/config/sync_handler.go` | 存在直接 `EnqueueGPVBatches` 路径 | 改为 durable partial request；普通 RPC readback 仍保留原语义 |
| Periodic | `internal/provision/periodic_syncer.go` | 通过 `StartPathBSync` 进入 Path B | 仅 `periodicSyncEnabled=true` 时提交 durable request |
| Durable Submit | `internal/paramsync/service.go` | 具备 request/run/dispatcher，但 admission 主要是按设备 backoff | 增加全局/来源预算、ModelUpload queued 和可查询 admission |
| Result Consumer | `internal/paramsync/result_consumer.go` | 8 个 shard、每 shard 64 队列；队列满返回 error | 配置化并发；队列满产生背压，不静默丢失；超过投递次数先写 durable failure 再 Term |
| Recovery | `internal/paramsync/reconciler.go` | 按 `started_at` 顺序；单项错误中断整轮；无 claim/lease | `SKIP LOCKED` claim、退避、租约、公平轮转、单项失败隔离 |

当前数据库已有 `parameter_sync_requests`、`parameter_sync_runs`、`parameter_sync_task_results`、`parameter_sync_outbox` 和按设备 active run 唯一索引，但尚无本方案需要的 ModelUpload intent、全局 admission、recovery claim/lease 和 PARAM_SYNC durable failure 状态。

## 3. 目标行为和不可违反的约束

### 3.1 入口语义

“不触发”指事件处理函数不得直接创建对应 trigger reason 的参数同步 request；它不否定该事件启动的另一个业务流程在终态触发 `model_upload`。

| 事件/入口 | 目标行为 | 允许的参数同步 trigger |
|---|---|---|
| Bootstrap / BOOT | 入网、绑定、状态初始化、模型准备；不得直接创建同步 | 无 `bootstrap` |
| DeviceOnline | 设备在线状态及其它状态消费者；不得创建同步 | 无 `device_online` |
| FirmwareChanged | 模型/设备元数据处理；可启动 ModelUpload；不得直接同步 | 无 `firmware_changed`；后续 ModelUpload 可为 `model_upload` |
| ModelUpload `uploaded` | 模型文件接收、解析、映射提交成功后触发一次 | `model_upload` |
| ModelUpload `not_supported` | 明确能力不支持或策略禁用，且已有可用模型/映射时完成流程并触发一次 | `model_upload` |
| ModelUpload `failed` | 保留失败原因，进入可重试或人工处理；不得误触发同步 | 无 |
| Periodic | 仅周期开关开启且 admission 允许时提交 | `periodic` |
| Manual | 用户触发，独立于周期开关 | `manual` |
| Config Pull / License | 按现有业务提交 durable partial/full request | `config_pull` / `license` |
| MML / SPV / 单次故障 RPC | 保留单次 RPC 语义 | 不纳入本方案 |

`periodicSyncEnabled` 是周期同步开关，不是所有自动入口的总开关。ModelUpload 终态 trigger 不受周期开关阻断，但仍受全局 admission 和自身 budget 约束。

### 3.2 一次性和幂等规则

- ModelUpload intent 唯一键：`device_id + upload_task_id`；同一模型重复上传时追加 `model_hash/model_version` 用于审计。
- 上传完成事件必须携带 `upload_task_id`、`discovery_log_id`、`source_event_id` 和 `device_id`；不能只按设备查“最新 discovery log”。
- 参数同步 request 的 `idempotency_key` 使用稳定字符串，且同时保留 `source_event_id` 和 `origin_event_type`。
- 同一设备只能有一个非终态 `parameter_sync_run`；同一事件重投只返回原 request，不创建新 run。
- `parameter_sync_task_results` 的唯一边界为 `(run_id, task_id)`；`event_id` 用于消息审计，不作为唯一业务收口依据。

### 3.3 失败和终态规则

- Upload 的 MinIO、XML 解析、映射交集、数据库错误属于 `failed` 或可重试错误，不能改写为 `not_supported`。
- GPV task 超时进入 `expired`，必须有 durable result；不得无限追加 retry task。
- run 到达 deadline 时，未完成 task 必须按策略终态化为 expired/cancelled，并进入 result 收口。
- 只有所有 task 已终态且所有 result 已 `processed` 或 `failed`，run 才能 finalize；不得靠清空 `active_run_id` 伪造完成。

## 4. 分阶段实施方案

### Phase 0：上线前止血开关和基线

目的：在新代码完全上线前停止继续扩大积压，同时不修改历史数据。

实施内容：

1. 新增配置 `param_sync.routing_mode`，取值：
   - `legacy`：仅用于开发/回滚前兼容，不允许生产长期使用；
   - `durable_shadow`：入口只写 routing audit/metric，不写 `parameter_sync_requests`，也不创建南向 task，仅用于验证；
   - `durable`：正式目标模式；
   - `closed`：停止新的自动参数同步执行；ModelUpload 仍持久化 intent/outbox 但不 dispatch，Manual 返回明确 unavailable/rejected，继续 recovery 和已有 run 收口。
2. 在 `HandleDeviceOnline`、`HandleFirmwareChanged`、Bootstrap `handleAutoSync`、文件完成后的同步调用点增加 fail-closed gate。
3. 现场切换到 `closed` 或等价的入口封口配置，记录：active run、missing result、NATS pending、oldest stalled age、legacy task 增量、Manual API p95。
4. 不取消、不删除、不直接更新历史 run；只停止新入口。

门禁：连续 15 分钟无新的 `bootstrap/device_online/firmware_changed` 参数同步 request，且无新的 legacy `sync-gpv` 参数同步 task。shadow 模式的 routing audit 不计入 request 门禁。若门禁失败，不进入下一阶段。

### Phase 0.5：P0 实施前置约束（必须先完成）

以下 7 项是 P0-1/P0-2 的实施前置，不允许留作实现阶段的隐含约定。

1. **Admission 分桶和 reservation 回收**

   `parameter_sync_admission_state` 不再按单个 `global` 行串行化全部自动请求，改为 `(admission_class, bucket_id)` 复合主键。`bucket_id` 由 `hash(device_id) % N` 得到，global budget 通过固定数量 bucket 汇总，ModelUpload、Periodic、Manual 保留各自预算。预算 reservation 必须带 `request_id`、`lease_until` 和状态，至少支持 `reserved/released/expired`；planner、dispatcher、worker 崩溃和应用重启后由 reconciler 重算/回收，不允许永久泄漏 `reserved_runs/reserved_tasks`。

   admission 事务只锁受影响的 bucket 和 request 幂等键；容量统计使用 durable counters 或有界快照，禁止在 Submit 事务内扫描全量 backlog。验收增加：10,000 个并发 admission 不被单行锁串行化，重启后 reservation 最终可回收。

2. **ModelUpload 请求级 exactly-once 约束**

   除 `unique(device_id, upload_task_id)` 外，增加 `parameter_sync_requests` 的部分唯一索引 `unique(model_upload_intent_id) where model_upload_intent_id is not null`。outbox dispatcher 必须使用稳定的 `caller_type + idempotency_key`，并以数据库唯一约束作为最终边界；“intent 已完成但 request 状态未更新”的崩溃窗口必须可安全重放。

3. **明确 `event_id` 与业务幂等边界**

   `(run_id, task_id)` 是唯一业务结果边界；`event_id` 只做消息审计和 failure/DLQ 关联。迁移中删除现有 `uq_parameter_sync_task_results_event` 唯一索引，改为普通查询索引；所有重复消费必须通过 `(run_id, task_id)` 的条件更新判断，不能继续依赖 `ON CONFLICT(event_id)` 的副作用。

4. **ModelUpload 数据库事务边界**

   “intent、discovery log、provisioning task、trigger outbox 同一事务”必须通过 transaction-aware repository/Unit of Work 实现。`IntersectService`、discovery repository 和 provisioning task repository 要支持共享 `pgx.Tx`；MinIO、NATS、设备 RPC 仍采用 DB 状态 + outbox + 幂等重试 saga。若共享事务暂时无法实现，必须降级为明确的状态机和补偿任务，不能在文档中宣称跨 repository 原子提交。

5. **数据库写放大和在线批处理隔离**

   P0/P1 任务中显式包含：`parameter_sync_task_results`/staging 批量 upsert、`device_parameters` 差异 upsert、result/recovery 与 device online/batch processor 的独立 worker/DB semaphore、单事务行数上限和 statement timeout。仅增加 channel、连接池或 NATS 并发不得作为容量修复。10,000 台 × 32 task 压测必须同时记录 PG CPU、LockManager、连接池等待和 terminal-result backlog。

6. **routing_mode 与 ModelUpload/Manual 语义**

   `durable_shadow` 只写审计，不产生参数同步 request；`closed` 不丢失 ModelUpload 终态，只把 intent/outbox 留在 queued，Manual 按 API 契约返回 unavailable/rejected。`periodicSyncEnabled=false` 仍不阻断正常 durable 模式下的 ModelUpload，但全局 `closed` 是更高优先级的紧急执行熔断。

7. **Manual 离线 API 契约**

   新增 `param_sync.manual_offline_mode=queue|reject`，默认 `reject`：设备离线时返回 `503 DEVICE_OFFLINE`；显式 `queue` 时创建可查询的 queued request，不创建南向 task，待设备可用后由 dispatcher 继续。两种模式均需有 API、前端、指标和自动化测试，且不允许应用层预检后静默丢弃请求。

前置门禁：以上 7 项完成 schema/API/状态机设计评审，并各自拥有迁移、单测、集成测试和验收 SQL 后，才开始 P0-1/P0-2。

### Phase 1：数据库扩展和兼容代码

新增主库迁移，建议文件名：`omcgo/migrations/000026_param_sync_recovery_and_admission.sql`。当前主库 schema 已是 consolidated baseline，新增迁移必须从 `000026` 起步；既有现场库还需按实际 goose 版本另做升级校验，不能直接重跑 `000001`。

#### 4.1 ModelUpload intent

新增 `model_upload_intents`：

```text
id uuid primary key
device_id uuid not null
upload_task_id uuid not null
discovery_log_id uuid
source_event_id varchar(128) not null
model_version varchar(128)
model_hash varchar(128)
status requested/uploaded/not_supported/failed/sync_queued/sync_submitted/manual_review
failure_code varchar(64)
failure_message text
attempts int not null default 0
next_attempt_at timestamptz not null default now()
last_error text
created_at/updated_at timestamptz not null
```

约束：`unique(device_id, upload_task_id)`；`unique(source_event_id)`。对 `status,next_attempt_at` 建 dispatcher 索引。参数同步 request 另加 `unique(model_upload_intent_id) where model_upload_intent_id is not null`。

#### 4.2 Request 扩展

扩展 `parameter_sync_requests`：

```text
source_event_id
origin_event_type
model_upload_intent_id
model_upload_status
admission_class
admission_reason
admission_snapshot jsonb
deduplicated_to_request_id
```

不新增 `admission_waiting` 状态，统一使用现有 `queued`，通过 `admission_class/admission_reason` 区分等待原因。新增结果码 `AUTOMATIC_BACKPRESSURE`，保留旧 `AUTOMATIC_BACKOFF` 兼容读取。

#### 4.3 Admission 状态

新增 `parameter_sync_admission_state`，按 `admission_class + bucket_id` 分片保存；不得用单个 global 行串行化全部设备上线请求：

```text
admission_class varchar(32) not null
bucket_id smallint not null
active_run_limit
active_task_limit
missing_result_limit
create_rate_per_minute
reserved_runs
reserved_tasks
version
updated_at
primary key(admission_class, bucket_id)
```

admission 事务只做短操作：幂等查询、request 创建、预算预留、同设备 active run 检查。不能在事务内执行模型规划、GPV task 批量创建或长时间 result 写入。

#### 4.4 Recovery claim

新增 `parameter_sync_recovery_state`，由它独占 claim/lease：

```text
run_id uuid not null
task_id uuid not null
status pending/processing/processed/failed/manual_review
attempts int not null default 0
next_retry_at timestamptz not null default now()
lease_token uuid
lease_until timestamptz
last_error text
claimed_at/processed_at timestamptz
created_at/updated_at timestamptz not null
primary key(run_id, task_id)
```

`parameter_sync_task_results.status` 保持 receipt 语义 `received/processed/failed`，不再新增 `processing`；claim 状态只在 recovery state 中维护，避免双状态源。

#### 4.5 PARAM_SYNC failure/DLQ

新增 `parameter_sync_event_failures`：

```text
id uuid primary key
subject
event_id
device_id/device_sn
request_id/run_id/task_id
raw_payload jsonb
delivery_count
status pending/replayed/recovered/manual_review
last_error
next_retry_at
replayed_at/recovered_at
created_at/updated_at
unique(subject,event_id)
```

若复用现有 `dead_letters`，必须增加同等关联字段和索引，不能依赖 payload 搜索恢复。

#### 4.6 迁移兼容要求

- 先扩表、索引和可空字段，再部署读写兼容代码；不在同一次发布中删除旧字段或收紧旧枚举。
- 旧 `received` 结果可被新 recovery 读取；旧 request 没有来源字段时使用 `legacy_unknown`。
- 迁移完成后先回填 recovery state 的缺失 task，再开启 recovery worker。
- 任何旧实例都不得因为新增字段为空而创建新 legacy 参数同步 task。
- 删除 `uq_parameter_sync_task_results_event`，改建 `idx_parameter_sync_task_results_event` 普通索引；`(run_id, task_id)` 继续作为主键和唯一业务边界。
- 为 `model_upload_intent_id` 建部分唯一索引，reservation 建 request 关联和 lease 索引；所有新索引须在现场库执行 `EXPLAIN` 验证。

### Phase 2：统一参数同步入口

#### 4.7 Durable Submitter 和 admission

修改 `internal/paramsync/service.go`、`pg_repository.go`：

1. `TriggerReason.Automatic()` 纳入 `model_upload`；Bootstrap 不再作为可提交入口。
2. `Submit` 先执行幂等和 admission 事务。
3. ModelUpload admission 不通过时保留 `queued` request，由 dispatcher 重试；Periodic 可 `queued` 或 `rejected`，但必须有原因记录；Manual 使用独立预算。
4. active run 冲突时写 `deduplicated_to_request_id` 和 `active_run_id`，不可只返回日志。
5. planner/dispatcher 失败时释放 admission reservation，并将 request/run 转为可重试或明确失败。
6. admission 的主要判断使用 durable backlog、missing result 数、oldest stalled age；PG CPU/LockManager 只作为带滞回的辅助信号。

#### 4.8 Bootstrap 和 DeviceOnline

修改 `internal/provision/engine.go`：

- 删除 Bootstrap `handleAutoSync` 的 Path B 调用；Bootstrap 任务在模型准备完成后进入既定生命周期终态。
- `HandleDeviceOnline` 删除 `StartPathBSync` 调用，只保留设备查询、绑定和在线状态相关消费者。
- 保留 online 节流只用于在线状态业务，不再作为参数同步去重机制。

#### 4.9 FirmwareChanged

- 删除直接 `StartPathBSync(... firmware_changed)` fallback。
- 不再写 `provision:syncreason` 这种临时同步原因。
- 如果业务要求重新获取模型，则只调用 ModelUpload intent；ModelUpload 终态通过 `model_upload` request 触发一次同步。
- ModelUpload 未配置时按 `not_supported` 还是 `failed` 处理必须由产品配置明确；若无可用映射，不得创建空同步。

#### 4.10 ModelUpload

修改 `internal/provision/model_upload.go`、`engine.go`：

1. `RequestModelUpload` 创建 `model_upload_intents`，生成稳定 `upload_task_id`，将该 ID 写入 Upload task 的 source/params。
2. `datamodel.file.received` payload 必须携带 `upload_task_id` 和事件 ID；处理时按 intent/discovery log 精确加载，不按设备“最新记录”猜测。
3. 文件下载、解析、交集写入成功后，通过共享 `pgx.Tx` 的 Unit of Work 在同一数据库事务中完成：
   - intent=`uploaded`；
   - discovery log completed；
   - provisioning task 可进入完成/下一状态；
   - 写入参数同步 trigger outbox。
4. 产品策略禁用或设备明确 SOAP Fault 时进入 `not_supported`，但只有已有可用模型/映射时才允许进入 sync queued。
5. 临时失败进入 `failed` + `next_attempt_at`，不触发同步；达到重试上限进入 `manual_review`。
6. outbox dispatcher 负责提交 `model_upload` request；提交成功后 intent=`sync_submitted`。重复事件只能命中唯一键。
7. `HandleUploadFailed` 必须接入真实事件订阅，并完成 provisioning task、discovery log、intent 的一致状态更新。

外部 MinIO、NATS 和设备 RPC 不能与 PostgreSQL 共享一个真正的 ACID 事务，因此采用“DB 状态 + outbox + 幂等重试”的 saga，不承诺跨系统原子提交。

#### 4.11 Manual、Config Pull、License、Periodic

- Manual、Config Pull、License 删除 durable 不可用时的 legacy fallback；返回 `unavailable/rejected` 和 request ID（若已创建）。
- Manual 离线按 `manual_offline_mode` 执行：`queue` 不创建南向 task，`reject` 返回 `DEVICE_OFFLINE`；两者均保留 durable 审计。
- Config Pull 将路径规划交给 durable Planner，创建 `scope=partial` 的 `config_pull` request。
- PeriodicSyncer 只负责扫描、周期开关和 scan window 幂等键，实际执行调用 durable Submitter。
- `periodicSyncEnabled=false` 时只跳过新的周期扫描；不影响 Manual、ModelUpload intent dispatcher 和已有 run recovery。

### Phase 3：抽取 GPV Planner，删除 legacy task 物化

修改 `internal/provision/sync.go`、`sync_pathb.go`、`internal/paramsync/planner.go` 及相关调用点：

1. 抽取 `PathBStorablePrefixes`、标准路径转换、批次划分和 payload 预算为中性 Planner 包。
2. durable Dispatcher 只创建 `device_tasks(source=param_sync)`，并通过 `parameter_sync_outbox` 发布结果事件。
3. `EnqueueGPVBatches` 不再作为参数同步业务入口；可以保留为 Planner 内部算法或普通单次 RPC 工具，但不得创建 legacy `sync-gpv` 参数同步 task。
4. 全仓库静态检查必须确认参数同步语义不再调用 `StartPathBSync` 或 legacy `EnqueueGPVBatches`。
5. MML、SPV 和其它明确单次 RPC 操作继续使用普通 task，但必须有独立 source/reason，不能被误计入参数同步 backlog。

### Phase 4：Result Consumer 和 Recovery

#### 4.12 Result Consumer

修改 `internal/paramsync/result_consumer.go` 和 EventBus 配置：

- shard 数、queue depth、pull concurrency、ack wait、max ack pending 全部配置化。
- 同一设备保持顺序，不同设备按 shard 并行。
- 队列满时优先阻塞 pull/backpressure，不使用无条件丢弃；若底层接口不能阻塞，则写入 `parameter_sync_event_failures` 后返回可重投错误。
- 只有 PGResultProcessor 事务提交成功后 ACK。
- payload 解析失败、达到最大投递次数时，先写 failure/DLQ，再 Term；写 failure 失败只能 NAK，不能静默 Term。
- 指标增加：active workers、queue depth、queue full、process latency、processed、duplicate、error、NAK、Term、DLQ、replay、recovered。

#### 4.13 PGResultProcessor

修改 `internal/paramsync/result_processor.go`：

- 对 `(run_id, task_id)` 使用条件更新保证重复事件只推进一次。
- `received` 不视为 processed；重投、recovery 和 worker 崩溃都可以安全重跑。
- 处理 task result、staging/official values、run progress 和 finalize 时保持同一业务幂等边界；长事务按批次拆分。
- Projection 与 run finalize 解耦，但只有 durable result/run 状态一致后才清理 active run。
- 对 expired、cancelled、failed task 明确写入失败 result，不要求设备再次响应。
- 结果处理按批量大小和单事务耗时上限拆分；staging/official values 使用批量写入，`device_parameters` 使用差异 upsert。
- result/recovery 与设备在线、batch processor 使用独立 worker pool、DB semaphore 和 statement timeout，避免仅靠连接池扩容掩盖 PostgreSQL 锁竞争。

#### 4.14 Recovery

修改 `internal/paramsync/reconciler.go` 和维护调度：

1. 查询候选时按 `trigger_reason` 轮转，不再只按 `started_at` 抢占最早的一批。
2. 用 `FOR UPDATE SKIP LOCKED` 短事务 claim recovery state，写入 lease 后立即提交；完整 result 处理不能持有 claim 锁。
3. 完成时校验 lease token；lease 丢失只做幂等查询，不覆盖新 owner。
4. 单 task/run 失败记录 `last_error`、attempts、next_retry_at，跳过当前项继续后续项；维护轮次最后汇总错误。
5. recovery 查询同时覆盖：缺失 result、超时 received、失效 processing、run deadline 超时和 cancelling 收口。
6. `CleanStaging`、`CollectMetrics`、`ReconcileTerminalBindings`、`RecoverMissingResults` 使用独立 context timeout；一个动作失败不得阻断其它动作。
7. recovery 与实时 consumer 共用 `(run_id, task_id)` 幂等边界，不能一个走 recovery state、另一个绕过 state 直接写结果。

### Phase 5：状态接口、前端和可观测性

1. `parameter_sync_runs` 作为 active 状态唯一来源；修复 active 接口与旧 `/parameters/sync-status` 的 idle/running 冲突。
2. 旧接口如需兼容，必须明确映射：
   - durable active run 存在 -> `running`；
   - durable run stalled -> `error/stalled`；
   - 无 active run -> `idle`。
3. 前端展示 request ID、run ID、trigger reason、queued/rejected reason、stalled age 和人工处理状态。
4. 指标至少包含：
   - terminal task 数、processed result 数、missing result 数；
   - oldest stalled age、per-trigger recovery lag、healthy run wait time；
   - admission accepted/queued/rejected/deduplicated；
   - result processed/error/duplicate/redelivery；
   - recovery attempts/claim conflict/lease takeover/manual review；
   - NATS pending/NAK/Term/DLQ/replay/recovered；
   - legacy task created count，目标为 0。

## 5. 存量 backlog 处理

禁止直接 SQL 清空 `active_run_id`、删除 run 或强制改 succeeded。

处理顺序：

1. 封口后冻结新 Bootstrap/DeviceOnline/FirmwareChanged 参数同步入口。
2. 先回补已经终态但缺 result 的 task，使用 `device_tasks` 的真实 status/result 生成 durable recovery payload。
3. 对 `terminal < expected` 的 run，按 run deadline 终态化未完成 task；设备无响应使用 expired，管理员取消使用 cancelled。
4. 对反复失败的 task/run 进入 `manual_review`，保留 run/request/task/error 关联，不无限重试。
5. Recovery 按 manual、历史 run、来源预算轮转；真实设备和 Manual 保留独立预算。
6. 只有当 missing result backlog 连续下降、oldest stalled age 不增长、没有新增 legacy task 且 DLQ 可回放时，才逐步开启 Periodic。

## 6. 发布和回滚顺序

### 6.1 发布顺序

```text
R0 入口封口开关和监控基线
R1 数据库 expand migration + 双读兼容
R2 recovery claim/lease + durable failure/DLQ
R3 result consumer 背压与单项失败隔离
R4 durable admission + Manual/Config Pull/License/Periodic 收敛
R5 ModelUpload intent、终态和 model_upload request
R6 停止所有新 legacy 参数同步 task
R7 排空历史 backlog
R8 删除 legacy fallback 和业务入口
R9 逐步开启 Periodic
```

R1-R3 发布期间不得依赖新字段非空；R6 之前保留旧符号但禁止新业务调用；确认全仓库无 legacy 参数同步 task 后，才删除旧实现。

### 6.2 回滚规则

- 任何阶段回滚只允许切换到 `closed`：停止新的自动 request，继续 recovery 已有 durable run。
- 不重新打开 Bootstrap、DeviceOnline、FirmwareChanged 直接同步。
- 不重新打开 legacy Path B 作为 durable 不可用时的 fallback。
- 若 ModelUpload intent 已存在，继续由 intent dispatcher 重试或进入 manual review，不删除 intent。
- 删除旧表/旧符号必须晚于至少一个完整发布周期，并确认无旧实例、无旧消费者、无新增 legacy task。

## 7. 测试和验收门禁

### 7.1 单元测试

- 入口矩阵：Bootstrap/DeviceOnline/FirmwareChanged 不直接提交；ModelUpload 终态提交一次。
- ModelUpload `uploaded/not_supported/failed` 状态转换和重复事件。
- admission 的 queued/rejected/deduplicated、预算释放和幂等。
- 两个 recovery worker 竞争同一 task 时只有一个 lease owner。
- lease 过期接管、worker 崩溃、单项 DB timeout 不阻塞其它 task。
- received/processing/processed 重复消费和 result 幂等。
- Manual、Config Pull、License durable 不可用时不调用 legacy。

### 7.2 集成测试

- PostgreSQL migration 在旧数据上可执行，旧 request/result 可双读。
- NATS ACK、NAK、Term、DLQ、Replay 全链路。
- `device_tasks` 先终态、result 事件后到；result 先到、task projection 后到；两种顺序都能收口。
- 10,000 台模拟设备、每台 32 task 的 result backlog 压测。
- PostgreSQL timeout、LockManager 等待、网络断开、MinIO 暂时失败和重复事件注入。

### 7.3 首轮上线门禁

以下是建议初始门禁，正式压测前由 QA、后端和运维共同确认具体值：

| 指标 | 门禁 |
|---|---|
| 新 legacy 参数同步 task | 连续 15 分钟为 0 |
| 新 `bootstrap/device_online/firmware_changed` request | 连续 15 分钟为 0 |
| PARAM_SYNC dropped | 0；每个 Term 必须有 durable failure 记录 |
| missing result backlog | 连续 30 分钟下降，不能只靠 NATS/DB 存量增长 |
| oldest stalled age | 连续 30 分钟不增长 |
| healthy run recovery lag | p95 不超过 5 分钟，具体值需压测前确认 |
| Manual request | admission response p95 不超过 2 秒；完成时延单独按设备能力统计 |
| recovery 单项失败 | 不影响同批后续 healthy run |
| ModelUpload trigger | 每个 uploaded/not_supported intent 最多一个 request |
| API 状态 | active、sync-status、request history 对同一 run 结果一致 |

任何一项不满足，保持 `closed` 或只恢复 Manual 独立配额，不开启 Periodic 和批量自动入口。

## 8. 实施任务拆分

建议按以下可独立验收的任务排期：

1. **P0-0 前置约束落地**：7 项设计门禁、状态机、唯一性、事务边界和验收 SQL。
2. **P0-1 入口封口和 routing mode**：Engine 四个入口、配置校验、运行指标。
3. **P0-2 数据库 expand migration**：intent、admission bucket/reservation、recovery、failure/DLQ 表和索引。
3. **P0-3 Recovery lease 和失败隔离**：claim、退避、轮转、独立 deadline。
4. **P0-4 Result consumer 背压和 DLQ**：队列、ACK/NAK/Term、durable failure。
5. **P0-5 Durable admission**：预算、幂等、queued、deduplicate、token 释放。
6. **P0-6 入口统一**：Manual、Config Pull、License、Periodic 全部进入 durable。
7. **P0-7 ModelUpload intent 和终态闭环**：上传身份、能力判定、outbox、一次性 trigger。
8. **P1-1 Planner 抽取和 legacy task 停止**：保留算法，删除业务物化。
9. **P1-2 API/前端状态统一**：唯一状态源、错误码和人工处理展示。
10. **P1-3 压测、故障注入和现场排空**：按门禁执行，形成发布报告。

每个任务必须包含：代码 owner、数据库变更、自动化测试、指标、灰度开关、回滚动作和验收 SQL/查询。P0-1/P0-2 开始前必须先完成 P0-0。

## 9. 明确不做的事情

- 不按 SN 前缀、设备组或产品永久跳过压测设备。
- 不直接删除历史 run/task，不直接清空 active run。
- 不把 ModelUpload 临时失败伪装成 not_supported。
- 不通过增大 channel 或连接池掩盖 DB 锁竞争。
- 不把 MML、SPV 等单次 RPC readback 强行改成参数同步。
- 不以重新开启 legacy Path B 作为 durable 链路的回滚方案。

最终完成标准是：所有参数同步语义可从 request 到 run、task、result、recovery、finalize 全链路追踪；新生命周期入口不再制造参数同步堆积；历史 task 能安全收口或进入人工处理；真实设备 Manual 具备独立可用的执行预算；ModelUpload 终态一次且仅一次触发 durable 参数同步。
