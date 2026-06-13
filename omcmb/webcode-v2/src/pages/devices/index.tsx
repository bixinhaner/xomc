import { useMemo, useState } from 'react'
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
} from '@tanstack/react-table'
import {
  ChevronLeft,
  ChevronRight,
  Loader2,
  Radio,
  Search,
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
import { cn } from '@/lib/utils'

import { useDeviceList } from '@core/hooks/api/useDevices'
import type { Device, ConnStatus, DeviceFilter } from '@core/types/device'
import type { PageRequest } from '@core/types/pagination'
import type { AlarmSeverity } from '@core/types/common'

type ConnFilter = '' | '1' | '0'

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

function ConnStatusBadge({ status }: { status: ConnStatus }) {
  return (
    <Badge variant={status === 'online' ? 'success' : 'muted'}>
      <span
        className={cn(
          'mr-1 inline-block size-1.5 rounded-full',
          status === 'online' ? 'bg-emerald-500' : 'bg-muted-foreground/40'
        )}
      />
      {status === 'online' ? '在线' : '离线'}
    </Badge>
  )
}

function AlarmBadge({ level }: { level: AlarmSeverity | 'none' }) {
  return <Badge variant={ALARM_VARIANT[level]}>{ALARM_LABEL[level]}</Badge>
}

function formatTime(iso?: string) {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function DevicesPage() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [searchText, setSearchText] = useState('')
  const [connStatus, setConnStatus] = useState<ConnFilter>('')

  const queryParams = useMemo<DeviceFilter & PageRequest>(
    () => ({
      page,
      pageSize,
      ...(searchText.trim() ? { searchText: searchText.trim() } : {}),
      // 后端/mock 接受数字型状态码（'0' 离线，'1' 在线），运行时做 map，这里用 as 绕过类型对齐
      ...(connStatus ? { connStatus: connStatus as unknown as ConnStatus } : {}),
    }),
    [page, pageSize, searchText, connStatus]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useDeviceList(queryParams)

  const columns = useMemo<ColumnDef<Device>[]>(
    () => [
      {
        accessorKey: 'sn',
        header: 'SN',
        cell: ({ getValue }) => (
          <span className="font-mono text-xs">{String(getValue() ?? '—')}</span>
        ),
      },
      { accessorKey: 'name', header: '名称' },
      { accessorKey: 'vendor', header: '厂商' },
      { accessorKey: 'productClass', header: '产品型号' },
      { accessorKey: 'networkType', header: '制式' },
      {
        accessorKey: 'connStatus',
        header: '连接状态',
        cell: ({ row }) => <ConnStatusBadge status={row.original.connStatus} />,
      },
      {
        accessorKey: 'alarmLevel',
        header: '告警',
        cell: ({ row }) => <AlarmBadge level={row.original.alarmLevel} />,
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
    ],
    []
  )

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const table = useReactTable({
    data: rows,
    columns,
    getCoreRowModel: getCoreRowModel(),
  })

  return (
    <div className="mx-auto max-w-[1400px] px-6 py-8">
      <div className="mb-6 flex items-end justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">设备管理</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            端到端验证：@core/hooks/api/useDeviceList + TanStack Table + shadcn
          </p>
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          {isFetching && <Loader2 className="size-3.5 animate-spin" />}
          <span>总计 {total} 台设备</span>
        </div>
      </div>

      {/* Stats */}
      {data?.stats && (
        <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
          <Stat label="全部" value={data.stats.total} />
          <Stat label="在线" value={data.stats.online ?? 0} tone="emerald" />
          <Stat label="离线" value={data.stats.offline ?? 0} tone="muted" />
          <Stat label="有告警" value={data.stats.alarmed} tone="amber" />
        </div>
      )}

      {/* Toolbar */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-80 pl-9"
            placeholder="搜索 SN / 名称 / IP"
            value={searchText}
            onChange={(e) => {
              setSearchText(e.target.value)
              setPage(1)
            }}
          />
        </div>

        <Select
          value={connStatus || 'all'}
          onValueChange={(v) => {
            setConnStatus(v === 'all' ? '' : (v as ConnFilter))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="连接状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="1">在线</SelectItem>
            <SelectItem value="0">离线</SelectItem>
          </SelectContent>
        </Select>

        <div className="ml-auto">
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <Radio /> 刷新
          </Button>
        </div>
      </div>

      {/* Table */}
      <div className="overflow-hidden rounded-lg border bg-card">
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
              <TableRow>
                <TableCell colSpan={columns.length} className="h-32 text-center">
                  <Loader2 className="mx-auto size-5 animate-spin text-muted-foreground" />
                </TableCell>
              </TableRow>
            ) : isError ? (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-32 text-center text-destructive"
                >
                  加载失败：{error instanceof Error ? error.message : '未知错误'}
                </TableCell>
              </TableRow>
            ) : rows.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-32 text-center text-muted-foreground"
                >
                  没有匹配的设备
                </TableCell>
              </TableRow>
            ) : (
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id}>
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
      </div>

      {/* Pagination */}
      <div className="mt-4 flex items-center justify-between text-sm">
        <span className="text-muted-foreground">
          第 {page} / {totalPages} 页 · 每页 {pageSize} 条
        </span>
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            variant="outline"
            disabled={page <= 1}
            onClick={() => setPage((p) => Math.max(1, p - 1))}
          >
            <ChevronLeft /> 上一页
          </Button>
          <Button
            size="sm"
            variant="outline"
            disabled={page >= totalPages}
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
          >
            下一页 <ChevronRight />
          </Button>
        </div>
      </div>
    </div>
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
