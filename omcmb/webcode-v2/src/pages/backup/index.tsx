import { useMemo, useState } from 'react'
import {
  AlertTriangle,
  ChevronDown,
  ChevronRight,
  Plus,
  RefreshCcw,
  RotateCcw,
  StopCircle,
  Trash2,
} from 'lucide-react'

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
  useBackupRestoreTasks,
} from '@core/hooks/api/useBackup'
import type {
  BackupTask,
  RestoreTask,
  RestoreStatus,
} from '@core/mock/data/backup'

import { RestoreDialog } from './RestoreDialog'

// ============================================================
// 备份管理 — Tab：备份任务 / 配置还原
// 业务语义对齐 v1 webcode/src/pages/backup（BackupTasks + RestoreData）：
//   · 任务列表：状态/类型筛选 + 概览统计 + 进度/结果分解 + 行内详情下钻
//   · 任务操作：终止（cancel）/ 删除（delete），均带二次确认
//   · 配置还原：还原任务列表 + 创建还原（按路径 / 按任务 ID）
// 全部数据走 @core hooks（真实后端），三态完整。
// ============================================================

type Tab = 'tasks' | 'restore'

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

const RESTORE_STATUS_LABEL: Record<RestoreStatus, string> = {
  pending: '待启动',
  running: '还原中',
  completed: '已完成',
  failed: '失败',
  cancelled: '已取消',
}

const RESTORE_STATUS_VARIANT: Record<
  RestoreStatus,
  'success' | 'destructive' | 'warning' | 'muted'
> = {
  pending: 'muted',
  running: 'warning',
  completed: 'success',
  failed: 'destructive',
  cancelled: 'muted',
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

// ------------------------------------------------------------
// 备份任务 Tab
// ------------------------------------------------------------
function BackupTasksTab() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [status, setStatus] = useState('')
  const [taskType, setTaskType] = useState('')
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [confirm, setConfirm] = useState<
    { kind: 'cancel' | 'delete'; task: BackupTask } | null
  >(null)
  const [toast, setToast] = useState<{ kind: 'ok' | 'err'; msg: string } | null>(
    null
  )

  // status 推到后端；taskType（manual/scheduled）后端无对应轴，客户端过滤当前页。
  const params = useMemo(
    () => ({ page, pageSize, ...(status ? { status } : {}) }),
    [page, pageSize, status]
  )

  const { data, isLoading, isError, error, refetch } = useBackupTasks(params)
  const cancelTask = useCancelBackupTask()
  const deleteTasks = useDeleteBackupTasks()

  const allRows = data?.items ?? []
  const rows = useMemo(
    () =>
      taskType ? allRows.filter((t) => t.taskType === taskType) : allRows,
    [allRows, taskType]
  )
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  // 当前页统计（后端列表无聚合 stats，按当前页派生，足以给运营一个概览）
  const stats = useMemo(() => {
    const s = {
      total: allRows.length,
      success: 0,
      running: 0,
      failed: 0,
    }
    for (const t of allRows) {
      if (t.status === 'success') s.success += 1
      else if (t.status === 'running' || t.status === 'pending') s.running += 1
      else if (t.status === 'failed' || t.status === 'partial') s.failed += 1
    }
    return s
  }, [allRows])

  const cols = [
    '',
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
    <>
      {/* 概览统计 */}
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="本页任务" value={stats.total} />
        <Stat label="成功" value={stats.success} tone="emerald" />
        <Stat label="进行中/待启动" value={stats.running} tone="amber" />
        <Stat label="失败/部分" value={stats.failed} tone="red" />
      </div>

      {/* 工具栏 */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
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
              {cols.map((c, i) => (
                <TableHead key={c || `col-${i}`}>{c}</TableHead>
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
                const expanded = expandedId === t.id
                const canCancel = t.status === 'running' || t.status === 'pending'
                return (
                  <FragmentRow key={t.id}>
                    <TableRow
                      className="cursor-pointer"
                      onClick={() => setExpandedId(expanded ? null : t.id)}
                    >
                      <TableCell className="w-8 text-muted-foreground">
                        {expanded ? (
                          <ChevronDown className="size-4" />
                        ) : (
                          <ChevronRight className="size-4" />
                        )}
                      </TableCell>
                      <TableCell className="font-medium">{t.taskName}</TableCell>
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
                          <span className="text-destructive">
                            {' '}
                            · 失败 {t.failCount}
                          </span>
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
                      <TableCell onClick={(e) => e.stopPropagation()}>
                        <div className="flex items-center gap-1">
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
                    {expanded ? (
                      <TableRow className="bg-muted/30 hover:bg-muted/30">
                        <TableCell colSpan={cols.length} className="p-4">
                          <div className="grid gap-4 md:grid-cols-2">
                            <div className="space-y-1.5 text-sm">
                              <DetailItem label="任务 ID" value={t.id} mono />
                              <DetailItem label="创建人" value={t.creator || '—'} />
                              <DetailItem
                                label="存储位置"
                                value={t.storageLocation || '—'}
                                mono
                              />
                              <DetailItem
                                label="更新时间"
                                value={formatTime(t.updatedAt)}
                              />
                              {t.message ? (
                                <DetailItem label="备注" value={t.message} />
                              ) : null}
                            </div>
                            <div className="space-y-2">
                              <div className="text-xs font-medium text-muted-foreground">
                                目标设备（{t.deviceSns?.length ?? 0} 台）
                              </div>
                              <div className="flex max-h-32 flex-wrap gap-1 overflow-auto">
                                {(t.deviceSns ?? []).length === 0 ? (
                                  <span className="text-xs text-muted-foreground">
                                    —
                                  </span>
                                ) : (
                                  (t.deviceSns ?? []).map((sn) => (
                                    <Badge
                                      key={sn}
                                      variant="outline"
                                      className="font-mono"
                                    >
                                      {sn}
                                    </Badge>
                                  ))
                                )}
                              </div>
                            </div>
                          </div>
                        </TableCell>
                      </TableRow>
                    ) : null}
                  </FragmentRow>
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

      {/* 二次确认 */}
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
    </>
  )
}

// ------------------------------------------------------------
// 配置还原 Tab
// ------------------------------------------------------------
function RestoreTab() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [status, setStatus] = useState<RestoreStatus | ''>('')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [toast, setToast] = useState<string | null>(null)

  const params = useMemo(
    () => ({ page, pageSize, ...(status ? { status } : {}) }),
    [page, pageSize, status]
  )

  const { data, isLoading, isError, error, refetch } =
    useBackupRestoreTasks(params)

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = [
    '源对象路径',
    '目标设备',
    '状态',
    '进度',
    '错误',
    '开始时间',
    '完成时间',
    '创建人',
  ]

  const flash = (msg: string) => {
    setToast(msg)
    window.setTimeout(() => setToast(null), 3000)
  }

  return (
    <>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <Select
          value={status || 'all'}
          onValueChange={(v) => {
            setStatus(v === 'all' ? '' : (v as RestoreStatus))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="pending">待启动</SelectItem>
            <SelectItem value="running">还原中</SelectItem>
            <SelectItem value="completed">已完成</SelectItem>
            <SelectItem value="failed">失败</SelectItem>
            <SelectItem value="cancelled">已取消</SelectItem>
          </SelectContent>
        </Select>

        <div className="ml-auto flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
          </Button>
          <Button size="sm" onClick={() => setDialogOpen(true)}>
            <Plus /> 创建还原
          </Button>
        </div>
      </div>

      {toast ? (
        <div className="mb-3 rounded-md bg-emerald-500/10 px-3 py-2 text-sm text-emerald-600 dark:text-emerald-400">
          {toast}
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
              <EmptyRow colSpan={cols.length}>暂无还原任务</EmptyRow>
            ) : (
              rows.map((r: RestoreTask) => (
                <TableRow key={r.id}>
                  <TableCell
                    className="max-w-xs truncate font-mono text-xs"
                    title={`${r.sourceBucket}/${r.sourceObjectPath}`}
                  >
                    {r.sourceObjectPath}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {r.targetDeviceSns.length} 台
                  </TableCell>
                  <TableCell>
                    <Badge variant={RESTORE_STATUS_VARIANT[r.status]}>
                      {RESTORE_STATUS_LABEL[r.status] ?? r.status}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <ProgressBar
                      value={r.progress ?? 0}
                      tone={
                        r.status === 'failed'
                          ? 'destructive'
                          : r.status === 'completed'
                            ? 'success'
                            : 'default'
                      }
                    />
                  </TableCell>
                  <TableCell>
                    {r.errorMessage ? (
                      <span
                        className="inline-flex items-center gap-1 text-xs text-destructive"
                        title={r.errorMessage}
                      >
                        <AlertTriangle className="size-3.5" />
                        <span className="max-w-[10rem] truncate">
                          {r.errorMessage}
                        </span>
                      </span>
                    ) : (
                      <span className="text-xs text-muted-foreground">—</span>
                    )}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(r.startedAt)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(r.completedAt)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {r.createdBy ?? '—'}
                  </TableCell>
                </TableRow>
              ))
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

      <RestoreDialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        onSuccess={(msg) => flash(msg)}
      />
    </>
  )
}

// ------------------------------------------------------------
// 容器
// ------------------------------------------------------------
export function BackupPage() {
  const [tab, setTab] = useState<Tab>('tasks')

  // 顶部 isFetching 由两个 tab 各自的查询体现，这里仅用作壳标题；
  // 拆分查询的 isFetching 已在各 tab 的工具栏刷新按钮上体现。
  return (
    <PageShell
      title="备份管理"
      description="设备配置/全量备份任务与配置还原"
      toolbar={
        <div className="flex items-center gap-1 rounded-lg border bg-card p-1">
          <Button
            variant={tab === 'tasks' ? 'default' : 'ghost'}
            size="sm"
            onClick={() => setTab('tasks')}
          >
            备份任务
          </Button>
          <Button
            variant={tab === 'restore' ? 'default' : 'ghost'}
            size="sm"
            onClick={() => setTab('restore')}
          >
            <RotateCcw /> 配置还原
          </Button>
        </div>
      }
    >
      {tab === 'tasks' ? <BackupTasksTab /> : <RestoreTab />}
    </PageShell>
  )
}

// 行内详情项
function DetailItem({
  label,
  value,
  mono,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="flex gap-2">
      <span className="w-20 shrink-0 text-xs text-muted-foreground">{label}</span>
      <span className={cn('text-xs', mono && 'font-mono break-all')}>{value}</span>
    </div>
  )
}

// 让一行展开为多个 <tr>（任务行 + 详情行）而不引入额外 DOM 包裹
function FragmentRow({ children }: { children: React.ReactNode }) {
  return <>{children}</>
}

// 轻量二次确认弹窗（v2 无共享 Dialog，模块内自建）
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
