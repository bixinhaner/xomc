# BaiBNQ trpath 删除 Review

## 对比口径

- XML 文件：`/Users/wangyong/OBJECT/Codex/xomc/omcgo/data/param-mappings/BaiBNQ.xml`
- CSV 文件：`/Users/wangyong/OBJECT/Codex/xomc/omcgo/data/param-mappings/references/NR全量参数集.csv`
- CSV 字段：`trpath.name`
- 对比时将路径中的数字实例段归一为 `{i}`，例如 `.1.` 按 `.{i}.` 处理，避免实例表示差异导致误删。
- XML 独有但仍有子路径的对象/表节点视为有效参数，保留不删除。
- `RRCTimers` 已按 `NR全量参数集.csv` 中记录更新类型、范围和默认值；路径中的 `FAPService.1.` 按通用多实例泛化为 `FAPService.{i}.`。

## 删除汇总

- 删除参数数量：18
- 删除范围：仅删除 `<parameters>` 下 XML 独有且没有子路径的叶子 `<param>` 节点。
- 保留对象/表节点数量：39
- RRCTimers 更新数量：11
- `totalEntries`：已从 `841` 调整为 `826`。

## 删除参数清单

| 序号 | 原 XML 行号 | trpath |
| ---: | ---: | --- |
| 1 | 86 | `Device.FAP.License.LicenseItem.{i}.DelayAvailable` |
| 2 | 88 | `Device.FAP.License.LicenseItem.{i}.Desciption` |
| 3 | 106 | `Device.FaultMgmt.CurrentAlarm.{i}.OUI` |
| 4 | 109 | `Device.FaultMgmt.CurrentAlarm.{i}.SerialNumber` |
| 5 | 119 | `Device.FaultMgmt.ExpeditedEvent.{i}.OUI` |
| 6 | 122 | `Device.FaultMgmt.ExpeditedEvent.{i}.SerialNumber` |
| 7 | 132 | `Device.FaultMgmt.HistoryEvent.{i}.OUI` |
| 8 | 135 | `Device.FaultMgmt.HistoryEvent.{i}.SerialNumber` |
| 9 | 146 | `Device.FaultMgmt.QueuedEvent.{i}.OUI` |
| 10 | 149 | `Device.FaultMgmt.QueuedEvent.{i}.SerialNumber` |
| 11 | 245 | `Device.Services.FAPService.{i}.FAPControl.NR.X_COM_HSS.RequestX2Related` |
| 12 | 249 | `Device.Services.FAPService.{i}.FAPControl.NR.X_COM_HSS.X2RelatedParameters` |
| 13 | 353 | `Device.FAP.NguIpBind1.BindInterface` |
| 14 | 810 | `Device.FAP.Synchronization.AntennaStatus` |
| 15 | 811 | `Device.FAP.Synchronization.Longitude` |
| 16 | 812 | `Device.FAP.Synchronization.Latitude` |
| 17 | 813 | `Device.FAP.Synchronization.Altitude` |
| 18 | 814 | `Device.FAP.Synchronization.NumberOfSatellites` |

## RRCTimers 更新清单

| 序号 | XML trpath | CSV trpath | type | min | max | defaultValue |
| ---: | --- | --- | --- | ---: | ---: | ---: |
| 1 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RRCTimers.T300` | `Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.RRCTimers.T300` | `U_INT` | 0 | 7 | 1 |
| 2 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RRCTimers.T301` | `Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.RRCTimers.T301` | `U_INT` | 0 | 7 | 2 |
| 3 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RRCTimers.T302` | `Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.RRCTimers.T302` | `U_INT` | 1 | 16 | 1 |
| 4 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RRCTimers.T304` | `Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.RRCTimers.T304` | `U_INT` | 0 | 7 | 6 |
| 5 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RRCTimers.T310` | `Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.RRCTimers.T310` | `U_INT` | 0 | 6 | 6 |
| 6 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RRCTimers.T311` | `Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.RRCTimers.T311` | `U_INT` | 0 | 6 | 1 |
| 7 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RRCTimers.T320` | `Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.RRCTimers.T320` | `U_INT` | 0 | 6 | 5 |
| 8 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RRCTimers.N310` | `Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.RRCTimers.N310` | `U_INT` | 0 | 6 | 7 |
| 9 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RRCTimers.N311` | `Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.RRCTimers.N311` | `U_INT` | 0 | 6 | 1 |
| 10 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RRCTimers.T319` | `Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.RRCTimers.T319` | `U_INT` | 0 | 7 | 7 |
| 11 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RRCTimers.T380` | `Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.RRCTimers.T380` | `U_INT` | 5 | 7 | 0 |

## 保留的对象/表节点

| 序号 | 原 XML 行号 | trpath |
| ---: | ---: | --- |
| 1 | 85 | `Device.FAP.License.LicenseItem.{i}` |
| 2 | 262 | `Device.Ethernet.Interface.{i}` |
| 3 | 263 | `Device.Ethernet.Interface.{i}.IPv4Address.{i}` |
| 4 | 269 | `Device.Ethernet.Interface.{i}.IPv6Address.{i}` |
| 5 | 275 | `Device.Ethernet.Interface.{i}.PppoeAddress.{i}` |
| 6 | 284 | `Device.Ethernet.Interface.{i}.VlanInterface.{i}` |
| 7 | 285 | `Device.Ethernet.Interface.{i}.VlanInterface.{i}.IPv4Address.{i}` |
| 8 | 291 | `Device.Ethernet.Interface.{i}.VlanInterface.{i}.IPv6Address.{i}` |
| 9 | 299 | `Device.Ethernet.Interface.{i}.VlanInterface.{i}.VlanPppoeAddress.{i}` |
| 10 | 308 | `Device.Ethernet.IpRoute.{i}` |
| 11 | 321 | `Device.FAP.Ipsec.{i}` |
| 12 | 397 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.CN.TA.{i}` |
| 13 | 399 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.CN.TA.{i}.PLMNList.{i}` |
| 14 | 402 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.CN.TA.{i}.PLMNList.{i}.SliceList.{i}` |
| 15 | 415 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.ConnMode.EUTRA.Carrier.{i}` |
| 16 | 420 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}` |
| 17 | 435 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}` |
| 18 | 454 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.ConnMode.NR.A1MeasureCtrl.{i}` |
| 19 | 473 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.ConnMode.NR.A2MeasureCtrl.{i}` |
| 20 | 492 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.ConnMode.NR.A3MeasureCtrl.{i}` |
| 21 | 512 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.ConnMode.NR.A4MeasureCtrl.{i}` |
| 22 | 532 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.ConnMode.NR.A5MeasureCtrl.{i}` |
| 23 | 556 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.ConnMode.NR.InterFreq.Carrier.{i}` |
| 24 | 580 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.ConnMode.NR.PeriodMeasCtrl.{i}` |
| 25 | 587 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.IdleMode.EUTRA.Carrier.{i}` |
| 26 | 601 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}` |
| 27 | 634 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.NeighborList.LTECell.{i}` |
| 28 | 641 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.NeighborList.NRCell.{i}` |
| 29 | 658 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.PHY.BWP.BWPDL.{i}` |
| 30 | 672 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.PHY.BWP.BWPDL.{i}.PDCCH.CommonSearchSpaceList.{i}` |
| 31 | 686 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.PHY.BWP.BWPDL.{i}.PDCCH.DedicatedCtlResourceSet.{i}` |
| 32 | 701 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.PHY.BWP.BWPDL.{i}.PDSCH.PDSCHDedicatedTimeDomainResourceAllocationList.{i}` |
| 33 | 703 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.PHY.BWP.BWPDL.{i}.PDSCH.PDSCHTimeDomainResourceAllocationList.{i}` |
| 34 | 731 | `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.QOS.{i}` |
| 35 | 783 | `Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}` |
| 36 | 787 | `Device.Services.FAPService.{i}.FAPControl.NR.DscpList.{i}` |
| 37 | 846 | `Device.Services.FAPService.{i}.FAPControl.NR.XnBlackList.{i}` |
| 38 | 848 | `Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}` |
| 39 | 856 | `Device.Services.FAPService.{i}.FAPControl.Qos.QosSstInfo.{i}` |
