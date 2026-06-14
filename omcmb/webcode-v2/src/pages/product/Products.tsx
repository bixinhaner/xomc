import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Search, X } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
} from '@/components/layout/PageShell'

import { useProductList, useMatchOrder } from '@core/hooks/api/useProducts'
import type { Product, ProductListFilter } from '@core/types/product'

// ============================================================
// 产品装配件清单 — 对照 v1 webcode/src/pages/product/products
// 按匹配顺序(MatchOrder 最小 sortOrder)排序;点产品名进详情。
// ============================================================

interface PatternStat {
  minSortOrder: number
  total: number
  active: number
}

const SORT_TAIL = Number.MAX_SAFE_INTEGER
const PAGE_SIZE = 20

export function ProductsPage() {
  const navigate = useNavigate()
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)

  const filter = useMemo<ProductListFilter>(
    () => (keyword.trim() ? { keyword: keyword.trim() } : {}),
    [keyword]
  )
  const { data, isLoading, isError, error, isFetching } = useProductList(filter)
  const matchOrderQuery = useMatchOrder()

  const items = useMemo(() => data?.items ?? [], [data])

  const patternStats = useMemo(() => {
    const stats = new Map<string, PatternStat>()
    for (const row of matchOrderQuery.data?.items ?? []) {
      const cur = stats.get(row.productId)
      if (!cur) {
        stats.set(row.productId, {
          minSortOrder: row.sortOrder,
          total: 1,
          active: row.isActive ? 1 : 0,
        })
      } else {
        cur.minSortOrder = Math.min(cur.minSortOrder, row.sortOrder)
        cur.total += 1
        if (row.isActive) cur.active += 1
      }
    }
    return stats
  }, [matchOrderQuery.data])

  const sortedItems = useMemo(
    () =>
      [...items].sort((a, b) => {
        const sa = patternStats.get(a.id)?.minSortOrder ?? SORT_TAIL
        const sb = patternStats.get(b.id)?.minSortOrder ?? SORT_TAIL
        return sa - sb
      }),
    [items, patternStats]
  )

  const total = sortedItems.length
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const pageRows = sortedItems.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)
  const colCount = 8

  return (
    <PageShell
      title="产品装配件"
      description={`共 ${total} 个产品 · 按匹配顺序排序`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-80 pl-9"
              placeholder="搜索产品名 / 厂商 / 制式"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          {keyword.trim() && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setKeyword('')
                setPage(1)
              }}
            >
              <X className="size-4" /> 重置
            </Button>
          )}
          <div className="ml-auto">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                void matchOrderQuery.refetch()
              }}
            >
              <RefreshCcw className="size-4" /> 刷新
            </Button>
          </div>
        </div>
      }
    >
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>产品名称</TableHead>
              <TableHead>厂商</TableHead>
              <TableHead>制式</TableHead>
              <TableHead>参数模型库</TableHead>
              <TableHead>指标平台</TableHead>
              <TableHead>告警网元类型</TableHead>
              <TableHead>匹配正则</TableHead>
              <TableHead>未知告警</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={error} />
            ) : pageRows.length === 0 ? (
              <EmptyRow colSpan={colCount}>
                {keyword.trim() ? '没有匹配的产品' : '暂无产品'}
              </EmptyRow>
            ) : (
              pageRows.map((row) => <ProductRow key={row.id} row={row} navigate={navigate} />)
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </PageShell>
  )
}

function ProductRow({
  row,
  navigate,
}: {
  row: Product
  navigate: ReturnType<typeof useNavigate>
}) {
  const patterns = row.patterns ?? []
  const visible = patterns.slice(0, 3)
  const rest = patterns.length - visible.length

  return (
    <TableRow>
      <TableCell>
        <button
          type="button"
          className="font-medium text-primary hover:underline"
          onClick={() => navigate(`/product/products/${row.id}`)}
        >
          {row.name}
        </button>
        {row.isBuiltin && (
          <Badge variant="muted" className="ml-2">
            内置
          </Badge>
        )}
      </TableCell>
      <TableCell className="text-sm text-muted-foreground">{row.vendor || '—'}</TableCell>
      <TableCell>
        <Badge variant="outline">{(row.tech || '—').toUpperCase()}</Badge>
      </TableCell>
      <TableCell className="text-sm text-muted-foreground">{row.paramModelName || '—'}</TableCell>
      <TableCell className="text-sm text-muted-foreground">{row.indicatorPlatform || '—'}</TableCell>
      <TableCell className="text-sm text-muted-foreground">{row.alarmNeType || '—'}</TableCell>
      <TableCell>
        {patterns.length === 0 ? (
          <span className="text-muted-foreground">—</span>
        ) : (
          <div className="flex flex-wrap gap-1">
            {visible.map((p) => (
              <Badge key={p} variant="outline" className="font-mono text-[11px]">
                {p}
              </Badge>
            ))}
            {rest > 0 && (
              <Badge variant="muted" title={patterns.slice(3).join('\n')}>
                +{rest}
              </Badge>
            )}
          </div>
        )}
      </TableCell>
      <TableCell>
        {row.enableUnknownAlarm ? (
          <Badge variant="warning">接收</Badge>
        ) : (
          <Badge variant="muted">丢弃</Badge>
        )}
      </TableCell>
    </TableRow>
  )
}

export default ProductsPage
