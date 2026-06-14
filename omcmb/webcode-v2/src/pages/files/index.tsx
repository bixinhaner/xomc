import { useMemo, useState } from 'react'
import {
  Ban,
  Download,
  HardDrive,
  Pause,
  Play,
  RefreshCcw,
  RotateCcw,
  Search,
  Trash2,
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
import { cn } from '@/lib/utils'
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
  useDeleteFiles,
  useDownloadFile,
  useFileList,
  useStorageStats,
} from '@core/hooks/api/useFiles'
import {
  useDeleteUfteTask,
  useRetryUfteTask,
  useStartUfteTask,
  useSuspendUfteTask,
  useTerminateUfteTask,
  useUnifiedFileTransferOverview,
  useUnifiedFileTransferTasks,
} from '@core/hooks/api/useUnifiedFileTransfer'
import type { FileStatus, FileType } from '@core/mock/data/fileManagement'
import type {
  TransferTaskResult,
  TransferTaskStatus,
} from '@core/types/unifiedFileTransfer'

// ---------------------------------------------------------------------------
// Static label / variant maps（与 v1 file 模块语义对齐）
// ---------------------------------------------------------------------------

const FILE_TYPE_OPTIONS: { value: FileType; label: string }[] = [
  { value: 'config', label: '配置' },
  { value: 'log', label: '日志' },
  { value: 'firmware', label: '固件' },
  { value: 'backup', label: '备份' },
  { value: 'report', label: '报表' },
  { value: 'certificate', label: '证书' },
]

const FILE_TYPE_LABEL: Record<string, string> = Object.fromEntries(
  FILE_TYPE_OPTIONS.map((o) => [o.value, o.label])
)

const FILE_STATUS_OPTIONS: { value: FileStatus; label: string }[] = [
  { value: 'available', label: '可用' },
  { value: 'uploading', label: '上传中' },
  { value: 'processing', label: '处理中' },
  { value: 'expired', label: '已过期' },
  { value: 'deleted', label: '已删除' },
]

const FILE_STATUS_LABEL: Record<string, string> = Object.fromEntries(
  FILE_STATUS_OPTIONS.map((o) => [o.value, o.label])
)

function fileStatusVariant(
  s: FileStatus
): 'success' | 'warning' | 'muted' {
  if (s === 'available') return 'success'
  if (s === 'expired' || s === 'deleted') return 'muted'
  return 'warning'
}

const TASK_STATUS_OPTIONS: { value: TransferTaskStatus; label: string }[] = [
  { value: 'pending', label: '待执行' },
  { value: 'in_progress', label: '执行中' },
  { value: 'suspended', label: '已暂停' },
  { value: 'ended', label: '已结束' },
]

const TASK_STATUS_LABEL: Record<string, string> = Object.fromEntries(
  TASK_STATUS_OPTIONS.map((o) => [o.value, o.label])
)

function taskStatusVariant(
  s: TransferTaskStatus
): 'default' | 'warning' | 'muted' {
  if (s === 'in_progress') return 'default'
  if (s === 'suspended') return 'warning'
  return 'muted'
}

const TASK_RESULT_LABEL: Record<TransferTaskResult, string> = {
  success: '成功',
  partial: '部分成功',
  failure: '失败',
  terminated: '已终止',
}

function taskResultVariant(
  r: TransferTaskResult
): 'success' | 'warning' | 'destructive' | 'muted' {
  if (r === 'success') return 'success'
  if (r === 'partial') return 'warning'
  if (r === 'failure') return 'destructive'
  return 'muted'
}

type TabKey = 'library' | 'transfer'

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export function FilesPage() {
  const [tab, setTab] = useState<TabKey>('library')

  const stats = useStorageStats()
  const overview = useUnifiedFileTransferOverview()

  const isFetching = stats.isFetching || overview.isFetching

  return (
    <PageShell
      title="文件管理"
      description="文件库（PM / MR / 固件 / 备份 / 自定义）与统一文件传输任务"
      isFetching={isFetching}
    >
      {/* 概览统计卡 */}
      <div className="mb-5 grid grid-cols-2 gap-3 md:grid-cols-4 xl:grid-cols-6">
        <StatCard
          icon={<HardDrive className="size-4" />}
          label="文件总数"
          value={stats.data ? String(stats.data.fileCount) : '—'}
        />
        <StatCard
          label="已用容量"
          value={stats.data ? formatBytes(stats.data.usedSize) : '—'}
        />
        <StatCard
          label="启用传输类型"
          value={overview.data ? String(overview.data.enabledTypeCount) : '—'}
        />
        <StatCard
          label="运行中任务"
          value={overview.data ? String(overview.data.runningTaskCount) : '—'}
          tone={overview.data && overview.data.runningTaskCount > 0 ? 'emerald' : 'default'}
        />
        <StatCard
          label="自定义类型"
          value={overview.data ? String(overview.data.customTypeCount) : '—'}
        />
        <StatCard
          label="30天成功率"
          value={
            overview.data ? `${(overview.data.successRate30d * 100).toFixed(1)}%` : '—'
          }
          tone="emerald"
        />
      </div>

      {/* Tab 切换 */}
      <div className="mb-3 inline-flex rounded-lg border bg-card p-1">
        <TabButton active={tab === 'library'} onClick={() => setTab('library')}>
          文件库
        </TabButton>
        <TabButton active={tab === 'transfer'} onClick={() => setTab('transfer')}>
          统一文件传输任务
        </TabButton>
      </div>

      {tab === 'library' ? <FileLibraryTab /> : <TransferTaskTab />}
    </PageShell>
  )
}

// ---------------------------------------------------------------------------
// Tab: 文件库（真实 useFileList + useStorageStats + 下载/删除）
// ---------------------------------------------------------------------------

function FileLibraryTab() {
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [keyword, setKeyword] = useState('')
  const [fileType, setFileType] = useState<FileType | ''>('')
  const [status, setStatus] = useState<FileStatus | ''>('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(fileType ? { fileType } : {}),
      ...(status ? { status } : {}),
    }),
    [page, keyword, fileType, status]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useFileList(params)
  const download = useDownloadFile()
  const remove = useDeleteFiles()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['文件名', '类型', '状态', '大小', '设备', '上传时间', '上传者', '操作']

  return (
    <>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-72 pl-9"
            placeholder="搜索文件名"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value)
              setPage(1)
            }}
          />
        </div>

        <Select
          value={fileType || 'all'}
          onValueChange={(v) => {
            setFileType(v === 'all' ? '' : (v as FileType))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="文件类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部类型</SelectItem>
            {FILE_TYPE_OPTIONS.map((o) => (
              <SelectItem key={o.value} value={o.value}>
                {o.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select
          value={status || 'all'}
          onValueChange={(v) => {
            setStatus(v === 'all' ? '' : (v as FileStatus))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            {FILE_STATUS_OPTIONS.map((o) => (
              <SelectItem key={o.value} value={o.value}>
                {o.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <div className="ml-auto flex items-center gap-2">
          {isFetching && (
            <span className="text-xs text-muted-foreground">刷新中…</span>
          )}
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
              <EmptyRow colSpan={cols.length}>暂无文件</EmptyRow>
            ) : (
              rows.map((f) => (
                <TableRow key={f.id}>
                  <TableCell className="font-medium">{f.fileName}</TableCell>
                  <TableCell>
                    <Badge variant="outline">
                      {FILE_TYPE_LABEL[f.fileType] ?? f.fileType}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={fileStatusVariant(f.status)}>
                      {FILE_STATUS_LABEL[f.status] ?? f.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatBytes(f.fileSize)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {f.deviceName || f.deviceSn || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(f.uploadTime)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {f.uploader}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 px-2"
                        disabled={download.isPending}
                        onClick={() => download.mutate(f.id)}
                      >
                        <Download /> 下载
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 px-2 text-destructive hover:text-destructive"
                        disabled={remove.isPending}
                        onClick={() => {
                          if (window.confirm(`确认删除文件「${f.fileName}」？`)) {
                            remove.mutate([f.id])
                          }
                        }}
                      >
                        <Trash2 /> 删除
                      </Button>
                    </div>
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
    </>
  )
}

// ---------------------------------------------------------------------------
// Tab: 统一文件传输任务（真实 useUnifiedFileTransferTasks + 生命周期操作）
// ---------------------------------------------------------------------------

function TransferTaskTab() {
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState<TransferTaskStatus | ''>('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(status ? { status } : {}),
    }),
    [page, keyword, status]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useUnifiedFileTransferTasks(params)

  const start = useStartUfteTask()
  const suspend = useSuspendUfteTask()
  const terminate = useTerminateUfteTask()
  const retry = useRetryUfteTask()
  const remove = useDeleteUfteTask()

  const busy =
    start.isPending ||
    suspend.isPending ||
    terminate.isPending ||
    retry.isPending ||
    remove.isPending

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = [
    '任务名称',
    '类型',
    '状态',
    '结果',
    '进度',
    '成功/失败/总数',
    '创建人',
    '创建时间',
    '操作',
  ]

  return (
    <>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-72 pl-9"
            placeholder="搜索任务名称"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value)
              setPage(1)
            }}
          />
        </div>

        <Select
          value={status || 'all'}
          onValueChange={(v) => {
            setStatus(v === 'all' ? '' : (v as TransferTaskStatus))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="任务状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            {TASK_STATUS_OPTIONS.map((o) => (
              <SelectItem key={o.value} value={o.value}>
                {o.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <div className="ml-auto flex items-center gap-2">
          {isFetching && (
            <span className="text-xs text-muted-foreground">刷新中…</span>
          )}
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
              <EmptyRow colSpan={cols.length}>暂无传输任务</EmptyRow>
            ) : (
              rows.map((task) => (
                <TableRow key={task.id}>
                  <TableCell className="font-medium">
                    <div>{task.taskName}</div>
                    <div className="text-xs text-muted-foreground">
                      {task.categoryLabel}
                    </div>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {task.typeDisplayName}
                  </TableCell>
                  <TableCell>
                    <Badge variant={taskStatusVariant(task.status)}>
                      {TASK_STATUS_LABEL[task.status] ?? task.status}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    {task.result ? (
                      <Badge variant={taskResultVariant(task.result)}>
                        {TASK_RESULT_LABEL[task.result]}
                      </Badge>
                    ) : (
                      <span className="text-xs text-muted-foreground">—</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <ProgressBar value={task.progress} />
                  </TableCell>
                  <TableCell className="text-xs tabular-nums">
                    <span className="text-emerald-600 dark:text-emerald-400">
                      {task.successCount}
                    </span>
                    {' / '}
                    <span className="text-destructive">{task.failCount}</span>
                    {' / '}
                    <span className="text-muted-foreground">{task.totalCount}</span>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {task.createUser}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(task.createdAt)}
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-wrap items-center gap-1">
                      {task.status === 'pending' || task.status === 'suspended' ? (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 px-2"
                          disabled={busy}
                          onClick={() => start.mutate(task.id)}
                        >
                          <Play /> 启动
                        </Button>
                      ) : null}
                      {task.status === 'in_progress' ? (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 px-2"
                          disabled={busy}
                          onClick={() => suspend.mutate(task.id)}
                        >
                          <Pause /> 暂停
                        </Button>
                      ) : null}
                      {task.status === 'in_progress' ||
                      task.status === 'suspended' ? (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 px-2"
                          disabled={busy}
                          onClick={() => terminate.mutate(task.id)}
                        >
                          <Ban /> 终止
                        </Button>
                      ) : null}
                      {task.status === 'ended' &&
                      (task.result === 'failure' ||
                        task.result === 'partial') ? (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 px-2"
                          disabled={busy}
                          onClick={() => retry.mutate(task.id)}
                        >
                          <RotateCcw /> 重试
                        </Button>
                      ) : null}
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 px-2 text-destructive hover:text-destructive"
                        disabled={busy}
                        onClick={() => {
                          if (window.confirm(`确认删除任务「${task.taskName}」？`)) {
                            remove.mutate(task.id)
                          }
                        }}
                      >
                        <Trash2 /> 删除
                      </Button>
                    </div>
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
    </>
  )
}

// ---------------------------------------------------------------------------
// Local presentational helpers（本模块私有，未触碰共享组件）
// ---------------------------------------------------------------------------

function StatCard({
  icon,
  label,
  value,
  tone = 'default',
}: {
  icon?: React.ReactNode
  label: string
  value: string
  tone?: 'default' | 'emerald'
}) {
  const toneClass =
    tone === 'emerald'
      ? 'text-emerald-600 dark:text-emerald-400'
      : 'text-foreground'
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="flex items-center gap-1.5 text-xs uppercase tracking-wider text-muted-foreground">
        {icon}
        {label}
      </div>
      <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>
        {value}
      </div>
    </div>
  )
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
        active
          ? 'bg-primary text-primary-foreground'
          : 'text-muted-foreground hover:text-foreground'
      )}
    >
      {children}
    </button>
  )
}

function ProgressBar({ value }: { value: number }) {
  const pct = Math.max(0, Math.min(100, Math.round(value)))
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-20 overflow-hidden rounded-full bg-muted">
        <div
          className="h-full rounded-full bg-primary transition-all"
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="text-xs tabular-nums text-muted-foreground">{pct}%</span>
    </div>
  )
}
