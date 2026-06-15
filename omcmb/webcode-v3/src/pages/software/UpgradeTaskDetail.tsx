import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { formatTime } from '@/lib/format'
import { useUpgradeTaskById, useSubTasks } from '@core/hooks/api/useSoftware'
import type { UpgradeSubTaskInfo } from '@core/mock/data/software'

import { NEON, TASK_STATUS, TASK_TYPE, RESULT_COLOR, SUB_STATUS, KV, MiniStat, Syncing, ErrorBlock, EmptyBlock } from './_shared'

// ---------------------------------------------------------------------------
// 升级任务详情 · software/upgrade-plan/:id （回退任务复用同路由族）
// useParams 取 id → useUpgradeTaskById + useSubTasks 加载真实数据
// ---------------------------------------------------------------------------

export default function UpgradeTaskDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: task, isLoading, isError, error } = useUpgradeTaskById(id)
  const subs = useSubTasks(id, { page: 1, pageSize: 200 })
  const subList: UpgradeSubTaskInfo[] = subs.data?.items ?? []

  const counts = useMemo(() => {
    const c: Record<string, number> = {}
    for (const s of subList) c[s.status] = (c[s.status] ?? 0) + 1
    return c
  }, [subList])

  const sc = task ? TASK_STATUS[task.status] ?? { label: task.status, color: NEON.dim } : null
  const tc = task ? TASK_TYPE[task.taskType] ?? { label: `TYPE ${task.taskType}`, color: NEON.cyan } : null
  const pct = task && task.totalCount ? Math.round(((task.successCount + task.failCount) / task.totalCount) * 100) : 0

  return (
    <PageShell
      code="F06"
      title="UPGRADE TASK DETAIL · 升级任务详情"
      subtitle={task ? task.taskName : `TASK ${id || '—'}`}
      isFetching={subs.isFetching}
      bare
      toolbar={
        <NeonButton icon={<ArrowLeft />} onClick={() => navigate(-1)}>
          返回
        </NeonButton>
      }
    >
      {isLoading ? (
        <Syncing label="LOADING TASK…" />
      ) : isError ? (
        <ErrorBlock msg={error instanceof Error ? error.message : '未知错误'} />
      ) : !task ? (
        <EmptyBlock label="TASK NOT FOUND · 任务不存在" />
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
          {/* 概览 */}
          <GlassPanel strong title="OVERVIEW · 任务概览" className="lg:col-span-1">
            <div className="space-y-4 p-4">
              <div className="grid grid-cols-2 gap-3">
                <KV label="状态">
                  <span className="chip" style={{ color: sc?.color }}>{sc?.label}</span>
                </KV>
                <KV label="类型">
                  <span className="chip" style={{ color: tc?.color }}>{tc?.label}</span>
                </KV>
                <KV label="制式">{task.productClass || '—'}</KV>
                <KV label="操作员">{task.createUser || '—'}</KV>
                <KV label="固件">
                  <span className="break-all font-mono text-[11px]">{task.fileName || '—'}</span>
                </KV>
                <KV label="并发">{task.maxConcurrent}</KV>
                <KV label="保留配置">{task.isKeepConfig ? '是' : '否'}</KV>
                {task.result ? (
                  <KV label="结果">
                    <span style={{ color: RESULT_COLOR[task.result] ?? NEON.dim }}>{task.result}</span>
                  </KV>
                ) : null}
              </div>

              <div className="flex items-center justify-center py-1">
                <RadialGauge
                  value={pct}
                  label="PROGRESS"
                  size={120}
                  color={task.result === 'failed' ? NEON.rose : pct >= 100 ? NEON.green : NEON.cyan}
                />
              </div>

              <div className="grid grid-cols-2 gap-1 font-mono text-[10px] text-cyan-300/55">
                <div>开始 {task.startedAt ? formatTime(task.startedAt) : '—'}</div>
                <div>结束 {task.endedAt ? formatTime(task.endedAt) : '—'}</div>
                <div>创建 {formatTime(task.createdAt)}</div>
                <div>更新 {formatTime(task.updatedAt)}</div>
              </div>
            </div>
          </GlassPanel>

          {/* 子任务设备明细 */}
          <GlassPanel
            strong
            title="DEVICE SUB-TASKS · 设备明细"
            meta={subs.isFetching ? 'LIVE · 3s' : `${subList.length} 台`}
            className="lg:col-span-2"
          >
            <div className="space-y-3 p-4">
              <div className="grid grid-cols-4 gap-2">
                <MiniStat label="TOTAL" value={subList.length} color={NEON.cyan} />
                <MiniStat label="成功" value={counts.completed ?? 0} color={NEON.green} />
                <MiniStat label="进行" value={(counts.downloading ?? 0) + (counts.rebooting ?? 0) + (counts.verifying ?? 0)} color={NEON.amber} />
                <MiniStat label="失败" value={counts.failed ?? 0} color={NEON.rose} />
              </div>

              {subs.isLoading ? (
                <Syncing label="LOADING SUB-TASKS…" />
              ) : subs.isError ? (
                <ErrorBlock msg={subs.error instanceof Error ? subs.error.message : '未知错误'} />
              ) : subList.length === 0 ? (
                <EmptyBlock label="NO SUB-TASKS · 无设备子任务" />
              ) : (
                <div className="space-y-1.5">
                  {subList.map((s) => {
                    const ss = SUB_STATUS[s.status] ?? { label: s.status, color: NEON.dim }
                    return (
                      <div
                        key={s.id}
                        className="fleet-row grid grid-cols-[10px_1.6fr_1fr_120px] items-center gap-3 rounded-sm px-3 py-2"
                        style={{ ['--row-color' as never]: ss.color }}
                      >
                        <span className="size-2 rounded-full" style={{ background: ss.color, boxShadow: `0 0 6px ${ss.color}` }} />
                        <div className="min-w-0">
                          <div className="truncate font-mono text-xs text-cyan-100">{s.deviceSn || s.deviceId}</div>
                          <div className="truncate font-mono text-[10px] text-cyan-300/55">
                            {/* qa-614 #371：目标版本完成后由后端回填，未回报时显示"待上报"而非裸 —。 */}
                            {s.oriVersion || '—'} → {s.destVersion || '待上报'}
                            {s.retryCount > 0 ? ` · 重试 ${s.retryCount}/${s.maxRetries}` : ''}
                          </div>
                          {s.failureReason || s.errorMessage ? (
                            <div className="truncate font-mono text-[10px] text-rose-300/70">{s.failureReason || s.errorMessage}</div>
                          ) : null}
                        </div>
                        {/* qa-614 #371：子任务起止时间——5G 回退起始时间现已记录，分行展示开始/结束。 */}
                        <div className="font-mono text-[10px] leading-tight text-cyan-300/65">
                          <div>始 {s.startedAt ? formatTime(s.startedAt) : '—'}</div>
                          <div>终 {s.completedAt ? formatTime(s.completedAt) : '—'}</div>
                        </div>
                        <div className="text-right">
                          <span className="chip" style={{ color: ss.color }}>
                            <span className="size-1.5 rounded-full bg-current shadow-[0_0_6px_currentColor]" />
                            {ss.label}
                          </span>
                        </div>
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          </GlassPanel>
        </div>
      )}
    </PageShell>
  )
}
