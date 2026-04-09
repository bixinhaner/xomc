import { RouterProvider } from 'react-router-dom';
import { App as AntApp, Spin } from 'antd';
import QueryProvider from './providers/QueryProvider';
import ThemeProvider from './providers/ThemeProvider';
import LocaleProvider from './providers/LocaleProvider';
import router from './router';
import { useUserStore } from './store/userStore';

/**
 * HydrationGate delays rendering the router until zustand persist
 * has rehydrated auth state from localStorage. Without this gate,
 * PrivateRoute reads the initial defaults (isAuthenticated=false)
 * and immediately redirects to /login on every browser refresh.
 */
function HydrationGate({ children }: { children: React.ReactNode }) {
  const hasHydrated = useUserStore((s) => s._hasHydrated);

  if (!hasHydrated) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <Spin size="large" />
      </div>
    );
  }

  return <>{children}</>;
}

/**
 * App is the root composition component.
 *
 * Provider nesting order (outermost → innermost):
 *   QueryProvider       — React Query cache and client
 *   ThemeProvider       — Ant Design theme + data-theme attribute on <html>
 *   LocaleProvider      — react-intl + Ant Design locale
 *   AntApp              — Ant Design App component for message/modal/notification support
 *   HydrationGate       — Waits for zustand persist rehydration before rendering routes
 *   RouterProvider      — React Router v6 with all routes
 */
export default function App() {
  return (
    <QueryProvider>
      <ThemeProvider>
        <LocaleProvider>
          <AntApp>
            <HydrationGate>
              <RouterProvider router={router} />
            </HydrationGate>
          </AntApp>
        </LocaleProvider>
      </ThemeProvider>
    </QueryProvider>
  );
}
