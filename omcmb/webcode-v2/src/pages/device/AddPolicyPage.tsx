import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Loader2, RotateCw, Save } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { PageShell, formatTime } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useProvisioningTask,
  useCreateProvisioningTask,
  useRetryProvisioningTask,
} from '@core/hooks/api/useProvisioning'
import {
  provisioningStatusCode,
  type ProvisioningTaskStatusCode,
} from '@core/services/api/provisionApi'

// ============================================================
// 即插即用 — 开站任务表单 / 详情（add · edit · view 三态合一）
// 对照 v1 webcode/src/pages/device/PlugAndPlay/AddPolicyPage：v1 是大型本地
// 策略表单（mock）。v2 聚焦真实后端 ProvisioningTask：
//   - add：录入 deviceId，调用 useCreateProvisioningTask 真实建任务
//   - view/:id：useProvisioningTask 真实加载任务详情（含失败重试）
//   - edit/:id：同 view，加重试入口（后端无任务字段更新接口，故只读 + 重试）
// 路由由 /device/plug-and-play/{add,edit/:id,view/:id} 决定模式。
// ============================================================

type Mode = 'add' | 'edit' | 'view'

const STATUS_META: Record<
  ProvisioningTaskStatusCode,
  { label: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'muted' }
> = {
  '0': { label: '成功', variant: 'success' },
  '1': { label: '失败', variant: 'destructive' },
  '2': { label: '执行中', variant: 'warning' },
  '3': { label: '未执行', variant: 'muted' },
  '4': { label: '跳过', variant: 'muted' },
}

function useMode(): Mode {
  const { id } = useParams<{ id?: string }>()
  const path = window.location.pathname
  if (!id) return 'add'
  if (path.includes('/edit/')) return 'edit'
  return 'view'
}

function Field({
  label,
  value,
  mono,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="flex items-center justify-between gap-4 border-b py-2.5 text-sm">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span className={cn('truncate text-right', mono && 'font-mono text-xs')}>
        {value || '—'}
      </span>
    </div>
  )
}

function AddForm() {
  const navigate = useNavigate()
  const create = useCreateProvisioningTask()
  const [deviceId, setDeviceId] = useState('')
  const trimmed = deviceId.trim()

  function handleSubmit() {
    if (!trimmed) return
    create.mutate(
      { deviceId: trimmed },
      {
        onSuccess: () => navigate('/device/plug-and-play'),
      }
    )
  }

  return (
    <Card className="max-w-xl">
      <CardHeader className="p-5 pb-2">
        <CardTitle className="text-base font-medium">新建开站任务</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4 p-5 pt-3">
        <div className="space-y-1.5">
          <Label htmlFor="deviceId">设备 ID</Label>
          <Input
            id="deviceId"
            placeholder="输入待开站设备的 device ID（UUID）"
            value={deviceId}
            onChange={(e) => setDeviceId(e.target.value)}
          />
          <p className="text-xs text-muted-foreground">
            提交后将为该设备创建自动开站任务（发现 → 匹配模板 → 配置下发 → 校验）。
          </p>
        </div>

        {create.isError && (
          <div className="rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
            创建失败：
            {create.error instanceof Error ? create.error.message : '未知错误'}
          </div>
        )}

        <div className="flex items-center gap-2">
          <Button disabled={!trimmed || create.isPending} onClick={handleSubmit}>
            {create.isPending ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <Save className="size-4" />
            )}
            创建任务
          </Button>
          <Button
            variant="ghost"
            onClick={() => navigate('/device/plug-and-play')}
          >
            取消
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

function TaskDetail({ id, mode }: { id: string; mode: Mode }) {
  const navigate = useNavigate()
  const { data: task, isLoading, isError, error, isFetching, refetch } =
    useProvisioningTask(id)
  const retry = useRetryProvisioningTask()

  const meta = useMemo(
    () =>
      task ? STATUS_META[provisioningStatusCode(task.status)] : null,
    [task]
  )

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="size-6 animate-spin text-muted-foreground" />
      </div>
    )
  }
  if (isError) {
    return (
      <Card className="flex h-48 items-center justify-center p-6 text-sm text-destructive">
        加载失败：{error instanceof Error ? error.message : '未知错误'}
      </Card>
    )
  }
  if (!task) {
    return (
      <Card className="flex h-48 flex-col items-center justify-center gap-2 p-6 text-muted-foreground">
        <span className="text-sm">未找到任务 {id}</span>
        <Button
          variant="outline"
          size="sm"
          onClick={() => navigate('/device/plug-and-play')}
        >
          返回任务列表
        </Button>
      </Card>
    )
  }

  const progress =
    task.totalSteps > 0 ? `${task.currentStep}/${task.totalSteps}` : '—'
  const pct =
    task.totalSteps > 0
      ? Math.min(100, Math.round((task.currentStep / task.totalSteps) * 100))
      : 0

  return (
    <div className="flex max-w-2xl flex-col gap-4">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between p-5 pb-2">
          <CardTitle className="text-base font-medium">任务信息</CardTitle>
          {meta && <Badge variant={meta.variant}>{meta.label}</Badge>}
        </CardHeader>
        <CardContent className="p-5 pt-3">
          <div className="grid grid-cols-1 gap-x-8 sm:grid-cols-2">
            <Field label="任务 ID" value={task.id} mono />
            <Field label="设备 ID" value={task.deviceId} mono />
            <Field label="模板 ID" value={task.templateId ?? '—'} mono />
            <Field label="原始状态" value={task.status} />
            <Field label="进度" value={progress} />
            <Field
              label="重试次数"
              value={`${task.retryCount}/${task.maxRetries}`}
            />
            <Field label="开始时间" value={formatTime(task.startedAt)} />
            <Field label="结束时间" value={formatTime(task.completedAt)} />
            <Field label="创建时间" value={formatTime(task.createdAt)} />
            <Field label="更新时间" value={formatTime(task.updatedAt)} />
          </div>

          {/* 进度条 */}
          <div className="mt-4">
            <div className="mb-1 flex items-center justify-between text-xs text-muted-foreground">
              <span>执行进度</span>
              <span className="tabular-nums">{pct}%</span>
            </div>
            <div className="h-2 overflow-hidden rounded-full bg-muted">
              <div
                className={cn(
                  'h-full rounded-full',
                  task.status === 'failed'
                    ? 'bg-destructive'
                    : task.status === 'completed'
                      ? 'bg-emerald-500'
                      : 'bg-primary'
                )}
                style={{ width: `${pct}%` }}
              />
            </div>
          </div>

          {task.errorMessage && (
            <div className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
              <span className="font-medium">失败原因：</span>
              {task.errorMessage}
            </div>
          )}
        </CardContent>
      </Card>

      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          {isFetching ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <RotateCw className="size-4" />
          )}
          刷新
        </Button>
        {(mode === 'edit' || task.status === 'failed') && (
          <Button
            size="sm"
            disabled={retry.isPending}
            onClick={() => retry.mutate(task.id)}
          >
            {retry.isPending ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <RotateCw className="size-4" />
            )}
            重试任务
          </Button>
        )}
      </div>

      {retry.isError && (
        <div className="rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
          重试失败：
          {retry.error instanceof Error ? retry.error.message : '未知错误'}
        </div>
      )}
    </div>
  )
}

export default function AddPolicyPage() {
  const { id } = useParams<{ id?: string }>()
  const navigate = useNavigate()
  const mode = useMode()

  const title =
    mode === 'add' ? '新增开站策略' : mode === 'edit' ? '编辑开站任务' : '开站任务详情'

  return (
    <PageShell
      title={title}
      description="自动开站（即插即用）"
      toolbar={
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate('/device/plug-and-play')}
        >
          <ArrowLeft className="size-4" /> 返回
        </Button>
      }
    >
      {mode === 'add' || !id ? (
        <AddForm />
      ) : (
        <TaskDetail id={id} mode={mode} />
      )}
    </PageShell>
  )
}
