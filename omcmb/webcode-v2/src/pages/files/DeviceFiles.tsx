import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Download, FileText, RefreshCcw, ScrollText, Search, Trash2 } from 'lucide-react'

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
import { cn } from '@/lib/utils'

import { useDownloadFile, useFileList } from '@core/hooks/api/useFiles'
import {
  useDeleteStationLog,
  useDownloadStationLog,
  useStationLogList,
} from '@core/hooks/api/useStationLog'

// ============================================================
// 设备文件（对齐 v1 file/device-files）
//   两个子页：
//     配置文件 —— useFileList(fileType=config, deviceSn)，文件名 → /files/detail/:id
//     日志文件 —— useStationLogList（运行/故障日志），下载/删除
//   均按设备 SN 过滤；数据全走 @core React Query hooks，三态完整。
// ============================================================

const PAGE_SIZE = 20

type TabKey = 'config' | 'log'

export default function DeviceFiles() {
  const [tab, setTab] = useState<TabKey>('config')
  const [deviceSn, setDeviceSn] = useState('')

  return (
    <PageShell
      title="设备文件"
      description="按设备查看其配置文件与运行/故障日志文件"
    >
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="inline-flex rounded-lg border bg-card p-1">
          <TabButton active={tab === 'config'} onClick={() => setTab('config')} icon={<FileText className="size-4" />}>
            配置文件
          </TabButton>
          <TabButton active={tab === 'log'} onClick={() => setTab('log')} icon={<ScrollText className="size-4" />}>
            日志文件
          </TabButton>
        </div>
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-64 pl-9"
            placeholder="按设备 SN 过滤"
            value={deviceSn}
            onChange={(e) => setDeviceSn(e.target.value)}
          />
        </div>
      </div>

      {tab === 'config' ? <ConfigFilesTab deviceSn={deviceSn} /> : <LogFilesTab deviceSn={deviceSn} />}
    </PageShell>
  )
}

function TabButton({
  active,
  onClick,
  icon,
  children,
}: {
  active: boolean
  onClick: () => void
  icon: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
        active
          ? 'bg-primary text-primary-foreground'
          : 'text-muted-foreground hover:text-foreground',
      )}
    >
      {icon}
      {children}
    </button>
  )
}

function ConfigFilesTab({ deviceSn }: { deviceSn: string }) {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      fileType: 'config' as const,
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
    }),
    [page, deviceSn],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useFileList(params)
  const download = useDownloadFile()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['文件名', '设备', '大小', '上传时间', '操作']

  return (
    <>
      <div className="mb-3 flex items-center gap-2">
        <span className="text-sm text-muted-foreground">共 {total} 个配置文件</span>
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
              <EmptyRow colSpan={cols.length}>暂无配置文件</EmptyRow>
            ) : (
              rows.map((f) => (
                <TableRow key={f.id}>
                  <TableCell>
                    <button
                      type="button"
                      className="font-medium text-primary hover:underline"
                      onClick={() => navigate(`/files/detail/${f.id}`)}
                    >
                      {f.fileName}
                    </button>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {f.deviceName || f.deviceSn || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatBytes(f.fileSize)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(f.uploadTime)}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-8 px-2"
                      disabled={download.isPending}
                      onClick={() => download.mutate(f.id)}
                    >
                      <Download /> 下载
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>
      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </>
  )
}

const LOG_TYPE_LABEL: Record<'running' | 'fault', string> = {
  running: '运行日志',
  fault: '故障日志',
}

function LogFilesTab({ deviceSn }: { deviceSn: string }) {
  const [page, setPage] = useState(1)
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

  const cols = ['文件名', '设备 SN', '类型', '大小', '采集时间', '上传时间', '操作']

  return (
    <>
      <div className="mb-3 flex flex-wrap items-center gap-2">
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
        <span className="text-sm text-muted-foreground">共 {total} 个日志文件</span>
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
                  <TableCell className="font-mono text-xs">{f.deviceSn}</TableCell>
                  <TableCell>
                    <Badge variant={f.logType === 'fault' ? 'warning' : 'default'}>
                      {LOG_TYPE_LABEL[f.logType] ?? f.logType}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatBytes(f.fileSize)}
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
    </>
  )
}
