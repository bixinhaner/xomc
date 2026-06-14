import type { ComponentType } from 'react'
import { Loader2, AlertTriangle, Inbox } from 'lucide-react'

// ===========================================================================
// CONFIG 模块本地共享 HUD 片段（仅本模块复用，不触碰 components/ 共享层）。
// ===========================================================================

export function StatCard({
  label,
  value,
  color,
}: {
  label: string
  value: number | string
  color: string
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3"
      style={{ borderLeftColor: color }}
    >
      <div className="scanline" />
      <div className="relative font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
        {label}
      </div>
      <div
        className="relative font-display text-3xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {typeof value === 'number' ? value.toLocaleString() : value}
      </div>
    </div>
  )
}

export function HudLoading({ text = 'SYNCING…' }: { text?: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
      <Loader2 className="size-4 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">{text}</span>
    </div>
  )
}

export function HudError({ error }: { error: unknown }) {
  return (
    <div className="m-3 flex items-center gap-2 border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
      <AlertTriangle className="size-4 shrink-0" />
      FAILURE · {error instanceof Error ? error.message : '未知错误'}
    </div>
  )
}

export function HudEmpty({
  icon: Icon = Inbox,
  text,
}: {
  icon?: ComponentType<{ className?: string }>
  text: string
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16">
      <Icon className="size-9 text-cyan-400/45" />
      <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">{text}</div>
    </div>
  )
}

/** 三态守卫：返回 null 表示「数据就绪、继续渲染」。 */
export function hudGuard({
  isLoading,
  isError,
  error,
  isEmpty,
  emptyIcon,
  emptyText,
  loadingText,
}: {
  isLoading: boolean
  isError: boolean
  error: unknown
  isEmpty: boolean
  emptyIcon?: ComponentType<{ className?: string }>
  emptyText: string
  loadingText?: string
}) {
  if (isLoading) return <HudLoading text={loadingText} />
  if (isError) return <HudError error={error} />
  if (isEmpty) return <HudEmpty icon={emptyIcon} text={emptyText} />
  return null
}
