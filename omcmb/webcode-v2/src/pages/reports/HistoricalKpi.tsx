import { useMemo, useState } from 'react'
import { RefreshCcw, LineChart as LineChartIcon } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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
  formatTime,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useAllKPIs, useMultipleKPISeries } from '@core/hooks/api/usePerformance'

// ============================================================
// 历史 KPI (report/historical-kpi) — v2
// 业务对齐 v1 webcode/src/pages/report/HistoricalKPI：多 KPI 折线下钻 + 明细表。
// 真实数据：
//   · useAllKPIs() → KPI 字典（kpiCode/kpiName/unit），供选择器。
//   · useMultipleKPISeries(codes, deviceSn) → 各 KPI 时序（/pm/kpi 真实端点）。
// 无 echarts，用 CSS 柱状近似呈现每条序列的趋势 + 明细表逐点展示。
// ============================================================

const DAYS_OPTIONS = [
  { value: '1', label: '近 1 天' },
  { value: '7', label: '近 7 天' },
  { value: '30', label: '近 30 天' },
]

const PALETTE = [
  'bg-sky-500',
  'bg-emerald-500',
  'bg-amber-500',
  'bg-violet-500',
  'bg-rose-500',
  'bg-cyan-500',
]

const PAGE_SIZE = 20

interface TableRowModel {
  timestamp: string
  values: Record<string, number | undefined>
}

export default function HistoricalKpi() {
  const [deviceSn, setDeviceSn] = useState('')
  const [days, setDays] = useState('7')
  const [selectedCodes, setSelectedCodes] = useState<string[]>([])
  const [page, setPage] = useState(1)

  const kpisQuery = useAllKPIs()
  const allKpis = useMemo(() => kpisQuery.data ?? [], [kpisQuery.data])

  // 默认选中前两个 KPI（一次性，依赖列表加载后）。
  const effectiveCodes = useMemo(() => {
    if (selectedCodes.length > 0) return selectedCodes
    return allKpis.slice(0, 2).map((k) => k.kpiCode)
  }, [selectedCodes, allKpis])

  const seriesQuery = useMultipleKPISeries(
    effectiveCodes,
    deviceSn.trim() || undefined,
  )
  // 按所选天数对返回序列做时间截断（hook 自身不接受 days 参数）。
  const series = useMemo(() => {
    const raw = seriesQuery.data ?? []
    const cutoff = Date.now() - Number(days) * 24 * 60 * 60 * 1000
    return raw.map((s) => ({
      ...s,
      data: s.data.filter((p) => {
        const t = new Date(p.timestamp).getTime()
        return Number.isNaN(t) ? true : t >= cutoff
      }),
    }))
  }, [seriesQuery.data, days])

  // KPI code → 显示名/单位映射。
  const codeMeta = useMemo(() => {
    const m = new Map<string, { name: string; unit: string }>()
    for (const k of allKpis) m.set(k.kpiCode, { name: k.kpiName, unit: k.unit })
    return m
  }, [allKpis])

  // 把多条序列对齐到统一时间轴（取并集），构造明细表。
  const tableRows: TableRowModel[] = useMemo(() => {
    const tsSet = new Set<string>()
    for (const s of series) for (const p of s.data) tsSet.add(p.timestamp)
    const timestamps = Array.from(tsSet).sort((a, b) => a.localeCompare(b))
    return timestamps.map((ts) => {
      const values: Record<string, number | undefined> = {}
      effectiveCodes.forEach((code, i) => {
        const s = series[i]
        const pt = s?.data.find((p) => p.timestamp === ts)
        values[code] = pt?.value
      })
      return { timestamp: ts, values }
    })
  }, [series, effectiveCodes])

  const total = tableRows.length
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const start = (page - 1) * PAGE_SIZE
  const paginated = tableRows.slice(start, start + PAGE_SIZE)

  const toggleCode = (code: string) => {
    setSelectedCodes((prev) => {
      const base = prev.length > 0 ? prev : allKpis.slice(0, 2).map((k) => k.kpiCode)
      return base.includes(code) ? base.filter((c) => c !== code) : [...base, code]
    })
    setPage(1)
  }

  const isFetching = kpisQuery.isFetching || seriesQuery.isFetching

  return (
    <PageShell
      title="历史 KPI"
      description={`多指标历史趋势 · ${effectiveCodes.length} 个 KPI · ${total} 个时间点`}
      isFetching={isFetching}
      toolbar={
        <>
          <div className="flex items-center gap-1.5">
            <Label className="text-xs text-muted-foreground">设备 SN</Label>
            <Input
              value={deviceSn}
              onChange={(e) => setDeviceSn(e.target.value)}
              placeholder="全部设备"
              className="w-40"
            />
          </div>
          <Select value={days} onValueChange={setDays}>
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {DAYS_OPTIONS.map((o) => (
                <SelectItem key={o.value} value={o.value}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={() => {
              void kpisQuery.refetch()
              void seriesQuery.refetch()
            }}
          >
            <RefreshCcw /> 刷新
          </Button>
        </>
      }
    >
      {/* KPI 选择器 */}
      <Card className="mb-4">
        <CardContent className="p-4">
          <div className="mb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
            选择 KPI 指标
          </div>
          {kpisQuery.isLoading ? (
            <div className="h-6 w-48 animate-pulse rounded bg-muted" />
          ) : kpisQuery.isError ? (
            <div className="text-sm text-destructive">
              KPI 字典加载失败：
              {kpisQuery.error instanceof Error ? kpisQuery.error.message : '未知错误'}
            </div>
          ) : allKpis.length === 0 ? (
            <div className="text-sm text-muted-foreground">暂无 KPI 指标</div>
          ) : (
            <div className="flex flex-wrap gap-1.5">
              {allKpis.map((k) => {
                const active = effectiveCodes.includes(k.kpiCode)
                return (
                  <button
                    key={k.kpiCode}
                    type="button"
                    onClick={() => toggleCode(k.kpiCode)}
                    className={cn(
                      'rounded-md border px-2.5 py-1 text-xs transition-colors',
                      active
                        ? 'border-primary/40 bg-primary/10 text-primary'
                        : 'text-muted-foreground hover:bg-muted',
                    )}
                  >
                    {k.kpiName}
                  </button>
                )
              })}
            </div>
          )}
        </CardContent>
      </Card>

      {/* 趋势(CSS 柱状近似) */}
      <Card className="mb-4">
        <CardContent className="p-4">
          <div className="mb-3 flex items-center gap-1.5 text-xs font-medium uppercase tracking-wider text-muted-foreground">
            <LineChartIcon className="size-3.5" /> 趋势概览
          </div>
          {seriesQuery.isLoading ? (
            <div className="h-32 animate-pulse rounded bg-muted" />
          ) : seriesQuery.isError ? (
            <div className="text-sm text-destructive">
              时序加载失败：
              {seriesQuery.error instanceof Error ? seriesQuery.error.message : '未知错误'}
            </div>
          ) : series.length === 0 || series.every((s) => s.data.length === 0) ? (
            <div className="py-8 text-center text-sm text-muted-foreground">
              所选 KPI 在该范围内暂无数据
            </div>
          ) : (
            <div className="space-y-4">
              {series.map((s, i) => {
                const code = effectiveCodes[i]
                const meta = code ? codeMeta.get(code) : undefined
                const max = Math.max(...s.data.map((p) => p.value), 1)
                return (
                  <div key={code ?? s.kpiName}>
                    <div className="mb-1 flex items-center justify-between text-xs">
                      <span className="flex items-center gap-1.5 font-medium">
                        <span className={cn('inline-block size-2 rounded-sm', PALETTE[i % PALETTE.length])} />
                        {meta?.name ?? s.kpiName}
                      </span>
                      <span className="text-muted-foreground">
                        {s.data.length} 点 · 峰值 {max.toFixed(2)}
                        {meta?.unit ? ` ${meta.unit}` : ''}
                      </span>
                    </div>
                    <div className="flex h-16 items-end gap-px">
                      {s.data.slice(-80).map((p, j) => (
                        <div
                          key={j}
                          className={cn('flex-1 rounded-t-sm', PALETTE[i % PALETTE.length])}
                          style={{ height: `${Math.max(2, (p.value / max) * 100)}%` }}
                          title={`${formatTime(p.timestamp)} · ${p.value}`}
                        />
                      ))}
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </CardContent>
      </Card>

      {/* 明细表 */}
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>时间</TableHead>
              {effectiveCodes.map((code) => (
                <TableHead key={code}>
                  {codeMeta.get(code)?.name ?? code}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {seriesQuery.isLoading ? (
              <LoadingRow colSpan={effectiveCodes.length + 1} />
            ) : seriesQuery.isError ? (
              <ErrorRow colSpan={effectiveCodes.length + 1} error={seriesQuery.error} />
            ) : paginated.length === 0 ? (
              <EmptyRow colSpan={effectiveCodes.length + 1}>暂无时序数据</EmptyRow>
            ) : (
              paginated.map((row) => (
                <TableRow key={row.timestamp}>
                  <TableCell className="font-mono text-xs">
                    {formatTime(row.timestamp)}
                  </TableCell>
                  {effectiveCodes.map((code) => {
                    const v = row.values[code]
                    return (
                      <TableCell key={code} className="tabular-nums">
                        {v !== undefined ? v.toFixed(3) : '—'}
                      </TableCell>
                    )
                  })}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={PAGE_SIZE}
        onChange={setPage}
      />
    </PageShell>
  )
}
