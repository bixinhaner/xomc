import { describe, expect, it } from 'vitest';
import {
  buildDefaultInstanceSelectors,
  buildPerPathStatementPaths,
  buildStandardQueryColumns,
  buildStandardRawRows,
  computeInstanceSlots,
  resolveObjectPath,
  resolveQueryPath,
} from '../adapters';
import { validateRawPath } from '../rawPathValidate';
import type { CommandItem, CommandParamPath } from '../types';
import type { MMLOperationType } from '@core/types/mml';

const path = (p: string, extra: Partial<CommandParamPath> = {}): CommandParamPath => ({
  path: p,
  label: p.split('.').filter(Boolean).pop() ?? p,
  writable: false,
  isObject: false,
  ...extra,
});

const cmd = (op: MMLOperationType, paths: CommandParamPath[], targetObject?: string): CommandItem => ({
  id: 'c1',
  groupName: 'g',
  commandCode: 'X',
  commandName: 'x',
  operationType: op,
  description: '',
  paramPaths: paths,
  targetObject,
});

describe('validateRawPath (plan A — ADD/RMV/{i} 约束)', () => {
  it('一律拒绝 {i} 占位符', () => {
    expect(validateRawPath('LST', 'Device.X.{i}.Y')).toMatch(/\{i\}/);
    expect(validateRawPath('ADD', 'Device.X.{i}.LTECell.')).toMatch(/\{i\}/);
  });

  it('ADD：须以 . 结尾、末级不能是实例号', () => {
    expect(validateRawPath('ADD', 'Device.X.LTECell.')).toBeNull();
    expect(validateRawPath('ADD', 'Device.X.LTECell')).not.toBeNull(); // 缺尾点
    expect(validateRawPath('ADD', 'Device.X.LTECell.3.')).not.toBeNull(); // 不能带实例号
  });

  it('RMV：须以 .<实例号>. 结尾', () => {
    expect(validateRawPath('RMV', 'Device.X.LTECell.3.')).toBeNull();
    expect(validateRawPath('RMV', 'Device.X.LTECell.')).not.toBeNull(); // 缺实例号
  });

  it('空 path 视为通过（由"至少一行"另行约束）', () => {
    expect(validateRawPath('ADD', '   ')).toBeNull();
  });
});

describe('computeInstanceSlots (命令参数 {i} 槽位)', () => {
  it('LST/MOD：取 paramPaths 中占位符最多的一条，逐段取对象名', () => {
    const c = cmd('LST', [
      path('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.{i}.X'),
      path('Device.Services.FAPService.{i}.Y'),
    ]);
    const slots = computeInstanceSlots(c);
    expect(slots.map((s) => s.key)).toEqual(['i01', 'i02']);
    expect(slots.map((s) => s.label)).toEqual(['FAPService', 'Common']);
  });

  it('ADD/RMV：取 targetObject 的父级占位符', () => {
    const c = cmd('ADD', [], 'Device.Services.FAPService.{i}.NeighborList.LTECell.');
    const slots = computeInstanceSlots(c);
    expect(slots).toHaveLength(1);
    expect(slots[0]).toEqual({ key: 'i01', label: 'FAPService' });
  });

  it('LST：识别以 .{i} 结尾的末级对象实例槽位', () => {
    const c = cmd('LST', [
      path('Device.Services.FAPService.{i}.NeighborList.LTECell.{i}'),
    ]);
    const slots = computeInstanceSlots(c);
    expect(slots).toEqual([
      { key: 'i01', label: 'FAPService' },
      { key: 'i02', label: 'LTECell' },
    ]);
  });

  it('无 {i} → 空槽位', () => {
    expect(computeInstanceSlots(cmd('LST', [path('Device.DeviceInfo.SoftwareVersion')]))).toHaveLength(0);
  });
});

describe('resolveObjectPath (ADD/RMV 目标对象路径展示)', () => {
  it('按 instanceSelectors 左→右替换 .{i}.（缺省 1）', () => {
    expect(
      resolveObjectPath('Device.Services.FAPService.{i}.NeighborList.InterRATCell.GSM.', { i01: '2' }),
    ).toBe('Device.Services.FAPService.2.NeighborList.InterRATCell.GSM.');
  });

  it('多个 {i} 按 i01/i02 顺序替换；未提供则用 1', () => {
    expect(
      resolveObjectPath('Device.Services.FAPService.{i}.X.{i}.Y.', { i01: '3', i02: '5' }),
    ).toBe('Device.Services.FAPService.3.X.5.Y.');
    expect(resolveObjectPath('Device.Services.FAPService.{i}.X.{i}.Y.')).toBe(
      'Device.Services.FAPService.1.X.1.Y.',
    );
  });

  it('无 {i} 占位 / 空串原样返回', () => {
    const noPlaceholder = 'Device.Services.FAPService.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.';
    expect(resolveObjectPath(noPlaceholder, { i01: '2' })).toBe(noPlaceholder);
    expect(resolveObjectPath(undefined)).toBe('');
    expect(resolveObjectPath('')).toBe('');
  });
});

describe('buildDefaultInstanceSelectors', () => {
  const slots = [
    { key: 'i01', label: 'A' },
    { key: 'i02', label: 'B' },
    { key: 'i03', label: 'C' },
  ];

  it('查询命令仅将最后一层默认留空', () => {
    expect(buildDefaultInstanceSelectors(slots.slice(0, 1), 'LST')).toEqual({ i01: '' });
    expect(buildDefaultInstanceSelectors(slots.slice(0, 2), 'DSP')).toEqual({
      i01: '1',
      i02: '',
    });
    expect(buildDefaultInstanceSelectors(slots, 'LST')).toEqual({
      i01: '1',
      i02: '1',
      i03: '',
    });
  });

  it('写命令继续默认全部实例为 1', () => {
    for (const op of ['MOD', 'ADD', 'RMV'] as const) {
      expect(buildDefaultInstanceSelectors(slots.slice(0, 2), op)).toEqual({
        i01: '1',
        i02: '1',
      });
    }
  });
});

describe('resolveQueryPath', () => {
  const twoLayer = 'Device.A.{i}.B.{i}.Value';

  it('在单层或最后一层空实例处截断并保留对象尾点', () => {
    expect(resolveQueryPath('Device.A.{i}.Value', { i01: '' })).toBe('Device.A.');
    expect(resolveQueryPath(twoLayer, { i01: '1', i02: '' })).toBe('Device.A.1.B.');
  });

  it('第一层或中间层为空时忽略后续实例与 Path', () => {
    expect(resolveQueryPath(twoLayer, { i01: '', i02: '2' })).toBe('Device.A.');
    expect(
      resolveQueryPath('Device.A.{i}.B.{i}.C.{i}.Value', {
        i01: '1',
        i02: '',
        i03: '3',
      }),
    ).toBe('Device.A.1.B.');
  });

  it('完整实例生成叶子 Path，缺少后层 selector 视为空，多余 selector 被忽略', () => {
    expect(resolveQueryPath(twoLayer, { i01: '3', i02: '2' })).toBe(
      'Device.A.3.B.2.Value',
    );
    expect(resolveQueryPath(twoLayer, { i01: '3' })).toBe('Device.A.3.B.');
    expect(resolveQueryPath('Device.A.{i}.Value', { i01: '4', i02: '9' })).toBe(
      'Device.A.4.Value',
    );
  });

  it('末级对象实例为空时截断为对象集合路径，非空时替换为具体实例', () => {
    const objectPath = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}';
    expect(resolveQueryPath(objectPath, { i01: '1', i02: '' })).toBe(
      'Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.',
    );
    expect(resolveQueryPath(objectPath, { i01: '1', i02: '2' })).toBe(
      'Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.2',
    );
  });

  it('无占位符 Path 原样返回', () => {
    expect(resolveQueryPath('Device.Info.SerialNumber', { i01: '' })).toBe(
      'Device.Info.SerialNumber',
    );
  });
});

describe('buildStandardRawRows', () => {
  const paths = [
    'Device.A.{i}.Value',
    'Device.A.{i}.B.{i}.Value',
  ];

  it.each(['LST', 'DSP'] as const)(
    '%s 查询在进入裸路径通道前生成截断 Path',
    (operationType) => {
      expect(
        buildStandardRawRows(operationType, paths, undefined, {
          i01: '1',
          i02: '',
        }),
      ).toEqual([
        { path: 'Device.A.1.Value', value: '' },
        { path: 'Device.A.1.B.', value: '' },
      ]);
    },
  );

  it('写命令保留原 Path 和原 value', () => {
    expect(
      buildStandardRawRows(
        'MOD',
        [paths[0]],
        { [paths[0]]: 'new-value' },
        { i01: '' },
      ),
    ).toEqual([{ path: paths[0], value: 'new-value' }]);
  });
});

describe('buildStandardQueryColumns', () => {
  it('uses resolved Paths without mutating standard command Paths', () => {
    const paths = [
      'Device.A.{i}.Value',
      'Device.A.{i}.B.{i}.Value',
    ];
    const command = cmd('LST', paths.map((value) => path(value)));

    expect(
      buildStandardQueryColumns(command, paths, { i01: '1', i02: '' })
        .map((column) => column.path),
    ).toEqual([
      'Device.A.1.Value',
      'Device.A.1.B.',
    ]);
    expect(command.paramPaths.map((item) => item.path)).toEqual(paths);
  });

  it('多个叶子解析为同一对象前缀时按首次出现稳定去重', () => {
    const paths = [
      'Device.A.{i}.B.{i}.Value',
      'Device.A.{i}.B.{i}.Name',
    ];
    const command = cmd('LST', paths.map((value) => path(value)));

    expect(
      buildStandardQueryColumns(command, paths, { i01: '1', i02: '' })
        .map(({ key, path: resolvedPath }) => ({ key, path: resolvedPath })),
    ).toEqual([
      { key: 'c0', path: 'Device.A.1.B.' },
    ]);
  });

  it('多个选中叶子折叠为对象查询时保留全部原始 Path 模板', () => {
    const paths = [
      'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth',
      'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ULBandwidth',
    ];
    const command = cmd('LST', paths.map((value) => path(value)));

    expect(buildStandardQueryColumns(command, paths, { i01: '' })).toEqual([
      expect.objectContaining({
        path: 'Device.Services.FAPService.',
        selectedPathTemplates: paths,
      }),
    ]);
  });
});

describe('buildPerPathStatementPaths', () => {
  it('LST 多个叶子截断为同一对象前缀时只创建一条逐 PATH statement', () => {
    const paths = [
      'Device.A.{i}.B.{i}.Value',
      'Device.A.{i}.B.{i}.Name',
      'Device.C.{i}.Value',
    ];
    const command = cmd('LST', paths.map((value) => path(value)));

    expect(
      buildPerPathStatementPaths(command, paths, { i01: '1', i02: '' }),
    ).toEqual([
      'Device.A.{i}.B.{i}.Value',
      'Device.C.{i}.Value',
    ]);
  });

  it('MOD 逐 PATH 保留全部原始 path，不应用查询截断去重', () => {
    const paths = [
      'Device.A.{i}.B.{i}.Value',
      'Device.A.{i}.B.{i}.Name',
    ];
    const command = cmd('MOD', paths.map((value) => path(value, { writable: true })));

    expect(
      buildPerPathStatementPaths(command, paths, { i01: '1', i02: '' }),
    ).toEqual(paths);
  });
});
