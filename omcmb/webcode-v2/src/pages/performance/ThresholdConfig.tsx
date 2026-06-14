import { useMemo, useState } from 'react'
import { RefreshCcw, Search, Trash2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
  formatTime,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useThresholds,
  useUpdateThreshold,
  useDeleteThresholds,
} from '@core/hooks/api/usePerformance'
import type { PageRequest } from '@core/types/pagination'

// ============================================================
// 阈值配置 — 对齐 v1 /performance/threshold
//   阈值规则列表 + 启用/停用切换 + 删除。
// ============================================================

const PAGE_SIZE = 20

const OPERATOR_LABEL: Record<string, string> = {
  gt: '>',
  lt: '<',
  gte: '≥',
  lte: '≤',
  eq: '=',
  ne: '≠',
}

function Stat({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: number | string
  tone?: 'default' | 'emerald' | 'amber' | 'muted'
}) {
  const toneClass = {
    default: 'text-foreground',
    emerald: 'text-emerald-600 dark:text-emerald-400',
    amber: 'text-amber-600 dark:text-amber-400',
    muted: 'text-muted-foreground',
  }[tone]
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>{value}</div>
    </div>
  )
}

export function ThresholdConfigPage() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [pendingId, setPendingId] = useState<string | null>(null)

  const params = useMemo<PageRequest>(() => ({ page, pageSize: PAGE_SIZE }), [page])
  const { data, isLoading, isError, error, isFetching, refetch } = useThresholds(params)
  const updateThreshold = useUpdateThreshold()
  const deleteThresholds = useDeleteThresholds()

  const allRows = data?.items ?? []
  const rows = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return allRows
    return allRows.filter(
      (r) =>
        r.thresholdName.toLowerCase().includes(kw) || r.kpiCode.toLowerCase().includes(kw)
    )
  }, [allRows, keyword])

  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const enabledCount = allRows.filter((r) => r.enabled).length

  const handleToggle = (id: string, next: boolean) => {
    setPendingId(id)
    updateThreshold.mutate({ id, data: { enabled: next } }, { onSettled: () => setPendingId(null) })
  }

  const handleDelete = (id: string) => {
    setPendingId(id)
    deleteThresholds.mutate([id], { onSettled: () => setPendingId(null) })
  }

  const cols = ['名称', 'KPI 编码', '比较', '告警阈值', '严重阈值', '状态', '更新时间', '操作']

  return (
    <PageShell
      title="阈值配置"
      description="KPI 告警/严重阈值规则 · 启用停用与删除"
      isFetching={isFetching}
    >
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="阈值规则" value={total} />
        <Stat label="已启用" value={enabledCount} tone="emerald" />
        <Stat label="已停用" value={Math.max(0, allRows.length - enabledCount)} tone="muted" />
        <Stat label="当前页" value={`${page}/${totalPages}`} tone="muted" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-72 pl-9"
            placeholder="阈值名称 / KPI 编码（本页过滤）"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
        <Button
          variant="outline"
          size="sm"
          className="ml-auto"
          disabled={isFetching || updateThreshold.isPending || deleteThresholds.isPending}
          onClick={() => void refetch()}
        >
          <RefreshCcw className="size-4" /> 刷新
        </Button>
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
              <EmptyRow colSpan={cols.length}>暂无阈值规则</EmptyRow>
            ) : (
              rows.map((r) => {
                const busy = pendingId === r.id
                return (
                  <TableRow key={r.id} className={cn(busy && 'opacity-50')}>
                    <TableCell className="font-medium">{r.thresholdName}</TableCell>
                    <TableCell className="font-mono text-xs">{r.kpiCode}</TableCell>
                    <TableCell className="font-mono text-xs">
                      {OPERATOR_LABEL[r.operator] ?? r.operator}
                    </TableCell>
                    <TableCell className="tabular-nums">
                      {r.warningValue}
                      {r.unit}
                    </TableCell>
                    <TableCell className="tabular-nums">
                      {r.criticalValue}
                      {r.unit}
                    </TableCell>
                    <TableCell>
                      <Badge variant={r.enabled ? 'success' : 'muted'}>
                        {r.enabled ? '启用' : '停用'}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(r.updateTime)}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1.5">
                        <Button
                          size="sm"
                          variant="outline"
                          disabled={busy}
                          onClick={() => handleToggle(r.id, !r.enabled)}
                        >
                          {r.enabled ? '停用' : '启用'}
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          disabled={busy}
                          className="text-destructive hover:text-destructive"
                          onClick={() => handleDelete(r.id)}
                          aria-label="删除阈值"
                        >
                          <Trash2 className="size-4" />
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

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </PageShell>
  )
}

export default ThresholdConfigPage
