import { useState } from 'react'
import { Search, RefreshCcw, Zap, RotateCcw, Loader2, Rocket } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useProvisioningTasks, useRetryProvisioningTask } from '@core/hooks/api/useProvisioning'
import type { ProvisioningTask } from '@core/services/api/provisionApi'
import { formatTime } from '@/lib/format'
import { StatCard, HudLoading, HudError, HudEmpty } from './_hud'

// 后端生命周期状态 → HUD 视觉。
const STATUS_VIS: Record<string, { label: string; badge: string; color: string }> = {
  completed: { label: '成功', badge: 'ok', color: '#00ff88' },
  failed: { label: '失败', badge: 'critical', color: '#ff2d6f' },
  discovered: { label: '已发现', badge: 'unknown', color: '#5b9eff' },
  identifying: { label: '识别中', badge: 'active', color: '#00f0ff' },
  matching: { label: '匹配中', badge: 'active', color: '#00f0ff' },
  configuring: { label: '配置中', badge: 'active', color: '#00f0ff' },
  verifying: { label: '校验中', badge: 'active', color: '#00f0ff' },
  discovering: { label: '探测中', badge: 'active', color: '#00f0ff' },
  syncing: { label: '同步中', badge: 'active', color: '#00f0ff' },
}

function vis(status: string) {
  return STATUS_VIS[status] ?? { label: status || '—', badge: 'warning', color: '#ffaa00' }
}

// ===========================================================================
// CONFIG · 自动开站
// 真实即插即用任务（useProvisioningTasks，10s 轮询）→ 状态机进度 +
// 失败可重试（useRetryProvisioningTask）。
// ===========================================================================
export default function AutoProvisioning() {
  const [keyword, setKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState('')

  const tasksQ = useProvisioningTasks({ page: 1, pageSize: 200, status: statusFilter || undefined })
  const retry = useRetryProvisioningTask()
  const tasks: ProvisioningTask[] = tasksQ.data?.items ?? []

  const kw = keyword.trim().toLowerCase()
  const rows = kw
    ? tasks.filter((t) => t.deviceId?.toLowerCase().includes(kw) || t.id?.toLowerCase().includes(kw))
    : tasks

  const completed = tasks.filter((t) => t.status === 'completed').length
  const failed = tasks.filter((t) => t.status === 'failed').length
  const inProgress = tasks.length - completed - failed

  const STATUSES = ['', 'completed', 'failed', 'configuring', 'verifying']

  return (
    <PageShell
      code="F09"
      title="AUTO PROVISION · 自动开站"
      subtitle="ZERO-TOUCH PROVISIONING"
      isFetching={tasksQ.isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="设备 ID / 任务 ID"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={() => void tasksQ.refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-4 gap-3">
        <StatCard label="TASKS · 任务" value={tasksQ.data?.total ?? tasks.length} color="#00f0ff" />
        <StatCard label="COMPLETED · 成功" value={completed} color="#00ff88" />
        <StatCard label="IN-PROGRESS · 进行中" value={inProgress} color="#ffaa00" />
        <StatCard label="FAILED · 失败" value={failed} color={failed > 0 ? '#ff2d6f' : '#525a78'} />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        {STATUSES.map((s) => (
          <button
            key={s || 'all'}
            type="button"
            onClick={() => setStatusFilter(s)}
            className={`chip ${statusFilter === s ? 'text-[#00f0ff] shadow-[0_0_10px_currentColor]' : 'text-[#6b86b6]'}`}
          >
            {s === '' ? '全部状态' : vis(s).label}
          </button>
        ))}
      </div>

      <GlassPanel title="PROVISIONING TASKS" meta={`${rows.length}`} className="min-h-0">
        {tasksQ.isLoading ? (
          <HudLoading />
        ) : tasksQ.isError ? (
          <HudError error={tasksQ.error} />
        ) : rows.length === 0 ? (
          <HudEmpty icon={Rocket} text="无开站任务" />
        ) : (
          <div className="max-h-[56vh] overflow-auto">
            <div className="grid grid-cols-[1.6fr_1.4fr_90px_90px_120px] gap-3 border-b border-cyan-500/15 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/45">
              <span>DEVICE / 设备</span>
              <span>PROGRESS / 进度</span>
              <span>RETRY</span>
              <span>STATUS</span>
              <span className="text-right">ACTION</span>
            </div>
            {rows.map((t) => {
              const v = vis(t.status)
              const pct =
                t.totalSteps > 0
                  ? Math.max(0, Math.min(100, Math.round((t.currentStep / t.totalSteps) * 100)))
                  : t.status === 'completed'
                    ? 100
                    : 0
              return (
                <div
                  key={t.id}
                  className="fleet-row grid grid-cols-[1.6fr_1.4fr_90px_90px_120px] items-center gap-3 rounded-sm px-3 py-2.5"
                  style={{ ['--row-color' as never]: v.color }}
                >
                  <div className="min-w-0">
                    <code className="block truncate font-mono text-xs text-cyan-100">{t.deviceId}</code>
                    <div className="truncate font-mono text-[10px] text-cyan-300/50">
                      {t.errorMessage || `创建 ${formatTime(t.createdAt)}`}
                    </div>
                  </div>
                  <div>
                    <div className="mb-1 flex items-center justify-between font-mono text-[10px] text-cyan-300/60">
                      <span>STEP {t.currentStep}/{t.totalSteps || '—'}</span>
                      <span>{pct}%</span>
                    </div>
                    <div className="h-1.5 overflow-hidden rounded-full bg-cyan-500/10">
                      <div
                        className="h-full rounded-full transition-all"
                        style={{ width: `${pct}%`, background: v.color, boxShadow: `0 0 8px ${v.color}` }}
                      />
                    </div>
                  </div>
                  <div className="font-mono text-[11px] text-cyan-300/65">
                    {t.retryCount}/{t.maxRetries}
                  </div>
                  <div>
                    <StatusBadge status={v.badge} label={v.label} />
                  </div>
                  <div className="text-right">
                    {t.status === 'failed' ? (
                      <NeonButton
                        icon={retry.isPending ? <Loader2 className="animate-spin" /> : <RotateCcw />}
                        onClick={() => retry.mutate(t.id)}
                        disabled={retry.isPending}
                      >
                        重试
                      </NeonButton>
                    ) : (
                      <span className="inline-flex items-center gap-1 font-mono text-[10px] text-cyan-300/40">
                        <Zap className="size-3" />
                        ZTP
                      </span>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </GlassPanel>
    </PageShell>
  )
}
