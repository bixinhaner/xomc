import { useMemo, useState } from 'react'
import { CheckCircle2, RefreshCcw, Search, X, XCircle } from 'lucide-react'

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
import { useT } from '@/hooks/useT'
import { cn } from '@/lib/utils'

import { useMMLTaskResults, useMMLTasks } from '@core/hooks/api/useMML'
import { getMmlTaskProgress } from '@core/utils/mmlTaskProgress'
import type {
  DeviceTaskResultItem,
  MMLExecuteType,
  MMLTask,
  MMLTaskOrigin,
  MMLTaskResultsStats,
  MMLTaskStatus,
} from '@core/types/mml'

// ============================================================
// MML 任务记录 — mml_tasks 执行记录（对齐 v1 mml/TaskRecord，参照 v3 mml/task-records）
// real：useMMLTasks + useMMLTaskResults。只读列表 + 逐设备结果下钻。
// ============================================================

const STATUS_META: Record<
  MMLTaskStatus,
  { label: string; variant: 'default' | 'destructive' | 'warning' | 'muted' }
> = {
  pending: { label: '待执行', variant: 'muted' },
  running: { label: '执行中', variant: 'warning' },
  paused: { label: '已暂停', variant: 'warning' },
  completed: { label: '已完成', variant: 'default' },
  cancelled: { label: '已取消', variant: 'muted' },
  failed: { label: '失败', variant: 'destructive' },
}

const EXEC_TYPE_LABEL: Record<MMLExecuteType, string> = {
  immediate: '立即执行',
  suspended: '挂起',
  scheduled: '定时执行',
  periodic: '周期任务',
}

const TASK_ORIGIN_LABEL: Record<MMLTaskOrigin, string> = {
  console: 'mml.taskOrigin.console',
  script: 'mml.taskOrigin.script',
}

const TASK_STATUS_ORDER: MMLTaskStatus[] = [
  'pending',
  'running',
  'paused',
  'completed',
  'cancelled',
  'failed',
]

function statusMeta(s: MMLTaskStatus) {
  return STATUS_META[s] ?? { label: s, variant: 'muted' as const }
}

const PAGE_SIZE = 20
const TASK_RESULT_PAGE_SIZE = 20

export default function TaskRecord() {
  const tr = useT()
  const [page, setPage] = useState(1)
  const [taskNameInput, setTaskNameInput] = useState('')
  const [taskName, setTaskName] = useState('')
  const [taskOrigin, setTaskOrigin] = useState<MMLTaskOrigin | ''>('')
  const [status, setStatus] = useState<MMLTaskStatus | ''>('')
  const [executeType, setExecuteType] = useState<MMLExecuteType | ''>('')
  const [result, setResult] = useState<'success' | 'partial' | 'failed' | ''>('')
  const [viewing, setViewing] = useState<MMLTask | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(taskName.trim() ? { taskName: taskName.trim() } : {}),
      ...(taskOrigin ? { taskOrigin } : {}),
      ...(status ? { status } : {}),
      ...(executeType ? { executeType } : {}),
      ...(result ? { result } : {}),
    }),
    [page, taskName, taskOrigin, status, executeType, result]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useMMLTasks(params)
  const tasks = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const statusCounts = useMemo(() => {
    const m: Record<MMLTaskStatus, number> = {
      pending: 0,
      running: 0,
      paused: 0,
      completed: 0,
      cancelled: 0,
      failed: 0,
    }
    for (const t of tasks) {
      if (m[t.status] !== undefined) m[t.status] += 1
    }
    return m
  }, [tasks])

  const applySearch = () => {
    setTaskName(taskNameInput)
    setPage(1)
  }

  const cols = [
    '任务名 / ID',
    '执行人',
    tr('mml.taskOrigin'),
    '类型',
    '状态',
    '进度',
    '创建时间',
    '操作',
  ]

  return (
    <PageShell
      title="MML 任务记录"
      description="批量命令任务执行记录 · mml_tasks"
      isFetching={isFetching}
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-56 pl-9"
              placeholder="任务名称"
              value={taskNameInput}
              onChange={(e) => setTaskNameInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') applySearch()
              }}
            />
          </div>

          <Select
            value={taskOrigin || 'all'}
            onValueChange={(v) => {
              setTaskOrigin(v === 'all' ? '' : (v as MMLTaskOrigin))
              setPage(1)
            }}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder={tr('mml.taskOrigin')} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{tr('mml.taskOrigin.all')}</SelectItem>
              <SelectItem value="console">{tr('mml.taskOrigin.console')}</SelectItem>
              <SelectItem value="script">{tr('mml.taskOrigin.script')}</SelectItem>
            </SelectContent>
          </Select>

          <Select
            value={executeType || 'all'}
            onValueChange={(v) => {
              setExecuteType(v === 'all' ? '' : (v as MMLExecuteType))
              setPage(1)
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="执行类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部类型</SelectItem>
              <SelectItem value="immediate">立即执行</SelectItem>
              <SelectItem value="suspended">挂起</SelectItem>
              <SelectItem value="scheduled">定时执行</SelectItem>
              <SelectItem value="periodic">周期任务</SelectItem>
            </SelectContent>
          </Select>

          <Select
            value={status || 'all'}
            onValueChange={(v) => {
              setStatus(v === 'all' ? '' : (v as MMLTaskStatus))
              setPage(1)
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              {TASK_STATUS_ORDER.map((s) => (
                <SelectItem key={s} value={s}>
                  {statusMeta(s).label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select
            value={result || 'all'}
            onValueChange={(v) => {
              setResult(v === 'all' ? '' : (v as 'success' | 'partial' | 'failed'))
              setPage(1)
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="结果" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部结果</SelectItem>
              <SelectItem value="success">成功</SelectItem>
              <SelectItem value="partial">部分成功</SelectItem>
              <SelectItem value="failed">失败</SelectItem>
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
        </>
      }
    >
      {/* 统计卡片 */}
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4 lg:grid-cols-7">
        <Stat label="总计" value={total} />
        {TASK_STATUS_ORDER.map((s) => (
          <Stat
            key={s}
            label={statusMeta(s).label}
            value={statusCounts[s]}
            tone={
              s === 'failed' || s === 'cancelled'
                ? 'destructive'
                : s === 'completed'
                  ? 'emerald'
                  : s === 'running'
                    ? 'amber'
                    : 'default'
            }
          />
        ))}
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
            ) : tasks.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无任务记录</EmptyRow>
            ) : (
              tasks.map((t) => {
                const { done, total: tot } = getMmlTaskProgress(t)
                const pct = tot > 0 ? Math.round((done / tot) * 100) : 0
                const meta = statusMeta(t.status)
                return (
                  <TableRow key={t.id} className="align-top">
                    <TableCell>
                      <button
                        type="button"
                        className="text-left font-medium hover:underline"
                        onClick={() => setViewing(t)}
                      >
                        {t.taskName || '—'}
                      </button>
                      <div className="font-mono text-[11px] text-muted-foreground">
                        {t.id}
                      </div>
                    </TableCell>
                    <TableCell className="text-xs">{t.creator || '—'}</TableCell>
                    <TableCell className="text-xs">
                      <Badge variant="outline">
                        {TASK_ORIGIN_LABEL[t.taskOrigin] ? tr(TASK_ORIGIN_LABEL[t.taskOrigin]) : t.taskOrigin}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs">
                      {EXEC_TYPE_LABEL[t.executeType] ?? t.executeType}
                    </TableCell>
                    <TableCell>
                      <Badge variant={meta.variant}>{meta.label}</Badge>
                    </TableCell>
                    <TableCell className="min-w-[140px]">
                      <div className="flex items-center gap-2">
                        <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
                          <div
                            className={cn(
                              'h-full rounded-full',
                              t.status === 'failed' ? 'bg-destructive' : 'bg-primary'
                            )}
                            style={{ width: `${pct}%` }}
                          />
                        </div>
                        <span className="w-12 text-right font-mono text-[11px] text-muted-foreground">
                          {done}/{tot}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(t.createdAt)}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-7 px-2 text-xs"
                        onClick={() => setViewing(t)}
                      >
                        查看
                      </Button>
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
        pageSize={PAGE_SIZE}
        onChange={setPage}
      />

      {viewing ? (
        <TaskDetailDrawer key={viewing.id} task={viewing} onClose={() => setViewing(null)} />
      ) : null}
    </PageShell>
  )
}

// ---------------------------------------------------------------------------
// 详情抽屉 — 逐设备执行结果
// ---------------------------------------------------------------------------

function TaskDetailDrawer({ task, onClose }: { task: MMLTask; onClose: () => void }) {
  const t = useT()
  const [resultPage, setResultPage] = useState(1)
  const { data, isLoading, isError } = useMMLTaskResults(task.id, resultPage, TASK_RESULT_PAGE_SIZE)
  const rows: DeviceTaskResultItem[] = useMemo(
    () => data?.items ?? [],
    [data]
  )
  // apiSwitch 把真实 API（stats: MMLTaskResultsStats）与 mock（默认 stats {}）
  // 的返回类型取交集，stats 被收窄；运行期真实 API 会填充翻译审计元数据，
  // 这里按已声明的 MMLTaskResultsStats 收窄读取。
  const stats = data?.stats as MMLTaskResultsStats | undefined
  const meta = statusMeta(task.status)

  return (
    <div className="fixed inset-0 z-50 flex justify-end" role="dialog" aria-modal="true">
      <div className="absolute inset-0 bg-black/40" onClick={onClose} aria-hidden />
      <div className="relative flex h-full w-full max-w-[760px] flex-col overflow-hidden border-l bg-background shadow-xl">
        <div className="flex items-center justify-between border-b px-4 py-3">
          <div className="min-w-0">
            <div className="truncate text-base font-semibold">
              {task.taskName || '执行结果'}
            </div>
            <div className="truncate font-mono text-[11px] text-muted-foreground">
              {task.id}
            </div>
          </div>
          <Button variant="ghost" size="icon" onClick={onClose} aria-label="关闭">
            <X />
          </Button>
        </div>

        <div className="grid grid-cols-2 gap-3 border-b px-4 py-3 sm:grid-cols-4">
          <Field label="状态" value={<Badge variant={meta.variant}>{meta.label}</Badge>} />
          <Field
            label={t('mml.taskOrigin')}
            value={TASK_ORIGIN_LABEL[task.taskOrigin] ? t(TASK_ORIGIN_LABEL[task.taskOrigin]) : task.taskOrigin}
          />
          <Field
            label="类型"
            value={EXEC_TYPE_LABEL[task.executeType] ?? task.executeType}
          />
          <Field label="设备数" value={String(task.totalDevices ?? 0)} />
          <Field
            label="成功 / 失败"
            value={`${task.successCount ?? 0} / ${task.failedCount ?? 0}`}
          />
          <Field label="创建" value={formatTime(task.createdAt)} />
          <Field label="开始" value={formatTime(task.startedAt)} />
          <Field label="结束" value={formatTime(task.finishedAt)} />
          <Field label="执行人" value={task.creator || '—'} />
        </div>

        {task.productResolved === false ? (
          <div className="mx-4 mt-3 rounded border border-orange-500/40 bg-orange-500/5 px-3 py-2 font-mono text-[11px] text-orange-600 dark:text-orange-400">
            设备 product_class 未匹配产品，所有 path 走原路径下发（orphan_passthrough）。建议运维补登记 product_class_patterns。
          </div>
        ) : null}
        {stats?.pathTranslationSource && stats.pathTranslationSource !== 'discovered' ? (
          <div className="mx-4 mt-2 font-mono text-[11px] text-muted-foreground">
            路径翻译来源：{stats.pathTranslationSource}
            {stats.matchedProductClass ? ` · ${stats.matchedProductClass}` : ''}
          </div>
        ) : null}

        <div className="min-h-0 flex-1 overflow-auto px-4 py-3">
          <div className="mb-2 text-xs uppercase tracking-wider text-muted-foreground">
            逐设备执行结果
          </div>
          {isLoading ? (
            <div className="py-10 text-center text-sm text-muted-foreground">
              加载中…
            </div>
          ) : isError ? (
            <div className="py-10 text-center text-sm text-destructive">
              结果加载失败
            </div>
          ) : rows.length === 0 ? (
            <div className="py-10 text-center text-sm text-muted-foreground">
              暂无设备结果
            </div>
          ) : (
            <div className="space-y-2">
              {rows.map((r, i) => {
                const ok = r.result?.success
                return (
                  <div
                    key={`${r.deviceSn}-${r.deviceTaskId ?? i}`}
                    className="rounded border bg-muted/30 p-2.5"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <div className="flex items-center gap-2">
                        {ok ? (
                          <CheckCircle2 className="size-3.5 text-emerald-500" />
                        ) : (
                          <XCircle className="size-3.5 text-destructive" />
                        )}
                        <span className="font-mono text-xs">{r.deviceSn}</span>
                        {r.deviceName ? (
                          <span className="text-[11px] text-muted-foreground">
                            {r.deviceName}
                          </span>
                        ) : null}
                        {r.planLineNo ? (
                          <span className="rounded border bg-background px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground">
                            #{r.planLineNo}/{r.planOrder ?? '-'} {r.commandCode ?? ''}
                          </span>
                        ) : null}
                      </div>
                      <Badge variant={ok ? 'default' : 'destructive'}>
                        {ok ? '成功' : '失败'}
                      </Badge>
                    </div>
                    {r.failReason ? (
                      <div className="mt-1 font-mono text-[11px] text-destructive">
                        {r.failReason}
                      </div>
                    ) : null}
                    {r.result?.rawOutput ? (
                      <pre className="mt-1.5 max-h-40 overflow-auto whitespace-pre-wrap rounded bg-background p-2 font-mono text-[11px] leading-relaxed">
                        {r.result.rawOutput}
                      </pre>
                    ) : null}
                  </div>
                )
              })}
            </div>
          )}
          {(data?.total ?? 0) > TASK_RESULT_PAGE_SIZE ? (
            <div className="mt-3">
              <Pagination
                page={resultPage}
                totalPages={Math.max(1, Math.ceil((data?.total ?? 0) / TASK_RESULT_PAGE_SIZE))}
                pageSize={TASK_RESULT_PAGE_SIZE}
                onChange={setResultPage}
              />
            </div>
          ) : null}
        </div>
      </div>
    </div>
  )
}

function Field({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className="mt-0.5 truncate text-sm">{value}</div>
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
  tone?: 'default' | 'emerald' | 'amber' | 'destructive'
}) {
  const toneClass = {
    default: 'text-foreground',
    emerald: 'text-emerald-600 dark:text-emerald-400',
    amber: 'text-amber-600 dark:text-amber-400',
    destructive: 'text-destructive',
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
