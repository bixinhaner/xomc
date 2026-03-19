# TR069 Session 设计文档

> OMC ACS 引擎的 TR069/CWMP 会话管理系统设计与实现说明。

---

## 1. 概述

TR069/CWMP 协议基于 HTTP 之上的 SOAP/XML 通信，是一种**有状态的请求-响应协议**。每次设备（CPE）与 ACS 的交互都在一个"会话"（Session）中完成，包含一轮或多轮 HTTP 请求/响应交换。

OMC 的 Session 设计核心目标：

| 目标 | 说明 |
|------|------|
| 协议合规 | 严格遵循 TR069 规范的会话生命周期 |
| 高并发 | 支持 10,000+ 并发会话（可配置至更高） |
| 资源安全 | 会话异常终止时自动回收资源，防止泄漏 |
| 可观测 | 全面的 Prometheus 指标监控 |
| 可测试 | 接口抽象 + Mock 实现，单元测试覆盖全部状态转换 |

---

## 2. 核心数据结构

### 2.1 Session 结构体

```go
// 文件: omcgo/internal/acs/session.go

type Session struct {
    DeviceSN     string       `json:"device_sn"`      // 设备序列号（唯一标识）
    State        SessionState `json:"state"`           // 当前状态
    LastRPC      string       `json:"last_rpc"`        // 最近一次下发的 RPC 方法名
    InstanceID   string       `json:"instance_id"`     // ACS 实例标识（RemoteAddr）
    StartedAt    time.Time    `json:"started_at"`      // 会话开始时间
    UpdatedAt    time.Time    `json:"updated_at"`      // 最后更新时间
    InformEvents []string     `json:"inform_events"`   // Inform 携带的事件码列表
    CWMPId       string       `json:"cwmp_id"`         // SOAP Header 中的 CWMP ID
}
```

### 2.2 会话状态枚举

```go
type SessionState string

const (
    StateIdle           SessionState = "IDLE"              // 空闲/初始态
    StateInformReceived SessionState = "INFORM_RECEIVED"   // 收到 Inform 消息
    StateProcessing     SessionState = "PROCESSING"        // 正在处理（检查命令队列）
    StateRPCPending     SessionState = "RPC_PENDING"       // 已下发 RPC 请求，等待响应
    StateRPCResponse    SessionState = "RPC_RESPONSE"      // 收到 RPC 响应
    StateComplete       SessionState = "COMPLETE"          // 会话完成
)
```

### 2.3 连接级追踪条目

```go
// 文件: omcgo/internal/acs/handler.go

type connSessionEntry struct {
    DeviceSN  string    // 设备序列号
    CreatedAt time.Time // 绑定创建时间（用于过期清理）
}
```

---

## 3. 状态机设计

### 3.1 状态转换规则

```go
var validTransitions = map[SessionState][]SessionState{
    StateIdle:           {StateInformReceived},
    StateInformReceived: {StateProcessing, StateComplete},
    StateProcessing:     {StateRPCPending, StateComplete},
    StateRPCPending:     {StateRPCResponse},
    StateRPCResponse:    {StateProcessing, StateRPCPending, StateComplete},
    StateComplete:       {StateIdle},
}
```

### 3.2 状态转换图

```
                              ┌─────────────────────────────────────┐
                              │                                     │
                              ▼                                     │
┌──────┐   Inform    ┌───────────────┐   Empty POST   ┌────────────┤──┐
│ IDLE │────────────▶│INFORM_RECEIVED│──────────────▶│ PROCESSING │  │
└──────┘             └───────┬───────┘               └─────┬──────┘  │
   ▲                         │                             │         │
   │                         │ 无命令                 有命令│         │
   │                         │                             ▼         │
   │                         │                     ┌─────────────┐   │
   │                         │                     │ RPC_PENDING │   │
   │                         │                     └──────┬──────┘   │
   │                         │                            │          │
   │                         │                  CPE 响应  │          │
   │                         │                            ▼          │
   │                         │                    ┌──────────────┐   │
   │                         │                    │ RPC_RESPONSE │───┤
   │                         │                    └───┬──────┬───┘   │
   │                         │                        │      │       │
   │                         │             有更多命令  │      │无命令  │
   │                         │                        │      │       │
   │                         │                        ▼      │       │
   │                         │                  ┌───────────┐│       │
   │                         │                  │RPC_PENDING││       │
   │                         │                  │  (循环)   ││       │
   │                         │                  └───────────┘│       │
   │                         │                               │       │
   │                         ▼                               ▼       │
   │                    ┌──────────┐◀─────────────────────────┘       │
   └────────────────────│ COMPLETE │◀────────────────────────────────┘
                        └──────────┘
```

### 3.3 各状态详细说明

| 状态 | 含义 | 进入条件 | 退出条件 |
|------|------|---------|---------|
| `IDLE` | 设备无活跃会话 | 会话完成后逻辑上回到此状态 | 收到 Inform 消息 |
| `INFORM_RECEIVED` | ACS 已收到 Inform 并返回 InformResponse | CPE 发送 Inform | CPE 发送 Empty POST 或无命令直接完成 |
| `PROCESSING` | ACS 正在检查命令队列 | CPE 发送 Empty POST 或 RPC 响应后回到处理 | 有命令则转 RPC_PENDING，无命令则转 COMPLETE |
| `RPC_PENDING` | ACS 已下发 RPC 请求（如 GetParameterValues），等待 CPE 响应 | 命令队列弹出命令并��建 SOAP 请求发送 | CPE 返回 RPC 响应 |
| `RPC_RESPONSE` | ACS 收到 CPE 的 RPC 响应 | CPE 发送 RPC 响应（如 GetParameterValuesResponse） | 检查是否有更多命令：有则转 RPC_PENDING，无则转 COMPLETE |
| `COMPLETE` | 会话结束，资源释放 | 命令队列为空时从 PROCESSING / RPC_RESPONSE / INFORM_RECEIVED 转入 | 逻辑上回到 IDLE |

### 3.4 状态转换安全保护

`TransitionTo()` 方法在转换前校验合法性，非法转换返回错误，防止状态机进入不一致状态：

```go
func (s *Session) TransitionTo(newState SessionState) error {
    allowed, ok := validTransitions[s.State]
    if !ok {
        return fmt.Errorf("no transitions defined for state %s", s.State)
    }
    for _, a := range allowed {
        if a == newState {
            s.State = newState
            s.UpdatedAt = time.Now()
            return nil
        }
    }
    return fmt.Errorf("invalid session transition: %s → %s", s.State, newState)
}
```

---

## 4. 两层会话追踪架构

OMC 采用**两层追踪**来管理 TR069 会话：

### 4.1 Layer 1: Redis 域级会话（SessionStore）

| 维度 | 说明 |
|------|------|
| 存储 | Redis |
| Key 格式 | `acs:session:{device_sn}` |
| 内容 | Session 结构体的 JSON 序列化 |
| TTL | 可配置，默认 5 分钟 |
| 粒度 | 每设备一个 |
| 用途 | 跨 HTTP 请求持久化会话状态 |

**接口定义：**

```go
// 文件: omcgo/internal/acs/session_store.go

type SessionStore interface {
    Create(ctx context.Context, deviceSN string, session *Session) error
    Get(ctx context.Context, deviceSN string) (*Session, error)
    Update(ctx context.Context, deviceSN string, session *Session) error
    Delete(ctx context.Context, deviceSN string) error
    SetTTL(ctx context.Context, deviceSN string, ttl time.Duration) error
}
```

**设计要点：**

- 每次 Create/Update 都重置 TTL，确保活跃会话不会过期
- Redis TTL 作为兜底机制，即使 ACS 崩溃，会话也会自动过期
- 接口抽象使得测试时可以用内存 Mock 替代

### 4.2 Layer 2: 内存连接映射（connSessions）

| 维度 | 说明 |
|------|------|
| 存储 | 进程内 `sync.Map` |
| Key | HTTP `RemoteAddr`（如 `192.168.1.1:5000`） |
| Value | `connSessionEntry{DeviceSN, CreatedAt}` |
| 用途 | 将 TCP 连接映射到设备序列号 |

**为什么需要两层？**

TR069 协议的特点是：**Inform 消息携带设备标识，但后续的 Empty POST 和 RPC 响应不携带设备标识**。因此需要 connSessions 层将 TCP 连接（RemoteAddr）映射到设备 SN，使后续请求能找到对应的 Redis 会话。

```
CPE 请求 1: Inform (包含 DeviceSN)
  → 创建 Redis Session + connSessions[RemoteAddr] = DeviceSN

CPE 请求 2: Empty POST (不包含 DeviceSN)
  → connSessions[RemoteAddr] → DeviceSN → Redis Session

CPE 请求 3: RPCResponse (不包含 DeviceSN)
  → connSessions[RemoteAddr] → DeviceSN → Redis Session
```

---

## 5. 完整会话生命周期

### 5.1 场景 A：无命令的简单会话（Inform → Complete）

```
CPE                         ACS                        Redis              connSessions
 │                           │                          │                    │
 │──── Inform (SN=ABC) ────▶│                          │                    │
 │                           │── Create Session ───────▶│ acs:session:ABC    │
 │                           │                          │ State=INFORM_RECV  │
 │                           │── Store Binding ─────────│───────────────────▶│ RemoteAddr→ABC
 │                           │── Acquire Admission ─────│                    │
 │                           │── Publish Events ────────│                    │
 │◀─── InformResponse ──────│                          │                    │
 │                           │                          │                    │
 │──── Empty POST ──────────▶│                          │                    │
 │                           │── Lookup RemoteAddr ─────│───────────────────▶│ → ABC
 │                           │── Get Session ──────────▶│ → INFORM_RECEIVED  │
 │                           │── Update: PROCESSING ──▶│                    │
 │                           │── Pop CmdQueue ─────────▶│ → (empty)          │
 │                           │                          │                    │
 │                           │── completeSession() ─────│                    │
 │                           │   ├ Delete connSessions  │                    │◀── Delete
 │                           │   ├ Release Admission    │                    │
 │                           │   ├ Dec ActiveSessions   │                    │
 │                           │   └ Update: COMPLETE ───▶│ State=COMPLETE     │
 │◀─── 204 No Content ──────│                          │                    │
```

### 5.2 场景 B：带 RPC 命令的会话（Inform → RPC → Complete）

```
CPE                         ACS                        Redis              CmdQueue
 │                           │                          │                    │
 │──── Inform (SN=ABC) ────▶│                          │                    │
 │                           │── Create Session ───────▶│ INFORM_RECEIVED    │
 │◀─── InformResponse ──────│                          │                    │
 │                           │                          │                    │
 │──── Empty POST ──────────▶│                          │                    │
 │                           │── Update: PROCESSING ──▶│                    │
 │                           │── Pop CmdQueue ─────────▶│───────────────────▶│ GetParameterValues
 │                           │── Update: RPC_PENDING ──▶│ LastRPC=GPV        │
 │◀─── GetParameterValues ──│                          │                    │
 │                           │                          │                    │
 │──── GPV Response ────────▶│                          │                    │
 │                           │── Update: RPC_RESPONSE ─▶│                    │
 │                           │── Record RPC Duration    │                    │
 │                           │── Publish Response Event │                    │
 │                           │── Pop CmdQueue ─────────▶│───────────────────▶│ (empty)
 │                           │── completeSession() ─────│ COMPLETE           │
 │◀─── 204 No Content ──────│                          │                    │
```

### 5.3 场景 C：多步 RPC 链式命令

```
CPE                         ACS                        CmdQueue
 │                           │                          │
 │──── Inform ──────────────▶│                          │ [GPV, SPV, Reboot]
 │◀─── InformResponse ──────│                          │
 │                           │                          │
 │──── Empty POST ──────────▶│── Pop ──────────────────▶│ → GPV
 │◀─── GetParameterValues ──│                          │ [SPV, Reboot]
 │                           │                          │
 │──── GPV Response ────────▶│── Pop ──────────────────▶│ → SPV
 │◀─── SetParameterValues ──│                          │ [Reboot]
 │                           │                          │
 │──── SPV Response ────────▶│── Pop ──────────────────▶│ → Reboot
 │◀─── Reboot ──────────────│                          │ (empty)
 │                           │                          │
 │──── Reboot Response ─────▶│── Pop ──────────────────▶│ → (empty)
 │                           │── completeSession()      │
 │◀─── 204 No Content ──────│                          │
```

状态变化序列：
```
IDLE → INFORM_RECEIVED → PROCESSING → RPC_PENDING → RPC_RESPONSE
     → RPC_PENDING → RPC_RESPONSE → RPC_PENDING → RPC_RESPONSE → COMPLETE
```

---

## 6. 会话保护机制

### 6.1 准入控制（Admission Controller）

```go
// 文件: omcgo/internal/acs/admission.go

type AdmissionController struct {
    maxSessions     int64          // 最大并发会话数（默认 10,000）
    currentSessions atomic.Int64   // 当前活跃会话数（原子操作）
}
```

**工作原理：**

- 收到 Inform 时调用 `Acquire()`，使用 CAS 原子操作获取会话槽位
- 会话完成时调用 `Release()` 释放槽位
- 达到上限时返回 HTTP 503 Service Unavailable
- 无锁设计（atomic），高并发下无竞争

**生命周期：**

```
Inform 到达 → Acquire() 成功 → 会话处理 → completeSession() → Release()
Inform 到达 → Acquire() 失败 → 返回 503
```

### 6.2 设备级限流（Device Rate Limiter）

```go
// 文件: omcgo/internal/acs/ratelimit.go

type DeviceRateLimiter struct {
    cache  *lru.Cache[string, *DeviceEntry]  // LRU 缓存
    limit  rate.Limit                        // 令牌桶速率
    burst  int                               // 突发容量
}
```

**设计要点：**

| 参数 | 默认值 | 说明 |
|------|--------|------|
| PerDevice | 10/分钟 | 每设备每分钟最大 Inform 请求数 |
| Burst | 5 | 令牌桶突发容量 |
| MaxDevices | 100,000 | 最大追踪设备数 |
| CleanupInterval | 5 分钟 | 过期设备清理扫描间隔 |
| CleanupTimeout | 10 分钟 | 设备不活跃淘汰超时 |

**防护层次：**

```
CPE Inform → 设备限流（防单设备洪泛）→ 准入控制（防全局过载）→ 会话创建
```

### 6.3 后台会话收割器（Session Reaper）

```go
// 文件: omcgo/internal/acs/handler.go

func (h *Handler) startSessionReaper(interval, maxAge time.Duration)
```

**目的：** 清理因 TCP 连接异常断开（网络故障、设备重启等）而遗留在 connSessions 中的孤立条目。

| 参数 | 默认值 | 说明 |
|------|--------|------|
| interval | 30 秒 | 扫描间隔 |
| maxAge | 5 分钟 | 条目最大存活时间 |

**收割流程：**

1. 每 30 秒遍历 `connSessions`
2. 对每个条目，检查 `CreatedAt` 是否超过 5 分钟
3. 过期条目：
   - 从 connSessions 中删除
   - 释放 Admission 槽位
   - 递减 ActiveSessions 指标
   - 记录 SessionDuration 指标
   - 打印 WARN 日志

**与 Redis TTL 的配合：**

- Redis 会话 TTL（默认 5 分钟）处理域级数据过期
- Reaper 处理内存级 connSessions 条目清理和 Admission 槽位回收
- 两者独立运行，互为补充

### 6.4 Redis TTL 自动过期

Redis Session 在 Create 和 Update 时都设置 TTL（默认 5 分钟）。即使 ACS 进程崩溃或 Reaper 未能清理，Redis 也会自动删除过期会话数据，防止数据泄漏。

---

## 7. 命令队列集成

### 7.1 CommandQueue 接口

```go
// 文件: omcgo/internal/acs/cmdqueue/queue.go

type CommandQueue interface {
    Push(ctx context.Context, deviceSN string, cmd *Command) error
    Pop(ctx context.Context, deviceSN string) (*Command, error)
    Peek(ctx context.Context, deviceSN string) (*Command, error)
    Len(ctx context.Context, deviceSN string) (int64, error)
    Clear(ctx context.Context, deviceSN string) error
}

type Command struct {
    ID         string          `json:"id"`
    Method     string          `json:"method"`      // GetParameterValues, SetParameterValues, Download, Reboot...
    Params     json.RawMessage `json:"params"`
    Priority   int             `json:"priority"`    // 值越小优先级越高
    CreatedAt  time.Time       `json:"created_at"`
    ExpiresAt  *time.Time      `json:"expires_at"`  // 可选过期时间
    CommandKey string          `json:"command_key"`
}
```

### 7.2 Redis 实现

| 维度 | 说明 |
|------|------|
| Redis Key | `acs:cmdq:{device_sn}` |
| 数据结构 | Sorted Set |
| Score 计算 | `priority * 1e12 + timestamp`（优先级内 FIFO） |
| 过期命令 | Pop 时检查 ExpiresAt，过期则跳过取下��个 |

### 7.3 会话中的命令调度

命令队列在会话中的两个检查点被消费：

1. **handleEmpty()** — CPE 发送 Inform 后的 Empty POST，ACS 检查队列决定是否下发 RPC
2. **handleRPCResponse()** — CPE 返回 RPC 响应后，ACS 检查队列决定是否继续下发下一个 RPC

这种设计支持在单次会话中执行**多步 RPC 操作**（如：先读参数 → 写参数 → 重启）。

---

## 8. 事件驱动

### 8.1 Inform 事件发布

收到 Inform 后，根据 EventCode 发布不同的事件主题：

| EventCode | 事件 Subject | 说明 |
|-----------|-------------|------|
| 0 BOOTSTRAP | `device.inform.bootstrap` | 设备首次注册 |
| 2 PERIODIC | `device.inform.periodic` | 周期心跳 |
| 4 VALUE CHANGE | `device.inform.value_change` | 参数值变化 |
| Alarm 相关 | `device.inform.alarm` | 告警事件 |
| M Reboot | `device.inform.reboot_complete` | 重启完成 |
| 6 CONNECTION REQUEST | `device.inform.connection_request` | 连接请求响应 |
| TRANSFER COMPLETE | `device.inform.transfer_complete` | 传输完成 |

### 8.2 RPC 响应事件发布

RPC 响应处理后，根据方法类型发布对应事件：

| RPC 方法 | 事件 Subject |
|----------|-------------|
| GetParameterValuesResponse | `command.get_parameters.response` |
| SetParameterValuesResponse | `command.set_parameters.response` |
| DownloadResponse | `command.download.response` |
| UploadResponse | `command.upload.response` |
| RebootResponse | `command.reboot.response` |
| FactoryResetResponse | `command.factory_reset.response` |
| AddObjectResponse | `command.add_object.response` |
| DeleteObjectResponse | `command.delete_object.response` |
| GetParameterNamesResponse | `command.get_names.response` |
| GetParameterAttributesResponse | `command.get_attrs.response` |
| SetParameterAttributesResponse | `command.set_attrs.response` |

这些事件被其他模块（如自动开站 F09、配置管理 F02）订阅和处理。

---

## 9. 可观测性

### 9.1 Prometheus 指标

```go
// 文件: omcgo/internal/acs/metrics.go

type ACSMetrics struct {
    ActiveSessions       prometheus.Gauge           // 当前活跃会话数
    InformTotal          *prometheus.CounterVec      // Inform 消息总数（按 event_type 标签）
    RPCDuration          *prometheus.HistogramVec    // RPC 方法执行时长（按 method 标签）
    RPCErrorsTotal       *prometheus.CounterVec      // RPC 错误总数（按 method 标签）
    SessionDuration      prometheus.Histogram        // 会话持续时长分布
    RateLimitRejected    prometheus.Counter          // 被限流拒绝的请求总数
    RateLimitDeviceCount prometheus.Gauge            // 限流器追踪的设备数
}
```

**指标名称与用途：**

| 指标 | 类型 | 用途 |
|------|------|------|
| `acs_active_sessions` | Gauge | 实时监控并发会话数，配合准入控制告警 |
| `acs_inform_total` | Counter | 统计各类 Inform 事件频率 |
| `acs_rpc_duration_seconds` | Histogram | 分析 RPC 方法执行耗时 |
| `acs_rpc_errors_total` | Counter | RPC 错误率监控 |
| `acs_session_duration_seconds` | Histogram | 会话时长分布（桶：0.1s-60s） |
| `acs_rate_limit_rejected_total` | Counter | 限流触发频率 |
| `acs_rate_limit_device_count` | Gauge | 限流器管理的设备规模 |

### 9.2 结构化日志

关键日志点：

| 时机 | 级别 | 内容 |
|------|------|------|
| Inform 接收 | INFO | device_sn, oui, product_class, events, param_count |
| RPC 响应接收 | INFO | method, cwmp_id |
| 限流拒绝 | WARN | device_sn |
| 准入拒绝 | WARN | device_sn |
| 过期会话收割 | WARN | device_sn, remote_addr, age |
| 解码错误 | ERROR | 具体错误信息 |

---

## 10. 配置

```yaml
# 文件: omcgo/cmd/acs/etc/config.*.yaml

session:
  timeout: 5m              # Redis 会话 TTL
  max_concurrent: 10000    # 最大���发会话数

rate_limit:
  per_device: 10           # 每设备每分钟最大 Inform 数
  burst: 5                 # 令牌桶突发容量
  max_devices: 100000      # 限流器追踪的最大设备数
  cleanup_interval: 5m     # 过期设备清理间隔
  cleanup_timeout: 10m     # 设备不活跃淘汰超时
```

对应配置结构体：

```go
// 文件: omcgo/internal/core/appconfig/config.go

type SessionConfig struct {
    Timeout       time.Duration `mapstructure:"timeout"`
    MaxConcurrent int64         `mapstructure:"max_concurrent"`
}

type RateLimitConfig struct {
    PerDevice       int           `mapstructure:"per_device"`
    Burst           int           `mapstructure:"burst"`
    MaxDevices      int           `mapstructure:"max_devices"`
    CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
    CleanupTimeout  time.Duration `mapstructure:"cleanup_timeout"`
}
```

---

## 11. 错误码

```go
// 文件: omcgo/global/errors.go

ErrCodeACSSessionTimeout  = 3001   // 会话超时
ErrCodeACSSessionNotFound = 3002   // 会话未找到
```

---

## 12. Redis Key 命名空间

```
acs:session:{device_sn}              — 会话状态 JSON（TTL 5 分钟）
acs:cmdq:{device_sn}                 — 命令队列 Sorted Set
acs:heartbeat:{device_sn}            — 心跳时间戳（TTL = 2×inform_interval）
acs:connreq:pending:{device_sn}      — Connection Request 去重（TTL 30 秒）
```

---

## 13. 初始化与依赖注入

### 13.1 ACS 进程启动流程

```go
// 文件: omcgo/cmd/acs/main.go

func runACS(cmd *cobra.Command, args []string) error {
    // 1. 加载配置
    var cfg appconfig.ACSConfig
    appconfig.Load(cfgPath, &cfg)

    // 2. 初始化基础设施（Redis, NATS, Logger, Metrics）
    app, _ := bootstrap.InitForACS(ctx, &cfg)

    // 3. 创建 Session Store 和 Command Queue
    sessionStore := acs.NewRedisSessionStore(app.Redis, cfg.Session.Timeout)
    cmdQueue := cmdqueue.NewRedisCommandQueue(app.Redis)

    // 4. 组装依赖
    deps := acs.NewDefaultDeps(sessionStore, cmdQueue, app.EventBus, ...)

    // 5. 创建并启动 ACS Server
    //    内部自动启动 Session Reaper 和 Rate Limiter Cleanup
    acsServer := acs.NewACSServer(cfg, deps)
    acsServer.Start()
}
```

### 13.2 依赖结构

```go
// 文件: omcgo/internal/acs/server.go

type ServerDeps struct {
    SessionStore  SessionStore              // 会话存储（Redis 实现）
    CommandQueue  cmdqueue.CommandQueue      // 命令队列（Redis 实现）
    EventBus      event.EventBus            // 事件总线（NATS 或 Channel）
    Authenticator auth.DeviceAuthenticator  // CPE 认证
    RPCDispatcher *rpc.Dispatcher           // RPC 方法分发器
    RateLimiter   *DeviceRateLimiter        // 设备级限流
    Admission     *AdmissionController      // 准入控制
    Metrics       *ACSMetrics               // Prometheus 指标
    Logger        *zap.Logger               // 结构化日志
}
```

---

## 14. 测试覆盖

### 14.1 状态机单元测试

```go
// 文件: omcgo/internal/acs/session_test.go

func TestSession_ValidTransitions(t *testing.T)
// - 验证所有合法转换（7 个）
// - 验证非法转换（3 个）
// - 确保非法转换不改变当前状态
```

### 14.2 Handler 集成测试

```go
// 文件: omcgo/internal/acs/handler_test.go

// HTTP 方法验证
TestServeHTTP_NonPOST_Returns405

// 空请求处理
TestServeHTTP_EmptyBody_NoSession_Returns204
TestServeHTTP_EmptyBody_WithSession_NoCommands_CompletesSession
TestServeHTTP_EmptyBody_WithSession_HasCommand_SendsRPC

// SOAP 方法识别
TestServeHTTP_UnknownSOAP_Returns400

// Inform 流程
TestServeHTTP_Inform_Bootstrap_Success
TestServeHTTP_Inform_Periodic_PublishesPeriodicEvent
TestServeHTTP_Inform_MalformedXML_Returns400

// 流控保护
TestServeHTTP_Inform_RateLimited_Returns503
TestServeHTTP_Inform_AdmissionDenied_Returns503

// RPC 响应处理
TestServeHTTP_RPCResponse_CompletesSessionWhenNoMoreCommands
TestServeHTTP_RPCResponse_ChainsNextCommand
TestServeHTTP_RPCResponse_NoBinding_Returns204

// TransferComplete
TestServeHTTP_TransferComplete_PublishesEvent

// 资源管理
TestCompleteSession_ReleasesResources
TestCompleteSession_NilSession_StillReleasesResources

// 后台清理
TestStartSessionReaper_CleansStaleEntries

// 完整生命周期
TestFullSessionLifecycle_InformThenEmpty
TestFullSessionLifecycle_InformThenRPCThenEmpty
```

测试使用内存 Mock 实现（`acsHSessionStore`, `acsHCmdQueue`, `acsHEventBus`），无需真实 Redis/NATS 依赖。

---

## 15. 前端展示

| 使用位置 | 说明 |
|---------|------|
| 系统仪表盘 | 显示 `activeSessions`（当前活跃 ACS 会话数） |
| 系统配置页 | 显示/配置 `sessionTimeout`（会话超时时间） |

---

## 16. 关键文件索引

| 文件 | 说明 |
|------|------|
| `omcgo/internal/acs/session.go` | Session 结构体、状态枚举、状态机转换规则 |
| `omcgo/internal/acs/session_store.go` | SessionStore 接口 + Redis 实现 |
| `omcgo/internal/acs/handler.go` | 会话生命周期处理（Inform/Empty/RPC/Complete） |
| `omcgo/internal/acs/admission.go` | 并发准入控制器 |
| `omcgo/internal/acs/ratelimit.go` | 设备级令牌桶限流器 |
| `omcgo/internal/acs/metrics.go` | Prometheus 指标定义 |
| `omcgo/internal/acs/server.go` | ACS Server 创建、依赖注入、后台任务启动 |
| `omcgo/internal/acs/cmdqueue/queue.go` | 命令队列接口 + Redis Sorted Set 实现 |
| `omcgo/internal/core/appconfig/config.go` | Session 和 RateLimit 配置结构体 |
| `omcgo/internal/acs/session_test.go` | 状态机单元测试 |
| `omcgo/internal/acs/handler_test.go` | Handler 集成测试（27 个测试用例） |
| `omcgo/cmd/acs/main.go` | ACS 进程入口，组装依赖 |
| `omcgo/global/errors.go` | Session 相关错误码 |
