/**
 * GroupsManageModal — KPI 指标分组(功能集)维护 UI (Issue #525, v3 webcode-v3 / STARFORGE HUD)。
 *
 * 一个 HUD 风格浮层列出某制式下的全部指标分组,支持新建/编辑/删除:
 *   · 新建分组 → 内嵌表单,id 由前端 crypto.randomUUID() 生成,name 必填。
 *   · 编辑/删除 → 内置组(isBuildIn,后端 is_build_in==='1')禁用 + Tooltip 提示。
 *   · 删除走手写确认浮层二次确认;成功后 hook 自带 invalidate,列表自动刷新。
 *
 * 复用 frontend-core 同一套 hook 与 i18n key(product.kpi.group.*),不引入 Antd。
 */
import type { ReactNode } from 'react'
import { useMemo, useState } from 'react'
import { X, Plus, Pencil, Trash2, Loader2, AlertTriangle } from 'lucide-react'
import type { AxiosError } from 'axios'

import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useT } from '@/hooks/useT'
import {
  useIndicatorGroups,
  useCreateGroup,
  useUpdateGroup,
  useDeleteGroup,
} from '@core/hooks/api/useIndicatorsLibrary'
import type { DeviceType, IndicatorGroup } from '@core/types/indicatorLibrary'
import GroupTreeSelect from './GroupTreeSelect'

interface Props {
  open: boolean
  onClose: () => void
  deviceType: DeviceType
  operatorCode?: string
}

type FormState =
  | { mode: 'create' }
  | { mode: 'edit'; group: IndicatorGroup }
  | null


// 按 parent_id 树深度优先展开,保留层级顺序并带 depth(用于缩进展示树形)。
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
  width = 720,
  zIndex = 60,
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

export default function GroupsManageModal({
  open,
  onClose,
  deviceType,
  operatorCode,
}: Props) {
  const t = useT()
  // 分组管理列出该制式全部分组(不按 platform 过滤),否则新建的空分组不在平台关联里会看不到。
  const { data, isLoading } = useIndicatorGroups(deviceType, operatorCode)
  const createMut = useCreateGroup()
  const updateMut = useUpdateGroup()
  const deleteMut = useDeleteGroup()

  const [formState, setFormState] = useState<FormState>(null)
  const [name, setName] = useState('')
  const [parentId, setParentId] = useState<string>('')
  const [description, setDescription] = useState('')
  const [formError, setFormError] = useState<string | null>(null)
  // 待删除的分组(手写二次确认浮层)
  const [pendingDelete, setPendingDelete] = useState<IndicatorGroup | null>(null)
  const [banner, setBanner] = useState<string | null>(null)

  const rows = useMemo(() => flattenWithDepth(data?.items), [data])

  // 父分组下拉改用 GroupTreeSelect(真·树形,可展开收起);编辑时通过
  // excludeId 排除自身及子树,避免成环。数据源由组件内部获取。
  const editingId = formState?.mode === 'edit' ? formState.group.id : undefined

  if (!open) return null

  const openCreate = () => {
    setName('')
    setParentId('')
    setDescription('')
    setFormError(null)
    setFormState({ mode: 'create' })
  }

  const openEdit = (group: IndicatorGroup) => {
    setName(group.name ?? '')
    setParentId(group.parentId ?? '')
    setDescription(group.description ?? '')
    setFormError(null)
    setFormState({ mode: 'edit', group })
  }

  const closeForm = () => {
    setFormState(null)
    setFormError(null)
  }

  const submitForm = async () => {
    const trimmed = name.trim()
    if (!trimmed) {
      setFormError(t('product.kpi.group.nameRequired'))
      return
    }
    if (!formState) return
    try {
      if (formState.mode === 'create') {
        await createMut.mutateAsync({
          deviceType,
          input: {
            id: crypto.randomUUID().replace(/-/g, ''),
            name: trimmed,
            // 未选父分组 → '0'(后端 buildTree 约定的顶层 root),即新建平级/顶层节点。
            parentId: parentId || '0',
            description: description || undefined,
            operatorCode,
          },
        })
        setBanner(t('product.kpi.group.createSuccess'))
      } else {
        await updateMut.mutateAsync({
          deviceType,
          id: formState.group.id,
          input: {
            name: trimmed,
            parentId: parentId || undefined,
            description: description || undefined,
          },
        })
        setBanner(t('product.kpi.group.updateSuccess'))
      }
      closeForm()
    } catch (e) {
      setFormError(errMsg(e))
    }
  }

  const confirmDelete = async () => {
    if (!pendingDelete) return
    try {
      await deleteMut.mutateAsync({ deviceType, id: pendingDelete.id })
      setBanner(t('product.kpi.group.deleteSuccess'))
      setPendingDelete(null)
    } catch (e) {
      setBanner(errMsg(e))
      setPendingDelete(null)
    }
  }

  const saving = createMut.isPending || updateMut.isPending

  return (
    <>
      <HudOverlay
        title={t('product.kpi.group.manage')}
        subtitle="INDICATOR GROUP CATALOG · CRUD"
        onClose={onClose}
        footer={
          <NeonButton onClick={onClose}>CLOSE</NeonButton>
        }
      >
        <div className="mb-3 flex items-center justify-between">
          <NeonButton icon={<Plus />} onClick={openCreate}>
            {t('product.kpi.group.createTitle')}
          </NeonButton>
          {banner ? (
            <span className="font-mono text-[11px] text-emerald-300/80">{banner}</span>
          ) : null}
        </div>

        {/* 列头 */}
        {rows.length > 0 && (
          <div className="mb-1 grid grid-cols-[2fr_0.7fr_140px] items-center gap-3 px-3 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
            <span>{t('product.kpi.group.nameLabel')}</span>
            <span>{t('common.builtin')}</span>
            <span className="text-right">{t('common.operation')}</span>
          </div>
        )}

        <div className="space-y-1.5">
          {isLoading ? (
            <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
              <Loader2 className="size-4 animate-spin" />
              <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
            </div>
          ) : rows.length === 0 ? (
            <div className="py-12 text-center font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
              {t('product.kpi.group.empty')}
            </div>
          ) : (
            rows.map(({ g, depth }) => {
              const builtin = Boolean(g.isBuildIn)
              return (
                <div
                  key={g.id}
                  className="fleet-row grid grid-cols-[2fr_0.7fr_140px] items-center gap-3 rounded-sm px-3 py-2.5"
                  style={{ ['--row-color' as never]: builtin ? '#5b9eff' : '#00ff88' }}
                >
                  <div className="min-w-0" style={{ paddingLeft: depth * 16 }}>
                    <div className="truncate text-sm font-bold text-cyan-100" title={g.name || g.id}>
                      {g.name || g.id}
                    </div>
                    {g.description ? (
                      <div className="truncate font-mono text-[10px] text-cyan-300/55" title={g.description}>
                        {g.description}
                      </div>
                    ) : null}
                  </div>
                  <div>
                    {builtin ? (
                      <StatusBadge status="active" label={t('common.builtin')} className="scale-90" />
                    ) : (
                      <span className="text-cyan-300/40">—</span>
                    )}
                  </div>
                  <div className="flex justify-end gap-2">
                    <span title={builtin ? t('product.kpi.group.builtinNoEdit') : undefined}>
                      <NeonButton
                        icon={<Pencil />}
                        disabled={builtin}
                        onClick={() => openEdit(g)}
                      >
                        {t('common.edit')}
                      </NeonButton>
                    </span>
                    <span title={builtin ? t('product.kpi.group.builtinNoDelete') : undefined}>
                      <NeonButton
                        tone="danger"
                        icon={<Trash2 />}
                        disabled={builtin || deleteMut.isPending}
                        onClick={() => setPendingDelete(g)}
                      >
                        {t('common.delete')}
                      </NeonButton>
                    </span>
                  </div>
                </div>
              )
            })
          )}
        </div>
      </HudOverlay>

      {/* 新建 / 编辑 表单浮层 */}
      {formState ? (
        <HudOverlay
          title={
            formState.mode === 'edit'
              ? t('product.kpi.group.editTitle')
              : t('product.kpi.group.createTitle')
          }
          onClose={closeForm}
          width={560}
          zIndex={70}
          footer={
            <>
              <NeonButton onClick={closeForm} disabled={saving}>
                {t('common.cancel')}
              </NeonButton>
              <NeonButton onClick={() => void submitForm()} disabled={saving}>
                {saving ? <Loader2 className="size-3.5 animate-spin" /> : null}
                {t('common.save')}
              </NeonButton>
            </>
          }
        >
          <div className="space-y-4">
            <label className="block">
              <span className="mb-1 block font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/60">
                {t('product.kpi.group.nameLabel')}
              </span>
              <input
                className="neon-input w-full"
                value={name}
                maxLength={128}
                autoFocus
                onChange={(e) => setName(e.target.value)}
              />
            </label>

            <label className="block">
              <span className="mb-1 block font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/60">
                {t('product.kpi.group.parentLabel')}
              </span>
              <GroupTreeSelect
                deviceType={deviceType}
                operatorCode={operatorCode}
                value={parentId || undefined}
                onChange={(v) => setParentId(v ?? '')}
                excludeId={editingId}
                allowClear
                placeholder="—"
              />
            </label>

            <label className="block">
              <span className="mb-1 block font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/60">
                {t('product.kpi.group.descriptionLabel')}
              </span>
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
          </div>
        </HudOverlay>
      ) : null}

      {/* 删除二次确认浮层(v3 无 Popconfirm,手写) */}
      {pendingDelete ? (
        <HudOverlay
          title={t('common.delete')}
          onClose={() => setPendingDelete(null)}
          width={440}
          zIndex={80}
          footer={
            <>
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
            </>
          }
        >
          <div className="flex items-start gap-3">
            <AlertTriangle className="mt-0.5 size-5 shrink-0 text-amber-300" />
            <div className="space-y-1">
              <div className="text-sm text-cyan-100">
                {pendingDelete.name || pendingDelete.id}
              </div>
              <div className="font-mono text-xs text-cyan-300/70">
                {t('product.kpi.group.confirmDelete')}
              </div>
            </div>
          </div>
        </HudOverlay>
      ) : null}
    </>
  )
}
