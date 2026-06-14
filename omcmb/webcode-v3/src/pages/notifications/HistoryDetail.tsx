import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  AlertTriangle,
  ArrowLeft,
  FileWarning,
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
import { useNotificationHistoryDetail } from '@core/hooks/api/useNotifications'
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

export default function HistoryDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data, isLoading, isError, error, isFetching, refetch } =
    useNotificationHistoryDetail(id ?? '')

  // useNotificationHistoryDetail 在 404 时 resolve 为 null（非 reject）
  const record = data ?? null
  const meta = record ? CHANNEL_META[record.channel] : null
  const badge = record ? STATUS_BADGE[record.status] : null

  const failed = useMemo(
    () => record?.status === 'failed' || record?.status === 'dead_letter',
    [record]
  )

  return (
    <PageShell
      code="F-N"
      title="DISPATCH DETAIL · 发送详情"
      subtitle={id ? `MSG-ID · ${id}` : 'NO MESSAGE SELECTED'}
      isFetching={isFetching}
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/notifications/history')}>
            BACK
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {isLoading ? (
        <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
        </div>
      ) : isError ? (
        <div className="flex flex-col items-center gap-3 py-14">
          <AlertTriangle className="size-9 text-rose-400/70" />
          <div className="font-mono text-sm text-rose-300">
            FAILURE · {error instanceof Error ? error.message : '未知错误'}
          </div>
          <NeonButton tone="danger" icon={<RefreshCcw />} onClick={() => refetch()}>
            RETRY
          </NeonButton>
        </div>
      ) : !record ? (
        <div className="flex flex-col items-center justify-center gap-3 py-16">
          <FileWarning className="size-10 text-cyan-300/40" />
          <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/45">
            RECORD NOT FOUND · 未找到发送记录 {id}
          </div>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/notifications/history')}>
            返回发送历史
          </NeonButton>
        </div>
      ) : (
        <div className="grid grid-cols-12 gap-3">
          {/* 概要 */}
          <GlassPanel
            title="OVERVIEW · 概要"
            meta={badge?.label}
            className="col-span-12 lg:col-span-5"
          >
            <div className="space-y-0">
              <Field label="渠道">
                <span className="chip" style={{ color: meta?.color }}>
                  {meta?.label}
                </span>
              </Field>
              <Field label="状态">
                {badge && <StatusBadge status={badge.status} label={badge.label} />}
              </Field>
              <Field label="重试次数">{record.retryCount}</Field>
              <Field label="收件人">
                {record.recipients.length === 0 ? (
                  '—'
                ) : (
                  <span className="flex flex-wrap gap-1.5">
                    {record.recipients.map((r, i) => (
                      <span key={`${r}-${i}`} className="chip text-cyan-300/80">
                        {r}
                      </span>
                    ))}
                  </span>
                )}
              </Field>
              {record.templateId && (
                <Field label="模板">
                  <button
                    type="button"
                    onClick={() =>
                      navigate(`/notifications/templates/${record.templateId}`)
                    }
                    className="break-all font-mono text-[11px] text-cyan-300 underline-offset-2 hover:underline"
                  >
                    {record.templateId}
                  </button>
                </Field>
              )}
              {record.alarmId && (
                <Field label="关联告警">
                  <span className="break-all font-mono text-[11px] text-cyan-200/85">
                    {record.alarmId}
                  </span>
                </Field>
              )}
              <Field label="创建时间">{formatTime(record.createdAt)}</Field>
              <Field label="发送时间">{record.sentAt ? formatTime(record.sentAt) : '—'}</Field>
              <Field label="消息 ID">
                <span className="break-all font-mono text-[11px] text-cyan-300/70">
                  {record.id}
                </span>
              </Field>
            </div>
          </GlassPanel>

          {/* 主题 + 内容 + 错误 */}
          <div className="col-span-12 space-y-3 lg:col-span-7">
            {failed && record.errorMessage && (
              <GlassPanel title="ERROR · 错误信息">
                <div className="px-3.5 py-3 font-mono text-[12px] leading-relaxed text-rose-300/90">
                  {record.errorMessage}
                </div>
              </GlassPanel>
            )}
            <GlassPanel title="SUBJECT · 主题">
              <div className="px-3.5 py-3 font-mono text-sm text-cyan-100/90">
                {record.subject || '—'}
              </div>
            </GlassPanel>
            <GlassPanel title="BODY · 内容">
              <pre className="max-h-[420px] overflow-auto whitespace-pre-wrap break-words px-3.5 py-3 font-mono text-[12px] leading-relaxed text-cyan-200/85">
                {record.body || '—'}
              </pre>
            </GlassPanel>
          </div>
        </div>
      )}
    </PageShell>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[100px_1fr] items-center gap-3 border-b border-cyan-500/10 px-3.5 py-2 last:border-b-0">
      <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">
        {label}
      </div>
      <div className="break-words text-[12px] text-cyan-100/90">{children ?? '—'}</div>
    </div>
  )
}
