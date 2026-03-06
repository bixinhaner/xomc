import { Navigate, useLocation } from 'react-router-dom';
import { useUserStore } from '@/store/userStore';

interface PrivateRouteProps {
  children: React.ReactNode;
}

/**
 * PrivateRoute wraps protected content.
 * If the user is not authenticated it redirects to /login,
 * preserving the current location so that after login the user
 * can be redirected back to where they were.
 */
export default function PrivateRoute({ children }: PrivateRouteProps) {
  const isAuthenticated = useUserStore((s) => s.isAuthenticated);
  const location = useLocation();

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  return <>{children}</>;
}
