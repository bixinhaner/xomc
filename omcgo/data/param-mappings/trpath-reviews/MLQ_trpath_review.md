# MLQ trpath 删除 Review

## 对比口径

- XML 文件：`/Users/wangyong/OBJECT/Codex/xomc/omcgo/data/param-mappings/MLQ.xml`
- CSV 文件：`/Users/wangyong/OBJECT/Codex/xomc/omcgo/data/param-mappings/references/LTE全量参数集.csv`
- CSV 字段：`trpath.name`
- 对比时将路径中的数字实例段归一为 `{i}`，例如 `.1.` 按 `.{i}.` 处理。
- XML 独有但仍有子路径的对象/表节点视为有效参数，保留不删除。
- `RRCTimers` 按 `LTE全量参数集.csv` 中记录更新 `access`、`type`、`min`、`max`、`defaultValue`。

## 删除汇总

- 删除参数数量：53
- 删除范围：仅删除 `<parameters>` 下 XML 独有且没有子路径的叶子 `<param>` 节点。
- 保留对象/表节点数量：7
- RRCTimers 更新数量：10
- `totalEntries`：已从 `560` 调整为 `507`。

## 删除参数清单

| 序号 | 原 XML 行号 | trpath |
| ---: | ---: | --- |
| 1 | 33 | `Device.DeviceInfo.FAPService.UeAccess.Enable` |
| 2 | 34 | `Device.DeviceInfo.ForceRadioEnable` |
| 3 | 88 | `Device.DeviceInfo.X_COM_LTE_LBO_Switch` |
| 4 | 89 | `Device.DeviceInfo.X_COM_LTE_LBO_TRANSFER_Mode` |
| 5 | 175 | `Device.ManagementServer.STUNMaximumKeepAlivePeriod` |
| 6 | 250 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.{i}.apnDefault` |
| 7 | 251 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.{i}.apnName` |
| 8 | 252 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.{i}.apnType` |
| 9 | 253 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.{i}.apnVlanId` |
| 10 | 254 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.VxLan.{i}.apnName` |
| 11 | 255 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.VxLan.{i}.id` |
| 12 | 256 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.l2TunnelEnable` |
| 13 | 257 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.l2TunnelIp` |
| 14 | 258 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.l2TunnelMode` |
| 15 | 272 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_HSS.EMBEDDED_EPCEnableState` |
| 16 | 273 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_HSS.EMBEDDED_EPCMode` |
| 17 | 275 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_HSS.IMSInfo` |
| 18 | 277 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_HSS.ims` |
| 19 | 280 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.Capacity.{i}.DelayAvaliable` |
| 20 | 289 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.EMBEDDED_EPCCentralizedModeAvailable` |
| 21 | 290 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.EMBEDDED_EPCSingleModeAvailable` |
| 22 | 299 | `Device.Services.FAPService.{i}.X_COM.LTE.UE.X_MMMM_UE_SPEED_STATISTICS` |
| 23 | 424 | `Device.DeviceInfo.X_COM_LTE_LBO_First_Static_Ip_Address` |
| 24 | 425 | `Device.DeviceInfo.X_COM_LTE_LBO_IMSI_IP_List` |
| 25 | 426 | `Device.DeviceInfo.X_COM_LTE_LBO_Ifname` |
| 26 | 427 | `Device.DeviceInfo.X_COM_LTE_LBO_Last_Static_Ip_Address` |
| 27 | 428 | `Device.DeviceInfo.X_COM_LTE_LBO_NET_Mask` |
| 28 | 429 | `Device.DeviceInfo.X_COM_LTE_LBO_START_UE_Addr` |
| 29 | 430 | `Device.DeviceInfo.X_COM_LTE_LBO_Static_Ip_Addr_Switch` |
| 30 | 498 | `Device.Services.FAPService.{i}.CellConfig.LTE.MocnConfigParam.{i}.MMEStatus` |
| 31 | 532 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.1.apnDefault` |
| 32 | 533 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.1.apnName` |
| 33 | 534 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.1.apnType` |
| 34 | 535 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.1.apnVlanId` |
| 35 | 536 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.2.apnDefault` |
| 36 | 537 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.2.apnName` |
| 37 | 538 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.2.apnType` |
| 38 | 539 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.2.apnVlanId` |
| 39 | 540 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.3.apnDefault` |
| 40 | 541 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.3.apnName` |
| 41 | 542 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.3.apnType` |
| 42 | 543 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.3.apnVlanId` |
| 43 | 544 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.4.apnDefault` |
| 44 | 545 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.4.apnName` |
| 45 | 546 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.4.apnType` |
| 46 | 547 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.Apn.4.apnVlanId` |
| 47 | 548 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.VxLan.1.id` |
| 48 | 549 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.VxLan.2.id` |
| 49 | 550 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.VxLan.3.id` |
| 50 | 551 | `Device.Services.FAPService.{i}.FAPControl.EMBEDDED_EPC.L2.VxLan.4.id` |
| 51 | 552 | `Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.MultiS1ConnectionMode` |
| 52 | 559 | `Device.Services.FAPService.{i}.X_COM.LTE.GrpPwrCtrl.PhyIctaadjsamp` |
| 53 | 560 | `Device.Services.FAPService.{i}.X_COM.LTE.GrpPwrCtrl.PhyPpssncgpsadjsamp` |

## 保留对象/表节点清单

| 序号 | 原 XML 行号 | trpath | 保留原因 |
| ---: | ---: | --- | --- |
| 1 | 279 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.Capacity.{i}` | 存在子路径 |
| 2 | 441 | `Device.FAP.Ipsec.{i}` | 存在子路径 |
| 3 | 443 | `Device.FAP.NL.{i}` | 存在子路径 |
| 4 | 453 | `Device.Services.FAPService.1.CellConfig.LTE.MultiIpsecConfigParam.{i}` | 存在子路径 |
| 5 | 496 | `Device.Services.FAPService.{i}.CellConfig.LTE.MocnConfigParam.{i}` | 存在子路径 |
| 6 | 512 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}` | 存在子路径 |
| 7 | 522 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}` | 存在子路径 |

## RRCTimers 更新清单

| 序号 | 原 XML 行号 | CSV trpath | XML trpath | type | min | max | defaultValue |
| ---: | ---: | --- | --- | --- | ---: | ---: | ---: |
| 1 | 242 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300` | `U_INT` | 2000 | 2000 | 600 |
| 2 | 243 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301` | `U_INT` | 2000 | 2000 | 600 |
| 3 | 244 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302` | `U_INT` | 2000 | 2000 | 300 |
| 4 | 245 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA` | `U_INT` | 2000 | 2000 | 500 |
| 5 | 246 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT` | `U_INT` | 8000 | 8000 | 1000 |
| 6 | 247 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310` | `U_INT` | 2000 | 2000 | 1000 |
| 7 | 248 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311` | `U_INT` | 30000 | 30000 | 3000 |
| 8 | 249 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320` | `U_INT` | 180 | 180 | 5 |
| 9 | 240 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310` | `U_INT` | 20 | 20 | 3 |
| 10 | 241 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311` | `U_INT` | 10 | 10 | 1 |

## 校验结果

- XML 可正常解析。
- `RRCTimers` 在 XML 中按归一化规则全部能匹配到 CSV。
- XML 剩余不匹配项仅为有子路径的对象/表节点。
- XML 独有叶子参数数量为 `0`。
