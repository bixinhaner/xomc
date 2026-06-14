import { useCallback, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  RefreshCcw,
  Download,
  Trash2,
  AlertOctagon,
  Loader2,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatBytes, formatTime } from '@/lib/format'
import {
  useAbnormalRebootList,
  useDownloadAbnormalReboot,
  useDeleteAbnormalReboot,
} from '@core/hooks/api/useDeviceAbnormalReboot'
import type {
  AbnormalReboot,
  RecordStatus,
} from '@core/services/api/deviceAbnormalRebootApi'
import { StateGate, StatCard, Pager, formatDuration } from './_shared'

const PAGE_SIZE = 20

const STATUS_META: Record<RecordStatus, { label: string; color: string }> = {
  detected: { label: 'DETECTED · 已检测', color: '#ffaa00' },
  file_received: { label: 'RECEIVED · 已收文件', color: '#00ff88' },
  collection_failed: { label: 'FAILED · 采集失败', color: '#ff2d6f' },
}

const STATUS_FILTER: { v: '' | RecordStatus; t: string }[] = [
  { v: '', t: 'ALL' },
  { v: 'detected', t: '已检测' },
  { v: 'file_received', t: '已收文件' },
  { v: 'collection_failed', t: '采集失败' },
]

export default function FleetAbnormalReboot() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [deviceSn, setDeviceSn] = useState('')
  const [recordStatus, setRecordStatus] = useState<'' | RecordStatus>('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
      ...(recordStatus ? { recordStatus } : {}),
    }),
    [page, deviceSn, recordStatus]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useAbnormalRebootList(params)
  const download = useDownloadAbnormalReboot()
  const del = useDeleteAbnormalReboot()

  const items: AbnormalReboot[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const summary = useMemo(() => {
    const acc = { received: 0, failed: 0 }
    for (const r of items) {
      if (r.recordStatus === 'file_received') acc.received++
      else if (r.recordStatus === 'collection_failed') acc.failed++
    }
    return acc
  }, [items])

  const onDelete = useCallback(
    (id: string) => {
      del.mutate(id, { onSuccess: () => void refetch() })
    },
    [del, refetch]
  )

  return (
    <PageShell
      code="F06"
      title="ABNORMAL REBOOT · 异常重启"
      subtitle="STATION FAULT LOGS · HALT REASON FORENSICS"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-60 pl-9"
              placeholder="设备 SN"
              value={deviceSn}
              onChange={(e) => {
                setDeviceSn(e.target.value)
                setPage(1)
              }}
            />
          </div>
          {STATUS_FILTER.map((s) => (
            <button
              key={s.v || 'all'}
              type="button"
              onClick={() => {
                setRecordStatus(s.v)
                setPage(1)
              }}
              className={`chip transition-all ${
                recordStatus === s.v
                  ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                  : 'text-cyan-300/45 hover:text-cyan-300/80'
              }`}
            >
              {s.t}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-3 gap-3">
        <StatCard label="RECORDS" value={total} color="#00f0ff" />
        <StatCard label="FILE RECEIVED" value={summary.received} color="#00ff88" />
        <StatCard label="COLLECT FAILED" value={summary.failed} color="#ff2d6f" />
      </div>

      <StateGate
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={items.length === 0}
        loadingLabel="LOADING FAULT LOGS…"
        emptyLabel="NO ABNORMAL REBOOTS"
      >
        <div className="space-y-2">
          {items.map((r) => {
            const meta = STATUS_META[r.recordStatus]
            return (
              <div
                key={r.id}
                className="fleet-row grid grid-cols-[16px_1.5fr_1.4fr_1fr_140px_120px] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: meta.color }}
              >
                <AlertOctagon className="size-4" style={{ color: meta.color }} />
                <button
                  type="button"
                  className="min-w-0 text-left"
                  onClick={() => navigate(`/fleet/detail/${encodeURIComponent(r.deviceSn)}`)}
                >
                  <div className="truncate font-display text-sm font-bold text-cyan-100">
                    {r.deviceName || r.deviceSn}
                  </div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">
                    SN {r.deviceSn} · {r.deviceType || (r.isGnb ? 'gNB' : 'eNB')}
                  </div>
                </button>
                <div className="min-w-0">
                  <div className="truncate text-xs text-cyan-100/85">
                    {r.haltMainReason || '—'}
                  </div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">
                    {r.haltDetailReason || '—'}
                  </div>
                </div>
                <div className="font-mono text-[10px] text-cyan-300/65">
                  <div className="text-cyan-100/85">{formatDuration(r.runtimeBeforeReboot)}</div>
                  <div>{r.fileName ? formatBytes(r.fileSize) : '—'}</div>
                </div>
                <div className="text-right">
                  <span className="chip" style={{ color: meta.color }}>
                    {meta.label}
                  </span>
                  <div className="mt-0.5 font-mono text-[10px] text-cyan-300/55">
                    {formatTime(r.collectedAt)}
                  </div>
                </div>
                <div className="flex items-center justify-end gap-1.5">
                  {r.recordStatus === 'file_received' && (
                    <NeonButton
                      className="!py-1 !px-2"
                      icon={download.isPending ? <Loader2 className="animate-spin" /> : <Download />}
                      disabled={download.isPending}
                      onClick={() => download.mutate(r.id)}
                    >
                      FILE
                    </NeonButton>
                  )}
                  <NeonButton
                    tone="danger"
                    className="!py-1 !px-2"
                    icon={<Trash2 />}
                    disabled={del.isPending}
                    onClick={() => onDelete(r.id)}
                  >
                    DEL
                  </NeonButton>
                </div>
              </div>
            )
          })}
        </div>

        <Pager
          page={page}
          totalPages={totalPages}
          total={total}
          pageSize={PAGE_SIZE}
          onPrev={() => setPage((p) => Math.max(1, p - 1))}
          onNext={() => setPage((p) => Math.min(totalPages, p + 1))}
        />
      </StateGate>
    </PageShell>
  )
}
