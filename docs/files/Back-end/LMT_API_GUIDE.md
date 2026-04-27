# LMT (本地管理工具) API 使用指南

## 目录
- [概述](#概述)
- [快速开始](#快速开始)
- [接口规范](#接口规范)
- [支持的RPC方法](#支持的rpc方法)
- [请求示例](#请求示例)
- [错误处理](#错误处理)
- [性能考虑](#性能考虑)
- [安全说明](#安全说明)

---

## 概述

### 什么是LMT？

LMT (Local Management Tool，本地管理工具) 是TR069协议的简化版本，专为本地管理场景设计。与标准TR069 OMC交互不同，LMT提供：

- **简化流程**: 单次请求-响应模式，无需维护长会话
- **本地访问**: 通过本地网络直接访问设备
- **即时响应**: 无需等待会话建立，立即获得结果
- **零配置**: 无需认证，适合可信内网环境

### LMT vs OMC

| 特性 | LMT | OMC (TR069) |
|------|-----|-------------|
| **会话模式** | 临时会话，单次请求-响应 | 长会话，支持多轮交互 |
| **认证** | 无需认证 | 需要HTTP Basic认证 |
| **端点路径** | `/lmt` | `/acs` 或 `/` |
| **会话超时** | 60秒自动清理 | 可配置（通常较长） |
| **使用场景** | 本地工具、调试、快速查询 | 远程管理、批量操作 |
| **连接保持** | 不保持 | Keep-Alive |

### 双端口架构

TR069 Worker 实现双服务器架构，将 OMC 和 LMT 访问分离：

| 服务器 | 绑定地址 | 端口 | 用途 |
|--------|----------|------|------|
| OMC Server | 0.0.0.0 | 7547 | OMC远程管理 |
| LMT Server | 127.0.0.1 | 17547 | 本地维护工具 |

**安全隔离**: LMT服务器仅绑定到本地回环地址，外部网络无法访问。

**配置选项**:
```cpp
// TR069WorkerConfiguration
bool enableLmtServer = true;                    // 启用LMT专用服务器
uint16_t lmtServerPort = 17547;                 // LMT服务器端口
std::string lmtServerBindAddress = "127.0.0.1"; // LMT绑定地址（仅本地）
bool enableLmtServerAuth = false;               // LMT服务器认证（本地可禁用）
bool enableLmtOnOmcServer = true;               // OMC服务器是否支持LMT路由（调试用）
```

### 架构设计

```
┌─────────────────────────────────────────────────────────────────┐
│  TR069 Worker                                                   │
│                                                                 │
│  ┌────────────────────────┐    ┌────────────────────────┐      │
│  │ OMC Server (0.0.0.0)   │    │ LMT Server (127.0.0.1) │      │
│  │ Port: 7547             │    │ Port: 17547            │      │
│  │                        │    │                        │      │
│  │ POST /lmt → LMT适配器  │    │ POST /lmt → LMT适配器  │      │
│  │ POST /acs → OMC会话    │    │ (仅本地访问)           │      │
│  │ GET  /    → 连接请求   │    │                        │      │
│  └────────────────────────┘    └────────────────────────┘      │
│                    │                      │                     │
│                    └──────────┬───────────┘                     │
│                               ▼                                 │
│  ┌──────────────────────────────────────┐                      │
│  │ LMT协议适配器                        │                      │
│  │  - 临时会话管理                      │                      │
│  │  - SOAP消息解析                      │                      │
│  │  - RPC处理器路由                     │                      │
│  │  - 自动会话清理                      │                      │
│  │  - 部分失败处理支持                  │                      │
│  └──────────────────────────────────────┘                      │
│                               │                                 │
│                               ▼                                 │
│  ┌──────────────────────────────────────┐                      │
│  │ 共享的RPC处理器 (12个)               │                      │
│  │  - GetParameterValues  (支持部分失败)│                      │
│  │  - SetParameterValues  (支持部分失败)│                      │
│  │  - GetParameterNames                 │                      │
│  │  - GetParameterAttributes            │                      │
│  │  - SetParameterAttributes            │                      │
│  │  - AddObject / DeleteObject          │                      │
│  │  - Download / Upload                 │                      │
│  │  - Reboot / FactoryReset             │                      │
│  │  - GetRPCMethods                     │                      │
│  └──────────────────────────────────────┘                      │
└─────────────────────────────────────────────────────────────────┘
```

---

## 快速开始

### 前置条件

1. TR069 Worker进程正在运行
2. ConnectionRequestServer已启动在端口7547
3. 可以访问设备的本地网络

### 最简单的请求示例

使用curl获取设备序列号：

```bash
# 通过 LMT 专用端口 (推荐，仅本地访问)
curl -X POST http://127.0.0.1:17547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">request-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterValues>
      <ParameterNames>
        <string>Device.DeviceInfo.SerialNumber</string>
      </ParameterNames>
    </cwmp:GetParameterValues>
  </soap:Body>
</soap:Envelope>'

# 或通过 OMC 端口 (需要网络可达)
curl -X POST http://192.168.1.100:7547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '...'
```

**响应**:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">0</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterValuesResponse>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[1]">
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SerialNumber</Name>
          <Value xsi:type="xsd:string" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
            XGYTEST000001
          </Value>
          <Writable>false</Writable>
          <Range></Range>
          <RebootRequired>false</RebootRequired>
          <DefaultValue>no default value</DefaultValue>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:GetParameterValuesResponse>
  </soap:Body>
</soap:Envelope>
```

---

## 接口规范

### 基本信息

- **协议**: HTTP/1.1
- **端口**: 7547 (默认TR069端口)
- **路径**: `/lmt`
- **方法**: POST
- **Content-Type**: `text/xml; charset=utf-8`
- **编码**: UTF-8

### HTTP请求头

```http
POST /lmt HTTP/1.1
Host: <设备IP>:7547
Content-Type: text/xml; charset=utf-8
Content-Length: <消息长度>
```

### SOAP消息结构

所有LMT请求必须遵循TR069 SOAP消息格式：

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">消息ID</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <!-- RPC方法调用 -->
  </soap:Body>
</soap:Envelope>
```

**必需元素**:
- `soap:Envelope` - SOAP信封
- `soap:Header` - SOAP头部
  - `cwmp:ID` - 消息标识符（任意字符串）
- `soap:Body` - SOAP正文
  - 包含具体的RPC方法调用

### 响应格式

成功响应：
```http
HTTP/1.1 200 OK
Server: TR-069-CPE/1.0
Content-Type: text/xml; charset=utf-8
Content-Length: <长度>
Connection: close

<SOAP响应消息>
```

错误响应（SOAP Fault）：
```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <soap:Fault>
      <faultcode>Client</faultcode>
      <faultstring>错误描述</faultstring>
      <detail>
        <cwmp:Fault>
          <FaultCode>错误代码</FaultCode>
          <FaultString>详细错误信息</FaultString>
        </cwmp:Fault>
      </detail>
    </soap:Fault>
  </soap:Body>
</soap:Envelope>
```

---

## 支持的RPC方法

LMT支持所有标准TR069 RPC方法，与OMC共享相同的RPC处理器。

### 1. GetParameterValues - 获取参数值

**用途**: 查询一个或多个参数的当前值

**LMT 部分失败处理**:

LMT 请求支持部分失败模式：当批量查询中部分参数失败时，返回成功的结果和失败的详情。

| 请求来源 | 行为 |
|---------|------|
| OMC | 全或无（任一参数失败则整体返回 SOAP Fault） |
| LMT | 部分失败（成功参数返回值 + `PartialFailureList` 错误详情） |

**请求**:
```xml
<cwmp:GetParameterValues>
  <ParameterNames>
    <string>参数路径1</string>
    <string>参数路径2</string>
    <!-- 更多参数 -->
  </ParameterNames>
</cwmp:GetParameterValues>
```

**LMT 部分成功响应** (混合有效/无效参数):
```xml
<cwmp:GetParameterValuesResponse>
  <ParameterList soap:arrayType="cwmp:ParameterValueStruct[N]">
    <ParameterValueStruct>
      <Name>Device.DeviceInfo.Manufacturer</Name>
      <Value xsi:type="xsd:string">OXM</Value>
      <Writable>true</Writable>
      <Range>Length: 0-65535</Range>
      <RebootRequired>false</RebootRequired>
      <DefaultValue>oxm</DefaultValue>
    </ParameterValueStruct>
    <!-- 更多成功的参数 -->
  </ParameterList>
  <!-- LMT 扩展：部分失败详情 -->
  <PartialFailureList soap:arrayType="cwmp:FaultStruct[M]">
    <FaultStruct>
      <ParameterName>Device.Invalid.NotExist</ParameterName>
      <FaultCode>9005</FaultCode>
      <FaultString>Invalid parameter name: Device.Invalid.NotExist</FaultString>
    </FaultStruct>
  </PartialFailureList>
</cwmp:GetParameterValuesResponse>
```

**OMC 响应** (标准TR069):
```xml
<cwmp:GetParameterValuesResponse>
  <ParameterList soap:arrayType="cwmp:ParameterValueStruct[N]">
    <ParameterValueStruct>
      <Name>参数名</Name>
      <Value xsi:type="xsd:数据类型">值</Value>
      <!-- 扩展元数据字段 -->
      <Writable>true/false</Writable>
      <Range>取值范围描述</Range>
      <RebootRequired>true/false</RebootRequired>
      <DefaultValue>默认值</DefaultValue>
    </ParameterValueStruct>
    <!-- 更多参数 -->
  </ParameterList>
</cwmp:GetParameterValuesResponse>
```

**扩展字段说明**:
- `Writable`: 参数是否可写 (true/false)
- `Range`: 参数的取值范围或长度限制 (例如 "0-100", "Length: 0-64")
- `RebootRequired`: 修改该参数是否需要重启生效 (true/false)
- `DefaultValue`: 参数的默认值 (如果没有默认值则显示 "no default value")

**示例**: 参见[请求示例](#请求示例)部分

---

### 2. SetParameterValues - 设置参数值

**用途**: 修改一个或多个参数的值

**LMT 部分失败处理**:

LMT 请求支持部分失败模式：逐个参数设置，即使某些参数失败，其他参数仍会被设置。

| 请求来源 | 行为 |
|---------|------|
| OMC | 全或无（任一参数失败则整体回滚，返回 SOAP Fault） |
| LMT | 部分失败（成功数量 + `PartialFailureList` 错误详情） |

**请求**:
```xml
<cwmp:SetParameterValues>
  <ParameterList soap:arrayType="cwmp:ParameterValueStruct[N]">
    <ParameterValueStruct>
      <Name>参数名</Name>
      <Value xsi:type="xsd:数据类型">新值</Value>
    </ParameterValueStruct>
    <!-- 更多参数 -->
  </ParameterList>
  <ParameterKey>可选的参数键</ParameterKey>
</cwmp:SetParameterValues>
```

**LMT 部分成功响应** (混合可写/只读参数):
```xml
<cwmp:SetParameterValuesResponse>
  <Status>1</Status>  <!-- 1=部分成功 -->
  <SuccessCount>3</SuccessCount>  <!-- LMT 扩展：成功数量 -->
  <!-- LMT 扩展：部分失败详情 -->
  <PartialFailureList soap:arrayType="cwmp:SetParameterFaultStruct[M]">
    <SetParameterFaultStruct>
      <ParameterName>Device.ReadOnly.Param</ParameterName>
      <FaultCode>9008</FaultCode>
      <FaultString>Attempt to set a non-writable parameter</FaultString>
    </SetParameterFaultStruct>
    <SetParameterFaultStruct>
      <ParameterName>Device.Invalid.NotExist</ParameterName>
      <FaultCode>9005</FaultCode>
      <FaultString>Invalid parameter name</FaultString>
    </SetParameterFaultStruct>
  </PartialFailureList>
</cwmp:SetParameterValuesResponse>
```

**OMC 响应** (标准TR069):
```xml
<cwmp:SetParameterValuesResponse>
  <Status>0</Status>  <!-- 0=成功, 1=需要应用 -->
</cwmp:SetParameterValuesResponse>
```

---

### 3. GetParameterNames - 获取参数名称

**用途**: 列出指定路径下的所有参数/对象

**请求**:
```xml
<cwmp:GetParameterNames>
  <ParameterPath>路径前缀</ParameterPath>
  <NextLevel>false</NextLevel>  <!-- true=仅下一层, false=所有层级 -->
</cwmp:GetParameterNames>
```

**响应**:
```xml
<cwmp:GetParameterNamesResponse>
  <ParameterList soap:arrayType="cwmp:ParameterInfoStruct[N]">
    <ParameterInfoStruct>
      <Name>参数/对象名</Name>
      <Writable>true/false</Writable>
    </ParameterInfoStruct>
    <!-- 更多项目 -->
  </ParameterList>
</cwmp:GetParameterNamesResponse>
```

---

### 4. GetParameterAttributes - 获取参数属性

**用途**: 查询参数的属性（通知设置、访问控制等）

**请求**:
```xml
<cwmp:GetParameterAttributes>
  <ParameterNames>
    <string>参数路径</string>
    <!-- 更多参数 -->
  </ParameterNames>
</cwmp:GetParameterAttributes>
```

---

### 5. SetParameterAttributes - 设置参数属性

**用途**: 修改参数的通知设置

**请求**:
```xml
<cwmp:SetParameterAttributes>
  <ParameterList>
    <SetParameterAttributesStruct>
      <Name>参数名</Name>
      <NotificationChange>true</NotificationChange>
      <Notification>2</Notification>  <!-- 0=关闭, 1=被动, 2=主动 -->
    </SetParameterAttributesStruct>
  </ParameterList>
</cwmp:SetParameterAttributes>
```

---

### 6. AddObject - 添加对象实例

**用途**: 在多实例对象下创建新实例

**请求**:
```xml
<cwmp:AddObject>
  <ObjectName>对象路径</ObjectName>
  <ParameterKey>可选的参数键</ParameterKey>
</cwmp:AddObject>
```

**响应**:
```xml
<cwmp:AddObjectResponse>
  <InstanceNumber>新实例编号</InstanceNumber>
  <Status>0</Status>
</cwmp:AddObjectResponse>
```

---

### 7. DeleteObject - 删除对象实例

**用途**: 删除多实例对象的特定实例

**请求**:
```xml
<cwmp:DeleteObject>
  <ObjectName>对象实例路径</ObjectName>
  <ParameterKey>可选的参数键</ParameterKey>
</cwmp:DeleteObject>
```

**响应**:
```xml
<cwmp:DeleteObjectResponse>
  <Status>0</Status>
</cwmp:DeleteObjectResponse>
```

---

### 8. Download - 下载文件

**用途**: 下载固件、配置文件等

**请求**:
```xml
<cwmp:Download>
  <CommandKey>命令键</CommandKey>
  <FileType>文件类型</FileType>
  <URL>下载URL</URL>
  <Username>可选的用户名</Username>
  <Password>可选的密码</Password>
  <FileSize>文件大小(字节)</FileSize>
  <TargetFileName>目标文件名</TargetFileName>
  <DelaySeconds>延迟执行秒数</DelaySeconds>
  <SuccessURL>成功回调URL(可选)</SuccessURL>
  <FailureURL>失败回调URL(可选)</FailureURL>
  <RawMode>升级模式(可选，厂商扩展)</RawMode>
</cwmp:Download>
```

**字段说明**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| CommandKey | string | 是 | 命令标识符，用于关联 TransferComplete 响应 |
| FileType | string | 是 | 文件类型，见下表 |
| URL | string | 是 | 下载地址，支持 HTTP/HTTPS/FTP，本地文件使用 `file://` |
| Username | string | 否 | HTTP Basic/FTP 认证用户名 |
| Password | string | 否 | HTTP Basic/FTP 认证密码 |
| FileSize | unsignedInt | 是 | 文件大小（字节），用于进度计算和空间检查 |
| TargetFileName | string | 否 | 目标文件名（不含路径），留空则从 URL 提取 |
| DelaySeconds | unsignedInt | 否 | 延迟执行秒数，0 表示立即执行 |
| SuccessURL | string | 否 | 下载成功后回调的 URL |
| FailureURL | string | 否 | 下载失败后回调的 URL |
| RawMode | unsignedInt | 否 | 升级模式控制（厂商扩展），见下表 |

**RawMode 取值**（厂商扩展参数）:

| 值 | 说明 | 升级后配置处理 |
|----|------|----------------|
| `0` 或不填 | 保配置升级（默认） | 升级后保留原有配置文件 |
| `1` | 不保配置升级（RawMode） | 升级后恢复出厂配置，不保留原有配置 |

> **注意**: RawMode 参数为厂商扩展，非 TR-069 标准定义。使用场景包括：
> - 配置文件损坏需要完全重置
> - 跨大版本升级时配置不兼容
> - 需要清除所有用户自定义配置

**FileType 取值**:

| FileType | 说明 | 处理方式 |
|----------|------|----------|
| `1 Firmware Upgrade Image` | 固件升级镜像 | 系统升级（自动重启） |
| `3 Vendor Configuration File` | 厂商配置文件 | 配置导入 |
| `10 <OUI> Configuration File` | 厂商特定配置文件 | 自动解析并批量导入参数 |
| `101 Script File` | 脚本文件 | 脚本执行 |
| `103 Base Station Startup File` | 基站启动文件 | 启动配置 |
| `License File` | License 文件 | 签名验证后保存到 config/ |
| `Tr069 Ssl Cert File` | TR069 SSL证书文件 | 保存到运行时目录 data/ |

**响应**:
```xml
<cwmp:DownloadResponse>
  <Status>1</Status>  <!-- 0=已完成, 1=正在进行 -->
  <StartTime>0001-01-01T00:00:00Z</StartTime>
  <CompleteTime>0001-01-01T00:00:00Z</CompleteTime>
</cwmp:DownloadResponse>
```

**URL 格式说明**:

| 场景 | URL 格式 | 示例 |
|------|----------|------|
| HTTP 远程下载 | `http://server/path/file` | `http://192.168.1.10/firmware/v2.0.bin` |
| HTTPS 远程下载 | `https://server/path/file` | `https://ota.example.com/fw.bin` |
| FTP 下载 | `ftp://server/path/file` | `ftp://192.168.1.10/pub/firmware.bin` |
| 本地文件 | `file:///absolute/path` | `file:///tmp/upgrade/firmware.bin` |

---

### 9. Upload - 上传文件

**用途**: 将设备文件上传到指定服务器，支持配置文件导出、日志上传等

**请求**:
```xml
<cwmp:Upload>
  <CommandKey>命令键</CommandKey>
  <FileType>文件类型</FileType>
  <URL>上传URL</URL>
  <Username>可选的用户名</Username>
  <Password>可选的密码</Password>
  <DelaySeconds>延迟执行秒数</DelaySeconds>
</cwmp:Upload>
```

**字段说明**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| CommandKey | string | 是 | 命令标识符，用于关联 TransferComplete 响应 |
| FileType | string | 是 | 文件类型，见下表 |
| URL | string | 是 | 上传地址，支持 HTTP/HTTPS |
| Username | string | 否 | HTTP Basic 认证用户名 |
| Password | string | 否 | HTTP Basic 认证密码 |
| DelaySeconds | unsignedInt | 否 | 延迟执行秒数，0 表示立即执行 |

**FileType 取值**:

| FileType | 说明 | 生成的文件 |
|----------|------|-----------|
| `1 Vendor Configuration File` | 厂商配置文件 | 当前配置快照 |
| `2 Vendor Log File` | 厂商日志文件 | 日志打包 (tar.gz) |
| `4 Vendor Log File` | 厂商日志文件(扩展) | 日志打包 (tar.gz) |
| `10 <OUI> Configuration File` | 厂商特定配置文件 | 完整配置导出 XML |
| `11 Parameter Model` | 参数模型定义 | 预生成的 datamodel.xml |
| `Tr069 Ssl Cert File` | TR069 SSL证书文件 | data/tr069_ca.crt |

**响应**:
```xml
<cwmp:UploadResponse>
  <Status>1</Status>  <!-- 0=已完成, 1=正在进行 -->
  <StartTime/>
  <CompleteTime/>
</cwmp:UploadResponse>
```

---

### 10. Reboot - 重启设备

**用途**: 重启CPE设备

**请求**:
```xml
<cwmp:Reboot>
  <CommandKey>命令键</CommandKey>
</cwmp:Reboot>
```

**响应**:
```xml
<cwmp:RebootResponse />
```

---

### 11. FactoryReset - 恢复出厂设置

**用途**: 将设备恢复到出厂默认配置

**请求**:
```xml
<cwmp:FactoryReset>
  <PreserveConfig>1</PreserveConfig>  <!-- 可选参数，默认=1 -->
</cwmp:FactoryReset>
```

**参数说明**:

| 参数 | 类型 | 必须 | 说明 |
|------|------|------|------|
| PreserveConfig | int | 否 | 是否保留部分配置。`1`=保留网络/IPSec等关键配置（默认）；`0`=完全恢复出厂设置 |

**保留的配置项** (当 PreserveConfig=1 时):
- 设备身份信息（序列号、OUI、产品类型等）
- 网管服务器配置（ACS URL、认证信息）
- IPSec/安全配置
- 网络接口配置（以太网、IP接口）
- 静态路由配置
- 接口绑定参数（S1C/S1U/X2/TR069等）
- DNS配置参数

**响应**:
```xml
<cwmp:FactoryResetResponse />
```

**示例**:

完全恢复出厂设置（不保留任何配置）:
```xml
<cwmp:FactoryReset>
  <PreserveConfig>0</PreserveConfig>
</cwmp:FactoryReset>
```

保留网络配置的恢复出厂设置（默认行为）:
```xml
<cwmp:FactoryReset>
  <PreserveConfig>1</PreserveConfig>
</cwmp:FactoryReset>
```

---

### 12. GetRPCMethods - 获取支持的RPC方法

**用途**: 查询设备支持的所有RPC方法

**请求**:
```xml
<cwmp:GetRPCMethods />
```

**响应**:
```xml
<cwmp:GetRPCMethodsResponse>
  <MethodList soap:arrayType="xsd:string[N]">
    <string>GetParameterValues</string>
    <string>SetParameterValues</string>
    <!-- 更多方法 -->
  </MethodList>
</cwmp:GetRPCMethodsResponse>
```

---

## 请求示例

### 示例1: 获取单个参数

```bash
curl -X POST http://192.168.1.100:7547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">req-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterValues>
      <ParameterNames>
        <string>Device.DeviceInfo.SerialNumber</string>
      </ParameterNames>
    </cwmp:GetParameterValues>
  </soap:Body>
</soap:Envelope>'
```

---

### 示例2: 获取多个参数

```bash
curl -X POST http://192.168.1.100:7547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">req-002</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterValues>
      <ParameterNames>
        <string>Device.DeviceInfo.SerialNumber</string>
        <string>Device.DeviceInfo.Manufacturer</string>
        <string>Device.DeviceInfo.ModelName</string>
        <string>Device.DeviceInfo.SoftwareVersion</string>
      </ParameterNames>
    </cwmp:GetParameterValues>
  </soap:Body>
</soap:Envelope>'
```

---

### 示例3: 设置参数值

```bash
curl -X POST http://192.168.1.100:7547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">req-003</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:SetParameterValues>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[1]">
        <ParameterValueStruct>
          <Name>Device.ManagementServer.PeriodicInformInterval</Name>
          <Value xsi:type="xsd:unsignedInt">3600</Value>
        </ParameterValueStruct>
      </ParameterList>
      <ParameterKey>lmt-update-001</ParameterKey>
    </cwmp:SetParameterValues>
  </soap:Body>
</soap:Envelope>'
```

---

### 示例4: 获取对象树结构

```bash
curl -X POST http://192.168.1.100:7547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">req-004</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterNames>
      <ParameterPath>Device.DeviceInfo.</ParameterPath>
      <NextLevel>true</NextLevel>
    </cwmp:GetParameterNames>
  </soap:Body>
</soap:Envelope>'
```

---

### 示例5: Python脚本调用

```python
#!/usr/bin/env python3
import requests

def lmt_get_parameter(device_ip, parameter_name):
    """通过LMT获取单个参数值"""

    url = f"http://{device_ip}:7547/lmt"

    soap_request = f'''<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">python-req-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterValues>
      <ParameterNames>
        <string>{parameter_name}</string>
      </ParameterNames>
    </cwmp:GetParameterValues>
  </soap:Body>
</soap:Envelope>'''

    headers = {
        'Content-Type': 'text/xml; charset=utf-8'
    }

    response = requests.post(url, data=soap_request, headers=headers)

    if response.status_code == 200:
        return response.text
    else:
        raise Exception(f"LMT请求失败: {response.status_code}")

# 使用示例
if __name__ == "__main__":
    device_ip = "192.168.1.100"
    param = "Device.DeviceInfo.SerialNumber"

    try:
        result = lmt_get_parameter(device_ip, param)
        print(f"参数 {param} 的值:")
        print(result)
    except Exception as e:
        print(f"错误: {e}")
```

---

### 示例6: 触发版本回退

**用途**: 通过设置 `X_COM_ROLLBACK_CONTROL` 参数为 1，触发系统版本回退到上一个稳定版本。

**参数说明**:
| 参数路径 | MIB映射 | 类型 | 值域 | 说明 |
|---------|---------|------|------|------|
| `Device.DeviceInfo.X_COM_ROLLBACK_CONTROL` | `FAP.0.SW_ROLLBACK_CONTROL` | unsignedInt | 0/1 | 1=触发回退 |

**curl 请求**:
```bash
curl -X POST http://192.168.1.100:7547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">rollback-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:SetParameterValues>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[1]">
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.X_COM_ROLLBACK_CONTROL</Name>
          <Value xsi:type="xsd:unsignedInt">1</Value>
        </ParameterValueStruct>
      </ParameterList>
      <ParameterKey>rollback_trigger_001</ParameterKey>
    </cwmp:SetParameterValues>
  </soap:Body>
</soap:Envelope>'
```

**成功响应**:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">0</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:SetParameterValuesResponse>
      <Status>0</Status>
    </cwmp:SetParameterValuesResponse>
  </soap:Body>
</soap:Envelope>
```

**内部处理流程**:
```
SetParameterValues 请求
    ↓
TR069 Worker 转换路径: Device.DeviceInfo.X_COM_ROLLBACK_CONTROL → FAP.0.SW_ROLLBACK_CONTROL
    ↓
Parameter Worker 设置参数值
    ↓
Device Worker 收到参数订阅回调 (handleSwActivateEnableChange)
    ↓
当值 == 1 时，调用 performSystemFallback("sys")
    ↓
SystemUpgradeManager 通过 Ubus 执行文件系统回退
    ↓
请求 Master 进程重启系统
```

**注意事项**:
- 回退操作会触发系统重启，请确保当前无重要业务运行
- 回退后系统将使用备份分区的软件版本
- 相关只读参数 `Device.DeviceInfo.X_COM_ROLLBACK_ENABLE` 可查询回退功能是否可用

**相关参数**:
| 参数路径 | 类型 | 访问 | 说明 |
|---------|------|------|------|
| `Device.DeviceInfo.X_COM_ROLLBACK_ENABLE` | unsignedInt | 只读 | 回退功能是否启用 (0/1) |
| `Device.DeviceInfo.X_COM_ROLLBACK_CONTROL` | unsignedInt | 读写 | 回退控制开关 (设为1触发) |
| `Device.DeviceInfo.SoftwareVersion` | string | 只读 | 当前软件版本 |

---

### 示例7: 固件升级 (Download)

**用途**: 通过 Download RPC 触发固件升级，支持远程下载或本地文件升级。

#### 7.1 远程 HTTP 下载升级

```bash
curl -X POST http://192.168.1.100:7547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">upgrade-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Download>
      <CommandKey>FW_UPGRADE_20250113</CommandKey>
      <FileType>1 Firmware Upgrade Image</FileType>
      <URL>http://192.168.1.10/firmware/oam_v2.0.0.bin</URL>
      <Username>admin</Username>
      <Password>password123</Password>
      <FileSize>52428800</FileSize>
      <TargetFileName>oam_v2.0.0.bin</TargetFileName>
      <DelaySeconds>0</DelaySeconds>
      <SuccessURL></SuccessURL>
      <FailureURL></FailureURL>
      <RawMode>0</RawMode>
    </cwmp:Download>
  </soap:Body>
</soap:Envelope>'
```

#### 7.2 本地文件升级

当升级包已通过其他方式（如 SCP、USB）传输到设备本地时，使用 `file:///` 协议（如 `file:///tmp/upload/xxx.bin`）：

```bash
curl -X POST http://192.168.1.100:7547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">upgrade-local-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Download>
      <CommandKey>FW_LOCAL_UPGRADE</CommandKey>
      <FileType>1 Firmware Upgrade Image</FileType>
      <URL>file:///tmp/upload/oam_v2.0.0.bin</URL>
      <Username></Username>
      <Password></Password>
      <FileSize>52428800</FileSize>
      <TargetFileName></TargetFileName>
      <DelaySeconds>0</DelaySeconds>
      <SuccessURL></SuccessURL>
      <FailureURL></FailureURL>
      <RawMode>0</RawMode>
    </cwmp:Download>
  </soap:Body>
</soap:Envelope>'
```

#### 7.3 不保配置升级（RawMode）

当需要清除所有配置，恢复出厂设置时，设置 `RawMode=1`：

```bash
curl -X POST http://192.168.1.100:7547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">upgrade-rawmode-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Download>
      <CommandKey>FW_RAWMODE_UPGRADE</CommandKey>
      <FileType>1 Firmware Upgrade Image</FileType>
      <URL>http://192.168.1.10/firmware/oam_v2.0.0.bin</URL>
      <Username>admin</Username>
      <Password>password123</Password>
      <FileSize>52428800</FileSize>
      <TargetFileName>oam_v2.0.0.bin</TargetFileName>
      <DelaySeconds>0</DelaySeconds>
      <SuccessURL></SuccessURL>
      <FailureURL></FailureURL>
      <RawMode>1</RawMode>
    </cwmp:Download>
  </soap:Body>
</soap:Envelope>'
```

> **警告**: `RawMode=1` 会在升级完成后清除所有用户配置，设备将恢复出厂默认设置。
> 请确保在执行前已备份重要配置。

**升级流程**:
```
Download 请求 (FileType = "1 Firmware Upgrade Image")
    ↓
TR069 Worker 创建下载任务 (TransferManager)
    ↓
下载完成后发送 DEVICE_UPGRADE_REQUEST 给 Device Worker
    ↓
Device Worker 调用 SystemUpgradeManager 执行升级
    ↓
升级成功后请求 Master 重启系统
    ↓
系统重启后使用新版本
```

**成功响应**:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">0</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:DownloadResponse>
      <Status>1</Status>
      <StartTime>0001-01-01T00:00:00Z</StartTime>
      <CompleteTime>0001-01-01T00:00:00Z</CompleteTime>
    </cwmp:DownloadResponse>
  </soap:Body>
</soap:Envelope>
```

**注意事项**:
- `FileType` 必须为 `1 Firmware Upgrade Image` 才会触发升级流程
- `FileSize` 应为实际文件大小，用于空间检查和进度计算
- 本地文件路径必须是绝对路径，使用 `file:///` 前缀（三个斜杠）
- 升级成功后系统会自动重启，请确保无重要业务运行
- `RawMode=0`（默认）：保配置升级，升级后保留原有配置
- `RawMode=1`：不保配置升级，升级后恢复出厂设置

---

### 示例8: 查询升级状态

**用途**: 通过查询升级状态参数，监控升级进度和结果。

```bash
curl -X POST http://192.168.1.100:7547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">status-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterValues>
      <ParameterNames>
        <string>Device.DeviceInfo.SwUpgrade.Stage</string>
        <string>Device.DeviceInfo.SwUpgrade.Status</string>
        <string>Device.DeviceInfo.SwUpgrade.FailureCause</string>
        <string>Device.DeviceInfo.SoftwareVersion</string>
      </ParameterNames>
    </cwmp:GetParameterValues>
  </soap:Body>
</soap:Envelope>'
```

**升级状态参数说明**:

| 参数路径 | MIB映射 | 类型 | 说明 |
|---------|---------|------|------|
| `Device.DeviceInfo.SwUpgrade.Stage` | `FAP.0.SW_UPGRADE_STAGE` | unsignedInt | 升级阶段 |
| `Device.DeviceInfo.SwUpgrade.Status` | `FAP.0.SW_UPGRADE_STATUS` | unsignedInt | 阶段结果 |
| `Device.DeviceInfo.SwUpgrade.FailureCause` | `FAP.0.SW_UPGRADE_FAILURE_CAUSE` | string | 失败原因 |
| `Device.DeviceInfo.SoftwareVersion` | `FAP.0.SOFTWARE_VERSION` | string | 当前软件版本 |

**Stage 阶段值**:

| 值 | 阶段 | 说明 |
|----|------|------|
| 0 | 空闲 | 无升级任务 |
| 1 | 初始化 | 升级任务创建 |
| 2 | 下载中 | 正在下载升级包 |
| 3 | 校验中 | 升级包校验 |
| 4 | 安装中 | 正在安装 |
| 5 | 完成 | 升级完成 |

**Status 状态值**:

| 值 | 状态 | 说明 |
|----|------|------|
| 0 | 未知 | 初始状态 |
| 1 | 成功 | 当前阶段成功 |
| 2 | 失败 | 当前阶段失败，查看 FailureCause |

**升级状态流转**:
```
正常升级流程:
Stage=1,Status=1 → Stage=2,Status=1 → Stage=3,Status=1 → Stage=4,Status=1 → Stage=5,Status=1
    (初始化)         (下载成功)         (校验成功)         (安装成功)         (升级完成)

下载失败场景:
Stage=2,Status=2,FailureCause="download failed (code 9001)"

安装失败场景:
Stage=4,Status=2,FailureCause="flash write error"
```

**响应示例** (升级成功后):
```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">0</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterValuesResponse>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[4]">
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SwUpgrade.Stage</Name>
          <Value xsi:type="xsd:unsignedInt">5</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SwUpgrade.Status</Name>
          <Value xsi:type="xsd:unsignedInt">1</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SwUpgrade.FailureCause</Name>
          <Value xsi:type="xsd:string"></Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SoftwareVersion</Name>
          <Value xsi:type="xsd:string">2.0.0.build123</Value>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:GetParameterValuesResponse>
  </soap:Body>
</soap:Envelope>
```

---

### 示例9: 配置文件导入 (Download)

**用途**: 通过 Download RPC 下载并导入配置文件（FileType 10），支持从 ACS 服务器下载或使用本地文件。

#### 9.1 配置文件格式说明

配置文件为 XML 格式，包含以下主要节点：

```xml
<?xml version="1.0" encoding="UTF-8"?>
<importConfigFile genrateTime="2026-01-26T12:00:00" networkType="LTE"
                  serialNumber="1202000588233HB0039" vendor="48BF74">
    <vendorSpecific>
        <!-- 厂商扩展（忽略） -->
    </vendorSpecific>
    <objectSpecific>
        <!-- 对象实例列表（用于验证） -->
    </objectSpecific>
    <mibParamers>
        <!-- MIB 参数（直接设置，无需路径转换） -->
        <mibParameter object="FAP.0.NR_CELL" instance="0"
                     parameterName="CellIdentity" value="12345"/>
    </mibParamers>
    <dataModelSpecific>
        <!-- TR-069 参数（需要路径转换） -->
        <config name="Device.ManagementServer.URL"
               value="http://acs.example.com/tr069"/>
    </dataModelSpecific>
</importConfigFile>
```

**XML 节点说明**:

| 节点 | 说明 |
|------|------|
| `importConfigFile` | 根元素，含设备元数据 |
| `genrateTime` | 配置文件生成时间 |
| `networkType` | 网络类型（如 LTE, NR） |
| `serialNumber` | 设备序列号 |
| `vendor` | 厂商 OUI |
| `objectSpecific` | 对象实例列表（用于验证） |
| `mibParamers` | MIB 参数列表（直接设置） |
| `dataModelSpecific` | TR-069 参数列表（需路径转换） |

#### 9.2 远程下载导入配置文件

```bash
curl -X POST http://192.168.1.100:7547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">config-import-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Download>
      <CommandKey>CONFIG_IMPORT_20260127</CommandKey>
      <FileType>10 48BF74 Configuration File</FileType>
      <URL>http://192.168.1.10/config/device_config.xml</URL>
      <Username>admin</Username>
      <Password>password123</Password>
      <FileSize>8192</FileSize>
      <TargetFileName>device_config.xml</TargetFileName>
      <DelaySeconds>0</DelaySeconds>
      <SuccessURL></SuccessURL>
      <FailureURL></FailureURL>
    </cwmp:Download>
  </soap:Body>
</soap:Envelope>'
```

#### 9.3 本地文件导入配置

当配置文件已通过其他方式（如 SCP、USB）传输到设备本地时，使用 `file:///` 协议直接导入（如 `file:///tmp/upload/config.xml`）：

```bash
curl -X POST http://192.168.1.100:7547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">config-local-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Download>
      <CommandKey>CONFIG_LOCAL_IMPORT</CommandKey>
      <FileType>10 48BF74 Configuration File</FileType>
      <URL>file:///tmp/upload/device_config.xml</URL>
      <Username></Username>
      <Password></Password>
      <FileSize>8192</FileSize>
      <TargetFileName></TargetFileName>
      <DelaySeconds>0</DelaySeconds>
      <SuccessURL></SuccessURL>
      <FailureURL></FailureURL>
    </cwmp:Download>
  </soap:Body>
</soap:Envelope>'
```

**字段说明**:

| 字段 | 说明 |
|------|------|
| `FileType` | 必须为 `10 <OUI> Configuration File` 格式，OUI 为厂商标识（如 48BF74） |
| `URL` | 远程下载使用 `http://` 或 `https://`；本地文件使用 `file:///` + 绝对路径 |
| `FileSize` | 文件大小（字节），本地文件可设为 0 |
| `TargetFileName` | 本地文件模式可留空 |

**成功响应**:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">0</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:DownloadResponse>
      <Status>1</Status>
      <StartTime/>
      <CompleteTime/>
    </cwmp:DownloadResponse>
  </soap:Body>
</soap:Envelope>
```

**内部处理流程**:
```
Download 请求 (FileType = "10 <OUI> Configuration File")
    ↓
TR069 Worker 创建下载任务 (TransferManager)
    ↓
[远程文件] 下载配置文件到本地
[本地文件] 直接使用本地路径
    ↓
下载完成后检测 FileType 为 10
    ↓
自动调用 ConfigFileImporter 导入配置
    ↓
进入批量导入模式 (BATCH_IMPORT_BEGIN)
    ↓
解析 XML 并设置参数:
  - mibParamers: 直接设置 MIB 参数
  - dataModelSpecific: TR Path 转换后设置
    ↓
退出批量导入模式 (BATCH_IMPORT_END)
    ↓
保存 NV 参数，发送变更通知
    ↓
触发 TransferComplete 事件
```

**注意事项**:
- `FileType` 必须以 `10 ` 开头且包含 `Configuration File` 才会触发自动导入
- 本地文件路径必须是绝对路径，使用 `file:///` 前缀（三个斜杠）
- 配置导入是原子操作：全部成功或全部回滚
- 导入过程中会进入批量导入模式，不会立即触发订阅通知，待导入完成后统一通知
- 如果导入失败，TransferComplete 会包含错误信息

**相关参数**:

导入的参数会自动更新，可通过 GetParameterValues 查询：
```bash
curl -X POST http://192.168.1.100:7547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">verify-import-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterValues>
      <ParameterNames>
        <string>Device.ManagementServer.URL</string>
      </ParameterNames>
    </cwmp:GetParameterValues>
  </soap:Body>
</soap:Envelope>'
```

---

### 示例10: 配置文件导出 (Upload)

**用途**: 通过 Upload RPC 导出设备配置文件并上传到指定服务器。

#### 10.1 导出配置文件到 HTTP 服务器

```bash
curl -X POST http://192.168.1.100:7547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">config-export-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Upload>
      <CommandKey>CONFIG_EXPORT_20260127</CommandKey>
      <FileType>10 48BF74 Configuration File</FileType>
      <URL>http://192.168.1.10/upload/config</URL>
      <Username>admin</Username>
      <Password>password123</Password>
      <DelaySeconds>0</DelaySeconds>
    </cwmp:Upload>
  </soap:Body>
</soap:Envelope>'
```

#### 10.2 LMT 本地日志导出

**用途**: LMT 专用的日志导出方式，日志文件生成到 LMT 指定的本地路径，LMT 客户端通过 SCP 等方式下载。

**接口特点**:
- **URL 格式**: 使用 `file:///` 协议指定目标路径（如 `file:///tmp/upgrade/logs.tar.gz`，3个斜杠符合 RFC 8089）
- **CollectLogType 字段**: 厂商扩展字段，用于指定要收集的日志类型
- **响应简化**: UploadResponse 不再返回 FilePath（LMT 从请求 URL 已知目标路径）

**日志类型说明**:

通过 `<CollectLogType>` 字段指定要收集的日志类型（多个类型用逗号分隔）：

| 日志类型 | 说明 | 包含内容 |
|----------|------|----------|
| `all` | 所有日志（默认） | 包含以下所有类型 |
| `oam` | OAM 日志 | `/run/targetoxm/logs/` 下的日志文件 |
| `event` | Event 事件数据 | `/log/flashlog/` 事件记录 |
| `pm` | PM 性能数据 | KPI、MR 文件 |
| `system` | 系统状态信息 | 内存、CPU、进程状态快照 |
| `gsm` | 协议栈日志 | GSM/LTE 协议栈日志 |
| `die` | 已有异常日志 | `/mnt/log/Log_*.tar.bz2` 异常日志包 |
| `phy` | PHY 层日志 | 物理层日志文件 |
| `var` | VAR 日志 | `/var/` 目录下的日志 |
| `gps` | GPS 日志 | GPS 定位相关日志 |
| `ru` | RU 日志 | 远端射频单元日志（自动收集所有在线 RU，可通过 `CollectRUDevices` 指定特定 RU） |

> **注意**: `core` 类型仅在 Master 崩溃时由系统自动收集（`--fault` 模式），LMT 无法主动收集 core 文件。

**RU 日志收集说明**：

RU 日志通过 O-RAN NETCONF `file-upload` RPC 触发，RU 设备将日志上传到 BBU 的 SFTP 服务器。

**默认行为**：
- `all` 类型或包含 `ru` 类型时，**自动收集所有在线 RU 日志**
- 无需显式指定 `CollectRUDevices`，系统会自动触发
- 如果没有 RU 在线，会立即跳过（不影响其他日志收集）

**CollectRUDevices 字段**（可选，用于指定特定 RU）：

| 值 | 说明 |
|----|------|
| `all` | 收集所有在线 RU 日志（默认行为，可省略） |
| `SN1,SN2` | 只收集指定 RU 日志（逗号分隔 SN） |
| 空/不填 | 如果 logTypes 包含 `all` 或 `ru`，自动设为 `all` |

**不收集 RU 日志的方式**：
- 明确指定 `CollectLogType` 为不包含 `ru` 的类型，如 `oam,gsm,pm`

> **详细设计**: 参见 `docs/implementation/RU_LOG_COLLECTION_ENHANCEMENT_PLAN.md`

**请求示例（收集所有日志，含 RU）**:

```bash
curl -X POST http://127.0.0.1:17547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">log-lmt-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Upload>
      <CommandKey>LOG_LMT_20260210</CommandKey>
      <FileType>2 Vendor Log File</FileType>
      <URL>file:///tmp/upgrade/all_logs.tar.gz</URL>
      <Username></Username>
      <Password></Password>
      <DelaySeconds>0</DelaySeconds>
      <CollectLogType>all</CollectLogType>
    </cwmp:Upload>
  </soap:Body>
</soap:Envelope>'
```

**请求示例（指定日志类型 - OAM + GSM）**:

```bash
curl -X POST http://127.0.0.1:17547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">log-lmt-002</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Upload>
      <CommandKey>LOG_OAM_GSM</CommandKey>
      <FileType>2 Vendor Log File</FileType>
      <URL>file:///tmp/upgrade/oam_gsm_logs.tar.gz</URL>
      <Username></Username>
      <Password></Password>
      <DelaySeconds>0</DelaySeconds>
      <CollectLogType>oam,gsm</CollectLogType>
    </cwmp:Upload>
  </soap:Body>
</soap:Envelope>'
```

**请求示例（收集异常日志包）**:

收集 `/mnt/log/` 下已有的异常日志包（`Log_*.tar.bz2`）：

```bash
curl -X POST http://127.0.0.1:17547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">log-die-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Upload>
      <CommandKey>LOG_DIE_COLLECT</CommandKey>
      <FileType>2 Vendor Log File</FileType>
      <URL>file:///tmp/upgrade/die_logs.tar.gz</URL>
      <Username></Username>
      <Password></Password>
      <DelaySeconds>0</DelaySeconds>
      <CollectLogType>die</CollectLogType>
    </cwmp:Upload>
  </soap:Body>
</soap:Envelope>'
```

**请求示例（只收集 RU 日志）**:

```bash
curl -X POST http://127.0.0.1:17547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">log-ru-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Upload>
      <CommandKey>LOG_RU_ONLY</CommandKey>
      <FileType>2 Vendor Log File</FileType>
      <URL>file:///tmp/upgrade/ru_logs.tar.gz</URL>
      <Username></Username>
      <Password></Password>
      <DelaySeconds>0</DelaySeconds>
      <CollectLogType>ru</CollectLogType>
    </cwmp:Upload>
  </soap:Body>
</soap:Envelope>'
```

**请求示例（收集 OAM + 指定 RU 日志）**:

```bash
curl -X POST http://127.0.0.1:17547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">log-ru-002</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Upload>
      <CommandKey>LOG_OAM_SPECIFIC_RU</CommandKey>
      <FileType>2 Vendor Log File</FileType>
      <URL>file:///tmp/upgrade/oam_ru_logs.tar.gz</URL>
      <Username></Username>
      <Password></Password>
      <DelaySeconds>0</DelaySeconds>
      <CollectLogType>oam</CollectLogType>
      <CollectRUDevices>RU001,RU002</CollectRUDevices>
    </cwmp:Upload>
  </soap:Body>
</soap:Envelope>'
```

**字段说明**:

| 字段 | 说明 |
|------|------|
| `URL` | 使用 `file:///` 协议指定日志最终存放路径（3个斜杠） |
| `CollectLogType` | 厂商扩展字段，指定要收集的日志类型（逗号分隔）。**默认值**: 如果不填或为空，默认为 `all`（收集所有日志，包含 RU） |
| `CollectRUDevices` | 可选，RU 设备列表。`all` = 所有在线 RU，`SN1,SN2` = 指定 RU，空 = 当 logTypes 为 `all` 或包含 `ru` 时自动收集全部 RU |

**LMT 响应**:

LMT 请求的响应不再包含 FilePath 字段（目标路径已在请求 URL 中指定）：

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">0</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:UploadResponse>
      <Status>1</Status>
      <StartTime/>
      <CompleteTime/>
    </cwmp:UploadResponse>
  </soap:Body>
</soap:Envelope>
```

**响应字段说明**:

| 字段 | 说明 |
|------|------|
| `Status` | `1` = 任务已接受，异步执行中 |
| 无 FilePath | LMT 从请求 URL 已知目标路径 |

**查询任务状态（REST API）**:

LMT 支持通过 REST API 查询日志收集任务的状态：

```bash
# 查询任务状态
curl http://127.0.0.1:17547/lmt/task/LOG_LMT_20260210
```

**任务状态响应（JSON）**:

```json
{
  "commandKey": "LOG_LMT_20260210",
  "status": "Completed",
  "filePath": "/tmp/upgrade/all_logs.tar.gz",
  "errorCode": 0,
  "errorMessage": ""
}
```

**任务状态值**:

| status | 说明 |
|--------|------|
| `Pending` | 任务等待执行 |
| `InProgress` | 任务正在执行 |
| `Completed` | 任务已完成，可下载文件 |
| `Failed` | 任务失败，查看 errorMessage |

**下载日志文件**:

任务完成后，从 LMT 指定的目标路径下载日志文件：

```bash
# 使用 SCP 下载（路径来自请求 URL）
scp root@192.168.1.100:/tmp/upgrade/all_logs.tar.gz ./

# 或使用 SFTP
sftp root@192.168.1.100:/tmp/upgrade/
```

**内部处理流程**:
```
Upload 请求 (FileType = "2 Vendor Log File")
    ↓
TR069 Worker 检测到 LMT 请求
    ↓
从 URL 提取目标路径（file:/// 协议）
    ↓
从 CollectLogType 字段获取日志类型
    ↓
判断是否需要收集 RU 日志（logTypes 为 all 或包含 ru）
    ↓ 是
IPC 通知 RUEU Worker 触发 RU 日志上传（同步等待，最长 90 秒）
    ↓ RU 日志上传到 /tmp/RRULogs/
创建异步日志收集任务
    ↓
调用 collect-logs.sh --output <temp_dir> --type <types>
    ↓
脚本收集指定类型的日志文件（含 /tmp/RRULogs/ 中的 RU 日志）
    ↓
打包为 tar.gz 文件，移动到 LMT 指定的目标路径
    ↓
更新任务状态为 Completed
    ↓
LMT 客户端通过 REST API 查询状态并下载文件
```

**退出码说明**:

日志收集脚本可能返回以下退出码：

| 退出码 | 说明 | 处理建议 |
|--------|------|----------|
| `0` | 成功 | 文件已生成，可下载 |
| `100` | 并发冲突 | 另一个日志收集任务正在运行，稍后重试 |
| `101` | 磁盘空间不足 | 清理磁盘空间后重试 |
| `102` | 没有找到日志 | die 模式专用，/mnt/log/ 下没有异常日志包 |

---

#### 10.3 导出配置文件

配置文件导出使用 `FileType = "10 <OUI> Configuration File"`：

```bash
curl -X POST http://192.168.1.100:7547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">config-export-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Upload>
      <CommandKey>CONFIG_EXPORT_20260127</CommandKey>
      <FileType>10 48BF74 Configuration File</FileType>
      <URL>http://192.168.1.10/upload/config</URL>
      <Username>admin</Username>
      <Password>password123</Password>
      <DelaySeconds>0</DelaySeconds>
    </cwmp:Upload>
  </soap:Body>
</soap:Envelope>'
```

**导出的配置文件格式**:

配置文件为 XML 格式，与导入格式相同：
- `importConfigFile` 根元素（含设备元数据）
- `objectSpecific` 有数据的对象实例列表
- `dataModelSpecific` TR-069 参数（有 TR Path 映射）
- `mibParamers` MIB 参数（无 TR Path 映射）

**注意事项**:
- `FileType` 必须为 `10 <OUI> Configuration File` 格式，OUI 必须与设备 OUI 匹配
- 配置文件先生成到本地 `logs/` 目录，然后上传到指定 URL
- 上传完成后会触发 TransferComplete 事件

---

#### 10.4 导出参数模型定义

参数模型导出使用 `FileType = "11 Parameter Model"`，用于获取设备完整的参数模型定义（XML 格式），包含所有 TR-069 参数和对象的元数据信息。

**LMT 本地导出**（使用 `file:///` URL，文件直接保存到本地路径）:

```bash
curl -X POST http://192.168.1.100:17547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">datamodel-export-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Upload>
      <CommandKey>DATAMODEL_EXPORT_20260331</CommandKey>
      <FileType>11 Parameter Model</FileType>
      <URL>file:///tmp/datamodel.xml</URL>
      <Username></Username>
      <Password></Password>
      <DelaySeconds>0</DelaySeconds>
    </cwmp:Upload>
  </soap:Body>
</soap:Envelope>'
```

**OMC 远程上传**（使用 HTTP URL，文件上传到远端服务器）:

```bash
curl -X POST http://192.168.1.100:17547/lmt \
  -H "Content-Type: text/xml; charset=utf-8" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">datamodel-export-002</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Upload>
      <CommandKey>DATAMODEL_EXPORT_20260331</CommandKey>
      <FileType>11 Parameter Model</FileType>
      <URL>http://192.168.1.10/upload/datamodel</URL>
      <Username>admin</Username>
      <Password>password123</Password>
      <DelaySeconds>0</DelaySeconds>
    </cwmp:Upload>
  </soap:Body>
</soap:Envelope>'
```

**导出的参数模型文件说明**:

参数模型文件 (`datamodel.xml`) 为编译时预生成的 XML 文件，包含设备支持的所有 TR-069 参数和对象定义：

| 内容 | 说明 |
|------|------|
| 对象定义 | ~142 个 TR-069 对象 |
| 参数定义 | ~1,954 个参数 |
| 文件大小 | ~437-484 KB |

每个参数/对象条目包含以下元数据：
- **参数路径**: TR-069 完整路径（如 `Device.ManagementServer.URL`）
- **访问类型**: `READ_ONLY` 或 `READ_WRITE`
- **数据类型**: `STRING`、`INT`、`U_INT`、`BOOLEAN`、`DATE_TIME`、`LONG`、`U_LONG` 等
- **取值约束**: 最小值/最大值（适用时）
- **通知类型**: `PASSIVE`、`ACTIVE`、`INSTANT`、`NONE` 等
- **默认值**: 参数出厂默认值
- **变更生效策略**: `Immediate`、`OnReboot`、`OnEnable`

**注意事项**:
- FileType 11 **仅支持 Upload（上传/导出）**，不支持 Download
- 参数模型文件在编译/打包时预生成，运行时零开销
- 文件源路径为 `$OAM_ROOT/data/datamodel.xml`
- LMT 模式下（`file:///` URL），文件会复制到指定本地路径
- OMC 模式下（HTTP URL），返回预生成文件路径，由传输模块上传

---

## 错误处理

### SOAP Fault代码

LMT使用标准TR069 Fault代码：

| Fault代码 | 说明 | 解决方法 |
|-----------|------|----------|
| 9000 | 方法不支持 | 检查RPC方法名是否正确 |
| 9001 | 请求被拒绝 | 检查参数访问权限 |
| 9002 | 内部错误 | 检查设备日志，可能是系统错误 |
| 9003 | 无效参数 | 检查参数名、类型、值是否正确 |
| 9004 | 资源超限 | 减少请求频率或参数数量 |
| 9005 | 无效参数名 | 参数路径不存在 |
| 9006 | 无效参数类型 | 参数类型不匹配 |
| 9007 | 无效参数值 | 参数值超出范围或格式错误 |
| 9008 | 非可写参数 | 尝试修改只读参数 |

### 错误响应示例

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Body>
    <soap:Fault>
      <faultcode>Client</faultcode>
      <faultstring>CWMP fault</faultstring>
      <detail>
        <cwmp:Fault>
          <FaultCode>9005</FaultCode>
          <FaultString>Invalid parameter name: Device.Invalid.Parameter</FaultString>
        </cwmp:Fault>
      </detail>
    </soap:Fault>
  </soap:Body>
</soap:Envelope>
```

### 常见错误场景

#### 1. 连接被拒绝
```
Error: Connection refused
```
**原因**: TR069 Worker未运行或端口7547未监听
**解决**: 检查Worker进程状态，确认端口正常监听

#### 2. 格式错误的XML
```
SOAP Fault 9003: Failed to parse SOAP message
```
**原因**: XML格式不正确
**解决**: 验证XML语法，确保所有标签闭合

#### 3. 参数不存在
```
SOAP Fault 9005: Invalid parameter name
```
**原因**: 指定的参数路径不存在
**解决**: 使用GetParameterNames先查询可用参数

#### 4. 权限错误
```
SOAP Fault 9008: Attempt to set a non-writable parameter
```
**原因**: 尝试修改只读参数
**解决**: 使用GetParameterNames检查参数的Writable属性

---

## 性能考虑

### 会话生命周期

```
┌──────────────────────────────────────────────────────┐
│ LMT请求生命周期 (典型: < 100ms)                      │
├──────────────────────────────────────────────────────┤
│ 1. 接收HTTP请求           [~1ms]                     │
│ 2. 路径路由到LMT          [<1ms]                     │
│ 3. 创建临时会话           [~2ms]                     │
│ 4. 解析SOAP消息           [~5ms]                     │
│ 5. 调用RPC处理器          [~10-50ms, 取决于操作]     │
│ 6. 生成SOAP响应           [~5ms]                     │
│ 7. 发送HTTP响应           [~1ms]                     │
│ 8. 清理临时会话           [~2ms]                     │
└──────────────────────────────────────────────────────┘
```

### 性能指标

| 指标 | 典型值 | 最大值 |
|------|--------|--------|
| **单次请求延迟** | 20-100ms | 500ms |
| **并发请求数** | 10+ | 受限于系统资源 |
| **会话超时** | 60秒 | 固定 |
| **最大消息大小** | - | 1MB |

### 优化建议

1. **批量查询**: 在单个GetParameterValues中查询多个参数，而不是发送多个请求
   ```xml
   <!-- 好: 一次查询多个参数 -->
   <ParameterNames>
     <string>Device.DeviceInfo.SerialNumber</string>
     <string>Device.DeviceInfo.Manufacturer</string>
     <string>Device.DeviceInfo.ModelName</string>
   </ParameterNames>

   <!-- 差: 多次查询单个参数 -->
   <!-- 发送3次独立请求 -->
   ```

2. **使用NextLevel参数**: GetParameterNames时使用NextLevel=true减少返回数据量
   ```xml
   <cwmp:GetParameterNames>
     <ParameterPath>Device.WiFi.</ParameterPath>
     <NextLevel>true</NextLevel>  <!-- 只返回下一层 -->
   </cwmp:GetParameterNames>
   ```

3. **连接复用**: 使用HTTP/1.1但不依赖Keep-Alive（LMT自动关闭连接）

4. **控制请求频率**: 避免过于频繁的轮询，建议间隔至少1秒

### 并发处理

LMT支持并发请求，每个请求使用独立的临时会话：

```
时间轴:
t0: Request-1 开始  ━━━━━━━━━━━━━━━━━━━━┓
t1: Request-2 开始       ━━━━━━━━━━━━━━━┓
t2: Request-3 开始            ━━━━━━━━━━┓
t3: Request-1 完成  ━━━━━━━━━━━━━━━━━━━━┛
t4: Request-2 完成       ━━━━━━━━━━━━━━━┛
t5: Request-3 完成            ━━━━━━━━━━┛

会话隔离: 每个请求独立的临时会话ID
```

---

## 安全说明

### 安全模型

⚠️ **LMT设计用于可信内网环境，不提供身份认证**

```
┌─────────────────────────────────────────┐
│  安全边界                               │
├─────────────────────────────────────────┤
│                                         │
│  [可信内网]                             │
│   ├─── 管理员工作站                     │
│   ├─── 本地管理工具                     │
│   └─── CPE设备 (LMT: 端口7547/lmt)     │
│                                         │
│  ─────────────────────────  防火墙      │
│                                         │
│  [公网/不可信网络]                      │
│   └─── OMC (认证访问: 端口7547/acs)    │
│                                         │
└─────────────────────────────────────────┘
```

### 安全建议

1. **网络隔离**:
   - ✅ 仅在内网中使用LMT
   - ✅ 使用防火墙阻止外网访问端口7547
   - ❌ 不要在公网暴露LMT端点

2. **访问控制**:
   - 在防火墙或路由器上限制访问源IP
   - 考虑使用VPN或专用管理网络

3. **审计日志**:
   - LMT请求会记录在TR069 Worker日志中
   - 监控异常的LMT访问模式

4. **生产环境**:
   - 如果不需要LMT功能，可以考虑禁用（修改代码移除LMT路由）
   - 或使用防火墙规则阻止`/lmt`路径访问

### OMC路径安全

OMC路径（`/acs`, `/`）仍然需要HTTP Basic认证：

```
POST /acs HTTP/1.1
Authorization: Basic <base64(username:password)>
```

这与LMT形成对比：
- **LMT (`/lmt`)**: 无认证，仅限内网
- **OMC (`/acs`, `/`)**: 需要认证，支持公网访问

---

## 故障排查

### 检查服务状态

1. **验证TR069 Worker运行状态**:
   ```bash
   ps aux | grep oam_tr069_worker
   ```

2. **检查端口监听**:
   ```bash
   netstat -an | grep 7547
   # 或
   lsof -i :7547
   ```

   应该看到:
   ```
   oam_tr069  <PID>  user  12u  IPv4  ...  TCP *:7547 (LISTEN)
   ```

3. **查看Worker日志**:
   ```bash
   tail -f logs/tr069_worker_omc.log | grep -i "lmt\|connection.*request"
   ```

   应该看到:
   ```
   [INFO] LMTProtocolAdapter initialized successfully
   [INFO] ConnectionRequestServer started on port 7547
   [INFO] LMT callback configured for CONNECTION_REQUEST server
   ```

### 测试连接

```bash
# 测试端口是否可达
nc -zv 192.168.1.100 7547

# 测试HTTP响应
curl -v http://192.168.1.100:7547/lmt
```

### 调试请求

启用详细日志查看LMT请求处理过程：

```bash
# 查看实时LMT请求日志
tail -f logs/tr069_worker_omc.log | grep -E "LMT|lmt-"
```

日志示例：
```
[INFO] LMT request received from client: 192.168.1.50
[DEBUG] Created temporary LMT session: lmt-1732270595123-4567
[DEBUG] Parsed RPC method: GetParameterValues
[INFO] LMT request processed successfully
[DEBUG] LMT session lmt-1732270595123-4567 cleaned up successfully
```

### 常见问题

**Q: 为什么我的请求超时？**
A: 检查防火墙设置，确保端口7547开放。检查设备是否在同一网络。

**Q: 返回401 Unauthorized**
A: 你访问的是OMC路径（`/`或`/acs`），请使用`/lmt`路径。

**Q: 返回404 Not Found**
A: 确认URL路径为`/lmt`，检查拼写。

**Q: SOAP解析错误**
A: 验证XML格式，确保使用UTF-8编码，检查命名空间声明。

---

## 附录

### A. 完整的SOAP命名空间

```xml
<soap:Envelope
  xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
  xmlns:soap-enc="http://schemas.xmlsoap.org/soap/encoding/"
  xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
  xmlns:xsd="http://www.w3.org/2001/XMLSchema"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
```

### B. TR069数据类型映射

| TR069类型 | XSD类型 | 示例 |
|-----------|---------|------|
| string | xsd:string | `"Hello"` |
| int | xsd:int | `-123` |
| unsignedInt | xsd:unsignedInt | `456` |
| boolean | xsd:boolean | `true` |
| dateTime | xsd:dateTime | `2025-11-22T16:00:00Z` |
| base64 | xsd:base64Binary | `SGVsbG8=` |

### C. 参考资源

- **TR-069协议规范**: DSL Forum TR-069 Amendment 5
- **数据模型**: TR-181 Device:2 Data Model
- **实现计划**: `docs/tr069/TR069_LMT_SUPPORT_IMPLEMENTATION_PLAN.md`
- **API参考**: `docs/tr069/TR069_LMT_API_REFERENCE.md`

### D. 联系与支持

如遇问题，请查看：
1. Worker日志: `logs/tr069_worker_omc.log`
2. 单元测试: `tests/tr069/test_lmt_protocol_adapter.cpp`
3. 集成测试: `tests/integration/test_lmt_omc_concurrent.cpp`

---

**文档版本**: 1.6
**最后更新**: 2026-02-10
**适用版本**: OAM System v1.0.0+
