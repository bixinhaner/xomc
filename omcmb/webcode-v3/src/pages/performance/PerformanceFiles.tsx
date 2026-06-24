import { useMemo, useState } from 'react'
import { Database, FileWarning, HardDrive, Loader2, RefreshCcw, Search, Trash2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { usePMFileDevices, useBatchDeletePMFiles } from '@core/hooks/api/usePerformance'
import { useProductList, useProductNameResolver } from '@core/hooks/api/useProducts'
import type { PMFileDeviceItem } from '@core/services/api/pmApi'

/**
 * F03 · 性能文件（performance/files）
 * 按设备聚合的 PM 文件视图（真实 usePMFileDevices，10s 轮询）。
 * 选中设备 → 批量删除（真实 useBatchDeletePMFiles，PG + MinIO）。
 */

const PAGE_SIZE = 20

export default function PerformanceFiles() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  // #602：产品名称下拉过滤，传 product.id
  const [productId, setProductId] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(productId ? { productId } : {}),
    }),
    [page, keyword, productId],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = usePMFileDevices(params)
  const batchDelete = useBatchDeletePMFiles()
  const { data: productsData } = useProductList()
  const productNameOptions = useMemo(
    () =>
      (productsData?.items ?? [])
        .map((p) => ({ label: `${p.name} (${p.tech})`, value: p.id }))
        .sort((a, b) => a.label.localeCompare(b.label, 'zh-CN')),
    [productsData]
  )
  const resolveProductName = useProductNameResolver()

  const rows: PMFileDeviceItem[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const reportingCount = useMemo(() => rows.filter((r) => r.reporting).length, [rows])
  const pageFileCount = useMemo(() => rows.reduce((s, r) => s + (r.fileCount || 0), 0), [rows])

  const allSelected = rows.length > 0 && rows.every((r) => selected.has(r.deviceSn))
  const toggleAll = () => {
    setSelected((prev) => {
      if (rows.every((r) => prev.has(r.deviceSn))) return new Set()
      return new Set(rows.map((r) => r.deviceSn))
    })
  }
  const toggleOne = (sn: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(sn)) next.delete(sn)
      else next.add(sn)
      return next
    })
  }

  const handleDelete = (sns: string[]) => {
    if (sns.length === 0) return
    batchDelete.mutate(sns, {
      onSuccess: () => {
        setSelected((prev) => {
          const next = new Set(prev)
          for (const sn of sns) next.delete(sn)
          return next
        })
        void refetch()
      },
    })
  }

  return (
    <PageShell
      code="F03"
      title="PM FILES · 性能文件"
      subtitle="PER-DEVICE PM FILE AGGREGATE · MinIO + PG"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="设备 SN / 基站名搜索"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <select
            className="neon-input w-56"
            value={productId}
            onChange={(e) => { setProductId(e.target.value); setPage(1) }}
          >
            <option value="">产品名称 · 全部</option>
            {productNameOptions.map((opt) => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="DEVICES · 上报设备" value={String(rows.length)} color="#00f0ff" icon={<HardDrive className="size-4" />} />
        <Stat label="REPORTING · 上报中" value={String(reportingCount)} color="#00ff88" icon={<Database className="size-4" />} />
        <Stat label="PAGE FILES · 本页文件" value={pageFileCount.toLocaleString()} color="#a855f7" icon={<Database className="size-4" />} />
        <Stat label="TOTAL · 设备总数" value={total.toLocaleString()} color="#ffaa00" icon={<HardDrive className="size-4" />} />
      </div>

      <GlassPanel strong title="PM FILE MANIFEST · 文件清单（按设备）" meta={`${total} DEVICES`}>
        <div className="p-3">
          {selected.size > 0 ? (
            <div className="mb-2 flex justify-end">
              <NeonButton
                tone="danger"
                icon={<Trash2 />}
                disabled={batchDelete.isPending}
                onClick={() => handleDelete([...selected])}
              >
                删除选中设备文件 ({selected.size})
              </NeonButton>
            </div>
          ) : null}

          <div className="grid grid-cols-[28px_1.4fr_1.6fr_1fr_90px_1.2fr_90px] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            <input type="checkbox" checked={allSelected} onChange={toggleAll} className="accent-cyan-400" aria-label="select all" />
            <span>SN · 序列号</span>
            <span>SITE · 基站名</span>
            <span>PRODUCT NAME · 产品名称</span>
            <span className="text-right">FILES</span>
            <span>LAST COLLECT</span>
            <span>STATUS</span>
          </div>

          <div className="mt-1.5 space-y-1">
            {isLoading ? (
              <Loading text="SYNCING PM FILES…" />
            ) : isError ? (
              <ErrorBox text={`LOAD FAILED · ${error instanceof Error ? error.message : '未知错误'}`} />
            ) : rows.length === 0 ? (
              <div className="flex flex-col items-center justify-center gap-3 py-16">
                <FileWarning className="size-10 text-cyan-400/40" />
                <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
                  NO PM FILE · 暂无性能文件
                </div>
              </div>
            ) : (
              rows.map((r) => (
                <div
                  key={r.deviceSn}
                  className="grid grid-cols-[28px_1.4fr_1.6fr_1fr_90px_1.2fr_90px] items-center gap-3 rounded-sm px-3 py-2 transition-colors hover:bg-cyan-500/5"
                >
                  <input
                    type="checkbox"
                    checked={selected.has(r.deviceSn)}
                    onChange={() => toggleOne(r.deviceSn)}
                    className="accent-cyan-400"
                    aria-label={`select ${r.deviceSn}`}
                  />
                  <span className="truncate font-mono text-[11px] text-cyan-100" title={r.deviceSn}>
                    {r.deviceSn}
                  </span>
                  <span className="truncate text-[13px] text-cyan-100/85" title={r.siteName}>
                    {r.siteName || '—'}
                  </span>
                  <span className="truncate font-mono text-[11px] text-cyan-300/70" title={r.productClass}>
                    {(() => {
                      const display = resolveProductName(r.productClass)
                      return display ? display : '—'
                    })()}
                  </span>
                  <span className="text-right font-display text-sm font-bold text-cyan-200">
                    {(r.fileCount ?? 0).toLocaleString()}
                  </span>
                  <span className="font-mono text-[11px] text-cyan-300/65">
                    {r.lastCollectTime ? formatTime(r.lastCollectTime) : '—'}
                  </span>
                  <StatusBadge
                    status={r.reporting ? 'ok' : 'off'}
                    label={r.reporting ? '上报中' : '静默'}
                    className="w-fit"
                  />
                </div>
              ))
            )}
          </div>

          <div className="mt-3 flex items-center justify-between">
            <span className="flex items-center gap-2 font-mono text-[11px] text-cyan-300/55">
              {batchDelete.isPending ? <Loader2 className="size-3 animate-spin" /> : null}
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
