import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { PageShell, formatTime } from '@/components/layout/PageShell'

import { useEventLog } from '@core/hooks/api/useEventLog'
import type { EventLevel } from '@core/services/api/eventLogApi'

// ===========================================================================
// 设备事件详情（/logs/event/:id）— 真实单条端点 eventLogApi.getById
// ===========================================================================

const LEVEL_LABEL: Record<EventLevel, string> = {
  info: '信息',
  warning: '警告',
  error: '错误',
  success: '成功',
}

const LEVEL_VARIANT: Record<EventLevel, 'default' | 'warning' | 'destructive' | 'success'> = {
  info: 'default',
  warning: 'warning',
  error: 'destructive',
  success: 'success',
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1 border-b py-3 last:border-b-0">
      <span className="text-xs uppercase tracking-wider text-muted-foreground">{label}</span>
      <span className="text-sm">{children}</span>
    </div>
  )
}

function stringifyEventData(data: unknown): string | null {
  if (data === null || data === undefined) return null
  if (typeof data === 'string') return data
  try {
    return JSON.stringify(data, null, 2)
  } catch {
    return String(data)
  }
}

export default function EventLogDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data: item, isLoading, isError, error } = useEventLog(id)
  const eventDataText = item ? stringifyEventData(item.eventData) : null

  return (
    <PageShell
      title="设备事件详情"
      description={item ? item.deviceSn : id ? `事件 ID ${id}` : undefined}
      toolbar={
        <Button variant="outline" size="sm" onClick={() => navigate('/logs/event')}>
          <ArrowLeft className="size-4" /> 返回列表
        </Button>
      }
    >
      {isLoading ? (
        <div className="rounded-lg border bg-card p-8 text-center text-sm text-muted-foreground">
          加载中…
        </div>
      ) : isError ? (
        <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-8 text-center text-sm text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </div>
      ) : !item ? (
        <div className="rounded-lg border bg-card p-8 text-center text-sm text-muted-foreground">
          未找到该事件
        </div>
      ) : (
        <div className="max-w-2xl rounded-lg border bg-card px-5 py-2">
          <Field label="设备名称">{item.deviceName || '—'}</Field>
          <Field label="设备 SN">
            <span className="font-mono">{item.deviceSn || '—'}</span>
          </Field>
          <Field label="制式">{item.deviceType || '—'}</Field>
          <Field label="管理 IP">
            <span className="font-mono text-xs">{item.operateIp || '—'}</span>
          </Field>
          <Field label="软件版本">
            <span className="font-mono text-xs">{item.softwareVersion || '—'}</span>
          </Field>
          <Field label="事件类型">
            <span className="font-mono text-xs">{item.eventType || '—'}</span>
          </Field>
          <Field label="级别">
            <Badge variant={LEVEL_VARIANT[item.eventLevel] ?? 'default'}>
              {LEVEL_LABEL[item.eventLevel] ?? item.eventLevel}
            </Badge>
          </Field>
          <Field label="事件原因">{item.eventReason || '—'}</Field>
          <Field label="发生时间">{formatTime(item.occurredAt)}</Field>
          <Field label="入库时间">{formatTime(item.createdAt)}</Field>
          {eventDataText ? (
            <Field label="事件数据">
              <pre className="whitespace-pre-wrap break-all rounded-md bg-muted p-3 font-mono text-xs">
                {eventDataText}
              </pre>
            </Field>
          ) : null}
        </div>
      )}
    </PageShell>
  )
}
