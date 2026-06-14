import { useMemo, useState } from 'react'
import {
  RefreshCcw,
  Search,
  ListChecks,
  LayoutTemplate,
  PauseCircle,
  PlayCircle,
  Ban,
  Trash2,
  X,
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
  useDeleteOpsTemplates,
} from '@core/hooks/api/useOpsTools'
import type { OpsTask, OpsTemplate, OpsStep } from '@core/mock/data/opsTools'

// ============================================================
// 运维工具箱 — 自动化任务 + 运维模板（适配 v1 业务深度）
// ============================================================

type TabKey = 'tasks' | 'templates'

// ----- 任务状态映射 -----
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

// ----- 模板分类映射 -----
const TEMPLATE_CATEGORIES = [
  '巡检运维',
  '故障处置',
  '性能优化',
  '软件管理',
  '网络配置',
  '维护操作',
] as const

const CATEGORY_VARIANT: Record<string, 'default' | 'warning' | 'success' | 'destructive' | 'secondary'> = {
  巡检运维: 'default',
  故障处置: 'destructive',
  性能优化: 'success',
  软件管理: 'secondary',
  网络配置: 'default',
  维护操作: 'warning',
}

const DEVICE_TYPES = ['eNB', 'gNB', 'CPE', 'eGW'] as const

const STEP_TYPE_LABEL: Record<OpsStep['stepType'], string> = {
  mml: 'MML',
  check: '检查',
  wait: '等待',
  notify: '通知',
  script: '脚本',
}

const STEP_TYPE_VARIANT: Record<OpsStep['stepType'], 'default' | 'warning' | 'success' | 'secondary' | 'muted'> = {
  mml: 'default',
  check: 'success',
  wait: 'warning',
  notify: 'secondary',
  script: 'muted',
}

function formatDuration(seconds: number): string {
  if (!seconds) return '—'
  if (seconds >= 60) return `${Math.floor(seconds / 60)} 分`
  return `${seconds} 秒`
}

// ============================================================
// 统计卡片
// ============================================================
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

// ============================================================
// 进度条
// ============================================================
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

// ============================================================
// 主页面
// ============================================================
export function OpsPage() {
  const [tab, setTab] = useState<TabKey>('tasks')

  return (
    <PageShell title="运维工具箱" description="自动化任务编排与运维模板库">
      {/* 主切换：任务 / 模板 */}
      <div className="mb-4 inline-flex items-center gap-1 rounded-lg border bg-card p-1">
        <Button
          variant={tab === 'tasks' ? 'default' : 'ghost'}
          size="sm"
          onClick={() => setTab('tasks')}
        >
          <ListChecks className="size-4" /> 自动化任务
        </Button>
        <Button
          variant={tab === 'templates' ? 'default' : 'ghost'}
          size="sm"
          onClick={() => setTab('templates')}
        >
          <LayoutTemplate className="size-4" /> 运维模板
        </Button>
      </div>

      {tab === 'tasks' ? <TasksPanel /> : <TemplatesPanel />}
    </PageShell>
  )
}

// ============================================================
// 任务面板
// ============================================================
function TasksPanel() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState<OpsTask['status'] | ''>('')
  const [creator, setCreator] = useState('')
  const [selected, setSelected] = useState<OpsTask | null>(null)

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

  // 模板名映射（用于显示任务关联模板名）
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

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  // 当前页统计（后端未返回 stats，基于当前页数据派生）
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
    <>
      {/* 统计卡片 */}
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <StatCard label="执行中" value={stats.running} tone="default" icon={<Activity className="size-4" />} />
        <StatCard label="成功" value={stats.success} tone="emerald" icon={<CheckCircle2 className="size-4" />} />
        <StatCard label="失败" value={stats.failed} tone="red" icon={<XCircle className="size-4" />} />
        <StatCard label="等待/暂停" value={stats.pending} tone="amber" icon={<Clock className="size-4" />} />
      </div>

      {/* 筛选 */}
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
                      onClick={() => setSelected(t)}
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
                      {t.status === 'success' ||
                      t.status === 'failed' ||
                      t.status === 'cancelled' ? (
                        <span className="text-xs text-muted-foreground">—</span>
                      ) : null}
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={setPage} />

      {selected ? (
        <TaskDetailDrawer
          task={selected}
          templateName={
            selected.templateId ? templateNameMap[selected.templateId] : undefined
          }
          onClose={() => setSelected(null)}
        />
      ) : null}
    </>
  )
}

// ============================================================
// 任务详情抽屉
// ============================================================
function TaskDetailDrawer({
  task,
  templateName,
  onClose,
}: {
  task: OpsTask
  templateName?: string
  onClose: () => void
}) {
  return (
    <div className="fixed inset-0 z-50 flex justify-end" role="dialog" aria-modal="true">
      <button
        type="button"
        aria-label="关闭"
        className="absolute inset-0 bg-black/30"
        onClick={onClose}
      />
      <div className="relative h-full w-full max-w-xl overflow-y-auto border-l bg-card p-6 shadow-xl">
        <div className="mb-4 flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold">任务详情</h2>
            <p className="mt-0.5 text-sm text-muted-foreground">{task.taskName}</p>
          </div>
          <Button variant="ghost" size="sm" onClick={onClose}>
            <X className="size-4" />
          </Button>
        </div>

        <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
          <Field label="状态">
            <Badge variant={TASK_STATUS_VARIANT[task.status]}>
              {TASK_STATUS_LABEL[task.status]}
            </Badge>
          </Field>
          <Field label="关联模板">{templateName ?? task.templateId ?? '—'}</Field>
          <Field label="进度" full>
            <ProgressBar value={task.progress} status={task.status} />
            <div className="mt-1 text-xs text-muted-foreground tabular-nums">
              {task.progress}% · 步骤 {task.currentStep}/{task.totalSteps}
            </div>
          </Field>
          <Field label="设备总数">
            <span className="tabular-nums">{task.totalCount}</span>
          </Field>
          <Field label="成功 / 失败">
            <span className="tabular-nums">
              <span className="text-emerald-600 dark:text-emerald-400">{task.successCount}</span>
              {' / '}
              <span className={task.failCount > 0 ? 'text-destructive' : ''}>
                {task.failCount}
              </span>
            </span>
          </Field>
          <Field label="创建人">{task.creator}</Field>
          <Field label="创建时间">{formatTime(task.createdAt)}</Field>
          {task.startedAt ? <Field label="开始时间">{formatTime(task.startedAt)}</Field> : null}
          {task.completedAt ? (
            <Field label="完成时间">{formatTime(task.completedAt)}</Field>
          ) : null}
          {task.message ? (
            <Field label="执行信息" full>
              <span className="text-muted-foreground">{task.message}</span>
            </Field>
          ) : null}
        </dl>

        <div className="mt-6">
          <div className="mb-2 text-sm font-medium">
            目标设备（{task.deviceSns.length}）
          </div>
          <div className="max-h-48 overflow-y-auto rounded-md border bg-muted/30 p-3 font-mono text-xs">
            {task.deviceSns.length > 0 ? task.deviceSns.join(', ') : '—'}
          </div>
        </div>
      </div>
    </div>
  )
}

// ============================================================
// 模板面板
// ============================================================
function TemplatesPanel() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [category, setCategory] = useState('')
  const [targetDeviceType, setTargetDeviceType] = useState('')
  const [selected, setSelected] = useState<OpsTemplate | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(category ? { category } : {}),
      ...(targetDeviceType ? { targetDeviceType } : {}),
    }),
    [page, pageSize, keyword, category, targetDeviceType]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useOpsTemplates(params)
  const deleteTemplates = useDeleteOpsTemplates()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['模板', '分类', '适用设备', '步骤数', '预计耗时', '使用次数', '创建人', '更新时间', '操作']

  return (
    <>
      {/* 筛选 */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-64 pl-9"
            placeholder="模板名称"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <Select
          value={category || 'all'}
          onValueChange={(v) => {
            setCategory(v === 'all' ? '' : v)
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="分类" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部分类</SelectItem>
            {TEMPLATE_CATEGORIES.map((c) => (
              <SelectItem key={c} value={c}>
                {c}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={targetDeviceType || 'all'}
          onValueChange={(v) => {
            setTargetDeviceType(v === 'all' ? '' : v)
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="适用设备" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部设备类型</SelectItem>
            {DEVICE_TYPES.map((d) => (
              <SelectItem key={d} value={d}>
                {d}
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
              <EmptyRow colSpan={cols.length}>暂无模板</EmptyRow>
            ) : (
              rows.map((t) => (
                <TableRow key={t.id}>
                  <TableCell>
                    <button
                      type="button"
                      className="text-left font-medium hover:underline"
                      onClick={() => setSelected(t)}
                    >
                      {t.templateName}
                    </button>
                    <div className="text-xs text-muted-foreground line-clamp-1">
                      {t.description}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={CATEGORY_VARIANT[t.category] ?? 'outline'}>
                      {t.category || '—'}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs">
                    <div className="flex flex-wrap gap-1">
                      {(t.targetDeviceTypes || []).length > 0
                        ? t.targetDeviceTypes.map((d) => (
                            <Badge key={d} variant="muted">
                              {d}
                            </Badge>
                          ))
                        : '—'}
                    </div>
                  </TableCell>
                  <TableCell className="text-xs tabular-nums">{t.steps?.length ?? 0}</TableCell>
                  <TableCell className="text-xs tabular-nums">
                    {formatDuration(t.estimatedDuration)}
                  </TableCell>
                  <TableCell className="text-xs tabular-nums">{t.useCount}</TableCell>
                  <TableCell className="text-xs">{t.creator}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(t.updateTime)}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={deleteTemplates.isPending}
                      className="text-destructive hover:text-destructive"
                      onClick={() => {
                        if (window.confirm(`确认删除模板「${t.templateName}」？`)) {
                          deleteTemplates.mutate([t.id])
                        }
                      }}
                      title="删除"
                    >
                      <Trash2 className="size-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={setPage} />

      {selected ? (
        <TemplateDetailDrawer template={selected} onClose={() => setSelected(null)} />
      ) : null}
    </>
  )
}

// ============================================================
// 模板详情抽屉（含执行步骤）
// ============================================================
function TemplateDetailDrawer({
  template,
  onClose,
}: {
  template: OpsTemplate
  onClose: () => void
}) {
  return (
    <div className="fixed inset-0 z-50 flex justify-end" role="dialog" aria-modal="true">
      <button
        type="button"
        aria-label="关闭"
        className="absolute inset-0 bg-black/30"
        onClick={onClose}
      />
      <div className="relative h-full w-full max-w-2xl overflow-y-auto border-l bg-card p-6 shadow-xl">
        <div className="mb-4 flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold">模板详情</h2>
            <p className="mt-0.5 text-sm text-muted-foreground">{template.templateName}</p>
          </div>
          <Button variant="ghost" size="sm" onClick={onClose}>
            <X className="size-4" />
          </Button>
        </div>

        <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
          <Field label="分类">
            <Badge variant={CATEGORY_VARIANT[template.category] ?? 'outline'}>
              {template.category || '—'}
            </Badge>
          </Field>
          <Field label="使用次数">
            <span className="tabular-nums">{template.useCount}</span>
          </Field>
          <Field label="适用设备" full>
            <div className="flex flex-wrap gap-1">
              {template.targetDeviceTypes.length > 0
                ? template.targetDeviceTypes.map((d) => (
                    <Badge key={d} variant="muted">
                      {d}
                    </Badge>
                  ))
                : '—'}
            </div>
          </Field>
          <Field label="预计耗时">{formatDuration(template.estimatedDuration)}</Field>
          <Field label="创建人">{template.creator}</Field>
          {template.tags.length > 0 ? (
            <Field label="标签" full>
              <div className="flex flex-wrap gap-1">
                {template.tags.map((tag) => (
                  <Badge key={tag} variant="secondary">
                    {tag}
                  </Badge>
                ))}
              </div>
            </Field>
          ) : null}
          {template.description ? (
            <Field label="描述" full>
              <span className="text-muted-foreground">{template.description}</span>
            </Field>
          ) : null}
        </dl>

        <div className="mt-6">
          <div className="mb-3 text-sm font-medium">执行步骤（{template.steps.length}）</div>
          {template.steps.length === 0 ? (
            <div className="rounded-md border bg-muted/30 p-4 text-center text-sm text-muted-foreground">
              暂无步骤
            </div>
          ) : (
            <ol className="space-y-3">
              {template.steps.map((step) => (
                <li
                  key={step.stepNo}
                  className="relative rounded-md border bg-card p-3 pl-10"
                >
                  <span className="absolute left-3 top-3 flex size-5 items-center justify-center rounded-full bg-primary/10 text-[11px] font-semibold text-primary tabular-nums">
                    {step.stepNo}
                  </span>
                  <div className="flex items-center gap-2">
                    <Badge variant={STEP_TYPE_VARIANT[step.stepType]}>
                      {STEP_TYPE_LABEL[step.stepType]}
                    </Badge>
                    <span className="text-sm font-medium">{step.stepName}</span>
                  </div>
                  {step.description ? (
                    <div className="mt-1 text-xs text-muted-foreground">{step.description}</div>
                  ) : null}
                  {step.command ? (
                    <div className="mt-2 rounded bg-muted/50 px-2 py-1 font-mono text-xs">
                      {step.command}
                    </div>
                  ) : null}
                  {step.condition ? (
                    <div className="mt-1 text-xs text-primary">条件: {step.condition}</div>
                  ) : null}
                  {step.waitSeconds ? (
                    <div className="mt-1 text-xs text-amber-600 dark:text-amber-400">
                      等待 {step.waitSeconds} 秒
                    </div>
                  ) : null}
                  {step.notifyTarget ? (
                    <div className="mt-1 text-xs text-muted-foreground">
                      通知对象: {step.notifyTarget}
                    </div>
                  ) : null}
                  {step.rollbackCommand ? (
                    <div className="mt-1 text-xs text-destructive">
                      回退: {step.rollbackCommand}
                    </div>
                  ) : null}
                </li>
              ))}
            </ol>
          )}
        </div>
      </div>
    </div>
  )
}

// ============================================================
// 详情字段
// ============================================================
function Field({
  label,
  children,
  full,
}: {
  label: string
  children: React.ReactNode
  full?: boolean
}) {
  return (
    <div className={cn(full && 'col-span-2')}>
      <dt className="text-xs uppercase tracking-wider text-muted-foreground">{label}</dt>
      <dd className="mt-1">{children}</dd>
    </div>
  )
}
