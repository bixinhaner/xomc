import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  AlertTriangle,
  AppWindow,
  ArrowDownRight,
  ArrowUpRight,
  Cpu,
  FileText,
  Minus,
  Network,
  RefreshCcw,
  Server,
  Settings,
  Signal,
  Terminal,
  Users,
  Wifi,
} from 'lucide-react'
import { useQueryClient } from '@tanstack/react-query'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
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
  ErrorRow,
  LoadingRow,
  PageShell,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useAlarmTrend,
  useDashboardData,
  useDeviceStatusByType,
  useRegionStats,
  useTopAlarmDevices,
} from '@core/hooks/api/useDashboard'
import { useUserStore } from '@core/store/userStore'
import { formatSystemTime } from '@core/utils/systemTime'

// ============================================================================
// 常量
// ============================================================================

type Technology = 'lte' | 'nr' | 'gsm'

const TECH_OPTIONS: { value: Technology; label: string }[] = [
  { value: 'lte', label: 'LTE' },
  { value: 'nr', label: '5G NR' },
  { value: 'gsm', label: 'GSM' },
]

const TECH_DISPLAY_NAME: Record<string, string> = {
  lte: 'LTE',
  nr: '5G NR',
  gsm: 'GSM',
}

// 每种制式重点关注的 KPI（来自后端指标库，summary.kpiSummary 以这些 symbolic key 提供）
const KPI_GROUPS: Record<Technology, { key: string; label: string; unit: string }[]> = {
  lte: [
    { key: 'RRC_CONN_SETUP_SR', label: 'RRC 建立成功率', unit: '%' },
    { key: 'ERAB_SETUP_SR', label: 'E-RAB 建立成功率', unit: '%' },
    { key: 'LTE_PDCP_RATE_DL', label: 'LTE 下行速率', unit: 'Mbps' },
    { key: 'LTE_PRB_UTIL_DL', label: 'LTE 下行 PRB 利用率', unit: '%' },
  ],
  nr: [
    { key: 'NR_PDCP_RATE_DL', label: 'NR 下行速率', unit: 'Mbps' },
    { key: 'NR_PDCP_RATE_UL', label: 'NR 上行速率', unit: 'Mbps' },
    { key: 'NR_SA_HO_SR', label: 'NR 切换成功率', unit: '%' },
    { key: 'NR_PRB_UTIL_DL', label: 'NR 下行 PRB 利用率', unit: '%' },
  ],
  gsm: [
    { key: 'CALL_DROP_RATE', label: '呼叫掉话率', unit: '%' },
    { key: 'CSFB_SR', label: 'CSFB 成功率', unit: '%' },
    { key: 'WIRELESS_SETUP_SR', label: '无线建立成功率', unit: '%' },
  ],
}

const ALARM_SEVERITY: {
  key: 'critical' | 'major' | 'minor' | 'warning'
  label: string
  bar: string
  dot: string
}[] = [
  { key: 'critical', label: '紧急', bar: 'bg-red-500', dot: 'bg-red-500' },
  { key: 'major', label: '重要', bar: 'bg-orange-500', dot: 'bg-orange-500' },
  { key: 'minor', label: '次要', bar: 'bg-amber-400', dot: 'bg-amber-400' },
  { key: 'warning', label: '警告', bar: 'bg-blue-400', dot: 'bg-blue-400' },
]

const QUICK_ACCESS: { label: string; path: string; icon: React.ReactNode }[] = [
  { label: '设备列表', path: '/devices', icon: <AppWindow /> },
  { label: '当前告警', path: '/alarms', icon: <AlertTriangle /> },
  { label: '拓扑监控', path: '/topology', icon: <Network /> },
  { label: '性能 KPI', path: '/performance', icon: <Signal /> },
  { label: 'MML 控制台', path: '/mml', icon: <Terminal /> },
  { label: '系统日志', path: '/logs', icon: <FileText /> },
  { label: '系统设置', path: '/system', icon: <Settings /> },
]

// ============================================================================
// 工具函数
// ============================================================================

function formatNumber(n?: number): string {
  if (n == null || Number.isNaN(n)) return '—'
  return n.toLocaleString('zh-CN')
}

function formatMetric(n?: number): string {
  if (n == null || Number.isNaN(n)) return '—'
  return n.toFixed(2)
}

function formatLastLogin(iso?: string): string {
  // #459 子单 D：保留后端系统时区钟面，不按浏览器本地二次转换。
  return formatSystemTime(iso, { format: 'YYYY-MM-DD HH:mm', placeholder: '—' })
}

// ============================================================================
// 子组件
// ============================================================================

type Trend = 'up' | 'down' | 'stable'

function KpiCard({
  icon,
  iconClass,
  label,
  value,
  deltaText,
  deltaLabel,
  trend,
  loading,
  onClick,
}: {
  icon: React.ReactNode
  iconClass: string
  label: string
  value: string
  deltaText?: string
  deltaLabel?: string
  trend?: Trend
  loading?: boolean
  onClick?: () => void
}) {
  const TrendIcon = trend === 'up' ? ArrowUpRight : trend === 'down' ? ArrowDownRight : Minus
  const trendColor =
    trend === 'up'
      ? 'text-emerald-600 dark:text-emerald-400'
      : trend === 'down'
        ? 'text-red-600 dark:text-red-400'
        : 'text-muted-foreground'

  return (
    <Card
      className={cn(onClick && 'cursor-pointer transition-shadow hover:shadow-md')}
      onClick={onClick}
    >
      <CardContent className="flex items-start gap-4 p-5">
        <div className={cn('grid size-11 shrink-0 place-items-center rounded-lg [&_svg]:size-5', iconClass)}>
          {icon}
        </div>
        <div className="min-w-0 flex-1">
          <div className="text-xs uppercase tracking-wider text-muted-foreground">{label}</div>
          <div className="mt-0.5 text-2xl font-semibold tabular-nums">
            {loading ? <span className="text-muted-foreground">…</span> : value}
          </div>
          {(deltaText || deltaLabel) && !loading && (
            <div className="mt-1 flex items-center gap-1 text-xs">
              {deltaText && (
                <span className={cn('inline-flex items-center gap-0.5 font-medium [&_svg]:size-3', trendColor)}>
                  <TrendIcon />
                  {deltaText}
                </span>
              )}
              {deltaLabel && <span className="text-muted-foreground">{deltaLabel}</span>}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

// ============================================================================
// 主页面
// ============================================================================

export function DashboardPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const currentUser = useUserStore((s) => s.currentUser)

  const [technology, setTechnology] = useState<Technology>('lte')

  const {
    data: dashboardData,
    isLoading,
    isError,
    error,
    isFetching,
  } = useDashboardData()

  const {
    data: statusByType,
    isLoading: isStatusLoading,
    isError: isStatusError,
    error: statusError,
  } = useDeviceStatusByType()

  const {
    data: topAlarmDevices,
    isLoading: isTopLoading,
    isError: isTopError,
    error: topError,
  } = useTopAlarmDevices()

  const {
    data: regionStats,
    isLoading: isRegionLoading,
    isError: isRegionError,
    error: regionError,
  } = useRegionStats()

  const {
    data: alarmTrend,
    isLoading: isTrendLoading,
    isError: isTrendError,
    error: trendError,
  } = useAlarmTrend(7)

  const summary = dashboardData?.summary
  const deviceCounts = summary?.deviceCounts
  const alarmCounts = summary?.alarmCounts
  const kpiSummary = summary?.kpiSummary ?? {}
  const kpiDeltas = summary?.kpiDeltas ?? {}

  const totalDevices = deviceCounts?.total ?? 0
  const onlineDevices = deviceCounts?.online ?? 0
  const activeAlarms = alarmCounts?.total ?? 0
  const onlineRate = totalDevices > 0 ? Math.round((onlineDevices / totalDevices) * 100) : 0

  const totalDevicesDelta = kpiDeltas['total_devices']
  const activeAlarmsDelta = kpiDeltas['active_alarms']
  const ueDelta = kpiDeltas['UE_ACTIVE']
  const currentActiveUE = Math.floor(kpiSummary['UE_ACTIVE'] ?? 0)

  const deltaText = (changePercent?: number) =>
    changePercent != null ? `${Math.abs(changePercent).toFixed(1)}%` : undefined

  const deltaLabel = (compareType?: 'yesterday' | 'last_week') =>
    compareType === 'last_week' ? '较上周' : '较昨日'

  // 当前制式重点 KPI（取 summary.kpiSummary 的 symbolic key）
  const kpiList = useMemo(() => {
    return KPI_GROUPS[technology].map((item) => ({
      ...item,
      value: kpiSummary[item.key],
    }))
  }, [technology, kpiSummary])

  // 设备按制式分组（device-status-by-type）
  const statusRows = useMemo(() => {
    if (!statusByType) return []
    return Object.entries(statusByType).map(([tech, counts]) => ({
      tech,
      label: TECH_DISPLAY_NAME[tech] ?? tech,
      online: counts.online ?? 0,
      offline: counts.offline ?? 0,
      alarm: counts.alarm ?? 0,
    }))
  }, [statusByType])

  // 告警等级分布最大值（用于条形比例）
  const maxAlarmCount = useMemo(() => {
    if (!alarmCounts) return 0
    return Math.max(
      alarmCounts.critical,
      alarmCounts.major,
      alarmCounts.minor,
      alarmCounts.warning,
      1
    )
  }, [alarmCounts])

  const hasAlarms =
    !!alarmCounts &&
    (alarmCounts.critical > 0 ||
      alarmCounts.major > 0 ||
      alarmCounts.minor > 0 ||
      alarmCounts.warning > 0)

  // 告警趋势条形最大值
  const maxTrendTotal = useMemo(() => {
    if (!alarmTrend || alarmTrend.length === 0) return 1
    return Math.max(
      1,
      ...alarmTrend.map((d) => d.critical + d.major + d.minor + d.warning)
    )
  }, [alarmTrend])

  const handleRefresh = () => {
    void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
  }

  return (
    <PageShell
      title="控制台"
      description="设备 / 告警 / KPI 总览 · 共享 @core 业务层"
      isFetching={isFetching}
      toolbar={
        <div className="ml-auto">
          <Button variant="outline" size="sm" onClick={handleRefresh}>
            <RefreshCcw className="mr-1 size-4" />
            刷新
          </Button>
        </div>
      }
    >
      {/* 全局错误（summary 拉取失败时） */}
      {isError && (
        <div className="mb-4 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          仪表盘数据加载失败：{error instanceof Error ? error.message : '未知错误'}
        </div>
      )}

      {/* Row 1: KPI 概览卡片 */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          icon={<Cpu />}
          iconClass="bg-blue-50 text-blue-600 dark:bg-blue-950/40 dark:text-blue-400"
          label="设备总数"
          value={formatNumber(totalDevices)}
          deltaText={deltaText(totalDevicesDelta?.changePercent)}
          deltaLabel={totalDevicesDelta ? deltaLabel(totalDevicesDelta.compareType) : undefined}
          trend={totalDevicesDelta?.trend ?? 'stable'}
          loading={isLoading}
          onClick={() => navigate('/devices')}
        />
        <KpiCard
          icon={<Wifi />}
          iconClass="bg-emerald-50 text-emerald-600 dark:bg-emerald-950/40 dark:text-emerald-400"
          label="在线设备"
          value={formatNumber(onlineDevices)}
          deltaText={`${onlineRate}%`}
          deltaLabel="在线率"
          trend="up"
          loading={isLoading}
          onClick={() => navigate('/devices')}
        />
        <KpiCard
          icon={<AlertTriangle />}
          iconClass="bg-red-50 text-red-600 dark:bg-red-950/40 dark:text-red-400"
          label="活动告警"
          value={formatNumber(activeAlarms)}
          deltaText={deltaText(activeAlarmsDelta?.changePercent)}
          deltaLabel={activeAlarmsDelta ? deltaLabel(activeAlarmsDelta.compareType) : undefined}
          trend={activeAlarmsDelta?.trend ?? 'stable'}
          loading={isLoading}
          onClick={() => navigate('/alarms')}
        />
        <KpiCard
          icon={<Users />}
          iconClass="bg-violet-50 text-violet-600 dark:bg-violet-950/40 dark:text-violet-400"
          label="活动用户数 (UE)"
          value={formatNumber(currentActiveUE)}
          deltaText={deltaText(ueDelta?.changePercent)}
          deltaLabel={ueDelta ? deltaLabel(ueDelta.compareType) : undefined}
          trend={ueDelta?.trend ?? 'stable'}
          loading={isLoading}
        />
      </div>

      {/* Row 2: 制式切换 + 重点 KPI */}
      <Card className="mt-6">
        <CardHeader className="flex-row items-center justify-between gap-4 space-y-0">
          <div>
            <CardTitle>网络 KPI 概览</CardTitle>
            <CardDescription>按制式查看关键性能指标</CardDescription>
          </div>
          <div className="flex items-center gap-1 rounded-lg border bg-muted/40 p-1">
            {TECH_OPTIONS.map((opt) => (
              <button
                key={opt.value}
                type="button"
                onClick={() => setTechnology(opt.value)}
                className={cn(
                  'rounded-md px-3 py-1 text-sm font-medium transition-colors',
                  technology === opt.value
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'
                )}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              {KPI_GROUPS[technology].map((k) => (
                <div key={k.key} className="rounded-lg border bg-card px-4 py-3">
                  <div className="text-xs text-muted-foreground">{k.label}</div>
                  <div className="mt-1 text-xl font-semibold text-muted-foreground">…</div>
                </div>
              ))}
            </div>
          ) : (
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              {kpiList.map((k) => (
                <div key={k.key} className="rounded-lg border bg-card px-4 py-3">
                  <div className="truncate text-xs text-muted-foreground" title={k.label}>
                    {k.label}
                  </div>
                  <div className="mt-1 flex items-baseline gap-1">
                    <span className="text-xl font-semibold tabular-nums">
                      {formatMetric(k.value)}
                    </span>
                    <span className="text-xs text-muted-foreground">{k.unit}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Row 3: 设备按制式状态 + 告警等级分布 */}
      <div className="mt-6 grid gap-6 lg:grid-cols-2">
        {/* 设备按制式状态 */}
        <Card>
          <CardHeader>
            <CardTitle>设备状态（按制式）</CardTitle>
            <CardDescription>各制式在线 / 离线 / 告警分布</CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>制式</TableHead>
                  <TableHead className="text-right">在线</TableHead>
                  <TableHead className="text-right">离线</TableHead>
                  <TableHead className="text-right">告警</TableHead>
                  <TableHead className="text-right">合计</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isStatusLoading ? (
                  <LoadingRow colSpan={5} />
                ) : isStatusError ? (
                  <ErrorRow colSpan={5} error={statusError} />
                ) : statusRows.length === 0 ? (
                  <EmptyRow colSpan={5}>暂无设备数据</EmptyRow>
                ) : (
                  statusRows.map((r) => (
                    <TableRow
                      key={r.tech}
                      className={cn(r.tech === technology && 'bg-muted/40')}
                    >
                      <TableCell className="font-medium">{r.label}</TableCell>
                      <TableCell className="text-right tabular-nums text-emerald-600 dark:text-emerald-400">
                        {r.online}
                      </TableCell>
                      <TableCell className="text-right tabular-nums text-muted-foreground">
                        {r.offline}
                      </TableCell>
                      <TableCell className="text-right tabular-nums text-amber-600 dark:text-amber-400">
                        {r.alarm}
                      </TableCell>
                      <TableCell className="text-right tabular-nums font-medium">
                        {r.online + r.offline + r.alarm}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        {/* 告警等级分布 */}
        <Card>
          <CardHeader className="flex-row items-center justify-between space-y-0">
            <div>
              <CardTitle>告警等级分布</CardTitle>
              <CardDescription>当前活动告警按严重级别统计</CardDescription>
            </div>
            <Button variant="link" size="sm" className="px-0" onClick={() => navigate('/alarms')}>
              查看全部
            </Button>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-4 py-2">
                {ALARM_SEVERITY.map((s) => (
                  <div key={s.key} className="h-6 animate-pulse rounded bg-muted" />
                ))}
              </div>
            ) : isError ? (
              <div className="py-8 text-center text-sm text-destructive">
                告警数据加载失败
              </div>
            ) : !hasAlarms ? (
              <div className="py-8 text-center text-sm text-muted-foreground">
                当前无活动告警
              </div>
            ) : (
              <div className="space-y-3 py-1">
                {ALARM_SEVERITY.map((s) => {
                  const count = alarmCounts?.[s.key] ?? 0
                  const pct = Math.round((count / maxAlarmCount) * 100)
                  return (
                    <div key={s.key} className="flex items-center gap-3">
                      <span className="flex w-16 items-center gap-1.5 text-sm">
                        <span className={cn('inline-block size-2 rounded-full', s.dot)} />
                        {s.label}
                      </span>
                      <div className="h-5 flex-1 overflow-hidden rounded bg-muted">
                        <div
                          className={cn('h-full rounded', s.bar)}
                          style={{ width: `${count > 0 ? Math.max(pct, 4) : 0}%` }}
                        />
                      </div>
                      <span className="w-10 text-right text-sm tabular-nums font-medium">
                        {count}
                      </span>
                    </div>
                  )
                })}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Row 4: Top 告警设备 + 告警趋势 */}
      <div className="mt-6 grid gap-6 lg:grid-cols-2">
        {/* Top 告警设备 */}
        <Card>
          <CardHeader>
            <CardTitle>Top 告警设备</CardTitle>
            <CardDescription>近期告警数最多的网元</CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>设备名称</TableHead>
                  <TableHead>SN</TableHead>
                  <TableHead>制式</TableHead>
                  <TableHead className="text-right">告警数</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isTopLoading ? (
                  <LoadingRow colSpan={4} />
                ) : isTopError ? (
                  <ErrorRow colSpan={4} error={topError} />
                ) : !topAlarmDevices || topAlarmDevices.length === 0 ? (
                  <EmptyRow colSpan={4}>暂无告警设备</EmptyRow>
                ) : (
                  topAlarmDevices.map((d) => (
                    <TableRow key={d.deviceSN}>
                      <TableCell className="font-medium">{d.deviceName || '—'}</TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {d.deviceSN}
                      </TableCell>
                      <TableCell>
                        <Badge variant="muted">{d.technology || '—'}</Badge>
                      </TableCell>
                      <TableCell className="text-right tabular-nums font-medium">
                        {d.alarmCount}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        {/* 告警趋势（近 7 天，堆叠条形） */}
        <Card>
          <CardHeader>
            <CardTitle>告警趋势（近 7 天）</CardTitle>
            <CardDescription>每日按严重级别的告警量</CardDescription>
          </CardHeader>
          <CardContent>
            {isTrendLoading ? (
              <div className="flex h-48 items-end gap-2">
                {Array.from({ length: 7 }).map((_, i) => (
                  <div key={i} className="flex-1 animate-pulse rounded bg-muted" style={{ height: `${30 + i * 8}%` }} />
                ))}
              </div>
            ) : isTrendError ? (
              <div className="py-12 text-center text-sm text-destructive">
                趋势数据加载失败：{trendError instanceof Error ? trendError.message : '未知错误'}
              </div>
            ) : !alarmTrend || alarmTrend.length === 0 ? (
              <div className="py-12 text-center text-sm text-muted-foreground">暂无趋势数据</div>
            ) : (
              <>
                <div className="flex h-48 items-end gap-2">
                  {alarmTrend.map((day) => {
                    const total = day.critical + day.major + day.minor + day.warning
                    const h = (n: number) => `${(n / maxTrendTotal) * 100}%`
                    return (
                      <div
                        key={day.date}
                        className="flex flex-1 flex-col justify-end"
                        title={`${day.date} · 共 ${total}`}
                      >
                        <div className="flex flex-col-reverse" style={{ height: '100%' }}>
                          <div className="w-full rounded-t-none bg-red-500" style={{ height: h(day.critical) }} />
                          <div className="w-full bg-orange-500" style={{ height: h(day.major) }} />
                          <div className="w-full bg-amber-400" style={{ height: h(day.minor) }} />
                          <div className="w-full rounded-t bg-blue-400" style={{ height: h(day.warning) }} />
                        </div>
                        <div className="mt-1 truncate text-center text-[10px] text-muted-foreground">
                          {day.date.slice(5)}
                        </div>
                      </div>
                    )
                  })}
                </div>
                <div className="mt-3 flex flex-wrap items-center justify-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
                  {ALARM_SEVERITY.map((s) => (
                    <span key={s.key} className="flex items-center gap-1">
                      <span className={cn('inline-block size-2 rounded-full', s.dot)} />
                      {s.label}
                    </span>
                  ))}
                </div>
              </>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Row 5: 区域设备统计 + 用户信息/快捷入口 */}
      <div className="mt-6 grid gap-6 lg:grid-cols-3">
        {/* 区域设备统计 */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>区域设备统计</CardTitle>
            <CardDescription>各区域设备总量与在线情况</CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>区域</TableHead>
                  <TableHead className="text-right">设备总数</TableHead>
                  <TableHead className="text-right">在线</TableHead>
                  <TableHead className="text-right">离线</TableHead>
                  <TableHead className="text-right">在线率</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isRegionLoading ? (
                  <LoadingRow colSpan={5} />
                ) : isRegionError ? (
                  <ErrorRow colSpan={5} error={regionError} />
                ) : !regionStats || regionStats.length === 0 ? (
                  <EmptyRow colSpan={5}>暂无区域数据</EmptyRow>
                ) : (
                  regionStats.map((r) => {
                    const rate = r.total > 0 ? Math.round((r.online / r.total) * 100) : 0
                    return (
                      <TableRow key={r.region}>
                        <TableCell className="font-medium">{r.region}</TableCell>
                        <TableCell className="text-right tabular-nums">{r.total}</TableCell>
                        <TableCell className="text-right tabular-nums text-emerald-600 dark:text-emerald-400">
                          {r.online}
                        </TableCell>
                        <TableCell className="text-right tabular-nums text-muted-foreground">
                          {r.offline}
                        </TableCell>
                        <TableCell className="text-right">
                          <Badge variant={rate >= 90 ? 'success' : rate >= 70 ? 'warning' : 'destructive'}>
                            {rate}%
                          </Badge>
                        </TableCell>
                      </TableRow>
                    )
                  })
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        {/* 用户信息 + 快捷入口 */}
        <Card>
          <CardHeader>
            <CardTitle>账户与快捷入口</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center gap-3 rounded-lg border bg-muted/30 p-3">
              <div className="grid size-12 shrink-0 place-items-center rounded-full bg-primary/10 text-primary [&_svg]:size-6">
                <Server />
              </div>
              <div className="min-w-0">
                <div className="truncate font-medium">
                  {currentUser?.displayName || currentUser?.username || '系统管理员'}
                </div>
                <div className="truncate text-xs text-muted-foreground">
                  {currentUser?.email || '—'}
                </div>
                <div className="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
                  <Badge variant="success">在线</Badge>
                  <span>上次登录 {formatLastLogin(currentUser?.lastLoginTime)}</span>
                </div>
              </div>
            </div>

            <div className="grid grid-cols-3 gap-2">
              {QUICK_ACCESS.map((item) => (
                <button
                  key={item.path}
                  type="button"
                  onClick={() => navigate(item.path)}
                  className="flex flex-col items-center gap-1.5 rounded-lg border bg-card px-2 py-3 text-center transition-colors hover:border-primary/50 hover:bg-accent [&_svg]:size-5 [&_svg]:text-muted-foreground"
                >
                  {item.icon}
                  <span className="text-xs">{item.label}</span>
                </button>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </PageShell>
  )
}
