import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Download, RefreshCcw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
  formatBytes,
  formatTime,
} from '@/components/layout/PageShell'

import { useStationLogList, useDownloadStationLog } from '@core/hooks/api/useStationLog'
import type { StationLogFile } from '@core/services/api/stationLogApi'

import { LogNavTabs, PAGE_SIZE, SearchInput, Stat } from './_shared'

// ===========================================================================
// 基站日志（对齐 v1 log/device）— 设备采集的运行 / 故障日志文件列表，可下载、看详情
// 真实端点：stationLogApi.list（GET /station-logs，运行 FileType6 / 故障 FileType8）
// ===========================================================================

type LogTypeFilter = 'all' | 'running' | 'fault'

const LOG_TYPE_LABEL: Record<StationLogFile['logType'], string> = {
  running: '运行日志',
  fault: '故障日志',
}

const LOG_TYPE_VARIANT: Record<StationLogFile['logType'], 'default' | 'destructive'> = {
  running: 'default',
  fault: 'destructive',
}

export default function DeviceLog() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [logType, setLogType] = useState<LogTypeFilter>('all')
  const [deviceId, setDeviceId] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(logType !== 'all' ? { logType } : {}),
      ...(deviceId.trim() ? { deviceId: deviceId.trim() } : {}),
    }),
    [page, logType, deviceId]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useStationLogList(params)
  const download = useDownloadStationLog()
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const counts = useMemo(() => {
    let running = 0
    let fault = 0
    for (const r of rows) {
      if (r.logType === 'fault') fault += 1
      else running += 1
    }
    return { running, fault }
  }, [rows])

  const cols = ['设备 SN', '日志类型', '文件名', '大小', '故障原因', '采集时间', '']

  return (
    <PageShell
      title="基站日志"
      description="设备上报的运行 / 故障日志文件 · 可下载、查看详情"
      isFetching={isFetching}
      toolbar={<LogNavTabs />}
    >
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-3">
        <Stat label="本页总数" value={rows.length} />
        <Stat label="运行日志" value={counts.running} tone="emerald" />
        <Stat label="故障日志" value={counts.fault} tone="rose" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <SearchInput
          placeholder="设备 ID"
          value={deviceId}
          onChange={(v) => {
            setDeviceId(v)
            setPage(1)
          }}
        />
        <Select
          value={logType}
          onValueChange={(v) => {
            setLogType(v as LogTypeFilter)
            setPage(1)
          }}
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder="日志类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部类型</SelectItem>
            <SelectItem value="running">运行日志</SelectItem>
            <SelectItem value="fault">故障日志</SelectItem>
          </SelectContent>
        </Select>
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
              <EmptyRow colSpan={cols.length}>暂无基站日志</EmptyRow>
            ) : (
              rows.map((l) => (
                <TableRow key={l.id}>
                  <TableCell>
                    <button
                      type="button"
                      className="font-mono text-xs text-primary hover:underline"
                      onClick={() => navigate(`/logs/device/${l.id}`)}
                    >
                      {l.deviceSn || '—'}
                    </button>
                  </TableCell>
                  <TableCell>
                    <Badge variant={LOG_TYPE_VARIANT[l.logType]}>
                      {LOG_TYPE_LABEL[l.logType]}
                    </Badge>
                  </TableCell>
                  <TableCell className="max-w-[280px] truncate font-mono text-xs">
                    {l.fileName || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatBytes(l.fileSize)}
                  </TableCell>
                  <TableCell className="max-w-[200px] truncate text-xs text-muted-foreground">
                    {l.faultReason || '—'}
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                    {formatTime(l.collectedAt || l.createdAt)}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={download.isPending}
                        onClick={() => download.mutate(l.id)}
                      >
                        <Download className="size-4" /> 下载
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => navigate(`/logs/device/${l.id}`)}
                      >
                        详情
                      </Button>
                    </div>
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
