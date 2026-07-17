import type { DeviceGroupNode } from '@core/types/map';
import type { Domain } from '@core/types/topology';
import { getI18nText, type Locale } from '@core/utils/i18nText';

export function normalizeGroupLocale(locale: string): Locale {
  return locale.toLowerCase().startsWith('en') ? 'en-US' : 'zh-CN';
}

export function getGroupNodeDisplayName(node: DeviceGroupNode, locale: Locale): string {
  return getI18nText(node.nameI18n, locale, node.name);
}

export function domainToGroupNode(domain: Domain): DeviceGroupNode {
  return {
    id: domain.id,
    name: domain.name,
    nameI18n: domain.nameI18n,
    parentId: domain.parentId ?? null,
    level: domain.level,
    children: domain.children?.map(domainToGroupNode),
    deviceCount: domain.deviceCount,
    isLeaf: !domain.children?.length,
  };
}

export function filterGroupTreeBySearch(
  tree: DeviceGroupNode[],
  keyword: string,
  locale: Locale,
): DeviceGroupNode[] {
  const searchLower = keyword.trim().toLowerCase();
  if (!searchLower) return tree;

  const filterNode = (node: DeviceGroupNode): DeviceGroupNode | null => {
    const selfMatch =
      getGroupNodeDisplayName(node, locale).toLowerCase().includes(searchLower)
      || node.id.toLowerCase().includes(searchLower);
    const filteredChildren = node.children
      ?.map(filterNode)
      .filter((child): child is DeviceGroupNode => child !== null) ?? [];

    if (!selfMatch && filteredChildren.length === 0) return null;
    return {
      ...node,
      children: selfMatch ? node.children : filteredChildren,
    };
  };

  return tree
    .map(filterNode)
    .filter((node): node is DeviceGroupNode => node !== null);
}

export function buildGroupDisplayNameById(
  tree: DeviceGroupNode[],
  locale: Locale,
): Map<string, string> {
  const result = new Map<string, string>();
  const walk = (nodes: DeviceGroupNode[]) => {
    for (const node of nodes) {
      result.set(node.id, getGroupNodeDisplayName(node, locale));
      if (node.children?.length) walk(node.children);
    }
  };
  walk(tree);
  return result;
}
