import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, FolderTree, ChevronRight, Layers } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { useDeviceGroups } from '@core/hooks/api/useDevices'
import type { DeviceGroup } from '@core/types/device'
import { StateGate, StatCard } from './_shared'

export default function FleetDeviceGroup() {
  const navigate = useNavigate()
  const { data, isLoading, isError, error, isFetching, refetch } = useDeviceGroups()
  const groups: DeviceGroup[] = useMemo(() => data?.groups ?? [], [data])
  const reportedTotal = data?.stats?.totalDevices

  // 组织为两级树：L1（parentId == null）→ children
  const tree = useMemo(() => {
    const roots = groups.filter((g) => !g.parentId)
    return roots.map((root) => ({
      root,
      children: groups.filter((g) => g.parentId === root.id),
    }))
  }, [groups])

  const totalDevices = useMemo(
    () => groups.reduce((acc, g) => (g.parentId ? acc + g.deviceCount : acc), 0),
    [groups]
  )

  return (
    <PageShell
      code="F06"
      title="GROUPS · 设备分组"
      subtitle="HIERARCHY · TWO-LEVEL CLUSTERS"
      isFetching={isFetching}
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-3">
        <StatCard label="ROOT GROUPS" value={tree.length} color="#00f0ff" />
        <StatCard label="TOTAL GROUPS" value={groups.length} color="#a855f7" />
        <StatCard label="DEVICES IN GROUPS" value={reportedTotal ?? totalDevices} color="#00ff88" />
      </div>

      <StateGate
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={groups.length === 0}
        loadingLabel="LOADING GROUPS…"
        emptyLabel="NO GROUPS DEFINED"
      >
        <div className="space-y-3">
          {tree.map(({ root, children }) => (
            <GlassPanel
              key={root.id}
              title={
                <span className="flex items-center gap-2">
                  <FolderTree className="size-3.5" />
                  {root.name}
                </span>
              }
              meta={`${root.deviceCount} DEV${root.builtIn ? ' · 内置' : ''}`}
            >
              {children.length === 0 ? (
                <div className="px-3 py-3 font-mono text-[11px] text-cyan-300/40">
                  NO SUBGROUPS
                </div>
              ) : (
                <div className="divide-y divide-cyan-500/8">
                  {children.map((c) => (
                    <button
                      key={c.id}
                      type="button"
                      onClick={() => navigate(`/fleet?groupId=${c.id}`)}
                      className="flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors hover:bg-cyan-500/5"
                    >
                      <Layers className="size-3.5 text-cyan-400/70" />
                      <div className="min-w-0 flex-1">
                        <div className="truncate font-mono text-xs text-cyan-100">{c.name}</div>
                        {c.networkType || c.productClass ? (
                          <div className="truncate font-mono text-[10px] text-cyan-300/50">
                            {[c.networkType, c.productClass].filter(Boolean).join(' · ')}
                          </div>
                        ) : null}
                      </div>
                      <span className="font-display text-sm font-bold text-cyan-200">
                        {c.deviceCount}
                      </span>
                      <ChevronRight className="size-3.5 text-cyan-300/40" />
                    </button>
                  ))}
                </div>
              )}
            </GlassPanel>
          ))}
        </div>
      </StateGate>
    </PageShell>
  )
}
