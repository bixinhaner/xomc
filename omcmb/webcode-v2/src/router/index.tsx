import { Navigate, createBrowserRouter } from 'react-router-dom'

import { AppShell } from '@/components/layout/AppShell'
import { LoginPage } from '@/pages/login'
import { DashboardPage } from '@/pages/dashboard'
import { DevicesPage } from '@/pages/devices'
import { AlarmsPage } from '@/pages/alarms'
import { TopologyPage } from '@/pages/topology'
import { ConfigPage } from '@/pages/config'
import { MMLPage } from '@/pages/mml'
import { SoftwarePage } from '@/pages/software'
import { BackupPage } from '@/pages/backup'
import { OpsPage } from '@/pages/ops'
import { PerformancePage } from '@/pages/performance'
import { MRPage } from '@/pages/mr'
import { ReportsPage } from '@/pages/reports'
import { FilesPage } from '@/pages/files'
import { LogsPage } from '@/pages/logs'
import { LicensePage } from '@/pages/license'
import { SystemPage } from '@/pages/system'
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
      { path: '/alarms', element: <AlarmsPage /> },
      { path: '/topology', element: <TopologyPage /> },
      { path: '/config', element: <ConfigPage /> },
      { path: '/mml', element: <MMLPage /> },
      { path: '/software', element: <SoftwarePage /> },
      { path: '/backup', element: <BackupPage /> },
      { path: '/ops', element: <OpsPage /> },
      { path: '/performance', element: <PerformancePage /> },
      { path: '/mr', element: <MRPage /> },
      { path: '/reports', element: <ReportsPage /> },
      { path: '/files', element: <FilesPage /> },
      { path: '/logs', element: <LogsPage /> },
      { path: '/license', element: <LicensePage /> },
      { path: '/system', element: <SystemPage /> },
    ],
  },
  { path: '*', element: <Navigate to="/dashboard" replace /> },
], { basename: import.meta.env.BASE_URL?.replace(/\/$/, '') || undefined })
