# MML 控制台空 PATH 命令 — 标准规范 PATH 对照留档

> **生成日期**：2026-06-09  
> **范围**：`mml/console-v2` 中 LST/MOD 类型、参数 PATH 为空的 12 条命令  
> **数据源**：本地 catalog（`mml_commands` / `mml_command_sub_fields` / `standard_params`）+ 运营商规范 `规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md`  
> **性质**：分析留档，不含代码改动  

---

## 1. 背景与根因

`mml/console-v2` 选择 LST/MOD 命令后，参数 PATH 来自 `mml_command_sub_fields`，而 sub_field 通过 `standard_path_id` 外键指向 `standard_params` 字典表。**只有当命令声明的 standardPath 已存在于 `standard_params` 时，sub_field 才会被建立**（loader 对字典中不存在的 path 静默跳过）。

本文 12 条命令的参数 PATH 为空，**全部因其声明的 standardPath 未进 `standard_params` 字典**（并非设备 paramModel 过滤所致）。这 12 条命令在运营商规范 `cmcc-tdlte-southbound-data-model-v2.3` 中**均有明确定义**（下表逐条列出规范中对应的 PATH、参数名、中文名、权限、类型），只是尚未导入字典。

> 两大族：**NR 异系统邻区**（5G-NR 邻区，2 条）与 **扩展型一体化皮基站参数**（MU/槽位/扩展单元/射频远端单元/射频通道/软件升级硬件盘点，10 条）。

| 指标 | 数值 |
|------|------|
| 空 PATH 命令数 | 12（LST 7 + MOD 5） |
| 涉及命令分组 | 2（邻区参数管理 / 扩展型一体化皮基站参数） |
| 声明 PATH 总数 | 85 |
| 规范中可查到定义的 PATH | 85 / 85 |
| 当前在 `standard_params` 字典中 | 0 / 85（全部缺失，故 PATH 为空） |

---

## 2. 逐命令规范 PATH 对照

> 「权限」RW=可读写、R=只读；「类型」括号内为规范给出的取值范围/长度；「TR-181 标准路径」即应导入 `standard_params.standard_path` 的值。

### 2.1 分组：扩展型一体化皮基站参数

#### 列出 射频单元软件升级 （`LST RU_SW_UPGRADE`，3 个 PATH）

| # | TR-181 标准路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|----------------|--------|--------|------|------|
| 1 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.SwUpgrade.Stage` | Stage | 升级阶段 | R | unsignedInt[1:5] |
| 2 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.SwUpgrade.Status` | Status | 状态 | R | unsignedInt[1:3] |
| 3 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.SwUpgrade.FailureCause` | FailureCause | 升级失败原因 | R | string(512) |

#### 列出 射频远端单元信息 （`LST RU`，16 个 PATH）

| # | TR-181 标准路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|----------------|--------|--------|------|------|
| 1 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.UserLabel` | UserLabel | 用户友好名 | RW | string |
| 2 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.VendorUnitFamilyType` | VendorUnitFamilyType | 归属类型 | R | string |
| 3 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.VendorUnitTypeNumber` | VendorUnitTypeNumber | 资产单元类型版本号 | R | string |
| 4 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RouteIndex` | RouteIndex | 射频单元路由指示 | R | string(64) |
| 5 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.Status` | Status | 状态 | R | unsignedInt[1:3] |
| 6 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.ManufacturerOUI` | ManufacturerOUI | 制造商OUI | R | string(6) |
| 7 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.Manufacturer` | Manufacturer | 制造商 | R | string(64) |
| 8 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.ModelName` | ModelName | 设备型号 | R | string(64) |
| 9 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.SerialNumber` | SerialNumber | 序列号 | R | string(64) |
| 10 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.HardwareVersion` | HardwareVersion | 硬件版本 | R | string(64) |
| 11 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.SoftwareVersion` | SoftwareVersion | 软件版本 | R | string(64) |
| 12 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.ProvisioningCode` | ProvisioningCode | 供应商代号 | R | string(64) |
| 13 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.Reboot` | Reboot | 重启开关 | RW | boolean |
| 14 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.FrequencyBand` | FrequencyBand | 支持频段 | RW | string |
| 15 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RFTxStatus` | RFTxStatus | 射频状态 | RW | boolean |
| 16 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.DateOfManufacture` | DateOfManufacture | 生产日期 | R | dateTime |

#### 列出 射频通道信息 （`LST RF_CHANNEL`，2 个 PATH）

| # | TR-181 标准路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|----------------|--------|--------|------|------|
| 1 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RFChannel.{i}.TxGain` | TxGain | 下行发射功率增益 | RW | string |
| 2 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RFChannel.{i}.NoisePwdBm` | NoisePwdBm | 底噪 | R | string |

#### 列出 扩展单元信息 （`LST EU`，13 个 PATH）

| # | TR-181 标准路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|----------------|--------|--------|------|------|
| 1 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.UserLabel` | UserLabel | 用户友好名 | RW | string |
| 2 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RouteIndex` | RouteIndex | 扩展单元路由指示 | R | string(64) |
| 3 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.ManufacturerOUI` | ManufacturerOUI | 制造商OUI | R | string(6) |
| 4 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.Manufacturer` | Manufacturer | 制造商 | R | string(64) |
| 5 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.ModelName` | ModelName | 设备型号 | R | string(64) |
| 6 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.SerialNumber` | SerialNumber | 序列号 | R | string(64) |
| 7 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.HardwareVersion` | HardwareVersion | 硬件版本 | R | string(64) |
| 8 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.SoftwareVersion` | SoftwareVersion | 软件版本 | R | string(64) |
| 9 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.ProvisioningCode` | ProvisioningCode | 供应商代号 | R | string(64) |
| 10 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.Status` | Status | 状态 | R | unsignedInt[1:3] |
| 11 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.DLCRCSum` | DLCRCSum | 扩展单元下行误码率 | R | string(64) |
| 12 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.ULCRCSum` | ULCRCSum | 扩展单元上行误码率 | R | string(64) |
| 13 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.Reboot` | Reboot | 重启开关 | RW | boolean |

#### 列出 扩展单元软件升级 （`LST EU_SW_UPGRADE`，3 个 PATH）

| # | TR-181 标准路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|----------------|--------|--------|------|------|
| 1 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.SwUpgrade.Stage` | Stage | 升级阶段 | R | unsignedInt[1:5] |
| 2 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.SwUpgrade.Status` | Status | 状态 | R | unsignedInt[1:3] |
| 3 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.SwUpgrade.FailureCause` | FailureCause | 升级失败原因 | R | string(512) |

#### 列出 槽位板卡信息 （`LST SLOT`，16 个 PATH）

| # | TR-181 标准路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|----------------|--------|--------|------|------|
| 1 | `Device.DeviceInfo.MU.{i}.Slot.{i}.PackPosition` | PackPosition | 板卡位置 | R | string(64) |
| 2 | `Device.DeviceInfo.MU.{i}.Slot.{i}.SlotsOccupied` | SlotsOccupied | 占用槽位 | R | string(64) |
| 3 | `Device.DeviceInfo.MU.{i}.Slot.{i}.ManufacturerOUI` | ManufacturerOUI | 制造商OUI | R | string(6) |
| 4 | `Device.DeviceInfo.MU.{i}.Slot.{i}.Manufacturer` | Manufacturer | 制造商 | R | string(64) |
| 5 | `Device.DeviceInfo.MU.{i}.Slot.{i}.ModelName` | ModelName | 设备型号 | R | string(64) |
| 6 | `Device.DeviceInfo.MU.{i}.Slot.{i}.SerialNumber` | SerialNumber | 序列号 | R | string(64) |
| 7 | `Device.DeviceInfo.MU.{i}.Slot.{i}.HardwareVersion` | HardwareVersion | 硬件版本 | R | string(64) |
| 8 | `Device.DeviceInfo.MU.{i}.Slot.{i}.SoftwareVersion` | SoftwareVersion | 软件版本 | R | string(64) |
| 9 | `Device.DeviceInfo.MU.{i}.Slot.{i}.ProvisioningCode` | ProvisioningCode | 供应商代号 | R | string(64) |
| 10 | `Device.DeviceInfo.MU.{i}.Slot.{i}.VendorUnitFamilyType` | VendorUnitFamilyType | 板卡类型 | R | string(64) |
| 11 | `Device.DeviceInfo.MU.{i}.Slot.{i}.UpTime` | UpTime | 运行时间 | R | unsignedInt |
| 12 | `Device.DeviceInfo.MU.{i}.Slot.{i}.DataModelSpecVersion` | DataModelSpecVersion | 数据模型版本 | R | string |
| 13 | `Device.DeviceInfo.MU.{i}.Slot.{i}.3GPPSpecVersion` | 3GPPSpecVersion | 3GPP协议版本 | R | string |
| 14 | `Device.DeviceInfo.MU.{i}.Slot.{i}.FirstUseDate` | FirstUseDate | 首次使用日期 | R | dateTime |
| 15 | `Device.DeviceInfo.MU.{i}.Slot.{i}.Status` | Status | 板卡状态 | R | unsignedInt[1:3] |
| 16 | `Device.DeviceInfo.MU.{i}.Slot.{i}.Reboot` | Reboot | 重启开关 | RW | boolean |

#### 修改 射频远端单元信息 （`MOD RU`，4 个 PATH）

| # | TR-181 标准路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|----------------|--------|--------|------|------|
| 1 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.UserLabel` | UserLabel | 用户友好名 | RW | string |
| 2 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.Reboot` | Reboot | 重启开关 | RW | boolean |
| 3 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.FrequencyBand` | FrequencyBand | 支持频段 | RW | string |
| 4 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RFTxStatus` | RFTxStatus | 射频状态 | RW | boolean |

#### 修改 射频通道信息 （`MOD RF_CHANNEL`，1 个 PATH）

| # | TR-181 标准路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|----------------|--------|--------|------|------|
| 1 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RFChannel.{i}.TxGain` | TxGain | 下行发射功率增益 | RW | string |

#### 修改 扩展单元信息 （`MOD EU`，2 个 PATH）

| # | TR-181 标准路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|----------------|--------|--------|------|------|
| 1 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.UserLabel` | UserLabel | 用户友好名 | RW | string |
| 2 | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.Reboot` | Reboot | 重启开关 | RW | boolean |

#### 修改 槽位板卡信息 （`MOD SLOT`，1 个 PATH）

| # | TR-181 标准路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|----------------|--------|--------|------|------|
| 1 | `Device.DeviceInfo.MU.{i}.Slot.{i}.Reboot` | Reboot | 重启开关 | RW | boolean |

### 2.2 分组：邻区参数管理

#### 列出 NR 异系统邻区 （`LST INTER_RAT_CELL_NR`，12 个 PATH）

| # | TR-181 标准路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|----------------|--------|--------|------|------|
| 1 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.PLMNID` | PLMNID | 邻区PLMN ID | RW | string(6) |
| 2 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.CID` | CID | 邻区CID | RW | unsignedLong[1:68719476735] |
| 3 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.GnbIdLen` | GnbIdLen | 邻区GnbId长度 | RW | unsignedInt[22:32] |
| 4 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.SsbFrequency` | SsbFrequency | SSB频点号 | RW | unsignedInt[0:3279165] |
| 5 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.SsbPeriodicity` | SsbPeriodicity_r15 | SSB周期 | RW | unsignedInt[5,10,20,40,80,160] |
| 6 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.SsbOffset` | SsbOffset_r15 | SSB偏移 | RW | unsignedInt[0..159] |
| 7 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.Ssb_Duration` | Ssb_Duration_r15 | ssb_Duration_r15 | RW | unsignedInt[1,2,3,4,5] |
| 8 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.PhyCellID` | PhyCellID | 邻区物理小区ID | RW | unsignedInt[0:1007] |
| 9 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.TAC` | TAC | 邻区TAC | RW | unsignedInt[0:16777215] |
| 10 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.Qoffset` | QOffset | 小区特定偏移量 | RW | int[-24:-8, -6:6, 8:24] |
| 11 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.NRband` | NRband_r15 | Nr频段 | RW | unsignedInt[1:1024] |
| 12 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.NeighType` | NeighType | 邻区类型 | RW | unsignedInt[0，1，2] |

#### 修改 NR 异系统邻区 （`MOD INTER_RAT_CELL_NR`，12 个 PATH）

| # | TR-181 标准路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|----------------|--------|--------|------|------|
| 1 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.PLMNID` | PLMNID | 邻区PLMN ID | RW | string(6) |
| 2 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.CID` | CID | 邻区CID | RW | unsignedLong[1:68719476735] |
| 3 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.GnbIdLen` | GnbIdLen | 邻区GnbId长度 | RW | unsignedInt[22:32] |
| 4 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.SsbFrequency` | SsbFrequency | SSB频点号 | RW | unsignedInt[0:3279165] |
| 5 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.SsbPeriodicity` | SsbPeriodicity_r15 | SSB周期 | RW | unsignedInt[5,10,20,40,80,160] |
| 6 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.SsbOffset` | SsbOffset_r15 | SSB偏移 | RW | unsignedInt[0..159] |
| 7 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.Ssb_Duration` | Ssb_Duration_r15 | ssb_Duration_r15 | RW | unsignedInt[1,2,3,4,5] |
| 8 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.PhyCellID` | PhyCellID | 邻区物理小区ID | RW | unsignedInt[0:1007] |
| 9 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.TAC` | TAC | 邻区TAC | RW | unsignedInt[0:16777215] |
| 10 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.Qoffset` | QOffset | 小区特定偏移量 | RW | int[-24:-8, -6:6, 8:24] |
| 11 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.NRband` | NRband_r15 | Nr频段 | RW | unsignedInt[1:1024] |
| 12 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.NeighType` | NeighType | 邻区类型 | RW | unsignedInt[0，1，2] |

---

## 3. 补全建议

让这 12 条命令在 console-v2 出现参数 PATH，需把上表 TR-181 路径补进字典并重建 sub_field：

1. **补字典**：在对应 paramModel 的 XML（`data/param-mappings/*.xml`）中增加上述 standardPath 条目（参数名/中文名/access/类型/范围按本文规范列填写），dictload 写入 `standard_params` + `param_mappings`。
2. **重建 sub_field**：字典就位后，命令的 `target_paths` 即可 JOIN 到 `standard_params`，sub_field 生成环节自动建立映射（或走 admin Catalog 的 XML 导入端点）。
3. **校验**：`mml/console-v2` 选这 12 条命令应出现对应参数 PATH；E2E 加断言「每条 LST/MOD 命令 sub_field ≥ 1」。

> 注：NR 异系统邻区 ADD/RMV（`ADD/RMV INTER_RAT_CELL_NR`）走目标对象路径，不在本文 LST/MOD 范围内；其父对象 `…InterRATCell.NR.` 亦需在字典中存在方可被设备 supported set 命中。
