import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  Loader2,
  Inbox,
  XCircle,
  CheckCircle2,
  ChevronRight,
  ChevronDown,
  FolderTree,
  Cpu,
  Rocket,
  Terminal,
  X,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useDeviceList } from '@core/hooks/api/useDevices'
import { useGroupTree, useCommandSubFields, useExecuteStatementsStructured } from '@core/hooks/api/useMmlConsole'
import { useUserStore } from '@core/store/userStore'
import type { Device } from '@core/types/device'
import type {
  GroupTreeNode,
  GroupTreeCommand,
  SubFieldDef,
  StructuredStatement,
  ConsoleSupportedOp,
} from '@core/types/mmlConsole'

// ─────────────────────────────────────────────────────────────
// MML CONSOLE · 结构化执行控制台（real）
// v1 路由 mml/console（Console）—— 三步：选设备 → 选命令 → 配置参数并执行。
//   设备列表 useDeviceList；命令树 useGroupTree(productClass)；
//   sub-fields useCommandSubFields；执行 useExecuteStatementsStructured。
// 执行成功后导航到任务记录页查看推进状态。
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
}

const STRUCTURED_OPS: ConsoleSupportedOp[] = ['LST', 'MOD', 'ADD', 'RMV']

export function MMLConsolePage() {
  const navigate = useNavigate()
  const creator = useUserStore((s) => s.currentUser?.username) ?? ''

  // 设备选择 ---------------------------------------------------
  const [deviceKw, setDeviceKw] = useState('')
  const [selectedSns, setSelectedSns] = useState<string[]>([])
  const { data: devicePage, isLoading: devLoading, isError: devError } = useDeviceList({
    page: 1,
    pageSize: 200,
  })
  const allDevices = devicePage?.items ?? []
  const kw = deviceKw.trim().toLowerCase()
  const devices = kw
    ? allDevices.filter((d) => d.sn?.toLowerCase().includes(kw) || d.name?.toLowerCase().includes(kw))
    : allDevices

  // 首个选中设备的 product_class 作为命令树过滤键（与 v1 一致）。
  const productClass = useMemo(() => {
    const first = allDevices.find((d) => selectedSns.includes(d.sn))
    return first?.productClass || undefined
  }, [allDevices, selectedSns])

  // 命令树 -----------------------------------------------------
  const [cmdKw, setCmdKw] = useState('')
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [command, setCommand] = useState<GroupTreeCommand | null>(null)
  const { data: tree = [], isLoading: treeLoading, isError: treeError } = useGroupTree(
    undefined,
    'zh-CN',
    productClass
  )
  const topGroups = useMemo<GroupTreeNode[]>(() => tree.filter((g) => !g.path.includes('.')), [tree])

  // sub-fields（命令选中后）------------------------------------
  const deviceKey = selectedSns[0]
  const { data: subFields = [], isLoading: sfLoading } = useCommandSubFields(
    command?.id,
    'zh-CN',
    deviceKey
  )
  const [checkedPaths, setCheckedPaths] = useState<Set<string>>(new Set())
  const [values, setValues] = useState<Record<string, string>>({})

  const exec = useExecuteStatementsStructured()
  const [submitErr, setSubmitErr] = useState('')

  const op = (command?.operationType as ConsoleSupportedOp | undefined) ?? undefined
  const isStructured = op ? STRUCTURED_OPS.includes(op) : false
  const isMod = op === 'MOD' || op === 'ADD'

  const toggleDevice = (sn: string) =>
    setSelectedSns((prev) => (prev.includes(sn) ? prev.filter((x) => x !== sn) : [...prev, sn]))

  const selectCommand = (c: GroupTreeCommand) => {
    setCommand(c)
    setCheckedPaths(new Set())
    setValues({})
    setSubmitErr('')
  }

  const togglePath = (path: string) =>
    setCheckedPaths((prev) => {
      const next = new Set(prev)
      if (next.has(path)) next.delete(path)
      else next.add(path)
      return next
    })

  const canExecute =
    selectedSns.length > 0 && !!command && isStructured && checkedPaths.size > 0 && !exec.isPending

  const handleExecute = () => {
    if (!command || !canExecute || !op) return
    setSubmitErr('')
    const paths = [...checkedPaths]
    const stmt: StructuredStatement = {
      commandId: command.id,
      operationType: op,
      commandCode: command.commandCode,
      paths,
      ...(isMod
        ? {
            values: Object.fromEntries(
              paths.filter((p) => values[p] !== undefined).map((p) => [p, values[p]])
            ),
          }
        : {}),
    }
    exec.mutate(
      {
        deviceSns: selectedSns,
        statements: [stmt],
        taskName: `${command.commandCode}_${new Date().toISOString().slice(0, 19)}`,
        creator,
        executeType: 'immediate',
      },
      {
        onSuccess: () => navigate('/mml/task-records'),
        onError: (e) => setSubmitErr(e instanceof Error ? e.message : '执行失败'),
      }
    )
  }

  const toggleGroup = (id: string) =>
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })

  const cmdFilter = cmdKw.trim().toLowerCase()

  return (
    <PageShell
      code="F06"
      title="MML CONSOLE · 结构化控制台"
      subtitle="MAN-MACHINE LANGUAGE · DEVICE → COMMAND → DISPATCH"
      bare
      toolbar={
        <div className="flex items-center gap-2 font-mono text-[11px] text-cyan-300/60">
          <span className="chip" style={{ color: '#00f0ff' }}>
            <Cpu className="size-3" /> {selectedSns.length} DEV
          </span>
          {productClass ? (
            <span className="chip" style={{ color: '#a855f7' }}>
              {productClass}
            </span>
          ) : null}
        </div>
      }
    >
      <div className="grid h-full grid-cols-[300px_320px_1fr] gap-3">
        {/* STEP 1 · 设备 */}
        <GlassPanel title="① TARGET DEVICES" meta={`${selectedSns.length}/${allDevices.length}`} className="min-h-0 overflow-hidden">
          <div className="flex h-full flex-col">
            <div className="relative p-2.5">
              <Search className="pointer-events-none absolute left-5 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
              <input
                className="neon-input w-full pl-9"
                placeholder="SN / 名称"
                value={deviceKw}
                onChange={(e) => setDeviceKw(e.target.value)}
              />
            </div>
            <div className="min-h-0 flex-1 overflow-auto px-2 pb-2">
              {devLoading ? (
                <CenterState>
                  <Loader2 className="size-4 animate-spin" />
                  <span>LOADING…</span>
                </CenterState>
              ) : devError ? (
                <CenterState tone="err">
                  <XCircle className="size-5" />
                  <span>设备加载失败</span>
                </CenterState>
              ) : devices.length === 0 ? (
                <CenterState>
                  <Inbox className="size-5" />
                  <span>无设备</span>
                </CenterState>
              ) : (
                devices.map((d: Device) => {
                  const on = selectedSns.includes(d.sn)
                  return (
                    <button
                      type="button"
                      key={d.id}
                      onClick={() => toggleDevice(d.sn)}
                      className={`mb-1 flex w-full items-center gap-2 rounded-sm border px-2.5 py-2 text-left transition-all ${
                        on
                          ? 'border-cyan-400/60 bg-cyan-500/15'
                          : 'border-cyan-500/12 bg-cyan-500/5 hover:border-cyan-400/40'
                      }`}
                    >
                      <span
                        className={`flex size-4 shrink-0 items-center justify-center rounded-sm border ${
                          on ? 'border-cyan-400 bg-cyan-400/20' : 'border-cyan-500/30'
                        }`}
                      >
                        {on ? <CheckCircle2 className="size-3 text-cyan-200" /> : null}
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className="block truncate font-mono text-[11px] text-cyan-100">{d.sn}</span>
                        <span className="block truncate text-[10px] text-cyan-300/50">{d.name || '—'}</span>
                      </span>
                      <StatusBadge status={d.isOnline ? 'online' : 'offline'} label={d.isOnline ? 'ON' : 'OFF'} />
                    </button>
                  )
                })
              )}
            </div>
          </div>
        </GlassPanel>

        {/* STEP 2 · 命令树 */}
        <GlassPanel
          title="② COMMAND"
          meta={productClass ? productClass : 'ALL'}
          className="min-h-0 overflow-hidden"
        >
          <div className="flex h-full flex-col">
            <div className="relative p-2.5">
              <Search className="pointer-events-none absolute left-5 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
              <input
                className="neon-input w-full pl-9"
                placeholder="命令名 / 命令码"
                value={cmdKw}
                onChange={(e) => setCmdKw(e.target.value)}
              />
            </div>
            <div className="min-h-0 flex-1 overflow-auto px-2 pb-2">
              {treeLoading ? (
                <CenterState>
                  <Loader2 className="size-4 animate-spin" />
                  <span>LOADING TREE…</span>
                </CenterState>
              ) : treeError ? (
                <CenterState tone="err">
                  <XCircle className="size-5" />
                  <span>命令树加载失败</span>
                </CenterState>
              ) : topGroups.length === 0 ? (
                <CenterState>
                  <Inbox className="size-5" />
                  <span>无命令</span>
                </CenterState>
              ) : (
                topGroups.map((g) => {
                  const cmds = (g.commands ?? []).filter(
                    (c) =>
                      !cmdFilter ||
                      c.displayName?.toLowerCase().includes(cmdFilter) ||
                      c.commandCode?.toLowerCase().includes(cmdFilter) ||
                      g.displayName?.toLowerCase().includes(cmdFilter)
                  )
                  if (cmdFilter && cmds.length === 0) return null
                  const isOpen = expanded.has(g.id) || cmdFilter !== ''
                  return (
                    <div key={g.id} className="mb-1">
                      <button
                        type="button"
                        onClick={() => toggleGroup(g.id)}
                        className="flex w-full items-center gap-1.5 rounded-sm border border-cyan-500/15 bg-cyan-500/5 px-2 py-1.5 text-left hover:border-cyan-400/40"
                      >
                        {isOpen ? (
                          <ChevronDown className="size-3.5 shrink-0 text-cyan-300/60" />
                        ) : (
                          <ChevronRight className="size-3.5 shrink-0 text-cyan-300/60" />
                        )}
                        <FolderTree className="size-3.5 shrink-0 text-cyan-300/55" />
                        <span className="truncate font-display text-[12px] font-bold text-cyan-100">
                          {g.displayName || g.groupCode}
                        </span>
                        <span className="ml-auto shrink-0 font-mono text-[9px] text-cyan-300/45">
                          {g.commands?.length ?? 0}
                        </span>
                      </button>
                      {isOpen ? (
                        <div className="ml-3 mt-0.5 space-y-0.5 border-l border-cyan-500/15 pl-2">
                          {cmds.map((c) => {
                            const color = OP_COLOR[c.operationType] ?? '#6b86b6'
                            const active = command?.id === c.id
                            return (
                              <button
                                type="button"
                                key={c.id}
                                onClick={() => selectCommand(c)}
                                className={`flex w-full items-center gap-1.5 rounded-sm px-2 py-1.5 text-left transition-all ${
                                  active ? 'border border-cyan-400/50 bg-cyan-500/10' : 'border border-transparent hover:bg-cyan-500/5'
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
          </div>
        </GlassPanel>

        {/* STEP 3 · 参数 + 执行 */}
        <GlassPanel
          title="③ DISPATCH"
          meta={command ? command.commandCode : '—'}
          className="min-h-0 overflow-hidden"
        >
          <div className="flex h-full flex-col">
            {!command ? (
              <CenterState>
                <Terminal className="size-6" />
                <span>选择命令以配置参数</span>
              </CenterState>
            ) : (
              <>
                <div className="flex items-center justify-between gap-2 border-b border-cyan-500/15 px-3 py-2">
                  <div className="flex items-center gap-2">
                    <span
                      className="rounded-sm border px-1.5 py-0.5 font-mono text-[10px]"
                      style={{
                        color: OP_COLOR[command.operationType] ?? '#6b86b6',
                        borderColor: `${OP_COLOR[command.operationType] ?? '#6b86b6'}55`,
                      }}
                    >
                      {command.operationType}
                    </span>
                    <span className="truncate font-display text-sm font-bold text-cyan-100">
                      {command.displayName || command.commandCode}
                    </span>
                  </div>
                  <button
                    type="button"
                    onClick={() => setCommand(null)}
                    className="rounded-sm border border-cyan-500/25 p-1 text-cyan-300/60 hover:text-cyan-100"
                  >
                    <X className="size-3.5" />
                  </button>
                </div>

                <div className="min-h-0 flex-1 overflow-auto p-3">
                  {!isStructured ? (
                    <div className="rounded-sm border border-amber-500/40 bg-amber-500/8 px-3 py-2 font-mono text-[11px] text-amber-300">
                      该命令操作类型 {command.operationType} 不走结构化执行通道（仅 LST/MOD/ADD/RMV 支持）；请在 v1 主皮肤执行。
                    </div>
                  ) : sfLoading ? (
                    <CenterState>
                      <Loader2 className="size-4 animate-spin" />
                      <span>LOADING SUB-FIELDS…</span>
                    </CenterState>
                  ) : subFields.length === 0 ? (
                    <CenterState>
                      <Inbox className="size-5" />
                      <span>该命令无可选 PATH</span>
                    </CenterState>
                  ) : (
                    <div className="space-y-1.5">
                      {subFields.map((sf: SubFieldDef) => {
                        const on = checkedPaths.has(sf.tr069Path)
                        return (
                          <div
                            key={sf.id}
                            className={`rounded-sm border px-2.5 py-2 transition-all ${
                              on ? 'border-cyan-400/50 bg-cyan-500/8' : 'border-cyan-500/12 bg-[#03050d]/50'
                            }`}
                          >
                            <button
                              type="button"
                              onClick={() => togglePath(sf.tr069Path)}
                              className="flex w-full items-center gap-2 text-left"
                            >
                              <span
                                className={`flex size-4 shrink-0 items-center justify-center rounded-sm border ${
                                  on ? 'border-cyan-400 bg-cyan-400/20' : 'border-cyan-500/30'
                                }`}
                              >
                                {on ? <CheckCircle2 className="size-3 text-cyan-200" /> : null}
                              </span>
                              <span className="min-w-0 flex-1">
                                <span className="block truncate text-[12px] text-cyan-100">{sf.label || sf.mmlCode}</span>
                                <span className="block truncate font-mono text-[10px] text-cyan-300/50">{sf.tr069Path}</span>
                              </span>
                              {sf.isSupported === false ? (
                                <StatusBadge status="minor" label="未确认" />
                              ) : null}
                            </button>
                            {on && isMod ? (
                              <input
                                className="neon-input mt-1.5 w-full"
                                placeholder={sf.defaultValue ? `默认 ${sf.defaultValue}` : `输入 ${sf.valueType} 值`}
                                value={values[sf.tr069Path] ?? ''}
                                onChange={(e) =>
                                  setValues((prev) => ({ ...prev, [sf.tr069Path]: e.target.value }))
                                }
                              />
                            ) : null}
                          </div>
                        )
                      })}
                    </div>
                  )}
                </div>

                {submitErr ? (
                  <div className="mx-3 mb-2 rounded-sm border border-rose-500/40 bg-rose-500/8 px-3 py-1.5 font-mono text-[11px] text-rose-300">
                    {submitErr}
                  </div>
                ) : null}

                <div className="flex items-center justify-between gap-2 border-t border-cyan-500/15 px-3 py-2.5">
                  <span className="font-mono text-[11px] text-cyan-300/55">
                    {selectedSns.length} 设备 × {checkedPaths.size} PATH
                  </span>
                  <NeonButton icon={exec.isPending ? <Loader2 className="animate-spin" /> : <Rocket />} disabled={!canExecute} onClick={handleExecute}>
                    {exec.isPending ? 'DISPATCHING…' : 'EXECUTE'}
                  </NeonButton>
                </div>
              </>
            )}
          </div>
        </GlassPanel>
      </div>
    </PageShell>
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

export default MMLConsolePage
