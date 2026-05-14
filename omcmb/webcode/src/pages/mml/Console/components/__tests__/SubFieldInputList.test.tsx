import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import type { Statement, SubFieldDef } from '@core/types/mmlConsole';

const setValue = vi.fn();

vi.mock('@core/store/mmlConsoleStore', () => ({
  useMmlConsoleStore: <T,>(selector: (s: { setValue: typeof setValue }) => T) =>
    selector({ setValue }),
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

  it('required field shows red asterisk + typing fires store.setValue', () => {
    const fields = [
      sf({ id: 'r', label: 'Mandatory', mmlCode: 'MAND', isRequired: true }),
    ];
    render(<SubFieldInputList statement={modStmt(fields)} />);
    expect(screen.getByLabelText('required')).toBeInTheDocument();

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'abc' } });
    expect(setValue).toHaveBeenCalledWith('uid-mod', 'MAND', 'abc');
  });

  it('constraintText renders as hint below input', () => {
    const fields = [
      sf({ id: 'c', label: 'Constrained', constraintText: '1..100' }),
    ];
    render(<SubFieldInputList statement={modStmt(fields)} />);
    expect(screen.getByText(/1\.\.100/)).toBeInTheDocument();
  });
});
