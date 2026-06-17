import { useMemo, useState } from 'react'
import {
  ChevronDown,
  ChevronRight,
  Loader2,
  RefreshCcw,
  Search,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card } from '@/components/ui/card'
import {
  EmptyRow,
  ErrorRow,
  LoadingRow,
  PageShell,
  TableCard,
} from '@/components/layout/PageShell'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { cn } from '@/lib/utils'

import {
  useGroupTreeFlat,
  useSearchCommands,
} from '@core/hooks/api/useMmlConsole'
import {
  isLstObjectPath,
  isModObjectPath,
  isStringObjectPath,
  parseOpFromCommandName,
  stripCommandNameOpPrefix,
} from '@core/types/mmlConsole'
import type { FlatCommand, FlatGroup } from '@core/types/mmlConsole'

// ============================================================
// MML 控制台 V2 — 扁平命令树浏览（对齐 v1 webcode mml/Console 的数据通道）
// 数据：
//   - useGroupTreeFlat   GET /mml/group-tree?format=flat（章节分组 → 命令叶子 + object_path）
//   - useSearchCommands  命令联合搜索（command_code / 名称 / path / 描述）
// 左栏分组树选中命令 → 右栏展示该命令的 object_path 明细（按 LST/MOD/ADD/RMV 区分渲染）。
// 纯浏览/检查视图（结构化执行表单见 v1 控制台）。
// ============================================================

const OP_VARIANT: Record<string, 'default' | 'warning' | 'success' | 'destructive' | 'outline'> =
  {
    LST: 'default',
    MOD: 'warning',
    ADD: 'success',
    RMV: 'destructive',
  }

function PathDetail({ command }: { command: FlatCommand }) {
  const op = parseOpFromCommandName(command.name)
  const name = stripCommandNameOpPrefix(command.name)
  const path = command.object_path

  return (
    <div className="flex flex-col gap-3">
      <Card className="p-4">
        <div className="flex items-center gap-2">
          {op && <Badge variant={OP_VARIANT[op] ?? 'outline'}>{op}</Badge>}
          <h3 className="text-sm font-semibold">{name}</h3>
        </div>
      </Card>

      {isStringObjectPath(path) ? (
        <Card className="p-4">
          <h4 className="mb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
            目标对象路径
          </h4>
          <code className="block break-all rounded-md border bg-muted/40 p-2 font-mono text-xs">
            {path || '—'}
          </code>
        </Card>
      ) : isModObjectPath(path) ? (
        <TableCard>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>参数路径</TableHead>
                <TableHead>类型</TableHead>
                <TableHead>约束</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {path.length === 0 ? (
                <EmptyRow colSpan={3}>无参数</EmptyRow>
              ) : (
                path.map((p) => (
                  <TableRow key={p.path}>
                    <TableCell className="font-mono text-xs">{p.path}</TableCell>
                    <TableCell className="text-xs">{p.type}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {p.min != null || p.max != null
                        ? `范围 ${p.min ?? '-∞'} ~ ${p.max ?? '+∞'}`
                        : p.max_length != null
                          ? `最大长度 ${p.max_length}`
                          : '—'}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableCard>
      ) : isLstObjectPath(path) ? (
        <TableCard>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>参数路径（只读）</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {path.length === 0 ? (
                <EmptyRow colSpan={1}>无参数路径</EmptyRow>
              ) : (
                path.map((p) => (
                  <TableRow key={p}>
                    <TableCell className="font-mono text-xs">{p}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableCard>
      ) : (
        <Card className="p-4 text-xs text-muted-foreground">无路径信息</Card>
      )}
    </div>
  )
}

function SearchResults({
  keyword,
  onPick,
}: {
  keyword: string
  onPick: (commandId: string) => void
}) {
  const { data, isLoading, isError, error } = useSearchCommands(keyword)
  const rows = data ?? []
  const cols = ['命令名称', '命令码', '操作', '分组', '匹配 PATH']

  return (
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
            <EmptyRow colSpan={cols.length}>无匹配命令</EmptyRow>
          ) : (
            rows.map((r) => (
              <TableRow key={r.commandId}>
                <TableCell>
                  <button
                    type="button"
                    className="font-medium text-primary hover:underline"
                    onClick={() => onPick(r.commandId)}
                  >
                    {r.displayName}
                  </button>
                </TableCell>
                <TableCell className="font-mono text-xs">{r.commandCode}</TableCell>
                <TableCell>
                  <Badge variant={OP_VARIANT[r.operationType] ?? 'outline'}>
                    {r.operationType}
                  </Badge>
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {r.groupName}
                </TableCell>
                <TableCell className="max-w-xs text-xs text-muted-foreground">
                  <span className="line-clamp-1">
                    {r.matchedPaths.length > 0 ? r.matchedPaths.join(', ') : '—'}
                  </span>
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </TableCard>
  )
}

export default function Console() {
  const { data, isLoading, isError, error, isFetching, refetch } =
    useGroupTreeFlat()
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [selectedCommandId, setSelectedCommandId] = useState<string | null>(null)
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')

  const groups = useMemo<FlatGroup[]>(() => data?.groups ?? [], [data])

  const selectedCommand = useMemo<FlatCommand | null>(() => {
    for (const g of groups) {
      const c = g.commands.find((cmd) => cmd.id === selectedCommandId)
      if (c) return c
    }
    return null
  }, [groups, selectedCommandId])

  const toggleGroup = (code: string) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(code)) next.delete(code)
      else next.add(code)
      return next
    })
  }

  const pickFromSearch = (commandId: string) => {
    setSelectedCommandId(commandId)
    // 定位到所在分组并展开
    const grp = groups.find((g) => g.commands.some((c) => c.id === commandId))
    if (grp) setExpanded((prev) => new Set(prev).add(grp.code))
    setSearch('')
    setSearchInput('')
  }

  return (
    <PageShell
      title="MML 控制台 V2"
      description="扁平命令树浏览 · 命令搜索 · PATH 明细检查"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-80 pl-9"
              placeholder="搜索命令码 / 名称 / PATH / 描述"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') setSearch(searchInput)
              }}
            />
          </div>
          <Button variant="outline" size="sm" onClick={() => setSearch(searchInput)}>
            <Search /> 搜索
          </Button>
          {search && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setSearch('')
                setSearchInput('')
              }}
            >
              清除搜索
            </Button>
          )}
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw /> 刷新
            </Button>
          </div>
        </div>
      }
    >
      {search ? (
        <SearchResults keyword={search} onPick={pickFromSearch} />
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-[20rem_1fr]">
          {/* 左栏：扁平分组树 */}
          <Card className="flex max-h-[42rem] min-h-[24rem] flex-col p-2">
            <div className="min-h-0 flex-1 overflow-auto">
              {isLoading ? (
                <div className="flex h-32 items-center justify-center">
                  <Loader2 className="size-5 animate-spin text-muted-foreground" />
                </div>
              ) : isError ? (
                <div className="p-3 text-center text-xs text-destructive">
                  加载失败：{error instanceof Error ? error.message : '未知错误'}
                </div>
              ) : groups.length === 0 ? (
                <div className="p-6 text-center text-xs text-muted-foreground">
                  暂无命令树
                </div>
              ) : (
                <ul className="py-1">
                  {groups.map((g) => {
                    const open = expanded.has(g.code)
                    return (
                      <li key={g.code}>
                        <button
                          type="button"
                          onClick={() => toggleGroup(g.code)}
                          className="flex w-full items-center gap-1 px-2 py-1.5 text-left text-xs font-medium hover:bg-muted"
                        >
                          {open ? (
                            <ChevronDown className="size-3.5 shrink-0" />
                          ) : (
                            <ChevronRight className="size-3.5 shrink-0" />
                          )}
                          <span className="truncate">{g.name}</span>
                          <Badge variant="muted" className="ml-auto">
                            {g.commands.length}
                          </Badge>
                        </button>
                        {open && (
                          <ul className="pl-5">
                            {g.commands.map((c) => {
                              const op = parseOpFromCommandName(c.name)
                              return (
                                <li key={c.id}>
                                  <button
                                    type="button"
                                    onClick={() => setSelectedCommandId(c.id)}
                                    className={cn(
                                      'flex w-full items-center gap-1.5 px-2 py-1 text-left text-xs transition-colors hover:bg-muted',
                                      selectedCommandId === c.id &&
                                        'bg-primary/5 text-primary'
                                    )}
                                  >
                                    {op && (
                                      <Badge variant="outline" className="shrink-0">
                                        {op}
                                      </Badge>
                                    )}
                                    <span className="truncate">
                                      {stripCommandNameOpPrefix(c.name)}
                                    </span>
                                  </button>
                                </li>
                              )
                            })}
                          </ul>
                        )}
                      </li>
                    )
                  })}
                </ul>
              )}
            </div>
          </Card>

          {/* 右栏：命令 PATH 明细 */}
          <div>
            {selectedCommand ? (
              <PathDetail command={selectedCommand} />
            ) : (
              <Card className="flex h-48 items-center justify-center text-sm text-muted-foreground">
                请在左侧选择一个命令查看 PATH 明细
              </Card>
            )}
          </div>
        </div>
      )}
    </PageShell>
  )
}
