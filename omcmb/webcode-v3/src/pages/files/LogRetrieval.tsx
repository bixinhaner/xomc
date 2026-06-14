import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  RefreshCcw,
  Download,
  Trash2,
  ScrollText,
  AlertOctagon,
  FileStack,
  ChevronRight,
} from 'lucide-react'

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

import { ChipFilter, OverviewStat, Pager, Row, StateGate, NEON } from './_shared'

// 日志获取 = 基站日志文件库（运行日志 / 故障日志），真实接口 GET /station-logs
const PAGE_SIZE = 20

type LogTypeFilter = '' | 'running' | 'fault'

const LOG_TYPE_META: Record<StationLogFile['logType'], { color: string; label: string }> = {
  running: { color: NEON.cyan, label: '运行日志' },
  fault: { color: NEON.amber, label: '故障日志' },
}

export default function LogRetrieval() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [logType, setLogType] = useState<LogTypeFilter>('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(logType ? { logType } : {}),
    }),
    [page, logType]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useStationLogList(params)
  const download = useDownloadStationLog()
  const remove = useDeleteStationLog()

  const rows: StationLogFile[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const runningCount = rows.filter((r) => r.logType === 'running').length
  const faultCount = rows.filter((r) => r.logType === 'fault').length
  const pageBytes = rows.reduce((s, r) => s + (r.fileSize || 0), 0)
  const deviceCount = new Set(rows.map((r) => r.deviceSn)).size

  return (
    <PageShell
      code="F06"
      title="LOG PULL · 日志获取"
      subtitle="STATION LOG VAULT · RUNNING / FAULT"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <ChipFilter<LogTypeFilter>
            value={logType}
            onChange={(v) => {
              setLogType(v)
              setPage(1)
            }}
            options={[
              { value: '', label: 'ALL' },
              { value: 'running', label: '运行', color: NEON.cyan },
              { value: 'fault', label: '故障', color: NEON.amber },
            ]}
          />
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 lg:grid-cols-4">
        <OverviewStat icon={<FileStack className="size-4" />} label="LOG FILES" color={NEON.cyan} value={total.toLocaleString()} hint="日志文件总数" />
        <OverviewStat icon={<ScrollText className="size-4" />} label="RUNNING" color={NEON.cyan} value={runningCount} hint="本页运行日志" />
        <OverviewStat icon={<AlertOctagon className="size-4" />} label="FAULT" color={NEON.amber} value={faultCount} hint="本页故障日志" />
        <OverviewStat label="PAGE SIZE" color={NEON.green} value={formatBytes(pageBytes)} hint={`${deviceCount} 台设备`} />
      </div>

      <GlassPanel strong title="STATION LOG MANIFEST" meta={`${total} FILES`}>
        <div className="p-3">
          <div className="grid grid-cols-[2.4fr_1fr_1fr_1fr_1.4fr_140px] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            <span>FILE</span>
            <span>DEVICE SN</span>
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
              emptyLabel="NO STATION LOGS · 暂无基站日志文件"
            >
              {rows.map((f) => {
                const meta = LOG_TYPE_META[f.logType]
                return (
                  <Row
                    key={f.id}
                    color={f.logType === 'fault' ? NEON.amber : NEON.cyan}
                    className="grid-cols-[2.4fr_1fr_1fr_1fr_1.4fr_140px]"
                  >
                    <div className="min-w-0">
                      <div className="truncate font-display text-sm font-bold text-cyan-100" title={f.fileName}>{f.fileName}</div>
                      {f.faultReason ? (
                        <div className="truncate font-mono text-[10px] text-rose-300/70" title={f.faultDetail || f.faultReason}>{f.faultReason}</div>
                      ) : (
                        <div className="truncate font-mono text-[10px] text-cyan-300/45">{f.objectPath}</div>
                      )}
                    </div>
                    <div className="font-mono text-[11px] text-cyan-100/85">{f.deviceSn}</div>
                    <div>
                      <span className="chip" style={{ color: meta.color }}>{meta.label}</span>
                      <div className="mt-0.5 font-mono text-[10px] text-cyan-300/55">{formatBytes(f.fileSize)}</div>
                    </div>
                    <div>
                      <StatusBadge status={f.isDeleted ? 'offline' : 'ok'} label={f.isDeleted ? '已删除' : '可用'} />
                    </div>
                    <div className="font-mono text-[11px] text-cyan-300/75">{formatTime(f.collectedAt)}</div>
                    <div className="flex justify-end gap-1.5">
                      <NeonButton
                        icon={<Download />}
                        disabled={f.isDeleted || download.isPending}
                        onClick={() => download.mutate(f.id)}
                      >
                        DL
                      </NeonButton>
                      <NeonButton
                        tone="danger"
                        icon={<Trash2 />}
                        disabled={f.isDeleted || remove.isPending}
                        onClick={() => remove.mutate(f.id)}
                      >
                        DEL
                      </NeonButton>
                      <NeonButton icon={<ChevronRight />} onClick={() => navigate(`/files/log-retrieval/${f.deviceSn}`)}>
                        设备
                      </NeonButton>
                    </div>
                  </Row>
                )
              })}
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
