import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeft, Loader2, X } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
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
  formatBytes,
} from '@/components/layout/PageShell'

import { useTraceTask, useTraceMessages } from '@core/hooks/api/useTrace'
import { traceApi } from '@core/services/api/traceApi'
import type { TraceMessage, TraceTaskStatus } from '@core/types/trace'

// ============================================================
// 报文跟踪详情 — /ops/message-trace/:id
// 真实数据：useTraceTask(id) + useTraceMessages(id)（5s 轮询）
// 选中报文 lazy fetch 外置 payload 并美化 XML 展示
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

const DIRECTION_LABEL: Record<TraceMessage['direction'], string> = {
  in: '上行 (CPE→ACS)',
  out: '下行 (ACS→CPE)',
}

const DIRECTION_VARIANT: Record<TraceMessage['direction'], 'success' | 'default'> = {
  in: 'success',
  out: 'default',
}

// 简易 XML 美化（保留 CDATA），与 v1 prettyXML 等价思路
function prettyXML(xml: string): string {
  if (!xml || !xml.trim()) return xml
  const cdata: string[] = []
  const safe = xml.replace(/<!\[CDATA\[[\s\S]*?\]\]>/g, (m) => {
    cdata.push(m)
    return `__CDATA_${cdata.length - 1}__`
  })
  const broken = safe.replace(/>\s*</g, '>\n<')
  let depth = 0
  const lines = broken.split('\n').map((raw) => {
    const line = raw.trim()
    if (!line) return ''
    const isClose = /^<\//.test(line)
    const isSelfClose = /\/>$/.test(line)
    const isDecl = /^<\?|^<!/.test(line)
    const isInline = /^<[^/!?][^>]*>[^<]*<\/[^>]+>$/.test(line)
    if (isClose) depth = Math.max(0, depth - 1)
    const indented = '  '.repeat(depth) + line
    if (!isClose && !isSelfClose && !isDecl && !isInline) depth++
    return indented
  })
  return lines.join('\n').replace(/__CDATA_(\d+)__/g, (_, i) => cdata[Number(i)])
}

function MessageDetailDrawer({ msg, onClose }: { msg: TraceMessage; onClose: () => void }) {
  // external 报文（inline 空 + 有 object_key）→ lazy fetch
  const isExternal = !msg.payloadInline && Boolean(msg.payloadObjectKey)
  const { data: fetched, isLoading, error } = useQuery({
    queryKey: ['trace', 'message-payload', msg.taskId, msg.id],
    queryFn: () => traceApi.getMessagePayload(msg.taskId, msg.id),
    enabled: isExternal,
  })
  const raw = msg.payloadInline || fetched || ''
  const payload = useMemo(() => prettyXML(raw), [raw])
  const errMsg = error instanceof Error ? error.message : null

  return (
    <div className="fixed inset-0 z-50 flex justify-end" role="dialog" aria-modal="true">
      <button type="button" aria-label="关闭" className="absolute inset-0 bg-black/30" onClick={onClose} />
      <div className="relative h-full w-full max-w-3xl overflow-y-auto border-l bg-card p-6 shadow-xl">
        <div className="mb-4 flex items-start justify-between gap-4">
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={DIRECTION_VARIANT[msg.direction]}>{DIRECTION_LABEL[msg.direction]}</Badge>
            {msg.rpcMethod ? <Badge variant="muted">{msg.rpcMethod}</Badge> : null}
            {msg.payloadObjectKey ? <Badge variant="warning">external</Badge> : null}
            <span className="text-xs text-muted-foreground">{formatTime(msg.capturedAt)}</span>
          </div>
          <Button variant="ghost" size="sm" onClick={onClose}>
            <X className="size-4" />
          </Button>
        </div>
        <pre className="max-h-[70vh] overflow-auto rounded-md border bg-muted/40 p-3 font-mono text-xs leading-relaxed whitespace-pre">
          {isLoading ? '加载报文中…' : errMsg ? errMsg : payload || '(empty)'}
        </pre>
      </div>
    </div>
  )
}

export default function MessageTraceDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [detailMsg, setDetailMsg] = useState<TraceMessage | null>(null)

  const { data: task, isLoading: taskLoading, isError: taskError, error: taskErr, isFetching } =
    useTraceTask(id)
  const {
    data: msgData,
    isLoading: msgLoading,
    isError: msgIsError,
    error: msgError,
  } = useTraceMessages(id, { page, pageSize })

  const msgRows: TraceMessage[] = msgData?.items ?? []
  const msgTotal = msgData?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(msgTotal / pageSize))

  const cols = ['抓取时间', '方向', 'RPC 方法', '大小', '操作']

  return (
    <PageShell
      title={task ? `报文跟踪 — ${task.deviceSn}` : '报文跟踪详情'}
      description={id ? `任务 ID ${id}` : undefined}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => navigate('/ops/message-trace')}>
            <ArrowLeft className="size-4" /> 返回跟踪列表
          </Button>
          {task && (
            <Badge variant={STATUS_VARIANT[task.status]}>{STATUS_LABEL[task.status]}</Badge>
          )}
        </div>
      }
    >
      {taskLoading ? (
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </div>
      ) : taskError ? (
        <Card className="flex h-48 items-center justify-center p-6 text-sm text-destructive">
          加载失败：{taskErr instanceof Error ? taskErr.message : '未知错误'}
        </Card>
      ) : !task ? (
        <Card className="flex h-48 flex-col items-center justify-center gap-2 p-6 text-muted-foreground">
          <span className="text-sm">未找到跟踪任务 {id}</span>
          <Button variant="outline" size="sm" onClick={() => navigate('/ops/message-trace')}>
            返回跟踪列表
          </Button>
        </Card>
      ) : (
        <div className="flex flex-col gap-4">
          <Card className="grid grid-cols-2 gap-x-8 gap-y-3 p-5 text-sm md:grid-cols-4">
            <div>
              <div className="text-xs uppercase tracking-wider text-muted-foreground">设备 SN</div>
              <div className="mt-1 font-mono text-xs">{task.deviceSn}</div>
            </div>
            <div>
              <div className="text-xs uppercase tracking-wider text-muted-foreground">报文数</div>
              <div className="mt-1 tabular-nums">{task.messageCount}</div>
            </div>
            <div>
              <div className="text-xs uppercase tracking-wider text-muted-foreground">开始时间</div>
              <div className="mt-1">{formatTime(task.startTime)}</div>
            </div>
            <div>
              <div className="text-xs uppercase tracking-wider text-muted-foreground">过期时间</div>
              <div className="mt-1">{formatTime(task.expiresAt)}</div>
            </div>
            <div>
              <div className="text-xs uppercase tracking-wider text-muted-foreground">创建人</div>
              <div className="mt-1">{task.createdBy}</div>
            </div>
            <div>
              <div className="text-xs uppercase tracking-wider text-muted-foreground">运营商</div>
              <div className="mt-1">{task.operatorCode || '—'}</div>
            </div>
            {task.stoppedAt ? (
              <div>
                <div className="text-xs uppercase tracking-wider text-muted-foreground">停止时间</div>
                <div className="mt-1">{formatTime(task.stoppedAt)}</div>
              </div>
            ) : null}
          </Card>

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
                {msgLoading ? (
                  <LoadingRow colSpan={cols.length} />
                ) : msgIsError ? (
                  <ErrorRow colSpan={cols.length} error={msgError} />
                ) : msgRows.length === 0 ? (
                  <EmptyRow colSpan={cols.length}>暂无报文</EmptyRow>
                ) : (
                  msgRows.map((m) => (
                    <TableRow key={m.id}>
                      <TableCell className="text-xs text-muted-foreground">
                        {formatTime(m.capturedAt)}
                      </TableCell>
                      <TableCell>
                        <Badge variant={DIRECTION_VARIANT[m.direction]}>
                          {DIRECTION_LABEL[m.direction]}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-xs">
                        {m.rpcMethod ? <Badge variant="muted">{m.rpcMethod}</Badge> : '—'}
                      </TableCell>
                      <TableCell className="text-xs tabular-nums">
                        {formatBytes(m.payloadSizeBytes)}
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="hover:underline"
                          onClick={() => setDetailMsg(m)}
                        >
                          报文详情
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableCard>

          <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={setPage} />
        </div>
      )}

      {detailMsg ? <MessageDetailDrawer msg={detailMsg} onClose={() => setDetailMsg(null)} /> : null}
    </PageShell>
  )
}
