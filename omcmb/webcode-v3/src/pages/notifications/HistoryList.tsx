import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  AlertTriangle,
  ChevronRight,
  Inbox,
  Loader2,
  Mail,
  MessageSquare,
  RefreshCcw,
  Webhook,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useNotificationHistory } from '@core/hooks/api/useNotifications'
import type {
  NotificationChannel,
  NotificationHistoryStatus,
} from '@core/types/notification'

const PAGE_SIZE = 20

const CHANNEL_META: Record<
  NotificationChannel,
  { label: string; color: string; icon: typeof Mail }
> = {
  email: { label: '邮件', color: '#5b9eff', icon: Mail },
  sms: { label: '短信', color: '#00ff88', icon: MessageSquare },
  webhook: { label: 'Webhook', color: '#b96bff', icon: Webhook },
}

const STATUS_BADGE: Record<NotificationHistoryStatus, { status: string; label: string }> = {
  pending: { status: 'warning', label: '待发送' },
  sent: { status: 'ok', label: '已发送' },
  failed: { status: 'critical', label: '失败' },
  dead_letter: { status: 'major', label: '死信' },
}

const CHANNEL_FILTER: { v: '' | NotificationChannel; t: string }[] = [
  { v: '', t: '全部渠道' },
  { v: 'email', t: '邮件' },
  { v: 'sms', t: '短信' },
  { v: 'webhook', t: 'Webhook' },
]

const STATUS_FILTER: { v: '' | NotificationHistoryStatus; t: string }[] = [
  { v: '', t: '全部状态' },
  { v: 'sent', t: '已发送' },
  { v: 'pending', t: '待发送' },
  { v: 'failed', t: '失败' },
  { v: 'dead_letter', t: '死信' },
]

export default function HistoryList() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [channel, setChannel] = useState<'' | NotificationChannel>('')
  const [status, setStatus] = useState<'' | NotificationHistoryStatus>('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(channel ? { channel } : {}),
      ...(status ? { status } : {}),
    }),
    [page, channel, status]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useNotificationHistory(params)

  const items = useMemo(() => data?.items ?? [], [data])
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const counts = useMemo(() => {
    const acc = { sent: 0, failed: 0, pending: 0 }
    for (const h of items) {
      if (h.status === 'sent') acc.sent += 1
      else if (h.status === 'failed' || h.status === 'dead_letter') acc.failed += 1
      else if (h.status === 'pending') acc.pending += 1
    }
    return acc
  }, [items])

  return (
    <PageShell
      code="F-N"
      title="DISPATCH HISTORY · 发送历史"
      subtitle="NOTIFICATION DELIVERY LOG"
      isFetching={isFetching}
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      <div className="mb-3 grid grid-cols-4 gap-3">
        <Stat label="TOTAL · 总记录" color="#00f0ff" value={total} />
        <Stat label="SENT · 本页已发" color="#00ff88" value={counts.sent} />
        <Stat label="PENDING · 本页待发" color="#ffaa00" value={counts.pending} />
        <Stat label="FAILED · 本页失败" color="#ff2d6f" value={counts.failed} />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        {CHANNEL_FILTER.map((o) => (
          <button
            key={o.v || 'all-ch'}
            type="button"
            onClick={() => {
              setChannel(o.v)
              setPage(1)
            }}
            className={`chip transition-all ${
              channel === o.v
                ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                : 'text-cyan-300/45 hover:text-cyan-300/80'
            }`}
          >
            {o.t}
          </button>
        ))}
        <span className="mx-1 h-4 w-px bg-cyan-500/20" />
        {STATUS_FILTER.map((o) => (
          <button
            key={o.v || 'all-st'}
            type="button"
            onClick={() => {
              setStatus(o.v)
              setPage(1)
            }}
            className={`chip transition-all ${
              status === o.v
                ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                : 'text-cyan-300/45 hover:text-cyan-300/80'
            }`}
          >
            {o.t}
          </button>
        ))}
      </div>

      <GlassPanel title="DELIVERY LOG · 发送记录" meta={`PAGE ${page}/${totalPages}`}>
        {items.length > 0 && (
          <div className="grid grid-cols-[1.2fr_0.8fr_0.8fr_2fr_1.6fr_0.6fr_120px] items-center gap-3 border-b border-cyan-500/10 px-3.5 py-2 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
            <span>TIME · 时间</span>
            <span>CHANNEL · 渠道</span>
            <span>STATE · 状态</span>
            <span>SUBJECT · 主题</span>
            <span>RECIPIENTS · 收件人</span>
            <span>RETRY · 重试</span>
            <span className="text-right">OPS · 操作</span>
          </div>
        )}

        {isLoading ? (
          <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
            <Loader2 className="size-4 animate-spin" />
            <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
          </div>
        ) : isError ? (
          <div className="flex flex-col items-center gap-3 px-4 py-10">
            <AlertTriangle className="size-7 text-rose-400/70" />
            <div className="font-mono text-sm text-rose-300">
              FAILURE · {error instanceof Error ? error.message : '未知错误'}
            </div>
            <NeonButton tone="danger" icon={<RefreshCcw />} onClick={() => refetch()}>
              RETRY
            </NeonButton>
          </div>
        ) : items.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-3 py-14">
            <Inbox className="size-9 text-cyan-300/40" />
            <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/45">
              NO HISTORY · 暂无发送记录
            </div>
          </div>
        ) : (
          items.map((h) => {
            const meta = CHANNEL_META[h.channel]
            const Icon = meta.icon
            const badge = STATUS_BADGE[h.status]
            return (
              <div
                key={h.id}
                className="grid grid-cols-[1.2fr_0.8fr_0.8fr_2fr_1.6fr_0.6fr_120px] items-center gap-3 border-b border-cyan-500/8 px-3.5 py-2.5 last:border-b-0 hover:bg-cyan-500/5"
              >
                <span className="font-mono text-[11px] text-cyan-300/70">
                  {formatTime(h.createdAt)}
                </span>
                <span className="chip" style={{ color: meta.color }}>
                  <Icon className="size-3" />
                  {meta.label}
                </span>
                <span>
                  <StatusBadge status={badge.status} label={badge.label} />
                </span>
                <button
                  type="button"
                  onClick={() => navigate(`/notifications/history/${h.id}`)}
                  className="min-w-0 truncate text-left font-display text-xs font-bold text-cyan-100 hover:text-cyan-50"
                >
                  {h.subject || '（无主题）'}
                </button>
                <span className="truncate font-mono text-[11px] text-cyan-200/80">
                  {h.recipients.length === 0 ? '—' : h.recipients.join(', ')}
                </span>
                <span className="font-mono text-[11px] text-cyan-300/70">{h.retryCount}</span>
                <div className="flex items-center justify-end">
                  <button
                    type="button"
                    title="详情"
                    onClick={() => navigate(`/notifications/history/${h.id}`)}
                    className="flex size-6 items-center justify-center rounded-sm border border-cyan-500/25 text-cyan-300/75 transition-colors hover:border-cyan-400/60 hover:text-cyan-100"
                  >
                    <ChevronRight className="size-3.5" />
                  </button>
                </div>
              </div>
            )
          })
        )}
      </GlassPanel>

      <div className="mt-4 flex items-center justify-between">
        <span className="font-mono text-[11px] text-cyan-300/55">
          PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
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

function Stat({ label, color, value }: { label: string; color: string; value: number }) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: color }}
    >
      <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/65">
        {label}
      </div>
      <div
        className="font-display text-2xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}
