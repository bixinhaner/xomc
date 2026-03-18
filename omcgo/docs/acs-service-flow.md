# ACS 服务完整实现流程分析

> 本文档分析 ACS（TR-069 自动配置服务器）引擎的完整实现流程，包含请求处理、EventBus 发布/订阅、命令队列反向通道等核心链路。

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

### 6.4 Connection Request 唤醒机制

当设备不在线（无活跃会话）时，推送命令后需要主动唤醒设备：

```go
// internal/acs/connreq/client.go
type ConnReqClient interface {
    Send(ctx context.Context, deviceSN, url string) error
}
```

- 通过 HTTP GET 请求设备的 ConnectionRequestURL
- **去重**：Redis Key `acs:connreq:pending:{device_serial}`，TTL 30 秒
- 支持 Digest/Basic 认证
- 设备收到 Connection Request 后主动发起新的 TR-069 会话

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
