# ACS RPC 交互流程：现状分析报告 & 开发设计方案

> 来源：基于 `omcgo/doc/acs-rpc-interaction-flows.md` 中整理的 32 个 TR069 交互流程
> 对比对象：`omcgo/` 当前代码实现
> 版本：1.0.0
> 日期：2026-03-10

---

## 一、现状总结

### 已实现的基础设施（完成度约 70%）

| 组件 | 文件 | 状态 |
|------|------|------|
| HTTP Server + POST /acs | `internal/acs/server.go` | ✅ |
| SOAP 编码 - 9 个 RPC 模板 | `pkg/soap/templates.go` | ✅ |
| SOAP 解码 - Inform/TransferComplete/GetParameterValues/SetParameterValues/Download | `pkg/soap/decoder.go` | ✅ 部分 |
| RPC Method 常量 + DetectRPCMethod | `pkg/soap/envelope.go` | ✅ 部分 |
| TR069 类型定义 | `pkg/tr069/types.go` | ✅ 部分 |
| TR069 事件码常量 (0-8) | `pkg/tr069/events.go` | ✅ 部分 |
| Session 状态机 (6 状态) | `internal/acs/session.go` | ✅ |
| Redis Session Store | `internal/acs/session_store.go` | ✅ |
| Redis 命令队列 (Sorted Set) | `internal/acs/cmdqueue/queue.go` | ✅ |
| RPC Dispatcher + 9 Handler | `internal/acs/rpc/dispatcher.go` | ✅ 部分 |
| Connection Request 客户端 | `internal/acs/connreq/client.go` | ✅ 基础 |
| CPE 认证 (Digest/Basic/Noop) | `internal/acs/auth/authenticator.go` | ✅ |
| 设备级限流 + 全局准入控制 | `internal/acs/ratelimit.go`, `admission.go` | ✅ |
| Prometheus 指标 | `internal/acs/metrics.go` | ✅ |
| EventBus 发布 (Inform/RPC/TransferComplete) | `internal/acs/handler.go` | ✅ 部分 |
| 设备管理 (注册/心跳/参数) | `internal/omcr/device/` | ✅ |
| PM 采集/KPI 计算 | `internal/pm/` | ✅ |
| 告警引擎 (去重/生命周期) | `internal/alarm/` | ✅ |
| MR 采集/解析 | `internal/mr/` | ✅ |
| 固件管理 (版本/升级任务) | `internal/omcr/software/` | ✅ |
| 自动开站 | `internal/provision/` | ✅ |
| 三运营商适配 | `internal/carrier/` | ✅ |

---

## 二、逐流程对比分析（32 个流程）

### 缺陷分类标记

- 🔴 **Critical** — 协议不合规或功能完全缺失
- 🟡 **Major** — 功能缺失但有 workaround
- 🟢 **OK** — 已实现或可通过现有机制覆盖

### 配置管理 (A.1-A.8)

| 流程 | 核心 RPC | 模板 | 解码器 | Handler | 事件 | 状态 |
|------|---------|------|--------|---------|------|------|
| A.1 参数名称查询 | GetParameterNames | ✅ | ❌ GetParameterNamesResponse | ✅ Handler | 🟡 无 GetParameterNamesResponse 事件 | 🟡 |
| A.2 参数值查询 | GetParameterValues | ✅ | ✅ | ✅ | ✅ | 🟢 |
| A.3 参数值修改 | SetParameterValues | ✅ | ✅ | ✅ | ✅ | 🟢 |
| A.4 对象新增 | AddObject | ✅ | ❌ AddObjectResponse | ✅ Handler | 🟡 无事件 | 🟡 |
| A.5 对象删除 | DeleteObject | ✅ | ❌ DeleteObjectResponse | ✅ Handler | 🟡 无事件 | 🟡 |
| A.6 参数变更上报 | Inform(VALUE_CHANGE) | — | ✅ Inform | ✅ | ✅ | 🟢 |
| A.7 参数属性查询 | **GetParameterAttributes** | ❌ | ❌ | ❌ | ❌ | 🔴 |
| A.8 参数属性修改 | **SetParameterAttributes** | ❌ | ❌ | ❌ | ❌ | 🔴 |

### 性能管理 (A.9-A.11, A.32)

| 流程 | 核心 RPC | 状态 | 缺陷 |
|------|---------|------|------|
| A.9 性能任务定制 | SetParameterValues | 🟢 | 通过 SetParameterValues 实现 |
| A.10 文件上传(PM) | HTTP PUT/POST | 🟢 | MinIO + 文件服务器 |
| A.11 文件上传结果通知 | Inform + **AutonomousTransferComplete** | 🔴 | AutonomousTransferComplete 完全缺失 |
| A.32 性能数据补采 | SetParameterValues + **AutonomousTransferComplete** | 🔴 | AutonomousTransferComplete 缺失 + 补采编排逻辑无 |

### 故障管理 (A.12-A.13)

| 流程 | 核心 RPC | 状态 | 缺陷 |
|------|---------|------|------|
| A.12 实时告警上报 | Inform(ALARM) | 🟡 | handler 未识别 CMCC 告警事件码 |
| A.13 告警同步 | GetParameterValues | 🟢 | 通过 GetParameterValues 查询告警参数 |

### 维护管理 (A.14-A.29)

| 流程 | 核心 RPC | 状态 | 缺陷 |
|------|---------|------|------|
| A.14 设备在线通知 | Inform(PERIODIC) | 🟢 | — |
| A.15 设备重启 | Reboot | 🟡 | ❌ RebootResponse 解码器 |
| A.16 重启结果通知 | Inform(M Reboot) | 🟡 | 缺 `M Reboot` 事件码识别 |
| A.17 软件升级 | GetParameterValues+Download+TransferComplete | 🟡 | 多步编排缺 UploadResponse 解码 |
| A.18 软件下载 | HTTP GET | 🟢 | 非 ACS 流程 |
| A.19 软件下载完成通知 | Inform(7+M Download) | 🟡 | 缺 `M Download` 事件码 |
| A.20 单元升级结果通知 | Inform(VALUE_CHANGE) | 🟢 | — |
| A.21 升级完成通知 | Inform(VALUE_CHANGE) | 🟢 | — |
| A.22 查询升级进度 | GetParameterValues | 🟢 | — |
| A.23 恢复出厂设置 | FactoryReset | 🟡 | ❌ FactoryResetResponse 解码器 |
| A.24 日志任务定制 | SetParameterValues | 🟢 | — |
| A.25 文件下载 | HTTP GET | 🟢 | 非 ACS 流程 |
| A.26 文件下载结果通知 | Inform+TransferComplete | 🟢 | TransferComplete 已支持 |
| A.27 日志文件获取 | Upload | 🟡 | ❌ UploadResponse 解码器 |
| A.28 设备信息文件上传 | Upload+TransferComplete | 🟡 | ❌ UploadResponse 解码器 |
| A.29 设备信息文件下载 | Download+TransferComplete | 🟢 | Download/TransferComplete 已支持 |

### MR管理 (A.30-A.31)

| 流程 | 核心 RPC | 状态 | 缺陷 |
|------|---------|------|------|
| A.30 MR任务定制 | SetParameterValues | 🟢 | 通过 SetParameterValues 实现 |
| A.31 MR文件上传结果通知 | Inform + **AutonomousTransferComplete** | 🔴 | AutonomousTransferComplete 完全缺失 |

### 统计

| 状态 | 数量 | 占比 |
|------|------|------|
| 🟢 OK | 17 | 53% |
| 🟡 Major | 10 | 31% |
| 🔴 Critical | 5 | 16% |

---

## 三、7 大缺陷类别详解

### 缺陷1：TR069 会话流程不合规 🔴

**文件**：`internal/acs/handler.go:145-161`

**问题**：`handleInform()` 在发现待执行命令时，直接将 RPC 请求作为 Inform 的 HTTP 响应发送，跳过了 InformResponse。

**规范要求**（附录A 所有 OMC 主动流程）：
```
CPE → Inform        → ACS 必须先回 InformResponse
CPE → Empty POST    → ACS 才能发 RPC Request
CPE → RPC Response  → ACS 发 Empty Response 或下一个 RPC
```

**当前行为**：
```
CPE → Inform → ACS 直接发 RPC Request (跳过 InformResponse)
```

**handleEmpty() 缺陷**：`handler.go:251-257` 仅返回 204，不检查命令队列。

**影响**：A.1-A.5, A.7-A.8, A.13, A.15, A.17, A.22-A.23, A.27-A.29, A.32 全部流程。

---

### 缺陷2：AutonomousTransferComplete 完全缺失 🔴

**缺失内容**：
- `pkg/soap/envelope.go` — 无 `MethodAutonomousTransferComplete` 常量
- `pkg/tr069/types.go` — 无 `AutonomousTransferComplete` 结构体
- `pkg/soap/decoder.go` — 无 `DecodeAutonomousTransferComplete()`
- `pkg/soap/templates.go` — 无 `AutonomousTransferCompleteResponse` 模板
- `internal/acs/handler.go` — 无 `handleAutonomousTransferComplete()`
- `DetectRPCMethod()` 不检测 AutonomousTransferComplete

**影响**：A.11（PM文件上传结果通知）、A.31（MR文件上传结果通知）、A.32（PM补采结果通知）

---

### 缺陷3：GetParameterAttributes / SetParameterAttributes 完全缺失 🔴

**缺失内容**：
- `pkg/tr069/types.go` — 无 GetParameterAttributes/SetParameterAttributes 请求/响应类型
- `pkg/soap/templates.go` — 无 GetParameterAttributes/SetParameterAttributes SOAP 模板
- `pkg/soap/decoder.go` — 无 GetParameterAttributes/SetParameterAttributes Response 解码器
- `internal/acs/rpc/dispatcher.go` — 未注册 GetParameterAttributes/SetParameterAttributes Handler

**影响**：A.7（参数属性查询）、A.8（参数属性修改）—— 这是配置管理中设置通知策略的关键 RPC。

---

### 缺陷4：多个 RPC Response 解码器缺失 🟡

**已有解码器**（5个）：Inform, TransferComplete, GetParameterValuesResponse, SetParameterValuesResponse, DownloadResponse

**缺失解码器**（7个）：

| 解码器 | 复杂度 | 涉及流程 |
|--------|--------|---------|
| `DecodeGetParameterNamesResponse` | 中 | A.1 |
| `DecodeAddObjectResponse` | 低 | A.4 |
| `DecodeDeleteObjectResponse` | 低 | A.5 |
| `DecodeUploadResponse` | 低 | A.27, A.28 |
| `DecodeRebootResponse` | 极低(空) | A.15 |
| `DecodeFactoryResetResponse` | 极低(空) | A.23 |
| `DecodeAutonomousTransferComplete` | 中 | A.11, A.31, A.32 |

---

### 缺陷5：事件码与事件发布不完整 🟡

**events.go 缺失的事件码**：
```go
// 中国移动扩展事件码
EventMReboot        = "M Reboot"
EventMDownload      = "M Download"
EventMUpload        = "M Upload"
EventAddObject      = "103 ADD OBJECT"      // CMCC 对象新增通知
EventDeleteObject   = "104 DELETE OBJECT"    // CMCC 对象删除通知
```

**handler.go publishInformEvents 缺陷**：
- 不识别告警事件码（如 `X_CMCC_ALARM`）→ 告警 Inform 被当作 PERIODIC 处理
- 不识别 `M Reboot`/`M Download` 厂商事件码
- 不识别 `103 ADD OBJECT`/`104 DELETE OBJECT`

**handler.go publishRPCResponseEvent 缺陷**：
- 仅处理 GetParameterValuesResponse/SetParameterValuesResponse/DownloadResponse 三种
- 缺失：UploadResponse, GetParameterNamesResponse, AddObjectResponse, DeleteObjectResponse, RebootResponse, FactoryResetResponse, GetParameterAttributesResponse, SetParameterAttributesResponse

---

### 缺陷6：设备 SN 会话关联机制不标准 🟡

**文件**：`handler.go:181`

**问题**：`handleRPCResponse()` 通过 `r.Header.Get("X-Device-SN")` 获取设备 SN，但这不是 TR069 标准 header，真实设备不会发送。

**正确做法**：
- 方案A：通过 HTTP 连接（同一 TCP 连接的 Inform 中已解析 SN）关联 session
- 方案B：通过 CWMP ID 关联（Inform 和后续 RPC 使用同一 CWMP ID）
- 方案C：在 ACS 侧为每个 HTTP 连接维护一个 context（推荐）

---

### 缺陷7：ConnReq 客户端能力不足 🟡

**文件**：`internal/acs/connreq/client.go`

**缺失**：
- Connection Request 不支持 HTTP Digest 认证（CPE 通常要求）
- 无指数退避重试（规范要求最多 3 次）
- 无 TLS 支持

---

## 四、开发设计方案

### Phase 1: TR069 协议合规（会话流程修复）⭐ 最高优先级

**目标**：修复 handler.go 使会话流程符合 TR069/附录A 规范

#### 1.1 重构 handleInform() — `internal/acs/handler.go`

```
修改前：Inform → 直接发 RPC（跳过 InformResponse）
修改后：Inform → 始终发 InformResponse → 在后续 Empty POST 时发 RPC
```

**改动**：
- `handleInform()` 只做：解析→认证→限流→创建会话→发布事件→**始终发送 InformResponse**
- 将命令队列检查从 handleInform 移到 handleEmpty

#### 1.2 重构 handleEmpty() — `internal/acs/handler.go`

```go
// 改动前：直接 204
// 改动后：
func handleEmpty():
  1. 获取当前会话（通过连接 context）
  2. if session.State == StateInformReceived:
       session.State = StateProcessing
  3. 检查命令队列 Peek → Pop
  4. if cmd != nil:
       BuildRequest → 发送 RPC → session.State = StateRPCPending
  5. else:
       发送空响应 → session.State = StateComplete
```

#### 1.3 实现连接级 Session 关联

**方案**：HTTP 连接 Context 注入

```go
// 新增 middleware 或在 handleInform 中：
type connKey struct{}

func (h *Handler) handleInform(...):
  ...
  // 将 deviceSN 绑定到 HTTP 连接
  ctx := context.WithValue(r.Context(), connKey{}, deviceSN)
  // 存储到 per-connection map（sync.Map[remoteAddr]deviceSN）
  h.connSessions.Store(r.RemoteAddr, deviceSN)
```

**文件改动**：
- `internal/acs/handler.go` — 核心重构
- `internal/acs/session.go` — 可能需要添加 connSessions map

---

### Phase 2: 补全 AutonomousTransferComplete 🔴

#### 2.1 新增 TR069 类型 — `pkg/tr069/types.go`

```go
type AutonomousTransferComplete struct {
    AnnounceURL      string    `xml:"AnnounceURL"`
    TransferURL      string    `xml:"TransferURL"`
    IsDownload       bool      `xml:"IsDownload"`
    FileType         string    `xml:"FileType"`
    FileSize         int64     `xml:"FileSize"`
    TargetFileName   string    `xml:"TargetFileName"`
    FaultStruct      *Fault    `xml:"FaultStruct"`
    StartTime        time.Time `xml:"StartTime"`
    CompleteTime     time.Time `xml:"CompleteTime"`
}
```

#### 2.2 新增 SOAP 常量 — `pkg/soap/envelope.go`

```go
MethodAutonomousTransferComplete     RPCMethod = "AutonomousTransferComplete"
MethodAutonomousTransferCompleteResp RPCMethod = "AutonomousTransferCompleteResponse"
```

同步更新 `DetectRPCMethod()` 检测列表。

#### 2.3 新增解码器 — `pkg/soap/decoder.go`

```go
func DecodeAutonomousTransferComplete(r io.Reader) (*tr069.AutonomousTransferComplete, string, error)
```

#### 2.4 新增响应模板 — `pkg/soap/templates.go`

```go
var AutonomousTransferCompleteRespTmpl *template.Template

const autonomousTransferCompleteResponseXML = soapEnvelopeOpen + `
    <cwmp:AutonomousTransferCompleteResponse/>` + soapEnvelopeClose
```

#### 2.5 新增处理函数 — `internal/acs/handler.go`

```go
func (h *Handler) handleAutonomousTransferComplete(w, r, body):
  1. 解码 AutonomousTransferComplete 消息
  2. 发布事件 (device.inform.autonomous_transfer_complete)
  3. 发送 AutonomousTransferCompleteResponse
```

同步更新 `ServeHTTP()` 的 switch 分支。

#### 2.6 新增事件主题 — `internal/common/event/subjects.go`

```go
SubjectDeviceAutonomousTransferComplete = "device.inform.autonomous_transfer_complete"
```

---

### Phase 3: 补全 GetParameterAttributes/SetParameterAttributes 🔴

#### 3.1 新增 TR069 类型 — `pkg/tr069/types.go`

```go
type SetParameterAttributeStruct struct {
    Name                string `xml:"Name"`
    NotificationChange  bool   `xml:"NotificationChange"`
    Notification        int    `xml:"Notification"` // 0=off, 1=passive, 2=active
    AccessListChange    bool   `xml:"AccessListChange"`
    AccessList          []string `xml:"AccessList>string"`
}

type ParameterAttributeStruct struct {
    Name         string   `xml:"Name"`
    Notification int      `xml:"Notification"`
    AccessList   []string `xml:"AccessList>string"`
}

type GetParameterAttributesResponse struct {
    ParameterList []ParameterAttributeStruct `xml:"ParameterList>ParameterAttributeStruct"`
}

type SetParameterAttributesResponse struct{} // 空响应
```

#### 3.2 新增 SOAP 模板 — `pkg/soap/templates.go`

```go
var GetParameterAttributesTmpl *template.Template
var SetParameterAttributesTmpl *template.Template

// GetParameterAttributes 请求模板
// SetParameterAttributes 请求模板（含 NotificationChange/Notification/AccessListChange/AccessList）
```

新增数据结构：

```go
type GetParameterAttributesData struct {
    ID     string
    Params []ParameterNameData
}

type SetParameterAttributesData struct {
    ID     string
    Params []SetParameterAttributeData
}

type SetParameterAttributeData struct {
    Name                string
    NotificationChange  bool
    Notification        int
    AccessListChange    bool
    AccessList          []string
}
```

#### 3.3 新增解码器 — `pkg/soap/decoder.go`

```go
func DecodeGetParameterAttributesResponse(r io.Reader) ([]tr069.ParameterAttributeStruct, string, error)
func DecodeSetParameterAttributesResponse(r io.Reader) (string, error)
```

#### 3.4 新增 RPC Handler — `internal/acs/rpc/dispatcher.go`

```go
type GetParameterAttributesHandler struct{}
type SetParameterAttributesHandler struct{}
```

注册到 `NewDispatcher()`。

#### 3.5 新增 SOAP 常量 — `pkg/soap/envelope.go`

```go
MethodGetParameterAttributes     RPCMethod = "GetParameterAttributes"
MethodGetParameterAttributesResp RPCMethod = "GetParameterAttributesResponse"
MethodSetParameterAttributes     RPCMethod = "SetParameterAttributes"
MethodSetParameterAttributesResp RPCMethod = "SetParameterAttributesResponse"
```

同步更新 `DetectRPCMethod()` 和 `handler.go` switch。

---

### Phase 4: 补全 Response 解码器 🟡

所有缺失的解码器均在 `pkg/soap/decoder.go` 中实现：

| 解码器 | 返回类型 | 工作量 |
|--------|---------|--------|
| `DecodeGetParameterNamesResponse` | `[]ParameterInfoStruct` | 参照 GetParameterValues 解码器改造 |
| `DecodeAddObjectResponse` | `(instanceNumber int, status int)` | 简单 |
| `DecodeDeleteObjectResponse` | `(status int)` | 简单 |
| `DecodeUploadResponse` | `(status int, completeTime)` | 参照 Download 解码器 |
| `DecodeRebootResponse` | 无返回值 | 极简 |
| `DecodeFactoryResetResponse` | 无返回值 | 极简 |

---

### Phase 5: 事件码 & 事件发布完善 🟡

#### 5.1 扩展事件码 — `pkg/tr069/events.go`

```go
// 厂商事件码
EventMReboot      = "M Reboot"
EventMDownload    = "M Download"
EventMUpload      = "M Upload"

// CMCC 扩展事件码
EventAddObject    = "103 ADD OBJECT"
EventDeleteObject = "104 DELETE OBJECT"

// 告警识别辅助函数
func IsAlarm(events []EventStruct) bool
func IsConnectionRequest(events []EventStruct) bool
func IsRebootComplete(events []EventStruct) bool
```

#### 5.2 完善 Inform 事件分发 — `handler.go:publishInformEvents`

```go
switch {
case IsBootstrap:    → device.inform.bootstrap
case IsPeriodic:     → device.inform.periodic
case IsValueChange:  → device.inform.value_change
case IsAlarm:        → device.inform.alarm          // 新增
case IsRebootComplete: → device.inform.reboot_complete // 新增
case HasEvent(TransferComplete): → device.inform.transfer_complete
case HasEvent(ConnectionRequest): → device.inform.connection_request // 新增
default:             → device.inform.periodic
}
```

#### 5.3 完善 RPC Response 事件 — `handler.go:publishRPCResponseEvent`

补全所有 Response 类型的事件发布。新增事件主题：

```go
SubjectCommandUploadResponse       = "command.upload.response"
SubjectCommandGetNamesResponse     = "command.get_names.response"
SubjectCommandAddObjectResponse    = "command.add_object.response"
SubjectCommandDeleteObjectResponse = "command.delete_object.response"
SubjectCommandRebootResponse       = "command.reboot.response"
SubjectCommandFactoryResetResponse = "command.factory_reset.response"
SubjectCommandGetAttrsResponse     = "command.get_attrs.response"
SubjectCommandSetAttrsResponse     = "command.set_attrs.response"
```

---

### Phase 6: ConnReq 客户端增强 🟡

**文件**：`internal/acs/connreq/client.go`

1. **HTTP Digest 认证**：复用 `auth/authenticator.go` 的 Digest 逻辑，在 ConnReq GET 请求中带 Digest 响应
2. **指数退避重试**：最多 3 次，延迟 1s → 2s → 4s
3. **TLS 支持**：支持 HTTPS URL 的 Connection Request

---

## 五、文件改动清单

| Phase | 文件路径 | 操作 | 改动量 |
|-------|---------|------|--------|
| P1 | `internal/acs/handler.go` | 重构 handleInform + handleEmpty + 添加 connSessions | ~150行 |
| P2 | `pkg/tr069/types.go` | 新增 AutonomousTransferComplete 类型 | ~15行 |
| P2 | `pkg/soap/envelope.go` | 新增 AutonomousTransferComplete 常量 + DetectRPCMethod 更新 | ~5行 |
| P2 | `pkg/soap/decoder.go` | 新增 DecodeAutonomousTransferComplete | ~40行 |
| P2 | `pkg/soap/templates.go` | 新增 AutonomousTransferComplete Response 模板 | ~10行 |
| P2 | `internal/acs/handler.go` | 新增 handleAutonomousTransferComplete + switch 分支 | ~30行 |
| P2 | `internal/common/event/subjects.go` | 新增 AutonomousTransferComplete 事件主题 | ~2行 |
| P3 | `pkg/tr069/types.go` | 新增 GetParameterAttributes/SetParameterAttributes 类型 | ~30行 |
| P3 | `pkg/soap/templates.go` | 新增 GetParameterAttributes/SetParameterAttributes 模板 + 数据类型 | ~50行 |
| P3 | `pkg/soap/decoder.go` | 新增 GetParameterAttributes/SetParameterAttributes Response 解码器 | ~80行 |
| P3 | `internal/acs/rpc/dispatcher.go` | 新增 GetParameterAttributes/SetParameterAttributes Handler + 注册 | ~40行 |
| P3 | `pkg/soap/envelope.go` | 新增 GetParameterAttributes/SetParameterAttributes 常量 | ~6行 |
| P4 | `pkg/soap/decoder.go` | 新增 6 个 Response 解码器 | ~150行 |
| P5 | `pkg/tr069/events.go` | 新增事件码 + 辅助函数 | ~30行 |
| P5 | `internal/acs/handler.go` | 完善事件分发逻辑 | ~40行 |
| P5 | `internal/common/event/subjects.go` | 新增 8+ 事件主题 | ~15行 |
| P6 | `internal/acs/connreq/client.go` | Digest 认证 + 重试 + TLS | ~80行 |

**预计总新增/改动**：~750 行

---

## 六、验证方案

### 单元测试
- `pkg/soap/decoder_test.go` — 每个新解码器的 SOAP XML fixture 测试
- `pkg/soap/templates_test.go` — GetParameterAttributes/SetParameterAttributes 模板渲染正确性
- `internal/acs/session_test.go` — 会话状态转移覆盖新流程
- `internal/acs/handler_test.go` — 新增：模拟 Inform→InformResponse→EmptyPOST→RPC 完整流程
- `internal/acs/connreq/client_test.go` — Digest 认证 + 重试逻辑

### 集成测试
- 启动 ACS 进程 + Redis，用 curl 发送 SOAP Inform XML
- 验证 InformResponse 正确返回
- 验证 Empty POST 触发命令队列检查
- 验证 AutonomousTransferComplete 响应正确

### E2E 测试补充
- 在 `scripts/e2e_verify.sh` 中添加 ACS SOAP 协议测试用例
- 构造 SOAP fixture 文件用于 curl 测试

---

## 七、实施顺序

```
Phase 1 (会话流程) ← 最高优先，影响所有 32 个流程
  ↓
Phase 2 (AutonomousTransferComplete) ← 影响 A.11, A.31, A.32（PM/MR 结果通知）
  ↓
Phase 3 (GetParameterAttributes/SetParameterAttributes) ← 影响 A.7, A.8（参数属性管理）
  ↓
Phase 4 (解码器)   ← 逐步补全，按使用频率排序
  ↓
Phase 5 (事件)     ← 完善事件驱动架构
  ↓
Phase 6 (ConnReq)  ← 增强 Connection Request 能力
```
