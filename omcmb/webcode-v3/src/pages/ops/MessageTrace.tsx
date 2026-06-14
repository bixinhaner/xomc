import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Download, Eye, Loader2, Plus, Radio, StopCircle, Trash2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatTime } from '@/lib/format'
import type { TraceTaskStatus } from '@core/types/trace'
import {
  useCreateTraceTask,
  useDeleteTraceTask,
  useRequestTraceExport,
  useStopTraceTask,
  useTraceExportJob,
  useTraceSseRefresh,
  useTraceTasks,
} from '@core/hooks/api/useTrace'

import { FormRow, IconBtn, Pager, StatCard, StateBlock, Toolbar } from './_shared'
import { Modal } from './Modal'

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
const STATUS_FILTERS: { value: '' | TraceTaskStatus; label: string }[] = [
  { value: '', label: 'ALL' },
  { value: 'running', label: '采集中' },
  { value: 'stopped', label: '已停止' },
  { value: 'purged', label: '已清除' },
]

export default function MessageTrace() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [deviceSn, setDeviceSn] = useState('')
  const [status, setStatus] = useState<'' | TraceTaskStatus>('')
  const [createOpen, setCreateOpen] = useState(false)
  const [exportJobId, setExportJobId] = useState<string | undefined>(undefined)

  // SSE 实时刷新 trace.task.*
  useTraceSseRefresh()

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
      ...(status ? { status } : {}),
    }),
    [page, deviceSn, status],
  )

  const { data, isLoading, isError, isFetching, refetch } = useTraceTasks(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0

  const stat = useMemo(() => {
    const running = rows.filter((r) => r.status === 'running').length
    const msgs = rows.reduce((sum, r) => sum + (r.messageCount || 0), 0)
    return { running, msgs }
  }, [rows])

  const stopMut = useStopTraceTask()
  const exportMut = useRequestTraceExport()
  const deleteMut = useDeleteTraceTask()
  const busy = stopMut.isPending || deleteMut.isPending

  return (
    <PageShell
      code="F01"
      title="OPS · 报文跟踪"
      subtitle="TR-069 MESSAGE TRACE · LIVE CWMP CAPTURE"
      isFetching={isFetching}
      bare
    >
      <div className="mb-3 grid grid-cols-3 gap-3">
        <StatCard label="本页任务 · PAGE" value={rows.length} color="#00f0ff" />
        <StatCard label="采集中 · LIVE" value={stat.running} color={stat.running > 0 ? '#00ff88' : '#525a78'} />
        <StatCard label="本页报文 · MESSAGES" value={stat.msgs} color="#a855f7" />
      </div>

      <Toolbar
        isFetching={isFetching}
        onRefresh={() => refetch()}
        extra={
          <NeonButton icon={<Plus />} onClick={() => setCreateOpen(true)}>
            NEW TRACE
          </NeonButton>
        }
      >
        <input
          className="neon-input w-48"
          placeholder="设备 SN"
          value={deviceSn}
          onChange={(e) => {
            setDeviceSn(e.target.value)
            setPage(1)
          }}
        />
        {STATUS_FILTERS.map((f) => (
          <button
            key={f.value || 'all'}
            type="button"
            onClick={() => {
              setStatus(f.value)
              setPage(1)
            }}
            className={`chip ${status === f.value ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55'}`}
            style={{ color: f.value ? STATUS_COLOR[f.value] : '#00f0ff' }}
          >
            {f.label}
          </button>
        ))}
      </Toolbar>

      <StateBlock
        loading={isLoading}
        error={isError}
        empty={rows.length === 0}
        emptyText="NO TRACE TASKS · 暂无跟踪任务"
      >
        <div className="overflow-hidden rounded-sm border border-cyan-500/12">
          <div className="grid grid-cols-[1.6fr_100px_1.2fr_1.2fr_90px_110px_150px] gap-3 border-b border-cyan-500/15 bg-cyan-500/[0.05] px-3 py-2 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/70">
            <span>设备 · DEVICE</span>
            <span>状态</span>
            <span>开始 · START</span>
            <span>到期 · EXPIRES</span>
            <span>报文数</span>
            <span>发起人</span>
            <span className="text-right">操作 · ACTIONS</span>
          </div>
          {rows.map((r) => {
            const c = STATUS_COLOR[r.status]
            return (
              <div
                key={r.id}
                className="fleet-row grid grid-cols-[1.6fr_100px_1.2fr_1.2fr_90px_110px_150px] items-center gap-3 px-3 py-2.5"
                style={{ ['--row-color' as never]: c }}
              >
                <span className="flex items-center gap-1.5 truncate font-mono text-[12px] text-cyan-100/90">
                  <Radio className={`size-3 shrink-0 ${r.status === 'running' ? 'animate-pulse text-[#00ff88]' : 'text-cyan-300/50'}`} />
                  {r.deviceSn}
                </span>
                <span>
                  <span className="chip" style={{ color: c }}>
                    {STATUS_LABEL[r.status]}
                  </span>
                </span>
                <span className="font-mono text-[11px] text-cyan-300/70">{formatTime(r.startTime)}</span>
                <span className="font-mono text-[11px] text-cyan-300/70">{formatTime(r.expiresAt)}</span>
                <span className="font-mono text-[12px] text-cyan-100/85">{r.messageCount}</span>
                <span className="truncate text-[11px] text-cyan-300/75">{r.createdBy}</span>
                <span className="flex items-center justify-end gap-1.5">
                  <IconBtn title="查看报文" color="#00f0ff" onClick={() => navigate(`/ops/message-trace/${r.id}`)}>
                    <Eye className="size-3.5" />
                  </IconBtn>
                  {r.status === 'running' ? (
                    <IconBtn
                      title="停止"
                      color="#ffaa00"
                      disabled={busy}
                      onClick={() => stopMut.mutate({ id: r.id, purge: false })}
                    >
                      <StopCircle className="size-3.5" />
                    </IconBtn>
                  ) : null}
                  {r.status !== 'purged' && r.messageCount > 0 ? (
                    <IconBtn
                      title="导出 XML"
                      color="#a855f7"
                      onClick={() =>
                        exportMut.mutate(r.id, { onSuccess: (job) => setExportJobId(job.id) })
                      }
                    >
                      <Download className="size-3.5" />
                    </IconBtn>
                  ) : null}
                  {r.status !== 'running' ? (
                    <IconBtn
                      title="删除"
                      color="#ff2d6f"
                      disabled={busy}
                      onClick={() => deleteMut.mutate(r.id)}
                    >
                      <Trash2 className="size-3.5" />
                    </IconBtn>
                  ) : null}
                </span>
              </div>
            )
          })}
        </div>
        <Pager page={page} total={total} pageSize={PAGE_SIZE} onPage={setPage} />
      </StateBlock>

      <CreateTraceModal open={createOpen} onClose={() => setCreateOpen(false)} onDone={() => refetch()} />

      <ExportProgressModal jobId={exportJobId} onClose={() => setExportJobId(undefined)} />
    </PageShell>
  )
}

function CreateTraceModal({
  open,
  onClose,
  onDone,
}: {
  open: boolean
  onClose: () => void
  onDone: () => void
}) {
  const [deviceSn, setDeviceSn] = useState('')
  const [duration, setDuration] = useState('5')
  const create = useCreateTraceTask()

  const reset = () => {
    setDeviceSn('')
    setDuration('5')
  }
  const close = () => {
    reset()
    onClose()
  }

  const submit = () => {
    if (!deviceSn.trim()) return
    create.mutate(
      { deviceSn: deviceSn.trim(), durationMinutes: Number(duration) || 5 },
      {
        onSuccess: () => {
          close()
          onDone()
        },
      },
    )
  }

  return (
    <Modal
      open={open}
      title="新建报文跟踪"
      subtitle="START TR-069 TRACE"
      width={460}
      onClose={close}
      footer={
        <>
          <NeonButton onClick={close}>取消</NeonButton>
          <NeonButton
            icon={create.isPending ? <Loader2 className="animate-spin" /> : <Plus />}
            disabled={create.isPending || !deviceSn.trim()}
            onClick={submit}
          >
            开始采集
          </NeonButton>
        </>
      }
    >
      <div className="space-y-4">
        {create.isError ? (
          <div className="border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-[11px] text-rose-300">
            创建失败，该设备可能已有进行中的跟踪任务
          </div>
        ) : null}
        <FormRow label="设备 SN">
          <input
            className="neon-input w-full"
            placeholder="输入设备 SN（支持粘贴完整 SN）"
            value={deviceSn}
            onChange={(e) => setDeviceSn(e.target.value)}
          />
        </FormRow>
        <FormRow label="持续时间（分钟，1-60）">
          <input
            className="neon-input w-40"
            type="number"
            min={1}
            max={60}
            value={duration}
            onChange={(e) => setDuration(e.target.value)}
          />
        </FormRow>
      </div>
    </Modal>
  )
}

// 异步导出进度：轮询 job，done 时自动触发浏览器下载预签名 URL
function ExportProgressModal({ jobId, onClose }: { jobId: string | undefined; onClose: () => void }) {
  const { data: job } = useTraceExportJob(jobId)

  const triggerDownload = (url: string) => {
    const a = document.createElement('a')
    a.href = url
    a.rel = 'noopener noreferrer'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  }

  return (
    <Modal open={Boolean(jobId)} title="导出报文 XML" subtitle="EXPORT TRACE" width={460} onClose={onClose}>
      <div className="space-y-3">
        {!job || job.status === 'queued' ? (
          <div className="flex items-center gap-2 font-mono text-[12px] text-cyan-300/70">
            <Loader2 className="size-4 animate-spin" /> 任务已入队，等待执行…
          </div>
        ) : null}
        {job?.status === 'running' ? (
          <div className="flex items-center gap-2 font-mono text-[12px] text-cyan-300/70">
            <Loader2 className="size-4 animate-spin" /> 打包导出中…
          </div>
        ) : null}
        {job?.status === 'done' ? (
          <div className="space-y-3">
            <div className="font-mono text-[12px] text-[#00ff88]">
              导出完成 · {job.messageCount} 条报文
            </div>
            {job.downloadUrl ? (
              <NeonButton icon={<Download />} onClick={() => triggerDownload(job.downloadUrl!)}>
                下载 XML
              </NeonButton>
            ) : null}
          </div>
        ) : null}
        {job?.status === 'failed' ? (
          <div className="border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-[11px] text-rose-300">
            导出失败：{job.errorMessage || '未知错误'}
          </div>
        ) : null}
      </div>
    </Modal>
  )
}
