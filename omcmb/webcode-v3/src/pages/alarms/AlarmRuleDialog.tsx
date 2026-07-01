import { useEffect, useMemo, useState } from 'react'
import { ChevronLeft, ChevronRight, Search, X } from 'lucide-react'

import { NeonButton } from '@/components/ui/NeonButton'
import { cn } from '@/lib/utils'

import { useAllAlarmDefinitions } from '@core/hooks/api/useAlarmDefinitions'
import { useDeviceGroups, useDeviceList, useDevicesByIds } from '@core/hooks/api/useDevices'
import type { AlarmRule } from '@core/types/alarm'
import type { Device, DeviceGroup } from '@core/types/device'
import {
  getAlarmRuleSelection,
  type AlarmRuleSelectionMode,
} from '@core/utils/alarmRuleConditions'

const DEVICE_PAGE_SIZE = 8
const ALARM_PAGE_SIZE = 8

const RULE_TYPE_OPTIONS = [
  { value: 'ignore', label: '禁止上报' },
  { value: 'auto_acknowledge', label: '自动确认' },
  { value: 'auto_clear', label: '自动清除' },
] as const

type AlarmDefinitionItem = {
  identifier: string
  cnName?: string
  cnProbableCause?: string
  enName?: string
  enProbableCause?: string
}

export interface AlarmRuleFormValue {
  ruleName: string
  ruleType: string
  enabled: boolean
  deviceSelectionMode: AlarmRuleSelectionMode
  selectedDevices: string[]
  selectedGroups: string[]
  selectedAlarms: string[]
}

function inferDeviceType(device: Device): string {
  const networkType = device.networkType.toLowerCase()
  if (networkType.includes('nr') || networkType.includes('5g') || networkType.includes('gnb')) {
    return 'gNB'
  }
  if (networkType.includes('gsm')) {
    return 'GSM'
  }
  return 'eNB'
}

function normalizeAlarmName(item: AlarmDefinitionItem) {
  return item.cnProbableCause || item.cnName || item.enProbableCause || item.enName || item.identifier
}

export function AlarmRuleDialog({
  open,
  mode,
  rule,
  loading,
  onSubmit,
  onCancel,
}: {
  open: boolean
  mode: 'add' | 'edit' | 'view'
  rule: AlarmRule | null
  loading?: boolean
  onSubmit: (value: AlarmRuleFormValue) => void | Promise<void>
  onCancel: () => void
}) {
  const [ruleName, setRuleName] = useState('')
  const [ruleType, setRuleType] = useState('ignore')
  const [enabled, setEnabled] = useState(false)
  const [deviceSelectionMode, setDeviceSelectionMode] = useState<AlarmRuleSelectionMode>('devices')
  const [selectedDevices, setSelectedDevices] = useState<string[]>([])
  const [selectedGroups, setSelectedGroups] = useState<string[]>([])
  const [selectedAlarms, setSelectedAlarms] = useState<string[]>([])
  const [deviceKeyword, setDeviceKeyword] = useState('')
  const [alarmKeyword, setAlarmKeyword] = useState('')
  const [devicePage, setDevicePage] = useState(1)
  const [alarmPage, setAlarmPage] = useState(1)

  const isViewMode = mode === 'view'

  const deviceQuery = useDeviceList(
    {
      page: devicePage,
      pageSize: DEVICE_PAGE_SIZE,
      ...(deviceKeyword.trim() ? { searchText: deviceKeyword.trim() } : {}),
    },
    { enabled: open },
  )
  const groupsQuery = useDeviceGroups()
  const alarmDefsQuery = useAllAlarmDefinitions()
  const selectedDeviceQueries = useDevicesByIds(selectedDevices)

  useEffect(() => {
    if (!open) return
    const selection = getAlarmRuleSelection(rule)
    setRuleName(rule?.ruleName ?? '')
    setRuleType(rule?.ruleType && rule.ruleType !== 'default' ? rule.ruleType : 'ignore')
    setEnabled(rule?.enabled ?? false)
    setDeviceSelectionMode(selection.deviceSelectionMode)
    setSelectedDevices(selection.selectedDevices)
    setSelectedGroups(selection.selectedGroups)
    setSelectedAlarms(selection.selectedAlarms)
    setDeviceKeyword('')
    setAlarmKeyword('')
    setDevicePage(1)
    setAlarmPage(1)
  }, [open, rule])

  useEffect(() => {
    setDevicePage(1)
  }, [deviceKeyword])

  useEffect(() => {
    setAlarmPage(1)
  }, [alarmKeyword])

  const deviceItems = deviceQuery.data?.items ?? []
  const deviceTotal = deviceQuery.data?.total ?? 0
  const devicePageCount = Math.max(1, Math.ceil(deviceTotal / DEVICE_PAGE_SIZE))
  const groups = groupsQuery.data?.groups ?? []

  const selectedDeviceRows = useMemo(() => {
    const merged = new Map<string, Device>()
    deviceItems.forEach((device) => merged.set(device.id, device))
    selectedDeviceQueries.forEach((query) => {
      if (query.data) {
        merged.set(query.data.id, query.data)
      }
    })

    return selectedDevices.map((id) => merged.get(id)).filter((device): device is Device => Boolean(device))
  }, [deviceItems, selectedDeviceQueries, selectedDevices])

  const selectedAlarmSet = useMemo(() => new Set(selectedAlarms), [selectedAlarms])
  const selectedDeviceSet = useMemo(() => new Set(selectedDevices), [selectedDevices])
  const selectedGroupSet = useMemo(() => new Set(selectedGroups), [selectedGroups])

  const alarmItems = useMemo(() => {
    const items = alarmDefsQuery.data?.items ?? []
    const keyword = alarmKeyword.trim().toLowerCase()
    const filtered = keyword
      ? items.filter((item) => {
          const name = normalizeAlarmName(item).toLowerCase()
          return item.identifier.toLowerCase().includes(keyword) || name.includes(keyword)
        })
      : items

    return [...filtered].sort((left, right) => {
      const leftSelected = selectedAlarmSet.has(left.identifier)
      const rightSelected = selectedAlarmSet.has(right.identifier)
      return Number(rightSelected) - Number(leftSelected)
    })
  }, [alarmDefsQuery.data?.items, alarmKeyword, selectedAlarmSet])

  const alarmTotal = alarmItems.length
  const alarmPageCount = Math.max(1, Math.ceil(alarmTotal / ALARM_PAGE_SIZE))
  const pagedAlarms = useMemo(
    () => alarmItems.slice((alarmPage - 1) * ALARM_PAGE_SIZE, alarmPage * ALARM_PAGE_SIZE),
    [alarmItems, alarmPage],
  )

  const selectedAlarmItems = useMemo<AlarmDefinitionItem[]>(() => {
    const itemMap = new Map<string, AlarmDefinitionItem>((alarmDefsQuery.data?.items ?? []).map((item) => [item.identifier, item]))
    return selectedAlarms
      .map((identifier) => itemMap.get(identifier))
      .filter((item): item is AlarmDefinitionItem => Boolean(item))
  }, [alarmDefsQuery.data?.items, selectedAlarms])

  const visibleDeviceSelectedCount = deviceItems.filter((item) => selectedDeviceSet.has(item.id)).length
  const visibleAlarmSelectedCount = pagedAlarms.filter((item) => selectedAlarmSet.has(item.identifier)).length

  const allVisibleDevicesSelected = deviceItems.length > 0 && visibleDeviceSelectedCount === deviceItems.length
  const allVisibleAlarmsSelected = pagedAlarms.length > 0 && visibleAlarmSelectedCount === pagedAlarms.length
  const allGroupsSelected = groups.length > 0 && selectedGroups.length === groups.length
  const canSubmit = ruleName.trim().length > 0 && selectedAlarms.length > 0 && !loading

  if (!open) return null

  const submit = () => {
    if (isViewMode || !canSubmit) return
    void onSubmit({
      ruleName: ruleName.trim(),
      ruleType,
      enabled,
      deviceSelectionMode,
      selectedDevices,
      selectedGroups,
      selectedAlarms,
    })
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4 py-6 backdrop-blur-sm">
      <div className="glass flex max-h-[92vh] w-full max-w-6xl flex-col overflow-hidden rounded-sm border border-cyan-500/25">
        <div className="flex items-start justify-between gap-4 border-b border-cyan-500/15 px-5 py-4">
          <div>
            <div className="font-mono text-[11px] uppercase tracking-[0.22em] text-cyan-300/55">
              Alarm Rule Editor
            </div>
            <h2 className="font-display text-2xl font-bold text-cyan-100">
              {mode === 'add' ? '新建告警规则' : mode === 'edit' ? '编辑告警规则' : '查看告警规则'}
            </h2>
            <p className="mt-1 text-sm text-cyan-300/60">
              设备列表、告警列表和 v1 抽屉采用同一筛选与选择口径
            </p>
          </div>
          <button
            type="button"
            onClick={onCancel}
            className="flex size-8 items-center justify-center rounded-sm border border-cyan-500/20 text-cyan-300/70 transition hover:border-cyan-400/60 hover:text-cyan-100"
          >
            <X className="size-4" />
          </button>
        </div>

        <div className="space-y-4 overflow-auto px-5 py-4">
          <div className="grid gap-4 md:grid-cols-3">
            <label className="block">
              <span className="mb-1.5 block font-mono text-[11px] uppercase tracking-[0.16em] text-cyan-300/60">
                规则名称
              </span>
              <input
                className="neon-input w-full"
                value={ruleName}
                maxLength={50}
                placeholder="输入规则名称"
                onChange={(e) => setRuleName(e.target.value)}
                disabled={loading || isViewMode}
              />
            </label>

            <label className="block">
              <span className="mb-1.5 block font-mono text-[11px] uppercase tracking-[0.16em] text-cyan-300/60">
                执行动作
              </span>
              <select
                className="neon-input w-full"
                value={ruleType}
                onChange={(e) => setRuleType(e.target.value)}
                disabled={loading || isViewMode}
              >
                {RULE_TYPE_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>

            <div className="glass rounded-sm border border-cyan-500/15 px-3 py-2.5">
              <div className="font-mono text-[11px] uppercase tracking-[0.16em] text-cyan-300/60">
                立即启用
              </div>
              <button
                type="button"
                onClick={() => !isViewMode && setEnabled((value) => !value)}
                disabled={isViewMode}
                className={cn(
                  'mt-3 flex w-full items-center justify-between rounded-sm border px-3 py-2 text-sm transition',
                  enabled ? 'border-emerald-400/50 text-emerald-300' : 'border-cyan-500/15 text-cyan-300/70',
                  isViewMode && 'cursor-not-allowed opacity-70',
                )}
              >
                <span>{enabled ? '启用中' : '未启用'}</span>
                <span className={cn('chip', enabled ? 'text-emerald-300' : 'text-cyan-300/60')}>
                  {enabled ? 'ON' : 'OFF'}
                </span>
              </button>
            </div>
          </div>

          <section className="glass rounded-sm border border-cyan-500/15 px-4 py-4">
            <div className="flex items-center justify-between gap-4">
              <div>
                <div className="font-mono text-[11px] uppercase tracking-[0.16em] text-cyan-300/60">
                  Device Scope
                </div>
                <div className="text-sm text-cyan-100">设备范围</div>
              </div>
              <div className="flex gap-2">
                {(['devices', 'groups'] as const).map((value) => (
                  <button
                    key={value}
                    type="button"
                    disabled={isViewMode}
                    onClick={() => setDeviceSelectionMode(value)}
                    className={cn(
                      'chip transition-all',
                      deviceSelectionMode === value ? 'text-cyan-100 shadow-[0_0_10px_currentColor]' : 'text-cyan-300/45',
                    )}
                  >
                    {value === 'devices' ? '设备列表' : '设备组'}
                  </button>
                ))}
              </div>
            </div>

            {deviceSelectionMode === 'devices' ? (
              <>
                <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
                  <div className="relative w-full max-w-xs">
                    <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-cyan-300/45" />
                    <input
                      className="neon-input w-full pl-9"
                      value={deviceKeyword}
                      onChange={(e) => setDeviceKeyword(e.target.value)}
                      placeholder="搜索设备 SN/名称"
                    />
                  </div>
                  <div className="flex flex-wrap items-center gap-4 font-mono text-[11px] text-cyan-300/60">
                    <span>已选 {selectedDevices.length} 台</span>
                    <span>共 {deviceTotal} 台</span>
                    <label className="flex items-center gap-2 text-cyan-100">
                      <input
                        type="checkbox"
                        className="size-4 accent-cyan-400"
                        checked={allVisibleDevicesSelected}
                        onChange={(e) => {
                          const visibleIds = deviceItems.map((item) => item.id)
                          if (e.target.checked) {
                            setSelectedDevices((prev) => Array.from(new Set([...prev, ...visibleIds])))
                          } else {
                            setSelectedDevices((prev) => prev.filter((id) => !visibleIds.includes(id)))
                          }
                        }}
                        disabled={isViewMode || deviceItems.length === 0}
                      />
                      当前页全选
                    </label>
                  </div>
                </div>

                <div className="mt-3 overflow-hidden rounded-sm border border-cyan-500/15">
                  <table className="w-full text-sm">
                    <thead className="border-b border-cyan-500/10 bg-cyan-500/5">
                      <tr className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/50">
                        <th className="px-3 py-2 text-left">选</th>
                        <th className="px-3 py-2 text-left">SN</th>
                        <th className="px-3 py-2 text-left">名称</th>
                        <th className="px-3 py-2 text-left">制式</th>
                        <th className="px-3 py-2 text-left">状态</th>
                      </tr>
                    </thead>
                    <tbody>
                      {deviceQuery.isLoading ? (
                        <tr>
                          <td colSpan={5} className="px-3 py-6 text-center text-cyan-300/55">正在加载设备…</td>
                        </tr>
                      ) : deviceItems.length === 0 ? (
                        <tr>
                          <td colSpan={5} className="px-3 py-6 text-center text-cyan-300/55">暂无可选设备</td>
                        </tr>
                      ) : (
                        deviceItems.map((device) => (
                          <tr key={device.id} className="border-b border-cyan-500/8 last:border-b-0 hover:bg-cyan-500/5">
                            <td className="px-3 py-2.5">
                              <input
                                type="checkbox"
                                className="size-4 accent-cyan-400"
                                checked={selectedDeviceSet.has(device.id)}
                                onChange={(e) => {
                                  setSelectedDevices((prev) =>
                                    e.target.checked
                                      ? Array.from(new Set([...prev, device.id]))
                                      : prev.filter((id) => id !== device.id),
                                  )
                                }}
                                disabled={isViewMode}
                              />
                            </td>
                            <td className="px-3 py-2.5 font-mono text-cyan-100">{device.sn}</td>
                            <td className="px-3 py-2.5 text-cyan-300/80">{device.name || '—'}</td>
                            <td className="px-3 py-2.5 text-cyan-300/80">{inferDeviceType(device)}</td>
                            <td className="px-3 py-2.5 text-cyan-300/80">{device.isOnline ? '在线' : '离线'}</td>
                          </tr>
                        ))
                      )}
                    </tbody>
                  </table>
                </div>

                <div className="mt-3 flex items-center justify-between font-mono text-[11px] text-cyan-300/60">
                  <span>共 {deviceTotal} 台</span>
                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      className="chip"
                      onClick={() => setDevicePage((page) => Math.max(1, page - 1))}
                      disabled={devicePage <= 1}
                    >
                      <ChevronLeft className="size-3.5" />
                    </button>
                    <span>{devicePage}/{devicePageCount}</span>
                    <button
                      type="button"
                      className="chip"
                      onClick={() => setDevicePage((page) => Math.min(devicePageCount, page + 1))}
                      disabled={devicePage >= devicePageCount}
                    >
                      <ChevronRight className="size-3.5" />
                    </button>
                  </div>
                </div>

                {selectedDeviceRows.length > 0 && (
                  <div className="mt-3 flex flex-wrap gap-2">
                    {selectedDeviceRows.map((device) => (
                      <span key={device.id} className="chip inline-flex items-center gap-2 text-cyan-100">
                        <span className="font-mono">{device.sn}</span>
                        <span className="text-cyan-300/60">{device.name || '未命名设备'}</span>
                        {!isViewMode && (
                          <button type="button" onClick={() => setSelectedDevices((prev) => prev.filter((id) => id !== device.id))}>
                            <X className="size-3" />
                          </button>
                        )}
                      </span>
                    ))}
                  </div>
                )}
              </>
            ) : (
              <>
                <div className="mt-3 flex flex-wrap items-center justify-between gap-3 font-mono text-[11px] text-cyan-300/60">
                  <div className="flex items-center gap-4">
                    <span>已选 {selectedGroups.length} 组</span>
                    <span>共 {groups.length} 组</span>
                  </div>
                  <label className="flex items-center gap-2 text-cyan-100">
                    <input
                      type="checkbox"
                      className="size-4 accent-cyan-400"
                      checked={allGroupsSelected}
                      onChange={(e) => setSelectedGroups(e.target.checked ? groups.map((group) => group.id) : [])}
                      disabled={isViewMode || groups.length === 0}
                    />
                    全选设备组
                  </label>
                </div>

                <div className="mt-3 overflow-hidden rounded-sm border border-cyan-500/15">
                  <table className="w-full text-sm">
                    <thead className="border-b border-cyan-500/10 bg-cyan-500/5">
                      <tr className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/50">
                        <th className="px-3 py-2 text-left">选</th>
                        <th className="px-3 py-2 text-left">设备组名称</th>
                        <th className="px-3 py-2 text-left">设备数</th>
                      </tr>
                    </thead>
                    <tbody>
                      {groups.length === 0 ? (
                        <tr>
                          <td colSpan={3} className="px-3 py-6 text-center text-cyan-300/55">暂无设备组</td>
                        </tr>
                      ) : (
                        groups.map((group: DeviceGroup) => (
                          <tr key={group.id} className="border-b border-cyan-500/8 last:border-b-0 hover:bg-cyan-500/5">
                            <td className="px-3 py-2.5">
                              <input
                                type="checkbox"
                                className="size-4 accent-cyan-400"
                                checked={selectedGroupSet.has(group.id)}
                                onChange={(e) => {
                                  setSelectedGroups((prev) =>
                                    e.target.checked
                                      ? Array.from(new Set([...prev, group.id]))
                                      : prev.filter((id) => id !== group.id),
                                  )
                                }}
                                disabled={isViewMode}
                              />
                            </td>
                            <td className="px-3 py-2.5 text-cyan-100">{group.name}</td>
                            <td className="px-3 py-2.5 text-cyan-300/80">{group.deviceCount}</td>
                          </tr>
                        ))
                      )}
                    </tbody>
                  </table>
                </div>
              </>
            )}
          </section>

          <section className="glass rounded-sm border border-cyan-500/15 px-4 py-4">
            <div>
              <div className="font-mono text-[11px] uppercase tracking-[0.16em] text-cyan-300/60">
                Alarm Scope
              </div>
              <div className="text-sm text-cyan-100">告警范围</div>
            </div>

            <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
              <div className="relative w-full max-w-xs">
                <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-cyan-300/45" />
                <input
                  className="neon-input w-full pl-9"
                  value={alarmKeyword}
                  onChange={(e) => setAlarmKeyword(e.target.value)}
                  placeholder="搜索告警标识或可能原因"
                />
              </div>
              <div className="flex flex-wrap items-center gap-4 font-mono text-[11px] text-cyan-300/60">
                <span>已选 {selectedAlarms.length} 条</span>
                <span>共 {alarmTotal} 条</span>
                <label className="flex items-center gap-2 text-cyan-100">
                  <input
                    type="checkbox"
                    className="size-4 accent-cyan-400"
                    checked={allVisibleAlarmsSelected}
                    onChange={(e) => {
                      const visibleIds = pagedAlarms.map((item) => item.identifier)
                      if (e.target.checked) {
                        setSelectedAlarms((prev) => Array.from(new Set([...prev, ...visibleIds])))
                      } else {
                        setSelectedAlarms((prev) => prev.filter((id) => !visibleIds.includes(id)))
                      }
                    }}
                    disabled={isViewMode || pagedAlarms.length === 0}
                  />
                  当前页全选
                </label>
              </div>
            </div>

            <div className="mt-3 overflow-hidden rounded-sm border border-cyan-500/15">
              <table className="w-full text-sm">
                <thead className="border-b border-cyan-500/10 bg-cyan-500/5">
                  <tr className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/50">
                    <th className="px-3 py-2 text-left">选</th>
                    <th className="px-3 py-2 text-left">告警标识</th>
                    <th className="px-3 py-2 text-left">可能原因</th>
                  </tr>
                </thead>
                <tbody>
                  {alarmDefsQuery.isLoading ? (
                    <tr>
                      <td colSpan={3} className="px-3 py-6 text-center text-cyan-300/55">正在加载告警…</td>
                    </tr>
                  ) : pagedAlarms.length === 0 ? (
                    <tr>
                      <td colSpan={3} className="px-3 py-6 text-center text-cyan-300/55">暂无可选告警</td>
                    </tr>
                  ) : (
                    pagedAlarms.map((alarm) => (
                      <tr key={alarm.identifier} className="border-b border-cyan-500/8 last:border-b-0 hover:bg-cyan-500/5">
                        <td className="px-3 py-2.5">
                          <input
                            type="checkbox"
                            className="size-4 accent-cyan-400"
                            checked={selectedAlarmSet.has(alarm.identifier)}
                            onChange={(e) => {
                              setSelectedAlarms((prev) =>
                                e.target.checked
                                  ? Array.from(new Set([...prev, alarm.identifier]))
                                  : prev.filter((id) => id !== alarm.identifier),
                              )
                            }}
                            disabled={isViewMode}
                          />
                        </td>
                        <td className="px-3 py-2.5 font-mono text-cyan-100">{alarm.identifier}</td>
                        <td className="px-3 py-2.5 text-cyan-300/80">{normalizeAlarmName(alarm)}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>

            <div className="mt-3 flex items-center justify-between font-mono text-[11px] text-cyan-300/60">
              <span>共 {alarmTotal} 条</span>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  className="chip"
                  onClick={() => setAlarmPage((page) => Math.max(1, page - 1))}
                  disabled={alarmPage <= 1}
                >
                  <ChevronLeft className="size-3.5" />
                </button>
                <span>{alarmPage}/{alarmPageCount}</span>
                <button
                  type="button"
                  className="chip"
                  onClick={() => setAlarmPage((page) => Math.min(alarmPageCount, page + 1))}
                  disabled={alarmPage >= alarmPageCount}
                >
                  <ChevronRight className="size-3.5" />
                </button>
              </div>
            </div>

            {selectedAlarmItems.length > 0 && (
              <div className="mt-3 flex flex-wrap gap-2">
                {selectedAlarmItems.map((alarm) => (
                  <span key={alarm.identifier} className="chip inline-flex items-center gap-2 text-cyan-100">
                    <span className="font-mono">{alarm.identifier}</span>
                    <span className="text-cyan-300/60">{normalizeAlarmName(alarm)}</span>
                    {!isViewMode && (
                      <button type="button" onClick={() => setSelectedAlarms((prev) => prev.filter((id) => id !== alarm.identifier))}>
                        <X className="size-3" />
                      </button>
                    )}
                  </span>
                ))}
              </div>
            )}
          </section>
        </div>

        <div className="flex justify-end gap-2 border-t border-cyan-500/15 px-5 py-4">
          <NeonButton onClick={onCancel}>{isViewMode ? '关闭' : '取消'}</NeonButton>
          {!isViewMode && (
            <NeonButton onClick={submit} disabled={!canSubmit}>
              {mode === 'edit' ? '保存' : '创建'}
            </NeonButton>
          )}
        </div>
      </div>
    </div>
  )
}