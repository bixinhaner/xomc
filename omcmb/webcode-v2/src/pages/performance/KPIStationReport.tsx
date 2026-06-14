import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Search } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
  formatTime,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useDeviceList } from '@core/hooks/api/useDevices'
import type { DeviceFilter } from '@core/types/device'
import type { PageRequest } from '@core/types/pagination'

// ============================================================
// 基站性能上报维护 — 对齐 v1 /performance/kpi-station
//   逐基站列出 PM 上报状态（reporting / online），行可进设备详情。
// ============================================================

const PAGE_SIZE = 20

function Stat({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: number | string
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
      <div className="text-xs uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>{value}</div>
    </div>
  )
}

export function KPIStationReportPage() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')

  const params = useMemo<DeviceFilter & PageRequest>(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { searchText: keyword.trim() } : {}),
    }),
    [page, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useDeviceList(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const reportingCount = useMemo(
    () => rows.filter((d) => d.pmReportStatus === 'enabled' || d.isOnline).length,
    [rows]
  )
  const onlineCount = useMemo(() => rows.filter((d) => d.isOnline).length, [rows])

  const cols = ['序列号', '基站名称', '小区 ID', 'PM 上报', '在线状态', '最近上线时间']

  return (
    <PageShell
      title="基站性能上报"
      description="逐基站查看 PM 性能采集 / 上报状态，点击序列号进设备详情"
      isFetching={isFetching}
    >
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="本页基站" value={rows.length} />
        <Stat label="基站总数" value={total} tone="emerald" />
        <Stat label="本页上报中" value={reportingCount} tone="amber" />
        <Stat label="本页在线" value={onlineCount} tone="muted" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-72 pl-9"
            placeholder="SN / 基站名称 / IP"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <Button
          variant="outline"
          size="sm"
          className="ml-auto"
          disabled={isFetching}
          onClick={() => void refetch()}
        >
          <RefreshCcw className="size-4" /> 刷新
        </Button>
      </div>

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无基站数据</EmptyRow>
            ) : (
              rows.map((d) => {
                const reporting = d.pmReportStatus === 'enabled' || d.isOnline
                return (
                  <TableRow key={d.id}>
                    <TableCell className="font-mono text-xs">
                      <button
                        type="button"
                        className="text-primary hover:underline"
                        onClick={() => navigate(`/devices/detail/${d.sn}`)}
                      >
                        {d.sn || '—'}
                      </button>
                    </TableCell>
                    <TableCell className="font-medium">
                      {d.hostName || d.name || d.deviceName || '—'}
                    </TableCell>
                    <TableCell className="font-mono text-xs">{d.cellId || '—'}</TableCell>
                    <TableCell>
                      <Badge variant={reporting ? 'success' : 'muted'}>
                        {reporting ? '上报中' : '未上报'}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={d.isOnline ? 'success' : 'muted'}>
                        {d.isOnline ? '在线' : '离线'}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(d.lastOnlineTime)}
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </PageShell>
  )
}

export default KPIStationReportPage
