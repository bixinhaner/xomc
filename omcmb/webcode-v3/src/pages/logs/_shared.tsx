import type { ReactNode } from 'react'
import { Loader2, X } from 'lucide-react'

import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'

// ──────────────────────────────────────────────────────────────────────────
// pages/logs 私有 HUD 原子（仅本模块复用，避免改共享 components/*）。
// 与 pages/logs/index.tsx 内同名小组件保持一致的 STARFORGE 审美。
// ──────────────────────────────────────────────────────────────────────────

export const LOG_PAGE_SIZE = 50

export function StatCard({
  label,
  color,
  value,
  hint,
}: {
  label: string
  color: string
  value: number | string
  hint?: string
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3"
      style={{ borderLeftColor: color }}
    >
      <div className="flex items-center justify-between">
        <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
          {label}
        </span>
        {hint && <span className="font-mono text-[9px] text-cyan-300/35">{hint}</span>}
      </div>
      <div
        className="font-display text-2xl font-bold leading-tight tabular-nums"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}

export function Pager({
  page,
  totalPages,
  total,
  pageSize = LOG_PAGE_SIZE,
  onChange,
}: {
  page: number
  totalPages: number
  total: number
  pageSize?: number
  onChange: (updater: (p: number) => number) => void
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {pageSize}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <NeonButton onClick={() => onChange((p) => Math.max(1, p - 1))} disabled={page <= 1}>
          ◂ PREV
        </NeonButton>
        <NeonButton
          onClick={() => onChange((p) => Math.min(totalPages, p + 1))}
          disabled={page >= totalPages}
        >
          NEXT ▸
        </NeonButton>
      </div>
    </div>
  )
}

export function FeedLoading({ text = 'SYNCING…' }: { text?: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
      <Loader2 className="size-4 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">{text}</span>
    </div>
  )
}

export function FeedError({ error }: { error: unknown }) {
  return (
    <div className="m-3 border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
      SYNC FAILED · {error instanceof Error ? error.message : '未知错误'}
    </div>
  )
}

export function EmptyFeed({ icon, text }: { icon: ReactNode; text: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16">
      {icon}
      <div className="font-mono text-xs uppercase tracking-[0.2em] text-emerald-300/70">{text}</div>
    </div>
  )
}

export function DetailOverlay({
  title,
  meta,
  accent,
  onClose,
  children,
}: {
  title: string
  meta: string
  accent: string
  onClose: () => void
  children: ReactNode
}) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-6 backdrop-blur-sm"
      onClick={onClose}
    >
      <div className="w-full max-w-2xl" onClick={(e) => e.stopPropagation()}>
        <GlassPanel strong className="overflow-hidden">
          <div
            className="flex items-center justify-between border-b px-4 py-3"
            style={{ borderColor: `${accent}40` }}
          >
            <div className="flex items-center gap-2">
              <span
                className="size-1.5 rounded-full"
                style={{ background: accent, boxShadow: `0 0 8px ${accent}` }}
              />
              <span className="font-display text-sm font-bold text-cyan-100">{title}</span>
              <span className="font-mono text-[10px] text-cyan-300/50">{meta}</span>
            </div>
            <button
              type="button"
              onClick={onClose}
              className="text-cyan-300/60 transition-colors hover:text-cyan-100"
            >
              <X className="size-4" />
            </button>
          </div>
          <div className="p-4">{children}</div>
        </GlassPanel>
      </div>
    </div>
  )
}

export function DRow({
  k,
  v,
  mono,
  color,
}: {
  k: string
  v?: string | number | null
  mono?: boolean
  color?: string
}) {
  const text = v === null || v === undefined || v === '' ? '—' : String(v)
  return (
    <>
      <dt className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/50">{k}</dt>
      <dd
        className={`break-all ${mono ? 'font-mono text-[11px]' : 'text-[12px]'}`}
        style={{ color: color ?? '#cfe6ff' }}
      >
        {text}
      </dd>
    </>
  )
}

export function FilterInput({
  value,
  onChange,
  placeholder,
  icon,
  width = 'w-60',
}: {
  value: string
  onChange: (v: string) => void
  placeholder: string
  icon: ReactNode
  width?: string
}) {
  return (
    <div className="relative">
      <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-cyan-300/50 [&_svg]:size-3.5">
        {icon}
      </span>
      <input
        className={`neon-input ${width} pl-9`}
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  )
}
