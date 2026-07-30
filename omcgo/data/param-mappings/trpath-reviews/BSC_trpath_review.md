# BSC trpath 删除 Review

## 对比口径

- XML 文件：`/Users/wangyong/OBJECT/Codex/xomc/omcgo/data/param-mappings/BSC.xml`
- CSV 文件：`/Users/wangyong/OBJECT/Codex/xomc/omcgo/data/param-mappings/references/GSM全量参数集.csv`
- CSV 字段：`trpath.name`
- 对比时将路径中的数字实例段归一为 `{i}`，例如 `.1.` 按 `.{i}.` 处理。
- XML 独有但仍有子路径的对象/表节点视为有效参数，保留不删除。
- BSC 产品私有参数特例：`DeviceGSM.` 开头的参数全部保留，不按 GSM CSV 缺失删除。

## 删除汇总

- 删除参数数量：4
- 删除范围：仅删除 `<parameters>` 下 XML 独有、没有子路径、且不属于 `DeviceGSM.` 产品私有前缀的叶子 `<param>` 节点。
- 保留对象/表节点数量：1
- `DeviceGSM.` 产品私有参数保留数量：135
- `totalEntries`：已从 `232` 调整为 `228`。

## 删除参数清单

| 序号 | 原 XML 行号 | trpath |
| ---: | ---: | --- |
| 1 | 37 | `Device.DeviceInfo.GSM.RealtimeUserCount` |
| 2 | 46 | `Device.FaultMgmt.CurrentAlarm.{i}.AlarmChangedTime` |
| 3 | 102 | `Device.KeepalivedMgmt.LocalUplinkIpAddr` |
| 4 | 107 | `Device.KeepalivedMgmt.VrrpMgmt.Index` |

## 保留对象/表节点清单

| 序号 | 原 XML 行号 | trpath | 保留原因 |
| ---: | ---: | --- | --- |
| 1 | 145 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.Capacity.{i}` | 存在子路径 |

## DeviceGSM 产品私有参数保留清单

| 序号 | 原 XML 行号 | trpath | 保留原因 |
| ---: | ---: | --- | --- |
| 1 | 125 | `DeviceGSM.Bts.Index` | BSC 产品私有前缀 `DeviceGSM.` |
| 2 | 126 | `DeviceGSM.Bts.Trx.Index` | BSC 产品私有前缀 `DeviceGSM.` |
| 3 | 127 | `DeviceGSM.Bts.{i}.Band` | BSC 产品私有前缀 `DeviceGSM.` |
| 4 | 128 | `DeviceGSM.Bts.{i}.Bsic` | BSC 产品私有前缀 `DeviceGSM.` |
| 5 | 129 | `DeviceGSM.Bts.{i}.C0PowerRed` | BSC 产品私有前缀 `DeviceGSM.` |
| 6 | 130 | `DeviceGSM.Bts.{i}.CellId` | BSC 产品私有前缀 `DeviceGSM.` |
| 7 | 131 | `DeviceGSM.Bts.{i}.CellReselectHysteresis` | BSC 产品私有前缀 `DeviceGSM.` |
| 8 | 132 | `DeviceGSM.Bts.{i}.CellReselectOffset` | BSC 产品私有前缀 `DeviceGSM.` |
| 9 | 133 | `DeviceGSM.Bts.{i}.CellReselectPenaltyTime` | BSC 产品私有前缀 `DeviceGSM.` |
| 10 | 134 | `DeviceGSM.Bts.{i}.CodecSupport` | BSC 产品私有前缀 `DeviceGSM.` |
| 11 | 135 | `DeviceGSM.Bts.{i}.GprsMode` | BSC 产品私有前缀 `DeviceGSM.` |
| 12 | 136 | `DeviceGSM.Bts.{i}.HandoverAlgorithm` | BSC 产品私有前缀 `DeviceGSM.` |
| 13 | 137 | `DeviceGSM.Bts.{i}.IpaRslIp` | BSC 产品私有前缀 `DeviceGSM.` |
| 14 | 138 | `DeviceGSM.Bts.{i}.IpaUnitId` | BSC 产品私有前缀 `DeviceGSM.` |
| 15 | 139 | `DeviceGSM.Bts.{i}.LocationAreaCode` | BSC 产品私有前缀 `DeviceGSM.` |
| 16 | 140 | `DeviceGSM.Bts.{i}.MsMaxPower` | BSC 产品私有前缀 `DeviceGSM.` |
| 17 | 141 | `DeviceGSM.Bts.{i}.NeighborCgiAdd` | BSC 产品私有前缀 `DeviceGSM.` |
| 18 | 142 | `DeviceGSM.Bts.{i}.NeighborCgiDel` | BSC 产品私有前缀 `DeviceGSM.` |
| 19 | 143 | `DeviceGSM.Bts.{i}.NeighborListMode` | BSC 产品私有前缀 `DeviceGSM.` |
| 20 | 149 | `DeviceGSM.Bts.0.NumofTrxChannel` | BSC 产品私有前缀 `DeviceGSM.` |
| 21 | 150 | `DeviceGSM.Bts.0.OmlConnectState` | BSC 产品私有前缀 `DeviceGSM.` |
| 22 | 151 | `DeviceGSM.Bts.{i}.OmlIpaStreamId` | BSC 产品私有前缀 `DeviceGSM.` |
| 23 | 152 | `DeviceGSM.Bts.{i}.RachCellBarred` | BSC 产品私有前缀 `DeviceGSM.` |
| 24 | 153 | `DeviceGSM.Bts.{i}.Si2quaterNeighborListAdd` | BSC 产品私有前缀 `DeviceGSM.` |
| 25 | 154 | `DeviceGSM.Bts.{i}.Si2quaterNeighborListDel` | BSC 产品私有前缀 `DeviceGSM.` |
| 26 | 155 | `DeviceGSM.Bts.{i}.Trx.{i}.Arfcn` | BSC 产品私有前缀 `DeviceGSM.` |
| 27 | 156 | `DeviceGSM.Bts.{i}.Trx.{i}.MaxPowerRed` | BSC 产品私有前缀 `DeviceGSM.` |
| 28 | 157 | `DeviceGSM.Bts.{i}.Trx.{i}.RfLocked` | BSC 产品私有前缀 `DeviceGSM.` |
| 29 | 158 | `DeviceGSM.Bts.{i}.Trx.{i}.Ts.{i}.PhyChanConfig` | BSC 产品私有前缀 `DeviceGSM.` |
| 30 | 159 | `DeviceGSM.Bts.{i}.handover` | BSC 产品私有前缀 `DeviceGSM.` |
| 31 | 160 | `DeviceGSM.Bts.{i}.handover1.maximum.distance` | BSC 产品私有前缀 `DeviceGSM.` |
| 32 | 161 | `DeviceGSM.Bts.{i}.handover1.power.budget.hysteresis` | BSC 产品私有前缀 `DeviceGSM.` |
| 33 | 162 | `DeviceGSM.Bts.{i}.handover1.power.budget.interval` | BSC 产品私有前缀 `DeviceGSM.` |
| 34 | 163 | `DeviceGSM.Bts.{i}.handover1.window.rxlev.averaging` | BSC 产品私有前缀 `DeviceGSM.` |
| 35 | 164 | `DeviceGSM.Bts.{i}.handover1.window.rxlev.neighbor.averaging` | BSC 产品私有前缀 `DeviceGSM.` |
| 36 | 165 | `DeviceGSM.Bts.{i}.handover1.window.rxqual.averaging` | BSC 产品私有前缀 `DeviceGSM.` |
| 37 | 166 | `DeviceGSM.Bts.{i}.handover2.afs-bias.rxlev` | BSC 产品私有前缀 `DeviceGSM.` |
| 38 | 167 | `DeviceGSM.Bts.{i}.handover2.afs-bias.rxqual` | BSC 产品私有前缀 `DeviceGSM.` |
| 39 | 168 | `DeviceGSM.Bts.{i}.handover2.assignment` | BSC 产品私有前缀 `DeviceGSM.` |
| 40 | 169 | `DeviceGSM.Bts.{i}.handover2.max-handovers` | BSC 产品私有前缀 `DeviceGSM.` |
| 41 | 170 | `DeviceGSM.Bts.{i}.handover2.maximum.distance` | BSC 产品私有前缀 `DeviceGSM.` |
| 42 | 171 | `DeviceGSM.Bts.{i}.handover2.min-free-slots.tch-f` | BSC 产品私有前缀 `DeviceGSM.` |
| 43 | 172 | `DeviceGSM.Bts.{i}.handover2.min-free-slots.tch-h` | BSC 产品私有前缀 `DeviceGSM.` |
| 44 | 173 | `DeviceGSM.Bts.{i}.handover2.min.rxlev` | BSC 产品私有前缀 `DeviceGSM.` |
| 45 | 174 | `DeviceGSM.Bts.{i}.handover2.min.rxqual` | BSC 产品私有前缀 `DeviceGSM.` |
| 46 | 175 | `DeviceGSM.Bts.{i}.handover2.penalty-time.failed-assignment` | BSC 产品私有前缀 `DeviceGSM.` |
| 47 | 176 | `DeviceGSM.Bts.{i}.handover2.penalty-time.failed-ho` | BSC 产品私有前缀 `DeviceGSM.` |
| 48 | 177 | `DeviceGSM.Bts.{i}.handover2.penalty-time.low-rxqual-assignment` | BSC 产品私有前缀 `DeviceGSM.` |
| 49 | 178 | `DeviceGSM.Bts.{i}.handover2.penalty-time.low-rxqual-ho` | BSC 产品私有前缀 `DeviceGSM.` |
| 50 | 179 | `DeviceGSM.Bts.{i}.handover2.penalty-time.max-distance` | BSC 产品私有前缀 `DeviceGSM.` |
| 51 | 180 | `DeviceGSM.Bts.{i}.handover2.power.budget.hysteresis` | BSC 产品私有前缀 `DeviceGSM.` |
| 52 | 181 | `DeviceGSM.Bts.{i}.handover2.power.budget.interval` | BSC 产品私有前缀 `DeviceGSM.` |
| 53 | 182 | `DeviceGSM.Bts.{i}.handover2.retries` | BSC 产品私有前缀 `DeviceGSM.` |
| 54 | 183 | `DeviceGSM.Bts.{i}.handover2.tdma-measurement` | BSC 产品私有前缀 `DeviceGSM.` |
| 55 | 184 | `DeviceGSM.Bts.{i}.handover2.window.rxlev.averaging` | BSC 产品私有前缀 `DeviceGSM.` |
| 56 | 185 | `DeviceGSM.Bts.{i}.handover2.window.rxlev.neighbor.averaging` | BSC 产品私有前缀 `DeviceGSM.` |
| 57 | 186 | `DeviceGSM.Bts.{i}.handover2.window.rxqual.averaging` | BSC 产品私有前缀 `DeviceGSM.` |
| 58 | 187 | `DeviceGSM.BtsNum` | BSC 产品私有前缀 `DeviceGSM.` |
| 59 | 188 | `DeviceGSM.Cs7Instance.ForAs.Index` | BSC 产品私有前缀 `DeviceGSM.` |
| 60 | 189 | `DeviceGSM.Cs7Instance.ForAsp.Index` | BSC 产品私有前缀 `DeviceGSM.` |
| 61 | 190 | `DeviceGSM.Cs7Instance.ForSccpAddr.Index` | BSC 产品私有前缀 `DeviceGSM.` |
| 62 | 191 | `DeviceGSM.Cs7Instance.{i}.As.{i}.Name` | BSC 产品私有前缀 `DeviceGSM.` |
| 63 | 192 | `DeviceGSM.Cs7Instance.{i}.As.{i}.RoutingKey` | BSC 产品私有前缀 `DeviceGSM.` |
| 64 | 193 | `DeviceGSM.Cs7Instance.{i}.As.{i}.TrafficMode` | BSC 产品私有前缀 `DeviceGSM.` |
| 65 | 194 | `DeviceGSM.Cs7Instance.{i}.As.{i}.addAspName` | BSC 产品私有前缀 `DeviceGSM.` |
| 66 | 195 | `DeviceGSM.Cs7Instance.{i}.As.{i}.delAspName` | BSC 产品私有前缀 `DeviceGSM.` |
| 67 | 196 | `DeviceGSM.Cs7Instance.{i}.Asp.{i}.AspRole` | BSC 产品私有前缀 `DeviceGSM.` |
| 68 | 197 | `DeviceGSM.Cs7Instance.{i}.Asp.{i}.LocalIp` | BSC 产品私有前缀 `DeviceGSM.` |
| 69 | 198 | `DeviceGSM.Cs7Instance.{i}.Asp.{i}.LocalPort` | BSC 产品私有前缀 `DeviceGSM.` |
| 70 | 199 | `DeviceGSM.Cs7Instance.{i}.Asp.{i}.Name` | BSC 产品私有前缀 `DeviceGSM.` |
| 71 | 200 | `DeviceGSM.Cs7Instance.{i}.Asp.{i}.Protocol` | BSC 产品私有前缀 `DeviceGSM.` |
| 72 | 201 | `DeviceGSM.Cs7Instance.{i}.Asp.{i}.RemoteIp` | BSC 产品私有前缀 `DeviceGSM.` |
| 73 | 202 | `DeviceGSM.Cs7Instance.{i}.Asp.{i}.RemotePort` | BSC 产品私有前缀 `DeviceGSM.` |
| 74 | 203 | `DeviceGSM.Cs7Instance.{i}.Asp.{i}.SctpRole` | BSC 产品私有前缀 `DeviceGSM.` |
| 75 | 204 | `DeviceGSM.Cs7Instance.{i}.NetworkIndicator` | BSC 产品私有前缀 `DeviceGSM.` |
| 76 | 205 | `DeviceGSM.Cs7Instance.{i}.PointCode` | BSC 产品私有前缀 `DeviceGSM.` |
| 77 | 206 | `DeviceGSM.Cs7Instance.{i}.PointCodeFormat` | BSC 产品私有前缀 `DeviceGSM.` |
| 78 | 207 | `DeviceGSM.Cs7Instance.{i}.SccpAddr.{i}.Name` | BSC 产品私有前缀 `DeviceGSM.` |
| 79 | 208 | `DeviceGSM.Cs7Instance.{i}.SccpAddr.{i}.PointCode` | BSC 产品私有前缀 `DeviceGSM.` |
| 80 | 209 | `DeviceGSM.Cs7Instance.{i}.SccpAddr.{i}.RoutingIndicator` | BSC 产品私有前缀 `DeviceGSM.` |
| 81 | 210 | `DeviceGSM.Cs7Instance.{i}.SccpAddr.{i}.SubsystemNumber` | BSC 产品私有前缀 `DeviceGSM.` |
| 82 | 211 | `DeviceGSM.Cs7Instance.{i}.XuaRkmRoutingKeyAllocation` | BSC 产品私有前缀 `DeviceGSM.` |
| 83 | 212 | `DeviceGSM.Encryption` | BSC 产品私有前缀 `DeviceGSM.` |
| 84 | 213 | `DeviceGSM.HandoverAlgorithm` | BSC 产品私有前缀 `DeviceGSM.` |
| 85 | 214 | `DeviceGSM.Hodec2CongestionCheck` | BSC 产品私有前缀 `DeviceGSM.` |
| 86 | 215 | `DeviceGSM.Mcc` | BSC 产品私有前缀 `DeviceGSM.` |
| 87 | 216 | `DeviceGSM.Mgw.{i}.MgwEndpointDomain` | BSC 产品私有前缀 `DeviceGSM.` |
| 88 | 217 | `DeviceGSM.Mgw.{i}.MgwLocalPort` | BSC 产品私有前缀 `DeviceGSM.` |
| 89 | 218 | `DeviceGSM.Mgw.{i}.MgwRemoteIp` | BSC 产品私有前缀 `DeviceGSM.` |
| 90 | 219 | `DeviceGSM.Mgw.{i}.MgwRemotePort` | BSC 产品私有前缀 `DeviceGSM.` |
| 91 | 220 | `DeviceGSM.Mnc` | BSC 产品私有前缀 `DeviceGSM.` |
| 92 | 221 | `DeviceGSM.Msc.{i}.AllowEmergency` | BSC 产品私有前缀 `DeviceGSM.` |
| 93 | 222 | `DeviceGSM.Msc.{i}.Amr10_2` | BSC 产品私有前缀 `DeviceGSM.` |
| 94 | 223 | `DeviceGSM.Msc.{i}.Amr12_2` | BSC 产品私有前缀 `DeviceGSM.` |
| 95 | 224 | `DeviceGSM.Msc.{i}.Amr4_75` | BSC 产品私有前缀 `DeviceGSM.` |
| 96 | 225 | `DeviceGSM.Msc.{i}.Amr5_15` | BSC 产品私有前缀 `DeviceGSM.` |
| 97 | 226 | `DeviceGSM.Msc.{i}.Amr5_90` | BSC 产品私有前缀 `DeviceGSM.` |
| 98 | 227 | `DeviceGSM.Msc.{i}.Amr6_70` | BSC 产品私有前缀 `DeviceGSM.` |
| 99 | 228 | `DeviceGSM.Msc.{i}.Amr7_40` | BSC 产品私有前缀 `DeviceGSM.` |
| 100 | 229 | `DeviceGSM.Msc.{i}.Amr7_95` | BSC 产品私有前缀 `DeviceGSM.` |
| 101 | 230 | `DeviceGSM.Msc.{i}.AmrPayload` | BSC 产品私有前缀 `DeviceGSM.` |
| 102 | 231 | `DeviceGSM.Msc.{i}.AspProtocol` | BSC 产品私有前缀 `DeviceGSM.` |
| 103 | 232 | `DeviceGSM.Msc.{i}.AttachProportion` | BSC 产品私有前缀 `DeviceGSM.` |
| 104 | 233 | `DeviceGSM.Msc.{i}.LclsMismatch` | BSC 产品私有前缀 `DeviceGSM.` |
| 105 | 234 | `DeviceGSM.Msc.{i}.LclsMode` | BSC 产品私有前缀 `DeviceGSM.` |
| 106 | 235 | `DeviceGSM.Msc.{i}.MscAddr` | BSC 产品私有前缀 `DeviceGSM.` |
| 107 | 236 | `DeviceGSM.Msc.{i}.MscCodecList` | BSC 产品私有前缀 `DeviceGSM.` |
| 108 | 237 | `DeviceGSM.Msc.{i}.NriAdd` | BSC 产品私有前缀 `DeviceGSM.` |
| 109 | 238 | `DeviceGSM.Msc.{i}.NriDel` | BSC 产品私有前缀 `DeviceGSM.` |
| 110 | 239 | `DeviceGSM.NriBitLen` | BSC 产品私有前缀 `DeviceGSM.` |
| 111 | 240 | `DeviceGSM.NriNullAdd` | BSC 产品私有前缀 `DeviceGSM.` |
| 112 | 241 | `DeviceGSM.NriNullDel` | BSC 产品私有前缀 `DeviceGSM.` |
| 113 | 242 | `DeviceGSM.TimerNetT3212` | BSC 产品私有前缀 `DeviceGSM.` |
| 114 | 243 | `DeviceGSM.handover` | BSC 产品私有前缀 `DeviceGSM.` |
| 115 | 244 | `DeviceGSM.handover2.afs-bias.rxlev` | BSC 产品私有前缀 `DeviceGSM.` |
| 116 | 245 | `DeviceGSM.handover2.afs-bias.rxqual` | BSC 产品私有前缀 `DeviceGSM.` |
| 117 | 246 | `DeviceGSM.handover2.assignment` | BSC 产品私有前缀 `DeviceGSM.` |
| 118 | 247 | `DeviceGSM.handover2.max-handovers` | BSC 产品私有前缀 `DeviceGSM.` |
| 119 | 248 | `DeviceGSM.handover2.maximum.distance` | BSC 产品私有前缀 `DeviceGSM.` |
| 120 | 249 | `DeviceGSM.handover2.min-free-slots.tch-f` | BSC 产品私有前缀 `DeviceGSM.` |
| 121 | 250 | `DeviceGSM.handover2.min-free-slots.tch-h` | BSC 产品私有前缀 `DeviceGSM.` |
| 122 | 251 | `DeviceGSM.handover2.min.rxlev` | BSC 产品私有前缀 `DeviceGSM.` |
| 123 | 252 | `DeviceGSM.handover2.min.rxqual` | BSC 产品私有前缀 `DeviceGSM.` |
| 124 | 253 | `DeviceGSM.handover2.penalty-time.failed-assignment` | BSC 产品私有前缀 `DeviceGSM.` |
| 125 | 254 | `DeviceGSM.handover2.penalty-time.failed-ho` | BSC 产品私有前缀 `DeviceGSM.` |
| 126 | 255 | `DeviceGSM.handover2.penalty-time.low-rxqual-assignment` | BSC 产品私有前缀 `DeviceGSM.` |
| 127 | 256 | `DeviceGSM.handover2.penalty-time.low-rxqual-ho` | BSC 产品私有前缀 `DeviceGSM.` |
| 128 | 257 | `DeviceGSM.handover2.penalty-time.max-distance` | BSC 产品私有前缀 `DeviceGSM.` |
| 129 | 258 | `DeviceGSM.handover2.power.budget.hysteresis` | BSC 产品私有前缀 `DeviceGSM.` |
| 130 | 259 | `DeviceGSM.handover2.power.budget.interval` | BSC 产品私有前缀 `DeviceGSM.` |
| 131 | 260 | `DeviceGSM.handover2.retries` | BSC 产品私有前缀 `DeviceGSM.` |
| 132 | 261 | `DeviceGSM.handover2.tdma-measurement` | BSC 产品私有前缀 `DeviceGSM.` |
| 133 | 262 | `DeviceGSM.handover2.window.rxlev.averaging` | BSC 产品私有前缀 `DeviceGSM.` |
| 134 | 263 | `DeviceGSM.handover2.window.rxlev.neighbor.averaging` | BSC 产品私有前缀 `DeviceGSM.` |
| 135 | 264 | `DeviceGSM.handover2.window.rxqual.averaging` | BSC 产品私有前缀 `DeviceGSM.` |

## 校验结果

- XML 可正常解析。
- XML 剩余不匹配项仅为有子路径的对象/表节点，或 `DeviceGSM.` 产品私有参数。
- XML 独有叶子参数（排除 `DeviceGSM.` 产品私有前缀）数量为 `0`。
