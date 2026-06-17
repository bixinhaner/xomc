import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Download, RefreshCcw, Search } from 'lucide-react'

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
  formatBytes,
  formatTime,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
} from '@/components/layout/PageShell'

import { useReportRecords, useDownloadReport } from '@core/hooks/api/useReports'
import type { ReportRecord } from '@core/mock/data/reports'

// ============================================================
// 测量报告 → 分析报告
// 对照 v1 webcode/src/pages/mr/Reports —— MR 分析报告列表 + 下载。
// v1 列表为 mock；v2 改走真实 useReportRecords（GET /reports/records）
// + useDownloadReport（GET /reports/records/:id/download）。
// 报告名称单元格链接到报告详情子路由 /mr/reports/:id。
// ============================================================

const PAGE_SIZE = 20

const RECORD_STATUS_LABEL: Record<ReportRecord['status'], string> = {
  generating: '生成中',
  ready: '就绪',
  failed: '失败',
}

const RECORD_STATUS_VARIANT: Record<
  ReportRecord['status'],
  'warning' | 'success' | 'destructive'
> = {
  generating: 'warning',
  ready: 'success',
  failed: 'destructive',
}

export default function Reports() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')

  const { data, isLoading, isError, error, isFetching, refetch } = useReportRecords({
    page,
    pageSize: PAGE_SIZE,
  })
  const download = useDownloadReport()

  const allRows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const rows = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return allRows
    return allRows.filter((r) => r.reportName.toLowerCase().includes(kw))
  }, [allRows, keyword])

  const handleDownload = (r: ReportRecord) => {
    download.mutate(r.id, {
      onSuccess: (res) => {
        if (res?.url) window.open(res.url, '_blank')
      },
    })
  }

  const cols = ['报告名称', '周期', '格式', '状态', '生成时间', '文件大小', '操作']

  return (
    <PageShell
      title="MR 分析报告"
      description="MR 覆盖/干扰/移动性等分析报告生成记录与下载"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-72 pl-9"
              placeholder="搜索报告名称"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <span className="text-sm text-muted-foreground">共 {total} 份报告</span>
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
              <EmptyRow colSpan={cols.length}>暂无分析报告</EmptyRow>
            ) : (
              rows.map((r) => (
                <TableRow key={r.id}>
                  <TableCell>
                    <button
                      type="button"
                      className="text-left text-sm font-medium text-primary hover:underline"
                      onClick={() => navigate(`/mr/reports`)}
                    >
                      {r.reportName}
                    </button>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">{r.period || '—'}</TableCell>
                  <TableCell className="text-xs uppercase">{r.format || '—'}</TableCell>
                  <TableCell>
                    <Badge variant={RECORD_STATUS_VARIANT[r.status]}>
                      {RECORD_STATUS_LABEL[r.status]}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(r.generateTime)}
                  </TableCell>
                  <TableCell className="tabular-nums text-xs">{formatBytes(r.fileSize)}</TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={r.status !== 'ready' || download.isPending}
                      onClick={() => handleDownload(r)}
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
