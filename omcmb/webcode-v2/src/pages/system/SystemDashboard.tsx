import { useMemo } from 'react'
import {
  AlertTriangle,
  Boxes,
  CheckCircle2,
  CloudCog,
  Cpu,
  Database,
  HardDrive,
  ListChecks,
  Server,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { PageShell, formatTime } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useSystemInfo } from '@core/hooks/api/useSystem'
import { useDashboardSummary } from '@core/hooks/api/useDashboard'

// ============================================================
// 系统管理 / 系统概览 — 对齐 v1 webcode/src/pages/system/SystemDashboard
// 真实数据 useSystemInfo（版本 / 运行时长 / DB / 缓存）+ useDashboardSummary
//   （设备 / 告警 / 任务统计）。v1 的 CPU/内存/磁盘仪表为占位 mock，后端无对应字段，
// 这里改为渲染真实的设备 / 告警 / 任务概览。详见返回 notes。
// ============================================================

function StatusDot({ ok }: { ok: boolean }) {
  return (
    <span
      className={cn(
        'mr-1.5 inline-block size-2 rounded-full',
        ok ? 'bg-emerald-500' : 'bg-destructive'
      )}
    />
  )
}

function StatCard({
  icon: Icon,
  label,
  value,
  sub,
  tone = 'default',
}: {
  icon: typeof Server
  label: string
  value: React.ReactNode
  sub?: React.ReactNode
  tone?: 'default' | 'emerald' | 'amber' | 'destructive'
}) {
  const toneClass = {
    default: 'text-foreground',
    emerald: 'text-emerald-600 dark:text-emerald-400',
    amber: 'text-amber-600 dark:text-amber-400',
    destructive: 'text-destructive',
  }[tone]
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="flex items-center gap-1.5 text-xs uppercase tracking-wider text-muted-foreground">
        <Icon className="size-3.5" />
        {label}
      </div>
      <div className={cn('mt-1.5 text-2xl font-semibold tabular-nums', toneClass)}>
        {value}
      </div>
      {sub ? <div className="mt-0.5 text-[11px] text-muted-foreground">{sub}</div> : null}
    </div>
  )
}

export default function SystemDashboard() {
  const infoQuery = useSystemInfo()
  const summaryQuery = useDashboardSummary()

  const info = infoQuery.data
  const summary = summaryQuery.data

  const dbOk = useMemo(() => {
    const s = (info?.dbStatus ?? '').toLowerCase()
    return s === 'ok' || s === 'up' || s === 'healthy'
  }, [info])
  const cacheOk = useMemo(() => {
    const s = (info?.cacheStatus ?? '').toLowerCase()
    return s === 'ok' || s === 'up' || s === 'healthy'
  }, [info])

  const dev = summary?.deviceCounts
  const alarms = summary?.alarmCounts
  const tasks = summary?.taskSummary

  return (
    <PageShell
      title="系统概览"
      description="系统运行状态 · 设备 / 告警 / 任务汇总"
      isFetching={infoQuery.isFetching || summaryQuery.isFetching}
    >
      {/* 进程信息 */}
      <div className="mb-2 text-sm font-medium">运行信息</div>
      <div className="mb-6 grid grid-cols-2 gap-3 md:grid-cols-4">
        <StatCard
          icon={Server}
          label="版本"
          value={
            <span className="text-lg">{info?.version || '—'}</span>
          }
          sub={info?.buildDate ? `构建于 ${formatTime(info.buildDate)}` : undefined}
        />
        <StatCard
          icon={HardDrive}
          label="运行时长"
          value={
            <span className="text-lg">
              {info?.uptimeHours != null
                ? `${info.uptimeHours.toFixed(1)} 小时`
                : '—'}
            </span>
          }
        />
        <StatCard
          icon={Database}
          label="数据库"
          value={
            <span className="inline-flex items-center text-lg">
              <StatusDot ok={dbOk} />
              <span className={dbOk ? 'text-emerald-600 dark:text-emerald-400' : 'text-destructive'}>
                {info?.dbStatus || '—'}
              </span>
            </span>
          }
        />
        <StatCard
          icon={Boxes}
          label="缓存"
          value={
            <span className="inline-flex items-center text-lg">
              <StatusDot ok={cacheOk} />
              <span className={cacheOk ? 'text-emerald-600 dark:text-emerald-400' : 'text-destructive'}>
                {info?.cacheStatus || '—'}
              </span>
            </span>
          }
        />
      </div>

      {summaryQuery.isError ? (
        <div className="rounded-lg border bg-card px-4 py-3 text-sm text-muted-foreground">
          汇总数据暂不可用
        </div>
      ) : (
        <>
          {/* 设备 */}
          <div className="mb-2 text-sm font-medium">设备</div>
          <div className="mb-6 grid grid-cols-2 gap-3 md:grid-cols-4">
            <StatCard icon={CloudCog} label="全部设备" value={dev?.total ?? '—'} />
            <StatCard icon={CheckCircle2} label="在线" value={dev?.online ?? '—'} tone="emerald" />
            <StatCard icon={Cpu} label="离线" value={dev?.offline ?? '—'} />
            <StatCard icon={AlertTriangle} label="有告警" value={dev?.alarm ?? '—'} tone="amber" />
          </div>

          {/* 告警 */}
          <div className="mb-2 text-sm font-medium">活动告警</div>
          <div className="mb-6 grid grid-cols-2 gap-3 md:grid-cols-5">
            <StatCard icon={AlertTriangle} label="紧急" value={alarms?.critical ?? '—'} tone="destructive" />
            <StatCard icon={AlertTriangle} label="重要" value={alarms?.major ?? '—'} tone="destructive" />
            <StatCard icon={AlertTriangle} label="次要" value={alarms?.minor ?? '—'} tone="amber" />
            <StatCard icon={AlertTriangle} label="警告" value={alarms?.warning ?? '—'} tone="amber" />
            <StatCard icon={ListChecks} label="合计" value={alarms?.total ?? '—'} />
          </div>

          {/* 任务 */}
          <div className="mb-2 text-sm font-medium">任务</div>
          <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
            <StatCard icon={ListChecks} label="执行中" value={tasks?.running ?? '—'} />
            <StatCard icon={ListChecks} label="待处理" value={tasks?.pending ?? '—'} tone="amber" />
            <StatCard icon={CheckCircle2} label="成功" value={tasks?.success ?? '—'} tone="emerald" />
            <StatCard icon={AlertTriangle} label="失败" value={tasks?.failed ?? '—'} tone="destructive" />
          </div>

          {summary ? (
            <div className="mt-6 flex items-center gap-2">
              <Badge variant="muted">数据每 30 秒自动刷新</Badge>
            </div>
          ) : null}
        </>
      )}
    </PageShell>
  )
}
