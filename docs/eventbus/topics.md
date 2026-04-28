# EventBus 主题（Subjects）登记表

> 本文件登记 OMC 系统中所有 EventBus 主题（NATS JetStream / ChannelEventBus）。
> 所有主题常量定义在 `omcgo/internal/core/event/subjects.go`。
> 命名规范遵循 CLAUDE.md §5.4：点分层级 `domain.action.detail`。

## 1. 总览

| 类别 | 主题前缀 | 用途 | 实现 |
|------|---------|------|------|
| Device | `device.*` | 设备生命周期、Inform 事件 | NATS JetStream |
| Command Response | `command.*.response` | RPC 响应回调 | NATS JetStream |
| Alarm | `alarm.*` | 告警生命周期 | NATS JetStream |
| Task | `task.*` | 统一任务队列终态 | NATS JetStream |
| PM | `pm.*` | 性能数据文件流转 | NATS JetStream |
| MR | `mr.*` | 测量报告文件流转 | NATS JetStream |
| Provision | `provision.*` | 自动开站任务 | NATS JetStream |
| DataModel | `datamodel.*` | 参数模型上传/解析 | NATS JetStream |
| Software | `firmware.*` / `upgrade.*` | 固件升级 | NATS JetStream |
| Backup | `backup.*` | 配置备份 | NATS JetStream |
| Report | `report.*` | 报表生成 | NATS JetStream |
| OSS / Northbound | `oss.*` | 北向接口推送 | NATS JetStream |
| NEDirect | `nedirect.*` | 网元直连 | NATS JetStream |
| System | `sys.*` | 系统级控制（Casbin 重载） | NATS JetStream |

## 2. 关键主题详细登记（W3.E.2 焦点 ≥ 3 类）

### 2.1 Alarm（告警生命周期）

| Subject | 常量 | Publisher | Consumer | Payload |
|---------|------|-----------|----------|---------|
| `alarm.raised` | `SubjectAlarmRaised` | `alarm.AlarmEngine.Raise` | 北向 push 引擎 | `model.Alarm` JSON（`device_sn`, `alarm_id`, `severity`, `code`, `raised_at`） |
| `alarm.cleared` | `SubjectAlarmCleared` | `alarm.AlarmEngine.Clear` | 北向 push 引擎 | `model.Alarm` JSON（同上 + `cleared_at`） |
| `alarm.acknowledged` | `SubjectAlarmAcknowledged` | `alarm.AlarmEngine.Acknowledge` | （暂无）| `model.Alarm` JSON + `acknowledged_by` |
| `alarm.updated` | `SubjectAlarmUpdated` | `alarm.AlarmEngine.UpdateByEvent` | 北向 push 引擎 | 变更后的 `model.Alarm` |
| `alarm.sync.requested` | `SubjectAlarmSyncRequested` | `alarm.AlarmReceiver` | `alarm.AlarmSyncService` | `{device_sn, requested_at}` |
| `alarm.sync.completed` | `SubjectAlarmSyncCompleted` | `alarm.AlarmSyncProcessor` | （暂无）| `{device_sn, count, duration_ms}` |

**业务流程**：
1. ACS 收到 Inform（含 `M AlarmInfo` 或 ExpeditedEvent）→ 发布 `device.inform.alarm` / `device.inform.expedited_alarm`
2. `alarm.AlarmEngine` 订阅 → 去重 / 关联 / 写库 → 发布 `alarm.raised` / `alarm.cleared`
3. 北向 push 引擎订阅 `alarm.raised|cleared|updated` → 转发给 OSS（`oss.alarm.forward`）

### 2.2 Device（设备生命周期 / Inform）

| Subject | 常量 | Publisher | Consumer | Payload |
|---------|------|-----------|----------|---------|
| `device.inform.bootstrap` | `SubjectDeviceBootstrap` | `acs/handler.go` | `device.InformHandler` + `provision.Engine` | `InformPayload`（`sn`, `oui`, `product_class`, `events`, `params`） |
| `device.inform.periodic` | `SubjectDevicePeriodic` | `acs/handler.go` | `device.InformHandler`（更新心跳） | `InformPayload` |
| `device.inform.value_change` | `SubjectDeviceValueChange` | `acs/handler.go` | `device.InformHandler` | `InformPayload` |
| `device.inform.alarm` | `SubjectDeviceAlarm` | `acs/handler.go` | `alarm.AlarmReceiver` | `InformPayload` + 提取的告警字段 |
| `device.inform.transfer_complete` | `SubjectDeviceTransferComplete` | `acs/handler.go handleTransferComplete` | `software.Service` | `{sn, command_key, fault_code, fault_string}` |
| `device.inform.autonomous_transfer_complete` | `SubjectDeviceAutonomousTransferComplete` | `acs/handler.go` | `transfer.Bridge` | `{sn, file_type, file_url, target_filename}` |
| `device.inform.reboot_complete` | `SubjectDeviceRebootComplete` | `acs/handler.go` | `device.InformHandler` + `task.RebootCloser` + 北向 push | `InformPayload` |
| `device.inform.upgrade_finish` | `SubjectDeviceUpgradeFinish` | `acs/handler.go`（5G） | `software.UpgradeExecutor` | `InformPayload` |
| `device.registered` | `SubjectDeviceRegistered` | `device.Service.RegisterDevice` | `provision.Engine`（开站入口） | `{sn, oui, product_class, carrier, tech}` |
| `device.reboot.abnormal` | `SubjectDeviceRebootAbnormal` | `device.DeviceService.RecordBootFromInform` | `alarm.RebootMonitor` | `{sn, last_boot_at, boot_count}` |
| `device.inform.expedited_alarm` | `SubjectDeviceExpeditedAlarm` | `acs/handler.go`（检测到 ExpeditedEvent 参数） | `alarm.ExpeditedEventReceiver` | `InformPayload` |
| `device.connection.lost` | `SubjectDeviceConnectionLost` | `device.HeartbeatMonitor` | （暂无）| `{sn, last_seen_at}` |

### 2.3 Command Response（RPC 响应回调）

| Subject | 常量 | Publisher | Consumer | Payload |
|---------|------|-----------|----------|---------|
| `command.set_parameters.response` | `SubjectCommandSetParamsResponse` | `acs/handler.go` | （暂无） | `{cwmp_id, sn, status}` |
| `command.get_parameters.response` | `SubjectCommandGetParamsResponse` | `acs/handler.go` | `provision.Engine`（队列 `provision-gpv`） | `{cwmp_id, sn, params:[]Parameter}` |
| `command.get_names.response` | `SubjectCommandGetNamesResponse` | `acs/handler.go` | `provision.Engine`（队列 `provision-gpn`） | `{cwmp_id, sn, names:[]string}` |
| `command.download.response` | `SubjectCommandDownloadResponse` | `acs/handler.go` | （暂无） | `{cwmp_id, sn, status, start_time}` |
| `command.upload.response` | `SubjectCommandUploadResponse` | `acs/handler.go` | （暂无） | `{cwmp_id, sn, status}` |
| `command.add_object.response` | `SubjectCommandAddObjectResponse` | `acs/handler.go` | （暂无） | `{cwmp_id, sn, instance_number}` |
| `command.delete_object.response` | `SubjectCommandDeleteObjectResponse` | `acs/handler.go` | （暂无） | `{cwmp_id, sn, status}` |
| `command.reboot.response` | `SubjectCommandRebootResponse` | `acs/handler.go` | （暂无） | `{cwmp_id, sn}` |
| `command.factory_reset.response` | `SubjectCommandFactoryResetResponse` | `acs/handler.go` | （暂无） | `{cwmp_id, sn}` |
| `command.get_attrs.response` | `SubjectCommandGetAttrsResponse` | `acs/handler.go` | （暂无） | `{cwmp_id, sn, attrs}` |
| `command.set_attrs.response` | `SubjectCommandSetAttrsResponse` | `acs/handler.go` | （暂无） | `{cwmp_id, sn, status}` |

> 历史决策：`command.*`（不带 `.response`）的请求事件已移除。RPC 下发统一通过 Redis 任务队列（`acs:taskq:{sn}`）完成，ACS 在会话中 Pop 组装 SOAP，无需 NATS 层的请求事件。详见 `subjects.go` 注释。

### 2.4 Task（统一任务队列终态）

| Subject | 常量 | Publisher | Consumer | Payload |
|---------|------|-----------|----------|---------|
| `task.completed` | `SubjectTaskCompleted` | `internal/task.TaskService.MarkTaskCompleted` | `worker.taskEventBridge` → `mml.ResultAggregator.OnTaskCompleted` | `task.Task` JSON |
| `task.failed` | `SubjectTaskFailed` | `internal/task.TaskService.MarkTaskFailed` | `worker.taskEventBridge` → `mml.ResultAggregator.OnTaskCompleted` | `task.Task` JSON（含 `error_message`） |

### 2.5 PM / MR

| Subject | 常量 | Publisher | Consumer |
|---------|------|-----------|----------|
| `pm.file.received` | `SubjectPMFileReceived` | `transfer.Bridge` | `pm.Collector` |
| `pm.file.parsed` | `SubjectPMFileParsed` | `pm.Collector` | （暂无）|
| `mr.file.received` | `SubjectMRFileReceived` | `transfer.Bridge` | `mr.Collector` |
| `mr.file.parsed` | `SubjectMRFileParsed` | `mr.Collector` | （暂无）|

## 3. 实现选择：ChannelBus vs NATS JetStream

### 3.1 ChannelEventBus（开发 / 单实例）

- **位置**：`internal/core/event/channel_bus.go`
- **存储**：进程内 Go channel（`bufferSize` 默认 256）
- **持久化**：无（进程重启即丢失）
- **跨实例**：不支持
- **NATS 通配符**：支持 `*`（单级）、`>`（多级）
- **典型场景**：单元测试、单进程开发自测、CI

### 3.2 NATSEventBus（生产 / 多实例）

- **位置**：`internal/core/event/nats_bus.go`
- **存储**：NATS JetStream（持久化到磁盘）
- **持久化**：是（重启不丢失，支持回放）
- **跨实例**：支持（`Subscribe` 各实例独立收，`QueueSubscribe` 同 queue 负载均衡）
- **交付语义**：At-Least-Once（Ack/Nak/Term + 指数退避）
- **重试策略**：handler 失败 → `NakWithDelay` 指数退避（1s, 2s, 4s, 8s ...）；达到 `maxDeliveries=5` → `Term` 终止避免无限重试
- **典型场景**：app / acs / worker 多进程部署

### 3.3 实现切换

主会话整合（DI 容器）按配置选择：
- 配置 `nats.enabled=false` → `NewChannelEventBus`
- 配置 `nats.enabled=true` → `NewNATSEventBus(conn, js, logger)`

> 切换由 `cmd/app/provider/modules.go` 决定，本任务（W3.E.1）只实现 NATSBus 类型 + 测试，DI 整合留给主会话。

## 4. 命名规范（强制）

1. **点分层级**：`domain.action.detail`（例：`device.inform.bootstrap`）
2. **小写字母 + 下划线**：禁用驼峰、连字符；多词 segment 用下划线（例：`value_change`、`set_parameters`）
3. **常量定义集中**：所有 subject 必须在 `internal/core/event/subjects.go` 定义为常量，禁止在 publisher / consumer 处硬编码字符串
4. **publisher / consumer 注释**：每个常量注释中必须列出 publisher 模块、consumer 模块（"暂无"也要写）
5. **payload 通过 `event.NewEvent` 序列化**：业务结构体直接传入，由 `json.Marshal` 处理，consumer 用 `event.DecodePayload` 解码

## 5. 验证清单

- [x] 所有 subject 在 `subjects.go` 集中定义（grep `Subject\w+\s*=`）
- [x] 命名遵循点分层级（test：`TestSubjectConstants_DotSeparated`）
- [x] 至少 3 类关键事件已规范化迁移：
  - `alarm.*`：raised / cleared / acknowledged / updated（W3.E.2 ≥ 1 类 ✓）
  - `device.inform.*`：bootstrap / periodic / value_change / alarm / reboot_complete（W3.E.2 ≥ 1 类 ✓）
  - `command.*.response`：set / get / download / upload / reboot ...（W3.E.2 ≥ 1 类 ✓）
  - `task.completed` / `task.failed`（W3.E.2 ≥ 1 类 ✓）
- [x] NATSBus 实现 EventBus 接口（NewNATSEventBus + Publish + Subscribe + QueueSubscribe + Close）
- [x] decideAck 重试策略测试覆盖：Ack / Nak with backoff / Term

## 6. 后续 TODO（不在 W3.E.1/W3.E.2 范围）

- 主会话整合：在 `cmd/app/provider/modules.go` 加入 `nats.enabled` 配置切换 ChannelBus / NATSBus
- JetStream Stream / Consumer 的初始化与运维 Runbook（哪些 subject 进哪个 Stream、保留策略、磁盘上限）
- 死信队列（DLQ）：`Term` 终止的消息归档到 `dlq.*` subject 供运维查看
- 跨进程链路追踪：在 `Event.Metadata` 注入 `trace_id`，consumer 还原为新 span
