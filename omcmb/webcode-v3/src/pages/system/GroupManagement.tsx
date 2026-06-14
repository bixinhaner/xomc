import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { UsersRound, ShieldCheck, ChevronRight } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useGroups } from '@core/hooks/api/useSystem'
import type { Group } from '@core/types/system'

import { StateBlock, MiniStat, RowHeader, KeywordToolbar, Pager } from './_shared'

export default function GroupManagement() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(keyword.trim() ? { groupName: keyword.trim() } : {}),
    }),
    [page, keyword]
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useGroups(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const builtInCount = useMemo(() => rows.filter((g) => g.builtIn > 0).length, [rows])
  const totalUsers = useMemo(() => rows.reduce((acc, g) => acc + (g.userCount ?? 0), 0), [rows])
  const totalRoles = useMemo(() => rows.reduce((acc, g) => acc + (g.roleCount ?? 0), 0), [rows])

  return (
    <PageShell
      code="F06"
      title="GROUPS · 用户组"
      subtitle="USER GROUPS"
      isFetching={isFetching}
      bare
      toolbar={
        <KeywordToolbar
          placeholder="用户组名"
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
          <MiniStat label="用户组总数" value={total.toLocaleString()} color="#00f0ff" icon={<UsersRound className="size-3.5" />} />
          <MiniStat label="内置组" value={builtInCount} color="#a855f7" icon={<ShieldCheck className="size-3.5" />} />
          <MiniStat label="成员合计" value={totalUsers} color="#00ff88" />
          <MiniStat label="角色合计" value={totalRoles} color="#5b9eff" />
        </div>

        <div className="glass-strong relative flex-1 min-h-0 overflow-hidden rounded-sm">
          <div className="scanline" />
          <div className="relative h-full overflow-auto p-3">
            <StateBlock
              isLoading={isLoading}
              isError={isError}
              error={error}
              isEmpty={rows.length === 0}
              emptyLabel="NO GROUPS · 无用户组"
            >
              <div className="space-y-1.5">
                <RowHeader cols="2.2fr_0.8fr_0.8fr_0.8fr_1fr_1.4fr_0.4fr">
                  <span>用户组 · GROUP</span>
                  <span>成员</span>
                  <span>角色</span>
                  <span>类型</span>
                  <span>更新人</span>
                  <span>更新时间</span>
                  <span />
                </RowHeader>
                {rows.map((g: Group) => (
                  <button
                    key={g.id}
                    type="button"
                    onClick={() => navigate(`/system/groups/${g.id}`)}
                    className="fleet-row grid w-full grid-cols-[2.2fr_0.8fr_0.8fr_0.8fr_1fr_1.4fr_0.4fr] items-center gap-3 rounded-sm px-3 py-2.5 text-left"
                    style={{ ['--row-color' as never]: g.builtIn > 0 ? '#a855f7' : '#00f0ff' }}
                  >
                    <div className="min-w-0">
                      <div className="truncate font-display text-sm font-bold text-cyan-100">
                        {g.groupName}
                      </div>
                      <div className="truncate font-mono text-[10px] text-cyan-300/55">
                        {g.description || '—'}
                      </div>
                    </div>
                    <div className="font-display text-sm font-bold text-cyan-200">{g.userCount ?? 0}</div>
                    <div className="font-display text-sm font-bold text-[#5b9eff]">{g.roleCount ?? 0}</div>
                    <div>
                      <StatusBadge
                        status={g.builtIn > 0 ? 'inactive' : 'online'}
                        label={g.builtIn > 0 ? '内置' : '自定义'}
                      />
                    </div>
                    <div className="truncate font-mono text-[11px] text-cyan-300/70">
                      {g.updUser || '—'}
                    </div>
                    <div className="font-mono text-[11px] text-cyan-300/75">
                      {g.updTime ? formatTime(g.updTime) : '—'}
                    </div>
                    <ChevronRight className="size-3.5 justify-self-end text-cyan-300/40" />
                  </button>
                ))}
              </div>
            </StateBlock>
          </div>
        </div>

        <Pager page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} />
      </div>
    </PageShell>
  )
}
