import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ChevronRight, Layers, RefreshCcw, Search } from 'lucide-react'

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
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useDeviceList } from '@core/hooks/api/useDevices'
import { DEFAULT_ALARM_SEVERITY_LABELS_ZH, formatAlarmSeverityBadgeLabel, getAlarmSeverityBadgeVariant } from '@core/utils/alarmSeverity'
import type { DeviceFilter } from '@core/types/device'
import type { PageRequest } from '@core/types/pagination'

// ============================================================
// 系统管理 / 设备分类 — 对齐 v1 webcode/src/pages/system/DeviceClassification
// v1 为静态 mock 树 + mock 设备；这里改为真实数据 useDeviceList：
//   左侧按制式（eNB/gNB/GSM）分类树（计数来自后端 stats.byType 或本页聚合），
//   右侧按所选分类过滤的真实设备表，点 SN 进 /devices/detail/:sn。
// ============================================================

const PAGE_SIZE = 20

type Category = { key: string; label: string; networkType?: string }

const CATEGORIES: Category[] = [
  { key: 'all', label: '全部设备' },
  { key: 'enb', label: '4G 设备 (eNB / LTE)', networkType: 'eNB' },
  { key: 'gnb', label: '5G 设备 (gNB / NR)', networkType: 'gNB' },
  { key: 'gsm', label: '2G 设备 (GSM)', networkType: 'GSM' },
]

export default function DeviceClassification() {
  const navigate = useNavigate()
  const [category, setCategory] = useState<string>('all')
  const [searchText, setSearchText] = useState('')
  const [page, setPage] = useState(1)

  const selected = CATEGORIES.find((c) => c.key === category) ?? CATEGORIES[0]

  const params = useMemo<DeviceFilter & PageRequest>(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(searchText.trim() ? { searchText: searchText.trim() } : {}),
      ...(selected.networkType ? { networkType: selected.networkType } : {}),
    }),
    [page, searchText, selected.networkType]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useDeviceList(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const stats = data?.stats

  const cols = ['SN', '名称', '厂商', '产品型号', '制式', '连接', '告警', '软件版本']

  return (
    <PageShell
      title="设备分类"
      description={`按制式分类浏览设备 · 当前分类 ${total} 台`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-72 pl-9"
              placeholder="搜索 SN / 名称"
              value={searchText}
              onChange={(e) => {
                setSearchText(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw className="size-4" /> 刷新
            </Button>
          </div>
        </div>
      }
    >
      <div className="grid gap-4 lg:grid-cols-[280px_1fr]">
        {/* 左：分类树 */}
        <div className="overflow-hidden rounded-lg border bg-card">
          <div className="flex items-center gap-2 border-b px-3 py-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
            <Layers className="size-3.5" /> 设备分类
          </div>
          <ul className="p-1">
            {CATEGORIES.map((c) => {
              const active = c.key === category
              return (
                <li key={c.key}>
                  <button
                    type="button"
                    onClick={() => {
                      setCategory(c.key)
                      setPage(1)
                    }}
                    className={cn(
                      'flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm transition-colors',
                      active
                        ? 'bg-primary/10 text-primary'
                        : 'hover:bg-muted'
                    )}
                  >
                    <ChevronRight
                      className={cn('size-3.5', active && 'text-primary')}
                    />
                    {c.label}
                  </button>
                </li>
              )
            })}
          </ul>
        </div>

        {/* 右：设备表 */}
        <div>
          {stats ? (
            <div className="mb-3 flex flex-wrap gap-2 text-xs text-muted-foreground">
              <Badge variant="muted">在线 {stats.online_count ?? stats.online ?? 0}</Badge>
              <Badge variant="muted">离线 {stats.offline_count ?? stats.offline ?? 0}</Badge>
            </div>
          ) : null}
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
                  <EmptyRow colSpan={cols.length}>该分类下暂无设备</EmptyRow>
                ) : (
                  rows.map((d) => (
                    <TableRow key={d.id}>
                      <TableCell>
                        <button
                          type="button"
                          className="font-mono text-xs text-primary hover:underline"
                          onClick={() => navigate(`/device/detail/${d.sn}`)}
                        >
                          {d.sn || '—'}
                        </button>
                      </TableCell>
                      <TableCell className="text-sm">
                        {d.deviceName || d.name || '—'}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {d.vendor || '—'}
                      </TableCell>
                      <TableCell className="text-xs">
                        {d.deviceModel || d.productClass || '—'}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {d.networkType || '—'}
                      </TableCell>
                      <TableCell>
                        <Badge variant={d.isOnline ? 'success' : 'muted'}>
                          {d.isOnline ? '在线' : '离线'}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant={getAlarmSeverityBadgeVariant(d.alarmLevel)}>
                          {formatAlarmSeverityBadgeLabel(d.alarmLevel, 0, DEFAULT_ALARM_SEVERITY_LABELS_ZH)}
                        </Badge>
                      </TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {d.softwareVersion || '—'}
                      </TableCell>
                    </TableRow>
                  ))
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
        </div>
      </div>
    </PageShell>
  )
}
