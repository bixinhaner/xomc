import type { TabItem } from '@core/store/tabStore';
import type { Menu } from '@core/types/menu';
import { resolveMenuLabel } from '@core/types/menu';
import { isSystemLicensePath } from '@core/utils/systemLicenseAccess';
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
  // License 恢复面是功能性页面（licenseOnlyMode 下是唯一可达路由），但其动态菜单
  // 项 show_status=hide（默认不进侧边栏）。若不在此显式建 tab，落地 /license 时
  // 下方动态分支因 showStatus!=='hide' 过滤掉它 → 返回 null → tab 条停在"仪表板"。
  if (isSystemLicensePath(pathname)) {
    return {
      key: pathname,
      label: pathname === '/license/history' ? 'systemLicense.history.title' : 'nav.systemLicense',
      path: pathname,
      closable: true,
      labelRaw: false,
    };
  }
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
