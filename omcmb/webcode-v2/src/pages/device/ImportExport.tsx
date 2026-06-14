import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Download, FileSpreadsheet, RefreshCcw, Search } from 'lucide-react'

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
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
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

import { useDeviceList } from '@core/hooks/api/useDevices'
import type { Device, DeviceFilter } from '@core/types/device'
import type { PageRequest } from '@core/types/pagination'

// ============================================================
// 设备导入导出 — 设备清单按条件预览 + 客户端 CSV 导出
// 对照 v1 webcode/src/pages/device/ImportExport：v1 上传/导入历史为 mock。
// v2 用真实 useDeviceList 提供"导出预览"（按制式/连接状态过滤），并支持
// 把当前页设备导出为 CSV（无三方依赖，浏览器 Blob 下载）。文件上传导入
// 未覆盖（v2 无 Dragger 等上传组件）。点击 SN 跳设备详情。
// ============================================================

const PAGE_SIZE = 20

const EXPORT_COLUMNS: { key: keyof Device; label: string }[] = [
  { key: 'sn', label: 'SN' },
  { key: 'deviceName', label: '名称' },
  { key: 'networkType', label: '制式' },
  { key: 'deviceModel', label: '型号' },
  { key: 'vendor', label: '厂商' },
  { key: 'ipAddress', label: 'IP' },
  { key: 'softwareVersion', label: '软件版本' },
  { key: 'groupName', label: '分组' },
]

function toCsvCell(value: unknown): string {
  const s = value == null ? '' : String(value)
  if (/[",\n]/.test(s)) return `"${s.replace(/"/g, '""')}"`
  return s
}

function buildCsv(rows: Device[]): string {
  const header = EXPORT_COLUMNS.map((c) => c.label).join(',')
  const body = rows
    .map((r) =>
      EXPORT_COLUMNS.map((c) => toCsvCell(r[c.key])).join(',')
    )
    .join('\n')
  return `${header}\n${body}`
}

export default function ImportExport() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [searchText, setSearchText] = useState('')
  const [networkType, setNetworkType] = useState('all')
  const [onlineFilter, setOnlineFilter] = useState<'all' | 'online' | 'offline'>(
    'all'
  )
  const [format, setFormat] = useState<'csv'>('csv')

  const queryParams = useMemo<DeviceFilter & PageRequest>(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(searchText.trim() ? { searchText: searchText.trim() } : {}),
      ...(networkType !== 'all' ? { networkType } : {}),
      ...(onlineFilter !== 'all' ? { isOnline: onlineFilter === 'online' } : {}),
    }),
    [page, searchText, networkType, onlineFilter]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useDeviceList(queryParams)

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const colCount = 6

  function handleExport() {
    if (rows.length === 0) return
    const csv = buildCsv(rows)
    // 加 UTF-8 BOM，避免 Excel 打开中文乱码
    const blob = new Blob(['﻿', csv], {
      type: 'text/csv;charset=utf-8;',
    })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    const ts = new Date()
      .toISOString()
      .replace(/[-:T]/g, '')
      .slice(0, 14)
    a.href = url
    a.download = `devices_${ts}.${format}`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <PageShell
      title="设备导入导出"
      description="设备清单按条件预览，并导出为 CSV"
      isFetching={isFetching}
    >
      {/* 导出操作卡片 */}
      <Card className="mb-4">
        <CardHeader className="p-4 pb-2">
          <CardTitle className="flex items-center gap-2 text-base font-medium">
            <Download className="size-4 text-primary" /> 导出
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-wrap items-end gap-3 p-4 pt-2">
          <div className="flex flex-col gap-1.5">
            <span className="text-xs text-muted-foreground">导出格式</span>
            <Select value={format} onValueChange={(v) => setFormat(v as 'csv')}>
              <SelectTrigger className="w-40">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="csv">CSV (.csv)</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Button onClick={handleExport} disabled={rows.length === 0}>
            <FileSpreadsheet className="size-4" /> 导出当前 {rows.length} 条
          </Button>
          <p className="text-xs text-muted-foreground">
            按下方筛选条件导出本页设备清单（共 {total} 条匹配）。
          </p>
        </CardContent>
      </Card>

      {/* 预览筛选条 */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-64 pl-9"
            placeholder="搜索 SN / 名称 / IP"
            value={searchText}
            onChange={(e) => {
              setSearchText(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <Select
          value={networkType}
          onValueChange={(v) => {
            setNetworkType(v)
            setPage(1)
          }}
        >
          <SelectTrigger className="w-32">
            <SelectValue placeholder="制式" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部制式</SelectItem>
            <SelectItem value="eNB">eNB</SelectItem>
            <SelectItem value="gNB">gNB</SelectItem>
            <SelectItem value="GSM">GSM</SelectItem>
          </SelectContent>
        </Select>
        <Select
          value={onlineFilter}
          onValueChange={(v) => {
            setOnlineFilter(v as 'all' | 'online' | 'offline')
            setPage(1)
          }}
        >
          <SelectTrigger className="w-32">
            <SelectValue placeholder="连接状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="online">在线</SelectItem>
            <SelectItem value="offline">离线</SelectItem>
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
              <TableHead>SN</TableHead>
              <TableHead>名称</TableHead>
              <TableHead>制式</TableHead>
              <TableHead>IP 地址</TableHead>
              <TableHead>软件版本</TableHead>
              <TableHead>最近在线</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={colCount}>没有匹配的设备</EmptyRow>
            ) : (
              rows.map((d: Device) => (
                <TableRow key={d.id}>
                  <TableCell>
                    <button
                      type="button"
                      className="font-mono text-xs text-primary hover:underline"
                      onClick={() => navigate(`/devices/detail/${d.sn}`)}
                    >
                      {d.sn || '—'}
                    </button>
                  </TableCell>
                  <TableCell className="text-sm">
                    {d.deviceName || d.name || '—'}
                  </TableCell>
                  <TableCell>
                    {d.networkType ? (
                      <Badge variant="outline">{d.networkType}</Badge>
                    ) : (
                      <span className="text-xs text-muted-foreground">—</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <span className="font-mono text-xs text-muted-foreground">
                      {d.ipAddress || '—'}
                    </span>
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {d.softwareVersion || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(d.lastOnlineTime)}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={PAGE_SIZE}
        onChange={setPage}
      />
    </PageShell>
  )
}
