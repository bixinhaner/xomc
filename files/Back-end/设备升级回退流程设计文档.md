# 设备升级流程设计文档（4G 与 5G）

> **梳理日期**：2026-03-16  
> **定位**：本文档面向重构参考，描述升级系统的业务逻辑与数据约定，不绑定具体实现类名。

> **⚠️ 本文档描述范围说明**
>
> 本文档**仅描述标准 4G/5G 基站（eNB/gNB）的普通 IMG 软件主镜像升级流程**，暂不涉及以下特殊场景：
> - **NXP 分布式基站**：EU（扩展单元）/ RU（射频单元）/ BU（基带单元）分体式升级逻辑
> - **AP 升级**：taskType=7，无权限校验，独立处理路径
> - **其他特殊类型**：PATCH、BIOS/UBOOT、FPGA 等类型共用相同状态机框架，差异点在权限码和 FileType 字段，本文不展开描述
>
> 如需了解上述特殊场景，请参考对应模块的专项文档。

---

## 一、升级文件类型说明

升级文件在数据库中以 `file_type` 字段区分，存储于 `t_version_info` 表：

| file_type | 含义 | 文件存储目录 |
|-----------|------|-------------|
| **0** | ⭐ **普通 IMG 软件升级文件（4G/5G 基站主镜像）— 本文档重点描述** | `software_path/upgrade/` |
| 1 | PATCH 补丁包 | `software_path/ca/` |
| 2 | BIOS/UBOOT 升级文件 | `software_path/bios/` |
| 6 | FPGA 升级文件 | `software_path/fpga/` |

> **本文档升级流程描述均以 file_type=0（IMG 软件主镜像）为核心场景。** PATCH、BIOS、FPGA 等类型共用相同的状态机框架，仅在 FileType 字段、权限码上有差异，差异点已在对应章节注明。

---

## 二、设备平台类型与 4G/5G 分类

升级逻辑以平台类型区分 4G 和 5G：

- **5G 平台**：BBU-XSS、BBU-QSS、中国电信定制 5G
- **4G 平台**：BAIBLQ、MLQ、MLN、BM、Nova430 系列、QA 系列、Intel 系列、QB、BLX、NB-IOT DXDF（永鼎）等

---

## 三、升级文件入库流程

### 3.1 上传入库

```
前端上传文件
    ├─ 1. 磁盘空间检查（达到告警阈值则拒绝）
    ├─ 2. 解析文件类型（fileType → 数据库 file_type 整型）
    ├─ 3. 版本唯一性校验（file_type + version 唯一）
    ├─ 4. 确定物理存储目录（按 fileType 分目录，见第一章）
    ├─ 5. 保存文件（本地文件系统 或 FTP，由配置决定）
    ├─ 6. 计算 MD5
    ├─ 7. 写入升级文件元数据表（file_name/file_type/version/size/md5/product 等）
    ├─ 8. 若开放给运营商（to_who != "none"）→ 写入版本通知表（登录提示用）
    └─ 9. 记录操作日志，返回 MD5 给前端
```

### 3.2 版本预校验

前端可在正式上传前调用预校验接口，校验 `file_type + version` 组合唯一性。

### 3.3 文件删除

1. 清除运营商版本通知记录
2. 删除物理文件（本地或 FTP）
3. 删除数据库记录

### 3.4 文件列表查询过滤维度

| 过滤字段 | 说明 |
|----------|------|
| `is_gnb` | 4G（0）/ 5G（1）|
| `file_type` | 升级文件类型 |
| `productValue` | 产品名称正则匹配 |
| `operatorCode` | 运营商编码（权限过滤）|
| `user_type` | admin / beta / 普通用户（可见范围不同）|
| `searchText` | 关键字模糊搜索 |

---

## 四、设备升级执行流程

### 4.1 任务调度机制

升级执行采用**状态机模式**，任务调度器循环推进步骤：

```
任务调度器轮询 → 根据当前步骤（nextStep）分发：
    checkDevicePermission  → verityEnbIsOnline
    → sendDownloadCmd      → waitDownloadCmdComplete
    → waitTransferComplete（4G 终态）
    → waitUpgradeFinish    （5G 专有）
```

**升级任务关键参数**：

| 字段 | 说明 |
|------|------|
| `taskId` | 任务唯一标识 |
| `taskType` | 任务类型（见第七章）|
| `productType` | 设备产品类型（区分 4G/5G 平台）|
| `filename` | 升级文件名 |
| `filesize` | 升级文件大小（字节）|
| `md5` | 升级文件 MD5 |
| `rawMode` | 保留配置标志（`false`=保留，`true`=不保留）|
| `commandKey` | Download 命令唯一键，同时用于 TransferComplete 匹配 |
| `isGnb` | 是否 5G 基站（`"1"`=5G，`"0"`=4G）|
| `userCode` | 操作用户 |

---

### 4.2 4G 设备升级流程

#### 4G 适用平台
BAIBLQ、MLQ、MLN、BM、Nova430 系列、QA 系列、Intel 系列、QB、BLX、NB-IOT DXDF（永鼎）等

#### 4G 升级完整流程图

```
[开始：任务创建，写入升级任务表]
         │
         ▼
 Step 0: checkDevicePermission（权限检查）
         │
         ├─ 其他类型：校验用户对设备的升级权限（featureCode 控制）
         │   - IMG 升级：CODE_ENB_UPGRADE_IMAGE
         │   - PATCH：  CODE_ENB_UPGRADE_PATCH
         │   - FPGA：   CODE_ENB_UPGRADE_FPGA
         │   - 回滚：   CODE_ENB_ROLLBACK
         │   无权限 → 子任务 FAILURE，结束
         │   有权限 → 下一步
         ▼
 Step 1: verityEnbIsOnline（检查基站在线状态）
         │
         ├─ 检查是否有其他任务正在升级此设备（getUpgradingInfo）
         │   已有任务 → 子任务失败：YiJingCunZaiZaiZhiXingDeShengJiRenWu，结束
         │
         ├─ 记录升级中任务（addUpgradingInfo）
         │
         ├─ 基站在线（cellConnected == true）
         │   → 进入 sendDownloadCmd
         │
         └─ 基站不在线
             记录等待时间 lastWaitEnbReconnectTime
             → 子任务状态：WAITTING（XiaFaXiaZaiMingLingJiZhanBuZaiXian）
             等待基站上线（轮询间隔由任务调度决定）：
             - 在线 → 进入 sendDownloadCmd
             - 超过 waitEnbReconnectTime（默认1小时） → 超时失败
                     resultStr: JiZhanChaoGuoYiXiaoShiBuZaiXian
         ▼
 Step 2: sendDownloadCmd（下发 TR069 Download 命令）
         │
         ├─ 更新主任务状态 → IN_PROGRESS
         ├─ 更新子任务状态 → IN_PROGRESS
         │
         ├─ 从 jobParam 中取出本次下发所需的核心参数：
         │   ┌─────────────────────────────────────────────────────────────────┐
         │   │ filename  ← jobParam["filename"]  升级文件名（含路径前缀）        │
         │   │ filesize  ← jobParam["filesize"]  文件字节数                     │
         │   │ md5       ← jobParam["md5"]        文件 MD5 校验值               │
         │   │ rawMode   ← jobParam["rawMode"]    保留配置标志（见下文说明）      │
         │   │ filetype  ← jobParam["filetype"]   DownloadFileType 枚举名       │
         │   └─────────────────────────────────────────────────────────────────┘
         │
         ├─ 保留配置（rawMode）说明：
         │   rawMode = "false"  → 保留当前基站配置（升级后配置不丢失）
         │   rawMode = "true"   → 不保留配置（纯镜像覆盖，恢复出厂参数）
         │   rawMode = ""       → 不携带此字段（视设备默认行为）
         │   该值在创建任务时由前端 isKeepConfig 字段决定：
         │     isKeepConfig = "1" → rawMode = "false"（保留）
         │     isKeepConfig = "0"/空 → rawMode = "true"（不保留）
         │   RawMode 字段写入 TR069 Download SOAP 报文的扩展字段 <RawMode>
         │
         ├─ 构造 commandKey：
         │   普通平台：commandKey = "Download Upgrade," + UUID（随机 UUID）
         │   永鼎（DXDF）平台：commandKey = "Download Upgrade," + UUID 前12位
         │   ★ commandKey 写入 jobParam["commandKey"]，在 waitTransferComplete
         │     阶段取 commandKey.split(",")[1]（即 UUID 部分）作为 Redis 匹配键
         │
         ├─ 确定 TR069 FileType 字段（由 DownloadFileType 枚举映射）：
         │   FIRMWARE       → "1 Firmware Upgrade Image"（4G/5G 主镜像）⭐ 核心场景
         │   BIOS_UPGRADE   → "0 Firmware Upgrade Bios"（UBOOT）
         │   PATCH          → "X {OUI} Software Upgrade Patch"（OUI 从设备取）
         │   FPGA           → "Firmware Upgrade Fpga"
         │
         ├─ 构造 TR069 Download 报文（Download.java），关键字段：
         │   ┌──────────────────┬──────────────────────────────────────────────────────┐
         │   │ 字段             │ 值                                                   │
         │   ├──────────────────┼──────────────────────────────────────────────────────┤
         │   │ CommandKey       │ commandKey（见上文）                                  │
         │   │ FileType         │ DownloadFileType 映射后的字符串（见上文）              │
         │   │ URL              │ 见下文 URL 拼接规则                                   │
         │   │ Username         │ prop.softwareCellDownloadHttpUsername（配置文件）      │
         │   │ Password         │ prop.softwareCellDownloadHttpPassword（配置文件）      │
         │   │ FileSize         │ filesize（字节数）                                    │
         │   │ TargetFileName   │ filename                                              │
         │   │ Md5              │ md5                                                   │
         │   │ RawMode          │ rawMode（"true"/"false"/""，保留配置标志）             │
         │   └──────────────────┴──────────────────────────────────────────────────────┘
         │
         ├─ URL 拼接规则（TR069 Server 中构造，HTTP 协议）：
         │   基础 URL：
         │     服务端为该设备查询下载根地址 + "/downloadFileServlet?filename=" + filename
         │   附加参数（eNB/gNB 基站）：使用转义 \& 拼接
         │     url += "\\&task_id=" + taskId
         │     url += "\\&taskType=" + taskTypeStr   （如 "UPGRADE"/"UPGRADE_BIOS"/"AP"...）
         │     url += "\\&smallCellCode=" + cellCode
         │   附加参数（NB-IOT/永鼎设备，使用普通 & 拼接）：
         │     url += "&task_id=" + taskId
         │     url += "&taskType=" + taskTypeStr
         │     url += "&smallCellCode=" + cellCode
         │   IPv6 设备：使用 IPv6 地址的 Web URL
         │   公网部署（webPublicUrl 非空）：使用公网 URL
         │   taskTypeStr 映射：
         │     TaskType.UPGRADE      → "UPGRADE"
         │     TaskType.BIOS_UPGRADE → "UPGRADE_BIOS"
         │     TaskType.CA_UPGRADE   → "UPGRADE_CA"
         │     TaskType.FPGA_UPGRADE → "UPGRADE_FPGA"
         │
         ├─ 调用下载接口封装，写入 Redis 通知 TR069 进程向设备发送 Download 命令
         │     指令队列 key：{cellCode}_DownloadReqMgr
         │   → TR069 进程读取后，构造 SOAP Download 报文发给设备
         │   → 发送 UDP ConnRequest 通知设备主动发起 Inform
         │
         ├─ 设置 TR069 等待标识 setNewRedisCompl：
         │   uuid = "TransferCompleteReq_" + commandKey.split(",")[1]
         │   → redisUtil.hmset(uuid, {"complete": "0"})
         │   → 用于在 waitDownloadCmdComplete 阶段提前捕获设备发来的 TransferComplete 错误
         │
         ├─ 记录 commandKey 到 jobParam["commandKey"]（供 waitTransferComplete 阶段使用）
         │
         ├─ 等待 TR069 进程反馈 DownloadResponse：
         │   ┌─ TIMEOUT：
         │   │   resultStr: JiZhanZaiXianXiaFaXiaZaiMeiYouHuiFu
         │   │   子任务失败，删除任务，deleteDownLoadMaxConcurrentCaches
         │   │
         │   └─ FINISHED：
         │       设备返回 faultCode → 错误信息写入 resultStr，子任务失败
         │       设备返回正常 → 清空 tr069OperParam，setNextStep → waitDownloadCmdComplete
         ▼
 Step 3: waitDownloadCmdComplete（等待设备下载升级文件完成）
         │
         ├─ Redis Key: DownloadingFlag[taskId_cellCode]
         │   ┌─ 无（null）：等待设备来 OMC 取文件
         │   │   WaitEnbRequestDownloadTime 未设置 → 记录时间
         │   │   已超过 waitDownloadFileReconnectTime（默认10分钟）
         │   │     → 超时：SheBeiMeiYouQingQiuXiaZaiShengJiWenJian
         │   │
         │   ├─ "0"：正在下载中（downloading）
         │   │   → 继续等待（更新任务记录）
         │   │
         │   ├─ "2"：下载中断（断点续传等待）
         │   │   lastWaitDownloadFileReconnectTime 未设置 → 记录时间
         │   │   → 子任务状态：WAITTING（XiaZaiCuoWuDaiXuChuan）
         │   │   已超过 waitDownloadFileReconnectTime（默认10分钟）
         │   │     → 超时：XiaZaiShiBai
         │   │
         │   ├─ "3"：升级文件不存在
         │   │   → 失败：MuBiaoBanBenWenJianBuCunZai
         │   │
         │   └─ 其他（下载完成）：
         │       deleteTransferCompleteFlag（清除等待 TC 的 Redis 标识）
         │       → 进入 waitTransferComplete
         │
         ├─ 检查是否提前收到了 TransferComplete（getTransferCompleteResult）
         │   已收到 TransferComplete 且有 faultCode → 子任务失败
         │
         └─ UpgradeTaskFlag.isTerminated → 任务被终止则立即结束
         ▼
 Step 4（4G 终态）: waitTransferComplete（等待 TransferComplete 回执）
         │
         ├─ 初始化时：从 jobParam["commandKey"] 中取出 commandKey，截取 UUID 部分：
         │   uuid = "TransferCompleteReq_" + commandKey.split(",")[1]
         │   ★ 此 uuid 与 sendDownloadCmd 阶段通过 setNewRedisCompl 写入的 Redis key 完全一致
         │   ★ TR069 进程收到设备上报的 TransferComplete 消息后，将 commandKey 与此 uuid 匹配，
         │     确认是同一次 Download 任务，再将结果写入该 Redis key（complete=1，result={...}）
         │   → 通过 tr069Oper.initTransferCompleteReq() + tr069Oper.oper() 订阅该 Redis key
         │
         ├─ WAIT：继续等待 TR069 进程写入结果
         │
         ├─ TIMEOUT：
         │   ★ 容错机制：TransferComplete 可能因网络问题丢失，但设备实际已升级成功
         │   向设备侧查询当前运行版本，与本次任务的目标版本进行比对：
         │     版本一致  → 认为升级成功（TransferComplete 丢失，忽略超时）
         │                  resultStr 不写入错误，子任务判定为 SUCCESS
         │     版本不一致 → 升级失败
         │                  resultStr: XiaZaiChengGongWeiShouDaoChuanShuWanCheng
         │   无论哪种结果：isSubTaskEnd=true，删除任务，清理 DownloadingFlag
         │
         └─ FINISHED（收到 TransferComplete）：
             设备返回 faultCode → 失败：ShouDaoChuanShuWanChengCuoWu
             正常返回（无 faultCode）：
               4G → isSubTaskEnd=true，子任务成功，删除任务，清理 DownloadingFlag
               【4G 升级至此完成】
         ▼
[子任务汇总：delSummarResult → doSummarizedTaskInfo]
         │
         ├─ 更新子任务状态：END + SUCCESS/FAILURE
         ├─ 查询未完成子任务数（getNotEndSubTaskCount）
         └─ 所有子任务完成 → 更新主任务状态：
              正常 → updateUpgradeTaskResult + TaskStatus.END
              终止 → rmvTerminateFlag + updateUpgradeTaskResult + END
              挂起 → rmvSuspendFlag + TaskStatus.SUSPEND
              等待 → rmvWaittingFlag + updateUpgradeTaskResult
```

---

### 4.3 5G 设备升级流程

#### 5G 适用平台
BBU-XSS、BBU-QSS、中国电信 5G 定制

#### 5G 与 4G 流程差异核心说明

5G 设备升级在 `waitTransferComplete` 步骤后，**不直接结束任务**，而是进入额外的 `waitUpgradeFinish` 步骤等待设备上报升级完成事件（TR069 EventCode `102 UPGRADE FINISH` 或状态变更事件），这是因为：
- 5G 设备文件下载完成（TransferComplete）后，设备还需要进行重启和版本切换
- 升级完成标识由设备主动上报 `102 UPGRADE FINISH` 或通过状态变更通知 OMC

同时，5G 升级在 `sendDownloadCmd` 结束后，会**预删除** Redis 中的 `{deviceCode}_upgrade_finish` 标识，防止旧标识残留干扰。

#### 5G 升级完整流程图

```
[开始：任务创建，写入升级任务表]
         │
         ▼
 Step 0: checkDevicePermission（权限检查）
         │  [与 4G 相同，5G 使用 CODE_GNB_ROLLBACK 权限码]
         ▼
 Step 1: verityEnbIsOnline（检查基站在线状态）
         │  [与 4G 相同]
         ▼
 Step 2: sendDownloadCmd（下发 TR069 Download 命令）
         │
         ├─ 与 4G 完全相同（URL 拼接/MD5/rawMode/commandKey 生成/DownloadFileType 映射）
         │
         └─ 关键差异：DownloadResponse 正常返回后（FINISHED 无 faultCode），额外执行：
             预删除 Redis 标识：redisUtil.del({cellCode}_upgrade_finish)
             → 防止上次升级残留的 102 事件标识干扰本次升级完成判断
         ▼
 Step 3: waitDownloadCmdComplete（等待设备下载升级文件完成）
         │
         ├─ 与 4G 基本相同
         │
         └─ 关键差异：下载完成时（DownloadingFlag 有值且非 0/2/3）
             5G：不删除 TransferComplete 标识
             （因为 5G 的 TransferComplete 是文件下载完成，还需后续步骤）
             → 进入 waitTransferComplete
         ▼
 Step 4: waitTransferComplete（等待 TransferComplete 回执）
         │
         ├─ 初始化：与 4G 完全相同
         │   uuid = "TransferCompleteReq_" + commandKey.split(",")[1]
         │   ★ TR069 进程用设备上报的 TransferComplete 中的 CommandKey 与此 uuid 匹配，
         │     确认是同一次 Download 命令的回执，再写入结果
         │
         ├─ TIMEOUT：与 4G 相同，版本比对容错
         │
         └─ FINISHED（收到 TransferComplete）：
             设备返回 faultCode → 5G 特殊处理：
               isSubTaskEnd=true，删除任务，清理 DownloadingFlag，getReq 清理消息队列
               【失败，流程终止】
             正常返回（无 faultCode）：
               ★ 5G 特殊：清空 tr069OperParam
               ★ 5G 特殊：setNextStep → waitUpgradeFinish
               【继续等待升级完成标识，不直接结束！】
         ▼
 Step 5（5G 专有）: waitUpgradeFinish（等待升级完成事件）
         │
         ├─ 初始化（tr069OperParam 为 null 时首次进入）：
         │   构造 TR069OperParam：
         │     deviceCode = cellCode
         │     operType   = UPGRADE_FINISH
         │     step       = INIT
         │   → 调用 tr069Oper.oper() 向 TR069 进程发起等待订阅
         │   → TR069 进程监听该设备的 102 UPGRADE FINISH 事件或状态变更通知
         │   → 保存 tr069OperParam，更新任务记录
         │
         ├─ WAIT：TR069 进程尚未写入结果
         │   继续轮询等待，更新任务记录
         │
         ├─ TIMEOUT：TR069 进程等待 102 事件超时
         │   ★ 此阶段超时不做版本比对，直接失败
         │   → 失败：ShengJiChaoShiWeiShouDaoShengJiWanChengBiaoShi
         │   isSubTaskEnd = true，删除任务
         │
         └─ FINISHED：TR069 进程收到 102 UPGRADE FINISH 事件或状态变更，写入结果
             结果中包含：upgradeStatus（升级状态码）、failureReason（失败原因）
             ┌──────────────────────────────────────────────────────┐
             │ upgradeStatus == "2" 或 "3" → 升级失败               │
             │   failureReason 非空 → resultStr = failureReason     │
             │   failureReason 为空 → resultStr = "Fail"            │
             │ 其他值（含 "1"）     → 升级成功，resultStr 不写错误   │
             └──────────────────────────────────────────────────────┘
             isSubTaskEnd = true，删除任务
             【5G 升级至此完成】
         ▼
[子任务汇总：delSummarResult → doSummarizedTaskInfo]
         │  [与 4G 相同]
```

---

### 4.4 4G 与 5G 升级流程核心差异对比

| 环节 | 4G | 5G |
|------|----|----|
| **权限码（回滚）** | `CODE_ENB_UPGRADE` | `CODE_GNB_UPGRADE` |
| **sendDownloadCmd 完成** | 正常进入等待下载 | 额外预删除 `{cellCode}_upgrade_finish` Redis 标识 |
| **URL 拼接** | `getEnbDownloadRootUrl(cellCode)` + `\&` 转义参数 | 相同（均使用 `\&` 转义）|
| **RawMode（保留配置）** | 写入 Download SOAP 扩展字段 | 相同 |
| **commandKey 生成** | `"Download Upgrade," + UUID` | 相同 |
| **TransferComplete 匹配** | `uuid = commandKey.split(",")[1]` | 相同 |
| **waitDownloadCmdComplete 完成** | 删除 TransferComplete Redis 标识 | **不删除** TransferComplete 标识 |
| **waitTransferComplete faultCode** | 立即失败 | 特殊处理：立即结束，清理 DownloadingFlag 和消息队列 |
| **waitTransferComplete 成功** | **直接结束（isSubTaskEnd=true）** | **进入 waitUpgradeFinish（不结束）** |
| **waitUpgradeFinish** | 无此步骤 | 专有步骤：等待设备上报 `102 UPGRADE FINISH` 事件 |
| **升级完成判定** | TransferComplete 无错误即成功 | 收到 102 事件且 upgradeStatus 为成功才算完成 |
| **超时容错** | 向设备查询当前版本，与目标版本比对：一致→成功，不一致→失败 | 相同（TC 超时时同样比对版本）；102 UPGRADE FINISH 超时则直接失败 |

---

### 4.5 软件版本回退（Rollback）流程

#### 4.5.1 任务类型

| taskType 值 | 说明 |
|-------------|------|
| `2` | 软件版本回退（Rollback）|

#### 4.5.2 权限控制

回退执行线程在每台设备处理入口处完成权限判断：

```
对队列中每台设备：
    isGnb == "1"（5G）→ 检查 CODE_GNB_ROLLBACK
    isGnb != "1"（4G）→ 检查 CODE_ENB_ROLLBACK
    无权限 → 子任务 FAILURE，跳过
    有权限 → 继续执行回退
```

#### 4.5.3 TR069 参数路径（按平台类型）

回退通过 TR069 **SetParameterValues** 向设备写入回退触发参数，不同平台路径不同：

| 平台类型 | SetValues 写入参数路径 | 是否需要先 GetValues 查询 |
|----------|-----------------------|--------------------------|
| DXDF（永鼎） | `Device.DeviceInfo.ACTIVATE_ENABLE`（平台私有）| 否 |
| 5G BBU-XSS / BBU-QSS | `Device.SoftwareCtrl.ActivateEnable`（固定）| 否 |
| NB-IOT | `Device.DeviceInfo.ROLLBACK_CONTROL`（平台私有）| 否 |
| 其余 4G（Intel/高通）| `Device.DeviceInfo.ROLLBACK_CONTROL`（平台私有）| ✅ 是（先查 `Device.DeviceInfo.ROLLBACK_ENABLE`）|

> 参数的实际私有路径通过平台参数映射表动态获取，标准路径仅作逻辑抽象用。

#### 4.5.4 SetValues 参数值差异

- **高通平台**（其 ROLLBACK_ENABLE 私有路径与高通一致）：写入 `"true"`
- **Intel 及其他平台**：写入 `"1"`  

#### 4.5.5 回退执行主流程

```
[队列中取出设备]
         │
         ├─ 权限检查（见 4.5.2）
         │
         ├─ 冲突检查：是否有其他任务正在升级此设备
         │   存在 → 子任务失败，跳过
         │   不存在 → 标记设备"正在回退"（写入 upgrading_info）
         │
         ├─ 确定平台类型，映射 rollbackControlPath / rollbackEnablePath
         │
         ├─ 子任务状态 → IN_PROGRESS
         │
         ├─ 判断平台：DXDF/NB-IOT/5G ？
         │   ├─ 是：直接组装 SetValues 参数，跳过 GetValues
         │   └─ 否（Intel/高通等 4G）：
         │       ① TR069 GetParameterValues（查询 ROLLBACK_ENABLE）
         │          - 超时 → 写入等待队列（upgradeWaitDeviceMap），置 WaittingFlag，等待重连
         │          - faultCode → 子任务失败
         │          - enable=false/0 → 子任务失败（设备不支持回退）
         │          - enable=true/1 → 允许，继续
         │       ② 组装 SetValues 参数值（高通="true"，其他="1"）
         │
         ├─ TR069 SetParameterValues（写入回退触发参数）
         │   - 超时 → 写入等待队列，等待重连后恢复
         │   - faultCode → 子任务失败
         │   - 正常返回 → 等待重启
         │
         ├─ 等待设备重启完成（RebootComplete）
         │   若已有重启标识存在 → 跳过重复等待
         │   - 重启成功 → 子任务 SUCCESS
         │   - 重启超时 → 子任务 FAILURE
         │
         └─ finally：清除 upgrading_info 标记（等待重连场景除外）
```

#### 4.5.6 断线等待恢复

当 GetValues 或 SetValues 阶段设备无响应时，流程**暂停而不失败**：

```
超时 → 写入等待重连队列（upgradeWaitDeviceMap，field=cellCode）
      JSON：{ taskId, upgradeType:"2", waitTimeBegin }
      置 WaittingFlag，保留 upgrading_info 标记

设备重新上线 → 调度器触发重连恢复逻辑：
    取出等待队列中的 taskId
    按原流程重试 GetValues / SetValues / RebootComplete
    finally：清除 upgrading_info 标记
```

#### 4.5.7 任务中止与挂起检查

每次从队列取设备前：
- 已终止（TerminateFlag）→ 停止处理线程
- 已挂起（SuspendFlag）→ 停止处理线程

写入等待队列前也会二次检查，若已中止/挂起则不入队，直接抛出异常终止。

#### 4.5.8 4G 与 5G 版本回退差异对比

| 差异点 | 4G（普通基站：Intel/高通） | 5G（BaiBNX/BBU_QSS）及 DXDF（永鼎）/ NB-IOT |
|--------|--------------------------|---------------------------------------------|
| **权限码** | `CODE_ENB_ROLLBACK` | `CODE_GNB_ROLLBACK` |
| **TR069 参数路径** | `ROLLBACK_CONTROL`（私有路径，平台映射） | `Device.SoftwareCtrl.ActivateEnable`（固定）|
| **SetValues 参数值** | 高通=`"true"`，Intel=`"1"` | `"1"` |
| **是否需要先 GetValues 查询回退可用性** | ✅ 需要（查询 `ROLLBACK_ENABLE`）| ❌ 不需要（直接下发）|
| **等待重启** | 等待 `RebootCompleteReq`（最新批量任务已去掉此步）| 等待 `RebootCompleteReq`（同）|
| **断线等待恢复** | ✅ 支持（upgradeWaitDeviceMap + rollbackForReconnect）| ✅ 支持（同）|

> **注意**：`rollback()` 注释提到"等待重启步骤在最新的批量任务中已经不存在"，说明部分版本的批量回退任务在 SetValues 成功后即认为下发完成，不再等待 RebootComplete。具体以当前部署版本为准。

#### 4.5.9 回退流程完整图（4G 普通平台为例）

```
[创建回退任务，taskType=2，写入升级任务表]
         │
         ▼
[回退执行线程循环取设备]
         │
         ├─ 检查任务是否已终止/挂起
         │
         ├─ 权限检查（CODE_ENB_ROLLBACK）
         │
         ├─ 冲突检查：有其他任务正在执行 → 失败，跳过
         │
         ├─ 标记设备正在回退（upgrading_info）
         │
         ├─ GetParameterValues（ROLLBACK_ENABLE）
         │   ├─ 超时 → 入等待队列，等待重连
         │   ├─ faultCode → 失败
         │   └─ enable=false → 失败（不支持回退）
         │
         ├─ SetParameterValues（ROLLBACK_CONTROL = "1"/"true"）
         │   ├─ 超时 → 入等待队列，等待重连
         │   └─ faultCode → 失败
         │
         ├─ 等待 RebootComplete
         │   ├─ 成功 → 子任务 SUCCESS
         │   └─ 超时 → 子任务 FAILURE
         │
         └─ finally → 清除 upgrading_info 标记
         ▼
[子任务汇总 → 更新主任务状态]
```

---

## 五、任务状态机完整说明

### 5.1 主任务状态（TaskStatus）

| 状态 | 说明 |
|------|------|
| `IN_PROGRESS` | 升级进行中 |
| `END` | 升级已结束（成功或失败） |
| `SUSPEND` | 任务被手动挂起 |

### 5.2 子任务状态（SubTaskStatus）

| 状态 | 说明 |
|------|------|
| `IN_PROGRESS` | 该设备升级进行中 |
| `WAITTING` | 等待设备上线或文件续传 |
| `END` | 升级结束 |

### 5.3 子任务结果（SubTaskResult）

| 状态 | 说明 |
|------|------|
| `SUCCESS` | 成功 |
| `FAILURE` | 失败 |
| `DEFAULT` | 默认（等待中）|

### 5.4 UpgradeTaskFlag（内存标志）

| 标志 | 说明 |
|------|------|
| `TerminateFlag` | 任务被终止 |
| `SuspendFlag` | 任务被挂起 |
| `WaittingFlag` | 任务在等待（不在线批量升级场景）|

---

## 六、Redis 关键 Key 说明

| Redis Key | 类型 | 说明 |
|-----------|------|------|
| `DownloadingFlag[taskId_cellCode]` | Hash | 文件下载状态：0=下载中，2=中断，3=文件不存在，1=完成 |
| `TransferCompleteReq_{uuid}` | Hash | TransferComplete 结果缓存：complete=0未返回/1已返回 |
| `{cellCode}_upgrade_finish` | String | 5G 升级完成标识 |
| `GlobalDownloadDevicesPool[cellCode]` | Hash | 全局下载并发池，防止超出最大并发数 |
| `Upgrade_{taskId}_*` | String | HaloD 升级信息（8.2.1新增）|
| `upgradeWaitDeviceMap`（Hash，field=cellCode） | Hash | 版本回退等待重连信息：JSON 列表，含 taskId / upgradeType / waitTimeBegin |
| `{cellCode}_RebootCompleteReqMgr` | String | 回退后等待重启完成标识（已存在则跳过重复等待）|

---

## 七、任务类型（taskType）与对应升级逻辑

| taskType 值 | 对应升级类型 | 权限码 |
|-------------|-------------|--------|
| `1` / SOFTWARE_UPGRADE | IMG 软件升级 | `CODE_ENB_UPGRADE_IMAGE` |
| `2` / UPGRADE_ROLLBACK | 版本回滚 | `CODE_ENB_ROLLBACK`（4G）/ `CODE_GNB_ROLLBACK`（5G）|
| `4` / UPGRADE_PATCH | PATCH 补丁升级 | `CODE_ENB_UPGRADE_PATCH` |
| `6` / UPGRADE_FPGA | FPGA 升级 | `CODE_ENB_UPGRADE_FPGA` |
| `8` / UPGRADE_UBOOT | BIOS/UBOOT 升级 | `CODE_ENB_UPGRADE_IMAGE`（默认）|

---

## 八、整体架构层次与模块关系

```
┌────────────────────────────────────────────┐
│              Web 服务层                      │
│  升级文件管理（上传/查询/删除/预校验）         │
│  升级任务管理（创建/查询/终止/恢复）           │
│  → RPC → 设备管理微服务                      │
└───────────────────┬────────────────────────┘
                    │ 任务调度
┌───────────────────▼────────────────────────┐
│          任务执行引擎（状态机）               │
│  步骤：权限检查 → 在线检查 → 下发Download    │
│       → 等待文件下载 → 等待TransferComplete  │
│       → 等待升级完成（5G）                   │
│  → Redis 通道 ← TR069 处理进程               │
│  → RPC → 设备信息微服务（DB 操作）           │
└───────────────────┬────────────────────────┘
                    │ TR069 协议
┌───────────────────▼────────────────────────┐
│            TR069 处理进程                    │
│  处理 Download / TransferComplete 消息       │
│  5G：处理 102 UPGRADE FINISH 事件            │
│  → 结果写入 Redis，任务引擎轮询消费           │
└───────────────────┬────────────────────────┘
                    │ TR069 over HTTP/HTTPS
               设备（4G eNB / 5G gNB）
```

---

---

## 九、关键超时参数配置说明

| 系统参数 Key | 默认值 | 说明 |
|-------------|--------|------|
| `waitEnbReconnectTime` | 1 小时 | Step1 等待基站上线超时（单位：小时）|
| `WaitDownloadFileReconnectTime` | 10 分钟 | Step3 等待文件下载/断点续传超时（单位：秒，默认600）|
| `WaitEnbRequestDownloadTime` | 10 分钟 | Step3 等待设备来 OMC 取文件的超时（同上）|
| `RebootWaitTime` | 5 分钟 | 等待设备重启重连的超时时间（单位：分钟）|

---

---

## 十、数据库表设计

> 所有表均位于 `small_cell` schema 下。

### 10.1 升级文件表：`update_file_info`

存储所有上传的升级文件元数据（对应 Java 实体类 `VersionInfo`）。

| 字段名 | 数据类型 | 说明 |
|--------|----------|------|
| `id` | int, PK, AUTO_INCREMENT | 文件唯一标识（version_id）|
| `file_name` | varchar | 文件名（物理文件名）|
| `size` | int | 文件大小（字节）|
| `upload_time` | datetime | 上传时间（UTC）|
| `version` | varchar | 版本号（如 `BaiBS37B-A/QRTB-2.7.4.5`）|
| `uploader` | varchar | 上传用户 usercode |
| `is_latest` | varchar | 是否最新：`Yes` / `No`（默认 No）|
| `file_type` | varchar | 文件类型（0~11，见文件类型说明）|
| `product` | varchar | 产品标识（对应 product_type 表 value 字段）|
| `manufacturer` | varchar | 生产厂家 |
| `to_who` | varchar | 开放范围：`all`=全部运营商 / `beta`=Beta运营商 / `none`=不开放 |
| `desc` | varchar | 版本描述 |
| `md5_error` | varchar | MD5 校验错误信息（正常为空）|
| `md5_val` | varchar | 文件 MD5 值 |
| `operator_code` | varchar | 文件所属运营商编码（上传者所属）|
| `recommend` | varchar | 推荐标识：`1`=推荐 / `0`=取消推荐 |
| `model_name` | varchar | 设备型号（多值逗号分隔，特定场景使用）|

**索引**：`file_name + file_type`（唯一性约束）

---

### 10.2 文件与产品类型关联表：`rela_upgrade_file_product`

存储升级文件与适用产品类型的多对多关系（一个文件可适用多种产品类型）。

| 字段名 | 数据类型 | 说明 |
|--------|----------|------|
| `file_id` | int | 关联 `update_file_info.id` |
| `product` | varchar(100) | 产品类型正则值（如 `FAP/\\w+/SC`）|

**索引**：`file_id_index(file_id)`

---

### 10.3 升级主任务表：`upgrade_main_task`

存储基站（eNB/gNB）软件升级的主任务记录。

| 字段名 | 数据类型 | 说明 |
|--------|----------|------|
| `task_id` | int, PK, AUTO_INCREMENT | 任务唯一 ID |
| `task_name` | varchar | 任务名称（用户自定义）|
| `file_id` | int | 关联 `update_file_info.id` |
| `file_name` | varchar | 升级文件名（冗余）|
| `create_user` | varchar | 创建人 usercode |
| `create_time` | datetime | 任务创建时间（UTC）|
| `task_status` | int | 任务状态：0=进行中(IN_PROGRESS) / 1=等待(WAITTING) / 2=暂停(SUSPEND) / 3=结束(END) |
| `task_result` | int | 任务结果：0=全部成功 / 1=部分成功 / 2=全部失败 / 3=已终止 |
| `start_time` | datetime | 任务开始执行时间 |
| `end_time` | datetime | 任务结束时间 |
| `operator_code` | varchar | 归属运营商编码 |
| `product_type` | varchar | 产品类型正则（如 `FAP/\\w+/SC`）|
| `is_keep_config` | varchar | 是否保留配置（rawMode）：`1`=保留 |
| `task_type` | int | 任务类型：1=软件升级 / 2=版本回滚 / 4=PATCH升级 / 6=FPGA升级 |
| `total_num` | int | 总设备数量 |
| `create_status` | varchar | 创建状态（active=立即执行 / suspend=挂起 / timing=定时）|
| `create_status_time` | datetime | 创建状态时间（定时任务的执行时间）|
| `select_all` | varchar | 是否全选设备 |
| `nxp_upgrade_flag` | varchar | 分布式基站升级标志（保留字段）|
| `is_gnb` | varchar | 5G 标识：`1`=5G / `0`=4G |
| `maxConcurrentNumber` | int | 最大并发升级数量 |
| `is_online_execute` | varchar | 是否仅对在线设备执行 |

---

### 10.4 升级子任务表：`upgrade_sub_task`

存储每台设备（基站）的升级执行记录。

| 字段名 | 数据类型 | 说明 |
|--------|----------|------|
| `task_id` | int | 关联 `upgrade_main_task.task_id` |
| `small_cell_code` | varchar | 基站编码 |
| `progress_status` | varchar | 子任务执行状态：WAITTING=等待 / IN_PROGRESS=进行中 / END=已结束 |
| `progress_result` | varchar | 子任务执行结果：SUCCESS=成功 / FAILURE=失败 / DEFAULT=默认 |
| `failure_reason` | varchar | 失败原因（国际化 key 或具体错误信息）|
| `execute_time` | datetime | 最后执行/更新时间 |
| `ori_version` | varchar | 升级前版本号 |
| `dest_version` | varchar | 目标版本号（升级完成后更新）|
| `update_time` | datetime | 版本更新时间 |
| `result` | varchar | 版本升级结果（明细）|
| `is_online_execute` | varchar | 是否等待上线执行标志 |

---

### 10.5 BIOS/UBOOT 升级任务表：`bios_upgrade_task`

| 字段名 | 数据类型 | 说明 |
|--------|----------|------|
| `TASK_ID` | int, PK, AUTO_INCREMENT | 任务ID |
| `TASK_NAME` | varchar | 任务名称 |
| `FILE_ID` | int | 关联 `update_file_info.id`（BIOS文件）|
| `FILE_NAME` | varchar | BIOS 文件名（冗余）|
| `TASK_STATUS` | varchar | 状态：0=激活 / 1=挂起 |
| `TASK_PROGRESS` | varchar | 进度：0=未开始 / 1=进行中 / 2=已结束 |
| `TASK_RESULT` | varchar | 结果：0=成功 / 1=部分成功 / 2=失败 / 3=已终止 |
| `START_TIME` | datetime | 开始时间 |
| `STOP_TIME` | datetime | 结束时间 |
| `OPERATOR_CODE` | varchar | 归属运营商 |

**BIOS 升级进度明细表：`bios_upgrad_task_progress`**

| 字段名 | 说明 |
|--------|------|
| `TASK_ID` | 关联主任务ID |
| `SMALL_CELL_CODE` | 基站编码 |
| `PROGRESS_DETAIL` | 进度详情（国际化key，如 `WeiKaiShi` / `ShengJiWanCheng`）|
| `RUN_TIME` | 最后更新时间 |

---

### 10.6 PATCH 升级任务表：`ca_upgrade_task`

字段与 `bios_upgrade_task` 结构完全相同，对应 PATCH/CA 证书文件升级任务。

---

### 10.7 正在升级标识表：`upgrading_info`

记录当前正在进行升级的基站，防止重复发起升级任务。

| 字段名 | 说明 |
|--------|------|
| `small_cell_code` | 基站编码（索引）|
| `task_id` | 对应升级任务ID |
| `task_type` | 任务类型（软件升级 / 回滚 / PATCH 等）|
| `task_name` | 任务名称（用于提示"已存在正在执行的任务"）|
| `start_time` | 升级标识记录时间（超过1小时自动清理）|

---

### 10.8 用户版本通知表：`sys_user_version_notice`

记录哪些用户需要被通知新上传的升级文件（登录时弹窗提示）。

| 字段名 | 说明 |
|--------|------|
| `user_code` / `user_id` | 用户标识 |
| `version_no` | 对应 `update_file_info.id`（版本文件ID）|

---

## 十一、总结

### 4G 升级流程
```
权限检查 → 在线等待 → 下发 Download → 等待文件下载 → 等待 TransferComplete → 结束
```
- TransferComplete 是 4G 升级完成的**最终判定依据**
- 超时容错：版本比对一致时视为升级成功

### 5G 升级流程
```
权限检查 → 在线等待 → 下发 Download → 等待文件下载 → 等待 TransferComplete → 等待 102 升级完成事件 → 结束
```
- TransferComplete 仅表示**文件下载完成**，不代表升级成功
- 最终成功判定依赖设备上报 **`102 UPGRADE FINISH`** 事件

### NXP 分布式基站（B4860）
> 暂不在本文档描述范围内，EU/RU/BU 分体升级属特殊场景，另行专项说明。

### 软件版本回退（Rollback）
```
权限检查 → 冲突检查 → [GetValues 查询可回退性（4G 普通）] → SetValues 触发回退 → 等待重启 → 结束
```
- 回退通过 TR069 **SetParameterValues** 触发，**不走 Download 流程**
- 5G/永鼎/NB-IOT：直接下发，无需先查询
- 4G 普通（Intel/高通）：先查询 `ROLLBACK_ENABLE`，`true`/`1` 才允许回退
- 高通写入 `"true"`，Intel 等写入 `"1"`
- 设备无响应时不立即失败，写入等待队列，重连后恢复执行
