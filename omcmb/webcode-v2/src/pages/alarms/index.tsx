import { useMemo, useState } from 'react'
import { Search, RefreshCcw } from 'lucide-react'

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

import { useAlarmList } from '@core/hooks/api/useAlarms'
import type { AlarmSeverity } from '@core/types/common'

const SEVERITY_LABEL: Record<AlarmSeverity, string> = {
  critical: '紧急',
  major: '重要',
  minor: '次要',
  warning: '警告',
}
const SEVERITY_VARIANT: Record<
  AlarmSeverity,
  'destructive' | 'warning' | 'default' | 'muted'
> = {
  critical: 'destructive',
  major: 'destructive',
  minor: 'warning',
  warning: 'warning',
}

export function AlarmsPage() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [severity, setSeverity] = useState<AlarmSeverity | ''>('')
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      isActive: true,
      ...(severity ? { severity } : {}),
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, pageSize, severity, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useAlarmList(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['严重度', '告警名称', '设备', '故障时间', '状态']

  return (
    <PageShell
      title="告警中心"
      description="实时告警列表 · 每 30 秒自动刷新"
      isFetching={isFetching}
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-72 pl-9"
              placeholder="告警名称 / 设备 SN"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <Select
            value={severity || 'all'}
            onValueChange={(v) => {
              setSeverity(v === 'all' ? '' : (v as AlarmSeverity))
              setPage(1)
            }}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="严重度" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部严重度</SelectItem>
              <SelectItem value="critical">紧急</SelectItem>
              <SelectItem value="major">重要</SelectItem>
              <SelectItem value="minor">次要</SelectItem>
              <SelectItem value="warning">警告</SelectItem>
            </SelectContent>
          </Select>
          <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
          </Button>
        </>
      }
    >
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
              <EmptyRow colSpan={cols.length}>暂无活动告警</EmptyRow>
            ) : (
              rows.map((a) => (
                <TableRow key={a.id}>
                  <TableCell>
                    <Badge variant={SEVERITY_VARIANT[a.severity]}>
                      {SEVERITY_LABEL[a.severity]}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="font-medium">{a.alarmName}</div>
                    <div className="text-xs text-muted-foreground">{a.alarmCode}</div>
                  </TableCell>
                  <TableCell>
                    <div className="font-medium">{a.deviceName || '—'}</div>
                    <div className="font-mono text-xs text-muted-foreground">
                      {a.deviceSn}
                    </div>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(a.eventTime)}
                  </TableCell>
                  <TableCell>
                    <Badge variant={a.dealState === '0' ? 'warning' : 'success'}>
                      {a.dealState === '0'
                        ? '未确认'
                        : a.dealState === '1'
                          ? '已确认'
                          : a.dealState === '2'
                            ? '已清除'
                            : '已处理'}
                    </Badge>
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
        pageSize={pageSize}
        onChange={setPage}
      />
    </PageShell>
  )
}
