import { useMemo } from 'react'

import { GlassPanel } from '@/components/ui/GlassPanel'
import { Sparkline } from '@/components/viz/Sparkline'
import { alignPointsToGrid } from '@core/utils/buildRegularTimeGrid'
import type { AggregatedRow, Granularity } from '@core/types/pmDashboard'

interface Props {
  metricPath: string
  displayName: string
  rows: AggregatedRow[]
  color: string
  // issue #429：查询窗口起止 + 粒度（由父级透传），用于铺规整网格、缺采槽位断开线。
  startTime: string
  endTime: string
  granularity: Granularity
}

/**
 * 单指标趋势卡：把 PM 聚合直查行（long 格式）按时间桶转置成有序序列，
 * 渲染霓虹火花线 + 当前/均值/峰值/谷值四数。
 * issue #429：横轴按"查询窗口起止 + 粒度"铺规整网格(窗口内每个整点槽位都上轴),缺采
 * 槽位保留 null(不再丢弃),喂给 Sparkline 的是 (number|null)[]、null 处断开线。统计值
 * (avg/min/max/latest/points)仍只算非 null 点(保持 null=缺采、不参与统计语义)。
 */
export function MetricTrendChart({
  metricPath,
  displayName,
  rows,
  color,
  startTime,
  endTime,
  granularity,
}: Props) {
  const { series, latest, avg, min, max, points } = useMemo(() => {
    // 同桶多设备/小区取最后一条值聚一条总线；缺采行(metricValue==null)记 null 占位。
    const byBucket = new Map<string, number | null>()
    rows.forEach((r) => {
      const v = r.metricValue
      byBucket.set(r.startTime, v === undefined ? null : v)
    })
    const sparsePoints = Array.from(byBucket.entries()).map(([time, value]) => ({
      timeMs: Date.parse(time),
      value,
    }))
    // 按窗口+粒度铺规整网格，空槽 null(断档)；窗口非法/未知粒度 → grid 为空回退。
    const { values } = alignPointsToGrid(
      sparsePoints,
      Date.parse(startTime),
      Date.parse(endTime),
      granularity,
    )
    // 统计只算非 null 点。
    const nums = values.filter((v): v is number => v !== null)
    if (nums.length === 0) {
      return {
        series: values as (number | null)[],
        latest: null,
        avg: null,
        min: null,
        max: null,
        points: 0,
      }
    }
    const sum = nums.reduce((a, b) => a + b, 0)
    // latest = 最后一个非 null 点。
    const lastNonNull = [...nums]
    return {
      series: values as (number | null)[],
      latest: lastNonNull[lastNonNull.length - 1],
      avg: sum / nums.length,
      min: Math.min(...nums),
      max: Math.max(...nums),
      points: nums.length,
    }
  }, [rows, startTime, endTime, granularity])

  const fmt = (v: number | null) => {
    if (v === null) return '—'
    if (Math.abs(v) >= 1000) return v.toLocaleString(undefined, { maximumFractionDigits: 0 })
    return Number.isInteger(v) ? String(v) : v.toFixed(2)
  }

  return (
    <GlassPanel title={displayName} meta={`${points} PTS`}>
      <div className="p-4">
        {points === 0 ? (
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
