import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import type { Statement } from '@core/types/mmlConsole';

const setRmvIndex = vi.fn();

vi.mock('@core/store/mmlConsoleStore', () => ({
  useMmlConsoleStore: <T,>(selector: (s: { setRmvIndex: typeof setRmvIndex }) => T) =>
    selector({ setRmvIndex }),
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string) => id,
}));

import InstancePicker from '../InstancePicker';

function rmvStmt(rmvInstanceIndex?: number): Statement {
  return {
    uid: 'uid-rmv',
    commandId: 'cmd1',
    commandCode: 'RMV_X',
    logicalCode: 'X',
    operationType: 'RMV',
    logicalNameI18n: {},
    subFields: [],
    selectedSubFieldIds: [],
    values: {},
    rmvInstanceIndex,
    unknownCodes: [],
  };
}

describe('InstancePicker', () => {
  beforeEach(() => {
    setRmvIndex.mockReset();
  });

  it('shows index label + helper text using i18n keys', () => {
    render(<InstancePicker statement={rmvStmt()} />);
    expect(screen.getByText('mml.console.picker.indexLabel')).toBeInTheDocument();
    expect(screen.getByText('mml.console.picker.indexHelp')).toBeInTheDocument();
  });

  it('entering a valid positive index calls store.setRmvIndex with the number', () => {
    render(<InstancePicker statement={rmvStmt()} />);
    const input = screen.getByRole('spinbutton');
    fireEvent.change(input, { target: { value: '5' } });
    // antd InputNumber 仅在 commit（blur / enter）时调 onChange；显式触发 blur
    fireEvent.blur(input);
    expect(setRmvIndex).toHaveBeenCalledWith('uid-rmv', 5);
  });

  it('clearing the input calls store.setRmvIndex with undefined', () => {
    render(<InstancePicker statement={rmvStmt(3)} />);
    const input = screen.getByRole('spinbutton') as HTMLInputElement;
    fireEvent.change(input, { target: { value: '' } });
    fireEvent.blur(input);
    expect(setRmvIndex).toHaveBeenCalledWith('uid-rmv', undefined);
  });
});
