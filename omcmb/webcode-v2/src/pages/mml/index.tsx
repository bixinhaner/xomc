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

import { useMMLScripts } from '@core/hooks/api/useMML'

export function MMLPage() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const params = useMemo(() => ({ page, pageSize }), [page, pageSize])

  const { data, isLoading, isError, error, isFetching, refetch } = useMMLScripts(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['脚本', '类型', '状态', '进度', '设备类型', '创建人', '更新时间']

  return (
    <PageShell
      title="MML 脚本"
      description="人机命令脚本与批量任务"
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
              <EmptyRow colSpan={cols.length}>暂无脚本</EmptyRow>
            ) : (
              rows.map((s) => (
                <TableRow key={s.id}>
                  <TableCell>
                    <div className="font-medium">{s.scriptName}</div>
                    <div className="text-xs text-muted-foreground line-clamp-1">
                      {s.description}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{s.type}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={s.status === 'active' ? 'success' : 'muted'}>
                      {s.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs tabular-nums">
                    {typeof s.progress === 'number' ? `${s.progress}%` : '—'}
                  </TableCell>
                  <TableCell className="text-xs">{s.deviceType}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">{s.creator}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(s.updateTime)}
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
