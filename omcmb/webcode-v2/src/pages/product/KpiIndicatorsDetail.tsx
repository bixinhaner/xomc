import { useEffect, useMemo, useRef, useState } from 'react'
import { ArrowLeft, Pencil, Plus, RefreshCcw, Search, Trash2, X } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  EmptyRow,
  ErrorRow,
  LoadingRow,
  PageShell,
  TableCard,
} from '@/components/layout/PageShell'

import {
  useIndicatorList,
  useEnabledIndicators,
  useSetEnabledIndicators,
  useDeleteIndicator,
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
import { useT } from '@/hooks/useT'
import { cn } from '@/lib/utils'

// ============================================================
// KPI 指标详情页（二级，v2 webcode-v2，Issue #535）。
// 业务深度对齐 v1（webcode/.../IndicatorTab + IndicatorFormModal）：
//   · 列出 (制式, 平台) 下指标：ID / 中文名 / 英文名 / 分组 / 级别 / 启用。
//   · 搜索（ID/名称）、分组筛选（useIndicatorGroups）、启用开关、删除（二次确认）。
//   · 新建 / 编辑表单（shadcn Dialog）：cnName/enName/groupId 必填 + unit/statisType/
//     level/indicatorType/arithmetic/description。指标类型 Radio 二选一（直接采集 / 公式计算），
//     选「公式计算」后下方实时展开公式输入 — 一次在新建表单里配好公式。统计类型下拉选择 sum/avg/max/min/pct
//     （对齐老 OMC perf_indicators.statis_type）。
//     内置指标（isBuildIn）编辑时归属分组只读（XML 真相源覆盖）。
//   · 契约（model.go CreateIndicatorRequest）：必填 en_name/cn_name/group_id；
//     建/改用 enName + cnName + groupId（无 name 字段，device_type 由 query 注入）。
// 设计语言：shadcn / Tailwind；弹窗用 fixed 覆盖层（对齐 KpiGroupsDialog）。
// ============================================================

// 启用状态走 default 行（XML 真相源；运营商覆盖能力后端保留但 UI 不暴露选择器，对齐 v1）。
const OPERATOR_CODE = 'default'
const PAGE_SIZE = 50


function errMsg(e: unknown): string {
  return e instanceof Error ? e.message : String(e)
}

/** 轻量开关（v2 无 Switch 组件，用受控 button 模拟，对齐 KpiConfig 的复选风格但更像 Switch）。 */
function Toggle({
  checked,
  disabled,
  onChange,
  label,
}: {
  checked: boolean
  disabled?: boolean
  onChange: (next: boolean) => void
  label: string
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        'relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors',
        checked ? 'bg-primary' : 'bg-input',
        disabled && 'cursor-not-allowed opacity-50'
      )}
    >
      <span
        className={cn(
          'inline-block size-4 transform rounded-full bg-background shadow transition-transform',
          checked ? 'translate-x-4' : 'translate-x-0.5'
        )}
      />
    </button>
  )
}

interface Props {
  deviceType: DeviceType
  platform: string
  onBack: () => void
}

type FormState =
  | { mode: 'create' }
  | { mode: 'edit'; indicator: IndicatorInfo }
  | null

export function KpiIndicatorsDetail({ deviceType, platform, onBack }: Props) {
  const t = useT()

  const [keyword, setKeyword] = useState('')
  const [keywordInput, setKeywordInput] = useState('')
  const [groupId, setGroupId] = useState<string>('')
  const [page, setPage] = useState(1)
  const [pendingId, setPendingId] = useState<string | null>(null)
  const [pendingDelete, setPendingDelete] = useState<IndicatorInfo | null>(null)
  const [formState, setFormState] = useState<FormState>(null)
  const [toast, setToast] = useState<{ kind: 'ok' | 'err'; msg: string } | null>(null)

  const { data, isLoading, isError, error, isFetching, refetch } = useIndicatorList(
    deviceType,
    {
      platformName: platform,
      keyword: keyword || undefined,
      groupId: groupId || undefined,
      page,
      pageSize: PAGE_SIZE,
    }
  )

  // 分组筛选 / 归属分组下拉都由 GroupTreeSelect 内部 useIndicatorGroups 获取数据
  // (真·树形,可展开收起),不再需要父层 useMemo + groupTreeOptions 拍平。

  const { data: enabledData } = useEnabledIndicators(deviceType, OPERATOR_CODE)
  const enabledSet = useMemo(() => new Set(enabledData?.items || []), [enabledData])

  const setEnabledMut = useSetEnabledIndicators()
  const deleteMut = useDeleteIndicator()

  const items = useMemo(() => data?.items || [], [data])
  const total = data?.total || 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const showLevel = deviceType !== 'GNB'
  const colCount = showLevel ? 7 : 6

  const toggleEnabled = (id: string, next: boolean) => {
    setPendingId(id)
    setToast(null)
    setEnabledMut.mutate(
      { deviceType, operatorCode: OPERATOR_CODE, indicatorIds: [id], enable: next },
      {
        onSettled: () => setPendingId(null),
        onError: (e) => setToast({ kind: 'err', msg: errMsg(e) }),
      }
    )
  }

  const confirmDelete = () => {
    if (!pendingDelete) return
    setToast(null)
    deleteMut.mutate(
      { deviceType, id: pendingDelete.id },
      {
        onSuccess: () => {
          setToast({ kind: 'ok', msg: t('product.kpi.indicator.deleteSuccess') })
          setPendingDelete(null)
        },
        onError: (e) => {
          setToast({ kind: 'err', msg: errMsg(e) })
          setPendingDelete(null)
        },
      }
    )
  }

  const applySearch = () => {
    setKeyword(keywordInput.trim())
    setPage(1)
  }

  return (
    <PageShell
      title={t('product.kpi.indicatorsByDevice', { deviceType: `${deviceType} · ${platform}` })}
      description={t('common.totalCount', { count: total })}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button variant="ghost" size="sm" onClick={onBack}>
            <ArrowLeft className="size-4" /> {t('common.back')}
          </Button>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-72 pl-9"
              placeholder={t('product.kpi.idNameSearchPh')}
              value={keywordInput}
              onChange={(e) => setKeywordInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') applySearch()
              }}
            />
          </div>
          <Button variant="outline" size="sm" onClick={applySearch}>
            <Search className="size-4" /> {t('common.search')}
          </Button>
          <GroupTreeSelect
            deviceType={deviceType}
            operatorCode={OPERATOR_CODE}
            value={groupId || undefined}
            onChange={(v) => {
              setGroupId(v ?? '')
              setPage(1)
            }}
            allowClear
            placeholder={t('product.kpi.indicator.group')}
            className="w-52"
          />
          {(keyword || groupId) && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setKeyword('')
                setKeywordInput('')
                setGroupId('')
                setPage(1)
              }}
            >
              <X className="size-4" /> {t('common.reset')}
            </Button>
          )}
          <div className="ml-auto flex items-center gap-2">
            <Button size="sm" onClick={() => setFormState({ mode: 'create' })}>
              <Plus className="size-4" /> {t('product.kpi.indicator.createTitle')}
            </Button>
            <Button variant="outline" size="sm" onClick={() => void refetch()}>
              <RefreshCcw className="size-4" /> {t('common.refresh')}
            </Button>
          </div>
        </div>
      }
    >
      {toast ? (
        <div
          className={cn(
            'mb-3 rounded-md border px-3 py-2 text-sm',
            toast.kind === 'ok'
              ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-600'
              : 'border-destructive/40 bg-destructive/10 text-destructive'
          )}
        >
          {toast.msg}
        </div>
      ) : null}

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-36">ID</TableHead>
              <TableHead>{t('common.cnName')}</TableHead>
              <TableHead>{t('common.enName')}</TableHead>
              <TableHead className="w-32">{t('common.group')}</TableHead>
              {showLevel && <TableHead className="w-20">{t('common.level')}</TableHead>}
              <TableHead className="w-20">{t('common.enable')}</TableHead>
              <TableHead className="w-28 text-right">{t('common.action')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={error} />
            ) : items.length === 0 ? (
              <EmptyRow colSpan={colCount}>{t('product.kpi.group.empty')}</EmptyRow>
            ) : (
              items.map((row) => (
                <TableRow key={row.id}>
                  <TableCell className="font-mono text-xs text-muted-foreground">{row.id}</TableCell>
                  <TableCell className="font-medium">{row.cnName || '—'}</TableCell>
                  <TableCell className="text-muted-foreground">{row.enName || '—'}</TableCell>
                  <TableCell>{row.groupName || row.groupId || '—'}</TableCell>
                  {showLevel && <TableCell>{row.indicatorLevel || '—'}</TableCell>}
                  <TableCell>
                    <Toggle
                      checked={enabledSet.has(row.id)}
                      disabled={setEnabledMut.isPending && pendingId === row.id}
                      onChange={(next) => toggleEnabled(row.id, next)}
                      label={t('common.enable')}
                    />
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label={t('common.edit')}
                        title={t('common.edit')}
                        onClick={() => setFormState({ mode: 'edit', indicator: row })}
                      >
                        <Pencil className="size-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive hover:text-destructive"
                        aria-label={t('common.delete')}
                        title={t('common.delete')}
                        onClick={() => setPendingDelete(row)}
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      {total > PAGE_SIZE ? (
        <div className="mt-3 flex items-center justify-end gap-2 text-sm">
          <Button
            variant="outline"
            size="sm"
            disabled={page <= 1}
            onClick={() => setPage((p) => Math.max(1, p - 1))}
          >
            {t('common.prevPage')}
          </Button>
          <span className="tabular-nums text-muted-foreground">
            {page} / {totalPages}
          </span>
          <Button
            variant="outline"
            size="sm"
            disabled={page >= totalPages}
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
          >
            {t('common.nextPage')}
          </Button>
        </div>
      ) : null}

      {formState ? (
        <IndicatorForm
          deviceType={deviceType}
          operatorCode={OPERATOR_CODE}
          // 详情态锁定的 platform 透传 → 后端同事务写占位 formula，
          // 避免“保存后查不到”（详情列表 EXISTS 过滤需要 perf_formulas 存在该 platform 的关联行）。
          platform={platform}
          indicator={formState.mode === 'edit' ? formState.indicator : null}
          onClose={() => setFormState(null)}
          onDone={(msg) => {
            setToast({ kind: 'ok', msg })
            setFormState(null)
          }}
        />
      ) : null}

      {pendingDelete ? (
        <div className="fixed inset-0 z-[60] flex items-center justify-center">
          <div
            className="absolute inset-0 bg-black/40"
            onClick={() => setPendingDelete(null)}
            aria-hidden
          />
          <div className="relative w-full max-w-sm rounded-lg border bg-background p-5 shadow-xl">
            <h2 className="text-base font-semibold">{t('common.delete')}</h2>
            <p className="mt-3 text-sm text-muted-foreground">
              {t('product.kpi.confirmDeleteIndicator', { id: pendingDelete.id })}
            </p>
            <div className="mt-5 flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setPendingDelete(null)}>
                {t('common.cancel')}
              </Button>
              <Button
                size="sm"
                variant="destructive"
                onClick={confirmDelete}
                disabled={deleteMut.isPending}
              >
                {t('common.yes')}
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </PageShell>
  )
}

// ── 新建 / 编辑指标表单（fixed 覆盖层弹窗） ──────────────────────────────────
function IndicatorForm({
  deviceType,
  operatorCode,
  platform,
  indicator: propIndicator,
  onClose,
  onDone,
}: {
  deviceType: DeviceType
  operatorCode?: string
  // 详情态锁定的 platform，新建时随 payload 下发给后端，后端同事务在 perf_formulas_<dt>
  // 写占位行，避免“保存后查不到” bug（详情列表 platform_name EXISTS 过滤需要关联行）。
  platform?: string
  indicator: IndicatorInfo | null
  onClose: () => void
  onDone: (msg: string) => void
}) {
  const t = useT()
  // currentIndicator 本地态：新建起点 null，create 成功后注入返回值 → 转「编辑态」，
  // FormulaSection 自动出现，与修改页面完全一致。
  const [currentIndicator, setCurrentIndicator] = useState<IndicatorInfo | null>(propIndicator)
  // drafts(issue #640 C 方案)— 新建态 + kpi 类型的本地公式草稿,提交时随 createIndicator
  // 一并下发,后端事务原子写入。
  const [drafts, setDrafts] = useState<FormulaDraft[]>([])
  // openedInCreateRef — 本次开弹是否新建态(propIndicator==null),锁定本次生命周期不变。
  // 与 v1 IndicatorFormModal 同口径:新建态下公式区立刻可见(local 模式),不必等保存。
  const openedInCreateRef = useRef<boolean>(propIndicator == null)
  const isEdit = Boolean(currentIndicator)
  // 内置指标（is_build_in==='1'）编辑时归属分组只读 — XML 真相源会覆盖，改了也无效。
  const builtinGroupReadonly = isEdit && Boolean(currentIndicator?.isBuildIn)

  const [cnName, setCnName] = useState(propIndicator?.cnName ?? '')
  const [enName, setEnName] = useState(propIndicator?.enName ?? '')
  const [groupId, setGroupId] = useState(propIndicator?.groupId ?? '')
  const [unit, setUnit] = useState(propIndicator?.unit ?? '')
  const [statisType, setStatisType] = useState(propIndicator?.statisType ?? '')
  const [level, setLevel] = useState(propIndicator?.indicatorLevel ?? '')
  // 指标类型（替代旧 isCounter Toggle）：默认 kpi，选 kpi 后 保存下 转编辑态 FormulaSection 出现。
  const [indicatorType, setIndicatorType] = useState<IndicatorTypeValue>(
    propIndicator?.isCounter ? 'counter' : 'kpi',
  )
  const [description, setDescription] = useState(propIndicator?.description ?? '')
  const [errors, setErrors] = useState<{ cnName?: boolean; enName?: boolean; groupId?: boolean }>({})
  const [err, setErr] = useState<string | null>(null)

  const createMut = useCreateIndicator()
  const updateMut = useUpdateIndicator()
  const pending = createMut.isPending || updateMut.isPending

  // 弹窗打开后聚焦首个字段（不影响表单状态初值）。
  useEffect(() => {
    setErr(null)
  }, [propIndicator])

  const showLevel = deviceType !== 'GNB'

  const submit = () => {
    if (pending) return
    const cn = cnName.trim()
    const en = enName.trim()
    const nextErrors = {
      cnName: cn === '',
      enName: en === '',
      groupId: !builtinGroupReadonly && groupId === '',
    }
    setErrors(nextErrors)
    if (nextErrors.cnName || nextErrors.enName || nextErrors.groupId) return
    setErr(null)

    const desc = description.trim() || undefined
    // 指标类型 → isCounter 映射（counter='1' / kpi='0'）。arithmetic 不再由本表单携带 —
    // 公式统一走下方 FormulaSection（perf_formulas_<dt> 多平台 CRUD）。
    const isCounterFlag: '0' | '1' = indicatorType === 'counter' ? '1' : '0'
    if (currentIndicator) {
      updateMut.mutate(
        {
          deviceType,
          id: currentIndicator.id,
          input: {
            cnName: cn,
            enName: en,
            // 内置指标归属由 XML 决定，不下发 group_id（避免无效写）。
            ...(builtinGroupReadonly ? {} : { groupId }),
            unit: unit.trim() || undefined,
            statisType: statisType || undefined,
            indicatorLevel: showLevel ? level.trim() || undefined : undefined,
            isCounter: isCounterFlag,
            cnDescription: desc,
            enDescription: desc,
          },
        },
        {
          onSuccess: (data) => {
            // 同步本地态（后端可能自动修正 isCounter），并提示成功；不关闭弹窗。
            setCurrentIndicator(data)
            onDone(t('product.kpi.indicator.updateSuccess'))
          },
          onError: (e) => setErr(errMsg(e)),
        }
      )
    } else {
      // C 方案(issue #640):kpi 类型新建必须至少配 1 条公式。
      if (indicatorType === 'kpi' && drafts.length === 0) {
        setErr(t('product.kpi.indicator.formulaAtLeastOne'))
        return
      }
      createMut.mutate(
        {
          deviceType,
          input: {
            id: crypto.randomUUID().replace(/-/g, ''),
            // 后端 payload 映射不发 name（只认 en_name/cn_name/group_id）；
            // name 仅为满足前端类型契约，取中文名占位。
            name: cn,
            cnName: cn,
            enName: en,
            groupId,
            unit: unit.trim() || undefined,
            statisType: statisType || undefined,
            indicatorLevel: showLevel ? level.trim() || undefined : undefined,
            isCounter: isCounterFlag,
            cnDescription: desc,
            enDescription: desc,
            operatorCode,
            // kpi 类型走 formulas(真实公式集合,事务原子写入);counter 类型保留旧
            // platform 占位逻辑(详情列表 platform_name EXISTS 过滤需要关联行)。
            ...(indicatorType === 'kpi' && drafts.length > 0
              ? { formulas: drafts }
              : { platform: platform || undefined }),
          },
        },
        {
          onSuccess: (data) => {
            // 创建成功 → 公式已经原子写入,直接关弹回详情列表(由 onDone 触发刷新)。
            setCurrentIndicator(data)
            onDone(t('product.kpi.indicator.createSuccess'))
          },
          onError: (e) => setErr(errMsg(e)),
        }
      )
    }
  }

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40" onClick={onClose} aria-hidden />
      <div className="relative flex max-h-[88vh] w-full max-w-lg flex-col overflow-auto rounded-lg border bg-background p-5 shadow-xl">
        <h2 className="text-base font-semibold">
          {isEdit
            ? t('product.kpi.indicator.editTitle')
            : t('product.kpi.indicator.createTitle')}
        </h2>

        {err ? (
          <div className="mt-3 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {err}
          </div>
        ) : null}

        <div className="mt-4 space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="ind-cn">{t('product.kpi.indicator.cnNameLabel')}</Label>
            <Input
              id="ind-cn"
              value={cnName}
              maxLength={128}
              autoFocus
              onChange={(e) => {
                setCnName(e.target.value)
                if (e.target.value.trim()) setErrors((s) => ({ ...s, cnName: false }))
              }}
            />
            {errors.cnName ? (
              <p className="text-xs text-destructive">
                {t('product.kpi.indicator.cnNameRequired')}
              </p>
            ) : null}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="ind-en">{t('product.kpi.indicator.enNameLabel')}</Label>
            <Input
              id="ind-en"
              value={enName}
              maxLength={128}
              onChange={(e) => {
                setEnName(e.target.value)
                if (e.target.value.trim()) setErrors((s) => ({ ...s, enName: false }))
              }}
            />
            {errors.enName ? (
              <p className="text-xs text-destructive">
                {t('product.kpi.indicator.enNameRequired')}
              </p>
            ) : null}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="ind-group">{t('product.kpi.indicator.groupLabel')}</Label>
            <GroupTreeSelect
              deviceType={deviceType}
              operatorCode={OPERATOR_CODE}
              id="ind-group"
              value={groupId || undefined}
              disabled={builtinGroupReadonly}
              invalid={errors.groupId}
              placeholder="—"
              onChange={(v) => {
                const next = v ?? ''
                setGroupId(next)
                if (next) setErrors((s) => ({ ...s, groupId: false }))
              }}
            />
            {builtinGroupReadonly ? (
              <p className="text-xs text-muted-foreground">
                {t('product.kpi.indicator.builtinGroupReadonly')}
              </p>
            ) : errors.groupId ? (
              <p className="text-xs text-destructive">
                {t('product.kpi.indicator.groupRequired')}
              </p>
            ) : null}
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="ind-unit">{t('product.kpi.indicator.unitLabel')}</Label>
              {/* Unit 下拉 — 选项集见 frontend-core INDICATOR_UNIT_OPTIONS。 */}
              <Select
                value={unit || undefined}
                onValueChange={(v) => setUnit(v)}
              >
                <SelectTrigger id="ind-unit">
                  <SelectValue placeholder={t('product.kpi.indicator.unitRequired')} />
                </SelectTrigger>
                <SelectContent>
                  {INDICATOR_UNIT_OPTIONS.map((v) => (
                    <SelectItem key={v} value={v}>{v}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="ind-st">{t('product.kpi.indicator.statisTypeLabel')}</Label>
              <Select
                value={statisType || undefined}
                onValueChange={(v) => setStatisType(v)}
              >
                <SelectTrigger id="ind-st">
                  <SelectValue placeholder={t('product.kpi.indicator.statisTypeRequired')} />
                </SelectTrigger>
                <SelectContent>
                  {STATIS_TYPE_VALUES.map((v) => (
                    <SelectItem key={v} value={v}>
                      {t(`product.kpi.indicator.statisType.${v}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {showLevel ? (
            <div className="space-y-1.5">
              <Label htmlFor="ind-level">{t('product.kpi.indicator.levelLabel')}</Label>
              {/* Level 下拉 — 仅 Device / PLMN；老数据 'both' 允许回显但不作为可选项。 */}
              <Select
                value={level || undefined}
                onValueChange={(v) => setLevel(v)}
              >
                <SelectTrigger id="ind-level">
                  <SelectValue placeholder={t('product.kpi.indicator.levelRequired')} />
                </SelectTrigger>
                <SelectContent>
                  {INDICATOR_LEVEL_OPTIONS.map((o) => (
                    <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          ) : null}

          <div className="space-y-1.5">
            <Label>{t('product.kpi.indicator.typeLabel')}</Label>
            {/* 指标类型二选一（替代旧「计数器」Toggle）— 原生 radio + Tailwind，v2 未引 shadcn RadioGroup。 */}
            <div className="flex flex-col gap-2" role="radiogroup" aria-label={t('product.kpi.indicator.typeLabel')}>
              {INDICATOR_TYPE_OPTIONS.map((o) => {
                const labelKey = o.value === 'counter' ? 'typeCounter' : 'typeKpi'
                const hintKey = o.value === 'counter' ? 'typeCounterHint' : 'typeKpiHint'
                return (
                  <label
                    key={o.value}
                    className="flex cursor-pointer items-start gap-2 rounded-md border border-input bg-transparent p-2.5 hover:bg-accent/30"
                  >
                    <input
                      type="radio"
                      name="ind-type"
                      className="mt-0.5 size-4 accent-primary"
                      checked={indicatorType === o.value}
                      onChange={() => setIndicatorType(o.value)}
                    />
                    <span className="flex flex-col gap-0.5">
                      <span className="text-sm font-medium leading-none">
                        {t(`product.kpi.indicator.${labelKey}`)}
                      </span>
                      <span className="text-xs text-muted-foreground">
                        {t(`product.kpi.indicator.${hintKey}`)}
                      </span>
                    </span>
                  </label>
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
              <FormulaSection
                mode="local"
                deviceType={deviceType}
                value={drafts}
                onChange={setDrafts}
              />
            ) : currentIndicator ? (
              <FormulaSection
                mode="server"
                deviceType={deviceType}
                indicatorId={currentIndicator.id}
              />
            ) : null
          ) : null}

          <div className="space-y-1.5">
            <Label htmlFor="ind-desc">{t('product.kpi.indicator.descLabel')}</Label>
            <textarea
              id="ind-desc"
              value={description}
              maxLength={512}
              rows={3}
              onChange={(e) => setDescription(e.target.value)}
              className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            />
          </div>
        </div>

        <div className="mt-5 flex justify-end gap-2">
          <Button variant="outline" size="sm" onClick={onClose}>
            {currentIndicator ? t('common.close') : t('common.cancel')}
          </Button>
          <Button size="sm" onClick={submit} disabled={pending}>
            {t('common.save')}
          </Button>
        </div>
      </div>
    </div>
  )
}

// ── 每平台公式 CRUD（编辑指标弹窗内嵌区，对齐 v1 IndicatorDrawer「全平台公式」） ───────────
//   · server 模式(默认/编辑态)→ useFormulas + useUpsertFormula/useDeleteFormula 直接走后端。
//   · local 模式(issue #640 新建态)→ drafts 父组件持有,本组件只做 UI 增删改回写,不调任何 API。
//   · 新增 / 编辑：platform + formula 两输入。前端做括号匹配校验，后端做完整语法验证。i18n key 复用 v1。
function checkBrackets(formula: string): boolean {
  let depth = 0
  for (const ch of formula) {
    if (ch === '(') depth++
    else if (ch === ')') depth--
    if (depth < 0) return false
  }
  return depth === 0
}

type FormulaSectionProps =
  | {
      mode?: 'server'
      deviceType: DeviceType
      indicatorId: string
    }
  | {
      mode: 'local'
      deviceType: DeviceType
      value: FormulaDraft[]
      onChange: (next: FormulaDraft[]) => void
    }

function FormulaSection(props: FormulaSectionProps) {
  const t = useT()
  const isLocal = props.mode === 'local'
  // server 模式专用 — local 模式 indicatorId 给 undefined 让 useFormulas 内部 enabled
  // 守卫不发请求(保 Hook 调用顺序稳定)。
  const serverIndicatorId = isLocal ? undefined : props.indicatorId
  const { data, isLoading } = useFormulas(props.deviceType, serverIndicatorId)
  // 平台下拉数据源:两模式共用。
  const { data: platformsData } = usePlatformList(props.deviceType)
  const upsertMut = useUpsertFormula()
  const deleteMut = useDeleteFormula()

  // 列表数据源:local 模式来自 props.value;server 模式来自 useFormulas。
  // PlatformFormula 与 FormulaDraft 均含 platformName/formula,union 兼容表格渲染。
  const formulas = useMemo<Array<PlatformFormula | FormulaDraft>>(
    () => (isLocal ? props.value : (data?.items ?? [])),
    [isLocal, props, data],
  )

  // 平台选项：ALL 常驻顶部（对应后端 indicator.PlatformAll 约定、表示跨平台共用），
  // 其余从后端返回的 distinct 名单中拼接。原生 <select> 允许受控外值为老数据回显。
  const platformOptions = useMemo(() => {
    const fromApi = platformsData?.items ?? []
    const ordered = ['ALL', ...fromApi.filter((p) => p !== 'ALL')]
    const seen = new Set<string>()
    return ordered.filter((p) => {
      if (seen.has(p)) return false
      seen.add(p)
      return true
    })
  }, [platformsData])

  // 编辑器状态：editing=正在编辑的平台公式；creating=新增态。两者互斥。
  const [editing, setEditing] = useState<PlatformFormula | FormulaDraft | null>(null)
  const [creating, setCreating] = useState(false)
  const [platform, setPlatform] = useState('')
  const [formula, setFormula] = useState('')
  const [fErr, setFErr] = useState<string | null>(null)
  const [pendingDelete, setPendingDelete] = useState<PlatformFormula | FormulaDraft | null>(null)

  const editorOpen = creating || Boolean(editing)

  const openCreate = () => {
    setEditing(null)
    setCreating(true)
    setPlatform('')
    setFormula('')
    setFErr(null)
  }

  const openEdit = (row: PlatformFormula | FormulaDraft) => {
    setCreating(false)
    setEditing(row)
    setPlatform(row.platformName)
    setFormula(row.formula)
    setFErr(null)
  }

  const closeEditor = () => {
    setEditing(null)
    setCreating(false)
    setPlatform('')
    setFormula('')
    setFErr(null)
  }

  const submit = () => {
    if (!isLocal && upsertMut.isPending) return
    const p = platform.trim()
    const f = formula.trim()
    if (!p || !f) {
      setFErr(t('common.required'))
      return
    }
    if (!checkBrackets(f)) {
      setFErr(t('product.kpi.formulaBracketMismatch'))
      return
    }
    setFErr(null)
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
    upsertMut.mutate(
      { deviceType: props.deviceType, indicatorId: props.indicatorId, platform: p, formula: f },
      {
        onSuccess: () => closeEditor(),
        onError: (e) => setFErr(errMsg(e)),
      }
    )
  }

  const confirmDelete = () => {
    if (!pendingDelete) return
    if (isLocal) {
      // local 模式:从 drafts 数组按 platformName 删除,onChange 通知父级。
      props.onChange(props.value.filter((d) => d.platformName !== pendingDelete.platformName))
      setPendingDelete(null)
      return
    }
    deleteMut.mutate(
      {
        deviceType: props.deviceType,
        indicatorId: props.indicatorId,
        platform: pendingDelete.platformName,
      },
      { onSettled: () => setPendingDelete(null) }
    )
  }

  return (
    <div className="space-y-2 border-t pt-4">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium">{t('product.kpi.indicator.formulasTitle')}</span>
        <Button size="sm" variant="outline" onClick={openCreate}>
          <Plus className="size-4" /> {t('product.kpi.newFormula')}
        </Button>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-40">{t('product.kpi.indicator.platformName')}</TableHead>
              <TableHead>{t('product.kpi.indicator.formulaLabel')}</TableHead>
              <TableHead className="w-24 text-right">{t('common.action')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={3} />
            ) : formulas.length === 0 ? (
              <EmptyRow colSpan={3}>—</EmptyRow>
            ) : (
              formulas.map((row) => (
                <TableRow key={row.platformName}>
                  <TableCell className="font-mono text-xs">{row.platformName}</TableCell>
                  <TableCell className="font-mono text-xs break-all">{row.formula}</TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label={t('common.edit')}
                        title={t('common.edit')}
                        onClick={() => openEdit(row)}
                      >
                        <Pencil className="size-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive hover:text-destructive"
                        aria-label={t('common.delete')}
                        title={t('common.delete')}
                        onClick={() => setPendingDelete(row)}
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* 新增 / 编辑公式编辑器（内嵌，platform 编辑时只读 = 按平台覆盖） */}
      {editorOpen ? (
        <div className="space-y-3 rounded-md border bg-muted/30 p-3">
          <div className="text-sm font-medium">
            {editing
              ? t('product.kpi.editFormulaTitle', { name: editing.platformName })
              : t('product.kpi.newPlatformFormula')}
          </div>
          {fErr ? (
            <div className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-xs text-destructive">
              {fErr}
            </div>
          ) : null}
          <div className="space-y-1.5">
            <Label htmlFor="formula-platform">{t('product.kpi.indicator.platformName')}</Label>
            {/* 平台下拉选择：ALL = 对所有平台共用（同指标在具体平台另有公式时具体平台优先）。
                编辑现有公式时 platform 是主键不可改 → disabled；老数据 platform 不在下拉集里时原生 select
                会透明回显，加一个额外 option 以避免丢带 platform 名。 */}
            <select
              id="formula-platform"
              value={platform}
              disabled={Boolean(editing)}
              onChange={(e) => setPlatform(e.target.value)}
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
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
            <p className="text-xs text-muted-foreground">{t('product.kpi.platformExtra')}</p>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="formula-text">{t('product.kpi.indicator.formulaLabel')}</Label>
            <textarea
              id="formula-text"
              value={formula}
              rows={3}
              placeholder={t('product.kpi.indicator.formulaPh')}
              onChange={(e) => setFormula(e.target.value)}
              className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 font-mono text-xs shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            />
            <p className="text-xs text-muted-foreground">{t('product.kpi.formulaExtra')}</p>
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" onClick={closeEditor}>
              {t('common.cancel')}
            </Button>
            <Button size="sm" onClick={submit} disabled={upsertMut.isPending}>
              {t('common.save')}
            </Button>
          </div>
        </div>
      ) : null}

      {/* 删除二次确认 */}
      {pendingDelete ? (
        <div className="fixed inset-0 z-[70] flex items-center justify-center">
          <div
            className="absolute inset-0 bg-black/40"
            onClick={() => setPendingDelete(null)}
            aria-hidden
          />
          <div className="relative w-full max-w-sm rounded-lg border bg-background p-5 shadow-xl">
            <h2 className="text-base font-semibold">{t('common.delete')}</h2>
            <p className="mt-3 text-sm text-muted-foreground">
              {t('product.kpi.confirmDeletePlatformFormula', { name: pendingDelete.platformName })}
            </p>
            <div className="mt-5 flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setPendingDelete(null)}>
                {t('common.cancel')}
              </Button>
              <Button
                size="sm"
                variant="destructive"
                onClick={confirmDelete}
                disabled={!isLocal && deleteMut.isPending}
              >
                {t('common.yes')}
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  )
}

export default KpiIndicatorsDetail
