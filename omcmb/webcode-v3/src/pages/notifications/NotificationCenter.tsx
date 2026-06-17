import { useMemo } from 'react'
import {
  AlertTriangle,
  FileText,
  Loader2,
  Mail,
  MessageSquare,
  RefreshCcw,
  Send,
  Webhook,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { formatTime } from '@/lib/format'
import {
  useNotificationHistory,
  useNotificationTemplates,
} from '@core/hooks/api/useNotifications'
import type {
  NotificationChannel,
  NotificationHistoryStatus,
} from '@core/types/notification'

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

export default function NotificationCenter() {

  const templatesQ = useNotificationTemplates({ page: 1, pageSize: 200 })
  const historyQ = useNotificationHistory({ page: 1, pageSize: 50 })

  const templates = useMemo(() => templatesQ.data?.items ?? [], [templatesQ.data])
  const history = useMemo(() => historyQ.data?.items ?? [], [historyQ.data])

  const templateTotal = templatesQ.data?.total ?? 0
  const historyTotal = historyQ.data?.total ?? 0

  const enabledTemplates = useMemo(
    () => templates.filter((t) => t.enabled).length,
    [templates]
  )

  const statusCounts = useMemo(() => {
    const acc: Record<NotificationHistoryStatus, number> = {
      pending: 0,
      sent: 0,
      failed: 0,
      dead_letter: 0,
    }
    for (const h of history) acc[h.status] += 1
    return acc
  }, [history])

  const channelCounts = useMemo(() => {
    const acc: Record<NotificationChannel, number> = { email: 0, sms: 0, webhook: 0 }
    for (const h of history) acc[h.channel] += 1
    return acc
  }, [history])

  // 成功率：基于本页采样的最近发送记录
  const successRate = useMemo(() => {
    const settled = statusCounts.sent + statusCounts.failed + statusCounts.dead_letter
    if (settled === 0) return 0
    return (statusCounts.sent / settled) * 100
  }, [statusCounts])

  const recent = useMemo(() => history.slice(0, 8), [history])

  const isLoading = templatesQ.isLoading || historyQ.isLoading
  const isError = templatesQ.isError || historyQ.isError
  const isFetching = templatesQ.isFetching || historyQ.isFetching
  const errMsg =
    (templatesQ.error instanceof Error && templatesQ.error.message) ||
    (historyQ.error instanceof Error && historyQ.error.message) ||
    '未知错误'

  const refetchAll = () => {
    void templatesQ.refetch()
    void historyQ.refetch()
  }

  return (
    <PageShell
      code="F-N"
      title="NOTIFICATION CENTER · 通知中心"
      subtitle="TEMPLATES · DISPATCH HISTORY · CHANNELS"
      isFetching={isFetching}
      toolbar={
        <>
          <NeonButton icon={<RefreshCcw />} onClick={refetchAll}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {isError ? (
        <div className="flex flex-col items-center gap-3 py-16">
          <AlertTriangle className="size-9 text-rose-400/70" />
          <div className="font-mono text-sm text-rose-300">FAILURE · {errMsg}</div>
          <NeonButton tone="danger" icon={<RefreshCcw />} onClick={refetchAll}>
            RETRY
          </NeonButton>
        </div>
      ) : isLoading ? (
        <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
        </div>
      ) : (
        <div className="grid grid-cols-12 gap-3">
          {/* KPI 概览 */}
          <div className="col-span-12 grid grid-cols-2 gap-3 lg:grid-cols-4">
            <Stat label="TEMPLATES · 模板总数" color="#00f0ff" value={templateTotal} />
            <Stat label="ENABLED · 启用模板" color="#00ff88" value={enabledTemplates} />
            <Stat label="HISTORY · 发送记录" color="#5b9eff" value={historyTotal} />
            <Stat
              label="FAILED · 近期失败"
              color="#ff2d6f"
              value={statusCounts.failed + statusCounts.dead_letter}
            />
          </div>

          {/* 成功率 + 状态分布 */}
          <GlassPanel
            title="DELIVERY · 投递成功率"
            meta="最近采样"
            className="col-span-12 lg:col-span-4"
          >
            <div className="flex flex-col items-center gap-3 py-5">
              <RadialGauge
                value={successRate}
                label="SUCCESS"
                size={150}
                color={successRate >= 90 ? '#00ff88' : successRate >= 60 ? '#ffaa00' : '#ff2d6f'}
              />
              <div className="grid w-full grid-cols-2 gap-2 px-3">
                <StatusLine
                  label="已发送"
                  color="#00ff88"
                  value={statusCounts.sent}
                />
                <StatusLine label="待发送" color="#ffaa00" value={statusCounts.pending} />
                <StatusLine label="失败" color="#ff2d6f" value={statusCounts.failed} />
                <StatusLine
                  label="死信"
                  color="#ff7a1a"
                  value={statusCounts.dead_letter}
                />
              </div>
            </div>
          </GlassPanel>

          {/* 渠道分布 */}
          <GlassPanel
            title="CHANNELS · 渠道分布"
            meta="按近期记录"
            className="col-span-12 lg:col-span-4"
          >
            <div className="space-y-0">
              {(Object.keys(CHANNEL_META) as NotificationChannel[]).map((ch) => {
                const meta = CHANNEL_META[ch]
                const Icon = meta.icon
                const cnt = channelCounts[ch]
                const pct = history.length > 0 ? (cnt / history.length) * 100 : 0
                return (
                  <div
                    key={ch}
                    className="border-b border-cyan-500/10 px-3.5 py-3 last:border-b-0"
                  >
                    <div className="mb-1.5 flex items-center justify-between">
                      <span
                        className="flex items-center gap-2 font-display text-sm font-bold"
                        style={{ color: meta.color }}
                      >
                        <Icon className="size-4" />
                        {meta.label}
                      </span>
                      <span className="font-mono text-xs text-cyan-300/70">
                        {cnt} · {pct.toFixed(0)}%
                      </span>
                    </div>
                    <div className="h-1.5 w-full overflow-hidden rounded-full bg-cyan-500/10">
                      <div
                        className="h-full rounded-full transition-all"
                        style={{
                          width: `${pct}%`,
                          background: meta.color,
                          boxShadow: `0 0 8px ${meta.color}`,
                        }}
                      />
                    </div>
                  </div>
                )
              })}
            </div>
          </GlassPanel>

          {/* 最近发送 */}
          <GlassPanel
            title="RECENT DISPATCH · 最近发送"
            meta={`TOP ${recent.length}`}
            className="col-span-12 lg:col-span-4"
          >
            {recent.length === 0 ? (
              <div className="flex flex-col items-center justify-center gap-3 py-12">
                <Send className="size-8 text-cyan-300/35" />
                <div className="font-mono text-[11px] uppercase tracking-[0.18em] text-cyan-300/45">
                  NO DISPATCH · 暂无发送记录
                </div>
              </div>
            ) : (
              <div>
                {recent.map((h) => {
                  const badge = STATUS_BADGE[h.status]
                  const chMeta = CHANNEL_META[h.channel]
                  return (
                    <div
                      key={h.id}
                      className="flex w-full items-center gap-2 border-b border-cyan-500/8 px-3.5 py-2.5 text-left last:border-b-0"
                    >
                      <span
                        className="size-1.5 shrink-0 rounded-full"
                        style={{ background: chMeta.color, boxShadow: `0 0 6px ${chMeta.color}` }}
                      />
                      <div className="min-w-0 flex-1">
                        <div className="truncate font-display text-xs font-bold text-cyan-100">
                          {h.subject || '（无主题）'}
                        </div>
                        <div className="truncate font-mono text-[10px] text-cyan-300/50">
                          {chMeta.label} · {formatTime(h.createdAt)}
                        </div>
                      </div>
                      <StatusBadge status={badge.status} label={badge.label} />
                    </div>
                  )
                })}
              </div>
            )}
          </GlassPanel>

          {/* 模板速览 */}
          <GlassPanel
            title="TEMPLATE OVERVIEW · 模板速览"
            meta={`${templates.length} 项`}
            className="col-span-12"
          >
            {templates.length === 0 ? (
              <div className="flex flex-col items-center justify-center gap-3 py-12">
                <FileText className="size-8 text-cyan-300/35" />
                <div className="font-mono text-[11px] uppercase tracking-[0.18em] text-cyan-300/45">
                  NO TEMPLATES · 暂无模板
                </div>
              </div>
            ) : (
              <div className="grid grid-cols-1 gap-2 p-3 md:grid-cols-2 xl:grid-cols-3">
                {templates.slice(0, 9).map((tpl) => {
                  const chMeta = CHANNEL_META[tpl.channel]
                  return (
                    <div
                      key={tpl.id}
                      className="glass flex items-center gap-3 rounded-sm border-l-2 px-3 py-2.5 text-left"
                      style={{ borderLeftColor: chMeta.color }}
                    >
                      <div className="min-w-0 flex-1">
                        <div className="truncate font-display text-sm font-bold text-cyan-100">
                          {tpl.name}
                        </div>
                        <div className="truncate font-mono text-[10px] text-cyan-300/50">
                          {chMeta.label} · {tpl.language}
                        </div>
                      </div>
                      <StatusBadge
                        status={tpl.enabled ? 'active' : 'inactive'}
                        label={tpl.enabled ? '启用' : '禁用'}
                      />
                    </div>
                  )
                })}
              </div>
            )}
          </GlassPanel>
        </div>
      )}
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

function StatusLine({ label, color, value }: { label: string; color: string; value: number }) {
  return (
    <div className="flex items-center justify-between rounded-sm border border-cyan-500/10 px-2.5 py-1.5">
      <span className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.12em] text-cyan-300/65">
        <span
          className="size-1.5 rounded-full"
          style={{ background: color, boxShadow: `0 0 6px ${color}` }}
        />
        {label}
      </span>
      <span className="font-display text-sm font-bold" style={{ color }}>
        {value}
      </span>
    </div>
  )
}
