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

import { useRoles } from '@core/hooks/api/useSystem'

// ============================================================
// 系统管理 / 角色与权限 — 对齐 v1 webcode/src/pages/system/RolePermission
// 真实数据 useRoles（adminApi.getRoles）。点角色名进 /system/roles/:id。
// ============================================================

const PAGE_SIZE = 20

export default function RolePermission() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [roleName, setRoleName] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(roleName.trim() ? { roleName: roleName.trim() } : {}),
    }),
    [page, roleName]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useRoles(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['角色名', '编码', '用户数', '设备权限', '内置', '描述', '更新时间']

  return (
    <PageShell
      title="角色与权限"
      description={`共 ${total} 个角色`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-72 pl-9"
              placeholder="搜索角色名"
              value={roleName}
              onChange={(e) => {
                setRoleName(e.target.value)
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
              <EmptyRow colSpan={cols.length}>暂无角色</EmptyRow>
            ) : (
              rows.map((r) => {
                const isBuiltIn = r.builtIn === 1 || r.builtIn === 2
                const groupCount = r.deviceGroupIds?.length ?? 0
                return (
                  <TableRow key={r.id}>
                    <TableCell>
                      <button
                        type="button"
                        className="text-left font-medium text-primary hover:underline"
                        onClick={() => navigate(`/system/roles/${r.id}`)}
                      >
                        {r.roleName}
                      </button>
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      {r.roleCode}
                    </TableCell>
                    <TableCell className="tabular-nums">{r.userCount}</TableCell>
                    <TableCell className="text-xs">
                      {isBuiltIn ? (
                        <Badge variant="default">全部设备</Badge>
                      ) : groupCount > 0 ? (
                        <Badge variant="outline">{groupCount} 个分组</Badge>
                      ) : (
                        <Badge variant="warning">未绑定</Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      {isBuiltIn ? (
                        <Badge variant="secondary">内置</Badge>
                      ) : (
                        <span className="text-xs text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell
                      className="max-w-[220px] truncate text-xs text-muted-foreground"
                      title={r.description}
                    >
                      {r.description || '—'}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(r.updateTime)}
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
