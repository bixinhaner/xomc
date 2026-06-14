import { useEffect, useMemo, useState } from 'react'
import { Activity, Loader2, RefreshCcw, Search } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { Sparkline } from '@/components/viz/Sparkline'
import { cn } from '@/lib/utils'
import { useKPIList, useMultipleKPISeries } from '@core/hooks/api/usePerformance'
import { useDeviceList } from '@core/hooks/api/useDevices'
import type { Device } from '@core/types/device'
import type { KPISeries } from '@core/types/performance'

/**
 * F03 · 性能图表（performance/charts）
 * 选设备（真实 useDeviceList）+ 多 KPI（真实 useKPIList 定义）→ 真实多序列
 * useMultipleKPISeries，按 KPI 一卡渲染火花线趋势 + 当前/均/峰/谷。
 */

const COLORS = ['#00f0ff', '#00ff88', '#ffaa00', '#a855f7', '#ff00aa', '#5b9eff', '#ff7a1a', '#ffd400']

const MAX_KPIS = 8

// 目录就绪后默认勾选的 KPI 条数（默认即出真实图，不空等用户手选）。
const DEFAULT_KPI_COUNT = 4

function fmt(v: number | null): string {
  if (v === null) return '—'
  if (Math.abs(v) >= 1000) return v.toLocaleString(undefined, { maximumFractionDigits: 0 })
  return Number.isInteger(v) ? String(v) : v.toFixed(2)
}

export default function PerformanceCharts() {
  const [deviceSn, setDeviceSn] = useState('')
  const [deviceKeyword, setDeviceKeyword] = useState('')
  const [kpiKeyword, setKpiKeyword] = useState('')
  const [selectedKpis, setSelectedKpis] = useState<string[]>([])

  // 设备候选（真实 /devices）。
  const deviceParams = useMemo(
    () => ({ page: 1, pageSize: 30, ...(deviceKeyword.trim() ? { searchText: deviceKeyword.trim() } : {}) }),
    [deviceKeyword],
  )
  const { data: deviceData, isLoading: devLoading, isError: devError } = useDeviceList(deviceParams)
  const devices: Device[] = deviceData?.items ?? []

  // KPI 定义候选（真实 /pm/kpi/definitions）。
  const { data: kpiData, isLoading: kpiLoading, isError: kpiError } = useKPIList({
    page: 1,
    pageSize: 100,
    ...(kpiKeyword.trim() ? { keyword: kpiKeyword.trim() } : {}),
  })
  const kpis = kpiData?.items ?? []
  const kpiUnit = useMemo(() => {
    const m: Record<string, string> = {}
    kpis.forEach((k) => {
      m[k.kpiCode] = k.unit
    })
    return m
  }, [kpis])
  const kpiName = useMemo(() => {
    const m: Record<string, string> = {}
    kpis.forEach((k) => {
      m[k.kpiCode] = k.kpiName
    })
    return m
  }, [kpis])

  // 默认选首个设备（取数 series 端点本身按 kpiCode 取全网/设备序列）。
  useEffect(() => {
    if (!deviceSn && devices.length > 0) setDeviceSn(devices[0].sn)
  }, [devices, deviceSn])

  // KPI 目录就绪后默认勾选前 N 条真实 KPI（仅在无搜索词时，避免搜索结果污染默认）。
  useEffect(() => {
    if (kpiKeyword.trim() || kpis.length === 0) return
    setSelectedKpis((prev) =>
      prev.length > 0 ? prev : kpis.slice(0, DEFAULT_KPI_COUNT).map((k) => k.kpiCode),
    )
  }, [kpis, kpiKeyword])

  // 真实多 KPI 序列（enabled 仅当 kpiCodes 非空）。
  const { data: seriesData, isLoading: seriesLoading, isFetching, isError: seriesError, refetch } =
    useMultipleKPISeries(selectedKpis, deviceSn || undefined)

  // KPISeries.kpiName 可能是后端编码或名称；按选中顺序对位回退取序列。
  const seriesByCode = useMemo(() => {
    const map = new Map<string, KPISeries>()
    ;(seriesData ?? []).forEach((s) => {
      map.set(s.kpiName, s)
    })
    return map
  }, [seriesData])

  const toggleKpi = (code: string) => {
    setSelectedKpis((prev) => {
      if (prev.includes(code)) return prev.filter((c) => c !== code)
      if (prev.length >= MAX_KPIS) return prev
      return [...prev, code]
    })
  }

  const cards = useMemo(() => {
    return selectedKpis.map((code, idx) => {
      // 真实 series 可能以 kpiName=code 或 kpiName=显示名命中，两路都尝试。
      const s = seriesByCode.get(code) ?? seriesByCode.get(kpiName[code] ?? '')
      const points = (s?.data ?? []).map((d) => d.value)
      const xs = (s?.data ?? []).map((d) => d.timestamp)
      const latest = points.length ? points[points.length - 1] : null
      const avg = points.length ? points.reduce((a, b) => a + b, 0) / points.length : null
      const min = points.length ? Math.min(...points) : null
      const max = points.length ? Math.max(...points) : null
      return {
        code,
        name: kpiName[code] ?? s?.kpiName ?? code,
        unit: s?.unit ?? kpiUnit[code] ?? '',
        color: COLORS[idx % COLORS.length],
        points,
        xs,
        latest,
        avg,
        min,
        max,
      }
    })
  }, [selectedKpis, seriesByCode, kpiName, kpiUnit])

  return (
    <PageShell
      code="F03"
      title="PERFORMANCE CHARTS · 性能图表"
      subtitle="MULTI-KPI TIME SERIES · NEON TREND"
      isFetching={isFetching}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()} disabled={selectedKpis.length === 0}>
          REFRESH
        </NeonButton>
      }
    >
      <div className="grid grid-cols-12 gap-3">
        {/* 左：设备 + KPI 选择 */}
        <div className="col-span-12 space-y-3 lg:col-span-4 xl:col-span-3">
          <GlassPanel title="设备 · DEVICE" meta={deviceSn || '未选'}>
            <div className="p-3">
              <div className="relative mb-2">
                <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
                <input
                  className="neon-input w-full pl-9"
                  placeholder="SN / 名称 搜索"
                  value={deviceKeyword}
                  onChange={(e) => setDeviceKeyword(e.target.value)}
                />
              </div>
              <div className="max-h-48 overflow-auto rounded-sm border border-cyan-500/12">
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
                        <div className="truncate text-[10px] text-cyan-300/55">{d.deviceName || d.name || '—'}</div>
                      </div>
                    </button>
                  ))
                )}
              </div>
            </div>
          </GlassPanel>

          <GlassPanel title="KPI 选择 · METRIC" meta={`${selectedKpis.length}/${MAX_KPIS}`}>
            <div className="p-3">
              <div className="relative mb-2">
                <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
                <input
                  className="neon-input w-full pl-9"
                  placeholder="KPI 名 / 编码搜索"
                  value={kpiKeyword}
                  onChange={(e) => setKpiKeyword(e.target.value)}
                />
              </div>
              <div className="max-h-72 overflow-auto rounded-sm border border-cyan-500/12">
                {kpiLoading ? (
                  <Loading text="LOADING KPI DEFS…" small />
                ) : kpiError ? (
                  <ErrorBox text="KPI DEFS FAILED" small />
                ) : kpis.length === 0 ? (
                  <EmptyBox text="NO KPI" small />
                ) : (
                  kpis.map((k) => {
                    const checked = selectedKpis.includes(k.kpiCode)
                    const disabled = !checked && selectedKpis.length >= MAX_KPIS
                    return (
                      <button
                        key={k.id}
                        type="button"
                        disabled={disabled}
                        onClick={() => toggleKpi(k.kpiCode)}
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
                          <div className="truncate text-[11px] text-cyan-100">{k.kpiName}</div>
                          <div className="truncate font-mono text-[9px] text-cyan-300/45">
                            {k.kpiCode}
                            {k.unit ? ` · ${k.unit}` : ''}
                          </div>
                        </div>
                      </button>
                    )
                  })
                )}
              </div>
            </div>
          </GlassPanel>
        </div>

        {/* 右：图表 */}
        <div className="col-span-12 lg:col-span-8 xl:col-span-9">
          {selectedKpis.length === 0 ? (
            <GlassPanel title="CHARTS · 图表">
              <div className="flex flex-col items-center justify-center gap-3 py-20">
                <Activity className="size-10 text-cyan-300/40" />
                <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/45">
                  从左侧勾选 KPI 渲染趋势
                </div>
              </div>
            </GlassPanel>
          ) : seriesLoading ? (
            <GlassPanel title="CHARTS · 图表" meta="SYNC">
              <Loading text="QUERYING SERIES…" />
            </GlassPanel>
          ) : seriesError ? (
            <GlassPanel title="CHARTS · 图表">
              <div className="p-4">
                <ErrorBox text="SERIES QUERY FAILED" />
              </div>
            </GlassPanel>
          ) : (
            <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
              {cards.map((c) => (
                <GlassPanel key={c.code} title={c.name} meta={`${c.points.length} PTS${c.unit ? ` · ${c.unit}` : ''}`}>
                  <div className="p-4">
                    {c.points.length === 0 ? (
                      <div className="flex items-center justify-center py-8 font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/40">
                        NO SAMPLE
                      </div>
                    ) : (
                      <>
                        <div className="mb-3 flex items-end justify-between">
                          <div>
                            <div
                              className="font-display text-3xl font-bold leading-none"
                              style={{ color: c.color, textShadow: `0 0 8px ${c.color}` }}
                            >
                              {fmt(c.latest)}
                            </div>
                            <div className="mt-1 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/50">
                              LATEST {c.unit}
                            </div>
                          </div>
                          <Sparkline data={c.points} color={c.color} width={200} height={48} />
                        </div>
                        <div className="grid grid-cols-3 gap-2 border-t border-cyan-500/12 pt-2.5">
                          <MiniStat label="AVG" value={fmt(c.avg)} />
                          <MiniStat label="MIN" value={fmt(c.min)} />
                          <MiniStat label="MAX" value={fmt(c.max)} />
                        </div>
                        <div className="mt-2 truncate font-mono text-[9px] text-cyan-300/35">{c.code}</div>
                      </>
                    )}
                  </div>
                </GlassPanel>
              ))}
            </div>
          )}
        </div>
      </div>
    </PageShell>
  )
}

/* ───────── 复用小件 ───────── */

function MiniStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="text-center">
      <div className="font-display text-sm font-bold text-cyan-100">{value}</div>
      <div className="font-mono text-[9px] uppercase tracking-[0.2em] text-cyan-300/45">{label}</div>
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
