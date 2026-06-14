import { useMemo, useState } from 'react'
import { Search, RefreshCcw, Share2, GitBranch, Radio } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useNeighborParams } from '@core/hooks/api/useConfig'
import type { NeighborParam } from '@core/types/config'
import { formatTime } from '@/lib/format'
import { StatCard, HudLoading, HudError, HudEmpty } from './_hud'

const NBR_TYPE: Record<NeighborParam['neighborType'], { label: string; color: string; badge: string }> = {
  'intra-freq': { label: '同频', color: '#00ff88', badge: 'ok' },
  'inter-freq': { label: '异频', color: '#ffaa00', badge: 'warning' },
  'inter-rat': { label: '异系统', color: '#a855f7', badge: 'unknown' },
}

// ===========================================================================
// CONFIG · 邻区参数
// 真实邻区关系列表（useNeighborParams）→ 按源小区分组 + 邻区类型过滤 →
// 选中邻区看其 params（同/异频/异系统切换参数）。
// ===========================================================================
export default function NeighborParams() {
  const [keyword, setKeyword] = useState('')
  const [nbrType, setNbrType] = useState<'all' | NeighborParam['neighborType']>('all')
  const [selectedId, setSelectedId] = useState('')

  const nbrQ = useNeighborParams({ page: 1, pageSize: 300 })
  const neighbors = nbrQ.data?.items ?? []

  const kw = keyword.trim().toLowerCase()
  const rows = neighbors.filter((n) => {
    if (nbrType !== 'all' && n.neighborType !== nbrType) return false
    if (!kw) return true
    return (
      n.sourceCellName?.toLowerCase().includes(kw) ||
      n.targetCellName?.toLowerCase().includes(kw) ||
      n.sourceCellId?.toLowerCase().includes(kw) ||
      n.targetCellId?.toLowerCase().includes(kw)
    )
  })

  const effectiveId = selectedId && rows.some((n) => n.id === selectedId) ? selectedId : rows[0]?.id || ''
  const selected = neighbors.find((n) => n.id === effectiveId) ?? null

  const sourceCells = useMemo(
    () => new Set(neighbors.map((n) => n.sourceCellId)).size,
    [neighbors],
  )

  return (
    <PageShell
      code="F02"
      title="NEIGHBOR · 邻区参数"
      subtitle="ANR / NEIGHBOR RELATION"
      isFetching={nbrQ.isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="源 / 目标小区"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={() => void nbrQ.refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-4 gap-3">
        <StatCard label="RELATIONS · 邻区关系" value={nbrQ.data?.total ?? neighbors.length} color="#00f0ff" />
        <StatCard label="SOURCE CELLS · 源小区" value={sourceCells} color="#a855f7" />
        <StatCard
          label="INTRA-FREQ · 同频"
          value={neighbors.filter((n) => n.neighborType === 'intra-freq').length}
          color="#00ff88"
        />
        <StatCard
          label="INTER-RAT · 异系统"
          value={neighbors.filter((n) => n.neighborType === 'inter-rat').length}
          color="#ff7a1a"
        />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        {(['all', 'intra-freq', 'inter-freq', 'inter-rat'] as const).map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setNbrType(t)}
            className={`chip ${nbrType === t ? 'text-[#00f0ff] shadow-[0_0_10px_currentColor]' : 'text-[#6b86b6]'}`}
          >
            {t === 'all' ? '全部类型' : NBR_TYPE[t].label}
          </button>
        ))}
      </div>

      <div className="grid grid-cols-[1fr_360px] gap-3">
        <GlassPanel title="NEIGHBORS · 邻区列表" meta={`${rows.length}`} className="min-h-0">
          {nbrQ.isLoading ? (
            <HudLoading />
          ) : nbrQ.isError ? (
            <HudError error={nbrQ.error} />
          ) : rows.length === 0 ? (
            <HudEmpty icon={Share2} text="无邻区关系" />
          ) : (
            <div className="max-h-[56vh] overflow-auto">
              <div className="grid grid-cols-[1.4fr_1.4fr_90px_1fr] gap-3 border-b border-cyan-500/15 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/45">
                <span>SOURCE / 源</span>
                <span>TARGET / 目标</span>
                <span>TYPE</span>
                <span>UPDATED</span>
              </div>
              {rows.map((n) => {
                const on = n.id === effectiveId
                const ty = NBR_TYPE[n.neighborType]
                return (
                  <button
                    key={n.id}
                    type="button"
                    onClick={() => setSelectedId(n.id)}
                    className={`grid w-full grid-cols-[1.4fr_1.4fr_90px_1fr] items-center gap-3 border-b border-cyan-500/8 px-3 py-2 text-left transition-colors ${
                      on ? 'bg-cyan-500/12' : 'hover:bg-cyan-500/5'
                    }`}
                  >
                    <div className="min-w-0">
                      <div className="truncate text-xs text-cyan-100/90">{n.sourceCellName}</div>
                      <code className="block truncate font-mono text-[10px] text-cyan-300/45">{n.sourceCellId}</code>
                    </div>
                    <div className="min-w-0">
                      <div className="truncate text-xs text-cyan-100/90">{n.targetCellName}</div>
                      <code className="block truncate font-mono text-[10px] text-cyan-300/45">{n.targetCellId}</code>
                    </div>
                    <div>
                      <StatusBadge status={ty.badge} label={ty.label} />
                    </div>
                    <div className="font-mono text-[11px] text-cyan-300/65">{formatTime(n.updateTime)}</div>
                  </button>
                )
              })}
            </div>
          )}
        </GlassPanel>

        <GlassPanel
          title={selected ? 'RELATION · 邻区详情' : 'RELATION · 邻区详情'}
          meta={selected ? NBR_TYPE[selected.neighborType].label : undefined}
          className="min-h-0"
        >
          {selected ? (
            <NeighborDetail neighbor={selected} />
          ) : (
            <HudEmpty icon={Radio} text="选择邻区查看切换参数" />
          )}
        </GlassPanel>
      </div>
    </PageShell>
  )
}

function NeighborDetail({ neighbor }: { neighbor: NeighborParam }) {
  const ty = NBR_TYPE[neighbor.neighborType]
  const entries = Object.entries(neighbor.params ?? {})
  return (
    <div className="max-h-[56vh] overflow-auto p-3">
      <div className="mb-3 flex items-center gap-2">
        <GitBranch className="size-4" style={{ color: ty.color }} />
        <span className="font-display text-sm font-bold text-cyan-100">{neighbor.sourceCellName}</span>
        <span className="text-cyan-300/40">→</span>
        <span className="font-display text-sm font-bold text-cyan-100">{neighbor.targetCellName}</span>
      </div>
      <div className="mb-3 flex flex-wrap gap-2">
        <StatusBadge status={ty.badge} label={ty.label} />
        <span className="chip text-[#5b9eff]">建 {formatTime(neighbor.createTime)}</span>
      </div>

      <div className="mb-2 font-mono text-[11px] uppercase tracking-[0.18em] text-cyan-300/55">
        SWITCH PARAMS · 切换参数 ({entries.length})
      </div>
      {entries.length > 0 ? (
        <div className="space-y-1">
          {entries.map(([k, v]) => (
            <div key={k} className="flex items-center justify-between gap-2 rounded-sm border border-cyan-500/10 px-3 py-1.5">
              <code className="min-w-0 flex-1 truncate font-mono text-[11px] text-cyan-200/80">{k}</code>
              <span className="shrink-0 font-mono text-[11px] text-cyan-100/90">{String(v)}</span>
            </div>
          ))}
        </div>
      ) : (
        <p className="font-mono text-[11px] text-cyan-300/45">无切换参数</p>
      )}
    </div>
  )
}
