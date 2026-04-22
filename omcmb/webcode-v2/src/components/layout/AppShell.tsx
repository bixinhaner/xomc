import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import {
  Activity,
  LayoutDashboard,
  Cpu,
  AlertTriangle,
  Sliders,
  Package,
  FileStack,
  KeySquare,
  FolderOpen,
  FileBarChart,
  Settings,
  Terminal,
  LineChart,
  Radio,
  Globe,
  Wrench,
  ScrollText,
  LogOut,
} from 'lucide-react'
import type { ReactNode } from 'react'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { useUserStore } from '@core/store/userStore'

type NavItem = { to: string; label: string; icon: ReactNode }
type NavGroup = { label: string; items: NavItem[] }

const NAV_GROUPS: NavGroup[] = [
  {
    label: '监控',
    items: [
      { to: '/dashboard', label: '控制台', icon: <LayoutDashboard /> },
      { to: '/devices', label: '设备管理', icon: <Cpu /> },
      { to: '/alarms', label: '告警中心', icon: <AlertTriangle /> },
      { to: '/topology', label: '拓扑视图', icon: <Globe /> },
    ],
  },
  {
    label: '运维',
    items: [
      { to: '/config', label: '配置管理', icon: <Sliders /> },
      { to: '/mml', label: 'MML 脚本', icon: <Terminal /> },
      { to: '/software', label: '软件版本', icon: <Package /> },
      { to: '/backup', label: '备份管理', icon: <FileStack /> },
      { to: '/ops', label: '运维工具箱', icon: <Wrench /> },
    ],
  },
  {
    label: '数据',
    items: [
      { to: '/performance', label: '性能管理', icon: <LineChart /> },
      { to: '/mr', label: '测量报告', icon: <Radio /> },
      { to: '/reports', label: '报表中心', icon: <FileBarChart /> },
      { to: '/files', label: '文件管理', icon: <FolderOpen /> },
    ],
  },
  {
    label: '系统',
    items: [
      { to: '/logs', label: '系统日志', icon: <ScrollText /> },
      { to: '/license', label: '许可证', icon: <KeySquare /> },
      { to: '/system', label: '系统管理', icon: <Settings /> },
    ],
  },
]

export function AppShell() {
  const navigate = useNavigate()
  const user = useUserStore((s) => s.currentUser)
  const clearAuth = useUserStore((s) => s.clearAuth)

  const onLogout = () => {
    clearAuth()
    navigate('/login')
  }

  const userLabel =
    (user as { displayName?: string; username?: string; userName?: string } | null)
      ?.displayName ||
    (user as { displayName?: string; username?: string; userName?: string } | null)
      ?.username ||
    user?.userName ||
    '未登录'

  return (
    <div className="flex min-h-screen bg-background">
      <aside className="w-60 shrink-0 overflow-y-auto border-r border-border/60 bg-card/40">
        <div className="sticky top-0 z-10 flex h-14 items-center gap-2 border-b border-border/60 bg-card/40 px-4 backdrop-blur">
          <Activity className="size-5 text-primary" />
          <span className="font-semibold tracking-wide">OMC · v2</span>
          <span className="ml-auto rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">
            alpha
          </span>
        </div>
        <nav className="p-2 pb-6">
          {NAV_GROUPS.map((group) => (
            <div key={group.label} className="mb-4">
              <div className="px-3 pb-1 pt-2 text-[11px] uppercase tracking-wider text-muted-foreground/60">
                {group.label}
              </div>
              {group.items.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end
                  className={({ isActive }) =>
                    cn(
                      'mb-0.5 flex items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors',
                      '[&_svg]:size-4',
                      isActive
                        ? 'bg-primary/10 text-primary font-medium'
                        : 'text-foreground/80 hover:bg-muted hover:text-foreground'
                    )
                  }
                >
                  {item.icon}
                  <span>{item.label}</span>
                </NavLink>
              ))}
            </div>
          ))}
        </nav>
      </aside>

      <div className="flex flex-1 flex-col">
        <header className="sticky top-0 z-10 flex h-14 items-center justify-end gap-3 border-b border-border/60 bg-card/40 px-6 backdrop-blur">
          <span className="text-sm text-muted-foreground">{userLabel}</span>
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
