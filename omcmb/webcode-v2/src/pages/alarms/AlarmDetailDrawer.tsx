import { X } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { formatTime } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useAlarmById } from '@core/hooks/api/useAlarms'
import type { Alarm, DealState, EventType } from '@core/types/alarm'
import type { AlarmSeverity } from '@core/types/common'

const SEVERITY_LABEL: Record<AlarmSeverity, string> = {
  critical: '紧急',
  major: '重要',
  minor: '次要',
  warning: '警告',
}
const SEVERITY_VARIANT: Record<
  AlarmSeverity,
  'destructive' | 'warning' | 'default' | 'muted'
> = {
  critical: 'destructive',
  major: 'destructive',
  minor: 'warning',
  warning: 'warning',
}

const DEAL_STATE_LABEL: Record<DealState, string> = {
  '0': '未确认未清除',
  '1': '已确认未清除',
  '2': '未确认已清除',
  '3': '已确认已清除',
}

const EVENT_TYPE_LABEL: Record<EventType, string> = {
  communication: '通信告警',
  qualityOfService: '服务质量告警',
  processingError: '处理出错告警',
  device: '设备告警',
  environment: '环境告警',
  performance: '业务质量告警',
}

function Field({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5 py-2">
      <span className="text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </span>
      <span className="break-words text-sm">{value || '—'}</span>
    </div>
  )
}

export function AlarmDetailDrawer({
  alarm,
  open,
  onClose,
}: {
  alarm: Alarm | null
  open: boolean
  onClose: () => void
}) {
  // 进一步从后端补全详情（命中时覆盖列表行的精简字段）
  const { data: fetched } = useAlarmById(alarm?.id ?? '')
  const resolved = fetched ?? alarm

  if (!open) return null

  const additionalEntries = Object.entries(resolved?.additionalInfo ?? {}).filter(
    ([key]) =>
      key !== 'additional_text' &&
      key !== 'managed_object_instance' &&
      key !== 'additional_information'
  )

  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      {/* 遮罩 */}
      <div
        className="absolute inset-0 bg-black/40"
        onClick={onClose}
        aria-hidden
      />
      {/* 抽屉 */}
      <div className="relative flex h-full w-full max-w-xl flex-col border-l bg-background shadow-xl">
        <div className="flex items-center justify-between border-b px-5 py-4">
          <div className="flex items-center gap-2">
            {resolved ? (
              <Badge variant={SEVERITY_VARIANT[resolved.severity]}>
                {SEVERITY_LABEL[resolved.severity]}
              </Badge>
            ) : null}
            <h2 className="text-base font-semibold">
              {resolved?.probableCause ||
                resolved?.alarmName ||
                resolved?.description ||
                '告警详情'}
            </h2>
          </div>
          <Button variant="ghost" size="icon" onClick={onClose} aria-label="关闭">
            <X />
          </Button>
        </div>

        <div className="flex-1 overflow-auto px-5 py-4">
          {!resolved ? (
            <div className="py-16 text-center text-sm text-muted-foreground">
              暂无数据
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-x-6">
              <Field label="告警标识" value={<span className="font-mono">{resolved.alarmIdentifier}</span>} />
              <Field
                label="处理状态"
                value={
                  <span
                    className={cn(
                      resolved.dealState === '0' && 'text-destructive',
                      (resolved.dealState === '1' || resolved.dealState === '3') &&
                        'text-emerald-600 dark:text-emerald-400'
                    )}
                  >
                    {DEAL_STATE_LABEL[resolved.dealState]}
                  </span>
                }
              />
              <Field label="可能原因" value={resolved.alarmName} />
              <Field label="具体故障" value={resolved.specificProblem} />
              <Field label="设备名称" value={resolved.deviceName} />
              <Field
                label="设备 SN"
                value={<span className="font-mono">{resolved.deviceSn}</span>}
              />
              <Field label="网元类型" value={resolved.neType} />
              <Field label="事件类型" value={EVENT_TYPE_LABEL[resolved.eventType]} />
              <Field label="告警源" value={resolved.alarmSource} />
              <Field label="告警次数" value={resolved.alarmCount} />
              <Field label="故障时间" value={formatTime(resolved.eventTime)} />
              <Field label="更新时间" value={formatTime(resolved.updTime)} />
              {resolved.dealUser ? (
                <Field label="确认人" value={resolved.dealUser} />
              ) : null}
              {resolved.dealTime ? (
                <Field label="确认时间" value={formatTime(resolved.dealTime)} />
              ) : null}
              {resolved.dealMemo ? (
                <Field label="确认备注" value={resolved.dealMemo} />
              ) : null}
              {resolved.clearUser ? (
                <Field label="清除人" value={resolved.clearUser} />
              ) : null}
              {resolved.clearTime ? (
                <Field label="清除时间" value={formatTime(resolved.clearTime)} />
              ) : null}
              {resolved.clearMemo ? (
                <Field label="清除备注" value={resolved.clearMemo} />
              ) : null}
              <div className="col-span-2">
                <Field label="告警内容" value={resolved.description} />
              </div>
              {resolved.additionalText ? (
                <div className="col-span-2">
                  <Field label="附加信息" value={resolved.additionalText} />
                </div>
              ) : null}
              {additionalEntries.length > 0 ? (
                <div className="col-span-2">
                  <div className="py-2">
                    <span className="text-xs uppercase tracking-wider text-muted-foreground">
                      扩展属性
                    </span>
                    <div className="mt-1 rounded-md border bg-muted/30 p-2">
                      {additionalEntries.map(([k, v]) => (
                        <div
                          key={k}
                          className="flex justify-between gap-3 py-0.5 text-xs"
                        >
                          <span className="text-muted-foreground">{k}</span>
                          <span className="break-all text-right font-mono">{v}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              ) : null}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
