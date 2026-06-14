import { useMemo, useState } from 'react'
import { Network, Boxes } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { useApiEndpoints, useApiGroups } from '@core/hooks/api/useSystem'
import type { ApiEndpoint } from '@core/types/system'

import { StateBlock, MiniStat, RowHeader, KeywordToolbar, Pager } from './_shared'

const METHOD_COLOR: Record<string, string> = {
  GET: '#00ff88',
  POST: '#00f0ff',
  PUT: '#ffaa00',
  PATCH: '#a855f7',
  DELETE: '#ff2d6f',
}

export default function ApiManagement() {
  const [page, setPage] = useState(1)
  const pageSize = 30
  const [keyword, setKeyword] = useState('')
  const [group, setGroup] = useState('')

  const { data: groups } = useApiGroups()

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(keyword.trim() ? { path: keyword.trim() } : {}),
      ...(group ? { apiGroup: group } : {}),
    }),
    [page, keyword, group]
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useApiEndpoints(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <PageShell
      code="F06"
      title="API · 接口管理"
      subtitle="API ENDPOINT REGISTRY"
      isFetching={isFetching}
      bare
      toolbar={
        <KeywordToolbar
          placeholder="路径 / PATH"
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
          <MiniStat label="接口总数" value={total.toLocaleString()} color="#00f0ff" icon={<Network className="size-3.5" />} />
          <MiniStat label="分组数 · GROUPS" value={groups?.length ?? 0} color="#a855f7" icon={<Boxes className="size-3.5" />} />
          <MiniStat label="本页条数" value={rows.length} color="#00ff88" />
          <MiniStat label="页码" value={`${page}/${totalPages}`} color="#5b9eff" />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={() => {
              setGroup('')
              setPage(1)
            }}
            className={`chip transition-all ${
              group === '' ? 'text-cyan-200 shadow-[0_0_10px_currentColor]' : 'text-cyan-300/55 hover:text-cyan-200'
            }`}
          >
            ALL
          </button>
          {(groups ?? []).map((g) => (
            <button
              key={g}
              type="button"
              onClick={() => {
                setGroup(g)
                setPage(1)
              }}
              className={`chip transition-all ${
                group === g ? 'text-cyan-200 shadow-[0_0_10px_currentColor]' : 'text-cyan-300/55 hover:text-cyan-200'
              }`}
            >
              {g}
            </button>
          ))}
        </div>

        <div className="glass-strong relative flex-1 min-h-0 overflow-hidden rounded-sm">
          <div className="scanline" />
          <div className="relative h-full overflow-auto p-3">
            <StateBlock
              isLoading={isLoading}
              isError={isError}
              error={error}
              isEmpty={rows.length === 0}
              emptyLabel="NO ENDPOINTS · 无接口"
            >
              <div className="space-y-1.5">
                <RowHeader cols="0.7fr_2.4fr_1.6fr_1fr_1fr">
                  <span>方法</span>
                  <span>路径 · PATH</span>
                  <span>名称 · NAME</span>
                  <span>分组</span>
                  <span>模块</span>
                </RowHeader>
                {rows.map((api: ApiEndpoint) => {
                  const color = METHOD_COLOR[api.method?.toUpperCase()] ?? '#6b86b6'
                  return (
                    <div
                      key={api.id}
                      className="fleet-row grid grid-cols-[0.7fr_2.4fr_1.6fr_1fr_1fr] items-center gap-3 rounded-sm px-3 py-2.5"
                      style={{ ['--row-color' as never]: color }}
                    >
                      <div>
                        <span className="chip font-mono" style={{ color }}>
                          {api.method?.toUpperCase()}
                        </span>
                      </div>
                      <div className="truncate font-mono text-xs text-cyan-100">{api.path}</div>
                      <div className="truncate text-xs text-cyan-100/85">{api.name || '—'}</div>
                      <div className="truncate font-mono text-[11px] text-cyan-300/70">
                        {api.apiGroup || '—'}
                      </div>
                      <div className="truncate font-mono text-[11px] text-cyan-300/70">
                        {api.module || '—'}
                      </div>
                    </div>
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
