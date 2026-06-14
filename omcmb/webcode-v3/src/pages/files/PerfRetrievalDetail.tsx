import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, RefreshCcw, Download } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatBytes, formatTime } from '@/lib/format'
import { useFileById, useDownloadFile } from '@core/hooks/api/useFiles'
import type { FileStatus } from '@core/mock/data/fileManagement'

import { KV, OverviewStat, StateGate, NEON } from './_shared'

const STATUS_TOKEN: Record<FileStatus, string> = {
  available: 'ok',
  uploading: 'warning',
  processing: 'minor',
  expired: 'offline',
  deleted: 'error',
}
const STATUS_LABEL: Record<FileStatus, string> = {
  available: '可用',
  uploading: '上传中',
  processing: '处理中',
  expired: '已过期',
  deleted: '已删除',
}

export default function PerfRetrievalDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data: file, isLoading, isError, error, isFetching, refetch } = useFileById(id)
  const download = useDownloadFile()

  return (
    <PageShell
      code="F06"
      title={file ? `PERF FILE · ${file.fileName}` : `PERF FILE · ${id.slice(0, 8)}`}
      subtitle="PM RESULT FILE DOSSIER"
      isFetching={isFetching}
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/files/perf-retrieval')}>
            返回列表
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
          {file ? (
            <NeonButton
              icon={<Download />}
              disabled={file.status !== 'available' || download.isPending}
              onClick={() => download.mutate(file.id)}
            >
              {download.isPending ? '下载中…' : 'DOWNLOAD'}
            </NeonButton>
          ) : null}
        </>
      }
    >
      <StateGate
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={!file}
        loadingLabel="LOADING FILE…"
        emptyLabel="FILE NOT FOUND · 文件不存在"
      >
        {file ? (
          <div className="flex flex-col gap-3">
            <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
              <OverviewStat label="SIZE" color={NEON.cyan} value={formatBytes(file.fileSize)} hint="文件大小" />
              <OverviewStat label="STATUS" color={NEON.green} value={STATUS_LABEL[file.status]} hint={file.fileType} />
              <OverviewStat label="DEVICE" color={NEON.violet} value={file.deviceSn || '系统'} hint={file.deviceName || '系统资产'} />
              <OverviewStat label="UPLOADER" color={NEON.gold} value={file.uploader || '—'} hint={formatTime(file.uploadTime)} />
            </div>

            <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
              <GlassPanel title="FILE META">
                <div className="py-1">
                  <KV label="文件名">{file.fileName}</KV>
                  <KV label="类型">{file.fileType}</KV>
                  <KV label="MIME">{file.mimeType}</KV>
                  <KV label="状态"><StatusBadge status={STATUS_TOKEN[file.status]} label={STATUS_LABEL[file.status]} /></KV>
                  <KV label="校验和">{file.checksum}</KV>
                  <KV label="上传时间">{formatTime(file.uploadTime)}</KV>
                  {file.expiryTime ? <KV label="过期时间">{formatTime(file.expiryTime)}</KV> : null}
                </div>
              </GlassPanel>

              <GlassPanel title="ASSOCIATION">
                <div className="py-1">
                  <KV label="设备 SN">{file.deviceSn || '—'}</KV>
                  <KV label="设备名称">{file.deviceName || '—'}</KV>
                  <KV label="上传人">{file.uploader || '—'}</KV>
                  <KV label="描述">{file.description || '—'}</KV>
                  <KV label="下载地址">{file.downloadUrl || '—'}</KV>
                </div>
                {file.tags.length > 0 ? (
                  <div className="flex flex-wrap gap-1.5 px-3 pb-3 pt-1">
                    {file.tags.map((tg) => (
                      <span key={tg} className="chip" style={{ color: NEON.cyan }}>#{tg}</span>
                    ))}
                  </div>
                ) : null}
              </GlassPanel>
            </div>
          </div>
        ) : null}
      </StateGate>
    </PageShell>
  )
}
