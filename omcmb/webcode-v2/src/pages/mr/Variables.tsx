import { useMemo, useState } from 'react'
import { RefreshCcw, Search, SlidersHorizontal } from 'lucide-react'

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
  TableCard,
} from '@/components/layout/PageShell'

import { useAllMRIndicators } from '@core/hooks/api/useMR'

// ============================================================
// 测量报告 → 测量变量
// 对照 v1 webcode/src/pages/mr/Variables —— MR 测量变量（取值范围/单位/类别）。
// v1 为本地 mock 编辑表；v2 改走真实 useAllMRIndicators（GET /mr/indicators/all），
// 把每个指标的取值范围、单位、类别作为可观测的测量变量来源，避免硬编码假数据。
// ============================================================

// 由取值范围推断变量类型（整型范围 → integer，含小数 → float）。
function inferVarType(min: number, max: number): { key: string; label: string } {
  const isInt = Number.isInteger(min) && Number.isInteger(max)
  return isInt ? { key: 'integer', label: '整型' } : { key: 'float', label: '浮点' }
}

const VAR_TYPE_VARIANT: Record<string, 'default' | 'secondary' | 'outline'> = {
  integer: 'default',
  float: 'secondary',
}

export default function Variables() {
  const [keyword, setKeyword] = useState('')
  const [category, setCategory] = useState('')

  const { data, isLoading, isError, error, isFetching, refetch } = useAllMRIndicators()

  const allRows = data ?? []

  const categories = useMemo(() => {
    const set = new Set<string>()
    for (const r of allRows) if (r.category) set.add(r.category)
    return Array.from(set).sort()
  }, [allRows])

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

  const cols = ['变量名', '类型', '取值范围', '单位', '类别', '描述']

  return (
    <PageShell
      title="MR 测量变量"
      description="MR 测量变量取值范围与类型定义（源自指标库）"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-72 pl-9"
              placeholder="搜索变量名 / 编码"
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
            <SlidersHorizontal className="mr-1 inline size-4" />共 {rows.length} 个变量
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
              <EmptyRow colSpan={cols.length}>暂无测量变量</EmptyRow>
            ) : (
              rows.map((v) => {
                const vt = inferVarType(v.valueRange[0], v.valueRange[1])
                return (
                  <TableRow key={v.id}>
                    <TableCell className="font-mono text-xs font-medium">{v.indicatorCode}</TableCell>
                    <TableCell>
                      <Badge variant={VAR_TYPE_VARIANT[vt.key] ?? 'outline'}>{vt.label}</Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      [{v.valueRange[0]}, {v.valueRange[1]}]
                    </TableCell>
                    <TableCell className="text-xs">{v.unit || '—'}</TableCell>
                    <TableCell className="text-xs">{v.category || '—'}</TableCell>
                    <TableCell className="max-w-md truncate text-xs text-muted-foreground" title={v.description}>
                      {v.description || '—'}
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>
    </PageShell>
  )
}
