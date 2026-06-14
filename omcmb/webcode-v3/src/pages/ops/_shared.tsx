import type { ReactNode } from 'react'
import { Loader2, RefreshCcw, Search } from 'lucide-react'

import { NeonButton } from '@/components/ui/NeonButton'

/**
 * 运维模块内部复用的小组件（三态 / 分页 / 工具栏 / 图标按钮 / 统计卡）。
 *
 * 按硬约束，这些复用件落在 pages/ops/ 目录内，不污染共享 components/。
 * 保持 STARFORGE 暗色霓虹 HUD 审美：玻璃拟态 + 等宽大写标签 + 霓虹描边。
 */

export function Center({ children }: { children: ReactNode }) {
  return <div className="flex min-h-[180px] w-full items-center justify-center">{children}</div>
}

/** 三态渲染：loading / error / empty / content 完整覆盖。 */
export function StateBlock({
  loading,
  error,
  empty,
  emptyText,
  children,
}: {
  loading: boolean
  error: boolean
  empty: boolean
  emptyText: string
  children: ReactNode
}) {
  if (loading) {
    return (
      <Center>
        <span className="flex items-center gap-2 font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/60">
          <Loader2 className="size-4 animate-spin" /> SYNCING…
        </span>
      </Center>
    )
  }
  if (error) {
    return (
      <Center>
        <span className="border border-rose-500/40 bg-rose-500/5 px-4 py-3 font-mono text-sm text-rose-300">
          SYNC FAILED · 数据加载失败
        </span>
      </Center>
    )
  }
  if (empty) {
    return (
      <Center>
        <span className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/40">
          {emptyText}
        </span>
      </Center>
    )
  }
  return <>{children}</>
}

export function Pager({
  page,
  total,
  pageSize,
  onPage,
}: {
  page: number
  total: number
  pageSize: number
  onPage: (p: number) => void
}) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  return (
    <div className="mt-3 flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {pageSize}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <NeonButton onClick={() => onPage(Math.max(1, page - 1))} disabled={page <= 1}>
          ◂ PREV
        </NeonButton>
        <NeonButton onClick={() => onPage(Math.min(totalPages, page + 1))} disabled={page >= totalPages}>
          NEXT ▸
        </NeonButton>
      </div>
    </div>
  )
}

export function Toolbar({
  keyword,
  keywordPlaceholder,
  onKeyword,
  isFetching,
  onRefresh,
  extra,
  children,
}: {
  keyword?: string
  keywordPlaceholder?: string
  onKeyword?: (v: string) => void
  isFetching: boolean
  onRefresh: () => void
  extra?: ReactNode
  children?: ReactNode
}) {
  return (
    <div className="mb-3 flex flex-wrap items-center gap-2">
      {onKeyword ? (
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
          <input
            className="neon-input w-56 pl-9"
            placeholder={keywordPlaceholder}
            value={keyword ?? ''}
            onChange={(e) => onKeyword(e.target.value)}
          />
        </div>
      ) : null}
      {children}
      <div className="ml-auto flex items-center gap-2">
        {isFetching ? (
          <span className="flex items-center gap-1.5 font-mono text-[11px] text-cyan-300/55">
            <Loader2 className="size-3 animate-spin" /> SYNC
          </span>
        ) : null}
        <NeonButton icon={<RefreshCcw />} onClick={onRefresh}>
          REFRESH
        </NeonButton>
        {extra}
      </div>
    </div>
  )
}

export function IconBtn({
  title,
  color,
  disabled,
  onClick,
  children,
}: {
  title: string
  color: string
  disabled?: boolean
  onClick: () => void
  children: ReactNode
}) {
  return (
    <button
      type="button"
      title={title}
      disabled={disabled}
      onClick={onClick}
      className="rounded-sm border border-cyan-500/25 p-1 transition-colors hover:bg-cyan-500/10 disabled:cursor-not-allowed disabled:opacity-40"
      style={{ color }}
    >
      {children}
    </button>
  )
}

export function StatCard({
  label,
  value,
  color,
}: {
  label: string
  value: ReactNode
  color: string
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3"
      style={{ borderLeftColor: color }}
    >
      <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">{label}</div>
      <div
        className="font-display text-3xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}

export function FormRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-1.5">
      <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/65">{label}</div>
      {children}
    </div>
  )
}
