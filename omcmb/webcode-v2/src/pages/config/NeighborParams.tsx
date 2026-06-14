import { useState } from 'react'
import { RefreshCcw, Search } from 'lucide-react'

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
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'

import { useNeighborParams } from '@core/hooks/api/useConfig'

// ============================================================
// 邻区参数 — 对照 v1 config/NeighborParams。
// 真实数据：useNeighborParams（按源小区可过滤）；展示邻区关系表。
// ============================================================

const NEIGHBOR_TYPE: Record<string, { variant: 'default' | 'secondary' | 'warning'; label: string }> = {
  'intra-freq': { variant: 'default', label: '同频' },
  'inter-freq': { variant: 'secondary', label: '异频' },
  'inter-rat': { variant: 'warning', label: '异系统' },
}

export default function NeighborParams() {
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [sourceCellId, setSourceCellId] = useState('')
  const [pending, setPending] = useState('')

  const { data, isLoading, isError, error, isFetching, refetch } = useNeighborParams({
    page,
    pageSize,
    ...(sourceCellId ? { sourceCellId } : {}),
  })
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['源小区', '目标小区', '邻区类型', '创建时间', '更新时间']

  const applyFilter = () => {
    setSourceCellId(pending.trim())
    setPage(1)
  }

  return (
    <PageShell
      title="邻区参数"
      description="小区邻区关系（同频 / 异频 / 异系统）"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-64 pl-9"
              placeholder="按源小区 ID 过滤"
              value={pending}
              onChange={(e) => setPending(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') applyFilter()
              }}
            />
          </div>
          <Button variant="outline" size="sm" onClick={applyFilter}>
            查询
          </Button>
          {sourceCellId && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setPending('')
                setSourceCellId('')
                setPage(1)
              }}
            >
              清除
            </Button>
          )}
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => void refetch()}>
              <RefreshCcw className="size-4" /> 刷新
            </Button>
          </div>
        </div>
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
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无邻区关系</EmptyRow>
            ) : (
              rows.map((n) => {
                const tp = NEIGHBOR_TYPE[n.neighborType] ?? {
                  variant: 'secondary' as const,
                  label: n.neighborType,
                }
                return (
                  <TableRow key={n.id}>
                    <TableCell>
                      <div className="text-sm">{n.sourceCellName || '—'}</div>
                      <div className="font-mono text-xs text-muted-foreground">{n.sourceCellId}</div>
                    </TableCell>
                    <TableCell>
                      <div className="text-sm">{n.targetCellName || '—'}</div>
                      <div className="font-mono text-xs text-muted-foreground">{n.targetCellId}</div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={tp.variant}>{tp.label}</Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(n.createTime)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(n.updateTime)}
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>
      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={setPage} />
    </PageShell>
  )
}
