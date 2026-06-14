import { useNavigate, useParams } from 'react-router-dom'
import { Activity, ArrowLeft, Cpu } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useEventLog } from '@core/hooks/api/useEventLog'
import type { EventLevel } from '@core/services/api/eventLogApi'

import { DRow, FeedError, FeedLoading } from './_shared'

// ──────────────────────────────────────────────────────────────────────────
// /logs/event/:id —— 设备事件详情（EventLog 列表下钻）
// 真实端点：useEventLog(id) → eventLogApi GET /event-logs/:id（真实 getById）
// ──────────────────────────────────────────────────────────────────────────

const LEVEL_META: Record<EventLevel, { label: string; color: string; badge: string }> = {
  info: { label: '信息', color: '#00f0ff', badge: 'unknown' },
  success: { label: '成功', color: '#00ff88', badge: 'ok' },
  warning: { label: '告警', color: '#ffaa00', badge: 'warning' },
  error: { label: '错误', color: '#ff2d6f', badge: 'critical' },
}

function prettyData(data: unknown): string | null {
  if (data === null || data === undefined) return null
  if (typeof data === 'string') return data
  try {
    return JSON.stringify(data, null, 2)
  } catch {
    return String(data)
  }
}

export default function EventLogDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data: record, isLoading, isError, error, isFetching } = useEventLog(id)

  const meta = record ? LEVEL_META[record.eventLevel] : undefined
  const dataText = record ? prettyData(record.eventData) : null

  return (
    <PageShell
      code="F06"
      title="EVENT LOG · 事件详情"
      subtitle={record ? record.deviceSn : (id ?? '—')}
      isFetching={isFetching}
      toolbar={
        <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/logs/event')}>
          BACK
        </NeonButton>
      }
    >
      {isLoading ? (
        <FeedLoading />
      ) : isError ? (
        <FeedError error={error} />
      ) : !record ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <Activity className="size-12 text-emerald-400/50" />
          <div className="font-mono text-xs uppercase tracking-[0.2em] text-emerald-300/70">
            未找到事件 · {id}
          </div>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/logs/event')}>
            返回列表
          </NeonButton>
        </div>
      ) : (
        <div className="flex flex-col gap-4">
          <div className="flex flex-wrap items-center gap-3">
            <span className="chip" style={{ color: meta?.color ?? '#00f0ff' }}>
              {meta?.label ?? record.eventLevel}
            </span>
            <StatusBadge status={meta?.badge ?? 'unknown'} label={record.eventType || 'event'} />
            <span className="font-mono text-[11px] text-cyan-300/55">
              {formatTime(record.occurredAt)}
            </span>
          </div>

          <GlassPanel strong title="事件信息" meta={<Cpu className="size-3.5" />}>
            <dl className="grid grid-cols-[120px_1fr] gap-x-4 gap-y-2.5 p-4 text-[12px]">
              <DRow k="设备名" v={record.deviceName} />
              <DRow k="设备 SN" v={record.deviceSn} mono />
              <DRow k="设备 ID" v={record.deviceId} mono />
              <DRow k="制式" v={record.deviceType || (record.isGnb ? 'gNB' : 'eNB')} />
              <DRow k="软件版本" v={record.softwareVersion} mono />
              <DRow k="管理 IP" v={record.operateIp} mono />
              <DRow k="事件类型" v={record.eventType} />
              <DRow k="事件级别" v={meta?.label ?? record.eventLevel} color={meta?.color} />
              <DRow k="事件原因" v={record.eventReason} />
              <DRow k="发生时间" v={formatTime(record.occurredAt)} mono />
              <DRow k="入库时间" v={formatTime(record.createdAt)} mono />
            </dl>
          </GlassPanel>

          {dataText && (
            <GlassPanel strong title="事件数据" meta="event_data">
              <div className="terminal m-4 max-h-[45vh] overflow-auto whitespace-pre-wrap break-all p-3 text-[11px] text-emerald-100/85">
                {dataText}
              </div>
            </GlassPanel>
          )}
        </div>
      )}
    </PageShell>
  )
}
