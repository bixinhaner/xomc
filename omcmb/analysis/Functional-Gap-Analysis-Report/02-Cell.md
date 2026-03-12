# 02 — Cell (小区管理)

> **覆盖状态：⚠️ 部分覆盖（差距最大，原系统 277 个文件）**

---

## 原系统二级菜单

| # | 二级菜单 | 原系统文件数 | 新系统对应 | 状态 |
|---|---------|------------|----------|------|
| 1 | Cell Param (小区参数配置) | 118 | Config → ParamSync/LiveParamConfig/ParamList | ⚠️ 大量缺失 |
| 2 | Task → Upgrade (升级任务) | 15 | Software → UpgradePlan | ⚠️ 简化 |
| 3 | Task → Reboot (重启任务) | 7 | — | ❌ 缺失 |
| 4 | Task → ConfigBackup (配置备份) | 3 | Backup → BackupTasks | ⚠️ 简化 |
| 5 | Task → ConfigRecover (配置恢复) | 3 | Backup → RestoreData | ⚠️ 简化 |
| 6 | Task → MMLScript (MML脚本) | 5 | MML → ScriptTask | ✅ |
| 7 | Task → Trace (信令追踪) | 3 | — | ❌ 缺失 |
| 8 | Task → TRXTrace (TRX追踪) | 4 | — | ❌ 缺失 |
| 9 | Task → FactoryReset (工厂复位) | 3 | — | ❌ 缺失 |
| 10 | Task → PciAndEarfcn (PCI/EARFCN) | 5 | — | ❌ 缺失 |
| 11 | Task → ChangePassword (密码修改) | 3 | — | ❌ 缺失 |
| 12 | Task → Profile (配置文件) | 3 | — | ❌ 缺失 |
| 13 | Task → DataModel (数据模型) | 2 | — | ❌ 缺失 |
| 14 | Task → UpgradeCa (CA证书升级) | 4 | — | ❌ 缺失 |
| 15 | Task → UpgradeFpga (FPGA升级) | 1 | — | ❌ 缺失 |
| 16 | Task → UpgradeRollback (升级回滚) | 1 | — | ❌ 缺失 |
| 17 | Task → NosenceReboot (强制重启) | 3 | — | ❌ 缺失 |
| 18 | Upload (文件上传/版本管理) | 13 | File + Software → FirmwareUpload | ⚠️ 部分 |
| 19 | Fault (故障/告警管理) | 18 | Alarm → CurrentAlarms/History/Statistics | ⚠️ 部分 |
| 20 | SON (自组织网络) | 14 | — | ❌ 缺失 |
| 21 | SelfStart (设备自启动) | 17 | — | ❌ 缺失 |
| 22 | CPE (CPE频率锁定) | 3 | — | ❌ 缺失 |
| 23 | EPC (核心网管理) | 24 | — | ❌ 缺失 |
| 24 | eGW (Cell内网关) | 3 | — | ❌ 缺失 |
| 25 | NRM (网络资源模型) | 2 | — | ❌ 缺失 |

---

## 2.1 Cell Param — 小区参数配置

### 原系统 `cell/cellParam/` (118 文件)

**核心页面：**
- `cellParam.jsp` — 参数编辑器主页（树形导航 + 参数列表）
- `base.jsp` — 基本小区参数
- `cellParamFileConfig.jsp` — 配置文件导入导出

**参数配置页面 `config/` (115 文件) — 每个参数独立修改页面：**

| 分类 | 代表页面 | 功能 |
|------|---------|------|
| 小区增删 | `addEUTRANNCELL`, `addGSMNCELL`, `addTDSNCELL`, `rmvXxx` | LTE/GSM/TD-LTE 小区增删 |
| 频点管理 | `addEUTRANNFREQ`, `addGSMNFREQ`, `rmvXxxNFREQ` | 频点增删 |
| PLMN | `addPLMN`, `rmvPLMN` | PLMN 管理 |
| 核心参数 | `modCell`, `modEUTRANNCELL`, `modGSMNCELL` | 修改小区核心参数 |
| 接入控制 | `modCellAccessControlSetting`, `modCellBarredInfo` | 小区禁止/接入控制 |
| 随机接入 | `modRandomAccess` | RACH 参数 |
| 功率控制 | `modPowerControl`, `modULPOWERCONTRL` | 上下行功率控制 |
| RRC配置 | `modRRCConfig`, `modRRCStatus` | RRC 参数/状态 |
| 测量事件 | `modA1`~`modA5`, `modB2` | A1-A5/B2 事件阈值 |
| 切换配置 | `modX2`, `modMME`, `modMMEPool` (多变体) | X2/MME/MME池 |
| DRX/Gap | `modDRX`, `modMeasGAP` | DRX/测量间隔 |
| 定时器 | `modUETimer`, `modTATimer` | UE/TA 定时器 |
| 同步 | `modNTP`, `modGPSSync`, `mod1588`, `modSyncAdjust` | NTP/GPS/1588同步 |
| 端口/网络 | `modPortSetting`, `modWanPortSetting`, `modSCTPSetting` | 端口/WAN/SCTP |
| 安全 | `modSecurity`, `modSSHSetting`, `modIpsecEnable/Config` | SSH/IPSec |
| IPSec隧道 | `addIpsecSetting`, `modIpsecSetting`, `rmvIpsecSetting` | IPSec CRUD |
| 天线 | `modAldAction/Scan/Tilt`, `modRETTilt`, `modRETAld` | ALD/RET 天线调整 |
| ANR | `modANR`, `modANR4860`, `modANRMLN` | ANR 配置 |
| 负载均衡 | `modLBSetting` | 负载均衡 |
| LGW | `modLGW`, `modLGWConfig` (多变体) | 本地网关 |
| 其他 | `modCapacity`, `modCarrierMode`, `modPCIRange` 等 | 容量/载波/PCI/KPI |

### 新系统对应

| 新系统页面 | 功能 | 覆盖 |
|-----------|------|------|
| Config → ParamSync | NE 树 + 参数���表 + 同步状态 | ⚠️ 有列表无编辑表单 |
| Config → LiveParamConfig | 实时参数内联编辑 | ⚠️ 通用编辑，无专用表单 |
| Config → ParamList | 参数只读查看 | ⚠️ 只读 |
| Config → BatchParamClass | 参数分类查看 | ⚠️ 分类树 |
| Config → BatchParamTemplate | 批量参数模板 | ⚠️ 模板 CRUD |
| Config → CellManagement | 小区 CRUD | ⚠️ 字段远少于原系统 |

### CellManagement 字段对比

| 原系统字段 | 新系统 | 状态 |
|-----------|--------|------|
| cellId, cellName | cellId, cellName | ✅ |
| enbId, hostName | stationName | ⚠️ 合并 |
| PHYCELLID (PCI) | pci (0-503) | ✅ |
| TAC | tac (0-65535) | ✅ |
| EARFCNDLINUSE | earfcn | ✅ |
| bandwidth | bandwidth (1.4~100 MHz) | ✅ |
| plmnid | — | ❌ |
| signment, specialSubframe | — | ❌ |
| rootIndex | — | ❌ |
| tx_power | — | ❌ |
| network_model | — | ❌ |
| 115 个独立参数修改页面 | LiveParamConfig 通用编辑 | ⚠️ 无专用验证 |

---

## 2.2 Task → Upgrade — 升级任务

### 原系统 `cell/task/upgrade/` (15 文件)

**addTask.jsp 表单字段：**
- `taskname` — 任务名称（最长 100 字符）
- `product` — 产品类型选择
- `nxpUpgradeFlag` — 强制升级勾选
- `taskType` — 升级类型 (1=软件, 4=补丁, 6=FPGA, 7=AP)
- 设备 PairGrid 左列：`serial_number`, `cell_name`, `cell_ip`, `rollback_version`, `software_version`, `module_type`, `group_name`
- 设备 PairGrid 右列：`serial_number`, `host_name`
- `status` — 执行模式（立即/计划/挂起）

**task_list.jsp 表格列：**
`TASK_NAME`, `FILE_NAME`, `VERSION`, `TASK_STATUS`(0=挂起/1=激活), `TASK_PROGRESS`(1=进行/2=完成), `TASK_RESULT`, `START_TIME`, `STOP_TIME`

**task_progress.jsp：** 逐设备进度列表

**多设备类型支持：** addTaskForCpe, task_progress_cpe, addTaskForUPS, task_progress_UPS, addTask_super
**多升级类型：** newUpgradeTaskEnbSoftware/Patch/Fpga/Uboot, newUpgradeTaskCpeOdu/Idu

### 新系统 Software → UpgradePlan

**表格列：** planName, targetVersion, deviceCount, status, progress, createTime, startTime, endTime

| 原系统功能 | 新系统 | 状态 |
|-----------|--------|------|
| 多设备类型升级(eNB/CPE/UPS) | 通用升级 | ❌ 无类型区分 |
| 升级类型(软件/补丁/FPGA/Boot) | — | ❌ |
| 设备 PairGrid 选择器 | — | ❌ |
| 逐设备进度监控 | 整体进度条 | ⚠️ 简化 |
| 升级回滚 | — | ❌ |

---

## 2.3 Task → Reboot — 重启任务（❌ 完全缺失）

**原系统 7 文件：** addTask, task_list, task_progress, addTaskForCpe, task_list_cpe, task_progress_cpe, recurringRebootTask

**表单字段：** taskname, product, 设备PairGrid(serial_number, host_name), status(立即/计划/周期/循环)

---

## 2.4 Task → ConfigBackup/Recover — 配置备份恢复

**原系统 6 文件**

**新系统 Backup 模块已有：** 任务列表、备份策略、FTP配置、数据恢复
**缺失：** 设备级 PairGrid、逐设备进度、完整创建表单

---

## 2.5 Task → MMLScript — MML脚本 ✅ 基本覆盖

**原系统 5 文件 → 新系统 MML → Console + ScriptTask + CommandTree**

**task_list.jsp 表格列：** TASK_NAME, CREATE_USER, CREATE_TIME, CREATE_STATUS, TASK_STATUS, TASK_PROGRESS, TASK_RESULT, START_TIME, END_TIME
**结果表格列：** SERIAL_NUMBER, HOST_NAME, MML, PROGRESS_STATUS, PROGRESS_RESULT, FAILURE_REASON, DETAIL, RUN_TIME, END_TIME

---

## 2.6~2.17 其他 Task 子模块（全部 ❌ 缺失）

| 二级菜单 | 文件数 | 关键表单字段 |
|---------|--------|-------------|
| Trace (信令追踪) | 3 | 设备选择, 追踪类型, 时间范围 |
| TRXTrace (TRX追踪) | 4 | 设备选择, 追踪参数, ECharts 图表 |
| FactoryReset (工厂复位) | 3 | 设备 PairGrid, 执行模式 |
| PciAndEarfcn | 5 | PCI值, EARFCN, CPE PCI锁定 |
| ChangePassword | 3 | 设备选择, 新密码, 确认密码, 修改记录 |
| Profile (配置文件) | 3 | 配置文件选择, 文件列表 |
| DataModel | 2 | 参数名/类型/值 |
| UpgradeCa | 4 | CA文件选择, 设备选择 |
| UpgradeFpga | 1 | FPGA文件, 设备选择 |
| UpgradeRollback | 1 | 回滚版本, 设备选择 |
| NosenceReboot | 3 | 设备选择, 强制重启 |

---

## 2.18 Upload — 文件上传/版本管理

| 原系统页面 | 功能 | 新系统 | 状态 |
|-----------|------|--------|------|
| `fileMgr.jsp` | 文件管理器 | File → UserFiles/DeviceFiles | ⚠️ |
| `fileImport.jsp` | 文件导入(version, product, modelName) | Software → FirmwareUpload | ⚠️ |
| `fileEdit.jsp` | 文件编辑 | — | ❌ |
| `fileView.jsp` | 文件查看 | — | ❌ |
| `fileMd5.jsp` | MD5 校验 | — | ❌ |
| `versionMgr.jsp` | 版本管理 | Software → VersionQuery | ⚠️ |
| `versionMgrForCpe.jsp` | CPE版本管理 | — | ❌ |

---

## 2.19 Fault — 故障/告警管理

| 原系统页面 | 功能 | 新系统 | 状态 |
|-----------|------|--------|------|
| `fault_list.jsp` | 活跃告警 | Alarm → CurrentAlarms | ✅ |
| `alarm_detail.jsp` (19字段) | 告警详情 | CurrentAlarms 详情列(10字段) | ⚠️ 缺9字段 |
| `alarm_confirm.jsp` | 确认告警 | CurrentAlarms 确认按钮 | ✅ |
| `alarm_level_conf.jsp` | 等级配置 | Alarm → AlarmRules | ⚠️ |
| `alarm_statistic.jsp` | 统计 | Alarm → AlarmStatistics | ✅ |
| `notification.jsp` | 邮件通知 | — | ❌ |
| `alarmNoticeSetting.jsp` | 通知设置(声音+邮件+模板) | — | ❌ |
| `view.jsp` | 可配置仪表板 | — | ❌ |
| `itfn_fault_list.jsp` | 接口故障 | — | ❌ |
| `statistic_add/view.jsp` | 自定义统计 | — | ❌ |

**告警详情缺失字段：** SPECIFIC_PROBLEM, ADDITIONAL_INFORMATION, ADDITIONAL_TEXT, NE_TYPE, EQUIP_INFO, DEAL_TIME, CLEAR_USER, CLEAR_TIME, SUGGESTION, DEAL_MEMO

---

## 2.20 SON — 自组织���络 (14 文件，全部 ❌ 缺失)

| 页面 | 功能 |
|------|------|
| `PCIOptimization_v2.jsp` | PCI 自动优化 |
| `PCIDetection.jsp` | PCI 冲突检测 |
| `PCIValue_v2.jsp` | PCI 分配查看 |
| `PCI_Range.jsp` | PCI 范围配置 |
| `anrLog.jsp` | ANR 日志 |
| `selfConfiguration/plug_and_play.jsp` | 即插即用 |
| `selfConfiguration/gnb_plug_and_play.jsp` | gNB 即插即用 |
| `selfConfiguration/selfConfig_control.jsp` | 自配置控制 |
| `accessControl/accessControl.jsp` | SON 接入控制 |

---

## 2.21 SelfStart — 自启动 (17 文件，全部 ❌ 缺失)

| 页面 | 功能 |
|------|------|
| `selfstart.jsp` | 管理主页 |
| `selfstart_info/progress.jsp` | 信息/进度 |
| `selfconfig_importFile/setting/record.jsp` | 导入/设置/记录 |
| `editParamTask/ConfigFileTask/SoftwareTask.jsp` | 编辑参数/配置/软件 |

---

## 2.22 EPC — 核心网管理 (24 文件，全部 ❌ 缺失)

| 子模块 | 页面 | 功能 |
|--------|------|------|
| config | epcManageConfig, epcServerManage | EPC 配置/服务器 |
| config | apnConfig, apnSetting | APN 接入点 |
| config | deviceList, devicePolicy, addDevicePolicy | 设备策略 |
| config | imsiAllocated_new, allImsiList | IMSI 分配 |
| trace | addENB/EPCSignalingTrace | 信令追踪 |
| trace | viewENB/EPCSignalingTrace | 追踪查看 |
| trace | trace_message, trace_detail | 追踪详情 |
| counter | authenticationCounter | 认证计数器 |

---

## 缺失操作流程汇总

| 操作流程 | 状态 | 影响 |
|---------|------|------|
| 小区参数专用编辑（115 类型） | ❌ | 无法精确配置参数 |
| 设备重启全流程 | ❌ | 无法远程重启 |
| 工厂复位 | ❌ | 无法复位设备 |
| 信令追踪 | ❌ | 无法排障 |
| PCI 自动优化 | ❌ | 无法解决 PCI 冲突 |
| 即插即用自配置 | ❌ | 无法零接触部署 |
| EPC APN/IMSI 管理 | ❌ | 无法管理核心网 |
| CA 证书/FPGA/Boot 升级 | ❌ | 无法更新硬件固件 |
| 升级回滚 | ❌ | 无法回滚失败升级 |
| 告警通知(邮件+声音) | ❌ | 无法自动通知 |
