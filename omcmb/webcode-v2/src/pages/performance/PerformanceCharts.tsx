import { useMemo, useState } from 'react'
import { RefreshCcw } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { PageShell } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useMultipleKPISeries } from '@core/hooks/api/usePerformance'
import type { KPISeries } from '@core/types/performance'

// ============================================================
// 性能趋势图 — 对齐 v1 /performance/charts
//   多 KPI 时序曲线。v2 无 echarts，用 CSS 柱状近似（每个 KPI 一张卡）。
//   数据走真实 useMultipleKPISeries（/pm/kpi）。
// ============================================================

const KPI_OPTIONS: { label: string; value: string }[] = [
  { label: 'RRC 建立成功率', value: 'RRC_SR' },
  { label: 'E-RAB 建立成功率', value: 'ERAB_SR' },
  { label: '切换成功率', value: 'HO_SR' },
  { label: '下行吞吐量', value: 'DL_THROUGHPUT' },
  { label: '上行吞吐量', value: 'UL_THROUGHPUT' },
  { label: '最大用户数', value: 'MAX_USERS' },
  { label: '可用率', value: 'AVAILABILITY' },
]

const DAYS_OPTIONS: { label: string; value: string }[] = [
  { label: '近 1 天', value: '1' },
  { label: '近 7 天', value: '7' },
  { label: '近 30 天', value: '30' },
]

function shortTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function MiniBarChart({ series }: { series: KPISeries }) {
  const pts = series.data
  const max = pts.reduce((m, p) => Math.max(m, p.value), 0)
  const min = pts.reduce((m, p) => Math.min(m, p.value), pts.length ? pts[0].value : 0)
  const span = max - min || 1

  if (pts.length === 0) {
    return (
      <Card className="p-4">
        <div className="mb-1 text-sm font-medium">{series.kpiName}</div>
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
        <div className="text-sm font-medium">{series.kpiName}</div>
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
  const [selected, setSelected] = useState<string[]>(['RRC_SR', 'DL_THROUGHPUT'])
  const [days, setDays] = useState('7')

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
        <div className="mb-2 text-xs uppercase tracking-wider text-muted-foreground">
          选择 KPI 指标
        </div>
        <div className="flex flex-wrap gap-1.5">
          {KPI_OPTIONS.map((o) => {
            const active = selected.includes(o.value)
            return (
              <Button
                key={o.value}
                size="sm"
                variant={active ? 'default' : 'outline'}
                onClick={() => toggle(o.value)}
              >
                {o.label}
              </Button>
            )
          })}
        </div>
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
            <MiniBarChart key={`${s.kpiName}-${i}`} series={s} />
          ))}
        </div>
      )}
    </PageShell>
  )
}

export default PerformanceChartsPage
