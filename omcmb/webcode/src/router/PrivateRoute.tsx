import { Navigate, useLocation } from 'react-router-dom';
import { useUserStore } from '@core/store/userStore';

interface PrivateRouteProps {
  children: React.ReactNode;
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
 */
export default function PrivateRoute({ children }: PrivateRouteProps) {
  const { isAuthenticated, accessToken, refreshToken, isTokenExpired } = useUserStore();
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

  return <>{children}</>;
}
