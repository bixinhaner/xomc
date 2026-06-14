import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Loader2, RefreshCcw, Search, ChevronRight, MapPin } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { useSites, useDomains } from '@core/hooks/api/useTopology'
import type { Site, SiteStatus } from '@core/types/topology'

const STATUS_COLOR: Record<string, string> = {
  active: '#00ff88',
  maintenance: '#ffaa00',
  inactive: '#525a78',
}
const STATUS_LABEL: Record<string, string> = {
  active: '在线',
  maintenance: '维护',
  inactive: '离线',
}
const STATUS_OPTIONS: SiteStatus[] = ['active', 'maintenance', 'inactive']

export default function SiteManagement() {
  const navigate = useNavigate()
  const [keyword, setKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<SiteStatus | ''>('')
  const [domainFilter, setDomainFilter] = useState<string>('')

  const { data: domainsData } = useDomains()
  const {
    data: sitesData,
    isLoading,
    isError,
    error,
    isFetching,
    refetch,
  } = useSites({ pageSize: 500 })

  const domainNameMap = useMemo(() => {
    const m: Record<string, string> = {}
    for (const d of domainsData ?? []) m[d.id] = d.name
    return m
  }, [domainsData])

  const sites = useMemo<Site[]>(() => sitesData?.items ?? [], [sitesData])

  const filtered = useMemo(() => {
    const kw = keyword.trim()
    return sites.filter((s) => {
      if (kw && !s.name.includes(kw) && !(s.address ?? '').includes(kw)) return false
      if (statusFilter && s.status !== statusFilter) return false
      if (domainFilter && s.domainId !== domainFilter) return false
      return true
    })
  }, [sites, keyword, statusFilter, domainFilter])

  const statusCount = useMemo(() => {
    const acc = { active: 0, maintenance: 0, inactive: 0 }
    for (const s of sites) {
      if (s.status in acc) acc[s.status as keyof typeof acc] += 1
    }
    return acc
  }, [sites])

  const totalDevices = useMemo(
    () => filtered.reduce((acc, s) => acc + (s.deviceCount || 0), 0),
    [filtered]
  )

  return (
    <PageShell
      code="F06"
      title="SITES · 站点管理"
      subtitle="PHYSICAL SITES · 地理坐标 / 设备归属"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-56 pl-9"
              placeholder="站点名称 / 地址"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          {(['', ...STATUS_OPTIONS] as const).map((s) => (
            <button
              key={s || 'all'}
              type="button"
              onClick={() => setStatusFilter(s)}
              className={`chip transition-all ${
                statusFilter === s
                  ? 'shadow-[0_0_10px_currentColor]'
                  : 'opacity-55 hover:opacity-100'
              }`}
              style={{ color: s ? STATUS_COLOR[s] : '#00f0ff' }}
            >
              {s ? STATUS_LABEL[s] : 'ALL'}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => void refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="SITES · 站点" value={sites.length} color="#00f0ff" />
        <Stat label="ACTIVE · 在线" value={statusCount.active} color="#00ff88" />
        <Stat label="MAINT · 维护" value={statusCount.maintenance} color="#ffaa00" />
        <Stat label="DEVICES · 设备" value={totalDevices} color="#a855f7" />
      </div>

      {/* 域筛选 chips */}
      {(domainsData?.length ?? 0) > 0 && (
        <div className="mb-3 flex flex-wrap items-center gap-2">
          <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/55">
            DOMAIN
          </span>
          <button
            type="button"
            onClick={() => setDomainFilter('')}
            className={`chip text-[10px] transition-all ${
              domainFilter === ''
                ? 'text-cyan-200 shadow-[0_0_8px_currentColor]'
                : 'text-cyan-300/45 hover:text-cyan-200'
            }`}
          >
            ALL
          </button>
          {(domainsData ?? []).map((d) => (
            <button
              key={d.id}
              type="button"
              onClick={() => setDomainFilter((prev) => (prev === d.id ? '' : d.id))}
              className={`chip text-[10px] transition-all ${
                domainFilter === d.id
                  ? 'text-cyan-200 shadow-[0_0_8px_currentColor]'
                  : 'text-cyan-300/45 hover:text-cyan-200'
              }`}
            >
              {d.name}
            </button>
          ))}
        </div>
      )}

      <GlassPanel title="SITE LIST · 站点清单" meta={`${filtered.length} / ${sites.length}`} strong>
        {/* 列头 */}
        <div className="grid grid-cols-[12px_1.8fr_2fr_1.2fr_120px_90px_24px] items-center gap-3 border-b border-cyan-500/15 px-4 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
          <span />
          <span>站点 / 域</span>
          <span>地址</span>
          <span>经纬度</span>
          <span className="text-right">设备数</span>
          <span className="text-right">状态</span>
          <span />
        </div>

        <div className="max-h-[560px] overflow-auto">
          {isLoading ? (
            <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
              <Loader2 className="size-4 animate-spin" />
              <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING SITES…</span>
            </div>
          ) : isError ? (
            <div className="m-3 border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
              <div className="font-bold uppercase tracking-[0.2em]">SITE SYNC FAILED</div>
              <div className="mt-1 text-rose-200/80">
                {error instanceof Error ? error.message : '未知错误'}
              </div>
            </div>
          ) : filtered.length === 0 ? (
            <div className="py-12 text-center font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/40">
              {sites.length === 0 ? 'NO SITES' : 'NO MATCH · 无匹配站点'}
            </div>
          ) : (
            filtered.map((s) => {
              const color = STATUS_COLOR[s.status] ?? '#525a78'
              return (
                <button
                  key={s.id}
                  type="button"
                  onClick={() => navigate(`/topology/site/${s.id}`)}
                  className="group grid w-full grid-cols-[12px_1.8fr_2fr_1.2fr_120px_90px_24px] items-center gap-3 border-b border-cyan-500/8 px-4 py-2.5 text-left transition-colors hover:bg-cyan-500/5"
                >
                  <span
                    className="size-2.5 rounded-full"
                    style={{ background: color, boxShadow: `0 0 8px ${color}` }}
                  />
                  <div className="min-w-0">
                    <div className="truncate font-display text-sm font-bold text-cyan-100">
                      {s.name}
                    </div>
                    <div className="truncate font-mono text-[10px] text-cyan-300/55">
                      {domainNameMap[s.domainId] ?? s.domainId ?? '—'}
                    </div>
                  </div>
                  <div className="min-w-0 truncate text-xs text-cyan-100/80">
                    {s.address || '—'}
                  </div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/65">
                    {s.longitude != null && s.latitude != null
                      ? `${s.longitude.toFixed(3)}, ${s.latitude.toFixed(3)}`
                      : '— 无坐标'}
                  </div>
                  <div className="text-right font-display text-base font-bold text-cyan-200">
                    {s.deviceCount}
                  </div>
                  <div className="text-right font-mono text-[10px]" style={{ color }}>
                    {STATUS_LABEL[s.status] ?? s.status}
                  </div>
                  <ChevronRight className="size-3.5 text-cyan-300/40 transition-colors group-hover:text-cyan-200" />
                </button>
              )
            })
          )}
        </div>

        <div className="flex items-center gap-2 border-t border-cyan-500/10 px-4 py-2 font-mono text-[10px] text-cyan-300/45">
          <MapPin className="size-3" />
          点击任意行 ⟶ 进入站点详情
        </div>
      </GlassPanel>
    </PageShell>
  )
}

function Stat({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: color }}
    >
      <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
        {label}
      </div>
      <div
        className="font-display text-2xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value.toLocaleString()}
      </div>
    </div>
  )
}
