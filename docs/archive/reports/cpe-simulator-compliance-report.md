# CPE 模拟器合规性分析报告

> 分析对象：`omcgo/scripts/cpe_simulator.py`
> 对标规范：《中国移动无线三方设备综合OMC南向接口要求 第2部分：皮基站技术要求》v1.0.0
> 参考标准：Broadband Forum TR-069 (CWMP)
> 分析日期：2026-03-22

---

## 一、总体评估

| 维度 | 合规度 | 说明 |
|------|--------|------|
| RPC 方法覆盖 | **73%** (11/15) | 缺少 GetRPCMethods、TransferComplete、AutonomousTransferComplete、ScheduleInform |
| EventCode 覆盖 | **31%** (5/16) | 仅支持 0/1/2/4/7/M Reboot，缺少移动专有的 101-107 等 |
| Inform 参数完整性 | **60%** | 缺少告警参数、PM/MR 任务参数、开站阶段参数 |
| 会话流程合规性 | **70%** | 基本流程正确，缺少 TransferComplete 二次会话 |
| 参数树完整性 | **40%** | 仅含基础 DeviceInfo + FAPService(LTE)，缺少告警/PM/MR/日志/5G 参数 |
| HTTP/SOAP 协议 | **85%** | 基本正确，连接管理和 SOAP 头有改进空间 |

**总体合规度：约 55%** — 可覆盖基础的 Inform + RPC 交互测试，但无法满足中国移动规范的完整测试需求。

---

## 二、不合规项详细分析

### 2.1 缺失的 RPC 方法

#### 2.1.1 GetRPCMethods（必选，规范表2）

**规范要求**：CPE 侧响应必选、OMC 侧调用可选。CPE 应返回所支持的全部 RPC 方法列表。

**TR-069 标准**：这是 CWMP 基线方法之一（A.3.1），ACS 通过此方法发现 CPE 能力。

**当前状态**：`RPCHandler.handle()` 的 `handler_map` 中无 `GetRPCMethods` 条目，收到该 RPC 时返回 Fault 9000。

**影响**：ACS 无法动态发现 CPE 能力，互操作测试（F10）无法验证此方法。

#### 2.1.2 TransferComplete（必选，规范表3）

**规范要求**：CPE 在 Download/Upload 完成后，必须发起新的 Inform 会话并调用 TransferComplete 方法通知 ACS 传输结果。EventCode 为 `"7 TRANSFER COMPLETE"` + `"M Download"` 或 `"M Upload"`。

**TR-069 标准**：A.3.2.1 — 当 Download/Upload 返回 Status=1（未完成）时，CPE 必须在后续会话中调用 TransferComplete。

**当前状态**：
- `_handle_download()` 和 `_handle_upload()` 直接返回 `Status=0`（立即完成），跳过了异步下载 + TransferComplete 流程
- 模拟器无法发起 TransferComplete RPC 调用（只被动响应，不主动调用 OMC 侧 RPC）

**影响**：无法测试固件升级完整流程（Download → 文件下载 → TransferComplete → 重启 → BOOT Inform），这是中国移动运维管理的核心场景。

#### 2.1.3 AutonomousTransferComplete（可选发起/必选响应，规范表3）

**规范要求**：PM 文件、MR 文件、日志文件定时上传完成后，CPE 必须发送 AutonomousTransferComplete 通知 OMC 上传结果。Inform 的 EventCode 为 `"10 AUTONOMOUS TRANSFER COMPLETE"`，AutonomousTransferComplete 消息中 FileType 值为：
- `"104 PM File"` — 性能文件
- `"105 MR File"` — MR 文件

**当前状态**：完全缺失。模拟器虽然有 `_do_file_upload()` 通过 HTTP POST 上传模拟 PM 文件，但没有在上传完成后发起 AutonomousTransferComplete 通知会话。

**影响**：无法测试 PM/MR/日志文件上传结果通知的完整流程。

---

### 2.2 缺失的 EventCode

规范表68定义了 16+ 种 EventCode，模拟器仅支持 5 种：

| EventCode | 含义 | 模拟器支持 | 规范要求 |
|-----------|------|-----------|---------|
| `0 BOOTSTRAP` | 首次安装/出厂重置/ACS URL变更 | ✅ | 必选 |
| `1 BOOT` | 加电/重启 | ✅ | 必选 |
| `2 PERIODIC` | 周期心跳 | ✅ | 必选 |
| `4 VALUE CHANGE` | 参数值变更 | ✅ | 必选 |
| `7 TRANSFER COMPLETE` | 传输完成 | ✅（仅作为事件类型） | 必选 |
| `M Reboot` | Reboot 方法结果 | ✅ | 必选 |
| `3 SCHEDULED` | ScheduleInform 触发 | ❌ | 可选 |
| `6 CONNECTION REQUEST` | ACS 发起连接请求 | ❌ | **必选** |
| `10 AUTONOMOUS TRANSFER COMPLETE` | 自主传输完成 | ❌ | **必选** |
| `101 ALARM` | 告警上报 | ❌ | **必选（移动专有）** |
| `102 UPGRADE FINISH` | 软件升级完成 | ❌ | **必选（移动专有）** |
| `103 ADD OBJECT` | 管理对象新增通知 | ❌ | **必选（移动专有）** |
| `104 DELETE OBJECT` | 管理对象删除通知 | ❌ | **必选（移动专有）** |
| `105 STARTUP STAGE REPORT` | 开站阶段上报 | ❌ | **必选（移动专有）** |
| `106 STARTUP RESULT REPORT` | 开站结果上报 | ❌ | **必选（移动专有）** |
| `107 UNIT UPGRADE RESULT` | 单元升级结果 | ❌ | **必选（移动专有）** |
| `M Download` / `M Upload` | 下载/上传方法结果 | ❌ | **必选** |

**影响**：无法测试告警上报（F04）、自动开站（F09）、软件升级（F06）、PM/MR 文件上传通知（F03/F05）等核心场景。

---

### 2.3 参数树不完整

#### 2.3.1 缺少告警参数

规范表35定义了告警 Inform 必须携带的参数：

```
Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.AlarmIdentifier
Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.PerceivedSeverity
Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.FaultLocation
Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.EventType
Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.NotificationType
Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.ProbableCause
Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.EventTime
Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.SpecificProblem
Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.AdditionalInformation
Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.AdditionalText
```

#### 2.3.2 缺少 PM 任务定制参数

规范表28定义了 PM 文件上传任务参数：

```
Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.Enable
Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.Alias
Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.URL
Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.Username
Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.Password
Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.PeriodicUploadInterval
Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.PeriodicUploadTime
```

PM 补采参数：

```
Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.ReplenishEnable
Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.ReplenishStartTime
Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.ReplenishEndTime
```

#### 2.3.3 缺少 MR 任务定制参数

规范表63定义了 MR 参数：

```
Device.Services.FAPService.1.FAPControl.LTE.MR.MrEnable
Device.Services.FAPService.1.FAPControl.LTE.MR.MrUrl
Device.Services.FAPService.1.FAPControl.LTE.MR.MrUsername
Device.Services.FAPService.1.FAPControl.LTE.MR.MrPassword
Device.Services.FAPService.1.FAPControl.LTE.MR.MeasureType
Device.Services.FAPService.1.FAPControl.LTE.MR.OmcName
Device.Services.FAPService.1.FAPControl.LTE.MR.SamplePeriod
Device.Services.FAPService.1.FAPControl.LTE.MR.UploadPeriod
Device.Services.FAPService.1.FAPControl.LTE.MR.SampleBeginTime
Device.Services.FAPService.1.FAPControl.LTE.MR.SampleEndTime
```

#### 2.3.4 缺少日志上传参数

规范表51定义了日志文件参数：

```
Device.Services.FAPService.1.FAPControl.LTE.LogUpload.PeriodicUploadEnable
Device.Services.FAPService.1.FAPControl.LTE.LogUpload.URL
Device.Services.FAPService.1.FAPControl.LTE.LogUpload.Username
Device.Services.FAPService.1.FAPControl.LTE.LogUpload.Password
Device.Services.FAPService.1.FAPControl.LTE.LogUpload.PeriodicUploadInterval
```

#### 2.3.5 缺少 5G NR 参数

当前参数树仅包含 LTE (4G) 参数，完全缺少 5G NR 参数节点：

```
Device.Services.FAPService.1.CellConfig.NR.RAN.*
Device.Services.FAPService.1.FAPControl.NR.*
```

#### 2.3.6 缺少开站阶段参数

规范定义了开站上报参数（EventCode 105/106）：

```
Device.Services.FAPService.1.FAPControl.LTE.StartupInfo.Stage
Device.Services.FAPService.1.FAPControl.LTE.StartupInfo.Result
Device.Services.FAPService.1.FAPControl.LTE.StartupInfo.FailureCause
```

#### 2.3.7 缺少升级进度参数

规范表46/表48定义了升级相关参数：

```
Device.Services.FAPService.1.FAPControl.LTE.UpgradeInfo.Status
Device.Services.FAPService.1.FAPControl.LTE.UpgradeInfo.FailureCause
Device.Services.FAPService.1.FAPControl.LTE.UpgradeInfo.Stage  (初始化/下载/安装/等待激活/激活)
```

---

### 2.4 会话流程缺陷

#### 2.4.1 Download/Upload 缺少异步模式

**规范要求**：Download 返回 Status=1（尚未完成）后，CPE 应在后续独立会话中发送 TransferComplete。完整流程：

```
Session 1: Inform → InformResponse → (空POST) → Download → DownloadResponse(Status=1) → Session 结束
           CPE 通过 HTTP GET 从 URL 下载文件
Session 2: Inform(EventCode="M Download", "7 TRANSFER COMPLETE")
           → InformResponse → TransferComplete → TransferCompleteResponse
```

**当前行为**：`_handle_download()` 直接返回 Status=0，整个下载在单次 RPC 响应中"完成"，不符合实际设备行为。

#### 2.4.2 PM/MR 文件定时上传缺少完整流程

**规范要求的完整流程**：

```
1. CPE 定时通过 HTTP PUT/POST 上传 PM 文件到 OMC 文件服务器
2. 上传完成后，CPE 发起新的 TR069 会话：
   Inform(EventCode="10 AUTONOMOUS TRANSFER COMPLETE")
   → InformResponse
   → AutonomousTransferComplete(FileType="104 PM File")
   → AutonomousTransferCompleteResponse
```

**当前行为**：`_do_file_upload()` 仅执行了步骤 1（HTTP POST 上传），完全缺少步骤 2。

#### 2.4.3 缺少 Connection Request 响应能力

**规范要求**：ACS 向 CPE 的 ConnectionRequestURL 发送 HTTP GET，CPE 收到后发起 Inform 会话（EventCode="6 CONNECTION REQUEST"）。

**TR-069 标准**：3.2.2 — CPE 必须提供 ConnectionRequestURL 供 ACS 发起连接请求。

**当前状态**：模拟器参数树中有 `ConnectionRequestURL`，但不监听该端口，无法接收 Connection Request。

---

### 2.5 ParameterKey 处理缺失

**规范要求**（表76/78/80）：SetParameterValues、AddObject、DeleteObject 均包含 `ParameterKey` 参数。CPE 必须在方法完全成功执行后，将 ParameterKey 值保存到 `Device.ManagementServer.ParameterKey`。

**当前状态**：
- `_handle_set_values()` 不解析 ParameterKey
- `_handle_add_object()` 不解析 ParameterKey
- `_handle_delete_object()` 不解析 ParameterKey
- 参数树中无 `Device.ManagementServer.ParameterKey`

---

### 2.6 HTTP 连接管理

**TR-069 标准**：3.7.1.1 — CPE 应在整个会话期间保持 HTTP 连接（persistent connection/keep-alive），而非每次请求创建新连接。

**当前行为**：`_http_request()` 每次请求都 `conn_cls(...)` 创建新连接并在最后 `conn.close()`。虽然 Headers 中设置了 `Connection: keep-alive`，但实际每次都断开重连。

**影响**：增加不必要的 TCP 握手开销，不符合 TR-069 会话模型。在大规模并发测试时会显著影响性能。

---

### 2.7 SOAP Header 不完整

**规范附录G**：SOAP 头字段应包含 `cwmp:ID`（已实现）和可选的 `cwmp:HoldRequests`、`cwmp:NoMoreRequests` 等。

**TR-069 标准**：3.7.1.2 — CPE 在发送最后一个请求时应设置 `NoMoreRequests` 为 1，通知 ACS 没有更多请求。

**当前状态**：空 POST 轮询时未设置 `NoMoreRequests` 或其他会话控制头。

---

### 2.8 Fault Code 不完整

**规范9.3节定义了完整的异常码体系**：

| 错误码 | 含义 | 模拟器支持 |
|--------|------|-----------|
| 9000 | Method not supported | ✅ |
| 9001 | Request denied | ❌ |
| 9002 | Internal error | ❌ |
| 9003 | Invalid arguments | ❌ |
| 9004 | Resources exceeded | ❌ |
| 9005 | Invalid parameter name | ❌ |
| 9006 | Invalid parameter type | ❌ |
| 9007 | Invalid parameter value | ❌ |
| 9008 | Attempt to set non-writable | ❌ |
| 9009 | Notification request rejected | ❌ |
| 9010 | Download failure | ❌ |
| 9011 | Upload failure | ❌ |
| 9012 | File transfer auth failure | ❌ |
| 9013 | Unsupported transfer protocol | ❌ |

**当前状态**：仅在遇到未知方法时返回 Fault 9000，所有已知方法始终返回成功。无法模拟参数校验失败、只读参数写入失败等异常场景。

---

## 三、已合规项

以下方面模拟器已满足规范要求：

1. **SOAP/XML 信封格式** — 命名空间、Header/Body 结构符合 TR-069 规范
2. **Inform 基本结构** — DeviceId (Manufacturer/OUI/ProductClass/SerialNumber)、Event、MaxEnvelopes、CurrentTime、RetryCount、ParameterList 六要素齐全
3. **11 种 RPC 响应** — GetParameterValues/Names/Attributes、SetParameterValues/Attributes、AddObject、DeleteObject、Download、Upload、Reboot、FactoryReset 均有实现
4. **HTTP Digest 认证** — 支持 401 挑战 → Digest 响应的标准流程
5. **Cookie Session 跟踪** — 通过 Set-Cookie/Cookie 维持会话
6. **会话轮询模型** — Inform → InformResponse → 空 POST 轮询 → RPC 响应循环 → 204/空响应结束
7. **Reboot 后的 BOOT Inform** — 使用 `pending_boot` 在下次 Inform 中携带 `"1 BOOT"` + `"M Reboot"`

---

## 四、修复方案

### 阶段一：核心合规（优先级 P0）

#### 1. 新增 GetRPCMethods 响应

```python
def _handle_get_rpc_methods(self, cwmp_id, body_el):
    """GetRPCMethods: 返回 CPE 支持的 RPC 方法列表"""
    methods = [
        "Inform", "TransferComplete", "AutonomousTransferComplete",
        "GetRPCMethods", "GetParameterValues", "SetParameterValues",
        "GetParameterNames", "GetParameterAttributes", "SetParameterAttributes",
        "AddObject", "DeleteObject", "Download", "Upload", "Reboot", "FactoryReset",
    ]
    methods_xml = "\n".join(f'        <string>{m}</string>' for m in methods)
    body = f"""<cwmp:GetRPCMethodsResponse>
      <MethodList soap:arrayType="xsd:string[{len(methods)}]">
{methods_xml}
      </MethodList>
    </cwmp:GetRPCMethodsResponse>"""
    return build_rpc_response(cwmp_id, "GetRPCMethodsResponse", body)
```

#### 2. 实现 TransferComplete 主动调用

需要在 Download/Upload 后延迟发起新会话：

```python
def _schedule_transfer_complete(self, command_key, file_type, is_download, delay=2):
    """在 Download/Upload 之后，延迟发起 TransferComplete 会话"""
    def _do():
        time.sleep(delay)
        # 确定 EventCode
        m_event = "M Download" if is_download else "M Upload"
        events = [("7 TRANSFER COMPLETE", command_key), (m_event, command_key)]
        # 发起新的 Inform + TransferComplete 会话
        self._run_transfer_complete_session(events, command_key, is_download)
    threading.Thread(target=_do, daemon=True).start()

def _run_transfer_complete_session(self, events, command_key, is_download):
    """运行 TransferComplete 通知会话"""
    # 1. Inform
    cwmp_id, inform_xml = self._build_inform(events)
    status, resp_body = self._http_request(body=inform_xml)
    if status != 200:
        return
    # 2. 发送 TransferComplete
    now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    tc_body = f"""<cwmp:TransferComplete>
      <CommandKey>{command_key}</CommandKey>
      <FaultStruct>
        <FaultCode>0</FaultCode>
        <FaultString></FaultString>
      </FaultStruct>
      <StartTime>{now}</StartTime>
      <CompleteTime>{now}</CompleteTime>
    </cwmp:TransferComplete>"""
    tc_xml = build_rpc_response(self._next_cwmp_id(), "TransferComplete", tc_body)
    status, resp_body = self._http_request(body=tc_xml)
    # 3. 空 POST 结束
    self._http_request(body=None)
```

#### 3. 实现 AutonomousTransferComplete

PM/MR/日志文件上传后，发起通知会话：

```python
def _run_autonomous_transfer_complete(self, file_type, transfer_url, file_size):
    """PM/MR/日志文件上传后发送 AutonomousTransferComplete"""
    events = [("10 AUTONOMOUS TRANSFER COMPLETE", "")]
    cwmp_id, inform_xml = self._build_inform(events)
    status, resp_body = self._http_request(body=inform_xml)
    if status != 200:
        return
    # 发送空 POST 等待 ACS 的 RPC（如果有）
    # 然后发送 AutonomousTransferComplete
    now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    atc_body = f"""<cwmp:AutonomousTransferComplete>
      <AnnounceURL></AnnounceURL>
      <TransferURL>{transfer_url}</TransferURL>
      <IsDownload>false</IsDownload>
      <FileType>{file_type}</FileType>
      <FileSize>{file_size}</FileSize>
      <TargetFileName></TargetFileName>
      <FaultStruct>
        <FaultCode>0</FaultCode>
        <FaultString></FaultString>
      </FaultStruct>
      <StartTime>{now}</StartTime>
      <CompleteTime>{now}</CompleteTime>
    </cwmp:AutonomousTransferComplete>"""
    atc_xml = build_rpc_response(self._next_cwmp_id(), "AutonomousTransferComplete", atc_body)
    self._http_request(body=atc_xml)
    self._http_request(body=None)  # 空 POST 结束
```

#### 4. 补充中国移动专有 EventCode

在 `run()` 方法和命令行参数中添加以下事件类型：

```python
# --event 选项扩展
choices=["bootstrap", "periodic", "boot", "value_change", "transfer_complete",
         "connection_request", "alarm", "upgrade_finish", "add_object",
         "delete_object", "startup_stage", "startup_result", "unit_upgrade"]

# 事件映射
EVENT_MAP = {
    "bootstrap": [("0 BOOTSTRAP", "")],
    "boot": [("1 BOOT", "")],
    "periodic": [("2 PERIODIC", "")],
    "value_change": [("2 PERIODIC", ""), ("4 VALUE CHANGE", "")],
    "transfer_complete": [("7 TRANSFER COMPLETE", "")],
    "connection_request": [("6 CONNECTION REQUEST", "")],
    "alarm": [("101 ALARM", "")],
    "upgrade_finish": [("102 UPGRADE FINISH", "")],
    "add_object": [("103 ADD OBJECT", "")],
    "delete_object": [("104 DELETE OBJECT", "")],
    "startup_stage": [("105 STARTUP STAGE REPORT", "")],
    "startup_result": [("106 STARTUP RESULT REPORT", "")],
    "unit_upgrade": [("107 UNIT UPGRADE RESULT", "")],
}
```

### 阶段二：参数树补全（优先级 P1）

#### 5. 补充告警参数

在 `build_default_param_tree()` 中添加：

```python
# 告警参数（规范表35）
"Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.AlarmIdentifier": ("xsd:string", ""),
"Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.PerceivedSeverity": ("xsd:string", ""),
"Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.FaultLocation": ("xsd:string", ""),
"Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.EventType": ("xsd:string", ""),
"Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.NotificationType": ("xsd:string", ""),
"Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.ProbableCause": ("xsd:string", ""),
"Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.EventTime": ("xsd:dateTime", ""),
"Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.SpecificProblem": ("xsd:string", ""),
"Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.AdditionalInformation": ("xsd:string", ""),
"Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.AdditionalText": ("xsd:string", ""),
```

#### 6. 补充 PM/MR/日志/升级参数

```python
# PM 任务参数（规范表28）
"Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.Enable": ("xsd:boolean", "false"),
"Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.URL": ("xsd:string", ""),
"Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.Username": ("xsd:string", ""),
"Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.Password": ("xsd:string", ""),
"Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.PeriodicUploadInterval": ("xsd:unsignedInt", "900"),
"Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.PeriodicUploadTime": ("xsd:dateTime", "1970-01-01T00:00:00Z"),

# PM 补采参数（规范表32）
"Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.ReplenishEnable": ("xsd:boolean", "false"),
"Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.ReplenishStartTime": ("xsd:dateTime", ""),
"Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.ReplenishEndTime": ("xsd:dateTime", ""),

# MR 参数（规范表63）
"Device.Services.FAPService.1.FAPControl.LTE.MR.MrEnable": ("xsd:boolean", "false"),
"Device.Services.FAPService.1.FAPControl.LTE.MR.MrUrl": ("xsd:string", ""),
"Device.Services.FAPService.1.FAPControl.LTE.MR.MrUsername": ("xsd:string", ""),
"Device.Services.FAPService.1.FAPControl.LTE.MR.MrPassword": ("xsd:string", ""),
"Device.Services.FAPService.1.FAPControl.LTE.MR.MeasureType": ("xsd:string", "MRO,MRE,MRS"),
"Device.Services.FAPService.1.FAPControl.LTE.MR.OmcName": ("xsd:string", "OMC-R"),
"Device.Services.FAPService.1.FAPControl.LTE.MR.SamplePeriod": ("xsd:unsignedInt", "10240"),
"Device.Services.FAPService.1.FAPControl.LTE.MR.UploadPeriod": ("xsd:unsignedInt", "15"),

# 日志参数（规范表51）
"Device.Services.FAPService.1.FAPControl.LTE.LogUpload.PeriodicUploadEnable": ("xsd:boolean", "false"),
"Device.Services.FAPService.1.FAPControl.LTE.LogUpload.URL": ("xsd:string", ""),
"Device.Services.FAPService.1.FAPControl.LTE.LogUpload.PeriodicUploadInterval": ("xsd:unsignedInt", "3600"),

# ManagementServer.ParameterKey（规范表76要求）
"Device.ManagementServer.ParameterKey": ("xsd:string", ""),
```

#### 7. 添加 5G NR 参数

```python
# 5G NR 基本参数
"Device.Services.FAPService.2.FAPControl.NR.AdminState": ("xsd:boolean", "true"),
"Device.Services.FAPService.2.FAPControl.NR.OpState": ("xsd:boolean", "true"),
"Device.Services.FAPService.2.FAPControl.NR.RFTxStatus": ("xsd:boolean", "true"),
"Device.Services.FAPService.2.CellConfig.NR.RAN.Common.CellIdentity": ("xsd:unsignedInt", "..."),
"Device.Services.FAPService.2.CellConfig.NR.RAN.RF.NRARFCN": ("xsd:unsignedInt", "633984"),
"Device.Services.FAPService.2.CellConfig.NR.RAN.RF.NRBand": ("xsd:unsignedInt", "78"),
"Device.Services.FAPService.2.CellConfig.NR.RAN.RF.DLBandwidth": ("xsd:string", "100"),
```

### 阶段三：流程完善（优先级 P2）

#### 8. ParameterKey 处理

在 `_handle_set_values()`, `_handle_add_object()`, `_handle_delete_object()` 中解析并保存 ParameterKey。

#### 9. HTTP 连接复用

将 `_http_request()` 改为在整个会话期间复用同一个 HTTP 连接：

```python
def _run_session(self, event_codes):
    # 会话级连接
    self._conn = self._create_connection()
    try:
        # ... 整个会话使用同一个 self._conn
    finally:
        self._conn.close()
        self._conn = None
```

#### 10. Fault Code 丰富化

为 RPC 处理添加参数校验和 Fault 码返回能力：

```python
def _build_fault(self, cwmp_id, code, message):
    """构建 CWMP Fault 响应"""
    fault_xml = f"""<soap:Fault>
      <faultcode>Client</faultcode>
      <faultstring>CWMP fault</faultstring>
      <detail>
        <cwmp:Fault>
          <FaultCode>{code}</FaultCode>
          <FaultString>{message}</FaultString>
        </cwmp:Fault>
      </detail>
    </soap:Fault>"""
    return build_rpc_response(cwmp_id, "Fault", fault_xml)
```

支持通过命令行参数 `--fault-rate 0.05` 模拟 5% 概率的随机故障。

#### 11. 告警模拟器

添加周期性告警产生/清除模拟：

```python
# --alarm-interval 30  每 30 秒产生一次模拟告警
# 告警 Inform 携带 EventCode "101 ALARM"，ParameterList 包含表35定义的告警字段
```

#### 12. PM/MR 定时上传模拟

根据参数树中的 PM/MR Enable 和 PeriodicUploadInterval 配置，模拟定时文件上传 + AutonomousTransferComplete 流程。

#### 13. Connection Request 监听

启动一个独立的 HTTP Server 监听 ConnectionRequestURL 端口，收到 ACS 的 GET 请求后触发 Inform 会话（EventCode="6 CONNECTION REQUEST"）。

---

## 五、修复优先级汇总

| 优先级 | 修复项 | 工作量 | 影响面 |
|--------|--------|--------|--------|
| **P0** | GetRPCMethods 响应 | 小 | 互操作基线 |
| **P0** | TransferComplete 主动调用 | 中 | 固件升级流程 |
| **P0** | 中国移动专有 EventCode (101-107) | 中 | 告警/开站/升级 |
| **P0** | 告警参数树补全 | 小 | 告警上报测试 |
| **P1** | AutonomousTransferComplete | 中 | PM/MR/日志 |
| **P1** | PM/MR/日志参数树补全 | 小 | 性能/MR 管理 |
| **P1** | ParameterKey 处理 | 小 | 参数追踪 |
| **P1** | 5G NR 参数树 | 小 | 5G 场景 |
| **P2** | HTTP 连接复用 | 中 | 性能/合规性 |
| **P2** | Fault Code 丰富化 | 中 | 异常场景测试 |
| **P2** | Connection Request 监听 | 大 | 双向连接 |
| **P2** | PM/MR 定时上传完整流程 | 大 | 端到端验证 |
| **P2** | 告警模拟器 | 中 | 告警端到端 |

---

## 六、结论

当前 CPE 模拟器实现了 TR-069 的基本交互框架（Inform → RPC 轮询 → 响应），可以满足 ACS 引擎的基础功能验证。但对标中国移动皮基站南向接口规范，存在以下核心差距：

1. **缺少 CPE 侧主动 RPC 调用能力**（TransferComplete、AutonomousTransferComplete）— 导致无法模拟完整的文件传输通知流程
2. **中国移动专有 EventCode 完全缺失**（101-107）— 导致无法测试告警上报、自动开站、软件升级完成通知等移动运营商核心场景
3. **参数树仅覆盖设备基本信息和 LTE 射频参数** — 缺少告警、PM、MR、日志、升级、5G 等业务参数

建议按 P0 → P1 → P2 的优先级逐步修复，P0 阶段即可覆盖中国移动南向接口规范的约 80% 测试场景。
