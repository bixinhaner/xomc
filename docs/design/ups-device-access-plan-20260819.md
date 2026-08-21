# UPS 设备接入开发计划

日期：2026-08-19

开发分支：`feature/ups-device-support`

开发 worktree：`/Users/renpengfei/code/goomc/goomcnew/xomc-ups`

参考文档：`/Users/renpengfei/Desktop/doc/futuredoc/UPS-功能梳理与重构参考文档.md`

## 1. 目标与边界

本次接入目标是在现有 OMC 体系中新增 UPS 设备类型，并完整承接旧系统 UPS 文档中描述的接入、监控、详情、告警、升级、重启、分组、注册、License、导出等能力。

UPS 不作为基站型号的变种处理，而是作为独立设备类型接入当前系统的标准化能力：

- 产品管理：UPS 是产品库中的标准产品。
- 数据模型管理：UPS 有独立参数模型 `UPS.xml`。
- 告警管理：UPS 有独立告警定义 `UPS.xml`，`neType=UPS`。
- 设备列表：设备列表通过 Tab 分开，UPS 使用专属 Tab 和专属列。
- 文件传输：UPS 升级通过 UFTE 内置模板标准化承载。

硬性识别规则：TR-069 Inform 中 `ProductClass` 以大写 `UPS` 开头即判定为 UPS 设备，例如 `UPS_M3_BMU`。该规则大小写敏感，不依赖 vendor、radio mode、technology。

## 2. 旧系统功能适配矩阵

| 功能域 | 旧系统要求 | 当前系统适配方式 | 状态 |
|---|---|---|---|
| 设备识别 | `ProductClass.startsWith("UPS")` 判定 UPS | 产品库 `^UPS.*` + Inform 热路径前缀判断 | 已实现 |
| TR-069 根路径 | UPS 使用 `InternetGatewayDevice.` | UPS 参数模型独立使用 IGD 根路径，不混入基站模型 | 已实现 |
| 产品管理 | UPS 独立产品 | `products.xml` 增加 `UPS`，`paramModel=UPS`，`alarm neType=UPS` | 已实现 |
| 数据模型管理 | 覆盖 DeviceInfo、ChargerInfo、BMSInfo、电池包、告警路径 | 新增 `data/param-mappings/UPS.xml` | 已实现 |
| 告警定义 | 19 条告警，ID `42000..42018`，`deviceType=5`，`neType=UPS` | 新增/校验 `data/alarm-definitions/UPS.xml` | 已实现 |
| Inform 入库 | 解析 UPS 所有关键 PATH，写 UPS 缓存/DB | 保留原始参数到 `device_parameters`，同时写 UPS 投影表 | 已实现 |
| 新 UPS 电池包 | `PackCounts` 非空时读 `BMSInfo.1..N.*`，跳过空 SN/`N/A` | `device_ups_batteries` 按 `PackCounts` 投影 | 已实现 |
| 老 UPS 电池包 | `PackCounts` 为空但非索引 BMS 有值时兼容 1 包，缺型号/版本/SN 显示 `--` | 兼容非索引 BMS，缺字段投影为 `--` | 已补齐 |
| 设备列表 | UPS 专属监控列表字段，不展示基站小区/MME/同步/RF 字段 | 设备列表新增 UPS Tab，独立列配置 ID，UPS 专属 DTO | 已实现并补列 |
| 设备详情 | 三块：电源系统参数、电源运行参数、电池运行参数 | `/devices/:id/detail` 追加 `ups` 三段 DTO，前端 UPS 详情分支展示 | 已补齐 |
| CSV 导出 | UPS 列表导出 UPS 监控列，SOC 带 `%`，电压/电流带单位 | 当前列表导出按可见列导出，UPS 专属列补单位格式化 | 已补齐 |
| 分组 | UPS 可分组、移动分组，一个 UPS 属于一个组 | 复用统一 `device_groups` / `device_group_members`，列表 API 不限制设备类型时包含 UPS | 已补齐 |
| 注册 | 单台注册、Excel 导入、模板下载；旧系统 `ups_+sn` 有重复风险 | 单台注册可选 UPS 并默认 `ProductClass=UPS_M3_BMU`；批量预登记 CSV 增加 `ProductClass`；按 SN 统一设备身份 | 已补齐 |
| License | `isSupportUPS`、`UPSMaxNum` 控制接入容量 | 旧 `upsMaxNum` 已映射到 `devices_support["UPS"]`；UPS 产品 `neType=UPS` 后走现有 LicenseEnforcer | 已补齐，需真实 license 回归 |
| 在线超时 | 旧 UPS 默认 `UpsTimeout=300s` | 新增 `upsTimeout`，默认 300 秒，SQL 和在线索引均按 UPS 优先分类 | 已补齐 |
| 告警解析 | VALUE CHANGE 触发 UPS 告警，equipInfo 用 `SN=<sn>` | UPS 告警定义已接入；ACS 对 `VALUE CHANGE + FaultMgmt.CurrentAlarm.*` 额外发布标准告警事件；UPS 告警 `additional_info.equipment_info=SN=<sn>` | 已补齐，需真实报文联调 |
| UPS 升级模板 | UPS 文件类型 `file_type=5`/AP，Download + TransferComplete + BOOT 比版本 | UFTE 新增 `UPS_AP_UPGRADE`，AP 固件上传、TC 后等待 BOOT、版本核验均已接入 | 已补齐 |
| UPS 无反向连接 | 不走 STUN/UDP，命令主要等下次 Inform piggyback | 当前命令队列天然支持 Inform PopTask；UPS 升级抽屉提示“等待设备下次心跳下发” | 已补齐 |
| UPS 重启 | 支持 Reboot，设备 BOOT 后收敛 | 复用当前 `RebootDevice` / Reboot RPC | 已实现并测试 |
| 旧 Redis 队列 | `upsInformQueue`、`{upsCode}_upsCache` 等 | 不复制旧 Redis 名称，改由当前事件/PG 投影承载 | 等价替代 |

## 3. 数据与模型适配

### 3.1 产品模型

`data/param-mappings/products.xml` 增加 UPS 产品：

- `name="UPS"`
- `paramModel="UPS"`
- `alarm neType="UPS"`
- `pattern="^UPS.*"`
- `enableFileType11=false`

产品匹配优先使用产品库规则。Inform 热路径保留 UPS 前缀判断，避免在产品库未加载或缓存延迟时把 UPS 错识别为基站。

### 3.2 参数模型

新增 `data/param-mappings/UPS.xml`，所有 UPS 私有路径以 `InternetGatewayDevice.` 为根。必须覆盖：

- 设备信息：`Manufacturer`、`ManufacturerOUI`、`ProductClass`、`SerialNumber`、`HardwareVersion`、`SoftwareVersion`、`UpTime`
- 网络地址：`WANDevice.1.WANConnectionDevice.1.WANIPConnection.1.ExternalIPAddress`
- 充电器信息：`ACPower`、`ACVoltage`、`DCVoltage`、`DCCurrent`、`BoardTemperature`、`SFPState`、`PORT0State` 至 `PORT3State`、`AverageSOC`、`PackCounts`
- BMS 总览：`BMSInfo.Voltage`、`Temperature`、`Current`、`Charging`
- 电池包：`BMSInfo.{i}.SOC`、`Voltage`、`Temperature`、`Current`、`Status`、`RecycleCount`、`Charging`、`Model`、`SoftwareVersion`、`SerialNumber`
- 告警相关：`CurrentAlarm.*`、`ExpeditedEvent.*`

### 3.3 UPS 运行态投影

新增 `device_ups_runtime`，用于列表和详情高频字段：

- `device_id`、`device_serial_number`
- `external_ip`
- `total_voltage`、`total_temperature`、`total_current`、`bms_charging`
- `software_version`、`hardware_version`、`manufacturer`、`manufacturer_oui`、`up_time_seconds`
- `ac_power`、`ac_voltage`、`dc_voltage`、`dc_current`、`board_temperature`
- `sfp_state`、`port0_state`、`port1_state`、`port2_state`、`port3_state`
- `average_soc`、`pack_counts`、`last_inform_at`

新增 `device_ups_batteries`，用于电池包明细：

- `device_id`、`device_serial_number`、`pack_index`
- `serial_number`、`soc`、`voltage`、`temperature`、`current_value`
- `status`、`recycle_count`、`charging`、`model`、`software_version`

投影表只服务读模型，原始参数仍进入 `device_parameters`，保证参数树、参数同步、MML/GPV/SPV 能复用现有机制。

## 4. Inform 接入流程

UPS 接入流程必须满足：

1. 收到 Inform 后读取 `ProductClass`。
2. `ProductClass` 以 `UPS` 开头时，产品匹配为 UPS，设备类型输出为 UPS。
3. 设备注册/更新仍写统一 `devices` 和 `device_parameters`。
4. 参数入库后执行 UPS 运行态投影，失败只记录日志，不阻断 Inform 主流程。
5. 软件版本优先读 `Device.DeviceInfo.SoftwareVersion`，UPS 兜底读 `InternetGatewayDevice.DeviceInfo.SoftwareVersion`。
6. UPS IP 优先读 `InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANIPConnection.1.ExternalIPAddress`。
7. Connection Request URL/UDP 地址按现有通用字段读取，并补 IGD ManagementServer 兼容。
8. VALUE CHANGE 事件进入 UPS 告警解析：`FaultMgmt.CurrentAlarm.*` 转发到标准告警事件，`FaultMgmt.ExpeditedEvent.*` 转发到实时告警事件。
9. `1 BOOT` 事件需要同时服务普通重启收敛和 UPS 升级后版本核验。

## 5. 设备列表与导出

设备列表按 Tab 展示：

- 基站 Tab：默认入口，过滤 `ProductClass NOT LIKE 'UPS%'`。
- UPS Tab：过滤 `ProductClass LIKE 'UPS%'`。
- 保留内部 `ALL` 能力，但不作为默认页面入口。

UPS Tab 默认/可选列应覆盖旧系统监控列：

- 连接状态
- 告警级别/告警数量
- 市电状态 `ACPower`
- 设备序列号
- 产品类型 / 设备型号
- 软件版本 / 运行时长
- 站点 ID / 设备名称 / IP 地址
- BMS 总电压、总温度、总电流
- AC 电压、DC 电压、DC 电流
- 板温
- SFP 状态
- Port0 至 Port3 状态
- 平均 SOC
- 电池包数量

展示规则：

- SOC：`<40` 红色，`40..69` 黄色，`>=70` 绿色，导出追加 `%`。
- 板温：`<=0` 红色，`0..10` 黄色，`10..40` 绿色，`40..60` 黄色，`>=60` 红色。
- 电压导出追加 `V`，电流导出追加 `A`，温度导出追加 `°C`。
- UPS Tab 隐藏基站制式筛选，避免出现 LTE/NR/GSM。
- 基站 Tab 不展示 UPS 电源/电池字段。

## 6. 设备详情

UPS 详情在统一 `/devices/:id/detail` 中追加 `ups` 字段，保持原基站详情结构不变。

UPS 详情分三块：

- `power_system_param`：厂商、OUI、SN、硬件版本、软件版本、运行时长、ProductClass。
- `power_run_param`：IP、BMS 总电压/温度/电流、BMS 充放电、市电、AC/DC 电压电流、板温、SFP、Port0..3、平均 SOC、电池包数量、最近 Inform。
- `battery_run_param`：每个电池包的 SOC、电压、温度、电流、状态、循环次数、充放电、型号、SN、软件版本。

前端 UPS 详情只在 UPS 设备上启用该三块布局；基站、GSM、5G 继续使用原有站点/小区/状态布局。

## 7. 告警

UPS 告警要求：

- 告警定义文件 `data/alarm-definitions/UPS.xml`。
- 告警 ID：`42000..42018`。
- `neType=UPS`。
- `deviceType=5`。
- 默认等级：旧文档为 Major `31002`，分类 `30003`。
- 内置告警库内容按旧文档 11.1 落库：英文名、中文名、触发判定覆盖 `42000..42018` 共 19 条。
- 告警源设备信息使用 `SN=<serial_number>`，不拼 CellName。
- VALUE CHANGE 触发告警解析。
- 告警管理页面可按 UPS 网元类型筛选、查看、启停、维护规则。

已完成告警定义接入。ACS 在收到 `4 VALUE CHANGE` 且参数中包含 `Device.FaultMgmt.CurrentAlarm.*` 或 `InternetGatewayDevice.FaultMgmt.CurrentAlarm.*` 时，会在保留原 `device.inform.value_change` 的同时发布 `device.inform.alarm`，使现有告警模块按统一链路消费 UPS CurrentAlarm。UPS 告警 `additional_info.equipment_info` 写入 `SN=<serial_number>`。后续联调必须用真实或样例 VALUE CHANGE 报文验证告警统计、筛选和页面显示口径。

## 8. 文件传输、UPS 升级与重启

### 8.1 UPS 升级模板

新增 UFTE 内置模板：

- `typeCode=UPS_AP_UPGRADE`
- `category=ups_upgrade`
- `displayName=UPS 软件升级`
- `Products=["UPS"]`
- `PlatformScope=["UPS"]`
- `softwareTaskType=TaskTypeUpgrade`
- 固件库 `firmware_versions.file_type=5`
- TR-069 Download `FileType="1 Firmware Upgrade Image"`
- 下载路径 `/smallcell/FileDownloadService/firmware/ap/{path}`

这里区分两个 file type：

- 固件库 file type `5`：用于 AP/UPS 固件上传、筛选、任务选择。
- TR-069 Download FileType `1 Firmware Upgrade Image`：用于下发给 UPS 设备，保持 CWMP 协议语义。

### 8.2 UPS 升级状态机

旧系统要求 UPS 完整流程为：

1. 根据文件后缀识别 PATCH / CONFIG / FIRMWARE，UPS AP 固件按 FIRMWARE Download 下发。
2. 下发 Download RPC，`CommandKey` 使用 `Download Upgrade,<UUID>` 风格或当前系统等价关联键。
3. 等待 DownloadResponse。
4. 等待 TransferComplete。
5. TransferComplete 成功后不立刻完成，进入等待重启/版本核验状态。
6. 收到后续 `1 BOOT` Inform 后读取 UPS 软件版本。
7. 比较上报版本与任务目标版本。
8. 一致则完成；不一致则失败并记录明确失败原因。
9. 等待超时、任务暂停、任务终止都要进入对应终态/中间态。

当前 worktree 已完成模板、AP 固件类型、受管下载 URL、UPS 任务归类、TC 后等待 BOOT、BOOT 后软件版本核验：

- UPS TransferComplete 成功后进入 `rebooting`，不按 4G 立即 completed。
- `device.inform.reboot_complete` 中对 UPS `1 BOOT` 做版本比较。
- 版本不一致时写 `VERSION_MISMATCH` 和明确错误信息。
- UI 状态链展示 `WAIT_REBOOT_COMPLETE`。

UPS 无反向连接时，UPS 升级抽屉展示“命令将在设备下次 Inform/心跳时下发”的提示。

### 8.3 UPS 重启

UPS 支持复用当前设备重启能力：

- 设备列表 UPS Tab 单台/批量重启调用现有 Reboot RPC。
- 后端 `RebootDevice` 不按无线制式限制，只要设备存在即可创建 Reboot 命令。
- 如果设备没有可用 Connection Request URL，命令在下一次 Inform 时由 ACS PopTask 下发。
- UPS 上报 `1 BOOT`/`M Reboot` 后进入现有重启完成收敛链路。

## 9. 注册、分组与 License

### 9.1 注册

旧系统有 UPS 单台注册、Excel 导入和模板下载。当前系统复用统一设备注册/导入能力，并已补 UPS 设备默认适配：

- ProductClass 可填 UPS 型号，或设备首次 Inform 自动注册。
- 单台注册可选择 UPS，默认 `ProductClass=UPS_M3_BMU`，并校验 UPS 设备的 ProductClass 必须以 `UPS` 开头。
- 批量预登记模板增加 `ProductClass` 列，UPS 设备填写 `UPS*` 后可在首次 Inform 前进入 UPS 识别口径。
- 默认 technology 仅满足当前 `devices.technology` 枚举/分区键要求，不作为 UPS 展示依据。
- 序列号校验可沿用通用规则，必要时兼容旧系统 `^([0-9]|[a-zA-Z]|-)+$`。
- 不复制旧系统 `ups_+sn` 与 TR-069 `OUI_SN` 可能重复的缺陷，新系统统一以 `serial_number` 作为设备主身份。

### 9.2 分组

UPS 使用统一设备组模型：

- 一个设备同一时刻只归属一个设备组。
- UPS 列表和分组页可按组筛选、移动组、导出。
- 旧表 `rela_device_group_ups` 不迁移，映射为 `device_group_members`。
- 设备分组页调用通用设备列表接口，未指定 `device_type` 时包含 UPS；移动分组复用 `BatchAssignToGroup`，一个 UPS 设备只保留一条有效 membership。

### 9.3 License

当前 worktree 已确认并补齐：

- 旧 TrueLicense `upsMaxNum` 映射为 `devices_support["UPS"]`。
- Inform 首次接入 UPS 时按 UPS 容量检查。
- 手工注册 UPS 时按当前系统策略做过期/容量检查。
- `CODE_UPS` 可进入功能授权树和系统 License 页面展示。

仍需真实 license 文件回归：

- 旧 license 未包含 `upsMaxNum` 或 UPS 容量为 0 时，UPS 注册/上线应被拒绝。
- 若产品要求 `isSupportUPS=false` 也隐藏 UPS 功能入口，需要在菜单 feature_code 中补 UPS 页面授权映射。

## 10. 在线超时

旧系统 UPS 默认超时为 300 秒。当前 worktree 已新增 UPS 独立阈值：

- sys_config：`category=device`，`key=upsTimeout`，默认 `300`。
- Reconciler SQL 分类优先级：UPS → CPE → BaseStation。
- 在线索引二次确认也使用 UPS 阈值。
- 扫描周期按 ENB/CPE/UPS 三者最小阈值的一半计算，上限仍 60 秒。

这样 UPS 超时调整不会影响基站和 CPE。

## 11. 兼容与风险控制

- 所有 UPS 差异化逻辑必须以 `ProductClass LIKE 'UPS%'` 或产品库 UPS 命中为边界。
- 不改变非 UPS 产品匹配和既有参数模型。
- 不改变基站默认设备列表入口。
- UPS 投影表是附加读模型，不替代 `devices` 和 `device_parameters`。
- Inform 主流程对 UPS 投影失败 fail-soft。
- UPS 升级模板限定 `Products=["UPS"]`，AP 固件限定 `file_type=5`。
- 不复制旧系统 Redis 队列和旧表名，除非真实业务依赖需要兼容迁移。
- 不复制旧系统 UPS 设备编码重复风险，统一以 SN 为主身份。

## 12. 测试与验收

后端测试：

- `ProductClass=UPS*` 命中 UPS，非 UPS 不命中。
- UPS 新版本电池包按 `PackCounts` 读取，跳过空 SN 和 `N/A`。
- UPS 老版本非索引 BMS 兼容，缺型号/版本/SN 为 `--`。
- UPS 设备列表 `device_type=UPS` 只返回 UPS，基站 Tab 排除 UPS。
- UPS 详情接口返回 `power_system_param`、`power_run_param`、`battery_run_param`。
- UPS 告警定义 19 条，ID 和 `deviceType=5` 校验。
- AP 固件 `file_type=5` 可上传，Download URL 走 `/firmware/ap/`。
- UPS 升级任务优先匹配 `UPS_AP_UPGRADE`。
- UPS TransferComplete 后等待 BOOT 版本核验。
- UPS 重启可创建 Reboot 命令任务。
- UPS timeout 使用 `upsTimeout=300`，不影响 ENB/CPE。

前端测试：

- 类型检查通过。
- 设备列表默认进入基站 Tab。
- `deviceType=UPS` URL 可直接进入 UPS Tab。
- UPS Tab 展示 UPS 专属列，隐藏网络制式筛选。
- UPS 列导出带单位和 SOC `%`。
- UPS 详情展示三块运行参数，基站详情不受影响。
- 文件传输存在 `UPS升级` 分类和 `UPS_AP_UPGRADE` 模板。
- 固件上传支持 AP/UPS 文件类型。
- UPS 无反向连接时任务页面有等待下次 Inform 的提示。

回归测试：

- 基站/CPE/GSM/5G Inform 注册、更新、参数同步。
- 原设备列表筛选、列配置、导出。
- 原 4G/5G/GSM 升级模板和 IMAGE/PATCH/FPGA 上传。
- 产品管理、数据模型管理、告警管理原有功能加载。

## 13. 分阶段落地

### P0：接入骨架

- 建立 UPS worktree 和 feature 分支。
- 增加产品库、参数模型库、告警库基础资产。
- 增加 UPS Inform 识别和运行态投影。

状态：已完成。

### P1：列表可用

- 后端列表接口增加 `device_type`。
- 前端设备列表增加基站/UPS Tab。
- UPS Tab 接入 `ups_summary` 字段并配置专属列。
- UPS 列表导出补单位和 SOC 格式。

状态：已完成。

### P2：详情与基础运维闭环

- UPS 详情接口返回三块结构化 DTO。
- UPS 详情页展示电源系统、电源运行、电池运行。
- UPS Reboot 复用现有重启能力。
- 注册/导入/分组做 UPS 默认值和回归验证。

状态：详情、Reboot、单台注册 UPS 默认值、批量预登记 `ProductClass`、分组通用模型适配已完成；现场仍建议做一次页面冒烟。

### P3：升级与超时闭环

- UPS 升级模板和 AP 固件库接入。
- TransferComplete 后进入等待 BOOT/版本核验。
- BOOT 后比较目标版本和 Inform 上报版本。
- 新增 UPS 300 秒在线超时阈值。
- UPS 无反向连接提示。

状态：模板、AP 固件库、版本核验、300 秒超时、无反向连接 UI 提示已完成。

### P4：License 与告警联调

- UPS License 容量和支持开关接入。
- VALUE CHANGE 告警样例联调。
- 告警 `SN=<sn>` equipInfo 口径验证。
- 告警统计、筛选、CSV/页面显示回归。

状态：License 容量、旧 `upsMaxNum` 映射、UPS 告警 `equipment_info` 已完成；真实 license 文件和真实 VALUE CHANGE 报文仍需联调回归。

## 14. 当前 worktree 已落代码摘要

- 新增 UPS 产品定义和参数模型。
- 校验 UPS 告警定义进入内置资产。
- 新增 UPS 运行态和电池包投影表。
- Inform 注册/更新路径增加 UPS 投影。
- 设备列表增加基站/UPS Tab、UPS 专属过滤和列配置。
- UPS 列表 DTO 增加 `device_type` 和 `ups_summary`。
- UPS 详情接口增加 `ups` 三段 DTO。
- UPS 详情页增加三段运行参数展示。
- 文件传输增加 `UPS_AP_UPGRADE` 模板。
- UPS 升级 TransferComplete 后等待 `1 BOOT`，并校验上报软件版本。
- UPS 升级抽屉提示无反向连接时等待下次 Inform 下发。
- 固件上传增加 AP 文件类型 `file_type=5`。
- UPS 重启复用 Reboot RPC 并有单元测试覆盖。
- UPS 在线超时默认 300 秒，且不影响 ENB/CPE 阈值。
- License 旧 `upsMaxNum` 到 `devices_support["UPS"]` 已有映射，UPS 注册走 `neType=UPS` 容量检查。
- UPS 告警落库补 `additional_info.equipment_info=SN=<sn>`。
- 单台注册可选择 UPS 并默认 `ProductClass=UPS_M3_BMU`；批量预登记模板支持 `ProductClass` 列。

## 15. 最终验收标准

- `ProductClass=UPS*` 的设备自动进入 UPS Tab。
- 非 UPS 设备仍进入基站 Tab 或现有对应视图。
- UPS 设备在产品管理、数据模型管理、告警管理中都有标准化数据。
- UPS 列表不出现基站小区/MME/AMF/RF/同步字段，基站列表不出现 UPS 电源/电池字段。
- UPS 详情展示旧系统文档要求的三块信息。
- UPS 告警 19 条可解析、筛选、统计，设备信息口径为 SN。
- UPS 升级从 AP 固件下发，TransferComplete 后等待 BOOT，并按版本一致性完成/失败。
- UPS 可重启，无反向连接时等待下次 Inform 下发。
- UPS 在线超时默认 300 秒，不影响基站/CPE。
- 所有新增能力通过单元测试、XML 校验、前端 typecheck，并完成基站主流程回归。
