import { X, User } from 'lucide-react'

import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useAlarmById } from '@core/hooks/api/useAlarms'
import type { Alarm, DealState, EventType } from '@core/types/alarm'
import type { AlarmSeverity } from '@core/types/common'

const SEV_LABEL: Record<AlarmSeverity, string> = {
  critical: '紧急',
  major: '重要',
  minor: '次要',
  warning: '警告',
}
const SEV_COLOR: Record<AlarmSeverity, string> = {
  critical: '#ff2d6f',
  major: '#ff7a1a',
  minor: '#ffd400',
  warning: '#5b9eff',
}
const EVENT_LABEL: Record<EventType, string> = {
  communication: '通信告警',
  qualityOfService: '服务质量',
  processingError: '处理错误',
  device: '设备告警',
  environment: '环境告警',
  performance: '性能告警',
}
const DEAL_LABEL: Record<DealState, string> = {
  '0': '未确认未清除',
  '1': '已确认未清除',
  '2': '未确认已清除',
  '3': '已确认已清除',
}

export function neTypeLabel(value?: string | null): string {
  if (!value) return '—'
  const map: Record<string, string> = {
    eNB: 'eNB (LTE)',
    lte: 'eNB (LTE)',
    LTE: 'eNB (LTE)',
    gNB: 'gNB (NR)',
    nr: 'gNB (NR)',
    NR: 'gNB (NR)',
    '5G NR': 'gNB (NR)',
    GSM: 'GSM',
    gsm: 'GSM',
  }
  if (map[value]) return map[value]
  const lower = value.toLowerCase()
  const hit = Object.keys(map).find((k) => k.toLowerCase() === lower)
  return hit ? map[hit] : value
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[120px_1fr] gap-3 border-b border-cyan-500/10 px-3.5 py-2 last:border-b-0">
      <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">
        {label}
      </div>
      <div className="break-words text-[12px] text-cyan-100/90">{children ?? '—'}</div>
    </div>
  )
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <div className="border-b border-cyan-500/15 bg-cyan-500/[0.04] px-3.5 py-1.5 font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/80">
      {children}
    </div>
  )
}

export function AlarmDetailPanel({
  alarm,
  open,
  onClose,
}: {
  alarm: Alarm | null
  open: boolean
  onClose: () => void
}) {
  // 详情挂载时按 id 拉权威详情，缺失回退列表行
  const { data: fetched } = useAlarmById(open ? (alarm?.id ?? '') : '')
  const a = fetched ?? alarm

  if (!open) return null

  const isConfirmed = a?.dealState === '1' || a?.dealState === '3'
  const isCleared = a?.dealState === '2' || a?.dealState === '3'
  const extraEntries = Object.entries(a?.additionalInfo ?? {}).filter(
    ([k]) =>
      k !== 'additional_text' &&
      k !== 'managed_object_instance' &&
      k !== 'additional_information'
  )
  const additionalText = a?.additionalText || a?.additionalInfo?.['additional_text']
  const additionalInformation = a?.additionalInfo?.['additional_information']

  return (
    <div className="fixed inset-0 z-50 flex justify-end" role="dialog" aria-modal="true">
      {/* 遮罩 */}
      <button
        type="button"
        aria-label="关闭"
        className="absolute inset-0 bg-black/55 backdrop-blur-sm"
        onClick={onClose}
      />
      {/* 抽屉 */}
      <div className="glass-strong warp-in relative flex h-full w-[460px] max-w-[92vw] flex-col border-l border-cyan-500/25">
        <div className="flex items-center justify-between border-b border-cyan-500/20 px-4 py-3">
          <div className="flex min-w-0 items-center gap-2">
            {a && (
              <span
                className="chip shrink-0"
                style={{ color: SEV_COLOR[a.severity] }}
              >
                {SEV_LABEL[a.severity] ?? a.severity}
              </span>
            )}
            <span className="truncate font-display text-sm font-bold text-cyan-100">
              {a?.probableCause || a?.alarmName || a?.description || '告警详情'}
            </span>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="shrink-0 rounded-sm border border-cyan-500/25 p-1 text-cyan-300/70 hover:border-cyan-400/60 hover:text-cyan-200"
          >
            <X className="size-4" />
          </button>
        </div>

        <div className="flex-1 overflow-auto">
          {!a ? (
            <div className="py-16 text-center font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
              NO DATA
            </div>
          ) : (
            <div className="space-y-4 p-4">
              <div className="glass rounded-sm">
                <SectionTitle>基本信息 · BASIC</SectionTitle>
                <Field label="告警标识">
                  <span className="font-mono">{a.alarmIdentifier || '—'}</span>
                </Field>
                <Field label="可能原因">{a.probableCause || '—'}</Field>
                <Field label="具体故障">{a.description || '—'}</Field>
                <Field label="严重程度">
                  <span style={{ color: SEV_COLOR[a.severity] }}>
                    {SEV_LABEL[a.severity] ?? a.severity}
                  </span>
                </Field>
                <Field label="事件类型">{EVENT_LABEL[a.eventType] ?? '未知'}</Field>
                <Field label="告警次数">{a.alarmCount ?? '—'}</Field>
              </div>

              <div className="glass rounded-sm">
                <SectionTitle>设备信息 · DEVICE</SectionTitle>
                <Field label="设备 SN">
                  <span className="font-mono">{a.deviceSn || '—'}</span>
                </Field>
                <Field label="设备名称">{a.deviceName || '—'}</Field>
                <Field label="网元类型">{neTypeLabel(a.neType)}</Field>
              </div>

              <div className="glass rounded-sm">
                <SectionTitle>状态与时间 · STATUS</SectionTitle>
                <Field label="告警状态">
                  <StatusBadge
                    status={a.dealState === '0' ? 'critical' : isCleared ? 'ok' : 'warn'}
                    label={DEAL_LABEL[a.dealState] ?? '未知'}
                  />
                </Field>
                <Field label="故障时间">{formatTime(a.eventTime)}</Field>
                <Field label="更新时间">{formatTime(a.updTime)}</Field>
                {isConfirmed && (
                  <Field label="确认人">
                    <span className="inline-flex items-center gap-1.5">
                      <User className="size-3 text-cyan-300/60" />
                      {a.dealUser || '—'}
                    </span>
                  </Field>
                )}
                {isConfirmed && a.dealTime && (
                  <Field label="确认时间">{formatTime(a.dealTime)}</Field>
                )}
                {isCleared && (
                  <Field label="清除人">
                    <span className="inline-flex items-center gap-1.5">
                      <User className="size-3 text-cyan-300/60" />
                      {a.clearUser || '—'}
                    </span>
                  </Field>
                )}
                {isCleared && <Field label="清除时间">{formatTime(a.clearTime)}</Field>}
              </div>

              <div className="glass rounded-sm">
                <SectionTitle>处理信息 · HANDLING</SectionTitle>
                <Field label="确认备注">{a.dealMemo || '—'}</Field>
                {isCleared && <Field label="清除备注">{a.clearMemo || '—'}</Field>}
              </div>

              {(additionalText || additionalInformation || extraEntries.length > 0) && (
                <div className="glass rounded-sm">
                  <SectionTitle>附加信息 · EXTRA</SectionTitle>
                  {additionalText && <Field label="附加文本">{additionalText}</Field>}
                  {additionalInformation && (
                    <Field label="附加描述">{additionalInformation}</Field>
                  )}
                  {extraEntries.map(([k, v]) => (
                    <Field key={k} label={k}>
                      {v || '—'}
                    </Field>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
