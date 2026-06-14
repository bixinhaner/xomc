import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Loader2, ArrowLeft, MapPin, Radio, Crosshair } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { RadialGauge } from '@/components/viz/RadialGauge'
import {
  useSiteById,
  useDomains,
  useTopoNodes,
} from '@core/hooks/api/useTopology'
import type { TopoNode } from '@core/types/topology'

const SITE_STATUS_COLOR: Record<string, string> = {
  active: '#00ff88',
  maintenance: '#ffaa00',
  inactive: '#525a78',
}
const SITE_STATUS_LABEL: Record<string, string> = {
  active: '在线',
  maintenance: '维护',
  inactive: '离线',
}
const NODE_STATUS_COLOR: Record<string, string> = {
  online: '#00ff88',
  offline: '#525a78',
  alarm: '#ff2d6f',
  maintenance: '#ffaa00',
}
const NODE_STATUS_LABEL: Record<string, string> = {
  online: '在线',
  offline: '离线',
  alarm: '告警',
  maintenance: '维护',
}
const NODE_TYPE_COLOR: Record<string, string> = {
  eNB: '#5b9eff',
  gNB: '#a855f7',
  CPE: '#00f0ff',
  eGW: '#ff7a1a',
  domain: '#00ff88',
  site: '#ffd400',
  router: '#7c8cff',
  switch: '#ff5d8f',
}

export default function SiteDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const {
    data: site,
    isLoading,
    isError,
    error,
    isFetching,
    refetch,
  } = useSiteById(id)
  const { data: domainsData } = useDomains()
  // 拉一批节点，客户端按 siteId 归属过滤出本站点设备
  const { data: nodesData, isLoading: nodesLoading } = useTopoNodes({
    pageSize: 1000,
  })

  const domainName = useMemo(() => {
    if (!site) return '—'
    const d = (domainsData ?? []).find((x) => x.id === site.domainId)
    return d?.name ?? site.domainId ?? '—'
  }, [site, domainsData])

  const siteNodes = useMemo<TopoNode[]>(() => {
    if (!site) return []
    return (nodesData?.items ?? []).filter((n) => n.siteId === site.id)
  }, [nodesData, site])

  const nodeStatusCount = useMemo(() => {
    const acc: Record<string, number> = {}
    for (const n of siteNodes) acc[n.status] = (acc[n.status] ?? 0) + 1
    return acc
  }, [siteNodes])

  const onlineRate = siteNodes.length
    ? ((nodeStatusCount.online ?? 0) / siteNodes.length) * 100
    : 0

  return (
    <PageShell
      code="F06"
      title="SITE DETAIL · 站点详情"
      subtitle={`SITE ${id.slice(0, 12)}${id.length > 12 ? '…' : ''}`}
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/topology/site')}>
            BACK
          </NeonButton>
          <NeonButton icon={<Crosshair />} onClick={() => void refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {isLoading ? (
        <div className="flex items-center justify-center gap-2 py-20 text-cyan-300/60">
          <Loader2 className="size-5 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">LOADING SITE…</span>
        </div>
      ) : isError ? (
        <div className="m-3 border border-rose-500/40 bg-rose-500/5 px-4 py-8 font-mono text-sm text-rose-300">
          <div className="font-bold uppercase tracking-[0.2em]">SITE SYNC FAILED</div>
          <div className="mt-1 text-rose-200/80">
            {error instanceof Error ? error.message : '未知错误'}
          </div>
        </div>
      ) : !site ? (
        <div className="m-3 border border-amber-500/30 bg-amber-500/5 px-4 py-10 text-center font-mono text-sm text-amber-300">
          <div className="font-bold uppercase tracking-[0.2em]">SITE NOT FOUND</div>
          <div className="mt-2 text-amber-200/70">未找到站点 {id}</div>
          <div className="mt-4">
            <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/topology/site')}>
              返回站点列表
            </NeonButton>
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3 xl:grid-cols-[1fr_320px]">
          {/* ── 主信息 ── */}
          <div className="flex flex-col gap-3">
            <GlassPanel title="OVERVIEW · 站点概览" meta={SITE_STATUS_LABEL[site.status] ?? site.status} strong>
              <div className="flex flex-col gap-4 p-4 md:flex-row md:items-center">
                <div className="flex items-center gap-4">
                  <span
                    className="flex size-14 items-center justify-center rounded-sm"
                    style={{
                      background: `${SITE_STATUS_COLOR[site.status] ?? '#525a78'}22`,
                      boxShadow: `0 0 18px ${SITE_STATUS_COLOR[site.status] ?? '#525a78'}44`,
                    }}
                  >
                    <MapPin
                      className="size-7"
                      style={{ color: SITE_STATUS_COLOR[site.status] ?? '#525a78' }}
                    />
                  </span>
                  <div>
                    <div className="font-display text-xl font-bold text-cyan-100">
                      {site.name}
                    </div>
                    <div className="font-mono text-[11px] text-cyan-300/55">{site.address || '—'}</div>
                  </div>
                </div>
                <div className="md:ml-auto">
                  <RadialGauge
                    value={onlineRate}
                    label="ONLINE"
                    size={110}
                    color={onlineRate >= 90 ? '#00ff88' : onlineRate >= 60 ? '#ffaa00' : '#ff2d6f'}
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-px border-t border-cyan-500/10 bg-cyan-500/5 md:grid-cols-4">
                <Field k="所属域" v={domainName} />
                <Field
                  k="状态"
                  v={SITE_STATUS_LABEL[site.status] ?? site.status}
                  color={SITE_STATUS_COLOR[site.status]}
                />
                <Field k="设备数" v={String(site.deviceCount)} />
                <Field
                  k="经纬度"
                  v={
                    site.longitude != null && site.latitude != null
                      ? `${site.longitude.toFixed(4)}, ${site.latitude.toFixed(4)}`
                      : '— 无坐标'
                  }
                  mono
                />
              </div>
            </GlassPanel>

            {/* 站点设备清单 */}
            <GlassPanel title="DEVICES · 站点设备" meta={`${siteNodes.length}`}>
              <div className="max-h-[360px] overflow-auto">
                {nodesLoading ? (
                  <div className="flex items-center justify-center gap-2 py-8 text-cyan-300/60">
                    <Loader2 className="size-4 animate-spin" />
                    <span className="font-mono text-[10px] uppercase tracking-[0.2em]">
                      SYNCING NODES…
                    </span>
                  </div>
                ) : siteNodes.length === 0 ? (
                  <div className="flex flex-col items-center gap-2 py-10 text-cyan-300/40">
                    <Radio className="size-7 opacity-40" />
                    <span className="font-mono text-[10px] uppercase tracking-[0.2em]">
                      该站点暂无关联设备节点
                    </span>
                  </div>
                ) : (
                  siteNodes.map((n) => {
                    const color = NODE_STATUS_COLOR[n.status] ?? '#525a78'
                    return (
                      <div
                        key={n.id}
                        className="flex items-center gap-3 border-b border-cyan-500/8 px-4 py-2.5"
                      >
                        <span
                          className="size-2 shrink-0 rounded-full"
                          style={{ background: color, boxShadow: `0 0 6px ${color}` }}
                        />
                        <div className="min-w-0 flex-1">
                          <div className="truncate font-mono text-xs text-cyan-100">
                            {n.label}
                          </div>
                          <div className="truncate font-mono text-[10px] text-cyan-300/55">
                            {n.deviceSn ?? '—'}
                          </div>
                        </div>
                        <span
                          className="shrink-0 font-mono text-[10px]"
                          style={{ color: NODE_TYPE_COLOR[n.type] ?? '#6b86b6' }}
                        >
                          {n.type}
                        </span>
                        <span className="shrink-0 font-mono text-[10px]" style={{ color }}>
                          {NODE_STATUS_LABEL[n.status] ?? n.status}
                        </span>
                      </div>
                    )
                  })
                )}
              </div>
            </GlassPanel>
          </div>

          {/* ── 右栏：节点状态分布 ── */}
          <div className="flex flex-col gap-3">
            <GlassPanel title="NODE STATUS · 状态分布" meta={`${siteNodes.length}`}>
              <div className="grid grid-cols-2 gap-px border-b border-cyan-500/10 bg-cyan-500/5">
                {(['online', 'offline', 'alarm', 'maintenance'] as const).map((s) => (
                  <MiniStat
                    key={s}
                    label={NODE_STATUS_LABEL[s]}
                    value={nodeStatusCount[s] ?? 0}
                    color={NODE_STATUS_COLOR[s]}
                  />
                ))}
              </div>
            </GlassPanel>

            <GlassPanel title="LOCATION · 坐标定位" meta="GEO">
              <div className="p-4">
                {site.longitude != null && site.latitude != null ? (
                  <svg viewBox="0 0 200 200" className="w-full">
                    <defs>
                      <filter id="siteGlow">
                        <feGaussianBlur stdDeviation="2.5" />
                        <feMerge>
                          <feMergeNode />
                          <feMergeNode in="SourceGraphic" />
                        </feMerge>
                      </filter>
                    </defs>
                    {[40, 70, 100].map((r) => (
                      <circle
                        key={r}
                        cx={100}
                        cy={100}
                        r={r}
                        fill="none"
                        stroke="rgba(0,240,255,0.14)"
                        strokeWidth={1}
                        strokeDasharray="2 6"
                      />
                    ))}
                    <line x1={0} y1={100} x2={200} y2={100} stroke="rgba(0,240,255,0.12)" />
                    <line x1={100} y1={0} x2={100} y2={200} stroke="rgba(0,240,255,0.12)" />
                    <circle cx={100} cy={100} r={24} fill="rgba(0,240,255,0.12)" />
                    <circle cx={100} cy={100} r={6} fill="#00f0ff" filter="url(#siteGlow)" />
                  </svg>
                ) : (
                  <div className="py-8 text-center font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/40">
                    无地理坐标
                  </div>
                )}
                <div className="mt-2 text-center font-mono text-[11px] text-cyan-300/70">
                  {site.longitude != null && site.latitude != null
                    ? `${site.longitude.toFixed(4)}°E · ${site.latitude.toFixed(4)}°N`
                    : '—'}
                </div>
              </div>
            </GlassPanel>
          </div>
        </div>
      )}
    </PageShell>
  )
}

function Field({
  k,
  v,
  color,
  mono,
}: {
  k: string
  v: string
  color?: string
  mono?: boolean
}) {
  return (
    <div className="bg-[#03050d]/40 px-3 py-3">
      <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/55">
        {k}
      </div>
      <div
        className={`mt-0.5 truncate text-sm ${mono ? 'font-mono' : 'font-display font-bold'}`}
        style={{ color: color ?? '#d8e9ff' }}
      >
        {v}
      </div>
    </div>
  )
}

function MiniStat({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div className="bg-[#03050d]/40 px-2 py-3 text-center">
      <div className="font-mono text-[9px] uppercase tracking-[0.15em] text-cyan-300/55">
        {label}
      </div>
      <div
        className="font-display text-xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 6px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}
