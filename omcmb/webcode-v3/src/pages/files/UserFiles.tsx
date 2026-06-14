import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  RefreshCcw,
  Download,
  Trash2,
  FolderUp,
  Database,
  HardDrive,
  User,
  ChevronRight,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatBytes, formatTime } from '@/lib/format'
import {
  useFileList,
  useStorageStats,
  useDownloadFile,
  useDeleteFiles,
} from '@core/hooks/api/useFiles'
import type {
  ManagedFile,
  FileType,
  FileStatus,
} from '@core/mock/data/fileManagement'

import { ChipFilter, OverviewStat, Pager, Row, StateGate, NEON } from './_shared'

// 用户文件 = 文件管理库的用户上传资产视图，真实接口 GET /files
const PAGE_SIZE = 20

const TYPE_ORDER: Array<FileType | ''> = ['', 'config', 'firmware', 'backup', 'report', 'certificate']
const TYPE_LABEL: Record<FileType, string> = {
  config: '配置',
  log: '日志',
  firmware: '固件',
  backup: '备份',
  report: '报表',
  certificate: '证书',
}
const TYPE_COLOR: Record<FileType, string> = {
  config: NEON.cyan,
  log: NEON.amber,
  firmware: NEON.violet,
  backup: NEON.green,
  report: NEON.blue,
  certificate: NEON.gold,
}

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

export default function UserFiles() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [fileType, setFileType] = useState<FileType | ''>('')
  const [keyword, setKeyword] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(fileType ? { fileType } : {}),
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, fileType, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useFileList(params)
  const { data: storage } = useStorageStats()
  const download = useDownloadFile()
  const remove = useDeleteFiles()

  const rows: ManagedFile[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const pageBytes = rows.reduce((s, f) => s + (f.fileSize || 0), 0)
  const uploaderCount = new Set(rows.map((f) => f.uploader).filter(Boolean)).size

  const toggleOne = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  const handleDelete = (ids: string[]) => {
    if (ids.length === 0) return
    remove.mutate(ids, {
      onSuccess: () => {
        setSelected((prev) => {
          const next = new Set(prev)
          for (const id of ids) next.delete(id)
          return next
        })
        void refetch()
      },
    })
  }

  return (
    <PageShell
      code="F06"
      title="USER FILES · 用户文件"
      subtitle="USER ASSET VAULT · MinIO"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="文件名 / 设备 SN"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          {selected.size > 0 ? (
            <NeonButton tone="danger" icon={<Trash2 />} disabled={remove.isPending} onClick={() => handleDelete([...selected])}>
              删除选中 ({selected.size})
            </NeonButton>
          ) : null}
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 lg:grid-cols-4">
        <OverviewStat icon={<Database className="size-4" />} label="OBJECTS" color={NEON.cyan} value={(storage?.fileCount ?? total).toLocaleString()} hint="对象总数" />
        <OverviewStat icon={<HardDrive className="size-4" />} label="PAGE SIZE" color={NEON.green} value={formatBytes(pageBytes)} hint="本页占用" />
        <OverviewStat icon={<FolderUp className="size-4" />} label="MATCHED" color={NEON.violet} value={total.toLocaleString()} hint="当前筛选命中" />
        <OverviewStat icon={<User className="size-4" />} label="UPLOADERS" color={NEON.gold} value={uploaderCount} hint="本页上传人" />
      </div>

      <GlassPanel strong title="USER FILE MANIFEST" meta={`${total} OBJECTS`}>
        <div className="p-3">
          <div className="mb-3 flex flex-wrap items-center gap-2">
            <ChipFilter<FileType | ''>
              value={fileType}
              onChange={(v) => {
                setFileType(v)
                setPage(1)
              }}
              options={TYPE_ORDER.map((t) => ({
                value: t,
                label: t ? TYPE_LABEL[t] : 'ALL',
                color: t ? TYPE_COLOR[t] : NEON.cyan,
              }))}
            />
          </div>

          <div className="grid grid-cols-[28px_2.4fr_1fr_1fr_1.4fr_1fr_120px] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            <span />
            <span>NAME</span>
            <span>TYPE / SIZE</span>
            <span>STATUS</span>
            <span>DEVICE</span>
            <span>UPLOAD</span>
            <span className="text-right">ACTIONS</span>
          </div>
          <div className="mt-1.5 space-y-1.5">
            <StateGate
              isLoading={isLoading}
              isError={isError}
              error={error}
              isEmpty={rows.length === 0}
              loadingLabel="SYNCING…"
              emptyLabel="NO USER FILES · 暂无用户文件"
            >
              {rows.map((f) => (
                <Row
                  key={f.id}
                  color={TYPE_COLOR[f.fileType]}
                  className="grid-cols-[28px_2.4fr_1fr_1fr_1.4fr_1fr_120px]"
                >
                  <input
                    type="checkbox"
                    checked={selected.has(f.id)}
                    onChange={() => toggleOne(f.id)}
                    className="accent-cyan-400"
                    aria-label={`select ${f.fileName}`}
                  />
                  <div className="min-w-0">
                    <div className="truncate font-display text-sm font-bold text-cyan-100" title={f.fileName}>{f.fileName}</div>
                    {f.tags.length > 0 ? (
                      <div className="mt-0.5 flex flex-wrap gap-1">
                        {f.tags.slice(0, 4).map((tg) => (
                          <span key={tg} className="font-mono text-[9px] text-cyan-300/45">#{tg}</span>
                        ))}
                      </div>
                    ) : f.description ? (
                      <div className="truncate font-mono text-[10px] text-cyan-300/45">{f.description}</div>
                    ) : null}
                  </div>
                  <div>
                    <span className="chip" style={{ color: TYPE_COLOR[f.fileType] }}>{TYPE_LABEL[f.fileType]}</span>
                    <div className="mt-0.5 font-mono text-[10px] text-cyan-300/55">{formatBytes(f.fileSize)}</div>
                  </div>
                  <div>
                    <StatusBadge status={STATUS_TOKEN[f.status]} label={STATUS_LABEL[f.status]} />
                  </div>
                  <div className="min-w-0">
                    <div className="truncate text-xs text-cyan-100/85">{f.deviceName || (f.deviceSn ? '—' : '系统资产')}</div>
                    <div className="font-mono text-[10px] text-cyan-300/55">{f.deviceSn || '—'}</div>
                  </div>
                  <div>
                    <div className="font-mono text-[11px] text-cyan-300/75">{formatTime(f.uploadTime)}</div>
                    <div className="font-mono text-[10px] text-cyan-300/45">{f.uploader || '—'}</div>
                  </div>
                  <div className="flex justify-end gap-1.5">
                    <NeonButton icon={<Download />} disabled={f.status !== 'available' || download.isPending} onClick={() => download.mutate(f.id)}>
                      DL
                    </NeonButton>
                    <NeonButton tone="danger" icon={<Trash2 />} disabled={remove.isPending} onClick={() => handleDelete([f.id])}>
                      DEL
                    </NeonButton>
                    <NeonButton icon={<ChevronRight />} onClick={() => navigate(`/files/perf-retrieval/${f.id}`)}>
                      详情
                    </NeonButton>
                  </div>
                </Row>
              ))}
            </StateGate>
          </div>
          <Pager
            page={page}
            totalPages={totalPages}
            total={total}
            pageSize={PAGE_SIZE}
            onPrev={() => setPage((p) => Math.max(1, p - 1))}
            onNext={() => setPage((p) => Math.min(totalPages, p + 1))}
          />
        </div>
      </GlassPanel>
    </PageShell>
  )
}
