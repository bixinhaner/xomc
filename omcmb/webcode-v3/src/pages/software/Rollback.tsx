import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  RefreshCcw,
  Loader2,
  Undo2,
  Activity,
  ChevronRight,
  Pause,
  Play,
  Square,
  RotateCcw,
  Trash2,
  Check,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatTime } from '@/lib/format'
import {
  useUpgradeTasks,
  useCreateRollback,
  useSuspendTask,
  useResumeTask,
  useTerminateTask,
  useRetryTask,
  useDeleteTask,
} from '@core/hooks/api/useSoftware'
import { useDeviceList } from '@core/hooks/api/useDevices'
import { useUserStore } from '@core/store/userStore'
import type { UpgradeTaskInfo, TaskStatusType } from '@core/mock/data/software'
import type { Device } from '@core/types/device'

import { NEON, TASK_STATUS, RESULT_COLOR, StatCard, Drawer, Syncing, ErrorBlock, EmptyBlock, Pager } from './_shared'

// ---------------------------------------------------------------------------
// 版本回退 · software/rollback
// 回退任务编排板（taskType=2）+ 新建回退（选设备 → useCreateRollback）
// 行 → 详情 software/upgrade-plan/:id
// ---------------------------------------------------------------------------

const STATUS_FILTERS = ['', 'pending', 'in_progress', 'suspended', 'ended'] as const
const ROLLBACK_TASK_TYPE = 2

export default function Rollback() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const pageSize = 12
  const [statusFilter, setStatusFilter] = useState<TaskStatusType | ''>('')
  const [keyword, setKeyword] = useState('')
  const [createOpen, setCreateOpen] = useState(false)

  const params = useMemo(
    () => ({
      page,
      pageSize,
      taskType: ROLLBACK_TASK_TYPE,
      ...(statusFilter ? { status: statusFilter } : {}),
    }),
    [page, statusFilter]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useUpgradeTasks(params)
  const allTasks = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const tasks = useMemo(() => {
    const k = keyword.trim().toLowerCase()
    if (!k) return allTasks
    return allTasks.filter(
      (t) =>
        t.taskName.toLowerCase().includes(k) ||
        t.productClass.toLowerCase().includes(k) ||
        t.createUser.toLowerCase().includes(k)
    )
  }, [allTasks, keyword])

  const overview = useMemo(() => {
    const active = allTasks.filter((t) => t.status !== 'ended').length
    const devSuccess = allTasks.reduce((a, t) => a + t.successCount, 0)
    const devFail = allTasks.reduce((a, t) => a + t.failCount, 0)
    return { active, devSuccess, devFail }
  }, [allTasks])

  return (
    <PageShell
      code="F06"
      title="VERSION ROLLBACK · 版本回退"
      subtitle="ROLLBACK ORCHESTRATION · TASK TYPE 2"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-56 pl-9"
              placeholder="任务名 / 制式 / 操作员"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
          <NeonButton icon={<Undo2 />} onClick={() => setCreateOpen(true)}>
            新建回退
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 lg:grid-cols-4">
        <StatCard label="回退任务 · TASKS" value={total.toLocaleString()} color={NEON.amber} icon={<Undo2 className="size-4" />} />
        <StatCard label="执行中 · ACTIVE" value={overview.active.toLocaleString()} color={overview.active > 0 ? NEON.amber : NEON.dim} icon={<Activity className="size-4" />} />
        <StatCard label="回退成功 · OK" value={overview.devSuccess.toLocaleString()} color={NEON.green} icon={<Activity className="size-4" />} />
        <StatCard label="回退失败 · FAIL" value={overview.devFail.toLocaleString()} color={overview.devFail > 0 ? NEON.rose : NEON.dim} icon={<Activity className="size-4" />} />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        {STATUS_FILTERS.map((s) => (
          <button
            key={s || 'all'}
            type="button"
            onClick={() => { setStatusFilter(s); setPage(1) }}
            className={`chip transition-all ${statusFilter === s ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55 hover:opacity-100'}`}
            style={{ color: s ? TASK_STATUS[s].color : NEON.cyan }}
          >
            {s ? TASK_STATUS[s].label : 'ALL'}
          </button>
        ))}
        {isFetching ? (
          <span className="flex items-center gap-1.5 text-[11px] text-cyan-300/60">
            <Loader2 className="size-3 animate-spin" /> LIVE · 5s
          </span>
        ) : null}
      </div>

      {isLoading ? (
        <Syncing label="LOADING ROLLBACK QUEUE…" />
      ) : isError ? (
        <ErrorBlock msg={error instanceof Error ? error.message : '未知错误'} />
      ) : tasks.length === 0 ? (
        <EmptyBlock label="NO ROLLBACK TASKS · 无回退任务" />
      ) : (
        <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
          {tasks.map((t) => (
            <RollbackCard key={t.id} t={t} onInspect={() => navigate(`/software/upgrade-plan/${t.id}`)} />
          ))}
        </div>
      )}

      <Pager page={page} totalPages={totalPages} pageSize={pageSize} total={total} onPage={setPage} />

      {createOpen ? <CreateRollbackDrawer onClose={() => setCreateOpen(false)} /> : null}
    </PageShell>
  )
}

function RollbackCard({ t, onInspect }: { t: UpgradeTaskInfo; onInspect: () => void }) {
  const suspend = useSuspendTask()
  const resume = useResumeTask()
  const terminate = useTerminateTask()
  const retry = useRetryTask()
  const del = useDeleteTask()

  const sc = TASK_STATUS[t.status] ?? { label: t.status, color: NEON.dim }
  const pct = t.totalCount ? Math.round(((t.successCount + t.failCount) / t.totalCount) * 100) : 0
  const ended = t.status === 'ended'
  const barColor = t.result === 'failed' || t.result === 'terminated' ? NEON.rose : t.result === 'partial' ? NEON.amber : ended ? NEON.green : NEON.amber

  const confirm = (msg: string, fn: () => void) => {
    if (typeof window === 'undefined' || window.confirm(msg)) fn()
  }

  return (
    <div className="glass relative overflow-hidden rounded-sm border-l-2 p-3" style={{ borderLeftColor: sc.color }}>
      <div className="scanline" />
      <div className="relative">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <div className="truncate font-display text-sm font-bold text-cyan-100">{t.taskName}</div>
            <div className="truncate font-mono text-[10px] text-cyan-300/55">
              {t.productClass || '—'} · BY {t.createUser || '—'}
            </div>
          </div>
          <span className="chip shrink-0" style={{ color: sc.color }}>
            <span className="size-1.5 rounded-full bg-current shadow-[0_0_6px_currentColor]" />
            {sc.label}
          </span>
        </div>

        <div className="mt-2.5">
          <div className="mb-1 flex items-center justify-between font-mono text-[10px] text-cyan-300/60">
            <span>ROLLBACK <span className="text-cyan-100/85">{t.successCount + t.failCount}/{t.totalCount}</span></span>
            <span style={{ color: barColor }}>{pct}%</span>
          </div>
          <div className="h-2 overflow-hidden rounded-full bg-cyan-500/10">
            <div className="h-full rounded-full transition-all" style={{ width: `${pct}%`, background: barColor, boxShadow: `0 0 8px ${barColor}` }} />
          </div>
          <div className="mt-1.5 flex items-center gap-3 font-mono text-[10px]">
            <span style={{ color: NEON.green }}>✓ {t.successCount}</span>
            <span style={{ color: t.failCount > 0 ? NEON.rose : NEON.dim }}>✗ {t.failCount}</span>
            {t.result ? <span style={{ color: RESULT_COLOR[t.result] ?? NEON.dim }}>{t.result}</span> : null}
          </div>
        </div>

        <div className="mt-2 grid grid-cols-2 gap-1 font-mono text-[10px] text-cyan-300/55">
          <div>开始 {t.startedAt ? formatTime(t.startedAt) : '—'}</div>
          <div>结束 {t.endedAt ? formatTime(t.endedAt) : '—'}</div>
        </div>

        <div className="mt-2.5 flex flex-wrap items-center gap-2 border-t border-cyan-500/10 pt-2.5">
          <NeonButton className="!py-1 !px-2" onClick={onInspect}>
            <ChevronRight className="size-3.5" /> 设备明细
          </NeonButton>
          {t.status === 'in_progress' ? (
            <NeonButton className="!py-1 !px-2" icon={<Pause />} disabled={suspend.isPending} onClick={() => suspend.mutate(t.id)}>暂停</NeonButton>
          ) : null}
          {t.status === 'suspended' ? (
            <NeonButton className="!py-1 !px-2" icon={<Play />} disabled={resume.isPending} onClick={() => resume.mutate(t.id)}>恢复</NeonButton>
          ) : null}
          {!ended ? (
            <NeonButton tone="danger" className="!py-1 !px-2" icon={<Square />} disabled={terminate.isPending} onClick={() => confirm('终止回退任务？', () => terminate.mutate(t.id))}>终止</NeonButton>
          ) : null}
          {ended && t.failCount > 0 ? (
            <NeonButton className="!py-1 !px-2" icon={<RotateCcw />} disabled={retry.isPending} onClick={() => confirm('重试失败设备？', () => retry.mutate(t.id))}>重试失败</NeonButton>
          ) : null}
          {ended ? (
            <NeonButton tone="danger" className="!py-1 !px-2" icon={<Trash2 />} disabled={del.isPending} onClick={() => confirm('删除该任务记录？', () => del.mutate(t.id))}>删除</NeonButton>
          ) : null}
        </div>
      </div>
    </div>
  )
}

function CreateRollbackDrawer({ onClose }: { onClose: () => void }) {
  const currentUser = useUserStore((s) => s.currentUser)
  const create = useCreateRollback()

  const [taskName, setTaskName] = useState('')
  const [keyword, setKeyword] = useState('')
  const [selected, setSelected] = useState<Record<string, true>>({})
  const [err, setErr] = useState('')

  const { data, isLoading, isError, error } = useDeviceList({
    page: 1,
    pageSize: 50,
    ...(keyword.trim() ? { searchText: keyword.trim() } : {}),
  })
  const devices: Device[] = data?.items ?? []
  const selectedIds = Object.keys(selected)

  const toggle = (id: string) =>
    setSelected((prev) => {
      const next = { ...prev }
      if (next[id]) delete next[id]
      else next[id] = true
      return next
    })

  const handleSubmit = () => {
    setErr('')
    if (!taskName.trim()) {
      setErr('任务名称必填')
      return
    }
    if (selectedIds.length === 0) {
      setErr('请至少选择一台设备')
      return
    }
    create.mutate(
      {
        deviceIds: selectedIds,
        taskName: taskName.trim(),
        createUser: currentUser?.username ?? 'admin',
      },
      { onSuccess: onClose, onError: (e) => setErr(e instanceof Error ? e.message : '创建失败') }
    )
  }

  return (
    <Drawer title="CREATE ROLLBACK · 新建回退任务" onClose={onClose} wide>
      <div className="space-y-4 p-4">
        <label className="block">
          <span className="mb-1 block font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">任务名称 · TASK NAME</span>
          <input className="neon-input w-full" value={taskName} onChange={(e) => setTaskName(e.target.value)} placeholder="如 华东 gNB 回退至上一稳定版本" />
        </label>

        <div>
          <div className="mb-1.5 flex items-center justify-between">
            <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">选择设备 · DEVICES</span>
            <span className="font-mono text-[11px]" style={{ color: NEON.green }}>已选 {selectedIds.length}</span>
          </div>
          <div className="relative mb-2">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input className="neon-input w-full pl-9" placeholder="SN / 名称 / IP" value={keyword} onChange={(e) => setKeyword(e.target.value)} />
          </div>

          {isLoading ? (
            <Syncing label="LOADING DEVICES…" />
          ) : isError ? (
            <ErrorBlock msg={error instanceof Error ? error.message : '未知错误'} />
          ) : devices.length === 0 ? (
            <EmptyBlock label="NO DEVICES · 无设备" />
          ) : (
            <div className="max-h-[46vh] space-y-1.5 overflow-auto pr-1">
              {devices.map((d) => {
                const on = Boolean(selected[d.id])
                return (
                  <button
                    key={d.id}
                    type="button"
                    onClick={() => toggle(d.id)}
                    className="fleet-row grid w-full grid-cols-[20px_1.4fr_1fr_120px] items-center gap-3 rounded-sm px-3 py-2 text-left"
                    style={{ ['--row-color' as never]: on ? NEON.green : NEON.cyan }}
                  >
                    <span
                      className="flex size-4 items-center justify-center rounded-sm border"
                      style={{ borderColor: on ? NEON.green : 'rgba(0,240,255,0.3)', background: on ? NEON.green : 'transparent' }}
                    >
                      {on ? <Check className="size-3 text-black" /> : null}
                    </span>
                    <div className="min-w-0">
                      <div className="truncate font-mono text-xs text-cyan-100">{d.sn}</div>
                      <div className="truncate font-mono text-[10px] text-cyan-300/55">{d.name || '—'}</div>
                    </div>
                    <span className="truncate font-mono text-[10px] text-cyan-300/65">{d.softwareVersion || '—'}</span>
                    <span className="text-right">
                      <span className="chip" style={{ color: d.isOnline ? NEON.green : NEON.dim }}>
                        <span className="size-1.5 rounded-full bg-current shadow-[0_0_6px_currentColor]" />
                        {d.isOnline ? '在线' : '离线'}
                      </span>
                    </span>
                  </button>
                )
              })}
            </div>
          )}
        </div>

        {err ? <div className="font-mono text-[11px] text-rose-300">{err}</div> : null}

        <div className="flex justify-end gap-2 border-t border-cyan-500/10 pt-3">
          <NeonButton onClick={onClose}>取消</NeonButton>
          <NeonButton icon={<Undo2 />} disabled={create.isPending} onClick={handleSubmit}>
            {create.isPending ? '创建中…' : `创建回退 (${selectedIds.length})`}
          </NeonButton>
        </div>
      </div>
    </Drawer>
  )
}
