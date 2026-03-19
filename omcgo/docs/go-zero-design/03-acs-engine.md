# 03 — ACS 引擎设计

> ACS（Auto Configuration Server）引擎在 go-zero 架构中的定位与实现方案

---

## 1. 架构定位

ACS 引擎是 OMC 系统的核心组件，负责通过 TR069/CWMP 协议与 10万~100万台基站设备通信。由于 go-zero 不支持 SOAP/XML 协议，ACS 引擎作为**独立进程**运行，通过 **zRPC** 接入 go-zero 微服务生态。

```
                          go-zero 微服务生态
                    ┌─────────────────────────────┐
                    │ device-rpc  config-rpc  ...  │
                    └──────────┬──────────────────┘
                               │ zRPC (gRPC)
                    ┌──────────┴──────────────────┐
                    │        ACS Engine            │
                    │  ┌──────────────────────┐    │
                    │  │ zRPC Server (:50050)  │    │  ← 管理面调用入口
                    │  │ QueueCommand          │    │
                    │  │ SendConnectionRequest  │    │
                    │  │ GetStatus             │    │
                    │  └──────────────────────┘    │
                    │  ┌──────────────────────┐    │
                    │  │ zRPC Client           │    │  ← ACS 调用管理面
                    │  │ → device-rpc          │    │
                    │  │ → config-rpc          │    │
                    │  └──────────────────────┘    │
                    │  ┌──────────────────────┐    │
                    │  │ HTTP Server           │    │
                    │  │ :7547 (HTTP)          │    │  ← CPE 设备连接
                    │  │ :7548 (HTTPS/TLS)     │    │
                    │  │ SOAP/XML 协议栈       │    │
                    │  └──────────────────────┘    │
                    │  ┌──────────────────────┐    │
                    │  │ NATS EventBus         │    │  ← 异步事件发布
                    │  │ device.inform.*       │    │
                    │  │ alarm.raised          │    │
                    │  └──────────────────────┘    │
                    └──────────────────────────────┘
```

### 与纯 go-zero 服务的区别

| 维度 | 纯 go-zero 服务 | ACS 引擎 |
|------|----------------|---------|
| HTTP 框架 | go-zero rest 模块 | net/http stdlib |
| 序列化格式 | JSON / Protobuf | SOAP/XML |
| 代码生成 | goctl 生成 handler/types | 全部手写 |
| 中间件 | go-zero 内置链 | 自定义管线 |
| 服务发现 | etcd 自动注册 | etcd 手动注册 |
| RPC 接口 | zRPC 服务端 | zRPC 双角色（服务端 + 客户端） |
| 限流 | go-zero 自适应限流 | 自定义设备级限流 |

---

## 2. zRPC 接口定义

### 2.1 ACS 控制接口（供管理面调用）

```protobuf
// api/proto/acs_control.proto
syntax = "proto3";

package acs;
option go_package = "omcgo/service/acs/zrpc/pb";

// ACS 控制服务 —— 管理面通过此接口操作 ACS 引擎
service AcsControl {
    // 向设备命令队列添加 RPC 命令
    rpc QueueCommand(QueueCommandReq) returns (QueueCommandResp);

    // 向设备发送 Connection Request（主动拉取设备连接）
    rpc SendConnectionRequest(ConnReqMsg) returns (ConnReqResp);

    // 查询指定设备的当前会话状态
    rpc GetSessionStatus(SessionStatusReq) returns (SessionStatusResp);

    // 获取 ACS 引擎全局状态（活跃会话数、吞吐量等）
    rpc GetAcsStatus(GetAcsStatusReq) returns (AcsStatusResp);

    // 流式订阅设备事件（用于实时监控面板）
    rpc StreamDeviceEvents(StreamEventsReq) returns (stream DeviceEvent);
}

// ========== 命令队列 ==========

message QueueCommandReq {
    string device_serial = 1;     // 设备序列号
    string command_type = 2;      // get_parameters / set_parameters / download / upload / reboot / factory_reset
    int32 priority = 3;           // 优先级（0=最高，100=最低）
    bytes payload = 4;            // JSON 编码的命令参数
    string command_key = 5;       // TR069 CommandKey（用于 TransferComplete 关联）
}

message QueueCommandResp {
    string command_id = 1;        // 命令 ID
    bool queued = 2;              // 是否入队成功
    string error_message = 3;     // 失败原因
}

// ========== Connection Request ==========

message ConnReqMsg {
    string device_serial = 1;
    string connection_request_url = 2;  // 设备 CR URL
    string username = 3;                // CR 认证用户名
    string password = 4;                // CR 认证密码
}

message ConnReqResp {
    bool success = 1;
    string error_message = 2;
}

// ========== 会话状态 ==========

message SessionStatusReq {
    string device_serial = 1;
}

message SessionStatusResp {
    bool has_active_session = 1;
    string state = 2;             // IDLE / INFORM_RECEIVED / PROCESSING / RPC_PENDING / RPC_RESPONSE / COMPLETE
    int64 started_at = 3;         // Unix timestamp
    int32 rpc_count = 4;          // 本会话已执行的 RPC 数量
}

// ========== ACS 全局状态 ==========

message GetAcsStatusReq {}

message AcsStatusResp {
    int32 active_sessions = 1;    // 当前活跃会话数
    int32 max_sessions = 2;       // 最大并发会话数
    int64 total_informs = 3;      // 累计 Inform 数
    int64 total_rpcs = 4;         // 累计 RPC 执行数
    double avg_session_ms = 5;    // 平均会话时长（毫秒）
    int32 queued_commands = 6;    // 待执行命令总数
}

// ========== 设备事件流 ==========

message StreamEventsReq {
    repeated string event_types = 1;  // 过滤事件类型（空=全部）
}

message DeviceEvent {
    string device_serial = 1;
    string event_type = 2;        // bootstrap / periodic / value_change / alarm / transfer_complete
    int64 timestamp = 3;
    bytes data = 4;               // JSON 编码的事件数据
}
```

### 2.2 ACS 作为 zRPC 客户端调用的服务

```
ACS Engine → device-rpc:
  - RegisterDevice()      // Bootstrap Inform 时注册新设备
  - UpdateHeartbeat()     // Periodic Inform 时更新心跳
  - TransitionStatus()    // 设备状态变更

ACS Engine → config-rpc:
  - ResolveDataModel()    // 解析设备对应的数据模型（三级回退）
  - GetConfigTemplate()   // 获取开站配置模板
```

---

## 3. 会话状态机

### 3.1 状态定义

```
                    ┌──────┐
                    │ IDLE │ ◄────── 会话超时/异常
                    └──┬───┘
                       │ CPE 发送 Inform
                       ▼
              ┌──────────────────┐
              │ INFORM_RECEIVED  │
              └──────┬───────────┘
                     │ 解析 Inform、注册/更新设备
                     ▼
              ┌──────────────────┐
              │   PROCESSING     │ ◄──── 检查命令队列
              └──────┬───────────┘
                     │
           ┌─────────┴──────────┐
           │ 有待执行命令？      │
           ├─── 是 ─────┐       │
           │             ▼       │
           │    ┌─────────────┐  │
           │    │ RPC_PENDING │  │ ← 发送 RPC 请求给 CPE
           │    └──────┬──────┘  │
           │           │         │
           │           ▼         │
           │    ┌─────────────┐  │
           │    │ RPC_RESPONSE│  │ ← 收到 CPE 响应
           │    └──────┬──────┘  │
           │           │         │
           │           └─────────┘ (循环: 继续下一个命令)
           │
           ├─── 否 ────┐
           │            ▼
           │     ┌───────────┐
           │     │ COMPLETE  │ → 发送空 HTTP 响应，会话结束
           │     └───────────┘
           └────────────────────
```

### 3.2 会话存储（Redis）

```go
// Redis Hash: acs:session:{device_serial}
type SessionState struct {
    SessionID    string    `redis:"session_id"`
    DeviceSerial string    `redis:"device_serial"`
    State        string    `redis:"state"`
    CreatedAt    int64     `redis:"created_at"`
    UpdatedAt    int64     `redis:"updated_at"`
    InformData   []byte    `redis:"inform_data"`   // 序列化的 Inform 解析结果
    CurrentRPC   string    `redis:"current_rpc"`   // 当前执行的 RPC 类型
    RPCCount     int       `redis:"rpc_count"`     // 已执行 RPC 数量
}
// TTL: 5 分钟（超时自动清理）
```

### 3.3 命令队列（Redis Sorted Set）

```
Key:    acs:cmdq:{device_serial}
Member: JSON 编码的命令对象
Score:  优先级（越小越优先）+ 时间戳小数部分（同优先级 FIFO）

示例:
  ZADD acs:cmdq:SN123456 0.1709654321 '{"type":"get_parameters","params":["Device.DeviceInfo."]}'
  ZADD acs:cmdq:SN123456 50.1709654322 '{"type":"set_parameters","params":[{"name":"...","value":"..."}]}'
  ZADD acs:cmdq:SN123456 100.1709654323 '{"type":"reboot"}'
```

---

## 4. SOAP/XML 处理方案

### 4.1 发送方向（ACS → CPE）：text/template

```go
// 预编译的 SOAP 模板
var getParameterValuesTemplate = template.Must(template.New("gpv").Parse(`
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">{{.ID}}</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterValues>
      <ParameterNames>
        {{range .ParameterNames}}
        <string>{{.}}</string>
        {{end}}
      </ParameterNames>
    </cwmp:GetParameterValues>
  </soap:Body>
</soap:Envelope>
`))
```

### 4.2 接收方向（CPE → ACS）：xml.Decoder 流式解析

```go
func parseInform(r io.Reader) (*InformMessage, error) {
    decoder := xml.NewDecoder(r)
    var inform InformMessage

    for {
        token, err := decoder.Token()
        if err == io.EOF {
            break
        }
        if err != nil {
            return nil, fmt.Errorf("parse inform XML: %w", err)
        }

        switch se := token.(type) {
        case xml.StartElement:
            switch se.Name.Local {
            case "Inform":
                if err := decoder.DecodeElement(&inform, &se); err != nil {
                    return nil, fmt.Errorf("decode inform element: %w", err)
                }
            }
        }
    }
    return &inform, nil
}
```

### 4.3 Content-Type 处理

```
CPE → ACS: Content-Type: text/xml; charset=utf-8
ACS → CPE: Content-Type: text/xml; charset=utf-8

空响应（会话结束）: HTTP 204 No Content
```

---

## 5. 认证

### 5.1 CPE 认证（HTTP Digest/Basic）

ACS 验证 CPE 身份，与 go-zero JWT 完全独立。

```go
type DeviceAuthenticator struct {
    // 从数据库或配置获取设备凭证
    credentialStore CredentialStore
}

func (a *DeviceAuthenticator) Authenticate(r *http.Request) (*DeviceIdentity, error) {
    // 1. 尝试 HTTP Digest 认证
    if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Digest ") {
        return a.verifyDigest(r)
    }
    // 2. 回退到 HTTP Basic 认证
    if username, password, ok := r.BasicAuth(); ok {
        return a.verifyBasic(username, password)
    }
    // 3. 返回 401 WWW-Authenticate 挑战
    return nil, ErrAuthRequired
}
```

### 5.2 Connection Request 认证

ACS 主动连接 CPE 时的反向认证。

```go
func (c *ConnReqSender) Send(ctx context.Context, req *ConnReqMsg) error {
    httpReq, _ := http.NewRequestWithContext(ctx, "GET", req.ConnectionRequestURL, nil)
    httpReq.SetBasicAuth(req.Username, req.Password)

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return fmt.Errorf("connection request to %s: %w", req.DeviceSerial, err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
        return fmt.Errorf("connection request failed: status %d", resp.StatusCode)
    }
    return nil
}
```

---

## 6. 限流与准入控制

### 6.1 设备级限流

```go
type DeviceRateLimiter struct {
    limiters sync.Map // map[string]*rate.Limiter
    rate     rate.Limit
    burst    int
}

func (d *DeviceRateLimiter) Allow(deviceSerial string) bool {
    limiter, _ := d.limiters.LoadOrStore(deviceSerial,
        rate.NewLimiter(d.rate, d.burst))
    return limiter.(*rate.Limiter).Allow()
}
```

配置：每设备每分钟最多 5 次 Inform（防洪泛）。

### 6.2 全局准入控制

```go
type AdmissionController struct {
    current  atomic.Int32
    maxConns int32
}

func (a *AdmissionController) TryAcquire() bool {
    for {
        cur := a.current.Load()
        if cur >= a.maxConns {
            return false // 返回 503 Service Unavailable
        }
        if a.current.CompareAndSwap(cur, cur+1) {
            return true
        }
    }
}

func (a *AdmissionController) Release() {
    a.current.Add(-1)
}
```

默认最大并发会话数：1000（100K 设备场景）。

---

## 7. 请求处理管线

```go
func (s *ACSServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 1. 设备级限流
    deviceSerial := extractDeviceSerial(r)
    if !s.rateLimiter.Allow(deviceSerial) {
        http.Error(w, "Rate Limited", http.StatusServiceUnavailable)
        return
    }

    // 2. 全局准入控制
    if !s.admission.TryAcquire() {
        http.Error(w, "Server Overloaded", http.StatusServiceUnavailable)
        return
    }
    defer s.admission.Release()

    // 3. CPE 认证
    device, err := s.authenticator.Authenticate(r)
    if err != nil {
        w.Header().Set("WWW-Authenticate", s.authenticator.Challenge())
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // 4. 解析 SOAP/XML 请求体
    soapMsg, err := s.soapParser.Parse(r.Body)
    if err != nil {
        s.logger.Error("SOAP parse error", zap.Error(err))
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }

    // 5. 分发处理（Inform / RPC Response / Empty）
    resp, err := s.dispatcher.Dispatch(r.Context(), device, soapMsg)
    if err != nil {
        s.logger.Error("dispatch error", zap.Error(err))
        http.Error(w, "Internal Error", http.StatusInternalServerError)
        return
    }

    // 6. 返回 SOAP 响应
    if resp == nil {
        w.WriteHeader(http.StatusNoContent) // 会话结束
        return
    }
    w.Header().Set("Content-Type", "text/xml; charset=utf-8")
    w.Write(resp)
}
```

---

## 8. 与 go-zero 生态的集成

### 8.1 etcd 服务注册

ACS 引擎手动注册到 etcd，使管理面可通过 zRPC 发现 ACS。

```go
import "github.com/zeromicro/go-zero/zrpc"

func main() {
    // 1. 启动 zRPC 服务端（供管理面调用）
    zrpcServer := zrpc.MustNewServer(zrpc.RpcServerConf{
        ListenOn: ":50050",
        Etcd: discov.EtcdConf{
            Hosts: []string{"etcd1:2379", "etcd2:2379", "etcd3:2379"},
            Key:   "acs.rpc",
        },
    }, func(s *grpc.Server) {
        pb.RegisterAcsControlServer(s, acsControlServer)
    })

    // 2. 启动 HTTP 服务器（SOAP/XML）
    httpServer := &http.Server{
        Addr:    ":7547",
        Handler: acsHandler,
    }

    // 3. 创建 zRPC 客户端（调用 device-rpc、config-rpc）
    deviceRpc := zrpc.MustNewClient(zrpc.RpcClientConf{
        Etcd: discov.EtcdConf{
            Hosts: []string{"etcd1:2379", "etcd2:2379", "etcd3:2379"},
            Key:   "device.rpc",
        },
    })

    // 4. 使用 go-zero ServiceGroup 管理生命周期
    group := service.NewServiceGroup()
    group.Add(zrpcServer)
    group.Add(httpServerWrapper)
    group.Start()
}
```

### 8.2 Prometheus 指标

```go
// ACS 自定义指标（与 go-zero 内置指标共存）
var (
    acsActiveSessions = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "acs_active_sessions",
        Help: "Number of active TR069 sessions",
    })
    acsInformTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "acs_inform_total",
        Help: "Total number of Inform messages received",
    }, []string{"event_type", "carrier"})
    acsRPCDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "acs_rpc_duration_seconds",
        Help:    "Duration of TR069 RPC executions",
        Buckets: prometheus.DefBuckets,
    }, []string{"rpc_method"})
    acsSessionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
        Name:    "acs_session_duration_seconds",
        Help:    "Duration of TR069 sessions",
        Buckets: []float64{0.1, 0.25, 0.5, 1, 2, 5},
    })
)
```

---

## 9. 容量规划

| 规模 | 设备数 | 峰值 Inform/s | 并发会话 | ACS 实例数 |
|------|--------|:------------:|:-------:|:---------:|
| 小型 | 10K | 33 | 50 | 1 |
| 中型 | 100K | 333 | 500 | 2 |
| 大型 | 1M | 3333 | 5000 | 10-20 |

- 每次 Inform 会话：3-5 次 HTTP 往返，持续 500ms-2s
- 单 ACS 实例承载：~200 并发会话（CPU 密集型 XML 解析）
- 水平扩展方式：负载均衡器按 device_serial 一致性 hash 分发

### Redis 内存估算

```
每设备会话状态: ~1KB
10万设备 × 0.5% 并发率 = 500 会话 × 1KB = 500KB
命令队列平均 2 个/设备 × 10万 × 200B = 40MB
心跳 Key: 10万 × 100B = 10MB
总计: ~50MB（Redis 轻松承载）
```
