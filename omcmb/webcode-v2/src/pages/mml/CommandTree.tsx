import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ListTree, RefreshCcw, Search } from 'lucide-react'

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
import { cn } from '@/lib/utils'

import { useMMLCommands } from '@core/hooks/api/useMML'

// ============================================================
// MML 命令树 — 命令字典分类浏览（对齐 v1 webcode mml/CommandTree）
// 左侧分类树 + 右侧命令列表，行可进命令详情子路由 /mml/command-detail/:id
// 数据走 useMMLCommands（keyword + category 服务端过滤）。
// ============================================================

const CATEGORIES: { key: string; label: string }[] = [
  { key: '', label: '全部分类' },
  { key: '1', label: '小区管理' },
  { key: '2', label: '邻区管理' },
  { key: '3', label: '基站管理' },
  { key: '4', label: '告警查询' },
  { key: '5', label: '性能采集' },
  { key: '6', label: '传输管理' },
  { key: '7', label: '版本管理' },
]

const CATEGORY_LABEL: Record<string, string> = Object.fromEntries(
  CATEGORIES.filter((c) => c.key).map((c) => [c.key, c.label])
)

const PAGE_SIZE = 20

export default function CommandTree() {
  const navigate = useNavigate()
  const [category, setCategory] = useState('')
  const [kwInput, setKwInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(category ? { category } : {}),
    }),
    [page, keyword, category]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useMMLCommands(params)

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['命令名称', '命令码', '操作', '分类', '描述', '参数', '操作']

  return (
    <PageShell
      title="MML 命令树"
      description="命令字典分类浏览 · 查看命令绑定参数"
      isFetching={isFetching}
    >
      <div className="flex gap-4">
        {/* 左侧分类树 */}
        <div className="w-52 shrink-0">
          <div className="rounded-lg border bg-card p-2">
            <div className="mb-1 flex items-center gap-1.5 px-2 py-1 text-xs font-medium uppercase tracking-wider text-muted-foreground">
              <ListTree className="size-3.5" /> 命令分类
            </div>
            <ul className="space-y-0.5">
              {CATEGORIES.map((c) => {
                const active = category === c.key
                return (
                  <li key={c.key || 'all'}>
                    <button
                      type="button"
                      onClick={() => {
                        setCategory(c.key)
                        setPage(1)
                      }}
                      className={cn(
                        'w-full rounded-md px-2 py-1.5 text-left text-sm transition-colors',
                        active
                          ? 'bg-primary/10 font-medium text-primary'
                          : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                      )}
                    >
                      {c.label}
                    </button>
                  </li>
                )
              })}
            </ul>
          </div>
        </div>

        {/* 右侧命令列表 */}
        <div className="min-w-0 flex-1">
          <div className="mb-3 flex flex-wrap items-center gap-2">
            <div className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                className="w-72 pl-9"
                placeholder="搜索命令名 / 命令码 / 描述"
                value={kwInput}
                onChange={(e) => setKwInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    setKeyword(kwInput)
                    setPage(1)
                  }
                }}
              />
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setKeyword(kwInput)
                setPage(1)
              }}
            >
              <Search /> 搜索
            </Button>
            <span className="text-sm text-muted-foreground">
              {category ? CATEGORY_LABEL[category] : '全部命令'} · 共 {total} 条
            </span>
            <div className="ml-auto">
              <Button variant="outline" size="sm" onClick={() => refetch()}>
                <RefreshCcw /> 刷新
              </Button>
            </div>
          </div>

          <TableCard>
            <Table>
              <TableHeader>
                <TableRow>
                  {cols.map((c, i) => (
                    <TableHead key={`${c}-${i}`}>{c}</TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  <LoadingRow colSpan={cols.length} />
                ) : isError ? (
                  <ErrorRow colSpan={cols.length} error={error} />
                ) : rows.length === 0 ? (
                  <EmptyRow colSpan={cols.length}>暂无命令</EmptyRow>
                ) : (
                  rows.map((c) => {
                    const paramCount = c.paramRefs?.length ?? c.params.length
                    return (
                      <TableRow key={c.id}>
                        <TableCell>
                          <button
                            type="button"
                            className="font-medium text-primary hover:underline"
                            onClick={() => navigate(`/mml/command-detail/${c.id}`)}
                          >
                            {c.commandName}
                          </button>
                        </TableCell>
                        <TableCell className="font-mono text-xs">
                          {c.commandCode}
                        </TableCell>
                        <TableCell>
                          {c.operationType ? (
                            <Badge variant="outline">{c.operationType}</Badge>
                          ) : (
                            <span className="text-xs text-muted-foreground">—</span>
                          )}
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {CATEGORY_LABEL[c.category] || c.category || '—'}
                        </TableCell>
                        <TableCell className="max-w-sm text-xs text-muted-foreground">
                          <span className="line-clamp-1">{c.description || '—'}</span>
                        </TableCell>
                        <TableCell className="text-xs tabular-nums">
                          {paramCount}
                        </TableCell>
                        <TableCell>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => navigate(`/mml/command-detail/${c.id}`)}
                          >
                            详情
                          </Button>
                        </TableCell>
                      </TableRow>
                    )
                  })
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
        </div>
      </div>
    </PageShell>
  )
}
