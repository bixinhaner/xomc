import { useMemo, useState } from 'react'
import {
  Search,
  RefreshCcw,
  Loader2,
  Inbox,
  XCircle,
  ChevronRight,
  ChevronDown,
  FolderTree,
  Terminal,
  ShieldCheck,
  Lock,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useGroupTree } from '@core/hooks/api/useMmlConsole'
import type { GroupTreeNode, GroupTreeCommand } from '@core/types/mmlConsole'

// ─────────────────────────────────────────────────────────────
// ADMIN CATALOG · 命令字典管理（real：useGroupTree 分组树）
// v1 路由 mml/admin/catalog（admin-only）—— 左侧分组/命令树 + 右侧命令详情。
// 这里以 HUD 风格只读呈现命令分组结构（增删改在 v1 主皮肤维护）。
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

interface Selected {
  command: GroupTreeCommand
  groupName: string
}

export function MMLAdminCatalogPage() {
  const { data: tree = [], isLoading, isError, error, isFetching, refetch } = useGroupTree(
    undefined,
    'zh-CN'
  )

  const [search, setSearch] = useState('')
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [selected, setSelected] = useState<Selected | null>(null)

  // 顶层分组（一级分组语义：path 不含「.」）。
  const topGroups = useMemo<GroupTreeNode[]>(() => tree.filter((g) => !g.path.includes('.')), [tree])

  const kw = search.trim().toLowerCase()
  const matchGroup = (g: GroupTreeNode): boolean => {
    if (!kw) return true
    if (g.displayName?.toLowerCase().includes(kw) || g.groupCode?.toLowerCase().includes(kw)) return true
    return (g.commands ?? []).some(
      (c) =>
        c.displayName?.toLowerCase().includes(kw) ||
        c.commandCode?.toLowerCase().includes(kw) ||
        c.logicalCode?.toLowerCase().includes(kw)
    )
  }

  const visibleGroups = useMemo(() => topGroups.filter(matchGroup), [topGroups, kw])

  const totalCommands = useMemo(
    () => topGroups.reduce((acc, g) => acc + (g.commands?.length ?? 0), 0),
    [topGroups]
  )
  const protectedCount = useMemo(
    () =>
      topGroups.reduce(
        (acc, g) => acc + (g.commands ?? []).filter((c) => c.catalogProtected).length,
        0
      ),
    [topGroups]
  )

  const toggle = (id: string) =>
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })

  return (
    <PageShell
      code="F06"
      title="ADMIN CATALOG · 命令字典管理"
      subtitle="MAN-MACHINE LANGUAGE · mml_command_groups TREE (READ-ONLY)"
      bare
      isFetching={isFetching}
      toolbar={
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="分组 / 命令名 / 命令码"
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
        <div className="grid grid-cols-3 gap-3">
          <StatCard label="GROUPS" value={topGroups.length} color="#00f0ff" />
          <StatCard label="COMMANDS" value={totalCommands} color="#00ff88" />
          <StatCard label="PROTECTED" value={protectedCount} color="#ffaa00" />
        </div>

        <div className="grid min-h-0 flex-1 grid-cols-[360px_1fr] gap-3">
          {/* 分组/命令树 */}
          <GlassPanel title="CATALOG TREE" meta={`${visibleGroups.length} GROUP`} className="min-h-0 overflow-hidden">
            <div className="h-full overflow-auto p-2">
              {isLoading ? (
                <CenterState>
                  <Loader2 className="size-4 animate-spin" />
                  <span>LOADING TREE…</span>
                </CenterState>
              ) : isError ? (
                <CenterState tone="err">
                  <XCircle className="size-5" />
                  <span>分组树加载失败 · {error instanceof Error ? error.message : '未知错误'}</span>
                </CenterState>
              ) : visibleGroups.length === 0 ? (
                <CenterState>
                  <Inbox className="size-6" />
                  <span>无匹配分组</span>
                </CenterState>
              ) : (
                visibleGroups.map((g) => {
                  const isOpen = expanded.has(g.id) || kw !== ''
                  const cmds = g.commands ?? []
                  const shownCmds = kw
                    ? cmds.filter(
                        (c) =>
                          c.displayName?.toLowerCase().includes(kw) ||
                          c.commandCode?.toLowerCase().includes(kw) ||
                          c.logicalCode?.toLowerCase().includes(kw) ||
                          g.displayName?.toLowerCase().includes(kw)
                      )
                    : cmds
                  return (
                    <div key={g.id} className="mb-1">
                      <button
                        type="button"
                        onClick={() => toggle(g.id)}
                        className="flex w-full items-center gap-1.5 rounded-sm border border-cyan-500/15 bg-cyan-500/5 px-2.5 py-2 text-left hover:border-cyan-400/40"
                      >
                        {isOpen ? (
                          <ChevronDown className="size-3.5 shrink-0 text-cyan-300/60" />
                        ) : (
                          <ChevronRight className="size-3.5 shrink-0 text-cyan-300/60" />
                        )}
                        <FolderTree className="size-3.5 shrink-0 text-cyan-300/55" />
                        <span className="truncate font-display text-sm font-bold text-cyan-100">
                          {g.displayName || g.groupCode}
                        </span>
                        <span className="ml-auto shrink-0 rounded-sm bg-cyan-500/15 px-1.5 py-0.5 font-mono text-[9px] text-cyan-200/80">
                          {cmds.length}
                        </span>
                        {g.catalogProtected ? (
                          <Lock className="size-3 shrink-0 text-amber-300/70" />
                        ) : null}
                      </button>
                      {isOpen && shownCmds.length > 0 ? (
                        <div className="ml-3 mt-0.5 space-y-0.5 border-l border-cyan-500/15 pl-2">
                          {shownCmds.map((c) => {
                            const color = OP_COLOR[c.operationType] ?? '#6b86b6'
                            const active = selected?.command.id === c.id
                            return (
                              <button
                                type="button"
                                key={c.id}
                                onClick={() => setSelected({ command: c, groupName: g.displayName })}
                                className={`flex w-full items-center gap-1.5 rounded-sm px-2 py-1.5 text-left transition-all ${
                                  active
                                    ? 'border border-cyan-400/50 bg-cyan-500/10'
                                    : 'border border-transparent hover:bg-cyan-500/5'
                                }`}
                              >
                                <span
                                  className="shrink-0 rounded-sm border px-1 py-0.5 font-mono text-[8px]"
                                  style={{ color, borderColor: `${color}55` }}
                                >
                                  {c.operationType}
                                </span>
                                <span className="truncate font-mono text-[11px] text-cyan-100/85">
                                  {c.displayName || c.commandCode}
                                </span>
                              </button>
                            )
                          })}
                        </div>
                      ) : null}
                    </div>
                  )
                })
              )}
            </div>
          </GlassPanel>

          {/* 命令详情 */}
          <GlassPanel title="COMMAND DETAIL" meta={selected ? selected.command.commandCode : '—'} className="min-h-0 overflow-hidden">
            <div className="h-full overflow-auto p-4">
              {!selected ? (
                <CenterState>
                  <Terminal className="size-6" />
                  <span>从左侧分组树选择一条命令</span>
                </CenterState>
              ) : (
                <CommandDetail command={selected.command} groupName={selected.groupName} />
              )}
            </div>
          </GlassPanel>
        </div>
      </div>
    </PageShell>
  )
}

function CommandDetail({ command, groupName }: { command: GroupTreeCommand; groupName: string }) {
  const color = OP_COLOR[command.operationType] ?? '#6b86b6'
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <span
          className="rounded-sm border px-2 py-0.5 font-mono text-[11px]"
          style={{ color, borderColor: `${color}55` }}
        >
          {command.operationType}
        </span>
        <span className="font-display text-lg font-bold text-cyan-100">
          {command.displayName || command.commandCode}
        </span>
        {command.requireConfirm ? <StatusBadge status="critical" label="需确认" /> : null}
        {command.catalogProtected ? <StatusBadge status="warning" label="受保护" /> : null}
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        <Meta label="分组" value={groupName} />
        <Meta label="命令码" value={command.commandCode} mono />
        <Meta label="逻辑码" value={command.logicalCode || '—'} mono />
        <Meta label="逻辑名" value={command.logicalName || '—'} />
        <Meta label="RPC 方法" value={command.rpcMethod || '—'} mono />
        <Meta label="目标对象" value={command.targetObject || '—'} mono />
        <Meta label="来源" value={command.source || '—'} />
        {command.supportedPathCount !== undefined ? (
          <Meta label="支持 PATH 数" value={String(command.supportedPathCount)} />
        ) : null}
      </div>

      {command.instanceRangeMeta && command.instanceRangeMeta.length > 0 && (
        <div className="rounded-sm border border-cyan-500/15 bg-cyan-500/4 p-3">
          <div className="mb-2 flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            <ShieldCheck className="size-3.5" /> 实例号范围 metadata · {command.instanceRangeMeta.length}
          </div>
          <div className="space-y-1.5">
            {command.instanceRangeMeta.map((r) => (
              <div key={r.layer} className="flex items-center gap-3 font-mono text-[11px] text-cyan-100/80">
                <span className="rounded-sm bg-cyan-500/15 px-1.5 py-0.5 text-cyan-200/80">L{r.layer}</span>
                <span className="text-cyan-300/85">{r.rangeExpr}</span>
                {r.dynamic ? <span className="text-amber-300/70">dynamic{r.nSource ? ` · ${r.nSource}` : ''}</span> : null}
                {r.description ? <span className="truncate text-cyan-300/50">{r.description}</span> : null}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function StatCard({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: color }}
    >
      <div className="truncate font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/60">{label}</div>
      <div className="font-display text-2xl font-bold leading-tight" style={{ color, textShadow: `0 0 8px ${color}` }}>
        {value}
      </div>
    </div>
  )
}

function Meta({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="min-w-0">
      <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/50">{label}</div>
      <div className={`truncate text-xs text-cyan-100/90 ${mono ? 'font-mono' : ''}`}>{value}</div>
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

export default MMLAdminCatalogPage
