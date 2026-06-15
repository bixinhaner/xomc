import { useEffect, useMemo, useState } from 'react'
import { Activity, ChevronRight, Cpu, AlertTriangle, Signal } from 'lucide-react'

import { GlassPanel } from '@/components/ui/GlassPanel'
import { HoloGlobe } from '@/components/viz/HoloGlobe'
import { Sparkline } from '@/components/viz/Sparkline'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { useDashboardData, useDeviceStatusByType, useTopAlarmDevices } from '@core/hooks/api/useDashboard'
import { useAlarmCount } from '@core/hooks/api/useAlarms'
import type { AlarmStats, TopAlarmDevice } from '@core/types/dashboard'
import type { BackendDeviceStatusByType } from '@core/types/dashboard'

// 制式展示名（与 v1 TECH_DISPLAY_NAME 一致）。
const TECH_DISPLAY_NAME: Record<string, string> = {
  lte: 'LTE',
  nr: '5G NR',
  gsm: 'GSM',
}

export function BridgePage() {
  const { data, isFetching } = useDashboardData()
  const { data: alarmCount } = useAlarmCount()
  // issue #360：接入真实「按制式设备状态」与「高频告警设备」，替换写死/伪随机假数据。
  const { data: deviceStatusByType } = useDeviceStatusByType()
  const { data: topAlarmDevices } = useTopAlarmDevices()

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
        {/* 左侧 — 按制式设备状态（在线/离线/告警，真实数据，issue #360） */}
        <div className="col-span-3 flex flex-col gap-3 min-h-0">
          <GlassPanel title="DEVICE STATUS · 按制式状态" className="flex-1 min-h-0 overflow-hidden">
            <DeviceStatusByTech statusByType={deviceStatusByType} onlineRate={onlineRate} />
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
            <TopAlarmedDevices devices={topAlarmDevices} />
          </GlassPanel>

          <GlassPanel title="ACS THROUGHPUT" meta="60s">
            <LivePulse />
          </GlassPanel>
        </div>
      </div>

      {/* 底部 — 严重度分布（真实 alarmCounts，issue #360） */}
      <div className="col-span-12">
        <GlassPanel title="ALARM SPECTRUM · 严重度分布" meta="LIVE">
          <SeveritySpectrum alarmCounts={summary?.alarmCounts} />
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

/**
 * DeviceStatusByTech —— 按制式（LTE/5G NR/GSM）渲染真实「在线/离线/告警」分布（issue #360）。
 * 数据来自 useDeviceStatusByType() → GET /dashboard/device-status-by-type。
 * 每制式一行：在线率环 + 在线/离线/告警三色数值条。无数据时给明确空状态而非写死假数。
 */
function DeviceStatusByTech({
  statusByType,
  onlineRate,
}: {
  statusByType: BackendDeviceStatusByType | undefined
  onlineRate: number
}) {
  const rows = useMemo(() => {
    if (!statusByType) return []
    return Object.entries(statusByType).map(([tech, counts]) => {
      const total = counts.online + counts.offline
      const rate = total > 0 ? (counts.online / total) * 100 : 0
      return {
        tech,
        label: TECH_DISPLAY_NAME[tech] ?? tech.toUpperCase(),
        online: counts.online,
        offline: counts.offline,
        alarm: counts.alarm,
        rate,
      }
    })
  }, [statusByType])

  if (rows.length === 0) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-2 p-6 text-center">
        <RadialGauge value={onlineRate} label="ONLINE" size={104} color="#00ff88" />
        <div className="font-mono text-[11px] tracking-[0.2em] text-cyan-300/55">
          暂无按制式设备状态数据
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-3 overflow-auto p-4">
      {rows.map((r) => (
        <div key={r.tech} className="flex items-center gap-3">
          <RadialGauge
            value={r.rate}
            label={r.label}
            size={72}
            color={r.alarm > 0 ? '#ffaa00' : '#00ff88'}
          />
          <div className="min-w-0 flex-1 space-y-1.5">
            <StatusBar label="在线 · ON" value={r.online} max={r.online + r.offline} color="#00ff88" />
            <StatusBar label="离线 · OFF" value={r.offline} max={r.online + r.offline} color="#5b9eff" />
            <StatusBar label="告警 · ALM" value={r.alarm} max={r.online + r.offline} color="#ff2d6f" />
          </div>
        </div>
      ))}
    </div>
  )
}

function StatusBar({
  label,
  value,
  max,
  color,
}: {
  label: string
  value: number
  max: number
  color: string
}) {
  const pct = max > 0 ? Math.min(100, (value / max) * 100) : 0
  return (
    <div className="flex items-center gap-2">
      <div className="w-16 font-mono text-[9px] uppercase tracking-[0.15em] text-cyan-300/60">
        {label}
      </div>
      <div className="h-2 flex-1 overflow-hidden rounded-[1px] bg-cyan-500/8">
        <span
          className="block h-full rounded-[1px]"
          style={{ width: `${pct}%`, background: color, boxShadow: `0 0 6px ${color}` }}
        />
      </div>
      <div
        className="w-8 text-right font-display text-xs font-bold"
        style={{ color, textShadow: `0 0 4px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}

/**
 * TopAlarmedDevices —— 高频告警设备 Top-N（真实数据，issue #360）。
 * 数据来自 useTopAlarmDevices() → /dashboard/summary recent_alarms 折算。
 */
function TopAlarmedDevices({ devices }: { devices: TopAlarmDevice[] | undefined }) {
  const severityColor = (sev: string): string => {
    switch (sev.toLowerCase()) {
      case 'critical':
        return '#ff2d6f'
      case 'major':
        return '#ff7a1a'
      case 'minor':
        return '#ffd400'
      default:
        return '#5b9eff'
    }
  }

  if (!devices || devices.length === 0) {
    return (
      <div className="flex h-full items-center justify-center p-6 font-mono text-[11px] tracking-[0.2em] text-cyan-300/55">
        暂无故障设备
      </div>
    )
  }

  return (
    <div className="overflow-auto">
      {devices.map((d) => {
        const color = severityColor(d.severity)
        return (
          <div
            key={d.deviceSN}
            className="flex items-center gap-2 border-b border-cyan-500/8 px-3 py-2 hover:bg-cyan-500/5"
          >
            <span
              className="size-2 rounded-full animate-breathe"
              style={{ background: color, color, boxShadow: '0 0 8px currentColor' }}
            />
            <div className="min-w-0 flex-1">
              <div className="truncate font-mono text-xs text-cyan-100">{d.deviceSN}</div>
              <div className="truncate text-[10px] uppercase tracking-[0.15em] text-cyan-300/55">
                {d.deviceName || TECH_DISPLAY_NAME[d.technology] || d.technology}
              </div>
            </div>
            <div
              className="font-display text-sm font-bold"
              style={{ color, textShadow: `0 0 6px ${color}` }}
            >
              {d.alarmCount}
            </div>
            <ChevronRight className="size-3 text-cyan-300/40" />
          </div>
        )
      })}
    </div>
  )
}

/**
 * SeveritySpectrum —— 告警严重度分布（真实 alarmCounts，issue #360）。
 * 四档（critical/major/minor/warning）按真实计数渲染条形，替换原 prand 伪随机。
 */
function SeveritySpectrum({ alarmCounts }: { alarmCounts: AlarmStats | undefined }) {
  const rows = useMemo(() => {
    const c = alarmCounts
    return [
      { sev: 'CRIT', color: '#ff2d6f', count: c?.critical ?? 0 },
      { sev: 'MAJ', color: '#ff7a1a', count: c?.major ?? 0 },
      { sev: 'MIN', color: '#ffd400', count: c?.minor ?? 0 },
      { sev: 'WRN', color: '#5b9eff', count: c?.warning ?? 0 },
    ]
  }, [alarmCounts])

  const maxCount = Math.max(1, ...rows.map((r) => r.count))

  return (
    <div className="space-y-1.5 p-4">
      {rows.map((r) => {
        const pct = Math.min(100, (r.count / maxCount) * 100)
        return (
          <div key={r.sev} className="flex items-center gap-3">
            <div
              className="w-10 font-mono text-[10px] font-bold tracking-[0.2em]"
              style={{ color: r.color, textShadow: `0 0 4px ${r.color}` }}
            >
              {r.sev}
            </div>
            <div className="h-3 flex-1 overflow-hidden rounded-[1px] bg-cyan-500/8">
              <span
                className="block h-full rounded-[1px]"
                style={{
                  width: `${pct}%`,
                  background: r.color,
                  boxShadow: r.count > 0 ? `0 0 6px ${r.color}` : undefined,
                }}
              />
            </div>
            <div
              className="w-10 text-right font-display text-xs font-bold"
              style={{ color: r.color }}
            >
              {r.count}
            </div>
          </div>
        )
      })}
    </div>
  )
}
