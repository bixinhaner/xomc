import { useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  Search,
  Loader2,
  BellRing,
  ArrowLeft,
  ChevronRight,
  RefreshCcw,
  ListChecks,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useAlarmNeTypeStats, useAlarmDefinitionList } from '@core/hooks/api/useAlarmDefinitions'
import type {
  AlarmNeTypeStat,
  AlarmDefinition,
  AlarmDefinitionFilter,
} from '@core/types/alarmDefinition'

const EMPTY_LOADED_FROM = '__empty__'
const PAGE_SIZE = 20

// 严重级别 code → HUD 配色（同时兼容 1-4 短码与 31001-31004 长码）。
const SEV_COLOR: Record<number, string> = {
  1: '#ff2d6f',
  31001: '#ff2d6f',
  2: '#ff7a1a',
  31002: '#ff7a1a',
  3: '#ffd400',
  31003: '#ffd400',
  4: '#5b9eff',
  31004: '#5b9eff',
}

/**
 * 告警库（对照 v1 product/alarm-library）。
 * drill-down 主从：一级按 (网元类型, 加载源) 聚合 → 点击进入该网元类型告警定义明细。
 * 选中走 URL ?neType=ENB&loadedFrom=...，刷新保留二级页。
 */
export default function AlarmLibraryPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const selectedNeType = searchParams.get('neType') || undefined
  const selectedLoadedFrom = (() => {
    const v = searchParams.get('loadedFrom')
    if (v === null) return undefined
    return v === EMPTY_LOADED_FROM ? '' : v
  })()

  const setSelected = (neType?: string, loadedFrom?: string) => {
    const params = new URLSearchParams(searchParams)
    if (neType) {
      params.set('neType', neType)
      params.set('loadedFrom', loadedFrom === '' ? EMPTY_LOADED_FROM : (loadedFrom ?? ''))
    } else {
      params.delete('neType')
      params.delete('loadedFrom')
    }
    setSearchParams(params, { replace: false })
  }

  return selectedNeType ? (
    <DefinitionsView
      neType={selectedNeType}
      loadedFrom={selectedLoadedFrom}
      onBack={() => setSelected(undefined)}
    />
  ) : (
    <NeTypesView onSelect={(s) => setSelected(s.neType, s.loadedFrom)} />
  )
}

// ── 一级：网元类型聚合 ──────────────────────────────────────────────────
function NeTypesView({ onSelect }: { onSelect: (stat: AlarmNeTypeStat) => void }) {
  const [draft, setDraft] = useState('')
  const [keyword, setKeyword] = useState('')

  const { data, isLoading, isError, error, isFetching, refetch } = useAlarmNeTypeStats()

  const all = useMemo<AlarmNeTypeStat[]>(() => data?.items ?? [], [data])
  const rows = useMemo(() => {
    const k = keyword.trim().toLowerCase()
    if (!k) return all
    return all.filter((r) => r.neType.toLowerCase().includes(k))
  }, [all, keyword])

  const defTotal = useMemo(() => all.reduce((acc, r) => acc + (r.total || 0), 0), [all])
  const critTotal = useMemo(() => all.reduce((acc, r) => acc + (r.criticalCnt || 0), 0), [all])

  const applySearch = () => setKeyword(draft.trim())

  return (
    <PageShell
      code="F04"
      title="ALARM LIBRARY · 告警库"
      subtitle="PER-NE-TYPE ALARM DEFINITION CATALOG"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-72 pl-9"
              placeholder="网元类型"
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
      <div className="mb-3 grid grid-cols-3 gap-3">
        <Stat label="NE TYPES" color="#00f0ff" value={all.length} />
        <Stat label="TOTAL DEFS" color="#00ff88" value={defTotal} />
        <Stat label="CRITICAL DEFS" color="#ff2d6f" value={critTotal} />
      </div>

      {rows.length > 0 && (
        <div className="mb-1 grid grid-cols-[1fr_2.2fr_0.7fr_0.7fr_0.7fr_0.7fr_0.7fr_60px] items-center gap-3 px-3 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
          <span>NE TYPE</span>
          <span>LOADED FROM</span>
          <span className="text-right">TOTAL</span>
          <span className="text-right">CRIT</span>
          <span className="text-right">MAJ</span>
          <span className="text-right">MIN</span>
          <span className="text-right">WRN</span>
          <span className="text-right">OPEN</span>
        </div>
      )}

      <div className="space-y-1.5">
        {isLoading ? (
          <LoadingRow />
        ) : isError ? (
          <ErrorRow message={error instanceof Error ? error.message : '未知错误'} />
        ) : rows.length === 0 ? (
          <EmptyRow keyword={keyword} icon={<BellRing className="size-10 text-cyan-400/50" />} label="NO NE TYPES · 暂无告警库" />
        ) : (
          rows.map((r) => (
            <button
              key={`${r.neType}__${r.loadedFrom}`}
              type="button"
              onClick={() => onSelect(r)}
              className="fleet-row grid w-full grid-cols-[1fr_2.2fr_0.7fr_0.7fr_0.7fr_0.7fr_0.7fr_60px] items-center gap-3 rounded-sm px-3 py-2.5 text-left"
              style={{ ['--row-color' as never]: r.criticalCnt > 0 ? '#ff2d6f' : '#00f0ff' }}
            >
              <div className="min-w-0 truncate font-display text-sm font-bold text-cyan-100">
                {r.neType}
              </div>
              {r.loadedFrom ? (
                <code className="min-w-0 truncate font-mono text-[11px] text-cyan-300/70" title={r.loadedFrom}>
                  {r.loadedFrom}
                </code>
              ) : (
                <span><StatusBadge status="warning" label="手工新增" className="scale-90" /></span>
              )}
              <span className="text-right font-display text-sm font-bold text-cyan-200">{r.total}</span>
              <SevCell value={r.criticalCnt} color={SEV_COLOR[1]} />
              <SevCell value={r.majorCnt} color={SEV_COLOR[2]} />
              <SevCell value={r.minorCnt} color={SEV_COLOR[3]} />
              <SevCell value={r.warningCnt} color={SEV_COLOR[4]} />
              <div className="flex justify-end">
                <ChevronRight className="size-4 text-cyan-300/60" />
              </div>
            </button>
          ))
        )}
      </div>
    </PageShell>
  )
}

// ── 二级：告警定义明细 ──────────────────────────────────────────────────
function DefinitionsView({
  neType,
  loadedFrom,
  onBack,
}: {
  neType: string
  loadedFrom?: string
  onBack: () => void
}) {
  const [draft, setDraft] = useState('')
  const [keyword, setKeyword] = useState('')
  const [severityCode, setSeverityCode] = useState<number | ''>('')
  const [page, setPage] = useState(1)

  const filter = useMemo<AlarmDefinitionFilter>(
    () => ({
      neType,
      loadedFrom,
      page,
      pageSize: PAGE_SIZE,
      ...(keyword ? { keyword } : {}),
      ...(severityCode !== '' ? { severityCode } : {}),
    }),
    [neType, loadedFrom, page, keyword, severityCode],
  )

  const { data, isLoading, isError, error, isFetching } = useAlarmDefinitionList(filter)

  // 严重级别下拉选项：独立全量查询（不带 severityCode 过滤，避免选中后下拉收缩）。
  const severitySourceFilter = useMemo<AlarmDefinitionFilter>(
    () => ({ neType, loadedFrom, page: 1, pageSize: 200 }),
    [neType, loadedFrom],
  )
  const { data: severitySource } = useAlarmDefinitionList(severitySourceFilter)

  const severityOptions = useMemo(() => {
    const byCode = new Map<number, string>()
    for (const row of severitySource?.items ?? []) {
      if (!byCode.has(row.severityCode)) byCode.set(row.severityCode, row.severityName ?? '')
    }
    return Array.from(byCode.entries())
      .map(([code, label]) => ({ code, label: label ? `${code} · ${label}` : String(code) }))
      .sort((a, b) => a.code - b.code)
  }, [severitySource])

  const rows = useMemo<AlarmDefinition[]>(() => data?.items ?? [], [data])
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const applySearch = () => {
    setKeyword(draft.trim())
    setPage(1)
  }

  return (
    <PageShell
      code="F04"
      title={`ALARM DEFS · ${neType}`}
      subtitle="ALARM DEFINITION DETAIL"
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
              className="neon-input w-72 pl-9"
              placeholder="标识 / 名称"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') applySearch()
              }}
            />
          </div>
          <select
            className="neon-input"
            value={severityCode}
            onChange={(e) => {
              setSeverityCode(e.target.value === '' ? '' : Number(e.target.value))
              setPage(1)
            }}
          >
            <option value="" className="bg-[#03050d]">
              全部级别
            </option>
            {severityOptions.map((o) => (
              <option key={o.code} value={o.code} className="bg-[#03050d]">
                {o.label}
              </option>
            ))}
          </select>
          <NeonButton icon={<Search />} onClick={applySearch}>
            SEARCH
          </NeonButton>
        </>
      }
    >
      {rows.length > 0 && (
        <div className="mb-1 grid grid-cols-[1fr_1.6fr_2fr_1.1fr_1fr_0.7fr] items-center gap-3 px-3 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
          <span>IDENTIFIER</span>
          <span>CN NAME</span>
          <span>EN NAME</span>
          <span>SEVERITY</span>
          <span>EVENT TYPE</span>
          <span className="text-right">VISIBLE</span>
        </div>
      )}

      <div className="space-y-1.5">
        {isLoading ? (
          <LoadingRow />
        ) : isError ? (
          <ErrorRow message={error instanceof Error ? error.message : '未知错误'} />
        ) : rows.length === 0 ? (
          <EmptyRow keyword={keyword} icon={<ListChecks className="size-10 text-cyan-400/50" />} label="NO DEFINITIONS · 该网元类型无告警定义" />
        ) : (
          rows.map((d) => {
            const color = SEV_COLOR[d.severityCode] ?? '#6b86b6'
            return (
              <div
                key={d.id}
                className="fleet-row grid grid-cols-[1fr_1.6fr_2fr_1.1fr_1fr_0.7fr] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: color }}
              >
                <code className="min-w-0 truncate font-mono text-[11px] text-cyan-200/90">{d.identifier}</code>
                <div className="min-w-0 truncate text-xs text-cyan-100/85" title={d.cnName}>
                  {d.cnName || '—'}
                </div>
                <div className="min-w-0 truncate font-mono text-[11px] text-cyan-300/65" title={d.enName}>
                  {d.enName || '—'}
                </div>
                <div className="min-w-0">
                  <span className="chip" style={{ color }}>
                    {d.severityCode} · {d.severityName || '—'}
                  </span>
                </div>
                <span className="font-mono text-[11px] text-cyan-300/70">
                  {d.eventType != null && d.eventType !== '' ? String(d.eventType) : '—'}
                </span>
                <div className="flex justify-end">
                  <StatusBadge
                    status={d.isShow ? 'online' : 'offline'}
                    label={d.isShow ? '可见' : '隐藏'}
                    className="scale-90"
                  />
                </div>
              </div>
            )
          })
        )}
      </div>

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

function SevCell({ value, color }: { value: number; color: string }) {
  if (!value) return <span className="text-right font-mono text-[11px] text-cyan-300/30">—</span>
  return (
    <span
      className="text-right font-display text-sm font-bold"
      style={{ color, textShadow: `0 0 6px ${color}` }}
    >
      {value}
    </span>
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
