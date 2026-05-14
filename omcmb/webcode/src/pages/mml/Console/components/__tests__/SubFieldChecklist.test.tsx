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

  it('OnReboot field shows the "需重启生效" tag (via i18n key)', () => {
    const fields = [
      sf({ id: 'r', label: 'RebootField', changeApplies: 'OnReboot' }),
      sf({ id: 'n', label: 'Normal', changeApplies: 'Immediate', sortOrder: 1 }),
    ];
    render(<SubFieldChecklist statement={stmt(fields)} />);
    expect(screen.getByText('mml.console.subField.onReboot')).toBeInTheDocument();
  });
});
