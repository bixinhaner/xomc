import { useMemo, useState } from 'react'
import { AlertTriangle, Download, RefreshCcw, Search, ServerCrash } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatBytes, formatTime } from '@/lib/format'
import {
  useAbnormalRebootList,
  useDownloadAbnormalReboot,
} from '@core/hooks/api/useDeviceAbnormalReboot'
import type {
  AbnormalReboot,
  RecordStatus,
} from '@core/services/api/deviceAbnormalRebootApi'

import {
  DRow,
  DetailOverlay,
  EmptyFeed,
  FeedError,
  FeedLoading,
  FilterInput,
  LOG_PAGE_SIZE,
  Pager,
  StatCard,
} from './_shared'

// ──────────────────────────────────────────────────────────────────────────
// /logs/exception —— 设备异常重启记录（对照 v1 webcode log/exception → device/abnormal-reboot）
// 真实端点：useAbnormalRebootList → deviceAbnormalRebootApi GET /device-abnormal-reboots
//          useDownloadAbnormalReboot → GET /:id/download
// HUD 形态：状态统计 + 行列表 + 下载 + 详情下钻（含运行时长 / 停机主因）
// ──────────────────────────────────────────────────────────────────────────

const STATUS_META: Record<RecordStatus, { label: string; color: string }> = {
  detected: { label: '已检出', color: '#ffaa00' },
  file_received: { label: '已收文件', color: '#00ff88' },
  collection_failed: { label: '采集失败', color: '#ff2d6f' },
}

const COLS = 'grid-cols-[150px_90px_110px_1.2fr_120px_140px_60px]'

function runtimeText(sec: number): string {
  if (!sec || sec < 0) return '—'
  const d = Math.floor(sec / 86400)
  const h = Math.floor((sec % 86400) / 3600)
  const m = Math.floor((sec % 3600) / 60)
  if (d > 0) return `${d}d ${h}h`
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

export default function ExceptionLogPage() {
  const [page, setPage] = useState(1)
  const [deviceSn, setDeviceSn] = useState('')
  const [recordStatus, setRecordStatus] = useState<RecordStatus | ''>('')
  const [deviceType, setDeviceType] = useState('')
  const [selected, setSelected] = useState<AbnormalReboot | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize: LOG_PAGE_SIZE,
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
      ...(recordStatus ? { recordStatus } : {}),
      ...(deviceType ? { deviceType } : {}),
    }),
    [page, deviceSn, recordStatus, deviceType],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useAbnormalRebootList(params)
  const download = useDownloadAbnormalReboot()
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / LOG_PAGE_SIZE))

  const statusCount = useMemo(() => {
    const c: Record<string, number> = { detected: 0, file_received: 0, collection_failed: 0 }
    rows.forEach((r) => {
      c[r.recordStatus] = (c[r.recordStatus] ?? 0) + 1
    })
    return c
  }, [rows])

  return (
    <PageShell
      code="F06"
      title="EXCEPTION LOG · 异常重启"
      subtitle="ABNORMAL REBOOT RECORDS"
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
          <StatCard
            label="已收文件"
            color={STATUS_META.file_received.color}
            value={statusCount.file_received}
            hint="本页"
          />
          <StatCard
            label="已检出"
            color={STATUS_META.detected.color}
            value={statusCount.detected}
            hint="本页"
          />
          <StatCard
            label="采集失败"
            color={STATUS_META.collection_failed.color}
            value={statusCount.collection_failed}
            hint="本页"
          />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <FilterInput
            value={deviceSn}
            onChange={(v) => {
              setDeviceSn(v)
              setPage(1)
            }}
            placeholder="按设备 SN 过滤"
            icon={<Search />}
          />
          <select
            className="neon-input w-32"
            value={deviceType}
            onChange={(e) => {
              setDeviceType(e.target.value)
              setPage(1)
            }}
          >
            <option value="">全部制式</option>
            <option value="eNB">eNB (4G)</option>
            <option value="gNB">gNB (5G)</option>
          </select>
          {(['', 'detected', 'file_received', 'collection_failed'] as const).map((s) => (
            <button
              key={s || 'all'}
              type="button"
              onClick={() => {
                setRecordStatus(s as RecordStatus | '')
                setPage(1)
              }}
              className={`chip ${recordStatus === s ? 'shadow-[0_0_10px_currentColor]' : 'opacity-60'}`}
              style={{ color: s ? STATUS_META[s as RecordStatus].color : '#00f0ff' }}
            >
              {s ? STATUS_META[s as RecordStatus].label : 'ALL'}
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
              <span>STATUS</span>
              <span>停机主因</span>
              <span>RUNTIME</span>
              <span>COLLECTED</span>
              <span className="text-right">ACT</span>
            </div>

            {isLoading ? (
              <FeedLoading />
            ) : isError ? (
              <FeedError error={error} />
            ) : rows.length === 0 ? (
              <EmptyFeed icon={<ServerCrash className="size-10 text-emerald-400/60" />} text="无异常重启记录" />
            ) : (
              rows.map((r) => {
                const meta = STATUS_META[r.recordStatus] ?? {
                  label: r.recordStatus,
                  color: '#6b86b6',
                }
                const canDownload = r.recordStatus === 'file_received' && Boolean(r.fileName)
                return (
                  <div
                    key={r.id}
                    className={`fleet-row grid w-full ${COLS} items-center gap-3 px-3 py-2.5`}
                    style={{ ['--row-color' as never]: meta.color }}
                  >
                    <button
                      type="button"
                      onClick={() => setSelected(r)}
                      className="truncate text-left font-display text-sm font-bold text-cyan-100 hover:text-cyan-300"
                      title="查看详情"
                    >
                      {r.deviceSn || '—'}
                    </button>
                    <span className="chip" style={{ color: r.isGnb ? '#a855f7' : '#5b9eff' }}>
                      {r.deviceType || (r.isGnb ? 'gNB' : 'eNB')}
                    </span>
                    <span className="chip" style={{ color: meta.color }}>
                      {meta.label}
                    </span>
                    <span className="min-w-0 truncate text-xs text-cyan-100/85">
                      {r.haltMainReason || '—'}
                    </span>
                    <span className="font-mono text-[11px] text-cyan-300/70">
                      {runtimeText(r.runtimeBeforeReboot)}
                    </span>
                    <span className="font-mono text-[11px] text-cyan-300/70">
                      {formatTime(r.collectedAt)}
                    </span>
                    <span className="flex justify-end">
                      <button
                        type="button"
                        onClick={() => canDownload && download.mutate(r.id)}
                        disabled={!canDownload || download.isPending}
                        className="text-cyan-300/70 transition-colors hover:text-cyan-100 disabled:opacity-30 [&_svg]:size-4"
                        title={canDownload ? '下载日志文件' : '无可下载文件'}
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

        {selected && (
          <DetailOverlay
            title={`${selected.deviceSn} · ${STATUS_META[selected.recordStatus]?.label ?? selected.recordStatus}`}
            meta={formatTime(selected.collectedAt)}
            accent={STATUS_META[selected.recordStatus]?.color ?? '#00f0ff'}
            onClose={() => setSelected(null)}
          >
            <dl className="grid grid-cols-[120px_1fr] gap-x-4 gap-y-2.5 p-1 text-[12px]">
              <DRow k="设备名" v={selected.deviceName} />
              <DRow k="设备 SN" v={selected.deviceSn} mono />
              <DRow k="制式" v={selected.deviceType || (selected.isGnb ? 'gNB' : 'eNB')} />
              <DRow k="软件版本" v={selected.softwareVersion} mono />
              <DRow k="管理 IP" v={selected.operateIp} mono />
              <DRow
                k="状态"
                v={STATUS_META[selected.recordStatus]?.label ?? selected.recordStatus}
                color={STATUS_META[selected.recordStatus]?.color}
              />
              <DRow k="停机主因" v={selected.haltMainReason} color="#ff7a1a" />
              <DRow k="停机明细" v={selected.haltDetailReason} />
              <DRow k="重启前运行" v={runtimeText(selected.runtimeBeforeReboot)} />
              <DRow k="文件名" v={selected.fileName} mono />
              <DRow k="文件大小" v={selected.fileName ? formatBytes(selected.fileSize) : null} />
              {selected.collectionFailReason && (
                <DRow k="采集失败原因" v={selected.collectionFailReason} color="#ff2d6f" />
              )}
              <DRow k="采集时间" v={formatTime(selected.collectedAt)} mono />
            </dl>
            {selected.recordStatus === 'file_received' && selected.fileName && (
              <div className="mt-3 flex justify-end">
                <NeonButton
                  icon={<Download />}
                  onClick={() => download.mutate(selected.id)}
                  disabled={download.isPending}
                >
                  {download.isPending ? '…' : '下载日志文件'}
                </NeonButton>
              </div>
            )}
            {selected.recordStatus === 'collection_failed' && (
              <div className="mt-3 flex items-center gap-2 border border-rose-500/30 bg-rose-500/5 px-3 py-2 text-[11px] text-rose-300">
                <AlertTriangle className="size-3.5" />
                {selected.collectionFailReason || '日志采集失败，无可下载文件'}
              </div>
            )}
          </DetailOverlay>
        )}
      </div>
    </PageShell>
  )
}
