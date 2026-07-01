export interface DeviceGroupCountNode {
  id: string;
  parentId: string | null;
  deviceCount: number;
}

export function buildDeviceGroupSubtreeCountMap<T extends DeviceGroupCountNode>(groups: T[]): Map<string, number> {
  const childrenByParent = new Map<string, T[]>();

  for (const group of groups) {
    if (!group.parentId) continue;
    const siblings = childrenByParent.get(group.parentId) ?? [];
    siblings.push(group);
    childrenByParent.set(group.parentId, siblings);
  }

  const memo = new Map<string, number>();

  const countSubtree = (group: T): number => {
    const cached = memo.get(group.id);
    if (cached !== undefined) {
      return cached;
    }

    const children = childrenByParent.get(group.id) ?? [];
    const total = children.length === 0
      ? group.deviceCount
      : children.reduce((sum, child) => sum + countSubtree(child), 0);

    memo.set(group.id, total);
    return total;
  };

  for (const group of groups) {
    countSubtree(group);
  }

  return memo;
}