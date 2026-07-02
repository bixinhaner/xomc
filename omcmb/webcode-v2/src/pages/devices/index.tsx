import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
} from '@tanstack/react-table'
import {
  AlertTriangle,
  CheckCircle2,
  Loader2,
  Power,
  RefreshCw,
  Search,
  Users,
  X,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
import {
  EmptyRow,
  ErrorRow,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useAppStore } from '@core/store/appStore'
import { deviceTaskApi, isAbortError } from '@core/services/api/deviceTaskApi'
import { expandSelectedGroupIds } from '@core/utils/deviceGroupFilter'
import { mapWithConcurrencyLimit } from '@core/utils/asyncPool'
import {
  prefetchDeviceDetailContext,
  useDeviceList,
  useDeviceGroups,
  useBatchRebootDevices,
  useSyncDeviceParams,
  useUpdateDevice,
} from '@core/hooks/api/useDevices'
import { useProductList } from '@core/hooks/api/useProducts'
import { useAlarmCount, useTriggerAlarmSync } from '@core/hooks/api/useAlarms'
import { useDictionary } from '@core/hooks/api/useSystem'
import { activationStatusLabelOf, activationStatusOf } from '@core/utils/activationStatus'
import type { Device, DeviceFilter } from '@core/types/device'
import type { PageRequest } from '@core/types/pagination'
import type { AlarmSeverity } from '@core/types/common'

// ============================================================
// 设备管理 — 主列表 + 概览统计 + 多维筛选 + 批量操作
// 对照 v1 webcode/src/pages/device/DeviceList 的业务深度
// ============================================================

type OnlineFilter = 'all' | 'online' | 'offline'
type OpStateFilter = 'all' | '1' | '0'

const AUTO_REFRESH_OPTIONS = [
  { label: '自动刷新', value: 'off' },
  { label: '15秒', value: '15' },
  { label: '30秒', value: '30' },
  { label: '1分钟', value: '60' },
  { label: '5分钟', value: '300' },
] as const

const ALARM_SYNC_BATCH_CONCURRENCY = 4

const ALARM_VARIANT: Record<
  AlarmSeverity | 'none',
  'destructive' | 'warning' | 'default' | 'muted'
> = {
  critical: 'destructive',
  major: 'destructive',
  minor: 'warning',
  warning: 'warning',
  none: 'muted',
}

const ALARM_LABEL: Record<AlarmSeverity | 'none', string> = {
  critical: '紧急',
  major: '重要',
  minor: '次要',
  warning: '警告',
  none: '无',
}

function ConnStatusBadge({ online }: { online: boolean }) {
  return (
    <Badge variant={online ? 'success' : 'muted'}>
      <span
        className={cn(
          'mr-1 inline-block size-1.5 rounded-full',
          online ? 'bg-emerald-500' : 'bg-muted-foreground/40'
        )}
      />
      {online ? '在线' : '离线'}
    </Badge>
  )
}

function AlarmBadge({ level, count }: { level: AlarmSeverity | 'none'; count?: number }) {
  const label =
    level !== 'none' && (count ?? 0) > 0 ? `${ALARM_LABEL[level]} · ${count}` : ALARM_LABEL[level]
  return <Badge variant={ALARM_VARIANT[level]}>{label}</Badge>
}

function ActivationBadge({
  opState,
  details,
  locale,
}: {
  opState: string | undefined | null
  details?: { label?: string; value?: string; status?: boolean }[]
  locale: 'zh-CN' | 'en-US'
}) {
  const status = activationStatusOf(opState)
  const label = activationStatusLabelOf(opState, details, {
    active: '激活',
    inactive: '未激活',
  }, locale)
  if (status === 'active') return <Badge variant="success">{label}</Badge>
  if (status === 'inactive') return <Badge variant="muted">{label}</Badge>
  return <span className="text-xs text-muted-foreground">—</span>
}

export function DevicesPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [autoRefresh, setAutoRefresh] = useState(false)
  const [refreshInterval, setRefreshInterval] = useState(30)
  const [searchText, setSearchText] = useState('')
  const [onlineFilter, setOnlineFilter] = useState<OnlineFilter>('all')
  const [opState, setOpState] = useState<OpStateFilter>('all')
  const [networkType, setNetworkType] = useState<string>('all')
  const [productId, setProductId] = useState<string>('all')
  const [groupId, setGroupId] = useState<string>('all')
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())

    const prefetchDeviceDetailEntry = useCallback((device: Device) => {
      void import('@/pages/device/DeviceDetail')
      void prefetchDeviceDetailContext(queryClient, device)
    }, [queryClient])

    const openDeviceDetail = useCallback((device: Device) => {
      prefetchDeviceDetailEntry(device)
      void navigate(`/device/detail/${device.sn}`)
    }, [navigate, prefetchDeviceDetailEntry])

    const groupsQuery = useDeviceGroups()
    const productsQuery = useProductList()
    const alarmCountQuery = useAlarmCount()
    const appLocale = useAppStore((s) => s.locale)
    const { data: opStateDict } = useDictionary('op_state')

    const groupOptions = useMemo(
      () => (groupsQuery.data?.groups ?? []).filter((g) => g.parentId !== null),
      [groupsQuery.data]
    )
    const productOptions = productsQuery.data?.items ?? []

    const queryParams = useMemo<DeviceFilter & PageRequest>(() => {
      const expandedGroupIDs =
        groupId !== 'all' ? expandSelectedGroupIds(groupId, groupsQuery.data?.groups ?? []) : undefined

      return {
        page,
        pageSize,
        ...(searchText.trim() ? { searchText: searchText.trim() } : {}),
        ...(onlineFilter !== 'all' ? { isOnline: onlineFilter === 'online' } : {}),
        ...(opState !== 'all' ? { opState } : {}),
        ...(networkType !== 'all' ? { networkType } : {}),
        ...(productId !== 'all' ? { productId } : {}),
        ...(expandedGroupIDs ? { groupId: expandedGroupIDs } : {}),
      }
    }, [page, pageSize, searchText, onlineFilter, opState, networkType, productId, groupId, groupsQuery.data?.groups])

    const { data, isLoading, isError, error, isFetching, refetch } =
      useDeviceList(queryParams, { refetchInterval: autoRefresh ? refreshInterval * 1000 : 0 })

    const batchReboot = useBatchRebootDevices()
    const syncParams = useSyncDeviceParams()
    const alarmSync = useTriggerAlarmSync()
    const updateDevice = useUpdateDevice()
    const batchAlarmSyncAbortRef = useRef<AbortController | null>(null)
    const [batchAlarmSyncRunning, setBatchAlarmSyncRunning] = useState(false)
    const [batchAlarmSyncFeedback, setBatchAlarmSyncFeedback] = useState<{
      tone: 'success' | 'warning' | 'error'
      text: string
    } | null>(null)
    const [editingInstallAddressId, setEditingInstallAddressId] = useState<string | null>(null)
    const [editingInstallAddressValue, setEditingInstallAddressValue] = useState('')
    const [savingInstallAddressId, setSavingInstallAddressId] = useState<string | null>(null)

    useEffect(() => {
      if (!autoRefresh) return
      void refetch()
    }, [autoRefresh, refreshInterval, refetch])

    useEffect(() => {
      return () => {
        batchAlarmSyncAbortRef.current?.abort()
      }
    }, [])

    const rows = useMemo(
      () => data?.items ?? [],
      [data?.items]
    )
    const total = data?.total ?? 0
    const totalPages = Math.max(1, Math.ceil(total / pageSize))
    const stats = data?.stats

    const onlineCount = stats?.online_count ?? stats?.online ?? 0
    const offlineCount = stats?.offline_count ?? stats?.offline ?? 0
    const alarmedCount = alarmCountQuery.data?.total_active ?? stats?.alarmed ?? 0

    const pageIds = useMemo(() => rows.map((d) => d.id), [rows])
    const allOnPageSelected =
      pageIds.length > 0 && pageIds.every((id) => selectedIds.has(id))
    const someOnPageSelected = pageIds.some((id) => selectedIds.has(id))

    function toggleOne(id: string) {
      setSelectedIds((prev) => {
        const next = new Set(prev)
        if (next.has(id)) next.delete(id)
        else next.add(id)
        return next
      })
    }

    const toggleAllOnPage = useCallback(() => {
      setSelectedIds((prev) => {
        const next = new Set(prev)
        if (allOnPageSelected) {
          pageIds.forEach((id) => next.delete(id))
        } else {
          pageIds.forEach((id) => next.add(id))
        }
        return next
      })
    }, [allOnPageSelected, pageIds])

    function clearSelection() {
      setSelectedIds(new Set())
    }

    const selectedDevices = useMemo(
      () => rows.filter((d) => selectedIds.has(d.id)),
      [rows, selectedIds]
    )
    const selectedCount = selectedIds.size

    function handleBatchReboot() {
      if (selectedCount === 0) return
      batchReboot.mutate(Array.from(selectedIds), {
        onSuccess: () => clearSelection(),
      })
    }

    function handleBatchSyncParams() {
      selectedDevices.forEach((d) => {
        syncParams.mutate({ deviceId: d.id })
      })
    }

    function handleBatchAlarmSync() {
      if (selectedCount === 0 || batchAlarmSyncRunning) return
      void (async () => {
        let abortController: AbortController | null = null
        setBatchAlarmSyncRunning(true)
        setBatchAlarmSyncFeedback(null)
        try {
          const runnable = selectedDevices.filter((d) => d.isOnline && Boolean(d.sn))
          const blockedCount = selectedDevices.length - runnable.length
          if (runnable.length === 0) {
            setBatchAlarmSyncFeedback({
              tone: 'warning',
              text: '告警同步仅支持在线设备。',
            })
            return
          }

          batchAlarmSyncAbortRef.current?.abort()
          abortController = new AbortController()
        const activeAbortController = abortController
        batchAlarmSyncAbortRef.current = activeAbortController

          const results = await mapWithConcurrencyLimit(
            runnable,
            ALARM_SYNC_BATCH_CONCURRENCY,
            async (device) => {
              const triggerResult = await alarmSync.mutateAsync(device.sn)
              if (!triggerResult.taskId) {
                throw new Error('告警同步任务不可用，请稍后重试。')
              }
              const task = await deviceTaskApi.waitForTerminal(triggerResult.taskId, {
              signal: activeAbortController.signal,
              })
              if (task.status !== 'completed') {
                throw new Error(task.errorMessage || task.status)
              }
            },
          )

        const aborted = activeAbortController.signal.aborted
            || results.some((result) => result.status === 'rejected' && isAbortError(result.reason))
          if (aborted) {
            return
          }

          const successCount = results.filter((result) => result.status === 'fulfilled').length
          const failedCount = runnable.length - successCount + blockedCount
          if (successCount > 0) {
            await queryClient.invalidateQueries({ queryKey: ['alarms'] })
          }
          if (successCount > 0 && failedCount === 0) {
            setBatchAlarmSyncFeedback({ tone: 'success', text: '告警同步完成。' })
          } else if (successCount > 0) {
            setBatchAlarmSyncFeedback({
              tone: 'warning',
              text: `告警同步部分完成：成功 ${successCount} 台，失败 ${failedCount} 台。`,
            })
          } else {
            setBatchAlarmSyncFeedback({ tone: 'error', text: '告警同步失败。' })
          }
          clearSelection()
        } finally {
          if (abortController && batchAlarmSyncAbortRef.current === abortController) {
            batchAlarmSyncAbortRef.current = null
          }
          if (!(abortController?.signal.aborted ?? false)) {
            setBatchAlarmSyncRunning(false)
          }
        }
      })()
    }

    function resetFilters() {
      setSearchText('')
      setOnlineFilter('all')
      setOpState('all')
      setNetworkType('all')
      setProductId('all')
      setGroupId('all')
      setPage(1)
    }

    const hasActiveFilter =
      Boolean(searchText.trim()) ||
      onlineFilter !== 'all' ||
      opState !== 'all' ||
      networkType !== 'all' ||
      productId !== 'all' ||
      groupId !== 'all'

    const autoRefreshValue = autoRefresh ? String(refreshInterval) : 'off'

    function startInstallAddressEdit(device: Device) {
      setEditingInstallAddressId(device.id)
      setEditingInstallAddressValue(device.installAddress || '')
    }

    const cancelInstallAddressEdit = useCallback(() => {
      setEditingInstallAddressId(null)
      setEditingInstallAddressValue('')
      setSavingInstallAddressId(null)
    }, [])

    const saveInstallAddressEdit = useCallback((device: Device) => {
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
            queryClient.setQueryData(['devices', 'list', queryParams], (prev: typeof data) => {
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
    }, [
      cancelInstallAddressEdit,
      editingInstallAddressValue,
      queryClient,
      queryParams,
      savingInstallAddressId,
      updateDevice,
    ])

    // ---- 列定义 ----
    const columns = useMemo<ColumnDef<Device>[]>(
      () => [
        {
          id: 'select',
          header: () => (
            <input
              type="checkbox"
              aria-label="全选本页"
              className="size-4 cursor-pointer accent-primary"
              checked={allOnPageSelected}
            ref={(el) => {
              if (el) el.indeterminate = !allOnPageSelected && someOnPageSelected
            }}
            onChange={toggleAllOnPage}
          />
        ),
        cell: ({ row }) => (
          <input
            type="checkbox"
            aria-label={`选择 ${row.original.sn}`}
            className="size-4 cursor-pointer accent-primary"
            checked={selectedIds.has(row.original.id)}
            onChange={() => toggleOne(row.original.id)}
          />
        ),
      },
      {
        accessorKey: 'sn',
        header: 'SN',
        cell: ({ row }) => (
          <button
            type="button"
            className="font-mono text-xs text-primary hover:underline"
            onMouseEnter={() => prefetchDeviceDetailEntry(row.original)}
            onFocus={() => prefetchDeviceDetailEntry(row.original)}
            onClick={() => openDeviceDetail(row.original)}
          >
            {row.original.sn || '—'}
          </button>
        ),
      },
      {
        accessorKey: 'deviceName',
        header: '名称',
        cell: ({ row }) => (
          <span className="text-sm">
            {row.original.deviceName || row.original.name || '—'}
          </span>
        ),
      },
      {
        accessorKey: 'isOnline',
        header: '连接状态',
        cell: ({ row }) => <ConnStatusBadge online={row.original.isOnline} />,
      },
      {
        accessorKey: 'alarmLevel',
        header: '告警',
        cell: ({ row }) => <AlarmBadge level={row.original.alarmLevel} count={row.original.activeAlarmCount} />,
      },
      {
        accessorKey: 'networkType',
        header: '制式',
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground">
            {row.original.networkType || '—'}
          </span>
        ),
      },
      {
        accessorKey: 'deviceModel',
        header: '产品型号',
        cell: ({ row }) => (
          <span className="text-xs">
            {row.original.deviceModel || row.original.productClass || '—'}
          </span>
        ),
      },
      {
        accessorKey: 'softwareVersion',
        header: '软件版本',
        cell: ({ row }) => (
          <span className="font-mono text-xs text-muted-foreground">
            {row.original.softwareVersion || '—'}
          </span>
        ),
      },
      {
        accessorKey: 'ipAddress',
        header: 'IP 地址',
        cell: ({ row }) => (
          <span className="font-mono text-xs text-muted-foreground">
            {row.original.ipAddress || '—'}
          </span>
        ),
      },
      {
        accessorKey: 'groupName',
        header: '分组',
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground">
            {row.original.groupName || '—'}
          </span>
        ),
      },
      {
        accessorKey: 'installAddress',
        header: '安装详细地址',
        cell: ({ row }) => {
          const device = row.original
          const isEditing = editingInstallAddressId === device.id
          const isSaving = savingInstallAddressId === device.id
          const isEmptyAddress = !device.installAddress

          if (isEditing) {
            return (
              <Input
                autoFocus
                className="h-8"
                maxLength={256}
                value={editingInstallAddressValue}
                placeholder="安装详细地址"
                onClick={(e) => e.stopPropagation()}
                onChange={(e) => setEditingInstallAddressValue(e.target.value)}
                onBlur={() => saveInstallAddressEdit(device)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') saveInstallAddressEdit(device)
                  if (e.key === 'Escape') cancelInstallAddressEdit()
                }}
              />
            )
          }

          return (
            <div
              title={device.installAddress || '双击编辑安装详细地址'}
              onDoubleClick={(e) => {
                e.stopPropagation()
                startInstallAddressEdit(device)
              }}
              className={cn('min-h-[22px] text-sm', isEmptyAddress && 'text-muted-foreground')}
            >
              {isSaving ? '保存中…' : device.installAddress || '双击编辑安装详细地址'}
            </div>
          )
        },
      },
      {
        accessorKey: 'txPower',
        header: 'Tx Power',
        cell: ({ row }) => (
          <span className="font-mono text-xs text-muted-foreground">
            {row.original.txPower || '—'}
          </span>
        ),
      },
      {
        accessorKey: 'opState',
        header: '激活状态',
        cell: ({ row }) => <ActivationBadge opState={row.original.opState} details={opStateDict?.sysDictionaryDetails} locale={appLocale} />,
      },
      {
        accessorKey: 'lastOnlineTime',
        header: '最近在线',
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground">
            {formatTime(row.original.lastOnlineTime)}
          </span>
        ),
      },
      {
        accessorKey: 'ueCount',
        header: '在线 UE',
        cell: ({ row }) => {
          const d = row.original
          const label = d.deviceName || d.name || d.sn
          return (
            <button
              type="button"
              className="inline-flex items-center gap-1 text-xs text-primary tabular-nums hover:underline"
              onClick={() =>
                navigate(
                  `/device/ue-detail/${encodeURIComponent(d.sn)}?name=${encodeURIComponent(label || '')}&ueCount=${d.ueCount ?? 0}`
                )
              }
            >
              <Users className="size-3.5" />
              {d.ueCount ?? 0}
            </button>
          )
        },
      },
    ],
    [
      allOnPageSelected,
      appLocale,
      cancelInstallAddressEdit,
      editingInstallAddressId,
      editingInstallAddressValue,
      navigate,
      openDeviceDetail,
      opStateDict?.sysDictionaryDetails,
      prefetchDeviceDetailEntry,
      saveInstallAddressEdit,
      savingInstallAddressId,
      selectedIds,
      someOnPageSelected,
      toggleAllOnPage,
    ]
  )

  const table = useReactTable({
    data: rows,
    columns,
    getCoreRowModel: getCoreRowModel(),
  })

  const colCount = columns.length

  return (
    <PageShell
      title="设备管理"
      description={`共 ${total} 台设备 · 在线 ${onlineCount} · 离线 ${offlineCount}`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          {/* 搜索 */}
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-72 pl-9"
              placeholder="搜索 SN / 名称 / IP / MAC"
              value={searchText}
              onChange={(e) => {
                setSearchText(e.target.value)
                setPage(1)
              }}
            />
          </div>

          {/* 连接状态 */}
          <Select
            value={onlineFilter}
            onValueChange={(v) => {
              setOnlineFilter(v as OnlineFilter)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="连接状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              <SelectItem value="online">在线</SelectItem>
              <SelectItem value="offline">离线</SelectItem>
            </SelectContent>
          </Select>

          {/* 激活状态 */}
          <Select
            value={opState}
            onValueChange={(v) => {
              setOpState(v as OpStateFilter)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="激活状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部激活</SelectItem>
              <SelectItem value="1">激活</SelectItem>
              <SelectItem value="0">未激活</SelectItem>
            </SelectContent>
          </Select>

          {/* 制式 */}
          <Select
            value={networkType}
            onValueChange={(v) => {
              setNetworkType(v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="制式" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部制式</SelectItem>
              <SelectItem value="eNB">eNB (LTE)</SelectItem>
              <SelectItem value="gNB">gNB (NR)</SelectItem>
              <SelectItem value="GSM">GSM</SelectItem>
            </SelectContent>
          </Select>

          {/* 产品 */}
          <Select
            value={productId}
            onValueChange={(v) => {
              setProductId(v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-40">
              <SelectValue placeholder="产品" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部产品</SelectItem>
              {productOptions.map((p) => (
                <SelectItem key={p.id} value={p.id}>
                  {p.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {/* 分组 */}
          <Select
            value={groupId}
            onValueChange={(v) => {
              setGroupId(v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-44">
              <SelectValue placeholder="分组" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部分组</SelectItem>
              {groupOptions.map((g) => (
                <SelectItem key={g.id} value={g.id}>
                  {g.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {hasActiveFilter && (
            <Button variant="ghost" size="sm" onClick={resetFilters}>
              <X className="size-4" /> 重置
            </Button>
          )}

          <div className="ml-auto flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCw className="size-4" /> 刷新
            </Button>
            <Select
              value={autoRefreshValue}
              onValueChange={(value) => {
                if (value === 'off') {
                  setAutoRefresh(false)
                  return
                }
                setRefreshInterval(Number(value))
                setAutoRefresh(true)
              }}
            >
              <SelectTrigger className="w-32">
                <SelectValue placeholder="自动刷新" />
              </SelectTrigger>
              <SelectContent>
                {AUTO_REFRESH_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      }
    >
      {/* 概览统计 */}
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="全部设备" value={stats?.total ?? total} />
        <Stat label="在线" value={onlineCount} tone="emerald" />
        <Stat label="离线" value={offlineCount} tone="muted" />
        <Stat label="有告警" value={alarmedCount} tone="amber" />
      </div>

      {/* 批量操作条 */}
      {selectedCount > 0 && (
        <div className="mb-3 flex flex-wrap items-center gap-2 rounded-lg border border-primary/30 bg-primary/5 px-4 py-2.5 text-sm">
          <CheckCircle2 className="size-4 text-primary" />
          <span className="font-medium">已选 {selectedCount} 台</span>
          <div className="ml-auto flex flex-wrap items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={batchReboot.isPending}
              onClick={handleBatchReboot}
            >
              {batchReboot.isPending ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <Power className="size-4" />
              )}
              批量重启
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={syncParams.isPending}
              onClick={handleBatchSyncParams}
            >
              {syncParams.isPending ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <RefreshCw className="size-4" />
              )}
              同步参数
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={alarmSync.isPending || batchAlarmSyncRunning}
              onClick={handleBatchAlarmSync}
            >
              {alarmSync.isPending || batchAlarmSyncRunning ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <AlertTriangle className="size-4" />
              )}
              同步告警
            </Button>
            <Button variant="ghost" size="sm" onClick={clearSelection}>
              <X className="size-4" /> 取消选择
            </Button>
          </div>
        </div>
      )}

      {/* 批量操作错误反馈 */}
      {batchReboot.isError && (
        <div className="mb-3 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-2 text-sm text-destructive">
          批量重启失败：
          {batchReboot.error instanceof Error
            ? batchReboot.error.message
            : '未知错误'}
        </div>
      )}
      {batchAlarmSyncFeedback && (
        <div
          className={cn(
            'mb-3 rounded-md border px-4 py-2 text-sm',
            batchAlarmSyncFeedback.tone === 'success' && 'border-emerald-500/30 bg-emerald-500/5 text-emerald-700',
            batchAlarmSyncFeedback.tone === 'warning' && 'border-amber-500/30 bg-amber-500/5 text-amber-700',
            batchAlarmSyncFeedback.tone === 'error' && 'border-destructive/30 bg-destructive/5 text-destructive',
          )}
        >
          {batchAlarmSyncFeedback.text}
        </div>
      )}

      {/* 表格 */}
      <TableCard>
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((hg) => (
              <TableRow key={hg.id}>
                {hg.headers.map((h) => (
                  <TableHead key={h.id}>
                    {h.isPlaceholder
                      ? null
                      : flexRender(h.column.columnDef.header, h.getContext())}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={colCount}>
                {hasActiveFilter ? '没有匹配的设备' : '暂无设备'}
              </EmptyRow>
            ) : (
              table.getRowModel().rows.map((row) => (
                <TableRow
                  key={row.id}
                  data-state={
                    selectedIds.has(row.original.id) ? 'selected' : undefined
                  }
                >
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      {/* 分页 */}
      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={pageSize}
        onChange={setPage}
      />
    </PageShell>
  )
}

function Stat({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: number
  tone?: 'default' | 'emerald' | 'amber' | 'muted'
}) {
  const toneClass = {
    default: 'text-foreground',
    emerald: 'text-emerald-600 dark:text-emerald-400',
    amber: 'text-amber-600 dark:text-amber-400',
    muted: 'text-muted-foreground',
  }[tone]

  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>
        {value}
      </div>
    </div>
  )
}
