import { Navigate, createBrowserRouter } from 'react-router-dom'
import type { ReactNode } from 'react'

import { BridgeShell } from '@/components/shell/BridgeShell'
import { LoginPage } from '@/pages/login'
import { useUserStore } from '@core/store/userStore'
import { ALL_ROUTES } from './navConfig'

function Protected({ children }: { children: ReactNode }) {
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
        <BridgeShell />
      </Protected>
    ),
    // 路由清单数据驱动（见 ./navConfig）。各模块对齐 v1 子路由只往清单追加。
    children: ALL_ROUTES.map((r) => ({ path: r.path, element: r.element })),
  },
  { path: '*', element: <Navigate to="/dashboard" replace /> },
], { basename: import.meta.env.BASE_URL?.replace(/\/$/, '') || undefined })
