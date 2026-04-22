import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import {
  Activity,
  LayoutDashboard,
  Cpu,
  AlertTriangle,
  Sliders,
  LogOut,
} from 'lucide-react'
import type { ReactNode } from 'react'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { useUserStore } from '@core/store/userStore'

type NavItem = {
  to: string
  label: string
  icon: ReactNode
  disabled?: boolean
}

const NAV: NavItem[] = [
  { to: '/dashboard', label: '控制台', icon: <LayoutDashboard /> },
  { to: '/devices', label: '设备管理', icon: <Cpu /> },
  { to: '/alarms', label: '告警中心', icon: <AlertTriangle />, disabled: true },
  { to: '/config', label: '配置管理', icon: <Sliders />, disabled: true },
]

export function AppShell() {
  const navigate = useNavigate()
  const user = useUserStore((s) => s.currentUser)
  const clearAuth = useUserStore((s) => s.clearAuth)

  const onLogout = () => {
    clearAuth()
    navigate('/login')
  }

  return (
    <div className="flex min-h-screen bg-background">
      <aside className="w-60 shrink-0 border-r border-border/60 bg-card/40">
        <div className="flex h-14 items-center gap-2 border-b border-border/60 px-4">
          <Activity className="size-5 text-primary" />
          <span className="font-semibold tracking-wide">OMC · v2</span>
          <span className="ml-auto rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">
            alpha
          </span>
        </div>
        <nav className="p-2">
          {NAV.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              aria-disabled={item.disabled}
              onClick={(e) => {
                if (item.disabled) e.preventDefault()
              }}
              className={({ isActive }) =>
                cn(
                  'mb-0.5 flex items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors',
                  '[&_svg]:size-4',
                  item.disabled
                    ? 'cursor-not-allowed text-muted-foreground/50'
                    : isActive
                      ? 'bg-primary/10 text-primary font-medium'
                      : 'text-foreground/80 hover:bg-muted hover:text-foreground'
                )
              }
              end
            >
              {item.icon}
              <span>{item.label}</span>
              {item.disabled && (
                <span className="ml-auto text-[10px] text-muted-foreground/60">
                  TBD
                </span>
              )}
            </NavLink>
          ))}
        </nav>
      </aside>

      <div className="flex flex-1 flex-col">
        <header className="flex h-14 items-center justify-end gap-3 border-b border-border/60 bg-card/30 px-6 backdrop-blur">
          <span className="text-sm text-muted-foreground">
            {user?.displayName || user?.username || '未登录'}
          </span>
          <Button size="sm" variant="ghost" onClick={onLogout}>
            <LogOut /> 退出
          </Button>
        </header>

        <main className="flex-1 overflow-auto">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
