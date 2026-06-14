import { useMemo, useState } from 'react'
import { Search, Loader2, Ghost, RefreshCcw } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useOrphanDevices } from '@core/hooks/api/useProducts'
import type { OrphanDevice } from '@core/types/product'

const PAGE_SIZE = 20

/**
 * 孤儿设备 — 未匹配到任何产品装配件(productClass 无对应正则)的在网设备。
 * 纯只读列表(对照 v1 orphan-devices）：SN / 主机名 / OUI / productClass / 厂商 / 末次 Inform。
 */
export default function OrphanDevicesPage() {
  const [draft, setDraft] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)

  const { data, isLoading, isError, error, isFetching, refetch } = useOrphanDevices({
    page,
    pageSize: PAGE_SIZE,
    search,
  })

  const rows = useMemo<OrphanDevice[]>(() => data?.items ?? [], [data])
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const applySearch = () => {
    setSearch(draft.trim())
    setPage(1)
  }

  return (
    <PageShell
      code="F06"
      title="ORPHAN NODES · 孤儿设备"
      subtitle="UNMATCHED PRODUCT CLASS · READ-ONLY REGISTRY"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-80 pl-9"
              placeholder="SN / OUI / productClass / 厂商"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') applySearch()
              }}
            />
          </div>
          <NeonButton icon={<Search />} onClick={applySearch}>
            SEARCH
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {/* 概览统计 */}
      <div className="mb-3 grid grid-cols-2 gap-3 sm:grid-cols-3">
        <Stat label="ORPHANS" color="#ff7a1a" value={total} />
        <Stat label="PAGE" color="#00f0ff" value={page} suffix={`/ ${totalPages}`} />
        <Stat label="PAGE SIZE" color="#5b9eff" value={PAGE_SIZE} />
      </div>

      {/* 列表头 */}
      {rows.length > 0 && (
        <div className="mb-1 grid grid-cols-[1.6fr_1.4fr_0.9fr_1.3fr_1.1fr_1.4fr] items-center gap-3 px-3 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
          <span>SERIAL NO.</span>
          <span>HOSTNAME</span>
          <span>OUI</span>
          <span>PRODUCT CLASS</span>
          <span>VENDOR</span>
          <span className="text-right">LAST INFORM</span>
        </div>
      )}

      <div className="space-y-1.5">
        {isLoading ? (
          <LoadingRow />
        ) : isError ? (
          <ErrorRow message={error instanceof Error ? error.message : '未知错误'} />
        ) : rows.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-3 py-16">
            <Ghost className="size-10 text-emerald-400/60" />
            <div className="font-mono text-xs uppercase tracking-[0.2em] text-emerald-300/70">
              {search ? `NO MATCH · 无匹配「${search}」` : 'ALL MATCHED · 无孤儿设备'}
            </div>
          </div>
        ) : (
          rows.map((d) => (
            <div
              key={d.id}
              className="fleet-row grid grid-cols-[1.6fr_1.4fr_0.9fr_1.3fr_1.1fr_1.4fr] items-center gap-3 rounded-sm px-3 py-2.5"
              style={{ ['--row-color' as never]: '#ff7a1a' }}
            >
              <div className="min-w-0 truncate font-display text-sm font-bold text-cyan-100">
                {d.serialNumber}
              </div>
              <div className="min-w-0 truncate text-xs text-cyan-100/85">
                {d.deviceName || '—'}
              </div>
              <div className="min-w-0 truncate font-mono text-[11px] text-cyan-300/70">
                {d.oui || '—'}
              </div>
              <div className="min-w-0">
                {d.productClass ? (
                  <StatusBadge status="warning" label={d.productClass} />
                ) : (
                  <span className="text-cyan-300/45">—</span>
                )}
              </div>
              <div className="min-w-0 truncate text-xs text-cyan-100/85">
                {d.manufacturer || '—'}
              </div>
              <div className="text-right font-mono text-[11px] text-cyan-300/75">
                {formatTime(d.lastInformAt)}
              </div>
            </div>
          ))
        )}
      </div>

      {/* 分页 */}
      <div className="mt-4 flex items-center justify-between">
        <span className="font-mono text-[11px] text-cyan-300/55">
          PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
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
    </PageShell>
  )
}

function Stat({
  label,
  color,
  value,
  suffix,
}: {
  label: string
  color: string
  value: number
  suffix?: string
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: color }}
    >
      <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/65">
        {label}
      </div>
      <div
        className="font-display text-2xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
        {suffix && <span className="ml-1 text-sm opacity-60">{suffix}</span>}
      </div>
    </div>
  )
}

function LoadingRow() {
  return (
    <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
      <Loader2 className="size-4 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
    </div>
  )
}

function ErrorRow({ message }: { message: string }) {
  return (
    <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
      FAILURE · {message}
    </div>
  )
}
