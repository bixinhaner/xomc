import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Gauge, Loader2, RefreshCcw, SignalHigh } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { Sparkline } from '@/components/viz/Sparkline'
import { useMRIndicators, useMRIndicatorStats } from '@core/hooks/api/useMR'

/**
 * MR 指标详情（/mr/indicators/:code）。
 * 指标元数据来自 useMRIndicators（在指标库里按 code 命中），统计来自 useMRIndicatorStats。
 * 两路都走 @core（mock/real 自动切换），保证 :code 可用真实数据加载。
 */
export default function IndicatorDetail() {
  const navigate = useNavigate()
  const { code = '' } = useParams<{ code: string }>()

  // 指标库较小，一次拉大页定位元数据
  const catalog = useMRIndicators({ page: 1, pageSize: 500 })
  const indicator = useMemo(
    () => (catalog.data?.items ?? []).find((i) => i.indicatorCode === code) ?? null,
    [catalog.data, code],
  )

  const stats = useMRIndicatorStats(code)

  const s = stats.data
  // 把 avg 在 [min,max] 量程里映射为 0-100 供环形仪表显示
  const gaugePct = useMemo(() => {
    if (!s) return 0
    const lo = indicator?.valueRange[0] ?? s.min
    const hi = indicator?.valueRange[1] ?? s.max
    const span = hi - lo || 1
    return Math.max(0, Math.min(100, ((s.avg - lo) / span) * 100))
  }, [s, indicator])

  // 由 min/p50/avg/p95/max 派生分布趋势点（真实统计值）
  const distro = useMemo(() => {
    if (!s) return [] as number[]
    return [s.min, s.p50, s.avg, s.p95, s.max]
  }, [s])

  const isFetching = catalog.isFetching || stats.isFetching
  const color = '#00f0ff'

  return (
    <PageShell
      code="F05"
      title={`INDICATOR · ${code}`}
      subtitle={indicator ? indicator.indicatorName : 'MEASUREMENT INDICATOR DETAIL'}
      isFetching={isFetching}
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/mr/indicators')}>
            BACK
          </NeonButton>
          <NeonButton
            icon={<RefreshCcw />}
            onClick={() => {
              void catalog.refetch()
              void stats.refetch()
            }}
          >
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="grid grid-cols-12 gap-3">
        {/* 元数据卡 */}
        <div className="col-span-12 lg:col-span-5">
          <GlassPanel title="INDICATOR META · 指标元数据">
            <div className="p-4">
              {catalog.isLoading ? (
                <Centered>
                  <Loader2 className="size-4 animate-spin" />
                  <span className="font-mono text-xs uppercase tracking-[0.2em]">LOADING META…</span>
                </Centered>
              ) : catalog.isError ? (
                <ErrLine msg={catalog.error instanceof Error ? catalog.error.message : '指标库加载失败'} />
              ) : !indicator ? (
                <div className="flex flex-col items-center gap-3 py-10">
                  <SignalHigh className="size-9 text-cyan-300/40" />
                  <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
                    指标 {code} 不在当前指标库
                  </div>
                </div>
              ) : (
                <div className="flex flex-col gap-3">
                  <div className="flex items-baseline justify-between gap-2">
                    <span className="font-mono text-2xl font-bold text-glow" style={{ color }}>
                      {indicator.indicatorCode}
                    </span>
                    {indicator.unit ? (
                      <span className="font-mono text-xs text-cyan-300/55">{indicator.unit}</span>
                    ) : null}
                  </div>
                  <div className="text-sm text-cyan-100/90">{indicator.indicatorName}</div>
                  {indicator.description ? (
                    <div className="text-xs leading-relaxed text-cyan-300/65">
                      {indicator.description}
                    </div>
                  ) : null}
                  <div className="grid grid-cols-2 gap-2 pt-1">
                    <MetaCell label="CATEGORY · 分类" value={indicator.category || '—'} />
                    <MetaCell
                      label="RANGE · 量程"
                      value={`${indicator.valueRange[0]} ~ ${indicator.valueRange[1]}`}
                    />
                  </div>
                </div>
              )}
            </div>
          </GlassPanel>
        </div>

        {/* 统计卡 */}
        <div className="col-span-12 lg:col-span-7">
          <GlassPanel title="MEASUREMENT STATS · 实测统计">
            <div className="p-4">
              {stats.isLoading ? (
                <Centered>
                  <Loader2 className="size-4 animate-spin" />
                  <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING STATS…</span>
                </Centered>
              ) : stats.isError ? (
                <ErrLine msg={stats.error instanceof Error ? stats.error.message : '统计加载失败'} />
              ) : !s ? (
                <div className="flex flex-col items-center gap-3 py-10">
                  <Gauge className="size-9 text-cyan-300/40" />
                  <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
                    暂无统计数据
                  </div>
                </div>
              ) : (
                <div className="flex flex-col gap-4">
                  <div className="flex items-center gap-5">
                    <RadialGauge
                      value={gaugePct}
                      label="AVG · 量程占比"
                      size={132}
                      color={color}
                      unit="%"
                    />
                    <div className="flex-1">
                      <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/55">
                        DISTRIBUTION · min → p50 → avg → p95 → max
                      </div>
                      <Sparkline data={distro} color={color} width={260} height={56} />
                      <div className="mt-1 flex justify-between font-mono text-[10px] text-cyan-300/45">
                        <span>{s.min}</span>
                        <span>{s.p50}</span>
                        <span>{s.avg.toFixed(1)}</span>
                        <span>{s.p95}</span>
                        <span>{s.max}</span>
                      </div>
                    </div>
                  </div>
                  <div className="grid grid-cols-3 gap-2">
                    <StatCell label="AVG · 均值" value={s.avg.toFixed(2)} color={color} />
                    <StatCell label="MIN · 最小" value={String(s.min)} color="#5b9eff" />
                    <StatCell label="MAX · 最大" value={String(s.max)} color="#ffaa00" />
                    <StatCell label="P50 · 中位" value={String(s.p50)} color="#00ff88" />
                    <StatCell label="P95 · 95分位" value={String(s.p95)} color="#a855f7" />
                    <StatCell
                      label="SAMPLES · 样本数"
                      value={Number(s.sampleCount ?? 0).toLocaleString()}
                      color="#00f0ff"
                    />
                  </div>
                </div>
              )}
            </div>
          </GlassPanel>
        </div>
      </div>
    </PageShell>
  )
}

function MetaCell({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-sm border border-cyan-500/12 bg-[#070b16]/45 px-3 py-2">
      <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/45">{label}</div>
      <div className="mt-0.5 text-sm text-cyan-100/90">{value}</div>
    </div>
  )
}

function StatCell({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <div
      className="rounded-sm border-l-2 bg-[#070b16]/45 px-3 py-2"
      style={{ borderLeftColor: color }}
    >
      <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/55">{label}</div>
      <div className="font-display text-xl font-bold leading-tight text-glow" style={{ color }}>
        {value}
      </div>
    </div>
  )
}

function Centered({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">{children}</div>
  )
}

function ErrLine({ msg }: { msg: string }) {
  return (
    <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
      FAILURE · {msg}
    </div>
  )
}
