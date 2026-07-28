# MLN trpath 删除 Review

## 对比口径

- XML 文件：`/Users/wangyong/OBJECT/Codex/xomc/omcgo/data/param-mappings/MLN.xml`
- CSV 文件：`/Users/wangyong/OBJECT/Codex/xomc/omcgo/data/param-mappings/references/LTE全量参数集.csv`
- CSV 字段：`trpath.name`
- 对比时将路径中的数字实例段归一为 `{i}`，例如 `.1.` 按 `.{i}.` 处理。
- XML 独有但仍有子路径的对象/表节点视为有效参数，保留不删除。
- `RRCTimers` 按 `LTE全量参数集.csv` 中记录更新 `access`、`type`、`min`、`max`、`defaultValue`。

## 删除汇总

- 删除参数数量：39
- 删除范围：仅删除 `<parameters>` 下 XML 独有且没有子路径的叶子 `<param>` 节点。
- 保留对象/表节点数量：10
- RRCTimers 更新数量：10
- `totalEntries`：已从 `566` 调整为 `527`。

## 删除参数清单

| 序号 | 原 XML 行号 | trpath |
| ---: | ---: | --- |
| 1 | 71 | `Device.DeviceInfo.FAPService.UeAccess.Enable` |
| 2 | 72 | `Device.DeviceInfo.ForceRadioEnable` |
| 3 | 130 | `Device.DeviceInfo.X_COM_LTE_LBO_TRANSFER_Mode` |
| 4 | 216 | `Device.ManagementServer.STUNMaximumKeepAlivePeriod` |
| 5 | 316 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_HSS.EMBEDDED_EPCEnableState` |
| 6 | 317 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_HSS.EMBEDDED_EPCMode` |
| 7 | 319 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_HSS.IMSInfo` |
| 8 | 322 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_HSS.ims` |
| 9 | 325 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.Capacity.{i}.DelayAvaliable` |
| 10 | 334 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.EMBEDDED_EPCCentralizedModeAvailable` |
| 11 | 335 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.EMBEDDED_EPCSingleModeAvailable` |
| 12 | 343 | `Device.Services.FAPService.{i}.X_COM.LTE.UE.X_MMMM_UE_SPEED_STATISTICS` |
| 13 | 385 | `Device.DeviceInfo.SiteId` |
| 14 | 427 | `Device.DeviceInfo.X_COM_LTE_LBO_First_Static_Ip_Address` |
| 15 | 428 | `Device.DeviceInfo.X_COM_LTE_LBO_IMSI_IP_List` |
| 16 | 429 | `Device.DeviceInfo.X_COM_LTE_LBO_Last_Static_Ip_Address` |
| 17 | 430 | `Device.DeviceInfo.X_COM_LTE_LBO_NET_Mask` |
| 18 | 431 | `Device.DeviceInfo.X_COM_LTE_LBO_START_UE_Addr` |
| 19 | 432 | `Device.DeviceInfo.X_COM_LTE_LBO_Static_Ip_Addr_Switch` |
| 20 | 433 | `Device.DeviceInfo.X_COM_LTE_LBO_Switch` |
| 21 | 538 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOQos.{i}.ArpPci` |
| 22 | 539 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOQos.{i}.ArpPl` |
| 23 | 540 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOQos.{i}.ArpPvi` |
| 24 | 541 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOQos.{i}.GbrDl` |
| 25 | 542 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOQos.{i}.GbrUl` |
| 26 | 543 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOQos.{i}.MbrDl` |
| 27 | 544 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOQos.{i}.MbrUl` |
| 28 | 545 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOQos.{i}.PccId` |
| 29 | 546 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOQos.{i}.PccName` |
| 30 | 547 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOQos.{i}.PfList` |
| 31 | 548 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOQos.{i}.Precedence` |
| 32 | 549 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOQos.{i}.Qci` |
| 33 | 551 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOTft.{i}.AppName` |
| 34 | 552 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOTft.{i}.IpMask` |
| 35 | 553 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOTft.{i}.IpPort` |
| 36 | 554 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOTft.{i}.PfId` |
| 37 | 555 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOTft.{i}.ProtocolId` |
| 38 | 537 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOQos.{i}` |
| 39 | 550 | `Device.Services.FAPService.EMBEDDED_EPCBearerLBOTft.{i}` |

## 保留对象/表节点清单

| 序号 | 原 XML 行号 | trpath | 保留原因 |
| ---: | ---: | --- | --- |
| 1 | 324 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.Capacity.{i}` | 存在子路径 |
| 2 | 435 | `Device.FAP.Ipsec.{i}` | 存在子路径 |
| 3 | 438 | `Device.Services.FAPService.1.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}` | 存在子路径 |
| 4 | 448 | `Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.{i}` | 存在子路径 |
| 5 | 473 | `Device.Services.FAPService.2.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}` | 存在子路径 |
| 6 | 483 | `Device.Services.FAPService.2.CellConfig.LTE.RAN.NeighborList.LTECell.{i}` | 存在子路径 |
| 7 | 506 | `Device.Services.FAPService.3.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}` | 存在子路径 |
| 8 | 516 | `Device.Services.FAPService.3.CellConfig.LTE.RAN.NeighborList.LTECell.{i}` | 存在子路径 |
| 9 | 562 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}` | 存在子路径 |
| 10 | 565 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}` | 存在子路径 |

## RRCTimers 更新清单

| 序号 | 原 XML 行号 | CSV trpath | XML trpath | type | min | max | defaultValue |
| ---: | ---: | --- | --- | --- | ---: | ---: | ---: |
| 1 | 296 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300` | `U_INT` | 2000 | 2000 | 600 |
| 2 | 297 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301` | `U_INT` | 2000 | 2000 | 600 |
| 3 | 298 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302` | `U_INT` | 2000 | 2000 | 300 |
| 4 | 299 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA` | `U_INT` | 2000 | 2000 | 500 |
| 5 | 300 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT` | `U_INT` | 8000 | 8000 | 1000 |
| 6 | 301 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310` | `U_INT` | 2000 | 2000 | 1000 |
| 7 | 302 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311` | `U_INT` | 30000 | 30000 | 3000 |
| 8 | 303 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320` | `U_INT` | 180 | 180 | 5 |
| 9 | 294 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310` | `U_INT` | 20 | 20 | 3 |
| 10 | 295 | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311` | `U_INT` | 10 | 10 | 1 |

## 校验结果

- XML 可正常解析。
- `RRCTimers` 在 XML 中按归一化规则全部能匹配到 CSV。
- XML 剩余不匹配项仅为有子路径的对象/表节点。
- XML 独有叶子参数数量为 `0`。
