import { useEffect, useMemo, useState } from 'react'
import { RefreshCcw, Save } from 'lucide-react'

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
  TableCard,
} from '@/components/layout/PageShell'

import {
  useSysConfigsByCategory,
  useBatchUpdateSysConfigs,
} from '@core/hooks/api/useSystem'
import type { BatchUpdateSysConfigItem } from '@core/types/system'

import { LogNavTabs, Stat } from './_shared'

// ===========================================================================
// 日志保留配置（对齐 v1 log/config）— 审计/业务日志按时间保留天数
// 真实端点：adminApi.getSysConfigsByCategory('log.retention') + batchUpdateSysConfigs
// （后端 seed 000005：audit_days / oper_days / system_days / ne_message_days … + enabled 总开关）
// ===========================================================================

const CATEGORY = 'log.retention'

export default function LogConfig() {
  const { data, isLoading, isError, error, isFetching, refetch } =
    useSysConfigsByCategory(CATEGORY)
  const batchUpdate = useBatchUpdateSysConfigs()

  const items = useMemo(() => data ?? [], [data])

  // 本地编辑态：key -> value（字符串，提交时保留原 value_type）
  const [drafts, setDrafts] = useState<Record<string, string>>({})

  // 后端数据回来后用其值初始化草稿（仅在 key 集合变化时重置，避免覆盖用户正在编辑的输入）
  useEffect(() => {
    setDrafts((prev) => {
      const next: Record<string, string> = {}
      for (const it of items) {
        next[it.key] = it.key in prev ? prev[it.key] : it.value
      }
      return next
    })
  }, [items])

  const dirty = useMemo(
    () => items.some((it) => (drafts[it.key] ?? it.value) !== it.value),
    [items, drafts]
  )

  function handleSave() {
    const changed: BatchUpdateSysConfigItem[] = items
      .filter((it) => (drafts[it.key] ?? it.value) !== it.value)
      .map((it) => ({
        key: it.key,
        value: drafts[it.key] ?? it.value,
        value_type: it.valueType,
      }))
    if (changed.length === 0) return
    batchUpdate.mutate({ category: CATEGORY, items: changed })
  }

  function handleReset() {
    setDrafts(Object.fromEntries(items.map((it) => [it.key, it.value])))
  }

  const cols = ['配置项', '说明', '类型', '当前值', '']

  const enabledItem = items.find((it) => it.key === 'enabled')
  const retentionCount = items.filter((it) => it.key !== 'enabled').length

  return (
    <PageShell
      title="日志保留配置"
      description="审计 / 业务日志按时间保留天数 · 可改可生效（worker 每日 05:00 清理过期行）"
      isFetching={isFetching}
      toolbar={<LogNavTabs />}
    >
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-3">
        <Stat label="配置项总数" value={items.length} />
        <Stat label="保留维度" value={retentionCount} tone="primary" />
        <Stat
          label="清理总开关"
          value={enabledItem ? (enabledItem.value === 'true' ? '开启' : '关闭') : '—'}
          tone={enabledItem?.value === 'true' ? 'emerald' : 'muted'}
        />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        {batchUpdate.isSuccess && !dirty ? (
          <span className="text-xs text-emerald-600 dark:text-emerald-400">已保存</span>
        ) : null}
        {batchUpdate.isError ? (
          <span className="text-xs text-destructive">
            保存失败：
            {batchUpdate.error instanceof Error ? batchUpdate.error.message : '未知错误'}
          </span>
        ) : null}
        <div className="ml-auto flex items-center gap-2">
          {dirty ? (
            <Button variant="ghost" size="sm" onClick={handleReset}>
              撤销修改
            </Button>
          ) : null}
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw className="size-4" /> 刷新
          </Button>
          <Button size="sm" disabled={!dirty || batchUpdate.isPending} onClick={handleSave}>
            <Save className="size-4" /> 保存
          </Button>
        </div>
      </div>

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c, i) => (
                <TableHead key={c || `c-${i}`}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : items.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无日志保留配置</EmptyRow>
            ) : (
              items.map((it) => {
                const draft = drafts[it.key] ?? it.value
                const isDirty = draft !== it.value
                const isBool = it.valueType === 'bool'
                return (
                  <TableRow key={it.key} data-state={isDirty ? 'selected' : undefined}>
                    <TableCell className="font-mono text-xs">{it.key}</TableCell>
                    <TableCell className="max-w-[360px] text-xs text-muted-foreground">
                      {it.description || '—'}
                    </TableCell>
                    <TableCell>
                      <Badge variant="muted">{it.valueType || 'string'}</Badge>
                    </TableCell>
                    <TableCell>
                      {isBool ? (
                        <Select
                          value={draft}
                          onValueChange={(v) =>
                            setDrafts((prev) => ({ ...prev, [it.key]: v }))
                          }
                        >
                          <SelectTrigger className="w-28">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="true">开启</SelectItem>
                            <SelectItem value="false">关闭</SelectItem>
                          </SelectContent>
                        </Select>
                      ) : (
                        <Input
                          className="w-36"
                          type={it.valueType === 'int' || it.valueType === 'float' ? 'number' : 'text'}
                          value={draft}
                          onChange={(e) =>
                            setDrafts((prev) => ({ ...prev, [it.key]: e.target.value }))
                          }
                        />
                      )}
                    </TableCell>
                    <TableCell className="text-right text-xs text-muted-foreground">
                      {isDirty ? <span className="text-amber-600 dark:text-amber-400">已改</span> : null}
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>
    </PageShell>
  )
}
