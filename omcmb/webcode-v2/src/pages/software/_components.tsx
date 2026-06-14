// ---------------------------------------------------------------------------
// 软件版本模块共享小组件（模块内私有，非路由页面）
// ---------------------------------------------------------------------------
import type { ReactNode } from 'react'
import type { Package } from 'lucide-react'

export function Stat({
  label,
  value,
  icon: Icon,
  tone = 'default',
}: {
  label: string
  value: number | string
  icon: typeof Package
  tone?: 'default' | 'emerald' | 'amber' | 'muted'
}) {
  const toneClass = {
    default: 'text-foreground',
    emerald: 'text-emerald-600 dark:text-emerald-400',
    amber: 'text-amber-600 dark:text-amber-400',
    muted: 'text-muted-foreground',
  }[tone]
  return (
    <div className="flex items-center gap-3 rounded-lg border bg-card px-4 py-3">
      <div className="rounded-md bg-muted p-2 text-muted-foreground">
        <Icon className="size-4" />
      </div>
      <div className="min-w-0">
        <div className="text-xs uppercase tracking-wider text-muted-foreground">{label}</div>
        <div className={`mt-0.5 text-xl font-semibold tabular-nums ${toneClass}`}>{value}</div>
      </div>
    </div>
  )
}

export function ProgressBar({ pct }: { pct: number }) {
  const w = Math.min(100, Math.max(0, pct))
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-24 overflow-hidden rounded-full bg-muted">
        <div className="h-full rounded-full bg-primary transition-all" style={{ width: `${w}%` }} />
      </div>
      <span className="tabular-nums text-xs text-muted-foreground">{pct}%</span>
    </div>
  )
}

export function IconBtn({
  title,
  onClick,
  disabled,
  tone = 'default',
  children,
}: {
  title: string
  onClick: () => void
  disabled?: boolean
  tone?: 'default' | 'destructive'
  children: ReactNode
}) {
  return (
    <button
      type="button"
      title={title}
      disabled={disabled}
      onClick={onClick}
      className={`inline-flex items-center justify-center rounded-md border p-1.5 transition-colors disabled:cursor-not-allowed disabled:opacity-40 ${
        tone === 'destructive'
          ? 'text-destructive hover:bg-destructive/10'
          : 'text-muted-foreground hover:bg-muted hover:text-foreground'
      }`}
    >
      {children}
    </button>
  )
}

export function InfoRow({ label, value, mono }: { label: string; value: ReactNode; mono?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-4 border-b py-2.5 text-sm">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span className={`truncate text-right ${mono ? 'font-mono text-xs' : ''}`}>
        {value === '' || value == null ? '—' : value}
      </span>
    </div>
  )
}
