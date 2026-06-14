import { useMemo, useState } from 'react'
import { Search, RefreshCcw, FolderTree, ServerCog, Cpu } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useConfigParams } from '@core/hooks/api/useConfig'
import { useDeviceList } from '@core/hooks/api/useDevices'
import type { ConfigParam } from '@core/types/config'
import { StatCard, HudLoading, HudError, HudEmpty } from './_hud'

// ===========================================================================
// CONFIG · 参数同步
// 选择设备 → 拉取真实参数 → 按 category 分组成树 → 选组对比当前/默认值
// 数据全走 useConfigParams（真实 deviceApi.getParameters）。
// ===========================================================================
export default function ParamSync() {
  const [deviceId, setDeviceId] = useState('')
  const [deviceKw, setDeviceKw] = useState('')
  const [category, setCategory] = useState('')
  const [paramKw, setParamKw] = useState('')

  const devicesQ = useDeviceList({ page: 1, pageSize: 200 })
  const devices = devicesQ.data?.items ?? []
  const dkw = deviceKw.trim().toLowerCase()
  const filteredDevices = dkw
    ? devices.filter(
        (d) => d.sn?.toLowerCase().includes(dkw) || d.name?.toLowerCase().includes(dkw),
      )
    : devices

  const effectiveDeviceId = deviceId || devices[0]?.id || ''
  const paramsQ = useConfigParams({ deviceId: effectiveDeviceId, page: 1, pageSize: 500 })
  const params: ConfigParam[] = (paramsQ.data?.items as ConfigParam[] | undefined) ?? []

  // 按 category 分组
  const grouped = useMemo(() => {
    const g: Record<string, ConfigParam[]> = {}
    for (const p of params) {
      const k = p.category || '未分类'
      ;(g[k] ??= []).push(p)
    }
    return g
  }, [params])

  const categories = Object.keys(grouped)
  const activeCategory = category && grouped[category] ? category : categories[0] || ''
  const catParams = activeCategory ? grouped[activeCategory] ?? [] : params

  const pkw = paramKw.trim().toLowerCase()
  const rows = pkw
    ? catParams.filter(
        (p) =>
          p.paramName?.toLowerCase().includes(pkw) ||
          p.paramCode?.toLowerCase().includes(pkw),
      )
    : catParams

  const driftCount = params.filter(
    (p) => String(p.paramValue) !== String(p.defaultValue),
  ).length

  const selectedDevice = devices.find((d) => d.id === effectiveDeviceId) ?? null

  return (
    <PageShell
      code="F02"
      title="PARAM SYNC · 参数同步"
      subtitle="TR-069 DEVICE PARAMETER TREE"
      isFetching={devicesQ.isFetching || paramsQ.isFetching}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => void paramsQ.refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      <div className="mb-3 grid grid-cols-4 gap-3">
        <StatCard label="DEVICES · 设备" value={devicesQ.data?.total ?? devices.length} color="#00f0ff" />
        <StatCard label="PARAMS · 参数" value={params.length} color="#a855f7" />
        <StatCard label="GROUPS · 分组" value={categories.length} color="#5b9eff" />
        <StatCard
          label="DRIFT · 偏离默认"
          value={driftCount}
          color={driftCount > 0 ? '#ffaa00' : '#00ff88'}
        />
      </div>

      <div className="grid grid-cols-[300px_220px_1fr] gap-3">
        {/* 设备选择 */}
        <GlassPanel title="DEVICES · 设备" className="min-h-0">
          <div className="p-2">
            <div className="relative mb-2">
              <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
              <input
                className="neon-input w-full pl-9"
                placeholder="SN / 名称"
                value={deviceKw}
                onChange={(e) => setDeviceKw(e.target.value)}
              />
            </div>
            <div className="max-h-[54vh] space-y-1 overflow-auto">
              {devicesQ.isLoading ? (
                <HudLoading />
              ) : filteredDevices.length === 0 ? (
                <HudEmpty icon={Cpu} text="无设备" />
              ) : (
                filteredDevices.map((d) => {
                  const on = d.id === effectiveDeviceId
                  return (
                    <button
                      key={d.id}
                      type="button"
                      onClick={() => {
                        setDeviceId(d.id)
                        setCategory('')
                      }}
                      className={`flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left transition-colors ${
                        on ? 'bg-cyan-500/12' : 'hover:bg-cyan-500/6'
                      }`}
                    >
                      <span
                        className="size-1.5 shrink-0 rounded-full"
                        style={{
                          background: d.isOnline ? '#00ff88' : '#525a78',
                          boxShadow: d.isOnline ? '0 0 6px #00ff88' : 'none',
                        }}
                      />
                      <span className="min-w-0 flex-1">
                        <span className="block truncate font-mono text-xs text-cyan-100">{d.sn}</span>
                        {d.name && d.name !== d.sn ? (
                          <span className="block truncate text-[10px] text-cyan-300/50">{d.name}</span>
                        ) : null}
                      </span>
                    </button>
                  )
                })
              )}
            </div>
          </div>
        </GlassPanel>

        {/* 分组树 */}
        <GlassPanel title="GROUPS · 参数组" className="min-h-0">
          <div className="max-h-[58vh] overflow-auto p-2">
            {paramsQ.isLoading ? (
              <HudLoading />
            ) : categories.length === 0 ? (
              <HudEmpty icon={FolderTree} text="无参数组" />
            ) : (
              categories.map((c) => {
                const on = c === activeCategory
                const drift = grouped[c].filter(
                  (p) => String(p.paramValue) !== String(p.defaultValue),
                ).length
                return (
                  <button
                    key={c}
                    type="button"
                    onClick={() => setCategory(c)}
                    className={`flex w-full items-center justify-between gap-2 rounded-sm px-2 py-1.5 text-left transition-colors ${
                      on ? 'bg-cyan-500/12' : 'hover:bg-cyan-500/6'
                    }`}
                  >
                    <span className="min-w-0 flex items-center gap-2">
                      <FolderTree className="size-3.5 shrink-0 text-cyan-300/60" />
                      <span className="truncate text-xs text-cyan-100/90">{c}</span>
                    </span>
                    <span className="shrink-0 font-mono text-[10px] text-cyan-300/45">
                      {grouped[c].length}
                      {drift > 0 ? <span className="ml-1 text-[#ffaa00]">·{drift}</span> : null}
                    </span>
                  </button>
                )
              })
            )}
          </div>
        </GlassPanel>

        {/* 参数明细 */}
        <GlassPanel
          title={selectedDevice ? `PARAMS · ${selectedDevice.sn}` : 'PARAMS · 参数'}
          meta={activeCategory ? `${activeCategory} · ${rows.length}` : undefined}
          className="min-h-0"
        >
          <div className="border-b border-cyan-500/15 p-2">
            <div className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
              <input
                className="neon-input w-full pl-9"
                placeholder="按参数名 / 路径筛选"
                value={paramKw}
                onChange={(e) => setParamKw(e.target.value)}
              />
            </div>
          </div>
          {paramsQ.isLoading ? (
            <HudLoading />
          ) : paramsQ.isError ? (
            <HudError error={paramsQ.error} />
          ) : rows.length === 0 ? (
            <HudEmpty icon={ServerCog} text="无参数 · 选择设备查看" />
          ) : (
            <div className="max-h-[48vh] overflow-auto">
              <div className="grid grid-cols-[1.6fr_1fr_1fr_90px] gap-3 border-b border-cyan-500/15 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/45">
                <span>PARAM</span>
                <span>DEFAULT / 默认</span>
                <span>CURRENT / 当前</span>
                <span className="text-right">SYNC</span>
              </div>
              {rows.map((p) => {
                const drift = String(p.paramValue) !== String(p.defaultValue)
                return (
                  <div
                    key={p.id}
                    className="grid grid-cols-[1.6fr_1fr_1fr_90px] items-center gap-3 border-b border-cyan-500/8 px-3 py-2 hover:bg-cyan-500/5"
                  >
                    <div className="min-w-0">
                      <div className="truncate text-xs text-cyan-100/90">{p.paramName}</div>
                      <code className="block truncate font-mono text-[10px] text-cyan-300/50">
                        {p.paramCode}
                      </code>
                    </div>
                    <div className="font-mono text-[11px] text-cyan-300/70">
                      {String(p.defaultValue)}
                      {p.unit ? <span className="text-cyan-300/40"> {p.unit}</span> : null}
                    </div>
                    <div
                      className="font-mono text-[11px]"
                      style={{ color: drift ? '#ffaa00' : 'rgba(190,225,255,0.7)' }}
                    >
                      {String(p.paramValue)}
                      {p.unit ? <span className="opacity-50"> {p.unit}</span> : null}
                    </div>
                    <div className="flex justify-end">
                      {drift ? (
                        <StatusBadge status="warning" label="偏离" />
                      ) : (
                        <StatusBadge status="ok" label="同步" />
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </GlassPanel>
      </div>
    </PageShell>
  )
}
