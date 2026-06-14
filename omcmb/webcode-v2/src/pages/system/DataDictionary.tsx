import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Loader2, RefreshCcw, Search } from 'lucide-react'

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
  LoadingRow,
  PageShell,
  TableCard,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { adminApi } from '@core/services/api/adminApi'
import type { Dictionary } from '@core/services/api/adminApi'

// ============================================================
// 系统管理 / 数据字典 — 对齐 v1 webcode/src/pages/system/DataDictionary
// 真实数据：adminApi.getDictionaryList（左侧字典列表） +
//   adminApi.getDictionaryDetailList（右侧字典项树）。左右联动只读视图。
// ============================================================

function originBadge(origin: string) {
  return origin === 'auto' ? (
    <Badge variant="default">自动同步</Badge>
  ) : (
    <Badge variant="muted">手工录入</Badge>
  )
}

export default function DataDictionary() {
  const [search, setSearch] = useState('')
  const [selectedId, setSelectedId] = useState<number | null>(null)

  const listQuery = useQuery({
    queryKey: ['dictionaries'],
    queryFn: () => adminApi.getDictionaryList(),
  })

  const dicts = listQuery.data?.list ?? []

  const filteredDicts = useMemo(() => {
    if (!search.trim()) return dicts
    const lower = search.toLowerCase()
    return dicts.filter(
      (d) =>
        d.name.toLowerCase().includes(lower) ||
        d.type.toLowerCase().includes(lower)
    )
  }, [dicts, search])

  // 默认选中第一条，保证右侧详情不恒空。
  const effectiveSelectedId =
    selectedId ?? (filteredDicts.length > 0 ? filteredDicts[0].id : null)
  const selectedDict: Dictionary | null =
    dicts.find((d) => d.id === effectiveSelectedId) ?? null

  const detailQuery = useQuery({
    queryKey: ['dictionary-details', effectiveSelectedId],
    queryFn: () =>
      adminApi.getDictionaryDetailList({
        sysDictionaryId: effectiveSelectedId as number,
      }),
    enabled: effectiveSelectedId != null,
  })

  const details = detailQuery.data?.list ?? []
  const detailCols = ['展示名', '值', '层级', '来源', '状态', '排序']

  return (
    <PageShell
      title="数据字典"
      description={`共 ${dicts.length} 个字典`}
      isFetching={listQuery.isFetching || detailQuery.isFetching}
      toolbar={
        <div className="flex w-full items-center gap-2">
          <span className="text-sm text-muted-foreground">字典 + 字典项（只读视图）</span>
          <div className="ml-auto">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                void listQuery.refetch()
                void detailQuery.refetch()
              }}
            >
              <RefreshCcw className="size-4" /> 刷新
            </Button>
          </div>
        </div>
      }
    >
      <div className="grid gap-4 lg:grid-cols-[320px_1fr]">
        {/* 左：字典列表 */}
        <div className="flex flex-col overflow-hidden rounded-lg border bg-card">
          <div className="border-b p-2">
            <div className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                className="pl-9"
                placeholder="搜索字典名 / 类型"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
          </div>
          <div className="max-h-[600px] flex-1 overflow-auto">
            {listQuery.isLoading ? (
              <div className="flex h-32 items-center justify-center">
                <Loader2 className="size-5 animate-spin text-muted-foreground" />
              </div>
            ) : listQuery.isError ? (
              <div className="p-4 text-sm text-destructive">
                加载失败：
                {listQuery.error instanceof Error
                  ? listQuery.error.message
                  : '未知错误'}
              </div>
            ) : filteredDicts.length === 0 ? (
              <div className="p-4 text-sm text-muted-foreground">暂无字典</div>
            ) : (
              <ul>
                {filteredDicts.map((d) => {
                  const active = d.id === effectiveSelectedId
                  return (
                    <li key={d.id}>
                      <button
                        type="button"
                        onClick={() => setSelectedId(d.id)}
                        className={cn(
                          'flex w-full flex-col gap-1 border-l-2 px-3 py-2 text-left transition-colors',
                          active
                            ? 'border-primary bg-primary/5'
                            : 'border-transparent hover:bg-muted'
                        )}
                      >
                        <span
                          className={cn(
                            'truncate text-sm font-medium',
                            active && 'text-primary'
                          )}
                        >
                          {d.name}
                        </span>
                        <span className="flex flex-wrap items-center gap-1">
                          <Badge variant="outline">{d.type}</Badge>
                          {!d.status ? (
                            <Badge variant="muted">禁用</Badge>
                          ) : null}
                          {d.sourceTable ? (
                            <Badge variant="secondary">托管</Badge>
                          ) : null}
                        </span>
                      </button>
                    </li>
                  )
                })}
              </ul>
            )}
          </div>
        </div>

        {/* 右：字典项 */}
        <div>
          <div className="mb-2 text-sm font-medium">
            {selectedDict ? (
              <>
                字典项 — {selectedDict.name}{' '}
                <span className="text-xs text-muted-foreground">
                  共 {detailQuery.data?.total ?? details.length} 项
                </span>
              </>
            ) : (
              '字典项'
            )}
          </div>
          <TableCard>
            <Table>
              <TableHeader>
                <TableRow>
                  {detailCols.map((c) => (
                    <TableHead key={c}>{c}</TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {!selectedDict ? (
                  <EmptyRow colSpan={detailCols.length}>请选择左侧字典</EmptyRow>
                ) : detailQuery.isLoading ? (
                  <LoadingRow colSpan={detailCols.length} />
                ) : detailQuery.isError ? (
                  <tr>
                    <td
                      colSpan={detailCols.length}
                      className="h-32 p-3 text-center text-destructive"
                    >
                      加载失败：
                      {detailQuery.error instanceof Error
                        ? detailQuery.error.message
                        : '未知错误'}
                    </td>
                  </tr>
                ) : details.length === 0 ? (
                  <EmptyRow colSpan={detailCols.length}>暂无字典项</EmptyRow>
                ) : (
                  details.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell
                        className="text-sm"
                        style={{ paddingLeft: 12 + (item.level ?? 0) * 18 }}
                      >
                        {item.label}
                      </TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {item.value}
                      </TableCell>
                      <TableCell className="tabular-nums text-xs text-muted-foreground">
                        {item.level ?? 0}
                      </TableCell>
                      <TableCell>{originBadge(item.origin)}</TableCell>
                      <TableCell>
                        <Badge variant={item.status ? 'success' : 'muted'}>
                          {item.status ? '启用' : '禁用'}
                        </Badge>
                      </TableCell>
                      <TableCell className="tabular-nums text-xs text-muted-foreground">
                        {item.sort}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableCard>
        </div>
      </div>
    </PageShell>
  )
}
