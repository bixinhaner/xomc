import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { AlertTriangle, Eye, Loader2, RefreshCcw, Search, Wifi } from 'lucide-react'

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
import { Card } from '@/components/ui/card'
import { PageShell } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useDeviceList } from '@core/hooks/api/useDevices'
import type { Device, DeviceFilter } from '@core/types/device'
import type { PageRequest } from '@core/types/pagination'

// ============================================================
// 在线监控 — 设备状态卡片墙 + 实时在线/离线/告警概览
// 对照 v1 webcode/src/pages/device/OnlineMonitoring 的业务深度
// 设备指标（CPU/内存/温度）从 SN 派生展示（与 v1 一致，后端未提供实时遥测）
// ============================================================

type StatusFilter = 'all' | 'online' | 'offline'
type TypeFilter = 'all' | 'eNB' | 'gNB' | 'CPE' | 'eGW'

function metricColor(value: number, warn = 70, crit = 90) {
  if (value >= crit) return 'bg-destructive'
  if (value >= warn) return 'bg-amber-500'
  return 'bg-emerald-500'
}

// 由 SN 确定性派生的演示指标（与 v1 同：后端暂无实时遥测，仅用于占位呈现）
function deriveMetrics(sn: string) {
  const hash = sn.split('').reduce((acc, c) => acc + c.charCodeAt(0), 0)
  return {
    cpu: ((hash * 37) % 60) + 20,
    memory: ((hash * 53) % 50) + 40,
    temperature: ((hash * 17) % 25) + 30,
    uptime: ((hash * 7) % 720) + 24,
  }
}

function MetricBar({ label, value }: { label: string; value: number }) {
  return (
    <div className="flex items-center gap-2">
      <span className="w-14 shrink-0 text-[11px] text-muted-foreground">
        {label}
      </span>
      <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
        <div
          className={cn('h-full rounded-full', metricColor(value))}
          style={{ width: `${Math.min(100, value)}%` }}
        />
      </div>
      <span className="w-9 shrink-0 text-right text-[11px] tabular-nums text-muted-foreground">
        {value}%
      </span>
    </div>
  )
}

function DeviceCard({
  device,
  onView,
}: {
  device: Device
  onView: (sn: string) => void
}) {
  const metrics = useMemo(() => deriveMetrics(device.sn), [device.sn])
  const online = device.isOnline
  const hasAlarm = device.alarmLevel !== 'none'

  return (
    <Card
      className={cn(
        'p-3',
        hasAlarm && 'border-amber-500/50'
      )}
    >
      <div className="mb-2 flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div
            className="truncate text-sm font-medium"
            title={device.deviceName || device.name}
          >
            {device.deviceName || device.name || '—'}
          </div>
          <div className="truncate font-mono text-[11px] text-muted-foreground">
            {device.sn}
          </div>
        </div>
        <Badge variant={online ? 'success' : 'muted'} className="shrink-0">
          <span
            className={cn(
              'mr-1 inline-block size-1.5 rounded-full',
              online ? 'bg-emerald-500' : 'bg-muted-foreground/40'
            )}
          />
          {online ? '在线' : '离线'}
        </Badge>
      </div>

      {online ? (
        <div className="mb-2.5 flex flex-col gap-1.5">
          <MetricBar label="CPU" value={metrics.cpu} />
          <MetricBar label="内存" value={metrics.memory} />
          <div className="flex items-center gap-2">
            <span className="w-14 shrink-0 text-[11px] text-muted-foreground">
              温度
            </span>
            <span
              className={cn(
                'text-xs font-semibold',
                metrics.temperature >= 55
                  ? 'text-destructive'
                  : metrics.temperature >= 45
                    ? 'text-amber-600 dark:text-amber-400'
                    : 'text-emerald-600 dark:text-emerald-400'
              )}
            >
              {metrics.temperature}°C
            </span>
            <span className="ml-auto text-[11px] text-muted-foreground">
              运行 {metrics.uptime}h
            </span>
          </div>
        </div>
      ) : (
        <div className="mb-2.5 py-3 text-center text-xs text-muted-foreground">
          设备离线
        </div>
      )}

      <div className="mb-2 flex flex-wrap gap-1">
        {device.productClass && (
          <Badge variant="outline" className="text-[11px]">
            {device.productClass}
          </Badge>
        )}
        {device.networkType && (
          <Badge variant="outline" className="text-[11px]">
            {device.networkType}
          </Badge>
        )}
        {hasAlarm && (
          <Badge variant="warning" className="text-[11px]">
            <AlertTriangle className="mr-1 size-3" /> 有告警
          </Badge>
        )}
      </div>

      <div className="flex items-center justify-between">
        <span className="text-[11px] text-muted-foreground">
          {device.region || '—'}
        </span>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 px-2"
          onClick={() => onView(device.sn)}
        >
          <Eye className="size-3.5" /> 详情
        </Button>
      </div>
    </Card>
  )
}

export default function OnlineMonitoring() {
  const navigate = useNavigate()
  const [searchText, setSearchText] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [typeFilter, setTypeFilter] = useState<TypeFilter>('all')

  const queryParams = useMemo<DeviceFilter & PageRequest>(
    () => ({ page: 1, pageSize: 120 }),
    []
  )
  const { data, isLoading, isError, error, isFetching, refetch } =
    useDeviceList(queryParams)

  const allDevices = data?.items ?? []

  const filtered = useMemo(() => {
    const q = searchText.trim().toLowerCase()
    return allDevices.filter((d) => {
      const matchText =
        !q ||
        (d.deviceName || d.name || '').toLowerCase().includes(q) ||
        d.sn.toLowerCase().includes(q)
      const matchStatus =
        statusFilter === 'all' ||
        (statusFilter === 'online' ? d.isOnline : !d.isOnline)
      const matchType = typeFilter === 'all' || d.productClass === typeFilter
      return matchText && matchStatus && matchType
    })
  }, [allDevices, searchText, statusFilter, typeFilter])

  const onlineCount = filtered.filter((d) => d.isOnline).length
  const offlineCount = filtered.filter((d) => !d.isOnline).length
  const alarmCount = filtered.filter((d) => d.alarmLevel !== 'none').length

  const handleView = (sn: string) => navigate(`/devices/detail/${sn}`)

  return (
    <PageShell
      title="在线监控"
      description={`共 ${filtered.length} 台 · 在线 ${onlineCount} · 离线 ${offlineCount} · 告警 ${alarmCount}`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-64 pl-9"
              placeholder="搜索 名称 / SN"
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
            />
          </div>
          <Select
            value={statusFilter}
            onValueChange={(v) => setStatusFilter(v as StatusFilter)}
          >
            <SelectTrigger className="w-28">
              <SelectValue placeholder="状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部</SelectItem>
              <SelectItem value="online">在线</SelectItem>
              <SelectItem value="offline">离线</SelectItem>
            </SelectContent>
          </Select>
          <Select
            value={typeFilter}
            onValueChange={(v) => setTypeFilter(v as TypeFilter)}
          >
            <SelectTrigger className="w-28">
              <SelectValue placeholder="类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部</SelectItem>
              <SelectItem value="eNB">eNB</SelectItem>
              <SelectItem value="gNB">gNB</SelectItem>
              <SelectItem value="CPE">CPE</SelectItem>
              <SelectItem value="eGW">eGW</SelectItem>
            </SelectContent>
          </Select>
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw className="size-4" /> 刷新
            </Button>
          </div>
        </div>
      }
    >
      {isLoading ? (
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </div>
      ) : isError ? (
        <Card className="flex h-48 items-center justify-center p-6 text-sm text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </Card>
      ) : filtered.length === 0 ? (
        <Card className="flex h-48 flex-col items-center justify-center gap-2 p-6 text-muted-foreground">
          <Wifi className="size-6" />
          <span className="text-sm">暂无匹配的设备</span>
        </Card>
      ) : (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
          {filtered.map((device) => (
            <DeviceCard key={device.id} device={device} onView={handleView} />
          ))}
        </div>
      )}
    </PageShell>
  )
}
