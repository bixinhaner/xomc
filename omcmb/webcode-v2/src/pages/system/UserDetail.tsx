import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { PageShell, formatTime } from '@/components/layout/PageShell'

import { useUserById } from '@core/hooks/api/useSystem'
import type { UserStatus } from '@core/types/system'

// ============================================================
// 系统管理 / 用户详情 — :id 带参，真实数据 useUserById（adminApi.getUserById）
// ============================================================

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

export default function UserDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: user, isLoading, isError, error } = useUserById(id)

  const statusMeta = user
    ? USER_STATUS_META[user.status] ?? USER_STATUS_META.disabled
    : null

  return (
    <PageShell
      title="用户详情"
      description={user ? user.username : id}
      toolbar={
        <Button variant="outline" size="sm" onClick={() => navigate('/system/users')}>
          <ArrowLeft className="size-4" /> 返回用户列表
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
      ) : !user ? (
        <div className="rounded-lg border bg-card px-4 py-3 text-sm text-muted-foreground">
          未找到该用户
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          <div className="overflow-hidden rounded-lg border bg-card">
            <Field label="显示名">{user.displayName || '—'}</Field>
            <Field label="账号">
              <span className="font-mono text-xs">{user.username}</span>
            </Field>
            <Field label="邮箱">{user.email || '—'}</Field>
            <Field label="电话">{user.phone || '—'}</Field>
            <Field label="部门">{user.department || '—'}</Field>
          </div>
          <div className="overflow-hidden rounded-lg border bg-card">
            <Field label="状态">
              {statusMeta ? (
                <Badge variant={statusMeta.variant}>{statusMeta.label}</Badge>
              ) : (
                '—'
              )}
            </Field>
            <Field label="来源">
              <Badge variant={user.source === 'builtIn' ? 'default' : 'outline'}>
                {user.source ? USER_SOURCE_LABEL[user.source] ?? user.source : '—'}
              </Badge>
            </Field>
            <Field label="角色">
              {(user.roles ?? []).length === 0 ? (
                <span className="text-muted-foreground">—</span>
              ) : (
                <div className="flex flex-wrap gap-1">
                  {(user.roles ?? []).map((r) => (
                    <Badge key={r} variant="outline">
                      {r}
                    </Badge>
                  ))}
                </div>
              )}
            </Field>
            <Field label="最近登录">{formatTime(user.lastLoginTime)}</Field>
            <Field label="创建时间">{formatTime(user.createTime)}</Field>
          </div>
          {user.description ? (
            <div className="overflow-hidden rounded-lg border bg-card md:col-span-2">
              <Field label="备注">{user.description}</Field>
            </div>
          ) : null}
        </div>
      )}
    </PageShell>
  )
}
