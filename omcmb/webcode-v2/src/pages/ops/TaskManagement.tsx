import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  RefreshCcw,
  Search,
  PauseCircle,
  PlayCircle,
  Ban,
  Eye,
  Activity,
  CheckCircle2,
  XCircle,
  Clock,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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

import {
  useOpsTasks,
  useOpsTemplates,
  usePauseOpsTask,
  useResumeOpsTask,
  useCancelOpsTask,
} from '@core/hooks/api/useOpsTools'
import type { OpsTask } from '@core/mock/data/opsTools'

// ============================================================
// 自动化任务 — 对齐 v1 webcode/src/pages/ops/TaskManagement
// 列表 + 筛选 + 暂停/继续/取消；点任务名进 /ops/tasks/:id 详情
// ============================================================

const TASK_STATUS_VALUES: ReadonlyArray<OpsTask['status']> = [
  'pending',
  'running',
  'paused',
  'success',
  'failed',
  'cancelled',
]

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

function StatCard({
  label,
  value,
  tone = 'default',
  icon,
}: {
  label: string
  value: number
  tone?: 'default' | 'emerald' | 'amber' | 'red' | 'muted'
  icon?: React.ReactNode
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
      <div className="flex items-center justify-between">
        <div className="text-xs uppercase tracking-wider text-muted-foreground">{label}</div>
        {icon ? <span className="text-muted-foreground">{icon}</span> : null}
      </div>
      <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>{value}</div>
    </div>
  )
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

export default function TaskManagement() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState<OpsTask['status'] | ''>('')
  const [creator, setCreator] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(status ? { status } : {}),
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(creator.trim() ? { creator: creator.trim() } : {}),
    }),
    [page, pageSize, status, keyword, creator]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useOpsTasks(params, {
    refetchOnMount: 'always',
  })

  // 模板名映射（任务列表只携带 templateId）
  const { data: templatesData } = useOpsTemplates({ page: 1, pageSize: 200 })
  const templateNameMap = useMemo(
    () =>
      Object.fromEntries(
        (templatesData?.items ?? []).map((tpl) => [tpl.id, tpl.templateName])
      ),
    [templatesData]
  )

  const pauseTask = usePauseOpsTask()
  const resumeTask = useResumeOpsTask()
  const cancelTask = useCancelOpsTask()
  const busy = pauseTask.isPending || resumeTask.isPending || cancelTask.isPending

  const rows: OpsTask[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const stats = useMemo(() => {
    const acc = { running: 0, success: 0, failed: 0, pending: 0 }
    for (const t of rows) {
      if (t.status === 'running') acc.running += 1
      else if (t.status === 'success') acc.success += 1
      else if (t.status === 'failed') acc.failed += 1
      else if (t.status === 'pending' || t.status === 'paused') acc.pending += 1
    }
    return acc
  }, [rows])

  const cols = ['任务名称', '关联模板', '设备数', '状态', '进度', '创建人', '创建时间', '操作']

  return (
    <PageShell title="自动化任务" description="按模板批量编排执行的运维任务，支持暂停/继续/取消">
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <StatCard label="执行中" value={stats.running} tone="default" icon={<Activity className="size-4" />} />
        <StatCard label="成功" value={stats.success} tone="emerald" icon={<CheckCircle2 className="size-4" />} />
        <StatCard label="失败" value={stats.failed} tone="red" icon={<XCircle className="size-4" />} />
        <StatCard label="等待/暂停" value={stats.pending} tone="amber" icon={<Clock className="size-4" />} />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-64 pl-9"
            placeholder="任务名称"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <Input
          className="w-40"
          placeholder="创建人"
          value={creator}
          onChange={(e) => {
            setCreator(e.target.value)
            setPage(1)
          }}
        />
        <Select
          value={status || 'all'}
          onValueChange={(v) => {
            setStatus(v === 'all' ? '' : (v as OpsTask['status']))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            {TASK_STATUS_VALUES.map((s) => (
              <SelectItem key={s} value={s}>
                {TASK_STATUS_LABEL[s]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
          <RefreshCcw className={cn(isFetching && 'animate-spin')} /> 刷新
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
              <EmptyRow colSpan={cols.length}>暂无任务</EmptyRow>
            ) : (
              rows.map((t) => (
                <TableRow key={t.id}>
                  <TableCell>
                    <button
                      type="button"
                      className="text-left font-medium hover:underline"
                      onClick={() => navigate(`/ops/tasks`)}
                    >
                      {t.taskName}
                    </button>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {t.templateId ? templateNameMap[t.templateId] ?? t.templateId : '—'}
                  </TableCell>
                  <TableCell className="tabular-nums">{t.totalCount}</TableCell>
                  <TableCell>
                    <Badge variant={TASK_STATUS_VARIANT[t.status]}>
                      {TASK_STATUS_LABEL[t.status]}
                    </Badge>
                  </TableCell>
                  <TableCell className="min-w-[160px]">
                    <ProgressBar value={t.progress} status={t.status} />
                    <div className="mt-1 text-[11px] text-muted-foreground tabular-nums">
                      步骤 {t.currentStep}/{t.totalSteps}
                      {(t.status === 'success' || t.status === 'failed') &&
                        ` · 成功 ${t.successCount} / 失败 ${t.failCount}`}
                    </div>
                  </TableCell>
                  <TableCell className="text-xs">{t.creator}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(t.createdAt)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => navigate(`/ops/tasks`)}
                        title="详情"
                      >
                        <Eye className="size-4" />
                      </Button>
                      {t.status === 'running' && (
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={busy}
                          onClick={() => pauseTask.mutate(t.id)}
                          title="暂停"
                        >
                          <PauseCircle className="size-4" />
                        </Button>
                      )}
                      {(t.status === 'paused' || t.status === 'pending') && (
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={busy}
                          onClick={() => resumeTask.mutate(t.id)}
                          title="继续"
                        >
                          <PlayCircle className="size-4" />
                        </Button>
                      )}
                      {(t.status === 'running' ||
                        t.status === 'paused' ||
                        t.status === 'pending') && (
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={busy}
                          onClick={() => {
                            if (window.confirm(`确认取消任务「${t.taskName}」？`)) {
                              cancelTask.mutate(t.id)
                            }
                          }}
                          title="取消"
                          className="text-destructive hover:text-destructive"
                        >
                          <Ban className="size-4" />
                        </Button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={setPage} />
    </PageShell>
  )
}
