import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  ArrowLeft,
  RefreshCcw,
  Loader2,
  Radio,
  AlertTriangle,
  Cpu,
  Globe,
  Clock,
  Activity,
  Play,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { Sparkline } from '@/components/viz/Sparkline'
import { formatTime } from '@/lib/format'
import { cn } from '@/lib/utils'
import { useDeviceBySn } from '@core/hooks/api/useDevices'
import { useIndicatorList } from '@core/hooks/api/useIndicatorsLibrary'
import { useAggregatedMetricsByDevices } from '@core/hooks/api/usePmQuery'
import type { DeviceType } from '@core/types/indicatorLibrary'
import type { Granularity, AggregatedRow } from '@core/types/pmDashboard'

// ---------------------------------------------------------------------------
// /reports/station/:sn · 基站报表详情（钻取）
//   useParams 取设备 SN → useDeviceBySn 拉真实设备台账记录；
//   按设备制式从指标库取 KPI 候选，自动选前 N 条，经 useAggregatedMetricsByDevices
//   拉「按设备」的真实 KPI 时序，渲染霓虹趋势卡。全程真实数据，无恒定占位。
// ---------------------------------------------------------------------------

const COLORS = ['#00f0ff', '#00ff88', '#ffaa00', '#a855f7', '#ff00aa', '#5b9eff', '#ff7a1a', '#ffd400']

const NETWORK_TO_DEVICE_TYPE: Record<string, DeviceType> = {
  lte: 'ENB',
  enb: 'ENB',
  nr: 'GNB',
  gnb: 'GNB',
  '5g': 'GNB',
  gsm: 'GSM',
}

const GRAN_OPTS: { label: string; value: Granularity }[] = [
  { label: '15分钟', value: '15min' },
  { label: '小时', value: 'hourly' },
  { label: '天', value: 'daily' },
]

const RANGE_OPTS: { label: string; hours: number }[] = [
  { label: '近 24 小时', hours: 24 },
  { label: '近 7 天', hours: 24 * 7 },
  { label: '近 30 天', hours: 24 * 30 },
]

const MAX_METRICS = 8

const ALARM_COLOR: Record<string, string> = {
  critical: '#ff2d6f',
  major: '#ff7a1a',
  minor: '#ffd400',
  warning: '#5b9eff',
  none: '#00ff88',
}

interface Submitted {
  metricPaths: string[]
  granularity: Granularity
  startTime: string
  endTime: string
}

export default function StationDetailPage() {
  const navigate = useNavigate()
  const { sn = '' } = useParams<{ sn: string }>()
  const decodedSn = decodeURIComponent(sn)

  const { data: device, isLoading: devLoading, isError: devError, error: devErrObj, isFetching, refetch } =
    useDeviceBySn(decodedSn)

  const deviceType: DeviceType = useMemo(() => {
    const nt = (device?.networkType || '').toLowerCase()
    return NETWORK_TO_DEVICE_TYPE[nt] ?? 'ENB'
  }, [device?.networkType])

  // 指标候选（真实指标库，按制式）；只取 KPI（非计数器）作为基站报表趋势。
  const { data: indicatorData, isLoading: indLoading } = useIndicatorList(deviceType, { pageSize: 200 })
  const indicators = useMemo(() => (indicatorData?.items ?? []).filter((it) => !it.isCounter), [indicatorData])
  const metricLabels = useMemo(() => {
    const m: Record<string, string> = {}
    indicators.forEach((it) => {
      m[it.id] = it.cnName || it.enName || it.name || it.id
    })
    return m
  }, [indicators])

  const [selectedMetrics, setSelectedMetrics] = useState<string[]>([])
  const [granularity, setGranularity] = useState<Granularity>('hourly')
  const [rangeHours, setRangeHours] = useState(24 * 7)
  const [submitted, setSubmitted] = useState<Submitted | null>(null)

  // 指标库就绪后自动选前几条 KPI（让详情页默认即出真实图，不空等用户）。
  useEffect(() => {
    if (indicators.length === 0) return
    setSelectedMetrics((prev) => (prev.length > 0 ? prev : indicators.slice(0, 4).map((it) => it.id)))
  }, [indicators])

  // SN 切换时重置已选/已提交。
  useEffect(() => {
    setSelectedMetrics([])
    setSubmitted(null)
  }, [decodedSn])

  const baseParams = useMemo(() => {
    if (!submitted) return null
    return {
      granularity: submitted.granularity,
      metricPaths: submitted.metricPaths,
      startTime: submitted.startTime,
      endTime: submitted.endTime,
      limit: 5000,
      fillEmpty: true,
    }
  }, [submitted])

  const {
    data: rows,
    total,
    truncated,
    isLoading: aggLoading,
    isFetching: aggFetching,
    isError: aggError,
    errors,
    refetch: refetchAgg,
  } = useAggregatedMetricsByDevices(
    baseParams ?? { granularity: 'hourly', metricPaths: [], startTime: undefined, endTime: undefined },
    submitted && device ? [device.sn] : [],
    Boolean(baseParams && device),
  )

  const grouped = useMemo(() => {
    if (!submitted) return [] as { metricPath: string; displayName: string; rows: AggregatedRow[] }[]
    return submitted.metricPaths.map((mp) => {
      const sub = rows.filter((r) => r.metricPath === mp)
      const displayName = sub.find((r) => r.displayName)?.displayName ?? metricLabels[mp] ?? mp
      return { metricPath: mp, displayName, rows: sub }
    })
  }, [rows, submitted, metricLabels])

  const toggleMetric = (id: string) => {
    setSelectedMetrics((prev) => {
      if (prev.includes(id)) return prev.filter((x) => x !== id)
      if (prev.length >= MAX_METRICS) return prev
      return [...prev, id]
    })
  }

  const canQuery = Boolean(device) && selectedMetrics.length > 0

  const runQuery = () => {
    if (!canQuery) return
    const end = new Date()
    const start = new Date(end.getTime() - rangeHours * 3600_000)
    setSubmitted({
      metricPaths: [...selectedMetrics],
      granularity,
      startTime: start.toISOString(),
      endTime: end.toISOString(),
    })
  }

  const alarm = device?.alarmLevel || 'none'
  const aColor = ALARM_COLOR[alarm] ?? '#6b86b6'

  return (
    <PageShell
      code="F06"
      title={`STATION · ${device?.name || decodedSn}`}
      subtitle="STATION KPI DRILL-DOWN · 基站 KPI 趋势钻取"
      isFetching={isFetching || aggFetching}
      bare
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/reports/station')}>
            返回台账
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => { void refetch(); if (submitted) void refetchAgg() }}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {devLoading ? (
        <Centered>
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">LOADING STATION…</span>
        </Centered>
      ) : devError ? (
        <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
          STATION LOAD FAILED · {devErrObj instanceof Error ? devErrObj.message : '加载失败'}
        </div>
      ) : !device ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <AlertTriangle className="size-9 text-cyan-300/40" />
          <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
            未找到基站 · {decodedSn}
          </div>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/reports/station')}>
            返回台账
          </NeonButton>
        </div>
      ) : (
        <div className="space-y-3">
          {/* 基站台账卡（真实设备字段） */}
          <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
            <InfoTile icon={<Radio className="size-4" />} label="SN / 名称" value={device.sn} sub={device.name || '—'} color="#00f0ff" />
            <InfoTile
              icon={<Cpu className="size-4" />}
              label="制式 / 型号"
              value={(device.networkType || '—').toUpperCase()}
              sub={device.productClass || device.deviceModel || '—'}
              color="#a855f7"
            />
            <InfoTile
              icon={<Globe className="size-4" />}
              label="区域 / IP"
              value={device.region || device.site || '—'}
              sub={device.ipAddress || '—'}
              color="#00ff88"
            />
            <div className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3" style={{ borderLeftColor: aColor }}>
              <div className="scanline" />
              <div className="relative">
                <div className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/65">
                  <Clock className="size-4" style={{ color: aColor }} />
                  状态 / 告警
                </div>
                <div className="mt-1 flex items-center gap-2">
                  <StatusBadge status={device.isOnline ? 'online' : 'offline'} label={device.isOnline ? '在线' : '离线'} />
                  <span className="chip" style={{ color: aColor }}>
                    {alarm === 'none' ? '无告警' : alarm.toUpperCase()}
                  </span>
                </div>
                <div className="mt-1 truncate font-mono text-[10px] text-cyan-300/45">
                  最近在线 {device.lastOnlineTime ? formatTime(device.lastOnlineTime) : '—'}
                </div>
              </div>
            </div>
          </div>

          <div className="grid grid-cols-12 gap-3">
            {/* 左：KPI 选择 + 窗口 */}
            <div className="col-span-12 space-y-3 lg:col-span-4 xl:col-span-3">
              <GlassPanel title="KPI 指标 · METRIC" meta={`${selectedMetrics.length}/${MAX_METRICS}`}>
                <div className="max-h-72 overflow-auto p-2">
                  {indLoading ? (
                    <Centered small>
                      <Loader2 className="size-4 animate-spin" />
                      <span className="font-mono text-[10px] uppercase tracking-[0.2em]">LOADING LIBRARY…</span>
                    </Centered>
                  ) : indicators.length === 0 ? (
                    <div className="py-6 text-center font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/40">
                      该制式无 KPI 指标
                    </div>
                  ) : (
                    indicators.map((it) => {
                      const checked = selectedMetrics.includes(it.id)
                      const disabled = !checked && selectedMetrics.length >= MAX_METRICS
                      return (
                        <button
                          key={it.id}
                          type="button"
                          disabled={disabled}
                          onClick={() => toggleMetric(it.id)}
                          className={cn(
                            'flex w-full items-center gap-2 border-b border-cyan-500/8 px-2 py-1.5 text-left transition-colors hover:bg-cyan-500/5',
                            checked && 'bg-cyan-500/10',
                            disabled && 'cursor-not-allowed opacity-35',
                          )}
                        >
                          <span
                            className={cn(
                              'flex size-3.5 shrink-0 items-center justify-center rounded-[2px] border',
                              checked ? 'border-cyan-400 bg-cyan-400/80' : 'border-cyan-500/40',
                            )}
                          >
                            {checked ? <span className="size-1.5 rounded-[1px] bg-[#03050d]" /> : null}
                          </span>
                          <div className="min-w-0 flex-1">
                            <div className="truncate text-[11px] text-cyan-100">{it.cnName || it.enName || it.name}</div>
                            <div className="truncate font-mono text-[9px] text-cyan-300/45">
                              {it.id}
                              {it.unit ? ` · ${it.unit}` : ''}
                            </div>
                          </div>
                        </button>
                      )
                    })
                  )}
                </div>
              </GlassPanel>

              <GlassPanel title="粒度 / 时段 · WINDOW">
                <div className="space-y-2.5 p-3">
                  <div className="flex flex-wrap gap-1.5">
                    {GRAN_OPTS.map((g) => (
                      <button
                        key={g.value}
                        type="button"
                        onClick={() => setGranularity(g.value)}
                        className={cn(
                          'chip transition-all',
                          granularity === g.value ? 'text-cyan-200 shadow-[0_0_10px_currentColor]' : 'text-cyan-300/55 opacity-70 hover:opacity-100',
                        )}
                      >
                        {g.label}
                      </button>
                    ))}
                  </div>
                  <div className="flex flex-wrap gap-1.5">
                    {RANGE_OPTS.map((r) => (
                      <button
                        key={r.hours}
                        type="button"
                        onClick={() => setRangeHours(r.hours)}
                        className={cn(
                          'chip transition-all',
                          rangeHours === r.hours ? 'text-cyan-200 shadow-[0_0_10px_currentColor]' : 'text-cyan-300/55 opacity-70 hover:opacity-100',
                        )}
                      >
                        {r.label}
                      </button>
                    ))}
                  </div>
                  <NeonButton icon={<Play />} onClick={runQuery} disabled={!canQuery} className="w-full justify-center">
                    出图 · PLOT
                  </NeonButton>
                </div>
              </GlassPanel>
            </div>

            {/* 右：趋势 */}
            <div className="col-span-12 lg:col-span-8 xl:col-span-9">
              {!submitted ? (
                <GlassPanel title="KPI TREND · 趋势">
                  <div className="flex flex-col items-center justify-center gap-3 py-20">
                    <Activity className="size-10 text-cyan-300/40" />
                    <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/45">
                      选择 KPI 后点「出图」
                    </div>
                  </div>
                </GlassPanel>
              ) : aggLoading || aggFetching ? (
                <GlassPanel title="KPI TREND · 趋势" meta="SYNC">
                  <Centered>
                    <Loader2 className="size-4 animate-spin" />
                    <span className="font-mono text-xs uppercase tracking-[0.2em]">QUERYING TIME-SERIES…</span>
                  </Centered>
                </GlassPanel>
              ) : aggError ? (
                <GlassPanel title="KPI TREND · 趋势">
                  <div className="m-3 border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
                    QUERY FAILED · {(errors[0] as Error)?.message ?? '未知错误'}
                  </div>
                </GlassPanel>
              ) : (
                <div className="space-y-3">
                  <GlassPanel title="KPI TREND · 趋势" meta={`${grouped.length} METRIC · ${total} ROW`}>
                    <div className="flex flex-wrap items-center gap-2 p-3 font-mono text-[10px] text-cyan-300/60">
                      <span className="chip text-cyan-200">{device.sn}</span>
                      <span className="chip text-cyan-300/70">{(device.networkType || '—').toUpperCase()}</span>
                      <span className="chip text-cyan-300/70">{granularity}</span>
                      {truncated ? <span className="chip text-[#ffaa00]">已截断 · 仅显示 {rows.length}/{total}</span> : null}
                    </div>
                  </GlassPanel>
                  <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
                    {grouped.map((g, i) => (
                      <TrendCard key={g.metricPath} metricPath={g.metricPath} displayName={g.displayName} rows={g.rows} color={COLORS[i % COLORS.length]} />
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </PageShell>
  )
}

function TrendCard({
  metricPath,
  displayName,
  rows,
  color,
}: {
  metricPath: string
  displayName: string
  rows: AggregatedRow[]
  color: string
}) {
  const { series, latest, avg, min, max, points } = useMemo(() => {
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
    return { series: vals, latest: vals[vals.length - 1], avg: sum / vals.length, min: Math.min(...vals), max: Math.max(...vals), points: vals.length }
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
            该窗口无样本
          </div>
        ) : (
          <>
            <div className="mb-3 flex items-end justify-between">
              <div>
                <div className="font-display text-3xl font-bold leading-none" style={{ color, textShadow: `0 0 8px ${color}` }}>
                  {fmt(latest)}
                </div>
                <div className="mt-1 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/50">LATEST</div>
              </div>
              <Sparkline data={series} color={color} width={180} height={48} />
            </div>
            <div className="grid grid-cols-3 gap-2 border-t border-cyan-500/12 pt-2.5">
              <MiniStat label="AVG" value={fmt(avg)} />
              <MiniStat label="MIN" value={fmt(min)} />
              <MiniStat label="MAX" value={fmt(max)} />
            </div>
            <div className="mt-2 truncate font-mono text-[9px] text-cyan-300/35">{metricPath}</div>
          </>
        )}
      </div>
    </GlassPanel>
  )
}

function MiniStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="text-center">
      <div className="font-display text-sm font-bold text-cyan-100">{value}</div>
      <div className="font-mono text-[9px] uppercase tracking-[0.2em] text-cyan-300/45">{label}</div>
    </div>
  )
}

function InfoTile({
  icon,
  label,
  value,
  sub,
  color,
}: {
  icon: React.ReactNode
  label: string
  value: string
  sub: string
  color: string
}) {
  return (
    <div className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3" style={{ borderLeftColor: color }}>
      <div className="scanline" />
      <div className="relative">
        <div className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/65">
          <span style={{ color }}>{icon}</span>
          {label}
        </div>
        <div className="mt-1 truncate font-display text-base font-bold text-cyan-100">{value}</div>
        <div className="truncate font-mono text-[10px] text-cyan-300/45">{sub}</div>
      </div>
    </div>
  )
}

function Centered({ children, small }: { children: React.ReactNode; small?: boolean }) {
  return <div className={cn('flex items-center justify-center gap-2 text-cyan-300/60', small ? 'py-6' : 'py-16')}>{children}</div>
}
