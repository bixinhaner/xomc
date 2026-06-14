import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Download, FileX2, Loader2, RefreshCcw } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatBytes, formatTime } from '@/lib/format'
import { useMRFiles, useDownloadMRFile } from '@core/hooks/api/useMR'
import type { MRFileItem } from '@core/services/api/mrApi'

const PAGE_SIZE = 20

const MR_TYPE_COLOR: Record<string, string> = {
  MRO: '#00f0ff',
  MRE: '#00ff88',
  MRS: '#ffaa00',
}

/**
 * 单设备 MR 文件（/mr/files/:deviceSn）。
 * 文件列表经 useMRFiles({deviceSn})，下载经 useDownloadMRFile。全部 @core 真实数据；可按起止时间筛选。
 */
export default function FilesDevice() {
  const navigate = useNavigate()
  const { deviceSn = '' } = useParams<{ deviceSn: string }>()
  const [page, setPage] = useState(1)
  const [mrType, setMrType] = useState('')
  const [start, setStart] = useState('')
  const [end, setEnd] = useState('')

  const timeRange = useMemo<[string, string] | undefined>(() => {
    if (start && end) {
      return [new Date(start).toISOString(), new Date(end).toISOString()]
    }
    return undefined
  }, [start, end])

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      deviceSn,
      mrType: mrType || undefined,
      timeRange,
    }),
    [page, deviceSn, mrType, timeRange],
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useMRFiles(params)
  const download = useDownloadMRFile()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const TYPE_FILTERS = ['', 'MRO', 'MRE', 'MRS'] as const

  return (
    <PageShell
      code="F05"
      title={`MR FILES · ${deviceSn}`}
      subtitle="SINGLE DEVICE MR FILE ARCHIVE"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/mr/files')}>
            BACK
          </NeonButton>
          {TYPE_FILTERS.map((tp) => (
            <button
              key={tp || 'all'}
              type="button"
              onClick={() => {
                setMrType(tp)
                setPage(1)
              }}
              className={`chip transition-all ${
                mrType === tp ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55 hover:opacity-100'
              }`}
              style={{ color: tp ? MR_TYPE_COLOR[tp] : '#6b86b6' }}
            >
              {tp || 'ALL'}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => void refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <GlassPanel title="COLLECT TIME RANGE · 采集时间窗" className="mb-3">
        <div className="flex flex-wrap items-center gap-3 p-3">
          <label className="flex items-center gap-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            FROM
            <input
              type="datetime-local"
              className="neon-input"
              value={start}
              onChange={(e) => {
                setStart(e.target.value)
                setPage(1)
              }}
            />
          </label>
          <label className="flex items-center gap-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            TO
            <input
              type="datetime-local"
              className="neon-input"
              value={end}
              onChange={(e) => {
                setEnd(e.target.value)
                setPage(1)
              }}
            />
          </label>
          {start || end ? (
            <button
              type="button"
              className="chip text-cyan-300/70 hover:opacity-100"
              onClick={() => {
                setStart('')
                setEnd('')
                setPage(1)
              }}
            >
              清除时间窗
            </button>
          ) : null}
        </div>
      </GlassPanel>

      {isLoading ? (
        <div className="flex items-center justify-center gap-2 py-20 text-cyan-300/60">
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING FILES…</span>
        </div>
      ) : isError ? (
        <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
          FAILURE · {error instanceof Error ? error.message : '加载失败'}
        </div>
      ) : rows.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <FileX2 className="size-10 text-cyan-400/50" />
          <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/60">
            NO FILES · 该设备暂无 MR 文件
          </div>
        </div>
      ) : (
        <div className="space-y-1.5">
          {rows.map((f: MRFileItem) => {
            const color = MR_TYPE_COLOR[f.mrType] ?? '#6b86b6'
            return (
              <div
                key={f.id}
                className="fleet-row grid grid-cols-[64px_1fr_1fr_120px_100px] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: color }}
              >
                <span className="chip justify-center" style={{ color }}>
                  {f.mrType}
                </span>
                <div className="min-w-0 font-mono text-xs text-cyan-100/85">
                  <div className="truncate">{f.fileName}</div>
                  <div className="font-mono text-[10px] text-cyan-300/45">
                    {f.parsed ? '已解析' : '未解析'} · {Number(f.recordCount ?? 0).toLocaleString()} rec
                  </div>
                </div>
                <div className="font-mono text-[10px] text-cyan-300/70">
                  COLLECT · {formatTime(f.collectTime)}
                </div>
                <div className="font-mono text-xs text-cyan-300/80">{formatBytes(f.fileSize)}</div>
                <div className="text-right">
                  <NeonButton
                    className="px-2 py-0.5"
                    icon={<Download />}
                    disabled={download.isPending}
                    onClick={() => download.mutate(f.id)}
                  >
                    下载
                  </NeonButton>
                </div>
              </div>
            )
          })}
        </div>
      )}

      <div className="mt-4 flex items-center justify-between">
        <span className="font-mono text-[11px] text-cyan-300/55">
          PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
        </span>
        <div className="flex gap-2">
          <NeonButton onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
            ◂ PREV
          </NeonButton>
          <NeonButton
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages}
          >
            NEXT ▸
          </NeonButton>
        </div>
      </div>
    </PageShell>
  )
}
