import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  ArrowLeft,
  Loader2,
  XCircle,
  Inbox,
  ShieldAlert,
  Boxes,
  KeyRound,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useAllMMLCommands } from '@core/hooks/api/useMML'
import type { MMLParamRef, ParamPath } from '@core/types/mml'

// ─────────────────────────────────────────────────────────────
// COMMAND DETAIL · 单命令详情（real：从 useAllMMLCommands 列表按 :id 命中）
// 列表与详情共享同一 query（['mml','commands','all']），保证带参路由可加载真实数据。
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

export function MMLCommandDetailPage() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: commands = [], isLoading, isError, error } = useAllMMLCommands()

  const command = useMemo(() => commands.find((c) => c.id === id) ?? null, [commands, id])

  const back = (
    <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/mml/commands')}>
      CATALOG
    </NeonButton>
  )

  if (isLoading) {
    return (
      <PageShell code="F06" title="COMMAND DETAIL · 命令详情" subtitle={id} toolbar={back}>
        <CenterState>
          <Loader2 className="size-4 animate-spin" />
          <span>LOADING COMMAND…</span>
        </CenterState>
      </PageShell>
    )
  }

  if (isError) {
    return (
      <PageShell code="F06" title="COMMAND DETAIL · 命令详情" subtitle={id} toolbar={back}>
        <CenterState tone="err">
          <XCircle className="size-5" />
          <span>命令字典加载失败 · {error instanceof Error ? error.message : '未知错误'}</span>
        </CenterState>
      </PageShell>
    )
  }

  if (!command) {
    return (
      <PageShell code="F06" title="COMMAND DETAIL · 命令详情" subtitle={id} toolbar={back}>
        <CenterState>
          <Inbox className="size-6" />
          <span>命令不存在或已下线 · {id}</span>
        </CenterState>
      </PageShell>
    )
  }

  const op = command.operationType
  const color = (op && OP_COLOR[op]) || '#6b86b6'
  const paths: ParamPath[] = command.paramPaths ?? []
  const refs: MMLParamRef[] = command.paramRefs ?? []

  return (
    <PageShell
      code="F06"
      title={`COMMAND · ${command.commandCode}`}
      subtitle={command.commandName}
      toolbar={back}
    >
      <div className="flex flex-col gap-3">
        {/* 元信息 */}
        <GlassPanel title="META · 命令元信息">
          <div className="grid grid-cols-2 gap-3 p-4 sm:grid-cols-3 lg:grid-cols-4">
            <Meta label="命令码" value={command.commandCode} mono />
            <Meta label="命令名" value={command.commandName} />
            <Meta label="分类" value={command.category || '—'} />
            <div className="min-w-0">
              <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/50">操作类型</div>
              {op ? (
                <span
                  className="mt-0.5 inline-block rounded-sm border px-2 py-0.5 font-mono text-[11px]"
                  style={{ color, borderColor: `${color}55` }}
                >
                  {op}
                </span>
              ) : (
                <div className="truncate text-xs text-cyan-100/90">—</div>
              )}
            </div>
            <Meta label="目标对象" value={command.targetObject || '—'} mono />
            <Meta label="PATH 数" value={String(paths.length)} />
            <Meta label="参数数" value={String(refs.length || command.params?.length || 0)} />
            <div className="min-w-0">
              <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/50">二次确认</div>
              <div className="mt-0.5">
                <StatusBadge
                  status={command.requireConfirm ? 'critical' : 'off'}
                  label={command.requireConfirm ? '需确认' : '常规'}
                />
              </div>
            </div>
          </div>

          {command.requireConfirm ? (
            <div className="mx-4 mb-4 flex items-start gap-2 rounded-sm border border-rose-500/40 bg-rose-500/8 px-3 py-2 font-mono text-[11px] text-rose-300">
              <ShieldAlert className="mt-0.5 size-3.5 shrink-0" />
              <span>危险命令 · 执行前需二次确认（require_confirm=true）。</span>
            </div>
          ) : null}
        </GlassPanel>

        {/* 描述 / 帮助 */}
        {(command.description || command.helpDoc || command.notes) && (
          <GlassPanel title="DESCRIPTION · 说明">
            <div className="space-y-3 p-4">
              {command.description ? (
                <div className="text-sm leading-relaxed text-cyan-100/85">{command.description}</div>
              ) : null}
              {command.helpDoc ? (
                <pre className="overflow-auto whitespace-pre-wrap rounded-sm border border-cyan-500/15 bg-black/40 p-3 font-mono text-[11px] leading-relaxed text-cyan-200/80">
                  {command.helpDoc}
                </pre>
              ) : null}
              {command.notes ? (
                <div className="font-mono text-[11px] text-cyan-300/60">备注：{command.notes}</div>
              ) : null}
            </div>
          </GlassPanel>
        )}

        {/* 绑定 TR-069 PATH */}
        <GlassPanel title="TR-069 PATHS · 绑定路径" meta={`${paths.length}`}>
          <div className="p-3">
            {paths.length === 0 ? (
              <CenterState>
                <Inbox className="size-5" />
                <span>该命令无绑定 PATH</span>
              </CenterState>
            ) : (
              <div className="space-y-1.5">
                {paths.map((p, i) => (
                  <div
                    key={`${p.path}-${i}`}
                    className="flex items-center justify-between gap-3 rounded-sm border border-cyan-500/12 bg-[#03050d]/50 px-3 py-2"
                  >
                    <div className="flex min-w-0 items-center gap-2">
                      <Boxes className="size-3.5 shrink-0 text-cyan-300/45" />
                      <span className="truncate font-mono text-[11px] text-cyan-100/90">{p.path}</span>
                    </div>
                    <div className="flex shrink-0 items-center gap-2">
                      {p.label ? (
                        <span className="text-[10.5px] text-cyan-300/55">{p.label}</span>
                      ) : null}
                      <StatusBadge status={p.writable ? 'warning' : 'off'} label={p.writable ? '可写' : '只读'} />
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </GlassPanel>

        {/* 参数引用 */}
        {refs.length > 0 && (
          <GlassPanel title="PARAM REFS · 参数引用" meta={`${refs.length}`}>
            <div className="overflow-auto">
              <div className="sticky top-0 grid grid-cols-[1.4fr_1.4fr_2fr_0.8fr_0.8fr] gap-3 border-b border-cyan-500/20 bg-[#03050d]/85 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/60">
                <span>参数码</span>
                <span>中文名</span>
                <span>TR-069 路径</span>
                <span>类型</span>
                <span>可写</span>
              </div>
              {refs.map((r) => (
                <div
                  key={r.id}
                  className="grid grid-cols-[1.4fr_1.4fr_2fr_0.8fr_0.8fr] items-center gap-3 border-b border-cyan-500/8 px-3 py-2"
                >
                  <span className="truncate font-mono text-[11px] text-cyan-100/90">
                    <KeyRound className="mr-1 inline size-3 text-cyan-300/40" />
                    {r.paramCode}
                  </span>
                  <span className="truncate text-xs text-cyan-100/75">{r.paramNameZh || '—'}</span>
                  <span className="truncate font-mono text-[10.5px] text-cyan-300/65">{r.tr069Path}</span>
                  <span className="font-mono text-[10.5px] text-cyan-300/70">{r.valueType}</span>
                  <span>
                    <StatusBadge status={r.isWritable ? 'warning' : 'off'} label={r.isWritable ? '可写' : '只读'} />
                  </span>
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

export default MMLCommandDetailPage
