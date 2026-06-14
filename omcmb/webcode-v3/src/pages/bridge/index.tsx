import { useEffect, useMemo, useState } from 'react'
import { Activity, ChevronRight, Cpu, AlertTriangle, Signal } from 'lucide-react'

import { GlassPanel } from '@/components/ui/GlassPanel'
import { HoloGlobe } from '@/components/viz/HoloGlobe'
import { Sparkline } from '@/components/viz/Sparkline'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { useDashboardData } from '@core/hooks/api/useDashboard'
import { useAlarmCount } from '@core/hooks/api/useAlarms'

export function BridgePage() {
  const { data, isFetching } = useDashboardData()
  const { data: alarmCount } = useAlarmCount()

  const summary = data?.summary
  const totalDev = summary?.deviceCounts?.total ?? 0
  const onlineDev = summary?.deviceCounts?.online ?? 0
  const onlineRate = totalDev > 0 ? (onlineDev / totalDev) * 100 : 0
  const activeAlm = alarmCount?.total_active ?? summary?.alarmCounts?.total ?? 0
  const critAlm = alarmCount?.critical ?? summary?.alarmCounts?.critical ?? 0

  return (
    <div className="warp-in grid h-full grid-cols-12 grid-rows-[auto_1fr_auto] gap-3">
      {/* 顶部三大数 */}
      <div className="col-span-12 grid grid-cols-4 gap-3">
        <BigStat
          code="01"
          label="设备总数 · TOTAL"
          value={totalDev.toLocaleString()}
          color="#00f0ff"
          icon={<Cpu className="size-4" />}
          trend={[42, 48, 55, 51, 60, 62, 71, 78, 82, 88]}
        />
        <BigStat
          code="02"
          label="在线率 · ONLINE"
          value={`${onlineRate.toFixed(1)}%`}
          color="#00ff88"
          icon={<Signal className="size-4" />}
          trend={[80, 78, 84, 82, 88, 86, 90, 92, 91, 94]}
        />
        <BigStat
          code="03"
          label="活动告警 · ACTIVE"
          value={activeAlm.toLocaleString()}
          color={activeAlm > 50 ? '#ff2d6f' : '#ffaa00'}
          icon={<AlertTriangle className="size-4" />}
          trend={[12, 18, 14, 22, 30, 28, 34, 40, 38, 32]}
        />
        <BigStat
          code="04"
          label="紧急告警 · CRIT"
          value={critAlm.toLocaleString()}
          color="#ff2d6f"
          icon={<Activity className="size-4" />}
          trend={[1, 2, 1, 4, 3, 5, 6, 5, 7, 8]}
        />
      </div>

      {/* 中央 + 左右 */}
      <div className="col-span-12 grid grid-cols-12 gap-3 min-h-0">
        {/* 左侧 KPI 圆环组 */}
        <div className="col-span-3 flex flex-col gap-3 min-h-0">
          <GlassPanel title="GLOBAL KPI · 全网指标" className="flex-1 min-h-0">
            <div className="grid grid-cols-2 gap-4 p-5">
              <div className="flex flex-col items-center gap-1">
                <RadialGauge value={onlineRate} label="ONLINE" size={104} color="#00ff88" />
              </div>
              <div className="flex flex-col items-center gap-1">
                <RadialGauge value={73.5} label="RRC SR" size={104} color="#00f0ff" />
              </div>
              <div className="flex flex-col items-center gap-1">
                <RadialGauge value={84} label="HO SR" size={104} color="#a855f7" />
              </div>
              <div className="flex flex-col items-center gap-1">
                <RadialGauge value={47} label="LOAD" size={104} color="#ffaa00" />
              </div>
            </div>
            <div className="border-t border-cyan-500/15 px-4 py-2 font-mono text-[10px] tracking-[0.18em] text-cyan-300/55">
              {isFetching ? 'SYNC… · 30s 自动' : 'TICK · 30s 自动'}
            </div>
          </GlassPanel>
        </div>

        {/* 中央地球 */}
        <div className="col-span-6 min-h-0">
          <GlassPanel
            title="GLOBE · 全网态势"
            meta="LIVE · 22 NODES VISIBLE"
            className="h-full"
            strong
          >
            <div className="relative flex h-[calc(100%-40px)] items-center justify-center overflow-hidden">
              <HoloGlobe size={420} />

              {/* 角标 KPI */}
              <FloatingChip
                pos="top-4 left-4"
                label="LATENCY"
                value="18ms"
                color="#00ff88"
              />
              <FloatingChip
                pos="top-4 right-4"
                label="THROUGHPUT"
                value="3.42 Gbps"
                color="#00f0ff"
              />
              <FloatingChip
                pos="bottom-4 left-4"
                label="ACS LOAD"
                value="42%"
                color="#a855f7"
              />
              <FloatingChip
                pos="bottom-4 right-4"
                label="PEAK CPS"
                value="284"
                color="#ffaa00"
              />
            </div>
          </GlassPanel>
        </div>

        {/* 右侧 Top-N + 实时波形 */}
        <div className="col-span-3 flex flex-col gap-3 min-h-0">
          <GlassPanel title="TOP ALARMED · 故障设备" className="flex-1 min-h-0 overflow-hidden">
            <div className="overflow-auto">
              {TOP_DEVS.map((d) => (
                <div
                  key={d.sn}
                  className="flex items-center gap-2 border-b border-cyan-500/8 px-3 py-2 hover:bg-cyan-500/5"
                >
                  <span
                    className="size-2 rounded-full animate-breathe"
                    style={{ background: d.color, color: d.color, boxShadow: '0 0 8px currentColor' }}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="truncate font-mono text-xs text-cyan-100">{d.sn}</div>
                    <div className="truncate text-[10px] uppercase tracking-[0.15em] text-cyan-300/55">
                      {d.region}
                    </div>
                  </div>
                  <div
                    className="font-display text-sm font-bold"
                    style={{ color: d.color, textShadow: `0 0 6px ${d.color}` }}
                  >
                    {d.cnt}
                  </div>
                  <ChevronRight className="size-3 text-cyan-300/40" />
                </div>
              ))}
            </div>
          </GlassPanel>

          <GlassPanel title="ACS THROUGHPUT" meta="60s">
            <LivePulse />
          </GlassPanel>
        </div>
      </div>

      {/* 底部 — 严重度光谱 */}
      <div className="col-span-12">
        <GlassPanel title="ALARM SPECTRUM · 严重度分布" meta="60min · LIVE">
          <SeveritySpectrum />
        </GlassPanel>
      </div>
    </div>
  )
}

function BigStat({
  code,
  label,
  value,
  color,
  icon,
  trend,
}: {
  code: string
  label: string
  value: string
  color: string
  icon: React.ReactNode
  trend: number[]
}) {
  return (
    <div className="glass relative overflow-hidden rounded-sm">
      <div className="scanline" />
      <div className="relative flex items-stretch gap-3 p-4">
        <div className="flex flex-col items-center justify-center gap-1 border-r border-cyan-500/15 pr-3">
          <span className="font-display text-[10px] tracking-[0.25em] text-cyan-300/55">
            {code}
          </span>
          <span className="text-cyan-300" style={{ color }}>
            {icon}
          </span>
        </div>
        <div className="flex flex-1 flex-col">
          <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
            {label}
          </div>
          <div
            className="font-display text-3xl font-bold leading-tight text-glow"
            style={{ color }}
          >
            {value}
          </div>
        </div>
        <Sparkline data={trend} color={color} width={96} height={36} />
      </div>
    </div>
  )
}

function FloatingChip({
  pos,
  label,
  value,
  color,
}: {
  pos: string
  label: string
  value: string
  color: string
}) {
  return (
    <div
      className={`absolute ${pos} rounded-sm border border-cyan-500/30 bg-[#03050d]/65 px-2.5 py-1.5 backdrop-blur-md`}
    >
      <div className="font-mono text-[9px] uppercase tracking-[0.2em] text-cyan-300/60">
        {label}
      </div>
      <div
        className="font-display text-sm font-bold"
        style={{ color, textShadow: `0 0 6px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}

function LivePulse() {
  // 持续生成的实时数据（初始用确定性种子；之后由 useEffect 周期性更新）
  const [data, setData] = useState<number[]>(() =>
    prand(7, 36).map((r, i) => 30 + Math.sin(i / 2) * 18 + r * 8)
  )
  useEffect(() => {
    const t = setInterval(() => {
      setData((prev) => [
        ...prev.slice(1),
        Math.max(10, Math.min(90, prev[prev.length - 1] + (Math.random() - 0.5) * 22)),
      ])
    }, 900)
    return () => clearInterval(t)
  }, [])
  const last = data[data.length - 1]
  return (
    <div className="flex items-center gap-3 p-4">
      <Sparkline data={data} color="#00f0ff" width={180} height={56} />
      <div className="text-right">
        <div className="font-display text-2xl font-bold text-cyan-200 text-glow">
          {last.toFixed(0)}
        </div>
        <div className="font-mono text-[10px] tracking-[0.2em] text-cyan-300/55">
          MSG/s
        </div>
      </div>
    </div>
  )
}

/** 简单确定性 PRNG —— 同 seed 始终输出同序列，避免 render 中调用 Math.random */
function prand(seed: number, n: number): number[] {
  let h = seed >>> 0
  const out: number[] = []
  for (let i = 0; i < n; i++) {
    h = (h * 1664525 + 1013904223) >>> 0
    out.push((h % 1000) / 1000)
  }
  return out
}

const TOP_DEVS = [
  { sn: 'GNB-1102-LXK', region: '北京 · 海淀', cnt: 12, color: '#ff2d6f' },
  { sn: 'ENB-2241-PUS', region: '上海 · 浦东', cnt: 9, color: '#ff2d6f' },
  { sn: 'GNB-3308-XCD', region: '广州 · 越秀', cnt: 7, color: '#ff7a1a' },
  { sn: 'CPE-A1B0E9-CD', region: '成都 · 高新', cnt: 6, color: '#ff7a1a' },
  { sn: 'ENB-7714-CHL', region: '长沙 · 雨花', cnt: 5, color: '#ffd400' },
  { sn: 'GNB-4521-WHM', region: '武汉 · 江汉', cnt: 4, color: '#ffd400' },
  { sn: 'CPE-CC3F32-XJ', region: '乌鲁木齐', cnt: 3, color: '#5b9eff' },
]

function SeveritySpectrum() {
  // 60 列 × 4 行（critical/major/minor/warning）
  // 用稳定的伪随机种子，避免 render 中调用 Math.random
  const rows = useMemo(
    () =>
      (
        [
          { sev: 'CRIT', color: '#ff2d6f', seed: 17 },
          { sev: 'MAJ', color: '#ff7a1a', seed: 41 },
          { sev: 'MIN', color: '#ffd400', seed: 73 },
          { sev: 'WRN', color: '#5b9eff', seed: 109 },
        ] as const
      ).map((r) => ({
        sev: r.sev,
        color: r.color,
        data: prand(r.seed, 60),
      })),
    []
  )
  return (
    <div className="space-y-1.5 p-4">
      {rows.map((r) => (
        <div key={r.sev} className="flex items-center gap-3">
          <div
            className="w-10 font-mono text-[10px] font-bold tracking-[0.2em]"
            style={{ color: r.color, textShadow: `0 0 4px ${r.color}` }}
          >
            {r.sev}
          </div>
          <div className="flex flex-1 items-end gap-[3px]">
            {r.data.map((v, i) => (
              <span
                key={i}
                className="block flex-1 rounded-[1px]"
                style={{
                  height: 4 + v * 18,
                  background: r.color,
                  opacity: 0.18 + v * 0.7,
                  boxShadow: v > 0.65 ? `0 0 6px ${r.color}` : undefined,
                }}
              />
            ))}
          </div>
          <div
            className="w-10 text-right font-display text-xs font-bold"
            style={{ color: r.color }}
          >
            {Math.floor(r.data.reduce((a, b) => a + b, 0))}
          </div>
        </div>
      ))}
    </div>
  )
}
