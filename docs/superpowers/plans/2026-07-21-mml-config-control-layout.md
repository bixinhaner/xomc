# MML 参数控件宽度与下拉箭头布局 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (recommended) to implement this plan task-by-task.

**Goal:** 让 MML 参数页的普通输入框、枚举下拉框和 BOOLEAN 下拉框保持一致宽度，并将下拉箭头固定在控件最右侧。

**Architecture:** 只调整 `ConfigParamsModal` 的参数控件渲染样式。通过统一控件类名和 Select 的语义样式槽位控制外层宽度、内部选择区域、右侧尾部空间与箭头位置；不改 API、数据模型和校验流程。

**Tech Stack:** React、TypeScript、Ant Design、Vitest、Testing Library。

## Global Constraints

- 不修改 XML、数据库迁移、接口字段和参数校验逻辑。
- 普通 `Input`、枚举 `Select`、BOOLEAN `Select` 必须占用同一列宽。
- 枚举文本过长时不能推动箭头或改变控件宽度。
- 浏览器验收不点击“执行”，不触发设备侧下发。

---

### Task 1: 为统一控件宽度和箭头位置补充失败测试

**Files:**
- Modify: `omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.test.tsx`

**Interfaces:**

- Consumes: 当前 `ConfigParamsModal` 的普通输入、枚举 Select 和 BOOLEAN Select 渲染。
- Produces: 可断言的 `.mml-config-param-control` 控件类名和 `.mml-config-param-select` Select 类名。

- [ ] **Step 1: Write the failing test**

在现有 modal 测试中增加一个 MOD 命令，分别提供普通 string、enum options 和 boolean 参数，断言三类控件均有统一控件类名，且 Select 使用统一 Select 类名：

```tsx
it('keeps input and select controls at the same width with a right-aligned arrow', () => {
  renderModal({
    command: {
      ...command,
      operationType: 'MOD',
      paramPaths: [
        { path: 'Device.Param.Text', label: 'Text', writable: true, isObject: false },
        {
          path: 'Device.Param.Mode',
          label: 'Mode',
          writable: true,
          isObject: false,
          enumOptions: [{ value: 'a-very-long-enum-value', label: 'a-very-long-enum-value' }],
        },
        { path: 'Device.Param.Enabled', label: 'Enabled', writable: true, isObject: false, valueType: 'boolean' },
      ],
    },
    selectedPathKeys: ['Device.Param.Text', 'Device.Param.Mode', 'Device.Param.Enabled'],
  });

  expect(document.querySelectorAll('.mml-config-param-control')).toHaveLength(3);
  expect(document.querySelectorAll('.mml-config-param-select')).toHaveLength(2);
});
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd omcmb && npm run test --workspace webcode -- --run src/pages/mml/Console/components/ConfigParamsModal.test.tsx
```

Expected: FAIL because the new classes do not exist yet.

### Task 2: 实现统一宽度和右侧箭头布局

**Files:**
- Modify: `omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.tsx`

**Interfaces:**

- Consumes: Task 1 的 DOM class assertions。
- Produces: 参数控件统一布局；Select 的箭头始终位于最右侧。

- [ ] **Step 1: Write minimal implementation**

为 Input 和 Select 增加统一控件类名；Select 使用 Ant Design 6 的 semantic styles 固定 root 宽度、content 右侧空间和 suffix 箭头位置：

```tsx
const PARAM_CONTROL_STYLE = { display: 'block', width: '100%', marginTop: 6 };

<Select
  className="mml-config-param-select mml-config-param-control"
  styles={{
    root: { width: '100%' },
    content: { minWidth: 0, paddingInlineEnd: 32, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' },
    suffix: { insetInlineEnd: 12 },
  }}
  style={PARAM_CONTROL_STYLE}
  ...
/>

<Input
  className="mml-config-param-control"
  style={PARAM_CONTROL_STYLE}
  ...
/>
```

同时保证 Select 选中项文本使用可收缩的单行布局，长枚举值不挤压箭头。

- [ ] **Step 2: Run focused test to verify it passes**

```bash
cd omcmb && npm run test --workspace webcode -- --run src/pages/mml/Console/components/ConfigParamsModal.test.tsx
```

Expected: PASS。

### Task 3: 前端验证和 113 浏览器验收

**Files:**
- No source changes expected.

- [ ] **Step 1: Run typecheck and focused tests**

```bash
cd omcmb && npm run typecheck
cd omcmb && npm run test --workspace webcode -- --run src/pages/mml/Console/components/ConfigParamsModal.test.tsx
```

Expected: typecheck exits 0 and focused tests pass。

- [ ] **Step 2: Verify the real page**

在 `http://172.21.158.113:8081/mml/console` 打开参数配置弹窗，读取实际 DOM：普通输入框和 Select 外层宽度一致，Select 箭头位于控件最右侧；只浏览/选择，不点击执行按钮。

- [ ] **Step 3: Run final diff checks**

```bash
git diff --check
git status --short --branch
```

记录任何环境限制，不把未执行的验证写成已通过。
