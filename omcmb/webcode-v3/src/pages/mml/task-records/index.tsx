import { useMemo, useState } from 'react'
import {
  Search,
  RefreshCcw,
  Loader2,
  Inbox,
  XCircle,
  CheckCircle2,
  X,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { Sparkline } from '@/components/viz/Sparkline'
import { useT } from '@/hooks/useT'
import { formatTime } from '@/lib/format'
import { useMMLTasks, useMMLTaskResults } from '@core/hooks/api/useMML'
import { getMmlTaskProgress } from '@core/utils/mmlTaskProgress'
import type {
  MMLTask,
  MMLTaskStatus,
  MMLExecuteType,
  MMLTaskOrigin,
  DeviceTaskResultItem,
  MMLTaskResultsStats,
} from '@core/types/mml'

// ─────────────────────────────────────────────────────────────
// TASK ARCHIVE · mml_tasks 执行记录（real：useMMLTasks + useMMLTaskResults）
// v1 路由 mml/task-records（TaskRecord）—— 只读任务记录 + 逐设备结果下钻。
// ─────────────────────────────────────────────────────────────

const STATUS_BADGE: Record<MMLTaskStatus, { tone: string; label: string }> = {
  pending: { tone: 'off', label: '待执行' },
  running: { tone: 'warning', label: '执行中' },
  paused: { tone: 'minor', label: '已暂停' },
  completed: { tone: 'ok', label: '已完成' },
  cancelled: { tone: 'offline', label: '已取消' },
  failed: { tone: 'critical', label: '失败' },
}

const EXEC_TYPE_LABEL: Record<MMLExecuteType, { label: string; color: string }> = {
  immediate: { label: '立即执行', color: '#00ff88' },
  suspended: { label: '挂起', color: '#ff7a1a' },
  scheduled: { label: '定时执行', color: '#00f0ff' },
  periodic: { label: '周期任务', color: '#a855f7' },
}

const TASK_ORIGIN_LABEL: Record<MMLTaskOrigin, { key: string; color: string }> = {
  console: { key: 'mml.taskOrigin.console', color: '#00f0ff' },
  script: { key: 'mml.taskOrigin.script', color: '#a855f7' },
}

const TASK_STATUS_ORDER: MMLTaskStatus[] = [
  'pending',
  'running',
  'paused',
  'completed',
  'cancelled',
  'failed',
]

function statusLabel(s: MMLTaskStatus): string {
  return STATUS_BADGE[s]?.label ?? s
}

const PAGE_SIZE = 20
const TASK_RESULT_PAGE_SIZE = 20

export function MMLTaskRecordsPage() {
  const tr = useT()
  const [page, setPage] = useState(1)
  const [taskName, setTaskName] = useState('')
  const [status, setStatus] = useState<MMLTaskStatus | ''>('')
  const [taskOrigin, setTaskOrigin] = useState<MMLTaskOrigin | ''>('')
  const [executeType, setExecuteType] = useState<MMLExecuteType | ''>('')
  const [result, setResult] = useState<'success' | 'partial' | 'failed' | ''>('')
  const [viewing, setViewing] = useState<MMLTask | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(taskName.trim() ? { taskName: taskName.trim() } : {}),
      ...(taskOrigin ? { taskOrigin } : {}),
      ...(status ? { status } : {}),
      ...(executeType ? { executeType } : {}),
      ...(result ? { result } : {}),
    }),
    [page, taskName, taskOrigin, status, executeType, result]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useMMLTasks(params)
  const tasks = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const statusCounts = useMemo(() => {
    const m: Record<MMLTaskStatus, number> = {
      pending: 0,
      running: 0,
      paused: 0,
      completed: 0,
      cancelled: 0,
      failed: 0,
    }
    for (const t of tasks) {
      if (m[t.status] !== undefined) m[t.status] += 1
    }
    return m
  }, [tasks])

  return (
    <PageShell
      code="F06"
      title="TASK ARCHIVE · 任务记录"
      subtitle="MAN-MACHINE LANGUAGE · mml_tasks EXECUTION LOG"
      bare
      isFetching={isFetching}
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          {isFetching ? 'SYNC…' : 'REFRESH'}
        </NeonButton>
      }
    >
      <div className="flex h-full flex-col gap-3">
        {/* 统计带 */}
        <div className="grid grid-cols-4 gap-3 md:grid-cols-7">
          <StatCard label="TOTAL" value={total} color="#00f0ff" trend={[6, 9, 7, 12, 10, 14, 16]} />
          {TASK_STATUS_ORDER.map((s) => (
            <StatCard
              key={s}
              label={statusLabel(s)}
              value={statusCounts[s]}
              color={
                s === 'failed' || s === 'cancelled'
                  ? '#ff2d6f'
                  : s === 'completed'
                    ? '#00ff88'
                    : s === 'running'
                      ? '#ffaa00'
                      : '#5b9eff'
              }
            />
          ))}
        </div>

        {/* 筛选 */}
        <div className="flex flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-56 pl-9"
              placeholder="任务名称"
              value={taskName}
              onChange={(e) => {
                setTaskName(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <FilterChips<MMLExecuteType>
            all="ALL TYPE"
            value={executeType}
            onChange={(v) => {
              setExecuteType(v)
              setPage(1)
            }}
            options={(['immediate', 'suspended', 'scheduled', 'periodic'] as const).map((v) => ({
              value: v,
              label: EXEC_TYPE_LABEL[v].label,
              color: EXEC_TYPE_LABEL[v].color,
            }))}
          />
          <FilterChips<MMLTaskOrigin>
            all="ALL ORIGIN"
            value={taskOrigin}
            onChange={(v) => {
              setTaskOrigin(v)
              setPage(1)
            }}
            options={(['console', 'script'] as const).map((v) => ({
              value: v,
              label: tr(TASK_ORIGIN_LABEL[v].key),
              color: TASK_ORIGIN_LABEL[v].color,
            }))}
          />
          <FilterChips<MMLTaskStatus>
            all="ALL STATUS"
            value={status}
            onChange={(v) => {
              setStatus(v)
              setPage(1)
            }}
            options={TASK_STATUS_ORDER.map((v) => ({
              value: v,
              label: statusLabel(v),
              color:
                v === 'failed' || v === 'cancelled' ? '#ff2d6f' : v === 'completed' ? '#00ff88' : '#5b9eff',
            }))}
          />
          <FilterChips<'success' | 'partial' | 'failed'>
            all="ALL RESULT"
            value={result}
            onChange={(v) => {
              setResult(v)
              setPage(1)
            }}
            options={[
              { value: 'success', label: '成功', color: '#00ff88' },
              { value: 'partial', label: '部分成功', color: '#ffaa00' },
              { value: 'failed', label: '失败', color: '#ff2d6f' },
            ]}
          />
        </div>

        {/* 列表 */}
        <GlassPanel
          title="TASK ARCHIVE · 执行记录"
          meta={`PAGE ${page}/${totalPages}`}
          className="min-h-0 flex-1 overflow-hidden"
        >
          <div className="h-full overflow-auto">
            <div className="sticky top-0 z-10 grid grid-cols-[2fr_1fr_1fr_1fr_1fr_1.2fr_1.4fr_70px] gap-3 border-b border-cyan-500/20 bg-[#03050d]/85 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/60 backdrop-blur">
              <span>任务名 / ID</span>
              <span>执行人</span>
              <span>{tr('mml.taskOrigin')}</span>
              <span>类型</span>
              <span>状态</span>
              <span>进度</span>
              <span>创建时间</span>
              <span className="text-right">操作</span>
            </div>

            {isLoading ? (
              <CenterState>
                <Loader2 className="size-4 animate-spin" />
                <span>SYNCING TASK ARCHIVE…</span>
              </CenterState>
            ) : isError ? (
              <CenterState tone="err">
                <XCircle className="size-5" />
                <span>FAILURE · {error instanceof Error ? error.message : '未知错误'}</span>
              </CenterState>
            ) : tasks.length === 0 ? (
              <CenterState>
                <Inbox className="size-6" />
                <span>无任务记录</span>
              </CenterState>
            ) : (
              tasks.map((t) => {
                const { done, total: tot } = getMmlTaskProgress(t)
                const pct = tot > 0 ? Math.round((done / tot) * 100) : 0
                const et = EXEC_TYPE_LABEL[t.executeType]
                const origin = TASK_ORIGIN_LABEL[t.taskOrigin]
                return (
                  <div
                    key={t.id}
                    className="grid grid-cols-[2fr_1fr_1fr_1fr_1fr_1.2fr_1.4fr_70px] items-center gap-3 border-b border-cyan-500/8 px-3 py-2.5 hover:bg-cyan-500/5"
                  >
                    <div className="min-w-0">
                      <div className="truncate font-display text-sm font-bold text-cyan-100">
                        {t.taskName || '—'}
                      </div>
                      <div className="truncate font-mono text-[10px] text-cyan-300/45">{t.id}</div>
                    </div>
                    <div className="truncate text-xs text-cyan-100/80">{t.creator || '—'}</div>
                    <div>
                      <span className="chip" style={{ color: origin?.color ?? '#6b86b6' }}>
                        {origin ? tr(origin.key) : t.taskOrigin}
                      </span>
                    </div>
                    <div>
                      <span className="chip" style={{ color: et?.color ?? '#6b86b6' }}>
                        {et?.label ?? t.executeType}
                      </span>
                    </div>
                    <div>
                      <StatusBadge status={STATUS_BADGE[t.status]?.tone ?? 'unknown'} label={statusLabel(t.status)} />
                    </div>
                    <div className="flex items-center gap-2">
                      <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-cyan-500/10">
                        <div
                          className="h-full rounded-full"
                          style={{
                            width: `${pct}%`,
                            background: t.status === 'failed' ? '#ff2d6f' : '#00f0ff',
                            boxShadow: '0 0 6px currentColor',
                            color: t.status === 'failed' ? '#ff2d6f' : '#00f0ff',
                          }}
                        />
                      </div>
                      <span className="font-mono text-[10px] text-cyan-300/65">
                        {done}/{tot}
                      </span>
                    </div>
                    <div className="font-mono text-[11px] text-cyan-300/70">{formatTime(t.createdAt)}</div>
                    <div className="text-right">
                      <button
                        type="button"
                        onClick={() => setViewing(t)}
                        className="rounded-sm border border-cyan-500/30 bg-cyan-500/10 px-2 py-1 font-mono text-[10px] uppercase tracking-[0.15em] text-cyan-200 hover:border-cyan-400/60 hover:bg-cyan-500/20"
                      >
                        查看
                      </button>
                    </div>
                  </div>
                )
              })
            )}
          </div>
        </GlassPanel>

        <Pager page={page} totalPages={totalPages} total={total} onPage={setPage} />
      </div>

      {viewing && <TaskDetailDrawer key={viewing.id} task={viewing} onClose={() => setViewing(null)} />}
    </PageShell>
  )
}

function TaskDetailDrawer({ task, onClose }: { task: MMLTask; onClose: () => void }) {
  const tr = useT()
  const [resultPage, setResultPage] = useState(1)
  const { data, isLoading, isError } = useMMLTaskResults(task.id, resultPage, TASK_RESULT_PAGE_SIZE)
  const rows: DeviceTaskResultItem[] = useMemo(
    () => data?.items ?? [],
    [data]
  )
  // apiSwitch 把真实 API（stats: MMLTaskResultsStats）与 mock（默认 stats {}）
  // 的返回类型取交集，stats 被收窄为 {}；运行期真实 API 会填充翻译审计元数据，
  // 这里按已声明的 MMLTaskResultsStats 收窄读取。
  const stats = data?.stats as MMLTaskResultsStats | undefined

  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/60 backdrop-blur-sm" onClick={onClose}>
      <div
        className="glass-strong flex h-full w-full max-w-[760px] flex-col overflow-hidden border-l border-cyan-500/30"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-cyan-500/20 px-4 py-3">
          <div className="min-w-0">
            <div className="truncate font-display text-base font-bold text-cyan-100">
              {task.taskName || '执行结果'}
            </div>
            <div className="truncate font-mono text-[10px] text-cyan-300/50">{task.id}</div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-sm border border-cyan-500/25 p-1.5 text-cyan-300/70 hover:border-cyan-400/60 hover:text-cyan-100"
          >
            <X className="size-4" />
          </button>
        </div>

        <div className="grid grid-cols-2 gap-3 border-b border-cyan-500/15 px-4 py-3 sm:grid-cols-4">
          <Meta label="状态" value={statusLabel(task.status)} />
          <Meta
            label={tr('mml.taskOrigin')}
            value={TASK_ORIGIN_LABEL[task.taskOrigin] ? tr(TASK_ORIGIN_LABEL[task.taskOrigin].key) : task.taskOrigin}
          />
          <Meta label="类型" value={EXEC_TYPE_LABEL[task.executeType]?.label ?? task.executeType} />
          <Meta label="设备数" value={String(task.totalDevices ?? 0)} />
          <Meta label="成功 / 失败" value={`${task.successCount ?? 0} / ${task.failedCount ?? 0}`} />
          <Meta label="创建" value={formatTime(task.createdAt)} />
          <Meta label="开始" value={formatTime(task.startedAt)} />
          <Meta label="结束" value={formatTime(task.finishedAt)} />
          <Meta label="执行人" value={task.creator || '—'} />
        </div>

        {task.productResolved === false && (
          <div className="mx-4 mt-3 rounded-sm border border-orange-500/40 bg-orange-500/8 px-3 py-2 font-mono text-[11px] text-orange-300">
            设备 product_class 未匹配产品，所有 path 走原路径下发（orphan_passthrough）。建议运维补登记 product_class_patterns。
          </div>
        )}
        {stats?.pathTranslationSource && stats.pathTranslationSource !== 'discovered' && (
          <div className="mx-4 mt-2 font-mono text-[10px] text-cyan-300/55">
            路径翻译来源：{stats.pathTranslationSource}
            {stats.matchedProductClass ? ` · ${stats.matchedProductClass}` : ''}
          </div>
        )}

        {task.commandsDetail && task.commandsDetail.length > 0 && (
          <div className="mx-4 mt-3 rounded-sm border border-cyan-500/15 bg-cyan-500/4 p-2.5">
            <div className="mb-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
              命令明细 · {task.commandsDetail.length}
            </div>
            <div className="space-y-1">
              {task.commandsDetail.map((c, i) => (
                <div key={`${c.commandCode}-${i}`} className="font-mono text-[11px] text-cyan-200/85">
                  {c.operationType ? `[${c.operationType}] ` : ''}
                  {c.commandName || c.commandCode}
                  {c.paramPaths && c.paramPaths.length > 0 ? (
                    <span className="text-cyan-300/45"> · {c.paramPaths.length} path</span>
                  ) : null}
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="min-h-0 flex-1 overflow-auto px-4 py-3">
          <div className="mb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">逐设备执行结果</div>
          {isLoading ? (
            <CenterState>
              <Loader2 className="size-4 animate-spin" />
              <span>LOADING RESULTS…</span>
            </CenterState>
          ) : isError ? (
            <CenterState tone="err">
              <XCircle className="size-5" />
              <span>结果加载失败</span>
            </CenterState>
          ) : rows.length === 0 ? (
            <CenterState>
              <Inbox className="size-5" />
              <span>暂无设备结果</span>
            </CenterState>
          ) : (
            <div className="space-y-2">
              {rows.map((r, i) => {
                const ok = r.result?.success
                return (
                  <div
                    key={`${r.deviceSn}-${r.deviceTaskId ?? i}`}
                    className="rounded-sm border border-cyan-500/15 bg-[#03050d]/50 p-2.5"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <div className="flex items-center gap-2">
                        {ok ? (
                          <CheckCircle2 className="size-3.5 text-emerald-400" />
                        ) : (
                          <XCircle className="size-3.5 text-rose-400" />
                        )}
                        <span className="font-mono text-xs text-cyan-100">{r.deviceSn}</span>
                        {r.deviceName ? <span className="text-[11px] text-cyan-300/55">{r.deviceName}</span> : null}
                        {r.planLineNo ? (
                          <span className="rounded-sm border border-cyan-500/25 bg-cyan-500/8 px-1.5 py-0.5 font-mono text-[10px] text-cyan-300/75">
                            #{r.planLineNo}/{r.planOrder ?? '-'} {r.commandCode ?? ''}
                          </span>
                        ) : null}
                      </div>
                      <StatusBadge status={ok ? 'ok' : 'critical'} label={ok ? '成功' : '失败'} />
                    </div>
                    {r.failReason ? (
                      <div className="mt-1 font-mono text-[11px] text-rose-300/80">{r.failReason}</div>
                    ) : null}
                    {r.result?.rawOutput ? (
                      <pre className="mt-1.5 max-h-40 overflow-auto whitespace-pre-wrap rounded-sm bg-black/40 p-2 font-mono text-[11px] leading-relaxed text-emerald-300/85">
                        {r.result.rawOutput}
                      </pre>
                    ) : null}
                  </div>
                )
              })}
            </div>
          )}
          {(data?.total ?? 0) > TASK_RESULT_PAGE_SIZE ? (
            <div className="mt-3 flex items-center justify-between">
              <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                RESULT PAGE {resultPage} / {Math.max(1, Math.ceil((data?.total ?? 0) / TASK_RESULT_PAGE_SIZE))}
              </span>
              <div className="flex gap-2">
                <NeonButton onClick={() => setResultPage((p) => Math.max(1, p - 1))} disabled={resultPage <= 1}>
                  ◂ PREV
                </NeonButton>
                <NeonButton
                  onClick={() =>
                    setResultPage((p) => Math.min(Math.max(1, Math.ceil((data?.total ?? 0) / TASK_RESULT_PAGE_SIZE)), p + 1))
                  }
                  disabled={resultPage >= Math.max(1, Math.ceil((data?.total ?? 0) / TASK_RESULT_PAGE_SIZE))}
                >
                  NEXT ▸
                </NeonButton>
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  )
}

function StatCard({
  label,
  value,
  color,
  trend,
}: {
  label: string
  value: number
  color: string
  trend?: number[]
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: color }}
    >
      <div className="truncate font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/60">{label}</div>
      <div className="flex items-end justify-between gap-1">
        <div className="font-display text-2xl font-bold leading-tight" style={{ color, textShadow: `0 0 8px ${color}` }}>
          {value}
        </div>
        {trend ? <Sparkline data={trend} color={color} width={54} height={22} /> : null}
      </div>
    </div>
  )
}

function FilterChips<T extends string>({
  all,
  value,
  onChange,
  options,
}: {
  all: string
  value: T | ''
  onChange: (v: T | '') => void
  options: Array<{ value: T; label: string; color: string }>
}) {
  return (
    <div className="flex flex-wrap items-center gap-1">
      <button
        type="button"
        onClick={() => onChange('')}
        className={`chip transition-all ${value === '' ? 'shadow-[0_0_8px_currentColor]' : 'opacity-55 hover:opacity-100'}`}
        style={{ color: '#00f0ff' }}
      >
        {all}
      </button>
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          onClick={() => onChange(value === o.value ? '' : o.value)}
          className={`chip transition-all ${value === o.value ? 'shadow-[0_0_8px_currentColor]' : 'opacity-55 hover:opacity-100'}`}
          style={{ color: o.color }}
        >
          {o.label}
        </button>
      ))}
    </div>
  )
}

function Pager({
  page,
  totalPages,
  total,
  onPage,
}: {
  page: number
  totalPages: number
  total: number
  onPage: (updater: (p: number) => number) => void
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <NeonButton onClick={() => onPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
          ◂ PREV
        </NeonButton>
        <NeonButton onClick={() => onPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages}>
          NEXT ▸
        </NeonButton>
      </div>
    </div>
  )
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/50">{label}</div>
      <div className="truncate text-xs text-cyan-100/90">{value}</div>
    </div>
  )
}

function CenterState({
  children,
  tone = 'cyan',
}: {
  children: React.ReactNode
  tone?: 'cyan' | 'err'
}) {
  return (
    <div
      className={`flex flex-col items-center justify-center gap-2 py-14 font-mono text-xs uppercase tracking-[0.2em] ${
        tone === 'err' ? 'text-rose-300/80' : 'text-cyan-300/60'
      }`}
    >
      {children}
    </div>
  )
}

export default MMLTaskRecordsPage
