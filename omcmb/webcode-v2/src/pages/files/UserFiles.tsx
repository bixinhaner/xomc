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
  useDeleteFiles,
  useDownloadFile,
  useFileList,
} from '@core/hooks/api/useFiles'
import type { FileStatus, FileType } from '@core/mock/data/fileManagement'

// ============================================================
// 用户文件（对齐 v1 file/user-files）
//   用户上传/管理的文件库（useFileList），支持类型/状态/关键字筛选
//   多选批量删除 + 单行下载/删除；文件名 → 文件详情 /files/detail/:id
//   数据全走 @core React Query hooks，三态完整。
// ============================================================

const PAGE_SIZE = 20

const FILE_TYPE_OPTIONS: { value: FileType; label: string }[] = [
  { value: 'config', label: '配置' },
  { value: 'log', label: '日志' },
  { value: 'firmware', label: '固件' },
  { value: 'backup', label: '备份' },
  { value: 'report', label: '报表' },
  { value: 'certificate', label: '证书' },
]
const FILE_TYPE_LABEL: Record<string, string> = Object.fromEntries(
  FILE_TYPE_OPTIONS.map((o) => [o.value, o.label]),
)

const FILE_STATUS_OPTIONS: { value: FileStatus; label: string }[] = [
  { value: 'available', label: '可用' },
  { value: 'uploading', label: '上传中' },
  { value: 'processing', label: '处理中' },
  { value: 'expired', label: '已过期' },
  { value: 'deleted', label: '已删除' },
]
const FILE_STATUS_LABEL: Record<string, string> = Object.fromEntries(
  FILE_STATUS_OPTIONS.map((o) => [o.value, o.label]),
)

function fileStatusVariant(s: FileStatus): 'success' | 'warning' | 'muted' {
  if (s === 'available') return 'success'
  if (s === 'expired' || s === 'deleted') return 'muted'
  return 'warning'
}

export default function UserFiles() {
  const navigate = useNavigate()

  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [fileType, setFileType] = useState<FileType | ''>('')
  const [status, setStatus] = useState<FileStatus | ''>('')
  const [selected, setSelected] = useState<Set<string>>(new Set())

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(fileType ? { fileType } : {}),
      ...(status ? { status } : {}),
    }),
    [page, keyword, fileType, status],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useFileList(params)
  const download = useDownloadFile()
  const remove = useDeleteFiles()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const toggle = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }
  const toggleAll = () => {
    setSelected((prev) => {
      if (rows.length > 0 && rows.every((f) => prev.has(f.id))) return new Set()
      return new Set(rows.map((f) => f.id))
    })
  }
  const selectedList = Array.from(selected)
  const allChecked = rows.length > 0 && rows.every((f) => selected.has(f.id))

  const handleBatchDelete = () => {
    if (selectedList.length === 0) return
    if (!window.confirm(`确认删除选中的 ${selectedList.length} 个文件？`)) return
    remove.mutate(selectedList, { onSuccess: () => setSelected(new Set()) })
  }

  const cols = ['', '文件名', '类型', '状态', '大小', '设备', '上传时间', '上传者', '操作']

  return (
    <PageShell
      title="用户文件"
      description="用户上传与管理的文件库（配置 / 日志 / 固件 / 备份 / 报表 / 证书）"
    >
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-64 pl-9"
            placeholder="搜索文件名"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <Select
          value={fileType || 'all'}
          onValueChange={(v) => {
            setFileType(v === 'all' ? '' : (v as FileType))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="文件类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部类型</SelectItem>
            {FILE_TYPE_OPTIONS.map((o) => (
              <SelectItem key={o.value} value={o.value}>
                {o.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={status || 'all'}
          onValueChange={(v) => {
            setStatus(v === 'all' ? '' : (v as FileStatus))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            {FILE_STATUS_OPTIONS.map((o) => (
              <SelectItem key={o.value} value={o.value}>
                {o.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button
          variant="outline"
          size="sm"
          disabled={selectedList.length === 0 || remove.isPending}
          onClick={handleBatchDelete}
        >
          <Trash2 className="text-destructive" /> 批量删除（{selectedList.length}）
        </Button>
        <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
          <RefreshCcw className={isFetching ? 'animate-spin' : ''} /> 刷新
        </Button>
      </div>

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c, idx) =>
                idx === 0 ? (
                  <TableHead key="sel" className="w-10">
                    <input
                      type="checkbox"
                      aria-label="本页全选"
                      checked={allChecked}
                      onChange={toggleAll}
                      className="size-4 cursor-pointer"
                    />
                  </TableHead>
                ) : (
                  <TableHead key={c}>{c}</TableHead>
                ),
              )}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无文件</EmptyRow>
            ) : (
              rows.map((f) => (
                <TableRow key={f.id}>
                  <TableCell className="w-10">
                    <input
                      type="checkbox"
                      aria-label={`选择 ${f.fileName}`}
                      checked={selected.has(f.id)}
                      onChange={() => toggle(f.id)}
                      className="size-4 cursor-pointer"
                    />
                  </TableCell>
                  <TableCell>
                    <button
                      type="button"
                      className="font-medium text-primary hover:underline"
                      onClick={() => navigate(`/file/config-retrieval`)}
                    >
                      {f.fileName}
                    </button>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{FILE_TYPE_LABEL[f.fileType] ?? f.fileType}</Badge>
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
                  <TableCell className="text-xs text-muted-foreground">{f.uploader}</TableCell>
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
                          if (window.confirm(`确认删除文件「${f.fileName}」？`)) {
                            remove.mutate([f.id])
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
