import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  RefreshCcw,
  Loader2,
  HardDriveDownload,
  Download,
  FolderOpen,
  ChevronRight,
  Boxes,
  Archive,
  KeyRound,
  Radio,
  Cpu,
  FileText,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import {
  useUnifiedFileTransferDevices,
  useUnifiedFileTransferTaskTypes,
} from '@core/hooks/api/useUnifiedFileTransfer'
import type {
  UnifiedFileTransferDeviceItem,
  UnifiedFileTransferDeviceStatus,
  UnifiedFileTransferTaskType,
} from '@core/types/unifiedFileTransfer'

const PAGE_SIZE = 30

const DEV_STATUS_BADGE: Record<UnifiedFileTransferDeviceStatus, { status: string; label: string }> = {
  pending: { status: 'unknown', label: '待执行' },
  downloading: { status: 'active', label: '下载中' },
  uploading: { status: 'active', label: '上传中' },
  awaiting_tc: { status: 'warning', label: '等待完成' },
  verifying: { status: 'warning', label: '校验中' },
  suspended: { status: 'warning', label: '已挂起' },
  ended: { status: 'ok', label: '已完成' },
  failed: { status: 'critical', label: '失败' },
}

// 文件库快捷入口 —— 仅链接到本皮肤实际存在的 v3 路由（router/index.tsx 已注册）。
const LIBRARY_LINKS: { label: string; desc: string; path: string; icon: React.ReactNode; color: string }[] = [
  { label: '版本固件库', desc: 'SOFTWARE · 升级镜像 / 补丁 / FPGA', path: '/software/version', icon: <Boxes className="size-4" />, color: '#00f0ff' },
  { label: '配置备份库', desc: 'BACKUP · 配置快照 / License 备份', path: '/backup/tasks', icon: <Archive className="size-4" />, color: '#00ff88' },
  { label: '设备 License', desc: 'LICENSE · 授权文件管理', path: '/license', icon: <KeyRound className="size-4" />, color: '#a855f7' },
  { label: 'MR 测量文件', desc: 'MR · MRO / MRS / MRE', path: '/mr/files', icon: <Radio className="size-4" />, color: '#5b9eff' },
  { label: '文件总览', desc: 'FILES · 统一文件浏览', path: '/file/config-retrieval', icon: <FileText className="size-4" />, color: '#ffaa00' },
]

export default function TransferFileManagementPage() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [category, setCategory] = useState<string>('')

  const { data: taskTypes } = useUnifiedFileTransferTaskTypes()
  const categories = useMemo(() => {
    const map = new Map<string, string>()
    ;(taskTypes ?? []).forEach((t: UnifiedFileTransferTaskType) => {
      if (!map.has(t.category)) map.set(t.category, t.categoryLabel)
    })
    return Array.from(map.entries()).map(([value, label]) => ({ value, label }))
  }, [taskTypes])

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(category ? { category } : {}),
    }),
    [page, keyword, category],
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useUnifiedFileTransferDevices(params)
  const rows = useMemo<UnifiedFileTransferDeviceItem[]>(() => data?.items ?? [], [data])
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  // 仅展示已产出文件（OUTPUT 类：备份 / 日志采集 / 配置）的记录优先；统计可下载数。
  const downloadableCount = useMemo(() => rows.filter((r) => !!r.downloadUrl).length, [rows])

  return (
    <PageShell
      code="F06"
      title="FILE LIBRARY · 文件管理"
      subtitle="TRANSFERRED ARTIFACTS · DEVICE FILE RECORDS"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="设备 SN / 名称 / 文件"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {/* 文件库快捷入口 */}
      <div className="mb-3 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        {LIBRARY_LINKS.map((lib) => (
          <button
            key={lib.path}
            type="button"
            onClick={() => navigate(lib.path)}
            className="glass group relative flex flex-col gap-2 overflow-hidden rounded-sm border-l-2 px-3 py-3 text-left transition-all hover:bg-cyan-500/5"
            style={{ borderLeftColor: lib.color }}
          >
            <div className="flex items-center justify-between">
              <span style={{ color: lib.color }}>{lib.icon}</span>
              <ChevronRight className="size-3.5 text-cyan-300/40 transition-transform group-hover:translate-x-0.5 group-hover:text-cyan-200" />
            </div>
            <div>
              <div className="font-display text-sm font-bold text-cyan-100">{lib.label}</div>
              <div className="font-mono text-[10px] uppercase tracking-[0.12em] text-cyan-300/50">
                {lib.desc}
              </div>
            </div>
          </button>
        ))}
      </div>

      {/* 分类筛选 + 统计 */}
      <GlassPanel
        title="TRANSFER RECORDS · 传输文件记录"
        meta={`${total} RECORDS · ${downloadableCount} DOWNLOADABLE (THIS PAGE)`}
        className="mb-3"
      >
        <div className="flex flex-wrap gap-2 p-3.5">
          <button
            type="button"
            onClick={() => {
              setCategory('')
              setPage(1)
            }}
            className={`chip transition-all ${
              category === '' ? 'text-cyan-200 shadow-[0_0_10px_currentColor]' : 'text-cyan-300/45 hover:text-cyan-300/80'
            }`}
          >
            ALL · 全部
          </button>
          {categories.map((c) => (
            <button
              key={c.value}
              type="button"
              onClick={() => {
                setCategory(c.value)
                setPage(1)
              }}
              className={`chip transition-all ${
                category === c.value
                  ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                  : 'text-cyan-300/45 hover:text-cyan-300/80'
              }`}
            >
              {c.label}
            </button>
          ))}
        </div>
      </GlassPanel>

      {/* 记录列表 */}
      {rows.length > 0 && (
        <div className="mb-1 grid grid-cols-[1.6fr_1.4fr_1fr_120px_1.6fr_140px] items-center gap-3 px-3 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
          <span>DEVICE · 设备</span>
          <span>TASK · 任务</span>
          <span>TYPE · 类型</span>
          <span>STATUS</span>
          <span>FILE · 文件</span>
          <span>START / END · 开始 / 结束</span>
        </div>
      )}

      <div className="space-y-1.5">
        {isLoading ? (
          <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
            <Loader2 className="size-4 animate-spin" />
            <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
          </div>
        ) : isError ? (
          <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
            FAILURE · {error instanceof Error ? error.message : '未知错误'}
          </div>
        ) : rows.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-3 py-16">
            <FolderOpen className="size-10 text-cyan-400/50" />
            <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
              NO FILE RECORDS · 暂无传输文件记录
            </div>
            <span className="font-mono text-[11px] text-cyan-300/40">
              可前往上方文件库查看已上传 / 已生成的文件
            </span>
          </div>
        ) : (
          rows.map((d) => {
            const badge = DEV_STATUS_BADGE[d.status]
            return (
              <div
                key={d.id}
                className="fleet-row grid grid-cols-[1.6fr_1.4fr_1fr_120px_1.6fr_140px] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: d.status === 'failed' ? '#ff2d6f' : '#00f0ff' }}
              >
                <button
                  type="button"
                  className="min-w-0 text-left"
                  onClick={() => navigate(`/transfer/center/${encodeURIComponent(d.taskId)}`)}
                  title="查看所属任务"
                >
                  <div className="truncate text-xs text-cyan-100/90">{d.deviceName || '—'}</div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">
                    <Cpu className="mr-1 inline size-3" />
                    {d.deviceSn}
                  </div>
                </button>
                <div className="min-w-0">
                  <div className="truncate text-xs text-cyan-100/85">{d.taskName}</div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">{d.categoryLabel}</div>
                </div>
                <div className="min-w-0 truncate text-xs text-cyan-100/80">{d.typeDisplayName}</div>
                <div>
                  <StatusBadge status={badge.status} label={badge.label} className="scale-90" />
                </div>
                <div className="min-w-0">
                  {d.targetFile ? (
                    d.downloadUrl ? (
                      <a
                        href={d.downloadUrl}
                        target="_blank"
                        rel="noreferrer"
                        className="flex items-center gap-1 truncate font-mono text-[11px] text-cyan-300 hover:text-cyan-100"
                        title={d.targetFile}
                      >
                        <Download className="size-3 shrink-0" />
                        <span className="truncate">{d.targetFile}</span>
                      </a>
                    ) : (
                      <span
                        className="flex items-center gap-1 truncate font-mono text-[11px] text-cyan-300/45"
                        title={`${d.targetFile} · 文件未就绪`}
                      >
                        <HardDriveDownload className="size-3 shrink-0" />
                        <span className="truncate">{d.targetFile}</span>
                      </span>
                    )
                  ) : (
                    <span className="text-cyan-300/40">—</span>
                  )}
                </div>
                {/* issue #655: 「最近上报」单值改为「开始 / 结束」双行，未达对应状态显示 '—'。 */}
                <div className="flex flex-col gap-0.5 font-mono text-[11px] leading-tight text-cyan-300/75">
                  <span title="开始时间">▶ {d.startedAt ? formatTime(d.startedAt) : '—'}</span>
                  <span title="结束时间" className="text-cyan-300/55">■ {d.endedAt ? formatTime(d.endedAt) : '—'}</span>
                </div>
              </div>
            )
          })
        )}
      </div>

      {/* 分页 */}
      <div className="mt-4 flex items-center justify-between">
        <span className="font-mono text-[11px] text-cyan-300/55">
          PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
        </span>
        <div className="flex gap-2">
          <NeonButton onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
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
    </PageShell>
  )
}
