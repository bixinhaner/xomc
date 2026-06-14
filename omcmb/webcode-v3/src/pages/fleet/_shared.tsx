import type { ReactNode } from 'react'
import { Loader2, Inbox, AlertTriangle } from 'lucide-react'

/** 统一加载态 */
export function LoadingBlock({ label = 'SYNCING…' }: { label?: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
      <Loader2 className="size-4 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">{label}</span>
    </div>
  )
}

/** 统一错误态 */
export function ErrorBlock({ error }: { error: unknown }) {
  const msg = error instanceof Error ? error.message : '未知错误'
  return (
    <div className="flex items-start gap-2 border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
      <AlertTriangle className="mt-0.5 size-4 shrink-0" />
      <div>
        <div className="font-bold uppercase tracking-[0.2em]">FAILURE</div>
        <div className="mt-1 text-rose-200/80">{msg}</div>
      </div>
    </div>
  )
}

/** 统一空态 */
export function EmptyBlock({ label = 'NO DATA' }: { label?: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 border border-cyan-500/15 px-4 py-16 text-center">
      <Inbox className="size-8 text-cyan-400/40" />
      <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/40">{label}</div>
    </div>
  )
}

/** 三态包装：依次判断 loading / error / empty / 内容 */
export function StateGate({
  isLoading,
  isError,
  error,
  isEmpty,
  loadingLabel,
  emptyLabel,
  children,
}: {
  isLoading: boolean
  isError: boolean
  error?: unknown
  isEmpty: boolean
  loadingLabel?: string
  emptyLabel?: string
  children: ReactNode
}) {
  if (isLoading) return <LoadingBlock label={loadingLabel} />
  if (isError) return <ErrorBlock error={error} />
  if (isEmpty) return <EmptyBlock label={emptyLabel} />
  return <>{children}</>
}

/** 单元 KPI 小卡 */
export function StatCard({
  label,
  value,
  color = '#00f0ff',
  hint,
}: {
  label: string
  value: ReactNode
  color?: string
  hint?: ReactNode
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: color }}
    >
      <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/65">
        {label}
      </div>
      <div
        className="font-display text-2xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
      {hint ? <div className="mt-0.5 font-mono text-[10px] text-cyan-300/45">{hint}</div> : null}
    </div>
  )
}

/** 键值描述行 */
export function KV({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5 border-b border-cyan-500/8 px-3 py-2">
      <span className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/50">
        {label}
      </span>
      <span className="font-mono text-xs text-cyan-100/90 break-all">{children || '—'}</span>
    </div>
  )
}

/** 简单分页条 */
export function Pager({
  page,
  totalPages,
  total,
  pageSize,
  onPrev,
  onNext,
}: {
  page: number
  totalPages: number
  total: number
  pageSize: number
  onPrev: () => void
  onNext: () => void
}) {
  return (
    <div className="mt-4 flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {pageSize}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <button
          type="button"
          className="neon-btn"
          onClick={onPrev}
          disabled={page <= 1}
        >
          ◂ PREV
        </button>
        <button
          type="button"
          className="neon-btn"
          onClick={onNext}
          disabled={page >= totalPages}
        >
          NEXT ▸
        </button>
      </div>
    </div>
  )
}

/** 秒 → 人类可读时长 */
export function formatDuration(seconds?: number | null): string {
  if (seconds == null || seconds <= 0) return '—'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const parts: string[] = []
  if (d) parts.push(`${d}d`)
  if (h) parts.push(`${h}h`)
  if (m || parts.length === 0) parts.push(`${m}m`)
  return parts.join(' ')
}
