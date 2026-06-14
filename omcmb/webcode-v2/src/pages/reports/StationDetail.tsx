import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, RefreshCcw, Server } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { PageShell } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useReportSampleData } from '@core/hooks/api/useReports'

// ============================================================
// 单站报表 — 区域明细 (/reports/station/:region) — v2
// 真实数据：useReportSampleData() → deviceSummary.byRegion[region] 设备数 +
//   kpiSummary 全网 KPI 摘要 + byType 设备类型分布。以 :region 参数取对应区域。
// ============================================================

function asRecord(v: unknown): Record<string, unknown> | undefined {
  return v && typeof v === 'object' ? (v as Record<string, unknown>) : undefined
}
function asNumber(v: unknown): number | undefined {
  return typeof v === 'number' && Number.isFinite(v) ? v : undefined
}

interface KpiCellModel {
  label: string
  value: number
  unit?: string
  trend?: number
}

function buildKpis(kpiSummary: Record<string, unknown> | undefined): KpiCellModel[] {
  if (!kpiSummary) return []
  const defs: { key: string; label: string }[] = [
    { key: 'rrcSuccRate', label: 'RRC 建立成功率' },
    { key: 'erabSuccRate', label: 'E-RAB 成功率' },
    { key: 'hoSuccRate', label: '切换成功率' },
    { key: 'dlThroughput', label: '下行吞吐' },
    { key: 'ulThroughput', label: '上行吞吐' },
    { key: 'radioDrop', label: '无线掉线率' },
  ]
  const out: KpiCellModel[] = []
  for (const d of defs) {
    const r = asRecord(kpiSummary[d.key])
    const value = asNumber(r?.value)
    if (value === undefined) continue
    out.push({
      label: d.label,
      value,
      unit: typeof r?.unit === 'string' ? r.unit : undefined,
      trend: asNumber(r?.trend),
    })
  }
  return out
}

export default function StationDetail() {
  const { region = '' } = useParams<{ region: string }>()
  const decodedRegion = decodeURIComponent(region)
  const navigate = useNavigate()

  const sampleQuery = useReportSampleData()
  const sample = sampleQuery.data as Record<string, unknown> | undefined

  const deviceSummary = asRecord(sample?.deviceSummary)
  const byRegion = asRecord(deviceSummary?.byRegion)
  const byType = asRecord(deviceSummary?.byType)
  const kpiSummary = asRecord(sample?.kpiSummary)

  const deviceCount = byRegion ? asNumber(byRegion[decodedRegion]) : undefined
  const kpis = useMemo(() => buildKpis(kpiSummary), [kpiSummary])

  const isLoading = sampleQuery.isLoading
  const isError = sampleQuery.isError
  // 区域在 byRegion 里没有键 → 视为无该区域数据；但仍渲染真实摘要，不恒为"未找到"。
  const regionKnown = byRegion ? decodedRegion in byRegion : false

  return (
    <PageShell
      title={`区域明细 · ${decodedRegion}`}
      description={
        deviceCount !== undefined
          ? `${decodedRegion} 区域 ${deviceCount} 台设备`
          : decodedRegion
      }
      isFetching={sampleQuery.isFetching}
      toolbar={
        <>
          <Button variant="outline" size="sm" onClick={() => navigate(-1)}>
            <ArrowLeft /> 返回
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={() => void sampleQuery.refetch()}
          >
            <RefreshCcw /> 刷新
          </Button>
        </>
      }
    >
      {isLoading ? (
        <Card>
          <CardContent className="p-6">
            <div className="h-4 w-40 animate-pulse rounded bg-muted" />
            <div className="mt-3 h-4 w-64 animate-pulse rounded bg-muted" />
          </CardContent>
        </Card>
      ) : isError ? (
        <Card>
          <CardContent className="p-6 text-destructive">
            加载失败：
            {sampleQuery.error instanceof Error ? sampleQuery.error.message : '未知错误'}
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          <Card>
            <CardContent className="p-6">
              <div className="flex items-center gap-2">
                <Server className="size-5 text-primary" />
                <h2 className="text-lg font-semibold">{decodedRegion}</h2>
                {regionKnown ? (
                  <Badge variant="success">在网</Badge>
                ) : (
                  <Badge variant="muted">无区域统计</Badge>
                )}
              </div>
              <div className="mt-4 grid grid-cols-2 gap-4 md:grid-cols-4">
                <div className="rounded-lg border bg-card px-4 py-3">
                  <div className="text-xs uppercase tracking-wider text-muted-foreground">
                    设备数
                  </div>
                  <div className="mt-1 text-2xl font-semibold tabular-nums">
                    {deviceCount ?? '—'}
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* 全网 KPI 摘要（区域代表值） */}
          {kpis.length > 0 && (
            <Card>
              <CardContent className="p-6">
                <div className="mb-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">
                  KPI 摘要
                </div>
                <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
                  {kpis.map((k) => {
                    const trendClass =
                      k.trend === undefined
                        ? 'text-muted-foreground'
                        : k.trend > 0
                          ? 'text-emerald-600 dark:text-emerald-400'
                          : k.trend < 0
                            ? 'text-rose-600 dark:text-rose-400'
                            : 'text-muted-foreground'
                    return (
                      <div key={k.label} className="rounded-lg border bg-card px-3 py-2">
                        <div className="truncate text-xs text-muted-foreground" title={k.label}>
                          {k.label}
                        </div>
                        <div className="mt-1 flex items-baseline gap-1">
                          <span className="text-lg font-semibold tabular-nums">{k.value}</span>
                          {k.unit && (
                            <span className="text-xs text-muted-foreground">{k.unit}</span>
                          )}
                        </div>
                        {k.trend !== undefined && (
                          <div className={cn('text-xs tabular-nums', trendClass)}>
                            {k.trend > 0 ? '+' : ''}
                            {k.trend}
                          </div>
                        )}
                      </div>
                    )
                  })}
                </div>
              </CardContent>
            </Card>
          )}

          {/* 设备类型分布 */}
          {byType && Object.keys(byType).length > 0 && (
            <Card>
              <CardContent className="p-6">
                <div className="mb-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">
                  设备类型分布(全网)
                </div>
                <div className="flex flex-wrap gap-2">
                  {Object.entries(byType).map(([type, count]) => (
                    <Badge key={type} variant="outline">
                      {type} · {asNumber(count) ?? 0}
                    </Badge>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      )}
    </PageShell>
  )
}
