import { useEffect, useMemo, useState } from 'react'
import { RefreshCcw } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { PageShell } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useAllKPIs, useMultipleKPISeries } from '@core/hooks/api/usePerformance'
import type { KPISeries } from '@core/types/performance'
import { formatSystemTime } from '@core/utils/systemTime'

// ============================================================
// 性能趋势图 — 对齐 v1 /performance/charts
//   多 KPI 时序曲线。v2 无 echarts，用 CSS 柱状近似（每个 KPI 一张卡）。
//   KPI 目录走真实 useAllKPIs（/pm/kpi/definitions），时序走真实
//   useMultipleKPISeries（/pm/kpi 按 kpi_name=指标编号 查询）。
// ============================================================

const DAYS_OPTIONS: { label: string; value: string }[] = [
  { label: '近 1 天', value: '1' },
  { label: '近 7 天', value: '7' },
  { label: '近 30 天', value: '30' },
]

// 目录就绪后默认勾选的 KPI 条数。
const DEFAULT_KPI_COUNT = 4

function shortTime(iso: string): string {
  // #459 子单 D：图表轴标签保留系统时区钟面（MM-DD HH:mm），不按浏览器本地转换。
  return formatSystemTime(iso, { format: 'MM-DD HH:mm', placeholder: iso })
}

function MiniBarChart({ series, name }: { series: KPISeries; name: string }) {
  const pts = series.data
  const max = pts.reduce((m, p) => Math.max(m, p.value), 0)
  const min = pts.reduce((m, p) => Math.min(m, p.value), pts.length ? pts[0].value : 0)
  const span = max - min || 1

  if (pts.length === 0) {
    return (
      <Card className="p-4">
        <div className="mb-1 text-sm font-medium">{name}</div>
        <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
          暂无数据
        </div>
      </Card>
    )
  }

  const last = pts[pts.length - 1]

  return (
    <Card className="p-4">
      <div className="mb-2 flex items-baseline justify-between">
        <div className="text-sm font-medium">{name}</div>
        <div className="text-xs tabular-nums text-muted-foreground">
          最新 {last.value.toFixed(2)}
          {series.unit}
        </div>
      </div>
      <div className="flex h-40 items-end gap-px">
        {pts.map((p, i) => {
          const h = 8 + ((p.value - min) / span) * 88
          return (
            <div
              key={`${p.timestamp}-${i}`}
              className="flex-1 rounded-t bg-primary/70 transition-all hover:bg-primary"
              style={{ height: `${h}%` }}
              title={`${shortTime(p.timestamp)} · ${p.value}${series.unit}`}
            />
          )
        })}
      </div>
      <div className="mt-2 flex justify-between text-[10px] text-muted-foreground">
        <span>{shortTime(pts[0].timestamp)}</span>
        <span>共 {pts.length} 点</span>
        <span>{shortTime(last.timestamp)}</span>
      </div>
    </Card>
  )
}

export function PerformanceChartsPage() {
  const [selected, setSelected] = useState<string[]>([])
  const [days, setDays] = useState('7')
  const [keyword, setKeyword] = useState('')

  // 真实 KPI 目录（/pm/kpi/definitions）：kpiCode=指标编号(K…)，kpiName=中文显示名。
  const { data: kpiData, isLoading: kpiLoading, isError: kpiError } = useAllKPIs()
  const catalog = useMemo(() => kpiData ?? [], [kpiData])

  const filteredCatalog = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return catalog
    return catalog.filter(
      (k) => k.kpiName.toLowerCase().includes(kw) || k.kpiCode.toLowerCase().includes(kw),
    )
  }, [catalog, keyword])

  // kpiCode → 显示名 映射，给图表卡片标题用真实中文名而非裸编号。
  const codeToName = useMemo(() => {
    const m: Record<string, string> = {}
    catalog.forEach((k) => {
      m[k.kpiCode] = k.kpiName || k.kpiCode
    })
    return m
  }, [catalog])

  // 目录就绪后默认勾选前 N 条真实 KPI（默认即出真实图）。
  useEffect(() => {
    if (catalog.length === 0) return
    setSelected((prev) =>
      prev.length > 0 ? prev : catalog.slice(0, DEFAULT_KPI_COUNT).map((k) => k.kpiCode),
    )
  }, [catalog])

  const { data, isLoading, isError, error, isFetching, refetch } = useMultipleKPISeries(selected)
  const seriesList = data ?? []

  const toggle = (code: string) => {
    setSelected((prev) =>
      prev.includes(code) ? prev.filter((c) => c !== code) : [...prev, code]
    )
  }

  const daysNum = useMemo(() => Number(days), [days])

  return (
    <PageShell
      title="性能趋势图"
      description={`KPI 时序曲线 · 当前窗口近 ${daysNum} 天`}
      isFetching={isFetching}
      toolbar={
        <div className="flex flex-wrap items-center gap-2">
          <Label className="text-xs text-muted-foreground">时间窗</Label>
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
            disabled={isFetching}
            onClick={() => void refetch()}
          >
            <RefreshCcw className="size-4" /> 刷新
          </Button>
        </div>
      }
    >
      <Card className="mb-4 p-4">
        <div className="mb-2 flex items-center justify-between gap-3">
          <div className="text-xs uppercase tracking-wider text-muted-foreground">
            选择 KPI 指标（{selected.length} 已选）
          </div>
          <Input
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder="搜索 KPI 名称 / 编码"
            className="h-8 w-56"
          />
        </div>
        {kpiLoading ? (
          <div className="py-4 text-sm text-muted-foreground">加载 KPI 目录…</div>
        ) : kpiError ? (
          <div className="py-4 text-sm text-destructive">KPI 目录加载失败</div>
        ) : filteredCatalog.length === 0 ? (
          <div className="py-4 text-sm text-muted-foreground">无匹配 KPI</div>
        ) : (
          <div className="flex max-h-48 flex-wrap gap-1.5 overflow-auto">
            {filteredCatalog.map((k) => {
              const active = selected.includes(k.kpiCode)
              return (
                <Button
                  key={k.kpiCode}
                  size="sm"
                  variant={active ? 'default' : 'outline'}
                  onClick={() => toggle(k.kpiCode)}
                  title={k.kpiCode}
                >
                  {k.kpiName || k.kpiCode}
                </Button>
              )
            })}
          </div>
        )}
      </Card>

      {selected.length === 0 ? (
        <Card className="p-10 text-center text-sm text-muted-foreground">
          请至少选择一个 KPI 指标
        </Card>
      ) : isLoading ? (
        <Card className="p-10 text-center text-sm text-muted-foreground">加载曲线数据…</Card>
      ) : isError ? (
        <Card className="p-10 text-center text-sm text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </Card>
      ) : (
        <div className={cn('grid grid-cols-1 gap-4 lg:grid-cols-2', isFetching && 'opacity-70')}>
          {seriesList.map((s, i) => (
            <MiniBarChart
              key={`${s.kpiName}-${i}`}
              series={s}
              name={codeToName[s.kpiName] ?? s.kpiName}
            />
          ))}
        </div>
      )}
    </PageShell>
  )
}

export default PerformanceChartsPage
