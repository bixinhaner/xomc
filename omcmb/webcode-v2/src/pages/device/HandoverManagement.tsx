import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Search, X } from 'lucide-react'

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

import { useDeviceList } from '@core/hooks/api/useDevices'
import type { Device, DeviceFilter, DeviceLifecycle } from '@core/types/device'
import type { PageRequest } from '@core/types/pagination'

// ============================================================
// 交维管理 — 建设交付给运维的设备交接跟踪
// 对照 v1 webcode/src/pages/device/HandoverManagement：v1 列表为 mock 交维任务。
// v2 用真实 useDeviceList，按生命周期还原交维状态：
//   provisioning/registered → 待交维；commissioned → 交维中；maintenance → 已交维；
//   decommissioned → 已退役。点击 SN 跳设备详情。
// ============================================================

const PAGE_SIZE = 20

interface HandoverMeta {
  label: string
  variant: 'default' | 'success' | 'warning' | 'destructive' | 'muted'
  fromTeam: string
  toTeam: string
}

const HANDOVER_META: Record<DeviceLifecycle, HandoverMeta> = {
  discovered: {
    label: '待交维',
    variant: 'muted',
    fromTeam: '建设组',
    toTeam: '运维组',
  },
  registered: {
    label: '待交维',
    variant: 'muted',
    fromTeam: '建设组',
    toTeam: '运维组',
  },
  provisioning: {
    label: '待交维',
    variant: 'warning',
    fromTeam: '建设组',
    toTeam: '运维组',
  },
  commissioned: {
    label: '交维中',
    variant: 'warning',
    fromTeam: '建设组',
    toTeam: '运维组',
  },
  maintenance: {
    label: '已交维',
    variant: 'success',
    fromTeam: '建设组',
    toTeam: '运维组',
  },
  decommissioned: {
    label: '已退役',
    variant: 'destructive',
    fromTeam: '运维组',
    toTeam: '—',
  },
}

type StatusFilter = 'all' | 'pending' | 'inProgress' | 'done'

const STATUS_TO_LIFECYCLE: Record<
  Exclude<StatusFilter, 'all'>,
  DeviceLifecycle[]
> = {
  pending: ['provisioning', 'registered'],
  inProgress: ['commissioned'],
  done: ['maintenance'],
}

export default function HandoverManagement() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [searchText, setSearchText] = useState('')
  const [status, setStatus] = useState<StatusFilter>('all')

  const queryParams = useMemo<DeviceFilter & PageRequest>(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(searchText.trim() ? { searchText: searchText.trim() } : {}),
      ...(status !== 'all'
        ? { lifecycleState: STATUS_TO_LIFECYCLE[status] }
        : {}),
    }),
    [page, searchText, status]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useDeviceList(queryParams)

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const colCount = 8

  const hasActiveFilter = Boolean(searchText.trim()) || status !== 'all'

  return (
    <PageShell
      title="交维管理"
      description={`共 ${total} 台设备 · 建设交付与运维交接跟踪`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-72 pl-9"
              placeholder="搜索 SN / 名称 / IP"
              value={searchText}
              onChange={(e) => {
                setSearchText(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <Select
            value={status}
            onValueChange={(v) => {
              setStatus(v as StatusFilter)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="交维状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              <SelectItem value="pending">待交维</SelectItem>
              <SelectItem value="inProgress">交维中</SelectItem>
              <SelectItem value="done">已交维</SelectItem>
            </SelectContent>
          </Select>
          {hasActiveFilter && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setSearchText('')
                setStatus('all')
                setPage(1)
              }}
            >
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
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>SN</TableHead>
              <TableHead>设备名称</TableHead>
              <TableHead>交维状态</TableHead>
              <TableHead>交出方</TableHead>
              <TableHead>接收方</TableHead>
              <TableHead>分组</TableHead>
              <TableHead>连接状态</TableHead>
              <TableHead>创建时间</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={colCount}>
                {hasActiveFilter ? '没有匹配的设备' : '暂无交维任务'}
              </EmptyRow>
            ) : (
              rows.map((d: Device) => {
                const meta =
                  HANDOVER_META[d.lifecycleState] ?? HANDOVER_META.discovered
                return (
                  <TableRow key={d.id}>
                    <TableCell>
                      <button
                        type="button"
                        className="font-mono text-xs text-primary hover:underline"
                        onClick={() => navigate(`/devices/detail/${d.sn}`)}
                      >
                        {d.sn || '—'}
                      </button>
                    </TableCell>
                    <TableCell className="text-sm">
                      {d.deviceName || d.name || '—'}
                    </TableCell>
                    <TableCell>
                      <Badge variant={meta.variant}>{meta.label}</Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {meta.fromTeam}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {meta.toTeam}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {d.groupName || '—'}
                    </TableCell>
                    <TableCell>
                      <Badge variant={d.isOnline ? 'success' : 'muted'}>
                        {d.isOnline ? '在线' : '离线'}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(d.createTime)}
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={PAGE_SIZE}
        onChange={setPage}
      />
    </PageShell>
  )
}
