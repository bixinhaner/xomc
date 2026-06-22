import { useMemo, useState } from 'react'
import { Pencil, Plus, Trash2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
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
  useIndicatorGroups,
  useCreateGroup,
  useUpdateGroup,
  useDeleteGroup,
} from '@core/hooks/api/useIndicatorsLibrary'
import type { DeviceType, IndicatorGroup } from '@core/types/indicatorLibrary'

import GroupTreeSelect from './GroupTreeSelect'
import { useT } from '@/hooks/useT'

// ============================================================
// KPI 指标分组(功能集)维护弹窗 (v2 webcode-v2, Issue #525)。
// 业务深度对齐 v1 (webcode/.../GroupsManageModal.tsx)：
//   · 列出某制式下全部分组(树拍平为一维)，新建 / 编辑 / 删除。
//   · 新建 id 由前端 crypto.randomUUID() 生成；name 必填。
//   · 内置组 (isBuildIn，后端 is_build_in==='1') 禁删禁编辑：按钮 disabled + title 提示。
//   · 删除走二次确认子弹窗。所有文案走 t()，复用 S1 已加的 product.kpi.group.* key。
// 设计语言：shadcn / Tailwind，弹窗用 fixed 覆盖层（对齐 v2 StandardParamDialog）。
// ============================================================

const DEVICE_TYPE_OPTIONS: { value: DeviceType; label: string }[] = [
  { value: 'ENB', label: 'ENB (LTE)' },
  { value: 'GSM', label: 'GSM' },
  { value: 'GNB', label: 'GNB (5G NR)' },
]

// 按 parent_id 树做深度优先展开,保留层级顺序并带 depth(用于缩进展示树形)。
function flattenWithDepth(
  nodes: IndicatorGroup[] | undefined,
  depth = 0,
): { g: IndicatorGroup; depth: number }[] {
  const out: { g: IndicatorGroup; depth: number }[] = []
  ;(nodes || []).forEach((n) => {
    out.push({ g: n, depth })
    if (n.children) out.push(...flattenWithDepth(n.children, depth + 1))
  })
  return out
}

function errMsg(e: unknown): string {
  return e instanceof Error ? e.message : String(e)
}

interface Props {
  open: boolean
  onClose: () => void
  initialDeviceType?: DeviceType
}

type FormState =
  | { mode: 'create' }
  | { mode: 'edit'; group: IndicatorGroup }
  | null

export function KpiGroupsDialog({ open, onClose, initialDeviceType = 'ENB' }: Props) {
  const t = useT()

  const [deviceType, setDeviceType] = useState<DeviceType>(initialDeviceType)
  const [formState, setFormState] = useState<FormState>(null)
  const [name, setName] = useState('')
  const [parentId, setParentId] = useState<string>('')
  const [description, setDescription] = useState('')
  const [nameErr, setNameErr] = useState(false)
  const [toast, setToast] = useState<{ kind: 'ok' | 'err'; msg: string } | null>(null)
  const [pendingDelete, setPendingDelete] = useState<IndicatorGroup | null>(null)

  const { data, isLoading } = useIndicatorGroups(deviceType)
  const createMut = useCreateGroup()
  const updateMut = useUpdateGroup()
  const deleteMut = useDeleteGroup()

  const rows = useMemo(() => flattenWithDepth(data?.items), [data])

  // 父分组下拉改用 GroupTreeSelect(真·树形,可展开收起);编辑时通过 excludeId
  // 排除自身及子树,避免成环。数据源由组件内部 useIndicatorGroups 获取。
  const editingId = formState?.mode === 'edit' ? formState.group.id : undefined

  if (!open) return null

  const closeForm = () => {
    setFormState(null)
    setName('')
    setParentId('')
    setDescription('')
    setNameErr(false)
  }

  const openCreate = () => {
    setName('')
    setParentId('')
    setDescription('')
    setNameErr(false)
    setFormState({ mode: 'create' })
  }

  const openEdit = (group: IndicatorGroup) => {
    setName(group.name ?? '')
    setParentId(group.parentId ?? '')
    setDescription(group.description ?? '')
    setNameErr(false)
    setFormState({ mode: 'edit', group })
  }

  const submitForm = () => {
    if (createMut.isPending || updateMut.isPending) return
    if (name.trim() === '') {
      setNameErr(true)
      return
    }
    setToast(null)
    if (formState?.mode === 'create') {
      createMut.mutate(
        {
          deviceType,
          input: {
            id: crypto.randomUUID().replace(/-/g, ''),
            name: name.trim(),
            // 未选父分组 → '0'(后端 buildTree 约定的顶层 root)，即新建平级/顶层节点。
            parentId: parentId || '0',
            description: description.trim() || undefined,
          },
        },
        {
          onSuccess: () => {
            setToast({ kind: 'ok', msg: t('product.kpi.group.createSuccess') })
            closeForm()
          },
          onError: (e) => setToast({ kind: 'err', msg: errMsg(e) }),
        }
      )
    } else if (formState?.mode === 'edit') {
      updateMut.mutate(
        {
          deviceType,
          id: formState.group.id,
          input: {
            name: name.trim(),
            parentId: parentId || undefined,
            description: description.trim() || undefined,
          },
        },
        {
          onSuccess: () => {
            setToast({ kind: 'ok', msg: t('product.kpi.group.updateSuccess') })
            closeForm()
          },
          onError: (e) => setToast({ kind: 'err', msg: errMsg(e) }),
        }
      )
    }
  }

  const confirmDelete = () => {
    if (!pendingDelete) return
    setToast(null)
    deleteMut.mutate(
      { deviceType, id: pendingDelete.id },
      {
        onSuccess: () => {
          setToast({ kind: 'ok', msg: t('product.kpi.group.deleteSuccess') })
          setPendingDelete(null)
        },
        onError: (e) => {
          setToast({ kind: 'err', msg: errMsg(e) })
          setPendingDelete(null)
        },
      }
    )
  }

  const formTitle =
    formState?.mode === 'edit'
      ? t('product.kpi.group.editTitle')
      : t('product.kpi.group.createTitle')

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40" onClick={onClose} aria-hidden />
      <div className="relative flex max-h-[85vh] w-full max-w-3xl flex-col rounded-lg border bg-background p-5 shadow-xl">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold">{t('product.kpi.group.manage')}</h2>
          <Select value={deviceType} onValueChange={(v) => setDeviceType(v as DeviceType)}>
            <SelectTrigger className="w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {DEVICE_TYPE_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {toast ? (
          <div
            className={
              'mt-3 rounded-md border px-3 py-2 text-sm ' +
              (toast.kind === 'ok'
                ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-600'
                : 'border-destructive/40 bg-destructive/10 text-destructive')
            }
          >
            {toast.msg}
          </div>
        ) : null}

        <div className="mt-3 flex justify-end">
          <Button size="sm" onClick={openCreate}>
            <Plus className="size-4" /> {t('product.kpi.group.createTitle')}
          </Button>
        </div>

        <div className="mt-3 overflow-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('product.kpi.group.nameLabel')}</TableHead>
                <TableHead className="w-24">{t('common.builtin')}</TableHead>
                <TableHead className="w-32 text-right">{t('common.operation')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={3} className="py-8 text-center text-sm text-muted-foreground">
                    …
                  </TableCell>
                </TableRow>
              ) : rows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={3} className="py-8 text-center text-sm text-muted-foreground">
                    {t('product.kpi.group.empty')}
                  </TableCell>
                </TableRow>
              ) : (
                rows.map(({ g: row, depth }) => {
                  const builtin = Boolean(row.isBuildIn)
                  return (
                    <TableRow key={row.id}>
                      <TableCell className="font-medium">
                        <span style={{ paddingLeft: depth * 18 }}>{row.name || row.id}</span>
                      </TableCell>
                      <TableCell>
                        {builtin ? <Badge variant="muted">{t('common.builtin')}</Badge> : null}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-1">
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={builtin}
                            title={builtin ? t('product.kpi.group.builtinNoEdit') : t('common.edit')}
                            aria-label={t('common.edit')}
                            onClick={() => openEdit(row)}
                          >
                            <Pencil className="size-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-destructive hover:text-destructive"
                            disabled={builtin}
                            title={
                              builtin ? t('product.kpi.group.builtinNoDelete') : t('common.delete')
                            }
                            aria-label={t('common.delete')}
                            onClick={() => setPendingDelete(row)}
                          >
                            <Trash2 className="size-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })
              )}
            </TableBody>
          </Table>
        </div>

        <div className="mt-4 flex justify-end">
          <Button variant="outline" size="sm" onClick={onClose}>
            {t('common.cancel')}
          </Button>
        </div>
      </div>

      {/* 新建 / 编辑表单子弹窗 */}
      {formState ? (
        <div className="fixed inset-0 z-[60] flex items-center justify-center">
          <div className="absolute inset-0 bg-black/40" onClick={closeForm} aria-hidden />
          <div className="relative w-full max-w-lg rounded-lg border bg-background p-5 shadow-xl">
            <h2 className="text-base font-semibold">{formTitle}</h2>
            <div className="mt-4 space-y-4">
              <div className="space-y-1.5">
                <Label htmlFor="grp-name">{t('product.kpi.group.nameLabel')}</Label>
                <Input
                  id="grp-name"
                  value={name}
                  maxLength={128}
                  autoFocus
                  onChange={(e) => {
                    setName(e.target.value)
                    if (e.target.value.trim() !== '') setNameErr(false)
                  }}
                />
                {nameErr ? (
                  <p className="text-xs text-destructive">
                    {t('product.kpi.group.nameRequired')}
                  </p>
                ) : null}
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="grp-parent">{t('product.kpi.group.parentLabel')}</Label>
                <GroupTreeSelect
                  deviceType={deviceType}
                  id="grp-parent"
                  value={parentId || undefined}
                  onChange={(v) => setParentId(v ?? '')}
                  excludeId={editingId}
                  allowClear
                  placeholder="—"
                />
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="grp-desc">{t('product.kpi.group.descriptionLabel')}</Label>
                <Input
                  id="grp-desc"
                  value={description}
                  maxLength={512}
                  onChange={(e) => setDescription(e.target.value)}
                />
              </div>
            </div>

            <div className="mt-5 flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={closeForm}>
                {t('common.cancel')}
              </Button>
              <Button
                size="sm"
                onClick={submitForm}
                disabled={createMut.isPending || updateMut.isPending}
              >
                {t('common.save')}
              </Button>
            </div>
          </div>
        </div>
      ) : null}

      {/* 删除二次确认子弹窗 */}
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
              {t('product.kpi.group.confirmDelete')}
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

export default KpiGroupsDialog
