import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ShieldCheck, AlertTriangle, ChevronRight } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useRoles } from '@core/hooks/api/useSystem'
import type { Role } from '@core/types/system'

import { StateBlock, MiniStat, RowHeader, KeywordToolbar, Pager } from './_shared'

export default function RolePermission() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(keyword.trim() ? { roleName: keyword.trim() } : {}),
    }),
    [page, keyword]
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useRoles(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const builtInCount = useMemo(() => rows.filter((r) => r.builtIn > 0).length, [rows])
  const boundUsers = useMemo(() => rows.reduce((acc, r) => acc + (r.userCount ?? 0), 0), [rows])
  const noGroupCount = useMemo(
    () =>
      rows.filter((r) => r.builtIn === 0 && (!r.deviceGroupIds || r.deviceGroupIds.length === 0))
        .length,
    [rows]
  )

  return (
    <PageShell
      code="F06"
      title="ROLES · 角色权限"
      subtitle="RBAC ROLES"
      isFetching={isFetching}
      bare
      toolbar={
        <KeywordToolbar
          placeholder="角色名 / 编码"
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
          <MiniStat label="角色总数" value={total.toLocaleString()} color="#00f0ff" icon={<ShieldCheck className="size-3.5" />} />
          <MiniStat label="内置角色" value={builtInCount} color="#a855f7" />
          <MiniStat label="绑定用户合计" value={boundUsers} color="#00ff88" />
          <MiniStat
            label="未绑设备组"
            value={noGroupCount}
            color={noGroupCount > 0 ? '#ff7a1a' : '#525a78'}
            icon={noGroupCount > 0 ? <AlertTriangle className="size-3.5" /> : undefined}
          />
        </div>

        <div className="glass-strong relative flex-1 min-h-0 overflow-hidden rounded-sm">
          <div className="scanline" />
          <div className="relative h-full overflow-auto p-3">
            <StateBlock
              isLoading={isLoading}
              isError={isError}
              error={error}
              isEmpty={rows.length === 0}
              emptyLabel="NO ROLES · 无角色"
            >
              <div className="space-y-1.5">
                <RowHeader cols="2fr_1.4fr_0.8fr_0.8fr_1fr_1.4fr_0.4fr">
                  <span>角色 · ROLE</span>
                  <span>编码 · CODE</span>
                  <span>用户数</span>
                  <span>类型</span>
                  <span>设备组</span>
                  <span>更新时间</span>
                  <span />
                </RowHeader>
                {rows.map((r: Role) => {
                  const noGroup =
                    r.builtIn === 0 && (!r.deviceGroupIds || r.deviceGroupIds.length === 0)
                  return (
                    <button
                      key={r.id}
                      type="button"
                      onClick={() => navigate(`/system/roles/${r.id}`)}
                      className="fleet-row grid w-full grid-cols-[2fr_1.4fr_0.8fr_0.8fr_1fr_1.4fr_0.4fr] items-center gap-3 rounded-sm px-3 py-2.5 text-left"
                      style={{ ['--row-color' as never]: r.builtIn > 0 ? '#a855f7' : '#00f0ff' }}
                    >
                      <div className="min-w-0">
                        <div className="truncate font-display text-sm font-bold text-cyan-100">
                          {r.roleName}
                        </div>
                        <div className="truncate font-mono text-[10px] text-cyan-300/55">
                          {r.description || '—'}
                        </div>
                      </div>
                      <div className="truncate font-mono text-xs text-cyan-100/85">{r.roleCode}</div>
                      <div className="font-display text-sm font-bold text-cyan-200">
                        {r.userCount ?? 0}
                      </div>
                      <div>
                        <StatusBadge
                          status={r.builtIn > 0 ? 'inactive' : 'online'}
                          label={r.builtIn > 0 ? '内置' : '自定义'}
                        />
                      </div>
                      <div className="text-xs">
                        {r.builtIn > 0 ? (
                          <span className="font-mono text-[10px] text-cyan-300/55">全部</span>
                        ) : noGroup ? (
                          <span className="chip text-[#ff7a1a]">未绑定</span>
                        ) : (
                          <span className="font-mono text-[11px] text-cyan-200">
                            {r.deviceGroupIds?.length ?? 0} 组
                          </span>
                        )}
                      </div>
                      <div className="font-mono text-[11px] text-cyan-300/75">
                        {r.updateTime ? formatTime(r.updateTime) : '—'}
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
