import { Navigate, useLocation } from 'react-router-dom';
import { useUserStore } from '@core/store/userStore';
import { useMenuStore } from '@core/store/menuStore';
import { isDynamicMenuEnabled } from '@/components/MenuBootstrap/featureFlag';

interface PrivateRouteProps {
  children: React.ReactNode;
}

// 路径白名单：即使不在用户菜单也可访问。
//   - /dashboard：默认登录后跳转目标，所有角色都应能进
//   - /403：未授权页本身
//   - /login：在 routes 上层，本守卫不会经过；列出仅作文档
const ALWAYS_ALLOWED_PATHS = new Set<string>(['/dashboard', '/403', '/login']);

/**
 * 路径前缀允许：用户菜单含父级 path 时，子路径 / detail 子路由也允许。
 * 例：menus 含 '/device/list'，则 '/device/list/xxx' 也允许（参 React Router relative routing）。
 */
function isPathAllowedByMenu(routePaths: Set<string>, pathname: string): boolean {
  if (routePaths.has(pathname)) return true;
  for (const p of routePaths) {
    if (pathname.startsWith(`${p}/`)) return true;
  }
  return false;
}

/**
 * PrivateRoute wraps protected content.
 *
 * 三层守卫：
 *   1. 未登录 → /login
 *   2. Token 过期且无 refresh → /login
 *   3. 已登录 + 灰度开启 + 菜单已加载 + 当前 path 不在用户菜单 → /403
 *
 * 第 3 步仅在 VITE_DYNAMIC_MENU=true 且 menuStore.loaded=true 时启用，
 * 避免静态模式 / 首屏未加载完成时误拦截。
 */
export default function PrivateRoute({ children }: PrivateRouteProps) {
  const { isAuthenticated, accessToken, refreshToken, isTokenExpired } = useUserStore();
  const menuLoaded = useMenuStore((s) => s.loaded);
  const routePaths = useMenuStore((s) => s.routePaths);
  const location = useLocation();

  // Not authenticated at all → redirect to login
  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // Token is expired and no refresh token → force re-login
  if (isTokenExpired() && !refreshToken) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // No access token and no refresh token → force re-login
  if (!accessToken && !refreshToken) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // 路径守卫：仅动态模式 + 菜单已加载时启用，避免误判
  if (isDynamicMenuEnabled() && menuLoaded) {
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
