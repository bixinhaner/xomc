import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  RefreshCcw,
  Loader2,
  Activity,
  CheckCircle2,
  XCircle,
  Gauge,
  ChevronRight,
  AlertTriangle,
  Cpu,
  Clock,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { formatTime } from '@/lib/format'
import { cn } from '@/lib/utils'
import { usePerformanceTasks } from '@core/hooks/api/usePerformance'

// ---------------------------------------------------------------------------
// /reports/poll-stats · 轮询统计
//   对齐 v1 webcode/report/PollStatistics：采集任务轮询的成功率 / 进度 / 状态。
//   v1 旧页面是纯本地 mock；本皮肤改用真实 @core/usePerformanceTasks
//   （PM 采集/提取任务 = /pm/tasks），按状态渲染轮询任务流 + 概览仪表。
//   列表 → 详情：选中任务展开任务详情面板（设备/KPI/进度/时间窗）。
// ---------------------------------------------------------------------------

// PM 任务的 status 在 mock 与真实后端取值不完全一致，统一归一到展示状态。
type DisplayStatus = 'running' | 'success' | 'failed' | 'pending' | 'cancelled'

const STATUS_NORMALIZE: Record<string, DisplayStatus> = {
  running: 'running',
  pending: 'pending',
  success: 'success',
  completed: 'success',
  done: 'success',
  failed: 'failed',
  error: 'failed',
  cancelled: 'cancelled',
  canceled: 'cancelled',
  paused: 'pending',
}

const STATUS_LABEL: Record<DisplayStatus, string> = {
  running: '运行中',
  success: '成功',
  failed: '失败',
  pending: '等待',
  cancelled: '已取消',
}

const STATUS_BADGE: Record<DisplayStatus, string> = {
  running: 'active',
  success: 'ok',
  failed: 'critical',
  pending: 'warning',
  cancelled: 'offline',
}

const STATUS_COLOR: Record<DisplayStatus, string> = {
  running: '#00f0ff',
  success: '#00ff88',
  failed: '#ff2d6f',
  pending: '#ffaa00',
  cancelled: '#525a78',
}

const TASK_TYPE_LABEL: Record<string, string> = {
  extraction: '采集',
  report: '报表',
  'threshold-check': '门限核查',
}

const STATUS_FILTERS: { label: string; value: DisplayStatus | '' }[] = [
  { label: 'ALL', value: '' },
  { label: '运行中', value: 'running' },
  { label: '成功', value: 'success' },
  { label: '失败', value: 'failed' },
  { label: '等待', value: 'pending' },
]

const PAGE_SIZE = 50

function normStatus(s: string): DisplayStatus {
  return STATUS_NORMALIZE[(s || '').toLowerCase()] ?? 'pending'
}

export default function PollStatisticsReportPage() {
  const navigate = useNavigate()
  const [statusFilter, setStatusFilter] = useState<DisplayStatus | ''>('')
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const { data, isLoading, isError, error, isFetching, refetch } = usePerformanceTasks({ page: 1, pageSize: PAGE_SIZE })
  const allTasks = data?.items ?? []

  const tasks = useMemo(() => {
    if (!statusFilter) return allTasks
    return allTasks.filter((t) => normStatus(t.status) === statusFilter)
  }, [allTasks, statusFilter])

  // 概览统计（真实任务派生）
  const stats = useMemo(() => {
    const total = allTasks.length
    const running = allTasks.filter((t) => normStatus(t.status) === 'running').length
    const success = allTasks.filter((t) => normStatus(t.status) === 'success').length
    const failed = allTasks.filter((t) => normStatus(t.status) === 'failed').length
    const successRate = total > 0 ? Math.round((success / total) * 100) : 0
    return { total, running, success, failed, successRate }
  }, [allTasks])

  const selected = selectedId ? allTasks.find((t) => t.id === selectedId) ?? null : null

  return (
    <PageShell
      code="F06"
      title="POLL STATISTICS · 轮询统计"
      subtitle="PM COLLECTION TASKS · SUCCESS RATE & PROGRESS"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <button type="button" onClick={() => navigate('/reports')} className="chip text-cyan-300/70 hover:opacity-100">
            ← 简报中心
          </button>
          {STATUS_FILTERS.map((f) => (
            <button
              key={f.value || 'all'}
              type="button"
              onClick={() => setStatusFilter(f.value)}
              className={cn(
                'chip transition-all',
                statusFilter === f.value ? 'text-cyan-200 shadow-[0_0_10px_currentColor]' : 'text-cyan-300/55 opacity-70 hover:opacity-100',
              )}
            >
              {f.label}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => void refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-5">
        <Stat label="TASKS · 任务总数" value={stats.total} color="#00f0ff" icon={<Activity className="size-4" />} />
        <Stat label="RUNNING · 运行中" value={stats.running} color="#00f0ff" icon={<Loader2 className="size-4" />} />
        <Stat label="SUCCESS · 成功" value={stats.success} color="#00ff88" icon={<CheckCircle2 className="size-4" />} />
        <Stat label="FAILED · 失败" value={stats.failed} color="#ff2d6f" icon={<XCircle className="size-4" />} />
        <div className="glass relative flex items-center justify-center overflow-hidden rounded-sm border-l-2 px-3 py-2" style={{ borderLeftColor: '#a855f7' }}>
          <div className="scanline" />
          <div className="relative flex items-center gap-3">
            <RadialGauge value={stats.successRate} label="成功率" size={64} color="#a855f7" unit="%" />
            <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/60">
              SUCCESS
              <br />
              RATE
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-12 gap-3">
        {/* 左：任务流 */}
        <div className={cn('col-span-12', selected ? 'lg:col-span-7' : '')}>
          <GlassPanel title="POLL TASKS · 采集任务流" meta={`${tasks.length} / ${stats.total}`}>
            <div className="grid grid-cols-[1.4fr_0.7fr_1fr_0.7fr_auto] gap-2 border-b border-cyan-500/15 px-3.5 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
              <span>任务 · TASK</span>
              <span>类型</span>
              <span>进度 / 设备</span>
              <span>状态</span>
              <span className="text-right">详情</span>
            </div>

            <div className="max-h-[calc(100vh-340px)] min-h-[240px] overflow-auto">
              {isLoading ? (
                <Centered>
                  <Loader2 className="size-4 animate-spin" />
                  <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING TASKS…</span>
                </Centered>
              ) : isError ? (
                <div className="m-3 border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
                  TASK FEED UNAVAILABLE · {error instanceof Error ? error.message : '加载失败'}
                </div>
              ) : tasks.length === 0 ? (
                <Empty text={statusFilter ? '该状态下无任务' : '暂无采集任务'} />
              ) : (
                tasks.map((t) => {
                  const ds = normStatus(t.status)
                  const color = STATUS_COLOR[ds]
                  const active = selectedId === t.id
                  const progress = Math.max(0, Math.min(100, Number(t.progress) || 0))
                  return (
                    <div
                      key={t.id}
                      role="button"
                      tabIndex={0}
                      onClick={() => setSelectedId(active ? null : t.id)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') setSelectedId(active ? null : t.id)
                      }}
                      className={cn(
                        'fleet-row grid cursor-pointer grid-cols-[1.4fr_0.7fr_1fr_0.7fr_auto] items-center gap-2 px-3.5 py-2.5',
                        active && 'bg-cyan-500/10',
                      )}
                      style={{ ['--row-color' as never]: color }}
                    >
                      <div className="min-w-0">
                        <div className="truncate font-display text-sm font-bold text-cyan-100">{t.taskName}</div>
                        <div className="truncate font-mono text-[10px] text-cyan-300/45">
                          {t.creator || '—'} · {t.createdAt ? formatTime(t.createdAt) : '—'}
                        </div>
                      </div>
                      <span className="chip w-fit text-cyan-300/75">{TASK_TYPE_LABEL[t.taskType] ?? t.taskType}</span>
                      <div className="min-w-0">
                        <div className="h-1.5 w-full overflow-hidden rounded-full bg-cyan-500/10">
                          <div className="h-full rounded-full" style={{ width: `${progress}%`, background: color, boxShadow: `0 0 6px ${color}` }} />
                        </div>
                        <div className="mt-1 font-mono text-[9px] text-cyan-300/50">
                          {progress}% · {t.deviceSns.length} 设备 / {t.kpiCodes.length} KPI
                        </div>
                      </div>
                      <StatusBadge status={STATUS_BADGE[ds]} label={STATUS_LABEL[ds]} className="w-fit" />
                      <ChevronRight className={cn('size-4 justify-self-end text-cyan-300/40 transition-transform', active && 'rotate-90 text-cyan-200')} />
                    </div>
                  )
                })
              )}
            </div>
          </GlassPanel>
        </div>

        {/* 右：任务详情 */}
        {selected ? (
          <div className="col-span-12 lg:col-span-5">
            <GlassPanel title="TASK DETAIL · 任务详情" meta={selected.id.slice(0, 8)}>
              <div className="space-y-3 p-3.5">
                <div>
                  <div className="font-display text-base font-bold text-cyan-100">{selected.taskName}</div>
                  <div className="mt-1 flex flex-wrap items-center gap-1.5">
                    <span className="chip text-cyan-300/75">{TASK_TYPE_LABEL[selected.taskType] ?? selected.taskType}</span>
                    <StatusBadge status={STATUS_BADGE[normStatus(selected.status)]} label={STATUS_LABEL[normStatus(selected.status)]} />
                    <span className="chip text-cyan-300/70">{selected.granularity}</span>
                  </div>
                </div>

                <div className="flex items-center justify-center py-1">
                  <RadialGauge
                    value={Math.max(0, Math.min(100, Number(selected.progress) || 0))}
                    label="进度"
                    size={120}
                    color={STATUS_COLOR[normStatus(selected.status)]}
                    unit="%"
                  />
                </div>

                <div className="grid grid-cols-2 gap-2">
                  <DetailTile icon={<Cpu className="size-3.5" />} label="设备数" value={String(selected.deviceSns.length)} />
                  <DetailTile icon={<Gauge className="size-3.5" />} label="KPI 数" value={String(selected.kpiCodes.length)} />
                  <DetailTile icon={<Clock className="size-3.5" />} label="创建" value={selected.createdAt ? formatTime(selected.createdAt) : '—'} />
                  <DetailTile icon={<Clock className="size-3.5" />} label="更新" value={selected.updatedAt ? formatTime(selected.updatedAt) : '—'} />
                </div>

                {selected.deviceSns.length > 0 ? (
                  <div>
                    <div className="mb-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                      目标设备 · DEVICES
                    </div>
                    <div className="flex max-h-32 flex-wrap gap-1.5 overflow-auto">
                      {selected.deviceSns.map((s) => (
                        <button
                          key={s}
                          type="button"
                          onClick={() => navigate(`/reports/station/${encodeURIComponent(s)}`)}
                          className="chip text-cyan-200 hover:opacity-100"
                          title={`钻取 ${s} KPI 趋势`}
                        >
                          {s}
                        </button>
                      ))}
                    </div>
                  </div>
                ) : null}

                {selected.kpiCodes.length > 0 ? (
                  <div>
                    <div className="mb-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                      采集 KPI · METRICS
                    </div>
                    <div className="flex max-h-32 flex-wrap gap-1.5 overflow-auto">
                      {selected.kpiCodes.map((c) => (
                        <span key={c} className="chip text-cyan-300/70">
                          {c}
                        </span>
                      ))}
                    </div>
                  </div>
                ) : null}
              </div>
            </GlassPanel>
          </div>
        ) : null}
      </div>
    </PageShell>
  )
}

function Stat({
  label,
  value,
  color,
  icon,
}: {
  label: string
  value: number | string
  color: string
  icon: React.ReactNode
}) {
  return (
    <div className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3" style={{ borderLeftColor: color }}>
      <div className="scanline" />
      <div className="relative flex items-center justify-between">
        <div>
          <div className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/65">
            <span style={{ color }}>{icon}</span>
            {label}
          </div>
          <div className="font-display text-3xl font-bold leading-tight text-glow" style={{ color }}>
            {value}
          </div>
        </div>
      </div>
    </div>
  )
}

function DetailTile({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="rounded-sm border border-cyan-500/12 bg-[#070b16]/55 px-3 py-2">
      <div className="flex items-center gap-1.5 font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/50">
        <span className="text-cyan-300/60">{icon}</span>
        {label}
      </div>
      <div className="mt-0.5 truncate font-display text-sm font-bold text-cyan-100">{value}</div>
    </div>
  )
}

function Centered({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">{children}</div>
}

function Empty({ text }: { text: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16">
      <AlertTriangle className="size-9 text-cyan-300/40" />
      <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">{text}</div>
    </div>
  )
}
