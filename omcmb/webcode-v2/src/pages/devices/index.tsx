import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
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
  RefreshCcw,
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

import {
  useDeviceList,
  useDeviceGroups,
  useBatchRebootDevices,
  useSyncDeviceParams,
} from '@core/hooks/api/useDevices'
import { useProductList } from '@core/hooks/api/useProducts'
import { useAlarmCount, useTriggerAlarmSync } from '@core/hooks/api/useAlarms'
import type { Device, DeviceFilter } from '@core/types/device'
import type { PageRequest } from '@core/types/pagination'
import type { AlarmSeverity } from '@core/types/common'

// ============================================================
// 设备管理 — 主列表 + 概览统计 + 多维筛选 + 批量操作
// 对照 v1 webcode/src/pages/device/DeviceList 的业务深度
// ============================================================

type OnlineFilter = 'all' | 'online' | 'offline'
type OpStateFilter = 'all' | '1' | '0'

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
  // #361: 有活动告警时把告警数拼进 label（如「重要 · 3」）。
  const label =
    level !== 'none' && (count ?? 0) > 0 ? `${ALARM_LABEL[level]} · ${count}` : ALARM_LABEL[level]
  return <Badge variant={ALARM_VARIANT[level]}>{label}</Badge>
}

function ActivationBadge({ opState }: { opState: string }) {
  // 「激活状态」判定走 frontend-core/utils/activationStatus —— 与 webcode/webcode-v3
  // 同一来源,后端兜底的 'unknown' 与空值一律显示 '—',不再回显 raw 字符串。
  const status = activationStatusOf(opState)
  if (status === 'active') return <Badge variant="success">激活</Badge>
  if (status === 'inactive') return <Badge variant="muted">未激活</Badge>
  return <span className="text-xs text-muted-foreground">—</span>
}

export function DevicesPage() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [searchText, setSearchText] = useState('')
  const [onlineFilter, setOnlineFilter] = useState<OnlineFilter>('all')
  const [opState, setOpState] = useState<OpStateFilter>('all')
  const [networkType, setNetworkType] = useState<string>('all')
  const [productId, setProductId] = useState<string>('all')
  const [groupId, setGroupId] = useState<string>('all')
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())

  // ---- 辅助下拉数据 ----
  const groupsQuery = useDeviceGroups()
  const productsQuery = useProductList()
  // 全量在线告警计数（与列表 stats.alarmed 占位字段相比更准确，v1 同此做法）
  const alarmCountQuery = useAlarmCount()

  const groupOptions = useMemo(
    () => (groupsQuery.data?.groups ?? []).filter((g) => g.parentId !== null),
    [groupsQuery.data]
  )
  const productOptions = productsQuery.data?.items ?? []

  // ---- 列表查询参数 ----
  const queryParams = useMemo<DeviceFilter & PageRequest>(
    () => ({
      page,
      pageSize,
      ...(searchText.trim() ? { searchText: searchText.trim() } : {}),
      ...(onlineFilter !== 'all' ? { isOnline: onlineFilter === 'online' } : {}),
      ...(opState !== 'all' ? { opState } : {}),
      ...(networkType !== 'all' ? { networkType } : {}),
      ...(productId !== 'all' ? { productId } : {}),
      ...(groupId !== 'all' ? { groupId } : {}),
    }),
    [page, pageSize, searchText, onlineFilter, opState, networkType, productId, groupId]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useDeviceList(queryParams)

  // ---- 批量操作 mutations ----
  const batchReboot = useBatchRebootDevices()
  const syncParams = useSyncDeviceParams()
  const alarmSync = useTriggerAlarmSync()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const stats = data?.stats

  const onlineCount = stats?.online_count ?? stats?.online ?? 0
  const offlineCount = stats?.offline_count ?? stats?.offline ?? 0
  const alarmedCount = alarmCountQuery.data?.total_active ?? stats?.alarmed ?? 0

  // ---- 选择 ----
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

  function toggleAllOnPage() {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (allOnPageSelected) {
        pageIds.forEach((id) => next.delete(id))
      } else {
        pageIds.forEach((id) => next.add(id))
      }
      return next
    })
  }

  function clearSelection() {
    setSelectedIds(new Set())
  }

  const selectedDevices = useMemo(
    () => rows.filter((d) => selectedIds.has(d.id)),
    [rows, selectedIds]
  )
  const selectedCount = selectedIds.size

  // ---- 批量操作处理 ----
  function handleBatchReboot() {
    if (selectedCount === 0) return
    batchReboot.mutate(Array.from(selectedIds), {
      onSuccess: () => clearSelection(),
    })
  }

  function handleBatchSyncParams() {
    // useSyncDeviceParams 是单设备 mutation；批量场景顺序触发（每设备一次 Path B）
    selectedDevices.forEach((d) => {
      syncParams.mutate({ deviceId: d.id })
    })
  }

  function handleBatchAlarmSync() {
    // alarm sync 以 SN 为入参
    selectedDevices.forEach((d) => {
      if (d.sn) alarmSync.mutate(d.sn)
    })
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
            onClick={() => navigate(`/devices/detail/${row.original.sn}`)}
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
        accessorKey: 'opState',
        header: '激活状态',
        cell: ({ row }) => <ActivationBadge opState={row.original.opState} />,
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
                  `/devices/ue-detail/${encodeURIComponent(d.sn)}?name=${encodeURIComponent(label || '')}&ueCount=${d.ueCount ?? 0}`
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
    [allOnPageSelected, someOnPageSelected, selectedIds]
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

          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw className="size-4" /> 刷新
            </Button>
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
              disabled={alarmSync.isPending}
              onClick={handleBatchAlarmSync}
            >
              {alarmSync.isPending ? (
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
