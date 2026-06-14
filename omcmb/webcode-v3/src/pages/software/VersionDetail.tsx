import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Download, Star } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatTime } from '@/lib/format'
import { useSoftwareVersionById, useDownloadFirmware } from '@core/hooks/api/useSoftware'

import { NEON, VERSION_STATUS, KV, TagGroup, Syncing, ErrorBlock, EmptyBlock, formatFileSize } from './_shared'

// ---------------------------------------------------------------------------
// 版本详情 · software/version/:id
// 由 useParams 取 id，useSoftwareVersionById 加载真实版本
// ---------------------------------------------------------------------------

export default function VersionDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: v, isLoading, isError, error } = useSoftwareVersionById(id)
  const download = useDownloadFirmware()

  return (
    <PageShell
      code="F06"
      title="FIRMWARE DETAIL · 版本详情"
      subtitle={`VERSION ${id || '—'}`}
      bare
      toolbar={
        <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/software/version')}>
          返回列表
        </NeonButton>
      }
    >
      {isLoading ? (
        <Syncing label="LOADING VERSION…" />
      ) : isError ? (
        <ErrorBlock msg={error instanceof Error ? error.message : '未知错误'} />
      ) : !v ? (
        <EmptyBlock label="VERSION NOT FOUND · 版本不存在" />
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
          <GlassPanel strong title="META · 元数据" className="lg:col-span-1">
            <div className="space-y-4 p-4">
              <div>
                <div className="flex items-center gap-2">
                  <div className="font-mono text-base font-bold text-cyan-100 text-glow">{v.versionCode}</div>
                  {v.recommend ? (
                    <Star className="size-4" style={{ color: NEON.gold, fill: NEON.gold, filter: `drop-shadow(0 0 5px ${NEON.gold})` }} />
                  ) : null}
                </div>
                <div className="font-mono text-[11px] text-cyan-300/55">{v.versionName}</div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <KV label="状态">
                  <span className="chip" style={{ color: (VERSION_STATUS[v.status] ?? { color: NEON.dim }).color }}>
                    {(VERSION_STATUS[v.status] ?? { label: v.status }).label}
                  </span>
                </KV>
                <KV label="推荐">
                  <span style={{ color: v.recommend ? NEON.gold : NEON.dim }}>{v.recommend ? '是' : '否'}</span>
                </KV>
                <KV label="制式">{v.deviceType || '—'}</KV>
                <KV label="厂商">{v.vendor || v.manufacturer || '—'}</KV>
                <KV label="文件大小">{formatFileSize(v.fileSize)}</KV>
                <KV label="发布日期">{v.releaseDate ? formatTime(v.releaseDate) : '—'}</KV>
                <KV label="最低硬件">{v.minHardwareVersion || '—'}</KV>
                <KV label="上传者">{v.uploader || '—'}</KV>
              </div>

              <NeonButton
                className="w-full justify-center"
                icon={<Download />}
                disabled={download.isPending || !v.fileName}
                onClick={() => download.mutate({ id: v.id, fileName: v.fileName })}
              >
                {download.isPending ? '下载中…' : '下载固件'}
              </NeonButton>
            </div>
          </GlassPanel>

          <GlassPanel strong title="DETAIL · 详细信息" className="lg:col-span-2">
            <div className="space-y-4 p-4">
              <KV label="文件名">
                <span className="break-all font-mono text-[11px] text-cyan-100/80">{v.fileName || '—'}</span>
              </KV>
              <KV label="校验和 · MD5">
                <span className="break-all font-mono text-[11px] text-cyan-100/80">{v.checksum || '—'}</span>
              </KV>

              {v.releaseNotes ? (
                <div>
                  <div className="mb-1 font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/55">RELEASE NOTES · 发布说明</div>
                  <div className="rounded-sm border border-cyan-500/15 bg-cyan-500/[0.03] p-2 text-[12px] leading-relaxed text-cyan-100/80">
                    {v.releaseNotes}
                  </div>
                </div>
              ) : null}

              {v.features?.length ? <TagGroup title="FEATURES · 特性" color={NEON.blue} items={v.features} /> : null}
              {v.bugFixes?.length ? <TagGroup title="BUGFIXES · 修复" color={NEON.green} items={v.bugFixes} /> : null}
              {v.known_issues?.length ? <TagGroup title="KNOWN ISSUES · 已知问题" color={NEON.amber} items={v.known_issues} /> : null}
            </div>
          </GlassPanel>
        </div>
      )}
    </PageShell>
  )
}
