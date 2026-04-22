import { cn } from '@/lib/utils'

interface Props {
  label: string
  value: number // 0-100
  color?: string
  className?: string
}

export function PulseHex({ label, value, color = 'var(--neon-cyan)', className }: Props) {
  const v = Math.max(0, Math.min(100, value))
  return (
    <div className={cn('pulse-hex hex', className)} style={{ ['--pulse-color' as never]: color }}>
      <span className="hex-bg hex" />
      <span
        className="hex-fill hex"
        style={{
          ['--pulse-color' as never]: color,
          transform: `scaleY(${0.2 + (v / 100) * 0.8})`,
          transformOrigin: 'bottom',
        }}
      />
      <span className="pulse-label" style={{ color }}>
        {label}
      </span>
    </div>
  )
}
