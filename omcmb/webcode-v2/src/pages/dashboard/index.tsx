import { Cpu, AlertTriangle, Signal } from 'lucide-react'

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

export function DashboardPage() {
  return (
    <div className="mx-auto max-w-7xl px-6 py-8">
      <div className="mb-8">
        <h1 className="text-2xl font-semibold tracking-tight">控制台</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          v2 骨架页 · 共享 @omc/frontend-core 业务层 · 后续按模块补齐
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <StatCard icon={<Cpu className="text-blue-500" />} label="设备总数" value="—" />
        <StatCard icon={<Signal className="text-emerald-500" />} label="在线率" value="—" />
        <StatCard icon={<AlertTriangle className="text-amber-500" />} label="活动告警" value="—" />
      </div>

      <Card className="mt-8">
        <CardHeader>
          <CardTitle>下一步</CardTitle>
          <CardDescription>
            沿 backlog T-0035 逐模块补齐；已打通 login + dashboard + devices 三页
          </CardDescription>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground">
          <ul className="list-disc space-y-1 pl-4">
            <li>
              <a href="/devices" className="text-primary hover:underline">
                设备管理
              </a>
              （TanStack Table + @core/hooks/api/useDeviceList 已接入）
            </li>
            <li>告警中心（实时数据 + shadcn 形态）</li>
            <li>主题切换（暗色模式已内置 CSS vars）</li>
          </ul>
        </CardContent>
      </Card>
    </div>
  )
}

function StatCard({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode
  label: string
  value: string
}) {
  return (
    <Card>
      <CardContent className="flex items-center gap-4 p-6">
        <div className="grid size-10 place-items-center rounded-lg bg-muted [&_svg]:size-5">
          {icon}
        </div>
        <div>
          <div className="text-xs uppercase tracking-wider text-muted-foreground">
            {label}
          </div>
          <div className="text-xl font-semibold">{value}</div>
        </div>
      </CardContent>
    </Card>
  )
}
