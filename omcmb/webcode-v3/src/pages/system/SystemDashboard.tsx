import { RefreshCcw, Database, Server, Clock, Cpu, AlertTriangle } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { formatTime } from '@/lib/format'
import { useSystemInfo } from '@core/hooks/api/useSystem'
import { useDashboardSummary } from '@core/hooks/api/useDashboard'

import { StateBlock, MiniStat, FieldRow } from './_shared'

function healthBadge(status?: string): { badge: string; label: string } {
  const s = (status || '').toLowerCase()
  if (s === 'ok' || s === 'up' || s === 'healthy' || s === 'connected') {
    return { badge: 'online', label: status || 'OK' }
  }
  if (!status) return { badge: 'unknown', label: 'UNKNOWN' }
  return { badge: 'critical', label: status }
}

export default function SystemDashboard() {
  const { data: info, isLoading, isError, error, isFetching, refetch } = useSystemInfo()
  const { data: summary } = useDashboardSummary()

  const db = healthBadge(info?.dbStatus)
  const cache = healthBadge(info?.cacheStatus)
  const dbOk = db.badge === 'online'
  const cacheOk = cache.badge === 'online'
  const healthScore = info ? (dbOk ? 50 : 0) + (cacheOk ? 50 : 0) : 0

  const dev = summary?.deviceCounts
  const alarms = summary?.alarmCounts

  return (
    <PageShell
      code="F06"
      title="SYS DASH · 系统态势"
      subtitle="SYSTEM HEALTH OVERVIEW"
      isFetching={isFetching}
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      <StateBlock
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={!info}
        emptyLabel="NO SYSTEM INFO · 无系统信息"
      >
        {info ? (
          <div className="space-y-5">
            <div className="grid gap-4 lg:grid-cols-[280px_1fr]">
              <GlassPanel strong title="核心健康 · CORE HEALTH" className="p-4">
                <div className="flex flex-col items-center gap-3 py-3">
                  <RadialGauge
                    value={healthScore}
                    label="HEALTH"
                    color={
                      healthScore >= 100 ? '#00ff88' : healthScore >= 50 ? '#ffaa00' : '#ff2d6f'
                    }
                  />
                  <div className="flex gap-2">
                    <StatusBadge status={db.badge} label={`DB ${db.label}`} />
                    <StatusBadge status={cache.badge} label={`CACHE ${cache.label}`} />
                  </div>
                </div>
              </GlassPanel>

              <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
                <MiniStat
                  label="运行时长 · HRS"
                  value={info.uptimeHours?.toLocaleString?.() ?? info.uptimeHours}
                  color="#00f0ff"
                  icon={<Clock className="size-3.5" />}
                />
                <MiniStat
                  label="数据库 · DB"
                  value={db.label}
                  color={dbOk ? '#00ff88' : '#ff2d6f'}
                  icon={<Database className="size-3.5" />}
                />
                <MiniStat
                  label="缓存 · CACHE"
                  value={cache.label}
                  color={cacheOk ? '#00ff88' : '#ff2d6f'}
                  icon={<Cpu className="size-3.5" />}
                />
                <MiniStat
                  label="版本 · VER"
                  value={info.version || '—'}
                  color="#a855f7"
                  icon={<Server className="size-3.5" />}
                />
                {dev ? (
                  <>
                    <MiniStat label="设备总数" value={dev.total} color="#00f0ff" />
                    <MiniStat label="在线 · ONLINE" value={dev.online} color="#00ff88" />
                    <MiniStat label="离线 · OFFLINE" value={dev.offline} color="#525a78" />
                    <MiniStat
                      label="活动告警"
                      value={alarms?.total ?? 0}
                      color={(alarms?.total ?? 0) > 0 ? '#ff7a1a' : '#525a78'}
                      icon={
                        (alarms?.total ?? 0) > 0 ? <AlertTriangle className="size-3.5" /> : undefined
                      }
                    />
                  </>
                ) : null}
              </div>
            </div>

            <GlassPanel strong title="系统信息 · SYSTEM INFO" className="p-4">
              <div className="grid gap-x-8 md:grid-cols-2">
                <div>
                  <FieldRow label="版本 · VERSION">{info.version || '—'}</FieldRow>
                  <FieldRow label="构建日期">{info.buildDate || '—'}</FieldRow>
                  <FieldRow label="服务器时间">
                    {info.serverTime ? formatTime(info.serverTime) : '—'}
                  </FieldRow>
                </div>
                <div>
                  <FieldRow label="运行时长（小时）">{info.uptimeHours ?? '—'}</FieldRow>
                  <FieldRow label="数据库状态">
                    <StatusBadge status={db.badge} label={db.label} />
                  </FieldRow>
                  <FieldRow label="缓存状态">
                    <StatusBadge status={cache.badge} label={cache.label} />
                  </FieldRow>
                </div>
              </div>
            </GlassPanel>
          </div>
        ) : null}
      </StateBlock>
    </PageShell>
  )
}
