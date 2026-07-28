# BTS trpath 删除 Review

## 对比口径

- XML 文件：`/Users/wangyong/OBJECT/Codex/xomc/omcgo/data/param-mappings/BTS.xml`
- CSV 文件：`/Users/wangyong/OBJECT/Codex/xomc/omcgo/data/param-mappings/references/GSM全量参数集.csv`
- CSV 字段：`trpath.name`
- 对比时将路径中的数字实例段归一为 `{i}`，例如 `.1.` 按 `.{i}.` 处理。
- XML 独有但仍有子路径的对象/表节点视为有效参数，保留不删除。

## 删除汇总

- 删除参数数量：5
- 删除范围：仅删除 `<parameters>` 下 XML 独有且没有子路径的叶子 `<param>` 节点。
- 保留对象/表节点数量：1
- `totalEntries`：已从 `279` 调整为 `274`。

## 删除参数清单

| 序号 | 原 XML 行号 | trpath |
| ---: | ---: | --- |
| 1 | 31 | `Device.FaultMgmt.CurrentAlarm.{i}.AlarmChangedTime` |
| 2 | 282 | `DeviceGSM.BscSelect` |
| 3 | 283 | `DeviceGSM.IpaUnitId` |
| 4 | 284 | `DeviceGSM.OmlRemoteIp` |
| 5 | 285 | `DeviceGSM.OmlRemoteIpBak` |

## 保留对象/表节点清单

| 序号 | 原 XML 行号 | trpath | 保留原因 |
| ---: | ---: | --- | --- |
| 1 | 60 | `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.Capacity.{i}` | 存在子路径 |

## 校验结果

- XML 可正常解析。
- XML 剩余不匹配项仅为有子路径的对象/表节点。
- XML 独有叶子参数数量为 `0`。
