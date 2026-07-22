# MML 控制台 boolean 参数选择框设计

## 背景

MML 控制台的 MOD/ADD 参数目前统一使用文本输入框。对于 `boolean` 类型参数，文本输入容易填入无效值，也无法直观看出可提交的取值范围。

## 方案

在 `ConfigParamsModal` 的标准参数编辑区按 `CommandParamPath.valueType` 识别 boolean 类型：

- `boolean` 参数使用 Ant Design `Select`，选项固定为 `true` 和 `false`。
- 选项的 label 与 value 均使用字符串 `true` / `false`，保持现有 `ExecRequest.values: Record<string, string>` 契约及 TR-069 下发值格式。
- 已有 `defaultValue` 为 `true`/`false` 时沿用它；兼容历史 `minValue` 的 `0`/`1` 表示；均未提供时默认 `false`，不提供空选项。
- 非 boolean 参数继续使用现有 `Input`，现有范围校验和提交时提示逻辑不变。
- 选择框使用与文本输入相同的整行宽度，长 PATH 的布局和滚动行为不变。

## 测试与验收

- 组件测试确认 boolean 参数渲染为 `combobox` 而非 `textbox`，包含 `true`/`false` 两个选项。
- 测试选择 `true` 后执行请求携带对应 PATH 的字符串值 `"true"`。
- 测试无默认值时初始值为 `false`。
- 测试普通 string 参数仍渲染文本输入框。
- 运行 `cd omcmb && npm run typecheck` 与 MML 控件测试。
- 在 113 环境实际打开 MML 控制台，确认 boolean 参数显示选择框、切换值和执行前请求值正确。
