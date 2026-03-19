# DD-04: 事件总线

> 关联功能域：全部
> 关联 backend-design.md 章节：第六章（消息/事件架构）
> 实施阶段：Phase 1（基础建设）
> 依赖文档：DD-02, DD-03

---

## 1. 概述

### 1.1 模块定位

事件总线（`internal/common/event/`）是系统内模块间异步通信的核心机制，支持进程内 Go channel 和跨进程 NATS JetStream 两种实现，通过配置切换。

### 1.2 核心职责

- 定义统一的 EventBus 接口
- 提供 ChannelEventBus（单进程部署）
- 提供 NATSEventBus（多实例部署）
- 事件 Subject 命名规范管理
- 序列化/反序列化策略

---

## 2. 接口设计

### 2.1 EventBus 接口 — `internal/common/event/bus.go`

```go
type EventBus interface {
    // Publish 发布事件
    Publish(ctx context.Context, subject string, event Event) error
    // Subscribe 订阅事件（每个订阅者独立收到消息）
    Subscribe(subject string, handler EventHandler) (Subscription, error)
    // QueueSubscribe 队列订阅（同一 queue group 内负载均衡）
    QueueSubscribe(subject string, queue string, handler EventHandler) (Subscription, error)
    // Close 关闭事件总线
    Close() error
}

type Event struct {
    ID        string            `json:"id"`
    Subject   string            `json:"subject"`
    Payload   json.RawMessage   `json:"payload"`
    Metadata  map[string]string `json:"metadata,omitempty"`
    Timestamp time.Time         `json:"timestamp"`
}

type EventHandler func(ctx context.Context, event Event) error

type Subscription interface {
    Unsubscribe() error
}
```

### 2.2 事件 Subject 常量 — `internal/common/event/types.go`

```go
// 设备事件
const (
    SubjectDeviceBootstrap       = "device.inform.bootstrap"
    SubjectDevicePeriodic        = "device.inform.periodic"
    SubjectDeviceValueChange     = "device.inform.value_change"
    SubjectDeviceAlarm           = "device.inform.alarm"
    SubjectDeviceTransferComplete = "device.inform.transfer_complete"
    SubjectDeviceConnectionLost  = "device.connection.lost"
)

// 命令事件
const (
    SubjectCommandGetParams  = "command.get_parameters"
    SubjectCommandSetParams  = "command.set_parameters"
    SubjectCommandDownload   = "command.download"
    SubjectCommandUpload     = "command.upload"
    SubjectCommandReboot     = "command.reboot"
    SubjectCommandReset      = "command.factory_reset"
)

// 数据事件
const (
    SubjectPMFileReceived  = "pm.file.received"
    SubjectPMFileParsed    = "pm.file.parsed"
    SubjectMRFileReceived  = "mr.file.received"
    SubjectMRFileParsed    = "mr.file.parsed"
    SubjectAlarmRaised     = "alarm.raised"
    SubjectAlarmCleared    = "alarm.cleared"
    SubjectAlarmAcknowledged = "alarm.acknowledged"
)

// 北向事件
const (
    SubjectOSSAlarmForward  = "oss.alarm.forward"
    SubjectOSSPMExport      = "oss.pm.export"
    SubjectOSSConfigSnapshot = "oss.config.snapshot"
)
```

---

## 3. 详细设计

### 3.1 ChannelEventBus — `internal/common/event/channel.go`

进程内实现，使用 Go channel 实现 fan-out 分发。

```go
type ChannelEventBus struct {
    subscribers map[string][]*channelSubscription
    mu          sync.RWMutex
    bufferSize  int
    closed      atomic.Bool
}

type channelSubscription struct {
    ch      chan Event
    handler EventHandler
    queue   string // 非空表示 queue subscribe
    cancel  context.CancelFunc
}

func NewChannelEventBus(bufferSize int) *ChannelEventBus
```

**关键行为**：
- `Publish`：遍历匹配 subject 的订阅者，非阻塞发送到 channel
- `Subscribe`：每个订阅者独立 goroutine 消费 channel
- `QueueSubscribe`：同一 queue group 的订阅者共享一个 channel，轮询分发
- Subject 支持通配符匹配（`device.>` 匹配所有 `device.*.*`）
- 缓冲区满时丢弃旧消息并记录日志

### 3.2 NATSEventBus — `internal/common/event/nats.go`

跨进程实现，使用 NATS JetStream 持久化消息。

```go
type NATSEventBus struct {
    conn    *nats.Conn
    js      nats.JetStreamContext
    subs    []*nats.Subscription
    mu      sync.Mutex
}

func NewNATSEventBus(client *NATSClient) (*NATSEventBus, error)
```

**关键行为**：
- `Publish`：序列化 Event 为 JSON，发布到 JetStream
- `Subscribe`：创建 Push Subscriber，使用 DeliverAll 策略
- `QueueSubscribe`：创建 Queue Subscriber（Durable Consumer）
- 自动 Ack 模式：handler 返回 nil 时 Ack，返回 error 时 Nak（触发重试）
- Dead Letter Queue：超过最大重试次数后转发到 `*.dlq` subject

### 3.3 配置驱动切换

```go
// NewEventBus 根据配置创建 EventBus 实例
func NewEventBus(cfg EventBusConfig) (EventBus, error) {
    switch cfg.Type {
    case "channel":
        return NewChannelEventBus(cfg.BufferSize), nil
    case "nats":
        return NewNATSEventBus(cfg.NATSClient)
    default:
        return nil, fmt.Errorf("unsupported event bus type: %s", cfg.Type)
    }
}
```

### 3.4 Subject 通配符规则

遵循 NATS Subject 语法：
- `.` 分层（如 `device.inform.bootstrap`）
- `*` 匹配单层（如 `device.inform.*` 匹配 `device.inform.bootstrap`）
- `>` 匹配多层（如 `device.>` 匹配所有 `device.*.*`）

---

## 4. 实施子阶段

### 阶段 4a：接口 + Event 类型 + Subject 常量

**交付物**：`bus.go`、`types.go`
**验证**：编译通过

### 阶段 4b：ChannelEventBus 实现

**交付物**：`channel.go` + 单元测试
**验证**：发布/订阅/QueueSubscribe 功能测试通过

### 阶段 4c：NATSEventBus 实现

**交付物**：`nats.go` + 集成测试
**验证**：连接 NATS，发布消息，消费者接收

### 阶段 4d：监控集成

**交付物**：事件发布/消费速率 Prometheus 指标
**验证**：metrics 端点可查看事件指标

---

## 5. 文件清单

```
internal/common/event/bus.go
internal/common/event/types.go
internal/common/event/channel.go
internal/common/event/nats.go
internal/common/event/config.go
```

---

## 6. 测试策略

- ChannelEventBus：单元测试覆盖 Publish/Subscribe/QueueSubscribe、通配符匹配、并发安全
- NATSEventBus：集成测试（需要 NATS 实例）
- 序列化/反序列化测试

---

## 7. 参考

- backend-design.md 第六章：消息/事件架构
- CLAUDE.md 第 5.4 节：事件驱动规范
