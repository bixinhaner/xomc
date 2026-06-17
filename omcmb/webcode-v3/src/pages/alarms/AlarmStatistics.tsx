import { useCallback, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Activity,
  AlertTriangle,
  ChevronRight,
  Gauge,
  Loader2,
  RefreshCcw,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { Sparkline } from '@/components/viz/Sparkline'
import { useAlarmCount } from '@core/hooks/api/useAlarms'
import { useAlarmTrend, useTopAlarmDevices } from '@core/hooks/api/useDashboard'
import type { AlarmSeverity } from '@core/types/common'
import { neTypeLabel } from './AlarmDetailPanel'

const SEV_LABEL: Record<AlarmSeverity, string> = {
  critical: '紧急',
  major: '重要',
  minor: '次要',
  warning: '警告',
}
const SEV_COLOR: Record<AlarmSeverity, string> = {
  critical: '#ff2d6f',
  major: '#ff7a1a',
  minor: '#ffd400',
  warning: '#5b9eff',
}
const SEV_KEYS: AlarmSeverity[] = ['critical', 'major', 'minor', 'warning']

const RANGE_OPTIONS: { v: number; t: string }[] = [
  { v: 7, t: '近 7 天' },
  { v: 30, t: '近 30 天' },
  { v: 90, t: '近 90 天' },
]

type TrendPoint = {
  date: string
  critical: number
  major: number
  minor: number
  warning: number
}

function formatLocalDate(date: Date) {
  const year = date.getFullYear()
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  return `${year}-${month}-${day}`
}

function buildTrendSeries(raw: TrendPoint[] | undefined, days: number): TrendPoint[] {
  const byDate = new Map((raw ?? []).map((item) => [item.date, item]))
  return Array.from({ length: days }, (_, index) => {
    const date = new Date()
    date.setDate(date.getDate() - (days - index - 1))
    const key = formatLocalDate(date)
    const item = byDate.get(key)
    return {
      date: key,
      critical: item?.critical ?? 0,
      major: item?.major ?? 0,
      minor: item?.minor ?? 0,
      warning: item?.warning ?? 0,
    }
  })
}

export default function AlarmStatistics() {
  const navigate = useNavigate()
  const [days, setDays] = useState(7)

  const {
    data: cnt,
    isLoading: cntLoading,
    isError: cntError,
    isFetching: cntFetching,
    refetch: refetchCount,
  } = useAlarmCount()
  const {
    data: trend,
    isLoading: trendLoading,
    isError: trendError,
    isFetching: trendFetching,
    refetch: refetchTrend,
  } = useAlarmTrend(days)
  const {
    data: topDevices,
    isLoading: topLoading,
    isError: topError,
    isFetching: topFetching,
    refetch: refetchTop,
  } = useTopAlarmDevices()

  const isFetching = cntFetching || trendFetching || topFetching

  const refreshAll = useCallback(() => {
    void refetchCount()
    void refetchTrend()
    void refetchTop()
  }, [refetchCount, refetchTrend, refetchTop])

  // 严重度分布（饼/环替代 → 用 RadialGauge 表示占比）
  const sevTotals = useMemo(() => {
    const c = cnt?.critical ?? 0
    const ma = cnt?.major ?? 0
    const mi = cnt?.minor ?? 0
    const w = cnt?.warning ?? 0
    const total = c + ma + mi + w
    return { critical: c, major: ma, minor: mi, warning: w, total }
  }, [cnt])

  // 健康度：以 warning 占比近似（critical 越多越红）
  const healthValue = useMemo(() => {
    if (sevTotals.total === 0) return 100
    const weighted =
      sevTotals.critical * 4 +
      sevTotals.major * 3 +
      sevTotals.minor * 2 +
      sevTotals.warning * 1
    const worst = sevTotals.total * 4
    return Math.max(0, Math.round(100 - (weighted / worst) * 100))
  }, [sevTotals])

  const trendSeries = useMemo(() => buildTrendSeries(trend, days), [trend, days])

  const trendSums = useMemo(() => {
    const base: Record<AlarmSeverity, number> = { critical: 0, major: 0, minor: 0, warning: 0 }
    for (const d of trendSeries) {
      base.critical += d.critical
      base.major += d.major
      base.minor += d.minor
      base.warning += d.warning
    }
    return base
  }, [trendSeries])

  const goDrill = useCallback(
    (severity: AlarmSeverity) => {
      navigate(`/alarm/current?severity=${severity}`)
    },
    [navigate]
  )

  const sortedTop = useMemo(() => {
    if (!topDevices) return []
    return [...topDevices].sort((a, b) => b.alarmCount - a.alarmCount).slice(0, 10)
  }, [topDevices])
  const topMax = sortedTop[0]?.alarmCount ?? 1

  return (
    <PageShell
      code="F04"
      title="ALARM STATISTICS · 告警统计"
      subtitle="SEVERITY DISTRIBUTION · TREND · TOP OFFENDERS"
      isFetching={isFetching}
      toolbar={
        <>
          <NeonButton icon={<RefreshCcw />} onClick={refreshAll}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="grid grid-cols-12 gap-3">
        {/* 总量计数条 */}
        <div className="col-span-12 grid grid-cols-2 gap-3 lg:grid-cols-5">
          <StatTile label="TOTAL ACTIVE" color="#00f0ff" value={cnt?.total_active ?? 0} />
          {SEV_KEYS.map((s) => (
            <StatTile
              key={s}
              label={`${s.toUpperCase()} · ${SEV_LABEL[s]}`}
              color={SEV_COLOR[s]}
              value={sevTotals[s]}
              onClick={() => goDrill(s)}
            />
          ))}
        </div>

        {/* 健康度 + 严重度分布 */}
        <GlassPanel
          title="NETWORK HEALTH · 网络健康度"
          meta="LIVE"
          className="col-span-12 lg:col-span-4"
        >
          <div className="flex flex-col items-center gap-3 p-4">
            {cntLoading ? (
              <LoadingBox />
            ) : cntError ? (
              <ErrorBox label="计数加载失败" onRetry={() => void refetchCount()} />
            ) : (
              <>
                <RadialGauge
                  value={healthValue}
                  label="HEALTH"
                  unit="%"
                  size={150}
                  color={
                    healthValue > 80
                      ? '#00ff88'
                      : healthValue > 50
                        ? '#ffaa00'
                        : '#ff2d6f'
                  }
                />
                <div className="grid w-full grid-cols-2 gap-2">
                  <MiniStat label="未确认" color="#00f0ff" value={cnt?.unacknowledged ?? 0} />
                  <MiniStat label="未读" color="#a855f7" value={cnt?.unread ?? 0} />
                </div>
              </>
            )}
          </div>
        </GlassPanel>

        {/* 严重度分布占比 */}
        <GlassPanel
          title="SEVERITY MIX · 等级分布"
          meta={`TOTAL ${sevTotals.total}`}
          className="col-span-12 lg:col-span-8"
        >
          {cntLoading ? (
            <LoadingBox />
          ) : cntError ? (
            <ErrorBox label="计数加载失败" onRetry={() => void refetchCount()} />
          ) : sevTotals.total === 0 ? (
            <EmptyBox label="ALL CLEAR · 暂无活动告警" tone="ok" />
          ) : (
            <div className="space-y-2.5 p-4">
              {SEV_KEYS.map((s) => {
                const v = sevTotals[s]
                const pct = sevTotals.total > 0 ? (v / sevTotals.total) * 100 : 0
                return (
                  <button
                    key={s}
                    type="button"
                    onClick={() => goDrill(s)}
                    className="group w-full text-left"
                  >
                    <div className="mb-1 flex items-center justify-between font-mono text-[11px]">
                      <span
                        className="font-bold tracking-[0.14em]"
                        style={{ color: SEV_COLOR[s], textShadow: `0 0 5px ${SEV_COLOR[s]}` }}
                      >
                        {SEV_LABEL[s]} · {s.toUpperCase()}
                      </span>
                      <span className="text-cyan-300/70">
                        {v} · {pct.toFixed(1)}%
                      </span>
                    </div>
                    <div className="h-2 overflow-hidden rounded-sm bg-cyan-500/8">
                      <div
                        className="h-full rounded-sm transition-all group-hover:brightness-125"
                        style={{
                          width: `${pct}%`,
                          background: SEV_COLOR[s],
                          boxShadow: `0 0 8px ${SEV_COLOR[s]}`,
                        }}
                      />
                    </div>
                  </button>
                )
              })}
            </div>
          )}
        </GlassPanel>

        {/* 趋势 */}
        <GlassPanel
          title={`${days}-DAY TREND · 告警趋势`}
          meta="BY SEVERITY"
          className="col-span-12 lg:col-span-7"
        >
          <div className="space-y-3 p-4">
            <TrendRangeSelector days={days} onChange={setDays} />
            {trendLoading ? (
              <LoadingBox />
            ) : trendError ? (
              <ErrorBox label="趋势加载失败" onRetry={() => void refetchTrend()} />
            ) : trendSeries.every((d) => d.critical + d.major + d.minor + d.warning === 0) ? (
              <EmptyBox label="NO TREND DATA · 暂无趋势数据" />
            ) : (
              <div className="space-y-2">
              {SEV_KEYS.map((s) => {
                const points = trendSeries.map((d) => d[s])
                return (
                  <div key={s} className="flex items-center gap-3">
                    <span
                      className="w-10 shrink-0 font-mono text-[10px] font-bold tracking-[0.12em]"
                      style={{ color: SEV_COLOR[s], textShadow: `0 0 4px ${SEV_COLOR[s]}` }}
                    >
                      {s.slice(0, 4).toUpperCase()}
                    </span>
                    <div className="flex-1">
                      <Sparkline data={points} color={SEV_COLOR[s]} width={320} height={28} fill />
                    </div>
                    <span
                      className="w-12 shrink-0 text-right font-display text-sm font-bold"
                      style={{ color: SEV_COLOR[s] }}
                    >
                      {trendSums[s]}
                    </span>
                  </div>
                )
              })}
              <div className="flex justify-between pt-1 font-mono text-[10px] text-cyan-300/40">
                <span>{trendSeries[0]?.date}</span>
                <span>{trendSeries[trendSeries.length - 1]?.date}</span>
              </div>
              </div>
            )}
          </div>
        </GlassPanel>

        {/* Top 故障设备 */}
        <GlassPanel
          title="TOP OFFENDERS · 故障设备 TOP10"
          meta="LIVE"
          className="col-span-12 lg:col-span-5"
        >
          {topLoading ? (
            <LoadingBox />
          ) : topError ? (
            <ErrorBox label="设备排行加载失败" onRetry={() => void refetchTop()} />
          ) : sortedTop.length === 0 ? (
            <EmptyBox label="NO DATA · 暂无故障设备" />
          ) : (
            <div className="divide-y divide-cyan-500/8">
              {sortedTop.map((d, i) => {
                const color =
                  d.severity in SEV_COLOR ? SEV_COLOR[d.severity as AlarmSeverity] : '#5b9eff'
                const pct = (d.alarmCount / topMax) * 100
                return (
                  <button
                    key={d.deviceSN}
                    type="button"
                    onClick={() => navigate(`/alarm/current?keyword=${encodeURIComponent(d.deviceSN)}`)}
                    className="flex w-full items-center gap-2.5 px-3 py-2 text-left hover:bg-cyan-500/5"
                  >
                    <span className="w-5 shrink-0 text-center font-mono text-[11px] text-cyan-300/40">
                      {i + 1}
                    </span>
                    <div className="min-w-0 flex-1">
                      <div className="truncate font-mono text-xs text-cyan-100">
                        {d.deviceName || d.deviceSN}
                      </div>
                      <div className="mt-1 h-1.5 overflow-hidden rounded-sm bg-cyan-500/8">
                        <div
                          className="h-full rounded-sm"
                          style={{ width: `${pct}%`, background: color, boxShadow: `0 0 6px ${color}` }}
                        />
                      </div>
                      <div className="mt-0.5 truncate font-mono text-[9px] uppercase tracking-[0.12em] text-cyan-300/45">
                        {neTypeLabel(d.technology)} · {d.deviceSN}
                      </div>
                    </div>
                    <span
                      className="shrink-0 font-display text-sm font-bold"
                      style={{ color, textShadow: `0 0 6px ${color}` }}
                    >
                      {d.alarmCount}
                    </span>
                    <ChevronRight className="size-3.5 shrink-0 text-cyan-300/30" />
                  </button>
                )
              })}
            </div>
          )}
        </GlassPanel>
      </div>
    </PageShell>
  )
}

function TrendRangeSelector({ days, onChange }: { days: number; onChange: (days: number) => void }) {
  return (
    <div className="flex flex-wrap justify-end gap-2">
      {RANGE_OPTIONS.map((r) => (
        <button
          key={r.v}
          type="button"
          onClick={() => onChange(r.v)}
          className={`chip transition-all ${
            days === r.v
              ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
              : 'text-cyan-300/45 hover:text-cyan-300/80'
          }`}
        >
          {r.t}
        </button>
      ))}
    </div>
  )
}

function StatTile({
  label,
  color,
  value,
  onClick,
}: {
  label: string
  color: string
  value: number
  onClick?: () => void
}) {
  const Comp = onClick ? 'button' : 'div'
  return (
    <Comp
      type={onClick ? 'button' : undefined}
      onClick={onClick}
      className={`glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5 text-left ${
        onClick ? 'transition-all hover:brightness-110' : ''
      }`}
      style={{ borderLeftColor: color }}
    >
      <div className="font-mono text-[10px] uppercase tracking-[0.14em] text-cyan-300/65">
        {label}
      </div>
      <div
        className="font-display text-2xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
    </Comp>
  )
}

function MiniStat({ label, color, value }: { label: string; color: string; value: number }) {
  return (
    <div className="rounded-sm border border-cyan-500/15 bg-cyan-500/[0.03] px-2.5 py-1.5">
      <div className="font-mono text-[9px] uppercase tracking-[0.14em] text-cyan-300/55">
        {label}
      </div>
      <div className="font-display text-lg font-bold" style={{ color }}>
        {value}
      </div>
    </div>
  )
}

function LoadingBox() {
  return (
    <div className="flex items-center justify-center gap-2 px-4 py-10 text-cyan-300/60">
      <Loader2 className="size-4 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
    </div>
  )
}

function ErrorBox({ label, onRetry }: { label: string; onRetry: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 px-4 py-8">
      <AlertTriangle className="size-7 text-rose-400/70" />
      <div className="font-mono text-xs uppercase tracking-[0.16em] text-rose-300/80">{label}</div>
      <NeonButton tone="danger" icon={<RefreshCcw />} onClick={onRetry}>
        RETRY
      </NeonButton>
    </div>
  )
}

function EmptyBox({ label, tone = 'idle' }: { label: string; tone?: 'idle' | 'ok' }) {
  const Icon = tone === 'ok' ? Gauge : Activity
  return (
    <div className="flex flex-col items-center justify-center gap-2 px-4 py-10">
      <Icon className={tone === 'ok' ? 'size-7 text-emerald-400/60' : 'size-7 text-cyan-300/40'} />
      <span
        className={`font-mono text-xs uppercase tracking-[0.18em] ${
          tone === 'ok' ? 'text-emerald-300/70' : 'text-cyan-300/45'
        }`}
      >
        {label}
      </span>
    </div>
  )
}
