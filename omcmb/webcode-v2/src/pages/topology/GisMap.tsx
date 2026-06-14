import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { MapPin, RefreshCcw, Search, Crosshair } from 'lucide-react'

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
import type { DeviceStatus, MapFilterParams } from '@core/types/map'

// ============================================================
// GIS 地图视图（v2 皮肤）— 对照 v1 webcode/pages/topology/GISMapView
//  v1 用 OpenLayers 渲染真实地图瓦片 + 设备打点 + 聚合 + 飞行定位；
//  v2 无 openlayers/echarts 依赖，改用「域树筛选 + 统计卡 + 设备地理列表」近似：
//   - useDomainTree   ：左侧设备组（域）多选筛选源
//   - useMapDevicesGeo：按域/状态/关键字拉取设备经纬度列表（服务端筛选）
//   - useMapStats     ：顶部状态统计（在线激活 / 在线未激活 / 离线 / 告警）
//   - useMapDeviceSearch：关键字 >=2 字时服务端搜索定位单设备
//  设备行可点击跳转设备详情（/devices/detail/:sn），复刻 v1「在地图上定位/查看」。
// ============================================================

const DEVICE_STATUS_META: Record<
  DeviceStatus,
  { label: string; variant: 'success' | 'warning' | 'muted'; dot: string }
> = {
  onlineActive: { label: '在线激活', variant: 'success', dot: 'bg-emerald-500' },
  onlineInactive: { label: '在线未激活', variant: 'warning', dot: 'bg-yellow-500' },
  offline: { label: '离线', variant: 'muted', dot: 'bg-gray-400' },
}

type StatusFilter = '' | DeviceStatus

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

  // 关键字 >=2 字时触发服务端设备搜索（v1 的「节点查找」），命中可一键跳详情
  const { data: searchResults } = useMapDeviceSearch(keyword.trim())

  const devices = geoData?.items ?? []
  const total = geoData?.total ?? devices.length

  const stats = statsData?.statusCount
  const alarmCount = statsData?.alarmCount ?? 0

  const cols = ['设备名称', 'SN', '所属域', '状态', '坐标', '告警']

  return (
    <PageShell
      title="GIS 地图"
      description={`设备地理分布 · 共 ${total} 台${alarmCount > 0 ? ` · ${alarmCount} 告警` : ''}`}
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
            搜索命中 {searchResults?.length} 项
          </div>
          <div className="flex flex-wrap gap-2">
            {(searchResults ?? []).slice(0, 12).map((r) => (
              <button
                key={r.id}
                type="button"
                onClick={() => navigate(`/devices/detail/${encodeURIComponent(r.sn)}`)}
                className="inline-flex items-center gap-1.5 rounded-md border bg-background px-2.5 py-1 text-xs hover:bg-accent"
              >
                <span
                  className={cn(
                    'inline-block size-1.5 rounded-full',
                    DEVICE_STATUS_META[r.status]?.dot ?? 'bg-gray-400',
                  )}
                />
                <span className="font-medium">{r.name}</span>
                <span className="font-mono text-muted-foreground">{r.sn}</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* 设备地理列表 */}
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
                return (
                  <TableRow
                    key={d.id}
                    className="cursor-pointer"
                    onClick={() => navigate(`/devices/detail/${encodeURIComponent(d.sn)}`)}
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
                      <span className="inline-flex items-center gap-1">
                        <MapPin className="h-3 w-3" />
                        {d.latitude != null && d.longitude != null
                          ? `${d.latitude.toFixed(3)}, ${d.longitude.toFixed(3)}`
                          : '—'}
                      </span>
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
