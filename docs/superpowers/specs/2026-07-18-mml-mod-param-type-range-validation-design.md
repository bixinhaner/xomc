# MML 修改参数类型展示与范围校验设计

> 日期：2026-07-18
> 状态：已确认，待实现
> 范围：`/mml/console` 的 MOD“选择命令 → 配置参数”流程

## 1. 背景

MML 控制台的 MOD 配置参数页当前只展示参数标签、Path 和文本输入框。虽然命令关联的
Path 均来自标准参数树，页面没有展示 `standard_params.data_type`，也没有按
`standard_params.min_value` / `max_value` 校验用户填写的修改值。

现有链路已经把 `data_type` 和 `min_value` 传到部分前端模型，但存在两个缺口：

- MML 命令子字段接口没有返回 `max_value`。
- 配置页把 `min_value` 用作默认输入值，只校验选中参数是否为空，没有校验范围。

本次需求以标准参数树为唯一真值源，在不改变 MML 执行请求格式的前提下，为 MOD 参数增加
类型展示和输入阶段的范围校验。

## 2. 已确认需求

1. MOD 配置参数页在每个 Path 后展示该参数的数据类型。
2. 数据类型必须与标准参数树 `standard_params.data_type` 保持一致，不在前端重新推断。
3. 参数的最小值和最大值来自标准参数树。
4. 标准参数树配置了任一边界时才进行对应范围校验；两个边界都未配置时不做范围校验。
5. `string` 的最小值和最大值表示字符串长度范围。
6. `int`、`unsignedInt` 等数值类型的最小值和最大值表示整数取值范围。
7. `boolean`、`dateTime` 沿用标准参数树现有语义，不做范围校验。
8. 只配置最小值时仅校验下限，只配置最大值时仅校验上限；双边范围为闭区间。
9. 保留现有默认值行为：有 `min_value` 时继续把它填入 MOD 输入框。
10. 所有 MML 参数都存在于标准参数树中，内置和自定义 MOD 命令使用同一套规则。
11. 本次只调整标准“命令参数”页，不调整“指定参数”裸路径页和 ADD/RMV 流程。

## 3. 方案选择

### 3.1 采用方案：扩展现有参数元数据链路

在现有命令子字段和自定义命令 Path 接口上补齐 `max_value`，前端把标准参数树的
`data_type`、`min_value`、`max_value` 统一归一到 `CommandParamPath`。配置弹框只消费
已经加载的命令参数元数据，不在输入过程中追加查询。

优点：

- 标准参数树保持唯一真值源。
- 内置命令不增加额外请求。
- 内置和自定义命令可复用同一套展示与校验函数。
- 执行请求继续使用现有 `checkedPaths` 和 `values`，不改变后端执行协议。

### 3.2 未采用方案

- **前端按 Path 单独查询标准参数树**：会产生额外请求和加载状态，多个 Path 时容易形成
  N+1 请求，也可能出现命令元数据与页面查询结果时间点不一致。
- **只在执行阶段由后端拦截**：不能在输入框旁即时提示，也不能提前禁用执行按钮，无法满足
  页面校验需求。
- **把类型规则直接写在 JSX 中**：初始改动少，但内置/自定义命令和组件测试会复制逻辑，
  后续数据类型扩展时容易产生差异。

## 4. 数据来源与接口设计

### 4.1 内置命令

现有数据流保持不变，只补齐最大值：

```text
standard_params
  → PgSubFieldRepository.ListEnrichedByCommand
  → MMLCommandSubFieldEnriched
  → ConsoleService.GetCommandSubFields / SubFieldDTO
  → GET /api/v1/mml/commands/:id/sub-fields
  → BackendSubField / SubFieldDef
  → CommandParamPath
  → ConfigParamsModal
```

接口响应单元新增：

```json
{
  "value_type": "unsignedInt",
  "min_value": 1,
  "max_value": 65535
}
```

字段均允许为空；JSON 中未返回或返回 `null` 时，前端统一归一为 `undefined`。

### 4.2 自定义命令

自定义命令兼容 API 以 `mml_custom_command.param_paths` JSON 保存字符串 Path，
富化视图则通过 `mml_custom_command_paths.standard_path_id` 关联
`standard_params`。创建或更新命令时，后端必须在同一事务中把 JSON Path 解析为
关联行；读取同步逻辑上线前的历史命令时，富化查询还需按 JSON Path 回退 JOIN
`standard_params`，不能因缺少关联行返回空列表。关联增删改接口也必须在同一事务中
同步 JSON：新增追加 Path，排序更新 JSON 顺序，删除同时移除 JSON Path，避免接口成功
但控制台有效 Path 不变。

历史 JSON-only 回退行没有真实关联 ID，接口以 `mutable=false` 明确标记为只读；其
字符串 Path 按正常写入规则去首尾空白（含 Tab/CR/LF）、丢弃空值、按首次出现位置
去重并稳定排序。

父命令 PUT 允许省略 `param_paths`。后端必须保留“字段是否提供”的更新掩码，并在
repository 事务锁定父命令后读取最新 JSON；省略时使用锁内最新值，防止 service 层
旧快照覆盖并发的 Path 关联增删改。

自定义命令不再仅依赖字符串 Path 构造 MOD 配置项；JSON 只确定命令声明和产品过滤后的
Path 集合，类型、访问权限和范围必须来自 `standard_params`。富化结果必须与
`GET /mml/templates?product_id=...` 已裁剪的 `paramPaths` 取交集，防止产品不支持的
关联 Path 回流到选择器和执行请求。

### 4.3 数据库与执行协议

- 不新增数据库字段，不新增 migration。
- 不修改标准参数树维护页面的数据结构。
- 不修改 MOD 执行请求、任务结构、TR-069 下发格式和结果展示协议。

## 5. 前端模型

`CommandParamPath` 补齐以下字段：

```ts
interface CommandParamPath {
  valueType?: string;
  minValue?: number;
  maxValue?: number;
}
```

字段语义：

- `valueType`：标准参数树原始类型字符串，用于页面展示和选择范围校验语义。
- `minValue`：范围下限，同时保留现有输入框默认填充值行为。
- `maxValue`：范围上限。

内置和自定义命令都在 adapter 层完成字段归一，配置弹框不识别后端 snake_case 字段。

## 6. 交互设计

### 6.1 类型展示

MOD 配置页每行保持“参数标签 + Path + 输入框”的布局，在 Path 后增加中性类型标签：

```text
小区标识  Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellIdentity  [unsignedInt]
```

- 标签内容直接展示标准参数树中的原始 `data_type`。
- 不把 `unsignedInt` 转换成自定义中文名称，避免与标准参数树定义产生差异。
- 类型标签只用于说明，不改变输入框的数据提交格式。

### 6.2 默认值

保持现有行为：

- `min_value` 存在时，打开 MOD 配置页后输入框默认填入该值。
- `min_value` 不存在时，输入框保持为空。
- 默认填充值仍参与与用户输入相同的校验。
- 若标准参数树出现 `min_value > max_value` 的异常配置，页面不得静默放行；默认值会显示为
  越界并禁用执行，促使维护人员修复标准参数树。

### 6.3 校验反馈

每个输入框独立显示错误状态和错误文案。任一选中参数为空或范围校验失败时，
“确定并执行”按钮保持禁用。

错误反馈至少区分：

- 必填值为空。
- 数值范围参数填写的内容不是合法整数。
- 小于最小值。
- 大于最大值。
- 字符串长度小于最小长度。
- 字符串长度大于最大长度。

用户修改为合法值后即时清除对应错误，不需要额外点击校验按钮。

所有用户可见文案同时维护中文和英文，不在组件中硬编码。

## 7. 校验规则

提取纯函数处理单个 `CommandParamPath` 和输入值，组件只负责展示结果。

### 7.1 通用规则

1. 先执行现有 MOD 必填校验。
2. `minValue` 和 `maxValue` 都为空时，范围校验直接通过，不额外校验数据类型。
3. 只有一个边界时，只检查该边界。
4. 边界值本身合法，范围为闭区间。
5. 标准树存量类型同时存在规范小驼峰与历史大写/下划线形式，例如
   `string/STRING`、`boolean/BOOLEAN`、`dateTime/DATE_TIME`、
   `unsignedInt/U_INT`；范围语义判断大小写不敏感并忽略分隔符，类型标签仍展示原值。

### 7.2 字符串

当 `dataTypeRangeKind(valueType) === 'length'` 时：

- 按 Unicode 字符数量计算长度，与 Go 后端 `utf8.RuneCountInString` 语义对齐。
- 不使用 JavaScript UTF-16 code unit 数量，避免 emoji 等字符在前后端得到不同长度。
- 将实际字符数与 `minValue` / `maxValue` 比较。

### 7.3 数值

当 `dataTypeRangeKind(valueType) === 'value'` 且至少配置一个边界时：

- 去掉输入首尾空白后按十进制整数解析。
- 无法解析为整数时返回类型错误。
- 解析成功后与 `minValue` / `maxValue` 比较。
- `unsignedInt` 的有效下限仍由标准参数树中的边界决定；本次不在前端额外补一个隐式 `0`。

最后一条保证“配置了才校验”：即使类型为 `unsignedInt`，若标准参数树没有配置范围，
也不因本需求新增范围限制。

### 7.4 无范围类型

当 `dataTypeRangeKind(valueType) === 'none'` 时不做范围校验。标准参数树维护页面目前会在
切换到 `boolean` 或 `dateTime` 时清空 min/max，本页沿用该约束语义。

## 8. 组件边界

### 后端

- `PgSubFieldRepository`：查询并扫描 `sp.max_value`。
- `MMLCommandSubFieldEnriched`、`SubFieldDTO`：增加 `MaxValue`。
- 自定义命令创建/更新 repository：同事务同步 JSON `param_paths` 与关联表。
- 自定义命令 Path repository/view：关联增删改同时同步 JSON，查询返回
  `min_value`、`max_value` 和 `mutable`，并兼容历史 JSON-only 命令。
- 父命令更新：以字段掩码区分“省略 Path”和“明确清空 Path”，在父行锁内合并最新
  JSON，避免与 Path 写接口并发时丢更新。
- 对应 service/handler 保持透传，不引入新的业务推断。

### `frontend-core`

- `BackendSubField`、`SubFieldDef`：增加 `maxValue` 映射。
- 自定义命令 Path 类型/API：增加 `minValue`、`maxValue`、`mutable` 映射。
- 复用大小写不敏感的 `dataTypeRangeKind` 判断长度、数值和无范围类型。
- 模板、产品参数映射或标准参数元数据变化后，同时失效产品化自定义命令列表与富化
  Path 查询缓存。

### MML Console adapter

- 内置命令和自定义命令都归一为完整的 `CommandParamPath`。
- 保持 Path 原有顺序、可写过滤和已选 Path 裁剪规则。

### `ConfigParamsModal`

- 在 Path 后渲染类型标签。
- 保留 `minValue` 初始化。
- 调用纯校验函数生成逐字段错误。
- 合并现有必填校验和范围校验结果，控制执行按钮状态。
- 构造请求时仍只提交已选 Path 的字符串值。

## 9. 测试设计

按 TDD 先补失败测试，再实现最小改动。

### 9.1 后端测试

1. 内置命令子字段查询同时返回 `value_type`、`min_value`、`max_value`。
2. 仅配置单边范围时正确透传另一边为空。
3. 未配置范围时两个字段均为空。
4. 自定义命令 Path 富化视图返回相同的类型和范围。

### 9.2 纯函数测试

1. 两个边界都为空时任意非空输入通过。
2. `string` 在最小长度、最大长度边界上通过。
3. `string` 低于最小长度或高于最大长度时失败。
4. Unicode 字符和 emoji 按字符数量而不是 UTF-16 code unit 数量计算。
5. 数值在最小值、最大值边界上通过。
6. 数值低于最小值或高于最大值时失败。
7. 仅最小值和仅最大值分别生效。
8. 配置数值范围时，非整数输入返回类型错误。
9. `boolean`、`dateTime` 不做范围校验。

### 9.3 组件测试

1. MOD 参数 Path 后展示与标准参数树一致的类型标签。
2. `minValue` 继续作为初始输入值。
3. 无范围参数填写非空值后可执行。
4. 越界参数显示逐字段错误并禁用执行。
5. 修改为边界内值后错误消失、执行按钮恢复。
6. 多个选中 Path 中任一项失败时不能执行。
7. 只对上一页选中的可写 Path 展示和校验。
8. 自定义 MOD 命令使用与内置命令相同的类型和范围。
9. 查询命令、ADD/RMV 和裸路径页行为不变。

## 10. 验证

实现完成后执行：

- 后端相关 repository/service 测试。
- MML 后端包测试：`cd omcgo && go test ./internal/mml/...`。
- 前端相关 Vitest 测试。
- 前端类型检查：`cd omcmb && npm run typecheck`。
- `git diff --check`。
- 在目标测试环境使用真实浏览器验证 MOD 选择和配置流程。

浏览器验收至少覆盖：

1. 选择 MOD 命令和两个可写 Path。
2. 下一页只展示选中的两个 Path。
3. 每个 Path 后展示正确类型。
4. `min_value` 默认填充保持不变。
5. 输入边界值可以执行。
6. 输入越界值时显示错误且禁用执行。
7. 选择标准参数树未配置范围的 Path，确认不触发范围校验。
8. 不实际下发修改命令，避免改动真实设备。

开始实现前的基线记录：

- `cd omcmb && npm run typecheck` 通过。
- `cd omcgo && go test ./...` 中 `internal/mml` 通过。
- 完整 Go 测试存在两个与本需求无关的既有失败：
  - `internal/core/carrier.TestRegistryResolveByOUI`：OUI `00E0FC` 预期 `cmcc`，实际 `ctcc`。
  - `internal/core/dictloader.TestCacheVersion_SelfIncrement_DoesNotFireOwnBump`：
    自增版本意外触发一次自身 `OnBump`。

## 11. 非目标

- 不修改标准参数树的维护规则和取值范围数据。
- 不为未配置范围的参数推断隐式上下限。
- 不增加枚举、正则、日期格式或 boolean 格式校验。
- 不调整查询命令、ADD/RMV 或“指定参数”裸路径输入。
- 不改变 Path 前置选择、设备过滤、执行模式和权限判断。
- 不在本需求中修复开始实现前发现的两个无关 Go 基线失败。

## 12. 风险与控制

- **前后端长度语义不一致**：前端按 Unicode 字符数量计算，对齐 Go 的 rune 数。
- **只补前端模型但接口缺字段**：后端 repository、DTO、wire type 和 adapter 分层补测试。
- **自定义命令退化为字符串 Path**：选择时必须加载标准参数树富化元数据，再归一到
  `CommandParamPath`。
- **无范围参数被过度校验**：纯函数首先判断两个边界是否都为空；为空则不执行类型和范围校验。
- **默认最小值行为回退**：组件测试固定 `minValue` 初始化规则。
- **标准参数树范围配置异常**：页面显示校验错误并禁止执行，不在客户端纠正或交换上下限。
