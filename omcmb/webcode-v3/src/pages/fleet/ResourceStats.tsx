import { useMemo } from 'react'
import { RefreshCcw, MapPin, Server } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { useDashboardData, useRegionStats } from '@core/hooks/api/useDashboard'
import { StateGate, StatCard } from './_shared'

export default function FleetResourceStats() {
  const { data, isLoading, isError, error, isFetching, refetch } = useDashboardData()
  const { data: regions, isLoading: regionLoading } = useRegionStats()

  const counts = data?.summary?.deviceCounts
  const total = counts?.total ?? 0
  const online = counts?.online ?? 0
  const offline = counts?.offline ?? 0
  const alarm = counts?.alarm ?? 0
  const onlinePct = total > 0 ? (online / total) * 100 : 0

  const maxRegionTotal = useMemo(
    () => (regions ?? []).reduce((m, r) => Math.max(m, r.total), 0) || 1,
    [regions]
  )

  return (
    <PageShell
      code="F06"
      title="RESOURCE STATS · 资源统计"
      subtitle="FLEET INVENTORY ANALYTICS · LIVE"
      isFetching={isFetching}
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      <StateGate
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={!counts}
        loadingLabel="AGGREGATING…"
        emptyLabel="NO STATISTICS"
      >
        <div className="space-y-4">
          {/* 总览 */}
          <div className="grid grid-cols-1 gap-3 md:grid-cols-12">
            <GlassPanel strong className="md:col-span-4">
              <div className="flex items-center justify-center gap-4 p-4">
                <RadialGauge value={onlinePct} label="ONLINE" size={128} color="#00ff88" />
                <div className="space-y-1">
                  <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                    AVAILABILITY
                  </div>
                  <div className="font-display text-3xl font-bold text-emerald-200 text-glow">
                    {onlinePct.toFixed(1)}%
                  </div>
                </div>
              </div>
            </GlassPanel>
            <div className="grid grid-cols-2 gap-3 md:col-span-8 md:grid-cols-4">
              <StatCard label="TOTAL" value={total} color="#00f0ff" />
              <StatCard label="ONLINE" value={online} color="#00ff88" />
              <StatCard label="OFFLINE" value={offline} color="#525a78" />
              <StatCard label="ALARMED" value={alarm} color="#ff2d6f" />
            </div>
          </div>

          {/* 区域分布 */}
          <GlassPanel
            title={
              <span className="flex items-center gap-2">
                <MapPin className="size-3.5" /> REGION DISTRIBUTION · 区域分布
              </span>
            }
            meta="LIVE"
          >
            {regionLoading ? (
              <div className="px-4 py-8 text-center font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/45">
                LOADING…
              </div>
            ) : !regions || regions.length === 0 ? (
              <div className="px-4 py-8 text-center font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/45">
                NO REGION DATA
              </div>
            ) : (
              <div className="space-y-2.5 p-4">
                {regions.map((r) => {
                  const pct = (r.total / maxRegionTotal) * 100
                  const onPct = r.total > 0 ? (r.online / r.total) * 100 : 0
                  return (
                    <div key={r.region} className="flex items-center gap-3">
                      <span className="flex w-28 shrink-0 items-center gap-1.5 truncate font-mono text-[11px] text-cyan-100/85">
                        <Server className="size-3 text-cyan-400/60" />
                        {r.region || '未知'}
                      </span>
                      <div className="relative h-4 flex-1 overflow-hidden rounded-sm bg-cyan-500/8">
                        <div
                          className="absolute inset-y-0 left-0 rounded-sm"
                          style={{
                            width: `${pct}%`,
                            background:
                              'linear-gradient(90deg, rgba(0,240,255,0.35), rgba(0,255,136,0.45))',
                          }}
                        />
                      </div>
                      <span className="w-16 shrink-0 text-right font-display text-sm font-bold text-cyan-100">
                        {r.total}
                      </span>
                      <span
                        className="w-12 shrink-0 text-right font-mono text-[10px]"
                        style={{ color: onPct > 80 ? '#00ff88' : '#ffaa00' }}
                      >
                        {onPct.toFixed(0)}%
                      </span>
                    </div>
                  )
                })}
              </div>
            )}
          </GlassPanel>
        </div>
      </StateGate>
    </PageShell>
  )
}
