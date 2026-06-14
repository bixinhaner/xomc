import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Wifi, WifiOff, Radio } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { formatTime } from '@/lib/format'
import { useDeviceList } from '@core/hooks/api/useDevices'
import type { Device } from '@core/types/device'
import { StateGate, StatCard, Pager } from './_shared'

const PAGE_SIZE = 20
type Mode = 'online' | 'offline'

export default function FleetOnlineMonitor() {
  const navigate = useNavigate()
  const [mode, setMode] = useState<Mode>('online')
  const [page, setPage] = useState(1)

  const params = useMemo(
    () => ({ page, pageSize: PAGE_SIZE, isOnline: mode === 'online' }),
    [page, mode]
  )

  // 15s 自动刷新——实时在线监控
  const { data, isLoading, isError, error, isFetching, refetch } = useDeviceList(params, {
    refetchInterval: 15000,
  })

  const items: Device[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const stats = data?.stats
  const onlineCount = stats?.online_count ?? stats?.online ?? 0
  const offlineCount = stats?.offline_count ?? stats?.offline ?? 0
  const grandTotal = stats?.total ?? onlineCount + offlineCount
  const onlinePct = grandTotal > 0 ? (onlineCount / grandTotal) * 100 : 0

  return (
    <PageShell
      code="F06"
      title="ONLINE MONITOR · 在线监控"
      subtitle="LIVE CONNECTIVITY · 15s AUTO-REFRESH"
      isFetching={isFetching}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      {/* 在线率仪表 + 统计 */}
      <div className="mb-4 grid grid-cols-1 gap-3 md:grid-cols-12">
        <GlassPanel strong className="md:col-span-4">
          <div className="flex items-center justify-center gap-4 p-4">
            <RadialGauge value={onlinePct} label="ONLINE" size={120} color="#00ff88" />
            <div className="space-y-1">
              <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                CONNECTIVITY
              </div>
              <div className="font-display text-2xl font-bold text-emerald-200 text-glow">
                {onlinePct.toFixed(1)}%
              </div>
              <div className="font-mono text-[11px] text-cyan-300/55">{grandTotal} TOTAL</div>
            </div>
          </div>
        </GlassPanel>
        <div className="grid grid-cols-2 gap-3 md:col-span-8 md:grid-cols-3">
          <StatCard label="ONLINE" value={onlineCount} color="#00ff88" hint="实时在线" />
          <StatCard label="OFFLINE" value={offlineCount} color="#525a78" hint="实时离线" />
          <StatCard label="ALARMED" value={stats?.alarmed ?? 0} color="#ff2d6f" hint="含活动告警" />
        </div>
      </div>

      {/* 模式切换 */}
      <div className="mb-3 flex items-center gap-2">
        {(
          [
            { k: 'online' as const, label: 'ONLINE · 在线', icon: <Wifi className="size-3.5" /> },
            { k: 'offline' as const, label: 'OFFLINE · 离线', icon: <WifiOff className="size-3.5" /> },
          ]
        ).map((m) => (
          <button
            key={m.k}
            type="button"
            onClick={() => {
              setMode(m.k)
              setPage(1)
            }}
            className={`chip flex items-center gap-1.5 transition-all ${
              mode === m.k
                ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                : 'text-cyan-300/45 hover:text-cyan-300/80'
            }`}
          >
            {m.icon}
            {m.label}
          </button>
        ))}
      </div>

      <StateGate
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={items.length === 0}
        loadingLabel="SCANNING…"
        emptyLabel={mode === 'online' ? 'NO ONLINE UNITS' : 'NO OFFLINE UNITS'}
      >
        <div className="space-y-2">
          {items.map((d) => (
            <button
              key={d.id}
              type="button"
              onClick={() => navigate(`/fleet/detail/${encodeURIComponent(d.sn)}`)}
              className="fleet-row grid w-full grid-cols-[12px_1.6fr_1fr_1fr_1fr] items-center gap-3 rounded-sm px-3 py-2.5 text-left"
              style={{ ['--row-color' as never]: d.isOnline ? '#00ff88' : '#525a78' }}
            >
              <span
                className="size-2.5 rounded-full"
                style={{
                  background: d.isOnline ? '#00ff88' : '#525a78',
                  boxShadow: `0 0 8px ${d.isOnline ? '#00ff88' : '#525a78'}`,
                }}
              />
              <div className="min-w-0">
                <div className="truncate font-display text-sm font-bold text-cyan-100">
                  {d.name || d.sn}
                </div>
                <div className="truncate font-mono text-[10px] text-cyan-300/55">
                  SN {d.sn} · {d.vendor || '—'}
                </div>
              </div>
              <div>
                <StatusBadge status={d.isOnline ? 'online' : 'offline'} />
              </div>
              <div className="min-w-0 font-mono text-[10px] text-cyan-300/65">
                <div className="flex items-center gap-1 text-cyan-100/85">
                  <Radio className="size-3" /> UE {d.ueCount ?? 0}
                </div>
                <div>{d.ipAddress || '—'}</div>
              </div>
              <div className="font-mono text-[10px] text-cyan-300/65">
                <div className="text-cyan-100/85">{formatTime(d.lastOnlineTime)}</div>
                <div>{d.region || '—'}</div>
              </div>
            </button>
          ))}
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
