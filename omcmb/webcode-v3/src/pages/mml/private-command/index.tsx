import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  RefreshCcw,
  Loader2,
  Inbox,
  XCircle,
  ChevronRight,
  Lock,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { Sparkline } from '@/components/viz/Sparkline'
import { formatTime } from '@/lib/format'
import { useMMLTemplates } from '@core/hooks/api/useMML'

// ─────────────────────────────────────────────────────────────
// PRIVATE COMMANDS · mml_custom_command (scope=private) 私有命令
// v1 路由 mml/private-command（PrivateCommand）—— 后端 RBAC 自动过滤
// （creator self-fallback OR group-share）。这里只读列表 + 行→详情 navigate。
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

const PAGE_SIZE = 20

export function MMLPrivateCommandPage() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [commandCode, setCommandCode] = useState('')

  const params = useMemo(
    () => ({
      templateScope: 'private' as const,
      page,
      pageSize: PAGE_SIZE,
      ...(commandCode.trim() ? { commandCode: commandCode.trim() } : {}),
    }),
    [page, commandCode]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useMMLTemplates(params)
  const items = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const opCounts = useMemo(() => {
    const m = new Map<string, number>()
    for (const c of items) m.set(c.operationType, (m.get(c.operationType) ?? 0) + 1)
    return m
  }, [items])

  return (
    <PageShell
      code="F06"
      title="PRIVATE COMMANDS · 私有命令"
      subtitle="MAN-MACHINE LANGUAGE · mml_custom_command (PRIVATE SCOPE)"
      bare
      isFetching={isFetching}
      toolbar={
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="命令码"
              value={commandCode}
              onChange={(e) => {
                setCommandCode(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            {isFetching ? 'SYNC…' : 'REFRESH'}
          </NeonButton>
        </div>
      }
    >
      <div className="flex h-full flex-col gap-3">
        <div className="grid grid-cols-3 gap-3 md:grid-cols-5">
          <StatCard label="TOTAL" value={total} color="#00f0ff" trend={[3, 5, 4, 6, 5, 7, 8]} />
          <StatCard label="LST" value={opCounts.get('LST') ?? 0} color={OP_COLOR.LST} />
          <StatCard label="MOD" value={opCounts.get('MOD') ?? 0} color={OP_COLOR.MOD} />
          <StatCard label="ADD" value={opCounts.get('ADD') ?? 0} color={OP_COLOR.ADD} />
          <StatCard label="RMV" value={opCounts.get('RMV') ?? 0} color={OP_COLOR.RMV} />
        </div>

        <GlassPanel
          title="PRIVATE COMMANDS · 私有命令"
          meta={`PAGE ${page}/${totalPages}`}
          className="min-h-0 flex-1 overflow-hidden"
        >
          <div className="h-full overflow-auto">
            <div className="sticky top-0 z-10 grid grid-cols-[1.6fr_2fr_0.8fr_2fr_1fr_1.4fr_40px] gap-3 border-b border-cyan-500/20 bg-[#03050d]/85 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/60 backdrop-blur">
              <span>命令名</span>
              <span>命令码</span>
              <span>操作</span>
              <span>描述</span>
              <span>创建人</span>
              <span>更新时间</span>
              <span />
            </div>

            {isLoading ? (
              <CenterState>
                <Loader2 className="size-4 animate-spin" />
                <span>SYNCING PRIVATE COMMANDS…</span>
              </CenterState>
            ) : isError ? (
              <CenterState tone="err">
                <XCircle className="size-5" />
                <span>FAILURE · {error instanceof Error ? error.message : '未知错误'}</span>
              </CenterState>
            ) : items.length === 0 ? (
              <CenterState>
                <Inbox className="size-6" />
                <span>无私有命令</span>
              </CenterState>
            ) : (
              items.map((c) => {
                const color = OP_COLOR[c.operationType] ?? '#6b86b6'
                return (
                  <button
                    type="button"
                    key={c.id}
                    onClick={() => navigate(`/mml/private-command/${c.id}`)}
                    className="group grid w-full grid-cols-[1.6fr_2fr_0.8fr_2fr_1fr_1.4fr_40px] items-center gap-3 border-b border-cyan-500/8 px-3 py-2.5 text-left hover:bg-cyan-500/5"
                  >
                    <div className="flex min-w-0 items-center gap-1.5">
                      <Lock className="size-3 shrink-0 text-cyan-300/40" />
                      <span className="truncate font-display text-sm font-bold text-cyan-100">
                        {c.commandName}
                      </span>
                    </div>
                    <span className="truncate font-mono text-[11px] text-cyan-300/70">{c.commandCode}</span>
                    <span>
                      <span
                        className="rounded-sm border px-1.5 py-0.5 font-mono text-[9px]"
                        style={{ color, borderColor: `${color}55` }}
                      >
                        {c.operationType}
                      </span>
                    </span>
                    <span className="truncate text-xs text-cyan-100/70">{c.description || '—'}</span>
                    <span className="truncate text-xs text-cyan-100/80">{c.creator || '—'}</span>
                    <span className="font-mono text-[11px] text-cyan-300/70">{formatTime(c.updatedAt)}</span>
                    <ChevronRight className="size-3.5 text-cyan-300/40 transition-transform group-hover:translate-x-0.5 group-hover:text-cyan-200" />
                  </button>
                )
              })
            )}
          </div>
        </GlassPanel>

        <Pager page={page} totalPages={totalPages} total={total} onPage={setPage} />
      </div>
    </PageShell>
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

export default MMLPrivateCommandPage
