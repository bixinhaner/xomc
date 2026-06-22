/**
 * IndicatorFormModal — KPI 指标新建/编辑 UI (Issue #535, v3 webcode-v3 / STARFORGE HUD)。
 *
 * 指标全生命周期管理统一进 产品中心→KPI指标库。本 HUD 浮层承载「新建/编辑指标」:
 *   · 字段:中文名(必填) / 英文名(必填) / 归属分组(必填) / 单位 / 数据类型 /
 *     级别(GNB 无) / 计数器(开关) / 描述。
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
import { useEffect, useState } from 'react'
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
} from '@core/hooks/api/useIndicatorsLibrary'
import type {
  DeviceType,
  IndicatorInfo,
  PlatformFormula,
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
function PlatformFormulaSection({
  deviceType,
  indicatorId,
}: {
  deviceType: DeviceType
  indicatorId: string
}) {
  const t = useT()
  const { data: formulasData, isLoading } = useFormulas(deviceType, indicatorId)
  const upsertMut = useUpsertFormula()
  const deleteMut = useDeleteFormula()

  const formulas: PlatformFormula[] = formulasData?.items ?? []

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
  const openEdit = (row: PlatformFormula) => {
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
    try {
      await upsertMut.mutateAsync({ deviceType, indicatorId, platform: p, formula: f })
      closeEditor()
    } catch (e) {
      setEditorError(errMsg(e))
    }
  }

  const handleDelete = async (platformName: string) => {
    setRowError(null)
    try {
      await deleteMut.mutateAsync({ deviceType, indicatorId, platform: platformName })
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
                      disabled={deleteMut.isPending}
                    >
                      {t('common.cancel')}
                    </NeonButton>
                    <NeonButton
                      tone="danger"
                      onClick={() => void handleDelete(row.platformName)}
                      disabled={deleteMut.isPending}
                    >
                      {deleteMut.isPending ? <Loader2 className="size-3 animate-spin" /> : null}
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
            <input
              className="neon-input w-full disabled:cursor-not-allowed disabled:opacity-50"
              value={platform}
              maxLength={128}
              disabled={isEditingExisting}
              onChange={(e) => setPlatform(e.target.value)}
            />
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
            <NeonButton onClick={closeEditor} disabled={upsertMut.isPending}>
              {t('common.cancel')}
            </NeonButton>
            <NeonButton onClick={() => void handleSaveFormula()} disabled={upsertMut.isPending}>
              {upsertMut.isPending ? <Loader2 className="size-3 animate-spin" /> : null}
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
  indicator,
}: Props) {
  const t = useT()

  const isEdit = Boolean(indicator)
  // 内置指标(is_build_in==='1')编辑时归属分组只读 — XML 真相源会覆盖,改了也无效。
  const builtinGroupReadonly = isEdit && Boolean(indicator?.isBuildIn)

  // 归属分组下拉改用 GroupTreeSelect(真·树形,可展开收起),列出该制式全部分组
  // (不按 platform 过滤),数据源由组件内部 useIndicatorGroups 获取。

  const createMut = useCreateIndicator()
  const updateMut = useUpdateIndicator()

  const [cnName, setCnName] = useState('')
  const [enName, setEnName] = useState('')
  const [groupId, setGroupId] = useState('')
  const [unit, setUnit] = useState('')
  const [dataType, setDataType] = useState('')
  const [indicatorLevel, setIndicatorLevel] = useState('')
  const [isCounter, setIsCounter] = useState(false)
  const [description, setDescription] = useState('')
  const [formError, setFormError] = useState<string | null>(null)

  // open 切换 / 目标指标变化时同步表单初值(编辑预填,新建清空)。
  useEffect(() => {
    if (!open) return
    setFormError(null)
    if (indicator) {
      setCnName(indicator.cnName ?? '')
      setEnName(indicator.enName ?? '')
      setGroupId(indicator.groupId ?? '')
      setUnit(indicator.unit ?? '')
      // 编辑预填数据类型用列表行的 counterType(后端 data_type_label/data_type)。
      setDataType(indicator.counterType ?? '')
      setIndicatorLevel(indicator.indicatorLevel ?? '')
      setIsCounter(Boolean(indicator.isCounter))
      setDescription(indicator.description ?? '')
    } else {
      setCnName('')
      setEnName('')
      setGroupId('')
      setUnit('')
      setDataType('')
      setIndicatorLevel('')
      setIsCounter(false)
      setDescription('')
    }
  }, [open, indicator])

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
    try {
      if (indicator) {
        await updateMut.mutateAsync({
          deviceType,
          id: indicator.id,
          input: {
            cnName: cn,
            enName: en,
            // 内置指标归属由 XML 决定,不下发 group_id(避免无效写)。
            ...(builtinGroupReadonly ? {} : { groupId }),
            unit: unit.trim() || undefined,
            dataType: dataType.trim() || undefined,
            indicatorLevel: indicatorLevel.trim() || undefined,
            isCounter: isCounter ? '1' : '0',
            cnDescription: description.trim() || undefined,
            enDescription: description.trim() || undefined,
          },
        })
      } else {
        await createMut.mutateAsync({
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
            dataType: dataType.trim() || undefined,
            indicatorLevel: indicatorLevel.trim() || undefined,
            isCounter: isCounter ? '1' : '0',
            cnDescription: description.trim() || undefined,
            enDescription: description.trim() || undefined,
            operatorCode,
            // 详情态新建 → 透传 platform，后端同事务写占位 formula（避免“保存后查不到”）。
            platform: platform || undefined,
          },
        })
      }
      onClose()
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
            {t('common.cancel')}
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
            <input
              className="neon-input w-full"
              value={unit}
              maxLength={64}
              onChange={(e) => setUnit(e.target.value)}
            />
          </label>
          <label className="block">
            <FieldLabel>{t('product.kpi.indicator.dataTypeLabel')}</FieldLabel>
            <input
              className="neon-input w-full"
              value={dataType}
              maxLength={64}
              onChange={(e) => setDataType(e.target.value)}
            />
          </label>
        </div>

        {!isGnb ? (
          <label className="block">
            <FieldLabel>{t('product.kpi.indicator.levelLabel')}</FieldLabel>
            <input
              className="neon-input w-full"
              value={indicatorLevel}
              maxLength={64}
              onChange={(e) => setIndicatorLevel(e.target.value)}
            />
          </label>
        ) : null}

        <label className="flex items-center justify-between gap-3">
          <FieldLabel>{t('product.kpi.indicator.isCounterLabel')}</FieldLabel>
          <button
            type="button"
            role="switch"
            aria-checked={isCounter}
            onClick={() => setIsCounter((v) => !v)}
            className={
              'relative h-5 w-10 shrink-0 rounded-full border transition-colors ' +
              (isCounter
                ? 'border-cyan-400/70 bg-cyan-500/30'
                : 'border-cyan-500/25 bg-cyan-950/40')
            }
          >
            <span
              className={
                'absolute top-0.5 size-3.5 rounded-full bg-cyan-200 transition-all ' +
                (isCounter ? 'left-5 shadow-[0_0_6px_#00f0ff]' : 'left-0.5')
              }
            />
          </button>
        </label>

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

        {/* 每平台公式 CRUD:仅编辑已存在指标时显示(需 indicatorId);新建态不渲染。 */}
        {indicator ? (
          <PlatformFormulaSection deviceType={deviceType} indicatorId={indicator.id} />
        ) : null}
      </div>
    </HudOverlay>
  )
}
