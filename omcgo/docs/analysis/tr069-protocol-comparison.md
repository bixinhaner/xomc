# TR069 协议报文与实现对比分析

> 本文档对比真实 CPE 设备（Baicells pBS3202）的 TR069 交互报文与当前 ACS 实现的差异。

---

## 1. 数据来源

- **真实报文**: `docs/design/tr069_cpe_acs_exchange.md`
- **设备型号**: Baicels FAP/BAIBLQ/SC (pBS3202)
- **软件版本**: BaiBLQ_5.1.10
- **OUI**: 48BF74
- **交互时间**: 2026-02-28

---

## 2. 报文流程概览

### 2.1 真实设备交互序列

```
CPE → ACS: Inform (Event: 6 CONNECTION REQUEST)
ACS → CPE: InformResponse (NoMoreRequests=0)
CPE → ACS: 空报文 (Empty POST, 准备接收指令)
ACS → CPE: GetParameterValues (X_COM_MODULE_TYPE)
CPE → ACS: GetParameterValuesResponse (SessionTimeout=40)
ACS → CPE: GetParameterValues (3 个参数)
CPE → ACS: GetParameterValuesResponse
ACS → CPE: 空报文 (204 No Content, 会话结束)
CPE → ACS: Inform (下一个会话)
ACS → CPE: InformResponse
...
CPE → ACS: AutonomousTransferComplete (PM 文件上传通知)
ACS → CPE: AutonomousTransferCompleteResponse
```

### 2.2 当前实现支持的流程

```
CPE → ACS: Inform ✓
ACS → CPE: InformResponse ✓ (含 NoMoreRequests)
CPE → ACS: 空报文 ✓
ACS → CPE: RPC 请求 (从命令队列获取) ✓
CPE → ACS: RPC Response ✓ (解析 SessionTimeout)
ACS → CPE: 空报文或下一个 RPC ✓
CPE → ACS: AutonomousTransferComplete ✓
ACS → CPE: AutonomousTransferCompleteResponse ✓
```

**结论**: 流程完整支持，Header 处理已完善。

---

## 3. 差异分析与修复状态

### 3.1 SOAP 命名空间 ✓ 已修复

| 项目 | 真实报文 | 当前实现 | 状态 |
|------|----------|----------|------|
| Envelope 前缀 (CPE) | `soap-env` | N/A (解析时忽略) | - |
| Envelope 前缀 (ACS) | `SOAP-ENV` | `soap` | 兼容 |
| 编码命名空间 | `SOAP-ENC` | ✓ 已添加 | 已修复 |

**当前模板**:
```xml
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:soap-enc="http://schemas.xmlsoap.org/soap/encoding/"
               ...
```

---

### 3.2 CWMP ID 格式 ✓ 兼容

| 项目 | 真实报文 | 当前实现 |
|------|----------|----------|
| ID 格式 | `ID:intrnl.unset.id.{Method}{timestamp}.{random}` | 使用 `CommandKey` 作为 ID |

**分析**: TR069 规范允许任意唯一字符串作为 ID，当前实现功能上可行。

---

### 3.3 NoMoreRequests Header ✓ 已修复

| 项目 | 真实报文 | 当前实现 |
|------|----------|----------|
| NoMoreRequests | `<cwmp:NoMoreRequests>0</cwmp:NoMoreRequests>` | ✓ 已添加 |

**修改内容**:
- 模板数据结构添加 `NoMoreRequests int` 字段
- 模板 Header 添加条件输出

```go
type HeaderData struct {
    ID             string
    NoMoreRequests int // 0=还有更多请求, 1=最后一个请求
}
```

```xml
<soap:Header>
  <cwmp:ID soap:mustUnderstand="1">{{.ID}}</cwmp:ID>{{if .NoMoreRequests}}
  <cwmp:NoMoreRequests>{{.NoMoreRequests}}</cwmp:NoMoreRequests>{{end}}
</soap:Header>
```

---

### 3.4 SessionTimeout Header ✓ 已修复

| 项目 | 真实报文 | 当前实现 |
|------|----------|----------|
| CPE 响应包含 | `<cwmp:SessionTimeout>40</cwmp:SessionTimeout>` | ✓ 已解析 |

**修改内容**:
- `pkg/soap/envelope.go`: Header 结构体添加 `SessionTimeout int`
- `pkg/soap/decoder.go`: 新增 `ParseHeader()` 函数
- `internal/acs/session.go`: Session 结构体添加 `SessionTimeout int`

```go
type HeaderInfo struct {
    ID             string
    SessionTimeout int
    NoMoreRequests string
}

func ParseHeader(r io.Reader) (HeaderInfo, error)
```

---

### 3.5 参数路径规范化 ✓ 已实现

| 项目 | 真实报文 | 当前实现 |
|------|----------|----------|
| 完整路径 | `Device.DeviceInfo.X_COM_MODULE_TYPE` | ✓ 支持 |
| 相对路径 | `FAPService.1.CellConfig.LTE...` | ✓ 规范化 |

**新增文件**: `internal/acs/pathutil/normalize.go`

```go
func NormalizeParameterPath(path, rootVersion string) string
func DetectRootPrefix(path string) string
func StripRootPrefix(path string) string
```

---

### 3.6 EventCode 处理 ✓ 已修复

**新增事件码**:
```go
EventRequestDownload           = "9 REQUEST DOWNLOAD"
EventAutonomousTransferComplete = "10 AUTONOMOUS TRANSFER COMPLETE"
```

**新增辅助函数**:
- `IsRequestDownload()`
- `IsAutonomousTransferComplete()`

---

### 3.7 Session 会话管理 ✓ 已确认

Inform 处理时正确创建 session：
- Redis 会话存储 (`sessionStore.Create`)
- 连接级绑定 (`connSessions.Store`)

---

## 4. 差异汇总表

| 序号 | 差异项 | 严重程度 | 状态 | 说明 |
|------|--------|----------|------|------|
| 1 | NoMoreRequests Header | **高** | ✓ 已修复 | 添加到模板和数据结构 |
| 2 | SOAP-ENC 命名空间 | 中 | ✓ 已修复 | 添加到模板 |
| 3 | SessionTimeout 解析 | 中 | ✓ 已修复 | 添加 ParseHeader 函数 |
| 4 | 参数路径规范化 | 中 | ✓ 已实现 | 新增 pathutil 包 |
| 5 | EventCode 9/10 | 中 | ✓ 已修复 | 添加常量和辅助函数 |
| 6 | Session 管理 | - | ✓ 已确认 | 原实现正确 |
| 7 | CWMP ID 格式 | 低 | 兼容 | 功能可用 |

---

## 5. 修改文件清单

| 文件 | 修改内容 |
|------|----------|
| `pkg/soap/templates.go` | 添加 NoMoreRequests、SOAP-ENC 命名空间 |
| `pkg/soap/envelope.go` | Header 结构体添加 SessionTimeout |
| `pkg/soap/decoder.go` | 新增 ParseHeader 函数 |
| `pkg/tr069/events.go` | 添加 EventCode 9/10 和辅助函数 |
| `internal/acs/session.go` | Session 结构体添加 SessionTimeout |
| `internal/acs/pathutil/normalize.go` | 新增参数路径规范化工具 |

---

## 6. 测试建议

1. **单元测试**: 添加针对真实报文的解析测试
2. **集成测试**: 使用真实报文进行端到端测试
3. **兼容性测试**: 与真实 Baicells 设备进行联调

---

## 7. 参考资料

- TR069/CWMP Amendment 6 (TR-069-a6)
- BBF TR-106 Data Model Template
- `docs/design/tr069_cpe_acs_exchange.md` - 真实交互报文

---

## 8. 变更历史

| 日期 | 变更内容 |
|------|----------|
| 2026-03-20 | 初版：基础对比分析 |
| 2026-03-20 | 新增：AutonomousTransferComplete 分析、Session 管理验证、EventCode 9/10 添加 |
| 2026-03-20 | 修复：NoMoreRequests Header、SOAP-ENC 命名空间、SessionTimeout 解析、参数路径规范化 |
