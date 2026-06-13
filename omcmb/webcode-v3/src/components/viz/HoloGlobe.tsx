import { useMemo } from 'react'

interface Dot {
  lat: number // -90 ~ 90
  lng: number // -180 ~ 180
  level: 'ok' | 'warn' | 'crit'
}

interface Props {
  size?: number
  dots?: Dot[]
}

const DEFAULT_DOTS: Dot[] = [
  { lat: 39.9, lng: 116.4, level: 'ok' }, // 北京
  { lat: 31.2, lng: 121.5, level: 'ok' }, // 上海
  { lat: 22.3, lng: 114.2, level: 'warn' }, // 香港
  { lat: 23.1, lng: 113.3, level: 'ok' }, // 广州
  { lat: 30.6, lng: 104.1, level: 'crit' }, // 成都
  { lat: 34.3, lng: 108.9, level: 'ok' }, // 西安
  { lat: 36.1, lng: 120.4, level: 'ok' }, // 青岛
  { lat: 28.2, lng: 113.0, level: 'warn' }, // 长沙
  { lat: 25.0, lng: 102.7, level: 'ok' }, // 昆明
  { lat: 43.8, lng: 87.6, level: 'crit' }, // 乌鲁木齐
  { lat: 40.7, lng: -74.0, level: 'ok' }, // 纽约
  { lat: 35.7, lng: 139.7, level: 'ok' }, // 东京
  { lat: 51.5, lng: -0.1, level: 'warn' }, // 伦敦
  { lat: 1.35, lng: 103.8, level: 'ok' }, // 新加坡
  { lat: -33.9, lng: 151.2, level: 'ok' }, // 悉尼
  { lat: 55.8, lng: 37.6, level: 'warn' }, // 莫斯科
  { lat: -23.5, lng: -46.6, level: 'crit' }, // 圣保罗
  { lat: 19.4, lng: -99.1, level: 'ok' }, // 墨西哥城
  { lat: 28.6, lng: 77.2, level: 'ok' }, // 新德里
  { lat: -1.3, lng: 36.8, level: 'warn' }, // 内罗毕
  { lat: 30.0, lng: 31.2, level: 'ok' }, // 开罗
  { lat: 41.0, lng: 28.9, level: 'ok' }, // 伊斯坦布尔
]

/**
 * 全息地球：CSS 3D 自转 + 经纬网格 + 基站发光点（投影到 2D 椭圆，简化版）
 * 旋转通过 spinSlow 关键帧实现，发光点跟随容器旋转。
 */
export function HoloGlobe({ size = 380, dots = DEFAULT_DOTS }: Props) {
  // 在 2D 圆面上做正交投影（简化）：x = R * cos(lat) * sin(lng), y = -R * sin(lat)
  const projected = useMemo(() => {
    return dots.map((d, i) => {
      const lat = (d.lat * Math.PI) / 180
      const lng = (d.lng * Math.PI) / 180
      const x = Math.cos(lat) * Math.sin(lng) // -1..1
      const y = -Math.sin(lat) // -1..1
      const z = Math.cos(lat) * Math.cos(lng) // -1..1
      return {
        i,
        level: d.level,
        // 投影到 [0%, 100%]
        left: 50 + x * 42,
        top: 50 + y * 42,
        // z 用来判断是否在背面（淡化）
        front: z >= 0,
      }
    })
  }, [dots])

  return (
    <div
      className="holo-globe"
      style={{ width: size, height: size }}
      aria-hidden
    >
      {/* 外圈光晕 */}
      <div className="holo-globe-glow" />

      {/* 旋转的本体 */}
      <div className="holo-globe-sphere">
        {/* 主圈 */}
        <div className="holo-globe-ring" />

        {/* 经线 — 5 条 */}
        {[0, 36, 72, 108, 144].map((deg) => (
          <div
            key={`mer-${deg}`}
            className="holo-globe-meridian"
            style={{ transform: `rotateY(${deg}deg)` }}
          />
        ))}

        {/* 纬线 — 用 ring 缩放 */}
        {[15, 30, 45, 60, 75].map((lat) => {
          const radPct = Math.cos((lat * Math.PI) / 180) * 100
          const yPct = Math.sin((lat * Math.PI) / 180) * 50
          return (
            <div key={`par-${lat}`}>
              <div
                className="holo-globe-ring"
                style={{
                  width: `${radPct}%`,
                  height: '2px',
                  borderTop: '1px dashed rgba(0,240,255,0.18)',
                  border: 'none',
                  borderRadius: 0,
                  background: 'transparent',
                  top: `${50 - yPct}%`,
                  left: `${(100 - radPct) / 2}%`,
                  position: 'absolute',
                }}
              />
              <div
                className="holo-globe-ring"
                style={{
                  width: `${radPct}%`,
                  height: '2px',
                  borderTop: '1px dashed rgba(0,240,255,0.18)',
                  border: 'none',
                  borderRadius: 0,
                  background: 'transparent',
                  top: `${50 + yPct}%`,
                  left: `${(100 - radPct) / 2}%`,
                  position: 'absolute',
                }}
              />
            </div>
          )
        })}

        {/* 基站发光点 */}
        {projected.map((p) => (
          <span
            key={p.i}
            className={`holo-dot ${p.level}`}
            style={{
              left: `${p.left}%`,
              top: `${p.top}%`,
              opacity: p.front ? 1 : 0.18,
              transform: `translate(-50%, -50%) scale(${p.front ? 1 : 0.6})`,
            }}
          />
        ))}
      </div>

      {/* 静止的十字准星 + 角标 */}
      <div className="absolute inset-0 pointer-events-none">
        <div className="absolute left-1/2 top-0 bottom-0 w-px bg-cyan-400/15" />
        <div className="absolute top-1/2 left-0 right-0 h-px bg-cyan-400/15" />
        <Corners />
      </div>
    </div>
  )
}

function Corners() {
  const c = 'absolute size-3 border-cyan-400/70'
  return (
    <>
      <span className={`${c} left-0 top-0 border-l-2 border-t-2`} />
      <span className={`${c} right-0 top-0 border-r-2 border-t-2`} />
      <span className={`${c} left-0 bottom-0 border-l-2 border-b-2`} />
      <span className={`${c} right-0 bottom-0 border-r-2 border-b-2`} />
    </>
  )
}
