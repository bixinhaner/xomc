import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, RefreshCcw, Users, Radio, Smartphone, Info } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { formatTime } from '@/lib/format'
import { useDeviceBySn } from '@core/hooks/api/useDevices'
import { useRebootRecordList } from '@core/hooks/api/useRebootRecord'
import type { Device } from '@core/types/device'
import { KV, StatCard, StateGate, formatDuration } from './_shared'

export default function FleetUeDetail() {
  const { sn = '' } = useParams<{ sn: string }>()
  const navigate = useNavigate()

  const { data: device, isLoading, isError, error, isFetching, refetch } = useDeviceBySn(sn)
  // 该 SN 的近期重启记录（真实数据），作为接入侧运行证据。
  const { data: reboots } = useRebootRecordList(
    { deviceSn: sn, page: 1, pageSize: 10 },
    Boolean(sn)
  )

  return (
    <PageShell
      code="F06"
      title={`UE · ${sn}`}
      subtitle="USER EQUIPMENT TELEMETRY · PER-CELL ACCESS"
      isFetching={isFetching}
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate(`/fleet/detail/${encodeURIComponent(sn)}`)}>
            UNIT
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <StateGate
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={!device}
        loadingLabel="LOADING UE TELEMETRY…"
        emptyLabel={`NO UNIT FOR SN ${sn}`}
      >
        {device ? <UeBody d={device} rebootRows={reboots?.items ?? []} /> : null}
      </StateGate>
    </PageShell>
  )
}

function UeBody({
  d,
  rebootRows,
}: {
  d: Device
  rebootRows: { id: string; rebootTime: string; reason: string; isAbnormal: boolean }[]
}) {
  const ue = d.ueCount ?? 0
  const cpe = d.cpeCount ?? 0
  // 接入负载占用比（按工程经验 256 UE/cell 满载估算容量，仅用于可视化刻度）
  const load = Math.min(100, (ue / 256) * 100)

  return (
    <div className="space-y-4">
      {/* 接入概况 */}
      <div className="grid grid-cols-1 gap-3 md:grid-cols-12">
        <GlassPanel strong className="md:col-span-4">
          <div className="flex items-center justify-center gap-4 p-4">
            <RadialGauge value={load} label="UE LOAD" size={120} color="#00f0ff" />
            <div className="space-y-1">
              <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                ACTIVE UE
              </div>
              <div className="font-display text-3xl font-bold text-cyan-200 text-glow">{ue}</div>
              <div className="font-mono text-[11px] text-cyan-300/55">CPE {cpe}</div>
            </div>
          </div>
        </GlassPanel>
        <div className="grid grid-cols-2 gap-3 md:col-span-8 md:grid-cols-4">
          <StatCard label="UE COUNT" value={ue} color="#00f0ff" />
          <StatCard label="CPE COUNT" value={cpe} color="#a855f7" />
          <StatCard label="CELL STATUS" value={<span className="text-base">{d.cellStatus || '—'}</span>} color="#00ff88" />
          <StatCard label="ONLINE" value={<span className="text-base">{d.isOnline ? 'YES' : 'NO'}</span>} color={d.isOnline ? '#00ff88' : '#525a78'} />
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {/* 接入小区无线参数 */}
        <GlassPanel title="ACCESS CELL · 接入小区" meta={<Radio className="size-3.5" />}>
          <KV label="CELL ID / ECI">{`${d.cellId || '—'} · ${d.eci || '—'}`}</KV>
          <KV label="PCI / TAC">{`${d.pci || '—'} · ${d.tac || '—'}`}</KV>
          <KV label="PLMN">{d.plmnId}</KV>
          <KV label="BANDWIDTH / BAND">{`${d.bandwidth || '—'} · ${d.band || '—'}`}</KV>
          <KV label="DL / UL EARFCN">{`${d.dlEarfcn || '—'} · ${d.ulEarfcn || '—'}`}</KV>
          <KV label="TX POWER">{d.txPower}</KV>
          <KV label="ADMIN / OP STATE">{`${d.adminState || '—'} · ${d.opState || '—'}`}</KV>
        </GlassPanel>

        {/* 该 SN 的运行证据（重启记录，真实） */}
        <GlassPanel title="REBOOT TRAIL · 重启记录" meta={<Smartphone className="size-3.5" />}>
          {rebootRows.length === 0 ? (
            <div className="px-3 py-6 text-center font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/40">
              NO REBOOT RECORDS
            </div>
          ) : (
            <div className="divide-y divide-cyan-500/8">
              {rebootRows.map((r) => (
                <div key={r.id} className="flex items-center gap-3 px-3 py-2">
                  <StatusBadge
                    status={r.isAbnormal ? 'critical' : 'ok'}
                    label={r.isAbnormal ? 'ABNORMAL' : 'NORMAL'}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="truncate font-mono text-[11px] text-cyan-100/85">
                      {r.reason || '—'}
                    </div>
                  </div>
                  <span className="shrink-0 font-mono text-[10px] text-cyan-300/55">
                    {formatTime(r.rebootTime)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </GlassPanel>
      </div>

      {/* 在线时长 */}
      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        <StatCard label="UPTIME" value={formatDuration(d.upTime)} color="#00ff88" />
        <StatCard label="ONLINE TOTAL" value={formatDuration(d.cumulativeOnlineDuration)} color="#a855f7" />
        <StatCard label="LAST INFORM" value={<span className="text-sm">{formatTime(d.lastInformTime)}</span>} color="#00f0ff" />
        <StatCard label="GROUP" value={<span className="text-sm">{d.groupName || '—'}</span>} color="#ffaa00" />
      </div>

      <div className="flex items-start gap-2 rounded-sm border border-cyan-500/15 bg-cyan-500/[0.03] px-3 py-2 font-mono text-[10px] text-cyan-300/55">
        <Info className="mt-0.5 size-3.5 shrink-0 text-cyan-400/60" />
        <span className="flex items-center gap-1">
          <Users className="size-3" />
          UE 明细由设备实时上报的接入计数与小区参数派生；逐 UE 列表无独立后端端点。
        </span>
      </div>
    </div>
  )
}
