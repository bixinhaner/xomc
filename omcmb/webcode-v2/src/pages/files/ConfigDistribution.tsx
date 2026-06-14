import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { CheckCircle2, RefreshCcw, Search, Send } from 'lucide-react'

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
import { useDistributeFile, useFileList } from '@core/hooks/api/useFiles'

// ============================================================
// 配置文件下发（对齐 v1 file/config-distribution）
//   左：选一个配置文件（useFileList，fileType=config）作为下发源
//   右：勾选目标设备（useDeviceList）；点"下发"调用 useDistributeFile
//   文件名 → 文件详情 /files/detail/:id
//   数据全走 @core React Query hooks，三态完整。
// ============================================================

const PAGE_SIZE = 15

export default function ConfigDistribution() {
  const navigate = useNavigate()
  const [selectedFileId, setSelectedFileId] = useState<string>('')
  const [selectedFileName, setSelectedFileName] = useState<string>('')
  const [selectedSns, setSelectedSns] = useState<Set<string>>(new Set())

  const distribute = useDistributeFile()

  const handleDistribute = () => {
    const sns = Array.from(selectedSns)
    if (!selectedFileId || sns.length === 0) return
    if (!window.confirm(`确认将「${selectedFileName}」下发到选中的 ${sns.length} 台设备？`)) return
    distribute.mutate({ fileId: selectedFileId, deviceSns: sns })
  }

  return (
    <PageShell
      title="配置文件下发"
      description="选择配置文件源，勾选目标设备并下发"
    >
      <div className="mb-4 flex flex-wrap items-center gap-3 rounded-lg border bg-card px-4 py-3 text-sm">
        <span className="text-muted-foreground">下发源：</span>
        {selectedFileId ? (
          <Badge variant="success" className="gap-1">
            <CheckCircle2 className="size-3.5" />
            {selectedFileName}
          </Badge>
        ) : (
          <span className="text-muted-foreground">未选择</span>
        )}
        <span className="ml-4 text-muted-foreground">目标设备：</span>
        <Badge variant="outline">{selectedSns.size} 台</Badge>
        <Button
          size="sm"
          className="ml-auto"
          disabled={!selectedFileId || selectedSns.size === 0 || distribute.isPending}
          onClick={handleDistribute}
        >
          <Send /> 下发
        </Button>
        {distribute.isSuccess ? (
          <span className="text-xs text-emerald-600">已提交下发任务</span>
        ) : null}
        {distribute.isError ? (
          <span className="text-xs text-destructive">下发失败，请重试</span>
        ) : null}
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <SourceFiles
          selectedFileId={selectedFileId}
          onSelect={(id, name) => {
            setSelectedFileId(id)
            setSelectedFileName(name)
          }}
          onOpen={(id) => navigate(`/files/detail/${id}`)}
        />
        <TargetDevices selected={selectedSns} onChange={setSelectedSns} />
      </div>
    </PageShell>
  )
}

function SourceFiles({
  selectedFileId,
  onSelect,
  onOpen,
}: {
  selectedFileId: string
  onSelect: (id: string, name: string) => void
  onOpen: (id: string) => void
}) {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      fileType: 'config' as const,
      status: 'available' as const,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, keyword],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useFileList(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['选择', '文件名', '大小', '更新时间']

  return (
    <div>
      <div className="mb-3 flex items-center gap-2">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-full pl-9"
            placeholder="搜索配置文件名"
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
              <EmptyRow colSpan={cols.length}>暂无可用配置文件</EmptyRow>
            ) : (
              rows.map((f) => (
                <TableRow key={f.id} className={selectedFileId === f.id ? 'bg-accent/50' : undefined}>
                  <TableCell className="w-12">
                    <input
                      type="radio"
                      name="source-file"
                      aria-label={`选择 ${f.fileName}`}
                      checked={selectedFileId === f.id}
                      onChange={() => onSelect(f.id, f.fileName)}
                      className="size-4 cursor-pointer"
                    />
                  </TableCell>
                  <TableCell>
                    <button
                      type="button"
                      className="font-medium text-primary hover:underline"
                      onClick={() => onOpen(f.id)}
                    >
                      {f.fileName}
                    </button>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatBytes(f.fileSize)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(f.uploadTime)}
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

function TargetDevices({
  selected,
  onChange,
}: {
  selected: Set<string>
  onChange: (next: Set<string>) => void
}) {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')

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

  const toggle = (sn: string) => {
    const next = new Set(selected)
    if (next.has(sn)) next.delete(sn)
    else next.add(sn)
    onChange(next)
  }
  const toggleAll = () => {
    if (rows.length > 0 && rows.every((d) => selected.has(d.sn))) {
      const next = new Set(selected)
      rows.forEach((d) => next.delete(d.sn))
      onChange(next)
    } else {
      const next = new Set(selected)
      rows.forEach((d) => next.add(d.sn))
      onChange(next)
    }
  }

  const allChecked = rows.length > 0 && rows.every((d) => selected.has(d.sn))
  const cols = ['', '设备 SN', '名称', '在线']

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
              <EmptyRow colSpan={cols.length}>暂无设备</EmptyRow>
            ) : (
              rows.map((d) => (
                <TableRow key={d.id}>
                  <TableCell className="w-10">
                    <input
                      type="checkbox"
                      aria-label={`选择 ${d.sn}`}
                      checked={selected.has(d.sn)}
                      onChange={() => toggle(d.sn)}
                      className="size-4 cursor-pointer"
                    />
                  </TableCell>
                  <TableCell className="font-mono text-xs">{d.sn}</TableCell>
                  <TableCell className="text-xs">{d.deviceName || d.name || '—'}</TableCell>
                  <TableCell>
                    {d.isOnline ? (
                      <Badge variant="success">在线</Badge>
                    ) : (
                      <Badge variant="muted">离线</Badge>
                    )}
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
