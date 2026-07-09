import type { DeviceGroup } from '../types/device';

/**
 * 「未分组设备」内置虚拟节点 ID（后端 global/consts.go DefaultLevel2GroupID）。
 *
 * 它不是真实分组，只是"没有任何分组归属记录"的虚拟视图。把设备「移动 / 添加」到
 * 该节点 = **移出分组**（后端 move/add 接口收到此目标时删除该批设备的全部归属记录，
 * 见 issue #478 T1）。前端选目标时不再把它当普通写入目标，而是作为"移出分组"项。
 */
export const UNASSIGNED_GROUP_ID = '00000000-0000-0000-0000-000000000002';

/** 判定是否为「未分组设备」内置虚拟节点。 */
export function isUnassignedGroup(groupId: string | null | undefined): boolean {
  return groupId === UNASSIGNED_GROUP_ID;
}

export interface GroupTargetOption {
  label: string;
  value: string;
  /** true = 「移出分组」语义项（未分组内置节点），false = 普通真实分组写入目标。 */
  isRemove: boolean;
}

/**
 * 构造「移动 / 添加到分组」下拉的目标选项。
 *
 * 规则（issue #478）：
 * - 一级分组（root, parentId == null）是容器，不作为设备归属目标 —— 过滤掉。
 * - 「未分组设备」内置节点保留为可选项，展示名仍按分组数据渲染；
 *   但语义是"移出分组"（isRemove=true），选它后端会删除归属记录。
 * - 其余二级/叶子真实分组作为普通写入目标。
 *
 * @param groups       全部分组（含 root 与内置节点）
 * @param getParentName 按 parentId 取父分组名（用于拼 "父 / 子" 标签）
 * @param getGroupName 按 group 取展示名（用于适配 nameI18n / 当前 locale）
 */
export function buildGroupTargetOptions(
  groups: DeviceGroup[],
  getParentName: (parentId: string | null) => string,
  getGroupName: (group: DeviceGroup) => string = (group) => group.name,
): GroupTargetOption[] {
  return groups
    .filter((g) => g.parentId)
    .map((g) => {
      const parentName = getParentName(g.parentId ?? null);
      const groupName = getGroupName(g);
      return {
        label: parentName ? `${parentName} / ${groupName}` : groupName,
        value: g.id,
        isRemove: isUnassignedGroup(g.id),
      };
    });
}
