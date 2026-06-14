import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, RefreshCcw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { PageShell, formatBytes, formatTime } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useBackupTaskById } from '@core/hooks/api/useBackup'
import type { BackupTask } from '@core/mock/data/backup'

// ============================================================
// 备份任务详情 — /backup/tasks/:id（带参，hidden）
// 真实数据走 useBackupTaskById；展示任务元信息 + 进度 + 结果分解 + 目标设备。
// ============================================================

const BACKUP_TYPE_LABEL: Record<BackupTask['backupType'], string> = {
  full: '全量',
  incremental: '增量',
  'config-only': '配置',
}

const TASK_STATUS_LABEL: Record<BackupTask['status'], string> = {
  pending: '待启动',
  running: '运行中',
  success: '已成功',
  failed: '失败',
  cancelled: '已取消',
  partial: '部分成功',
}

const TASK_STATUS_VARIANT: Record<
  BackupTask['status'],
  'success' | 'destructive' | 'warning' | 'muted' | 'secondary'
> = {
  pending: 'muted',
  running: 'warning',
  success: 'success',
  failed: 'destructive',
  cancelled: 'muted',
  partial: 'secondary',
}

function ProgressBar({ value, tone }: { value: number; tone: 'default' | 'success' | 'destructive' }) {
  const v = Math.max(0, Math.min(100, value))
  const bar =
    tone === 'success'
      ? 'bg-emerald-500'
      : tone === 'destructive'
        ? 'bg-destructive'
        : 'bg-primary'
  return (
    <div className="flex items-center gap-2">
      <div className="h-2 w-48 overflow-hidden rounded-full bg-muted">
        <div className={cn('h-full rounded-full', bar)} style={{ width: `${v}%` }} />
      </div>
      <span className="text-sm tabular-nums text-muted-foreground">{v}%</span>
    </div>
  )
}

function DetailItem({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex gap-3 border-b py-2 last:border-b-0">
      <span className="w-24 shrink-0 text-sm text-muted-foreground">{label}</span>
      <span className={cn('text-sm', mono && 'break-all font-mono')}>{value}</span>
    </div>
  )
}

export default function BackupTaskDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: task, isLoading, isError, error, isFetching, refetch } =
    useBackupTaskById(id ?? '')

  return (
    <PageShell
      title="备份任务详情"
      description={id ? `任务 ID：${id}` : undefined}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => navigate('/backup/tasks')}>
            <ArrowLeft className="size-4" /> 返回任务列表
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={() => refetch()}
          >
            <RefreshCcw className="size-4" /> 刷新
          </Button>
        </div>
      }
    >
      {isLoading ? (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground">
            加载中…
          </CardContent>
        </Card>
      ) : isError ? (
        <Card>
          <CardContent className="py-12 text-center text-destructive">
            加载失败：{error instanceof Error ? error.message : '未知错误'}
          </CardContent>
        </Card>
      ) : !task ? (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground">
            未找到该备份任务
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                {task.taskName || '—'}
                <Badge variant={TASK_STATUS_VARIANT[task.status]}>
                  {TASK_STATUS_LABEL[task.status] ?? task.status}
                </Badge>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <DetailItem label="任务 ID" value={task.id} mono />
              <DetailItem
                label="任务类型"
                value={task.taskType === 'manual' ? '手动' : '定时'}
              />
              <DetailItem
                label="备份方式"
                value={BACKUP_TYPE_LABEL[task.backupType] ?? task.backupType}
              />
              <DetailItem label="创建人" value={task.creator || '—'} />
              <DetailItem label="存储位置" value={task.storageLocation || '—'} mono />
              <DetailItem label="文件大小" value={formatBytes(task.fileSize)} />
              <DetailItem label="创建时间" value={formatTime(task.createdAt)} />
              <DetailItem label="更新时间" value={formatTime(task.updatedAt)} />
              {task.message ? <DetailItem label="备注" value={task.message} /> : null}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">执行进度</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <ProgressBar
                value={task.progress ?? 0}
                tone={
                  task.status === 'failed'
                    ? 'destructive'
                    : task.status === 'success'
                      ? 'success'
                      : 'default'
                }
              />
              <div className="grid grid-cols-3 gap-3 text-center">
                <div className="rounded-lg border bg-card px-3 py-2">
                  <div className="text-xs text-muted-foreground">总数</div>
                  <div className="mt-1 text-xl font-semibold tabular-nums">
                    {task.totalCount}
                  </div>
                </div>
                <div className="rounded-lg border bg-card px-3 py-2">
                  <div className="text-xs text-muted-foreground">成功</div>
                  <div className="mt-1 text-xl font-semibold tabular-nums text-emerald-600 dark:text-emerald-400">
                    {task.successCount}
                  </div>
                </div>
                <div className="rounded-lg border bg-card px-3 py-2">
                  <div className="text-xs text-muted-foreground">失败</div>
                  <div className="mt-1 text-xl font-semibold tabular-nums text-destructive">
                    {task.failCount}
                  </div>
                </div>
              </div>

              <div>
                <div className="mb-2 text-sm font-medium text-muted-foreground">
                  目标设备（{task.deviceSns?.length ?? 0} 台）
                </div>
                <div className="flex max-h-40 flex-wrap gap-1 overflow-auto">
                  {(task.deviceSns ?? []).length === 0 ? (
                    <span className="text-sm text-muted-foreground">—</span>
                  ) : (
                    (task.deviceSns ?? []).map((sn) => (
                      <Badge key={sn} variant="outline" className="font-mono">
                        {sn}
                      </Badge>
                    ))
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </PageShell>
  )
}
