# MML Command Path Selection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move LST/DSP/MOD Path selection into the MML command-selection modal so the configuration modal only confirms selected query Paths or accepts values for selected writable MOD Paths.

**Architecture:** Keep the complete `CommandItem` separate from the current `selectedPathKeys` in `MMLConsole`. Centralize operation-aware filtering, ordered selection, and MOD value validation in pure functions; use a focused controlled `CommandPathSelector` for the command modal, then scope `ConfigParamsModal` to the selected Paths without changing `ExecRequest` or backend APIs.

**Tech Stack:** React 19, TypeScript 6 strict mode, Ant Design 6, Vitest 4, Testing Library, existing OMC frontend i18n and MML APIs.

## Global Constraints

- Work only in `.worktrees/mml-command-path-selection`, based on `origin/main` commit `0fc870ea5`.
- Preserve the original checkout's local commit, tracked modification, and untracked files.
- LST/DSP candidates are all currently executable Paths; MOD candidates are only `writable=true` Paths.
- A newly selected or changed command starts with zero selected Paths.
- Reopening the currently confirmed command restores its confirmed Path selection.
- LST/DSP/MOD cannot proceed with zero selected Paths.
- ADD/RMV keep the existing `target_object` and instance-index flow.
- MOD requires a non-whitespace value for every selected Path.
- Keep full command metadata in `CommandItem`; store the current selection separately as `string[]`.
- Do not change backend routes, request schemas, task execution, SSE, or result rendering.
- Add every user-visible string to both `frontend-core/src/i18n/zh-CN/index.ts` and `frontend-core/src/i18n/en-US/index.ts`.
- Browser acceptance target is `http://172.17.9.239:8081/mml/console`; do not click the final execute button.

---

### Task 1: Pure Path selection and validation rules

**Files:**
- Create: `omcmb/webcode/src/pages/mml/Console/pathSelection.ts`
- Test: `omcmb/webcode/src/pages/mml/Console/pathSelection.test.ts`

**Interfaces:**
- Consumes: `MMLOperationType`, `CommandParamPath`, and existing `isReadOp`.
- Produces:
  - `commandUsesPathSelection(op): boolean`
  - `getSelectableCommandPaths(op, paths): CommandParamPath[]`
  - `getOrderedSelectedCommandPaths(paths, selectedPathKeys): CommandParamPath[]`
  - `getOrderedSelectedPathKeys(paths, selectedPathKeys): string[]`
  - `areSelectedPathValuesComplete(paths, values): boolean`

- [ ] **Step 1: Write failing tests for operation-aware candidates**

Create `pathSelection.test.ts` with explicit readable and writable fixtures:

```ts
import { describe, expect, it } from 'vitest';
import type { CommandParamPath } from './types';
import {
  areSelectedPathValuesComplete,
  commandUsesPathSelection,
  getOrderedSelectedCommandPaths,
  getOrderedSelectedPathKeys,
  getSelectableCommandPaths,
} from './pathSelection';

const paths: CommandParamPath[] = [
  { path: 'Device.Info.Serial', label: 'Serial', writable: false, isObject: false },
  { path: 'Device.Info.Name', label: 'Name', writable: true, isObject: false },
  { path: 'Device.Info.Mode', label: 'Mode', writable: true, isObject: false },
];

describe('MML command Path selection rules', () => {
  it('uses Path selection only for LST, DSP, and MOD', () => {
    expect(commandUsesPathSelection('LST')).toBe(true);
    expect(commandUsesPathSelection('DSP')).toBe(true);
    expect(commandUsesPathSelection('MOD')).toBe(true);
    expect(commandUsesPathSelection('ADD')).toBe(false);
    expect(commandUsesPathSelection('RMV')).toBe(false);
  });

  it('offers every executable Path for query commands', () => {
    expect(getSelectableCommandPaths('LST', paths)).toEqual(paths);
    expect(getSelectableCommandPaths('DSP', paths)).toEqual(paths);
  });

  it('offers only writable Paths for MOD', () => {
    expect(getSelectableCommandPaths('MOD', paths).map((path) => path.path)).toEqual([
      'Device.Info.Name',
      'Device.Info.Mode',
    ]);
  });

  it('does not introduce Path selection for ADD or RMV', () => {
    expect(getSelectableCommandPaths('ADD', paths)).toEqual([]);
    expect(getSelectableCommandPaths('RMV', paths)).toEqual([]);
  });
});
```

- [ ] **Step 2: Run the focused test and verify the red state**

Run:

```bash
cd omcmb
npm run test --workspace webcode -- src/pages/mml/Console/pathSelection.test.ts
```

Expected: FAIL because `./pathSelection` does not exist.

- [ ] **Step 3: Implement operation-aware candidates**

Create `pathSelection.ts`:

```ts
import type { MMLOperationType } from '@core/types/mml';
import { isReadOp } from './constants';
import type { CommandParamPath } from './types';

export function commandUsesPathSelection(op: MMLOperationType | string | undefined): boolean {
  return isReadOp(op) || op === 'MOD';
}

export function getSelectableCommandPaths(
  op: MMLOperationType | string | undefined,
  paths: CommandParamPath[],
): CommandParamPath[] {
  if (isReadOp(op)) return paths;
  if (op === 'MOD') return paths.filter((path) => path.writable);
  return [];
}
```

- [ ] **Step 4: Add failing tests for order, stale-key pruning, and MOD values**

Append:

```ts
describe('selected Path projection', () => {
  it('keeps command-definition order and drops stale keys', () => {
    const selected = ['Device.Info.Mode', 'Device.Missing', 'Device.Info.Name'];
    expect(getOrderedSelectedCommandPaths(paths, selected).map((path) => path.path)).toEqual([
      'Device.Info.Name',
      'Device.Info.Mode',
    ]);
    expect(getOrderedSelectedPathKeys(paths, selected)).toEqual([
      'Device.Info.Name',
      'Device.Info.Mode',
    ]);
  });

  it('requires a non-whitespace value for every selected MOD Path', () => {
    const selected = paths.slice(1);
    expect(areSelectedPathValuesComplete(selected, {
      'Device.Info.Name': 'cell-a',
      'Device.Info.Mode': '1',
    })).toBe(true);
    expect(areSelectedPathValuesComplete(selected, {
      'Device.Info.Name': 'cell-a',
      'Device.Info.Mode': '   ',
    })).toBe(false);
    expect(areSelectedPathValuesComplete(selected, {
      'Device.Info.Name': 'cell-a',
    })).toBe(false);
  });
});
```

- [ ] **Step 5: Run the test and verify the second red state**

Run the same focused command.

Expected: FAIL because the three projection/value functions are not exported.

- [ ] **Step 6: Implement ordered projection and value validation**

Append to `pathSelection.ts`:

```ts
export function getOrderedSelectedCommandPaths(
  paths: CommandParamPath[],
  selectedPathKeys: string[],
): CommandParamPath[] {
  const selected = new Set(selectedPathKeys);
  return paths.filter((path) => selected.has(path.path));
}

export function getOrderedSelectedPathKeys(
  paths: CommandParamPath[],
  selectedPathKeys: string[],
): string[] {
  return getOrderedSelectedCommandPaths(paths, selectedPathKeys).map((path) => path.path);
}

export function areSelectedPathValuesComplete(
  paths: CommandParamPath[],
  values: Record<string, string>,
): boolean {
  return paths.length > 0 && paths.every((path) => (values[path.path] ?? '').trim() !== '');
}
```

- [ ] **Step 7: Run the focused test and verify green**

Expected: all tests in `pathSelection.test.ts` PASS.

- [ ] **Step 8: Commit the rule layer**

```bash
git add omcmb/webcode/src/pages/mml/Console/pathSelection.ts \
  omcmb/webcode/src/pages/mml/Console/pathSelection.test.ts
git commit -m "test(mml): 锁定命令 Path 选择规则"
```

---

### Task 2: Path multi-select in the command-selection modal

**Files:**
- Create: `omcmb/webcode/src/pages/mml/Console/components/CommandPathSelector.tsx`
- Test: `omcmb/webcode/src/pages/mml/Console/components/CommandPathSelector.test.tsx`
- Test: `omcmb/webcode/src/pages/mml/Console/components/CommandSelectModal.test.tsx`
- Modify: `omcmb/webcode/src/pages/mml/Console/components/CommandSelectModal.tsx:1-390`
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts:6363-6373`
- Modify: `omcmb/frontend-core/src/i18n/en-US/index.ts:6334-6344`

**Interfaces:**
- Consumes: Task 1 selection helpers and existing `CommandParamPath`.
- Produces:
  - Controlled `CommandPathSelector({ paths, value, onChange })`.
  - `CommandSelectModal.selectedPathKeys: string[]`.
  - `CommandSelectModal.onConfirm(command, selectedPathKeys)`.

- [ ] **Step 1: Write the failing controlled-selector component tests**

Create `CommandPathSelector.test.tsx`:

```tsx
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { CommandParamPath } from '../types';
import CommandPathSelector from './CommandPathSelector';

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string, values?: Record<string, unknown>) =>
    values ? `${id}:${JSON.stringify(values)}` : id,
}));

const paths: CommandParamPath[] = [
  { path: 'Device.Info.Serial', label: 'Serial', writable: false, isObject: false },
  { path: 'Device.Info.Name', label: 'Name', writable: true, isObject: false },
];

describe('CommandPathSelector', () => {
  it('starts fully unchecked and reports selected and total counts', () => {
    render(<CommandPathSelector paths={paths} value={[]} onChange={() => undefined} />);

    expect(screen.getAllByRole('checkbox')).toHaveLength(3);
    expect(screen.getAllByRole('checkbox').every((checkbox) => !(checkbox as HTMLInputElement).checked)).toBe(true);
    expect(screen.getByText(/"selected":0/)).toBeInTheDocument();
    expect(screen.getByText(/"total":2/)).toBeInTheDocument();
  });

  it('emits selected keys in candidate order and supports select all', () => {
    const onChange = vi.fn();
    const { rerender } = render(<CommandPathSelector paths={paths} value={[]} onChange={onChange} />);

    fireEvent.click(screen.getByRole('checkbox', { name: /Name/ }));
    expect(onChange).toHaveBeenLastCalledWith(['Device.Info.Name']);

    rerender(<CommandPathSelector paths={paths} value={['Device.Info.Name']} onChange={onChange} />);
    fireEvent.click(screen.getByRole('checkbox', { name: 'mml.consoleV2.cmdSelect.selectAll' }));
    expect(onChange).toHaveBeenLastCalledWith(['Device.Info.Serial', 'Device.Info.Name']);
  });
});
```

- [ ] **Step 2: Run the selector test and verify it fails**

Run:

```bash
cd omcmb
npm run test --workspace webcode -- src/pages/mml/Console/components/CommandPathSelector.test.tsx
```

Expected: FAIL because `CommandPathSelector` does not exist.

- [ ] **Step 3: Implement the controlled selector**

Create `CommandPathSelector.tsx`:

```tsx
import { Checkbox, Space, Typography } from 'antd';
import { useT } from '@/hooks/useT';
import { getOrderedSelectedPathKeys } from '../pathSelection';
import type { CommandParamPath } from '../types';

const { Text } = Typography;

interface CommandPathSelectorProps {
  paths: CommandParamPath[];
  value: string[];
  onChange: (selectedPathKeys: string[]) => void;
}

export default function CommandPathSelector({ paths, value, onChange }: CommandPathSelectorProps) {
  const t = useT();
  const selected = getOrderedSelectedPathKeys(paths, value);
  const allKeys = paths.map((path) => path.path);
  const allSelected = paths.length > 0 && selected.length === paths.length;

  return (
    <Space orientation="vertical" size={8} style={{ width: '100%' }}>
      <Space size={12}>
        <Checkbox
          checked={allSelected}
          indeterminate={selected.length > 0 && !allSelected}
          onChange={(event) => onChange(event.target.checked ? allKeys : [])}
        >
          {t('mml.consoleV2.cmdSelect.selectAll')}
        </Checkbox>
        <Text type="secondary">
          {t('mml.consoleV2.cmdSelect.selectedCount', {
            selected: selected.length,
            total: paths.length,
          })}
        </Text>
      </Space>

      <Checkbox.Group
        style={{ display: 'flex', flexDirection: 'column', gap: 6, width: '100%' }}
        value={selected}
        onChange={(keys) => onChange(getOrderedSelectedPathKeys(paths, keys as string[]))}
      >
        {paths.map((path) => (
          <Checkbox key={path.path} value={path.path} style={{ whiteSpace: 'nowrap' }}>
            <Text>{path.label}</Text>{' '}
            <Text type="secondary" code style={{ fontSize: 11 }}>
              {path.path}
            </Text>
          </Checkbox>
        ))}
      </Checkbox.Group>
    </Space>
  );
}
```

- [ ] **Step 4: Add bilingual selector copy**

Add adjacent to the existing `cmdSelect` keys:

```ts
// zh-CN
'mml.consoleV2.cmdSelect.selectAll':       '全选',
'mml.consoleV2.cmdSelect.selectedCount':   '已选 {selected} 项 / 共 {total} 项',
'mml.consoleV2.cmdSelect.pickPathFirst':   '请至少选择一个 PATH',

// en-US
'mml.consoleV2.cmdSelect.selectAll':       'Select All',
'mml.consoleV2.cmdSelect.selectedCount':   '{selected} selected / {total} total',
'mml.consoleV2.cmdSelect.pickPathFirst':   'Select at least one PATH',
```

- [ ] **Step 5: Run the selector test and verify green**

Expected: both selector tests PASS.

- [ ] **Step 6: Write failing modal integration tests**

Create `CommandSelectModal.test.tsx` with stable mocks and render helpers:

```tsx
import { fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import type { CommandItem } from '../types';
import CommandSelectModal from './CommandSelectModal';

const fixtures = vi.hoisted(() => {
  const commands = [
    {
      id: 'lst-1',
      commandCode: 'LST INFO',
      displayName: '查询设备信息',
      operationType: 'LST',
      targetObject: '',
    },
    {
      id: 'mod-1',
      commandCode: 'MOD INFO',
      displayName: '修改设备信息',
      operationType: 'MOD',
      targetObject: '',
    },
  ];

  return {
    subFields: [
      { tr069Path: 'Device.Info.Serial', label: 'Serial', accessType: 'READ_ONLY', isObject: false },
      { tr069Path: 'Device.Info.Name', label: 'Name', accessType: 'READ_WRITE', isObject: false },
    ],
    groupTree: [{
      id: 'group-1',
      displayName: '设备信息',
      commands,
      children: [],
    }],
  };
});

vi.mock('@core/hooks/api/useMmlConsole', () => ({
  useGroupTree: () => ({ data: fixtures.groupTree, isLoading: false }),
  useCommandSubFields: (id?: string) => ({
    data: id ? fixtures.subFields : undefined,
    isFetching: false,
  }),
  useUnsupportedPaths: () => ({ data: [] }),
}));

vi.mock('@/hooks/useI18nText', () => ({
  useI18nText: () => ({ locale: 'zh-CN' }),
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string) => id,
}));

vi.mock('../../components/customizedSubtree', async () => {
  const actual = await vi.importActual<typeof import('../../components/customizedSubtree')>(
    '../../components/customizedSubtree',
  );
  return {
    ...actual,
    useCustomCommands: () => ({ commands: [] }),
  };
});

function makeCommandItem(id: string, operationType: 'LST' | 'MOD'): CommandItem {
  return {
    id,
    groupName: '设备信息',
    commandCode: `${operationType} INFO`,
    commandName: operationType === 'LST' ? '查询设备信息' : '修改设备信息',
    operationType,
    description: '',
    paramPaths: [
      { path: 'Device.Info.Serial', label: 'Serial', writable: false, isObject: false },
      { path: 'Device.Info.Name', label: 'Name', writable: true, isObject: false },
    ],
  };
}

type ModalProps = React.ComponentProps<typeof CommandSelectModal>;

function renderCommandModal(props: Partial<ModalProps> = {}) {
  return (
    <App>
      <CommandSelectModal
        open
        value={null}
        selectedPathKeys={[]}
        deviceSn="device-1"
        productId="product-1"
        onCancel={() => undefined}
        onConfirm={() => undefined}
        onGotoRawParams={() => undefined}
        {...props}
      />
    </App>
  );
}

function renderModal(props: Partial<ModalProps> = {}) {
  return render(renderCommandModal(props));
}
```

Cover these assertions:

```tsx
it('disables confirmation until an LST Path is selected and returns only selected keys', async () => {
  const onConfirm = vi.fn();
  const { container } = renderModal({ onConfirm, value: null, selectedPathKeys: [] });
  fireEvent.click(container.querySelector('.ant-tree-switcher')!);
  fireEvent.click(await screen.findByText('查询设备信息'));

  expect(screen.getByRole('button', { name: 'mml.consoleV2.cmdSelect.okText' })).toBeDisabled();
  fireEvent.click(await screen.findByRole('checkbox', { name: /Name/ }));
  fireEvent.click(screen.getByRole('button', { name: 'mml.consoleV2.cmdSelect.okText' }));

  expect(onConfirm).toHaveBeenCalledWith(
    expect.objectContaining({ id: 'lst-1', paramPaths: expect.arrayContaining([expect.objectContaining({ path: 'Device.Info.Serial' })]) }),
    ['Device.Info.Name'],
  );
});

it('shows only writable Path candidates for MOD', async () => {
  const { container } = renderModal({ value: null, selectedPathKeys: [] });
  fireEvent.click(container.querySelector('.ant-tree-switcher')!);
  fireEvent.click(await screen.findByText('修改设备信息'));

  expect(await screen.findByRole('checkbox', { name: /Name/ })).toBeInTheDocument();
  expect(screen.queryByRole('checkbox', { name: /Serial/ })).not.toBeInTheDocument();
});

it('restores confirmed keys on reopen and clears the draft when switching commands', async () => {
  const { container, rerender } = renderModal({
    value: makeCommandItem('lst-1', 'LST'),
    selectedPathKeys: ['Device.Info.Name'],
  });
  expect(await screen.findByRole('checkbox', { name: /Name/ })).toBeChecked();

  rerender(renderCommandModal({ open: false, value: makeCommandItem('lst-1', 'LST'), selectedPathKeys: ['Device.Info.Name'] }));
  rerender(renderCommandModal({ open: true, value: makeCommandItem('lst-1', 'LST'), selectedPathKeys: ['Device.Info.Name'] }));
  expect(await screen.findByRole('checkbox', { name: /Name/ })).toBeChecked();

  fireEvent.click(container.querySelector('.ant-tree-switcher')!);
  fireEvent.click(await screen.findByText('修改设备信息'));
  expect(await screen.findByRole('checkbox', { name: /Name/ })).not.toBeChecked();
});
```

The test helpers must wrap the modal in Ant Design `App`, return stable mock objects, and cast the intentionally minimal group/command fixtures to the hook return type rather than weakening production types.

- [ ] **Step 7: Run the modal test and verify red**

Run:

```bash
cd omcmb
npm run test --workspace webcode -- src/pages/mml/Console/components/CommandSelectModal.test.tsx
```

Expected: FAIL because the modal has no `selectedPathKeys` prop, returns one callback argument, and renders no Path checkboxes.

- [ ] **Step 8: Wire draft selection into `CommandSelectModal`**

Make these interface and state changes:

```ts
interface CommandSelectModalProps {
  open: boolean;
  value: CommandItem | null;
  selectedPathKeys: string[];
  onCancel: () => void;
  onConfirm: (command: CommandItem, selectedPathKeys: string[]) => void;
  // existing props unchanged
}

const [draftPathKeys, setDraftPathKeys] = useState<string[]>([]);

if (open !== wasOpen) {
  setWasOpen(open);
  if (open) {
    setKeyword('');
    setExpandedKeys([]);
    setSelectedId(value ? (value.isCustom ? `${CUSTOM_KEY_PREFIX}${value.id}` : value.id) : undefined);
    setDraftPathKeys(value ? selectedPathKeys : []);
  }
}
```

When the tree selection changes, reset the draft:

```ts
if (k && !k.startsWith('group:') && k !== CUSTOM_ROOT_KEY) {
  if (k !== selectedId) setDraftPathKeys([]);
  setSelectedId(k);
}
```

After `paramPaths` and operation type are known:

```ts
const selectedOperation = selectedCustom?.operationType ?? selectedEntry?.command.operationType;
const usesPathSelection = commandUsesPathSelection(selectedOperation);
const selectablePaths = getSelectableCommandPaths(selectedOperation, paramPaths);
const effectiveDraftPathKeys = getOrderedSelectedPathKeys(selectablePaths, draftPathKeys);
```

Update `okDisabled` so LST/DSP/MOD require `effectiveDraftPathKeys.length > 0`, while ADD/RMV and other operations preserve current rules.

Render `CommandPathSelector` for LST/DSP/MOD. Keep the existing target-object block for ADD/RMV and existing read-only preview for other operations.

In `handleOk`, keep the mapped command's complete filtered `paramPaths`, and pass the ordered selected keys as the second callback argument:

```ts
onConfirm(mappedCommand, usesPathSelection ? effectiveDraftPathKeys : []);
```

- [ ] **Step 9: Run selector and modal tests**

Expected: all Task 2 tests PASS.

- [ ] **Step 10: Commit the command-selection UI**

```bash
git add omcmb/webcode/src/pages/mml/Console/components/CommandPathSelector.tsx \
  omcmb/webcode/src/pages/mml/Console/components/CommandPathSelector.test.tsx \
  omcmb/webcode/src/pages/mml/Console/components/CommandSelectModal.tsx \
  omcmb/webcode/src/pages/mml/Console/components/CommandSelectModal.test.tsx \
  omcmb/frontend-core/src/i18n/zh-CN/index.ts \
  omcmb/frontend-core/src/i18n/en-US/index.ts
git commit -m "feat(mml): 在命令页前置选择 Path"
```

---

### Task 3: Scope the configuration page to confirmed Paths

**Files:**
- Modify: `omcmb/webcode/src/pages/mml/Console/index.tsx:75-82,508-549`
- Modify: `omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.tsx:1-425`
- Test: `omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.test.tsx`
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts:6387-6393`
- Modify: `omcmb/frontend-core/src/i18n/en-US/index.ts:6358-6364`

**Interfaces:**
- Consumes: `CommandSelectModal.onConfirm(command, selectedPathKeys)`, Task 1 projection/value helpers.
- Produces: `ConfigParamsModal.selectedPathKeys`, query-only summary, selected MOD inputs, and unchanged `ExecRequest.checkedPaths`.

- [ ] **Step 1: Write failing query and MOD configuration tests**

Create `ConfigParamsModal.test.tsx` with complete mocks and render helpers:

```tsx
import { fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import type { CommandItem } from '../types';
import ConfigParamsModal from './ConfigParamsModal';

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string) => id,
}));

vi.mock('@core/hooks/usePermission', () => ({
  usePermission: () => true,
}));

const command: CommandItem = {
  id: 'command-1',
  groupName: '设备信息',
  commandCode: 'LST INFO',
  commandName: '查询设备信息',
  operationType: 'LST',
  description: '',
  paramPaths: [
    { path: 'Device.Info.Serial', label: 'Serial', writable: false, isObject: false },
    { path: 'Device.Info.Name', label: 'Name', writable: true, isObject: false },
    { path: 'Device.Info.Mode', label: 'Mode', writable: true, isObject: false },
  ],
};

type ModalProps = React.ComponentProps<typeof ConfigParamsModal>;

function renderModal(props: Partial<ModalProps> = {}) {
  return render(
    <App>
      <ConfigParamsModal
        open
        command={command}
        selectedPathKeys={[]}
        deviceCount={1}
        initialMode="standard"
        onCancel={() => undefined}
        onConfirmAndExecute={() => undefined}
        {...props}
      />
    </App>,
  );
}
```

Add:

```tsx
it('shows only confirmed query Paths without second-stage checkboxes', () => {
  renderModal({
    command,
    selectedPathKeys: ['Device.Info.Name', 'Device.Info.Mode'],
  });

  expect(screen.queryByText('Serial')).not.toBeInTheDocument();
  expect(screen.getByText('Name')).toBeInTheDocument();
  expect(screen.getByText('Mode')).toBeInTheDocument();
  expect(screen.queryAllByRole('checkbox')).toHaveLength(0);
});

it('shows inputs only for selected writable MOD Paths and requires every value', () => {
  const onConfirmAndExecute = vi.fn();
  renderModal({
    command: { ...command, operationType: 'MOD', commandCode: 'MOD INFO', commandName: '修改设备信息' },
    selectedPathKeys: ['Device.Info.Name'],
    onConfirmAndExecute,
  });

  expect(screen.queryByText('Serial')).not.toBeInTheDocument();
  expect(screen.getByText('Name')).toBeInTheDocument();
  expect(screen.queryByText('Mode')).not.toBeInTheDocument();

  const execute = screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ });
  expect(execute).toBeDisabled();
  fireEvent.change(screen.getByRole('textbox'), { target: { value: 'cell-a' } });
  expect(execute).toBeEnabled();
  fireEvent.click(execute);

  expect(onConfirmAndExecute).toHaveBeenCalledWith(expect.objectContaining({
    mode: 'standard',
    checkedPaths: ['Device.Info.Name'],
    values: { 'Device.Info.Name': 'cell-a' },
  }));
});
```

- [ ] **Step 2: Add a failing ADD/RMV regression test**

Append:

```tsx
it('preserves ADD target-object and value behavior without Path selection', () => {
  renderModal({
    command: {
      ...command,
      id: 'add-1',
      operationType: 'ADD',
      commandCode: 'ADD SERVICE',
      commandName: '新增服务对象',
      targetObject: 'Device.Services.FAPService.{i}.',
      paramPaths: [
        { path: 'Device.Services.FAPService.{i}.Enable', label: 'Enable', writable: true, isObject: false },
      ],
    },
    selectedPathKeys: [],
  });

  expect(screen.getByText(/Device\.Services\.FAPService/)).toBeInTheDocument();
  expect(screen.getByText('Enable')).toBeInTheDocument();
  expect(screen.getByRole('textbox')).toBeInTheDocument();
});

it('preserves RMV instance input and whole-request execution mode', () => {
  renderModal({
    command: {
      ...command,
      id: 'rmv-1',
      operationType: 'RMV',
      commandCode: 'RMV SERVICE',
      commandName: '删除服务对象',
      targetObject: 'Device.Services.FAPService.{i}.',
      paramPaths: [],
    },
    selectedPathKeys: [],
  });

  expect(screen.getAllByRole('spinbutton').length).toBeGreaterThan(0);
  expect(screen.getByRole('radio', { name: 'mml.consoleV2.config.execPerPath' })).toBeDisabled();
  expect(screen.getByRole('radio', { name: 'mml.consoleV2.config.execWhole' })).toBeChecked();
});
```

- [ ] **Step 3: Run the configuration test and verify red**

Run:

```bash
cd omcmb
npm run test --workspace webcode -- src/pages/mml/Console/components/ConfigParamsModal.test.tsx
```

Expected: FAIL because the modal has no `selectedPathKeys` prop, query Paths are still checkboxes, and MOD accepts empty values.

- [ ] **Step 4: Store confirmed selection in `MMLConsole`**

Add state:

```ts
const [selectedPathKeys, setSelectedPathKeys] = useState<string[]>([]);
```

Pass it into `CommandSelectModal`, receive the second callback value, and initialize standard config with that confirmed selection:

```tsx
<CommandSelectModal
  open={commandModalOpen}
  value={command}
  selectedPathKeys={selectedPathKeys}
  // existing props
  onConfirm={(cmd, pathKeys) => {
    setCommand(cmd);
    setSelectedPathKeys(pathKeys);
    setConfig({
      mode: 'standard',
      checkedPaths: commandUsesPathSelection(cmd.operationType)
        ? pathKeys
        : cmd.paramPaths.map((path) => path.path),
    });
    setConfigTouched(false);
    setConfigMode('standard');
    setCommandModalOpen(false);
    setConfigModalOpen(true);
  }}
/>
```

Pass `selectedPathKeys` to `ConfigParamsModal`. Keep raw-Path mode independent.

- [ ] **Step 5: Scope `ConfigParamsModal` to the confirmed selection**

Add the prop:

```ts
selectedPathKeys: string[];
```

Derive the scoped command and Paths:

```ts
const usesPathSelection = commandUsesPathSelection(command?.operationType);
const selectedParamPaths = useMemo(
  () => getOrderedSelectedCommandPaths(command?.paramPaths ?? [], selectedPathKeys),
  [command, selectedPathKeys],
);
const standardParamPaths = usesPathSelection ? selectedParamPaths : (command?.paramPaths ?? []);
const scopedCommand = useMemo(
  () => command && usesPathSelection ? { ...command, paramPaths: standardParamPaths } : command,
  [command, standardParamPaths, usesPathSelection],
);
const instanceSlots = useMemo(
  () => scopedCommand ? computeInstanceSlots(scopedCommand) : [],
  [scopedCommand],
);
```

Replace `lastCmdId` with a configuration key that includes the ordered selected Paths:

```ts
const configKey = command
  ? `${command.id}:${standardParamPaths.map((path) => path.path).join('\u0000')}`
  : '';
const [lastConfigKey, setLastConfigKey] = useState('');
```

When the key changes:

- `checkedPaths` is the selected ordered list for LST/DSP/MOD.
- ADD and other existing write operations retain all writable Paths.
- Values are rebuilt only from `standardParamPaths`, applying existing `minValue` defaults.
- Instance selectors are rebuilt from `scopedCommand`.
- Hidden values from a previous selection are removed.

- [ ] **Step 6: Replace query checkboxes with a read-only confirmation list**

Remove the query “全选” control and `Checkbox.Group`. Render:

```tsx
<Space orientation="vertical" size={6} style={{ width: '100%', marginTop: 8 }}>
  {selectedParamPaths.map((path) => (
    <div key={path.path} style={{ whiteSpace: 'nowrap' }}>
      <Text>{path.label}</Text>{' '}
      <Text type="secondary" code style={{ fontSize: 11 }}>
        {path.path}
      </Text>
    </div>
  ))}
</Space>
```

Use selected count in the command header. For MOD, render inputs from `selectedParamPaths.filter((path) => path.writable)`. For ADD, continue rendering all existing writable command Paths.

- [ ] **Step 7: Enforce complete selected MOD values**

Change standard validity:

```ts
const modValuesValid =
  command?.operationType !== 'MOD' ||
  areSelectedPathValuesComplete(selectedParamPaths, values);

const standardValid =
  !!command &&
  (isAddRmvCmd
    ? !!command.targetObject?.trim()
    : checkedPaths.length > 0 && modValuesValid);
```

Before constructing `ExecRequest`, project `values` to `checkedPaths` so no hidden value can be sent:

```ts
const checkedPathSet = new Set(checkedPaths);
const selectedValues = Object.fromEntries(
  Object.entries(values).filter(([path]) => checkedPathSet.has(path)),
);
```

Use `selectedValues` for MOD/ADD request values.

- [ ] **Step 8: Add bilingual confirmation/value copy**

Add:

```ts
// zh-CN
'mml.consoleV2.config.confirmSelectedPaths': '确认本次执行的 PATH',
'mml.consoleV2.config.modValueRequired':      '请填写所有已选 PATH 的修改值',

// en-US
'mml.consoleV2.config.confirmSelectedPaths': 'Confirm PATHs for this execution',
'mml.consoleV2.config.modValueRequired':      'Enter a value for every selected PATH',
```

Use `confirmSelectedPaths` for the query hint and show `modValueRequired` as a validation hint while selected MOD values are incomplete.

- [ ] **Step 9: Run configuration and full focused tests**

Run:

```bash
cd omcmb
npm run test --workspace webcode -- \
  src/pages/mml/Console/pathSelection.test.ts \
  src/pages/mml/Console/components/CommandPathSelector.test.tsx \
  src/pages/mml/Console/components/CommandSelectModal.test.tsx \
  src/pages/mml/Console/components/ConfigParamsModal.test.tsx
```

Expected: all focused tests PASS with no unhandled React or Ant Design errors.

- [ ] **Step 10: Commit the configuration flow**

```bash
git add omcmb/webcode/src/pages/mml/Console/index.tsx \
  omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.tsx \
  omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.test.tsx \
  omcmb/frontend-core/src/i18n/zh-CN/index.ts \
  omcmb/frontend-core/src/i18n/en-US/index.ts
git commit -m "feat(mml): 按已选 Path 配置查询与修改命令"
```

---

### Task 4: Frontend regression gates and 239 browser acceptance

**Files:**
- Modify only if a gate exposes a defect in Task 1-3 files.
- Browser evidence is reported in the ship-flow verification record; no generated screenshots or temporary browser scripts are committed.

**Interfaces:**
- Consumes: completed UI flow from Tasks 1-3.
- Produces: P5/P6 hard-gate evidence for type safety, focused behavior, and the real 239 environment.

- [ ] **Step 1: Make dependencies available in the isolated worktree**

Reuse the already installed dependency tree without modifying lockfiles:

```bash
ln -s /Users/a1/Desktop/vscode/gitlab/omcmb/node_modules omcmb/node_modules
```

Expected: `omcmb/node_modules/.package-lock.json` is readable and `git status --short` does not list `node_modules`.

- [ ] **Step 2: Run the focused unit/component suite**

Run the command from Task 3 Step 9.

Expected: all focused tests PASS.

- [ ] **Step 3: Run the mandatory frontend typecheck**

Run:

```bash
cd omcmb
npm run typecheck
```

Expected: exit code 0 with no TypeScript errors.

- [ ] **Step 4: Run lint on changed source and test files**

Run:

```bash
cd omcmb
npx eslint \
  webcode/src/pages/mml/Console/pathSelection.ts \
  webcode/src/pages/mml/Console/pathSelection.test.ts \
  webcode/src/pages/mml/Console/components/CommandPathSelector.tsx \
  webcode/src/pages/mml/Console/components/CommandPathSelector.test.tsx \
  webcode/src/pages/mml/Console/components/CommandSelectModal.tsx \
  webcode/src/pages/mml/Console/components/CommandSelectModal.test.tsx \
  webcode/src/pages/mml/Console/components/ConfigParamsModal.tsx \
  webcode/src/pages/mml/Console/components/ConfigParamsModal.test.tsx \
  webcode/src/pages/mml/Console/index.tsx \
  frontend-core/src/i18n/zh-CN/index.ts \
  frontend-core/src/i18n/en-US/index.ts
```

Expected: exit code 0.

- [ ] **Step 5: Check patch hygiene**

Run:

```bash
git diff --check origin/main...HEAD
git status --short --branch
```

Expected: no whitespace errors; only planned feature/spec/plan files are committed or modified.

- [ ] **Step 6: Deploy the frontend build to the user-selected 239 test environment**

Resolve the 239 deployment with read-only commands:

```bash
ssh root@172.17.9.239 'docker ps --format "{{.Names}}\t{{.Image}}\t{{.Ports}}"'
ssh root@172.17.9.239 'find /opt /srv -maxdepth 4 -path "*/deployments/docker/docker-compose.yml" -print 2>/dev/null'
```

Identify the compose project from the running `web` container's `com.docker.compose.project` and `com.docker.compose.project.working_dir` labels:

```bash
ssh root@172.17.9.239 'docker inspect $(docker ps --filter name=web --format "{{.ID}}" | head -n 1) --format "{{index .Config.Labels \"com.docker.compose.project\"}} {{index .Config.Labels \"com.docker.compose.project.working_dir\"}}"'
```

Stream only the runtime frontend files into the reported working directory, preserving their repository-relative paths, and rebuild the same compose project:

```bash
tar -cf - \
  omcmb/webcode/src/pages/mml/Console/pathSelection.ts \
  omcmb/webcode/src/pages/mml/Console/components/CommandPathSelector.tsx \
  omcmb/webcode/src/pages/mml/Console/components/CommandSelectModal.tsx \
  omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.tsx \
  omcmb/webcode/src/pages/mml/Console/index.tsx \
  omcmb/frontend-core/src/i18n/zh-CN/index.ts \
  omcmb/frontend-core/src/i18n/en-US/index.ts \
| ssh root@172.17.9.239 'web_id=$(docker ps --filter name=web --format "{{.ID}}" | head -n 1); project=$(docker inspect "$web_id" --format "{{index .Config.Labels \"com.docker.compose.project\"}}"); workdir=$(docker inspect "$web_id" --format "{{index .Config.Labels \"com.docker.compose.project.working_dir\"}}"); test -n "$project" && test -d "$workdir"; tar -xf - -C "$workdir"; cd "$workdir"; docker compose -p "$project" -f deployments/docker/docker-compose.yml up -d --no-deps --build --force-recreate web; docker compose -p "$project" -f deployments/docker/docker-compose.yml ps web'
curl -I http://172.17.9.239:8081/mml/console
```

If SSH access or the compose labels are unavailable, stop P6 and report that environmental gate instead of guessing a deployment path.

Expected: `http://172.17.9.239:8081/mml/console` returns HTTP 200 and serves the feature branch frontend bundle.

- [ ] **Step 7: Verify LST behavior in visible Chrome**

Using the existing signed-in 239 session:

1. Select the visible test device.
2. Open “选择命令”, expand “设备信息参数管理”, and select its LST command.
3. Assert all Path checkboxes start unchecked and “确定选择” is disabled.
4. Select exactly two Paths and assert the selected count is 2.
5. Continue to “配置参数”.
6. Assert exactly those two labels/Paths are visible and no Path checkbox exists.
7. Record URL, visible texts, checkbox counts, button state, browser console errors, and `pageerror`.

- [ ] **Step 8: Verify MOD behavior without executing**

Return to command selection:

1. Select the corresponding MOD command.
2. Assert read-only Path `Device.DeviceInfo.SerialNumber` is absent when present in the complete command definition, while writable candidates are visible.
3. Assert all writable Path checkboxes start unchecked.
4. Select exactly two writable Paths and continue.
5. Assert exactly two value inputs are visible.
6. Assert the execute button is disabled while either input is empty.
7. Fill both inputs and assert the execute button becomes enabled.
8. Do not click execute.
9. Leave visible Chrome on the completed MOD confirmation state.

- [ ] **Step 9: Commit any verification-driven fixes**

If Steps 2-8 required code changes, repeat all failed gates and commit only those fixes:

```bash
git add omcmb/webcode/src/pages/mml/Console \
  omcmb/frontend-core/src/i18n/zh-CN/index.ts \
  omcmb/frontend-core/src/i18n/en-US/index.ts
git commit -m "fix(mml): 修正 Path 选择验收问题"
```

If no fix was needed, do not create an empty commit.

## Final Plan Self-Review

- Spec requirements 1-6 are covered by Tasks 1-3 and browser Steps 7-8.
- ADD/RMV preservation is covered by Task 1 and Task 3 regression tests.
- Backend non-change is enforced by the file list and Global Constraints.
- Default-none, reopen restore, switch reset, stale-key pruning, ordering, and MOD required values all have explicit tests.
- Chinese and English copy are changed together.
- Type names and signatures are consistent across tasks:
  - `selectedPathKeys: string[]`
  - `onConfirm(command: CommandItem, selectedPathKeys: string[])`
  - `ExecRequest.checkedPaths` remains unchanged.
- No implementation placeholder or unresolved design decision remains.
