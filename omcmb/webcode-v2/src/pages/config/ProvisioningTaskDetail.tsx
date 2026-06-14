import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Loader2, RefreshCcw, RotateCcw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { PageShell, formatTime } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useProvisioningTask,
  useRetryProvisioningTask,
} from '@core/hooks/api/useProvisioning'

// ============================================================
// 自动开站任务详情 — /config/auto-provision/:id（隐藏路由）。
// 真实数据：useProvisioningTask（5s 轮询）；失败任务可重试。
// ============================================================

const STATUS_VARIANT: Record<
  string,
  'success' | 'warning' | 'destructive' | 'secondary'
> = {
  completed: 'success',
  failed: 'destructive',
  discovered: 'secondary',
}

export default function ProvisioningTaskDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data, isLoading, isError, error, isFetching, refetch } = useProvisioningTask(id)
  const retry = useRetryProvisioningTask()

  const pct =
    data && data.totalSteps > 0 ? Math.round((data.currentStep / data.totalSteps) * 100) : 0

  return (
    <PageShell
      title="开站任务详情"
      description={data ? `任务 ${data.id}` : '自动开站任务'}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => navigate('/config/auto-provision')}>
            <ArrowLeft className="size-4" /> 返回列表
          </Button>
          <div className="ml-auto flex items-center gap-2">
            {data?.status === 'failed' && (
              <Button
                size="sm"
                disabled={retry.isPending}
                onClick={() => retry.mutate(data.id)}
              >
                {retry.isPending ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : (
                  <RotateCcw className="size-4" />
                )}
                重试
              </Button>
            )}
            <Button variant="outline" size="sm" onClick={() => void refetch()}>
              <RefreshCcw className="size-4" /> 刷新
            </Button>
          </div>
        </div>
      }
    >
      {isLoading ? (
        <div className="flex h-48 items-center justify-center">
          <Loader2 className="size-5 animate-spin text-muted-foreground" />
        </div>
      ) : isError ? (
        <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-6 text-center text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </div>
      ) : !data ? (
        <div className="rounded-lg border px-4 py-6 text-center text-muted-foreground">
          未找到该任务
        </div>
      ) : (
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                任务概览
                <Badge variant={STATUS_VARIANT[data.status] ?? 'warning'}>{data.status}</Badge>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="mb-4">
                <div className="mb-1 flex items-center justify-between text-xs text-muted-foreground">
                  <span>执行进度</span>
                  <span className="tabular-nums">
                    {data.currentStep}/{data.totalSteps}（{pct}%）
                  </span>
                </div>
                <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                  <div
                    className={cn(
                      'h-full rounded-full',
                      data.status === 'failed' ? 'bg-destructive' : 'bg-primary'
                    )}
                    style={{ width: `${pct}%` }}
                  />
                </div>
              </div>
              <dl className="grid grid-cols-2 gap-x-8 gap-y-2 text-sm md:grid-cols-3">
                <Field label="任务 ID" value={data.id} mono />
                <Field label="设备 ID" value={data.deviceId} mono />
                <Field label="模板 ID" value={data.templateId || '—'} mono />
                <Field label="重试" value={`${data.retryCount} / ${data.maxRetries}`} />
                <Field label="开始时间" value={formatTime(data.startedAt)} />
                <Field label="完成时间" value={formatTime(data.completedAt)} />
                <Field label="创建时间" value={formatTime(data.createdAt)} />
                <Field label="更新时间" value={formatTime(data.updatedAt)} />
              </dl>
            </CardContent>
          </Card>

          {data.errorMessage && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base text-destructive">错误信息</CardTitle>
              </CardHeader>
              <CardContent>
                <pre className="whitespace-pre-wrap break-words rounded-md bg-destructive/5 p-3 font-mono text-xs text-destructive">
                  {data.errorMessage}
                </pre>
              </CardContent>
            </Card>
          )}
        </div>
      )}
    </PageShell>
  )
}

function Field({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-xs uppercase tracking-wider text-muted-foreground">{label}</dt>
      <dd className={cn('mt-0.5 font-medium', mono && 'font-mono text-xs')}>{value}</dd>
    </div>
  )
}
