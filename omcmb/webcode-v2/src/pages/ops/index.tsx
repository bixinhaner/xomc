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

import { useOpsTemplates } from '@core/hooks/api/useOpsTools'

export function OpsPage() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const params = useMemo(() => ({ page, pageSize }), [page, pageSize])

  const { data, isLoading, isError, error, isFetching, refetch } = useOpsTemplates(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['模板', '类别', '适用设备', '步骤数', '耗时（分）', '使用次数', '更新时间']

  return (
    <PageShell
      title="运维工具箱"
      description="自动化模板与批量任务"
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
              <EmptyRow colSpan={cols.length}>暂无模板</EmptyRow>
            ) : (
              rows.map((t) => (
                <TableRow key={t.id}>
                  <TableCell>
                    <div className="font-medium">{t.templateName}</div>
                    <div className="text-xs text-muted-foreground line-clamp-1">
                      {t.description}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{t.category}</Badge>
                  </TableCell>
                  <TableCell className="text-xs">
                    {(t.targetDeviceTypes || []).join(', ') || '—'}
                  </TableCell>
                  <TableCell className="text-xs tabular-nums">{t.steps?.length ?? 0}</TableCell>
                  <TableCell className="text-xs tabular-nums">{t.estimatedDuration}</TableCell>
                  <TableCell className="text-xs tabular-nums">{t.useCount}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(t.updateTime)}
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
