# 三服务事件驱动流程：ACS → App → Worker

> 完整描述 omcgo 三个部署单元（ACS、App、Worker）之间通过 NATS JetStream EventBus 进行事件驱动通信的全链路实现。

---

## 总览

```
                         NATS JetStream (8 Streams)
                    ┌─────────────────────────────────────┐
                    │  DEVICE  COMMAND  PM  MR  ALARM     │
                    │  OSS  PROVISION  DATAMODEL           │
                    └─────────────────────────────────────┘
                      ↑ Publish          ↓ Subscribe
          ┌───────────┴───────┬──────────┴────────────┐
          │                   │                        │
    omcgo-acs (:7547)   omcgo-app (:8080)      omcgo-worker
    [事件生产者]         [事件消费+生产]        [事件消费+生产]
    ─────────────       ──────────────         ─────────────
    Inform 事件发布     设备注册/更新          PM 文件解析
    RPC 响应事件发布    自动开站               MR 文件解析
    文件传输事件发布    固件升级               告警处理
                        北向推送               配置备份
                                               报表生成
                                               文件传输桥接
```

---

## 第一部分：EventBus 基础设施

### 1.1 核心接口

**文件**: `internal/core/event/bus.go`

```go
type EventBus interface {
    Publish(ctx context.Context, subject string, event Event) error
    Subscribe(subject string, handler EventHandler) (Subscription, error)
    QueueSubscribe(subject string, queue string, handler EventHandler) (Subscription, error)
    Close() error
}

type EventHandler func(ctx context.Context, event Event) error
```

### 1.2 Event 数据结构

**文件**: `internal/core/event/types.go`

```go
type Event struct {
    ID        string            `json:"id"`         // UUID，NewEvent() 自动生成
    Subject   string            `json:"subject"`    // 如 "device.inform.bootstrap"
    Payload   json.RawMessage   `json:"payload"`    // JSON 编码的事件负载
    Metadata  map[string]string `json:"metadata"`   // 可选：correlation ID 等
    Timestamp time.Time         `json:"timestamp"`  // 事件创建时间
}
```

- `NewEvent(subject, payload)` → 自动分配 UUID + 序列化 payload
- `event.DecodePayload(&target)` → 反序列化到目标结构体

### 1.3 Subject 常量定义

**文件**: `internal/core/event/subjects.go` — 共 54 个常量，覆盖 11 个功能域

#### Device 事件（ACS 发布）

| 常量 | Subject | 含义 |
|------|---------|------|
| `SubjectDeviceBootstrap` | `device.inform.bootstrap` | 新设备首次上线 (0-BOOTSTRAP / 1-BOOT) |
| `SubjectDevicePeriodic` | `device.inform.periodic` | 周期心跳 (2-PERIODIC) |
| `SubjectDeviceValueChange` | `device.inform.value_change` | 参数变更 (4-VALUE CHANGE) |
| `SubjectDeviceAlarm` | `device.inform.alarm` | 告警事件 (6-ALARM) |
| `SubjectDeviceTransferComplete` | `device.inform.transfer_complete` | 文件传输完成 (5-TRANSFER COMPLETE) |
| `SubjectDeviceAutonomousTransferComplete` | `device.inform.autonomous_transfer_complete` | 设备主动传输完成 |
| `SubjectDeviceRebootComplete` | `device.inform.reboot_complete` | 重启完成 (8-REBOOT) |
| `SubjectDeviceConnectionRequest` | `device.inform.connection_request` | 连接请求 (7-CONNECTION REQUEST) |
| `SubjectDeviceConnectionLost` | `device.connection.lost` | 设备会话超时 |
| `SubjectDeviceRegistered` | `device.registered` | 设备注册完成（App 发布） |

#### Command 事件

| 常量 | Subject |
|------|---------|
| `SubjectCommandGetParams` | `command.get_parameters` |
| `SubjectCommandSetParams` | `command.set_parameters` |
| `SubjectCommandDownload` | `command.download` |
| `SubjectCommandUpload` | `command.upload` |
| `SubjectCommandReboot` | `command.reboot` |
| `SubjectCommandReset` | `command.factory_reset` |

#### Command Response 事件（ACS 发布）

| 常量 | Subject |
|------|---------|
| `SubjectCommandGetParamsResponse` | `command.get_parameters.response` |
| `SubjectCommandSetParamsResponse` | `command.set_parameters.response` |
| `SubjectCommandDownloadResponse` | `command.download.response` |
| `SubjectCommandUploadResponse` | `command.upload.response` |
| `SubjectCommandGetNamesResponse` | `command.get_names.response` |
| `SubjectCommandAddObjectResponse` | `command.add_object.response` |
| `SubjectCommandDeleteObjectResponse` | `command.delete_object.response` |
| `SubjectCommandRebootResponse` | `command.reboot.response` |
| `SubjectCommandFactoryResetResponse` | `command.factory_reset.response` |
| `SubjectCommandGetAttrsResponse` | `command.get_attrs.response` |
| `SubjectCommandSetAttrsResponse` | `command.set_attrs.response` |

#### PM / MR / Alarm / Provision / DataModel / Firmware / Backup / Report / OSS / NEDirect

| Subject | 含义 |
|---------|------|
| `pm.file.received` / `pm.file.parsed` | PM 文件接收/解析完成 |
| `mr.file.received` / `mr.file.parsed` | MR 文件接收/解析完成 |
| `alarm.raised` / `alarm.cleared` / `alarm.acknowledged` | 告警生命周期 |
| `provision.started` / `provision.completed` / `provision.failed` / `provision.step.done` | 自动开站 |
| `datamodel.upload.requested` / `.completed` / `.failed` / `datamodel.file.received` | 数据模型 |
| `firmware.uploaded` / `upgrade.started` / `upgrade.completed` / `upgrade.failed` | 固件升级 |
| `backup.task.created` / `backup.task.done` | 配置备份 |
| `report.generate.requested` / `report.generate.done` | 报表生成 |
| `oss.alarm.forward` / `oss.pm.export` / `oss.config.snapshot` | 北向推送 |
| `nedirect.register` / `nedirect.fault` / `nedirect.session.connect` / `.disconnect` / `nedirect.command.sent` | 网元直连 |

### 1.4 NATS JetStream 实现

**文件**: `internal/core/event/nats_bus.go`

#### Publish 流程

```go
func (b *NATSEventBus) Publish(ctx context.Context, subject string, evt Event) error {
    evt.Subject = subject
    data, err := json.Marshal(evt)        // JSON 序列化
    _, err = b.js.Publish(subject, data)  // JetStream 持久化发布
}
```

#### wrapHandler — 重试与错误处理核心（L101-135）

```go
const maxDeliveries = 5

func (b *NATSEventBus) wrapHandler(handler EventHandler) nats.MsgHandler {
    return func(msg *nats.Msg) {
        var evt Event
        if err := json.Unmarshal(msg.Data, &evt); err != nil {
            msg.Term()  // JSON 解析失败 → 永久放弃（防止 poison pill）
            return
        }

        if err := handler(b.ctx, evt); err != nil {
            meta, _ := msg.Metadata()
            deliveries := meta.NumDelivered

            if deliveries >= maxDeliveries {
                msg.Term()  // 达到最大重试 → 永久放弃
            } else {
                backoff := time.Duration(1<<(deliveries-1)) * time.Second
                msg.NakWithDelay(backoff)  // 指数退避重试
                // 第1次: 1s, 第2次: 2s, 第3次: 4s, 第4次: 8s, 第5次: Term
            }
            return
        }
        msg.Ack()  // 处理成功 → 确认消息
    }
}
```

#### QueueSubscribe — 负载均衡消费

```go
func (b *NATSEventBus) QueueSubscribe(subject, queue string, handler EventHandler) {
    b.js.QueueSubscribe(subject, queue, b.wrapHandler(handler),
        nats.Durable(queue),      // 持久化消费者（重启后继续消费）
        nats.AckExplicit(),       // 必须显式 Ack/Nak/Term
    )
}
```

### 1.5 JetStream Stream 配置

**文件**: `internal/core/components/nats/nats.go`

```go
func DefaultStreams() []StreamDef {
    return []StreamDef{
        {Name: "DEVICE",     Subjects: []string{"device.>"}},
        {Name: "COMMAND",    Subjects: []string{"command.>"}},
        {Name: "PM",         Subjects: []string{"pm.>"}},
        {Name: "MR",         Subjects: []string{"mr.>"}},
        {Name: "ALARM",      Subjects: []string{"alarm.>"}},
        {Name: "OSS",        Subjects: []string{"oss.>"}},
        {Name: "PROVISION",  Subjects: []string{"provision.>"}},
        {Name: "DATAMODEL",  Subjects: []string{"datamodel.>"}},
    }
}
```

每个 Stream 的 JetStream 配置：

| 参数 | 值 | 说明 |
|------|-----|------|
| Retention | `WorkQueuePolicy` | 消息被所有消费者 ACK 后删除 |
| MaxAge | `72h` | 72 小时后自动清理未消费消息 |
| Storage | `FileStorage` | 磁盘持久化 |
| Replicas | `1` | 单副本（生产可改为 3） |

### 1.6 Channel 实现（开发/测试环境）

**文件**: `internal/core/event/channel_bus.go`

| 特性 | 说明 |
|------|------|
| 缓冲区 | 每订阅者 256 个事件（默认） |
| 通配符 | 支持 NATS 风格 `*`（单段）和 `>`（多段尾匹配） |
| Queue 负载均衡 | 同 queue group 内 round-robin |
| 重试 | 无（handler 错误仅日志记录） |
| 持久化 | 无（进程退出即丢失） |

### 1.7 三服务初始化

**文件**: `internal/core/components/infra.go:156-159`

```go
func (inf *Infra) CreateEventBus() {
    inf.EventBus = event.NewNATSEventBus(inf.NATS.Conn, inf.NATS.JS, inf.Logger)
    inf.GS.Register("eventbus", 2, func(ctx context.Context) error {
        return inf.EventBus.Close()
    })
}
```

三个服务（ACS/App/Worker）各自初始化独立的 `NATSEventBus` 实例，连接同一 NATS 集群。

优雅关闭优先级：`1=HTTP` → `2=NATS+EventBus` → `3=Redis` → `4=PostgreSQL` → `5=Tracer`

---

## 第二部分：omcgo-acs（事件生产者）

ACS 是事件的**主要生产者**，负责将 CPE 设备的 SOAP/XML Inform 解析后转换为 NATS 事件。

### 2.1 Inform 事件发布

**文件**: `internal/acs/handler.go`

#### 处理流程

```
ServeHTTP() (L184)
  → handleInform() (L270)
    → 解析 SOAP XML → InformMessage
    → 速率限制 (per-device rate.Limiter)
    → 准入控制 (AdmissionController: 限制并发会话数)
    → 创建 Session (Redis Hash, TTL 5min)
    → 缓存 ConnectionRequestURL (Redis)
    → publishInformEvents() (L1140-1194)     ★ 事件发布核心
    → 返回 InformResponse (SOAP XML)
```

#### publishInformEvents() 实现 (L1140-1194)

```go
func (h *Handler) publishInformEvents(ctx context.Context, inform *tr069.InformMessage) {
    payload := InformEventPayload{
        DeviceId:      inform.DeviceId,           // OUI + SN + ProductClass + Manufacturer
        Events:        tr069.EventCodes(inform),  // 事件码列表
        ParameterList: inform.ParameterList,      // 参数键值对
        CurrentTime:   inform.CurrentTime,
        RetryCount:    inform.RetryCount,
    }

    // 根据事件码确定 subject
    for _, eventCode := range inform.Event {
        switch eventCode.EventCode {
        case "0 BOOTSTRAP", "1 BOOT":
            h.eventBus.Publish(ctx, event.SubjectDeviceBootstrap, ...)
        case "2 PERIODIC":
            h.eventBus.Publish(ctx, event.SubjectDevicePeriodic, ...)
        case "4 VALUE CHANGE":
            h.eventBus.Publish(ctx, event.SubjectDeviceValueChange, ...)
        case "6 ALARM":
            h.eventBus.Publish(ctx, event.SubjectDeviceAlarm, ...)
        case "5 TRANSFER COMPLETE":
            h.eventBus.Publish(ctx, event.SubjectDeviceTransferComplete, ...)
        case "7 CONNECTION REQUEST":
            h.eventBus.Publish(ctx, event.SubjectDeviceConnectionRequest, ...)
        case "M Reboot":
            h.eventBus.Publish(ctx, event.SubjectDeviceRebootComplete, ...)
        default:
            h.eventBus.Publish(ctx, event.SubjectDevicePeriodic, ...)
        }
    }
}
```

#### InformEventPayload 结构

```go
type InformEventPayload struct {
    DeviceId      tr069.DeviceIdStruct        // {OUI, SerialNumber, ProductClass, Manufacturer}
    Events        []string                    // ["0 BOOTSTRAP", "2 PERIODIC", ...]
    ParameterList []tr069.ParameterValueStruct // [{Name, Value}, ...]
    CurrentTime   string                      // 设备当前时间
    RetryCount    int                         // Inform 重试次数
}
```

### 2.2 RPC 响应事件发布

ACS 执行完 RPC 方法后，将 CPE 响应作为事件发布，供 App 的 Provision Engine 消费。

**文件**: `internal/acs/handler.go` — RPC 响应处理

```
handleRPCResponse()
  → 解析 CPE 返回的 SOAP Response
  → 根据 RPC 方法类型发布对应事件：
    GetParameterValuesResponse  → "command.get_parameters.response"
    SetParameterValuesResponse  → "command.set_parameters.response"
    GetParameterNamesResponse   → "command.get_names.response"
    DownloadResponse            → "command.download.response"
    UploadResponse              → "command.upload.response"
    AddObjectResponse           → "command.add_object.response"
    DeleteObjectResponse        → "command.delete_object.response"
    RebootResponse              → "command.reboot.response"
    FactoryResetResponse        → "command.factory_reset.response"
```

### 2.3 自主传输完成事件

当 CPE 主动上传文件（非 RPC 触发）时：

```
handleInform() → 识别 "10 AUTONOMOUS TRANSFER COMPLETE" 事件码
  → eventBus.Publish("device.inform.autonomous_transfer_complete", payload)
```

### 2.4 ACS 发布事件汇总

| Subject | 触发时机 | 消费者 |
|---------|---------|--------|
| `device.inform.bootstrap` | CPE 首次 Inform (0/1) | App: InformHandler |
| `device.inform.periodic` | CPE 心跳 Inform (2) | App: InformHandler |
| `device.inform.value_change` | CPE 参数变更 (4) | App: InformHandler |
| `device.inform.alarm` | CPE 告警 (6) | Worker: AlarmReceiver |
| `device.inform.transfer_complete` | 文件传输完成 (5) | App: SoftwareService |
| `device.inform.autonomous_transfer_complete` | CPE 主动传输 (10) | Worker: TransferBridge |
| `device.inform.reboot_complete` | CPE 重启完成 (8) | — |
| `device.inform.connection_request` | 连接请求 (7) | — |
| `command.*.response` (11 种) | RPC 响应 | App: ProvisionEngine |

---

## 第三部分：omcgo-app（事件消费 + 二级事件生产）

App 服务在启动时注册 4 组事件订阅，同时也是二级事件的生产者。

### 3.1 订阅注册入口

**文件**: `cmd/app/router/router.go` — `Setup()` 函数

```go
// L87: 设备 Inform 事件处理
informHandler := device.NewInformHandler(deviceService, carrierRegistry, model.CarrierCMCC, logger)
informHandler.Subscribe(eventBus)

// L153: 自动开站引擎
provisionEngine := provision.NewProvisioningEngine(...)
provisionEngine.Subscribe(eventBus)

// L196: 固件升级服务
softwareService := software.NewSoftwareService(...)
softwareService.Subscribe(eventBus)

// L438: 北向推送引擎
pushEngine := push.NewEngine(cfg.Northbound.PushTargets, logger)
pushEngine.Subscribe(eventBus)
```

### 3.2 InformHandler — 设备注册与更新

**文件**: `internal/device/inform_handler.go`

#### 订阅注册 (L47-66)

```go
func (h *InformHandler) Subscribe(bus event.EventBus) error {
    bus.QueueSubscribe("device.inform.bootstrap",    "device-mgr-bootstrap",   h.handleBootstrap)
    bus.QueueSubscribe("device.inform.periodic",     "device-mgr-periodic",    h.handlePeriodic)
    bus.QueueSubscribe("device.inform.value_change", "device-mgr-valuechange", h.handlePeriodic)
    return nil
}
```

#### handleBootstrap — 新设备注册 (L69-120)

```
handleBootstrap(ctx, event)
  1. event.DecodePayload(&payload)         // 解码 InformEventPayload
  2. h.resolveCarrier(payload.DeviceId.OUI)  // CarrierRegistry.ResolveByOUI → 运营商
  3. h.service.RegisterFromInform(ctx, inform, carrierCode)
     │
     │  RegisterFromInform (service.go:201-317):
     │  ├─ deviceRepo.GetBySerialNumber(sn)         // 查重
     │  ├─ if 已存在 → 调用 UpdateFromInform        // 走更新路径
     │  ├─ 构建 Device model:
     │  │   ID: uuid.New()
     │  │   SerialNumber, OUI, ProductClass, Manufacturer
     │  │   Carrier (从参数推断), Technology (LTE/NR)
     │  │   Status = Active
     │  │   FirmwareVersion, ConnectionRequestURL, IPAddress
     │  │   LastInformAt = now, LastInformEvents
     │  ├─ deviceRepo.Create(ctx, device)            // PostgreSQL INSERT
     │  ├─ storeInformParameters(ctx, id, params)    // device_parameters INSERT
     │  └─ heartbeat.RefreshHeartbeat()              // Redis 心跳
     │
  4. h.service.PublishDeviceRegistered(ctx, device)   // ★ 发布二级事件
```

**DB 写入**: `devices` 表 INSERT + `device_parameters` 表 BatchUpsert

#### handlePeriodic — 设备更新 (L122-181)

```
handlePeriodic(ctx, event)
  1. event.DecodePayload(&payload)
  2. h.service.UpdateFromInform(ctx, inform)
     │
     │  UpdateFromInform (service.go:320-399):
     │  ├─ deviceRepo.GetBySerialNumber(sn)
     │  ├─ if device == nil → return (nil, nil)     // 设备不存在，不报错
     │  ├─ 更新字段: OUI, ProductClass, FirmwareVersion,
     │  │   ConnectionRequestURL, IPAddress, LastInformAt
     │  ├─ 自动状态转换: Discovered/Offline/Registered → Active
     │  ├─ deviceRepo.Update(ctx, device)            // PostgreSQL UPDATE
     │  ├─ storeInformParameters(ctx, id, params)
     │  └─ heartbeat.RefreshHeartbeat()
     │
  3. if device == nil → 回退到自动注册:
     ├─ h.resolveCarrier(oui)
     ├─ h.service.RegisterFromInform(ctx, inform, carrier)
     ├─ h.service.PublishDeviceRegistered(ctx, device)
     └─ return nil    // ACK 消息，不触发 NATS 重试
```

**DB 写入**: `devices` 表 UPDATE + `device_parameters` 表 BatchUpsert

**错误处理**: 设备不存在时回退到注册（不返回 error，避免 NATS 无限重试）

### 3.3 ProvisioningEngine — 自动开站

**文件**: `internal/provision/engine.go`

#### 订阅注册 (L85-131)

```go
func (e *ProvisioningEngine) Subscribe(bus event.EventBus) error {
    // 新设备注册 → 触发自动开站流程
    bus.QueueSubscribe("device.registered", "provisioning", func(...) {
        e.HandleBootstrap(ctx, bootstrapEvent)
    })
    // 数据模型文件上传 → 参数发现
    bus.QueueSubscribe("datamodel.file.received", "provision-model-upload", func(...) {
        e.handleDataModelFileReceived(ctx, evt)
    })
    // GetParameterValues RPC 响应 → 参数同步
    bus.QueueSubscribe("command.get_parameters.response", "provision-gpv", func(...) {
        e.handleGPVResponse(ctx, evt)
    })
    // GetParameterNames RPC 响应 → 两阶段同步
    bus.QueueSubscribe("command.get_names.response", "provision-gpn", func(...) {
        e.handleGPNResponse(ctx, evt)
    })
    return nil
}
```

#### HandleBootstrap 处理流程

```
HandleBootstrap(ctx, bootstrapEvent)
  → 解码 {device_id, serial_number, oui, product_class, carrier, technology}
  → 三条并行路径（根据配置启用）:
    Path A: Auto-Configure — 模板匹配 → 参数下发 → 配置激活
    Path B: Auto-Sync — 参数发现 → 同步到设备
    Path C: Auto-Discovery — 参数模型获取
```

**DB 写入**: `provisioning_tasks` 表（状态跟踪）、`parameter_discovery_logs` 表、`device_parameters` 表

### 3.4 SoftwareService — 固件升级

**文件**: `internal/software/service.go`

#### 订阅注册 (L320-324)

```go
func (s *SoftwareService) Subscribe(bus event.EventBus) error {
    bus.Subscribe("device.inform.transfer_complete", s.HandleTransferComplete)
    return nil
}
```

注意：使用 `Subscribe`（fan-out）而非 `QueueSubscribe`（负载均衡）。

#### HandleTransferComplete 处理流程

```
HandleTransferComplete(ctx, evt)
  → 解码 TransferComplete 事件
  → 查找对应的 upgrade_tasks 记录
  → 更新任务状态 → UpgradeCompleted
  → eventBus.Publish("upgrade.completed", ...)    // ★ 发布二级事件
```

**DB 写入**: `upgrade_tasks` 表 UPDATE

### 3.5 PushEngine — 北向 OSS 推送

**文件**: `internal/northbound/push/engine.go`

#### 订阅注册 (L74-102)

```go
func (e *Engine) Subscribe(bus event.EventBus) error {
    subjects := []string{
        "oss.alarm.forward",     // 告警转发
        "oss.pm.export",         // PM 数据导出
        "oss.config.snapshot",   // 配置快照
    }
    for _, subj := range subjects {
        bus.Subscribe(subj, e.handleEvent)  // fan-out: 所有实例均接收
    }
    return nil
}
```

#### 处理流程

```
handleEvent(ctx, evt) / EnqueueEvent(ctx, evt)
  → 根据配置模式:
    Direct Mode: 立即 HTTP 推送到 OSS 目标
    Outbox Mode: 写入 oss_events_outbox 表 → OutboxWorker 异步投递
  → 每目标配置:
    - 重试次数 (RetryCount)
    - 熔断器 (Circuit Breaker)
    - 批量大小
    - 认证方式 (Bearer Token / HMAC)
```

### 3.6 App 发布的二级事件汇总

| Subject | 来源 | 触发时机 |
|---------|------|---------|
| `device.registered` | DeviceService.PublishDeviceRegistered | Bootstrap 注册完成后 |
| `firmware.uploaded` | SoftwareService.UploadFirmware | 固件包上传 |
| `upgrade.started` | SoftwareService.StartUpgrade | 下发升级 RPC |
| `upgrade.completed` | SoftwareService.HandleTransferComplete | 设备确认升级完成 |

### 3.7 App 订阅清单汇总

#### QueueSubscribe（负载均衡，多实例部署时只有一个实例处理）

| Subject | Queue Group | Handler | 来源 |
|---------|-------------|---------|------|
| `device.inform.bootstrap` | `device-mgr-bootstrap` | InformHandler.handleBootstrap | ACS |
| `device.inform.periodic` | `device-mgr-periodic` | InformHandler.handlePeriodic | ACS |
| `device.inform.value_change` | `device-mgr-valuechange` | InformHandler.handlePeriodic | ACS |
| `device.registered` | `provisioning` | ProvisionEngine.HandleBootstrap | App 自身 |
| `datamodel.file.received` | `provision-model-upload` | ProvisionEngine.handleDataModelFileReceived | Worker |
| `command.get_parameters.response` | `provision-gpv` | ProvisionEngine.handleGPVResponse | ACS |
| `command.get_names.response` | `provision-gpn` | ProvisionEngine.handleGPNResponse | ACS |

#### Subscribe（fan-out，所有实例均接收）

| Subject | Handler | 来源 |
|---------|---------|------|
| `device.inform.transfer_complete` | SoftwareService.HandleTransferComplete | ACS |
| `oss.alarm.forward` | PushEngine.handleEvent | App/Worker |
| `oss.pm.export` | PushEngine.handleEvent | App/Worker |
| `oss.config.snapshot` | PushEngine.handleEvent | App/Worker |

---

## 第四部分：omcgo-worker（后台事件处理）

Worker 服务负责 CPU/IO 密集的后台任务：文件解析、告警处理、备份执行、报表生成。

### 4.1 初始化与订阅注册

**文件**: `cmd/worker/main.go`

```go
func runWorker() {
    // 基础设施初始化 (bootstrap.go:24-56)
    inf := NewInfra(...)
    inf.ConnectPostgres(...)   // PostgreSQL
    inf.ConnectTimescale(...)  // TimescaleDB
    inf.ConnectRedis(...)      // Redis
    inf.ConnectNATS(...)       // NATS JetStream
    inf.CreateEventBus()       // EventBus

    // 注册订阅 (main.go:69-147)
    registerSubscribers(inf)

    // ���塞等待 SIGTERM/SIGINT
    inf.WaitAndShutdown()
}
```

### 4.2 TransferBridge — 文件传输桥接

**文件**: `internal/transfer/bridge.go`

**订阅**: `device.inform.autonomous_transfer_complete` → Queue: `transfer-bridge`

TransferBridge 是 Worker 的**入口事件**，负责将 CPE 主动上传的文件分类后转发给对应处理器。

#### 处理流程 (L83-236)

```
handleAutonomousTransferComplete(ctx, evt)
  1. 解码 atcPayload {DeviceSN, TransferURL, FileType, TargetFileName, ...}
  2. 校验: TransferURL 非空, Fault 为 null
  3. deviceRepo.GetBySerialNumber(sn) → 获取设备信息
  4. 分类 FileType:
     │
     ├─ FileType="4" 或 contains "PM"
     │   → HTTP GET 下载文件
     │   → 上传 MinIO (pm-files bucket): {date}/{device_sn}/{filename}
     │   → Publish("pm.file.received", {minio_path, device_id, device_sn, carrier, technology})
     │                                     ↓
     │                              PM Collector 消费
     │
     ├─ FileType="5" 或 contains "MR"
     │   → HTTP GET 下载文件
     │   → 上传 MinIO (mr-files bucket): {date}/{device_sn}/{filename}
     │   → Publish("mr.file.received", {minio_path, bucket, device_id, device_sn, carrier, file_name})
     │                                     ↓
     │                              MR Collector 消费
     │
     ├─ FileType="11" 或 contains "PARAMETER MODEL"
     │   → HTTP GET 下载文件
     │   → 上传 MinIO (logs bucket): datamodel/{date}/{device_sn}/{filename}
     │   → Publish("datamodel.file.received", {minio_bucket, minio_path, device_id, ...})
     │                                     ↓
     │                           App ProvisionEngine 消费
     │
     └─ 其他 (FileType="3" LOG 等)
         → HTTP GET 下载文件
         → 上传 MinIO (logs bucket): {date}/{device_sn}/{filename}
```

**DB 读取**: `devices` 表 (获取设备信息)
**MinIO 写入**: 文件存储到对应 bucket
**二级事件**: `pm.file.received` / `mr.file.received` / `datamodel.file.received`

### 4.3 PM Collector — 性能文件解析

**文件**: `internal/pm/collector/collector.go`

**订阅**: `pm.file.received` → Queue: `pm-workers`

#### 处理流程 (L71-170)

```
handleFileReceived(ctx, evt)
  1. 解码 FileReceivedPayload {MinIOPath, DeviceID, DeviceSN, Carrier, Technology}
  2. MinIO 下载文件 (bucket: pm-files)
  3. 保存文件元数据 → pm_files 表 (PostgreSQL)
  4. PMXMLParser 解析 XML 内容 (3GPP 32.435 格式)
     → 提取计数器: [{cell_id, counter_name, counter_value, period_start, period_end}, ...]
  5. 批量写入 → pm_counters 表 (TimescaleDB hypertable)
  6. 对每个唯一 cell_id:
     → KPIEngine.CalculateAndStore()
     → 计算 KPI 指标 → kpi_values 表 (TimescaleDB hypertable)
  7. 更新 pm_files 记录状态 (parsed/failed)
  8. Publish("pm.file.parsed", {minio_path, device_id, counter_count})
```

**DB 写入**:

| 表 | 类型 | 数据 |
|-----|------|------|
| `pm_files` | PostgreSQL | 文件元数据、解析状态 |
| `pm_counters` | TimescaleDB hypertable | PM 计数器值 |
| `kpi_values` | TimescaleDB hypertable | KPI 计算结果 |

**Prometheus 指标**: `pm_files_processed_total{status}`, `pm_processing_duration_seconds`

### 4.4 MR Collector — 测量报告解析

**文件**: `internal/mr/collector/collector.go`

**订阅**: `mr.file.received` → Queue: `mr-workers`

#### 处理流程 (L76-181)

```
handleFileReceived(ctx, evt)
  1. 解码 MRFilePayload {MinioPath, Bucket, DeviceID, DeviceSN, Carrier, FileName}
  2. DetectMRType(fileName) → MRO / MRS / MRE
  3. 创建文件元数据 → mr_files 表
  4. MinIO 下载文件
  5. 根据类型选择解析器:
     MRO → parser.NewMROParser()   // 覆盖优化
     MRS → parser.NewMRSParser()   // 切换统计
     MRE → parser.NewMREParser()   // 事件触发
  6. 解析 → ParseData {Records: [{cell_id, rsrp, rsrq, sinr, ...}]}
  7. 批量写入 → mr_records 表 (TimescaleDB hypertable)
  8. 更新 mr_files 记录（record_count, 状态）
  9. Publish("mr.file.parsed", {file_id, device_id, mr_type, record_count})
```

**DB 写入**:

| 表 | 类型 | 数据 |
|-----|------|------|
| `mr_files` | PostgreSQL | 文件元数据 |
| `mr_records` | TimescaleDB hypertable | 测量记录 (RSRP/RSRQ/SINR) |

### 4.5 Alarm Receiver — 告警处理

**文件**: `internal/alarm/receiver.go` + `internal/alarm/engine.go`

**订阅**: `device.inform.alarm` → Queue: `alarm-workers`

#### 处理流程 (receiver.go:52-86 → engine.go:54-157)

```
handleAlarmEvent(ctx, evt)
  1. 解码 AlarmPayload {DeviceID, DeviceSN, Carrier, AlarmCode, Severity, ...}
  2. AlarmEngine.Process(ctx, alarm):
     │
     ├─ Carrier.AlarmSeverityMapping(alarmCode) → 运营商级严重度映射
     ├─ Redis 查重: GET acs:alarm:{device_sn}:{alarm_code}
     │   ├─ 已存在 → 更新现有告警，return
     │   └─ 不存在 → 继续创建
     ├─ alarm.ID = uuid.New()
     ├─ alarm.Status = AlarmActive
     ├─ alarmRepo.Create(ctx, alarm)           // → alarms_active 表
     ├─ Redis SET acs:alarm:{device_sn}:{alarm_code} = alarm.ID
     ├─ Metrics: ReceivedTotal[severity].Inc(), ActiveTotal[severity,carrier].Inc()
     └─ Publish("alarm.raised", alarm)         // ★ 二级事件
```

**DB 写入**:

| 表 | 类型 | 数据 |
|-----|------|------|
| `alarms_active` | PostgreSQL | 活跃告警 |
| `alarms_history` | TimescaleDB hypertable | 告警历史（清除/归档时写入） |

**Redis 写入**: `acs:alarm:{device_sn}:{alarm_code}` → 告警 UUID (去重)

**二级事件**:
- `alarm.raised` — 新告警
- `alarm.acknowledged` — 用户确认
- `alarm.cleared` — 告警清除

**Prometheus 指标**: `alarms_received_total{severity}`, `alarms_active{severity,carrier}`

### 4.6 Backup Executor — 配置备份

**文件**: `internal/backup/executor.go`

**订阅**: `backup.task.created` → Queue: `backup-executors`

#### 处理流程 (L67-190)

```
handleTaskCreated(ctx, evt)
  1. 解码 {TaskID}
  2. backupRepo.GetTask(ctx, taskID) → 获取备份任务
  3. 校验 task.Status == TaskPending
  4. 更新 task → TaskRunning, StartedAt = now
  5. for each device_sn in task.TargetIDs:
     │  ├─ deviceRepo.GetBySerialNumber(sn)
     │  ├─ 构建 TR-069 Upload RPC (FileType="2" VendorConfigurationFile)
     │  ├─ Redis ZADD acs:cmdq:{device_sn} → 命令队列
     │  ├─ HTTP POST ConnectionRequestURL → 唤醒设备
     │  └─ 更新 task.Progress = (i+1)*100/total
  6. 更新 task → TaskCompleted/TaskFailed, CompletedAt = now
  7. Publish("backup.task.done", {task_id, status})
```

**DB 写入**: `backup_tasks` 表 (状态、进度、完成时间)
**Redis 写入**: `acs:cmdq:{device_sn}` Sorted Set (Upload 命令)

### 4.7 Report Generator — 报表生成

**文件**: `internal/report/generator.go`

**订阅**: `report.generate.requested` → Queue: `report-generators`

#### 处理流程 (L62-164)

```
handleGenerateRequest(ctx, evt)
  1. 解码 {RecordID, DefinitionID}
  2. reportRepo.GetDefinition(ctx, defID)  → 报表定义
  3. reportRepo.GetRecord(ctx, recordID)   → 报表记录
  4. 按 def.ReportType 采集数据:
     │  Performance → 查询 kpi_values (TimescaleDB)
     │  Alarm       → 查询 alarms_history (TimescaleDB)
     │  Device/Capacity/Security → 通用数据查询
  5. 构建 JSON 报表结构
  6. 上传 MinIO: reports/{def_id}/{period}/{record_id}.json
  7. 更新 record: minio_path, file_size, status = RecordReady
  8. Publish("report.generate.done", {record_id, definition_id, minio_path})
```

**DB 读取**: `report_definitions`, `report_records`, `kpi_values`, `alarms_history`
**DB 写入**: `report_records` 表 (状态、MinIO 路径)
**MinIO 写入**: JSON 报表文件

**错误处理**: 采集/序列化/上传失败时标记 record.Status = RecordFailed，return nil（不重试）

### 4.8 Worker 订阅清单汇总

| Subject | Queue Group | Handler | DB 写入 |
|---------|-------------|---------|---------|
| `device.inform.autonomous_transfer_complete` | `transfer-bridge` | TransferBridge.handleATC | MinIO |
| `pm.file.received` | `pm-workers` | PMCollector.handleFileReceived | pm_files, pm_counters, kpi_values |
| `mr.file.received` | `mr-workers` | MRCollector.handleFileReceived | mr_files, mr_records |
| `device.inform.alarm` | `alarm-workers` | AlarmReceiver.handleAlarmEvent | alarms_active, alarms_history |
| `backup.task.created` | `backup-executors` | BackupExecutor.handleTaskCreated | backup_tasks |
| `report.generate.requested` | `report-generators` | ReportGenerator.handleGenerateRequest | report_records |

### 4.9 Worker 发布的二级事件

| Subject | 来源 | 消费者 |
|---------|------|--------|
| `pm.file.received` | TransferBridge (PM 文件) | Worker: PMCollector |
| `mr.file.received` | TransferBridge (MR 文件) | Worker: MRCollector |
| `datamodel.file.received` | TransferBridge (数据模型文件) | App: ProvisionEngine |
| `pm.file.parsed` | PMCollector | — |
| `mr.file.parsed` | MRCollector | — |
| `alarm.raised` / `.cleared` / `.acknowledged` | AlarmEngine | App: PushEngine (→ OSS) |
| `backup.task.done` | BackupExecutor | — |
| `report.generate.done` | ReportGenerator | — |

---

## 第五部分：完整事件流图

### 5.1 设备注册全链路

```
┌─────────┐   SOAP/XML    ┌─────────┐   NATS    ┌─────────┐   SQL    ┌──────────┐
│   CPE   │ ──Inform(0)──→│   ACS   │──────────→│   App   │────────→│ PostgreSQL│
│ Device  │               │ :7547   │           │ :8080   │         │ devices   │
└─────────┘               └─────────┘           └─────────┘         └──────────┘
                                                     │
                          device.inform.bootstrap    │  device.registered
                                                     ↓
                                              ┌─────────────┐
                                              │  Provision   │
                                              │  Engine      │
                                              │ (自动开站)    │
                                              └─────────────┘
```

### 5.2 PM 文件处理全链路

```
┌─────────┐  Upload   ┌─────────┐  ATC事件  ┌──────────────┐  pm.file.received  ┌──────────┐
│   CPE   │─────────→│   ACS   │────────→│TransferBridge│─────────────────→│PM Collector│
│ Device  │          │ :7547   │          │  (Worker)    │                  │  (Worker)  │
└─────────┘          └─────────┘          │              │                  │            │
                                          │ ①下载文件    │                  │ ①下载MinIO │
                                          │ ②存入MinIO   │                  │ ②解析XML   │
                                          └──────────────┘                  │ ③pm_counters│
                                                                            │ ④KPI计算    │
                                                                            └─────┬──────┘
                                                                                  │
                                                    pm.file.parsed                ↓
                                                                          ┌──────────────┐
                                                                          │ TimescaleDB  │
                                                                          │ pm_counters  │
                                                                          │ kpi_values   │
                                                                          └──────────────┘
```

### 5.3 告警处理全链路

```
┌─────────┐  Inform(6)  ┌─────────┐  device.inform.alarm  ┌──────────────┐  alarm.raised  ┌────────────┐
│   CPE   │───────────→│   ACS   │──────────────────────→│AlarmReceiver │──────────────→│PushEngine  │
│ Device  │            │ :7547   │                        │  (Worker)    │               │  (App)     │
└─────────┘            └─────────┘                        │              │               │            │
                                                          │ AlarmEngine: │               │ HTTP 推送  │
                                                          │ ①去重(Redis) │               │ → OSS 系统 │
                                                          │ ②入库(PgSQL) │               └────────────┘
                                                          │ ③发布事件    │
                                                          └──────────────┘
```

### 5.4 完整事件拓扑

```
                    omcgo-acs (事件生产者)
                    ──────────────────────
                    device.inform.bootstrap ─────────→ [App] InformHandler
                    device.inform.periodic ──────────→ [App] InformHandler
                    device.inform.value_change ──────→ [App] InformHandler
                    device.inform.alarm ─────────────→ [Worker] AlarmReceiver
                    device.inform.transfer_complete ─→ [App] SoftwareService
                    device.inform.autonomous_tc ─────→ [Worker] TransferBridge
                    command.*.response (11种) ───────→ [App] ProvisionEngine


                    omcgo-app (消费+生产)
                    ─────────────────────
                    ← device.inform.bootstrap/periodic/value_change
                    ← device.registered (自消费)
                    ← datamodel.file.received
                    ← command.get_parameters.response
                    ← command.get_names.response
                    ← device.inform.transfer_complete
                    ← oss.alarm.forward / oss.pm.export / oss.config.snapshot
                    → device.registered
                    → firmware.uploaded / upgrade.started / upgrade.completed


                    omcgo-worker (消费+生产)
                    ────────────────────────
                    ← device.inform.autonomous_transfer_complete
                    ← device.inform.alarm
                    ← pm.file.received (自消费，TransferBridge 发布)
                    ← mr.file.received (自消费，TransferBridge 发布)
                    ← backup.task.created
                    ← report.generate.requested
                    → pm.file.received / mr.file.received / datamodel.file.received
                    → pm.file.parsed / mr.file.parsed
                    → alarm.raised / alarm.cleared / alarm.acknowledged
                    → backup.task.done / report.generate.done
```

---

## 第六部分：数据库写入汇总

### App 服务写入

| 模块 | 表 | 类型 | 操作 |
|------|-----|------|------|
| InformHandler | `devices` | PostgreSQL | INSERT / UPDATE |
| InformHandler | `device_parameters` | PostgreSQL | BatchUpsert |
| ProvisionEngine | `provisioning_tasks` | PostgreSQL | INSERT / UPDATE |
| ProvisionEngine | `parameter_discovery_logs` | PostgreSQL | INSERT |
| SoftwareService | `upgrade_tasks` | PostgreSQL | UPDATE |
| PushEngine | `oss_events_outbox` | PostgreSQL | INSERT (outbox 模式) |

### Worker 服务写入

| 模块 | 表 | 存储 | 操作 |
|------|-----|------|------|
| PMCollector | `pm_files` | PostgreSQL | INSERT / UPDATE |
| PMCollector | `pm_counters` | TimescaleDB | BatchInsert |
| KPIEngine | `kpi_values` | TimescaleDB | INSERT |
| MRCollector | `mr_files` | PostgreSQL | INSERT / UPDATE |
| MRCollector | `mr_records` | TimescaleDB | BatchInsert |
| AlarmReceiver | `alarms_active` | PostgreSQL | INSERT / UPDATE |
| AlarmReceiver | `alarms_history` | TimescaleDB | INSERT |
| BackupExecutor | `backup_tasks` | PostgreSQL | UPDATE |
| ReportGenerator | `report_records` | PostgreSQL | UPDATE |

### Redis 写入

| 模块 | Key 模式 | 类型 | TTL |
|------|----------|------|-----|
| ACS handler | `acs:session:{sn}` | Hash | 5 min |
| InformHandler | heartbeat keys | String | 2×inform_interval |
| AlarmReceiver | `acs:alarm:{sn}:{code}` | String | 无 |
| BackupExecutor | `acs:cmdq:{sn}` | Sorted Set | 无 |

### MinIO 写入

| 模块 | Bucket | 路径模式 |
|------|--------|---------|
| TransferBridge | `pm-files` | `{date}/{device_sn}/{filename}` |
| TransferBridge | `mr-files` | `{date}/{device_sn}/{filename}` |
| TransferBridge | `logs` | `datamodel/{date}/{device_sn}/{filename}` |
| ReportGenerator | `reports` | `{def_id}/{period}/{record_id}.json` |

---

## 第七部分：错误处理与可靠性

### NATS 重试策略

| 重试次数 | 延迟 | 行为 |
|---------|------|------|
| 第 1 次 | 1s | `msg.NakWithDelay(1s)` |
| 第 2 次 | 2s | `msg.NakWithDelay(2s)` |
| 第 3 次 | 4s | `msg.NakWithDelay(4s)` |
| 第 4 次 | 8s | `msg.NakWithDelay(8s)` |
| 第 5 次 | — | `msg.Term()` 永久放弃 |

### 各 Handler 错误策略

| Handler | 临时错误 (DB/网络) | 业务错误 (数据不存在) |
|---------|-------------------|---------------------|
| InformHandler.handleBootstrap | return err → 重试 | 设备已存在 → 走 Update，return nil |
| InformHandler.handlePeriodic | return err → 重试 | 设备不存在 → 回退注册，return nil |
| AlarmReceiver | return err → 重试 | 去重命中 → 更新，return nil |
| PMCollector | return err → 重试 | 解析失败 → return err → 重试 |
| MRCollector | return err → 重试 | 类型检测失败 → return err → 重试 |
| TransferBridge | return err → 重试 | 设备不存在 → return nil (ACK) |
| BackupExecutor | return err → 重试 | 任务状态非 Pending → return nil |
| ReportGenerator | return nil (ACK) | 失败标记 RecordFailed → return nil |

### 关键可靠性机制

| 机制 | 说明 |
|------|------|
| Queue Group 负载均衡 | 同 queue 内多实例共享消费，一条消息只由一个实例处理 |
| Durable Consumer | 实例重启后从上次位置继续消费 |
| JetStream 持久化 | 消息存磁盘，72h 自动清理 |
| 查重 | RegisterFromInform 先 GetBySerialNumber 查重 |
| 告警去重 | Redis `acs:alarm:{sn}:{code}` 去重窗口 |
| Poison Pill 防护 | JSON 解析失败 → msg.Term() 永久放弃 |
| 会话隔离 | ACS 单设备错误不影响其他设备 |

---

## 第八部分：关键文件索引

### ACS (事件生产者)

| 文件 | 关键函数 |
|------|---------|
| `internal/acs/handler.go` | `publishInformEvents()` L1140-1194 |
| `internal/acs/handler.go` | `handleInform()` L270 |
| `internal/acs/handler.go` | `handleRPCResponse()` — RPC 响应事件 |
| `cmd/acs/bootstrap.go` | L39 EventBus 初始化 |

### App (事件消费+生产)

| 文件 | 关键函数 |
|------|---------|
| `cmd/app/router/router.go` | L87, L153, L196, L438 — 四组订阅注册 |
| `internal/device/inform_handler.go` | `Subscribe()` L47-66, `handleBootstrap()` L69-120, `handlePeriodic()` L122-181 |
| `internal/device/service.go` | `RegisterFromInform()` L201-317, `UpdateFromInform()` L320-399, `PublishDeviceRegistered()` L497-519 |
| `internal/device/pg_repository.go` | `Create()`, `Update()`, `GetBySerialNumber()` |
| `internal/provision/engine.go` | `Subscribe()` L85-131, `HandleBootstrap()` L135 |
| `internal/software/service.go` | `Subscribe()` L320-324, `HandleTransferComplete()` |
| `internal/northbound/push/engine.go` | `Subscribe()` L74-102, `handleEvent()` |

### Worker (后台事件处理)

| 文件 | 关键函数 |
|------|---------|
| `cmd/worker/main.go` | `registerSubscribers()` L69-147 |
| `cmd/worker/bootstrap.go` | L24-56 基础设施初始化 |
| `internal/transfer/bridge.go` | `Subscribe()` L55-67, `handleAutonomousTransferComplete()` L83-236 |
| `internal/pm/collector/collector.go` | `Subscribe()` L62-69, `handleFileReceived()` L71-170 |
| `internal/mr/collector/collector.go` | `Subscribe()` L63-74, `handleFileReceived()` L76-181 |
| `internal/alarm/receiver.go` | `Subscribe()` L39-50, `handleAlarmEvent()` L52-86 |
| `internal/alarm/engine.go` | `Process()` L54-157 |
| `internal/backup/executor.go` | `Subscribe()` L49-61, `handleTaskCreated()` L67-190 |
| `internal/report/generator.go` | `Subscribe()` L57-60, `handleGenerateRequest()` L62-164 |

### EventBus 基础设施

| 文件 | 关键内容 |
|------|---------|
| `internal/core/event/bus.go` | EventBus 接口定义 |
| `internal/core/event/types.go` | Event 结构体, NewEvent(), DecodePayload() |
| `internal/core/event/subjects.go` | 54 个 Subject 常量 |
| `internal/core/event/nats_bus.go` | NATS 实现, wrapHandler() 重试逻辑, maxDeliveries=5 |
| `internal/core/event/channel_bus.go` | Channel 实现, matchSubject() 通配符 |
| `internal/core/components/infra.go` | `CreateEventBus()` L156-159 |
| `internal/core/components/nats/nats.go` | `DefaultStreams()` 8 个 JetStream Stream |
| `internal/core/components/shutdown.go` | 优雅关闭优先级 |
