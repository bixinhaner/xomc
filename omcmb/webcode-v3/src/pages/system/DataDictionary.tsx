import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { BookMarked, Database, ChevronRight, Link2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { adminApi } from '@core/services/api/adminApi'
import type { Dictionary } from '@core/services/api/adminApi'

import { StateBlock, MiniStat, RowHeader, KeywordToolbar } from './_shared'

export default function DataDictionary() {
  const navigate = useNavigate()
  const [keyword, setKeyword] = useState('')

  const { data, isLoading, isError, error, isFetching, refetch } = useQuery({
    queryKey: ['system', 'dictionaries'],
    queryFn: () => adminApi.getDictionaryList(),
    staleTime: 5 * 60 * 1000,
  })

  const all = data?.list ?? []
  const rows = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return all
    return all.filter(
      (d) =>
        d.name.toLowerCase().includes(kw) ||
        d.type.toLowerCase().includes(kw) ||
        (d.desc ?? '').toLowerCase().includes(kw)
    )
  }, [all, keyword])

  const managedCount = useMemo(() => all.filter((d) => !!d.sourceTable).length, [all])
  const enabledCount = useMemo(() => all.filter((d) => d.status).length, [all])

  return (
    <PageShell
      code="F06"
      title="DICT · 数据字典"
      subtitle="DATA DICTIONARY REGISTRY"
      isFetching={isFetching}
      bare
      toolbar={
        <KeywordToolbar
          placeholder="字典名 / 类型 / 描述"
          value={keyword}
          onChange={setKeyword}
          onRefresh={() => refetch()}
        />
      }
    >
      <div className="flex h-full flex-col gap-3">
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <MiniStat label="字典总数" value={all.length} color="#00f0ff" icon={<BookMarked className="size-3.5" />} />
          <MiniStat label="启用 · ENABLED" value={enabledCount} color="#00ff88" />
          <MiniStat label="托管 · MANAGED" value={managedCount} color="#a855f7" icon={<Database className="size-3.5" />} />
          <MiniStat label="匹配结果" value={rows.length} color="#5b9eff" />
        </div>

        <div className="glass-strong relative flex-1 min-h-0 overflow-hidden rounded-sm">
          <div className="scanline" />
          <div className="relative h-full overflow-auto p-3">
            <StateBlock
              isLoading={isLoading}
              isError={isError}
              error={error}
              isEmpty={rows.length === 0}
              emptyLabel="NO DICTIONARIES · 无字典"
            >
              <div className="space-y-1.5">
                <RowHeader cols="1.8fr_1.4fr_2fr_0.9fr_1fr_0.4fr">
                  <span>字典名 · NAME</span>
                  <span>类型 · TYPE</span>
                  <span>描述</span>
                  <span>状态</span>
                  <span>来源</span>
                  <span />
                </RowHeader>
                {rows.map((d: Dictionary) => (
                  <button
                    key={d.id}
                    type="button"
                    onClick={() => navigate(`/system/data-dictionary/${encodeURIComponent(d.type)}`)}
                    className="fleet-row grid w-full grid-cols-[1.8fr_1.4fr_2fr_0.9fr_1fr_0.4fr] items-center gap-3 rounded-sm px-3 py-2.5 text-left"
                    style={{ ['--row-color' as never]: d.sourceTable ? '#a855f7' : '#00f0ff' }}
                  >
                    <div className="truncate font-display text-sm font-bold text-cyan-100">
                      {d.name}
                    </div>
                    <div className="truncate font-mono text-xs text-cyan-100/85">{d.type}</div>
                    <div className="truncate text-xs text-cyan-100/75">{d.desc || '—'}</div>
                    <div>
                      <StatusBadge status={d.status ? 'online' : 'off'} label={d.status ? '启用' : '停用'} />
                    </div>
                    <div className="flex items-center gap-1 font-mono text-[11px] text-cyan-300/70">
                      {d.sourceTable ? (
                        <>
                          <Link2 className="size-3 text-[#a855f7]" />
                          托管
                        </>
                      ) : (
                        '手工'
                      )}
                    </div>
                    <ChevronRight className="size-3.5 justify-self-end text-cyan-300/40" />
                  </button>
                ))}
              </div>
            </StateBlock>
          </div>
        </div>
      </div>
    </PageShell>
  )
}
