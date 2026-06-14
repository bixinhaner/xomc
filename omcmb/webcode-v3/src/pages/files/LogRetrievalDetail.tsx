import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, RefreshCcw, Download, Trash2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatBytes, formatTime } from '@/lib/format'
import {
  useStationLogList,
  useDownloadStationLog,
  useDeleteStationLog,
} from '@core/hooks/api/useStationLog'
import type { StationLogFile } from '@core/services/api/stationLogApi'

import { KV, OverviewStat, Row, StateGate, NEON } from './_shared'

const LOG_TYPE_META: Record<StationLogFile['logType'], { color: string; label: string }> = {
  running: { color: NEON.cyan, label: '运行日志' },
  fault: { color: NEON.amber, label: '故障日志' },
}

export default function LogRetrievalDetail() {
  const { deviceSn = '' } = useParams<{ deviceSn: string }>()
  const navigate = useNavigate()

  // station-logs 接口按 deviceId 过滤，这里取较大窗口后按 deviceSn 客户端归并
  const { data, isLoading, isError, error, isFetching, refetch } = useStationLogList({
    page: 1,
    pageSize: 200,
  })
  const download = useDownloadStationLog()
  const remove = useDeleteStationLog()

  const rows: StationLogFile[] = useMemo(
    () => (data?.items ?? []).filter((f) => f.deviceSn === deviceSn),
    [data, deviceSn]
  )

  const runningCount = rows.filter((r) => r.logType === 'running').length
  const faultCount = rows.filter((r) => r.logType === 'fault').length
  const totalBytes = rows.reduce((s, r) => s + (r.fileSize || 0), 0)
  const latest = rows.reduce<StationLogFile | null>((acc, r) => {
    if (!acc) return r
    return r.collectedAt > acc.collectedAt ? r : acc
  }, null)

  return (
    <PageShell
      code="F06"
      title={`LOG PULL · ${deviceSn}`}
      subtitle="STATION LOG DOSSIER · PER DEVICE"
      isFetching={isFetching}
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/files/log-retrieval')}>
            返回列表
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <OverviewStat label="LOG FILES" color={NEON.cyan} value={rows.length} hint="该设备日志数" />
          <OverviewStat label="RUNNING" color={NEON.cyan} value={runningCount} hint="运行日志" />
          <OverviewStat label="FAULT" color={NEON.amber} value={faultCount} hint="故障日志" />
          <OverviewStat label="TOTAL SIZE" color={NEON.green} value={formatBytes(totalBytes)} hint="累计占用" />
        </div>

        <div className="grid grid-cols-1 gap-3 xl:grid-cols-[300px_1fr]">
          <GlassPanel title="DEVICE">
            <div className="py-1">
              <KV label="设备 SN">{deviceSn}</KV>
              <KV label="最新采集">{latest ? formatTime(latest.collectedAt) : '—'}</KV>
              <KV label="最新文件">{latest?.fileName || '—'}</KV>
              <KV label="桶 / 路径">{latest ? `${latest.bucket} / ${latest.objectPath}` : '—'}</KV>
            </div>
          </GlassPanel>

          <GlassPanel strong title="LOG HISTORY" meta={`${rows.length} FILES`}>
            <div className="p-3">
              <div className="grid grid-cols-[2.4fr_1fr_1fr_1.4fr_120px] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                <span>FILE</span>
                <span>TYPE / SIZE</span>
                <span>STATUS</span>
                <span>COLLECTED</span>
                <span className="text-right">ACTIONS</span>
              </div>
              <div className="mt-1.5 space-y-1.5">
                <StateGate
                  isLoading={isLoading}
                  isError={isError}
                  error={error}
                  isEmpty={rows.length === 0}
                  loadingLabel="SYNCING LOGS…"
                  emptyLabel="NO LOGS · 该设备暂无日志文件"
                >
                  {rows.map((f) => {
                    const meta = LOG_TYPE_META[f.logType]
                    return (
                      <Row
                        key={f.id}
                        color={f.logType === 'fault' ? NEON.amber : NEON.cyan}
                        className="grid-cols-[2.4fr_1fr_1fr_1.4fr_120px]"
                      >
                        <div className="min-w-0">
                          <div className="truncate font-display text-sm font-bold text-cyan-100" title={f.fileName}>{f.fileName}</div>
                          {f.faultReason ? (
                            <div className="truncate font-mono text-[10px] text-rose-300/70" title={f.faultDetail || f.faultReason}>{f.faultReason}</div>
                          ) : null}
                        </div>
                        <div>
                          <span className="chip" style={{ color: meta.color }}>{meta.label}</span>
                          <div className="mt-0.5 font-mono text-[10px] text-cyan-300/55">{formatBytes(f.fileSize)}</div>
                        </div>
                        <div>
                          <StatusBadge status={f.isDeleted ? 'offline' : 'ok'} label={f.isDeleted ? '已删除' : '可用'} />
                        </div>
                        <div className="font-mono text-[11px] text-cyan-300/75">{formatTime(f.collectedAt)}</div>
                        <div className="flex justify-end gap-1.5">
                          <NeonButton icon={<Download />} disabled={f.isDeleted || download.isPending} onClick={() => download.mutate(f.id)}>
                            DL
                          </NeonButton>
                          <NeonButton tone="danger" icon={<Trash2 />} disabled={f.isDeleted || remove.isPending} onClick={() => remove.mutate(f.id)}>
                            DEL
                          </NeonButton>
                        </div>
                      </Row>
                    )
                  })}
                </StateGate>
              </div>
            </div>
          </GlassPanel>
        </div>
      </div>
    </PageShell>
  )
}
