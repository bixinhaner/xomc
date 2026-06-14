import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Download, Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { PageShell, formatBytes, formatTime } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useReportRecords,
  useReportDefinitionById,
  useDownloadReport,
} from '@core/hooks/api/useReports'
import type { ReportRecord } from '@core/mock/data/reports'

// ============================================================
// 测量报告 → 分析报告详情（带参 /mr/reports/:id）
// 真实数据：useReportRecords 拉一页后按 id 命中记录；命中后再用
// useReportDefinitionById 取定义元信息（类型/描述/周期/KPI）。
// 就绪状态可下载。
// ============================================================

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

type FieldVal = string | number | null | undefined

interface Field {
  label: string
  value: FieldVal
  mono?: boolean
}

function InfoGrid({ fields }: { fields: Field[] }) {
  return (
    <div className="grid grid-cols-1 gap-x-8 gap-y-0 sm:grid-cols-2 lg:grid-cols-3">
      {fields.map((f) => (
        <div
          key={f.label}
          className="flex items-center justify-between gap-4 border-b py-2.5 text-sm"
        >
          <span className="shrink-0 text-muted-foreground">{f.label}</span>
          <span
            className={cn('truncate text-right', f.mono && 'font-mono text-xs')}
            title={f.value != null ? String(f.value) : undefined}
          >
            {f.value === '' || f.value == null ? '—' : f.value}
          </span>
        </div>
      ))}
    </div>
  )
}

export default function ReportDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data, isLoading, isError, error, isFetching } = useReportRecords({
    page: 1,
    pageSize: 500,
  })
  const download = useDownloadReport()

  const record: ReportRecord | undefined = useMemo(
    () => (data?.items ?? []).find((r) => r.id === id),
    [data, id],
  )

  const { data: definition } = useReportDefinitionById(record?.reportDefinitionId ?? '')

  const recordFields: Field[] = record
    ? [
        { label: '报告 ID', value: record.id, mono: true },
        { label: '报告名称', value: record.reportName },
        { label: '统计周期', value: record.period },
        { label: '格式', value: (record.format || '').toUpperCase() },
        { label: '生成时间', value: formatTime(record.generateTime) },
        { label: '文件大小', value: formatBytes(record.fileSize) },
      ]
    : []

  const defFields: Field[] = definition
    ? [
        { label: '报告类型', value: definition.reportType },
        { label: '调度周期', value: definition.period },
        { label: '自动生成', value: definition.autoGenerate ? '是' : '否' },
        { label: 'Cron', value: definition.cronExpression || '—', mono: true },
        { label: 'KPI 指标数', value: definition.kpiCodes?.length ?? 0 },
        { label: '创建人', value: definition.creator },
        { label: '描述', value: definition.description },
      ]
    : []

  const handleDownload = () => {
    if (!record) return
    download.mutate(record.id, {
      onSuccess: (res) => {
        if (res?.url) window.open(res.url, '_blank')
      },
    })
  }

  return (
    <PageShell
      title={record ? record.reportName : 'MR 分析报告详情'}
      description={id ? `报告 ${id}` : undefined}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => navigate('/mr/reports')}>
            <ArrowLeft className="size-4" /> 返回
          </Button>
          {record && (
            <>
              <Badge variant={RECORD_STATUS_VARIANT[record.status]}>
                {RECORD_STATUS_LABEL[record.status]}
              </Badge>
              <Button
                variant="outline"
                size="sm"
                className="ml-auto"
                disabled={record.status !== 'ready' || download.isPending}
                onClick={handleDownload}
              >
                {download.isPending ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : (
                  <Download className="size-4" />
                )}
                下载报告
              </Button>
            </>
          )}
        </div>
      }
    >
      {isLoading ? (
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </div>
      ) : isError ? (
        <Card className="flex h-48 items-center justify-center p-6 text-sm text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </Card>
      ) : !record ? (
        <Card className="flex h-48 flex-col items-center justify-center gap-2 p-6 text-muted-foreground">
          <span className="text-sm">未找到报告 {id}</span>
          <Button variant="outline" size="sm" onClick={() => navigate('/mr/reports')}>
            返回报告列表
          </Button>
        </Card>
      ) : (
        <div className="flex flex-col gap-4">
          <Card>
            <CardHeader className="p-4 pb-2">
              <CardTitle className="text-base font-medium">报告记录</CardTitle>
            </CardHeader>
            <CardContent className="p-4 pt-2">
              <InfoGrid fields={recordFields} />
            </CardContent>
          </Card>
          {definition && (
            <Card>
              <CardHeader className="p-4 pb-2">
                <CardTitle className="text-base font-medium">报告定义</CardTitle>
              </CardHeader>
              <CardContent className="p-4 pt-2">
                <InfoGrid fields={defFields} />
              </CardContent>
            </Card>
          )}
        </div>
      )}
    </PageShell>
  )
}
