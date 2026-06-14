import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  ArrowLeft,
  Loader2,
  RefreshCcw,
  Send,
  PauseCircle,
  StopCircle,
  RotateCcw,
  Trash2,
  Cpu,
  Download,
  AlertTriangle,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { formatTime } from '@/lib/format'
import {
  useUnifiedFileTransferTasks,
  useUnifiedFileTransferDevices,
  useStartUfteTask,
  useSuspendUfteTask,
  useTerminateUfteTask,
  useRetryUfteTask,
  useDeleteUfteTask,
} from '@core/hooks/api/useUnifiedFileTransfer'
import type {
  UnifiedFileTransferTask,
  UnifiedFileTransferDeviceItem,
  UnifiedFileTransferDeviceStatus,
  TransferTaskStatus,
  TransferTaskResult,
} from '@core/types/unifiedFileTransfer'

const STATUS_BADGE: Record<TransferTaskStatus, { status: string; label: string }> = {
  pending: { status: 'unknown', label: '待执行' },
  in_progress: { status: 'active', label: '执行中' },
  suspended: { status: 'warning', label: '已挂起' },
  ended: { status: 'offline', label: '已结束' },
}

const RESULT_BADGE: Record<TransferTaskResult, { status: string; label: string }> = {
  success: { status: 'ok', label: '成功' },
  partial: { status: 'warning', label: '部分成功' },
  failure: { status: 'critical', label: '失败' },
  terminated: { status: 'offline', label: '已终止' },
}

const DEV_STATUS_BADGE: Record<UnifiedFileTransferDeviceStatus, { status: string; label: string }> = {
  pending: { status: 'unknown', label: '待执行' },
  downloading: { status: 'active', label: '下载中' },
  uploading: { status: 'active', label: '上传中' },
  awaiting_tc: { status: 'warning', label: '等待完成' },
  verifying: { status: 'warning', label: '校验中' },
  suspended: { status: 'warning', label: '已挂起' },
  ended: { status: 'ok', label: '已完成' },
  failed: { status: 'critical', label: '失败' },
}

export default function TransferTaskDetailPage() {
  const { taskId = '' } = useParams<{ taskId: string }>()
  const navigate = useNavigate()
  const [opError, setOpError] = useState<string | null>(null)

  // 任务摘要：UFTE 无单任务接口，拉一页较大的任务列表按 id 命中。
  const tasksQuery = useUnifiedFileTransferTasks({ page: 1, pageSize: 200 })
  const task = useMemo<UnifiedFileTransferTask | undefined>(
    () => (tasksQuery.data?.items ?? []).find((t) => t.id === taskId),
    [tasksQuery.data, taskId],
  )

  // 设备执行明细：用 typeCode 缩小范围，再按 taskId 客户端过滤（device item 携带 taskId）。
  const devicesQuery = useUnifiedFileTransferDevices({
    page: 1,
    pageSize: 200,
    ...(task?.typeCode ? { typeCode: task.typeCode } : {}),
  })
  const devices = useMemo<UnifiedFileTransferDeviceItem[]>(
    () => (devicesQuery.data?.items ?? []).filter((d) => d.taskId === taskId),
    [devicesQuery.data, taskId],
  )

  const startTask = useStartUfteTask()
  const suspendTask = useSuspendUfteTask()
  const terminateTask = useTerminateUfteTask()
  const retryTask = useRetryUfteTask()
  const deleteTask = useDeleteUfteTask()
  const submitting =
    startTask.isPending ||
    suspendTask.isPending ||
    terminateTask.isPending ||
    retryTask.isPending ||
    deleteTask.isPending

  const refetchAll = async () => {
    await Promise.all([tasksQuery.refetch(), devicesQuery.refetch()])
  }
  const runOp = async (fn: () => Promise<unknown>, after?: () => void) => {
    setOpError(null)
    try {
      await fn()
      if (after) after()
      else await refetchAll()
    } catch (e) {
      setOpError(e instanceof Error ? e.message : '操作失败')
    }
  }

  const back = (
    <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/transfer/center')}>
      BACK
    </NeonButton>
  )

  if (tasksQuery.isLoading) {
    return (
      <PageShell code="F06" title="TRANSFER TASK · 任务详情" subtitle="LOADING" toolbar={back}>
        <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
        </div>
      </PageShell>
    )
  }

  if (tasksQuery.isError || !task) {
    return (
      <PageShell code="F06" title="TRANSFER TASK · 任务详情" subtitle="ERROR" toolbar={back}>
        <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
          {tasksQuery.isError
            ? `FAILURE · ${tasksQuery.error instanceof Error ? tasksQuery.error.message : '未知错误'}`
            : 'NOT FOUND · 任务不存在或已删除'}
        </div>
      </PageShell>
    )
  }

  const statusBadge = STATUS_BADGE[task.status]
  const resultBadge = task.result ? RESULT_BADGE[task.result] : undefined
  const canStart = task.status === 'pending' || task.status === 'suspended'
  const canSuspend = task.status === 'in_progress'
  const canTerminate = task.status === 'in_progress' || task.status === 'pending'
  const canRetry = task.status === 'ended' && (task.result === 'failure' || task.result === 'partial')
  const pct = Math.max(0, Math.min(100, Math.round(task.progress)))

  return (
    <PageShell
      code="F06"
      title={`TASK · ${task.taskName}`}
      subtitle={`${task.categoryLabel} · ${task.typeDisplayName}`}
      isFetching={tasksQuery.isFetching || devicesQuery.isFetching}
      toolbar={
        <>
          {back}
          <NeonButton icon={<RefreshCcw />} onClick={() => void refetchAll()}>
            REFRESH
          </NeonButton>
          {canStart && (
            <NeonButton
              icon={<Send />}
              disabled={submitting}
              onClick={() => void runOp(() => startTask.mutateAsync(task.id))}
            >
              启动
            </NeonButton>
          )}
          {canSuspend && (
            <NeonButton
              icon={<PauseCircle />}
              disabled={submitting}
              onClick={() => void runOp(() => suspendTask.mutateAsync(task.id))}
            >
              挂起
            </NeonButton>
          )}
          {canRetry && (
            <NeonButton
              icon={<RotateCcw />}
              disabled={submitting}
              onClick={() => void runOp(() => retryTask.mutateAsync(task.id))}
            >
              重试
            </NeonButton>
          )}
          {canTerminate && (
            <NeonButton
              tone="danger"
              icon={<StopCircle />}
              disabled={submitting}
              onClick={() => void runOp(() => terminateTask.mutateAsync(task.id))}
            >
              终止
            </NeonButton>
          )}
          <NeonButton
            tone="danger"
            icon={<Trash2 />}
            disabled={submitting}
            onClick={() => void runOp(() => deleteTask.mutateAsync(task.id), () => navigate('/transfer/center'))}
          >
            删除
          </NeonButton>
        </>
      }
    >
      {opError && (
        <div className="mb-3 border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">
          OP FAILED · {opError}
          {submitting && <Loader2 className="ml-2 inline size-3 animate-spin" />}
        </div>
      )}

      {/* 任务概览 */}
      <div className="mb-4 grid grid-cols-12 gap-3">
        <GlassPanel title="TASK PROGRESS · 任务进度" className="col-span-12 lg:col-span-4">
          <div className="flex items-center justify-around gap-2 px-3 py-4">
            <RadialGauge
              value={pct}
              label="进度"
              size={120}
              color={task.result === 'failure' ? '#ff2d6f' : task.result === 'partial' ? '#ffaa00' : '#00f0ff'}
            />
            <div className="flex flex-col gap-2">
              <CountStat label="总数" value={task.totalCount} color="#5b9eff" />
              <CountStat label="成功" value={task.successCount} color="#00ff88" />
              <CountStat label="失败" value={task.failCount} color="#ff2d6f" />
            </div>
          </div>
        </GlassPanel>

        <GlassPanel title="TASK META · 任务元数据" meta="REGISTRY" className="col-span-12 lg:col-span-8">
          <dl className="grid grid-cols-1 divide-y divide-cyan-500/8 sm:grid-cols-2 sm:divide-y-0">
            <Field
              label="状态"
              valueNode={
                <div className="flex items-center gap-1.5">
                  <StatusBadge status={statusBadge.status} label={statusBadge.label} />
                  {resultBadge && <StatusBadge status={resultBadge.status} label={resultBadge.label} />}
                </div>
              }
            />
            <Field label="类型编码" value={task.typeCode} />
            <Field label="执行方式" value={task.executionMode} />
            <Field label="创建人" value={task.createUser} />
            <Field label="创建时间" value={formatTime(task.createdAt)} />
            {task.scheduledAt && <Field label="计划时间" value={formatTime(task.scheduledAt)} />}
            {task.productType && <Field label="产品型号" value={task.productType} />}
            {task.targetVersion && <Field label="目标版本" value={task.targetVersion} />}
            {task.isKeepConfig !== undefined && (
              <Field
                label="保留配置"
                valueNode={
                  <StatusBadge
                    status={task.isKeepConfig ? 'ok' : 'offline'}
                    label={task.isKeepConfig ? '是' : '否'}
                  />
                }
              />
            )}
            {task.operatorScope && <Field label="运营商" value={task.operatorScope} />}
          </dl>
        </GlassPanel>
      </div>

      {/* 设备执行明细 */}
      <GlassPanel title="DEVICE EXECUTION · 设备执行明细" meta={`${devices.length} DEVICES`}>
        {devicesQuery.isLoading ? (
          <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
            <Loader2 className="size-4 animate-spin" />
            <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
          </div>
        ) : devicesQuery.isError ? (
          <div className="m-3 border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
            FAILURE · {devicesQuery.error instanceof Error ? devicesQuery.error.message : '未知错误'}
          </div>
        ) : devices.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-3 py-14">
            <Cpu className="size-9 text-cyan-400/45" />
            <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
              NO DEVICE RECORDS · 暂无设备明细
            </div>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <div className="mb-1 grid min-w-[820px] grid-cols-[2fr_1.4fr_120px_1.2fr_1.2fr_1.6fr] items-center gap-3 px-3.5 pt-3 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
              <span>DEVICE · 设备</span>
              <span>VERSION · 版本</span>
              <span>STATUS</span>
              <span>PROGRESS</span>
              <span>FILE · 文件</span>
              <span>NOTE · 失败信息</span>
            </div>
            <div className="min-w-[820px] divide-y divide-cyan-500/8">
              {devices.map((d) => {
                const dBadge = DEV_STATUS_BADGE[d.status]
                const dpct = Math.max(0, Math.min(100, Math.round(d.progress)))
                return (
                  <div
                    key={d.id}
                    className="grid grid-cols-[2fr_1.4fr_120px_1.2fr_1.2fr_1.6fr] items-center gap-3 px-3.5 py-2.5 hover:bg-cyan-500/5"
                  >
                    <div className="min-w-0">
                      <div className="truncate text-xs text-cyan-100/90">{d.deviceName || '—'}</div>
                      <div className="truncate font-mono text-[10px] text-cyan-300/55">{d.deviceSn}</div>
                    </div>
                    <div className="min-w-0 font-mono text-[11px] text-cyan-300/75">
                      <div className="truncate">{d.currentVersion || '—'}</div>
                      {d.targetVersion && (
                        <div className="truncate text-cyan-300/55">→ {d.targetVersion}</div>
                      )}
                    </div>
                    <div className="flex flex-col items-start gap-1">
                      <StatusBadge status={dBadge.status} label={dBadge.label} className="scale-90" />
                      {d.result && (
                        <StatusBadge
                          status={RESULT_BADGE[d.result].status}
                          label={RESULT_BADGE[d.result].label}
                          className="scale-90"
                        />
                      )}
                    </div>
                    <div className="min-w-0">
                      <div className="mb-1 font-mono text-[10px] text-cyan-300/55">{dpct}%</div>
                      <div className="h-1.5 overflow-hidden rounded-full bg-cyan-500/10">
                        <div
                          className="h-full rounded-full"
                          style={{
                            width: `${dpct}%`,
                            background: d.status === 'failed' ? '#ff2d6f' : '#00f0ff',
                            boxShadow: `0 0 6px ${d.status === 'failed' ? '#ff2d6f' : '#00f0ff'}`,
                          }}
                        />
                      </div>
                    </div>
                    <div className="min-w-0">
                      {d.targetFile ? (
                        d.downloadUrl ? (
                          <a
                            href={d.downloadUrl}
                            target="_blank"
                            rel="noreferrer"
                            className="flex items-center gap-1 truncate font-mono text-[11px] text-cyan-300 hover:text-cyan-100"
                            title={d.targetFile}
                          >
                            <Download className="size-3 shrink-0" />
                            <span className="truncate">{d.targetFile}</span>
                          </a>
                        ) : (
                          <span className="truncate font-mono text-[11px] text-cyan-300/45" title={d.targetFile}>
                            {d.targetFile}
                          </span>
                        )
                      ) : (
                        <span className="text-cyan-300/40">—</span>
                      )}
                    </div>
                    <div className="min-w-0">
                      {d.failureReason || d.failureDetail ? (
                        <div
                          className="flex items-start gap-1 text-[11px] text-rose-300/85"
                          title={d.failureDetail || d.failureReason}
                        >
                          <AlertTriangle className="mt-0.5 size-3 shrink-0" />
                          <span className="line-clamp-2 break-all">{d.failureReason || d.failureDetail}</span>
                        </div>
                      ) : (
                        <span className="text-cyan-300/40">—</span>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        )}
      </GlassPanel>
    </PageShell>
  )
}

function CountStat({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div className="flex items-baseline gap-2">
      <div
        className="font-display text-lg font-bold leading-none"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
      <div className="font-mono text-[10px] uppercase tracking-[0.14em] text-cyan-300/55">{label}</div>
    </div>
  )
}

function Field({
  label,
  value,
  valueNode,
}: {
  label: string
  value?: string
  valueNode?: React.ReactNode
}) {
  return (
    <div className="flex items-center justify-between gap-4 px-3.5 py-2">
      <dt className="shrink-0 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">
        {label}
      </dt>
      <dd className="min-w-0 truncate text-right text-xs text-cyan-100/90">
        {valueNode ?? (value && value.trim() ? value : <span className="text-cyan-300/40">—</span>)}
      </dd>
    </div>
  )
}
