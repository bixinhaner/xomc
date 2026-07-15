# 参数同步架构整改方案

- 日期：2026-07-15
- 关联评审：`REVIEW_0303179_wangyong_sync-gpv.md`
- 适用范围：Path B 参数同步、ACS GPV、设备任务、provisioning、参数树状态与结果汇总
- 方案状态：待评审

## 1. 结论

参数同步建议采用“同步运行单（SyncRun）+ 单一结果消费者 + 全量快照收口”的架构，并分阶段替换当前依赖 `device_tasks` 状态、command key、Redis 临时 key 推断同步状态的实现。

整改完成后应满足以下核心约束：

1. 一轮参数同步具有稳定且唯一的运行实体。
2. task 执行终态与参数结果处理完成分别记录，不能相互替代。
3. 同一个 GPV 结果只有一个消费者负责写 `device_parameters`。
4. 全量同步和局部读取使用结构化字段区分，不能由 command key 推断。
5. 全量同步只有在所有批次结果处理成功后才能提交、刷新投影和标记成功。
6. 所有结果事件可以安全重放，重复投递不会重复计数、重复写入或重复收尾。

## 2. 当前主要问题

当前实现存在以下高优先级问题：

- Path B provisioning 缺少稳定的成功收口路径，正常完成、防重跳过、无可同步路径都可能使 provisioning 停留在 `syncing`。
- `ProvisioningEngine` 和 `RPCResponseSubscriber` 同时消费 GPV 响应并写同一张参数表，过滤、类型和 writable 语义互相覆盖。
- provision pull consumer 在结果进入内存队列后即 ACK，数据库写入失败或进程崩溃会永久丢失结果。
- ACS 先将 task 标记终态，再发布结果事件；当前却通过 task 是否终态判断结果是否已全部处理，可能过早或重复 finalize。
- 手工 scoped sync、Config Pull、License 读取等局部操作使用 `sync-gpv-*` command key，被误判为全量同步。
- 非全量 GPV 也可能更新 `last_param_sync_at`，导致周期全量同步长期不执行。
- 多批同步中失败事件和成功 finalize 分别覆盖设备级状态，最终状态取决于事件顺序。
- 批量入队逐条执行，第 N 条失败时会留下前 N-1 条可执行的半轮任务。
- GPV Fault 汇总只把 `param_faults` 中的一条提示计为失败，未执行的其他请求路径可能被统计为成功。
- 大对象展开使用完整 5 MiB 阈值，没有给 NATS event envelope 和字段波动预留空间。

## 3. 目标整体架构

新增独立领域模块 `internal/paramsync`，统一承接所有参数同步入口。

```text
触发适配层
provision / device API / periodic / config pull / license
                         │
                         ▼
              创建独立 SyncRequest
              ├─ 分配设备单调序号
              ├─ 幂等接收调用请求
              └─ 排队或绑定兼容 SyncRun
                         │
                         ▼
               ParamSyncScheduler
               ├─ 同设备串行调度
               ├─ 创建或加入 SyncRun
               ├─ 生成同步计划
               └─ 原子创建 device_tasks
                         │
                         ▼
                  ACS 协议执行层
               PopTask → GPV → terminal
                         │
                         ▼
             param_sync.task_result 事件
                         │
                         ▼
          ParamSyncResultConsumer（唯一写入者）
               ├─ 翻译、过滤、校验
               ├─ 幂等保存批次结果
               ├─ 更新 processed_tasks
               └─ 满足条件后 Finalize
                         │
                         ▼
                 ParamSyncFinalizer
               ├─ 全量快照提交
               ├─ diff / reconcile
               ├─ device_info 投影
               ├─ last_param_sync_at
               └─ provisioning 状态联动
```

### 3.1 模块职责

#### ParamSyncService

- 接收所有全量、局部参数同步请求。
- 为每次调用创建独立 SyncRequest，并支持 idempotency key。
- 将请求排队或绑定到兼容的 SyncRun。
- 解析产品和 MappingSet，生成 GPV 批次计划。
- 原子创建 task 计划并提交到任务执行层。
- 返回 request ID、run ID、设备序号和明确的启动结果，不再使用 `(bool, int, error)` 表达多种状态。

#### ParamSyncScheduler

- 以 PostgreSQL 为顺序权威，为同设备请求分配单调递增序号。
- 同设备严格串行，不同设备允许并行。
- 根据 full/partial 兼容规则合并请求或创建新 run。
- 只在上一 run 的 Finalizer 事务提交后启动同设备下一 run。

#### ACS

- 只负责 CWMP 会话、RPC 关联、SOAP 解析、task 状态转换和结果发布。
- 同一设备 run 的全部 GPV 批次预入队，在一个 CWMP 会话内按请求/响应顺序连续下发。
- GPV 9005 记录为设备不支持，不创建重试 task，并继续下一批。
- 不负责参数库业务翻译、全量/局部判断或 run 收尾。

#### ParamSyncResultConsumer

- 参数同步结果的唯一写入者。
- 在数据库事务中完成幂等判定、翻译过滤、结果暂存和 run 计数更新。
- 只有数据库事务成功后才允许 ACK JetStream 消息。

#### ParamSyncFinalizer

- 判断 run 是否满足唯一收尾条件。
- 全量同步使用完整运行快照提交参数、执行 diff/reconcile 和刷新投影。
- 局部读取只更新请求参数，不修改全量同步时间或执行删除对账。

## 4. 数据模型

### 4.1 parameter_sync_runs

```sql
CREATE TABLE parameter_sync_runs (
    id                    uuid PRIMARY KEY,
    device_id             uuid NOT NULL,
    device_sn             varchar NOT NULL,

    trigger_reason        varchar NOT NULL,
    sync_scope            varchar NOT NULL,
    status                varchar NOT NULL,

    requested_paths       jsonb NOT NULL DEFAULT '[]',
    expected_task_count   integer NOT NULL DEFAULT 0,
    terminal_task_count   integer NOT NULL DEFAULT 0,
    processed_task_count  integer NOT NULL DEFAULT 0,
    failed_task_count     integer NOT NULL DEFAULT 0,

    error_message         text,
    started_at            timestamptz NOT NULL,
    completed_at          timestamptz,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    version               bigint NOT NULL DEFAULT 0
);
```

字段约束：

- `sync_scope`：`full` / `partial`。
- `status`：`planning` / `enqueuing` / `running` / `processing` / `succeeded` / `failed`。
- `trigger_reason`：`bootstrap` / `device_online` / `periodic` / `firmware_changed` / `manual` / `config_pull` / `license` 等。

同设备只允许一个活动同步：

```sql
CREATE UNIQUE INDEX uq_parameter_sync_active_device
ON parameter_sync_runs(device_id)
WHERE status IN ('planning', 'enqueuing', 'running', 'processing');
```

### 4.2 device_tasks 与 run 的关系

继续复用 `device_tasks.source_id`，但对参数同步建立统一契约：

```text
source    = param_sync
source_id = parameter_sync_runs.id
```

不再让 `source_id` 混合表示 provisioning task、设备 ID、手工来源等不同概念。

### 4.3 parameter_sync_requests

每次 API、周期任务或 provisioning 调用先创建一条独立请求。调用者跟踪 request，而不是直接跟踪共享的 run 或 task：

```sql
CREATE TABLE parameter_sync_requests (
    id                uuid PRIMARY KEY,
    device_id         uuid NOT NULL,
    caller_type       varchar NOT NULL,
    caller_id         varchar,
    idempotency_key   varchar,

    sync_scope        varchar NOT NULL,
    requested_paths   jsonb NOT NULL DEFAULT '[]',
    status            varchar NOT NULL,

    run_id            uuid,
    device_sequence   bigint NOT NULL,
    queued_at         timestamptz NOT NULL DEFAULT now(),
    started_at        timestamptz,
    completed_at      timestamptz,
    error_message     text,
    result_summary    jsonb
);
```

建议对 `(caller_type, idempotency_key)` 建唯一约束；调用方超时重试时返回原 request，避免一次业务操作生成多个同步请求。

### 4.4 parameter_sync_request_bindings

兼容请求可以共用一个 run，但每个 request 的覆盖范围和结果仍独立保存：

```sql
CREATE TABLE parameter_sync_request_bindings (
    request_id       uuid PRIMARY KEY,
    run_id           uuid NOT NULL,
    coverage_type    varchar NOT NULL,
    covered_paths    jsonb NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now()
);
```

- Full 请求使用 `coverage_type=full`。
- 加入 Full run 的 Partial 请求使用 `coverage_type=path_subset`。
- 合并的 Partial 请求在 `covered_paths` 中保留各自请求路径。
- provisioning task ID、手工调用者、周期任务等身份保存在 request 的 `caller_type/caller_id`，run 完成后按 request 独立回调。

### 4.5 parameter_sync_device_sequences

用于给同设备请求分配数据库权威的单调序号：

```sql
CREATE TABLE parameter_sync_device_sequences (
    device_id       uuid PRIMARY KEY,
    next_sequence   bigint NOT NULL
);
```

序号分配必须在短事务内使用 `SELECT ... FOR UPDATE`，不能依赖 Redis score、NATS 到达顺序或应用实例本地时钟。

### 4.6 parameter_sync_task_results

记录每个 task 的结果处理状态，并作为事件幂等键：

```sql
CREATE TABLE parameter_sync_task_results (
    run_id               uuid NOT NULL,
    task_id              uuid NOT NULL,
    success              boolean NOT NULL,
    requested_paths      jsonb NOT NULL,
    failed_paths         jsonb NOT NULL DEFAULT '[]',
    parameter_count      integer NOT NULL DEFAULT 0,
    processed_at         timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, task_id)
);
```

### 4.7 parameter_sync_staging_values

全量同步先写暂存区，完成后一次性提交正式参数快照：

```sql
CREATE TABLE parameter_sync_staging_values (
    run_id           uuid NOT NULL,
    device_id        uuid NOT NULL,
    parameter_path   varchar NOT NULL,
    parameter_value  text,
    parameter_type   varchar,
    writable         boolean NOT NULL,
    private_path     varchar,
    PRIMARY KEY (run_id, parameter_path)
);
```

这样可以避免暴露半轮结果，并保证 diff/reconcile 基于整轮结果而不是最后一个批次。

## 5. 启动接口与防重策略

### 5.1 明确的启动返回值

```go
type StartDisposition string

const (
    SyncStarted     StartDisposition = "started"
    SyncJoined      StartDisposition = "joined"
    SyncAlreadyBusy StartDisposition = "already_busy"
    SyncUnavailable StartDisposition = "unavailable"
)

type StartResult struct {
    Disposition StartDisposition
    RunID       uuid.UUID
    TaskCount   int
}
```

### 5.2 各入口处理策略

| 入口 | 已有全量同步运行中 | Path B 不可用 |
|---|---|---|
| Bootstrap provisioning | 加入现有 run，保存 binding | 进入 ModelUpload 或明确失败 |
| device.online | 跳过并记录 existing run ID | 跳过 |
| PeriodicSyncer | 跳过 | 跳过 |
| firmware.changed | 加入兼容全量 run，或模型更新后新建 | 使用 default mapping 或失败 |
| 手工全量同步 | 返回 409 和当前 run ID | 返回 503 |
| 手工局部同步 | 默认返回 409 | 返回 503 |
| Config Pull / License | 创建 `partial` run | 返回明确错误 |

Provisioning 必须先创建或加入 run，再进入 `syncing`：

```text
创建/加入 run 成功
    → provisioning syncing
    → run succeeded
    → provisioning completed

创建 run 失败
    → provisioning failed

加入已有 run
    → 保存 binding
    → 由同一 run 的终态回调完成 provisioning
```

## 6. 任务创建可靠性

新增批量入口：

```go
CreateParamSyncTasks(
    ctx context.Context,
    run *ParameterSyncRun,
    requests []task.CreateTaskRequest,
) error
```

建议实现流程：

1. 一个 PostgreSQL 事务内创建 run、全部 `device_tasks` 和全部 enqueue outbox 记录。
2. 提交事务后由 outbox dispatcher 将同一 run 的全部 task 按 `command_index/priority` 推入设备 Redis 队列。
3. 同一轮 dispatcher 先无唤醒地完成 task 入队，再按设备去重触发一次逻辑 Connection Request；不能按批次逐个唤醒。普通单任务仍由 TaskService 即时唤醒。
4. 手动同步入口不得额外直接唤醒设备，durable Outbox 是该流程唯一的唤醒所有者。
3. Redis 推送失败持续重试。
4. Redis 只承担执行加速，PG task 始终为权威状态。
5. 全部任务可执行后将 run 转为 `running`。

设备队列中的 task 仍严格串行，不能并发发送 RPC。预入队的目的，是让 CPE 在当前 CWMP 会话响应上一批后立即取得下一批；一次逻辑唤醒只负责建立会话，不能因有 N 个批次就发送 N 次 Connection Request。非 9005 真实失败时，PG 将剩余 pending task 转为 cancelled，取消 outbox 负责从 Redis 幂等驱逐。

滚动升级兼容：旧版本已创建的 active run 可能只有首批 enqueue outbox，ResultConsumer 暂时保留“缺失时补下一批”的幂等语句；新 run 的全部 outbox 已存在，因此不会触发逐批释放。旧 run 排空后删除该兼容分支。

第一阶段可以复用现有 `RestorePendingQueues` 兜底，最终应使用 outbox 消除半轮任务。

## 7. 结果事件与 ACK 语义

### 7.1 专用轻量事件

```go
type ParamSyncTaskResultEvent struct {
    EventID   string
    RunID     uuid.UUID
    TaskID    string
    DeviceID  uuid.UUID
    DeviceSN  string
    Success   bool
    ResultRef string
}
```

原始 SOAP 和大参数结果保存到数据库分块表或对象存储，NATS 只传结果引用，不再发布包含巨大 `raw_response` 的完整 task。

### 7.2 ACK 时机

```text
PullSubscribe 收到事件
    → 投入按 device 分片 worker
    → 等待 worker 返回
        → 数据事务成功：handler return nil → ACK
        → 数据库失败：handler return err → NAK / retry
```

禁止结果只进入内存 channel 后立即 ACK。

### 7.3 单条结果事务

每条 task result 在同一 PG 事务内：

1. 插入 `parameter_sync_task_results`，主键冲突表示已处理，直接幂等返回。
2. 解析结果并按 MappingSet 翻译、过滤和校验。
3. full run 写 staging；partial run 写正式参数。
4. 更新 `processed_task_count` 和 `failed_task_count`。
5. 判断是否满足 finalize 条件。
6. 提交事务后 ACK。

## 8. 全量同步收口

只有同时满足以下条件才能收口：

```text
run.status in (running, processing)
AND terminal_task_count = expected_task_count
AND processed_task_count = expected_task_count
AND failed_task_count = 0
```

通过 CAS 确保多实例只执行一次 Finalizer：

```sql
UPDATE parameter_sync_runs
SET status = 'processing', version = version + 1
WHERE id = $1
  AND status = 'running'
  AND terminal_task_count = expected_task_count
  AND processed_task_count = expected_task_count
  AND failed_task_count = 0
RETURNING id;
```

### 8.1 Full run 收尾

同一事务内执行：

1. staging values 批量 upsert 到 `device_parameters`。
2. 基于整轮 seen paths 计算 missing diff。
3. 只对本轮完整且成功覆盖的对象执行 reconcile。
4. 刷新 `device_info`。
5. 执行设备名称同步 hook。
6. 更新 `last_param_sync_at`。
7. 清空 `last_param_sync_failed_at/error`。
8. run 转为 `succeeded`。
9. 完成所有 provisioning binding。
10. 清理 staging。

### 8.2 Partial run 收尾

局部读取只允许：

- 保存本次请求参数；
- run 转为 `succeeded`；
- 返回调用方结果。

明确禁止：

- 更新 `last_param_sync_at`；
- 全库 missing diff；
- reconcile 删除；
- 设备名称全量同步 hook；
- 将结果展示为“上次全量同步”。

## 9. 失败与 Fault 统计

GPV command 整体失败时，该 task 的全部 requested paths 都应计为失败；`param_faults` 只补充具体路径的详细错误。

```text
task success:
    requested paths 全部成功

task failed / expired / cancelled:
    requested paths 全部失败
    param_faults 覆盖对应路径的 fault_code / fault_text
    其余路径使用 command 级 error_code / error_message
```

路径统计使用 `COUNT(DISTINCT path)`，重试链按 root task/path 去重。

任一最终不可恢复 task 失败时：

1. run 标为 `failed`。
2. 不提交 full staging。
3. 不更新 `last_param_sync_at`。
4. 更新 `last_param_sync_failed_at/error`。
5. 所有 provisioning binding 转为 failed。
6. 批量取消该 run 剩余 pending task。

## 10. 现有模块调整

### 10.1 ProvisioningEngine

移除：

- 自行消费 GPV 并直接写参数；
- `shouldFinalizePathBSync`；
- Redis pending batches；
- 根据 task PG 状态猜测最后一批；
- 从 command key 判断 full sync。

保留：

- provisioning 入口适配；
- SyncRun binding；
- run 终态后的 provisioning 状态转换。

### 10.2 RPCResponseSubscriber

- 参数同步 task 不再落库，交给 `ParamSyncResultConsumer`。
- 普通临时 GPV 仍由该 subscriber 处理。
- 不再直接消费同步 task.failed 覆盖设备同步状态。
- 改为明确 durable consumer 或只消费新事件，禁止每次启动 `DeliverAll` 重放历史状态。

### 10.3 ACS

- 保留协议和 task 执行职责。
- 发布专用轻量 result event。
- GPV 9005 不创建 recovery/retry task，不增加 `expected_task_count`，继续消费已预入队的下一批。
- 每个 completed/failed/expired/cancelled task 都必须产生可处理的 canonical result event。

## 11. NATS 大对象策略

短期立即将展开阈值改为已有的安全预算：

```go
expandThreshold = gpvNATSPayloadBudgetBytes
```

即使用 5 MiB 的 75%，而不是完整 5 MiB。

目标方案中：

- task terminal event 不携带 raw SOAP。
- command result event 不直接携带超大 parameter values。
- 大结果写数据库分块表或对象存储。
- NATS 只传 `result_ref`。
- consumer 分页读取结果并写 staging。

## 12. 分阶段实施

### 阶段一：止血修复

- 修复 provisioning 正常成功和零任务场景的收口。
- `StartPathBSync` 改为明确返回 disposition。
- `RPCResponseSubscriber` 跳过 `source=param_sync` 的 GPV，消除双写。
- 非 full GPV 禁止更新 `last_param_sync_at`。
- manual scoped、Config Pull、License 使用明确 partial 标识。
- 修复 Fault 全请求路径失败统计。
- 大对象展开阈值改为 75% NATS 预算。

阶段效果：解决卡死、双写、局部冒充全量以及原评审中的三个 WARNING。

### 阶段二：引入 SyncRun

- 增加 runs、bindings、task_results、staging 表。
- 所有同步入口统一调用 `ParamSyncService.Start`。
- 参数同步 task 的 `source_id` 统一指向 run。
- 参数树状态和结果汇总改为查询 run。

阶段效果：同步状态不再依赖时间窗口、command key 或 Redis 临时 hint。

### 阶段三：可靠结果消费

- 引入专用 result event。
- 建立单一结果消费者。
- ACK 等待数据库事务完成。
- 实现 task result 幂等记录。
- 实现 full staging 和一次性 finalize。

阶段效果：参数投影可重放、可恢复，并具备接近 exactly-once 的业务语义。

### 阶段四：结果去 payload 化

- raw SOAP 和大参数结果迁移到数据库分块表或对象存储。
- NATS 只传引用。
- 删除 Redis pending/reason 临时状态。

阶段效果：大对象、Redis TTL、服务重启和 NATS payload 上限不再影响同步正确性。

## 13. 修改后可达成的效果

- Provisioning 不再永久停留在 `syncing`。
- 同设备不会出现两轮参数同步并发污染。
- 同一参数不会被两个消费者用不同语义覆盖。
- 进程崩溃、数据库短暂故障后结果可以自动重投。
- 重复事件不会重复写入、重复计数或重复 finalize。
- 全量同步只在所有批次都处理成功后显示成功。
- 任一批次失败时整轮明确失败，不会被后续成功事件覆盖。
- 局部 GPV 不影响周期全量同步时间，也不会触发参数删除。
- diff/reconcile 基于整轮快照，不再产生批次级误判。
- 结果统计能够准确区分成功路径、失败路径和 Fault 详情。
- 大对象不再依赖将完整业务结果塞入 5 MiB NATS envelope。
- Redis 继续承担任务执行加速，但不再承担同步正确性的权威状态。

## 14. 验收标准

至少覆盖以下自动化场景：

1. 正常单批、多批 full sync 均只 finalize 一次。
2. provisioning 启动新 run、加入现有 run、零可同步路径均能进入明确终态。
3. 任一批次 failed/expired/cancelled 后，run 最终为 failed，且不会被其他成功批次覆盖。
4. 数据库暂时失败时事件 NAK，恢复后自动重投并成功落库。
5. 同一事件重复投递五次，参数、计数和 hook 均只执行一次。
6. worker 在 ACK 前退出，重启后事件重新投递。
7. partial sync 不更新 `last_param_sync_at`，不执行 diff/reconcile。
8. full sync 的 missing diff 使用整轮结果，而不是最后一个批次。
9. failed GPV 的全部请求路径均统计失败，`param_faults` 仅补充详情。
10. 不支持存储的 mapping 不会被其他消费者重新写入。
11. 5 MiB 临界对象不会产生超过 NATS 上限的 event。
12. 服务重启不会重放历史 task.failed 并覆盖当前设备同步状态。

## 15. 多请求并发、时序与一一对应

### 15.1 Request 与 Run 必须分离

`SyncRequest` 表示一次调用者请求，必须一一对应并可独立查询；`SyncRun` 表示设备实际执行的一轮同步，可以承载多个兼容请求。

```text
SyncRequest（每次调用独立一条）
        │
        ▼
SyncRequestBinding
        │
        ▼
SyncRun（设备实际执行的一轮）
        │
        ▼
DeviceTask → TaskResult
```

完整关联链为：

```text
request_id
    ↓
request_binding
    ↓
run_id
    ↓
task_id + batch_no
    ↓
task_result
    ↓
request_result
```

调用方始终使用 request ID 跟踪自己的请求；run ID 用于内部执行聚合；task ID 用于协议批次和故障诊断。

### 15.2 不同设备并行、同设备串行

不同设备拥有独立执行通道：

```text
Device A: Run A1 → Run A2 → Run A3
Device B: Run B1 → Run B2
Device C: Run C1
```

同一设备任何时刻最多存在一个 active run：

```text
Device A: Full Run #101 → Partial Run #102 → Full Run #103
```

一个 active run 内部的全部 GPV 批次预先进入该设备队列，但 ACS 始终保持单 RPC 在途：

```text
同一 CWMP Session:
GPV Batch 1 → Response 1 → GPV Batch 2 → Response 2 → … → Session End
```

“同设备串行”约束的是 RPC 在途数量和 run 顺序，不等于“上一批结果经过异步消费后才释放下一批”。后者会导致会话提前关闭并为每批重新唤醒设备，属于禁止的性能退化。

请求创建时，在数据库短事务内锁定 `parameter_sync_device_sequences` 对应设备行并分配序号：

```sql
SELECT next_sequence
FROM parameter_sync_device_sequences
WHERE device_id = $1
FOR UPDATE;
```

多个应用实例同时接收同设备请求时，最终也只能依次获得 `101、102、103...`。设备请求的业务顺序以该序号为准。

调度器领取下一请求时使用：

```sql
SELECT *
FROM parameter_sync_requests
WHERE device_id = $1
  AND status = 'queued'
ORDER BY device_sequence
LIMIT 1
FOR UPDATE SKIP LOCKED;
```

数据库 active-run 唯一索引承担最终并发防线。前一 run 必须在 Finalizer 成功或失败事务提交后，调度器才能启动同设备下一 run。

### 15.3 全量和局部请求的合并矩阵

| 当前设备状态 | 新请求 | 处理方式 |
|---|---|---|
| Full 正在运行 | Full | 加入当前 Full run |
| Full 正在运行 | Partial | Full 计划完整覆盖请求路径时加入，否则排队 |
| Partial 正在运行 | Full | 不加入，Full 排在 Partial 后面 |
| Partial 正在运行 | Partial | 尚未生成 task 时可合并；已经下发则排队 |
| Full 正在 processing/finalize | 任意请求 | 排队，不允许修改正在收口的快照 |
| 无运行中 run | 任意请求 | 创建新 run |

#### Full 加入 Full

```text
Request F1 ─┐
            ├─ Full Run R100
Request F2 ─┘
```

R100 完成后，F1、F2 分别产生自己的 request result。任何一个调用者取消等待都不能取消共享 run，除非该 run 已经没有其他有效 request binding。

#### Partial 加入 Full

只有 Full run 的已冻结计划覆盖 Partial 请求的全部标准路径时才允许加入：

```text
Request Full ───────┐
                    ├─ Full Run R101
Request Partial A ──┘
```

Partial A 最终只返回自己请求路径的统计和失败详情。若请求路径不在 Full MappingSet、被 unsupported 过滤或 Full run 已进入 processing，则 Partial 必须排队创建独立 run。

#### Full 到达正在运行的 Partial

Full 不得加入 Partial：

```text
Partial Run R102
        ↓ 完成
Full Run R103
```

否则可能把 Partial 的有限结果误用作全量快照。

#### Partial 合并 Partial

只有在 run 仍处于 `planning/enqueuing` 且尚未冻结 task 计划时，才允许将多个 Partial 请求路径取并集生成任务。task 已创建或下发后，新 Partial 请求必须排队，避免动态修改 `expected_task_count` 和收口边界。

### 15.4 请求级结果计算

多个请求共享一个 run 时，不能直接复制 run 的整体状态，而应根据 binding 的 `covered_paths` 独立计算。

例如：

```text
Run 请求路径：A、B、C、D
失败路径：C

Request 1：A、B  → succeeded
Request 2：C     → failed
Request 3：A、D  → succeeded
```

因此建议支持：

```text
Run 状态：succeeded / partially_failed / failed
Request 状态：succeeded / failed / cancelled
```

具体规则：

- Full request：任一最终不可恢复 task 失败即失败，且 full staging 不提交。
- 独立 Partial request：只根据自己的 requested paths 判断成功或失败。
- 加入 Full run 的 Partial request：推荐使用路径模式；如果自身 covered paths 全部获得成功结果，可以成功，但 Full request 仍失败且不提交全量快照。
- 调用者取消一个 joined request 时，只取消该 request binding；不能直接取消其他调用者共享的 run。

### 15.5 防止旧结果覆盖新结果

结果消费者处理事件时必须依次校验：

1. task ID 是否属于事件中的 run ID。
2. run ID 是否属于该设备。
3. `parameter_sync_task_results` 是否已经存在该 task 结果。
4. run 是否仍允许接收该 task 的结果。
5. task 是否在 run 冻结的执行计划内。

处理规则：

- task result 已存在：幂等返回并 ACK。
- run 已终态但收到重复事件：记录审计，禁止再次写 staging、正式参数或计数。
- Full run 只能写自己的 staging，不能直接覆盖正式参数。
- 下一 run 只能在上一 run Finalizer 提交后启动。
- 下一 run 启动后到达的上一 run 重复事件，只能命中已有幂等记录，不能覆盖新数据。

### 15.6 Redis 与 NATS 的顺序职责

- PostgreSQL：请求序号、active run、request/run/task 关联和结果幂等，是唯一顺序权威。
- Redis：task 待执行加速队列，不决定业务先后和同步终态。
- NATS：结果通知与可靠重投，不以消息到达先后决定设备同步顺序。

Redis PopTask 后仍需校验 task 所属 run 是该设备当前 active run。非当前 active run 的 task 不允许下发，应移出执行队列并交由恢复器重新判断。

### 15.7 对外接口

提交同步请求：

```http
POST /devices/{id}/sync-params
Idempotency-Key: <caller-generated-key>
```

建议返回：

```json
{
  "request_id": "req-uuid",
  "run_id": "run-uuid",
  "scope": "partial",
  "status": "joined",
  "device_sequence": 103,
  "queue_position": 0,
  "joined_existing_run": true
}
```

独立查询请求：

```http
GET /parameter-sync/requests/{request_id}
```

查询结果只返回该 request 的路径统计、失败详情和 run 关联。

查询设备同步队列：

```http
GET /devices/{id}/parameter-sync/runs
```

返回当前 active run、排队 requests、设备 sequence、request/run 合并关系以及 task/processed 进度。

### 15.8 并发场景补充验收

1. 两个实例同时提交同设备请求，只能生成连续且不重复的 device sequence。
2. 不同设备请求可并行执行，同设备请求严格按 sequence 串行。
3. 两个 Full request 加入同一 run，但分别获得独立 request result。
4. Partial 只有在路径被 Full 计划完整覆盖时才能加入。
5. Partial run 执行期间到达的 Full request 必须排队。
6. 已冻结 task 计划后到达的 Partial request 不能动态并入当前 run。
7. 合并 Partial 中某路径失败时，只影响覆盖该路径的 request。
8. 取消 joined request 不影响共享 run 和其他 request。
9. 上一 run 的重复或迟到事件不能写入下一 run staging，也不能覆盖正式参数。
10. 相同 idempotency key 重试只返回原 request，不创建新 sequence 或新 run。

## 16. 10 万级设备规模化控制面

### 16.1 与基础方案的关系

基础方案解决“单设备的一轮同步如何正确、可靠地完成”，10 万级方案在其上增加大规模调度控制面：

```text
基础方案 = 可靠数据面
规模方案 = 可靠数据面 + 10 万设备控制面
```

可靠数据面包括 SyncRequest/SyncRun/DeviceTask 稳定关联、task terminal 与 result processed 分离、单一结果消费者、ACK-after-commit、task result 幂等、full staging 和一次性 Finalizer。

规模控制面包括 SyncCampaign、分布式 Scheduler、多级限流、公平调度、渐进任务物化、补偿器、数据分区和容量治理。

### 16.2 可以分两步实施，但基础契约必须一次到位

可以先实现现有可靠数据面，再建设 10 万设备控制面。为了避免规模化阶段二次重构，以下能力必须在第一阶段落下：

1. 每次触发都有 request ID 和可查询终态。
2. 每轮设备执行都有 run ID。
3. 参数同步 task 的 `source_id` 统一指向 run ID。
4. task 执行终态和结果消费完成分别记录。
5. task result 具有数据库幂等键。
6. run/task 创建使用事务和 Outbox，禁止依赖 PG + Redis 非原子双写。
7. full sync 使用 staging，不能逐批直接作为全量结果收口。
8. 结果事件采用轻量引用契约，不能绑定完整 raw payload。
9. Request 表预留 `priority`、`next_attempt_at`、`deadline_at`、`campaign_id` 等调度字段。
10. Scheduler 通过接口访问请求队列，业务入口不能直接批量创建全部 device_tasks。

可以后置到规模控制面阶段的能力：

- Campaign 分批扫描和总体进度。
- 多实例 Scheduler 分片领取。
- 按运营商、区域、产品型号和 ACS 实例配置配额。
- 加权公平调度和动态速率控制。
- PostgreSQL 表分区和批量写入专项优化。
- 大结果对象存储、冷热分层和自动扩缩容。

### 16.3 规模化整体架构

```text
用户操作 / 上线事件 / 固件变化 / 周期策略
                         │
                         ▼
                   SyncRequest API
               ├─ 持久化 request
               ├─ 返回 request_id
               └─ 写 Request Outbox
                         │
                         ▼
                 Distributed Scheduler
               ├─ 分片领取 request
               ├─ 全局/分组限流
               ├─ 同设备互斥
               └─ 创建 SyncRun
                         │
                         ▼
                    Task Planner
               ├─ MappingSet
               ├─ full / partial
               ├─ GPV 批次
               └─ PG task + Task Outbox
                         │
                         ▼
              Redis 执行索引 / ACS Inform
                         │
                         ▼
               Result Blob + Result Event
                         │
                         ▼
           分片 ParamSyncResultConsumer
               ├─ task_id 幂等
               ├─ 写 full staging
               ├─ 更新 run 计数
               ├─ 单设备 finalize
               └─ 更新 request 终态
```

批量周期同步在请求层之上增加 Campaign：

```text
SyncCampaign
    ├─ Device Request 1
    ├─ Device Request 2
    ├─ ...
    └─ Device Request 100000
```

Campaign 不能一次性为 10 万设备创建全部 GPV task，而应根据系统水位渐进生成 request；Scheduler 只有在获得执行额度时才物化 run 和 task。

### 16.4 SyncCampaign

```sql
CREATE TABLE parameter_sync_campaigns (
    id                    uuid PRIMARY KEY,
    status                varchar NOT NULL,
    trigger_reason        varchar NOT NULL,
    total_devices         integer NOT NULL,
    generated_requests    integer NOT NULL DEFAULT 0,
    succeeded_requests    integer NOT NULL DEFAULT 0,
    failed_requests       integer NOT NULL DEFAULT 0,
    deduplicated_requests integer NOT NULL DEFAULT 0,
    started_at            timestamptz,
    completed_at          timestamptz,
    created_at            timestamptz NOT NULL DEFAULT now()
);
```

Campaign 执行规则：

1. 使用设备主键游标分页，禁止大 OFFSET。
2. 每轮按可配置批量生成 request，例如 500～2000 条。
3. 只有 request backlog 和执行水位低于阈值时才继续生成。
4. Campaign 不直接生成 device task。
5. 单设备 request 被 Scheduler 领取并获得配额后才创建 run/task。
6. Campaign 汇总每个 request 的 succeeded/failed/deduplicated/timed_out 终态。

### 16.5 Request 是每次触发的持久化回执

10 万设备场景下，Request 的首要职责是保证每次触发都有结果，而不是实现复杂的用户请求合并。

建议终态包括：

```text
succeeded / failed / timed_out / cancelled / deduplicated / rejected
```

自动请求遇到同设备 active full run 时不能静默跳过，应写入明确结果：

```json
{
  "request_id": "req-uuid",
  "status": "deduplicated",
  "result_code": "ACTIVE_SYNC_EXISTS",
  "active_run_id": "run-uuid"
}
```

用户操作继续沿用当前前后端互斥，冲突时返回 409；Provisioning 可以作为特例绑定已有兼容 full run。基础阶段不要求实现通用 Full/Partial 合并或同设备无限请求队列。

建议为 `parameter_sync_requests` 预留：

```sql
ALTER TABLE parameter_sync_requests
    ADD COLUMN campaign_id uuid,
    ADD COLUMN priority integer NOT NULL DEFAULT 100,
    ADD COLUMN next_attempt_at timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN deadline_at timestamptz,
    ADD COLUMN result_code varchar,
    ADD COLUMN active_run_id uuid;
```

### 16.6 分布式 Scheduler 与公平调度

多实例 Scheduler 使用 PostgreSQL 短事务领取请求：

```sql
SELECT id
FROM parameter_sync_requests
WHERE status = 'queued'
  AND next_attempt_at <= now()
  AND (deadline_at IS NULL OR deadline_at > now())
ORDER BY priority, created_at
LIMIT $batch_size
FOR UPDATE SKIP LOCKED;
```

领取后立即提交事务，不在设备执行期间持有数据库锁或 advisory transaction lock。

Scheduler 至少控制：

- 全局每秒启动设备数和 executing 设备数。
- 每个 ACS 实例执行配额。
- 每运营商、区域、产品型号执行配额。
- 每设备一个 active full run。
- 每设备 task 队列深度。
- NATS ack pending / pending bytes。
- PostgreSQL 连接池和写入延迟。

建议优先级：

```text
P0：Provisioning / 固件变化
P1：用户手工同步
P2：设备重新上线
P3：周期全量同步
P4：后台补偿重试
```

不能只使用严格优先级，应通过加权公平调度避免周期任务、某运营商或某产品型号长期饥饿。

### 16.7 多级背压

规模化方案必须在 task queue 之前控制扩张：

```text
Campaign
    ↓ 控制 request 生成速度
Request Queue
    ↓ 控制 run 启动速度
SyncRun
    ↓ 控制 task 物化速度
Task Queue
    ↓ 控制 ACS RPC 下发速度
Result Consumer
    ↓ 控制数据库写入速度
```

禁止等 Redis 已堆积数百万 task 后才开始限流。

### 16.8 设备资源状态与故障隔离

run 的资源状态至少区分：

```text
waiting_device：等待 Inform，不占 RPC worker
executing：正在执行 RPC，占 ACS 执行配额
processing：结果落库，占 result worker
terminal：已完成，不占运行资源
```

每个 request/run 独立维护 deadline、retry count、next attempt 和 error code。单设备连续超时、长期离线或产生 poison result 时，只失败或熔断该设备，不得占满全局 worker。

结果消费者按 `hash(device_id) % shard_count` 分片；同设备事件进入同一 shard 保序，不同设备并行。使用固定 shard/worker pool，禁止为 10 万设备创建 10 万常驻 goroutine。

### 16.9 可靠入队与补偿闭环

规模化环境中 Outbox 是必选项：

```text
BEGIN
  INSERT parameter_sync_runs
  INSERT device_tasks...
  INSERT task_outbox...
COMMIT
```

Outbox dispatcher 负责将 task 推入 Redis；Redis 不可用时持续重试。Redis 只是可重建的执行索引，PG 始终为权威数据。

为了保证每个 request 最终都有结果，需要三个补偿器：

- Request Sweeper：处理长期 accepted/queued/dispatching 请求，超过 deadline 转为 `timed_out`。
- Run Reconciler：从 `device_tasks` 重建 expected/terminal/failed 计数并修复不一致。
- Result Reconciler：查找 task 已终态但 task result 不存在的记录，重新发布或直接补处理。

### 16.10 大结果与批量写入

NATS 只传结果引用：

```json
{
  "event_id": "evt-uuid",
  "request_id": "req-uuid",
  "run_id": "run-uuid",
  "task_id": "task-uuid",
  "device_id": "device-uuid",
  "result_ref": "param-result/run/task",
  "success": true
}
```

原始 SOAP 和大参数结果写数据库分块表或对象存储。正式参数写入使用 `pgx.CopyFrom`、`unnest` 或等价批量方式，禁止逐参数单条 upsert。

数据层建议：

- `device_parameters` 以 `device_id` 为主要访问和分区键。
- request/run/task result 按时间分区或设置明确保留期。
- staging 按 run/device 分区并设置短生命周期。
- raw result 使用对象存储和冷热分层。

### 16.11 容量计算

```text
启动速率 = 设备数 / 同步窗口秒数
任务吞吐 = 启动速率 × 平均 task 数/设备
执行并发 ≈ 启动速率 × 平均同步持续时间
```

例如 10 万设备要求 6 小时完成，平均每台 30 个 task、平均持续 24 秒：

```text
启动速率：100000 / 21600 ≈ 4.63 台/秒
任务吞吐：4.63 × 30 ≈ 139 task/秒
执行并发：4.63 × 24 ≈ 111 台
```

按两倍余量设计的初始目标可设为 10 台/秒、300 task/秒、250 台同时 executing。实际参数必须通过压测并结合 Inform 周期、批次数、RPC 延迟和数据库延迟校准。

### 16.12 规模化监控与不变量

必须监控 Campaign 完成率、Request 各终态、queued age P95/P99、waiting/executing/processing run 数、启动速率、task/result 吞吐、NATS 重投、Redis 队列深度、Outbox backlog、结果处理延迟和各设备分组成功率。

关键不变量：

```text
每个 request 最终必须进入终态
每个 active device 最多一个 full run
processed_task_count <= expected_task_count
failed_task_count <= processed_task_count
succeeded run 必须 expected = terminal = processed
Full succeeded 后才允许更新 last_param_sync_at
```

### 16.13 分阶段交付路线

#### 交付 A：可靠数据面

- SyncRequest / SyncRun / task result 基础表。
- 同设备 active run 唯一约束。
- 明确 full/partial/readback。
- 单一结果消费者。
- ACK-after-commit 和 task result 幂等。
- full staging + Finalizer。
- Request/Run 基础终态查询。
- Outbox 基础能力。

交付效果：先解决卡死、双写、结果丢失、错误收口和每次请求无结果的问题，同时为规模控制面提供稳定接口。

#### 交付 B：10 万设备控制面

- SyncCampaign。
- 分布式 Scheduler。
- 分片 worker 和固定资源池。
- 多维配额、加权公平和多级背压。
- 三类 Sweeper/Reconciler。
- 批量写入、表分区和结果对象存储。
- 容量压测和动态速率控制。

交付效果：在不修改可靠数据面业务契约的前提下，将系统扩展到 10 万级设备的受控批量同步。
