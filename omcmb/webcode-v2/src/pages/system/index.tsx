import { useMemo, useState } from 'react'
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

import { useUsers } from '@core/hooks/api/useSystem'

export function SystemPage() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [userName, setUserName] = useState('')

  const params = useMemo(
    () => ({ page, pageSize, ...(userName.trim() ? { userName: userName.trim() } : {}) }),
    [page, pageSize, userName]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useUsers(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['用户名', '邮箱', '用户组', '状态', '在线', '最近登录']

  return (
    <PageShell
      title="系统管理"
      description="用户 · 角色 · 权限（当前视图：用户列表）"
      isFetching={isFetching}
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-72 pl-9"
              placeholder="搜索用户名"
              value={userName}
              onChange={(e) => {
                setUserName(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
          </Button>
        </>
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
              <EmptyRow colSpan={cols.length}>暂无用户</EmptyRow>
            ) : (
              rows.map((u) => (
                <TableRow key={u.id}>
                  <TableCell>
                    <div className="font-medium">{u.userName}</div>
                    {u.department ? (
                      <div className="text-xs text-muted-foreground">{u.department}</div>
                    ) : null}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">{u.email}</TableCell>
                  <TableCell className="text-xs">
                    {(u.groupNames || []).slice(0, 2).map((g) => (
                      <Badge key={g} variant="outline" className="mr-1">
                        {g}
                      </Badge>
                    ))}
                  </TableCell>
                  <TableCell>
                    <Badge variant={u.status === 'enabled' ? 'success' : 'muted'}>
                      {u.status === 'enabled' ? '启用' : '禁用'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={u.onlineStatus === 'online' ? 'success' : 'muted'}>
                      {u.onlineStatus === 'online' ? '在线' : '离线'}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(u.lastLoginTime)}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={setPage} />
    </PageShell>
  )
}
