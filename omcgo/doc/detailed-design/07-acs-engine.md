# DD-07: ACS 引擎核心（F01）

> 关联功能域：F01（南向接口管理）
> 关联 backend-design.md 章节：第二章（模块 1）、第四章（TR069 ACS 引擎设计）
> 实施阶段：Phase 1（基础 Inform）→ Phase 2（完整 RPC、命令队列、ConnReq）
> 依赖文档：DD-02, DD-04, DD-06

---

## 1. 概述

### 1.1 模块定位

ACS 引擎（`internal/acs/`）是 OMC 系统的南向通信核心，处理所有 TR069/CWMP 协议通信。作为独立进程（`omcgo-acs`）部署，支持水平扩展。

### 1.2 核心职责

- HTTP/HTTPS 服务器，接受 CPE 连接
- SOAP/XML 编解码
- TR069 会话状态机管理
- 9 种 RPC 方法实现
- 设备命令队列（Redis-backed）
- Connection Request 发起
- CPE 认证（HTTP Digest/Basic）
- 流量控制（设备级限流 + 全局准入控制）

### 1.3 与其他模块的交互关系

```
CPE 设备 ←→ [ACS 引擎] ←gRPC→ [App 主应用]
                ↕                    ↕
            [Redis]              [PostgreSQL]
            (会话/命令)           (设备/配置)
                ↕
            [NATS/EventBus]
            (事件通知)
```

---

## 2. 接口设计

### 2.1 ACSServer 结构体 — `internal/acs/server.go`

```go
type ACSServer struct {
    httpServer     *http.Server
    sessionStore   SessionStore
    commandQueue   CommandQueue
    eventBus       event.EventBus
    deviceAuth     DeviceAuthenticator
    soapCodec      *soap.Codec
    rpcDispatcher  *RPCDispatcher
    rateLimiter    *DeviceRateLimiter
    admission      *AdmissionController
    metrics        *ACSMetrics
    logger         *zap.Logger
}

func NewACSServer(cfg ACSConfig, deps ACSServerDeps) *ACSServer
func (s *ACSServer) Start() error
func (s *ACSServer) Shutdown(ctx context.Context) error
```

### 2.2 SessionStore 接口 — `internal/acs/session_store.go`

```go
type SessionStore interface {
    Create(ctx context.Context, deviceSN string, session *Session) error
    Get(ctx context.Context, deviceSN string) (*Session, error)
    Update(ctx context.Context, deviceSN string, session *Session) error
    Delete(ctx context.Context, deviceSN string) error
    SetTTL(ctx context.Context, deviceSN string, ttl time.Duration) error
}
```

Redis 实现：`acs:session:{device_serial}` → Hash，TTL 5 分钟

### 2.3 CommandQueue 接口 — `internal/acs/cmdqueue/queue.go`

```go
type CommandQueue interface {
    // Push 向设备命令队列添加命令
    Push(ctx context.Context, deviceSN string, cmd *Command) error
    // Pop 取出优先级最高的命令
    Pop(ctx context.Context, deviceSN string) (*Command, error)
    // Peek 查看下一条命令（不移除）
    Peek(ctx context.Context, deviceSN string) (*Command, error)
    // Len 队列长度
    Len(ctx context.Context, deviceSN string) (int64, error)
    // Clear 清空队列
    Clear(ctx context.Context, deviceSN string) error
}

type Command struct {
    ID         string          `json:"id"`
    Method     string          `json:"method"` // GetParameterValues, SetParameterValues, Download...
    Params     json.RawMessage `json:"params"`
    Priority   int             `json:"priority"` // 数值越高优先级越高
    CreatedAt  time.Time       `json:"created_at"`
    ExpiresAt  *time.Time      `json:"expires_at,omitempty"`
    CommandKey string          `json:"command_key"`
}
```

Redis 实现：`acs:cmdq:{device_serial}` → Sorted Set（score = priority）

### 2.4 DeviceAuthenticator 接口 — `internal/acs/auth/`

```go
type DeviceAuthenticator interface {
    Authenticate(r *http.Request) (*DeviceIdentity, error)
}

type DeviceIdentity struct {
    SerialNumber string
    OUI          string
    Carrier      CarrierCode
}
```

### 2.5 gRPC 接口（ACS ↔ App）— `api/proto/acs.proto`

```protobuf
service ACSControl {
    rpc QueueCommand(QueueCommandRequest) returns (QueueCommandResponse);
    rpc SendConnectionRequest(ConnectionRequestMsg) returns (ConnectionRequestResponse);
    rpc StreamDeviceEvents(StreamEventsRequest) returns (stream DeviceEvent);
    rpc GetStatus(StatusRequest) returns (ACSStatus);
}
```

---

## 3. 详细设计

### 3.1 HTTP 请求处理流程 — `internal/acs/handler.go`

```
CPE HTTP POST (Content-Type: text/xml)
  │
  ├── 1. 限流检查（DeviceRateLimiter）
  │     └── 超限 → 返回 503
  │
  ├── 2. 准入控制（AdmissionController）
  │     └── 超限 → 返回 503
  │
  ├── 3. 认证（DeviceAuthenticator）
  │     └── 失败 → 返回 401
  │
  ├── 4. SOAP 解析（检测 RPC 方法类型）
  │     ├── Inform → handleInform()
  │     ├── TransferComplete → handleTransferComplete()
  │     ├── GetParameterValuesResponse → handleRPCResponse()
  │     ├── SetParameterValuesResponse → handleRPCResponse()
  │     ├── ... 其他 RPC 响应
  │     └── 空 POST（会话结束标志）→ handleEmpty()
  │
  ├── 5. 事件处理 + 命令队列检查
  │
  └── 6. 生成 SOAP 响应（或空响应结束会话）
```

### 3.2 会话状态机 — `internal/acs/session.go`

```go
type SessionState string

const (
    StateIdle            SessionState = "IDLE"
    StateInformReceived  SessionState = "INFORM_RECEIVED"
    StateProcessing      SessionState = "PROCESSING"
    StateRPCPending      SessionState = "RPC_PENDING"
    StateRPCResponse     SessionState = "RPC_RESPONSE"
    StateComplete        SessionState = "COMPLETE"
)

type Session struct {
    DeviceSN    string       `json:"device_sn"`
    State       SessionState `json:"state"`
    LastRPC     string       `json:"last_rpc"`
    InstanceID  string       `json:"instance_id"` // ACS 实例标识
    StartedAt   time.Time    `json:"started_at"`
    UpdatedAt   time.Time    `json:"updated_at"`
    InformEvents []string    `json:"inform_events"`
}
```

**状态转移规则**：

```
IDLE
  │ 收到 Inform
  ↓
INFORM_RECEIVED
  │ 发送 InformResponse
  ↓
PROCESSING
  │ 检查命令队列
  ├── 有命令 → 发送 RPC 请求 → RPC_PENDING
  └── 无命令 → 发送空响应 → COMPLETE

RPC_PENDING
  │ 收到 CPE 的 RPC 响应
  ↓
RPC_RESPONSE
  │ 处理响应，检查下一条命令
  ├── 有命令 → 发送 RPC 请求 → RPC_PENDING
  └── 无命令 → 发送空响应 → COMPLETE

COMPLETE
  │ 清理会话
  ↓
IDLE
```

### 3.3 Inform 处理流程 — `handleInform()`

```
1. xml.Decoder 流式解析 Inform 报文
2. 提取 DeviceId（OUI, ProductClass, SerialNumber）
3. 提取事件列表（BOOTSTRAP/PERIODIC/VALUE_CHANGE/ALARM/TRANSFER_COMPLETE）
4. 创建/更新会话状态（Redis）
5. 根据事件类型发布到 EventBus：
   ├── BOOTSTRAP → device.inform.bootstrap（开站模块监听）
   ├── PERIODIC → device.inform.periodic（心跳更新）
   ├── VALUE_CHANGE → device.inform.value_change
   ├── ALARM → device.inform.alarm（告警模块监听）
   └── TRANSFER_COMPLETE → device.inform.transfer_complete（PM/MR 模块监听）
6. 更新心跳：SET acs:heartbeat:{device_serial} TTL=2×inform_interval
7. 渲染 InformResponse SOAP 模板
8. 返回 HTTP 200
```

### 3.4 RPC 方法实现 — `internal/acs/rpc/`

每个 RPC 方法独立文件：

| 文件 | RPC 方法 | 方向 | 说明 |
|------|---------|------|------|
| `get_parameter_values.go` | GetParameterValues | ACS→CPE | 读取设备参数 |
| `set_parameter_values.go` | SetParameterValues | ACS→CPE | 写入设备参数 |
| `get_parameter_names.go` | GetParameterNames | ACS→CPE | 发现参数路径 |
| `add_object.go` | AddObject | ACS→CPE | 创建多实例对象 |
| `delete_object.go` | DeleteObject | ACS→CPE | 删除多实例对象 |
| `download.go` | Download | ACS→CPE | 推送文件（固件/配置） |
| `upload.go` | Upload | ACS→CPE | 请求文件（PM/MR/日志） |
| `reboot.go` | Reboot | ACS→CPE | 远程重启 |
| `factory_reset.go` | FactoryReset | ACS→CPE | 恢复出厂 |
| `schedule_inform.go` | ScheduleInform | ACS→CPE | 调度 Inform |

**RPCDispatcher**：

```go
type RPCDispatcher struct {
    handlers map[string]RPCHandler
}

type RPCHandler interface {
    // BuildRequest 构建 RPC 请求 SOAP XML
    BuildRequest(cmd *Command) ([]byte, error)
    // HandleResponse 处理 CPE 的 RPC 响应
    HandleResponse(ctx context.Context, session *Session, response []byte) error
}
```

### 3.5 Connection Request — `internal/acs/connreq/client.go`

```go
type ConnReqClient struct {
    httpClient *http.Client
    redis      redis.UniversalClient
    logger     *zap.Logger
}

// Send 向 CPE 发送 Connection Request
// 1. 去重检查：acs:connreq:pending:{device_serial}（TTL 30s）
// 2. 向 CPE 的 ConnectionRequestURL 发送 HTTP GET（带 Digest 认证）
// 3. 超时重试（最多 3 次，指数退避）
func (c *ConnReqClient) Send(ctx context.Context, device *Device) error
```

### 3.6 流量控制

**设备级限流** — `internal/acs/rate_limiter.go`：

```go
type DeviceRateLimiter struct {
    limiters sync.Map // map[string]*rate.Limiter
    limit    rate.Limit // 每设备每分钟最大请求数
    burst    int
}

func (rl *DeviceRateLimiter) Allow(deviceSN string) bool
```

**全局准入控制** — `internal/acs/admission.go`：

```go
type AdmissionController struct {
    maxConcurrentSessions int64
    currentSessions       atomic.Int64
}

func (ac *AdmissionController) Acquire() bool
func (ac *AdmissionController) Release()
```

---

## 4. 运营商差异

| 差异点 | CMCC | CTCC | CUCC |
|--------|------|------|------|
| 根对象路径 | `Device.` | `Device.` 或 `InternetGatewayDevice.` | `Device.` |
| 自定义事件码 | `X_CMCC_*` | `X_CT_*` | `X_CUCC_*` |
| Digest 认证 | 必选 | 可选 | 可选 |
| Connection Request | 标准 | 标准 | 标准 |

运营商差异通过 Carrier 适配器处理（DD-05），ACS 引擎本身保持运营商无关。

---

## 5. 实施子阶段

### 阶段 7a：server + handler + SOAP 解析（仅 Inform）+ 空响应（Phase 1）

**交付物**：
- `server.go` — HTTP 服务器启动
- `handler.go` — Inform 接收处理
- `soap/codec.go` — SOAP 编解码
- `soap/inform.go` — Inform 解析

**验证**：使用 curl 发送 Inform SOAP XML，收到 InformResponse

### 阶段 7b：会话状态机 + SessionStore（Phase 1）

**交付物**：
- `session.go` — 会话状态机
- `session_store.go` — Redis 实现

**验证**：会话状态正确转移，Redis 中可查看会话数据

### 阶段 7c：CPE 认证（Phase 1）

**交付物**：`auth/digest.go`、`auth/basic.go`
**验证**：Digest/Basic 认证功能测试

### 阶段 7d：RPC 方法（Phase 1 仅 GetParameterValues；Phase 2 补全全部）

**交付物**：`rpc/` 下 9 个 RPC 方法实现
**验证**：每个 RPC 方法的 SOAP 请求/响应正确

### 阶段 7e：命令队列 + Connection Request（Phase 2）

**交付物**：`cmdqueue/queue.go`、`connreq/client.go`
**验证**：命令入队/出队、Connection Request 发送

### 阶段 7f：流量控制 + 指标（Phase 2）

**交付物**：`rate_limiter.go`、`admission.go`、Prometheus 指标
**验证**：限流功能测试、metrics 端点验证

### 阶段 7g：gRPC 接口（Phase 2）

**交付物**：`api/proto/acs.proto`、gRPC server/client 代码
**验证**：gRPC 端到端调用测试

---

## 6. 文件清单

```
internal/acs/server.go
internal/acs/handler.go
internal/acs/session.go
internal/acs/session_store.go
internal/acs/soap/codec.go
internal/acs/soap/inform.go
internal/acs/soap/templates.go
internal/acs/rpc/dispatcher.go
internal/acs/rpc/get_parameter_values.go
internal/acs/rpc/set_parameter_values.go
internal/acs/rpc/get_parameter_names.go
internal/acs/rpc/add_object.go
internal/acs/rpc/delete_object.go
internal/acs/rpc/download.go
internal/acs/rpc/upload.go
internal/acs/rpc/reboot.go
internal/acs/rpc/factory_reset.go
internal/acs/rpc/schedule_inform.go
internal/acs/connreq/client.go
internal/acs/cmdqueue/queue.go
internal/acs/auth/digest.go
internal/acs/auth/basic.go
internal/acs/rate_limiter.go
internal/acs/admission.go
internal/acs/metrics.go
api/proto/acs.proto
```

---

## 7. 测试策略

- **单元测试**：SOAP 解析、会话状态转移、限流逻辑、认证
- **集成测试**：完整 Inform → InformResponse 流程（需要 Redis）
- **E2E 测试**：模拟 CPE 发送完整 TR069 会话
- **Fixture**：各运营商典型 SOAP 报文

---

## 8. 依赖与风险

- **风险**：SOAP/XML 解析性能，需要 benchmark 验证流式解析效果
- **风险**：会话超时处理，需确保 Redis TTL 与 HTTP Keep-Alive 协调
- **依赖**：Redis（会话存储、命令队列）、NATS（事件总线）

---

## 9. 参考

- backend-design.md 第四章：TR069 ACS 引擎设计
- doc/features/01-southbound-interface.md：F01 全部子功能
- doc/architecture/interface-topology.md：南向接口协议栈
