import type { TabItem } from '@core/store/tabStore';
import type { Menu } from '@core/types/menu';
import { resolveMenuLabel } from '@core/types/menu';
import type { NavConfig } from './Sidebar/navConfig';

interface ResolveRouteTabOptions {
  pathname: string;
  dynamicMenuEnabled: boolean;
  dynamicMenus: Menu[];
  staticNav: NavConfig;
  locale: string;
  isAdmin: boolean;
  isSuperAdmin: boolean;
}

/** Resolve only visible menu routes; error, redirect, and hidden routes must not create tabs. */
export function resolveRouteTab({
  pathname,
  dynamicMenuEnabled,
  dynamicMenus,
  staticNav,
  locale,
  isAdmin,
  isSuperAdmin,
}: ResolveRouteTabOptions): TabItem | null {
  if (dynamicMenuEnabled) {
    const menu = dynamicMenus.find(
      (item) =>
        item.type === 'menu' &&
        item.routePath === pathname &&
        item.status === 'normal' &&
        item.showStatus !== 'hide',
    );
    if (!menu?.routePath) return null;
    return {
      key: menu.routePath,
      label: resolveMenuLabel(menu, locale),
      path: menu.routePath,
      closable: true,
      labelRaw: true,
    };
  }

  for (const group of staticNav) {
    if (group.requireSuperAdmin && !isSuperAdmin) continue;
    const child = group.children.find(
      (item) => item.path === pathname && (!item.requireAdmin || isAdmin || isSuperAdmin),
    );
    if (!child) continue;
    return {
      key: child.key,
      label: child.label,
      path: child.path,
      closable: child.path !== '/dashboard',
      labelRaw: false,
    };
  }

  return null;
}
