import type { ReactNode } from 'react'
import { Loader2, CircleDot, Search, RefreshCcw } from 'lucide-react'

import { NeonButton } from '@/components/ui/NeonButton'

/* ============================ 通用三态包裹 ============================ */

export function StateBlock({
  isLoading,
  isError,
  error,
  isEmpty,
  emptyLabel,
  children,
}: {
  isLoading: boolean
  isError: boolean
  error?: unknown
  isEmpty: boolean
  emptyLabel: string
  children: ReactNode
}) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
        <Loader2 className="size-4 animate-spin" />
        <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
      </div>
    )
  }
  if (isError) {
    return (
      <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
        FAILURE · {error instanceof Error ? error.message : '未知错误'}
      </div>
    )
  }
  if (isEmpty) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-16 text-cyan-300/45">
        <CircleDot className="size-9 opacity-50" />
        <div className="font-mono text-xs uppercase tracking-[0.2em]">{emptyLabel}</div>
      </div>
    )
  }
  return <>{children}</>
}

/* ============================ 统计卡 ============================ */

export function MiniStat({
  label,
  value,
  color,
  icon,
}: {
  label: string
  value: ReactNode
  color: string
  icon?: ReactNode
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
        className="font-display text-3xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}

/* ============================ 行头 ============================ */

export function RowHeader({ cols, children }: { cols: string; children: ReactNode }) {
  return (
    <div
      className="grid items-center gap-3 px-3 pb-1 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/45"
      style={{ gridTemplateColumns: cols.split('_').join(' ') }}
    >
      {children}
    </div>
  )
}

/* ============================ 关键字工具条 ============================ */

export function KeywordToolbar({
  placeholder,
  value,
  onChange,
  onRefresh,
  width = 'w-72',
}: {
  placeholder: string
  value: string
  onChange: (v: string) => void
  onRefresh?: () => void
  width?: string
}) {
  return (
    <>
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
        <input
          className={`neon-input ${width} pl-9`}
          placeholder={placeholder}
          value={value}
          onChange={(e) => onChange(e.target.value)}
        />
      </div>
      {onRefresh ? (
        <NeonButton icon={<RefreshCcw />} onClick={onRefresh}>
          REFRESH
        </NeonButton>
      ) : null}
    </>
  )
}

/* ============================ 分页器 ============================ */

export function Pager({
  page,
  totalPages,
  total,
  pageSize,
  onPage,
}: {
  page: number
  totalPages: number
  total: number
  pageSize: number
  onPage: (updater: (p: number) => number) => void
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {pageSize}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <NeonButton onClick={() => onPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
          ◂ PREV
        </NeonButton>
        <NeonButton
          onClick={() => onPage((p) => Math.min(totalPages, p + 1))}
          disabled={page >= totalPages}
        >
          NEXT ▸
        </NeonButton>
      </div>
    </div>
  )
}

/* ============================ KV 字段渲染 ============================ */

export function FieldRow({
  label,
  children,
}: {
  label: string
  children: ReactNode
}) {
  return (
    <div className="grid grid-cols-[160px_1fr] items-start gap-3 border-b border-cyan-500/8 py-2.5">
      <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">
        {label}
      </div>
      <div className="min-w-0 text-sm text-cyan-100/90">{children}</div>
    </div>
  )
}
