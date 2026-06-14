import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  ArrowLeft,
  Loader2,
  PauseCircle,
  PlayCircle,
  Ban,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { PageShell, formatTime } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useOpsTaskById,
  useOpsTemplates,
  usePauseOpsTask,
  useResumeOpsTask,
  useCancelOpsTask,
} from '@core/hooks/api/useOpsTools'
import type { OpsTask } from '@core/mock/data/opsTools'

// ============================================================
// 任务详情 — /ops/tasks/:id
// 真实数据：useOpsTaskById(id)（5s 轮询）；含暂停/继续/取消操作
// ============================================================

const TASK_STATUS_LABEL: Record<OpsTask['status'], string> = {
  pending: '待执行',
  running: '执行中',
  paused: '已暂停',
  success: '成功',
  failed: '失败',
  cancelled: '已取消',
}

const TASK_STATUS_VARIANT: Record<
  OpsTask['status'],
  'default' | 'warning' | 'success' | 'destructive' | 'muted'
> = {
  pending: 'muted',
  running: 'default',
  paused: 'warning',
  success: 'success',
  failed: 'destructive',
  cancelled: 'muted',
}

function ProgressBar({ value, status }: { value: number; status: OpsTask['status'] }) {
  const pct = Math.max(0, Math.min(100, value))
  const barColor =
    status === 'failed'
      ? 'bg-destructive'
      : status === 'success'
        ? 'bg-emerald-500'
        : status === 'running'
          ? 'bg-primary'
          : 'bg-muted-foreground/50'
  return (
    <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
      <div className={cn('h-full rounded-full transition-all', barColor)} style={{ width: `${pct}%` }} />
    </div>
  )
}

function Field({
  label,
  children,
  full,
}: {
  label: string
  children: React.ReactNode
  full?: boolean
}) {
  return (
    <div className={cn(full && 'sm:col-span-2')}>
      <dt className="text-xs uppercase tracking-wider text-muted-foreground">{label}</dt>
      <dd className="mt-1">{children}</dd>
    </div>
  )
}

export default function TaskDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: task, isLoading, isError, error, isFetching } = useOpsTaskById(id)

  const { data: templatesData } = useOpsTemplates({ page: 1, pageSize: 200 })
  const templateName = useMemo(() => {
    if (!task?.templateId) return undefined
    return (templatesData?.items ?? []).find((tpl) => tpl.id === task.templateId)?.templateName
  }, [task, templatesData])

  const pauseTask = usePauseOpsTask()
  const resumeTask = useResumeOpsTask()
  const cancelTask = useCancelOpsTask()
  const busy = pauseTask.isPending || resumeTask.isPending || cancelTask.isPending

  return (
    <PageShell
      title={task ? task.taskName : '任务详情'}
      description={id ? `任务 ID ${id}` : undefined}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => navigate('/ops/tasks')}>
            <ArrowLeft className="size-4" /> 返回任务列表
          </Button>
          {task && (
            <div className="ml-auto flex items-center gap-2">
              {task.status === 'running' && (
                <Button
                  variant="outline"
                  size="sm"
                  disabled={busy}
                  onClick={() => pauseTask.mutate(task.id)}
                >
                  <PauseCircle className="size-4" /> 暂停
                </Button>
              )}
              {(task.status === 'paused' || task.status === 'pending') && (
                <Button
                  variant="outline"
                  size="sm"
                  disabled={busy}
                  onClick={() => resumeTask.mutate(task.id)}
                >
                  <PlayCircle className="size-4" /> 继续
                </Button>
              )}
              {(task.status === 'running' ||
                task.status === 'paused' ||
                task.status === 'pending') && (
                <Button
                  variant="outline"
                  size="sm"
                  disabled={busy}
                  className="text-destructive hover:text-destructive"
                  onClick={() => {
                    if (window.confirm(`确认取消任务「${task.taskName}」？`)) {
                      cancelTask.mutate(task.id)
                    }
                  }}
                >
                  <Ban className="size-4" /> 取消
                </Button>
              )}
            </div>
          )}
        </div>
      }
    >
      {isLoading ? (
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </div>
      ) : isError ? (
        <Card className="flex h-48 items-center justify-center p-6 text-sm text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </Card>
      ) : !task ? (
        <Card className="flex h-48 flex-col items-center justify-center gap-2 p-6 text-muted-foreground">
          <span className="text-sm">未找到任务 {id}</span>
          <Button variant="outline" size="sm" onClick={() => navigate('/ops/tasks')}>
            返回任务列表
          </Button>
        </Card>
      ) : (
        <div className="flex flex-col gap-4">
          <Card className="p-5">
            <dl className="grid grid-cols-1 gap-x-8 gap-y-4 text-sm sm:grid-cols-2">
              <Field label="状态">
                <Badge variant={TASK_STATUS_VARIANT[task.status]}>
                  {TASK_STATUS_LABEL[task.status]}
                </Badge>
              </Field>
              <Field label="关联模板">{templateName ?? task.templateId ?? '—'}</Field>
              <Field label="进度" full>
                <ProgressBar value={task.progress} status={task.status} />
                <div className="mt-1 text-xs text-muted-foreground tabular-nums">
                  {task.progress}% · 步骤 {task.currentStep}/{task.totalSteps}
                </div>
              </Field>
              <Field label="设备总数">
                <span className="tabular-nums">{task.totalCount}</span>
              </Field>
              <Field label="成功 / 失败">
                <span className="tabular-nums">
                  <span className="text-emerald-600 dark:text-emerald-400">{task.successCount}</span>
                  {' / '}
                  <span className={task.failCount > 0 ? 'text-destructive' : ''}>
                    {task.failCount}
                  </span>
                </span>
              </Field>
              <Field label="创建人">{task.creator}</Field>
              <Field label="创建时间">{formatTime(task.createdAt)}</Field>
              {task.startedAt ? <Field label="开始时间">{formatTime(task.startedAt)}</Field> : null}
              {task.completedAt ? (
                <Field label="完成时间">{formatTime(task.completedAt)}</Field>
              ) : null}
              {task.message ? (
                <Field label="执行信息" full>
                  <span className="text-muted-foreground">{task.message}</span>
                </Field>
              ) : null}
            </dl>
          </Card>

          <Card className="p-5">
            <div className="mb-2 text-sm font-medium">目标设备（{task.deviceSns.length}）</div>
            <div className="max-h-64 overflow-y-auto rounded-md border bg-muted/30 p-3 font-mono text-xs">
              {task.deviceSns.length > 0 ? task.deviceSns.join(', ') : '—'}
            </div>
          </Card>
        </div>
      )}
    </PageShell>
  )
}
