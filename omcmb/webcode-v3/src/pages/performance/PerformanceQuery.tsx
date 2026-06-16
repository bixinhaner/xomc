import { useEffect, useMemo, useState } from 'react'
import { LineChart, Loader2, Play, RefreshCcw, Search } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { cn } from '@/lib/utils'
import { useIndicatorList } from '@core/hooks/api/useIndicatorsLibrary'
import { useAggregatedMetricsByDevices } from '@core/hooks/api/usePmQuery'
import { useDeviceList } from '@core/hooks/api/useDevices'
import type { DeviceType } from '@core/types/indicatorLibrary'
import type { Granularity } from '@core/types/pmDashboard'
import type { Device } from '@core/types/device'
import { MetricTrendChart } from './MetricTrendChart'
import { useT } from '@/hooks/useT'

/**
 * F03 · 实时取数（performance/query）
 * 选制式 → 选设备（真实 useDeviceList）→ 选指标（真实指标库 useIndicatorList）→
 * 聚合直查 useAggregatedMetricsByDevices，按指标分组出趋势卡。
 */

const COLORS = ['#00f0ff', '#00ff88', '#ffaa00', '#a855f7', '#ff00aa', '#5b9eff', '#ff7a1a', '#ffd400']

type Tech = 'lte' | 'nr' | 'gsm'

const TECH_OPTS: { label: string; value: Tech; deviceType: DeviceType }[] = [
  { label: 'LTE', value: 'lte', deviceType: 'ENB' },
  { label: 'NR', value: 'nr', deviceType: 'GNB' },
  { label: 'GSM', value: 'gsm', deviceType: 'GSM' },
]

// label 改为 i18n key（在渲染处经 t() 翻译），避免模块级常量写死中文。
const GRAN_OPTS: { labelKey: string; value: Granularity }[] = [
  { labelKey: 'perf.kpiQuery.gran.15min', value: '15min' },
  { labelKey: 'perf.kpiQuery.gran.hourly', value: 'hourly' },
  { labelKey: 'perf.kpiQuery.gran.daily', value: 'daily' },
]

const RANGE_OPTS: { labelKey: string; hours: number }[] = [
  { labelKey: 'perf.kpiQuery.range.last1h', hours: 1 },
  { labelKey: 'perf.kpiQuery.range.last24h', hours: 24 },
  { labelKey: 'perf.kpiQuery.range.last7d', hours: 24 * 7 },
  { labelKey: 'perf.kpiQuery.range.last30d', hours: 24 * 30 },
]

const MAX_METRICS = 12

interface Submitted {
  deviceSn: string
  metricPaths: string[]
  granularity: Granularity
  startTime: string
  endTime: string
}

export default function PerformanceQuery() {
  const t = useT()
  const [tech, setTech] = useState<Tech>('lte')
  const deviceType = TECH_OPTS.find((t) => t.value === tech)!.deviceType
  const [deviceSn, setDeviceSn] = useState('')
  const [granularity, setGranularity] = useState<Granularity>('15min')
  const [rangeHours, setRangeHours] = useState(24)
  const [selectedMetrics, setSelectedMetrics] = useState<string[]>([])
  const [metricKeyword, setMetricKeyword] = useState('')
  const [deviceKeyword, setDeviceKeyword] = useState('')
  const [submitted, setSubmitted] = useState<Submitted | null>(null)

  useEffect(() => {
    setDeviceSn('')
    setSelectedMetrics([])
    setSubmitted(null)
  }, [tech])

  const deviceParams = useMemo(
    () => ({ page: 1, pageSize: 30, ...(deviceKeyword.trim() ? { searchText: deviceKeyword.trim() } : {}) }),
    [deviceKeyword],
  )
  const { data: deviceData, isLoading: devLoading, isError: devError } = useDeviceList(deviceParams)
  const devices: Device[] = deviceData?.items ?? []

  const metricFilter = useMemo(
    () => ({ pageSize: 300, ...(metricKeyword.trim() ? { keyword: metricKeyword.trim() } : {}) }),
    [metricKeyword],
  )
  const { data: indicatorData, isLoading: indLoading, isError: indError } = useIndicatorList(deviceType, metricFilter)
  const indicators = indicatorData?.items ?? []
  const metricLabels = useMemo(() => {
    const m: Record<string, string> = {}
    indicators.forEach((it) => {
      m[it.id] = it.cnName || it.enName || it.name || it.id
    })
    return m
  }, [indicators])

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
    data: aggRows,
    total,
    truncated,
    isLoading: aggLoading,
    isFetching: aggFetching,
    isError: aggError,
    errors,
    refetch,
  } = useAggregatedMetricsByDevices(
    baseParams ?? { granularity: '15min', metricPaths: [], startTime: undefined, endTime: undefined },
    submitted ? [submitted.deviceSn] : [],
    Boolean(baseParams),
  )

  const grouped = useMemo(() => {
    if (!submitted) return [] as { metricPath: string; displayName: string; rows: typeof aggRows }[]
    return submitted.metricPaths.map((mp) => {
      const sub = aggRows.filter((r) => r.metricPath === mp)
      const displayName = sub.find((r) => r.displayName)?.displayName ?? metricLabels[mp] ?? mp
      return { metricPath: mp, displayName, rows: sub }
    })
  }, [aggRows, submitted, metricLabels])

  const toggleMetric = (id: string) => {
    setSelectedMetrics((prev) => {
      if (prev.includes(id)) return prev.filter((x) => x !== id)
      if (prev.length >= MAX_METRICS) return prev
      return [...prev, id]
    })
  }

  const canQuery = Boolean(deviceSn) && selectedMetrics.length > 0

  const runQuery = () => {
    if (!canQuery) return
    const end = new Date()
    const start = new Date(end.getTime() - rangeHours * 3600_000)
    setSubmitted({
      deviceSn,
      metricPaths: [...selectedMetrics],
      granularity,
      startTime: start.toISOString(),
      endTime: end.toISOString(),
    })
  }

  return (
    <PageShell
      code="F03"
      title={`TIME-SERIES QUERY · ${t('perf.kpiQuery.v3.realtimeQuery')}`}
      subtitle="DEVICE × METRIC AGGREGATION · LIVE PLOT"
      isFetching={aggFetching}
      bare
    >
      <div className="grid grid-cols-12 gap-3">
        {/* 左：条件 */}
        <div className="col-span-12 space-y-3 lg:col-span-4 xl:col-span-3">
          <GlassPanel title={`${t('perf.kpiQuery.tech')} · TECH`}>
            <div className="flex gap-1.5 p-3">
              {TECH_OPTS.map((t) => (
                <button
                  key={t.value}
                  type="button"
                  onClick={() => setTech(t.value)}
                  className={cn(
                    'chip flex-1 justify-center transition-all',
                    tech === t.value
                      ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                      : 'text-cyan-300/55 opacity-70 hover:opacity-100',
                  )}
                >
                  {t.label}
                </button>
              ))}
            </div>
          </GlassPanel>

          <GlassPanel title={`${t('perf.kpiQuery.device')} · DEVICE`} meta={deviceSn ? '1' : '0'}>
            <div className="p-3">
              <div className="relative mb-2">
                <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
                <input
                  className="neon-input w-full pl-9"
                  placeholder={t('perf.kpiQuery.deviceSearchPlaceholder')}
                  value={deviceKeyword}
                  onChange={(e) => setDeviceKeyword(e.target.value)}
                />
              </div>
              <div className="max-h-56 overflow-auto rounded-sm border border-cyan-500/12">
                {devLoading ? (
                  <Loading text="SCANNING FLEET…" small />
                ) : devError ? (
                  <ErrorBox text="FLEET SCAN FAILED" small />
                ) : devices.length === 0 ? (
                  <EmptyBox text="NO DEVICE" small />
                ) : (
                  devices.map((d) => (
                    <button
                      key={d.id}
                      type="button"
                      onClick={() => setDeviceSn(d.sn)}
                      className={cn(
                        'flex w-full items-center gap-2 border-b border-cyan-500/8 px-2.5 py-1.5 text-left transition-colors hover:bg-cyan-500/5',
                        deviceSn === d.sn && 'bg-cyan-500/10',
                      )}
                    >
                      <StatusBadge status={d.isOnline ? 'online' : 'offline'} label={d.isOnline ? 'ON' : 'OFF'} />
                      <div className="min-w-0 flex-1">
                        <div className="truncate font-mono text-[11px] text-cyan-100">{d.sn}</div>
                        <div className="truncate text-[10px] text-cyan-300/55">
                          {d.deviceName || d.name || '—'}
                          {d.productClass ? ` · ${d.productClass}` : ''}
                        </div>
                      </div>
                    </button>
                  ))
                )}
              </div>
            </div>
          </GlassPanel>

          <GlassPanel title={`${t('perf.kpiQuery.metric')} · METRIC`} meta={`${selectedMetrics.length}/${MAX_METRICS}`}>
            <div className="p-3">
              <div className="relative mb-2">
                <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
                <input
                  className="neon-input w-full pl-9"
                  placeholder={t('perf.kpiQuery.metricSearchPlaceholder')}
                  value={metricKeyword}
                  onChange={(e) => setMetricKeyword(e.target.value)}
                />
              </div>
              <div className="max-h-64 overflow-auto rounded-sm border border-cyan-500/12">
                {indLoading ? (
                  <Loading text="LOADING LIBRARY…" small />
                ) : indError ? (
                  <ErrorBox text="LIBRARY LOAD FAILED" small />
                ) : indicators.length === 0 ? (
                  <EmptyBox text="NO INDICATOR" small />
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
                          'flex w-full items-center gap-2 border-b border-cyan-500/8 px-2.5 py-1.5 text-left transition-colors hover:bg-cyan-500/5',
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
                          <div className="truncate text-[11px] text-cyan-100">
                            {it.cnName || it.enName || it.name}
                          </div>
                          <div className="truncate font-mono text-[9px] text-cyan-300/45">
                            {it.id}
                            {it.isCounter ? ' · CTR' : ' · KPI'}
                            {it.unit ? ` · ${it.unit}` : ''}
                          </div>
                        </div>
                      </button>
                    )
                  })
                )}
              </div>
            </div>
          </GlassPanel>

          <GlassPanel title={`${t('perf.kpiQuery.granularityWindow')} · WINDOW`}>
            <div className="space-y-2.5 p-3">
              <div className="flex flex-wrap gap-1.5">
                {GRAN_OPTS.map((g) => (
                  <button
                    key={g.value}
                    type="button"
                    onClick={() => setGranularity(g.value)}
                    className={cn(
                      'chip transition-all',
                      granularity === g.value
                        ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                        : 'text-cyan-300/55 opacity-70 hover:opacity-100',
                    )}
                  >
                    {t(g.labelKey)}
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
                      rangeHours === r.hours
                        ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                        : 'text-cyan-300/55 opacity-70 hover:opacity-100',
                    )}
                  >
                    {t(r.labelKey)}
                  </button>
                ))}
              </div>
            </div>
          </GlassPanel>

          <div className="flex gap-2">
            <NeonButton icon={<Play />} onClick={runQuery} disabled={!canQuery} className="flex-1 justify-center">
              {t('perf.kpiQuery.plot')} · PLOT
            </NeonButton>
            <NeonButton icon={<RefreshCcw />} onClick={() => refetch()} disabled={!submitted}>
              {t('perf.kpiQuery.refresh')}
            </NeonButton>
          </div>
        </div>

        {/* 右：结果 */}
        <div className="col-span-12 lg:col-span-8 xl:col-span-9">
          {!submitted ? (
            <GlassPanel title={`QUERY RESULT · ${t('perf.kpiQuery.v3.queryResult')}`}>
              <div className="flex flex-col items-center justify-center gap-3 py-20">
                <LineChart className="size-10 text-cyan-300/40" />
                <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/45">
                  {t('perf.kpiQuery.v3.emptyHint')}
                </div>
              </div>
            </GlassPanel>
          ) : aggLoading || aggFetching ? (
            <GlassPanel title={`QUERY RESULT · ${t('perf.kpiQuery.v3.queryResult')}`} meta="SYNC">
              <Loading text="QUERYING TIME-SERIES…" />
            </GlassPanel>
          ) : aggError ? (
            <GlassPanel title={`QUERY RESULT · ${t('perf.kpiQuery.v3.queryResult')}`}>
              <div className="p-4">
                <ErrorBox text={`QUERY FAILED · ${(errors[0] as Error)?.message ?? t('perf.kpiQuery.unknownError')}`} />
              </div>
            </GlassPanel>
          ) : (
            <div className="space-y-3">
              <GlassPanel title={`QUERY RESULT · ${t('perf.kpiQuery.v3.queryResult')}`} meta={`${grouped.length} METRIC · ${total} ROW`}>
                <div className="flex flex-wrap items-center gap-2 p-3 font-mono text-[10px] text-cyan-300/60">
                  <span className="chip text-cyan-200">{submitted.deviceSn}</span>
                  <span className="chip text-cyan-300/70">{tech.toUpperCase()}</span>
                  <span className="chip text-cyan-300/70">{granularity}</span>
                  {truncated ? (
                    <span className="chip text-[#ffaa00]">
                      {t('perf.kpiQuery.truncatedShown', { shown: aggRows.length, total })}
                    </span>
                  ) : null}
                </div>
              </GlassPanel>
              <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
                {grouped.map((g, i) => (
                  <MetricTrendChart
                    key={g.metricPath}
                    metricPath={g.metricPath}
                    displayName={g.displayName}
                    rows={g.rows}
                    color={COLORS[i % COLORS.length]}
                    startTime={submitted.startTime}
                    endTime={submitted.endTime}
                    granularity={submitted.granularity}
                  />
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </PageShell>
  )
}

/* ───────── 复用小件 ───────── */

function Loading({ text, small }: { text: string; small?: boolean }) {
  return (
    <div className={cn('flex items-center justify-center gap-2 text-cyan-300/60', small ? 'py-6' : 'py-16')}>
      <Loader2 className={cn('animate-spin', small ? 'size-4' : 'size-5')} />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">{text}</span>
    </div>
  )
}

function ErrorBox({ text, small }: { text: string; small?: boolean }) {
  return (
    <div
      className={cn(
        'border border-rose-500/40 bg-rose-500/5 font-mono text-rose-300',
        small ? 'px-3 py-4 text-xs' : 'px-4 py-6 text-sm',
      )}
    >
      {text}
    </div>
  )
}

function EmptyBox({ text, small }: { text: string; small?: boolean }) {
  return (
    <div
      className={cn(
        'flex items-center justify-center font-mono uppercase tracking-[0.2em] text-cyan-300/40',
        small ? 'py-6 text-[10px]' : 'py-12 text-xs',
      )}
    >
      {text}
    </div>
  )
}
