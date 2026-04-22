import { Navigate, createBrowserRouter } from 'react-router-dom'

import { LoginPage } from '@/pages/login'
import { DashboardPage } from '@/pages/dashboard'
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
    path: '/dashboard',
    element: (
      <Protected>
        <DashboardPage />
      </Protected>
    ),
  },
  { path: '*', element: <Navigate to="/dashboard" replace /> },
])
