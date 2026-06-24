import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Boxes, Download, Eye, FileBox, Package, RefreshCcw, Star, Trash2, Upload, X } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  EmptyRow,
  ErrorRow,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
  formatBytes,
  formatTime,
} from '@/components/layout/PageShell'

import {
  useSoftwareVersions,
  useToggleRecommend,
  useDeleteSoftwareVersions,
  useDownloadFirmware,
  useUploadFirmware,
} from '@core/hooks/api/useSoftware'
import { useProductList } from '@core/hooks/api/useProducts'
import type { SoftwareVersion } from '@core/mock/data/software'

import { VERSION_STATUS } from './_shared'
import { IconBtn, Stat } from './_components'

// ===========================================================================
// 固件管理 — 对照 v1 webcode/src/pages/software/FirmwareUpload
// 按文件类型（固件/补丁/FPGA）浏览固件库，支持下载 / 推荐切换 / 删除 / 详情下钻。
// qa-614 #379：补齐导入入口（三皮肤铁律）。本皮肤不引入 antd，导入表单用 shadcn 基础
// 组件 + 原生 file input（参考 v3 Firmware.tsx 写法），与 v1/v3 字段对齐。
// ===========================================================================

type FileTypeTab = 'upgrade' | 'patch' | 'fpga'

const FILE_TYPE_PARAM: Record<FileTypeTab, 0 | 1 | 6> = {
  upgrade: 0,
  patch: 1,
  fpga: 6,
}

const FILE_TYPE_TABS: { key: FileTypeTab; label: string }[] = [
  { key: 'upgrade', label: '升级固件' },
  { key: 'patch', label: '补丁' },
  { key: 'fpga', label: 'FPGA' },
]

export default function FirmwareUpload() {
  const navigate = useNavigate()
  const [fileType, setFileType] = useState<FileTypeTab>('upgrade')
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [search, setSearch] = useState('')
  const [filterProductId, setFilterProductId] = useState('')
  const [showImport, setShowImport] = useState(false)

  // #492：固件按产品名（产品中心目录）。products 用于过滤/上传选择 + product_id→产品名展示。
  const { data: productsData } = useProductList()
  const products = useMemo(() => productsData?.items ?? [], [productsData])
  const productNameById = useMemo(() => {
    const map = new Map<string, string>()
    products.forEach((p) => map.set(p.id, p.name))
    return map
  }, [products])

  const params = useMemo(
    () => ({
      page,
      pageSize,
      fileType: FILE_TYPE_PARAM[fileType],
      ...(filterProductId ? { productId: filterProductId } : {}),
    }),
    [page, pageSize, fileType, filterProductId]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useSoftwareVersions(params)
  const toggleRecommend = useToggleRecommend()
  const remove = useDeleteSoftwareVersions()
  const download = useDownloadFirmware()
  const busy = toggleRecommend.isPending || remove.isPending || download.isPending

  const allRows = data?.items ?? []
  const kw = search.trim().toLowerCase()
  const rows = kw
    ? allRows.filter(
        (v) =>
          v.versionCode.toLowerCase().includes(kw) ||
          v.versionName.toLowerCase().includes(kw) ||
          (v.fileName ?? '').toLowerCase().includes(kw)
      )
    : allRows
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const recommendCount = allRows.filter((v) => v.recommend).length
  const totalSize = allRows.reduce((acc, v) => acc + (v.fileSize ?? 0), 0)

  const cols = ['版本号', '文件名', '设备型号', '厂商', '状态', '推荐', '文件大小', '上传时间', '操作']

  return (
    <PageShell title="固件管理" description="固件 / 补丁 / FPGA 文件库：下载、推荐切换、删除与详情">
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="文件总数" value={total} icon={FileBox} />
        <Stat label="推荐固件" value={recommendCount} icon={Star} tone="amber" />
        <Stat label="本页占用" value={formatBytes(totalSize)} icon={Boxes} tone="muted" />
        <Stat label="当前分类" value={FILE_TYPE_TABS.find((t) => t.key === fileType)?.label ?? '—'} icon={Package} />
      </div>

      <div className="mb-3 flex items-center gap-1 border-b">
        {FILE_TYPE_TABS.map((tab) => (
          <button
            key={tab.key}
            type="button"
            className={`-mb-px border-b-2 px-4 py-2 text-sm font-medium transition-colors ${
              fileType === tab.key
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
            onClick={() => {
              setFileType(tab.key)
              setPage(1)
            }}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <Input
          className="w-72"
          placeholder="搜索版本号 / 文件名"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <Select
          value={filterProductId || 'all'}
          onValueChange={(v) => {
            setFilterProductId(v === 'all' ? '' : v)
            setPage(1)
          }}
        >
          <SelectTrigger className="w-44">
            <SelectValue placeholder="产品名称" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部产品</SelectItem>
            {products.map((p) => (
              <SelectItem key={p.id} value={p.id}>
                {p.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button
          size="sm"
          className="ml-auto"
          onClick={() => setShowImport((v) => !v)}
        >
          <Upload className="size-3.5" /> 导入固件
        </Button>
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          <RefreshCcw className={isFetching ? 'animate-spin' : ''} /> 刷新
        </Button>
      </div>

      {showImport ? (
        <FirmwareImportPanel
          fileType={fileType}
          products={products}
          onClose={() => setShowImport(false)}
          onSuccess={() => {
            setShowImport(false)
            void refetch()
          }}
        />
      ) : null}

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无固件文件</EmptyRow>
            ) : (
              rows.map((v: SoftwareVersion) => {
                const st = VERSION_STATUS[v.status]
                return (
                  <TableRow key={v.id}>
                    <TableCell>
                      <button
                        type="button"
                        className="text-left font-mono text-xs text-primary hover:underline"
                        onClick={() => navigate(`/software/version`)}
                        title="查看版本详情"
                      >
                        {v.versionCode}
                      </button>
                    </TableCell>
                    <TableCell className="max-w-[200px] truncate text-xs" title={v.fileName}>
                      {v.fileName || '—'}
                    </TableCell>
                    <TableCell>{(v.productId && productNameById.get(v.productId)) || v.deviceType || '—'}</TableCell>
                    <TableCell>{v.vendor || v.manufacturer || '—'}</TableCell>
                    <TableCell>
                      <Badge variant={st.variant}>{st.label}</Badge>
                    </TableCell>
                    <TableCell>
                      {v.recommend ? (
                        <Badge variant="warning">
                          <Star className="mr-1 size-3" /> 推荐
                        </Badge>
                      ) : (
                        <span className="text-xs text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatBytes(v.fileSize)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(v.releaseDate)}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        <IconBtn
                          title="下载"
                          disabled={busy}
                          onClick={() =>
                            download.mutate({
                              id: v.id,
                              fileName: v.fileName || `${v.versionCode}.bin`,
                            })
                          }
                        >
                          <Download className="size-3.5" />
                        </IconBtn>
                        <IconBtn
                          title={v.recommend ? '取消推荐' : '设为推荐'}
                          disabled={busy}
                          onClick={() => toggleRecommend.mutate(v.id)}
                        >
                          <Star className={`size-3.5 ${v.recommend ? 'fill-current text-amber-500' : ''}`} />
                        </IconBtn>
                        <IconBtn title="详情" disabled={busy} onClick={() => navigate(`/software/version`)}>
                          <Eye className="size-3.5" />
                        </IconBtn>
                        <IconBtn
                          title="删除"
                          tone="destructive"
                          disabled={busy}
                          onClick={() => remove.mutate([v.id])}
                        >
                          <Trash2 className="size-3.5" />
                        </IconBtn>
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={setPage} />
    </PageShell>
  )
}

// FirmwareImportPanel — qa-614 #379：v2 导入固件表单（不引 antd）。
// 字段与 v1/v3 对齐：产品类型（可手填）、版本号（必填）、推荐、描述、文件。
function FirmwareImportPanel({
  fileType,
  products,
  onClose,
  onSuccess,
}: {
  fileType: FileTypeTab
  products: { id: string; name: string; tech: string }[]
  onClose: () => void
  onSuccess: () => void
}) {
  const upload = useUploadFirmware()
  const [file, setFile] = useState<File | null>(null)
  const [version, setVersion] = useState('')
  // #492：固件所属产品 = 选产品名，提交 product_id。
  const [productId, setProductId] = useState('')
  const [recommend, setRecommend] = useState(false)
  const [description, setDescription] = useState('')
  const [err, setErr] = useState('')
  // #624：上传进度（0-100），driven by axios onUploadProgress。
  const [progress, setProgress] = useState(0)

  const handleSubmit = () => {
    setErr('')
    if (!version.trim()) {
      setErr('版本号必填')
      return
    }
    if (!file) {
      setErr('请选择固件文件')
      return
    }
    setProgress(0)
    upload.mutate(
      {
        file,
        metadata: {
          version: version.trim(),
          productId: productId || undefined,
          releaseNotes: description,
          fileType: FILE_TYPE_PARAM[fileType],
          recommend,
          description,
        },
        onProgress: setProgress,
      },
      {
        onSuccess,
        // qa-614 #372/#379：透出后端真实失败原因（如重复导入 → 409 文案）。
        onError: (e) => {
          setProgress(0)
          setErr(e instanceof Error && e.message ? e.message : '导入失败')
        },
      }
    )
  }

  return (
    <Card className="mb-3">
      <CardContent className="space-y-3 pt-4">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium">
            导入{FILE_TYPE_TABS.find((t) => t.key === fileType)?.label ?? ''}
          </span>
          <Button variant="ghost" size="sm" onClick={onClose} aria-label="关闭">
            <X className="size-4" />
          </Button>
        </div>

        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          <div className="space-y-1.5">
            <Label htmlFor="fw-product-name">产品名称</Label>
            {/* #492：固件所属产品 = 选产品名（产品中心目录），提交 product_id。 */}
            <Select value={productId || undefined} onValueChange={setProductId}>
              <SelectTrigger id="fw-product-name">
                <SelectValue placeholder="选择产品名称" />
              </SelectTrigger>
              <SelectContent>
                {products.map((p) => (
                  <SelectItem key={p.id} value={p.id}>
                    {p.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="fw-version">版本号 *</Label>
            <Input
              id="fw-version"
              maxLength={45}
              value={version}
              onChange={(e) => setVersion(e.target.value)}
              placeholder="如 V100R011C10SPC200"
            />
          </div>
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="fw-file">固件文件 *</Label>
          <Input
            id="fw-file"
            type="file"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="fw-desc">描述</Label>
          <Input
            id="fw-desc"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="发布说明 / 描述"
          />
        </div>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={recommend}
            onChange={(e) => setRecommend(e.target.checked)}
          />
          设为推荐版本
        </label>

        {err ? <div className="text-xs text-destructive">{err}</div> : null}

        {upload.isPending || progress > 0 ? (
          <div className="space-y-1">
            <div className="flex items-center justify-between text-xs text-muted-foreground">
              <span>上传进度</span>
              <span>{progress}%</span>
            </div>
            <div className="h-2 w-full overflow-hidden rounded bg-muted">
              <div
                className="h-full bg-primary transition-[width] duration-150"
                style={{ width: `${progress}%` }}
              />
            </div>
          </div>
        ) : null}

        <div className="flex justify-end gap-2">
          <Button variant="outline" size="sm" onClick={onClose}>
            取消
          </Button>
          <Button size="sm" disabled={upload.isPending} onClick={handleSubmit}>
            {upload.isPending ? '提交中…' : '确认导入'}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
