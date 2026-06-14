import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Search, RefreshCcw, RadioTower, Radio, Network, Activity } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { useDeviceList } from '@core/hooks/api/useDevices'
import type { Device } from '@core/types/device'
import { StatCard, HudLoading, HudError, HudEmpty } from './_hud'

// 设备记录中确实承载小区信息的字段（来自 Device 监控扩展字段组）。
function cellIdentity(d: Device): string {
  return d.cellId || d.eci || d.nrCellId || d.enbId || d.gnbId || ''
}

// ===========================================================================
// CONFIG · 小区管理
// 从真实设备记录（useDeviceList）派生小区视图：每台承载小区的设备一行，
// 选中后右侧展示小区无线参数（eci/pci/tac/bandwidth/txPower…）+ 状态。
// ===========================================================================
export default function CellManagement() {
  const navigate = useNavigate()
  const [keyword, setKeyword] = useState('')
  const [tech, setTech] = useState<'all' | 'lte' | 'nr'>('all')
  const [selectedId, setSelectedId] = useState('')

  const devicesQ = useDeviceList({ page: 1, pageSize: 300 })
  const devices = devicesQ.data?.items ?? []

  // 仅保留承载小区标识的设备
  const cells = useMemo(() => devices.filter((d) => cellIdentity(d) !== ''), [devices])

  const kw = keyword.trim().toLowerCase()
  const rows = cells.filter((d) => {
    if (tech === 'lte' && d.networkType?.toLowerCase() !== 'lte') return false
    if (tech === 'nr' && !(d.networkType?.toLowerCase() === 'nr' || d.nrCellId)) return false
    if (!kw) return true
    return (
      cellIdentity(d).toLowerCase().includes(kw) ||
      d.sn?.toLowerCase().includes(kw) ||
      d.name?.toLowerCase().includes(kw)
    )
  })

  const effectiveId = selectedId && rows.some((d) => d.id === selectedId) ? selectedId : rows[0]?.id || ''
  const selected = cells.find((d) => d.id === effectiveId) ?? null
  const onlineCells = cells.filter((d) => d.isOnline).length

  return (
    <PageShell
      code="F02"
      title="CELL MGMT · 小区管理"
      subtitle="RADIO CELL CONFIGURATION"
      isFetching={devicesQ.isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="CELL ID / SN / 名称"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={() => void devicesQ.refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-3 gap-3">
        <StatCard label="CELLS · 小区" value={cells.length} color="#00f0ff" />
        <StatCard label="ONLINE · 在线" value={onlineCells} color="#00ff88" />
        <StatCard label="MATCHED · 匹配" value={rows.length} color="#a855f7" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        {(['all', 'lte', 'nr'] as const).map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setTech(t)}
            className={`chip ${tech === t ? 'text-[#00f0ff] shadow-[0_0_10px_currentColor]' : 'text-[#6b86b6]'}`}
          >
            {t === 'all' ? '全部制式' : t.toUpperCase()}
          </button>
        ))}
      </div>

      <div className="grid grid-cols-[1fr_380px] gap-3">
        <GlassPanel title="CELL LIST · 小区列表" meta={`${rows.length}`} className="min-h-0">
          {devicesQ.isLoading ? (
            <HudLoading />
          ) : devicesQ.isError ? (
            <HudError error={devicesQ.error} />
          ) : rows.length === 0 ? (
            <HudEmpty icon={RadioTower} text="无小区数据" />
          ) : (
            <div className="max-h-[56vh] overflow-auto">
              <div className="grid grid-cols-[1.4fr_1fr_90px_1fr_90px] gap-3 border-b border-cyan-500/15 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/45">
                <span>CELL / 小区</span>
                <span>NET / 制式</span>
                <span>PCI</span>
                <span>STATUS / 状态</span>
                <span className="text-right">LINK</span>
              </div>
              {rows.map((d) => {
                const on = d.id === effectiveId
                return (
                  <button
                    key={d.id}
                    type="button"
                    onClick={() => setSelectedId(d.id)}
                    className={`grid w-full grid-cols-[1.4fr_1fr_90px_1fr_90px] items-center gap-3 border-b border-cyan-500/8 px-3 py-2 text-left transition-colors ${
                      on ? 'bg-cyan-500/12' : 'hover:bg-cyan-500/5'
                    }`}
                  >
                    <div className="min-w-0">
                      <div className="truncate font-mono text-xs text-cyan-100">{cellIdentity(d)}</div>
                      <div className="truncate text-[10px] text-cyan-300/50">{d.name || d.sn}</div>
                    </div>
                    <div className="font-mono text-[11px] text-cyan-300/70">
                      {(d.networkType || '—').toUpperCase()}
                    </div>
                    <div className="font-mono text-[11px] text-cyan-200/80">{d.pci || '—'}</div>
                    <div className="font-mono text-[11px] text-cyan-300/65">{d.cellStatus || d.opState || '—'}</div>
                    <div className="flex justify-end">
                      <StatusBadge status={d.isOnline ? 'online' : 'offline'} label={d.isOnline ? '在线' : '离线'} />
                    </div>
                  </button>
                )
              })}
            </div>
          )}
        </GlassPanel>

        <GlassPanel
          title={selected ? `CELL · ${cellIdentity(selected)}` : 'CELL · 小区详情'}
          meta={selected ? (selected.networkType || '').toUpperCase() : undefined}
          className="min-h-0"
        >
          {selected ? (
            <CellDetail device={selected} onOpenDevice={() => navigate(`/fleet?sn=${selected.sn}`)} />
          ) : (
            <HudEmpty icon={Radio} text="选择小区查看无线参数" />
          )}
        </GlassPanel>
      </div>
    </PageShell>
  )
}

function CellDetail({ device, onOpenDevice }: { device: Device; onOpenDevice: () => void }) {
  const tx = Number(device.txPower)
  const txPct = Number.isFinite(tx) ? Math.max(0, Math.min(100, ((tx + 60) / 60) * 100)) : 0

  const radio: Array<[string, string]> = [
    ['ENB / GNB ID', device.enbId || device.gnbId || '—'],
    ['CELL ID', device.cellId || '—'],
    ['ECI / NCI', device.eci || device.nrCellId || '—'],
    ['PCI', device.pci || '—'],
    ['TAC', device.tac || '—'],
    ['PLMN', device.plmnId || '—'],
    ['BAND', device.band || '—'],
    ['BANDWIDTH', device.bandwidth || '—'],
    ['DL EARFCN', device.dlEarfcn || '—'],
    ['UL EARFCN', device.ulEarfcn || '—'],
  ]
  const status: Array<[string, string]> = [
    ['CELL STATUS', device.cellStatus || '—'],
    ['OP STATE', device.opState || '—'],
    ['ADMIN', device.adminState || device.lockStatus || '—'],
    ['SERVICE', device.serviceStatus || '—'],
    ['RF', device.rfStatus || '—'],
    ['SYNC', device.syncStatus || '—'],
  ]

  return (
    <div className="max-h-[56vh] overflow-auto p-3">
      <div className="mb-4 flex items-center gap-4">
        <RadialGauge
          value={txPct}
          label="TX PWR"
          size={86}
          color="#00f0ff"
          unit={Number.isFinite(tx) ? 'dBm' : ''}
        />
        <div className="min-w-0 flex-1">
          <div className="truncate font-display text-base font-bold text-cyan-100">{device.name || device.sn}</div>
          <code className="block truncate font-mono text-[11px] text-cyan-300/55">{device.sn}</code>
          <div className="mt-2 flex flex-wrap gap-2">
            <StatusBadge status={device.isOnline ? 'online' : 'offline'} label={device.isOnline ? '在线' : '离线'} />
            {device.ueCount > 0 ? <span className="chip text-[#a855f7]">UE {device.ueCount}</span> : null}
            {device.vendor ? <span className="chip text-[#5b9eff]">{device.vendor}</span> : null}
          </div>
        </div>
      </div>

      <Group icon={Network} title="RADIO · 无线参数" rows={radio} />
      <Group icon={Activity} title="STATUS · 运行状态" rows={status} />

      <div className="mt-4">
        <NeonButton icon={<RadioTower />} onClick={onOpenDevice}>
          在设备清单中查看
        </NeonButton>
      </div>
    </div>
  )
}

function Group({
  icon: Icon,
  title,
  rows,
}: {
  icon: typeof Network
  title: string
  rows: Array<[string, string]>
}) {
  return (
    <div className="mb-4">
      <div className="mb-2 flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.18em] text-cyan-300/55">
        <Icon className="size-3.5 text-cyan-300/60" />
        {title}
      </div>
      <div className="grid grid-cols-2 gap-1.5">
        {rows.map(([k, v]) => (
          <div key={k} className="rounded-sm border border-cyan-500/10 px-2.5 py-1.5">
            <div className="font-mono text-[9px] uppercase tracking-[0.15em] text-cyan-300/40">{k}</div>
            <div className="truncate font-mono text-[11px] text-cyan-100/85">{v}</div>
          </div>
        ))}
      </div>
    </div>
  )
}
