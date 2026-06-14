import { useMemo, useState } from 'react'
import {
  Activity,
  Gauge,
  Loader2,
  Radar,
  RefreshCcw,
  Route,
  Search,
  X,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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

import {
  useDiagnosticPing,
  useDiagnosticThroughput,
  useDiagnosticTraceroute,
  useOpsDiagnostics,
} from '@core/hooks/api/useOpsExt'
import type { DiagnosticStatus, OpsDiagnostic } from '@core/services/api/opsExtApi'

// ============================================================
// 网络诊断 — 对齐 v1 webcode/src/pages/ops/NetworkDiagnosis · v3 webcode-v3
// TR-181 Diagnostics（IPPing / TraceRoute / Throughput）
// 发起表单 + 历史记录列表 + 详情（请求/结果 JSON）
// ============================================================

const STATUS_LABEL: Record<DiagnosticStatus, string> = {
  pending: '排队',
  running: '执行中',
  complete: '完成',
  failed: '失败',
  timeout: '超时',
}

const STATUS_VARIANT: Record<
  DiagnosticStatus,
  'default' | 'warning' | 'success' | 'destructive' | 'muted'
> = {
  pending: 'muted',
  running: 'default',
  complete: 'success',
  failed: 'destructive',
  timeout: 'warning',
}

const DIAG_TYPE_OPTIONS: { value: string; label: string }[] = [
  { value: 'ping', label: 'IPPing' },
  { value: 'traceroute', label: 'TraceRoute' },
  { value: 'throughput', label: 'Throughput' },
]

const STATUS_OPTIONS: { value: DiagnosticStatus; label: string }[] = [
  { value: 'complete', label: '完成' },
  { value: 'running', label: '执行中' },
  { value: 'pending', label: '排队' },
  { value: 'failed', label: '失败' },
  { value: 'timeout', label: '超时' },
]

type DiagKind = 'ping' | 'traceroute' | 'throughput'

const PAGE_SIZE = 15

export default function NetworkDiagnosis() {
  const [page, setPage] = useState(1)
  const [deviceSn, setDeviceSn] = useState('')
  const [diagType, setDiagType] = useState('')
  const [status, setStatus] = useState<DiagnosticStatus | ''>('')

  const [detail, setDetail] = useState<OpsDiagnostic | null>(null)
  const [launchOpen, setLaunchOpen] = useState(false)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
      ...(diagType ? { diagType } : {}),
      ...(status ? { status } : {}),
    }),
    [page, deviceSn, diagType, status]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useOpsDiagnostics(params)
  const rows: OpsDiagnostic[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['类型', '设备 SN', '发起人', '开始时间', '耗时', '状态', '操作']

  return (
    <PageShell
      title="网络诊断"
      description="TR-181 Diagnostics 统一接入（IPPing / TraceRoute / Throughput）"
      isFetching={isFetching}
    >
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-56 pl-9"
            placeholder="设备 SN"
            value={deviceSn}
            onChange={(e) => {
              setDeviceSn(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <Select
          value={diagType || 'all'}
          onValueChange={(v) => {
            setDiagType(v === 'all' ? '' : v)
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="诊断类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部类型</SelectItem>
            {DIAG_TYPE_OPTIONS.map((d) => (
              <SelectItem key={d.value} value={d.value}>
                {d.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={status || 'all'}
          onValueChange={(v) => {
            setStatus(v === 'all' ? '' : (v as DiagnosticStatus))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-32">
            <SelectValue placeholder="状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            {STATUS_OPTIONS.map((s) => (
              <SelectItem key={s.value} value={s.value}>
                {s.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button size="sm" className="ml-auto" onClick={() => setLaunchOpen(true)}>
          <Radar className="size-4" /> 发起诊断
        </Button>
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          <RefreshCcw className={cn('size-4', isFetching && 'animate-spin')} /> 刷新
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
              <EmptyRow colSpan={cols.length}>暂无诊断记录</EmptyRow>
            ) : (
              rows.map((r) => (
                <TableRow key={r.id}>
                  <TableCell>
                    <span className="flex items-center gap-1.5 font-mono text-xs">
                      <Activity className="size-3.5 text-muted-foreground" />
                      {r.diag_type}
                    </span>
                  </TableCell>
                  <TableCell className="font-mono text-xs">{r.device_sn || '—'}</TableCell>
                  <TableCell className="text-xs">{r.initiator || r.operator || '—'}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(r.started_at)}
                  </TableCell>
                  <TableCell className="tabular-nums text-xs">
                    {r.duration_ms ? `${r.duration_ms} ms` : '—'}
                  </TableCell>
                  <TableCell>
                    <Badge variant={STATUS_VARIANT[r.status]}>{STATUS_LABEL[r.status]}</Badge>
                  </TableCell>
                  <TableCell>
                    <Button variant="ghost" size="sm" onClick={() => setDetail(r)} title="详情">
                      <Gauge className="size-4" /> 详情
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />

      <DiagnosticDetailDrawer diag={detail} open={detail !== null} onClose={() => setDetail(null)} />

      <LaunchDiagnosticDialog
        open={launchOpen}
        onClose={() => setLaunchOpen(false)}
        onDone={() => void refetch()}
      />
    </PageShell>
  )
}

function Field({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5 py-2">
      <span className="text-xs uppercase tracking-wider text-muted-foreground">{label}</span>
      <span className="break-words text-sm">{value || '—'}</span>
    </div>
  )
}

function safeJson(v: unknown): string {
  if (v == null) return '—'
  if (typeof v === 'string') return v
  try {
    return JSON.stringify(v, null, 2)
  } catch {
    return String(v)
  }
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
  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div className="absolute inset-0 bg-black/40" onClick={onClose} aria-hidden />
      <div className="relative flex h-full w-full max-w-xl flex-col border-l bg-background shadow-xl">
        <div className="flex items-center justify-between border-b px-5 py-4">
          <div className="flex items-center gap-2">
            <Badge variant={STATUS_VARIANT[diag.status]}>{STATUS_LABEL[diag.status]}</Badge>
            <h2 className="text-base font-semibold">
              {diag.diag_type} · {diag.device_sn || '—'}
            </h2>
          </div>
          <Button variant="ghost" size="icon" onClick={onClose} aria-label="关闭">
            <X />
          </Button>
        </div>
        <div className="flex-1 space-y-4 overflow-auto px-5 py-4">
          <div className="grid grid-cols-2 gap-x-6">
            <Field label="类型" value={diag.diag_type} />
            <Field label="设备 SN" value={<span className="font-mono">{diag.device_sn || '—'}</span>} />
            <Field label="发起人" value={diag.initiator || diag.operator} />
            <Field label="开始时间" value={formatTime(diag.started_at)} />
            {diag.completed_at ? (
              <Field label="完成时间" value={formatTime(diag.completed_at)} />
            ) : null}
            <Field label="耗时" value={diag.duration_ms ? `${diag.duration_ms} ms` : '—'} />
            {diag.error_message ? (
              <div className="col-span-2">
                <Field
                  label="错误"
                  value={<span className="text-destructive">{diag.error_message}</span>}
                />
              </div>
            ) : null}
          </div>

          <div>
            <div className="mb-1 text-xs uppercase tracking-wider text-muted-foreground">
              请求参数
            </div>
            <pre className="max-h-48 overflow-auto rounded-md border bg-muted/30 p-3 text-xs">
              {safeJson(diag.request)}
            </pre>
          </div>

          <div>
            <div className="mb-1 text-xs uppercase tracking-wider text-muted-foreground">
              诊断结果
            </div>
            <pre className="max-h-72 overflow-auto rounded-md border bg-muted/30 p-3 text-xs">
              {safeJson(diag.result)}
            </pre>
          </div>
        </div>
        <div className="flex justify-end gap-2 border-t px-5 py-3">
          <Button size="sm" onClick={onClose}>
            关闭
          </Button>
        </div>
      </div>
    </div>
  )
}

const KIND_META: { key: DiagKind; label: string; icon: typeof Radar }[] = [
  { key: 'ping', label: 'IPPing', icon: Activity },
  { key: 'traceroute', label: 'TraceRoute', icon: Route },
  { key: 'throughput', label: 'Throughput', icon: Gauge },
]

function LaunchDiagnosticDialog({
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
  const [errMsg, setErrMsg] = useState('')

  const ping = useDiagnosticPing()
  const traceroute = useDiagnosticTraceroute()
  const throughput = useDiagnosticThroughput()
  const pending = ping.isPending || traceroute.isPending || throughput.isPending

  if (!open) return null

  const reset = () => {
    setKind('ping')
    setSn('')
    setHost('')
    setCount('4')
    setUrl('')
    setDirection('download')
    setErrMsg('')
  }
  const close = () => {
    reset()
    onClose()
  }

  const targetMissing =
    kind === 'throughput' ? url.trim().length === 0 : host.trim().length === 0
  const canSubmit = sn.trim().length > 0 && !targetMissing && !pending

  const submit = () => {
    if (!canSubmit) return
    setErrMsg('')
    const done = {
      onSuccess: () => {
        reset()
        onClose()
        onDone()
      },
      onError: (e: unknown) =>
        setErrMsg(e instanceof Error ? e.message : '诊断发起失败，请检查设备在线状态后重试'),
    }
    if (kind === 'ping') {
      ping.mutate({ device_sn: sn.trim(), host: host.trim(), count: Number(count) || 4 }, done)
    } else if (kind === 'traceroute') {
      traceroute.mutate({ device_sn: sn.trim(), host: host.trim() }, done)
    } else {
      throughput.mutate({ device_sn: sn.trim(), url: url.trim(), direction }, done)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/40" onClick={close} aria-hidden />
      <div className="relative z-10 w-full max-w-md rounded-xl border bg-card shadow-lg">
        <div className="flex items-center justify-between border-b px-5 py-3">
          <h2 className="text-base font-semibold">发起网络诊断</h2>
          <Button variant="ghost" size="sm" onClick={close}>
            <X className="size-4" />
          </Button>
        </div>
        <div className="space-y-4 px-5 py-4">
          <div className="space-y-1.5">
            <Label>诊断类型</Label>
            <div className="flex flex-wrap gap-2">
              {KIND_META.map((m) => {
                const Icon = m.icon
                return (
                  <Button
                    key={m.key}
                    type="button"
                    size="sm"
                    variant={kind === m.key ? 'default' : 'outline'}
                    onClick={() => setKind(m.key)}
                  >
                    <Icon className="size-4" /> {m.label}
                  </Button>
                )
              })}
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="diag-sn">设备 SN</Label>
            <Input
              id="diag-sn"
              placeholder="ENB00001"
              value={sn}
              onChange={(e) => setSn(e.target.value)}
            />
          </div>
          {kind !== 'throughput' ? (
            <div className="space-y-1.5">
              <Label htmlFor="diag-host">目标主机 / IP</Label>
              <Input
                id="diag-host"
                placeholder="8.8.8.8 / example.com"
                value={host}
                onChange={(e) => setHost(e.target.value)}
              />
            </div>
          ) : null}
          {kind === 'ping' ? (
            <div className="space-y-1.5">
              <Label htmlFor="diag-count">探测次数</Label>
              <Input
                id="diag-count"
                type="number"
                min={1}
                max={20}
                className="w-32"
                value={count}
                onChange={(e) => setCount(e.target.value)}
              />
            </div>
          ) : null}
          {kind === 'throughput' ? (
            <>
              <div className="space-y-1.5">
                <Label htmlFor="diag-url">测速 URL</Label>
                <Input
                  id="diag-url"
                  placeholder="http://speedtest.example/test.bin"
                  value={url}
                  onChange={(e) => setUrl(e.target.value)}
                />
              </div>
              <div className="space-y-1.5">
                <Label>方向</Label>
                <div className="flex gap-2">
                  {(['download', 'upload'] as const).map((d) => (
                    <Button
                      key={d}
                      type="button"
                      size="sm"
                      variant={direction === d ? 'default' : 'outline'}
                      onClick={() => setDirection(d)}
                    >
                      {d === 'download' ? '下行' : '上行'}
                    </Button>
                  ))}
                </div>
              </div>
            </>
          ) : null}
          {errMsg ? (
            <p className="rounded-md bg-destructive/10 px-3 py-2 text-xs text-destructive">{errMsg}</p>
          ) : null}
        </div>
        <div className="flex justify-end gap-2 border-t px-5 py-3">
          <Button variant="outline" size="sm" onClick={close}>
            取消
          </Button>
          <Button size="sm" disabled={!canSubmit} onClick={submit}>
            {pending ? <Loader2 className="size-4 animate-spin" /> : <Radar className="size-4" />}
            发起诊断
          </Button>
        </div>
      </div>
    </div>
  )
}
