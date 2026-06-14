import { useEffect, useRef } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'

import { HUDStatusBar } from '@/components/shell/HUDStatusBar'
import { CockpitDock } from '@/components/shell/CockpitDock'
import { MiniRadar } from '@/components/shell/MiniRadar'
import { TelemetryStream } from '@/components/shell/TelemetryStream'
import { cn } from '@/lib/utils'
import { MODULES } from '@/router/navConfig'

// 当前模块的子路由横向导航（v1 子路由全量对齐后，HUD 下用上下文子导航暴露子页）。
function SubNav() {
  const { pathname } = useLocation()
  const seg = '/' + (pathname.split('/')[1] || '')
  const mod = MODULES.find((m) => m.routes.some((r) => r.path === seg || r.path.startsWith(seg + '/')))
  if (!mod) return null
  const visible = mod.routes.filter((r) => !r.hidden)
  if (visible.length <= 1) return null
  return (
    <div className="mb-2 flex flex-wrap gap-1.5">
      {visible.map((r) => (
        <NavLink
          key={r.path}
          to={r.path}
          end
          className={({ isActive }) =>
            cn(
              'rounded border px-2.5 py-1 font-mono text-[11px] tracking-wide transition-colors',
              isActive
                ? 'border-cyan-400/60 bg-cyan-400/15 text-cyan-100'
                : 'border-cyan-500/15 bg-cyan-500/5 text-cyan-300/65 hover:border-cyan-400/40 hover:text-cyan-200'
            )
          }
        >
          {r.label}
        </NavLink>
      ))}
    </div>
  )
}

export function BridgeShell() {
  const root = useRef<HTMLDivElement>(null)

  // 鼠标光斑（CSS 变量驱动）
  useEffect(() => {
    const el = root.current
    if (!el) return
    const handler = (e: MouseEvent) => {
      el.style.setProperty('--mx', `${e.clientY}px`)
      el.style.setProperty('--my', `${e.clientX}px`)
    }
    window.addEventListener('mousemove', handler)
    return () => window.removeEventListener('mousemove', handler)
  }, [])

  return (
    <div ref={root} className="cursor-spot relative h-screen w-screen overflow-hidden">
      {/* 背景层 */}
      <div className="starforge-backdrop">
        <div className="starforge-stars" />
      </div>

      {/* 主层 */}
      <div className="relative z-10 flex h-full w-full flex-col">
        <HUDStatusBar />

        {/* 主体三栏：左饰条 + 内容 + 右遥测 */}
        <div className="grid flex-1 min-h-0 grid-cols-[64px_1fr_320px] gap-3 px-4 pt-3">
          {/* 左侧饰条：垂直信号 + 雷达 */}
          <aside className="relative flex flex-col items-center justify-between py-3 pb-28">
            <div className="flex flex-col items-center gap-3">
              <span className="font-mono text-[9px] tracking-[0.3em] text-cyan-300/60 [writing-mode:vertical-rl]">
                CHANNEL · 7547
              </span>
              <div className="flex flex-col gap-1">
                {[
                  '#00f0ff',
                  '#a855f7',
                  '#00ff88',
                  '#ffaa00',
                  '#ff00aa',
                  '#5b9eff',
                  '#00f0ff',
                ].map((c, i) => (
                  <span
                    key={i}
                    className="block h-1.5 w-3 rounded-sm animate-breathe"
                    style={{
                      background: c,
                      boxShadow: `0 0 8px ${c}`,
                      animationDelay: `${i * 0.18}s`,
                    }}
                  />
                ))}
              </div>
            </div>

            <MiniRadar size={120} />
          </aside>

          {/* 中央内容 */}
          <main className="relative flex min-h-0 flex-col overflow-hidden pb-28">
            <SubNav />
            <div className="min-h-0 flex-1 overflow-auto">
              <Outlet />
            </div>
          </main>

          {/* 右侧遥测 */}
          <aside className="min-h-0 pb-28">
            <TelemetryStream />
          </aside>
        </div>

        <CockpitDock />
      </div>
    </div>
  )
}
