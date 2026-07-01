import type { DeviceGroup } from '../types/device';

function normalizeSelectedGroupIds(groupId?: string | string[]): string[] {
  if (!groupId) return [];
  if (Array.isArray(groupId)) return groupId.filter(Boolean);
  return groupId ? [groupId] : [];
}

// 设备列表按父组过滤时，需要把后代组一并带上；后端列表筛选仍按真实 membership
// 精确匹配 group_id，不会自动做树展开。
export function expandSelectedGroupIds(
  groupId: string | string[] | undefined,
  groups: DeviceGroup[],
): string[] | undefined {
  const selectedGroupIDs = normalizeSelectedGroupIds(groupId);
  if (selectedGroupIDs.length === 0) return undefined;

  const childrenByParent = new Map<string, string[]>();
  for (const group of groups) {
    if (!group.parentId) continue;
    const siblings = childrenByParent.get(group.parentId) ?? [];
    siblings.push(group.id);
    childrenByParent.set(group.parentId, siblings);
  }

  const expanded: string[] = [];
  const seen = new Set<string>();
  const queue = [...selectedGroupIDs];
  while (queue.length > 0) {
    const current = queue.shift();
    if (!current || seen.has(current)) continue;
    seen.add(current);
    expanded.push(current);
    const children = childrenByParent.get(current) ?? [];
    queue.push(...children);
  }

  return expanded;
}