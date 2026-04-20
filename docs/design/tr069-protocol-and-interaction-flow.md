# TR-069 协议与交互流程梳理

> **文档版本**: v1.0
> **创建日期**: 2026-04-20
> **适用项目**: OMC（基站网络运营管理系统）

---

## 1. 概述

本文档梳理 OMC 系统中 TR-069/CWMP 协议的完整交互流程，包括 CPE 与 ACS 的 SOAP 会话、三个部署单元（ACS/APP/Worker）之间的协作机制、消息订阅体系，以及 MML 批量命令的端到端执行链路。

### 三个部署单元

| 单元 | 进程 | 端口 | 核心职责 |
|------|------|------|---------|
| `omcgo-acs` | ACS 引擎 | :7547 | TR-069 SOAP 会话管理、RPC 下发 |
| `omcgo-app` | 主应用 | :8080 | REST API、业务逻辑、任务创建 |
| `omcgo-worker` | 后台工作进程 | 无 HTTP | PM/MR 文件解析、告警处理、备份、报表 |

---

## 2. CPE 通过 TR-069 协议与 ACS 建立连接、发送空报文

### 2.1 TR-069 会话状态机

```
IDLE → INFORM_RECEIVED → PROCESSING → RPC_PENDING → RPC_RESPONSE → COMPLETE → IDLE
```

| 状态 | 含义 | 触发 |
|------|------|------|
| `IDLE` | 初始/最终状态 | — |
| `INFORM_RECEIVED` | 收到 Inform | CPE 发送 Inform |
| `PROCESSING` | 检查是否有待下发命令 | CPE 发送空 POST |
| `RPC_PENDING` | 等待 CPE 响应 | ACS 发送 SOAP RPC |
| `RPC_RESPONSE` | 收到 CPE 响应 | CPE 发送 RPC Response |
| `COMPLETE` | 会话结束 | 队列为空或达到 RPC 上限 |

**关键文件**: `internal/acs/session.go`

### 2.2 完整 SOAP 会话流程

```
CPE                                    ACS Handler
 │                                         │
 │── POST Inform (SOAP XML) ──────────────>│  handleInform()
 │                                         │   1. DecodeInform() 解析设备标识
 │                                         │   2. RateLimiter.Allow() 设备级限流
 │                                         │   3. Admission.Acquire() 全局准入控制
 │                                         │   4. 创建 Session (Redis TTL 5min)
 │                                         │   5. 发布 EventBus 事件
 │<─ 200 OK InformResponse ────────────────│  6. 返回 InformResponse + Set-Cookie
 │   Set-Cookie: SESSION=xxx               │
 │                                         │
 │── POST (empty body) ───────────────────>│  handleEmpty()
 │   Cookie: SESSION=xxx                   │   1. 从 Cookie 恢复 Session
 │                                         │   2. 状态: INFORM_RECEIVED → PROCESSING
 │                                         │   3. TaskService.PopTask(deviceSN)
 │                                         │   4. 生成 CWMP ID
 │                                         │   5. TaskService.MarkTaskSent()
 │<─ 200 OK (SOAP RPC Request) ────────────│  6. 构建 SOAP XML 响应
 │   例: GetParameterValues                │
 │                                         │
 │── POST RPC Response ───────────────────>│  handleRPCResponse()
 │   例: GetParameterValuesResponse        │   1. 解析 CPE 响应
 │                                         │   2. GetTaskByCWMPID() 关联任务
 │                                         │   3. MarkTaskCompleted() 或 MarkTaskFailed()
 │                                         │   4. 发布 EventBus 事件
 │                                         │   5. PopTask() 下发下一个任务
 │<─ 200 OK (下一个 SOAP RPC) ─────────────│  (重复直到队列清空)
 │                                         │
 │── POST RPC Response (最后一个) ─────────>│  completeSession()
 │                                         │   1. Admission.Release()
 │<─ 204 No Content ───────────────────────│   2. 删除 Redis Session
 │   Connection: close                     │   3. go postSessionWake() 异步续唤
```

**关键文件**: `internal/acs/handler.go` — `ServeHTTP()` 按请求 body 和 SOAP 方法名分发到上述 handler。

### 2.3 SOAP 编解码

| 方向 | 方式 | 文件 |
|------|------|------|
| **接收** (CPE → ACS) | `encoding/xml.Decoder` 流式解析 | `pkg/soap/decoder.go` |
| **发送** (ACS → CPE) | `text/template` 预编译模板渲染 | `pkg/soap/templates.go` |

支持的 RPC 方法（11 种）: Get/SetParameterValues, Get/SetParameterNames, Add/DeleteObject, Download, Upload, Reboot, FactoryReset, Get/SetParameterAttributes

### 2.4 流量控制

| 机制 | 实现 | 配置 |
|------|------|------|
| 设备级限流 | `rate.Limiter` 令牌桶，LRU Cache (10万设备) | `rate_limit.per_device=10/min` |
| 全局准入控制 | `atomic.Int64` CAS 并发会话计数 | `session.max_concurrent=10000` |
| 单会话 RPC 上限 | `maxRPCPerSession` 检查 | 默认 15 次 |

---

## 3. ACS 获取设备 RPC 任务并下行给 CPE

ACS 通过双优先级策略获取任务：

```
handleEmpty() / handleRPCResponse()
  │
  ├─[优先级1] TaskService.PopTask(deviceSN)     ← 新任务队列 (device_tasks)
  │             └─ Redis ZPOPMIN acs:taskq:{sn}
  │
  ├─[优先级2] CommandQueue.Pop(deviceSN)        ← 旧命令队列 (向后兼容)
  │             └─ Redis ZPOPMIN acs:cmdq:{sn}
  │
  └─[都为空]  completeSession() + postSessionWake()
```

### 3.1 任务队列 Redis 数据结构

| 结构 | Key | 用途 |
|------|-----|------|
| Sorted Set | `acs:taskq:{deviceSN}` | 任务队列，score = priority×10^13 + timestamp |
| Hash | `acs:task:{taskID}` | 任务详情 JSON，TTL 24h |
| String | `acs:cwmp2task:{hash}` | CWMP ID → Task ID 映射，TTL 24h |

**关键文件**: `internal/task/redis_queue.go`

### 3.2 RPC 方法映射

MML 操作类型到 TR-069 RPC 方法的映射：

| 操作 | RPC 方法 | 说明 |
|------|---------|------|
| LST/DSP | `GetParameterValues` | 读取参数 |
| MOD/ACT/DEA/CLR | `SetParameterValues` | 修改参数 |
| ADD | `AddObject` | 添加对象实例 |
| RMV | `DeleteObject` | 删除对象实例 |
| UPG | `Download` | 固件下载 |
| RST | `Reboot` | 重启设备 |

### 3.3 Connection Request 唤醒设备

任务创建后，如果设备不在线，通过 Connection Request 异步唤醒：

```
TaskService.wakeDevice(deviceSN)               internal/task/service.go:440
  │  go func() (异步, fire-and-forget)
  │
  └─→ connreq.Dispatcher.Send()
        ├─[策略1] UDP Connection Request (NAT穿透，通过 STUN 获取公网地址)
        ├─[策略2] HTTP Connection Request (直连，支持 Digest 认证)
        └─[去重]   Redis SetNX acs:connreq:pending:{sn} TTL=30s
```

**关键文件**: `internal/acs/connreq/dispatcher.go`, `internal/acs/connreq/udp_sender.go`

---

## 4. CPE 完成 RPC 任务，ACS 更新任务结果

### 4.1 结果处理流程

```
CPE RPC Response
  │
  v
handleRPCResponse()                            internal/acs/handler.go:662
  │
  ├─ 提取 CWMP ID (SOAP Header)
  ├─ GetTaskByCWMPID(cwmpID)                   Redis Hash 反查 → PG 兜底
  │
  ├─[SOAP Fault]  MarkTaskFailed(taskID, faultCode, faultMsg)
  │                 └─ Redis Hash 更新 + PG 同步 + Prometheus 指标
  │                 └─ notifyCompletion() → ResultAggregator (MML 回调)
  │
  └─[成功]       MarkTaskCompleted(taskID, resultJSON)
                    └─ Redis Hash 更新 + PG 同步 + Prometheus 指标
                    └─ notifyCompletion() → ResultAggregator (MML 回调)
```

### 4.2 MML 任务结果回聚

```
TaskService.notifyCompletion()                 internal/task/service.go:460
  │  仅处理 Source=MML 且有 ParentTaskID 的 device_task
  │
  v
ResultAggregator.OnTaskCompleted()             internal/mml/result_aggregator.go:31
  │
  ├─ IncrementStats(mmlID, successDelta, failedDelta)   PG 原子递增
  │
  └─ finalizeIfComplete()
       ├─ 检查 done == total
       ├─ 确定 finalStatus (Completed/Failed) + finalResult (Success/Partial/Failed)
       ├─ Update mml_task 状态
       └─ SSE 推送 "mml_task_completed" 事件给前端
```

---

## 5. ACS 与 APP、Worker 的交互

### 5.1 进程间通信拓扑

```
         CPE 设备
            │ TR-069 SOAP/XML
            v
    ┌───────────────┐
    │   ACS 进程     │ :7547
    │               │
    │ SessionStore  │──── Redis ────┐
    │ TaskService   │               │
    │ EventBus      │─── NATS ────┐ │
    └───────────────┘             │ │
                                  │ │
           NATS JetStream         │ │
          ┌───────────┐           │ │
          │  device.> │           │ │
          │  command.>│           │ │
          │  pm.>     │           │ │
          │  alarm.>  │           │ │
          │  backup.> │           │ │
          └─────┬─────┘           │ │
                │                 │ │
    ┌───────────┴─────┐  ┌───────┴─┴───────┐
    │   Worker 进程    │  │    APP 进程       │ :8080
    │                 │  │                  │
    │ PM Collector    │  │ REST API         │
    │ MR Collector    │  │ MML Service      │
    │ AlarmReceiver   │  │ Task Service     │
    │ TransferBridge  │  │ EventBus Sub     │
    │ BackupExecutor  │  │ Northbound Push  │
    │ ReportGenerator │  │ SSE MessageHub   │
    └─────────────────┘  └──────────────────┘
              │                    │
              └──── PostgreSQL ────┘
                 (共享 device_tasks)
```

### 5.2 共享数据层

**PostgreSQL 共享表**:

| 表 | 写入方 | 读取方 | 说明 |
|----|-------|-------|------|
| `device_tasks` | APP (创建) + ACS (状态更新) | APP (API查询) + ACS (PopTask) | 统一任务队列 |
| `devices` | APP | APP | 设备注册信息 |

**Redis 共享 Key**:

| Key 模式 | 写入方 | 读取方 |
|---------|-------|-------|
| `acs:taskq:{deviceSN}` | APP (TaskService.Push) | ACS (TaskService.Pop) |
| `acs:task:{taskID}` | APP + ACS | APP + ACS |
| `acs:cwmp2task:{hash}` | ACS | ACS |
| `acs:cmdq:{deviceSN}` | APP (旧接口) | ACS (旧接口) |
| `acs:connreq:pending:{sn}` | APP + ACS | APP + ACS |

---

## 6. APP 后台 MML 模块批量命令执行

### 6.1 端到端完整调用链

```
前端 POST /api/v1/mml/execute
  │  {command_code, device_sns, parameters, execute_type}
  v
MML Handler.Execute()                          cmd/app/provider/router.go
  │
  v
MML Service.ExecuteCommand()                   internal/mml/service.go:304
  │
  ├─ [1] 解析 command_code → MMLCommand (rpc_method, parameters)
  ├─ [2] 或解析 script_id → 展开脚本为 commands[]
  ├─ [3] 构建 MMLTask{Status=Pending, Commands, DeviceSNs}
  ├─ [4] PostgreSQL: INSERT mml_tasks
  ├─ [5] 审计日志: INSERT mml_audit_logs
  │
  ├─ [6] Fanouter.Fanout()                     internal/mml/fanout.go:35
  │       │  N commands × M devices = N×M 个 device_task
  │       └─ TaskService.BatchCreateTasks()     internal/task/service.go:400
  │              ├─ PostgreSQL: INSERT device_tasks
  │              ├─ Redis: ZADD acs:taskq:{sn} + HSET acs:task:{id}
  │              └─ wakeDevice(sn) 异步 Connection Request
  │
  ├─ [7] PostgreSQL: UPDATE mml_tasks SET status='running'
  └─ [8] SSE: PublishSimple("mml_task_status")
  │
  v
返回 MMLTask JSON → 前端轮询 GET /mml/tasks/:id
```

### 6.2 设备端执行与结果回聚

```
ACS: PopTask → SOAP 下发 → CPE 执行
  │
  v
ACS: MarkTaskCompleted/Failed()
  │
  v
TaskService.notifyCompletion()                 仅处理 MML 来源的 device_task
  │
  v
ResultAggregator.OnTaskCompleted()             internal/mml/result_aggregator.go
  ├─ IncrementStats(mmlID, +1 success / +1 failed)
  └─ finalizeIfComplete()
       ├─ done == total ?
       ├─ finalResult = Success / Partial / Failed
       ├─ UPDATE mml_tasks SET status, result, finished_at
       └─ SSE: PublishSimple("mml_task_completed", {stats})
```

### 6.3 数据流向图

```
前端                APP                         共享存储              ACS                 CPE
  │                  │                            │                   │                   │
  │ POST /execute    │                            │                   │                   │
  │─────────────────>│                            │                   │                   │
  │                  │ Create mml_task             │                   │                   │
  │                  │────────────────────────────>│ PG: mml_tasks     │                   │
  │                  │                            │                   │                   │
  │                  │ Fanout: N×M device_tasks   │                   │                   │
  │                  │────────────────────────────>│ PG: device_tasks  │                   │
  │                  │────────────────────────────>│ Redis: acs:taskq  │                   │
  │                  │                            │                   │                   │
  │                  │ Connection Request          │                   │                   │
  │<─────────────────│<───────────────────────────│                   │                   │
  │                  │                            │                   │<──── UDP/HTTP ────│
  │                  │                            │                   │                   │
  │                  │                            │                   │ PopTask           │
  │                  │                            │<──────────────────│                   │
  │                  │                            │                   │ SOAP RPC ────────>│
  │                  │                            │                   │<──── Response ────│
  │                  │                            │                   │                   │
  │                  │                            │                   │ MarkCompleted     │
  │                  │                            │<──────────────────│                   │
  │                  │                            │ PG: device_tasks  │                   │
  │                  │                            │                   │                   │
  │                  │ ResultAggregator            │                   │                   │
  │                  │<───────────────────────────│ PG: mml_tasks     │                   │
  │                  │                            │                   │                   │
  │ SSE: completed   │                            │                   │                   │
  │<─────────────────│                            │                   │                   │
```

---

## 7. 消息订阅完整流程

### 7.1 EventBus 事件发布-订阅矩阵

#### 设备事件（ACS → Worker/APP）

| Subject | 触发时机 | 订阅者 | 订阅队列 |
|---------|---------|--------|---------|
| `device.inform.bootstrap` | 新设备首次 Inform | `device.InformHandler` | `device-mgr-bootstrap` |
| `device.inform.periodic` | 周期性心跳 | `device.InformHandler` | `device-mgr-periodic` |
| `device.inform.value_change` | 参数变更通知 | `device.InformHandler` | `device-mgr-valuechange` |
| `device.inform.alarm` | 告警事件 | `alarm.AlarmReceiver` | `alarm-workers` |
| `device.inform.transfer_complete` | TransferComplete | `software.SoftwareService` | (fan-out) |
| `device.inform.autonomous_transfer_complete` | CPE 自动上传 | `transfer.Bridge` | `transfer-bridge` |
| `device.inform.reboot_complete` | Reboot 完成 | (预留) | — |
| `device.inform.connection_request` | Connection Request | (预留) | — |
| `device.registered` | 新设备注册 | `provision.Engine` | `provisioning` |
| `device.reboot.abnormal` | 异常重启 | (预留) | — |
| `device.connection.lost` | 连接丢失 | (预留) | — |
| `device.offline` | 设备离线 | (预留) | — |

**发布者**: ACS `handler.go` (Inform 事件)、`device.Service.RegisterDevice` (`device.registered`)、`device.OfflineDetector` (`device.offline`)

#### 命令响应事件（ACS 发布）

| Subject | 触发时机 | 订阅者 | 订阅队列 |
|---------|---------|--------|---------|
| `command.get_parameters.response` | GPV 响应 | `provision.Engine` | `provision-gpv` |
| `command.set_parameters.response` | SPV 响应 | (预留) | — |
| `command.get_names.response` | GPN 响应 | `provision.Engine` | `provision-gpn` |
| `command.add_object.response` | AddObject 响应 | (预留) | — |
| `command.delete_object.response` | DeleteObject 响应 | (预留) | — |
| `command.download.response` | Download 响应 | (预留) | — |
| `command.upload.response` | Upload 响应 | (预留) | — |
| `command.reboot.response` | Reboot 响应 | (预留) | — |
| `command.factory_reset.response` | FactoryReset 响应 | (预留) | — |
| `command.get_attrs.response` | GetAttrs 响应 | (预留) | — |
| `command.set_attrs.response` | SetAttrs 响应 | (预留) | — |

**注意**: 另有 6 个命令请求事件（`command.get_parameters` 等）已定义常量，标注为"reserved for future use"，当前无发布者和订阅者。

#### PM 数据管线（Worker 内部）

| Subject | 发布者 | 订阅者 | 订阅队列 |
|---------|-------|--------|---------|
| `pm.file.received` | `transfer.Bridge` | `pm.Collector` | `pm-workers` |
| `pm.file.parsed` | `pm.Collector` | (预留) | — |

#### MR 数据管线（Worker 内部）

| Subject | 发布者 | 订阅者 | 订阅队列 |
|---------|-------|--------|---------|
| `mr.file.received` | `transfer.Bridge` | `mr.Collector` | (QueueSubscribe) |
| `mr.file.parsed` | `mr.Collector` | (预留) | — |

#### 告警事件

| Subject | 发布者 | 订阅者 | 进程 |
|---------|-------|--------|------|
| `alarm.raised` | `alarm.AlarmEngine` | `northbound.PushEngine` | Worker → APP |
| `alarm.cleared` | `alarm.AlarmEngine` | `northbound.PushEngine` | Worker → APP |
| `alarm.acknowledged` | `alarm.AlarmEngine` | `northbound.PushEngine` | Worker → APP |

#### 备份与报表（APP → Worker）

| Subject | 发布者 | 订阅者 | 订阅队列 |
|---------|-------|--------|---------|
| `backup.task.created` | `backup.Service` | `backup.Executor` | `backup-executors` |
| `backup.task.done` | `backup.Executor` | (预留) | — |
| `report.generate.requested` | `report.Service` | `report.Generator` | `report-generators` |
| `report.generate.done` | `report.Generator` | (预留) | — |

#### 数据模型（ACS/Worker → APP）

| Subject | 发布者 | 订阅者 | 订阅队列 |
|---------|-------|--------|---------|
| `datamodel.file.received` | `acs/upload/handler.go` | `provision.Engine` | `provision-model-upload` |
| `datamodel.upload.requested` | `provision.ModelUploadService` | (预留) | — |
| `datamodel.upload.completed` | `provision.ModelUploadService` | (预留) | — |
| `datamodel.upload.failed` | `provision.ModelUploadService` | (预留) | — |

#### 自动开站（Worker 内部）

| Subject | 发布者 | 订阅者 | 说明 |
|---------|-------|--------|------|
| `provision.started` | `provision.Engine` | (预留) | 开站流程启动 |
| `provision.completed` | `provision.Engine` | (预留) | 开站完成 |
| `provision.failed` | `provision.Engine` | (预留) | 开站失败 |
| `provision.step.done` | `provision.Engine` | `provision.Engine` | 自订阅驱动下一步 |

#### 固件升级

| Subject | 发布者 | 订阅者 | 说明 |
|---------|-------|--------|------|
| `firmware.uploaded` | `software.Service` | (预留) | 固件上传完成 |
| `upgrade.started` | `software.Service` | (预留) | 升级启动 |
| `upgrade.completed` | `software.Service` | (预留) | 升级完成 |
| `upgrade.failed` | `software.Service` | (预留) | 升级失败 |

#### 北向/OSS 接口（Worker → APP）

| Subject | 发布者 | 订阅者 | 说明 |
|---------|-------|--------|------|
| `oss.alarm.forward` | `alarm.Engine` | `northbound.PushEngine` | 告警北向转发 |
| `oss.pm.export` | `pm.Collector` | `northbound.PushEngine` | PM 数据北向导出 |
| `oss.config.snapshot` | `provision.Engine` | `northbound.PushEngine` | 配置快照北向推送 |

#### 网元直连

| Subject | 发布者 | 订阅者 | 说明 |
|---------|-------|--------|------|
| `nedirect.register` | `nedirect.Service` | (预留) | 网关注册 |
| `nedirect.fault` | `nedirect.Service` | (预留) | 网元故障 |
| `nedirect.session.connect` | `nedirect.Service` | (预留) | 会话建立 |
| `nedirect.session.disconnect` | `nedirect.Service` | (预留) | 会话断开 |
| `nedirect.command.sent` | `nedirect.Service` | (预留) | 命令下发 |

### 7.2 NATS JetStream 流定义

`DefaultStreams()` 定义 8 个持久化流（WorkQueue 保留策略，72 小时最大有效期，文件存储）：

| Stream | 通配符主题 | 涵盖的事件 |
|--------|-----------|-----------|
| `DEVICE` | `device.>` | 所有设备事件 (12 个) |
| `COMMAND` | `command.>` | 所有命令响应 (11 个) |
| `PM` | `pm.>` | PM 文件事件 (2 个) |
| `MR` | `mr.>` | MR 文件事件 (2 个) |
| `ALARM` | `alarm.>` | 告警事件 (3 个) |
| `OSS` | `oss.>` | 北向接口事件 (3 个) |
| `PROVISION` | `provision.>` | 自动开站事件 (4 个) |
| `DATAMODEL` | `datamodel.>` | 数据模型事件 (4 个) |

**注意**: `firmware.*`、`upgrade.*`、`backup.*`、`report.*`、`nedirect.*` 事件目前未纳入 JetStream 流定义，发布时需确保 NATS 有对应流或使用即时发布。

**事件常量定义文件**: `internal/core/event/subjects.go` (57 个常量) + `internal/device/offline_detector.go` 中硬编码 `"device.offline"` (1 个) = **共 58 个事件主题**。

### 7.2 SSE 实时推送（APP → 前端）

| 事件类型 | 发布者 | 目标 | 数据 |
|----------|-------|------|------|
| `mml_task_status` | `mml.Service` | task.Executor | `{task_id, old_status, new_status}` |
| `mml_task_completed` | `mml.ResultAggregator` | task.Executor | `{task_id, status, result, success_count, failed_count}` |
| `notification` | `notification.Service` | notif.UserID | 通知内容 |

前端通过 `EventSource` 连接 `GET /api/v1/events/stream?token={jwt}`，支持 `Last-Event-ID` 重连回放。

---

## 8. 项目代码结构与各层职责

### 8.1 分层架构

```
Handler 层 (HTTP)          路由分发、请求解析、响应序列化
    │
Service 层 (业务逻辑)       核心业务规则、状态机、扇出/聚合
    │
Repository 层 (持久化)      SQL 构建 (Squirrel) + pgx 执行
    │
Infrastructure 层           PostgreSQL / Redis / NATS / MinIO 连接管理
```

### 8.2 各模块职责

| 层 | 文件模式 | 职责 | 示例 |
|----|---------|------|------|
| **Handler** | `handler.go` | HTTP 请求解析、参数校验、响应格式化 | `mml/handler.go` |
| **Service** | `service.go` | 业务逻辑编排、状态转换、事件发布 | `mml/service.go`, `task/service.go` |
| **Repository** | `pg_repository.go` | Squirrel SQL 构建 + pgx 执行 | `mml/pg_repository.go`, `task/pg_repository.go` |
| **Model** | `model.go` | 数据结构定义、状态常量、工厂方法 | `mml/model.go`, `task/model.go` |
| **Queue** | `redis_queue.go` | Redis 运行时队列操作 | `task/redis_queue.go` |
| **Adapter** | `fanout.go`, `bridge_queue.go` | 模块间桥接与适配 | `mml/fanout.go` |

### 8.3 请求处理完整调用链

以 MML 命令执行为例：

```
Gin Router                          cmd/app/provider/router.go:339
  │  permGroup("devices") → MML Handler
  v
MML Handler.Execute()               internal/mml/handler.go
  │  ShouldBindJSON → ExecuteHTTPRequest
  │  提取 username (JWT middleware)
  v
MML Service.ExecuteCommand()        internal/mml/service.go
  │  解析 command_code/script_id
  │  构建 MMLTask
  │  taskRepo.Create()               ← PgTaskRepository (mml_tasks)
  │  writeAuditLogs()                ← AuditRepository (mml_audit_logs)
  │  fanouter.Fanout()               ← Fanouter → TaskService (device_tasks)
  │  hub.PublishSimple()             ← SSE MessageHub
  v
Task Service.BatchCreateTasks()     internal/task/service.go
  │  repo.BatchCreate()              ← PgTaskRepository (device_tasks)
  │  queue.Push() × N                ← RedisTaskQueue (Redis Sorted Set)
  │  wakeDevice() × M                ← connreq.Dispatcher (UDP/HTTP)
  v
ACS Handler.PopTask()                internal/acs/handler.go
  │  Redis ZPOPMIN
  │  RPCDispatcher.BuildRequest()    ← SOAP Template 渲染
  v
ACS Handler.MarkTaskCompleted()      internal/acs/handler.go
  │  Redis Hash 更新 + PG 同步
  │  notifyCompletion()              ← ResultAggregator 回调
  v
ResultAggregator                    internal/mml/result_aggregator.go
  │  IncrementStats()                ← PgTaskRepository (mml_tasks)
  │  finalizeIfComplete()
  │  hub.PublishSimple()             ← SSE → 前端
```

---

## 9. 关键文件索引

| 模块 | 文件路径 |
|------|---------|
| **ACS 入口** | `cmd/acs/main.go`, `cmd/acs/bootstrap.go` |
| **ACS HTTP 服务器** | `internal/acs/server.go` |
| **ACS 请求处理** | `internal/acs/handler.go` |
| **会话状态机** | `internal/acs/session.go`, `internal/acs/session_store.go` |
| **SOAP 编解码** | `pkg/soap/decoder.go`, `pkg/soap/templates.go`, `pkg/soap/envelope.go` |
| **RPC 分发** | `internal/acs/rpc/dispatcher.go` |
| **CPE 认证** | `internal/acs/auth/authenticator.go` |
| **流量控制** | `internal/acs/ratelimit.go`, `internal/acs/admission.go` |
| **任务队列** | `internal/task/service.go`, `internal/task/redis_queue.go`, `internal/task/pg_repository.go` |
| **任务模型** | `internal/task/model.go` |
| **旧队列桥接** | `internal/task/bridge_queue.go` |
| **MML 扇出** | `internal/mml/fanout.go`, `internal/mml/result_aggregator.go` |
| **MML 服务** | `internal/mml/service.go`, `internal/mml/handler.go` |
| **Connection Request** | `internal/acs/connreq/dispatcher.go`, `internal/acs/connreq/udp_sender.go` |
| **EventBus** | `internal/core/event/bus.go`, `internal/core/event/nats_bus.go`, `internal/core/event/subjects.go` |
| **SSE 推送** | `internal/events/hub.go`, `internal/events/handler.go` |
| **APP 入口** | `cmd/app/main.go`, `cmd/app/provider/router.go`, `cmd/app/provider/modules.go` |
| **Worker 入口** | `cmd/worker/main.go` |
