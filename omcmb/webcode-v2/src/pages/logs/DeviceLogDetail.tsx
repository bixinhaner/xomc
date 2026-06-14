import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Download } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { PageShell, formatBytes, formatTime } from '@/components/layout/PageShell'

import { useStationLogList, useDownloadStationLog } from '@core/hooks/api/useStationLog'
import type { StationLogFile } from '@core/services/api/stationLogApi'

// ===========================================================================
// 基站日志详情（/logs/device/:id）— stationLogApi 无单条端点，从列表中按 id 命中
// （列表 staleTime 30s，跳转前多半已缓存；否则首拉一页 200 条覆盖常见场景）
// ===========================================================================

const LOG_TYPE_LABEL: Record<StationLogFile['logType'], string> = {
  running: '运行日志',
  fault: '故障日志',
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1 border-b py-3 last:border-b-0">
      <span className="text-xs uppercase tracking-wider text-muted-foreground">{label}</span>
      <span className="text-sm">{children}</span>
    </div>
  )
}

export default function DeviceLogDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data, isLoading, isError, error } = useStationLogList({ page: 1, pageSize: 200 })
  const download = useDownloadStationLog()

  const item = useMemo(
    () => (data?.items ?? []).find((r) => r.id === id) ?? null,
    [data, id]
  )

  return (
    <PageShell
      title="基站日志详情"
      description={id ? `日志 ID ${id}` : undefined}
      toolbar={
        <Button variant="outline" size="sm" onClick={() => navigate('/logs/device')}>
          <ArrowLeft className="size-4" /> 返回列表
        </Button>
      }
    >
      {isLoading ? (
        <div className="rounded-lg border bg-card p-8 text-center text-sm text-muted-foreground">
          加载中…
        </div>
      ) : isError ? (
        <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-8 text-center text-sm text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </div>
      ) : !item ? (
        <div className="rounded-lg border bg-card p-8 text-center text-sm text-muted-foreground">
          未找到该日志记录（可能已被清理，或不在最近 200 条内）
        </div>
      ) : (
        <div className="max-w-2xl rounded-lg border bg-card px-5 py-2">
          <Field label="设备 SN">
            <span className="font-mono">{item.deviceSn || '—'}</span>
          </Field>
          <Field label="日志类型">
            <Badge variant={item.logType === 'fault' ? 'destructive' : 'default'}>
              {LOG_TYPE_LABEL[item.logType]}
            </Badge>
          </Field>
          <Field label="文件名">
            <span className="font-mono text-xs break-all">{item.fileName || '—'}</span>
          </Field>
          <Field label="文件大小">{formatBytes(item.fileSize)}</Field>
          <Field label="故障原因">{item.faultReason || '—'}</Field>
          <Field label="故障详情">{item.faultDetail || '—'}</Field>
          <Field label="存储位置">
            <span className="font-mono text-xs break-all">
              {item.bucket ? `${item.bucket}/${item.objectPath}` : item.objectPath || '—'}
            </span>
          </Field>
          <Field label="任务 ID">
            <span className="font-mono text-xs">{item.taskId || '—'}</span>
          </Field>
          <Field label="采集时间">{formatTime(item.collectedAt)}</Field>
          <Field label="入库时间">{formatTime(item.createdAt)}</Field>
          <div className="py-3">
            <Button
              size="sm"
              disabled={download.isPending}
              onClick={() => download.mutate(item.id)}
            >
              <Download className="size-4" /> 下载日志文件
            </Button>
          </div>
        </div>
      )}
    </PageShell>
  )
}
