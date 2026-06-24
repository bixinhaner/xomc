import { useMemo, useState } from 'react'
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
import { cn } from '@/lib/utils'

import {
  usePMFileDevices,
  useBatchDeletePMFiles,
} from '@core/hooks/api/usePerformance'
import { useProductList, useProductNameResolver } from '@core/hooks/api/useProducts'
import type { PageRequest } from '@core/types/pagination'

// ============================================================
// PM 文件 — 对齐 v1 /performance/files
//   v1 旧页是 mock；v2 改对接真 API：按设备聚合的 PM 文件视图（usePMFileDevices）。
//   支持按 SN 批量删除（按设备）。
// ============================================================

const PAGE_SIZE = 20

function Stat({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: number | string
  tone?: 'default' | 'emerald' | 'amber' | 'muted'
}) {
  const toneClass = {
    default: 'text-foreground',
    emerald: 'text-emerald-600 dark:text-emerald-400',
    amber: 'text-amber-600 dark:text-amber-400',
    muted: 'text-muted-foreground',
  }[tone]
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>{value}</div>
    </div>
  )
}

export function PerformanceFilesPage() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [input, setInput] = useState('')
  // #602：产品名称下拉过滤，传 product.id
  const [productId, setProductId] = useState('')
  const [pendingSn, setPendingSn] = useState<string | null>(null)

  const params = useMemo<
    { keyword?: string; siteName?: string; productClass?: string; productId?: string } & PageRequest
  >(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(productId ? { productId } : {}),
    }),
    [page, keyword, productId]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = usePMFileDevices(params)
  const batchDelete = useBatchDeletePMFiles()
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

  const reportingCount = useMemo(() => rows.filter((r) => r.reporting).length, [rows])
  const totalFiles = useMemo(() => rows.reduce((sum, r) => sum + r.fileCount, 0), [rows])

  const submitSearch = () => {
    setKeyword(input)
    setPage(1)
  }

  const handleDelete = (sn: string) => {
    setPendingSn(sn)
    batchDelete.mutate([sn], { onSettled: () => setPendingSn(null) })
  }

  const cols = ['设备 SN', '基站名称', '产品名称', '文件数', '首次采集', '最近采集', '上报', '操作']

  return (
    <PageShell
      title="PM 文件"
      description="按基站聚合的 PM 性能文件采集情况（10s 自动刷新）"
      isFetching={isFetching}
    >
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="本页基站" value={rows.length} />
        <Stat label="基站总数" value={total} tone="emerald" />
        <Stat label="本页文件数" value={totalFiles} tone="amber" />
        <Stat label="本页上报中" value={reportingCount} tone="muted" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-72 pl-9"
            placeholder="SN / 基站名称"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') submitSearch()
            }}
          />
        </div>
        <Button variant="outline" size="sm" onClick={submitSearch}>
          搜索
        </Button>
        <Select
          value={productId || 'all'}
          onValueChange={(v) => {
            setProductId(v === 'all' ? '' : v)
            setPage(1)
          }}
        >
          <SelectTrigger className="w-56">
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
          className="ml-auto"
          disabled={isFetching || batchDelete.isPending}
          onClick={() => void refetch()}
        >
          <RefreshCcw className="size-4" /> 刷新
        </Button>
      </div>

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
              <EmptyRow colSpan={cols.length}>暂无 PM 文件</EmptyRow>
            ) : (
              rows.map((r) => {
                const busy = pendingSn === r.deviceSn
                return (
                  <TableRow key={r.deviceSn} className={cn(busy && 'opacity-50')}>
                    <TableCell className="font-mono text-xs">{r.deviceSn}</TableCell>
                    <TableCell className="font-medium">{r.siteName || '—'}</TableCell>
                    <TableCell className="text-xs">
                      {(() => {
                        const name = resolveProductName(r.productClass)
                        if (!name) return '—'
                        return <Badge variant="outline">{name}</Badge>
                      })()}
                    </TableCell>
                    <TableCell className="tabular-nums">{r.fileCount}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(r.firstCollectTime)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(r.lastCollectTime)}
                    </TableCell>
                    <TableCell>
                      <Badge variant={r.reporting ? 'success' : 'muted'}>
                        {r.reporting ? '上报中' : '空闲'}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={busy}
                        className="text-destructive hover:text-destructive"
                        onClick={() => handleDelete(r.deviceSn)}
                        aria-label="删除该设备 PM 文件"
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </PageShell>
  )
}

export default PerformanceFilesPage
