import { describe, it, expect } from 'vitest';
import {
  UNASSIGNED_GROUP_ID,
  isUnassignedGroup,
  buildGroupTargetOptions,
} from '../deviceGroupTargets';
import type { DeviceGroup } from '../../types/device';

function mkGroup(p: Partial<DeviceGroup> & Pick<DeviceGroup, 'id' | 'name' | 'parentId'>): DeviceGroup {
  return {
    deviceCount: 0,
    description: '',
    builtIn: 0,
    ...p,
  } as DeviceGroup;
}

const ROOT_ID = '00000000-0000-0000-0000-000000000001';

describe('isUnassignedGroup — 未分组内置节点判定', () => {
  it('命中 ...0002 → true', () => {
    expect(isUnassignedGroup(UNASSIGNED_GROUP_ID)).toBe(true);
  });
  it('真实分组 / null / undefined → false', () => {
    expect(isUnassignedGroup('5a002196-2c4e-4b52-a0a9-ab6e93e30deb')).toBe(false);
    expect(isUnassignedGroup(null)).toBe(false);
    expect(isUnassignedGroup(undefined)).toBe(false);
  });
});

describe('buildGroupTargetOptions — 移动/添加到分组的目标下拉（issue #478）', () => {
  const getParentName = (parentId: string | null) =>
    parentId === ROOT_ID ? '默认分组' : '';

  it('成功路径：真实二级分组作为普通写入目标（isRemove=false，标签拼父/子）', () => {
    const groups: DeviceGroup[] = [
      mkGroup({ id: ROOT_ID, name: '默认分组', parentId: null }),
      mkGroup({ id: 'g-real', name: '北京一区', parentId: ROOT_ID }),
    ];
    const opts = buildGroupTargetOptions(groups, getParentName, '移出分组');
    expect(opts).toEqual([
      { label: '默认分组 / 北京一区', value: 'g-real', isRemove: false },
    ]);
  });

  it('一级分组(root, parentId==null)被过滤，不作为目标', () => {
    const groups: DeviceGroup[] = [
      mkGroup({ id: ROOT_ID, name: '默认分组', parentId: null }),
    ];
    expect(buildGroupTargetOptions(groups, getParentName, '移出分组')).toEqual([]);
  });

  it('特例路径：「未分组设备」内置节点保留为"移出分组"项（isRemove=true，用注入文案）', () => {
    const groups: DeviceGroup[] = [
      mkGroup({ id: ROOT_ID, name: '默认分组', parentId: null }),
      mkGroup({ id: UNASSIGNED_GROUP_ID, name: '未分组设备', parentId: ROOT_ID, builtIn: 1 }),
      mkGroup({ id: 'g-real', name: '北京一区', parentId: ROOT_ID }),
    ];
    const opts = buildGroupTargetOptions(groups, getParentName, '移出分组（未分组设备）');
    expect(opts).toContainEqual({
      label: '移出分组（未分组设备）',
      value: UNASSIGNED_GROUP_ID,
      isRemove: true,
    });
    // 未分组项的 value 仍是 ...0002，交给后端按"移出分组"语义处理（删归属记录）
    const removeOpt = opts.find((o) => o.isRemove);
    expect(removeOpt?.value).toBe(UNASSIGNED_GROUP_ID);
    // 真实分组仍正常列出
    expect(opts.some((o) => o.value === 'g-real' && !o.isRemove)).toBe(true);
  });
});
