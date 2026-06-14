import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, RotateCcw, Settings2, Loader2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatTime } from '@/lib/format'
import {
  useProvisioningTasks,
  useRetryProvisioningTask,
} from '@core/hooks/api/useProvisioning'
import type { ProvisioningTask } from '@core/services/api/provisionApi'
import { provisioningStatusCode } from '@core/services/api/provisionApi'
import { StateGate, StatCard, Pager } from './_shared'

const PAGE_SIZE = 20

// UI 简化状态码 → 展示元数据
const STATUS_META: Record<string, { label: string; color: string }> = {
  '0': { label: 'COMPLETED · 成功', color: '#00ff88' },
  '1': { label: 'FAILED · 失败', color: '#ff2d6f' },
  '2': { label: 'RUNNING · 执行中', color: '#00f0ff' },
  '3': { label: 'PENDING · 未执行', color: '#525a78' },
  '4': { label: 'SKIPPED · 跳过', color: '#a855f7' },
}

const STATUS_FILTER: { v: string; t: string }[] = [
  { v: '', t: 'ALL' },
  { v: 'completed', t: '成功' },
  { v: 'failed', t: '失败' },
]

export default function FleetCommissioning() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState('')

  const params = useMemo(
    () => ({ page, pageSize: PAGE_SIZE, ...(status ? { status } : {}) }),
    [page, status]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useProvisioningTasks(params)
  const retry = useRetryProvisioningTask()

  const items: ProvisioningTask[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const summary = useMemo(() => {
    const acc = { done: 0, failed: 0, running: 0 }
    for (const t of items) {
      const code = provisioningStatusCode(t.status)
      if (code === '0') acc.done++
      else if (code === '1') acc.failed++
      else acc.running++
    }
    return acc
  }, [items])

  return (
    <PageShell
      code="F09"
      title="COMMISSIONING · 开通调测"
      subtitle="AUTO-PROVISION TASK FEED · 10s AUTO-REFRESH"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          {STATUS_FILTER.map((s) => (
            <button
              key={s.v || 'all'}
              type="button"
              onClick={() => {
                setStatus(s.v)
                setPage(1)
              }}
              className={`chip transition-all ${
                status === s.v ? 'text-cyan-200 shadow-[0_0_10px_currentColor]' : 'text-cyan-300/45 hover:text-cyan-300/80'
              }`}
            >
              {s.t}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-4">
        <StatCard label="TOTAL" value={total} color="#00f0ff" />
        <StatCard label="COMPLETED" value={summary.done} color="#00ff88" />
        <StatCard label="RUNNING" value={summary.running} color="#a855f7" />
        <StatCard label="FAILED" value={summary.failed} color="#ff2d6f" />
      </div>

      <StateGate
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={items.length === 0}
        loadingLabel="SYNCING TASKS…"
        emptyLabel="NO PROVISIONING TASKS"
      >
        <div className="space-y-2">
          {items.map((t) => {
            const code = provisioningStatusCode(t.status)
            const meta = STATUS_META[code] ?? STATUS_META['3']
            const progress = t.totalSteps > 0 ? (t.currentStep / t.totalSteps) * 100 : 0
            return (
              <div
                key={t.id}
                className="fleet-row grid grid-cols-[16px_1.5fr_1fr_1fr_140px_88px] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: meta.color }}
              >
                <Settings2 className="size-4 text-cyan-400/70" />
                <button
                  type="button"
                  className="min-w-0 text-left"
                  onClick={() => navigate(`/fleet/detail/${encodeURIComponent(t.deviceId)}`)}
                >
                  <div className="truncate font-display text-sm font-bold text-cyan-100">
                    TASK {t.id.slice(0, 8)}
                  </div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">
                    DEV {t.deviceId}
                  </div>
                </button>
                <div className="min-w-0">
                  <div className="mb-1 font-mono text-[10px] text-cyan-300/65">
                    STEP {t.currentStep}/{t.totalSteps || '—'}
                  </div>
                  <div className="h-1.5 overflow-hidden rounded-full bg-cyan-500/10">
                    <div
                      className="h-full rounded-full"
                      style={{ width: `${progress}%`, background: meta.color, boxShadow: `0 0 6px ${meta.color}` }}
                    />
                  </div>
                </div>
                <div className="font-mono text-[10px] text-cyan-300/65">
                  <div className="text-cyan-100/85">{formatTime(t.startedAt)}</div>
                  <div>{t.errorMessage ? t.errorMessage.slice(0, 32) : '—'}</div>
                </div>
                <div className="text-right">
                  <span className="chip" style={{ color: meta.color }}>
                    {meta.label}
                  </span>
                </div>
                <div className="flex items-center justify-end">
                  {code === '1' ? (
                    <NeonButton
                      className="!py-1 !px-2"
                      icon={retry.isPending ? <Loader2 className="animate-spin" /> : <RotateCcw />}
                      disabled={retry.isPending}
                      onClick={() => retry.mutate(t.id)}
                    >
                      RETRY
                    </NeonButton>
                  ) : (
                    <span className="font-mono text-[10px] text-cyan-300/40">
                      ×{t.retryCount}/{t.maxRetries}
                    </span>
                  )}
                </div>
              </div>
            )
          })}
        </div>

        <Pager
          page={page}
          totalPages={totalPages}
          total={total}
          pageSize={PAGE_SIZE}
          onPrev={() => setPage((p) => Math.max(1, p - 1))}
          onNext={() => setPage((p) => Math.min(totalPages, p + 1))}
        />
      </StateGate>
    </PageShell>
  )
}
