# BLQ trpath 删除 Review

## 对比口径

- XML 文件：`/Users/wangyong/OBJECT/Codex/xomc/omcgo/data/param-mappings/BLQ.xml`
- CSV 文件：`/Users/wangyong/OBJECT/Codex/xomc/omcgo/data/param-mappings/references/LTE全量参数集.csv`
- CSV 字段：`trpath.name`
- 对比时将路径中的数字实例段归一为 `{i}`，例如 `.1.` 按 `.{i}.` 处理。
- XML 独有但仍有子路径的对象/表节点视为有效参数，保留不删除。
- `RRCTimers` 按 `LTE全量参数集.csv` 中记录更新 `access`、`type`、`min`、`max`、`defaultValue`。

## 删除汇总

- 删除参数数量：31
- 删除范围：仅删除 `<parameters>` 下 XML 独有且没有子路径的叶子 `<param>` 节点。
- 保留对象/表节点数量：8
- RRCTimers 更新数量：10
- `totalEntries`：已从 `767` 调整为 `736`。

## 删除参数清单

| 序号 | 原 XML 行号 | trpath |
| ---: | ---: | --- |
| 1 | 20 | `Device.DeviceInfo.AdditionalHardwareVersion` |
| 2 | 21 | `Device.DeviceInfo.AdditionalSoftwareVersion` |
| 3 | 22 | `Device.DeviceInfo.DataModelSpecVersion` |
| 4 | 25 | `Device.DeviceInfo.HardwarePlatform` |
| 5 | 29 | `Device.DeviceInfo.UserLabel` |
| 6 | 48 | `Device.DeviceInfo.FAPService.UeAccess.Enable` |
| 7 | 50 | `Device.DeviceInfo.ForceRadioEnable` |
| 8 | 116 | `Device.DeviceInfo.X_COM_SCTP_CONFIG_MTU` |
| 9 | 193 | `Device.ManagementServer.STUNMaximumKeepAlivePeriod` |
| 10 | 308 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_HSS.IMSInfo` |
| 11 | 310 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_HSS.ims` |
| 12 | 313 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.Capacity.{i}.DelayAvaliable` |
| 13 | 345 | `Device.DeviceInfo.NET_IFCONFIG_INTERFACE` |
| 14 | 443 | `Device.DeviceInfo.SlaveInterface` |
| 15 | 444 | `Device.DeviceInfo.SlaveInterfaceIP` |
| 16 | 595 | `Device.RemoteDeviceList.{i}.AdminState` |
| 17 | 596 | `Device.RemoteDeviceList.{i}.ConnStatus` |
| 18 | 597 | `Device.RemoteDeviceList.{i}.HardwareVersion` |
| 19 | 598 | `Device.RemoteDeviceList.{i}.IpAddress` |
| 20 | 599 | `Device.RemoteDeviceList.{i}.LastLogin` |
| 21 | 600 | `Device.RemoteDeviceList.{i}.LastLogout` |
| 22 | 601 | `Device.RemoteDeviceList.{i}.Latitude` |
| 23 | 602 | `Device.RemoteDeviceList.{i}.Longitude` |
| 24 | 603 | `Device.RemoteDeviceList.{i}.PhyState` |
| 25 | 604 | `Device.RemoteDeviceList.{i}.SerialNumber` |
| 26 | 605 | `Device.RemoteDeviceList.{i}.SoftwareVersion` |
| 27 | 606 | `Device.RemoteDeviceList.{i}.SyncState` |
| 28 | 607 | `Device.RemoteDeviceList.{i}.TfcsSyncMgrState` |
| 29 | 758 | `Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.MultiS1ConnectionMode` |
| 30 | 765 | `Device.Services.FAPService.{i}.X_COM.LTE.GrpPwrCtrl.PhyIctaadjsamp` |
| 31 | 766 | `Device.Services.FAPService.{i}.X_COM.LTE.GrpPwrCtrl.PhyPpssncgpsadjsamp` |

## 保留对象/表节点清单

| 序号 | 原 XML 行号 | trpath | 保留原因 |
| ---: | ---: | --- | --- |
| 1 | 312 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.Capacity.{i}` | 存在子路径 |
| 2 | 583 | `Device.FAP.Ipsec.{i}` | 存在子路径 |
| 3 | 585 | `Device.FAP.NL.{i}` | 存在子路径 |
| 4 | 609 | `Device.Services.FAPService.1.CellConfig.LTE.MultiIpsecConfigParam.{i}` | 存在子路径 |
| 5 | 678 | `Device.Services.FAPService.2.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}` | 存在子路径 |
| 6 | 688 | `Device.Services.FAPService.2.CellConfig.LTE.RAN.NeighborList.LTECell.{i}` | 存在子路径 |
| 7 | 713 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}` | 存在子路径 |
| 8 | 723 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}` | 存在子路径 |

## RRCTimers 更新清单

| 序号 | 原 XML 行号 | CSV trpath | XML trpath | type | min | max | defaultValue |
| ---: | ---: | --- | --- | --- | ---: | ---: | ---: |
| 1 | 275 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300` | `U_INT` | 2000 | 2000 | 600 |
| 2 | 276 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301` | `U_INT` | 2000 | 2000 | 600 |
| 3 | 277 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302` | `U_INT` | 2000 | 2000 | 300 |
| 4 | 278 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA` | `U_INT` | 2000 | 2000 | 500 |
| 5 | 279 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT` | `U_INT` | 8000 | 8000 | 1000 |
| 6 | 280 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310` | `U_INT` | 2000 | 2000 | 1000 |
| 7 | 281 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311` | `U_INT` | 30000 | 30000 | 3000 |
| 8 | 282 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320` | `U_INT` | 180 | 180 | 5 |
| 9 | 273 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310` | `U_INT` | 20 | 20 | 3 |
| 10 | 274 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311` | `U_INT` | 10 | 10 | 1 |

## 校验结果

- XML 可正常解析。
- `RRCTimers` 在 XML 中按归一化规则全部能匹配到 CSV。
- XML 剩余不匹配项仅为有子路径的对象/表节点。
- XML 独有叶子参数数量为 `0`。
