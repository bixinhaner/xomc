import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import type { Statement, SubFieldDef } from '@core/types/mmlConsole';
import type { UseGPVProbeResult } from '@core/hooks/api/useGPVProbe';

// R-7 多选改造后：store 同时暴露 setRmvIndex（兼容）+ setRmvIndices（新主路径）。
// InstancePicker 现在统一调 setRmvIndices；setRmvIndex 留作 mock 接口完整性占位。
const setRmvIndex = vi.fn();
const setRmvIndices = vi.fn();
let mockSelectedDeviceSns: string[] = [];

vi.mock('@core/store/mmlConsoleStore', () => ({
  useMmlConsoleStore: <T,>(
    selector: (s: {
      setRmvIndex: typeof setRmvIndex;
      setRmvIndices: typeof setRmvIndices;
      selectedDeviceSns: string[];
    }) => T,
  ) =>
    selector({
      setRmvIndex,
      setRmvIndices,
      selectedDeviceSns: mockSelectedDeviceSns,
    }),
}));

vi.mock('@/hooks/useT', () => ({
  useT:
    () =>
    (id: string, values?: Record<string, unknown>) =>
      values ? `${id}|${JSON.stringify(values)}` : id,
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
import {
  effectiveIndices,
  parseTagValues,
} from '../instanceSelection';

function rmvStmt(opts?: {
  rmvInstanceIndex?: number;
  rmvInstanceIndices?: number[];
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
    rmvInstanceIndices: opts?.rmvInstanceIndices,
    unknownCodes: [],
    targetObject: opts?.targetObject,
  };
}

describe('InstancePicker (R-7 多选 + T-0130 GPV SSE 闭环)', () => {
  beforeEach(() => {
    setRmvIndex.mockReset();
    setRmvIndices.mockReset();
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
    // indexHelp 同时用于 Select placeholder + 下方 helper 文案，故 ≥ 1 处
    expect(
      screen.getAllByText('mml.console.picker.indexHelp').length,
    ).toBeGreaterThan(0);
    expect(screen.getByTestId('instance-picker-probe-btn')).toBeInTheDocument();
  });

  it('探测按钮在 0 设备时 disabled', () => {
    mockSelectedDeviceSns = [];
    render(<InstancePicker statement={rmvStmt({ targetObject: 'Device.Foo.' })} />);
    expect(screen.getByTestId('instance-picker-probe-btn')).toBeDisabled();
  });

  it('探测按钮在多设备时 disabled（仅支持单设备探测）', () => {
    mockSelectedDeviceSns = ['SN-A', 'SN-B'];
    render(<InstancePicker statement={rmvStmt({ targetObject: 'Device.Foo.' })} />);
    expect(screen.getByTestId('instance-picker-probe-btn')).toBeDisabled();
  });

  it('探测按钮在无 targetObject 时 disabled', () => {
    mockSelectedDeviceSns = ['SN-A'];
    render(<InstancePicker statement={rmvStmt()} />);
    expect(screen.getByTestId('instance-picker-probe-btn')).toBeDisabled();
  });

  it('单设备 + targetObject + idle → 探测按钮 enabled，点击触发 probe()', () => {
    mockSelectedDeviceSns = ['SN-A'];
    render(<InstancePicker statement={rmvStmt({ targetObject: 'Device.Foo.' })} />);
    const btn = screen.getByTestId('instance-picker-probe-btn');
    expect(btn).not.toBeDisabled();
    fireEvent.click(btn);
    expect(mockProbe).toHaveBeenCalledTimes(1);
  });

  it('R-7：所有状态都显示 tags-Select（不再区分 InputNumber/Select 分支）', () => {
    mockSelectedDeviceSns = ['SN-A'];
    mockProbeResult = {
      probe: mockProbe,
      reset: mockReset,
      status: 'failed',
      instances: [],
      error: 'fail',
    };
    render(<InstancePicker statement={rmvStmt({ targetObject: 'Device.Foo.' })} />);
    expect(screen.getByTestId('instance-picker-select')).toBeInTheDocument();
    // 旧 InputNumber 不再存在
    expect(screen.queryByTestId('instance-picker-input')).toBeNull();
  });

  it('success + instances 非空 时 Select 渲染探测选项', () => {
    mockSelectedDeviceSns = ['SN-A'];
    mockProbeResult = {
      probe: mockProbe,
      reset: mockReset,
      status: 'success',
      instances: [1, 2, 5],
    };
    render(<InstancePicker statement={rmvStmt({ targetObject: 'Device.Foo.' })} />);
    expect(screen.getByTestId('instance-picker-select')).toBeInTheDocument();
    // 探测成功提示带 count
    expect(
      screen.getByText(/mml\.console\.picker\.probeSuccess\|.*"count":3/),
    ).toBeInTheDocument();
  });

  it('rmvInstanceIndices=[1,3,5] → 显示 multiSelected 计数徽章', () => {
    mockSelectedDeviceSns = ['SN-A'];
    render(
      <InstancePicker
        statement={rmvStmt({
          rmvInstanceIndices: [1, 3, 5],
          targetObject: 'Device.Foo.',
        })}
      />,
    );
    expect(
      screen.getByText(/mml\.console\.picker\.multiSelected\|.*"count":3/),
    ).toBeInTheDocument();
  });

  it('单实例（仅 rmvInstanceIndex=3）不显示 multiSelected 徽章（count=1 不算多选）', () => {
    mockSelectedDeviceSns = ['SN-A'];
    render(
      <InstancePicker
        statement={rmvStmt({ rmvInstanceIndex: 3, targetObject: 'Device.Foo.' })}
      />,
    );
    expect(
      screen.queryByText(/mml\.console\.picker\.multiSelected/),
    ).toBeNull();
  });

  it('rmvInstanceIndices 与 rmvInstanceIndex 共存时优先 indices', () => {
    mockSelectedDeviceSns = ['SN-A'];
    render(
      <InstancePicker
        statement={rmvStmt({
          rmvInstanceIndex: 99,
          rmvInstanceIndices: [1, 2],
          targetObject: 'Device.Foo.',
        })}
      />,
    );
    // count 反映 indices.length=2，而非旧单值 99
    expect(
      screen.getByText(/mml\.console\.picker\.multiSelected\|.*"count":2/),
    ).toBeInTheDocument();
  });

  it('reset 按钮（探测成功后出现）调 setRmvIndices(uid, undefined) 清空选择', () => {
    mockSelectedDeviceSns = ['SN-A'];
    mockProbeResult = {
      probe: mockProbe,
      reset: mockReset,
      status: 'success',
      instances: [1, 2, 3],
    };
    render(<InstancePicker statement={rmvStmt({ targetObject: 'Device.Foo.' })} />);
    const resetBtn = screen.getByText('mml.console.picker.reset');
    fireEvent.click(resetBtn);
    expect(mockReset).toHaveBeenCalledTimes(1);
    expect(setRmvIndices).toHaveBeenCalledWith('uid-rmv', undefined);
  });

  // antd Select tags 的回车追加在 jsdom 下不稳定；改为直测核心解析逻辑。
  // 端到端 onChange→setRmvIndices 走 Playwright E2E 覆盖。
  describe('parseTagValues / effectiveIndices 单元', () => {
    it('parseTagValues 过滤非数字、负数、重复', () => {
      expect(parseTagValues(['1', '3', '5'])).toEqual([1, 3, 5]);
      expect(parseTagValues(['1', '1', '2'])).toEqual([1, 2]);
      expect(parseTagValues(['-3', 'abc', '4', ''])).toEqual([4]);
      expect(parseTagValues([' 7 ', '7'])).toEqual([7]);
      expect(parseTagValues([])).toEqual([]);
    });

    it('effectiveIndices 优先 indices，回退到单 Index，否则空', () => {
      expect(
        effectiveIndices(rmvStmt({ rmvInstanceIndices: [1, 3, 5] })),
      ).toEqual([1, 3, 5]);
      expect(effectiveIndices(rmvStmt({ rmvInstanceIndex: 7 }))).toEqual([7]);
      expect(
        effectiveIndices(
          rmvStmt({ rmvInstanceIndex: 99, rmvInstanceIndices: [1, 2] }),
        ),
      ).toEqual([1, 2]);
      expect(effectiveIndices(rmvStmt())).toEqual([]);
      // indices 空数组 ≡ 未选
      expect(effectiveIndices(rmvStmt({ rmvInstanceIndices: [] }))).toEqual([]);
    });
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
