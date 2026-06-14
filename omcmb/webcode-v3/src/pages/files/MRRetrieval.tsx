import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  RefreshCcw,
  ChevronRight,
  SignalHigh,
  Activity,
  Boxes,
  Power,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useMRTasks, useStopMRTask } from '@core/hooks/api/useMrTasks'
import type { MRTask, MRTaskStatus } from '@core/types/mrTask'

import { ChipFilter, OverviewStat, Pager, Row, StateGate, NEON } from './_shared'

// 测量报告获取 = MR 测量任务（MRO/MRS/MRE 上报），真实接口 GET /mr/tasks
const PAGE_SIZE = 20

const STATUS_META: Record<MRTaskStatus, { token: string; label: string }> = {
  waitting: { token: 'unknown', label: '待开启' },
  on: { token: 'ok', label: '上报中' },
  off: { token: 'off', label: '已结束' },
  suspend: { token: 'warning', label: '已暂停' },
  termination: { token: 'major', label: '已终止' },
}

const MR_TYPE_COLOR: Record<string, string> = {
  MRO: NEON.cyan,
  MRE: NEON.green,
  MRS: NEON.amber,
}

type StatusFilter = '' | MRTaskStatus

export default function MRRetrieval() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState<StatusFilter>('')
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(status ? { status } : {}),
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, status, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useMRTasks(params)
  const stopTask = useStopMRTask()

  const rows: MRTask[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const onCount = rows.filter((t) => t.taskStatus === 'on').length
  const deviceCount = rows.reduce((s, t) => s + t.targetDeviceSns.length, 0)

  return (
    <PageShell
      code="F05"
      title="MR PULL · 测量报告获取"
      subtitle="MRO / MRS / MRE TASK INTAKE · 10s REFRESH"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-56 pl-9"
              placeholder="任务名"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <ChipFilter<StatusFilter>
            value={status}
            onChange={(v) => {
              setStatus(v)
              setPage(1)
            }}
            options={[
              { value: '', label: 'ALL' },
              { value: 'on', label: '上报中', color: NEON.green },
              { value: 'waitting', label: '待开启' },
              { value: 'off', label: '已结束' },
            ]}
          />
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 lg:grid-cols-4">
        <OverviewStat icon={<SignalHigh className="size-4" />} label="TASKS" color={NEON.cyan} value={total.toLocaleString()} hint="测量任务总数" />
        <OverviewStat icon={<Activity className="size-4" />} label="REPORTING" color={NEON.green} value={onCount} hint="本页上报中" />
        <OverviewStat icon={<Boxes className="size-4" />} label="DEVICES" color={NEON.violet} value={deviceCount} hint="本页目标设备数" />
        <OverviewStat label="PAGE" color={NEON.gold} value={`${page}/${totalPages}`} hint={`${PAGE_SIZE}/页`} />
      </div>

      <GlassPanel strong title="MR TASK MANIFEST" meta={`${total} TASKS`}>
        <div className="p-3">
          <div className="grid grid-cols-[2fr_1.4fr_1fr_1fr_1.2fr_120px] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            <span>TASK</span>
            <span>MR TYPE</span>
            <span>STATUS</span>
            <span>DEVICES</span>
            <span>START</span>
            <span className="text-right">ACTIONS</span>
          </div>
          <div className="mt-1.5 space-y-1.5">
            <StateGate
              isLoading={isLoading}
              isError={isError}
              error={error}
              isEmpty={rows.length === 0}
              loadingLabel="SYNCING TASKS…"
              emptyLabel="NO MR TASKS · 暂无测量任务"
            >
              {rows.map((t) => {
                const st = STATUS_META[t.taskStatus]
                const mrTypes = t.mrType.split(',').map((s) => s.trim()).filter(Boolean)
                return (
                  <Row
                    key={t.taskId}
                    color={t.taskStatus === 'on' ? NEON.green : NEON.cyan}
                    className="grid-cols-[2fr_1.4fr_1fr_1fr_1.2fr_120px]"
                  >
                    <div className="min-w-0">
                      <div className="truncate font-display text-sm font-bold text-cyan-100" title={t.taskName}>{t.taskName}</div>
                      <div className="font-mono text-[10px] text-cyan-300/45">{t.creator} · 周期 {t.reportPeriod}min</div>
                    </div>
                    <div className="flex flex-wrap gap-1">
                      {mrTypes.map((m) => (
                        <span key={m} className="chip" style={{ color: MR_TYPE_COLOR[m] ?? NEON.blue }}>{m}</span>
                      ))}
                    </div>
                    <div>
                      <StatusBadge status={st.token} label={st.label} />
                    </div>
                    <div className="font-mono text-[12px] text-cyan-100/85">{t.targetDeviceSns.length}</div>
                    <div className="font-mono text-[11px] text-cyan-300/75">{formatTime(t.startTime)}</div>
                    <div className="flex justify-end gap-1.5">
                      {t.taskStatus === 'on' || t.taskStatus === 'waitting' ? (
                        <NeonButton
                          tone="danger"
                          icon={<Power />}
                          disabled={stopTask.isPending}
                          onClick={() => stopTask.mutate(t.taskId)}
                        >
                          停止
                        </NeonButton>
                      ) : null}
                      <NeonButton icon={<ChevronRight />} onClick={() => navigate(`/file/mr-retrieval/${t.taskId}`)}>
                        详情
                      </NeonButton>
                    </div>
                  </Row>
                )
              })}
            </StateGate>
          </div>
          <Pager
            page={page}
            totalPages={totalPages}
            total={total}
            pageSize={PAGE_SIZE}
            onPrev={() => setPage((p) => Math.max(1, p - 1))}
            onNext={() => setPage((p) => Math.min(totalPages, p + 1))}
          />
        </div>
      </GlassPanel>
    </PageShell>
  )
}
