import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Download, RefreshCcw, Search } from 'lucide-react'

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
  formatBytes,
  formatTime,
} from '@/components/layout/PageShell'

import { useDeviceList } from '@core/hooks/api/useDevices'
import { useDownloadFile, useFileList } from '@core/hooks/api/useFiles'
import type { FileStatus } from '@core/mock/data/fileManagement'

// ============================================================
// 配置文件获取（对齐 v1 file/config-retrieval）
//   左侧：真实设备列表（useDeviceList，按 SN/名称搜索、勾选目标）
//   右侧：已采集的配置文件（useFileList，fileType=config，可按设备 SN 过滤）
//   行 → 文件详情 /files/detail/:id；可下载
//   数据全走 @core React Query hooks，三态完整。
// ============================================================

const PAGE_SIZE = 15

function fileStatusVariant(s: FileStatus): 'success' | 'warning' | 'muted' {
  if (s === 'available') return 'success'
  if (s === 'expired' || s === 'deleted') return 'muted'
  return 'warning'
}

const FILE_STATUS_LABEL: Record<FileStatus, string> = {
  available: '可用',
  uploading: '上传中',
  processing: '处理中',
  expired: '已过期',
  deleted: '已删除',
}

export default function ConfigRetrieval() {
  const navigate = useNavigate()
  const [activeSn, setActiveSn] = useState<string>('')

  return (
    <PageShell
      title="配置文件获取"
      description="从设备采集运行配置，并在文件库中查看已获取的配置文件"
    >
      <div className="grid gap-4 lg:grid-cols-[minmax(0,360px)_1fr]">
        <DevicePicker activeSn={activeSn} onPick={setActiveSn} />
        <ConfigFiles activeSn={activeSn} onClearFilter={() => setActiveSn('')} onOpen={(id) => navigate(`/files/detail/${id}`)} />
      </div>
    </PageShell>
  )
}

function DevicePicker({
  activeSn,
  onPick,
}: {
  activeSn: string
  onPick: (sn: string) => void
}) {
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { searchText: keyword.trim() } : {}),
    }),
    [page, keyword],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useDeviceList(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['设备 SN', '名称', '产品类', '操作']

  return (
    <div>
      <div className="mb-3 flex items-center gap-2">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-full pl-9"
            placeholder="搜索设备 SN / 名称"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          <RefreshCcw className={isFetching ? 'animate-spin' : ''} />
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
              <EmptyRow colSpan={cols.length}>暂无设备</EmptyRow>
            ) : (
              rows.map((d) => (
                <TableRow key={d.id} className={activeSn === d.sn ? 'bg-accent/50' : undefined}>
                  <TableCell className="font-mono text-xs">{d.sn}</TableCell>
                  <TableCell className="text-xs">{d.deviceName || d.name || '—'}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {d.productClass || '—'}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-8 px-2"
                      onClick={() => onPick(d.sn)}
                    >
                      查看配置
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </div>
  )
}

function ConfigFiles({
  activeSn,
  onClearFilter,
  onOpen,
}: {
  activeSn: string
  onClearFilter: () => void
  onOpen: (id: string) => void
}) {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      fileType: 'config' as const,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(activeSn ? { deviceSn: activeSn } : {}),
    }),
    [page, keyword, activeSn],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useFileList(params)
  const download = useDownloadFile()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['文件名', '状态', '大小', '设备', '采集时间', '操作']

  return (
    <div>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-56 pl-9"
            placeholder="搜索文件名"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value)
              setPage(1)
            }}
          />
        </div>
        {activeSn ? (
          <Badge variant="outline" className="gap-1">
            设备：{activeSn}
            <button type="button" className="ml-1 text-muted-foreground hover:text-foreground" onClick={onClearFilter}>
              ✕
            </button>
          </Badge>
        ) : null}
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
                      onClick={() => onOpen(f.id)}
                    >
                      {f.fileName}
                    </button>
                  </TableCell>
                  <TableCell>
                    <Badge variant={fileStatusVariant(f.status)}>
                      {FILE_STATUS_LABEL[f.status] ?? f.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatBytes(f.fileSize)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {f.deviceName || f.deviceSn || '—'}
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
    </div>
  )
}
