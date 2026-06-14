import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeft, ArrowDownLeft, ArrowUpRight, Eye, Loader2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatBytes, formatTime } from '@/lib/format'
import { traceApi } from '@core/services/api/traceApi'
import type { TraceDirection, TraceMessage, TraceTaskStatus } from '@core/types/trace'
import { useTraceMessages, useTraceTask } from '@core/hooks/api/useTrace'

import { Center, IconBtn, Pager, StatCard, StateBlock } from './_shared'
import { Drawer, Field, SectionTitle } from './Modal'

const PAGE_SIZE = 20

const STATUS_COLOR: Record<TraceTaskStatus, string> = {
  running: '#00f0ff',
  stopped: '#525a78',
  purged: '#ffaa00',
}
const STATUS_LABEL: Record<TraceTaskStatus, string> = {
  running: '采集中',
  stopped: '已停止',
  purged: '已清除',
}
const DIRECTION_COLOR: Record<TraceDirection, string> = {
  in: '#00f0ff',
  out: '#5b9eff',
}
const DIRECTION_LABEL: Record<TraceDirection, string> = {
  in: '上行 IN',
  out: '下行 OUT',
}

// SOAP/XML 简单美化：按 tag 边界换行 + 缩进，CDATA 保护。
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

export default function MessageTraceDetail() {
  const { taskId } = useParams<{ taskId: string }>()
  const navigate = useNavigate()

  const { data: task, isLoading: taskLoading, isError: taskError } = useTraceTask(taskId)

  const [page, setPage] = useState(1)
  const { data: msgData, isLoading: msgLoading, isError: msgError } = useTraceMessages(taskId, {
    page,
    pageSize: PAGE_SIZE,
  })
  const messages = msgData?.items ?? []
  const total = msgData?.total ?? 0

  const [detailMsg, setDetailMsg] = useState<TraceMessage | null>(null)

  const subtitle = task ? `TASK ${task.id.slice(0, 8)} · ${task.deviceSn}` : 'TR-069 MESSAGE TRACE'

  return (
    <PageShell
      code="F01"
      title="OPS · 跟踪报文"
      subtitle={subtitle}
      isFetching={msgLoading}
      toolbar={
        <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/ops/message-trace')}>
          BACK
        </NeonButton>
      }
      bare
    >
      {taskLoading ? (
        <Center>
          <span className="flex items-center gap-2 font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/60">
            <Loader2 className="size-4 animate-spin" /> SYNCING…
          </span>
        </Center>
      ) : taskError || !task ? (
        <Center>
          <span className="border border-rose-500/40 bg-rose-500/5 px-4 py-3 font-mono text-sm text-rose-300">
            TASK NOT FOUND · 跟踪任务不存在或已被清除
          </span>
        </Center>
      ) : (
        <>
          <div className="mb-3 grid grid-cols-[repeat(4,1fr)] gap-3">
            <StatCard label="设备 · DEVICE" value={<span className="text-base">{task.deviceSn}</span>} color="#00f0ff" />
            <StatCard
              label="状态 · STATUS"
              value={<span className="text-base">{STATUS_LABEL[task.status]}</span>}
              color={STATUS_COLOR[task.status]}
            />
            <StatCard label="报文数 · MESSAGES" value={task.messageCount} color="#a855f7" />
            <StatCard label="发起人 · BY" value={<span className="text-base">{task.createdBy}</span>} color="#5b9eff" />
          </div>

          <div className="mb-3 grid grid-cols-2 gap-3 font-mono text-[11px] text-cyan-300/70 md:grid-cols-4">
            <InfoChip label="开始" value={formatTime(task.startTime)} />
            <InfoChip label="到期" value={formatTime(task.expiresAt)} />
            <InfoChip label="运营商" value={task.operatorCode || '—'} />
            <InfoChip label="停止于" value={task.stoppedAt ? formatTime(task.stoppedAt) : '—'} />
          </div>

          <StateBlock
            loading={msgLoading}
            error={msgError}
            empty={messages.length === 0}
            emptyText="NO MESSAGES · 暂无报文（采集中或未触发会话）"
          >
            <div className="overflow-hidden rounded-sm border border-cyan-500/12">
              <div className="grid grid-cols-[180px_110px_1.6fr_100px_60px] gap-3 border-b border-cyan-500/15 bg-cyan-500/[0.05] px-3 py-2 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/70">
                <span>时间 · CAPTURED</span>
                <span>方向</span>
                <span>方法 · RPC</span>
                <span>大小</span>
                <span className="text-right">操作</span>
              </div>
              {messages.map((m) => {
                const dc = DIRECTION_COLOR[m.direction]
                const DirIcon = m.direction === 'in' ? ArrowDownLeft : ArrowUpRight
                return (
                  <div
                    key={m.id}
                    className="fleet-row grid grid-cols-[180px_110px_1.6fr_100px_60px] items-center gap-3 px-3 py-2"
                    style={{ ['--row-color' as never]: dc }}
                  >
                    <span className="font-mono text-[11px] text-cyan-300/70">{formatTime(m.capturedAt)}</span>
                    <span>
                      <span className="chip" style={{ color: dc }}>
                        <DirIcon className="size-3" />
                        {DIRECTION_LABEL[m.direction]}
                      </span>
                    </span>
                    <span className="truncate font-mono text-[12px] text-cyan-100/85">
                      {m.rpcMethod || '—'}
                      {m.payloadObjectKey ? (
                        <span className="ml-2 chip text-[#ffaa00]">external</span>
                      ) : null}
                    </span>
                    <span className="font-mono text-[11px] text-cyan-300/75">{formatBytes(m.payloadSizeBytes)}</span>
                    <span className="text-right">
                      <IconBtn title="报文详情" color="#00f0ff" onClick={() => setDetailMsg(m)}>
                        <Eye className="size-3.5" />
                      </IconBtn>
                    </span>
                  </div>
                )
              })}
            </div>
            <Pager page={page} total={total} pageSize={PAGE_SIZE} onPage={setPage} />
          </StateBlock>
        </>
      )}

      <MessagePayloadDrawer msg={detailMsg} onClose={() => setDetailMsg(null)} />
    </PageShell>
  )
}

function InfoChip({ label, value }: { label: string; value: string }) {
  return (
    <div className="glass rounded-sm px-3 py-1.5">
      <span className="text-cyan-300/50">{label} · </span>
      <span className="text-cyan-100/85">{value}</span>
    </div>
  )
}

function MessagePayloadDrawer({ msg, onClose }: { msg: TraceMessage | null; onClose: () => void }) {
  const isExternal = Boolean(msg && !msg.payloadInline && msg.payloadObjectKey)
  const { data: fetched, isLoading, error } = useQuery({
    queryKey: ['trace', 'message-payload', msg?.taskId, msg?.id],
    queryFn: () => traceApi.getMessagePayload(msg!.taskId, msg!.id),
    enabled: Boolean(msg) && isExternal,
  })

  const raw = msg?.payloadInline || fetched || ''
  const payload = useMemo(() => prettyXML(raw), [raw])
  const errMsg = error instanceof Error ? error.message : null

  if (!msg) return null
  const dc = DIRECTION_COLOR[msg.direction]

  return (
    <Drawer
      open={Boolean(msg)}
      onClose={onClose}
      width={760}
      title={msg.rpcMethod || '(no rpc method)'}
      badge={
        <span className="chip shrink-0" style={{ color: dc }}>
          {DIRECTION_LABEL[msg.direction]}
        </span>
      }
    >
      <div className="space-y-4 p-4">
        <div className="glass rounded-sm">
          <SectionTitle>报文元信息 · META</SectionTitle>
          <Field label="捕获时间">{formatTime(msg.capturedAt)}</Field>
          <Field label="方向">{DIRECTION_LABEL[msg.direction]}</Field>
          <Field label="RPC 方法">{msg.rpcMethod || '—'}</Field>
          <Field label="CWMP ID">{msg.cwmpId || '—'}</Field>
          <Field label="Session ID">{msg.sessionId || '—'}</Field>
          {msg.httpStatus ? <Field label="HTTP 状态">{msg.httpStatus}</Field> : null}
          <Field label="大小">{formatBytes(msg.payloadSizeBytes)}</Field>
          {msg.payloadObjectKey ? (
            <Field label="外置存储">
              <span className="font-mono break-all text-[11px]">{msg.payloadObjectKey}</span>
            </Field>
          ) : null}
        </div>

        <div className="glass rounded-sm">
          <SectionTitle>报文体 · PAYLOAD</SectionTitle>
          <pre className="terminal max-h-[480px] overflow-auto p-3 text-[11px] text-cyan-200/85" style={{ whiteSpace: 'pre' }}>
            {isLoading ? '加载报文体…' : errMsg ? errMsg : payload || '(empty)'}
          </pre>
        </div>
      </div>
    </Drawer>
  )
}
