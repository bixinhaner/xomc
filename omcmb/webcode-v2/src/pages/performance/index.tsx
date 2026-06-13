import { useMemo, useState } from 'react'
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
} from '@/components/layout/PageShell'

import { useKPIList } from '@core/hooks/api/usePerformance'

export function PerformancePage() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(25)
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({ page, pageSize, ...(keyword.trim() ? { keyword: keyword.trim() } : {}) }),
    [page, pageSize, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useKPIList(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['编码', '名称', '类别', '单位', '描述']

  return (
    <PageShell
      title="性能管理"
      description="KPI 定义库 · 趋势与告警阈值由子模块承接（待补齐）"
      isFetching={isFetching}
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-72 pl-9"
              placeholder="KPI 名称/编码"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
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
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无 KPI</EmptyRow>
            ) : (
              rows.map((k) => (
                <TableRow key={k.id}>
                  <TableCell className="font-mono text-xs">{k.kpiCode}</TableCell>
                  <TableCell className="font-medium">{k.kpiName}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{k.category}</Badge>
                  </TableCell>
                  <TableCell className="text-xs">{k.unit}</TableCell>
                  <TableCell className="text-xs text-muted-foreground line-clamp-1">
                    {k.description}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={setPage} />
    </PageShell>
  )
}
