import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Download } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { PageShell, formatBytes, formatTime } from '@/components/layout/PageShell'

import {
  useAbnormalReboot,
  useDownloadAbnormalReboot,
} from '@core/hooks/api/useDeviceAbnormalReboot'
import type { RecordStatus } from '@core/services/api/deviceAbnormalRebootApi'

import { formatRuntime } from './_format'

// ===========================================================================
// 异常重启详情（/logs/exception/:id）— 真实单条端点 deviceAbnormalRebootApi.getById
// ===========================================================================

const STATUS_LABEL: Record<RecordStatus, string> = {
  detected: '已检测',
  file_received: '文件已收',
  collection_failed: '采集失败',
}

const STATUS_VARIANT: Record<RecordStatus, 'default' | 'success' | 'destructive'> = {
  detected: 'default',
  file_received: 'success',
  collection_failed: 'destructive',
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1 border-b py-3 last:border-b-0">
      <span className="text-xs uppercase tracking-wider text-muted-foreground">{label}</span>
      <span className="text-sm">{children}</span>
    </div>
  )
}

export default function ExceptionLogDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data: item, isLoading, isError, error } = useAbnormalReboot(id)
  const download = useDownloadAbnormalReboot()

  const hasFile = Boolean(item?.fileName) && item?.recordStatus === 'file_received'

  return (
    <PageShell
      title="异常重启详情"
      description={item ? item.deviceSn : id ? `记录 ID ${id}` : undefined}
      toolbar={
        <Button variant="outline" size="sm" onClick={() => navigate('/logs/exception')}>
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
          未找到该记录
        </div>
      ) : (
        <div className="max-w-2xl rounded-lg border bg-card px-5 py-2">
          <Field label="设备名称">{item.deviceName || '—'}</Field>
          <Field label="设备 SN">
            <span className="font-mono">{item.deviceSn || '—'}</span>
          </Field>
          <Field label="制式">{item.deviceType || '—'}</Field>
          <Field label="管理 IP">
            <span className="font-mono text-xs">{item.operateIp || '—'}</span>
          </Field>
          <Field label="软件版本">
            <span className="font-mono text-xs">{item.softwareVersion || '—'}</span>
          </Field>
          <Field label="采集状态">
            <Badge variant={STATUS_VARIANT[item.recordStatus] ?? 'muted'}>
              {STATUS_LABEL[item.recordStatus] ?? item.recordStatus}
            </Badge>
          </Field>
          <Field label="停机主因">{item.haltMainReason || '—'}</Field>
          <Field label="停机详因">{item.haltDetailReason || '—'}</Field>
          <Field label="重启前运行时长">{formatRuntime(item.runtimeBeforeReboot)}</Field>
          {item.collectionFailReason ? (
            <Field label="采集失败原因">
              <span className="text-destructive">{item.collectionFailReason}</span>
            </Field>
          ) : null}
          <Field label="日志文件名">
            <span className="font-mono text-xs break-all">{item.fileName || '—'}</span>
          </Field>
          <Field label="文件大小">{formatBytes(item.fileSize)}</Field>
          <Field label="发生时间">{formatTime(item.collectedAt)}</Field>
          <Field label="入库时间">{formatTime(item.createdAt)}</Field>
          {hasFile ? (
            <div className="py-3">
              <Button
                size="sm"
                disabled={download.isPending}
                onClick={() => download.mutate(item.id)}
              >
                <Download className="size-4" /> 下载故障日志
              </Button>
            </div>
          ) : null}
        </div>
      )}
    </PageShell>
  )
}
