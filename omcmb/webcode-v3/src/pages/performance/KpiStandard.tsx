import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Activity,
  ChevronRight,
  Database,
  Layers,
  Loader2,
  RefreshCcw,
  Search,
  ToggleLeft,
  ToggleRight,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { cn } from '@/lib/utils'
import {
  useIndicatorGroupTree,
  useIndicatorList,
  useEnableIndicators,
  useDisableIndicators,
} from '@core/hooks/api/useIndicator'
import type { IndicatorGroup } from '@core/types/indicator'

/**
 * F03 · KPI 标准报表（指标库管理）
 * 左：功能集树（按制式）；右：指标列表（启用/停用测量 + 跳详情）。
 * 全走真实 useIndicatorGroupTree / useIndicatorList / useEnable·DisableIndicators。
 */

type DeviceTab = 'ENB' | 'GSM' | 'GNB'

const DEVICE_TABS: { key: DeviceTab; label: string }[] = [
  { key: 'ENB', label: 'LTE · ENB' },
  { key: 'GNB', label: 'NR · GNB' },
  { key: 'GSM', label: 'GSM' },
]

const TYPE_FILTERS: { value: string; label: string }[] = [
  { value: '', label: 'ALL' },
  { value: '0', label: 'KPI' },
  { value: '1', label: 'COUNTER' },
]

const ENABLE_FILTERS: { value: string; label: string }[] = [
  { value: '', label: '全部测量' },
  { value: '1', label: '已测量' },
  { value: '0', label: '未测量' },
]

const PAGE_SIZE = 20

// 扁平化功能集树（含层级缩进），供左侧树渲染。
interface FlatGroup {
  id: string
  label: string
  depth: number
  isRoot: boolean
}

function flattenGroups(groups: IndicatorGroup[], depth = 0, acc: FlatGroup[] = []): FlatGroup[] {
  for (const g of groups) {
    const isRoot = g.parentId === g.id
    acc.push({
      id: g.id,
      label: g.cnName || g.enName || g.id,
      depth,
      isRoot,
    })
    if (g.children?.length) flattenGroups(g.children, depth + 1, acc)
  }
  return acc
}

export default function KpiStandard() {
  const navigate = useNavigate()
  const [deviceType, setDeviceType] = useState<DeviceTab>('ENB')
  const [selectedGroup, setSelectedGroup] = useState<string>('')
  const [keyword, setKeyword] = useState('')
  const [typeFilter, setTypeFilter] = useState('')
  const [enableFilter, setEnableFilter] = useState('')
  const [page, setPage] = useState(1)
  const [treeSearch, setTreeSearch] = useState('')

  const enableMut = useEnableIndicators()
  const disableMut = useDisableIndicators()

  const { data: groupTree = [], isLoading: treeLoading } = useIndicatorGroupTree({ deviceType })

  const flatGroups = useMemo(() => {
    const flat = flattenGroups(groupTree)
    if (!treeSearch.trim()) return flat
    const kw = treeSearch.trim().toLowerCase()
    return flat.filter((g) => g.label.toLowerCase().includes(kw) || g.isRoot)
  }, [groupTree, treeSearch])

  // 根节点（自引用）不作为过滤条件。
  const effectiveGroupId = useMemo(() => {
    if (!selectedGroup) return undefined
    const root = flattenGroups(groupTree).find((g) => g.id === selectedGroup && g.isRoot)
    return root ? undefined : selectedGroup
  }, [selectedGroup, groupTree])

  const listParams = useMemo(
    () => ({
      deviceType,
      catagoryId: effectiveGroupId,
      searchText: keyword.trim() || undefined,
      indicatorType: typeFilter || undefined,
      isEnable: enableFilter || undefined,
      page,
      rows: PAGE_SIZE,
    }),
    [deviceType, effectiveGroupId, keyword, typeFilter, enableFilter, page],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useIndicatorList(listParams)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const switchDevice = (dt: DeviceTab) => {
    setDeviceType(dt)
    setSelectedGroup('')
    setPage(1)
  }

  const toggleMeasure = (kpiId: string, enable: boolean) => {
    const mut = enable ? enableMut : disableMut
    mut.mutate(
      { deviceType, operatorCode: 'default', indicatorIds: [kpiId], enable },
      { onSettled: () => void refetch() },
    )
  }

  const mutating = enableMut.isPending || disableMut.isPending

  return (
    <PageShell
      code="F03"
      title="KPI STANDARD · 标准指标库"
      subtitle="INDICATOR LIBRARY · MEASUREMENT TOGGLE · K/C CATALOG"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          {DEVICE_TABS.map((t) => (
            <button
              key={t.key}
              type="button"
              onClick={() => switchDevice(t.key)}
              className={cn(
                'chip transition-all',
                deviceType === t.key
                  ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                  : 'text-cyan-300/55 opacity-70 hover:opacity-100',
              )}
            >
              {t.label}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="grid grid-cols-12 gap-3">
        {/* 左：功能集树 */}
        <div className="col-span-12 lg:col-span-3">
          <GlassPanel title="FUNCTION SET · 功能集" meta={`${flatGroups.length}`}>
            <div className="border-b border-cyan-500/12 p-2.5">
              <div className="relative">
                <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
                <input
                  className="neon-input w-full pl-9"
                  placeholder="搜索功能集"
                  value={treeSearch}
                  onChange={(e) => setTreeSearch(e.target.value)}
                />
              </div>
            </div>
            <div className="max-h-[60vh] overflow-auto p-1.5">
              {treeLoading ? (
                <Loading text="LOADING TREE…" small />
              ) : flatGroups.length === 0 ? (
                <EmptyBox text="NO FUNCTION SET" small />
              ) : (
                <>
                  <button
                    type="button"
                    onClick={() => {
                      setSelectedGroup('')
                      setPage(1)
                    }}
                    className={cn(
                      'flex w-full items-center gap-1.5 rounded-sm px-2.5 py-1.5 text-left transition-colors hover:bg-cyan-500/5',
                      selectedGroup === '' && 'bg-cyan-500/10',
                    )}
                  >
                    <Layers className="size-3.5 text-cyan-300/70" />
                    <span className="text-[12px] text-cyan-100">全部指标</span>
                  </button>
                  {flatGroups.map((g) => (
                    <button
                      key={g.id}
                      type="button"
                      onClick={() => {
                        setSelectedGroup(g.id)
                        setPage(1)
                      }}
                      className={cn(
                        'flex w-full items-center gap-1.5 rounded-sm py-1.5 pr-2.5 text-left transition-colors hover:bg-cyan-500/5',
                        selectedGroup === g.id && 'bg-cyan-500/10',
                      )}
                      style={{ paddingLeft: 10 + g.depth * 14 }}
                    >
                      <ChevronRight className="size-3 shrink-0 text-cyan-300/40" />
                      <span
                        className={cn(
                          'truncate text-[12px]',
                          g.isRoot ? 'font-bold text-cyan-200' : 'text-cyan-100/85',
                        )}
                        title={g.label}
                      >
                        {g.label}
                      </span>
                    </button>
                  ))}
                </>
              )}
            </div>
          </GlassPanel>
        </div>

        {/* 右：指标列表 */}
        <div className="col-span-12 lg:col-span-9">
          <GlassPanel strong title="INDICATOR LIST · 指标" meta={`${total} TOTAL`}>
            <div className="flex flex-wrap items-center gap-2 border-b border-cyan-500/12 p-3">
              <div className="relative">
                <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
                <input
                  className="neon-input w-56 pl-9"
                  placeholder="指标名 / 编号搜索"
                  value={keyword}
                  onChange={(e) => {
                    setKeyword(e.target.value)
                    setPage(1)
                  }}
                />
              </div>
              {TYPE_FILTERS.map((f) => (
                <button
                  key={f.value || 'all'}
                  type="button"
                  onClick={() => {
                    setTypeFilter(f.value)
                    setPage(1)
                  }}
                  className={cn(
                    'chip transition-all',
                    typeFilter === f.value
                      ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                      : 'text-cyan-300/55 opacity-70 hover:opacity-100',
                  )}
                >
                  {f.label}
                </button>
              ))}
              <span className="mx-1 h-4 w-px bg-cyan-500/15" />
              {ENABLE_FILTERS.map((f) => (
                <button
                  key={f.value || 'allm'}
                  type="button"
                  onClick={() => {
                    setEnableFilter(f.value)
                    setPage(1)
                  }}
                  className={cn(
                    'chip transition-all',
                    enableFilter === f.value
                      ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                      : 'text-cyan-300/55 opacity-70 hover:opacity-100',
                  )}
                >
                  {f.label}
                </button>
              ))}
            </div>

            <div className="p-3">
              {/* 表头 */}
              <div className="grid grid-cols-[150px_1fr_90px_70px_90px_100px] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                <span>ID · 编号</span>
                <span>NAME · 名称</span>
                <span>TYPE</span>
                <span>UNIT</span>
                <span>SOURCE</span>
                <span className="text-right">MEASURE</span>
              </div>

              <div className="mt-1.5 space-y-1">
                {isLoading ? (
                  <Loading text="SYNCING INDICATORS…" />
                ) : isError ? (
                  <ErrorBox text={`LOAD FAILED · ${error instanceof Error ? error.message : '未知错误'}`} />
                ) : rows.length === 0 ? (
                  <EmptyBox text="NO INDICATOR" />
                ) : (
                  rows.map((r) => (
                    <div
                      key={r.kpiId}
                      className="grid grid-cols-[150px_1fr_90px_70px_90px_100px] items-center gap-3 rounded-sm px-3 py-2 transition-colors hover:bg-cyan-500/5"
                    >
                      <button
                        type="button"
                        onClick={() =>
                          navigate(`/performance/kpi-standard/detail/${deviceType}/${r.kpiId}`)
                        }
                        className="truncate text-left font-mono text-[11px] text-cyan-300 hover:text-cyan-100 hover:underline"
                        title={r.kpiId}
                      >
                        {r.kpiId}
                      </button>
                      <div className="min-w-0">
                        <div className="truncate font-display text-sm font-bold text-cyan-100" title={r.kpiName}>
                          {r.kpiName}
                        </div>
                        {r.custName ? (
                          <div className="truncate font-mono text-[10px] text-cyan-300/45">
                            {r.custName}
                          </div>
                        ) : null}
                      </div>
                      <span
                        className="chip w-fit"
                        style={{ color: r.indicatorType === 'counter' ? '#00ff88' : '#5b9eff' }}
                      >
                        {r.indicatorType === 'counter' ? 'CTR' : 'KPI'}
                      </span>
                      <span className="font-mono text-[11px] text-cyan-300/70">{r.unit || '—'}</span>
                      <StatusBadge
                        status={r.isCustomize ? 'warning' : 'online'}
                        label={r.isCustomize ? '自定义' : '内置'}
                        className="w-fit"
                      />
                      <div className="flex justify-end">
                        <button
                          type="button"
                          disabled={mutating}
                          onClick={() => toggleMeasure(r.kpiId, !r.isEnable)}
                          className={cn(
                            'flex items-center gap-1 font-mono text-[10px] uppercase tracking-[0.1em] transition-colors disabled:opacity-40',
                            r.isEnable ? 'text-[#00ff88]' : 'text-cyan-300/45 hover:text-cyan-200',
                          )}
                          title={r.isEnable ? '点击停用测量' : '点击启用测量'}
                        >
                          {r.isEnable ? (
                            <ToggleRight className="size-4" />
                          ) : (
                            <ToggleLeft className="size-4" />
                          )}
                          {r.isEnable ? 'ON' : 'OFF'}
                        </button>
                      </div>
                    </div>
                  ))
                )}
              </div>

              {/* 分页 */}
              <div className="mt-3 flex items-center justify-between">
                <span className="flex items-center gap-2 font-mono text-[11px] text-cyan-300/55">
                  {mutating ? <Loader2 className="size-3 animate-spin" /> : null}
                  PAGE {page} / {totalPages} · TOTAL {total}
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
            </div>
          </GlassPanel>
          <p className="mt-2 flex items-center gap-1.5 px-1 font-mono text-[10px] text-cyan-300/35">
            <Database className="size-3" />
            指标 ID 为 K/C 编号 · 点编号进详情 · MEASURE 切换即调真实启用/停用接口
            <Activity className="size-3" />
          </p>
        </div>
      </div>
    </PageShell>
  )
}

/* ───────── 复用小件 ───────── */

function Loading({ text, small }: { text: string; small?: boolean }) {
  return (
    <div className={cn('flex items-center justify-center gap-2 text-cyan-300/60', small ? 'py-6' : 'py-16')}>
      <Loader2 className={cn('animate-spin', small ? 'size-4' : 'size-5')} />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">{text}</span>
    </div>
  )
}

function ErrorBox({ text }: { text: string }) {
  return (
    <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
      {text}
    </div>
  )
}

function EmptyBox({ text, small }: { text: string; small?: boolean }) {
  return (
    <div
      className={cn(
        'flex items-center justify-center font-mono uppercase tracking-[0.2em] text-cyan-300/40',
        small ? 'py-6 text-[10px]' : 'py-12 text-xs',
      )}
    >
      {text}
    </div>
  )
}
