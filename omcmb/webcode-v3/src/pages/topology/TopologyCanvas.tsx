import { useMemo, useState } from 'react'
import { Loader2, RefreshCcw, Search, Layers, X, Workflow } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { useTopoGraph } from '@core/hooks/api/useTopology'
import type {
  TopoNode,
  TopoEdge,
  NodeType,
  NodeStatus,
} from '@core/types/topology'

const NODE_STATUS_COLOR: Record<string, string> = {
  online: '#00ff88',
  offline: '#525a78',
  alarm: '#ff2d6f',
  maintenance: '#ffaa00',
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
  active: 'rgba(0,255,136,0.5)',
  inactive: 'rgba(82,90,120,0.42)',
  degraded: 'rgba(255,170,0,0.55)',
}

const LAYOUT_OPTIONS = ['force', 'tree', 'circular', 'hierarchy'] as const
const LAYOUT_LABEL: Record<string, string> = {
  force: 'GRID',
  tree: 'TREE',
  circular: 'RING',
  hierarchy: 'TIER',
}
const LIMIT_OPTIONS = [100, 500, 1000, 2000]

const CANVAS_W = 1000
const CANVAS_H = 660

interface NodePos {
  node: TopoNode
  x: number
  y: number
}

/**
 * 网格布局：把节点铺成自适应网格，与主拓扑页的「同心轨道」区分。
 * tree/hierarchy 用 type 分层（每种 type 一行），circular 用单环。
 */
function layoutNodes(nodes: TopoNode[], mode: string): NodePos[] {
  if (nodes.length === 0) return []
  const pad = 60
  const w = CANVAS_W - pad * 2
  const h = CANVAS_H - pad * 2

  if (mode === 'circular') {
    const cx = CANVAS_W / 2
    const cy = CANVAS_H / 2
    const r = Math.min(w, h) / 2 - 20
    return nodes.map((node, i) => {
      const a = (i / nodes.length) * Math.PI * 2 - Math.PI / 2
      return { node, x: cx + Math.cos(a) * r, y: cy + Math.sin(a) * r }
    })
  }

  if (mode === 'tree' || mode === 'hierarchy') {
    // 按 type 分层，每种类型占一行
    const byType = new Map<string, TopoNode[]>()
    for (const n of nodes) {
      const arr = byType.get(n.type) ?? []
      arr.push(n)
      byType.set(n.type, arr)
    }
    const types = [...byType.keys()]
    const out: NodePos[] = []
    types.forEach((type, row) => {
      const rowNodes = byType.get(type) ?? []
      const y = pad + (types.length === 1 ? h / 2 : (row / (types.length - 1)) * h)
      rowNodes.forEach((node, col) => {
        const x =
          pad + (rowNodes.length === 1 ? w / 2 : (col / (rowNodes.length - 1)) * w)
        out.push({ node, x, y })
      })
    })
    return out
  }

  // force → 自适应网格
  const cols = Math.ceil(Math.sqrt(nodes.length))
  const rows = Math.ceil(nodes.length / cols)
  return nodes.map((node, i) => {
    const c = i % cols
    const r = Math.floor(i / cols)
    const x = pad + (cols === 1 ? w / 2 : (c / (cols - 1)) * w)
    const y = pad + (rows === 1 ? h / 2 : (r / (rows - 1)) * h)
    return { node, x, y }
  })
}

export default function TopologyCanvas() {
  const [keyword, setKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<NodeStatus | ''>('')
  const [typeFilter, setTypeFilter] = useState<NodeType | ''>('')
  const [layoutType, setLayoutType] = useState<string>('force')
  const [limit, setLimit] = useState(500)
  const [selected, setSelected] = useState<TopoNode | null>(null)

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

  const nodes = useMemo<TopoNode[]>(() => graph?.nodes ?? [], [graph])
  const edges = useMemo<TopoEdge[]>(() => graph?.edges ?? [], [graph])
  const stats = graph?.statistics

  const filteredNodes = useMemo(() => {
    const kw = keyword.trim()
    if (!kw) return nodes
    return nodes.filter(
      (n) => n.label.includes(kw) || (n.deviceSn ?? '').includes(kw)
    )
  }, [nodes, keyword])

  const visibleIds = useMemo(
    () => new Set(filteredNodes.map((n) => n.id)),
    [filteredNodes]
  )
  const filteredEdges = useMemo(
    () => edges.filter((e) => visibleIds.has(e.source) && visibleIds.has(e.target)),
    [edges, visibleIds]
  )

  const positions = useMemo(
    () => layoutNodes(filteredNodes, layoutType),
    [filteredNodes, layoutType]
  )
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
          x1: a.x,
          y1: a.y,
          x2: b.x,
          y2: b.y,
          color: EDGE_STATUS_COLOR[e.status] ?? EDGE_STATUS_COLOR.inactive,
        })
      }
    })
    return out
  }, [filteredEdges, posById])

  const atLimit = nodes.length >= limit

  return (
    <PageShell
      code="F06"
      title="TOPOLOGY CANVAS · 拓扑画布"
      subtitle="GRAPH ENGINE · 多布局节点 / 链路渲染"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-56 pl-9"
              placeholder="节点标签 / 设备 SN"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          {(['', 'online', 'offline', 'alarm', 'maintenance'] as const).map((s) => (
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
              {s || 'ALL'}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => void refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
        <Stat
          label="NODES · 节点"
          value={stats?.totalNodes ?? nodes.length}
          color="#00f0ff"
          sub={`已加载 ${nodes.length}/${limit}${atLimit ? ' · 上限' : ''}`}
          warn={atLimit}
        />
        <Stat label="ONLINE · 在线" value={stats?.onlineNodes ?? 0} color="#00ff88" />
        <Stat label="OFFLINE · 离线" value={stats?.offlineNodes ?? 0} color="#525a78" />
        <Stat label="ALARM · 告警" value={stats?.alarmNodes ?? 0} color="#ff2d6f" />
        <Stat label="MAINT · 维护" value={stats?.maintenanceNodes ?? 0} color="#ffaa00" />
        <Stat
          label="EDGES · 链路"
          value={stats?.totalEdges ?? edges.length}
          color="#a855f7"
          sub={stats ? `活动 ${stats.activeEdges} · 降级 ${stats.degradedEdges}` : undefined}
        />
      </div>

      <div className="grid grid-cols-1 gap-3 xl:grid-cols-[1fr_300px]">
        <GlassPanel
          title="CANVAS · 节点画布"
          meta={`${filteredNodes.length} NODES · ${filteredEdges.length} EDGES`}
          strong
          className="min-h-[560px]"
        >
          {/* 布局 + 上限 工具条 */}
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

          <div className="relative h-[500px]">
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
                  <filter id="canvasGlow">
                    <feGaussianBlur stdDeviation="2" />
                    <feMerge>
                      <feMergeNode />
                      <feMergeNode in="SourceGraphic" />
                    </feMerge>
                  </filter>
                </defs>

                {edgeLines.map((e) => (
                  <line
                    key={e.key}
                    x1={e.x1}
                    y1={e.y1}
                    x2={e.x2}
                    y2={e.y2}
                    stroke={e.color}
                    strokeWidth={1}
                  />
                ))}

                {positions.map((p) => {
                  const color = NODE_STATUS_COLOR[p.node.status] ?? '#525a78'
                  const typeColor = NODE_TYPE_COLOR[p.node.type] ?? '#6b86b6'
                  const isSel = selected?.id === p.node.id
                  return (
                    <g
                      key={p.node.id}
                      className="cursor-pointer"
                      onClick={() => setSelected(p.node)}
                    >
                      <circle
                        cx={p.x}
                        cy={p.y}
                        r={isSel ? 14 : 10}
                        fill="none"
                        stroke={isSel ? '#fff' : typeColor}
                        strokeWidth={1}
                        opacity={isSel ? 0.9 : 0.45}
                      />
                      <circle
                        cx={p.x}
                        cy={p.y}
                        r={5}
                        fill={color}
                        filter="url(#canvasGlow)"
                      />
                    </g>
                  )
                })}
              </svg>
            )}
          </div>

          {/* 类型分布 */}
          <div className="flex flex-wrap items-center gap-x-3 gap-y-2 border-t border-cyan-500/10 px-3 py-2">
            <Workflow className="size-3.5 text-cyan-300/55" />
            <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/55">
              TYPES
            </span>
            {stats?.nodeTypeCounts &&
              Object.entries(stats.nodeTypeCounts).map(([type, count]) => (
                <button
                  key={type}
                  type="button"
                  onClick={() =>
                    setTypeFilter((prev) => (prev === type ? '' : (type as NodeType)))
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
          </div>
        </GlassPanel>

        {/* 右栏：选中详情 + 节点清单 */}
        <div className="flex flex-col gap-3">
          {selected && (
            <GlassPanel title="NODE · 节点详情" meta="SELECTED">
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
                  {selected.label}
                </div>
                <DetailRow
                  k="状态"
                  v={
                    <span style={{ color: NODE_STATUS_COLOR[selected.status] ?? '#6b86b6' }}>
                      {selected.status}
                    </span>
                  }
                />
                <DetailRow
                  k="类型"
                  v={
                    <span style={{ color: NODE_TYPE_COLOR[selected.type] ?? '#d8e9ff' }}>
                      {selected.type}
                    </span>
                  }
                />
                <DetailRow k="设备 SN" v={selected.deviceSn ?? '—'} mono />
                <DetailRow k="站点" v={selected.siteId ?? '—'} mono />
                <DetailRow k="域" v={selected.domainId ?? '—'} mono />
              </div>
            </GlassPanel>
          )}

          <GlassPanel title="NODES · 节点清单" meta={`${filteredNodes.length}`}>
            <div className="max-h-[420px] overflow-auto">
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
                  const isSel = selected?.id === n.id
                  return (
                    <button
                      key={n.id}
                      type="button"
                      onClick={() => setSelected(n)}
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
