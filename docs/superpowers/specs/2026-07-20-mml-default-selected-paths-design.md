# MML 控制台 Path 默认勾选联动设计

> 日期：2026-07-20
> 状态：已确认，待实现
> 范围：MML 配置 Path 属性与 `/mml/console` 命令选择页面

## 1. 背景

MML 配置的“编辑 Path 属性”页面已经支持维护 `default_selected`，后端
`GET /api/v1/mml/commands/:id/sub-fields` 和自定义命令 Path 接口也已经返回该字段。
但是 MML 控制台把接口数据转换为 `CommandParamPath` 时没有保留
`defaultSelected`，命令选择页面仍按旧规则将所有 Path 初始化为未选中。

因此，配置页面中的“默认勾选”目前只完成了存储和展示，没有影响控制台的实际选择行为。

## 2. 已确认需求

1. MML 配置 Path 的“默认勾选”是控制台 Path 初始选择状态的唯一配置来源。
2. 每次打开命令选择页面，都重新按当前命令的默认勾选配置恢复，不保留上一次控制台中的手工选择。
3. 在同一次打开过程中切换命令时，按新命令的默认勾选配置重新初始化。
4. 用户在当前弹框内可以手工勾选或取消；这些临时操作仅在当前弹框打开期间有效。
5. 如果不希望某个 Path 默认选中，必须在 MML 配置中关闭该 Path 的“默认勾选”。
6. 查询命令只默认选中当前设备、产品和运行时过滤后仍可见的 Path。
7. MOD 命令只默认选中可写 Path；即使只读 Path 配置了默认勾选，也不能进入 MOD 选择结果。
8. 没有任何默认勾选 Path 时保持全部不选，用户必须手工选择后才能确认。
9. 标准命令和自定义命令统一遵循 `default_selected` 语义。
10. ADD、RMV 等不使用 Path 复选框的命令保持现有行为。
11. 不改变后续配置参数页、执行请求、任务和结果展示协议。

## 3. 方案选择

### 3.1 采用方案：复用现有字段并补齐前端初始化与缓存失效

沿用现有 `default_selected` 数据库字段和接口字段，在前端 adapter 层将其归一到
`CommandParamPath.defaultSelected`。命令选择页面在每次打开和切换命令时，从当前可选
Path 中派生默认选中 key。

配置保存后，同时失效控制台 sub-fields 查询缓存，确保用户返回控制台后读取到最新配置。

该方案的优点：

- 不新增数据库字段或后端接口。
- MML 配置继续作为唯一真值源。
- 默认选择发生在产品能力、读写权限和运行时不支持 Path 过滤之后，不会选中不可执行项。
- 执行阶段继续使用现有 `selectedPathKeys` 和 `checkedPaths`。

### 3.2 未采用方案

- **后端新增默认 Path 列表**：会重复现有 `default_selected` 字段，并把前端可见性过滤规则
  复制到后端新结构中。
- **每次打开都绕过缓存强制请求**：可以取得最新数据，但会增加不必要的请求；配置写操作
  正确失效控制台缓存后，正常 React Query 拉取即可保证同一系统内的更新立即生效。
- **在控制台持久化用户最后一次选择**：与“每次按配置恢复”的已确认规则冲突。

## 4. 数据流

### 4.1 标准命令

```text
mml_command_sub_fields.default_selected
  → ConsoleService.GetCommandSubFields
  → BackendSubField.default_selected
  → SubFieldDef.defaultSelected
  → subFieldsToParamPaths
  → CommandParamPath.defaultSelected
  → 当前命令可选 Path 过滤
  → 默认选中 Path keys
  → CommandPathSelector
```

现有后端响应和 `SubFieldDef` 已包含该字段，本次只补齐
`SubFieldDef → CommandParamPath` 的映射和选择初始化。

### 4.2 自定义命令

```text
mml_custom_command_paths.default_selected
  → MMLCustomCommandPathDef.defaultSelected
  → customCommandPathDefsToParamPaths
  → CommandParamPath.defaultSelected
  → 默认选中 Path keys
```

自定义命令仍先与产品过滤后的 `paramPaths` 取交集，再计算默认选择，避免关联表中的
产品不支持 Path 回流。

### 4.3 配置保存后的缓存

标准 Path 编辑当前只失效：

- `['mml', 'console', 'group-tree']`
- MML admin 列表缓存

本次增加控制台 sub-fields 前缀：

```text
['mml', 'console', 'sub-fields']
```

新增、批量新增、修改和删除 Path 后均失效该前缀。自定义命令 Path 写操作继续使用现有
`MML_CUSTOM_COMMAND_PATHS_QUERY_KEY` 失效机制。

## 5. 控制台交互规则

### 5.1 打开命令选择页面

- 如果父页面已经确认过一个命令，仍回填该命令为左侧选中项。
- 不回填父页面保存的 `selectedPathKeys`。
- 参数 Path 加载完成后，从当前候选中选择 `defaultSelected === true` 的 Path。
- 上次手工新增的选择被丢弃，上次手工取消的默认项重新选中。

### 5.2 切换命令

- 清空前一个命令的弹框草稿选择。
- 新命令 Path 加载完成后，按新命令的默认配置初始化。
- 用户切换离开后再切回同一命令，也视为一次重新选择，重新应用默认配置。

### 5.3 当前弹框内手工调整

- 默认初始化只执行一次，不得因组件重渲染、计数变化或查询状态变化反复覆盖用户操作。
- 用户勾选、取消、全选或取消全选后，当前弹框内以用户草稿为准。
- 候选 Path 因设备能力或不支持 Path 过滤发生变化时，继续清理已不可选 key。
- 关闭弹框后丢弃草稿；下次打开重新按配置初始化。

### 5.4 操作类型

| 操作类型 | 默认勾选候选 |
| --- | --- |
| `LST`、`DSP` 和其他查询类型 | 当前过滤后的全部可执行 Path 中 `defaultSelected=true` 的项 |
| `MOD` | 当前过滤后的可写 Path 中 `defaultSelected=true` 的项 |
| `ADD`、`RMV` | 不使用 Path 复选框，保持目标对象流程 |

默认选中数量为零时，沿用现有校验：确定按钮禁用，直到用户至少选择一个 Path。

## 6. 前端模型与组件边界

### `CommandParamPath`

增加兼容字段：

```ts
interface CommandParamPath {
  defaultSelected?: boolean;
}
```

字段缺失按 `false` 处理，保证旧测试数据、兼容数据和没有该元数据的调用方继续保持默认不选。

### Adapter

- `subFieldsToParamPaths` 映射标准命令 `defaultSelected`。
- `customCommandPathDefsToParamPaths` 映射自定义命令 `defaultSelected`。
- 只负责数据归一，不维护交互状态。

### Path 选择纯函数

增加纯函数，从已经完成操作类型和产品能力过滤的候选 Path 中按原顺序返回默认 key：

```ts
getDefaultSelectedPathKeys(paths): string[]
```

该函数只接受当前候选，不自行判断操作类型，确保“先过滤、后默认选择”的顺序明确。

### `CommandSelectModal`

- 区分“当前打开会话和命令尚未初始化默认值”与“用户已经手工修改草稿”。
- 打开弹框或切换命令时重置初始化标记。
- 在异步 Path 数据可用后初始化一次默认 key。
- 当前会话中后续重渲染只清理无效 key，不重复覆盖用户选择。
- 确认回调和父页面状态结构保持不变。

### MML admin hooks

标准命令 Path 写操作的统一 `invalidateAfterWrite` 增加 console sub-fields 缓存失效，不在
各 mutation 中重复维护。

## 7. 错误与边界处理

- Path 接口加载失败时沿用现有错误和空列表行为，不产生虚假的默认选择。
- `defaultSelected=true` 但 Path 被产品能力过滤时，不展示也不选中。
- `defaultSelected=true` 但 Path 对 MOD 不可写时，不展示也不选中。
- 重复 Path 继续由现有接口和候选顺序处理；默认 key 使用 Path 字符串，与当前选择模型一致。
- 所有 Path 均为默认关闭时，页面显示已选 `0` 项并禁用确认，不自动回退为全选。
- 本次不新增用户可见文案，不需要新增国际化键。

## 8. 测试设计

按 TDD 先补失败测试，再实现。

### 8.1 纯函数和 adapter

1. 标准 sub-field 的 `defaultSelected` 映射到 `CommandParamPath`。
2. 自定义命令 Path 的 `defaultSelected` 映射到 `CommandParamPath`。
3. 默认 key 保持当前候选 Path 顺序。
4. `defaultSelected=false` 或字段缺失的 Path 不进入默认 key。

### 8.2 `CommandSelectModal`

1. 首次选择查询命令时默认选中配置开启的 Path。
2. 查询命令不会选中已被产品或运行时规则过滤的默认 Path。
3. MOD 只默认选中可写且配置开启的 Path。
4. 没有默认 Path 时保持全部不选并禁用确定按钮。
5. 用户在当前弹框取消默认项后，后续重渲染不会重新勾选。
6. 关闭再打开同一命令后，重新恢复配置默认值，不回填父页面旧选择。
7. 切换命令时按新命令默认值初始化。
8. 切换离开再切回时再次恢复该命令默认值。
9. 全选和取消全选行为保持不变。
10. ADD、RMV 目标对象流程保持不变。
11. 自定义查询和自定义 MOD 使用相同默认选择规则。

### 8.3 缓存失效

1. 标准 Path 新增后失效 console sub-fields。
2. 标准 Path 修改默认勾选后失效 console sub-fields。
3. 标准 Path 删除和批量新增后失效 console sub-fields。
4. 既有 group-tree 和 admin 列表失效行为保持不变。

### 8.4 验证命令

```bash
cd omcmb
npm test --workspace webcode -- src/pages/mml/Console
npm run typecheck
git diff --check
```

开始实现前的基线：

- MML 控制台测试：10 个测试文件、98 个测试全部通过。
- `npm run typecheck` 通过。

## 9. 浏览器验收

在 113 自测环境验证，不执行真实查询或修改命令：

1. 在 MML 配置中选取一个包含多个 Path 的查询命令。
2. 开启其中两个 Path 的“默认勾选”，关闭另一个 Path，保存。
3. 进入 MML 控制台并选择设备、打开该命令，确认只默认选中前两个 Path。
4. 手工取消一个默认项并关闭命令选择页面。
5. 再次打开，确认两个配置默认项均重新选中。
6. 返回 MML 配置关闭其中一个默认项，保存后再次进入控制台，确认该 Path 不再默认选中。
7. 对 MOD 命令重复验证，确认只读 Path 不显示，只有配置开启的可写 Path 默认选中。
8. 验证无默认项时确定按钮禁用，手工选择后恢复可用。
9. 记录关键 DOM、复选框状态、控制台错误和页面错误。

## 10. 非目标

- 不修改 `default_selected` 的数据库默认值或历史数据。
- 不新增后端 API、migration 或执行请求字段。
- 不保存控制台用户的临时 Path 选择。
- 不改变配置参数确认页、实例选择、参数值校验或执行模式。
- 不改变设备、产品、参数模型和运行时不支持 Path 的过滤规则。
- 不在浏览器验收中向真实设备下发命令。

## 11. 验收标准

- 配置为默认勾选的可执行 Path 在每次打开控制台命令选择页时自动选中。
- 用户关闭再打开或切换命令后，选择状态重新以配置为准。
- 在 MML 配置中关闭默认勾选后，下一次进入控制台不再默认选择该 Path。
- 查询、MOD、标准命令和自定义命令遵守相同规则，并保持各自现有可见性限制。
- 无默认项时不自动选中任何 Path。
- 相关组件测试、缓存失效测试和前端类型检查全部通过。
