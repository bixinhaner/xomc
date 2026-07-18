# MML MOD Parameter Type and Range Validation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 MML 控制台 MOD 配置页展示标准参数树数据类型，并仅在标准参数树配置了最小值或最大值时即时校验输入范围。

**Architecture:** 后端现有内置命令 sub-fields 和自定义命令 paths 接口负责透传 `standard_params.data_type/min_value/max_value`；`frontend-core` 将两类响应归一到 Console 的 `CommandParamPath`。独立纯函数按标准参数树既有语义校验字符串长度或整数范围，`ConfigParamsModal` 只负责展示类型、错误和执行按钮状态。

**Tech Stack:** Go 1.x、pgx/PostgreSQL、React 19、TypeScript 6、Ant Design 6、TanStack Query、Vitest/Testing Library。

## Global Constraints

- 所有 MML 参数以 `standard_params` 为唯一类型和范围真值源，不在前端推断范围。
- 仅 MOD“命令参数”页新增类型展示和范围校验；查询、ADD/RMV、裸路径页行为不变。
- `min_value` 存在时继续作为 MOD 输入框默认值。
- `min_value` 和 `max_value` 都为空时，除现有必填规则外不做类型或范围校验。
- `string` 按 Unicode 字符数量校验；数值类型按十进制整数闭区间校验。
- `boolean`、`dateTime` 不做范围校验。
- 不新增 migration，不修改执行请求、任务结构或 TR-069 下发格式。
- 所有新增用户文案同时维护中文和英文。
- 开始实现前基线：前端 typecheck 和 `go test ./internal/mml/...` 通过；完整 Go 测试已有 `internal/core/carrier`、`internal/core/dictloader` 两项无关失败。

---

## File Structure

### Backend

- `omcgo/internal/mml/sub_field_model.go`：内置命令富化子字段增加 `MaxValue`。
- `omcgo/internal/mml/admin_repository.go`：从 `standard_params` 查询并扫描 `max_value`。
- `omcgo/internal/mml/console_service.go`：sub-fields DTO 透传最大值。
- `omcgo/internal/mml/custom_command_path_model.go`：自定义命令 Path 视图增加最小值、最大值。
- `omcgo/internal/mml/custom_command_path_repository.go`：自定义 Path 查询并扫描范围。
- `omcgo/internal/mml/admin_repository_test.go`：集成验证内置命令范围元数据。
- `omcgo/internal/mml/console_service_test.go`：验证 DTO 透传。
- `omcgo/internal/mml/custom_command_path_service_test.go`：验证自定义 Path 富化结果不丢范围。

### Frontend shared layer

- `omcmb/frontend-core/src/types/mmlConsole.ts`：sub-field wire/domain 类型增加 `maxValue`。
- `omcmb/frontend-core/src/types/mml.ts`：定义自定义命令富化 Path 类型。
- `omcmb/frontend-core/src/services/api/mmlApi.ts`：映射 sub-field 最大值并新增自定义 Path 查询。
- `omcmb/frontend-core/src/services/api/__tests__/mmlConsoleApi.test.ts`：验证两个接口的字段映射。
- `omcmb/frontend-core/src/hooks/api/useMmlConsole.ts`：新增按自定义命令 ID 加载富化 Path 的 query hook。
- `omcmb/frontend-core/src/i18n/zh-CN/index.ts`、`en-US/index.ts`：新增逐字段校验文案。

### Frontend Console layer

- `omcmb/webcode/src/pages/mml/Console/types.ts`：`CommandParamPath` 增加 `maxValue`。
- `omcmb/webcode/src/pages/mml/Console/adapters.ts`：内置和自定义 Path 归一化。
- `omcmb/webcode/src/pages/mml/Console/components/CommandSelectModal.tsx`：选中自定义命令时加载富化 Path。
- `omcmb/webcode/src/pages/mml/Console/components/CommandSelectModal.test.tsx`：验证自定义命令透传类型和范围。
- `omcmb/webcode/src/pages/mml/Console/modParamValidation.ts`：纯范围校验函数。
- `omcmb/webcode/src/pages/mml/Console/modParamValidation.test.ts`：范围语义单测。
- `omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.tsx`：类型标签、逐字段错误和执行禁用。
- `omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.test.tsx`：交互回归测试。

---

### Task 1: 后端透传标准参数树范围

**Files:**

- Modify: `omcgo/internal/mml/sub_field_model.go`
- Modify: `omcgo/internal/mml/admin_repository.go`
- Modify: `omcgo/internal/mml/console_service.go`
- Modify: `omcgo/internal/mml/custom_command_path_model.go`
- Modify: `omcgo/internal/mml/custom_command_path_repository.go`
- Test: `omcgo/internal/mml/admin_repository_test.go`
- Test: `omcgo/internal/mml/console_service_test.go`
- Test: `omcgo/internal/mml/custom_command_path_service_test.go`

**Interfaces:**

- Produces: `MMLCommandSubFieldEnriched.MaxValue *int64`
- Produces: `SubFieldDTO.MaxValue *int64`
- Produces: `MMLCustomCommandPathView.MinValue *int64`
- Produces: `MMLCustomCommandPathView.MaxValue *int64`

- [ ] **Step 1: 写内置命令范围透传失败测试**

在 `console_service_test.go` 增加：

```go
func TestGetCommandSubFields_PropagatesStandardRange(t *testing.T) {
	cmdID := uuid.New()
	minValue, maxValue := int64(2), int64(32)
	sfRepo := newFakeSubFieldRepo()
	sfRepo.byCommandEnriched[cmdID] = []MMLCommandSubFieldEnriched{{
		MMLCommandSubField: MMLCommandSubField{
			ID: uuid.New(), CommandID: cmdID, MMLCode: "NAME",
			LabelI18n: map[string]string{"zh-CN": "名称"},
		},
		Tr069Path: "Device.Info.Name",
		ValueType: "string",
		AccessType: "READ_WRITE",
		MinValue: &minValue,
		MaxValue: &maxValue,
	}}

	svc := NewConsoleService(&fakeGroupTreeRepo{}, sfRepo, newFakeCommandRepo(), nil)
	got, err := svc.GetCommandSubFields(context.Background(), cmdID, "", "", "zh-CN")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.NotNil(t, got[0].MinValue)
	require.NotNil(t, got[0].MaxValue)
	assert.EqualValues(t, 2, *got[0].MinValue)
	assert.EqualValues(t, 32, *got[0].MaxValue)
}
```

在 `admin_repository_test.go` 增加 PostgreSQL 可用时运行的集成测试：创建 sub-field 后
更新对应 `standard_params.data_type/min_value/max_value`，调用
`ListEnrichedByCommand` 并断言三个字段。

- [ ] **Step 2: 运行测试并确认 RED**

Run:

```bash
cd omcgo
go test ./internal/mml -run 'TestGetCommandSubFields_PropagatesStandardRange|Test_ListEnrichedByCommand_ReturnsStandardRange' -count=1
```

Expected: 编译失败，提示 `MaxValue undefined`，证明测试命中缺失字段。

- [ ] **Step 3: 最小实现内置命令最大值链路**

在 `MMLCommandSubFieldEnriched` 和 `SubFieldDTO` 中增加：

```go
MaxValue *int64 `json:"max_value,omitempty"`
```

把 `ListEnrichedByCommand` SELECT 尾部范围调整为：

```sql
sp.min_value AS min_value,
sp.max_value AS max_value,
```

扫描时按相同顺序加入：

```go
&e.MinValue, &e.MaxValue,
```

构造 DTO 时加入：

```go
MinValue: e.MinValue,
MaxValue: e.MaxValue,
```

- [ ] **Step 4: 写自定义命令 Path 范围失败测试**

在 `custom_command_path_service_test.go` 增加服务透传测试：

```go
func TestService_ListCustomCommandPaths_PreservesStandardRange(t *testing.T) {
	commandID := uuid.New()
	minValue, maxValue := int64(1), int64(13)
	want := []MMLCustomCommandPathView{{
		CommandID: commandID,
		StandardPath: "Device.Radio.Channel",
		DataType: "unsignedInt",
		MinValue: &minValue,
		MaxValue: &maxValue,
	}}
	svc := newCRUDServiceWithRepo(&mockCustomCommandRepo{})
	svc.SetCustomCommandPathRepo(&mockCustomCommandPathRepo{
		listFn: func(_ context.Context, gotID uuid.UUID) ([]MMLCustomCommandPathView, error) {
			require.Equal(t, commandID, gotID)
			return want, nil
		},
	})

	got, err := svc.ListCustomCommandPaths(context.Background(), commandID)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
```

同时在测试中读取 `custom_command_path_repository.go`，断言 SELECT 同时包含
`sp.min_value`、`sp.max_value`，防止 repository 漏查。

- [ ] **Step 5: 运行测试并确认 RED**

Run:

```bash
cd omcgo
go test ./internal/mml -run 'TestService_ListCustomCommandPaths_PreservesStandardRange|Test_CustomCommandPathRepository_SelectsStandardRange' -count=1
```

Expected: 编译失败，提示 `MinValue` 或 `MaxValue` 不存在。

- [ ] **Step 6: 最小实现自定义命令 Path 范围**

在 `MMLCustomCommandPathView` 增加：

```go
MinValue *int64 `json:"min_value,omitempty"`
MaxValue *int64 `json:"max_value,omitempty"`
```

repository SELECT 加入：

```sql
sp.min_value, sp.max_value,
```

扫描顺序加入：

```go
&v.MinValue, &v.MaxValue,
```

- [ ] **Step 7: 运行后端目标测试**

Run:

```bash
cd omcgo
gofmt -w internal/mml/sub_field_model.go internal/mml/admin_repository.go internal/mml/console_service.go internal/mml/custom_command_path_model.go internal/mml/custom_command_path_repository.go internal/mml/admin_repository_test.go internal/mml/console_service_test.go internal/mml/custom_command_path_service_test.go
go test ./internal/mml/... -count=1
```

Expected: `ok github.com/omcgo/omcgo/internal/mml`，无新增失败。

- [ ] **Step 8: 提交后端元数据改动**

```bash
git add omcgo/internal/mml/sub_field_model.go \
  omcgo/internal/mml/admin_repository.go \
  omcgo/internal/mml/console_service.go \
  omcgo/internal/mml/custom_command_path_model.go \
  omcgo/internal/mml/custom_command_path_repository.go \
  omcgo/internal/mml/admin_repository_test.go \
  omcgo/internal/mml/console_service_test.go \
  omcgo/internal/mml/custom_command_path_service_test.go
git commit -m "feat(mml): 透传标准参数范围元数据"
```

---

### Task 2: 前端加载并归一内置与自定义 Path 元数据

**Files:**

- Modify: `omcmb/frontend-core/src/types/mmlConsole.ts`
- Modify: `omcmb/frontend-core/src/types/mml.ts`
- Modify: `omcmb/frontend-core/src/services/api/mmlApi.ts`
- Modify: `omcmb/frontend-core/src/hooks/api/useMmlConsole.ts`
- Test: `omcmb/frontend-core/src/services/api/__tests__/mmlConsoleApi.test.ts`
- Modify: `omcmb/webcode/src/pages/mml/Console/types.ts`
- Modify: `omcmb/webcode/src/pages/mml/Console/adapters.ts`
- Modify: `omcmb/webcode/src/pages/mml/Console/components/CommandSelectModal.tsx`
- Test: `omcmb/webcode/src/pages/mml/Console/components/CommandSelectModal.test.tsx`

**Interfaces:**

- Consumes: 后端 `max_value`、自定义命令 `GET /mml/templates/:id/paths`
- Produces: `MMLCustomCommandPathDef`
- Produces: `mmlApi.getTemplatePaths(commandId): Promise<MMLCustomCommandPathDef[]>`
- Produces: `useCustomCommandPaths(commandId)`
- Produces: `CommandParamPath.maxValue?: number`

- [ ] **Step 1: 写 API 映射失败测试**

在 `mmlConsoleApi.test.ts` 增加：

```ts
it('maps standard min and max values from command sub-fields', async () => {
  getMock.mockResolvedValue({ data: { sub_fields: [{
    id: 'sf-1',
    command_id: 'command-1',
    param_id: 'param-1',
    mml_code: 'CHANNEL',
    label: 'Channel',
    label_i18n: {},
    tr069_path: 'Device.Radio.Channel',
    value_type: 'unsignedInt',
    access_type: 'READ_WRITE',
    is_object: false,
    supports_add: false,
    supports_delete: false,
    change_applies: 'Immediate',
    constraint_text: '[1, 13]',
    constraint_text_i18n: {},
    min_value: 1,
    max_value: 13,
    default_selected: false,
    is_required: false,
    sort_order: 1,
  }] } });

  const result = await mmlApi.getCommandSubFields('command-1');

  expect(result[0]).toMatchObject({
    valueType: 'unsignedInt',
    minValue: 1,
    maxValue: 13,
  });
});

it('loads and maps enriched custom command paths', async () => {
  getMock.mockResolvedValue({ data: { items: [{
    id: 'path-1',
    command_id: 'custom-1',
    standard_path_id: 'standard-1',
    standard_path: 'Device.Info.Name',
    entry_type: 'parameter',
    access: 'readWrite',
    data_type: 'string',
    description: 'Name',
    min_value: 2,
    max_value: 32,
    default_selected: false,
    sort_order: 1,
  }] } });

  const result = await mmlApi.getTemplatePaths('custom-1');

  expect(getMock).toHaveBeenCalledWith('/mml/templates/custom-1/paths');
  expect(result[0]).toMatchObject({
    standardPath: 'Device.Info.Name',
    dataType: 'string',
    minValue: 2,
    maxValue: 32,
  });
});
```

- [ ] **Step 2: 运行 API 测试并确认 RED**

Run:

```bash
cd omcmb
npm test --workspace webcode -- --run ../frontend-core/src/services/api/__tests__/mmlConsoleApi.test.ts
```

Expected: 第一条断言缺少 `maxValue`，第二条提示 `getTemplatePaths is not a function`。

- [ ] **Step 3: 实现 shared types、API 和 hook**

`BackendSubField`、`SubFieldDef` 分别增加：

```ts
max_value?: number | null;
maxValue?: number;
```

在 `mml.ts` 定义：

```ts
export interface MMLCustomCommandPathDef {
  id: string;
  commandId: string;
  standardPathId: string;
  standardPath: string;
  entryType: string;
  access: string;
  dataType: string;
  description: string;
  minValue?: number;
  maxValue?: number;
  defaultSelected: boolean;
  sortOrder: number;
}
```

`mapSubField` 加入：

```ts
maxValue: s.max_value ?? undefined,
```

新增 `BackendMMLCustomCommandPath`、映射函数和 API：

```ts
async getTemplatePaths(commandId: string): Promise<MMLCustomCommandPathDef[]> {
  const { data } = await http.get<{ items: BackendMMLCustomCommandPath[] }>(
    `/mml/templates/${commandId}/paths`,
  );
  return (data.items ?? []).map(mapBackendCustomCommandPath);
},
```

在 `useMmlConsole.ts` 新增：

```ts
export function useCustomCommandPaths(commandId?: string) {
  return useQuery({
    queryKey: ['mml', 'console', 'custom-command-paths', commandId ?? ''],
    queryFn: () => mmlApi.getTemplatePaths(commandId!),
    staleTime: 30 * 60 * 1000,
    enabled: Boolean(commandId),
  });
}
```

- [ ] **Step 4: 写 Console 自定义 Path 归一失败测试**

扩展 `CommandSelectModal.test.tsx` 的 hook mock，提供 `useCustomCommandPaths`。增加测试：

```tsx
it('confirms enriched type and range metadata for a custom MOD command', async () => {
  fixtures.customCommands = [{
    id: 'custom-1',
    commandName: '修改名称',
    commandCode: 'MOD CUSTOM',
    operationType: 'MOD',
    commandScope: 'public',
    categoryGroup: '',
    parameters: {},
    paramPaths: ['Device.Info.Name'],
    description: '',
    creator: 'admin',
    createdAt: '',
    updatedAt: '',
  }];
  fixtures.customPaths = [{
    id: 'path-1',
    commandId: 'custom-1',
    standardPathId: 'standard-1',
    standardPath: 'Device.Info.Name',
    entryType: 'parameter',
    access: 'readWrite',
    dataType: 'string',
    description: 'Name',
    minValue: 2,
    maxValue: 32,
    defaultSelected: false,
    sortOrder: 1,
  }];
  const onConfirm = vi.fn();
  renderModal({ onConfirm });

  // 展开 Customized 和 PublicTemplate，选择自定义命令，再勾选 Path。
  fireEvent.click(await screen.findByText('修改名称'));
  fireEvent.click(await screen.findByRole('checkbox', { name: /Name/ }));
  fireEvent.click(screen.getByRole('button', { name: 'mml.consoleV2.cmdSelect.okText' }));

  expect(onConfirm).toHaveBeenCalledWith(
    expect.objectContaining({
      isCustom: true,
      paramPaths: [expect.objectContaining({
        path: 'Device.Info.Name',
        valueType: 'string',
        minValue: 2,
        maxValue: 32,
      })],
    }),
    ['Device.Info.Name'],
  );
});
```

测试中依次点击 `document.querySelectorAll('.ant-tree-switcher')` 返回的折叠开关，直到
“修改名称”叶子可见，再执行上述选择和断言；不要通过直接调用 adapter 绕过组件行为。

- [ ] **Step 5: 运行组件测试并确认 RED**

Run:

```bash
cd omcmb
npm test --workspace webcode -- --run src/pages/mml/Console/components/CommandSelectModal.test.tsx
```

Expected: 自定义 Path 缺少 `valueType/minValue/maxValue` 或 hook 未定义。

- [ ] **Step 6: 实现 Console 归一化和自定义 Path 加载**

`CommandParamPath` 增加：

```ts
maxValue?: number;
```

`subFieldsToParamPaths` 加入：

```ts
maxValue: sf.maxValue,
```

新增 adapter：

```ts
export function customCommandPathDefsToParamPaths(
  paths: MMLCustomCommandPathDef[],
): CommandParamPath[] {
  return paths.map((p) => ({
    path: p.standardPath,
    label: p.description || p.standardPath.split('.').filter(Boolean).pop() || p.standardPath,
    writable: p.access.replace(/[_-]/g, '').toLowerCase() === 'readwrite',
    isObject: p.entryType === 'object',
    valueType: p.dataType,
    minValue: p.minValue,
    maxValue: p.maxValue,
    description: p.description,
  }));
}
```

`CommandSelectModal` 在 `selectedCustomId` 有值时调用 `useCustomCommandPaths`，自定义命令
的 `paramPaths`、`pathsLoading` 和确认按钮都使用富化结果；模板列表中的字符串
`paramPaths` 仅继续用于未选中命令的可见性预过滤。

- [ ] **Step 7: 运行前端数据链路测试**

Run:

```bash
cd omcmb
npm test --workspace webcode -- --run \
  ../frontend-core/src/services/api/__tests__/mmlConsoleApi.test.ts \
  src/pages/mml/Console/components/CommandSelectModal.test.tsx
npm run typecheck
```

Expected: 两个 Vitest 文件及 typecheck 全部通过。

- [ ] **Step 8: 提交前端元数据链路**

```bash
git add omcmb/frontend-core/src/types/mmlConsole.ts \
  omcmb/frontend-core/src/types/mml.ts \
  omcmb/frontend-core/src/services/api/mmlApi.ts \
  omcmb/frontend-core/src/services/api/__tests__/mmlConsoleApi.test.ts \
  omcmb/frontend-core/src/hooks/api/useMmlConsole.ts \
  omcmb/webcode/src/pages/mml/Console/types.ts \
  omcmb/webcode/src/pages/mml/Console/adapters.ts \
  omcmb/webcode/src/pages/mml/Console/components/CommandSelectModal.tsx \
  omcmb/webcode/src/pages/mml/Console/components/CommandSelectModal.test.tsx
git commit -m "feat(mml): 加载参数类型与范围元数据"
```

---

### Task 3: 实现 MOD 纯范围校验

**Files:**

- Create: `omcmb/webcode/src/pages/mml/Console/modParamValidation.ts`
- Create: `omcmb/webcode/src/pages/mml/Console/modParamValidation.test.ts`

**Interfaces:**

- Consumes: `CommandParamPath`
- Consumes: `dataTypeRangeKind(valueType)`
- Produces: `validateModParamValue(path, value): ModParamValidationError | null`
- Produces: `getModParamValidationErrors(paths, values): Record<string, ModParamValidationError>`

- [ ] **Step 1: 写完整规则的失败测试**

创建 `modParamValidation.test.ts`，至少包含：

```ts
describe('validateModParamValue', () => {
  it('skips type and range checks when both bounds are missing', () => {
    expect(validateModParamValue(param({ valueType: 'unsignedInt' }), 'not-a-number')).toBeNull();
  });

  it.each([
    ['a', 'minLength', 2, 4],
    ['abcde', 'maxLength', 2, 4],
  ])('validates string Unicode length for %s', (value, code, minValue, maxValue) => {
    expect(validateModParamValue(
      param({ valueType: 'string', minValue, maxValue }),
      value,
    )).toEqual(expect.objectContaining({ code }));
  });

  it('counts an emoji as one Unicode character', () => {
    expect(validateModParamValue(
      param({ valueType: 'string', minValue: 1, maxValue: 1 }),
      '😀',
    )).toBeNull();
  });

  it('accepts inclusive integer boundaries', () => {
    const path = param({ valueType: 'unsignedInt', minValue: 1, maxValue: 13 });
    expect(validateModParamValue(path, '1')).toBeNull();
    expect(validateModParamValue(path, '13')).toBeNull();
  });

  it.each([
    ['x', 'integer'],
    ['0', 'minValue'],
    ['14', 'maxValue'],
  ])('rejects invalid ranged integer %s', (value, code) => {
    expect(validateModParamValue(
      param({ valueType: 'unsignedInt', minValue: 1, maxValue: 13 }),
      value,
    )).toEqual(expect.objectContaining({ code }));
  });

  it('does not range-check boolean and dateTime', () => {
    expect(validateModParamValue(
      param({ valueType: 'boolean', minValue: 1, maxValue: 1 }),
      'anything',
    )).toBeNull();
  });
});
```

另测空白值返回 `required`，单边范围只检查已配置的一侧，以及聚合函数按 Path 返回错误。

- [ ] **Step 2: 运行测试并确认 RED**

Run:

```bash
cd omcmb
npm test --workspace webcode -- --run src/pages/mml/Console/modParamValidation.test.ts
```

Expected: 模块不存在或导出函数不存在。

- [ ] **Step 3: 最小实现纯校验函数**

创建以下公开类型和函数：

```ts
export type ModParamValidationCode =
  | 'required'
  | 'integer'
  | 'minValue'
  | 'maxValue'
  | 'minLength'
  | 'maxLength';

export interface ModParamValidationError {
  code: ModParamValidationCode;
  bound?: number;
}

export function validateModParamValue(
  path: CommandParamPath,
  value: string,
): ModParamValidationError | null {
  if (value.trim() === '') return { code: 'required' };
  if (path.minValue == null && path.maxValue == null) return null;

  const rangeKind = dataTypeRangeKind(path.valueType);
  if (rangeKind === 'none') return null;
  if (rangeKind === 'length') {
    const length = Array.from(value).length;
    if (path.minValue != null && length < path.minValue) {
      return { code: 'minLength', bound: path.minValue };
    }
    if (path.maxValue != null && length > path.maxValue) {
      return { code: 'maxLength', bound: path.maxValue };
    }
    return null;
  }

  const trimmed = value.trim();
  if (!/^[+-]?\d+$/.test(trimmed)) return { code: 'integer' };
  const numericValue = Number(trimmed);
  if (!Number.isSafeInteger(numericValue)) return { code: 'integer' };
  if (path.minValue != null && numericValue < path.minValue) {
    return { code: 'minValue', bound: path.minValue };
  }
  if (path.maxValue != null && numericValue > path.maxValue) {
    return { code: 'maxValue', bound: path.maxValue };
  }
  return null;
}
```

聚合函数遍历已选 Path，只收集非空错误。

- [ ] **Step 4: 运行校验测试并确认 GREEN**

Run:

```bash
cd omcmb
npm test --workspace webcode -- --run src/pages/mml/Console/modParamValidation.test.ts
```

Expected: 全部通过。

- [ ] **Step 5: 提交纯校验规则**

```bash
git add omcmb/webcode/src/pages/mml/Console/modParamValidation.ts \
  omcmb/webcode/src/pages/mml/Console/modParamValidation.test.ts
git commit -m "feat(mml): 增加 MOD 参数范围校验规则"
```

---

### Task 4: 在 MOD 配置页展示类型和逐字段错误

**Files:**

- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- Modify: `omcmb/frontend-core/src/i18n/en-US/index.ts`
- Modify: `omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.tsx`
- Test: `omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.test.tsx`

**Interfaces:**

- Consumes: `getModParamValidationErrors`
- Consumes: `CommandParamPath.valueType/minValue/maxValue`
- Produces: MOD Path 后的数据类型标签、输入框错误状态、国际化错误文案

- [ ] **Step 1: 写类型展示和范围错误的失败组件测试**

在 `ConfigParamsModal.test.tsx` 增加：

```tsx
it('shows the standard data type and keeps minValue as the MOD default', () => {
  renderModal({
    command: {
      ...command,
      operationType: 'MOD',
      paramPaths: [{
        path: 'Device.Info.Name',
        label: 'Name',
        writable: true,
        isObject: false,
        valueType: 'string',
        minValue: 2,
        maxValue: 8,
      }],
    },
    selectedPathKeys: ['Device.Info.Name'],
  });

  expect(screen.getByText('string')).toBeInTheDocument();
  expect(screen.getByRole('textbox')).toHaveValue('2');
});

it('blocks execution and shows a per-field error outside the configured range', () => {
  renderModal({
    command: {
      ...command,
      operationType: 'MOD',
      paramPaths: [{
        path: 'Device.Radio.Channel',
        label: 'Channel',
        writable: true,
        isObject: false,
        valueType: 'unsignedInt',
        minValue: 1,
        maxValue: 13,
      }],
    },
    selectedPathKeys: ['Device.Radio.Channel'],
  });

  const input = screen.getByRole('textbox');
  const execute = screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ });
  fireEvent.change(input, { target: { value: '14' } });

  expect(screen.getByText('mml.consoleV2.config.validation.maxValue')).toBeInTheDocument();
  expect(input).toHaveAttribute('aria-invalid', 'true');
  expect(execute).toBeDisabled();

  fireEvent.change(input, { target: { value: '13' } });
  expect(screen.queryByText('mml.consoleV2.config.validation.maxValue')).not.toBeInTheDocument();
  expect(execute).toBeEnabled();
});
```

另加无范围 `unsignedInt` 非数字输入仍可执行的测试，锁定“没配范围不校验”。

- [ ] **Step 2: 运行组件测试并确认 RED**

Run:

```bash
cd omcmb
npm test --workspace webcode -- --run src/pages/mml/Console/components/ConfigParamsModal.test.tsx
```

Expected: 找不到类型标签和逐字段错误，越界时按钮仍可用。

- [ ] **Step 3: 增加中英文文案**

新增相同 key：

```ts
'mml.consoleV2.config.validation.required'
'mml.consoleV2.config.validation.integer'
'mml.consoleV2.config.validation.minValue'
'mml.consoleV2.config.validation.maxValue'
'mml.consoleV2.config.validation.minLength'
'mml.consoleV2.config.validation.maxLength'
```

中文示例：

```ts
'mml.consoleV2.config.validation.maxValue': '不能大于 {bound}',
'mml.consoleV2.config.validation.maxLength': '长度不能超过 {bound} 个字符',
```

英文示例：

```ts
'mml.consoleV2.config.validation.maxValue': 'Must not be greater than {bound}',
'mml.consoleV2.config.validation.maxLength': 'Must not exceed {bound} characters',
```

- [ ] **Step 4: 最小实现 MOD UI**

在组件中计算：

```ts
const modValidationErrors = useMemo(
  () => command?.operationType === 'MOD'
    ? getModParamValidationErrors(selectedParamPaths, values)
    : {},
  [command?.operationType, selectedParamPaths, values],
);
const modValuesValid =
  command?.operationType !== 'MOD' || Object.keys(modValidationErrors).length === 0;
```

每个 MOD Path 后展示：

```tsx
{command.operationType === 'MOD' && p.valueType && (
  <Tag style={{ marginInlineStart: 6 }}>{p.valueType}</Tag>
)}
```

输入框设置 `status`、`aria-invalid`，并在下方按错误 code 调用 i18n：

```tsx
<Input
  status={error ? 'error' : undefined}
  aria-invalid={Boolean(error)}
  value={values[p.path] ?? ''}
  onChange={(e) => setValues((prev) => ({ ...prev, [p.path]: e.target.value }))}
/>
{error && (
  <Text type="danger" style={{ fontSize: 12 }}>
    {t(`mml.consoleV2.config.validation.${error.code}`, { bound: error.bound ?? '' })}
  </Text>
)}
```

删除重复的 MOD 全局必填提示，逐字段 `required` 已覆盖同一行为。ADD/RMV 分支不调用新校验。

- [ ] **Step 5: 运行前端回归**

Run:

```bash
cd omcmb
npm test --workspace webcode -- --run \
  src/pages/mml/Console/modParamValidation.test.ts \
  src/pages/mml/Console/components/ConfigParamsModal.test.tsx \
  src/pages/mml/Console/components/CommandSelectModal.test.tsx
npm run typecheck
```

Expected: 全部通过，无 React act、console error 或 TypeScript 错误。

- [ ] **Step 6: 提交 MOD 页面交互**

```bash
git add omcmb/frontend-core/src/i18n/zh-CN/index.ts \
  omcmb/frontend-core/src/i18n/en-US/index.ts \
  omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.tsx \
  omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.test.tsx
git commit -m "feat(mml): 展示参数类型并校验取值范围"
```

---

### Task 5: 完整验证与浏览器验收

**Files:**

- Verify only; unexpected fixes must start with a new failing test in the owning task.

**Interfaces:**

- Consumes: Tasks 1–4 的完整实现。
- Produces: 可审查的测试和浏览器验收证据。

- [ ] **Step 1: 运行后端构建和 MML 测试**

```bash
cd omcgo
go build ./...
go test ./internal/mml/... -count=1
```

Expected: 两条命令均成功。

- [ ] **Step 2: 运行前端测试、类型检查和 diff 检查**

```bash
cd omcmb
npm test --workspace webcode -- --run \
  ../frontend-core/src/services/api/__tests__/mmlConsoleApi.test.ts \
  src/pages/mml/Console/modParamValidation.test.ts \
  src/pages/mml/Console/components/CommandSelectModal.test.tsx \
  src/pages/mml/Console/components/ConfigParamsModal.test.tsx
npm run typecheck
cd ..
git diff --check
git status --short --branch
```

Expected: 所有目标测试和 typecheck 通过；`git diff --check` 无输出；工作区只含计划中改动。

- [ ] **Step 3: 运行完整 Go 测试并区分基线**

```bash
cd omcgo
go test ./... -count=1
```

Expected: 本需求涉及的 `internal/mml` 通过。若仍只有已记录的
`internal/core/carrier.TestRegistryResolveByOUI` 和
`internal/core/dictloader.TestCacheVersion_SelfIncrement_DoesNotFireOwnBump` 失败，
按既有基线报告；出现新的失败则停止交付并修复。

- [ ] **Step 4: 部署 113 自测环境（172.21.158.113）**

先只读解析正在运行的 compose 项目和工作目录。团队既有 113 部署镜像位于
`/opt/omcgo-src`；Compose label 的 working directory 是其
`deployments/docker` 子目录：

```bash
ssh root@172.21.158.113 'docker ps --format "{{.Names}}\t{{.Image}}\t{{.Ports}}"'
ssh root@172.21.158.113 'docker inspect omcgo-web-1 --format "{{index .Config.Labels \"com.docker.compose.project\"}} {{index .Config.Labels \"com.docker.compose.project.working_dir\"}} {{index .Config.Labels \"com.docker.compose.project.config_files\"}}"'
```

确认 project 为 `omcgo`、config 指向 `/opt/omcgo-src/deployments/docker/docker-compose.yml`
且 `/opt/omcgo-src` 存在后，仅同步本需求运行时文件并重建 `app`、`web`：

```bash
tar -cf - \
  omcgo/internal/mml/sub_field_model.go \
  omcgo/internal/mml/admin_repository.go \
  omcgo/internal/mml/console_service.go \
  omcgo/internal/mml/custom_command_path_model.go \
  omcgo/internal/mml/custom_command_path_repository.go \
  omcgo/internal/mml/pg_repository.go \
  omcmb/frontend-core/src/types/mmlConsole.ts \
  omcmb/frontend-core/src/types/mml.ts \
  omcmb/frontend-core/src/types/paramModel.ts \
  omcmb/frontend-core/src/services/api/mmlApi.ts \
  omcmb/frontend-core/src/hooks/api/mmlQueryKeys.ts \
  omcmb/frontend-core/src/hooks/api/useMML.ts \
  omcmb/frontend-core/src/hooks/api/useMmlConsole.ts \
  omcmb/frontend-core/src/hooks/api/useParamModels.ts \
  omcmb/frontend-core/src/i18n/zh-CN/index.ts \
  omcmb/frontend-core/src/i18n/en-US/index.ts \
  omcmb/webcode/src/pages/mml/Console/types.ts \
  omcmb/webcode/src/pages/mml/Console/adapters.ts \
  omcmb/webcode/src/pages/mml/Console/modParamValidation.ts \
  omcmb/webcode/src/pages/mml/Console/components/CommandSelectModal.tsx \
  omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.tsx \
| ssh root@172.21.158.113 'test -d /opt/omcgo-src; tar -xf - -C /opt/omcgo-src; cd /opt/omcgo-src; docker compose -p omcgo -f deployments/docker/docker-compose.yml up -d --no-deps --build --force-recreate app web; docker compose -p omcgo -f deployments/docker/docker-compose.yml ps app web'
curl -I http://172.21.158.113:8081/mml/console
```

若 SSH、compose labels 或远端 working directory 任一不可用，停止部署并报告实际门禁，
不猜测目录。Expected: `app`、`web` 正常运行，页面返回 HTTP 200。

- [ ] **Step 5: 在 113 环境进行只读浏览器验收**

使用真实浏览器打开 `http://172.21.158.113:8081/mml/console`：

1. 登录后选择一台测试设备。
2. 选择内置 MOD 命令和两个可写 Path。
3. 确认配置页只展示选中 Path，且 Path 后类型与标准参数树一致。
4. 确认 `min_value` 默认填充。
5. 输入边界值，确认执行按钮可用。
6. 输入越界值，确认逐字段错误和执行按钮禁用。
7. 选择未配置范围的参数，输入任意非空值，确认不触发范围校验。
8. 选择自定义 MOD 命令，重复类型和范围检查。
9. 不点击最终执行按钮，不修改真实设备。

记录页面 URL、类型标签文本、输入值、错误文案、按钮禁用状态、console error 和 page error。

- [ ] **Step 6: 最终审查提交序列**

```bash
git status --short --branch
git log --oneline origin/main..HEAD
git diff --stat origin/main...HEAD
```

Expected: 分支包含设计文档及 4 个小步实现提交；无未提交文件、无无关改动、未推送。

---

### Task 6: 修复最终审查发现的真实数据兼容问题

**Files:**

- Modify: `omcmb/frontend-core/src/types/paramModel.ts`
- Modify: `omcmb/webcode/src/pages/mml/Console/modParamValidation.test.ts`
- Modify: `omcmb/webcode/src/pages/mml/Console/components/CommandSelectModal.tsx`
- Modify: `omcmb/webcode/src/pages/mml/Console/components/CommandSelectModal.test.tsx`
- Modify: `omcmb/webcode/src/pages/mml/components/customizedSubtree.tsx`
- Modify: `omcgo/internal/mml/pg_repository.go`
- Modify: `omcgo/internal/mml/model.go`
- Modify: `omcgo/internal/mml/service.go`
- Modify: `omcgo/internal/mml/custom_command_path_model.go`
- Modify: `omcgo/internal/mml/custom_command_path_repository.go`
- Create: `omcgo/internal/mml/custom_command_path_integration_test.go`
- Modify: `omcmb/frontend-core/src/types/mml.ts`
- Modify: `omcmb/frontend-core/src/services/api/mmlApi.ts`
- Modify: `omcmb/frontend-core/src/services/api/__tests__/mmlConsoleApi.test.ts`
- Create: `omcmb/frontend-core/src/hooks/api/mmlQueryKeys.ts`
- Modify: `omcmb/frontend-core/src/hooks/api/useMML.ts`
- Modify: `omcmb/frontend-core/src/hooks/api/useMmlConsole.ts`
- Modify: `omcmb/frontend-core/src/hooks/api/useParamModels.ts`
- Create: `omcmb/frontend-core/src/hooks/api/__tests__/useMMLCustomPathInvalidation.test.tsx`

- [x] **Step 1: 以真实标准树类型写失败测试**

使用 113 只读查询确认存量包含 `STRING`、`BOOLEAN`、`DATE_TIME`、`INT`、
`U_INT` 等类型。为大写字符串长度、布尔/日期无范围语义写测试并确认原实现误判。

- [x] **Step 2: 规范化类型语义**

`dataTypeRangeKind`、`isStringDataType`、`isUnsignedDataType` 按大小写不敏感、
忽略下划线/空白/连字符的 key 判断；保留原始值用于类型标签。

- [x] **Step 3: 以正常自定义命令创建/更新链路写集成失败测试**

通过现有 service 创建带 JSON `param_paths` 的命令，再调用富化 Path 读取；验证修复前
返回空。测试还覆盖更新后替换关联、关联接口增删改双写、非法 Path 事务回滚、
`default_selected` 保留，以及历史 JSON-only 命令的 trim/空值过滤/去重回退。

- [x] **Step 4: 同事务同步关联并兼容历史数据**

`PgCustomCommandRepository.Create/Update` 在同一事务中解析 `standard_params` 并同步
`mml_custom_command_paths`；关联增删改接口同步更新 JSON；富化查询补入没有关联行的
历史 JSON Path，并以 `mutable=false` 标记为只读。缺失标准 Path 整体回滚并返回输入错误。
父命令 PUT 省略 `param_paths` 时通过字段掩码在父行锁内读取最新 JSON，不能用
service 层旧快照覆盖并发的 Path 关联修改。

- [x] **Step 5: 保留产品过滤并失效缓存**

富化 Path 与产品过滤后的 `selectedCustom.paramPaths` 取交集。模板、产品参数映射或
标准参数元数据变更后，同时失效 `['mml','custom-commands']` 和
`['mml','console','custom-command-paths']` 前缀。

- [x] **Step 6: 运行定向回归**

```bash
cd omcgo
go build ./...
go test ./internal/mml/... -count=1

cd ../omcmb
npm test --workspace webcode -- --run \
  ../frontend-core/src/services/api/__tests__/mmlConsoleApi.test.ts \
  ../frontend-core/src/hooks/api/__tests__/useMMLCustomPathInvalidation.test.tsx \
  src/pages/mml/Console/modParamValidation.test.ts \
  src/pages/mml/Console/components/CommandSelectModal.test.tsx \
  src/pages/mml/Console/components/ConfigParamsModal.test.tsx
npm run typecheck
```

Expected: Go 构建/MML 测试通过；前端 5 个文件 53 个测试和 typecheck 通过。

- [ ] **Step 7: 重新部署并走正常创建链路验收**

部署 Task 5 更新后的运行时文件。在 113 通过现有
`POST /api/v1/mml/templates` 创建临时自定义 MOD，不直接插入关联表；确认后端自动生成
关联、Console 可见富化类型/范围，并覆盖大写 `STRING` 长度校验。验收后通过现有
`DELETE /api/v1/mml/templates/:id` 删除命令，复核命令和关联均为 0。
