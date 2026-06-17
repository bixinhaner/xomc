import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent } from '@/components/ui/card'
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

import { usePerformanceTasks } from '@core/hooks/api/usePerformance'

// ============================================================
// 轮询统计 (report/poll-stats) — v2
// 业务对齐 v1 webcode/src/pages/report/PollStatistics：采集/轮询任务的执行统计。
// 真实数据：usePerformanceTasks() → /pm/tasks 的提取(采集)任务列表，含
//   taskName/taskType/granularity/status/progress/deviceSns/createdAt/updatedAt。
//   据此呈现采集任务执行情况（设备数、粒度、进度、状态），行点击进入 /reports/poll-stats/:id。
// ============================================================

// usePerformanceTasks 在 mock 与 real 两分支返回同字段集合（id/taskName/taskType/
// deviceSns/kpiCodes/granularity/timeRange/status/progress/creator/createdAt/updatedAt）。
// 仅消费公共字段，避免依赖分支特有的精确类型。
interface PollTaskView {
  id: string
  taskName: string
  taskType: string
  deviceSns: string[]
  granularity: string
  status: string
  progress: number
  updatedAt: string
}

const STATUS_LABEL: Record<string, string> = {
  pending: '等待',
  running: '运行中',
  success: '成功',
  failed: '失败',
  cancelled: '已取消',
}

const STATUS_VARIANT: Record<string, 'success' | 'warning' | 'destructive' | 'muted' | 'secondary'> = {
  pending: 'secondary',
  running: 'warning',
  success: 'success',
  failed: 'destructive',
  cancelled: 'muted',
}

const TASK_TYPE_LABEL: Record<string, string> = {
  extraction: '数据采集',
  report: '报表生成',
  'threshold-check': '门限检查',
}

const STATUS_OPTIONS = [
  { value: 'running', label: '运行中' },
  { value: 'pending', label: '等待' },
  { value: 'success', label: '成功' },
  { value: 'failed', label: '失败' },
  { value: 'cancelled', label: '已取消' },
]

const PAGE_SIZE = 20

export default function PollStatistics() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState('')

  const tasksQuery = usePerformanceTasks({ page, pageSize: PAGE_SIZE })

  const tasks: PollTaskView[] = useMemo(
    () =>
      (tasksQuery.data?.items ?? []).map((t) => ({
        id: t.id,
        taskName: t.taskName,
        taskType: t.taskType,
        deviceSns: t.deviceSns ?? [],
        granularity: t.granularity,
        status: t.status,
        progress: t.progress,
        updatedAt: t.updatedAt,
      })),
    [tasksQuery.data],
  )

  const total = tasksQuery.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const filtered = tasks.filter((t) => {
    if (keyword && !t.taskName.toLowerCase().includes(keyword.toLowerCase())) return false
    if (statusFilter && t.status !== statusFilter) return false
    return true
  })

  const runningCount = tasks.filter((t) => t.status === 'running').length
  const failedCount = tasks.filter((t) => t.status === 'failed').length

  return (
    <PageShell
      title="轮询统计"
      description={`采集/轮询任务 · 共 ${total} 个`}
      isFetching={tasksQuery.isFetching}
      toolbar={
        <>
          <Input
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder="搜索任务名"
            className="w-44"
          />
          <Select
            value={statusFilter || 'all'}
            onValueChange={(v) => setStatusFilter(v === 'all' ? '' : v)}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              {STATUS_OPTIONS.map((o) => (
                <SelectItem key={o.value} value={o.value}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={() => void tasksQuery.refetch()}
          >
            <RefreshCcw /> 刷新
          </Button>
        </>
      }
    >
      {/* 概览 */}
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Card>
          <CardContent className="p-4">
            <div className="text-xs uppercase tracking-wider text-muted-foreground">
              任务总数
            </div>
            <div className="mt-1 text-2xl font-semibold tabular-nums">{total}</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="text-xs uppercase tracking-wider text-muted-foreground">
              运行中
            </div>
            <div className="mt-1 text-2xl font-semibold tabular-nums text-amber-600 dark:text-amber-400">
              {runningCount}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="text-xs uppercase tracking-wider text-muted-foreground">
              失败
            </div>
            <div className="mt-1 text-2xl font-semibold tabular-nums text-rose-600 dark:text-rose-400">
              {failedCount}
            </div>
          </CardContent>
        </Card>
      </div>

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>任务名称</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>设备数</TableHead>
              <TableHead>采集粒度</TableHead>
              <TableHead>进度</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>最近执行</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {tasksQuery.isLoading ? (
              <LoadingRow colSpan={8} />
            ) : tasksQuery.isError ? (
              <ErrorRow colSpan={8} error={tasksQuery.error} />
            ) : filtered.length === 0 ? (
              <EmptyRow colSpan={8}>暂无采集任务</EmptyRow>
            ) : (
              filtered.map((t) => (
                <TableRow
                  key={t.id}
                  className="cursor-pointer"
                  onClick={() => navigate(`/report/poll-stats`)}
                >
                  <TableCell className="font-medium text-primary hover:underline">
                    {t.taskName}
                  </TableCell>
                  <TableCell className="text-xs">
                    {TASK_TYPE_LABEL[t.taskType] ?? t.taskType}
                  </TableCell>
                  <TableCell className="tabular-nums">{t.deviceSns.length}</TableCell>
                  <TableCell className="text-xs">{t.granularity}</TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <div className="h-1.5 w-24 overflow-hidden rounded-full bg-muted">
                        <div
                          className={cn(
                            'h-full rounded-full',
                            t.status === 'failed' ? 'bg-rose-500' : 'bg-emerald-500',
                          )}
                          style={{ width: `${Math.min(100, Math.max(0, t.progress))}%` }}
                        />
                      </div>
                      <span className="text-xs tabular-nums text-muted-foreground">
                        {Math.round(t.progress)}%
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={STATUS_VARIANT[t.status] ?? 'muted'}>
                      {STATUS_LABEL[t.status] ?? t.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(t.updatedAt)}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation()
                        navigate(`/report/poll-stats`)
                      }}
                    >
                      查看
                    </Button>
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
        pageSize={PAGE_SIZE}
        onChange={setPage}
      />
    </PageShell>
  )
}
