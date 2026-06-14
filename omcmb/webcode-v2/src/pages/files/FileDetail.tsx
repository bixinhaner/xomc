import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Download, Loader2, Trash2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  PageShell,
  formatBytes,
  formatTime,
} from '@/components/layout/PageShell'

import {
  useDeleteFiles,
  useDownloadFile,
  useFileById,
} from '@core/hooks/api/useFiles'
import type { FileStatus, FileType } from '@core/mock/data/fileManagement'

// ============================================================
// 文件详情 —— /files/detail/:id
//   useParams 取文件 ID，useFileById 加载真实文件元数据
//   展示文件名/类型/状态/大小/校验和/设备/标签等；支持下载、删除
//   数据走真实 @core hook，三态完整。
// ============================================================

const FILE_TYPE_LABEL: Record<FileType, string> = {
  config: '配置',
  log: '日志',
  firmware: '固件',
  backup: '备份',
  report: '报表',
  certificate: '证书',
}

const FILE_STATUS_LABEL: Record<FileStatus, string> = {
  available: '可用',
  uploading: '上传中',
  processing: '处理中',
  expired: '已过期',
  deleted: '已删除',
}

function fileStatusVariant(s: FileStatus): 'success' | 'warning' | 'muted' {
  if (s === 'available') return 'success'
  if (s === 'expired' || s === 'deleted') return 'muted'
  return 'warning'
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1 border-b py-2.5 last:border-b-0 sm:flex-row sm:items-center sm:gap-4">
      <div className="w-32 shrink-0 text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className="text-sm">{children}</div>
    </div>
  )
}

export default function FileDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data, isLoading, isError, error } = useFileById(id)
  const download = useDownloadFile()
  const remove = useDeleteFiles()

  return (
    <PageShell
      title="文件详情"
      description={id ? `文件 ID：${id}` : undefined}
      toolbar={
        <Button variant="outline" size="sm" onClick={() => navigate('/files/user-files')}>
          <ArrowLeft /> 返回文件库
        </Button>
      }
    >
      {isLoading ? (
        <div className="flex h-48 items-center justify-center text-muted-foreground">
          <Loader2 className="mr-2 size-5 animate-spin" /> 加载中
        </div>
      ) : isError ? (
        <div className="flex h-48 items-center justify-center text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </div>
      ) : !data ? (
        <div className="flex h-48 items-center justify-center text-muted-foreground">
          未找到该文件
        </div>
      ) : (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between gap-4">
            <CardTitle className="break-all">{data.fileName}</CardTitle>
            <div className="flex shrink-0 items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={download.isPending}
                onClick={() => download.mutate(data.id)}
              >
                <Download /> 下载
              </Button>
              <Button
                variant="outline"
                size="sm"
                className="text-destructive hover:text-destructive"
                disabled={remove.isPending}
                onClick={() => {
                  if (window.confirm(`确认删除文件「${data.fileName}」？`)) {
                    remove.mutate([data.id], {
                      onSuccess: () => navigate('/files/user-files'),
                    })
                  }
                }}
              >
                <Trash2 /> 删除
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <Field label="类型">
              <Badge variant="outline">{FILE_TYPE_LABEL[data.fileType] ?? data.fileType}</Badge>
            </Field>
            <Field label="状态">
              <Badge variant={fileStatusVariant(data.status)}>
                {FILE_STATUS_LABEL[data.status] ?? data.status}
              </Badge>
            </Field>
            <Field label="大小">{formatBytes(data.fileSize)}</Field>
            <Field label="MIME 类型">
              <span className="font-mono text-xs">{data.mimeType || '—'}</span>
            </Field>
            <Field label="关联设备">{data.deviceName || data.deviceSn || '—'}</Field>
            <Field label="上传者">{data.uploader || '—'}</Field>
            <Field label="上传时间">{formatTime(data.uploadTime)}</Field>
            <Field label="过期时间">{data.expiryTime ? formatTime(data.expiryTime) : '—'}</Field>
            <Field label="校验和">
              <span className="break-all font-mono text-xs text-muted-foreground">
                {data.checksum || '—'}
              </span>
            </Field>
            <Field label="描述">{data.description || '—'}</Field>
            <Field label="标签">
              {data.tags.length > 0 ? (
                <div className="flex flex-wrap gap-1">
                  {data.tags.map((tag) => (
                    <Badge key={tag} variant="secondary">
                      {tag}
                    </Badge>
                  ))}
                </div>
              ) : (
                '—'
              )}
            </Field>
          </CardContent>
        </Card>
      )}
    </PageShell>
  )
}
