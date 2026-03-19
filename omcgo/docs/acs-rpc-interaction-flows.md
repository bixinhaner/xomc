# ACS RPC 交互流程规范整理

> 来源：《中国移动无线三方设备综合OMC南向接口要求 第2部分：皮基站技术要求》附录A
> 版本：1.0.0
> 关联设计文档：`doc/detailed-design/07-acs-engine.md`

---

## 1. 概述

### 1.1 角色定义

| 角色 | 协议身份 | 说明 |
|------|---------|------|
| **OMC** | ACS (Auto-Configuration Server) | 网络管理系统，TR069 服务端 |
| **CPE** | CPE (Customer Premises Equipment) | NanoCell/Extended Pico Cell 皮基站设备，TR069 客户端 |
| **FileServer** | HTTP/HTTPS 文件服务器 | 用于性能/日志/MR 文件的上传下载，由 OMC 指定地址 |

### 1.2 通用会话模式

所有 TR069 交互都基于 **HTTP 会话** 进行，CPE 始终作为 HTTP 客户端发起 TCP 连接。

#### 模式一：OMC 主动发起（Connection Request 触发）

```
CPE                                     OMC(ACS)
 |                                         |
 |  <--- HTTP GET (Connection Request) ----|   ← OMC 向 CPE 的 ConnReqURL 发起
 |  --- HTTP 200 OK ---------------------->|
 |                                         |
 |  --- Inform (Event="6 CONN REQ") ------>|   ← CPE 主动连接 ACS
 |  <-- InformResponse --------------------|
 |  --- Empty HTTP POST ------------------>|   ← CPE 表示无更多请求
 |  <-- RPC Request (如 GPV/SPV/...) ------|   ← OMC 下发 RPC 命令
 |  --- RPC Response --------------------->|   ← CPE 返回结果
 |  ...（可重复 RPC 请求/响应）...           |
 |  <-- Empty HTTP Response ---------------|   ← OMC 无更多命令，会话结束
 |  (断开连接)                              |
```

#### 模式二：CPE 主动发起（周期心跳/事件上报）

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform (Event=具体事件码) --------->|   ← CPE 主动上报
 |  <-- InformResponse --------------------|
 |  --- Empty HTTP POST ------------------>|
 |  <-- Empty HTTP Response ---------------|   ← 若 OMC 无命令则直接结束
 |  (断开连接)                              |
```

#### 模式三：CPE 上报 + 后续 RPC 交互（混合模式）

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform (Event=具体事件码) --------->|
 |  <-- InformResponse --------------------|
 |  --- TransferComplete / ATC ----------->|   ← CPE 上报传输结果
 |  <-- TC Response / ATC Response --------|
 |  --- Empty HTTP POST ------------------>|
 |  <-- Empty HTTP Response ---------------|
 |  (断开连接)                              |
```

> **关键规则**：在同一会话中，OMC 可多次重复调用同一 RPC 方法（如连续发送多个 GetParameterValues）。

---

## 2. 操作→RPC 方法映射总表

### 2.1 CPE 侧 RPC 方法（CPE 响应，OMC 调用）

| 方法名 | 中文说明 | CPE 要求 | OMC 要求 |
|--------|---------|---------|---------|
| GetRPCMethods | 获取 RPC 方法列表 | 必选(响应) | 可选(调用) |
| **SetParameterValues** | 设置参数值 | 必选 | 必选 |
| **GetParameterValues** | 获取参数值 | 必选 | 必选 |
| **GetParameterNames** | 获取参数列表 | 必选 | 必选 |
| **SetParameterAttributes** | 设置参数属性 | 必选 | 必选 |
| **GetParameterAttributes** | 获取参数属性 | 必选 | 必选 |
| **AddObject** | 增加对象 | 必选 | 必选 |
| **DeleteObject** | 删除对象 | 必选 | 必选 |
| **Reboot** | 重启 | 必选 | 必选 |
| **Download** | 下载 | 必选 | 必选 |
| **Upload** | 上传 | 必选 | 必选 |
| **FactoryReset** | 恢复出厂设置 | 必选 | 必选 |

### 2.2 OMC 侧 RPC 方法（OMC 响应，CPE 调用）

| 方法名 | 中文说明 | CPE 要求 | OMC 要求 |
|--------|---------|---------|---------|
| GetRPCMethods | 获取 RPC 方法列表 | 可选(调用) | 必选(响应) |
| **Inform** | 通知 | 必选 | 必选 |
| **TransferComplete** | 传输完成 | 必选 | 必选 |
| **AutonomousTransferComplete** | 自动传输完成 | 可选 | 必选 |

### 2.3 接口操作→TR069 方法映射

| 分析阶段操作 | TR069 RPC 方法 | 方向 | 管理域 |
|-------------|---------------|------|--------|
| getCmName | GetParameterNames | ACS→CPE | 配置管理 |
| getCmValue | GetParameterValues | ACS→CPE | 配置管理 |
| setCmValue | SetParameterValues | ACS→CPE | 配置管理 |
| addCmData | AddObject | ACS→CPE | 配置管理 |
| delCmData | DeleteObject | ACS→CPE | 配置管理 |
| cmDataNotify | Inform (4 VALUE CHANGE) | CPE→ACS | 配置管理 |
| addCmDataNotify | Inform (103 ADD OBJECT) | CPE→ACS | 配置管理 |
| delCmDataNotify | Inform (104 DELETE OBJECT) | CPE→ACS | 配置管理 |
| getCmAttributes | GetParameterAttributes | ACS→CPE | 配置管理 |
| setCmAttributes | SetParameterAttributes | ACS→CPE | 配置管理 |
| setPMValues | SetParameterValues | ACS→CPE | 性能管理 |
| uploadPmFile | HTTP PUT/POST | CPE→FileServer | 性能管理 |
| uploadPmResultNotify | Inform + AutonomousTransferComplete | CPE→ACS | 性能管理 |
| setPMReUploadValues | SetParameterValues | ACS→CPE | 性能管理 |
| reportAlarm | Inform | CPE→ACS | 故障管理 |
| syncAlarm | GetParameterValues | ACS→CPE | 故障管理 |
| deviceOnlineNotify | Inform (2 PERIODIC) | CPE→ACS | 维护管理 |
| deviceReboot | Reboot | ACS→CPE | 维护管理 |
| deviceRebootNotify | Inform (M Reboot) | CPE→ACS | 维护管理 |
| deviceUpgrade | Download | ACS→CPE | 维护管理 |
| deviceSoftwareDownload | HTTP GET | CPE→FileServer | 维护管理 |
| deviceSoftwareDownloadCompleteNotify | Inform (7 TRANSFER COMPLETE) | CPE→ACS | 维护管理 |
| deviceUnitUpgradeNotify | Inform (4 VALUE CHANGE) | CPE→ACS | 维护管理 |
| deviceUpgradeFinishNotify | Inform (4 VALUE CHANGE) | CPE→ACS | 维护管理 |
| getUnitUpgradeStage | GetParameterValues | ACS→CPE | 维护管理 |
| deviceFactoryReset | FactoryReset | ACS→CPE | 维护管理 |
| setLogUploadInfo | SetParameterValues | ACS→CPE | 维护管理 |
| getDeviceLogFile | Upload | ACS→CPE | 维护管理 |
| deviceFileUploadRequest | Upload | ACS→CPE | 维护管理 |
| deviceFileDownloadRequest | Download | ACS→CPE | 维护管理 |
| setMRValues | SetParameterValues | ACS→CPE | MR管理 |

---

## 3. 配置管理交互流程（A.1 — A.8）

### A.1 配置参数名称查询

**触发方**：OMC
**核心RPC**：`GetParameterNames` / `GetParameterNamesResponse`
**场景**：OMC 查询 CPE 设备上的配置参数名称（发现参数树）

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform (6 CONN REQ) ------------->|  1. CPE 发起 Inform，说明由 OMC 要求建立会话
 |  <-- InformResponse -------------------|  2. OMC 返回 InformResponse
 |  --- Empty HTTP POST ----------------->|  3. CPE 发送空 POST 表示无请求
 |  <-- GetParameterNames ----------------|  4. OMC 下发 GetParameterNames
 |  --- GetParameterNamesResponse ------->|  5. CPE 返回参数名称列表（含读写属性）
 |  <-- Empty HTTP Response --------------|  6. OMC 无更多命令
 |  (断开连接)                             |  7. 会话结束
```

**输入参数**：
- `parameterPath` — 指定参数的路径，查询根节点可为空
- `nextLevel` — `true` 只查下一级；`false` 查询所有下级

**返回**：`parameterList` — 参数名/路径名列表，含读写属性

**备注**：OMC 可在同一会话中多次重复调用 GetParameterNames。

---

### A.2 配置参数值查询

**触发方**：OMC
**核心RPC**：`GetParameterValues` / `GetParameterValuesResponse`
**场景**：OMC 查询 CPE 设备上的配置参数值

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform (6 CONN REQ) ------------->|  1. CPE 发起 Inform
 |  <-- InformResponse -------------------|  2. OMC 返回 InformResponse
 |  --- Empty HTTP POST ----------------->|  3. CPE 发送空 POST
 |  <-- GetParameterValues ---------------|  4. OMC 下发 GetParameterValues
 |  --- GetParameterValuesResponse ------>|  5. CPE 返回参数名称和值
 |  <-- Empty HTTP Response --------------|  6. OMC 无更多命令
 |  (断开连接)                             |  7. 会话结束
```

**输入参数**：`parameterNames` — 待查询的参数名称列表
**返回**：`parameterList` — 参数名和参数值列表

**备注**：OMC 可多次重复调用 GetParameterValues。

---

### A.3 配置参数值修改

**触发方**：OMC
**核心RPC**：`SetParameterValues` / `SetParameterValuesResponse`
**场景**：OMC 修改 CPE 设备上的配置参数值

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform (6 CONN REQ) ------------->|  1. CPE 发起 Inform
 |  <-- InformResponse -------------------|  2. OMC 返回 InformResponse
 |  --- Empty HTTP POST ----------------->|  3. CPE 发送空 POST
 |  <-- SetParameterValues ---------------|  4. OMC 下发 SetParameterValues
 |  --- SetParameterValuesResponse ------>|  5. CPE 返回修改结果
 |  <-- Empty HTTP Response --------------|  6. OMC 无更多命令
 |  (断开连接)                             |  7. 会话结束
```

**输入参数**：`parameterList` — 参数名称和值对的列表
**返回**：`status` — 已应用（立即生效）或已提交（需重启生效）

**备注**：OMC 可多次重复调用 SetParameterValues。

---

### A.4 配置管理对象新增

**触发方**：OMC
**核心RPC**：`AddObject` / `AddObjectResponse`
**场景**：OMC 在 CPE 上新增多实例配置参数对象

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform (6 CONN REQ) ------------->|  1. CPE 发起 Inform
 |  <-- InformResponse -------------------|  2. OMC 返回 InformResponse
 |  --- Empty HTTP POST ----------------->|  3. CPE 发送空 POST
 |  <-- AddObject ------------------------|  4. OMC 下发 AddObject
 |  --- AddObjectResponse --------------->|  5. CPE 返回新实例号码
 |  <-- Empty HTTP Response --------------|  6. OMC 无更多命令
 |  (断开连接)                             |  7. 会话结束
```

**输入参数**：`ObjectName` — 新实例的对象集合路径
**返回**：
- `instanceNumber` — 新建对象的实例号码
- `status` — 已建立 或 已提交（需重启生效）

**备注**：OMC 可多次重复调用 AddObject。

---

### A.5 配置管理对象删除

**触发方**：OMC
**核心RPC**：`DeleteObject` / `DeleteObjectResponse`
**场景**：OMC 在 CPE 上删除多实例配置参数对象

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform (6 CONN REQ) ------------->|  1. CPE 发起 Inform
 |  <-- InformResponse -------------------|  2. OMC 返回 InformResponse
 |  --- Empty HTTP POST ----------------->|  3. CPE 发送空 POST
 |  <-- DeleteObject ---------------------|  4. OMC 下发 DeleteObject
 |  --- DeleteObjectResponse ------------>|  5. CPE 返回删除结果
 |  <-- Empty HTTP Response --------------|  6. OMC 无更多命令
 |  (断开连接)                             |  7. 会话结束
```

**输入参数**：`ObjectName` — 欲删除的对象实例路径名
**返回**：`status` — 已删除 或 已提交（需重启生效）

**备注**：OMC 可多次重复调用 DeleteObject。

---

### A.6 配置参数变更上报

**触发方**：CPE
**核心RPC**：`Inform` / `InformResponse`
**EventCode**：`4 VALUE CHANGE`
**场景**：CPE 上的配置参数值发生变化（非 OMC 修改导致），CPE 主动向 OMC 上报

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform (4 VALUE CHANGE) --------->|  1. CPE 发起 Inform，ParameterList 中
 |                                         |     列出所有变更的参数值
 |  <-- InformResponse -------------------|  2. OMC 认证设备后返回 InformResponse
 |  --- Empty HTTP POST ----------------->|  3. CPE 发送空 POST
 |  <-- Empty HTTP Response --------------|  4. OMC 无命令，结束会话
 |  (断开连接)                             |  5. 会话结束
```

**通知属性规则**：
- **主动通知**：参数变化后立即发起到 OMC 的连接（不影响当前会话）
- **被动通知**：等待下一次 Inform 消息（如周期性 Inform）时携带

**备注**：
- EventCode `103 ADD OBJECT` — 配置管理对象新增上报（非 OMC 的 AddObject 触发）
- EventCode `104 DELETE OBJECT` — 配置管理对象删除上报（非 OMC 的 DeleteObject 触发）
- 一个 Inform 报文中只能携带一种变化类型的通知

---

### A.7 配置参数属性查询

**触发方**：OMC
**核心RPC**：`GetParameterAttributes` / `GetParameterAttributesResponse`
**场景**：OMC 查询 CPE 上的配置参数属性（通知类型、权限列表）

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform (6 CONN REQ) ------------->|  1. CPE 发起 Inform
 |  <-- InformResponse -------------------|  2. OMC 返回 InformResponse
 |  --- Empty HTTP POST ----------------->|  3. CPE 发送空 POST
 |  <-- GetParameterAttributes -----------|  4. OMC 下发 GetParameterAttributes
 |  --- GetParameterAttributesResponse -->|  5. CPE 返回参数属性结果
 |  <-- Empty HTTP Response --------------|  6. OMC 无更多命令
 |  (断开连接)                             |  7. 会话结束
```

**输入参数**：`parameterNames` — 待查询的参数名称列表
**返回**：`parameterList` — 参数名、通知类型、权限列表

**备注**：OMC 可多次重复调用 GetParameterAttributes。

---

### A.8 配置参数属性修改

**触发方**：OMC
**核心RPC**：`SetParameterAttributes` / `SetParameterAttributesResponse`
**场景**：OMC 修改 CPE 上的配置参数属性（设置通知策略）

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform (6 CONN REQ) ------------->|  1. CPE 发起 Inform
 |  <-- InformResponse -------------------|  2. OMC 返回 InformResponse
 |  --- Empty HTTP POST ----------------->|  3. CPE 发送空 POST
 |  <-- SetParameterAttributes -----------|  4. OMC 下发 SetParameterAttributes
 |  --- SetParameterAttributesResponse -->|  5. CPE 返回确认
 |  <-- Empty HTTP Response --------------|  6. OMC 无更多命令
 |  (断开连接)                             |  7. 会话结束
```

**输入参数**：`parameterList` — 参数名、通知类型是否更改、通知类型值、权限列表是否更改、权限列表
**返回**：无

**备注**：OMC 可多次重复调用 SetParameterAttributes。

---

## 4. 性能管理交互流程（A.9 — A.11, A.32）

### A.9 性能测量任务定制

**触发方**：OMC
**核心RPC**：`SetParameterValues` / `SetParameterValuesResponse`
**场景**：OMC 设置 CPE 的性能文件上传周期等参数

```
CPE                                     OMC(ACS)
 |                                         |
 |  (已建立 TR069 会话)                     |
 |  <-- SetParameterValues ---------------|  1. OMC 下发性能任务参数
 |  --- SetParameterValuesResponse ------>|  2. CPE 确认设置结果
```

**关键参数**：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| Enable | boolean | 开启/关闭周期性上传 |
| Alias | string(64) | 别名，OMC 识别实例 |
| URL | string(256) | 目标文件服务器地址 |
| Username | string(256) | 文件服务器认证用户名 |
| Password | string(256) | 文件服务器认证密码 |
| PeriodicUploadInterval | unsignedInt | 上传间隔（秒） |
| PeriodicUploadTime | dateTime | 周期上传的时间基准点(UTC) |

**性能要求**：

| 采集粒度 | 最大上传时延 |
|---------|------------|
| 5分钟 | ≤1分钟 |
| 15分钟 | ≤10分钟 |
| 30分钟 | ≤15分钟 |
| 60分钟 | ≤30分钟 |

---

### A.10 文件上传（性能文件到文件服务器）

**触发方**：CPE（自动周期触发）
**协议**：HTTP PUT/POST（非 TR069 RPC）
**场景**：CPE 按周期将性能数据文件上传到 OMC 指定的文件服务器

```
CPE                                     FileServer
 |                                         |
 |  --- HTTP PUT/POST (PM文件) ---------->|  CPE 周期性上传性能文件
 |  <-- HTTP 200 OK ----------------------|
```

**备注**：
- 上传失败时 CPE 应至少重传 3 次
- 未能上传的文件应至少保存 24 小时
- OMC 北向应保存原始性能数据不少于 7 天

---

### A.11 文件上传结果通知

**触发方**：CPE
**核心RPC**：`Inform` + `AutonomousTransferComplete` / `AutonomousTransferCompleteResponse`
**场景**：CPE 完成性能文件上传后，向 OMC 通报上传结果

```
CPE                                     OMC(ACS)
 |                                         |
 |  (已建立 TR069 会话)                     |
 |  --- Inform --------------------------->|  1. CPE 上报文件上传成功/失败
 |  <-- InformResponse --------------------|  2. OMC 响应
 |  --- AutonomousTransferComplete ------->|  3. CPE 发送自动传输完成通知
 |  <-- AutonomousTransferCompleteResp ----|  4. OMC 响应
```

**AutonomousTransferComplete 参数**：
- `TransferURL` — 上传文件目的地址
- `FileSize` — 上传文件大小
- `FaultStruct` — 错误码及错误信息
- `StartTime` — 上传开始时间
- `CompleteTime` — 上传结束时间

---

### A.32 性能数据文件补采

**触发方**：OMC
**核心RPC**：`SetParameterValues` + `AutonomousTransferComplete`
**场景**：OMC 发现某时段性能文件缺失或损坏，要求 CPE 重新上传

```
CPE                                     OMC(ACS)             FileServer
 |                                         |                     |
 |  --- Inform (6 CONN REQ) ------------->|                     |  1.  CPE 发起 Inform
 |  <-- InformResponse -------------------|                     |  2.  OMC 返回
 |  --- Empty HTTP POST ----------------->|                     |  3.  CPE 空 POST
 |  <-- SetParameterValues ---------------|                     |  4.  OMC 下发补采参数
 |  --- SetParameterValuesResponse ------>|                     |     （补采开关=true）
 |  <-- ... 其他交互 ... -----------------|                     |  5-6.
 |                                         |                     |
 |  --- HTTP PUT/POST (PM文件) -------------------------------->|  7.  CPE 上传补采文件
 |                                         |                     |
 |  --- Inform --------------------------->|                     |  8.  CPE 发起新会话
 |  <-- InformResponse --------------------|                     |  9.  OMC 认证响应
 |  --- AutonomousTransferComplete ------->|                     |  10. 上报上传结果
 |  <-- AutonomousTransferCompleteResp ----|                     |  11. OMC 响应
 |  --- Empty HTTP POST ------------------>|                     |  12.
 |  <-- Empty HTTP Response ----------------|                     |  13.
 |  (断开连接)                              |                     |  14.
```

**补采参数**：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| 补采开关 | boolean | true=开始补采，完成后自动置为 false |
| 补采开始时间 | dateTime | 需补采文件的起始采集时间(UTC) |
| 补采结束时间 | dateTime | 需补采文件的结束采集时间(UTC) |

**备注**：多个补采文件时，可重复步骤 7-14，或批量上传后统一执行 8-14（其中 10-11 重复多次）。

---

## 5. 故障管理交互流程（A.12 — A.13）

### A.12 实时告警上报

**触发方**：CPE
**核心RPC**：`Inform` / `InformResponse`
**场景**：CPE 发生告警时，根据预定义的告警报告机制主动向 OMC 上报

```
CPE                                     OMC(ACS)
 |                                         |
 |  (已建立 TR069 会话)                     |
 |  --- Inform (告警信息) ---------------->|  1. CPE 通过 Inform 上报告警
 |  <-- InformResponse -------------------|  2. OMC 响应确认
```

**告警信息格式**：

| 字段 | 数据类型 | 说明 |
|------|---------|------|
| 告警标识 | string | 唯一标识 |
| 告警源定位信息 | string | MOI 的 DN 表示 |
| 告警级别 | string | Critical / Major / Minor / Warning |
| 告警类型 | string | 通讯/环境/设备/处理错误/服务质量告警 |
| 通知类型 | string | NewAlarm / ClearedAlarm / ChangedAlarm |
| 告警原因 | string | |
| 告警事件发生时间 | dateTime | 发生/改变/清除的时间 |
| 告警描述 | string | |
| 告警附加信息 | string | 映射到北向 AlarmText 字段 |
| 告警附加文本 | string | 厂家告警编码 |

---

### A.13 告警同步

**触发方**：OMC
**核心RPC**：`GetParameterValues` / `GetParameterValuesResponse`
**场景**：OMC 主动查询 CPE 的告警参数信息，确保 OMC 与 CPE 告警信息一致

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform (6 CONN REQ) ------------->|  1. CPE 发起 Inform
 |  <-- InformResponse -------------------|  2. OMC 返回 InformResponse
 |  --- Empty HTTP POST ----------------->|  3. CPE 发送空 POST
 |  <-- GetParameterValues ---------------|  4. OMC 查询告警参数
 |  --- GetParameterValuesResponse ------>|  5. CPE 返回告警参数结果
 |  <-- Empty HTTP Response --------------|  6. OMC 无更多命令
 |  (断开连接)                             |  7. 会话结束
```

**备注**：OMC 可多次重复调用 GetParameterValues 查询不同告警参数。

---

## 6. 维护管理交互流程（A.14 — A.29）

### A.14 设备在线状态通知

**触发方**：CPE
**核心RPC**：`Inform` / `InformResponse`
**EventCode**：`2 PERIODIC`
**场景**：CPE 周期性上报心跳消息，通知 OMC 设备在线

```
CPE                                     OMC(ACS)
 |                                         |
 |  (已建立 TR069 会话)                     |
 |  --- Inform (2 PERIODIC) ------------->|  1. CPE 发送周期性 Inform
 |  <-- InformResponse -------------------|  2. OMC 响应
```

**备注**：OMC 通过心跳超时检测设备离线。ACS 引擎应设置 `acs:heartbeat:{device_serial}` TTL = 2 × inform_interval。

---

### A.15 设备重启

**触发方**：OMC
**核心RPC**：`Reboot` / `RebootResponse`
**场景**：OMC 通知 CPE 设备进行重启

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform (6 CONN REQ) ------------->|  1.  CPE 发起 Inform
 |  <-- InformResponse -------------------|  2.  OMC 返回 InformResponse
 |  --- Empty HTTP POST ----------------->|  3.  CPE 发送空 POST
 |  <-- Reboot ---------------------------|  4.  OMC 下发 Reboot
 |  --- RebootResponse ------------------>|  5.  CPE 响应确认
 |  <-- Empty HTTP Response --------------|  6.  OMC 无更多命令
 |  (断开连接)                             |  7.  会话结束
 |                                         |
 |  *** CPE 执行重启 ***                    |
 |                                         |
 |  --- Inform (M Reboot) --------------->|  8.  重启后 CPE 重新上报 Inform
 |  <-- InformResponse -------------------|  9.  OMC 认证后响应
 |  --- Empty HTTP POST ----------------->|  10. CPE 无请求
 |  <-- Empty HTTP Response --------------|  11. OMC 无命令
 |  (断开连接)                             |  12. 新会话结束
```

---

### A.16 设备重启结果通知

**触发方**：CPE
**核心RPC**：`Inform` / `InformResponse`
**EventCode**：`M Reboot`
**场景**：CPE 重启后，通过 Inform 通知 OMC 重启完成

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform (M Reboot) --------------->|  1. CPE 重启后上报 Inform
 |  <-- InformResponse -------------------|  2. OMC 响应
```

**备注**：此流程通常包含在 A.15 设备重启的步骤 8-12 中。

---

### A.17 设备软件升级（强制性）

**触发方**：OMC
**核心RPC**：`GetParameterValues` + `Download` + `TransferComplete`
**场景**：OMC 强制升级 CPE 软件版本，全过程无需用户干预

```
CPE                                     OMC(ACS)             FileServer
 |                                         |                     |
 |  --- Inform (6 CONN REQ) ------------->|                     |  1.  CPE 发起 Inform
 |  <-- InformResponse -------------------|                     |  2.  OMC 响应
 |  --- Empty HTTP POST ----------------->|                     |  3.  CPE 空 POST
 |  <-- GetParameterValues ---------------|                     |  4.  OMC 查询软件版本(可选)
 |  --- GetParameterValuesResponse ------>|                     |  5.  CPE 返回版本号(可选)
 |  <-- Download -------------------------|                     |  6.  OMC 下发 Download
 |  --- DownloadResponse ---------------->|                     |  7.  CPE 确认
 |  <-- ... 其他交互 ... -----------------|                     |  8.
 |                                         |                     |
 |  --- HTTP GET (固件文件) ----------------------------------->|  ←  CPE 从文件服务器下载
 |  *** CPE 本地升级（可能重启）***          |                     |
 |                                         |                     |
 |  --- Inform --------------------------->|                     |  9.  CPE 发起新会话
 |  <-- InformResponse --------------------|                     |  10. OMC 认证响应
 |  --- TransferComplete ----------------->|                     |  11. CPE 上报升级结果
 |  <-- TransferCompleteResponse ----------|                     |  12. OMC 响应
 |  --- Empty HTTP POST ------------------>|                     |  13.
 |  <-- Empty HTTP Response ----------------|                     |  14.
 |  (断开连接)                              |                     |  15. 会话结束
```

**Download 参数**：
- `fileType` — 文件类型（固件包）
- `URL` — HTTP 下载地址
- `userName` / `password` — 认证信息
- `fileSize` — 文件大小
- `delaySeconds` — 延时等待时间（秒）

**备注**：
- 若 OMC 已通过其他方式获得版本号，步骤 4-5 可跳过
- 若升级完成时会话仍在，跳过步骤 9-10，直接从 11 开始
- CPE 需具备版本回退保护机制

---

### A.18 设备软件下载

**触发方**：CPE（自动触发）
**协议**：HTTP/HTTPS GET（非 TR069 RPC）
**场景**：CPE 从文件服务器下载软件升级文件

```
CPE                                     FileServer
 |                                         |
 |  --- HTTP GET (软件文件) ------------->|  CPE 下载升级文件
 |  <-- 软件文件 ------------------------|
```

---

### A.19 设备软件下载完成通知

**触发方**：CPE
**核心RPC**：`Inform` / `InformResponse`
**EventCode**：`7 TRANSFER COMPLETE`、`M Download`
**场景**：CPE 软件下载完成后，通过 Inform 通知 OMC 并上报新版本号

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform (7 TRANSFER COMPLETE, ---->|  CPE 上报下载完成 + 新版本号
 |        M Download)                      |
 |  <-- InformResponse -------------------|
```

---

### A.20 设备单元软件升级结果通知

**触发方**：CPE
**核心RPC**：`Inform` / `InformResponse`
**EventCode**：`4 VALUE CHANGE`
**场景**：扩展型 CPE 上报各单元的软件升级结果（成功/失败+原因）

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform (4 VALUE CHANGE) --------->|  1. CPE 发起连接，ParameterList 中
 |                                         |     列出各单元升级结果参数值
 |  <-- InformResponse -------------------|  2. OMC 认证后响应
 |  --- Empty HTTP POST ----------------->|  3. CPE 空 POST
 |  <-- Empty HTTP Response --------------|  4. OMC 无命令
 |  (断开连接)                             |  5. 会话结束
```

**单元升级结果参数**：

| 字段 | 类型 | 说明 |
|------|------|------|
| 升级状态 | boolean | 单元升级成功/失败 |
| 失败原因 | string | 升级失败原因（成功时可选） |

---

### A.21 设备软件升级完成通知

**触发方**：CPE
**核心RPC**：`Inform` / `InformResponse`
**场景**：所有单元升级完成后，CPE 通知 OMC 整站升级完成

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform --------------------------->|  1. CPE 发起连接，若之前有未上报的
 |                                         |     单元升级结果则在 ParameterList 中携带
 |  <-- InformResponse --------------------|  2. OMC 认证后响应
 |  --- Empty HTTP POST ------------------>|  3.
 |  <-- Empty HTTP Response ----------------|  4.
 |  (断开连接)                              |  5. 会话结束
```

---

### A.22 查询设备单元软件升级进度

**触发方**：OMC
**核心RPC**：`GetParameterValues` / `GetParameterValuesResponse`
**场景**：OMC 在升级过程中查询 CPE 各单元的当前升级阶段

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform (6 CONN REQ) ------------->|  1. CPE 发起 Inform
 |  <-- InformResponse -------------------|  2. OMC 返回 InformResponse
 |  --- Empty HTTP POST ----------------->|  3. CPE 发送空 POST
 |  <-- GetParameterValues ---------------|  4. OMC 查询升级进度参数
 |  --- GetParameterValuesResponse ------>|  5. CPE 返回升级进度
 |  <-- Empty HTTP Response --------------|  6. OMC 无更多命令
 |  (断开连接)                             |  7. 会话结束
```

**升级阶段**（数字型）：初始化 → 下载 → 安装 → 等待激活 → 激活

**备注**：OMC 可多次重复调用，查询多个单元的进度。

---

### A.23 设备恢复出厂设置

**触发方**：OMC
**核心RPC**：`FactoryReset` / `FactoryResetResponse`
**场景**：OMC 要求 CPE 恢复出厂设置

```
CPE                                     OMC(ACS)
 |                                         |
 |  --- Inform (6 CONN REQ) ------------->|  1. CPE 发起 Inform
 |  <-- InformResponse -------------------|  2. OMC 返回 InformResponse
 |  --- Empty HTTP POST ----------------->|  3. CPE 发送空 POST
 |  <-- FactoryReset ---------------------|  4. OMC 下发 FactoryReset
 |  --- FactoryResetResponse ------------>|  5. CPE 返回成功/失败
 |  <-- Empty HTTP Response --------------|  6. OMC 无更多命令
 |  (断开连接)                             |  7. 会话结束
```

> 注：规范原文拼写为 `FacotryReset`，实际实现应使用标准 TR069 拼写 `FactoryReset`。

---

### A.24 设备日志任务定制

**触发方**：OMC
**核心RPC**：`SetParameterValues` / `SetParameterValuesResponse`
**场景**：OMC 设置 CPE 的日志文件上传周期

```
CPE                                     OMC(ACS)
 |                                         |
 |  (已建立 TR069 会话)                     |
 |  <-- SetParameterValues ---------------|  1. OMC 下发日志上传参数
 |  --- SetParameterValuesResponse ------>|  2. CPE 确认设置
```

**关键参数**：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| PeriodicUploadEnable | boolean | 开启/关闭周期性日志上传 |
| URL | string(256) | 目标文件服务器地址 |
| Username | string(256) | 认证用户名 |
| Password | string(256) | 认证密码 |
| PeriodicUploadInterval | unsignedInt | 上传间隔（秒） |

**备注**：默认上报周期为 1 小时，CPE 上传该时段内的增量日志。日志达到容量上限时应立即上传并开启新周期。

---

### A.25 文件下载

**触发方**：CPE（自动触发）
**协议**：HTTP/HTTPS GET（非 TR069 RPC）
**场景**：CPE 从文件服务器下载指定文件

```
CPE                                     FileServer
 |                                         |
 |  --- HTTP GET (文件) ----------------->|  CPE 下载文件
 |  <-- 文件内容 -------------------------|
```

---

### A.26 文件下载结果通知

**触发方**：CPE
**核心RPC**：`Inform` + `TransferComplete` / `TransferCompleteResponse`
**场景**：CPE 完成文件下载后，向 OMC 通报下载结果

```
CPE                                     OMC(ACS)
 |                                         |
 |  (已建立 TR069 会话)                     |
 |  --- Inform --------------------------->|  1. CPE 上报下载成功/失败
 |  <-- InformResponse --------------------|  2. OMC 响应
 |  --- TransferComplete ----------------->|  3. CPE 发送传输完成通知
 |  <-- TransferCompleteResponse ----------|  4. OMC 响应
```

**TransferComplete 参数**：
- `FaultStruct` — 错误码及错误信息
- `StartTime` — 下载开始时间
- `CompleteTime` — 下载结束时间

---

### A.27 日志文件获取

**触发方**：OMC
**核心RPC**：`Upload` / `UploadResponse`
**场景**：OMC 下发 Upload 命令，要求 CPE 将日志文件上传到指定文件服务器

```
CPE                                     OMC(ACS)             FileServer
 |                                         |                     |
 |  (已建立 TR069 会话)                     |                     |
 |  <-- Upload (日志文件) -----------------|                     |
 |  --- UploadResponse ------------------>|                     |
 |                                         |                     |
 |  --- HTTP PUT/POST (日志文件) ------------------------------>|  CPE 上传到文件服务器
```

**Upload 参数**：
- `fileType` — 文件类型
- `URL` — 上传目标地址
- `userName` / `password` — 认证信息

---

### A.28 设备信息文件上传

**触发方**：OMC
**核心RPC**：`Upload` + `TransferComplete`
**场景**：OMC 要求 CPE 将设备信息文件（如参数配置文件）上传到指定地址

```
CPE                                     OMC(ACS)             FileServer
 |                                         |                     |
 |  --- Inform (6 CONN REQ) ------------->|                     |  1.  CPE 发起 Inform
 |  <-- InformResponse -------------------|                     |  2.  OMC 响应
 |  --- Empty HTTP POST ----------------->|                     |  3.  CPE 空 POST
 |  <-- Upload ----------------------------|                     |  4.  OMC 下发 Upload
 |  --- UploadResponse ------------------->|                     |  5.  CPE 确认
 |  <-- ... 其他交互 ... ------------------|                     |  6.
 |                                         |                     |
 |  --- HTTP PUT/POST (信息文件) -------------------------------->|  CPE 上传到文件服务器
 |                                         |                     |
 |  --- Inform --------------------------->|                     |  7.  CPE 发起新会话(若需)
 |  <-- InformResponse --------------------|                     |  8.  OMC 响应
 |  --- TransferComplete ----------------->|                     |  9.  CPE 上报上传结果
 |  <-- TransferCompleteResponse ----------|                     |  10. OMC 响应
 |  --- Empty HTTP POST ------------------>|                     |  11.
 |  <-- Empty HTTP Response ----------------|                     |  12.
 |  (断开连接)                              |                     |  13. 会话结束
```

**Upload 参数**：
- `fileType` — 文件类型
- `URL` — 上传目标地址
- `userName` / `password` — 认证信息
- `delaySeconds` — 延时等待时间（秒）

**备注**：若上传完成时会话仍在，跳过步骤 7-8，直接从步骤 9 开始。

---

### A.29 设备信息文件下载（可选）

**触发方**：OMC
**核心RPC**：`Download` + `TransferComplete`
**场景**：OMC 要求 CPE 从指定地址下载设备信息文件

```
CPE                                     OMC(ACS)             FileServer
 |                                         |                     |
 |  --- Inform (6 CONN REQ) ------------->|                     |  1.  CPE 发起 Inform
 |  <-- InformResponse -------------------|                     |  2.  OMC 响应
 |  --- Empty HTTP POST ----------------->|                     |  3.  CPE 空 POST
 |  <-- Download -------------------------|                     |  4.  OMC 下发 Download
 |  --- DownloadResponse ---------------->|                     |  5.  CPE 确认
 |  <-- ... 其他交互 ... -----------------|                     |  6.
 |                                         |                     |
 |  --- HTTP GET (信息文件) ----------------------------------->|  CPE 从文件服务器下载
 |                                         |                     |
 |  --- Inform --------------------------->|                     |  7.  CPE 发起新会话(若需)
 |  <-- InformResponse --------------------|                     |  8.  OMC 响应
 |  --- TransferComplete ----------------->|                     |  9.  CPE 上报下载结果
 |  <-- TransferCompleteResponse ----------|                     |  10. OMC 响应
 |  --- Empty HTTP POST ------------------>|                     |  11.
 |  <-- Empty HTTP Response ----------------|                     |  12.
 |  (断开连接)                              |                     |  13. 会话结束
```

**Download 参数**：
- `fileType` — 文件类型
- `URL` — HTTP 下载地址
- `userName` / `password` — 认证信息
- `fileSize` — 文件大小
- `delaySeconds` — 延时等待时间（秒）

**备注**：此流程为可选实现。若下载完成时会话仍在，跳过步骤 7-8。

---

## 7. MR管理交互流程（A.30 — A.31，仅适用于4/5G皮基站）

### A.30 MR测量任务定制

**触发方**：OMC
**核心RPC**：`SetParameterValues` / `SetParameterValuesResponse`
**场景**：OMC 设置 CPE 的 MR 数据文件上传周期

```
CPE                                     OMC(ACS)
 |                                         |
 |  (已建立 TR069 会话)                     |
 |  <-- SetParameterValues ---------------|  1. OMC 下发 MR 任务参数
 |  --- SetParameterValuesResponse ------>|  2. CPE 确认设置
```

**备注**：参数结构与性能测量任务定制（A.9）类似，包含 Enable、URL、Username、Password、PeriodicUploadInterval 等。

---

### A.31 上传MR数据文件结果通知

**触发方**：CPE
**核心RPC**：`Inform` + `AutonomousTransferComplete` / `AutonomousTransferCompleteResponse`
**场景**：CPE 完成 MR 数据文件上传后，向 OMC 通报结果

```
CPE                                     OMC(ACS)
 |                                         |
 |  (已建立 TR069 会话)                     |
 |  --- Inform --------------------------->|  1. CPE 上报 MR 文件上传成功/失败
 |  <-- InformResponse --------------------|  2. OMC 响应
 |  --- AutonomousTransferComplete ------->|  3. CPE 发送自动传输完成通知
 |  <-- AutonomousTransferCompleteResp ----|  4. OMC 响应
```

---

## 8. ACS 引擎开发参考

### 8.1 RPC 方法使用频率分析

| RPC 方法 | 使用次数 | 涉及流程 |
|----------|---------|---------|
| **Inform / InformResponse** | 全部流程 | A.1-A.32（所有会话建立） |
| **GetParameterValues** | 5 | A.2, A.13, A.17, A.22 + 复用场景 |
| **SetParameterValues** | 5 | A.3, A.9, A.24, A.30, A.32 |
| **Download** | 3 | A.17, A.29 + 通用文件下载 |
| **Upload** | 3 | A.27, A.28 + 通用文件上传 |
| **TransferComplete** | 4 | A.17, A.26, A.28, A.29 |
| **AutonomousTransferComplete** | 3 | A.11, A.31, A.32 |
| **GetParameterNames** | 1 | A.1 |
| **AddObject** | 1 | A.4 |
| **DeleteObject** | 1 | A.5 |
| **GetParameterAttributes** | 1 | A.7 |
| **SetParameterAttributes** | 1 | A.8 |
| **Reboot** | 1 | A.15 |
| **FactoryReset** | 1 | A.23 |

### 8.2 会话模式分类

| 模式 | 流程 | ACS 引擎关注点 |
|------|------|---------------|
| **OMC→CPE 单次 RPC** | A.1-A.5, A.7-A.8, A.13, A.15, A.22-A.23 | 命令队列 Pop → 构建 RPC → 解析 Response |
| **OMC→CPE 多步 RPC** | A.17 (GPV+Download) | 命令队列支持有序多命令 |
| **CPE→OMC 纯上报** | A.6, A.12, A.14, A.16, A.19-A.21 | Inform 解析 → EventBus 分发 |
| **CPE→OMC 上报+RPC** | A.11, A.26, A.31 | Inform + TransferComplete/ATC 处理 |
| **任务定制（会话内 SPV）** | A.9, A.24, A.30, A.32 | SetParameterValues 作为会话内操作 |
| **文件传输协调** | A.10, A.17-A.18, A.25, A.27-A.29 | Upload/Download + 文件服务器 + 传输结果回调 |

### 8.3 与 ACS 引擎设计的对应关系

| 本文档内容 | ACS 引擎设计（DD-07） |
|-----------|---------------------|
| 通用会话模式（§1.2） | 会话状态机 `session.go`（§3.2） |
| OMC 主动发起模式 | Connection Request `connreq/client.go`（§3.5） |
| 所有 ACS→CPE 的 RPC | RPCDispatcher + 各 `rpc/*.go`（§3.4） |
| Inform 处理（A.6, A.12, A.14, A.16...） | `handleInform()`（§3.3） |
| TransferComplete / ATC | `handleTransferComplete()` + EventBus |
| 命令队列（多 RPC 编排） | CommandQueue `cmdqueue/queue.go`（§2.3） |
| 文件传输（Upload/Download） | `rpc/upload.go`, `rpc/download.go` + MinIO |

### 8.4 开发优先级建议

1. **Phase 1（基础）**：Inform 解析 + InformResponse + 会话状态机 + GetParameterValues
2. **Phase 2（核心 RPC）**：SetParameterValues, GetParameterNames, AddObject, DeleteObject, Reboot, FactoryReset
3. **Phase 3（文件传输）**：Download, Upload, TransferComplete, AutonomousTransferComplete
4. **Phase 4（完整）**：GetParameterAttributes, SetParameterAttributes, Connection Request, 命令队列编排
