import { useEffect, useMemo, useState } from 'react'
import { Activity, Database, Gauge, Layers, LineChart, Loader2, Play, RefreshCcw, Search } from 'lucide-react'

import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { Sparkline } from '@/components/viz/Sparkline'
import { cn } from '@/lib/utils'
import { useKPIList } from '@core/hooks/api/usePerformance'
import { useIndicatorSummary, useIndicatorList } from '@core/hooks/api/useIndicatorsLibrary'
import { useAggregatedMetricsByDevices } from '@core/hooks/api/usePmQuery'
import { useDeviceList } from '@core/hooks/api/useDevices'
import type { DeviceType } from '@core/types/indicatorLibrary'
import type { Granularity } from '@core/types/pmDashboard'
import type { Device } from '@core/types/device'
import { MetricTrendChart } from './MetricTrendChart'

const COLORS = ['#00f0ff', '#00ff88', '#ffaa00', '#a855f7', '#ff00aa', '#5b9eff', '#ff7a1a', '#ffd400']

type Tab = 'spectrum' | 'library' | 'query'

const TABS: { key: Tab; label: string; icon: typeof Gauge }[] = [
  { key: 'spectrum', label: 'KPI 光谱', icon: Gauge },
  { key: 'query', label: '实时取数', icon: LineChart },
  { key: 'library', label: '指标库', icon: Database },
]

export function PerformancePage() {
  const [tab, setTab] = useState<Tab>('spectrum')

  return (
    <div className="warp-in flex h-full w-full flex-col gap-4">
      {/* 标题带 + 子视图切换 */}
      <div className="flex items-end justify-between gap-4">
        <div className="flex items-end gap-4">
          <div className="font-display text-4xl font-bold leading-none text-cyan-300/85 text-glow">
            F03
          </div>
          <div>
            <div className="font-display text-lg leading-tight text-cyan-100">
              PERFORMANCE · 性能管理
            </div>
            <div className="font-mono text-[11px] uppercase tracking-[0.18em] text-cyan-300/55">
              KPI / COUNTER · INDICATOR LIBRARY · TIME-SERIES QUERY
            </div>
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {TABS.map((t) => {
            const Icon = t.icon
            return (
              <button
                key={t.key}
                type="button"
                onClick={() => setTab(t.key)}
                className={cn(
                  'chip flex items-center gap-1.5 transition-all',
                  tab === t.key
                    ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                    : 'text-cyan-300/55 opacity-70 hover:opacity-100'
                )}
              >
                <Icon className="size-3.5" />
                {t.label}
              </button>
            )
          })}
        </div>
      </div>

      <div className="flex-1 min-h-0 overflow-auto">
        {tab === 'spectrum' && <SpectrumView />}
        {tab === 'query' && <QueryView />}
        {tab === 'library' && <LibraryView />}
      </div>
    </div>
  )
}

/* ───────────────────────── KPI 光谱（概览） ───────────────────────── */

function SpectrumView() {
  const [keyword, setKeyword] = useState('')
  const { data, isLoading, isError, isFetching } = useKPIList({ page: 1, pageSize: 60, keyword })
  const { data: summary } = useIndicatorSummary()
  const items = data?.items ?? []
  const total = data?.total ?? 0

  // 指标库概览统计（真实 /indicators/summary）：平台数、指标总数、内置/自定义平台数。
  const libStats = useMemo(() => {
    const platforms = summary?.items ?? []
    const indicators = platforms.reduce((s, p) => s + (p.indicators ?? 0), 0)
    const builtin = platforms.filter((p) => p.source === 'builtin').length
    const custom = platforms.filter((p) => p.source === 'custom').length
    return { platforms: platforms.length, indicators, builtin, custom }
  }, [summary])

  return (
    <div className="space-y-3">
      {/* 统计带 */}
      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        <MetricStat label="KPI 定义 · DEFS" value={total.toLocaleString()} color="#00f0ff" icon={<Gauge className="size-4" />} />
        <MetricStat label="指标总数 · INDICATORS" value={libStats.indicators.toLocaleString()} color="#00ff88" icon={<Activity className="size-4" />} />
        <MetricStat label="平台 · PLATFORMS" value={libStats.platforms.toLocaleString()} color="#a855f7" icon={<Layers className="size-4" />} />
        <MetricStat label="自定义 · CUSTOM" value={libStats.custom.toLocaleString()} color="#ffaa00" icon={<Database className="size-4" />} />
      </div>

      <GlassPanel
        title="KPI SPECTRUM · 指标光谱"
        meta={
          <div className="flex items-center gap-2">
            {isFetching ? <Loader2 className="size-3 animate-spin" /> : null}
            <span>{items.length} ON BOARD</span>
          </div>
        }
      >
        <div className="border-b border-cyan-500/12 px-3.5 py-2.5">
          <div className="relative inline-block">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="搜索 KPI 名称 / 编码"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
        </div>

        <div className="p-3.5">
          {isLoading ? (
            <Loading text="SYNCING SPECTRUM…" />
          ) : isError ? (
            <ErrorBox text="SPECTRUM SYNC FAILED" />
          ) : items.length === 0 ? (
            <EmptyBox text="NO KPI ON BOARD" />
          ) : (
            <div className="grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-4">
              {items.map((kpi, i) => {
                const color = COLORS[i % COLORS.length]
                const value = pseudoVal(kpi.kpiCode || kpi.kpiName, 100)
                const trend = pseudoTrend(kpi.kpiCode || kpi.kpiName, 24)
                return (
                  <GlassPanel key={kpi.id} title={kpi.kpiCode} meta={kpi.unit}>
                    <div className="flex flex-col items-center gap-3 p-4">
                      <RadialGauge value={value} label={kpi.unit || ''} size={132} color={color} />
                      <div className="text-center">
                        <div className="font-display text-sm font-bold text-cyan-100">{kpi.kpiName}</div>
                        <div className="mt-1 font-mono text-[10px] text-cyan-300/55">
                          {kpi.category || 'KPI'}
                        </div>
                      </div>
                      <div className="w-full">
                        <Sparkline data={trend} color={color} width={220} height={36} />
                      </div>
                    </div>
                  </GlassPanel>
                )
              })}
            </div>
          )}
        </div>
      </GlassPanel>
      <p className="px-1 font-mono text-[10px] text-cyan-300/35">
        光谱仪表为指标库定义的可视化预览；真实时序请走「实时取数」按设备查询。
      </p>
    </div>
  )
}

/* ───────────────────────── 实时取数（设备 + 指标聚合查询） ───────────────────────── */

type Tech = 'lte' | 'nr' | 'gsm'

const TECH_OPTS: { label: string; value: Tech; deviceType: DeviceType }[] = [
  { label: 'LTE', value: 'lte', deviceType: 'ENB' },
  { label: 'NR', value: 'nr', deviceType: 'GNB' },
  { label: 'GSM', value: 'gsm', deviceType: 'GSM' },
]

const GRAN_OPTS: { label: string; value: Granularity }[] = [
  { label: '15分钟', value: '15min' },
  { label: '小时', value: 'hourly' },
  { label: '天', value: 'daily' },
]

const RANGE_OPTS: { label: string; hours: number }[] = [
  { label: '近 1 小时', hours: 1 },
  { label: '近 24 小时', hours: 24 },
  { label: '近 7 天', hours: 24 * 7 },
  { label: '近 30 天', hours: 24 * 30 },
]

const MAX_METRICS = 12

interface Submitted {
  deviceSn: string
  metricPaths: string[]
  granularity: Granularity
  startTime: string
  endTime: string
}

function QueryView() {
  const [tech, setTech] = useState<Tech>('lte')
  const deviceType = TECH_OPTS.find((t) => t.value === tech)!.deviceType
  const [deviceSn, setDeviceSn] = useState<string>('')
  const [granularity, setGranularity] = useState<Granularity>('15min')
  const [rangeHours, setRangeHours] = useState(24)
  const [selectedMetrics, setSelectedMetrics] = useState<string[]>([])
  const [metricKeyword, setMetricKeyword] = useState('')
  const [deviceKeyword, setDeviceKeyword] = useState('')
  const [submitted, setSubmitted] = useState<Submitted | null>(null)

  // 切制式：清空设备与指标（设备制式应与指标库制式一致）。
  useEffect(() => {
    setDeviceSn('')
    setSelectedMetrics([])
    setSubmitted(null)
  }, [tech])

  // 设备候选（真实 /devices；按制式 networkType 不强过滤，靠搜索框收敛）。
  const deviceParams = useMemo(
    () => ({ page: 1, pageSize: 30, ...(deviceKeyword.trim() ? { searchText: deviceKeyword.trim() } : {}) }),
    [deviceKeyword]
  )
  const { data: deviceData, isLoading: devLoading, isError: devError } = useDeviceList(deviceParams)
  const devices: Device[] = deviceData?.items ?? []

  // 指标候选（真实指标库，按制式 deviceType）。
  const metricFilter = useMemo(
    () => ({ pageSize: 300, ...(metricKeyword.trim() ? { keyword: metricKeyword.trim() } : {}) }),
    [metricKeyword]
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

  // 聚合取数（真实 /pm/metrics/aggregated，单设备）。
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
    refetch,
  } = useAggregatedMetricsByDevices(
    baseParams ?? { granularity: '15min', metricPaths: [], startTime: undefined, endTime: undefined },
    submitted ? [submitted.deviceSn] : [],
    Boolean(baseParams)
  )

  // 按 metricPath 分组聚合行（每指标一张趋势卡）。
  const grouped = useMemo(() => {
    if (!submitted) return [] as { metricPath: string; displayName: string; rows: typeof rows }[]
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
    <div className="grid grid-cols-12 gap-3">
      {/* 左：条件面板 */}
      <div className="col-span-12 lg:col-span-4 xl:col-span-3 space-y-3">
        <GlassPanel title="制式 · TECH">
          <div className="flex gap-1.5 p-3">
            {TECH_OPTS.map((t) => (
              <button
                key={t.value}
                type="button"
                onClick={() => setTech(t.value)}
                className={cn(
                  'chip flex-1 justify-center transition-all',
                  tech === t.value ? 'text-cyan-200 shadow-[0_0_10px_currentColor]' : 'text-cyan-300/55 opacity-70 hover:opacity-100'
                )}
              >
                {t.label}
              </button>
            ))}
          </div>
        </GlassPanel>

        <GlassPanel title="设备 · DEVICE" meta={deviceSn ? '已选 1' : '未选'}>
          <div className="p-3">
            <div className="relative mb-2">
              <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
              <input
                className="neon-input w-full pl-9"
                placeholder="SN / 名称 / IP 搜索"
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
                      deviceSn === d.sn && 'bg-cyan-500/10'
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

        <GlassPanel title={`指标 · METRIC`} meta={`${selectedMetrics.length}/${MAX_METRICS}`}>
          <div className="p-3">
            <div className="relative mb-2">
              <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
              <input
                className="neon-input w-full pl-9"
                placeholder="指标名 / 编码搜索"
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
                        disabled && 'cursor-not-allowed opacity-35'
                      )}
                    >
                      <span
                        className={cn(
                          'flex size-3.5 shrink-0 items-center justify-center rounded-[2px] border',
                          checked ? 'border-cyan-400 bg-cyan-400/80' : 'border-cyan-500/40'
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
                    granularity === g.value ? 'text-cyan-200 shadow-[0_0_10px_currentColor]' : 'text-cyan-300/55 opacity-70 hover:opacity-100'
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
                    rangeHours === r.hours ? 'text-cyan-200 shadow-[0_0_10px_currentColor]' : 'text-cyan-300/55 opacity-70 hover:opacity-100'
                  )}
                >
                  {r.label}
                </button>
              ))}
            </div>
          </div>
        </GlassPanel>

        <div className="flex gap-2">
          <NeonButton icon={<Play />} onClick={runQuery} disabled={!canQuery} className="flex-1 justify-center">
            出图 · PLOT
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()} disabled={!submitted}>
            刷新
          </NeonButton>
        </div>
      </div>

      {/* 右：结果 */}
      <div className="col-span-12 lg:col-span-8 xl:col-span-9">
        {!submitted ? (
          <GlassPanel title="QUERY RESULT · 取数结果">
            <div className="flex flex-col items-center justify-center gap-3 py-20">
              <LineChart className="size-10 text-cyan-300/40" />
              <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/45">
                选择设备 + 指标后点「出图」
              </div>
            </div>
          </GlassPanel>
        ) : aggLoading || aggFetching ? (
          <GlassPanel title="QUERY RESULT · 取数结果" meta="SYNC">
            <Loading text="QUERYING TIME-SERIES…" />
          </GlassPanel>
        ) : aggError ? (
          <GlassPanel title="QUERY RESULT · 取数结果">
            <div className="p-4">
              <ErrorBox text={`QUERY FAILED · ${(errors[0] as Error)?.message ?? '未知错误'}`} />
            </div>
          </GlassPanel>
        ) : (
          <div className="space-y-3">
            <GlassPanel title="QUERY RESULT · 取数结果" meta={`${grouped.length} METRIC · ${total} ROW`}>
              <div className="flex flex-wrap items-center gap-2 p-3 font-mono text-[10px] text-cyan-300/60">
                <span className="chip text-cyan-200">{submitted.deviceSn}</span>
                <span className="chip text-cyan-300/70">{tech.toUpperCase()}</span>
                <span className="chip text-cyan-300/70">{granularity}</span>
                {truncated ? (
                  <span className="chip text-[#ffaa00]">已截断 · 仅显示 {rows.length}/{total}</span>
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
                />
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

/* ───────────────────────── 指标库（平台概览） ───────────────────────── */

const SOURCE_STATUS: Record<string, string> = {
  builtin: 'online',
  custom: 'warning',
  unknown: 'unknown',
}

function LibraryView() {
  const { data, isLoading, isError } = useIndicatorSummary()
  const items = data?.items ?? []

  const stats = useMemo(() => {
    const indicators = items.reduce((s, p) => s + (p.indicators ?? 0), 0)
    const byTech = items.reduce<Record<string, number>>((acc, p) => {
      acc[p.tech] = (acc[p.tech] ?? 0) + (p.indicators ?? 0)
      return acc
    }, {})
    return { platforms: items.length, indicators, byTech }
  }, [items])

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-3 md:grid-cols-5">
        <MetricStat label="平台 · PLATFORMS" value={String(stats.platforms)} color="#00f0ff" icon={<Layers className="size-4" />} />
        <MetricStat label="指标总数 · TOTAL" value={stats.indicators.toLocaleString()} color="#00ff88" icon={<Database className="size-4" />} />
        <MetricStat label="LTE · ENB" value={String(stats.byTech['enb'] ?? 0)} color="#5b9eff" icon={<Activity className="size-4" />} />
        <MetricStat label="NR · GNB" value={String(stats.byTech['gnb'] ?? 0)} color="#a855f7" icon={<Activity className="size-4" />} />
        <MetricStat label="GSM" value={String(stats.byTech['gsm'] ?? 0)} color="#ffaa00" icon={<Activity className="size-4" />} />
      </div>

      <GlassPanel title="INDICATOR LIBRARY · 指标库平台" meta={`${items.length} PLATFORMS`}>
        {isLoading ? (
          <Loading text="LOADING LIBRARY…" />
        ) : isError ? (
          <div className="p-4">
            <ErrorBox text="LIBRARY LOAD FAILED" />
          </div>
        ) : items.length === 0 ? (
          <EmptyBox text="NO PLATFORM REGISTERED" />
        ) : (
          <div className="overflow-auto">
            <div className="grid grid-cols-[80px_1fr_110px_110px_1fr] gap-2 border-b border-cyan-500/15 px-3.5 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
              <span>TECH</span>
              <span>平台 · PLATFORM</span>
              <span className="text-right">指标数</span>
              <span>来源</span>
              <span>加载源 · LOADED FROM</span>
            </div>
            {items.map((p) => (
              <div
                key={`${p.tech}-${p.platform}`}
                className="grid grid-cols-[80px_1fr_110px_110px_1fr] items-center gap-2 border-b border-cyan-500/8 px-3.5 py-2.5 hover:bg-cyan-500/5"
              >
                <span className="chip w-fit text-cyan-300/80">{p.tech.toUpperCase()}</span>
                <span className="truncate font-display text-sm font-bold text-cyan-100">{p.platform}</span>
                <span className="text-right font-display text-sm font-bold text-cyan-200">
                  {(p.indicators ?? 0).toLocaleString()}
                </span>
                <StatusBadge
                  status={SOURCE_STATUS[p.source] ?? 'unknown'}
                  label={p.source === 'builtin' ? '内置' : p.source === 'custom' ? '自定义' : '未知'}
                  className="w-fit"
                />
                <span className="truncate font-mono text-[10px] text-cyan-300/45" title={p.loadedFrom}>
                  {p.loadedFrom || '—'}
                </span>
              </div>
            ))}
          </div>
        )}
      </GlassPanel>
    </div>
  )
}

/* ───────────────────────── 复用小件 ───────────────────────── */

function MetricStat({
  label,
  value,
  color,
  icon,
}: {
  label: string
  value: string
  color: string
  icon: React.ReactNode
}) {
  return (
    <div className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3" style={{ borderLeftColor: color }}>
      <div className="scanline" />
      <div className="relative flex items-center justify-between">
        <div>
          <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/60">{label}</div>
          <div className="font-display text-2xl font-bold leading-tight" style={{ color, textShadow: `0 0 8px ${color}` }}>
            {value}
          </div>
        </div>
        <span style={{ color }}>{icon}</span>
      </div>
    </div>
  )
}

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
        small ? 'px-3 py-4 text-xs' : 'px-4 py-6 text-sm'
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
        small ? 'py-6 text-[10px]' : 'py-12 text-xs'
      )}
    >
      {text}
    </div>
  )
}

/* 光谱仪表的占位值/趋势（指标库无实时值时的可视化预览，确定性种子，非随机渲染）。 */
function pseudoVal(seed: string, max: number) {
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) >>> 0
  return (h % (max * 100)) / 100
}

function pseudoTrend(seed: string, n: number) {
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) >>> 0
  const arr: number[] = []
  for (let i = 0; i < n; i++) {
    h = (h * 1664525 + 1013904223) >>> 0
    arr.push((h % 100) / 100)
  }
  return arr
}
