# MML控制台
# 命令树功能完善
1、选择 MOD 类型的命令时，控制面板显示的参数列表都是支持修改的参数，用户未填入修改数据，在 API 提交时默认过滤此参数，API 只提交需要修改的参数列表 ✅
2、选择 MOD 类型的命令时，在输入框的提示语，根据每个参数给出更加准确的提示，比如；数字组成的字符串，范围：001-999；结合 TR069协议对每个参数进行合理的提示文案，并完成对应的初始化数据 ✅

---
## 实施记录（2026-04-27）

### 需求 1：MOD 提交时过滤未填参数

实现位置：[omcmb/webcode/src/pages/mml/Console/hooks/useCommandExecution.ts](omcmb/webcode/src/pages/mml/Console/hooks/useCommandExecution.ts) `buildExecutePayload` 函数。

改动：在 `activeTab === 'control'` 分支判断当前命令是否为编辑类（MOD/ADD），若是则跳过 `parameters[code]` 为 `undefined`/`null`/`''` 的字段，**仅传用户实际填写的参数**。`0` / `false` 是合法值，保留。

LST/DSP/RMV 等非编辑类操作维持原行为（仅勾选语义，value 占 ""）。

### 需求 2：智能 placeholder + 默认值提示

#### 后端：MMLParamRef 暴露 `default_value` / `js_regex`

之前 SQL 只 SELECT 7 列，placeholder/默认值所需的两个字段没下发。

- [omcgo/internal/mml/model.go:210-220](omcgo/internal/mml/model.go#L210-L220)：`MMLParamRef` 加 `DefaultValue` / `JsRegex` 字段，`omitempty` 不影响兼容
- [omcgo/internal/mml/pg_repository.go:1617-1700](omcgo/internal/mml/pg_repository.go#L1617-L1700)：`ListByCommandID` / `ListByCommandIDs` 的 SELECT 加 `COALESCE(p.default_value,'')` / `COALESCE(p.js_regex,'')`，Scan 同步扩展

#### 前端：placeholder 生成器

新文件 [omcmb/webcode/src/pages/mml/Console/utils/paramHints.ts](omcmb/webcode/src/pages/mml/Console/utils/paramHints.ts) 提供 `buildParamPlaceholder(ref, t)`：

| 优先级 | 数据来源 | 输出示例 |
|------|---------|---------|
| 1 | `valueConstraint.raw`（人类可读串）+ 自动展开 `strint-[001-999]` / `unsignedInt[0:256]` | `数字字符串，范围 001-999` / `整数，范围 0-256` |
| 2 | `valueConstraint.desc` / `description` | 原文 |
| 3 | 按 `valueType` + `min/max` 拼装 | `整数，范围 1-15` / `字符串，长度 0-16` / `字符串，长度 ≤ 64` |
| 4 | `jsRegex`（去 `/` 包裹符） | `^[0-9]{3}$` |
| 5 | 通用 fallback | `请输入参数值` |

`defaultValue` 不为空时附加 `（默认: X）`，例如 TimerNetT3212 显示 `整数，范围 1-2147483647（默认: 10）`。

挂载点：[ParamFormRenderer.tsx](omcmb/webcode/src/pages/mml/Console/components/ParamFormRenderer.tsx) `renderParamRefControl` 用 `buildParamPlaceholder(ref, t)` 替换原写死的 `t('mml.console.inputValue')`。

#### i18n

[omcmb/frontend-core/src/i18n/{zh-CN,en-US}/index.ts](omcmb/frontend-core/src/i18n/zh-CN/index.ts) 各加 8 条 `mml.console.hint.*` key，覆盖：
- `numericStringRange` / `integerRange` / `integerMin` / `integerMax`
- `stringFixedLength` / `stringLengthRange` / `stringMaxLength`
- `defaultSuffix`

### 7 个 MOD DEVICE_INFO 参数实际显示效果

| 参数 | 展开后 placeholder |
|------|-----------------|
| MCC | `数字字符串，范围 001-999` |
| MNC | `数字字符串，范围 00-999` |
| BtsNum | `整数，范围 0-256` |
| Encryption | `整数，范围 0-1` |
| TimerNetT3212 | `整数，范围 1-2147483647（默认: 10）` |
| NriBitLen | `整数，范围 1-15` |
| NriNullAdd | `字符串，长度 0-16` |

### 验证

- 前端 `tsc --noEmit` 通过
- 后端 `go build ./...` + `go test ./internal/mml/...` 通过
- 浏览器侧需手工抽测：在 MOD DEVICE_INFO 命令面板下，只填一个参数提交，观察请求 body `commands[0].parameters` 是否只含填写的字段（之前会带 7 个空 ""）。
