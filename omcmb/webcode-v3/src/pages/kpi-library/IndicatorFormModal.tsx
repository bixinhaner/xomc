/**
 * IndicatorFormModal — KPI 指标新建/编辑 UI (Issue #535, v3 webcode-v3 / STARFORGE HUD)。
 *
 * 指标全生命周期管理统一进 产品中心→KPI指标库。本 HUD 浮层承载「新建/编辑指标」:
 *   · 字段:中文名(必填) / 英文名(必填) / 归属分组(必填) / 单位 / 统计类型(statisType) /
 *     级别(GNB 无) / 指标类型(直接采集/公式计算) / 公式(仅公式计算类型显示) / 描述。
 *   · 指标类型替代旧「计数器」开关：HUD radio 二选一，选「公式计算」后下方实时展开公式输入。
 *   · 统计类型对齐老 OMC perf_indicators.statis_type 业务：sum/avg/max/min/pct 下拉选择，
 *     驱动后端 G5 cron 聚合（见 omcgo/internal/pm/kpi/calculator.go::AggregateByStatisType）。
 *   · 归属分组 Select 选项来自 useIndicatorGroups(把分组树拍平),必填。
 *   · create 模式:id 由 crypto.randomUUID().replace(/-/g,'') 生成,默认是自定义指标,可选组。
 *   · edit 模式且 indicator.isBuildIn 为真 → 归属分组 Select disabled + 提示
 *     (XML 真相源会覆盖);仅自定义指标可改组。
 *
 * 契约(omcgo/internal/pm/indicator/model.go CreateIndicatorRequest):
 *   必填 en_name / cn_name / group_id(device_type 由 query 注入,前端无需传);
 *   没有 name 字段 → 建/改用 enName + cnName + groupId(不要只传 name)。
 *
 * 复用 frontend-core 同一套 hook 与 i18n key(product.kpi.indicator.*),不引入 Antd。
 */
import type { ReactNode } from 'react'
import { useEffect, useRef, useState } from 'react'
import { X, Loader2, AlertTriangle, Lock, Plus, Pencil, Trash2 } from 'lucide-react'
import type { AxiosError } from 'axios'

import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { useT } from '@/hooks/useT'
import {
  useCreateIndicator,
  useUpdateIndicator,
  useFormulas,
  useUpsertFormula,
  useDeleteFormula,
  usePlatformList,
} from '@core/hooks/api/useIndicatorsLibrary'
import type {
  DeviceType,
  FormulaDraft,
  IndicatorInfo,
  IndicatorTypeValue,
  PlatformFormula,
} from '@core/types/indicatorLibrary'
import {
  STATIS_TYPE_VALUES,
  INDICATOR_UNIT_OPTIONS,
  INDICATOR_LEVEL_OPTIONS,
  INDICATOR_TYPE_OPTIONS,
} from '@core/types/indicatorLibrary'
import GroupTreeSelect from './GroupTreeSelect'

interface Props {
  open: boolean
  onClose: () => void
  deviceType: DeviceType
  operatorCode?: string
  // 详情态（URL ?platform= 锁定）时从父级透传；新建时随 payload 下发给后端，后端
  // 同事务在 perf_formulas_<dt> 写占位行 → 避免详情列表 platform_name EXISTS 过滤掉刚建的指标。
  platform?: string
  // null/undefined → 新建模式;有值 → 编辑模式(预填该行)。
  indicator?: IndicatorInfo | null
}


function errMsg(e: unknown): string {
  const ax = e as AxiosError<{ msg?: string; message?: string }>
  return (
    ax.response?.data?.msg ??
    ax.response?.data?.message ??
    (e instanceof Error ? e.message : String(e))
  )
}

// ── HUD 浮层骨架(本模块自带,v3 无共享 Modal) ───────────────────────────
function HudOverlay({
  title,
  subtitle,
  onClose,
  footer,
  width = 560,
  zIndex = 70,
  children,
}: {
  title: string
  subtitle?: string
  onClose: () => void
  footer?: ReactNode
  width?: number
  zIndex?: number
  children: ReactNode
}) {
  return (
    <div
      className="fixed inset-0 flex items-center justify-center bg-[#02040a]/72 backdrop-blur-sm"
      style={{ zIndex }}
      role="dialog"
      aria-modal="true"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <GlassPanel strong className="warp-in max-h-[88vh] overflow-hidden" style={{ width }}>
        <div className="relative flex items-center justify-between border-b border-cyan-500/15 px-4 py-3">
          <div>
            <div className="font-display text-base text-cyan-100">{title}</div>
            {subtitle ? (
              <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                {subtitle}
              </div>
            ) : null}
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="close"
            className="rounded-sm border border-cyan-500/25 p-1 text-cyan-300/60 transition-colors hover:border-cyan-400/60 hover:bg-cyan-500/10 hover:text-cyan-200"
          >
            <X className="size-4" />
          </button>
        </div>
        <div className="max-h-[64vh] overflow-auto px-4 py-4">{children}</div>
        {footer ? (
          <div className="flex items-center justify-end gap-2 border-t border-cyan-500/15 px-4 py-3">
            {footer}
          </div>
        ) : null}
      </GlassPanel>
    </div>
  )
}

function FieldLabel({ children, required }: { children: ReactNode; required?: boolean }) {
  return (
    <span className="mb-1 block font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/60">
      {children}
      {required ? <span className="ml-1 text-rose-400">*</span> : null}
    </span>
  )
}

/** 括号匹配校验:左右括号成对且不出现先 ) 后 (。返回 true 表示匹配。 */
function bracketsBalanced(formula: string): boolean {
  let depth = 0
  for (const ch of formula) {
    if (ch === '(') depth++
    else if (ch === ')') depth--
    if (depth < 0) return false
  }
  return depth === 0
}

/**
 * 每平台公式 CRUD 区(HUD 风格,与 v1 IndicatorDrawer「全平台公式」对等)。
 * 仅编辑已存在指标时渲染(需要 indicatorId)。真调 useFormulas / useUpsertFormula /
 * useDeleteFormula;删除走手写二次确认行内卡片;新增/编辑走内嵌 platform+formula 编辑器。
 */
function PlatformFormulaSection(
  props:
    | {
        // server 模式(默认/编辑态)— 直接走后端 CRUD。
        mode?: 'server'
        deviceType: DeviceType
        indicatorId: string
      }
    | {
        // local 模式(issue #640 新建态)— drafts 父组件持有,本组件只做 UI 增删改回写,不调任何 API。
        mode: 'local'
        deviceType: DeviceType
        value: FormulaDraft[]
        onChange: (next: FormulaDraft[]) => void
      },
) {
  const t = useT()
  const isLocal = props.mode === 'local'
  // server 模式专用 — local 模式 indicatorId 给 undefined,useFormulas 内部 enabled
  // 守卫不发请求(保 Hook 调用顺序稳定)。
  const serverIndicatorId = isLocal ? undefined : props.indicatorId
  const { data: formulasData, isLoading } = useFormulas(props.deviceType, serverIndicatorId)
  // 平台下拉数据源：后端 GET /api/v1/indicators/platforms 返回该 deviceType 下公式表里 distinct 出来的全部 platform_name。
  const { data: platformsData } = usePlatformList(props.deviceType)
  const upsertMut = useUpsertFormula()
  const deleteMut = useDeleteFormula()

  // 列表数据源:local 模式来自 props.value;server 模式来自 useFormulas。
  const formulas: Array<PlatformFormula | FormulaDraft> = isLocal
    ? props.value
    : (formulasData?.items ?? [])

  // 平台选项：ALL 常驻顶部（对应后端 indicator.PlatformAll 约定、表示跨平台共用），
  // 其余从后端返回的 distinct 名单中拼接。
  const platformOptions: string[] = (() => {
    const fromApi = platformsData?.items ?? []
    const ordered = ['ALL', ...fromApi.filter((p) => p !== 'ALL')]
    const seen = new Set<string>()
    return ordered.filter((p) => {
      if (seen.has(p)) return false
      seen.add(p)
      return true
    })
  })()

  // 编辑器:'create' | 编辑中的 platformName | null(关闭)。
  const [editorMode, setEditorMode] = useState<'create' | string | null>(null)
  const [platform, setPlatform] = useState('')
  const [formula, setFormula] = useState('')
  const [editorError, setEditorError] = useState<string | null>(null)
  // 删除二次确认:正在确认删除的 platformName。
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)
  const [rowError, setRowError] = useState<string | null>(null)

  const isEditingExisting = editorMode !== null && editorMode !== 'create'

  const openCreate = () => {
    setEditorMode('create')
    setPlatform('')
    setFormula('')
    setEditorError(null)
  }
  const openEdit = (row: PlatformFormula | FormulaDraft) => {
    setEditorMode(row.platformName)
    setPlatform(row.platformName)
    setFormula(row.formula)
    setEditorError(null)
  }
  const closeEditor = () => {
    setEditorMode(null)
    setPlatform('')
    setFormula('')
    setEditorError(null)
  }

  const handleSaveFormula = async () => {
    const p = platform.trim()
    const f = formula.trim()
    if (!p || !f) {
      setEditorError(t('common.required'))
      return
    }
    if (!bracketsBalanced(f)) {
      setEditorError(t('product.kpi.formulaBracketMismatch'))
      return
    }
    setEditorError(null)
    if (isLocal) {
      // local 模式:同 platform 已存在则覆盖(与 server upsert 同语义),否则追加。
      const exists = props.value.some((d) => d.platformName === p)
      const next = exists
        ? props.value.map((d) => (d.platformName === p ? { platformName: p, formula: f } : d))
        : [...props.value, { platformName: p, formula: f }]
      props.onChange(next)
      closeEditor()
      return
    }
    try {
      await upsertMut.mutateAsync({
        deviceType: props.deviceType,
        indicatorId: props.indicatorId,
        platform: p,
        formula: f,
      })
      closeEditor()
    } catch (e) {
      setEditorError(errMsg(e))
    }
  }

  const handleDelete = async (platformName: string) => {
    setRowError(null)
    if (isLocal) {
      // local 模式:从 drafts 数组按 platformName 删除。
      props.onChange(props.value.filter((d) => d.platformName !== platformName))
      setConfirmDelete(null)
      return
    }
    try {
      await deleteMut.mutateAsync({
        deviceType: props.deviceType,
        indicatorId: props.indicatorId,
        platform: platformName,
      })
      setConfirmDelete(null)
    } catch (e) {
      setRowError(errMsg(e))
    }
  }

  return (
    <div className="mt-2 border-t border-cyan-500/15 pt-4">
      <div className="mb-2 flex items-center justify-between">
        <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/60">
          {t('product.kpi.indicator.formulasTitle')}
        </span>
        <NeonButton
          icon={<Plus className="size-3.5" />}
          onClick={openCreate}
          disabled={editorMode !== null}
        >
          {t('product.kpi.newFormula')}
        </NeonButton>
      </div>

      {isLoading ? (
        <div className="flex items-center gap-2 py-3 font-mono text-xs text-cyan-300/50">
          <Loader2 className="size-3.5 animate-spin" />
        </div>
      ) : formulas.length === 0 && editorMode !== 'create' ? (
        <div className="py-3 text-center font-mono text-[11px] text-cyan-300/40">—</div>
      ) : (
        <div className="space-y-1.5">
          {formulas.map((row) => (
            <div
              key={row.platformName}
              className="rounded-sm border border-cyan-500/15 bg-cyan-950/20 px-3 py-2"
            >
              <div className="flex items-center justify-between gap-2">
                <span className="rounded-sm border border-cyan-400/40 bg-cyan-500/10 px-1.5 py-0.5 font-mono text-[10px] text-cyan-200">
                  {row.platformName}
                </span>
                <div className="flex items-center gap-1.5">
                  <NeonButton
                    icon={<Pencil className="size-3" />}
                    onClick={() => openEdit(row)}
                    disabled={editorMode !== null}
                    aria-label={t('common.edit')}
                  />
                  <NeonButton
                    tone="danger"
                    icon={<Trash2 className="size-3" />}
                    onClick={() => {
                      setRowError(null)
                      setConfirmDelete(row.platformName)
                    }}
                    disabled={editorMode !== null}
                    aria-label={t('common.delete')}
                  />
                </div>
              </div>
              <code className="mt-1 block break-all font-mono text-[11px] text-cyan-100/80">
                {row.formula}
              </code>

              {confirmDelete === row.platformName ? (
                <div className="mt-2 rounded-sm border border-rose-500/40 bg-rose-500/5 px-2.5 py-2">
                  <div className="font-mono text-[11px] text-rose-200">
                    {t('product.kpi.confirmDeletePlatformFormula', { name: row.platformName })}
                  </div>
                  <div className="mt-1.5 flex items-center justify-end gap-1.5">
                    <NeonButton
                      onClick={() => setConfirmDelete(null)}
                      disabled={!isLocal && deleteMut.isPending}
                    >
                      {t('common.cancel')}
                    </NeonButton>
                    <NeonButton
                      tone="danger"
                      onClick={() => void handleDelete(row.platformName)}
                      disabled={!isLocal && deleteMut.isPending}
                    >
                      {!isLocal && deleteMut.isPending ? <Loader2 className="size-3 animate-spin" /> : null}
                      {t('common.delete')}
                    </NeonButton>
                  </div>
                </div>
              ) : null}
            </div>
          ))}
        </div>
      )}

      {rowError ? (
        <div className="mt-2 flex items-start gap-2 rounded-sm border border-rose-500/40 bg-rose-500/5 px-3 py-2 text-rose-300">
          <AlertTriangle className="mt-0.5 size-4 shrink-0" />
          <div className="font-mono text-xs">{rowError}</div>
        </div>
      ) : null}

      {editorMode !== null ? (
        <div className="mt-3 rounded-sm border border-cyan-500/25 bg-cyan-950/30 px-3 py-3">
          <div className="mb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/70">
            {isEditingExisting
              ? t('product.kpi.editFormulaTitle', { name: editorMode })
              : t('product.kpi.newPlatformFormula')}
          </div>
          <label className="block">
            <FieldLabel required>{t('product.kpi.indicator.platformName')}</FieldLabel>
            {/* 平台下拉：ALL = 对所有平台共用（同指标在具体平台另有公式时以具体平台为准）。
                编辑现有公式时 platform 是主键不可改 → disabled；老数据 platform 不在下拉集里时额外
                补一个 option 以避免丢带。 */}
            <select
              className="neon-input w-full disabled:cursor-not-allowed disabled:opacity-50"
              value={platform}
              disabled={isEditingExisting}
              onChange={(e) => setPlatform(e.target.value)}
            >
              <option value="" disabled>
                {t('product.kpi.indicator.platformName')}
              </option>
              {platform && !platformOptions.includes(platform) ? (
                <option value={platform}>{platform}</option>
              ) : null}
              {platformOptions.map((p) => (
                <option key={p} value={p}>
                  {p === 'ALL' ? `${p}${t('product.kpi.platformAllSuffix')}` : p}
                </option>
              ))}
            </select>
            <span className="mt-1 block font-mono text-[10px] text-cyan-300/45">
              {t('product.kpi.platformExtra')}
            </span>
          </label>
          <label className="mt-3 block">
            <FieldLabel required>{t('product.kpi.indicator.formulaLabel')}</FieldLabel>
            <textarea
              className="neon-input w-full"
              rows={3}
              value={formula}
              placeholder={t('product.kpi.indicator.formulaPh')}
              onChange={(e) => setFormula(e.target.value)}
            />
            <span className="mt-1 block font-mono text-[10px] text-cyan-300/45">
              {t('product.kpi.formulaExtra')}
            </span>
          </label>
          {editorError ? (
            <div className="mt-2 flex items-start gap-2 rounded-sm border border-rose-500/40 bg-rose-500/5 px-3 py-2 text-rose-300">
              <AlertTriangle className="mt-0.5 size-4 shrink-0" />
              <div className="font-mono text-xs">{editorError}</div>
            </div>
          ) : null}
          <div className="mt-3 flex items-center justify-end gap-2">
            <NeonButton onClick={closeEditor} disabled={!isLocal && upsertMut.isPending}>
              {t('common.cancel')}
            </NeonButton>
            <NeonButton onClick={() => void handleSaveFormula()} disabled={!isLocal && upsertMut.isPending}>
              {!isLocal && upsertMut.isPending ? <Loader2 className="size-3 animate-spin" /> : null}
              {t('common.save')}
            </NeonButton>
          </div>
        </div>
      ) : null}
    </div>
  )
}

export default function IndicatorFormModal({
  open,
  onClose,
  deviceType,
  operatorCode,
  platform,
  indicator: propIndicator,
}: Props) {
  const t = useT()

  // currentIndicator 本地态：新建起点 null，create 成功后注入返回值 → 转「编辑态」。
  // 这样新建与编辑公式维护方式完全一致（都用 PlatformFormulaSection）。
  const [currentIndicator, setCurrentIndicator] = useState<IndicatorInfo | null>(propIndicator ?? null)
  // drafts(issue #640 C 方案)— 新建态 + kpi 类型的本地公式草稿,提交时随 createIndicator
  // 一并下发,后端事务原子写入。
  const [drafts, setDrafts] = useState<FormulaDraft[]>([])
  // openedInCreateRef — 本次开弹是否新建态(propIndicator==null),锁定本次生命周期不变。
  // 与 v1/v2 同口径:新建态下公式区立刻可见(local 模式),不必等保存。
  const openedInCreateRef = useRef<boolean>(propIndicator == null)
  const isEdit = Boolean(currentIndicator)
  // 内置指标(is_build_in==='1')编辑时归属分组只读 — XML 真相源会覆盖,改了也无效。
  const builtinGroupReadonly = isEdit && Boolean(currentIndicator?.isBuildIn)

  // 归属分组下拉改用 GroupTreeSelect(真·树形,可展开收起),列出该制式全部分组
  // (不按 platform 过滤),数据源由组件内部 useIndicatorGroups 获取。

  const createMut = useCreateIndicator()
  const updateMut = useUpdateIndicator()

  const [cnName, setCnName] = useState('')
  const [enName, setEnName] = useState('')
  const [groupId, setGroupId] = useState('')
  const [unit, setUnit] = useState('')
  const [statisType, setStatisType] = useState('')
  const [indicatorLevel, setIndicatorLevel] = useState('')
  // 指标类型（替代旧 isCounter Switch）：默认 kpi，选 kpi 后 保存下 转编辑态 PlatformFormulaSection 出现。
  const [indicatorType, setIndicatorType] = useState<IndicatorTypeValue>('kpi')
  const [description, setDescription] = useState('')
  const [formError, setFormError] = useState<string | null>(null)

  // open 切换 / 目标指标变化时同步本地态 + 表单初值。
  useEffect(() => {
    if (!open) return
    setFormError(null)
    openedInCreateRef.current = propIndicator == null
    setCurrentIndicator(propIndicator ?? null)
    if (propIndicator) {
      setCnName(propIndicator.cnName ?? '')
      setEnName(propIndicator.enName ?? '')
      setGroupId(propIndicator.groupId ?? '')
      setUnit(propIndicator.unit ?? '')
      // 编辑预填统计类型，后端 BackendIndicator.statis_type 原始透传。
      setStatisType(propIndicator.statisType ?? '')
      setIndicatorLevel(propIndicator.indicatorLevel ?? '')
      // 按后端 isCounter 字段反推指标类型。
      setIndicatorType(propIndicator.isCounter ? 'counter' : 'kpi')
      setDescription(propIndicator.description ?? '')
    } else {
      setCnName('')
      setEnName('')
      setGroupId('')
      setUnit('')
      setStatisType('')
      setIndicatorLevel('')
      // 新建默认 kpi（公式计算），老 OMC 自定义指标几乎都是派生 KPI。
      setIndicatorType('kpi')
      setDescription('')
      // 新建态每次开弹清空 drafts(避免上次未提交的草稿污染本次)。
      setDrafts([])
    }
  }, [open, propIndicator])

  if (!open) return null

  const handleSubmit = async () => {
    const cn = cnName.trim()
    const en = enName.trim()
    if (!cn) {
      setFormError(t('product.kpi.indicator.cnNameRequired'))
      return
    }
    if (!en) {
      setFormError(t('product.kpi.indicator.enNameRequired'))
      return
    }
    // 内置指标归属由 XML 决定,不校验/不下发 group_id。
    if (!builtinGroupReadonly && !groupId) {
      setFormError(t('product.kpi.indicator.groupRequired'))
      return
    }
    setFormError(null)
    // 指标类型 → isCounter 映射（counter='1' / kpi='0'）。arithmetic 不再由本表单携带 —
    // 公式统一走下方 PlatformFormulaSection（perf_formulas_<dt> 多平台 CRUD）。
    const isCounterFlag: '0' | '1' = indicatorType === 'counter' ? '1' : '0'
    try {
      if (currentIndicator) {
        const updated = await updateMut.mutateAsync({
          deviceType,
          id: currentIndicator.id,
          input: {
            cnName: cn,
            enName: en,
            // 内置指标归属由 XML 决定,不下发 group_id(避免无效写)。
            ...(builtinGroupReadonly ? {} : { groupId }),
            unit: unit.trim() || undefined,
            statisType: statisType || undefined,
            indicatorLevel: indicatorLevel.trim() || undefined,
            isCounter: isCounterFlag,
            cnDescription: description.trim() || undefined,
            enDescription: description.trim() || undefined,
          },
        })
        setCurrentIndicator(updated)
      } else {
        // C 方案(issue #640):kpi 类型新建必须至少配 1 条公式。
        if (indicatorType === 'kpi' && drafts.length === 0) {
          setFormError(t('product.kpi.indicator.formulaAtLeastOne'))
          return
        }
        const created = await createMut.mutateAsync({
          deviceType,
          input: {
            id: crypto.randomUUID().replace(/-/g, ''),
            // 后端 payload 映射不发 name(只认 en_name/cn_name/group_id);
            // name 仅为满足前端类型契约,取中文名占位。
            name: cn,
            cnName: cn,
            enName: en,
            groupId,
            unit: unit.trim() || undefined,
            statisType: statisType || undefined,
            indicatorLevel: indicatorLevel.trim() || undefined,
            isCounter: isCounterFlag,
            cnDescription: description.trim() || undefined,
            enDescription: description.trim() || undefined,
            operatorCode,
            // kpi 类型走 formulas(真实公式集合,事务原子写入);counter 类型保留旧的
            // platform 占位逻辑(详情列表 platform_name EXISTS 过滤需要关联行)。
            ...(indicatorType === 'kpi' && drafts.length > 0
              ? { formulas: drafts }
              : { platform: platform || undefined }),
          },
        })
        // 创建成功 → 公式已经原子写入,直接关弹回详情列表。
        setCurrentIndicator(created)
        onClose()
      }
    } catch (e) {
      setFormError(errMsg(e))
    }
  }

  const saving = createMut.isPending || updateMut.isPending
  const isGnb = deviceType === 'GNB'

  return (
    <HudOverlay
      title={
        isEdit
          ? t('product.kpi.indicator.editTitle')
          : t('product.kpi.indicator.createTitle')
      }
      subtitle="INDICATOR LIFECYCLE · CREATE / EDIT"
      onClose={onClose}
      footer={
        <>
          <NeonButton onClick={onClose} disabled={saving}>
            {currentIndicator ? t('common.close') : t('common.cancel')}
          </NeonButton>
          <NeonButton onClick={() => void handleSubmit()} disabled={saving}>
            {saving ? <Loader2 className="size-3.5 animate-spin" /> : null}
            {t('common.save')}
          </NeonButton>
        </>
      }
    >
      <div className="space-y-4">
        <label className="block">
          <FieldLabel required>{t('product.kpi.indicator.cnNameLabel')}</FieldLabel>
          <input
            className="neon-input w-full"
            value={cnName}
            maxLength={128}
            autoFocus
            onChange={(e) => setCnName(e.target.value)}
          />
        </label>

        <label className="block">
          <FieldLabel required>{t('product.kpi.indicator.enNameLabel')}</FieldLabel>
          <input
            className="neon-input w-full"
            value={enName}
            maxLength={128}
            onChange={(e) => setEnName(e.target.value)}
          />
        </label>

        <label className="block">
          <FieldLabel required>{t('product.kpi.indicator.groupLabel')}</FieldLabel>
          <GroupTreeSelect
            deviceType={deviceType}
            operatorCode={operatorCode}
            value={groupId || undefined}
            onChange={(v) => setGroupId(v ?? '')}
            disabled={builtinGroupReadonly}
            placeholder="—"
          />
          {builtinGroupReadonly ? (
            <span className="mt-1 flex items-center gap-1 font-mono text-[10px] text-amber-300/80">
              <Lock className="size-3" />
              {t('product.kpi.indicator.builtinGroupReadonly')}
            </span>
          ) : null}
        </label>

        <div className="grid grid-cols-2 gap-3">
          <label className="block">
            <FieldLabel>{t('product.kpi.indicator.unitLabel')}</FieldLabel>
            {/* Unit 下拉 — 选项集见 frontend-core INDICATOR_UNIT_OPTIONS。老数据值不在集合时以
                额外 option 允许回显（原生 select 容忍未列出的 value，用户选其他后不可逆则会丢）。 */}
            <select
              className="neon-input w-full"
              value={unit}
              onChange={(e) => setUnit(e.target.value)}
            >
              <option value="">—</option>
              {INDICATOR_UNIT_OPTIONS.map((v) => (
                <option key={v} value={v}>{v}</option>
              ))}
            </select>
          </label>
          <label className="block">
            <FieldLabel>{t('product.kpi.indicator.statisTypeLabel')}</FieldLabel>
            <select
              className="neon-input w-full"
              value={statisType}
              onChange={(e) => setStatisType(e.target.value)}
            >
              <option value="">—</option>
              {STATIS_TYPE_VALUES.map((v) => (
                <option key={v} value={v}>
                  {t(`product.kpi.indicator.statisType.${v}`)}
                </option>
              ))}
            </select>
          </label>
        </div>

        {!isGnb ? (
          <label className="block">
            <FieldLabel>{t('product.kpi.indicator.levelLabel')}</FieldLabel>
            {/* Level 下拉 — 仅 Device / PLMN；老数据 'both' 允许回显（不作为可选项）。 */}
            <select
              className="neon-input w-full"
              value={indicatorLevel}
              onChange={(e) => setIndicatorLevel(e.target.value)}
            >
              <option value="">—</option>
              {INDICATOR_LEVEL_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </select>
          </label>
        ) : null}

        <div>
          <FieldLabel>{t('product.kpi.indicator.typeLabel')}</FieldLabel>
          {/* HUD radio 二选一（替代旧「计数器」Switch）— button 梧棭状选中高亮。 */}
          <div role="radiogroup" className="grid grid-cols-2 gap-2">
            {INDICATOR_TYPE_OPTIONS.map((o) => {
              const labelKey = o.value === 'counter' ? 'typeCounter' : 'typeKpi'
              const hintKey = o.value === 'counter' ? 'typeCounterHint' : 'typeKpiHint'
              const selected = indicatorType === o.value
              return (
                <button
                  key={o.value}
                  type="button"
                  role="radio"
                  aria-checked={selected}
                  onClick={() => setIndicatorType(o.value)}
                  className={
                    'flex flex-col items-start gap-1 rounded-sm border px-3 py-2 text-left transition-colors ' +
                    (selected
                      ? 'border-cyan-400/70 bg-cyan-500/15 text-cyan-100 shadow-[0_0_10px_rgba(0,240,255,0.15)]'
                      : 'border-cyan-500/20 bg-cyan-950/30 text-cyan-300/70 hover:border-cyan-400/40 hover:bg-cyan-500/5')
                  }
                >
                  <span className="font-mono text-xs uppercase tracking-wider">
                    {t(`product.kpi.indicator.${labelKey}`)}
                  </span>
                  <span className="font-mono text-[10px] leading-snug text-cyan-300/55">
                    {t(`product.kpi.indicator.${hintKey}`)}
                  </span>
                </button>
              )
            })}
          </div>
        </div>

        {indicatorType === 'kpi' ? (
          // 公式维护区(issue #640 C 方案):
          //   · 新建态(openedInCreateRef=true)→ local 模式,drafts 父组件持有,提交时
          //     随 createIndicator 一并下发,后端事务原子写入。
          //   · 编辑态(openedInCreateRef=false 且 currentIndicator!=null)→ server 模式,
          //     直接走 useFormulas/useUpsertFormula/useDeleteFormula。
          openedInCreateRef.current ? (
            <PlatformFormulaSection
              mode="local"
              deviceType={deviceType}
              value={drafts}
              onChange={setDrafts}
            />
          ) : currentIndicator ? (
            <PlatformFormulaSection
              mode="server"
              deviceType={deviceType}
              indicatorId={currentIndicator.id}
            />
          ) : null
        ) : null}

        <label className="block">
          <FieldLabel>{t('product.kpi.indicator.descLabel')}</FieldLabel>
          <textarea
            className="neon-input w-full"
            rows={3}
            maxLength={512}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </label>

        {formError ? (
          <div className="flex items-start gap-2 rounded-sm border border-rose-500/40 bg-rose-500/5 px-3 py-2 text-rose-300">
            <AlertTriangle className="mt-0.5 size-4 shrink-0" />
            <div className="font-mono text-xs">{formError}</div>
          </div>
        ) : null}

        {/* 每平台公式 CRUD 上移至 indicatorType='kpi' && currentIndicator 条件区，这里不再重复渲染。 */}
      </div>
    </HudOverlay>
  )
}
