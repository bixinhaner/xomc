import { NavLink } from 'react-router-dom'
import { Bot } from 'lucide-react'
import { cn } from '@/lib/utils'
import { MODULES } from '@/router/navConfig'
import { useModuleVisibility } from '@core/hooks/useRouteGuard'
import { useT } from '@/hooks/useT'

// 动态菜单门禁开关（与 v1/v2 对齐）。
const DYNAMIC_MENU = import.meta.env.VITE_DYNAMIC_MENU === 'true'

interface CockpitDockProps {
  agentVisible?: boolean
  agentOpen?: boolean
  onAgentToggle?: () => void
}

export function CockpitDock({ agentVisible = false, agentOpen = false, onAgentToggle }: CockpitDockProps) {
  const t = useT()
  // 数据驱动：每个模块取首个路由作为停靠入口；按菜单可见性过滤（对齐 v1 菜单驱动侧栏）。
  const moduleVisible = useModuleVisibility(DYNAMIC_MENU)
  const items = MODULES.filter((m) => moduleVisible(m.routes[0]?.path ?? '/dashboard')).map((m) => ({
    key: m.key,
    to: m.routes[0]?.path ?? '/dashboard',
    label: m.label.split(' ')[0],
    icon: m.icon,
  }))
  return (
    <div className="pointer-events-none absolute bottom-0 left-0 right-0 z-30 flex justify-center">
      <div className="pointer-events-auto relative">
        <div className="absolute -bottom-12 left-1/2 -translate-x-1/2 h-32 w-[1100px] max-w-[95vw] rounded-[50%] bg-cyan-400/10 blur-2xl" />
        <div className="relative glass-strong rounded-t-2xl border-b-0 px-6 pt-3">
          <div className="cockpit-dock">
            {items.map((it, idx) => (
              <NavLink key={it.key} to={it.to} className="relative">
                {({ isActive }) => (
                  <div
                    className={cn('dock-cell', isActive && 'active')}
                    style={{ transform: `translateY(${arcOffset(idx, items.length)}px)` }}
                  >
                    <span className="hex-bg hex" />
                    <span className="icon-wrap [&_svg]:size-5">{it.icon}</span>
                    <span className="label">{it.label}</span>
                  </div>
                )}
              </NavLink>
            ))}
            {agentVisible && (
              <button
                type="button"
                className="relative"
                onClick={onAgentToggle}
                aria-pressed={agentOpen}
                aria-label={t('agent.open')}
                title={t('agent.open')}
              >
                <div
                  className={cn('dock-cell', agentOpen && 'active')}
                  style={{ transform: `translateY(${arcOffset(items.length, items.length + 1)}px)` }}
                >
                  <span className="hex-bg hex" />
                  <span className="icon-wrap">
                    <Bot className="size-5" />
                  </span>
                  <span className="label">{t('agent.short')}</span>
                </div>
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

function arcOffset(i: number, total: number) {
  const center = (total - 1) / 2
  const t = (i - center) / center
  return Math.abs(t) * 12
}
