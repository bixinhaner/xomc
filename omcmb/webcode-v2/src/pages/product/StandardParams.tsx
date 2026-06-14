import { useMemo, useState } from 'react'
import { Search, X } from 'lucide-react'

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
} from '@/components/layout/PageShell'

import { useStandardParams } from '@core/hooks/api/useParamModels'
import type { StandardParam, StandardParamFilter } from '@core/types/paramModel'

// ============================================================
// 标准参数树 — 对照 v1 webcode/src/pages/product/standard-params。
// 后端按 keyword/entryType 服务端过滤,本页再做客户端分页切片。
// ============================================================

type EntryFilter = 'all' | 'object' | 'parameter'
const PAGE_SIZE = 50

export function StandardParamsPage() {
  const [keyword, setKeyword] = useState('')
  const [entryType, setEntryType] = useState<EntryFilter>('all')
  const [page, setPage] = useState(1)

  const filter = useMemo<StandardParamFilter>(
    () => ({
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(entryType !== 'all' ? { entryType } : {}),
    }),
    [keyword, entryType]
  )

  const { data, isLoading, isError, error, isFetching } = useStandardParams(filter)

  const items = data?.items ?? []
  const total = items.length
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const pageRows = items.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)
  const colCount = 7

  const hasFilter = Boolean(keyword.trim()) || entryType !== 'all'

  return (
    <PageShell
      title="标准参数树"
      description={`共 ${total} 条标准参数`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-80 pl-9"
              placeholder="搜索标准路径"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <Select
            value={entryType}
            onValueChange={(v) => {
              setEntryType(v as EntryFilter)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="条目类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部类型</SelectItem>
              <SelectItem value="parameter">parameter</SelectItem>
              <SelectItem value="object">object</SelectItem>
            </SelectContent>
          </Select>
          {hasFilter && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setKeyword('')
                setEntryType('all')
                setPage(1)
              }}
            >
              <X className="size-4" /> 重置
            </Button>
          )}
        </div>
      }
    >
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>标准路径</TableHead>
              <TableHead className="w-28">条目类型</TableHead>
              <TableHead className="w-32">访问</TableHead>
              <TableHead className="w-28">数据类型</TableHead>
              <TableHead className="w-28">生效方式</TableHead>
              <TableHead className="w-24 text-right">最小值</TableHead>
              <TableHead className="w-24 text-right">最大值</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={error} />
            ) : pageRows.length === 0 ? (
              <EmptyRow colSpan={colCount}>
                {hasFilter ? '没有匹配的标准参数' : '暂无标准参数'}
              </EmptyRow>
            ) : (
              pageRows.map((row) => <StdRow key={row.standardPath} row={row} />)
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </PageShell>
  )
}

function StdRow({ row }: { row: StandardParam }) {
  return (
    <TableRow>
      <TableCell className="max-w-[420px] truncate font-mono text-xs" title={row.standardPath}>
        {row.standardPath}
      </TableCell>
      <TableCell>
        <Badge variant="outline">{row.entryType}</Badge>
      </TableCell>
      <TableCell className="text-xs text-muted-foreground">{row.access || '—'}</TableCell>
      <TableCell className="text-xs text-muted-foreground">{row.dataType || '—'}</TableCell>
      <TableCell className="text-xs text-muted-foreground">{row.changeApplies || '—'}</TableCell>
      <TableCell className="text-right text-xs tabular-nums text-muted-foreground">
        {row.minValue ?? '—'}
      </TableCell>
      <TableCell className="text-right text-xs tabular-nums text-muted-foreground">
        {row.maxValue ?? '—'}
      </TableCell>
    </TableRow>
  )
}

export default StandardParamsPage
