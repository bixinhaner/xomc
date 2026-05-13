import { useState, useCallback, useMemo } from 'react'
import { RefreshCcw, MapPin, ZoomIn, ZoomOut } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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

import { useSites, useTopoGraph } from '@core/hooks/api/useTopology'
import { cn } from '@/lib/utils'

// ============================================================
// 拓扑监控页面 - 带缩放功能
// ============================================================

type ZoomLevel = number // 改为 number 类型，避免字面量类型问题

const ZOOM_LEVELS: ZoomLevel[] = [0.5, 0.75, 1, 1.25, 1.5]

export function TopologyPage() {
  // ================== 状态管理 ==================
  const [zoom, setZoom] = useState<ZoomLevel>(1)
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const [isDragging, setIsDragging] = useState(false)
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 })

  const canvasRef = useRef<HTMLDivElement>(null)

  // ================== 数据查询 ==================
  const { data: sitesData, isLoading, isError, error, isFetching, refetch } = useSites()
  const { data: graphData, isLoading: isLoadingGraph } = useTopoGraph()

  const rows = sitesData?.items ?? []
  const nodes = graphData?.nodes ?? []
  const edges = graphData?.edges ?? []

  // ================== 计算属性 ==================
  const stats = useMemo(() => ({
    totalNodes: nodes.length,
    onlineCount: nodes.filter((n) => n.status === 'online').length,
    offlineCount: nodes.filter((n) => n.status === 'offline').length,
    alarmCount: nodes.filter((n) => n.status === 'alarm').length,
    totalEdges: edges.length,
    activeEdges: edges.filter((e) => e.status === 'active').length,
  }), [nodes, edges])

  const currentZoomIndex = ZOOM_LEVELS.indexOf(zoom)
  const canZoomIn = currentZoomIndex < ZOOM_LEVELS.length - 1
  const canZoomOut = currentZoomIndex > 0

  // ================== 交互处理 ==================
  const handleZoomIn = useCallback(() => {
    if (canZoomIn) {
      setZoom(ZOOM_LEVELS[currentZoomIndex + 1])
    }
  }, [canZoomIn, currentZoomIndex])

  const handleZoomOut = useCallback(() => {
    if (canZoomOut) {
      setZoom(ZOOM_LEVELS[currentZoomIndex - 1])
    }
  }, [canZoomOut, currentZoomIndex])

  const handleResetView = useCallback(() => {
    setZoom(1)
    setPan({ x: 0, y: 0 })
  }, [])

  // 拖拽处理
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    if (e.button === 0 && e.target === canvasRef.current) {
      setIsDragging(true)
      setDragStart({ x: e.clientX - pan.x, y: e.clientY - pan.y })
    }
  }, [pan])

  const handleMouseMove = useCallback((e: React.MouseEvent) => {
    if (isDragging) {
      setPan({ x: e.clientX - dragStart.x, y: e.clientY - dragStart.y })
    }
  }, [isDragging, dragStart])

  const handleMouseUp = useCallback(() => {
    setIsDragging(false)
  }, [])

  // 滚轮缩放 - 移除 preventDefault，改用 useEffect 添加非 passive 监听器
  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return

    const handleWheel = (e: WheelEvent) => {
      if (e.deltaY < 0 && canZoomIn) {
        handleZoomIn()
      } else if (e.deltaY > 0 && canZoomOut) {
        handleZoomOut()
      }
    }

    // 添加非 passive 的事件监听器
    canvas.addEventListener('wheel', handleWheel, { passive: false })

    return () => {
      canvas.removeEventListener('wheel', handleWheel)
    }
  }, [canZoomIn, canZoomOut, handleZoomIn, handleZoomOut])

  const cols = ['站点', '地址', '设备数', '状态', '坐标']
  const descriptionText = `站点列表 · 拓扑图可视化 ${stats.alarmCount > 0 ? `· ${stats.alarmCount} 告警` : ''}`

  return (
    <PageShell
      title="拓扑监控"
      description={descriptionText}
      isFetching={isFetching || isLoadingGraph}
      toolbar={
        <div className="flex items-center gap-2">
          {/* 统计信息 */}
          <div className="hidden md:flex items-center gap-3 text-sm text-muted-foreground mr-4">
            <span>节点: {stats.onlineCount}/{stats.totalNodes}</span>
            <span>连接: {stats.activeEdges}/{stats.totalEdges}</span>
          </div>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw className="w-4 h-4 mr-1" />
            刷新
          </Button>
        </div>
      }
    >
      {/* ================== 拓扑图可视化区域 ================== */}
      {(nodes.length > 0 || isLoadingGraph) && (
        <div className="mb-6">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-muted-foreground">拓扑图</h3>
            <div className="flex items-center gap-3">
              {/* 图例 */}
              {nodes.length > 0 && (
                <div className="hidden sm:flex items-center gap-4 text-xs text-muted-foreground">
                  <span className="flex items-center gap-1">
                    <span className="w-2 h-2 rounded-full bg-green-500"></span>
                    在线 {stats.onlineCount}
                  </span>
                  <span className="flex items-center gap-1">
                    <span className="w-2 h-2 rounded-full bg-gray-400"></span>
                    离线 {stats.offlineCount}
                  </span>
                  {stats.alarmCount > 0 && (
                    <span className="flex items-center gap-1">
                      <span className="w-2 h-2 rounded-full bg-red-500"></span>
                      告警 {stats.alarmCount}
                    </span>
                  )}
                </div>
              )}
              {/* 缩放控制 */}
              <div className="flex items-center gap-1">
                <Button variant="ghost" size="sm" onClick={handleZoomOut} disabled={!canZoomOut}>
                  <ZoomOut className="w-4 h-4" />
                </Button>
                <span className="text-xs text-muted-foreground w-10 text-center">
                  {Math.round(zoom * 100)}%
                </span>
                <Button variant="ghost" size="sm" onClick={handleZoomIn} disabled={!canZoomIn}>
                  <ZoomIn className="w-4 h-4" />
                </Button>
                <Button variant="ghost" size="sm" onClick={handleResetView}>
                  重置
                </Button>
              </div>
            </div>
          </div>

          {/* 拓扑图画布 */}
          <div
            ref={canvasRef}
            className="relative bg-slate-50 dark:bg-slate-900 rounded-lg border overflow-hidden select-none"
            style={{ height: 400 }}
            onMouseDown={handleMouseDown}
            onMouseMove={handleMouseMove}
            onMouseUp={handleMouseUp}
            onMouseLeave={handleMouseUp}
          >
            {isLoadingGraph ? (
              <div className="absolute inset-0 flex items-center justify-center">
                <div className="text-center">
                  <div className="inline-block w-8 h-8 border-4 border-primary border-t-transparent rounded-full animate-spin" />
                  <p className="mt-2 text-sm text-muted-foreground">加载拓扑数据...</p>
                </div>
              </div>
            ) : nodes.length === 0 ? (
              <div className="absolute inset-0 flex items-center justify-center text-muted-foreground text-sm">
                暂无拓扑节点数据
              </div>
            ) : (
              <div
                className="absolute origin-top-left will-change-transform"
                style={{
                  transform: `scale(${zoom}) translate(${pan.x}px, ${pan.y}px)`,
                  // 确保 SVG 坐标系正确
                  minWidth: Math.max(800, ...nodes.map(n => n.x + 150)),
                  minHeight: Math.max(380, ...nodes.map(n => n.y + 120)),
                }}
              >
                {/* 网格背景 */}
                <div
                  className="absolute inset-0 pointer-events-none opacity-10"
                  style={{
                    backgroundImage: `
                      linear-gradient(to right, #64748b 1px, transparent 1px),
                      linear-gradient(to bottom, #64748b 1px, transparent 1px)
                    `,
                    backgroundSize: '40px 40px',
                  }}
                />

                {/* SVG 连接线层 */}
                <svg className="absolute inset-0 w-full h-full pointer-events-none" style={{ zIndex: 1 }}>
                  {edges.map((edge) => {
                    const source = nodes.find((n) => n.id === edge.source)
                    const target = nodes.find((n) => n.id === edge.target)
                    if (!source || !target) return null

                    // 节点中心点 (节点宽100px, 高约80px)
                    const sourceX = source.x + 50
                    const sourceY = source.y + 40
                    const targetX = target.x + 50
                    const targetY = target.y + 40

                    const isActive = edge.status === 'active'
                    const isDegraded = edge.status === 'degraded'

                    return (
                      <g key={edge.id}>
                        {/* 连接线 */}
                        <line
                          x1={sourceX}
                          y1={sourceY}
                          x2={targetX}
                          y2={targetY}
                          stroke={
                            isDegraded
                              ? '#f59e0b'
                              : isActive
                                ? '#10b981'
                                : '#94a3b8'
                          }
                          strokeWidth={isDegraded ? 2 : isActive ? 2 : 1.5}
                          strokeDasharray={isDegraded || !isActive ? '4,4' : undefined}
                          opacity={isActive ? 1 : 0.5}
                        />
                        {/* 连接标签 */}
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
                  {nodes.map((node) => (
                    <div
                      key={node.id}
                      className={cn(
                        "absolute inline-block rounded-lg border-2 bg-white dark:bg-slate-800 shadow-sm transition-all hover:shadow-md cursor-default",
                        node.status === 'online' && "border-green-500",
                        node.status === 'offline' && "border-gray-300 dark:border-gray-600",
                        node.status === 'alarm' && "border-red-500 bg-red-50 dark:bg-red-950/20",
                        node.status === 'maintenance' && "border-yellow-500 bg-yellow-50 dark:bg-yellow-950/20",
                      )}
                      style={{
                        left: node.x,
                        top: node.y,
                        width: 100,
                      }}
                    >
                      <div className="p-2 text-center">
                        {/* 节点类型标签 */}
                        <div
                          className={cn(
                            "inline-flex items-center justify-center w-10 h-10 rounded-full text-xs font-bold mb-1",
                            node.type === 'eNB' &&
                              "bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300",
                            node.type === 'gNB' &&
                              "bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300",
                            node.type === 'eGW' &&
                              "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300",
                            node.type === 'CPE' &&
                              "bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300",
                          )}
                        >
                          {node.type}
                        </div>
                        {/* 节点名称 */}
                        <div className="text-xs font-medium truncate" title={node.label}>
                          {node.label}
                        </div>
                        {/* 设备序列号 */}
                        {node.deviceSn && (
                          <div className="text-[10px] text-muted-foreground truncate" title={node.deviceSn}>
                            {node.deviceSn.slice(-8)}
                          </div>
                        )}
                        {/* 状态指示灯 */}
                        <div className="flex justify-center mt-1">
                          <div
                            className={cn(
                              "w-2 h-2 rounded-full",
                              node.status === 'online' && "bg-green-500",
                              node.status === 'offline' && "bg-gray-400",
                              node.status === 'alarm' && "bg-red-500 animate-pulse",
                              node.status === 'maintenance' && "bg-yellow-500",
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
      )}

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
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无站点</EmptyRow>
            ) : (
              rows.map((s) => (
                <TableRow key={s.id}>
                  <TableCell className="font-medium">{s.name}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">{s.address}</TableCell>
                  <TableCell className="tabular-nums">{s.deviceCount}</TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        s.status === 'active'
                          ? 'default'
                          : s.status === 'maintenance'
                            ? 'secondary'
                            : 'outline'
                      }
                    >
                      {s.status === 'active' ? '正常' : s.status === 'maintenance' ? '维护' : '停用'}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    <span className="inline-flex items-center gap-1">
                      <MapPin className="w-3 h-3" />
                      {s.latitude.toFixed(3)}, {s.longitude.toFixed(3)}
                    </span>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>
    </PageShell>
  )
}
