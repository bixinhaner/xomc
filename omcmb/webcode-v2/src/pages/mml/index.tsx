import { useMemo, useState } from 'react'
import {
  ListTree,
  RefreshCcw,
  Search,
  Terminal,
  Trash2,
  X,
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
  useMMLScripts,
  useMMLTasks,
  useMMLCommands,
  useMMLTaskResults,
  useDeleteMMLScripts,
} from '@core/hooks/api/useMML'
import type {
  MMLScript,
  MMLTask,
  MMLCommand,
  MMLTaskStatus,
  MMLExecuteType,
  DeviceTaskResultItem,
} from '@core/types/mml'

// ============================================================
// MML 人机命令 — 脚本库 / 任务记录 / 命令字典
// 对齐 v1 (webcode) 的 ScriptTask / TaskRecord / CommandTree 三个列表维度，
// 含关键统计、筛选、查看详情、删除脚本等主操作。
// ============================================================

type Tab = 'scripts' | 'tasks' | 'commands'

const TABS: { key: Tab; label: string; icon: typeof Terminal }[] = [
  { key: 'scripts', label: '脚本库', icon: ListTree },
  { key: 'tasks', label: '任务记录', icon: Terminal },
  { key: 'commands', label: '命令字典', icon: ListTree },
]

// ---- 任务执行方式 / 状态映射 ----------------------------------------------

const EXECUTE_TYPE_LABEL: Record<MMLExecuteType, string> = {
  immediate: '立即执行',
  suspended: '挂起',
  scheduled: '定时执行',
  periodic: '周期任务',
}

const TASK_STATUS_META: Record<
  MMLTaskStatus,
  { label: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'muted' }
> = {
  pending: { label: '待执行', variant: 'muted' },
  running: { label: '执行中', variant: 'default' },
  paused: { label: '已暂停', variant: 'warning' },
  completed: { label: '已完成', variant: 'success' },
  cancelled: { label: '已取消', variant: 'destructive' },
  failed: { label: '失败', variant: 'destructive' },
}

const SCRIPT_STATUS_META: Record<
  string,
  { label: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'muted' }
> = {
  active: { label: '可用', variant: 'success' },
  archived: { label: '已归档', variant: 'muted' },
  pending: { label: '待执行', variant: 'muted' },
  running: { label: '执行中', variant: 'default' },
  paused: { label: '已暂停', variant: 'warning' },
  completed: { label: '已完成', variant: 'success' },
  failed: { label: '失败', variant: 'destructive' },
  cancelled: { label: '已取消', variant: 'destructive' },
}

const TASK_RESULT_META: Record<
  string,
  { label: string; variant: 'success' | 'warning' | 'destructive' }
> = {
  success: { label: '全部成功', variant: 'success' },
  partial: { label: '部分成功', variant: 'warning' },
  failed: { label: '失败', variant: 'destructive' },
}

const PAGE_SIZE = 20

export function MMLPage() {
  const [tab, setTab] = useState<Tab>('scripts')

  return (
    <PageShell
      title="MML 人机命令"
      description="脚本库 · 任务执行记录 · 命令字典"
    >
      {/* Tab 切换 */}
      <div className="mb-4 inline-flex items-center gap-1 rounded-lg border bg-muted/40 p-1">
        {TABS.map((tItem) => {
          const Icon = tItem.icon
          const active = tab === tItem.key
          return (
            <button
              key={tItem.key}
              type="button"
              onClick={() => setTab(tItem.key)}
              className={cn(
                'inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                active
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              <Icon className="size-4" />
              {tItem.label}
            </button>
          )
        })}
      </div>

      {tab === 'scripts' && <ScriptsTab />}
      {tab === 'tasks' && <TasksTab />}
      {tab === 'commands' && <CommandsTab />}
    </PageShell>
  )
}

// ============================================================
// 统计卡片
// ============================================================

function Stat({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: number | string
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
      <div className="text-xs uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>{value}</div>
    </div>
  )
}

// ============================================================
// Tab 1 — 脚本库
// ============================================================

function ScriptsTab() {
  const [page, setPage] = useState(1)
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')

  const params = useMemo(
    () => ({ page, pageSize: PAGE_SIZE, ...(search.trim() ? { search: search.trim() } : {}) }),
    [page, search]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useMMLScripts(params)
  const deleteScripts = useDeleteMMLScripts()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const [detail, setDetail] = useState<MMLScript | null>(null)
  const [deleteErr, setDeleteErr] = useState<string | null>(null)

  // 当页脚本状态分布（来自已加载页面，作概览参考）
  const stats = useMemo(() => {
    const active = rows.filter((s) => s.status === 'active').length
    const batch = rows.filter((s) => s.type === 'batch').length
    return { total, active, batch }
  }, [rows, total])

  const handleDelete = (s: MMLScript) => {
    if (!window.confirm(`确认删除脚本「${s.scriptName}」？此操作不可撤销。`)) return
    setDeleteErr(null)
    deleteScripts.mutate([s.id], {
      onError: (e) => setDeleteErr(e instanceof Error ? e.message : '删除失败'),
      onSuccess: () => {
        if (detail?.id === s.id) setDetail(null)
      },
    })
  }

  const cols = ['脚本名称', '描述', '类型', '状态', '标签', '创建人', '更新时间', '操作']

  return (
    <div className="flex items-center gap-1 flex-col">
      <div className="w-full">
        <div className="mb-4 grid grid-cols-3 gap-3">
          <Stat label="脚本总数" value={stats.total} />
          <Stat label="当页可用" value={stats.active} tone="emerald" />
          <Stat label="当页批量脚本" value={stats.batch} tone="muted" />
        </div>

        {/* 工具栏 */}
        <div className="mb-3 flex flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-72 pl-9"
              placeholder="搜索脚本名称 / 描述"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  setSearch(searchInput)
                  setPage(1)
                }
              }}
            />
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setSearch(searchInput)
              setPage(1)
            }}
          >
            <Search /> 搜索
          </Button>
          <div className="ml-auto flex items-center gap-2">
            {isFetching && <span className="text-xs text-muted-foreground">加载中…</span>}
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw /> 刷新
            </Button>
          </div>
        </div>

        {deleteErr && (
          <div className="mb-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive">
            {deleteErr}
          </div>
        )}

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
                <EmptyRow colSpan={cols.length}>暂无脚本</EmptyRow>
              ) : (
                rows.map((s) => {
                  const sm = SCRIPT_STATUS_META[s.status] ?? {
                    label: s.status,
                    variant: 'muted' as const,
                  }
                  return (
                    <TableRow key={s.id}>
                      <TableCell className="font-medium">{s.scriptName}</TableCell>
                      <TableCell className="max-w-xs text-xs text-muted-foreground">
                        <span className="line-clamp-1">{s.description || '—'}</span>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">
                          {s.type === 'batch' ? '批量' : '手动'}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant={sm.variant}>{sm.label}</Badge>
                      </TableCell>
                      <TableCell className="text-xs">
                        {s.tags && s.tags.length > 0 ? (
                          <div className="flex flex-wrap gap-1">
                            {s.tags.map((tag) => (
                              <Badge key={tag} variant="secondary">
                                {tag}
                              </Badge>
                            ))}
                          </div>
                        ) : (
                          '—'
                        )}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">{s.creator}</TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {formatTime(s.updateTime)}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Button variant="ghost" size="sm" onClick={() => setDetail(s)}>
                            详情
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-destructive hover:text-destructive"
                            disabled={deleteScripts.isPending}
                            onClick={() => handleDelete(s)}
                          >
                            <Trash2 />
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

        <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
      </div>

      {detail && <ScriptDetailPanel script={detail} onClose={() => setDetail(null)} />}
    </div>
  )
}

function ScriptDetailPanel({
  script,
  onClose,
}: {
  script: MMLScript
  onClose: () => void
}) {
  return (
    <div className="mt-4 w-full rounded-lg border bg-card p-4">
      <div className="mb-3 flex items-center justify-between">
        <h3 className="text-sm font-semibold">脚本详情 · {script.scriptName}</h3>
        <Button variant="ghost" size="sm" onClick={onClose}>
          <X /> 关闭
        </Button>
      </div>
      <dl className="grid grid-cols-2 gap-x-6 gap-y-2 text-sm">
        <Field label="脚本名称" value={script.scriptName} />
        <Field label="创建人" value={script.creator} />
        <Field label="类型" value={script.type === 'batch' ? '批量' : '手动'} />
        <Field
          label="状态"
          value={(SCRIPT_STATUS_META[script.status] ?? { label: script.status }).label}
        />
        <Field label="创建时间" value={formatTime(script.createTime)} />
        <Field label="更新时间" value={formatTime(script.updateTime)} />
        <Field label="描述" value={script.description || '—'} span />
        <Field
          label="标签"
          value={script.tags && script.tags.length > 0 ? script.tags.join(', ') : '—'}
          span
        />
      </dl>
      <div className="mt-3">
        <div className="mb-1 text-xs font-medium text-muted-foreground">脚本内容</div>
        <pre className="max-h-72 overflow-auto rounded-md border bg-muted/40 p-3 font-mono text-xs">
          {script.content || '（空）'}
        </pre>
      </div>
    </div>
  )
}

function Field({ label, value, span }: { label: string; value: string; span?: boolean }) {
  return (
    <div className={span ? 'col-span-2' : undefined}>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 break-words">{value}</dd>
    </div>
  )
}

// ============================================================
// Tab 2 — 任务记录
// ============================================================

type StatusFilter = '' | MMLTaskStatus
type ExecFilter = '' | MMLExecuteType

function TasksTab() {
  const [page, setPage] = useState(1)
  const [nameInput, setNameInput] = useState('')
  const [taskName, setTaskName] = useState('')
  const [status, setStatus] = useState<StatusFilter>('')
  const [executeType, setExecuteType] = useState<ExecFilter>('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(taskName.trim() ? { taskName: taskName.trim() } : {}),
      ...(status ? { status } : {}),
      ...(executeType ? { executeType } : {}),
    }),
    [page, taskName, status, executeType]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useMMLTasks(params)

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const [viewing, setViewing] = useState<MMLTask | null>(null)

  // 当页执行结果分布概览
  const stats = useMemo(() => {
    const running = rows.filter((t) => t.status === 'running').length
    const completed = rows.filter((t) => t.status === 'completed').length
    const failed = rows.filter((t) => t.status === 'failed' || t.status === 'cancelled').length
    return { total, running, completed, failed }
  }, [rows, total])

  const cols = ['任务名称', '创建人', '类型', '状态', '进度', '结果', '开始时间', '结束时间', '操作']

  return (
    <div>
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="任务总数" value={stats.total} />
        <Stat label="当页执行中" value={stats.running} tone="default" />
        <Stat label="当页已完成" value={stats.completed} tone="emerald" />
        <Stat label="当页失败/取消" value={stats.failed} tone="red" />
      </div>

      {/* 工具栏 */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-60 pl-9"
            placeholder="搜索任务名称"
            value={nameInput}
            onChange={(e) => setNameInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                setTaskName(nameInput)
                setPage(1)
              }
            }}
          />
        </div>

        <Select
          value={executeType || 'all'}
          onValueChange={(v) => {
            setExecuteType(v === 'all' ? '' : (v as MMLExecuteType))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="执行方式" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部方式</SelectItem>
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
          <SelectTrigger className="w-36">
            <SelectValue placeholder="状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="pending">待执行</SelectItem>
            <SelectItem value="running">执行中</SelectItem>
            <SelectItem value="paused">已暂停</SelectItem>
            <SelectItem value="completed">已完成</SelectItem>
            <SelectItem value="cancelled">已取消</SelectItem>
            <SelectItem value="failed">失败</SelectItem>
          </SelectContent>
        </Select>

        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            setTaskName(nameInput)
            setPage(1)
          }}
        >
          <Search /> 搜索
        </Button>

        <div className="ml-auto flex items-center gap-2">
          {isFetching && <span className="text-xs text-muted-foreground">加载中…</span>}
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
          </Button>
        </div>
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
              <EmptyRow colSpan={cols.length}>暂无任务记录</EmptyRow>
            ) : (
              rows.map((task) => {
                const sm = TASK_STATUS_META[task.status] ?? {
                  label: task.status,
                  variant: 'muted' as const,
                }
                const done = (task.successCount ?? 0) + (task.failedCount ?? 0)
                const totalUnits =
                  (task.totalDevices ?? 0) * Math.max(1, task.commands?.length ?? 1)
                const resultMeta = task.result ? TASK_RESULT_META[task.result] : undefined
                return (
                  <TableRow key={task.id}>
                    <TableCell className="font-medium">{task.taskName}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{task.creator}</TableCell>
                    <TableCell>
                      <Badge variant="outline">
                        {EXECUTE_TYPE_LABEL[task.executeType] ?? task.executeType}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={sm.variant}>{sm.label}</Badge>
                    </TableCell>
                    <TableCell className="text-xs tabular-nums">
                      {done}/{totalUnits}
                    </TableCell>
                    <TableCell>
                      {resultMeta ? (
                        <Badge variant={resultMeta.variant}>{resultMeta.label}</Badge>
                      ) : (
                        <span className="text-xs text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(task.startedAt)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(task.finishedAt)}
                    </TableCell>
                    <TableCell>
                      <Button variant="ghost" size="sm" onClick={() => setViewing(task)}>
                        查看结果
                      </Button>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />

      {viewing && <TaskResultPanel task={viewing} onClose={() => setViewing(null)} />}
    </div>
  )
}

function TaskResultPanel({ task, onClose }: { task: MMLTask; onClose: () => void }) {
  const { data, isLoading, isError, error } = useMMLTaskResults(task.id, 1, 200)

  const rows: DeviceTaskResultItem[] = useMemo(
    () => data?.items ?? task.results ?? [],
    [data, task.results]
  )

  const cols = ['设备 SN', '命令', '状态', '结果', '失败原因', '完成时间']

  return (
    <div className="mt-4 rounded-lg border bg-card p-4">
      <div className="mb-3 flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold">执行结果 · {task.taskName}</h3>
          <p className="mt-0.5 text-xs text-muted-foreground">
            设备 {task.totalDevices ?? 0} · 成功 {task.successCount ?? 0} · 失败{' '}
            {task.failedCount ?? 0}
          </p>
        </div>
        <Button variant="ghost" size="sm" onClick={onClose}>
          <X /> 关闭
        </Button>
      </div>

      {task.orphanCommandCodes && task.orphanCommandCodes.length > 0 && (
        <div className="mb-3 rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-600 dark:text-amber-400">
          以下命令已下线，未实际下发：{task.orphanCommandCodes.join(', ')}
        </div>
      )}
      {task.productResolved === false && (
        <div className="mb-3 rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-600 dark:text-amber-400">
          设备产品型号未匹配，path 走原路径透传下发，建议补登记 product_class。
        </div>
      )}

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
              <EmptyRow colSpan={cols.length}>暂无执行结果</EmptyRow>
            ) : (
              rows.map((r, i) => (
                <TableRow key={`${r.deviceSn}-${r.deviceTaskId ?? i}`}>
                  <TableCell className="font-mono text-xs">{r.deviceSn || '—'}</TableCell>
                  <TableCell className="text-xs">
                    <span className="line-clamp-1">{r.mmlScript || '—'}</span>
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        r.status === 'completed'
                          ? 'success'
                          : r.status === 'running'
                            ? 'default'
                            : 'muted'
                      }
                    >
                      {r.status === 'completed'
                        ? '完成'
                        : r.status === 'running'
                          ? '执行中'
                          : r.status === 'pending'
                            ? '待执行'
                            : '—'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={r.result?.success ? 'success' : 'destructive'}>
                      {r.result?.success ? '成功' : '失败'}
                    </Badge>
                  </TableCell>
                  <TableCell className="max-w-xs text-xs text-destructive">
                    <span className="line-clamp-1">{r.failReason || '—'}</span>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(r.finishedAt)}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>
    </div>
  )
}

// ============================================================
// Tab 3 — 命令字典
// ============================================================

function CommandsTab() {
  const [page, setPage] = useState(1)
  const [kwInput, setKwInput] = useState('')
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({ page, pageSize: PAGE_SIZE, ...(keyword.trim() ? { keyword: keyword.trim() } : {}) }),
    [page, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useMMLCommands(params)

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const [detail, setDetail] = useState<MMLCommand | null>(null)

  const cols = ['命令名称', '命令码', '操作', '分类', '描述', '参数', '操作']

  return (
    <div>
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-3">
        <Stat label="命令总数" value={total} />
        <Stat label="当页命令" value={rows.length} tone="muted" />
        <Stat
          label="当页含参数命令"
          value={rows.filter((c) => (c.paramRefs?.length ?? c.params.length) > 0).length}
          tone="default"
        />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-72 pl-9"
            placeholder="搜索命令名 / 命令码 / 描述"
            value={kwInput}
            onChange={(e) => setKwInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                setKeyword(kwInput)
                setPage(1)
              }
            }}
          />
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            setKeyword(kwInput)
            setPage(1)
          }}
        >
          <Search /> 搜索
        </Button>
        <div className="ml-auto flex items-center gap-2">
          {isFetching && <span className="text-xs text-muted-foreground">加载中…</span>}
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
          </Button>
        </div>
      </div>

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c, i) => (
                <TableHead key={`${c}-${i}`}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无命令</EmptyRow>
            ) : (
              rows.map((c) => {
                const paramCount = c.paramRefs?.length ?? c.params.length
                return (
                  <TableRow key={c.id}>
                    <TableCell className="font-medium">{c.commandName}</TableCell>
                    <TableCell className="font-mono text-xs">{c.commandCode}</TableCell>
                    <TableCell>
                      {c.operationType ? (
                        <Badge variant="outline">{c.operationType}</Badge>
                      ) : (
                        <span className="text-xs text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {c.category || '—'}
                    </TableCell>
                    <TableCell className="max-w-sm text-xs text-muted-foreground">
                      <span className="line-clamp-1">{c.description || '—'}</span>
                    </TableCell>
                    <TableCell className="text-xs tabular-nums">{paramCount}</TableCell>
                    <TableCell>
                      <Button variant="ghost" size="sm" onClick={() => setDetail(c)}>
                        详情
                      </Button>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />

      {detail && <CommandDetailPanel command={detail} onClose={() => setDetail(null)} />}
    </div>
  )
}

function CommandDetailPanel({ command, onClose }: { command: MMLCommand; onClose: () => void }) {
  const refs = command.paramRefs ?? []
  return (
    <div className="mt-4 rounded-lg border bg-card p-4">
      <div className="mb-3 flex items-center justify-between">
        <h3 className="text-sm font-semibold">
          命令详情 · <span className="font-mono">{command.commandCode}</span>
        </h3>
        <Button variant="ghost" size="sm" onClick={onClose}>
          <X /> 关闭
        </Button>
      </div>
      <dl className="grid grid-cols-2 gap-x-6 gap-y-2 text-sm">
        <Field label="命令名称" value={command.commandName} />
        <Field label="命令码" value={command.commandCode} />
        <Field label="操作类型" value={command.operationType || '—'} />
        <Field label="分类" value={command.category || '—'} />
        <Field label="描述" value={command.description || '—'} span />
        {command.targetObject && <Field label="目标对象" value={command.targetObject} span />}
      </dl>

      {refs.length > 0 ? (
        <div className="mt-3">
          <div className="mb-1 text-xs font-medium text-muted-foreground">
            绑定参数（{refs.length}）
          </div>
          <TableCard>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>参数名</TableHead>
                  <TableHead>TR-069 路径</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>可写</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {refs.map((p) => (
                  <TableRow key={p.id}>
                    <TableCell className="text-xs">{p.paramNameZh || p.paramCode}</TableCell>
                    <TableCell className="font-mono text-xs">{p.tr069Path}</TableCell>
                    <TableCell className="text-xs">{p.valueType}</TableCell>
                    <TableCell>
                      <Badge variant={p.isWritable ? 'success' : 'muted'}>
                        {p.isWritable ? '可写' : '只读'}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableCard>
        </div>
      ) : (
        <p className="mt-3 text-xs text-muted-foreground">该命令无绑定参数。</p>
      )}
    </div>
  )
}
