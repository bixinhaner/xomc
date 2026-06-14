import { useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  Search,
  Loader2,
  Boxes,
  ArrowLeft,
  ChevronRight,
  RefreshCcw,
  GitCompareArrows,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useParamModelList, useParamMappings } from '@core/hooks/api/useParamModels'
import type { ParamModel, ParamMapping } from '@core/types/paramModel'

const PAGE_SIZE = 30

/**
 * 参数模型库（对照 v1 product/param-model）。
 * drill-down 主从：模型清单 → 点击模型名进入其 mapping 列表（标准路径 ↔ 私有路径双向翻译表）。
 * 选中模型走 URL ?name=<模型名>，刷新/分享保留二级页。
 */
export default function ParamModelPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const selectedName = searchParams.get('name') || undefined

  const selectModel = (name?: string) => {
    const params = new URLSearchParams(searchParams)
    if (name) params.set('name', name)
    else params.delete('name')
    setSearchParams(params, { replace: false })
  }

  return selectedName ? (
    <MappingsView name={selectedName} onBack={() => selectModel(undefined)} />
  ) : (
    <ModelsView onSelect={selectModel} />
  )
}

// ── 一级：模型清单 ──────────────────────────────────────────────────────
function ModelsView({ onSelect }: { onSelect: (name: string) => void }) {
  const [draft, setDraft] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)

  const { data, isLoading, isError, error, isFetching, refetch } = useParamModelList()

  const all = useMemo<ParamModel[]>(() => {
    // 隐藏无加载源(source=unknown)的孤儿模型（对照 v1 双保险）。
    const raw = (data?.items ?? []).filter((m) => (m.source ?? 'unknown') !== 'unknown')
    const k = keyword.trim().toLowerCase()
    if (!k) return raw
    return raw.filter((m) => `${m.name} ${m.description ?? ''}`.toLowerCase().includes(k))
  }, [data, keyword])

  const total = all.length
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const rows = useMemo(() => all.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE), [all, page])

  const paramTotal = useMemo(() => all.reduce((acc, m) => acc + (m.totalParams || 0), 0), [all])
  const activeCount = useMemo(() => all.filter((m) => m.isActive).length, [all])

  const applySearch = () => {
    setKeyword(draft.trim())
    setPage(1)
  }

  return (
    <PageShell
      code="F02"
      title="PARAM MODEL LIBRARY · 参数模型库"
      subtitle="STANDARD ↔ PRIVATE PATH TRANSLATION DICTIONARY"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-80 pl-9"
              placeholder="模型名 / 描述"
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
      <div className="mb-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Stat label="MODELS" color="#00f0ff" value={total} />
        <Stat label="ACTIVE" color="#00ff88" value={activeCount} />
        <Stat label="INACTIVE" color="#525a78" value={total - activeCount} />
        <Stat label="TOTAL PARAMS" color="#5b9eff" value={paramTotal} />
      </div>

      {rows.length > 0 && (
        <div className="mb-1 grid grid-cols-[1.6fr_2fr_0.7fr_0.7fr_0.7fr_0.7fr_60px] items-center gap-3 px-3 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
          <span>MODEL · 模型</span>
          <span>LOADED FROM</span>
          <span className="text-right">ENTRIES</span>
          <span className="text-right">OBJECTS</span>
          <span className="text-right">PARAMS</span>
          <span className="text-right">STATE</span>
          <span className="text-right">OPEN</span>
        </div>
      )}

      <div className="space-y-1.5">
        {isLoading ? (
          <LoadingRow />
        ) : isError ? (
          <ErrorRow message={error instanceof Error ? error.message : '未知错误'} />
        ) : rows.length === 0 ? (
          <EmptyRow keyword={keyword} icon={<Boxes className="size-10 text-cyan-400/50" />} label="NO MODELS · 暂无参数模型" />
        ) : (
          rows.map((m) => (
            <button
              key={m.id}
              type="button"
              onClick={() => onSelect(m.name)}
              className="fleet-row grid w-full grid-cols-[1.6fr_2fr_0.7fr_0.7fr_0.7fr_0.7fr_60px] items-center gap-3 rounded-sm px-3 py-2.5 text-left"
              style={{ ['--row-color' as never]: m.isActive ? '#00ff88' : '#525a78' }}
            >
              <div className="min-w-0">
                <div className="truncate font-display text-sm font-bold text-cyan-100">{m.name}</div>
                {m.description && (
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">{m.description}</div>
                )}
              </div>
              <code className="min-w-0 truncate font-mono text-[11px] text-cyan-300/70" title={m.loadedFrom}>
                {m.loadedFrom || '—'}
              </code>
              <span className="text-right font-mono text-[11px] text-cyan-300/75">{m.totalEntries}</span>
              <span className="text-right font-mono text-[11px] text-cyan-300/75">{m.totalObjects}</span>
              <span className="text-right font-display text-sm font-bold text-cyan-200">{m.totalParams}</span>
              <div className="flex justify-end">
                <StatusBadge
                  status={m.isActive ? 'online' : 'offline'}
                  label={m.isActive ? '生效' : '停用'}
                  className="scale-90"
                />
              </div>
              <div className="flex justify-end">
                <ChevronRight className="size-4 text-cyan-300/60" />
              </div>
            </button>
          ))
        )}
      </div>

      <Pager page={page} totalPages={totalPages} total={total} onPrev={() => setPage((p) => Math.max(1, p - 1))} onNext={() => setPage((p) => Math.min(totalPages, p + 1))} />
    </PageShell>
  )
}

// ── 二级：mapping 列表 ──────────────────────────────────────────────────
function MappingsView({ name, onBack }: { name: string; onBack: () => void }) {
  const [draft, setDraft] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)

  const { data, isLoading, isError, error, isFetching } = useParamMappings(name)

  const all = useMemo<ParamMapping[]>(() => {
    const raw = data?.items ?? []
    const k = keyword.trim().toLowerCase()
    if (!k) return raw
    return raw.filter((m) =>
      `${m.standardPath} ${m.privatePath} ${m.dataType}`.toLowerCase().includes(k),
    )
  }, [data, keyword])

  const total = all.length
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const rows = useMemo(() => all.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE), [all, page])

  const applySearch = () => {
    setKeyword(draft.trim())
    setPage(1)
  }

  return (
    <PageShell
      code="F02"
      title={`PARAM MODEL · ${name}`}
      subtitle="STANDARD ↔ PRIVATE MAPPING TABLE"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={onBack}>
            BACK
          </NeonButton>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-80 pl-9"
              placeholder="标准路径 / 私有路径"
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
        </>
      }
    >
      {rows.length > 0 && (
        <div className="mb-1 grid grid-cols-[2fr_2fr_0.8fr_0.9fr_0.8fr_0.7fr] items-center gap-3 px-3 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
          <span>STANDARD PATH</span>
          <span>PRIVATE PATH</span>
          <span>ENTRY</span>
          <span>ACCESS</span>
          <span>TYPE</span>
          <span className="text-right">STATE</span>
        </div>
      )}

      <div className="space-y-1.5">
        {isLoading ? (
          <LoadingRow />
        ) : isError ? (
          <ErrorRow message={error instanceof Error ? error.message : '未知错误'} />
        ) : rows.length === 0 ? (
          <EmptyRow keyword={keyword} icon={<GitCompareArrows className="size-10 text-cyan-400/50" />} label="NO MAPPINGS · 该模型无映射条目" />
        ) : (
          rows.map((m) => (
            <div
              key={m.id}
              className="fleet-row grid grid-cols-[2fr_2fr_0.8fr_0.9fr_0.8fr_0.7fr] items-center gap-3 rounded-sm px-3 py-2.5"
              style={{ ['--row-color' as never]: m.isActive ? '#00f0ff' : '#525a78' }}
            >
              <code className="min-w-0 truncate font-mono text-[11px] text-cyan-100/90" title={m.standardPath}>
                {m.standardPath}
              </code>
              <code className="min-w-0 truncate font-mono text-[11px] text-cyan-200/80" title={m.privatePath}>
                {m.privatePath}
              </code>
              <span className="font-mono text-[11px] text-cyan-300/75">{m.entryType || '—'}</span>
              <span className="font-mono text-[11px] text-cyan-300/75">{m.access || '—'}</span>
              <span className="font-mono text-[11px] text-cyan-100/80">{m.dataType || '—'}</span>
              <div className="flex justify-end">
                <StatusBadge
                  status={m.isActive ? 'online' : 'offline'}
                  label={m.isActive ? '生效' : '停用'}
                  className="scale-90"
                />
              </div>
            </div>
          ))
        )}
      </div>

      <Pager page={page} totalPages={totalPages} total={total} onPrev={() => setPage((p) => Math.max(1, p - 1))} onNext={() => setPage((p) => Math.min(totalPages, p + 1))} />
    </PageShell>
  )
}

function Stat({ label, color, value }: { label: string; color: string; value: number }) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: color }}
    >
      <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/65">{label}</div>
      <div
        className="font-display text-2xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}

function Pager({
  page,
  totalPages,
  total,
  onPrev,
  onNext,
}: {
  page: number
  totalPages: number
  total: number
  onPrev: () => void
  onNext: () => void
}) {
  return (
    <div className="mt-4 flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <NeonButton onClick={onPrev} disabled={page <= 1}>
          ◂ PREV
        </NeonButton>
        <NeonButton onClick={onNext} disabled={page >= totalPages}>
          NEXT ▸
        </NeonButton>
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

function EmptyRow({
  keyword,
  icon,
  label,
}: {
  keyword: string
  icon: React.ReactNode
  label: string
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16">
      {icon}
      <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
        {keyword ? `NO MATCH · 无匹配「${keyword}」` : label}
      </div>
    </div>
  )
}
