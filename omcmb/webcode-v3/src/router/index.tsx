import { Navigate, createBrowserRouter } from 'react-router-dom'
import type { ReactNode } from 'react'

import { BridgeShell } from '@/components/shell/BridgeShell'
import { LoginPage } from '@/pages/login'
import { BridgePage } from '@/pages/bridge'
import { FleetPage } from '@/pages/fleet'
import { AlarmsPage } from '@/pages/alarms'
import { TopologyPage } from '@/pages/topology'
import { PerformancePage } from '@/pages/performance'
import { MMLPage } from '@/pages/mml'
import { LogsPage } from '@/pages/logs'
import { ConfigPage } from '@/pages/config'
import { SoftwarePage } from '@/pages/software'
import { BackupPage } from '@/pages/backup'
import { OpsPage } from '@/pages/ops'
import { MRPage } from '@/pages/mr'
import { ReportsPage } from '@/pages/reports'
import { FilesPage } from '@/pages/files'
import { LicensePage } from '@/pages/license'
import { SystemPage } from '@/pages/system'

import { useUserStore } from '@core/store/userStore'

function Protected({ children }: { children: ReactNode }) {
  const isAuthed = useUserStore((s) => s.isAuthenticated)
  if (!isAuthed) return <Navigate to="/login" replace />
  return <>{children}</>
}

export const router = createBrowserRouter([
  { path: '/', element: <Navigate to="/bridge" replace /> },
  { path: '/login', element: <LoginPage /> },
  {
    element: (
      <Protected>
        <BridgeShell />
      </Protected>
    ),
    children: [
      { path: '/bridge', element: <BridgePage /> },
      { path: '/fleet', element: <FleetPage /> },
      { path: '/alarms', element: <AlarmsPage /> },
      { path: '/topology', element: <TopologyPage /> },
      { path: '/performance', element: <PerformancePage /> },
      { path: '/mml', element: <MMLPage /> },
      { path: '/config', element: <ConfigPage /> },
      { path: '/software', element: <SoftwarePage /> },
      { path: '/backup', element: <BackupPage /> },
      { path: '/ops', element: <OpsPage /> },
      { path: '/mr', element: <MRPage /> },
      { path: '/reports', element: <ReportsPage /> },
      { path: '/files', element: <FilesPage /> },
      { path: '/logs', element: <LogsPage /> },
      { path: '/license', element: <LicensePage /> },
      { path: '/system', element: <SystemPage /> },
    ],
  },
  { path: '*', element: <Navigate to="/bridge" replace /> },
], { basename: import.meta.env.BASE_URL?.replace(/\/$/, '') || undefined })
