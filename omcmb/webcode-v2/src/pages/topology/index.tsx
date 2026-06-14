import { useState, useCallback, useMemo, useRef, useEffect } from 'react'
import { RefreshCcw, MapPin, ZoomIn, ZoomOut, Search, Network } from 'lucide-react'

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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  EmptyRow,
  ErrorRow,
  LoadingRow,
  PageShell,
  TableCard,
} from '@/components/layout/PageShell'

import { useSites, useTopoGraph, useDomains } from '@core/hooks/api/useTopology'
import type {
  NodeStatus,
  NodeType,
  SiteStatus,
} from '@core/types/topology'
import { cn } from '@/lib/utils'

// ============================================================
// 拓扑监控页面（v2 皮肤）
// 业务深度对照 v1 webcode/pages/topology：
//  - 拓扑图可视化 + 缩放 + 节点限制（useTopoGraph 服务端筛选 nodeType/status/limit）
//  - 节点类型 / 状态 服务端筛选 + 节点关键字客户端搜索
//  - 统计面板（statistics：节点/边总量 + 类型分布）
//  - 站点列表（useSites 服务端筛选 keyword/domainId/status）+ 域归属解析（useDomains）
//  - loading / 空 / 错误 三态
// ============================================================

type ZoomLevel = number

const ZOOM_LEVELS: ZoomLevel[] = [0.5, 0.75, 1, 1.25, 1.5]
const NODE_LIMIT_OPTIONS = [100, 500, 1000, 2000]

// 站点状态徽标映射
const SITE_STATUS: Record<SiteStatus, { label: string; variant: 'success' | 'warning' | 'muted' }> = {
  active: { label: '正常', variant: 'success' },
  maintenance: { label: '维护', variant: 'warning' },
  inactive: { label: '停用', variant: 'muted' },
}

// 节点状态徽标 / 颜色
const NODE_STATUS_META: Record<NodeStatus, { label: string; variant: 'success' | 'muted' | 'destructive' | 'warning'; dot: string }> = {
  online: { label: '在线', variant: 'success', dot: 'bg-emerald-500' },
  offline: { label: '离线', variant: 'muted', dot: 'bg-gray-400' },
  alarm: { label: '告警', variant: 'destructive', dot: 'bg-red-500' },
  maintenance: { label: '维护', variant: 'warning', dot: 'bg-yellow-500' },
}

// 节点类型彩色徽标 class
const NODE_TYPE_CLASS: Record<string, string> = {
  eNB: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
  gNB: 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300',
  eGW: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300',
  CPE: 'bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300',
}

type NodeTypeFilter = '' | NodeType
type NodeStatusFilter = '' | NodeStatus
type SiteStatusFilter = '' | SiteStatus

export function TopologyPage() {
  // ================== 拓扑图视图状态 ==================
  const [zoom, setZoom] = useState<ZoomLevel>(1)
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const [isDragging, setIsDragging] = useState(false)
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 })
  const canvasRef = useRef<HTMLDivElement>(null)

  // ================== 拓扑图筛选（服务端） ==================
  const [nodeTypeFilter, setNodeTypeFilter] = useState<NodeTypeFilter>('')
  const [nodeStatusFilter, setNodeStatusFilter] = useState<NodeStatusFilter>('')
  const [nodeLimit, setNodeLimit] = useState<number>(500)
  // 节点关键字搜索（客户端）
  const [nodeSearch, setNodeSearch] = useState('')

  // ================== 站点列表筛选（服务端） ==================
  const [siteKeyword, setSiteKeyword] = useState('')
  const [siteDomainId, setSiteDomainId] = useState('')
  const [siteStatus, setSiteStatus] = useState<SiteStatusFilter>('')

  // ================== 数据查询 ==================
  const { data: domainsData } = useDomains()
  const {
    data: sitesData,
    isLoading: isLoadingSites,
    isError: isSitesError,
    error: sitesError,
    isFetching: isFetchingSites,
    refetch: refetchSites,
  } = useSites({
    ...(siteKeyword.trim() ? { keyword: siteKeyword.trim() } : {}),
    ...(siteDomainId ? { domainId: siteDomainId } : {}),
    ...(siteStatus ? { status: siteStatus } : {}),
  })
  const {
    data: graphData,
    isLoading: isLoadingGraph,
    isError: isGraphError,
    error: graphError,
    refetch: refetchGraph,
  } = useTopoGraph({
    ...(nodeTypeFilter ? { nodeType: nodeTypeFilter } : {}),
    ...(nodeStatusFilter ? { status: nodeStatusFilter } : {}),
    limit: nodeLimit,
  })

  const sites = sitesData?.items ?? []
  const allNodes = useMemo(() => graphData?.nodes ?? [], [graphData])
  const allEdges = useMemo(() => graphData?.edges ?? [], [graphData])
  const statistics = graphData?.statistics

  // 域 id → 名称（站点列表展示域归属 + 站点筛选下拉）
  const domains = domainsData ?? []
  const domainNameMap = useMemo(() => {
    const m: Record<string, string> = {}
    for (const d of domains) m[d.id] = d.name
    return m
  }, [domains])

  // 节点关键字客户端过滤（按 label / deviceSn）
  const filteredNodes = useMemo(() => {
    const kw = nodeSearch.trim()
    if (!kw) return allNodes
    return allNodes.filter(
      (n) => n.label.includes(kw) || (n.deviceSn ?? '').includes(kw),
    )
  }, [allNodes, nodeSearch])

  const filteredEdges = useMemo(
    () =>
      allEdges.filter(
        (e) =>
          filteredNodes.some((n) => n.id === e.source) &&
          filteredNodes.some((n) => n.id === e.target),
      ),
    [allEdges, filteredNodes],
  )

  // ================== 统计（优先用服务端 statistics，回退本地计算） ==================
  const stats = useMemo(() => {
    if (statistics) {
      return {
        totalNodes: statistics.totalNodes,
        onlineCount: statistics.onlineNodes,
        offlineCount: statistics.offlineNodes,
        alarmCount: statistics.alarmNodes,
        maintenanceCount: statistics.maintenanceNodes,
        totalEdges: statistics.totalEdges,
        activeEdges: statistics.activeEdges,
        nodeTypeCounts: statistics.nodeTypeCounts ?? {},
      }
    }
    const counts: Record<string, number> = {}
    for (const n of allNodes) counts[n.type] = (counts[n.type] ?? 0) + 1
    return {
      totalNodes: allNodes.length,
      onlineCount: allNodes.filter((n) => n.status === 'online').length,
      offlineCount: allNodes.filter((n) => n.status === 'offline').length,
      alarmCount: allNodes.filter((n) => n.status === 'alarm').length,
      maintenanceCount: allNodes.filter((n) => n.status === 'maintenance').length,
      totalEdges: allEdges.length,
      activeEdges: allEdges.filter((e) => e.status === 'active').length,
      nodeTypeCounts: counts,
    }
  }, [statistics, allNodes, allEdges])

  const limitReached = allNodes.length >= nodeLimit

  // ================== 缩放 / 拖拽交互 ==================
  const currentZoomIndex = ZOOM_LEVELS.indexOf(zoom)
  const canZoomIn = currentZoomIndex < ZOOM_LEVELS.length - 1
  const canZoomOut = currentZoomIndex > 0

  const handleZoomIn = useCallback(() => {
    if (canZoomIn) setZoom(ZOOM_LEVELS[currentZoomIndex + 1])
  }, [canZoomIn, currentZoomIndex])

  const handleZoomOut = useCallback(() => {
    if (canZoomOut) setZoom(ZOOM_LEVELS[currentZoomIndex - 1])
  }, [canZoomOut, currentZoomIndex])

  const handleResetView = useCallback(() => {
    setZoom(1)
    setPan({ x: 0, y: 0 })
  }, [])

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      if (e.button === 0 && e.target === canvasRef.current) {
        setIsDragging(true)
        setDragStart({ x: e.clientX - pan.x, y: e.clientY - pan.y })
      }
    },
    [pan],
  )

  const handleMouseMove = useCallback(
    (e: React.MouseEvent) => {
      if (isDragging) setPan({ x: e.clientX - dragStart.x, y: e.clientY - dragStart.y })
    },
    [isDragging, dragStart],
  )

  const handleMouseUp = useCallback(() => setIsDragging(false), [])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const handleWheel = (e: WheelEvent) => {
      if (e.deltaY < 0 && canZoomIn) handleZoomIn()
      else if (e.deltaY > 0 && canZoomOut) handleZoomOut()
    }
    canvas.addEventListener('wheel', handleWheel, { passive: false })
    return () => canvas.removeEventListener('wheel', handleWheel)
  }, [canZoomIn, canZoomOut, handleZoomIn, handleZoomOut])

  const refreshAll = useCallback(() => {
    void refetchGraph()
    void refetchSites()
  }, [refetchGraph, refetchSites])

  const cols = ['站点', '所属域', '地址', '设备数', '状态', '坐标']
  const descriptionText = `站点列表 · 拓扑图可视化 ${
    stats.alarmCount > 0 ? `· ${stats.alarmCount} 告警` : ''
  }`

  return (
    <PageShell
      title="拓扑监控"
      description={descriptionText}
      isFetching={isFetchingSites || isLoadingGraph}
      toolbar={
        <div className="flex w-full items-center gap-2">
          <div className="hidden items-center gap-3 text-sm text-muted-foreground md:flex">
            <span>节点: {stats.onlineCount}/{stats.totalNodes}</span>
            <span>连接: {stats.activeEdges}/{stats.totalEdges}</span>
          </div>
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={refreshAll}>
              <RefreshCcw className="mr-1 h-4 w-4" />
              刷新
            </Button>
          </div>
        </div>
      }
    >
      {/* ================== 统计面板 ================== */}
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-6">
        <Stat label="节点总数" value={stats.totalNodes} sub={limitReached ? `上限 ${nodeLimit}` : `限 ${nodeLimit}`} tone={limitReached ? 'amber' : 'default'} />
        <Stat label="在线" value={stats.onlineCount} tone="emerald" />
        <Stat label="离线" value={stats.offlineCount} tone="muted" />
        <Stat label="告警" value={stats.alarmCount} tone="amber" />
        <Stat label="连接数" value={stats.totalEdges} sub={`活跃 ${stats.activeEdges}`} />
        <Stat label="站点" value={sitesData?.total ?? sites.length} />
      </div>

      {/* 节点类型分布 */}
      {Object.keys(stats.nodeTypeCounts).length > 0 && (
        <div className="mb-4 flex flex-wrap items-center gap-2 rounded-lg border bg-card px-4 py-2.5">
          <span className="text-xs text-muted-foreground">类型分布:</span>
          {Object.entries(stats.nodeTypeCounts).map(([type, count]) => (
            <span
              key={type}
              className={cn(
                'inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium',
                NODE_TYPE_CLASS[type] ?? 'bg-muted text-muted-foreground',
              )}
            >
              {type}
              <span className="tabular-nums">{count}</span>
            </span>
          ))}
        </div>
      )}

      {/* ================== 拓扑图筛选工具条 ================== */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-64 pl-9"
            placeholder="搜索节点名称 / SN"
            value={nodeSearch}
            onChange={(e) => setNodeSearch(e.target.value)}
          />
        </div>

        <Select
          value={nodeTypeFilter || 'all'}
          onValueChange={(v) => setNodeTypeFilter(v === 'all' ? '' : (v as NodeType))}
        >
          <SelectTrigger className="w-36">
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
          value={nodeStatusFilter || 'all'}
          onValueChange={(v) => setNodeStatusFilter(v === 'all' ? '' : (v as NodeStatus))}
        >
          <SelectTrigger className="w-36">
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

        <Select value={String(nodeLimit)} onValueChange={(v) => setNodeLimit(Number(v))}>
          <SelectTrigger className="w-36">
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
      </div>

      {/* ================== 拓扑图可视化区域 ================== */}
      <div className="mb-6">
        <div className="mb-3 flex items-center justify-between">
          <h3 className="flex items-center gap-1.5 text-sm font-semibold text-muted-foreground">
            <Network className="h-4 w-4" />
            拓扑图
          </h3>
          <div className="flex items-center gap-3">
            {/* 图例 */}
            {allNodes.length > 0 && (
              <div className="hidden items-center gap-4 text-xs text-muted-foreground sm:flex">
                <span className="flex items-center gap-1">
                  <span className="h-2 w-2 rounded-full bg-green-500" />
                  在线 {stats.onlineCount}
                </span>
                <span className="flex items-center gap-1">
                  <span className="h-2 w-2 rounded-full bg-gray-400" />
                  离线 {stats.offlineCount}
                </span>
                {stats.alarmCount > 0 && (
                  <span className="flex items-center gap-1">
                    <span className="h-2 w-2 rounded-full bg-red-500" />
                    告警 {stats.alarmCount}
                  </span>
                )}
              </div>
            )}
            {/* 缩放控制 */}
            <div className="flex items-center gap-1">
              <Button variant="ghost" size="sm" onClick={handleZoomOut} disabled={!canZoomOut}>
                <ZoomOut className="h-4 w-4" />
              </Button>
              <span className="w-10 text-center text-xs text-muted-foreground">
                {Math.round(zoom * 100)}%
              </span>
              <Button variant="ghost" size="sm" onClick={handleZoomIn} disabled={!canZoomIn}>
                <ZoomIn className="h-4 w-4" />
              </Button>
              <Button variant="ghost" size="sm" onClick={handleResetView}>
                重置
              </Button>
            </div>
          </div>
        </div>

        {/* 画布 */}
        <div
          ref={canvasRef}
          className="relative select-none overflow-hidden rounded-lg border bg-slate-50 dark:bg-slate-900"
          style={{ height: 400 }}
          onMouseDown={handleMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp}
          onMouseLeave={handleMouseUp}
        >
          {isLoadingGraph ? (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="text-center">
                <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
                <p className="mt-2 text-sm text-muted-foreground">加载拓扑数据...</p>
              </div>
            </div>
          ) : isGraphError ? (
            <div className="absolute inset-0 flex items-center justify-center text-sm text-destructive">
              拓扑加载失败：{graphError instanceof Error ? graphError.message : '未知错误'}
            </div>
          ) : filteredNodes.length === 0 ? (
            <div className="absolute inset-0 flex items-center justify-center text-sm text-muted-foreground">
              {allNodes.length === 0 ? '暂无拓扑节点数据' : '当前筛选无匹配节点'}
            </div>
          ) : (
            <div
              className="absolute origin-top-left will-change-transform"
              style={{
                transform: `scale(${zoom}) translate(${pan.x}px, ${pan.y}px)`,
                minWidth: Math.max(800, ...filteredNodes.map((n) => n.x + 150)),
                minHeight: Math.max(380, ...filteredNodes.map((n) => n.y + 120)),
              }}
            >
              {/* 网格背景 */}
              <div
                className="pointer-events-none absolute inset-0 opacity-10"
                style={{
                  backgroundImage: `
                    linear-gradient(to right, #64748b 1px, transparent 1px),
                    linear-gradient(to bottom, #64748b 1px, transparent 1px)
                  `,
                  backgroundSize: '40px 40px',
                }}
              />

              {/* 连接线层 */}
              <svg className="pointer-events-none absolute inset-0 h-full w-full" style={{ zIndex: 1 }}>
                {filteredEdges.map((edge) => {
                  const source = filteredNodes.find((n) => n.id === edge.source)
                  const target = filteredNodes.find((n) => n.id === edge.target)
                  if (!source || !target) return null
                  const sourceX = source.x + 50
                  const sourceY = source.y + 40
                  const targetX = target.x + 50
                  const targetY = target.y + 40
                  const isActive = edge.status === 'active'
                  const isDegraded = edge.status === 'degraded'
                  return (
                    <g key={edge.id}>
                      <line
                        x1={sourceX}
                        y1={sourceY}
                        x2={targetX}
                        y2={targetY}
                        stroke={isDegraded ? '#f59e0b' : isActive ? '#10b981' : '#94a3b8'}
                        strokeWidth={isDegraded ? 2 : isActive ? 2 : 1.5}
                        strokeDasharray={isDegraded || !isActive ? '4,4' : undefined}
                        opacity={isActive ? 1 : 0.5}
                      />
                      {edge.label && (
                        <text
                          x={(sourceX + targetX) / 2}
                          y={(sourceY + targetY) / 2 - 5}
                          fontSize="10"
                          textAnchor="middle"
                          fill="#64748b"
                          className="dark:fill-slate-400"
                        >
                          {edge.label}
                        </text>
                      )}
                    </g>
                  )
                })}
              </svg>

              {/* 节点层 */}
              <div className="relative" style={{ zIndex: 2 }}>
                {filteredNodes.map((node) => (
                  <div
                    key={node.id}
                    className={cn(
                      'absolute inline-block cursor-default rounded-lg border-2 bg-white shadow-sm transition-all hover:shadow-md dark:bg-slate-800',
                      node.status === 'online' && 'border-green-500',
                      node.status === 'offline' && 'border-gray-300 dark:border-gray-600',
                      node.status === 'alarm' && 'border-red-500 bg-red-50 dark:bg-red-950/20',
                      node.status === 'maintenance' && 'border-yellow-500 bg-yellow-50 dark:bg-yellow-950/20',
                    )}
                    style={{ left: node.x, top: node.y, width: 100 }}
                  >
                    <div className="p-2 text-center">
                      <div
                        className={cn(
                          'mb-1 inline-flex h-10 w-10 items-center justify-center rounded-full text-xs font-bold',
                          NODE_TYPE_CLASS[node.type] ?? 'bg-muted text-muted-foreground',
                        )}
                      >
                        {node.type}
                      </div>
                      <div className="truncate text-xs font-medium" title={node.label}>
                        {node.label}
                      </div>
                      {node.deviceSn && (
                        <div className="truncate text-[10px] text-muted-foreground" title={node.deviceSn}>
                          {node.deviceSn.slice(-8)}
                        </div>
                      )}
                      <div className="mt-1 flex justify-center">
                        <div
                          className={cn(
                            'h-2 w-2 rounded-full',
                            node.status === 'alarm' ? 'animate-pulse' : '',
                            NODE_STATUS_META[node.status].dot,
                          )}
                        />
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* ================== 站点列表筛选工具条 ================== */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-64 pl-9"
            placeholder="搜索站点名称 / 地址"
            value={siteKeyword}
            onChange={(e) => setSiteKeyword(e.target.value)}
          />
        </div>

        <Select
          value={siteDomainId || 'all'}
          onValueChange={(v) => setSiteDomainId(v === 'all' ? '' : v)}
        >
          <SelectTrigger className="w-44">
            <SelectValue placeholder="所属域" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部域</SelectItem>
            {domains.map((d) => (
              <SelectItem key={d.id} value={d.id}>
                {d.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select
          value={siteStatus || 'all'}
          onValueChange={(v) => setSiteStatus(v === 'all' ? '' : (v as SiteStatus))}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="站点状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="active">正常</SelectItem>
            <SelectItem value="maintenance">维护</SelectItem>
            <SelectItem value="inactive">停用</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* ================== 站点列表表格 ================== */}
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoadingSites ? (
              <LoadingRow colSpan={cols.length} />
            ) : isSitesError ? (
              <ErrorRow colSpan={cols.length} error={sitesError} />
            ) : sites.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无站点</EmptyRow>
            ) : (
              sites.map((s) => {
                const statusMeta = SITE_STATUS[s.status] ?? SITE_STATUS.inactive
                return (
                  <TableRow key={s.id}>
                    <TableCell className="font-medium">{s.name}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {domainNameMap[s.domainId] ?? s.domainId ?? '—'}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">{s.address}</TableCell>
                    <TableCell className="tabular-nums">{s.deviceCount}</TableCell>
                    <TableCell>
                      <Badge variant={statusMeta.variant}>{statusMeta.label}</Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      <span className="inline-flex items-center gap-1">
                        <MapPin className="h-3 w-3" />
                        {s.latitude != null && s.longitude != null
                          ? `${s.latitude.toFixed(3)}, ${s.longitude.toFixed(3)}`
                          : '—'}
                      </span>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>
    </PageShell>
  )
}

// ============================================================
// 模块内小组件：统计卡片
// ============================================================
function Stat({
  label,
  value,
  sub,
  tone = 'default',
}: {
  label: string
  value: number
  sub?: string
  tone?: 'default' | 'emerald' | 'amber' | 'muted'
}) {
  const toneClass = {
    default: 'text-foreground',
    emerald: 'text-emerald-600 dark:text-emerald-400',
    amber: 'text-amber-600 dark:text-amber-400',
    muted: 'text-muted-foreground',
  }[tone]

  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>{value}</div>
      {sub && <div className="mt-0.5 text-[11px] text-muted-foreground">{sub}</div>}
    </div>
  )
}
