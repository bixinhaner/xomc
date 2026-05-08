import { Navigate, useLocation } from 'react-router-dom';
import { useUserStore } from '@core/store/userStore';

interface PrivateRouteProps {
  children: React.ReactNode;
  /** T-0098-P4-02：true 时仅 super_admin 可访问，否则跳 /403。 */
  requireSuperAdmin?: boolean;
}

/**
 * PrivateRoute wraps protected content.
 * If the user is not authenticated it redirects to /login,
 * preserving the current location so that after login the user
 * can be redirected back to where they were.
 *
 * Token expiration is also checked — if there is no valid access token
 * and no refresh token available, the user is redirected to login.
 * When the access token is expired but a refresh token exists, the
 * request interceptor in http.ts will handle silent refresh.
 *
 * 当 requireSuperAdmin=true 时，进一步校验当前用户的 isSuperAdmin（后端
 * source==='builtIn' 派生）。非超管访问 /product/* 治理菜单 → 重定向 /403。
 */
export default function PrivateRoute({ children, requireSuperAdmin = false }: PrivateRouteProps) {
  const { isAuthenticated, accessToken, refreshToken, isTokenExpired, currentUser } = useUserStore();
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

  // T-0098-P4-02：super_admin 治理路由组守卫
  if (requireSuperAdmin && !currentUser?.isSuperAdmin) {
    return <Navigate to="/403" replace />;
  }

  return <>{children}</>;
}
