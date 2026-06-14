import { useMemo, useState } from 'react'
import {
  Antenna,
  Loader2,
  Radio,
  RefreshCcw,
  Search,
  ToggleLeft,
  ToggleRight,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { cn } from '@/lib/utils'
import { useDeviceList } from '@core/hooks/api/useDevices'
import { useEnableIndicators, useDisableIndicators } from '@core/hooks/api/useIndicator'
import type { Device } from '@core/types/device'

/**
 * F03 · KPI 基站测量（kpi-station）
 * 设备清单 + 每基站 PM 测量上报开关（启用/停用走真实 useEnable·DisableIndicators）。
 * 设备数据全走真实 useDeviceList。
 */

const PAGE_SIZE = 20

type StatusFilter = '' | 'online' | 'offline'

const STATUS_FILTERS: { value: StatusFilter; label: string }[] = [
  { value: '', label: 'ALL' },
  { value: 'online', label: '在线' },
  { value: 'offline', label: '离线' },
]

// 设备是否处于测量上报态：pmReportStatus=enabled 或在线视为上报中。
function isReporting(d: Device): boolean {
  return d.pmReportStatus === 'enabled' || d.isOnline
}

export default function KpiStation() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('')

  const enableMut = useEnableIndicators()
  const disableMut = useDisableIndicators()

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { searchText: keyword.trim() } : {}),
      ...(statusFilter === 'online' ? { isOnline: true } : {}),
      ...(statusFilter === 'offline' ? { isOnline: false } : {}),
    }),
    [page, keyword, statusFilter],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useDeviceList(params)
  const rows: Device[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const reportingCount = useMemo(() => rows.filter(isReporting).length, [rows])
  const onlineCount = useMemo(() => rows.filter((d) => d.isOnline).length, [rows])

  const mutating = enableMut.isPending || disableMut.isPending

  // 切换设备测量上报：以设备 id 作为指标维度操作目标（与 v1 同口径，制式按 networkType 推断）。
  const toggleReport = (d: Device) => {
    const enable = !isReporting(d)
    const mut = enable ? enableMut : disableMut
    const dt = d.networkType?.toUpperCase() === 'NR' ? 'GNB' : d.networkType?.toUpperCase() === 'GSM' ? 'GSM' : 'ENB'
    mut.mutate(
      { deviceType: dt, operatorCode: '', indicatorIds: [d.id], enable },
      { onSettled: () => void refetch() },
    )
  }

  return (
    <PageShell
      code="F03"
      title="STATION MEASUREMENT · 基站测量"
      subtitle="PM REPORTING TOGGLE · PER-STATION COLLECTION"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="SN / 名称 搜索"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          {STATUS_FILTERS.map((f) => (
            <button
              key={f.value || 'all'}
              type="button"
              onClick={() => {
                setStatusFilter(f.value)
                setPage(1)
              }}
              className={cn(
                'chip transition-all',
                statusFilter === f.value
                  ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                  : 'text-cyan-300/55 opacity-70 hover:opacity-100',
              )}
            >
              {f.label}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {/* 概览 */}
      <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="STATIONS · 本页基站" value={String(rows.length)} color="#00f0ff" icon={<Antenna className="size-4" />} />
        <Stat label="ONLINE · 在线" value={String(onlineCount)} color="#00ff88" icon={<Radio className="size-4" />} />
        <Stat label="REPORTING · 上报中" value={String(reportingCount)} color="#a855f7" icon={<Radio className="size-4" />} />
        <Stat label="TOTAL · 总数" value={total.toLocaleString()} color="#ffaa00" icon={<Antenna className="size-4" />} />
      </div>

      <GlassPanel strong title="STATION ROSTER · 基站测量清单" meta={`${total} STATIONS`}>
        <div className="p-3">
          <div className="grid grid-cols-[150px_1.6fr_1fr_1fr_1.2fr_110px] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            <span>SN · 序列号</span>
            <span>NAME · 名称</span>
            <span>STATUS · 连接</span>
            <span>PRODUCT · 产品</span>
            <span>LAST ONLINE</span>
            <span className="text-right">REPORTING</span>
          </div>

          <div className="mt-1.5 space-y-1">
            {isLoading ? (
              <Loading text="SCANNING FLEET…" />
            ) : isError ? (
              <ErrorBox text={`SCAN FAILED · ${error instanceof Error ? error.message : '未知错误'}`} />
            ) : rows.length === 0 ? (
              <EmptyBox text="NO STATION" />
            ) : (
              rows.map((d) => {
                const reporting = isReporting(d)
                return (
                  <div
                    key={d.id}
                    className="grid grid-cols-[150px_1.6fr_1fr_1fr_1.2fr_110px] items-center gap-3 rounded-sm px-3 py-2 transition-colors hover:bg-cyan-500/5"
                  >
                    <span className="truncate font-mono text-[11px] text-cyan-100" title={d.sn}>
                      {d.sn}
                    </span>
                    <span
                      className="truncate font-display text-sm font-bold text-cyan-100"
                      title={d.deviceName || d.name || d.hostName}
                    >
                      {d.deviceName || d.name || d.hostName || '—'}
                    </span>
                    <StatusBadge
                      status={d.isOnline ? 'online' : 'offline'}
                      label={d.isOnline ? '在线' : '离线'}
                      className="w-fit"
                    />
                    <span className="truncate font-mono text-[11px] text-cyan-300/70" title={d.productClass}>
                      {d.productClass || '—'}
                    </span>
                    <span className="font-mono text-[11px] text-cyan-300/65">
                      {d.lastOnlineTime ? formatTime(d.lastOnlineTime) : '—'}
                    </span>
                    <div className="flex justify-end">
                      <button
                        type="button"
                        disabled={mutating}
                        onClick={() => toggleReport(d)}
                        className={cn(
                          'flex items-center gap-1 font-mono text-[10px] uppercase tracking-[0.1em] transition-colors disabled:opacity-40',
                          reporting ? 'text-[#00ff88]' : 'text-cyan-300/45 hover:text-cyan-200',
                        )}
                        title={reporting ? '点击停止上报' : '点击启用上报'}
                      >
                        {reporting ? <ToggleRight className="size-4" /> : <ToggleLeft className="size-4" />}
                        {reporting ? 'ON' : 'OFF'}
                      </button>
                    </div>
                  </div>
                )
              })
            )}
          </div>

          <div className="mt-3 flex items-center justify-between">
            <span className="flex items-center gap-2 font-mono text-[11px] text-cyan-300/55">
              {mutating ? <Loader2 className="size-3 animate-spin" /> : null}
              PAGE {page} / {totalPages} · TOTAL {total}
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
        </div>
      </GlassPanel>
    </PageShell>
  )
}

/* ───────── 复用小件 ───────── */

function Stat({ label, value, color, icon }: { label: string; value: string; color: string; icon: React.ReactNode }) {
  return (
    <div className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3" style={{ borderLeftColor: color }}>
      <div className="scanline" />
      <div className="relative flex items-center justify-between">
        <div>
          <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/60">{label}</div>
          <div className="font-display text-2xl font-bold leading-tight" style={{ color, textShadow: `0 0 8px ${color}` }}>
            {value}
          </div>
        </div>
        <span style={{ color }}>{icon}</span>
      </div>
    </div>
  )
}

function Loading({ text }: { text: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
      <Loader2 className="size-5 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">{text}</span>
    </div>
  )
}

function ErrorBox({ text }: { text: string }) {
  return (
    <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">{text}</div>
  )
}

function EmptyBox({ text }: { text: string }) {
  return (
    <div className="flex items-center justify-center py-12 font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/40">
      {text}
    </div>
  )
}
