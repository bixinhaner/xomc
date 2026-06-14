import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw } from 'lucide-react'

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
  formatTime,
} from '@/components/layout/PageShell'

import { useAbnormalRebootList } from '@core/hooks/api/useDeviceAbnormalReboot'
import type { RecordStatus } from '@core/services/api/deviceAbnormalRebootApi'

import { formatRuntime } from './_format'
import { LogNavTabs, PAGE_SIZE, SearchInput, Stat } from './_shared'

// ===========================================================================
// 异常重启（对齐 v1 log/exception → device/abnormal-reboot）
// 真实端点：deviceAbnormalRebootApi.list（GET /device-abnormal-reboots，station_fault_logs）
// ===========================================================================

type DeviceTypeFilter = 'all' | 'eNB' | 'gNB'
type StatusFilter = 'all' | RecordStatus

const STATUS_LABEL: Record<RecordStatus, string> = {
  detected: '已检测',
  file_received: '文件已收',
  collection_failed: '采集失败',
}

const STATUS_VARIANT: Record<RecordStatus, 'default' | 'success' | 'destructive'> = {
  detected: 'default',
  file_received: 'success',
  collection_failed: 'destructive',
}

export default function ExceptionLog() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [deviceSn, setDeviceSn] = useState('')
  const [deviceType, setDeviceType] = useState<DeviceTypeFilter>('all')
  const [recordStatus, setRecordStatus] = useState<StatusFilter>('all')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
      ...(deviceType !== 'all' ? { deviceType } : {}),
      ...(recordStatus !== 'all' ? { recordStatus } : {}),
    }),
    [page, deviceSn, deviceType, recordStatus]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useAbnormalRebootList(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const counts = useMemo(() => {
    let received = 0
    let failed = 0
    for (const r of rows) {
      if (r.recordStatus === 'file_received') received += 1
      else if (r.recordStatus === 'collection_failed') failed += 1
    }
    return { received, failed }
  }, [rows])

  const cols = ['设备', '制式', '原因', '重启前运行', '采集状态', '发生时间', '']

  return (
    <PageShell
      title="异常重启"
      description="设备异常重启记录与故障日志采集状态（station_fault_logs）"
      isFetching={isFetching}
      toolbar={<LogNavTabs />}
    >
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-3">
        <Stat label="本页总数" value={rows.length} />
        <Stat label="文件已收" value={counts.received} tone="emerald" />
        <Stat label="采集失败" value={counts.failed} tone="rose" />
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
        <Select
          value={deviceType}
          onValueChange={(v) => {
            setDeviceType(v as DeviceTypeFilter)
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="制式" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部制式</SelectItem>
            <SelectItem value="eNB">eNB (LTE)</SelectItem>
            <SelectItem value="gNB">gNB (NR)</SelectItem>
          </SelectContent>
        </Select>
        <Select
          value={recordStatus}
          onValueChange={(v) => {
            setRecordStatus(v as StatusFilter)
            setPage(1)
          }}
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder="采集状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="detected">已检测</SelectItem>
            <SelectItem value="file_received">文件已收</SelectItem>
            <SelectItem value="collection_failed">采集失败</SelectItem>
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
              <EmptyRow colSpan={cols.length}>暂无异常重启记录</EmptyRow>
            ) : (
              rows.map((r) => (
                <TableRow key={r.id}>
                  <TableCell>
                    <button
                      type="button"
                      className="text-left hover:underline"
                      onClick={() => navigate(`/logs/exception/${r.id}`)}
                    >
                      <div className="font-medium text-primary">{r.deviceName || r.deviceSn}</div>
                      <div className="font-mono text-xs text-muted-foreground">{r.deviceSn}</div>
                    </button>
                  </TableCell>
                  <TableCell>
                    <span className="text-xs text-muted-foreground">{r.deviceType || '—'}</span>
                  </TableCell>
                  <TableCell className="max-w-[240px] truncate text-xs">
                    {r.haltMainReason || '—'}
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                    {formatRuntime(r.runtimeBeforeReboot)}
                  </TableCell>
                  <TableCell>
                    <Badge variant={STATUS_VARIANT[r.recordStatus] ?? 'muted'}>
                      {STATUS_LABEL[r.recordStatus] ?? r.recordStatus}
                    </Badge>
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                    {formatTime(r.collectedAt || r.createdAt)}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => navigate(`/logs/exception/${r.id}`)}
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
