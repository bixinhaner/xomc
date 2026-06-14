import { useMemo, useState } from 'react'
import { Loader2, Search, RefreshCcw, Layers, X } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { useTopoGraph, useSites, useDomains } from '@core/hooks/api/useTopology'
import type {
  TopoNode,
  TopoEdge,
  NodeType,
  NodeStatus,
  Site,
} from '@core/types/topology'

// ── 视觉常量 ──
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
const SITE_STATUS_COLOR: Record<string, string> = {
  active: '#00ff88',
  maintenance: '#ffaa00',
  inactive: '#525a78',
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
const EDGE_STATUS_COLOR: Record<string, string> = {
  active: 'rgba(0,240,255,0.42)',
  inactive: 'rgba(82,90,120,0.42)',
  degraded: 'rgba(255,170,0,0.5)',
}

const STATUS_OPTIONS: NodeStatus[] = ['online', 'offline', 'alarm', 'maintenance']
const LAYOUT_OPTIONS = ['force', 'tree', 'circular', 'hierarchy'] as const
const LAYOUT_LABEL: Record<string, string> = {
  force: 'FORCE',
  tree: 'TREE',
  circular: 'RING',
  hierarchy: 'TIER',
}
const LIMIT_OPTIONS = [100, 500, 1000, 2000]

// 同心轨道布局画布尺寸
const CANVAS_W = 960
const CANVAS_H = 640
const CENTER_X = 480
const CENTER_Y = 320
const RINGS = [90, 170, 250, 330]

interface NodePos {
  node: TopoNode
  cx: number
  cy: number
}

export function TopologyPage() {
  const [keyword, setKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<NodeStatus | ''>('')
  const [typeFilter, setTypeFilter] = useState<NodeType | ''>('')
  const [layoutType, setLayoutType] = useState<string>('force')
  const [limit, setLimit] = useState(500)
  const [selectedNode, setSelectedNode] = useState<TopoNode | null>(null)

  // 主数据：拓扑图（节点 + 边 + 统计），服务端按 type/status/limit 过滤
  const {
    data: graph,
    isLoading,
    isError,
    error,
    isFetching,
    refetch,
  } = useTopoGraph({
    layoutType,
    nodeType: typeFilter || undefined,
    status: statusFilter || undefined,
    limit,
  })
  // 辅助数据：站点 / 域（用于右栏概览，失败不阻塞主图）
  const { data: sitesData } = useSites({ pageSize: 100 })
  const { data: domains } = useDomains()

  const nodes = useMemo<TopoNode[]>(() => graph?.nodes ?? [], [graph])
  const edges = useMemo<TopoEdge[]>(() => graph?.edges ?? [], [graph])
  const stats = graph?.statistics
  const sites = useMemo<Site[]>(() => sitesData?.items ?? [], [sitesData])

  // 客户端关键字搜索（标签 / 设备 SN）
  const filteredNodes = useMemo(() => {
    const kw = keyword.trim()
    if (!kw) return nodes
    return nodes.filter(
      (n) => n.label.includes(kw) || (n.deviceSn ?? '').includes(kw)
    )
  }, [nodes, keyword])

  // 可见节点的 id 集合，用于过滤边
  const visibleIds = useMemo(
    () => new Set(filteredNodes.map((n) => n.id)),
    [filteredNodes]
  )
  const filteredEdges = useMemo(
    () => edges.filter((e) => visibleIds.has(e.source) && visibleIds.has(e.target)),
    [edges, visibleIds]
  )

  // 同心轨道布局：把节点均匀铺在多层圆环上
  const positions = useMemo<NodePos[]>(() => {
    if (filteredNodes.length === 0) return []
    const perRing = 10
    return filteredNodes.map((node, i) => {
      const ring = Math.floor(i / perRing)
      const idxInRing = i % perRing
      const radius = 80 + ring * 72
      const angle = (idxInRing / perRing) * Math.PI * 2 + ring * 0.45
      return {
        node,
        cx: CENTER_X + Math.cos(angle) * radius,
        cy: CENTER_Y + Math.sin(angle) * radius,
      }
    })
  }, [filteredNodes])

  // 边的渲染坐标（找到两端节点位置）
  const posById = useMemo(() => {
    const m = new Map<string, NodePos>()
    positions.forEach((p) => m.set(p.node.id, p))
    return m
  }, [positions])
  const edgeLines = useMemo(() => {
    const out: {
      key: string
      x1: number
      y1: number
      x2: number
      y2: number
      color: string
    }[] = []
    filteredEdges.forEach((e) => {
      const a = posById.get(e.source)
      const b = posById.get(e.target)
      if (a && b) {
        out.push({
          key: e.id,
          x1: a.cx,
          y1: a.cy,
          x2: b.cx,
          y2: b.cy,
          color: EDGE_STATUS_COLOR[e.status] ?? EDGE_STATUS_COLOR.inactive,
        })
      }
    })
    return out
  }, [filteredEdges, posById])

  // 站点状态分桶
  const siteStatusCount = useMemo(() => {
    const acc = { active: 0, maintenance: 0, inactive: 0 }
    for (const s of sites) {
      if (s.status in acc) acc[s.status as keyof typeof acc] += 1
    }
    return acc
  }, [sites])

  const atLimit = nodes.length >= limit
  const onlineRate =
    stats && stats.totalNodes > 0
      ? (stats.onlineNodes / stats.totalNodes) * 100
      : 0

  return (
    <PageShell
      code="F06"
      title="TOPOLOGY · 战场态势"
      subtitle="NETWORK GRAPH · 节点 / 边 / 域 实时拓扑"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-60 pl-9"
              placeholder="节点标签 / 设备 SN"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          {/* 状态筛选 */}
          {(['', ...STATUS_OPTIONS] as const).map((s) => (
            <button
              key={s || 'all-st'}
              type="button"
              onClick={() => setStatusFilter(s)}
              className={`chip transition-all ${
                statusFilter === s
                  ? 'shadow-[0_0_10px_currentColor]'
                  : 'opacity-55 hover:opacity-100'
              }`}
              style={{ color: s ? NODE_STATUS_COLOR[s] : '#00f0ff' }}
            >
              {s ? NODE_STATUS_LABEL[s] : 'ALL'}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => void refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {/* ── 统计带 ── */}
      <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
        <Stat
          label="NODES · 节点"
          value={stats?.totalNodes ?? nodes.length}
          color="#00f0ff"
          sub={`已加载 ${nodes.length}/${limit}${atLimit ? ' · 上限' : ''}`}
          warn={atLimit}
        />
        <Stat
          label="ONLINE · 在线"
          value={stats?.onlineNodes ?? 0}
          color="#00ff88"
          sub={`${onlineRate.toFixed(1)}% 在线率`}
        />
        <Stat label="OFFLINE · 离线" value={stats?.offlineNodes ?? 0} color="#525a78" />
        <Stat label="ALARM · 告警" value={stats?.alarmNodes ?? 0} color="#ff2d6f" />
        <Stat
          label="MAINT · 维护"
          value={stats?.maintenanceNodes ?? 0}
          color="#ffaa00"
        />
        <Stat
          label="EDGES · 链路"
          value={stats?.totalEdges ?? edges.length}
          color="#a855f7"
          sub={
            stats
              ? `活动 ${stats.activeEdges} · 降级 ${stats.degradedEdges}`
              : undefined
          }
        />
      </div>

      <div className="grid grid-cols-1 gap-3 xl:grid-cols-[1fr_320px]">
        {/* ── 主图 ── */}
        <GlassPanel
          title="GRAPH · 网络拓扑"
          meta={`${filteredNodes.length} NODES · ${filteredEdges.length} EDGES`}
          strong
          className="min-h-[520px]"
        >
          {/* 图工具条：布局 + 节点上限 */}
          <div className="flex flex-wrap items-center gap-2 border-b border-cyan-500/10 px-3 py-2">
            <Layers className="size-3.5 text-cyan-300/55" />
            <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/55">
              LAYOUT
            </span>
            {LAYOUT_OPTIONS.map((opt) => (
              <button
                key={opt}
                type="button"
                onClick={() => setLayoutType(opt)}
                className={`chip text-[10px] transition-all ${
                  layoutType === opt
                    ? 'text-cyan-200 shadow-[0_0_8px_currentColor]'
                    : 'text-cyan-300/45 hover:text-cyan-200'
                }`}
              >
                {LAYOUT_LABEL[opt]}
              </button>
            ))}
            <span className="mx-1 h-3.5 w-px bg-cyan-500/20" />
            <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/55">
              LIMIT
            </span>
            {LIMIT_OPTIONS.map((n) => (
              <button
                key={n}
                type="button"
                onClick={() => setLimit(n)}
                className={`chip text-[10px] transition-all ${
                  limit === n
                    ? 'text-cyan-200 shadow-[0_0_8px_currentColor]'
                    : 'text-cyan-300/45 hover:text-cyan-200'
                }`}
              >
                {n}
              </button>
            ))}
          </div>

          <div className="relative h-[460px]">
            {isLoading ? (
              <Center>
                <Loader2 className="size-6 animate-spin text-cyan-300/70" />
              </Center>
            ) : isError ? (
              <Center>
                <span className="font-mono text-sm text-rose-300">
                  SYNC FAILED · {error instanceof Error ? error.message : '未知错误'}
                </span>
              </Center>
            ) : filteredNodes.length === 0 ? (
              <Center>
                <span className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/40">
                  {nodes.length === 0 ? 'NO TOPOLOGY DATA' : 'NO MATCH · 无匹配节点'}
                </span>
              </Center>
            ) : (
              <svg
                viewBox={`0 0 ${CANVAS_W} ${CANVAS_H}`}
                preserveAspectRatio="xMidYMid meet"
                className="size-full"
              >
                <defs>
                  <radialGradient id="topoCenterGlow">
                    <stop offset="0%" stopColor="#00f0ff" stopOpacity="0.85" />
                    <stop offset="100%" stopColor="#00f0ff" stopOpacity="0" />
                  </radialGradient>
                  <filter id="topoGlow">
                    <feGaussianBlur stdDeviation="2" />
                    <feMerge>
                      <feMergeNode />
                      <feMergeNode in="SourceGraphic" />
                    </feMerge>
                  </filter>
                </defs>

                {/* 同心轨道 */}
                {RINGS.map((r) => (
                  <circle
                    key={r}
                    cx={CENTER_X}
                    cy={CENTER_Y}
                    r={r}
                    fill="none"
                    stroke="rgba(0,240,255,0.12)"
                    strokeWidth={1}
                    strokeDasharray="2 6"
                  />
                ))}

                {/* 中心 ACS */}
                <circle cx={CENTER_X} cy={CENTER_Y} r={60} fill="url(#topoCenterGlow)" />
                <circle
                  cx={CENTER_X}
                  cy={CENTER_Y}
                  r={14}
                  fill="#00f0ff"
                  filter="url(#topoGlow)"
                />
                <text
                  x={CENTER_X}
                  y={CENTER_Y + 4}
                  textAnchor="middle"
                  className="fill-[#03050d] font-bold"
                  fontSize="10"
                >
                  ACS
                </text>

                {/* 边 */}
                {edgeLines.map((e) => (
                  <line
                    key={e.key}
                    x1={e.x1}
                    y1={e.y1}
                    x2={e.x2}
                    y2={e.y2}
                    stroke={e.color}
                    strokeWidth={1}
                    strokeDasharray="4 4"
                  >
                    <animate
                      attributeName="stroke-dashoffset"
                      from="0"
                      to="-32"
                      dur="3s"
                      repeatCount="indefinite"
                    />
                  </line>
                ))}
                {/* 中心辐条：无显式边时连到 ACS，保证视觉连通 */}
                {edgeLines.length === 0 &&
                  positions.map((p) => (
                    <line
                      key={`spoke-${p.node.id}`}
                      x1={CENTER_X}
                      y1={CENTER_Y}
                      x2={p.cx}
                      y2={p.cy}
                      stroke="rgba(0,240,255,0.16)"
                      strokeWidth={1}
                      strokeDasharray="3 6"
                    />
                  ))}

                {/* 节点 */}
                {positions.map((p) => {
                  const color = NODE_STATUS_COLOR[p.node.status] ?? '#525a78'
                  const isSel = selectedNode?.id === p.node.id
                  return (
                    <g
                      key={p.node.id}
                      className="cursor-pointer"
                      onClick={() => setSelectedNode(p.node)}
                    >
                      <circle
                        cx={p.cx}
                        cy={p.cy}
                        r={isSel ? 22 : 18}
                        fill="none"
                        stroke={isSel ? '#fff' : color}
                        strokeWidth={1}
                        opacity={isSel ? 0.9 : 0.4}
                      />
                      <circle
                        cx={p.cx}
                        cy={p.cy}
                        r={8}
                        fill={color}
                        filter="url(#topoGlow)"
                      />
                      <text
                        x={p.cx}
                        y={p.cy + 28}
                        textAnchor="middle"
                        fill="#d8e9ff"
                        fontSize="10"
                        fontFamily="JetBrains Mono, monospace"
                      >
                        {p.node.label.length > 14
                          ? p.node.label.slice(0, 14) + '…'
                          : p.node.label}
                      </text>
                    </g>
                  )
                })}
              </svg>
            )}
          </div>

          {/* 图例 + 类型分布 */}
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2 border-t border-cyan-500/10 px-3 py-2">
            <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/55">
              LEGEND
            </span>
            {STATUS_OPTIONS.map((s) => (
              <span key={s} className="flex items-center gap-1.5">
                <span
                  className="size-2 rounded-full"
                  style={{
                    background: NODE_STATUS_COLOR[s],
                    boxShadow: `0 0 6px ${NODE_STATUS_COLOR[s]}`,
                  }}
                />
                <span className="text-[10px] text-cyan-300/70">
                  {NODE_STATUS_LABEL[s]}
                </span>
              </span>
            ))}
            {stats?.nodeTypeCounts &&
              Object.keys(stats.nodeTypeCounts).length > 0 && (
                <>
                  <span className="mx-1 h-3 w-px bg-cyan-500/20" />
                  {Object.entries(stats.nodeTypeCounts).map(([type, count]) => (
                    <button
                      key={type}
                      type="button"
                      onClick={() =>
                        setTypeFilter((prev) =>
                          prev === type ? '' : (type as NodeType)
                        )
                      }
                      className={`chip text-[10px] transition-all ${
                        typeFilter === type
                          ? 'shadow-[0_0_8px_currentColor]'
                          : 'opacity-65 hover:opacity-100'
                      }`}
                      style={{ color: NODE_TYPE_COLOR[type] ?? '#6b86b6' }}
                    >
                      {type}: {count}
                    </button>
                  ))}
                </>
              )}
          </div>
        </GlassPanel>

        {/* ── 右栏 ── */}
        <div className="flex flex-col gap-3">
          {/* 节点详情（选中时） */}
          {selectedNode && (
            <GlassPanel title="NODE · 节点详情" meta="SELECTED">
              <div className="relative p-3">
                <button
                  type="button"
                  onClick={() => setSelectedNode(null)}
                  className="absolute right-2 top-2 text-cyan-300/50 hover:text-cyan-200"
                  aria-label="关闭"
                >
                  <X className="size-3.5" />
                </button>
                <div className="mb-2 font-display text-sm font-bold text-cyan-100">
                  {selectedNode.label}
                </div>
                <DetailRow
                  k="状态"
                  v={
                    <span
                      style={{
                        color:
                          NODE_STATUS_COLOR[selectedNode.status] ?? '#6b86b6',
                      }}
                    >
                      {NODE_STATUS_LABEL[selectedNode.status] ?? selectedNode.status}
                    </span>
                  }
                />
                <DetailRow
                  k="类型"
                  v={
                    <span
                      style={{ color: NODE_TYPE_COLOR[selectedNode.type] ?? '#d8e9ff' }}
                    >
                      {selectedNode.type}
                    </span>
                  }
                />
                <DetailRow k="设备 SN" v={selectedNode.deviceSn ?? '—'} mono />
                <DetailRow k="站点" v={selectedNode.siteId ?? '—'} mono />
                <DetailRow k="域" v={selectedNode.domainId ?? '—'} mono />
              </div>
            </GlassPanel>
          )}

          {/* 节点清单 */}
          <GlassPanel
            title="NODES · 节点清单"
            meta={`${filteredNodes.length}`}
            className="min-h-0"
          >
            <div className="max-h-[300px] overflow-auto">
              {isLoading ? (
                <div className="flex items-center justify-center gap-2 py-8 text-cyan-300/60">
                  <Loader2 className="size-4 animate-spin" />
                  <span className="font-mono text-[10px] uppercase tracking-[0.2em]">
                    SYNCING…
                  </span>
                </div>
              ) : filteredNodes.length === 0 ? (
                <div className="py-8 text-center font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/40">
                  NO NODES
                </div>
              ) : (
                filteredNodes.slice(0, 200).map((n) => {
                  const color = NODE_STATUS_COLOR[n.status] ?? '#525a78'
                  const isSel = selectedNode?.id === n.id
                  return (
                    <button
                      key={n.id}
                      type="button"
                      onClick={() => setSelectedNode(n)}
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
                          {n.label}
                        </div>
                        <div className="truncate text-[10px] text-cyan-300/55">
                          {n.deviceSn ?? '—'}
                        </div>
                      </div>
                      <span
                        className="shrink-0 font-mono text-[10px]"
                        style={{ color: NODE_TYPE_COLOR[n.type] ?? '#6b86b6' }}
                      >
                        {n.type}
                      </span>
                    </button>
                  )
                })
              )}
              {filteredNodes.length > 200 && (
                <div className="px-3 py-2 text-center font-mono text-[10px] text-cyan-300/40">
                  仅显示前 200 · 共 {filteredNodes.length}
                </div>
              )}
            </div>
          </GlassPanel>

          {/* 站点概览 */}
          <GlassPanel title="SITES · 站点" meta={`${sites.length}`}>
            <div className="grid grid-cols-3 gap-px border-b border-cyan-500/10 bg-cyan-500/5">
              <MiniStat
                label="ACTIVE"
                value={siteStatusCount.active}
                color="#00ff88"
              />
              <MiniStat
                label="MAINT"
                value={siteStatusCount.maintenance}
                color="#ffaa00"
              />
              <MiniStat
                label="INACTIVE"
                value={siteStatusCount.inactive}
                color="#525a78"
              />
            </div>
            <div className="max-h-[220px] overflow-auto">
              {sites.length === 0 ? (
                <div className="py-6 text-center font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/40">
                  NO SITES
                </div>
              ) : (
                sites.map((s) => (
                  <div
                    key={s.id}
                    className="flex items-center gap-2 border-b border-cyan-500/8 px-3 py-2 hover:bg-cyan-500/5"
                  >
                    <span
                      className="size-2 shrink-0 rounded-full"
                      style={{
                        background: SITE_STATUS_COLOR[s.status] ?? '#525a78',
                        boxShadow: `0 0 6px ${SITE_STATUS_COLOR[s.status] ?? '#525a78'}`,
                      }}
                    />
                    <div className="min-w-0 flex-1">
                      <div className="truncate font-mono text-xs text-cyan-100">
                        {s.name}
                      </div>
                      <div className="truncate text-[10px] text-cyan-300/55">
                        {s.address || '—'}
                      </div>
                    </div>
                    <div className="font-display text-sm font-bold text-cyan-200">
                      {s.deviceCount}
                    </div>
                  </div>
                ))
              )}
            </div>
          </GlassPanel>

          {/* 域概览 */}
          <GlassPanel title="DOMAINS · 设备域" meta={`${domains?.length ?? 0}`}>
            <div className="max-h-[180px] overflow-auto">
              {!domains || domains.length === 0 ? (
                <div className="py-6 text-center font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/40">
                  NO DOMAINS
                </div>
              ) : (
                domains.map((d) => (
                  <div
                    key={d.id}
                    className="flex items-center gap-2 border-b border-cyan-500/8 px-3 py-1.5"
                    style={{ paddingLeft: 12 + (d.level - 1) * 12 }}
                  >
                    <span className="size-1.5 rounded-full bg-cyan-400/70" />
                    <span className="min-w-0 flex-1 truncate font-mono text-xs text-cyan-100/90">
                      {d.name}
                    </span>
                    <span className="font-mono text-[10px] text-cyan-300/45">
                      L{d.level}
                    </span>
                  </div>
                ))
              )}
            </div>
          </GlassPanel>
        </div>
      </div>
    </PageShell>
  )
}

function Stat({
  label,
  value,
  color,
  sub,
  warn,
}: {
  label: string
  value: number
  color: string
  sub?: string
  warn?: boolean
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: warn ? '#ffaa00' : color }}
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
      {sub && (
        <div
          className="truncate font-mono text-[10px]"
          style={{ color: warn ? '#ffaa00' : 'rgba(56,189,248,0.55)' }}
        >
          {sub}
        </div>
      )}
    </div>
  )
}

function MiniStat({
  label,
  value,
  color,
}: {
  label: string
  value: number
  color: string
}) {
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

function DetailRow({
  k,
  v,
  mono,
}: {
  k: string
  v: React.ReactNode
  mono?: boolean
}) {
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
