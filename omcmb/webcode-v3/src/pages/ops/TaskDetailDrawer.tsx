import { useMemo } from 'react'

import { useTaskExecutions } from '@core/hooks/api/useOpsExt'
import type { OpsTask } from '@core/mock/data/opsTools'
import { formatTime } from '@/lib/format'
import { Drawer, Field, SectionTitle } from './Modal'

const STATUS_COLOR: Record<OpsTask['status'], string> = {
  pending: '#5b9eff',
  running: '#00f0ff',
  paused: '#ffaa00',
  success: '#00ff88',
  failed: '#ff2d6f',
  cancelled: '#525a78',
}
const STATUS_LABEL: Record<OpsTask['status'], string> = {
  pending: '待执行',
  running: '执行中',
  paused: '已暂停',
  success: '成功',
  failed: '失败',
  cancelled: '已取消',
}

export function TaskDetailDrawer({
  task,
  open,
  onClose,
  templateName,
}: {
  task: OpsTask | null
  open: boolean
  onClose: () => void
  templateName?: string
}) {
  // 下钻：逐设备步骤执行流水（T-0101，仅在真实后端可用；mock 下返回空集）
  const { data: execData, isLoading: execLoading } = useTaskExecutions(
    open ? (task?.id ?? undefined) : undefined,
    { page: 1, pageSize: 200 }
  )
  const executions = useMemo(() => execData?.items ?? [], [execData])

  if (!open || !task) return null
  const color = STATUS_COLOR[task.status]

  return (
    <Drawer
      open={open}
      onClose={onClose}
      width={560}
      title={task.taskName}
      badge={
        <span className="chip shrink-0" style={{ color }}>
          {STATUS_LABEL[task.status]}
        </span>
      }
    >
      <div className="space-y-4 p-4">
        {/* 进度 */}
        <div className="glass rounded-sm p-4">
          <div className="mb-2 flex items-center justify-between">
            <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/60">
              PROGRESS · 进度
            </span>
            <span className="font-display text-lg font-bold" style={{ color }}>
              {task.progress}%
            </span>
          </div>
          <div className="h-1.5 w-full overflow-hidden rounded-full bg-cyan-500/10">
            <div
              className="h-full rounded-full transition-all"
              style={{ width: `${task.progress}%`, background: color, boxShadow: `0 0 8px ${color}` }}
            />
          </div>
          <div className="mt-2 flex items-center justify-between font-mono text-[10px] text-cyan-300/55">
            <span>
              STEP {task.currentStep}/{task.totalSteps}
            </span>
            <span>
              <span className="text-[#00ff88]">成功 {task.successCount}</span>
              {' · '}
              <span className={task.failCount > 0 ? 'text-[#ff2d6f]' : 'text-cyan-300/55'}>
                失败 {task.failCount}
              </span>
              {' · '}总 {task.totalCount}
            </span>
          </div>
        </div>

        {/* 基本信息 */}
        <div className="glass rounded-sm">
          <SectionTitle>基本信息 · BASIC</SectionTitle>
          <Field label="任务名称">{task.taskName}</Field>
          <Field label="关联模板">{templateName ?? task.templateId ?? '—'}</Field>
          <Field label="发起人">{task.creator}</Field>
          <Field label="创建时间">{formatTime(task.createdAt)}</Field>
          {task.startedAt ? <Field label="开始时间">{formatTime(task.startedAt)}</Field> : null}
          {task.completedAt ? (
            <Field label="完成时间">{formatTime(task.completedAt)}</Field>
          ) : null}
          {task.message ? <Field label="结果消息">{task.message}</Field> : null}
        </div>

        {/* 目标设备 */}
        <div className="glass rounded-sm">
          <SectionTitle>目标设备 · TARGETS ({task.deviceSns.length})</SectionTitle>
          <div className="flex flex-wrap gap-1.5 p-3.5">
            {task.deviceSns.map((sn) => (
              <span
                key={sn}
                className="rounded-sm border border-cyan-500/25 bg-cyan-500/5 px-2 py-0.5 font-mono text-[11px] text-cyan-100/85"
              >
                {sn}
              </span>
            ))}
          </div>
        </div>

        {/* 下钻：逐设备步骤流水 */}
        <div className="glass rounded-sm">
          <SectionTitle>执行流水 · EXECUTIONS</SectionTitle>
          {execLoading ? (
            <div className="px-3.5 py-4 font-mono text-[11px] text-cyan-300/55">SYNC…</div>
          ) : executions.length === 0 ? (
            <div className="px-3.5 py-4 font-mono text-[11px] text-cyan-300/45">
              // 暂无逐设备执行流水（mock 模式不提供 / 任务尚未触发执行）
            </div>
          ) : (
            <div className="max-h-72 overflow-auto">
              {executions.map((e) => {
                const ok = e.status === 'success' || e.status === 'complete'
                const sc = ok ? '#00ff88' : e.status === 'failed' ? '#ff2d6f' : '#ffaa00'
                return (
                  <div
                    key={e.id}
                    className="grid grid-cols-[1.4fr_1fr_70px_70px] items-center gap-2 border-b border-cyan-500/8 px-3.5 py-1.5 last:border-b-0"
                  >
                    <span className="truncate font-mono text-[11px] text-cyan-100/85">
                      #{e.step_index} {e.step_name}
                    </span>
                    <span className="truncate font-mono text-[10px] text-cyan-300/60">
                      {e.device_sn}
                    </span>
                    <span className="font-mono text-[10px] uppercase" style={{ color: sc }}>
                      {e.status}
                    </span>
                    <span className="text-right font-mono text-[10px] text-cyan-300/55">
                      {e.duration_ms}ms
                    </span>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </div>
    </Drawer>
  )
}
