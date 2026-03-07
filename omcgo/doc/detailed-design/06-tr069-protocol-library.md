# DD-06: TR069 协议库

> 关联功能域：F01（南向接口）
> 关联 backend-design.md 章节：第四章（TR069 ACS 引擎设计）、第十三章（数据模型抽象 13.1）
> 实施阶段：Phase 1（基础建设）
> 依赖文档：DD-01

---

## 1. 概述

### 1.1 模块定位

TR069 协议库（`pkg/tr069/`、`pkg/soap/`、`pkg/xmlutil/`）是公共可复用的 TR069/CWMP 协议类型定义和工具集，不依赖业务逻辑，可被 ACS 引擎和其他模块引用。

### 1.2 核心职责

- TR069 RPC 类型定义（Inform、Get/Set/Add/Delete/Download/Upload/Reboot/FactoryReset/ScheduleInform）
- Inform 事件码常量
- CWMP 错误码
- SOAP 信封结构与模板
- XML 流式解析工具

---

## 2. 数据模型

### 2.1 TR069 核心类型 — `pkg/tr069/types.go`

```go
// DeviceId CPE 设备标识（来自 Inform）
type DeviceId struct {
    Manufacturer string `xml:"Manufacturer"`
    OUI          string `xml:"OUI"`
    ProductClass string `xml:"ProductClass"`
    SerialNumber string `xml:"SerialNumber"`
}

// ParameterValueStruct 参数值
type ParameterValueStruct struct {
    Name  string `xml:"Name"`
    Value string `xml:"Value"`
    Type  string `xml:"type,attr,omitempty"`
}

// EventStruct Inform 事件
type EventStruct struct {
    EventCode  string `xml:"EventCode"`
    CommandKey string `xml:"CommandKey"`
}

// InformMessage Inform 报文
type InformMessage struct {
    DeviceId       DeviceId               `xml:"DeviceId"`
    Event          []EventStruct          `xml:"Event>EventStruct"`
    MaxEnvelopes   int                    `xml:"MaxEnvelopes"`
    CurrentTime    time.Time              `xml:"CurrentTime"`
    RetryCount     int                    `xml:"RetryCount"`
    ParameterList  []ParameterValueStruct `xml:"ParameterList>ParameterValueStruct"`
}

// InformResponse Inform 响应
type InformResponse struct {
    MaxEnvelopes int `xml:"MaxEnvelopes"`
}
```

### 2.2 RPC 请求/响应类型 — `pkg/tr069/types.go`（续）

```go
// GetParameterValues
type GetParameterValuesRequest struct {
    ParameterNames []string `xml:"ParameterNames>string"`
}

type GetParameterValuesResponse struct {
    ParameterList []ParameterValueStruct `xml:"ParameterList>ParameterValueStruct"`
}

// SetParameterValues
type SetParameterValuesRequest struct {
    ParameterList []ParameterValueStruct `xml:"ParameterList>ParameterValueStruct"`
    ParameterKey  string                 `xml:"ParameterKey"`
}

type SetParameterValuesResponse struct {
    Status int `xml:"Status"` // 0=立即生效 1=需重启
}

// GetParameterNames
type GetParameterNamesRequest struct {
    ParameterPath string `xml:"ParameterPath"`
    NextLevel     bool   `xml:"NextLevel"`
}

type ParameterInfoStruct struct {
    Name     string `xml:"Name"`
    Writable bool   `xml:"Writable"`
}

type GetParameterNamesResponse struct {
    ParameterList []ParameterInfoStruct `xml:"ParameterList>ParameterInfoStruct"`
}

// AddObject / DeleteObject
type AddObjectRequest struct {
    ObjectName   string `xml:"ObjectName"`
    ParameterKey string `xml:"ParameterKey"`
}

type AddObjectResponse struct {
    InstanceNumber int `xml:"InstanceNumber"`
    Status         int `xml:"Status"`
}

type DeleteObjectRequest struct {
    ObjectName   string `xml:"ObjectName"`
    ParameterKey string `xml:"ParameterKey"`
}

type DeleteObjectResponse struct {
    Status int `xml:"Status"`
}

// Download
type DownloadRequest struct {
    CommandKey     string `xml:"CommandKey"`
    FileType       string `xml:"FileType"`       // "1 Firmware" / "3 Vendor Config"
    URL            string `xml:"URL"`
    Username       string `xml:"Username"`
    Password       string `xml:"Password"`
    FileSize       int64  `xml:"FileSize"`
    TargetFileName string `xml:"TargetFileName"`
    DelaySeconds   int    `xml:"DelaySeconds"`
    SuccessURL     string `xml:"SuccessURL"`
    FailureURL     string `xml:"FailureURL"`
}

type DownloadResponse struct {
    Status       int       `xml:"Status"` // 0=完成 1=进行中
    StartTime    time.Time `xml:"StartTime"`
    CompleteTime time.Time `xml:"CompleteTime"`
}

// Upload
type UploadRequest struct {
    CommandKey string `xml:"CommandKey"`
    FileType   string `xml:"FileType"` // "1 Vendor Config" / "2 Vendor Log" / "4 PM File"
    URL        string `xml:"URL"`
    Username   string `xml:"Username"`
    Password   string `xml:"Password"`
    DelaySeconds int  `xml:"DelaySeconds"`
}

type UploadResponse struct {
    Status       int       `xml:"Status"`
    StartTime    time.Time `xml:"StartTime"`
    CompleteTime time.Time `xml:"CompleteTime"`
}

// Reboot / FactoryReset
type RebootRequest struct {
    CommandKey string `xml:"CommandKey"`
}

type RebootResponse struct{}

type FactoryResetResponse struct{}

// ScheduleInform
type ScheduleInformRequest struct {
    DelaySeconds int    `xml:"DelaySeconds"`
    CommandKey   string `xml:"CommandKey"`
}

type ScheduleInformResponse struct{}

// TransferComplete（CPE → ACS）
type TransferComplete struct {
    CommandKey   string    `xml:"CommandKey"`
    FaultStruct  *Fault    `xml:"FaultStruct"`
    StartTime    time.Time `xml:"StartTime"`
    CompleteTime time.Time `xml:"CompleteTime"`
}
```

### 2.3 事件码常量 — `pkg/tr069/events.go`

```go
const (
    EventBootstrap        = "0 BOOTSTRAP"
    EventBoot             = "1 BOOT"
    EventPeriodic         = "2 PERIODIC"
    EventScheduled        = "3 SCHEDULED"
    EventValueChange      = "4 VALUE CHANGE"
    EventKicked           = "5 KICKED"
    EventConnectionRequest = "6 CONNECTION REQUEST"
    EventTransferComplete = "7 TRANSFER COMPLETE"
    EventDiagnosticsComplete = "8 DIAGNOSTICS COMPLETE"
)

// IsBootstrap 判断是否为首次开站
func IsBootstrap(events []EventStruct) bool

// IsPeriodic 判断是否为周期心跳
func IsPeriodic(events []EventStruct) bool

// HasEvent 检查事件列表中是否包含指定事件
func HasEvent(events []EventStruct, code string) bool
```

### 2.4 CWMP 错误码 — `pkg/tr069/faults.go`

```go
type Fault struct {
    FaultCode   int    `xml:"FaultCode"`
    FaultString string `xml:"FaultString"`
}

const (
    FaultMethodNotSupported    = 9000
    FaultRequestDenied         = 9001
    FaultInternalError         = 9002
    FaultInvalidArguments      = 9003
    FaultResourcesExceeded     = 9004
    FaultInvalidParameterName  = 9005
    FaultInvalidParameterType  = 9006
    FaultInvalidParameterValue = 9007
    FaultNotWritable           = 9008
    FaultNotificationRejected  = 9009
    FaultDownloadFailure       = 9010
    FaultUploadFailure         = 9011
    FaultFileTransferAuth      = 9012
    FaultFileTransferProtocol  = 9013
)

func NewFault(code int, message string) *Fault
func (f *Fault) Error() string
```

---

## 3. SOAP 工具 — `pkg/soap/`

### 3.1 SOAP 信封 — `pkg/soap/envelope.go`

```go
// Envelope SOAP 信封
type Envelope struct {
    Header Header `xml:"Header"`
    Body   Body   `xml:"Body"`
}

type Header struct {
    ID       string `xml:"cwmp:ID"`
    NoMoreRequests string `xml:"cwmp:NoMoreRequests,omitempty"`
}

type Body struct {
    Content interface{} `xml:",any"` // 动态内容
}

// ParseEnvelope 从 io.Reader 解析 SOAP 信封
func ParseEnvelope(r io.Reader) (*Envelope, error)

// DetectRPCMethod 检测 Body 中的 RPC 方法名
func DetectRPCMethod(body []byte) (string, error)
```

### 3.2 SOAP 模板 — `pkg/soap/templates.go`

使用 `text/template` 预编译所有 ACS → CPE 的 SOAP 响应模板：

```go
var (
    InformResponseTemplate      *template.Template
    GetParameterValuesTemplate  *template.Template
    SetParameterValuesTemplate  *template.Template
    GetParameterNamesTemplate   *template.Template
    AddObjectTemplate           *template.Template
    DeleteObjectTemplate        *template.Template
    DownloadTemplate            *template.Template
    UploadTemplate              *template.Template
    RebootTemplate              *template.Template
    FactoryResetTemplate        *template.Template
    ScheduleInformTemplate      *template.Template
    FaultResponseTemplate       *template.Template
)

func init() {
    // 预编译所有模板
}

// RenderResponse 渲染 RPC 响应为 SOAP XML
func RenderResponse(tmpl *template.Template, data interface{}) ([]byte, error)
```

**模板示例**（InformResponse）：

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">{{.ID}}</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:InformResponse>
      <MaxEnvelopes>1</MaxEnvelopes>
    </cwmp:InformResponse>
  </soap:Body>
</soap:Envelope>
```

---

## 4. XML 工具 — `pkg/xmlutil/`

### 4.1 流式解析 Helper — `pkg/xmlutil/decoder.go`

```go
// FindElement 在 XML 流中查找指定元素
func FindElement(decoder *xml.Decoder, name string) (*xml.StartElement, error)

// ReadText 读取当前元素的文本内容
func ReadText(decoder *xml.Decoder) (string, error)

// SkipElement 跳过当前元素及其子元素
func SkipElement(decoder *xml.Decoder) error
```

### 4.2 etree 工具 — `pkg/xmlutil/etree.go`

```go
// WalkParameterTree 遍历 etree 参数树，对每个叶子节点执行 callback
func WalkParameterTree(root *etree.Element, callback func(path string, elem *etree.Element) error) error

// FindParameterByPath 在 etree 中按 TR069 路径查找参数
func FindParameterByPath(root *etree.Element, path string) *etree.Element
```

---

## 5. 实施子阶段

### 阶段 6a：TR069 类型 + 事件码 + 错误码

**交付物**：`pkg/tr069/types.go`、`pkg/tr069/events.go`、`pkg/tr069/faults.go`
**验证**：编译通过 + 事件判断函数单元测试

### 阶段 6b：SOAP 信封 + 模板

**交付物**：`pkg/soap/envelope.go`、`pkg/soap/templates.go`
**验证**：解析/生成 SOAP XML 往返测试

### 阶段 6c：XML 解析工具 + etree 封装

**交付物**：`pkg/xmlutil/decoder.go`、`pkg/xmlutil/etree.go`
**验证**：解析示例 Inform 报文测试

### 阶段 6d：测试 Fixtures

**交付物**：`test/fixtures/soap/` 下放置各类 SOAP 报文样本
**验证**：所有 fixture 文件可被正确解析

---

## 6. 文件清单

```
pkg/tr069/types.go
pkg/tr069/events.go
pkg/tr069/faults.go
pkg/soap/envelope.go
pkg/soap/templates.go
pkg/xmlutil/decoder.go
pkg/xmlutil/etree.go
test/fixtures/soap/inform_bootstrap.xml
test/fixtures/soap/inform_periodic.xml
test/fixtures/soap/get_parameter_values_response.xml
test/fixtures/soap/set_parameter_values_response.xml
test/fixtures/soap/transfer_complete.xml
```

---

## 7. 测试策略

- **Table-driven 测试**：每种 RPC 类型的序列化/反序列化
- **Fixture 测试**：使用真实运营商 SOAP 报文样本验证解析
- **模板渲染测试**：验证每个模板输出的 XML 格式正确

---

## 8. 参考

- backend-design.md 第四章：TR069 ACS 引擎设计（4.3 SOAP/XML 处理策略）
- backend-design.md 第���三章：统一参数模型（13.1）
- doc/features/01-southbound-interface.md：F01 南向接口管理
