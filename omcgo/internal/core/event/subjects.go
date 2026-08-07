package event

// Parameter synchronization events. Payloads contain durable identifiers and
// result references only; raw SOAP/results remain in storage.
const (
	SubjectParamSyncTaskResult   = "param_sync.task.result"
	SubjectParamSyncRequested    = "param_sync.requested"
	SubjectParamSyncRunCompleted = "param_sync.run.completed"
	SubjectParamSyncRunFailed    = "param_sync.run.failed"
)

// Device events
//
// 这类事件由 ACS Handler 在处理 CPE TR-069 Inform 报文时发布。
const (
	// SubjectDeviceBootstrap 是设备首次入网或出厂重置结束时发布。
	// Inform 事件码包含 "0 BOOTSTRAP"。
	// 发布者：acs/handler.go，订阅者：device.InformHandler（设备注册）、provision.Engine（自动开站入口）
	SubjectDeviceBootstrap = "device.inform.bootstrap"

	// SubjectDevicePeriodic 是设备周期性心跳 Inform 时发布。
	// Inform 事件码包含 "2 PERIODIC"。
	// 发布者：acs/handler.go，订阅者：device.InformHandler（更新心跳时间、在线状态）
	SubjectDevicePeriodic = "device.inform.periodic"

	// SubjectDeviceValueChange 是设备参数发生变化时发布。
	// Inform 事件码包含 "4 VALUE CHANGE"。
	// 发布者：acs/handler.go，订阅者：device.InformHandler（同心跳处理逻辑）
	SubjectDeviceValueChange = "device.inform.value_change"

	// SubjectDeviceAlarm 是设备上报告警信息时发布。
	// Inform 事件码包含 "M AlarmInfo" 类型。
	// 发布者：acs/handler.go，订阅者：alarm.AlarmReceiver（告警活动展示、历史入库）
	SubjectDeviceAlarm = "device.inform.alarm"

	// SubjectDeviceTransferComplete 是 CPE 主动上报 TransferComplete SOAP 报文时发布。
	// 这是 ACS 主动发起 Upload/Download RPC 后的异步确认。
	// 发布者：acs/handler.go handleTransferComplete，订阅者：software.Service（固件升级完成确认）
	SubjectDeviceTransferComplete = "device.inform.transfer_complete"

	// SubjectDeviceAutonomousTransferComplete 是 CPE 自主发起文件传输并上报 AutonomousTransferComplete 时发布。
	// 典型场景：CPE 定时自动上传 PM/MR 文件。
	// 发布者：acs/handler.go handleAutonomousTransferComplete，订阅者：transfer.Bridge（下载并分类到 MinIO）
	SubjectDeviceAutonomousTransferComplete = "device.inform.autonomous_transfer_complete"

	// SubjectDeviceRebootComplete 是设备重启完成后再次接入时发布。
	// Inform 事件码包含 "1 BOOT"（自主重启）或 "M Reboot"（ACS 主动下发 Reboot
	// 后的回包），且不含 "0 BOOTSTRAP"（首次入网走 bootstrap 主题）。
	// 发布者：acs/handler.go；
	// 订阅者：
	//   - device.InformHandler.handleRebootComplete：更新 last_inform_at / 状态 /
	//     IP / ConnectionRequestURL，原子递增 boot_count 并写入 last_boot_at；
	//     按 HaltReason 口径识别异常重启并发布 SubjectDeviceRebootAbnormal。
	//   - task.RebootCloser：收到含 "M Reboot" 的 Inform 时，兜底收敛 device_tasks
	//     里仍为 pending/sent 的 Reboot / FactoryReset 任务（覆盖 RebootResponse
	//     丢包、CPE 跳 ACK 直接重启等边界）。
	//   - northbound.push.Engine：透传给 DataTypes 含 "device_event" 的 OSS 目标。
	SubjectDeviceRebootComplete = "device.inform.reboot_complete"

	// SubjectDeviceUpgradeFinish 是 5G 设备上报 102 UPGRADE FINISH 事件时发布。
	// 表示 5G 设备固件升级流程结束（TransferComplete 仅表示下载完成，5G 需额外等此事件）。
	// 发布者：acs/handler.go，订阅者：software.UpgradeExecutor（5G 升级终态判定）
	SubjectDeviceUpgradeFinish = "device.inform.upgrade_finish"

	// SubjectDeviceConnectionRequest 是收到设备发起的 ConnectionRequest Inform 时发布。
	// 发布者：acs/handler.go，订阅者：暂无
	SubjectDeviceConnectionRequest = "device.inform.connection_request"

	// SubjectDeviceStartupStageReport is the CMCC automatic-start stage event
	// ("105 STARTUP STAGE REPORT"). Payload follows the regular Inform event
	// shape and carries Stage in ParameterList.
	SubjectDeviceStartupStageReport = "device.inform.startup_stage_report"

	// SubjectDeviceStartupResultReport is the CMCC automatic-start terminal
	// event ("106 STARTUP RESULT REPORT"). Payload carries Status and optional
	// FailureCause in ParameterList.
	SubjectDeviceStartupResultReport = "device.inform.startup_result_report"

	// SubjectDeviceConnectionLost 是设备连接超时或主动断线时发布。
	// 发布者：device.HeartbeatMonitor，订阅者：暂无（可用于天致告警联动）
	SubjectDeviceConnectionLost = "device.connection.lost"

	// SubjectDeviceRegistered 是新设备首次入库（写入 devices 表）后发布。
	// 发布者：device.Service.RegisterDevice，订阅者：provision.Engine（触发自动开站流程）
	SubjectDeviceRegistered = "device.registered"

	// SubjectDeviceAttributesChanged 是设备分组关键属性（LAC/TAC 等）在 Inform
	// 回写时实际变化后发布。
	//
	// 背景：device_groups 支持按 LAC/TAC/SerialNumber/DeviceName 匹配，但
	// device.registered 只携带 device_id + serial_number，且仅在首次注册触发；
	// 周期 Inform 写入 device_info.lac/tac 之后若不另发事件，分组匹配必须等
	// @hourly cron 兜底，最坏 60 分钟漂移。
	//
	// 发布者：device.DeviceService.UpdateFromInform / RegisterFromInform —— 解析
	// Inform ParameterList 拿到 LAC/TAC，与旧值 diff 后**只在实际变化时 publish**
	// （首次从 NULL 变成有值也算变化）。空值不覆盖已有，无变化不 publish 避免
	// 在 10w 设备 / 5min Inform 周期下产生 333 events/秒 风暴。
	//
	// 订阅者：topology.GroupMatchEngine.handleAttributesChanged —— 调
	// AssignDeviceToGroup(MatchRequest{device_id, sn, lac, tac}) 单设备瞬时归组。
	//
	// Payload：{ device_id, serial_number, lac, tac, changed_fields }
	SubjectDeviceAttributesChanged = "device.attributes.changed"

	// SubjectDeviceOffline 是已存在设备状态由 active 跌为 offline 时发布。
	// 发布者：device.OfflineDetector（last_inform_at 超阈值时扫描标记）；
	// 订阅者：暂无（保留为通用基础设施事件）。
	// 注：当前 offline_detector.go 仍用字面量 "device.offline" 发布；常量定义在此处供未来迁移。
	SubjectDeviceOffline = "device.offline"

	// SubjectDeviceOnline 是已存在设备从 offline 状态恢复 active 时发布（T-0123）。
	// 发布者：device.DeviceService.UpdateFromInform；
	// 订阅者：provision.Engine.HandleDeviceOnline — 触发 durable 全量同步检测离线期间参数漂移。
	// 与 SubjectDeviceFirmwareChanged 二选一：同一 Inform 若 swVersion 也变化则只发 firmware.changed
	// 不发 online（避免两路 Path B 重复同步）。
	SubjectDeviceOnline = "device.online"

	// SubjectDeviceFirmwareChanged 是设备固件版本变化时发布（T-0125）。
	// 发布者：device.DeviceService.UpdateFromInform — 比对 oldVersion vs newVersion 不同时触发；
	// 订阅者：provision.Engine.HandleFirmwareChanged — Redis 串行锁 + RequestModelUpload；
	// 若同时发生 offline→active，则补交一次 durable 全量同步。
	// 与 SubjectDeviceOnline 二选一：firmware 变化时优先，避免两路全量同步重复触发。
	SubjectDeviceFirmwareChanged = "device.firmware.changed"

	// SubjectDeviceRebootAbnormal 是检测到设备异常重启时发布。
	// 识别口径：必须有 "1 BOOT"；5G gNB 还要求 HaltReason.MainReason=halt_reboot；
	// 其它设备沿用 HaltReason.MainReason 非空（参数缺失时退回旧事件码组合）。
	// 发布者：device.DeviceService.RecordBootFromInform；
	// 订阅者：alarm.RebootMonitor — 滑动窗口内累计 >=阈值触发 FREQUENT_ABNORMAL_REBOOT 告警。
	SubjectDeviceRebootAbnormal = "device.reboot.abnormal"

	// SubjectDeviceExpeditedAlarm 是 VALUE CHANGE Inform 中包含 ExpeditedEvent 参数时发布。
	// ExpeditedEvent 携带实时告警通知（NewAlarm / ChangedAlarm / ClearedAlarm）。
	// 发布者：acs/handler.go（检测到 Device.FaultMgmt.ExpeditedEvent.* 参数时）。
	// 订阅者：alarm.ExpeditedEventReceiver（解析参数并路由到告警引擎）。
	SubjectDeviceExpeditedAlarm = "device.inform.expedited_alarm"

	// SubjectDeviceFaultDetected 是 ACS 检测到设备异常重启并携带故障原因时发布。
	// Inform 事件码包含 "1 BOOT" 且满足设备类型对应的 HaltReason 口径；或 5G 软重启（"4 VALUE_CHANGE" + soft_reboot）。
	// 发布者：acs/handler.go publishInformEvents（当 IsBoot 且有故障原因参数时）。
	// 订阅者：stationlog.FaultLogService（按收集模式决定是否自动下发 SetParam + FaultLogURL）。
	SubjectDeviceFaultDetected = "device.fault.detected"
)

// System-wide control-plane events
//
// sys.> 系统级控制通知事件。这些事件影响所有部署单元的全局状态（权限、
// 配置热重载等），需要跨 app/acs/worker 广播。
const (
	// SubjectSysCasbinPolicyReload 是 Casbin RBAC 策略变更时发布。
	// 各进程的 CasbinAuthorizer 订阅后调用 Enforcer.LoadPolicy() 从 PG
	// 全量重载策略，实现多实例权限变更实时同步（取代原 Redis Pub/Sub
	// casbin:policy:reload 通道，获得 JetStream 持久化 + 重连回放保证）。
	//
	// 发布者：admin.CasbinAuthorizer.NotifyPolicyChange（管理员通过 API
	// 修改角色/权限/用户角色关系后）。
	// 订阅者：所有运行 CasbinAuthorizer 的进程（app / 未来拆分到 acs/worker
	// 的 RBAC 实例）。
	SubjectSysCasbinPolicyReload = "sys.casbin.policy.reload"

	// SubjectSysConfigSaved is published after sys_configs BatchUpsert commits.
	// Payload: SysConfigSavedPayload{Category}.
	// 发布者：app 进程的 sys_config 保存 wiring；
	// 订阅者：需要立即失效本地运行时配置缓存的各部署单元。
	SubjectSysConfigSaved = "sys.config.saved"
)

// Command 请求型事件的 Subject 常量已移除（2026-04-22）：
// 原先保留的 command.get_parameters / command.set_parameters / command.download /
// command.upload / command.reboot / command.factory_reset 属于早期设计残留，
// 实际 RPC 下发一律通过 Redis 任务队列（acs:taskq:{sn}）完成，ACS 在会话中 Pop
// 组装 SOAP 响应，无需 NATS 层的请求事件。响应事件仍保留（见下方 Command
// response events 分组）。

// PM events
//
// PM 数据文件的采集与解析流转事件。
const (
	// SubjectPMFileReceived 是 PM 文件入库到 MinIO 后发布。
	// 发布者：transfer.Bridge（收到 AutonomousTransferComplete 后下载文件）。
	// 订阅者：pm.Collector（解析 XML、入库计数器、计算 KPI）
	SubjectPMFileReceived = "pm.file.received"

	// SubjectPMFileDeferred 隔离“文件已到、设备注册尚未可见”的 PM 事件。
	// 主 pm-workers consumer 在持久化接力成功后立即 ACK，避免注册竞态事件占满
	// MaxAckPending 并阻塞已注册设备；pm-registration-wait consumer 保留原事件
	// Timestamp，继续执行注册宽限期重投和最终 DLQ。
	SubjectPMFileDeferred = "pm.file.deferred"

	// SubjectPMFileParsed 是 PM XML 解析完成后发布。
	// 发布者：pm.Collector，订阅者：暂无（可用于选择性后续处理）
	SubjectPMFileParsed = "pm.file.parsed"

	// SubjectPMAggregationNormalized 承载已完成 15 分钟 Counter/KPI 计算的标准事件。
	// ACK 后仅短期保留，供当前小时窗口恢复；日/周/月不再重放原始 15 分钟事件。
	SubjectPMAggregationNormalized = "pmaggregation.15m.normalized"

	// SubjectPMAggregationHourlyRollup 承载已发布小时窗口的紧凑 Counter 状态。
	SubjectPMAggregationHourlyRollup = "pmaggregation.hourly.rollup"

	// SubjectPMAggregationDailyRollup 承载已发布日窗口的紧凑 Counter 状态。
	// 同一事件同时供周窗口和月窗口累计。
	SubjectPMAggregationDailyRollup = "pmaggregation.daily.rollup"

	// SubjectPMAggregationTaskVersionChanged 通知 worker 刷新不可变任务版本快照。
	SubjectPMAggregationTaskVersionChanged = "pmaggregation.control.task_version.changed"
)

// MR events
//
// 测量报告（Measurement Report）文件的采集与解析流转事件。
const (
	// SubjectMRFileReceived 是 MR 文件入库到 MinIO 后发布。
	// 发布者：transfer.Bridge（AutonomousTransferComplete 路径），订阅者：mr.Collector（解析并入库）
	SubjectMRFileReceived = "mr.file.received"

	// SubjectMRFileUploaded 是设备通过 HTTP POST 直传 MR 文件成功后发布（F05 任务管理路径）。
	// 与 SubjectMRFileReceived 区别：本事件由 acs/upload Handler 在收到 fileType=MR 上传时发出，
	// payload 含 URL 查询参数里的 cellCode（任务管理用于写 Redis 心跳）。
	// 发布者：acs.upload.Handler，订阅者：mr/task.HeartbeatSubscriber（Redis SET + PG TouchHeartbeat）
	SubjectMRFileUploaded = "mr.file.uploaded"

	// SubjectMRFileParsed 是 MR XML 解析完成后发布。
	// 发布者：mr.Collector，订阅者：暂无
	SubjectMRFileParsed = "mr.file.parsed"
)

// Alarm events
//
// 告警生命周期事件，由告警引擎发布。
const (
	// SubjectAlarmRaised 是新告警产生时发布。
	// 发布者：alarm.AlarmEngine，订阅者：北向接口模块（告警推送）
	SubjectAlarmRaised = "alarm.raised"

	// SubjectAlarmCleared 是告警恢复（清除）时发布。
	// 发布者：alarm.AlarmEngine，订阅者：北向接口模块
	SubjectAlarmCleared = "alarm.cleared"

	// SubjectAlarmAcknowledged 是告警被确认时发布。
	// 发布者：alarm.AlarmEngine，订阅者：暂无
	SubjectAlarmAcknowledged = "alarm.acknowledged"

	// SubjectAlarmSyncRequested 是请求同步设备告警时发布。
	// 发布者：alarm.AlarmReceiver，订阅者：alarm.AlarmSyncService
	SubjectAlarmSyncRequested = "alarm.sync.requested"

	// SubjectAlarmSyncCompleted 是告警同步完成时发布。
	// 发布者：alarm.AlarmSyncProcessor，订阅者：暂无
	SubjectAlarmSyncCompleted = "alarm.sync.completed"

	// SubjectAlarmUpdated 是告警属性变更（如严重程度）时发布。
	// 发布者：alarm.AlarmEngine.UpdateByEvent（处理 ChangedAlarm 通知）。
	// 订阅者：北向接口模块（告警推送）
	SubjectAlarmUpdated = "alarm.updated"
)

// Provisioning events
//
// 自动开站引擎的任务生命周期事件。
const (
	// SubjectProvisionStarted 是开站任务启动时发布。
	// 发布者：provision.Engine，订阅者：暂无（可用于任务状态追踪）
	SubjectProvisionStarted = "provision.started"

	// SubjectProvisionCompleted 是开站任务全部步骤成功完成时发布。
	// 发布者：provision.Engine，订阅者：暂无
	SubjectProvisionCompleted = "provision.completed"

	// SubjectProvisionFailed 是开站任务失败时发布。
	// 发布者：provision.Engine，订阅者：暂无
	SubjectProvisionFailed = "provision.failed"

	// SubjectProvisionStepDone 是开站单个步骤执行完成时发布。
	// 发布者：provision.Engine，订阅者：provision.Engine 自身（驱动下一步骤）
	SubjectProvisionStepDone = "provision.step.done"
)

// DataModel events
//
// 参数模型（XML 数据模型）上传与处理流转事件。
const (
	// SubjectDataModelUploadRequested 是系统开始为设备下发 Upload RPC （FileType=11）时发布。
	// 发布者：provision.ModelUploadService，订阅者：暂无
	SubjectDataModelUploadRequested = "datamodel.upload.requested"

	// SubjectDataModelUploadCompleted 是 Upload RPC 块成功收到设备响应时发布。
	// 发布者：provision.ModelUploadService，订阅者：暂无
	SubjectDataModelUploadCompleted = "datamodel.upload.completed"

	// SubjectDataModelUploadFailed 是 Upload RPC 失败时发布。
	// 发布者：provision.ModelUploadService，订阅者：暂无
	SubjectDataModelUploadFailed = "datamodel.upload.failed"

	// SubjectDataModelFileReceived 是 CPE 实际上传的 XML 文件应用到 MinIO 后发布。
	// 发布者：acs/upload/handler.go publishDataModelEvent（FileType=11 上传完成）。
	// 订阅者：provision.Engine（对应服务：cmd/app）—下载 XML、解析、写入 data_model_definitions 表
	SubjectDataModelFileReceived = "datamodel.file.received"
)

// Command response events
//
// 这类事件由 ACS Handler 在收到 CPE RPC 响应时发布，主要用于参数同步和开站引擎的状态追踪。
const (
	// SubjectCommandGetParamsResponse 是收到 GetParameterValuesResponse 时发布。
	// 发布者：acs/handler.go，订阅者：provision.Engine（"provision-gpv" 队列，处理参数同步）
	SubjectCommandGetParamsResponse = "command.get_parameters.response"

	// SubjectCommandSetParamsResponse 是收到 SetParameterValuesResponse 时发布。
	// 发布者：acs/handler.go，订阅者：暂无
	SubjectCommandSetParamsResponse = "command.set_parameters.response"

	// SubjectCommandDownloadResponse 是收到 DownloadResponse 时发布。
	// 发布者：acs/handler.go，订阅者：暂无
	SubjectCommandDownloadResponse = "command.download.response"

	// SubjectCommandUploadResponse 是收到 UploadResponse 时发布。
	// 发布者：acs/handler.go，订阅者：暂无
	SubjectCommandUploadResponse = "command.upload.response"

	// SubjectCommandGetNamesResponse 是收到 GetParameterNamesResponse 时发布。
	// 发布者：acs/handler.go，订阅者：provision.Engine（"provision-gpn" 队列，处理参数路径发现）
	SubjectCommandGetNamesResponse = "command.get_names.response"

	// SubjectCommandAddObjectResponse 是收到 AddObjectResponse 时发布。
	// 发布者：acs/handler.go，订阅者：暂无
	SubjectCommandAddObjectResponse = "command.add_object.response"

	// SubjectCommandDeleteObjectResponse 是收到 DeleteObjectResponse 时发布。
	// 发布者：acs/handler.go，订阅者：暂无
	SubjectCommandDeleteObjectResponse = "command.delete_object.response"

	// SubjectCommandRebootResponse 是收到 RebootResponse 时发布。
	// 发布者：acs/handler.go，订阅者：暂无
	SubjectCommandRebootResponse = "command.reboot.response"

	// SubjectCommandFactoryResetResponse 是收到 FactoryResetResponse 时发布。
	// 发布者：acs/handler.go，订阅者：暂无
	SubjectCommandFactoryResetResponse = "command.factory_reset.response"

	// SubjectCommandGetAttrsResponse 是收到 GetParameterAttributesResponse 时发布。
	// 发布者：acs/handler.go，订阅者：暂无
	SubjectCommandGetAttrsResponse = "command.get_attrs.response"

	// SubjectCommandSetAttrsResponse 是收到 SetParameterAttributesResponse 时发布。
	// 发布者：acs/handler.go，订阅者：暂无
	SubjectCommandSetAttrsResponse = "command.set_attrs.response"
)

// Device task lifecycle events
//
// 统一任务队列（device_tasks + acs:taskq Redis）跨进程通知主题。
// ACS 进程在 RPC 响应匹配到任务后调用 TaskService.MarkTaskCompleted/Failed，
// 两者在更新完 Redis + PostgreSQL 后发布下列事件，APP/Worker 进程订阅事件把
// 终态推送给上层聚合器（如 MML ResultAggregator 更新 mml_tasks 统计）。
// 载荷即 task.Task JSON，消费者自行 DecodePayload。
const (
	// SubjectTaskCreated 是 device_tasks 入队成功时发布（status=pending）。
	// T-0157 C5 引入，专供消息中心订阅器消费写"进行中"状态消息。
	// 发布者：internal/task.TaskService.CreateTask；
	// 订阅者：internal/notification.TaskSubscriber → notification.UpsertByDedup。
	// 注：CompletionRouter / event_bridge 不订阅该主题（仅关注终态），无影响。
	SubjectTaskCreated = "task.created"

	// SubjectTaskCompleted 是 device_tasks 任务成功完成时发布（status=completed）。
	// 发布者：internal/task.TaskService.MarkTaskCompleted；
	// 订阅者：worker.taskEventBridge → mml.ResultAggregator.OnTaskCompleted。
	SubjectTaskCompleted = "task.completed"

	// SubjectTaskFailed 是 device_tasks 任务失败或过期时发布（status=failed/expired）。
	// 发布者：internal/task.TaskService.MarkTaskFailed；
	// 订阅者：worker.taskEventBridge → mml.ResultAggregator.OnTaskCompleted。
	SubjectTaskFailed = "task.failed"

	// SubjectTaskCancelled 是 device_tasks 任务被取消时发布（status=cancelled）。
	// 发布者：internal/task.TaskService.CancelTask；
	// 订阅者：按需订阅终态清理逻辑（如告警同步锁释放）。
	SubjectTaskCancelled = "task.cancelled"
)

// Software/Firmware events
//
// 固件升级生命周期事件。
const (
	// SubjectFirmwareUploaded 是固件文件上传到 MinIO 后发布。
	// 发布者：software.Service，订阅者：暂无
	SubjectFirmwareUploaded = "firmware.uploaded"

	// SubjectUpgradeStarted 是升级任务开始（已对设备下发 Download RPC）时发布。
	// 发布者：software.Service，订阅者：暂无
	SubjectUpgradeStarted = "upgrade.started"

	// SubjectUpgradeCompleted 是设备上报 TransferComplete 且升级成功后发布。
	// 发布者：software.Service（订阅 SubjectDeviceTransferComplete 后处理），订阅者：暂无
	SubjectUpgradeCompleted = "upgrade.completed"

	// SubjectUpgradeFailed 是升级失败（TransferComplete 含 fault 或超时）时发布。
	// 发布者：software.Service，订阅者：暂无
	SubjectUpgradeFailed = "upgrade.failed"
)

// Backup events
//
// 设备配置备份任务事件。
const (
	// SubjectBackupTaskCreated 是用户创建备份任务后发布。
	// 发布者：backup.Service，订阅者：backup.Executor（下发 Upload RPC 获取配置文件）
	SubjectBackupTaskCreated = "backup.task.created"

	// SubjectBackupTaskDone 是备份文件已上传到 MinIO 并完成入库后发布。
	// 发布者：backup.Executor，订阅者：暂无
	SubjectBackupTaskDone = "backup.task.done"

	// SubjectBackupFileReceived 是 CPE 上传备份配置文件落 MinIO 后发布（T-0079）。
	// 发布者：acs/upload/handler.go ServeHTTP（FileType=3 / FileTypeConfig 分支）
	// 订阅者：backup.FilePathRecorder（解析 filename 中嵌的 backup_task_id 前缀
	// 后写回 backup_tasks.file_path，建立 task↔path 链路供 restore_by_task_id
	// 模式使用）。Payload 见 backup.BackupFileReceivedPayload。
	SubjectBackupFileReceived = "backup.file.received"

	// SubjectBackupScheduleChanged 在 backup_schedules 表发生 CRUD 后发布
	// （backup-restore-alignment-plan M3 reload 通道）。
	// 发布者：backup.Service.{Create,Update,Delete}Schedule（app 进程）
	// 订阅者：backup.PeriodScheduler.Reload（worker 进程）—— 跨进程热更新 cron。
	// Payload 为空（接收方只需重新拉表）。
	SubjectBackupScheduleChanged = "backup.schedule.changed"
)

// Station log events
//
// 基站日志采集文件事件。
const (
	// SubjectLogFileReceived 是 CPE 上传运行日志（FileType "6"）或故障日志（FileType "8"/"RL"）
	// 成功落 MinIO 后，由 acs/upload/handler.go 发布。
	// 订阅者：stationlog.Service（运行日志入库；故障日志不写入重启记录）。
	// Payload：LogFileReceivedPayload
	SubjectLogFileReceived = "log.file.received"
)

// Report events
//
// 报表生成任务事件。
const (
	// SubjectReportGenerateRequested 是用户触发报表生成时发布。
	// 发布者：report.Service，订阅者：report.Generator（异步执行报表生成并写入 MinIO）
	SubjectReportGenerateRequested = "report.generate.requested"

	// SubjectReportGenerateDone 是报表文件生成完成时发布。
	// 发布者：report.Generator，订阅者：暂无
	SubjectReportGenerateDone = "report.generate.done"
)

// Northbound/OSS events
//
// 北向接口模块对外推送事件，用于与上层管理系统对接。
const (
	// SubjectOSSAlarmForward 是告警需向北向系统推送时发布。
	// 发布者：alarm.Engine，订阅者：北向模块 Push Engine
	SubjectOSSAlarmForward = "oss.alarm.forward"

	// SubjectOSSPMExport 是 PM 数据需对外导出时发布。
	// 发布者：pm.Collector，订阅者：北向模块 Push Engine
	SubjectOSSPMExport = "oss.pm.export"

	// SubjectOSSConfigSnapshot 是配置快照需对外同步时发布。
	// 发布者：provision.Engine，订阅者：北向模块 Sync Service
	SubjectOSSConfigSnapshot = "oss.config.snapshot"

	// SubjectNorthboundServerChanged 是北向主备 OSS 服务器配置变更时发布
	// （SetActive 切换 / Update 修改 host/port）。
	// 发布者：northbound.ServerService，订阅者：northbound.push.Engine
	// 触发 push engine 调用 RefreshActiveTarget(ctx) reload 推送目标（T-0099）。
	// Payload Data: {"role":"primary|standby","action":"active_switch|update","host":"...","port":N}
	SubjectNorthboundServerChanged = "northbound.server.changed"
)

// NE Direct events
//
// 网元直连（NE Direct）模块的会话与命令事件。
const (
	// SubjectNEDirectRegister 是设备初次连接网元直连服务时发布。
	// 发布者：nedirect.Service，订阅者：暂无
	SubjectNEDirectRegister = "nedirect.register"

	// SubjectNEDirectFault 是直连会话发生错误时发布。
	// 发布者：nedirect.Service，订阅者：暂无
	SubjectNEDirectFault = "nedirect.fault"

	// SubjectNEDirectConnect 是直连 WebSocket 会话建立时发布。
	// 发布者：nedirect.Service，订阅者：暂无
	SubjectNEDirectConnect = "nedirect.session.connect"

	// SubjectNEDirectDisconnect 是直连 WebSocket 会话断开时发布。
	// 发布者：nedirect.Service，订阅者：暂无
	SubjectNEDirectDisconnect = "nedirect.session.disconnect"

	// SubjectNEDirectCommand 是通过直连通道发送 MML 命令时发布。
	// 发布者：nedirect.Service，订阅者：暂无
	SubjectNEDirectCommand = "nedirect.command.sent"
)

// Trace events (T-0137 M2)
//
// TR069 报文跟踪模块跨进程协作事件。`trace.task.*` 用 InterestPolicy（fan-out 多
// 订阅者：ACS 各实例维护白名单 + app 进程做 SSE 推送）；`trace.message.captured` 用
// WorkQueuePolicy（worker 群组消费 + 批量落库）。
const (
	// SubjectTraceTaskStarted 是抓包任务创建时发布。
	// 发布者：app.trace.Service.CreateTask，订阅者：acs.WhitelistCache（加 SN）+
	// app.MessageHub（推 SSE 给在线用户）。
	// Payload：trace.TaskEvent {task_id, device_sn, status, expires_at, operator_code, created_by}
	SubjectTraceTaskStarted = "trace.task.started"

	// SubjectTraceTaskStopped 是抓包任务停止（手动/超时）时发布。
	// 发布者：app.trace.Service.StopTask 或 worker 巡检 Sweeper，
	// 订阅者：acs.WhitelistCache（删 SN）+ app.MessageHub。
	SubjectTraceTaskStopped = "trace.task.stopped"

	// SubjectTraceTaskPurged 是抓包任务清理报文时发布。
	// 发布者：app.trace.Service.StopTask(purge=true)，
	// 订阅者：worker（DELETE PG + 删 MinIO 对象）+ app.MessageHub。
	SubjectTraceTaskPurged = "trace.task.purged"

	// SubjectTraceMessageCaptured 是 ACS 命中白名单抓到一条 SOAP 报文时发布。
	// 发布者：acs.handler.maybeCaptureTrace（JetStream WorkQueue），
	// 订阅者：worker.TraceCaptureConsumer（QueueSubscribe 批量落库 + MinIO 外置）。
	// Payload：trace.Message（含 payload_inline；大报文由 worker 转 MinIO）。
	SubjectTraceMessageCaptured = "trace.message.captured"

	// SubjectTraceExportRequested 是用户触发异步导出时发布。
	// 发布者：app.trace.Handler.ExportXML 异步路径，
	// 订阅者：worker.TraceExporter（生成 XML 写 MinIO exchange + 更新 job 状态）。
	SubjectTraceExportRequested = "trace.export.requested"

	// SubjectDictionaryRefreshFailed 是数据字典数据源同步失败时发布(T-0182)。
	// 发布者：worker daily cron(SyncSourceBoundAll)单字典失败 + 总览失败,
	// 订阅者：notification 中心(P3+/F04 后续接入,把字典同步失败通过邮件/Webhook 通知运维)。
	// Payload: {dict_id, dict_name, error_summary, when}。
	SubjectDictionaryRefreshFailed = "dictionary.refresh.failed"
)
