import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  ChevronDown,
  ChevronRight,
  Loader2,
  Lock,
  RefreshCcw,
  Search,
  Trash2,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card } from '@/components/ui/card'
import { PageShell } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useGroupTree } from '@core/hooks/api/useMmlConsole'
import {
  useDeleteGroup,
  useDeleteCommand,
} from '@core/hooks/api/useMmlAdmin'
import type { GroupTreeNode, GroupTreeCommand } from '@core/types/mmlConsole'

// ============================================================
// MML 命令字典管理 — 分组 / 命令树（对齐 v1 webcode mml/admin/catalog）
// 数据：useGroupTree（hierarchical 树）+ useDeleteGroup / useDeleteCommand。
// 左栏分组树（可展开、可删除分组）；命令叶子点击进命令详情 /mml/admin/command/:id，
// 命令含 catalog_protected 标记时禁删（仅展示锁标）。
// （新增 / 编辑分组/命令的复杂表单见 v1 admin 页，本皮肤聚焦字典浏览 + 删除维护。）
// ============================================================

export default function Catalog() {
  const navigate = useNavigate()
  const { data, isLoading, isError, error, isFetching, refetch } = useGroupTree()
  const deleteGroup = useDeleteGroup()
  const deleteCommand = useDeleteCommand()

  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [filter, setFilter] = useState('')
  const [actionErr, setActionErr] = useState<string | null>(null)

  const topGroups = useMemo<GroupTreeNode[]>(
    () => (data ?? []).filter((g) => !g.path.includes('.')),
    [data]
  )

  const q = filter.trim().toLowerCase()

  const toggleGroup = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const handleDeleteGroup = (g: GroupTreeNode) => {
    if ((g.commands ?? []).length > 0) {
      setActionErr(`分组「${g.displayName}」非空，请先删除其下命令。`)
      return
    }
    if (!window.confirm(`确认删除分组「${g.displayName}」？`)) return
    setActionErr(null)
    deleteGroup.mutate(g.id, {
      onError: (e) => setActionErr(e instanceof Error ? e.message : '删除分组失败'),
    })
  }

  const handleDeleteCommand = (c: GroupTreeCommand) => {
    if (!window.confirm(`确认删除命令「${c.displayName}」？`)) return
    setActionErr(null)
    deleteCommand.mutate(c.id, {
      onError: (e) => setActionErr(e instanceof Error ? e.message : '删除命令失败'),
    })
  }

  const totalCommands = useMemo(
    () => topGroups.reduce((acc, g) => acc + (g.commands ?? []).length, 0),
    [topGroups]
  )

  return (
    <PageShell
      title="MML 命令字典管理"
      description="分组 / 命令树维护（删除、查看明细）"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-72 pl-9"
              placeholder="过滤命令名 / 命令码"
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
            />
          </div>
          <span className="text-sm text-muted-foreground">
            {topGroups.length} 个分组 · {totalCommands} 条命令
          </span>
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw /> 刷新
            </Button>
          </div>
        </div>
      }
    >
      {actionErr && (
        <div className="mb-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive">
          {actionErr}
        </div>
      )}

      <Card className="p-2">
        {isLoading ? (
          <div className="flex h-48 items-center justify-center">
            <Loader2 className="size-6 animate-spin text-muted-foreground" />
          </div>
        ) : isError ? (
          <div className="flex h-48 items-center justify-center p-6 text-sm text-destructive">
            加载失败：{error instanceof Error ? error.message : '未知错误'}
          </div>
        ) : topGroups.length === 0 ? (
          <div className="flex h-48 items-center justify-center p-6 text-sm text-muted-foreground">
            暂无命令字典
          </div>
        ) : (
          <ul className="py-1">
            {topGroups.map((g) => {
              const cmds = (g.commands ?? []).filter((c) =>
                q
                  ? c.displayName.toLowerCase().includes(q) ||
                    c.commandCode.toLowerCase().includes(q)
                  : true
              )
              if (q && cmds.length === 0) return null
              const open = q ? true : expanded.has(g.id)
              return (
                <li key={g.id} className="border-b last:border-b-0">
                  <div className="flex items-center gap-1 px-2 py-2 hover:bg-muted/60">
                    <button
                      type="button"
                      onClick={() => toggleGroup(g.id)}
                      className="flex min-w-0 flex-1 items-center gap-1.5 text-left text-sm font-medium"
                    >
                      {open ? (
                        <ChevronDown className="size-4 shrink-0" />
                      ) : (
                        <ChevronRight className="size-4 shrink-0" />
                      )}
                      <span className="truncate">{g.displayName}</span>
                      {g.catalogProtected && (
                        <Lock className="size-3 shrink-0 text-muted-foreground" />
                      )}
                      <Badge variant="muted" className="shrink-0">
                        {(g.commands ?? []).length}
                      </Badge>
                      {g.source && (
                        <Badge variant="outline" className="shrink-0">
                          {g.source}
                        </Badge>
                      )}
                    </button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      disabled={
                        g.catalogProtected ||
                        deleteGroup.isPending ||
                        (g.commands ?? []).length > 0
                      }
                      title={
                        g.catalogProtected
                          ? '受保护分组，不可删除'
                          : (g.commands ?? []).length > 0
                            ? '分组非空,先删命令'
                            : '删除分组'
                      }
                      onClick={() => handleDeleteGroup(g)}
                    >
                      <Trash2 />
                    </Button>
                  </div>

                  {open && (
                    <ul className="bg-muted/20 pb-1 pl-7">
                      {cmds.length === 0 ? (
                        <li className="px-2 py-1.5 text-xs text-muted-foreground">
                          该分组无命令
                        </li>
                      ) : (
                        cmds.map((c) => (
                          <li
                            key={c.id}
                            className="flex items-center gap-2 px-2 py-1.5 text-xs hover:bg-muted/60"
                          >
                            <Badge variant="outline" className="shrink-0">
                              {c.operationType}
                            </Badge>
                            <button
                              type="button"
                              className={cn(
                                'min-w-0 flex-1 truncate text-left text-primary hover:underline'
                              )}
                              onClick={() =>
                                navigate(`/mml/admin/command/${c.id}`)
                              }
                            >
                              {c.displayName}
                            </button>
                            <span className="shrink-0 font-mono text-[11px] text-muted-foreground">
                              {c.commandCode}
                            </span>
                            {c.catalogProtected && (
                              <Lock className="size-3 shrink-0 text-muted-foreground" />
                            )}
                            <Button
                              variant="ghost"
                              size="sm"
                              className="text-destructive hover:text-destructive"
                              disabled={c.catalogProtected || deleteCommand.isPending}
                              title={
                                c.catalogProtected ? '受保护命令，不可删除' : '删除命令'
                              }
                              onClick={() => handleDeleteCommand(c)}
                            >
                              <Trash2 />
                            </Button>
                          </li>
                        ))
                      )}
                    </ul>
                  )}
                </li>
              )
            })}
          </ul>
        )}
      </Card>
    </PageShell>
  )
}
