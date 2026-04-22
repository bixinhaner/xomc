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
  formatBytes,
  formatTime,
} from '@/components/layout/PageShell'

import { useSoftwareVersions } from '@core/hooks/api/useSoftware'

export function SoftwarePage() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const params = useMemo(() => ({ page, pageSize }), [page, pageSize])

  const { data, isLoading, isError, error, isFetching, refetch } =
    useSoftwareVersions(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['版本号', '版本名称', '设备型号', '厂商', '状态', '文件大小', '发布日期']

  return (
    <PageShell
      title="软件版本"
      description="设备固件版本库 · 灰度升级由独立计划驱动"
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
              <EmptyRow colSpan={cols.length}>暂无版本</EmptyRow>
            ) : (
              rows.map((v) => (
                <TableRow key={v.id}>
                  <TableCell className="font-mono">{v.versionCode}</TableCell>
                  <TableCell>{v.versionName}</TableCell>
                  <TableCell>{v.deviceType}</TableCell>
                  <TableCell>{v.vendor}</TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        v.status === 'current'
                          ? 'success'
                          : v.status === 'beta'
                            ? 'warning'
                            : 'muted'
                      }
                    >
                      {v.status === 'current'
                        ? '当前'
                        : v.status === 'beta'
                          ? '测试'
                          : v.status === 'deprecated'
                            ? '已废弃'
                            : '已归档'}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatBytes(v.fileSize)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(v.releaseDate)}
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
