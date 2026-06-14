import type { ReactNode } from 'react'
import { X } from 'lucide-react'

import { GlassPanel } from '@/components/ui/GlassPanel'

/**
 * 本模块自带的玻璃拟态浮层（v3 皮肤无共享 Modal，按硬约束放在本模块目录内）。
 * 用于运维任务/模板的创建表单与确认弹窗，保持暗色霓虹审美。
 */
export function Modal({
  open,
  title,
  subtitle,
  onClose,
  footer,
  width = 560,
  children,
}: {
  open: boolean
  title: string
  subtitle?: string
  onClose: () => void
  footer?: ReactNode
  width?: number
  children: ReactNode
}) {
  if (!open) return null
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-[#02040a]/70 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <GlassPanel strong className="warp-in max-h-[88vh] overflow-hidden" style={{ width }}>
        <div className="relative flex items-center justify-between border-b border-cyan-500/15 px-4 py-3">
          <div>
            <div className="font-display text-base text-cyan-100">{title}</div>
            {subtitle ? (
              <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                {subtitle}
              </div>
            ) : null}
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-sm p-1 text-cyan-300/60 transition-colors hover:bg-cyan-500/10 hover:text-cyan-200"
          >
            <X className="size-4" />
          </button>
        </div>
        <div className="max-h-[62vh] overflow-auto px-4 py-4">{children}</div>
        {footer ? (
          <div className="flex items-center justify-end gap-2 border-t border-cyan-500/15 px-4 py-3">
            {footer}
          </div>
        ) : null}
      </GlassPanel>
    </div>
  )
}

/** 右侧抽屉浮层（详情用）。 */
export function Drawer({
  open,
  title,
  badge,
  onClose,
  children,
  width = 520,
}: {
  open: boolean
  title: string
  badge?: ReactNode
  onClose: () => void
  children: ReactNode
  width?: number
}) {
  if (!open) return null
  return (
    <div className="fixed inset-0 z-50 flex justify-end" role="dialog" aria-modal="true">
      <button
        type="button"
        aria-label="关闭"
        className="absolute inset-0 bg-black/55 backdrop-blur-sm"
        onClick={onClose}
      />
      <div
        className="glass-strong warp-in relative flex h-full max-w-[94vw] flex-col border-l border-cyan-500/25"
        style={{ width }}
      >
        <div className="flex items-center justify-between gap-2 border-b border-cyan-500/20 px-4 py-3">
          <div className="flex min-w-0 items-center gap-2">
            {badge}
            <span className="truncate font-display text-sm font-bold text-cyan-100">{title}</span>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="shrink-0 rounded-sm border border-cyan-500/25 p-1 text-cyan-300/70 hover:border-cyan-400/60 hover:text-cyan-200"
          >
            <X className="size-4" />
          </button>
        </div>
        <div className="flex-1 overflow-auto">{children}</div>
      </div>
    </div>
  )
}

/** 详情区分组标题。 */
export function SectionTitle({ children }: { children: ReactNode }) {
  return (
    <div className="border-b border-cyan-500/15 bg-cyan-500/[0.04] px-3.5 py-1.5 font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/80">
      {children}
    </div>
  )
}

/** 详情字段行。 */
export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="grid grid-cols-[120px_1fr] gap-3 border-b border-cyan-500/10 px-3.5 py-2 last:border-b-0">
      <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">
        {label}
      </div>
      <div className="break-words text-[12px] text-cyan-100/90">{children ?? '—'}</div>
    </div>
  )
}
