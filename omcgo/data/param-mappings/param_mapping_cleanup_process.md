# 参数映射清理流程

本文档用于后续处理其他站型的参数映射 XML。

## 处理目标

以 `references/NR全量参数集.csv` 中的 `trpath.name` 为基准，清理目标站型 XML 中独有且无效的叶子参数，同时保留有效对象/表节点，并按 CSV 补齐需要更新的参数组。

## 输入文件

- 目标站型 XML：`omcgo/data/param-mappings/<站型>.xml`
- 基准 CSV：`omcgo/data/param-mappings/references/NR全量参数集.csv`

## 对比口径

1. XML 参数来源：`<parameters>` 下的 `<param name="...">`。
2. CSV 参数来源：`trpath.name` 字段。
3. 对比前统一做实例归一化：
   - `FAPService.1.` 视为 `FAPService.{i}.`
   - `CellConfig.1.` 视为 `CellConfig.{i}.`
   - `.1.`、`.2.` 等数字实例段统一视为 `.{i}.`
4. CSV 中 `trpath.name` 为空或为 `--` 的记录不参与路径匹配。

## 删除规则

1. 找出 XML 中存在、但 CSV 中按归一化规则匹配不到的参数。
2. 对这些 XML 独有参数继续判断是否有子路径：
   - 如果存在其他参数以 `当前路径 + "."` 开头，则当前路径是对象/表节点，保留。
   - 如果不存在子路径，则当前路径是独有叶子参数，删除。
3. 示例：
   - `Device.FAP.License.LicenseItem.{i}` 有 `Device.FAP.License.LicenseItem.{i}.ID` 等子参数，保留。
   - `Device.FAP.License.LicenseItem.{i}.Desciption` 没有子参数，且 CSV 中不存在，删除。

## CSV 更新规则

如果 CSV 中存在某组参数，而 XML 中缺失或被旧口径误删，需要按 CSV 补齐。

以 `RRCTimers` 为例：

1. 从 CSV 中筛选 `trpath.name` 包含 `RRCTimers` 的记录。
2. 写入 XML 时路径仍按通用多实例处理：
   - CSV：`Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.RRCTimers.T300`
   - XML：`Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RRCTimers.T300`
3. 参数属性按 CSV 更新：
   - `access` 使用 `trpath.access`
   - `type` 使用 `trpath.datatype`
   - `min` 使用 `trpath.minval`
   - `max` 使用 `trpath.maxval`
   - `defaultValue` 优先使用 `attr.default`

## XML 更新要求

1. 删除最终确认的独有叶子 `<param>` 节点。
2. 保留有子路径的对象/表节点。
3. 补齐需要按 CSV 更新的参数组。
4. 路径中的固定实例号按通用多实例泛化。
5. 同步调整根节点 `totalEntries`。

## Review 文档要求

每次处理一个站型时，新建 review 文档，建议命名：

`<站型>_trpath_review.md`

文档至少包含：

1. 对比口径。
2. 删除参数数量。
3. 删除参数清单。
4. 保留的对象/表节点清单。
5. 如果有按 CSV 更新的参数组，单独列出更新清单。
6. `totalEntries` 调整说明。

## 最终校验

处理完成后必须校验：

1. XML 能正常解析。
2. CSV 中需要补齐的参数组，在 XML 中按归一化规则全部能匹配到。
3. XML 剩余不匹配项只能是有子路径的对象/表节点。
4. XML 独有叶子参数数量为 `0`。
5. `git diff` 只包含目标 XML 和对应 review 文档。

## 本次 BaiBNQ 处理口径

本次 `BaiBNQ.xml` 的最终口径为：

- 按实例归一化对比。
- 只删除 XML 独有且没有子路径的叶子参数。
- 有子节点的对象/表节点保留。
- `RRCTimers` 按 CSV 补齐并泛化 `FAPService.1.` 为 `FAPService.{i}.`。
- Review 文档记录 CSV 来源路径和 XML 泛化路径。
