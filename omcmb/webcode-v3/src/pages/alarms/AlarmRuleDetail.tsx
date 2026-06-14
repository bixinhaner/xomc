import { useCallback, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  AlertTriangle,
  ArrowLeft,
  Loader2,
  Power,
  RefreshCcw,
  ShieldQuestion,
  Trash2,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import {
  useAlarmRules,
  useUpdateAlarmRule,
  useDeleteAlarmRules,
} from '@core/hooks/api/useAlarms'
import type { AlarmRule, AlarmRuleAction, AlarmRuleCondition } from '@core/types/alarm'
import type { AlarmSeverity } from '@core/types/common'

const SEV_COLOR: Record<AlarmSeverity, string> = {
  critical: '#ff2d6f',
  major: '#ff7a1a',
  minor: '#ffd400',
  warning: '#5b9eff',
}
const SEV_LABEL: Record<AlarmSeverity, string> = {
  critical: '紧急',
  major: '重要',
  minor: '次要',
  warning: '警告',
}

const RULE_TYPE_LABEL: Record<string, { label: string; color: string }> = {
  default: { label: '默认', color: '#5b9eff' },
  ignore: { label: '禁止上报', color: '#ff2d6f' },
  auto_acknowledge: { label: '自动确认', color: '#00ff88' },
  auto_clear: { label: '自动清除', color: '#ff7a1a' },
}

const FIELD_LABEL: Record<string, string> = {
  alarm_identifier: '告警标识',
  alarm_source: '告警源',
  device_group_id: '设备组',
  device_id: '设备',
}

const OPERATOR_LABEL: Record<AlarmRuleCondition['operator'], string> = {
  eq: '等于',
  ne: '不等于',
  gt: '大于',
  lt: '小于',
  gte: '大于等于',
  lte: '小于等于',
  contains: '包含',
  startsWith: '以…开头',
  endsWith: '以…结尾',
}

const ACTION_TYPE_LABEL: Record<AlarmRuleAction['type'], string> = {
  notify: '通知',
  email: '邮件',
  sms: '短信',
  webhook: 'Webhook',
  suppress: '抑制',
}

function condValue(value: AlarmRuleCondition['value']): string {
  if (Array.isArray(value)) return value.join(', ')
  return String(value)
}

export default function AlarmRuleDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [busy, setBusy] = useState(false)
  const [opError, setOpError] = useState<string | null>(null)

  // 无专用 byId hook：用列表大页拉取后按 id 命中（真实后端数据）
  const { data, isLoading, isError, error, isFetching, refetch } = useAlarmRules({
    page: 1,
    pageSize: 200,
  })
  const updateRule = useUpdateAlarmRule()
  const deleteRules = useDeleteAlarmRules()

  const rule: AlarmRule | undefined = useMemo(
    () => data?.items.find((r) => r.id === id),
    [data, id]
  )

  const onToggle = useCallback(async () => {
    if (!rule) return
    setBusy(true)
    setOpError(null)
    try {
      await updateRule.mutateAsync({ id: rule.id, data: { enabled: !rule.enabled } })
      await refetch()
    } catch (e) {
      setOpError(e instanceof Error ? e.message : '切换状态失败')
    } finally {
      setBusy(false)
    }
  }, [rule, updateRule, refetch])

  const onDelete = useCallback(async () => {
    if (!rule) return
    if (rule.isDefault) {
      setOpError('默认规则不可删除')
      return
    }
    if (rule.enabled) {
      setOpError('请先禁用规则再删除')
      return
    }
    setBusy(true)
    setOpError(null)
    try {
      await deleteRules.mutateAsync([rule.id])
      navigate('/alarms/rules')
    } catch (e) {
      setOpError(e instanceof Error ? e.message : '删除失败')
      setBusy(false)
    }
  }, [rule, deleteRules, navigate])

  const typeCfg = rule
    ? (RULE_TYPE_LABEL[rule.ruleType] ?? { label: rule.ruleType || '—', color: '#5b9eff' })
    : null
  const sevColor = rule ? SEV_COLOR[rule.severity] : '#5b9eff'

  return (
    <PageShell
      code="F04"
      title="RULE DETAIL · 规则详情"
      subtitle={id ? `RULE-ID · ${id}` : 'NO RULE SELECTED'}
      isFetching={isFetching}
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/alarms/rules')}>
            BACK
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {opError && (
        <div className="mb-2 border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">
          OP FAILED · {opError}
        </div>
      )}

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
      ) : !rule ? (
        <div className="flex flex-col items-center justify-center gap-3 py-16">
          <ShieldQuestion className="size-10 text-cyan-300/40" />
          <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/45">
            RULE NOT FOUND · 未找到规则 {id}
          </div>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/alarms/rules')}>
            返回规则列表
          </NeonButton>
        </div>
      ) : (
        <div className="grid grid-cols-12 gap-3">
          {/* 概要 */}
          <GlassPanel
            title="OVERVIEW · 概要"
            meta={typeCfg?.label}
            className="col-span-12 lg:col-span-5"
          >
            <div className="space-y-0">
              <Field label="规则名称">
                <span className="flex items-center gap-2">
                  {rule.isDefault && <span className="chip text-[#5b9eff]">默认规则</span>}
                  <span className="font-display text-sm font-bold text-cyan-100">
                    {rule.ruleName}
                  </span>
                </span>
              </Field>
              <Field label="执行动作">
                <span className="chip" style={{ color: typeCfg?.color }}>
                  {typeCfg?.label}
                </span>
              </Field>
              <Field label="启用状态">
                <StatusBadge
                  status={rule.enabled ? 'active' : 'inactive'}
                  label={rule.enabled ? '已启用' : '已禁用'}
                />
              </Field>
              <Field label="告警级别">
                <span className="chip" style={{ color: sevColor }}>
                  {SEV_LABEL[rule.severity]}
                </span>
              </Field>
              {rule.deviceType && <Field label="告警源类型">{rule.deviceType}</Field>}
              <Field label="操作人">{rule.userCode || '—'}</Field>
              <Field label="创建时间">{formatTime(rule.createTime)}</Field>
              <Field label="更新时间">{formatTime(rule.updateTime)}</Field>
            </div>

            <div className="flex flex-wrap gap-2 border-t border-cyan-500/15 p-3.5">
              {busy ? (
                <span className="flex items-center gap-2 font-mono text-xs text-cyan-300/70">
                  <Loader2 className="size-4 animate-spin" /> 处理中…
                </span>
              ) : (
                <>
                  <NeonButton
                    icon={<Power />}
                    tone={rule.enabled ? 'danger' : 'cyan'}
                    onClick={() => void onToggle()}
                  >
                    {rule.enabled ? '禁用' : '启用'}
                  </NeonButton>
                  <NeonButton
                    icon={<Trash2 />}
                    tone="danger"
                    disabled={rule.isDefault || rule.enabled}
                    onClick={() => void onDelete()}
                  >
                    删除
                  </NeonButton>
                </>
              )}
            </div>
          </GlassPanel>

          {/* 条件 + 动作 */}
          <div className="col-span-12 space-y-3 lg:col-span-7">
            <GlassPanel title="MATCH CONDITIONS · 匹配条件" meta={`${rule.conditions.length} 项`}>
              {rule.conditions.length === 0 ? (
                <div className="px-3.5 py-6 text-center font-mono text-[11px] uppercase tracking-[0.16em] text-cyan-300/45">
                  NO CONDITIONS · 命中全部告警
                </div>
              ) : (
                <div className="divide-y divide-cyan-500/8">
                  {rule.conditions.map((c, i) => (
                    <div
                      key={`${c.field}-${i}`}
                      className="grid grid-cols-[1.2fr_0.8fr_1.6fr] items-center gap-3 px-3.5 py-2.5"
                    >
                      <span className="font-display text-xs font-bold text-cyan-100">
                        {FIELD_LABEL[c.field] ?? c.field}
                      </span>
                      <span className="chip text-cyan-300/80">
                        {OPERATOR_LABEL[c.operator] ?? c.operator}
                      </span>
                      <span className="break-words font-mono text-[11px] text-cyan-200/85">
                        {condValue(c.value)}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </GlassPanel>

            <GlassPanel title="ACTIONS · 执行动作" meta={`${rule.actions.length} 项`}>
              {rule.actions.length === 0 ? (
                <div className="px-3.5 py-6 text-center font-mono text-[11px] uppercase tracking-[0.16em] text-cyan-300/45">
                  NO ACTIONS · 无执行动作
                </div>
              ) : (
                <div className="divide-y divide-cyan-500/8">
                  {rule.actions.map((a, i) => (
                    <div
                      key={`${a.type}-${i}`}
                      className="grid grid-cols-[1fr_2fr] items-center gap-3 px-3.5 py-2.5"
                    >
                      <span className="chip text-[#00f0ff]">
                        {ACTION_TYPE_LABEL[a.type] ?? a.type}
                      </span>
                      <span className="break-words font-mono text-[11px] text-cyan-200/85">
                        {a.target ?? a.template ?? '—'}
                      </span>
                    </div>
                  ))}
                </div>
              )}
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
