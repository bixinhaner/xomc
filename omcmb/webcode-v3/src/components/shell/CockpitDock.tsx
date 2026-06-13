import { NavLink } from 'react-router-dom'
import {
  LayoutDashboard,
  Cpu,
  AlertTriangle,
  Globe,
  LineChart,
  Terminal,
  Sliders,
  Package,
  FileStack,
  Wrench,
  Radio,
  FileBarChart,
  FolderOpen,
  ScrollText,
  KeySquare,
  Settings,
} from 'lucide-react'
import { cn } from '@/lib/utils'

type Item = { to: string; label: string; icon: React.ReactNode }

const ITEMS: Item[] = [
  { to: '/bridge', label: 'BRIDGE', icon: <LayoutDashboard className="size-5" /> },
  { to: '/fleet', label: 'FLEET', icon: <Cpu className="size-5" /> },
  { to: '/alarms', label: 'ALARM', icon: <AlertTriangle className="size-5" /> },
  { to: '/topology', label: 'TOPO', icon: <Globe className="size-5" /> },
  { to: '/performance', label: 'PERF', icon: <LineChart className="size-5" /> },
  { to: '/mml', label: 'MML', icon: <Terminal className="size-5" /> },
  { to: '/config', label: 'CONF', icon: <Sliders className="size-5" /> },
  { to: '/software', label: 'SOFT', icon: <Package className="size-5" /> },
  { to: '/backup', label: 'BACK', icon: <FileStack className="size-5" /> },
  { to: '/ops', label: 'OPS', icon: <Wrench className="size-5" /> },
  { to: '/mr', label: 'MR', icon: <Radio className="size-5" /> },
  { to: '/reports', label: 'RPT', icon: <FileBarChart className="size-5" /> },
  { to: '/files', label: 'FILE', icon: <FolderOpen className="size-5" /> },
  { to: '/logs', label: 'LOG', icon: <ScrollText className="size-5" /> },
  { to: '/license', label: 'LIC', icon: <KeySquare className="size-5" /> },
  { to: '/system', label: 'SYS', icon: <Settings className="size-5" /> },
]

export function CockpitDock() {
  return (
    <div className="pointer-events-none absolute bottom-0 left-0 right-0 z-30 flex justify-center">
      {/* 弧形发光底座 */}
      <div className="pointer-events-auto relative">
        <div className="absolute -bottom-12 left-1/2 -translate-x-1/2 h-32 w-[1100px] max-w-[95vw] rounded-[50%] bg-cyan-400/10 blur-2xl" />
        <div className="relative glass-strong rounded-t-2xl border-b-0 px-6 pt-3">
          <div className="cockpit-dock">
            {ITEMS.map((it, idx) => (
              <NavLink key={it.to} to={it.to} className="relative">
                {({ isActive }) => (
                  <div
                    className={cn('dock-cell', isActive && 'active')}
                    // 弧形偏移：中间高，两边低
                    style={{
                      transform: `translateY(${arcOffset(idx, ITEMS.length)}px)`,
                    }}
                  >
                    <span className="hex-bg hex" />
                    <span className="icon-wrap">{it.icon}</span>
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
  const t = (i - center) / center // -1..1
  return Math.abs(t) * 12 // 边缘下沉
}
