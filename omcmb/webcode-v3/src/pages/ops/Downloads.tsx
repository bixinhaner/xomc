import { useMemo, useState } from 'react'
import { Download, FileDown, Loader2, Package } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatBytes, formatTime } from '@/lib/format'
import type { DownloadStatus, OpsDownload } from '@core/services/api/opsExtApi'
import { useCollectDownload, useOpsDownloads } from '@core/hooks/api/useOpsExt'

import { FormRow, IconBtn, Pager, StatCard, StateBlock, Toolbar } from './_shared'
import { Drawer, Field, SectionTitle } from './Modal'

const PAGE_SIZE = 15

const STATUS_COLOR: Record<DownloadStatus, string> = {
  pending: '#5b9eff',
  uploading: '#00f0ff',
  complete: '#00ff88',
  failed: '#ff2d6f',
  expired: '#525a78',
}
const STATUS_LABEL: Record<DownloadStatus, string> = {
  pending: '排队',
  uploading: '采集中',
  complete: '完成',
  failed: '失败',
  expired: '已过期',
}

const CONTENT_TYPES: { value: string; label: string }[] = [
  { value: '', label: '全部' },
  { value: 'config', label: '配置' },
  { value: 'log', label: '日志' },
  { value: 'pm', label: 'PM' },
  { value: 'mr', label: 'MR' },
  { value: 'diagnostic', label: '诊断包' },
  { value: 'pcap', label: 'PCAP' },
]

// 采集弹窗可勾选的内容类型（与后端 content_types 入参一致）
const COLLECT_TYPES = ['config', 'log', 'pm', 'mr', 'diagnostic', 'pcap', 'gps']

export default function Downloads() {
  const [page, setPage] = useState(1)
  const [deviceSn, setDeviceSn] = useState('')
  const [contentType, setContentType] = useState('')

  const [detail, setDetail] = useState<OpsDownload | null>(null)
  const [collectOpen, setCollectOpen] = useState(false)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
      ...(contentType ? { contentType } : {}),
    }),
    [page, deviceSn, contentType],
  )

  const { data, isLoading, isError, isFetching, refetch } = useOpsDownloads(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0

  const stat = useMemo(() => {
    const collecting = rows.filter((r) => r.status === 'pending' || r.status === 'uploading').length
    const ready = rows.filter((r) => r.status === 'complete').length
    const totalBytes = rows.reduce((sum, r) => sum + (r.file_size || 0), 0)
    return { collecting, ready, totalBytes }
  }, [rows])

  return (
    <PageShell
      code="F06"
      title="OPS · 运维下载"
      subtitle="ON-DEMAND COLLECTION · CONFIG / LOG / PM / MR / PCAP"
      isFetching={isFetching}
      bare
    >
      <div className="mb-3 grid grid-cols-3 gap-3">
        <StatCard label="采集中 · COLLECTING" value={stat.collecting} color="#00f0ff" />
        <StatCard label="可下载 · READY" value={stat.ready} color="#00ff88" />
        <StatCard label="本页总量 · SIZE" value={formatBytes(stat.totalBytes)} color="#a855f7" />
      </div>

      <Toolbar
        isFetching={isFetching}
        onRefresh={() => refetch()}
        extra={
          <NeonButton icon={<FileDown />} onClick={() => setCollectOpen(true)}>
            COLLECT
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
        {CONTENT_TYPES.map((c) => (
          <button
            key={c.value || 'all'}
            type="button"
            onClick={() => {
              setContentType(c.value)
              setPage(1)
            }}
            className={`chip text-[#00f0ff] ${contentType === c.value ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55'}`}
          >
            {c.label}
          </button>
        ))}
      </Toolbar>

      <StateBlock
        loading={isLoading}
        error={isError}
        empty={rows.length === 0}
        emptyText="NO DOWNLOADS · 暂无采集任务"
      >
        <div className="overflow-hidden rounded-sm border border-cyan-500/12">
          <div className="grid grid-cols-[1.4fr_120px_1.6fr_100px_90px_60px] gap-3 border-b border-cyan-500/15 bg-cyan-500/[0.05] px-3 py-2 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/70">
            <span>设备 · DEVICE</span>
            <span>类型 · TYPE</span>
            <span>文件 · FILE</span>
            <span>大小</span>
            <span>状态</span>
            <span className="text-right">操作</span>
          </div>
          {rows.map((r) => {
            const c = STATUS_COLOR[r.status]
            return (
              <div
                key={r.id}
                className="fleet-row grid grid-cols-[1.4fr_120px_1.6fr_100px_90px_60px] items-center gap-3 px-3 py-2"
                style={{ ['--row-color' as never]: c }}
              >
                <span className="flex items-center gap-1.5 truncate font-mono text-[11px] text-cyan-100/85">
                  <Package className="size-3 shrink-0 text-cyan-300/60" />
                  {r.device_sn}
                </span>
                <span>
                  <span className="chip text-[#5b9eff]">{r.content_type}</span>
                </span>
                <span className="truncate font-mono text-[11px] text-cyan-300/65">{r.file_path || '—'}</span>
                <span className="font-mono text-[11px] text-cyan-300/75">{formatBytes(r.file_size)}</span>
                <span>
                  <span className="chip" style={{ color: c }}>
                    {STATUS_LABEL[r.status]}
                  </span>
                </span>
                <span className="text-right">
                  <IconBtn title="详情" color="#00f0ff" onClick={() => setDetail(r)}>
                    <Download className="size-3.5" />
                  </IconBtn>
                </span>
              </div>
            )
          })}
        </div>
        <Pager page={page} total={total} pageSize={PAGE_SIZE} onPage={setPage} />
      </StateBlock>

      <DownloadDetailDrawer download={detail} open={detail !== null} onClose={() => setDetail(null)} />

      <CollectDrawer open={collectOpen} onClose={() => setCollectOpen(false)} onDone={() => refetch()} />
    </PageShell>
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
  const c = STATUS_COLOR[download.status]
  return (
    <Drawer
      open={open}
      onClose={onClose}
      width={560}
      title={`${download.content_type} · ${download.device_sn}`}
      badge={
        <span className="chip shrink-0" style={{ color: c }}>
          {STATUS_LABEL[download.status]}
        </span>
      }
    >
      <div className="space-y-4 p-4">
        <div className="glass rounded-sm">
          <SectionTitle>采集信息 · COLLECTION</SectionTitle>
          <Field label="设备 SN">
            <span className="font-mono">{download.device_sn}</span>
          </Field>
          <Field label="内容类型">{download.content_type}</Field>
          <Field label="文件路径">
            <span className="font-mono break-all text-[11px]">{download.file_path || '—'}</span>
          </Field>
          <Field label="文件大小">{formatBytes(download.file_size)}</Field>
          {download.checksum ? (
            <Field label="校验和">
              <span className="font-mono break-all text-[11px]">{download.checksum}</span>
            </Field>
          ) : null}
          <Field label="操作人">{download.operator || '—'}</Field>
          <Field label="创建时间">{formatTime(download.created_at)}</Field>
          {download.expires_at ? <Field label="过期时间">{formatTime(download.expires_at)}</Field> : null}
        </div>
      </div>
    </Drawer>
  )
}

function CollectDrawer({
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
  const collect = useCollectDownload()

  const reset = () => {
    setSn('')
    setTypes(['config'])
  }
  const close = () => {
    reset()
    onClose()
  }
  const toggle = (t: string) =>
    setTypes((prev) => (prev.includes(t) ? prev.filter((x) => x !== t) : [...prev, t]))

  const submit = () => {
    if (!sn.trim() || types.length === 0) return
    collect.mutate(
      { device_sn: sn.trim(), content_types: types },
      {
        onSuccess: () => {
          close()
          onDone()
        },
      },
    )
  }

  if (!open) return null

  return (
    <Drawer open={open} onClose={close} width={460} title="发起按需采集" badge={<FileDown className="size-4 text-cyan-300" />}>
      <div className="space-y-4 p-4">
        {collect.isError ? (
          <div className="border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-[11px] text-rose-300">
            采集发起失败，请检查设备在线状态后重试
          </div>
        ) : null}
        <FormRow label="设备 SN">
          <input
            className="neon-input w-full"
            placeholder="ENB00001"
            value={sn}
            onChange={(e) => setSn(e.target.value)}
          />
        </FormRow>
        <FormRow label={`采集内容（${types.length}）`}>
          <div className="flex flex-wrap gap-1.5">
            {COLLECT_TYPES.map((t) => (
              <button
                key={t}
                type="button"
                onClick={() => toggle(t)}
                className={`chip text-[#5b9eff] ${types.includes(t) ? 'shadow-[0_0_8px_currentColor]' : 'opacity-55'}`}
              >
                {t}
              </button>
            ))}
          </div>
        </FormRow>
        <div className="flex justify-end gap-2 pt-2">
          <NeonButton onClick={close}>取消</NeonButton>
          <NeonButton
            icon={collect.isPending ? <Loader2 className="animate-spin" /> : <FileDown />}
            disabled={collect.isPending || !sn.trim() || types.length === 0}
            onClick={submit}
          >
            发起采集
          </NeonButton>
        </div>
      </div>
    </Drawer>
  )
}
