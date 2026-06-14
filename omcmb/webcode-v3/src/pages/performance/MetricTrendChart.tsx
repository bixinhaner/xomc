import { useMemo } from 'react'

import { GlassPanel } from '@/components/ui/GlassPanel'
import { Sparkline } from '@/components/viz/Sparkline'
import type { AggregatedRow } from '@core/types/pmDashboard'

interface Props {
  metricPath: string
  displayName: string
  rows: AggregatedRow[]
  color: string
}

/**
 * 单指标趋势卡：把 PM 聚合直查行（long 格式）按时间桶转置成有序序列，
 * 渲染霓虹火花线 + 当前/均值/峰值/谷值四数。null（fill_empty 占位）当缺采，
 * 不参与统计、不画点。纯展示组件，数据由父级 useAggregatedMetricsByDevices 提供。
 */
export function MetricTrendChart({ metricPath, displayName, rows, color }: Props) {
  const { series, latest, avg, min, max, points } = useMemo(() => {
    // 按时间桶升序去重，缺采行（metricValue==null）跳过；同桶多设备/小区取最后一条值聚一条总线。
    const byBucket = new Map<string, number>()
    rows.forEach((r) => {
      if (r.metricValue === null || r.metricValue === undefined) return
      byBucket.set(r.startTime, r.metricValue)
    })
    const buckets = Array.from(byBucket.keys()).sort()
    const vals = buckets.map((b) => byBucket.get(b) as number)
    if (vals.length === 0) {
      return { series: [] as number[], latest: null, avg: null, min: null, max: null, points: 0 }
    }
    const sum = vals.reduce((a, b) => a + b, 0)
    return {
      series: vals,
      latest: vals[vals.length - 1],
      avg: sum / vals.length,
      min: Math.min(...vals),
      max: Math.max(...vals),
      points: vals.length,
    }
  }, [rows])

  const fmt = (v: number | null) => {
    if (v === null) return '—'
    if (Math.abs(v) >= 1000) return v.toLocaleString(undefined, { maximumFractionDigits: 0 })
    return Number.isInteger(v) ? String(v) : v.toFixed(2)
  }

  return (
    <GlassPanel title={displayName} meta={`${points} PTS`}>
      <div className="p-4">
        {series.length === 0 ? (
          <div className="flex items-center justify-center py-6 font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/40">
            NO SAMPLE IN WINDOW
          </div>
        ) : (
          <>
            <div className="mb-3 flex items-end justify-between">
              <div>
                <div
                  className="font-display text-3xl font-bold leading-none"
                  style={{ color, textShadow: `0 0 8px ${color}` }}
                >
                  {fmt(latest)}
                </div>
                <div className="mt-1 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/50">
                  LATEST
                </div>
              </div>
              <Sparkline data={series} color={color} width={180} height={48} />
            </div>
            <div className="grid grid-cols-3 gap-2 border-t border-cyan-500/12 pt-2.5">
              <Stat label="AVG" value={fmt(avg)} />
              <Stat label="MIN" value={fmt(min)} />
              <Stat label="MAX" value={fmt(max)} />
            </div>
            <div className="mt-2 truncate font-mono text-[9px] text-cyan-300/35">
              {metricPath}
            </div>
          </>
        )}
      </div>
    </GlassPanel>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="text-center">
      <div className="font-display text-sm font-bold text-cyan-100">{value}</div>
      <div className="font-mono text-[9px] uppercase tracking-[0.2em] text-cyan-300/45">
        {label}
      </div>
    </div>
  )
}
