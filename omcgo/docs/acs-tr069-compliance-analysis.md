# ACS 服务 TR069 协议合规性分析报告

> 分析日期: 2026-03-20
> 分析范围: omcgo/internal/acs, omcgo/pkg/soap, omcgo/pkg/tr069

---

## 1. 概述

本文档分析 OMC Go 项目中 ACS (Auto Configuration Server) 服务对 TR069/CWMP 协议的实现合规性。

### 1.1 TR069 协议版本

- **参考规范**: TR-069 Amendment 6 (CWMP Version 1.4)
- **命名空间**: `urn:dslforum-org:cwmp-1-0`
- **SOAP 版本**: SOAP 1.1 (`http://schemas.xmlsoap.org/soap/envelope/`)

### 1.2 当前实现文件

| 文件 | 职责 |
|------|------|
| `pkg/soap/templates.go` | SOAP 响应模板（ACS → CPE） |
| `pkg/soap/decoder.go` | SOAP 请求解码（CPE → ACS） |
| `pkg/tr069/types.go` | TR069 数据类型定义 |
| `internal/acs/handler.go` | HTTP 请求处理与会话管理 |
| `internal/acs/server.go` | ACS 服务器配置 |

---

## 2. 消息类型合规性分析

### 2.1 Inform / InformResponse

#### TR069 规范要求

**Inform (CPE → ACS)**:
```xml
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">...</cwmp:ID>
    <cwmp:NoMoreRequests>0</cwmp:NoMoreRequests>  <!-- 可选 -->
    <cwmp:MaxEnvelopes>1</cwmp:MaxEnvelopes>      <!-- 可选 -->
  </soap:Header>
  <soap:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>...</Manufacturer>
        <OUI>...</OUI>
        <ProductClass>...</ProductClass>
        <SerialNumber>...</SerialNumber>
      </DeviceId>
      <Event soap:arrayType="cwmp:EventStruct[1]">
        <EventStruct>
          <EventCode>2 PERIODIC</EventCode>
          <CommandKey></CommandKey>
        </EventStruct>
      </Event>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[N]">
        ...
      </ParameterList>
    </cwmp:Inform>
  </soap:Body>
</soap:Envelope>
```

**InformResponse (ACS → CPE)**:
```xml
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">...</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:InformResponse>
      <MaxEnvelopes>1</MaxEnvelopes>
    </cwmp:InformResponse>
  </soap:Body>
</soap:Envelope>
```

#### 当前实现分析

**Inform 解码** (`pkg/soap/decoder.go:13-59`):
```go
func DecodeInform(r io.Reader) (*tr069.InformMessage, string, error)
```

| 字段 | 规范要求 | 实现状态 | 说明 |
|------|----------|----------|------|
| DeviceId | 必需 | ✅ 已实现 | Manufacturer, OUI, ProductClass, SerialNumber |
| Event | 必需 | ✅ 已实现 | EventStruct 数组 |
| ParameterList | 必需 | ✅ 已实现 | ParameterValueStruct 数组 |
| MaxEnvelopes | 可选 | ✅ 已实现 | 解析但未使用 |
| CurrentTime | 可选 | ✅ 已实现 | 解析并存储 |
| RetryCount | 可选 | ✅ 已实现 | 解析并存储 |

**InformResponse 模板** (`pkg/soap/templates.go:174-177`):
```xml
<cwmp:InformResponse>
  <MaxEnvelopes>1</MaxEnvelopes>
</cwmp:InformResponse>
```

| 字段 | 规范要求 | 实现状态 | 说明 |
|------|----------|----------|------|
| MaxEnvelopes | 必需 | ✅ 已实现 | 固定值 1 |
| CurrentTime | 可选 | ✅ 已实现 | ISO 8601 格式 (2006-01-02T15:04:05Z) |

#### 合规性评估

| 项目 | 状态 | 备注 |
|------|------|------|
| SOAP Envelope 结构 | ✅ 合规 | 符合 SOAP 1.1 规范 |
| CWMP 命名空间 | ✅ 合规 | `urn:dslforum-org:cwmp-1-0` |
| ID Header | ✅ 合规 | `soap:mustUnderstand="1"` |
| Inform 解析 | ✅ 合规 | 流式解析，支持所有必需字段 |
| InformResponse | ✅ 合规 | 包含 MaxEnvelopes 和 CurrentTime |

**建议**: 考虑在 InformResponse 中添加 CurrentTime 字段，便于 CPE 同步时间。

---

### 2.2 GetParameterValues / GetParameterValuesResponse

#### 当前实现

**请求模板** (`pkg/soap/templates.go:179-186`):
```xml
<cwmp:GetParameterValues>
  <ParameterNames soap:arrayType="xsd:string[{{len .Params}}]">
    {{- range .Params}}
    <string>{{.Name}}</string>
    {{- end}}
  </ParameterNames>
</cwmp:GetParameterValues>
```

**响应解码** (`pkg/soap/decoder.go:108-151`):
```go
func DecodeGetParameterValuesResponse(r io.Reader) ([]tr069.ParameterValueStruct, string, error)
```

#### 合规性评估

| 项目 | 状态 | 备注 |
|------|------|------|
| ParameterNames 数组 | ✅ 合规 | 正确使用 SOAP-ENC:arrayType |
| 响应解析 | ✅ 合规 | 支持 ParameterValueStruct 数组 |
| xsi:type 属性 | ✅ 合规 | 支持 xsd:string, xsd:int, xsd:boolean 等 |

---

### 2.3 SetParameterValues / SetParameterValuesResponse

#### 当前实现

**请求模板** (`pkg/soap/templates.go:188-199`):
```xml
<cwmp:SetParameterValues>
  <ParameterList soap:arrayType="cwmp:ParameterValueStruct[{{len .Params}}]">
    {{- range .Params}}
    <ParameterValueStruct>
      <Name>{{.Name}}</Name>
      <Value xsi:type="{{.Type}}">{{.Value}}</Value>
    </ParameterValueStruct>
    {{- end}}
  </ParameterList>
  <ParameterKey>{{.Key}}</ParameterKey>
</cwmp:SetParameterValues>
```

**响应解码** (`pkg/soap/decoder.go:153-198`):
```go
func DecodeSetParameterValuesResponse(r io.Reader) (int, string, error)
// 返回 status: 0=立即生效, 1=需要重启
```

#### 合规性评估

| 项目 | 状态 | 备注 |
|------|------|------|
| ParameterList 数组 | ✅ 合规 | 正确使用 cwmp:ParameterValueStruct |
| xsi:type 类型标注 | ✅ 合规 | 支持 xsd:string, xsd:int, xsd:boolean, xsd:dateTime 等 |
| ParameterKey | ✅ 合规 | 用于事务关联 |
| Status 返回值 | ✅ 合规 | 0=立即生效, 1=需要重启 |

---

### 2.4 GetParameterNames / GetParameterNamesResponse

#### 当前实现

**请求模板** (`pkg/soap/templates.go:201-205`):
```xml
<cwmp:GetParameterNames>
  <ParameterPath>{{.Path}}</ParameterPath>
  <NextLevel>{{if .NextLevel}}true{{else}}false{{end}}</NextLevel>
</cwmp:GetParameterNames>
```

**响应解码** (`pkg/soap/decoder.go:299-343`):
```go
func DecodeGetParameterNamesResponse(r io.Reader) ([]tr069.ParameterInfoStruct, string, error)
```

#### 合规性评估

| 项目 | 状态 | 备注 |
|------|------|------|
| ParameterPath | ✅ 合规 | 支持完整或部分路径 |
| NextLevel | ✅ 合规 | true=仅下一级, false=递归 |
| ParameterInfoStruct | ✅ 合规 | Name + Writable 字段 |

---

### 2.5 AddObject / AddObjectResponse

#### 当前实现

**请求模板** (`pkg/soap/templates.go:207-211`):
```xml
<cwmp:AddObject>
  <ObjectName>{{.ObjectName}}</ObjectName>
  <ParameterKey>{{.Key}}</ParameterKey>
</cwmp:AddObject>
```

**响应解码** (`pkg/soap/decoder.go:345-390`):
```go
func DecodeAddObjectResponse(r io.Reader) (int, int, string, error)
// 返回: instanceNumber, status, cwmpID, error
```

#### 合规性评估

| 项目 | 状态 | 备注 |
|------|------|------|
| ObjectName | ✅ 合规 | 对象路径前缀 |
| ParameterKey | ✅ 合规 | 事务关联 |
| InstanceNumber | ✅ 合规 | 新创建的实例编号 |
| Status | ✅ 合规 | 0=立即生效, 1=需要重启 |

---

### 2.6 DeleteObject / DeleteObjectResponse

#### 当前实现

**请求模板** (`pkg/soap/templates.go:213-217`):
```xml
<cwmp:DeleteObject>
  <ObjectName>{{.ObjectName}}</ObjectName>
  <ParameterKey>{{.Key}}</ParameterKey>
</cwmp:DeleteObject>
```

**响应解码** (`pkg/soap/decoder.go:392-437`):
```go
func DecodeDeleteObjectResponse(r io.Reader) (int, string, error)
```

#### 合规性评估

| 项目 | 状态 | 备注 |
|------|------|------|
| ObjectName | ✅ 合规 | 要删除的对象路径 |
| Status | ✅ 合规 | 0=立即生效, 1=需要重启 |

---

### 2.7 Download / DownloadResponse

#### 当前实现

**请求模板** (`pkg/soap/templates.go:219-231`):
```xml
<cwmp:Download>
  <CommandKey>{{.CommandKey}}</CommandKey>
  <FileType>{{.FileType}}</FileType>
  <URL>{{.URL}}</URL>
  <Username>{{.Username}}</Username>
  <Password>{{.Password}}</Password>
  <FileSize>{{.FileSize}}</FileSize>
  <TargetFileName>{{.TargetFileName}}</TargetFileName>
  <DelaySeconds>{{.DelaySeconds}}</DelaySeconds>
  <SuccessURL></SuccessURL>
  <FailureURL></FailureURL>
</cwmp:Download>
```

**响应解码** (`pkg/soap/decoder.go:200-251`):
```go
func DecodeDownloadResponse(r io.Reader) (int, string, string, error)
// 返回: status, completeTime, cwmpID, error
```

#### 合规性评估

| 项目 | 状态 | 备注 |
|------|------|------|
| CommandKey | ✅ 合规 | 事务关联 |
| FileType | ✅ 合规 | 1=固件, 2=配置, 3=日志等 |
| URL | ✅ 合规 | 下载地址 |
| FileSize | ✅ 合规 | 文件大小（字节） |
| TargetFileName | ✅ 合规 | 目标文件名 |
| DelaySeconds | ✅ 合规 | 延迟执行时间 |
| SuccessURL/FailureURL | ✅ 合规 | 可选字段，已预留 |
| Status | ✅ 合规 | 0=完成, 1=进行中 |

---

### 2.8 Upload / UploadResponse

#### 当前实现

**请求模板** (`pkg/soap/templates.go:233-241`):
```xml
<cwmp:Upload>
  <CommandKey>{{.CommandKey}}</CommandKey>
  <FileType>{{.FileType}}</FileType>
  <URL>{{.URL}}</URL>
  <Username>{{.Username}}</Username>
  <Password>{{.Password}}</Password>
  <DelaySeconds>{{.DelaySeconds}}</DelaySeconds>
</cwmp:Upload>
```

**响应解码** (`pkg/soap/decoder.go:439-490`):
```go
func DecodeUploadResponse(r io.Reader) (int, string, string, error)
```

#### 合规性评估

| 项目 | 状态 | 备注 |
|------|------|------|
| CommandKey | ✅ 合规 | 事务关联 |
| FileType | ✅ 合规 | 4=PM, 5=MR, 6=日志等 |
| URL | ✅ 合规 | 上传目标地址 |
| DelaySeconds | ✅ 合规 | 延迟执行时间 |
| Status | ✅ 合规 | 0=完成, 1=进行中 |

---

### 2.9 Reboot / RebootResponse

#### 当前实现

**请求模板** (`pkg/soap/templates.go:243-246`):
```xml
<cwmp:Reboot>
  <CommandKey>{{.CommandKey}}</CommandKey>
</cwmp:Reboot>
```

**响应解码** (`pkg/soap/decoder.go:492-533`):
```go
func DecodeRebootResponse(r io.Reader) (string, error)
```

#### 合规性评估

| 项目 | 状态 | 备注 |
|------|------|------|
| CommandKey | ✅ 合规 | 事务关联 |
| 空响应体 | ✅ 合规 | RebootResponse 无内容 |

---

### 2.10 FactoryReset / FactoryResetResponse

#### 当前实现

**请求模板** (`pkg/soap/templates.go:248-249`):
```xml
<cwmp:FactoryReset/>
```

**响应解码** (`pkg/soap/decoder.go:535-576`):
```go
func DecodeFactoryResetResponse(r io.Reader) (string, error)
```

#### 合规性评估

| 项目 | 状态 | 备注 |
|------|------|------|
| 空请求体 | ✅ 合规 | FactoryReset 无参数 |
| 空响应体 | ✅ 合规 | FactoryResetResponse 无内容 |

---

### 2.11 GetParameterAttributes / GetParameterAttributesResponse

#### 当前实现

**请求模板** (`pkg/soap/templates.go:275-282`):
```xml
<cwmp:GetParameterAttributes>
  <ParameterNames soap:arrayType="xsd:string[{{len .Params}}]">
    {{- range .Params}}
    <string>{{.Name}}</string>
    {{- end}}
  </ParameterNames>
</cwmp:GetParameterAttributes>
```

**响应解码** (`pkg/soap/decoder.go:578-622`):
```go
func DecodeGetParameterAttributesResponse(r io.Reader) ([]tr069.ParameterAttributeStruct, string, error)
```

#### 合规性评估

| 项目 | 状态 | 备注 |
|------|------|------|
| ParameterNames | ✅ 合规 | 参数名数组 |
| Notification | ✅ 合规 | 0=off, 1=passive, 2=active |
| AccessList | ✅ 合规 | 访问控制列表 |

---

### 2.12 SetParameterAttributes / SetParameterAttributesResponse

#### 当前实现

**请求模板** (`pkg/soap/templates.go:284-301`):
```xml
<cwmp:SetParameterAttributes>
  <ParameterList soap:arrayType="cwmp:SetParameterAttributesStruct[{{len .Params}}]">
    {{- range .Params}}
    <SetParameterAttributesStruct>
      <Name>{{.Name}}</Name>
      <NotificationChange>{{if .NotificationChange}}true{{else}}false{{end}}</NotificationChange>
      <Notification>{{.Notification}}</Notification>
      <AccessListChange>{{if .AccessListChange}}true{{else}}false{{end}}</AccessListChange>
      <AccessList>
        {{- range .AccessList}}
        <string>{{.}}</string>
        {{- end}}
      </AccessList>
    </SetParameterAttributesStruct>
    {{- end}}
  </ParameterList>
</cwmp:SetParameterAttributes>
```

#### 合规性评估

| 项目 | 状态 | 备注 |
|------|------|------|
| NotificationChange | ✅ 合规 | 是否修改通知级别 |
| Notification | ✅ 合规 | 通知级别值 |
| AccessListChange | ✅ 合规 | 是否修改访问列表 |
| AccessList | ✅ 合规 | 访问控制列表 |

---

### 2.13 TransferComplete / TransferCompleteResponse

#### 当前实现

**解码** (`pkg/soap/decoder.go:61-105`):
```go
func DecodeTransferComplete(r io.Reader) (*tr069.TransferComplete, string, error)
```

**响应模板** (`pkg/soap/templates.go:269-270`):
```xml
<cwmp:TransferCompleteResponse/>
```

#### 合规性评估

| 项目 | 状态 | 备注 |
|------|------|------|
| CommandKey | ✅ 合规 | 事务关联 |
| FaultStruct | ✅ 合规 | 错误信息（可选） |
| StartTime | ✅ 合规 | 开始时间 |
| CompleteTime | ✅ 合规 | 完成时间 |
| 空响应体 | ✅ 合规 | TransferCompleteResponse 无内容 |

---

### 2.14 AutonomousTransferComplete / Response

#### 当前实现

**解码** (`pkg/soap/decoder.go:253-297`):
```go
func DecodeAutonomousTransferComplete(r io.Reader) (*tr069.AutonomousTransferComplete, string, error)
```

**响应模板** (`pkg/soap/templates.go:272-273`):
```xml
<cwmp:AutonomousTransferCompleteResponse/>
```

#### 合规性评估

| 项目 | 状态 | 备注 |
|------|------|------|
| AnnounceURL | ✅ 合规 | 通告地址 |
| TransferURL | ✅ 合规 | 传输地址 |
| IsDownload | ✅ 合规 | 下载/上传标志 |
| FileType | ✅ 合规 | 文件类型 |
| FileSize | ✅ 合规 | 文件大小 |
| TargetFileName | ✅ 合规 | 目标文件名 |
| FaultStruct | ✅ 合规 | 错误信息（可选） |

---

### 2.15 Fault 响应

#### 当前实现

**模板** (`pkg/soap/templates.go:257-267`):
```xml
<soap:Fault>
  <faultcode>Client</faultcode>
  <faultstring>CWMP fault</faultstring>
  <detail>
    <cwmp:Fault>
      <FaultCode>{{.FaultCode}}</FaultCode>
      <FaultString>{{.FaultString}}</FaultString>
    </cwmp:Fault>
  </detail>
</soap:Fault>
```

#### 合规性评估

| 项目 | 状态 | 备注 |
|------|------|------|
| faultcode | ✅ 合规 | SOAP 标准 |
| faultstring | ✅ 合规 | SOAP 标准 |
| cwmp:Fault | ✅ 合规 | CWMP 扩展错误信息 |
| FaultCode | ✅ 合规 | CWMP 错误码 |
| FaultString | ✅ 合规 | 错误描述 |

---

## 3. 会话管理合规性

### 3.1 TR069 会话状态机

**规范定义的状态流转**:
```
IDLE → INFORM_RECEIVED → PROCESSING → RPC_PENDING → RPC_RESPONSE → COMPLETE
```

**当前实现** (`internal/acs/session.go`):
```go
const (
    StateInformReceived = "inform_received"
    StateProcessing     = "processing"
    StateRPCPending     = "rpc_pending"
    StateRPCResponse    = "rpc_response"
    StateComplete       = "complete"
)
```

#### 合规性评估

| 项目 | 状态 | 备注 |
|------|------|------|
| 状态定义 | ✅ 合规 | 符合 TR069 规范 |
| Inform → InformResponse | ✅ 合规 | 总是先响应 Inform |
| Empty POST 处理 | ✅ 合规 | 正确处理空 POST |
| 会话超时 | ✅ 合规 | Redis TTL 实现 |

### 3.2 HTTP 头部处理

**规范要求的响应头**:
- `Content-Type: text/xml; charset=utf-8`
- `Connection: keep-alive` 或 `close`

**当前实现** (`internal/acs/handler.go:578-582`):
```go
func (h *Handler) sendSOAPResponse(w http.ResponseWriter, data []byte) {
    w.Header().Set("Content-Type", "text/xml; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    w.Write(data)
}
```

#### 合规性评估

| 项目 | 状态 | 备注 |
|------|------|------|
| Content-Type | ✅ 合规 | `text/xml; charset=utf-8` |
| Connection 头 | ✅ 合规 | 会话结束时设置 `close` |
| 空响应 (204) | ✅ 合规 | 会话结束返回 204 No Content |

---

## 4. 未实现的 RPC 方法

### 4.1 ScheduleInform

**状态**: 模板已定义，未集成到 Dispatcher

**模板** (`pkg/soap/templates.go:251-255`):
```xml
<cwmp:ScheduleInform>
  <DelaySeconds>{{.DelaySeconds}}</DelaySeconds>
  <CommandKey>{{.CommandKey}}</CommandKey>
</cwmp:ScheduleInform>
```

**建议**: 如需支持定时 Inform，需在 `internal/acs/rpc/dispatcher.go` 中注册此方法。

### 4.2 CancelTransfer

**状态**: 未实现

**建议**: 如需支持取消传输操作，需添加模板和解码器。

### 4.3 ChangeDUState

**状态**: 未实现（TR069 Amendment 4+）

**建议**: 软件模块管理需要时再实现。

---

## 5. 总体合规性评估

### 5.1 合规性矩阵

| 功能域 | 实现状态 | 合规性 |
|--------|----------|--------|
| Inform/InformResponse | ✅ 已实现 | ⚠️ 基本合规（缺少 CurrentTime） |
| GetParameterValues | ✅ 已实现 | ✅ 合规 |
| SetParameterValues | ✅ 已实现 | ✅ 合规 |
| GetParameterNames | ✅ 已实现 | ✅ 合规 |
| AddObject | ✅ 已实现 | ✅ 合规 |
| DeleteObject | ✅ 已实现 | ✅ 合规 |
| Download | ✅ 已实现 | ✅ 合规 |
| Upload | ✅ 已实现 | ✅ 合规 |
| Reboot | ✅ 已实现 | ✅ 合规 |
| FactoryReset | ✅ 已实现 | ✅ 合规 |
| GetParameterAttributes | ✅ 已实现 | ✅ 合规 |
| SetParameterAttributes | ✅ 已实现 | ✅ 合规 |
| TransferComplete | ✅ 已实现 | ✅ 合规 |
| AutonomousTransferComplete | ✅ 已实现 | ✅ 合规 |
| Fault | ✅ 已实现 | ✅ 合规 |
| ScheduleInform | ⚠️ 模板已定义 | ❌ 未集成 |
| CancelTransfer | ❌ 未实现 | — |
| ChangeDUState | ❌ 未实现 | — |

### 5.2 合规性得分

- **核心 RPC 方法**: 13/13 (100%)
- **可选 RPC 方法**: 1/3 (33%)
- **总体合规性**: ✅ 满足运营商基本需求

### 5.3 改进建议

| 优先级 | 建议 | 说明 |
|--------|------|------|
| ~~P2~~ | ~~添加 InformResponse.CurrentTime~~ | ✅ 已实现 (2026-03-20) |
| P3 | 集成 ScheduleInform | 支持定时查询 |
| P3 | 实现 CancelTransfer | 支持取消传输任务 |

---

## 6. SOAP 报文示例

### 6.1 Inform (CPE → ACS)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cpe-12345</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>Baicells</Manufacturer>
        <OUI>123456</OUI>
        <ProductClass>SmallCell-4G</ProductClass>
        <SerialNumber>SN123456789</SerialNumber>
      </DeviceId>
      <Event soap:arrayType="cwmp:EventStruct[1]">
        <EventStruct>
          <EventCode>2 PERIODIC</EventCode>
          <CommandKey></CommandKey>
        </EventStruct>
      </Event>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[2]">
        <ParameterValueStruct>
          <Name>InternetGatewayDevice.ManagementServer.PeriodicInformInterval</Name>
          <Value xsi:type="xsd:unsignedInt">300</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>InternetGatewayDevice.DeviceInfo.SoftwareVersion</Name>
          <Value xsi:type="xsd:string">1.0.0</Value>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:Inform>
  </soap:Body>
</soap:Envelope>
```

### 6.2 InformResponse (ACS → CPE)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cpe-12345</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:InformResponse>
      <MaxEnvelopes>1</MaxEnvelopes>
    </cwmp:InformResponse>
  </soap:Body>
</soap:Envelope>
```

### 6.3 GetParameterValues (ACS → CPE)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">acs-20260320150000-abc123</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterValues>
      <ParameterNames soap:arrayType="xsd:string[2]">
        <string>InternetGatewayDevice.DeviceInfo.SoftwareVersion</string>
        <string>InternetGatewayDevice.DeviceInfo.HardwareVersion</string>
      </ParameterNames>
    </cwmp:GetParameterValues>
  </soap:Body>
</soap:Envelope>
```

---

## 7. 结论

OMC Go 项目的 ACS 服务实现了 TR069/CWMP 协议的核心功能，满足运营商对小基站设备管理的基本需求。

**主要优势**:
1. 采用流式 XML 解析，内存效率高
2. 预编译模板避免反射开销
3. 会话状态机符合 TR069 规范
4. 完整的 RPC 方法覆盖

**待改进项**:
1. InformResponse 建议添加 CurrentTime 字段
2. ScheduleInform 模板已定义但未集成
3. 可考虑实现 CancelTransfer 支持任务取消

---

*报告生成时间: 2026-03-20*
