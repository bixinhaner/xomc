import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { PageShell } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

// ============================================================
// 图例系统（v2 皮肤）— 对照 v1 webcode/pages/topology/LegendSystem
//  纯静态图例参考：设备类型 / 节点状态 / 告警级别 / 边类型。
//  无异步数据，故不涉及三态；色板与 v1 拓扑/GIS 图例一致。
// ============================================================

interface DeviceTypeItem {
  label: string
  className: string
  shape: 'circle' | 'rect' | 'diamond'
  desc: string
}

const DEVICE_TYPES: DeviceTypeItem[] = [
  { label: 'eNB', className: 'bg-blue-100 text-blue-700 border-blue-300', shape: 'circle', desc: 'LTE 基站' },
  { label: 'gNB', className: 'bg-purple-100 text-purple-700 border-purple-300', shape: 'circle', desc: '5G NR 基站' },
  { label: 'CPE', className: 'bg-cyan-100 text-cyan-700 border-cyan-300', shape: 'circle', desc: '用户终端设备' },
  { label: 'eGW', className: 'bg-orange-100 text-orange-700 border-orange-300', shape: 'diamond', desc: '边缘网关' },
  { label: 'RT', className: 'bg-green-100 text-green-700 border-green-300', shape: 'rect', desc: '路由器' },
  { label: 'SW', className: 'bg-red-100 text-red-700 border-red-300', shape: 'rect', desc: '交换机' },
  { label: '域', className: 'bg-gray-100 text-gray-700 border-gray-300', shape: 'circle', desc: '设备域 / 分组' },
  { label: '站', className: 'bg-amber-100 text-amber-700 border-amber-300', shape: 'circle', desc: '物理站点' },
]

interface ColorItem {
  label: string
  dot: string
  desc: string
}

const NODE_STATUS: ColorItem[] = [
  { label: '在线', dot: 'bg-emerald-500', desc: '节点正常在线' },
  { label: '离线', dot: 'bg-gray-400', desc: '节点失联 / 断开' },
  { label: '告警', dot: 'bg-red-500', desc: '存在活动告警' },
  { label: '维护', dot: 'bg-yellow-500', desc: '处于维护状态' },
]

interface SeverityItem {
  label: string
  badge: string
  desc: string
}

const ALARM_SEVERITY: SeverityItem[] = [
  { label: '紧急', badge: 'bg-red-700 text-white', desc: 'Critical · 立即处理' },
  { label: '重要', badge: 'bg-red-500 text-white', desc: 'Major · 尽快处理' },
  { label: '次要', badge: 'bg-orange-500 text-white', desc: 'Minor · 关注' },
  { label: '警告', badge: 'bg-yellow-500 text-white', desc: 'Warning · 提示' },
  { label: '提示', badge: 'bg-blue-500 text-white', desc: 'Notice · 通知' },
]

interface EdgeItem {
  label: string
  stroke: string
  dash?: string
  desc: string
}

const EDGE_TYPES: EdgeItem[] = [
  { label: '正常连接', stroke: '#10b981', desc: '链路活跃 (active)' },
  { label: '降级连接', stroke: '#f59e0b', dash: '8,4', desc: '链路降级 (degraded)' },
  { label: '断开连接', stroke: '#94a3b8', dash: '4,4', desc: '链路非活跃 (inactive)' },
]

function shapeClass(shape: DeviceTypeItem['shape']) {
  if (shape === 'circle') return 'rounded-full'
  if (shape === 'rect') return 'rounded-md'
  return 'rotate-45 rounded-sm'
}

export default function LegendSystem() {
  return (
    <PageShell title="图例系统" description="拓扑 / GIS 视图符号与色板参考">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {/* 设备类型 */}
        <Card>
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-base font-medium">设备类型</CardTitle>
          </CardHeader>
          <CardContent className="p-4 pt-2">
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              {DEVICE_TYPES.map((item) => (
                <div key={item.label} className="flex items-center gap-3">
                  <div
                    className={cn(
                      'flex size-9 shrink-0 items-center justify-center border-2 text-[10px] font-bold',
                      item.className,
                      shapeClass(item.shape),
                    )}
                  >
                    <span className={item.shape === 'diamond' ? '-rotate-45' : ''}>{item.label}</span>
                  </div>
                  <div>
                    <div className="text-sm font-medium">{item.label}</div>
                    <div className="text-xs text-muted-foreground">{item.desc}</div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* 节点状态 */}
        <Card>
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-base font-medium">节点状态</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 p-4 pt-2">
            {NODE_STATUS.map((item) => (
              <div key={item.label} className="flex items-center gap-3">
                <span className={cn('size-4 shrink-0 rounded-full', item.dot)} />
                <div>
                  <div className="text-sm font-medium">{item.label}</div>
                  <div className="text-xs text-muted-foreground">{item.desc}</div>
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        {/* 告警级别 */}
        <Card>
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-base font-medium">告警级别</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 p-4 pt-2">
            {ALARM_SEVERITY.map((item) => (
              <div key={item.label} className="flex items-center gap-3">
                <span
                  className={cn(
                    'inline-flex min-w-12 justify-center rounded px-2 py-0.5 text-xs font-semibold',
                    item.badge,
                  )}
                >
                  {item.label}
                </span>
                <div className="text-xs text-muted-foreground">{item.desc}</div>
              </div>
            ))}
          </CardContent>
        </Card>

        {/* 边类型 */}
        <Card>
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-base font-medium">边类型</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 p-4 pt-2">
            {EDGE_TYPES.map((item) => (
              <div key={item.label} className="flex items-center gap-3">
                <svg width={48} height={16} className="shrink-0">
                  <line
                    x1={0}
                    y1={8}
                    x2={44}
                    y2={8}
                    stroke={item.stroke}
                    strokeWidth={2.5}
                    strokeDasharray={item.dash}
                  />
                  <polygon points="44,4 48,8 44,12" fill={item.stroke} />
                </svg>
                <div>
                  <div className="text-sm font-medium">{item.label}</div>
                  <div className="text-xs text-muted-foreground">{item.desc}</div>
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>
    </PageShell>
  )
}
