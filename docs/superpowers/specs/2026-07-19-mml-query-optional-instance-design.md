# MML 查询对象实例可选与 Path 截断设计

**日期：** 2026-07-19
**状态：** 已确认
**适用范围：** MML 控制台 LST、DSP 和自定义查询命令

## 1. 背景

MML 控制台当前根据查询 Path 中 `.{i}.` 占位符的最大层数渲染对象实例输入框，并将每个实例默认填为 `1`。结构化 LST 请求把实例选择器透传给后端，后端要求实例选择器数量与 Path 占位符数量严格相等；DSP 和自定义查询走裸路径请求。

本需求将查询命令的对象实例改为可选输入。最后一层实例默认留空；当某层实例为空时，查询 Path 截断到该实例所在对象位置，不再拼接该实例及其后的 Path。

示例：

- `Device.A.{i}.B.{i}.Value`，实例为 `1 / 空`，实际查询 `Device.A.1.B.`
- `Device.A.{i}.Value`，实例为空，实际查询 `Device.A.`
- `Device.A.{i}.B.{i}.Value`，实例为 `空 / 1`，实际查询 `Device.A.`，第二层实例被忽略

截断结果是 TR-069 partial object path，供 GetParameterValues 类查询获取该对象下的数据。

## 2. 目标

- 查询 Path 只有一个 `{i}` 时，对象实例输入框默认空。
- 查询 Path 有多个 `{i}` 时，前 N−1 个输入框默认 `1`，最后一个输入框默认空。
- 查询类对象实例输入框全部非必填，用户可清空任意一层。
- 从左到右遇到第一个空实例时，将每条查询 Path 截断到当前对象位置。
- 同时覆盖标准 LST、DSP 和自定义查询。
- 保持现有 API 字段与标准参数树精确匹配逻辑不变。

## 3. 非目标

- 不修改 MOD 的完整实例路径要求。
- 不修改 ADD、RMV 的对象实例默认值、目标对象展示或下发逻辑。
- 不改变命令 Path 选择、多 Path 整体/逐 Path 执行、参数结果展示等其他流程。
- 不新增实例自动发现、实例选择列表或新的后端 API 字段。
- 不改变裸路径专家模式；本需求只处理“命令参数”标准模式中的查询命令。

## 4. 页面行为

### 4.1 实例输入框初始化

实例槽仍由已确认查询 Path 中占位符层数最多的 Path 决定。

| 实例槽数量 | 默认值 |
| --- | --- |
| 0 | 不显示实例输入框 |
| 1 | `空` |
| N，N > 1 | 前 N−1 个为 `1`，第 N 个为 `空` |

切换命令或重新确认 Path 后，按新的 Path 集合重新计算实例槽并重新初始化默认值。

### 4.2 输入约束

- 每个查询实例输入框允许为空。
- 非空值必须为大于等于 `1` 的整数。
- 清空任意一层后，不要求清空后续输入框；解析 Path 时会忽略第一个空实例之后的所有实例值。
- MOD、ADD、RMV 继续使用现有必需完整实例的交互和默认值。

### 4.3 不同深度 Path

多个已选 Path 的 `{i}` 数量可能不同。页面以最深 Path 渲染实例槽，但每条 Path 只消费自身包含的占位符：

- Path 占位符少于实例槽时，忽略多余实例值，不报数量不匹配。
- Path 在自身占位符范围内遇到空实例时，按该位置截断。
- 不含 `{i}` 的 Path 原样查询。

## 5. Path 解析规则

新增查询专用纯函数，输入原始 Path 与 `instanceSelectors`，输出实际查询 Path。

处理顺序：

1. 按 `i01`、`i02`、`i03` 顺序对应 Path 中从左到右的 `.{i}.`。
2. 对非空实例，将对应 `.{i}.` 替换为实例号。
3. 对缺失或空字符串实例，返回当前 `.{i}.` 之前的对象前缀，并保留末尾 `.`。
4. 一旦截断，不再读取后续实例，也不再拼接后续 Path。
5. Path 中所有占位符都有非空值时，返回完整实例 Path。
6. Path 不含占位符时原样返回。

示例：

| 原始 Path | 实例选择器 | 结果 |
| --- | --- | --- |
| `Device.A.{i}.Value` | `{i01: ""}` | `Device.A.` |
| `Device.A.{i}.B.{i}.Value` | `{i01: "1", i02: ""}` | `Device.A.1.B.` |
| `Device.A.{i}.B.{i}.Value` | `{i01: "", i02: "2"}` | `Device.A.` |
| `Device.A.{i}.B.{i}.Value` | `{i01: "3", i02: "2"}` | `Device.A.3.B.2.Value` |
| `Device.Info.SerialNumber` | `{}` | `Device.Info.SerialNumber` |

## 6. 数据流

### 6.1 标准 LST

标准 LST 必须继续用原始标准 Path 模板调用结构化接口，避免破坏 `StructuredToStatement` 对标准参数树的精确匹配。

流程：

1. 前端提交原始 `paths`，例如 `Device.A.{i}.B.{i}.Value`。
2. 前端提交完整实例槽 map，空实例以空字符串表示，例如 `{i01: "1", i02: ""}`。
3. 后端完成命令和标准 Path 匹配后，为 LST 的 `param_refs` 应用查询专用实例解析。
4. 解析得到 `Device.A.1.B.`，再进入现有 GetParameterValues payload 构建与去重逻辑。

该流程不修改结构化请求字段，也不放宽标准 Path 匹配。

### 6.2 DSP 与自定义查询

DSP 和自定义查询走现有裸路径通道，没有 `instance_selectors` 请求字段。前端在构建裸路径请求前，对每个已选 Path 应用同一查询 Path 解析规则，再将实际 Path 写入请求。

多条 Path 独立解析。整体执行和逐 Path 执行继续沿用现有分支。

### 6.3 写命令隔离

- MOD 继续使用现有严格实例替换：选择器数量必须匹配 Path 占位符数量，不能产生 partial path。
- ADD、RMV 继续使用现有 `resolveObjectPath` 和目标对象实例逻辑。
- 查询专用 helper 不被写命令调用。

## 7. 后端行为

后端新增 LST 查询专用实例解析，不修改现有 `substituteInstanceSelectors` 的严格语义。

查询专用解析器应支持：

- 空选择器值触发截断。
- 缺少某层 selector 等价于该层为空。
- 多余 selector 被当前 Path 忽略。
- 多个 `param_refs` 具有不同占位符深度时逐条独立解析。
- 非空实例值替换后继续走现有 Path 合规检查。

旧调用方完全不传 `instance_selectors` 时，保留现有兼容逻辑：GetParameterValues payload 构建器仍可把未解析 `{i}` Path 展开到第一个占位符之前的 partial path。

## 8. 错误处理

- 页面使用 `InputNumber min={1}`；空值合法，非空非法格式不进入请求。
- 后端收到非空但无法形成合法 TR-069 Path 的实例值时，继续通过现有 Path 校验返回受控错误。
- 某条 Path 截断后为空或不能形成合法对象前缀时，沿用现有 `ErrNoUsableParams` 行为。
- 不新增错误码或用户可见错误文案。

## 9. 测试策略

### 9.1 前端组件

覆盖 `ConfigParamsModal`：

- LST 单实例默认空。
- LST 双实例默认 `1 / 空`。
- LST 三实例默认 `1 / 1 / 空`。
- 查询实例输入框可清空。
- 切换已确认 Path 后重新初始化。
- MOD、ADD、RMV 原默认值与请求保持不变。

### 9.2 前端 Path 解析与请求

覆盖纯函数和控制台请求适配：

- 单层空实例截断。
- 双层和三层在最后一层截断。
- 第一层或中间层为空时提前截断。
- 不同深度 Path 使用同一 selector map。
- DSP 和自定义查询下发实际截断 Path。
- 标准 LST 仍发送原始模板 Path 和包含空字符串的 selector map。

### 9.3 后端

覆盖 LST 查询实例解析：

- 单层、双层和三层替换/截断。
- 缺失 selector 与空字符串语义一致。
- 多余 selector 被忽略。
- 不同深度 `param_refs` 可同时解析。
- MOD 严格数量校验回归保持不变。
- 最终 GetParameterValues payload 使用截断后的 partial object path。

### 9.4 验证命令

```bash
cd omcmb
npm test --workspace webcode -- src/pages/mml/Console/components/ConfigParamsModal.test.tsx src/pages/mml/Console/__tests__/instanceAndRaw.test.ts
npm run typecheck

cd ../omcgo
go test ./internal/mml -count=1
go build ./...
go test ./...
```

浏览器验收使用真实后端，至少覆盖：

- 单 `{i}` 查询默认空并下发父对象 Path。
- 双 `{i}` 查询默认 `1 / 空` 并下发第一层实例下的子对象 Path。
- 手工清空第一层时从第一层位置截断。
- MOD 实例行为无回归。

## 10. 验收标准

- 查询实例默认值符合“前层为 1、末层为空”规则。
- 所有查询实例输入框均可留空并执行。
- 第一个空实例之后的 Path 不进入实际查询请求。
- LST、DSP、自定义查询均得到正确的 partial object path。
- 标准参数树匹配、MOD、ADD、RMV 行为不变。
- 相关自动化测试、前端 typecheck、后端构建和测试全部通过。
