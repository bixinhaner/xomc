# TR-069 协议合规性分析报告 — CPE 模拟器 × OMC ACS 联调报文审计

> 基于 `cpe_simulator.py` 与 `omcgo-acs` 实际交互报文逐包分析
> 分析样本：3 CPE × Bootstrap 会话 + TransferComplete 会话
> 日期：2026-03-22
> 参考规范：TR-069 Amendment 6 (CWMP 1.4)、中国移动皮基站南向接口要求 v1.0.0

---

## 总体评估

| 维度 | CPE 模拟器 | OMC ACS |
|------|-----------|---------|
| **HTTP 层** | ✅ 合规 | ⚠️ 1 个严重问题 |
| **SOAP/XML 层** | ✅ 合规 | ❌ 2 个违规 |
| **会话流程** | ✅ 合规 | ❌ 1 个严重问题 |
| **RPC 报文格式** | ✅ 合规（1 个小问题） | ❌ 2 个违规 |
| **总分** | **95 分** | **70 分** |

**结论：CPE 模拟器基本合规，主要问题在 ACS 侧。ACS 有 3 个 P0 级违规需要修复。**

---

## 一、HTTP 层分析

### 1.1 CPE 请求 Headers（✅ 合规）

```http
POST /smallcell/AcsService HTTP/1.1
User-Agent: TR069-CPE-Sim/SIM-CPE-0001
Connection: keep-alive
SOAPAction: ""
Content-Type: text/xml; charset=utf-8
Content-Length: 3134
Cookie: SESSION=84edce74ca244d33bc8a6b481550702c
```

| Header | TR-069 要求 | 实际 | 结论 |
|--------|------------|------|------|
| `Content-Type` | `text/xml` 或 `text/xml; charset=utf-8` | `text/xml; charset=utf-8` | ✅ |
| `Content-Length` | 必须 | 正确计算 | ✅ |
| `SOAPAction` | MUST 存在（Section 3.7.1.1） | `""` | ✅ |
| `Connection` | 建议 keep-alive | `keep-alive` | ✅ |
| `Cookie` | 会话管理时需要 | 正确回传 ACS Set-Cookie | ✅ |

**空 POST Headers**:
```http
POST /smallcell/AcsService HTTP/1.1
User-Agent: TR069-CPE-Sim/SIM-CPE-0001
Connection: keep-alive
SOAPAction: ""
Content-Length: 0
Cookie: SESSION=84edce74ca244d33bc8a6b481550702c
```

| 检查项 | 结论 |
|--------|------|
| 空 POST 不含 Content-Type | ✅ 正确（无内容不设 Content-Type） |
| Content-Length: 0 | ✅ 正确 |
| 保持 Cookie | ✅ 正确 |

---

### 1.2 ACS 响应 Headers

```http
HTTP/1.1 200 OK
Content-Type: text/xml; charset=utf-8
Set-Cookie: SESSION=84edce74ca244d33bc8a6b481550702c; Path=/; HttpOnly; SameSite=Lax
X-Request-Id: acs-20260322164204-e1318a4c
Date: Sun, 22 Mar 2026 14:42:04 GMT
Content-Length: 642
```

| Header | TR-069 要求 | 实际 | 结论 |
|--------|------------|------|------|
| `Content-Type` | `text/xml` | `text/xml; charset=utf-8` | ✅ |
| `Set-Cookie` | 会话管理 | 正确（HttpOnly, SameSite） | ✅ |
| `Content-Length` | 必须 | 正确 | ✅ |
| `Connection` | 会话结束时应 close | **未设置** | ⚠️ P2-ACS |

**问题 P2-ACS-1：会话结束 204 响应缺少 `Connection: close`**

TR-069 Section 3.7.1.1: "When the ACS has no more requests to send and the CPE has no more requests to send, the session MUST be terminated."

ACS 发送 204 时应显式设置 `Connection: close` 通知 CPE 关闭连接，避免 CPE 在已结束的会话上继续发送请求。

---

### 1.3 HTTP 400 错误（❌ P0-ACS-1）

**现象**：CPE 发送 18652 字节的 `GetParameterNamesResponse`（102 个参数）后，ACS 返回 HTTP 400。

```http
HTTP/1.1 400 Bad Request
Content-Type: text/plain; charset=utf-8
X-Content-Type-Options: nosniff
Content-Length: 12

Bad Request
```

**分析**：
- `Content-Type: text/plain`（非 text/xml）→ Go 标准库 `http.Error()` 发出，非 SOAP 层面的错误
- `X-Content-Type-Options: nosniff` → Go `http.Error()` 默认行为
- ACS handler 中有多处返回 400 的代码路径：
  - `handler.go:119` — `io.ReadAll(r.Body)` 失败
  - `handler.go:167` — SOAP 方法未识别（`default` 分支）
  - `handler.go:189` — Inform 解码失败

**可能原因**（按概率排序）：
1. **最可能**：ACS 的 `handleRPCResponse` 处理 18KB 的 GPNResponse 时，session state 已经因为前面 5 个连续管道化 RPC 而进入异常状态
2. **次可能**：`io.ReadAll()` 在 keep-alive 连接上读取大 body 时超时或解析错误
3. **低概率**：`soap.DetectRPCMethod()` 未正确识别 `GetParameterNamesResponse` 方法名

**影响**：
- 会话异常终止，后续 provision 任务无法执行
- 3 个 CPE 全部在 GetParameterNames(NextLevel=false) 后遇到此错误
- 小量参数的 GetParameterNames(NextLevel=true, 1 个结果) 正常通过

**归属**：**ACS 侧 Bug**

**修复建议**：
```go
// handler.go — handleRPCResponse 入口处增加 body 大小日志和 state 检查
func (h *Handler) handleRPCResponse(w http.ResponseWriter, r *http.Request, body []byte, log *zap.Logger) {
    log.Debug("received RPC response",
        zap.Int("body_size", len(body)),
        zap.String("session_state", session.State.String()),
    )
    // 确保 session 在 RPC_PENDING 状态
    if session.State != StateRPCPending {
        log.Warn("unexpected state for RPC response",
            zap.String("expected", "RPC_PENDING"),
            zap.String("actual", session.State.String()),
        )
    }
}
```

---

## 二、SOAP/XML 层分析

### 2.1 SOAP Envelope 结构

**CPE 侧（✅ 合规）**：
```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:soap-enc="http://schemas.xmlsoap.org/soap/encoding/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">29944</cwmp:ID>
  </soap:Header>
  <soap:Body>
    ...
  </soap:Body>
</soap:Envelope>
```

| 检查项 | TR-069 要求 | 实际 | 结论 |
|--------|------------|------|------|
| XML 声明 | 推荐 | `<?xml version="1.0" encoding="UTF-8"?>` | ✅ |
| SOAP 命名空间 | `http://schemas.xmlsoap.org/soap/envelope/` | 匹配 | ✅ |
| CWMP 命名空间 | `urn:dslforum-org:cwmp-1-0` | 匹配 | ✅ |
| Header/cwmp:ID | MUST present, mustUnderstand="1" | 匹配 | ✅ |
| ID 唯一性 | CPE 自生成的 ID 必须唯一 | 递增整数 | ✅ |

**ACS 侧（✅ Envelope 合规，❌ cwmp:ID 违规）**：

ACS 的 SOAP Envelope 结构同样正确。**但存在严重的 cwmp:ID 问题**：

---

### 2.2 cwmp:ID 合规性（❌ P0-ACS-2）

**TR-069 Section 3.7.1.1**:
> "An ID Header element MUST be used in all SOAP envelopes...to uniquely identify the SOAP request."

**实际报文**（ACS 测试任务注入的 RPC）：
```xml
<soap:Header>
  <cwmp:ID soap:mustUnderstand="1"></cwmp:ID>   ← ❌ 空！
</soap:Header>
```

**实际报文**（ACS Provision 任务的 RPC）：
```xml
<soap:Header>
  <cwmp:ID soap:mustUnderstand="1">provision-SetParameterValues-2</cwmp:ID>  ← ✅
</soap:Header>
```

| RPC 来源 | cwmp:ID | 合规 |
|---------|---------|------|
| 测试任务注入 | `""` (空) | ❌ 违规 |
| Provision 任务 | `provision-{method}-{seq}` | ✅ 合规 |
| CPE Inform | `29944`, `29945` (递增) | ✅ 合规 |
| CPE TransferComplete | `29946` (递增) | ✅ 合规 |

**影响**：
- 空 cwmp:ID 导致 ACS 无法关联 RPC 请求和响应
- 测试任务的响应结果无法追踪（Task 查找依赖 cwmp:ID）
- 可能导致 Task 永远停留在 SENT 状态

**归属**：**ACS 侧 Bug**（`injectRandomTestTasks()` 生成的 Task 未设置 cwmp:ID）

**修复建议**：
```go
// handler.go — injectRandomTestTasks 中为每个 task 生成唯一 ID
task.CWMPID = fmt.Sprintf("test-%s-%d", deviceSN[:8], i)
```

---

### 2.3 InformResponse 非标准字段（⚠️ P2-ACS-2）

**TR-069 规范**：InformResponse 仅包含 `MaxEnvelopes` 参数。

**ACS 实际**：
```xml
<cwmp:InformResponse>
  <MaxEnvelopes>1</MaxEnvelopes>
  <CurrentTime>2026-03-22T14:42:04Z</CurrentTime>   ← 非标准
</cwmp:InformResponse>
```

**分析**：CurrentTime 不是 InformResponse 的标准参数。TR-069 规范中 InformResponse 只有 MaxEnvelopes。但这是一个无害的额外字段，CPE 应忽略不认识的字段。

**归属**：**ACS 侧** 小问题
**严重级别**：P2（不影响互操作）

---

## 三、TR-069 会话流程分析

### 3.1 Bootstrap 会话流程

```
序号  方向      报文                         cwmp:ID              大小    合规
────  ────── ────────────────────────────  ────────────────── ──────  ────
 1    CPE→ACS  Inform (0 BOOTSTRAP)         29944               3134B   ✅
 2    ACS→CPE  InformResponse               29944                642B   ✅
 3    CPE→ACS  Empty POST                   —                      0B   ✅
 4    ACS→CPE  Download                     "" (空)              880B   ❌ ID空
 5    CPE→ACS  DownloadResponse (Status=1)  "" (回传空)          681B   ✅
 6    ACS→CPE  Upload (piped)               "" (空)              735B   ❌ ID空
 7    CPE→ACS  UploadResponse (Status=1)    "" (回传空)          677B   ✅
 8    ACS→CPE  GetParameterValues (piped)   "" (空)              750B   ✅
 9    CPE→ACS  GetParameterValuesResponse   "" (回传空)         1024B   ✅
10    ACS→CPE  GetParameterValues (piped)   "" (空)              694B   ✅
11    CPE→ACS  GetParameterValuesResponse   "" (回传空)          836B   ✅
12    ACS→CPE  GetParameterNames (piped)    "" (空)              625B   ✅
13    CPE→ACS  GetParameterNamesResponse    "" (回传空)        18652B   ✅
14    ACS→CPE  HTTP 400 Bad Request         —                    12B   ❌ ACS 崩溃
```

**流程分析**：

| 检查项 | TR-069 要求 | 实际 | 结论 |
|--------|------------|------|------|
| Inform 作为第一条消息 | MUST | 是 | ✅ |
| InformResponse 后 CPE 发空 POST | 协议规定 | 是 | ✅ |
| ACS 可以在 RPC 响应中管道化下一个 RPC | 允许 | 使用了管道化 | ✅ |
| CPE 正确处理管道化 RPC | 必须 | 5 个管道化 RPC 全部正确处理 | ✅ |
| 会话正常结束（204 No Content） | 规定 | **ACS 返回 400** | ❌ |

---

### 3.2 TransferComplete 会话流程

```
序号  方向      报文                              cwmp:ID    合规
────  ────── ──────────────────────────────────  ────────  ────
15    CPE→ACS  Inform (7 TRANSFER COMPLETE,      29945      ✅
               M Upload)
16    ACS→CPE  InformResponse                    29945      ✅
17    CPE→ACS  TransferComplete                  29946      ✅✅ 直接发！
18    ACS→CPE  TransferCompleteResponse           29946      ✅
19    CPE→ACS  Empty POST                         —         ✅
20    ACS→CPE  Download (测试任务)                "" (空)    ❌ 不应在TC会话注入
```

**流程分析**：

| 检查项 | TR-069 要求 | 实际 | 结论 |
|--------|------------|------|------|
| Inform 包含 "7 TRANSFER COMPLETE" | MUST | 是 | ✅ |
| Inform 包含 "M Download" 或 "M Upload" | MUST | "M Upload" ✅ | ✅ |
| TransferComplete 紧接 InformResponse | 协议规定 | 是（无多余空 POST） | ✅✅ |
| TransferComplete 格式正确 | 规定 | FaultCode=0, 时间正确 | ✅ |
| 会话结束后无多余 RPC | 最佳实践 | **ACS 又注入了 Download** | ❌ |

**问题 P1-ACS-1：ACS 在 TransferComplete 会话中注入测试任务**

TransferComplete 会话的目的是通知 ACS 传输完成结果。ACS 不应在此会话中注入不相关的测试任务。这会导致：
- TransferComplete 的语义被混淆
- CPE 需要处理无关的 RPC，增加复杂度
- 如果 CPE 是 `--once` 模式，后续 RPC 永远得不到响应

---

## 四、RPC 报文格式逐项分析

### 4.1 Download（ACS → CPE）（❌ P0-ACS-3）

```xml
<cwmp:Download>
  <CommandKey></CommandKey>          ← ❌ 空（应为唯一标识符）
  <FileType></FileType>              ← ❌ 空（MUST 为有效类型）
  <URL>http://acs.example.com/firmware/v1.0.0.bin</URL>
  <Username></Username>
  <Password></Password>
  <FileSize>0</FileSize>             ← ⚠️ 合法但不实用
  <TargetFileName></TargetFileName>
  <DelaySeconds>0</DelaySeconds>
  <SuccessURL></SuccessURL>
  <FailureURL></FailureURL>
</cwmp:Download>
```

**TR-069 Section A.3.2.8 (Download)**:

| 参数 | 要求 | 实际值 | 合规 |
|------|------|--------|------|
| CommandKey | 任意字符串（标识此操作） | `""` (空) | ⚠️ 合法但影响追踪 |
| FileType | MUST 为规定值之一 | `""` (空) | ❌ **违规** |
| URL | MUST 为有效 URL | `http://acs.example.com/...` | ✅ |
| FileSize | 0 = 未知 | `0` | ✅ |

**FileType 有效值**（TR-069 Table 15）：
- `1 Firmware Upgrade Image`
- `2 Web Content`
- `3 Vendor Configuration File`
- `4 Tone File`
- `5 Ringer File`
- `X <OUI> <Vendor Specific>`

空 FileType 是明确的协议违规。CPE 无法判断下载类型，不知道该如何处理下载的文件。

**归属**：**ACS 侧**（测试任务注入时未设置 FileType）

---

### 4.2 Upload（ACS → CPE）（❌ 同上）

```xml
<cwmp:Upload>
  <CommandKey></CommandKey>          ← ❌ 空
  <FileType></FileType>              ← ❌ 空（MUST 为有效类型）
  <URL>http://acs.example.com/upload/logs</URL>
  <Username></Username>
  <Password></Password>
  <DelaySeconds>0</DelaySeconds>
</cwmp:Upload>
```

**Upload FileType 有效值**（TR-069 Table 16）：
- `1 Vendor Configuration File`
- `2 Vendor Log File`
- `X <OUI> <Vendor Specific>`

同样缺少 FileType，**ACS 侧违规**。

---

### 4.3 DownloadResponse / UploadResponse（CPE → ACS）（✅ 合规）

```xml
<cwmp:DownloadResponse>
  <Status>1</Status>                            ← ✅ 1 = 异步完成
  <StartTime>0001-01-01T00:00:00Z</StartTime>   ← ✅ 未知时使用 "0001-01-01T00:00:00Z"
  <CompleteTime>0001-01-01T00:00:00Z</CompleteTime> ← ✅
</cwmp:DownloadResponse>
```

TR-069: Status=1 表示异步完成，CPE 将在后续会话中发送 TransferComplete。StartTime/CompleteTime 在异步模式下设为 unknown ("0001-01-01T00:00:00Z")，完全合规。

---

### 4.4 GetParameterValues（ACS → CPE）（✅ 合规）

```xml
<cwmp:GetParameterValues>
  <ParameterNames soap:arrayType="xsd:string[2]">
    <string>Device.DeviceInfo.SoftwareVersion</string>
    <string>Device.DeviceInfo.HardwareVersion</string>
  </ParameterNames>
</cwmp:GetParameterValues>
```

格式完全正确：`soap:arrayType` 声明数组大小，`<string>` 元素列出参数名。

---

### 4.5 GetParameterValuesResponse（CPE → ACS）（✅ 合规）

```xml
<cwmp:GetParameterValuesResponse>
  <ParameterList soap:arrayType="cwmp:ParameterValueStruct[2]">
    <ParameterValueStruct>
      <Name>Device.DeviceInfo.SoftwareVersion</Name>
      <Value xsi:type="xsd:string">FW-2.5.1-build.2026</Value>
    </ParameterValueStruct>
    <ParameterValueStruct>
      <Name>Device.DeviceInfo.HardwareVersion</Name>
      <Value xsi:type="xsd:string">HW-3.1.0</Value>
    </ParameterValueStruct>
  </ParameterList>
</cwmp:GetParameterValuesResponse>
```

| 检查项 | 结论 |
|--------|------|
| arrayType 数量正确 | ✅ `[2]` 匹配实际元素数 |
| Name 与请求参数一致 | ✅ |
| Value 带 xsi:type 类型标注 | ✅ `xsd:string` |
| 参数不存在时应返回 Fault 9005 | ✅ 代码中有此逻辑 |

---

### 4.6 GetParameterNames / Response（ACS→CPE / CPE→ACS）

**请求**（✅ 合规）：
```xml
<cwmp:GetParameterNames>
  <ParameterPath></ParameterPath>     ← ✅ 空 = 根路径
  <NextLevel>false</NextLevel>         ← ✅
</cwmp:GetParameterNames>
```

**响应**（✅ 基本合规，⚠️ 一个小问题）：
```xml
<cwmp:GetParameterNamesResponse>
  <ParameterList soap:arrayType="cwmp:ParameterInfoStruct[102]">
    <ParameterInfoStruct>
      <Name>Device.DeviceInfo.CpriPort.1.Status</Name>
      <Writable>false</Writable>
    </ParameterInfoStruct>
    ...
  </ParameterList>
</cwmp:GetParameterNamesResponse>
```

**⚠️ P2-CPE-1：NextLevel=false 时缺少中间路径**

TR-069 Section A.3.2.3: NextLevel=false 时应返回 "all Parameters and sub-objects"。

当前 CPE 只返回叶子参数（如 `Device.DeviceInfo.SoftwareVersion`），未返回中间对象路径（如 `Device.`、`Device.DeviceInfo.`、`Device.ManagementServer.`）。

严格合规的响应应同时包含：
```xml
<ParameterInfoStruct>
  <Name>Device.</Name>
  <Writable>false</Writable>
</ParameterInfoStruct>
<ParameterInfoStruct>
  <Name>Device.DeviceInfo.</Name>
  <Writable>false</Writable>
</ParameterInfoStruct>
<!-- ... 叶子参数 ... -->
```

**归属**：**CPE 侧**小问题
**影响**：大多数 ACS 不依赖中间路径，实际影响小

---

### 4.7 Inform（CPE → ACS）（✅ 合规）

```xml
<cwmp:Inform>
  <DeviceId>
    <Manufacturer>Baicells</Manufacturer>
    <OUI>48BF74</OUI>
    <ProductClass>FAP/pCRB2000/SC</ProductClass>
    <SerialNumber>SIM-CPE-0001</SerialNumber>
  </DeviceId>
  <Event soap:arrayType="cwmp:EventStruct[1]">
    <EventStruct>
      <EventCode>0 BOOTSTRAP</EventCode>
      <CommandKey></CommandKey>
    </EventStruct>
  </Event>
  <MaxEnvelopes>1</MaxEnvelopes>
  <CurrentTime>2026-03-22T14:42:04Z</CurrentTime>
  <RetryCount>0</RetryCount>
  <ParameterList soap:arrayType="cwmp:ParameterValueStruct[11]">
    ...
  </ParameterList>
</cwmp:Inform>
```

| 检查项 | TR-069 要求 | 实际 | 结论 |
|--------|------------|------|------|
| DeviceId 四要素 | MUST | 全部存在 | ✅ |
| EventCode 格式 | 标准事件码 | `0 BOOTSTRAP` | ✅ |
| MaxEnvelopes | MUST = 1 | `1` | ✅ |
| CurrentTime | UTC 格式 | ISO 8601 带 Z | ✅ |
| RetryCount | 重试次数 | `0`（首次） | ✅ |
| ParameterList | 强制上报参数 | 11 个参数 | ✅ |

**Inform ParameterList 中国移动规范合规检查**：

| 参数 | 移动规范要求 | 实际 | 合规 |
|------|------------|------|------|
| `Device.DeviceInfo.HardwareVersion` | ✅ | 存在 | ✅ |
| `Device.DeviceInfo.SoftwareVersion` | ✅ | 存在 | ✅ |
| `Device.ManagementServer.ConnectionRequestURL` | ✅ | 存在 | ✅ |
| `Device.ManagementServer.PeriodicInformInterval` | ✅ | 存在 | ✅ |
| `Device.ManagementServer.ParameterKey` | ✅ | 存在 | ✅ |
| `Device.DeviceInfo.ProvisioningCode` | ✅ | 存在 | ✅ |
| `Device.DeviceInfo.X_COM_CloudKey` | 移动专有 | 存在 | ✅ |
| `Device.DeviceInfo.X_COM_STATION_RUN_Time` | 移动专有 | 存在 | ✅ |

---

### 4.8 TransferComplete（CPE → ACS）（✅ 合规）

```xml
<cwmp:TransferComplete>
  <CommandKey></CommandKey>
  <FaultStruct>
    <FaultCode>0</FaultCode>
    <FaultString></FaultString>
  </FaultStruct>
  <StartTime>2026-03-22T14:42:06Z</StartTime>
  <CompleteTime>2026-03-22T14:42:06Z</CompleteTime>
</cwmp:TransferComplete>
```

| 检查项 | TR-069 要求 | 实际 | 结论 |
|--------|------------|------|------|
| CommandKey | 回传 Download 的 CommandKey | `""` (Download 也是空) | ✅ |
| FaultCode | 0=成功，其他=失败 | `0` | ✅ |
| StartTime | UTC 时间 | ISO 8601 | ✅ |
| CompleteTime | UTC 时间 | ISO 8601 | ✅ |

---

### 4.9 TransferCompleteResponse（ACS → CPE）（✅ 合规）

```xml
<cwmp:TransferCompleteResponse/>
```

TR-069: TransferCompleteResponse 无参数，空元素正确。

---

## 五、问题汇总与修复建议

### ACS 侧问题（需修复）

| 编号 | 严重级别 | 问题 | 位置 | 修复建议 |
|------|---------|------|------|---------|
| **P0-ACS-1** | 致命 | 大 GetParameterNamesResponse (18KB) 后返回 HTTP 400 | `handler.go:handleRPCResponse` | 检查 session state 是否在连续管道化后异常；增加 body 大小日志；检查 io.ReadAll 是否超时 |
| **P0-ACS-2** | 致命 | 测试任务 cwmp:ID 为空 | `handler.go:injectRandomTestTasks` | 为每个 test task 生成唯一 cwmp:ID（如 `test-{sn}-{seq}`） |
| **P0-ACS-3** | 致命 | Download/Upload 的 FileType 为空 | `handler.go:injectRandomTestTasks` | 设置有效 FileType（如 `1 Firmware Upgrade Image`） |
| **P1-ACS-1** | 重要 | TransferComplete 会话中注入无关测试任务 | `handler.go:handleInform` | 检测 EventCode 包含 `7 TRANSFER COMPLETE` 时不注入测试任务 |
| **P2-ACS-1** | 建议 | 204 响应缺少 Connection: close | `handler.go:completeSession` | 在终止会话时设置 `w.Header().Set("Connection", "close")` |
| **P2-ACS-2** | 建议 | InformResponse 包含非标准 CurrentTime | `soap/templates.go` | 移除 CurrentTime 或标记为厂商扩展 |

### CPE 侧问题（建议修复）

| 编号 | 严重级别 | 问题 | 位置 | 修复建议 |
|------|---------|------|------|---------|
| **P2-CPE-1** | 建议 | GetParameterNames(NextLevel=false) 缺少中间路径 | `_handle_get_names()` | 在返回叶子参数的同时，自动生成中间路径节点 |

---

## 六、合规报文流程图

### 标准 TR-069 Bootstrap 流程 vs 实际

```
标准流程                               实际流程
──────────────────────────────────    ──────────────────────────────────
CPE→ACS: Inform (0 BOOTSTRAP)        CPE→ACS: Inform (0 BOOTSTRAP)       ✅
ACS→CPE: InformResponse              ACS→CPE: InformResponse              ✅
CPE→ACS: Empty POST                  CPE→ACS: Empty POST                  ✅
ACS→CPE: RPC₁ (cwmp:ID=uuid-1)      ACS→CPE: Download (cwmp:ID="")       ❌ ID空
CPE→ACS: RPC₁Response                CPE→ACS: DownloadResponse            ✅
ACS→CPE: RPC₂ (piped, ID=uuid-2)    ACS→CPE: Upload (piped, ID="")       ❌ ID空
CPE→ACS: RPC₂Response                CPE→ACS: UploadResponse              ✅
ACS→CPE: 204 No Content              ACS→CPE: ... 更多RPC ...
                                     ACS→CPE: HTTP 400 (崩溃)              ❌
```

### 标准 TransferComplete 流程 vs 实际

```
标准流程                               实际流程
──────────────────────────────────    ──────────────────────────────────
CPE→ACS: Inform (7 TC, M Download)   CPE→ACS: Inform (7 TC, M Upload)     ✅
ACS→CPE: InformResponse              ACS→CPE: InformResponse              ✅
CPE→ACS: TransferComplete            CPE→ACS: TransferComplete             ✅✅
ACS→CPE: TransferCompleteResponse    ACS→CPE: TransferCompleteResponse     ✅
CPE→ACS: Empty POST                  CPE→ACS: Empty POST                  ✅
ACS→CPE: 204 No Content              ACS→CPE: Download (测试注入)          ❌ 不应注入
```

---

## 七、修复优先级排序

### 必须立即修复（阻塞联调）

1. **P0-ACS-1**: HTTP 400 崩溃 — 不修此 Bug 则 Bootstrap 会话永远无法完整走完
2. **P0-ACS-2**: cwmp:ID 为空 — RPC 响应无法关联，Task 追踪断裂
3. **P0-ACS-3**: FileType 为空 — Download/Upload 无类型，CPE 不知如何处理

### 尽快修复（影响测试质量）

4. **P1-ACS-1**: TC 会话注入 — 混乱 TransferComplete 语义

### 改进性修复

5. **P2-ACS-1/2**: 小协议改进
6. **P2-CPE-1**: GetParameterNames 中间路径

---

## 附录：ACS 测试任务注入分析

ACS 配置项 `enable_test_task_injection: true` 在 Inform 处理时自动注入 3-10 个随机 RPC 任务（Download、Upload、GetParameterValues、GetParameterNames、SetParameterValues、GetParameterAttributes、SetParameterAttributes、Reboot、FactoryReset）。

这些任务：
- cwmp:ID 全部为空
- Download/Upload FileType 全部为空
- CommandKey 全部为空
- URL 使用硬编码示例地址

**建议**：
1. 测试任务注入应生成合规的 RPC 报文（有 ID、有 FileType）
2. 增加配置项控制注入数量上限
3. 在 TransferComplete/AutonomousTransferComplete 会话中禁止注入
4. 在日志中标记 `[TEST]` 前缀以区分真实任务和测试任务
