import { useEffect, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import type { AlarmRule } from '@core/types/alarm'

export interface AlarmRuleFormValue {
  ruleName: string
  ruleType: string
  enabled: boolean
  alarmIdentifiers: string[]
}

// 从规则的 conditions 反解出已选告警标识（与 AlarmRules.handleSubmit 写入口径一致）
function extractIdentifiers(rule: AlarmRule | null): string[] {
  if (!rule) return []
  const cond = rule.conditions.find((c) => c.field === 'alarm_identifier')
  if (!cond) return []
  if (Array.isArray(cond.value)) return cond.value.map((v) => String(v))
  return cond.value != null ? [String(cond.value)] : []
}

export function AlarmRuleDialog({
  open,
  rule,
  loading,
  onSubmit,
  onCancel,
}: {
  open: boolean
  rule: AlarmRule | null
  loading?: boolean
  onSubmit: (value: AlarmRuleFormValue) => void
  onCancel: () => void
}) {
  const [ruleName, setRuleName] = useState('')
  const [ruleType, setRuleType] = useState('ignore')
  const [enabled, setEnabled] = useState(false)
  const [identifiers, setIdentifiers] = useState('')

  useEffect(() => {
    if (!open) return
    setRuleName(rule?.ruleName ?? '')
    setRuleType(rule?.ruleType && rule.ruleType !== 'default' ? rule.ruleType : 'ignore')
    setEnabled(rule?.enabled ?? false)
    setIdentifiers(extractIdentifiers(rule).join(', '))
  }, [open, rule])

  if (!open) return null

  const canSubmit = ruleName.trim().length > 0 && !loading

  const submit = () => {
    if (!canSubmit) return
    onSubmit({
      ruleName: ruleName.trim(),
      ruleType,
      enabled,
      alarmIdentifiers: identifiers
        .split(/[,，\s]+/)
        .map((s) => s.trim())
        .filter(Boolean),
    })
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40" onClick={onCancel} aria-hidden />
      <div className="relative w-full max-w-md rounded-lg border bg-background p-5 shadow-xl">
        <h2 className="text-base font-semibold">
          {rule ? '编辑告警规则' : '新建告警规则'}
        </h2>

        <div className="mt-4 space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="rule-name">规则名称</Label>
            <Input
              id="rule-name"
              value={ruleName}
              placeholder="输入规则名称"
              maxLength={50}
              onChange={(e) => setRuleName(e.target.value)}
              autoFocus
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="rule-type">执行动作</Label>
            <Select value={ruleType} onValueChange={setRuleType}>
              <SelectTrigger id="rule-type">
                <SelectValue placeholder="选择执行动作" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="ignore">禁止上报</SelectItem>
                <SelectItem value="auto_acknowledge">自动确认</SelectItem>
                <SelectItem value="auto_clear">自动清除</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="rule-ids">告警标识</Label>
            <Input
              id="rule-ids"
              value={identifiers}
              placeholder="多个标识用逗号分隔，留空匹配全部"
              onChange={(e) => setIdentifiers(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              规则命中条件：告警标识包含所列任一值
            </p>
          </div>

          <div className="flex items-center justify-between">
            <Label htmlFor="rule-enabled">立即启用</Label>
            <button
              id="rule-enabled"
              type="button"
              onClick={() => setEnabled((v) => !v)}
              className={
                'relative inline-flex h-5 w-9 items-center rounded-full transition-colors ' +
                (enabled ? 'bg-primary' : 'bg-muted')
              }
              aria-label="切换启用"
            >
              <span
                className={
                  'inline-block size-3.5 transform rounded-full bg-white transition-transform ' +
                  (enabled ? 'translate-x-4' : 'translate-x-1')
                }
              />
            </button>
          </div>
        </div>

        <div className="mt-5 flex justify-end gap-2">
          <Button variant="outline" size="sm" onClick={onCancel} disabled={loading}>
            取消
          </Button>
          <Button size="sm" onClick={submit} disabled={!canSubmit}>
            {rule ? '保存' : '创建'}
          </Button>
        </div>
      </div>
    </div>
  )
}
