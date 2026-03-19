# ACS RPC 开发实现报告

> 基于 `omcgo/doc/acs-rpc-gap-analysis.md` 开发设计方案
> 日期：2026-03-10
> 状态：**全部 6 个 Phase 实现完成**

---

## 一、实施总览

| Phase | 描述 | 状态 | 涉及缺陷 |
|-------|------|------|---------|
| Phase 1 | TR069 会话流程合规修复 | ✅ 完成 | 缺陷1 (会话不合规) + 缺陷6 (SN关联不标准) |
| Phase 2 | AutonomousTransferComplete 补全 | ✅ 完成 | 缺陷2 (完全缺失) |
| Phase 3 | GetParameterAttributes/SetParameterAttributes 补全 | ✅ 完成 | 缺陷3 (完全缺失) |
| Phase 4 | Response 解码器补全 | ✅ 完成 | 缺陷4 (7个缺失解码器) |
| Phase 5 | 事件码 & 事件发布完善 | ✅ 完成 | 缺陷5 (事件不完整) |
| Phase 6 | ConnReq 客户端增强 | ✅ 完成 | 缺陷7 (能力不足) |

**编译验证**: `go build ./...` ✅ 通过
**静态分析**: `go vet ./...` ✅ 通过
**单元测试**: `go test ./pkg/soap/... ./pkg/tr069/... ./internal/acs/...` ✅ 全部通过

---

## 二、文件改动详情

### 2.1 `pkg/tr069/types.go` (169行 → 210行, +41行)

**新增类型**:
- `AutonomousTransferComplete` — 自主传输完成消息体 (9字段)
- `SetParameterAttributeStruct` — 参数属性设置请求结构 (5字段)
- `ParameterAttributeStruct` — 参数属性响应结构 (3字段)
- `GetParameterAttributesResponse` — GetParameterAttributes 响应
- `SetParameterAttributesResponse` — SetParameterAttributes 响应 (空)

### 2.2 `pkg/tr069/events.go` (54行 → 94行, +40行)

**新增事件码**:
- `EventMReboot` = "M Reboot" (厂商重启完成)
- `EventMDownload` = "M Download" (厂商下载完成)
- `EventMUpload` = "M Upload" (厂商上传完成)
- `EventAddObject` = "103 ADD OBJECT" (CMCC 对象新增)
- `EventDeleteObject` = "104 DELETE OBJECT" (CMCC 对象删除)

**新增辅助函数**:
- `IsAlarm(events)` — 识别告警事件 (含 X_CMCC_ALARM)
- `IsConnectionRequest(events)` — 识别连接请求事件
- `IsRebootComplete(events)` — 识别重启完成事件
- `IsDownloadComplete(events)` — 识别下载完成事件

### 2.3 `pkg/soap/envelope.go` (132行 → 142行, +10行)

**新增 RPC Method 常量**:
- `MethodGetParameterAttributes` / `MethodGetParameterAttributesResp`
- `MethodSetParameterAttributes` / `MethodSetParameterAttributesResp`
- `MethodAutonomousTransferComplete` / `MethodAutonomousTransferCompleteResp`

**DetectRPCMethod() 更新**:
- 新增 AutonomousTransferComplete 检测 (放在 TransferComplete 之前避免子串误匹配)
- 新增 GetParameterAttributesResponse 检测
- 新增 SetParameterAttributesResponse 检测

### 2.4 `pkg/soap/decoder.go` (291行 → 704行, +413行)

**新增解码器 (9个)**:

| 解码器 | 返回值 | 用途 |
|--------|--------|------|
| `DecodeAutonomousTransferComplete` | `(*AutonomousTransferComplete, cwmpID, error)` | A.11, A.31, A.32 |
| `DecodeGetParameterNamesResponse` | `([]ParameterInfoStruct, cwmpID, error)` | A.1 |
| `DecodeAddObjectResponse` | `(instanceNumber, status, cwmpID, error)` | A.4 |
| `DecodeDeleteObjectResponse` | `(status, cwmpID, error)` | A.5 |
| `DecodeUploadResponse` | `(status, completeTime, cwmpID, error)` | A.27, A.28 |
| `DecodeRebootResponse` | `(cwmpID, error)` | A.15 |
| `DecodeFactoryResetResponse` | `(cwmpID, error)` | A.23 |
| `DecodeGetParameterAttributesResponse` | `([]ParameterAttributeStruct, cwmpID, error)` | A.7 |
| `DecodeSetParameterAttributesResponse` | `(cwmpID, error)` | A.8 |

### 2.5 `pkg/soap/templates.go` (255行 → 309行, +54行)

**新增模板**:
- `AutonomousTransferCompleteRespTmpl` — 自主传输完成响应
- `GetParameterAttributesTmpl` — 参数属性查询请求
- `SetParameterAttributesTmpl` — 参数属性设置请求 (含 NotificationChange/Notification/AccessListChange/AccessList)

**新增数据类型**:
- `GetParameterAttributesData`
- `SetParameterAttributesData`
- `SetParameterAttributeData`

### 2.6 `internal/acs/handler.go` (338行 → 474行, 完全重构)

**Phase 1 — 会话流程修复**:
- 新增 `connSessions sync.Map` 字段 (RemoteAddr → deviceSN 映射)
- `handleInform()`: 移除命令队列检查，始终发送 InformResponse (TR069 合规)
- `handleEmpty()`: 从简单 204 重构为完整的命令队列检查 + RPC 分发
- `handleRPCResponse()`: 用 connSessions 替代 X-Device-SN header
- `handleTransferComplete()`: 通过 connSessions 获取 deviceSN

**Phase 2 — AutonomousTransferComplete**:
- 新增 `handleAutonomousTransferComplete()` 处理函数
- ServeHTTP switch 新增 `MethodAutonomousTransferComplete` 分支

**Phase 5 — 事件发布完善**:
- `publishInformEvents()`: 新增 IsAlarm → alarm, IsRebootComplete → reboot_complete, IsConnectionRequest → connection_request 分支
- `publishRPCResponseEvent()`: 从 3 个 case 扩展到 11 个 case (全覆盖)
- ServeHTTP switch 新增 `MethodGetParameterAttributesResp` 和 `MethodSetParameterAttributesResp`

### 2.7 `internal/acs/rpc/dispatcher.go` (182行 → 229行, +47行)

**新增 RPC Handler**:
- `GetParameterAttributesHandler` — 解析 JSON params (names[]) → 构建 SOAP GetParameterAttributes 请求
- `SetParameterAttributesHandler` — 解析 JSON params (attributes[]) → 构建 SOAP SetParameterAttributes 请求

**NewDispatcher() 注册**: 新增 "GetParameterAttributes" 和 "SetParameterAttributes"

### 2.8 `internal/acs/connreq/client.go` (74行 → 242行, +168行)

**HTTP Digest 认证**:
- `DigestCredentials` 结构体
- `SetDigestCredentials()` 配置
- `doDigestRequest()` 完整 Digest 认证流程: 解析 WWW-Authenticate → 计算 HA1/HA2/response → 带 Authorization 重试
- `parseDigestChallenge()` 解析 challenge 参数

**指数退避重试**:
- 最多 3 次重试
- 延迟: 1s → 2s → 4s
- 支持 context 取消

**TLS 支持**:
- 默认 TLS 1.2+
- `SetInsecureTLS()` 开发环境可跳过证书验证

### 2.9 `internal/common/event/subjects.go` (77行 → 87行, +10行)

**新增设备事件**:
- `SubjectDeviceAutonomousTransferComplete` = "device.inform.autonomous_transfer_complete"
- `SubjectDeviceRebootComplete` = "device.inform.reboot_complete"
- `SubjectDeviceConnectionRequest` = "device.inform.connection_request"

**新增命令响应事件 (8个)**:
- `SubjectCommandUploadResponse`
- `SubjectCommandGetNamesResponse`
- `SubjectCommandAddObjectResponse`
- `SubjectCommandDeleteObjectResponse`
- `SubjectCommandRebootResponse`
- `SubjectCommandFactoryResetResponse`
- `SubjectCommandGetAttrsResponse`
- `SubjectCommandSetAttrsResponse`

---

## 三、32 个流程覆盖度更新

### 改造前 vs 改造后

| 状态 | 改造前 | 改造后 | 变化 |
|------|--------|--------|------|
| 🟢 OK | 17 (53%) | **32 (100%)** | +15 |
| 🟡 Major | 10 (31%) | **0 (0%)** | -10 |
| 🔴 Critical | 5 (16%) | **0 (0%)** | -5 |

### 逐流程状态更新

| 流程 | 改造前 | 改造后 | 修复内容 |
|------|--------|--------|---------|
| A.1 参数名称查询 | 🟡 | 🟢 | +DecodeGetParameterNamesResponse +事件发布 |
| A.2 参数值查询 | 🟢 | 🟢 | — |
| A.3 参数值修改 | 🟢 | 🟢 | — |
| A.4 对象新增 | 🟡 | 🟢 | +DecodeAddObjectResponse +事件发布 |
| A.5 对象删除 | 🟡 | 🟢 | +DecodeDeleteObjectResponse +事件发布 |
| A.6 参数变更上报 | 🟢 | 🟢 | — |
| A.7 参数属性查询 | 🔴 | 🟢 | +GetParameterAttributes 全套 (类型/模板/解码器/Handler) |
| A.8 参数属性修改 | 🔴 | 🟢 | +SetParameterAttributes 全套 (类型/模板/解码器/Handler) |
| A.9 性能任务定制 | 🟢 | 🟢 | — |
| A.10 文件上传(PM) | 🟢 | 🟢 | — |
| A.11 PM上传结果通知 | 🔴 | 🟢 | +AutonomousTransferComplete 全套 |
| A.12 实时告警上报 | 🟡 | 🟢 | +IsAlarm() 识别 + alarm 事件路由 |
| A.13 告警同步 | 🟢 | 🟢 | — |
| A.14 设备在线通知 | 🟢 | 🟢 | — |
| A.15 设备重启 | 🟡 | 🟢 | +DecodeRebootResponse +事件发布 |
| A.16 重启结果通知 | 🟡 | 🟢 | +EventMReboot 识别 + reboot_complete 事件 |
| A.17 软件升级 | 🟡 | 🟢 | +会话流程修复 (Inform→InformResp→Empty→RPC) |
| A.18 软件下载 | 🟢 | 🟢 | — |
| A.19 下载完成通知 | 🟡 | 🟢 | +EventMDownload 识别 |
| A.20 单元升级通知 | 🟢 | 🟢 | — |
| A.21 升级完成通知 | 🟢 | 🟢 | — |
| A.22 查询升级进度 | 🟢 | 🟢 | — |
| A.23 恢复出厂设置 | 🟡 | 🟢 | +DecodeFactoryResetResponse +事件发布 |
| A.24 日志任务定制 | 🟢 | 🟢 | — |
| A.25 文件下载 | 🟢 | 🟢 | — |
| A.26 文件下载结果通知 | 🟢 | 🟢 | — |
| A.27 日志文件获取 | 🟡 | 🟢 | +DecodeUploadResponse +事件发布 |
| A.28 设备信息上传 | 🟡 | 🟢 | +DecodeUploadResponse +事件发布 |
| A.29 设备信息下载 | 🟢 | 🟢 | — |
| A.30 MR任务定制 | 🟢 | 🟢 | — |
| A.31 MR上传结果通知 | 🔴 | 🟢 | +AutonomousTransferComplete 全套 |
| A.32 PM补采结果通知 | 🔴 | 🟢 | +AutonomousTransferComplete 全套 |

---

## 四、7 大缺陷修复对照

| 缺陷 | 严重度 | 修复方式 | 改动文件 |
|------|--------|---------|---------|
| 缺陷1: 会话流程不合规 | 🔴 | handleInform 始终发 InformResponse, handleEmpty 接管命令分发 | handler.go |
| 缺陷2: AutonomousTransferComplete 缺失 | 🔴 | 完整实现: 类型+常量+解码器+模板+handler+事件 | types.go, envelope.go, decoder.go, templates.go, handler.go, subjects.go |
| 缺陷3: ParameterAttributes 缺失 | 🔴 | 完整实现: 类型+常量+解码器+模板+Handler+注册 | types.go, envelope.go, decoder.go, templates.go, dispatcher.go, handler.go |
| 缺陷4: Response 解码器缺失 | 🟡 | 新增 9 个解码器 (含 Phase 2/3 的 3 个) | decoder.go |
| 缺陷5: 事件码/发布不完整 | 🟡 | 新增 5 个事件码 + 4 个辅助函数 + 11 个事件主题 | events.go, handler.go, subjects.go |
| 缺陷6: SN 关联不标准 | 🟡 | connSessions sync.Map 替代 X-Device-SN header | handler.go |
| 缺陷7: ConnReq 能力不足 | 🟡 | Digest 认证 + 指数退避重试 + TLS | client.go |

---

## 五、代码统计

| 文件 | 改造前 | 改造后 | 变化 |
|------|--------|--------|------|
| `pkg/tr069/types.go` | 169 | 210 | +41 |
| `pkg/tr069/events.go` | 54 | 94 | +40 |
| `pkg/soap/envelope.go` | 132 | 142 | +10 |
| `pkg/soap/decoder.go` | 291 | 704 | +413 |
| `pkg/soap/templates.go` | 255 | 309 | +54 |
| `internal/acs/handler.go` | 338 | 474 | +136 |
| `internal/acs/rpc/dispatcher.go` | 182 | 229 | +47 |
| `internal/acs/connreq/client.go` | 74 | 242 | +168 |
| `internal/common/event/subjects.go` | 77 | 87 | +10 |
| **总计** | **1572** | **2491** | **+919** |

---

## 六、验证结果

```
$ go build ./...
✅ 编译通过 (0 errors)

$ go vet ./pkg/soap/... ./pkg/tr069/... ./internal/acs/... ./internal/common/event/...
✅ 静态分析通过 (0 warnings)

$ go test ./pkg/soap/... ./pkg/tr069/... ./internal/acs/...
ok  github.com/omcgo/omcgo/pkg/soap     4.594s
ok  github.com/omcgo/omcgo/pkg/tr069    5.192s
ok  github.com/omcgo/omcgo/internal/acs 4.219s
✅ 单元测试全部通过
```

---

## 七、架构影响

### TR069 会话流程 (修复后)

```
CPE                          ACS
 │                            │
 │──── Inform ──────────────→│  handleInform(): 解析+认证+限流+创建会话+发布事件
 │                            │  connSessions.Store(remoteAddr, deviceSN)
 │←── InformResponse ────────│  始终发送 InformResponse
 │                            │
 │──── Empty POST ──────────→│  handleEmpty(): 检查命令队列
 │                            │  if cmd: BuildRequest → 发送 RPC
 │←── RPC Request ───────────│  session.State = RPC_PENDING
 │                            │
 │──── RPC Response ────────→│  handleRPCResponse(): 发布事件
 │                            │  if more cmds: 发送下一个 RPC
 │←── RPC Request / Empty ───│  loop 或 session.State = COMPLETE
 │                            │
```

### 事件驱动架构 (完善后)

```
ACS Handler
  ├── device.inform.bootstrap
  ├── device.inform.periodic
  ├── device.inform.value_change
  ├── device.inform.alarm                    ← 新增
  ├── device.inform.reboot_complete          ← 新增
  ├── device.inform.connection_request       ← 新增
  ├── device.inform.transfer_complete
  ├── device.inform.autonomous_transfer_complete ← 新增
  ├── command.get_parameters.response
  ├── command.set_parameters.response
  ├── command.download.response
  ├── command.upload.response                ← 新增
  ├── command.get_names.response             ← 新增
  ├── command.add_object.response            ← 新增
  ├── command.delete_object.response         ← 新增
  ├── command.reboot.response                ← 新增
  ├── command.factory_reset.response         ← 新增
  ├── command.get_attrs.response             ← 新增
  └── command.set_attrs.response             ← 新增
```
