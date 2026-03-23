# STUN/UDP Connection Request 实现方案

> **版本**: v2.0
> **日期**: 2026-03-23
> **状态**: Phase 1-4 全部完成

---

## 1. 背景与问题

### 当前状态

TR-069 协议要求 CPE（基站）主动发起会话，OMC 无法直接建连。当前系统已实现：

| 组件 | 状态 | 位置 |
|------|------|------|
| HTTP Connection Request 客户端 | ✅ 已实现 | `internal/acs/connreq/client.go` |
| 设备模型存储 ConnectionRequestURL | ✅ 已实现 | `internal/core/model/device.go` |
| UDPConnectionRequestAddress 提取 | ✅ 已实现 | `internal/device/service.go:325` |
| Event 6 CONNECTION REQUEST 处理 | ✅ 已实现 | `internal/acs/handler.go:805` |
| Task 创建后触发 CR | ✅ 已实现 | `internal/task/service.go` 异步 wakeDevice() |
| STUN UDP Server | ✅ 已实现 | `internal/acs/stun/` (4 文件, 35 测试) |
| UDP Connection Request 发送 | ✅ 已实现 | `internal/acs/connreq/udp_sender.go` + `dispatcher.go` |

### 核心问题

1. **任务下发延迟大**：当前需等待设备下次周期性 Inform（60s-300s），无法主动唤醒设备
2. **NAT 穿透缺失**：大量基站位于 NAT 后，HTTP Connection Request URL 不可达
3. **任务创建与 CR 脱节**：`TaskService.CreateTask()` 只入队不唤醒设备

### 目标

任务创建后，秒级唤醒设备发起 Inform（Event 6），ACS 在会话中下发已排队的任务。

---

## 2. 方案概览

### 2.1 整体流程

```
用户操作（配置下发/重启/升级...）
  │
  ▼
TaskService.CreateTask()
  │ 1. 持久化到 PostgreSQL
  │ 2. 推入 Redis 任务队列
  │ 3. 触发 Connection Request  ← 新增
  │
  ▼
ConnectionRequestDispatcher（新组件）
  │
  ├─ 设备有 STUN 地址？ → UDP Connection Request
  ├─ 设备有 HTTP CR URL？ → HTTP Connection Request（已有）
  └─ 都没有？ → 等待下次周期性 Inform
  │
  ▼
设备收到 CR → 发起 Inform（Event 6 CONNECTION REQUEST）
  │
  ▼
ACS Handler → 从 Redis 队列弹出任务 → 下发 RPC
```

### 2.2 架构决策

| 决策 | 选项 | 选择 | 理由 |
|------|------|------|------|
| STUN Server 部署位置 | 独立进程 / 嵌入 ACS | **嵌入 ACS** | STUN 与 ACS 共享设备地址数据，减少进程间通信；ACS 已有 UDP 需求 |
| 地址缓存存储 | 纯内存 / Redis / DB | **Redis + 内存 L1** | 多 ACS 实例需共享地址；L1 减少 Redis 查询；重启后设备重新 STUN 上报即恢复 |
| CR 触发方式 | 同步 / 异步队列 | **异步 goroutine** | CR 不应阻塞任务创建；失败不影响任务入队 |
| 消息队列 | Redis List / NATS | **直接调用**（同进程）| Task 与 CR 在同一 app 进程中，无需消息队列；跨进程场景通过 NATS 事件 |

---

## 3. 详细设计

### 3.1 STUN UDP Server（嵌入 ACS 进程）

**新增包**: `internal/acs/stun/`

```
internal/acs/stun/
├── server.go          # UDP 监听器 + 消息分发
├── message.go         # STUN 消息编解码（RFC 5389 子集）
├── processor.go       # 标准 STUN 处理（CPE）+ 非标准处理（基站）
├── store.go           # 设备 STUN 地址缓存（内存 L1 + Redis L2）
└── server_test.go     # 测试
```

#### 3.1.1 UDP Server

```go
// server.go
type Config struct {
    ListenAddr string // ":3478"
    WorkerSize int    // runtime.NumCPU()
    BufferSize int    // 2048 bytes
}

type Server struct {
    conn     *net.UDPConn
    store    *StunStore
    logger   *zap.Logger
    metrics  *StunMetrics
    done     chan struct{}
}

func (s *Server) Start(ctx context.Context) error
func (s *Server) Stop() error
```

- 单个 UDP socket，多 goroutine 消费（`WorkerSize` 个 goroutine 并发读取）
- 收到包后判断：前 2 字节匹配 STUN Magic → 标准处理；否则 → 非标准处理

#### 3.1.2 STUN 消息编解码

只需实现 RFC 5389 的最小子集：

- **Binding Request** (0x0001)：解析 Transaction ID、可选 RESPONSE-ADDRESS 属性
- **Binding Response** (0x0101)：构造 MAPPED-ADDRESS 属性（设备公网 IP:Port）
- **Binding Error Response** (0x0111)：错误情况

```go
// message.go
type Message struct {
    Type          uint16
    TransactionID [12]byte
    Attributes    []Attribute
}

type Attribute struct {
    Type  uint16
    Value []byte
}

func Decode(data []byte) (*Message, error)
func (m *Message) Encode() []byte
```

#### 3.1.3 消息处理

```go
// processor.go

// 标准 STUN 处理（CPE 设备）
func (s *Server) handleStunBinding(msg *Message, addr *net.UDPAddr) {
    // 1. 提取设备公网 IP:Port = addr
    // 2. 存入 StunStore（以 Transaction ID 或后续 Inform 中的 SN 关联）
    // 3. 构造 Binding Response，填入 MappedAddress = addr
    // 4. 发送响应
}

// 非标准处理（基站设备）
func (s *Server) handleNonStandard(data []byte, addr *net.UDPAddr) {
    // 1. 解析 SN（UTF-8 纯文本）
    // 2. 存入 StunStore: key=SN, value={IP, Port, Time}
    // 3. 发送 "echoreply" 确认
}
```

#### 3.1.4 地址缓存（StunStore）

```go
// store.go
type StunInfo struct {
    IP        string
    Port      int
    UpdatedAt time.Time
}

type StunStore struct {
    mu    sync.RWMutex
    local map[string]*StunInfo  // L1: 内存缓存，key=设备SN
    redis redis.UniversalClient // L2: Redis 缓存
    ttl   time.Duration         // 默认 30 分钟
}

// 写入：同时写 L1 + L2
func (s *StunStore) Set(ctx context.Context, deviceSN string, info *StunInfo) error

// 读取：先 L1，miss 则查 L2 并回填 L1
func (s *StunStore) Get(ctx context.Context, deviceSN string) (*StunInfo, error)

// 批量写入（从 Inform 中收集的 UDPConnectionRequestAddress）
func (s *StunStore) SetFromInform(ctx context.Context, deviceSN, udpAddr string) error
```

**Redis Key 命名**: `acs:stun:{device_sn}` → Hash `{ip, port, updated_at}`，TTL 30 分钟

**为什么不用纯内存**：多 ACS 实例部署时需要共享地址数据。Redis 作为 L2 解决跨实例问题，L1 内存缓存避免高频 Redis 查询。

### 3.2 Connection Request Dispatcher（新组件）

**新增**: `internal/acs/connreq/dispatcher.go`

统一调度 HTTP 和 UDP 两种 Connection Request。

```go
// dispatcher.go
type Dispatcher struct {
    httpClient *Client           // 已有的 HTTP CR 客户端
    stunStore  *stun.StunStore   // STUN 地址缓存
    udpConn    *net.UDPConn      // 共享 STUN Server 的 UDP socket 或独立 socket
    logger     *zap.Logger
    metrics    *DispatcherMetrics
}

// Send 统一入口，自动选择 HTTP 或 UDP
func (d *Dispatcher) Send(ctx context.Context, deviceSN string, httpURL string) error {
    // 优先级：
    // 1. 尝试 UDP CR（如果有 STUN 地址）
    // 2. 回退到 HTTP CR（如果有 HTTP URL）
    // 3. 都没有 → 返回 ErrNoConnectionMethod（不报错，等待周期 Inform）
}
```

#### 3.2.1 UDP Connection Request 发送

两种格式：

**基站（非标准）**：
```
"infromrequest"  ← 直接发送纯文本 UDP 包到设备公网地址
```

**CPE（类 HTTP 格式）**：
```
GET http://{server_ip}:{server_port}/?ts={timestamp}&id={random}&un=dps&cn={nonce}&sig={hmac} HTTP/1.1\r\n\r\n
```

签名算法：`HMAC-SHA1(ts+id+un+cn, shared_secret)`

```go
// udp_sender.go
func (d *Dispatcher) sendUDPConnectionRequest(ctx context.Context, deviceSN string, info *stun.StunInfo) error {
    // 1. 判断设备类型（通过 SN 前缀或数据库标记）
    // 2. 构造消息
    // 3. UDP 发送到 info.IP:info.Port
    // 4. 重试 3 次（UDP 不可靠）
}
```

### 3.3 TaskService 集成（改动最小的集成点）

在 `CreateTask()` 中添加异步 CR 触发：

```go
// task/service.go 修改
type TaskService struct {
    queue      *RedisTaskQueue
    repo       *PgTaskRepository
    metrics    *TaskMetrics
    logger     *zap.Logger
    dispatcher ConnectionRequestDispatcher  // 新增
}

// ConnectionRequestDispatcher 接口（在 task 包中定义，遵循"消费者定义接口"原则）
type ConnectionRequestDispatcher interface {
    Send(ctx context.Context, deviceSN string, httpURL string) error
}

func (s *TaskService) CreateTask(ctx context.Context, req *CreateTaskRequest) (*Task, error) {
    // ... 现有逻辑（持久化 + 入队）不变 ...

    // 新增：异步触发 Connection Request 唤醒设备
    if s.dispatcher != nil {
        go func() {
            if err := s.dispatcher.Send(context.Background(), task.DeviceSN, ""); err != nil {
                s.logger.Debug("connection request after task creation",
                    zap.String("task_id", task.ID),
                    zap.String("device_sn", task.DeviceSN),
                    zap.Error(err))
            }
        }()
    }

    return task, nil
}
```

**设计要点**：
- Dispatcher 通过接口注入，保持 task 包不依赖 connreq/stun 包
- CR 失败不影响任务创建（goroutine 内只记日志）
- 需要从 DeviceRepository 查询设备的 ConnectionRequestURL（可缓存）
- 30 秒去重已由 `connreq.Client` 保证，避免重复唤醒

### 3.4 Inform 中的 UDPConnectionRequestAddress 处理增强

当前 `device/service.go:325` 已提取 `UDPConnectionRequestAddress` 存入 `device.IPAddress`。需要增强：

```go
// device/service.go 修改（UpdateDeviceFromInform 中）
func (s *DeviceService) UpdateDeviceFromInform(...) {
    // 现有逻辑...

    // 增强：将 UDPConnectionRequestAddress 同步到 STUN Store
    udpAddr := findParamValue(inform.ParameterList, "Device.ManagementServer.UDPConnectionRequestAddress")
    if udpAddr != "" && s.stunStore != nil {
        s.stunStore.SetFromInform(ctx, device.SerialNumber, udpAddr)
    }
}
```

这样即使设备从未向 STUN Server 发包，只要它在 Inform 中上报了 UDPConnectionRequestAddress，系统也能获得其公网地址用于 UDP CR。

### 3.5 数据库迁移

新增字段存储设备的 STUN/NAT 信息（可选，用于设备详情展示和故障排查）：

```sql
-- 000034_add_device_stun_fields.up.sql
ALTER TABLE devices
    ADD COLUMN IF NOT EXISTS nat_detected BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS udp_connection_request_address VARCHAR(64);

COMMENT ON COLUMN devices.nat_detected IS 'STUN 检测到设备位于 NAT 后';
COMMENT ON COLUMN devices.udp_connection_request_address IS 'UDP Connection Request 地址 (IP:Port)';

-- 000034_add_device_stun_fields.down.sql
ALTER TABLE devices
    DROP COLUMN IF EXISTS nat_detected,
    DROP COLUMN IF EXISTS udp_connection_request_address;
```

### 3.6 配置

```yaml
# cmd/acs/etc/config.dev.yaml 新增
stun:
  enabled: true
  listen_addr: ":3478"
  worker_size: 4          # 处理 goroutine 数
  buffer_size: 2048       # UDP 包最大字节
  cache_ttl: 30m          # STUN 地址缓存 TTL
  shared_secret: "dps"    # CPE UDP CR 签名密钥
```

### 3.7 Prometheus 指标

```
# STUN Server
acs_stun_packets_received_total{type="standard|nonstandard"}
acs_stun_store_size{type="cpe|enb"}

# Connection Request Dispatcher
acs_connection_request_sent_total{method="http|udp", result="success|failure"}
acs_connection_request_duration_seconds{method="http|udp"}
```

---

## 4. 实施阶段

### Phase 1: Task → CR 集成 ✅ 已完成

**目标**：任务创建后自动触发 Connection Request

**已完成**：
1. ✅ `task/service.go` — 添加 `DeviceLookup` + `ConnectionRequestSender` 消费者定义接口
2. ✅ `task/service.go` — `CreateTask()` 和 `BatchCreateTasks()` 中异步调用 `wakeDevice()`
3. ✅ `cmd/app/router/router.go` — 注入 `taskDeviceLookup` + `taskCRSender` 适配器
4. ✅ `appconfig/config.go` — 新增 `ConnReqConfig` 配置结构体
5. ✅ `cmd/app/etc/config.dev.yaml` — 添加 `conn_req` 配置节

**效果**：任务创建后立即尝试唤醒设备，优先 UDP CR（NAT 穿透），回退 HTTP CR。

### Phase 2: STUN Server + 地址缓存 ✅ 已完成

**目标**：建立 STUN 地址发现基础设施

**已完成**：
1. ✅ 新增 `internal/acs/stun/` 包（server.go、message.go、processor.go、store.go）
2. ✅ 集成到 ACS main.go（启动 STUN UDP listener）
3. ✅ 添加 `STUNConfig` 配置结构体到 `appconfig`
4. ✅ 35 个测试全部通过（消息编解码、地址缓存、消息处理器、服务器集成测试）
5. ✅ Prometheus 指标支持（packets_received、packets_invalid、store_size）

**后续已完成**：
- ✅ `device/service.go` — Inform 处理增强，UDPConnectionRequestAddress → StunStore
- ✅ 数据库迁移 `000034_add_device_stun_fields`（nat_detected + udp_connection_request_address）

### Phase 3: UDP Connection Request 发送 ✅ 已完成

**目标**：实现 UDP CR 发送，完成全链路

**已完成**：
1. ✅ `connreq/dispatcher.go` — 统一调度器（UDP 优先 → HTTP 回退 → ErrNoConnectionMethod）
2. ✅ `connreq/udp_sender.go` — UDP CR 消息构造 + HMAC-SHA1 签名 + 3 次重试发送
3. ✅ 支持两种设备格式：eNB（"infromrequest"纯文本）和 CPE（HTTP-like + HMAC 签名）
4. ✅ `connreq/udp_sender_test.go` + `connreq/dispatcher_test.go` — 测试覆盖

### Phase 4: 基站重启 + 监控 ✅ 已完成

**目标**：完善基站管理能力

**已完成**：
1. ✅ `connreq/udp_sender.go` — `SendRestart()` 发送 `/restart_{md5(SN)}` UDP 包到不可达设备
2. ✅ `connreq/metrics.go` — Dispatcher Prometheus 指标（`acs_connection_request_sent_total`、`acs_connection_request_duration_seconds`）
3. ✅ `connreq/dispatcher.go` — 集成指标记录到 Send() 方法
4. ✅ STUN Server 缓存监控已在 Phase 2 完成（`acs_stun_store_size`、`acs_stun_packets_received_total`）

---

## 5. 关键设计决策说明

### 5.1 为什么 STUN Server 嵌入 ACS 而不独立部署？

老系统 stunUDPServer 是独立 Java 进程，通过 Redis 与 TR069Server 交换数据。新系统选择嵌入 ACS：

| 维度 | 独立进程 | 嵌入 ACS |
|------|---------|---------|
| 数据共享 | 需要 Redis 中转 | 直接内存访问 |
| 部署复杂度 | 多一个进程管理 | 无额外进程 |
| 延迟 | 多一跳 Redis 查询 | 内存 O(1) |
| 扩展性 | 独立扩展 | 随 ACS 水平扩展 |
| 代码复杂度 | 需要独立 main.go | 作为 ACS 子模块 |

ACS 本身就需要水平扩展，STUN Server 嵌入后自然随之扩展。Go 的 goroutine 模型处理 UDP 比 Java Netty 更轻量。

### 5.2 为什么优先 UDP CR 而非 HTTP CR？

实际运营环境中，大量基站位于 NAT/防火墙后：
- HTTP CR URL（如 `http://192.168.1.1:7547`）是内网地址，从 OMC 侧不可达
- STUN 发现的公网地址才是可达的
- UDP CR 是 NAT 穿透的标准方案（TR-069 Amendment 5 定义）

因此 Dispatcher 优先尝试 UDP CR。

### 5.3 老系统的 RabbitMQ 重启功能如何适配？

老系统用 RabbitMQ 消费重启指令。新系统统一用 NATS JetStream：
- 重启操作通过 TaskService 创建 Reboot 类型任务
- 任务创建自动触发 CR → 设备 Inform → ACS 下发 Reboot RPC
- 特殊场景（设备完全无响应）可保留 UDP 直接重启命令（`/restart_{md5}`），作为兜底

---

## 6. 与现有模块的交互

```
┌──────────────────────────────────────────────────────────┐
│                       omcgo-app                           │
│                                                          │
│  TaskService.CreateTask()                                │
│    └─ wakeDevice() (goroutine)                           │
│         └─ taskCRSender.Send()                           │
│              └─ Dispatcher.Send()                        │
│                   ├─ UDPSender ── STUN Store (Redis L2)  │
│                   │    ├─ eNB: "infromrequest"            │
│                   │    └─ CPE: HTTP-like + HMAC-SHA1      │
│                   └─ HTTP Client (已有, Digest Auth)     │
│                                                          │
│  DeviceService.UpdateFromInform()                        │
│    └─ UDPConnectionRequestAddress → StunStore.SetFromInform()│
│                                                          │
│  SoftwareService / DeviceService                         │
│    └─ connReqClient.Send() (HTTP only, 已有)             │
│                                                          │
│  UDPSender.SendRestart()  ← 兜底重启（不可达设备）       │
│    └─ "/restart_{md5(SN)}" UDP 包                        │
└──────────────────────────────────────────────────────────┘
                          │ Redis (STUN 地址共享)
┌─────────────────────────┼────────────────────────────────┐
│                  omcgo-acs                                │
│                         │                                │
│  ┌─── STUN Server ──────┼──────┐                         │
│  │  UDP :3478                  │                         │
│  │  ├─ 标准 STUN → StunStore   │                         │
│  │  └─ 非标准 SN → StunStore   │                         │
│  │  └─ StunStore (L1+L2 Redis) │                         │
│  └─────────────────────────────┘                         │
│                                                          │
│  ┌─── ACS Handler ─────────────────────────────────┐    │
│  │  HTTP :7547                                      │    │
│  │  Inform Event 6 → 弹出 Redis 任务队列 → 下发 RPC │    │
│  └──────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────┘
```

**实际实现方案**：Dispatcher 位于 app 进程中，直接通过 Redis L2 读取 STUN 地址。
无需 gRPC/NATS 跨进程通信，因为：
- STUN 地址通过 Redis 在 app 和 acs 进程间共享
- HTTP CR 客户端本就在 app 进程中
- UDP 发送只需 `net.DialUDP()`，无需经过 ACS 进程

---

## 7. 文件清单

### 新增文件

| 文件 | 说明 | 状态 |
|------|------|------|
| `internal/acs/stun/server.go` | STUN UDP 服务器 | ✅ |
| `internal/acs/stun/message.go` | STUN 消息编解码 | ✅ |
| `internal/acs/stun/processor.go` | 标准/非标准消息处理 | ✅ |
| `internal/acs/stun/store.go` | 地址缓存（L1 内存 + L2 Redis） | ✅ |
| `internal/acs/stun/*_test.go` | STUN 测试（35 个） | ✅ |
| `internal/acs/connreq/dispatcher.go` | CR 调度器（UDP 优先 + HTTP 回退 + Prometheus 指标） | ✅ |
| `internal/acs/connreq/udp_sender.go` | UDP CR 消息构造与发送（eNB + CPE + Restart） | ✅ |
| `internal/acs/connreq/metrics.go` | Dispatcher Prometheus 指标 | ✅ |
| `internal/acs/connreq/dispatcher_test.go` | Dispatcher 测试 | ✅ |
| `internal/acs/connreq/udp_sender_test.go` | UDP Sender 测试 | ✅ |
| `migrations/000034_add_device_stun_fields.up.sql` | DB 迁移（nat_detected + udp_connection_request_address） | ✅ |
| `migrations/000034_add_device_stun_fields.down.sql` | DB 回滚 | ✅ |

### 修改文件

| 文件 | 修改内容 | 状态 |
|------|---------|------|
| `internal/task/service.go` | 添加 DeviceLookup/ConnectionRequestSender 接口 + wakeDevice() 异步触发 | ✅ |
| `internal/core/appconfig/config.go` | 新增 STUNConfig（ACS）和 ConnReqConfig（App） | ✅ |
| `internal/core/model/device.go` | 新增 NatDetected、UDPConnectionRequestAddress 字段 | ✅ |
| `internal/device/service.go` | Inform 处理增强：UDPAddr → 设备模型 + StunStore 同步 | ✅ |
| `internal/device/pg_repository.go` | CRUD 支持 nat_detected + udp_connection_request_address 列 | ✅ |
| `cmd/acs/main.go` | 启动 STUN Server | ✅ |
| `cmd/acs/etc/config.dev.yaml` | 添加 stun 配置节 | ✅ |
| `cmd/app/router/router.go` | 注入 StunStore + UDPSender + Dispatcher + Metrics 到各 Service | ✅ |
| `cmd/app/etc/config.dev.yaml` | 添加 conn_req 配置节 | ✅ |
| 各运营商 `params.go` | 无需修改：UDPConnectionRequestAddress 是标准 TR-069 参数，直接提取 | ✅ 已确认 |

---

## 8. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| UDP 包丢失导致 CR 失败 | 设备无法及时唤醒 | 重试 3 次 + 回退到等待周期 Inform |
| NAT 映射过期 | STUN 缓存地址失效 | TTL 30 分钟 + 设备重新 STUN 刷新 |
| 大量并发 CR 造成网络风暴 | ACS 出口带宽压力 | 30 秒去重 + 批量操作限流 |
| STUN 端口被防火墙封禁 | 无法建立 UDP 通道 | 运维确保 3478 端口开放；降级到 HTTP CR |
| 多 ACS 实例重复发送 CR | 浪费带宽 | Redis 去重机制（已有） |
