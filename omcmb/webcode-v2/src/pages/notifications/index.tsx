import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowRight, FileText, History, Send } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { PageShell } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useNotificationTemplates,
  useNotificationHistory,
} from '@core/hooks/api/useNotifications'
import type { NotificationHistoryStatus } from '@core/types/notification'

import { HISTORY_STATUS_LABEL } from './shared'

// ============================================================
// 通知中心 — 总览首页（/notifications）
// 对照 v1 webcode/src/pages/notifications/index（Tabs：模板 / 历史）。
// v2 拆为独立子路由，本页作为模块入口：聚合模板与近期发送统计，
// 提供进入「通知模板」「发送历史」两个子模块的导航。
// 数据全走真实 @core hooks（useNotificationTemplates / useNotificationHistory）。
// ============================================================

function StatCard({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: number | string
  tone?: 'default' | 'emerald' | 'amber' | 'red' | 'muted'
}) {
  const toneClass = {
    default: 'text-foreground',
    emerald: 'text-emerald-600 dark:text-emerald-400',
    amber: 'text-amber-600 dark:text-amber-400',
    red: 'text-destructive',
    muted: 'text-muted-foreground',
  }[tone]
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>
        {value}
      </div>
    </div>
  )
}

export default function NotificationsPage() {
  const navigate = useNavigate()

  // 模板：仅取首页用于统计总数 + 启用数
  const templatesQuery = useNotificationTemplates({ page: 1, pageSize: 100 })
  // 历史：取较大一页用于近期状态分布统计
  const historyQuery = useNotificationHistory({ page: 1, pageSize: 100 })

  const isFetching = templatesQuery.isFetching || historyQuery.isFetching

  const templates = templatesQuery.data?.items ?? []
  const templateTotal = templatesQuery.data?.total ?? 0
  const enabledTemplates = useMemo(
    () => templates.filter((t) => t.enabled).length,
    [templates],
  )

  const historyRows = historyQuery.data?.items ?? []
  const historyTotal = historyQuery.data?.total ?? 0

  const statusCounts = useMemo(() => {
    const acc: Record<NotificationHistoryStatus, number> = {
      pending: 0,
      sent: 0,
      failed: 0,
      dead_letter: 0,
    }
    for (const h of historyRows) acc[h.status] = (acc[h.status] ?? 0) + 1
    return acc
  }, [historyRows])

  return (
    <PageShell
      title="通知中心"
      description="通知模板维护与发送历史追踪（邮件 / 短信 / Webhook）"
      isFetching={isFetching}
    >
      <div className="mb-6 grid grid-cols-2 gap-3 md:grid-cols-4">
        <StatCard label="模板总数" value={templateTotal} />
        <StatCard label="已启用模板" value={enabledTemplates} tone="emerald" />
        <StatCard label="历史记录" value={historyTotal} />
        <StatCard
          label="近期失败/死信"
          value={statusCounts.failed + statusCounts.dead_letter}
          tone="red"
        />
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <NavCard
          icon={<FileText className="size-5" />}
          title="通知模板"
          desc="维护邮件 / 短信 / Webhook 通知模板，支持启用与多语言"
          onClick={() => navigate('/notifications/templates')}
          footer={
            <span className="text-xs text-muted-foreground">
              共 {templateTotal} 个模板 · 启用 {enabledTemplates}
            </span>
          }
        />
        <NavCard
          icon={<History className="size-5" />}
          title="发送历史"
          desc="查看通知发送记录、状态分布与失败原因，支持重试计数追踪"
          onClick={() => navigate('/notifications/history')}
          footer={
            <div className="flex flex-wrap items-center gap-1.5">
              <RecentBadge label={HISTORY_STATUS_LABEL.sent} value={statusCounts.sent} tone="success" />
              <RecentBadge label={HISTORY_STATUS_LABEL.pending} value={statusCounts.pending} tone="warning" />
              <RecentBadge label={HISTORY_STATUS_LABEL.failed} value={statusCounts.failed} tone="destructive" />
              <RecentBadge label={HISTORY_STATUS_LABEL.dead_letter} value={statusCounts.dead_letter} tone="muted" />
            </div>
          }
        />
      </div>

      <Card className="mt-4">
        <CardHeader className="p-4 pb-2">
          <CardTitle className="flex items-center gap-2 text-base font-medium">
            <Send className="size-4" /> 快速操作
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2 p-4 pt-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => navigate('/notifications/templates/new')}
          >
            新建模板
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => navigate('/notifications/history')}
          >
            查看发送历史
          </Button>
        </CardContent>
      </Card>
    </PageShell>
  )
}

function NavCard({
  icon,
  title,
  desc,
  footer,
  onClick,
}: {
  icon: React.ReactNode
  title: string
  desc: string
  footer?: React.ReactNode
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="group flex flex-col rounded-lg border bg-card p-5 text-left transition-colors hover:border-primary/40 hover:bg-accent/40"
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-primary">
          {icon}
          <span className="text-base font-semibold text-foreground">{title}</span>
        </div>
        <ArrowRight className="size-4 text-muted-foreground transition-transform group-hover:translate-x-0.5 group-hover:text-foreground" />
      </div>
      <p className="mt-2 text-sm text-muted-foreground">{desc}</p>
      {footer ? <div className="mt-3">{footer}</div> : null}
    </button>
  )
}

function RecentBadge({
  label,
  value,
  tone,
}: {
  label: string
  value: number
  tone: 'success' | 'warning' | 'destructive' | 'muted'
}) {
  return (
    <Badge variant={tone}>
      {label} {value}
    </Badge>
  )
}
