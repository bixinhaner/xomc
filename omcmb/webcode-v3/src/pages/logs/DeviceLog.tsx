import { useMemo, useState } from 'react'
import { Download, FileWarning, HardDrive, RefreshCcw, Search } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatBytes, formatTime } from '@/lib/format'
import {
  useStationLogList,
  useDownloadStationLog,
} from '@core/hooks/api/useStationLog'
import type { StationLogFile } from '@core/services/api/stationLogApi'

import {
  EmptyFeed,
  FeedError,
  FeedLoading,
  FilterInput,
  LOG_PAGE_SIZE,
  Pager,
  StatCard,
} from './_shared'

// ──────────────────────────────────────────────────────────────────────────
// /logs/device —— 基站日志文件（对照 v1 webcode log/device · DeviceLog）
// 真实端点：useStationLogList → stationLogApi GET /station-logs（按设备/类型筛）
//          useDownloadStationLog → GET /station-logs/:id/download（新标签页打开）
// HUD 形态：日志类型统计 + 行列表 + 下载 + 行内下钻到 /logs/device/:id
// ──────────────────────────────────────────────────────────────────────────

type LogType = StationLogFile['logType']

const TYPE_META: Record<LogType, { label: string; color: string }> = {
  running: { label: '运行日志', color: '#00f0ff' },
  fault: { label: '故障日志', color: '#ff7a1a' },
}

const COLS = 'grid-cols-[150px_100px_1.4fr_110px_150px_70px]'

export default function DeviceLogPage() {
  const [page, setPage] = useState(1)
  const [deviceId, setDeviceId] = useState('')
  const [logType, setLogType] = useState<LogType | ''>('')

  const params = useMemo(
    () => ({
      page,
      pageSize: LOG_PAGE_SIZE,
      ...(deviceId.trim() ? { deviceId: deviceId.trim() } : {}),
      ...(logType ? { logType } : {}),
    }),
    [page, deviceId, logType],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useStationLogList(params)
  const download = useDownloadStationLog()
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / LOG_PAGE_SIZE))

  const stats = useMemo(() => {
    let running = 0
    let fault = 0
    let bytes = 0
    rows.forEach((r) => {
      if (r.logType === 'fault') fault += 1
      else running += 1
      bytes += r.fileSize || 0
    })
    return { running, fault, bytes }
  }, [rows])

  return (
    <PageShell
      code="F06"
      title="DEVICE LOG · 基站日志"
      subtitle="STATION LOG FILES"
      isFetching={isFetching}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          {isFetching ? 'SYNC…' : 'REFRESH'}
        </NeonButton>
      }
    >
      <div className="flex h-full flex-col gap-3">
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <StatCard label="TOTAL" color="#00f0ff" value={total} hint="后端总量" />
          <StatCard label="运行日志" color={TYPE_META.running.color} value={stats.running} hint="本页" />
          <StatCard label="故障日志" color={TYPE_META.fault.color} value={stats.fault} hint="本页" />
          <StatCard label="本页体积" color="#a855f7" value={formatBytes(stats.bytes)} hint="本页" />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <FilterInput
            value={deviceId}
            onChange={(v) => {
              setDeviceId(v)
              setPage(1)
            }}
            placeholder="按设备 ID 过滤"
            icon={<Search />}
          />
          {(['', 'running', 'fault'] as const).map((lt) => (
            <button
              key={lt || 'all'}
              type="button"
              onClick={() => {
                setLogType(lt as LogType | '')
                setPage(1)
              }}
              className={`chip ${logType === lt ? 'shadow-[0_0_10px_currentColor]' : 'opacity-60'}`}
              style={{ color: lt ? TYPE_META[lt as LogType].color : '#00f0ff' }}
            >
              {lt ? TYPE_META[lt as LogType].label : 'ALL'}
            </button>
          ))}
        </div>

        <GlassPanel strong className="min-h-0 flex-1 overflow-hidden">
          <div className="h-full overflow-auto">
            <div
              className={`sticky top-0 z-10 grid ${COLS} items-center gap-3 border-b border-cyan-500/15 bg-black/40 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55 backdrop-blur`}
            >
              <span>DEVICE SN</span>
              <span>TYPE</span>
              <span>FILE</span>
              <span>SIZE</span>
              <span>COLLECTED</span>
              <span className="text-right">ACT</span>
            </div>

            {isLoading ? (
              <FeedLoading />
            ) : isError ? (
              <FeedError error={error} />
            ) : rows.length === 0 ? (
              <EmptyFeed icon={<HardDrive className="size-10 text-emerald-400/60" />} text="无基站日志文件" />
            ) : (
              rows.map((r) => {
                const meta = TYPE_META[r.logType]
                return (
                  <div
                    key={r.id}
                    className={`fleet-row grid w-full ${COLS} items-center gap-3 px-3 py-2.5`}
                    style={{ ['--row-color' as never]: meta.color }}
                  >
                    <span className="truncate text-left font-display text-sm font-bold text-cyan-100">
                      {r.deviceSn || '—'}
                    </span>
                    <span className="chip" style={{ color: meta.color }}>
                      {meta.label}
                    </span>
                    <span className="min-w-0 truncate text-left font-mono text-[11px] text-cyan-300/75">
                      {r.fileName || r.objectPath || '—'}
                    </span>
                    <span className="font-mono text-[11px] text-cyan-300/70">
                      {formatBytes(r.fileSize)}
                    </span>
                    <span className="font-mono text-[11px] text-cyan-300/70">
                      {formatTime(r.collectedAt)}
                    </span>
                    <span className="flex justify-end">
                      <button
                        type="button"
                        onClick={() => download.mutate(r.id)}
                        disabled={download.isPending}
                        className="text-cyan-300/70 transition-colors hover:text-cyan-100 disabled:opacity-40 [&_svg]:size-4"
                        title="下载日志文件"
                      >
                        <Download />
                      </button>
                    </span>
                  </div>
                )
              })
            )}
          </div>
        </GlassPanel>

        <Pager page={page} totalPages={totalPages} total={total} onChange={setPage} />
      </div>
    </PageShell>
  )
}

export type { StationLogFile }
// 重新导出仅为方便详情页共享类型；本页主导出为列表组件。
export const DEVICE_LOG_TYPE_META = TYPE_META
export const DEVICE_LOG_FALLBACK_ICON = FileWarning
