import { useMemo, useState, useCallback } from 'react'
import {
  Search,
  RefreshCcw,
  Loader2,
  Package,
  Star,
  Upload as UploadIcon,
  Download,
  Pencil,
  Trash2,
  HardDrive,
  AlertTriangle,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatTime } from '@/lib/format'
import {
  useSoftwareVersions,
  useUploadFirmware,
  useDeleteSoftwareVersions,
  useToggleRecommend,
  useDownloadFirmware,
  useUpdateFirmware,
} from '@core/hooks/api/useSoftware'
import { useProductClasses } from '@core/hooks/api/useDevices'
import type { SoftwareVersion } from '@core/mock/data/software'

import { NEON, StatCard, Drawer, Syncing, ErrorBlock, EmptyBlock, Pager, formatFileSize } from './_shared'

// ---------------------------------------------------------------------------
// 固件库 · software/firmware
// 固件文件库：IMAGE / PATCH / FPGA 三类，上传 / 修改 / 删除 / 推荐 / 下载
// ---------------------------------------------------------------------------

type FileTypeTab = 'upgrade' | 'patch' | 'fpga'

const FILE_TYPE_PARAM: Record<FileTypeTab, 0 | 1 | 6> = {
  upgrade: 0,
  patch: 1,
  fpga: 6,
}

const FILE_TYPE_LABEL: Record<FileTypeTab, string> = {
  upgrade: 'IMAGE',
  patch: 'PATCH',
  fpga: 'FPGA',
}

const FALLBACK_PRODUCT_CLASSES = ['PM-B4860', 'QAFA', 'QAFB', 'FAP/BU1810']

export default function Firmware() {
  const [fileType, setFileType] = useState<FileTypeTab>('upgrade')
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [keyword, setKeyword] = useState('')
  const [drawer, setDrawer] = useState<{ mode: 'add' | 'edit'; file: SoftwareVersion | null } | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<SoftwareVersion | null>(null)

  const params = useMemo(
    () => ({ page, pageSize, fileType: FILE_TYPE_PARAM[fileType] }),
    [page, fileType]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useSoftwareVersions(params)
  const allItems = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const download = useDownloadFirmware()
  const toggleRecommend = useToggleRecommend()
  const deleteMutation = useDeleteSoftwareVersions()

  const items = useMemo(() => {
    const k = keyword.trim().toLowerCase()
    if (!k) return allItems
    return allItems.filter(
      (v) => v.versionCode.toLowerCase().includes(k) || (v.versionName ?? '').toLowerCase().includes(k)
    )
  }, [allItems, keyword])

  const stat = useMemo(() => {
    const recommended = allItems.filter((v) => v.recommend).length
    const totalBytes = allItems.reduce((acc, v) => acc + (v.fileSize || 0), 0)
    return { recommended, totalBytes }
  }, [allItems])

  const handleDelete = useCallback(() => {
    if (!deleteTarget) return
    deleteMutation.mutate([deleteTarget.id], { onSuccess: () => setDeleteTarget(null) })
  }, [deleteTarget, deleteMutation])

  return (
    <PageShell
      code="F06"
      title="FIRMWARE LIBRARY · 固件库"
      subtitle="IMAGE · PATCH · FPGA REGISTRY"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-56 pl-9"
              placeholder="版本 / 名称"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
          <NeonButton icon={<UploadIcon />} onClick={() => setDrawer({ mode: 'add', file: null })}>
            导入固件
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 lg:grid-cols-4">
        <StatCard label={`${FILE_TYPE_LABEL[fileType]} 总数 · FILES`} value={total.toLocaleString()} color={NEON.cyan} icon={<Package className="size-4" />} />
        <StatCard label="推荐 · RECOMMENDED" value={stat.recommended.toLocaleString()} color={NEON.gold} icon={<Star className="size-4" />} />
        <StatCard label="本页容量 · SIZE" value={formatFileSize(stat.totalBytes)} color={NEON.violet} icon={<HardDrive className="size-4" />} />
        <StatCard label="当前文件类型 · TYPE" value={FILE_TYPE_LABEL[fileType]} color={NEON.blue} icon={<HardDrive className="size-4" />} />
      </div>

      {/* 文件类型 tab */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        {(['upgrade', 'patch', 'fpga'] as const).map((ft) => (
          <button
            key={ft}
            type="button"
            onClick={() => { setFileType(ft); setPage(1) }}
            className={`chip transition-all ${fileType === ft ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55 hover:opacity-100'}`}
            style={{ color: NEON.cyan }}
          >
            {FILE_TYPE_LABEL[ft]}
          </button>
        ))}
        {isFetching ? (
          <span className="flex items-center gap-1.5 text-[11px] text-cyan-300/60">
            <Loader2 className="size-3 animate-spin" /> SYNC
          </span>
        ) : null}
      </div>

      {isLoading ? (
        <Syncing label="SCANNING LIBRARY…" />
      ) : isError ? (
        <ErrorBlock msg={error instanceof Error ? error.message : '未知错误'} />
      ) : items.length === 0 ? (
        <EmptyBlock label="NO FILES · 无固件文件" />
      ) : (
        <div className="space-y-1.5">
          {/* 表头 */}
          <div className="grid grid-cols-[1.6fr_1.4fr_110px_150px_220px] items-center gap-3 px-3 py-1 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/45">
            <span>版本 · VERSION</span>
            <span>制式 · PRODUCT</span>
            <span>大小 · SIZE</span>
            <span>上传 · UPLOADED</span>
            <span className="text-right">操作 · ACTIONS</span>
          </div>
          {items.map((v) => (
            <div
              key={v.id}
              className="fleet-row grid grid-cols-[1.6fr_1.4fr_110px_150px_220px] items-center gap-3 rounded-sm px-3 py-2"
              style={{ ['--row-color' as never]: v.recommend ? NEON.gold : NEON.cyan }}
            >
              <div className="flex min-w-0 items-center gap-2">
                {v.recommend ? (
                  <Star className="size-3.5 shrink-0" style={{ color: NEON.gold, fill: NEON.gold }} />
                ) : (
                  <span className="size-3.5 shrink-0" />
                )}
                <span className="truncate font-mono text-xs text-cyan-100">{v.versionCode}</span>
              </div>
              <span className="truncate font-mono text-[11px] text-cyan-300/70">{v.deviceType || '—'}</span>
              <span className="font-mono text-[11px] text-cyan-100/80">{formatFileSize(v.fileSize)}</span>
              <span className="font-mono text-[10px] text-cyan-300/60">{v.releaseDate ? formatTime(v.releaseDate) : '—'}</span>
              <div className="flex items-center justify-end gap-1.5">
                <button
                  type="button"
                  title="下载"
                  disabled={download.isPending || !v.fileName}
                  onClick={() => download.mutate({ id: v.id, fileName: v.fileName })}
                  className="text-cyan-300/70 transition-colors hover:text-cyan-200 disabled:opacity-40"
                >
                  <Download className="size-4" />
                </button>
                <button
                  type="button"
                  title="编辑"
                  onClick={() => setDrawer({ mode: 'edit', file: v })}
                  className="text-cyan-300/70 transition-colors hover:text-cyan-200"
                >
                  <Pencil className="size-4" />
                </button>
                <button
                  type="button"
                  title={v.recommend ? '取消推荐' : '设为推荐'}
                  disabled={toggleRecommend.isPending}
                  onClick={() => toggleRecommend.mutate(v.id)}
                  className="transition-transform hover:scale-110 disabled:opacity-40"
                >
                  <Star className="size-4" style={{ color: v.recommend ? NEON.gold : NEON.dim, fill: v.recommend ? NEON.gold : 'transparent' }} />
                </button>
                <button
                  type="button"
                  title="删除"
                  onClick={() => setDeleteTarget(v)}
                  className="text-rose-400/70 transition-colors hover:text-rose-300"
                >
                  <Trash2 className="size-4" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      <Pager page={page} totalPages={totalPages} pageSize={pageSize} total={total} onPage={setPage} />

      {drawer ? (
        <FirmwareDrawer
          mode={drawer.mode}
          file={drawer.file}
          fileType={fileType}
          onClose={() => setDrawer(null)}
        />
      ) : null}

      {deleteTarget ? (
        <Drawer title="CONFIRM DELETE · 删除确认" onClose={() => setDeleteTarget(null)}>
          <div className="space-y-4 p-4">
            <div className="flex items-start gap-3 rounded-sm border border-amber-500/30 bg-amber-500/[0.05] p-3">
              <AlertTriangle className="mt-0.5 size-5 shrink-0 text-amber-400" />
              <div className="space-y-1 font-mono text-[12px] text-cyan-100/85">
                <div>确认删除以下固件文件？此操作不可恢复。</div>
                <div>版本 <span className="text-cyan-100">{deleteTarget.versionCode}</span></div>
                <div>文件 <span className="break-all text-cyan-100">{deleteTarget.fileName || '—'}</span></div>
                <div>大小 <span className="text-cyan-100">{formatFileSize(deleteTarget.fileSize)}</span></div>
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <NeonButton onClick={() => setDeleteTarget(null)}>取消</NeonButton>
              <NeonButton tone="danger" icon={<Trash2 />} disabled={deleteMutation.isPending} onClick={handleDelete}>
                {deleteMutation.isPending ? '删除中…' : '确认删除'}
              </NeonButton>
            </div>
          </div>
        </Drawer>
      ) : null}
    </PageShell>
  )
}

function FirmwareDrawer({
  mode,
  file,
  fileType,
  onClose,
}: {
  mode: 'add' | 'edit'
  file: SoftwareVersion | null
  fileType: FileTypeTab
  onClose: () => void
}) {
  const { data: productClassesData } = useProductClasses()
  const productOptions = useMemo(
    () => (productClassesData && productClassesData.length > 0 ? productClassesData : FALLBACK_PRODUCT_CLASSES),
    [productClassesData]
  )

  const upload = useUploadFirmware()
  const update = useUpdateFirmware()

  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [version, setVersion] = useState(file?.versionCode ?? '')
  const [productClass, setProductClass] = useState(file?.deviceType ?? productOptions[0] ?? '')
  const [recommend, setRecommend] = useState(Boolean(file?.recommend))
  const [description, setDescription] = useState(file?.description ?? file?.releaseNotes ?? '')
  const [err, setErr] = useState('')

  const pending = upload.isPending || update.isPending

  const handleSubmit = () => {
    setErr('')
    if (!version.trim()) {
      setErr('版本号必填')
      return
    }
    if (mode === 'edit') {
      if (!file) return
      update.mutate(
        {
          id: file.id,
          metadata: { productClass, version: version.trim(), recommend, description },
        },
        { onSuccess: onClose, onError: (e) => setErr(e instanceof Error ? e.message : '修改失败') }
      )
      return
    }
    if (!selectedFile) {
      setErr('请选择固件文件')
      return
    }
    upload.mutate(
      {
        file: selectedFile,
        metadata: {
          version: version.trim(),
          productClass,
          releaseNotes: description,
          fileType: FILE_TYPE_PARAM[fileType],
          recommend,
          description,
        },
      },
      { onSuccess: onClose, onError: (e) => setErr(e instanceof Error ? e.message : '上传失败') }
    )
  }

  return (
    <Drawer title={mode === 'add' ? `IMPORT ${FILE_TYPE_LABEL[fileType]} · 导入固件` : 'EDIT · 修改固件'} onClose={onClose}>
      <div className="space-y-4 p-4">
        {mode === 'add' ? (
          <label className="block">
            <span className="mb-1 block font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">固件文件 · FILE</span>
            <div className="flex flex-col items-center gap-2 rounded-sm border border-dashed border-cyan-500/30 bg-cyan-500/[0.03] px-4 py-6 text-center transition-colors hover:border-cyan-400/60">
              <UploadIcon className="size-6 text-cyan-300/60" />
              <span className="font-mono text-[11px] text-cyan-100/80">
                {selectedFile ? selectedFile.name : '点击选择固件文件'}
              </span>
              <input
                type="file"
                className="hidden"
                onChange={(e) => setSelectedFile(e.target.files?.[0] ?? null)}
              />
            </div>
          </label>
        ) : (
          <div>
            <span className="mb-1 block font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">文件名 · FILE</span>
            <div className="break-all rounded-sm border border-cyan-500/15 bg-cyan-500/[0.03] px-3 py-2 font-mono text-[11px] text-cyan-100/80">
              {file?.fileName || '—'}
            </div>
          </div>
        )}

        <label className="block">
          <span className="mb-1 block font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">制式 · PRODUCT CLASS</span>
          <select className="neon-input w-full cursor-pointer" value={productClass} onChange={(e) => setProductClass(e.target.value)}>
            {productOptions.map((o) => (
              <option key={o} value={o}>{o}</option>
            ))}
          </select>
        </label>

        <label className="block">
          <span className="mb-1 block font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">版本号 · VERSION</span>
          <input className="neon-input w-full" maxLength={45} value={version} onChange={(e) => setVersion(e.target.value)} placeholder="如 V100R011C10SPC200" />
        </label>

        <label className="flex items-center gap-2">
          <input type="checkbox" checked={recommend} onChange={(e) => setRecommend(e.target.checked)} className="accent-amber-400" />
          <span className="font-mono text-[12px] text-cyan-100/85">设为推荐版本</span>
        </label>

        <label className="block">
          <span className="mb-1 block font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">描述 · DESCRIPTION</span>
          <textarea className="neon-input min-h-[72px] w-full" value={description} onChange={(e) => setDescription(e.target.value)} placeholder="发布说明 / 描述" />
        </label>

        {err ? <div className="font-mono text-[11px] text-rose-300">{err}</div> : null}

        <div className="flex justify-end gap-2">
          <NeonButton onClick={onClose}>取消</NeonButton>
          <NeonButton icon={<UploadIcon />} disabled={pending} onClick={handleSubmit}>
            {pending ? '提交中…' : '确认'}
          </NeonButton>
        </div>
      </div>
    </Drawer>
  )
}
