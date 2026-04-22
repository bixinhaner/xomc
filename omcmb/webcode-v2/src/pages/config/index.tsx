import { useMemo, useState } from 'react'
import { RefreshCcw } from 'lucide-react'

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

import { useConfigTemplates } from '@core/hooks/api/useConfig'

export function ConfigPage() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const params = useMemo(() => ({ page, pageSize }), [page, pageSize])

  const { data, isLoading, isError, error, isFetching, refetch } =
    useConfigTemplates(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['模板名称', '参数数量', '创建人', '创建时间']

  return (
    <PageShell
      title="配置管理"
      description="配置模板与基线 · 下发任务由独立流程承接"
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
              <EmptyRow colSpan={cols.length}>暂无配置模板</EmptyRow>
            ) : (
              rows.map((t) => (
                <TableRow key={t.id}>
                  <TableCell>
                    <div className="font-medium">{t.templateName}</div>
                    <div className="text-xs text-muted-foreground line-clamp-1">
                      {t.description}
                    </div>
                  </TableCell>
                  <TableCell className="tabular-nums">{t.params?.length ?? 0}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">{t.creator}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(t.createTime)}
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
