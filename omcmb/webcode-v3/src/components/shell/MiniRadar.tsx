import { useMemo } from 'react'

const COLORS = ['#00f0ff', '#ff00aa', '#00ff88', '#ffaa00']
// 固定种子，避免 render 中调用 Math.random
const SEEDS = [
  { left: 22, top: 18 },
  { left: 70, top: 28 },
  { left: 36, top: 60 },
  { left: 80, top: 72 },
  { left: 52, top: 44 },
  { left: 18, top: 78 },
]

export function MiniRadar({ size = 144 }: { size?: number }) {
  const blips = useMemo(
    () => SEEDS.map((s, i) => ({ i, left: s.left, top: s.top, color: COLORS[i % COLORS.length] })),
    []
  )

  return (
    <div className="relative" style={{ width: size, height: size }}>
      <div className="radar size-full" />
      {/* 静态散点 */}
      {blips.map((b) => (
        <span
          key={b.i}
          className="absolute size-1.5 rounded-full animate-breathe"
          style={{
            left: `${b.left}%`,
            top: `${b.top}%`,
            background: b.color,
            color: b.color,
            boxShadow: `0 0 6px currentColor`,
          }}
        />
      ))}

      {/* 角标 */}
      <div className="absolute -top-3 left-0 font-mono text-[9px] tracking-[0.2em] text-cyan-300/60">
        SCAN · 360°
      </div>
      <div className="absolute -bottom-4 right-0 font-mono text-[9px] tracking-[0.2em] text-cyan-300/60">
        ◉ {blips.length} TGT
      </div>
    </div>
  )
}
