import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Loader2, Plus, RefreshCcw, RotateCw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
  useProvisioningTasks,
  useRetryProvisioningTask,
} from '@core/hooks/api/useProvisioning'
import {
  mapTaskToExecuteView,
  type ProvisioningTaskStatusCode,
} from '@core/services/api/provisionApi'

// ============================================================
// 即插即用（自动开站）— 自动发现/匹配/配置下发任务的执行状态列表
// 对照 v1 webcode/src/pages/device/PlugAndPlay：v1 含本地策略 CRUD（mock）+
// 真实即插即用执行状态。v2 聚焦真实 useProvisioningTasks 执行状态列表 +
// 失败重试；策略表单走 add/edit/view 子路由（AddPolicyPage）。
// ============================================================

const PAGE_SIZE = 20

const STATUS_META: Record<
  ProvisioningTaskStatusCode,
  { label: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'muted' }
> = {
  '0': { label: '成功', variant: 'success' },
  '1': { label: '失败', variant: 'destructive' },
  '2': { label: '执行中', variant: 'warning' },
  '3': { label: '未执行', variant: 'muted' },
  '4': { label: '跳过', variant: 'muted' },
}

type StatusFilter = 'all' | 'completed' | 'failed' | 'running'

export default function PlugAndPlay() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(statusFilter !== 'all' ? { status: statusFilter } : {}),
    }),
    [page, statusFilter]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useProvisioningTasks(params)
  const retry = useRetryProvisioningTask()

  const views = useMemo(
    () => (data?.items ?? []).map(mapTaskToExecuteView),
    [data]
  )
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const colCount = 8

  return (
    <PageShell
      title="即插即用"
      description={`自动开站执行任务 · 共 ${total} 条`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Select
            value={statusFilter}
            onValueChange={(v) => {
              setStatusFilter(v as StatusFilter)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="执行状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              <SelectItem value="completed">成功</SelectItem>
              <SelectItem value="failed">失败</SelectItem>
              <SelectItem value="running">执行中</SelectItem>
            </SelectContent>
          </Select>
          <div className="ml-auto flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw className="size-4" /> 刷新
            </Button>
            <Button
              size="sm"
              onClick={() => navigate('/devices/plug-and-play/add')}
            >
              <Plus className="size-4" /> 新增策略
            </Button>
          </div>
        </div>
      }
    >
      {retry.isError && (
        <div className="mb-3 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-2 text-sm text-destructive">
          重试失败：
          {retry.error instanceof Error ? retry.error.message : '未知错误'}
        </div>
      )}

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>任务 ID</TableHead>
              <TableHead>设备 ID</TableHead>
              <TableHead>执行状态</TableHead>
              <TableHead>进度</TableHead>
              <TableHead>重试</TableHead>
              <TableHead>开始时间</TableHead>
              <TableHead>结束时间</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={error} />
            ) : views.length === 0 ? (
              <EmptyRow colSpan={colCount}>暂无开站任务</EmptyRow>
            ) : (
              views.map((v) => {
                const meta = STATUS_META[v.status]
                return (
                  <TableRow key={v.taskId}>
                    <TableCell>
                      <button
                        type="button"
                        className="font-mono text-xs text-primary hover:underline"
                        onClick={() =>
                          navigate(`/devices/plug-and-play/view/${v.taskId}`)
                        }
                      >
                        {v.taskId}
                      </button>
                    </TableCell>
                    <TableCell>
                      <span className="font-mono text-xs text-muted-foreground">
                        {v.deviceId || '—'}
                      </span>
                    </TableCell>
                    <TableCell>
                      <Badge variant={meta.variant}>{meta.label}</Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {v.executeProcedure || '—'}
                    </TableCell>
                    <TableCell className="text-xs tabular-nums text-muted-foreground">
                      {v.retryCount}/{v.maxRetries}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(v.startTime)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(v.endTime)}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() =>
                            navigate(
                              `/devices/plug-and-play/view/${v.taskId}`
                            )
                          }
                        >
                          详情
                        </Button>
                        {v.status === '1' && (
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={retry.isPending}
                            onClick={() => retry.mutate(v.taskId)}
                          >
                            {retry.isPending ? (
                              <Loader2 className="size-3.5 animate-spin" />
                            ) : (
                              <RotateCw className="size-3.5" />
                            )}
                            重试
                          </Button>
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
        pageSize={PAGE_SIZE}
        onChange={setPage}
      />
    </PageShell>
  )
}
