import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  RefreshCcw,
  Loader2,
  LineChart as LineChartIcon,
  Activity,
  AlertTriangle,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { Sparkline } from '@/components/viz/Sparkline'
import { cn } from '@/lib/utils'
import { useAllKPIs, useMultipleKPISeries } from '@core/hooks/api/usePerformance'
import type { KPI, KPISeries } from '@core/types/performance'

// ---------------------------------------------------------------------------
// /reports/historical-kpi · 历史 KPI 报表
//   对齐 v1 webcode/report/HistoricalKPI：选 KPI → 看历史时序 + 明细。
//   v1 旧页面图表走本地生成器；本皮肤改用真实 @core：
//     - useAllKPIs（/pm/kpi/definitions）取 KPI 目录
//     - useMultipleKPISeries（/pm/kpi 按 kpi_name 查询）取真实时序
//   覆盖加载/失败/空三态；选中 KPI 渲染霓虹多序列趋势 + 明细表。
// ---------------------------------------------------------------------------

const COLORS = ['#00f0ff', '#00ff88', '#ffaa00', '#a855f7', '#ff00aa', '#5b9eff', '#ff7a1a', '#ffd400']
const MAX_KPIS = 8

export default function HistoricalKpiReportPage() {
  const navigate = useNavigate()
  const [keyword, setKeyword] = useState('')
  const [selected, setSelected] = useState<string[]>([])

  const { data: kpis, isLoading: kpiLoading, isError: kpiError, error: kpiErrObj, isFetching, refetch } = useAllKPIs()

  const catalog: KPI[] = useMemo(() => {
    const list = kpis ?? []
    const kw = keyword.trim().toLowerCase()
    if (!kw) return list
    return list.filter(
      (k) =>
        k.kpiName.toLowerCase().includes(kw) ||
        k.kpiCode.toLowerCase().includes(kw) ||
        (k.category ?? '').toLowerCase().includes(kw),
    )
  }, [kpis, keyword])

  // KPI 目录就绪后默认选前两条（默认即出真实图，不空等）。
  useEffect(() => {
    if (!kpis || kpis.length === 0) return
    setSelected((prev) => (prev.length > 0 ? prev : kpis.slice(0, 2).map((k) => k.kpiCode)))
  }, [kpis])

  const {
    data: series,
    isLoading: seriesLoading,
    isFetching: seriesFetching,
    isError: seriesError,
    refetch: refetchSeries,
  } = useMultipleKPISeries(selected)

  const codeToName = useMemo(() => {
    const m: Record<string, string> = {}
    ;(kpis ?? []).forEach((k) => {
      m[k.kpiCode] = k.kpiName
    })
    return m
  }, [kpis])

  // 把多条 KPISeries 对齐到统一时间轴，构造明细表行。
  const detail = useMemo(() => {
    const list = series ?? []
    const tsSet = new Set<string>()
    list.forEach((s) => s.data.forEach((p) => tsSet.add(p.timestamp)))
    const timestamps = Array.from(tsSet).sort()
    const byCodeTs = new Map<string, Map<string, number>>()
    list.forEach((s, i) => {
      const code = selected[i]
      const m = new Map<string, number>()
      s.data.forEach((p) => m.set(p.timestamp, p.value))
      byCodeTs.set(code, m)
    })
    const rows = timestamps.slice(-60).map((ts) => ({
      ts,
      values: selected.map((code) => byCodeTs.get(code)?.get(ts) ?? null),
    }))
    return { timestamps, rows }
  }, [series, selected])

  const toggle = (code: string) => {
    setSelected((prev) => {
      if (prev.includes(code)) return prev.filter((c) => c !== code)
      if (prev.length >= MAX_KPIS) return prev
      return [...prev, code]
    })
  }

  const totalPoints = useMemo(() => (series ?? []).reduce((s, x) => s + x.data.length, 0), [series])

  return (
    <PageShell
      code="F06"
      title="HISTORICAL KPI · 历史 KPI 报表"
      subtitle="NETWORK KPI TIME-SERIES · MULTI-SERIES TREND"
      isFetching={isFetching || seriesFetching}
      bare
      toolbar={
        <>
          <button type="button" onClick={() => navigate('/reports')} className="chip text-cyan-300/70 hover:opacity-100">
            ← 简报中心
          </button>
          <NeonButton
            icon={<RefreshCcw />}
            onClick={() => {
              void refetch()
              if (selected.length > 0) void refetchSeries()
            }}
          >
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="grid grid-cols-12 gap-3">
        {/* 左：KPI 目录选择（真实 /pm/kpi/definitions） */}
        <div className="col-span-12 lg:col-span-4 xl:col-span-3">
          <GlassPanel title="KPI CATALOG · 指标目录" meta={`${selected.length}/${MAX_KPIS}`}>
            <div className="border-b border-cyan-500/12 p-2.5">
              <div className="relative">
                <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
                <input
                  className="neon-input w-full pl-9"
                  placeholder="KPI 名称 / 编码 / 分类"
                  value={keyword}
                  onChange={(e) => setKeyword(e.target.value)}
                />
              </div>
            </div>
            <div className="max-h-[calc(100vh-340px)] min-h-[300px] overflow-auto p-2">
              {kpiLoading ? (
                <Centered small>
                  <Loader2 className="size-4 animate-spin" />
                  <span className="font-mono text-[10px] uppercase tracking-[0.2em]">LOADING KPI CATALOG…</span>
                </Centered>
              ) : kpiError ? (
                <div className="m-1 border border-rose-500/40 bg-rose-500/5 px-3 py-4 font-mono text-xs text-rose-300">
                  CATALOG LOAD FAILED · {kpiErrObj instanceof Error ? kpiErrObj.message : '加载失败'}
                </div>
              ) : catalog.length === 0 ? (
                <div className="py-10 text-center font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/40">
                  无匹配 KPI
                </div>
              ) : (
                catalog.map((k) => {
                  const checked = selected.includes(k.kpiCode)
                  const disabled = !checked && selected.length >= MAX_KPIS
                  return (
                    <button
                      key={k.id}
                      type="button"
                      disabled={disabled}
                      onClick={() => toggle(k.kpiCode)}
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
                        <div className="truncate text-[11px] text-cyan-100">{k.kpiName}</div>
                        <div className="truncate font-mono text-[9px] text-cyan-300/45">
                          {k.kpiCode}
                          {k.category ? ` · ${k.category}` : ''}
                          {k.unit ? ` · ${k.unit}` : ''}
                        </div>
                      </div>
                    </button>
                  )
                })
              )}
            </div>
          </GlassPanel>
        </div>

        {/* 右：趋势 + 明细 */}
        <div className="col-span-12 space-y-3 lg:col-span-8 xl:col-span-9">
          <GlassPanel
            title="KPI TREND · 历史趋势"
            meta={selected.length === 0 ? '未选 KPI' : `${selected.length} KPI · ${totalPoints} PTS`}
          >
            <div className="p-3">
              {selected.length === 0 ? (
                <Empty icon={<LineChartIcon className="size-10 text-cyan-300/40" />} text="左侧勾选 KPI 查看历史趋势" />
              ) : seriesLoading || seriesFetching ? (
                <Centered>
                  <Loader2 className="size-4 animate-spin" />
                  <span className="font-mono text-xs uppercase tracking-[0.2em]">QUERYING TIME-SERIES…</span>
                </Centered>
              ) : seriesError ? (
                <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
                  KPI SERIES UNAVAILABLE
                </div>
              ) : totalPoints === 0 ? (
                <Empty icon={<Activity className="size-10 text-cyan-300/40" />} text="所选 KPI 在采集窗口内暂无样本" />
              ) : (
                <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
                  {(series ?? []).map((s, i) => (
                    <SeriesCard
                      key={selected[i] ?? s.kpiName}
                      code={selected[i] ?? s.kpiName}
                      name={codeToName[selected[i]] ?? s.kpiName}
                      series={s}
                      color={COLORS[i % COLORS.length]}
                    />
                  ))}
                </div>
              )}
            </div>
          </GlassPanel>

          {/* 明细表（对齐时间轴） */}
          {selected.length > 0 && totalPoints > 0 ? (
            <GlassPanel title="KPI DETAIL · 明细数据" meta={`${detail.rows.length} 行（近 60 桶）`}>
              <div className="max-h-[40vh] overflow-auto">
                <div
                  className="grid gap-2 border-b border-cyan-500/15 px-3.5 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55"
                  style={{ gridTemplateColumns: `180px repeat(${selected.length}, minmax(90px, 1fr))` }}
                >
                  <span>时间 · TIME</span>
                  {selected.map((code, i) => (
                    <span key={code} className="truncate text-right" style={{ color: COLORS[i % COLORS.length] }}>
                      {codeToName[code] ?? code}
                    </span>
                  ))}
                </div>
                {detail.rows.map((row) => (
                  <div
                    key={row.ts}
                    className="grid items-center gap-2 border-b border-cyan-500/8 px-3.5 py-1.5 hover:bg-cyan-500/5"
                    style={{ gridTemplateColumns: `180px repeat(${selected.length}, minmax(90px, 1fr))` }}
                  >
                    <span className="truncate font-mono text-[10px] text-cyan-300/70">{row.ts}</span>
                    {row.values.map((v, i) => (
                      <span key={i} className="text-right font-mono text-[11px] text-cyan-100">
                        {v === null ? <span className="text-cyan-300/30">—</span> : v.toFixed(2)}
                      </span>
                    ))}
                  </div>
                ))}
              </div>
            </GlassPanel>
          ) : null}
        </div>
      </div>
    </PageShell>
  )
}

function SeriesCard({
  code,
  name,
  series,
  color,
}: {
  code: string
  name: string
  series: KPISeries
  color: string
}) {
  const { vals, latest, avg, min, max } = useMemo(() => {
    const v = series.data.map((p) => p.value)
    if (v.length === 0) return { vals: [] as number[], latest: null, avg: null, min: null, max: null }
    const sum = v.reduce((a, b) => a + b, 0)
    return { vals: v, latest: v[v.length - 1], avg: sum / v.length, min: Math.min(...v), max: Math.max(...v) }
  }, [series])

  const fmt = (n: number | null) => (n === null ? '—' : Number.isInteger(n) ? String(n) : n.toFixed(2))

  return (
    <GlassPanel title={name} meta={`${series.data.length} PTS`}>
      <div className="p-4">
        {vals.length === 0 ? (
          <div className="flex items-center justify-center py-6 font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/40">
            无样本
          </div>
        ) : (
          <>
            <div className="mb-3 flex items-end justify-between">
              <div>
                <div className="font-display text-3xl font-bold leading-none" style={{ color, textShadow: `0 0 8px ${color}` }}>
                  {fmt(latest)}
                  {series.unit ? <span className="ml-1 text-sm text-cyan-300/55">{series.unit}</span> : null}
                </div>
                <div className="mt-1 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/50">LATEST</div>
              </div>
              <Sparkline data={vals} color={color} width={180} height={48} />
            </div>
            <div className="grid grid-cols-3 gap-2 border-t border-cyan-500/12 pt-2.5">
              <MiniStat label="AVG" value={fmt(avg)} />
              <MiniStat label="MIN" value={fmt(min)} />
              <MiniStat label="MAX" value={fmt(max)} />
            </div>
            <div className="mt-2 truncate font-mono text-[9px] text-cyan-300/35">{code}</div>
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

function Centered({ children, small }: { children: React.ReactNode; small?: boolean }) {
  return <div className={cn('flex items-center justify-center gap-2 text-cyan-300/60', small ? 'py-6' : 'py-16')}>{children}</div>
}

function Empty({ icon, text }: { icon: React.ReactNode; text: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16">
      {icon ?? <AlertTriangle className="size-9 text-cyan-300/40" />}
      <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/45">{text}</div>
    </div>
  )
}
