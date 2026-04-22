import { Navigate, createBrowserRouter } from 'react-router-dom'

import { AppShell } from '@/components/layout/AppShell'
import { LoginPage } from '@/pages/login'
import { DashboardPage } from '@/pages/dashboard'
import { DevicesPage } from '@/pages/devices'
import { useUserStore } from '@core/store/userStore'

function Protected({ children }: { children: React.ReactNode }) {
  const isAuthed = useUserStore((s) => s.isAuthenticated)
  if (!isAuthed) return <Navigate to="/login" replace />
  return <>{children}</>
}

export const router = createBrowserRouter([
  { path: '/', element: <Navigate to="/dashboard" replace /> },
  { path: '/login', element: <LoginPage /> },
  {
    element: (
      <Protected>
        <AppShell />
      </Protected>
    ),
    children: [
      { path: '/dashboard', element: <DashboardPage /> },
      { path: '/devices', element: <DevicesPage /> },
    ],
  },
  { path: '*', element: <Navigate to="/dashboard" replace /> },
])
