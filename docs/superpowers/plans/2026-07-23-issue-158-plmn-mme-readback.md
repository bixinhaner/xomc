# Issue 158 PLMN 与 MME 回读修复实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 MLN/MLQ 恢复独立服务 PLMN 配置，并防止 MME 保存后的旧回读值覆盖本地提交。

**Architecture:** MLN 使用字符串列表专用控件，MLQ 复用现有多实例表格。MME 保存状态记录目标路径和值，前端以条件轮询等待参数值真正回读成功后再清理草稿。

**Tech Stack:** Go、XML 参数模型、React、TypeScript、Ant Design、Zustand、Vitest。

## Global Constraints

- 服务 PLMN 与 MME IP+PLMN 保持独立。
- PLMN 最多 6 条，每条 5 或 6 位数字且不可重复。
- SPV 非成功终态和回读失败时必须保留用户草稿。
- 不使用固定延时作为回读完成条件。
- 所有用户可见新增文案必须国际化。

---

### Task 1: PLMN 列表领域函数

**Files:**
- Create: `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/plmnList.ts`
- Test: `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/__tests__/plmnList.test.ts`

**Interfaces:**
- Produces: `parsePlmnList(raw)`, `serializePlmnList(rows)`, `validatePlmnList(rows, maxRows)`.

- [ ] **Step 1: 写解析、序列化和校验的失败测试**

```ts
expect(parsePlmnList('46000, 46001')).toMatchObject([
  { plmn: '46000' },
  { plmn: '46001' },
]);
expect(serializePlmnList([{ key: '1', plmn: '46000' }])).toBe('46000');
expect(validatePlmnList([{ key: '1', plmn: '46000' }, { key: '2', plmn: '46000' }], 6))
  .toBe('duplicate');
```

- [ ] **Step 2: 运行测试并确认因模块或行为缺失而失败**

Run: `cd omcmb && npm test --workspace webcode -- plmnList.test.ts`

- [ ] **Step 3: 实现最小领域函数**

实现逗号、分号、换行解析；过滤空行；序列化为逗号分隔；校验格式、重复和最大数量。

- [ ] **Step 4: 运行测试并确认通过**

Run: `cd omcmb && npm test --workspace webcode -- plmnList.test.ts`

### Task 2: 条件式参数回读

**Files:**
- Create: `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/parameterReadback.ts`
- Test: `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/__tests__/parameterReadback.test.ts`

**Interfaces:**
- Produces: `waitForExpectedParameterValues(options): Promise<Map<string, string>>`.

- [ ] **Step 1: 写“旧值后继续等待”和“超时保留失败”的测试**

```ts
const result = await waitForExpectedParameterValues({
  expected: new Map([['Device.X', 'new']]),
  read: async () => reads.shift()!,
  intervalMs: 0,
  timeoutMs: 100,
});
expect(result.get('Device.X')).toBe('new');
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `cd omcmb && npm test --workspace webcode -- parameterReadback.test.ts`

- [ ] **Step 3: 实现条件轮询、超时和 AbortSignal**

每次调用 `read` 后比较所有目标路径；只有全部匹配才返回。超时抛出专用错误，取消抛出 AbortError。

- [ ] **Step 4: 运行测试并确认通过**

Run: `cd omcmb && npm test --workspace webcode -- parameterReadback.test.ts`

### Task 3: CellParameterForm 接入

**Files:**
- Modify: `omcmb/frontend-core/src/store/quickSettingsFeedbackStore.ts`
- Modify: `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/CellParameterForm.tsx`
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- Modify: `omcmb/frontend-core/src/i18n/en-US/index.ts`

**Interfaces:**
- Consumes: Task 1、Task 2 的领域函数。
- Produces: MLN PLMN 单列表格和 MME 保存后的目标值回读。

- [ ] **Step 1: 在 `CellFeedback` 中记录本次提交的路径和值**

```ts
expectedReadback?: Record<string, string>;
```

- [ ] **Step 2: 为 `enb-plmn` 增加 `plmn-list-table` 特殊控件**

控件支持新增、删除、格式提示、重复校验和 6 条上限。

- [ ] **Step 3: 修改保存序列化和表单初始化**

`ExistPlmnidList` 使用 Task 1 的解析和序列化函数；MME 保持现有 `IP+PLMN` 格式。

- [ ] **Step 4: 用 Task 2 替换 MME 的固定 500ms 回读**

任务失败或回读失败不清草稿；只有目标值匹配才回填并清草稿。

- [ ] **Step 5: 运行前端相关测试与类型检查**

Run: `cd omcmb && npm test --workspace webcode -- plmnList.test.ts parameterReadback.test.ts`

Run: `cd omcmb && npm run typecheck`

### Task 4: MLN/MLQ 快速设置元数据

**Files:**
- Test: `omcgo/internal/quicksettings/loader_test.go`
- Modify: `omcgo/data/quicksettings/MLN.xml`
- Modify: `omcgo/data/quicksettings/MLQ.xml`
- Modify: `omcgo/data/param-mappings/MLQ.xml`

**Interfaces:**
- Produces: MLN `enb-plmn` 单参数分组；MLQ `enb-plmn` 多实例分组和可写对象映射。

- [ ] **Step 1: 写内置配置失败测试**

断言 MLN 的 `ExistPlmnidList`、MLQ 的 `EPC.PLMNList.{i}.`、`maxInstances=6` 和 PLMNID 叶子。

- [ ] **Step 2: 运行测试并确认失败**

Run: `cd omcgo && go test ./internal/quicksettings -run 'TestBuiltin.*PLMN'`

- [ ] **Step 3: 修改 XML 元数据和 MLQ 对象映射**

MLN 添加单实例分组；MLQ 添加多实例分组和对象定义。

- [ ] **Step 4: 运行测试并确认通过**

Run: `cd omcgo && go test ./internal/quicksettings`

### Task 5: 综合验证

**Files:**
- Review all modified files.

- [ ] **Step 1: 运行前端测试和类型检查**

Run: `cd omcmb && npm test --workspace webcode -- plmnList.test.ts parameterReadback.test.ts`

Run: `cd omcmb && npm run typecheck`

- [ ] **Step 2: 运行后端构建和测试**

Run: `cd omcgo && go build ./...`

Run: `cd omcgo && go test ./...`

- [ ] **Step 3: 检查差异只包含 Issue 158 和设计/计划**

Run: `git diff --check`

Run: `git status --short`
