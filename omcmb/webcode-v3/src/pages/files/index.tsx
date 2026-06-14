import { useMemo, useState, type ReactNode } from 'react'
import {
  Search,
  RefreshCcw,
  Loader2,
  Database,
  Download,
  Trash2,
  FileWarning,
  HardDrive,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { Sparkline } from '@/components/viz/Sparkline'
import { formatBytes, formatTime } from '@/lib/format'
import {
  useFileList,
  useStorageStats,
  useDeleteFiles,
  useDownloadFile,
} from '@core/hooks/api/useFiles'
import { useUnifiedFileTransferOverview } from '@core/hooks/api/useUnifiedFileTransfer'
import type {
  ManagedFile,
  FileType,
  FileStatus,
} from '@core/mock/data/fileManagement'

// ---------------------------------------------------------------------------
// 文件类型 / 状态视觉映射
// ---------------------------------------------------------------------------

const TYPE_ORDER: FileType[] = [
  'config',
  'log',
  'firmware',
  'backup',
  'report',
  'certificate',
]

const TYPE_LABEL: Record<FileType, string> = {
  config: '配置',
  log: '日志',
  firmware: '固件',
  backup: '备份',
  report: '报表',
  certificate: '证书',
}

const TYPE_COLOR: Record<FileType, string> = {
  config: '#00f0ff',
  log: '#ff7a1a',
  firmware: '#b388ff',
  backup: '#00ff88',
  report: '#5b9eff',
  certificate: '#ffd400',
}

// FileStatus -> StatusBadge status token + 中文标签
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

const STATUS_FILTERS: Array<{ value: FileStatus | ''; label: string }> = [
  { value: '', label: 'ALL' },
  { value: 'available', label: '可用' },
  { value: 'processing', label: '处理中' },
  { value: 'expired', label: '已过期' },
]

// ---------------------------------------------------------------------------
// 页面
// ---------------------------------------------------------------------------

export function FilesPage() {
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [fileType, setFileType] = useState<FileType | ''>('')
  const [status, setStatus] = useState<FileStatus | ''>('')
  const [keyword, setKeyword] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(fileType ? { fileType } : {}),
      ...(status ? { status } : {}),
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, fileType, status, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useFileList(params)
  const { data: storage } = useStorageStats()
  const { data: overview } = useUnifiedFileTransferOverview()

  const deleteFiles = useDeleteFiles()
  const downloadFile = useDownloadFile()

  const rows: ManagedFile[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  // 当前页的类型分布（后端 storage stats 的 byType 当前为占位空对象，
  // 因此用真实文件列表派生分布，保证 UI 有数据）
  const typeDist = useMemo(() => {
    const acc: Record<FileType, { count: number; size: number }> = {
      config: { count: 0, size: 0 },
      log: { count: 0, size: 0 },
      firmware: { count: 0, size: 0 },
      backup: { count: 0, size: 0 },
      report: { count: 0, size: 0 },
      certificate: { count: 0, size: 0 },
    }
    for (const f of rows) {
      acc[f.fileType].count += 1
      acc[f.fileType].size += f.fileSize || 0
    }
    return acc
  }, [rows])

  const pageBytes = useMemo(
    () => rows.reduce((s, f) => s + (f.fileSize || 0), 0),
    [rows]
  )
  const maxTypeCount = Math.max(
    1,
    ...TYPE_ORDER.map((t) => typeDist[t].count)
  )
  // 文件大小火花线（按上传时间升序）
  const sizeSpark = useMemo(
    () =>
      [...rows]
        .sort(
          (a, b) =>
            new Date(a.uploadTime).getTime() -
            new Date(b.uploadTime).getTime()
        )
        .map((f) => Math.round((f.fileSize || 0) / 1024)),
    [rows]
  )

  const allSelected = rows.length > 0 && rows.every((r) => selected.has(r.id))
  const toggleAll = () => {
    setSelected((prev) => {
      if (rows.every((r) => prev.has(r.id))) return new Set()
      return new Set(rows.map((r) => r.id))
    })
  }
  const toggleOne = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const handleDownload = (f: ManagedFile) => {
    downloadFile.mutate(f.id)
  }
  const handleDelete = (ids: string[]) => {
    if (ids.length === 0) return
    deleteFiles.mutate(ids, {
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
      title="FILE VAULT · 档案库"
      subtitle="OBJECT STORAGE EXPLORER · MinIO"
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
          {STATUS_FILTERS.map((s) => (
            <button
              key={s.value || 'all'}
              type="button"
              onClick={() => {
                setStatus(s.value)
                setPage(1)
              }}
              className={`chip transition-all ${
                status === s.value
                  ? 'shadow-[0_0_10px_currentColor]'
                  : 'opacity-55 hover:opacity-100'
              }`}
              style={{ color: '#00f0ff' }}
            >
              {s.label}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {/* 概览统计 */}
      <div className="mb-3 grid grid-cols-2 gap-3 lg:grid-cols-5">
        <OverviewStat
          icon={<Database className="size-4" />}
          label="OBJECTS"
          color="#00f0ff"
          value={(storage?.fileCount ?? total).toLocaleString()}
          hint="对象总数"
        />
        <OverviewStat
          icon={<HardDrive className="size-4" />}
          label="PAGE SIZE"
          color="#00ff88"
          value={formatBytes(pageBytes)}
          hint="当前页占用"
        />
        <OverviewStat
          label="XFER RUNNING"
          color="#b388ff"
          value={String(overview?.runningTaskCount ?? 0)}
          hint="传输任务运行中"
        />
        <OverviewStat
          label="XFER TYPES"
          color="#5b9eff"
          value={String(overview?.enabledTypeCount ?? 0)}
          hint={`含自定义 ${overview?.customTypeCount ?? 0}`}
        />
        <OverviewStat
          label="SUCCESS 30D"
          color="#ffd400"
          value={`${overview ? Math.round(overview.successRate30d) : 0}%`}
          hint="近30天传输成功率"
        />
      </div>

      <div className="grid grid-cols-1 gap-3 xl:grid-cols-[1fr_300px]">
        {/* 文件列表 */}
        <GlassPanel
          strong
          title="OBJECT MANIFEST"
          meta={`${total} OBJECTS`}
        >
          <div className="p-3">
            {/* 类型快速过滤 */}
            <div className="mb-3 flex flex-wrap items-center gap-2">
              <button
                type="button"
                onClick={() => {
                  setFileType('')
                  setPage(1)
                }}
                className={`chip transition-all ${
                  fileType === ''
                    ? 'shadow-[0_0_10px_currentColor]'
                    : 'opacity-55 hover:opacity-100'
                }`}
                style={{ color: '#00f0ff' }}
              >
                ALL
              </button>
              {TYPE_ORDER.map((t) => (
                <button
                  key={t}
                  type="button"
                  onClick={() => {
                    setFileType(t)
                    setPage(1)
                  }}
                  className={`chip transition-all ${
                    fileType === t
                      ? 'shadow-[0_0_10px_currentColor]'
                      : 'opacity-55 hover:opacity-100'
                  }`}
                  style={{ color: TYPE_COLOR[t] }}
                >
                  {TYPE_LABEL[t]}
                </button>
              ))}
              {selected.size > 0 ? (
                <NeonButton
                  tone="danger"
                  icon={<Trash2 />}
                  className="ml-auto"
                  disabled={deleteFiles.isPending}
                  onClick={() => handleDelete([...selected])}
                >
                  删除选中 ({selected.size})
                </NeonButton>
              ) : null}
            </div>

            {/* 表头 */}
            <div className="grid grid-cols-[28px_2.4fr_1fr_0.8fr_1.4fr_1fr_120px] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
              <input
                type="checkbox"
                checked={allSelected}
                onChange={toggleAll}
                className="accent-cyan-400"
                aria-label="select all"
              />
              <span>NAME</span>
              <span>TYPE / SIZE</span>
              <span>STATUS</span>
              <span>DEVICE</span>
              <span>UPLOAD</span>
              <span className="text-right">ACTIONS</span>
            </div>

            {/* 行 */}
            <div className="mt-1.5 space-y-1.5">
              {isLoading ? (
                <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
                  <Loader2 className="size-4 animate-spin" />
                  <span className="font-mono text-xs uppercase tracking-[0.2em]">
                    SYNCING…
                  </span>
                </div>
              ) : isError ? (
                <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
                  FAILURE ·{' '}
                  {error instanceof Error ? error.message : '未知错误'}
                </div>
              ) : rows.length === 0 ? (
                <div className="flex flex-col items-center justify-center gap-3 py-16">
                  <FileWarning className="size-10 text-cyan-400/40" />
                  <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
                    NO OBJECTS · 暂无文件
                  </div>
                </div>
              ) : (
                rows.map((f) => (
                  <div
                    key={f.id}
                    className="fleet-row grid grid-cols-[28px_2.4fr_1fr_0.8fr_1.4fr_1fr_120px] items-center gap-3 rounded-sm px-3 py-2.5"
                    style={{
                      ['--row-color' as never]: TYPE_COLOR[f.fileType],
                    }}
                  >
                    <input
                      type="checkbox"
                      checked={selected.has(f.id)}
                      onChange={() => toggleOne(f.id)}
                      className="accent-cyan-400"
                      aria-label={`select ${f.fileName}`}
                    />
                    <div className="min-w-0">
                      <div
                        className="truncate font-display text-sm font-bold text-cyan-100"
                        title={f.fileName}
                      >
                        {f.fileName}
                      </div>
                      {f.tags.length > 0 ? (
                        <div className="mt-0.5 flex flex-wrap gap-1">
                          {f.tags.slice(0, 4).map((tg) => (
                            <span
                              key={tg}
                              className="font-mono text-[9px] text-cyan-300/45"
                            >
                              #{tg}
                            </span>
                          ))}
                        </div>
                      ) : f.description ? (
                        <div className="truncate font-mono text-[10px] text-cyan-300/45">
                          {f.description}
                        </div>
                      ) : null}
                    </div>
                    <div>
                      <span
                        className="chip"
                        style={{ color: TYPE_COLOR[f.fileType] }}
                      >
                        {TYPE_LABEL[f.fileType]}
                      </span>
                      <div className="mt-0.5 font-mono text-[10px] text-cyan-300/55">
                        {formatBytes(f.fileSize)}
                      </div>
                    </div>
                    <div>
                      <StatusBadge
                        status={STATUS_TOKEN[f.status]}
                        label={STATUS_LABEL[f.status]}
                      />
                    </div>
                    <div className="min-w-0">
                      <div className="truncate text-xs text-cyan-100/85">
                        {f.deviceName || (f.deviceSn ? '—' : '系统资产')}
                      </div>
                      <div className="font-mono text-[10px] text-cyan-300/55">
                        {f.deviceSn || '—'}
                      </div>
                    </div>
                    <div>
                      <div className="font-mono text-[11px] text-cyan-300/75">
                        {formatTime(f.uploadTime)}
                      </div>
                      <div className="font-mono text-[10px] text-cyan-300/45">
                        {f.uploader || '—'}
                      </div>
                    </div>
                    <div className="flex justify-end gap-1.5">
                      <NeonButton
                        icon={<Download />}
                        disabled={
                          f.status !== 'available' || downloadFile.isPending
                        }
                        onClick={() => handleDownload(f)}
                      >
                        DL
                      </NeonButton>
                      <NeonButton
                        tone="danger"
                        icon={<Trash2 />}
                        disabled={deleteFiles.isPending}
                        onClick={() => handleDelete([f.id])}
                      >
                        DEL
                      </NeonButton>
                    </div>
                  </div>
                ))
              )}
            </div>

            {/* 分页 */}
            <div className="mt-4 flex items-center justify-between">
              <span className="font-mono text-[11px] text-cyan-300/55">
                PAGE {page} / {totalPages} · {pageSize}/PAGE · TOTAL {total}
              </span>
              <div className="flex gap-2">
                <NeonButton
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                >
                  ◂ PREV
                </NeonButton>
                <NeonButton
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                >
                  NEXT ▸
                </NeonButton>
              </div>
            </div>
          </div>
        </GlassPanel>

        {/* 右侧：类型分布 + 趋势 */}
        <div className="flex flex-col gap-3">
          <GlassPanel title="TYPE DISTRIBUTION" meta="本页">
            <div className="space-y-2.5 p-3.5">
              {TYPE_ORDER.map((t) => {
                const d = typeDist[t]
                const pct = Math.round((d.count / maxTypeCount) * 100)
                return (
                  <button
                    key={t}
                    type="button"
                    onClick={() => {
                      setFileType((cur) => (cur === t ? '' : t))
                      setPage(1)
                    }}
                    className="block w-full text-left"
                  >
                    <div className="flex items-center justify-between font-mono text-[10px] uppercase tracking-[0.15em] text-cyan-300/65">
                      <span style={{ color: TYPE_COLOR[t] }}>
                        {TYPE_LABEL[t]}
                      </span>
                      <span>
                        {d.count} · {formatBytes(d.size)}
                      </span>
                    </div>
                    <div className="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-cyan-500/10">
                      <div
                        className="h-full rounded-full transition-all"
                        style={{
                          width: `${pct}%`,
                          background: TYPE_COLOR[t],
                          boxShadow: `0 0 8px ${TYPE_COLOR[t]}`,
                        }}
                      />
                    </div>
                  </button>
                )
              })}
            </div>
          </GlassPanel>

          <GlassPanel title="SIZE TREND" meta="KB / 上传序">
            <div className="flex items-center justify-center p-4">
              {sizeSpark.length > 1 ? (
                <Sparkline
                  data={sizeSpark}
                  width={250}
                  height={64}
                  color={TYPE_COLOR.config}
                />
              ) : (
                <span className="py-6 font-mono text-[11px] uppercase tracking-[0.18em] text-cyan-300/45">
                  INSUFFICIENT DATA
                </span>
              )}
            </div>
          </GlassPanel>

          <GlassPanel title="TRANSFER ENGINE" meta="UFTE">
            <div className="grid grid-cols-2 gap-3 p-3.5">
              <MiniKpi
                label="运行中"
                color="#b388ff"
                value={String(overview?.runningTaskCount ?? 0)}
              />
              <MiniKpi
                label="启用类型"
                color="#5b9eff"
                value={String(overview?.enabledTypeCount ?? 0)}
              />
              <MiniKpi
                label="自定义类型"
                color="#00f0ff"
                value={String(overview?.customTypeCount ?? 0)}
              />
              <MiniKpi
                label="成功率30d"
                color="#00ff88"
                value={`${overview ? Math.round(overview.successRate30d) : 0}%`}
              />
            </div>
          </GlassPanel>
        </div>
      </div>
    </PageShell>
  )
}

// ---------------------------------------------------------------------------
// 子组件
// ---------------------------------------------------------------------------

function OverviewStat({
  icon,
  label,
  color,
  value,
  hint,
}: {
  icon?: ReactNode
  label: string
  color: string
  value: string
  hint?: string
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3"
      style={{ borderLeftColor: color }}
    >
      <div className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
        {icon ? <span style={{ color }}>{icon}</span> : null}
        {label}
      </div>
      <div
        className="font-display text-2xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
      {hint ? (
        <div className="font-mono text-[10px] text-cyan-300/45">{hint}</div>
      ) : null}
    </div>
  )
}

function MiniKpi({
  label,
  color,
  value,
}: {
  label: string
  color: string
  value: string
}) {
  return (
    <div className="rounded-sm border border-cyan-500/12 bg-cyan-500/[0.03] px-3 py-2">
      <div className="font-mono text-[10px] uppercase tracking-[0.15em] text-cyan-300/55">
        {label}
      </div>
      <div
        className="font-display text-xl font-bold"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}
