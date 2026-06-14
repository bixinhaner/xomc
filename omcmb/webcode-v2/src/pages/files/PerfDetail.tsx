import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Download, RefreshCcw } from 'lucide-react'

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

import { useDownloadPMFile, usePMFiles } from '@core/hooks/api/usePerformance'

// ============================================================
// 性能文件详情（按设备）—— /files/perf-detail/:sn
//   useParams 取设备 SN，usePMFiles({ deviceSn }) 加载该设备的 PM 文件
//   展示制式/运营商/计数器数/是否解析；逐个下载（useDownloadPMFile）
//   数据走真实后端 hook（usePMFiles enabled: !useMock）。
// ============================================================

const PAGE_SIZE = 20

export default function PerfDetail() {
  const { sn = '' } = useParams<{ sn: string }>()
  const navigate = useNavigate()

  const [page, setPage] = useState(1)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      deviceSn: sn,
    }),
    [page, sn],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = usePMFiles(params)
  const download = useDownloadPMFile()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['文件名', '制式', '运营商', '大小', '计数器数', '已解析', '采集时间', '操作']

  return (
    <PageShell
      title={`性能文件 · ${sn}`}
      description="该设备采集的 PM 性能计数器文件明细"
      toolbar={
        <Button variant="outline" size="sm" onClick={() => navigate('/files/perf-retrieval')}>
          <ArrowLeft /> 返回列表
        </Button>
      }
    >
      <div className="mb-3 flex items-center gap-2">
        <span className="text-sm text-muted-foreground">共 {total} 个文件</span>
        <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
          <RefreshCcw className={isFetching ? 'animate-spin' : ''} /> 刷新
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
              <EmptyRow colSpan={cols.length}>该设备暂无 PM 文件</EmptyRow>
            ) : (
              rows.map((f) => (
                <TableRow key={f.id}>
                  <TableCell className="font-medium">{f.fileName}</TableCell>
                  <TableCell className="text-xs uppercase">{f.technology || '—'}</TableCell>
                  <TableCell className="text-xs uppercase text-muted-foreground">
                    {f.carrier || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatBytes(f.fileSize)}
                  </TableCell>
                  <TableCell className="tabular-nums text-xs">
                    {f.counterCount.toLocaleString()}
                  </TableCell>
                  <TableCell>
                    {f.parsed ? (
                      <Badge variant="success">已解析</Badge>
                    ) : (
                      <Badge variant="muted">未解析</Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(f.collectTime)}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-8 px-2"
                      disabled={download.isPending}
                      onClick={() => download.mutate(f.id)}
                    >
                      <Download /> 下载
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </PageShell>
  )
}
