import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
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

import { usePMFileDevices } from '@core/hooks/api/usePerformance'

// ============================================================
// 性能文件获取（对齐 v1 file/perf-retrieval）
//   按设备聚合的 PM 性能文件视图（usePMFileDevices）：基站 / 产品类 / 文件数 / 上报状态
//   行 → 性能文件详情 /files/perf-detail/:sn（查看该设备的 PM 文件明细）
//   数据全走 @core React Query hooks，三态完整。
// ============================================================

const PAGE_SIZE = 20

export default function PerfRetrieval() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [productClass, setProductClass] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(productClass.trim() ? { productClass: productClass.trim() } : {}),
    }),
    [page, keyword, productClass],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = usePMFileDevices(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['设备 SN', '基站名称', '产品类', '首次采集', '最近采集', '文件数', '上报状态']

  return (
    <PageShell
      title="性能文件获取"
      description="按设备聚合的 PM 性能计数器采集文件"
    >
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-56 pl-9"
            placeholder="搜索设备 SN"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <Input
          className="w-44"
          placeholder="产品类"
          value={productClass}
          onChange={(e) => {
            setProductClass(e.target.value)
            setPage(1)
          }}
        />
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
              <EmptyRow colSpan={cols.length}>暂无 PM 采集文件</EmptyRow>
            ) : (
              rows.map((d) => (
                <TableRow key={d.deviceSn}>
                  <TableCell>
                    <button
                      type="button"
                      className="font-mono text-xs text-primary hover:underline"
                      onClick={() => navigate(`/file/perf-retrieval`)}
                    >
                      {d.deviceSn}
                    </button>
                  </TableCell>
                  <TableCell className="text-xs">{d.siteName || '—'}</TableCell>
                  <TableCell className="text-xs">{d.productClass || '—'}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(d.firstCollectTime)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(d.lastCollectTime)}
                  </TableCell>
                  <TableCell className="tabular-nums">{d.fileCount.toLocaleString()}</TableCell>
                  <TableCell>
                    {d.reporting ? (
                      <Badge variant="default">上报中</Badge>
                    ) : (
                      <Badge variant="muted">已停止</Badge>
                    )}
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
