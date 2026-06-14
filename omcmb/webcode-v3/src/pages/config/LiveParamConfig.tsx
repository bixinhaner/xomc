import { useState } from 'react'
import { Search, RefreshCcw, Cpu, ServerCog, Pencil, X, Save, Loader2, RotateCcw } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useConfigParams, useUpdateConfigParam } from '@core/hooks/api/useConfig'
import { useDeviceList } from '@core/hooks/api/useDevices'
import type { ConfigParam } from '@core/types/config'
import { StatCard, HudLoading, HudError, HudEmpty } from './_hud'

// ===========================================================================
// CONFIG · 在线参数配置
// 选设备 → 拉真实参数 → 直接改可写参数 → useUpdateConfigParam 下发（pushConfig）
// ===========================================================================
export default function LiveParamConfig() {
  const [deviceId, setDeviceId] = useState('')
  const [deviceKw, setDeviceKw] = useState('')
  const [keyword, setKeyword] = useState('')
  const [editing, setEditing] = useState<ConfigParam | null>(null)

  const devicesQ = useDeviceList({ page: 1, pageSize: 200 })
  const devices = devicesQ.data?.items ?? []
  const dkw = deviceKw.trim().toLowerCase()
  const filteredDevices = dkw
    ? devices.filter((d) => d.sn?.toLowerCase().includes(dkw) || d.name?.toLowerCase().includes(dkw))
    : devices

  const effectiveDeviceId = deviceId || devices[0]?.id || ''
  const paramsQ = useConfigParams({ deviceId: effectiveDeviceId, page: 1, pageSize: 500 })
  const params: ConfigParam[] = (paramsQ.data?.items as ConfigParam[] | undefined) ?? []

  const kw = keyword.trim().toLowerCase()
  const rows = kw
    ? params.filter(
        (p) => p.paramName?.toLowerCase().includes(kw) || p.paramCode?.toLowerCase().includes(kw),
      )
    : params

  const writable = params.filter((p) => !p.readonly).length
  const selectedDevice = devices.find((d) => d.id === effectiveDeviceId) ?? null

  return (
    <PageShell
      code="F02"
      title="LIVE PARAM · 在线参数配置"
      subtitle="TR-069 SET-PARAMETER-VALUES"
      isFetching={devicesQ.isFetching || paramsQ.isFetching}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => void paramsQ.refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      <div className="mb-3 grid grid-cols-3 gap-3">
        <StatCard label="DEVICES · 设备" value={devicesQ.data?.total ?? devices.length} color="#00f0ff" />
        <StatCard label="PARAMS · 参数" value={params.length} color="#a855f7" />
        <StatCard label="WRITABLE · 可写" value={writable} color="#00ff88" />
      </div>

      <div className="grid grid-cols-[300px_1fr] gap-3">
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
            <div className="max-h-[58vh] space-y-1 overflow-auto">
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
                        setEditing(null)
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

        <GlassPanel
          title={selectedDevice ? `PARAMS · ${selectedDevice.sn}` : 'PARAMS · 参数'}
          meta={`${rows.length} ROWS`}
          className="min-h-0"
        >
          <div className="border-b border-cyan-500/15 p-2">
            <div className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
              <input
                className="neon-input w-full pl-9"
                placeholder="按参数名 / 路径筛选"
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
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
            <div className="max-h-[52vh] overflow-auto">
              <div className="grid grid-cols-[1.6fr_1fr_120px_90px] gap-3 border-b border-cyan-500/15 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/45">
                <span>PARAM</span>
                <span>VALUE / 当前值</span>
                <span>TYPE</span>
                <span className="text-right">EDIT</span>
              </div>
              {rows.map((p) => (
                <div
                  key={p.id}
                  className="grid grid-cols-[1.6fr_1fr_120px_90px] items-center gap-3 border-b border-cyan-500/8 px-3 py-2 hover:bg-cyan-500/5"
                >
                  <div className="min-w-0">
                    <div className="truncate text-xs text-cyan-100/90">{p.paramName}</div>
                    <code className="block truncate font-mono text-[10px] text-cyan-300/50">{p.paramCode}</code>
                  </div>
                  <div className="truncate font-mono text-[11px] text-cyan-200/85">
                    {String(p.paramValue)}
                    {p.unit ? <span className="text-cyan-300/40"> {p.unit}</span> : null}
                  </div>
                  <div>
                    <span className="chip text-[#5b9eff]">{p.paramType}</span>
                  </div>
                  <div className="flex justify-end">
                    {p.readonly ? (
                      <StatusBadge status="off" label="只读" />
                    ) : (
                      <NeonButton icon={<Pencil />} onClick={() => setEditing(p)}>
                        改
                      </NeonButton>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </GlassPanel>
      </div>

      {editing && effectiveDeviceId ? (
        <EditModal
          param={editing}
          deviceId={effectiveDeviceId}
          onClose={() => setEditing(null)}
        />
      ) : null}
    </PageShell>
  )
}

function EditModal({
  param,
  deviceId,
  onClose,
}: {
  param: ConfigParam
  deviceId: string
  onClose: () => void
}) {
  const [value, setValue] = useState(String(param.paramValue))
  const [err, setErr] = useState('')
  const update = useUpdateConfigParam()

  const submit = () => {
    setErr('')
    update.mutate(
      { id: param.paramCode, value, deviceId },
      {
        onSuccess: () => onClose(),
        onError: (e: Error) => setErr(e.message || '下发失败'),
      },
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-[#02030a]/75 backdrop-blur-sm">
      <div className="w-[520px] max-w-[92vw]">
        <GlassPanel strong title={`EDIT · ${param.paramName}`} meta={param.paramType}>
          <div className="space-y-3 p-4">
            <code className="block break-all font-mono text-[11px] text-cyan-300/60">{param.paramCode}</code>
            {param.description ? (
              <div className="text-xs text-cyan-300/70">{param.description}</div>
            ) : null}
            <div>
              <div className="mb-1 flex items-center justify-between font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/45">
                <span>NEW VALUE</span>
                <span>
                  默认 {String(param.defaultValue)}
                  {param.unit ? ` ${param.unit}` : ''}
                </span>
              </div>
              {param.paramType === 'enum' && param.enumOptions?.length ? (
                <select
                  className="neon-input w-full"
                  value={value}
                  onChange={(e) => setValue(e.target.value)}
                >
                  {param.enumOptions.map((o) => (
                    <option key={String(o.value)} value={String(o.value)}>
                      {o.label}
                    </option>
                  ))}
                </select>
              ) : (
                <input
                  className="neon-input w-full"
                  type={param.paramType === 'number' || param.paramType === 'range' ? 'number' : 'text'}
                  value={value}
                  min={param.minValue}
                  max={param.maxValue}
                  onChange={(e) => setValue(e.target.value)}
                />
              )}
            </div>
            {err ? (
              <div className="border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">
                {err}
              </div>
            ) : null}
          </div>
          <div className="flex items-center justify-between border-t border-cyan-500/15 px-4 py-3">
            <NeonButton
              icon={<RotateCcw />}
              onClick={() => setValue(String(param.defaultValue))}
            >
              恢复默认
            </NeonButton>
            <div className="flex gap-2">
              <NeonButton icon={<X />} onClick={onClose}>
                取消
              </NeonButton>
              <NeonButton
                icon={update.isPending ? <Loader2 className="animate-spin" /> : <Save />}
                onClick={submit}
                disabled={update.isPending}
              >
                下发
              </NeonButton>
            </div>
          </div>
        </GlassPanel>
      </div>
    </div>
  )
}
