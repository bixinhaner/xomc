import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  Loader2,
  Inbox,
  XCircle,
  CheckCircle2,
  Cpu,
  Rocket,
  ScanLine,
  AlertTriangle,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useDeviceList } from '@core/hooks/api/useDevices'
import { useParseMML, useExecuteStatements } from '@core/hooks/api/useMmlConsole'
import { useUserStore } from '@core/store/userStore'
import type { Device } from '@core/types/device'
import type { Statement, ParseError } from '@core/types/mmlConsole'

// ─────────────────────────────────────────────────────────────
// MML EDITOR · 多语句脚本控制台（console-v2，real）
// v1 路由 mml/console-v2（ConsoleV2）—— MML 文本编辑 → 解析 → N 设备扇出执行。
//   设备 useDeviceList；解析 useParseMML(POST /mml/parse)；
//   执行 useExecuteStatements(POST /mml/execute-statements)。
// 执行成功导航到任务记录页查看推进状态。
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

const SAMPLE = ['LST DEVICE_INFO;', 'LST FAP_CONTROL;'].join('\n')

export function MMLConsoleV2Page() {
  const navigate = useNavigate()
  const creator = useUserStore((s) => s.currentUser?.username) ?? ''

  const [deviceKw, setDeviceKw] = useState('')
  const [selectedSns, setSelectedSns] = useState<string[]>([])
  const [mml, setMml] = useState('')
  const [parsed, setParsed] = useState<Statement[]>([])
  const [parseErrors, setParseErrors] = useState<ParseError[]>([])
  const [submitErr, setSubmitErr] = useState('')

  const { data: devicePage, isLoading: devLoading, isError: devError } = useDeviceList({
    page: 1,
    pageSize: 200,
  })
  const allDevices = devicePage?.items ?? []
  const kw = deviceKw.trim().toLowerCase()
  const devices = kw
    ? allDevices.filter((d) => d.sn?.toLowerCase().includes(kw) || d.name?.toLowerCase().includes(kw))
    : allDevices

  const parse = useParseMML()
  const exec = useExecuteStatements()

  const toggleDevice = (sn: string) =>
    setSelectedSns((prev) => (prev.includes(sn) ? prev.filter((x) => x !== sn) : [...prev, sn]))

  const handleParse = () => {
    setSubmitErr('')
    if (!mml.trim()) {
      setParsed([])
      setParseErrors([])
      return
    }
    parse.mutate(
      { mmlString: mml, lang: 'zh-CN' },
      {
        onSuccess: (resp) => {
          setParsed(resp.statements)
          setParseErrors(resp.parseErrors)
        },
        onError: (e) => setSubmitErr(e instanceof Error ? e.message : '解析失败'),
      }
    )
  }

  const canExecute =
    selectedSns.length > 0 && parsed.length > 0 && parseErrors.length === 0 && !exec.isPending

  const handleExecute = () => {
    if (!canExecute) return
    setSubmitErr('')
    exec.mutate(
      {
        statements: parsed,
        deviceSns: selectedSns,
        taskName: `MML_BATCH_${new Date().toISOString().slice(0, 19)}`,
        creator,
        executeType: 'immediate',
      },
      {
        onSuccess: () => navigate('/mml/task-records'),
        onError: (e) => setSubmitErr(e instanceof Error ? e.message : '执行失败'),
      }
    )
  }

  const unknownCount = useMemo(
    () => parsed.reduce((acc, s) => acc + (s.unknownCodes?.length ?? 0), 0),
    [parsed]
  )

  return (
    <PageShell
      code="F06"
      title="MML EDITOR · 脚本控制台 V2"
      subtitle="MAN-MACHINE LANGUAGE · MULTI-STATEMENT BATCH DISPATCH"
      bare
      toolbar={
        <span className="chip" style={{ color: '#00f0ff' }}>
          <Cpu className="size-3" /> {selectedSns.length} DEV
        </span>
      }
    >
      <div className="grid h-full grid-cols-[300px_1fr] gap-3">
        {/* 设备选择 */}
        <GlassPanel title="TARGET DEVICES" meta={`${selectedSns.length}/${allDevices.length}`} className="min-h-0 overflow-hidden">
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
                        on ? 'border-cyan-400/60 bg-cyan-500/15' : 'border-cyan-500/12 bg-cyan-500/5 hover:border-cyan-400/40'
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

        {/* 编辑器 + 解析结果 */}
        <div className="grid min-h-0 grid-rows-[1fr_1fr] gap-3">
          <GlassPanel title="MML TEXT · 命令脚本" meta="LST/MOD/ADD/RMV; 每行一句" className="min-h-0 overflow-hidden">
            <div className="flex h-full flex-col">
              <div className="terminal m-3 min-h-0 flex-1">
                <textarea
                  value={mml}
                  onChange={(e) => setMml(e.target.value)}
                  spellCheck={false}
                  placeholder={`键入 MML，每句以 ; 结尾，例如：\n${SAMPLE}`}
                  className="h-full w-full resize-none bg-transparent p-3 font-mono text-[12.5px] leading-relaxed text-emerald-200 outline-none placeholder:text-emerald-700/50"
                />
              </div>
              <div className="flex items-center justify-between gap-2 border-t border-emerald-500/25 bg-black/30 px-3 py-2">
                <button
                  type="button"
                  onClick={() => setMml(SAMPLE)}
                  className="rounded-sm border border-cyan-500/20 bg-cyan-500/5 px-2.5 py-1 font-mono text-[10px] uppercase tracking-[0.15em] text-cyan-300/70 hover:text-cyan-100"
                >
                  载入示例
                </button>
                <NeonButton
                  icon={parse.isPending ? <Loader2 className="animate-spin" /> : <ScanLine />}
                  disabled={!mml.trim() || parse.isPending}
                  onClick={handleParse}
                >
                  {parse.isPending ? 'PARSING…' : 'PARSE'}
                </NeonButton>
              </div>
            </div>
          </GlassPanel>

          <GlassPanel
            title="PARSED STATEMENTS · 解析结果"
            meta={parsed.length ? `${parsed.length} STMT · ${parseErrors.length} ERR` : '—'}
            className="min-h-0 overflow-hidden"
          >
            <div className="flex h-full flex-col">
              <div className="min-h-0 flex-1 overflow-auto p-3">
                {parse.isError ? (
                  <CenterState tone="err">
                    <XCircle className="size-5" />
                    <span>解析失败</span>
                  </CenterState>
                ) : parsed.length === 0 && parseErrors.length === 0 ? (
                  <CenterState>
                    <ScanLine className="size-6" />
                    <span>键入 MML 并点击 PARSE</span>
                  </CenterState>
                ) : (
                  <div className="space-y-2">
                    {parseErrors.map((e, i) => (
                      <div
                        key={`err-${i}`}
                        className="flex items-start gap-2 rounded-sm border border-rose-500/40 bg-rose-500/8 px-3 py-2 font-mono text-[11px] text-rose-300"
                      >
                        <AlertTriangle className="mt-0.5 size-3.5 shrink-0" />
                        <span>
                          第 {e.statementIndex + 1} 句解析失败：{e.reason}
                          {e.raw ? <span className="text-rose-300/60"> · {e.raw}</span> : null}
                        </span>
                      </div>
                    ))}
                    {parsed.map((s) => {
                      const color = OP_COLOR[s.operationType] ?? '#6b86b6'
                      const hasUnknown = (s.unknownCodes?.length ?? 0) > 0
                      return (
                        <div
                          key={s.uid}
                          className="rounded-sm border border-cyan-500/15 bg-[#03050d]/50 px-3 py-2"
                        >
                          <div className="flex items-center gap-2">
                            <span
                              className="rounded-sm border px-1.5 py-0.5 font-mono text-[9px]"
                              style={{ color, borderColor: `${color}55` }}
                            >
                              {s.operationType}
                            </span>
                            <span className="truncate font-mono text-[11px] text-cyan-100">{s.commandCode}</span>
                            {s.commandId ? (
                              <StatusBadge status="ok" label="命中" />
                            ) : (
                              <StatusBadge status="critical" label="未命中" />
                            )}
                          </div>
                          {s.selectedSubFieldIds && s.selectedSubFieldIds.length > 0 ? (
                            <div className="mt-1 font-mono text-[10px] text-cyan-300/55">
                              {s.selectedSubFieldIds.length} 个 sub-field
                            </div>
                          ) : null}
                          {hasUnknown ? (
                            <div className="mt-1 font-mono text-[10px] text-amber-300/80">
                              未知字段：{s.unknownCodes.join(', ')}
                            </div>
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
                  {selectedSns.length} 设备 × {parsed.length} 语句
                  {unknownCount > 0 ? ` · ${unknownCount} 未知字段` : ''}
                </span>
                <NeonButton
                  icon={exec.isPending ? <Loader2 className="animate-spin" /> : <Rocket />}
                  disabled={!canExecute}
                  onClick={handleExecute}
                >
                  {exec.isPending ? 'DISPATCHING…' : 'EXECUTE'}
                </NeonButton>
              </div>
            </div>
          </GlassPanel>
        </div>
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

export default MMLConsoleV2Page
