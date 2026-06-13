import type { ReactNode } from 'react'
import { Loader2 } from 'lucide-react'

import { cn } from '@/lib/utils'
import { GlassPanel } from '@/components/ui/GlassPanel'

export function PageShell({
  code,
  title,
  subtitle,
  toolbar,
  isFetching,
  children,
  className,
  bare,
}: {
  code: string
  title: string
  subtitle?: string
  toolbar?: ReactNode
  isFetching?: boolean
  children: ReactNode
  className?: string
  bare?: boolean
}) {
  return (
    <div className={cn('warp-in flex h-full w-full flex-col gap-4', className)}>
      {/* 模块标题带 */}
      <div className="flex items-end justify-between gap-4">
        <div className="flex items-end gap-4">
          <div className="font-display text-4xl font-bold leading-none text-cyan-300/85 text-glow">
            {code}
          </div>
          <div>
            <div className="font-display text-lg leading-tight text-cyan-100">{title}</div>
            {subtitle && (
              <div className="font-mono text-[11px] uppercase tracking-[0.18em] text-cyan-300/55">
                {subtitle}
              </div>
            )}
          </div>
          {isFetching ? (
            <span className="flex items-center gap-1.5 self-end pb-1 text-[11px] text-cyan-300/60">
              <Loader2 className="size-3 animate-spin" />
              SYNC
            </span>
          ) : null}
        </div>
        {toolbar ? <div className="flex flex-wrap items-center gap-2">{toolbar}</div> : null}
      </div>

      {/* 内容 */}
      {bare ? (
        <div className="flex-1 min-h-0 overflow-auto">{children}</div>
      ) : (
        <GlassPanel strong className="flex-1 min-h-0 overflow-hidden">
          <div className="h-full overflow-auto p-4">{children}</div>
        </GlassPanel>
      )}
    </div>
  )
}
