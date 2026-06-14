import { useMemo, useState } from 'react'
import { Search, RefreshCcw, Terminal, ChevronRight, ListTree, FileCode2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useMMLCommands } from '@core/hooks/api/useMML'
import type { MMLCommand } from '@core/types/mml'
import { StatCard, HudLoading, HudError, HudEmpty } from './_hud'

const OP_COLOR: Record<string, string> = {
  LST: '#5b9eff',
  DSP: '#5b9eff',
  MOD: '#ffaa00',
  ADD: '#00ff88',
  RMV: '#ff2d6f',
  ACT: '#00f0ff',
  DEA: '#a855f7',
  RST: '#ff7a1a',
  CLR: '#ff2d6f',
  UPG: '#a855f7',
}

// ===========================================================================
// CONFIG · 命令行模式（MML 命令台）
// 真实 MML 命令目录（useMMLCommands）→ 按 category 过滤 → 选命令看
// 操作类型 / 参数 / TR-069 路径，HUD 终端风格呈现。
// ===========================================================================
export default function CommandMode() {
  const [keyword, setKeyword] = useState('')
  const [category, setCategory] = useState('')
  const [selectedId, setSelectedId] = useState('')

  const cmdsQ = useMMLCommands({ keyword: keyword.trim() || undefined, page: 1, pageSize: 500 })
  const commands = cmdsQ.data?.items ?? []

  const categories = useMemo(
    () => Array.from(new Set(commands.map((c) => c.category || '未分类'))),
    [commands],
  )

  const rows = category ? commands.filter((c) => (c.category || '未分类') === category) : commands
  const effectiveId = selectedId && rows.some((c) => c.id === selectedId) ? selectedId : rows[0]?.id || ''
  const selected = commands.find((c) => c.id === effectiveId) ?? null

  return (
    <PageShell
      code="F02"
      title="COMMAND MODE · 命令行模式"
      subtitle="MML COMMAND CATALOG"
      isFetching={cmdsQ.isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="命令名 / 命令码"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={() => void cmdsQ.refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-3 gap-3">
        <StatCard label="COMMANDS · 命令" value={cmdsQ.data?.total ?? commands.length} color="#00f0ff" />
        <StatCard label="CATEGORIES · 分类" value={categories.length} color="#a855f7" />
        <StatCard label="MATCHED · 匹配" value={rows.length} color="#5b9eff" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={() => setCategory('')}
          className={`chip ${category === '' ? 'text-[#00f0ff] shadow-[0_0_10px_currentColor]' : 'text-[#6b86b6]'}`}
        >
          全部分类
        </button>
        {categories.map((c) => (
          <button
            key={c}
            type="button"
            onClick={() => setCategory(c)}
            className={`chip ${category === c ? 'text-[#00f0ff] shadow-[0_0_10px_currentColor]' : 'text-[#6b86b6]'}`}
          >
            {c}
          </button>
        ))}
      </div>

      <div className="grid grid-cols-[360px_1fr] gap-3">
        <GlassPanel title="CATALOG · 命令目录" meta={`${rows.length}`} className="min-h-0">
          {cmdsQ.isLoading ? (
            <HudLoading />
          ) : cmdsQ.isError ? (
            <HudError error={cmdsQ.error} />
          ) : rows.length === 0 ? (
            <HudEmpty icon={Terminal} text="无命令" />
          ) : (
            <div className="max-h-[58vh] overflow-auto p-1.5">
              {rows.map((c) => {
                const on = c.id === effectiveId
                const op = c.operationType ?? '—'
                const opColor = OP_COLOR[op] ?? '#6b86b6'
                return (
                  <button
                    key={c.id}
                    type="button"
                    onClick={() => setSelectedId(c.id)}
                    className={`flex w-full items-center gap-2 rounded-sm px-2 py-2 text-left transition-colors ${
                      on ? 'bg-cyan-500/12' : 'hover:bg-cyan-500/6'
                    }`}
                  >
                    <span
                      className="shrink-0 rounded-sm px-1.5 py-0.5 font-mono text-[10px] font-bold"
                      style={{ color: opColor, border: `1px solid ${opColor}55` }}
                    >
                      {op}
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-xs text-cyan-100/90">{c.commandName}</span>
                      <code className="block truncate font-mono text-[10px] text-cyan-300/45">{c.commandCode}</code>
                    </span>
                    <ChevronRight className={`size-3.5 shrink-0 ${on ? 'text-cyan-300' : 'text-cyan-300/30'}`} />
                  </button>
                )
              })}
            </div>
          )}
        </GlassPanel>

        <GlassPanel
          title={selected ? `MML > ${selected.commandCode}` : 'MML CONSOLE'}
          meta={selected?.category}
          className="min-h-0"
        >
          {selected ? (
            <CommandDetail command={selected} />
          ) : (
            <HudEmpty icon={Terminal} text="选择左侧命令查看详情" />
          )}
        </GlassPanel>
      </div>
    </PageShell>
  )
}

function CommandDetail({ command }: { command: MMLCommand }) {
  const op = command.operationType ?? '—'
  const opColor = OP_COLOR[op] ?? '#6b86b6'
  const paths = command.paramPaths ?? []
  return (
    <div className="max-h-[58vh] overflow-auto p-3">
      {/* 命令行回显 */}
      <div className="mb-3 rounded-sm border border-cyan-500/15 bg-[#02030a]/60 px-3 py-2 font-mono text-xs">
        <span className="text-emerald-400">omc@mml</span>
        <span className="text-cyan-300/50"> $ </span>
        <span style={{ color: opColor }}>{op}</span>
        <span className="text-cyan-100"> {command.commandCode}</span>
        {command.targetObject ? <span className="text-cyan-300/55"> :{command.targetObject}</span> : null}
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <span className="font-display text-sm font-bold text-cyan-100">{command.commandName}</span>
        <span className="rounded-sm px-1.5 py-0.5 font-mono text-[10px] font-bold" style={{ color: opColor, border: `1px solid ${opColor}55` }}>
          {op}
        </span>
        {command.requireConfirm ? <StatusBadge status="warning" label="需确认" /> : null}
      </div>

      {command.description ? (
        <p className="mb-3 text-xs leading-relaxed text-cyan-300/70">{command.description}</p>
      ) : null}

      {/* 参数 */}
      <SectionHeader icon={ListTree} title={`PARAMS · 参数 (${command.params?.length ?? 0})`} />
      {command.params && command.params.length > 0 ? (
        <div className="mb-4 overflow-hidden rounded-sm border border-cyan-500/12">
          <div className="grid grid-cols-[1.4fr_90px_1fr_70px] gap-2 border-b border-cyan-500/15 bg-cyan-500/5 px-3 py-1.5 font-mono text-[10px] uppercase tracking-[0.15em] text-cyan-300/45">
            <span>NAME</span>
            <span>TYPE</span>
            <span>DESC</span>
            <span className="text-right">REQ</span>
          </div>
          {command.params.map((p) => (
            <div
              key={p.name}
              className="grid grid-cols-[1.4fr_90px_1fr_70px] items-center gap-2 border-b border-cyan-500/8 px-3 py-1.5"
            >
              <code className="truncate font-mono text-[11px] text-cyan-100/85">{p.name}</code>
              <span className="chip text-[#5b9eff]">{p.type}</span>
              <span className="truncate text-[11px] text-cyan-300/60">{p.description || '—'}</span>
              <span className="text-right">
                {p.required ? <StatusBadge status="warning" label="必填" /> : <span className="font-mono text-[10px] text-cyan-300/40">可选</span>}
              </span>
            </div>
          ))}
        </div>
      ) : (
        <p className="mb-4 font-mono text-[11px] text-cyan-300/45">无参数</p>
      )}

      {/* TR-069 路径 */}
      <SectionHeader icon={FileCode2} title={`TR-069 PATHS · 路径 (${paths.length})`} />
      {paths.length > 0 ? (
        <div className="space-y-1">
          {paths.map((p) => (
            <div
              key={p.path}
              className="flex items-center gap-2 rounded-sm border border-cyan-500/10 px-3 py-1.5"
            >
              <span
                className="size-1.5 shrink-0 rounded-full"
                style={{ background: p.writable ? '#00ff88' : '#525a78' }}
              />
              <code className="min-w-0 flex-1 truncate font-mono text-[11px] text-cyan-200/80">{p.path}</code>
              <span className="shrink-0 font-mono text-[10px] text-cyan-300/45">{p.label}</span>
            </div>
          ))}
        </div>
      ) : (
        <p className="font-mono text-[11px] text-cyan-300/45">无绑定路径</p>
      )}

      {command.notes ? (
        <div className="mt-4 border-l-2 border-amber-400/50 bg-amber-400/5 px-3 py-2 text-[11px] text-amber-200/80">
          {command.notes}
        </div>
      ) : null}
    </div>
  )
}

function SectionHeader({ icon: Icon, title }: { icon: typeof ListTree; title: string }) {
  return (
    <div className="mb-2 flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.18em] text-cyan-300/55">
      <Icon className="size-3.5 text-cyan-300/60" />
      {title}
    </div>
  )
}
