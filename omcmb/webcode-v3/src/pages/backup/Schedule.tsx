import { useMemo, useState } from 'react'
import { RefreshCcw, Loader2, Inbox, CalendarClock, Power, PowerOff } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { useBackupSchedules } from '@core/hooks/api/useBackup'
import type { BackupSchedule } from '@core/mock/data/backup'

const BACKUP_TYPE_LABEL: Record<BackupSchedule['backupType'], string> = {
  full: '全量',
  incremental: '增量',
  'config-only': '配置',
}

const PAGE_SIZE = 20

function getErrMsg(e: unknown): string {
  if (e instanceof Error) return e.message
  if (typeof e === 'object' && e && 'message' in e) {
    return String((e as { message: unknown }).message)
  }
  return '未知错误'
}

function formatTime(iso?: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString('zh-CN', { hour12: false })
}

// ---------------------------------------------------------------------------
// 定时备份策略 · /backup/schedule
// ---------------------------------------------------------------------------

export default function BackupSchedulePage() {
  const [page, setPage] = useState(1)
  const query = useBackupSchedules({ page, pageSize: PAGE_SIZE })

  const items = query.data?.items ?? []
  const total = query.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const stats = useMemo(() => {
    const enabled = items.filter((s) => s.enabled).length
    return { enabled, disabled: items.length - enabled }
  }, [items])

  return (
    <PageShell
      code="F06"
      title="BACKUP SCHEDULE · 定时备份"
      subtitle="CRON-DRIVEN BACKUP POLICIES"
      isFetching={query.isFetching || undefined}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => query.refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      {/* 概览 */}
      <div className="mb-3 grid grid-cols-3 gap-3">
        <MiniStat label="策略总数 · TOTAL" value={total} color="#00f0ff" icon={<CalendarClock className="size-4" />} />
        <MiniStat label="启用中 · ENABLED" value={stats.enabled} color="#00ff88" icon={<Power className="size-4" />} />
        <MiniStat label="已停用 · DISABLED" value={stats.disabled} color="#525a78" icon={<PowerOff className="size-4" />} />
      </div>

      {query.isLoading ? (
        <CenterSync />
      ) : query.isError ? (
        <FailureBox error={query.error} />
      ) : items.length === 0 ? (
        <EmptyBox hint="NO SCHEDULES · 无定时策略" />
      ) : (
        <>
          <div className="space-y-1.5">
            {items.map((s: BackupSchedule) => {
              const color = s.enabled ? '#00ff88' : '#525a78'
              return (
                <div
                  key={s.id}
                  className="fleet-row grid grid-cols-[10px_1.6fr_1fr_1.2fr_1fr] items-center gap-3 rounded-sm px-3 py-2.5"
                  style={{ ['--row-color' as never]: color }}
                >
                  <span
                    className="size-2 rounded-full"
                    style={{ background: color, boxShadow: `0 0 8px ${color}` }}
                  />
                  <div className="min-w-0">
                    <div className="truncate font-display text-sm font-bold text-cyan-100">
                      {s.scheduleName}
                    </div>
                    <div className="font-mono text-[10px] text-cyan-300/55">
                      {BACKUP_TYPE_LABEL[s.backupType]} · {s.deviceGroups.length} 组 ·{' '}
                      {s.creator}
                    </div>
                  </div>
                  <div className="font-mono text-[11px] text-cyan-200/85">
                    <div>{s.cronExpression}</div>
                    <div className="text-[10px] text-cyan-300/55">{s.cronDescription}</div>
                  </div>
                  <div className="font-mono text-[10px] text-cyan-300/65">
                    <div>下次 {formatTime(s.nextRunTime)}</div>
                    <div className="text-cyan-300/45">上次 {formatTime(s.lastRunTime)}</div>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="font-mono text-[10px] text-cyan-300/55">
                      保留 {s.retentionDays}d
                    </span>
                    <span className="chip" style={{ color }}>
                      {s.enabled ? '启用' : '停用'}
                    </span>
                  </div>
                </div>
              )
            })}
          </div>
          <Pager page={page} totalPages={totalPages} total={total} onPage={setPage} />
        </>
      )}
    </PageShell>
  )
}

// ---------------------------------------------------------------------------
// 局部组件
// ---------------------------------------------------------------------------

function MiniStat({
  label,
  value,
  color,
  icon,
}: {
  label: string
  value: number
  color: string
  icon: React.ReactNode
}) {
  return (
    <div className="glass relative overflow-hidden rounded-sm">
      <div className="scanline" />
      <div className="relative flex items-center gap-3 p-4">
        <span style={{ color }}>{icon}</span>
        <div>
          <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
            {label}
          </div>
          <div className="font-display text-2xl font-bold text-glow" style={{ color }}>
            {value.toLocaleString()}
          </div>
        </div>
      </div>
    </div>
  )
}

function Pager({
  page,
  totalPages,
  total,
  onPage,
}: {
  page: number
  totalPages: number
  total: number
  onPage: (updater: (p: number) => number) => void
}) {
  return (
    <div className="mt-4 flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <NeonButton onClick={() => onPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
          ◂ PREV
        </NeonButton>
        <NeonButton
          onClick={() => onPage((p) => Math.min(totalPages, p + 1))}
          disabled={page >= totalPages}
        >
          NEXT ▸
        </NeonButton>
      </div>
    </div>
  )
}

function CenterSync() {
  return (
    <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
      <Loader2 className="size-4 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
    </div>
  )
}

function FailureBox({ error }: { error: unknown }) {
  return (
    <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
      FAILURE · {getErrMsg(error)}
    </div>
  )
}

function EmptyBox({ hint }: { hint: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16">
      <Inbox className="size-10 text-cyan-400/50" />
      <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
        {hint}
      </div>
    </div>
  )
}
