import { useState } from 'react'
import { Loader2, Search } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { Sparkline } from '@/components/viz/Sparkline'
import { useKPIList } from '@core/hooks/api/usePerformance'

const COLORS = ['#00f0ff', '#00ff88', '#ffaa00', '#a855f7', '#ff00aa', '#5b9eff']

export function PerformancePage() {
  const [keyword, setKeyword] = useState('')
  const { data, isLoading, isError } = useKPIList({ page: 1, pageSize: 60, keyword })
  const items = data?.items ?? []

  return (
    <PageShell
      code="F03"
      title="PERFORMANCE · 数据光谱"
      subtitle="KPI / COUNTER GAUGES · LIVE"
      bare
      toolbar={
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
          <input
            className="neon-input w-64 pl-9"
            placeholder="搜索 KPI 名称 / 编码"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
      }
    >
      {isLoading ? (
        <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
          <Loader2 className="size-5 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING SPECTRUM…</span>
        </div>
      ) : isError ? (
        <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
          SPECTRUM SYNC FAILED
        </div>
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
                    <div className="font-display text-sm font-bold text-cyan-100">
                      {kpi.kpiName}
                    </div>
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
          {items.length === 0 && (
            <div className="col-span-full border border-cyan-500/15 px-4 py-12 text-center font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/40">
              NO KPI ON BOARD
            </div>
          )}
        </div>
      )}
    </PageShell>
  )
}

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
