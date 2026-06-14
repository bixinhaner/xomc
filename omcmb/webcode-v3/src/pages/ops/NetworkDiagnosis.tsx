import { useMemo, useState } from 'react'
import { Activity, Gauge, Loader2, Radar, Route } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatTime } from '@/lib/format'
import type { DiagnosticStatus, OpsDiagnostic } from '@core/services/api/opsExtApi'
import {
  useDiagnosticPing,
  useDiagnosticThroughput,
  useDiagnosticTraceroute,
  useOpsDiagnostics,
} from '@core/hooks/api/useOpsExt'

import { FormRow, IconBtn, Pager, StatCard, StateBlock, Toolbar } from './_shared'
import { Drawer, Field, SectionTitle } from './Modal'

const PAGE_SIZE = 15

const STATUS_COLOR: Record<DiagnosticStatus, string> = {
  pending: '#5b9eff',
  running: '#00f0ff',
  complete: '#00ff88',
  failed: '#ff2d6f',
  timeout: '#ffaa00',
}
const STATUS_LABEL: Record<DiagnosticStatus, string> = {
  pending: '排队',
  running: '执行中',
  complete: '完成',
  failed: '失败',
  timeout: '超时',
}

const DIAG_TYPES: { value: string; label: string }[] = [
  { value: '', label: '全部类型' },
  { value: 'ping', label: 'IPPing' },
  { value: 'traceroute', label: 'TraceRoute' },
  { value: 'throughput', label: 'Throughput' },
]

type DiagKind = 'ping' | 'traceroute' | 'throughput'

export default function NetworkDiagnosis() {
  const [page, setPage] = useState(1)
  const [deviceSn, setDeviceSn] = useState('')
  const [diagType, setDiagType] = useState('')

  const [detail, setDetail] = useState<OpsDiagnostic | null>(null)
  const [launchOpen, setLaunchOpen] = useState(false)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
      ...(diagType ? { diagType } : {}),
    }),
    [page, deviceSn, diagType],
  )

  const { data, isLoading, isError, isFetching, refetch } = useOpsDiagnostics(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0

  const stat = useMemo(() => {
    const running = rows.filter((r) => r.status === 'running' || r.status === 'pending').length
    const ok = rows.filter((r) => r.status === 'complete').length
    const bad = rows.filter((r) => r.status === 'failed' || r.status === 'timeout').length
    return { running, ok, bad }
  }, [rows])

  return (
    <PageShell
      code="F06"
      title="OPS · 网络诊断"
      subtitle="NETWORK DIAGNOSTICS · TR-181 IPPING / TRACEROUTE / THROUGHPUT"
      isFetching={isFetching}
      bare
    >
      <div className="mb-3 grid grid-cols-3 gap-3">
        <StatCard label="进行中 · ACTIVE" value={stat.running} color="#00f0ff" />
        <StatCard label="完成 · DONE" value={stat.ok} color="#00ff88" />
        <StatCard label="异常 · FAIL/TIMEOUT" value={stat.bad} color={stat.bad > 0 ? '#ff2d6f' : '#525a78'} />
      </div>

      <Toolbar
        isFetching={isFetching}
        onRefresh={() => refetch()}
        extra={
          <NeonButton icon={<Radar />} onClick={() => setLaunchOpen(true)}>
            NEW DIAGNOSTIC
          </NeonButton>
        }
      >
        <input
          className="neon-input w-44"
          placeholder="设备 SN"
          value={deviceSn}
          onChange={(e) => {
            setDeviceSn(e.target.value)
            setPage(1)
          }}
        />
        {DIAG_TYPES.map((d) => (
          <button
            key={d.value || 'all'}
            type="button"
            onClick={() => {
              setDiagType(d.value)
              setPage(1)
            }}
            className={`chip text-[#00f0ff] ${diagType === d.value ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55'}`}
          >
            {d.label}
          </button>
        ))}
      </Toolbar>

      <StateBlock
        loading={isLoading}
        error={isError}
        empty={rows.length === 0}
        emptyText="NO DIAGNOSTICS · 暂无诊断记录"
      >
        <div className="overflow-hidden rounded-sm border border-cyan-500/12">
          <div className="grid grid-cols-[140px_1.4fr_1fr_100px_90px_60px] gap-3 border-b border-cyan-500/15 bg-cyan-500/[0.05] px-3 py-2 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/70">
            <span>类型 · TYPE</span>
            <span>设备 · DEVICE</span>
            <span>发起 · STARTED</span>
            <span>耗时</span>
            <span>状态</span>
            <span className="text-right">操作</span>
          </div>
          {rows.map((r) => {
            const c = STATUS_COLOR[r.status]
            return (
              <div
                key={r.id}
                className="fleet-row grid grid-cols-[140px_1.4fr_1fr_100px_90px_60px] items-center gap-3 px-3 py-2"
                style={{ ['--row-color' as never]: c }}
              >
                <span className="flex items-center gap-1.5 font-mono text-[11px] text-cyan-100/85">
                  <Activity className="size-3 text-cyan-300/60" />
                  {r.diag_type}
                </span>
                <span className="truncate font-mono text-[11px] text-cyan-300/75">
                  {r.device_sn || '—'}
                </span>
                <span className="font-mono text-[11px] text-cyan-300/70">{formatTime(r.started_at)}</span>
                <span className="font-mono text-[11px] text-cyan-300/75">
                  {r.duration_ms ? `${r.duration_ms}ms` : '—'}
                </span>
                <span>
                  <span className="chip" style={{ color: c }}>
                    {STATUS_LABEL[r.status]}
                  </span>
                </span>
                <span className="text-right">
                  <IconBtn title="详情" color="#00f0ff" onClick={() => setDetail(r)}>
                    <Gauge className="size-3.5" />
                  </IconBtn>
                </span>
              </div>
            )
          })}
        </div>
        <Pager page={page} total={total} pageSize={PAGE_SIZE} onPage={setPage} />
      </StateBlock>

      <DiagnosticDetailDrawer diag={detail} open={detail !== null} onClose={() => setDetail(null)} />

      <LaunchDiagnosticDrawer open={launchOpen} onClose={() => setLaunchOpen(false)} onDone={() => refetch()} />
    </PageShell>
  )
}

function DiagnosticDetailDrawer({
  diag,
  open,
  onClose,
}: {
  diag: OpsDiagnostic | null
  open: boolean
  onClose: () => void
}) {
  if (!open || !diag) return null
  const c = STATUS_COLOR[diag.status]
  const requestText = safeJson(diag.request)
  const resultText = safeJson(diag.result)
  return (
    <Drawer
      open={open}
      onClose={onClose}
      width={600}
      title={`${diag.diag_type} · ${diag.device_sn || '—'}`}
      badge={
        <span className="chip shrink-0" style={{ color: c }}>
          {STATUS_LABEL[diag.status]}
        </span>
      }
    >
      <div className="space-y-4 p-4">
        <div className="glass rounded-sm">
          <SectionTitle>诊断信息 · DIAGNOSTIC</SectionTitle>
          <Field label="类型">{diag.diag_type}</Field>
          <Field label="设备 SN">
            <span className="font-mono">{diag.device_sn || '—'}</span>
          </Field>
          <Field label="发起人">{diag.initiator || diag.operator || '—'}</Field>
          <Field label="开始时间">{formatTime(diag.started_at)}</Field>
          {diag.completed_at ? <Field label="完成时间">{formatTime(diag.completed_at)}</Field> : null}
          <Field label="耗时">{diag.duration_ms ? `${diag.duration_ms} ms` : '—'}</Field>
          {diag.error_message ? (
            <Field label="错误">
              <span className="text-rose-300">{diag.error_message}</span>
            </Field>
          ) : null}
        </div>

        <div className="glass rounded-sm">
          <SectionTitle>请求参数 · REQUEST</SectionTitle>
          <pre className="terminal max-h-48 overflow-auto p-3 text-[11px] text-cyan-200/85">
            {requestText}
          </pre>
        </div>

        <div className="glass rounded-sm">
          <SectionTitle>诊断结果 · RESULT</SectionTitle>
          <pre className="terminal max-h-72 overflow-auto p-3 text-[11px] text-emerald-300/85">
            {resultText}
          </pre>
        </div>
      </div>
    </Drawer>
  )
}

function LaunchDiagnosticDrawer({
  open,
  onClose,
  onDone,
}: {
  open: boolean
  onClose: () => void
  onDone: () => void
}) {
  const [kind, setKind] = useState<DiagKind>('ping')
  const [sn, setSn] = useState('')
  const [host, setHost] = useState('')
  const [count, setCount] = useState('4')
  const [url, setUrl] = useState('')
  const [direction, setDirection] = useState<'download' | 'upload'>('download')

  const ping = useDiagnosticPing()
  const traceroute = useDiagnosticTraceroute()
  const throughput = useDiagnosticThroughput()
  const pending = ping.isPending || traceroute.isPending || throughput.isPending
  const isError = ping.isError || traceroute.isError || throughput.isError

  const reset = () => {
    setKind('ping')
    setSn('')
    setHost('')
    setCount('4')
    setUrl('')
    setDirection('download')
  }

  const close = () => {
    reset()
    onClose()
  }

  const submit = () => {
    if (!sn.trim()) return
    const done = { onSuccess: () => { close(); onDone() } }
    if (kind === 'ping') {
      if (!host.trim()) return
      ping.mutate({ device_sn: sn.trim(), host: host.trim(), count: Number(count) || 4 }, done)
    } else if (kind === 'traceroute') {
      if (!host.trim()) return
      traceroute.mutate({ device_sn: sn.trim(), host: host.trim() }, done)
    } else {
      if (!url.trim()) return
      throughput.mutate({ device_sn: sn.trim(), url: url.trim(), direction }, done)
    }
  }

  if (!open) return null

  const KIND_META: { key: DiagKind; label: string; icon: typeof Radar }[] = [
    { key: 'ping', label: 'IPPing', icon: Activity },
    { key: 'traceroute', label: 'TraceRoute', icon: Route },
    { key: 'throughput', label: 'Throughput', icon: Gauge },
  ]

  return (
    <Drawer open={open} onClose={close} width={480} title="发起网络诊断" badge={<Radar className="size-4 text-cyan-300" />}>
      <div className="space-y-4 p-4">
        {isError ? (
          <div className="border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-[11px] text-rose-300">
            诊断发起失败，请检查设备在线状态后重试
          </div>
        ) : null}
        <FormRow label="诊断类型">
          <div className="flex flex-wrap gap-1.5">
            {KIND_META.map((m) => {
              const Icon = m.icon
              return (
                <button
                  key={m.key}
                  type="button"
                  onClick={() => setKind(m.key)}
                  className={`chip text-[#00f0ff] ${kind === m.key ? 'shadow-[0_0_8px_currentColor]' : 'opacity-55'}`}
                >
                  <Icon className="size-3" />
                  {m.label}
                </button>
              )
            })}
          </div>
        </FormRow>
        <FormRow label="设备 SN">
          <input
            className="neon-input w-full"
            placeholder="ENB00001"
            value={sn}
            onChange={(e) => setSn(e.target.value)}
          />
        </FormRow>
        {kind !== 'throughput' ? (
          <FormRow label="目标主机 / IP">
            <input
              className="neon-input w-full"
              placeholder="8.8.8.8 / example.com"
              value={host}
              onChange={(e) => setHost(e.target.value)}
            />
          </FormRow>
        ) : null}
        {kind === 'ping' ? (
          <FormRow label="探测次数">
            <input
              className="neon-input w-32"
              type="number"
              min={1}
              max={20}
              value={count}
              onChange={(e) => setCount(e.target.value)}
            />
          </FormRow>
        ) : null}
        {kind === 'throughput' ? (
          <>
            <FormRow label="测速 URL">
              <input
                className="neon-input w-full"
                placeholder="http://speedtest.example/test.bin"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
              />
            </FormRow>
            <FormRow label="方向">
              <div className="flex gap-1.5">
                {(['download', 'upload'] as const).map((d) => (
                  <button
                    key={d}
                    type="button"
                    onClick={() => setDirection(d)}
                    className={`chip text-[#5b9eff] ${direction === d ? 'shadow-[0_0_8px_currentColor]' : 'opacity-55'}`}
                  >
                    {d === 'download' ? '下行' : '上行'}
                  </button>
                ))}
              </div>
            </FormRow>
          </>
        ) : null}
        <div className="flex justify-end gap-2 pt-2">
          <NeonButton onClick={close}>取消</NeonButton>
          <NeonButton
            icon={pending ? <Loader2 className="animate-spin" /> : <Radar />}
            disabled={pending || !sn.trim()}
            onClick={submit}
          >
            发起诊断
          </NeonButton>
        </div>
      </div>
    </Drawer>
  )
}

function safeJson(v: unknown): string {
  if (v == null) return '// —'
  if (typeof v === 'string') return v
  try {
    return JSON.stringify(v, null, 2)
  } catch {
    return String(v)
  }
}
