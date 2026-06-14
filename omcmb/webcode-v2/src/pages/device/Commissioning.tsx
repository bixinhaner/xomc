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
import { cn } from '@/lib/utils'

import { useDeviceList } from '@core/hooks/api/useDevices'
import type { Device, DeviceFilter, DeviceLifecycle } from '@core/types/device'
import type { PageRequest } from '@core/types/pagination'

// ============================================================
// 开站调测 — 处于开站/调测生命周期的设备及其进度
// 对照 v1 webcode/src/pages/device/Commissioning：v1 列表为 mock 调测任务。
// v2 用真实 useDeviceList，按业务生命周期(lifecycleState) 还原"调测任务"语义：
//   discovered/registered/provisioning/commissioned/maintenance 各对应一个阶段，
//   进度由生命周期推导。点击 SN 跳设备详情。
// ============================================================

const PAGE_SIZE = 20

// 生命周期 → 调测进度（百分比）+ 中文阶段名 + 状态语义
const LIFECYCLE_META: Record<
  DeviceLifecycle,
  {
    step: string
    progress: number
    variant: 'default' | 'success' | 'warning' | 'destructive' | 'muted'
  }
> = {
  discovered: { step: '设备发现', progress: 20, variant: 'muted' },
  registered: { step: '注册完成', progress: 40, variant: 'default' },
  provisioning: { step: '配置下发', progress: 70, variant: 'warning' },
  commissioned: { step: '调测完成', progress: 100, variant: 'success' },
  maintenance: { step: '运维中', progress: 100, variant: 'default' },
  decommissioned: { step: '已退役', progress: 0, variant: 'destructive' },
}

const TYPE_VARIANT: Record<string, 'default' | 'success' | 'warning'> = {
  eNB: 'default',
  gNB: 'success',
  GSM: 'warning',
}

type LifecycleFilter = 'all' | DeviceLifecycle

export default function Commissioning() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [searchText, setSearchText] = useState('')
  const [lifecycle, setLifecycle] = useState<LifecycleFilter>('all')

  const queryParams = useMemo<DeviceFilter & PageRequest>(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(searchText.trim() ? { searchText: searchText.trim() } : {}),
      ...(lifecycle !== 'all' ? { lifecycleState: [lifecycle] } : {}),
    }),
    [page, searchText, lifecycle]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useDeviceList(queryParams)

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const colCount = 7

  const hasActiveFilter = Boolean(searchText.trim()) || lifecycle !== 'all'

  return (
    <PageShell
      title="开站调测"
      description={`共 ${total} 台设备 · 按生命周期跟踪开站调测进度`}
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
            value={lifecycle}
            onValueChange={(v) => {
              setLifecycle(v as LifecycleFilter)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-40">
              <SelectValue placeholder="调测阶段" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部阶段</SelectItem>
              <SelectItem value="discovered">设备发现</SelectItem>
              <SelectItem value="registered">注册完成</SelectItem>
              <SelectItem value="provisioning">配置下发</SelectItem>
              <SelectItem value="commissioned">调测完成</SelectItem>
              <SelectItem value="maintenance">运维中</SelectItem>
            </SelectContent>
          </Select>
          {hasActiveFilter && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setSearchText('')
                setLifecycle('all')
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
              <TableHead>站点编码 (SN)</TableHead>
              <TableHead>站点名称</TableHead>
              <TableHead>制式</TableHead>
              <TableHead>调测阶段</TableHead>
              <TableHead>进度</TableHead>
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
                {hasActiveFilter ? '没有匹配的设备' : '暂无调测任务'}
              </EmptyRow>
            ) : (
              rows.map((d: Device) => {
                const meta =
                  LIFECYCLE_META[d.lifecycleState] ?? LIFECYCLE_META.discovered
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
                      {d.networkType ? (
                        <Badge variant={TYPE_VARIANT[d.networkType] ?? 'muted'}>
                          {d.networkType}
                        </Badge>
                      ) : (
                        <span className="text-xs text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <Badge variant={meta.variant}>{meta.step}</Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <div className="h-1.5 w-20 overflow-hidden rounded-full bg-muted">
                          <div
                            className={cn(
                              'h-full rounded-full',
                              meta.progress >= 100
                                ? 'bg-emerald-500'
                                : 'bg-primary'
                            )}
                            style={{ width: `${meta.progress}%` }}
                          />
                        </div>
                        <span className="text-[11px] tabular-nums text-muted-foreground">
                          {meta.progress}%
                        </span>
                      </div>
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
