import { useMemo, useState } from 'react'
import {
  Search,
  RefreshCcw,
  Loader2,
  Database,
  PlusCircle,
  StopCircle,
  Trash2,
  HardDriveDownload,
  Inbox,
  AlertTriangle,
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
  useBackupRestoreTasks,
  useCreateBackupRestore,
  useCreateBackupRestoreByTaskID,
  useBackupSchedules,
  useFTPConfigs,
  useBackupPolicy,
} from '@core/hooks/api/useBackup'
import type {
  BackupTask,
  RestoreTask,
  RestoreStatus,
  BackupSchedule,
  FTPConfig,
} from '@core/mock/data/backup'

import { Modal } from './Modal'

// ---------------------------------------------------------------------------
// 视觉常量 —— 备份/恢复状态 → 霓虹色
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

const RESTORE_STATUS_COLOR: Record<RestoreStatus, string> = {
  pending: '#5b9eff',
  running: '#00f0ff',
  completed: '#00ff88',
  failed: '#ff2d6f',
  cancelled: '#ffaa00',
}
const RESTORE_STATUS_LABEL: Record<RestoreStatus, string> = {
  pending: '等待',
  running: '运行中',
  completed: '已完成',
  failed: '失败',
  cancelled: '已取消',
}

const BACKUP_TYPE_LABEL: Record<BackupTask['backupType'], string> = {
  full: '全量',
  incremental: '增量',
  'config-only': '配置',
}

type Tab = 'tasks' | 'restore' | 'schedules' | 'ftp' | 'policy'
const TABS: { key: Tab; label: string; en: string }[] = [
  { key: 'tasks', label: '备份任务', en: 'TASKS' },
  { key: 'restore', label: '数据恢复', en: 'RESTORE' },
  { key: 'schedules', label: '定时策略', en: 'SCHEDULES' },
  { key: 'ftp', label: 'FTP 通道', en: 'FTP' },
  { key: 'policy', label: '保留策略', en: 'POLICY' },
]

const PAGE_SIZE = 20

function getErrMsg(e: unknown): string {
  if (e instanceof Error) return e.message
  if (typeof e === 'object' && e && 'message' in e) {
    return String((e as { message: unknown }).message)
  }
  return '未知错误'
}

function parseSnList(raw: string): string[] {
  return raw
    .split(/[\r\n,]+/)
    .map((s) => s.trim())
    .filter(Boolean)
}

// ---------------------------------------------------------------------------
// 主页面
// ---------------------------------------------------------------------------

export function BackupPage() {
  const [tab, setTab] = useState<Tab>('tasks')

  // 任务列表（同时驱动顶部概览统计）
  const [taskPage, setTaskPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState<TaskStatus | ''>('')
  const [keyword, setKeyword] = useState('')

  const taskParams = useMemo(
    () => ({
      page: taskPage,
      pageSize: PAGE_SIZE,
      ...(statusFilter ? { status: statusFilter } : {}),
    }),
    [taskPage, statusFilter]
  )
  const tasksQuery = useBackupTasks(taskParams)
  const cancelTask = useCancelBackupTask()
  const deleteTasks = useDeleteBackupTasks()

  const rawTasks = tasksQuery.data?.items ?? []
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

  // 顶部概览统计（基于当前任务页 —— 全量统计需后端 stats，未提供时降级为本页聚合）
  const overview = useMemo(() => {
    const total = tasksQuery.data?.total ?? rawTasks.length
    let ok = 0
    let running = 0
    let failed = 0
    let bytes = 0
    let devices = 0
    for (const t of rawTasks) {
      if (t.status === 'success') ok += 1
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
    return { total, ok, running, failed, bytes, devices, successRate }
  }, [rawTasks, tasksQuery.data?.total])

  // 删除确认 / 取消 modal 状态
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

  const totalTaskPages = Math.max(
    1,
    Math.ceil((tasksQuery.data?.total ?? 0) / PAGE_SIZE)
  )

  return (
    <PageShell
      code="F06"
      title="BACKUP · 备援存档"
      subtitle="DEVICE CONFIG SNAPSHOT · RESTORE · RETENTION"
      isFetching={
        tasksQuery.isFetching && tab === 'tasks' ? true : undefined
      }
      bare
      toolbar={
        <div className="flex flex-wrap items-center gap-1.5">
          {TABS.map((tDef) => (
            <button
              key={tDef.key}
              type="button"
              onClick={() => setTab(tDef.key)}
              className={`chip transition-all ${
                tab === tDef.key
                  ? 'shadow-[0_0_10px_currentColor] text-cyan-200'
                  : 'text-cyan-300/55 hover:text-cyan-200'
              }`}
            >
              {tDef.label} · {tDef.en}
            </button>
          ))}
        </div>
      }
    >
      {/* 顶部概览 —— 始终展示，跨页签复用任务聚合 */}
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

      {tab === 'tasks' && (
        <TasksSection
          query={tasksQuery}
          tasks={tasks}
          keyword={keyword}
          setKeyword={(v) => {
            setKeyword(v)
          }}
          statusFilter={statusFilter}
          setStatusFilter={(s) => {
            setStatusFilter(s)
            setTaskPage(1)
          }}
          page={taskPage}
          totalPages={totalTaskPages}
          total={tasksQuery.data?.total ?? 0}
          setPage={setTaskPage}
          onCancel={(t) => setConfirmTask({ task: t, action: 'cancel' })}
          onDelete={(t) => setConfirmTask({ task: t, action: 'delete' })}
        />
      )}
      {tab === 'restore' && <RestoreSection />}
      {tab === 'schedules' && <SchedulesSection />}
      {tab === 'ftp' && <FtpSection />}
      {tab === 'policy' && <PolicySection />}

      {/* 任务取消 / 删除确认 */}
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

      {/* 操作结果浮层 */}
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
// 概览卡
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

// ---------------------------------------------------------------------------
// 通用三态壳
// ---------------------------------------------------------------------------

function StateShell({
  isLoading,
  isError,
  error,
  isEmpty,
  emptyHint,
  children,
}: {
  isLoading: boolean
  isError: boolean
  error: unknown
  isEmpty: boolean
  emptyHint: string
  children: React.ReactNode
}) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
        <Loader2 className="size-4 animate-spin" />
        <span className="font-mono text-xs uppercase tracking-[0.2em]">
          SYNCING…
        </span>
      </div>
    )
  }
  if (isError) {
    return (
      <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
        FAILURE · {getErrMsg(error)}
      </div>
    )
  }
  if (isEmpty) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-16">
        <Inbox className="size-10 text-cyan-400/50" />
        <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
          {emptyHint}
        </div>
      </div>
    )
  }
  return <>{children}</>
}

// ---------------------------------------------------------------------------
// Pager
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// TASKS
// ---------------------------------------------------------------------------

function TasksSection({
  query,
  tasks,
  keyword,
  setKeyword,
  statusFilter,
  setStatusFilter,
  page,
  totalPages,
  total,
  setPage,
  onCancel,
  onDelete,
}: {
  query: ReturnType<typeof useBackupTasks>
  tasks: BackupTask[]
  keyword: string
  setKeyword: (v: string) => void
  statusFilter: TaskStatus | ''
  setStatusFilter: (s: TaskStatus | '') => void
  page: number
  totalPages: number
  total: number
  setPage: (updater: (p: number) => number) => void
  onCancel: (t: BackupTask) => void
  onDelete: (t: BackupTask) => void
}) {
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
    <div>
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
            onClick={() => setStatusFilter(s)}
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
        <NeonButton icon={<RefreshCcw />} onClick={() => query.refetch()}>
          REFRESH
        </NeonButton>
      </div>

      <StateShell
        isLoading={query.isLoading}
        isError={query.isError}
        error={query.error}
        isEmpty={tasks.length === 0}
        emptyHint="NO BACKUP TASKS · 无备份任务"
      >
        <div className="space-y-1.5">
          {tasks.map((t) => {
            const color = TASK_STATUS_COLOR[t.status]
            const total = t.totalCount ?? 0
            const done = t.status === 'running' || t.status === 'pending'
            return (
              <div
                key={t.id}
                className="fleet-row grid grid-cols-[10px_1.8fr_1fr_1.4fr_1fr_120px] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: color }}
              >
                <span
                  className="size-2 rounded-full"
                  style={{ background: color, boxShadow: `0 0 8px ${color}` }}
                />
                <div className="min-w-0">
                  <div className="truncate font-display text-sm font-bold text-cyan-100">
                    {t.taskName || t.id}
                  </div>
                  <div className="font-mono text-[10px] text-cyan-300/55">
                    {t.taskType === 'scheduled' ? '定时' : '手动'} ·{' '}
                    {BACKUP_TYPE_LABEL[t.backupType]} · {t.creator || '—'}
                  </div>
                </div>
                <div>
                  <div className="text-xs text-cyan-100/85">
                    {total} 台设备
                  </div>
                  <div className="font-mono text-[10px] text-cyan-300/55">
                    OK {t.successCount ?? 0} · ERR {t.failCount ?? 0}
                  </div>
                </div>
                <div>
                  <div className="mb-1 flex items-center justify-between">
                    <span
                      className="chip"
                      style={{ color }}
                    >
                      {TASK_STATUS_LABEL[t.status]}
                    </span>
                    <span className="font-mono text-[10px] text-cyan-300/65">
                      {t.progress ?? 0}%
                    </span>
                  </div>
                  <div className="h-1 overflow-hidden rounded-full bg-cyan-500/10">
                    <div
                      className="h-full rounded-full transition-all"
                      style={{
                        width: `${Math.min(100, Math.max(0, t.progress ?? 0))}%`,
                        background: color,
                        boxShadow: `0 0 6px ${color}`,
                      }}
                    />
                  </div>
                </div>
                <div className="font-mono text-[11px] text-cyan-300/75">
                  <div>{formatBytes(t.fileSize)}</div>
                  <div className="text-[10px] text-cyan-300/45">
                    {formatTime(t.createdAt)}
                  </div>
                </div>
                <div className="flex justify-end gap-1.5">
                  {done ? (
                    <NeonButton
                      icon={<StopCircle />}
                      onClick={() => onCancel(t)}
                    >
                      取消
                    </NeonButton>
                  ) : (
                    <NeonButton
                      tone="danger"
                      icon={<Trash2 />}
                      onClick={() => onDelete(t)}
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
      </StateShell>
    </div>
  )
}

// ---------------------------------------------------------------------------
// RESTORE
// ---------------------------------------------------------------------------

function RestoreSection() {
  const [page, setPage] = useState(1)
  const query = useBackupRestoreTasks({ page, pageSize: PAGE_SIZE })
  const createByPath = useCreateBackupRestore()
  const createByTask = useCreateBackupRestoreByTaskID()

  const items = query.data?.items ?? []
  const total = query.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const [open, setOpen] = useState(false)
  const [mode, setMode] = useState<'path' | 'task'>('path')
  const [bucket, setBucket] = useState('config_backup')
  const [objectPath, setObjectPath] = useState('')
  const [backupTaskId, setBackupTaskId] = useState('')
  const [snRaw, setSnRaw] = useState('')
  const [formErr, setFormErr] = useState<string | null>(null)
  const [resultMsg, setResultMsg] = useState<string | null>(null)

  const resetForm = () => {
    setMode('path')
    setBucket('config_backup')
    setObjectPath('')
    setBackupTaskId('')
    setSnRaw('')
    setFormErr(null)
  }

  const submitting = createByPath.isPending || createByTask.isPending

  const onSubmit = async () => {
    setFormErr(null)
    const sns = parseSnList(snRaw)
    if (sns.length === 0) {
      setFormErr('请至少填写一个目标设备 SN')
      return
    }
    try {
      if (mode === 'task') {
        if (!backupTaskId.trim()) {
          setFormErr('请填写备份任务 ID')
          return
        }
        await createByTask.mutateAsync({
          backupTaskId: backupTaskId.trim(),
          targetDeviceSns: sns,
        })
      } else {
        if (!objectPath.trim()) {
          setFormErr('请填写备份对象路径')
          return
        }
        if (objectPath.includes('..') || objectPath.startsWith('/')) {
          setFormErr('对象路径非法（不可含 .. 或以 / 开头）')
          return
        }
        await createByPath.mutateAsync({
          bucket: bucket.trim(),
          objectPath: objectPath.trim(),
          targetDeviceSns: sns,
        })
      }
      setOpen(false)
      resetForm()
      setResultMsg(`已提交恢复任务 · 目标 ${sns.length} 台设备`)
    } catch (e) {
      setFormErr(`提交失败：${getErrMsg(e)}`)
    }
  }

  return (
    <div>
      <div className="mb-3 flex items-center justify-between">
        <span className="font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/55">
          RESTORE FAN-OUT · 5s 自动刷新
        </span>
        <div className="flex gap-2">
          <NeonButton icon={<RefreshCcw />} onClick={() => query.refetch()}>
            REFRESH
          </NeonButton>
          <NeonButton
            icon={<PlusCircle />}
            onClick={() => {
              resetForm()
              setOpen(true)
            }}
          >
            新建恢复
          </NeonButton>
        </div>
      </div>

      <StateShell
        isLoading={query.isLoading}
        isError={query.isError}
        error={query.error}
        isEmpty={items.length === 0}
        emptyHint="NO RESTORE TASKS · 无恢复任务"
      >
        <div className="space-y-1.5">
          {items.map((r: RestoreTask) => {
            const color = RESTORE_STATUS_COLOR[r.status]
            return (
              <div
                key={r.id}
                className="fleet-row grid grid-cols-[10px_2.2fr_0.8fr_1.4fr_1.2fr] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: color }}
              >
                <span
                  className="size-2 rounded-full"
                  style={{ background: color, boxShadow: `0 0 8px ${color}` }}
                />
                <div className="min-w-0">
                  <div className="truncate font-mono text-xs text-cyan-100">
                    {r.sourceObjectPath || '—'}
                  </div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">
                    {r.sourceBucket}
                    {r.errorMessage ? ` · ⚠ ${r.errorMessage}` : ''}
                  </div>
                </div>
                <div className="text-xs text-cyan-100/85">
                  {r.targetDeviceSns.length} 台
                </div>
                <div>
                  <div className="mb-1 flex items-center justify-between">
                    <span className="chip" style={{ color }}>
                      {RESTORE_STATUS_LABEL[r.status]}
                    </span>
                    <span className="font-mono text-[10px] text-cyan-300/65">
                      {r.progress}%
                    </span>
                  </div>
                  <div className="h-1 overflow-hidden rounded-full bg-cyan-500/10">
                    <div
                      className="h-full rounded-full transition-all"
                      style={{
                        width: `${Math.min(100, Math.max(0, r.progress))}%`,
                        background: color,
                        boxShadow: `0 0 6px ${color}`,
                      }}
                    />
                  </div>
                </div>
                <div className="font-mono text-[10px] text-cyan-300/65">
                  <div>起 {formatTime(r.startedAt)}</div>
                  <div className="text-cyan-300/45">
                    终 {formatTime(r.completedAt)}
                  </div>
                </div>
              </div>
            )
          })}
        </div>
        <Pager page={page} totalPages={totalPages} total={total} onPage={setPage} />
      </StateShell>

      {/* 新建恢复 */}
      <Modal
        open={open}
        title="新建恢复任务"
        subtitle="RESTORE FAN-OUT"
        onClose={() => setOpen(false)}
        footer={
          <>
            <NeonButton onClick={() => setOpen(false)}>取消</NeonButton>
            <NeonButton
              icon={<HardDriveDownload />}
              disabled={submitting}
              onClick={() => {
                void onSubmit()
              }}
            >
              {submitting ? '提交中…' : '提交'}
            </NeonButton>
          </>
        }
      >
        <div className="space-y-4">
          <div className="flex gap-2">
            {(['path', 'task'] as const).map((m) => (
              <button
                key={m}
                type="button"
                onClick={() => {
                  setMode(m)
                  setFormErr(null)
                }}
                className={`chip transition-all ${
                  mode === m ? 'shadow-[0_0_10px_currentColor] text-cyan-200' : 'text-cyan-300/55'
                }`}
              >
                {m === 'path' ? '按对象路径' : '按备份任务'}
              </button>
            ))}
          </div>

          {mode === 'task' ? (
            <Field label="备份任务 ID" hint="从备份任务列表复制任务 ID，回放其快照">
              <input
                className="neon-input w-full"
                value={backupTaskId}
                placeholder="bkp-xxxx"
                onChange={(e) => setBackupTaskId(e.target.value)}
              />
            </Field>
          ) : (
            <>
              <Field label="存储桶 Bucket" hint="当前仅支持 config_backup">
                <input
                  className="neon-input w-full"
                  value={bucket}
                  onChange={(e) => setBucket(e.target.value)}
                />
              </Field>
              <Field
                label="对象路径 Object Path"
                hint="相对路径，不可含 .. 或以 / 开头"
              >
                <input
                  className="neon-input w-full"
                  value={objectPath}
                  placeholder="backup/2026/04/29/cfg-cmcc-lte-001.xml.gz"
                  onChange={(e) => setObjectPath(e.target.value)}
                />
              </Field>
            </>
          )}

          <Field label="目标设备 SN" hint="每行一个，或逗号分隔">
            <textarea
              className="neon-input min-h-[120px] w-full resize-y"
              value={snRaw}
              placeholder={'SN001\nSN002\nSN003'}
              onChange={(e) => setSnRaw(e.target.value)}
            />
          </Field>

          {formErr ? (
            <div className="border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">
              {formErr}
            </div>
          ) : null}
        </div>
      </Modal>

      {/* 结果浮层 */}
      <Modal
        open={Boolean(resultMsg)}
        title="操作结果"
        subtitle="RESULT"
        onClose={() => setResultMsg(null)}
        width={420}
        footer={<NeonButton onClick={() => setResultMsg(null)}>知道了</NeonButton>}
      >
        <p className="text-sm text-cyan-100/85">{resultMsg}</p>
      </Modal>
    </div>
  )
}

function Field({
  label,
  hint,
  children,
}: {
  label: string
  hint?: string
  children: React.ReactNode
}) {
  return (
    <label className="block">
      <div className="mb-1 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/65">
        {label}
      </div>
      {children}
      {hint ? (
        <div className="mt-1 font-mono text-[10px] text-cyan-300/40">{hint}</div>
      ) : null}
    </label>
  )
}

// ---------------------------------------------------------------------------
// SCHEDULES
// ---------------------------------------------------------------------------

function SchedulesSection() {
  const [page, setPage] = useState(1)
  const query = useBackupSchedules({ page, pageSize: PAGE_SIZE })
  const items = query.data?.items ?? []
  const total = query.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <div>
      <div className="mb-3 flex items-center justify-end">
        <NeonButton icon={<RefreshCcw />} onClick={() => query.refetch()}>
          REFRESH
        </NeonButton>
      </div>
      <StateShell
        isLoading={query.isLoading}
        isError={query.isError}
        error={query.error}
        isEmpty={items.length === 0}
        emptyHint="NO SCHEDULES · 无定时策略"
      >
        <div className="space-y-1.5">
          {items.map((s: BackupSchedule) => {
            const color = s.enabled ? '#00ff88' : '#525a78'
            return (
              <div
                key={s.id}
                className="fleet-row grid grid-cols-[10px_1.6fr_1fr_1.2fr_1fr] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: color }}
              >
                <span
                  className="size-2 rounded-full"
                  style={{ background: color, boxShadow: `0 0 8px ${color}` }}
                />
                <div className="min-w-0">
                  <div className="truncate font-display text-sm font-bold text-cyan-100">
                    {s.scheduleName}
                  </div>
                  <div className="font-mono text-[10px] text-cyan-300/55">
                    {BACKUP_TYPE_LABEL[s.backupType]} · {s.deviceGroups.length} 组 ·{' '}
                    {s.creator}
                  </div>
                </div>
                <div className="font-mono text-[11px] text-cyan-200/85">
                  <div>{s.cronExpression}</div>
                  <div className="text-[10px] text-cyan-300/55">
                    {s.cronDescription}
                  </div>
                </div>
                <div className="font-mono text-[10px] text-cyan-300/65">
                  <div>下次 {formatTime(s.nextRunTime)}</div>
                  <div className="text-cyan-300/45">
                    上次 {formatTime(s.lastRunTime)}
                  </div>
                </div>
                <div className="flex items-center justify-between">
                  <span className="font-mono text-[10px] text-cyan-300/55">
                    保留 {s.retentionDays}d
                  </span>
                  <span className="chip" style={{ color }}>
                    {s.enabled ? '启用' : '停用'}
                  </span>
                </div>
              </div>
            )
          })}
        </div>
        <Pager page={page} totalPages={totalPages} total={total} onPage={setPage} />
      </StateShell>
    </div>
  )
}

// ---------------------------------------------------------------------------
// FTP
// ---------------------------------------------------------------------------

function FtpSection() {
  const [page, setPage] = useState(1)
  const query = useFTPConfigs({ page, pageSize: PAGE_SIZE })
  const items = query.data?.items ?? []
  const total = query.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <div>
      <div className="mb-3 flex items-center justify-end">
        <NeonButton icon={<RefreshCcw />} onClick={() => query.refetch()}>
          REFRESH
        </NeonButton>
      </div>
      <StateShell
        isLoading={query.isLoading}
        isError={query.isError}
        error={query.error}
        isEmpty={items.length === 0}
        emptyHint="NO FTP CHANNELS · 无 FTP 通道"
      >
        <div className="space-y-1.5">
          {items.map((f: FTPConfig) => {
            const color = f.enabled ? '#00f0ff' : '#525a78'
            return (
              <div
                key={f.id}
                className="fleet-row grid grid-cols-[10px_1.4fr_1.6fr_1fr_0.8fr] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: color }}
              >
                <span
                  className="size-2 rounded-full"
                  style={{ background: color, boxShadow: `0 0 8px ${color}` }}
                />
                <div className="min-w-0">
                  <div className="truncate font-display text-sm font-bold text-cyan-100">
                    {f.configName}
                  </div>
                  <div className="font-mono text-[10px] text-cyan-300/55">
                    {f.protocol} · {f.passive ? 'PASSIVE' : 'ACTIVE'}
                  </div>
                </div>
                <div className="font-mono text-[11px] text-cyan-200/85">
                  <div>
                    {f.host}:{f.port}
                  </div>
                  <div className="truncate text-[10px] text-cyan-300/55">
                    {f.remotePath}
                  </div>
                </div>
                <div className="font-mono text-[11px] text-cyan-300/75">
                  {f.username}
                </div>
                <div className="text-right">
                  <span className="chip" style={{ color }}>
                    {f.enabled ? '启用' : '停用'}
                  </span>
                </div>
              </div>
            )
          })}
        </div>
        <Pager page={page} totalPages={totalPages} total={total} onPage={setPage} />
      </StateShell>
    </div>
  )
}

// ---------------------------------------------------------------------------
// POLICY（单例，只读概览 —— 写入是 v1 表单深度，本轮未做，见 notes）
// ---------------------------------------------------------------------------

function PolicySection() {
  const query = useBackupPolicy()
  const p = query.data

  return (
    <StateShell
      isLoading={query.isLoading}
      isError={query.isError}
      error={query.error}
      isEmpty={!p}
      emptyHint="NO POLICY · 无保留策略"
    >
      {p ? (
        <div className="space-y-3">
          {/* 存储用量仪表 + 关键阈值 */}
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
            <div className="glass relative flex items-center gap-4 overflow-hidden rounded-sm px-4 py-4">
              <div className="scanline" />
              <RadialGauge
                value={p.alertThresholdPercent}
                label="告警阈值"
                size={104}
                color={p.alertThresholdPercent >= 90 ? '#ff2d6f' : '#ffaa00'}
              />
              <div>
                <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
                  存储上限 · MAX
                </div>
                <div className="font-display text-2xl font-bold text-cyan-100 text-glow">
                  {p.maxStorageGB} GB
                </div>
                <div className="font-mono text-[10px] text-cyan-300/55">
                  {(p.storageBackend || '—').toUpperCase()} · {p.localPath || '—'}
                </div>
              </div>
            </div>

            <PolicyCard
              title="保留 · RETENTION"
              rows={[
                ['保留天数', `${p.retentionDays} 天`],
                ['保留份数', `${p.minBackupCount} ~ ${p.maxBackupCount}`],
                ['至少保留最近', `${p.keepLastN} 份`],
                ['自动清理', p.autoCleanup ? `开 · ${p.cleanupTime}` : '关'],
              ]}
            />

            <PolicyCard
              title="压缩 / 加密 · DATA"
              rows={[
                [
                  '压缩',
                  p.enableCompression
                    ? `${(p.compressionFormat || '').toUpperCase()} L${p.compressionLevel}`
                    : '关',
                ],
                [
                  '加密',
                  p.enableEncryption ? p.encryptionAlgorithm : '关',
                ],
                ['失败告警', p.alertOnFailure ? '开' : '关'],
                ['告警级别', (p.alertSeverity || '—').toUpperCase()],
              ]}
            />
          </div>

          <PolicyCard
            title="告警通知 · ALERT"
            rows={[
              ['告警邮箱', p.alertEmail || '未配置'],
              ['阈值百分比', `${p.alertThresholdPercent}%`],
              ['最近更新', formatTime(p.updatedAt)],
            ]}
            wide
          />

          <div className="font-mono text-[10px] text-cyan-300/40">
            策略字段为只读概览。编辑/下发为 v1 表单深度，本皮肤本轮未实现（见 notes）。
          </div>
        </div>
      ) : null}
    </StateShell>
  )
}

function PolicyCard({
  title,
  rows,
  wide,
}: {
  title: string
  rows: [string, string][]
  wide?: boolean
}) {
  return (
    <div className="glass relative overflow-hidden rounded-sm">
      <div className="scanline" />
      <div className="relative border-b border-cyan-500/15 px-4 py-2 font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/90">
        {title}
      </div>
      <div
        className={`relative grid gap-x-6 gap-y-2 px-4 py-3 ${
          wide ? 'grid-cols-1 sm:grid-cols-3' : 'grid-cols-1'
        }`}
      >
        {rows.map(([k, v]) => (
          <div key={k} className="flex items-center justify-between gap-3">
            <span className="font-mono text-[11px] text-cyan-300/55">{k}</span>
            <span className="font-display text-sm text-cyan-100">{v}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
