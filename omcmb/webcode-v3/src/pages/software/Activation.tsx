import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Search, RefreshCcw, Loader2, Zap, Activity, ChevronRight, Pause, Play, Square, Trash2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatTime } from '@/lib/format'
import {
  useUpgradeTasks,
  useSuspendTask,
  useResumeTask,
  useTerminateTask,
  useDeleteTask,
} from '@core/hooks/api/useSoftware'
import type { UpgradeTaskInfo, TaskStatusType } from '@core/mock/data/software'

import { NEON, TASK_STATUS, RESULT_COLOR, StatCard, Syncing, ErrorBlock, EmptyBlock, Pager } from './_shared'

// ---------------------------------------------------------------------------
// 激活计划 · software/activation
// 版本激活任务（taskType=8）。行 → 详情 software/upgrade-plan/:id
// ---------------------------------------------------------------------------

const STATUS_FILTERS = ['', 'pending', 'in_progress', 'suspended', 'ended'] as const
const ACTIVATION_TASK_TYPE = 8

export default function Activation() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const pageSize = 12
  const [statusFilter, setStatusFilter] = useState<TaskStatusType | ''>('')
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      taskType: ACTIVATION_TASK_TYPE,
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
        (t.fileName ?? '').toLowerCase().includes(k) ||
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
      title="ACTIVATION PLAN · 激活计划"
      subtitle="VERSION ACTIVATION ORCHESTRATION · TASK TYPE 8"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="任务名 / 版本 / 制式 / 操作员"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 lg:grid-cols-4">
        <StatCard label="激活任务 · TASKS" value={total.toLocaleString()} color={NEON.cyan} icon={<Zap className="size-4" />} />
        <StatCard label="执行中 · ACTIVE" value={overview.active.toLocaleString()} color={overview.active > 0 ? NEON.amber : NEON.dim} icon={<Activity className="size-4" />} />
        <StatCard label="激活成功 · OK" value={overview.devSuccess.toLocaleString()} color={NEON.green} icon={<Activity className="size-4" />} />
        <StatCard label="激活失败 · FAIL" value={overview.devFail.toLocaleString()} color={overview.devFail > 0 ? NEON.rose : NEON.dim} icon={<Activity className="size-4" />} />
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
        <Syncing label="LOADING ACTIVATION QUEUE…" />
      ) : isError ? (
        <ErrorBlock msg={error instanceof Error ? error.message : '未知错误'} />
      ) : tasks.length === 0 ? (
        <EmptyBlock label="NO ACTIVATION TASKS · 无激活任务" />
      ) : (
        <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
          {tasks.map((t) => (
            <ActivationCard key={t.id} t={t} onInspect={() => navigate(`/software/upgrade-plan/${t.id}`)} />
          ))}
        </div>
      )}

      <Pager page={page} totalPages={totalPages} pageSize={pageSize} total={total} onPage={setPage} />
    </PageShell>
  )
}

function ActivationCard({ t, onInspect }: { t: UpgradeTaskInfo; onInspect: () => void }) {
  const suspend = useSuspendTask()
  const resume = useResumeTask()
  const terminate = useTerminateTask()
  const del = useDeleteTask()

  const sc = TASK_STATUS[t.status] ?? { label: t.status, color: NEON.dim }
  const pct = t.totalCount ? Math.round(((t.successCount + t.failCount) / t.totalCount) * 100) : 0
  const ended = t.status === 'ended'
  const barColor = t.result === 'failed' || t.result === 'terminated' ? NEON.rose : t.result === 'partial' ? NEON.amber : ended ? NEON.green : NEON.cyan

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
              {t.productClass || '—'} · VER {t.fileName || '—'} · BY {t.createUser || '—'}
            </div>
          </div>
          <span className="chip shrink-0" style={{ color: sc.color }}>
            <span className="size-1.5 rounded-full bg-current shadow-[0_0_6px_currentColor]" />
            {sc.label}
          </span>
        </div>

        <div className="mt-2.5">
          <div className="mb-1 flex items-center justify-between font-mono text-[10px] text-cyan-300/60">
            <span>ACTIVATE <span className="text-cyan-100/85">{t.successCount + t.failCount}/{t.totalCount}</span></span>
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
            <NeonButton tone="danger" className="!py-1 !px-2" icon={<Square />} disabled={terminate.isPending} onClick={() => confirm('终止激活任务？', () => terminate.mutate(t.id))}>终止</NeonButton>
          ) : null}
          {ended ? (
            <NeonButton tone="danger" className="!py-1 !px-2" icon={<Trash2 />} disabled={del.isPending} onClick={() => confirm('删除该任务记录？', () => del.mutate(t.id))}>删除</NeonButton>
          ) : null}
        </div>
      </div>
    </div>
  )
}
