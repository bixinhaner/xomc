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
  formatBytes,
  formatTime,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
} from '@/components/layout/PageShell'

import { useMRFiles, useDownloadMRFile } from '@core/hooks/api/useMR'

// ============================================================
// 测量报告 → 设备文件明细（带参 /mr/files/:sn）
// 对照 v1 DeviceFilesDrawer —— 单台设备的 MR 文件列表 + 逐个下载。
// 数据走真实 useMRFiles（GET /mr/files?device_sn=...）+ useDownloadMRFile。
// 支持按 MR 类型（MRO/MRS/MRE）筛选当前页。
// ============================================================

const PAGE_SIZE = 20

const MR_TYPE_VARIANT: Record<string, 'default' | 'success' | 'warning' | 'outline'> = {
  MRO: 'default',
  MRE: 'success',
  MRS: 'warning',
}

const MR_TYPES = ['MRO', 'MRS', 'MRE']

export default function DeviceFiles() {
  const { sn = '' } = useParams<{ sn: string }>()
  const deviceSn = decodeURIComponent(sn)
  const navigate = useNavigate()

  const [page, setPage] = useState(1)
  const [mrType, setMrType] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      deviceSn,
      ...(mrType ? { mrType } : {}),
    }),
    [page, deviceSn, mrType],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useMRFiles(params)
  const download = useDownloadMRFile()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['MR 类型', '文件名', '文件大小', '记录数', '采集时间', '解析', '操作']

  return (
    <PageShell
      title="设备 MR 文件"
      description={deviceSn ? `设备 SN ${deviceSn}` : undefined}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => navigate('/mr/files')}>
            <ArrowLeft className="size-4" /> 返回
          </Button>
          <select
            className="h-9 rounded-md border bg-background px-3 text-sm"
            value={mrType}
            onChange={(e) => {
              setMrType(e.target.value)
              setPage(1)
            }}
          >
            <option value="">全部类型</option>
            {MR_TYPES.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>
          <span className="text-sm text-muted-foreground">共 {total} 个文件</span>
          <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
            <RefreshCcw className={isFetching ? 'animate-spin' : ''} /> 刷新
          </Button>
        </div>
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
              <EmptyRow colSpan={cols.length}>该设备暂无 MR 文件</EmptyRow>
            ) : (
              rows.map((f) => (
                <TableRow key={f.id}>
                  <TableCell>
                    <Badge variant={MR_TYPE_VARIANT[f.mrType] ?? 'outline'}>{f.mrType}</Badge>
                  </TableCell>
                  <TableCell className="max-w-md truncate font-mono text-xs" title={f.fileName}>
                    {f.fileName}
                  </TableCell>
                  <TableCell className="tabular-nums text-xs">{formatBytes(f.fileSize)}</TableCell>
                  <TableCell className="tabular-nums">{f.recordCount.toLocaleString()}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(f.collectTime)}
                  </TableCell>
                  <TableCell>
                    {f.parsed ? (
                      <Badge variant="success">已解析</Badge>
                    ) : (
                      <Badge variant="muted">未解析</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={download.isPending}
                      onClick={() => download.mutate(f.id)}
                    >
                      <Download className="size-4" /> 下载
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
