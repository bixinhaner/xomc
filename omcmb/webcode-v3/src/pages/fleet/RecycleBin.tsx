import { useCallback, useMemo, useState } from 'react'
import {
  Search,
  RefreshCcw,
  Trash2,
  RotateCcw,
  Check,
  Loader2,
  Recycle,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatTime } from '@/lib/format'
import {
  useRecycleBinList,
  useRestoreDevices,
  usePermanentDeleteDevices,
} from '@core/hooks/api/useDevices'
import type { Device } from '@core/types/device'
import { StateGate, Pager, ErrorBlock } from './_shared'

const PAGE_SIZE = 20

export default function FleetRecycleBin() {
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [opError, setOpError] = useState<unknown>(null)

  const params = useMemo(
    () => ({ page, pageSize: PAGE_SIZE, ...(search.trim() ? { search: search.trim() } : {}) }),
    [page, search]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useRecycleBinList(params)
  const restore = useRestoreDevices()
  const purge = usePermanentDeleteDevices()

  const items: Device[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const toggle = useCallback((id: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }, [])

  const ids = useMemo(() => Array.from(selected), [selected])
  const busy = restore.isPending || purge.isPending

  const run = useCallback(
    (mut: { mutateAsync: (i: string[]) => Promise<unknown> }) => {
      if (ids.length === 0) return
      setOpError(null)
      mut
        .mutateAsync(ids)
        .then(() => {
          setSelected(new Set())
          void refetch()
        })
        .catch((e) => setOpError(e))
    },
    [ids, refetch]
  )

  return (
    <PageShell
      code="F06"
      title="RECYCLE BIN · 回收站"
      subtitle="DELETED UNITS · RESTORE OR PURGE"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-72 pl-9"
              placeholder="SN / 名称"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value)
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
      {ids.length > 0 && (
        <div className="mb-2 flex flex-wrap items-center gap-2 rounded-sm border border-cyan-500/25 bg-cyan-500/[0.05] px-3 py-2">
          <span className="font-mono text-[11px] text-cyan-200">已选 {ids.length} 条</span>
          <NeonButton icon={<RotateCcw />} disabled={busy} onClick={() => run(restore)}>
            还原
          </NeonButton>
          <NeonButton tone="danger" icon={<Trash2 />} disabled={busy} onClick={() => run(purge)}>
            彻底删除
          </NeonButton>
          <NeonButton onClick={() => setSelected(new Set())}>清空选择</NeonButton>
          {busy && <Loader2 className="size-4 animate-spin text-cyan-300/70" />}
        </div>
      )}

      {opError ? (
        <div className="mb-2">
          <ErrorBlock error={opError} />
        </div>
      ) : null}

      <StateGate
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={items.length === 0}
        loadingLabel="LOADING TRASH…"
        emptyLabel="RECYCLE BIN EMPTY"
      >
        <div className="space-y-2">
          {items.map((d) => {
            const checked = selected.has(d.id)
            return (
              <div
                key={d.id}
                className="fleet-row grid grid-cols-[28px_16px_1.6fr_1fr_1fr_1fr] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: '#525a78' }}
              >
                <button
                  type="button"
                  onClick={() => toggle(d.id)}
                  className={`flex size-4 items-center justify-center rounded-sm border ${
                    checked ? 'border-cyan-400 bg-cyan-400/20' : 'border-cyan-500/40'
                  }`}
                  aria-label="select row"
                >
                  {checked && <Check className="size-3 text-cyan-200" />}
                </button>
                <Recycle className="size-4 text-cyan-400/50" />
                <div className="min-w-0">
                  <div className="truncate font-display text-sm font-bold text-cyan-100/85">
                    {d.name || d.sn}
                  </div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">
                    SN {d.sn} · {d.vendor || '—'}
                  </div>
                </div>
                <div className="font-mono text-[10px] text-cyan-300/65">
                  <div className="text-cyan-100/85">{d.carrier || '—'}</div>
                  <div>{d.productClass || '—'}</div>
                </div>
                <div className="font-mono text-[10px] text-cyan-300/65">
                  <div className="text-cyan-100/85">{d.deletedBy || '—'}</div>
                  <div>{d.region || '—'}</div>
                </div>
                <div className="font-mono text-[10px] text-rose-300/70">
                  {formatTime(d.deletedAt)}
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
