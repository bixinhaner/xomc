import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Download, RefreshCcw, Search, Trash2 } from 'lucide-react'

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
  formatBytes,
  formatTime,
} from '@/components/layout/PageShell'

import {
  useDeleteStationLog,
  useDownloadStationLog,
  useStationLogList,
} from '@core/hooks/api/useStationLog'

// ============================================================
// 日志获取（对齐 v1 file/log-retrieval）
//   全网基站日志文件（useStationLogList）：运行 / 故障日志
//   按设备 SN / 日志类型过滤；下载（预签名 URL）、删除
//   设备 SN → 设备文件页 /files/device-files（按设备查看）
//   数据全走 @core React Query hooks，三态完整。
// ============================================================

const PAGE_SIZE = 20

const LOG_TYPE_LABEL: Record<'running' | 'fault', string> = {
  running: '运行日志',
  fault: '故障日志',
}

export default function LogRetrieval() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [deviceSn, setDeviceSn] = useState('')
  const [logType, setLogType] = useState<'running' | 'fault' | ''>('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(deviceSn.trim() ? { deviceId: deviceSn.trim() } : {}),
      ...(logType ? { logType } : {}),
    }),
    [page, deviceSn, logType],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useStationLogList(params)
  const download = useDownloadStationLog()
  const remove = useDeleteStationLog()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['文件名', '设备 SN', '类型', '大小', '故障原因', '采集时间', '上传时间', '操作']

  return (
    <PageShell
      title="日志获取"
      description="全网基站运行/故障日志文件采集与下载"
    >
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-56 pl-9"
            placeholder="按设备 SN 过滤"
            value={deviceSn}
            onChange={(e) => {
              setDeviceSn(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <Select
          value={logType || 'all'}
          onValueChange={(v) => {
            setLogType(v === 'all' ? '' : (v as 'running' | 'fault'))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="日志类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部类型</SelectItem>
            <SelectItem value="running">运行日志</SelectItem>
            <SelectItem value="fault">故障日志</SelectItem>
          </SelectContent>
        </Select>
        <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
          <RefreshCcw className={isFetching ? 'animate-spin' : ''} /> 刷新
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
              <EmptyRow colSpan={cols.length}>暂无日志文件</EmptyRow>
            ) : (
              rows.map((f) => (
                <TableRow key={f.id}>
                  <TableCell className="font-medium">{f.fileName}</TableCell>
                  <TableCell>
                    <button
                      type="button"
                      className="font-mono text-xs text-primary hover:underline"
                      onClick={() => navigate('/file/device-files')}
                      title="在设备文件页查看该设备文件"
                    >
                      {f.deviceSn}
                    </button>
                  </TableCell>
                  <TableCell>
                    <Badge variant={f.logType === 'fault' ? 'warning' : 'default'}>
                      {LOG_TYPE_LABEL[f.logType] ?? f.logType}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatBytes(f.fileSize)}
                  </TableCell>
                  <TableCell className="max-w-[160px] truncate text-xs text-muted-foreground" title={f.faultReason}>
                    {f.faultReason || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(f.collectedAt)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(f.createdAt)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 px-2"
                        disabled={download.isPending}
                        onClick={() => download.mutate(f.id)}
                      >
                        <Download /> 下载
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 px-2 text-destructive hover:text-destructive"
                        disabled={remove.isPending}
                        onClick={() => {
                          if (window.confirm(`确认删除日志「${f.fileName}」？`)) {
                            remove.mutate(f.id)
                          }
                        }}
                      >
                        <Trash2 /> 删除
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
