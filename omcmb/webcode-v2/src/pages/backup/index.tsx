import { useMemo, useState } from 'react'
import { RefreshCcw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
  formatBytes,
  formatTime,
} from '@/components/layout/PageShell'

import { useBackupTasks } from '@core/hooks/api/useBackup'

export function BackupPage() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [status, setStatus] = useState('')

  const params = useMemo(
    () => ({ page, pageSize, ...(status ? { status } : {}) }),
    [page, pageSize, status]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useBackupTasks(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['任务', '类型', '状态', '进度', '设备', '大小', '创建时间']

  return (
    <PageShell
      title="备份管理"
      description="设备配置/全量备份任务列表"
      isFetching={isFetching}
      toolbar={
        <>
          <Select
            value={status || 'all'}
            onValueChange={(v) => {
              setStatus(v === 'all' ? '' : v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              <SelectItem value="pending">待启动</SelectItem>
              <SelectItem value="running">运行中</SelectItem>
              <SelectItem value="success">已成功</SelectItem>
              <SelectItem value="failed">失败</SelectItem>
              <SelectItem value="cancelled">已取消</SelectItem>
            </SelectContent>
          </Select>
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
              <EmptyRow colSpan={cols.length}>暂无任务</EmptyRow>
            ) : (
              rows.map((t) => (
                <TableRow key={t.id}>
                  <TableCell className="font-medium">{t.taskName}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{t.taskType === 'manual' ? '手动' : '定时'}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        t.status === 'success'
                          ? 'success'
                          : t.status === 'failed'
                            ? 'destructive'
                            : t.status === 'running'
                              ? 'warning'
                              : 'muted'
                      }
                    >
                      {t.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs">
                    {t.successCount}/{t.totalCount}
                    {t.failCount > 0 ? (
                      <span className="text-destructive"> · 失败 {t.failCount}</span>
                    ) : null}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {t.deviceSns?.length ?? 0} 台
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatBytes(t.fileSize)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(t.createdAt)}
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
