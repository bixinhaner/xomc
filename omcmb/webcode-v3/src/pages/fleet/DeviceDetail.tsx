import { useNavigate, useParams } from 'react-router-dom'
import {
  ArrowLeft,
  RefreshCcw,
  Power,
  Radio,
  Users,
  Activity,
  Cpu,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { formatTime } from '@/lib/format'
import { useDeviceBySn, useRebootDevice } from '@core/hooks/api/useDevices'
import { useAppStore } from '@core/store/appStore'
import { useDictionary } from '@core/hooks/api/useSystem'
import { activationStatusLabelOf } from '@core/utils/activationStatus'
import { formatAlarmSeverityBadgeLabel } from '@core/utils/alarmSeverity'
import { DEVICE_SYNC_STATUS_LABELS_EN, formatDeviceSyncStatus } from '@core/utils/deviceSyncStatus'
import type { Device } from '@core/types/device'
import { KV, StatCard, StateGate, formatDuration } from './_shared'

function openClassicQuickSettings(sn: string) {
  window.location.assign(`/device/detail/${sn}?tab=quickSettings`)
}

export default function FleetDeviceDetail() {
  const { sn = '' } = useParams<{ sn: string }>()
  const navigate = useNavigate()
  const { data: device, isLoading, isError, error, isFetching, refetch } = useDeviceBySn(sn)
  const reboot = useRebootDevice()

  return (
    <PageShell
      code="F06"
      title={`UNIT · ${sn}`}
      subtitle="DEVICE DOSSIER · LIVE TELEMETRY"
      isFetching={isFetching}
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/device/list')}>
            FLEET
          </NeonButton>
          {sn ? (
            <NeonButton icon={<Radio />} onClick={() => openClassicQuickSettings(sn)}>
              QUICK SETTINGS
            </NeonButton>
          ) : null}
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
          {device ? (
            <NeonButton
              tone="danger"
              icon={<Power />}
              disabled={reboot.isPending}
              onClick={() => reboot.mutate(device.id)}
            >
              {reboot.isPending ? 'SENDING…' : 'REBOOT'}
            </NeonButton>
          ) : null}
        </>
      }
    >
      <StateGate
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={!device}
        loadingLabel="LOADING UNIT…"
        emptyLabel={`NO UNIT FOR SN ${sn}`}
      >
        {device ? <DetailBody d={device} onUe={() => navigate(`/device/ue-detail/${device.sn}`)} /> : null}
      </StateGate>
    </PageShell>
  )
}

function DetailBody({ d, onUe }: { d: Device; onUe: () => void }) {
  const appLocale = useAppStore((s) => s.locale)
  const { data: opStateDict } = useDictionary('op_state')
  const gps = d.gpsSatelliteCount > 0 ? Math.min(100, (d.gpsSatelliteCount / 12) * 100) : 0
  const opStateLabel = activationStatusLabelOf(d.opState, opStateDict?.sysDictionaryDetails, {
    active: '激活',
    inactive: '未激活',
  }, appLocale)
  return (
    <div className="space-y-4">
      {/* 头部状态条 */}
      <GlassPanel strong>
        <div className="flex flex-wrap items-center gap-4 p-4">
          <div className="flex items-center gap-3">
            <span
              className="size-3 rounded-full"
              style={{
                background: d.isOnline ? '#00ff88' : '#525a78',
                boxShadow: `0 0 10px ${d.isOnline ? '#00ff88' : '#525a78'}`,
              }}
            />
            <div>
              <div className="font-display text-xl font-bold text-cyan-100">
                {d.name || d.deviceName || d.sn}
              </div>
              <div className="font-mono text-[11px] text-cyan-300/55">
                SN {d.sn} · {d.vendor || '—'} · {d.deviceModel || d.productClass || '—'}
              </div>
            </div>
          </div>
          <div className="ml-auto flex flex-wrap items-center gap-2">
            <StatusBadge status={d.isOnline ? 'online' : 'offline'} />
            <span className="chip text-cyan-300/80">{d.lifecycleState}</span>
            {d.alarmLevel !== 'none' ? (
              <StatusBadge
                status={d.alarmLevel}
                label={`ALM · ${formatAlarmSeverityBadgeLabel(d.alarmLevel, d.activeAlarmCount, {
                  critical: 'critical',
                  major: 'major',
                  minor: 'minor',
                  warning: 'warning',
                  none: 'none',
                })}`}
              />
            ) : (
              <span className="chip text-emerald-300/70">NO ALARM</span>
            )}
          </div>
        </div>
      </GlassPanel>

      {/* KPI 摘要 */}
      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        <StatCard
          label="UE COUNT"
          value={d.ueCount ?? 0}
          color="#00f0ff"
          hint={<span>CPE {d.cpeCount ?? 0}</span>}
        />
        <StatCard
          label="UPTIME"
          value={formatDuration(d.upTime)}
          color="#00ff88"
          hint="DEVICE RUNTIME"
        />
        <StatCard
          label="ONLINE TOTAL"
          value={formatDuration(d.cumulativeOnlineDuration)}
          color="#a855f7"
          hint="OMC CUMULATIVE"
        />
        <StatCard
          label="FW"
          value={<span className="text-base">{d.softwareVersion || d.firmwareVersion || '—'}</span>}
          color="#ffaa00"
          hint={d.rom ? `ROM ${d.rom}` : 'FIRMWARE'}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {/* 设备信息 */}
        <GlassPanel title="DEVICE · 设备信息" meta={<Cpu className="size-3.5" />}>
          <KV label="HOST NAME">{d.hostName}</KV>
          <KV label="PRODUCT">{d.productName || d.productClass}</KV>
          <KV label="OUI / CARRIER">{`${d.oui || '—'} · ${d.carrier || '—'}`}</KV>
          <KV label="MAC">{d.macAddress}</KV>
          <KV label="IP ADDRESS">{d.ipAddress}</KV>
          <KV label="GROUP">{d.groupName}</KV>
          <KV label="REGION / SITE">{`${d.region || '—'} · ${d.site || '—'}`}</KV>
          <KV label="INSTALL ADDRESS">{d.installAddress || '—'}</KV>
        </GlassPanel>

        {/* 小区信息 */}
        <GlassPanel title="CELL · 小区信息" meta={<Radio className="size-3.5" />}>
          <KV label="ENB / GNB ID">{`${d.enbId || '—'} · ${d.gnbId || '—'}`}</KV>
          <KV label="CELL ID / ECI">{`${d.cellId || '—'} · ${d.eci || '—'}`}</KV>
          <KV label="PCI / TAC">{`${d.pci || '—'} · ${d.tac || '—'}`}</KV>
          <KV label="PLMN">{d.plmnId}</KV>
          <KV label="BANDWIDTH / BAND">{`${d.bandwidth || '—'} · ${d.band || '—'}`}</KV>
          <KV label="DL / UL EARFCN">{`${d.dlEarfcn || '—'} · ${d.ulEarfcn || '—'}`}</KV>
          <KV label="TX POWER">{d.txPower}</KV>
        </GlassPanel>

        {/* 状态信息 + GPS */}
        <GlassPanel title="STATUS · 运行状态" meta={<Activity className="size-3.5" />}>
          <div className="flex items-center justify-center gap-4 px-3 py-3">
            <RadialGauge
              value={gps}
              label="GPS"
              unit=""
              size={96}
              color={gps > 50 ? '#00ff88' : '#ffaa00'}
            />
            <div className="space-y-1 font-mono text-[11px]">
              <div className="text-cyan-300/60">SATELLITES</div>
              <div className="font-display text-lg text-cyan-100">{d.gpsSatelliteCount}</div>
            </div>
          </div>
          <KV label="CELL STATUS">{d.cellStatus}</KV>
          <KV label="OP / ADMIN STATE">{`${opStateLabel || '—'} · ${d.adminState || '—'}`}</KV>
          <KV label="RF / SYNC">{`${d.rfStatus || '—'} · ${formatDeviceSyncStatus(d.syncStatus, DEVICE_SYNC_STATUS_LABELS_EN) || '—'}`}</KV>
          <KV label="SERVICE STATUS">{d.serviceStatus}</KV>
          <KV label="LAST ONLINE">{formatTime(d.lastOnlineTime)}</KV>
          <KV label="LAST INFORM">{formatTime(d.lastInformTime)}</KV>
        </GlassPanel>
      </div>

      {/* 联动入口 */}
      <div className="flex flex-wrap gap-2">
        <NeonButton icon={<Users />} onClick={onUe}>
          UE DETAIL · 用户终端
        </NeonButton>
      </div>
    </div>
  )
}
