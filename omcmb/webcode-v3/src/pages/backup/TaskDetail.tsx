import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  ArrowLeft,
  RefreshCcw,
  Loader2,
  StopCircle,
  Trash2,
  AlertTriangle,
  HardDrive,
  Cpu,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { formatBytes, formatTime } from '@/lib/format'
import {
  useBackupTaskById,
  useCancelBackupTask,
  useDeleteBackupTasks,
} from '@core/hooks/api/useBackup'
import type { BackupTask } from '@core/mock/data/backup'

import { Modal } from './Modal'

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

function getErrMsg(e: unknown): string {
  if (e instanceof Error) return e.message
  if (typeof e === 'object' && e && 'message' in e) {
    return String((e as { message: unknown }).message)
  }
  return '未知错误'
}

// ---------------------------------------------------------------------------
// 备份任务详情 · /backup/tasks/:id
// ---------------------------------------------------------------------------

export default function BackupTaskDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const query = useBackupTaskById(id ?? '')
  const cancelTask = useCancelBackupTask()
  const deleteTasks = useDeleteBackupTasks()

  const task = query.data as BackupTask | undefined

  const [confirm, setConfirm] = useState<'cancel' | 'delete' | null>(null)
  const [actionMsg, setActionMsg] = useState<string | null>(null)

  const onConfirm = async () => {
    if (!task || !confirm) return
    try {
      if (confirm === 'cancel') {
        await cancelTask.mutateAsync(task.id)
        setActionMsg(`已取消任务 ${task.taskName}`)
      } else {
        await deleteTasks.mutateAsync([task.id])
        setActionMsg(`已删除任务 ${task.taskName}`)
      }
      setConfirm(null)
    } catch (e) {
      setActionMsg(`操作失败：${getErrMsg(e)}`)
    }
  }

  const color = task ? TASK_STATUS_COLOR[task.status] : '#00f0ff'
  const inflight = task?.status === 'running' || task?.status === 'pending'

  return (
    <PageShell
      code="F06"
      title="TASK DETAIL · 备份任务详情"
      subtitle={id ? `TASK ${id}` : 'BACKUP TASK'}
      isFetching={query.isFetching || undefined}
      bare
      toolbar={
        <div className="flex flex-wrap items-center gap-2">
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/backup/tasks')}>
            返回列表
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => query.refetch()}>
            REFRESH
          </NeonButton>
        </div>
      }
    >
      {query.isLoading ? (
        <div className="flex items-center justify-center gap-2 py-20 text-cyan-300/60">
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
        </div>
      ) : query.isError ? (
        <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
          FAILURE · {getErrMsg(query.error)}
        </div>
      ) : !task ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <AlertTriangle className="size-10 text-amber-400/60" />
          <div className="font-mono text-xs uppercase tracking-[0.2em] text-amber-300/70">
            NOT FOUND · 任务不存在或已删除
          </div>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/backup/tasks')}>
            返回备份任务列表
          </NeonButton>
        </div>
      ) : (
        <div className="space-y-3">
          {/* 头部概览 */}
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
            <div className="glass relative flex items-center gap-4 overflow-hidden rounded-sm px-4 py-4 lg:col-span-1">
              <div className="scanline" />
              <RadialGauge
                value={Math.min(100, Math.max(0, task.progress ?? 0))}
                label="进度"
                size={108}
                color={color}
              />
              <div className="min-w-0">
                <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
                  状态 · STATE
                </div>
                <div
                  className="font-display text-2xl font-bold leading-tight text-glow"
                  style={{ color }}
                >
                  {TASK_STATUS_LABEL[task.status]}
                </div>
                <div className="font-mono text-[10px] text-cyan-300/55">
                  {task.taskType === 'scheduled' ? '定时任务' : '手动任务'} ·{' '}
                  {BACKUP_TYPE_LABEL[task.backupType]}
                </div>
              </div>
            </div>

            <GlassPanel
              title="ARCHIVE · 归档"
              meta={task.id}
              className="lg:col-span-2"
            >
              <div className="grid grid-cols-2 gap-x-6 gap-y-2 px-4 py-3 sm:grid-cols-3">
                <KV label="任务名称" value={task.taskName || '—'} icon={<HardDrive className="size-3.5" />} />
                <KV label="设备总数" value={`${task.totalCount ?? 0} 台`} icon={<Cpu className="size-3.5" />} />
                <KV label="文件大小" value={formatBytes(task.fileSize)} />
                <KV label="成功 / 失败" value={`${task.successCount ?? 0} / ${task.failCount ?? 0}`} />
                <KV label="创建人" value={task.creator || '—'} />
                <KV label="存储位置" value={task.storageLocation || '—'} />
                <KV label="创建时间" value={formatTime(task.createdAt)} />
                <KV label="更新时间" value={formatTime(task.updatedAt)} />
              </div>
            </GlassPanel>
          </div>

          {/* 目标设备 */}
          <GlassPanel title="TARGET DEVICES · 目标设备" meta={`${task.deviceSns.length} 台`}>
            <div className="flex flex-wrap gap-1.5 px-4 py-3">
              {task.deviceSns.length === 0 ? (
                <span className="font-mono text-[11px] text-cyan-300/45">—</span>
              ) : (
                task.deviceSns.map((sn) => (
                  <span
                    key={sn}
                    className="font-mono text-[11px] rounded-sm border border-cyan-500/20 bg-cyan-500/[0.04] px-2 py-1 text-cyan-200/85"
                  >
                    {sn}
                  </span>
                ))
              )}
            </div>
          </GlassPanel>

          {/* 执行消息 */}
          {task.message ? (
            <GlassPanel title="MESSAGE · 执行消息">
              <p
                className="px-4 py-3 font-mono text-xs"
                style={{ color: task.status === 'failed' ? '#ff9bb6' : '#9fd9ff' }}
              >
                {task.message}
              </p>
            </GlassPanel>
          ) : null}

          {/* 操作 */}
          <div className="flex gap-2">
            {inflight ? (
              <NeonButton icon={<StopCircle />} onClick={() => setConfirm('cancel')}>
                取消任务
              </NeonButton>
            ) : (
              <NeonButton tone="danger" icon={<Trash2 />} onClick={() => setConfirm('delete')}>
                删除任务
              </NeonButton>
            )}
          </div>
        </div>
      )}

      <Modal
        open={Boolean(confirm)}
        title={confirm === 'cancel' ? '取消备份任务' : '删除备份任务'}
        subtitle={confirm === 'cancel' ? 'ABORT TASK' : 'PURGE TASK'}
        onClose={() => setConfirm(null)}
        width={460}
        footer={
          <>
            <NeonButton onClick={() => setConfirm(null)}>取消</NeonButton>
            <NeonButton
              tone="danger"
              disabled={cancelTask.isPending || deleteTasks.isPending}
              onClick={() => {
                void onConfirm()
              }}
            >
              {cancelTask.isPending || deleteTasks.isPending ? '处理中…' : '确认'}
            </NeonButton>
          </>
        }
      >
        <p className="text-sm text-cyan-100/85">
          {confirm === 'cancel'
            ? '将中止该任务剩余设备的备份执行，已完成的快照保留。'
            : '将永久删除该任务记录，已生成的快照文件按保留策略另行清理。'}
        </p>
        <p className="mt-2 font-mono text-xs text-cyan-300/70">
          {task?.taskName} · {task?.id}
        </p>
      </Modal>

      <Modal
        open={Boolean(actionMsg)}
        title="操作结果"
        subtitle="RESULT"
        onClose={() => {
          setActionMsg(null)
          // 删除成功后回到列表
          if (actionMsg?.startsWith('已删除')) navigate('/backup/tasks')
        }}
        width={420}
        footer={
          <NeonButton
            onClick={() => {
              const msg = actionMsg
              setActionMsg(null)
              if (msg?.startsWith('已删除')) navigate('/backup/tasks')
            }}
          >
            知道了
          </NeonButton>
        }
      >
        <p className="text-sm text-cyan-100/85">{actionMsg}</p>
      </Modal>
    </PageShell>
  )
}

function KV({
  label,
  value,
  icon,
}: {
  label: string
  value: string
  icon?: React.ReactNode
}) {
  return (
    <div className="flex flex-col gap-0.5">
      <span className="flex items-center gap-1 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">
        {icon}
        {label}
      </span>
      <span className="truncate font-display text-sm text-cyan-100" title={value}>
        {value}
      </span>
    </div>
  )
}
