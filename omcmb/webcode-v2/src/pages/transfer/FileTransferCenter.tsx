import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Search, X } from 'lucide-react'

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
import {
  useUnifiedFileTransferOverview,
  useUnifiedFileTransferTaskTypes,
  useUnifiedFileTransferTasks,
  useUnifiedFileTransferDevices,
} from '@core/hooks/api/useUnifiedFileTransfer'
import type { PageRequest } from '@core/types/pagination'
import {
  DEVICE_UPGRADE_CATEGORY,
  aggregateCategoryOptions,
  isDeviceUpgradeMember,
  resolveBackendCategoryParam,
} from '@core/utils/ufteCategory'

import {
  DeviceStatusBadge,
  EXEC_MODE_LABEL,
  ProgressBar,
  TaskResultBadge,
  TaskStatusBadge,
} from './_shared'

// ============================================================
// 文件传输中心 — 对照 v1 transfer/center
// 任务列表 / 设备列表 双视图 + 分类/类型/状态筛选 + 概览统计
// 任务行可点进任务详情 /transfer/center/task/:id
// ============================================================

type ViewMode = 'tasks' | 'devices'

const PAGE_SIZE = 10

export default function FileTransferCenter() {
  const navigate = useNavigate()

  const [view, setView] = useState<ViewMode>('tasks')
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [keywordInput, setKeywordInput] = useState('')
  const [category, setCategory] = useState<string>('all')
  const [typeCode, setTypeCode] = useState<string>('all')
  const [status, setStatus] = useState<string>('all')

  const overviewQuery = useUnifiedFileTransferOverview()
  const { data: taskTypes = [] } = useUnifiedFileTransferTaskTypes()

  // 分类下拉:从任务类型聚合(category -> categoryLabel),去重。
  // qa-614 c6 #368：4G(enb_upgrade)+5G(gnb_upgrade) 折叠为单条『设备升级』(device_upgrade)，
  // 与 v1/v3 口径一致（聚合逻辑共享自 @core/utils/ufteCategory）。
  const categoryOptions = useMemo(() => {
    return aggregateCategoryOptions(taskTypes).map((opt) =>
      opt.value === DEVICE_UPGRADE_CATEGORY ? { value: opt.value, label: '设备升级' } : opt,
    )
  }, [taskTypes])

  // 类型下拉随分类联动；device_upgrade 选中时取 enb_upgrade+gnb_upgrade 两类 typeCode。
  const typeOptions = useMemo(
    () =>
      taskTypes
        .filter((tt) => {
          if (category === 'all') return true
          if (category === DEVICE_UPGRADE_CATEGORY) return isDeviceUpgradeMember(tt.category)
          return tt.category === category
        })
        .map((tt) => ({ value: tt.typeCode, label: tt.displayName })),
    [taskTypes, category]
  )

  // qa-614 c6 #368：device_upgrade 虚拟分类展开为后端 category 查询参数
  // （有 typeCode 按 typeCode 精确过滤；无 typeCode 传成员超集 enb_upgrade 含 4G+5G）。
  const backendCategory =
    category === 'all'
      ? undefined
      : resolveBackendCategoryParam(category, typeCode !== 'all' ? typeCode : undefined)

  const sharedParams = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(backendCategory ? { category: backendCategory } : {}),
      ...(typeCode !== 'all' ? { typeCode } : {}),
      ...(status !== 'all' ? { status } : {}),
    }),
    [page, keyword, backendCategory, typeCode, status]
  ) satisfies { keyword?: string; category?: string; typeCode?: string; status?: string } & PageRequest

  const tasksQuery = useUnifiedFileTransferTasks(sharedParams)
  const devicesQuery = useUnifiedFileTransferDevices(sharedParams)

  const active = view === 'tasks' ? tasksQuery : devicesQuery
  const total = active.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const hasActiveFilter =
    Boolean(keyword.trim()) ||
    category !== 'all' ||
    typeCode !== 'all' ||
    status !== 'all'

  function applyKeyword() {
    setKeyword(keywordInput)
    setPage(1)
  }

  function resetFilters() {
    setKeyword('')
    setKeywordInput('')
    setCategory('all')
    setTypeCode('all')
    setStatus('all')
    setPage(1)
  }

  const overview = overviewQuery.data

  return (
    <PageShell
      title="文件传输中心"
      description={
        overview
          ? `启用类型 ${overview.enabledTypeCount} · 运行中任务 ${overview.runningTaskCount} · 自定义类型 ${overview.customTypeCount} · 近30天成功率 ${overview.successRate30d}%`
          : '统一文件传输任务总览'
      }
      isFetching={active.isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          {/* 视图切换 */}
          <div className="inline-flex rounded-md border p-0.5">
            <Button
              variant={view === 'tasks' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => {
                setView('tasks')
                setPage(1)
              }}
            >
              任务列表
            </Button>
            <Button
              variant={view === 'devices' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => {
                setView('devices')
                setPage(1)
              }}
            >
              设备列表
            </Button>
          </div>

          {/* 搜索 */}
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-64 pl-9"
              placeholder={view === 'tasks' ? '搜索任务名 / 类型' : '搜索 SN / 设备名'}
              value={keywordInput}
              onChange={(e) => setKeywordInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') applyKeyword()
              }}
            />
          </div>

          {/* 分类 */}
          <Select
            value={category}
            onValueChange={(v) => {
              setCategory(v)
              setTypeCode('all')
              setPage(1)
            }}
          >
            <SelectTrigger className="w-40">
              <SelectValue placeholder="业务分类" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部分类</SelectItem>
              {categoryOptions.map((o) => (
                <SelectItem key={o.value} value={o.value}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {/* 类型 */}
          <Select
            value={typeCode}
            onValueChange={(v) => {
              setTypeCode(v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-44">
              <SelectValue placeholder="任务类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部类型</SelectItem>
              {typeOptions.map((o) => (
                <SelectItem key={o.value} value={o.value}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {/* 状态 */}
          <Select
            value={status}
            onValueChange={(v) => {
              setStatus(v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              {view === 'tasks' ? (
                <>
                  <SelectItem value="pending">待执行</SelectItem>
                  <SelectItem value="in_progress">执行中</SelectItem>
                  <SelectItem value="suspended">已暂停</SelectItem>
                  <SelectItem value="ended">已结束</SelectItem>
                </>
              ) : (
                <>
                  <SelectItem value="pending">待执行</SelectItem>
                  <SelectItem value="downloading">下载中</SelectItem>
                  <SelectItem value="uploading">上传中</SelectItem>
                  <SelectItem value="verifying">校验中</SelectItem>
                  <SelectItem value="ended">已结束</SelectItem>
                  <SelectItem value="failed">失败</SelectItem>
                </>
              )}
            </SelectContent>
          </Select>

          {hasActiveFilter && (
            <Button variant="ghost" size="sm" onClick={resetFilters}>
              <X className="size-4" /> 重置
            </Button>
          )}

          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => active.refetch()}>
              <RefreshCcw className="size-4" /> 刷新
            </Button>
          </div>
        </div>
      }
    >
      {view === 'tasks' ? (
        <TaskTable
          isLoading={tasksQuery.isLoading}
          isError={tasksQuery.isError}
          error={tasksQuery.error}
          rows={tasksQuery.data?.items ?? []}
          hasActiveFilter={hasActiveFilter}
          onOpen={() => navigate(`/transfer/center`)}
        />
      ) : (
        <DeviceTable
          isLoading={devicesQuery.isLoading}
          isError={devicesQuery.isError}
          error={devicesQuery.error}
          rows={devicesQuery.data?.items ?? []}
          hasActiveFilter={hasActiveFilter}
          onOpenTask={() => navigate(`/transfer/center`)}
        />
      )}

      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={PAGE_SIZE}
        onChange={setPage}
      />
    </PageShell>
  )
}

function TaskTable({
  isLoading,
  isError,
  error,
  rows,
  hasActiveFilter,
  onOpen,
}: {
  isLoading: boolean
  isError: boolean
  error: unknown
  rows: import('@core/types/unifiedFileTransfer').UnifiedFileTransferTask[]
  hasActiveFilter: boolean
  onOpen: (id: string) => void
}) {
  const cols = 8
  return (
    <TableCard>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>任务名称</TableHead>
            <TableHead>业务分类</TableHead>
            <TableHead>类型</TableHead>
            <TableHead>状态</TableHead>
            <TableHead>结果</TableHead>
            <TableHead>进度 (成功/失败/总)</TableHead>
            <TableHead>执行方式</TableHead>
            <TableHead>创建时间</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <LoadingRow colSpan={cols} />
          ) : isError ? (
            <ErrorRow colSpan={cols} error={error} />
          ) : rows.length === 0 ? (
            <EmptyRow colSpan={cols}>
              {hasActiveFilter ? '没有匹配的任务' : '暂无任务'}
            </EmptyRow>
          ) : (
            rows.map((task) => (
              <TableRow key={task.id}>
                <TableCell>
                  <button
                    type="button"
                    className="max-w-[220px] truncate text-left text-sm font-medium text-primary hover:underline"
                    title={task.taskName}
                    onClick={() => onOpen(task.id)}
                  >
                    {task.taskName || task.id}
                  </button>
                </TableCell>
                <TableCell>
                  <span className="text-xs text-muted-foreground">
                    {task.categoryLabel || task.category || '—'}
                  </span>
                </TableCell>
                <TableCell>
                  <span className="text-xs">{task.typeDisplayName || task.typeCode}</span>
                </TableCell>
                <TableCell>
                  <TaskStatusBadge status={task.status} />
                </TableCell>
                <TableCell>
                  <TaskResultBadge result={task.result} />
                </TableCell>
                <TableCell>
                  <div className="flex flex-col gap-1">
                    <ProgressBar percent={task.progress} />
                    <span className="text-[11px] tabular-nums text-muted-foreground">
                      {task.successCount} / {task.failCount} / {task.totalCount}
                    </span>
                  </div>
                </TableCell>
                <TableCell>
                  <span className="text-xs text-muted-foreground">
                    {EXEC_MODE_LABEL[task.executionMode] ?? task.executionMode}
                  </span>
                </TableCell>
                <TableCell>
                  <span className="text-xs text-muted-foreground">
                    {formatTime(task.createdAt)}
                  </span>
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </TableCard>
  )
}

function DeviceTable({
  isLoading,
  isError,
  error,
  rows,
  hasActiveFilter,
  onOpenTask,
}: {
  isLoading: boolean
  isError: boolean
  error: unknown
  rows: import('@core/types/unifiedFileTransfer').UnifiedFileTransferDeviceItem[]
  hasActiveFilter: boolean
  onOpenTask: (id: string) => void
}) {
  const cols = 8
  return (
    <TableCard>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>设备 SN</TableHead>
            <TableHead>设备名称</TableHead>
            <TableHead>所属任务</TableHead>
            <TableHead>产品型号</TableHead>
            <TableHead>版本 (当前→目标)</TableHead>
            <TableHead>状态</TableHead>
            <TableHead>进度</TableHead>
            <TableHead>最近上报</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <LoadingRow colSpan={cols} />
          ) : isError ? (
            <ErrorRow colSpan={cols} error={error} />
          ) : rows.length === 0 ? (
            <EmptyRow colSpan={cols}>
              {hasActiveFilter ? '没有匹配的设备' : '暂无设备执行项'}
            </EmptyRow>
          ) : (
            rows.map((d) => (
              <TableRow key={d.id}>
                <TableCell>
                  <span className="font-mono text-xs">{d.deviceSn || '—'}</span>
                </TableCell>
                <TableCell>
                  <span className="text-sm">{d.deviceName || '—'}</span>
                </TableCell>
                <TableCell>
                  <button
                    type="button"
                    className="max-w-[180px] truncate text-left text-xs text-primary hover:underline"
                    title={d.taskName}
                    onClick={() => onOpenTask(d.taskId)}
                  >
                    {d.taskName || d.taskId}
                  </button>
                </TableCell>
                <TableCell>
                  <span className="text-xs text-muted-foreground">
                    {d.productType || '—'}
                  </span>
                </TableCell>
                <TableCell>
                  <span className="font-mono text-[11px] text-muted-foreground">
                    {(d.currentVersion || '—') +
                      (d.targetVersion ? ` → ${d.targetVersion}` : '')}
                  </span>
                </TableCell>
                <TableCell>
                  <DeviceStatusBadge status={d.status} />
                </TableCell>
                <TableCell>
                  <ProgressBar percent={d.progress} />
                </TableCell>
                <TableCell>
                  <span className="text-xs text-muted-foreground">
                    {formatTime(d.lastReportAt)}
                  </span>
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </TableCard>
  )
}
