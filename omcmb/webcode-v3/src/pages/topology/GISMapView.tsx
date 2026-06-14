import { useMemo, useState } from 'react'
import { Loader2, RefreshCcw, Crosshair, Search, X } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { useMapDevicesGeo, useMapStats } from '@core/hooks/api/useTopology'
import type { DeviceGeo, DeviceStatus, MapFilterParams } from '@core/types/map'

// ── 视觉常量 ──
// 设备显示状态 → STARFORGE 配色（与 v1/v2 的三态语义对齐：激活/未激活/离线）
const STATUS_COLOR: Record<DeviceStatus, string> = {
  onlineActive: '#00ff88',
  onlineInactive: '#ffaa00',
  offline: '#525a78',
}
const STATUS_LABEL: Record<DeviceStatus, string> = {
  onlineActive: '在线激活',
  onlineInactive: '在线未激活',
  offline: '离线',
}

const MAP_W = 960
const MAP_H = 600
const PAD = 40
// 投影边界兜底：当无标记可拟合时，退回覆盖赞比亚区域的经纬度窗口
// （数据已灌：550 设备落在赞比亚 lon~25-29 / lat~-12~-18）
const FALLBACK_BBOX = { minLng: 24, maxLng: 30, minLat: -18, maxLat: -11 }

/**
 * 地图标记（由 DeviceGeo 投影渲染所需的最小形状）。
 * 复用 frontend-core 已有的 DeviceGeo，不新增共享类型。
 */
interface MarkerSite {
  id: string
  name: string
  sn: string
  longitude: number
  latitude: number
  status: DeviceStatus
  address?: string
  groupName?: string
  alarmCount?: number
}

interface MarkerPos {
  site: MarkerSite
  x: number
  y: number
}

interface BBox {
  minLng: number
  maxLng: number
  minLat: number
  maxLat: number
}

/**
 * 按一组标记动态拟合投影边界（min/max + padding）。
 * 让赞比亚坐标（lon~28, lat~-15）落在可视区，而非被中国经纬度窗口投到屏幕外。
 */
function fitBBox(markers: MarkerSite[]): BBox {
  if (markers.length === 0) return FALLBACK_BBOX

  let minLng = Infinity
  let maxLng = -Infinity
  let minLat = Infinity
  let maxLat = -Infinity
  for (const m of markers) {
    if (m.longitude < minLng) minLng = m.longitude
    if (m.longitude > maxLng) maxLng = m.longitude
    if (m.latitude < minLat) minLat = m.latitude
    if (m.latitude > maxLat) maxLat = m.latitude
  }

  // 经纬度跨度过小（单点 / 高度聚集）时给一个最小窗口，避免投影除零或全部叠在中心
  const lngSpan = Math.max(maxLng - minLng, 0.05)
  const latSpan = Math.max(maxLat - minLat, 0.05)
  // 10% 外扩留白，使边缘标记不贴边
  const lngPad = lngSpan * 0.1
  const latPad = latSpan * 0.1
  return {
    minLng: minLng - lngPad,
    maxLng: maxLng + lngPad,
    minLat: minLat - latPad,
    maxLat: maxLat + latPad,
  }
}

function projectWith(bbox: BBox, lng: number, lat: number): { x: number; y: number } {
  const x =
    PAD + ((lng - bbox.minLng) / (bbox.maxLng - bbox.minLng)) * (MAP_W - PAD * 2)
  // 纬度向北为正，屏幕 y 向下为正 → 取反
  const y =
    PAD + ((bbox.maxLat - lat) / (bbox.maxLat - bbox.minLat)) * (MAP_H - PAD * 2)
  return { x, y }
}

/** DeviceGeo（已保证经纬度非空）→ 渲染用 MarkerSite */
function toMarker(d: DeviceGeo): MarkerSite {
  return {
    id: d.id,
    name: d.name,
    sn: d.sn,
    longitude: d.longitude as number,
    latitude: d.latitude as number,
    status: d.status,
    address: d.address,
    groupName: d.groupName,
    alarmCount: d.alarmCount,
  }
}

export default function GISMapView() {
  const [keyword, setKeyword] = useState('')
  const [selected, setSelected] = useState<MarkerSite | null>(null)

  // 主数据：设备地理坐标（/devices/geo，赞比亚 550 设备有坐标）。
  // 与 v1/v2 一致；初始化即启用（v3 无设备组筛选侧栏，不做 enabled 门控）。
  const filterParams = useMemo<MapFilterParams>(() => ({ pageSize: 10000 }), [])
  const {
    data: devicesGeoData,
    isLoading,
    isError,
    error,
    isFetching,
    refetch,
  } = useMapDevicesGeo(filterParams)

  // 辅助：地图统计（在线/离线/告警计数），失败不阻塞主图
  const { data: stats } = useMapStats()

  // 仅保留含有效经纬度的设备，投影成标记
  const markers = useMemo<MarkerSite[]>(() => {
    const items = devicesGeoData?.items ?? []
    return items
      .filter((d) => d.longitude != null && d.latitude != null)
      .map(toMarker)
  }, [devicesGeoData])

  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return markers
    return markers.filter(
      (m) =>
        m.name.toLowerCase().includes(kw) ||
        m.sn.toLowerCase().includes(kw) ||
        (m.address ?? '').toLowerCase().includes(kw)
    )
  }, [markers, keyword])

  // 投影边界按全量标记拟合（保持稳定视窗，不随搜索缩放跳动）
  const bbox = useMemo(() => fitBBox(markers), [markers])

  const positions = useMemo<MarkerPos[]>(
    () =>
      filtered.map((m) => {
        const { x, y } = projectWith(bbox, m.longitude, m.latitude)
        return { site: m, x, y }
      }),
    [filtered, bbox]
  )

  // 标记状态计数（统计接口缺失时的兜底来源）
  const markerStatusCount = useMemo(() => {
    const acc: Record<DeviceStatus, number> = {
      onlineActive: 0,
      onlineInactive: 0,
      offline: 0,
    }
    for (const m of markers) acc[m.status] += 1
    return acc
  }, [markers])

  const totalAlarms = useMemo(
    () => markers.reduce((sum, m) => sum + (m.alarmCount ?? 0), 0),
    [markers]
  )

  return (
    <PageShell
      code="F06"
      title="GIS MAP · 地理态势"
      subtitle="GEO OVERLAY · 设备经纬投影 / 分布态势"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-60 pl-9"
              placeholder="设备名称 / 序列号 / 地址"
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
        <Stat label="TOTAL · 设备" value={stats?.total ?? markers.length} color="#00f0ff" />
        <Stat
          label="ONLINE · 激活"
          value={stats?.statusCount.onlineActive ?? markerStatusCount.onlineActive}
          color="#00ff88"
        />
        <Stat
          label="STANDBY · 未激活"
          value={stats?.statusCount.onlineInactive ?? markerStatusCount.onlineInactive}
          color="#ffaa00"
        />
        <Stat
          label="OFFLINE · 离线"
          value={stats?.statusCount.offline ?? markerStatusCount.offline}
          color="#525a78"
        />
        <Stat label="ALARM · 告警" value={stats?.alarmCount ?? totalAlarms} color="#ff2d6f" />
      </div>

      <div className="grid grid-cols-1 gap-3 xl:grid-cols-[1fr_320px]">
        {/* ── 地图 ── */}
        <GlassPanel
          title="MAP · 设备投影"
          meta={`${positions.length} NODES · GEO GRID`}
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
                  {markers.length === 0 ? 'NO GEO-TAGGED DEVICES' : 'NO MATCH · 无匹配设备'}
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

                {/* 设备点位 */}
                {positions.map((p) => {
                  const color = STATUS_COLOR[p.site.status] ?? '#525a78'
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
                        r={Math.min(4 + Math.sqrt((p.site.alarmCount ?? 0) + 1) * 1.4, 14)}
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
            {(['onlineActive', 'onlineInactive', 'offline'] as const).map((s) => (
              <span key={s} className="flex items-center gap-1.5">
                <span
                  className="size-2 rounded-full"
                  style={{
                    background: STATUS_COLOR[s],
                    boxShadow: `0 0 6px ${STATUS_COLOR[s]}`,
                  }}
                />
                <span className="text-[10px] text-cyan-300/70">
                  {STATUS_LABEL[s]} · {markerStatusCount[s]}
                </span>
              </span>
            ))}
            <span className="ml-auto font-mono text-[10px] text-cyan-300/40">
              点位面积 ∝ 告警数
            </span>
          </div>
        </GlassPanel>

        {/* ── 右栏 ── */}
        <div className="flex flex-col gap-3">
          {/* 选中设备详情 */}
          {selected && (
            <GlassPanel title="DEVICE · 设备详情" meta="SELECTED">
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
                    <span style={{ color: STATUS_COLOR[selected.status] ?? '#6b86b6' }}>
                      {STATUS_LABEL[selected.status] ?? selected.status}
                    </span>
                  }
                />
                <DetailRow k="序列号" v={selected.sn || '—'} mono />
                <DetailRow k="地址" v={selected.address || '—'} />
                <DetailRow
                  k="经纬度"
                  v={`${selected.longitude.toFixed(4)}, ${selected.latitude.toFixed(4)}`}
                  mono
                />
                <DetailRow k="告警数" v={String(selected.alarmCount ?? 0)} mono />
                <DetailRow k="设备组" v={selected.groupName || '—'} />
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

          {/* 状态分布 */}
          <GlassPanel title="STATUS · 状态分布" meta={`${markers.length}`}>
            <div className="grid grid-cols-3 gap-px border-b border-cyan-500/10 bg-cyan-500/5">
              {(['onlineActive', 'onlineInactive', 'offline'] as const).map((s) => (
                <MiniStat
                  key={s}
                  label={STATUS_LABEL[s]}
                  value={
                    s === 'onlineActive'
                      ? stats?.statusCount.onlineActive ?? markerStatusCount.onlineActive
                      : s === 'onlineInactive'
                        ? stats?.statusCount.onlineInactive ?? markerStatusCount.onlineInactive
                        : stats?.statusCount.offline ?? markerStatusCount.offline
                  }
                  color={STATUS_COLOR[s]}
                />
              ))}
            </div>
          </GlassPanel>

          {/* 设备清单 */}
          <GlassPanel title="DEVICES · 设备清单" meta={`${filtered.length}`}>
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
                  NO DEVICES
                </div>
              ) : (
                filtered.map((m) => {
                  const color = STATUS_COLOR[m.status] ?? '#525a78'
                  const isSel = selected?.id === m.id
                  return (
                    <button
                      key={m.id}
                      type="button"
                      onClick={() => setSelected(m)}
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
                          {m.name}
                        </div>
                        <div className="truncate text-[10px] text-cyan-300/55">
                          {m.sn || m.address || '—'}
                        </div>
                      </div>
                      <span className="shrink-0 font-display text-sm font-bold text-cyan-200">
                        {m.alarmCount ?? 0}
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
