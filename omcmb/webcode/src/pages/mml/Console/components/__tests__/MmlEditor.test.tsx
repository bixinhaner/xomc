import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { Modal } from 'antd';
import type { Statement, SubFieldDef, ParseError } from '@core/types/mmlConsole';

const setMmlTextDebounced = vi.fn();
const parseAsync = vi.fn().mockResolvedValue({ statements: [], parseErrors: [] });
const executeAsync = vi.fn().mockResolvedValue({ id: 'task-1' });

let storeSlice: {
  mmlText: string;
  parsePending: boolean;
  parseErrors: ParseError[];
  statements: Statement[];
  selectedDeviceSns: string[];
  setMmlTextDebounced: typeof setMmlTextDebounced;
};

vi.mock('@core/store/mmlConsoleStore', () => ({
  useMmlConsoleStore: <T,>(selector: (s: typeof storeSlice) => T) => selector(storeSlice),
}));

vi.mock('@core/hooks/api/useMmlConsole', () => ({
  useParseMML: () => ({ mutateAsync: parseAsync, isPending: false }),
  useExecuteStatements: () => ({ mutateAsync: executeAsync, isPending: false }),
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string) => id,
}));

import MmlEditor from '../MmlEditor';

function rebootSf(): SubFieldDef {
  return {
    id: 'sf-r',
    commandId: 'cmd1',
    paramId: 'p1',
    mmlCode: 'MAND',
    label: 'RebootField',
    labelI18n: {},
    tr069Path: 'Device.Foo',
    valueType: 'string',
    accessType: 'READ_WRITE',
    isObject: false,
    supportsAdd: false,
    supportsDelete: false,
    changeApplies: 'OnReboot',
    constraintText: '',
    constraintTextI18n: {},
    defaultSelected: true,
    isRequired: false,
    sortOrder: 0,
  };
}

function stmtWithReboot(): Statement {
  return {
    uid: 'uid-mod-1',
    commandId: 'cmd1',
    commandCode: 'MOD_X',
    logicalCode: 'X',
    operationType: 'MOD',
    logicalNameI18n: {},
    subFields: [rebootSf()],
    selectedSubFieldIds: [],
    values: { MAND: 'newVal' },
    unknownCodes: [],
  };
}

describe('MmlEditor', () => {
  beforeEach(() => {
    setMmlTextDebounced.mockReset();
    parseAsync.mockClear();
    executeAsync.mockClear();
    storeSlice = {
      mmlText: '',
      parsePending: false,
      parseErrors: [],
      statements: [],
      selectedDeviceSns: [],
      setMmlTextDebounced,
    };
  });

  it('typing into textbox calls store.setMmlTextDebounced with new value', () => {
    render(<MmlEditor />);
    const textarea = screen.getByRole('textbox');
    fireEvent.change(textarea, { target: { value: 'LST DEVICE_INFO:;' } });
    expect(setMmlTextDebounced).toHaveBeenCalledTimes(1);
    expect(setMmlTextDebounced.mock.calls[0][0]).toBe('LST DEVICE_INFO:;');
  });

  it('parseErrors render as an alert message list', () => {
    storeSlice.parseErrors = [
      { statementIndex: 0, raw: 'XXX', reason: 'unknown command' },
      { statementIndex: 1, raw: 'YYY', reason: 'missing colon' },
    ];
    render(<MmlEditor />);
    // i18n stub returns the key + JSON args we do not control; assert the alert container exists
    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByRole('alert').textContent).toMatch(/mml\.console\.parseError\.syntax/);
  });

  it('DO with OnReboot hits opens Modal.confirm before executing', () => {
    storeSlice.selectedDeviceSns = ['SN1'];
    storeSlice.statements = [stmtWithReboot()];

    const confirmSpy = vi.spyOn(Modal, 'confirm').mockImplementation(() => ({
      destroy: () => undefined,
      update: () => undefined,
      then: undefined as never,
    }));

    render(<MmlEditor />);
    fireEvent.click(screen.getByRole('button', { name: /mml\.console\.editor\.execute/ }));

    expect(confirmSpy).toHaveBeenCalledTimes(1);
    expect(executeAsync).not.toHaveBeenCalled(); // execute 推迟到 Modal.confirm onOk

    confirmSpy.mockRestore();
  });
});
