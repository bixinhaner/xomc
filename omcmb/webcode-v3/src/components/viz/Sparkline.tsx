import { useMemo } from 'react'

interface Props {
  data: number[]
  width?: number
  height?: number
  color?: string
  fill?: boolean
  strokeWidth?: number
  className?: string
}

/**
 * 纯 SVG 火花线，用于 KPI 卡片 / 趋势提示。
 * 数据 ≤ 64 点性能最佳。
 */
export function Sparkline({
  data,
  width = 120,
  height = 36,
  color = 'var(--neon-cyan)',
  fill = true,
  strokeWidth = 1.4,
  className,
}: Props) {
  const { line, area, lastDot } = useMemo(() => {
    if (data.length === 0) {
      return { line: '', area: '', lastDot: null }
    }
    const min = Math.min(...data)
    const max = Math.max(...data)
    const span = max - min || 1
    const stepX = data.length === 1 ? 0 : width / (data.length - 1)
    const points = data.map((v, i) => {
      const x = i * stepX
      const y = height - ((v - min) / span) * (height - 4) - 2
      return [x, y] as const
    })
    const linePath = points
      .map(([x, y], i) => (i === 0 ? `M ${x} ${y}` : `L ${x} ${y}`))
      .join(' ')
    const areaPath = `${linePath} L ${width} ${height} L 0 ${height} Z`
    const last = points[points.length - 1]
    return {
      line: linePath,
      area: areaPath,
      lastDot: { x: last[0], y: last[1] },
    }
  }, [data, width, height])

  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      width={width}
      height={height}
      className={className}
      style={{ overflow: 'visible' }}
    >
      <defs>
        <linearGradient id={`sl-grad-${color.replace(/\W/g, '')}`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={color} stopOpacity={0.45} />
          <stop offset="100%" stopColor={color} stopOpacity={0} />
        </linearGradient>
      </defs>
      {fill && (
        <path
          d={area}
          fill={`url(#sl-grad-${color.replace(/\W/g, '')})`}
        />
      )}
      <path
        d={line}
        fill="none"
        stroke={color}
        strokeWidth={strokeWidth}
        strokeLinejoin="round"
        strokeLinecap="round"
        style={{ filter: `drop-shadow(0 0 4px ${color})` }}
      />
      {lastDot && (
        <circle
          cx={lastDot.x}
          cy={lastDot.y}
          r={2.4}
          fill={color}
          style={{ filter: `drop-shadow(0 0 4px ${color})` }}
        />
      )}
    </svg>
  )
}
