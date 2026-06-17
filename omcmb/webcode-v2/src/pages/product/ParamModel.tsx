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

import { useParamModelList } from '@core/hooks/api/useParamModels'
import type { ParamModel } from '@core/types/paramModel'

// ============================================================
// 参数模型库清单 — 对照 v1 webcode/src/pages/product/param-model ModelsTab。
// 隐藏 source=unknown 的孤儿模型;点模型名进映射详情。
// ============================================================

const PAGE_SIZE = 20

export function ParamModelPage() {
  const navigate = useNavigate()
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)

  const { data, isLoading, isError, error, isFetching, refetch } = useParamModelList()

  const items = useMemo(() => {
    const raw = (data?.items ?? []).filter((m) => (m.source ?? 'unknown') !== 'unknown')
    const k = keyword.trim().toLowerCase()
    if (!k) return raw
    return raw.filter((m) => `${m.name} ${m.description ?? ''}`.toLowerCase().includes(k))
  }, [data, keyword])

  const total = items.length
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const pageRows = items.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)
  const colCount = 7

  return (
    <PageShell
      title="参数模型库"
      description={`共 ${total} 个参数模型`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-80 pl-9"
              placeholder="搜索模型名 / 描述"
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
            <Button variant="outline" size="sm" onClick={() => void refetch()}>
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
              <TableHead>名称</TableHead>
              <TableHead>加载源</TableHead>
              <TableHead className="w-24 text-right">条目数</TableHead>
              <TableHead className="w-24 text-right">对象数</TableHead>
              <TableHead className="w-24 text-right">参数数</TableHead>
              <TableHead className="w-24">状态</TableHead>
              <TableHead>描述</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={error} />
            ) : pageRows.length === 0 ? (
              <EmptyRow colSpan={colCount}>
                {keyword.trim() ? '没有匹配的参数模型' : '暂无参数模型'}
              </EmptyRow>
            ) : (
              pageRows.map((row) => <ModelRow key={row.id} row={row} navigate={navigate} />)
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </PageShell>
  )
}

function ModelRow({
  row,
  navigate,
}: {
  row: ParamModel
  navigate: ReturnType<typeof useNavigate>
}) {
  return (
    <TableRow>
      <TableCell>
        <button
          type="button"
          className="font-medium text-primary hover:underline"
          onClick={() => navigate(`/product/param-model`)}
        >
          {row.name}
        </button>
      </TableCell>
      <TableCell
        className="max-w-[260px] truncate font-mono text-xs text-muted-foreground"
        title={row.loadedFrom}
      >
        {row.loadedFrom || '—'}
      </TableCell>
      <TableCell className="text-right tabular-nums">{row.totalEntries}</TableCell>
      <TableCell className="text-right tabular-nums">{row.totalObjects}</TableCell>
      <TableCell className="text-right tabular-nums">{row.totalParams}</TableCell>
      <TableCell>
        {row.isActive ? (
          <Badge variant="success">启用</Badge>
        ) : (
          <Badge variant="muted">停用</Badge>
        )}
      </TableCell>
      <TableCell className="max-w-[240px] truncate text-sm text-muted-foreground" title={row.description}>
        {row.description || '—'}
      </TableCell>
    </TableRow>
  )
}

export default ParamModelPage
