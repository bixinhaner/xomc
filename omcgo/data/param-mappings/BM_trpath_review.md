# BM trpath 删除 Review

## 对比口径

- XML 文件：`/Users/wangyong/OBJECT/Codex/xomc/omcgo/data/param-mappings/BM.xml`
- CSV 文件：`/Users/wangyong/OBJECT/Codex/xomc/omcgo/data/param-mappings/references/BM全量参数集.csv`
- CSV 字段：`trpath.name`
- 对比时将路径中的数字实例段归一为 `{i}`，例如 `.1.` 按 `.{i}.` 处理。
- XML 独有但仍有子路径的对象/表节点视为有效参数，保留不删除。
- `RRCTimers` 按 `BM全量参数集.csv` 中记录更新 `access`、`type`、`min`、`max`、`defaultValue`；CSV 未提供的 `changeApplies` 保留 XML 原值。

## 删除汇总

- 删除参数数量：13
- 删除范围：仅删除 `<parameters>` 下 XML 独有且没有子路径的叶子 `<param>` 节点。
- 保留对象/表节点数量：1
- RRCTimers 更新数量：10
- `totalEntries`：已从 `545` 调整为 `532`。

## 删除参数清单

| 序号 | 原 XML 行号 | trpath |
| ---: | ---: | --- |
| 1 | 71 | `Device.DeviceInfo.Enable256QAM` |
| 2 | 72 | `Device.DeviceInfo.FAPService.UeAccess.Enable` |
| 3 | 73 | `Device.DeviceInfo.ForceRadioEnable` |
| 4 | 142 | `Device.DeviceInfo.X_COM_LTE_LGW_TRANSFER_Mode` |
| 5 | 252 | `Device.ManagementServer.STUNMaximumKeepAlivePeriod` |
| 6 | 356 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_HSS.HaloBEnableState` |
| 7 | 357 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_HSS.HaloBMode` |
| 8 | 359 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_HSS.IMSInfo` |
| 9 | 362 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_HSS.ims` |
| 10 | 365 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.Capacity.{i}.DelayAvaliable` |
| 11 | 375 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.HaloBCentralizedModeAvailable` |
| 12 | 376 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.HaloBSingleModeAvailable` |
| 13 | 383 | `Device.Services.FAPService.{i}.X_COM.LTE.UE.X_BAICELLS_UE_SPEED_STATISTICS` |

## 保留对象/表节点清单

| 序号 | 原 XML 行号 | trpath | 保留原因 |
| ---: | ---: | --- | --- |
| 1 | 364 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.Capacity.{i}` | 存在子路径 |

## RRCTimers 更新清单

| 序号 | 原 XML 行号 | CSV trpath | XML trpath |
| ---: | ---: | --- | --- |
| 1 | 335 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310` |
| 2 | 336 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311` |
| 3 | 337 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300` |
| 4 | 338 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301` |
| 5 | 339 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302` |
| 6 | 340 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA` |
| 7 | 341 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT` |
| 8 | 342 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310` |
| 9 | 343 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311` |
| 10 | 344 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320` |

## 校验结果

- XML 可正常解析。
- `RRCTimers` 在 XML 中按归一化规则全部能匹配到 CSV。
- XML 剩余不匹配项仅为有子路径的对象/表节点。
- XML 独有叶子参数数量为 `0`。
