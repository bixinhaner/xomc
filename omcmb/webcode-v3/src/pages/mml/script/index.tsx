import { useMemo, useState } from 'react'
import {
  Search,
  RefreshCcw,
  Loader2,
  Inbox,
  XCircle,
  X,
  ScrollText,
  Tag as TagIcon,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { Sparkline } from '@/components/viz/Sparkline'
import { formatTime } from '@/lib/format'
import { useMMLScripts } from '@core/hooks/api/useMML'
import type { MMLScript, MMLScriptStatus } from '@core/types/mml'

// ─────────────────────────────────────────────────────────────
// SCRIPT VAULT · mml_scripts 脚本库（real：useMMLScripts）
// v1 路由 mml/script（ScriptTask）—— 脚本库列表 + 详情。
// ─────────────────────────────────────────────────────────────

const STATUS_TONE: Record<MMLScriptStatus, string> = {
  active: 'ok',
  archived: 'off',
  pending: 'off',
  running: 'warning',
  paused: 'minor',
  completed: 'ok',
  failed: 'critical',
  cancelled: 'offline',
}

function scriptTone(status: string): string {
  return STATUS_TONE[status as MMLScriptStatus] ?? 'unknown'
}

const PAGE_SIZE = 20

export function MMLScriptPage() {
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [viewing, setViewing] = useState<MMLScript | null>(null)

  const params = useMemo(
    () => ({ page, pageSize: PAGE_SIZE, ...(search.trim() ? { search: search.trim() } : {}) }),
    [page, search]
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useMMLScripts(params)
  const scripts = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  // 当前页状态分桶（态势带）。
  const statusCounts = useMemo(() => {
    const m = new Map<string, number>()
    for (const s of scripts) m.set(s.status, (m.get(s.status) ?? 0) + 1)
    return m
  }, [scripts])

  return (
    <PageShell
      code="F06"
      title="SCRIPT VAULT · 脚本库"
      subtitle="MAN-MACHINE LANGUAGE · mml_scripts BATCH SCRIPTS"
      bare
      isFetching={isFetching}
      toolbar={
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
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
        </div>
      }
    >
      <div className="flex h-full flex-col gap-3">
        <div className="grid grid-cols-3 gap-3 md:grid-cols-5">
          <StatCard label="TOTAL" value={total} color="#00f0ff" trend={[4, 6, 5, 8, 7, 9, 11]} />
          <StatCard label="活跃" value={statusCounts.get('active') ?? 0} color="#00ff88" />
          <StatCard label="执行中" value={statusCounts.get('running') ?? 0} color="#ffaa00" />
          <StatCard label="已完成" value={statusCounts.get('completed') ?? 0} color="#00ff88" />
          <StatCard label="失败" value={statusCounts.get('failed') ?? 0} color="#ff2d6f" />
        </div>

        <GlassPanel
          title="SCRIPT VAULT · 脚本库"
          meta={`PAGE ${page}/${totalPages}`}
          className="min-h-0 flex-1 overflow-hidden"
        >
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
                    <div className="flex items-center gap-1.5">
                      <ScrollText className="size-3.5 shrink-0 text-cyan-300/45" />
                      <span className="truncate font-display text-sm font-bold text-cyan-100">
                        {s.scriptName}
                      </span>
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
                  <div className="font-mono text-[11px] text-cyan-300/70">{formatTime(s.updateTime)}</div>
                </button>
              ))
            )}
          </div>
        </GlassPanel>

        <Pager page={page} totalPages={totalPages} total={total} onPage={setPage} />
      </div>

      {viewing && <ScriptDetailDrawer script={viewing} onClose={() => setViewing(null)} />}
    </PageShell>
  )
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
            <div className="truncate font-display text-base font-bold text-cyan-100">{script.scriptName}</div>
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
          <div>
            <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/50">状态</div>
            <div className="mt-0.5">
              <StatusBadge status={scriptTone(script.status)} label={script.status} />
            </div>
          </div>
          <Meta label="类型" value={script.type} />
          <Meta label="创建人" value={script.creator || '—'} />
          <Meta label="进度" value={`${script.progress ?? 0}%`} />
          <Meta label="创建时间" value={formatTime(script.createTime)} />
          <Meta label="更新时间" value={formatTime(script.updateTime)} />
        </div>

        {script.description ? (
          <div className="border-b border-cyan-500/15 px-4 py-3">
            <div className="mb-1 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">描述</div>
            <div className="text-sm text-cyan-100/85">{script.description}</div>
          </div>
        ) : null}

        {script.tags && script.tags.length > 0 ? (
          <div className="flex flex-wrap items-center gap-1.5 border-b border-cyan-500/15 px-4 py-3">
            <TagIcon className="size-3.5 text-cyan-300/45" />
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
          <div className="mb-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">脚本内容</div>
          <pre className="overflow-auto whitespace-pre-wrap rounded-sm border border-emerald-500/20 bg-black/40 p-3 font-mono text-[12px] leading-relaxed text-emerald-300/90">
            {script.content || '(空)'}
          </pre>
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

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/50">{label}</div>
      <div className="truncate text-xs text-cyan-100/90">{value}</div>
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

export default MMLScriptPage
