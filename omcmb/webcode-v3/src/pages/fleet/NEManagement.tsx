import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Search, RefreshCcw, Server, ChevronRight } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useNEList } from '@core/hooks/api/useNEs'
import type { NE } from '@core/types/device'
import { StateGate, Pager } from './_shared'

const PAGE_SIZE = 20

export default function FleetNEManagement() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useNEList(params)
  const items: NE[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const online = items.filter((n) => n.connStatus === 'online').length

  return (
    <PageShell
      code="F06"
      title="NE · 网元管理"
      subtitle="NETWORK ELEMENTS · eNB / gNB INVENTORY"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-72 pl-9"
              placeholder="网元名 / SN"
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
      <div className="mb-3 flex flex-wrap items-center gap-4 text-[11px] uppercase tracking-[0.18em]">
        <span className="text-cyan-300/60">
          TOTAL <span className="ml-1 font-display text-base text-cyan-200 text-glow">{total}</span>
        </span>
        <span className="text-emerald-300/80">
          ONLINE <span className="ml-1 font-display text-base text-emerald-200 text-glow">{online}</span>
        </span>
      </div>

      <StateGate
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={items.length === 0}
        loadingLabel="SYNCING NE…"
        emptyLabel="NO NETWORK ELEMENTS"
      >
        <div className="space-y-2">
          {items.map((n) => (
            <div
              key={n.id}
              className="fleet-row grid grid-cols-[16px_1.6fr_1fr_1fr_120px_40px] items-center gap-3 rounded-sm px-3 py-2.5"
              style={{
                ['--row-color' as never]: n.connStatus === 'online' ? '#00ff88' : '#525a78',
              }}
            >
              <Server className="size-4 text-cyan-400/70" />
              <div className="min-w-0">
                <div className="truncate font-display text-sm font-bold text-cyan-100">
                  {n.neName || n.sn}
                </div>
                <div className="truncate font-mono text-[10px] text-cyan-300/55">
                  SN {n.sn} · {n.neType || '—'} · {n.vendor || '—'}
                </div>
              </div>
              <div className="flex flex-col gap-1">
                <StatusBadge status={n.connStatus} />
                {n.alarmLevel !== 'none' ? (
                  <StatusBadge status={n.alarmLevel} label={`ALM · ${n.alarmLevel}`} />
                ) : (
                  <span className="font-mono text-[10px] text-cyan-300/40">NO ALARM</span>
                )}
              </div>
              <div className="min-w-0 font-mono text-[10px] text-cyan-300/65">
                <div className="text-cyan-100/85">{n.region || '—'}</div>
                <div>{n.subnet || n.site || '—'}</div>
              </div>
              <div className="text-right">
                <NeonButton
                  className="!py-1 !px-2"
                  onClick={() => navigate(`/device/detail/${encodeURIComponent(n.sn)}`)}
                >
                  DETAIL
                </NeonButton>
              </div>
              <ChevronRight className="size-3.5 text-cyan-300/40" />
            </div>
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
