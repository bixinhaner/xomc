import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  Loader2,
  Boxes,
  RefreshCcw,
  ChevronRight,
  Cpu,
  ShieldCheck,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useProductList, useMatchOrder } from '@core/hooks/api/useProducts'
import type { Product, ProductListFilter } from '@core/types/product'

const SORT_TAIL = Number.MAX_SAFE_INTEGER

interface PatternStat {
  minSortOrder: number
  total: number
  active: number
}

/**
 * 产品装配件清单（对照 v1 product/products）。
 * 产品 = ProductRegistry 路由 productClass 的装配单元，绑定参数模型 / 指标平台 / 告警网元类型。
 * 行按全局匹配顺序（match-order 的最小 sortOrder）升序排列；点击进入详情页查看正则与覆盖。
 */
export default function ProductsPage() {
  const navigate = useNavigate()
  const [draft, setDraft] = useState('')
  const [filter, setFilter] = useState<ProductListFilter>({})

  const { data, isLoading, isError, error, isFetching, refetch } = useProductList(filter)
  const { data: matchOrderData } = useMatchOrder()

  const items = useMemo<Product[]>(() => data?.items ?? [], [data])

  const patternStats = useMemo(() => {
    const stats = new Map<string, PatternStat>()
    for (const row of matchOrderData?.items ?? []) {
      const cur = stats.get(row.productId)
      if (!cur) {
        stats.set(row.productId, {
          minSortOrder: row.sortOrder,
          total: 1,
          active: row.isActive ? 1 : 0,
        })
      } else {
        cur.minSortOrder = Math.min(cur.minSortOrder, row.sortOrder)
        cur.total += 1
        if (row.isActive) cur.active += 1
      }
    }
    return stats
  }, [matchOrderData])

  const rows = useMemo(() => {
    return [...items].sort((a, b) => {
      const sa = patternStats.get(a.id)?.minSortOrder ?? SORT_TAIL
      const sb = patternStats.get(b.id)?.minSortOrder ?? SORT_TAIL
      return sa - sb
    })
  }, [items, patternStats])

  const builtinCount = useMemo(() => rows.filter((r) => r.isBuiltin).length, [rows])
  const deviceTotal = useMemo(() => rows.reduce((acc, r) => acc + (r.deviceCount || 0), 0), [rows])

  const applySearch = () => {
    setFilter((f) => ({ ...f, keyword: draft.trim() || undefined }))
  }

  return (
    <PageShell
      code="F02"
      title="PRODUCT REGISTRY · 产品装配件"
      subtitle="PRODUCT CLASS ROUTING · MODEL / KPI / ALARM ASSEMBLY"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-72 pl-9"
              placeholder="产品名 / 厂商 / 制式"
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
      <div className="mb-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Stat label="PRODUCTS" color="#00f0ff" value={rows.length} icon={<Boxes className="size-3.5" />} />
        <Stat label="BUILTIN" color="#a855f7" value={builtinCount} icon={<ShieldCheck className="size-3.5" />} />
        <Stat label="CUSTOM" color="#00ff88" value={rows.length - builtinCount} />
        <Stat label="BOUND DEVICES" color="#5b9eff" value={deviceTotal} icon={<Cpu className="size-3.5" />} />
      </div>

      {/* 列表头 */}
      {rows.length > 0 && (
        <div className="mb-1 grid grid-cols-[1.8fr_1fr_0.7fr_1.4fr_1.1fr_0.8fr_0.7fr_60px] items-center gap-3 px-3 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
          <span>PRODUCT · 装配件</span>
          <span>VENDOR</span>
          <span>TECH</span>
          <span>PARAM MODEL</span>
          <span>INDICATOR · ALARM</span>
          <span className="text-right">PATTERNS</span>
          <span className="text-right">DEVICES</span>
          <span className="text-right">VIEW</span>
        </div>
      )}

      <div className="space-y-1.5">
        {isLoading ? (
          <LoadingRow />
        ) : isError ? (
          <ErrorRow message={error instanceof Error ? error.message : '未知错误'} />
        ) : rows.length === 0 ? (
          <EmptyRow keyword={filter.keyword} />
        ) : (
          rows.map((p) => {
            const stat = patternStats.get(p.id)
            return (
              <button
                key={p.id}
                type="button"
                onClick={() => navigate(`/product/products/${encodeURIComponent(p.id)}`)}
                className="fleet-row grid w-full grid-cols-[1.8fr_1fr_0.7fr_1.4fr_1.1fr_0.8fr_0.7fr_60px] items-center gap-3 rounded-sm px-3 py-2.5 text-left"
                style={{ ['--row-color' as never]: p.isBuiltin ? '#a855f7' : '#00ff88' }}
              >
                <div className="min-w-0">
                  <div className="flex items-center gap-1.5 font-display text-sm font-bold text-cyan-100">
                    <span className="truncate">{p.name}</span>
                    {p.isBuiltin && (
                      <StatusBadge status="active" label="BUILTIN" className="scale-90" />
                    )}
                  </div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">
                    {p.radioModes || p.description || '—'}
                  </div>
                </div>
                <div className="min-w-0 truncate text-xs text-cyan-100/85">{p.vendor || '—'}</div>
                <div>
                  <span className="chip text-cyan-200">{(p.tech || '—').toUpperCase()}</span>
                </div>
                <div className="min-w-0 truncate text-xs text-cyan-100/85">
                  {p.paramModelName || <span className="text-cyan-300/45">未绑定</span>}
                </div>
                <div className="min-w-0">
                  <div className="truncate text-[11px] text-cyan-100/80">{p.indicatorPlatform || '—'}</div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">{p.alarmNeType || '—'}</div>
                </div>
                <div className="text-right font-mono text-[11px]">
                  <span className="text-cyan-200">{stat?.active ?? (p.patterns?.length || 0)}</span>
                  <span className="text-cyan-300/45"> / {stat?.total ?? (p.patterns?.length || 0)}</span>
                </div>
                <div className="text-right font-display text-sm font-bold text-cyan-200">
                  {p.deviceCount ?? 0}
                </div>
                <div className="flex justify-end">
                  <ChevronRight className="size-4 text-cyan-300/60" />
                </div>
              </button>
            )
          })
        )}
      </div>
    </PageShell>
  )
}

function Stat({
  label,
  color,
  value,
  icon,
}: {
  label: string
  color: string
  value: number
  icon?: React.ReactNode
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: color }}
    >
      <div className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/65">
        {icon}
        {label}
      </div>
      <div
        className="font-display text-2xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
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

function EmptyRow({ keyword }: { keyword?: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16">
      <Boxes className="size-10 text-cyan-400/50" />
      <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
        {keyword ? `NO MATCH · 无匹配「${keyword}」` : 'NO PRODUCTS · 暂无装配件'}
      </div>
    </div>
  )
}
