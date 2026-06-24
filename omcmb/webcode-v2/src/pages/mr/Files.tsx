import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Search, Trash2 } from 'lucide-react'

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
  formatTime,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
} from '@/components/layout/PageShell'

import { useMRFileDevices, useBatchDeleteMRFiles } from '@core/hooks/api/useMR'
import { useProductList, useProductNameResolver } from '@core/hooks/api/useProducts'

// ============================================================
// 测量报告 → 采集文件（按设备聚合）
// 对照 v1 webcode/src/pages/mr/Files —— 按设备聚合的 MR 文件主列表。
// 数据走真实 useMRFileDevices（GET /mr/files/devices，10s 轮询）
// + useBatchDeleteMRFiles（按 SN 批量删除）。
// 设备 SN 单元格链接到该设备的文件明细子路由 /mr/files/:sn。
// ============================================================

const PAGE_SIZE = 20

export default function Files() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [siteName, setSiteName] = useState('')
  // #602：产品名称下拉过滤，传 product.id
  const [productId, setProductId] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(siteName.trim() ? { siteName: siteName.trim() } : {}),
      ...(productId ? { productId } : {}),
    }),
    [page, keyword, siteName, productId],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useMRFileDevices(params)
  const batchDelete = useBatchDeleteMRFiles()
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

  const toggle = (sn: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(sn)) next.delete(sn)
      else next.add(sn)
      return next
    })
  }
  const toggleAll = () => {
    setSelected((prev) => {
      if (prev.size === rows.length && rows.length > 0) return new Set()
      return new Set(rows.map((r) => r.deviceSn))
    })
  }

  const selectedList = Array.from(selected)

  const handleBatchDelete = () => {
    if (selectedList.length === 0) return
    if (!window.confirm(`确认删除选中 ${selectedList.length} 个设备的全部 MR 文件？删除后不可恢复。`)) return
    batchDelete.mutate(selectedList, {
      onSuccess: () => setSelected(new Set()),
    })
  }

  const allChecked = rows.length > 0 && selected.size === rows.length
  const cols = ['', '设备 SN', '基站名称', '产品名称', '首次采集', '最近采集', '文件数', '上报状态']

  return (
    <PageShell
      title="MR 采集文件"
      description="按设备聚合的 MRO/MRS/MRE 文件采集视图"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-44 pl-9"
              placeholder="搜索设备 SN"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <Input
            className="w-40"
            placeholder="基站名称"
            value={siteName}
            onChange={(e) => {
              setSiteName(e.target.value)
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
          <Button
            variant="outline"
            size="sm"
            disabled={selectedList.length === 0 || batchDelete.isPending}
            onClick={handleBatchDelete}
          >
            <Trash2 className="size-4 text-destructive" /> 批量删除（{selectedList.length}）
          </Button>
          <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
            <RefreshCcw className={isFetching ? 'animate-spin' : ''} /> 刷新
          </Button>
        </div>
      }
    >
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c, idx) =>
                idx === 0 ? (
                  <TableHead key="sel" className="w-10">
                    <input
                      type="checkbox"
                      aria-label="全选"
                      checked={allChecked}
                      onChange={toggleAll}
                      className="size-4 cursor-pointer"
                    />
                  </TableHead>
                ) : (
                  <TableHead key={c}>{c}</TableHead>
                ),
              )}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无 MR 采集文件</EmptyRow>
            ) : (
              rows.map((d) => (
                <TableRow key={d.deviceSn}>
                  <TableCell className="w-10">
                    <input
                      type="checkbox"
                      aria-label={`选择 ${d.deviceSn}`}
                      checked={selected.has(d.deviceSn)}
                      onChange={() => toggle(d.deviceSn)}
                      className="size-4 cursor-pointer"
                    />
                  </TableCell>
                  <TableCell>
                    <button
                      type="button"
                      className="font-mono text-xs text-primary hover:underline"
                      onClick={() => navigate(`/mr/files`)}
                    >
                      {d.deviceSn}
                    </button>
                  </TableCell>
                  <TableCell className="text-xs">{d.siteName || '—'}</TableCell>
                  <TableCell className="text-xs">
                    {(() => {
                      const display = resolveProductName(d.productClass)
                      return display ? display : '—'
                    })()}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(d.firstCollectTime)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(d.lastCollectTime)}
                  </TableCell>
                  <TableCell className="tabular-nums">{d.fileCount.toLocaleString()}</TableCell>
                  <TableCell>
                    {d.reporting ? (
                      <Badge variant="default">上报中</Badge>
                    ) : (
                      <Badge variant="muted">已停止</Badge>
                    )}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </PageShell>
  )
}
