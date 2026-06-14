import { useMemo, useState } from 'react'
import { Loader2, RefreshCcw, RefreshCw, Search, X } from 'lucide-react'

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
  useApiEndpoints,
  useApiGroups,
  useSyncApiEndpoints,
} from '@core/hooks/api/useSystem'

// ============================================================
// 系统管理 / API 管理 — 对齐 v1 webcode/src/pages/system/ApiManagement
// 真实数据 useApiEndpoints / useApiGroups（adminApi）。可一键从路由表同步端点。
// ============================================================

const PAGE_SIZE = 20
const METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH']

const METHOD_VARIANT: Record<
  string,
  'default' | 'success' | 'warning' | 'destructive' | 'secondary'
> = {
  GET: 'default',
  POST: 'success',
  PUT: 'warning',
  DELETE: 'destructive',
  PATCH: 'secondary',
}

export default function ApiManagement() {
  const [page, setPage] = useState(1)
  const [path, setPath] = useState('')
  const [method, setMethod] = useState<string>('all')
  const [apiGroup, setApiGroup] = useState<string>('all')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(path.trim() ? { path: path.trim() } : {}),
      ...(method !== 'all' ? { method } : {}),
      ...(apiGroup !== 'all' ? { apiGroup } : {}),
    }),
    [page, path, method, apiGroup]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useApiEndpoints(params)
  const groupsQuery = useApiGroups()
  const syncMutation = useSyncApiEndpoints()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const groups = groupsQuery.data ?? []

  const hasFilter =
    Boolean(path.trim()) || method !== 'all' || apiGroup !== 'all'

  function resetFilters() {
    setPath('')
    setMethod('all')
    setApiGroup('all')
    setPage(1)
  }

  const cols = ['路径', '方法', '名称', '分组', '模块', '描述']

  return (
    <PageShell
      title="API 管理"
      description={`后端接口端点 · 共 ${total} 个`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-64 pl-9"
              placeholder="搜索路径"
              value={path}
              onChange={(e) => {
                setPath(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <Select
            value={method}
            onValueChange={(v) => {
              setMethod(v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="方法" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部方法</SelectItem>
              {METHODS.map((m) => (
                <SelectItem key={m} value={m}>
                  {m}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select
            value={apiGroup}
            onValueChange={(v) => {
              setApiGroup(v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-44">
              <SelectValue placeholder="API 分组" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部分组</SelectItem>
              {groups.map((g) => (
                <SelectItem key={g} value={g}>
                  {g}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {hasFilter && (
            <Button variant="ghost" size="sm" onClick={resetFilters}>
              <X className="size-4" /> 重置
            </Button>
          )}
          <div className="ml-auto flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={syncMutation.isPending}
              onClick={() => syncMutation.mutate()}
            >
              {syncMutation.isPending ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <RefreshCw className="size-4" />
              )}
              同步路由
            </Button>
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw className="size-4" /> 刷新
            </Button>
          </div>
        </div>
      }
    >
      {syncMutation.isSuccess && syncMutation.data ? (
        <div className="mb-3 rounded-md border border-emerald-500/30 bg-emerald-500/5 px-4 py-2 text-sm text-emerald-600 dark:text-emerald-400">
          同步完成：新增 {syncMutation.data.created} · 更新{' '}
          {syncMutation.data.updated} · 总计 {syncMutation.data.total}
        </div>
      ) : null}
      {syncMutation.isError ? (
        <div className="mb-3 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-2 text-sm text-destructive">
          同步失败：
          {syncMutation.error instanceof Error
            ? syncMutation.error.message
            : '未知错误'}
        </div>
      ) : null}

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
              <EmptyRow colSpan={cols.length}>
                {hasFilter ? '没有匹配的接口' : '暂无接口'}
              </EmptyRow>
            ) : (
              rows.map((e) => (
                <TableRow key={e.id}>
                  <TableCell className="font-mono text-xs">{e.path}</TableCell>
                  <TableCell>
                    <Badge variant={METHOD_VARIANT[e.method] ?? 'outline'}>
                      {e.method}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-sm">{e.name || '—'}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {e.apiGroup || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {e.module || '—'}
                  </TableCell>
                  <TableCell
                    className="max-w-[260px] truncate text-xs text-muted-foreground"
                    title={e.description}
                  >
                    {e.description || '—'}
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
        pageSize={PAGE_SIZE}
        onChange={setPage}
      />
    </PageShell>
  )
}
