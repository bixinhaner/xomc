import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { EmptyRow, TableCard, formatTime } from '@/components/layout/PageShell'

import { useAlarmRules } from '@core/hooks/api/useAlarms'
import type { AlarmRule } from '@core/types/alarm'

const RULE_TYPE_LABEL: Record<string, string> = {
  default: '默认',
  ignore: '禁止上报',
  auto_acknowledge: '自动确认',
  auto_clear: '自动清除',
}

const OPERATOR_LABEL: Record<string, string> = {
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

const FIELD_LABEL: Record<string, string> = {
  alarm_identifier: '告警标识',
  alarm_source: '告警源',
  device_id: '设备',
  device_group_id: '设备组',
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

function formatCondValue(value: AlarmRule['conditions'][number]['value']): string {
  if (Array.isArray(value)) return value.join('、')
  return String(value)
}

export default function AlarmRuleDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  // 列表接口是唯一查询入口；本页从列表里按 id 取目标规则（拉满一页足够覆盖）
  const { data, isLoading, isError, error } = useAlarmRules({
    page: 1,
    pageSize: 200,
  })

  const rule = useMemo(
    () => data?.items.find((r) => r.id === id) ?? null,
    [data, id]
  )

  return (
    <div className="mx-auto max-w-[1000px] px-6 py-8">
      <Button
        variant="ghost"
        size="sm"
        className="mb-4 -ml-2"
        onClick={() => navigate('/alarms/rules')}
      >
        <ArrowLeft /> 返回规则列表
      </Button>

      {isLoading ? (
        <div className="flex items-center gap-2 py-16 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" /> 加载中
        </div>
      ) : isError ? (
        <div className="py-16 text-center text-sm text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </div>
      ) : !rule ? (
        <div className="py-16 text-center text-sm text-muted-foreground">
          未找到该告警规则
        </div>
      ) : (
        <>
          <div className="mb-4 flex items-center gap-2">
            <h1 className="text-2xl font-semibold tracking-tight">
              {rule.ruleName}
            </h1>
            {rule.isDefault ? <Badge variant="secondary">默认规则</Badge> : null}
            <Badge variant={rule.enabled ? 'default' : 'muted'}>
              {rule.enabled ? '已启用' : '已停用'}
            </Badge>
          </div>

          <Card className="mb-4">
            <CardHeader>
              <CardTitle className="text-sm">基本信息</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-x-6">
                <Field
                  label="执行动作"
                  value={RULE_TYPE_LABEL[rule.ruleType] ?? rule.ruleType}
                />
                <Field label="告警源" value={rule.deviceType} />
                <Field label="操作人" value={rule.userCode} />
                <Field label="创建时间" value={formatTime(rule.createTime)} />
                <Field label="更新时间" value={formatTime(rule.updateTime)} />
              </div>
            </CardContent>
          </Card>

          <Card className="mb-4">
            <CardHeader>
              <CardTitle className="text-sm">命中条件</CardTitle>
            </CardHeader>
            <CardContent>
              <TableCard>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>字段</TableHead>
                      <TableHead>运算符</TableHead>
                      <TableHead>值</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {rule.conditions.length === 0 ? (
                      <EmptyRow colSpan={3}>无条件（匹配全部）</EmptyRow>
                    ) : (
                      rule.conditions.map((c, i) => (
                        <TableRow key={`${c.field}-${i}`}>
                          <TableCell className="text-xs">
                            {FIELD_LABEL[c.field] ?? c.field}
                          </TableCell>
                          <TableCell className="text-xs">
                            {OPERATOR_LABEL[c.operator] ?? c.operator}
                          </TableCell>
                          <TableCell className="text-xs">
                            {formatCondValue(c.value)}
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </TableCard>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-sm">执行动作</CardTitle>
            </CardHeader>
            <CardContent>
              <TableCard>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>类型</TableHead>
                      <TableHead>目标</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {rule.actions.length === 0 ? (
                      <EmptyRow colSpan={2}>无动作</EmptyRow>
                    ) : (
                      rule.actions.map((a, i) => (
                        <TableRow key={`${a.type}-${i}`}>
                          <TableCell className="text-xs">{a.type}</TableCell>
                          <TableCell className="text-xs">
                            {RULE_TYPE_LABEL[a.target ?? ''] ?? a.target ?? '—'}
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </TableCard>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}
