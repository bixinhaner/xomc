import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Loader2, RefreshCcw, FolderTree, ChevronRight, Layers3 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { useDomainTree, useDomains, useSites } from '@core/hooks/api/useTopology'
import type { Domain, Site } from '@core/types/topology'

const SITE_STATUS_COLOR: Record<string, string> = {
  active: '#00ff88',
  maintenance: '#ffaa00',
  inactive: '#525a78',
}
const SITE_STATUS_LABEL: Record<string, string> = {
  active: '在线',
  maintenance: '维护',
  inactive: '离线',
}

interface FlatRow {
  domain: Domain
  depth: number
}

/** 把域树展平成带缩进深度的列表（保留树结构次序） */
function flatten(domains: Domain[], depth = 0, out: FlatRow[] = []): FlatRow[] {
  for (const d of domains) {
    out.push({ domain: d, depth })
    if (d.children && d.children.length > 0) {
      flatten(d.children, depth + 1, out)
    }
  }
  return out
}

export default function DomainManagement() {
  const navigate = useNavigate()
  const [selectedId, setSelectedId] = useState<string>('')

  const {
    data: tree,
    isLoading,
    isError,
    error,
    isFetching,
    refetch,
  } = useDomainTree()
  // 平铺域（用于在右栏映射 domainId → name），失败不阻塞
  const { data: flatDomains } = useDomains()
  // 选中域下的站点
  const { data: sitesData, isLoading: sitesLoading } = useSites(
    selectedId ? { domainId: selectedId, pageSize: 200 } : { pageSize: 200 }
  )

  const rows = useMemo<FlatRow[]>(() => (tree ? flatten(tree) : []), [tree])

  const domainNameMap = useMemo(() => {
    const m: Record<string, string> = {}
    for (const d of flatDomains ?? []) m[d.id] = d.name
    return m
  }, [flatDomains])

  const selectedDomain = useMemo(
    () => rows.find((r) => r.domain.id === selectedId)?.domain ?? null,
    [rows, selectedId]
  )

  const sites = useMemo<Site[]>(() => sitesData?.items ?? [], [sitesData])

  const totalDevices = useMemo(
    () => rows.reduce((acc, r) => acc + (r.domain.children ? 0 : r.domain.deviceCount), 0),
    [rows]
  )

  return (
    <PageShell
      code="F06"
      title="DOMAIN · 设备域管理"
      subtitle="DEVICE GROUP TREE · 分级域 / 站点归属"
      isFetching={isFetching}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => void refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="DOMAINS · 域" value={rows.length} color="#00f0ff" />
        <Stat label="LEAF · 叶子域" value={rows.filter((r) => !r.domain.children?.length).length} color="#a855f7" />
        <Stat label="DEVICES · 设备" value={totalDevices} color="#00ff88" />
        <Stat label="SITES · 选中域站点" value={sites.length} color="#ffd400" />
      </div>

      <div className="grid grid-cols-1 gap-3 xl:grid-cols-[360px_1fr]">
        {/* ── 域树 ── */}
        <GlassPanel title="TREE · 域层级" meta={`${rows.length} NODES`} strong className="min-h-[520px]">
          <div className="max-h-[560px] overflow-auto">
            {isLoading ? (
              <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
                <Loader2 className="size-4 animate-spin" />
                <span className="font-mono text-[10px] uppercase tracking-[0.2em]">SYNCING…</span>
              </div>
            ) : isError ? (
              <div className="border border-rose-500/40 bg-rose-500/5 m-3 px-4 py-6 font-mono text-sm text-rose-300">
                <div className="font-bold uppercase tracking-[0.2em]">DOMAIN SYNC FAILED</div>
                <div className="mt-1 text-rose-200/80">
                  {error instanceof Error ? error.message : '未知错误'}
                </div>
              </div>
            ) : rows.length === 0 ? (
              <div className="py-12 text-center font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/40">
                NO DOMAINS
              </div>
            ) : (
              rows.map(({ domain, depth }) => {
                const isSel = selectedId === domain.id
                const hasChildren = (domain.children?.length ?? 0) > 0
                return (
                  <button
                    key={domain.id}
                    type="button"
                    onClick={() => setSelectedId(domain.id)}
                    className={`flex w-full items-center gap-2 border-b border-cyan-500/8 py-2 pr-3 text-left transition-colors ${
                      isSel ? 'bg-cyan-500/10' : 'hover:bg-cyan-500/5'
                    }`}
                    style={{ paddingLeft: 12 + depth * 16 }}
                  >
                    {hasChildren ? (
                      <FolderTree className="size-3.5 shrink-0 text-cyan-300/70" />
                    ) : (
                      <span className="size-1.5 shrink-0 rounded-full bg-cyan-400/70" />
                    )}
                    <span className="min-w-0 flex-1 truncate font-mono text-xs text-cyan-100/90">
                      {domain.name}
                    </span>
                    <span className="shrink-0 font-mono text-[10px] text-cyan-300/45">
                      L{domain.level}
                    </span>
                    <span className="shrink-0 font-display text-xs font-bold text-cyan-200">
                      {domain.deviceCount}
                    </span>
                  </button>
                )
              })
            )}
          </div>
        </GlassPanel>

        {/* ── 选中域：站点清单 ── */}
        <GlassPanel
          title="SITES · 域内站点"
          meta={selectedDomain ? selectedDomain.name : 'SELECT A DOMAIN'}
          className="min-h-[520px]"
        >
          {!selectedDomain ? (
            <div className="flex h-[480px] flex-col items-center justify-center gap-3 text-cyan-300/45">
              <Layers3 className="size-10 opacity-40" />
              <span className="font-mono text-xs uppercase tracking-[0.2em]">
                请从左侧选择一个设备域
              </span>
            </div>
          ) : (
            <>
              {/* 域头 */}
              <div className="flex flex-wrap items-center gap-x-6 gap-y-1 border-b border-cyan-500/10 px-4 py-3">
                <Info k="域名" v={selectedDomain.name} />
                <Info k="层级" v={`L${selectedDomain.level}`} />
                <Info
                  k="父域"
                  v={
                    selectedDomain.parentId
                      ? domainNameMap[selectedDomain.parentId] ?? selectedDomain.parentId
                      : '— 根域'
                  }
                />
                <Info k="设备数" v={String(selectedDomain.deviceCount)} />
                <Info k="子域" v={String(selectedDomain.children?.length ?? 0)} />
              </div>

              <div className="max-h-[420px] overflow-auto">
                {sitesLoading ? (
                  <div className="flex items-center justify-center gap-2 py-10 text-cyan-300/60">
                    <Loader2 className="size-4 animate-spin" />
                    <span className="font-mono text-[10px] uppercase tracking-[0.2em]">
                      SYNCING SITES…
                    </span>
                  </div>
                ) : sites.length === 0 ? (
                  <div className="py-10 text-center font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/40">
                    该域下暂无站点
                  </div>
                ) : (
                  sites.map((s) => {
                    const color = SITE_STATUS_COLOR[s.status] ?? '#525a78'
                    return (
                      <button
                        key={s.id}
                        type="button"
                        onClick={() => navigate(`/topology/site/${s.id}`)}
                        className="group flex w-full items-center gap-3 border-b border-cyan-500/8 px-4 py-2.5 text-left transition-colors hover:bg-cyan-500/5"
                      >
                        <span
                          className="size-2 shrink-0 rounded-full"
                          style={{ background: color, boxShadow: `0 0 6px ${color}` }}
                        />
                        <div className="min-w-0 flex-1">
                          <div className="truncate font-mono text-xs text-cyan-100">
                            {s.name}
                          </div>
                          <div className="truncate text-[10px] text-cyan-300/55">
                            {s.address || '—'}
                          </div>
                        </div>
                        <span
                          className="shrink-0 font-mono text-[10px]"
                          style={{ color }}
                        >
                          {SITE_STATUS_LABEL[s.status] ?? s.status}
                        </span>
                        <span className="shrink-0 font-display text-sm font-bold text-cyan-200">
                          {s.deviceCount}
                        </span>
                        <ChevronRight className="size-3.5 shrink-0 text-cyan-300/40 transition-colors group-hover:text-cyan-200" />
                      </button>
                    )
                  })
                )}
              </div>
            </>
          )}
        </GlassPanel>
      </div>
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

function Info({ k, v }: { k: string; v: string }) {
  return (
    <div>
      <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/50">
        {k}
      </div>
      <div className="font-mono text-xs text-cyan-100/90">{v}</div>
    </div>
  )
}
