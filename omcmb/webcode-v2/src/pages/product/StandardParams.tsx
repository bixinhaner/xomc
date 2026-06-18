import { useMemo, useState } from 'react'
import { Pencil, Plus, Search, Trash2, X } from 'lucide-react'

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

import {
  useStandardParams,
  useUpsertStandard,
  useDeleteStandard,
} from '@core/hooks/api/useParamModels'
import type {
  StandardParam,
  StandardParamFilter,
  UpsertStandardInput,
} from '@core/types/paramModel'

import { StandardParamDialog } from './StandardParamDialog'

// ============================================================
// 标准参数树 (v2) — 对照 v1 webcode/src/pages/product/standard-params。
// 后端按 keyword/entryType 服务端过滤,本页再做客户端分页切片。
// ISSUE-488: 补齐新增 / 编辑 / 删除表单,业务深度对齐 v1（弹窗见 StandardParamDialog）。
// ============================================================

type EntryFilter = 'all' | 'object' | 'parameter'
const PAGE_SIZE = 50

export function StandardParamsPage() {
  const [keyword, setKeyword] = useState('')
  const [entryType, setEntryType] = useState<EntryFilter>('all')
  const [page, setPage] = useState(1)

  // 弹窗状态：creating=新增；editing=编辑某行；null+false=关闭。
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<StandardParam | null>(null)
  const [toast, setToast] = useState<{ kind: 'ok' | 'err'; msg: string } | null>(null)

  const filter = useMemo<StandardParamFilter>(
    () => ({
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(entryType !== 'all' ? { entryType } : {}),
    }),
    [keyword, entryType]
  )

  const { data, isLoading, isError, error, isFetching } = useStandardParams(filter)
  const upsertMut = useUpsertStandard()
  const deleteMut = useDeleteStandard()

  const items = data?.items ?? []
  const total = items.length
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const pageRows = items.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)
  const colCount = 8

  const hasFilter = Boolean(keyword.trim()) || entryType !== 'all'

  const dialogOpen = creating || editing !== null

  const closeDialog = () => {
    setCreating(false)
    setEditing(null)
  }

  const handleSubmit = (value: UpsertStandardInput, path?: string) => {
    setToast(null)
    upsertMut.mutate(
      { input: value, path },
      {
        onSuccess: () => {
          setToast({ kind: 'ok', msg: path ? '已保存' : '已创建' })
          closeDialog()
        },
        onError: (e) =>
          setToast({ kind: 'err', msg: e instanceof Error ? e.message : '保存失败' }),
      }
    )
  }

  const handleDelete = (row: StandardParam) => {
    if (!window.confirm(`确认删除标准参数「${row.standardPath}」？此操作不可撤销。`)) return
    setToast(null)
    deleteMut.mutate(row.standardPath, {
      onSuccess: () => setToast({ kind: 'ok', msg: '已删除' }),
      onError: (e) =>
        setToast({ kind: 'err', msg: e instanceof Error ? e.message : '删除失败' }),
    })
  }

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
          <div className="ml-auto">
            <Button
              size="sm"
              onClick={() => {
                setEditing(null)
                setCreating(true)
              }}
            >
              <Plus className="size-4" /> 新增
            </Button>
          </div>
        </div>
      }
    >
      {toast ? (
        <div
          className={
            'mb-3 rounded-md border px-3 py-2 text-sm ' +
            (toast.kind === 'ok'
              ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-600'
              : 'border-destructive/40 bg-destructive/10 text-destructive')
          }
        >
          {toast.msg}
        </div>
      ) : null}

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
              <TableHead className="w-28 text-right">操作</TableHead>
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
              pageRows.map((row) => (
                <StdRow
                  key={row.standardPath}
                  row={row}
                  onEdit={() => {
                    setCreating(false)
                    setEditing(row)
                  }}
                  onDelete={() => handleDelete(row)}
                />
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />

      <StandardParamDialog
        open={dialogOpen}
        editing={editing}
        loading={upsertMut.isPending}
        onSubmit={handleSubmit}
        onCancel={closeDialog}
      />
    </PageShell>
  )
}

function StdRow({
  row,
  onEdit,
  onDelete,
}: {
  row: StandardParam
  onEdit: () => void
  onDelete: () => void
}) {
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
      <TableCell className="text-right">
        <div className="flex justify-end gap-1">
          <Button variant="ghost" size="sm" onClick={onEdit} aria-label="编辑">
            <Pencil className="size-4" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="text-destructive hover:text-destructive"
            onClick={onDelete}
            aria-label="删除"
          >
            <Trash2 className="size-4" />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  )
}

export default StandardParamsPage
