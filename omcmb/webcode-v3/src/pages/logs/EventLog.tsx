import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Activity, RefreshCcw, Search } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatTime } from '@/lib/format'
import { useEventLogList } from '@core/hooks/api/useEventLog'
import type { EventLevel, EventLog } from '@core/services/api/eventLogApi'

import {
  EmptyFeed,
  FeedError,
  FeedLoading,
  FilterInput,
  LOG_PAGE_SIZE,
  Pager,
  StatCard,
} from './_shared'

// ──────────────────────────────────────────────────────────────────────────
// /logs/event —— 设备事件日志（对照 v1 webcode log/event → device 事件流）
// 真实端点：useEventLogList → eventLogApi GET /event-logs（按设备/类型/时间）
// HUD 形态：级别统计 + 行列表 + 下钻到 /logs/event/:id（真实 getById）
// ──────────────────────────────────────────────────────────────────────────

export const EVENT_LEVEL_META: Record<EventLevel, { label: string; color: string }> = {
  info: { label: '信息', color: '#00f0ff' },
  success: { label: '成功', color: '#00ff88' },
  warning: { label: '告警', color: '#ffaa00' },
  error: { label: '错误', color: '#ff2d6f' },
}

const COLS = 'grid-cols-[150px_90px_120px_1.4fr_150px]'

export default function EventLogPage() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [deviceSn, setDeviceSn] = useState('')
  const [eventType, setEventType] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: LOG_PAGE_SIZE,
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
      ...(eventType ? { eventType } : {}),
    }),
    [page, deviceSn, eventType],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useEventLogList(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / LOG_PAGE_SIZE))

  const levelCount = useMemo(() => {
    const c: Record<string, number> = { info: 0, success: 0, warning: 0, error: 0 }
    rows.forEach((r) => {
      c[r.eventLevel] = (c[r.eventLevel] ?? 0) + 1
    })
    return c
  }, [rows])

  return (
    <PageShell
      code="F06"
      title="EVENT LOG · 设备事件"
      subtitle="DEVICE EVENT STREAM"
      isFetching={isFetching}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          {isFetching ? 'SYNC…' : 'REFRESH'}
        </NeonButton>
      }
    >
      <div className="flex h-full flex-col gap-3">
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <StatCard label="TOTAL" color="#00f0ff" value={total} hint="后端总量" />
          <StatCard label="错误" color={EVENT_LEVEL_META.error.color} value={levelCount.error} hint="本页" />
          <StatCard label="告警" color={EVENT_LEVEL_META.warning.color} value={levelCount.warning} hint="本页" />
          <StatCard label="信息" color={EVENT_LEVEL_META.info.color} value={levelCount.info} hint="本页" />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <FilterInput
            value={deviceSn}
            onChange={(v) => {
              setDeviceSn(v)
              setPage(1)
            }}
            placeholder="按设备 SN 过滤"
            icon={<Search />}
          />
          <select
            className="neon-input w-36"
            value={eventType}
            onChange={(e) => {
              setEventType(e.target.value)
              setPage(1)
            }}
          >
            <option value="">全部事件</option>
            <option value="boot">boot · 启动</option>
          </select>
        </div>

        <GlassPanel strong className="min-h-0 flex-1 overflow-hidden">
          <div className="h-full overflow-auto">
            <div
              className={`sticky top-0 z-10 grid ${COLS} items-center gap-3 border-b border-cyan-500/15 bg-black/40 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55 backdrop-blur`}
            >
              <span>DEVICE SN</span>
              <span>LEVEL</span>
              <span>TYPE</span>
              <span>原因 / 详情</span>
              <span>OCCURRED</span>
            </div>

            {isLoading ? (
              <FeedLoading />
            ) : isError ? (
              <FeedError error={error} />
            ) : rows.length === 0 ? (
              <EmptyFeed icon={<Activity className="size-10 text-emerald-400/60" />} text="无设备事件" />
            ) : (
              rows.map((r) => {
                const meta = EVENT_LEVEL_META[r.eventLevel] ?? {
                  label: r.eventLevel,
                  color: '#6b86b6',
                }
                return (
                  <button
                    type="button"
                    key={r.id}
                    onClick={() => navigate(`/logs/event/${r.id}`)}
                    className={`fleet-row grid w-full ${COLS} items-center gap-3 px-3 py-2.5 text-left`}
                    style={{ ['--row-color' as never]: meta.color }}
                  >
                    <span className="truncate font-display text-sm font-bold text-cyan-100">
                      {r.deviceSn || '—'}
                    </span>
                    <span className="chip" style={{ color: meta.color }}>
                      {meta.label}
                    </span>
                    <span className="truncate font-mono text-[11px] text-cyan-300/70">
                      {r.eventType || '—'}
                    </span>
                    <span className="min-w-0">
                      <span className="block truncate text-xs text-cyan-100/85">
                        {r.eventReason || '—'}
                      </span>
                      <span className="block truncate font-mono text-[10px] text-cyan-300/45">
                        {r.deviceName || r.deviceType || '—'}
                      </span>
                    </span>
                    <span className="font-mono text-[11px] text-cyan-300/70">
                      {formatTime(r.occurredAt)}
                    </span>
                  </button>
                )
              })
            )}
          </div>
        </GlassPanel>

        <Pager page={page} totalPages={totalPages} total={total} onChange={setPage} />
      </div>
    </PageShell>
  )
}

export type { EventLog }
