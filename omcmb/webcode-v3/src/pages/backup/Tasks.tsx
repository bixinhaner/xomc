import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  RefreshCcw,
  Loader2,
  Database,
  StopCircle,
  Trash2,
  Inbox,
  AlertTriangle,
  ChevronRight,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { Sparkline } from '@/components/viz/Sparkline'
import { formatBytes, formatTime } from '@/lib/format'
import {
  useBackupTasks,
  useCancelBackupTask,
  useDeleteBackupTasks,
} from '@core/hooks/api/useBackup'
import type { BackupTask } from '@core/mock/data/backup'

import { Modal } from './Modal'

// ---------------------------------------------------------------------------
// 视觉常量
// ---------------------------------------------------------------------------

type TaskStatus = BackupTask['status']

const TASK_STATUS_COLOR: Record<TaskStatus, string> = {
  pending: '#5b9eff',
  running: '#00f0ff',
  success: '#00ff88',
  failed: '#ff2d6f',
  cancelled: '#ffaa00',
  partial: '#ff7a1a',
}
const TASK_STATUS_LABEL: Record<TaskStatus, string> = {
  pending: '等待',
  running: '运行中',
  success: '成功',
  failed: '失败',
  cancelled: '已取消',
  partial: '部分成功',
}
const BACKUP_TYPE_LABEL: Record<BackupTask['backupType'], string> = {
  full: '全量',
  incremental: '增量',
  'config-only': '配置',
}

const PAGE_SIZE = 20

function getErrMsg(e: unknown): string {
  if (e instanceof Error) return e.message
  if (typeof e === 'object' && e && 'message' in e) {
    return String((e as { message: unknown }).message)
  }
  return '未知错误'
}

// ---------------------------------------------------------------------------
// 备份任务列表页 · /backup/tasks
// ---------------------------------------------------------------------------

export default function BackupTasksPage() {
  const navigate = useNavigate()

  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState<TaskStatus | ''>('')
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(statusFilter ? { status: statusFilter } : {}),
    }),
    [page, statusFilter]
  )
  const query = useBackupTasks(params)
  const cancelTask = useCancelBackupTask()
  const deleteTasks = useDeleteBackupTasks()

  const rawTasks = query.data?.items ?? []
  const tasks = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return rawTasks
    return rawTasks.filter(
      (t) =>
        (t.taskName ?? '').toLowerCase().includes(kw) ||
        (t.deviceSns ?? []).some((sn) => sn.toLowerCase().includes(kw)) ||
        (t.creator ?? '').toLowerCase().includes(kw)
    )
  }, [rawTasks, keyword])

  const total = query.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  // 概览统计（基于当前页聚合 —— 后端未提供全网 stats 时降级）
  const overview = useMemo(() => {
    let running = 0
    let failed = 0
    let bytes = 0
    let devices = 0
    for (const t of rawTasks) {
      if (t.status === 'running' || t.status === 'pending') running += 1
      if (t.status === 'failed') failed += 1
      bytes += t.fileSize ?? 0
      devices += t.totalCount ?? 0
    }
    const sampleOk = rawTasks.filter((t) => t.status === 'success').length
    const sampleDone = rawTasks.filter(
      (t) => t.status !== 'running' && t.status !== 'pending'
    ).length
    const successRate = sampleDone > 0 ? (sampleOk / sampleDone) * 100 : 0
    return { total: query.data?.total ?? rawTasks.length, running, failed, bytes, devices, successRate }
  }, [rawTasks, query.data?.total])

  const [confirmTask, setConfirmTask] = useState<{
    task: BackupTask
    action: 'cancel' | 'delete'
  } | null>(null)
  const [actionMsg, setActionMsg] = useState<string | null>(null)

  const onConfirmTaskAction = async () => {
    if (!confirmTask) return
    try {
      if (confirmTask.action === 'cancel') {
        await cancelTask.mutateAsync(confirmTask.task.id)
        setActionMsg(`已取消任务 ${confirmTask.task.taskName}`)
      } else {
        await deleteTasks.mutateAsync([confirmTask.task.id])
        setActionMsg(`已删除任务 ${confirmTask.task.taskName}`)
      }
      setConfirmTask(null)
    } catch (e) {
      setActionMsg(`操作失败：${getErrMsg(e)}`)
    }
  }

  const statusOptions: (TaskStatus | '')[] = [
    '',
    'running',
    'pending',
    'success',
    'partial',
    'failed',
    'cancelled',
  ]

  return (
    <PageShell
      code="F06"
      title="BACKUP TASKS · 备份任务"
      subtitle="DEVICE CONFIG SNAPSHOT JOBS"
      isFetching={query.isFetching || undefined}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => query.refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      {/* 概览 */}
      <div className="mb-3 grid grid-cols-2 gap-3 lg:grid-cols-4">
        <OverviewStat
          label="备份任务 · TASKS"
          value={overview.total.toLocaleString()}
          color="#00f0ff"
          icon={<Database className="size-4" />}
          trend={[40, 44, 48, 46, 52, 58, 60, 66, 70, 74]}
        />
        <OverviewStat
          label="运行中 · RUNNING"
          value={overview.running.toLocaleString()}
          color="#5b9eff"
          icon={<Loader2 className="size-4" />}
          trend={[2, 4, 3, 6, 5, 8, 6, 9, 7, 10]}
        />
        <OverviewStat
          label="失败 · FAILED"
          value={overview.failed.toLocaleString()}
          color={overview.failed > 0 ? '#ff2d6f' : '#00ff88'}
          icon={<AlertTriangle className="size-4" />}
          trend={[1, 0, 2, 1, 3, 1, 2, 0, 1, 2]}
        />
        <div className="glass relative flex items-center gap-4 overflow-hidden rounded-sm px-4 py-3">
          <div className="scanline" />
          <RadialGauge
            value={overview.successRate}
            label="成功率"
            size={84}
            color="#00ff88"
          />
          <div className="min-w-0">
            <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
              本页归档 · ARCHIVED
            </div>
            <div
              className="font-display text-2xl font-bold leading-tight text-glow"
              style={{ color: '#a855f7' }}
            >
              {formatBytes(overview.bytes)}
            </div>
            <div className="font-mono text-[10px] text-cyan-300/55">
              {overview.devices.toLocaleString()} DEVICE SNAPSHOTS
            </div>
          </div>
        </div>
      </div>

      {/* 工具条 */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
          <input
            className="neon-input w-64 pl-9"
            placeholder="任务名 / 设备 SN / 创建人"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
        {statusOptions.map((s) => (
          <button
            key={s || 'all'}
            type="button"
            onClick={() => {
              setStatusFilter(s)
              setPage(1)
            }}
            className={`chip transition-all ${
              statusFilter === s
                ? 'shadow-[0_0_10px_currentColor]'
                : 'opacity-60 hover:opacity-100'
            }`}
            style={{ color: s ? TASK_STATUS_COLOR[s as TaskStatus] : '#00f0ff' }}
          >
            {s ? TASK_STATUS_LABEL[s as TaskStatus] : 'ALL'}
          </button>
        ))}
      </div>

      {/* 列表 */}
      {query.isLoading ? (
        <CenterSync />
      ) : query.isError ? (
        <FailureBox error={query.error} />
      ) : tasks.length === 0 ? (
        <EmptyBox hint="NO BACKUP TASKS · 无备份任务" />
      ) : (
        <>
          <div className="space-y-1.5">
            {tasks.map((t) => {
              const color = TASK_STATUS_COLOR[t.status]
              const inflight = t.status === 'running' || t.status === 'pending'
              return (
                <div
                  key={t.id}
                  className="fleet-row grid grid-cols-[10px_1.8fr_1fr_1.4fr_1fr_150px] items-center gap-3 rounded-sm px-3 py-2.5"
                  style={{ ['--row-color' as never]: color }}
                >
                  <span
                    className="size-2 rounded-full"
                    style={{ background: color, boxShadow: `0 0 8px ${color}` }}
                  />
                  <button
                    type="button"
                    onClick={() => navigate(`/backup/tasks/${t.id}`)}
                    className="min-w-0 text-left"
                  >
                    <div className="flex items-center gap-1 truncate font-display text-sm font-bold text-cyan-100 transition-colors hover:text-cyan-300">
                      {t.taskName || t.id}
                      <ChevronRight className="size-3.5 shrink-0 text-cyan-300/50" />
                    </div>
                    <div className="font-mono text-[10px] text-cyan-300/55">
                      {t.taskType === 'scheduled' ? '定时' : '手动'} ·{' '}
                      {BACKUP_TYPE_LABEL[t.backupType]} · {t.creator || '—'}
                    </div>
                  </button>
                  <div>
                    <div className="text-xs text-cyan-100/85">
                      {t.totalCount ?? 0} 台设备
                    </div>
                    <div className="font-mono text-[10px] text-cyan-300/55">
                      OK {t.successCount ?? 0} · ERR {t.failCount ?? 0}
                    </div>
                  </div>
                  <div>
                    <div className="mb-1 flex items-center justify-between">
                      <span className="chip" style={{ color }}>
                        {TASK_STATUS_LABEL[t.status]}
                      </span>
                      <span className="font-mono text-[10px] text-cyan-300/65">
                        {t.progress ?? 0}%
                      </span>
                    </div>
                    <ProgressBar value={t.progress ?? 0} color={color} />
                  </div>
                  <div className="font-mono text-[11px] text-cyan-300/75">
                    <div>{formatBytes(t.fileSize)}</div>
                    <div className="text-[10px] text-cyan-300/45">
                      {formatTime(t.createdAt)}
                    </div>
                  </div>
                  <div className="flex justify-end gap-1.5">
                    {inflight ? (
                      <NeonButton
                        icon={<StopCircle />}
                        onClick={() => setConfirmTask({ task: t, action: 'cancel' })}
                      >
                        取消
                      </NeonButton>
                    ) : (
                      <NeonButton
                        tone="danger"
                        icon={<Trash2 />}
                        onClick={() => setConfirmTask({ task: t, action: 'delete' })}
                      >
                        删除
                      </NeonButton>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
          <Pager page={page} totalPages={totalPages} total={total} onPage={setPage} />
        </>
      )}

      {/* 取消 / 删除确认 */}
      <Modal
        open={Boolean(confirmTask)}
        title={confirmTask?.action === 'cancel' ? '取消备份任务' : '删除备份任务'}
        subtitle={confirmTask?.action === 'cancel' ? 'ABORT TASK' : 'PURGE TASK'}
        onClose={() => setConfirmTask(null)}
        width={460}
        footer={
          <>
            <NeonButton onClick={() => setConfirmTask(null)}>取消</NeonButton>
            <NeonButton
              tone="danger"
              disabled={cancelTask.isPending || deleteTasks.isPending}
              onClick={() => {
                void onConfirmTaskAction()
              }}
            >
              {cancelTask.isPending || deleteTasks.isPending ? '处理中…' : '确认'}
            </NeonButton>
          </>
        }
      >
        <p className="text-sm text-cyan-100/85">
          {confirmTask?.action === 'cancel'
            ? '将中止该任务剩余设备的备份执行，已完成的快照保留。'
            : '将永久删除该任务记录，已生成的快照文件按保留策略另行清理。'}
        </p>
        <p className="mt-2 font-mono text-xs text-cyan-300/70">
          {confirmTask?.task.taskName} · {confirmTask?.task.id}
        </p>
      </Modal>

      <Modal
        open={Boolean(actionMsg)}
        title="操作结果"
        subtitle="RESULT"
        onClose={() => setActionMsg(null)}
        width={420}
        footer={<NeonButton onClick={() => setActionMsg(null)}>知道了</NeonButton>}
      >
        <p className="text-sm text-cyan-100/85">{actionMsg}</p>
      </Modal>
    </PageShell>
  )
}

// ---------------------------------------------------------------------------
// 局部组件
// ---------------------------------------------------------------------------

function OverviewStat({
  label,
  value,
  color,
  icon,
  trend,
}: {
  label: string
  value: string
  color: string
  icon: React.ReactNode
  trend: number[]
}) {
  return (
    <div className="glass relative overflow-hidden rounded-sm">
      <div className="scanline" />
      <div className="relative flex items-stretch gap-3 p-4">
        <div className="flex flex-col items-center justify-center border-r border-cyan-500/15 pr-3">
          <span style={{ color }}>{icon}</span>
        </div>
        <div className="flex flex-1 flex-col">
          <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
            {label}
          </div>
          <div
            className="font-display text-3xl font-bold leading-tight text-glow"
            style={{ color }}
          >
            {value}
          </div>
        </div>
        <Sparkline data={trend} color={color} width={84} height={34} />
      </div>
    </div>
  )
}

function ProgressBar({ value, color }: { value: number; color: string }) {
  return (
    <div className="h-1 overflow-hidden rounded-full bg-cyan-500/10">
      <div
        className="h-full rounded-full transition-all"
        style={{
          width: `${Math.min(100, Math.max(0, value))}%`,
          background: color,
          boxShadow: `0 0 6px ${color}`,
        }}
      />
    </div>
  )
}

function Pager({
  page,
  totalPages,
  total,
  onPage,
}: {
  page: number
  totalPages: number
  total: number
  onPage: (updater: (p: number) => number) => void
}) {
  return (
    <div className="mt-4 flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <NeonButton onClick={() => onPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
          ◂ PREV
        </NeonButton>
        <NeonButton
          onClick={() => onPage((p) => Math.min(totalPages, p + 1))}
          disabled={page >= totalPages}
        >
          NEXT ▸
        </NeonButton>
      </div>
    </div>
  )
}

function CenterSync() {
  return (
    <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
      <Loader2 className="size-4 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
    </div>
  )
}

function FailureBox({ error }: { error: unknown }) {
  return (
    <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
      FAILURE · {getErrMsg(error)}
    </div>
  )
}

function EmptyBox({ hint }: { hint: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16">
      <Inbox className="size-10 text-cyan-400/50" />
      <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
        {hint}
      </div>
    </div>
  )
}
