import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, ArrowRight, Check, Loader2, Rocket, Search } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { cn } from '@/lib/utils'
import {
  useCreatePmAdhoc,
  useUpdatePmAdhoc,
  usePmAdhocDetail,
} from '@core/hooks/api/usePmAdhoc'
import { useDeviceList } from '@core/hooks/api/useDevices'
import { useIndicatorCandidates } from '@core/hooks/api/usePerformance'
import type { IndicatorCandidate } from '@core/services/api/pmApi'
import type { AdhocDimension, AdhocMode } from '@core/types/pmAdhoc'
import type { DeviceType } from '@core/types/indicatorLibrary'

/**
 * F03 · 自定义聚合任务向导（performance/pm-adhoc/new + /:id/edit）
 * useParams 取 :id → 编辑模式 usePmAdhocDetail 预填；新建/编辑分别走
 * useCreatePmAdhoc / useUpdatePmAdhoc（真实）。设备/指标候选全走真实 hooks。
 */

type WizardTech = 'lte' | 'nr' | 'gsm'

const TECH_TO_DEVICE_TYPE: Record<WizardTech, DeviceType> = {
  lte: 'ENB',
  nr: 'GNB',
  gsm: 'GSM',
}

const TECH_OPTS: { value: WizardTech; label: string }[] = [
  { value: 'lte', label: 'LTE' },
  { value: 'nr', label: 'NR' },
  { value: 'gsm', label: 'GSM' },
]

const DIMENSIONS: { value: AdhocDimension; label: string; hint: string }[] = [
  { value: 'network', label: '全网汇总', hint: '按制式全量聚合成一条总线' },
  { value: 'device_group', label: '按设备组', hint: '每设备组一条线（全量）' },
  { value: 'product', label: '按产品', hint: '每产品一条线（全量）' },
  { value: 'band', label: '按频段', hint: '自动按频段分组' },
  { value: 'device', label: '按设备', hint: '每设备保留一条结果' },
  { value: 'aggregate_group', label: '自选设备组', hint: 'N 个设备临时组聚合一条' },
]

const GRAN_OPTS: { value: string; label: string }[] = [
  { value: '15min', label: '15分钟' },
  { value: 'hourly', label: '小时' },
  { value: 'daily', label: '天' },
]

const STEPS = ['基本信息', '聚合范围', '指标选择', '聚合设置 + 确认']

function toLocalInput(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function defaultWindowStart(): string {
  const d = new Date(Date.now() - 24 * 3600_000)
  return toLocalInput(d.toISOString())
}
function defaultWindowEnd(): string {
  return toLocalInput(new Date().toISOString())
}

export default function PmAdhocWizard() {
  const navigate = useNavigate()
  const { id: editId } = useParams<{ id: string }>()
  const isEdit = Boolean(editId)

  const createMut = useCreatePmAdhoc()
  const updateMut = useUpdatePmAdhoc()
  const { data: editTask, isLoading: editLoading } = usePmAdhocDetail(editId)

  const [step, setStep] = useState(0)

  // ① 基本信息
  const [name, setName] = useState('')
  const [tech, setTech] = useState<WizardTech>('lte')
  const [mode, setMode] = useState<AdhocMode>('oneshot')

  // ② 聚合范围
  const [dimension, setDimension] = useState<AdhocDimension>('network')
  const [selectedSns, setSelectedSns] = useState<string[]>([])
  const [deviceKeyword, setDeviceKeyword] = useState('')

  // ③ 指标
  const [metricPaths, setMetricPaths] = useState<string[]>([])
  const [metricKeyword, setMetricKeyword] = useState('')

  // ④ 聚合设置
  const [granularity, setGranularity] = useState<string>('hourly')
  const [windowStart, setWindowStart] = useState<string>(defaultWindowStart())
  const [windowEnd, setWindowEnd] = useState<string>(defaultWindowEnd())

  const [prefilled, setPrefilled] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)

  // 编辑模式预填一次。
  useEffect(() => {
    if (!isEdit || prefilled || !editTask) return
    setName(editTask.name)
    if (editTask.technology === 'lte' || editTask.technology === 'nr' || editTask.technology === 'gsm') {
      setTech(editTask.technology)
    }
    setMode(editTask.mode)
    setDimension(editTask.dimension)
    setSelectedSns(editTask.deviceSns ?? [])
    setMetricPaths(editTask.metricPaths ?? [])
    if (editTask.granularities?.length) setGranularity(editTask.granularities[0])
    if (editTask.mode === 'oneshot') {
      if (editTask.windowStart) setWindowStart(toLocalInput(editTask.windowStart))
      if (editTask.windowEnd) setWindowEnd(toLocalInput(editTask.windowEnd))
    }
    setPrefilled(true)
  }, [isEdit, prefilled, editTask])

  const deviceType = TECH_TO_DEVICE_TYPE[tech]
  const needsDevicePick = dimension === 'device' || dimension === 'aggregate_group'

  // 设备候选（按制式过滤，仅需挑设备时取）。
  const { data: deviceResp, isLoading: devLoading, isError: devError } = useDeviceList(
    {
      networkType: tech,
      page: 1,
      pageSize: 200,
      ...(deviceKeyword.trim() ? { searchText: deviceKeyword.trim() } : {}),
    },
    { enabled: needsDevicePick },
  )
  const devices = deviceResp?.items ?? []

  // 指标候选（按制式 → deviceType，含计数器）。
  const { data: candidates, isLoading: indLoading, isError: indError } = useIndicatorCandidates(deviceType, {
    includeCounters: true,
  })
  const indicators: IndicatorCandidate[] = useMemo(() => candidates ?? [], [candidates])
  const filteredIndicators = useMemo(() => {
    const kw = metricKeyword.trim().toLowerCase()
    if (!kw) return indicators
    return indicators.filter(
      (i) =>
        i.id.toLowerCase().includes(kw) ||
        i.cnName.toLowerCase().includes(kw) ||
        i.name.toLowerCase().includes(kw),
    )
  }, [indicators, metricKeyword])
  const indicatorById = useMemo(() => {
    const m = new Map<string, IndicatorCandidate>()
    indicators.forEach((i) => m.set(i.id, i))
    return m
  }, [indicators])

  // 校验。
  const step1Valid = name.trim().length > 0
  const step2Valid = needsDevicePick ? selectedSns.length > 0 : true
  const step3Valid = metricPaths.length >= 1
  const step4Valid =
    granularity.length > 0 &&
    (mode === 'continuous' || (Boolean(windowStart) && Boolean(windowEnd) && new Date(windowEnd) > new Date(windowStart)))
  const stepValid = [step1Valid, step2Valid, step3Valid, step4Valid][step]
  const allValid = step1Valid && step2Valid && step3Valid && step4Valid

  const toggleSn = (sn: string) => {
    setSelectedSns((prev) => (prev.includes(sn) ? prev.filter((x) => x !== sn) : [...prev, sn]))
  }
  const toggleMetric = (id: string) => {
    setMetricPaths((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))
  }

  const switchTech = (t: WizardTech) => {
    if (isEdit) return
    setTech(t)
    setSelectedSns([])
    setMetricPaths([])
  }

  const submitting = createMut.isPending || updateMut.isPending

  const submit = () => {
    if (!allValid) return
    setSubmitError(null)
    const wsIso = mode === 'oneshot' && windowStart ? new Date(windowStart).toISOString() : undefined
    const weIso = mode === 'oneshot' && windowEnd ? new Date(windowEnd).toISOString() : undefined
    if (isEdit && editId) {
      updateMut.mutate(
        {
          id: editId,
          input: {
            name: name.trim(),
            deviceSns: needsDevicePick ? selectedSns : [],
            metricPaths,
            granularities: [granularity],
            windowStart: wsIso,
            windowEnd: weIso,
          },
        },
        {
          onSuccess: () => navigate('/performance/pm-adhoc'),
          onError: (e) => setSubmitError(e instanceof Error ? e.message : '更新失败'),
        },
      )
      return
    }
    createMut.mutate(
      {
        name: name.trim(),
        mode,
        dimension,
        technology: tech,
        deviceSns: needsDevicePick ? selectedSns : [],
        metricPaths,
        granularities: [granularity],
        windowStart: wsIso,
        windowEnd: weIso,
      },
      {
        onSuccess: () => navigate('/performance/pm-adhoc'),
        onError: (e) => setSubmitError(e instanceof Error ? e.message : '创建失败'),
      },
    )
  }

  // 编辑模式详情加载中。
  if (isEdit && editLoading && !prefilled) {
    return (
      <PageShell code="F03" title="ADHOC WIZARD · 编辑聚合任务" subtitle="LOADING TASK…" bare>
        <GlassPanel title="LOADING">
          <div className="flex items-center justify-center gap-2 py-20 text-cyan-300/60">
            <Loader2 className="size-5 animate-spin" />
            <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING TASK…</span>
          </div>
        </GlassPanel>
      </PageShell>
    )
  }

  return (
    <PageShell
      code="F03"
      title={isEdit ? 'ADHOC WIZARD · 编辑聚合任务' : 'ADHOC WIZARD · 新建聚合任务'}
      subtitle={`STEP ${step + 1}/${STEPS.length} · ${STEPS[step]}`}
      isFetching={submitting}
      bare
      toolbar={
        <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/performance/pm-adhoc')}>
          返回列表
        </NeonButton>
      }
    >
      {/* 步骤条 */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        {STEPS.map((s, i) => (
          <button
            key={s}
            type="button"
            onClick={() => i <= step && setStep(i)}
            className={cn(
              'chip transition-all',
              i === step
                ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                : i < step
                  ? 'text-[#00ff88]'
                  : 'text-cyan-300/40',
            )}
          >
            {i + 1}. {s}
          </button>
        ))}
      </div>

      {/* 步骤内容 */}
      {step === 0 && (
        <GlassPanel strong title="① BASIC · 基本信息">
          <div className="max-w-xl space-y-4 p-4">
            <Field label="任务名称 · NAME">
              <input
                className="neon-input w-full"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="如：LTE 核心 KPI 小时聚合"
              />
            </Field>
            <Field label="制式 · TECH（建后不可改）">
              <div className="flex gap-1.5">
                {TECH_OPTS.map((t) => (
                  <button
                    key={t.value}
                    type="button"
                    disabled={isEdit}
                    onClick={() => switchTech(t.value)}
                    className={cn(
                      'chip transition-all disabled:opacity-40',
                      tech === t.value
                        ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                        : 'text-cyan-300/55 opacity-70 hover:opacity-100',
                    )}
                  >
                    {t.label}
                  </button>
                ))}
              </div>
            </Field>
            <Field label="持续性 · MODE（建后不可改）">
              <div className="flex gap-1.5">
                {(['oneshot', 'continuous'] as AdhocMode[]).map((m) => (
                  <button
                    key={m}
                    type="button"
                    disabled={isEdit}
                    onClick={() => setMode(m)}
                    className={cn(
                      'chip transition-all disabled:opacity-40',
                      mode === m
                        ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                        : 'text-cyan-300/55 opacity-70 hover:opacity-100',
                    )}
                  >
                    {m === 'oneshot' ? '单次' : '持续'}
                  </button>
                ))}
              </div>
            </Field>
          </div>
        </GlassPanel>
      )}

      {step === 1 && (
        <GlassPanel strong title="② SCOPE · 聚合范围">
          <div className="space-y-4 p-4">
            <Field label="聚合维度 · DIMENSION（建后不可改）">
              <div className="grid grid-cols-2 gap-2 md:grid-cols-3">
                {DIMENSIONS.map((d) => (
                  <button
                    key={d.value}
                    type="button"
                    disabled={isEdit}
                    onClick={() => setDimension(d.value)}
                    className={cn(
                      'rounded-sm border px-3 py-2 text-left transition-all disabled:opacity-40',
                      dimension === d.value
                        ? 'border-cyan-400 bg-cyan-500/10'
                        : 'border-cyan-500/15 hover:border-cyan-500/40',
                    )}
                  >
                    <div className="font-display text-sm font-bold text-cyan-100">{d.label}</div>
                    <div className="mt-0.5 font-mono text-[10px] text-cyan-300/50">{d.hint}</div>
                  </button>
                ))}
              </div>
            </Field>

            {needsDevicePick ? (
              <Field label={`选择设备 · DEVICES（已选 ${selectedSns.length}）`}>
                <div className="relative mb-2 max-w-md">
                  <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
                  <input
                    className="neon-input w-full pl-9"
                    placeholder="SN / 名称 搜索"
                    value={deviceKeyword}
                    onChange={(e) => setDeviceKeyword(e.target.value)}
                  />
                </div>
                <div className="max-h-64 overflow-auto rounded-sm border border-cyan-500/12">
                  {devLoading ? (
                    <Loading text="SCANNING FLEET…" small />
                  ) : devError ? (
                    <ErrorBox text="FLEET SCAN FAILED" small />
                  ) : devices.length === 0 ? (
                    <EmptyBox text="NO DEVICE · 该制式下无设备" small />
                  ) : (
                    devices.map((d) => {
                      const checked = selectedSns.includes(d.sn)
                      return (
                        <button
                          key={d.id}
                          type="button"
                          onClick={() => toggleSn(d.sn)}
                          className={cn(
                            'flex w-full items-center gap-2 border-b border-cyan-500/8 px-2.5 py-1.5 text-left transition-colors hover:bg-cyan-500/5',
                            checked && 'bg-cyan-500/10',
                          )}
                        >
                          <span
                            className={cn(
                              'flex size-3.5 shrink-0 items-center justify-center rounded-[2px] border',
                              checked ? 'border-cyan-400 bg-cyan-400/80' : 'border-cyan-500/40',
                            )}
                          >
                            {checked ? <span className="size-1.5 rounded-[1px] bg-[#03050d]" /> : null}
                          </span>
                          <StatusBadge status={d.isOnline ? 'online' : 'offline'} label={d.isOnline ? 'ON' : 'OFF'} />
                          <div className="min-w-0 flex-1">
                            <div className="truncate font-mono text-[11px] text-cyan-100">{d.sn}</div>
                            <div className="truncate text-[10px] text-cyan-300/55">{d.name || d.deviceName || '—'}</div>
                          </div>
                        </button>
                      )
                    })
                  )}
                </div>
              </Field>
            ) : (
              <div className="rounded-sm border border-cyan-500/15 bg-cyan-500/[0.03] px-3 py-2.5 font-mono text-[11px] text-cyan-300/60">
                该维度按制式全量聚合，无需手动挑设备。
              </div>
            )}
          </div>
        </GlassPanel>
      )}

      {step === 2 && (
        <GlassPanel strong title="③ METRICS · 指标选择" meta={`${metricPaths.length} 项已选`}>
          <div className="p-4">
            <div className="relative mb-3 max-w-md">
              <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
              <input
                className="neon-input w-full pl-9"
                placeholder="指标名 / 编号搜索"
                value={metricKeyword}
                onChange={(e) => setMetricKeyword(e.target.value)}
              />
            </div>
            <div className="max-h-[52vh] overflow-auto rounded-sm border border-cyan-500/12">
              {indLoading ? (
                <Loading text="LOADING CANDIDATES…" small />
              ) : indError ? (
                <ErrorBox text="CANDIDATES LOAD FAILED" small />
              ) : filteredIndicators.length === 0 ? (
                <EmptyBox text="NO INDICATOR" small />
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
                  {filteredIndicators.map((it) => {
                    const checked = metricPaths.includes(it.id)
                    return (
                      <button
                        key={it.id}
                        type="button"
                        onClick={() => toggleMetric(it.id)}
                        className={cn(
                          'flex items-center gap-2 border-b border-r border-cyan-500/8 px-2.5 py-1.5 text-left transition-colors hover:bg-cyan-500/5',
                          checked && 'bg-cyan-500/10',
                        )}
                      >
                        <span
                          className={cn(
                            'flex size-3.5 shrink-0 items-center justify-center rounded-[2px] border',
                            checked ? 'border-cyan-400 bg-cyan-400/80' : 'border-cyan-500/40',
                          )}
                        >
                          {checked ? <span className="size-1.5 rounded-[1px] bg-[#03050d]" /> : null}
                        </span>
                        <div className="min-w-0 flex-1">
                          <div className="truncate text-[11px] text-cyan-100">{it.cnName || it.name}</div>
                          <div className="truncate font-mono text-[9px] text-cyan-300/45">
                            {it.id}
                            {it.isCounter ? ' · CTR' : ' · KPI'}
                          </div>
                        </div>
                      </button>
                    )
                  })}
                </div>
              )}
            </div>
          </div>
        </GlassPanel>
      )}

      {step === 3 && (
        <div className="space-y-3">
          <GlassPanel strong title="④ SETTINGS · 聚合设置">
            <div className="space-y-4 p-4">
              <Field label="粒度 · GRANULARITY">
                <div className="flex gap-1.5">
                  {GRAN_OPTS.map((g) => (
                    <button
                      key={g.value}
                      type="button"
                      onClick={() => setGranularity(g.value)}
                      className={cn(
                        'chip transition-all',
                        granularity === g.value
                          ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                          : 'text-cyan-300/55 opacity-70 hover:opacity-100',
                      )}
                    >
                      {g.label}
                    </button>
                  ))}
                </div>
              </Field>
              {mode === 'oneshot' ? (
                <div className="grid max-w-lg grid-cols-2 gap-3">
                  <Field label="起始时间 · START">
                    <input
                      type="datetime-local"
                      className="neon-input w-full"
                      value={windowStart}
                      onChange={(e) => setWindowStart(e.target.value)}
                    />
                  </Field>
                  <Field label="结束时间 · END">
                    <input
                      type="datetime-local"
                      className="neon-input w-full"
                      value={windowEnd}
                      onChange={(e) => setWindowEnd(e.target.value)}
                    />
                  </Field>
                </div>
              ) : (
                <div className="rounded-sm border border-cyan-500/15 bg-cyan-500/[0.03] px-3 py-2.5 font-mono text-[11px] text-cyan-300/60">
                  持续型任务由后端开窗滚动聚合，无需指定时间范围。
                </div>
              )}
            </div>
          </GlassPanel>

          <GlassPanel title="CONFIRM · 确认">
            <div className="grid grid-cols-2 gap-px bg-cyan-500/8 md:grid-cols-3">
              <Summary label="任务名" value={name || '—'} />
              <Summary label="制式" value={tech.toUpperCase()} />
              <Summary label="模式" value={mode === 'oneshot' ? '单次' : '持续'} />
              <Summary label="维度" value={DIMENSIONS.find((d) => d.value === dimension)?.label ?? dimension} />
              <Summary label="设备数" value={needsDevicePick ? `${selectedSns.length} 台` : '全量'} />
              <Summary label="指标数" value={`${metricPaths.length} 项`} />
              <Summary label="粒度" value={GRAN_OPTS.find((g) => g.value === granularity)?.label ?? granularity} />
              {mode === 'oneshot' && (
                <Summary label="时间窗" value={`${windowStart || '—'} ~ ${windowEnd || '—'}`} />
              )}
            </div>
            {metricPaths.length > 0 ? (
              <div className="flex flex-wrap gap-1.5 border-t border-cyan-500/10 p-3">
                {metricPaths.slice(0, 24).map((id) => (
                  <span key={id} className="chip text-cyan-300/70" title={id}>
                    {indicatorById.get(id)?.cnName || indicatorById.get(id)?.name || id}
                  </span>
                ))}
                {metricPaths.length > 24 ? (
                  <span className="chip text-cyan-300/45">+{metricPaths.length - 24}</span>
                ) : null}
              </div>
            ) : null}
            {submitError ? (
              <div className="m-3 border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">
                {isEdit ? '更新失败' : '创建失败'} · {submitError}
              </div>
            ) : null}
          </GlassPanel>
        </div>
      )}

      {/* 底部导航 */}
      <div className="mt-3 flex items-center justify-between">
        <NeonButton icon={<ArrowLeft />} disabled={step === 0} onClick={() => setStep((s) => Math.max(0, s - 1))}>
          上一步
        </NeonButton>
        {step < STEPS.length - 1 ? (
          <NeonButton
            icon={<ArrowRight />}
            disabled={!stepValid}
            onClick={() => setStep((s) => Math.min(STEPS.length - 1, s + 1))}
          >
            下一步
          </NeonButton>
        ) : (
          <NeonButton
            icon={submitting ? <Loader2 className="animate-spin" /> : isEdit ? <Check /> : <Rocket />}
            disabled={!allValid || submitting}
            onClick={submit}
          >
            {isEdit ? '保存任务' : '创建任务'}
          </NeonButton>
        )}
      </div>
    </PageShell>
  )
}

/* ───────── 复用小件 ───────── */

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="mb-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">{label}</div>
      {children}
    </div>
  )
}

function Summary({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-[#03050d] px-3.5 py-2.5">
      <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">{label}</div>
      <div className="mt-0.5 truncate text-[13px] text-cyan-100" title={value}>
        {value}
      </div>
    </div>
  )
}

function Loading({ text, small }: { text: string; small?: boolean }) {
  return (
    <div className={cn('flex items-center justify-center gap-2 text-cyan-300/60', small ? 'py-6' : 'py-16')}>
      <Loader2 className={cn('animate-spin', small ? 'size-4' : 'size-5')} />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">{text}</span>
    </div>
  )
}

function ErrorBox({ text, small }: { text: string; small?: boolean }) {
  return (
    <div
      className={cn(
        'border border-rose-500/40 bg-rose-500/5 font-mono text-rose-300',
        small ? 'px-3 py-4 text-xs' : 'px-4 py-6 text-sm',
      )}
    >
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
