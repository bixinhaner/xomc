import { NavLink } from 'react-router-dom'
import { Search } from 'lucide-react'

import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

// ---------------------------------------------------------------------------
// 日志模块内部共享的小组件 / 常量（仅本模块使用，避免每个子页重复）
// ---------------------------------------------------------------------------

export const PAGE_SIZE = 20

export type StatTone = 'default' | 'emerald' | 'amber' | 'rose' | 'muted' | 'primary'

export function Stat({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: number | string
  tone?: StatTone
}) {
  const toneClass = {
    default: 'text-foreground',
    emerald: 'text-emerald-600 dark:text-emerald-400',
    amber: 'text-amber-600 dark:text-amber-400',
    rose: 'text-rose-600 dark:text-rose-400',
    muted: 'text-muted-foreground',
    primary: 'text-primary',
  }[tone]
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>{value}</div>
    </div>
  )
}

// 带放大镜图标的搜索输入框
export function SearchInput({
  value,
  onChange,
  placeholder,
  className,
}: {
  value: string
  onChange: (v: string) => void
  placeholder: string
  className?: string
}) {
  return (
    <div className="relative">
      <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
      <Input
        className={cn('w-64 pl-9', className)}
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  )
}

// 子页统一的导航标签条（系统 / 操作 / 设备 / 异常 / 事件 / 配置）
const LOG_NAV: { to: string; label: string }[] = [
  { to: '/log/system', label: '系统日志' },
  { to: '/log/operation', label: '操作审计' },
  { to: '/log/device', label: '基站日志' },
  { to: '/log/exception', label: '异常重启' },
  { to: '/log/event', label: '设备事件' },
  { to: '/log/config', label: '保留配置' },
]

// 各日志子页顶部统一的标签条（用真实路由跳转，对齐 v1 log 模块各子菜单）
export function LogNavTabs() {
  return (
    <div className="flex items-center gap-1 rounded-lg border bg-card p-1">
      {LOG_NAV.map((t) => (
        <NavLink
          key={t.to}
          to={t.to}
          className={({ isActive }) =>
            cn(
              'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
              isActive
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground'
            )
          }
        >
          {t.label}
        </NavLink>
      ))}
    </div>
  )
}
