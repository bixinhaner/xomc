import { useMemo, useState } from 'react'
import { RefreshCcw, Search } from 'lucide-react'

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
import { cn } from '@/lib/utils'

import { usePerformanceTasks } from '@core/hooks/api/usePerformance'
import type { PageRequest } from '@core/types/pagination'

// ============================================================
// 采集任务配置 — 对齐 v1 /performance/task-config
//   PM 采集任务列表（任务名/类型/设备数/指标数/粒度/状态/进度）。
// ============================================================

const PAGE_SIZE = 20

const TASK_STATUS_META: Record<
  string,
  { label: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'muted' }
> = {
  pending: { label: '等待中', variant: 'muted' },
  running: { label: '执行中', variant: 'warning' },
  success: { label: '成功', variant: 'success' },
  failed: { label: '失败', variant: 'destructive' },
  cancelled: { label: '已取消', variant: 'muted' },
}

type TaskStatusFilter = '' | 'pending' | 'running' | 'success' | 'failed' | 'cancelled'

function Stat({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: number | string
  tone?: 'default' | 'emerald' | 'amber' | 'muted'
}) {
  const toneClass = {
    default: 'text-foreground',
    emerald: 'text-emerald-600 dark:text-emerald-400',
    amber: 'text-amber-600 dark:text-amber-400',
    muted: 'text-muted-foreground',
  }[tone]
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>{value}</div>
    </div>
  )
}

export function PerformanceTaskConfigPage() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState<TaskStatusFilter>('')

  const params = useMemo<PageRequest>(() => ({ page, pageSize: PAGE_SIZE }), [page])
  const { data, isLoading, isError, error, isFetching, refetch } = usePerformanceTasks(params)

  const allRows = data?.items ?? []
  const rows = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    return allRows.filter((r) => {
      if (status && r.status !== status) return false
      if (kw && !r.taskName.toLowerCase().includes(kw)) return false
      return true
    })
  }, [allRows, keyword, status])

  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const runningCount = allRows.filter((r) => r.status === 'running').length
  const failedCount = allRows.filter((r) => r.status === 'failed').length

  const cols = ['任务名称', '类型', '设备数', '指标数', '粒度', '状态', '进度', '创建时间']

  return (
    <PageShell
      title="采集任务配置"
      description="PM 性能采集任务列表 · 状态与进度跟踪"
      isFetching={isFetching}
    >
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="任务总数" value={total} />
        <Stat label="执行中" value={runningCount} tone="amber" />
        <Stat label="失败" value={failedCount} tone={failedCount > 0 ? 'amber' : 'muted'} />
        <Stat label="当前页" value={`${page}/${totalPages}`} tone="muted" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-72 pl-9"
            placeholder="任务名称（本页过滤）"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
        <Select
          value={status || 'all'}
          onValueChange={(v) => setStatus(v === 'all' ? '' : (v as TaskStatusFilter))}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="pending">等待中</SelectItem>
            <SelectItem value="running">执行中</SelectItem>
            <SelectItem value="success">成功</SelectItem>
            <SelectItem value="failed">失败</SelectItem>
            <SelectItem value="cancelled">已取消</SelectItem>
          </SelectContent>
        </Select>
        <Button
          variant="outline"
          size="sm"
          className="ml-auto"
          disabled={isFetching}
          onClick={() => void refetch()}
        >
          <RefreshCcw className="size-4" /> 刷新
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
              <EmptyRow colSpan={cols.length}>暂无采集任务</EmptyRow>
            ) : (
              rows.map((t) => {
                const meta = TASK_STATUS_META[t.status] ?? {
                  label: t.status,
                  variant: 'outline' as const,
                }
                return (
                  <TableRow key={t.id}>
                    <TableCell className="font-medium">{t.taskName}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{t.taskType}</Badge>
                    </TableCell>
                    <TableCell className="tabular-nums">{t.deviceSns.length}</TableCell>
                    <TableCell className="tabular-nums">{t.kpiCodes.length}</TableCell>
                    <TableCell className="text-xs">{t.granularity || '—'}</TableCell>
                    <TableCell>
                      <Badge variant={meta.variant}>{meta.label}</Badge>
                    </TableCell>
                    <TableCell className="w-40">
                      <div className="flex items-center gap-2">
                        <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
                          <div
                            className="h-full rounded-full bg-primary transition-all"
                            style={{ width: `${Math.min(100, Math.max(0, t.progress))}%` }}
                          />
                        </div>
                        <span className="w-9 text-right text-xs tabular-nums text-muted-foreground">
                          {Math.round(t.progress)}%
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(t.createdAt)}
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </PageShell>
  )
}

export default PerformanceTaskConfigPage
