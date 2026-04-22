import { cn } from '@/lib/utils'

const COLOR: Record<string, string> = {
  ok: 'text-[#00ff88]',
  online: 'text-[#00ff88]',
  active: 'text-[#00ff88]',
  good: 'text-[#00ff88]',
  warning: 'text-[#ffaa00]',
  warn: 'text-[#ffaa00]',
  minor: 'text-[#ffd400]',
  major: 'text-[#ff7a1a]',
  critical: 'text-[#ff2d6f]',
  error: 'text-[#ff2d6f]',
  offline: 'text-[#525a78]',
  inactive: 'text-[#525a78]',
  off: 'text-[#525a78]',
  unknown: 'text-[#6b86b6]',
}

export function StatusBadge({
  status,
  label,
  className,
}: {
  status: string
  label?: string
  className?: string
}) {
  const color = COLOR[status] ?? COLOR.unknown
  return (
    <span className={cn('chip', color, className)}>
      <span className="size-1.5 rounded-full bg-current shadow-[0_0_8px_currentColor]" />
      {label ?? status}
    </span>
  )
}
