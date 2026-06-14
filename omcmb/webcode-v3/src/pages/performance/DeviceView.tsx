import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Antenna, Database, LineChart, Loader2, RefreshCcw, Search, Signal } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { cn } from '@/lib/utils'
import { useDeviceList } from '@core/hooks/api/useDevices'
import { usePMFileDevices } from '@core/hooks/api/usePerformance'
import type { Device } from '@core/types/device'

/**
 * F03 · 设备视图（performance/device-view）
 * 设备机队（真实 useDeviceList）叠加 PM 文件聚合上报态（真实 usePMFileDevices），
 * 让运维按设备视角看「在线 + 是否上报 PM + 文件数」，并一键跳实时取数。
 */

const PAGE_SIZE = 20

type Filter = '' | 'online' | 'reporting'

const FILTERS: { value: Filter; label: string }[] = [
  { value: '', label: 'ALL' },
  { value: 'online', label: '在线' },
  { value: 'reporting', label: '上报中' },
]

export default function DeviceView() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [filter, setFilter] = useState<Filter>('')

  const deviceParams = useMemo(
    () => ({ page, pageSize: PAGE_SIZE, ...(keyword.trim() ? { searchText: keyword.trim() } : {}) }),
    [page, keyword],
  )
  const { data: deviceData, isLoading, isError, error, isFetching, refetch } = useDeviceList(deviceParams)
  const devices: Device[] = deviceData?.items ?? []
  const total = deviceData?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  // PM 文件聚合（按设备）：用于叠加 reporting + fileCount。取较大页覆盖当前设备页。
  const { data: pmData } = usePMFileDevices({ page: 1, pageSize: 200 })
  const pmBySn = useMemo(() => {
    const m = new Map<string, { reporting: boolean; fileCount: number; lastCollectTime: string }>()
    ;(pmData?.items ?? []).forEach((p) => {
      m.set(p.deviceSn, {
        reporting: p.reporting,
        fileCount: p.fileCount,
        lastCollectTime: p.lastCollectTime,
      })
    })
    return m
  }, [pmData])

  const rows = useMemo(() => {
    const merged = devices.map((d) => ({ device: d, pm: pmBySn.get(d.sn) }))
    if (filter === 'online') return merged.filter((r) => r.device.isOnline)
    if (filter === 'reporting') return merged.filter((r) => r.pm?.reporting)
    return merged
  }, [devices, pmBySn, filter])

  const onlineCount = useMemo(() => devices.filter((d) => d.isOnline).length, [devices])
  const reportingCount = useMemo(
    () => devices.filter((d) => pmBySn.get(d.sn)?.reporting).length,
    [devices, pmBySn],
  )

  return (
    <PageShell
      code="F03"
      title="DEVICE VIEW · 设备视图"
      subtitle="PM COVERAGE BY DEVICE · FLEET × REPORTING"
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
          {FILTERS.map((f) => (
            <button
              key={f.value || 'all'}
              type="button"
              onClick={() => setFilter(f.value)}
              className={cn(
                'chip transition-all',
                filter === f.value
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
      <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="DEVICES · 本页设备" value={String(devices.length)} color="#00f0ff" icon={<Antenna className="size-4" />} />
        <Stat label="ONLINE · 在线" value={String(onlineCount)} color="#00ff88" icon={<Signal className="size-4" />} />
        <Stat label="REPORTING · 上报中" value={String(reportingCount)} color="#a855f7" icon={<Database className="size-4" />} />
        <Stat label="TOTAL · 总数" value={total.toLocaleString()} color="#ffaa00" icon={<Antenna className="size-4" />} />
      </div>

      <GlassPanel strong title="DEVICE PM ROSTER · 设备性能视图" meta={`${rows.length} / ${total}`}>
        <div className="p-3">
          <div className="grid grid-cols-[150px_1.6fr_1fr_90px_80px_1.2fr_90px] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            <span>SN · 序列号</span>
            <span>NAME · 名称</span>
            <span>PRODUCT · 产品</span>
            <span className="text-right">FILES</span>
            <span>LINK</span>
            <span>LAST COLLECT</span>
            <span>STATUS</span>
          </div>

          <div className="mt-1.5 space-y-1">
            {isLoading ? (
              <Loading text="SCANNING FLEET…" />
            ) : isError ? (
              <ErrorBox text={`SCAN FAILED · ${error instanceof Error ? error.message : '未知错误'}`} />
            ) : rows.length === 0 ? (
              <EmptyBox text="NO DEVICE" />
            ) : (
              rows.map(({ device: d, pm }) => (
                <div
                  key={d.id}
                  className="grid grid-cols-[150px_1.6fr_1fr_90px_80px_1.2fr_90px] items-center gap-3 rounded-sm px-3 py-2 transition-colors hover:bg-cyan-500/5"
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
                  <span className="truncate font-mono text-[11px] text-cyan-300/70" title={d.productClass}>
                    {d.productClass || '—'}
                  </span>
                  <span className="text-right font-display text-sm font-bold text-cyan-200">
                    {(pm?.fileCount ?? 0).toLocaleString()}
                  </span>
                  <button
                    type="button"
                    onClick={() => navigate('/performance/query')}
                    className="flex items-center gap-1 font-mono text-[10px] uppercase tracking-[0.1em] text-cyan-300 hover:text-cyan-100 hover:underline"
                    title="打开实时取数"
                  >
                    <LineChart className="size-3.5" />
                    取数
                  </button>
                  <span className="font-mono text-[11px] text-cyan-300/65">
                    {pm?.lastCollectTime ? formatTime(pm.lastCollectTime) : '—'}
                  </span>
                  <div className="flex flex-col items-start gap-1">
                    <StatusBadge
                      status={d.isOnline ? 'online' : 'offline'}
                      label={d.isOnline ? '在线' : '离线'}
                      className="w-fit"
                    />
                    {pm?.reporting ? <StatusBadge status="ok" label="上报" className="w-fit" /> : null}
                  </div>
                </div>
              ))
            )}
          </div>

          <div className="mt-3 flex items-center justify-between">
            <span className="font-mono text-[11px] text-cyan-300/55">
              PAGE {page} / {totalPages} · TOTAL {total}
            </span>
            <div className="flex gap-2">
              <NeonButton onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
                ◂ PREV
              </NeonButton>
              <NeonButton onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages}>
                NEXT ▸
              </NeonButton>
            </div>
          </div>
        </div>
      </GlassPanel>
      <p className="mt-2 px-1 font-mono text-[10px] text-cyan-300/35">
        在线态来自 /devices；上报态 / 文件数来自 /pm/files/devices 聚合视图；点「取数」跳实时取数页。
      </p>
    </PageShell>
  )
}

/* ───────── 复用小件 ───────── */

function Stat({ label, value, color, icon }: { label: string; value: string; color: string; icon: React.ReactNode }) {
  return (
    <div className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3" style={{ borderLeftColor: color }}>
      <div className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
        <span style={{ color }}>{icon}</span>
        {label}
      </div>
      <div className="font-display text-2xl font-bold leading-tight" style={{ color, textShadow: `0 0 8px ${color}` }}>
        {value}
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
