import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Search, RefreshCcw, GitCompare, FolderTree, ArrowRight } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useBaselineConfigs } from '@core/hooks/api/useConfig'
import type { BaselineConfig } from '@core/types/config'
import { formatTime } from '@/lib/format'
import { StatCard, HudLoading, HudError, HudEmpty } from './_hud'

const BASELINE_STATUS: Record<BaselineConfig['status'], { label: string; badge: string }> = {
  active: { label: '生效', badge: 'active' },
  draft: { label: '草稿', badge: 'warning' },
  deprecated: { label: '废弃', badge: 'off' },
}

// ===========================================================================
// CONFIG · 基线管理
// 真实基线列表（useBaselineConfigs）按 deviceType 分组成树；
// 列表项「查看差异」→ navigate(/config/baseline/:id) 进详情页。
// ===========================================================================
export default function BaselineManagement() {
  const navigate = useNavigate()
  const [keyword, setKeyword] = useState('')
  const [deviceType, setDeviceType] = useState('')

  const baselinesQ = useBaselineConfigs({ page: 1, pageSize: 200 })
  const baselines = baselinesQ.data?.items ?? []

  const kw = keyword.trim().toLowerCase()
  const filtered = baselines.filter((b) => {
    if (deviceType && (b.deviceType || '未分类') !== deviceType) return false
    if (!kw) return true
    return b.baselineName?.toLowerCase().includes(kw) || b.deviceType?.toLowerCase().includes(kw)
  })

  const deviceTypes = useMemo(
    () => Array.from(new Set(baselines.map((b) => b.deviceType || '未分类'))),
    [baselines],
  )

  const activeCount = baselines.filter((b) => b.status === 'active').length
  const draftCount = baselines.filter((b) => b.status === 'draft').length

  return (
    <PageShell
      code="F02"
      title="BASELINE · 基线管理"
      subtitle="CONFIG BASELINE LIBRARY"
      isFetching={baselinesQ.isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="基线名 / 设备型号"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={() => void baselinesQ.refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-4 gap-3">
        <StatCard label="BASELINES · 基线" value={baselinesQ.data?.total ?? baselines.length} color="#00f0ff" />
        <StatCard label="ACTIVE · 生效" value={activeCount} color="#00ff88" />
        <StatCard label="DRAFT · 草稿" value={draftCount} color="#ffaa00" />
        <StatCard label="DEVICE TYPES · 型号" value={deviceTypes.length} color="#a855f7" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={() => setDeviceType('')}
          className={`chip ${deviceType === '' ? 'text-[#00f0ff] shadow-[0_0_10px_currentColor]' : 'text-[#6b86b6]'}`}
        >
          全部型号
        </button>
        {deviceTypes.map((dt) => (
          <button
            key={dt}
            type="button"
            onClick={() => setDeviceType(dt)}
            className={`chip ${deviceType === dt ? 'text-[#00f0ff] shadow-[0_0_10px_currentColor]' : 'text-[#6b86b6]'}`}
          >
            <FolderTree className="size-3" />
            {dt}
          </button>
        ))}
      </div>

      <GlassPanel title="BASELINES" meta={`${filtered.length} / ${baselines.length}`} className="min-h-0">
        {baselinesQ.isLoading ? (
          <HudLoading />
        ) : baselinesQ.isError ? (
          <HudError error={baselinesQ.error} />
        ) : filtered.length === 0 ? (
          <HudEmpty icon={GitCompare} text="无基线" />
        ) : (
          <div className="max-h-[56vh] overflow-auto">
            <div className="grid grid-cols-[2fr_1fr_90px_90px_1fr_120px] gap-3 border-b border-cyan-500/15 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/45">
              <span>BASELINE / 基线</span>
              <span>DEVICE TYPE</span>
              <span>VER</span>
              <span>PARAMS</span>
              <span>UPDATED</span>
              <span className="text-right">DIFF</span>
            </div>
            {filtered.map((b) => {
              const st = BASELINE_STATUS[b.status]
              return (
                <div
                  key={b.id}
                  className="fleet-row grid grid-cols-[2fr_1fr_90px_90px_1fr_120px] items-center gap-3 rounded-sm px-3 py-2.5"
                  style={{ ['--row-color' as never]: b.status === 'active' ? '#00ff88' : '#00f0ff' }}
                >
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="truncate font-display text-sm font-bold text-cyan-100">
                        {b.baselineName}
                      </span>
                      <StatusBadge status={st.badge} label={st.label} />
                    </div>
                    <div className="truncate font-mono text-[10px] text-cyan-300/55">
                      {b.description || '无描述'}
                    </div>
                  </div>
                  <div className="truncate font-mono text-[11px] text-cyan-300/70">{b.deviceType || '—'}</div>
                  <div className="font-mono text-[11px] text-cyan-300/70">v{b.version}</div>
                  <div className="text-center font-display text-base font-bold text-cyan-200">
                    {b.params?.length ?? 0}
                  </div>
                  <div className="font-mono text-[11px] text-cyan-300/65">{formatTime(b.updateTime)}</div>
                  <div className="text-right">
                    <NeonButton
                      icon={<ArrowRight />}
                      onClick={() => navigate(`/config/baseline/${b.id}`)}
                    >
                      查看差异
                    </NeonButton>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </GlassPanel>
    </PageShell>
  )
}
