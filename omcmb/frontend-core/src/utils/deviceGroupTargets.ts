import type { DeviceGroup } from '../types/device';

/** 默认二级设备组 ID（后端 global/consts.go DefaultLevel2GroupID）。 */
export const UNASSIGNED_GROUP_ID = '00000000-0000-0000-0000-000000000002';

/** 判定是否为默认二级设备组。 */
export function isUnassignedGroup(groupId: string | null | undefined): boolean {
  return groupId === UNASSIGNED_GROUP_ID;
}

export interface GroupTargetOption {
  label: string;
  value: string;
  /** 保留兼容旧调用方；默认组现在也是普通真实分组目标，因此恒为 false。 */
  isRemove: boolean;
}

export interface BuildGroupTargetOptionsOptions {
  /** 已经归属的分组不应作为“移动到”目标。 */
  excludeGroupIds?: Array<string | null | undefined>;
}

/**
 * 构造「移动 / 添加到分组」下拉的目标选项。
 *
 * 规则（issue #478）：
 * - 一级分组（root, parentId == null）是容器，不作为设备归属目标 —— 过滤掉。
 * - 默认二级组和其余二级/叶子真实分组都作为普通写入目标。
 *
 * @param groups       全部分组（含 root 与内置节点）
 * @param getParentName 按 parentId 取父分组名（用于拼 "父 / 子" 标签）
 * @param getGroupName 按 group 取展示名（用于适配 nameI18n / 当前 locale）
 * @param options      附加选项，例如排除当前归属分组
 */
export function buildGroupTargetOptions(
  groups: DeviceGroup[],
  getParentName: (parentId: string | null) => string,
  getGroupName: (group: DeviceGroup) => string = (group) => group.name,
  options: BuildGroupTargetOptionsOptions = {},
): GroupTargetOption[] {
  const excludedGroupIds = new Set(options.excludeGroupIds?.filter((id): id is string => Boolean(id)) ?? []);
  return groups
    .filter((g) => g.parentId)
    .filter((g) => !excludedGroupIds.has(g.id))
    .map((g) => {
      const parentName = getParentName(g.parentId ?? null);
      const groupName = getGroupName(g);
      return {
        label: parentName ? `${parentName} / ${groupName}` : groupName,
        value: g.id,
        isRemove: false,
      };
    });
}
