import { LogOut, Activity, Cpu, AlertTriangle, Signal } from 'lucide-react'
import { useNavigate } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

import { useUserStore } from '@core/store/userStore'

export function DashboardPage() {
  const navigate = useNavigate()
  const user = useUserStore((s) => s.currentUser)
  const clearAuth = useUserStore((s) => s.clearAuth)

  const onLogout = () => {
    clearAuth()
    navigate('/login')
  }

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border/60 bg-card/50 backdrop-blur">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-6">
          <div className="flex items-center gap-2">
            <Activity className="size-5 text-primary" />
            <span className="font-semibold tracking-wide">OMC · v2</span>
            <span className="ml-2 rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground">
              scaffold
            </span>
          </div>
          <div className="flex items-center gap-3 text-sm">
            <span className="text-muted-foreground">
              {user?.displayName || user?.username || '未登录'}
            </span>
            <Button size="sm" variant="ghost" onClick={onLogout}>
              <LogOut /> 退出
            </Button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-6 py-8">
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
              沿 backlog T-0035 逐模块补齐；login + dashboard 壳验证端到端架构可工作
            </CardDescription>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground">
            <ul className="list-disc space-y-1 pl-4">
              <li>Device 模块（首个复杂页面，验证 TanStack Table + @core hooks）</li>
              <li>Alarm 模块（实时数据 + shadcn 形态）</li>
              <li>主题切换（暗色模式已内置 CSS vars）</li>
            </ul>
          </CardContent>
        </Card>
      </main>
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
