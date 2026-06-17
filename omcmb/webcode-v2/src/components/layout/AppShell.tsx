import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { Activity, Clock, LogOut, ShieldAlert } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { useUserStore } from '@core/store/userStore'
import { useAppStore } from '@core/store/appStore'
import { usePublicOmcName, resolveOmcName } from '@core/hooks/api/useOmcName'
import { useSystemTimezone } from '@core/hooks/api/useSystemTimezone'
import { useSystemClock } from '@core/hooks/useSystemClock'
import { useRouteAccessGuard, useModuleVisibility } from '@core/hooks/useRouteGuard'
import { MODULES, SECTIONS } from '@/router/navConfig'

// 动态菜单门禁开关（与 v1 webcode 对齐）。admin/超管恒放行，非 admin 按模块级菜单门禁。
const DYNAMIC_MENU = import.meta.env.VITE_DYNAMIC_MENU === 'true'

/**
 * 顶部只读时钟（#459 子单 D）：展示「系统时区 + 实时当前时间」，每秒按系统时区刷新，
 * 不可点切换。系统时区来自 sys_configs basic/timezoneCode（登录后写入 appStore）。
 */
function SystemClockBadge() {
  const { timezoneLabel, dateTime } = useSystemClock()
  return (
    <div
      className="flex select-none items-center gap-1.5 rounded-md border border-border/60 bg-muted/40 px-2.5 py-1 text-xs text-muted-foreground"
      title={`当前系统时区: ${timezoneLabel}`}
    >
      <Clock className="size-3.5" />
      <span className="text-muted-foreground/70">{timezoneLabel}</span>
      <span className="font-medium tabular-nums text-foreground">{dateTime}</span>
    </div>
  )
}

function Forbidden() {
  const navigate = useNavigate()
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 p-10 text-center">
      <ShieldAlert className="size-12 text-destructive/70" />
      <div className="text-lg font-semibold">403 · 无权限访问</div>
      <p className="max-w-md text-sm text-muted-foreground">
        抱歉，您没有权限访问此页面。如需访问，请联系系统管理员为您增加菜单权限。
      </p>
      <Button size="sm" variant="outline" onClick={() => navigate('/dashboard')}>
        返回控制台
      </Button>
    </div>
  )
}

export function AppShell() {
  const navigate = useNavigate()
  const location = useLocation()
  const user = useUserStore((s) => s.currentUser)
  const clearAuth = useUserStore((s) => s.clearAuth)
  // 路由门禁（admin 恒放行）：拉用户菜单 + 判定当前路由可访问性
  const allowed = useRouteAccessGuard(location.pathname, DYNAMIC_MENU)
  // 侧栏模块按菜单可见性过滤（对齐 v1 菜单驱动侧栏；admin 也按菜单 curated，不 bypass）
  const moduleVisible = useModuleVisibility(DYNAMIC_MENU)

  const onLogout = () => {
    clearAuth()
    navigate('/login')
  }

  const userLabel = user?.displayName || user?.username || '未登录'

  // 左上角品牌标题跟随「OMC 名称」配置（公开通道），空回退 'OMC · v2'。
  const { omcName } = usePublicOmcName()
  const storedOmcName = useAppStore((s) => s.omcName)
  const brandTitle = resolveOmcName(omcName ?? storedOmcName, 'OMC · v2')

  // 系统时区（#459 子单 D）：登录后拉取写入 appStore，供顶部时钟 + 时间筛选/显示。
  useSystemTimezone()

  const linkClass = ({ isActive }: { isActive: boolean }) =>
    cn(
      'mb-0.5 flex items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors [&_svg]:size-4',
      isActive
        ? 'bg-primary/10 text-primary font-medium'
        : 'text-foreground/80 hover:bg-muted hover:text-foreground'
    )
  const subLinkClass = ({ isActive }: { isActive: boolean }) =>
    cn(
      'mb-0.5 ml-7 flex items-center rounded-md px-3 py-1.5 text-[13px] transition-colors',
      isActive
        ? 'bg-primary/10 text-primary font-medium'
        : 'text-foreground/65 hover:bg-muted hover:text-foreground'
    )

  return (
    <div className="flex min-h-screen bg-background">
      <aside className="w-60 shrink-0 overflow-y-auto border-r border-border/60 bg-card/40">
        <div className="sticky top-0 z-10 flex h-14 items-center gap-2 border-b border-border/60 bg-card/40 px-4 backdrop-blur">
          <Activity className="size-5 text-primary" />
          <span className="font-semibold tracking-wide">{brandTitle}</span>
          <span className="ml-auto rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">
            alpha
          </span>
        </div>
        <nav className="p-2 pb-6">
          {SECTIONS.map((section) => {
            const mods = MODULES.filter((m) => {
              if (m.section !== section) return false
              // 与 v1 一致：模块全部路由 hidden 时不在侧栏出现（无可见叶子项）。
              const primary = m.routes.find((r) => !r.hidden)
              if (!primary) return false
              return moduleVisible(primary.path)
            })
            if (mods.length === 0) return null
            return (
              <div key={section} className="mb-4">
                <div className="px-3 pb-1 pt-2 text-[11px] uppercase tracking-wider text-muted-foreground/60">
                  {section}
                </div>
                {mods.map((m) => {
                  const visible = m.routes.filter((r) => !r.hidden)
                  const primary = visible[0] ?? m.routes[0]
                  // 单路由模块：直接一个链接；多路由模块：模块名 + 缩进子项
                  if (visible.length <= 1) {
                    return (
                      <NavLink key={m.key} to={primary.path} end className={linkClass}>
                        {m.icon}
                        <span>{m.label}</span>
                      </NavLink>
                    )
                  }
                  return (
                    <div key={m.key} className="mb-1">
                      <NavLink to={primary.path} end className={linkClass}>
                        {m.icon}
                        <span>{m.label}</span>
                      </NavLink>
                      {visible.slice(1).map((r) => (
                        <NavLink key={r.path} to={r.path} end className={subLinkClass}>
                          <span>{r.label}</span>
                        </NavLink>
                      ))}
                    </div>
                  )
                })}
              </div>
            )
          })}
        </nav>
      </aside>

      <div className="flex flex-1 flex-col">
        <header className="sticky top-0 z-10 flex h-14 items-center justify-end gap-3 border-b border-border/60 bg-card/40 px-6 backdrop-blur">
          <SystemClockBadge />
          <span className="text-sm text-muted-foreground">{userLabel}</span>
          <Button size="sm" variant="ghost" onClick={onLogout}>
            <LogOut /> 退出
          </Button>
        </header>

        <main className="flex-1 overflow-auto">
          {allowed ? <Outlet /> : <Forbidden />}
        </main>
      </div>
    </div>
  )
}
