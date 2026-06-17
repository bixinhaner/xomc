import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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

import { useEventLogList } from '@core/hooks/api/useEventLog'
import type { EventLevel } from '@core/services/api/eventLogApi'

import { LogNavTabs, PAGE_SIZE, SearchInput, Stat } from './_shared'

// ===========================================================================
// 设备事件（对齐 v1 log/event → device 重启记录的事件流）
// 真实端点：eventLogApi.list（GET /event-logs，event_logs 表，含设备启动/重启等事件）
// ===========================================================================

const LEVEL_LABEL: Record<EventLevel, string> = {
  info: '信息',
  warning: '警告',
  error: '错误',
  success: '成功',
}

const LEVEL_VARIANT: Record<EventLevel, 'default' | 'warning' | 'destructive' | 'success'> = {
  info: 'default',
  warning: 'warning',
  error: 'destructive',
  success: 'success',
}

export default function EventLog() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [deviceSn, setDeviceSn] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
    }),
    [page, deviceSn]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useEventLogList(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const counts = useMemo(() => {
    let err = 0
    let warn = 0
    for (const r of rows) {
      if (r.eventLevel === 'error') err += 1
      else if (r.eventLevel === 'warning') warn += 1
    }
    return { err, warn }
  }, [rows])

  const cols = ['设备', '制式', '事件类型', '级别', '原因', '软件版本', '发生时间', '']

  return (
    <PageShell
      title="设备事件"
      description="设备上报的事件日志（启动 / 重启等，event_logs）"
      isFetching={isFetching}
      toolbar={<LogNavTabs />}
    >
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-3">
        <Stat label="本页总数" value={rows.length} />
        <Stat label="错误" value={counts.err} tone="rose" />
        <Stat label="警告" value={counts.warn} tone="amber" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <SearchInput
          placeholder="设备 SN"
          value={deviceSn}
          onChange={(v) => {
            setDeviceSn(v)
            setPage(1)
          }}
        />
        <div className="ml-auto">
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw className="size-4" /> 刷新
          </Button>
        </div>
      </div>

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c, i) => (
                <TableHead key={c || `c-${i}`}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无设备事件</EmptyRow>
            ) : (
              rows.map((r) => (
                <TableRow key={r.id}>
                  <TableCell>
                    <button
                      type="button"
                      className="text-left hover:underline"
                      onClick={() => navigate(`/log/event/${r.id}`)}
                    >
                      <div className="font-medium text-primary">{r.deviceName || r.deviceSn}</div>
                      <div className="font-mono text-xs text-muted-foreground">{r.deviceSn}</div>
                    </button>
                  </TableCell>
                  <TableCell>
                    <span className="text-xs text-muted-foreground">{r.deviceType || '—'}</span>
                  </TableCell>
                  <TableCell className="font-mono text-xs">{r.eventType || '—'}</TableCell>
                  <TableCell>
                    <Badge variant={LEVEL_VARIANT[r.eventLevel] ?? 'default'}>
                      {LEVEL_LABEL[r.eventLevel] ?? r.eventLevel}
                    </Badge>
                  </TableCell>
                  <TableCell className="max-w-[240px] truncate text-xs">
                    {r.eventReason || '—'}
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {r.softwareVersion || '—'}
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                    {formatTime(r.occurredAt || r.createdAt)}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => navigate(`/log/event/${r.id}`)}
                    >
                      详情
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </PageShell>
  )
}
