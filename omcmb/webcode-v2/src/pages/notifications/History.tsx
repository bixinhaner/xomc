import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowLeft, Eye, RefreshCcw } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  EmptyRow,
  ErrorRow,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'

import { useNotificationHistory } from '@core/hooks/api/useNotifications'
import type {
  NotificationChannel,
  NotificationHistoryStatus,
} from '@core/types/notification'

import { ChannelBadge, HistoryStatusBadge } from './shared'

// ============================================================
// 通知中心 → 发送历史列表（/notifications/history）
// 对照 v1 webcode/src/pages/notifications/HistoryList。
//   - 渠道 / 状态 筛选
//   - 行内「详情」→ /notifications/history/:id（带参详情）
// useNotificationHistory 自带 30s 轮询；三态完整。
// ============================================================

const PAGE_SIZE = 10

type ChannelFilter = 'all' | NotificationChannel
type StatusFilter = 'all' | NotificationHistoryStatus

export default function NotificationHistory() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [channel, setChannel] = useState<ChannelFilter>('all')
  const [status, setStatus] = useState<StatusFilter>('all')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(channel !== 'all' ? { channel } : {}),
      ...(status !== 'all' ? { status } : {}),
    }),
    [page, channel, status],
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useNotificationHistory(params)

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const hasFilter = channel !== 'all' || status !== 'all'

  const cols = [
    '创建时间',
    '渠道',
    '状态',
    '主题',
    '接收方',
    '重试次数',
    '发送时间',
    '操作',
  ]

  return (
    <PageShell
      title="发送历史"
      description={`共 ${total} 条发送记录`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate('/notifications')}
          >
            <ArrowLeft className="size-4" /> 返回
          </Button>

          <Select
            value={channel}
            onValueChange={(v) => {
              setChannel(v as ChannelFilter)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="渠道" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部渠道</SelectItem>
              <SelectItem value="email">邮件</SelectItem>
              <SelectItem value="sms">短信</SelectItem>
              <SelectItem value="webhook">Webhook</SelectItem>
            </SelectContent>
          </Select>

          <Select
            value={status}
            onValueChange={(v) => {
              setStatus(v as StatusFilter)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              <SelectItem value="pending">待发送</SelectItem>
              <SelectItem value="sent">已发送</SelectItem>
              <SelectItem value="failed">发送失败</SelectItem>
              <SelectItem value="dead_letter">死信</SelectItem>
            </SelectContent>
          </Select>

          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={() => refetch()}
          >
            <RefreshCcw className={isFetching ? 'animate-spin' : ''} /> 刷新
          </Button>
        </div>
      }
    >
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>
                {hasFilter ? '没有匹配的发送记录' : '暂无发送记录'}
              </EmptyRow>
            ) : (
              rows.map((h) => (
                <TableRow key={h.id}>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(h.createdAt)}
                  </TableCell>
                  <TableCell>
                    <ChannelBadge channel={h.channel} />
                  </TableCell>
                  <TableCell>
                    <HistoryStatusBadge status={h.status} />
                  </TableCell>
                  <TableCell
                    className="max-w-xs truncate text-xs"
                    title={h.subject}
                  >
                    <button
                      type="button"
                      className="text-primary hover:underline"
                      onClick={() => navigate(`/notifications/history/${h.id}`)}
                    >
                      {h.subject || '(无主题)'}
                    </button>
                  </TableCell>
                  <TableCell
                    className="max-w-[220px] truncate text-xs text-muted-foreground"
                    title={h.recipients.join(', ')}
                  >
                    {h.recipients.length > 0 ? h.recipients.join(', ') : '—'}
                  </TableCell>
                  <TableCell className="tabular-nums text-xs">
                    {h.retryCount}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {h.sentAt ? formatTime(h.sentAt) : '—'}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => navigate(`/notifications/history/${h.id}`)}
                    >
                      <Eye className="size-4" /> 详情
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={PAGE_SIZE}
        onChange={setPage}
      />
    </PageShell>
  )
}
