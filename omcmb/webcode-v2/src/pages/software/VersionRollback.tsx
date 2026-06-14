import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  CheckCircle2,
  Pause,
  Play,
  RefreshCcw,
  RotateCcw,
  Square,
  Trash2,
  Undo2,
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
  formatTime,
} from '@/components/layout/PageShell'

import {
  useUpgradeTasks,
  useSuspendTask,
  useResumeTask,
  useTerminateTask,
  useDeleteTask,
  useRetryTask,
} from '@core/hooks/api/useSoftware'
import { useProductClasses } from '@core/hooks/api/useDevices'
import type { TaskStatusType, UpgradeTaskInfo } from '@core/mock/data/software'

import { TASK_RESULT, TASK_STATUS, progressPct } from './_shared'
import { IconBtn, ProgressBar, Stat } from './_components'

// ===========================================================================
// 版本回退 — 对照 v1 webcode/src/pages/software/VersionRollback
// 回退任务编排（taskType=2），动作集与升级一致；详情复用 /software/upgrade-plan/:id
// ===========================================================================

const STATUS_CODE: Record<TaskStatusType, number> = {
  pending: 1,
  in_progress: 2,
  suspended: 3,
  ended: 4,
}

export default function VersionRollback() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [status, setStatus] = useState<'' | TaskStatusType>('')
  const [productClass, setProductClass] = useState('')

  const { data: productClasses } = useProductClasses()

  const params = useMemo(
    () => ({
      page,
      pageSize,
      taskType: 2,
      ...(status ? { status } : {}),
      ...(productClass ? { productClass } : {}),
    }),
    [page, pageSize, status, productClass]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useUpgradeTasks(params)

  const suspend = useSuspendTask()
  const resume = useResumeTask()
  const terminate = useTerminateTask()
  const remove = useDeleteTask()
  const retry = useRetryTask()
  const busy =
    suspend.isPending || resume.isPending || terminate.isPending || remove.isPending || retry.isPending

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const running = rows.filter((t) => t.status === 'in_progress').length
  const suspended = rows.filter((t) => t.status === 'suspended').length
  const ended = rows.filter((t) => t.status === 'ended').length

  const cols = [
    '任务名称',
    '设备型号',
    '状态',
    '结果',
    '进度',
    '成功/失败/总数',
    '创建人',
    '创建时间',
    '操作',
  ]

  return (
    <PageShell title="版本回退" description="设备固件回退任务编排：暂停/恢复/终止/重试 + 子任务下钻">
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="回退任务总数" value={total} icon={Undo2} />
        <Stat label="执行中" value={running} icon={Play} />
        <Stat label="已暂停" value={suspended} icon={Pause} tone="amber" />
        <Stat label="已结束" value={ended} icon={CheckCircle2} tone="emerald" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <Select
          value={productClass || 'all'}
          onValueChange={(v) => {
            setProductClass(v === 'all' ? '' : v)
            setPage(1)
          }}
        >
          <SelectTrigger className="w-44">
            <SelectValue placeholder="设备型号" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部型号</SelectItem>
            {(productClasses ?? []).map((pc) => (
              <SelectItem key={pc} value={pc}>
                {pc}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={status || 'all'}
          onValueChange={(v) => {
            setStatus(v === 'all' ? '' : (v as TaskStatusType))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="pending">等待中</SelectItem>
            <SelectItem value="in_progress">执行中</SelectItem>
            <SelectItem value="suspended">已暂停</SelectItem>
            <SelectItem value="ended">已结束</SelectItem>
          </SelectContent>
        </Select>
        <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
          <RefreshCcw className={isFetching ? 'animate-spin' : ''} /> 刷新
        </Button>
      </div>

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
              <EmptyRow colSpan={cols.length}>暂无回退任务</EmptyRow>
            ) : (
              rows.map((t: UpgradeTaskInfo) => {
                const st = TASK_STATUS[t.status]
                const code = STATUS_CODE[t.status]
                const showStart = code === 1 || code === 3
                const showPause = code === 2
                const showTerminate = code === 1 || code === 2 || code === 3
                const showRetry = code === 4 && t.failCount > 0
                const showDelete = code !== 2
                return (
                  <TableRow key={t.id}>
                    <TableCell>
                      <button
                        type="button"
                        className="text-left font-medium text-primary hover:underline"
                        onClick={() => navigate(`/software/upgrade-plan/${t.id}`)}
                        title="查看子任务"
                      >
                        {t.taskName}
                      </button>
                    </TableCell>
                    <TableCell>{t.productClass || '—'}</TableCell>
                    <TableCell>
                      <Badge variant={st.variant}>{st.label}</Badge>
                    </TableCell>
                    <TableCell>
                      {t.result ? (
                        <Badge variant={TASK_RESULT[t.result].variant}>{TASK_RESULT[t.result].label}</Badge>
                      ) : (
                        <span className="text-xs text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell className="w-40">
                      <ProgressBar pct={progressPct(t)} />
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      <span className="text-emerald-600 dark:text-emerald-400">{t.successCount}</span>
                      {' / '}
                      <span className="text-destructive">{t.failCount}</span>
                      {' / '}
                      <span className="text-muted-foreground">{t.totalCount}</span>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">{t.createUser || '—'}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{formatTime(t.createdAt)}</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        {showStart && (
                          <IconBtn title="恢复/启动" disabled={busy} onClick={() => resume.mutate(t.id)}>
                            <Play className="size-3.5" />
                          </IconBtn>
                        )}
                        {showPause && (
                          <IconBtn title="暂停" disabled={busy} onClick={() => suspend.mutate(t.id)}>
                            <Pause className="size-3.5" />
                          </IconBtn>
                        )}
                        {showRetry && (
                          <IconBtn title="重试失败设备" disabled={busy} onClick={() => retry.mutate(t.id)}>
                            <RotateCcw className="size-3.5" />
                          </IconBtn>
                        )}
                        {showTerminate && (
                          <IconBtn title="终止" disabled={busy} onClick={() => terminate.mutate(t.id)}>
                            <Square className="size-3.5" />
                          </IconBtn>
                        )}
                        {showDelete && (
                          <IconBtn
                            title="删除"
                            disabled={busy}
                            tone="destructive"
                            onClick={() => remove.mutate(t.id)}
                          >
                            <Trash2 className="size-3.5" />
                          </IconBtn>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={setPage} />
    </PageShell>
  )
}
