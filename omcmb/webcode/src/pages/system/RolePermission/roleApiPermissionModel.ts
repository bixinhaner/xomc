import type { Menu } from '@core/types/menu';
import type { ApiEndpoint } from '@core/types/system';

const READ_API_GROUPS_BY_MENU_KEY: Readonly<Record<string, readonly string[]>> = {
  'dashboard:home': ['dashboard'],
  'device:list': ['devices', 'device-groups'],
  'device:access-control': ['device-access'],
  'system:user': ['users', 'roles'],
  'system:role': ['roles', 'menus', 'api-endpoints', 'device-groups', 'groups'],
  'system:menu': ['menus'],
  'system:operation-log': ['audit-logs'],
  'system:config': ['sysConfig', 'public'],
  'system:api-management': ['api-endpoints'],
  'system:data-dict': ['sysDictionary', 'sysDictionaryDetail'],
  'system:ui-custom': ['sysConfig'],
  'system:kpi-config': ['dashboard'],
  'mml:admin:catalog': ['mml_admin'],
};

function selectedMenusAndDescendants(menuIds: string[], menus: Menu[]): Menu[] {
  const selected = new Set(menuIds);
  const result: Menu[] = [];

  const walk = (items: Menu[], ancestorSelected: boolean) => {
    for (const menu of items) {
      const active = ancestorSelected || selected.has(menu.id);
      if (active) result.push(menu);
      if (menu.children?.length) walk(menu.children, active);
    }
  };

  walk(menus, false);
  return result;
}

function readGroupsForMenu(menu: Menu): readonly string[] {
  const exact = READ_API_GROUPS_BY_MENU_KEY[menu.permissionKey];
  if (exact) return exact;
  const pageKey = menu.permissionKey.split(':').slice(0, 2).join(':');
  return READ_API_GROUPS_BY_MENU_KEY[pageKey] ?? [];
}

export function inferReadApiEndpointIds(
  menuIds: string[],
  menus: Menu[],
  endpoints: ApiEndpoint[],
): string[] {
  const groups = new Set<string>();
  for (const menu of selectedMenusAndDescendants(menuIds, menus)) {
    for (const group of readGroupsForMenu(menu)) groups.add(group);
  }

  if (groups.size === 0) return [];

  const seen = new Set<string>();
  const result: string[] = [];
  for (const endpoint of endpoints) {
    if (endpoint.method.toUpperCase() !== 'GET') continue;
    if (!endpoint.apiGroup || !groups.has(endpoint.apiGroup)) continue;
    if (seen.has(endpoint.id)) continue;
    seen.add(endpoint.id);
    result.push(endpoint.id);
  }
  return result;
}

export function applyNewApiSuggestions(
  selectedIds: string[],
  suggestedIds: string[],
  alreadySuggestedIds: string[],
  autoSelectedIds: string[],
): { selectedIds: string[]; suggestedIds: string[]; autoSelectedIds: string[] } {
  const currentSuggestions = new Set(suggestedIds);
  const previousAutomatic = new Set(autoSelectedIds);
  const selected = selectedIds.filter(
    (id) => !previousAutomatic.has(id) || currentSuggestions.has(id),
  );
  const selectedSet = new Set(selected);
  const automatic = autoSelectedIds.filter(
    (id) => currentSuggestions.has(id) && selectedSet.has(id),
  );
  const automaticSet = new Set(automatic);
  const offered = Array.from(new Set(alreadySuggestedIds));
  const offeredSet = new Set(offered);

  for (const id of suggestedIds) {
    if (offeredSet.has(id)) continue;
    offeredSet.add(id);
    offered.push(id);
    if (!selectedSet.has(id)) {
      selectedSet.add(id);
      selected.push(id);
      if (!automaticSet.has(id)) {
        automaticSet.add(id);
        automatic.push(id);
      }
    }
  }

  return {
    selectedIds: selected,
    suggestedIds: offered,
    autoSelectedIds: automatic,
  };
}
