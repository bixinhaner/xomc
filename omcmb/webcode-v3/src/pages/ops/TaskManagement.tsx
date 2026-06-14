import { useMemo, useState } from 'react'
import { Eye, Loader2, Pause, Play, Plus, Square } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { formatTime } from '@/lib/format'
import type { OpsTask, OpsTemplate } from '@core/mock/data/opsTools'
import {
  useCancelOpsTask,
  useCreateOpsTask,
  useOpsTasks,
  useOpsTemplates,
  usePauseOpsTask,
  useResumeOpsTask,
} from '@core/hooks/api/useOpsTools'

import { FormRow, IconBtn, Pager, StatCard, StateBlock, Toolbar } from './_shared'
import { Modal } from './Modal'
import { TaskDetailDrawer } from './TaskDetailDrawer'

const PAGE_SIZE = 12

const TASK_STATUS_COLOR: Record<OpsTask['status'], string> = {
  pending: '#5b9eff',
  running: '#00f0ff',
  paused: '#ffaa00',
  success: '#00ff88',
  failed: '#ff2d6f',
  cancelled: '#525a78',
}
const TASK_STATUS_LABEL: Record<OpsTask['status'], string> = {
  pending: '待执行',
  running: '执行中',
  paused: '已暂停',
  success: '成功',
  failed: '失败',
  cancelled: '已取消',
}
const TASK_STATUS_VALUES: OpsTask['status'][] = [
  'pending',
  'running',
  'paused',
  'success',
  'failed',
  'cancelled',
]

export default function TaskManagement() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [creator, setCreator] = useState('')
  const [status, setStatus] = useState<OpsTask['status'] | ''>('')

  const [detail, setDetail] = useState<OpsTask | null>(null)
  const [createOpen, setCreateOpen] = useState(false)

  // 概览统计：宽口径拉一页任务用于状态分布与成功率
  const { data: overviewData } = useOpsTasks({ page: 1, pageSize: 200 })
  const overviewTasks = useMemo(() => overviewData?.items ?? [], [overviewData])
  const stat = useMemo(() => {
    const by: Record<OpsTask['status'], number> = {
      pending: 0,
      running: 0,
      paused: 0,
      success: 0,
      failed: 0,
      cancelled: 0,
    }
    overviewTasks.forEach((t) => {
      by[t.status] += 1
    })
    const finished = by.success + by.failed
    const successRate = finished > 0 ? (by.success / finished) * 100 : 0
    return { total: overviewTasks.length, by, successRate }
  }, [overviewTasks])

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(creator.trim() ? { creator: creator.trim() } : {}),
      ...(status ? { status } : {}),
    }),
    [page, keyword, creator, status],
  )

  const { data, isLoading, isError, isFetching, refetch } = useOpsTasks(params, {
    refetchOnMount: 'always',
  })
  const rows = data?.items ?? []
  const total = data?.total ?? 0

  // 模板名映射（关联展示）
  const { data: tplData } = useOpsTemplates({ page: 1, pageSize: 200 })
  const tplNameMap = useMemo(
    () => Object.fromEntries((tplData?.items ?? []).map((t) => [t.id, t.templateName])),
    [tplData],
  )

  const pause = usePauseOpsTask()
  const resume = useResumeOpsTask()
  const cancel = useCancelOpsTask()
  const busy = pause.isPending || resume.isPending || cancel.isPending

  return (
    <PageShell
      code="F06"
      title="OPS · 编队任务"
      subtitle="FLEET TASKS · BATCH MAINTENANCE ORCHESTRATION"
      isFetching={isFetching}
      bare
    >
      <div className="mb-3 grid grid-cols-[repeat(4,1fr)_auto] gap-3">
        <StatCard label="任务总数 · TOTAL" value={stat.total} color="#00f0ff" />
        <StatCard
          label="进行中 · ACTIVE"
          value={stat.by.running + stat.by.pending + stat.by.paused}
          color="#ffaa00"
        />
        <StatCard label="成功 · SUCCESS" value={stat.by.success} color="#00ff88" />
        <StatCard
          label="失败 · FAILED"
          value={stat.by.failed}
          color={stat.by.failed > 0 ? '#ff2d6f' : '#525a78'}
        />
        <div className="glass relative flex items-center justify-center overflow-hidden rounded-sm px-5">
          <div className="scanline" />
          <RadialGauge
            value={stat.successRate}
            label="SUCCESS"
            size={92}
            color={stat.successRate >= 90 ? '#00ff88' : stat.successRate >= 60 ? '#ffaa00' : '#ff2d6f'}
          />
        </div>
      </div>

      <Toolbar
        keyword={keyword}
        keywordPlaceholder="任务名"
        onKeyword={(v) => {
          setKeyword(v)
          setPage(1)
        }}
        isFetching={isFetching}
        onRefresh={() => refetch()}
        extra={
          <NeonButton icon={<Plus />} onClick={() => setCreateOpen(true)}>
            NEW TASK
          </NeonButton>
        }
      >
        <input
          className="neon-input w-40"
          placeholder="发起人"
          value={creator}
          onChange={(e) => {
            setCreator(e.target.value)
            setPage(1)
          }}
        />
        <button
          type="button"
          onClick={() => {
            setStatus('')
            setPage(1)
          }}
          className={`chip text-[#00f0ff] ${status === '' ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55'}`}
        >
          ALL
        </button>
        {TASK_STATUS_VALUES.map((s) => (
          <button
            key={s}
            type="button"
            onClick={() => {
              setStatus(status === s ? '' : s)
              setPage(1)
            }}
            className={`chip ${status === s ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55'}`}
            style={{ color: TASK_STATUS_COLOR[s] }}
          >
            {TASK_STATUS_LABEL[s]}
          </button>
        ))}
      </Toolbar>

      <StateBlock loading={isLoading} error={isError} empty={rows.length === 0} emptyText="NO TASKS · 无任务">
        <div className="overflow-hidden rounded-sm border border-cyan-500/12">
          <div className="grid grid-cols-[2fr_1.2fr_90px_1.4fr_1.2fr_140px] gap-3 border-b border-cyan-500/15 bg-cyan-500/[0.05] px-3 py-2 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/70">
            <span>任务 · TASK</span>
            <span>模板 · PLAYBOOK</span>
            <span>状态</span>
            <span>进度 · PROGRESS</span>
            <span>发起 · CREATED</span>
            <span className="text-right">操作 · ACTIONS</span>
          </div>
          {rows.map((t) => {
            const c = TASK_STATUS_COLOR[t.status]
            return (
              <div
                key={t.id}
                className="fleet-row grid grid-cols-[2fr_1.2fr_90px_1.4fr_1.2fr_140px] items-center gap-3 px-3 py-2.5"
                style={{ ['--row-color' as never]: c }}
              >
                <div className="min-w-0">
                  <div className="truncate font-display text-sm font-bold text-cyan-100">{t.taskName}</div>
                  <div className="font-mono text-[10px] text-cyan-300/55">
                    {t.totalCount} 设备 · {t.creator}
                  </div>
                </div>
                <div className="truncate font-mono text-[11px] text-cyan-300/75">
                  {t.templateId ? tplNameMap[t.templateId] ?? t.templateId : '—'}
                </div>
                <div>
                  <span className="chip" style={{ color: c }}>
                    {TASK_STATUS_LABEL[t.status]}
                  </span>
                </div>
                <div>
                  <div className="h-1 w-full overflow-hidden rounded-full bg-cyan-500/10">
                    <div
                      className="h-full rounded-full"
                      style={{ width: `${t.progress}%`, background: c, boxShadow: `0 0 6px ${c}` }}
                    />
                  </div>
                  <div className="mt-1 font-mono text-[10px] text-cyan-300/55">
                    {t.progress}% · STEP {t.currentStep}/{t.totalSteps}
                    {t.status === 'success' || t.status === 'failed'
                      ? ` · ${t.successCount}✓/${t.failCount}✗`
                      : ''}
                  </div>
                </div>
                <div className="font-mono text-[11px] text-cyan-300/70">{formatTime(t.createdAt)}</div>
                <div className="flex items-center justify-end gap-1.5">
                  {t.status === 'running' ? (
                    <IconBtn title="暂停" color="#ffaa00" disabled={busy} onClick={() => pause.mutate(t.id)}>
                      <Pause className="size-3.5" />
                    </IconBtn>
                  ) : null}
                  {t.status === 'paused' || t.status === 'pending' ? (
                    <IconBtn title="继续" color="#00ff88" disabled={busy} onClick={() => resume.mutate(t.id)}>
                      <Play className="size-3.5" />
                    </IconBtn>
                  ) : null}
                  {t.status === 'running' || t.status === 'paused' || t.status === 'pending' ? (
                    <IconBtn title="取消" color="#ff2d6f" disabled={busy} onClick={() => cancel.mutate(t.id)}>
                      <Square className="size-3.5" />
                    </IconBtn>
                  ) : null}
                  <IconBtn title="详情" color="#00f0ff" onClick={() => setDetail(t)}>
                    <Eye className="size-3.5" />
                  </IconBtn>
                </div>
              </div>
            )
          })}
        </div>
        <Pager page={page} total={total} pageSize={PAGE_SIZE} onPage={setPage} />
      </StateBlock>

      <TaskDetailDrawer
        task={detail}
        open={detail !== null}
        onClose={() => setDetail(null)}
        templateName={detail?.templateId ? tplNameMap[detail.templateId] : undefined}
      />

      <CreateTaskModal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        templates={tplData?.items ?? []}
        onDone={() => refetch()}
      />
    </PageShell>
  )
}

function CreateTaskModal({
  open,
  onClose,
  templates,
  onDone,
}: {
  open: boolean
  onClose: () => void
  templates: OpsTemplate[]
  onDone: () => void
}) {
  const [taskName, setTaskName] = useState('')
  const [templateId, setTemplateId] = useState('')
  const [deviceText, setDeviceText] = useState('')
  const create = useCreateOpsTask()

  const reset = () => {
    setTaskName('')
    setTemplateId('')
    setDeviceText('')
  }

  const submit = () => {
    const deviceSns = deviceText
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean)
    if (!taskName.trim() || deviceSns.length === 0) return
    create.mutate(
      {
        taskName: taskName.trim(),
        templateId: templateId || undefined,
        deviceSns,
        totalSteps: 5,
        totalCount: deviceSns.length,
        creator: 'admin',
      },
      {
        onSuccess: () => {
          reset()
          onClose()
          onDone()
        },
      },
    )
  }

  const deviceCount = deviceText.split('\n').filter((s) => s.trim()).length

  return (
    <Modal
      open={open}
      title="新建编队任务"
      subtitle="CREATE FLEET TASK"
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
            disabled={create.isPending || !taskName.trim() || deviceCount === 0}
            onClick={submit}
          >
            入队执行
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
        <FormRow label="任务名称">
          <input
            className="neon-input w-full"
            placeholder="如：华北区批量巡检"
            value={taskName}
            onChange={(e) => setTaskName(e.target.value)}
          />
        </FormRow>
        <FormRow label="关联模板">
          <div className="flex flex-wrap gap-1.5">
            <button
              type="button"
              onClick={() => setTemplateId('')}
              className={`chip text-[#5b9eff] ${templateId === '' ? 'shadow-[0_0_8px_currentColor]' : 'opacity-55'}`}
            >
              不关联
            </button>
            {templates.map((tpl) => (
              <button
                key={tpl.id}
                type="button"
                onClick={() => setTemplateId(tpl.id)}
                className={`chip text-[#00f0ff] ${templateId === tpl.id ? 'shadow-[0_0_8px_currentColor]' : 'opacity-55'}`}
              >
                {tpl.templateName}
              </button>
            ))}
          </div>
        </FormRow>
        <FormRow label={`目标设备 SN（每行一个 · ${deviceCount}）`}>
          <textarea
            className="neon-input h-28 w-full resize-none"
            placeholder={'ENB00001\nENB00002\nGNB00001'}
            value={deviceText}
            onChange={(e) => setDeviceText(e.target.value)}
          />
        </FormRow>
      </div>
    </Modal>
  )
}
