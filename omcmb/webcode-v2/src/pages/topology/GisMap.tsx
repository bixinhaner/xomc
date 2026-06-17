import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { MapPin, RefreshCcw, Search, Crosshair, Plus, Minus, Maximize2 } from 'lucide-react'

// OpenLayers —— 经 workspace hoist 可直接 import（参考 v1 webcode/components/GISMap/useOLMap）
// 注意：ol 的 Map 类与全局 Map 同名，别名为 OlMap 避免遮蔽 ES Map（样式缓存用到）
import OlMap from 'ol/Map'
import View from 'ol/View'
import TileLayer from 'ol/layer/Tile'
import VectorLayer from 'ol/layer/Vector'
import VectorSource from 'ol/source/Vector'
import Cluster from 'ol/source/Cluster'
import XYZ from 'ol/source/XYZ'
import Feature from 'ol/Feature'
import Point from 'ol/geom/Point'
import { fromLonLat } from 'ol/proj'
import { boundingExtent } from 'ol/extent'
import { defaults as defaultControls } from 'ol/control'
import { Style, Stroke, Circle as CircleStyle, Fill, Text } from 'ol/style'
import type { FeatureLike } from 'ol/Feature'
import 'ol/ol.css'

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
  LoadingRow,
  PageShell,
  TableCard,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useDomainTree,
  useMapDevicesGeo,
  useMapStats,
  useMapDeviceSearch,
} from '@core/hooks/api/useTopology'
import type { DeviceStatus, DeviceGeo, MapFilterParams } from '@core/types/map'

// ============================================================
// GIS 地图视图（v2 皮肤）— 对照 v1 webcode/pages/topology/GISMapView
//  v1 用 OpenLayers 渲染真实地图瓦片 + 设备打点 + 聚合 + 飞行定位；
//  v2 此处对齐到「交互式 OpenLayers 地图 + 设备标记/聚合」：
//   - useDomainTree   ：左侧设备组（域）下拉筛选源
//   - useMapDevicesGeo：按域/状态/关键字拉取设备经纬度列表（服务端筛选）→ 地图打点
//   - useMapStats     ：顶部状态统计（在线激活 / 在线未激活 / 离线 / 告警）
//   - useMapDeviceSearch：关键字 >=2 字时服务端搜索定位单设备
//  地图标记/列表行均可点击跳转设备详情（/devices/detail/:sn）。
//  底图瓦片本地缺失会 404 属环境项；矢量标记照常渲染，不影响打点与交互。
// ============================================================

const DEVICE_STATUS_META: Record<
  DeviceStatus,
  { label: string; variant: 'success' | 'warning' | 'muted'; dot: string; color: string }
> = {
  onlineActive: { label: '在线激活', variant: 'success', dot: 'bg-emerald-500', color: '#22c55e' },
  onlineInactive: { label: '在线未激活', variant: 'warning', dot: 'bg-yellow-500', color: '#eab308' },
  offline: { label: '离线', variant: 'muted', dot: 'bg-gray-400', color: '#ef4444' },
}

type StatusFilter = '' | DeviceStatus

// ============================================================
// OpenLayers 常量（就地内联，避免跨包相对路径引用 v1 的 constants.ts）
// ============================================================

// 在线 OSM 瓦片（本地无离线瓦片时降级用；404 仅影响底图，矢量标记照常渲染）
const OSM_TILE_URL = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png'
// 赞比亚区域中心 [lng, lat] 与默认缩放（所有设备经纬度均值附近）
const DEFAULT_CENTER: [number, number] = [28.0, -15.0]
const DEFAULT_ZOOM = 6
const MIN_ZOOM = 2
const MAX_ZOOM = 18
// 聚合像素距离；高缩放时禁用聚合让设备分散
const CLUSTER_DISTANCE = 42
const DISABLE_CLUSTER_ZOOM = 14
// 聚合圆颜色（蓝），与 v2 主题主色调一致
const CLUSTER_FILL = '#2563eb'
const CLUSTER_STROKE = 'rgba(37, 99, 235, 0.25)'
const MARKER_STROKE = '#ffffff'
const ALARM_BADGE_FILL = '#ef4444'

// 域树（Domain[] 带 children）扁平化为 id→name，供下拉显示层级名
function flattenDomains(
  nodes: { id: string; name: string; children?: unknown }[] | undefined,
  depth = 0,
  out: { id: string; name: string; depth: number }[] = [],
): { id: string; name: string; depth: number }[] {
  for (const n of nodes ?? []) {
    out.push({ id: n.id, name: n.name, depth })
    const children = (n as { children?: typeof nodes }).children
    if (children && children.length) flattenDomains(children, depth + 1, out)
  }
  return out
}

// 设备状态色（feature 样式用）
function statusColor(status: DeviceStatus): string {
  return DEVICE_STATUS_META[status]?.color ?? DEVICE_STATUS_META.offline.color
}

// ============================================================
// OpenLayers 样式函数
//  - 单设备：状态色实心圆 + 白边；有告警时叠加红色角标
//  - 聚合：蓝色圆 + 数量文字，半径随聚合规模增大
// ============================================================
function buildMarkerStyleFn() {
  // 缓存单设备样式，避免每帧重建（key=status:alarm>0）
  const cache = new Map<string, Style[]>()

  return (feature: FeatureLike): Style | Style[] => {
    const members = feature.get('features') as Feature[] | undefined
    // 非聚合源兜底：feature 本身即设备
    const list = members ?? [feature as Feature]
    const count = list.length

    if (count > 1) {
      // —— 聚合样式 ——
      const radius = Math.min(34, 14 + Math.log2(count) * 5)
      return new Style({
        image: new CircleStyle({
          radius,
          fill: new Fill({ color: CLUSTER_FILL }),
          stroke: new Stroke({ color: CLUSTER_STROKE, width: 6 }),
        }),
        text: new Text({
          text: String(count),
          fill: new Fill({ color: '#ffffff' }),
          font: 'bold 12px sans-serif',
        }),
      })
    }

    // —— 单设备样式 ——
    const device = list[0].get('device') as DeviceGeo | undefined
    const status = device?.status ?? 'offline'
    const hasAlarm = (device?.alarmCount ?? 0) > 0
    const cacheKey = `${status}:${hasAlarm ? '1' : '0'}`
    const cached = cache.get(cacheKey)
    if (cached) return cached

    const styles: Style[] = [
      new Style({
        image: new CircleStyle({
          radius: 7,
          fill: new Fill({ color: statusColor(status) }),
          stroke: new Stroke({ color: MARKER_STROKE, width: 2 }),
        }),
      }),
    ]
    if (hasAlarm) {
      // 告警角标：右上角小红点
      styles.push(
        new Style({
          image: new CircleStyle({
            radius: 4,
            fill: new Fill({ color: ALARM_BADGE_FILL }),
            stroke: new Stroke({ color: MARKER_STROKE, width: 1 }),
            displacement: [7, 7],
          }),
        }),
      )
    }
    cache.set(cacheKey, styles)
    return styles
  }
}

export default function GisMap() {
  const navigate = useNavigate()

  const [groupId, setGroupId] = useState('')
  const [status, setStatus] = useState<StatusFilter>('')
  const [keyword, setKeyword] = useState('')

  const { data: domainTree } = useDomainTree()
  const domainOptions = useMemo(() => flattenDomains(domainTree), [domainTree])

  const filterParams = useMemo<MapFilterParams>(
    () => ({
      ...(groupId ? { groupIds: [groupId] } : {}),
      ...(status ? { status: [status] } : {}),
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      // 一次性取尽量多的点用于地图打点（赞比亚区域约 550 台）
      pageSize: 5000,
    }),
    [groupId, status, keyword],
  )

  const {
    data: geoData,
    isLoading,
    isFetching,
    refetch,
  } = useMapDevicesGeo(filterParams)

  const { data: statsData } = useMapStats(
    groupId ? { groupIds: [groupId] } : undefined,
  )

  // 关键字 >=2 字时触发服务端设备搜索（v1 的「节点查找」），命中可一键跳详情/定位
  const { data: searchResults } = useMapDeviceSearch(keyword.trim())

  const devices = useMemo(() => geoData?.items ?? [], [geoData])
  const total = geoData?.total ?? devices.length

  const stats = statsData?.statusCount
  const alarmCount = statsData?.alarmCount ?? 0

  // 仅保留有有效经纬度的设备用于地图打点
  const geoDevices = useMemo(
    () => devices.filter((d) => d.latitude != null && d.longitude != null),
    [devices],
  )

  // ========== OpenLayers 地图实例 ==========
  const mapElRef = useRef<HTMLDivElement | null>(null)
  const mapRef = useRef<OlMap | null>(null)
  const sourceRef = useRef<VectorSource | null>(null)
  const clusterRef = useRef<Cluster | null>(null)
  // 是否已根据数据自动 fit 过一次视图（避免每次刷新都强制回正，打断用户平移）
  const fittedRef = useRef(false)

  // 把当前 geoDevices 喂进 VectorSource（构造 Point feature）
  const renderFeatures = useCallback(() => {
    const source = sourceRef.current
    if (!source) return
    source.clear()
    const features = geoDevices.map((d) => {
      const f = new Feature({
        geometry: new Point(fromLonLat([d.longitude as number, d.latitude as number])),
      })
      f.setId(d.id)
      f.set('device', d)
      return f
    })
    source.addFeatures(features)
  }, [geoDevices])

  // 根据数据 extent 适配视图（仅首批数据落地时执行一次）
  const fitToData = useCallback(() => {
    const map = mapRef.current
    if (!map || geoDevices.length === 0) return
    const coords = geoDevices.map((d) =>
      fromLonLat([d.longitude as number, d.latitude as number]),
    )
    if (coords.length === 1) {
      map.getView().animate({ center: coords[0], zoom: 12, duration: 600 })
      return
    }
    const extent = boundingExtent(coords)
    map.getView().fit(extent, {
      padding: [60, 60, 60, 60],
      maxZoom: 13,
      duration: 600,
    })
  }, [geoDevices])

  // 初始化地图（仅一次）
  useEffect(() => {
    if (!mapElRef.current || mapRef.current) return

    const source = new VectorSource()
    const cluster = new Cluster({ source, distance: CLUSTER_DISTANCE })
    const vectorLayer = new VectorLayer({
      source: cluster,
      style: buildMarkerStyleFn(),
      zIndex: 10,
    })

    const map = new OlMap({
      target: mapElRef.current,
      layers: [
        new TileLayer({
          source: new XYZ({
            url: OSM_TILE_URL,
            crossOrigin: 'anonymous',
            maxZoom: MAX_ZOOM,
          }),
          zIndex: 0,
        }),
        vectorLayer,
      ],
      view: new View({
        center: fromLonLat(DEFAULT_CENTER),
        zoom: DEFAULT_ZOOM,
        minZoom: MIN_ZOOM,
        maxZoom: MAX_ZOOM,
      }),
      // 关掉默认控件，缩放由自定义 shadcn 按钮驱动
      controls: defaultControls({ zoom: false, attribution: false, rotate: false }),
    })

    mapRef.current = map
    sourceRef.current = source
    clusterRef.current = cluster

    // 高缩放时禁用聚合，让设备点分散开
    map.getView().on('change:resolution', () => {
      const z = map.getView().getZoom() ?? DEFAULT_ZOOM
      const target = z >= DISABLE_CLUSTER_ZOOM ? 0 : CLUSTER_DISTANCE
      if (cluster.getDistance() !== target) cluster.setDistance(target)
    })

    // 点击：单设备 → 跳详情；聚合 → 放大展开；并切换鼠标指针
    map.on('click', (evt) => {
      const hit = map.forEachFeatureAtPixel(evt.pixel, (f) => f) as Feature | undefined
      if (!hit) return
      const members = hit.get('features') as Feature[] | undefined
      if (members && members.length > 1) {
        // 聚合：放大一级并以聚合中心为目标
        const geom = hit.getGeometry()
        const center = geom instanceof Point ? geom.getCoordinates() : evt.coordinate
        const z = map.getView().getZoom() ?? DEFAULT_ZOOM
        map.getView().animate({ center, zoom: Math.min(z + 2, MAX_ZOOM), duration: 300 })
        return
      }
      const leaf = members?.[0] ?? hit
      const device = leaf.get('device') as DeviceGeo | undefined
      if (device?.sn) {
        navigate(`/device/detail/${encodeURIComponent(device.sn)}`)
      }
    })

    map.on('pointermove', (evt) => {
      const hit = map.hasFeatureAtPixel(evt.pixel)
      map.getTargetElement().style.cursor = hit ? 'pointer' : ''
    })

    return () => {
      map.setTarget(undefined)
      mapRef.current = null
      sourceRef.current = null
      clusterRef.current = null
      fittedRef.current = false
    }
    // 仅初始化一次；navigate 引用稳定
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // 设备数据变化时重绘标记，并在首批数据落地时自动 fit
  useEffect(() => {
    renderFeatures()
    if (!fittedRef.current && geoDevices.length > 0) {
      fittedRef.current = true
      fitToData()
    }
  }, [renderFeatures, fitToData, geoDevices.length])

  // ========== 地图控制（自定义按钮） ==========
  const zoomBy = useCallback((delta: number) => {
    const map = mapRef.current
    if (!map) return
    const view = map.getView()
    const z = view.getZoom() ?? DEFAULT_ZOOM
    view.animate({ zoom: Math.max(MIN_ZOOM, Math.min(MAX_ZOOM, z + delta)), duration: 200 })
  }, [])

  // 搜索命中后定位到该设备（有坐标才飞行；否则仍可跳详情）
  const locateDevice = useCallback(
    (lng: number | null, lat: number | null) => {
      const map = mapRef.current
      if (!map || lng == null || lat == null) return
      map.getView().animate({ center: fromLonLat([lng, lat]), zoom: 14, duration: 600 })
    },
    [],
  )

  const cols = ['设备名称', 'SN', '所属域', '状态', '坐标', '告警']

  return (
    <PageShell
      title="GIS 地图"
      description={`设备地理分布 · 共 ${total} 台${
        geoDevices.length < devices.length
          ? `（${geoDevices.length} 台有坐标）`
          : ''
      }${alarmCount > 0 ? ` · ${alarmCount} 告警` : ''}`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full items-center gap-2">
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw className="mr-1 h-4 w-4" />
              刷新
            </Button>
          </div>
        </div>
      }
    >
      {/* 状态统计卡 */}
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="设备总数" value={statsData?.total ?? total} />
        <Stat label="在线激活" value={stats?.onlineActive ?? 0} tone="emerald" />
        <Stat label="在线未激活" value={stats?.onlineInactive ?? 0} tone="amber" />
        <Stat label="离线" value={stats?.offline ?? 0} tone="muted" />
      </div>

      {/* 筛选工具条 */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-64 pl-9"
            placeholder="搜索设备名称 / SN"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>

        <Select
          value={groupId || 'all'}
          onValueChange={(v) => setGroupId(v === 'all' ? '' : v)}
        >
          <SelectTrigger className="w-48">
            <SelectValue placeholder="所属域" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部域</SelectItem>
            {domainOptions.map((d) => (
              <SelectItem key={d.id} value={d.id}>
                {'　'.repeat(d.depth)}
                {d.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select
          value={status || 'all'}
          onValueChange={(v) => setStatus(v === 'all' ? '' : (v as DeviceStatus))}
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder="设备状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="onlineActive">在线激活</SelectItem>
            <SelectItem value="onlineInactive">在线未激活</SelectItem>
            <SelectItem value="offline">离线</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* 搜索命中（>=2 字触发服务端搜索，复刻 v1 节点查找定位） */}
      {keyword.trim().length >= 2 && (searchResults?.length ?? 0) > 0 && (
        <div className="mb-4 rounded-lg border bg-card p-3">
          <div className="mb-2 flex items-center gap-1.5 text-xs text-muted-foreground">
            <Crosshair className="h-3.5 w-3.5" />
            搜索命中 {searchResults?.length} 项 · 点击在地图定位，点 SN 跳详情
          </div>
          <div className="flex flex-wrap gap-2">
            {(searchResults ?? []).slice(0, 12).map((r) => (
              <button
                key={r.id}
                type="button"
                onClick={() => locateDevice(r.longitude, r.latitude)}
                className="inline-flex items-center gap-1.5 rounded-md border bg-background px-2.5 py-1 text-xs hover:bg-accent"
              >
                <span
                  className={cn(
                    'inline-block size-1.5 rounded-full',
                    DEVICE_STATUS_META[r.status]?.dot ?? 'bg-gray-400',
                  )}
                />
                <span className="font-medium">{r.name}</span>
                <span
                  role="link"
                  tabIndex={0}
                  onClick={(e) => {
                    e.stopPropagation()
                    navigate(`/device/detail/${encodeURIComponent(r.sn)}`)
                  }}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.stopPropagation()
                      navigate(`/device/detail/${encodeURIComponent(r.sn)}`)
                    }
                  }}
                  className="font-mono text-muted-foreground hover:text-primary hover:underline"
                >
                  {r.sn}
                </span>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* ================= 交互式地图（主视图，对齐 v1 OpenLayers 打点/聚合） ================= */}
      <div className="relative mb-4 h-[480px] overflow-hidden rounded-lg border bg-muted/30">
        <div ref={mapElRef} className="h-full w-full" />

        {/* 地图加载/空态遮罩 */}
        {isLoading ? (
          <div className="pointer-events-none absolute inset-0 flex items-center justify-center text-sm text-muted-foreground">
            地图加载中…
          </div>
        ) : geoDevices.length === 0 ? (
          <div className="pointer-events-none absolute inset-0 flex items-center justify-center text-sm text-muted-foreground">
            暂无可定位设备（无有效经纬度）
          </div>
        ) : null}

        {/* 缩放 / 回正控制（shadcn Button 叠加在地图右上角） */}
        <div className="absolute right-3 top-3 flex flex-col gap-1.5">
          <Button
            variant="secondary"
            size="icon"
            className="size-8 shadow-sm"
            onClick={() => zoomBy(1)}
            aria-label="放大"
          >
            <Plus className="h-4 w-4" />
          </Button>
          <Button
            variant="secondary"
            size="icon"
            className="size-8 shadow-sm"
            onClick={() => zoomBy(-1)}
            aria-label="缩小"
          >
            <Minus className="h-4 w-4" />
          </Button>
          <Button
            variant="secondary"
            size="icon"
            className="size-8 shadow-sm"
            onClick={fitToData}
            aria-label="适配全部设备"
          >
            <Maximize2 className="h-4 w-4" />
          </Button>
        </div>

        {/* 图例（左下角） */}
        <div className="absolute bottom-3 left-3 flex flex-wrap items-center gap-3 rounded-md border bg-card/90 px-3 py-1.5 text-xs shadow-sm backdrop-blur">
          {(Object.keys(DEVICE_STATUS_META) as DeviceStatus[]).map((s) => (
            <span key={s} className="inline-flex items-center gap-1.5">
              <span
                className="inline-block size-2.5 rounded-full ring-1 ring-white"
                style={{ background: DEVICE_STATUS_META[s].color }}
              />
              {DEVICE_STATUS_META[s].label}
            </span>
          ))}
          <span className="inline-flex items-center gap-1.5">
            <span
              className="inline-block size-3 rounded-full"
              style={{ background: CLUSTER_FILL }}
            />
            聚合
          </span>
        </div>
      </div>

      {/* 设备地理列表（辅助列表，地图为主） */}
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
            ) : devices.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无设备地理数据</EmptyRow>
            ) : (
              devices.map((d) => {
                const meta = DEVICE_STATUS_META[d.status] ?? DEVICE_STATUS_META.offline
                const canLocate = d.latitude != null && d.longitude != null
                return (
                  <TableRow
                    key={d.id}
                    className="cursor-pointer"
                    onClick={() => navigate(`/device/detail/${encodeURIComponent(d.sn)}`)}
                  >
                    <TableCell className="font-medium text-primary hover:underline">
                      {d.device_name || d.name}
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      {d.sn}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {d.groupName ?? d.groupId ?? '—'}
                    </TableCell>
                    <TableCell>
                      <Badge variant={meta.variant}>
                        <span className={cn('mr-1 inline-block size-1.5 rounded-full', meta.dot)} />
                        {meta.label}
                      </Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      <button
                        type="button"
                        disabled={!canLocate}
                        onClick={(e) => {
                          e.stopPropagation()
                          locateDevice(d.longitude, d.latitude)
                        }}
                        className={cn(
                          'inline-flex items-center gap-1',
                          canLocate ? 'hover:text-primary' : 'cursor-default opacity-60',
                        )}
                        title={canLocate ? '在地图上定位' : '无有效坐标'}
                      >
                        <MapPin className="h-3 w-3" />
                        {canLocate
                          ? `${d.latitude!.toFixed(3)}, ${d.longitude!.toFixed(3)}`
                          : '—'}
                      </button>
                    </TableCell>
                    <TableCell className="tabular-nums">
                      {d.alarmCount && d.alarmCount > 0 ? (
                        <Badge variant="destructive">{d.alarmCount}</Badge>
                      ) : (
                        <span className="text-muted-foreground">0</span>
                      )}
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

function Stat({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: number
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
    </div>
  )
}
