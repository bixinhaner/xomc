import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Search } from 'lucide-react'

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
  formatTime,
} from '@/components/layout/PageShell'

import { useGroups } from '@core/hooks/api/useSystem'

// ============================================================
// 系统管理 / 用户组 — 对齐 v1 webcode/src/pages/system/GroupManagement
// 真实数据 useGroups（adminApi.getGroups）。点用户组进 /system/groups/:id。
// ============================================================

const PAGE_SIZE = 20

export default function GroupManagement() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [groupName, setGroupName] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(groupName.trim() ? { groupName: groupName.trim() } : {}),
    }),
    [page, groupName]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useGroups(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['用户组', '用户数', '角色数', '内置', '描述', '更新人', '更新时间']

  return (
    <PageShell
      title="用户组"
      description={`共 ${total} 个用户组`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-72 pl-9"
              placeholder="搜索用户组"
              value={groupName}
              onChange={(e) => {
                setGroupName(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
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
              <EmptyRow colSpan={cols.length}>暂无用户组</EmptyRow>
            ) : (
              rows.map((g) => (
                <TableRow key={g.id}>
                  <TableCell>
                    <button
                      type="button"
                      className="text-left font-medium text-primary hover:underline"
                      onClick={() => navigate(`/system/groups/${g.id}`)}
                    >
                      {g.groupName}
                    </button>
                  </TableCell>
                  <TableCell className="tabular-nums">{g.userCount}</TableCell>
                  <TableCell className="tabular-nums">{g.roleCount}</TableCell>
                  <TableCell>
                    {g.builtIn ? (
                      <Badge variant="secondary">内置</Badge>
                    ) : (
                      <span className="text-xs text-muted-foreground">—</span>
                    )}
                  </TableCell>
                  <TableCell
                    className="max-w-[240px] truncate text-xs text-muted-foreground"
                    title={g.description}
                  >
                    {g.description || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {g.updUser || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(g.updTime)}
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
