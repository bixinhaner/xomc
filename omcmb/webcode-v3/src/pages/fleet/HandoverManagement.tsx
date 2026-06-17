import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Search, RefreshCcw, GitCompareArrows, Info } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useDeviceList } from '@core/hooks/api/useDevices'
import type { Device, DeviceLifecycle } from '@core/types/device'
import { StateGate, StatCard, Pager } from './_shared'

const PAGE_SIZE = 20

// 割接关注的生命周期态：维护中 + 已调测（在网可割接候选）。
const HANDOVER_STATES: DeviceLifecycle[] = ['maintenance', 'commissioned']

export default function FleetHandoverManagement() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      lifecycleState: HANDOVER_STATES,
      ...(keyword.trim() ? { searchText: keyword.trim() } : {}),
    }),
    [page, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useDeviceList(params)
  const items: Device[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const byLifecycle = data?.stats?.by_lifecycle
  const maintenance = byLifecycle?.maintenance ?? 0
  const commissioned = byLifecycle?.commissioned ?? 0

  return (
    <PageShell
      code="F06"
      title="HANDOVER · 割接管理"
      subtitle="IN-SERVICE CUTOVER CANDIDATES · MAINTENANCE FLEET"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-72 pl-9"
              placeholder="SN / 名称 / IP"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 flex items-start gap-2 rounded-sm border border-cyan-500/15 bg-cyan-500/[0.03] px-3 py-2 font-mono text-[10px] text-cyan-300/55">
        <Info className="mt-0.5 size-3.5 shrink-0 text-cyan-400/60" />
        割接候选机群按生命周期态（维护中 / 已调测）实时筛取自设备清单。
      </div>

      <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-3">
        <StatCard label="CANDIDATES" value={total} color="#00f0ff" />
        <StatCard label="MAINTENANCE" value={maintenance} color="#ffaa00" />
        <StatCard label="COMMISSIONED" value={commissioned} color="#00ff88" />
      </div>

      <StateGate
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={items.length === 0}
        loadingLabel="SCANNING FLEET…"
        emptyLabel="NO CUTOVER CANDIDATES"
      >
        <GlassPanel>
          <div className="divide-y divide-cyan-500/8">
            {items.map((d) => (
              <button
                key={d.id}
                type="button"
                onClick={() => navigate(`/device/detail/${encodeURIComponent(d.sn)}`)}
                className="grid w-full grid-cols-[16px_1.6fr_1fr_1fr_1fr] items-center gap-3 px-3 py-2.5 text-left transition-colors hover:bg-cyan-500/5"
              >
                <GitCompareArrows className="size-4 text-cyan-400/70" />
                <div className="min-w-0">
                  <div className="truncate font-display text-sm font-bold text-cyan-100">
                    {d.name || d.sn}
                  </div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">
                    SN {d.sn} · {d.vendor || '—'}
                  </div>
                </div>
                <div>
                  <span className="chip text-cyan-300/80">{d.lifecycleState}</span>
                </div>
                <div>
                  <StatusBadge status={d.isOnline ? 'online' : 'offline'} />
                </div>
                <div className="font-mono text-[10px] text-cyan-300/65">
                  <div className="text-cyan-100/85">{d.region || '—'}</div>
                  <div>{formatTime(d.lastOnlineTime)}</div>
                </div>
              </button>
            ))}
          </div>
        </GlassPanel>

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
