import { useMemo, useState } from 'react'
import { Eye, Loader2, Plus, Trash2, Wrench } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import type { OpsStep, OpsTemplate } from '@core/mock/data/opsTools'
import {
  useCreateOpsTemplate,
  useDeleteOpsTemplates,
  useOpsTemplates,
} from '@core/hooks/api/useOpsTools'

import { FormRow, IconBtn, Pager, StateBlock, Toolbar } from './_shared'
import { Modal } from './Modal'
import { TemplateDetailDrawer } from './TemplateDetailDrawer'

const TEMPLATE_CATEGORIES = ['巡检运维', '故障处置', '性能优化', '软件管理', '网络配置', '维护操作']
const DEVICE_TYPES = ['eNB', 'gNB', 'CPE', 'eGW']
const PAGE_SIZE = 12

export default function Templates() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [category, setCategory] = useState('')
  const [deviceType, setDeviceType] = useState('')

  const [detail, setDetail] = useState<OpsTemplate | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const [confirmDel, setConfirmDel] = useState<OpsTemplate | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(category ? { category } : {}),
      ...(deviceType ? { targetDeviceType: deviceType } : {}),
    }),
    [page, keyword, category, deviceType],
  )

  const { data, isLoading, isError, isFetching, refetch } = useOpsTemplates(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0

  const del = useDeleteOpsTemplates()

  return (
    <PageShell
      code="F06"
      title="OPS · 动作模板"
      subtitle="OPS PLAYBOOKS · STANDARDIZED MAINTENANCE FLOWS"
      isFetching={isFetching}
      bare
    >
      <Toolbar
        keyword={keyword}
        keywordPlaceholder="模板名"
        onKeyword={(v) => {
          setKeyword(v)
          setPage(1)
        }}
        isFetching={isFetching}
        onRefresh={() => refetch()}
        extra={
          <NeonButton icon={<Plus />} onClick={() => setCreateOpen(true)}>
            NEW PLAYBOOK
          </NeonButton>
        }
      >
        <button
          type="button"
          onClick={() => {
            setCategory('')
            setPage(1)
          }}
          className={`chip text-[#00f0ff] ${category === '' ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55'}`}
        >
          全部分类
        </button>
        {TEMPLATE_CATEGORIES.map((c) => (
          <button
            key={c}
            type="button"
            onClick={() => {
              setCategory(category === c ? '' : c)
              setPage(1)
            }}
            className={`chip text-[#a855f7] ${category === c ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55'}`}
          >
            {c}
          </button>
        ))}
        <span className="mx-1 h-4 w-px self-center bg-cyan-500/20" />
        {DEVICE_TYPES.map((d) => (
          <button
            key={d}
            type="button"
            onClick={() => {
              setDeviceType(deviceType === d ? '' : d)
              setPage(1)
            }}
            className={`chip text-[#5b9eff] ${deviceType === d ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55'}`}
          >
            {d}
          </button>
        ))}
      </Toolbar>

      <StateBlock
        loading={isLoading}
        error={isError}
        empty={rows.length === 0}
        emptyText="NO PLAYBOOKS · 无模板"
      >
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {rows.map((tpl) => (
            <div key={tpl.id} className="glass relative overflow-hidden rounded-sm p-3">
              <div className="scanline" />
              <div className="relative flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <div className="flex items-center gap-1.5">
                    <Wrench className="size-3.5 shrink-0 text-cyan-300/70" />
                    <div className="truncate font-display text-sm font-bold text-cyan-100">
                      {tpl.templateName}
                    </div>
                  </div>
                  <div className="mt-1 flex flex-wrap items-center gap-1">
                    <span className="chip text-[#a855f7]">{tpl.category}</span>
                    {tpl.targetDeviceTypes.map((d) => (
                      <span key={d} className="chip text-[#5b9eff]">
                        {d}
                      </span>
                    ))}
                  </div>
                </div>
              </div>
              <div className="relative mt-2 line-clamp-2 text-[11px] text-cyan-300/65">
                {tpl.description || '—'}
              </div>
              <div className="relative mt-3 flex items-center justify-between border-t border-cyan-500/12 pt-2 font-mono text-[10px] text-cyan-300/55">
                <span>
                  {tpl.steps.length} 步 ·{' '}
                  {tpl.estimatedDuration >= 60
                    ? `${Math.floor(tpl.estimatedDuration / 60)}min`
                    : `${tpl.estimatedDuration}s`}{' '}
                  · 用 {tpl.useCount} 次
                </span>
                <span className="flex items-center gap-1.5">
                  <IconBtn title="详情" color="#00f0ff" onClick={() => setDetail(tpl)}>
                    <Eye className="size-3.5" />
                  </IconBtn>
                  <IconBtn title="删除" color="#ff2d6f" onClick={() => setConfirmDel(tpl)}>
                    <Trash2 className="size-3.5" />
                  </IconBtn>
                </span>
              </div>
            </div>
          ))}
        </div>
        <Pager page={page} total={total} pageSize={PAGE_SIZE} onPage={setPage} />
      </StateBlock>

      <TemplateDetailDrawer template={detail} open={detail !== null} onClose={() => setDetail(null)} />

      <CreateTemplateModal open={createOpen} onClose={() => setCreateOpen(false)} onDone={() => refetch()} />

      <Modal
        open={confirmDel !== null}
        title="删除模板"
        subtitle="DELETE PLAYBOOK"
        width={420}
        onClose={() => setConfirmDel(null)}
        footer={
          <>
            <NeonButton onClick={() => setConfirmDel(null)}>取消</NeonButton>
            <NeonButton
              tone="danger"
              icon={del.isPending ? <Loader2 className="animate-spin" /> : <Trash2 />}
              disabled={del.isPending}
              onClick={() => {
                if (!confirmDel) return
                del.mutate([confirmDel.id], {
                  onSuccess: () => {
                    setConfirmDel(null)
                    refetch()
                  },
                })
              }}
            >
              确认删除
            </NeonButton>
          </>
        }
      >
        <p className="text-[13px] text-cyan-100/85">
          确认删除模板{' '}
          <span className="font-bold text-cyan-50">{confirmDel?.templateName}</span> ？此操作不可撤销。
        </p>
      </Modal>
    </PageShell>
  )
}

function CreateTemplateModal({
  open,
  onClose,
  onDone,
}: {
  open: boolean
  onClose: () => void
  onDone: () => void
}) {
  const [name, setName] = useState('')
  const [category, setCategory] = useState(TEMPLATE_CATEGORIES[0])
  const [deviceTypes, setDeviceTypes] = useState<string[]>([])
  const [description, setDescription] = useState('')
  const [duration, setDuration] = useState('60')
  const create = useCreateOpsTemplate()

  const reset = () => {
    setName('')
    setCategory(TEMPLATE_CATEGORIES[0])
    setDeviceTypes([])
    setDescription('')
    setDuration('60')
  }

  const toggleDev = (d: string) =>
    setDeviceTypes((prev) => (prev.includes(d) ? prev.filter((x) => x !== d) : [...prev, d]))

  const submit = () => {
    if (!name.trim() || deviceTypes.length === 0) return
    const payload: Omit<OpsTemplate, 'id' | 'createTime' | 'updateTime' | 'useCount'> = {
      templateName: name.trim(),
      description: description.trim(),
      category,
      targetDeviceTypes: deviceTypes,
      steps: [] as OpsStep[],
      estimatedDuration: Number(duration) || 60,
      creator: 'admin',
      tags: [],
    }
    create.mutate(payload, {
      onSuccess: () => {
        reset()
        onClose()
        onDone()
      },
    })
  }

  return (
    <Modal
      open={open}
      title="新建动作模板"
      subtitle="CREATE PLAYBOOK"
      onClose={() => {
        reset()
        onClose()
      }}
      footer={
        <>
          <NeonButton
            onClick={() => {
              reset()
              onClose()
            }}
          >
            取消
          </NeonButton>
          <NeonButton
            icon={create.isPending ? <Loader2 className="animate-spin" /> : <Plus />}
            disabled={create.isPending || !name.trim() || deviceTypes.length === 0}
            onClick={submit}
          >
            创建
          </NeonButton>
        </>
      }
    >
      <div className="space-y-4">
        {create.isError ? (
          <div className="border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-[11px] text-rose-300">
            创建失败，请重试
          </div>
        ) : null}
        <FormRow label="模板名称">
          <input
            className="neon-input w-full"
            placeholder="如：基站日常巡检流程"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </FormRow>
        <FormRow label="分类">
          <div className="flex flex-wrap gap-1.5">
            {TEMPLATE_CATEGORIES.map((c) => (
              <button
                key={c}
                type="button"
                onClick={() => setCategory(c)}
                className={`chip text-[#a855f7] ${category === c ? 'shadow-[0_0_8px_currentColor]' : 'opacity-55'}`}
              >
                {c}
              </button>
            ))}
          </div>
        </FormRow>
        <FormRow label="适用设备类型">
          <div className="flex flex-wrap gap-1.5">
            {DEVICE_TYPES.map((d) => (
              <button
                key={d}
                type="button"
                onClick={() => toggleDev(d)}
                className={`chip text-[#5b9eff] ${deviceTypes.includes(d) ? 'shadow-[0_0_8px_currentColor]' : 'opacity-55'}`}
              >
                {d}
              </button>
            ))}
          </div>
        </FormRow>
        <FormRow label="预计耗时（秒）">
          <input
            className="neon-input w-40"
            type="number"
            min={1}
            value={duration}
            onChange={(e) => setDuration(e.target.value)}
          />
        </FormRow>
        <FormRow label="说明">
          <textarea
            className="neon-input h-20 w-full resize-none"
            placeholder="模板用途说明…"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </FormRow>
      </div>
    </Modal>
  )
}
