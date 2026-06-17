import { useCallback, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Search, Zap } from 'lucide-react'

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
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'

import { useTriggerAlarmSync } from '@core/hooks/api/useAlarms'
import { useDeviceList } from '@core/hooks/api/useDevices'

// 同步任务（前端记录每次手动触发的结果；触发本身是真实接口）
interface SyncRecord {
  key: string
  deviceSn: string
  deviceName: string
  startedAt: string
  finishedAt: string | null
  status: 'running' | 'success' | 'failed'
}

const STATUS_META: Record<
  SyncRecord['status'],
  { label: string; variant: 'default' | 'destructive' | 'warning' }
> = {
  running: { label: '同步中', variant: 'warning' },
  success: { label: '成功', variant: 'default' },
  failed: { label: '失败', variant: 'destructive' },
}

export default function AlarmSync() {
  const navigate = useNavigate()
  const triggerSync = useTriggerAlarmSync()

  const [keyword, setKeyword] = useState('')
  const [records, setRecords] = useState<SyncRecord[]>([])

  const params = useMemo(() => {
    const p: { page: number; pageSize: number; keyword?: string } = {
      page: 1,
      pageSize: 20,
    }
    if (keyword.trim()) p.keyword = keyword.trim()
    return p
  }, [keyword])

  const { data, isLoading, isError, error, isFetching, refetch } =
    useDeviceList(params)
  const devices = data?.items ?? []

  const handleSync = useCallback(
    (sn: string, name: string) => {
      const key = `${sn}-${Date.now()}`
      setRecords((prev) => [
        {
          key,
          deviceSn: sn,
          deviceName: name,
          startedAt: new Date().toISOString(),
          finishedAt: null,
          status: 'running',
        },
        ...prev,
      ])
      triggerSync.mutate(sn, {
        onSuccess: () => {
          setRecords((prev) =>
            prev.map((r) =>
              r.key === key
                ? { ...r, status: 'success', finishedAt: new Date().toISOString() }
                : r
            )
          )
        },
        onError: () => {
          setRecords((prev) =>
            prev.map((r) =>
              r.key === key
                ? { ...r, status: 'failed', finishedAt: new Date().toISOString() }
                : r
            )
          )
        },
      })
    },
    [triggerSync]
  )

  const deviceCols = ['设备 SN', '设备名称', '厂商', '制式', '状态', '操作']
  const recordCols = ['设备 SN', '设备名称', '开始时间', '结束时间', '状态']

  return (
    <PageShell
      title="告警同步"
      description="向设备发起告警重新同步（AlarmList 主动拉取）"
      isFetching={isFetching}
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-64 pl-9"
              placeholder="设备 SN / 名称"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={() => refetch()}
          >
            <RefreshCcw /> 刷新
          </Button>
        </>
      }
    >
      <div className="mb-2 text-sm font-medium">设备列表</div>
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {deviceCols.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={deviceCols.length} />
            ) : isError ? (
              <ErrorRow colSpan={deviceCols.length} error={error} />
            ) : devices.length === 0 ? (
              <EmptyRow colSpan={deviceCols.length}>暂无设备</EmptyRow>
            ) : (
              devices.map((d) => (
                <TableRow key={d.id}>
                  <TableCell>
                    <button
                      type="button"
                      className="font-mono text-xs hover:underline"
                      onClick={() => navigate(`/device/detail/${d.sn}`)}
                    >
                      {d.sn}
                    </button>
                  </TableCell>
                  <TableCell className="text-xs">{d.name || '—'}</TableCell>
                  <TableCell className="text-xs">{d.vendor || '—'}</TableCell>
                  <TableCell className="text-xs uppercase">
                    {d.networkType || '—'}
                  </TableCell>
                  <TableCell>
                    <Badge variant={d.isOnline ? 'default' : 'muted'}>
                      {d.isOnline ? '在线' : '离线'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 px-2 text-xs"
                      disabled={triggerSync.isPending}
                      onClick={() => handleSync(d.sn, d.name)}
                    >
                      <Zap className="size-3.5" /> 同步告警
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <div className="mb-2 mt-6 text-sm font-medium">同步记录</div>
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {recordCols.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {records.length === 0 ? (
              <EmptyRow colSpan={recordCols.length}>
                暂无同步记录，点击上方“同步告警”发起
              </EmptyRow>
            ) : (
              records.map((r) => {
                const meta = STATUS_META[r.status]
                return (
                  <TableRow key={r.key}>
                    <TableCell className="font-mono text-xs">{r.deviceSn}</TableCell>
                    <TableCell className="text-xs">{r.deviceName || '—'}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(r.startedAt)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {r.finishedAt ? formatTime(r.finishedAt) : '—'}
                    </TableCell>
                    <TableCell>
                      <Badge variant={meta.variant}>{meta.label}</Badge>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>
    </PageShell>
  )
}
