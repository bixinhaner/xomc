import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Search } from 'lucide-react'

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
  formatTime,
} from '@/components/layout/PageShell'

import { useAllRoles, useUsers } from '@core/hooks/api/useSystem'
import type { UserStatus } from '@core/types/system'

// ============================================================
// 系统管理 / 用户管理 — 对齐 v1 webcode/src/pages/system/UserManagement
// 真实数据 useUsers（adminApi.getUsers）。点用户名进详情 /system/users/:id。
// ============================================================

const PAGE_SIZE = 20

const USER_STATUS_META: Record<
  UserStatus,
  { label: string; variant: 'success' | 'muted' | 'warning' | 'destructive' }
> = {
  active: { label: '启用', variant: 'success' },
  disabled: { label: '禁用', variant: 'muted' },
  inactive: { label: '未激活', variant: 'muted' },
  locked: { label: '已锁定', variant: 'destructive' },
}

const USER_SOURCE_LABEL: Record<string, string> = {
  builtIn: '内置',
  admin: '本地',
  LDAP: 'LDAP',
}

export default function UserManagement() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [userName, setUserName] = useState('')
  const [roleId, setRoleId] = useState('all')
  const [status, setStatus] = useState<'all' | UserStatus>('all')

  const { data: allRoles } = useAllRoles()

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(userName.trim() ? { userName: userName.trim() } : {}),
      ...(roleId !== 'all' ? { roleId } : {}),
      ...(status !== 'all' ? { status } : {}),
    }),
    [page, roleId, status, userName]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useUsers(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['用户', '账号', '角色', '来源', '状态', '部门', '最近登录', '创建时间']

  return (
    <PageShell
      title="用户管理"
      description={`共 ${total} 个用户`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-72 pl-9"
              placeholder="搜索用户名 / 显示名"
              value={userName}
              onChange={(e) => {
                setUserName(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <Select
            value={roleId}
            onValueChange={(value) => {
              setRoleId(value)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-44">
              <SelectValue placeholder="全部角色" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部角色</SelectItem>
              {(allRoles ?? []).map((role) => (
                <SelectItem key={role.id} value={role.id}>
                  {role.roleName}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select
            value={status}
            onValueChange={(value) => {
              setStatus(value as 'all' | UserStatus)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="全部状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              <SelectItem value="active">启用</SelectItem>
              <SelectItem value="disabled">禁用</SelectItem>
              <SelectItem value="inactive">未激活</SelectItem>
              <SelectItem value="locked">已锁定</SelectItem>
            </SelectContent>
          </Select>
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
              <EmptyRow colSpan={cols.length}>暂无用户</EmptyRow>
            ) : (
              rows.map((u) => {
                const statusMeta =
                  USER_STATUS_META[u.status] ?? USER_STATUS_META.disabled
                const roleNames = u.roles ?? []
                return (
                  <TableRow key={u.id}>
                    <TableCell>
                      <button
                        type="button"
                        className="text-left font-medium text-primary hover:underline"
                        onClick={() => navigate(`/system/users`)}
                      >
                        {u.displayName || u.username}
                      </button>
                      {u.email ? (
                        <div className="text-xs text-muted-foreground">
                          {u.email}
                        </div>
                      ) : null}
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {u.username}
                    </TableCell>
                    <TableCell className="text-xs">
                      {roleNames.length === 0 ? (
                        <span className="text-muted-foreground">—</span>
                      ) : (
                        <div className="flex flex-wrap gap-1">
                          {roleNames.slice(0, 2).map((r) => (
                            <Badge key={r} variant="outline">
                              {r}
                            </Badge>
                          ))}
                          {roleNames.length > 2 ? (
                            <Badge variant="muted">+{roleNames.length - 2}</Badge>
                          ) : null}
                        </div>
                      )}
                    </TableCell>
                    <TableCell className="text-xs">
                      <Badge
                        variant={u.source === 'builtIn' ? 'default' : 'outline'}
                      >
                        {u.source ? USER_SOURCE_LABEL[u.source] ?? u.source : '—'}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={statusMeta.variant}>
                        {statusMeta.label}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {u.department || '—'}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(u.lastLoginTime)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(u.createTime)}
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
    </PageShell>
  )
}
