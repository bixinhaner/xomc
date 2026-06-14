import { useMemo, useState } from 'react'
import {
  RefreshCcw,
  Download,
  FileBarChart2,
  FileClock,
  Play,
  Activity,
  BellRing,
  Server,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
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
  Pagination,
  TableCard,
  formatBytes,
  formatTime,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useReportDefinitions,
  useReportRecords,
  useGenerateReport,
  useDownloadReport,
  useReportSampleData,
} from '@core/hooks/api/useReports'
import type {
  ReportDefinition,
  ReportRecord,
  ReportType,
  ReportStatus,
} from '@core/mock/data/reports'

// ============================================================
// 报表中心 (F06 / report) — v2 (shadcn + tailwind)
// 业务深度对齐 v1 webcode/src/pages/report：
//   · 报表定义列表（类型/周期/格式/自动生成/状态）+ 类型/状态筛选
//   · 生成记录列表（周期/格式/大小/状态/生成时间）+ 按定义筛选
//   · 主操作：从定义生成报表（useGenerateReport）、下载记录（useDownloadReport）
//   · 概览统计：来自 /reports/sample-data（KPI / 告警 / 设备摘要）
// 未覆盖（见 notes）：v1 的 LTE 标准报表分类树、单站 KPI 报表、历史 KPI 折线下钻、
//   轮询统计——这些 v1 页仍消费本地 mock/生成器，无对应后端端点，本轮不在主页内重建。
// ============================================================

type ViewMode = 'definitions' | 'records'

const REPORT_TYPE_LABEL: Record<ReportType, string> = {
  performance: '性能',
  alarm: '告警',
  device: '设备',
  capacity: '容量',
  security: '安全',
}

const REPORT_TYPE_OPTIONS: { value: ReportType; label: string }[] = [
  { value: 'performance', label: '性能' },
  { value: 'alarm', label: '告警' },
  { value: 'device', label: '设备' },
  { value: 'capacity', label: '容量' },
  { value: 'security', label: '安全' },
]

const STATUS_LABEL: Record<ReportStatus, string> = {
  draft: '草稿',
  published: '已发布',
  archived: '已归档',
}

const STATUS_VARIANT: Record<ReportStatus, 'success' | 'muted' | 'secondary'> = {
  draft: 'secondary',
  published: 'success',
  archived: 'muted',
}

const PERIOD_LABEL: Record<string, string> = {
  daily: '日报',
  weekly: '周报',
  monthly: '月报',
  quarterly: '季报',
  custom: '自定义',
}

const RECORD_STATUS_LABEL: Record<ReportRecord['status'], string> = {
  generating: '生成中',
  ready: '就绪',
  failed: '失败',
}

const RECORD_STATUS_VARIANT: Record<
  ReportRecord['status'],
  'success' | 'warning' | 'destructive'
> = {
  generating: 'warning',
  ready: 'success',
  failed: 'destructive',
}

// ---- /reports/sample-data 的弱类型读取（real API 返回 Record<string, unknown>）----
type KpiMetric = { value: number; unit?: string; trend?: number; status?: string }

function asRecord(v: unknown): Record<string, unknown> | undefined {
  return v && typeof v === 'object' ? (v as Record<string, unknown>) : undefined
}
function asNumber(v: unknown): number | undefined {
  return typeof v === 'number' && Number.isFinite(v) ? v : undefined
}
function asKpiMetric(v: unknown): KpiMetric | undefined {
  const r = asRecord(v)
  const value = asNumber(r?.value)
  if (value === undefined) return undefined
  return {
    value,
    unit: typeof r?.unit === 'string' ? r.unit : undefined,
    trend: asNumber(r?.trend),
    status: typeof r?.status === 'string' ? r.status : undefined,
  }
}

interface OverviewModel {
  kpis: { label: string; metric: KpiMetric }[]
  alarmTotal?: number
  alarmAvgClear?: number
  deviceTotal?: number
  deviceOnline?: number
  deviceOffline?: number
}

function buildOverview(sample: Record<string, unknown> | undefined): OverviewModel | undefined {
  if (!sample) return undefined
  const kpiSummary = asRecord(sample.kpiSummary)
  const alarmSummary = asRecord(sample.alarmSummary)
  const deviceSummary = asRecord(sample.deviceSummary)

  const kpiDefs: { key: string; label: string }[] = [
    { key: 'rrcSuccRate', label: 'RRC 建立成功率' },
    { key: 'erabSuccRate', label: 'E-RAB 成功率' },
    { key: 'hoSuccRate', label: '切换成功率' },
    { key: 'dlThroughput', label: '下行吞吐' },
    { key: 'ulThroughput', label: '上行吞吐' },
    { key: 'radioDrop', label: '无线掉线率' },
  ]
  const kpis: { label: string; metric: KpiMetric }[] = []
  if (kpiSummary) {
    for (const d of kpiDefs) {
      const m = asKpiMetric(kpiSummary[d.key])
      if (m) kpis.push({ label: d.label, metric: m })
    }
  }

  const model: OverviewModel = {
    kpis,
    alarmTotal: asNumber(alarmSummary?.totalAlarms),
    alarmAvgClear: asNumber(alarmSummary?.avgClearTime),
    deviceTotal: asNumber(deviceSummary?.totalDevices),
    deviceOnline: asNumber(deviceSummary?.online),
    deviceOffline: asNumber(deviceSummary?.offline),
  }
  const hasAnything =
    model.kpis.length > 0 ||
    model.alarmTotal !== undefined ||
    model.deviceTotal !== undefined
  return hasAnything ? model : undefined
}

export function ReportsPage() {
  const [view, setView] = useState<ViewMode>('definitions')

  // -------- 报表定义筛选 --------
  const [defPage, setDefPage] = useState(1)
  const [typeFilter, setTypeFilter] = useState<ReportType | ''>('')
  const [statusFilter, setStatusFilter] = useState<ReportStatus | ''>('')

  // -------- 生成记录筛选 --------
  const [recPage, setRecPage] = useState(1)
  const [recDefFilter, setRecDefFilter] = useState<string>('')

  const pageSize = 20

  const defParams = useMemo(
    () => ({
      page: defPage,
      pageSize,
      ...(typeFilter ? { reportType: typeFilter } : {}),
      ...(statusFilter ? { status: statusFilter } : {}),
    }),
    [defPage, typeFilter, statusFilter],
  )

  const recParams = useMemo(
    () => ({
      page: recPage,
      pageSize,
      ...(recDefFilter ? { reportDefinitionId: recDefFilter } : {}),
    }),
    [recPage, recDefFilter],
  )

  const definitionsQuery = useReportDefinitions(defParams)
  const recordsQuery = useReportRecords(recParams)
  const sampleQuery = useReportSampleData()

  const generate = useGenerateReport()
  const download = useDownloadReport()

  const overview = useMemo(
    () => buildOverview(sampleQuery.data as Record<string, unknown> | undefined),
    [sampleQuery.data],
  )

  const defs = definitionsQuery.data?.items ?? []
  const defTotal = definitionsQuery.data?.total ?? 0
  const defTotalPages = Math.max(1, Math.ceil(defTotal / pageSize))

  const recs = recordsQuery.data?.items ?? []
  const recTotal = recordsQuery.data?.total ?? 0
  const recTotalPages = Math.max(1, Math.ceil(recTotal / pageSize))

  // 定义选项（供生成记录页按定义过滤）。取已加载的定义列表即可。
  const defOptions = defs

  const handleGenerate = (def: ReportDefinition) => {
    if (generate.isPending) return
    generate.mutate({ definitionId: def.id })
  }

  const handleViewRecords = (def: ReportDefinition) => {
    setRecDefFilter(def.id)
    setRecPage(1)
    setView('records')
  }

  const handleDownload = (rec: ReportRecord) => {
    if (download.isPending || rec.status !== 'ready') return
    download.mutate(rec.id, {
      onSuccess: (res) => {
        if (res?.url) window.open(res.url, '_blank', 'noopener,noreferrer')
      },
    })
  }

  const isFetching =
    definitionsQuery.isFetching ||
    recordsQuery.isFetching ||
    sampleQuery.isFetching ||
    generate.isPending ||
    download.isPending

  const description =
    view === 'definitions'
      ? `报表定义 · 共 ${defTotal} 项`
      : `生成记录 · 共 ${recTotal} 份`

  return (
    <PageShell
      title="报表中心"
      description={description}
      isFetching={isFetching}
      toolbar={
        <>
          <div className="inline-flex overflow-hidden rounded-md border">
            <button
              type="button"
              onClick={() => setView('definitions')}
              className={cn(
                'flex items-center gap-1.5 px-3 py-1.5 text-sm transition-colors',
                view === 'definitions'
                  ? 'bg-primary/10 text-primary'
                  : 'text-muted-foreground hover:bg-muted',
              )}
            >
              <FileBarChart2 className="size-4" />
              报表定义
            </button>
            <button
              type="button"
              onClick={() => setView('records')}
              className={cn(
                'flex items-center gap-1.5 border-l px-3 py-1.5 text-sm transition-colors',
                view === 'records'
                  ? 'bg-primary/10 text-primary'
                  : 'text-muted-foreground hover:bg-muted',
              )}
            >
              <FileClock className="size-4" />
              生成记录
            </button>
          </div>

          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={() => {
              void definitionsQuery.refetch()
              void recordsQuery.refetch()
              void sampleQuery.refetch()
            }}
          >
            <RefreshCcw /> 刷新
          </Button>
        </>
      }
    >
      {/* ============ 概览统计 ============ */}
      <OverviewSection
        overview={overview}
        isLoading={sampleQuery.isLoading}
        isError={sampleQuery.isError}
      />

      {/* ============ 报表定义视图 ============ */}
      {view === 'definitions' ? (
        <>
          <div className="mb-3 flex flex-wrap items-center gap-2">
            <Select
              value={typeFilter || 'all'}
              onValueChange={(v) => {
                setTypeFilter(v === 'all' ? '' : (v as ReportType))
                setDefPage(1)
              }}
            >
              <SelectTrigger className="w-40">
                <SelectValue placeholder="报表类型" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部类型</SelectItem>
                {REPORT_TYPE_OPTIONS.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Select
              value={statusFilter || 'all'}
              onValueChange={(v) => {
                setStatusFilter(v === 'all' ? '' : (v as ReportStatus))
                setDefPage(1)
              }}
            >
              <SelectTrigger className="w-36">
                <SelectValue placeholder="状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部状态</SelectItem>
                <SelectItem value="published">已发布</SelectItem>
                <SelectItem value="draft">草稿</SelectItem>
                <SelectItem value="archived">已归档</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <TableCard>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>报表</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>周期</TableHead>
                  <TableHead>格式</TableHead>
                  <TableHead>自动生成</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>最近生成</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {definitionsQuery.isLoading ? (
                  <LoadingRow colSpan={8} />
                ) : definitionsQuery.isError ? (
                  <ErrorRow colSpan={8} error={definitionsQuery.error} />
                ) : defs.length === 0 ? (
                  <EmptyRow colSpan={8}>暂无报表定义</EmptyRow>
                ) : (
                  defs.map((d) => (
                    <TableRow key={d.id}>
                      <TableCell>
                        <div className="font-medium">{d.reportName}</div>
                        <div className="line-clamp-1 text-xs text-muted-foreground">
                          {d.description || '—'}
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">
                          {REPORT_TYPE_LABEL[d.reportType] ?? d.reportType}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-xs">
                        {PERIOD_LABEL[d.period] ?? d.period}
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1">
                          {(d.format ?? []).length > 0 ? (
                            d.format.map((f) => (
                              <Badge key={f} variant="muted" className="uppercase">
                                {f}
                              </Badge>
                            ))
                          ) : (
                            <span className="text-xs text-muted-foreground">—</span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="text-xs">
                        {d.autoGenerate ? (
                          <Badge variant="success">
                            启用
                            {d.cronExpression ? (
                              <span className="ml-1 font-mono opacity-70">
                                {d.cronExpression}
                              </span>
                            ) : null}
                          </Badge>
                        ) : (
                          <Badge variant="muted">关闭</Badge>
                        )}
                      </TableCell>
                      <TableCell>
                        <Badge variant={STATUS_VARIANT[d.status] ?? 'muted'}>
                          {STATUS_LABEL[d.status] ?? d.status}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {formatTime(d.lastGenTime)}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center justify-end gap-1">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleViewRecords(d)}
                          >
                            <FileClock className="size-3.5" /> 记录
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={generate.isPending}
                            onClick={() => handleGenerate(d)}
                          >
                            <Play className="size-3.5" /> 生成
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableCard>

          <Pagination
            page={defPage}
            totalPages={defTotalPages}
            pageSize={pageSize}
            onChange={setDefPage}
          />
        </>
      ) : (
        /* ============ 生成记录视图 ============ */
        <>
          <div className="mb-3 flex flex-wrap items-center gap-2">
            <Select
              value={recDefFilter || 'all'}
              onValueChange={(v) => {
                setRecDefFilter(v === 'all' ? '' : v)
                setRecPage(1)
              }}
            >
              <SelectTrigger className="w-72">
                <SelectValue placeholder="按报表定义筛选" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部报表</SelectItem>
                {defOptions.map((d) => (
                  <SelectItem key={d.id} value={d.id}>
                    {d.reportName}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <TableCard>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>报表</TableHead>
                  <TableHead>周期</TableHead>
                  <TableHead>格式</TableHead>
                  <TableHead>大小</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>生成时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {recordsQuery.isLoading ? (
                  <LoadingRow colSpan={7} />
                ) : recordsQuery.isError ? (
                  <ErrorRow colSpan={7} error={recordsQuery.error} />
                ) : recs.length === 0 ? (
                  <EmptyRow colSpan={7}>暂无生成记录</EmptyRow>
                ) : (
                  recs.map((r) => (
                    <TableRow key={r.id}>
                      <TableCell className="font-medium">{r.reportName}</TableCell>
                      <TableCell className="text-xs">{r.period || '—'}</TableCell>
                      <TableCell>
                        <Badge variant="muted" className="uppercase">
                          {r.format}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-xs tabular-nums text-muted-foreground">
                        {formatBytes(r.fileSize)}
                      </TableCell>
                      <TableCell>
                        <Badge variant={RECORD_STATUS_VARIANT[r.status]}>
                          {RECORD_STATUS_LABEL[r.status]}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {formatTime(r.generateTime)}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center justify-end">
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={r.status !== 'ready' || download.isPending}
                            onClick={() => handleDownload(r)}
                          >
                            <Download className="size-3.5" /> 下载
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableCard>

          <Pagination
            page={recPage}
            totalPages={recTotalPages}
            pageSize={pageSize}
            onChange={setRecPage}
          />
        </>
      )}
    </PageShell>
  )
}

// ============================================================
// 概览统计区块
// ============================================================
function OverviewSection({
  overview,
  isLoading,
  isError,
}: {
  overview: OverviewModel | undefined
  isLoading: boolean
  isError: boolean
}) {
  if (isError) {
    // 概览失败不阻断主列表，仅静默隐藏
    return null
  }

  if (isLoading) {
    return (
      <div className="mb-6 grid grid-cols-2 gap-3 md:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Card key={i}>
            <CardContent className="p-4">
              <div className="h-3 w-16 animate-pulse rounded bg-muted" />
              <div className="mt-2 h-6 w-20 animate-pulse rounded bg-muted" />
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }

  if (!overview) return null

  const hasSummaryCards =
    overview.deviceTotal !== undefined || overview.alarmTotal !== undefined

  return (
    <div className="mb-6 space-y-3">
      {hasSummaryCards && (
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          {overview.deviceTotal !== undefined && (
            <SummaryCard
              icon={<Server className="size-4" />}
              label="设备总数"
              value={overview.deviceTotal}
              sub={
                overview.deviceOnline !== undefined
                  ? `在线 ${overview.deviceOnline} · 离线 ${overview.deviceOffline ?? 0}`
                  : undefined
              }
            />
          )}
          {overview.deviceOnline !== undefined && (
            <SummaryCard
              icon={<Activity className="size-4" />}
              label="在线设备"
              value={overview.deviceOnline}
              tone="emerald"
            />
          )}
          {overview.alarmTotal !== undefined && (
            <SummaryCard
              icon={<BellRing className="size-4" />}
              label="告警总数"
              value={overview.alarmTotal}
              tone="amber"
            />
          )}
          {overview.alarmAvgClear !== undefined && (
            <SummaryCard
              icon={<FileClock className="size-4" />}
              label="平均处置(分)"
              value={overview.alarmAvgClear}
            />
          )}
        </div>
      )}

      {overview.kpis.length > 0 && (
        <Card>
          <CardContent className="p-4">
            <div className="mb-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">
              KPI 摘要
            </div>
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
              {overview.kpis.map((k) => (
                <KpiCell key={k.label} label={k.label} metric={k.metric} />
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function SummaryCard({
  icon,
  label,
  value,
  sub,
  tone = 'default',
}: {
  icon: React.ReactNode
  label: string
  value: number
  sub?: string
  tone?: 'default' | 'emerald' | 'amber'
}) {
  const toneClass = {
    default: 'text-foreground',
    emerald: 'text-emerald-600 dark:text-emerald-400',
    amber: 'text-amber-600 dark:text-amber-400',
  }[tone]
  return (
    <Card>
      <CardContent className="p-4">
        <div className="flex items-center gap-1.5 text-xs uppercase tracking-wider text-muted-foreground">
          {icon}
          {label}
        </div>
        <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>
          {value.toLocaleString()}
        </div>
        {sub && <div className="mt-0.5 text-xs text-muted-foreground">{sub}</div>}
      </CardContent>
    </Card>
  )
}

function KpiCell({ label, metric }: { label: string; metric: KpiMetric }) {
  const trend = metric.trend
  const trendClass =
    trend === undefined
      ? 'text-muted-foreground'
      : trend > 0
        ? 'text-emerald-600 dark:text-emerald-400'
        : trend < 0
          ? 'text-rose-600 dark:text-rose-400'
          : 'text-muted-foreground'
  return (
    <div className="rounded-lg border bg-card px-3 py-2">
      <div className="truncate text-xs text-muted-foreground" title={label}>
        {label}
      </div>
      <div className="mt-1 flex items-baseline gap-1">
        <span className="text-lg font-semibold tabular-nums">{metric.value}</span>
        {metric.unit && (
          <span className="text-xs text-muted-foreground">{metric.unit}</span>
        )}
      </div>
      {trend !== undefined && (
        <div className={cn('text-xs tabular-nums', trendClass)}>
          {trend > 0 ? '+' : ''}
          {trend}
        </div>
      )}
    </div>
  )
}
