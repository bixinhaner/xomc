import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Search } from 'lucide-react'

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

import { useParamMappings } from '@core/hooks/api/useParamModels'
import type { ParamMapping } from '@core/types/paramModel'

// ============================================================
// 参数模型映射详情 — 对照 v1 param-model MappingsTab。
// 路由 /product/param-model/:name,经 useParamMappings 真实加载。
// 标准路径 ↔ 私有路径双向翻译条目 + 类型/访问/数据类型筛选。
// ============================================================

type EntryFilter = 'all' | 'object' | 'parameter'
const PAGE_SIZE = 50

export function ParamModelMappingsPage() {
  const navigate = useNavigate()
  const { name: rawName } = useParams<{ name: string }>()
  const name = rawName ? decodeURIComponent(rawName) : undefined

  const [keyword, setKeyword] = useState('')
  const [entryType, setEntryType] = useState<EntryFilter>('all')
  const [page, setPage] = useState(1)

  const { data, isLoading, isError, error, isFetching } = useParamMappings(name)

  const filtered = useMemo(() => {
    const raw = data?.items ?? []
    const k = keyword.trim().toLowerCase()
    return raw.filter((m) => {
      if (entryType !== 'all' && m.entryType !== entryType) return false
      if (!k) return true
      return `${m.standardPath} ${m.privatePath}`.toLowerCase().includes(k)
    })
  }, [data, keyword, entryType])

  const total = filtered.length
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const pageRows = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)
  const colCount = 7

  return (
    <PageShell
      title={`参数映射 · ${name ?? ''}`}
      description={`共 ${total} 条映射`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => navigate('/product/param-model')}>
            <ArrowLeft className="size-4" /> 返回
          </Button>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-80 pl-9"
              placeholder="搜索标准路径 / 私有路径"
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
        </div>
      }
    >
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>标准路径</TableHead>
              <TableHead>私有路径</TableHead>
              <TableHead className="w-24">类型</TableHead>
              <TableHead className="w-28">访问</TableHead>
              <TableHead className="w-28">数据类型</TableHead>
              <TableHead className="w-24">生效方式</TableHead>
              <TableHead className="w-24">状态</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={error} />
            ) : pageRows.length === 0 ? (
              <EmptyRow colSpan={colCount}>
                {keyword.trim() || entryType !== 'all' ? '没有匹配的映射' : '暂无映射'}
              </EmptyRow>
            ) : (
              pageRows.map((row) => <MappingRow key={row.id} row={row} />)
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </PageShell>
  )
}

function MappingRow({ row }: { row: ParamMapping }) {
  return (
    <TableRow>
      <TableCell className="max-w-[320px] truncate font-mono text-xs" title={row.standardPath}>
        {row.standardPath}
      </TableCell>
      <TableCell className="max-w-[320px] truncate font-mono text-xs text-muted-foreground" title={row.privatePath}>
        {row.privatePath}
      </TableCell>
      <TableCell>
        <Badge variant="outline">{row.entryType}</Badge>
      </TableCell>
      <TableCell className="text-xs text-muted-foreground">{row.access || '—'}</TableCell>
      <TableCell className="text-xs text-muted-foreground">{row.dataType || '—'}</TableCell>
      <TableCell className="text-xs text-muted-foreground">{row.changeApplies || '—'}</TableCell>
      <TableCell>
        {row.isActive ? (
          <Badge variant="success">启用</Badge>
        ) : (
          <Badge variant="muted">停用</Badge>
        )}
      </TableCell>
    </TableRow>
  )
}

export default ParamModelMappingsPage
