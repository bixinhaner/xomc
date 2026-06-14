import { useMemo } from 'react'
import {
  Boxes,
  CloudOff,
  Globe,
  Loader2,
  Wifi,
} from 'lucide-react'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { PageShell } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useDeviceList } from '@core/hooks/api/useDevices'
import type { Device, DeviceFilter } from '@core/types/device'
import type { PageRequest } from '@core/types/pagination'

// ============================================================
// 资源统计 — 设备总量/在线率 + 按制式/厂商/区域/产品分布
// 对照 v1 webcode/src/pages/device/ResourceStatistics 的业务深度
// v1 用 echarts + 写死示例数；本页用真实 useDeviceList 聚合（无 echarts，CSS 近似）
// ============================================================

const DIST_PALETTE = [
  'bg-primary',
  'bg-emerald-500',
  'bg-amber-500',
  'bg-violet-500',
  'bg-sky-500',
  'bg-rose-500',
  'bg-teal-500',
  'bg-orange-500',
]

interface DistItem {
  name: string
  value: number
}

// 统计某字段取值分布（空值归入「未知」），按数量降序
function tally(devices: Device[], pick: (d: Device) => string): DistItem[] {
  const map = new Map<string, number>()
  for (const d of devices) {
    const key = (pick(d) || '').trim() || '未知'
    map.set(key, (map.get(key) ?? 0) + 1)
  }
  return Array.from(map.entries())
    .map(([name, value]) => ({ name, value }))
    .sort((a, b) => b.value - a.value)
}

function StatCard({
  label,
  value,
  icon,
  tone = 'default',
}: {
  label: string
  value: string | number
  icon: React.ReactNode
  tone?: 'default' | 'emerald' | 'muted' | 'violet'
}) {
  const toneClass = {
    default: 'text-primary bg-primary/10',
    emerald: 'text-emerald-600 dark:text-emerald-400 bg-emerald-500/10',
    muted: 'text-muted-foreground bg-muted',
    violet: 'text-violet-600 dark:text-violet-400 bg-violet-500/10',
  }[tone]

  return (
    <Card>
      <CardContent className="flex items-center gap-3 p-4">
        <div
          className={cn(
            'flex size-10 shrink-0 items-center justify-center rounded-lg',
            toneClass
          )}
        >
          {icon}
        </div>
        <div>
          <div className="text-xs text-muted-foreground">{label}</div>
          <div className="mt-0.5 text-2xl font-semibold tabular-nums">
            {value}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function DistributionCard({
  title,
  items,
  total,
}: {
  title: string
  items: DistItem[]
  total: number
}) {
  return (
    <Card>
      <CardHeader className="p-4 pb-2">
        <CardTitle className="text-base font-medium">{title}</CardTitle>
      </CardHeader>
      <CardContent className="p-4 pt-2">
        {items.length === 0 ? (
          <div className="py-6 text-center text-sm text-muted-foreground">
            暂无数据
          </div>
        ) : (
          <div className="flex flex-col gap-2.5">
            {items.map((item, idx) => {
              const pct = total > 0 ? Math.round((item.value / total) * 100) : 0
              return (
                <div key={item.name} className="flex items-center gap-2">
                  <span className="w-28 shrink-0 truncate text-xs" title={item.name}>
                    {item.name}
                  </span>
                  <div className="h-2 flex-1 overflow-hidden rounded-full bg-muted">
                    <div
                      className={cn(
                        'h-full rounded-full',
                        DIST_PALETTE[idx % DIST_PALETTE.length]
                      )}
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                  <span className="w-16 shrink-0 text-right text-xs tabular-nums text-muted-foreground">
                    {item.value} · {pct}%
                  </span>
                </div>
              )
            })}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default function ResourceStatistics() {
  // 拉一页较大的设备样本做客户端聚合（资源统计为概览用途，非逐台精确报表）。
  const queryParams = useMemo<DeviceFilter & PageRequest>(
    () => ({ page: 1, pageSize: 500 }),
    []
  )
  const { data, isLoading, isError, error, isFetching } =
    useDeviceList(queryParams)

  const devices = data?.items ?? []
  const stats = data?.stats
  // 全量统计优先用后端 stats（基于全部设备而非当前样本页）；分布只能按样本算。
  const total = stats?.total ?? data?.total ?? devices.length
  const onlineCount =
    stats?.online_count ?? devices.filter((d) => d.isOnline).length
  const offlineCount = stats?.offline_count ?? Math.max(0, total - onlineCount)
  const onlineRate = total > 0 ? Math.round((onlineCount / total) * 100) : 0

  const byType = useMemo(
    () => tally(devices, (d) => d.productClass || d.networkType),
    [devices]
  )
  const byVendor = useMemo(() => tally(devices, (d) => d.vendor), [devices])
  const byNetwork = useMemo(
    () => tally(devices, (d) => d.networkType),
    [devices]
  )
  const byRegion = useMemo(() => tally(devices, (d) => d.region), [devices])

  return (
    <PageShell
      title="资源统计"
      description={`基于 ${devices.length} 台样本设备的分布概览`}
      isFetching={isFetching}
    >
      {isLoading ? (
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </div>
      ) : isError ? (
        <Card className="flex h-48 items-center justify-center p-6 text-sm text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </Card>
      ) : (
        <div className="flex flex-col gap-4">
          {/* 概览统计 */}
          <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
            <StatCard
              label="设备总数"
              value={total}
              icon={<Boxes className="size-5" />}
            />
            <StatCard
              label="在线"
              value={onlineCount}
              icon={<Wifi className="size-5" />}
              tone="emerald"
            />
            <StatCard
              label="离线"
              value={offlineCount}
              icon={<CloudOff className="size-5" />}
              tone="muted"
            />
            <StatCard
              label="在线率"
              value={`${onlineRate}%`}
              icon={<Globe className="size-5" />}
              tone="violet"
            />
          </div>

          {/* 分布卡片 */}
          {devices.length === 0 ? (
            <Card className="flex h-40 items-center justify-center p-6 text-sm text-muted-foreground">
              暂无设备样本，无法生成分布统计
            </Card>
          ) : (
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              <DistributionCard
                title="按产品类型"
                items={byType}
                total={devices.length}
              />
              <DistributionCard
                title="按厂商"
                items={byVendor}
                total={devices.length}
              />
              <DistributionCard
                title="按制式"
                items={byNetwork}
                total={devices.length}
              />
              <DistributionCard
                title="按区域"
                items={byRegion}
                total={devices.length}
              />
            </div>
          )}
        </div>
      )}
    </PageShell>
  )
}
