import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Search } from 'lucide-react'

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
} from '@/components/layout/PageShell'

import { useDeviceList } from '@core/hooks/api/useDevices'
import type { Device, DeviceFilter } from '@core/types/device'
import type { PageRequest } from '@core/types/pagination'

// ============================================================
// 小区管理 — 对照 v1 config/CellManagement。
// 真实数据：从设备列表派生每基站的小区信息（cellId/PCI/TAC/带宽/频点/状态）。
// 行 → 链接到设备详情 /devices/detail/:sn。
// ============================================================

type NetFilter = 'all' | 'eNB' | 'gNB' | 'GSM'

const CELL_STATUS: Record<string, { variant: 'success' | 'warning' | 'destructive' | 'muted'; label: string }> = {
  active: { variant: 'success', label: '激活' },
  inactive: { variant: 'muted', label: '未激活' },
  maintenance: { variant: 'warning', label: '维护' },
  fault: { variant: 'destructive', label: '故障' },
}

function cellStatusBadge(status: string) {
  const s = CELL_STATUS[status?.toLowerCase()]
  if (s) return <Badge variant={s.variant}>{s.label}</Badge>
  return <span className="text-xs text-muted-foreground">{status || '—'}</span>
}

export default function CellManagement() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [searchText, setSearchText] = useState('')
  const [networkType, setNetworkType] = useState<NetFilter>('all')

  const queryParams = useMemo<DeviceFilter & PageRequest>(
    () => ({
      page,
      pageSize,
      ...(searchText.trim() ? { searchText: searchText.trim() } : {}),
      ...(networkType !== 'all' ? { networkType } : {}),
    }),
    [page, searchText, networkType]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useDeviceList(queryParams)
  const rows: Device[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['小区 ID', '所属基站', '制式', 'PCI', 'TAC', '频点', '带宽', '状态']

  return (
    <PageShell
      title="小区管理"
      description="按基站派生的小区信息（PCI / TAC / 频点 / 带宽 / 状态）"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-64 pl-9"
              placeholder="搜索基站 SN / 名称"
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
              setNetworkType(v as NetFilter)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="制式" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部制式</SelectItem>
              <SelectItem value="eNB">eNB (LTE)</SelectItem>
              <SelectItem value="gNB">gNB (NR)</SelectItem>
              <SelectItem value="GSM">GSM</SelectItem>
            </SelectContent>
          </Select>
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => void refetch()}>
              <RefreshCcw className="size-4" /> 刷新
            </Button>
          </div>
        </div>
      }
    >
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
              <EmptyRow colSpan={cols.length}>暂无小区数据</EmptyRow>
            ) : (
              rows.map((d) => {
                const cellId = d.nrCellId || d.cellId || d.eci || '—'
                const freq = d.dlEarfcn || d.arfcn || d.downlinkFrequency || '—'
                return (
                  <TableRow key={d.id}>
                    <TableCell className="font-mono text-xs">{cellId}</TableCell>
                    <TableCell>
                      <button
                        type="button"
                        className="text-left text-primary hover:underline"
                        onClick={() => navigate(`/devices/detail/${d.sn}`)}
                      >
                        {d.deviceName || d.name || d.sn}
                      </button>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{d.networkType || '—'}</Badge>
                    </TableCell>
                    <TableCell className="tabular-nums">{d.pci || '—'}</TableCell>
                    <TableCell className="tabular-nums">{d.tac || '—'}</TableCell>
                    <TableCell className="font-mono text-xs">{freq}</TableCell>
                    <TableCell className="text-xs">{d.bandwidth || '—'}</TableCell>
                    <TableCell>{cellStatusBadge(d.cellStatus)}</TableCell>
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
