import { useMemo, useState } from 'react'
import { Search, RefreshCcw, Layers, FolderTree, Cpu, ServerCog } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { Sparkline } from '@/components/viz/Sparkline'
import { useConfigParams } from '@core/hooks/api/useConfig'
import { useDeviceList } from '@core/hooks/api/useDevices'
import type { ConfigParam } from '@core/types/config'
import { StatCard, HudLoading, HudError, HudEmpty } from './_hud'

// ===========================================================================
// CONFIG · 按参数类批量配置
// 选参考设备 → 真实参数按 category 聚成「参数类」→ 选类查看该类下全部参数
// 用于按类批量编排（与 v1 BatchParamClass 树形选类一致）。
// ===========================================================================
export default function BatchParamClass() {
  const [deviceId, setDeviceId] = useState('')
  const [deviceKw, setDeviceKw] = useState('')
  const [category, setCategory] = useState('')

  const devicesQ = useDeviceList({ page: 1, pageSize: 200 })
  const devices = devicesQ.data?.items ?? []
  const dkw = deviceKw.trim().toLowerCase()
  const filteredDevices = dkw
    ? devices.filter((d) => d.sn?.toLowerCase().includes(dkw) || d.name?.toLowerCase().includes(dkw))
    : devices

  const refDeviceId = deviceId || devices[0]?.id || ''
  const paramsQ = useConfigParams({ deviceId: refDeviceId, page: 1, pageSize: 500 })
  const params: ConfigParam[] = (paramsQ.data?.items as ConfigParam[] | undefined) ?? []

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
  const classParams = activeCategory ? grouped[activeCategory] ?? [] : []
  const writableInClass = classParams.filter((p) => !p.readonly).length

  // 每个类的可写占比，给左侧 sparkline 一点 HUD 质感
  const writeRatios = categories.map((c) => {
    const arr = grouped[c]
    return arr.length ? Math.round((arr.filter((p) => !p.readonly).length / arr.length) * 100) : 0
  })

  const refDevice = devices.find((d) => d.id === refDeviceId) ?? null

  return (
    <PageShell
      code="F02"
      title="BATCH BY CLASS · 按参数类批量"
      subtitle="PARAMETER CLASS GROUPING"
      isFetching={devicesQ.isFetching || paramsQ.isFetching}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => void paramsQ.refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      <div className="mb-3 grid grid-cols-4 gap-3">
        <StatCard label="REF DEVICE · 参考设备" value={refDevice?.sn ?? '—'} color="#00f0ff" />
        <StatCard label="CLASSES · 参数类" value={categories.length} color="#a855f7" />
        <StatCard label="PARAMS · 参数" value={params.length} color="#5b9eff" />
        <StatCard label="WRITABLE IN CLASS · 当前类可写" value={writableInClass} color="#00ff88" />
      </div>

      <div className="grid grid-cols-[300px_240px_1fr] gap-3">
        <GlassPanel title="DEVICES · 参考设备" className="min-h-0">
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
                  const on = d.id === refDeviceId
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
                      <Cpu className="size-3.5 shrink-0 text-cyan-300/55" />
                      <span className="min-w-0 flex-1">
                        <span className="block truncate font-mono text-xs text-cyan-100">{d.sn}</span>
                        {d.productClass ? (
                          <span className="block truncate text-[10px] text-cyan-300/45">{d.productClass}</span>
                        ) : null}
                      </span>
                    </button>
                  )
                })
              )}
            </div>
          </div>
        </GlassPanel>

        <GlassPanel title="PARAM CLASS · 参数类" className="min-h-0">
          <div className="max-h-[58vh] overflow-auto p-2">
            {paramsQ.isLoading ? (
              <HudLoading />
            ) : categories.length === 0 ? (
              <HudEmpty icon={FolderTree} text="无参数类" />
            ) : (
              categories.map((c, i) => {
                const on = c === activeCategory
                return (
                  <button
                    key={c}
                    type="button"
                    onClick={() => setCategory(c)}
                    className={`flex w-full items-center gap-2 rounded-sm px-2 py-2 text-left transition-colors ${
                      on ? 'bg-cyan-500/12' : 'hover:bg-cyan-500/6'
                    }`}
                  >
                    <Layers className="size-3.5 shrink-0 text-cyan-300/60" />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-xs text-cyan-100/90">{c}</span>
                      <span className="block font-mono text-[10px] text-cyan-300/45">
                        {grouped[c].length} PARAMS · {writeRatios[i]}% 可写
                      </span>
                    </span>
                  </button>
                )
              })
            )}
          </div>
        </GlassPanel>

        <GlassPanel
          title={activeCategory ? `CLASS · ${activeCategory}` : 'CLASS · 参数类明细'}
          meta={
            classParams.length
              ? `${classParams.length} PARAMS · ${writableInClass} 可写`
              : undefined
          }
          className="min-h-0"
        >
          {classParams.length > 0 ? (
            <div className="flex items-center gap-3 border-b border-cyan-500/12 px-3 py-2">
              <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/45">
                CLASS WRITE PROFILE
              </span>
              <Sparkline
                data={classParams.map((p) => (p.readonly ? 0 : 100))}
                width={160}
                height={20}
                color="#00ff88"
                fill
              />
            </div>
          ) : null}
          {paramsQ.isLoading ? (
            <HudLoading />
          ) : paramsQ.isError ? (
            <HudError error={paramsQ.error} />
          ) : classParams.length === 0 ? (
            <HudEmpty icon={ServerCog} text="无参数 · 选择参数类" />
          ) : (
            <div className="max-h-[48vh] overflow-auto">
              <div className="grid grid-cols-[1.6fr_1fr_1fr_90px] gap-3 border-b border-cyan-500/15 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/45">
                <span>PARAM</span>
                <span>DEFAULT</span>
                <span>VALUE / 当前</span>
                <span className="text-right">RW</span>
              </div>
              {classParams.map((p) => (
                <div
                  key={p.id}
                  className="grid grid-cols-[1.6fr_1fr_1fr_90px] items-center gap-3 border-b border-cyan-500/8 px-3 py-2 hover:bg-cyan-500/5"
                >
                  <div className="min-w-0">
                    <div className="truncate text-xs text-cyan-100/90">{p.paramName}</div>
                    <code className="block truncate font-mono text-[10px] text-cyan-300/50">{p.paramCode}</code>
                  </div>
                  <div className="font-mono text-[11px] text-cyan-300/70">{String(p.defaultValue)}</div>
                  <div className="font-mono text-[11px] text-cyan-200/85">{String(p.paramValue)}</div>
                  <div className="flex justify-end">
                    {p.readonly ? (
                      <StatusBadge status="off" label="只读" />
                    ) : (
                      <StatusBadge status="ok" label="可写" />
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </GlassPanel>
      </div>
    </PageShell>
  )
}
