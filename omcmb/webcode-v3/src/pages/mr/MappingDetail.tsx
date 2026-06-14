import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Boxes, Loader2, RefreshCcw } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useMRMappings, useToggleMRMapping } from '@core/hooks/api/useMR'
import { formatTime } from '@/lib/format'

/**
 * 设备小区映射详情（/mr/device-mapping/:id）。
 * 在 useMRMappings 大页里按 id 命中（无单条接口），开关走 useToggleMRMapping。全部 @core 真实数据。
 */
export default function MappingDetail() {
  const navigate = useNavigate()
  const { id = '' } = useParams<{ id: string }>()

  const { data, isLoading, isError, error, isFetching, refetch } = useMRMappings({
    page: 1,
    pageSize: 500,
  })
  const toggle = useToggleMRMapping()

  const mapping = useMemo(
    () => (data?.items ?? []).find((m) => m.id === id) ?? null,
    [data, id],
  )

  return (
    <PageShell
      code="F05"
      title="MAPPING DETAIL · 映射详情"
      subtitle={mapping ? `${mapping.deviceSn} · ${mapping.cellId}` : 'DEVICE ⇄ CELL BINDING'}
      isFetching={isFetching}
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/mr/device-mapping')}>
            BACK
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => void refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {isLoading ? (
        <Centered>
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING MAPPING…</span>
        </Centered>
      ) : isError ? (
        <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
          FAILURE · {error instanceof Error ? error.message : '加载失败'}
        </div>
      ) : !mapping ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <Boxes className="size-10 text-cyan-300/40" />
          <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
            映射 {id} 不存在或已删除
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-12 gap-3">
          <div className="col-span-12 lg:col-span-7">
            <GlassPanel
              title="BINDING · 绑定信息"
              meta={
                <StatusBadge
                  status={mapping.enabled ? 'active' : 'off'}
                  label={mapping.enabled ? '已启用' : '已禁用'}
                />
              }
            >
              <div className="grid grid-cols-2 gap-px bg-cyan-500/10">
                <Field label="DEVICE SN · 设备" value={mapping.deviceSn} mono />
                <Field label="DEVICE NAME · 设备名" value={mapping.deviceName || '—'} />
                <Field label="CELL ID · 小区" value={mapping.cellId} mono />
                <Field label="CELL NAME · 小区名" value={mapping.cellName || '—'} />
                <Field
                  label="SAMPLING · 采样间隔"
                  value={`${mapping.samplingInterval} min`}
                />
                <Field
                  label="RECORDS · 采集记录数"
                  value={Number(mapping.totalRecords ?? 0).toLocaleString()}
                />
                <Field
                  label="LAST COLLECT · 最后采集"
                  value={mapping.lastCollectTime ? formatTime(mapping.lastCollectTime) : '从未采集'}
                  span2
                />
              </div>
            </GlassPanel>
          </div>

          <div className="col-span-12 lg:col-span-5">
            <GlassPanel title="CONTROL · 采集开关">
              <div className="flex flex-col items-center gap-4 p-6">
                <div
                  className="font-display text-5xl font-bold text-glow"
                  style={{ color: mapping.enabled ? '#00ff88' : '#525a78' }}
                >
                  {mapping.enabled ? 'ON' : 'OFF'}
                </div>
                <div className="font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/55">
                  {mapping.enabled ? 'MR 采集进行中' : 'MR 采集已停止'}
                </div>
                <NeonButton
                  tone={mapping.enabled ? 'danger' : 'cyan'}
                  disabled={toggle.isPending}
                  onClick={() => toggle.mutate({ id: mapping.id, enabled: !mapping.enabled })}
                >
                  {toggle.isPending ? (
                    <Loader2 className="size-3.5 animate-spin" />
                  ) : null}
                  {mapping.enabled ? '禁用采集' : '启用采集'}
                </NeonButton>
              </div>
            </GlassPanel>
          </div>
        </div>
      )}
    </PageShell>
  )
}

function Field({
  label,
  value,
  mono,
  span2,
}: {
  label: string
  value: string
  mono?: boolean
  span2?: boolean
}) {
  return (
    <div className={`bg-[#070b16]/55 px-4 py-3 ${span2 ? 'col-span-2' : ''}`}>
      <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/45">{label}</div>
      <div className={`mt-1 text-sm text-cyan-100/90 ${mono ? 'font-mono' : ''}`}>{value}</div>
    </div>
  )
}

function Centered({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-center gap-2 py-20 text-cyan-300/60">{children}</div>
  )
}
