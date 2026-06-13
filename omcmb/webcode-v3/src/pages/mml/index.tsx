import { useEffect, useRef, useState } from 'react'
import { Play, Trash2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'

interface Line {
  id: string
  type: 'in' | 'out' | 'sys' | 'err'
  text: string
  ts: string
}

const BANNER = [
  '──────────────────────────────────────────────',
  '  STARFORGE · MML CONSOLE · v3.0   READY',
  '  TR-069 / CWMP MAN-MACHINE LANGUAGE TERMINAL',
  '──────────────────────────────────────────────',
  '',
  '> 输入命令后回车执行；本终端为本地交互模拟，',
  '> 接通后端后将打通真实 mml.exec 通道。',
  '',
]

const SAMPLE: Record<string, string[]> = {
  HELP: [
    '可用命令：',
    '  LST  DEV  [SN=...]            · 查询设备',
    '  GET  PARAM <PATH>             · 读取 TR-069 参数',
    '  SET  PARAM <PATH> = <VAL>     · 修改参数',
    '  RBT  DEV   <SN>               · 重启设备',
    '  ACT  CELL  <ID>               · 激活小区',
    '  DACT CELL  <ID>               · 去激活小区',
    '  CLS                            · 清屏',
  ],
  CLS: [],
  'LST DEV': [
    '+----+-------------------+----------+--------+',
    '| #  | SN                | TYPE     | STATE  |',
    '+----+-------------------+----------+--------+',
    '| 1  | ENB-2241-PUS      | LTE      | ONLINE |',
    '| 2  | GNB-1102-LXK      | 5G NR    | ONLINE |',
    '| 3  | CPE-A1B0E9-CD     | CPE      | OFFLINE|',
    '| 4  | GNB-3308-XCD      | 5G NR    | ONLINE |',
    '+----+-------------------+----------+--------+',
    '4 rows · 0.034s',
  ],
  RBT: [
    '> 已发起重启请求',
    '> Connection-Request → CPE …',
    '> CPE 接受 (HTTP 200)',
    '> 等待重启完成 (60s 超时)',
    '> [OK] 设备已上线',
  ],
  ACT: [
    '> 小区激活',
    '> SetParameterValues SOAP →',
    '> CPE 响应 OK',
    '> [DONE] 状态：ACTIVE',
  ],
}

function exec(cmd: string): string[] {
  const upper = cmd.trim().toUpperCase()
  if (!upper) return []
  if (upper === 'CLS') return ['__CLS__']
  if (upper === 'HELP' || upper === '?') return SAMPLE.HELP
  if (upper.startsWith('LST DEV')) return SAMPLE['LST DEV']
  if (upper.startsWith('RBT')) return SAMPLE.RBT
  if (upper.startsWith('ACT') || upper.startsWith('DACT')) return SAMPLE.ACT
  if (upper.startsWith('GET PARAM')) {
    return [`> ${upper}`, '> Device.DeviceInfo.SoftwareVersion = 6.4.2', '> 0.018s']
  }
  if (upper.startsWith('SET PARAM')) {
    return [`> ${upper}`, '> SetParameterValues OK', '> [DONE] 0.024s']
  }
  return [`unknown command: ${cmd}`, "type 'HELP' for command list"]
}

export function MMLPage() {
  const [lines, setLines] = useState<Line[]>(() =>
    BANNER.map((t, i) => ({
      id: `b-${i}`,
      type: 'sys',
      text: t,
      ts: '',
    }))
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

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const cmd = input.trim()
    if (!cmd) return
    const ts = nowHHMMSS()
    const echo: Line = { id: `e-${Date.now()}`, type: 'in', text: `OPR>$ ${cmd}`, ts }
    const out = exec(cmd)
    if (out[0] === '__CLS__') {
      setLines([])
    } else {
      setLines((prev) => [
        ...prev,
        echo,
        ...out.map((t, i) => ({
          id: `o-${Date.now()}-${i}`,
          type: 'out' as const,
          text: t,
          ts: '',
        })),
      ])
    }
    setHistory((h) => [cmd, ...h].slice(0, 50))
    setHi(-1)
    setInput('')
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

  return (
    <PageShell
      code="F06"
      title="MML CONSOLE · 终端"
      subtitle="MAN-MACHINE LANGUAGE · CRT EMULATION"
      bare
      toolbar={
        <NeonButton tone="danger" icon={<Trash2 />} onClick={() => setLines([])}>
          CLEAR
        </NeonButton>
      }
    >
      <div className="grid h-full grid-cols-[1fr_280px] gap-3">
        <GlassPanel title="TERMINAL · BRIDGE-04" meta="ONLINE" className="min-h-0">
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
                  {l.text || ' '}
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
                placeholder="键入命令并回车 · 例如 HELP / LST DEV / RBT DEV CPE-A1B0E9"
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

        <GlassPanel title="QUICK MACROS" meta="POOL">
          <div className="grid gap-2 p-3">
            {[
              'HELP',
              'LST DEV',
              'GET PARAM Device.DeviceInfo.SoftwareVersion',
              'SET PARAM Device.WiFi.SSID = STARFORGE',
              'RBT DEV CPE-A1B0E9',
              'ACT CELL 1102-3',
              'DACT CELL 1102-3',
              'CLS',
            ].map((m) => (
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
            ))}
          </div>
        </GlassPanel>
      </div>
    </PageShell>
  )
}

function nowHHMMSS() {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}
