import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, RefreshCcw, Download } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { formatTime } from '@/lib/format'
import {
  useUnifiedFileTransferDevices,
  useUnifiedFileTransferTasks,
} from '@core/hooks/api/useUnifiedFileTransfer'
import type {
  UnifiedFileTransferDeviceItem,
  UnifiedFileTransferDeviceStatus,
} from '@core/types/unifiedFileTransfer'

import { KV, OverviewStat, Pager, Row, StateGate, NEON } from './_shared'

const CATEGORY = 'config_backup'
const PAGE_SIZE = 50

const DEV_STATUS: Record<UnifiedFileTransferDeviceStatus, { token: string; label: string }> = {
  pending: { token: 'unknown', label: '待执行' },
  downloading: { token: 'warning', label: '下载中' },
  uploading: { token: 'warning', label: '上传中' },
  awaiting_tc: { token: 'minor', label: '等待完成' },
  verifying: { token: 'minor', label: '校验中' },
  suspended: { token: 'off', label: '已暂停' },
  ended: { token: 'ok', label: '已完成' },
  failed: { token: 'error', label: '失败' },
}

export default function ConfigRetrievalDetail() {
  const { taskId = '' } = useParams<{ taskId: string }>()
  const navigate = useNavigate()
  const [page, setPage] = useState(1)

  // UFTE 无单任务接口，从类目任务列表中按 id 命中（列表已 100 条覆盖常规规模）
  const tasks = useUnifiedFileTransferTasks({ category: CATEGORY, page: 1, pageSize: 100 })
  const task = tasks.data?.items.find((t) => t.id === taskId)

  const { data, isLoading, isError, error, isFetching, refetch } =
    useUnifiedFileTransferDevices({ category: CATEGORY, page, pageSize: PAGE_SIZE })

  // 设备列表是类目范围，按 taskId 过滤为本任务设备
  const rows: UnifiedFileTransferDeviceItem[] = useMemo(
    () => (data?.items ?? []).filter((d) => d.taskId === taskId),
    [data, taskId]
  )
  const totalPages = Math.max(1, Math.ceil((data?.total ?? 0) / PAGE_SIZE))

  const headerLoading = tasks.isLoading
  const notFound = !headerLoading && !task

  return (
    <PageShell
      code="F06"
      title={task ? `CONFIG PULL · ${task.taskName}` : `CONFIG PULL · ${taskId.slice(0, 8)}`}
      subtitle="UFTE TASK DOSSIER · CONFIG_BACKUP"
      isFetching={isFetching || tasks.isFetching}
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/file/config-retrieval')}>
            返回列表
          </NeonButton>
          <NeonButton
            icon={<RefreshCcw />}
            onClick={() => {
              void refetch()
              void tasks.refetch()
            }}
          >
            REFRESH
          </NeonButton>
        </>
      }
    >
      <StateGate
        isLoading={headerLoading}
        isError={tasks.isError}
        error={tasks.error}
        isEmpty={notFound}
        loadingLabel="LOADING TASK…"
        emptyLabel="TASK NOT FOUND · 任务不存在"
      >
        {task ? (
          <div className="flex flex-col gap-3">
            <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
              <OverviewStat label="TOTAL" color={NEON.cyan} value={task.totalCount} hint="设备总数" />
              <OverviewStat label="SUCCESS" color={NEON.green} value={task.successCount} hint="成功设备" />
              <OverviewStat label="FAILED" color={NEON.rose} value={task.failCount} hint="失败设备" />
              <OverviewStat
                label="MODE"
                color={NEON.violet}
                value={task.executionMode === 'scheduled' ? '定时' : task.executionMode === 'suspended' ? '挂起' : '立即'}
                hint={task.result ?? task.status}
              />
            </div>

            <div className="grid grid-cols-1 gap-3 xl:grid-cols-[300px_1fr]">
              <div className="flex flex-col gap-3">
                <GlassPanel title="TASK PROGRESS" meta={`${task.progress}%`}>
                  <div className="flex items-center justify-center p-4">
                    <RadialGauge value={task.progress} label="完成度" color={NEON.cyan} />
                  </div>
                </GlassPanel>
                <GlassPanel title="META">
                  <div className="py-1">
                    <KV label="任务类型">{task.typeDisplayName}</KV>
                    <KV label="类目">{task.categoryLabel}</KV>
                    <KV label="状态">{task.status}</KV>
                    <KV label="创建人">{task.createUser}</KV>
                    <KV label="创建时间">{formatTime(task.createdAt)}</KV>
                    {task.scheduledAt ? <KV label="计划时间">{formatTime(task.scheduledAt)}</KV> : null}
                    <KV label="操作域">{task.operatorScope}</KV>
                  </div>
                </GlassPanel>
              </div>

              <GlassPanel strong title="DEVICE EXECUTION" meta={`${rows.length} / ${task.totalCount}`}>
                <div className="p-3">
                  <div className="grid grid-cols-[2fr_1fr_1fr_1.4fr_90px] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                    <span>DEVICE / SN</span>
                    <span>STATUS</span>
                    <span>PROGRESS</span>
                    <span>LAST REPORT</span>
                    <span className="text-right">FILE</span>
                  </div>
                  <div className="mt-1.5 space-y-1.5">
                    <StateGate
                      isLoading={isLoading}
                      isError={isError}
                      error={error}
                      isEmpty={rows.length === 0}
                      loadingLabel="SYNCING DEVICES…"
                      emptyLabel="NO DEVICES · 本任务暂无设备执行项"
                    >
                      {rows.map((d) => {
                        const st = DEV_STATUS[d.status]
                        return (
                          <Row
                            key={d.id}
                            color={st.token === 'error' ? NEON.rose : NEON.cyan}
                            className="grid-cols-[2fr_1fr_1fr_1.4fr_90px]"
                          >
                            <div className="min-w-0">
                              <div className="truncate font-display text-sm font-bold text-cyan-100">{d.deviceName || '—'}</div>
                              <div className="font-mono text-[10px] text-cyan-300/55">{d.deviceSn}</div>
                            </div>
                            <div>
                              <StatusBadge status={st.token} label={st.label} />
                            </div>
                            <div className="font-mono text-[11px] text-cyan-300/75">{d.progress}%</div>
                            <div className="font-mono text-[11px] text-cyan-300/75">{formatTime(d.lastReportAt)}</div>
                            <div className="flex justify-end">
                              {d.downloadUrl ? (
                                <a href={d.downloadUrl} target="_blank" rel="noreferrer" className="neon-btn" title={d.targetFile}>
                                  <span className="[&_svg]:size-3.5"><Download /></span>
                                </a>
                              ) : (
                                <span className="font-mono text-[10px] text-cyan-300/35">未就绪</span>
                              )}
                            </div>
                          </Row>
                        )
                      })}
                    </StateGate>
                  </div>
                  <Pager
                    page={page}
                    totalPages={totalPages}
                    total={data?.total ?? 0}
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
