import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  RefreshCcw,
  Loader2,
  Inbox,
  XCircle,
  ChevronRight,
  Boxes,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { Sparkline } from '@/components/viz/Sparkline'
import { useAllMMLCommands } from '@core/hooks/api/useMML'
import type { MMLCommand, MMLOperationType } from '@core/types/mml'

// ─────────────────────────────────────────────────────────────
// COMMAND CATALOG · mml_commands 全量命令字典（real：useAllMMLCommands）
// v1 路由 mml/commands（CommandTree）—— 分类树 + 命令表 + 详情下钻。
// 这里以 HUD 风格呈现：左侧分类导航 + 右侧命令网格 + 行→详情 navigate。
// ─────────────────────────────────────────────────────────────

const OP_COLOR: Record<string, string> = {
  LST: '#00f0ff',
  DSP: '#00f0ff',
  MOD: '#ffaa00',
  ADD: '#00ff88',
  RMV: '#ff2d6f',
  ACT: '#00ff88',
  DEA: '#ff7a1a',
  RST: '#a855f7',
  CLR: '#ff7a1a',
  UPG: '#5b9eff',
}

function opColor(op?: string): string {
  return (op && OP_COLOR[op]) || '#6b86b6'
}

const ALL_KEY = '__ALL__'

export function MMLCommandsPage() {
  const navigate = useNavigate()
  const { data: commands = [], isLoading, isError, error, isFetching, refetch } =
    useAllMMLCommands()

  const [search, setSearch] = useState('')
  const [category, setCategory] = useState<string>(ALL_KEY)

  // 派生分类清单（命令真实 category 字段去重）。
  const categories = useMemo(() => {
    const m = new Map<string, number>()
    for (const c of commands) {
      const key = c.category?.trim() || '未分类'
      m.set(key, (m.get(key) ?? 0) + 1)
    }
    return [...m.entries()].sort((a, b) => b[1] - a[1])
  }, [commands])

  // 按操作类型分桶统计（态势带）。
  const opCounts = useMemo(() => {
    const m = new Map<string, number>()
    for (const c of commands) {
      const op = c.operationType ?? 'OTHER'
      m.set(op, (m.get(op) ?? 0) + 1)
    }
    return m
  }, [commands])

  const filtered = useMemo(() => {
    const kw = search.trim().toLowerCase()
    return commands.filter((c) => {
      if (category !== ALL_KEY) {
        const key = c.category?.trim() || '未分类'
        if (key !== category) return false
      }
      if (!kw) return true
      return (
        c.commandName?.toLowerCase().includes(kw) ||
        c.commandCode?.toLowerCase().includes(kw) ||
        c.description?.toLowerCase().includes(kw)
      )
    })
  }, [commands, search, category])

  return (
    <PageShell
      code="F06"
      title="COMMAND CATALOG · 命令字典"
      subtitle="MAN-MACHINE LANGUAGE · mml_commands DICTIONARY"
      bare
      isFetching={isFetching}
      toolbar={
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="命令名 / 命令码 / 描述"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            {isFetching ? 'SYNC…' : 'REFRESH'}
          </NeonButton>
        </div>
      }
    >
      <div className="flex h-full flex-col gap-3">
        {/* 操作类型态势带 */}
        <div className="grid grid-cols-4 gap-3 md:grid-cols-6 lg:grid-cols-8">
          <StatCard label="TOTAL" value={commands.length} color="#00f0ff" trend={[8, 12, 10, 16, 14, 18, 22]} />
          {['LST', 'MOD', 'ADD', 'RMV', 'ACT', 'DEA', 'RST'].map((op) => (
            <StatCard key={op} label={op} value={opCounts.get(op) ?? 0} color={opColor(op)} />
          ))}
        </div>

        <div className="grid min-h-0 flex-1 grid-cols-[230px_1fr] gap-3">
          {/* 分类导航 */}
          <GlassPanel title="CATEGORY" meta={`${categories.length}`} className="min-h-0 overflow-hidden">
            <div className="flex h-full flex-col gap-1 overflow-auto p-2.5">
              <CategoryBtn
                active={category === ALL_KEY}
                label="全部命令"
                count={commands.length}
                onClick={() => setCategory(ALL_KEY)}
              />
              {categories.map(([cat, count]) => (
                <CategoryBtn
                  key={cat}
                  active={category === cat}
                  label={cat}
                  count={count}
                  onClick={() => setCategory(cat)}
                />
              ))}
            </div>
          </GlassPanel>

          {/* 命令网格 */}
          <GlassPanel
            title="COMMANDS"
            meta={`${filtered.length} / ${commands.length}`}
            className="min-h-0 overflow-hidden"
          >
            <div className="h-full overflow-auto p-3">
              {isLoading ? (
                <CenterState>
                  <Loader2 className="size-4 animate-spin" />
                  <span>LOADING CATALOG…</span>
                </CenterState>
              ) : isError ? (
                <CenterState tone="err">
                  <XCircle className="size-5" />
                  <span>命令字典加载失败 · {error instanceof Error ? error.message : '未知错误'}</span>
                </CenterState>
              ) : filtered.length === 0 ? (
                <CenterState>
                  <Inbox className="size-6" />
                  <span>无匹配命令</span>
                </CenterState>
              ) : (
                <div className="grid grid-cols-1 gap-2 lg:grid-cols-2 2xl:grid-cols-3">
                  {filtered.map((c) => (
                    <CommandCard
                      key={c.id}
                      command={c}
                      onOpen={() => navigate(`/mml/commands/${c.id}`)}
                    />
                  ))}
                </div>
              )}
            </div>
          </GlassPanel>
        </div>
      </div>
    </PageShell>
  )
}

function CategoryBtn({
  active,
  label,
  count,
  onClick,
}: {
  active: boolean
  label: string
  count: number
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex items-center justify-between gap-2 rounded-sm border px-2.5 py-2 text-left font-mono text-[11px] transition-all ${
        active
          ? 'border-cyan-400/60 bg-cyan-500/15 text-cyan-100 shadow-[0_0_10px_rgba(0,240,255,0.2)]'
          : 'border-cyan-500/15 bg-cyan-500/5 text-cyan-300/70 hover:border-cyan-400/40 hover:text-cyan-100'
      }`}
    >
      <span className="truncate">{label}</span>
      <span className="shrink-0 rounded-sm bg-cyan-500/15 px-1.5 py-0.5 text-[9px] text-cyan-200/80">
        {count}
      </span>
    </button>
  )
}

function CommandCard({ command, onOpen }: { command: MMLCommand; onOpen: () => void }) {
  const op = command.operationType as MMLOperationType | undefined
  const color = opColor(op)
  const pathCount = command.paramPaths?.length ?? 0
  const refCount = command.paramRefs?.length ?? command.params?.length ?? 0
  return (
    <button
      type="button"
      onClick={onOpen}
      className="group flex flex-col gap-1.5 rounded-sm border border-cyan-500/15 bg-[#03050d]/55 p-3 text-left transition-all hover:border-cyan-400/50 hover:bg-cyan-500/5"
    >
      <div className="flex items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          {op ? (
            <span
              className="shrink-0 rounded-sm border px-1.5 py-0.5 font-mono text-[9px]"
              style={{ color, borderColor: `${color}55` }}
            >
              {op}
            </span>
          ) : null}
          <span className="truncate font-display text-sm font-bold text-cyan-100">
            {command.commandName || command.commandCode}
          </span>
        </div>
        <ChevronRight className="size-3.5 shrink-0 text-cyan-300/40 transition-transform group-hover:translate-x-0.5 group-hover:text-cyan-200" />
      </div>
      <div className="truncate font-mono text-[10.5px] text-cyan-300/55">{command.commandCode}</div>
      <div className="line-clamp-2 text-[11px] text-cyan-100/65">{command.description || '—'}</div>
      <div className="mt-0.5 flex items-center gap-3 font-mono text-[9.5px] text-cyan-300/45">
        <span className="inline-flex items-center gap-1">
          <Boxes className="size-3" /> {pathCount} PATH
        </span>
        <span>{refCount} PARAM</span>
        {command.requireConfirm ? <span className="text-rose-300/70">CONFIRM</span> : null}
      </div>
    </button>
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

export default MMLCommandsPage
