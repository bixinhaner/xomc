import { useMemo } from 'react'
import { Loader2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { useSites } from '@core/hooks/api/useTopology'
import type { Site } from '@core/types/topology'

const STATUS_COLOR: Record<string, string> = {
  active: '#00ff88',
  maintenance: '#ffaa00',
  inactive: '#525a78',
}

interface NodePos {
  id: string
  label: string
  cx: number
  cy: number
  status: string
  deviceCount: number
}

export function TopologyPage() {
  const { data, isLoading, isError } = useSites()
  const sites = useMemo<Site[]>(() => data?.items ?? [], [data])

  // 力导向布局太重，简化为同心圆 / 椭圆轨道分布
  const positions = useMemo<NodePos[]>(() => {
    if (sites.length === 0) return []
    const cx = 480
    const cy = 320
    return sites.map((s, i) => {
      const ring = Math.floor(i / 8)
      const idxInRing = i % 8
      const radius = 90 + ring * 80
      const angle = (idxInRing / 8) * Math.PI * 2 + ring * 0.4
      return {
        id: s.id,
        label: s.name,
        cx: cx + Math.cos(angle) * radius,
        cy: cy + Math.sin(angle) * radius,
        status: s.status,
        deviceCount: s.deviceCount,
      }
    })
  }, [sites])

  // 连线：每节点连到中心 + 相邻节点
  const edges = useMemo(() => {
    const out: { x1: number; y1: number; x2: number; y2: number; key: string }[] = []
    const cx = 480
    const cy = 320
    positions.forEach((p, i) => {
      out.push({ x1: cx, y1: cy, x2: p.cx, y2: p.cy, key: `c-${i}` })
      const next = positions[i + 1]
      if (next && Math.abs(next.cx - p.cx) + Math.abs(next.cy - p.cy) < 240) {
        out.push({ x1: p.cx, y1: p.cy, x2: next.cx, y2: next.cy, key: `n-${i}` })
      }
    })
    return out
  }, [positions])

  return (
    <PageShell
      code="F06"
      title="TOPOLOGY · 战场态势"
      subtitle="NETWORK GRAPH · 力导向同心轨道"
      bare
    >
      <div className="grid h-full grid-cols-[1fr_320px] gap-3">
        <GlassPanel title="GRAPH · 网络拓扑" meta={`${sites.length} NODES`} className="min-h-0">
          <div className="relative h-[calc(100%-40px)]">
            {isLoading ? (
              <Center>
                <Loader2 className="size-6 animate-spin text-cyan-300/70" />
              </Center>
            ) : isError ? (
              <Center>
                <span className="font-mono text-sm text-rose-300">SYNC FAILED</span>
              </Center>
            ) : sites.length === 0 ? (
              <Center>
                <span className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/40">
                  NO TOPOLOGY DATA
                </span>
              </Center>
            ) : (
              <svg
                viewBox="0 0 960 640"
                preserveAspectRatio="xMidYMid meet"
                className="size-full"
              >
                <defs>
                  <radialGradient id="centerGlow">
                    <stop offset="0%" stopColor="#00f0ff" stopOpacity="0.85" />
                    <stop offset="100%" stopColor="#00f0ff" stopOpacity="0" />
                  </radialGradient>
                  <filter id="glow">
                    <feGaussianBlur stdDeviation="2" />
                    <feMerge>
                      <feMergeNode />
                      <feMergeNode in="SourceGraphic" />
                    </feMerge>
                  </filter>
                </defs>

                {/* 同心轨道 */}
                {[90, 170, 250, 330].map((r) => (
                  <circle
                    key={r}
                    cx={480}
                    cy={320}
                    r={r}
                    fill="none"
                    stroke="rgba(0,240,255,0.12)"
                    strokeWidth={1}
                    strokeDasharray="2 6"
                  />
                ))}

                {/* 中心光晕 */}
                <circle cx={480} cy={320} r={60} fill="url(#centerGlow)" />
                <circle
                  cx={480}
                  cy={320}
                  r={14}
                  fill="#00f0ff"
                  filter="url(#glow)"
                />
                <text
                  x={480}
                  y={324}
                  textAnchor="middle"
                  className="fill-[#03050d] font-bold"
                  fontSize="10"
                >
                  ACS
                </text>

                {/* 边 */}
                {edges.map((e) => (
                  <line
                    key={e.key}
                    x1={e.x1}
                    y1={e.y1}
                    x2={e.x2}
                    y2={e.y2}
                    stroke="rgba(0,240,255,0.32)"
                    strokeWidth={1}
                    strokeDasharray="4 4"
                  >
                    <animate
                      attributeName="stroke-dashoffset"
                      from="0"
                      to="-32"
                      dur="3s"
                      repeatCount="indefinite"
                    />
                  </line>
                ))}

                {/* 节点 */}
                {positions.map((p) => (
                  <g key={p.id}>
                    <circle
                      cx={p.cx}
                      cy={p.cy}
                      r={18}
                      fill="none"
                      stroke={STATUS_COLOR[p.status] ?? '#525a78'}
                      strokeWidth={1}
                      opacity={0.4}
                      className="svg-node-ring"
                    />
                    <circle
                      cx={p.cx}
                      cy={p.cy}
                      r={8}
                      fill={STATUS_COLOR[p.status] ?? '#525a78'}
                      filter="url(#glow)"
                    />
                    <text
                      x={p.cx}
                      y={p.cy + 28}
                      textAnchor="middle"
                      fill="#d8e9ff"
                      fontSize="10"
                      fontFamily="JetBrains Mono, monospace"
                    >
                      {p.label.length > 12 ? p.label.slice(0, 12) + '…' : p.label}
                    </text>
                  </g>
                ))}
              </svg>
            )}
          </div>
        </GlassPanel>

        <GlassPanel title="SITES · 站点清单" meta="LIVE">
          <div className="max-h-[640px] overflow-auto">
            {sites.map((s) => (
              <div
                key={s.id}
                className="flex items-center gap-2 border-b border-cyan-500/8 px-3 py-2 hover:bg-cyan-500/5"
              >
                <span
                  className="size-2 rounded-full"
                  style={{
                    background: STATUS_COLOR[s.status] ?? '#525a78',
                    boxShadow: `0 0 6px ${STATUS_COLOR[s.status] ?? '#525a78'}`,
                  }}
                />
                <div className="min-w-0 flex-1">
                  <div className="truncate font-mono text-xs text-cyan-100">{s.name}</div>
                  <div className="truncate text-[10px] text-cyan-300/55">{s.address}</div>
                </div>
                <div className="font-display text-sm font-bold text-cyan-200">
                  {s.deviceCount}
                </div>
              </div>
            ))}
          </div>
        </GlassPanel>
      </div>
    </PageShell>
  )
}

function Center({ children }: { children: React.ReactNode }) {
  return <div className="flex h-full w-full items-center justify-center">{children}</div>
}
