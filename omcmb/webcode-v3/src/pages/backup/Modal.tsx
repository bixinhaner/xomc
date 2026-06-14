import type { ReactNode } from 'react'
import { X } from 'lucide-react'

import { GlassPanel } from '@/components/ui/GlassPanel'

/**
 * 本模块自带的玻璃拟态浮层（v3 皮肤没有共享 Modal 组件，按硬约束放在本模块目录内）。
 * 仅用于备份/恢复的创建表单与删除确认，保持暗色霓虹审美。
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
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <GlassPanel strong className="max-h-[88vh] overflow-hidden" style={{ width }}>
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
        <div className="max-h-[60vh] overflow-auto px-4 py-4">{children}</div>
        {footer ? (
          <div className="flex items-center justify-end gap-2 border-t border-cyan-500/15 px-4 py-3">
            {footer}
          </div>
        ) : null}
      </GlassPanel>
    </div>
  )
}
