import { useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  Search,
  Loader2,
  Gauge,
  ArrowLeft,
  ChevronRight,
  RefreshCcw,
  Activity,
  FolderCog,
  Plus,
  Pencil,
  Trash2,
  AlertTriangle,
  X,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useT } from '@/hooks/useT'
import {
  useIndicatorSummary,
  useIndicatorList,
  useDeleteIndicator,
  useEnabledIndicators,
  useSetEnabledIndicators,
} from '@core/hooks/api/useIndicatorsLibrary'
import GroupsManageModal from './GroupsManageModal'
import IndicatorFormModal from './IndicatorFormModal'
import type {
  DeviceType,
  TechLower,
  IndicatorPlatformSummary,
  IndicatorInfo,
} from '@core/types/indicatorLibrary'
import { techToDeviceType } from '@core/types/indicatorLibrary'

// 启用状态走 default 行（XML 真相源；运营商覆盖能力后端保留但 UI 不暴露选择器）。
const OPERATOR_CODE = 'default'

const VALID_TECHS: TechLower[] = ['enb', 'gsm', 'gnb']

const TECH_LABEL: Record<TechLower, string> = {
  enb: 'ENB · LTE',
  gsm: 'GSM',
  gnb: 'GNB · 5G NR',
}

const TECH_COLOR: Record<TechLower, string> = {
  enb: '#00f0ff',
  gsm: '#ffaa00',
  gnb: '#a855f7',
}

function parseTech(raw: string | null): TechLower | undefined {
  if (!raw) return undefined
  const lower = raw.toLowerCase()
  return VALID_TECHS.includes(lower as TechLower) ? (lower as TechLower) : undefined
}

/**
 * KPI 指标库（对照 v1 product/kpi-library）。
 * drill-down 主从：一级按 (制式, 平台) 聚合 → 点击平台进入该平台指标明细。
 * 选中平台走 URL ?tech=enb&platform=XXX，刷新保留二级页。
 */
export default function KpiLibraryPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const tech = parseTech(searchParams.get('tech'))
  const platform = searchParams.get('platform') || undefined

  const setTechPlatform = (t?: TechLower, p?: string) => {
    const params = new URLSearchParams(searchParams)
    if (t) params.set('tech', t)
    else params.delete('tech')
    if (p) params.set('platform', p)
    else params.delete('platform')
    setSearchParams(params, { replace: false })
  }

  return tech && platform ? (
    <IndicatorsView
      tech={tech}
      platform={platform}
      onBack={() => setTechPlatform(undefined, undefined)}
    />
  ) : (
    <SummaryView onSelect={(t, p) => setTechPlatform(t, p)} />
  )
}

// ── 一级：平台聚合 ──────────────────────────────────────────────────────
function SummaryView({ onSelect }: { onSelect: (tech: TechLower, platform: string) => void }) {
  const [draft, setDraft] = useState('')
  const [keyword, setKeyword] = useState('')

  const { data, isLoading, isError, error, isFetching, refetch } = useIndicatorSummary()

  const all = useMemo<IndicatorPlatformSummary[]>(() => data?.items ?? [], [data])
  const rows = useMemo(() => {
    const k = keyword.trim().toLowerCase()
    if (!k) return all
    return all.filter((r) => `${r.platform} ${r.description ?? ''}`.toLowerCase().includes(k))
  }, [all, keyword])

  const indicatorTotal = useMemo(() => all.reduce((acc, r) => acc + (r.indicators || 0), 0), [all])
  const techCount = useMemo(() => new Set(all.map((r) => r.tech)).size, [all])

  const applySearch = () => setKeyword(draft.trim())

  return (
    <PageShell
      code="F03"
      title="KPI LIBRARY · 指标库"
      subtitle="PER-PLATFORM INDICATOR CATALOG · COUNTER & FORMULA"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-80 pl-9"
              placeholder="平台名 / 描述"
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
        <Stat label="PLATFORMS" color="#00f0ff" value={all.length} />
        <Stat label="TECHNOLOGIES" color="#a855f7" value={techCount} />
        <Stat label="TOTAL INDICATORS" color="#00ff88" value={indicatorTotal} />
      </div>

      {rows.length > 0 && (
        <div className="mb-1 grid grid-cols-[0.8fr_1.4fr_2fr_0.8fr_0.8fr_60px] items-center gap-3 px-3 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
          <span>TECH</span>
          <span>PLATFORM</span>
          <span>LOADED FROM</span>
          <span className="text-right">INDICATORS</span>
          <span className="text-right">SOURCE</span>
          <span className="text-right">OPEN</span>
        </div>
      )}

      <div className="space-y-1.5">
        {isLoading ? (
          <LoadingRow />
        ) : isError ? (
          <ErrorRow message={error instanceof Error ? error.message : '未知错误'} />
        ) : rows.length === 0 ? (
          <EmptyRow keyword={keyword} icon={<Gauge className="size-10 text-cyan-400/50" />} label="NO PLATFORMS · 暂无指标平台" />
        ) : (
          rows.map((r) => (
            <button
              key={`${r.tech}__${r.platform}`}
              type="button"
              onClick={() => onSelect(r.tech, r.platform)}
              className="fleet-row grid w-full grid-cols-[0.8fr_1.4fr_2fr_0.8fr_0.8fr_60px] items-center gap-3 rounded-sm px-3 py-2.5 text-left"
              style={{ ['--row-color' as never]: TECH_COLOR[r.tech] }}
            >
              <span
                className="chip"
                style={{ color: TECH_COLOR[r.tech] }}
              >
                {TECH_LABEL[r.tech]}
              </span>
              <div className="min-w-0">
                <div className="truncate font-display text-sm font-bold text-cyan-100">{r.platform}</div>
                {r.description && (
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">{r.description}</div>
                )}
              </div>
              <code className="min-w-0 truncate font-mono text-[11px] text-cyan-300/70" title={r.loadedFrom}>
                {r.loadedFrom || '—'}
              </code>
              <span className="text-right font-display text-sm font-bold text-cyan-200">{r.indicators}</span>
              <div className="flex justify-end">
                <StatusBadge
                  status={r.source === 'builtin' ? 'active' : r.source === 'custom' ? 'ok' : 'unknown'}
                  label={r.source.toUpperCase()}
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
    </PageShell>
  )
}

// ── 二级：平台指标明细 ──────────────────────────────────────────────────
const PAGE_SIZE = 50

function IndicatorsView({
  tech,
  platform,
  onBack,
}: {
  tech: TechLower
  platform: string
  onBack: () => void
}) {
  const t = useT()
  const deviceType: DeviceType = techToDeviceType(tech)
  const [draft, setDraft] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [groupsOpen, setGroupsOpen] = useState(false)
  // 新建/编辑指标浮层:undefined=关闭;null=新建;有值=编辑该行。
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<IndicatorInfo | null>(null)
  // 待删除指标(手写二次确认浮层,v3 无 Popconfirm)。
  const [pendingDelete, setPendingDelete] = useState<IndicatorInfo | null>(null)
  // 启用开关 inflight 行 id(单行 loading 态)。
  const [pendingEnableId, setPendingEnableId] = useState<string | null>(null)
  const [banner, setBanner] = useState<string | null>(null)

  const { data, isLoading, isError, error, isFetching } = useIndicatorList(deviceType, {
    keyword: keyword || undefined,
    platformName: platform,
    page,
    pageSize: PAGE_SIZE,
  })
  const { data: enabledData } = useEnabledIndicators(deviceType, OPERATOR_CODE)
  const setEnabledMut = useSetEnabledIndicators()
  const deleteMut = useDeleteIndicator()

  const rows = useMemo<IndicatorInfo[]>(() => data?.items ?? [], [data])
  const enabledSet = useMemo(() => new Set(enabledData?.items ?? []), [enabledData])
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const isGnb = deviceType === 'GNB'

  const applySearch = () => {
    setKeyword(draft.trim())
    setPage(1)
  }

  const openCreate = () => {
    setEditing(null)
    setFormOpen(true)
  }
  const openEdit = (ind: IndicatorInfo) => {
    setEditing(ind)
    setFormOpen(true)
  }

  const toggleEnable = (id: string, enable: boolean) => {
    setPendingEnableId(id)
    setEnabledMut.mutate(
      { deviceType, operatorCode: OPERATOR_CODE, indicatorIds: [id], enable },
      {
        onSettled: () => setPendingEnableId(null),
        onError: (e) => setBanner(e instanceof Error ? e.message : String(e)),
      },
    )
  }

  const confirmDelete = async () => {
    if (!pendingDelete) return
    try {
      await deleteMut.mutateAsync({ deviceType, id: pendingDelete.id })
      setBanner(t('product.kpi.indicator.deleteSuccess'))
      setPendingDelete(null)
    } catch (e) {
      setBanner(e instanceof Error ? e.message : String(e))
      setPendingDelete(null)
    }
  }

  // 列布局:增 ENABLE + ACTIONS 两列(ENB 多一个 LEVEL 列)。
  const cols = isGnb
    ? 'grid-cols-[1.1fr_1.3fr_1.5fr_1fr_0.6fr_0.7fr_150px]'
    : 'grid-cols-[1.1fr_1.3fr_1.4fr_1fr_0.6fr_0.6fr_0.7fr_150px]'

  return (
    <PageShell
      code="F03"
      title={`KPI · ${platform}`}
      subtitle={`${TECH_LABEL[tech]} · INDICATOR DETAIL`}
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
              placeholder="指标 ID / 名称"
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
          <NeonButton icon={<Plus />} onClick={openCreate}>
            {t('product.kpi.indicator.newIndicator')}
          </NeonButton>
          <NeonButton icon={<FolderCog />} onClick={() => setGroupsOpen(true)}>
            {t('product.kpi.group.manage')}
          </NeonButton>
        </>
      }
    >
      <GroupsManageModal
        open={groupsOpen}
        onClose={() => setGroupsOpen(false)}
        deviceType={deviceType}
      />
      <IndicatorFormModal
        open={formOpen}
        onClose={() => setFormOpen(false)}
        deviceType={deviceType}
        operatorCode={OPERATOR_CODE}
        // 二级详情态由 URL ?platform= 锁定,新建时透传 → 后端同事务写占位 formula,
        // 避免详情列表 platform_name EXISTS 过滤把刚建的指标过滤掉。
        platform={platform}
        indicator={editing}
      />
      {banner ? (
        <div className="mb-2 flex items-center justify-between rounded-sm border border-cyan-500/20 bg-cyan-500/5 px-3 py-1.5">
          <span className="font-mono text-[11px] text-emerald-300/85">{banner}</span>
          <button
            type="button"
            onClick={() => setBanner(null)}
            aria-label="dismiss"
            className="text-cyan-300/50 hover:text-cyan-200"
          >
            <X className="size-3.5" />
          </button>
        </div>
      ) : null}
      {rows.length > 0 && (
        <div className={`mb-1 grid ${cols} items-center gap-3 px-3 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45`}>
          <span>ID</span>
          <span>CN NAME</span>
          <span>EN NAME</span>
          <span>GROUP</span>
          {!isGnb && <span>LEVEL</span>}
          <span className="text-right">TYPE</span>
          <span className="text-center">{t('common.enable')}</span>
          <span className="text-right">{t('common.operation')}</span>
        </div>
      )}

      <div className="space-y-1.5">
        {isLoading ? (
          <LoadingRow />
        ) : isError ? (
          <ErrorRow message={error instanceof Error ? error.message : '未知错误'} />
        ) : rows.length === 0 ? (
          <EmptyRow keyword={keyword} icon={<Activity className="size-10 text-cyan-400/50" />} label="NO INDICATORS · 该平台无指标" />
        ) : (
          rows.map((ind) => (
            <div
              key={ind.id}
              className={`fleet-row grid ${cols} items-center gap-3 rounded-sm px-3 py-2.5`}
              style={{ ['--row-color' as never]: ind.isCounter ? '#5b9eff' : '#00ff88' }}
            >
              <code className="min-w-0 truncate font-mono text-[11px] text-cyan-100/90">{ind.id}</code>
              <div className="min-w-0 truncate text-xs text-cyan-100/85" title={ind.cnName}>
                {ind.cnName || ind.name || '—'}
              </div>
              <div className="min-w-0 truncate font-mono text-[11px] text-cyan-300/65" title={ind.enName}>
                {ind.enName || '—'}
              </div>
              <div className="min-w-0">
                {ind.groupName || ind.groupId ? (
                  <span className="chip text-purple-300">{ind.groupName || ind.groupId}</span>
                ) : (
                  <span className="text-cyan-300/40">—</span>
                )}
              </div>
              {!isGnb && (
                <span className="font-mono text-[11px] text-cyan-300/70">{ind.indicatorLevel || '—'}</span>
              )}
              <div className="flex justify-end">
                <StatusBadge
                  status={ind.isCounter ? 'warning' : 'ok'}
                  label={ind.isCounter ? 'COUNTER' : 'KPI'}
                  className="scale-90"
                />
              </div>
              {/* 启用开关 */}
              <div className="flex justify-center">
                <button
                  type="button"
                  role="switch"
                  aria-checked={enabledSet.has(ind.id)}
                  disabled={pendingEnableId === ind.id}
                  title={enabledSet.has(ind.id) ? t('common.disable') : t('common.enable')}
                  onClick={() => toggleEnable(ind.id, !enabledSet.has(ind.id))}
                  className={
                    'relative h-4 w-8 shrink-0 rounded-full border transition-colors disabled:opacity-50 ' +
                    (enabledSet.has(ind.id)
                      ? 'border-cyan-400/70 bg-cyan-500/30'
                      : 'border-cyan-500/25 bg-cyan-950/40')
                  }
                >
                  {pendingEnableId === ind.id ? (
                    <Loader2 className="absolute left-1/2 top-1/2 size-2.5 -translate-x-1/2 -translate-y-1/2 animate-spin text-cyan-200" />
                  ) : (
                    <span
                      className={
                        'absolute top-0.5 size-3 rounded-full bg-cyan-200 transition-all ' +
                        (enabledSet.has(ind.id) ? 'left-4 shadow-[0_0_5px_#00f0ff]' : 'left-0.5')
                      }
                    />
                  )}
                </button>
              </div>
              {/* 操作:编辑 / 删除 */}
              <div className="flex justify-end gap-1.5">
                <NeonButton icon={<Pencil />} onClick={() => openEdit(ind)}>
                  {t('common.edit')}
                </NeonButton>
                <NeonButton
                  tone="danger"
                  icon={<Trash2 />}
                  disabled={deleteMut.isPending}
                  onClick={() => setPendingDelete(ind)}
                >
                  {t('common.delete')}
                </NeonButton>
              </div>
            </div>
          ))
        )}
      </div>

      {/* 删除二次确认浮层(v3 无 Popconfirm,手写) */}
      {pendingDelete ? (
        <div
          className="fixed inset-0 z-[80] flex items-center justify-center bg-[#02040a]/72 backdrop-blur-sm"
          role="dialog"
          aria-modal="true"
          onMouseDown={(e) => {
            if (e.target === e.currentTarget) setPendingDelete(null)
          }}
        >
          <GlassPanel strong className="warp-in overflow-hidden" style={{ width: 460 }}>
            <div className="flex items-center justify-between border-b border-cyan-500/15 px-4 py-3">
              <div className="font-display text-base text-cyan-100">{t('common.delete')}</div>
              <button
                type="button"
                onClick={() => setPendingDelete(null)}
                aria-label="close"
                className="rounded-sm border border-cyan-500/25 p-1 text-cyan-300/60 transition-colors hover:border-cyan-400/60 hover:bg-cyan-500/10 hover:text-cyan-200"
              >
                <X className="size-4" />
              </button>
            </div>
            <div className="flex items-start gap-3 px-4 py-4">
              <AlertTriangle className="mt-0.5 size-5 shrink-0 text-amber-300" />
              <div className="space-y-1">
                <div className="text-sm text-cyan-100">
                  {pendingDelete.cnName || pendingDelete.name || pendingDelete.id}
                </div>
                <div className="font-mono text-xs text-cyan-300/70">
                  {t('product.kpi.confirmDeleteIndicator', { id: pendingDelete.id })}
                </div>
              </div>
            </div>
            <div className="flex items-center justify-end gap-2 border-t border-cyan-500/15 px-4 py-3">
              <NeonButton onClick={() => setPendingDelete(null)} disabled={deleteMut.isPending}>
                {t('common.cancel')}
              </NeonButton>
              <NeonButton
                tone="danger"
                onClick={() => void confirmDelete()}
                disabled={deleteMut.isPending}
              >
                {deleteMut.isPending ? <Loader2 className="size-3.5 animate-spin" /> : null}
                {t('common.yes')}
              </NeonButton>
            </div>
          </GlassPanel>
        </div>
      ) : null}

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
