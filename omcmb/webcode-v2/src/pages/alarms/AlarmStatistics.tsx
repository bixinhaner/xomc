import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
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
  PageShell,
  TableCard,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useAlarmCount } from '@core/hooks/api/useAlarms'
import { useAlarmTrend, useTopAlarmDevices } from '@core/hooks/api/useDashboard'
import type { AlarmSeverity } from '@core/types/common'

// ---------------------------------------------------------------------------
// 展示常量
// ---------------------------------------------------------------------------

const SEVERITY_META: Record<
  AlarmSeverity,
  { label: string; bar: string; text: string }
> = {
  critical: { label: '紧急', bar: 'bg-red-500', text: 'text-red-600 dark:text-red-400' },
  major: { label: '重要', bar: 'bg-orange-500', text: 'text-orange-600 dark:text-orange-400' },
  minor: { label: '次要', bar: 'bg-yellow-500', text: 'text-yellow-600 dark:text-yellow-400' },
  warning: { label: '警告', bar: 'bg-sky-500', text: 'text-sky-600 dark:text-sky-400' },
}

const SEVERITY_ORDER: AlarmSeverity[] = ['critical', 'major', 'minor', 'warning']

const TIME_RANGES: { key: '7days' | '30days'; label: string; days: number }[] = [
  { key: '7days', label: '近 7 天', days: 7 },
  { key: '30days', label: '近 30 天', days: 30 },
]

type TrendPoint = {
  date: string
  critical: number
  major: number
  minor: number
  warning: number
}

function formatLocalDate(date: Date) {
  const year = date.getFullYear()
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  return `${year}-${month}-${day}`
}

function buildTrendSeries(raw: TrendPoint[] | undefined, days: number): TrendPoint[] {
  const byDate = new Map((raw ?? []).map((item) => [item.date, item]))
  return Array.from({ length: days }, (_, index) => {
    const date = new Date()
    date.setDate(date.getDate() - (days - index - 1))
    const key = formatLocalDate(date)
    const item = byDate.get(key)
    return {
      date: key,
      critical: item?.critical ?? 0,
      major: item?.major ?? 0,
      minor: item?.minor ?? 0,
      warning: item?.warning ?? 0,
    }
  })
}

// ---------------------------------------------------------------------------
// 子组件
// ---------------------------------------------------------------------------

function Stat({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: number
  tone?: 'default' | AlarmSeverity | 'amber'
}) {
  const toneClass =
    tone === 'default'
      ? 'text-foreground'
      : tone === 'amber'
        ? 'text-amber-600 dark:text-amber-400'
        : SEVERITY_META[tone].text
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>
        {value}
      </div>
    </div>
  )
}

// 严重度分布（横向 CSS 条形图，替代 echarts 饼图，点击钻取到列表）
function SeverityDistribution({
  data,
  onDrill,
}: {
  data: Record<AlarmSeverity, number>
  onDrill: (severity: AlarmSeverity) => void
}) {
  const total = SEVERITY_ORDER.reduce((sum, s) => sum + data[s], 0)
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">告警级别分布</CardTitle>
      </CardHeader>
      <CardContent>
        {total === 0 ? (
          <div className="py-10 text-center text-sm text-muted-foreground">
            暂无告警
          </div>
        ) : (
          <div className="space-y-3">
            {SEVERITY_ORDER.map((s) => {
              const v = data[s]
              const pct = total > 0 ? Math.round((v / total) * 100) : 0
              return (
                <button
                  key={s}
                  type="button"
                  onClick={() => onDrill(s)}
                  className="block w-full text-left"
                >
                  <div className="mb-1 flex items-center justify-between text-xs">
                    <span className={SEVERITY_META[s].text}>
                      {SEVERITY_META[s].label}
                    </span>
                    <span className="tabular-nums text-muted-foreground">
                      {v} · {pct}%
                    </span>
                  </div>
                  <div className="h-2.5 w-full overflow-hidden rounded-full bg-muted">
                    <div
                      className={cn('h-full rounded-full transition-all', SEVERITY_META[s].bar)}
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                </button>
              )
            })}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// 告警趋势（堆叠 CSS 柱图近似）
function TrendChart({
  series,
  range,
  onRangeChange,
}: {
  series: { date: string; critical: number; major: number; minor: number; warning: number }[]
  range: '7days' | '30days'
  onRangeChange: (range: '7days' | '30days') => void
}) {
  const max = useMemo(() => {
    let m = 0
    for (const d of series) {
      const sum = d.critical + d.major + d.minor + d.warning
      if (sum > m) m = sum
    }
    return m
  }, [series])

  const hasData = max > 0

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CardTitle className="text-sm">告警趋势</CardTitle>
          <div className="inline-flex rounded-md border p-0.5">
            {TIME_RANGES.map((r) => (
              <button
                key={r.key}
                type="button"
                className={cn(
                  'rounded px-3 py-1 text-sm transition-colors',
                  range === r.key
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:text-foreground'
                )}
                onClick={() => onRangeChange(r.key)}
              >
                {r.label}
              </button>
            ))}
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {!hasData ? (
          <div className="py-10 text-center text-sm text-muted-foreground">
            暂无趋势数据
          </div>
        ) : (
          <>
            <div className="flex h-48 items-end gap-1">
              {series.map((d) => {
                const sum = d.critical + d.major + d.minor + d.warning
                const h = max > 0 ? (sum / max) * 100 : 0
                return (
                  <div
                    key={d.date}
                    className="flex flex-1 flex-col items-center gap-1"
                    title={`${d.date} · 共 ${sum}`}
                  >
                    <div
                      className="flex w-full max-w-8 flex-col-reverse overflow-hidden rounded-t bg-muted/40"
                      style={{ height: `${Math.max(h, sum > 0 ? 4 : 0)}%` }}
                    >
                      {SEVERITY_ORDER.map((s) => {
                        const segPct = sum > 0 ? (d[s] / sum) * 100 : 0
                        return segPct > 0 ? (
                          <div
                            key={s}
                            className={SEVERITY_META[s].bar}
                            style={{ height: `${segPct}%` }}
                          />
                        ) : null
                      })}
                    </div>
                  </div>
                )
              })}
            </div>
            <div className="mt-2 flex flex-wrap gap-1 text-[10px] text-muted-foreground">
              {series.map((d) => (
                <span key={d.date} className="flex-1 text-center">
                  {d.date.slice(5)}
                </span>
              ))}
            </div>
            <div className="mt-3 flex flex-wrap items-center gap-3 text-xs">
              {SEVERITY_ORDER.map((s) => (
                <span key={s} className="inline-flex items-center gap-1.5">
                  <span className={cn('inline-block size-2.5 rounded-sm', SEVERITY_META[s].bar)} />
                  {SEVERITY_META[s].label}
                </span>
              ))}
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}

// ---------------------------------------------------------------------------
// 主页面
// ---------------------------------------------------------------------------

export default function AlarmStatistics() {
  const navigate = useNavigate()
  const [range, setRange] = useState<'7days' | '30days'>('7days')

  const days = range === '7days' ? 7 : 30

  const countQuery = useAlarmCount()
  const trendQuery = useAlarmTrend(days)
  const devicesQuery = useTopAlarmDevices()

  const alarmCount = countQuery.data
  const isFetching =
    countQuery.isFetching || trendQuery.isFetching || devicesQuery.isFetching

  const severityData = useMemo<Record<AlarmSeverity, number>>(
    () => ({
      critical: alarmCount?.critical ?? 0,
      major: alarmCount?.major ?? 0,
      minor: alarmCount?.minor ?? 0,
      warning: alarmCount?.warning ?? 0,
    }),
    [alarmCount]
  )

  const trendSeries = useMemo(
    () => buildTrendSeries(trendQuery.data, days),
    [trendQuery.data, days]
  )

  const topDevices = useMemo(
    () =>
      [...(devicesQuery.data ?? [])]
        .sort((a, b) => (b.alarmCount || 0) - (a.alarmCount || 0))
        .slice(0, 10),
    [devicesQuery.data]
  )
  const topMax = topDevices[0]?.alarmCount ?? 0

  const drillSeverity = (severity: AlarmSeverity) =>
    navigate(`/alarm/current?severity=${severity}`)
  const drillDevice = (sn: string) =>
    navigate(`/alarm/current?keyword=${encodeURIComponent(sn)}`)

  const refreshAll = () => {
    void countQuery.refetch()
    void trendQuery.refetch()
    void devicesQuery.refetch()
  }

  return (
    <PageShell
      title="告警统计"
      description="告警级别分布 · 趋势 · 高频告警设备排行"
      isFetching={isFetching}
      toolbar={
        <Button
          variant="outline"
          size="sm"
          className="ml-auto"
          onClick={refreshAll}
        >
          <RefreshCcw /> 刷新
        </Button>
      }
    >
      {/* 概览统计 */}
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-6">
        <Stat label="活动告警" value={alarmCount?.total_active ?? 0} />
        <Stat label="紧急" value={severityData.critical} tone="critical" />
        <Stat label="重要" value={severityData.major} tone="major" />
        <Stat label="次要" value={severityData.minor} tone="minor" />
        <Stat label="警告" value={severityData.warning} tone="warning" />
        <Stat label="未确认" value={alarmCount?.unacknowledged ?? 0} tone="amber" />
      </div>

      {/* 分布 + 趋势 */}
      <div className="mb-4 grid grid-cols-1 gap-4 lg:grid-cols-2">
        <SeverityDistribution data={severityData} onDrill={drillSeverity} />
        <TrendChart series={trendSeries} range={range} onRangeChange={setRange} />
      </div>

      {/* 高频告警设备排行 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">高频告警设备 Top 10</CardTitle>
        </CardHeader>
        <CardContent>
          <TableCard>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-12">#</TableHead>
                  <TableHead>设备 SN</TableHead>
                  <TableHead>设备名称</TableHead>
                  <TableHead>制式</TableHead>
                  <TableHead className="w-[40%]">告警数</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {topDevices.length === 0 ? (
                  <EmptyRow colSpan={5}>暂无高频告警设备</EmptyRow>
                ) : (
                  topDevices.map((d, i) => {
                    const pct = topMax > 0 ? (d.alarmCount / topMax) * 100 : 0
                    return (
                      <TableRow key={`${d.deviceSN}-${i}`}>
                        <TableCell className="tabular-nums text-muted-foreground">
                          {i + 1}
                        </TableCell>
                        <TableCell>
                          <button
                            type="button"
                            className="font-mono text-xs hover:underline"
                            onClick={() => drillDevice(d.deviceSN)}
                          >
                            {d.deviceSN || '—'}
                          </button>
                        </TableCell>
                        <TableCell className="text-xs">
                          {d.deviceName || '—'}
                        </TableCell>
                        <TableCell className="text-xs uppercase">
                          {d.technology || '—'}
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <div className="h-2 flex-1 overflow-hidden rounded-full bg-muted">
                              <div
                                className="h-full rounded-full bg-orange-500"
                                style={{ width: `${pct}%` }}
                              />
                            </div>
                            <span className="w-10 text-right tabular-nums text-xs">
                              {d.alarmCount}
                            </span>
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })
                )}
              </TableBody>
            </Table>
          </TableCard>
        </CardContent>
      </Card>
    </PageShell>
  )
}
