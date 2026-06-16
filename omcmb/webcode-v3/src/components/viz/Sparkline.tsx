import { useMemo } from 'react'

interface Props {
  // issue #429：data 支持 (number|null)[]，null 槽位=缺采/断档，路径在此断开(不连线)。
  // 纯 number[] 是其子集，老调用方无需改动即兼容。
  data: (number | null)[]
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
 * issue #429：data 含 null 时在缺采槽位断开路径（遇 null 不连线段，下一段非 null 起点重新 M）。
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
    const nums = data.filter((v): v is number => v !== null && v !== undefined)
    if (nums.length === 0) {
      return { line: '', area: '', lastDot: null }
    }
    const min = Math.min(...nums)
    const max = Math.max(...nums)
    const span = max - min || 1
    const stepX = data.length === 1 ? 0 : width / (data.length - 1)
    // 每个槽位算坐标；null 槽位记 null（路径在此断开）。
    const points = data.map((v, i) => {
      if (v === null || v === undefined) return null
      const x = i * stepX
      const y = height - ((v - min) / span) * (height - 4) - 2
      return [x, y] as const
    })
    // 遇 null 断开：null 后第一个非 null 用 M 重新起笔，连续段内用 L。
    let prevNull = true
    const segs: string[] = []
    for (const pt of points) {
      if (pt === null) {
        prevNull = true
        continue
      }
      const [x, y] = pt
      segs.push(prevNull ? `M ${x} ${y}` : `L ${x} ${y}`)
      prevNull = false
    }
    const linePath = segs.join(' ')
    // 填充面积按整体折线包络铺底（断档段不画填充会更碎，这里维持原视觉以末点收口）。
    const lastNonNull = [...points].reverse().find((p): p is readonly [number, number] => p !== null)
    const lastX = lastNonNull ? lastNonNull[0] : width
    const areaPath = lastNonNull ? `${linePath} L ${lastX} ${height} L 0 ${height} Z` : ''
    return {
      line: linePath,
      area: areaPath,
      lastDot: lastNonNull ? { x: lastNonNull[0], y: lastNonNull[1] } : null,
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
