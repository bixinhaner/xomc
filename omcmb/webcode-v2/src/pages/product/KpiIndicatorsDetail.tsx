import { useEffect, useMemo, useState } from 'react'
import { ArrowLeft, Pencil, Plus, RefreshCcw, Search, Trash2, X } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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
} from '@core/hooks/api/useIndicatorsLibrary'
import type {
  DeviceType,
  IndicatorInfo,
  PlatformFormula,
} from '@core/types/indicatorLibrary'

import GroupTreeSelect from './GroupTreeSelect'
import { useT } from '@/hooks/useT'
import { cn } from '@/lib/utils'

// ============================================================
// KPI 指标详情页（二级，v2 webcode-v2，Issue #535）。
// 业务深度对齐 v1（webcode/.../IndicatorTab + IndicatorFormModal）：
//   · 列出 (制式, 平台) 下指标：ID / 中文名 / 英文名 / 分组 / 级别 / 启用。
//   · 搜索（ID/名称）、分组筛选（useIndicatorGroups）、启用开关、删除（二次确认）。
//   · 新建 / 编辑表单（shadcn Dialog）：cnName/enName/groupId 必填 + unit/dataType/
//     level/isCounter/description。内置指标（isBuildIn）编辑时归属分组只读（XML 真相源覆盖）。
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
  indicator,
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
  const isEdit = Boolean(indicator)
  // 内置指标（is_build_in==='1'）编辑时归属分组只读 — XML 真相源会覆盖，改了也无效。
  const builtinGroupReadonly = isEdit && Boolean(indicator?.isBuildIn)

  const [cnName, setCnName] = useState(indicator?.cnName ?? '')
  const [enName, setEnName] = useState(indicator?.enName ?? '')
  const [groupId, setGroupId] = useState(indicator?.groupId ?? '')
  const [unit, setUnit] = useState(indicator?.unit ?? '')
  const [dataType, setDataType] = useState(indicator?.counterType ?? '')
  const [level, setLevel] = useState(indicator?.indicatorLevel ?? '')
  const [isCounter, setIsCounter] = useState(Boolean(indicator?.isCounter))
  const [description, setDescription] = useState(indicator?.description ?? '')
  const [errors, setErrors] = useState<{ cnName?: boolean; enName?: boolean; groupId?: boolean }>({})
  const [err, setErr] = useState<string | null>(null)

  const createMut = useCreateIndicator()
  const updateMut = useUpdateIndicator()
  const pending = createMut.isPending || updateMut.isPending

  // 弹窗打开后聚焦首个字段（不影响表单状态初值）。
  useEffect(() => {
    setErr(null)
  }, [indicator])

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
    if (indicator) {
      updateMut.mutate(
        {
          deviceType,
          id: indicator.id,
          input: {
            cnName: cn,
            enName: en,
            // 内置指标归属由 XML 决定，不下发 group_id（避免无效写）。
            ...(builtinGroupReadonly ? {} : { groupId }),
            unit: unit.trim() || undefined,
            dataType: dataType.trim() || undefined,
            indicatorLevel: showLevel ? level.trim() || undefined : undefined,
            isCounter: isCounter ? '1' : '0',
            cnDescription: desc,
            enDescription: desc,
          },
        },
        {
          onSuccess: () => onDone(t('product.kpi.indicator.updateSuccess')),
          onError: (e) => setErr(errMsg(e)),
        }
      )
    } else {
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
            dataType: dataType.trim() || undefined,
            indicatorLevel: showLevel ? level.trim() || undefined : undefined,
            isCounter: isCounter ? '1' : '0',
            cnDescription: desc,
            enDescription: desc,
            operatorCode,
            // 详情态锁定的 platform，后端同事务写占位 formula。
            platform: platform || undefined,
          },
        },
        {
          onSuccess: () => onDone(t('product.kpi.indicator.createSuccess')),
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
              <Input
                id="ind-unit"
                value={unit}
                maxLength={64}
                onChange={(e) => setUnit(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="ind-dt">{t('product.kpi.indicator.dataTypeLabel')}</Label>
              <Input
                id="ind-dt"
                value={dataType}
                maxLength={64}
                onChange={(e) => setDataType(e.target.value)}
              />
            </div>
          </div>

          {showLevel ? (
            <div className="space-y-1.5">
              <Label htmlFor="ind-level">{t('product.kpi.indicator.levelLabel')}</Label>
              <Input
                id="ind-level"
                value={level}
                maxLength={64}
                onChange={(e) => setLevel(e.target.value)}
              />
            </div>
          ) : null}

          <div className="flex items-center justify-between">
            <Label htmlFor="ind-counter">{t('product.kpi.indicator.isCounterLabel')}</Label>
            <Toggle
              checked={isCounter}
              onChange={setIsCounter}
              label={t('product.kpi.indicator.isCounterLabel')}
            />
          </div>

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

          {/* 每平台公式 CRUD —— 仅编辑已存在指标时出现（需要 indicatorId）；新建态不渲染。对齐 v1 IndicatorDrawer 全平台公式区。 */}
          {indicator ? (
            <FormulaSection deviceType={deviceType} indicatorId={indicator.id} />
          ) : null}
        </div>

        <div className="mt-5 flex justify-end gap-2">
          <Button variant="outline" size="sm" onClick={onClose}>
            {t('common.cancel')}
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
//   · useFormulas 列出 platformName → formula；每行 编辑 / 删除（删除二次确认 AlertDialog）。
//   · 新增 / 编辑：platform + formula 两输入 → useUpsertFormula（按 platform 覆盖）。
//   · 前端做括号匹配校验，后端做完整语法验证（对齐 v1）。i18n key 复用 v1。
function checkBrackets(formula: string): boolean {
  let depth = 0
  for (const ch of formula) {
    if (ch === '(') depth++
    else if (ch === ')') depth--
    if (depth < 0) return false
  }
  return depth === 0
}

function FormulaSection({
  deviceType,
  indicatorId,
}: {
  deviceType: DeviceType
  indicatorId: string
}) {
  const t = useT()
  const { data, isLoading } = useFormulas(deviceType, indicatorId)
  const upsertMut = useUpsertFormula()
  const deleteMut = useDeleteFormula()

  const formulas = useMemo<PlatformFormula[]>(() => data?.items ?? [], [data])

  // 编辑器状态：editing=正在编辑的平台公式；creating=新增态。两者互斥。
  const [editing, setEditing] = useState<PlatformFormula | null>(null)
  const [creating, setCreating] = useState(false)
  const [platform, setPlatform] = useState('')
  const [formula, setFormula] = useState('')
  const [fErr, setFErr] = useState<string | null>(null)
  const [pendingDelete, setPendingDelete] = useState<PlatformFormula | null>(null)

  const editorOpen = creating || Boolean(editing)

  const openCreate = () => {
    setEditing(null)
    setCreating(true)
    setPlatform('')
    setFormula('')
    setFErr(null)
  }

  const openEdit = (row: PlatformFormula) => {
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
    if (upsertMut.isPending) return
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
    upsertMut.mutate(
      { deviceType, indicatorId, platform: p, formula: f },
      {
        onSuccess: () => closeEditor(),
        onError: (e) => setFErr(errMsg(e)),
      }
    )
  }

  const confirmDelete = () => {
    if (!pendingDelete) return
    deleteMut.mutate(
      { deviceType, indicatorId, platform: pendingDelete.platformName },
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
            <Input
              id="formula-platform"
              value={platform}
              maxLength={128}
              disabled={Boolean(editing)}
              onChange={(e) => setPlatform(e.target.value)}
            />
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
                disabled={deleteMut.isPending}
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
