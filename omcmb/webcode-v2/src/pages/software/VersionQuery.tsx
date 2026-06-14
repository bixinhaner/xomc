import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Boxes, CheckCircle2, Download, Eye, Package, RefreshCcw, Star } from 'lucide-react'

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

import { useSoftwareVersions } from '@core/hooks/api/useSoftware'
import { useProductClasses } from '@core/hooks/api/useDevices'
import type { SoftwareVersion, VersionStatus } from '@core/mock/data/software'

import { FILE_TYPE_LABEL, VERSION_STATUS } from './_shared'
import { Stat } from './_components'

// ===========================================================================
// 版本查询 — 对照 v1 webcode/src/pages/software/VersionQuery
// 设备固件版本库的只读查询，支持下载与下钻到版本详情 /software/version/:id
// ===========================================================================

export default function VersionQuery() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState<'' | VersionStatus>('')
  const [deviceType, setDeviceType] = useState('')

  const { data: productClasses } = useProductClasses()

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(status ? { status } : {}),
      ...(deviceType ? { deviceType } : {}),
    }),
    [page, pageSize, status, deviceType]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useSoftwareVersions(params)

  const allRows = data?.items ?? []
  const kw = search.trim().toLowerCase()
  const rows = kw
    ? allRows.filter(
        (v) =>
          v.versionCode.toLowerCase().includes(kw) ||
          v.versionName.toLowerCase().includes(kw) ||
          (v.deviceType ?? '').toLowerCase().includes(kw)
      )
    : allRows
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const currentCount = allRows.filter((v) => v.status === 'current').length
  const recommendCount = allRows.filter((v) => v.recommend).length
  const totalSize = allRows.reduce((acc, v) => acc + (v.fileSize ?? 0), 0)

  const cols = ['版本号', '名称', '设备型号', '厂商', '类型', '状态', '推荐', '文件大小', '发布时间', '操作']

  return (
    <PageShell title="版本查询" description="设备固件版本库的只读查询，支持下载与版本详情下钻">
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="版本总数" value={total} icon={Package} />
        <Stat label="当前版本" value={currentCount} icon={CheckCircle2} tone="emerald" />
        <Stat label="推荐版本" value={recommendCount} icon={Star} tone="amber" />
        <Stat label="本页占用" value={formatBytes(totalSize)} icon={Boxes} tone="muted" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <Input
          className="w-72"
          placeholder="搜索版本号 / 名称 / 型号"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <Select
          value={deviceType || 'all'}
          onValueChange={(v) => {
            setDeviceType(v === 'all' ? '' : v)
            setPage(1)
          }}
        >
          <SelectTrigger className="w-44">
            <SelectValue placeholder="设备型号" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部型号</SelectItem>
            {(productClasses ?? []).map((pc) => (
              <SelectItem key={pc} value={pc}>
                {pc}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={status || 'all'}
          onValueChange={(v) => {
            setStatus(v === 'all' ? '' : (v as VersionStatus))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="current">当前</SelectItem>
            <SelectItem value="beta">测试</SelectItem>
            <SelectItem value="deprecated">已废弃</SelectItem>
            <SelectItem value="archived">已归档</SelectItem>
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
              <EmptyRow colSpan={cols.length}>暂无版本</EmptyRow>
            ) : (
              rows.map((v: SoftwareVersion) => {
                const st = VERSION_STATUS[v.status]
                return (
                  <TableRow key={v.id}>
                    <TableCell>
                      <button
                        type="button"
                        className="text-left font-mono text-xs text-primary hover:underline"
                        onClick={() => navigate(`/software/version/${v.id}`)}
                        title="查看版本详情"
                      >
                        {v.versionCode}
                      </button>
                    </TableCell>
                    <TableCell className="max-w-[220px] truncate" title={v.versionName}>
                      {v.versionName || '—'}
                    </TableCell>
                    <TableCell>{v.deviceType || '—'}</TableCell>
                    <TableCell>{v.vendor || v.manufacturer || '—'}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {v.fileType != null ? FILE_TYPE_LABEL[v.fileType] ?? '—' : '—'}
                    </TableCell>
                    <TableCell>
                      <Badge variant={st.variant}>{st.label}</Badge>
                    </TableCell>
                    <TableCell>
                      {v.recommend ? (
                        <Badge variant="warning">
                          <Star className="mr-1 size-3" /> 推荐
                        </Badge>
                      ) : (
                        <span className="text-xs text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatBytes(v.fileSize)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(v.releaseDate)}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={!v.downloadUrl}
                          onClick={() => {
                            if (v.downloadUrl) window.open(v.downloadUrl, '_blank')
                          }}
                        >
                          <Download className="size-3.5" /> 下载
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => navigate(`/software/version/${v.id}`)}
                        >
                          <Eye className="size-3.5" /> 详情
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={setPage} />
    </PageShell>
  )
}
