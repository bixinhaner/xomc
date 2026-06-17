import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Eye, StopCircle, Trash2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
  useTraceTasks,
  useStopTraceTask,
  useDeleteTraceTask,
  useTraceSseRefresh,
} from '@core/hooks/api/useTrace'
import type { TraceTask, TraceTaskStatus } from '@core/types/trace'

// ============================================================
// TR069 报文跟踪 — 对齐 v1 webcode/src/pages/ops/MessageTrace
// 跟踪任务列表 + 停止/删除；点设备 SN 进 /ops/message-trace/:id 看报文
// ============================================================

const STATUS_LABEL: Record<TraceTaskStatus, string> = {
  running: '跟踪中',
  stopped: '已停止',
  purged: '已清除',
}

const STATUS_VARIANT: Record<TraceTaskStatus, 'default' | 'muted' | 'warning'> = {
  running: 'default',
  stopped: 'muted',
  purged: 'warning',
}

export default function MessageTrace() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)

  // SSE 实时刷新 trace.task.*（与 v1 一致）
  useTraceSseRefresh()

  const { data, isLoading, isError, error, isFetching, refetch } = useTraceTasks({
    page,
    pageSize,
  })

  const stopTask = useStopTraceTask()
  const deleteTask = useDeleteTraceTask()
  const busy = stopTask.isPending || deleteTask.isPending

  const rows: TraceTask[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['设备 SN', '状态', '开始时间', '过期时间', '报文数', '创建人', '操作']

  return (
    <PageShell title="TR069 报文跟踪" description="按设备开启 CWMP 报文抓取，实时查看南向交互报文">
      <div className="mb-3 flex items-center gap-2">
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
              <EmptyRow colSpan={cols.length}>暂无跟踪任务</EmptyRow>
            ) : (
              rows.map((r) => (
                <TableRow key={r.id}>
                  <TableCell>
                    <button
                      type="button"
                      className="text-left font-mono text-xs font-medium hover:underline"
                      onClick={() => navigate(`/ops/message-trace`)}
                    >
                      {r.deviceSn}
                    </button>
                  </TableCell>
                  <TableCell>
                    <Badge variant={STATUS_VARIANT[r.status]}>{STATUS_LABEL[r.status]}</Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(r.startTime)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(r.expiresAt)}
                  </TableCell>
                  <TableCell className="tabular-nums">{r.messageCount}</TableCell>
                  <TableCell className="text-xs">{r.createdBy}</TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => navigate(`/ops/message-trace`)}
                        title="查看报文"
                      >
                        <Eye className="size-4" />
                      </Button>
                      {r.status === 'running' && (
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={busy}
                          onClick={() => {
                            if (window.confirm(`确认停止设备「${r.deviceSn}」的跟踪？`)) {
                              stopTask.mutate({ id: r.id, purge: false })
                            }
                          }}
                          title="停止"
                        >
                          <StopCircle className="size-4" />
                        </Button>
                      )}
                      {r.status !== 'running' && (
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={busy}
                          className="text-destructive hover:text-destructive"
                          onClick={() => {
                            if (window.confirm(`确认删除设备「${r.deviceSn}」的跟踪任务？`)) {
                              deleteTask.mutate(r.id)
                            }
                          }}
                          title="删除"
                        >
                          <Trash2 className="size-4" />
                        </Button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={setPage} />
    </PageShell>
  )
}
