import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Search } from 'lucide-react'

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
  formatTime,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
} from '@/components/layout/PageShell'

import { useMRMappings, useToggleMRMapping } from '@core/hooks/api/useMR'
import type { MRDeviceMapping } from '@core/mock/data/mr'

// ============================================================
// 测量报告 → 设备小区映射
// 对照 v1 webcode/src/pages/mr/DeviceMapping —— 设备/小区 MR 采集映射关系。
// 数据走真实 useMRMappings（GET /mr/mappings）+ useToggleMRMapping（启停）。
// 设备 SN 单元格链接到映射详情子路由 /mr/device-mapping/:id。
// ============================================================

const PAGE_SIZE = 20
type EnabledFilter = '' | 'true' | 'false'

export default function DeviceMapping() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [deviceSn, setDeviceSn] = useState('')
  const [enabled, setEnabled] = useState<EnabledFilter>('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
      ...(enabled ? { enabled: enabled === 'true' } : {}),
    }),
    [page, deviceSn, enabled],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useMRMappings(params)
  const toggleMutation = useToggleMRMapping()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const handleToggle = (m: MRDeviceMapping) => {
    toggleMutation.mutate({ id: m.id, enabled: !m.enabled })
  }

  const cols = ['设备 SN', '设备名称', '小区 ID', '小区名称', '采样间隔', '采集记录数', '映射状态', '最后采集', '操作']

  return (
    <PageShell
      title="设备小区映射"
      description="管理设备与小区的 MR 数据采集映射关系（启用/禁用、采样间隔）"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-56 pl-9"
              placeholder="搜索设备 SN"
              value={deviceSn}
              onChange={(e) => {
                setDeviceSn(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <select
            className="h-9 rounded-md border bg-background px-3 text-sm"
            value={enabled}
            onChange={(e) => {
              setEnabled(e.target.value as EnabledFilter)
              setPage(1)
            }}
          >
            <option value="">全部状态</option>
            <option value="true">已启用</option>
            <option value="false">已禁用</option>
          </select>
          <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
            <RefreshCcw className={isFetching ? 'animate-spin' : ''} /> 刷新
          </Button>
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
              <EmptyRow colSpan={cols.length}>暂无设备小区映射</EmptyRow>
            ) : (
              rows.map((m) => (
                <TableRow key={m.id}>
                  <TableCell>
                    <button
                      type="button"
                      className="font-mono text-xs text-primary hover:underline"
                      onClick={() => navigate(`/mr/device-mapping/${m.id}`)}
                    >
                      {m.deviceSn}
                    </button>
                  </TableCell>
                  <TableCell className="text-xs">{m.deviceName || '—'}</TableCell>
                  <TableCell className="font-mono text-xs">{m.cellId}</TableCell>
                  <TableCell className="text-xs">{m.cellName || '—'}</TableCell>
                  <TableCell className="tabular-nums text-xs">{m.samplingInterval} 分钟</TableCell>
                  <TableCell className="tabular-nums">{m.totalRecords.toLocaleString()}</TableCell>
                  <TableCell>
                    {m.enabled ? (
                      <Badge variant="success">已启用</Badge>
                    ) : (
                      <Badge variant="muted">已禁用</Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(m.lastCollectTime)}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={toggleMutation.isPending}
                      onClick={() => handleToggle(m)}
                    >
                      {m.enabled ? '禁用' : '启用'}
                    </Button>
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
