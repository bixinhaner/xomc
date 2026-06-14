import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Search } from 'lucide-react'

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
  formatTime,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useIndicatorList } from '@core/hooks/api/useIndicator'
import type { IndicatorListParams, PerfIndicator } from '@core/types/indicator'

// ============================================================
// KPI 标准库（指标定义库） — 对齐 v1 /performance/kpi-standard
//   - 设备类型（ENB/GSM/GNB）切换
//   - 关键字搜索 / 启用状态 / 指标类型筛选
//   - 行可进 详情：/performance/kpi-standard/detail/:deviceType/:indicatorId
// ============================================================

type DeviceTypeTab = 'ENB' | 'GSM' | 'GNB'

const DEVICE_TABS: { key: DeviceTypeTab; label: string }[] = [
  { key: 'ENB', label: 'LTE (ENB)' },
  { key: 'GSM', label: 'GSM' },
  { key: 'GNB', label: '5G NR (GNB)' },
]

const PAGE_SIZE = 20

type EnableFilter = 'all' | 'true' | 'false'
type TypeFilter = 'all' | 'kpi' | 'counter'

function indicatorTypeLabel(ind: PerfIndicator): { label: string; counter: boolean } {
  const counter = ind.isCounter || ind.indicatorType === 'counter'
  return { label: counter ? 'Counter' : 'KPI', counter }
}

export function KPIStandardReportPage() {
  const navigate = useNavigate()
  const [deviceType, setDeviceType] = useState<DeviceTypeTab>('ENB')
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [enable, setEnable] = useState<EnableFilter>('all')
  const [type, setType] = useState<TypeFilter>('all')

  const params = useMemo<IndicatorListParams>(
    () => ({
      deviceType,
      page,
      rows: PAGE_SIZE,
      ...(keyword.trim() ? { searchText: keyword.trim() } : {}),
      ...(enable !== 'all' ? { isEnable: enable } : {}),
      ...(type !== 'all' ? { indicatorType: type === 'counter' ? 'true' : 'false' } : {}),
    }),
    [deviceType, page, keyword, enable, type]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useIndicatorList(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['指标编号', '名称', '功能集', '类型', '单位', '统计类型', '启用', '更新时间']

  const resetPage = () => setPage(1)

  return (
    <PageShell
      title="KPI 标准库"
      description="指标定义库 · 按设备制式分类 · 内置/自定义指标与计数器"
      isFetching={isFetching}
      toolbar={
        <div className="flex flex-wrap items-center gap-1.5">
          {DEVICE_TABS.map((t) => (
            <Button
              key={t.key}
              size="sm"
              variant={t.key === deviceType ? 'default' : 'outline'}
              onClick={() => {
                setDeviceType(t.key)
                resetPage()
              }}
            >
              {t.label}
            </Button>
          ))}
        </div>
      }
    >
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-72 pl-9"
            placeholder="指标名称 / 编号"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value)
              resetPage()
            }}
          />
        </div>
        <Select
          value={type}
          onValueChange={(v) => {
            setType(v as TypeFilter)
            resetPage()
          }}
        >
          <SelectTrigger className="w-32">
            <SelectValue placeholder="类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部类型</SelectItem>
            <SelectItem value="kpi">KPI</SelectItem>
            <SelectItem value="counter">Counter</SelectItem>
          </SelectContent>
        </Select>
        <Select
          value={enable}
          onValueChange={(v) => {
            setEnable(v as EnableFilter)
            resetPage()
          }}
        >
          <SelectTrigger className="w-32">
            <SelectValue placeholder="启用状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="true">已启用</SelectItem>
            <SelectItem value="false">已停用</SelectItem>
          </SelectContent>
        </Select>
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
              <EmptyRow colSpan={cols.length}>暂无指标定义</EmptyRow>
            ) : (
              rows.map((ind) => {
                const ty = indicatorTypeLabel(ind)
                return (
                  <TableRow key={ind.kpiId}>
                    <TableCell className="font-mono text-xs">
                      <button
                        type="button"
                        className="text-primary hover:underline"
                        onClick={() =>
                          navigate(
                            `/performance/kpi-standard/detail/${deviceType}/${encodeURIComponent(ind.kpiId)}`
                          )
                        }
                      >
                        {ind.kpiId || '—'}
                      </button>
                    </TableCell>
                    <TableCell className="font-medium">
                      {ind.kpiNameZh || ind.kpiName || ind.kpiNameEn || '—'}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {ind.catagoryName || '—'}
                    </TableCell>
                    <TableCell>
                      <Badge variant={ty.counter ? 'outline' : 'default'}>{ty.label}</Badge>
                    </TableCell>
                    <TableCell className="text-xs">{ind.unit || '—'}</TableCell>
                    <TableCell className="text-xs">{ind.statisType || '—'}</TableCell>
                    <TableCell>
                      <Badge variant={ind.isEnable ? 'success' : 'muted'}>
                        {ind.isEnable ? '启用' : '停用'}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(ind.updateTime)}
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
      <div className={cn('mt-2 text-xs text-muted-foreground', isFetching && 'opacity-60')}>
        共 {total} 个指标
      </div>
    </PageShell>
  )
}

export default KPIStandardReportPage
