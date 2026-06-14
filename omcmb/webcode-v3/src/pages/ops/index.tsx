import { useMemo, useState } from 'react'
import {
  Search,
  RefreshCcw,
  Loader2,
  Plus,
  Play,
  Pause,
  Square,
  Trash2,
  Eye,
  Wrench,
  ListChecks,
  Terminal as TerminalIcon,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { formatTime } from '@/lib/format'
import type { OpsCommandRecord, OpsStep, OpsTask, OpsTemplate } from '@core/mock/data/opsTools'
import {
  useOpsTasks,
  useOpsTemplates,
  useOpsCommandRecords,
  useCreateOpsTask,
  useCreateOpsTemplate,
  useDeleteOpsTemplates,
  usePauseOpsTask,
  useResumeOpsTask,
  useCancelOpsTask,
} from '@core/hooks/api/useOpsTools'

import { Modal } from './Modal'
import { TaskDetailDrawer } from './TaskDetailDrawer'
import { TemplateDetailDrawer } from './TemplateDetailDrawer'
import { CommandDetailDrawer } from './CommandDetailDrawer'

// ============================================================
// 常量
// ============================================================

type Tab = 'tasks' | 'templates' | 'commands'

const TAB_META: { key: Tab; label: string; en: string; icon: typeof Wrench }[] = [
  { key: 'tasks', label: '编队任务', en: 'TASKS', icon: ListChecks },
  { key: 'templates', label: '动作模板', en: 'PLAYBOOKS', icon: Wrench },
  { key: 'commands', label: '指令流水', en: 'COMMANDS', icon: TerminalIcon },
]

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

const TEMPLATE_CATEGORIES = ['巡检运维', '故障处置', '性能优化', '软件管理', '网络配置', '维护操作']
const DEVICE_TYPES = ['eNB', 'gNB', 'CPE', 'eGW']

// ============================================================
// 小组件
// ============================================================

function Center({ children }: { children: React.ReactNode }) {
  return <div className="flex min-h-[180px] w-full items-center justify-center">{children}</div>
}

function StateBlock({
  loading,
  error,
  empty,
  emptyText,
  children,
}: {
  loading: boolean
  error: boolean
  empty: boolean
  emptyText: string
  children: React.ReactNode
}) {
  if (loading) {
    return (
      <Center>
        <span className="flex items-center gap-2 font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/60">
          <Loader2 className="size-4 animate-spin" /> SYNCING…
        </span>
      </Center>
    )
  }
  if (error) {
    return (
      <Center>
        <span className="border border-rose-500/40 bg-rose-500/5 px-4 py-3 font-mono text-sm text-rose-300">
          SYNC FAILED · 数据加载失败
        </span>
      </Center>
    )
  }
  if (empty) {
    return (
      <Center>
        <span className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/40">
          {emptyText}
        </span>
      </Center>
    )
  }
  return <>{children}</>
}

function Pager({
  page,
  total,
  pageSize,
  onPage,
}: {
  page: number
  total: number
  pageSize: number
  onPage: (p: number) => void
}) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  return (
    <div className="mt-3 flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {pageSize}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <NeonButton onClick={() => onPage(Math.max(1, page - 1))} disabled={page <= 1}>
          ◂ PREV
        </NeonButton>
        <NeonButton onClick={() => onPage(Math.min(totalPages, page + 1))} disabled={page >= totalPages}>
          NEXT ▸
        </NeonButton>
      </div>
    </div>
  )
}

// ============================================================
// 主页面
// ============================================================

export function OpsPage() {
  const [tab, setTab] = useState<Tab>('tasks')

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

  return (
    <PageShell
      code="F06"
      title="OPS · 运维工事"
      subtitle="OPS TOOLBOX · PLAYBOOKS / FLEET ACTIONS / COMMAND LOG"
      bare
    >
      {/* 概览统计带 */}
      <div className="mb-3 grid grid-cols-[repeat(4,1fr)_auto] gap-3">
        <StatCard label="任务总数 · TOTAL" value={stat.total} color="#00f0ff" />
        <StatCard
          label="执行中 · RUNNING"
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

      {/* Tab 切换 */}
      <div className="mb-3 flex flex-wrap gap-2">
        {TAB_META.map((m) => {
          const Icon = m.icon
          const active = tab === m.key
          return (
            <button
              key={m.key}
              type="button"
              onClick={() => setTab(m.key)}
              className={`chip text-[#00f0ff] transition-all ${
                active ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55 hover:opacity-100'
              }`}
            >
              <Icon className="size-3" />
              {m.en} · {m.label}
            </button>
          )
        })}
      </div>

      {tab === 'tasks' ? <TasksTab /> : null}
      {tab === 'templates' ? <TemplatesTab /> : null}
      {tab === 'commands' ? <CommandsTab /> : null}
    </PageShell>
  )
}

function StatCard({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3" style={{ borderLeftColor: color }}>
      <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">{label}</div>
      <div
        className="font-display text-3xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}

// ============================================================
// Tab 1 — 编队任务
// ============================================================

function TasksTab() {
  const [page, setPage] = useState(1)
  const pageSize = 12
  const [keyword, setKeyword] = useState('')
  const [creator, setCreator] = useState('')
  const [status, setStatus] = useState<OpsTask['status'] | ''>('')

  const [detail, setDetail] = useState<OpsTask | null>(null)
  const [createOpen, setCreateOpen] = useState(false)

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(creator.trim() ? { creator: creator.trim() } : {}),
      ...(status ? { status } : {}),
    }),
    [page, keyword, creator, status]
  )

  const { data, isLoading, isError, isFetching, refetch } = useOpsTasks(params, {
    refetchOnMount: 'always',
  })
  const rows = data?.items ?? []
  const total = data?.total ?? 0

  // 模板名映射（用于关联展示）
  const { data: tplData } = useOpsTemplates({ page: 1, pageSize: 200 })
  const tplNameMap = useMemo(
    () => Object.fromEntries((tplData?.items ?? []).map((t) => [t.id, t.templateName])),
    [tplData]
  )

  const pause = usePauseOpsTask()
  const resume = useResumeOpsTask()
  const cancel = useCancelOpsTask()

  const busy = pause.isPending || resume.isPending || cancel.isPending

  return (
    <>
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
              setStatus(s)
              setPage(1)
            }}
            className={`chip ${status === s ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55'}`}
            style={{ color: TASK_STATUS_COLOR[s] }}
          >
            {TASK_STATUS_LABEL[s]}
          </button>
        ))}
      </Toolbar>

      <StateBlock
        loading={isLoading}
        error={isError}
        empty={rows.length === 0}
        emptyText="NO TASKS · 无任务"
      >
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
                  <div className="truncate font-display text-sm font-bold text-cyan-100">
                    {t.taskName}
                  </div>
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
                    <IconBtn
                      title="暂停"
                      color="#ffaa00"
                      disabled={busy}
                      onClick={() => pause.mutate(t.id)}
                    >
                      <Pause className="size-3.5" />
                    </IconBtn>
                  ) : null}
                  {t.status === 'paused' || t.status === 'pending' ? (
                    <IconBtn
                      title="继续"
                      color="#00ff88"
                      disabled={busy}
                      onClick={() => resume.mutate(t.id)}
                    >
                      <Play className="size-3.5" />
                    </IconBtn>
                  ) : null}
                  {t.status === 'running' || t.status === 'paused' || t.status === 'pending' ? (
                    <IconBtn
                      title="取消"
                      color="#ff2d6f"
                      disabled={busy}
                      onClick={() => cancel.mutate(t.id)}
                    >
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
        <Pager page={page} total={total} pageSize={pageSize} onPage={setPage} />
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
    </>
  )
}

function IconBtn({
  title,
  color,
  disabled,
  onClick,
  children,
}: {
  title: string
  color: string
  disabled?: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      title={title}
      disabled={disabled}
      onClick={onClick}
      className="rounded-sm border border-cyan-500/25 p-1 transition-colors hover:bg-cyan-500/10 disabled:cursor-not-allowed disabled:opacity-40"
      style={{ color }}
    >
      {children}
    </button>
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
      }
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

function FormRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1.5">
      <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/65">{label}</div>
      {children}
    </div>
  )
}

// ============================================================
// Tab 2 — 动作模板
// ============================================================

function TemplatesTab() {
  const [page, setPage] = useState(1)
  const pageSize = 12
  const [keyword, setKeyword] = useState('')
  const [category, setCategory] = useState('')
  const [deviceType, setDeviceType] = useState('')

  const [detail, setDetail] = useState<OpsTemplate | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const [confirmDel, setConfirmDel] = useState<OpsTemplate | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(category ? { category } : {}),
      ...(deviceType ? { targetDeviceType: deviceType } : {}),
    }),
    [page, keyword, category, deviceType]
  )

  const { data, isLoading, isError, isFetching, refetch } = useOpsTemplates(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0

  const del = useDeleteOpsTemplates()

  return (
    <>
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
              setCategory(c)
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
                  <div className="truncate font-display text-sm font-bold text-cyan-100">
                    {tpl.templateName}
                  </div>
                  <div className="mt-0.5 flex flex-wrap items-center gap-1">
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
        <Pager page={page} total={total} pageSize={pageSize} onPage={setPage} />
      </StateBlock>

      <TemplateDetailDrawer
        template={detail}
        open={detail !== null}
        onClose={() => setDetail(null)}
      />

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
    </>
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

// ============================================================
// Tab 3 — 指令流水
// ============================================================

function CommandsTab() {
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [keyword, setKeyword] = useState('')
  const [deviceSn, setDeviceSn] = useState('')
  const [operator, setOperator] = useState('')
  const [result, setResult] = useState<'' | 'success' | 'failed'>('')

  const [detail, setDetail] = useState<OpsCommandRecord | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
      ...(operator.trim() ? { operator: operator.trim() } : {}),
      ...(result === 'success' ? { success: true } : result === 'failed' ? { success: false } : {}),
    }),
    [page, deviceSn, operator, result]
  )

  const { data, isLoading, isError, isFetching, refetch } = useOpsCommandRecords(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0

  // keyword 后端不支持，对已分页结果叠加前端模糊匹配
  const filtered = useMemo(() => {
    if (!keyword.trim()) return rows
    const kw = keyword.trim().toLowerCase()
    return rows.filter(
      (r) => r.commandText.toLowerCase().includes(kw) || r.deviceSn.toLowerCase().includes(kw)
    )
  }, [rows, keyword])

  const durationText = (ms: number) => (ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${ms}ms`)

  return (
    <>
      <Toolbar
        keyword={keyword}
        keywordPlaceholder="指令 / 设备"
        onKeyword={(v) => setKeyword(v)}
        isFetching={isFetching}
        onRefresh={() => refetch()}
      >
        <input
          className="neon-input w-36"
          placeholder="设备 SN"
          value={deviceSn}
          onChange={(e) => {
            setDeviceSn(e.target.value)
            setPage(1)
          }}
        />
        <input
          className="neon-input w-32"
          placeholder="操作人"
          value={operator}
          onChange={(e) => {
            setOperator(e.target.value)
            setPage(1)
          }}
        />
        {(['', 'success', 'failed'] as const).map((r) => (
          <button
            key={r || 'all'}
            type="button"
            onClick={() => {
              setResult(r)
              setPage(1)
            }}
            className={`chip ${result === r ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55'}`}
            style={{ color: r === 'success' ? '#00ff88' : r === 'failed' ? '#ff2d6f' : '#00f0ff' }}
          >
            {r === 'success' ? '成功' : r === 'failed' ? '失败' : 'ALL'}
          </button>
        ))}
      </Toolbar>

      <StateBlock
        loading={isLoading}
        error={isError}
        empty={filtered.length === 0}
        emptyText="NO RECORDS · 无指令记录"
      >
        <div className="overflow-hidden rounded-sm border border-cyan-500/12">
          <div className="grid grid-cols-[150px_2fr_1.4fr_100px_80px_70px_60px] gap-3 border-b border-cyan-500/15 bg-cyan-500/[0.05] px-3 py-2 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/70">
            <span>时间</span>
            <span>指令 · COMMAND</span>
            <span>设备 · DEVICE</span>
            <span>操作人</span>
            <span>耗时</span>
            <span>结果</span>
            <span className="text-right">操作</span>
          </div>
          {filtered.map((r) => {
            const ok = r.success
            const durColor = r.duration > 5000 ? '#ff2d6f' : r.duration > 2000 ? '#ffaa00' : '#00ff88'
            return (
              <div
                key={r.id}
                className="fleet-row grid grid-cols-[150px_2fr_1.4fr_100px_80px_70px_60px] items-center gap-3 px-3 py-2"
                style={{ ['--row-color' as never]: ok ? '#00ff88' : '#ff2d6f' }}
              >
                <span className="font-mono text-[11px] text-cyan-300/70">
                  {formatTime(r.executeTime)}
                </span>
                <span className="truncate rounded-sm bg-black/30 px-2 py-0.5 font-mono text-[11px] text-emerald-300/85">
                  {r.commandText}
                </span>
                <span className="min-w-0">
                  <span className="block truncate text-[11px] text-cyan-100/85">{r.deviceName}</span>
                  <span className="block truncate font-mono text-[10px] text-cyan-300/55">
                    {r.deviceSn}
                  </span>
                </span>
                <span className="truncate text-[11px] text-cyan-300/75">{r.operator}</span>
                <span className="font-mono text-[11px]" style={{ color: durColor }}>
                  {durationText(r.duration)}
                </span>
                <span>
                  <span className="chip" style={{ color: ok ? '#00ff88' : '#ff2d6f' }}>
                    {ok ? '成功' : '失败'}
                  </span>
                </span>
                <span className="text-right">
                  <IconBtn title="详情" color="#00f0ff" onClick={() => setDetail(r)}>
                    <Eye className="size-3.5" />
                  </IconBtn>
                </span>
              </div>
            )
          })}
        </div>
        <Pager page={page} total={total} pageSize={pageSize} onPage={setPage} />
      </StateBlock>

      <CommandDetailDrawer record={detail} open={detail !== null} onClose={() => setDetail(null)} />
    </>
  )
}

// ============================================================
// 通用工具栏
// ============================================================

function Toolbar({
  keyword,
  keywordPlaceholder,
  onKeyword,
  isFetching,
  onRefresh,
  extra,
  children,
}: {
  keyword: string
  keywordPlaceholder: string
  onKeyword: (v: string) => void
  isFetching: boolean
  onRefresh: () => void
  extra?: React.ReactNode
  children?: React.ReactNode
}) {
  return (
    <div className="mb-3 flex flex-wrap items-center gap-2">
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
        <input
          className="neon-input w-56 pl-9"
          placeholder={keywordPlaceholder}
          value={keyword}
          onChange={(e) => onKeyword(e.target.value)}
        />
      </div>
      {children}
      <div className="ml-auto flex items-center gap-2">
        {isFetching ? (
          <span className="flex items-center gap-1.5 font-mono text-[11px] text-cyan-300/55">
            <Loader2 className="size-3 animate-spin" /> SYNC
          </span>
        ) : null}
        <NeonButton icon={<RefreshCcw />} onClick={onRefresh}>
          REFRESH
        </NeonButton>
        {extra}
      </div>
    </div>
  )
}
