# OMC 日志收集功能逻辑梳理

> 供重构参考，重点说明：校验逻辑、各类型后台处理流程、设备 TR-069 报文交互流程。

---

## 一、三种日志类型说明

| 日志类型 | logType 值 | 触发机制 | 周期单位 | 文件存储目录 |
|---------|-----------|---------|---------|------------|
| 设备运行日志 | `deviceLog` | 立即或周期 | 分钟 | `.../log/` |
| 安全日志 | `securityLog` | 立即或周期 | **小时** | `.../security_log/` |
| 周期日志（eNB专属） | — | 周期 | 分钟 | `.../log/` |

安全日志不是独立流程，与设备日志共用同一套任务表和收集流程，区别仅在于：
- `reportPeriod` 前端展示以小时为单位，存库和下发设备的是秒，需双向转换
- 安全日志仅部分平台支持（高通 QAV3/QAV4），其他平台直接标记任务为失败
- 安全日志上报时 `fileType` 参数为 `SECURITY_LOG`，普通日志为 `LOG`

---

## 二、任务状态说明（task_status）

| 状态值 | 含义 |
|--------|------|
| 0 | 未开始（已创建任务，未开始下发） |
| 1 | 正在收集中（SetParam 或 Upload 命令已下发） |
| 2 | 收集完成（周期上报已按计划结束） |
| 3 | 收集失败（超时、设备返回错误、设备离线等） |
| 4 | 收集终止（用户手动停止） |
| 5 | 等待文件上传（立即收集：Upload 命令已响应，等待文件到达服务器） |
| 6 | 已结束（周期上报：用户终止，SetParam 成功） |
| 7 | 停止失败（周期上报：停止 SetParam 返回错误） |
| 8 | 等待重启（高通 DXDF 平台，SetParam 成功后需重启生效） |

---

## 三、立即收集日志

### 3.1 前端校验与入口逻辑

用户发起立即收集时，前端提交以下核心参数：
- `serial_number`：设备序列号（逗号分隔，支持批量）
- `device_code`：设备编码
- `device_type`：`eNB` / `EGW` / `CPE`
- `isGnb`：是否 5G 基站（`1`/`0`）
- `logType`：日志类型，`deviceLog` 或 `securityLog`
- `execute_type`：执行方式，`Immediately`（立即）或 `timing`（CPE 定时）
- `reportPeriod`：安全日志专用，前端单位为**小时**
- `start_time` / `end_time`：CPE 定时收集时使用

**Web 层唯一前置校验**：
- 检查 Redis 中 `disk_usage:run_log` 字段，若磁盘使用率 ≥ 90%，直接拒绝，返回错误码 `901`

通过后，透传所有参数（含 `login_user` 对象）通过 Feign 调用到 CELLHANDLER 服务。

---

### 3.2 CELLHANDLER 后台校验（核心）

CELLHANDLER 是真正校验和任务创建的地方，依次执行以下校验：

**① 重复任务校验**

查询 `log_device_report` 表，找出当前设备是否有状态为进行中（`0`、`1`）的同类型任务（eNB 同时匹配 `log_type`）。若存在，直接返回错误码 `401`，提示"任务进行中或未完成"。

**② 安全日志 reportPeriod 单位转换**

若 `logType == "securityLog"`，将前端传来的**小时**数转换为**秒**：`reportPeriod × 60 × 60`，后续全程以秒为单位。

**③ 文件数量上限校验**

从系统配置（`CollectMaxLogFileNum`）取最大日志文件数，然后查询每个设备已有的日志文件总数（立即收集文件 + 周期日志文件之和）。若超过上限，返回错误码 `901`，列出超限设备的序列号。

> 注意：**安全日志跳过此项校验**，不受文件数量限制。

**④ 任务记录写入数据库**

- 查询数据库，找出同一设备已存在的任务
- 若任务已结束（状态 2/3/4/6/7/8），则**复用**该任务记录（更新状态为 0，清空错误信息）
- 若不存在，则新建任务记录，插入 `log_device_report` 表，字段包括：`operator_code`、`task_id`（UUID）、`device_type`、`device_code`、`serial_number`、`task_status=0`、`execute_type`、`report_period`、`start_time`、`end_time`、`log_type`

**⑤ 投递消息队列，触发异步下发**

将 `methodName = "immediateCollectLogFile"` 的指令放入**实时队列**（`QueueActionMethodCall.putRealTimeMethod`），携带参数：
- `taskIdList`：任务 ID 与设备 CODE 的映射列表
- `deviceType`：设备类型
- `userCode`：操作用户
- `isGnb`：是否 5G
- `executeType`：`cellDeviceLog`

> CPE 定时收集特殊：不直接投队列，而是通过 Quartz 创建定时 Job，到时间后再投队列。

---

### 3.3 MethodExecuteHandler 下发线程（TR-069 报文交互）

队列被 MethodExecuteHandler 消费，根据 `TaskThreadCount` 配置启动多个工作线程，每个线程从任务队列中取任务处理。

**Step 1：终止标志检查**

检查内存中是否存在该 taskId 的终止标志（`ImmediateCollectLogTaskFlag`），若有则直接将任务标记为终止（状态 4），跳过。

**Step 2：操作权限校验**

根据 `userCode` 和功能码（4G 基站 / 5G 基站 / CPE / EGW 各有不同功能码）校验用户是否有操作该设备的权限，无权限则标记失败（状态 3）。

**Step 3：设备在线校验**

从 Redis 设备缓存判断设备是否在线，离线则标记失败（状态 3），错误信息 `CollectLogSheBeiBuZaiXian`。

**Step 4：下发 Upload 命令（TR-069 Upload RPC）**

更新任务状态为 1（正在收集），根据设备类型和平台类型构造上传 URL：

```
普通/5G 基站：  {fileuploadRootUrl}/FileUploadService?fileType=LOG&taskId={taskId}&filename=
DXDF 平台：    {fileuploadRootUrl}/YDFileUploadService/upload/log/logFile?fileType=LOG&taskId={taskId}&filename=
CPE：          {fileuploadPath}/FileUploadService?fileType=LOG&taskId={taskId}&filename={自动生成文件名}
安全日志：     {fileuploadRootUrl}/FileUploadService?fileType=SECURITY_LOG&taskId={taskId}&filename=
```

fileType 参数设置：
- 普通 4G 基站：`"4 Vendor Log File 1,2,3,4"`
- DXDF / CPE / 5G（BaiBNX/BaiBNQ/中国电信）：`"2 Vendor Log File"`

**TR-069 Upload Request 报文（OMC → 设备）：**
```
FileType:   "4 Vendor Log File 1,2,3,4" 或 "2 Vendor Log File"
URL:        上述构造的上传地址
CommandKey: "Collect LOG,{uuid}"
```

**设备回应 Upload Response**（等待超时 = `enbTimeout` 系统配置，单位秒）：
- 无响应 → 超时，状态 3，错误信息 `CollectLogWaitUploadResponseTimeout`
- FaultCode 不为空 → 状态 3，错误信息 `CollectLogReciveUploadReponseFaultStringOne/Two`
- 正常响应 → 继续 Step 5

**Step 5：等待 TransferComplete**

设备收到 Upload 命令后打包日志并上传，上传完成后发送 TransferComplete 通知：

**TR-069 TransferComplete Inform 报文（设备 → OMC）：**
```
CommandKey: "Collect LOG,{uuid}"
FaultCode:  0（成功）或错误码
```

- 无响应 → 超时，状态 3，错误信息 `CollectLogWaitTransfercompleteTimeout`
- FaultCode 不为空 → 状态 3，错误信息 `CollectLogReciveTransferCompleteFaultStringOne/Two`
- 正常响应 → 文件已上传到 OMC 服务器，由 FileUploadServer 接收后写入 `log_device_report_file` 表，更新任务状态为 2

**消息中心更新**（仅 `executeType == "cellDeviceLog"` 时）：
- 下发前：写入消息中心（进行中），将消息中心 ID 存入 Redis
- 超时/失败：更新消息中心为失败
- 成功：FileUploadServer 接收到文件后更新消息中心为成功

---

### 3.4 停止立即收集

**立即停止（execute_type = "Immediately"）**：
- 在内存中为该 taskId 添加终止标志
- 数据库任务状态更新为 4（终止）
- 广播刷新页面消息

**停止周期性上报任务（execute_type != "Immediately"）**：
- 删除可能存在的 Quartz 定时 Job
- 数据库任务状态更新为 4（终止）
- 投递 `periodReportLogFile`，`isEnableReport = "no"` 到队列，向设备下发关闭命令

---

## 四、周期日志

### 4.1 配置周期上报

**CELLHANDLER 后台校验**：
- 磁盘使用率校验：Redis `disk_usage:run_log` ≥ 90% 则拒绝

**时间策略判断**（根据 startTime 与当前时间的比较）：

| 场景 | 处理方式 |
|------|---------|
| startTime > 当前时间 | 创建 Quartz 定时 Job（`CustomizePeriodReportLogJob`），到时执行 |
| startTime ≤ 当前时间（立即） | 直接投递 `periodReportLogFile` 到队列 |
| endTime > 当前时间 | 同时创建结束 Quartz Job，到时间自动发送停止命令（`endReport=yes`） |

数据库同步写入任务记录到 `log_period_report_task` 表（含 `report_period`、`start_time`、`end_time`）。

---

### 4.2 周期下发线程（TR-069 报文交互）

队列被 MethodExecuteHandler 消费，启动多个工作线程处理。

**Step 1：任务状态校验**

查询任务当前状态，若已是终止（5 或 6），跳过。

**Step 2：操作权限校验**

与立即收集相同，按功能码校验用户权限。

**Step 3：设备在线校验**

从 Redis 判断设备是否在线，离线则标记失败（状态 3）。

**Step 4：平台兼容性校验**

判断设备平台类型：
- `Intel_CR`、`436Q`、`NEU430`、`BAIBLQ`、`MLQ`、`MLN`、`BM` 等平台**不支持**周期日志，直接标记失败，错误 `CollectLogSupportedDevice`
- 安全日志仅支持高通 QAV3/QAV4 平台；非此平台的安全日志请求，直接标记失败

**Step 5：获取设备专属 TR-069 参数路径**

根据平台类型，从参数映射缓存查询该平台对应的 TR-069 私有参数路径，需要操作的三个参数：
- 周期上报使能开关
- 日志上传 URL
- 上传间隔（秒）

> 安全日志使用固定的专属参数路径，不走通用参数映射。

**Step 6（可选）：GetParameterValues 查询（仅 DXDF 永鼎平台）**

下发前先查询设备当前参数值，与目标值比较，完全一致则跳过下发（避免重复配置）。

**TR-069 GetParameterValues 报文（OMC → 设备）：**
```
ParameterNames: [使能开关路径, URL路径, 间隔路径]
```

**Step 7：下发 SetParameterValues**

根据启用/停用构造参数值 Map：

```
启用（isEnableReport = "yes"）：
  使能开关 = "1"
  URL      = "{rootPath}/FileUploadService?fileType=LOG&filename="
             （安全日志：fileType=SECURITY_LOG）
             （DXDF：/YDFileUploadService/upload/log/logFile?fileType=LOG&filename=）
  间隔     = {reportPeriod（秒）}
             （安全日志特殊：值再 ÷ 3600 后下发给设备，设备侧接受小时）

停用（isEnableReport = "no"）：
  使能开关 = "0"
```

**TR-069 SetParameterValues 报文（OMC → 设备）：**
```
ParameterList: 上述参数路径与值的键值对
```

**设备回应 SetParameterValuesResponse**：
- 无响应 → 超时，状态 3，错误信息 `CollectLogWaitSetParameterValuesTimeout`
- FaultCode 不为空（启用时）→ 状态 3，`CollectLogReciveSetParameterValuesResponseFaultStringOne/Two`
- FaultCode 不为空（停用时）→ 状态 7（停止失败）
- 成功（启用）→ DXDF 等需重启平台状态 8，其他平台状态 1（进行中）
- 成功（停用，定时结束 endReport=yes）→ 状态 2（已完成）
- 成功（停用，用户手动终止）→ 状态 6（已结束）

**设备主动上报日志**（SetParam 成功后）：

设备按配置的间隔，通过 HTTP PUT 主动上传日志文件到 FileUploadService，OMC 接收后写入 `log_period_report_record` 表。

---

## 五、全链路流程图

### 5.1 立即收集 — 页面交互流程

描述从用户操作到任务创建完成、结果返回页面的完整链路。

```
┌─────────────────────────────────────────────────────────────────┐
│  用户操作层                                                       │
│  选择设备 → 选择日志类型（deviceLog / securityLog）→ 点击"立即收集" │
└──────────────────────────┬──────────────────────────────────────┘
                           │ HTTP POST /cell/collect/goImmediateCollectLogFile
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│  Web 层校验                                                       │
│  ① 磁盘使用率检查：Redis disk_usage:run_log >= 90%               │
│     └─ 是 → 返回 901「磁盘空间不足」，流程终止                    │
│     └─ 否 → 透传参数（含 login_user）→ Feign 调用 CELLHANDLER    │
└──────────────────────────┬──────────────────────────────────────┘
                           │ Feign RPC /enbLog/rpc/goImmediateCollectLogFile
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│  CELLHANDLER 校验 & 任务创建                                      │
│  ① 重复任务检查：同设备同类型 status 0/1 存在                     │
│     └─ 是 → 返回 401「任务进行中」，流程终止                      │
│  ② 安全日志单位转换：reportPeriod 小时 × 3600 → 秒               │
│  ③ 文件数量上限检查：已有文件数 >= CollectMaxLogFileNum           │
│     └─ 是（非安全日志）→ 返回 901「文件数超限」，流程终止          │
│  ④ 写入 log_device_report 表（复用已结束任务 or 新建）           │
│  ⑤ 投递 immediateCollectLogFile 到实时队列                       │
│     └─ 返回 200「操作成功」给 Web 层                             │
└──────────────────────────┬──────────────────────────────────────┘
                           │ 异步（队列消费，不阻塞页面响应）
                           ▼
              MethodExecuteHandler 消费队列
              （详见 5.2 设备交互流程）

        页面轮询任务状态（task_status）
        0=等待 → 1=收集中 → 2=完成 / 3=失败 / 4=终止
```

---

### 5.2 立即收集 — 设备 TR-069 交互流程

描述 MethodExecuteHandler 消费队列后，与单台设备的完整交互过程（每台设备独立线程处理）。

```
队列取出任务（taskId + cellCode）
  │
  ▼
┌─ 前置校验 ──────────────────────────────────────────────────────┐
│  ① 终止标志检查（内存 ImmediateCollectLogTaskFlag）              │
│     └─ 已标记 → 状态 4（终止），跳过此设备                       │
│  ② 操作权限校验（featureCode 用户授权）                          │
│     └─ 无权限 → 状态 3（失败）                                   │
│  ③ 设备在线检查（Redis 设备缓存）                                │
│     └─ 离线 → 状态 3（失败）                                     │
└─────────────────────────────────────────────────────────────────┘
  │ 校验全部通过
  │ 状态 → 1（正在收集），构造上传 URL
  │
  ▼
【第一次报文交互】OMC 主动下发 Upload Request
  │
  │  OMC ──────────────────────────────────► 设备
  │       Upload Request
  │         FileType: "4 Vendor Log File..."
  │              或    "2 Vendor Log File"
  │         URL:  .../FileUploadService?fileType=LOG&taskId=xxx
  │               （安全日志：fileType=SECURITY_LOG）
  │         CommandKey: "Collect LOG,{uuid}"
  │
  │  设备 ──────────────────────────────────► OMC
  │       Upload Response（等待超时 = enbTimeout 秒）
  │
  ├─ 无响应超时 → 状态 3，错误：CollectLogWaitUploadResponseTimeout
  ├─ 返回 FaultCode → 状态 3，错误：CollectLogReciveUploadReponseFaultString
  │
  ▼ 响应成功（设备开始打包并上传日志文件）
  │
【第二次报文交互】等待设备上传完成通知 TransferComplete
  │
  │  设备 ──────────────────────────────────► OMC
  │       TransferComplete Inform
  │         CommandKey: "Collect LOG,{uuid}"
  │         FaultCode:  0（成功）或错误码
  │
  ├─ 无响应超时 → 状态 3，错误：CollectLogWaitTransfercompleteTimeout
  ├─ 返回 FaultCode → 状态 3，错误：CollectLogReciveTransferCompleteFaultString
  │
  ▼ TransferComplete 成功（文件已通过 HTTP 上传至 FileUploadServer）
  │
  │  FileUploadServer 接收文件
  │    → 写入 log_device_report_file 表
  │    → 状态 → 2（收集完成）
  │    → 更新消息中心为成功
  │
  ▼
  页面收到轮询结果，展示"收集完成"及文件列表
```

---

### 5.3 周期日志 — 页面交互流程

描述从用户配置周期参数到任务写库、指令调度的完整链路。

```
┌─────────────────────────────────────────────────────────────────┐
│  用户操作层                                                       │
│  选择设备 → 设置上报周期、起止时间、是否重启 → 点击"定制上报"     │
└──────────────────────────┬──────────────────────────────────────┘
                           │ HTTP POST /cell/collect/customizePeriodReportLogFile
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│  CELLHANDLER 校验 & 调度                                          │
│  ① 磁盘使用率检查：Redis disk_usage:run_log >= 90% → 拒绝        │
│                                                                   │
│  ② 起止时间策略判断：                                             │
│     startTime > 当前时间                                          │
│       └─ 创建 Quartz 开始 Job（到时投队列，isEnableReport=yes）    │
│     startTime <= 当前时间（立即生效）                              │
│       └─ 直接投递 periodReportLogFile 到队列（isEnableReport=yes） │
│     endTime > 当前时间                                            │
│       └─ 创建 Quartz 结束 Job（到时投队列，endReport=yes）         │
│                                                                   │
│  ③ 写入 log_period_report_task 表                                 │
│     → 返回 200「操作成功」                                        │
└──────────────────────────┬──────────────────────────────────────┘
                           │ 异步（队列消费，不阻塞页面响应）
                           ▼
              MethodExecuteHandler 消费队列
              （详见 5.4 设备交互流程）

┌─────────────────────────────────────────────────────────────────┐
│  用户点击"停止上报"                                               │
│  → 删除未触发的 Quartz Job                                        │
│  → 数据库状态 → 4（终止）                                         │
│  → 投递 periodReportLogFile（isEnableReport=no）到队列            │
│     → MethodExecuteHandler 向设备下发关闭参数                     │
└─────────────────────────────────────────────────────────────────┘
```

---

### 5.4 周期日志 — 设备 TR-069 交互流程

描述 MethodExecuteHandler 消费队列后，与单台设备的完整交互过程。

```
队列取出任务（cellCode + isEnableReport + reportPeriod + logType）
  │
  ▼
┌─ 前置校验 ──────────────────────────────────────────────────────┐
│  ① 任务状态检查：status 5/6（已终止）→ 跳过此设备                │
│  ② 操作权限校验（featureCode 用户授权）                          │
│     └─ 无权限 → 状态 3（失败）                                   │
│  ③ 设备在线检查（Redis 设备缓存）                                │
│     └─ 离线 → 状态 3（失败）                                     │
│  ④ 平台兼容性校验：                                              │
│     Intel_CR / 436Q / NEU430 / BAIBLQ / MLQ / MLN / BM 不支持  │
│     └─ 不支持 → 状态 3，错误：CollectLogSupportedDevice          │
│  ⑤ 安全日志额外校验：logType=securityLog 时                      │
│     └─ 非 QAV3/QAV4 平台 → 状态 3，错误：CollectLogSupportedDevice│
└─────────────────────────────────────────────────────────────────┘
  │ 校验全部通过
  │ 从参数映射缓存获取该平台的 TR-069 私有参数路径
  │   （使能开关路径 / URL路径 / 间隔路径）
  │   安全日志使用固定专属路径，不走参数映射
  │
  ├─ [仅 DXDF 永鼎平台] ─────────────────────────────────────────┐
  │                                                               │
  │  【可选报文】OMC 先查询设备当前参数值                           │
  │                                                               │
  │   OMC ────────────────────────────────► 设备                  │
  │        GetParameterValues                                     │
  │          ParameterNames: [使能开关, URL, 间隔]                 │
  │                                                               │
  │   设备 ────────────────────────────────► OMC                  │
  │        GetParameterValuesResponse                             │
  │                                                               │
  │   比对返回值与目标值完全一致 → 跳过下发，流程结束               │
  └───────────────────────────────────────────────────────────────┘
  │
  │ 状态 → 1（进行中） / 状态 → 5（停止中）
  │
  ▼
【核心报文交互】OMC 下发 SetParameterValues
  │
  │  OMC ──────────────────────────────────► 设备
  │       SetParameterValues
  │
  │       启用时（isEnableReport = "yes"）：
  │         使能开关路径 = "1"
  │         URL路径      = .../FileUploadService?fileType=LOG&filename=
  │                       （安全日志：fileType=SECURITY_LOG）
  │                       （DXDF：/YDFileUploadService/upload/log/logFile?...）
  │         间隔路径     = reportPeriod（秒）
  │                       （安全日志：值 ÷ 3600，设备侧接受小时）
  │
  │       停用时（isEnableReport = "no"）：
  │         使能开关路径 = "0"
  │
  │  设备 ──────────────────────────────────► OMC
  │       SetParameterValuesResponse（等待超时 = enbTimeout 秒）
  │
  ├─ 无响应超时 → 状态 3，错误：CollectLogWaitSetParameterValuesTimeout
  │
  ├─ 返回 FaultCode（启用时）
  │    → 状态 3（失败），错误：CollectLogReciveSetParameterValuesResponseFaultString
  │
  ├─ 返回 FaultCode（停用时）
  │    → 状态 7（停止失败）
  │
  └─ 响应成功：
       启用 + 需重启平台（DXDF 等）→ 状态 8（等待重启）
       启用 + 无需重启              → 状态 1（进行中）
       停用 + 定时到期（endReport=yes）→ 状态 2（已完成）
       停用 + 用户手动终止           → 状态 6（已结束）
            │
            ▼（设备端，启用成功后自动运行）
     设备按 Interval 周期，HTTP 主动上传日志文件
       → FileUploadService 接收
       → 写入 log_period_report_record 表
       → 页面文件列表实时刷新
```

---

## 六、关键校验汇总（重构重点）

| 校验项 | 所在层 | 校验逻辑 | 失败处理 |
|-------|--------|---------|---------|
| 磁盘使用率 | Web 层（立即收集）/ CELLHANDLER（周期） | Redis `disk_usage:run_log` >= 90% | 返回 901 |
| 重复任务 | CELLHANDLER（仅立即收集） | 同设备同类型 status 0 或 1 已存在 | 返回 401 |
| 文件数量上限 | CELLHANDLER（仅立即收集） | 已有文件数 >= `CollectMaxLogFileNum`，安全日志豁免 | 返回 901 |
| 任务终止标志 | MethodExecuteHandler 线程 | 内存中 ImmediateCollectLogTaskFlag | 状态 4 |
| 操作权限 | MethodExecuteHandler 线程 | featureCode 用户授权 | 状态 3 |
| 设备在线 | MethodExecuteHandler 线程 | Redis 设备缓存 | 状态 3，CollectLogSheBeiBuZaiXian |
| 平台兼容性（周期） | MethodExecuteHandler 线程 | Intel_CR / 436Q / NEU430 等不支持 | 状态 3，CollectLogSupportedDevice |
| 安全日志平台（周期） | MethodExecuteHandler 线程 | 非 QAV3/QAV4 平台 | 状态 3，CollectLogSupportedDevice |
| Upload 响应 | MethodExecuteHandler 线程 | enbTimeout 超时 / FaultCode | 状态 3 |
| TransferComplete 响应 | MethodExecuteHandler 线程 | enbTimeout 超时 / FaultCode | 状态 3 |
| SetParameterValues 响应 | MethodExecuteHandler 线程 | enbTimeout 超时 / FaultCode | 状态 3 / 7 |

---

## 七、数据库表结构

| 表名 | 用途 |
|------|------|
| `log_device_report` | 立即收集任务记录（含周期任务，通过 log_type 区分安全日志） |
| `log_device_report_file` | 立即收集任务对应的日志文件记录 |
| `log_period_report_task` | eNB 周期上报任务记录（customizePeriodReportLogFile 专用） |
| `log_period_report_record` | 周期上报任务收到的日志文件记录 |

文件列表查询时，`log_device_report_file` 和 `log_period_report_record` 通过 `task_id` JOIN 合并，统一按 `upload_time` 排序后分页展示。

---

## 八、安全日志特殊处理汇总

| 环节 | 处理逻辑 |
|------|---------|
| 前端传参 | `reportPeriod` 单位为**小时** |
| Web 层接收 | 直接透传，不转换 |
| CELLHANDLER 任务列表展示 | 将秒 ÷ 3600 转为小时返回前端 |
| CELLHANDLER 写库 | 将小时 × 3600 转为秒存入 `report_period` |
| MethodExecuteHandler 下发 | 上报 URL 使用 `fileType=SECURITY_LOG`；周期间隔再 ÷ 3600 得小时值下发给设备（设备侧接受小时） |
| 文件存储 | 保存到 `.../security_log/` 目录 |
| 文件数量限制 | **跳过**，不受 `CollectMaxLogFileNum` 限制 |
| 平台限制 | 仅支持高通 QAV3/QAV4 平台 |
