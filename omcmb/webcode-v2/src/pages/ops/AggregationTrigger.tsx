import { useState } from 'react'
import { Zap, CheckCircle2, Loader2, Info } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Card } from '@/components/ui/card'
import { PageShell } from '@/components/layout/PageShell'

import { useTriggerRecompute } from '@core/hooks/api/usePmAggregation'
import type { RecomputeAggregationInput } from '@core/services/api/pmAggregationApi'

// ============================================================
// PM 聚合手动触发 — 对齐 v1 webcode/src/pages/ops/AggregationTrigger
// 表单（粒度 + 维度 + 时间窗）→ recompute 入队 async_jobs，返回 job_id
// ============================================================

type Granularity = RecomputeAggregationInput['granularity']
type Dimension = RecomputeAggregationInput['dimension']

const GRANULARITY_OPTIONS: { label: string; value: Granularity }[] = [
  { label: '小时 (hourly)', value: 'hourly' },
  { label: '日 (daily)', value: 'daily' },
  { label: '周 (weekly)', value: 'weekly' },
  { label: '月 (monthly)', value: 'monthly' },
]

const DIMENSION_OPTIONS: { label: string; value: Dimension }[] = [
  { label: '设备 (device)', value: 'device' },
  { label: '设备组 (device_group)', value: 'device_group' },
]

// datetime-local 值（本地时区，无秒）→ RFC3339 ISO
function toISO(local: string): string {
  if (!local) return ''
  const d = new Date(local)
  return Number.isNaN(d.getTime()) ? '' : d.toISOString()
}

// 默认时间窗：近一天 → 现在，格式化为 datetime-local 期望的 YYYY-MM-DDTHH:mm（本地时区）
function toLocalInput(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export default function AggregationTrigger() {
  const trigger = useTriggerRecompute()
  const [lastJobId, setLastJobId] = useState<string | null>(null)
  const [warning, setWarning] = useState<string | null>(null)

  const [granularity, setGranularity] = useState<Granularity>('hourly')
  const [dimension, setDimension] = useState<Dimension>('device')
  const now = new Date()
  const dayAgo = new Date(now.getTime() - 24 * 3600 * 1000)
  const [start, setStart] = useState(toLocalInput(dayAgo))
  const [end, setEnd] = useState(toLocalInput(now))

  const startISO = toISO(start)
  const endISO = toISO(end)
  const valid = Boolean(startISO && endISO && new Date(startISO) < new Date(endISO))

  const handleSubmit = () => {
    if (!valid) {
      setWarning('请选择有效的时间窗（开始时间需早于结束时间）')
      return
    }
    setWarning(null)
    setLastJobId(null)
    trigger.mutate(
      { granularity, dimension, start: startISO, end: endISO },
      {
        onSuccess: (resp) => {
          const jobId = (resp.job_id ?? resp.id ?? '') as string
          if (jobId) {
            setLastJobId(jobId)
          } else {
            setWarning('提交成功但响应未返回 job_id，请用 SQL 查 async_jobs 最新行')
          }
        },
        onError: (e) => {
          setWarning(e instanceof Error ? e.message : '入队失败')
        },
      }
    )
  }

  return (
    <PageShell
      title="PM 聚合手动触发"
      description="按粒度 + 维度 + 时间窗触发 recompute（后端 worker 异步执行）"
    >
      <div className="flex max-w-3xl flex-col gap-4">
        <Card className="flex gap-3 border-primary/30 bg-primary/5 p-4 text-sm">
          <Info className="mt-0.5 size-4 shrink-0 text-primary" />
          <div className="text-muted-foreground">
            <div className="mb-1 font-medium text-foreground">使用场景</div>
            <ul className="list-disc space-y-1 pl-4">
              <li>历史时段补算（如某段时间 worker 故障未跑聚合）</li>
              <li>QA 验证聚合结果一致性</li>
              <li>本工具不直接跑聚合，而是入队 async_jobs；worker 进程会抢任务执行</li>
              <li>
                暂无状态查询页，返回 job_id 后请用 SQL 查{' '}
                <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">async_jobs</code> 表
              </li>
            </ul>
          </div>
        </Card>

        <Card className="flex flex-col gap-5 p-5">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="granularity">聚合粒度</Label>
            <Select value={granularity} onValueChange={(v) => setGranularity(v as Granularity)}>
              <SelectTrigger id="granularity" className="w-full max-w-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {GRANULARITY_OPTIONS.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="dimension">维度</Label>
            <Select value={dimension} onValueChange={(v) => setDimension(v as Dimension)}>
              <SelectTrigger id="dimension" className="w-full max-w-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {DIMENSION_OPTIONS.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="start">开始时间</Label>
              <Input
                id="start"
                type="datetime-local"
                value={start}
                onChange={(e) => setStart(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="end">结束时间</Label>
              <Input
                id="end"
                type="datetime-local"
                value={end}
                onChange={(e) => setEnd(e.target.value)}
              />
            </div>
          </div>

          {warning ? <div className="text-sm text-destructive">{warning}</div> : null}

          <div>
            <Button disabled={!valid || trigger.isPending} onClick={handleSubmit}>
              {trigger.isPending ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <Zap className="size-4" />
              )}
              入队聚合任务
            </Button>
          </div>
        </Card>

        {lastJobId ? (
          <Card className="flex flex-col gap-3 border-emerald-500/30 bg-emerald-500/5 p-5">
            <div className="flex items-center gap-2 text-emerald-600 dark:text-emerald-400">
              <CheckCircle2 className="size-5" />
              <span className="font-medium">任务已入队</span>
            </div>
            <div className="text-sm">
              <span className="text-muted-foreground">job_id: </span>
              <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">{lastJobId}</code>
            </div>
            <div className="text-xs text-muted-foreground">状态查询 SQL（无 UI 暂用）：</div>
            <pre className="overflow-x-auto rounded-md border bg-muted/40 p-3 font-mono text-xs">
              {`docker exec omc-docker-postgres-1 psql -U omcgo -d omcgo -c "
SELECT id, status, started_at, finished_at, error_msg
FROM async_jobs WHERE id='${lastJobId}'"`}
            </pre>
          </Card>
        ) : null}
      </div>
    </PageShell>
  )
}
