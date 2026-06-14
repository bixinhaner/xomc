import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Users, Wifi, Lock, ShieldCheck, ChevronRight } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useUsers } from '@core/hooks/api/useSystem'
import type { User, UserStatus } from '@core/types/system'

import { StateBlock, MiniStat, RowHeader, KeywordToolbar, Pager } from './_shared'

const USER_STATUS_MAP: Record<UserStatus, { label: string; badge: string }> = {
  active: { label: '启用', badge: 'active' },
  disabled: { label: '禁用', badge: 'off' },
  inactive: { label: '未激活', badge: 'inactive' },
  locked: { label: '锁定', badge: 'critical' },
}

export default function UserManagement() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(keyword.trim() ? { userName: keyword.trim() } : {}),
    }),
    [page, keyword]
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useUsers(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const activeCount = useMemo(() => rows.filter((u) => u.status === 'active').length, [rows])
  const lockedCount = useMemo(
    () => rows.filter((u) => u.status === 'locked' || u.status === 'disabled').length,
    [rows]
  )
  const builtInCount = useMemo(() => rows.filter((u) => u.source === 'builtIn').length, [rows])

  return (
    <PageShell
      code="F06"
      title="USERS · 人员清单"
      subtitle="ACCOUNT REGISTRY"
      isFetching={isFetching}
      bare
      toolbar={
        <KeywordToolbar
          placeholder="账号 / 用户名"
          value={keyword}
          onChange={(v) => {
            setKeyword(v)
            setPage(1)
          }}
          onRefresh={() => refetch()}
        />
      }
    >
      <div className="flex h-full flex-col gap-3">
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <MiniStat label="账号总数" value={total.toLocaleString()} color="#00f0ff" icon={<Users className="size-3.5" />} />
          <MiniStat label="启用 · ACTIVE" value={activeCount} color="#00ff88" icon={<Wifi className="size-3.5" />} />
          <MiniStat label="禁用/锁定" value={lockedCount} color="#ff7a1a" icon={<Lock className="size-3.5" />} />
          <MiniStat label="内置 · BUILT-IN" value={builtInCount} color="#a855f7" icon={<ShieldCheck className="size-3.5" />} />
        </div>

        <div className="glass-strong relative flex-1 min-h-0 overflow-hidden rounded-sm">
          <div className="scanline" />
          <div className="relative h-full overflow-auto p-3">
            <StateBlock
              isLoading={isLoading}
              isError={isError}
              error={error}
              isEmpty={rows.length === 0}
              emptyLabel="NO ACCOUNTS · 无账号"
            >
              <div className="space-y-1.5">
                <RowHeader cols="2fr_1.4fr_1fr_1.2fr_1fr_1.4fr_0.4fr">
                  <span>用户 · IDENTITY</span>
                  <span>账号 · LOGIN</span>
                  <span>状态</span>
                  <span>角色 · ROLES</span>
                  <span>来源</span>
                  <span>最后登录</span>
                  <span />
                </RowHeader>
                {rows.map((u: User) => {
                  const st = USER_STATUS_MAP[u.status]
                  return (
                    <button
                      key={u.id}
                      type="button"
                      onClick={() => navigate(`/system/users/${u.id}`)}
                      className="fleet-row grid w-full grid-cols-[2fr_1.4fr_1fr_1.2fr_1fr_1.4fr_0.4fr] items-center gap-3 rounded-sm px-3 py-2.5 text-left"
                      style={{ ['--row-color' as never]: '#00f0ff' }}
                    >
                      <div className="min-w-0">
                        <div className="truncate font-display text-sm font-bold text-cyan-100">
                          {u.displayName || u.username}
                        </div>
                        <div className="truncate font-mono text-[10px] text-cyan-300/55">
                          {u.email || '—'}
                        </div>
                      </div>
                      <div className="truncate font-mono text-xs text-cyan-100/85">{u.username}</div>
                      <div>
                        <StatusBadge status={st.badge} label={st.label} />
                      </div>
                      <div className="min-w-0 truncate text-xs text-cyan-100/80">
                        {u.roles && u.roles.length > 0 ? u.roles.join(' · ') : '—'}
                      </div>
                      <div className="font-mono text-[10px] uppercase tracking-[0.12em] text-cyan-300/65">
                        {u.source === 'builtIn' ? '内置' : u.source === 'LDAP' ? 'LDAP' : '本地'}
                      </div>
                      <div className="font-mono text-[11px] text-cyan-300/75">
                        {u.lastLoginTime ? formatTime(u.lastLoginTime) : '—'}
                      </div>
                      <ChevronRight className="size-3.5 justify-self-end text-cyan-300/40" />
                    </button>
                  )
                })}
              </div>
            </StateBlock>
          </div>
        </div>

        <Pager page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} />
      </div>
    </PageShell>
  )
}
