import { useMemo, useState } from 'react'
import {
  Search,
  RefreshCcw,
  Loader2,
  Inbox,
  Download,
  Trash2,
  Check,
  FileBox,
  UploadCloud,
  HardDrive,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatBytes } from '@/lib/format'
import {
  useConfigSnapshots,
  useBatchDeleteConfigSnapshots,
} from '@core/hooks/api/useConfigSnapshot'
import { useProductList } from '@core/hooks/api/useProducts'
import {
  configSnapshotApi,
  type ConfigSnapshot,
  type SnapshotSource,
} from '@core/services/api/configSnapshotApi'
import { formatSystemTime } from '@core/utils/systemTime'

import { Modal } from './Modal'

const PAGE_SIZE = 20

const SOURCE_LABEL: Record<SnapshotSource, string> = {
  backup: '备份归档',
  manual_upload: '手动上传',
}
const SOURCE_COLOR: Record<SnapshotSource, string> = {
  backup: '#00f0ff',
  manual_upload: '#a855f7',
}

function getErrMsg(e: unknown): string {
  if (e instanceof Error) return e.message
  if (typeof e === 'object' && e && 'message' in e) {
    return String((e as { message: unknown }).message)
  }
  return '未知错误'
}

function formatTime(iso?: string): string {
  // #459 子单 D：保留后端系统时区钟面，不按浏览器本地二次转换。
  return formatSystemTime(iso, { placeholder: '—' })
}

// ---------------------------------------------------------------------------
// 配置快照库 · /backup/config-snapshots
// ---------------------------------------------------------------------------

export default function ConfigSnapshotLibraryPage() {
  const [page, setPage] = useState(1)
  const [serialNumber, setSerialNumber] = useState('')
  // #602：产品名称下拉过滤，传 product.id
  const [productId, setProductId] = useState('')
  const [sourceFilter, setSourceFilter] = useState<SnapshotSource | ''>('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(serialNumber.trim() ? { serialNumber: serialNumber.trim() } : {}),
      ...(productId ? { productId } : {}),
      ...(sourceFilter ? { source: sourceFilter } : {}),
    }),
    [page, serialNumber, productId, sourceFilter]
  )

  const query = useConfigSnapshots(params)
  const batchDelete = useBatchDeleteConfigSnapshots()
  const { data: productsData } = useProductList()
  const productNameOptions = useMemo(
    () =>
      (productsData?.items ?? [])
        .map((p) => ({ label: `${p.name} (${p.tech})`, value: p.id }))
        .sort((a, b) => a.label.localeCompare(b.label, 'zh-CN')),
    [productsData]
  )
  const productNameByPattern = useMemo(() => {
    const map = new Map<string, string>()
    ;(productsData?.items ?? []).forEach((p) => {
      ;(p.patterns ?? []).forEach((pat) => map.set(pat, p.name))
    })
    return map
  }, [productsData])

  const items = query.data?.items ?? []
  const total = query.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const stats = useMemo(() => {
    let bytes = 0
    let fromBackup = 0
    let fromUpload = 0
    for (const s of items) {
      bytes += s.fileSize ?? 0
      if (s.source === 'backup') fromBackup += 1
      else fromUpload += 1
    }
    return { bytes, fromBackup, fromUpload }
  }, [items])

  const [selected, setSelected] = useState<Set<string>>(new Set())
  const selectedSns = useMemo(() => [...selected], [selected])
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [actionMsg, setActionMsg] = useState<string | null>(null)
  const [downloading, setDownloading] = useState<string | null>(null)

  const toggle = (sn: string) =>
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(sn)) next.delete(sn)
      else next.add(sn)
      return next
    })

  const allChecked = items.length > 0 && items.every((s) => selected.has(s.serialNumber))
  const toggleAll = () =>
    setSelected((prev) => {
      if (items.every((s) => prev.has(s.serialNumber))) {
        const next = new Set(prev)
        items.forEach((s) => next.delete(s.serialNumber))
        return next
      }
      const next = new Set(prev)
      items.forEach((s) => next.add(s.serialNumber))
      return next
    })

  const onDownload = async (sn: string) => {
    setDownloading(sn)
    try {
      await configSnapshotApi.download(sn)
    } catch (e) {
      setActionMsg(`下载失败：${getErrMsg(e)}`)
    } finally {
      setDownloading(null)
    }
  }

  const onConfirmDelete = async () => {
    try {
      const r = await batchDelete.mutateAsync(selectedSns)
      setSelected(new Set())
      setConfirmDelete(false)
      setActionMsg(
        `删除完成 · 成功 ${r.succeeded.length} 个${
          r.failed.length ? ` · 失败 ${r.failed.length} 个` : ''
        }`
      )
    } catch (e) {
      setConfirmDelete(false)
      setActionMsg(`删除失败：${getErrMsg(e)}`)
    }
  }

  const sourceOptions: (SnapshotSource | '')[] = ['', 'backup', 'manual_upload']

  return (
    <PageShell
      code="F06"
      title="CONFIG SNAPSHOTS · 配置快照库"
      subtitle="ONE-PER-DEVICE LATEST CONFIG SNAPSHOT"
      isFetching={query.isFetching || undefined}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => query.refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      <div className="mb-3 grid grid-cols-3 gap-3">
        <MiniStat label="快照总数 · TOTAL" value={total.toLocaleString()} color="#00f0ff" icon={<FileBox className="size-4" />} />
        <MiniStat label="备份归档 · BACKUP" value={stats.fromBackup.toLocaleString()} color="#00ff88" icon={<HardDrive className="size-4" />} />
        <MiniStat label="本页占用 · SIZE" value={formatBytes(stats.bytes)} color="#a855f7" icon={<UploadCloud className="size-4" />} />
      </div>

      {/* 工具条 */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
          <input
            className="neon-input w-64 pl-9"
            placeholder="按设备 SN 精确查询"
            value={serialNumber}
            onChange={(e) => {
              setSerialNumber(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <select
          className="neon-input w-56"
          value={productId}
          onChange={(e) => { setProductId(e.target.value); setPage(1) }}
        >
          <option value="">产品名称 · 全部</option>
          {productNameOptions.map((opt) => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
        </select>
        {sourceOptions.map((s) => (
          <button
            key={s || 'all'}
            type="button"
            onClick={() => {
              setSourceFilter(s)
              setPage(1)
            }}
            className={`chip transition-all ${
              sourceFilter === s ? 'shadow-[0_0_10px_currentColor]' : 'opacity-60 hover:opacity-100'
            }`}
            style={{ color: s ? SOURCE_COLOR[s as SnapshotSource] : '#00f0ff' }}
          >
            {s ? SOURCE_LABEL[s as SnapshotSource] : 'ALL'}
          </button>
        ))}
      </div>

      {/* 批量操作条 */}
      {selectedSns.length > 0 && (
        <div className="mb-2 flex flex-wrap items-center gap-2 rounded-sm border border-cyan-500/25 bg-cyan-500/[0.05] px-3 py-2">
          <span className="font-mono text-[11px] text-cyan-200">
            已选 {selectedSns.length} 个快照
          </span>
          <NeonButton tone="danger" icon={<Trash2 />} onClick={() => setConfirmDelete(true)}>
            批量删除
          </NeonButton>
          <NeonButton onClick={() => setSelected(new Set())}>清空选择</NeonButton>
        </div>
      )}

      {query.isLoading ? (
        <CenterSync />
      ) : query.isError ? (
        <FailureBox error={query.error} />
      ) : items.length === 0 ? (
        <EmptyBox hint="NO SNAPSHOTS · 无配置快照" />
      ) : (
        <>
          {/* 表头 */}
          <div className="mb-1 grid grid-cols-[28px_1.4fr_1.6fr_1fr_0.8fr_1fr_150px] items-center gap-3 px-3 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
            <button
              type="button"
              onClick={toggleAll}
              className={`flex size-4 items-center justify-center rounded-sm border ${
                allChecked ? 'border-cyan-400 bg-cyan-400/20' : 'border-cyan-500/40'
              }`}
              aria-label="select all"
            >
              {allChecked && <Check className="size-3 text-cyan-200" />}
            </button>
            <span>SN / 网元</span>
            <span>FILE</span>
            <span>SIZE / EXT</span>
            <span>SOURCE</span>
            <span>UPDATED</span>
            <span className="text-right">OPS</span>
          </div>

          <div className="space-y-1.5">
            {items.map((s: ConfigSnapshot) => {
              const color = SOURCE_COLOR[s.source]
              const checked = selected.has(s.serialNumber)
              return (
                <div
                  key={s.serialNumber}
                  className="fleet-row grid grid-cols-[28px_1.4fr_1.6fr_1fr_0.8fr_1fr_150px] items-center gap-3 rounded-sm px-3 py-2.5"
                  style={{ ['--row-color' as never]: color }}
                >
                  <button
                    type="button"
                    onClick={() => toggle(s.serialNumber)}
                    className={`flex size-4 items-center justify-center rounded-sm border ${
                      checked ? 'border-cyan-400 bg-cyan-400/20' : 'border-cyan-500/40'
                    }`}
                    aria-label={`select ${s.serialNumber}`}
                  >
                    {checked && <Check className="size-3 text-cyan-200" />}
                  </button>
                  <div className="min-w-0">
                    <div className="truncate font-display text-sm font-bold text-cyan-100">
                      {s.serialNumber}
                    </div>
                    <div className="truncate font-mono text-[10px] text-cyan-300/55">
                      {s.enbName || '—'}
                      {(() => {
                        const raw = s.productType || ''
                        if (!raw) return ''
                        const name = productNameByPattern.get(raw) ?? raw
                        return ` · ${name}`
                      })()}
                    </div>
                  </div>
                  <div className="min-w-0 font-mono text-[11px] text-cyan-200/85">
                    <div className="truncate">{s.fileName}</div>
                    <div className="truncate text-[10px] text-cyan-300/45">
                      {s.objectBucket}/{s.objectPath}
                    </div>
                  </div>
                  <div className="font-mono text-[11px] text-cyan-300/75">
                    <div>{formatBytes(s.fileSize)}</div>
                    <div className="text-[10px] text-cyan-300/45">{(s.fileExt || '').toUpperCase()}</div>
                  </div>
                  <div>
                    <span className="chip" style={{ color }}>
                      {SOURCE_LABEL[s.source]}
                    </span>
                  </div>
                  <div className="font-mono text-[10px] text-cyan-300/65">
                    <div>{formatTime(s.updateTime)}</div>
                    <div className="text-cyan-300/45">{s.updateBy || '—'}</div>
                  </div>
                  <div className="flex justify-end gap-1.5">
                    <NeonButton
                      icon={
                        downloading === s.serialNumber ? (
                          <Loader2 className="animate-spin" />
                        ) : (
                          <Download />
                        )
                      }
                      disabled={downloading === s.serialNumber}
                      onClick={() => {
                        void onDownload(s.serialNumber)
                      }}
                    >
                      下载
                    </NeonButton>
                  </div>
                </div>
              )
            })}
          </div>
          <Pager page={page} totalPages={totalPages} total={total} onPage={setPage} />
        </>
      )}

      <Modal
        open={confirmDelete}
        title="批量删除配置快照"
        subtitle="PURGE SNAPSHOTS"
        onClose={() => setConfirmDelete(false)}
        width={460}
        footer={
          <>
            <NeonButton onClick={() => setConfirmDelete(false)}>取消</NeonButton>
            <NeonButton
              tone="danger"
              disabled={batchDelete.isPending}
              onClick={() => {
                void onConfirmDelete()
              }}
            >
              {batchDelete.isPending ? '处理中…' : '确认删除'}
            </NeonButton>
          </>
        }
      >
        <p className="text-sm text-cyan-100/85">
          将永久删除选中的 {selectedSns.length} 个配置快照（含对象存储文件），不可恢复。
        </p>
        <p className="mt-2 font-mono text-xs text-cyan-300/70">
          {selectedSns.slice(0, 8).join(' · ')}
          {selectedSns.length > 8 ? ` … 等 ${selectedSns.length} 个` : ''}
        </p>
      </Modal>

      <Modal
        open={Boolean(actionMsg)}
        title="操作结果"
        subtitle="RESULT"
        onClose={() => setActionMsg(null)}
        width={420}
        footer={<NeonButton onClick={() => setActionMsg(null)}>知道了</NeonButton>}
      >
        <p className="text-sm text-cyan-100/85">{actionMsg}</p>
      </Modal>
    </PageShell>
  )
}

// ---------------------------------------------------------------------------
// 局部组件
// ---------------------------------------------------------------------------

function MiniStat({
  label,
  value,
  color,
  icon,
}: {
  label: string
  value: string
  color: string
  icon: React.ReactNode
}) {
  return (
    <div className="glass relative overflow-hidden rounded-sm">
      <div className="scanline" />
      <div className="relative flex items-center gap-3 p-4">
        <span style={{ color }}>{icon}</span>
        <div>
          <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
            {label}
          </div>
          <div className="font-display text-2xl font-bold text-glow" style={{ color }}>
            {value}
          </div>
        </div>
      </div>
    </div>
  )
}

function Pager({
  page,
  totalPages,
  total,
  onPage,
}: {
  page: number
  totalPages: number
  total: number
  onPage: (updater: (p: number) => number) => void
}) {
  return (
    <div className="mt-4 flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <NeonButton onClick={() => onPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
          ◂ PREV
        </NeonButton>
        <NeonButton
          onClick={() => onPage((p) => Math.min(totalPages, p + 1))}
          disabled={page >= totalPages}
        >
          NEXT ▸
        </NeonButton>
      </div>
    </div>
  )
}

function CenterSync() {
  return (
    <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
      <Loader2 className="size-4 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
    </div>
  )
}

function FailureBox({ error }: { error: unknown }) {
  return (
    <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
      FAILURE · {getErrMsg(error)}
    </div>
  )
}

function EmptyBox({ hint }: { hint: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16">
      <Inbox className="size-10 text-cyan-400/50" />
      <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
        {hint}
      </div>
    </div>
  )
}
