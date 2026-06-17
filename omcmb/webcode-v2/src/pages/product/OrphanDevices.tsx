import { useState } from 'react'
import { Search, X } from 'lucide-react'

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
  formatTime,
} from '@/components/layout/PageShell'

import { useOrphanDevices } from '@core/hooks/api/useProducts'
import type { OrphanDevice } from '@core/types/product'

// ============================================================
// 孤儿设备 — 对照 v1 webcode/src/pages/product/orphan-devices。
// 纯只读列表;服务端分页 + SN/OUI/产品类型/厂商模糊搜索。
// ============================================================

const PAGE_SIZE = 20

export function OrphanDevicesPage() {
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)

  const { data, isLoading, isError, error, isFetching } = useOrphanDevices({
    page,
    pageSize: PAGE_SIZE,
    search,
  })

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const colCount = 6

  function applySearch() {
    setSearch(searchInput.trim())
    setPage(1)
  }

  return (
    <PageShell
      title="孤儿设备"
      description={`共 ${total} 台未匹配产品的设备`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-80 pl-9"
              placeholder="搜索 SN / OUI / 产品类型 / 厂商"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') applySearch()
              }}
            />
          </div>
          <Button variant="outline" size="sm" onClick={applySearch}>
            搜索
          </Button>
          {search && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setSearchInput('')
                setSearch('')
                setPage(1)
              }}
            >
              <X className="size-4" /> 重置
            </Button>
          )}
        </div>
      }
    >
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-52">SN</TableHead>
              <TableHead>设备名</TableHead>
              <TableHead className="w-28">OUI</TableHead>
              <TableHead>产品类型</TableHead>
              <TableHead className="w-36">厂商</TableHead>
              <TableHead className="w-48">最近上报</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={colCount}>
                {search ? `未找到与 “${search}” 匹配的设备` : '暂无数据'}
              </EmptyRow>
            ) : (
              rows.map((row) => <OrphanRow key={row.id} row={row} />)
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </PageShell>
  )
}

function OrphanRow({ row }: { row: OrphanDevice }) {
  return (
    <TableRow>
      <TableCell className="font-mono text-xs">{row.serialNumber || '—'}</TableCell>
      <TableCell className="max-w-[200px] truncate text-sm" title={row.deviceName}>
        {row.deviceName || '—'}
      </TableCell>
      <TableCell className="font-mono text-xs text-muted-foreground">{row.oui || '—'}</TableCell>
      <TableCell>
        <Badge variant="warning">{row.productClass || '—'}</Badge>
      </TableCell>
      <TableCell className="text-sm text-muted-foreground">{row.manufacturer || '—'}</TableCell>
      <TableCell className="text-xs text-muted-foreground">{formatTime(row.lastInformAt)}</TableCell>
    </TableRow>
  )
}

export default OrphanDevicesPage
