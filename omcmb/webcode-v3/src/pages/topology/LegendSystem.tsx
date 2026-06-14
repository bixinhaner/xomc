import { useMemo } from 'react'
import { Loader2, RefreshCcw, BookMarked } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { useTopoGraph } from '@core/hooks/api/useTopology'

// ── 图例定义（与拓扑/画布渲染色板一致）──
interface NodeTypeLegend {
  type: string
  color: string
  desc: string
}
const NODE_TYPES: NodeTypeLegend[] = [
  { type: 'eNB', color: '#5b9eff', desc: 'LTE 4G 基站' },
  { type: 'gNB', color: '#a855f7', desc: '5G NR 基站' },
  { type: 'CPE', color: '#00f0ff', desc: '客户终端设备' },
  { type: 'eGW', color: '#ff7a1a', desc: '边缘网关' },
  { type: 'router', color: '#7c8cff', desc: '路由器' },
  { type: 'switch', color: '#ff5d8f', desc: '交换机' },
  { type: 'domain', color: '#00ff88', desc: '设备域节点' },
  { type: 'site', color: '#ffd400', desc: '站点节点' },
]

interface StatusLegend {
  key: 'online' | 'offline' | 'alarm' | 'maintenance'
  label: string
  color: string
  desc: string
}
const NODE_STATUS: StatusLegend[] = [
  { key: 'online', label: '在线', color: '#00ff88', desc: '节点正常通联' },
  { key: 'offline', label: '离线', color: '#525a78', desc: '节点失联 / 断电' },
  { key: 'alarm', label: '告警', color: '#ff2d6f', desc: '存在活动告警' },
  { key: 'maintenance', label: '维护', color: '#ffaa00', desc: '维护 / 锁定中' },
]

interface SeverityLegend {
  label: string
  color: string
  desc: string
}
const ALARM_SEVERITY: SeverityLegend[] = [
  { label: '紧急 CRITICAL', color: '#ff2d6f', desc: '业务中断级' },
  { label: '重要 MAJOR', color: '#ff7a1a', desc: '严重影响业务' },
  { label: '次要 MINOR', color: '#ffd400', desc: '轻微影响' },
  { label: '警告 WARNING', color: '#5b9eff', desc: '提示性告警' },
]

interface EdgeLegend {
  label: string
  color: string
  dash?: string
  desc: string
}
const EDGE_TYPES: EdgeLegend[] = [
  { label: '活动 ACTIVE', color: '#00ff88', desc: '链路正常承载' },
  { label: '降级 DEGRADED', color: '#ffaa00', dash: '8 4', desc: '链路降级 / 抖动' },
  { label: '断开 INACTIVE', color: '#525a78', dash: '4 4', desc: '链路中断' },
]

export default function LegendSystem() {
  // 拉一份拓扑统计，给图例里的节点类型 / 状态标上「实时计数」
  const { data: graph, isLoading, isError, isFetching, refetch } = useTopoGraph({ limit: 2000 })
  const stats = graph?.statistics

  const typeCounts = useMemo<Record<string, number>>(
    () => stats?.nodeTypeCounts ?? {},
    [stats]
  )
  const statusCounts = useMemo<Record<string, number>>(() => {
    if (!stats) return {} as Record<string, number>
    return {
      online: stats.onlineNodes,
      offline: stats.offlineNodes,
      alarm: stats.alarmNodes,
      maintenance: stats.maintenanceNodes,
    }
  }, [stats])
  const edgeCounts = useMemo<Record<string, number>>(() => {
    if (!stats) return {} as Record<string, number>
    return {
      active: stats.activeEdges,
      degraded: stats.degradedEdges,
      inactive: stats.inactiveEdges,
    }
  }, [stats])

  return (
    <PageShell
      code="F06"
      title="LEGEND · 图例系统"
      subtitle="SYMBOL REFERENCE · 节点 / 状态 / 告警 / 链路"
      isFetching={isFetching}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => void refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      {isLoading ? (
        <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING COUNTS…</span>
        </div>
      ) : (
        <>
          {isError && (
            <div className="mb-3 flex items-center gap-2 border border-amber-500/30 bg-amber-500/5 px-3 py-2 font-mono text-[11px] uppercase tracking-[0.18em] text-amber-300">
              <BookMarked className="size-3.5" />
              实时计数获取失败 · 仅展示静态图例
            </div>
          )}
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
            {/* 节点类型 */}
            <GlassPanel title="NODE TYPE · 节点类型" meta="ICON" strong>
              <div className="grid grid-cols-1 gap-px bg-cyan-500/5 sm:grid-cols-2">
                {NODE_TYPES.map((it) => (
                  <div
                    key={it.type}
                    className="flex items-center gap-3 bg-[#03050d]/50 px-3 py-2.5"
                  >
                    <span
                      className="flex size-7 shrink-0 items-center justify-center rounded-full text-[10px] font-bold"
                      style={{
                        color: it.color,
                        border: `1px solid ${it.color}`,
                        boxShadow: `0 0 8px ${it.color}55`,
                        background: `${it.color}1a`,
                      }}
                    >
                      {it.type.slice(0, 2)}
                    </span>
                    <div className="min-w-0 flex-1">
                      <div className="font-mono text-xs text-cyan-100">{it.type}</div>
                      <div className="truncate text-[10px] text-cyan-300/55">{it.desc}</div>
                    </div>
                    <span className="shrink-0 font-display text-sm font-bold" style={{ color: it.color }}>
                      {typeCounts[it.type] ?? 0}
                    </span>
                  </div>
                ))}
              </div>
            </GlassPanel>

            {/* 节点状态 */}
            <GlassPanel title="NODE STATUS · 节点状态" meta="COLOR" strong>
              {NODE_STATUS.map((it) => (
                <div
                  key={it.key}
                  className="flex items-center gap-3 border-b border-cyan-500/8 px-3 py-2.5"
                >
                  <span
                    className="size-3 shrink-0 rounded-full"
                    style={{ background: it.color, boxShadow: `0 0 8px ${it.color}` }}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="font-mono text-xs text-cyan-100">{it.label}</div>
                    <div className="truncate text-[10px] text-cyan-300/55">{it.desc}</div>
                  </div>
                  <span className="shrink-0 font-display text-sm font-bold" style={{ color: it.color }}>
                    {statusCounts[it.key] ?? 0}
                  </span>
                </div>
              ))}
            </GlassPanel>

            {/* 告警级别 */}
            <GlassPanel title="ALARM SEVERITY · 告警级别" meta="SEVERITY" strong>
              {ALARM_SEVERITY.map((it) => (
                <div
                  key={it.label}
                  className="flex items-center gap-3 border-b border-cyan-500/8 px-3 py-2.5"
                >
                  <span
                    className="chip shrink-0"
                    style={{ color: it.color, borderColor: `${it.color}66` }}
                  >
                    <span
                      className="size-1.5 rounded-full"
                      style={{ background: it.color, boxShadow: `0 0 6px ${it.color}` }}
                    />
                    {it.label}
                  </span>
                  <span className="truncate text-[10px] text-cyan-300/55">{it.desc}</span>
                </div>
              ))}
            </GlassPanel>

            {/* 链路类型 */}
            <GlassPanel title="EDGE TYPE · 链路类型" meta="LINK" strong>
              {EDGE_TYPES.map((it, i) => {
                const key = (['active', 'degraded', 'inactive'] as const)[i]
                return (
                  <div
                    key={it.label}
                    className="flex items-center gap-3 border-b border-cyan-500/8 px-3 py-2.5"
                  >
                    <svg width={48} height={12} className="shrink-0">
                      <line
                        x1={0}
                        y1={6}
                        x2={48}
                        y2={6}
                        stroke={it.color}
                        strokeWidth={2}
                        strokeDasharray={it.dash}
                        style={{ filter: `drop-shadow(0 0 3px ${it.color})` }}
                      />
                    </svg>
                    <div className="min-w-0 flex-1">
                      <div className="font-mono text-xs text-cyan-100">{it.label}</div>
                      <div className="truncate text-[10px] text-cyan-300/55">{it.desc}</div>
                    </div>
                    <span className="shrink-0 font-display text-sm font-bold" style={{ color: it.color }}>
                      {edgeCounts[key] ?? 0}
                    </span>
                  </div>
                )
              })}
            </GlassPanel>
          </div>
        </>
      )}
    </PageShell>
  )
}
