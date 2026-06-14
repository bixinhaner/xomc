import { useMemo, useState } from 'react'
import { CalendarClock, Loader2, Plus, RefreshCcw, X } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { cn } from '@/lib/utils'
import { usePerformanceTasks, useCreatePerformanceTask } from '@core/hooks/api/usePerformance'

/**
 * F03 · 采集任务配置（performance/task-config）
 * 走真实 usePerformanceTasks 列表 + useCreatePerformanceTask 新建。
 * 列表三态完整；HUD 弹层创建采集任务。
 */

const PAGE_SIZE = 20

const GRAN_OPTS: { value: '15min' | '30min' | '1h' | '1d'; label: string }[] = [
  { value: '15min', label: '15分钟' },
  { value: '30min', label: '30分钟' },
  { value: '1h', label: '小时' },
  { value: '1d', label: '天' },
]

// 后端 status 字符串 → StatusBadge token + 中文。
const STATUS_TOKEN: Record<string, string> = {
  pending: 'minor',
  running: 'ok',
  success: 'ok',
  succeeded: 'ok',
  failed: 'error',
  cancelled: 'off',
  canceled: 'off',
  active: 'ok',
  paused: 'off',
  error: 'error',
}
const STATUS_LABEL: Record<string, string> = {
  pending: '排队',
  running: '运行中',
  success: '成功',
  succeeded: '成功',
  failed: '失败',
  cancelled: '已取消',
  canceled: '已取消',
  active: '运行中',
  paused: '暂停',
  error: '错误',
}

export default function PerformanceTaskConfig() {
  const [page, setPage] = useState(1)
  const [modalOpen, setModalOpen] = useState(false)
  const [taskName, setTaskName] = useState('')
  const [granularity, setGranularity] = useState<'15min' | '30min' | '1h' | '1d'>('15min')

  const { data, isLoading, isError, error, isFetching, refetch } = usePerformanceTasks({ page, pageSize: PAGE_SIZE })
  const createMut = useCreatePerformanceTask()

  const rows = data?.items ?? []
  const total = data?.total ?? rows.length
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const runningCount = useMemo(
    () => rows.filter((r) => r.status === 'running' || r.status === 'active').length,
    [rows],
  )

  const canCreate = taskName.trim().length > 0

  const submit = () => {
    if (!canCreate) return
    createMut.mutate(
      {
        taskName: taskName.trim(),
        taskType: 'extraction',
        deviceSns: [],
        kpiCodes: [],
        granularity,
        timeRange: ['', ''],
        creator: 'admin',
      },
      {
        onSuccess: () => {
          setModalOpen(false)
          setTaskName('')
          void refetch()
        },
      },
    )
  }

  return (
    <PageShell
      code="F03"
      title="TASK CONFIG · 采集任务"
      subtitle="PM COLLECTION SCHEDULE · EXTRACTION TASK"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <NeonButton icon={<Plus />} onClick={() => setModalOpen(true)}>
            新建任务
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-3">
        <Stat label="TASKS · 任务总数" value={String(total)} color="#00f0ff" />
        <Stat label="RUNNING · 运行中" value={String(runningCount)} color="#00ff88" />
        <Stat label="PAGE · 本页" value={String(rows.length)} color="#a855f7" />
      </div>

      <GlassPanel strong title="COLLECTION TASKS · 采集任务" meta={`${total} TASKS`}>
        <div className="p-3">
          <div className="grid grid-cols-[2fr_1fr_90px_1fr_1.2fr_90px_1fr] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            <span>NAME · 名称</span>
            <span>TYPE · 类型</span>
            <span>GRAN · 粒度</span>
            <span>DEVICES · 设备</span>
            <span>CREATED · 创建</span>
            <span>STATUS</span>
            <span>CREATOR</span>
          </div>

          <div className="mt-1.5 space-y-1">
            {isLoading ? (
              <Loading text="SYNCING TASKS…" />
            ) : isError ? (
              <ErrorBox text={`LOAD FAILED · ${error instanceof Error ? error.message : '未知错误'}`} />
            ) : rows.length === 0 ? (
              <EmptyBox text="NO TASK · 暂无采集任务" />
            ) : (
              rows.map((r) => (
                <div
                  key={r.id}
                  className="grid grid-cols-[2fr_1fr_90px_1fr_1.2fr_90px_1fr] items-center gap-3 rounded-sm px-3 py-2 transition-colors hover:bg-cyan-500/5"
                >
                  <span className="truncate font-display text-sm font-bold text-cyan-100" title={r.taskName}>
                    {r.taskName}
                  </span>
                  <span className="truncate font-mono text-[11px] text-cyan-300/70">{r.taskType || '—'}</span>
                  <span className="chip w-fit text-cyan-300/80">{r.granularity || '—'}</span>
                  <span className="font-mono text-[11px] text-cyan-300/70">
                    {r.deviceSns.length > 0 ? `${r.deviceSns.length} 台` : '全部'}
                  </span>
                  <span className="font-mono text-[11px] text-cyan-300/65">
                    {r.createdAt ? formatTime(r.createdAt) : '—'}
                  </span>
                  <StatusBadge
                    status={STATUS_TOKEN[r.status] ?? 'unknown'}
                    label={STATUS_LABEL[r.status] ?? r.status}
                    className="w-fit"
                  />
                  <span className="truncate font-mono text-[11px] text-cyan-300/70">{r.creator || '—'}</span>
                </div>
              ))
            )}
          </div>

          <div className="mt-3 flex items-center justify-between">
            <span className="font-mono text-[11px] text-cyan-300/55">
              PAGE {page} / {totalPages} · TOTAL {total}
            </span>
            <div className="flex gap-2">
              <NeonButton onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
                ◂ PREV
              </NeonButton>
              <NeonButton onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages}>
                NEXT ▸
              </NeonButton>
            </div>
          </div>
        </div>
      </GlassPanel>

      {modalOpen ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm">
          <GlassPanel
            strong
            className="w-full max-w-md"
            title="NEW TASK · 新建采集任务"
            meta={<CalendarClock className="size-3.5 text-cyan-300" />}
          >
            <button
              type="button"
              onClick={() => setModalOpen(false)}
              className="absolute right-3 top-2.5 text-cyan-300/55 hover:text-cyan-100"
              aria-label="close"
            >
              <X className="size-4" />
            </button>
            <div className="space-y-3 p-4">
              <div>
                <div className="mb-1 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                  任务名称 · NAME
                </div>
                <input
                  className="neon-input w-full"
                  value={taskName}
                  onChange={(e) => setTaskName(e.target.value)}
                  placeholder="如：全网 15 分钟定时采集"
                />
              </div>
              <div>
                <div className="mb-1 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                  采集粒度 · GRANULARITY
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {GRAN_OPTS.map((g) => (
                    <button
                      key={g.value}
                      type="button"
                      onClick={() => setGranularity(g.value)}
                      className={cn(
                        'chip transition-all',
                        granularity === g.value
                          ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                          : 'text-cyan-300/55 opacity-70 hover:opacity-100',
                      )}
                    >
                      {g.label}
                    </button>
                  ))}
                </div>
              </div>
              <div className="flex justify-end gap-2 pt-1">
                <NeonButton onClick={() => setModalOpen(false)}>取消</NeonButton>
                <NeonButton
                  icon={createMut.isPending ? <Loader2 className="animate-spin" /> : <Plus />}
                  disabled={!canCreate || createMut.isPending}
                  onClick={submit}
                >
                  创建
                </NeonButton>
              </div>
            </div>
          </GlassPanel>
        </div>
      ) : null}
    </PageShell>
  )
}

/* ───────── 复用小件 ───────── */

function Stat({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <div className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3" style={{ borderLeftColor: color }}>
      <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/60">{label}</div>
      <div className="font-display text-2xl font-bold leading-tight" style={{ color, textShadow: `0 0 8px ${color}` }}>
        {value}
      </div>
    </div>
  )
}

function Loading({ text }: { text: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
      <Loader2 className="size-5 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">{text}</span>
    </div>
  )
}

function ErrorBox({ text }: { text: string }) {
  return (
    <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">{text}</div>
  )
}

function EmptyBox({ text }: { text: string }) {
  return (
    <div className="flex items-center justify-center py-12 font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/40">
      {text}
    </div>
  )
}
