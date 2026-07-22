# 参数同步堆积影响真实设备 - 现场 RCA 与代码修复方案

- 报告日期：2026-07-17
- 现场环境：`http://172.24.224.78:8081`
- 前端版本：`v100.0.0-20260717-0708`
- 首轮排查：浏览器只读检查 + SSH 只读 SQL/NATS/日志/容器状态检查
- 本轮复核：已登录现场浏览器的只读 API 检查 + root SSH 只读 SQL/NATS/日志/容器状态检查
- 约束：本报告只保留现场证据、原因分析和代码修复方案；两轮排查均未执行取消、重启或数据修改。

## 0. 已确认的方案决策

以下决策作为本方案的硬约束：

1. **Bootstrap 不触发全量参数同步**。Bootstrap 只负责设备入网、绑定、生命周期状态和必要的模型准备。
2. **DeviceOnline 不作为参数同步入口**。DeviceOnline 仍可供其他设备状态消费者使用，但不得创建参数同步 request、run 或 GPV task。
3. **ModelUpload 流程结束后必须触发一次参数同步**。ModelUpload 请求本身不直接触发；当设备支持并成功完成上传，或确认设备不支持 ModelUpload 并完成能力判定后，均追加一次 `trigger_reason=model_upload` 的 durable 参数同步。支持状态记录为 `uploaded`，不支持状态记录为 `not_supported`，但两种状态都必须同步一次，且不得阻塞或回退到 Path B。
4. **FirmwareChanged 不触发参数同步**。如业务需要，可保留模型/设备元数据刷新，但不得调用参数同步链路。
5. **自动参数同步只保留 Periodic 和 ModelUpload 流程结束后的同步**。定时任务关闭时不创建新的周期同步，但 ModelUpload 成功或不支持后的同步不能被周期开关阻止。
6. **Manual 是独立的用户触发入口**，必须直接走 durable `parameter-sync`，不受定时任务开关影响。
7. **所有同步语义统一走 `parameter-sync`**：Periodic、Manual、ModelUpload 流程终态后的同步、Config Pull、License 参数同步均进入 `parameter_sync_requests -> parameter_sync_runs -> device_tasks(source=param_sync) -> result/recovery -> finalize` 生命周期；不存在任何 Path B 回退。
8. **Path B 和所有 legacy fallback 全部删除**。`StartPathBSync`、`EnqueueGPVBatches`、普通 `sync-gpv` task 及 `handled=false` fallback 不再承担参数同步职责。
9. MML、SPV 等明确的单次 RPC readback 不属于参数同步，不因本方案强行改造成 `parameter-sync`。

## 1. 结论

本次堆积不是“定时周期参数同步”导致。本轮通过已登录 API 重新确认 `device.periodicSyncEnabled=false`；原抽查设备仍存在一个由 `bootstrap` 触发的 durable run，说明问题来自历史上线自动同步链路，而不是 UI 周期扫描。按本次决策，修复后 Bootstrap、DeviceOnline 和 FirmwareChanged 均不创建参数同步 run；ModelUpload 无论成功上传还是确认不支持，在流程终态都追加一次 `model_upload` 同步，自动周期同步仍只在 Periodic 开启后运行，Manual 独立保留。

本轮没有重新获得足够证据证明同一轮上线必然同时创建了 Bootstrap 和 DeviceOnline 两个 run，因此“重复入口”应作为待关联验证的放大因素，不应写成已完全证实的唯一入口根因。无论是否重复，本次修复都直接移除这两个生命周期事件的参数同步触发。

现场问题的主体是：

```text
大量设备上线/自动同步请求
  -> 历史代码由 bootstrap/device_online 创建 durable run
  -> 批量 GPV 任务完成后，result 消费/收口能力不足
  -> device_tasks 已经终态
  -> parameter_sync_task_results 未及时写入 / 回补
  -> processed_task_count 不推进
  -> run 停在 waiting_device
  -> request 仍为 running 且 active_run_id 不清空
  -> 前端和接口认为该设备仍在参数同步中
```

超时容错重试可能放大压力，但本轮现场 API 证据不能单独证明它是主因。当前已确认的直接形态不是“任务一直重试未结束”，而是“任务已经终态，durable result 处理/恢复追不上”。数据库竞争导致“生产速率高于收口速率”目前是最强解释，仍需用同一时间窗口的增量指标完成验证，不能把单次快照直接写成完全证实的唯一根因。

通俗地说：设备侧那一批 GPV 任务已经跑完了，像快递已经送到门口；但后台还需要把“每个任务的完成回执”逐条登记进 `parameter_sync_task_results`，再更新 run/request 状态。现在卡住的是登记回执和汇总收口这一步。回执登记追不上，系统就一直认为这次同步还没正式结束，于是 `active_run_id` 不清空，新同步也会被挡住。

## 2. 现场证据

### 2.1 页面与接口状态（本轮复核）

设备列表可正常加载：

```text
版本：v100.0.0-20260717-0708
设备列表总数：10003
第一页抽样 200 台：param_sync_running = true 的数量为 0
第一页设备 SN 前缀：`120200024118AA`
第一页设备状态：在线 200 台，`sync_status=error` 200 台
```

抽样设备：

```text
SN = 120202662098AA02232
device_id = 1e5633bb-0987-4632-9f1d-61a9030825bc
```

`GET /api/v1/devices/:id/parameter-sync/active` 返回：

```text
stalled_finalizing = true
reason = terminal tasks completed but results are not processed
run_id = 0301128b-3fa5-4f00-a2de-09f694a598fd
request_id = 6055877d-60f2-4a99-9865-35288b127674
status = waiting_device
trigger_reason = bootstrap
expected_task_count = 32
terminal_task_count = 32
processed_task_count = 0
failed_task_count = 0
```

同一设备的 `GET /api/v1/devices/:id/parameters/sync-status` 返回：

```text
status = idle
pending_commands = 0
last_param_sync_at = 2026-07-16T11:47:27.819322Z
last_sync_gpv.task_count = 32
last_sync_gpv.successful_commands = 32
last_sync_gpv.failed_commands = 0
last_sync_gpv.first_created_at = 2026-07-16T07:29:53.879292Z
last_sync_gpv.last_completed_at = 2026-07-16T08:32:46.7937Z
```

本轮可以把两条证据关联到同一个 run：active run `started_at=2026-07-16T07:29:53.579672Z`，GPV 首个 task 创建于 `07:29:53.879292Z`，随后全部 32 个 task 终态，但 processed 仍为 0。因此该设备确实是当前 active bootstrap run 的 durable result 收口卡住，而不是上一轮同步状态误关联。

注意：设备列表是分页/排序结果，第一页 0 台 running 不能推导全局没有堆积；原抽查设备仍为 `param_sync_running=true`。

### 2.2 周期同步配置与历史全局状态

本轮已登录 API `GET /api/v1/admin/sysConfig?category=device` 返回：

```text
periodicSyncEnabled              = false
periodicSyncIntervalMinutes      = 1440
periodicSyncBatchSize            = 200
periodicSyncMaxConcurrent        = 10
periodicSyncStaggerWindowMinutes = 2
```

因此，本轮可直接排除 UI 配置的周期同步入口。

本轮 PostgreSQL 实时统计（UTC 2026-07-17 02:43~02:46，独立只读查询非同一事务快照）：

```text
parameter_sync_requests:
accepted     = 1
deduplicated = 108255
failed       = 5141
rejected     = 17462
running      = 5753
succeeded    = 2980

parameter_sync_runs:
cancelling     = 8
executing      = 9
failed         = 5140
succeeded      = 2981
waiting_device = 5736

stalled_finalizing = 4512
```

run 状态按触发来源聚合：

```text
trigger_reason | status         | count
---------------+----------------+------
device_online  | failed         | 5140
bootstrap      | waiting_device | 4470
bootstrap      | succeeded      | 2296
device_online  | waiting_device | 1265
device_online  | succeeded      | 683
device_online  | executing      | 8
device_online  | cancelling     | 8
manual         | succeeded      | 2
bootstrap      | executing      | 1
manual         | waiting_device | 1
```

结论：周期同步关闭；历史 active run 主要来自上线链路，其中 `bootstrap waiting_device=4470`、`device_online waiting_device=1265`。这证明历史版本两个入口都制造过积压，但是否同一轮上线重复建 run，仍需按 `device_id + 时间窗口 + request/run` 关联查询。修复后 Bootstrap 和 DeviceOnline 均不再创建参数同步 request；自动参数同步只保留 Periodic 和 ModelUpload 流程终态后的同步，Manual 独立保留。

### 2.3 device_tasks 与 result 表不匹配

本轮 `device_tasks`：

```text
source     | status    | count
-----------+-----------+-------
param_sync | cancelled | 1898
param_sync | completed | 175219
param_sync | expired   | 227254
param_sync | pending   | 40992

terminal_param_sync_tasks = 404371
parameter_sync_task_results = 260559
```

重试字段聚合：

```text
status=completed, retry_count=0: 175210
status=completed, retry_count=1: 9
status=expired,   retry_count=0: 227254
status=pending,   retry_count=0: 40992
```

当前没有看到 expired task 被大规模 retry 的证据；“重试风暴”不能作为本轮主根因。

runs 计数/进度聚合（02:44 独立查询快照，包含终态 run；因此不能和 02:43 状态总表逐项对齐）：

```text
status         | runs | expected | terminal | processed
---------------+------+----------+----------+----------
cancelling     | 8    | 256      | 256      | 248
executing      | 23   | 736      | 736      | 482
failed         | 5140 | 164480   | 164480   | 164480
succeeded      | 2873 | 91974    | 91974    | 91974
waiting_device | 5735 | 183501   | 139950   | 0
```

stalled missing result 按触发来源：

```text
trigger_reason | runs | unprocessed_tasks
---------------+------+----------------
bootstrap      | 3115 | 99680
device_online  | 1264 | 40448
manual         | 1    | 13
```

在 02:44 的 run 聚合中未处理 task 为 140141；随后独立 task/result 查询在 02:46 已达到终态 task 404371、result 260559，差额 143812。由于两组查询不是同一事务快照，且期间设备仍在持续上线/完成，这组数据只能证明现场存在大规模终态 task 与 durable result 的存量缺口，不能单独证明“终态 task 生成速度持续超过 durable result 登记速度”。“生产速率高于收口速率”需要用同一时间窗口内的增量数据确认：至少记录 terminal task 增量、processed result 增量、PARAM_SYNC pending 增量、PGResultProcessor 成功/失败/耗时，以及按 `run_id` 聚合的缺失结果变化。

### 2.4 批量设备段集中制造堆积

本轮 root SQL 的设备按 SN 前缀：

```text
sn_prefix      | alive | lifecycle_state | devices | online
---------------+-------+-----------------+---------+-------
120202662099BB |       | commissioned    | 2500    | 2492
120202662098AA |       | commissioned    | 2500    | 2493
120299024119AA |       | commissioned    | 2500    | 2500
120200024118AA |       | commissioned    | 2500    | 2498
120200054822CH | true  | commissioned    | 1       | 1
C6C07FEA7CBB9B | true  | commissioned    | 1       | 1
1202000240194D | true  | commissioned    | 1       | 1
```

当前 active/unprocessed run 按 SN 前缀：

```text
sn_prefix      | status         | trigger_reason | runs | missing
---------------+----------------+----------------+------+--------
120299024119AA | waiting_device | bootstrap      | 1795 | 57440
120202662098AA | waiting_device | bootstrap      | 1564 | 50048
120202662099BB | waiting_device | bootstrap      | 1142 | 36544
120202662099BB | waiting_device | device_online  | 769  | 24608
120202662098AA | waiting_device | device_online  | 494  | 15808
```

结论：本轮实时 SQL 确认堆积主体集中在四个 2500 台批量设备段；其中 `120299024119AA` 的 Bootstrap 未处理量最高，其次为 `120202662098AA`、`120202662099BB`。第四段 `120200024118AA` 当前没有进入前 20 的 stuck run 聚合，不能据此认为其完全不受 DB 压力影响。

### 2.5 NATS 与 DB 压力（本轮复核）

NATS monitoring API `http://127.0.0.1:8222/jsz?streams=true&consumers=true`：

```text
STREAM PARAM_SYNC:
messages = 58569
bytes = 33150131
first_seq = 706954
last_seq = 765651

CONSUMER param-sync-results-pull:
delivered = 34618
ack_floor = 34504
pending = 58505
ack_pending = 64
redelivered = 0
```

容器 CPU：

```text
omcgo-postgres-1  CPU ~= 333%
omcgo-app-1       CPU ~= 101%
omcgo-acs-1       CPU ~= 33%
omcgo-worker-1    CPU ~= 1%
omcgo-nats-1      CPU ~= 15%
```

PostgreSQL 活跃 SQL 主要表现为：

```text
大量设备列表/设备在线查询等待 `LWLock:LockManager`
INSERT INTO device_parameters ... ON CONFLICT DO UPDATE
UPDATE device_tasks ...
UPDATE devices ...
autovacuum VACUUM ANALYZE device_parameters_p09
```

app 日志本轮出现：

```text
lookup disconnected alarm on device online failed: resource not found
batch processor: worker channel full, dropping update
nats: slow consumer, messages dropped on connection for `device.registered`
```

结论：本轮确认 `param-sync-results-pull` 仍在消费且没有 redelivery；PARAM_SYNC pending 已达 58505，PG CPU 约 333%，大量查询等待 LockManager，app 还出现 batch worker channel full。现阶段最强解释是全局设备上线/更新流量与 paramsync result/recovery 共享主库，导致 PG 锁/写吞吐竞争，result consumer 收口能力不足并形成 backlog；但该结论仍应通过前述时间窗口增量指标验证。

同时，`batch processor: worker channel full, dropping update` 和 NATS `messages dropped` 不是单纯的性能信号，而是事件可能丢失的独立正确性风险。后续修复必须保证溢出事件可持久化、重放或由数据库补偿扫描恢复，不能只提高 result consumer 吞吐。

## 3. 为什么不是“定时同步”

本轮 API 读取到页面对应的周期参数同步配置为 `periodicSyncEnabled=false`。同时原抽查设备的 active run 原因是 `bootstrap`，不是周期扫描。

现场历史配置与目标配置如下：

```yaml
# 现场历史配置（本轮排查时）
provision:
  auto_sync:
    enabled: true
    sync_on_bootstrap: true
    sync_on_firmware_change: true
    max_concurrent: 20

# 历史报告中的旧目标配置（已废弃）
provision:
  auto_sync:
    enabled: true
    sync_on_bootstrap: false
    sync_on_device_online: false
    sync_on_firmware_change: false
    max_concurrent: 20

param_sync:
  run_enabled: true
  result_consumer_enabled: true
  staging_enabled: true
  canary_percent: 100
  legacy_fallback_enabled: false

periodic_sync:
  enabled: false                    # true 时才允许自动参数同步
```

因此，本次要区分两个概念：

```text
周期参数同步：
  UI 中的定时扫描开关，当前未开启；修复后这是唯一自动参数同步入口。

生命周期事件：
  Bootstrap、DeviceOnline、FirmwareChanged 只处理各自的生命周期和设备状态，不创建参数同步 request/task；ModelUpload 只有在成功完成且设备支持时，才追加一次 `model_upload` durable request。

手动参数同步：
  用户主动触发时直接提交 durable `parameter-sync` request，不受周期同步开关影响。
```

现场堆积来自历史生命周期自动同步链路，而不是周期扫描。修复后该历史链路不再产生新的参数同步任务；所有仍需执行的同步统一进入 `parameter_sync`，不再创建或回退到 Path B。

## 4. 超时容错重试 vs 上线消息

当前证据显示：

1. `device_online failed = 5140`，说明失败路径确实存在并制造压力。
2. 当前更大规模的 stuck run 是 `bootstrap waiting_device = 4470` 和 `device_online waiting_device = 1265`。
3. `waiting_device` 中有 `expected=183501`、`terminal=139950`、`processed=0`，说明一部分任务仍在执行/等待，另一大部分终态结果尚未收口。
4. `device_tasks` 的 retry 聚合显示：只有 9 个 completed task 的 `retry_count=1`，expired task 没有 retry；当前不能把“超时容错重试”定为主因。
5. 卡住点是 `processed_task_count` 不推进，即 result 处理 / recovery 处理不完。

所以判断是：

```text
上线链路是已确认的历史入口；原抽查设备当前 run 是历史 Bootstrap run。
Bootstrap 与 DeviceOnline 是否对同一轮上线重复建 run，仍需关联查询验证；方案决策不依赖该关联结果，直接停止两者的参数同步触发。
当前 retry 统计不支持“重试风暴是主因”；后续应把 retry 作为独立容量控制项治理，而不是 P0 根因修复项。
最终阻塞点在 durable result 消费与 PG recovery 收口，而不是设备侧任务未完成。
```

## 5. 为什么真实设备也会受影响

真实设备不会因为批量设备段的 SN 直接被改数据，但会受到两类间接影响：

1. 全局 PostgreSQL 写压力上升，真实设备的新同步 result 写入、projection、device_info 更新会变慢甚至超时。
2. 如果真实设备自身已有遗留 `active_run_id`，后续手动同步会被 `active_run` 拦截，表现为“已有同步正在运行”或页面持续同步中。

抽查昨天 RCA 的真实设备：

```text
SN = 120200054822CHB0004
run dbb9f001-bde0-46eb-9f22-e88ce0fff225:
status = succeeded
request = succeeded
active_run_id = NULL
```

说明不是所有真实设备都坏；问题是全局堆积会让有遗留 active run 的真实设备，以及新发起同步的真实设备，和批量上线流量争抢同一套 DB/result/recovery 能力。

## 6. 具体原因分解

### 6.1 入口层：历史生命周期入口过多，目标只保留 Periodic 和 ModelUpload 后同步

周期同步未开启。历史代码在 `AutoSync.Enabled` 时会从 Bootstrap 路径启动 `bootstrap` run，DeviceOnline 也会独立启动 `device_online` run，ModelUpload/FirmwareChanged 还存在继续触发同步的调用路径。设备大批上线时，多条生命周期入口都具备制造 durable request/run 的能力；本轮实时聚合确认历史上至少存在 Bootstrap 和 DeviceOnline 两类 run，但尚未证明每个设备同一轮上线必然重复。

这两个入口的语义有先后关系：

```text
Bootstrap:
  设备刚进入 ACS，上报基础身份和启动事件；此时产品绑定、参数模型、在线状态等上下文可能还在建立中。

DeviceOnline:
  设备上线状态已经确认，系统侧设备记录、产品匹配、上下文更完整。
```

本次决策是将自动参数同步收敛到 Periodic 和 ModelUpload 流程终态后的同步：Bootstrap、DeviceOnline、FirmwareChanged 不再调用任何参数同步入口；ModelUpload 无论成功且设备支持，还是确认不支持，均调用 durable submitter，创建一次 `model_upload` request，不创建 legacy GPV task。DeviceOnline 的可靠投递、online epoch 和 outbox 不纳入本次参数同步方案；如其他业务需要可靠消费，另行治理。

这类 run 的来源不是 UI 周期扫描，而是设备生命周期事件：

```text
BOOTSTRAP / BOOT
  -> 只做入网、绑定、状态初始化和模型准备
  -> 不创建参数同步 request/task
DEVICE_ONLINE / MODEL_UPLOAD / FIRMWARE_CHANGED
  -> 只处理设备状态、模型和生命周期，不创建参数同步 request/task

PERIODIC（仅 enabled=true）/ MANUAL / CONFIG_PULL / LICENSE
  -> paramsync.Submit(...)
  -> parameter_sync_requests / runs
  -> device_tasks(source=param_sync)
  -> PARAM_SYNC result consumer / recovery
  -> parameter_sync_task_results
  -> run/request finalize
```

现场 `bootstrap waiting_device=4470`、`device_online waiting_device=1265`，说明历史入口流量主要来自上线事件；修复后 Bootstrap 存量 run 只做 recovery 收口，不再产生新 run。

### 6.2 执行层：设备任务结束，但官方 result 收口未结束

durable paramsync 设计上把两个计数分开：

```text
terminal_task_count:
  device_tasks 是否进入 completed / failed / expired / cancelled。

processed_task_count:
  这些终态 task 是否已经被 PGResultProcessor.Process() 处理，
  并写入 parameter_sync_task_results，推动 run/request finalize。
```

现场大量 run 的状态是：

```text
terminal_task_count = expected_task_count
processed_task_count = 0 或小于 expected_task_count
status = waiting_device
```

这说明南向任务不是一直没结束；真正卡住的是终态任务结果没有被 durable result processor 消化。前端看到 active run 是正确的，因为后端 request/run 确实没有 finalize。

换成更直观的说法：

```text
device_tasks:
  像每个子任务的“执行记录”。这里显示 completed / expired / cancelled，
  说明任务本身已经有结果，不是在设备侧一直跑。

parameter_sync_task_results:
  像 durable paramsync 的“正式回执登记簿”。只有登记到这里，
  processed_task_count 才会增加，run 才能判断所有子任务都已被官方处理。

active_run_id:
  像设备级同步锁。只要 run 没 finalize，这把锁就不释放。
```

所以“任务已经终态，durable result 处理/恢复追不上”的含义是：任务结果已经产生，但后台登记簿没有及时登记；登记不完成，系统不能安全地把整次同步标记为成功/失败，也就不会释放设备上的同步锁。

### 6.3 消费层：实时 result consumer 与 recovery 共用 DB 写瓶颈

实时链路：

```text
task terminal event
  -> PARAM_SYNC stream
  -> param-sync-results-pull consumer
  -> PGResultProcessor.Process()
  -> parameter_sync_task_results
  -> processed_task_count + finalize
```

补偿链路：

```text
parameter_sync_runs 中找 terminal_task_count 已满但 processed_task_count 未满的 run
  -> 查 device_tasks 中缺失 result 的终态 task
  -> 直接调用 PGResultProcessor.Process()
  -> 回补 parameter_sync_task_results
```

这两条链路都写同一组重表：

```text
parameter_sync_task_results
parameter_sync_staging_values
device_parameters
device_info
parameter_sync_runs / requests
```

本轮 PostgreSQL CPU 333%，64 个活跃连接中大量查询等待 `LWLock:LockManager`；同时 app 日志出现 `batch processor: worker channel full, dropping update`，NATS 日志出现 `slow consumer`。说明不是 result consumer 没启动：它仍在消费、无 redelivery，但设备在线/更新批处理和 result/recovery 共享主库锁与写吞吐，导致 result 消费速度低于生产速度。

### 6.4 recovery 层：20 秒总 deadline + 首个错误返回造成头阻塞

当前维护循环的一个问题是多个动作共用一个较短的维护窗口：

```text
SweepExpiredRequests
ReconcileStalledRequests
ReconcileStalledRuns
ReconcileCancellingRuns
ReconcileRunCounts
RecoverMissingResults
ReconcileTerminalBindings
CleanStaging
CollectMetrics
```

其中 `RecoverMissingResults` 按最早 `started_at` 的 stuck run 顺序处理。它在某个 run/result 上遇到 DB timeout 后会返回错误，本轮后续 run 不再继续处理。首轮现场日志反复出现同一类错误；本轮 root 复核新增确认的是 LockManager 等待、batch channel full 和 NATS slow consumer：

```text
recover missing parameter sync result ... context deadline exceeded
reconcile terminal parameter sync bindings: context deadline exceeded
clean parameter sync staging: context deadline exceeded
collect active parameter sync runs: context deadline exceeded
```

因此会形成头阻塞：

```text
早期批量设备 stuck run
  -> recovery 反复尝试
  -> DB timeout
  -> 本轮提前失败
  -> 后面的 run，包括真实设备 run，恢复机会变少
```

### 6.5 背压层：压测流量下缺少全局预算和公平恢复

当前设备上线/更新数据面与 paramsync result/recovery 共享统一主库和部分批处理资源。不能在长期修复中通过 SN 前缀、设备组或产品策略直接跳过大规模上线流量，否则无法验证系统在大规模上线场景下的稳定性。当前缺少的是面向压测场景仍然有效的全局背压和公平恢复能力：

```text
按 trigger_reason 独立并发预算
按 request/run 做全局创建速率预算
同设备 active run 强去重，并将 deduplicated request 与入口事件关联
result/recovery 在 DB 超时后跳过当前 run 并继续处理后续 run
设备无响应超时直接终态化，重试有明确预算
result/recovery 与设备在线批处理具备隔离或独立预算
DB 写入路径具备批处理能力
```

因此，当前版本中大规模上线制造的大量历史 `bootstrap` / `device_online` run 会占满全局资源，真实设备即使数量很少，也会被同一个 DB 锁/写路径和 active-run 语义影响。修复后应先限制新 run 创建、保护 result/recovery 预算并恢复公平收口；Bootstrap 只保留存量 run 的 recovery，不再作为新参数同步入口。

## 7. 历史修复草案（已被第 8 节覆盖，不作为当前实施依据）

> 本节保留原始评审过程和历史候选方案，仅用于审计上下文。当前实施以第 8 节为准；其中关于保留 DeviceOnline/FirmwareChanged 触发参数同步、或跳过不支持设备 ModelUpload 后同步的内容均已废弃。

### 7.0 统一同步架构约束（本次修订的硬要求）

本次不再保留“新 `parameter_sync` 不可用时回退旧 Path B”的过渡语义。所有参数同步入口必须先、且只能进入 `parameter_sync` request/run/result 生命周期；Path B 只保留为参数规划/映射语义，不再作为独立任务执行链路。

统一规则：

```text
手动同步、DeviceOnline、FirmwareChanged、周期同步、ModelUpload 完成后的同步、配置参数 Pull、License 参数同步
  -> parameter_sync.Submit(...)
  -> parameter_sync_runs / parameter_sync_requests
  -> parameter_sync dispatcher 创建 device_tasks(source=param_sync)
  -> PARAM_SYNC result consumer / recovery
  -> parameter_sync_task_results
  -> run/request finalize
```

参数同步入口矩阵：

| 入口 | 目标行为 | `trigger_reason` | 幂等键 |
|---|---|---|---|
| Bootstrap / BOOT | 只做入网和模型准备，不创建同步 request/task | — | — |
| DeviceOnline | 提交全量 durable sync | `device_online` | `device_id + online_epoch` |
| FirmwareChanged | 提交 durable sync；ModelUpload 仅更新模型，完成后再提交同步 | `firmware_changed` / `model_upload` | 事件 envelope ID + device ID |
| Periodic | 提交 durable sync | `periodic` | device + policy epoch + scan window |
| Manual | 提交全量或指定路径 durable sync | `manual` | 调用方 Idempotency-Key |
| Config Pull | 提交部分路径 durable sync | `config_pull` | 调用方 Idempotency-Key |
| License 参数同步 | 提交部分或全量 durable sync | `license` | 调用方 Idempotency-Key |

表中所有“提交 durable sync”的入口都必须创建 `device_tasks(source=param_sync)`；不存在直接创建普通 GPV task 的例外。SPV/MML 等单次 RPC readback 只有在业务语义升级为“参数同步”时才纳入此矩阵。

必须删除或禁止以下路径：

```text
StartPathBSync -> legacy sync-gpv / EnqueueGPVBatches
param_sync 未命中 canary -> legacy Path B
配置 Pull 在 durable submitter 未处理时 -> GPVBatcher / 单 task 兜底
```

`legacy_fallback_enabled` 不再参与任何参数同步决策。`parameter_sync` 未启用、result consumer 未启用或映射不可用时，入口返回明确的不可用/拒绝结果，不得静默创建旧 `device_tasks`。

本约束不影响 MML、SPV、故障自愈等“单次 RPC 操作”本身；但只要业务语义是“参数同步”，就必须使用上述 durable 生命周期。

统一入口切换必须先完成迁移前置条件：

```text
1. 盘点所有参数同步调用点，并为每个调用点定义 durable trigger_reason、失败码、HTTP/API 映射和幂等键；Bootstrap 明确登记为“禁止创建参数同步”的入口。
2. 新版本先停止创建 legacy sync-gpv 任务，再保留 legacy task 的只读观测、超时收口和结果兼容处理；禁止把旧任务重新转成新的自动 request。
3. 统计并排空切换前遗留的 legacy task；确认没有新的 legacy task 产生后，才删除 EnqueueGPVBatches 的同步用途。
4. `parameter_sync` 不可用、映射不可用或 result consumer 未启用时，入口必须返回明确失败；不能通过 `handled=false` 继续进入旧 Path B。
5. 回滚只能停止新的自动 request 创建，并继续收口已有 durable run；不得重新打开 legacy fallback 作为回滚手段。
```

### 7.1 P0：把“上线自动同步”和“周期同步”拆成可独立治理的入口

目标：明确不同自动入口的开关语义，同时保证每个入口实际提交到 `parameter_sync`。

修复建议：

```text
1. 在配置和 UI 中明确区分：
   - 周期参数同步：按时间扫描 stale 设备。
   - Bootstrap：只完成入网、绑定、状态初始化和模型准备，永远不提交参数同步 request。
   - DeviceOnline 自动同步：设备从 offline 恢复在线时触发，若开启则提交 `trigger_reason=device_online` 的 durable request；同一 `online_epoch` 只提交一次。
   - 固件变化同步：提交 `trigger_reason=firmware_changed` 的 durable request；ModelUpload 只负责更新映射，不得另起旧 Path B。

2. 入口开关必须在真正的调用点生效：
   - Bootstrap 无论配置如何都不创建 `parameter_sync_request` 或 legacy GPV task。
   - `sync_on_device_online=false` 时 DeviceOnline 不创建任何 `parameter_sync_request`。
   - `sync_on_firmware_change=false` 时 FirmwareChanged 只完成必要的设备/模型状态处理，不创建参数同步 request。
   - ModelUpload 完成后的自动同步必须继承触发来源并提交 `parameter_sync`，不得调用旧 `EnqueueGPVBatches`。

3. 配置上保留独立运行时开关：
   - sync_on_device_online
   - sync_on_firmware_change

4. 开关应支持运行时热更新，配置读取失败时 fail-closed（不创建自动同步），并记录配置读取失败指标。

5. 所有自动入口统一调用 `parameter_sync.Submit`，并将 `trigger_reason`、事件 ID、设备在线 epoch 和来源任务 ID 写入 request，禁止由 `SyncService` 直接创建普通 GPV task。
```

验收标准：

```text
周期同步关闭时，不影响 DeviceOnline 自动同步状态展示。
任意 Bootstrap / BOOT 事件都不创建 parameter_sync_requests，也不创建 legacy GPV task。
DeviceOnline 关闭后，新上线设备不再创建自动 parameter_sync_requests。
任一参数同步入口开启时，只能看到对应 `trigger_reason` 的 `parameter_sync_requests`，不存在同一业务入口产生的 legacy `device_tasks`。
手动、Config Pull、License、FirmwareChanged、ModelUpload 完成后的同步均可在 request/run/result 链路中追踪。
```

### 7.2 P0：增加全局背压、资源预算和入口幂等

目标：不跳过大规模上线压测流量，在 10000+ 台设备批量上线时保护 PostgreSQL/result processor，避免新 run 创建速度超过收口能力。

修复建议：

```text
1. 不按 SN 前缀、设备组或产品跳过上线自动同步。
2. 将 admission、幂等和 request 落库合并到同一个数据库事务；不能在 `Submit` 之前做应用层预检查：
   - 使用单独的 `parameter_sync_admission_state` 行并在事务内加锁，不能使用应用进程内计数器。
   - 全局预算不得由单一热点行串行化全部设备上线；按全局/trigger_reason 分片或使用数据库可恢复的 token/lease，并限制单次统计查询的范围和耗时。
   - 同一事务内完成：幂等键检查 -> 创建 request -> 获取 admission token/配额 -> active-run 去重 -> 写入最终 request 状态。
   - admission 未通过时也必须创建 request，状态为 `rejected`，结果码为 `AUTOMATIC_BACKPRESSURE`，并保留拒绝原因和当前 backlog 快照。
   - 按 `trigger_reason` 维护 active-run 配额、每分钟创建配额和 manual 保留配额。
   - `model_upload`、`device_online`、`firmware_changed`、`periodic` 均必须纳入 automatic admission；Bootstrap 不创建参数同步 request，不进入 admission。
   - backlog 超过阈值时只拒绝新的自动 request，已有 run、manual request 和 recovery 继续获得预算；预算必须同时约束 request/run 数、待处理 task 数和 DB 写入并发。
   - admission 的主判断使用数据库可重建的 durable backlog、未处理 task 数和 oldest stalled age；PG LockManager 等待和 PG CPU 作为带滞回的辅助熔断信号，不能依赖单次监控采样决定单个 request。
   - 熔断进入/退出必须有日志、指标和滞回窗口；拒绝结果落库为 `AUTOMATIC_BACKPRESSURE`，不能静默丢事件。
3. 同一设备存在 active run 时强制 deduplicate：
   - 不继续制造新的 running request。
   - 记录 `deduplicated_reason`、source event id、active run id 和目标 request id，方便复盘入口放大倍数。
4. 为每个入口建立稳定幂等键：
   - Bootstrap/FirmwareChanged/ModelUpload 使用事件 envelope ID + 设备 ID。
   - DeviceOnline 使用设备的 `online_epoch`；同一 epoch 的重复投递必须命中同一 request。
   - Periodic 使用 `device_id + policy_epoch + scan_window`。
   - Manual/Config Pull/License 继续使用调用方 Idempotency-Key。
5. admission、幂等键和 deduplicate 结果都必须在 `parameter_sync_requests` 中可查询，不能只写日志或 Redis。
6. 增加迁移字段和索引：
   - `source_event_id`、`online_epoch`、`deduplicated_reason`、`deduplicated_to_request_id`、`admission_reason`。
   - 对 `(caller_type, idempotency_key)`、`(device_id, online_epoch)` 和 `source_event_id` 建立唯一/查询索引；允许同一事件的重放只返回原 request。
   - `AUTOMATIC_BACKPRESSURE` 增加到结果码常量、API 类型和前端展示契约。
```

验收标准：

```text
10000+ 台设备批量上线时，parameter_sync_requests running 不会超过配置预算，且 admission 决策与 request 落库在同一事务内完成。
同一设备同一轮上线最多存在一个 active run。
在目标压测负载下，`terminal_task_count - processed_task_count` 连续 15 分钟不增长且 backlog 呈下降趋势。
DB/result backlog 到达阈值后，新自动 run 创建会降速，已有 run 能继续 finalize；熔断进入/退出至少保持一个配置的滞回窗口。
熔断期间 manual request 仍有独立配额，真实设备不会被自动流量完全饿死。
同一事件重复投递只产生一个 durable request，且可通过 request 查询原始事件 ID。
```

### 7.3 P0：隔离 result/recovery，修复 recovery 头阻塞

目标：单个坏 run/result、设备在线批处理或 DB 锁等待不能阻塞整批 result/recovery；设备超时重试作为独立预算治理。

修复建议：

```text
1. RecoverMissingResults 处理单个 run/task 失败时：
   - 记录错误和 run_id。
   - 跳过该 run，继续处理后续 run。
   - 本轮结束后汇总错误。
   - 在 `parameter_sync_recovery_state` 中记录 `next_retry_at`、attempts、last_error、lease_token、lease_until；未到 retry 时间或已被其它 worker claim 的 run 不重复撞击。
   - recovery state 的 claim、失败退避和 lease 释放必须在数据库事务内完成；lease 超时后允许其它 worker 接管。
   - 单 task/result 的数据库错误不得让 `RecoverMissingResults` 提前返回并终止整轮；必须返回逐项结果，汇总错误只用于告警。

2. result consumer、recovery、设备在线批处理使用独立并发/队列/DB budget；至少要能单独限制：
   - result consumer pull concurrency
   - recovery task budget
   - device online batch flush concurrency
   - 每类操作的最大 DB connection/transaction budget
   - result processor 与 recovery 不得共享同一个无上限的写入信号量。
   - 独立连接池只能避免应用侧连接饥饿，不能隔离 PostgreSQL 的 CPU/锁竞争；必须同时设置每类操作的并发上限、statement timeout 和单事务耗时上限。

3. recovery 排序从单纯 started_at ASC 改为：
   - manual / bootstrap / device_online / firmware_changed / periodic 按预算轮转
   - 同一 trigger_reason 内按 started_at 分批处理
   - 本轮 DB 处理失败的 run 只跳过本轮，不阻塞其它 run

4. 设备超时重试单独限额：
   - GPV task 超时无响应后进入 expired 终态并写 durable result。
   - 不得无限追加同一设备、同一 run 的容错 GPV task。
   - retry_count、fault recovery count 纳入背压指标。
```

验收标准：

```text
构造一个设备无响应超时 task，系统登记 expired result，且不突破该入口 retry budget。
构造一个 recover 会因 DB 超时/失败的 run，不影响后续 100 个 healthy run 被回补。
healthy run 不会被大量早期 stuck run 长时间挡住。
设备在线批处理压力升高时，result consumer 仍有独立处理配额。
同一 recovery run 连续失败时，重试间隔按退避增长，且不会阻塞其它 trigger_reason。
并发启动两个 recovery worker 时，同一个 run 只能被一个 lease 持有；worker 崩溃后 lease 超时可被接管。
```

### 7.4 P0：拆分 maintenance deadline 与失败隔离

目标：一个维护动作超时不拖垮其它动作；这是恢复链路可用性的前置条件，不等待吞吐优化阶段。

修复建议：

```text
1. Sweep / stalled reconcile / run count / missing result recovery /
   terminal binding / clean staging / collect metrics 使用独立 context timeout。

2. missing result recovery 单独配置：
   - recovery_interval
   - recovery_run_limit
   - recovery_task_limit_per_run
   - recovery_task_budget
   - recovery_timeout

3. metrics 采集失败不影响 recovery；clean staging 失败不影响 result recovery。
4. 每个 maintenance 动作必须独立记录耗时、成功数、失败数和超时数；不能只依赖总 maintenance error。
```

验收标准：

```text
CleanStaging 超时时，RecoverMissingResults 仍能完成。
CollectMetrics 超时时，不影响 active_run 收口。
```

每个 maintenance 动作都必须独立设置 timeout，并输出成功数、失败数、超时数、耗时 p50/p95；总 maintenance error 只能作为汇总告警，不能作为唯一诊断依据。

### 7.5 P1：为 result/recovery 增加公平调度和触发来源预算

目标：在压测流量持续存在时，result/recovery 不被少数早期 stuck run 或单一 trigger_reason 长时间占满。

修复建议：

```text
1. recovery 查询按 trigger_reason、started_at 做轮转处理，不允许单一来源长期占满 `runLimit`；轮转顺序和每类预算必须可配置。
2. manual trigger 保持独立预算，避免被 auto_sync 压测流量完全淹没。
3. bootstrap / device_online / firmware_changed / periodic 各自有 recovery 批次预算。
4. 本轮 DB 处理失败的 run 只跳过本轮，不占用后续 run 的恢复窗口。
5. 增加 oldest stalled age、per-trigger recovery lag、healthy run wait time 指标。
```

验收标准：

```text
10000 台设备同时上线时，healthy run 不会被早期 stuck run 长时间挡住。
manual / device_online / firmware_changed 都能按预算获得 recovery 机会。
manual request 在自动流量持续时仍能在预算窗口内完成。
```

### 7.6 P1：减少 result processor 的 DB 写放大并隔离设备在线写入

目标：降低每个 result 对 DB 的写压力。

修复建议：

```text
1. 对 parameter_sync_task_results 做批量 upsert 或批量回补。
2. staging_values 可按 run/task 批量落库，避免逐 result 小事务。
3. device_parameters 的 delete/insert 策略改为按差异 upsert，减少大批 DELETE。
4. projection 到 device_info 与 run finalize 解耦，避免 result 收口被派生字段刷新拖慢。
5. 将设备在线/批量设备信息更新与 result/recovery 使用不同 worker pool、连接池或限流 budget，避免 `devices/device_info` 批量写入挤占 result 收口。
6. 对 `parameter_sync_task_results`、staging、run progress 的写入增加批量大小上限和单事务耗时上限，超限拆批，不允许单大事务长期持锁。
7. `batch processor` 和 NATS consumer 禁止静默丢弃事件：通道满时必须阻塞生产者、写入 durable overflow/outbox，或进入可重放的补偿表；增加 dropped、replayed、reconcile_found 指标。
8. PARAM_SYNC 消息达到最大投递次数后的 `Term` 必须进入可查询的 durable DLQ/overflow，并关联 `event_id`、`run_id`、`task_id`；不得以 Term 作为无记录的最终处置。数据库 recovery 发现对应 task 缺少 result 时，必须能把该消息标记为已回补。
```

验收标准：

```text
同等 10000 台设备压力下，Postgres CPU、LockManager 等待和 context deadline exceeded 相比修复前基线下降，并在连续 15 分钟稳定窗口内不再增长。
`terminal_task_count - processed_task_count` 的最大值和 oldest stalled age 均持续下降；目标值必须在压测前固定并写入测试记录。
PARAM_SYNC pending 在连续 15 分钟稳定窗口内持续下降，不能依靠不断增长的数据库/JetStream 存量“假稳定”。
事件 dropped 数为 0；发生 overflow 或 Term 时，最终 replay/reconcile 数量与恢复成功数量一致；所有 Term 都有对应的 durable 处置记录。
```

### 7.7 修复方案如何解决“没有及时登记”

这里的“登记”指把已经终态的 `device_tasks` 转成 durable paramsync 官方结果：

```text
device_tasks.completed / expired / cancelled
  -> parameter_sync_task_results 写入
  -> processed_task_count 增加
  -> run/request finalize
  -> active_run_id 清空
```

当前问题不是设备侧任务一直没有完成，而是这个登记链路追不上。完整修复方案按三层解决：

```text
第一层：减少需要登记的回执数量
  - 通过全局背压、backlog 熔断和 trigger_reason 预算控制新 run 创建速率。
  - 不按 SN 前缀、设备组或产品跳过压测流量。
  - 同一设备已有 active_run 时只 deduplicate，不继续制造新 request，并记录事件幂等键。
  - Bootstrap 不再触发参数同步；Bootstrap 历史遗留 run 只由 recovery 收口，不产生新的自动 request。

第二层：避免坏 run 阻塞登记队列
  - 超时无响应 task 直接登记 expired 结果并参与 run 收口；重试由独立预算限制，不把删除重试作为本轮主因。
  - RecoverMissingResults 遇到单个 run/result 超时后跳过，继续处理后续 run。
  - recovery 按 trigger_reason 和 started_at 分批轮转，避免单一来源长期占满恢复窗口。
  - result consumer、recovery 和设备在线批处理使用独立并发/队列/DB budget。

第三层：提升登记吞吐并降低 DB 写放大
  - 拆分 maintenance deadline，避免 clean staging / metrics 超时拖垮 result recovery。
  - parameter_sync_task_results 批量 upsert。
  - parameter_sync_staging_values 批量写入。
  - device_parameters 从大批 DELETE/INSERT 改为差异 upsert。
  - device_info projection 与 run finalize 解耦，但必须在 durable result/run 状态一致后释放 active_run。
  - 设备在线/批量设备信息更新与 result/recovery 做 worker pool、连接池或限流隔离。
```

预期效果：

```text
processed_task_count 能持续推进
stalled_finalizing 数量下降
active_run_id 能及时清空
真实设备手动同步不再被旧 run 长时间阻塞
大规模上线压测流量不被跳过，系统稳定性可以被真实压测验证
```

首轮验收必须同时验证以下可量化 SLO；每项必须在压测开始前写成具体数值阈值，不能只写“下降”“不增长”或事后调整：

```text
1. healthy run recovery lag、manual request p95 完成时间、oldest stalled age。
2. terminal-result backlog 的峰值、下降斜率和持续时间。
3. automatic admission 拒绝数、deduplicate 数、按 trigger_reason 的 run 创建速率。
4. PostgreSQL CPU、LockManager 等待、连接池等待、statement timeout。
5. result consumer processed/error/redelivery、recovery claim conflict/lease takeover。
6. batch/NATS dropped、overflow、replay 和数据库补偿恢复数量。
```

### 7.8 存量积压处置与切换顺序

方案不能只处理新流量，必须先安全收口当前已存在的 Bootstrap/DeviceOnline stalled run：

```text
1. 发布入口封口版本：Bootstrap 立即停止创建同步；DeviceOnline、FirmwareChanged、周期同步等自动入口先进入 admission；所有入口禁止 legacy fallback。
2. 记录切换基线：active run、terminal-result backlog、oldest stalled age、NATS pending、Term/DLQ、recovery error 和 manual p95。
3. 启动带 lease、退避和公平轮转的 recovery，先验证 healthy run 与 manual run 能获得独立预算。
4. 对缺失 result 的终态 task 执行 durable 回补；禁止直接把 run/request 强行改成 succeeded 或清空 active_run_id。
5. 确认 legacy `device_tasks` 不再新增、overflow/DLQ 均可回放后，再逐步恢复 DeviceOnline 自动同步。
6. 回滚只停止新的自动 request；已有 durable run 继续由 result/recovery 收口，不重新打开 Bootstrap 或 legacy Path B。
```

### 7.9 手动同步的离线语义

手动同步必须与自动同步的 admission 和 active-run 规则一致，并明确 API 契约：

```text
设备离线：不创建南向 GPV task；若产品需要“上线后执行”，返回可查询的 queued request；若不支持离线排队，则明确返回 503 + DEVICE_OFFLINE。
设备在线：创建 manual parameter_sync request，并保留独立 manual budget。
```

禁止仅通过 `is_online` 应用层预检改变语义而不更新 API、前端提示和 durable request 状态；离线拒绝/排队必须有测试和运行指标。

### 7.10 迁移与实施顺序

```text
P0-1 统一入口：所有参数同步入口只提交 parameter_sync；关闭 legacy fallback；Bootstrap 不创建参数同步 request。
P0-2 入口治理：补齐 DeviceOnline/FirmwareChanged/周期同步等实际开关判断、稳定幂等键和数据库 deduplicate 记录。
P0-3 收口恢复：recovery 单项失败跳过并退避，按 trigger_reason 公平轮转，result/recovery/在线批处理落实独立预算。
P0-4 维护隔离：拆分各 maintenance context 和指标，清理/metrics 失败不得影响 result recovery。
P1-1 吞吐优化：批量写入、projection 与 finalize 解耦、独立连接池或信号量隔离。
```

每个阶段都必须支持回滚到“停止自动入口、保留已有 `parameter_sync` run 收口”的安全状态；禁止通过重新打开 legacy Path B 作为回滚方式。

最终必须满足：

```text
所有参数同步 request/run/result 均可在 parameter_sync 表和指标中追踪。
所有 Path B 入口不再直接调用 legacy sync-gpv / EnqueueGPVBatches。
parameter_sync 未启用或不可用时明确失败，不静默降级。
自动流量受全局与 trigger_reason 预算约束，manual 有保留配额。
单个坏 run 不阻塞其它 run，且失败项具备可观测退避状态。
```

## 8. 当前生效的最终修复方案

### 8.0 术语边界与复用原则

本方案中的“删除 Path B”只表示删除旧的参数同步业务入口和 legacy `sync-gpv` 任务物化职责，不表示删除 GPV 路径选择、参数映射或批次划分算法。

当前代码的语义边界如下：

```text
StartPathBSync
  = 历史命名的参数同步启动入口/适配器。
  当前实现先调用 durable parameter_sync，handled=false 时再回退 legacy Path B。
  最终版本不再保留这个业务入口；所有参数同步入口直接调用 DurableParamSyncSubmitter。

EnqueueGPVBatches
  = 将已经得到的参数路径拆成 GetParameterValues 批次并创建 device_tasks。
  不是 XML/数据解析器，也不是参数值收口器。
  最终版本不允许它直接创建 legacy sync-gpv task，但其拆批算法必须被抽取为
  parameter_sync Planner 的内部/共享能力。

parameter_sync Planner/Dispatcher
  = 负责映射选择、storable path 计算、GPV 批次规划，以及创建
    device_tasks(source=param_sync) 和 parameter_sync_outbox。
```

当前 durable Planner 已复用 `PathBStorablePrefixes`、`PathBStorablePrefixesForStandardPaths` 和 `PathBGPVBatches` 的算法；迁移时应把这些算法移动到中性命名的参数同步规划包，或明确由 Planner 包提供，不能通过删除 `EnqueueGPVBatches` 误删批处理能力。

### 8.1 入口收敛

本次最终入口策略如下：

| 入口 | 修复后行为 | 是否创建参数同步 request/run |
|---|---|---:|
| Bootstrap / BOOT | 只做入网、绑定、状态初始化和模型准备 | 否 |
| DeviceOnline | 仅保留设备状态事件，供其他消费者使用 | 否 |
| ModelUpload 成功且设备支持 | 解析模型、计算映射、更新发现状态，并追加一次最新参数同步 | 是，`model_upload` |
| ModelUpload 不支持 | 记录 `not_supported`，使用现有模型/映射完成流程，并追加一次参数同步 | 是，`model_upload` |
| FirmwareChanged | 可做模型或设备元数据刷新，但不做参数同步 | 否 |
| Periodic | 仅在定时任务开启时提交 durable sync | 是 |
| Manual | 用户主动触发 durable sync | 是 |
| Config Pull / License | 按现有业务需要提交 durable sync，不得 fallback | 是 |

DeviceOnline 技术上可以通过 outbox、online epoch、可靠发布和补偿扫描修复可靠性，但本方案不再将其作为参数同步入口。这样可以避免把设备状态边沿可靠性问题继续耦合到参数同步容量问题；如果其他业务需要可靠消费，另行建设 DeviceOnline outbox。

ModelUpload 是完成后的同步入口：上传请求本身不直接触发同步，只有 ModelUpload 流程进入终态后才提交一次 `model_upload` durable request。支持设备在模型文件成功接收、解析和映射完成后提交；不支持设备在能力判定完成、继续使用现有模型/映射后同样提交。request 必须携带 `model_upload_status=uploaded|not_supported`，并使用 `device_id + model_version/model_hash + upload_task_id` 幂等，不能因为完成事件重复投递而重复同步。

#### 8.1.1 全局 admission、配额与 ModelUpload 例外

全局 admission 必须进入最终实施顺序，不能只停留在历史草案。`TriggerModelUpload` 必须纳入自动流量预算，但不能被 `periodicSyncEnabled=false` 静默丢弃。

规则固定为：

```text
1. ModelUpload 终态必须先创建一条 durable trigger intent/request，确保“只触发一次”可查询。
2. 全局 backlog、未处理 result、oldest stalled age 或 DB 熔断时，ModelUpload request 进入 queued/admission_waiting，
   由 durable dispatcher 延迟执行；不能因为 periodic 开关关闭而拒绝，也不能回退 legacy GPV。
3. Periodic 允许在 admission 熔断时被跳过或 rejected；Manual 保留独立执行配额。
4. admission 决策、request 创建、幂等检查和 deduplicate 结果必须在同一数据库事务内完成。
5. admission 至少维护 global、model_upload、periodic、manual 四类预算；预算同时约束 active run、待处理 task、
   missing result 和 result/recovery DB 并发。
6. 熔断只限制新自动执行，不影响已有 run 收口、历史 recovery 和 Manual 保留配额；进入/退出使用滞回窗口。
```

建议新增/扩展 durable 字段：`source_event_id`、`model_upload_status`、`admission_status`、`admission_reason`、`backlog_snapshot`、`deduplicated_to_request_id`，并为 ModelUpload 幂等键建立唯一约束。`TriggerReason.Automatic()` 必须同步纳入 `model_upload`，但“自动”不等于“受周期同步开关控制”。

#### 8.1.2 ModelUpload 终态状态机与一次性触发

ModelUpload 必须以 durable intent 驱动，不能由多个事件消费者各自猜测是否应该同步：

```text
requested
  -> uploaded       （文件接收、解析、映射交集和 discovery 状态提交成功）
  -> not_supported  （明确的能力不支持/产品禁用，且现有模型或映射可用）
  -> failed         （MinIO、解析、映射、数据库或其它处理失败）

uploaded/not_supported
  -> 唯一幂等键：device_id + model_version/model_hash + upload_task_id
  -> parameter_sync_requests(trigger_reason=model_upload)
  -> queued/admission_waiting/running
```

只有 `uploaded` 和 `not_supported` 触发参数同步；一般处理失败不能误判为 `not_supported`。终态写入、intent/outbox 写入和 provisioning task 的完成/失败必须定义事务边界。若 durable submit 暂时失败，intent 保留 `next_attempt_at/attempts/last_error`，由 worker 重试；不得把 provisioning task 长期留在 `discovering`，也不得转入 Path B。

当前 `enable_filetype11=false` 分支虽然更新了 discovery log，但返回对象仍可能保持 `discovering`；`HandleUploadFailed` 也没有形成实际调用链。实施前必须补齐能力判定事件、上传任务身份、event envelope ID 传递和错误重试/DLQ。

### 8.2 Path B 清理

以下调用和降级路径全部删除，不再由任何入口触发：

- Bootstrap、DeviceOnline、FirmwareChanged 的参数同步调用，以及 ModelUpload 未完成场景的同步调用；ModelUpload `not_supported` 也必须走新的 `parameter_sync`，不得走旧同步调用；
- 作为业务入口的 `StartPathBSync`；
- 直接创建 legacy `sync-gpv` task 的 `EnqueueGPVBatches` 调用；其批次算法须先抽取并由 durable Planner 复用；
- legacy `ParamSyncStarter` 实现及 provider fallback；设备层窄接口可以保留，但实现必须直接调用 durable submitter；
- PeriodicSyncer 对 Path B 的调用；
- ManualSync 对 Path B 的调用；
- durable submit 失败时的 `handled=false`、legacy `sync-gpv` fallback。

参数同步只允许使用：

```text
Periodic / Manual / ModelUpload-terminal / Config Pull / License
  -> DurableParamSyncSubmitter
  -> parameter_sync_requests
  -> parameter_sync_runs
  -> device_tasks(source=param_sync)
  -> NATS result consumer / recovery
  -> parameter_sync_task_results
  -> finalize
```

durable 功能未启用、映射不可用或消费者未启用时，返回明确的 `unavailable` 或 `rejected`，不能静默创建旧任务。MML、SPV 等单次 RPC readback 不属于本次 Path B 清理范围。

### 8.3 周期和手动同步语义

- `periodicSyncEnabled=false`：不创建新的周期 request/run/task，不做后台隐式重试；Manual 和 ModelUpload 流程结束后的同步仍可触发。
- 重新开启定时同步后，从下一个调度周期开始提交 request。
- 已创建的 durable run 继续执行和恢复，不因开关关闭被强制取消。
- Manual 不受周期同步开关影响，必须直接走 durable submitter，并返回 request/run ID。
- 同一周期重试使用 `device_id + policy_epoch + scan_window` 幂等键。
- 不为每次关闭状态的扫描创建大量 rejected request；可使用 skipped 计数和指标记录调度跳过。

### 8.4 ModelUpload 和 FirmwareChanged 状态闭环

删除旧的同步调用后必须补齐 ModelUpload 的任务生命周期，并增加流程终态后的 durable 同步：

- 模型上传成功：完成模型解析、映射和 discovery/provisioning task；
- 设备支持 ModelUpload：在上述状态提交一次 `model_upload` durable 参数同步，`model_upload_status=uploaded`；
- 设备不支持 ModelUpload：记录 `not_supported`，使用现有模型/映射完成模型发现流程，并提交一次 `model_upload` durable 参数同步，`model_upload_status=not_supported`；
- 两种状态的提交失败都必须进入可重试的 durable intent/outbox，不得静默丢失或转入 Path B；
- 模型上传失败：进入明确失败状态，保留错误原因；
- 不得因为移除 Path B 或设备能力不支持导致任务长期停在 `discovering`；
- FirmwareChanged 只保留必要的模型/设备元数据处理，不写入 `syncreason`，不创建 paramsync request。

需要重点检查 [engine.go](</Users/wangyong/OBJECT/Codex/xomc/omcgo/internal/provision/engine.go>) 和 [model_upload.go](</Users/wangyong/OBJECT/Codex/xomc/omcgo/internal/provision/model_upload.go>) 中上传成功后的 task completion。

### 8.5 Result consumer 正常修复

当前 NATS pull 并发与实际处理 worker 数量不匹配，需按现有架构做常规修复：

- 将 result worker/shard 数量配置化；
- NATS 拉取并发与业务处理并发分离；
- 同一设备保持顺序，不同设备允许并行；
- 队列满时阻塞或进入 durable overflow，不得丢弃；
- 增加 active workers、queue depth、processing latency、processed/error 指标；
- result consumer 与 recovery 使用独立预算，避免互相饿死。

不建议直接取消设备级顺序控制，否则可能造成同一设备参数结果乱序。

### 8.6 NATS 对应处理

维持 JetStream 至少一次投递语义：

- 正常处理后 ACK；
- 临时错误 NAK 并重试；
- 达到最大投递次数的消息写入 durable failure/DLQ 后再 Term；
- 通过 `event_id` 和 `(run_id, task_id)` 幂等；
- 队列满、slow consumer、channel full 不得静默丢消息；
- 增加 delivered、acked、nacked、terminated、replayed、recovered、queue-full 指标。

必须增加专用的 durable PARAM_SYNC DLQ/失败记录。可以复用现有 `dead_letters` 存储和管理能力，但不能只把原始 payload 当作唯一索引；必须保留稳定的事件、run 和 task 关联：

```text
parameter_sync_event_failures
  unique(subject, event_id)
  event_id, subject, device_id/device_sn
  run_id, request_id, task_id
  raw_payload, delivery_count, last_error
  status(pending/replayed/recovered/manual_review)
  next_retry_at, replayed_at, recovered_at, created_at, updated_at
```

处理规则固定为：

1. 达到 JetStream 最大投递次数后，先在数据库写入上述失败记录，再 `Term`；写入失败时只能 `NAK` 重试并告警，不能静默 `Term`。
2. Replay worker 按 `next_retry_at` 重发原事件，使用 `event_id` 和 `(run_id, task_id)` 幂等；重发成功只标记 `replayed`，不删除原始失败记录。
3. Recovery 交叉检查 `device_tasks` 终态、`parameter_sync_task_results` 以及 `(run_id, task_id)` 唯一键；已确认可补偿的记录标记 `recovered`，无法安全推断的记录进入 `manual_review`。
4. `Term`、DLQ、重放和补偿均必须有计数、年龄和失败原因指标，并提供按 `event_id/run_id/task_id` 查询和人工重放入口。

当前通用 `dead_letters` 表若继续复用，必须通过迁移增加上述关联字段/索引，或者由 `parameter_sync_event_failures` 作为参数同步专用投影；不能依赖 payload 字节搜索来做恢复。

### 8.6.1 Recovery migration、claim 与 lease

Recovery 不能只在代码中增加重试字段，必须先建立可迁移、可抢占、可回收的数据库状态模型：

```text
parameter_sync_recovery_state
  unique(run_id, task_id)
  status(pending/processing/processed/failed)
  attempts, next_retry_at, last_error
  lease_token, lease_until, claimed_at, processed_at
  created_at, updated_at

parameter_sync_task_results.status
  received -> processing -> processed
                         -> failed
```

迁移与运行规则：

1. 先发布向后兼容的表、索引和状态约束，再发布能读写新状态的 consumer/recovery；旧 `received/processed/failed` 数据必须可双读，禁止在同一发布中直接收紧旧约束。
2. claim 只持有短事务：使用 `FOR UPDATE SKIP LOCKED` 选择到期的 `pending` 或 lease 已过期的 `processing` 行，写入新的 `lease_token`、`lease_until`、`attempts` 后提交；不得在完整 result 处理期间持有数据库行锁。
3. 完成写入必须校验 `lease_token`；lease 丢失时只能做幂等查询/补偿，不能覆盖新 owner 的状态。lease 到期可被其他 worker 接管，但必须增加 `lease_takeover` 指标。
4. 单个 run/task 的 DB timeout、写入失败或下游异常记录 `last_error`，按指数退避更新 `next_retry_at` 后继续处理其他候选；达到 attempts 上限进入 `failed/manual_review`，不能中断整批。
5. consumer 与 Recovery 共享 `(run_id, task_id)` 幂等边界：stale `received`、stale `processing` 和缺失 result 都必须可安全重跑，成功后只允许一个 `processed` 结果推进 run 收口。
6. migration 必须覆盖索引、旧数据回填/兼容读取、lease 过期接管、重复消费、worker 崩溃和回滚；发布前后分别验证 claim 冲突、最老 backlog 年龄和 terminal/processed 差值。

### 8.7 Recovery 和 `received` 状态

当前 Recovery 存在“首个错误中断整批”和“遗漏 received result”两个缺陷，修复如下：

- 用事务和 `FOR UPDATE SKIP LOCKED` 抢占 run/task；
- 增加 `attempts`、`next_retry_at`、`last_error`、`lease_until`；
- 单个 run/task 失败后记录错误、退避并继续下一个；
- 同时扫描缺失 result、超时 `received` 和失效 `processing`；
- `received` 不能当作已处理，只有 `processed` 才能判定重复；
- 处理成功、写 result、更新进度必须幂等；
- 不得通过直接清空 active_run 或强制改成功来伪造收口。

建议状态语义固定为：

```text
received -> processing -> processed
                       -> failed
```

涉及 [reconciler.go](</Users/wangyong/OBJECT/Codex/xomc/omcgo/internal/paramsync/reconciler.go>) 和 [result_consumer.go](</Users/wangyong/OBJECT/Codex/xomc/omcgo/internal/paramsync/result_consumer.go>)。

### 8.8 其他缺陷修复清单

1. **状态接口不一致**：以 durable `parameter_sync_runs` 为唯一状态源，修复 active 接口返回 stalled 而旧 sync-status 返回 idle 的问题。
2. **配置语义不一致**：废弃或忽略 `sync_on_bootstrap`、`sync_on_device_online`、`sync_on_firmware_change` 对参数同步的影响，Periodic 成为唯一自动开关。
3. **Manual fallback**：Manual、Config Pull、License 在 durable 不可用时明确失败，禁止调用 legacy service。
4. **历史 backlog**：只恢复已经存在的 Bootstrap/DeviceOnline 历史 run，不重新启用这些入口。
5. **索引不足**：检查并补充 runs 的状态/重试索引、results 的状态/更新时间索引、`(run_id, task_id)` 唯一索引，并使用 EXPLAIN 验证。
6. **事件丢失**：`batch processor: worker channel full` 和 NATS slow consumer 必须进入 durable overflow、重放或数据库补偿，不能只调大 channel。
7. **资源竞争**：result、recovery、设备在线批处理分别设置并发、连接、事务和 statement timeout 预算。
8. **可观测性不足**：增加 oldest stalled age、missing result、received stale、recovery claim conflict、lease takeover、DLQ、replay、active worker 和 queue depth 指标。
9. **第 9 项暂不考虑**：不新增跨触发业务去重模型；仅保留现有 `event_id` 和 `(run_id, task_id)` 基础幂等约束。

### 8.9 实施顺序和回滚

```text
P0-1 立即封口：停止 Bootstrap/DeviceOnline/FirmwareChanged 新参数同步；ModelUpload 仅在流程终态提交一次 `parameter_sync`，支持与不支持均执行。
P0-2 兼容迁移：先发布全局 admission、ModelUpload intent/终态、durable DLQ、Recovery state/lease 的表结构、索引和双读/双写兼容代码；此阶段不删除旧符号。
P0-3 新链路就绪：Durable Planner 复用抽取后的 GPV 路径选择/批次算法；ModelUpload 终态、Manual、Config Pull、License、Periodic 均直接提交 durable request，并完成 request/run/task/result 的端到端验收。
P0-4 原子封口：停止所有新参数同步业务对 `StartPathBSync` 的调用，关闭 `handled=false` 和 provider fallback；ModelUpload `uploaded/not_supported` 只通过 durable intent 触发。
P0-5 修复收口：启用 Recovery 的 claim/lease/退避/公平轮转、PARAM_SYNC durable DLQ 和失败隔离；确认新增事件不会被无记录 `Term`。
P1-1 排空存量：冻结旧入口新任务并记录切换基线，按 manual/历史 run/来源预算逐项恢复 backlog；只有在 admission、DB、recovery 和 oldest-age 门禁满足后继续。
P1-2 删除旧实现：确认无业务入口调用 `StartPathBSync`、无 legacy `EnqueueGPVBatches` task materialization、无 `handled=false` fallback 后，删除旧入口/adapter；保留抽取后的 Planner 批次算法。
P1-3 统一状态：修复 API、前端和数据库状态不一致，并将 durable runs 作为唯一状态源。
P1-4 按门禁开启 Periodic；不满足容量和恢复 SLO 时保持关闭。
```

回滚只允许停止新的自动 request，已有 durable run 继续恢复；不得重新打开 Bootstrap、DeviceOnline 或 legacy Path B 作为回滚方案。

### 8.10 验收标准

```text
1. Bootstrap、DeviceOnline、FirmwareChanged 均不产生 paramsync request/run/task；ModelUpload 无论 `uploaded` 还是 `not_supported`，流程终态均只产生一次 `model_upload` paramsync request。
2. Periodic 关闭时不产生新的周期同步；Manual 和 ModelUpload 流程结束后的同步仍可独立创建 durable request。
3. 全仓库所有参数同步业务语义均进入 `parameter_sync`，业务入口不再调用 `StartPathBSync`，也不再通过 `EnqueueGPVBatches` 直接创建 legacy `sync-gpv` task；GPV 路径选择和批次规划算法仍由 durable Planner 复用。
4. durable 不可用时返回明确 unavailable/rejected，不静默降级。
5. NATS 队列满、NAK、Term、重放均有 durable 记录，不丢失事件。
6. received 状态可恢复；单个坏 run 不阻塞其他 run。
7. terminal_task_count - processed_task_count 持续下降，oldest stalled age 不再增长。
8. API、数据库和前端显示一致，不再出现 active/stalled 与 idle 冲突。
9. 历史 backlog 可以逐项恢复、失败收敛或进入人工处理，不能直接清空 active_run。
10. 通过静态搜索和自动化测试确认不存在旧参数同步入口和 legacy task materializer 的直接调用；允许 durable Planner 复用抽取后的 GPV 批处理算法，现有单次 MML/SPV readback 不受影响。
11. admission 决策、ModelUpload 幂等检查和 request/intent 创建在同一事务完成；global、model_upload、periodic、manual 四类预算均可观测，ModelUpload 不受 `periodicSyncEnabled` 阻断，Periodic 在熔断时只进入 queued/skipped/rejected 并可恢复。
12. Recovery migration 可在旧数据上双读运行；重复 claim、lease 过期接管、worker 崩溃、重复 result 和单项 DB timeout 均有自动化测试，单项失败不会阻塞其他 run/task。
13. 发布顺序满足“迁移兼容代码 -> durable 新链路 -> 原子关闭旧入口 -> 预算内排空存量 -> 删除旧实现”；任何阶段回滚都不重新打开 Bootstrap、DeviceOnline 或 legacy Path B。
```
