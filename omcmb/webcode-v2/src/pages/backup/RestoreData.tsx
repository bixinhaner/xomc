import { useMemo, useState } from 'react'
import { AlertTriangle, Plus, RefreshCcw } from 'lucide-react'

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
  formatTime,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useBackupRestoreTasks } from '@core/hooks/api/useBackup'
import type { RestoreStatus, RestoreTask } from '@core/mock/data/backup'

import { RestoreDialog } from './RestoreDialog'

// ============================================================
// 配置还原 — 对齐 v1 webcode/src/pages/backup/RestoreData（路由 /backup/restore）
//   · 还原任务列表（状态筛选）+ 进度/错误展示
//   · 创建还原：按对象路径 / 按备份任务 ID（复用模块内 RestoreDialog）
// 全部数据走 @core hooks（真实后端），三态完整。列表每 5s 轮询（hook 内置）。
// ============================================================

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
      <div className="h-1.5 w-20 overflow-hidden rounded-full bg-muted">
        <div className={cn('h-full rounded-full', bar)} style={{ width: `${v}%` }} />
      </div>
      <span className="text-xs tabular-nums text-muted-foreground">{v}%</span>
    </div>
  )
}

export default function RestoreData() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [status, setStatus] = useState<RestoreStatus | ''>('')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [toast, setToast] = useState<string | null>(null)

  const params = useMemo(
    () => ({ page, pageSize, ...(status ? { status } : {}) }),
    [page, pageSize, status]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
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
    <PageShell
      title="配置还原"
      description="将历史备份产物还原到目标设备"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
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
      }
    >
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
                    {r.sourceObjectPath || '—'}
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
    </PageShell>
  )
}
