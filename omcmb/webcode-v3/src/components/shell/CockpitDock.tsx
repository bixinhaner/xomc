import { NavLink } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { MODULES } from '@/router/navConfig'

// 数据驱动：每个模块取首个路由作为停靠入口（与侧栏/路由唯一事实源一致）。
const ITEMS = MODULES.map((m) => ({
  key: m.key,
  to: m.routes[0]?.path ?? '/bridge',
  label: m.label.split(' ')[0],
  icon: m.icon,
}))

export function CockpitDock() {
  return (
    <div className="pointer-events-none absolute bottom-0 left-0 right-0 z-30 flex justify-center">
      <div className="pointer-events-auto relative">
        <div className="absolute -bottom-12 left-1/2 -translate-x-1/2 h-32 w-[1100px] max-w-[95vw] rounded-[50%] bg-cyan-400/10 blur-2xl" />
        <div className="relative glass-strong rounded-t-2xl border-b-0 px-6 pt-3">
          <div className="cockpit-dock">
            {ITEMS.map((it, idx) => (
              <NavLink key={it.key} to={it.to} className="relative">
                {({ isActive }) => (
                  <div
                    className={cn('dock-cell', isActive && 'active')}
                    style={{ transform: `translateY(${arcOffset(idx, ITEMS.length)}px)` }}
                  >
                    <span className="hex-bg hex" />
                    <span className="icon-wrap [&_svg]:size-5">{it.icon}</span>
                    <span className="label">{it.label}</span>
                  </div>
                )}
              </NavLink>
            ))}
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
