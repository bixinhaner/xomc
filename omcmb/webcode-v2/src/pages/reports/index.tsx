import { useMemo, useState } from 'react'
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
  Pagination,
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'

import { useReportDefinitions } from '@core/hooks/api/useReports'

export function ReportsPage() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const params = useMemo(() => ({ page, pageSize }), [page, pageSize])

  const { data, isLoading, isError, error, isFetching, refetch } =
    useReportDefinitions(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['报表', '类型', '周期', '自动生成', '状态', '创建时间']

  return (
    <PageShell
      title="报表中心"
      description="报表定义与生成记录"
      isFetching={isFetching}
      toolbar={
        <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
          <RefreshCcw /> 刷新
        </Button>
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
              <EmptyRow colSpan={cols.length}>暂无报表定义</EmptyRow>
            ) : (
              rows.map((r) => (
                <TableRow key={r.id}>
                  <TableCell>
                    <div className="font-medium">{r.reportName}</div>
                    <div className="text-xs text-muted-foreground line-clamp-1">
                      {r.description}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{r.reportType}</Badge>
                  </TableCell>
                  <TableCell className="text-xs">{r.period}</TableCell>
                  <TableCell className="text-xs">
                    {r.autoGenerate ? (
                      <Badge variant="success">启用</Badge>
                    ) : (
                      <Badge variant="muted">关闭</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge variant={r.status === 'published' ? 'success' : 'muted'}>
                      {r.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(r.createTime)}
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
