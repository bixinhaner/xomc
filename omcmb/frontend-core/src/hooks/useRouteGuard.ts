import { useUserStore } from '../store/userStore';
import { useMenuStore } from '../store/menuStore';
import { useUserMenus } from './api/useMenus';
import { isRouteAllowed, isModuleVisible } from '../utils/routeAccess';

/**
 * 三皮肤统一的路由门禁 hook（v2/v3 用；v1 在 PrivateRoute 内联同等逻辑）。
 *
 * 职责：
 *   1. 拉用户菜单填充 menuStore（useUserMenus 内部按 userId enabled，未登录不发请求）。
 *   2. 按 {role/超管 bypass + 模块级菜单门禁} 判定当前路由是否可访问（见 routeAccess）。
 *
 * @param pathname        当前路由（含皮肤 base，如 /v2/devices）——由皮肤侧 useLocation 传入，
 *                        避免 frontend-core 直接依赖 react-router。
 * @param dynamicEnabled  该皮肤是否开启动态菜单门禁（读各自 VITE_DYNAMIC_MENU）。
 * @returns 是否放行。admin/超管恒 true；门禁未开 / 菜单未加载亦 true（不误拦）。
 */
export function useRouteAccessGuard(pathname: string, dynamicEnabled: boolean): boolean {
  // 已登录时后台拉用户菜单（stale-while-revalidate）填充 menuStore；admin 走 bypass 不依赖它。
  useUserMenus();
  const role = useUserStore((s) => s.currentUser?.role);
  const isSuperAdmin = useUserStore((s) => s.currentUser?.isSuperAdmin);
  const menuLoaded = useMenuStore((s) => s.loaded);
  const routePaths = useMenuStore((s) => s.routePaths);
  return isRouteAllowed(pathname, {
    role,
    isSuperAdmin,
    routePaths,
    dynamicEnabled,
    menuLoaded,
  });
}

/**
 * 侧边栏模块可见性判定器（v2/v3 用，对齐 v1 菜单驱动侧栏）。
 *
 * 返回一个 `(pathname) => boolean` 谓词：动态菜单门禁开启且菜单已加载时，按模块级
 * 菜单可见性过滤（仅显示用户菜单含的模块）；否则全显示（不误隐）。
 * 不做 admin bypass——与 v1 一致，admin 侧栏也是菜单驱动（curated），三皮肤侧栏一致。
 */
export function useModuleVisibility(dynamicEnabled: boolean): (pathname: string) => boolean {
  const menuLoaded = useMenuStore((s) => s.loaded);
  const routePaths = useMenuStore((s) => s.routePaths);
  return (pathname: string) => {
    if (!dynamicEnabled || !menuLoaded) return true;
    return isModuleVisible(pathname, routePaths);
  };
}
