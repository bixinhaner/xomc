import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ChevronRight, Database, Loader2, RefreshCcw, Search, Trash2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useMRFileDevices, useBatchDeleteMRFiles } from '@core/hooks/api/useMR'
import { useProductList, useProductNameResolver } from '@core/hooks/api/useProducts'
import type { MRFileDeviceItem } from '@core/services/api/mrApi'

const PAGE_SIZE = 20

/**
 * MR 文件（对齐 v1 mr/Files）。按设备聚合，10s 轮询。
 * 数据/删除全走 @core useMRFileDevices + useBatchDeleteMRFiles。点设备进入 /mr/files/:deviceSn 看单设备文件。
 */
export default function Files() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [siteName, setSiteName] = useState('')
  // #602：产品名称下拉过滤，传 product.id
  const [productId, setProductId] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      keyword: keyword.trim() || undefined,
      siteName: siteName.trim() || undefined,
      productId: productId || undefined,
    }),
    [page, keyword, siteName, productId],
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useMRFileDevices(params)
  const batchDelete = useBatchDeleteMRFiles()
  const { data: productsData } = useProductList()
  const productNameOptions = useMemo(
    () =>
      (productsData?.items ?? [])
        .map((p) => ({ label: `${p.name} (${p.tech})`, value: p.id }))
        .sort((a, b) => a.label.localeCompare(b.label, 'zh-CN')),
    [productsData]
  )
  const resolveProductName = useProductNameResolver()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const toggleRow = (sn: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(sn)) next.delete(sn)
      else next.add(sn)
      return next
    })
  }

  const handleDelete = () => {
    if (selected.size === 0) return
    const sns = [...selected]
    if (!window.confirm(`确认删除 ${sns.length} 台设备的全部 MR 文件？该操作不可恢复。`)) return
    batchDelete.mutate(sns, {
      onSuccess: () => {
        setSelected(new Set())
        void refetch()
      },
    })
  }

  return (
    <PageShell
      code="F05"
      title="MR FILES · 测量文件"
      subtitle="MRO / MRS / MRE FILE INTAKE · DEVICE AGGREGATE · 10s AUTO-REFRESH"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <SearchBox placeholder="设备 SN" value={keyword} onChange={(v) => { setKeyword(v); setPage(1) }} />
          <SearchBox placeholder="基站名称" value={siteName} onChange={(v) => { setSiteName(v); setPage(1) }} />
          <select
            className="neon-input w-48"
            value={productId}
            onChange={(e) => { setProductId(e.target.value); setPage(1) }}
          >
            <option value="">产品名称 · 全部</option>
            {productNameOptions.map((opt) => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>
          <NeonButton
            tone="danger"
            icon={<Trash2 />}
            disabled={selected.size === 0 || batchDelete.isPending}
            onClick={handleDelete}
          >
            DELETE ({selected.size})
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => void refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {isLoading ? (
        <div className="flex items-center justify-center gap-2 py-20 text-cyan-300/60">
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING FILES…</span>
        </div>
      ) : isError ? (
        <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
          FAILURE · {error instanceof Error ? error.message : '加载失败'}
        </div>
      ) : rows.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <Database className="size-10 text-cyan-400/50" />
          <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/60">
            NO MR FILES · 暂无文件
          </div>
        </div>
      ) : (
        <div className="space-y-1.5">
          {rows.map((d: MRFileDeviceItem) => {
            const checked = selected.has(d.deviceSn)
            return (
              <div
                key={d.deviceSn}
                className="fleet-row grid grid-cols-[28px_1.6fr_1.4fr_1fr_1.4fr_100px_24px] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: d.reporting ? '#00ff88' : '#00f0ff' }}
              >
                <input
                  type="checkbox"
                  checked={checked}
                  onClick={(e) => e.stopPropagation()}
                  onChange={() => toggleRow(d.deviceSn)}
                  className="size-3.5 accent-cyan-400"
                />
                <button
                  type="button"
                  onClick={() => navigate(`/mr/files/${encodeURIComponent(d.deviceSn)}`)}
                  className="min-w-0 text-left"
                >
                  <div className="truncate font-mono text-sm text-cyan-100 hover:text-glow">
                    {d.deviceSn}
                  </div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">
                    {d.siteName || '—'}
                  </div>
                </button>
                <div className="min-w-0">
                  <div className="truncate text-xs text-cyan-100/80">
                    {(() => {
                      const display = resolveProductName(d.productClass)
                      return display ? display : '—'
                    })()}
                  </div>
                  <div className="font-mono text-[10px] text-cyan-300/55">PRODUCT NAME</div>
                </div>
                <div>
                  <div
                    className="font-display text-lg font-bold leading-none"
                    style={{ color: '#00f0ff', textShadow: '0 0 6px #00f0ff' }}
                  >
                    {Number(d.fileCount ?? 0).toLocaleString()}
                  </div>
                  <div className="font-mono text-[10px] text-cyan-300/55">FILES</div>
                </div>
                <div className="font-mono text-[10px] leading-relaxed text-cyan-300/70">
                  <div>FIRST · {formatTime(d.firstCollectTime)}</div>
                  <div>LAST · {formatTime(d.lastCollectTime)}</div>
                </div>
                <div className="text-right">
                  <StatusBadge
                    status={d.reporting ? 'active' : 'off'}
                    label={d.reporting ? '上报中' : '已停止'}
                  />
                </div>
                <button
                  type="button"
                  onClick={() => navigate(`/mr/files/${encodeURIComponent(d.deviceSn)}`)}
                  className="flex items-center justify-end text-cyan-300/45 hover:text-cyan-200"
                >
                  <ChevronRight className="size-3.5" />
                </button>
              </div>
            )
          })}
        </div>
      )}

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

function SearchBox({
  placeholder,
  value,
  onChange,
}: {
  placeholder: string
  value: string
  onChange: (v: string) => void
}) {
  return (
    <div className="relative">
      <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
      <input
        className="neon-input w-44 pl-9"
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  )
}
