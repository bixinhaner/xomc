# ACS 服务完整实现流程分析

> 本文档分析 ACS（TR-069 自动配置服务器）引擎的完整实现流程，包含请求处理、EventBus 发布/订阅、命令队列反向通道等核心链路，以及每个交互流程的详细 SOAP/XML 报文示例。

---

## 一、总体架构

ACS 引擎是 OMCGo 系统的**南向接口**（F01），负责 TR-069/CWMP 协议处理。它作为独立进程部署（`omcgo-acs`），通过 **EventBus** 与主应用（`omcgo-app`）和工作进程（`omcgo-worker`）协作。

```
┌──────────┐  SOAP/XML   ┌──────────┐  EventBus    ┌──────────┐
│   CPE    │ ←─────────→ │   ACS    │ ──────────→  │   App    │
│ (基站)   │  HTTP:7547   │  Engine  │              │ (F02-F10)│
└──────────┘              └──────────┘              └──────────┘
                               │                         │
                          Redis CmdQueue            ┌──────────┐
                               ↑                    │  Worker  │
                               └────────────────────│ (PM/MR)  │
                                                    └──────────┘
```

### 三个部署单元的职责

| 单元 | 端口 | 职责 |
|------|------|------|
| `omcgo-acs` | :7547 | TR-069 协议处理，高并发会话管理 |
| `omcgo-app` | :8080 | 设备管理、配置、自动开站等业务逻辑（F02-F10） |
| `omcgo-worker` | — | PM/MR 文件解析、KPI 计算、告警处理等重 I/O 任务 |

---

## 二、ACS 请求处理主流程

### 核心文件

| 文件 | 职责 |
|------|------|
| `internal/acs/server.go` | HTTP 服务器生命周期、依赖注入 |
| `internal/acs/handler.go` | 请求路由与 SOAP 方法分发 |
| `internal/acs/session.go` | 会话状态机（Redis 存储） |
| `internal/acs/ratelimit.go` | 设备级限流（LRU + Token Bucket） |
| `internal/acs/admission.go` | 全局并发准入控制（Atomic CAS） |
| `internal/acs/soap/` | SOAP 编解码 |
| `internal/acs/rpc/` | RPC 方法构建（Get/Set/Download/Upload/Reboot...） |
| `internal/acs/cmdqueue/queue.go` | Redis 命令队列 |
| `internal/acs/connreq/client.go` | Connection Request 客户端 |

### 2.1 会话状态机

```
IDLE
  ↓ [CPE 发送 Inform]
StateInformReceived → 发送 InformResponse
  ↓ [CPE 发送 Empty POST]
StateProcessing → 检查命令队列
  ↓
  ├─ 有命令 → StateRPCPending → 发送 SOAP RPC
  │              ↓ [CPE 返回 RPC 响应]
  │           StateRPCResponse → 检查是否还有命令
  │              ├─ 有 → 循环回 StateRPCPending
  │              └─ 无 → 继续 ↓
  │
  └─ 无命令 ↓
StateComplete → 释放准入 → HTTP 204（会话结束）
```

### 2.2 Inform 处理流程

```
CPE 发起 HTTP POST (SOAP Inform)
    ↓
handler.ServeHTTP() — 路由分发
    ↓
handleInform()
  ├─ 1. 解析 SOAP XML（流式 xml.Decoder，不加载整个 XML 到内存）
  ├─ 2. 设备级限流检查
  │     DeviceRateLimiter.Allow(deviceSN)
  │     └─ 基于 hashicorp/golang-lru，O(1) 查找与淘汰
  │     └─ 拒绝 → 503 + metrics.RateLimitRejected.Inc()
  ├─ 3. 全局准入控制
  │     AdmissionController.TryAcquire()
  │     └─ Atomic CAS 控制最大并发会话数
  │     └─ 拒绝 → 503 Service Unavailable
  ├─ 4. 创建/更新会话
  │     SessionStore.Create(deviceSN, session)
  │     └─ Redis Hash: acs:session:{deviceSN}, TTL 5min
  ├─ 5. 绑定连接
  │     connectionMap.Store(RemoteAddr, deviceSN)
  │     └─ sync.Map，用于后续 Empty POST 查找设备
  ├─ 6. 发布事件
  │     publishInformEvents()
  │     └─ 根据 EventCode 发布到不同 Subject（详见第三节）
  └─ 7. 返回 InformResponse（SOAP XML）
```

### 2.3 Empty POST 处理（命令下发入口）

```
CPE 发送空 POST（表示"我准备好接收指令"）
    ↓
handleEmpty()
  ├─ 通过 connectionMap 查找 deviceSN
  ├─ commandQueue.Pop(ctx, deviceSN)
  │   ├─ 有命令：
  │   │   ├─ 更新会话状态 → StateRPCPending
  │   │   ├─ rpcDispatcher.BuildRequest(cmd) → 生成 SOAP XML
  │   │   └─ 发送 SOAP 响应给 CPE
  │   └─ 无命令：
  │       ├─ completeSession() → 释放准入槽位
  │       └─ HTTP 204 No Content（通知 CPE 会话结束）
```

### 2.4 RPC 响应处理

```
CPE 返回 RPC 响应（如 SetParameterValuesResponse）
    ↓
handleRPCResponse()
  ├─ 解析 SOAP 响应
  ├─ 发布 command.*.response 事件（详见第三节）
  ├─ 检查命令队列
  │   ├─ 还有命令 → Pop → BuildRequest → 发送下一个 RPC
  │   └─ 无更多命令 → completeSession() → HTTP 204
```

---

## 三、EventBus 发布端

### 3.1 Inform 事件发布

`handler.go` 的 `publishInformEvents()` 根据 Inform 报文中的 EventCode 映射到不同 Subject：

| EventCode | 事件 Subject | 含义 |
|-----------|-------------|------|
| `0 BOOTSTRAP` | `device.inform.bootstrap` | 新设备首次接入（出厂/恢复出厂） |
| `1 BOOT` | `device.inform.boot` | 设备重启完成 |
| `2 PERIODIC` | `device.inform.periodic` | 周期性心跳上报 |
| `4 VALUE CHANGE` | `device.inform.value_change` | 设备参数发生变化 |
| `6 CONNECTION REQUEST` | `device.inform.connection_request` | 响应 ACS 的 Connection Request |
| `7 TRANSFER COMPLETE` | `device.transfer.complete` | 文件传输（下载/上传）完成 |
| `8 DIAGNOSTICS COMPLETE` | `device.inform.diagnostics_complete` | 诊断任务完成 |
| `10 AUTONOMOUS TRANSFER` | `device.autonomous_transfer_complete` | 设备主动上传文件（PM/MR） |

### 3.2 RPC 响应事件发布

`handleRPCResponse()` 中的 `publishRPCResponseEvent()` 发布命令响应事件：

| RPC 方法 | 事件 Subject |
|----------|-------------|
| GetParameterValues | `command.get_parameters.response` |
| SetParameterValues | `command.set_parameters.response` |
| GetParameterNames | `command.get_parameter_names.response` |
| AddObject | `command.add_object.response` |
| DeleteObject | `command.delete_object.response` |
| Download | `command.download.response` |
| Upload | `command.upload.response` |
| Reboot | `command.reboot.response` |
| FactoryReset | `command.factory_reset.response` |
| GetRPCMethods | `command.get_rpc_methods.response` |
| ScheduleInform | `command.schedule_inform.response` |

---

## 四、EventBus 实现

### 4.1 接口定义

```go
// internal/core/event/bus.go
type EventBus interface {
    Publish(ctx context.Context, subject string, event Event) error
    Subscribe(subject string, handler EventHandler) (Subscription, error)
    QueueSubscribe(subject string, queue string, handler EventHandler) (Subscription, error)
    Close() error
}
```

### 4.2 事件信封

```go
// internal/core/event/types.go
type Event struct {
    ID        string            `json:"id"`        // UUID
    Subject   string            `json:"subject"`   // 事件主题
    Payload   json.RawMessage   `json:"payload"`   // 业务负载（JSON）
    Metadata  map[string]string `json:"metadata"`  // 元数据（carrier, tech 等）
    Timestamp time.Time         `json:"timestamp"` // 发布时间
}

type EventHandler func(ctx context.Context, event Event) error
```

### 4.3 双实现对比

| 维度 | ChannelEventBus | NATSEventBus |
|------|----------------|--------------|
| 适用场景 | 单进程 / 开发环境 | 多实例生产部署 |
| 底层 | Go channel（buffer 256） | NATS JetStream |
| 通配符 | 支持 `*`（单级）和 `>`（多级） | 原生支持 |
| Queue Group | 负载均衡（首个可用订阅者） | Durable 持久化队列组 |
| 非 Queue 订阅 | 广播（所有订阅者都收到） | 广播 |
| 满载策略 | 丢弃 + 日志告警 | JetStream 保证投递 |
| 失败重试 | 无 | 指数退避（1s→2s→4s→8s），5 次后 Term |
| 解析错误 | — | 立即 Term（避免无限重试） |

### 4.4 实现选择机制

#### 当前行为：无条件使用 NATSEventBus

三个部署单元（app、acs、worker）**统一使用 NATSEventBus**，没有运行时配置开关。决定逻辑位于 bootstrap 框架的 `createEventBus()` 方法中：

```go
// internal/core/bootstrap/bootstrap.go
func (a *App) createEventBus() {
    a.EventBus = event.NewNATSEventBus(a.NATS.Conn, a.NATS.JS, a.Logger)
    a.GS.Register("eventbus", 2, func(ctx context.Context) error {
        return a.EventBus.Close()
    })
}
```

该方法**无条件**创建 `NATSEventBus`，没有 `if/else` 分支，不读取任何配置字段。

#### 初始化链路

```
配置文件（YAML）
    ↓ viper 加载
NATSConfig{URL, MaxReconnect, ReconnectWait}
    ↓
bootstrap.InitForApp() / InitForACS() / InitForWorker()
    ↓
bootstrap.connectNATS(cfg.NATS)
    ↓ nats.Connect(cfg.URL, opts...)
    ↓ conn.JetStream()
App.NATS = &NATSClient{Conn, JS}
    ↓
App.createEventBus()                    ← 决策点（无条件 NATS）
    ↓
App.EventBus = NewNATSEventBus(...)     ← 传递给所有业务模块
```

三个部署单元的初始化函数都调用同一个 `createEventBus()`：

| 入口 | 初始化函数 | EventBus 创建 |
|------|-----------|--------------|
| `cmd/app/main.go` | `bootstrap.InitForApp()` | `createEventBus()` → NATSEventBus |
| `cmd/acs/main.go` | `bootstrap.InitForACS()` | `createEventBus()` → NATSEventBus |
| `cmd/worker/main.go` | `bootstrap.InitForWorker()` | `createEventBus()` → NATSEventBus |

#### NATS 配置

配置文件仅包含 NATS 连接参数，没有 EventBus 类型选择字段：

```yaml
# cmd/app/etc/config.dev.yaml（三个单元配置结构相同）
nats:
  url: "nats://localhost:4222"
  max_reconnect: -1         # -1 = 无限重连
  reconnect_wait: 2s
```

对应配置结构体：

```go
// internal/core/appconfig/config.go
type NATSConfig struct {
    URL           string        `mapstructure:"url"`
    MaxReconnect  int           `mapstructure:"max_reconnect"`
    ReconnectWait time.Duration `mapstructure:"reconnect_wait"`
}
```

#### ChannelEventBus 的使用场景

`ChannelEventBus` **仅用于单元测试和集成测试**，不参与生产运行：

```go
// test/integration/inform_register_test.go
eventBus := event.NewChannelEventBus(256, zap.NewNop())

// internal/backup/executor_test.go
eventBus := event.NewChannelEventBus(256, zap.NewNop())
```

测试中使用 `ChannelEventBus` 的好处：
- 无需启动 NATS 服务即可运行测试
- 事件同步传递，测试结果确定性高
- 进程内 channel 通信，无网络开销

#### 设计意图

| 实现 | 定位 | 使用场景 |
|------|------|---------|
| `NATSEventBus` | **生产实现** | 所有部署环境（dev/test/prod） |
| `ChannelEventBus` | **测试替身** | 单元测试、集成测试中替代 NATS |

这种设计遵循**接口优先**原则：`EventBus` 接口使得生产代码和测试代码可以使用不同实现，而业务逻辑无需感知底层传输机制。当前架构假设 NATS 始终可用（即使开发环境也通过 Docker Compose 启动 NATS），因此不需要运行时切换能力。

#### 关键文件

| 文件 | 说明 |
|------|------|
| `internal/core/bootstrap/bootstrap.go` | `createEventBus()` — 决策点 |
| `internal/core/appconfig/config.go` | `NATSConfig` 配置结构体 |
| `internal/core/components/nats/nats.go` | NATS 客户端初始化 |
| `internal/core/event/nats_bus.go` | NATSEventBus 实现（生产） |
| `internal/core/event/channel_bus.go` | ChannelEventBus 实现（测试） |

### 4.5 事件 Subject 完整列表

```
# 设备事件（ACS 发布）
device.inform.bootstrap
device.inform.boot
device.inform.periodic
device.inform.value_change
device.inform.connection_request
device.inform.diagnostics_complete
device.inform.alarm
device.transfer.complete
device.autonomous_transfer_complete

# 设备生命周期（App 发布）
device.registered                        # DeviceService 注册新设备入库后发布

# 命令响应（ACS 发布）
command.get_parameters.response
command.set_parameters.response
command.get_parameter_names.response
command.add_object.response
command.delete_object.response
command.download.response
command.upload.response
command.reboot.response
command.factory_reset.response
command.get_rpc_methods.response
command.schedule_inform.response

# PM/MR（Worker 内部流转）
pm.file.received / pm.file.parsed
mr.file.received / mr.file.parsed

# 告警
alarm.raised / alarm.cleared / alarm.acknowledged

# 自动开站
provision.started / provision.completed / provision.failed

# 固件
firmware.upgrade.started / firmware.upgrade.completed

# 备份
backup.task.created / backup.completed

# 报表
report.generate.requested / report.completed

# 北向/OSS
oss.alarm.forward / oss.pm.export / oss.config.snapshot

# 网元直连
nedirect.command.request / nedirect.command.response
```

---

## 五、EventBus 订阅端

### 5.1 App 进程订阅者

注册位置：`cmd/app/router/router.go`

| 消费者 | 订阅 Subject | Queue Group | 职责 |
|--------|-------------|-------------|------|
| `InformHandler` | `device.inform.bootstrap` | `device-manager` | `RegisterFromInform()` — 注册新设备到数据库，成功后发布 `device.registered` |
| `InformHandler` | `device.inform.periodic` | `device-manager` | `UpdateFromInform()` — 更新设备在线状态、参数快照 |
| `InformHandler` | `device.inform.value_change` | `device-manager` | 参数变更记录与通知 |
| `ProvisioningEngine` | `device.registered` | `provisioning` | 自动开站流程：数据模型解析 → 模板匹配 → 命令入队（确保设备已入库） |
| `SoftwareService` | `device.transfer.complete` | — | 固件升级状态机推进（下载完成 → 重启） |
| `PushEngine` | `oss.alarm.forward` | — | 北向告警推送到 OSS |
| `PushEngine` | `oss.pm.export` | — | 北向 PM 数据推送到 OSS |
| `PushEngine` | `oss.config.snapshot` | — | 北向配置快照推送到 OSS |

### 5.2 Worker 进程订阅者

注册位置：`cmd/worker/main.go`

| 消费者 | 订阅 Subject | Queue Group | 职责 |
|--------|-------------|-------------|------|
| `TransferBridge` | `device.autonomous_transfer_complete` | `transfer-bridge` | 下载文件 → 存 MinIO → 发布 `pm/mr.file.received` |
| `PMCollector` | `pm.file.received` | `pm-workers` | 解析 PM XML → 存计数器 → 计算 KPI → 发布 `pm.file.parsed` |
| `MRCollector` | `mr.file.received` | `mr-workers` | 检测类型(MRO/MRS/MRE) → 解析 → 存 DB → 发布 `mr.file.parsed` |
| `AlarmReceiver` | `device.inform.alarm` | `alarm-workers` | 告警去重 → 关联分析 → 存储 → 生命周期管理 |
| `BackupExecutor` | `backup.task.created` | `backup-executors` | 向命令队列推送 Upload 命令 → 唤醒设备 |
| `ReportGenerator` | `report.generate.requested` | `report-generators` | 数据采集 → 生成报表 → 上传 MinIO |

### 5.3 事件级联示意

```
设备主动上传 PM 文件:

CPE → Inform(AUTONOMOUS TRANSFER COMPLETE)
  ↓ ACS 发布
device.autonomous_transfer_complete
  ↓ TransferBridge 订阅
下载文件 → 存 MinIO → 发布 pm.file.received
  ↓ PMCollector 订阅
解析 XML → 存计数器 → 计算 KPI → 发布 pm.file.parsed
  ↓ (可��) 北向推送
oss.pm.export → PushEngine → OSS
```

---

## 六、命令队列反向通道（App → ACS → CPE）

### 6.1 存储结构

**Redis Sorted Set**：

```
Key:   acs:cmdq:{device_serial}
Score: priority * 1e12 + timestamp_ns / 1e9
Value: JSON(Command)
```

```go
// internal/acs/cmdqueue/queue.go
type Command struct {
    ID         string          // UUID
    Method     string          // TR069 RPC 方法名
    Params     json.RawMessage // 方法参数（JSON）
    Priority   int             // 优先级（越小越优先）
    CreatedAt  time.Time       // 创建时间（同优先级按 FIFO）
    ExpiresAt  *time.Time      // 可选过期时间
    CommandKey string          // 去重键
}
```

### 6.2 接口定义

```go
type CommandQueue interface {
    Push(ctx context.Context, deviceSN string, cmd *Command) error
    Pop(ctx context.Context, deviceSN string) (*Command, error)
    Peek(ctx context.Context, deviceSN string) (*Command, error)
    Len(ctx context.Context, deviceSN string) (int64, error)
    Clear(ctx context.Context, deviceSN string) error
}
```

### 6.3 命令推送方

| 模块 | 文件 | 推送场景 | 命令类型 |
|------|------|---------|---------|
| 自动开站 | `provision/orchestrator.go` | 新设备 bootstrap 事件触发 | Get/SetParameterValues, Reboot |
| 配置同步 | `config/sync_handler.go` | REST API `POST /api/v1/config/sync/{push\|pull}/:deviceId` | Get/SetParameterValues |
| 固件升级 | `software/service.go` | `StartUpgrade()` / `BatchUpgrade()` | Download |
| 配置备份 | `backup/executor.go` | 监听 `backup.task.created` 事件 | Upload |
| 文件分发 | `filemanager/handler.go` | REST API `POST /api/v1/files/:id/distribute` | Download |
| 互操作测试 | `interop/runner.go` | 测试用例执行 | 各种 RPC |

### 6.4 Connection Request 完整解析

#### 6.4.1 为什么需要 Connection Request

TR-069 协议中 **CPE 是 HTTP Client，ACS 是 HTTP Server**——只有 CPE 能主动发起连接。CPE 平时按自己的节奏（心跳、事件变化）连接 ACS。当 ACS 需要主动对设备执行操作（读参数、下发配置、固件升级）时，**唯一的办法**是通过 Connection Request 唤醒 CPE，让它发起一个新的 CWMP 会话。

#### 6.4.2 机制本质：两步握手

```
步骤 1 (ACS → CPE): ACS 向 CPE 的 ConnectionRequestURL 发送 HTTP GET
                     含义："我有事找你，请立即连接我"

步骤 2 (CPE → ACS): CPE 向 ACS 发送 Inform（事件码 = 6 CONNECTION REQUEST）
                     含义："好的，我来了，你说"
```

#### 6.4.3 步骤 1 实现：connreq.Client（ACS → CPE）

```go
// internal/acs/connreq/client.go
type Client struct {
    httpClient *http.Client          // 支持 TLS 1.2+
    redis      redis.UniversalClient // 去重
    logger     *zap.Logger
    digest     *DigestCredentials    // 可选 Digest 认证
}

func (c *Client) Send(ctx context.Context, deviceSN, url string) error
```

**发送流程**：

1. **Redis SetNX 去重** — `acs:connreq:pending:{deviceSN}`，TTL 30 秒，30 秒内不重复发送
2. **HTTP GET** — 向 CPE 的 ConnectionRequestURL 发送请求
3. **Digest 认证** — 如果 CPE 返回 401，解析 `WWW-Authenticate` 头，构造 Digest 响应重试
4. **指数退避重试** — 失败后 1s → 2s → 4s，最多 3 次

#### 6.4.4 触发 Connection Request 的业务场景

项目中有两个模块直接调用 `connReq.Send()`：

| 调用方 | 代码位置 | 业务场景 | 模式 |
|--------|---------|---------|------|
| `SoftwareService.StartUpgrade()` | `software/service.go:148` | 固件升级 — 推送 Download 命令后唤醒设备来取命令 | cmdQueue.Push → connReq.Send |
| `SoftwareService.BatchUpgrade()` | `software/service.go:222` | 批量固件升级 — 对每台设备执行同样流程 | cmdQueue.Push → connReq.Send |
| `BackupExecutor.handleTaskCreated()` | `backup/executor.go:132` | 配置备份 — 推送 Upload 命令后唤醒设备执行上传 | cmdQueue.Push → connReq.Send |

统一模式：**先入队命令（Redis Sorted Set），再唤醒设备（HTTP GET）**。

```go
// 固件升级（software/service.go）
s.cmdQueue.Push(ctx, dev.SerialNumber, cmd)                           // 1. 命令入队
s.connReq.Send(ctx, dev.SerialNumber, dev.ConnectionRequestURL)       // 2. 唤醒

// 配置备份（backup/executor.go）
e.cmdQueue.Push(ctx, dev.SerialNumber, cmd)                           // 1. 命令入队
e.connReq.Send(ctx, dev.SerialNumber, dev.ConnectionRequestURL)       // 2. 唤醒
```

#### 6.4.5 步骤 2 实现：CPE 响应（CPE → ACS）

CPE 收到 HTTP GET 后，向 ACS 发起一个新的 Inform 请求，事件码携带 `6 CONNECTION REQUEST`：

```go
// handler.go — publishInformEvents()
case tr069.IsConnectionRequest(inform.Event):
    subject = event.SubjectDeviceConnectionRequest  // "device.inform.connection_request"
```

之后进入 ACS 的正常会话流程：handleInform → InformResponse → Empty POST → handleEmpty → cmdQueue.Pop → 下发 RPC。

#### 6.4.6 端到端时序图（以固件升级为例）

```
运维人员              App                  Redis               ACS              CPE
  │                   │                    │                   │                │
  │ POST /upgrade     │                    │                   │                │
  ├──────────────────→│                    │                   │                │
  │                   │ Push(Download)     │                   │                │
  │                   ├───────────────────→│ acs:cmdq:{sn}     │                │
  │                   │                    │                   │                │
  │                   │ connReq.Send()     │                   │                │
  │                   ├───────────────────→│ SetNX 去重        │                │
  │                   │                    │                   │                │
  │                   │                    │               HTTP GET             │
  │                   │                    │                   ├───────────────→│
  │                   │                    │                   │    200 OK      │
  │                   │                    │                   │←───────────────┤
  │                   │                    │                   │                │
  │                   │                    │                   │  Inform        │
  │                   │                    │                   │  (6 CONN REQ)  │
  │                   │                    │                   │←───────────────┤
  │                   │                    │                   │ InformResponse │
  │                   │                    │                   ├───────────────→│
  │                   │                    │                   │                │
  │                   │                    │                   │  Empty POST    │
  │                   │                    │                   │←───────────────┤
  │                   │                    │ Pop(Download)     │                │
  │                   │                    │←──────────────────┤                │
  │                   │                    │                   │  Download RPC  │
  │                   │                    │                   ├───────────────→│
  │                   │                    │                   │                │
  │                   │                    │                   │ DownloadResp   │
  │                   │                    │                   │←───────────────┤
  │                   │                    │                   │  HTTP 204      │
  │                   │                    │                   ├───────────────→│
```

#### 6.4.7 与其他 Inform 事件码的对比

| 事件码 | 触发方 | 场景 |
|--------|--------|------|
| `0 BOOTSTRAP` | CPE 自发 | 首次上电或恢复出厂设置 |
| `1 BOOT` | CPE 自发 | 设备重启完成 |
| `2 PERIODIC` | CPE 自发 | 定时心跳（默认 300s-3600s 周期） |
| `4 VALUE CHANGE` | CPE 自发 | 参数值被本地或远端修改 |
| **`6 CONNECTION REQUEST`** | **ACS 触发，CPE 响应** | **ACS 发送 HTTP GET 唤醒后，CPE 回连** |
| `7 TRANSFER COMPLETE` | CPE 自发 | 文件下载/上传完成 |
| `8 DIAGNOSTICS COMPLETE` | CPE 自发 | 诊断任务完成 |
| `10 AUTONOMOUS TRANSFER` | CPE 自发 | 设备主动上传文件（PM/MR） |

**`6 CONNECTION REQUEST` 是 TR-069 协议中 ACS 反向控制 CPE 的唯一通道。**

### 6.5 完整下发流程示例

**场景：用户通过 REST API 推送配置到设备**

```
1. 用户调用 REST API
   POST /api/v1/config/sync/push/{deviceId}
   Body: { "parameters": [...] }
         ↓
2. config/SyncHandler.PushConfig()
   ├─ 构建 Command{Method: "SetParameterValues", Params: {...}}
   └─ cmdQueue.Push(ctx, deviceSN, cmd)
         ↓
3. 命令存入 Redis
   ZADD acs:cmdq:{deviceSN} <score> <command_json>
         ↓
4. 发送 Connection Request 唤醒设备
   connReq.Send(deviceSN, connectionRequestURL)
   └─ 去重检查: SET acs:connreq:pending:{deviceSN} NX EX 30
         ↓
5. 设备被唤醒，发起新的 TR-069 会话
   CPE → HTTP POST (SOAP Inform, EventCode=6 CONNECTION REQUEST)
         ↓
6. ACS handleInform()
   ├─ 限流 + 准入 + 创建会话
   ├─ 发布 device.inform.connection_request 事件
   └─ 返回 InformResponse
         ↓
7. CPE 发送 Empty POST（等待指令）
         ↓
8. ACS handleEmpty()
   ├─ cmdQueue.Pop(deviceSN)
   │   → 取出 SetParameterValues 命令
   ├─ rpcDispatcher.BuildRequest(cmd)
   │   → 生成 SOAP SetParameterValues XML
   └─ 发送 SOAP 响应给 CPE
         ↓
9. CPE 执行参数设置，返回 SetParameterValuesResponse
         ↓
10. ACS handleRPCResponse()
    ├─ 发布 command.set_parameters.response 事件
    ├─ cmdQueue.Pop(deviceSN) → 无更多命令
    └─ completeSession() → HTTP 204 结束会话
```

---

## 七、流量控制与保护机制

### 7.1 设备级限流

```go
// internal/acs/ratelimit.go
type DeviceRateLimiter struct {
    cache  *lru.Cache[string, *DeviceEntry]  // hashicorp/golang-lru, O(1) 淘汰
    limit  rate.Limit                         // 每秒令牌数
    burst  int                                // 令牌桶容量
}
```

- 每个设备独立的 Token Bucket（`golang.org/x/time/rate`）
- LRU 缓存限制最大设备数（默认 10 万），自动淘汰最久未访问的设备
- 后台定期清理超过 TTL 的设备条目

### 7.2 全局准入控制

```go
// internal/acs/admission.go — Atomic CAS
func (ac *AdmissionController) TryAcquire() bool
func (ac *AdmissionController) Release()
```

- 限制 ACS 实例的最大并发会话数
- 超出时返回 503，CPE 会自动重试

### 7.3 会话超时回收

```go
// internal/acs/server.go — Session Reaper
// 每 30 秒扫描，清理超过 5 分钟无活动的会话
```

---

## 八、Prometheus 监控指标

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `acs_inform_total` | Counter | Inform 请求总数（按 event_code 标签） |
| `acs_session_duration_seconds` | Histogram | 会话持续时间分布 |
| `acs_active_sessions` | Gauge | 当前活跃会话数 |
| `acs_rate_limit_rejected_total` | Counter | 被限流拒绝的请求数 |
| `acs_rate_limit_device_count` | Gauge | 限流器中的设备数 |
| `acs_rpc_requests_total` | Counter | RPC 请求总数（按 method 标签） |
| `acs_command_queue_length` | Gauge | 命令队列长度 |

---

## 九、Redis Key 一览

```
# 会话管理
acs:session:{device_serial}          — 会话状态 Hash（TTL 5 分钟）

# 命令队列
acs:cmdq:{device_serial}             — 命令队列 Sorted Set

# 心跳与限流
acs:heartbeat:{device_serial}        — 心跳时间戳（TTL = 2×inform_interval）
ratelimit:inform:{device_serial}     — 限流计数器

# Connection Request
acs:connreq:pending:{device_serial}  — 去重标记（TTL 30 秒）

# 数据模型缓存
datamodel:product:{carrier}:{tech}:{oui}:{product_class}
datamodel:oui:{carrier}:{tech}:{oui}
datamodel:default:{carrier}:{tech}
datamodel:resolve:{carrier}:{tech}:{oui}:{product_class}  — 解析结果（TTL 1 小时）
datamodel:cache_version              — 缓存版本号（跨实例协调）

# 告警
alarm:active:{device_serial}         — 活跃告警 Hash
```

---

## 十、事件链路总览

```
                         ┌─── App 进程 ────────────────────────────────┐
                         │                                              │
  device.inform.bootstrap ──→ InformHandler.RegisterFromInform()       │
                         │       ↓ 注册入库成功后发布                    │
                         │    device.registered                         │
                         │       ↓                                      │
                         │    ProvisioningEngine.HandleBootstrap()      │
                         │       ↓ cmdQueue.Push() + connReq.Send()    │
  device.inform.periodic ───→ InformHandler.UpdateFromInform()         │
  device.inform.alarm ──────→ (转发到 Worker alarm-workers 队列)       │
  device.transfer.complete ─→ SoftwareService 固件状态机               │
  oss.*.forward/export ─────→ PushEngine → 外部 OSS 系统              │
                         │                                              │
                         │  REST API → cmdQueue.Push():                 │
                         │    config/sync_handler (配置同步)             │
                         │    software/service (固件升级)                │
                         │    filemanager/handler (文件分发)             │
                         └──────────────────────────────────────────────┘

                         ┌─── Worker 进程 ─────────────────────────────┐
                         │                                              │
  device.autonomous_transfer_complete                                   │
       ↓ TransferBridge                                                │
  下载文件 → 存 MinIO                                                  │
       ├→ pm.file.received → PMCollector → XML 解析 → KPI 计算         │
       └→ mr.file.received → MRCollector → 类型检测 → 解析存储         │
                                                                        │
  device.inform.alarm ──→ AlarmReceiver → 去重/关联/存储               │
  backup.task.created ──→ BackupExecutor → cmdQueue.Push(Upload)       │
  report.generate.requested → ReportGenerator → 采集/生成 → MinIO     │
                         └──────────────────────────────────────────────┘

                         ┌─── ACS 引擎 ───────────────────────────────┐
                         │                                              │
  CPE Inform ──→ handleInform() ──→ 发布 device.inform.* 事件         │
  CPE Empty  ──→ handleEmpty()  ──→ cmdQueue.Pop() → 发送 SOAP RPC    │
  CPE Response → handleRPCResponse() → 发布 command.*.response        │
                         │                                              │
  Redis acs:cmdq:* ←──── 各模块 Push 命令                              │
  ConnReq HTTP GET ────→ 唤醒离线 CPE                                  │
                         └──────────────────────────────────────────────┘
```

---

## 十一、关键设计总结

| 设计点 | 实现方式 | 设计理由 |
|--------|---------|---------|
| **进程分离** | ACS / App / Worker 独立部署 | ACS 高并发会话与管理面/数据面分离，独立扩展 |
| **事件驱动** | EventBus 双实现（Channel / NATS） | 进程内零开销，多实例 JetStream 持久化 |
| **Queue Group** | 同事件多 Worker 只有一个处理 | ���持 Worker 水平扩展，避免重复处理 |
| **命令队列** | Redis Sorted Set（优先级 + FIFO） | 跨进程共享，支持优先级，命令过期自动清理 |
| **Connection Request** | HTTP GET + Redis 去重 | 主动唤醒离线设备，30 秒去重防洪泛 |
| **LRU 限流** | hashicorp/golang-lru + Token Bucket | O(1) 淘汰，10 万设备无性能瓶颈 |
| **全局准入** | Atomic CAS | 无锁并发控制，保护 ACS 实例不过载 |
| **事件级联** | 文件传输 → 下载 → 解析 → KPI/告警 | 异步管线，Worker 可独立扩缩容 |
| **因果时序** | `device.registered` 事件保证注册先于开站 | 消除 InformHandler 与 ProvisioningEngine 的竞态条件 |

---

## 十二、SOAP/XML 报文示例详解

本节给出每个 TR069 交互流程的完整 SOAP/XML 报文示例，包括 CPE 发送的请求和 ACS 返回的响应。

### 12.1 SOAP Envelope 结构

所有 TR069 报文都遵循 SOAP 1.1 规范，使用以下命名空间：

```xml
xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
xmlns:xsd="http://www.w3.org/2001/XMLSchema"
```

### 12.2 会话建立流程（Inform ↔ InformResponse）

#### 12.2.1 CPE → ACS：Inform 请求

**场景**：设备首次上电（BOOTSTRAP）或重启（BOOT）时发送 Inform 报文，携带设备标识、事件码和初始参数列表。

```xml
POST / HTTP/1.1
Host: acs.example.com:7547
Content-Type: text/xml; charset=utf-8
SOAPAction:

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">urn:uuid:550e8400-e29b-41d4-a716-446655440000</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>BaiCell</Manufacturer>
        <OUI>00256D</OUI>
        <ProductClass>pBS3202</ProductClass>
        <SerialNumber>BC20240100001</SerialNumber>
      </DeviceId>
      <Event soap:arrayType="cwmp:EventStruct[2]">
        <EventStruct>
          <EventCode>0 BOOTSTRAP</EventCode>
          <CommandKey></CommandKey>
        </EventStruct>
        <EventStruct>
          <EventCode>1 BOOT</EventCode>
          <CommandKey></CommandKey>
        </EventStruct>
      </Event>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[8]">
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.HardwareVersion</Name>
          <Value xsi:type="xsd:string">A01</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SoftwareVersion</Name>
          <Value xsi:type="xsd:string">BaiBLQ_5.1.10</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.ManagementServer.ConnectionRequestURL</Name>
          <Value xsi:type="xsd:string">http://172.21.100.43:7547</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.Services.FAPService.1.FAPControl.LTE.OpState</Name>
          <Value xsi:type="xsd:boolean">true</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus</Name>
          <Value xsi:type="xsd:boolean">true</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.IP.Interface.1.IPv4Address.1.IPAddress</Name>
          <Value xsi:type="xsd:string">172.21.100.43</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.FAP.GPS.LockedLatitude</Name>
          <Value xsi:type="xsd:int">0</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.FAP.GPS.LockedLongitude</Name>
          <Value xsi:type="xsd:int">0</Value>
        </ParameterValueStruct>
      </ParameterList>
      <MaxEnvelopes>1</MaxEnvelopes>
      <CurrentTime>2026-03-19T10:30:00Z</CurrentTime>
      <RetryCount>0</RetryCount>
    </cwmp:Inform>
  </soap:Body>
</soap:Envelope>
```

**关键字段说明**：

| 字段 | 说明 |
|------|------|
| `DeviceId.Manufacturer` | 设备厂商名称 |
| `DeviceId.OUI` | IEEE 分配的组织唯一标识符（前 3 字节 MAC） |
| `DeviceId.ProductClass` | 产品型号 |
| `DeviceId.SerialNumber` | 设备序列号（全局唯一） |
| `Event.EventCode` | 事件码，常见值：`0 BOOTSTRAP`、`1 BOOT`、`2 PERIODIC`、`4 VALUE CHANGE`、`6 CONNECTION REQUEST`、`7 TRANSFER COMPLETE`、`10 AUTONOMOUS TRANSFER` |
| `ParameterList` | 设备参数列表，包含名称、值和类型 |
| `MaxEnvelopes` | CPE 在单个 TCP 连接中可接收的 SOAP Envelope 数量（通常为 1） |

#### 12.2.2 ACS → CPE：InformResponse 响应

```xml
HTTP/1.1 200 OK
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">urn:uuid:550e8400-e29b-41d4-a716-446655440000</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:InformResponse>
      <MaxEnvelopes>1</MaxEnvelopes>
    </cwmp:InformResponse>
  </soap:Body>
</soap:Envelope>
```

**说明**：
- `cwmp:ID` 必须与请求中的 ID 保持一致（请求-响应关联）
- `MaxEnvelopes` 告知 CPE ACS 在单个响应中最多发送多少个 SOAP Envelope

---

### 12.3 Empty POST 处理流程

#### 12.3.1 CPE → ACS：空 POST 请求

**场景**：CPE 发送 InformResponse 后，立即发送一个空 POST，表示"我准备好接收指令了"。

```http
POST / HTTP/1.1
Host: acs.example.com:7547
Content-Type: text/xml; charset=utf-8
Content-Length: 0

（无请求体）
```

#### 12.3.2 ACS → CPE：有命令时返回 RPC 请求

**场景**：如果 Redis 命令队列中有待执行命令，ACS 返回对应的 SOAP RPC 请求。

（见 12.4 节各 RPC 方法示例）

#### 12.3.3 ACS → CPE：无命令时返回 HTTP 204

**场景**：如果命令队列为空，ACS 返回 HTTP 204 结束会话。

```http
HTTP/1.1 204 No Content
```

**说明**：CPE 收到 204 后，如果还有其他事件要上报（如周期性心跳），会发起新的 Inform 会话。

---

### 12.4 RPC 方法报文示例

#### 12.4.1 GetParameterValues（获取参数值）

**场景**：ACS 主动查询设备的参数值。

**ACS → CPE：GetParameterValues 请求**

```xml
HTTP/1.1 200 OK
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-get-params-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterValues>
      <ParameterNames soap:arrayType="xsd:string[3]">
        <string>Device.DeviceInfo.SoftwareVersion</string>
        <string>Device.Services.FAPService.1.FAPControl.LTE.OpState</string>
        <string>Device.IP.Interface.1.IPv4Address.1.IPAddress</string>
      </ParameterNames>
    </cwmp:GetParameterValues>
  </soap:Body>
</soap:Envelope>
```

**CPE → ACS：GetParameterValuesResponse 响应**

```xml
POST / HTTP/1.1
Host: acs.example.com:7547
Content-Type: text/xml; charset=utf-8
SOAPAction:

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-get-params-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterValuesResponse>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[3]">
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SoftwareVersion</Name>
          <Value xsi:type="xsd:string">BaiBLQ_5.1.10</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.Services.FAPService.1.FAPControl.LTE.OpState</Name>
          <Value xsi:type="xsd:boolean">true</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.IP.Interface.1.IPv4Address.1.IPAddress</Name>
          <Value xsi:type="xsd:string">172.21.100.43</Value>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:GetParameterValuesResponse>
  </soap:Body>
</soap:Envelope>
```

**常用参数类型**：

| xsi:type | 说明 | 示例 |
|----------|------|------|
| `xsd:string` | 字符串 | `"BaiBLQ_5.1.10"` |
| `xsd:int` | 32位整数 | `0`、`1`、`42` |
| `xsd:boolean` | 布尔值 | `true`、`false` |
| `xsd:dateTime` | 日期时间 | `2026-03-19T10:30:00Z` |
| `xsd:unsignedInt` | 无符号整数 | `46000` |

---

#### 12.4.2 SetParameterValues（设置参数值）

**场景**：ACS 向设备下发配置参数。

**ACS → CPE：SetParameterValues 请求**

```xml
HTTP/1.1 200 OK
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-set-params-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:SetParameterValues>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[2]">
        <ParameterValueStruct>
          <Name>Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.TxPower</Name>
          <Value xsi:type="xsd:int">20</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth</Name>
          <Value xsi:type="xsd:string">20MHz</Value>
        </ParameterValueStruct>
      </ParameterList>
      <ParameterKey>provision-20260319-001</ParameterKey>
    </cwmp:SetParameterValues>
  </soap:Body>
</soap:Envelope>
```

**关键字段说明**：

| 字段 | 说明 |
|------|------|
| `ParameterKey` | 配置事务标识符，CPE 会在后续 Inform 中回传此值，用于配置变更追踪 |

**CPE → ACS：SetParameterValuesResponse 响应**

```xml
POST / HTTP/1.1
Host: acs.example.com:7547
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-set-params-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:SetParameterValuesResponse>
      <Status>0</Status>
    </cwmp:SetParameterValuesResponse>
  </soap:Body>
</soap:Envelope>
```

**Status 值说明**：

| Status | 含义 |
|--------|------|
| `0` | 参数已立即生效 |
| `1` | 参数已保存，需要重启后生效 |

---

#### 12.4.3 GetParameterNames（获取参数名称列表）

**场景**：ACS 查询设备支持哪些参数路径，用于发现数据模型。

**ACS → CPE：GetParameterNames 请求**

```xml
HTTP/1.1 200 OK
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-get-names-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterNames>
      <ParameterPath>Device.Services.FAPService.1.</ParameterPath>
      <NextLevel>true</NextLevel>
    </cwmp:GetParameterNames>
  </soap:Body>
</soap:Envelope>
```

**关键字段说明**：

| 字段 | 说明 |
|------|------|
| `ParameterPath` | 查询的参数路径前缀，以 `.` 结尾表示对象路径 |
| `NextLevel` | `true`=只返回下一级子节点；`false`=递归返回所有子节点 |

**CPE → ACS：GetParameterNamesResponse 响应**

```xml
POST / HTTP/1.1
Host: acs.example.com:7547
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-get-names-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterNamesResponse>
      <ParameterList soap:arrayType="cwmp:ParameterInfoStruct[4]">
        <ParameterInfoStruct>
          <Name>Device.Services.FAPService.1.CellConfig.</Name>
          <Writable>false</Writable>
        </ParameterInfoStruct>
        <ParameterInfoStruct>
          <Name>Device.Services.FAPService.1.FAPControl.</Name>
          <Writable>false</Writable>
        </ParameterInfoStruct>
        <ParameterInfoStruct>
          <Name>Device.Services.FAPService.1.Capabilities.</Name>
          <Writable>false</Writable>
        </ParameterInfoStruct>
        <ParameterInfoStruct>
          <Name>Device.Services.FAPService.1.X_COM.</Name>
          <Writable>false</Writable>
        </ParameterInfoStruct>
      </ParameterList>
    </cwmp:GetParameterNamesResponse>
  </soap:Body>
</soap:Envelope>
```

---

#### 12.4.4 AddObject（添加对象实例）

**场景**：在多实例对象（如 VLAN、ACL）中添加新的实例。

**ACS → CPE：AddObject 请求**

```xml
HTTP/1.1 200 OK
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-add-obj-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:AddObject>
      <ObjectName>Device.Services.FAPService.1.CellConfig.LTE.NeighborList.</ObjectName>
      <ParameterKey>add-neighbor-001</ParameterKey>
    </cwmp:AddObject>
  </soap:Body>
</soap:Envelope>
```

**CPE → ACS：AddObjectResponse 响应**

```xml
POST / HTTP/1.1
Host: acs.example.com:7547
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-add-obj-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:AddObjectResponse>
      <InstanceNumber>3</InstanceNumber>
      <Status>0</Status>
    </cwmp:AddObjectResponse>
  </soap:Body>
</soap:Envelope>
```

**关键字段说明**：

| 字段 | 说明 |
|------|------|
| `InstanceNumber` | 新创建实例的编号（CPE 自动分配） |
| `Status` | `0`=立即生效，`1`=需要重启 |

---

#### 12.4.5 DeleteObject（删除对象实例）

**场景**：从多实例对象中删除指定实例。

**ACS → CPE：DeleteObject 请求**

```xml
HTTP/1.1 200 OK
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-del-obj-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:DeleteObject>
      <ObjectName>Device.Services.FAPService.1.CellConfig.LTE.NeighborList.3.</ObjectName>
      <ParameterKey>del-neighbor-001</ParameterKey>
    </cwmp:DeleteObject>
  </soap:Body>
</soap:Envelope>
```

**CPE → ACS：DeleteObjectResponse 响应**

```xml
POST / HTTP/1.1
Host: acs.example.com:7547
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-del-obj-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:DeleteObjectResponse>
      <Status>0</Status>
    </cwmp:DeleteObjectResponse>
  </soap:Body>
</soap:Envelope>
```

---

#### 12.4.6 Download（下载文件）

**场景**：ACS 指示 CPE 从指定 URL 下载文件（固件、配置文件等）。

**ACS → CPE：Download 请求**

```xml
HTTP/1.1 200 OK
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-download-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Download>
      <CommandKey>firmware-upgrade-20260319-001</CommandKey>
      <FileType>1 Firmware Upgrade Image</FileType>
      <URL>http://minio.example.com:9000/firmware/BaiBLQ_5.2.0.bin</URL>
      <Username>firmware-user</Username>
      <Password>secret123</Password>
      <FileSize>52428800</FileSize>
      <TargetFileName></TargetFileName>
      <DelaySeconds>0</DelaySeconds>
      <SuccessURL></SuccessURL>
      <FailureURL></FailureURL>
    </cwmp:Download>
  </soap:Body>
</soap:Envelope>
```

**FileType 常见值**：

| 值 | 说明 |
|----|------|
| `1 Firmware Upgrade Image` | 固件升级包 |
| `2 Web Content` | Web 内容文件 |
| `3 Vendor Configuration File` | 厂商配置文件 |
| `4 Tone File` | 音频文件 |
| `5 Ringer File` | 铃声文件 |

**CPE → ACS：DownloadResponse 响应（立即）**

```xml
POST / HTTP/1.1
Host: acs.example.com:7547
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-download-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:DownloadResponse>
      <Status>1</Status>
      <StartTime>2026-03-19T10:35:00Z</StartTime>
      <CompleteTime>1970-01-01T00:00:00Z</CompleteTime>
    </cwmp:DownloadResponse>
  </soap:Body>
</soap:Envelope>
```

**Status 值说明**：

| Status | 含义 |
|--------|------|
| `0` | 下载已完成（同步下载） |
| `1` | 下载正在进行（异步下载，CPE 会稍后发送 TransferComplete） |

**CPE → ACS：TransferComplete（下载完成后上报）**

```xml
POST / HTTP/1.1
Host: acs.example.com:7547
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">tc-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:TransferComplete>
      <CommandKey>firmware-upgrade-20260319-001</CommandKey>
      <FaultStruct>
        <FaultCode>0</FaultCode>
        <FaultString></FaultString>
      </FaultStruct>
      <StartTime>2026-03-19T10:35:00Z</StartTime>
      <CompleteTime>2026-03-19T10:37:30Z</CompleteTime>
    </cwmp:TransferComplete>
  </soap:Body>
</soap:Envelope>
```

**ACS → CPE：TransferCompleteResponse**

```xml
HTTP/1.1 200 OK
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">tc-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:TransferCompleteResponse/>
  </soap:Body>
</soap:Envelope>
```

---

#### 12.4.7 Upload（上传文件）

**场景**：ACS 指示 CPE 上传文件到指定 URL（配置备份、PM/MR 文件等）。

**ACS → CPE：Upload 请求**

```xml
HTTP/1.1 200 OK
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-upload-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Upload>
      <CommandKey>backup-20260319-001</CommandKey>
      <FileType>1 Vendor Configuration File</FileType>
      <URL>http://minio.example.com:9000/backup/BC20240100001-config.xml</URL>
      <Username>backup-user</Username>
      <Password>secret456</Password>
      <DelaySeconds>0</DelaySeconds>
    </cwmp:Upload>
  </soap:Body>
</soap:Envelope>
```

**CPE → ACS：UploadResponse 响应**

```xml
POST / HTTP/1.1
Host: acs.example.com:7547
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-upload-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:UploadResponse>
      <Status>0</Status>
      <StartTime>2026-03-19T10:40:00Z</StartTime>
      <CompleteTime>2026-03-19T10:40:05Z</CompleteTime>
    </cwmp:UploadResponse>
  </soap:Body>
</soap:Envelope>
```

---

#### 12.4.8 Reboot（重启设备）

**场景**：ACS 指示 CPE 重启。

**ACS → CPE：Reboot 请求**

```xml
HTTP/1.1 200 OK
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-reboot-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Reboot>
      <CommandKey>reboot-20260319-001</CommandKey>
    </cwmp:Reboot>
  </soap:Body>
</soap:Envelope>
```

**CPE → ACS：RebootResponse 响应**

```xml
POST / HTTP/1.1
Host: acs.example.com:7547
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-reboot-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:RebootResponse/>
  </soap:Body>
</soap:Envelope>
```

**说明**：CPE 收到 RebootResponse 后会执行重启。重启完成后会发送带 `1 BOOT` 事件码的 Inform。

---

#### 12.4.9 FactoryReset（恢复出厂设置）

**场景**：ACS 指示 CPE 恢复出厂设置。

**ACS → CPE：FactoryReset 请求**

```xml
HTTP/1.1 200 OK
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-factoryreset-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:FactoryReset/>
  </soap:Body>
</soap:Envelope>
```

**CPE → ACS：FactoryResetResponse 响应**

```xml
POST / HTTP/1.1
Host: acs.example.com:7547
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-factoryreset-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:FactoryResetResponse/>
  </soap:Body>
</soap:Envelope>
```

**说明**：CPE 恢复出厂后重启，会发送带 `0 BOOTSTRAP` 事件码的 Inform。

---

#### 12.4.10 GetParameterAttributes（获取参数属性）

**场景**：查询参数的通知属性和访问控制列表。

**ACS → CPE：GetParameterAttributes 请求**

```xml
HTTP/1.1 200 OK
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-get-attr-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterAttributes>
      <ParameterNames soap:arrayType="xsd:string[2]">
        <string>Device.DeviceInfo.SoftwareVersion</string>
        <string>Device.Services.FAPService.1.FAPControl.LTE.OpState</string>
      </ParameterNames>
    </cwmp:GetParameterAttributes>
  </soap:Body>
</soap:Envelope>
```

**CPE → ACS：GetParameterAttributesResponse 响应**

```xml
POST / HTTP/1.1
Host: acs.example.com:7547
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-get-attr-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterAttributesResponse>
      <ParameterList soap:arrayType="cwmp:ParameterAttributeStruct[2]">
        <ParameterAttributeStruct>
          <Name>Device.DeviceInfo.SoftwareVersion</Name>
          <Notification>0</Notification>
          <AccessList soap:arrayType="xsd:string[1]">
            <string>Subscriber</string>
          </AccessList>
        </ParameterAttributeStruct>
        <ParameterAttributeStruct>
          <Name>Device.Services.FAPService.1.FAPControl.LTE.OpState</Name>
          <Notification>2</Notification>
          <AccessList soap:arrayType="xsd:string[0]"/>
        </ParameterAttributeStruct>
      </ParameterList>
    </cwmp:GetParameterAttributesResponse>
  </soap:Body>
</soap:Envelope>
```

**Notification 值说明**：

| 值 | 说明 |
|----|------|
| `0` | Off — 参数变更不主动通知 |
| `1` | Passive — 参数变更时在下次 Inform 中携带 VALUE CHANGE 事件 |
| `2` | Active — 参数变更时立即发起 Inform（携带 VALUE CHANGE 事件） |

---

#### 12.4.11 SetParameterAttributes（设置参数属性）

**场景**：配置参数的主动通知属性，用于实时监控关键参数变化。

**ACS → CPE：SetParameterAttributes 请求**

```xml
HTTP/1.1 200 OK
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-set-attr-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:SetParameterAttributes>
      <ParameterList soap:arrayType="cwmp:SetParameterAttributesStruct[1]">
        <SetParameterAttributesStruct>
          <Name>Device.Services.FAPService.1.FAPControl.LTE.OpState</Name>
          <NotificationChange>true</NotificationChange>
          <Notification>2</Notification>
          <AccessListChange>false</AccessListChange>
          <AccessList/>
        </SetParameterAttributesStruct>
      </ParameterList>
    </cwmp:SetParameterAttributes>
  </soap:Body>
</soap:Envelope>
```

**CPE → ACS：SetParameterAttributesResponse 响应**

```xml
POST / HTTP/1.1
Host: acs.example.com:7547
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-set-attr-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:SetParameterAttributesResponse/>
  </soap:Body>
</soap:Envelope>
```

---

#### 12.4.12 AutonomousTransferComplete（设备主动上传完成通知）

**场景**：CPE 主动上传 PM/MR 等文件后，通知 ACS 上传结果。

**CPE → ACS：AutonomousTransferComplete 请求**

```xml
POST / HTTP/1.1
Host: acs.example.com:7547
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">atc-20260319-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:AutonomousTransferComplete>
      <AnnounceURL>http://minio.example.com:9000/pm/</AnnounceURL>
      <TransferURL>http://minio.example.com:9000/pm/BC20240100001-20260319-103000.xml</TransferURL>
      <IsDownload>false</IsDownload>
      <FileType>PM Measurement File</FileType>
      <FileSize>102400</FileSize>
      <TargetFileName>BC20240100001-20260319-103000.xml</TargetFileName>
      <FaultStruct>
        <FaultCode>0</FaultCode>
        <FaultString></FaultString>
      </FaultStruct>
      <StartTime>2026-03-19T10:30:00Z</StartTime>
      <CompleteTime>2026-03-19T10:30:05Z</CompleteTime>
    </cwmp:AutonomousTransferComplete>
  </soap:Body>
</soap:Envelope>
```

**ACS → CPE：AutonomousTransferCompleteResponse 响应**

```xml
HTTP/1.1 200 OK
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">atc-20260319-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:AutonomousTransferCompleteResponse/>
  </soap:Body>
</soap:Envelope>
```

---

### 12.5 Fault 错误响应

当 RPC 方法执行失败时，CPE 返回 SOAP Fault 响应：

```xml
HTTP/1.1 500 Internal Server Error
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cmd-set-params-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <soap:Fault>
      <faultcode>Client</faultcode>
      <faultstring>CWMP fault</faultstring>
      <detail>
        <cwmp:Fault>
          <FaultCode>9003</FaultCode>
          <FaultString>Invalid arguments</FaultString>
        </cwmp:Fault>
      </detail>
    </soap:Fault>
  </soap:Body>
</soap:Envelope>
```

**常用 FaultCode**：

| FaultCode | 说明 |
|-----------|------|
| `9000` | Method not supported |
| `9001` | Request denied |
| `9002` | Internal error |
| `9003` | Invalid arguments |
| `9004` | Resources exceeded |
| `9005` | Invalid parameter name |
| `9006` | Invalid parameter type |
| `9007` | Invalid parameter value |
| `9008` | Attempt to set non-writable parameter |
| `9009` | Notification request rejected |

---

### 12.6 Connection Request 流程

#### 12.6.1 ACS → CPE：HTTP GET（唤醒请求）

**场景**：ACS 主动唤醒设备以执行操作。

```http
GET / HTTP/1.1
Host: 172.21.100.43:7547
Connection: close
```

**CPE 响应**：

```http
HTTP/1.1 200 OK
Content-Length: 0
```

#### 12.6.2 CPE → ACS：Inform（事件码 6 CONNECTION REQUEST）

**场景**：CPE 被唤醒后，向 ACS 发起带 `6 CONNECTION REQUEST` 事件码的 Inform。

```xml
POST / HTTP/1.1
Host: acs.example.com:7547
Content-Type: text/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">urn:uuid:550e8400-e29b-41d4-a716-446655440001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>BaiCell</Manufacturer>
        <OUI>00256D</OUI>
        <ProductClass>pBS3202</ProductClass>
        <SerialNumber>BC20240100001</SerialNumber>
      </DeviceId>
      <Event soap:arrayType="cwmp:EventStruct[1]">
        <EventStruct>
          <EventCode>6 CONNECTION REQUEST</EventCode>
          <CommandKey></CommandKey>
        </EventStruct>
      </Event>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[0]"/>
      <MaxEnvelopes>1</MaxEnvelopes>
      <CurrentTime>2026-03-19T10:45:00Z</CurrentTime>
      <RetryCount>0</RetryCount>
    </cwmp:Inform>
  </soap:Body>
</soap:Envelope>
```

之后进入正常的会话流程（InformResponse → Empty POST → RPC 下发）。

---

### 12.7 完整会话示例（配置下发）

以下是一个完整的配置下发会话示例，展示从设备连接到配置生效的完整报文序列：

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ 1. CPE → ACS: Inform (EventCode=6 CONNECTION REQUEST)                        │
│    设备被 Connection Request 唤醒后发起                                        │
└──────────────────────────────────────────────────────────────────────────────┘
                                    ↓
┌──────────────────────────────────────────────────────────────────────────────┐
│ 2. ACS → CPE: InformResponse                                                 │
│    确认收到 Inform，告知 MaxEnvelopes=1                                       │
└──────────────────────────────────────────────────────────────────────────────┘
                                    ↓
┌──────────────────────────────────────────────────────────────────────────────┐
│ 3. CPE → ACS: Empty POST                                                     │
│    空请求体，表示准备接收指令                                                   │
└──────────────────────────────────────────────────────────────────────────────┘
                                    ↓
┌──────────────────────────────────────────────────────────────────────────────┐
│ 4. ACS → CPE: SetParameterValues                                             │
│    下发配置参数（TxPower=20, DLBandwidth=20MHz）                              │
└──────────────────────────────────────────────────────────────────────────────┘
                                    ↓
┌──────────────────────────────────────────────────────────────────────────────┐
│ 5. CPE → ACS: SetParameterValuesResponse (Status=0)                          │
│    配置已立即生效                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
                                    ↓
┌──────────────────────────────────────────────────────────────────────────────┐
│ 6. CPE → ACS: Empty POST                                                     │
│    继续等待更多指令                                                            │
└──────────────────────────────────────────────────────────────────────────────┘
                                    ↓
┌──────────────────────────────────────────────────────────────────────────────┐
│ 7. ACS → CPE: HTTP 204 No Content                                            │
│    无更多命令，会话结束                                                        │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

### 12.8 告警上报流程

#### 12.8.1 CPE → ACS：Inform（携带告警参数）

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">inform-alarm-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>BaiCell</Manufacturer>
        <OUI>00256D</OUI>
        <ProductClass>pBS3202</ProductClass>
        <SerialNumber>BC20240100001</SerialNumber>
      </DeviceId>
      <Event soap:arrayType="cwmp:EventStruct[1]">
        <EventStruct>
          <EventCode>4 VALUE CHANGE</EventCode>
          <CommandKey></CommandKey>
        </EventStruct>
      </Event>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[2]">
        <ParameterValueStruct>
          <Name>Device.FaultMgmt.CurrentAlarm.1.PerceivedSeverity</Name>
          <Value xsi:type="xsd:string">Critical</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.FaultMgmt.CurrentAlarm.1.ProbableCause</Name>
          <Value xsi:type="xsd:string">License has expired</Value>
        </ParameterValueStruct>
      </ParameterList>
      <MaxEnvelopes>1</MaxEnvelopes>
      <CurrentTime>2026-03-19T10:50:00Z</CurrentTime>
      <RetryCount>0</RetryCount>
    </cwmp:Inform>
  </soap:Body>
</soap:Envelope>
```

**说明**：当设备启用 Active Notification（Notification=2）的参数发生变化时，设备会主动发送带 `4 VALUE CHANGE` 事件码的 Inform。
