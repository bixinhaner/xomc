import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  RefreshCcw,
  Download,
  Trash2,
  LineChart,
  Database,
  HardDrive,
  ChevronRight,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { Sparkline } from '@/components/viz/Sparkline'
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

// 性能数据获取 = 性能结果文件库（PM/KPI 报表产物），真实接口 GET /files?file_type=report
const PAGE_SIZE = 20

type PerfType = Extract<FileType, 'report' | 'config'>
const PERF_TYPES: PerfType[] = ['report', 'config']

const TYPE_META: Record<PerfType, { color: string; label: string }> = {
  report: { color: NEON.cyan, label: '性能报表' },
  config: { color: NEON.violet, label: '采集配置' },
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

export default function PerfRetrieval() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [fileType, setFileType] = useState<PerfType>('report')
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({
      fileType,
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, fileType, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useFileList(params)
  const download = useDownloadFile()
  const remove = useDeleteFiles()

  const rows: ManagedFile[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const pageBytes = rows.reduce((s, f) => s + (f.fileSize || 0), 0)
  const deviceCount = new Set(rows.map((f) => f.deviceSn).filter(Boolean)).size
  const sizeSpark = useMemo(
    () =>
      [...rows]
        .sort((a, b) => new Date(a.uploadTime).getTime() - new Date(b.uploadTime).getTime())
        .map((f) => Math.round((f.fileSize || 0) / 1024)),
    [rows]
  )

  return (
    <PageShell
      code="F06"
      title="PERF PULL · 性能获取"
      subtitle="PM RESULT FILE VAULT · KPI / COUNTER"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="文件名 / 设备 SN"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <ChipFilter<PerfType>
            value={fileType}
            onChange={(v) => {
              setFileType(v)
              setPage(1)
            }}
            options={PERF_TYPES.map((t) => ({ value: t, label: TYPE_META[t].label, color: TYPE_META[t].color }))}
          />
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-3 xl:grid-cols-[1fr_300px]">
        <div className="flex flex-col gap-3">
          <div className="grid grid-cols-2 gap-3 lg:grid-cols-3">
            <OverviewStat icon={<Database className="size-4" />} label="FILES" color={NEON.cyan} value={total.toLocaleString()} hint="性能文件总数" />
            <OverviewStat icon={<HardDrive className="size-4" />} label="PAGE SIZE" color={NEON.green} value={formatBytes(pageBytes)} hint="本页占用" />
            <OverviewStat icon={<LineChart className="size-4" />} label="DEVICES" color={NEON.violet} value={deviceCount} hint="本页关联设备" />
          </div>

          <GlassPanel strong title="PERF FILE MANIFEST" meta={`${total} FILES`}>
            <div className="p-3">
              <div className="grid grid-cols-[2.4fr_1fr_1fr_1.4fr_1fr_120px] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                <span>FILE</span>
                <span>SIZE</span>
                <span>STATUS</span>
                <span>DEVICE</span>
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
                  emptyLabel="NO PERF FILES · 暂无性能文件"
                >
                  {rows.map((f) => (
                    <Row
                      key={f.id}
                      color={TYPE_META[fileType].color}
                      className="grid-cols-[2.4fr_1fr_1fr_1.4fr_1fr_120px]"
                    >
                      <div className="min-w-0">
                        <div className="truncate font-display text-sm font-bold text-cyan-100" title={f.fileName}>{f.fileName}</div>
                        {f.description ? (
                          <div className="truncate font-mono text-[10px] text-cyan-300/45">{f.description}</div>
                        ) : null}
                      </div>
                      <div className="font-mono text-[11px] text-cyan-300/75">{formatBytes(f.fileSize)}</div>
                      <div>
                        <StatusBadge status={STATUS_TOKEN[f.status]} label={STATUS_LABEL[f.status]} />
                      </div>
                      <div className="min-w-0">
                        <div className="truncate text-xs text-cyan-100/85">{f.deviceName || (f.deviceSn ? '—' : '系统资产')}</div>
                        <div className="font-mono text-[10px] text-cyan-300/55">{f.deviceSn || '—'}</div>
                      </div>
                      <div className="font-mono text-[11px] text-cyan-300/75">{formatTime(f.uploadTime)}</div>
                      <div className="flex justify-end gap-1.5">
                        <NeonButton
                          icon={<Download />}
                          disabled={f.status !== 'available' || download.isPending}
                          onClick={() => download.mutate(f.id)}
                        >
                          DL
                        </NeonButton>
                        <NeonButton tone="danger" icon={<Trash2 />} disabled={remove.isPending} onClick={() => remove.mutate([f.id])}>
                          DEL
                        </NeonButton>
                        <NeonButton icon={<ChevronRight />} onClick={() => navigate(`/file/perf-retrieval/${f.id}`)}>
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
        </div>

        <GlassPanel title="SIZE TREND" meta="KB / 上传序">
          <div className="flex items-center justify-center p-4">
            {sizeSpark.length > 1 ? (
              <Sparkline data={sizeSpark} width={250} height={64} color={TYPE_META[fileType].color} fill />
            ) : (
              <span className="py-6 font-mono text-[11px] uppercase tracking-[0.18em] text-cyan-300/45">
                INSUFFICIENT DATA
              </span>
            )}
          </div>
        </GlassPanel>
      </div>
    </PageShell>
  )
}
