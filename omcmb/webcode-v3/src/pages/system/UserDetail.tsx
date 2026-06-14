import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useUserById } from '@core/hooks/api/useSystem'
import type { UserStatus } from '@core/types/system'

import { StateBlock, FieldRow } from './_shared'

const USER_STATUS_MAP: Record<UserStatus, { label: string; badge: string }> = {
  active: { label: '启用', badge: 'active' },
  disabled: { label: '禁用', badge: 'off' },
  inactive: { label: '未激活', badge: 'inactive' },
  locked: { label: '锁定', badge: 'critical' },
}

export default function UserDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: user, isLoading, isError, error, isFetching } = useUserById(id)

  return (
    <PageShell
      code="F06"
      title="USER · 账号档案"
      subtitle={id}
      isFetching={isFetching}
      toolbar={
        <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/system/users')}>
          返回清单
        </NeonButton>
      }
    >
      <StateBlock
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={!user}
        emptyLabel="USER NOT FOUND · 账号不存在"
      >
        {user ? (
          <div className="space-y-5">
            <div className="flex items-center gap-4">
              <div className="flex size-14 items-center justify-center rounded-sm border border-cyan-500/30 bg-cyan-500/10 font-display text-2xl font-bold text-cyan-200">
                {(user.displayName || user.username || '?').slice(0, 1).toUpperCase()}
              </div>
              <div>
                <div className="font-display text-xl font-bold text-cyan-100">
                  {user.displayName || user.username}
                </div>
                <div className="font-mono text-xs text-cyan-300/60">{user.username}</div>
              </div>
              <div className="ml-auto">
                <StatusBadge
                  status={USER_STATUS_MAP[user.status].badge}
                  label={USER_STATUS_MAP[user.status].label}
                />
              </div>
            </div>

            <div className="grid gap-x-8 md:grid-cols-2">
              <div>
                <FieldRow label="账号 · LOGIN">{user.username}</FieldRow>
                <FieldRow label="显示名">{user.displayName || '—'}</FieldRow>
                <FieldRow label="邮箱 · EMAIL">{user.email || '—'}</FieldRow>
                <FieldRow label="手机 · PHONE">{user.phone || '—'}</FieldRow>
                <FieldRow label="部门">{user.department || '—'}</FieldRow>
                <FieldRow label="来源 · SOURCE">
                  {user.source === 'builtIn' ? '内置' : user.source === 'LDAP' ? 'LDAP' : '本地'}
                </FieldRow>
              </div>
              <div>
                <FieldRow label="角色 · ROLES">
                  {user.roles && user.roles.length > 0 ? (
                    <div className="flex flex-wrap gap-1.5">
                      {user.roles.map((r) => (
                        <span key={r} className="chip text-cyan-200">
                          {r}
                        </span>
                      ))}
                    </div>
                  ) : (
                    '—'
                  )}
                </FieldRow>
                <FieldRow label="超管 · SUPER">{user.isSuperAdmin ? '是' : '否'}</FieldRow>
                <FieldRow label="最后登录">
                  {user.lastLoginTime ? formatTime(user.lastLoginTime) : '—'}
                </FieldRow>
                <FieldRow label="到期 · EXPIRE">
                  {user.expireTime ? formatTime(user.expireTime) : '永久'}
                </FieldRow>
                <FieldRow label="创建时间">{formatTime(user.createTime)}</FieldRow>
                <FieldRow label="更新时间">
                  {user.updateTime ? formatTime(user.updateTime) : '—'}
                </FieldRow>
              </div>
            </div>

            {user.description ? (
              <div className="glass rounded-sm p-3 text-sm text-cyan-100/80">
                <div className="mb-1 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">
                  备注 · NOTE
                </div>
                {user.description}
              </div>
            ) : null}
          </div>
        ) : null}
      </StateBlock>
    </PageShell>
  )
}
