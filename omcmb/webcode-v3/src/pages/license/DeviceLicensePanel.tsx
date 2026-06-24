import { useCallback, useMemo, useRef, useState } from 'react'
import {
  Search,
  RefreshCcw,
  Loader2,
  CloudUpload,
  Download,
  Trash2,
  Package,
  CircleCheck,
  AlertTriangle,
} from 'lucide-react'

import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatBytes, formatTime } from '@/lib/format'
import {
  useDeviceLicenses,
  useImportDeviceLicenses,
  useDeleteDeviceLicense,
} from '@core/hooks/api/useDeviceLicense'
import { useProductList, useProductNameResolver } from '@core/hooks/api/useProducts'
import { deviceLicenseApi } from '@core/services/api/deviceLicenseApi'
import type { LicenseImportResult } from '@core/services/api/deviceLicenseApi'

import { Modal } from './Modal'

/**
 * DeviceLicensePanel — 设备 license 文件库（一台设备一份最新 .lic）。
 *
 * 数据：useDeviceLicenses（分页 + SN/enbName/productType 过滤）。
 * 操作：导入（useImportDeviceLicenses，多文件）、下载（deviceLicenseApi.download
 *      签发临时 URL）、删除（useDeleteDeviceLicense）。三态：loading / 空 / 错误。
 */
const PAGE_SIZE = 12

export function DeviceLicensePanel({
  onNotice,
}: {
  onNotice?: (msg: string, tone?: 'ok' | 'err') => void
}) {
  const [page, setPage] = useState(1)
  const [snInput, setSnInput] = useState('')
  const [sn, setSn] = useState('')
  // #602：产品名称下拉过滤，传 product.id
  const [productId, setProductId] = useState('')
  const [importOpen, setImportOpen] = useState(false)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(sn.trim() ? { serialNumber: sn.trim() } : {}),
      ...(productId ? { productId } : {}),
    }),
    [page, sn, productId],
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useDeviceLicenses(params)
  const del = useDeleteDeviceLicense()
  const { data: productsData } = useProductList()
  const productNameOptions = useMemo(
    () =>
      (productsData?.items ?? [])
        .map((p) => ({ label: `${p.name} (${p.tech})`, value: p.id }))
        .sort((a, b) => a.label.localeCompare(b.label, 'zh-CN')),
    [productsData]
  )
  const resolveProductName = useProductNameResolver()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const applySearch = () => {
    setSn(snInput)
    setPage(1)
  }

  const handleDownload = useCallback(
    async (serialNumber: string) => {
      try {
        await deviceLicenseApi.download(serialNumber)
      } catch (e) {
        onNotice?.(`下载失败 · ${(e as Error)?.message ?? serialNumber}`, 'err')
      }
    },
    [onNotice],
  )

  const handleDelete = useCallback(
    (serialNumber: string) => {
      if (!window.confirm(`确认删除 ${serialNumber} 的 license 文件？此操作不可恢复。`)) return
      del.mutate(serialNumber, {
        onSuccess: () => onNotice?.(`已删除 ${serialNumber} 的 license`, 'ok'),
        onError: (e) => onNotice?.(`删除失败 · ${e.message}`, 'err'),
      })
    },
    [del, onNotice],
  )

  return (
    <GlassPanel
      title="DEVICE LICENSE LIBRARY · 设备授权文件库"
      meta={`${total} FILES`}
      className="min-h-0"
    >
      {/* 工具条 */}
      <div className="flex flex-wrap items-center gap-2 border-b border-cyan-500/10 px-3.5 py-2.5">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
          <input
            className="neon-input w-64 pl-9"
            placeholder="按设备 SN 过滤"
            value={snInput}
            onChange={(e) => setSnInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') applySearch()
            }}
          />
        </div>
        <NeonButton icon={<Search />} onClick={applySearch}>
          SEARCH
        </NeonButton>
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
        <NeonButton
          icon={<RefreshCcw />}
          onClick={() => refetch()}
          className={isFetching ? 'opacity-70' : ''}
        >
          REFRESH
        </NeonButton>
        <div className="flex-1" />
        <NeonButton icon={<CloudUpload />} onClick={() => setImportOpen(true)}>
          IMPORT
        </NeonButton>
      </div>

      {/* 表头 */}
      <div className="grid grid-cols-[1.4fr_1.2fr_0.9fr_0.7fr_1fr_140px] gap-3 border-b border-cyan-500/10 px-3.5 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
        <span>SERIAL NO.</span>
        <span>eNB NAME</span>
        <span>PRODUCT NAME · 产品名称</span>
        <span>SIZE</span>
        <span>UPDATED</span>
        <span className="text-right">ACTIONS</span>
      </div>

      {/* 列表三态 */}
      <div className="min-h-[180px]">
        {isLoading ? (
          <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
            <Loader2 className="size-4 animate-spin" />
            <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
          </div>
        ) : isError ? (
          <div className="m-3.5 flex items-start gap-2 rounded-sm border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
            <AlertTriangle className="mt-0.5 size-4 shrink-0" />
            FAILURE · {error instanceof Error ? error.message : '未知错误'}
          </div>
        ) : rows.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-3 py-14">
            <Package className="size-10 text-cyan-300/30" />
            <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
              NO LICENSE FILES · 暂无设备授权文件
            </div>
            <NeonButton icon={<CloudUpload />} onClick={() => setImportOpen(true)}>
              导入第一个 .lic
            </NeonButton>
          </div>
        ) : (
          rows.map((r) => (
            <div
              key={r.serialNumber}
              className="grid grid-cols-[1.4fr_1.2fr_0.9fr_0.7fr_1fr_140px] items-center gap-3 border-b border-cyan-500/8 px-3.5 py-2.5 transition-colors hover:bg-cyan-500/5"
            >
              <span className="truncate font-mono text-xs text-cyan-100">{r.serialNumber}</span>
              <span className="truncate text-xs text-cyan-100/80">{r.enbName ?? '—'}</span>
              <span className="truncate font-mono text-[11px] text-cyan-300/70">
                {(() => {
                  const display = resolveProductName(r.productType)
                  return display ? display : '—'
                })()}
              </span>
              <span className="font-mono text-[11px] text-cyan-300/70">
                {formatBytes(r.fileSize)}
              </span>
              <span className="font-mono text-[11px] text-cyan-300/70">
                {formatTime(r.updateTime)}
              </span>
              <div className="flex items-center justify-end gap-1.5">
                <button
                  type="button"
                  title="下载"
                  onClick={() => handleDownload(r.serialNumber)}
                  className="rounded-sm border border-cyan-500/25 p-1.5 text-cyan-300/70 transition-colors hover:border-cyan-400/60 hover:bg-cyan-500/10 hover:text-cyan-100"
                >
                  <Download className="size-3.5" />
                </button>
                <button
                  type="button"
                  title="删除"
                  disabled={del.isPending}
                  onClick={() => handleDelete(r.serialNumber)}
                  className="rounded-sm border border-rose-500/25 p-1.5 text-rose-300/70 transition-colors hover:border-rose-400/60 hover:bg-rose-500/10 hover:text-rose-200 disabled:opacity-40"
                >
                  <Trash2 className="size-3.5" />
                </button>
              </div>
            </div>
          ))
        )}
      </div>

      {/* 分页 */}
      {!isLoading && !isError && rows.length > 0 ? (
        <div className="flex items-center justify-between border-t border-cyan-500/10 px-3.5 py-2.5">
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
      ) : null}

      <ImportDeviceLicenseModal
        open={importOpen}
        onClose={() => setImportOpen(false)}
        onDone={(r) => {
          const ok = r.succeeded.length
          const bad = r.failed.length
          onNotice?.(
            `导入完成 · 成功 ${ok} · 失败 ${bad}`,
            bad > 0 ? 'err' : 'ok',
          )
        }}
      />
    </GlassPanel>
  )
}

// ─────────────────────────────────────────────────────────────────────────
// 导入弹窗（多文件 <SN>.lic）
// ─────────────────────────────────────────────────────────────────────────

function ImportDeviceLicenseModal({
  open,
  onClose,
  onDone,
}: {
  open: boolean
  onClose: () => void
  onDone: (r: LicenseImportResult) => void
}) {
  const [files, setFiles] = useState<File[]>([])
  const [result, setResult] = useState<LicenseImportResult | null>(null)
  const [dragOver, setDragOver] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const importMut = useImportDeviceLicenses()

  const reset = useCallback(() => {
    setFiles([])
    setResult(null)
    setDragOver(false)
  }, [])

  const close = useCallback(() => {
    reset()
    onClose()
  }, [reset, onClose])

  const addFiles = (fl: FileList | null) => {
    if (!fl) return
    setResult(null)
    setFiles((prev) => {
      const names = new Set(prev.map((f) => f.name))
      const merged = [...prev]
      for (const f of Array.from(fl)) {
        if (!names.has(f.name)) merged.push(f)
      }
      return merged
    })
  }

  const submit = () => {
    if (files.length === 0) return
    importMut.mutate(files, {
      onSuccess: (r) => {
        setResult(r)
        onDone(r)
      },
    })
  }

  return (
    <Modal
      open={open}
      title="IMPORT DEVICE LICENSE · 导入设备授权"
      subtitle="MULTI-FILE · <SERIAL>.lic · ONE FILE PER DEVICE"
      onClose={close}
      footer={
        <>
          <NeonButton onClick={close} disabled={importMut.isPending}>
            关闭
          </NeonButton>
          <NeonButton onClick={submit} disabled={files.length === 0 || importMut.isPending}>
            {importMut.isPending ? '上传中…' : `导入 ${files.length} 个文件`}
          </NeonButton>
        </>
      }
    >
      <button
        type="button"
        onClick={() => inputRef.current?.click()}
        onDragOver={(e) => {
          e.preventDefault()
          setDragOver(true)
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={(e) => {
          e.preventDefault()
          setDragOver(false)
          addFiles(e.dataTransfer.files)
        }}
        className={`flex w-full flex-col items-center justify-center gap-2 rounded-sm border border-dashed py-8 transition-colors ${
          dragOver
            ? 'border-cyan-400/80 bg-cyan-500/10'
            : 'border-cyan-500/30 bg-cyan-500/[0.03] hover:border-cyan-400/60'
        }`}
      >
        <CloudUpload className="size-8 text-cyan-300/70" />
        <div className="font-mono text-xs uppercase tracking-[0.18em] text-cyan-300/70">
          点击选择或拖入 .lic 文件（可多选）
        </div>
        <div className="font-mono text-[10px] text-cyan-300/45">
          文件名须为 &lt;设备SN&gt;.lic
        </div>
        <input
          ref={inputRef}
          type="file"
          accept=".lic"
          multiple
          className="hidden"
          onChange={(e) => {
            addFiles(e.target.files)
            e.target.value = ''
          }}
        />
      </button>

      {/* 待上传文件 */}
      {files.length > 0 ? (
        <div className="mt-3 space-y-1">
          <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            QUEUE · {files.length} 个文件
          </div>
          {files.map((f) => (
            <div
              key={f.name}
              className="flex items-center justify-between rounded-sm border border-cyan-500/15 bg-cyan-500/5 px-3 py-1.5"
            >
              <span className="truncate font-mono text-xs text-cyan-100">{f.name}</span>
              <button
                type="button"
                onClick={() => setFiles((prev) => prev.filter((x) => x.name !== f.name))}
                className="font-mono text-[11px] text-rose-300/70 hover:text-rose-200"
              >
                移除
              </button>
            </div>
          ))}
        </div>
      ) : null}

      {/* 导入结果 */}
      {result ? (
        <div className="mt-4 space-y-2">
          {result.succeeded.length > 0 ? (
            <div className="rounded-sm border border-emerald-500/30 bg-emerald-500/5 px-3 py-2">
              <div className="flex items-center gap-1.5 font-mono text-[11px] text-emerald-300">
                <CircleCheck className="size-3.5" />
                成功 {result.succeeded.length}
              </div>
              <div className="mt-1 flex flex-wrap gap-1.5">
                {result.succeeded.map((s) => (
                  <span
                    key={s}
                    className="rounded-sm border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 font-mono text-[10px] text-emerald-200"
                  >
                    {s}
                  </span>
                ))}
              </div>
            </div>
          ) : null}
          {result.failed.length > 0 ? (
            <div className="rounded-sm border border-rose-500/30 bg-rose-500/5 px-3 py-2">
              <div className="flex items-center gap-1.5 font-mono text-[11px] text-rose-300">
                <AlertTriangle className="size-3.5" />
                失败 {result.failed.length}
              </div>
              <div className="mt-1 space-y-1">
                {result.failed.map((f, i) => (
                  <div key={`${f.fileName}-${i}`} className="font-mono text-[11px] text-rose-200/85">
                    {f.fileName} · {f.errorCode} · {f.message}
                  </div>
                ))}
              </div>
            </div>
          ) : null}
        </div>
      ) : null}
    </Modal>
  )
}
