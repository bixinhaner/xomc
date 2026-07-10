import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { Search, RefreshCw, Power, Loader2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { Sparkline } from '@/components/viz/Sparkline'
import { formatTime } from '@/lib/format'
import { prefetchDeviceDetailContext, useDeviceGroups, useDeviceList, useUpdateDevice } from '@core/hooks/api/useDevices'
import { formatAlarmSeverityBadgeLabel } from '@core/utils/alarmSeverity'
import { useAlarmCountWithDeviceListInvalidation } from '@core/hooks/api/useAlarms'
import type { Device } from '@core/types/device'
import { expandSelectedGroupIds } from '@core/utils/deviceGroupFilter'

const STATUS_COLOR: Record<string, string> = {
  online: '#00ff88',
  offline: '#525a78',
}

const AUTO_REFRESH_OPTIONS = [
  { label: 'AUTO REFRESH', value: 'off' },
  { label: '15S', value: '15' },
  { label: '30S', value: '30' },
  { label: '1MIN', value: '60' },
  { label: '5MIN', value: '300' },
] as const
export function FleetPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [keyword, setKeyword] = useState('')
  const [autoRefresh, setAutoRefresh] = useState(false)
  const [refreshInterval, setRefreshInterval] = useState(30)
  const [editingInstallAddressId, setEditingInstallAddressId] = useState<string | null>(null)
  const [editingInstallAddressValue, setEditingInstallAddressValue] = useState('')
  const [savingInstallAddressId, setSavingInstallAddressId] = useState<string | null>(null)
  const { data: groupsResp } = useDeviceGroups()
  const updateDevice = useUpdateDevice()
  const alarmCountQuery = useAlarmCountWithDeviceListInvalidation()

  const params = useMemo(() => {
    const rawGroupID = searchParams.get('groupId') ?? undefined
    const expandedGroupIDs = expandSelectedGroupIds(rawGroupID, groupsResp?.groups ?? [])

    return {
      page,
      pageSize,
      ...(keyword.trim() ? { searchText: keyword.trim() } : {}),
      ...(expandedGroupIDs ? { groupId: expandedGroupIDs } : {}),
    }
  }, [page, keyword, searchParams, groupsResp?.groups])

  const { data, isFetching, isLoading, isError, error, refetch } = useDeviceList(params, {
    refetchInterval: autoRefresh ? refreshInterval * 1000 : 0,
  })
  const items: Device[] = useMemo(
    () => data?.items ?? [],
    [data?.items]
  )
  const total = data?.total ?? 0
  const activeAlarmCount = alarmCountQuery.data?.total_active ?? data?.stats?.alarmed ?? 0

  useEffect(() => {
    if (!autoRefresh) return
    void refetch()
  }, [autoRefresh, refreshInterval, refetch])

  const prefetchDeviceDetailEntry = (device: Device) => {
    void import('@/pages/fleet/DeviceDetail')
    void prefetchDeviceDetailContext(queryClient, device)
  }

  const openDeviceDetail = (device: Device) => {
    prefetchDeviceDetailEntry(device)
    void navigate(`/device/detail/${device.sn}`)
  }

  function startInstallAddressEdit(device: Device) {
    setEditingInstallAddressId(device.id)
    setEditingInstallAddressValue(device.installAddress || '')
  }

  function cancelInstallAddressEdit() {
    setEditingInstallAddressId(null)
    setEditingInstallAddressValue('')
    setSavingInstallAddressId(null)
  }

  function saveInstallAddressEdit(device: Device) {
    const nextValue = editingInstallAddressValue.trim()
    const currentValue = (device.installAddress || '').trim()
    if (savingInstallAddressId === device.id) return
    if (nextValue === currentValue) {
      cancelInstallAddressEdit()
      return
    }

    setSavingInstallAddressId(device.id)
    updateDevice.mutate(
      {
        id: device.id,
        data: { installAddress: nextValue },
        fallbackDevice: {
          id: device.id,
          sn: device.sn,
          installAddress: device.installAddress,
          remark: device.remark,
        },
      },
      {
        onSuccess: (updatedDevice) => {
          queryClient.setQueryData(['devices', 'list', params], (prev: typeof data) => {
            if (!prev) return prev
            return {
              ...prev,
              items: prev.items.map((item) =>
                item.id === updatedDevice.id ? { ...item, installAddress: updatedDevice.installAddress } : item
              ),
            }
          })
          setEditingInstallAddressId(null)
          setEditingInstallAddressValue('')
          setSavingInstallAddressId(null)
        },
        onError: () => {
          setSavingInstallAddressId(null)
        },
      }
    )
  }

  const autoRefreshValue = autoRefresh ? String(refreshInterval) : 'off'

  return (
    <PageShell
      code="F02"
      title="FLEET · 舰队管理"
      subtitle="DEVICE INVENTORY · LIVE TR069 SESSIONS"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-72 pl-9"
              placeholder="SN / 名称 / IP / MAC"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <NeonButton icon={<RefreshCw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
          <label className="flex items-center gap-2 rounded-sm border border-cyan-400/30 bg-slate-950/70 px-3 py-2 text-[10px] uppercase tracking-[0.22em] text-cyan-200/80">
            <RefreshCw className={autoRefresh ? 'size-3.5 animate-spin text-cyan-200' : 'size-3.5 text-cyan-300/65'} />
            <select
              className="bg-transparent text-[10px] uppercase tracking-[0.22em] text-cyan-100 outline-none"
              value={autoRefreshValue}
              onChange={(e) => {
                const value = e.target.value
                if (value === 'off') {
                  setAutoRefresh(false)
                  return
                }
                setRefreshInterval(Number(value))
                setAutoRefresh(true)
              }}
            >
              {AUTO_REFRESH_OPTIONS.map((option) => (
                <option key={option.value} value={option.value} className="bg-slate-950 text-cyan-100">
                  {option.label}
                </option>
              ))}
            </select>
          </label>
        </>
      }
    >
      {/* 状态摘要 */}
      <div className="mb-3 flex flex-wrap items-center gap-3 text-[11px] uppercase tracking-[0.18em]">
        <span className="text-cyan-300/60">
          TOTAL <span className="ml-1 font-display text-base text-cyan-200 text-glow">{total}</span>
        </span>
        <span className="text-emerald-300/80">
          ONLINE
          <span className="ml-1 font-display text-base text-emerald-200 text-glow">
            {data?.stats?.online ?? '—'}
          </span>
        </span>
        <span className="text-zinc-300/70">
          OFFLINE
          <span className="ml-1 font-display text-base text-zinc-100">
            {data?.stats?.offline ?? '—'}
          </span>
        </span>
        <span className="text-rose-300/80">
          ALARMS
          <span className="ml-1 font-display text-base text-rose-200 text-glow">
            {activeAlarmCount}
          </span>
        </span>
      </div>

      {/* 数据矩阵 */}
      <div className="space-y-2">
        {isLoading ? (
          <Skeleton />
        ) : isError ? (
          <ErrorBlock msg={error instanceof Error ? error.message : '未知错误'} />
        ) : items.length === 0 ? (
          <EmptyBlock />
        ) : (
          items.map((d) => (
            <div
              key={d.id}
              className="fleet-row group grid grid-cols-[12px_1.6fr_1fr_1fr_1fr_120px_120px] items-center gap-3 rounded-sm px-3 py-2.5"
              style={{
                ['--row-color' as never]:
                  d.connStatus === 'online'
                    ? '#00ff88'
                    : d.alarmLevel === 'critical'
                      ? '#ff2d6f'
                      : d.alarmLevel === 'major'
                        ? '#ff7a1a'
                        : '#525a78',
              }}
            >
              {/* 状态圆灯 */}
              <span
                className="size-2.5 rounded-full"
                style={{
                  background: STATUS_COLOR[d.connStatus] ?? '#525a78',
                  boxShadow: `0 0 8px ${STATUS_COLOR[d.connStatus] ?? '#525a78'}`,
                }}
              />

              {/* 名称 + SN */}
              <div className="min-w-0">
                <div className="truncate font-display text-sm font-bold text-cyan-100">
                  {d.name || d.sn}
                </div>
                <div className="truncate font-mono text-[10px] text-cyan-300/55">
                  SN {d.sn} · {d.vendor} · {d.networkType || d.deviceModel}
                </div>
              </div>

              {/* 状态徽 */}
              <div className="flex flex-col gap-1">
                <StatusBadge status={d.connStatus} />
                {d.alarmLevel !== 'none' ? (
                  // #361: 把活动告警数拼进 HUD 徽标（如「ALM · major · 3」）。
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
                  <span className="font-mono text-[10px] text-cyan-300/40">NO ALARM</span>
                )}
              </div>

              {/* 位置 */}
              <div className="min-w-0">
                <div className="truncate text-xs text-cyan-100/85">{d.region || '—'}</div>
                <div className="truncate font-mono text-[10px] text-cyan-300/55">
                  {d.subnet || '—'} · {d.ipAddress || '—'}
                </div>
                {editingInstallAddressId === d.id ? (
                  <input
                    autoFocus
                    maxLength={256}
                    value={editingInstallAddressValue}
                    placeholder="双击编辑安装详细地址"
                    onClick={(e) => e.stopPropagation()}
                    onChange={(e) => setEditingInstallAddressValue(e.target.value)}
                    onBlur={() => saveInstallAddressEdit(d)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') saveInstallAddressEdit(d)
                      if (e.key === 'Escape') cancelInstallAddressEdit()
                    }}
                    className="mt-1 w-full rounded-sm border border-cyan-400/30 bg-slate-950/70 px-2 py-1 text-[10px] text-cyan-100 outline-none"
                  />
                ) : (
                  <div
                    title={d.installAddress || '双击编辑安装详细地址'}
                    onDoubleClick={(e) => {
                      e.stopPropagation()
                      startInstallAddressEdit(d)
                    }}
                    className={`mt-1 truncate text-[10px] ${d.installAddress ? 'text-cyan-200/75' : 'text-cyan-300/45'}`}
                  >
                    {savingInstallAddressId === d.id ? 'SAVING…' : d.installAddress || '双击编辑安装详细地址'}
                  </div>
                )}
              </div>

              {/* 最后在线 */}
              <div className="font-mono text-[10px] text-cyan-300/65">
                <div className="text-cyan-100/85">{formatTime(d.lastOnlineTime)}</div>
                <div>{d.softwareVersion ? `FW ${d.softwareVersion}` : '—'}</div>
                <div>{`TX ${d.txPower || '—'}`}</div>
              </div>

              {/* 信号 sparkline */}
              <div className="hidden md:block">
                <Sparkline
                  data={fakeSpark(d.id)}
                  color={d.connStatus === 'online' ? '#00ff88' : '#525a78'}
                  width={110}
                  height={28}
                  fill
                />
              </div>

              {/* 动作 */}
              <div className="flex items-center justify-end gap-2 opacity-60 transition-opacity group-hover:opacity-100">
                <NeonButton
                  tone="cyan"
                  className="!py-1 !px-2"
                  onMouseEnter={() => prefetchDeviceDetailEntry(d)}
                  onFocus={() => prefetchDeviceDetailEntry(d)}
                  onClick={() => openDeviceDetail(d)}
                >
                  DETAIL
                </NeonButton>
                <NeonButton tone="danger" icon={<Power />} className="!py-1 !px-2">
                  RBT
                </NeonButton>
              </div>
            </div>
          ))
        )}
      </div>

      {/* 分页 */}
      <div className="mt-4 flex items-center justify-between">
        <span className="font-mono text-[11px] text-cyan-300/55">
          PAGE {page} / {Math.max(1, Math.ceil(total / pageSize))} · {pageSize}/PAGE
        </span>
        <div className="flex gap-2">
          <NeonButton onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
            ◂ PREV
          </NeonButton>
          <NeonButton
            onClick={() => setPage((p) => p + 1)}
            disabled={page >= Math.ceil(total / pageSize)}
          >
            NEXT ▸
          </NeonButton>
        </div>
      </div>
    </PageShell>
  )
}

function Skeleton() {
  return (
    <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
      <Loader2 className="size-4 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING FLEET…</span>
    </div>
  )
}

function ErrorBlock({ msg }: { msg: string }) {
  return (
    <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
      <div className="font-bold uppercase tracking-[0.2em]">FAILURE</div>
      <div className="mt-1 text-rose-200/80">{msg}</div>
    </div>
  )
}

function EmptyBlock() {
  return (
    <div className="border border-cyan-500/15 px-4 py-12 text-center font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/40">
      NO DEVICES IN ORBIT
    </div>
  )
}

/** 基于 id 生成稳定的伪随机 sparkline */
function fakeSpark(seed: string): number[] {
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) >>> 0
  const out: number[] = []
  for (let i = 0; i < 24; i++) {
    h = (h * 1664525 + 1013904223) >>> 0
    out.push((h % 100) / 100)
  }
  return out
}
