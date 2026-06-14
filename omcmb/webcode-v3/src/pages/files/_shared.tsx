import type { ReactNode } from 'react'
import { Loader2, Inbox, AlertTriangle } from 'lucide-react'

import { cn } from '@/lib/utils'

// ---------------------------------------------------------------------------
// 三态 (loading / error / empty) —— FILES 模块自包含，不依赖其它页面目录
// ---------------------------------------------------------------------------

export function LoadingBlock({ label = 'SYNCING…' }: { label?: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
      <Loader2 className="size-4 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">{label}</span>
    </div>
  )
}

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

export function EmptyBlock({ label = 'NO DATA · 暂无数据' }: { label?: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 border border-cyan-500/15 px-4 py-16 text-center">
      <Inbox className="size-8 text-cyan-400/40" />
      <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/40">
        {label}
      </div>
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

// ---------------------------------------------------------------------------
// 概览统计卡 / KV 行 / 分页条
// ---------------------------------------------------------------------------

export function OverviewStat({
  icon,
  label,
  color,
  value,
  hint,
}: {
  icon?: ReactNode
  label: string
  color: string
  value: ReactNode
  hint?: ReactNode
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3"
      style={{ borderLeftColor: color }}
    >
      <div className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
        {icon ? <span style={{ color }}>{icon}</span> : null}
        {label}
      </div>
      <div
        className="font-display text-2xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
      {hint ? (
        <div className="font-mono text-[10px] text-cyan-300/45">{hint}</div>
      ) : null}
    </div>
  )
}

export function KV({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5 border-b border-cyan-500/8 px-3 py-2">
      <span className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/50">
        {label}
      </span>
      <span className="font-mono text-xs text-cyan-100/90 break-all">
        {children || '—'}
      </span>
    </div>
  )
}

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
        <button type="button" className="neon-btn" onClick={onPrev} disabled={page <= 1}>
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

// ---------------------------------------------------------------------------
// 列表行（霓虹左边框 hover 发光，复用 index.tsx 的 fleet-row 视觉语言）
// ---------------------------------------------------------------------------

export function Row({
  color,
  className,
  children,
}: {
  color: string
  className?: string
  children: ReactNode
}) {
  return (
    <div
      className={cn('fleet-row grid items-center gap-3 rounded-sm px-3 py-2.5', className)}
      style={{ ['--row-color' as never]: color }}
    >
      {children}
    </div>
  )
}

/** 顶部过滤 chip 切换组 */
export function ChipFilter<T extends string>({
  options,
  value,
  onChange,
  color = '#00f0ff',
}: {
  options: Array<{ value: T; label: string; color?: string }>
  value: T
  onChange: (v: T) => void
  color?: string
}) {
  return (
    <>
      {options.map((o) => (
        <button
          key={o.value || 'all'}
          type="button"
          onClick={() => onChange(o.value)}
          className={cn(
            'chip transition-all',
            value === o.value
              ? 'shadow-[0_0_10px_currentColor]'
              : 'opacity-55 hover:opacity-100'
          )}
          style={{ color: o.color ?? color }}
        >
          {o.label}
        </button>
      ))}
    </>
  )
}

// ---------------------------------------------------------------------------
// 格式化
// ---------------------------------------------------------------------------

export const NEON = {
  cyan: '#00f0ff',
  green: '#00ff88',
  violet: '#b388ff',
  blue: '#5b9eff',
  amber: '#ff7a1a',
  gold: '#ffd400',
  rose: '#ff2d6f',
} as const
