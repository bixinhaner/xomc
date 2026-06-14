import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Network, RefreshCcw, Search, ExternalLink } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { PageShell } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useTopoGraph } from '@core/hooks/api/useTopology'
import type { NodeStatus, NodeType, TopoNode } from '@core/types/topology'

// ============================================================
// 拓扑画布（v2 皮肤）— 对照 v1 webcode/pages/topology/TopologyCanvas
//  v1 是「左侧节点列表 + 右侧力导向画布（自研 TopologyCanvas 组件）」。
//  v2 无图形引擎依赖：保留双栏布局——左栏节点列表（服务端筛选 + 客户端关键字 +
//  分页），右栏用绝对定位 + SVG 连线复刻节点画布（坐标取后端返回的 node.x/y）。
//  选中节点高亮联动；带 deviceSn 的节点可一键跳设备详情。
// ============================================================

const LAYOUT_OPTIONS = [
  { label: '力导向', value: 'force' },
  { label: '树形', value: 'tree' },
  { label: '环形', value: 'circular' },
  { label: '分层', value: 'hierarchy' },
]

const NODE_LIMIT_OPTIONS = [100, 500, 1000, 2000]

const NODE_STATUS_META: Record<
  NodeStatus,
  { label: string; variant: 'success' | 'muted' | 'destructive' | 'warning'; dot: string }
> = {
  online: { label: '在线', variant: 'success', dot: 'bg-emerald-500' },
  offline: { label: '离线', variant: 'muted', dot: 'bg-gray-400' },
  alarm: { label: '告警', variant: 'destructive', dot: 'bg-red-500' },
  maintenance: { label: '维护', variant: 'warning', dot: 'bg-yellow-500' },
}

const NODE_TYPE_CLASS: Record<string, string> = {
  eNB: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
  gNB: 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300',
  eGW: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300',
  CPE: 'bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300',
}

const SIDEBAR_PAGE_SIZE = 50

type NodeTypeFilter = '' | NodeType
type NodeStatusFilter = '' | NodeStatus

export default function TopologyCanvas() {
  const navigate = useNavigate()

  const [layoutType, setLayoutType] = useState('force')
  const [nodeTypeFilter, setNodeTypeFilter] = useState<NodeTypeFilter>('')
  const [statusFilter, setStatusFilter] = useState<NodeStatusFilter>('')
  const [limit, setLimit] = useState(500)
  const [search, setSearch] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const {
    data: graphData,
    isLoading,
    isError,
    error,
    isFetching,
    refetch,
  } = useTopoGraph({
    layoutType,
    ...(nodeTypeFilter ? { nodeType: nodeTypeFilter } : {}),
    ...(statusFilter ? { status: statusFilter } : {}),
    limit,
  })

  const nodes = useMemo(() => graphData?.nodes ?? [], [graphData])
  const edges = useMemo(() => graphData?.edges ?? [], [graphData])
  const statistics = graphData?.statistics

  const filteredNodes = useMemo(() => {
    const kw = search.trim()
    if (!kw) return nodes
    return nodes.filter(
      (n) => n.label.includes(kw) || (n.deviceSn ?? '').includes(kw),
    )
  }, [nodes, search])

  const filteredEdges = useMemo(
    () =>
      edges.filter(
        (e) =>
          filteredNodes.some((n) => n.id === e.source) &&
          filteredNodes.some((n) => n.id === e.target),
      ),
    [edges, filteredNodes],
  )

  // 侧栏分页：筛选键变化时归 1（渲染期按依赖同步 state）
  const listKey = `${search}-${nodeTypeFilter}-${statusFilter}-${limit}`
  const [paging, setPaging] = useState({ key: listKey, page: 1 })
  const page = paging.key === listKey ? paging.page : 1
  if (paging.key !== listKey) setPaging({ key: listKey, page: 1 })

  const pagedNodes = useMemo(
    () => filteredNodes.slice((page - 1) * SIDEBAR_PAGE_SIZE, page * SIDEBAR_PAGE_SIZE),
    [filteredNodes, page],
  )
  const totalPages = Math.max(1, Math.ceil(filteredNodes.length / SIDEBAR_PAGE_SIZE))

  const limitReached = nodes.length >= limit

  const selectedNode: TopoNode | null = useMemo(
    () => filteredNodes.find((n) => n.id === selectedId) ?? null,
    [filteredNodes, selectedId],
  )

  return (
    <PageShell
      title="拓扑画布"
      description={`节点 ${statistics?.totalNodes ?? nodes.length} · 连接 ${statistics?.totalEdges ?? edges.length}${limitReached ? ` · 已达上限 ${limit}` : ''}`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-56 pl-9"
              placeholder="搜索节点 / SN"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>

          <Select
            value={nodeTypeFilter || 'all'}
            onValueChange={(v) => setNodeTypeFilter(v === 'all' ? '' : (v as NodeType))}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="节点类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部类型</SelectItem>
              <SelectItem value="eNB">eNB</SelectItem>
              <SelectItem value="gNB">gNB</SelectItem>
              <SelectItem value="CPE">CPE</SelectItem>
              <SelectItem value="eGW">eGW</SelectItem>
            </SelectContent>
          </Select>

          <Select
            value={statusFilter || 'all'}
            onValueChange={(v) => setStatusFilter(v === 'all' ? '' : (v as NodeStatus))}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="节点状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              <SelectItem value="online">在线</SelectItem>
              <SelectItem value="offline">离线</SelectItem>
              <SelectItem value="alarm">告警</SelectItem>
              <SelectItem value="maintenance">维护</SelectItem>
            </SelectContent>
          </Select>

          <Select value={String(limit)} onValueChange={(v) => setLimit(Number(v))}>
            <SelectTrigger className="w-32">
              <SelectValue placeholder="节点限制" />
            </SelectTrigger>
            <SelectContent>
              {NODE_LIMIT_OPTIONS.map((n) => (
                <SelectItem key={n} value={String(n)}>
                  {n} 节点
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw className="mr-1 h-4 w-4" />
              刷新
            </Button>
          </div>
        </div>
      }
    >
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[300px_1fr]">
        {/* 左栏：节点列表 */}
        <div className="flex flex-col rounded-lg border bg-card">
          <div className="border-b px-3 py-2 text-xs text-muted-foreground">
            节点 {filteredNodes.length} · 连接 {filteredEdges.length}
          </div>
          <div className="max-h-[520px] flex-1 overflow-auto">
            {pagedNodes.length === 0 ? (
              <div className="p-6 text-center text-sm text-muted-foreground">
                {nodes.length === 0 ? '暂无节点' : '无匹配节点'}
              </div>
            ) : (
              pagedNodes.map((node) => (
                <button
                  key={node.id}
                  type="button"
                  onClick={() => setSelectedId(node.id)}
                  className={cn(
                    'flex w-full flex-col gap-1 border-b px-3 py-2 text-left transition-colors hover:bg-accent',
                    selectedId === node.id && 'bg-accent',
                  )}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="truncate text-xs font-medium" title={node.label}>
                      {node.label}
                    </span>
                    <Badge variant={NODE_STATUS_META[node.status].variant} className="shrink-0">
                      {NODE_STATUS_META[node.status].label}
                    </Badge>
                  </div>
                  <div className="flex items-center gap-2">
                    <span
                      className={cn(
                        'rounded px-1.5 py-0.5 text-[10px] font-medium',
                        NODE_TYPE_CLASS[node.type] ?? 'bg-muted text-muted-foreground',
                      )}
                    >
                      {node.type}
                    </span>
                    {node.deviceSn && (
                      <span className="truncate font-mono text-[10px] text-muted-foreground">
                        {node.deviceSn}
                      </span>
                    )}
                  </div>
                </button>
              ))
            )}
          </div>
          {totalPages > 1 && (
            <div className="flex items-center justify-between border-t px-3 py-2 text-xs">
              <span className="text-muted-foreground">
                {page} / {totalPages}
              </span>
              <div className="flex gap-1">
                <Button
                  size="sm"
                  variant="outline"
                  disabled={page <= 1}
                  onClick={() => setPaging({ key: listKey, page: Math.max(1, page - 1) })}
                >
                  上一页
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={page >= totalPages}
                  onClick={() => setPaging({ key: listKey, page: Math.min(totalPages, page + 1) })}
                >
                  下一页
                </Button>
              </div>
            </div>
          )}
        </div>

        {/* 右栏：画布 + 布局选择 */}
        <div className="flex flex-col gap-3">
          <div className="flex flex-wrap items-center gap-2">
            <span className="flex items-center gap-1.5 text-sm font-semibold text-muted-foreground">
              <Network className="h-4 w-4" />
              拓扑图
            </span>
            <div className="ml-auto flex items-center gap-1">
              {LAYOUT_OPTIONS.map((opt) => (
                <Button
                  key={opt.value}
                  size="sm"
                  variant={layoutType === opt.value ? 'default' : 'outline'}
                  onClick={() => setLayoutType(opt.value)}
                >
                  {opt.label}
                </Button>
              ))}
            </div>
          </div>

          <div
            className="relative h-[460px] overflow-auto rounded-lg border bg-slate-50 dark:bg-slate-900"
            style={{
              backgroundImage: `
                linear-gradient(to right, rgba(100,116,139,0.12) 1px, transparent 1px),
                linear-gradient(to bottom, rgba(100,116,139,0.12) 1px, transparent 1px)
              `,
              backgroundSize: '40px 40px',
            }}
          >
            {isLoading ? (
              <div className="absolute inset-0 flex items-center justify-center text-sm text-muted-foreground">
                加载拓扑数据...
              </div>
            ) : isError ? (
              <div className="absolute inset-0 flex items-center justify-center text-sm text-destructive">
                加载失败：{error instanceof Error ? error.message : '未知错误'}
              </div>
            ) : filteredNodes.length === 0 ? (
              <div className="absolute inset-0 flex items-center justify-center text-sm text-muted-foreground">
                {nodes.length === 0 ? '暂无拓扑节点数据' : '当前筛选无匹配节点'}
              </div>
            ) : (
              <div
                className="relative"
                style={{
                  minWidth: Math.max(800, ...filteredNodes.map((n) => n.x + 150)),
                  minHeight: Math.max(440, ...filteredNodes.map((n) => n.y + 120)),
                }}
              >
                <svg className="pointer-events-none absolute inset-0 h-full w-full" style={{ zIndex: 1 }}>
                  {filteredEdges.map((edge) => {
                    const s = filteredNodes.find((n) => n.id === edge.source)
                    const t = filteredNodes.find((n) => n.id === edge.target)
                    if (!s || !t) return null
                    const isActive = edge.status === 'active'
                    const isDegraded = edge.status === 'degraded'
                    return (
                      <line
                        key={edge.id}
                        x1={s.x + 50}
                        y1={s.y + 40}
                        x2={t.x + 50}
                        y2={t.y + 40}
                        stroke={isDegraded ? '#f59e0b' : isActive ? '#10b981' : '#94a3b8'}
                        strokeWidth={isActive || isDegraded ? 2 : 1.5}
                        strokeDasharray={isDegraded || !isActive ? '4,4' : undefined}
                        opacity={isActive ? 1 : 0.5}
                      />
                    )
                  })}
                </svg>
                <div className="relative" style={{ zIndex: 2 }}>
                  {filteredNodes.map((node) => (
                    <button
                      type="button"
                      key={node.id}
                      onClick={() => setSelectedId(node.id)}
                      className={cn(
                        'absolute inline-block w-[100px] rounded-lg border-2 bg-white p-2 text-center shadow-sm transition-all hover:shadow-md dark:bg-slate-800',
                        node.status === 'online' && 'border-green-500',
                        node.status === 'offline' && 'border-gray-300 dark:border-gray-600',
                        node.status === 'alarm' && 'border-red-500 bg-red-50 dark:bg-red-950/20',
                        node.status === 'maintenance' && 'border-yellow-500 bg-yellow-50 dark:bg-yellow-950/20',
                        selectedId === node.id && 'ring-2 ring-primary ring-offset-1',
                      )}
                      style={{ left: node.x, top: node.y }}
                    >
                      <div
                        className={cn(
                          'mx-auto mb-1 inline-flex h-9 w-9 items-center justify-center rounded-full text-[11px] font-bold',
                          NODE_TYPE_CLASS[node.type] ?? 'bg-muted text-muted-foreground',
                        )}
                      >
                        {node.type}
                      </div>
                      <div className="truncate text-xs font-medium" title={node.label}>
                        {node.label}
                      </div>
                      <div className="mt-1 flex justify-center">
                        <span
                          className={cn(
                            'h-2 w-2 rounded-full',
                            node.status === 'alarm' && 'animate-pulse',
                            NODE_STATUS_META[node.status].dot,
                          )}
                        />
                      </div>
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* 选中节点详情卡 */}
          {selectedNode && (
            <div className="flex flex-wrap items-center gap-3 rounded-lg border bg-card px-4 py-3 text-sm">
              <span
                className={cn(
                  'rounded px-1.5 py-0.5 text-[11px] font-medium',
                  NODE_TYPE_CLASS[selectedNode.type] ?? 'bg-muted text-muted-foreground',
                )}
              >
                {selectedNode.type}
              </span>
              <span className="font-medium">{selectedNode.label}</span>
              <Badge variant={NODE_STATUS_META[selectedNode.status].variant}>
                {NODE_STATUS_META[selectedNode.status].label}
              </Badge>
              {selectedNode.deviceSn && (
                <span className="font-mono text-xs text-muted-foreground">
                  {selectedNode.deviceSn}
                </span>
              )}
              {selectedNode.deviceSn && (
                <Button
                  size="sm"
                  variant="outline"
                  className="ml-auto"
                  onClick={() =>
                    navigate(`/devices/detail/${encodeURIComponent(selectedNode.deviceSn!)}`)
                  }
                >
                  <ExternalLink className="mr-1 h-3.5 w-3.5" />
                  设备详情
                </Button>
              )}
            </div>
          )}
        </div>
      </div>
    </PageShell>
  )
}
