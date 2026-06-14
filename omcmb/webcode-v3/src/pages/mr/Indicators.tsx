import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Loader2, RefreshCcw, Search, SignalHigh, ChevronRight } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { useMRIndicators } from '@core/hooks/api/useMR'

const PAGE_SIZE = 24

const CAT_COLOR: Record<string, string> = {
  覆盖指标: '#00f0ff',
  覆盖质量: '#00f0ff',
  质量指标: '#00ff88',
  MIMO指标: '#a855f7',
  接入指标: '#ffaa00',
  无线接入: '#ffaa00',
  移动性: '#ff7a1a',
  系统负荷: '#5b9eff',
  干扰: '#ff2d6f',
}

function catColor(cat: string): string {
  return CAT_COLOR[cat] ?? '#6b86b6'
}

/**
 * MR 指标库（对齐 v1 mr/Indicators）。
 * 数据走 @core useMRIndicators（mock/real 自动切换）。点指标卡进入 /mr/indicators/:code 详情。
 */
export default function Indicators() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')

  const { data, isLoading, isError, error, isFetching, refetch } = useMRIndicators({
    page,
    pageSize: PAGE_SIZE,
  })

  const all = data?.items ?? []
  const rows = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return all
    return all.filter(
      (i) =>
        i.indicatorName.toLowerCase().includes(kw) ||
        i.indicatorCode.toLowerCase().includes(kw),
    )
  }, [all, keyword])

  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <PageShell
      code="F05"
      title="MR INDICATORS · 测量指标库"
      subtitle="RSRP / RSRQ / SINR / CQI · MEASUREMENT INDICATOR CATALOG"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-60 pl-9"
              placeholder="指标名 / 编码（本页过滤）"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={() => void refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {isLoading ? (
        <div className="flex items-center justify-center gap-2 py-20 text-cyan-300/60">
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING INDICATORS…</span>
        </div>
      ) : isError ? (
        <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
          FAILURE · {error instanceof Error ? error.message : '加载失败'}
        </div>
      ) : rows.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <SignalHigh className="size-10 text-cyan-400/50" />
          <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/60">
            NO INDICATORS · 暂无指标
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-2 lg:grid-cols-3 xl:grid-cols-4">
          {rows.map((ind) => {
            const color = catColor(ind.category)
            return (
              <button
                key={ind.id}
                type="button"
                onClick={() => navigate(`/mr/indicators/${encodeURIComponent(ind.indicatorCode)}`)}
                className="group fleet-row flex flex-col gap-2 rounded-sm px-3.5 py-3 text-left"
                style={{ ['--row-color' as never]: color }}
              >
                <div className="flex items-baseline justify-between gap-2">
                  <span className="font-mono text-sm font-bold text-glow" style={{ color }}>
                    {ind.indicatorCode}
                  </span>
                  {ind.unit ? (
                    <span className="font-mono text-[10px] text-cyan-300/55">{ind.unit}</span>
                  ) : null}
                </div>
                <div className="text-xs text-cyan-100/85">{ind.indicatorName}</div>
                {ind.description ? (
                  <div className="line-clamp-2 text-[11px] leading-relaxed text-cyan-300/55">
                    {ind.description}
                  </div>
                ) : null}
                <div className="mt-auto flex items-center justify-between border-t border-cyan-500/12 pt-2">
                  {ind.category ? (
                    <span className="chip" style={{ color }}>
                      {ind.category}
                    </span>
                  ) : (
                    <span className="font-mono text-[10px] text-cyan-300/40">
                      {ind.valueRange[0]} ~ {ind.valueRange[1]}
                    </span>
                  )}
                  <ChevronRight className="size-3.5 text-cyan-300/45 transition-transform group-hover:translate-x-0.5" />
                </div>
              </button>
            )
          })}
        </div>
      )}

      <div className="mt-4 flex items-center justify-between">
        <span className="font-mono text-[11px] text-cyan-300/55">
          PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
        </span>
        <div className="flex gap-2">
          <NeonButton onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
            ◂ PREV
          </NeonButton>
          <NeonButton
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages}
          >
            NEXT ▸
          </NeonButton>
        </div>
      </div>
    </PageShell>
  )
}
