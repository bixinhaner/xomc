import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  RefreshCcw,
  Download,
  Trash2,
  Cpu,
  HardDrive,
  Boxes,
  ChevronRight,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatBytes, formatTime } from '@/lib/format'
import {
  useFileList,
  useDownloadFile,
  useDeleteFiles,
} from '@core/hooks/api/useFiles'
import type {
  ManagedFile,
  FileType,
  FileStatus,
} from '@core/mock/data/fileManagement'

import { ChipFilter, OverviewStat, Pager, Row, StateGate, NEON } from './_shared'

// 网元文件 = 设备维度的文件资产（配置 / 备份 / 日志等，绑定 deviceSn），真实接口 GET /files
const PAGE_SIZE = 20

const TYPE_ORDER: Array<FileType | ''> = ['', 'config', 'backup', 'log', 'report']
const TYPE_LABEL: Record<FileType, string> = {
  config: '配置',
  log: '日志',
  firmware: '固件',
  backup: '备份',
  report: '报表',
  certificate: '证书',
}
const TYPE_COLOR: Record<FileType, string> = {
  config: NEON.cyan,
  log: NEON.amber,
  firmware: NEON.violet,
  backup: NEON.green,
  report: NEON.blue,
  certificate: NEON.gold,
}

const STATUS_TOKEN: Record<FileStatus, string> = {
  available: 'ok',
  uploading: 'warning',
  processing: 'minor',
  expired: 'offline',
  deleted: 'error',
}
const STATUS_LABEL: Record<FileStatus, string> = {
  available: '可用',
  uploading: '上传中',
  processing: '处理中',
  expired: '已过期',
  deleted: '已删除',
}

export default function DeviceFiles() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [fileType, setFileType] = useState<FileType | ''>('')
  const [deviceSn, setDeviceSn] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(fileType ? { fileType } : {}),
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
    }),
    [page, fileType, deviceSn]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useFileList(params)
  const download = useDownloadFile()
  const remove = useDeleteFiles()

  // 网元文件视图只展示绑定到设备（有 deviceSn）的文件
  const rows: ManagedFile[] = useMemo(
    () => (data?.items ?? []).filter((f) => Boolean(f.deviceSn)),
    [data]
  )
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const pageBytes = rows.reduce((s, f) => s + (f.fileSize || 0), 0)
  const deviceCount = new Set(rows.map((f) => f.deviceSn)).size

  return (
    <PageShell
      code="F06"
      title="DEVICE FILES · 网元文件"
      subtitle="PER-DEVICE FILE ASSETS · CONFIG / BACKUP / LOG"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-56 pl-9"
              placeholder="设备 SN"
              value={deviceSn}
              onChange={(e) => {
                setDeviceSn(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <ChipFilter<FileType | ''>
            value={fileType}
            onChange={(v) => {
              setFileType(v)
              setPage(1)
            }}
            options={TYPE_ORDER.map((t) => ({
              value: t,
              label: t ? TYPE_LABEL[t] : 'ALL',
              color: t ? TYPE_COLOR[t] : NEON.cyan,
            }))}
          />
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 lg:grid-cols-4">
        <OverviewStat icon={<Boxes className="size-4" />} label="DEVICES" color={NEON.cyan} value={deviceCount} hint="本页关联设备" />
        <OverviewStat icon={<Cpu className="size-4" />} label="FILES" color={NEON.violet} value={rows.length} hint="本页网元文件" />
        <OverviewStat icon={<HardDrive className="size-4" />} label="PAGE SIZE" color={NEON.green} value={formatBytes(pageBytes)} hint="本页占用" />
        <OverviewStat label="MATCHED" color={NEON.gold} value={total.toLocaleString()} hint="筛选命中（含系统）" />
      </div>

      <GlassPanel strong title="DEVICE FILE MANIFEST" meta={`${rows.length} FILES`}>
        <div className="p-3">
          <div className="grid grid-cols-[1.2fr_2.2fr_1fr_1fr_1.4fr_120px] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            <span>DEVICE</span>
            <span>FILE</span>
            <span>TYPE / SIZE</span>
            <span>STATUS</span>
            <span>UPLOAD</span>
            <span className="text-right">ACTIONS</span>
          </div>
          <div className="mt-1.5 space-y-1.5">
            <StateGate
              isLoading={isLoading}
              isError={isError}
              error={error}
              isEmpty={rows.length === 0}
              loadingLabel="SYNCING…"
              emptyLabel="NO DEVICE FILES · 暂无网元文件"
            >
              {rows.map((f) => (
                <Row
                  key={f.id}
                  color={TYPE_COLOR[f.fileType]}
                  className="grid-cols-[1.2fr_2.2fr_1fr_1fr_1.4fr_120px]"
                >
                  <div className="min-w-0">
                    <div className="truncate text-xs text-cyan-100/85" title={f.deviceName}>{f.deviceName || '—'}</div>
                    <div className="font-mono text-[10px] text-cyan-300/55">{f.deviceSn}</div>
                  </div>
                  <div className="min-w-0">
                    <div className="truncate font-display text-sm font-bold text-cyan-100" title={f.fileName}>{f.fileName}</div>
                    {f.description ? (
                      <div className="truncate font-mono text-[10px] text-cyan-300/45">{f.description}</div>
                    ) : null}
                  </div>
                  <div>
                    <span className="chip" style={{ color: TYPE_COLOR[f.fileType] }}>{TYPE_LABEL[f.fileType]}</span>
                    <div className="mt-0.5 font-mono text-[10px] text-cyan-300/55">{formatBytes(f.fileSize)}</div>
                  </div>
                  <div>
                    <StatusBadge status={STATUS_TOKEN[f.status]} label={STATUS_LABEL[f.status]} />
                  </div>
                  <div>
                    <div className="font-mono text-[11px] text-cyan-300/75">{formatTime(f.uploadTime)}</div>
                    <div className="font-mono text-[10px] text-cyan-300/45">{f.uploader || '—'}</div>
                  </div>
                  <div className="flex justify-end gap-1.5">
                    <NeonButton icon={<Download />} disabled={f.status !== 'available' || download.isPending} onClick={() => download.mutate(f.id)}>
                      DL
                    </NeonButton>
                    <NeonButton tone="danger" icon={<Trash2 />} disabled={remove.isPending} onClick={() => remove.mutate([f.id])}>
                      DEL
                    </NeonButton>
                    <NeonButton icon={<ChevronRight />} onClick={() => navigate(`/files/perf-retrieval/${f.id}`)}>
                      详情
                    </NeonButton>
                  </div>
                </Row>
              ))}
            </StateGate>
          </div>
          <Pager
            page={page}
            totalPages={totalPages}
            total={total}
            pageSize={PAGE_SIZE}
            onPrev={() => setPage((p) => Math.max(1, p - 1))}
            onNext={() => setPage((p) => Math.min(totalPages, p + 1))}
          />
        </div>
      </GlassPanel>
    </PageShell>
  )
}
