import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Loader2, XCircle, Inbox, Boxes, KeyRound } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useMMLTemplates } from '@core/hooks/api/useMML'

// ─────────────────────────────────────────────────────────────
// PRIVATE COMMAND DETAIL · 单条私有命令（real：从私有模板列表按 :id 命中）
// 拉一页较大的私有模板（与列表共享 endpoint），按 :id 过滤命中 → 保证真实加载。
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

export function MMLPrivateCommandDetailPage() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data, isLoading, isError, error } = useMMLTemplates({
    templateScope: 'private',
    page: 1,
    pageSize: 200,
  })

  const command = useMemo(() => (data?.items ?? []).find((c) => c.id === id) ?? null, [data, id])

  const back = (
    <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/mml/private-command')}>
      PRIVATE
    </NeonButton>
  )

  if (isLoading) {
    return (
      <PageShell code="F06" title="PRIVATE COMMAND · 私有命令详情" subtitle={id} toolbar={back}>
        <CenterState>
          <Loader2 className="size-4 animate-spin" />
          <span>LOADING COMMAND…</span>
        </CenterState>
      </PageShell>
    )
  }

  if (isError) {
    return (
      <PageShell code="F06" title="PRIVATE COMMAND · 私有命令详情" subtitle={id} toolbar={back}>
        <CenterState tone="err">
          <XCircle className="size-5" />
          <span>加载失败 · {error instanceof Error ? error.message : '未知错误'}</span>
        </CenterState>
      </PageShell>
    )
  }

  if (!command) {
    return (
      <PageShell code="F06" title="PRIVATE COMMAND · 私有命令详情" subtitle={id} toolbar={back}>
        <CenterState>
          <Inbox className="size-6" />
          <span>私有命令不存在或无权限查看 · {id}</span>
        </CenterState>
      </PageShell>
    )
  }

  const color = OP_COLOR[command.operationType] ?? '#6b86b6'
  const paramEntries = Object.entries(command.parameters ?? {})

  return (
    <PageShell
      code="F06"
      title={`PRIVATE · ${command.commandCode}`}
      subtitle={command.commandName}
      toolbar={back}
    >
      <div className="flex flex-col gap-3">
        <GlassPanel title="META · 命令元信息">
          <div className="grid grid-cols-2 gap-3 p-4 sm:grid-cols-3 lg:grid-cols-4">
            <Meta label="命令名" value={command.commandName} />
            <Meta label="命令码" value={command.commandCode} mono />
            <div className="min-w-0">
              <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/50">操作类型</div>
              <span
                className="mt-0.5 inline-block rounded-sm border px-2 py-0.5 font-mono text-[11px]"
                style={{ color, borderColor: `${color}55` }}
              >
                {command.operationType}
              </span>
            </div>
            <div className="min-w-0">
              <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/50">作用域</div>
              <div className="mt-0.5">
                <StatusBadge
                  status={command.commandScope === 'private' ? 'minor' : 'ok'}
                  label={command.commandScope === 'private' ? '私有' : '公有'}
                />
              </div>
            </div>
            <Meta label="分类组" value={command.categoryGroup || '—'} />
            <Meta label="创建人" value={command.creator || '—'} />
            <Meta label="创建时间" value={formatTime(command.createdAt)} />
            <Meta label="更新时间" value={formatTime(command.updatedAt)} />
          </div>
          {command.description ? (
            <div className="border-t border-cyan-500/15 px-4 py-3 text-sm leading-relaxed text-cyan-100/85">
              {command.description}
            </div>
          ) : null}
        </GlassPanel>

        <GlassPanel title="TR-069 PATHS · 绑定路径" meta={`${command.paramPaths?.length ?? 0}`}>
          <div className="p-3">
            {!command.paramPaths || command.paramPaths.length === 0 ? (
              <CenterState>
                <Inbox className="size-5" />
                <span>无绑定 PATH</span>
              </CenterState>
            ) : (
              <div className="space-y-1.5">
                {command.paramPaths.map((p, i) => (
                  <div
                    key={`${p}-${i}`}
                    className="flex items-center gap-2 rounded-sm border border-cyan-500/12 bg-[#03050d]/50 px-3 py-2"
                  >
                    <Boxes className="size-3.5 shrink-0 text-cyan-300/45" />
                    <span className="truncate font-mono text-[11px] text-cyan-100/90">{p}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </GlassPanel>

        {paramEntries.length > 0 && (
          <GlassPanel title="PARAMETERS · 默认参数值" meta={`${paramEntries.length}`}>
            <div className="overflow-auto">
              <div className="sticky top-0 grid grid-cols-[1.4fr_2fr] gap-3 border-b border-cyan-500/20 bg-[#03050d]/85 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/60">
                <span>参数 / PATH</span>
                <span>值</span>
              </div>
              {paramEntries.map(([k, v]) => (
                <div
                  key={k}
                  className="grid grid-cols-[1.4fr_2fr] items-center gap-3 border-b border-cyan-500/8 px-3 py-2"
                >
                  <span className="truncate font-mono text-[11px] text-cyan-100/90">
                    <KeyRound className="mr-1 inline size-3 text-cyan-300/40" />
                    {k}
                  </span>
                  <span className="truncate font-mono text-[11px] text-emerald-300/85">{String(v)}</span>
                </div>
              ))}
            </div>
          </GlassPanel>
        )}
      </div>
    </PageShell>
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

export default MMLPrivateCommandDetailPage
