import { describe, expect, it } from 'vitest';

import type { DeviceGroup } from '../../types/device';
import { expandSelectedGroupIds } from '../deviceGroupFilter';

function mkGroup(p: Partial<DeviceGroup> & Pick<DeviceGroup, 'id' | 'name' | 'parentId'>): DeviceGroup {
  return {
    deviceCount: 0,
    description: '',
    builtIn: 0,
    ...p,
  } as DeviceGroup;
}

describe('expandSelectedGroupIds', () => {
  const groups: DeviceGroup[] = [
    mkGroup({ id: 'root', name: '默认组', parentId: null }),
    mkGroup({ id: 'nr', name: 'NR 设备', parentId: 'root' }),
    mkGroup({ id: 'bnq', name: 'BNQ设备组', parentId: 'nr' }),
    mkGroup({ id: 'bnq-child', name: 'BNQ子组', parentId: 'bnq' }),
    mkGroup({ id: 'lte', name: 'LTE 设备', parentId: 'root' }),
  ];

  it('选父组时展开为父组加所有后代组', () => {
    expect(expandSelectedGroupIds('nr', groups)).toEqual(['nr', 'bnq', 'bnq-child']);
  });

  it('选多个组时去重并保留可达后代', () => {
    expect(expandSelectedGroupIds(['nr', 'bnq'], groups)).toEqual(['nr', 'bnq', 'bnq-child']);
  });

  it('空值返回 undefined', () => {
    expect(expandSelectedGroupIds(undefined, groups)).toBeUndefined();
    expect(expandSelectedGroupIds([], groups)).toBeUndefined();
  });
});