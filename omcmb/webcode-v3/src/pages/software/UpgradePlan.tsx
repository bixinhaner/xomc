import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  RefreshCcw,
  Loader2,
  Rocket,
  Activity,
  ChevronRight,
  Pause,
  Play,
  Square,
  RotateCcw,
  Trash2,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { formatTime } from '@/lib/format'
import {
  useUpgradeTasks,
  useSuspendTask,
  useResumeTask,
  useTerminateTask,
  useRetryTask,
  useDeleteTask,
  useAdvanceCanary,
  usePauseCanary,
  useResumeCanary,
  useAbortCanary,
} from '@core/hooks/api/useSoftware'
import { useProductList } from '@core/hooks/api/useProducts'
import type { UpgradeTaskInfo, TaskStatusType } from '@core/mock/data/software'

import { NEON, TASK_STATUS, TASK_TYPE, RESULT_COLOR, StatCard, Syncing, ErrorBlock, EmptyBlock, Pager } from './_shared'

// ---------------------------------------------------------------------------
// 升级计划 · software/upgrade-plan
// 软件升级任务编排板（taskType=1）。行 → 详情 software/upgrade-plan/:id
// ---------------------------------------------------------------------------

const STATUS_FILTERS = ['', 'pending', 'in_progress', 'suspended', 'ended'] as const
const UPGRADE_TASK_TYPE = 1

function progressOf(t: UpgradeTaskInfo): number {
  if (!t.totalCount) return 0
  return Math.round(((t.successCount + t.failCount) / t.totalCount) * 100)
}

export default function UpgradePlan() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const pageSize = 12
  const [statusFilter, setStatusFilter] = useState<TaskStatusType | ''>('')
  // #638：按“产品名”过滤升级任务。''=全部；productId 使用产品中心 products.id。
  const [productId, setProductId] = useState('')
  const [keyword, setKeyword] = useState('')

  const { data: productsData } = useProductList()

  const params = useMemo(
    () => ({
      page,
      pageSize,
      taskType: UPGRADE_TASK_TYPE,
      ...(statusFilter ? { status: statusFilter } : {}),
      ...(productId ? { productId } : {}),
    }),
    [page, statusFilter, productId]
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
    const devTotal = allTasks.reduce((a, t) => a + t.totalCount, 0)
    const devSuccess = allTasks.reduce((a, t) => a + t.successCount, 0)
    const devFail = allTasks.reduce((a, t) => a + t.failCount, 0)
    const rate = devTotal > 0 ? (devSuccess / devTotal) * 100 : 0
    return { active, devSuccess, devFail, rate }
  }, [allTasks])

  return (
    <PageShell
      code="F06"
      title="UPGRADE PLAN · 升级计划"
      subtitle="OTA UPGRADE ORCHESTRATION · TASK TYPE 1"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="任务名 / 固件 / 制式 / 操作员"
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
      {/* 概览 */}
      <div className="mb-3 grid grid-cols-12 gap-3">
        <div className="col-span-12 grid grid-cols-2 gap-3 lg:col-span-9 lg:grid-cols-4">
          <StatCard label="任务总数 · TASKS" value={total.toLocaleString()} color={NEON.cyan} icon={<Rocket className="size-4" />} />
          <StatCard label="执行中 · ACTIVE" value={overview.active.toLocaleString()} color={overview.active > 0 ? NEON.amber : NEON.dim} icon={<Activity className="size-4" />} />
          <StatCard label="升级成功 · OK" value={overview.devSuccess.toLocaleString()} color={NEON.green} icon={<Activity className="size-4" />} />
          <StatCard label="升级失败 · FAIL" value={overview.devFail.toLocaleString()} color={overview.devFail > 0 ? NEON.rose : NEON.dim} icon={<Activity className="size-4" />} />
        </div>
        <GlassPanel title="GLOBAL SUCCESS" meta="本页聚合" className="col-span-12 lg:col-span-3">
          <div className="flex items-center justify-center py-3">
            <RadialGauge
              value={overview.rate}
              label="SUCCESS"
              size={104}
              color={overview.rate >= 90 ? NEON.green : overview.rate >= 60 ? NEON.amber : NEON.rose}
            />
          </div>
        </GlassPanel>
      </div>

      {/* 筛选 */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        {/* #638：产品名过滤（v1/v2 对齐）。原生 select 保持 HUD 边框风格与该页一致。 */}
        <select
          className="neon-input w-44"
          value={productId}
          onChange={(e) => { setProductId(e.target.value); setPage(1) }}
        >
          <option value="">全部产品 · ALL PRODUCTS</option>
          {(productsData?.items ?? []).map((p) => (
            <option key={p.id} value={p.id}>{p.name}</option>
          ))}
        </select>
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

      {/* 任务板 */}
      {isLoading ? (
        <Syncing label="LOADING UPGRADE QUEUE…" />
      ) : isError ? (
        <ErrorBlock msg={error instanceof Error ? error.message : '未知错误'} />
      ) : tasks.length === 0 ? (
        <EmptyBlock label="NO UPGRADE TASKS · 无升级任务" />
      ) : (
        <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
          {tasks.map((t) => (
            <TaskCard key={t.id} t={t} onInspect={() => navigate(`/software/upgrade-plan/${t.id}`)} />
          ))}
        </div>
      )}

      <Pager page={page} totalPages={totalPages} pageSize={pageSize} total={total} onPage={setPage} />
    </PageShell>
  )
}

function TaskCard({ t, onInspect }: { t: UpgradeTaskInfo; onInspect: () => void }) {
  const suspend = useSuspendTask()
  const resume = useResumeTask()
  const terminate = useTerminateTask()
  const retry = useRetryTask()
  const del = useDeleteTask()
  const advanceCanary = useAdvanceCanary()
  const pauseCanary = usePauseCanary()
  const resumeCanary = useResumeCanary()
  const abortCanary = useAbortCanary()

  const sc = TASK_STATUS[t.status] ?? { label: t.status, color: NEON.dim }
  const tc = TASK_TYPE[t.taskType] ?? { label: `TYPE ${t.taskType}`, color: NEON.cyan }
  const pct = progressOf(t)
  const isCanary = t.strategy === 'canary'
  const stageStatus = t.stageStatus ?? 'pending'
  const canaryRunning = isCanary && stageStatus === 'running'
  const canaryPaused = isCanary && stageStatus === 'paused'
  const canaryDone = isCanary && (stageStatus === 'completed' || stageStatus === 'aborted')
  const totalStages = t.canaryStages?.length ?? 0
  const curStage = t.currentStage ?? 0
  const curPercent = t.canaryStages?.[Math.max(0, curStage - 1)]?.percent

  const ended = t.status === 'ended'
  const barColor =
    t.result === 'failed' || t.result === 'terminated'
      ? NEON.rose
      : t.result === 'partial'
        ? NEON.amber
        : ended
          ? NEON.green
          : NEON.cyan

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
              {t.productClass || '—'} · FW {t.fileName || '—'} · BY {t.createUser || '—'}
            </div>
          </div>
          <div className="flex shrink-0 flex-col items-end gap-1">
            <span className="chip" style={{ color: sc.color }}>
              <span className="size-1.5 rounded-full bg-current shadow-[0_0_6px_currentColor]" />
              {sc.label}
            </span>
            <span className="chip" style={{ color: tc.color }}>{tc.label}</span>
          </div>
        </div>

        {isCanary && totalStages > 0 ? (
          <div className="mt-2 flex items-center gap-2">
            <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-violet-300/70">CANARY</span>
            <div className="flex flex-1 items-center gap-1">
              {t.canaryStages?.map((s, i) => {
                const reached = i < curStage
                const active = i === curStage - 1
                return (
                  <div
                    key={i}
                    className="h-1.5 flex-1 rounded-full"
                    style={{
                      background: reached ? NEON.violet : 'rgba(168,85,247,0.15)',
                      boxShadow: active ? `0 0 6px ${NEON.violet}` : undefined,
                    }}
                    title={`阶段 ${i + 1} · ${s.percent}% · 阈值 ${s.failureThreshold}%`}
                  />
                )
              })}
            </div>
            <span className="font-mono text-[10px] text-violet-200/80">
              {curStage}/{totalStages}{curPercent != null ? ` · ${curPercent}%` : ''} · {stageStatus}
            </span>
          </div>
        ) : null}

        <div className="mt-2.5">
          <div className="mb-1 flex items-center justify-between font-mono text-[10px] text-cyan-300/60">
            <span>PROGRESS <span className="text-cyan-100/85">{t.successCount + t.failCount}/{t.totalCount}</span></span>
            <span style={{ color: barColor }}>{pct}%</span>
          </div>
          <div className="h-2 overflow-hidden rounded-full bg-cyan-500/10">
            <div className="h-full rounded-full transition-all" style={{ width: `${pct}%`, background: barColor, boxShadow: `0 0 8px ${barColor}` }} />
          </div>
          <div className="mt-1.5 flex items-center gap-3 font-mono text-[10px]">
            <span style={{ color: NEON.green }}>✓ {t.successCount} 成功</span>
            <span style={{ color: t.failCount > 0 ? NEON.rose : NEON.dim }}>✗ {t.failCount} 失败</span>
            <span className="text-cyan-300/55">并发 {t.maxConcurrent}</span>
            {t.result ? <span style={{ color: RESULT_COLOR[t.result] ?? NEON.dim }}>结果 {t.result}</span> : null}
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

          {canaryRunning ? (
            <>
              <NeonButton className="!py-1 !px-2" icon={<Play />} disabled={advanceCanary.isPending} onClick={() => confirm('推进到下一灰度阶段？', () => advanceCanary.mutate(t.id))}>推进阶段</NeonButton>
              <NeonButton className="!py-1 !px-2" icon={<Pause />} disabled={pauseCanary.isPending} onClick={() => pauseCanary.mutate(t.id)}>暂停灰度</NeonButton>
            </>
          ) : null}
          {canaryPaused ? (
            <NeonButton className="!py-1 !px-2" icon={<Play />} disabled={resumeCanary.isPending} onClick={() => resumeCanary.mutate(t.id)}>恢复灰度</NeonButton>
          ) : null}
          {isCanary && !canaryDone ? (
            <NeonButton tone="danger" className="!py-1 !px-2" icon={<Square />} disabled={abortCanary.isPending} onClick={() => confirm('中止灰度发布？已升级设备不回退。', () => abortCanary.mutate(t.id))}>中止灰度</NeonButton>
          ) : null}

          {!isCanary && t.status === 'in_progress' ? (
            <NeonButton className="!py-1 !px-2" icon={<Pause />} disabled={suspend.isPending} onClick={() => suspend.mutate(t.id)}>暂停</NeonButton>
          ) : null}
          {!isCanary && t.status === 'suspended' ? (
            <NeonButton className="!py-1 !px-2" icon={<Play />} disabled={resume.isPending} onClick={() => resume.mutate(t.id)}>恢复</NeonButton>
          ) : null}
          {!ended ? (
            <NeonButton tone="danger" className="!py-1 !px-2" icon={<Square />} disabled={terminate.isPending} onClick={() => confirm('终止整个升级任务？', () => terminate.mutate(t.id))}>终止</NeonButton>
          ) : null}
          {ended && t.failCount > 0 ? (
            <NeonButton className="!py-1 !px-2" icon={<RotateCcw />} disabled={retry.isPending} onClick={() => confirm('重试失败设备？', () => retry.mutate(t.id))}>重试失败</NeonButton>
          ) : null}
          {ended ? (
            <NeonButton tone="danger" className="!py-1 !px-2" icon={<Trash2 />} disabled={del.isPending} onClick={() => confirm('删除该任务记录？', () => del.mutate(t.id))}>删除</NeonButton>
          ) : null}
        </div>
      </div>
    </div>
  )
}
