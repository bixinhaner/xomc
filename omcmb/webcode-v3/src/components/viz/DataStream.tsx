import { useEffect, useState } from 'react'

export interface StreamLine {
  id: string
  ts: string
  level: 'info' | 'warn' | 'crit' | 'ok'
  text: string
}

const LEVEL_COLOR: Record<StreamLine['level'], string> = {
  info: '#5b9eff',
  warn: '#ffaa00',
  crit: '#ff2d6f',
  ok: '#00ff88',
}

const SEED: StreamLine[] = [
  { id: 'a', ts: '12:08:31', level: 'info', text: 'CPE-D8E1F4 Inform 接收 EVENT=2 PERIODIC' },
  { id: 'b', ts: '12:08:30', level: 'warn', text: 'pm-ingest queue depth = 1843 (>1500)' },
  { id: 'c', ts: '12:08:29', level: 'ok', text: 'firmware push ok dev=ENB-08F311 → 6.4.2' },
  { id: 'd', ts: '12:08:28', level: 'crit', text: '[ALM] LinkDown raised dev=GNB-1102' },
  { id: 'e', ts: '12:08:27', level: 'info', text: 'session.commit dev=CPE-A1B0E9 cwmpId=39' },
  { id: 'f', ts: '12:08:26', level: 'info', text: 'ACS connection-request → 100.64.18.31' },
  { id: 'g', ts: '12:08:25', level: 'ok', text: '[ALM] cleared LowMemoryWarning dev=ENB-7714' },
  { id: 'h', ts: '12:08:24', level: 'warn', text: 'cache miss ratio L2 = 4.2% (>3%)' },
  { id: 'i', ts: '12:08:23', level: 'info', text: 'sched: KPI-aggregator slot 12:08 dispatched' },
  { id: 'j', ts: '12:08:22', level: 'info', text: 'mr-parser file mre_2026... 2.4MB ok' },
  { id: 'k', ts: '12:08:21', level: 'ok', text: 'license 23/100K ok' },
  { id: 'l', ts: '12:08:20', level: 'info', text: 'auth.refresh user=admin' },
  { id: 'm', ts: '12:08:19', level: 'crit', text: '[ALM] HighTemperature dev=ENB-2241 T=82℃' },
  { id: 'n', ts: '12:08:18', level: 'warn', text: 'SOAP fault 9001 dev=CPE-CC3F32' },
  { id: 'o', ts: '12:08:17', level: 'info', text: 'NATS jetstream lag = 3' },
  { id: 'p', ts: '12:08:16', level: 'info', text: 'http 200 GET /api/v1/devices 18ms' },
]

const ROTATING = [
  '设备 ENB-{x} 注册成功',
  'KPI 采集任务 #{x} 完成',
  '告警 LinkDown 升级 dev=GNB-{x}',
  'firmware OTA dev=CPE-{x} 进度 {p}%',
  'SOAP fault dev=CPE-{x}',
  'session.expire dev=CPE-{x}',
  'pm-ingest 写入 {x} 行',
  'http 200 POST /api/v1/mml/exec {x}ms',
  '[ALM] cleared dev=ENB-{x}',
  'cache.eviction L2 keys={x}',
]

function rotateOne(): StreamLine {
  const tpl = ROTATING[Math.floor(Math.random() * ROTATING.length)]
  const text = tpl
    .replace('{x}', String(Math.floor(Math.random() * 9999)).padStart(4, '0'))
    .replace('{p}', String(Math.floor(Math.random() * 100)))
  const levels: StreamLine['level'][] = ['info', 'info', 'info', 'ok', 'warn', 'crit']
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return {
    id: `${Date.now()}-${Math.random()}`,
    ts: `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`,
    level: levels[Math.floor(Math.random() * levels.length)],
    text,
  }
}

export function DataStream({ max = 32 }: { max?: number }) {
  const [lines, setLines] = useState<StreamLine[]>(SEED)

  useEffect(() => {
    const t = setInterval(() => {
      setLines((prev) => [rotateOne(), ...prev].slice(0, max))
    }, 1500)
    return () => clearInterval(t)
  }, [max])

  return (
    <div className="font-mono text-[11px] leading-relaxed">
      {lines.map((l) => (
        <div key={l.id} className="flex gap-2 py-0.5 px-3 hover:bg-cyan-500/5">
          <span className="shrink-0 text-cyan-300/40">{l.ts}</span>
          <span
            className="shrink-0 uppercase font-semibold"
            style={{ color: LEVEL_COLOR[l.level], textShadow: `0 0 6px ${LEVEL_COLOR[l.level]}` }}
          >
            {l.level}
          </span>
          <span className="text-cyan-100/85 truncate">{l.text}</span>
        </div>
      ))}
    </div>
  )
}
