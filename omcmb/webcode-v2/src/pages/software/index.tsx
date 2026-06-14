import { useMemo, useState, type ReactNode } from 'react'
import {
  Boxes,
  CheckCircle2,
  ListChecks,
  Package,
  Pause,
  Play,
  RefreshCcw,
  RotateCcw,
  Square,
  Star,
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
  formatBytes,
  formatTime,
} from '@/components/layout/PageShell'

import {
  useSoftwareVersions,
  useUpgradeTasks,
  useSubTasks,
  useSuspendTask,
  useResumeTask,
  useTerminateTask,
  useDeleteTask,
  useRetryTask,
} from '@core/hooks/api/useSoftware'
import { useProductClasses } from '@core/hooks/api/useDevices'
import type {
  SoftwareVersion,
  VersionStatus,
  UpgradeTaskInfo,
  TaskStatusType,
  TaskResultType,
  TaskTypeValue,
  UpgradeSubTaskInfo,
  SubTaskStatusType,
} from '@core/mock/data/software'

type BadgeVariant =
  | 'default'
  | 'secondary'
  | 'destructive'
  | 'warning'
  | 'success'
  | 'muted'
  | 'outline'

type Tab = 'versions' | 'tasks'

// ---------------------------------------------------------------------------
// Display maps
// ---------------------------------------------------------------------------

const VERSION_STATUS: Record<VersionStatus, { label: string; variant: BadgeVariant }> = {
  current: { label: '当前', variant: 'success' },
  beta: { label: '测试', variant: 'warning' },
  deprecated: { label: '已废弃', variant: 'muted' },
  archived: { label: '已归档', variant: 'muted' },
}

const TASK_STATUS: Record<TaskStatusType, { label: string; variant: BadgeVariant }> = {
  pending: { label: '等待中', variant: 'muted' },
  in_progress: { label: '执行中', variant: 'default' },
  suspended: { label: '已暂停', variant: 'warning' },
  ended: { label: '已结束', variant: 'success' },
}

const TASK_RESULT: Record<TaskResultType, { label: string; variant: BadgeVariant }> = {
  success: { label: '成功', variant: 'success' },
  partial: { label: '部分成功', variant: 'warning' },
  failed: { label: '失败', variant: 'destructive' },
  terminated: { label: '已终止', variant: 'muted' },
}

const TASK_TYPE: Record<TaskTypeValue, string> = {
  1: '软件升级',
  2: '版本回退',
  4: '补丁升级',
  6: 'FPGA升级',
  8: '其他',
}

const SUB_TASK_STATUS: Record<SubTaskStatusType, { label: string; variant: BadgeVariant }> = {
  pending: { label: '等待中', variant: 'muted' },
  downloading: { label: '下载中', variant: 'default' },
  rebooting: { label: '重启中', variant: 'default' },
  verifying: { label: '校验中', variant: 'default' },
  completed: { label: '完成', variant: 'success' },
  failed: { label: '失败', variant: 'destructive' },
  suspended: { label: '已暂停', variant: 'warning' },
  terminated: { label: '已终止', variant: 'muted' },
}

const FILE_TYPE_LABEL: Record<number, string> = {
  0: '固件',
  1: '补丁',
  5: '配置',
  6: 'FPGA',
}

function progressPct(t: UpgradeTaskInfo): number {
  if (!t.totalCount) return 0
  return Math.round(((t.successCount + t.failCount) / t.totalCount) * 100)
}

// ---------------------------------------------------------------------------
// Stat tile (local, no shared-component edits)
// ---------------------------------------------------------------------------

function Stat({
  label,
  value,
  icon: Icon,
  tone = 'default',
}: {
  label: string
  value: number | string
  icon: typeof Package
  tone?: 'default' | 'emerald' | 'amber' | 'muted'
}) {
  const toneClass = {
    default: 'text-foreground',
    emerald: 'text-emerald-600 dark:text-emerald-400',
    amber: 'text-amber-600 dark:text-amber-400',
    muted: 'text-muted-foreground',
  }[tone]
  return (
    <div className="flex items-center gap-3 rounded-lg border bg-card px-4 py-3">
      <div className="rounded-md bg-muted p-2 text-muted-foreground">
        <Icon className="size-4" />
      </div>
      <div className="min-w-0">
        <div className="text-xs uppercase tracking-wider text-muted-foreground">
          {label}
        </div>
        <div className={`mt-0.5 text-xl font-semibold tabular-nums ${toneClass}`}>
          {value}
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export function SoftwarePage() {
  const [tab, setTab] = useState<Tab>('versions')

  return (
    <PageShell
      title="软件管理"
      description="设备固件版本库 + 升级任务编排（含暂停/恢复/终止/重试与子任务下钻）"
    >
      <div className="mb-4 flex items-center gap-2">
        <TabButton active={tab === 'versions'} onClick={() => setTab('versions')}>
          <Package className="size-4" /> 版本库
        </TabButton>
        <TabButton active={tab === 'tasks'} onClick={() => setTab('tasks')}>
          <ListChecks className="size-4" /> 升级任务
        </TabButton>
      </div>

      {tab === 'versions' ? <VersionsView /> : <TasksView />}
    </PageShell>
  )
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`inline-flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-sm font-medium transition-colors ${
        active
          ? 'border-primary bg-primary/10 text-primary'
          : 'border-transparent text-muted-foreground hover:bg-muted'
      }`}
    >
      {children}
    </button>
  )
}

// ---------------------------------------------------------------------------
// Versions view (firmware library)
// ---------------------------------------------------------------------------

function VersionsView() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState<'' | VersionStatus>('')
  const [deviceType, setDeviceType] = useState('')

  const { data: productClasses } = useProductClasses()

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(status ? { status } : {}),
      ...(deviceType ? { deviceType } : {}),
    }),
    [page, pageSize, status, deviceType]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useSoftwareVersions(params)

  const allRows = data?.items ?? []
  // 客户端补充关键字过滤（后端按 deviceType/status/vendor 过滤；版本号/名称在前端兜底过滤）
  const kw = search.trim().toLowerCase()
  const rows = kw
    ? allRows.filter(
        (v) =>
          v.versionCode.toLowerCase().includes(kw) ||
          v.versionName.toLowerCase().includes(kw) ||
          (v.deviceType ?? '').toLowerCase().includes(kw)
      )
    : allRows
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const recommendCount = allRows.filter((v) => v.recommend).length
  const currentCount = allRows.filter((v) => v.status === 'current').length
  const totalSize = allRows.reduce((acc, v) => acc + (v.fileSize ?? 0), 0)

  const cols = [
    '版本号',
    '名称',
    '设备型号',
    '厂商',
    '类型',
    '状态',
    '推荐',
    '文件大小',
    '上传者',
    '发布时间',
  ]

  return (
    <div>
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="版本总数" value={total} icon={Package} />
        <Stat label="当前版本" value={currentCount} icon={CheckCircle2} tone="emerald" />
        <Stat label="推荐版本" value={recommendCount} icon={Star} tone="amber" />
        <Stat label="占用空间" value={formatBytes(totalSize)} icon={Boxes} tone="muted" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <Input
          className="w-72"
          placeholder="搜索版本号 / 名称 / 型号"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <Select
          value={deviceType || 'all'}
          onValueChange={(v) => {
            setDeviceType(v === 'all' ? '' : v)
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
            setStatus(v === 'all' ? '' : (v as VersionStatus))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="current">当前</SelectItem>
            <SelectItem value="beta">测试</SelectItem>
            <SelectItem value="deprecated">已废弃</SelectItem>
            <SelectItem value="archived">已归档</SelectItem>
          </SelectContent>
        </Select>
        <Button
          variant="outline"
          size="sm"
          className="ml-auto"
          onClick={() => refetch()}
        >
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
              <EmptyRow colSpan={cols.length}>暂无版本</EmptyRow>
            ) : (
              rows.map((v: SoftwareVersion) => {
                const st = VERSION_STATUS[v.status]
                return (
                  <TableRow key={v.id}>
                    <TableCell className="font-mono text-xs">{v.versionCode}</TableCell>
                    <TableCell className="max-w-[220px] truncate" title={v.versionName}>
                      {v.versionName || '—'}
                    </TableCell>
                    <TableCell>{v.deviceType || '—'}</TableCell>
                    <TableCell>{v.vendor || v.manufacturer || '—'}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {v.fileType != null ? FILE_TYPE_LABEL[v.fileType] ?? '—' : '—'}
                    </TableCell>
                    <TableCell>
                      <Badge variant={st.variant}>{st.label}</Badge>
                    </TableCell>
                    <TableCell>
                      {v.recommend ? (
                        <Badge variant="warning">
                          <Star className="mr-1 size-3" /> 推荐
                        </Badge>
                      ) : (
                        <span className="text-xs text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatBytes(v.fileSize)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {v.uploader || '—'}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(v.releaseDate)}
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
    </div>
  )
}

// ---------------------------------------------------------------------------
// Tasks view (upgrade task orchestration)
// ---------------------------------------------------------------------------

function TasksView() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [status, setStatus] = useState<'' | TaskStatusType>('')
  const [productClass, setProductClass] = useState('')
  const [detailTask, setDetailTask] = useState<UpgradeTaskInfo | null>(null)

  const { data: productClasses } = useProductClasses()

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(status ? { status } : {}),
      ...(productClass ? { productClass } : {}),
    }),
    [page, pageSize, status, productClass]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useUpgradeTasks(params)

  const suspend = useSuspendTask()
  const resume = useResumeTask()
  const terminate = useTerminateTask()
  const remove = useDeleteTask()
  const retry = useRetryTask()

  const busy =
    suspend.isPending ||
    resume.isPending ||
    terminate.isPending ||
    remove.isPending ||
    retry.isPending

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const running = rows.filter((t) => t.status === 'in_progress').length
  const suspended = rows.filter((t) => t.status === 'suspended').length
  const ended = rows.filter((t) => t.status === 'ended').length

  const cols = [
    '任务名称',
    '类型',
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
    <div>
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="任务总数" value={total} icon={ListChecks} />
        <Stat label="执行中" value={running} icon={Play} tone="default" />
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
        <Button
          variant="outline"
          size="sm"
          className="ml-auto"
          onClick={() => refetch()}
        >
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
              <EmptyRow colSpan={cols.length}>暂无升级任务</EmptyRow>
            ) : (
              rows.map((t: UpgradeTaskInfo) => {
                const st = TASK_STATUS[t.status]
                const code = mapStatusToCode(t.status)
                const showStart = code === 1 || code === 3 // pending / suspended
                const showPause = code === 2 // in_progress
                const showTerminate = code === 1 || code === 2 || code === 3
                const showRetry = code === 4 && t.failCount > 0 // ended with failures
                const showDelete = code !== 2
                return (
                  <TableRow key={t.id}>
                    <TableCell>
                      <button
                        type="button"
                        className="text-left font-medium text-primary hover:underline"
                        onClick={() => setDetailTask(t)}
                        title="查看子任务"
                      >
                        {t.taskName}
                      </button>
                    </TableCell>
                    <TableCell className="text-xs">
                      {TASK_TYPE[t.taskType] ?? `类型${t.taskType}`}
                    </TableCell>
                    <TableCell>{t.productClass || '—'}</TableCell>
                    <TableCell>
                      <Badge variant={st.variant}>{st.label}</Badge>
                    </TableCell>
                    <TableCell>
                      {t.result ? (
                        <Badge variant={TASK_RESULT[t.result].variant}>
                          {TASK_RESULT[t.result].label}
                        </Badge>
                      ) : (
                        <span className="text-xs text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell className="w-40">
                      <ProgressBar pct={progressPct(t)} />
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      <span className="text-emerald-600 dark:text-emerald-400">
                        {t.successCount}
                      </span>
                      {' / '}
                      <span className="text-destructive">{t.failCount}</span>
                      {' / '}
                      <span className="text-muted-foreground">{t.totalCount}</span>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {t.createUser || '—'}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(t.createdAt)}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        {showStart && (
                          <IconBtn
                            title="恢复/启动"
                            disabled={busy}
                            onClick={() => resume.mutate(t.id)}
                          >
                            <Play className="size-3.5" />
                          </IconBtn>
                        )}
                        {showPause && (
                          <IconBtn
                            title="暂停"
                            disabled={busy}
                            onClick={() => suspend.mutate(t.id)}
                          >
                            <Pause className="size-3.5" />
                          </IconBtn>
                        )}
                        {showRetry && (
                          <IconBtn
                            title="重试失败设备"
                            disabled={busy}
                            onClick={() => retry.mutate(t.id)}
                          >
                            <RotateCcw className="size-3.5" />
                          </IconBtn>
                        )}
                        {showTerminate && (
                          <IconBtn
                            title="终止"
                            disabled={busy}
                            onClick={() => terminate.mutate(t.id)}
                          >
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

      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={pageSize}
        onChange={setPage}
      />

      {detailTask && (
        <SubTaskDrawer task={detailTask} onClose={() => setDetailTask(null)} />
      )}
    </div>
  )
}

function mapStatusToCode(s: TaskStatusType): number {
  const map: Record<TaskStatusType, number> = {
    pending: 1,
    in_progress: 2,
    suspended: 3,
    ended: 4,
  }
  return map[s] ?? 1
}

function ProgressBar({ pct }: { pct: number }) {
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-24 overflow-hidden rounded-full bg-muted">
        <div
          className="h-full rounded-full bg-primary transition-all"
          style={{ width: `${Math.min(100, Math.max(0, pct))}%` }}
        />
      </div>
      <span className="tabular-nums text-xs text-muted-foreground">{pct}%</span>
    </div>
  )
}

function IconBtn({
  title,
  onClick,
  disabled,
  tone = 'default',
  children,
}: {
  title: string
  onClick: () => void
  disabled?: boolean
  tone?: 'default' | 'destructive'
  children: ReactNode
}) {
  return (
    <button
      type="button"
      title={title}
      disabled={disabled}
      onClick={onClick}
      className={`inline-flex items-center justify-center rounded-md border p-1.5 transition-colors disabled:cursor-not-allowed disabled:opacity-40 ${
        tone === 'destructive'
          ? 'text-destructive hover:bg-destructive/10'
          : 'text-muted-foreground hover:bg-muted hover:text-foreground'
      }`}
    >
      {children}
    </button>
  )
}

// ---------------------------------------------------------------------------
// Sub-task drawer (down-drill into a single upgrade task)
// ---------------------------------------------------------------------------

function SubTaskDrawer({
  task,
  onClose,
}: {
  task: UpgradeTaskInfo
  onClose: () => void
}) {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const { data, isLoading, isError, error, isFetching } = useSubTasks(task.id, {
    page,
    pageSize,
  })

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['设备SN', '原版本', '目标版本', '状态', '重试', '失败原因', '完成时间']

  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div
        className="absolute inset-0 bg-black/40"
        onClick={onClose}
        aria-hidden
      />
      <div className="relative flex h-full w-full max-w-3xl flex-col border-l bg-card shadow-xl">
        <div className="flex items-start justify-between border-b p-5">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <h2 className="truncate text-lg font-semibold">{task.taskName}</h2>
              <Badge variant={TASK_STATUS[task.status].variant}>
                {TASK_STATUS[task.status].label}
              </Badge>
              {isFetching && (
                <span className="text-xs text-muted-foreground">同步中…</span>
              )}
            </div>
            <p className="mt-1 text-xs text-muted-foreground">
              {TASK_TYPE[task.taskType] ?? `类型${task.taskType}`} · 型号{' '}
              {task.productClass || '—'} · 并发 {task.maxConcurrent} · 保留配置{' '}
              {task.isKeepConfig ? '是' : '否'}
            </p>
            <div className="mt-2 flex items-center gap-3 text-xs">
              <span className="text-emerald-600 dark:text-emerald-400">
                成功 {task.successCount}
              </span>
              <span className="text-destructive">失败 {task.failCount}</span>
              <span className="text-muted-foreground">总数 {task.totalCount}</span>
              <span className="text-muted-foreground">
                进度 {progressPct(task)}%
              </span>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-muted"
            title="关闭"
          >
            <X className="size-4" />
          </button>
        </div>

        <div className="flex-1 overflow-auto p-5">
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
                  <EmptyRow colSpan={cols.length}>暂无子任务</EmptyRow>
                ) : (
                  rows.map((s: UpgradeSubTaskInfo) => {
                    const st = SUB_TASK_STATUS[s.status]
                    return (
                      <TableRow key={s.id}>
                        <TableCell className="font-mono text-xs">
                          {s.deviceSn || s.deviceId}
                        </TableCell>
                        <TableCell className="text-xs">
                          {s.oriVersion || '—'}
                        </TableCell>
                        <TableCell className="text-xs">
                          {s.destVersion || '—'}
                        </TableCell>
                        <TableCell>
                          <Badge variant={st.variant}>{st.label}</Badge>
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {s.retryCount}/{s.maxRetries}
                        </TableCell>
                        <TableCell
                          className="max-w-[200px] truncate text-xs text-destructive"
                          title={s.failureReason || s.errorMessage || ''}
                        >
                          {s.failureReason || s.errorMessage || '—'}
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {formatTime(s.completedAt)}
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
        </div>
      </div>
    </div>
  )
}
