import { useMemo, useState } from 'react'
import {
  Activity,
  FileText,
  ListChecks,
  RefreshCcw,
  Search,
  StopCircle,
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
import {
  EmptyRow,
  ErrorRow,
  formatTime,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
} from '@/components/layout/PageShell'

import { useMRIndicators, useMRFileDevices, useBatchDeleteMRFiles } from '@core/hooks/api/useMR'
import {
  useMRTasks,
  useStopMRTask,
  useDeleteMRTask,
} from '@core/hooks/api/useMrTasks'
import type { MRTask, MRTaskStatus } from '@core/types/mrTask'

// ============================================================
// 测量报告（MR）— v2 适配 v1 业务深度
// 三个业务子页：测量任务 / 文件（按设备聚合）/ 指标库
//   - 任务：列表 + 状态/关键字筛选 + 停止/删除（条件化）+ 统计
//   - 文件：按设备聚合 + 关键字/基站/产品类筛选 + 多选批量删除
//   - 指标：MRO/MRS/MRE 指标定义（编码/名称/类别/单位/取值范围）
// 数据全走 @core React Query hooks，三态完整。
// ============================================================

type TabKey = 'tasks' | 'files' | 'indicators'

const PAGE_SIZE = 20

// ---------- MR 任务状态 → Badge ----------
const TASK_STATUS_VARIANT: Record<
  MRTaskStatus,
  'default' | 'secondary' | 'success' | 'warning' | 'muted'
> = {
  waitting: 'muted',
  on: 'default',
  off: 'success',
  suspend: 'warning',
  termination: 'warning',
}

const TASK_STATUS_LABEL: Record<MRTaskStatus, string> = {
  waitting: '等待中',
  on: '上报中',
  off: '已完成',
  suspend: '已挂起',
  termination: '已终止',
}

function TaskStatusBadge({ status }: { status: MRTaskStatus }) {
  return (
    <Badge variant={TASK_STATUS_VARIANT[status] ?? 'muted'}>
      {TASK_STATUS_LABEL[status] ?? status}
    </Badge>
  )
}

// 顶部统计卡片
function Stat({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: number
  tone?: 'default' | 'emerald' | 'amber' | 'muted'
}) {
  const toneClass = {
    default: 'text-foreground',
    emerald: 'text-emerald-600',
    amber: 'text-amber-600',
    muted: 'text-muted-foreground',
  }[tone]
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className={`mt-1 text-2xl font-semibold tabular-nums ${toneClass}`}>{value}</div>
    </div>
  )
}

export function MRPage() {
  const [tab, setTab] = useState<TabKey>('tasks')

  return (
    <PageShell
      title="测量报告（MR）"
      description="MRO/MRS/MRE 测量任务、文件采集与指标定义"
    >
      <div className="mb-4 inline-flex items-center gap-1 rounded-lg border bg-card p-1">
        <TabButton active={tab === 'tasks'} onClick={() => setTab('tasks')} icon={<ListChecks className="size-4" />}>
          测量任务
        </TabButton>
        <TabButton active={tab === 'files'} onClick={() => setTab('files')} icon={<FileText className="size-4" />}>
          采集文件
        </TabButton>
        <TabButton active={tab === 'indicators'} onClick={() => setTab('indicators')} icon={<Activity className="size-4" />}>
          指标库
        </TabButton>
      </div>

      {tab === 'tasks' && <TasksPanel />}
      {tab === 'files' && <FilesPanel />}
      {tab === 'indicators' && <IndicatorsPanel />}
    </PageShell>
  )
}

function TabButton({
  active,
  onClick,
  icon,
  children,
}: {
  active: boolean
  onClick: () => void
  icon: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={
        'inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors ' +
        (active
          ? 'bg-primary text-primary-foreground'
          : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground')
      }
    >
      {icon}
      {children}
    </button>
  )
}

// ============================================================
// 子页 1：测量任务
// ============================================================
type TaskStatusFilter = '' | MRTaskStatus

function TasksPanel() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState<TaskStatusFilter>('')

  const filter = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(status ? { status } : {}),
    }),
    [page, keyword, status],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useMRTasks(filter)
  const stopMutation = useStopMRTask()
  const deleteMutation = useDeleteMRTask()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const stats = useMemo(() => {
    let running = 0
    let waiting = 0
    let done = 0
    for (const t of rows) {
      if (t.taskStatus === 'on') running += 1
      else if (t.taskStatus === 'waitting') waiting += 1
      else if (t.taskStatus === 'off' || t.taskStatus === 'termination') done += 1
    }
    return { running, waiting, done }
  }, [rows])

  const handleStop = (t: MRTask) => {
    if (!window.confirm(`确认停止任务「${t.taskName}」？停止后将关闭所有 cell 的 MR 上报。`)) return
    stopMutation.mutate(t.taskId)
  }
  const handleDelete = (t: MRTask) => {
    if (!window.confirm(`确认删除任务「${t.taskName}」？删除后不可恢复。`)) return
    deleteMutation.mutate(t.taskId)
  }

  const cols = ['任务名称', '状态', '测量类型', '上报周期', '目标设备', '开始时间', '结束时间', '操作']

  return (
    <div>
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="总任务" value={total} />
        <Stat label="上报中" value={stats.running} tone="emerald" />
        <Stat label="等待中" value={stats.waiting} tone="amber" />
        <Stat label="已结束" value={stats.done} tone="muted" />
      </div>

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
            setStatus(v === 'all' ? '' : (v as MRTaskStatus))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="任务状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="waitting">等待中</SelectItem>
            <SelectItem value="on">上报中</SelectItem>
            <SelectItem value="off">已完成</SelectItem>
            <SelectItem value="termination">已终止</SelectItem>
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
              <EmptyRow colSpan={cols.length}>暂无测量任务</EmptyRow>
            ) : (
              rows.map((t) => {
                const stoppable = t.taskStatus === 'waitting' || t.taskStatus === 'on'
                const deletable = t.taskStatus === 'off' || t.taskStatus === 'termination'
                return (
                  <TableRow key={t.taskId}>
                    <TableCell className="font-medium">{t.taskName}</TableCell>
                    <TableCell>
                      <TaskStatusBadge status={t.taskStatus} />
                    </TableCell>
                    <TableCell className="text-xs">
                      {t.mrType
                        .split(',')
                        .map((s) => s.trim())
                        .join(' / ')}
                    </TableCell>
                    <TableCell className="tabular-nums text-xs">{t.reportPeriod} 分钟</TableCell>
                    <TableCell className="tabular-nums">{t.targetDeviceSns?.length ?? 0}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(t.startTime)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {t.endTime ? formatTime(t.endTime) : '不限'}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={!stoppable || stopMutation.isPending}
                          onClick={() => handleStop(t)}
                        >
                          <StopCircle className="size-4" /> 停止
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={!deletable || deleteMutation.isPending}
                          onClick={() => handleDelete(t)}
                        >
                          <Trash2 className="size-4 text-destructive" /> 删除
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
  )
}

// ============================================================
// 子页 2：采集文件（按设备聚合）
// ============================================================
function FilesPanel() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [siteName, setSiteName] = useState('')
  const [productClass, setProductClass] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(siteName.trim() ? { siteName: siteName.trim() } : {}),
      ...(productClass.trim() ? { productClass: productClass.trim() } : {}),
    }),
    [page, keyword, siteName, productClass],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useMRFileDevices(params)
  const batchDelete = useBatchDeleteMRFiles()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const toggle = (sn: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(sn)) next.delete(sn)
      else next.add(sn)
      return next
    })
  }
  const toggleAll = () => {
    setSelected((prev) => {
      if (prev.size === rows.length && rows.length > 0) return new Set()
      return new Set(rows.map((r) => r.deviceSn))
    })
  }

  const selectedList = Array.from(selected)

  const handleBatchDelete = () => {
    if (selectedList.length === 0) return
    if (!window.confirm(`确认删除选中 ${selectedList.length} 个设备的全部 MR 文件？删除后不可恢复。`)) return
    batchDelete.mutate(selectedList, {
      onSuccess: () => setSelected(new Set()),
    })
  }

  const allChecked = rows.length > 0 && selected.size === rows.length
  const cols = ['', '设备 SN', '基站名称', '产品类', '首次采集', '最近采集', '文件数', '上报状态']

  return (
    <div>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-44 pl-9"
            placeholder="搜索设备 SN"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <Input
          className="w-40"
          placeholder="基站名称"
          value={siteName}
          onChange={(e) => {
            setSiteName(e.target.value)
            setPage(1)
          }}
        />
        <Input
          className="w-40"
          placeholder="产品类"
          value={productClass}
          onChange={(e) => {
            setProductClass(e.target.value)
            setPage(1)
          }}
        />
        <Button
          variant="outline"
          size="sm"
          disabled={selectedList.length === 0 || batchDelete.isPending}
          onClick={handleBatchDelete}
        >
          <Trash2 className="size-4 text-destructive" /> 批量删除（{selectedList.length}）
        </Button>
        <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
          <RefreshCcw className={isFetching ? 'animate-spin' : ''} /> 刷新
        </Button>
      </div>

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c, idx) =>
                idx === 0 ? (
                  <TableHead key="sel" className="w-10">
                    <input
                      type="checkbox"
                      aria-label="全选"
                      checked={allChecked}
                      onChange={toggleAll}
                      className="size-4 cursor-pointer"
                    />
                  </TableHead>
                ) : (
                  <TableHead key={c}>{c}</TableHead>
                ),
              )}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无 MR 采集文件</EmptyRow>
            ) : (
              rows.map((d) => (
                <TableRow key={d.deviceSn}>
                  <TableCell className="w-10">
                    <input
                      type="checkbox"
                      aria-label={`选择 ${d.deviceSn}`}
                      checked={selected.has(d.deviceSn)}
                      onChange={() => toggle(d.deviceSn)}
                      className="size-4 cursor-pointer"
                    />
                  </TableCell>
                  <TableCell className="font-mono text-xs">{d.deviceSn}</TableCell>
                  <TableCell className="text-xs">{d.siteName || '—'}</TableCell>
                  <TableCell className="text-xs">{d.productClass || '—'}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(d.firstCollectTime)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(d.lastCollectTime)}
                  </TableCell>
                  <TableCell className="tabular-nums">{d.fileCount.toLocaleString()}</TableCell>
                  <TableCell>
                    {d.reporting ? (
                      <Badge variant="default">上报中</Badge>
                    ) : (
                      <Badge variant="muted">已停止</Badge>
                    )}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </div>
  )
}

// ============================================================
// 子页 3：指标库（MRO/MRS/MRE 指标定义）
// ============================================================
function IndicatorsPanel() {
  const [page, setPage] = useState(1)
  const params = useMemo(() => ({ page, pageSize: PAGE_SIZE }), [page])

  const { data, isLoading, isError, error, isFetching, refetch } = useMRIndicators(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['编码', '名称', '类别', '单位', '取值范围']

  return (
    <div>
      <div className="mb-3 flex items-center gap-2">
        <span className="text-sm text-muted-foreground">共 {total} 项指标</span>
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
              <EmptyRow colSpan={cols.length}>暂无 MR 指标</EmptyRow>
            ) : (
              rows.map((i) => (
                <TableRow key={i.id}>
                  <TableCell className="font-mono text-xs">{i.indicatorCode}</TableCell>
                  <TableCell className="font-medium">{i.indicatorName}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{i.category}</Badge>
                  </TableCell>
                  <TableCell className="text-xs">{i.unit || '—'}</TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    [{i.valueRange[0]}, {i.valueRange[1]}]
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </div>
  )
}
