import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { PageShell, formatTime } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useNotificationHistoryDetail } from '@core/hooks/api/useNotifications'

import { ChannelBadge, HistoryStatusBadge } from './shared'

// ============================================================
// 通知中心 → 发送历史详情（/notifications/history/:id）
// 对照 v1 webcode/src/pages/notifications/HistoryDetail（Drawer 详情）。
// 真实加载：useNotificationHistoryDetail(id) → GET /notifications/history/:id。
// 展示渠道/状态/接收方/主题/正文/重试/错误/关联告警与模板/时间线。
// ============================================================

type FieldVal = string | number | null | undefined

interface Field {
  label: string
  value: FieldVal
  mono?: boolean
}

function InfoGrid({ fields }: { fields: Field[] }) {
  return (
    <div className="grid grid-cols-1 gap-x-8 gap-y-0 sm:grid-cols-2">
      {fields.map((f) => (
        <div
          key={f.label}
          className="flex items-center justify-between gap-4 border-b py-2.5 text-sm"
        >
          <span className="shrink-0 text-muted-foreground">{f.label}</span>
          <span
            className={cn('truncate text-right', f.mono && 'font-mono text-xs')}
            title={f.value != null ? String(f.value) : undefined}
          >
            {f.value === '' || f.value == null ? '—' : f.value}
          </span>
        </div>
      ))}
    </div>
  )
}

export default function NotificationHistoryDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data, isLoading, isError, error, isFetching } =
    useNotificationHistoryDetail(id)

  const metaFields: Field[] = data
    ? [
        { label: '记录 ID', value: data.id, mono: true },
        { label: '重试次数', value: data.retryCount },
        { label: '创建时间', value: formatTime(data.createdAt) },
        { label: '发送时间', value: data.sentAt ? formatTime(data.sentAt) : '—' },
        { label: '关联告警', value: data.alarmId ?? '—', mono: true },
        { label: '关联模板', value: data.templateId ?? '—', mono: true },
      ]
    : []

  return (
    <PageShell
      title="发送记录详情"
      description={id ? `记录 ${id}` : undefined}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate('/notifications/history')}
          >
            <ArrowLeft className="size-4" /> 返回历史列表
          </Button>
          {data && (
            <div className="ml-auto flex items-center gap-2">
              <ChannelBadge channel={data.channel} />
              <HistoryStatusBadge status={data.status} />
            </div>
          )}
        </div>
      }
    >
      {isLoading ? (
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </div>
      ) : isError ? (
        <Card className="flex h-48 items-center justify-center p-6 text-sm text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </Card>
      ) : !data ? (
        <Card className="flex h-48 flex-col items-center justify-center gap-2 p-6 text-muted-foreground">
          <span className="text-sm">未找到发送记录 {id}</span>
          <Button
            variant="outline"
            size="sm"
            onClick={() => navigate('/notifications/history')}
          >
            返回历史列表
          </Button>
        </Card>
      ) : (
        <div className="flex flex-col gap-4">
          <Card>
            <CardHeader className="p-4 pb-2">
              <CardTitle className="text-base font-medium">基本信息</CardTitle>
            </CardHeader>
            <CardContent className="p-4 pt-2">
              <InfoGrid fields={metaFields} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="p-4 pb-2">
              <CardTitle className="text-base font-medium">接收方</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-wrap gap-1.5 p-4 pt-2">
              {data.recipients.length === 0 ? (
                <span className="text-sm text-muted-foreground">—</span>
              ) : (
                data.recipients.map((r, i) => (
                  <Badge key={`${r}-${i}`} variant="outline">
                    {r}
                  </Badge>
                ))
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="p-4 pb-2">
              <CardTitle className="text-base font-medium">内容</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-3 p-4 pt-2">
              <div>
                <div className="mb-1 text-xs text-muted-foreground">主题</div>
                <div className="text-sm">{data.subject || '(无主题)'}</div>
              </div>
              <div>
                <div className="mb-1 text-xs text-muted-foreground">正文</div>
                <pre className="whitespace-pre-wrap break-words rounded-md border bg-muted/30 p-3 text-sm">
                  {data.body || '—'}
                </pre>
              </div>
            </CardContent>
          </Card>

          {data.errorMessage && (
            <Card className="border-destructive/30">
              <CardHeader className="p-4 pb-2">
                <CardTitle className="text-base font-medium text-destructive">
                  错误信息
                </CardTitle>
              </CardHeader>
              <CardContent className="p-4 pt-2">
                <pre className="whitespace-pre-wrap break-words text-sm text-destructive">
                  {data.errorMessage}
                </pre>
              </CardContent>
            </Card>
          )}
        </div>
      )}
    </PageShell>
  )
}
