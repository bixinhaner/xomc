import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, RefreshCcw, Power } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useMRTask, useMRTaskProgress, useStopMRTask } from '@core/hooks/api/useMrTasks'
import type { MRTaskStatus, MRProgressStatus, MRHealthStatus } from '@core/types/mrTask'

import { KV, OverviewStat, Pager, Row, StateGate, NEON } from './_shared'

const PAGE_SIZE = 50

const TASK_STATUS_META: Record<MRTaskStatus, { token: string; label: string }> = {
  waitting: { token: 'unknown', label: '待开启' },
  on: { token: 'ok', label: '上报中' },
  off: { token: 'off', label: '已结束' },
  suspend: { token: 'warning', label: '已暂停' },
  termination: { token: 'major', label: '已终止' },
}

const PROGRESS_META: Record<MRProgressStatus, { token: string; label: string }> = {
  pending: { token: 'unknown', label: '待下发' },
  openSuccess: { token: 'ok', label: '开启成功' },
  openFailure: { token: 'error', label: '开启失败' },
  closeSuccess: { token: 'off', label: '关闭成功' },
  closeFailure: { token: 'error', label: '关闭失败' },
  unsupport: { token: 'minor', label: '不支持' },
  timeOut: { token: 'major', label: '超时' },
  noPermission: { token: 'major', label: '无权限' },
}

const HEALTH_META: Record<MRHealthStatus, { token: string; label: string }> = {
  normal: { token: 'ok', label: '正常' },
  abnormal: { token: 'error', label: '异常' },
  unknown: { token: 'unknown', label: '未知' },
}

const MR_TYPE_COLOR: Record<string, string> = {
  MRO: NEON.cyan,
  MRE: NEON.green,
  MRS: NEON.amber,
}

export default function MRRetrievalDetail() {
  const { taskId = '' } = useParams<{ taskId: string }>()
  const navigate = useNavigate()
  const [page, setPage] = useState(1)

  const { data: task, isLoading, isError, error, isFetching, refetch } = useMRTask(taskId)
  const progress = useMRTaskProgress(taskId, { page, pageSize: PAGE_SIZE })
  const stopTask = useStopMRTask()

  const items = progress.data?.items ?? []
  const progressTotal = progress.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(progressTotal / PAGE_SIZE))

  const mrTypes = useMemo(
    () => (task ? task.mrType.split(',').map((s) => s.trim()).filter(Boolean) : []),
    [task]
  )

  const okCount = items.filter((i) => i.progressStatus === 'openSuccess').length
  const failCount = items.filter((i) => i.progressStatus === 'openFailure' || i.progressStatus === 'closeFailure').length

  return (
    <PageShell
      code="F05"
      title={task ? `MR PULL · ${task.taskName}` : `MR PULL · ${taskId.slice(0, 8)}`}
      subtitle="MR TASK DOSSIER · CELL PROGRESS"
      isFetching={isFetching || progress.isFetching}
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/file/mr-retrieval')}>
            返回列表
          </NeonButton>
          <NeonButton
            icon={<RefreshCcw />}
            onClick={() => {
              void refetch()
              void progress.refetch()
            }}
          >
            REFRESH
          </NeonButton>
          {task && (task.taskStatus === 'on' || task.taskStatus === 'waitting') ? (
            <NeonButton tone="danger" icon={<Power />} disabled={stopTask.isPending} onClick={() => stopTask.mutate(taskId)}>
              {stopTask.isPending ? '停止中…' : '停止任务'}
            </NeonButton>
          ) : null}
        </>
      }
    >
      <StateGate
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={!task}
        loadingLabel="LOADING TASK…"
        emptyLabel="TASK NOT FOUND · 任务不存在"
      >
        {task ? (
          <div className="flex flex-col gap-3">
            <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
              <OverviewStat label="DEVICES" color={NEON.cyan} value={task.targetDeviceSns.length} hint="目标设备数" />
              <OverviewStat label="CELL OK" color={NEON.green} value={okCount} hint="本页开启成功 Cell" />
              <OverviewStat label="CELL FAIL" color={NEON.rose} value={failCount} hint="本页失败 Cell" />
              <OverviewStat
                label="STATUS"
                color={NEON.gold}
                value={TASK_STATUS_META[task.taskStatus].label}
                hint={task.taskResult || task.taskStatus}
              />
            </div>

            <div className="grid grid-cols-1 gap-3 xl:grid-cols-[300px_1fr]">
              <GlassPanel title="TASK META">
                <div className="px-3 pb-2 pt-3">
                  <div className="mb-2 flex flex-wrap gap-1">
                    {mrTypes.map((m) => (
                      <span key={m} className="chip" style={{ color: MR_TYPE_COLOR[m] ?? NEON.blue }}>{m}</span>
                    ))}
                  </div>
                </div>
                <div className="pb-1">
                  <KV label="状态"><StatusBadge status={TASK_STATUS_META[task.taskStatus].token} label={TASK_STATUS_META[task.taskStatus].label} /></KV>
                  <KV label="上报周期">{task.reportPeriod} 分钟</KV>
                  <KV label="统计周期">{task.statisPeriod}</KV>
                  <KV label="创建人">{task.creator}</KV>
                  <KV label="开始时间">{formatTime(task.startTime)}</KV>
                  <KV label="结束时间">{task.endTime ? formatTime(task.endTime) : '无限制'}</KV>
                  <KV label="创建时间">{formatTime(task.createdAt)}</KV>
                </div>
              </GlassPanel>

              <GlassPanel strong title="CELL PROGRESS" meta={`${progressTotal} CELLS`}>
                <div className="p-3">
                  <div className="grid grid-cols-[1.6fr_1.4fr_1fr_1fr_1.4fr] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                    <span>CELL / SN</span>
                    <span>HOST</span>
                    <span>PROGRESS</span>
                    <span>HEALTH</span>
                    <span>HEARTBEAT</span>
                  </div>
                  <div className="mt-1.5 space-y-1.5">
                    <StateGate
                      isLoading={progress.isLoading}
                      isError={progress.isError}
                      error={progress.error}
                      isEmpty={items.length === 0}
                      loadingLabel="SYNCING CELLS…"
                      emptyLabel="NO CELL PROGRESS · 暂无 Cell 下发记录"
                    >
                      {items.map((c) => {
                        const ps = PROGRESS_META[c.progressStatus]
                        const hs = HEALTH_META[c.healthStatus]
                        return (
                          <Row
                            key={c.id}
                            color={ps.token === 'error' ? NEON.rose : NEON.cyan}
                            className="grid-cols-[1.6fr_1.4fr_1fr_1fr_1.4fr]"
                          >
                            <div className="min-w-0">
                              <div className="truncate font-display text-sm font-bold text-cyan-100">{c.smallCellCode}</div>
                              <div className="font-mono text-[10px] text-cyan-300/55">{c.serialNumber}</div>
                            </div>
                            <div className="truncate font-mono text-[11px] text-cyan-100/85">{c.hostName || '—'}</div>
                            <div>
                              <StatusBadge status={ps.token} label={ps.label} />
                              {c.faultCode ? (
                                <div className="mt-0.5 font-mono text-[9px] text-rose-300/70">FC: {c.faultCode}</div>
                              ) : null}
                            </div>
                            <div>
                              <StatusBadge status={hs.token} label={hs.label} />
                              {c.missedHeartbeat > 0 ? (
                                <div className="mt-0.5 font-mono text-[9px] text-amber-300/70">漏跳 {c.missedHeartbeat}</div>
                              ) : null}
                            </div>
                            <div className="font-mono text-[11px] text-cyan-300/75">
                              {c.lastHeartbeat ? formatTime(c.lastHeartbeat) : '从未上报'}
                            </div>
                          </Row>
                        )
                      })}
                    </StateGate>
                  </div>
                  <Pager
                    page={page}
                    totalPages={totalPages}
                    total={progressTotal}
                    pageSize={PAGE_SIZE}
                    onPrev={() => setPage((p) => Math.max(1, p - 1))}
                    onNext={() => setPage((p) => Math.min(totalPages, p + 1))}
                  />
                </div>
              </GlassPanel>
            </div>
          </div>
        ) : null}
      </StateGate>
    </PageShell>
  )
}
