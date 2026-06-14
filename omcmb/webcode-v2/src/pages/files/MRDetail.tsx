import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Download, RefreshCcw } from 'lucide-react'

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

import { useDownloadMRFile, useMRFiles } from '@core/hooks/api/useMR'

// ============================================================
// MR 文件详情（按设备）—— /files/mr-detail/:sn
//   useParams 取设备 SN，useMRFiles({ deviceSn }) 加载该设备的 MR 文件
//   可按 MR 类型筛选；逐个下载（useDownloadMRFile）
//   数据走真实后端 hook（useMRFiles enabled: !useMock）。
// ============================================================

const PAGE_SIZE = 20

const MR_TYPE_OPTIONS = [
  { value: 'MRO', label: 'MRO' },
  { value: 'MRS', label: 'MRS' },
  { value: 'MRE', label: 'MRE' },
]

function mrTypeVariant(t: string): 'default' | 'success' | 'warning' | 'outline' {
  if (t === 'MRO') return 'default'
  if (t === 'MRS') return 'success'
  if (t === 'MRE') return 'warning'
  return 'outline'
}

export default function MRDetail() {
  const { sn = '' } = useParams<{ sn: string }>()
  const navigate = useNavigate()

  const [page, setPage] = useState(1)
  const [mrType, setMrType] = useState<string>('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      deviceSn: sn,
      ...(mrType ? { mrType } : {}),
    }),
    [page, sn, mrType],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useMRFiles(params)
  const download = useDownloadMRFile()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['文件名', '类型', '运营商', '大小', '记录数', '已解析', '采集时间', '操作']

  return (
    <PageShell
      title={`MR 文件 · ${sn}`}
      description="该设备采集的 MRO/MRS/MRE 测量报告文件明细"
      toolbar={
        <Button variant="outline" size="sm" onClick={() => navigate('/files/mr-retrieval')}>
          <ArrowLeft /> 返回列表
        </Button>
      }
    >
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <Select
          value={mrType || 'all'}
          onValueChange={(v) => {
            setMrType(v === 'all' ? '' : v)
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="MR 类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部类型</SelectItem>
            {MR_TYPE_OPTIONS.map((o) => (
              <SelectItem key={o.value} value={o.value}>
                {o.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
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
              <EmptyRow colSpan={cols.length}>该设备暂无 MR 文件</EmptyRow>
            ) : (
              rows.map((f) => (
                <TableRow key={f.id}>
                  <TableCell className="font-medium">{f.fileName}</TableCell>
                  <TableCell>
                    <Badge variant={mrTypeVariant(f.mrType)}>{f.mrType}</Badge>
                  </TableCell>
                  <TableCell className="text-xs uppercase text-muted-foreground">
                    {f.carrier || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatBytes(f.fileSize)}
                  </TableCell>
                  <TableCell className="tabular-nums text-xs">
                    {f.recordCount.toLocaleString()}
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
