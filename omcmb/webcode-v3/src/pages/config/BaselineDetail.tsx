import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, GitCompare, RefreshCcw } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { useBaselineConfigById } from '@core/hooks/api/useConfig'
import type { BaselineConfig, ConfigParam } from '@core/types/config'
import { formatTime } from '@/lib/format'
import { StatCard, HudLoading, HudError, HudEmpty } from './_hud'

const BASELINE_STATUS: Record<BaselineConfig['status'], { label: string; badge: string }> = {
  active: { label: '生效', badge: 'active' },
  draft: { label: '草稿', badge: 'warning' },
  deprecated: { label: '废弃', badge: 'off' },
}

// ===========================================================================
// CONFIG · 基线详情（/config/baseline/:id）
// useParams 取 id → useBaselineConfigById 拉真实基线 → 参数差异对比表。
// ===========================================================================
export default function BaselineDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const detailQ = useBaselineConfigById(id)
  const baseline = detailQ.data ?? null
  const params: ConfigParam[] = baseline?.params ?? []
  const diffParams = params.filter((p) => String(p.paramValue) !== String(p.defaultValue))
  const compliance = params.length
    ? Math.round(((params.length - diffParams.length) / params.length) * 100)
    : 100

  const st = baseline ? BASELINE_STATUS[baseline.status] : null

  return (
    <PageShell
      code="F02"
      title={baseline ? `BASELINE · ${baseline.baselineName}` : 'BASELINE · 基线详情'}
      subtitle={id ? `BASELINE ${id}` : 'CONFIG BASELINE DETAIL'}
      isFetching={detailQ.isFetching}
      bare
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/config/baseline')}>
            返回基线
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => void detailQ.refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {detailQ.isLoading ? (
        <HudLoading />
      ) : detailQ.isError ? (
        <HudError error={detailQ.error} />
      ) : !baseline ? (
        <HudEmpty icon={GitCompare} text="未找到该基线" />
      ) : (
        <>
          <div className="mb-3 grid grid-cols-4 gap-3">
            <StatCard label="VERSION · 版本" value={`v${baseline.version}`} color="#00f0ff" />
            <StatCard label="PARAMS · 参数" value={params.length} color="#a855f7" />
            <StatCard
              label="DEVIATIONS · 偏离"
              value={diffParams.length}
              color={diffParams.length > 0 ? '#ffaa00' : '#00ff88'}
            />
            <StatCard label="DEVICE TYPE · 型号" value={baseline.deviceType || '—'} color="#5b9eff" />
          </div>

          <div className="grid grid-cols-[300px_1fr] gap-3">
            <GlassPanel title="META · 基线信息" className="min-h-0">
              <div className="flex flex-col items-center gap-3 p-4">
                <RadialGauge value={compliance} label="符合度" size={120} color="#00ff88" unit="%" />
                <div className="w-full space-y-1.5">
                  {st ? (
                    <Meta label="STATUS / 状态">
                      <StatusBadge status={st.badge} label={st.label} />
                    </Meta>
                  ) : null}
                  <Meta label="CREATOR / 创建者">{baseline.creator || '—'}</Meta>
                  <Meta label="CREATED / 创建">{formatTime(baseline.createTime)}</Meta>
                  <Meta label="UPDATED / 更新">{formatTime(baseline.updateTime)}</Meta>
                </div>
                {baseline.description ? (
                  <p className="w-full border-t border-cyan-500/10 pt-2 text-[11px] leading-relaxed text-cyan-300/65">
                    {baseline.description}
                  </p>
                ) : null}
              </div>
            </GlassPanel>

            <GlassPanel
              title="DIFF · 参数差异"
              meta={`${diffParams.length} DEVIATIONS · ${params.length} PARAMS`}
              className="min-h-0"
            >
              {params.length === 0 ? (
                <HudEmpty icon={GitCompare} text="该基线无参数" />
              ) : (
                <div className="max-h-[52vh] overflow-auto">
                  <div className="grid grid-cols-[1.6fr_1fr_1fr_90px] gap-3 border-b border-cyan-500/15 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/45">
                    <span>PARAM</span>
                    <span>BASELINE / 默认</span>
                    <span>CURRENT / 当前</span>
                    <span className="text-right">DIFF</span>
                  </div>
                  {params.map((p) => {
                    const isDiff = String(p.paramValue) !== String(p.defaultValue)
                    return (
                      <div
                        key={p.id}
                        className="grid grid-cols-[1.6fr_1fr_1fr_90px] items-center gap-3 border-b border-cyan-500/8 px-3 py-2 hover:bg-cyan-500/5"
                      >
                        <div className="min-w-0">
                          <div className="truncate text-xs text-cyan-100/90">{p.paramName}</div>
                          <code className="block truncate font-mono text-[10px] text-cyan-300/50">{p.paramCode}</code>
                        </div>
                        <div className="font-mono text-[11px] text-cyan-300/70">
                          {String(p.defaultValue)}
                          {p.unit ? <span className="text-cyan-300/40"> {p.unit}</span> : null}
                        </div>
                        <div
                          className="font-mono text-[11px]"
                          style={{ color: isDiff ? '#ffaa00' : 'rgba(190,225,255,0.7)', fontWeight: isDiff ? 600 : 400 }}
                        >
                          {String(p.paramValue)}
                          {p.unit ? <span className="opacity-50"> {p.unit}</span> : null}
                        </div>
                        <div className="flex justify-end">
                          {isDiff ? (
                            <StatusBadge status="warning" label="偏离" />
                          ) : (
                            <StatusBadge status="ok" label="一致" />
                          )}
                        </div>
                      </div>
                    )
                  })}
                </div>
              )}
            </GlassPanel>
          </div>
        </>
      )}
    </PageShell>
  )
}

function Meta({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-2 rounded-sm border border-cyan-500/10 px-2.5 py-1.5">
      <span className="font-mono text-[9px] uppercase tracking-[0.15em] text-cyan-300/40">{label}</span>
      <span className="truncate font-mono text-[11px] text-cyan-100/85">{children}</span>
    </div>
  )
}
