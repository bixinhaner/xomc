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
  formatBytes,
  formatTime,
} from '@/components/layout/PageShell'

import { useFileList } from '@core/hooks/api/useFiles'

export function FilesPage() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({ page, pageSize, ...(keyword.trim() ? { keyword: keyword.trim() } : {}) }),
    [page, pageSize, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useFileList(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['文件名', '类型', '状态', '大小', '设备', '上传时间', '上传者']

  return (
    <PageShell
      title="文件管理"
      description="PM / MR / 固件 / 备份 / 自定义文件库"
      isFetching={isFetching}
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-72 pl-9"
              placeholder="搜索文件名"
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
              <EmptyRow colSpan={cols.length}>暂无文件</EmptyRow>
            ) : (
              rows.map((f) => (
                <TableRow key={f.id}>
                  <TableCell className="font-medium">{f.fileName}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{f.fileType}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        f.status === 'available'
                          ? 'success'
                          : f.status === 'expired' || f.status === 'deleted'
                            ? 'muted'
                            : 'warning'
                      }
                    >
                      {f.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatBytes(f.fileSize)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {f.deviceName || f.deviceSn || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(f.uploadTime)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">{f.uploader}</TableCell>
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
