import { Badge } from '@/components/ui/badge'
import type {
  NotificationChannel,
  NotificationHistoryStatus,
} from '@core/types/notification'

// ============================================================
// 通知中心 — 模块内共享渠道/状态映射与小组件
// 对照 v1 webcode/src/pages/notifications（channelColorMap / statusColorMap）。
// v2 用 shadcn Badge 的语义化 variant 表达同样的视觉分级。
// ============================================================

export const CHANNEL_LABEL: Record<NotificationChannel, string> = {
  email: '邮件',
  sms: '短信',
  webhook: 'Webhook',
}

const CHANNEL_VARIANT: Record<
  NotificationChannel,
  'default' | 'success' | 'secondary'
> = {
  email: 'default',
  sms: 'success',
  webhook: 'secondary',
}

export const HISTORY_STATUS_LABEL: Record<NotificationHistoryStatus, string> = {
  pending: '待发送',
  sent: '已发送',
  failed: '发送失败',
  dead_letter: '死信',
}

const HISTORY_STATUS_VARIANT: Record<
  NotificationHistoryStatus,
  'warning' | 'success' | 'destructive' | 'muted'
> = {
  pending: 'warning',
  sent: 'success',
  failed: 'destructive',
  dead_letter: 'muted',
}

export const LANGUAGE_LABEL: Record<string, string> = {
  'zh-CN': '简体中文',
  'en-US': 'English',
}

export function ChannelBadge({ channel }: { channel: NotificationChannel }) {
  return (
    <Badge variant={CHANNEL_VARIANT[channel] ?? 'outline'}>
      {CHANNEL_LABEL[channel] ?? channel}
    </Badge>
  )
}

export function HistoryStatusBadge({
  status,
}: {
  status: NotificationHistoryStatus
}) {
  return (
    <Badge variant={HISTORY_STATUS_VARIANT[status] ?? 'muted'}>
      {HISTORY_STATUS_LABEL[status] ?? status}
    </Badge>
  )
}
