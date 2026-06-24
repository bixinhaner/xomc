import { useMemo, useState } from 'react'
import { Download, RefreshCcw, Search, Trash2, X } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
import { cn } from '@/lib/utils'

import {
  useConfigSnapshots,
  useBatchDeleteConfigSnapshots,
} from '@core/hooks/api/useConfigSnapshot'
import { useProductList, useProductNameResolver } from '@core/hooks/api/useProducts'
import {
  configSnapshotApi,
  type ConfigSnapshot,
  type SnapshotSource,
} from '@core/services/api/configSnapshotApi'

// ============================================================
// 配置快照库 — 对齐 v1 webcode/src/pages/backup/ConfigSnapshotLibrary（路由 /backup/config-snapshots）
//   · 每设备最新一份配置快照（serial_number 主键，覆盖式更新）
//   · 多维筛选（SN / 基站名 / 产品型号 / 来源）+ 下载 + 单/批量删除
// 全部数据走 @core hooks（真实后端），三态完整。
// ============================================================

const SOURCE_LABEL: Record<SnapshotSource, string> = {
  backup: '备份生成',
  manual_upload: '手动上传',
}

const SOURCE_VARIANT: Record<SnapshotSource, 'success' | 'secondary'> = {
  backup: 'success',
  manual_upload: 'secondary',
}

function getErrMsg(e: unknown): string {
  return e instanceof Error ? e.message : '操作失败'
}

export default function ConfigSnapshotLibrary() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [serialFilter, setSerialFilter] = useState('')
  const [enbFilter, setEnbFilter] = useState('')
  // #602：产品名称下拉过滤，传 product.id。
  const [productId, setProductId] = useState('')
  const [sourceFilter, setSourceFilter] = useState<SnapshotSource | ''>('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [confirmDelete, setConfirmDelete] = useState<string[] | null>(null)
  const [toast, setToast] = useState<{ kind: 'ok' | 'err'; msg: string } | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(serialFilter ? { serialNumber: serialFilter } : {}),
      ...(enbFilter ? { enbName: enbFilter } : {}),
      ...(productId ? { productId } : {}),
      ...(sourceFilter ? { source: sourceFilter } : {}),
    }),
    [page, pageSize, serialFilter, enbFilter, productId, sourceFilter]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useConfigSnapshots(params)
  const batchDelete = useBatchDeleteConfigSnapshots()
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
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['', 'SN', '基站名', '产品名称', '快照文件', '大小', '来源', '更新时间', '操作']

  const flash = (kind: 'ok' | 'err', msg: string) => {
    setToast({ kind, msg })
    window.setTimeout(() => setToast(null), 3000)
  }

  const pageSns = useMemo(() => rows.map((r) => r.serialNumber), [rows])
  const allOnPageSelected =
    pageSns.length > 0 && pageSns.every((sn) => selected.has(sn))
  const someOnPageSelected = pageSns.some((sn) => selected.has(sn))

  const toggleOne = (sn: string) =>
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(sn)) next.delete(sn)
      else next.add(sn)
      return next
    })

  const toggleAllOnPage = () =>
    setSelected((prev) => {
      const next = new Set(prev)
      if (allOnPageSelected) pageSns.forEach((sn) => next.delete(sn))
      else pageSns.forEach((sn) => next.add(sn))
      return next
    })

  const resetFilters = () => {
    setSerialFilter('')
    setEnbFilter('')
    setProductId('')
    setSourceFilter('')
    setPage(1)
  }

  const hasFilter =
    Boolean(serialFilter || enbFilter || productId || sourceFilter)

  const handleDownload = async (sn: string) => {
    try {
      await configSnapshotApi.download(sn)
    } catch (e) {
      flash('err', `下载失败：${getErrMsg(e)}`)
    }
  }

  const handleDelete = (sns: string[]) => {
    batchDelete.mutate(sns, {
      onSuccess: (res) => {
        flash(
          'ok',
          res.failed.length
            ? `删除 ${res.succeeded.length} 项，失败 ${res.failed.length} 项`
            : `已删除 ${res.succeeded.length} 项`
        )
        setSelected((prev) => {
          const next = new Set(prev)
          sns.forEach((sn) => next.delete(sn))
          return next
        })
        setConfirmDelete(null)
      },
      onError: (e: unknown) => flash('err', `删除失败：${getErrMsg(e)}`),
    })
  }

  const selectedCount = selected.size

  return (
    <PageShell
      title="配置快照库"
      description="每设备最新一份配置快照（备份自动晋升 / 手动上传）"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-48 pl-9"
              placeholder="设备 SN"
              value={serialFilter}
              onChange={(e) => {
                setSerialFilter(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <Input
            className="w-40"
            placeholder="基站名"
            value={enbFilter}
            onChange={(e) => {
              setEnbFilter(e.target.value)
              setPage(1)
            }}
          />
          <Select
            value={productId || 'all'}
            onValueChange={(v) => {
              setProductId(v === 'all' ? '' : v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-48">
              <SelectValue placeholder="产品名称" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部产品</SelectItem>
              {productNameOptions.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select
            value={sourceFilter || 'all'}
            onValueChange={(v) => {
              setSourceFilter(v === 'all' ? '' : (v as SnapshotSource))
              setPage(1)
            }}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="来源" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部来源</SelectItem>
              <SelectItem value="backup">备份生成</SelectItem>
              <SelectItem value="manual_upload">手动上传</SelectItem>
            </SelectContent>
          </Select>
          {hasFilter ? (
            <Button variant="ghost" size="sm" onClick={resetFilters}>
              <X className="size-4" /> 重置
            </Button>
          ) : null}
          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={() => refetch()}
          >
            <RefreshCcw className="size-4" /> 刷新
          </Button>
        </div>
      }
    >
      {toast ? (
        <div
          className={cn(
            'mb-3 rounded-md px-3 py-2 text-sm',
            toast.kind === 'ok'
              ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
              : 'bg-destructive/10 text-destructive'
          )}
        >
          {toast.msg}
        </div>
      ) : null}

      {selectedCount > 0 ? (
        <div className="mb-3 flex flex-wrap items-center gap-2 rounded-lg border border-primary/30 bg-primary/5 px-4 py-2.5 text-sm">
          <span className="font-medium">已选 {selectedCount} 项</span>
          <div className="ml-auto flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              className="text-destructive"
              disabled={batchDelete.isPending}
              onClick={() => setConfirmDelete(Array.from(selected))}
            >
              <Trash2 className="size-4" /> 批量删除
            </Button>
            <Button variant="ghost" size="sm" onClick={() => setSelected(new Set())}>
              <X className="size-4" /> 取消选择
            </Button>
          </div>
        </div>
      ) : null}

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c, i) => (
                <TableHead key={c || `col-${i}`}>
                  {i === 0 ? (
                    <input
                      type="checkbox"
                      aria-label="全选本页"
                      className="size-4 cursor-pointer accent-primary"
                      checked={allOnPageSelected}
                      ref={(el) => {
                        if (el)
                          el.indeterminate =
                            !allOnPageSelected && someOnPageSelected
                      }}
                      onChange={toggleAllOnPage}
                    />
                  ) : (
                    c
                  )}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>
                {hasFilter ? '没有匹配的快照' : '暂无配置快照'}
              </EmptyRow>
            ) : (
              rows.map((r: ConfigSnapshot) => (
                <TableRow
                  key={r.serialNumber}
                  data-state={selected.has(r.serialNumber) ? 'selected' : undefined}
                >
                  <TableCell>
                    <input
                      type="checkbox"
                      aria-label={`选择 ${r.serialNumber}`}
                      className="size-4 cursor-pointer accent-primary"
                      checked={selected.has(r.serialNumber)}
                      onChange={() => toggleOne(r.serialNumber)}
                    />
                  </TableCell>
                  <TableCell className="font-mono text-xs">{r.serialNumber}</TableCell>
                  <TableCell className="text-xs">{r.enbName || '—'}</TableCell>
                  <TableCell className="text-xs">
                    {(() => {
                      const display = resolveProductName(r.productType)
                      return display ? display : '—'
                    })()}
                  </TableCell>
                  <TableCell
                    className="max-w-[16rem] truncate font-mono text-xs"
                    title={r.fileName}
                  >
                    {r.fileName}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatBytes(r.fileSize)}
                  </TableCell>
                  <TableCell>
                    <Badge variant={SOURCE_VARIANT[r.source]}>
                      {SOURCE_LABEL[r.source] ?? r.source}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(r.updateTime)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => void handleDownload(r.serialNumber)}
                      >
                        <Download className="size-4" /> 下载
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive"
                        onClick={() => setConfirmDelete([r.serialNumber])}
                      >
                        <Trash2 className="size-4" /> 删除
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={pageSize}
        onChange={setPage}
      />

      {confirmDelete ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div
            className="absolute inset-0 bg-black/40"
            onClick={() => setConfirmDelete(null)}
            aria-hidden
          />
          <div className="relative z-10 w-full max-w-sm rounded-xl border bg-card shadow-lg">
            <div className="px-5 py-4">
              <h2 className="text-base font-semibold">删除配置快照</h2>
              <p className="mt-2 text-sm text-muted-foreground">
                确认删除选中的 {confirmDelete.length} 份配置快照？此操作不可恢复。
              </p>
            </div>
            <div className="flex justify-end gap-2 border-t px-5 py-3">
              <Button variant="outline" size="sm" onClick={() => setConfirmDelete(null)}>
                取消
              </Button>
              <Button
                variant="destructive"
                size="sm"
                disabled={batchDelete.isPending}
                onClick={() => handleDelete(confirmDelete)}
              >
                {batchDelete.isPending ? '处理中…' : '确认删除'}
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </PageShell>
  )
}
