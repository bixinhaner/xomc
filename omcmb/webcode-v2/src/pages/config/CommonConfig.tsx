import { useState } from 'react'
import { RefreshCcw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
  formatTime,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useSysConfigsByCategory } from '@core/hooks/api/useSystem'

// ============================================================
// 通用配置 — 对照 v1 config/CommonConfig（系统/网络/安全参数）。
// 真实数据：useSysConfigsByCategory(category)（sys_configs 表）。
// 按类别 Tab 切换，展示键/值/类型/说明/更新时间（只读视图）。
// ============================================================

const CATEGORIES: { key: string; label: string }[] = [
  { key: 'system', label: '系统参数' },
  { key: 'network', label: '网络参数' },
  { key: 'security', label: '安全参数' },
]

export default function CommonConfig() {
  const [category, setCategory] = useState('system')

  const { data, isLoading, isError, error, isFetching, refetch } =
    useSysConfigsByCategory(category)
  const rows = data ?? []
  const cols = ['配置键', '值', '类型', '公开', '说明', '更新时间']

  return (
    <PageShell
      title="通用配置"
      description="系统级配置项（sys_configs）— 按类别查看"
      isFetching={isFetching}
      toolbar={
        <Button variant="outline" size="sm" className="ml-auto" onClick={() => void refetch()}>
          <RefreshCcw className="size-4" /> 刷新
        </Button>
      }
    >
      <div className="mb-4 flex flex-wrap items-center gap-1 border-b">
        {CATEGORIES.map((c) => {
          const active = category === c.key
          return (
            <button
              key={c.key}
              type="button"
              onClick={() => setCategory(c.key)}
              className={cn(
                'border-b-2 px-3 py-2 text-sm font-medium transition-colors',
                active
                  ? 'border-primary text-foreground'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              )}
            >
              {c.label}
            </button>
          )
        })}
      </div>

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
              <EmptyRow colSpan={cols.length}>该类别暂无配置项</EmptyRow>
            ) : (
              rows.map((item) => (
                <TableRow key={item.id}>
                  <TableCell className="font-mono text-xs">{item.key}</TableCell>
                  <TableCell className="max-w-xs truncate text-sm">{item.value || '—'}</TableCell>
                  <TableCell>
                    <Badge variant="muted">{item.valueType || 'string'}</Badge>
                  </TableCell>
                  <TableCell>
                    {item.isPublic ? (
                      <Badge variant="success">是</Badge>
                    ) : (
                      <Badge variant="muted">否</Badge>
                    )}
                  </TableCell>
                  <TableCell className="max-w-sm truncate text-xs text-muted-foreground">
                    {item.description || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(item.updatedAt)}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>
    </PageShell>
  )
}
