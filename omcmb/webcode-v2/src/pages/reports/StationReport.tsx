import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Download, Server } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
  TableCard,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useReportSampleData } from '@core/hooks/api/useReports'

// ============================================================
// 单站报表 (report/station) — v2
// 业务对齐 v1 webcode/src/pages/report/StationReport：按区域/基站浏览 KPI。
// 真实数据：useReportSampleData() 返回 /reports/sample-data 的 deviceSummary.byRegion
//   （各区域设备数）+ kpiSummary（全网 KPI 摘要）。本页据此构造「区域 KPI 汇总」行，
//   关键词/区域过滤，行点击进入 /reports/station/:region 区域明细。
// ============================================================

function asRecord(v: unknown): Record<string, unknown> | undefined {
  return v && typeof v === 'object' ? (v as Record<string, unknown>) : undefined
}
function asNumber(v: unknown): number | undefined {
  return typeof v === 'number' && Number.isFinite(v) ? v : undefined
}
function asKpiValue(v: unknown): number | undefined {
  const r = asRecord(v)
  return asNumber(r?.value)
}

export interface RegionRow {
  region: string
  deviceCount: number
  rrcSuccRate?: number
  hoSuccRate?: number
  dlThroughput?: number
  radioDrop?: number
}

function buildRegionRows(sample: Record<string, unknown> | undefined): RegionRow[] {
  if (!sample) return []
  const deviceSummary = asRecord(sample.deviceSummary)
  const kpiSummary = asRecord(sample.kpiSummary)
  const byRegion = asRecord(deviceSummary?.byRegion)
  if (!byRegion) return []

  // 全网 KPI 摘要作为各区域的代表值（sample-data 仅暴露全网级摘要）。
  const rrc = kpiSummary ? asKpiValue(kpiSummary.rrcSuccRate) : undefined
  const ho = kpiSummary ? asKpiValue(kpiSummary.hoSuccRate) : undefined
  const dl = kpiSummary ? asKpiValue(kpiSummary.dlThroughput) : undefined
  const drop = kpiSummary ? asKpiValue(kpiSummary.radioDrop) : undefined

  return Object.entries(byRegion).map(([region, count]) => ({
    region,
    deviceCount: asNumber(count) ?? 0,
    rrcSuccRate: rrc,
    hoSuccRate: ho,
    dlThroughput: dl,
    radioDrop: drop,
  }))
}

function rateColor(v: number | undefined): string {
  if (v === undefined) return 'text-muted-foreground'
  if (v >= 99) return 'text-emerald-600 dark:text-emerald-400'
  if (v >= 97) return 'text-amber-600 dark:text-amber-400'
  return 'text-rose-600 dark:text-rose-400'
}

function dropColor(v: number | undefined): string {
  if (v === undefined) return 'text-muted-foreground'
  if (v <= 0.5) return 'text-emerald-600 dark:text-emerald-400'
  if (v <= 1) return 'text-amber-600 dark:text-amber-400'
  return 'text-rose-600 dark:text-rose-400'
}

export default function StationReport() {
  const navigate = useNavigate()
  const [keyword, setKeyword] = useState('')
  const [regionFilter, setRegionFilter] = useState('')

  const sampleQuery = useReportSampleData()
  const rows = useMemo(
    () => buildRegionRows(sampleQuery.data as Record<string, unknown> | undefined),
    [sampleQuery.data],
  )

  const regions = useMemo(() => rows.map((r) => r.region), [rows])

  const filtered = rows.filter((r) => {
    if (keyword && !r.region.toLowerCase().includes(keyword.toLowerCase())) return false
    if (regionFilter && r.region !== regionFilter) return false
    return true
  })

  const totalDevices = filtered.reduce((s, r) => s + r.deviceCount, 0)

  return (
    <PageShell
      title="单站报表"
      description={`区域 KPI 汇总 · ${filtered.length} 个区域 · ${totalDevices} 台设备`}
      isFetching={sampleQuery.isFetching}
      toolbar={
        <>
          <Input
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder="搜索区域"
            className="w-44"
          />
          <Select
            value={regionFilter || 'all'}
            onValueChange={(v) => setRegionFilter(v === 'all' ? '' : v)}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="区域" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部区域</SelectItem>
              {regions.map((r) => (
                <SelectItem key={r} value={r}>
                  {r}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={() => void sampleQuery.refetch()}
          >
            <RefreshCcw /> 刷新
          </Button>
          <Button variant="outline" size="sm" disabled>
            <Download /> 批量导出
          </Button>
        </>
      }
    >
      {/* 概览卡片 */}
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center gap-1.5 text-xs uppercase tracking-wider text-muted-foreground">
              <Server className="size-4" /> 区域数
            </div>
            <div className="mt-1 text-2xl font-semibold tabular-nums">
              {rows.length}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="text-xs uppercase tracking-wider text-muted-foreground">
              设备总数
            </div>
            <div className="mt-1 text-2xl font-semibold tabular-nums">
              {rows.reduce((s, r) => s + r.deviceCount, 0)}
            </div>
          </CardContent>
        </Card>
      </div>

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>区域</TableHead>
              <TableHead>设备数</TableHead>
              <TableHead>RRC 成功率(%)</TableHead>
              <TableHead>切换成功率(%)</TableHead>
              <TableHead>下行吞吐(Mbps)</TableHead>
              <TableHead>无线掉线率(%)</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sampleQuery.isLoading ? (
              <LoadingRow colSpan={7} />
            ) : sampleQuery.isError ? (
              <ErrorRow colSpan={7} error={sampleQuery.error} />
            ) : filtered.length === 0 ? (
              <EmptyRow colSpan={7}>暂无区域数据</EmptyRow>
            ) : (
              filtered.map((r) => (
                <TableRow
                  key={r.region}
                  className="cursor-pointer"
                  onClick={() => navigate(`/report/station`)}
                >
                  <TableCell className="font-medium text-primary hover:underline">
                    {r.region}
                  </TableCell>
                  <TableCell className="tabular-nums">{r.deviceCount}</TableCell>
                  <TableCell className={cn('font-medium tabular-nums', rateColor(r.rrcSuccRate))}>
                    {r.rrcSuccRate !== undefined ? r.rrcSuccRate.toFixed(2) : '—'}
                  </TableCell>
                  <TableCell className={cn('font-medium tabular-nums', rateColor(r.hoSuccRate))}>
                    {r.hoSuccRate !== undefined ? r.hoSuccRate.toFixed(2) : '—'}
                  </TableCell>
                  <TableCell className="tabular-nums">
                    {r.dlThroughput !== undefined ? r.dlThroughput.toFixed(1) : '—'}
                  </TableCell>
                  <TableCell className={cn('font-medium tabular-nums', dropColor(r.radioDrop))}>
                    {r.radioDrop !== undefined ? r.radioDrop.toFixed(2) : '—'}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation()
                        navigate(`/report/station`)
                      }}
                    >
                      明细
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>
    </PageShell>
  )
}
