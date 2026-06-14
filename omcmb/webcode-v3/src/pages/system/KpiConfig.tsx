import { useMemo, useState } from 'react'
import { Activity, Gauge, AlertTriangle } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useKPIDefinitions } from '@core/hooks/api/useDashboard'
import type { KPIDefinitionItem } from '@core/types/dashboard'

import { StateBlock, MiniStat, RowHeader, KeywordToolbar } from './_shared'

const TECH_LABEL: Record<string, string> = {
  lte: 'LTE · 4G',
  nr: 'NR · 5G',
  gsm: 'GSM · 2G',
}

const PANEL_COLOR: Record<string, string> = {
  traffic: '#00f0ff',
  availability: '#00ff88',
  utilization: '#a855f7',
  accessibility: '#5b9eff',
  retainability: '#ffaa00',
  mobility: '#ff7a1a',
}

export default function KpiConfig() {
  const { data, isLoading, isError, error, isFetching } = useKPIDefinitions()
  const [tech, setTech] = useState<string>('')
  const [keyword, setKeyword] = useState('')

  const technologies = data?.technologies ?? []

  // 默认选中第一个制式
  const activeTech = tech || technologies[0]?.tech || ''

  const items = useMemo<KPIDefinitionItem[]>(() => {
    const t = technologies.find((x) => x.tech === activeTech)
    const list = t?.items ?? []
    const kw = keyword.trim().toLowerCase()
    if (!kw) return list
    return list.filter(
      (i) =>
        i.key.toLowerCase().includes(kw) ||
        i.k_code.toLowerCase().includes(kw) ||
        i.cn_name.toLowerCase().includes(kw)
    )
  }, [technologies, activeTech, keyword])

  const availableCount = items.filter((i) => i.available).length
  const needsReviewCount = items.filter((i) => i.needs_review).length

  return (
    <PageShell
      code="F06"
      title="KPI CONFIG · 指标配置"
      subtitle="KPI DEFINITION CATALOG"
      isFetching={isFetching}
      bare
      toolbar={
        <KeywordToolbar
          placeholder="symbolic / K编号 / 中文名"
          value={keyword}
          onChange={setKeyword}
        />
      }
    >
      <div className="flex h-full flex-col gap-3">
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <MiniStat label="定义总数" value={data?.total ?? 0} color="#00f0ff" icon={<Activity className="size-3.5" />} />
          <MiniStat label="制式数" value={technologies.length} color="#a855f7" icon={<Gauge className="size-3.5" />} />
          <MiniStat label="本制式可用" value={availableCount} color="#00ff88" />
          <MiniStat
            label="待复核"
            value={needsReviewCount}
            color={needsReviewCount > 0 ? '#ffaa00' : '#525a78'}
            icon={needsReviewCount > 0 ? <AlertTriangle className="size-3.5" /> : undefined}
          />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {technologies.map((t) => (
            <button
              key={t.tech}
              type="button"
              onClick={() => setTech(t.tech)}
              className={`chip flex items-center gap-1.5 transition-all ${
                activeTech === t.tech
                  ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                  : 'text-cyan-300/55 hover:text-cyan-200'
              }`}
            >
              {TECH_LABEL[t.tech] ?? t.tech.toUpperCase()}
              <span className="font-mono text-[9px] opacity-60">{t.items.length}</span>
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
              isEmpty={items.length === 0}
              emptyLabel="NO KPI DEFINITIONS · 无指标定义"
            >
              <div className="space-y-1.5">
                <RowHeader cols="1.8fr_1fr_2fr_0.8fr_1fr_0.9fr">
                  <span>Symbolic Key</span>
                  <span>K 编号</span>
                  <span>中文名</span>
                  <span>单位</span>
                  <span>Panel</span>
                  <span>状态</span>
                </RowHeader>
                {items.map((i) => {
                  const color = PANEL_COLOR[i.panel] ?? '#6b86b6'
                  return (
                    <div
                      key={`${activeTech}-${i.key}`}
                      className="fleet-row grid grid-cols-[1.8fr_1fr_2fr_0.8fr_1fr_0.9fr] items-center gap-3 rounded-sm px-3 py-2.5"
                      style={{ ['--row-color' as never]: i.available ? color : '#525a78' }}
                    >
                      <div className="truncate font-mono text-xs text-cyan-100">{i.key}</div>
                      <div className="truncate font-mono text-[11px] text-cyan-300/80">
                        {i.k_code || '—'}
                      </div>
                      <div className="truncate text-xs text-cyan-100/85">{i.cn_name || '—'}</div>
                      <div className="font-mono text-[11px] text-cyan-300/70">{i.unit || '—'}</div>
                      <div>
                        {i.panel ? (
                          <span className="chip" style={{ color }}>
                            {i.panel}
                          </span>
                        ) : (
                          '—'
                        )}
                      </div>
                      <div className="flex gap-1">
                        <StatusBadge
                          status={i.available ? 'online' : 'off'}
                          label={i.available ? '可用' : '缺口'}
                        />
                        {i.needs_review ? <StatusBadge status="warning" label="复核" /> : null}
                      </div>
                    </div>
                  )
                })}
              </div>
            </StateBlock>
          </div>
        </div>
      </div>
    </PageShell>
  )
}
