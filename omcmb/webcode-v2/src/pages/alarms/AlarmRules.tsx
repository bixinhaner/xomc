import { useCallback, useMemo, useState } from 'react'
import { Plus, RefreshCcw, Search, Trash2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
import { cn } from '@/lib/utils'

import {
  useAlarmRules,
  useCreateAlarmRule,
  useDeleteAlarmRules,
  useUpdateAlarmRule,
} from '@core/hooks/api/useAlarms'
import type { AlarmRule } from '@core/types/alarm'
import {
  buildAlarmRuleActions,
  buildAlarmRuleConditions,
  type AlarmRuleSelectionMode,
} from '@core/utils/alarmRuleConditions'

import { AlarmRuleDialog } from './AlarmRuleDialog'

// 执行动作展示
const RULE_TYPE_META: Record<string, { label: string; variant: 'default' | 'destructive' | 'warning' | 'secondary' }> = {
  default: { label: '默认', variant: 'secondary' },
  ignore: { label: '禁止上报', variant: 'destructive' },
  auto_acknowledge: { label: '自动确认', variant: 'default' },
  auto_clear: { label: '自动清除', variant: 'warning' },
}

function ruleTypeMeta(t: string) {
  return RULE_TYPE_META[t] ?? { label: t || '—', variant: 'secondary' as const }
}

export default function AlarmRules() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [action, setAction] = useState('')
  const [enabled, setEnabled] = useState('')

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<AlarmRule | null>(null)
  const [dialogMode, setDialogMode] = useState<'add' | 'edit' | 'view'>('add')
  const [togglingId, setTogglingId] = useState<string | null>(null)

  const params = useMemo(() => {
    const p: {
      page: number
      pageSize: number
      keyword?: string
      action?: string
      enabled?: string
    } = { page, pageSize }
    if (keyword.trim()) p.keyword = keyword.trim()
    if (action) p.action = action
    if (enabled) p.enabled = enabled
    return p
  }, [page, pageSize, keyword, action, enabled])

  const { data, isLoading, isError, error, isFetching, refetch } =
    useAlarmRules(params)
  const rules = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const updateRule = useUpdateAlarmRule()
  const deleteRules = useDeleteAlarmRules()
  const createRule = useCreateAlarmRule()

  const handleToggle = useCallback(
    async (rule: AlarmRule) => {
      setTogglingId(rule.id)
      try {
        await updateRule.mutateAsync({
          id: rule.id,
          data: { enabled: !rule.enabled },
        })
        void refetch()
      } catch {
        /* hook 已 invalidate；失败保持原状 */
      } finally {
        setTogglingId(null)
      }
    },
    [updateRule, refetch]
  )

  const handleDelete = useCallback(
    async (rule: AlarmRule) => {
      if (rule.isDefault || rule.enabled) return
      try {
        await deleteRules.mutateAsync([rule.id])
        void refetch()
      } catch {
        /* 静默 */
      }
    },
    [deleteRules, refetch]
  )

  const openCreate = useCallback(() => {
    setDialogMode('add')
    setEditingRule(null)
    setDialogOpen(true)
  }, [])

  const openEdit = useCallback((rule: AlarmRule) => {
    setDialogMode('edit')
    setEditingRule(rule)
    setDialogOpen(true)
  }, [])

  const openView = useCallback((rule: AlarmRule) => {
    setDialogMode('view')
    setEditingRule(rule)
    setDialogOpen(true)
  }, [])

  const handleSubmit = useCallback(
    async (form: {
      ruleName: string
      ruleType: string
      enabled: boolean
      deviceSelectionMode: AlarmRuleSelectionMode
      selectedDevices: string[]
      selectedGroups: string[]
      selectedAlarms: string[]
    }) => {
      const payload = {
        ruleName: form.ruleName,
        ruleType: form.ruleType,
        enabled: form.enabled,
        severity: 'warning' as const,
        conditions: buildAlarmRuleConditions(form),
        actions: buildAlarmRuleActions(form.ruleType),
      }
      if (editingRule) {
        await updateRule.mutateAsync({ id: editingRule.id, data: payload })
      } else {
        await createRule.mutateAsync(payload)
      }
      setDialogOpen(false)
      void refetch()
    },
    [editingRule, updateRule, createRule, refetch]
  )

  const cols = [
    '状态',
    '规则名称',
    '执行动作',
    '操作人',
    '更新时间',
    '操作',
  ]

  return (
    <PageShell
      title="告警规则"
      description="告警过滤规则 · 禁止上报 / 自动确认 / 自动清除"
      isFetching={isFetching}
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-56 pl-9"
              placeholder="规则名称"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>

          <Select
            value={action || 'all'}
            onValueChange={(v) => {
              setAction(v === 'all' ? '' : v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="执行动作" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部动作</SelectItem>
              <SelectItem value="ignore">禁止上报</SelectItem>
              <SelectItem value="auto_acknowledge">自动确认</SelectItem>
              <SelectItem value="auto_clear">自动清除</SelectItem>
            </SelectContent>
          </Select>

          <Select
            value={enabled || 'all'}
            onValueChange={(v) => {
              setEnabled(v === 'all' ? '' : v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-28">
              <SelectValue placeholder="状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              <SelectItem value="true">已启用</SelectItem>
              <SelectItem value="false">已停用</SelectItem>
            </SelectContent>
          </Select>

          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
          </Button>
          <Button size="sm" className="ml-auto" onClick={openCreate}>
            <Plus /> 新建规则
          </Button>
        </>
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
            ) : rules.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无告警规则</EmptyRow>
            ) : (
              rules.map((r) => {
                const meta = ruleTypeMeta(r.ruleType)
                return (
                  <TableRow key={r.id}>
                    <TableCell>
                      <button
                        type="button"
                        disabled={togglingId === r.id}
                        onClick={() => handleToggle(r)}
                        className={cn(
                          'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
                          r.enabled ? 'bg-primary' : 'bg-muted',
                          togglingId === r.id && 'opacity-50'
                        )}
                        aria-label="切换启用状态"
                      >
                        <span
                          className={cn(
                            'inline-block size-3.5 transform rounded-full bg-white transition-transform',
                            r.enabled ? 'translate-x-4' : 'translate-x-1'
                          )}
                        />
                      </button>
                    </TableCell>
                    <TableCell>
                      <button
                        type="button"
                        className="flex items-center gap-2 hover:underline"
                        onClick={() => openView(r)}
                      >
                        {r.isDefault ? (
                          <Badge variant="secondary">默认</Badge>
                        ) : null}
                        <span>{r.ruleName}</span>
                      </button>
                    </TableCell>
                    <TableCell>
                      <Badge variant={meta.variant}>{meta.label}</Badge>
                    </TableCell>
                    <TableCell className="text-xs">{r.userCode || '—'}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(r.updateTime)}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 px-2 text-xs"
                          onClick={() => openView(r)}
                        >
                          查看
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 px-2 text-xs"
                          disabled={r.enabled}
                          onClick={() => openEdit(r)}
                        >
                          编辑
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 px-2 text-xs text-destructive"
                          disabled={r.enabled || r.isDefault || deleteRules.isPending}
                          onClick={() => handleDelete(r)}
                        >
                          <Trash2 className="size-3.5" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={pageSize}
        onChange={setPage}
      />

      <AlarmRuleDialog
        open={dialogOpen}
        mode={dialogMode}
        rule={editingRule}
        loading={createRule.isPending || updateRule.isPending}
        onSubmit={handleSubmit}
        onCancel={() => setDialogOpen(false)}
      />
    </PageShell>
  )
}
