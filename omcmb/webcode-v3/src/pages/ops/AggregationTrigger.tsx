import { useState } from 'react'
import { CheckCircle2, Copy, Loader2, Zap } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { useTriggerRecompute } from '@core/hooks/api/usePmAggregation'

import { FormRow } from './_shared'

type Granularity = 'hourly' | 'daily' | 'weekly' | 'monthly'
type Dimension = 'device' | 'device_group'

const GRANULARITY_OPTIONS: { value: Granularity; label: string }[] = [
  { value: 'hourly', label: '小时 hourly' },
  { value: 'daily', label: '日 daily' },
  { value: 'weekly', label: '周 weekly' },
  { value: 'monthly', label: '月 monthly' },
]
const DIMENSION_OPTIONS: { value: Dimension; label: string }[] = [
  { value: 'device', label: '设备 device' },
  { value: 'device_group', label: '设备组 device_group' },
]

// datetime-local 默认值：昨天此刻 → 现在
function toLocalInput(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export default function AggregationTrigger() {
  const trigger = useTriggerRecompute()

  const now = new Date()
  const yesterday = new Date(now.getTime() - 24 * 3600 * 1000)
  const [granularity, setGranularity] = useState<Granularity>('hourly')
  const [dimension, setDimension] = useState<Dimension>('device')
  const [start, setStart] = useState(toLocalInput(yesterday))
  const [end, setEnd] = useState(toLocalInput(now))
  const [lastJobId, setLastJobId] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const submit = () => {
    const startDate = new Date(start)
    const endDate = new Date(end)
    if (Number.isNaN(startDate.getTime()) || Number.isNaN(endDate.getTime())) return
    trigger.mutate(
      {
        granularity,
        dimension,
        start: startDate.toISOString(),
        end: endDate.toISOString(),
      },
      {
        onSuccess: (resp) => {
          const jobId = (resp.job_id ?? resp.id ?? '') as string
          setLastJobId(jobId || null)
          setCopied(false)
        },
      },
    )
  }

  const copyJobId = () => {
    if (!lastJobId) return
    void navigator.clipboard?.writeText(lastJobId)
    setCopied(true)
  }

  return (
    <PageShell
      code="F03"
      title="OPS · 聚合触发"
      subtitle="PM AGGREGATION RECOMPUTE · ENQUEUE ASYNC JOB"
      bare
    >
      <div className="grid gap-4 lg:grid-cols-[minmax(0,520px)_minmax(0,1fr)]">
        <GlassPanel title="触发参数 · PARAMETERS">
          <div className="space-y-4 p-4">
            <FormRow label="聚合粒度 · GRANULARITY">
              <div className="flex flex-wrap gap-1.5">
                {GRANULARITY_OPTIONS.map((o) => (
                  <button
                    key={o.value}
                    type="button"
                    onClick={() => setGranularity(o.value)}
                    className={`chip text-[#00f0ff] ${granularity === o.value ? 'shadow-[0_0_8px_currentColor]' : 'opacity-55'}`}
                  >
                    {o.label}
                  </button>
                ))}
              </div>
            </FormRow>
            <FormRow label="维度 · DIMENSION">
              <div className="flex flex-wrap gap-1.5">
                {DIMENSION_OPTIONS.map((o) => (
                  <button
                    key={o.value}
                    type="button"
                    onClick={() => setDimension(o.value)}
                    className={`chip text-[#a855f7] ${dimension === o.value ? 'shadow-[0_0_8px_currentColor]' : 'opacity-55'}`}
                  >
                    {o.label}
                  </button>
                ))}
              </div>
            </FormRow>
            <FormRow label="起始时间 · START">
              <input
                className="neon-input w-full"
                type="datetime-local"
                value={start}
                onChange={(e) => setStart(e.target.value)}
              />
            </FormRow>
            <FormRow label="结束时间 · END">
              <input
                className="neon-input w-full"
                type="datetime-local"
                value={end}
                onChange={(e) => setEnd(e.target.value)}
              />
            </FormRow>

            {trigger.isError ? (
              <div className="border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-[11px] text-rose-300">
                入队失败，请确认时间窗合法且后端可用后重试
              </div>
            ) : null}

            <div className="pt-1">
              <NeonButton
                icon={trigger.isPending ? <Loader2 className="animate-spin" /> : <Zap />}
                disabled={trigger.isPending}
                onClick={submit}
              >
                入队聚合任务
              </NeonButton>
            </div>
          </div>
        </GlassPanel>

        <div className="space-y-4">
          <GlassPanel title="使用场景 · USAGE">
            <ul className="space-y-2 p-4 text-[12px] text-cyan-300/75">
              <li className="flex gap-2">
                <span className="text-cyan-400">▹</span> 历史时段补算（如某段时间 worker 故障未跑聚合）
              </li>
              <li className="flex gap-2">
                <span className="text-cyan-400">▹</span> QA 验证聚合结果一致性
              </li>
              <li className="flex gap-2">
                <span className="text-cyan-400">▹</span> 本工具不直接跑聚合，而是入队 async_jobs；worker 进程抢任务执行
              </li>
              <li className="flex gap-2">
                <span className="text-cyan-400">▹</span> 暂无状态查询 UI，返回 job_id 后请用 SQL 查{' '}
                <code className="rounded-sm bg-black/40 px-1 font-mono text-emerald-300/85">async_jobs</code> 表
              </li>
            </ul>
          </GlassPanel>

          {lastJobId ? (
            <GlassPanel title="入队结果 · ENQUEUED">
              <div className="space-y-3 p-4">
                <div className="flex items-center gap-2 text-[#00ff88]">
                  <CheckCircle2 className="size-4" />
                  <span className="font-display text-sm">任务已入队</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="rounded-sm bg-black/40 px-2 py-1 font-mono text-[12px] text-emerald-300/90">
                    {lastJobId}
                  </span>
                  <button
                    type="button"
                    onClick={copyJobId}
                    title="复制 job_id"
                    className="rounded-sm border border-cyan-500/25 p-1 text-cyan-300/70 transition-colors hover:bg-cyan-500/10 hover:text-cyan-200"
                  >
                    <Copy className="size-3.5" />
                  </button>
                  {copied ? <span className="font-mono text-[10px] text-[#00ff88]">COPIED</span> : null}
                </div>
                <div>
                  <div className="mb-1 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">
                    状态查询 SQL
                  </div>
                  <pre className="terminal overflow-auto p-3 text-[11px] text-cyan-200/85">
{`docker exec omc-docker-postgres-1 psql -U omcgo -d omcgo -c "
SELECT id, status, started_at, finished_at, error_msg
FROM async_jobs WHERE id='${lastJobId}'"`}
                  </pre>
                </div>
              </div>
            </GlassPanel>
          ) : (
            <GlassPanel title="入队结果 · ENQUEUED">
              <div className="p-4 font-mono text-[11px] text-cyan-300/45">
                // 尚未触发，配置参数后点击「入队聚合任务」
              </div>
            </GlassPanel>
          )}
        </div>
      </div>
    </PageShell>
  )
}
