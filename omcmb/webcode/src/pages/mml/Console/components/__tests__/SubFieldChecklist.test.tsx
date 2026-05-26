import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import type { Statement, SubFieldDef } from '@core/types/mmlConsole';

const toggleSubField = vi.fn();

vi.mock('@core/store/mmlConsoleStore', () => ({
  useMmlConsoleStore: <T,>(selector: (s: { toggleSubField: typeof toggleSubField }) => T) =>
    selector({ toggleSubField }),
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string) => id,
}));

import SubFieldChecklist from '../SubFieldChecklist';

function sf(overrides: Partial<SubFieldDef>): SubFieldDef {
  return {
    id: 'sf1',
    commandId: 'cmd1',
    paramId: 'p1',
    mmlCode: 'CODE',
    label: 'Some Label',
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

function stmt(subFields: SubFieldDef[], selectedSubFieldIds: string[] = []): Statement {
  return {
    uid: 'uid-1',
    commandId: 'cmd1',
    commandCode: 'LST_X',
    logicalCode: 'X',
    operationType: 'LST',
    logicalNameI18n: {},
    subFields,
    selectedSubFieldIds,
    values: {},
    unknownCodes: [],
  };
}

describe('SubFieldChecklist', () => {
  beforeEach(() => {
    toggleSubField.mockReset();
  });

  it('renders all sub-fields sorted by sortOrder with access icon', () => {
    const fields = [
      sf({ id: 'a', label: 'Alpha', sortOrder: 2, accessType: 'READ_WRITE' }),
      sf({ id: 'b', label: 'Beta', sortOrder: 1, accessType: 'READ_ONLY' }),
    ];
    render(<SubFieldChecklist statement={stmt(fields)} />);
    const labels = screen.getAllByText(/Alpha|Beta/);
    expect(labels[0].textContent).toBe('Beta'); // sortOrder=1 first
    expect(labels[1].textContent).toBe('Alpha');
  });

  it('toggling a checkbox calls store.toggleSubField with uid + subFieldId', () => {
    const fields = [sf({ id: 'sf-x', label: 'X' })];
    render(<SubFieldChecklist statement={stmt(fields, [])} />);
    const cb = screen.getByRole('checkbox');
    fireEvent.click(cb);
    expect(toggleSubField).toHaveBeenCalledWith('uid-1', 'sf-x');
  });

  it('every row has a unified InfoCircle entry (replaces scattered # / OnReboot tag)', () => {
    // 用户决策 2026-05-26：每行尾部统一一个 field-info 图标，
    // OnReboot / 多实例 / access 等细节都在 popover 里。
    const fields = [
      sf({ id: 'r', label: 'RebootField', changeApplies: 'OnReboot' }),
      sf({ id: 'n', label: 'Normal', changeApplies: 'Immediate', sortOrder: 1 }),
      sf({ id: 'm', label: 'Multi', tr069Path: 'Device.X.{i}.Y', sortOrder: 2 }),
    ];
    render(<SubFieldChecklist statement={stmt(fields)} />);
    const icons = screen.getAllByLabelText('field-info');
    expect(icons).toHaveLength(3);
  });
});
