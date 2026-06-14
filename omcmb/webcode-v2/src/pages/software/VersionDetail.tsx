import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Download, Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { PageShell, formatBytes, formatTime } from '@/components/layout/PageShell'

import { useSoftwareVersionById } from '@core/hooks/api/useSoftware'
import type { SoftwareVersion } from '@core/mock/data/software'

import { FILE_TYPE_LABEL, VERSION_STATUS } from './_shared'
import { InfoRow } from './_components'

// ===========================================================================
// 版本详情 — 对照 v1 VersionQuery 的详情 Modal（Descriptions）
// 按 id 经 useSoftwareVersionById 加载真实单条版本数据
// ===========================================================================

function TagList({ items, variant }: { items?: string[]; variant: 'success' | 'warning' | 'default' }) {
  if (!items || items.length === 0) return <span className="text-sm text-muted-foreground">—</span>
  return (
    <div className="flex flex-wrap gap-1.5">
      {items.map((s, i) => (
        <Badge key={`${s}-${i}`} variant={variant}>
          {s}
        </Badge>
      ))}
    </div>
  )
}

export default function VersionDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data: version, isLoading, isError, error, isFetching } = useSoftwareVersionById(id)

  return (
    <PageShell
      title={version ? version.versionName || version.versionCode : '版本详情'}
      description={version ? version.versionCode : undefined}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => navigate('/software/version')}>
            <ArrowLeft className="size-4" /> 返回版本列表
          </Button>
          {version?.downloadUrl && (
            <Button
              variant="outline"
              size="sm"
              className="ml-auto"
              onClick={() => window.open(version.downloadUrl, '_blank')}
            >
              <Download className="size-4" /> 下载固件
            </Button>
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
      ) : !version ? (
        <Card className="flex h-48 flex-col items-center justify-center gap-2 p-6 text-muted-foreground">
          <span className="text-sm">未找到版本 {id}</span>
          <Button variant="outline" size="sm" onClick={() => navigate('/software/version')}>
            返回版本列表
          </Button>
        </Card>
      ) : (
        <Detail version={version} />
      )}
    </PageShell>
  )
}

function Detail({ version: v }: { version: SoftwareVersion }) {
  const st = VERSION_STATUS[v.status]
  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader className="p-4 pb-2">
          <CardTitle className="text-base font-medium">基本信息</CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-1 gap-x-8 p-4 pt-2 sm:grid-cols-2">
          <InfoRow label="版本号" value={v.versionCode} mono />
          <InfoRow label="名称" value={v.versionName} />
          <InfoRow label="设备型号" value={v.deviceType} />
          <InfoRow label="厂商" value={v.vendor || v.manufacturer} />
          <InfoRow label="类型" value={v.fileType != null ? FILE_TYPE_LABEL[v.fileType] ?? '—' : '—'} />
          <InfoRow label="状态" value={<Badge variant={st.variant}>{st.label}</Badge>} />
          <InfoRow label="推荐" value={v.recommend ? '是' : '否'} />
          <InfoRow label="文件大小" value={formatBytes(v.fileSize)} />
          <InfoRow label="文件名" value={v.fileName} mono />
          <InfoRow label="最低硬件版本" value={v.minHardwareVersion} />
          <InfoRow label="上传者" value={v.uploader} />
          <InfoRow label="发布时间" value={formatTime(v.releaseDate)} />
          <InfoRow label="校验和" value={v.checksum} mono />
          <InfoRow label="下载地址" value={v.downloadUrl} mono />
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="p-4 pb-2">
          <CardTitle className="text-base font-medium">发布说明</CardTitle>
        </CardHeader>
        <CardContent className="p-4 pt-2 text-sm">
          {v.releaseNotes || '—'}
          {v.description && v.description !== v.releaseNotes && (
            <p className="mt-2 text-muted-foreground">{v.description}</p>
          )}
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <Card>
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-base font-medium">新增特性</CardTitle>
          </CardHeader>
          <CardContent className="p-4 pt-2">
            <TagList items={v.features} variant="default" />
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-base font-medium">问题修复</CardTitle>
          </CardHeader>
          <CardContent className="p-4 pt-2">
            <TagList items={v.bugFixes} variant="success" />
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-base font-medium">已知问题</CardTitle>
          </CardHeader>
          <CardContent className="p-4 pt-2">
            <TagList items={v.known_issues} variant="warning" />
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
