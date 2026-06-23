import { useMemo, useState } from 'react'
import { RefreshCcw, Search } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
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

import { useQueryTemplates, useAggregatedMetricsByDevices } from '@core/hooks/api/usePmQuery'
import { pivotLongToWide, formatPivotNumber } from '@core/utils/pmPivotTransform'
import type { QueryTemplate } from '@core/types/pmQuery'
import type { AggregatedRow, Granularity } from '@core/types/pmDashboard'
import { useT, type TranslateFn } from '@/hooks/useT'

// ============================================================
// 指标查询 — 对齐 v1 /performance/query
//   - 左侧：查询模板列表（真 API /pm/query-templates）
//   - 选模板后右侧用模板内 deviceSns/metricPaths/granularity 跑聚合查询（/pm/metrics/aggregated）
//   - 结果透视：行=时间桶 / 列=指标 / 单元格=值
// ============================================================

// 把时窗预设映射成 RFC3339 起止时间。
function presetToRange(preset: QueryTemplate['payload']['timeRangePreset'], payload: QueryTemplate['payload']): {
  start: string
  end: string
} {
  const now = new Date()
  const end = now.toISOString()
  const ms = (n: number) => new Date(now.getTime() - n).toISOString()
  switch (preset) {
    case 'last_1h':
      return { start: ms(3600_000), end }
    case 'last_24h':
      return { start: ms(24 * 3600_000), end }
    case 'last_7d':
      return { start: ms(7 * 24 * 3600_000), end }
    case 'last_30d':
      return { start: ms(30 * 24 * 3600_000), end }
    case 'custom':
      return {
        start: payload.absoluteStart ?? ms(24 * 3600_000),
        end: payload.absoluteEnd ?? end,
      }
    default:
      return { start: ms(24 * 3600_000), end }
  }
}

// 聚合行透视：用 pivotLongToWide 将 long format 转 wide format。

function ResultTable({
  rows,
  total,
  truncated,
  isLoading,
  isError,
  errors,
  t,
}: {
  rows: AggregatedRow[]
  total: number
  truncated: boolean
  isLoading: boolean
  isError: boolean
  errors: unknown[]
  t: TranslateFn
}) {
  const pivoted = useMemo(() => pivotLongToWide(rows), [rows])
  const colCount = pivoted.columns.length + 5

  return (
    <div>
      <div className="mb-2 flex items-center gap-2 text-xs text-muted-foreground">
        <span>{t('perf.kpiQuery.pivot.totalRows', { count: total })}</span>
        {truncated && <Badge variant="warning">{t('perf.kpiQuery.truncated')}</Badge>}
      </div>
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('perf.kpiQuery.pivot.startTime')}</TableHead>
              <TableHead>{t('perf.kpiQuery.pivot.deviceSn')}</TableHead>
              <TableHead>Cell ID</TableHead>
              <TableHead>PLMN</TableHead>
              <TableHead>{t('perf.kpiQuery.pivot.measObject')}</TableHead>
              {pivoted.columns.map((c) => (
                <TableHead key={c.key}>{c.title}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={errors[0] ?? new Error(t('perf.kpiQuery.queryFailedShort'))} />
            ) : pivoted.rows.length === 0 ? (
              <EmptyRow colSpan={colCount}>{t('perf.kpiQuery.noResults')}</EmptyRow>
            ) : (
              pivoted.rows.map((row) => (
                <TableRow key={row.key}>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(row.time)}
                  </TableCell>
                  <TableCell className="text-xs">{row.deviceSn ?? '—'}</TableCell>
                  <TableCell className="text-xs">{row.cellId ?? '—'}</TableCell>
                  <TableCell className="text-xs">{row.plmn ?? '—'}</TableCell>
                  <TableCell className="text-xs">{row.measObject ?? '—'}</TableCell>
                  {pivoted.columns.map((c) => (
                    <TableCell key={c.key} className="tabular-nums">
                      {formatPivotNumber(row.cells[c.key])}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>
    </div>
  )
}

export function KPIQueryPage() {
  const t = useT()
  const [keyword, setKeyword] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [run, setRun] = useState(false)

  const {
    data: templates,
    isLoading: tplLoading,
    isError: tplError,
    error: tplErr,
    isFetching: tplFetching,
    refetch: refetchTpl,
  } = useQueryTemplates()

  const list = templates?.items ?? []
  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return list
    return list.filter((t) => t.name.toLowerCase().includes(kw))
  }, [list, keyword])

  const selected = useMemo(
    () => list.find((t) => t.id === selectedId) ?? null,
    [list, selectedId]
  )

  // 选中模板的查询参数（无模板/未运行时不发请求）
  const range = useMemo(
    () => (selected ? presetToRange(selected.payload.timeRangePreset, selected.payload) : null),
    [selected]
  )
  const granularity: Granularity = (selected?.payload.granularity as Granularity) ?? '15min'
  const deviceSns = selected?.payload.deviceSns ?? []
  const metricPaths = selected?.payload.metricPaths ?? []

  const agg = useAggregatedMetricsByDevices(
    {
      granularity,
      metricPaths,
      startTime: range?.start,
      endTime: range?.end,
    },
    deviceSns,
    run && Boolean(selected) && deviceSns.length > 0
  )

  const onSelect = (id: string) => {
    setSelectedId(id)
    setRun(false)
  }

  return (
    <PageShell
      title={t('perf.kpiQuery.v2.title')}
      description={t('perf.kpiQuery.v2.desc')}
      isFetching={tplFetching || agg.isFetching}
    >
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[320px_1fr]">
        {/* 模板列表 */}
        <Card className="p-3">
          <div className="mb-2 flex items-center gap-2">
            <div className="relative flex-1">
              <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                className="pl-9"
                placeholder={t('perf.kpiQuery.searchTemplate')}
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
              />
            </div>
            <Button
              variant="outline"
              size="sm"
              disabled={tplFetching}
              onClick={() => void refetchTpl()}
              aria-label={t('perf.kpiQuery.refreshTemplates')}
            >
              <RefreshCcw className="size-4" />
            </Button>
          </div>
          <div className="max-h-[560px] space-y-1 overflow-auto">
            {tplLoading ? (
              <div className="p-6 text-center text-sm text-muted-foreground">{t('perf.kpiQuery.loadingTemplates')}</div>
            ) : tplError ? (
              <div className="p-6 text-center text-sm text-destructive">
                {t('perf.kpiQuery.loadFailed', {
                  msg: tplErr instanceof Error ? tplErr.message : t('perf.kpiQuery.unknownError'),
                })}
              </div>
            ) : filtered.length === 0 ? (
              <div className="p-6 text-center text-sm text-muted-foreground">{t('perf.kpiQuery.noTemplates')}</div>
            ) : (
              filtered.map((tpl) => (
                <button
                  key={tpl.id}
                  type="button"
                  onClick={() => onSelect(tpl.id)}
                  className={cn(
                    'flex w-full flex-col items-start gap-1 rounded-md border px-3 py-2 text-left transition-colors hover:bg-accent',
                    selectedId === tpl.id && 'border-primary bg-accent'
                  )}
                >
                  <div className="flex w-full items-center justify-between gap-2">
                    <span className="truncate text-sm font-medium">{tpl.name}</span>
                    <Badge variant={tpl.visibility === 'public' ? 'success' : 'muted'}>
                      {tpl.visibility === 'public' ? t('perf.kpiQuery.public') : t('perf.kpiQuery.private')}
                    </Badge>
                  </div>
                  <span className="text-xs text-muted-foreground">
                    {t('perf.kpiQuery.deviceCount', { count: tpl.payload.deviceSns.length })} ·{' '}
                    {t('perf.kpiQuery.metricCount', { count: tpl.payload.metricPaths.length })} ·{' '}
                    {tpl.payload.granularity}
                  </span>
                </button>
              ))
            )}
          </div>
        </Card>

        {/* 查询结果 */}
        <div>
          {!selected ? (
            <Card className="flex h-64 items-center justify-center text-sm text-muted-foreground">
              {t('perf.kpiQuery.selectTemplateHint')}
            </Card>
          ) : (
            <div className="space-y-4">
              <Card className="p-4">
                <div className="mb-3 flex items-center justify-between gap-2">
                  <div>
                    <div className="text-sm font-medium">{selected.name}</div>
                    {selected.description && (
                      <div className="text-xs text-muted-foreground">{selected.description}</div>
                    )}
                  </div>
                  <Button
                    size="sm"
                    disabled={deviceSns.length === 0 || metricPaths.length === 0}
                    onClick={() => {
                      setRun(true)
                      if (run) agg.refetch()
                    }}
                  >
                    {t('perf.kpiQuery.runQuery')}
                  </Button>
                </div>
                <div className="flex flex-wrap gap-1.5 text-xs">
                  <Badge variant="outline">{t('perf.kpiQuery.granularityLabel', { value: granularity })}</Badge>
                  <Badge variant="outline">{t('perf.kpiQuery.deviceCount', { count: deviceSns.length })}</Badge>
                  <Badge variant="outline">{t('perf.kpiQuery.metricCount', { count: metricPaths.length })}</Badge>
                  <Badge variant="outline">{t('perf.kpiQuery.windowLabel', { value: selected.payload.timeRangePreset })}</Badge>
                </div>
                {deviceSns.length === 0 && (
                  <div className="mt-2 text-xs text-amber-600 dark:text-amber-400">
                    {t('perf.kpiQuery.noDeviceLinked')}
                  </div>
                )}
              </Card>

              {run && deviceSns.length > 0 && metricPaths.length > 0 ? (
                <ResultTable
                  rows={agg.data}
                  total={agg.total}
                  truncated={agg.truncated}
                  isLoading={agg.isLoading}
                  isError={agg.isError}
                  errors={agg.errors}
                  t={t}
                />
              ) : (
                <Card className="flex h-40 items-center justify-center text-sm text-muted-foreground">
                  {t('perf.kpiQuery.clickRunHint')}
                </Card>
              )}
            </div>
          )}
        </div>
      </div>
    </PageShell>
  )
}

export default KPIQueryPage
