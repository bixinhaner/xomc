import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, ListChecks } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useDictionary } from '@core/hooks/api/useSystem'
import type { DictionaryDetail } from '@core/services/api/adminApi'

import { StateBlock, MiniStat, RowHeader, FieldRow } from './_shared'

export default function DataDictionaryDetail() {
  const { type = '' } = useParams<{ type: string }>()
  const dictType = decodeURIComponent(type)
  const navigate = useNavigate()
  const { data: dict, isLoading, isError, error, isFetching } = useDictionary(dictType)

  const details = useMemo<DictionaryDetail[]>(
    () => (dict?.sysDictionaryDetails ?? []).slice().sort((a, b) => a.sort - b.sort),
    [dict]
  )
  const enabledItems = details.filter((d) => d.status).length
  const autoItems = details.filter((d) => d.origin === 'auto').length

  return (
    <PageShell
      code="F06"
      title="DICT · 字典详情"
      subtitle={dictType}
      isFetching={isFetching}
      toolbar={
        <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/system/data-dictionary')}>
          返回字典列表
        </NeonButton>
      }
    >
      <StateBlock
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={!dict}
        emptyLabel="DICTIONARY NOT FOUND · 字典不存在"
      >
        {dict ? (
          <div className="space-y-5">
            <div className="flex items-center gap-3">
              <div className="font-display text-xl font-bold text-cyan-100">{dict.name}</div>
              <span className="font-mono text-xs text-cyan-300/60">{dict.type}</span>
              <div className="ml-auto">
                <StatusBadge status={dict.status ? 'online' : 'off'} label={dict.status ? '启用' : '停用'} />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
              <MiniStat label="字典项数" value={details.length} color="#00f0ff" icon={<ListChecks className="size-3.5" />} />
              <MiniStat label="启用项" value={enabledItems} color="#00ff88" />
              <MiniStat label="自动同步项" value={autoItems} color="#a855f7" />
              <MiniStat label="来源 · SOURCE" value={dict.sourceTable ? '托管' : '手工'} color="#5b9eff" />
            </div>

            <div className="grid gap-x-8 md:grid-cols-2">
              <div>
                <FieldRow label="字典名">{dict.name}</FieldRow>
                <FieldRow label="类型 · TYPE">{dict.type}</FieldRow>
                <FieldRow label="描述">{dict.desc || '—'}</FieldRow>
              </div>
              <div>
                <FieldRow label="数据源表">{dict.sourceTable || '—'}</FieldRow>
                <FieldRow label="最近刷新">
                  {dict.lastRefreshAt ?? '—'}
                  {dict.lastRefreshStatus ? (
                    <span className="ml-2">
                      <StatusBadge
                        status={dict.lastRefreshStatus === 'ok' ? 'online' : dict.lastRefreshStatus === 'running' ? 'warning' : 'critical'}
                        label={dict.lastRefreshStatus}
                      />
                    </span>
                  ) : null}
                </FieldRow>
                <FieldRow label="刷新条数">{dict.lastRefreshCount ?? '—'}</FieldRow>
              </div>
            </div>

            <div className="glass-strong relative overflow-hidden rounded-sm">
              <div className="scanline" />
              <div className="relative max-h-[420px] overflow-auto p-3">
                {details.length > 0 ? (
                  <div className="space-y-1.5">
                    <RowHeader cols="2fr_1.6fr_0.6fr_0.6fr_0.8fr">
                      <span>标签 · LABEL</span>
                      <span>值 · VALUE</span>
                      <span>排序</span>
                      <span>状态</span>
                      <span>来源</span>
                    </RowHeader>
                    {details.map((d) => (
                      <div
                        key={d.id}
                        className="fleet-row grid grid-cols-[2fr_1.6fr_0.6fr_0.6fr_0.8fr] items-center gap-3 rounded-sm px-3 py-2"
                        style={{ ['--row-color' as never]: d.origin === 'auto' ? '#a855f7' : '#00f0ff' }}
                      >
                        <div className="truncate text-sm text-cyan-100">{d.label}</div>
                        <div className="truncate font-mono text-xs text-cyan-200">{d.value}</div>
                        <div className="font-mono text-xs text-cyan-300/70">{d.sort}</div>
                        <div>
                          <StatusBadge status={d.status ? 'online' : 'off'} label={d.status ? '启用' : '停用'} />
                        </div>
                        <div className="font-mono text-[10px] uppercase text-cyan-300/65">
                          {d.origin === 'auto' ? '同步' : '手工'}
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="py-10 text-center font-mono text-xs text-cyan-300/45">
                    NO DICT ITEMS · 该字典暂无项
                  </div>
                )}
              </div>
            </div>
          </div>
        ) : null}
      </StateBlock>
    </PageShell>
  )
}
