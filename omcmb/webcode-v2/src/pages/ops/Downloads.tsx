import { useMemo, useState } from 'react'
import {
  Download,
  FileDown,
  Loader2,
  Package,
  RefreshCcw,
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
  formatBytes,
  formatTime,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useCollectDownload, useOpsDownloads } from '@core/hooks/api/useOpsExt'
import type { DownloadStatus, OpsDownload } from '@core/services/api/opsExtApi'

// ============================================================
// 运维下载 — 对齐 v1 webcode/src/pages/ops/Downloads · v3 webcode-v3
// 设备 → OMC 按需文件采集（config / log / pm / mr / diagnostic / pcap / gps）
// 列表 + 类型/状态筛选 + 详情 + 发起采集（useCollectDownload）
// ============================================================

const STATUS_LABEL: Record<DownloadStatus, string> = {
  pending: '排队',
  uploading: '采集中',
  complete: '完成',
  failed: '失败',
  expired: '已过期',
}

const STATUS_VARIANT: Record<
  DownloadStatus,
  'default' | 'warning' | 'success' | 'destructive' | 'muted'
> = {
  pending: 'muted',
  uploading: 'default',
  complete: 'success',
  failed: 'destructive',
  expired: 'warning',
}

const CONTENT_TYPE_OPTIONS: { value: string; label: string }[] = [
  { value: 'config', label: '配置' },
  { value: 'log', label: '日志' },
  { value: 'pm', label: 'PM' },
  { value: 'mr', label: 'MR' },
  { value: 'diagnostic', label: '诊断包' },
  { value: 'pcap', label: 'PCAP' },
]

const STATUS_OPTIONS: { value: DownloadStatus; label: string }[] = [
  { value: 'complete', label: '完成' },
  { value: 'uploading', label: '采集中' },
  { value: 'pending', label: '排队' },
  { value: 'failed', label: '失败' },
  { value: 'expired', label: '已过期' },
]

// 采集弹窗可勾选的内容类型（与后端 collect 入参 content_types 一致）
const COLLECT_TYPES = ['config', 'log', 'pm', 'mr', 'diagnostic', 'pcap', 'gps']

const PAGE_SIZE = 20

export default function Downloads() {
  const [page, setPage] = useState(1)
  const [deviceSn, setDeviceSn] = useState('')
  const [contentType, setContentType] = useState('')
  const [status, setStatus] = useState<DownloadStatus | ''>('')

  const [detail, setDetail] = useState<OpsDownload | null>(null)
  const [collectOpen, setCollectOpen] = useState(false)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
      ...(contentType ? { contentType } : {}),
      ...(status ? { status } : {}),
    }),
    [page, deviceSn, contentType, status]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useOpsDownloads(params)
  const rows: OpsDownload[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['设备 SN', '内容类型', '文件路径', '文件大小', '状态', '操作人', '创建时间', '操作']

  return (
    <PageShell
      title="运维下载"
      description="设备 → OMC 按需文件采集（配置 / 日志 / PM / MR / 诊断包 / PCAP / GPS）"
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
          value={contentType || 'all'}
          onValueChange={(v) => {
            setContentType(v === 'all' ? '' : v)
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="内容类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部类型</SelectItem>
            {CONTENT_TYPE_OPTIONS.map((c) => (
              <SelectItem key={c.value} value={c.value}>
                {c.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={status || 'all'}
          onValueChange={(v) => {
            setStatus(v === 'all' ? '' : (v as DownloadStatus))
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
        <Button size="sm" className="ml-auto" onClick={() => setCollectOpen(true)}>
          <FileDown className="size-4" /> 发起采集
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
              <EmptyRow colSpan={cols.length}>暂无采集任务</EmptyRow>
            ) : (
              rows.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="font-mono text-xs">
                    <span className="flex items-center gap-1.5">
                      <Package className="size-3.5 shrink-0 text-muted-foreground" />
                      {r.device_sn}
                    </span>
                  </TableCell>
                  <TableCell>
                    <Badge variant="secondary">{r.content_type}</Badge>
                  </TableCell>
                  <TableCell className="max-w-[240px] truncate font-mono text-xs text-muted-foreground">
                    {r.file_path || '—'}
                  </TableCell>
                  <TableCell className="tabular-nums">{formatBytes(r.file_size)}</TableCell>
                  <TableCell>
                    <Badge variant={STATUS_VARIANT[r.status]}>{STATUS_LABEL[r.status]}</Badge>
                  </TableCell>
                  <TableCell className="text-xs">{r.operator || '—'}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(r.created_at)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setDetail(r)}
                        title="详情"
                      >
                        详情
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        disabled={r.status !== 'complete' || !r.file_path}
                        title="下载"
                      >
                        <Download className="size-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />

      <DownloadDetailDrawer download={detail} open={detail !== null} onClose={() => setDetail(null)} />

      <CollectDialog
        open={collectOpen}
        onClose={() => setCollectOpen(false)}
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

function DownloadDetailDrawer({
  download,
  open,
  onClose,
}: {
  download: OpsDownload | null
  open: boolean
  onClose: () => void
}) {
  if (!open || !download) return null
  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div className="absolute inset-0 bg-black/40" onClick={onClose} aria-hidden />
      <div className="relative flex h-full w-full max-w-lg flex-col border-l bg-background shadow-xl">
        <div className="flex items-center justify-between border-b px-5 py-4">
          <div className="flex items-center gap-2">
            <Badge variant={STATUS_VARIANT[download.status]}>{STATUS_LABEL[download.status]}</Badge>
            <h2 className="text-base font-semibold">
              {download.content_type} · {download.device_sn}
            </h2>
          </div>
          <Button variant="ghost" size="icon" onClick={onClose} aria-label="关闭">
            <X />
          </Button>
        </div>
        <div className="flex-1 overflow-auto px-5 py-4">
          <div className="grid grid-cols-2 gap-x-6">
            <Field label="设备 SN" value={<span className="font-mono">{download.device_sn}</span>} />
            <Field label="内容类型" value={download.content_type} />
            <Field label="文件大小" value={formatBytes(download.file_size)} />
            <Field label="操作人" value={download.operator} />
            <Field label="创建时间" value={formatTime(download.created_at)} />
            {download.expires_at ? (
              <Field label="过期时间" value={formatTime(download.expires_at)} />
            ) : null}
            <div className="col-span-2">
              <Field
                label="文件路径"
                value={<span className="break-all font-mono text-xs">{download.file_path || '—'}</span>}
              />
            </div>
            {download.checksum ? (
              <div className="col-span-2">
                <Field
                  label="校验和"
                  value={<span className="break-all font-mono text-xs">{download.checksum}</span>}
                />
              </div>
            ) : null}
          </div>
        </div>
        <div className="flex justify-end gap-2 border-t px-5 py-3">
          <Button
            variant="outline"
            size="sm"
            disabled={download.status !== 'complete' || !download.file_path}
          >
            <Download className="size-4" /> 下载
          </Button>
          <Button size="sm" onClick={onClose}>
            关闭
          </Button>
        </div>
      </div>
    </div>
  )
}

function CollectDialog({
  open,
  onClose,
  onDone,
}: {
  open: boolean
  onClose: () => void
  onDone: () => void
}) {
  const [sn, setSn] = useState('')
  const [types, setTypes] = useState<string[]>(['config'])
  const [errMsg, setErrMsg] = useState('')
  const collect = useCollectDownload()

  if (!open) return null

  const reset = () => {
    setSn('')
    setTypes(['config'])
    setErrMsg('')
  }
  const close = () => {
    reset()
    onClose()
  }
  const toggle = (t: string) =>
    setTypes((prev) => (prev.includes(t) ? prev.filter((x) => x !== t) : [...prev, t]))

  const canSubmit = sn.trim().length > 0 && types.length > 0 && !collect.isPending

  const submit = () => {
    if (!canSubmit) return
    setErrMsg('')
    collect.mutate(
      { device_sn: sn.trim(), content_types: types },
      {
        onSuccess: () => {
          reset()
          onClose()
          onDone()
        },
        onError: (e) =>
          setErrMsg(e instanceof Error ? e.message : '采集发起失败，请检查设备在线状态后重试'),
      }
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/40" onClick={close} aria-hidden />
      <div className="relative z-10 w-full max-w-md rounded-xl border bg-card shadow-lg">
        <div className="flex items-center justify-between border-b px-5 py-3">
          <h2 className="text-base font-semibold">发起按需采集</h2>
          <Button variant="ghost" size="sm" onClick={close}>
            <X className="size-4" />
          </Button>
        </div>
        <div className="space-y-4 px-5 py-4">
          <div className="space-y-1.5">
            <Label htmlFor="collect-sn">设备 SN</Label>
            <Input
              id="collect-sn"
              placeholder="ENB00001"
              value={sn}
              onChange={(e) => setSn(e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <Label>采集内容（{types.length}）</Label>
            <div className="flex flex-wrap gap-2">
              {COLLECT_TYPES.map((t) => (
                <Button
                  key={t}
                  type="button"
                  size="sm"
                  variant={types.includes(t) ? 'default' : 'outline'}
                  onClick={() => toggle(t)}
                >
                  {t}
                </Button>
              ))}
            </div>
          </div>
          {errMsg ? (
            <p className="rounded-md bg-destructive/10 px-3 py-2 text-xs text-destructive">{errMsg}</p>
          ) : null}
        </div>
        <div className="flex justify-end gap-2 border-t px-5 py-3">
          <Button variant="outline" size="sm" onClick={close}>
            取消
          </Button>
          <Button size="sm" disabled={!canSubmit} onClick={submit}>
            {collect.isPending ? <Loader2 className="size-4 animate-spin" /> : <FileDown className="size-4" />}
            发起采集
          </Button>
        </div>
      </div>
    </div>
  )
}
