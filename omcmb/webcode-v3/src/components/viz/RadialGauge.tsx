interface Props {
  value: number // 0-100
  label?: string
  size?: number
  color?: string
  trackColor?: string
  unit?: string
}

export function RadialGauge({
  value,
  label,
  size = 120,
  color = 'var(--neon-cyan)',
  trackColor = 'rgba(0, 240, 255, 0.12)',
  unit = '%',
}: Props) {
  const v = Math.max(0, Math.min(100, value))
  const r = size / 2 - 8
  const circumference = 2 * Math.PI * r
  // 仅画 270° 弧
  const visible = circumference * 0.75
  const offset = visible * (1 - v / 100)

  return (
    <div
      className="relative inline-flex items-center justify-center"
      style={{ width: size, height: size }}
    >
      <svg
        width={size}
        height={size}
        viewBox={`0 0 ${size} ${size}`}
        style={{ transform: 'rotate(135deg)' }}
      >
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          stroke={trackColor}
          strokeWidth={4}
          fill="none"
          strokeDasharray={`${visible} ${circumference}`}
          strokeLinecap="round"
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          stroke={color}
          strokeWidth={4}
          fill="none"
          strokeDasharray={`${visible - offset} ${circumference}`}
          strokeLinecap="round"
          style={{ filter: `drop-shadow(0 0 4px ${color})`, transition: 'stroke-dasharray 0.5s ease' }}
        />
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center text-center">
        <div
          className="font-display font-bold"
          style={{ color, textShadow: `0 0 8px ${color}` }}
        >
          <span style={{ fontSize: size * 0.28 }}>{Math.round(v)}</span>
          <span style={{ fontSize: size * 0.14 }} className="ml-0.5 opacity-70">
            {unit}
          </span>
        </div>
        {label && (
          <div className="mt-0.5 text-[10px] uppercase tracking-[0.2em] text-cyan-300/60">
            {label}
          </div>
        )}
      </div>
    </div>
  )
}
