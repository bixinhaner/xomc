import { useMemo, useState } from 'react'
import { Search, RefreshCcw, Loader2, BellRing } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { SeverityBar } from '@/components/viz/SeverityBar'
import { formatTime } from '@/lib/format'
import { useAlarmList, useAlarmCount } from '@core/hooks/api/useAlarms'
import type { AlarmSeverity } from '@core/types/common'

const SEV_LABEL: Record<AlarmSeverity, string> = {
  critical: '紧急',
  major: '重要',
  minor: '次要',
  warning: '警告',
}
const SEV_COLOR: Record<AlarmSeverity, string> = {
  critical: '#ff2d6f',
  major: '#ff7a1a',
  minor: '#ffd400',
  warning: '#5b9eff',
}

export function AlarmsPage() {
  const [page, setPage] = useState(1)
  const pageSize = 30
  const [severity, setSeverity] = useState<AlarmSeverity | ''>('')
  const [keyword, setKeyword] = useState('')

  const { data: cnt } = useAlarmCount()
  const params = useMemo(
    () => ({
      page,
      pageSize,
      isActive: true,
      ...(severity ? { severity } : {}),
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, severity, keyword]
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useAlarmList(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <PageShell
      code="F04"
      title="ALARM ARRAY · 警报阵列"
      subtitle="ACTIVE INCIDENT FEED · 30s AUTO-REFRESH"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-72 pl-9"
              placeholder="告警名 / 设备 SN"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          {(['', 'critical', 'major', 'minor', 'warning'] as const).map((s) => (
            <button
              key={s || 'all'}
              type="button"
              onClick={() => {
                setSeverity(s)
                setPage(1)
              }}
              className={`chip transition-all ${
                severity === s
                  ? 'shadow-[0_0_10px_currentColor]'
                  : 'opacity-60 hover:opacity-100'
              }`}
              style={{ color: s ? SEV_COLOR[s] : '#00f0ff' }}
            >
              {s ? SEV_LABEL[s as AlarmSeverity] : 'ALL'}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {/* 严重度统计 */}
      <div className="mb-3 grid grid-cols-4 gap-3">
        <SevStat label="CRITICAL" color="#ff2d6f" value={cnt?.critical ?? 0} />
        <SevStat label="MAJOR" color="#ff7a1a" value={cnt?.major ?? 0} />
        <SevStat label="MINOR" color="#ffd400" value={cnt?.minor ?? 0} />
        <SevStat label="WARNING" color="#5b9eff" value={cnt?.warning ?? 0} />
      </div>

      {/* 时间线 */}
      <div className="space-y-1.5">
        {isLoading ? (
          <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
            <Loader2 className="size-4 animate-spin" />
            <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
          </div>
        ) : isError ? (
          <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
            FAILURE · {error instanceof Error ? error.message : '未知错误'}
          </div>
        ) : rows.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-3 py-16">
            <BellRing className="size-10 text-emerald-400/60" />
            <div className="font-mono text-xs uppercase tracking-[0.2em] text-emerald-300/70">
              ALL CLEAR · 无活动告警
            </div>
          </div>
        ) : (
          rows.map((a) => (
            <div
              key={a.id}
              className="fleet-row grid grid-cols-[28px_2fr_1.4fr_1fr_140px] items-center gap-3 rounded-sm px-3 py-2.5"
              style={{ ['--row-color' as never]: SEV_COLOR[a.severity] }}
            >
              <SeverityBar severity={a.severity} />
              <div>
                <div className="font-display text-sm font-bold text-cyan-100">
                  {a.alarmName || a.alarmCode}
                </div>
                <div className="font-mono text-[10px] text-cyan-300/55">
                  CODE {a.alarmCode} · {a.eventType}
                </div>
              </div>
              <div>
                <div className="text-xs text-cyan-100/85">{a.deviceName || '—'}</div>
                <div className="font-mono text-[10px] text-cyan-300/55">{a.deviceSn}</div>
              </div>
              <div className="font-mono text-[11px] text-cyan-300/75">
                {formatTime(a.eventTime)}
              </div>
              <div className="text-right">
                <span
                  className="chip"
                  style={{
                    color: a.dealState === '0' ? SEV_COLOR[a.severity] : '#00ff88',
                  }}
                >
                  {a.dealState === '0'
                    ? '未确认'
                    : a.dealState === '1'
                      ? '已确认'
                      : a.dealState === '2'
                        ? '已清除'
                        : '已处理'}
                </span>
              </div>
            </div>
          ))
        )}
      </div>

      <div className="mt-4 flex items-center justify-between">
        <span className="font-mono text-[11px] text-cyan-300/55">
          PAGE {page} / {totalPages} · {pageSize}/PAGE · TOTAL {total}
        </span>
        <div className="flex gap-2">
          <NeonButton onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
            ◂ PREV
          </NeonButton>
          <NeonButton
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages}
          >
            NEXT ▸
          </NeonButton>
        </div>
      </div>
    </PageShell>
  )
}

function SevStat({
  label,
  color,
  value,
}: {
  label: string
  color: string
  value: number
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3"
      style={{ borderLeftColor: color }}
    >
      <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
        {label}
      </div>
      <div
        className="font-display text-3xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}
