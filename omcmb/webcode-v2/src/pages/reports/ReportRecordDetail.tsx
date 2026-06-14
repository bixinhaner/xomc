import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Download, RefreshCcw, FileText } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { PageShell, formatBytes, formatTime } from '@/components/layout/PageShell'

import {
  useReportRecords,
  useReportDefinitionById,
  useDownloadReport,
} from '@core/hooks/api/useReports'
import type { ReportRecord } from '@core/mock/data/reports'

// ============================================================
// 报表生成记录详情 (report/lte-standard 行 → /reports/record/:id) — v2
// 真实数据：useReportRecords 列表中按 :id 命中目标记录，再用其
//   reportDefinitionId 拉定义元信息（useReportDefinitionById）。
//   提供下载操作。
// ============================================================

const RECORD_STATUS_LABEL: Record<ReportRecord['status'], string> = {
  generating: '生成中',
  ready: '就绪',
  failed: '失败',
}

const RECORD_STATUS_VARIANT: Record<
  ReportRecord['status'],
  'success' | 'warning' | 'destructive'
> = {
  generating: 'warning',
  ready: 'success',
  failed: 'destructive',
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1 py-2">
      <span className="text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </span>
      <span className="text-sm">{children}</span>
    </div>
  )
}

export default function ReportRecordDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  // 拉一批记录后按 id 命中（列表端点是唯一可用的记录来源）。
  const recordsQuery = useReportRecords({ page: 1, pageSize: 100 })
  const record = useMemo(
    () => (recordsQuery.data?.items ?? []).find((r) => r.id === id),
    [recordsQuery.data, id],
  )

  const defQuery = useReportDefinitionById(record?.reportDefinitionId ?? '')
  const download = useDownloadReport()

  const isLoading = recordsQuery.isLoading
  const isError = recordsQuery.isError
  const notFound = !isLoading && !isError && !record

  const handleDownload = () => {
    if (!record || record.status !== 'ready' || download.isPending) return
    download.mutate(record.id, {
      onSuccess: (res) => {
        if (res?.url) window.open(res.url, '_blank', 'noopener,noreferrer')
      },
    })
  }

  return (
    <PageShell
      title="报表记录详情"
      description={record ? record.reportName : id}
      isFetching={recordsQuery.isFetching || defQuery.isFetching || download.isPending}
      toolbar={
        <>
          <Button variant="outline" size="sm" onClick={() => navigate(-1)}>
            <ArrowLeft /> 返回
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={() => void recordsQuery.refetch()}
          >
            <RefreshCcw /> 刷新
          </Button>
        </>
      }
    >
      {isLoading ? (
        <Card>
          <CardContent className="p-6">
            <div className="h-4 w-40 animate-pulse rounded bg-muted" />
            <div className="mt-3 h-4 w-64 animate-pulse rounded bg-muted" />
          </CardContent>
        </Card>
      ) : isError ? (
        <Card>
          <CardContent className="p-6 text-destructive">
            加载失败：{recordsQuery.error instanceof Error ? recordsQuery.error.message : '未知错误'}
          </CardContent>
        </Card>
      ) : notFound ? (
        <Card>
          <CardContent className="p-6 text-muted-foreground">
            未找到记录 {id}
          </CardContent>
        </Card>
      ) : record ? (
        <div className="space-y-4">
          <Card>
            <CardContent className="p-6">
              <div className="mb-4 flex items-center gap-2">
                <FileText className="size-5 text-primary" />
                <h2 className="text-lg font-semibold">{record.reportName}</h2>
                <Badge variant={RECORD_STATUS_VARIANT[record.status]}>
                  {RECORD_STATUS_LABEL[record.status]}
                </Badge>
              </div>
              <div className="grid grid-cols-2 gap-x-8 md:grid-cols-3">
                <Field label="记录 ID">
                  <span className="font-mono text-xs">{record.id}</span>
                </Field>
                <Field label="周期">{record.period || '—'}</Field>
                <Field label="格式">
                  <Badge variant="muted" className="uppercase">
                    {record.format}
                  </Badge>
                </Field>
                <Field label="文件大小">
                  <span className="tabular-nums">{formatBytes(record.fileSize)}</span>
                </Field>
                <Field label="生成时间">{formatTime(record.generateTime)}</Field>
                <Field label="定义 ID">
                  <span className="font-mono text-xs">{record.reportDefinitionId}</span>
                </Field>
              </div>
              <div className="mt-4">
                <Button
                  disabled={record.status !== 'ready' || download.isPending}
                  onClick={handleDownload}
                >
                  <Download /> 下载报表
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* 关联报表定义 */}
          {defQuery.data && (
            <Card>
              <CardContent className="p-6">
                <div className="mb-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">
                  关联报表定义
                </div>
                <div className="grid grid-cols-2 gap-x-8 md:grid-cols-3">
                  <Field label="定义名称">{defQuery.data.reportName}</Field>
                  <Field label="类型">
                    <Badge variant="outline">{defQuery.data.reportType}</Badge>
                  </Field>
                  <Field label="周期">{defQuery.data.period}</Field>
                  <Field label="创建人">{defQuery.data.creator}</Field>
                  <Field label="自动生成">
                    {defQuery.data.autoGenerate ? (
                      <Badge variant="success">启用</Badge>
                    ) : (
                      <Badge variant="muted">关闭</Badge>
                    )}
                  </Field>
                  <Field label="最近生成">{formatTime(defQuery.data.lastGenTime)}</Field>
                </div>
                {defQuery.data.description && (
                  <p className="mt-3 text-sm text-muted-foreground">
                    {defQuery.data.description}
                  </p>
                )}
              </CardContent>
            </Card>
          )}
        </div>
      ) : null}
    </PageShell>
  )
}
