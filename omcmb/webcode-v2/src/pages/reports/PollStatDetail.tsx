import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, RefreshCcw, Activity } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { PageShell, formatTime } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { usePerformanceTasks } from '@core/hooks/api/usePerformance'

// ============================================================
// 轮询统计 — 任务详情 (/reports/poll-stats/:id) — v2
// 真实数据：usePerformanceTasks() 列表中按 :id 命中目标采集任务，展示其
//   类型/粒度/进度/状态/设备清单/KPI 清单/时间范围等。
// ============================================================

const STATUS_LABEL: Record<string, string> = {
  pending: '等待',
  running: '运行中',
  success: '成功',
  failed: '失败',
  cancelled: '已取消',
}

const STATUS_VARIANT: Record<string, 'success' | 'warning' | 'destructive' | 'muted' | 'secondary'> = {
  pending: 'secondary',
  running: 'warning',
  success: 'success',
  failed: 'destructive',
  cancelled: 'muted',
}

const TASK_TYPE_LABEL: Record<string, string> = {
  extraction: '数据采集',
  report: '报表生成',
  'threshold-check': '门限检查',
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1 py-2">
      <span className="text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </span>
      <span className="text-sm">{children}</span>
    </div>
  )
}

export default function PollStatDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const tasksQuery = usePerformanceTasks({ page: 1, pageSize: 200 })
  const task = useMemo(
    () => (tasksQuery.data?.items ?? []).find((t) => t.id === id),
    [tasksQuery.data, id],
  )

  const isLoading = tasksQuery.isLoading
  const isError = tasksQuery.isError
  const notFound = !isLoading && !isError && !task

  return (
    <PageShell
      title="采集任务详情"
      description={task ? task.taskName : id}
      isFetching={tasksQuery.isFetching}
      toolbar={
        <>
          <Button variant="outline" size="sm" onClick={() => navigate(-1)}>
            <ArrowLeft /> 返回
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={() => void tasksQuery.refetch()}
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
            {tasksQuery.error instanceof Error ? tasksQuery.error.message : '未知错误'}
          </CardContent>
        </Card>
      ) : notFound ? (
        <Card>
          <CardContent className="p-6 text-muted-foreground">未找到任务 {id}</CardContent>
        </Card>
      ) : task ? (
        <div className="space-y-4">
          <Card>
            <CardContent className="p-6">
              <div className="mb-4 flex items-center gap-2">
                <Activity className="size-5 text-primary" />
                <h2 className="text-lg font-semibold">{task.taskName}</h2>
                <Badge variant={STATUS_VARIANT[task.status] ?? 'muted'}>
                  {STATUS_LABEL[task.status] ?? task.status}
                </Badge>
              </div>

              <div className="mb-4">
                <div className="mb-1 flex items-center justify-between text-xs text-muted-foreground">
                  <span>执行进度</span>
                  <span className="tabular-nums">{Math.round(task.progress)}%</span>
                </div>
                <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                  <div
                    className={cn(
                      'h-full rounded-full',
                      task.status === 'failed' ? 'bg-rose-500' : 'bg-emerald-500',
                    )}
                    style={{ width: `${Math.min(100, Math.max(0, task.progress))}%` }}
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-x-8 md:grid-cols-3">
                <Field label="任务 ID">
                  <span className="font-mono text-xs">{task.id}</span>
                </Field>
                <Field label="类型">
                  <Badge variant="outline">
                    {TASK_TYPE_LABEL[task.taskType] ?? task.taskType}
                  </Badge>
                </Field>
                <Field label="采集粒度">{task.granularity}</Field>
                <Field label="设备数">
                  <span className="tabular-nums">{(task.deviceSns ?? []).length}</span>
                </Field>
                <Field label="KPI 数">
                  <span className="tabular-nums">{(task.kpiCodes ?? []).length}</span>
                </Field>
                <Field label="创建人">{task.creator || '—'}</Field>
                <Field label="时间范围">
                  {Array.isArray(task.timeRange) && task.timeRange.length === 2
                    ? `${formatTime(task.timeRange[0])} ~ ${formatTime(task.timeRange[1])}`
                    : '—'}
                </Field>
                <Field label="创建时间">{formatTime(task.createdAt)}</Field>
                <Field label="更新时间">{formatTime(task.updatedAt)}</Field>
              </div>
            </CardContent>
          </Card>

          {/* 设备清单 */}
          {(task.deviceSns ?? []).length > 0 && (
            <Card>
              <CardContent className="p-6">
                <div className="mb-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">
                  采集设备 ({task.deviceSns.length})
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {task.deviceSns.map((sn) => (
                    <Badge key={sn} variant="muted" className="font-mono">
                      {sn}
                    </Badge>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}

          {/* KPI 清单 */}
          {(task.kpiCodes ?? []).length > 0 && (
            <Card>
              <CardContent className="p-6">
                <div className="mb-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">
                  采集 KPI ({task.kpiCodes.length})
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {task.kpiCodes.map((code) => (
                    <Badge key={code} variant="outline" className="font-mono">
                      {code}
                    </Badge>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      ) : null}
    </PageShell>
  )
}
