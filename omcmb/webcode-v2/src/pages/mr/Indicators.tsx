import { useMemo, useState } from 'react'
import { Activity, RefreshCcw, Search } from 'lucide-react'

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

import { useMRIndicators } from '@core/hooks/api/useMR'

// ============================================================
// 测量报告 → 指标库（MRO/MRS/MRE 指标定义）
// 对照 v1 webcode/src/pages/mr/Indicators —— 编码/名称/类别/单位/取值范围/描述。
// 数据走真实 useMRIndicators（GET /mr/indicators），客户端按关键字/类别筛选。
// ============================================================

const PAGE_SIZE = 20

export default function Indicators() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [category, setCategory] = useState('')

  const { data, isLoading, isError, error, isFetching, refetch } = useMRIndicators({
    page,
    pageSize: PAGE_SIZE,
  })

  const allRows = data?.items ?? []
  const total = data?.total ?? 0

  // 后端分页返回当前页；类别 / 关键字在当前页客户端过滤（与 v1 一致的轻量筛选）。
  const rows = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    return allRows.filter((r) => {
      if (kw && !r.indicatorName.toLowerCase().includes(kw) && !r.indicatorCode.toLowerCase().includes(kw)) {
        return false
      }
      if (category && r.category !== category) return false
      return true
    })
  }, [allRows, keyword, category])

  // 当前页所有类别（去重）供筛选下拉。
  const categories = useMemo(() => {
    const set = new Set<string>()
    for (const r of allRows) if (r.category) set.add(r.category)
    return Array.from(set).sort()
  }, [allRows])

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const cols = ['编码', '名称', '类别', '单位', '取值范围', '描述']

  return (
    <PageShell
      title="MR 指标库"
      description="MRO / MRS / MRE 测量指标定义（编码、名称、类别、单位、取值范围）"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-72 pl-9"
              placeholder="搜索指标名称 / 编码"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <select
            className="h-9 rounded-md border bg-background px-3 text-sm"
            value={category}
            onChange={(e) => setCategory(e.target.value)}
          >
            <option value="">全部类别</option>
            {categories.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
          <span className="text-sm text-muted-foreground">
            <Activity className="mr-1 inline size-4" />共 {total} 项指标
          </span>
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
              <EmptyRow colSpan={cols.length}>暂无 MR 指标</EmptyRow>
            ) : (
              rows.map((i) => (
                <TableRow key={i.id}>
                  <TableCell className="font-mono text-xs">{i.indicatorCode}</TableCell>
                  <TableCell className="font-medium">{i.indicatorName}</TableCell>
                  <TableCell>
                    {i.category ? <Badge variant="outline">{i.category}</Badge> : '—'}
                  </TableCell>
                  <TableCell className="text-xs">{i.unit || '—'}</TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    [{i.valueRange[0]}, {i.valueRange[1]}]
                  </TableCell>
                  <TableCell className="max-w-md truncate text-xs text-muted-foreground" title={i.description}>
                    {i.description || '—'}
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
