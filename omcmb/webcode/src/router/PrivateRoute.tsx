import { Navigate, useLocation } from 'react-router-dom';
import { useUserStore } from '@core/store/userStore';
import { useMenuStore } from '@core/store/menuStore';
import { isDynamicMenuEnabled } from '@/components/MenuBootstrap/featureFlag';

interface PrivateRouteProps {
  children: React.ReactNode;
  /** T-0098-P4-02：true 时仅 super_admin 可访问，否则跳 /403。 */
  requireSuperAdmin?: boolean;
  /** true 时仅 admin 角色可访问，否则跳 /403。 */
  requireAdmin?: boolean;
}

// 路径白名单：即使不在用户菜单也可访问。
//   - /dashboard：默认登录后跳转目标，所有角色都应能进
//   - /403：未授权页本身
//   - /login：在 routes 上层，本守卫不会经过；列出仅作文档
//   - /notifications：通知中心既是独立通知管理菜单，也保留顶栏铃铛“查看全部”入口；
//     页面级入口对已登录用户开放，规则/模板/渠道/投递写操作仍由后端 API 权限控制。
const ALWAYS_ALLOWED_PATHS = new Set<string>(['/dashboard', '/403', '/login', '/notifications']);

// 错误页：未登录用户访问这些路径被弹到 /login 时，**不要**把它们当成
// "登录后想去的地方"塞进 state.from —— 否则登录成功会回弹到错误页。
// 双层防御：LoginPage 那边也有 FROM_PATH_BLOCKLIST 做兜底。
const ERROR_PATH_SET = new Set<string>(['/403', '/404']);

/**
 * Drilldown 路由父级映射：key=drilldown 路径前缀，value=必须可见的菜单 path。
 *
 * 背景：详情 / 编辑 / 查看 等 drilldown 页面通常不挂菜单，URL 也不嵌套在菜单父
 * 路径下，导致 `isPathAllowedByMenu` 前缀匹配失败 → 直接 /403（即使设备列表
 * 在菜单里）。这里显式声明 "看到 list 就能进 detail" 的语义，避免改 URL。
 *
 * 添加新 drilldown 路由时，**必须**把它登记到这里，否则 admin 也会被 403。
 */
const DRILLDOWN_PARENT_MAP: Record<string, string> = {
  '/device/detail': '/device/list',
  '/device/ue-detail': '/device/list',
  '/performance/kpi-standard/detail': '/performance/kpi-standard',
};

function buildLoginRedirectState(pathname: string) {
  if (ERROR_PATH_SET.has(pathname)) return undefined;
  return { from: { pathname } };
}

/**
 * 路径前缀允许：用户菜单含父级 path 时，子路径 / detail 子路由也允许。
 * 例：menus 含 '/device/list'，则 '/device/list/xxx' 也允许（参 React Router relative routing）。
 *
 * 此外查 DRILLDOWN_PARENT_MAP：详情 / 编辑等 drilldown 页面的 URL 不嵌套在菜单
 * 父路径下时，仍然按 "菜单父项可见即放行" 处理。
 */
function isPathAllowedByMenu(routePaths: Set<string>, pathname: string): boolean {
  if (routePaths.has(pathname)) return true;
  for (const p of routePaths) {
    if (pathname.startsWith(`${p}/`)) return true;
  }
  for (const [drilldown, parent] of Object.entries(DRILLDOWN_PARENT_MAP)) {
    if ((pathname === drilldown || pathname.startsWith(`${drilldown}/`)) && routePaths.has(parent)) {
      return true;
    }
  }
  return false;
}

/**
 * PrivateRoute wraps protected content.
 *
 * 四层守卫（按序）：
 *   1. 未登录 → /login
 *   2. Token 过期且无 refresh → /login（access token 过期但 refresh 存在时，
 *      http.ts 拦截器自动 silent refresh，不在此层处理）
 *   3. requireSuperAdmin=true 但非超管 → /403
 *      （T-0098-P4-02 super_admin 治理路由组守卫；isSuperAdmin 由后端
 *      source==='builtIn' 派生）
 *   4. 已登录 + 灰度开启 + 菜单已加载 + 当前 path 不在用户菜单 → /403
 *      （仅 VITE_DYNAMIC_MENU=true 且 menuStore.loaded=true 时启用，
 *      避免静态模式 / 首屏未加载完成时误拦截）
 */
export default function PrivateRoute({
  children,
  requireSuperAdmin = false,
  requireAdmin = false,
}: PrivateRouteProps) {
  const { isAuthenticated, accessToken, refreshToken, isTokenExpired, currentUser } =
    useUserStore();
  const menuLoaded = useMenuStore((s) => s.loaded);
  const routePaths = useMenuStore((s) => s.routePaths);
  const location = useLocation();

  // Not authenticated at all → redirect to login
  if (!isAuthenticated) {
    return <Navigate to="/login" state={buildLoginRedirectState(location.pathname)} replace />;
  }

  // Token is expired and no refresh token → force re-login
  if (isTokenExpired() && !refreshToken) {
    return <Navigate to="/login" state={buildLoginRedirectState(location.pathname)} replace />;
  }

  // No access token and no refresh token → force re-login
  if (!accessToken && !refreshToken) {
    return <Navigate to="/login" state={buildLoginRedirectState(location.pathname)} replace />;
  }

  // T-0098-P4-02：super_admin 治理路由组守卫
  if (requireSuperAdmin && !currentUser?.isSuperAdmin) {
    return <Navigate to="/403" replace />;
  }

  if (requireAdmin && currentUser?.role !== 'admin' && !currentUser?.isSuperAdmin) {
    return <Navigate to="/403" replace />;
  }

  // 路径守卫：仅动态模式 + 菜单已加载时启用，避免误判
  //
  // 超管 / admin 整体放行（授权全量）：admin 是系统管理员，应能进所有业务路由，
  // 不受"菜单种子只覆盖部分路由"的限制。非 admin 角色仍按菜单门禁。
  // admin bypass 口径见 frontend-core/utils/routeAccess。
  const isAdminLike =
    currentUser?.role === 'admin' || currentUser?.isSuperAdmin === true;
  if (isDynamicMenuEnabled() && menuLoaded && !isAdminLike) {
    const pathname = location.pathname;
    if (
      !ALWAYS_ALLOWED_PATHS.has(pathname) &&
      !isPathAllowedByMenu(routePaths, pathname)
    ) {
      return <Navigate to="/403" replace />;
    }
  }

  return <>{children}</>;
}
