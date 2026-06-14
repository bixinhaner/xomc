import { useMemo, useState } from 'react'
import { RefreshCcw, Search } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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
  formatTime,
  LoadingRow,
  PageShell,
  TableCard,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useDeviceList } from '@core/hooks/api/useDevices'
import { useAggregatedMetricsByDevices } from '@core/hooks/api/usePmQuery'
import type { DeviceFilter } from '@core/types/device'
import type { PageRequest } from '@core/types/pagination'
import type { AggregatedRow, Granularity } from '@core/types/pmDashboard'

// ============================================================
// 设备性能即席查看 — 对齐 v1 /performance/device-view (PmDashboard/DeviceListPane)
//   不依赖任何聚合任务：选 1 个设备 + 粒度 + 时间窗，直接对接 /pm/metrics/aggregated。
//   出图：每指标一行（时间桶 × 值透视表 + CSS 柱状）。
// ============================================================

const DEVICE_PAGE_SIZE = 15

const GRANULARITY_OPTIONS: { label: string; value: Granularity }[] = [
  { label: '15 分钟', value: '15min' },
  { label: '小时', value: 'hourly' },
  { label: '天', value: 'daily' },
]

const WINDOW_OPTIONS: { label: string; value: string; ms: number }[] = [
  { label: '近 24 小时', value: '24h', ms: 24 * 3600_000 },
  { label: '近 7 天', value: '7d', ms: 7 * 24 * 3600_000 },
  { label: '近 30 天', value: '30d', ms: 30 * 24 * 3600_000 },
]

// 按指标分组聚合行，每指标取时间序列。
function groupByMetric(rows: AggregatedRow[]): {
  path: string
  label: string
  points: { time: string; value: number | null }[]
}[] {
  const map = new Map<string, { label: string; points: { time: string; value: number | null }[] }>()
  for (const r of rows) {
    const entry = map.get(r.metricPath) ?? { label: r.displayName || r.metricPath, points: [] }
    entry.points.push({ time: r.time, value: r.metricValue })
    map.set(r.metricPath, entry)
  }
  return Array.from(map.entries()).map(([path, v]) => ({
    path,
    label: v.label,
    points: v.points.sort((a, b) => a.time.localeCompare(b.time)),
  }))
}

function MetricCard({
  metric,
}: {
  metric: { path: string; label: string; points: { time: string; value: number | null }[] }
}) {
  const nums = metric.points.map((p) => p.value).filter((v): v is number => v !== null)
  const max = nums.reduce((m, v) => Math.max(m, v), 0)
  const min = nums.reduce((m, v) => Math.min(m, v), nums.length ? nums[0] : 0)
  const span = max - min || 1
  const last = nums.length ? nums[nums.length - 1] : null

  return (
    <Card className="p-4">
      <div className="mb-2 flex items-baseline justify-between gap-2">
        <div className="truncate text-sm font-medium" title={metric.label}>
          {metric.label}
        </div>
        <div className="shrink-0 text-xs tabular-nums text-muted-foreground">
          {last === null ? '—' : `最新 ${last}`}
        </div>
      </div>
      <div className="flex h-32 items-end gap-px">
        {metric.points.map((p, i) => {
          const h = p.value === null ? 2 : 8 + ((p.value - min) / span) * 88
          return (
            <div
              key={`${p.time}-${i}`}
              className={cn(
                'flex-1 rounded-t transition-all',
                p.value === null ? 'bg-muted' : 'bg-primary/70 hover:bg-primary'
              )}
              style={{ height: `${h}%` }}
              title={`${formatTime(p.time)} · ${p.value ?? '—'}`}
            />
          )
        })}
      </div>
      <div className="mt-1 text-[10px] text-muted-foreground">{metric.points.length} 个时间桶</div>
    </Card>
  )
}

export function DeviceViewPage() {
  const [keyword, setKeyword] = useState('')
  const [selectedSn, setSelectedSn] = useState<string | null>(null)
  const [granularity, setGranularity] = useState<Granularity>('15min')
  const [windowVal, setWindowVal] = useState('7d')
  const [run, setRun] = useState(false)

  const deviceParams = useMemo<DeviceFilter & PageRequest>(
    () => ({
      page: 1,
      pageSize: DEVICE_PAGE_SIZE,
      ...(keyword.trim() ? { searchText: keyword.trim() } : {}),
    }),
    [keyword]
  )
  const {
    data: deviceData,
    isLoading: devLoading,
    isError: devError,
    error: devErr,
    isFetching: devFetching,
  } = useDeviceList(deviceParams)
  const devices = deviceData?.items ?? []

  const range = useMemo(() => {
    const opt = WINDOW_OPTIONS.find((o) => o.value === windowVal) ?? WINDOW_OPTIONS[1]
    const now = Date.now()
    return { start: new Date(now - opt.ms).toISOString(), end: new Date(now).toISOString() }
  }, [windowVal])

  const agg = useAggregatedMetricsByDevices(
    { granularity, startTime: range.start, endTime: range.end },
    selectedSn ? [selectedSn] : [],
    run && Boolean(selectedSn)
  )

  const metrics = useMemo(() => groupByMetric(agg.data), [agg.data])

  const onPick = (sn: string) => {
    setSelectedSn(sn)
    setRun(false)
  }

  const devCols = ['SN', '基站名称']

  return (
    <PageShell
      title="设备性能查看"
      description="选基站 + 粒度 + 时间窗,即席查看 PM 聚合指标(不依赖聚合任务)"
      isFetching={devFetching || agg.isFetching}
    >
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[320px_1fr]">
        {/* 设备选择 */}
        <Card className="p-3">
          <div className="relative mb-2">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-9"
              placeholder="SN / 基站名称"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <TableCard>
            <Table>
              <TableHeader>
                <TableRow>
                  {devCols.map((c) => (
                    <TableHead key={c}>{c}</TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {devLoading ? (
                  <LoadingRow colSpan={devCols.length} />
                ) : devError ? (
                  <ErrorRow colSpan={devCols.length} error={devErr} />
                ) : devices.length === 0 ? (
                  <EmptyRow colSpan={devCols.length}>暂无设备</EmptyRow>
                ) : (
                  devices.map((d) => (
                    <TableRow
                      key={d.id}
                      className={cn(
                        'cursor-pointer',
                        selectedSn === d.sn && 'bg-accent'
                      )}
                      onClick={() => onPick(d.sn)}
                    >
                      <TableCell className="font-mono text-xs">{d.sn}</TableCell>
                      <TableCell className="text-xs">
                        {d.hostName || d.name || d.deviceName || '—'}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableCard>
        </Card>

        {/* 查询区 + 结果 */}
        <div className="space-y-4">
          <Card className="p-4">
            <div className="flex flex-wrap items-end gap-3">
              <div className="flex flex-col gap-1">
                <Label className="text-xs text-muted-foreground">粒度</Label>
                <Select value={granularity} onValueChange={(v) => setGranularity(v as Granularity)}>
                  <SelectTrigger className="w-32">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {GRANULARITY_OPTIONS.map((o) => (
                      <SelectItem key={o.value} value={o.value}>
                        {o.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="flex flex-col gap-1">
                <Label className="text-xs text-muted-foreground">时间窗</Label>
                <Select value={windowVal} onValueChange={setWindowVal}>
                  <SelectTrigger className="w-36">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {WINDOW_OPTIONS.map((o) => (
                      <SelectItem key={o.value} value={o.value}>
                        {o.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <Button
                size="sm"
                disabled={!selectedSn}
                onClick={() => {
                  setRun(true)
                  if (run) agg.refetch()
                }}
              >
                查询
              </Button>
              {agg.isFetching && (
                <Button variant="outline" size="sm" disabled>
                  <RefreshCcw className="size-4 animate-spin" /> 加载中
                </Button>
              )}
            </div>
            <div className="mt-3 flex flex-wrap items-center gap-1.5 text-xs">
              {selectedSn ? (
                <>
                  <Badge variant="outline">设备 {selectedSn}</Badge>
                  {run && <Badge variant="muted">共 {agg.total} 行</Badge>}
                  {run && agg.truncated && <Badge variant="warning">结果已截断</Badge>}
                </>
              ) : (
                <span className="text-muted-foreground">请先从左侧选择一个基站</span>
              )}
            </div>
          </Card>

          {!selectedSn ? (
            <Card className="flex h-48 items-center justify-center text-sm text-muted-foreground">
              选基站后点击「查询」加载聚合指标
            </Card>
          ) : !run ? (
            <Card className="flex h-48 items-center justify-center text-sm text-muted-foreground">
              点击「查询」加载聚合指标
            </Card>
          ) : agg.isLoading ? (
            <Card className="flex h-48 items-center justify-center text-sm text-muted-foreground">
              加载聚合数据…
            </Card>
          ) : agg.isError ? (
            <Card className="flex h-48 items-center justify-center text-sm text-destructive">
              加载失败：
              {agg.errors[0] instanceof Error ? agg.errors[0].message : '查询失败'}
            </Card>
          ) : metrics.length === 0 ? (
            <Card className="flex h-48 items-center justify-center text-sm text-muted-foreground">
              该设备在此时间窗内暂无聚合指标数据
            </Card>
          ) : (
            <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
              {metrics.map((m) => (
                <MetricCard key={m.path} metric={m} />
              ))}
            </div>
          )}
        </div>
      </div>
    </PageShell>
  )
}

export default DeviceViewPage
