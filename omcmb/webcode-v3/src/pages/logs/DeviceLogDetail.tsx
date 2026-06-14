import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Download, FileText, HardDrive } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatBytes, formatTime } from '@/lib/format'
import {
  useStationLogList,
  useDownloadStationLog,
} from '@core/hooks/api/useStationLog'
import type { StationLogFile } from '@core/services/api/stationLogApi'

import { DRow, FeedError, FeedLoading } from './_shared'

// ──────────────────────────────────────────────────────────────────────────
// /logs/device/:id —— 基站日志文件详情（DeviceLog 列表下钻）
// stationLogApi 无 getById：以较大 pageSize 拉列表并按 id 命中（从列表跳入命中缓存，
// 直接深链则触发一次真实列表查询后定位），从而保证以真实数据加载而非恒定"未找到"。
// ──────────────────────────────────────────────────────────────────────────

const TYPE_META: Record<StationLogFile['logType'], { label: string; color: string }> = {
  running: { label: '运行日志', color: '#00f0ff' },
  fault: { label: '故障日志', color: '#ff7a1a' },
}

export default function DeviceLogDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  // 拉取一页（较大）以便深链命中；列表跳入则直接命中 React Query 缓存。
  const { data, isLoading, isError, error, isFetching } = useStationLogList({
    page: 1,
    pageSize: 500,
  })
  const download = useDownloadStationLog()

  const record = useMemo<StationLogFile | undefined>(
    () => data?.items.find((r) => r.id === id),
    [data, id],
  )

  const meta = record ? TYPE_META[record.logType] : undefined

  return (
    <PageShell
      code="F06"
      title="DEVICE LOG · 日志详情"
      subtitle={record ? record.deviceSn : (id ?? '—')}
      isFetching={isFetching}
      toolbar={
        <div className="flex items-center gap-2">
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/logs/device')}>
            BACK
          </NeonButton>
          {record && (
            <NeonButton
              icon={<Download />}
              onClick={() => download.mutate(record.id)}
              disabled={download.isPending}
            >
              {download.isPending ? '…' : 'DOWNLOAD'}
            </NeonButton>
          )}
        </div>
      }
    >
      {isLoading ? (
        <FeedLoading />
      ) : isError ? (
        <FeedError error={error} />
      ) : !record ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <HardDrive className="size-12 text-emerald-400/50" />
          <div className="font-mono text-xs uppercase tracking-[0.2em] text-emerald-300/70">
            未找到日志 · {id}
          </div>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/logs/device')}>
            返回列表
          </NeonButton>
        </div>
      ) : (
        <div className="flex flex-col gap-4">
          <div className="flex flex-wrap items-center gap-3">
            <span
              className="chip"
              style={{ color: meta?.color ?? '#00f0ff' }}
            >
              {meta?.label ?? record.logType}
            </span>
            <StatusBadge
              status={record.isDeleted ? 'offline' : 'ok'}
              label={record.isDeleted ? '已删除' : '有效'}
            />
            <span className="font-mono text-[11px] text-cyan-300/55">
              {formatBytes(record.fileSize)} · {formatTime(record.collectedAt)}
            </span>
          </div>

          <GlassPanel strong title="文件信息" meta={record.bucket || 'MINIO'}>
            <dl className="grid grid-cols-[120px_1fr] gap-x-4 gap-y-2.5 p-4 text-[12px]">
              <DRow k="设备 SN" v={record.deviceSn} mono />
              <DRow k="设备 ID" v={record.deviceId} mono />
              <DRow k="日志类型" v={meta?.label ?? record.logType} color={meta?.color} />
              <DRow k="文件名" v={record.fileName} mono />
              <DRow k="对象路径" v={record.objectPath} mono />
              <DRow k="存储桶" v={record.bucket} mono />
              <DRow k="文件大小" v={formatBytes(record.fileSize)} />
              <DRow k="关联任务" v={record.taskId} mono />
              <DRow k="采集时间" v={formatTime(record.collectedAt)} mono />
              <DRow k="创建时间" v={formatTime(record.createdAt)} mono />
              <DRow k="更新时间" v={formatTime(record.updatedAt)} mono />
            </dl>
          </GlassPanel>

          {(record.faultReason || record.faultDetail) && (
            <GlassPanel strong title="故障原因" meta={<FileText className="size-3.5" />}>
              <dl className="grid grid-cols-[120px_1fr] gap-x-4 gap-y-2.5 p-4 text-[12px]">
                <DRow k="主因" v={record.faultReason} color="#ff7a1a" />
                <DRow k="明细" v={record.faultDetail} />
              </dl>
              {record.faultDetail && (
                <div className="terminal mx-4 mb-4 max-h-[40vh] overflow-auto whitespace-pre-wrap break-all p-3 text-[11px] text-emerald-100/85">
                  {record.faultDetail}
                </div>
              )}
            </GlassPanel>
          )}
        </div>
      )}
    </PageShell>
  )
}
