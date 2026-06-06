import { describe, expect, it } from 'vitest';
import { computeInstanceSlots } from '../adapters';
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

  it('无 {i} → 空槽位', () => {
    expect(computeInstanceSlots(cmd('LST', [path('Device.DeviceInfo.SoftwareVersion')]))).toHaveLength(0);
  });
});
