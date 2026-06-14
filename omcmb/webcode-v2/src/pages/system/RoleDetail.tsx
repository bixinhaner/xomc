import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { EmptyRow, PageShell, TableCard, formatTime } from '@/components/layout/PageShell'

import { useRoleById } from '@core/hooks/api/useSystem'
import { useRoleApiPermissions } from '@core/hooks/api/useAdmin'

// ============================================================
// 系统管理 / 角色详情 — :id 带参。真实数据：
//   useRoleById（adminApi.getRoleById） + useRoleApiPermissions（API 权限表）
// ============================================================

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="border-b px-4 py-3 last:border-b-0">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className="mt-1 text-sm">{children}</div>
    </div>
  )
}

export default function RoleDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: role, isLoading, isError, error } = useRoleById(id)
  const apiPerms = useRoleApiPermissions(id)

  const isBuiltIn = role ? role.builtIn === 1 || role.builtIn === 2 : false
  const permCols = ['方法', '路径']

  return (
    <PageShell
      title="角色详情"
      description={role ? role.roleCode : id}
      toolbar={
        <Button variant="outline" size="sm" onClick={() => navigate('/system/roles')}>
          <ArrowLeft className="size-4" /> 返回角色列表
        </Button>
      }
    >
      {isLoading ? (
        <div className="flex h-40 items-center justify-center text-muted-foreground">
          <Loader2 className="size-5 animate-spin" />
        </div>
      ) : isError ? (
        <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </div>
      ) : !role ? (
        <div className="rounded-lg border bg-card px-4 py-3 text-sm text-muted-foreground">
          未找到该角色
        </div>
      ) : (
        <div className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="overflow-hidden rounded-lg border bg-card">
              <Field label="角色名">{role.roleName}</Field>
              <Field label="编码">
                <span className="font-mono text-xs">{role.roleCode}</span>
              </Field>
              <Field label="用户数">
                <span className="tabular-nums">{role.userCount}</span>
              </Field>
              <Field label="内置">
                {isBuiltIn ? (
                  <Badge variant="secondary">内置</Badge>
                ) : (
                  <span className="text-muted-foreground">—</span>
                )}
              </Field>
            </div>
            <div className="overflow-hidden rounded-lg border bg-card">
              <Field label="设备权限">
                {isBuiltIn ? (
                  <Badge variant="default">全部设备</Badge>
                ) : (role.deviceGroupIds?.length ?? 0) > 0 ? (
                  <Badge variant="outline">
                    {role.deviceGroupIds?.length} 个分组
                  </Badge>
                ) : (
                  <Badge variant="warning">未绑定</Badge>
                )}
              </Field>
              <Field label="制式权限">
                {(role.networkTypes ?? []).length === 0 ? (
                  <span className="text-muted-foreground">全部 / 未限制</span>
                ) : (
                  <div className="flex flex-wrap gap-1">
                    {(role.networkTypes ?? []).map((n) => (
                      <Badge key={n} variant="outline">
                        {n}
                      </Badge>
                    ))}
                  </div>
                )}
              </Field>
              <Field label="描述">{role.description || '—'}</Field>
              <Field label="更新时间">{formatTime(role.updateTime)}</Field>
            </div>
          </div>

          <div>
            <div className="mb-2 text-sm font-medium">
              API 权限{' '}
              <span className="text-xs text-muted-foreground">
                共 {apiPerms.data?.length ?? 0} 条
              </span>
            </div>
            <TableCard>
              <Table>
                <TableHeader>
                  <TableRow>
                    {permCols.map((c) => (
                      <TableHead key={c}>{c}</TableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {apiPerms.isLoading ? (
                    <TableRow>
                      <TableCell colSpan={permCols.length} className="h-24 text-center">
                        <Loader2 className="mx-auto size-5 animate-spin text-muted-foreground" />
                      </TableCell>
                    </TableRow>
                  ) : (apiPerms.data?.length ?? 0) === 0 ? (
                    <EmptyRow colSpan={permCols.length}>
                      {isBuiltIn ? '内置角色拥有全部 API 权限' : '未分配 API 权限'}
                    </EmptyRow>
                  ) : (
                    (apiPerms.data ?? []).map((p, idx) => (
                      <TableRow key={`${p.method}-${p.path}-${idx}`}>
                        <TableCell>
                          <Badge variant="outline">{p.method}</Badge>
                        </TableCell>
                        <TableCell className="font-mono text-xs text-muted-foreground">
                          {p.path}
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </TableCard>
          </div>
        </div>
      )}
    </PageShell>
  )
}
