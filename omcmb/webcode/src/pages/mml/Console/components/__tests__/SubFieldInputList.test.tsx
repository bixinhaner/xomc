import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import type { Statement, SubFieldDef } from '@core/types/mmlConsole';

const setValue = vi.fn();
const toggleSubField = vi.fn();

vi.mock('@core/store/mmlConsoleStore', () => ({
  useMmlConsoleStore: <T,>(
    selector: (s: { setValue: typeof setValue; toggleSubField: typeof toggleSubField }) => T,
  ) => selector({ setValue, toggleSubField }),
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string) => id,
}));

import SubFieldInputList from '../SubFieldInputList';

function sf(overrides: Partial<SubFieldDef>): SubFieldDef {
  return {
    id: 'sf1',
    commandId: 'cmd1',
    paramId: 'p1',
    mmlCode: 'CODE',
    label: 'Label',
    labelI18n: {},
    tr069Path: 'Device.Foo',
    valueType: 'string',
    accessType: 'READ_WRITE',
    isObject: false,
    supportsAdd: false,
    supportsDelete: false,
    changeApplies: 'Immediate',
    constraintText: '',
    constraintTextI18n: {},
    defaultValue: undefined,
    jsRegex: undefined,
    defaultSelected: false,
    isRequired: false,
    sortOrder: 0,
    ...overrides,
  };
}

function modStmt(subFields: SubFieldDef[]): Statement {
  return {
    uid: 'uid-mod',
    commandId: 'cmd1',
    commandCode: 'MOD_X',
    logicalCode: 'X',
    operationType: 'MOD',
    logicalNameI18n: {},
    subFields,
    selectedSubFieldIds: [],
    values: {},
    unknownCodes: [],
  };
}

function addStmt(subFields: SubFieldDef[]): Statement {
  return {
    ...modStmt(subFields),
    uid: 'uid-add',
    operationType: 'ADD',
    commandCode: 'ADD_X',
  };
}

describe('SubFieldInputList', () => {
  beforeEach(() => {
    setValue.mockReset();
    toggleSubField.mockReset();
  });

  it('MOD mode filters out READ_ONLY sub-fields (PRD §7.3)', () => {
    const fields = [
      sf({ id: 'rw', label: 'Editable', accessType: 'READ_WRITE' }),
      sf({ id: 'ro', label: 'Locked', accessType: 'READ_ONLY', sortOrder: 1 }),
    ];
    render(<SubFieldInputList statement={modStmt(fields)} />);
    expect(screen.getByText('Editable')).toBeInTheDocument();
    expect(screen.queryByText('Locked')).not.toBeInTheDocument();
  });

  it('ADD mode preserves all sub-fields including READ_ONLY', () => {
    const fields = [
      sf({ id: 'rw', label: 'Editable', accessType: 'READ_WRITE' }),
      sf({ id: 'ro', label: 'Locked', accessType: 'READ_ONLY', sortOrder: 1 }),
    ];
    render(<SubFieldInputList statement={addStmt(fields)} />);
    expect(screen.getByText('Editable')).toBeInTheDocument();
    expect(screen.getByText('Locked')).toBeInTheDocument();
  });

  it('ADD required field shows red asterisk + typing fires store.setValue', () => {
    // 2026-05-23：MOD path 改为选填，required * 仅在 ADD 模式保留
    // （创建新对象仍需所有 is_required 字段；MOD 用户已显式 "取消" 该 path 的修改）。
    const fields = [
      sf({ id: 'r', label: 'Mandatory', mmlCode: 'MAND', isRequired: true }),
    ];
    render(<SubFieldInputList statement={addStmt(fields)} />);
    expect(screen.getByLabelText('required')).toBeInTheDocument();

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'abc' } });
    expect(setValue).toHaveBeenCalledWith('uid-add', 'MAND', 'abc');
  });

  it('MOD shows Checkbox per row + toggle fires store.toggleSubField', () => {
    // 2026-05-23：MOD path 选填，每行 Checkbox 与 LST 一致
    const fields = [
      sf({ id: 'r', label: 'OptionalPath', mmlCode: 'OPT', isRequired: true }),
    ];
    render(<SubFieldInputList statement={modStmt(fields)} />);
    // MOD 模式不显示 required *（path 选填，user 决定要不要改）
    expect(screen.queryByLabelText('required')).not.toBeInTheDocument();
    // Checkbox aria-label="select-<mmlCode>"
    const cb = screen.getByLabelText('select-OPT');
    fireEvent.click(cb);
    expect(toggleSubField).toHaveBeenCalledWith('uid-mod', 'r');
  });

  it('constraintText 收进 field-info popover（2026-06-02 决策移除 AccessTypeTag）', async () => {
    const fields = [
      sf({ id: 'c', label: 'Constrained', constraintText: '1..100' }),
    ];
    render(<SubFieldInputList statement={modStmt(fields)} />);
    // 用户决策 2026-06-02：移除"读写 <TYPE>"AccessTypeTag，取值范围并入每行尾部
    // field-info 图标的 popover（trigger 含 click）。默认不直接渲染独立 hint 行。
    expect(screen.queryByText(/1\.\.100/)).not.toBeInTheDocument();
    fireEvent.click(screen.getByLabelText('field-info'));
    expect(await screen.findByText(/1\.\.100/)).toBeInTheDocument();
  });
});
