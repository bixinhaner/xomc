import type { HTMLAttributes, ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface Props extends Omit<HTMLAttributes<HTMLDivElement>, 'title'> {
  strong?: boolean
  scanline?: boolean
  title?: ReactNode
  meta?: ReactNode
  children: ReactNode
}

export function GlassPanel({
  strong,
  scanline = true,
  title,
  meta,
  className,
  children,
  ...rest
}: Props) {
  return (
    <div
      className={cn(
        strong ? 'glass-strong' : 'glass',
        'relative rounded-sm overflow-hidden',
        className
      )}
      {...rest}
    >
      {scanline ? <div className="scanline" /> : null}
      {(title || meta) && (
        <div className="relative flex items-center justify-between border-b border-cyan-500/15 px-3.5 py-2">
          <div className="flex items-center gap-2 text-[11px] uppercase tracking-[0.2em] text-cyan-300/90">
            <span className="size-1.5 rounded-full bg-cyan-400 shadow-[0_0_8px_currentColor]" />
            {title}
          </div>
          {meta ? <div className="text-[10px] text-cyan-300/50">{meta}</div> : null}
        </div>
      )}
      <div className="relative">{children}</div>
    </div>
  )
}
