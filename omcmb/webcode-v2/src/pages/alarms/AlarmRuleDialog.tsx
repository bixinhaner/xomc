import { useEffect, useMemo, useState } from 'react'
import { ChevronLeft, ChevronRight, Search, X } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
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

function normalizeAlarmName(item: { identifier: string; cnName?: string; cnProbableCause?: string; enName?: string; enProbableCause?: string }) {
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

  const selectedAlarmItems = useMemo(() => {
    const itemMap = new Map((alarmDefsQuery.data?.items ?? []).map((item) => [item.identifier, item]))
    return selectedAlarms
      .map((identifier) => itemMap.get(identifier))
      .filter((item): item is NonNullable<typeof itemMap extends Map<string, infer T> ? T : never> => Boolean(item))
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
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40" onClick={onCancel} aria-hidden />
      <div className="relative flex max-h-[88vh] w-full max-w-6xl flex-col rounded-lg border bg-background p-5 shadow-xl">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-base font-semibold">
              {mode === 'add' ? '新建告警规则' : mode === 'edit' ? '编辑告警规则' : '查看告警规则'}
            </h2>
            <p className="mt-1 text-xs text-muted-foreground">
              设备列表、告警列表与 v1 抽屉保持同一选择口径
            </p>
          </div>
          <Button variant="ghost" size="icon" onClick={onCancel}>
            <X className="size-4" />
          </Button>
        </div>

        <div className="mt-4 grid gap-4 md:grid-cols-3">
          <div className="space-y-1.5 md:col-span-1">
            <Label htmlFor="rule-name">规则名称</Label>
            <Input
              id="rule-name"
              value={ruleName}
              placeholder="输入规则名称"
              maxLength={50}
              onChange={(e) => setRuleName(e.target.value)}
              autoFocus
              disabled={loading || isViewMode}
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="rule-type">执行动作</Label>
            <Select value={ruleType} onValueChange={setRuleType} disabled={loading || isViewMode}>
              <SelectTrigger id="rule-type">
                <SelectValue placeholder="选择执行动作" />
              </SelectTrigger>
              <SelectContent>
                {RULE_TYPE_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex items-center justify-between rounded-md border px-3 py-2 md:mt-6">
            <Label htmlFor="rule-enabled">立即启用</Label>
            <button
              id="rule-enabled"
              type="button"
              onClick={() => !isViewMode && setEnabled((value) => !value)}
              disabled={isViewMode}
              className={cn(
                'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
                enabled ? 'bg-primary' : 'bg-muted',
                isViewMode && 'cursor-not-allowed opacity-70',
              )}
              aria-label="切换启用"
            >
              <span
                className={cn(
                  'inline-block size-3.5 transform rounded-full bg-white transition-transform',
                  enabled ? 'translate-x-4' : 'translate-x-1',
                )}
              />
            </button>
          </div>
        </div>

        <div className="mt-4 flex-1 space-y-4 overflow-auto pr-1">
          <section className="rounded-lg border p-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <h3 className="text-sm font-semibold">设备范围</h3>
                <p className="text-xs text-muted-foreground">按设备或设备组限定规则生效范围</p>
              </div>
              <div className="flex rounded-md border bg-muted/40 p-1">
                {(['devices', 'groups'] as const).map((value) => (
                  <button
                    key={value}
                    type="button"
                    disabled={isViewMode}
                    onClick={() => setDeviceSelectionMode(value)}
                    className={cn(
                      'rounded px-3 py-1 text-xs transition-colors',
                      deviceSelectionMode === value ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground',
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
                    <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                    <Input
                      className="pl-9"
                      placeholder="搜索设备 SN/名称"
                      value={deviceKeyword}
                      onChange={(e) => setDeviceKeyword(e.target.value)}
                    />
                  </div>
                  <div className="flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
                    <span>已选 {selectedDevices.length} 台</span>
                    <span>共 {deviceTotal} 台</span>
                    <label className="flex items-center gap-2 text-foreground">
                      <input
                        type="checkbox"
                        className="size-4"
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

                <div className="mt-3 overflow-hidden rounded-md border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead className="w-12">选择</TableHead>
                        <TableHead>SN</TableHead>
                        <TableHead>名称</TableHead>
                        <TableHead>制式</TableHead>
                        <TableHead>状态</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {deviceQuery.isLoading ? (
                        <TableRow>
                          <TableCell colSpan={5} className="text-center text-sm text-muted-foreground">
                            正在加载设备…
                          </TableCell>
                        </TableRow>
                      ) : deviceItems.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={5} className="text-center text-sm text-muted-foreground">
                            暂无可选设备
                          </TableCell>
                        </TableRow>
                      ) : (
                        deviceItems.map((device) => (
                          <TableRow key={device.id}>
                            <TableCell>
                              <input
                                type="checkbox"
                                className="size-4"
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
                            </TableCell>
                            <TableCell className="font-medium">{device.sn}</TableCell>
                            <TableCell>{device.name || '—'}</TableCell>
                            <TableCell>{inferDeviceType(device)}</TableCell>
                            <TableCell>{device.isOnline ? '在线' : '离线'}</TableCell>
                          </TableRow>
                        ))
                      )}
                    </TableBody>
                  </Table>
                </div>

                <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
                  <span>共 {deviceTotal} 台</span>
                  <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" onClick={() => setDevicePage((page) => Math.max(1, page - 1))} disabled={devicePage <= 1}>
                      <ChevronLeft className="size-4" />
                    </Button>
                    <span>{devicePage}/{devicePageCount}</span>
                    <Button variant="outline" size="sm" onClick={() => setDevicePage((page) => Math.min(devicePageCount, page + 1))} disabled={devicePage >= devicePageCount}>
                      <ChevronRight className="size-4" />
                    </Button>
                  </div>
                </div>

                {selectedDeviceRows.length > 0 && (
                  <div className="mt-3 flex flex-wrap gap-2">
                    {selectedDeviceRows.map((device) => (
                      <span key={device.id} className="inline-flex items-center gap-2 rounded-full border px-2.5 py-1 text-xs">
                        <span className="font-medium">{device.sn}</span>
                        <span className="text-muted-foreground">{device.name || '未命名设备'}</span>
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
                <div className="mt-3 flex flex-wrap items-center justify-between gap-3 text-xs text-muted-foreground">
                  <div className="flex items-center gap-4">
                    <span>已选 {selectedGroups.length} 组</span>
                    <span>共 {groups.length} 组</span>
                  </div>
                  <label className="flex items-center gap-2 text-foreground">
                    <input
                      type="checkbox"
                      className="size-4"
                      checked={allGroupsSelected}
                      onChange={(e) => {
                        setSelectedGroups(e.target.checked ? groups.map((group) => group.id) : [])
                      }}
                      disabled={isViewMode || groups.length === 0}
                    />
                    全选设备组
                  </label>
                </div>
                <div className="mt-3 overflow-hidden rounded-md border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead className="w-12">选择</TableHead>
                        <TableHead>设备组名称</TableHead>
                        <TableHead className="w-24">设备数</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {groups.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={3} className="text-center text-sm text-muted-foreground">
                            暂无设备组
                          </TableCell>
                        </TableRow>
                      ) : (
                        groups.map((group: DeviceGroup) => (
                          <TableRow key={group.id}>
                            <TableCell>
                              <input
                                type="checkbox"
                                className="size-4"
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
                            </TableCell>
                            <TableCell>{group.name}</TableCell>
                            <TableCell>{group.deviceCount}</TableCell>
                          </TableRow>
                        ))
                      )}
                    </TableBody>
                  </Table>
                </div>
              </>
            )}
          </section>

          <section className="rounded-lg border p-4">
            <div>
              <h3 className="text-sm font-semibold">告警范围</h3>
              <p className="text-xs text-muted-foreground">至少选择一条告警标识作为命中条件</p>
            </div>
            <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
              <div className="relative w-full max-w-xs">
                <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  className="pl-9"
                  placeholder="搜索告警标识或可能原因"
                  value={alarmKeyword}
                  onChange={(e) => setAlarmKeyword(e.target.value)}
                />
              </div>
              <div className="flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
                <span>已选 {selectedAlarms.length} 条</span>
                <span>共 {alarmTotal} 条</span>
                <label className="flex items-center gap-2 text-foreground">
                  <input
                    type="checkbox"
                    className="size-4"
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

            <div className="mt-3 overflow-hidden rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-12">选择</TableHead>
                    <TableHead>告警标识</TableHead>
                    <TableHead>可能原因</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {alarmDefsQuery.isLoading ? (
                    <TableRow>
                      <TableCell colSpan={3} className="text-center text-sm text-muted-foreground">
                        正在加载告警…
                      </TableCell>
                    </TableRow>
                  ) : pagedAlarms.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={3} className="text-center text-sm text-muted-foreground">
                        暂无可选告警
                      </TableCell>
                    </TableRow>
                  ) : (
                    pagedAlarms.map((alarm) => (
                      <TableRow key={alarm.identifier}>
                        <TableCell>
                          <input
                            type="checkbox"
                            className="size-4"
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
                        </TableCell>
                        <TableCell className="font-medium">{alarm.identifier}</TableCell>
                        <TableCell>{normalizeAlarmName(alarm)}</TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>

            <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
              <span>共 {alarmTotal} 条</span>
              <div className="flex items-center gap-2">
                <Button variant="outline" size="sm" onClick={() => setAlarmPage((page) => Math.max(1, page - 1))} disabled={alarmPage <= 1}>
                  <ChevronLeft className="size-4" />
                </Button>
                <span>{alarmPage}/{alarmPageCount}</span>
                <Button variant="outline" size="sm" onClick={() => setAlarmPage((page) => Math.min(alarmPageCount, page + 1))} disabled={alarmPage >= alarmPageCount}>
                  <ChevronRight className="size-4" />
                </Button>
              </div>
            </div>

            {selectedAlarmItems.length > 0 && (
              <div className="mt-3 flex flex-wrap gap-2">
                {selectedAlarmItems.map((alarm) => (
                  <span key={alarm.identifier} className="inline-flex items-center gap-2 rounded-full border px-2.5 py-1 text-xs">
                    <span className="font-medium">{alarm.identifier}</span>
                    <span className="text-muted-foreground">{normalizeAlarmName(alarm)}</span>
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

        <div className="mt-5 flex justify-end gap-2">
          <Button variant="outline" size="sm" onClick={onCancel} disabled={loading}>
            {isViewMode ? '关闭' : '取消'}
          </Button>
          {!isViewMode && (
            <Button size="sm" onClick={submit} disabled={!canSubmit}>
              {mode === 'edit' ? '保存' : '创建'}
            </Button>
          )}
        </div>
      </div>
    </div>
  )
}
