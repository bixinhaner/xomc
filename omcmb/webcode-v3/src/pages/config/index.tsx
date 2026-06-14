import { useMemo, useState } from 'react'
import {
  Search,
  RefreshCcw,
  Loader2,
  Layers,
  GitCompare,
  ListChecks,
  Send,
  X,
  CheckCircle2,
  XCircle,
  FileStack,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import {
  useConfigTemplates,
  useBaselineConfigs,
  useBaselineConfigById,
  useConfigTasks,
} from '@core/hooks/api/useConfig'
import { useDispatchTemplate } from '@core/hooks/api/useTemplate'
import { useDeviceList } from '@core/hooks/api/useDevices'
import type {
  ConfigTemplate,
  BaselineConfig,
  ConfigTask,
  ConfigTaskStatus,
  ConfigParam,
} from '@core/types/config'
import type { DispatchTemplateResponse } from '@core/services/api/templateApi'

type TabKey = 'templates' | 'baselines' | 'tasks'

const TABS: { key: TabKey; label: string; icon: typeof Layers }[] = [
  { key: 'templates', label: '模板库 · TEMPLATES', icon: Layers },
  { key: 'baselines', label: '基线对比 · BASELINES', icon: GitCompare },
  { key: 'tasks', label: '下发任务 · TASKS', icon: ListChecks },
]

const PAGE_SIZE = 50

// ---------------------------------------------------------------------------
// 任务状态视觉映射
// ---------------------------------------------------------------------------
const TASK_STATUS: Record<ConfigTaskStatus, { label: string; color: string; badge: string }> = {
  pending: { label: '等待', color: '#5b9eff', badge: 'unknown' },
  running: { label: '运行中', color: '#00f0ff', badge: 'active' },
  success: { label: '成功', color: '#00ff88', badge: 'ok' },
  failed: { label: '失败', color: '#ff2d6f', badge: 'critical' },
  partial: { label: '部分成功', color: '#ffaa00', badge: 'warning' },
  cancelled: { label: '已取消', color: '#525a78', badge: 'off' },
}

const BASELINE_STATUS: Record<BaselineConfig['status'], { label: string; color: string }> = {
  active: { label: '生效', color: '#00ff88' },
  draft: { label: '草稿', color: '#ffaa00' },
  deprecated: { label: '废弃', color: '#525a78' },
}

const TASK_TYPE_LABEL: Record<ConfigTask['taskType'], string> = {
  'param-sync': '参数同步',
  'batch-config': '批量配置',
  'baseline-apply': '基线应用',
  'neighbor-update': '邻区更新',
}

export function ConfigPage() {
  const [tab, setTab] = useState<TabKey>('templates')
  const [keyword, setKeyword] = useState('')

  // 三个域各自分页（互不干扰）；当前页全部拉一页足够概览，规模化筛选留后续
  const templatesQ = useConfigTemplates({ page: 1, pageSize: PAGE_SIZE })
  const baselinesQ = useBaselineConfigs({ page: 1, pageSize: 100 })
  const tasksQ = useConfigTasks({ page: 1, pageSize: PAGE_SIZE })

  const templates = templatesQ.data?.items ?? []
  const baselines = baselinesQ.data?.items ?? []
  const tasks = tasksQ.data?.items ?? []

  // 顶部统计（全部来自真实数据）
  const stats = useMemo(() => {
    const inProgress = tasks.filter((t) => t.status === 'running' || t.status === 'pending').length
    const success24h = tasks.filter((t) => t.status === 'success').length
    const failed24h = tasks.filter((t) => t.status === 'failed' || t.status === 'partial').length
    const activeBaselines = baselines.filter((b) => b.status === 'active').length
    return {
      templates: templatesQ.data?.total ?? templates.length,
      activeBaselines,
      inProgress,
      success24h,
      failed24h,
    }
  }, [tasks, baselines, templates.length, templatesQ.data?.total])

  const isFetching =
    templatesQ.isFetching || baselinesQ.isFetching || tasksQ.isFetching

  const refetchActive = () => {
    if (tab === 'templates') void templatesQ.refetch()
    else if (tab === 'baselines') void baselinesQ.refetch()
    else void tasksQ.refetch()
  }

  // ---- 模板下发 modal 状态 ----
  const [dispatchTarget, setDispatchTarget] = useState<ConfigTemplate | null>(null)

  return (
    <PageShell
      code="F02"
      title="CONFIG · 配置编排"
      subtitle="TR-069 TEMPLATE / BASELINE / DISPATCH"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="名称 / 类型 / 创建者"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={refetchActive}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {/* 顶部统计 */}
      <div className="mb-3 grid grid-cols-4 gap-3">
        <StatCard label="TEMPLATES · 模板" value={stats.templates} color="#00f0ff" />
        <StatCard label="ACTIVE BASELINE · 生效基线" value={stats.activeBaselines} color="#a855f7" />
        <StatCard label="IN-PROGRESS · 进行中" value={stats.inProgress} color="#ffaa00" />
        <StatCard
          label="OK / FAIL · 任务结果"
          value={`${stats.success24h} / ${stats.failed24h}`}
          color={stats.failed24h > 0 ? '#ff2d6f' : '#00ff88'}
        />
      </div>

      {/* Tab 切换 */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        {TABS.map((tDef) => {
          const Icon = tDef.icon
          const on = tab === tDef.key
          return (
            <button
              key={tDef.key}
              type="button"
              onClick={() => setTab(tDef.key)}
              className={`chip transition-all ${
                on ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55 hover:opacity-100'
              }`}
              style={{ color: on ? '#00f0ff' : '#6b86b6' }}
            >
              <Icon className="size-3.5" />
              {tDef.label}
            </button>
          )
        })}
      </div>

      {/* 主内容 */}
      {tab === 'templates' && (
        <TemplatesPanel
          query={templatesQ}
          items={templates}
          keyword={keyword}
          onDispatch={setDispatchTarget}
        />
      )}
      {tab === 'baselines' && (
        <BaselinesPanel query={baselinesQ} items={baselines} keyword={keyword} />
      )}
      {tab === 'tasks' && <TasksPanel query={tasksQ} items={tasks} keyword={keyword} />}

      {dispatchTarget && (
        <DispatchModal template={dispatchTarget} onClose={() => setDispatchTarget(null)} />
      )}
    </PageShell>
  )
}

// ===========================================================================
// 顶部统计卡
// ===========================================================================
function StatCard({
  label,
  value,
  color,
}: {
  label: string
  value: number | string
  color: string
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3"
      style={{ borderLeftColor: color }}
    >
      <div className="scanline" />
      <div className="relative font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
        {label}
      </div>
      <div
        className="relative font-display text-3xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {typeof value === 'number' ? value.toLocaleString() : value}
      </div>
    </div>
  )
}

// ===========================================================================
// 三态包装：loading / error / empty
// ===========================================================================
function QueryStates({
  isLoading,
  isError,
  error,
  isEmpty,
  emptyText,
}: {
  isLoading: boolean
  isError: boolean
  error: unknown
  isEmpty: boolean
  emptyText: string
}) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
        <Loader2 className="size-4 animate-spin" />
        <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
      </div>
    )
  }
  if (isError) {
    return (
      <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
        FAILURE · {error instanceof Error ? error.message : '未知错误'}
      </div>
    )
  }
  if (isEmpty) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-16">
        <FileStack className="size-10 text-cyan-400/45" />
        <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/60">
          {emptyText}
        </div>
      </div>
    )
  }
  return null
}

interface QueryShape {
  isLoading: boolean
  isError: boolean
  error: unknown
}

// ===========================================================================
// 模板库面板
// ===========================================================================
function TemplatesPanel({
  query,
  items,
  keyword,
  onDispatch,
}: {
  query: QueryShape
  items: ConfigTemplate[]
  keyword: string
  onDispatch: (t: ConfigTemplate) => void
}) {
  const kw = keyword.trim().toLowerCase()
  const rows = kw
    ? items.filter(
        (t) =>
          t.templateName?.toLowerCase().includes(kw) ||
          t.description?.toLowerCase().includes(kw) ||
          t.creator?.toLowerCase().includes(kw)
      )
    : items

  const states = (
    <QueryStates
      isLoading={query.isLoading}
      isError={query.isError}
      error={query.error}
      isEmpty={rows.length === 0}
      emptyText="无模板"
    />
  )
  if (query.isLoading || query.isError || rows.length === 0) return states

  return (
    <div className="space-y-1.5">
      {rows.map((t) => (
        <div
          key={t.id}
          className="fleet-row grid grid-cols-[2fr_1.4fr_90px_1fr_120px] items-center gap-3 rounded-sm px-3 py-2.5"
          style={{ ['--row-color' as never]: '#00f0ff' }}
        >
          <div className="min-w-0">
            <div className="truncate font-display text-sm font-bold text-cyan-100">
              {t.templateName || '—'}
            </div>
            <div className="truncate font-mono text-[10px] text-cyan-300/55">
              {t.description || '无描述'}
            </div>
          </div>
          <div className="font-mono text-[11px] text-cyan-300/75">
            <span className="text-cyan-300/45">CREATOR </span>
            {t.creator || '—'}
          </div>
          <div className="text-center">
            <span
              className="font-display text-lg font-bold text-cyan-200"
              style={{ textShadow: '0 0 6px #00f0ff' }}
            >
              {t.params?.length ?? 0}
            </span>
            <div className="font-mono text-[9px] uppercase tracking-[0.15em] text-cyan-300/45">
              PARAMS
            </div>
          </div>
          <div className="font-mono text-[11px] text-cyan-300/75">{formatTime(t.createTime)}</div>
          <div className="text-right">
            <NeonButton icon={<Send />} onClick={() => onDispatch(t)}>
              下发
            </NeonButton>
          </div>
        </div>
      ))}
    </div>
  )
}

// ===========================================================================
// 基线对比面板（左侧基线树 + 右侧差异参数表）
// ===========================================================================
function BaselinesPanel({
  query,
  items,
  keyword,
}: {
  query: QueryShape
  items: BaselineConfig[]
  keyword: string
}) {
  const kw = keyword.trim().toLowerCase()
  const filtered = kw
    ? items.filter(
        (b) =>
          b.baselineName?.toLowerCase().includes(kw) ||
          b.deviceType?.toLowerCase().includes(kw)
      )
    : items

  const [selectedId, setSelectedId] = useState<string>('')
  const effectiveId = selectedId || filtered[0]?.id || ''
  const detailQ = useBaselineConfigById(effectiveId)
  const detailParams: ConfigParam[] = detailQ.data?.params ?? []

  // 按 deviceType 分组
  const grouped = useMemo(() => {
    const g: Record<string, BaselineConfig[]> = {}
    for (const b of filtered) {
      const k = b.deviceType || '未分类'
      if (!g[k]) g[k] = []
      g[k].push(b)
    }
    return g
  }, [filtered])

  const states = (
    <QueryStates
      isLoading={query.isLoading}
      isError={query.isError}
      error={query.error}
      isEmpty={filtered.length === 0}
      emptyText="无基线"
    />
  )
  if (query.isLoading || query.isError || filtered.length === 0) return states

  const selected = filtered.find((b) => b.id === effectiveId) ?? null
  // 差异：当前值与默认值不同视为偏离基线
  const diffParams = detailParams.filter(
    (p) => String(p.paramValue) !== String(p.defaultValue)
  )

  return (
    <div className="grid grid-cols-[300px_1fr] gap-3">
      {/* 左：基线树 */}
      <GlassPanel title="BASELINES · 基线" className="min-h-0">
        <div className="max-h-[58vh] overflow-auto p-2">
          {Object.entries(grouped).map(([deviceType, group]) => (
            <div key={deviceType} className="mb-2">
              <div className="px-2 py-1 font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/45">
                {deviceType}
              </div>
              {group.map((b) => {
                const on = b.id === effectiveId
                const st = BASELINE_STATUS[b.status]
                return (
                  <button
                    key={b.id}
                    type="button"
                    onClick={() => setSelectedId(b.id)}
                    className={`flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left transition-colors ${
                      on ? 'bg-cyan-500/12' : 'hover:bg-cyan-500/6'
                    }`}
                  >
                    <span
                      className="size-1.5 shrink-0 rounded-full"
                      style={{ background: st.color, boxShadow: `0 0 6px ${st.color}` }}
                    />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-xs text-cyan-100/90">
                        {b.baselineName}
                      </span>
                      <span className="block font-mono text-[10px] text-cyan-300/45">
                        v{b.version} · {st.label}
                      </span>
                    </span>
                  </button>
                )
              })}
            </div>
          ))}
        </div>
      </GlassPanel>

      {/* 右：差异参数表 */}
      <GlassPanel
        title={selected ? `DIFF · ${selected.baselineName}` : 'DIFF · 差异'}
        meta={
          selected
            ? `${diffParams.length} DEVIATIONS · ${detailParams.length} PARAMS`
            : undefined
        }
        className="min-h-0"
      >
        {detailQ.isLoading ? (
          <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
            <Loader2 className="size-4 animate-spin" />
            <span className="font-mono text-xs uppercase tracking-[0.2em]">LOADING…</span>
          </div>
        ) : detailQ.isError ? (
          <div className="m-3 border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
            FAILURE · {detailQ.error instanceof Error ? detailQ.error.message : '未知错误'}
          </div>
        ) : detailParams.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-3 py-16">
            <GitCompare className="size-9 text-cyan-400/45" />
            <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
              无参数 · 选择基线查看差异
            </div>
          </div>
        ) : (
          <div className="max-h-[58vh] overflow-auto">
            <div className="grid grid-cols-[1.4fr_1fr_1fr_90px] gap-3 border-b border-cyan-500/15 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/45">
              <span>PARAM</span>
              <span>BASELINE / 默认</span>
              <span>CURRENT / 当前</span>
              <span className="text-right">DIFF</span>
            </div>
            {detailParams.map((p) => {
              const isDiff = String(p.paramValue) !== String(p.defaultValue)
              return (
                <div
                  key={p.id}
                  className="grid grid-cols-[1.4fr_1fr_1fr_90px] items-center gap-3 border-b border-cyan-500/8 px-3 py-2 hover:bg-cyan-500/5"
                >
                  <div className="min-w-0">
                    <div className="truncate text-xs text-cyan-100/90">{p.paramName}</div>
                    <code className="block truncate font-mono text-[10px] text-cyan-300/50">
                      {p.paramCode}
                    </code>
                  </div>
                  <div className="font-mono text-[11px] text-cyan-300/70">
                    {String(p.defaultValue)}
                    {p.unit ? <span className="text-cyan-300/40"> {p.unit}</span> : null}
                  </div>
                  <div
                    className="font-mono text-[11px]"
                    style={{
                      color: isDiff ? '#ffaa00' : 'rgba(190,225,255,0.7)',
                      fontWeight: isDiff ? 600 : 400,
                    }}
                  >
                    {String(p.paramValue)}
                    {p.unit ? <span className="opacity-50"> {p.unit}</span> : null}
                  </div>
                  <div className="text-right">
                    {isDiff ? (
                      <StatusBadge status="warning" label="偏离" />
                    ) : (
                      <StatusBadge status="ok" label="一致" />
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </GlassPanel>
    </div>
  )
}

// ===========================================================================
// 下发任务面板
// ===========================================================================
function TasksPanel({
  query,
  items,
  keyword,
}: {
  query: QueryShape
  items: ConfigTask[]
  keyword: string
}) {
  const kw = keyword.trim().toLowerCase()
  const rows = kw
    ? items.filter(
        (t) =>
          t.taskName?.toLowerCase().includes(kw) ||
          t.creator?.toLowerCase().includes(kw) ||
          t.taskType?.toLowerCase().includes(kw)
      )
    : items

  const states = (
    <QueryStates
      isLoading={query.isLoading}
      isError={query.isError}
      error={query.error}
      isEmpty={rows.length === 0}
      emptyText="无下发任务"
    />
  )
  if (query.isLoading || query.isError || rows.length === 0) return states

  return (
    <div className="space-y-1.5">
      {rows.map((t) => {
        const st = TASK_STATUS[t.status]
        const pct = Math.max(0, Math.min(100, Math.round(t.progress ?? 0)))
        return (
          <div
            key={t.id}
            className="fleet-row grid grid-cols-[2fr_1fr_1.2fr_1fr_120px] items-center gap-3 rounded-sm px-3 py-2.5"
            style={{ ['--row-color' as never]: st.color }}
          >
            <div className="min-w-0">
              <div className="truncate font-display text-sm font-bold text-cyan-100">
                {t.taskName || '—'}
              </div>
              <div className="truncate font-mono text-[10px] text-cyan-300/55">
                {TASK_TYPE_LABEL[t.taskType] ?? t.taskType} · {t.deviceSns?.length ?? 0} 台
                {t.message ? ` · ${t.message}` : ''}
              </div>
            </div>
            <div className="font-mono text-[11px] text-cyan-300/75">{t.creator || '—'}</div>
            {/* 进度条 */}
            <div>
              <div className="mb-1 flex items-center justify-between font-mono text-[10px] text-cyan-300/60">
                <span>{pct}%</span>
                <span>
                  <span style={{ color: '#00ff88' }}>{t.successCount}</span>
                  {' / '}
                  <span style={{ color: '#ff2d6f' }}>{t.failCount}</span>
                  {' / '}
                  {t.totalCount}
                </span>
              </div>
              <div className="h-1.5 overflow-hidden rounded-full bg-cyan-500/10">
                <div
                  className="h-full rounded-full transition-all"
                  style={{
                    width: `${pct}%`,
                    background: st.color,
                    boxShadow: `0 0 8px ${st.color}`,
                  }}
                />
              </div>
            </div>
            <div className="font-mono text-[11px] text-cyan-300/70">{formatTime(t.createdAt)}</div>
            <div className="text-right">
              <StatusBadge status={st.badge} label={st.label} />
            </div>
          </div>
        )
      })}
    </div>
  )
}

// ===========================================================================
// 模板下发 Modal（选设备 → dispatch → 结果）
// ===========================================================================
function DispatchModal({
  template,
  onClose,
}: {
  template: ConfigTemplate
  onClose: () => void
}) {
  const [selectedIds, setSelectedIds] = useState<string[]>([])
  const [deviceKw, setDeviceKw] = useState('')
  const [result, setResult] = useState<DispatchTemplateResponse | null>(null)
  const [errMsg, setErrMsg] = useState<string>('')

  const dispatch = useDispatchTemplate()
  const { data: devicePage, isLoading: devLoading } = useDeviceList({ page: 1, pageSize: 200 })

  const allDevices = devicePage?.items ?? []
  const kw = deviceKw.trim().toLowerCase()
  const devices = kw
    ? allDevices.filter(
        (d) =>
          d.sn?.toLowerCase().includes(kw) || d.name?.toLowerCase().includes(kw)
      )
    : allDevices

  const toggle = (id: string) =>
    setSelectedIds((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]
    )

  const handleDispatch = () => {
    if (selectedIds.length === 0) return
    setErrMsg('')
    dispatch.mutate(
      { templateId: template.id, deviceIds: selectedIds },
      {
        onSuccess: (resp) => setResult(resp),
        onError: (err: Error) => setErrMsg(err.message || '下发失败'),
      }
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-[#02030a]/75 backdrop-blur-sm">
      <div className="w-[680px] max-w-[92vw]">
        <GlassPanel
          strong
          title={`DISPATCH · 下发模板：${template.templateName}`}
          meta={`${template.params?.length ?? 0} PARAMS`}
        >
          <div className="max-h-[72vh] overflow-auto p-4">
            {result ? (
              <DispatchResult result={result} />
            ) : (
              <>
                <div className="relative mb-3">
                  <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
                  <input
                    className="neon-input w-full pl-9"
                    placeholder="按 SN / 名称筛选设备"
                    value={deviceKw}
                    onChange={(e) => setDeviceKw(e.target.value)}
                  />
                </div>

                {devLoading ? (
                  <div className="flex items-center justify-center gap-2 py-10 text-cyan-300/60">
                    <Loader2 className="size-4 animate-spin" />
                    <span className="font-mono text-xs uppercase tracking-[0.2em]">
                      LOADING DEVICES…
                    </span>
                  </div>
                ) : devices.length === 0 ? (
                  <div className="py-10 text-center font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
                    无可选设备
                  </div>
                ) : (
                  <div className="max-h-[38vh] space-y-1 overflow-auto">
                    {devices.map((d) => {
                      const on = selectedIds.includes(d.id)
                      return (
                        <button
                          key={d.id}
                          type="button"
                          onClick={() => toggle(d.id)}
                          className={`flex w-full items-center gap-2 rounded-sm border px-3 py-2 text-left transition-colors ${
                            on
                              ? 'border-cyan-400/50 bg-cyan-500/12'
                              : 'border-cyan-500/10 hover:bg-cyan-500/6'
                          }`}
                        >
                          <span
                            className={`flex size-4 shrink-0 items-center justify-center rounded-sm border ${
                              on ? 'border-cyan-400 bg-cyan-400/20' : 'border-cyan-500/30'
                            }`}
                          >
                            {on ? <CheckCircle2 className="size-3 text-cyan-300" /> : null}
                          </span>
                          <span className="min-w-0 flex-1">
                            <span className="block truncate font-mono text-xs text-cyan-100">
                              {d.sn}
                            </span>
                            {d.name && d.name !== d.sn ? (
                              <span className="block truncate text-[10px] text-cyan-300/55">
                                {d.name}
                              </span>
                            ) : null}
                          </span>
                          {d.productClass ? (
                            <span className="chip text-[#a855f7]">{d.productClass}</span>
                          ) : null}
                        </button>
                      )
                    })}
                  </div>
                )}

                {errMsg ? (
                  <div className="mt-3 border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">
                    {errMsg}
                  </div>
                ) : null}

                <div className="mt-3 font-mono text-[10px] leading-relaxed text-cyan-300/45">
                  下发强制走 Path A（模板 standardPath → privatePath 翻译 → SetParameterValues），
                  不受全局 auto_configure 开关影响；每台设备独立成败。
                </div>
              </>
            )}
          </div>

          <div className="flex items-center justify-between border-t border-cyan-500/15 px-4 py-3">
            <span className="font-mono text-[11px] text-cyan-300/55">
              {result ? 'RESULT' : `SELECTED ${selectedIds.length}`}
            </span>
            <div className="flex gap-2">
              <NeonButton icon={<X />} onClick={onClose}>
                {result ? '关闭' : '取消'}
              </NeonButton>
              {!result && (
                <NeonButton
                  icon={dispatch.isPending ? <Loader2 className="animate-spin" /> : <Send />}
                  onClick={handleDispatch}
                  disabled={selectedIds.length === 0 || dispatch.isPending}
                >
                  确认下发到 {selectedIds.length} 台
                </NeonButton>
              )}
            </div>
          </div>
        </GlassPanel>
      </div>
    </div>
  )
}

function DispatchResult({ result }: { result: DispatchTemplateResponse }) {
  const rows = [
    ...result.dispatched.map((r) => ({ ...r, ok: true, key: `s-${r.deviceId}` })),
    ...result.failed.map((r) => ({ ...r, ok: false, key: `f-${r.deviceId}` })),
  ]
  return (
    <div>
      <div className="mb-3 flex gap-2">
        <span className="chip text-[#00f0ff]">总计 {result.totalDevices}</span>
        <span className="chip text-[#00ff88]">成功 {result.dispatched.length}</span>
        <span className={`chip ${result.failed.length > 0 ? 'text-[#ff2d6f]' : 'text-[#525a78]'}`}>
          失败 {result.failed.length}
        </span>
      </div>
      <div className="space-y-1">
        {rows.map((r) => (
          <div
            key={r.key}
            className="grid grid-cols-[28px_1fr_1.4fr] items-center gap-2 rounded-sm border border-cyan-500/10 px-3 py-2"
          >
            {r.ok ? (
              <CheckCircle2 className="size-4 text-emerald-400" />
            ) : (
              <XCircle className="size-4 text-rose-400" />
            )}
            <code className="truncate font-mono text-[11px] text-cyan-100/85">{r.deviceId}</code>
            <span className="truncate font-mono text-[11px]">
              {r.ok ? (
                <span className="text-cyan-300/70">TASK {r.taskId ?? '—'}</span>
              ) : (
                <span className="text-rose-300">{r.error ?? '失败'}</span>
              )}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}
