import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import type { Statement, SubFieldDef } from '@core/types/mmlConsole';
import type { UseGPVProbeResult } from '@core/hooks/api/useGPVProbe';

const setRmvIndex = vi.fn();
let mockSelectedDeviceSns: string[] = [];

vi.mock('@core/store/mmlConsoleStore', () => ({
  useMmlConsoleStore: <T,>(
    selector: (s: {
      setRmvIndex: typeof setRmvIndex;
      selectedDeviceSns: string[];
    }) => T,
  ) =>
    selector({
      setRmvIndex,
      selectedDeviceSns: mockSelectedDeviceSns,
    }),
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string) => id,
}));

const mockProbe = vi.fn();
const mockReset = vi.fn();
let mockProbeResult: UseGPVProbeResult = {
  probe: mockProbe,
  reset: mockReset,
  status: 'idle',
  instances: [],
};

vi.mock('@core/hooks/api/useGPVProbe', () => ({
  useGPVProbe: () => mockProbeResult,
}));

import InstancePicker from '../InstancePicker';

function rmvStmt(opts?: {
  rmvInstanceIndex?: number;
  targetObject?: string;
  subFields?: SubFieldDef[];
}): Statement {
  return {
    uid: 'uid-rmv',
    commandId: 'cmd1',
    commandCode: 'RMV_X',
    logicalCode: 'X',
    operationType: 'RMV',
    logicalNameI18n: {},
    subFields: opts?.subFields ?? [],
    selectedSubFieldIds: [],
    values: {},
    rmvInstanceIndex: opts?.rmvInstanceIndex,
    unknownCodes: [],
    targetObject: opts?.targetObject,
  };
}

describe('InstancePicker (T-0130 GPV SSE 闭环 + supports_delete)', () => {
  beforeEach(() => {
    setRmvIndex.mockReset();
    mockProbe.mockReset();
    mockReset.mockReset();
    mockSelectedDeviceSns = [];
    mockProbeResult = {
      probe: mockProbe,
      reset: mockReset,
      status: 'idle',
      instances: [],
    };
  });

  it('idle 状态显示 index label + helper text + 探测按钮', () => {
    render(<InstancePicker statement={rmvStmt()} />);
    expect(screen.getByText('mml.console.picker.indexLabel')).toBeInTheDocument();
    expect(screen.getByText('mml.console.picker.indexHelp')).toBeInTheDocument();
    expect(screen.getByTestId('instance-picker-probe-btn')).toBeInTheDocument();
  });

  it('探测按钮在 0 设备时 disabled', () => {
    mockSelectedDeviceSns = [];
    render(<InstancePicker statement={rmvStmt({ targetObject: 'Device.Foo.' })} />);
    const btn = screen.getByTestId('instance-picker-probe-btn');
    expect(btn).toBeDisabled();
  });

  it('探测按钮在多设备时 disabled（仅支持单设备探测）', () => {
    mockSelectedDeviceSns = ['SN-A', 'SN-B'];
    render(<InstancePicker statement={rmvStmt({ targetObject: 'Device.Foo.' })} />);
    const btn = screen.getByTestId('instance-picker-probe-btn');
    expect(btn).toBeDisabled();
  });

  it('探测按钮在无 targetObject 时 disabled', () => {
    mockSelectedDeviceSns = ['SN-A'];
    render(<InstancePicker statement={rmvStmt()} />);
    const btn = screen.getByTestId('instance-picker-probe-btn');
    expect(btn).toBeDisabled();
  });

  it('单设备 + targetObject + idle → 探测按钮 enabled，点击触发 probe()', () => {
    mockSelectedDeviceSns = ['SN-A'];
    render(<InstancePicker statement={rmvStmt({ targetObject: 'Device.Foo.' })} />);
    const btn = screen.getByTestId('instance-picker-probe-btn');
    expect(btn).not.toBeDisabled();
    fireEvent.click(btn);
    expect(mockProbe).toHaveBeenCalledTimes(1);
  });

  it('idle/failed/timeout 时显示 InputNumber 手输 fallback', () => {
    mockSelectedDeviceSns = ['SN-A'];
    mockProbeResult = {
      probe: mockProbe,
      reset: mockReset,
      status: 'failed',
      instances: [],
      error: 'fail',
    };
    render(<InstancePicker statement={rmvStmt({ targetObject: 'Device.Foo.' })} />);
    expect(screen.getByTestId('instance-picker-input')).toBeInTheDocument();
  });

  it('success + instances 非空 时显示 Select 替换 InputNumber', () => {
    mockSelectedDeviceSns = ['SN-A'];
    mockProbeResult = {
      probe: mockProbe,
      reset: mockReset,
      status: 'success',
      instances: [1, 2, 5],
    };
    render(<InstancePicker statement={rmvStmt({ targetObject: 'Device.Foo.' })} />);
    expect(screen.getByTestId('instance-picker-select')).toBeInTheDocument();
    expect(screen.queryByTestId('instance-picker-input')).toBeNull();
  });

  it('InputNumber 输入合法 index 调 setRmvIndex', () => {
    mockSelectedDeviceSns = ['SN-A'];
    render(<InstancePicker statement={rmvStmt({ targetObject: 'Device.Foo.' })} />);
    const input = screen.getByRole('spinbutton');
    fireEvent.change(input, { target: { value: '5' } });
    fireEvent.blur(input);
    expect(setRmvIndex).toHaveBeenCalledWith('uid-rmv', 5);
  });

  it('InputNumber 清空 → setRmvIndex(undefined)', () => {
    mockSelectedDeviceSns = ['SN-A'];
    render(<InstancePicker statement={rmvStmt({ rmvInstanceIndex: 3, targetObject: 'Device.Foo.' })} />);
    const input = screen.getByRole('spinbutton') as HTMLInputElement;
    fireEvent.change(input, { target: { value: '' } });
    fireEvent.blur(input);
    expect(setRmvIndex).toHaveBeenCalledWith('uid-rmv', undefined);
  });

  it('success 状态显示已探测计数信息', () => {
    mockSelectedDeviceSns = ['SN-A'];
    mockProbeResult = {
      probe: mockProbe,
      reset: mockReset,
      status: 'success',
      instances: [1, 2, 3],
    };
    render(<InstancePicker statement={rmvStmt({ targetObject: 'Device.Foo.' })} />);
    expect(screen.getByText('mml.console.picker.probeSuccess')).toBeInTheDocument();
  });

  it('failed 状态显示错误提示', () => {
    mockSelectedDeviceSns = ['SN-A'];
    mockProbeResult = {
      probe: mockProbe,
      reset: mockReset,
      status: 'failed',
      instances: [],
      error: 'GPV failed',
    };
    render(<InstancePicker statement={rmvStmt({ targetObject: 'Device.Foo.' })} />);
    expect(screen.getByText('mml.console.picker.probeError')).toBeInTheDocument();
  });
});
