import { useMemo, useState } from 'react'
import { Loader2, RefreshCcw, Crosshair, Search, X } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { useGeoData, useMapStats } from '@core/hooks/api/useTopology'
import type { Site, TopoNode } from '@core/types/topology'

// ── 视觉常量 ──
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

// 中国本土经纬度大致范围（用于把经纬度投影到画布）
const LNG_MIN = 73
const LNG_MAX = 135
const LAT_MIN = 18
const LAT_MAX = 53
const MAP_W = 960
const MAP_H = 600
const PAD = 40

interface SitePos {
  site: Site
  x: number
  y: number
}

function project(lng: number, lat: number): { x: number; y: number } {
  const x = PAD + ((lng - LNG_MIN) / (LNG_MAX - LNG_MIN)) * (MAP_W - PAD * 2)
  const y = PAD + ((LAT_MAX - lat) / (LAT_MAX - LAT_MIN)) * (MAP_H - PAD * 2)
  return { x, y }
}

export default function GISMapView() {
  const [keyword, setKeyword] = useState('')
  const [selected, setSelected] = useState<Site | null>(null)

  // 主数据：地理数据（站点 + 节点经纬度）
  const {
    data: geo,
    isLoading,
    isError,
    error,
    isFetching,
    refetch,
  } = useGeoData()
  // 辅助：地图统计（在线/离线/告警计数），失败不阻塞主图
  const { data: stats } = useMapStats()

  const sites = useMemo<Site[]>(() => geo?.sites ?? [], [geo])
  const nodes = useMemo<TopoNode[]>(() => geo?.nodes ?? [], [geo])

  // 仅保留含有效经纬度且落在投影范围内的站点
  const geoSites = useMemo(
    () =>
      sites.filter(
        (s) =>
          s.longitude != null &&
          s.latitude != null &&
          s.longitude >= LNG_MIN &&
          s.longitude <= LNG_MAX &&
          s.latitude >= LAT_MIN &&
          s.latitude <= LAT_MAX
      ),
    [sites]
  )

  const filtered = useMemo(() => {
    const kw = keyword.trim()
    if (!kw) return geoSites
    return geoSites.filter(
      (s) => s.name.includes(kw) || (s.address ?? '').includes(kw)
    )
  }, [geoSites, keyword])

  const positions = useMemo<SitePos[]>(
    () =>
      filtered.map((s) => {
        const { x, y } = project(s.longitude as number, s.latitude as number)
        return { site: s, x, y }
      }),
    [filtered]
  )

  const siteStatusCount = useMemo(() => {
    const acc = { active: 0, maintenance: 0, inactive: 0 }
    for (const s of geoSites) {
      if (s.status in acc) acc[s.status as keyof typeof acc] += 1
    }
    return acc
  }, [geoSites])

  const nodeStatusCount = useMemo(() => {
    const acc: Record<string, number> = {}
    for (const n of nodes) acc[n.status] = (acc[n.status] ?? 0) + 1
    return acc
  }, [nodes])

  return (
    <PageShell
      code="F06"
      title="GIS MAP · 地理态势"
      subtitle="GEO OVERLAY · 站点经纬投影 / 设备分布"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-60 pl-9"
              placeholder="站点名称 / 地址"
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
      {/* ── 统计带 ── */}
      <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-5">
        <Stat label="TOTAL · 设备" value={stats?.total ?? nodes.length} color="#00f0ff" />
        <Stat
          label="ONLINE · 激活"
          value={stats?.statusCount.onlineActive ?? siteStatusCount.active}
          color="#00ff88"
        />
        <Stat
          label="STANDBY · 未激活"
          value={stats?.statusCount.onlineInactive ?? siteStatusCount.maintenance}
          color="#ffaa00"
        />
        <Stat
          label="OFFLINE · 离线"
          value={stats?.statusCount.offline ?? siteStatusCount.inactive}
          color="#525a78"
        />
        <Stat label="ALARM · 告警" value={stats?.alarmCount ?? 0} color="#ff2d6f" />
      </div>

      <div className="grid grid-cols-1 gap-3 xl:grid-cols-[1fr_320px]">
        {/* ── 地图 ── */}
        <GlassPanel
          title="MAP · 站点投影"
          meta={`${positions.length} SITES · CN GRID`}
          strong
          className="min-h-[560px]"
        >
          <div className="relative h-[560px]">
            {isLoading ? (
              <Center>
                <Loader2 className="size-6 animate-spin text-cyan-300/70" />
              </Center>
            ) : isError ? (
              <Center>
                <span className="font-mono text-sm text-rose-300">
                  GEO SYNC FAILED · {error instanceof Error ? error.message : '未知错误'}
                </span>
              </Center>
            ) : positions.length === 0 ? (
              <Center>
                <span className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/40">
                  {geoSites.length === 0 ? 'NO GEO-TAGGED SITES' : 'NO MATCH · 无匹配站点'}
                </span>
              </Center>
            ) : (
              <svg
                viewBox={`0 0 ${MAP_W} ${MAP_H}`}
                preserveAspectRatio="xMidYMid meet"
                className="size-full"
              >
                <defs>
                  <filter id="gisGlow">
                    <feGaussianBlur stdDeviation="2.2" />
                    <feMerge>
                      <feMergeNode />
                      <feMergeNode in="SourceGraphic" />
                    </feMerge>
                  </filter>
                </defs>

                {/* 经纬网格 */}
                {Array.from({ length: 7 }).map((_, i) => {
                  const x = PAD + (i / 6) * (MAP_W - PAD * 2)
                  return (
                    <line
                      key={`vx-${i}`}
                      x1={x}
                      y1={PAD}
                      x2={x}
                      y2={MAP_H - PAD}
                      stroke="rgba(0,240,255,0.08)"
                      strokeWidth={1}
                    />
                  )
                })}
                {Array.from({ length: 6 }).map((_, i) => {
                  const y = PAD + (i / 5) * (MAP_H - PAD * 2)
                  return (
                    <line
                      key={`hz-${i}`}
                      x1={PAD}
                      y1={y}
                      x2={MAP_W - PAD}
                      y2={y}
                      stroke="rgba(0,240,255,0.08)"
                      strokeWidth={1}
                    />
                  )
                })}
                <rect
                  x={PAD}
                  y={PAD}
                  width={MAP_W - PAD * 2}
                  height={MAP_H - PAD * 2}
                  fill="none"
                  stroke="rgba(0,240,255,0.18)"
                  strokeWidth={1}
                  strokeDasharray="4 6"
                />

                {/* 站点点位 */}
                {positions.map((p) => {
                  const color = SITE_STATUS_COLOR[p.site.status] ?? '#525a78'
                  const isSel = selected?.id === p.site.id
                  return (
                    <g
                      key={p.site.id}
                      className="cursor-pointer"
                      onClick={() => setSelected(p.site)}
                    >
                      {isSel && (
                        <circle
                          cx={p.x}
                          cy={p.y}
                          r={16}
                          fill="none"
                          stroke="#fff"
                          strokeWidth={1}
                          opacity={0.8}
                        />
                      )}
                      <circle
                        cx={p.x}
                        cy={p.y}
                        r={Math.min(4 + Math.sqrt(p.site.deviceCount || 1) * 1.4, 14)}
                        fill={color}
                        fillOpacity={0.22}
                      />
                      <circle
                        cx={p.x}
                        cy={p.y}
                        r={4}
                        fill={color}
                        filter="url(#gisGlow)"
                      />
                    </g>
                  )
                })}
              </svg>
            )}
          </div>

          {/* 图例 */}
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2 border-t border-cyan-500/10 px-3 py-2">
            <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/55">
              LEGEND
            </span>
            {(['active', 'maintenance', 'inactive'] as const).map((s) => (
              <span key={s} className="flex items-center gap-1.5">
                <span
                  className="size-2 rounded-full"
                  style={{
                    background: SITE_STATUS_COLOR[s],
                    boxShadow: `0 0 6px ${SITE_STATUS_COLOR[s]}`,
                  }}
                />
                <span className="text-[10px] text-cyan-300/70">
                  {SITE_STATUS_LABEL[s]} · {siteStatusCount[s]}
                </span>
              </span>
            ))}
            <span className="ml-auto font-mono text-[10px] text-cyan-300/40">
              点位面积 ∝ 设备数
            </span>
          </div>
        </GlassPanel>

        {/* ── 右栏 ── */}
        <div className="flex flex-col gap-3">
          {/* 选中站点详情 */}
          {selected && (
            <GlassPanel title="SITE · 站点详情" meta="SELECTED">
              <div className="relative p-3">
                <button
                  type="button"
                  onClick={() => setSelected(null)}
                  className="absolute right-2 top-2 text-cyan-300/50 hover:text-cyan-200"
                  aria-label="关闭"
                >
                  <X className="size-3.5" />
                </button>
                <div className="mb-2 font-display text-sm font-bold text-cyan-100">
                  {selected.name}
                </div>
                <DetailRow
                  k="状态"
                  v={
                    <span style={{ color: SITE_STATUS_COLOR[selected.status] ?? '#6b86b6' }}>
                      {SITE_STATUS_LABEL[selected.status] ?? selected.status}
                    </span>
                  }
                />
                <DetailRow k="地址" v={selected.address || '—'} />
                <DetailRow
                  k="经纬度"
                  v={`${selected.longitude?.toFixed(4) ?? '—'}, ${selected.latitude?.toFixed(4) ?? '—'}`}
                  mono
                />
                <DetailRow k="设备数" v={String(selected.deviceCount)} mono />
                <DetailRow k="域" v={selected.domainId || '—'} mono />
                <div className="mt-3">
                  <NeonButton
                    icon={<Crosshair />}
                    className="!py-1 !px-2"
                    onClick={() => setSelected(selected)}
                  >
                    定位中心
                  </NeonButton>
                </div>
              </div>
            </GlassPanel>
          )}

          {/* 节点状态分布 */}
          <GlassPanel title="NODES · 节点状态" meta={`${nodes.length}`}>
            <div className="grid grid-cols-2 gap-px border-b border-cyan-500/10 bg-cyan-500/5">
              {(['online', 'offline', 'alarm', 'maintenance'] as const).map((s) => (
                <MiniStat
                  key={s}
                  label={s.toUpperCase()}
                  value={nodeStatusCount[s] ?? 0}
                  color={NODE_STATUS_COLOR[s]}
                />
              ))}
            </div>
          </GlassPanel>

          {/* 站点清单 */}
          <GlassPanel title="SITES · 站点清单" meta={`${filtered.length}`}>
            <div className="max-h-[300px] overflow-auto">
              {isLoading ? (
                <div className="flex items-center justify-center gap-2 py-8 text-cyan-300/60">
                  <Loader2 className="size-4 animate-spin" />
                  <span className="font-mono text-[10px] uppercase tracking-[0.2em]">
                    SYNCING…
                  </span>
                </div>
              ) : filtered.length === 0 ? (
                <div className="py-8 text-center font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/40">
                  NO SITES
                </div>
              ) : (
                filtered.map((s) => {
                  const color = SITE_STATUS_COLOR[s.status] ?? '#525a78'
                  const isSel = selected?.id === s.id
                  return (
                    <button
                      key={s.id}
                      type="button"
                      onClick={() => setSelected(s)}
                      className={`flex w-full items-center gap-2 border-b border-cyan-500/8 px-3 py-2 text-left transition-colors ${
                        isSel ? 'bg-cyan-500/10' : 'hover:bg-cyan-500/5'
                      }`}
                    >
                      <span
                        className="size-2 shrink-0 rounded-full"
                        style={{ background: color, boxShadow: `0 0 6px ${color}` }}
                      />
                      <div className="min-w-0 flex-1">
                        <div className="truncate font-mono text-xs text-cyan-100">
                          {s.name}
                        </div>
                        <div className="truncate text-[10px] text-cyan-300/55">
                          {s.address || '—'}
                        </div>
                      </div>
                      <span className="shrink-0 font-display text-sm font-bold text-cyan-200">
                        {s.deviceCount}
                      </span>
                    </button>
                  )
                })
              )}
            </div>
          </GlassPanel>
        </div>
      </div>
    </PageShell>
  )
}

function Stat({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: color }}
    >
      <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
        {label}
      </div>
      <div
        className="font-display text-2xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value.toLocaleString()}
      </div>
    </div>
  )
}

function MiniStat({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div className="bg-[#03050d]/40 px-2 py-2 text-center">
      <div className="font-mono text-[9px] uppercase tracking-[0.15em] text-cyan-300/55">
        {label}
      </div>
      <div
        className="font-display text-lg font-bold leading-tight"
        style={{ color, textShadow: `0 0 6px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}

function DetailRow({ k, v, mono }: { k: string; v: React.ReactNode; mono?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-3 border-b border-cyan-500/8 py-1.5">
      <span className="font-mono text-[10px] uppercase tracking-[0.15em] text-cyan-300/55">
        {k}
      </span>
      <span
        className={`min-w-0 truncate text-right text-xs text-cyan-100/90 ${
          mono ? 'font-mono' : ''
        }`}
      >
        {v}
      </span>
    </div>
  )
}

function Center({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-full w-full items-center justify-center">{children}</div>
  )
}
