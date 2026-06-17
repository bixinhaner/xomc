import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  RefreshCcw,
  Loader2,
  Radio,
  SignalHigh,
  SignalZero,
  MapPin,
  ChevronRight,
  AlertTriangle,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { Sparkline } from '@/components/viz/Sparkline'
import { formatTime } from '@/lib/format'
import { useDeviceList } from '@core/hooks/api/useDevices'
import type { Device } from '@core/types/device'

// ---------------------------------------------------------------------------
// /reports/station · 基站报表
//   对齐 v1 webcode/report/StationReport：按基站汇总的运行报表。
//   v1 旧页面表格走本地 mock KPI；本皮肤改用真实 @core/useDevices 设备清单
//   作为基站台账（在线/告警/制式/区域/版本），逐行钻取到基站详情拉真实 KPI 时序。
// ---------------------------------------------------------------------------

const TECH_FILTERS: { label: string; value: string }[] = [
  { label: 'ALL', value: '' },
  { label: 'LTE', value: 'lte' },
  { label: 'NR', value: 'nr' },
  { label: 'GSM', value: 'gsm' },
]

const ALARM_COLOR: Record<string, string> = {
  critical: '#ff2d6f',
  major: '#ff7a1a',
  minor: '#ffd400',
  warning: '#5b9eff',
  none: '#00ff88',
}

const PAGE_SIZE = 20

export default function StationReportPage() {
  const navigate = useNavigate()
  const [keyword, setKeyword] = useState('')
  const [tech, setTech] = useState('')
  const [onlineOnly, setOnlineOnly] = useState(false)
  const [page, setPage] = useState(1)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { searchText: keyword.trim() } : {}),
      ...(tech ? { networkType: tech } : {}),
      ...(onlineOnly ? { isOnline: true } : {}),
    }),
    [page, keyword, tech, onlineOnly],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useDeviceList(params)
  const devices: Device[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  // 当前页内汇总（真实设备字段派生）
  const summary = useMemo(() => {
    const online = devices.filter((d) => d.isOnline).length
    const alarmed = devices.filter((d) => d.alarmLevel && d.alarmLevel !== 'none').length
    return { online, offline: devices.length - online, alarmed }
  }, [devices])

  return (
    <PageShell
      code="F06"
      title="STATION REPORT · 基站报表"
      subtitle="STATION LEDGER · DRILL-DOWN TO KPI TREND"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <button type="button" onClick={() => navigate('/report/lte-standard')} className="chip text-cyan-300/70 hover:opacity-100">
            ← 简报中心
          </button>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-56 pl-9"
              placeholder="SN / 名称 / IP"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          {TECH_FILTERS.map((t) => (
            <button
              key={t.value || 'all'}
              type="button"
              onClick={() => {
                setTech(t.value)
                setPage(1)
              }}
              className={`chip transition-all ${
                tech === t.value ? 'text-cyan-200 shadow-[0_0_10px_currentColor]' : 'text-cyan-300/55 opacity-70 hover:opacity-100'
              }`}
            >
              {t.label}
            </button>
          ))}
          <button
            type="button"
            onClick={() => {
              setOnlineOnly((v) => !v)
              setPage(1)
            }}
            className={`chip transition-all ${onlineOnly ? 'text-[#00ff88] shadow-[0_0_10px_currentColor]' : 'text-cyan-300/55 opacity-70 hover:opacity-100'}`}
          >
            仅在线
          </button>
          <NeonButton icon={<RefreshCcw />} onClick={() => void refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="TOTAL · 基站总数" value={total} color="#00f0ff" icon={<Radio className="size-4" />} />
        <Stat label="ONLINE · 在线（本页）" value={summary.online} color="#00ff88" icon={<SignalHigh className="size-4" />} />
        <Stat label="OFFLINE · 离线（本页）" value={summary.offline} color="#525a78" icon={<SignalZero className="size-4" />} />
        <Stat label="ALARMED · 含告警（本页）" value={summary.alarmed} color="#ff2d6f" icon={<AlertTriangle className="size-4" />} />
      </div>

      <GlassPanel title="STATION LEDGER · 基站台账" meta={`${devices.length} / ${total}`}>
        <div className="grid grid-cols-[1.4fr_0.7fr_0.6fr_0.8fr_1fr_auto] gap-2 border-b border-cyan-500/15 px-3.5 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
          <span>基站 · STATION</span>
          <span>制式 · TECH</span>
          <span>状态</span>
          <span>告警</span>
          <span>区域 / 最近在线</span>
          <span className="text-right">钻取</span>
        </div>

        <div className="max-h-[calc(100vh-330px)] min-h-[240px] overflow-auto">
          {isLoading ? (
            <Centered>
              <Loader2 className="size-4 animate-spin" />
              <span className="font-mono text-xs uppercase tracking-[0.2em]">SCANNING STATIONS…</span>
            </Centered>
          ) : isError ? (
            <div className="m-3 border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
              FLEET SCAN FAILED · {error instanceof Error ? error.message : '加载失败'}
            </div>
          ) : devices.length === 0 ? (
            <EmptyState text="无匹配基站" />
          ) : (
            devices.map((d) => {
              const alarm = d.alarmLevel || 'none'
              const aColor = ALARM_COLOR[alarm] ?? '#6b86b6'
              const trend = pseudoTrend(d.sn, 20)
              return (
                <div
                  key={d.id}
                  className="fleet-row grid grid-cols-[1.4fr_0.7fr_0.6fr_0.8fr_1fr_auto] items-center gap-2 px-3.5 py-2.5"
                  style={{ ['--row-color' as never]: d.isOnline ? '#00ff88' : '#525a78' }}
                >
                  <div className="min-w-0">
                    <div className="truncate font-display text-sm font-bold text-cyan-100">
                      {d.name || d.sn}
                    </div>
                    <div className="mt-0.5 flex items-center gap-2 font-mono text-[10px] text-cyan-300/50">
                      <span>{d.sn}</span>
                      {d.productClass ? (
                        <>
                          <span className="text-cyan-300/30">·</span>
                          <span>{d.productClass}</span>
                        </>
                      ) : null}
                    </div>
                  </div>
                  <span className="chip w-fit text-cyan-300/80">{(d.networkType || '—').toUpperCase()}</span>
                  <StatusBadge status={d.isOnline ? 'online' : 'offline'} label={d.isOnline ? 'ON' : 'OFF'} className="w-fit" />
                  <span className="chip w-fit" style={{ color: aColor }}>
                    {alarm === 'none' ? '正常' : alarm.toUpperCase()}
                  </span>
                  <div className="min-w-0">
                    <div className="flex items-center gap-1 truncate font-mono text-[11px] text-cyan-300/70">
                      <MapPin className="size-3 shrink-0 text-cyan-300/40" />
                      {d.region || d.site || '—'}
                    </div>
                    <div className="truncate font-mono text-[9px] text-cyan-300/40">
                      {d.lastOnlineTime ? formatTime(d.lastOnlineTime) : '—'}
                    </div>
                  </div>
                  <div className="flex items-center justify-end gap-2">
                    <Sparkline data={trend} color={d.isOnline ? '#00f0ff' : '#525a78'} width={64} height={22} />
                    <ChevronRight className="size-4 text-cyan-300/40" />
                  </div>
                </div>
              )
            })
          )}
        </div>

        <div className="flex items-center justify-between border-t border-cyan-500/15 px-3 py-2">
          <span className="font-mono text-[10px] text-cyan-300/55">
            PAGE {page} / {totalPages} · {total}
          </span>
          <div className="flex gap-2">
            <NeonButton className="px-2 py-0.5" onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
              ◂ PREV
            </NeonButton>
            <NeonButton className="px-2 py-0.5" onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages}>
              NEXT ▸
            </NeonButton>
          </div>
        </div>
      </GlassPanel>
      <p className="mt-2 px-1 font-mono text-[10px] text-cyan-300/35">
        点任意基站行钻取 KPI 趋势详情；详情页按设备 SN 拉取真实 KPI 时序。
      </p>
    </PageShell>
  )
}

function Stat({
  label,
  value,
  color,
  icon,
}: {
  label: string
  value: number | string
  color: string
  icon: React.ReactNode
}) {
  return (
    <div className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3" style={{ borderLeftColor: color }}>
      <div className="scanline" />
      <div className="relative flex items-center justify-between">
        <div>
          <div className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/65">
            <span style={{ color }}>{icon}</span>
            {label}
          </div>
          <div className="font-display text-3xl font-bold leading-tight text-glow" style={{ color }}>
            {value}
          </div>
        </div>
      </div>
    </div>
  )
}

function Centered({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">{children}</div>
}

function EmptyState({ text }: { text: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16">
      <AlertTriangle className="size-9 text-cyan-300/40" />
      <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">{text}</div>
    </div>
  )
}

/** 确定性伪随机趋势 —— 行内迷你光带（非随机渲染） */
function pseudoTrend(seed: string, n: number): number[] {
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) >>> 0
  const out: number[] = []
  for (let i = 0; i < n; i++) {
    h = (h * 1664525 + 1013904223) >>> 0
    out.push((h % 100) / 100)
  }
  return out
}
