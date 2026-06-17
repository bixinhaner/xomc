import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, StopCircle, Trash2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  EmptyRow,
  ErrorRow,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
  formatBytes,
  formatTime,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useBackupTasks,
  useCancelBackupTask,
  useDeleteBackupTasks,
} from '@core/hooks/api/useBackup'
import type { BackupTask } from '@core/mock/data/backup'

// ============================================================
// 备份任务列表 — 对齐 v1 webcode/src/pages/backup/BackupTasks（路由 /backup/tasks）
//   · 状态/类型筛选 + 当前页概览统计 + 进度/结果分解
//   · 行内"详情"链接下钻到 /backup/tasks/:id（真实数据加载）
//   · 任务操作：终止（cancel）/ 删除（delete），均带二次确认
// 全部数据走 @core hooks（真实后端），三态完整。
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

// 后端列表无 manual/scheduled 过滤轴；status 推后端，taskType 在当前页客户端过滤。
const STATUS_NUM_TO_BACKEND: Record<string, string> = {
  pending: 'pending',
  running: 'running',
  success: 'success',
  failed: 'failed',
  cancelled: 'cancelled',
}

function ProgressBar({
  value,
  tone,
}: {
  value: number
  tone: 'default' | 'success' | 'destructive'
}) {
  const v = Math.max(0, Math.min(100, value))
  const bar =
    tone === 'success'
      ? 'bg-emerald-500'
      : tone === 'destructive'
        ? 'bg-destructive'
        : 'bg-primary'
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-20 overflow-hidden rounded-full bg-muted">
        <div className={cn('h-full rounded-full', bar)} style={{ width: `${v}%` }} />
      </div>
      <span className="text-xs tabular-nums text-muted-foreground">{v}%</span>
    </div>
  )
}

function Stat({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: number
  tone?: 'default' | 'emerald' | 'amber' | 'red' | 'muted'
}) {
  const toneClass = {
    default: 'text-foreground',
    emerald: 'text-emerald-600 dark:text-emerald-400',
    amber: 'text-amber-600 dark:text-amber-400',
    red: 'text-destructive',
    muted: 'text-muted-foreground',
  }[tone]
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>
        {value}
      </div>
    </div>
  )
}

function ConfirmDialog({
  title,
  message,
  danger,
  loading,
  onCancel,
  onConfirm,
}: {
  title: string
  message: string
  danger?: boolean
  loading?: boolean
  onCancel: () => void
  onConfirm: () => void
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/40" onClick={onCancel} aria-hidden />
      <div className="relative z-10 w-full max-w-sm rounded-xl border bg-card shadow-lg">
        <div className="px-5 py-4">
          <h2 className="text-base font-semibold">{title}</h2>
          <p className="mt-2 text-sm text-muted-foreground">{message}</p>
        </div>
        <div className="flex justify-end gap-2 border-t px-5 py-3">
          <Button variant="outline" size="sm" onClick={onCancel}>
            取消
          </Button>
          <Button
            variant={danger ? 'destructive' : 'default'}
            size="sm"
            disabled={loading}
            onClick={onConfirm}
          >
            {loading ? '处理中…' : '确认'}
          </Button>
        </div>
      </div>
    </div>
  )
}

export default function BackupTasks() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [status, setStatus] = useState('')
  const [taskType, setTaskType] = useState('')
  const [confirm, setConfirm] = useState<
    { kind: 'cancel' | 'delete'; task: BackupTask } | null
  >(null)
  const [toast, setToast] = useState<{ kind: 'ok' | 'err'; msg: string } | null>(
    null
  )

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(status && STATUS_NUM_TO_BACKEND[status]
        ? { status: STATUS_NUM_TO_BACKEND[status] }
        : {}),
    }),
    [page, pageSize, status]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useBackupTasks(params)
  const cancelTask = useCancelBackupTask()
  const deleteTasks = useDeleteBackupTasks()

  const allRows = data?.items ?? []
  const rows = useMemo(
    () => (taskType ? allRows.filter((t) => t.taskType === taskType) : allRows),
    [allRows, taskType]
  )
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const stats = useMemo(() => {
    const s = { total: allRows.length, success: 0, running: 0, failed: 0 }
    for (const t of allRows) {
      if (t.status === 'success') s.success += 1
      else if (t.status === 'running' || t.status === 'pending') s.running += 1
      else if (t.status === 'failed' || t.status === 'partial') s.failed += 1
    }
    return s
  }, [allRows])

  const cols = [
    '任务',
    '类型',
    '备份方式',
    '状态',
    '进度',
    '结果',
    '设备',
    '大小',
    '创建时间',
    '操作',
  ]

  const flash = (kind: 'ok' | 'err', msg: string) => {
    setToast({ kind, msg })
    window.setTimeout(() => setToast(null), 3000)
  }

  const doCancel = (t: BackupTask) =>
    cancelTask.mutate(t.id, {
      onSuccess: () => flash('ok', `已终止任务「${t.taskName}」`),
      onError: (e: unknown) =>
        flash('err', e instanceof Error ? e.message : '终止失败'),
    })

  const doDelete = (t: BackupTask) =>
    deleteTasks.mutate([t.id], {
      onSuccess: () => flash('ok', `已删除任务「${t.taskName}」`),
      onError: (e: unknown) =>
        flash('err', e instanceof Error ? e.message : '删除失败'),
    })

  const mutating = cancelTask.isPending || deleteTasks.isPending

  return (
    <PageShell
      title="备份任务"
      description="设备配置/全量备份任务列表，支持终止与删除"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Select
            value={status || 'all'}
            onValueChange={(v) => {
              setStatus(v === 'all' ? '' : v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              <SelectItem value="pending">待启动</SelectItem>
              <SelectItem value="running">运行中</SelectItem>
              <SelectItem value="success">已成功</SelectItem>
              <SelectItem value="failed">失败</SelectItem>
              <SelectItem value="cancelled">已取消</SelectItem>
            </SelectContent>
          </Select>

          <Select
            value={taskType || 'all'}
            onValueChange={(v) => setTaskType(v === 'all' ? '' : v)}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部类型</SelectItem>
              <SelectItem value="manual">手动</SelectItem>
              <SelectItem value="scheduled">定时</SelectItem>
            </SelectContent>
          </Select>

          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={() => refetch()}
          >
            <RefreshCcw /> 刷新
          </Button>
        </div>
      }
    >
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="本页任务" value={stats.total} />
        <Stat label="成功" value={stats.success} tone="emerald" />
        <Stat label="进行中/待启动" value={stats.running} tone="amber" />
        <Stat label="失败/部分" value={stats.failed} tone="red" />
      </div>

      {toast ? (
        <div
          className={cn(
            'mb-3 rounded-md px-3 py-2 text-sm',
            toast.kind === 'ok'
              ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
              : 'bg-destructive/10 text-destructive'
          )}
        >
          {toast.msg}
        </div>
      ) : null}

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无备份任务</EmptyRow>
            ) : (
              rows.map((t) => {
                const canCancel =
                  t.status === 'running' || t.status === 'pending'
                return (
                  <TableRow key={t.id}>
                    <TableCell>
                      <button
                        type="button"
                        className="text-left font-medium text-primary hover:underline"
                        onClick={() => navigate(`/backup/tasks`)}
                      >
                        {t.taskName || '—'}
                      </button>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">
                        {t.taskType === 'manual' ? '手动' : '定时'}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant="muted">
                        {BACKUP_TYPE_LABEL[t.backupType] ?? t.backupType}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={TASK_STATUS_VARIANT[t.status]}>
                        {TASK_STATUS_LABEL[t.status] ?? t.status}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <ProgressBar
                        value={t.progress ?? 0}
                        tone={
                          t.status === 'failed'
                            ? 'destructive'
                            : t.status === 'success'
                              ? 'success'
                              : 'default'
                        }
                      />
                    </TableCell>
                    <TableCell className="text-xs">
                      <span className="text-emerald-600 dark:text-emerald-400">
                        {t.successCount}
                      </span>
                      /{t.totalCount}
                      {t.failCount > 0 ? (
                        <span className="text-destructive"> · 失败 {t.failCount}</span>
                      ) : null}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {t.deviceSns?.length ?? 0} 台
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatBytes(t.fileSize)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(t.createdAt)}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => navigate(`/backup/tasks`)}
                        >
                          详情
                        </Button>
                        {canCancel ? (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-amber-600 dark:text-amber-400"
                            disabled={mutating}
                            onClick={() => setConfirm({ kind: 'cancel', task: t })}
                          >
                            <StopCircle className="size-4" /> 终止
                          </Button>
                        ) : null}
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive"
                          disabled={mutating}
                          onClick={() => setConfirm({ kind: 'delete', task: t })}
                        >
                          <Trash2 className="size-4" /> 删除
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={pageSize}
        onChange={setPage}
      />

      {confirm ? (
        <ConfirmDialog
          title={confirm.kind === 'cancel' ? '终止备份任务' : '删除备份任务'}
          message={
            confirm.kind === 'cancel'
              ? `确认终止任务「${confirm.task.taskName}」？运行中的设备备份将被中断。`
              : `确认删除任务「${confirm.task.taskName}」？此操作不可恢复。`
          }
          danger
          loading={mutating}
          onCancel={() => setConfirm(null)}
          onConfirm={() => {
            if (confirm.kind === 'cancel') doCancel(confirm.task)
            else doDelete(confirm.task)
            setConfirm(null)
          }}
        />
      ) : null}
    </PageShell>
  )
}
