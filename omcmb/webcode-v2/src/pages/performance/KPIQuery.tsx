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
import type { QueryTemplate } from '@core/types/pmQuery'
import type { AggregatedRow, Granularity } from '@core/types/pmDashboard'

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

// 聚合行透视：行键=time，列=metricPath（用 displayName 展示），单元格=metricValue。
function pivot(rows: AggregatedRow[]): {
  times: string[]
  metrics: { path: string; label: string }[]
  cell: Map<string, number | null>
} {
  const times: string[] = []
  const timeSeen = new Set<string>()
  const metricMap = new Map<string, string>()
  const cell = new Map<string, number | null>()
  for (const r of rows) {
    if (!timeSeen.has(r.time)) {
      timeSeen.add(r.time)
      times.push(r.time)
    }
    if (!metricMap.has(r.metricPath)) {
      metricMap.set(r.metricPath, r.displayName || r.metricPath)
    }
    cell.set(`${r.time}|${r.metricPath}`, r.metricValue)
  }
  times.sort((a, b) => a.localeCompare(b))
  const metrics = Array.from(metricMap.entries()).map(([path, label]) => ({ path, label }))
  return { times, metrics, cell }
}

function ResultTable({
  rows,
  total,
  truncated,
  isLoading,
  isError,
  errors,
}: {
  rows: AggregatedRow[]
  total: number
  truncated: boolean
  isLoading: boolean
  isError: boolean
  errors: unknown[]
}) {
  const { times, metrics, cell } = useMemo(() => pivot(rows), [rows])
  const colCount = metrics.length + 1

  return (
    <div>
      <div className="mb-2 flex items-center gap-2 text-xs text-muted-foreground">
        <span>共 {total} 行</span>
        {truncated && <Badge variant="warning">结果已截断</Badge>}
      </div>
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>时间</TableHead>
              {metrics.map((m) => (
                <TableHead key={m.path}>{m.label}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={errors[0] ?? new Error('查询失败')} />
            ) : times.length === 0 ? (
              <EmptyRow colSpan={colCount}>暂无查询结果</EmptyRow>
            ) : (
              times.map((time) => (
                <TableRow key={time}>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(time)}
                  </TableCell>
                  {metrics.map((m) => {
                    const v = cell.get(`${time}|${m.path}`)
                    return (
                      <TableCell key={m.path} className="tabular-nums">
                        {v === null || v === undefined ? '—' : v}
                      </TableCell>
                    )
                  })}
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
      title="指标查询"
      description="基于保存的查询模板跑多设备聚合查询，结果透视为时间 × 指标表"
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
                placeholder="模板名称"
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
              />
            </div>
            <Button
              variant="outline"
              size="sm"
              disabled={tplFetching}
              onClick={() => void refetchTpl()}
              aria-label="刷新模板"
            >
              <RefreshCcw className="size-4" />
            </Button>
          </div>
          <div className="max-h-[560px] space-y-1 overflow-auto">
            {tplLoading ? (
              <div className="p-6 text-center text-sm text-muted-foreground">加载模板…</div>
            ) : tplError ? (
              <div className="p-6 text-center text-sm text-destructive">
                加载失败：{tplErr instanceof Error ? tplErr.message : '未知错误'}
              </div>
            ) : filtered.length === 0 ? (
              <div className="p-6 text-center text-sm text-muted-foreground">暂无查询模板</div>
            ) : (
              filtered.map((t) => (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => onSelect(t.id)}
                  className={cn(
                    'flex w-full flex-col items-start gap-1 rounded-md border px-3 py-2 text-left transition-colors hover:bg-accent',
                    selectedId === t.id && 'border-primary bg-accent'
                  )}
                >
                  <div className="flex w-full items-center justify-between gap-2">
                    <span className="truncate text-sm font-medium">{t.name}</span>
                    <Badge variant={t.visibility === 'public' ? 'success' : 'muted'}>
                      {t.visibility === 'public' ? '公开' : '私有'}
                    </Badge>
                  </div>
                  <span className="text-xs text-muted-foreground">
                    {t.payload.deviceSns.length} 设备 · {t.payload.metricPaths.length} 指标 ·{' '}
                    {t.payload.granularity}
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
              请选择左侧一个查询模板
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
                    运行查询
                  </Button>
                </div>
                <div className="flex flex-wrap gap-1.5 text-xs">
                  <Badge variant="outline">粒度 {granularity}</Badge>
                  <Badge variant="outline">{deviceSns.length} 设备</Badge>
                  <Badge variant="outline">{metricPaths.length} 指标</Badge>
                  <Badge variant="outline">时窗 {selected.payload.timeRangePreset}</Badge>
                </div>
                {deviceSns.length === 0 && (
                  <div className="mt-2 text-xs text-amber-600 dark:text-amber-400">
                    模板未关联设备，无法运行查询。
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
                />
              ) : (
                <Card className="flex h-40 items-center justify-center text-sm text-muted-foreground">
                  点击「运行查询」加载聚合结果
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
