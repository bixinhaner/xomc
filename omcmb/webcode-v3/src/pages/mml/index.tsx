import { useEffect, useMemo, useRef, useState } from 'react'
import {
  Play,
  TerminalSquare,
  ListChecks,
  ScrollText,
  RefreshCcw,
  Search,
  Loader2,
  Inbox,
  X,
  CheckCircle2,
  XCircle,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { Sparkline } from '@/components/viz/Sparkline'
import { formatTime } from '@/lib/format'
import {
  useMMLTasks,
  useMMLScripts,
  useAllMMLCommands,
  useMMLTaskResults,
} from '@core/hooks/api/useMML'
import type {
  MMLTask,
  MMLScript,
  MMLCommand,
  MMLTaskStatus,
  MMLExecuteType,
  DeviceTaskResultItem,
  MMLTaskResultsStats,
} from '@core/types/mml'

// ─────────────────────────────────────────────────────────────
// 显示映射 — mml_tasks / mml_scripts 列状态着色（沿用 v1 业务语义）
// ─────────────────────────────────────────────────────────────

type Tab = 'console' | 'tasks' | 'scripts'

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

function progressOf(t: MMLTask): { done: number; total: number } {
  const done = (t.successCount ?? 0) + (t.failedCount ?? 0)
  // 进度分母 = 设备数 × 命令数（逐 PATH 拆成 N 条 device_task），与 v1 TaskRecord 一致
  const total = (t.totalDevices ?? 0) * Math.max(1, t.commands?.length ?? 1)
  return { done, total }
}

// ─────────────────────────────────────────────────────────────
// 本地终端命令模拟器（保留 v3 既有「桥舰 CRT」交互；真实下发走任务记录通道）
// ─────────────────────────────────────────────────────────────

interface Line {
  id: string
  type: 'in' | 'out' | 'sys' | 'err'
  text: string
}

const BANNER = [
  '──────────────────────────────────────────────',
  '  STARFORGE · MML CONSOLE · v3.0   READY',
  '  TR-069 / CWMP MAN-MACHINE LANGUAGE TERMINAL',
  '──────────────────────────────────────────────',
  '',
  '> 本终端为本地交互演示；批量下发请用 TASK ARCHIVE / SCRIPT VAULT。',
  '> 键入 HELP 查看命令清单，CMD 查看后端命令字典。',
  '',
]

function localExec(cmd: string, commands: MMLCommand[]): string[] {
  const upper = cmd.trim().toUpperCase()
  if (!upper) return []
  if (upper === 'CLS') return ['__CLS__']
  if (upper === 'HELP' || upper === '?') {
    return [
      '可用命令：',
      '  CMD                           · 列出后端命令字典（real）',
      '  LST DEV                       · 演示：查询设备',
      '  GET PARAM <PATH>              · 演示：读取 TR-069 参数',
      '  SET PARAM <PATH> = <VAL>      · 演示：修改参数',
      '  RBT DEV   <SN>                · 演示：重启设备',
      '  ACT CELL  <ID>                · 演示：激活小区',
      '  CLS                           · 清屏',
    ]
  }
  if (upper === 'CMD') {
    if (commands.length === 0) return ['> 命令字典为空或加载中…']
    const head = ['+ 后端命令字典 (mml_commands) +', '']
    const body = commands.slice(0, 40).map((c) => {
      const op = c.operationType ? `[${c.operationType}] ` : ''
      return `  ${op}${c.commandCode.padEnd(18)} ${c.commandName}`
    })
    const tail = commands.length > 40 ? ['', `… 共 ${commands.length} 条，仅显示前 40 条`] : ['', `共 ${commands.length} 条`]
    return [...head, ...body, ...tail]
  }
  if (upper.startsWith('LST DEV')) {
    return [
      '+----+-------------------+----------+--------+',
      '| #  | SN                | TYPE     | STATE  |',
      '+----+-------------------+----------+--------+',
      '| 1  | ENB-2241-PUS      | LTE      | ONLINE |',
      '| 2  | GNB-1102-LXK      | 5G NR    | ONLINE |',
      '| 3  | CPE-A1B0E9-CD     | CPE      | OFFLINE|',
      '+----+-------------------+----------+--------+',
      '3 rows · demo',
    ]
  }
  if (upper.startsWith('RBT')) {
    return ['> 已发起重启请求 (demo)', '> Connection-Request → CPE …', '> [OK] 设备已上线']
  }
  if (upper.startsWith('ACT') || upper.startsWith('DACT')) {
    return ['> SetParameterValues SOAP → (demo)', '> [DONE] 状态：ACTIVE']
  }
  if (upper.startsWith('GET PARAM')) {
    return [`> ${upper}`, '> Device.DeviceInfo.SoftwareVersion = 6.4.2 (demo)']
  }
  if (upper.startsWith('SET PARAM')) {
    return [`> ${upper}`, '> SetParameterValues OK (demo)']
  }
  return [`unknown command: ${cmd}`, "type 'HELP' for command list"]
}

// ─────────────────────────────────────────────────────────────
// 主页面
// ─────────────────────────────────────────────────────────────

export function MMLPage() {
  const [tab, setTab] = useState<Tab>('console')

  return (
    <PageShell
      code="F06"
      title="MML COMMAND CENTER · 命令中枢"
      subtitle="MAN-MACHINE LANGUAGE · CONSOLE / TASK ARCHIVE / SCRIPT VAULT"
      bare
      toolbar={
        <div className="flex items-center gap-1.5">
          <TabBtn active={tab === 'console'} onClick={() => setTab('console')} icon={<TerminalSquare />}>
            CONSOLE
          </TabBtn>
          <TabBtn active={tab === 'tasks'} onClick={() => setTab('tasks')} icon={<ListChecks />}>
            TASK ARCHIVE
          </TabBtn>
          <TabBtn active={tab === 'scripts'} onClick={() => setTab('scripts')} icon={<ScrollText />}>
            SCRIPT VAULT
          </TabBtn>
        </div>
      }
    >
      {tab === 'console' && <ConsoleView />}
      {tab === 'tasks' && <TaskArchiveView />}
      {tab === 'scripts' && <ScriptVaultView />}
    </PageShell>
  )
}

function TabBtn({
  active,
  onClick,
  icon,
  children,
}: {
  active: boolean
  onClick: () => void
  icon: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex items-center gap-1.5 rounded-sm border px-3 py-1.5 font-mono text-[11px] uppercase tracking-[0.18em] transition-all [&_svg]:size-3.5 ${
        active
          ? 'border-cyan-400/60 bg-cyan-500/15 text-cyan-100 shadow-[0_0_10px_rgba(0,240,255,0.25)]'
          : 'border-cyan-500/20 bg-cyan-500/5 text-cyan-300/60 hover:text-cyan-200'
      }`}
    >
      {icon}
      {children}
    </button>
  )
}

// ─────────────────────────────────────────────────────────────
// CONSOLE — 本地终端 + 后端命令字典快捷宏（real）
// ─────────────────────────────────────────────────────────────

function ConsoleView() {
  const { data: commands = [], isLoading, isError } = useAllMMLCommands()

  const [lines, setLines] = useState<Line[]>(() =>
    BANNER.map((t, i) => ({ id: `b-${i}`, type: 'sys' as const, text: t }))
  )
  const [input, setInput] = useState('')
  const [history, setHistory] = useState<string[]>([])
  const [hi, setHi] = useState(-1)
  const inputRef = useRef<HTMLInputElement>(null)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: 'smooth' })
  }, [lines])

  const runCmd = (raw: string) => {
    const cmd = raw.trim()
    if (!cmd) return
    const echo: Line = { id: `e-${Date.now()}`, type: 'in', text: `OPR>$ ${cmd}` }
    const out = localExec(cmd, commands)
    if (out[0] === '__CLS__') {
      setLines([])
    } else {
      setLines((prev) => [
        ...prev,
        echo,
        ...out.map((t, i) => ({ id: `o-${Date.now()}-${i}`, type: 'out' as const, text: t })),
      ])
    }
    setHistory((h) => [cmd, ...h].slice(0, 50))
    setHi(-1)
    setInput('')
  }

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    runCmd(input)
  }

  const onKey = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      const next = Math.min(history.length - 1, hi + 1)
      setHi(next)
      if (next >= 0) setInput(history[next])
    } else if (e.key === 'ArrowDown') {
      e.preventDefault()
      const next = Math.max(-1, hi - 1)
      setHi(next)
      setInput(next === -1 ? '' : history[next])
    }
  }

  // 命令字典宏：取后端真实命令码（前 16 条）+ 几条固定演示
  const catalogMacros = useMemo(
    () => commands.slice(0, 16).map((c) => c.commandCode),
    [commands]
  )

  return (
    <div className="grid h-full grid-cols-[1fr_300px] gap-3">
      <GlassPanel title="TERMINAL · BRIDGE-04" meta="LOCAL EMULATION" className="min-h-0">
        <div className="terminal m-3 flex h-[calc(100%-66px)] flex-col">
          <div ref={scrollRef} className="flex-1 overflow-auto p-3 text-[12px]">
            {lines.map((l) => (
              <div
                key={l.id}
                className={
                  l.type === 'in'
                    ? 'text-cyan-300'
                    : l.type === 'err'
                      ? 'text-rose-300'
                      : 'text-emerald-300'
                }
              >
                {l.text || ' '}
              </div>
            ))}
            <div className="terminal-cursor" />
          </div>
          <form
            onSubmit={onSubmit}
            className="flex items-center gap-2 border-t border-emerald-500/30 bg-black/40 px-3 py-2"
          >
            <span className="font-mono text-emerald-400">OPR&gt;$</span>
            <input
              ref={inputRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={onKey}
              spellCheck={false}
              autoComplete="off"
              className="flex-1 bg-transparent font-mono text-sm text-emerald-200 outline-none placeholder:text-emerald-700/60"
              placeholder="键入命令并回车 · 例如 HELP / CMD / LST DEV"
            />
            <button
              type="submit"
              className="flex items-center gap-1 rounded-sm border border-emerald-500/40 bg-emerald-500/10 px-2 py-1 text-[10px] uppercase tracking-[0.18em] text-emerald-300 hover:bg-emerald-500/20"
            >
              <Play className="size-3" />
              EXEC
            </button>
          </form>
        </div>
      </GlassPanel>

      <div className="flex min-h-0 flex-col gap-3">
        <GlassPanel title="QUICK MACROS" meta="DEMO + REAL">
          <div className="grid max-h-[40%] gap-2 p-3">
            {['HELP', 'CMD', 'LST DEV', 'GET PARAM Device.DeviceInfo.SoftwareVersion', 'CLS'].map(
              (m) => (
                <button
                  key={m}
                  onClick={() => {
                    setInput(m)
                    inputRef.current?.focus()
                  }}
                  className="rounded-sm border border-cyan-500/20 bg-cyan-500/5 px-3 py-1.5 text-left font-mono text-[11px] text-cyan-200 hover:border-cyan-400/60 hover:bg-cyan-500/10"
                >
                  {m}
                </button>
              )
            )}
          </div>
        </GlassPanel>

        <GlassPanel
          title="COMMAND CATALOG"
          meta={isLoading ? 'SYNC…' : `${commands.length} CMD`}
          className="min-h-0 flex-1 overflow-hidden"
        >
          {isLoading ? (
            <CenterState>
              <Loader2 className="size-4 animate-spin" />
              <span>LOADING CATALOG…</span>
            </CenterState>
          ) : isError ? (
            <CenterState tone="err">
              <XCircle className="size-5" />
              <span>命令字典加载失败</span>
            </CenterState>
          ) : commands.length === 0 ? (
            <CenterState>
              <Inbox className="size-5" />
              <span>命令字典为空</span>
            </CenterState>
          ) : (
            <div className="grid max-h-full gap-1.5 overflow-auto p-3">
              {catalogMacros.map((code, i) => (
                <button
                  key={`${code}-${i}`}
                  onClick={() => {
                    runCmd(code)
                  }}
                  title={commands[i]?.commandName}
                  className="truncate rounded-sm border border-cyan-500/15 bg-cyan-500/5 px-2.5 py-1 text-left font-mono text-[10.5px] text-cyan-300/80 hover:border-cyan-400/50 hover:text-cyan-100"
                >
                  {commands[i]?.operationType ? `[${commands[i]?.operationType}] ` : ''}
                  {code}
                </button>
              ))}
            </div>
          )}
        </GlassPanel>
      </div>
    </div>
  )
}

// ─────────────────────────────────────────────────────────────
// TASK ARCHIVE — 真实 mml_tasks 列表（统计 + 筛选 + 分页 + 详情下钻）
// ─────────────────────────────────────────────────────────────

function TaskArchiveView() {
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [taskName, setTaskName] = useState('')
  const [status, setStatus] = useState<MMLTaskStatus | ''>('')
  const [executeType, setExecuteType] = useState<MMLExecuteType | ''>('')
  const [result, setResult] = useState<'success' | 'partial' | 'failed' | ''>('')
  const [viewing, setViewing] = useState<MMLTask | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(taskName.trim() ? { taskName: taskName.trim() } : {}),
      ...(status ? { status } : {}),
      ...(executeType ? { executeType } : {}),
      ...(result ? { result } : {}),
    }),
    [page, taskName, status, executeType, result]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useMMLTasks(params)
  const tasks = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  // 统计：按状态分桶（当前页样本，作为态势面板；总量取 total）
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
            className="neon-input w-60 pl-9"
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
              v === 'failed' || v === 'cancelled'
                ? '#ff2d6f'
                : v === 'completed'
                  ? '#00ff88'
                  : '#5b9eff',
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
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          {isFetching ? 'SYNC…' : 'REFRESH'}
        </NeonButton>
      </div>

      {/* 列表 */}
      <GlassPanel title="TASK ARCHIVE · 执行记录" meta={`PAGE ${page}/${totalPages}`} className="min-h-0 flex-1 overflow-hidden">
        <div className="h-full overflow-auto">
          {/* 表头 */}
          <div className="sticky top-0 z-10 grid grid-cols-[2fr_1fr_1fr_1fr_1.2fr_1.4fr_70px] gap-3 border-b border-cyan-500/20 bg-[#03050d]/85 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/60 backdrop-blur">
            <span>任务名 / ID</span>
            <span>执行人</span>
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
              const { done, total: tot } = progressOf(t)
              const pct = tot > 0 ? Math.round((done / tot) * 100) : 0
              const et = EXEC_TYPE_LABEL[t.executeType]
              return (
                <div
                  key={t.id}
                  className="grid grid-cols-[2fr_1fr_1fr_1fr_1.2fr_1.4fr_70px] items-center gap-3 border-b border-cyan-500/8 px-3 py-2.5 hover:bg-cyan-500/5"
                >
                  <div className="min-w-0">
                    <div className="truncate font-display text-sm font-bold text-cyan-100">
                      {t.taskName || '—'}
                    </div>
                    <div className="truncate font-mono text-[10px] text-cyan-300/45">{t.id}</div>
                  </div>
                  <div className="truncate text-xs text-cyan-100/80">{t.creator || '—'}</div>
                  <div>
                    <span
                      className="chip"
                      style={{ color: et?.color ?? '#6b86b6' }}
                    >
                      {et?.label ?? t.executeType}
                    </span>
                  </div>
                  <div>
                    <StatusBadge
                      status={STATUS_BADGE[t.status]?.tone ?? 'unknown'}
                      label={statusLabel(t.status)}
                    />
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
                  <div className="font-mono text-[11px] text-cyan-300/70">
                    {formatTime(t.createdAt)}
                  </div>
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

      <Pager page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} />

      {viewing && <TaskDetailDrawer task={viewing} onClose={() => setViewing(null)} />}
    </div>
  )
}

// ─────────────────────────────────────────────────────────────
// 任务详情下钻 — 逐设备执行结果（real：useMMLTaskResults）
// ─────────────────────────────────────────────────────────────

function TaskDetailDrawer({ task, onClose }: { task: MMLTask; onClose: () => void }) {
  const { data, isLoading, isError } = useMMLTaskResults(task.id, 1, 200)
  const rows: DeviceTaskResultItem[] = useMemo(
    () => data?.items ?? (task.results as unknown as DeviceTaskResultItem[]) ?? [],
    [data, task]
  )
  // apiSwitch 取真实 API（stats: MMLTaskResultsStats）与 mock（默认 stats {}）的
  // 返回类型交集，stats 被收窄为 {}；运行期真实 API 填充翻译审计元数据，按已声明类型读取。
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

        {/* 任务元信息 */}
        <div className="grid grid-cols-2 gap-3 border-b border-cyan-500/15 px-4 py-3 sm:grid-cols-4">
          <Meta label="状态" value={statusLabel(task.status)} />
          <Meta label="类型" value={EXEC_TYPE_LABEL[task.executeType]?.label ?? task.executeType} />
          <Meta label="设备数" value={String(task.totalDevices ?? 0)} />
          <Meta
            label="成功 / 失败"
            value={`${task.successCount ?? 0} / ${task.failedCount ?? 0}`}
          />
          <Meta label="创建" value={formatTime(task.createdAt)} />
          <Meta label="开始" value={formatTime(task.startedAt)} />
          <Meta label="结束" value={formatTime(task.finishedAt)} />
          <Meta label="执行人" value={task.creator || '—'} />
        </div>

        {/* 路径翻译告警（v1 业务深度：orphan/未翻译提示） */}
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

        {/* 命令明细 */}
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

        {/* 逐设备结果 */}
        <div className="min-h-0 flex-1 overflow-auto px-4 py-3">
          <div className="mb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            逐设备执行结果
          </div>
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
                        {r.deviceName ? (
                          <span className="text-[11px] text-cyan-300/55">{r.deviceName}</span>
                        ) : null}
                      </div>
                      <StatusBadge
                        status={ok ? 'ok' : 'critical'}
                        label={ok ? '成功' : '失败'}
                      />
                    </div>
                    {r.failReason ? (
                      <div className="mt-1 font-mono text-[11px] text-rose-300/80">
                        {r.failReason}
                      </div>
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
        </div>
      </div>
    </div>
  )
}

// ─────────────────────────────────────────────────────────────
// SCRIPT VAULT — 真实 mml_scripts 脚本库列表（筛选 + 分页 + 详情下钻）
// ─────────────────────────────────────────────────────────────

function ScriptVaultView() {
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [search, setSearch] = useState('')
  const [viewing, setViewing] = useState<MMLScript | null>(null)

  const params = useMemo(
    () => ({ page, pageSize, ...(search.trim() ? { search: search.trim() } : {}) }),
    [page, search]
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useMMLScripts(params)
  const scripts = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
          <input
            className="neon-input w-72 pl-9"
            placeholder="脚本名称"
            value={search}
            onChange={(e) => {
              setSearch(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          {isFetching ? 'SYNC…' : 'REFRESH'}
        </NeonButton>
        <span className="ml-auto font-mono text-[11px] text-cyan-300/55">
          脚本库 · {total} 条 · 只读视图（执行 / 编辑见 v1）
        </span>
      </div>

      <GlassPanel title="SCRIPT VAULT · 脚本库" meta={`PAGE ${page}/${totalPages}`} className="min-h-0 flex-1 overflow-hidden">
        <div className="h-full overflow-auto">
          <div className="sticky top-0 z-10 grid grid-cols-[2fr_2.4fr_1fr_1fr_1.4fr] gap-3 border-b border-cyan-500/20 bg-[#03050d]/85 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/60 backdrop-blur">
            <span>脚本名</span>
            <span>描述 / 标签</span>
            <span>创建人</span>
            <span>状态</span>
            <span>更新时间</span>
          </div>

          {isLoading ? (
            <CenterState>
              <Loader2 className="size-4 animate-spin" />
              <span>SYNCING SCRIPT VAULT…</span>
            </CenterState>
          ) : isError ? (
            <CenterState tone="err">
              <XCircle className="size-5" />
              <span>FAILURE · {error instanceof Error ? error.message : '未知错误'}</span>
            </CenterState>
          ) : scripts.length === 0 ? (
            <CenterState>
              <Inbox className="size-6" />
              <span>无脚本</span>
            </CenterState>
          ) : (
            scripts.map((s) => (
              <button
                type="button"
                key={s.id}
                onClick={() => setViewing(s)}
                className="grid w-full grid-cols-[2fr_2.4fr_1fr_1fr_1.4fr] items-center gap-3 border-b border-cyan-500/8 px-3 py-2.5 text-left hover:bg-cyan-500/5"
              >
                <div className="min-w-0">
                  <div className="truncate font-display text-sm font-bold text-cyan-100">
                    {s.scriptName}
                  </div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/45">{s.id}</div>
                </div>
                <div className="min-w-0">
                  <div className="truncate text-xs text-cyan-100/75">{s.description || '—'}</div>
                  {s.tags && s.tags.length > 0 ? (
                    <div className="mt-0.5 flex flex-wrap gap-1">
                      {s.tags.slice(0, 4).map((tag) => (
                        <span
                          key={tag}
                          className="rounded-sm border border-cyan-500/25 bg-cyan-500/8 px-1.5 py-0.5 font-mono text-[9px] text-cyan-300/75"
                        >
                          {tag}
                        </span>
                      ))}
                    </div>
                  ) : null}
                </div>
                <div className="truncate text-xs text-cyan-100/80">{s.creator || '—'}</div>
                <div>
                  <StatusBadge status={scriptTone(s.status)} label={s.status} />
                </div>
                <div className="font-mono text-[11px] text-cyan-300/70">
                  {formatTime(s.updateTime)}
                </div>
              </button>
            ))
          )}
        </div>
      </GlassPanel>

      <Pager page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} />

      {viewing && <ScriptDetailDrawer script={viewing} onClose={() => setViewing(null)} />}
    </div>
  )
}

function scriptTone(status: string): string {
  switch (status) {
    case 'active':
    case 'completed':
      return 'ok'
    case 'running':
      return 'warning'
    case 'failed':
    case 'cancelled':
      return 'critical'
    case 'paused':
      return 'minor'
    default:
      return 'off'
  }
}

function ScriptDetailDrawer({ script, onClose }: { script: MMLScript; onClose: () => void }) {
  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/60 backdrop-blur-sm" onClick={onClose}>
      <div
        className="glass-strong flex h-full w-full max-w-[640px] flex-col overflow-hidden border-l border-cyan-500/30"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-cyan-500/20 px-4 py-3">
          <div className="min-w-0">
            <div className="truncate font-display text-base font-bold text-cyan-100">
              {script.scriptName}
            </div>
            <div className="truncate font-mono text-[10px] text-cyan-300/50">{script.id}</div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-sm border border-cyan-500/25 p-1.5 text-cyan-300/70 hover:border-cyan-400/60 hover:text-cyan-100"
          >
            <X className="size-4" />
          </button>
        </div>

        <div className="grid grid-cols-2 gap-3 border-b border-cyan-500/15 px-4 py-3">
          <Meta label="状态" value={script.status} />
          <Meta label="创建人" value={script.creator || '—'} />
          <Meta label="创建时间" value={formatTime(script.createTime)} />
          <Meta label="更新时间" value={formatTime(script.updateTime)} />
        </div>

        {script.description ? (
          <div className="border-b border-cyan-500/15 px-4 py-3">
            <div className="mb-1 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
              描述
            </div>
            <div className="text-sm text-cyan-100/85">{script.description}</div>
          </div>
        ) : null}

        {script.tags && script.tags.length > 0 ? (
          <div className="flex flex-wrap gap-1.5 border-b border-cyan-500/15 px-4 py-3">
            {script.tags.map((tag) => (
              <span
                key={tag}
                className="rounded-sm border border-cyan-500/25 bg-cyan-500/8 px-2 py-0.5 font-mono text-[10px] text-cyan-300/80"
              >
                {tag}
              </span>
            ))}
          </div>
        ) : null}

        <div className="min-h-0 flex-1 overflow-auto px-4 py-3">
          <div className="mb-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            脚本内容
          </div>
          <pre className="overflow-auto whitespace-pre-wrap rounded-sm border border-emerald-500/20 bg-black/40 p-3 font-mono text-[12px] leading-relaxed text-emerald-300/90">
            {script.content || '(空)'}
          </pre>
        </div>
      </div>
    </div>
  )
}

// ─────────────────────────────────────────────────────────────
// 共享小组件（本模块目录内）
// ─────────────────────────────────────────────────────────────

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
      <div className="truncate font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/60">
        {label}
      </div>
      <div className="flex items-end justify-between gap-1">
        <div
          className="font-display text-2xl font-bold leading-tight"
          style={{ color, textShadow: `0 0 8px ${color}` }}
        >
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
  pageSize,
  onPage,
}: {
  page: number
  totalPages: number
  total: number
  pageSize: number
  onPage: (updater: (p: number) => number) => void
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {pageSize}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <NeonButton onClick={() => onPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
          ◂ PREV
        </NeonButton>
        <NeonButton
          onClick={() => onPage((p) => Math.min(totalPages, p + 1))}
          disabled={page >= totalPages}
        >
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
