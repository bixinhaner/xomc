# F08 北向页面配置化设计（最新基线）

> 最新更新：2026-08-03
> 状态：最新设计文件，后续北向页面、后端模型和验收以本文为准。
> 页面原型：`omcmb/webcode/src/pages/config/NorthboundPageConfig/index.tsx`。
> 历史参考：`northbound-file-socket-snmp-redesign-20260730.md`、`overall-redesign-20260730.md`、`data-support-matrix-20260731.md`、桌面旧资料 `othernorth/*` 和 17 个旧北向场景文件。

## 1. 设计结论

北向不再按 XML 文件作为运行时配置入口。XML、旧场景文件和旧系统文档只作为内置模板、字段来源、命名规则和兼容分析依据；正式产品形态是页面可配置、数据库保存、运行时热加载。

当前北向页面固定为 5 个页签：

| 页签 | 作用 | 当前页面状态 |
|---|---|---|
| 北向文件配置 | 管理 CM、PM、MR、LOG 文件任务 | 已按 17 个内置场景 + 自定义配置入口实现原型 |
| Inventory 文件 | 管理 eNB、gNB、GSM、OMC Inventory 文件 | 已按 4 类 Inventory 列表实现原型 |
| Socket 告警 | 管理电信/联通 Socket 告警北向 | 已按内置配置行实现原型 |
| SNMP 告警 | 管理 SNMP V2/V3 Trap/Inform 告警北向 | 已按内置目标行实现原型 |
| 北向 API | 管理旧 northbound API 与当前 xomc 能力交集 | 已按接口清单实现原型 |

明确不做：

| 项 | 处理 |
|---|---|
| XML 运行时管理 | 不提供上传 XML 直接生效；只允许作为离线模板导入/兼容分析依据 |
| Inventory 放进北向文件业务域 | Inventory 独立页签管理，不在北向文件业务域下拉中出现 |
| PM 指标单独页签 | 不需要；PM 指标在北向文件配置的字段/指标配置里处理 |
| 传输/接口单独页签 | 不需要；FTP/SFTP、Socket 账号、SNMP 目标分别放到对应功能编辑抽屉 |
| 旧系统比对类列 | 不进入业务页面，只可作为内部分析文档或开发注释 |

## 2. 全局交互规则

所有北向能力遵循统一页面语言：

1. 第一列统一为 `操作`，包含启停开关和更多操作下拉。
2. 所有开关默认关闭，用户完成目标、周期、字段确认后再开启。
3. 列表统一有 `状态` 列，只显示 `正常` 或 `终止`。
4. `上报结果` 放在操作下拉中，不直接堆在列表列里。
5. 列表只展示稳定摘要，不展示内部来源、配置路径、旧系统说明、任务数、对象数等解释性文字。
6. 复杂内容进入查看/编辑抽屉；长内容用 tooltip 展开完整信息。
7. 分页和刷新组件保持各页签一致；即使当前行数少，也保留统一分页样式。
8. 路径和文件名只叫 `上传目录模板`、`文件名模板`；用户编辑时必须提供预览。
9. 页面显示用户友好的周期，例如 `15 分钟`、`60 分钟`、`每日`；cron 只作为后端保存格式，不直接暴露给用户。

`状态` 的定义：

| 显示 | 含义 |
|---|---|
| 正常 | 配置已启用，最近一次生成、上传、推送、Trap 或 Inform 没有失败；运行中也视为正常 |
| 终止 | 配置关闭，或最近一次执行失败、超时、中断 |

`上报结果` 抽屉统一展示：

| 字段 | 说明 |
|---|---|
| 状态 | 最近一次上报的详细状态 |
| 最近时间 | 最近一次生成、上传、推送或发送时间 |
| 最新文件/最新报文 | 文件名或报文标识 |
| 大小 | 文件或报文大小 |
| 上报目录/上报目标 | 文件远端目录，或 Socket/SNMP 目标 |
| 目标结果 | 多目标投递、客户端 ACK、Trap/Inform 结果 |
| 说明 | 失败原因或运行说明 |
| 内容预览 | 文件内容预览或 Socket/SNMP 报文 |

文件类上报结果提供 `下载最新文件`。

## 3. 北向文件配置

### 3.1 列表

北向文件列表以场景为入口。内置场景只显示 `S0001` 这种场景号，不显示内部文件名或来源说明。

| 列 | 内容 |
|---|---|
| 操作 | 开关 + 下拉：查看、编辑、上报结果、复制模板、手动执行 |
| 场景号 | `S0001` 到 `S0017`，自定义配置也使用配置编号 |
| 场景名称 | 中文场景名称，英文名称进入详情 |
| 状态 | `正常` 或 `终止` |
| 输出内容 | CM/PM/MR/LOG 与对象摘要，完整内容进 tooltip |
| 调度 | 每个业务域的周期摘要，完整生成计划进 tooltip |
| 文件规则 | 当前支持格式摘要 |
| 压缩 | 不压缩、zip 或 gz |

内置场景名称：

| 场景号 | 场景名称 | 英文名 | 摘要 |
|---|---|---|---|
| S0001 | 标准场景 | Standard | CM 日文件、PM PC 15 分钟、MR 15 分钟 |
| S0002 | 电信场景 | Telecom | PE/PC 性能对象 |
| S0003 | 上海电信 | SH-Tele | PE/PC 性能并行输出 |
| S0004 | 陕西电信 | SN-Tele | PE/PC，告警中文策略 |
| S0005 | 江苏电信 | JS-Tele | PE/PC，custom 日志 |
| S0006 | 泰国 True | Thai-True | CM/PM 精简，无 MR |
| S0007 | 泰国 AIS | Thai-AIS | CM CSV + COMS，PM 60 分钟 |
| S0008 | V1 场景 | V1 | CM CSV + COMS，fix 日志 |
| S0009 | 老挝电信 | Laos-Tele | 标准 CM/MR，PM 60 分钟 |
| S0010 | 上海联通 | SHUcom | 对象分目录，联通 Socket 相关策略 |
| S0011 | ISAT MTN | ISAT-NBI-MTN | 标准周期 |
| S0012 | 陕西移动 | Shaanxi Mobile | LTE + GNB 双制式 |
| S0013 | ZED 场景 | ZED | LTE/GSM/GNB，PM 60 分钟，pmresult 命名 |
| S0014 | 印尼 Telkomsel | YinNi_Telkomsel_MNO | 标准周期 |
| S0015 | 菲律宾 DITO | FeiLvBin-DITO | PE/PC 性能并行输出 |
| S0016 | MTN 场景 | MTN | LTE + GSM 性能混合输出 |
| S0017 | 黑龙江场景 | HLongjianng | 标准周期 |

### 3.2 支持业务域、对象和格式

北向文件业务域只包含 CM、PM、MR、LOG。

| 业务域 | 对象 | 当前格式 | 说明 |
|---|---|---|---|
| CM | CP、EP、CC、CE、COMS | CSV | 当前不开放 XML |
| PM | PC、PE | CSV | counter/KPI 从当前 PM 指标库选择 |
| MR | MRO、MRE、MRS | XML | MR 保留 XML，因为 MR 原始文件本身是 XML 体系 |
| LOG | 登录日志、操作日志、固定格式日志 | TXT、CSV | custom 用 TXT，fix 用 CSV |

格式边界：

| 结论 | 说明 |
|---|---|
| CM 不选择 XML | 当前系统没有稳定 CM XML renderer，页面不提供 XML |
| PM 不选择 XML | 当前 PM 文件输出以 CSV 为准，指标来自 PM 聚合结果 |
| Inventory 不在此处选择 | Inventory 独立页签，只支持 CSV |
| MR 可用 XML | MR 文件类数据保留 XML |

### 3.3 查看抽屉

查看抽屉展示：

1. 基础信息：场景号、中文名、英文名、配置名称、说明、业务域、启用状态。
2. 对象管理：业务域、对象、制式/模式、格式、统计周期、生成计划、压缩、上传目录模板、文件名模板、预览。
3. 传输目标：FTP/SFTP 目标列表。
4. 字段/指标配置：按业务域、制式、对象查看字段或指标。

### 3.4 编辑抽屉

编辑抽屉支持：

| 区域 | 配置项 |
|---|---|
| 基础信息 | 配置编号、中文名、英文名、配置名称、说明、业务域 |
| 对象管理 | 新增/删除对象行、业务域、对象多选、制式/模式、格式、周期、生成计划、压缩、目录模板、文件名模板、预览 |
| 传输目标 | 新增/删除 FTP/SFTP 目标、启用、协议、主机、端口、账号、认证方式、凭据、`#FTPRoot#`、重试、超时、FTP 被动模式、测试连接 |
| 字段/指标配置 | 按业务域、制式、对象过滤；搜索未选择字段/指标；新增、删除、编辑输出别名 |
| 启用配置 | 保存前默认关闭，用户确认后开启 |

字段/指标表：

| 列 | 说明 |
|---|---|
| 上报 | 字段级开关，默认关闭 |
| 对象 | CP/EP/CC/CE/PC/PE/MRO/MRE/MRS 等 |
| 制式 | LTE/GNB/GSM 或全部 |
| 输出别名 | 文件表头，可编辑 |
| 系统字段/指标 | 当前 xomc 字段 key 或 PM `metric_path` |
| 取数字段 | 数据库列、参数路径或指标来源 |
| 适用范围 | 支持的机型/ProductClass |
| 类型/口径 | 字段类型，或 counter/KPI + statis_type |
| 单位/渲染 | 单位、quote、timestamp、枚举等 |
| 中文名 | 页面辅助识别 |

新增字段/指标规则：

1. 候选下拉支持模糊搜索。
2. 候选按制式和对象过滤。
3. 已经在当前对象中选择的字段不再出现在候选里。
4. 删除后可以重新搜索并新增。
5. PM 必须保存当前系统 `metric_path`，旧 `KPI_*` 或旧 Mongo 字段名只能作为输出别名。

### 3.5 路径和文件名模板

默认上传目录模板：

| 业务域 | 默认目录 |
|---|---|
| CM | `/#FTPRoot#/#Province#/#OMC-R#/CM/#DateTime#/` |
| PM | `/#FTPRoot#/#Province#/#OMC-R#/PM/#DateTime#/` |
| MR | `/#FTPRoot#/#Province#/#OMC-R#/MR/#DateTime#/` |
| LOG | `/#FTPRoot#/#Province#/#OMC-R#/LOGS/#DateTime#/` |

默认文件名模板：

| 业务域 | 默认文件名 |
|---|---|
| CM | `Baicells-#Object#-#LocalHost#-#DataVersion#-#DateTime#[-#Ri#][-#FileID#]` |
| PM | `Baicells-#Object#-#LocalHost#-#DataVersion#-#DateTime#[-#Ri#]-#DataPeriod#[-#FileID#]` |
| MR | `#ModuleType#-Baicells-#Object#-#LocalHost#-#eNBID#-#DateTime#[-#Ri#].xml` |
| LOG custom | `Northbound-log-{login|operation}-#DateTime#.txt` |
| LOG fix | `Northbound-log-fix-{login|operation}-#DateTime#.csv` |

模板变量必须在页面编辑时预览实际目录和文件名。后端保存原模板，同时运行记录保存最终展开后的本地路径和远端路径。

## 4. Inventory 文件

Inventory 单独成页，不混入北向文件业务域。列表直接展示 4 类 Inventory：

| 类型 | 名称 | 制式 | 格式 |
|---|---|---|---|
| eNB | eNB Inventory | LTE | CSV |
| gNB | gNB Inventory | GNB | CSV |
| GSM | GSM Inventory | GSM | CSV |
| OMC | OMC Inventory | 系统 | CSV |

列表列：

| 列 | 内容 |
|---|---|
| 操作 | 开关 + 下拉：查看、编辑、上报结果、手动执行 |
| 类型 | eNB、gNB、GSM、OMC |
| 名称 | Inventory 名称 |
| 状态 | `正常` 或 `终止` |
| 输出内容 | 类型、制式、字段数摘要 |
| 调度 | 用户友好周期 |
| 文件规则 | CSV 和对象类型 |
| 压缩 | 不压缩、zip 或 gz |

查看抽屉展示基础信息、输出规则、传输目标、字段配置。编辑抽屉支持：

1. 修改名称、统计周期、启用状态、压缩和压缩格式。
2. 编辑生成计划、上传目录模板、文件名模板，并实时预览。
3. 为每类 Inventory 独立维护多个 FTP/SFTP 传输目标。
4. 搜索当前类型下尚未选择的字段并新增。
5. 删除字段后，该字段重新回到候选下拉。
6. 编辑输出别名。

Inventory 默认目录和文件名：

| 类型 | 默认目录 | 默认文件名 |
|---|---|---|
| eNB | `/#FTPRoot#/#Province#/#OMC-R#/Inventory/eNB/#DateTime#/` | `BaiOMC_eNB_#DateTime#.csv` |
| gNB | `/#FTPRoot#/#Province#/#OMC-R#/Inventory/gNB/#DateTime#/` | `BaiOMC_gNB_#DateTime#.csv` |
| GSM | `/#FTPRoot#/#Province#/#OMC-R#/Inventory/GSM/#DateTime#/` | `BaiOMC_GSM_#DateTime#.csv` |
| OMC | `/#FTPRoot#/#Province#/#OMC-R#/Inventory/OMC/#DateTime#/` | `BaiOMC_OMC_#DateTime#.csv` |

数据边界：

1. eNB/gNB/GSM Inventory 字段来自 `devices`、`device_info`、`device_parameters`、设备组、产品字典和告警聚合。
2. OMC Inventory 字段来自系统配置、构建信息和设备聚合统计。
3. 旧 `enbMonitorExport.md` 中支持的字段可以进入默认模板；部分支持字段默认关闭；不支持字段不进入候选。
4. 规划、联系人、线路、维护团队等资产字段当前没有稳定模型，不能伪造输出。

## 5. Socket 告警

Socket 告警是北向服务端能力，OMC 监听端口，上级 NMS/客户端连接 OMC。

列表不提供新增按钮，只展示内置配置：

| 配置 | 协议场景 | 默认监听 | 同步能力 |
|---|---|---|---|
| 电信 Socket 告警 | CTCC | `0.0.0.0:31232` | 实时推送、消息同步 |
| 联通 Socket 告警 | CUCC | `0.0.0.0:31233` | 实时推送、消息同步、文件同步 |

列表列：

| 列 | 内容 |
|---|---|
| 操作 | 开关 + 下拉：查看、编辑、上报结果、测试告警 |
| 配置名称 | 电信 Socket 告警、联通 Socket 告警 |
| 状态 | `正常` 或 `终止` |
| 协议场景 | 电信/联通、UTF-8 |
| 监听地址 | 服务端监听地址 |
| 协议帧 | 电信 16 字节帧头，联通 9 字节帧头 |
| 会话与心跳 | 最大连接、心跳周期、超时阈值 |
| 同步策略 | 实时推送、客户端同步、文件同步 |
| 账号 | 已启用账号类型 |
| 字段模板 | 告警字段数量 |

查看抽屉展示基础信息、账号管理、告警字段映射。联通场景额外展示文件同步传输目标。

编辑抽屉支持：

| 区域 | 配置项 |
|---|---|
| 服务参数 | 协议场景、启用配置、监听地址、监听端口、最大连接数、心跳周期、超时阈值、字符编码 |
| 同步策略 | 实时推送、客户端同步、同步方式、联通连续序号 |
| 账号管理 | 新增/删除账号、启用、类型、用户名、密码、能力范围 |
| 文件同步传输目标 | 联通场景支持多个 FTP/SFTP 目标 |
| 字段映射 | 告警字段与 xomc 告警数据源映射 |

账号规则：

1. 支持多个 Socket 用户名和密码。
2. 账号类型包括 `msg` 和 `ftp`。
3. `msg` 用于登录、实时告警、历史消息同步。
4. `ftp` 用于联通文件同步。
5. 密码保存为密文或 secret 引用，接口不返回明文。

Socket 与 SNMP 的分工：

| 维度 | Socket | SNMP |
|---|---|---|
| 通信模式 | TCP 长连接 | UDP 通知 + SNMP 查询 |
| 典型用途 | 高频实时推送、断点同步、文件补录 | 标准综合网管对接、MIB 查询、Trap/Inform |
| 补录能力 | 强，按序号/时间/文件 | 中，以 MIB 快照和通知为主 |

## 6. SNMP 告警

SNMP 告警是标准北向网管接口。这里的 `Inform` 是 SNMP 通知类型，不是 TR-069 南向设备心跳 Inform。

SNMP 能力包含：

1. NMS 主动查询 OMC 告警 MIB 表。
2. OMC 主动向 NMS 发送 Trap 或 Inform。

SNMP 告警字段、OID 和通知对象以旧系统正式文件 `omcAlarmMIB.mib` 为准。当前 xomc 后端 `internal/northbound/snmp` 仍是早期骨架，存在占位 OID 和简化字段的问题；生产实现必须替换为本文 MIB 口径。

Trap 与 Inform：

| 通知类型 | 含义 | 页面处理 |
|---|---|---|
| Trap | 单向通知，不要求 NMS 确认 | 适合普通实时告警推送 |
| Inform | 需要 NMS 确认 | 需要配置超时和重试 |

列表不提供新增按钮，只展示内置目标行：

| 目标 | 版本 | 通知类型 |
|---|---|---|
| NMS V2 Trap | V2 | Trap |
| NMS V3 Inform | V3 | Inform |

列表列：

| 列 | 内容 |
|---|---|
| 操作 | 开关 + 下拉：查看、编辑、上报结果、测试发送 |
| 目标名称 | NMS V2 Trap、NMS V3 Inform |
| 状态 | `正常` 或 `终止` |
| 版本 | V2 或 V3 |
| 通知 | Trap 或 Inform |
| Agent 监听 | OMC SNMP Agent 监听地址 |
| 通知目标 | NMS 接收地址 |
| 安全配置 | V2 显示 community；V3 显示安全名、认证算法、加密算法 |
| MIB/清除策略 | MIB 查询开关、清除告警 severity 策略 |
| 重试 | 超时秒数和重试次数 |

查看抽屉展示基础信息和 `omcAlarmEntry` 字段映射。编辑抽屉支持：

| 区域 | 配置项 |
|---|---|
| 目标参数 | 版本、通知类型、启用配置、MIB 查询、Agent IP/端口、目标 IP/端口 |
| 安全与运行 | Community、安全名、认证算法、加密算法、清除告警级别、超时、重试次数 |
| 字段映射 | 18 列 omcAlarmEntry 字段映射 |

MIB 口径：

| 项 | OID/说明 |
|---|---|
| 企业根 | `1.3.6.1.4.1.53058` |
| OMC 告警 MIB | `1.3.6.1.4.1.53058.1.1` |
| Notification 根 | `1.3.6.1.4.1.53058.1.1.0` |
| 告警通知 | `omcAlarmNotification = 1.3.6.1.4.1.53058.1.1.0.1` |
| Object 根 | `1.3.6.1.4.1.53058.1.1.1` |
| 告警表 | `omcAlarmTable = 1.3.6.1.4.1.53058.1.1.1.1` |
| 告警表行 | `omcAlarmEntry = 1.3.6.1.4.1.53058.1.1.1.1.1` |

`omcAlarmEntry = 1.3.6.1.4.1.53058.1.1.1.1.1`，索引为 `notificationID`。字段固定 18 列，顺序、字段名、类型和长度约束不能随页面编辑改变：

| 序号 | MIB 字段 | OID | MIB 类型/约束 | xomc 来源/转换 |
|---:|---|---|---|---|
| 1 | `notificationID` | `omcAlarmEntry.1` | `Integer32(1..2147483647)` | 北向告警序列号，需单调递增 |
| 2 | `alarmUniqueId` | `omcAlarmEntry.2` | `OCTET STRING(SIZE(5))` | `alarms.alarm_identifier`，不足/超长需按 MIB 规则校验 |
| 3 | `notificationType` | `omcAlarmEntry.3` | `OCTET STRING(SIZE(1))` | 活动告警输出 `1`，清除告警输出 `0` |
| 4 | `eventTime` | `omcAlarmEntry.4` | `Counter64(13)` | `raised_at/cleared_at` 转 UTC Unix 毫秒 |
| 5 | `equipmentSDN` | `omcAlarmEntry.5` | `OCTET STRING(SIZE(1..45))` | 设备 SN 或设备 DN |
| 6 | `equipmentName` | `omcAlarmEntry.6` | `OCTET STRING(SIZE(0..50))` | `alarms.device_name`，缺失时补 `devices` 名称 |
| 7 | `equipmentClass` | `omcAlarmEntry.7` | `OCTET STRING(SIZE(0..45))` | 设备制式/类型，例如 LTE、GNB、GSM |
| 8 | `objectSDN` | `omcAlarmEntry.8` | `OCTET STRING(SIZE(1..45))` | `alarms.alarm_source` 或对象 DN |
| 9 | `objectInstanceName` | `omcAlarmEntry.9` | `OCTET STRING(SIZE(0..50))` | `alarms.network_location` 或对象名称 |
| 10 | `objectClass` | `omcAlarmEntry.10` | `OCTET STRING(SIZE(0..45))` | `alarms.event_type` 或对象类型 |
| 11 | `additionalText` | `omcAlarmEntry.11` | `OCTET STRING(SIZE(0..256))` | `alarms.description` |
| 12 | `deviceVendorOUI` | `omcAlarmEntry.12` | `OCTET STRING(SIZE(0..10))` | `devices.oui`，缺失时按厂家映射 |
| 13 | `specificProblemID` | `omcAlarmEntry.13` | `OCTET STRING(SIZE(5))` | `alarm_identifier` 或告警定义中的厂家告警码 |
| 14 | `specificProblem` | `omcAlarmEntry.14` | `OCTET STRING(SIZE(0..256))` | 告警标题/描述 |
| 15 | `alarmType` | `omcAlarmEntry.15` | `OCTET STRING(SIZE(5))` | `alarms.alarm_type`，需规范化为 5 位编码 |
| 16 | `perceivedSeverity` | `omcAlarmEntry.16` | `OCTET STRING(SIZE(5..8))` | `alarms.severity` 转字符串级别 |
| 17 | `probableCause` | `omcAlarmEntry.17` | `OCTET STRING(SIZE(0..256))` | `alarms.probable_cause` |
| 18 | `additionalInformation` | `omcAlarmEntry.18` | `OCTET STRING(SIZE(0..256))` | `alarms.additional_info` 序列化后的辅助信息 |

`omcAlarmNotification` 的 OBJECTS 必须使用同一组 18 个字段，顺序与 `omcAlarmEntry` 完全一致。Trap 和 Inform 只改变通知交互方式，不改变 VarBind 字段集合。

实现修正规则：

1. 后端 OID 常量必须改为 `1.3.6.1.4.1.53058.1.1` MIB 树，不能继续使用占位企业号。
2. SNMP mapper 必须输出完整 18 个 VarBind，不能只发送告警 ID、级别、设备 SN 等简化字段。
3. 字段长度和必填范围按 MIB 校验；超长字段要截断并记录审计，不能导致整条通知崩溃。
4. `eventTime` 使用 Unix 毫秒，不使用秒级 Unix 时间。
5. `notificationType` 使用 MIB 定义的 `0(clear)/1(active)`，不是页面上的 `Trap/Inform`。

## 7. 北向 API

北向 API 页只展示旧 northbound API 已支持，且当前 xomc 有对应能力的交集。旧系统没有的当前新增接口不展示。

列表列：

| 列 | 内容 |
|---|---|
| 操作 | 开关 + 下拉：查看 |
| 类型 | 鉴权管理、业务复用、正式北向 |
| 模块 | 鉴权、设备、配置、任务、设备维护 |
| 接口名称 | 面向用户的接口名称 |
| 方法 | GET、POST、PUT、DELETE |
| 接口 URL | 当前 xomc URL |
| 鉴权 | JWT 或无需预置 token |

列表不展示旧系统比对类信息。查看抽屉展示基础信息、请求参数示例、返回结果示例。

当前页面展示接口：

| 模块 | 接口名称 | 方法 | 接口 URL |
|---|---|---|---|
| 鉴权 | 获取访问令牌 | POST | `/api/v1/auth/login` |
| 设备 | 查询设备列表 | GET | `/api/v1/devices` |
| 设备 | 查询设备详情 | GET | `/api/v1/devices/{id}/detail` |
| 配置 | 导出设备参数 | GET | `/api/v1/northbound/export/config/{deviceId}` |
| 配置 | 查询参数树 | GET | `/api/v1/devices/{id}/parameters/tree` |
| 配置 | 同步设备参数 | POST | `/api/v1/config/sync/pull/{device_sn}` |
| 配置 | 修改设备参数 | PUT | `/api/v1/devices/{id}/parameters` |
| 任务 | 查询任务结果 | GET | `/api/v1/devices/tasks/{task_id}` |
| 设备维护 | 重启设备 | POST | `/api/v1/devices/{id}/reboot` |

API 设计原则：

1. 页面展示当前系统 URL，不展示老系统 URL。
2. 请求/响应示例按当前 xomc 返回结构展示。
3. 如果后续要求老 OSS 无改造接入，应新增 `/v1` 兼容 adapter，而不是在页面把老接口和新接口混在一起。

## 8. FTP/SFTP 传输目标

北向文件、Inventory、联通 Socket 文件同步都需要文件投递目标。目标不单独成页，放在对应功能的编辑抽屉中。

传输目标支持多条配置：

| 字段 | 说明 |
|---|---|
| 启用 | 单目标开关 |
| 目标名称 | 运维可识别名称 |
| 协议 | FTP 或 SFTP |
| 主机/端口 | 目标地址 |
| 用户名 | 登录账号 |
| 认证方式 | 用户名密码或私钥 |
| 密码/私钥 | 加密存储，页面不回显明文 |
| `#FTPRoot#` | 远端根目录变量 |
| 重试/超时 | 目标级投递策略 |
| FTP 被动 | FTP 专用 |
| 测试连接 | 保存前验证连通性 |

同一文件可以按目标配置 fanout 到多个 FTP/SFTP。后端运行记录必须按“文件 run + 目标 delivery”记录每个目标的结果。

## 9. 数据准确性规则

页面只能配置当前 xomc 可输出的数据。旧场景字段、旧 XML Map、旧接口文档不能直接作为真实数据来源。

| 数据 | 当前来源 |
|---|---|
| 设备基础信息 | `devices`、`device_info`、`products`、设备组 |
| 设备参数字段 | `device_parameters` 或稳定参数投影 |
| CM 文件 | 设备基础信息、参数投影、系统配置 |
| PM counter/KPI | `pm_metric_dictionary`、`perf_indicators_enb/gsm/gnb`、PM 聚合结果 |
| MR 文件 | `mr_files`、对象存储记录或 MR 原始文件 |
| LOG 文件 | 登录、操作、任务、API 调用等审计日志 |
| Inventory | 设备信息聚合、系统配置、构建信息 |
| Socket 告警 | 告警事件投影、活动/历史告警 |
| SNMP MIB | 活动告警、设备信息补全、统一告警 mapper |
| API | 当前 Go handler/DTO 能力 |

支持状态处理：

| 状态 | 页面处理 |
|---|---|
| 支持 | 可进入候选，允许启用 |
| 部分支持 | 可进入候选但默认关闭，需产品/现场确认语义 |
| 不支持 | 不进入候选，只在兼容报告里说明 |

PM 规则：

1. 当前指标库合计 1763 个指标，其中 LTE 1408、GNB 282、GSM 73。
2. 旧 GNB PC 119 项全部命中当前指标库。
3. 旧 GSM PC 14 项中 12 项命中。
4. 旧 LTE PC 1347 个唯一 Map 中 827 个命中，520 个缺失。
5. 缺失指标如果业务必须输出，应先进入指标库和 `pm_metric_dictionary`，再允许页面选择。

Inventory 规则：

1. 默认模板只启用支持字段。
2. 部分支持字段默认关闭。
3. 不支持字段不进入字段选择器。
4. 人工资产字段必须先建设资产模型，不能临时塞扩展字段无约束输出。

## 10. 后端模型建议

后端保存配置时建议拆成能力配置、对象/字段配置、传输目标和运行记录。

| 模型 | 作用 |
|---|---|
| `northbound_file_profiles` | 场景/配置头：编号、名称、版本、启用状态 |
| `northbound_file_tasks` | CM/PM/MR/LOG 对象行：业务域、对象、格式、周期、模板、压缩 |
| `northbound_field_mappings` | 字段/指标配置：输出别名、系统字段、metric_path、类型、单位、渲染规则 |
| `northbound_inventory_profiles` | eNB/gNB/GSM/OMC Inventory 配置 |
| `northbound_endpoints` | FTP/SFTP、Socket、SNMP 目标/账号 |
| `northbound_file_runs` | 文件生成运行记录 |
| `northbound_file_deliveries` | 文件单目标投递记录 |
| `northbound_alarm_events` | Socket/SNMP 共用告警事件投影 |
| `northbound_socket_sessions` | Socket 登录、心跳、同步审计 |
| `northbound_api_clients` | 北向 API 客户端、scope、IP 白名单、限流 |

安全要求：

1. FTP/SFTP 密码、私钥、Socket 密码、SNMP V3 auth/priv 只保存密文或 secret 引用。
2. API 响应只返回是否已配置，不返回明文。
3. 手动执行、测试发送、下载最新文件必须写审计日志。

## 11. 运行链路

```text
页面编辑配置
  -> 保存草稿
  -> 后端校验字段/指标/格式/周期/目标
  -> 启用配置
  -> 调度或手动执行
  -> 生成文件/推送告警/发送 Trap 或 Inform
  -> 写运行记录和目标结果
  -> 列表状态刷新为 正常 或 终止
  -> 操作 -> 上报结果 查看详情或下载最新文件
```

配置变更热加载建议：

| 事件 | 消费方 |
|---|---|
| `northbound.file.profile.changed` | 文件调度器、文件生成器 |
| `northbound.inventory.profile.changed` | Inventory 调度器 |
| `northbound.endpoint.changed` | 文件投递器、Socket、SNMP |
| `northbound.socket.config.changed` | Socket 服务端 |
| `northbound.snmp.target.changed` | SNMP Engine |
| `northbound.api.client.changed` | API 鉴权/限流中间件 |

## 12. 管理 API 建议

页面所需管理 API：

| 方法 | 路径 | 用途 |
|---|---|---|
| GET | `/api/v1/northbound/file/profiles` | 北向文件配置列表 |
| POST | `/api/v1/northbound/file/profiles` | 新增配置 |
| PUT | `/api/v1/northbound/file/profiles/{id}` | 保存草稿 |
| POST | `/api/v1/northbound/file/profiles/{id}/run` | 手动执行 |
| GET | `/api/v1/northbound/file/profiles/{id}/last-result` | 上报结果 |
| GET | `/api/v1/northbound/file/runs/{runId}/download` | 下载最新文件 |
| GET | `/api/v1/northbound/inventory/profiles` | Inventory 列表 |
| PUT | `/api/v1/northbound/inventory/profiles/{type}` | 保存 Inventory 配置 |
| POST | `/api/v1/northbound/inventory/profiles/{type}/run` | 手动执行 Inventory |
| GET | `/api/v1/northbound/socket/configs` | Socket 配置列表 |
| PUT | `/api/v1/northbound/socket/configs/{id}` | 保存 Socket 配置 |
| POST | `/api/v1/northbound/socket/configs/{id}/test` | 测试告警 |
| GET | `/api/v1/northbound/snmp/targets` | SNMP 目标列表 |
| PUT | `/api/v1/northbound/snmp/targets/{id}` | 保存 SNMP 目标 |
| POST | `/api/v1/northbound/snmp/targets/{id}/test` | 测试 Trap/Inform |
| GET | `/api/v1/northbound/apis` | 北向 API 清单 |
| PUT | `/api/v1/northbound/apis/{key}/enabled` | 修改 API 开关 |

## 13. 前端实现要求

当前原型文件：

| 文件 | 说明 |
|---|---|
| `omcmb/webcode/src/pages/config/NorthboundPageConfig/index.tsx` | 页面逻辑和原型数据 |
| `omcmb/webcode/src/pages/config/NorthboundPageConfig/index.module.css` | 页面样式 |
| `omcmb/webcode/src/router/routes.tsx` | 路由入口 |
| `omcmb/webcode/src/router/componentRegistry.ts` | 组件注册 |

后续从原型转生产时：

1. 原型常量数据替换为管理 API 数据。
2. 保存草稿、启用、手动执行、测试连接、测试告警、测试 Trap/Inform 接真实后端。
3. `上报结果` 从运行记录接口读取。
4. 字段/指标候选从数据字典和 PM 指标接口读取。
5. 页面保留当前列表列结构和抽屉交互，不再新增解释性列。

## 14. 验收重点

页面验收：

1. 北向文件列表只显示 17 个场景编号和中文场景名，不出现内部文件名和来源说明。
2. 北向文件业务域下拉不出现 Inventory。
3. CM/PM 不允许选择 XML，MR 允许 XML，Inventory 只允许 CSV。
4. 所有开关默认关闭。
5. 列表 `状态` 只显示 `正常` 或 `终止`。
6. `上报结果` 在操作下拉中打开，文件类可下载最新文件。
7. Inventory 页面和北向文件页面风格一致。
8. Socket/SNMP 列表页没有新增按钮。
9. Socket 编辑里可以维护多个用户名/密码。
10. SNMP 版本只显示 V2/V3。
11. SNMP 通知类型显示 Trap/Inform，并支持超时/重试。
12. 北向 API 列表不显示旧系统比对类信息。
13. 北向 API、Inventory、北向文件分页组件保持一致。

后端验收：

1. 保存配置时校验格式和业务域边界。
2. PM 保存时校验 `metric_path` 存在。
3. 不支持字段不能通过 API 强行保存为启用状态。
4. 多个 FTP/SFTP 目标分别记录投递结果。
5. 手动执行和定时执行都生成运行记录。
6. Socket 实时推送和客户端同步使用同一告警事件序列。
7. SNMP Trap/Inform 与 MIB 查询使用同一告警 mapper。
8. 密码和密钥不明文返回。

## 15. 分期实施

| 阶段 | 内容 |
|---|---|
| P1 | 后端配置模型、字段字典、传输目标、运行记录 |
| P2 | 北向文件和 Inventory 生成/投递闭环 |
| P3 | Socket 告警服务端、账号、实时推送、历史同步、联通文件同步 |
| P4 | SNMP V2/V3、Trap/Inform、MIB 查询、测试发送 |
| P5 | 北向 API 开关、客户端、鉴权、限流、调用日志 |
| P6 | 旧 XML 离线导入工具和兼容报告，不进入主业务页面 |

## 16. 默认假设

1. 所有北向配置默认关闭。
2. 页面显示只以当前 xomc 可输出数据为准。
3. 旧场景、旧 XML、旧文档是模板来源，不是运行时配置。
4. 北向文件、Inventory、Socket、SNMP、API 可以独立启停。
5. 文件投递目标支持多个 FTP/SFTP。
6. API 兼容老 OSS 不是本页面默认目标；需要时另做 `/v1` adapter。
