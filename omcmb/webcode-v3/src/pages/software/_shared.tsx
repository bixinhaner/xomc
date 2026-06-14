import type { ReactNode } from 'react'
import { Loader2, X } from 'lucide-react'

import { NeonButton } from '@/components/ui/NeonButton'

// ---------------------------------------------------------------------------
// 软件武库 — 模块内共享视觉常量与小组件
// （仅在 pages/software/ 内复用；非全局共享组件）
// ---------------------------------------------------------------------------

export const NEON = {
  cyan: '#00f0ff',
  green: '#00ff88',
  violet: '#a855f7',
  amber: '#ffaa00',
  gold: '#ffd400',
  rose: '#ff2d6f',
  blue: '#5b9eff',
  orange: '#ff7a1a',
  dim: '#525a78',
} as const

export function formatFileSize(bytes?: number | null): string {
  const b = bytes ?? 0
  if (b >= 1024 * 1024 * 1024) return `${(b / 1024 / 1024 / 1024).toFixed(2)} GB`
  if (b >= 1024 * 1024) return `${(b / 1024 / 1024).toFixed(2)} MB`
  if (b >= 1024) return `${(b / 1024).toFixed(2)} KB`
  return `${b} B`
}

export function StatCard({
  label,
  value,
  color,
  icon,
}: {
  label: string
  value: string
  color: string
  icon: ReactNode
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3"
      style={{ borderLeftColor: color }}
    >
      <div className="scanline" />
      <div className="relative flex items-center justify-between">
        <div className="min-w-0">
          <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">{label}</div>
          <div className="font-display text-2xl font-bold leading-tight text-glow" style={{ color }}>
            {value}
          </div>
        </div>
        <span style={{ color }}>{icon}</span>
      </div>
    </div>
  )
}

export function MiniStat({ label, value, color }: { label: string; value: number | string; color: string }) {
  return (
    <div className="glass rounded-sm px-3 py-2 text-center">
      <div className="font-mono text-[9px] uppercase tracking-[0.2em] text-cyan-300/55">{label}</div>
      <div className="font-display text-lg font-bold" style={{ color, textShadow: `0 0 6px ${color}` }}>
        {value}
      </div>
    </div>
  )
}

export function KV({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5">
      <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/50">{label}</span>
      <span className="text-[13px] text-cyan-100/85">{children}</span>
    </div>
  )
}

export function TagGroup({ title, color, items }: { title: string; color: string; items: string[] }) {
  return (
    <div>
      <div className="mb-1 font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/55">{title}</div>
      <div className="flex flex-wrap gap-1.5">
        {items.map((it, i) => (
          <span key={i} className="chip" style={{ color }}>
            {it}
          </span>
        ))}
      </div>
    </div>
  )
}

export function Drawer({
  title,
  meta,
  onClose,
  children,
  wide,
}: {
  title: string
  meta?: string
  onClose: () => void
  children: ReactNode
  wide?: boolean
}) {
  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div className="absolute inset-0 bg-black/55 backdrop-blur-sm" onClick={onClose} />
      <div
        className={`warp-in glass-strong relative h-full overflow-auto border-l border-cyan-500/25 ${
          wide ? 'w-[640px] max-w-[92vw]' : 'w-[460px] max-w-[92vw]'
        }`}
      >
        <div className="sticky top-0 z-10 flex items-center justify-between border-b border-cyan-500/20 bg-[#03050d]/80 px-4 py-3 backdrop-blur-md">
          <div className="flex items-center gap-2">
            <span className="size-1.5 rounded-full bg-cyan-400 shadow-[0_0_8px_currentColor]" />
            <span className="truncate font-mono text-[12px] uppercase tracking-[0.18em] text-cyan-200">{title}</span>
          </div>
          <div className="flex items-center gap-3">
            {meta ? <span className="font-mono text-[10px] text-cyan-300/50">{meta}</span> : null}
            <button type="button" onClick={onClose} className="text-cyan-300/60 hover:text-cyan-200">
              <X className="size-4" />
            </button>
          </div>
        </div>
        {children}
      </div>
    </div>
  )
}

export function Syncing({ label }: { label: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
      <Loader2 className="size-4 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">{label}</span>
    </div>
  )
}

export function ErrorBlock({ msg }: { msg: string }) {
  return (
    <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
      <div className="font-bold uppercase tracking-[0.2em]">FAILURE</div>
      <div className="mt-1 text-rose-200/80">{msg}</div>
    </div>
  )
}

export function EmptyBlock({ label }: { label: string }) {
  return (
    <div className="border border-cyan-500/15 px-4 py-12 text-center font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/40">
      {label}
    </div>
  )
}

export function Pager({
  page,
  totalPages,
  pageSize,
  total,
  onPage,
}: {
  page: number
  totalPages: number
  pageSize: number
  total: number
  onPage: (fn: (p: number) => number) => void
}) {
  return (
    <div className="mt-4 flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {pageSize}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <NeonButton onClick={() => onPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
          ◂ PREV
        </NeonButton>
        <NeonButton onClick={() => onPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages}>
          NEXT ▸
        </NeonButton>
      </div>
    </div>
  )
}

// 升级任务状态/类型/结果配色 ----------------------------------------------------

export const VERSION_STATUS: Record<string, { label: string; color: string }> = {
  current: { label: '当前', color: NEON.green },
  beta: { label: '测试', color: NEON.blue },
  deprecated: { label: '弃用', color: NEON.amber },
  archived: { label: '归档', color: NEON.dim },
}

export const TASK_STATUS: Record<string, { label: string; color: string }> = {
  pending: { label: '等待', color: NEON.dim },
  in_progress: { label: '执行中', color: NEON.cyan },
  suspended: { label: '已暂停', color: NEON.amber },
  ended: { label: '已结束', color: NEON.green },
}

export const TASK_TYPE: Record<number, { label: string; color: string }> = {
  1: { label: '软件升级', color: NEON.green },
  2: { label: '版本回退', color: NEON.amber },
  4: { label: '补丁升级', color: NEON.blue },
  6: { label: 'FPGA 升级', color: NEON.violet },
  8: { label: '激活', color: NEON.cyan },
}

export const RESULT_COLOR: Record<string, string> = {
  success: NEON.green,
  partial: NEON.amber,
  failed: NEON.rose,
  terminated: NEON.dim,
}

export const SUB_STATUS: Record<string, { label: string; color: string }> = {
  pending: { label: '等待', color: NEON.dim },
  downloading: { label: '下载中', color: NEON.cyan },
  rebooting: { label: '重启中', color: NEON.violet },
  verifying: { label: '校验中', color: NEON.blue },
  completed: { label: '成功', color: NEON.green },
  failed: { label: '失败', color: NEON.rose },
  suspended: { label: '已暂停', color: NEON.amber },
  terminated: { label: '已终止', color: NEON.dim },
}
