import { useMemo, useState } from 'react'
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
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'

import { useDeviceList } from '@core/hooks/api/useDevices'
import { useDeviceParameters } from '@core/hooks/api/useDeviceParameters'
import type { DeviceParameter } from '@core/types/deviceParameter'

// ============================================================
// 参数列表 — 对照 v1 config/ParamList（参数字典）。
// 真实数据：选设备 → 拉该设备的 TR-069 参数（DeviceParameter）。
// 字典语义在真机上即「设备已发现参数清单」，支持按路径/类型过滤。
// ============================================================

const TYPE_VARIANT: Record<
  string,
  'default' | 'secondary' | 'success' | 'warning' | 'muted'
> = {
  string: 'default',
  int: 'success',
  unsignedInt: 'success',
  boolean: 'warning',
  dateTime: 'secondary',
}

export default function ParamList() {
  const [deviceId, setDeviceId] = useState('')
  const [keyword, setKeyword] = useState('')

  const deviceQuery = useDeviceList({ page: 1, pageSize: 200 })
  const devices = deviceQuery.data?.items ?? []

  const paramsQuery = useDeviceParameters(deviceId)
  const allRows: DeviceParameter[] = paramsQuery.data?.items ?? []

  const rows = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return allRows
    return allRows.filter(
      (p) =>
        p.parameterPath.toLowerCase().includes(kw) ||
        p.parameterValue.toLowerCase().includes(kw)
    )
  }, [allRows, keyword])

  const cols = ['参数路径', '当前值', '类型', '可写', '最近更新']

  return (
    <PageShell
      title="参数列表"
      description="设备 TR-069 参数字典 — 选设备查看已发现参数清单"
      isFetching={paramsQuery.isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Select value={deviceId || 'none'} onValueChange={(v) => setDeviceId(v === 'none' ? '' : v)}>
            <SelectTrigger className="w-72">
              <SelectValue placeholder="选择设备" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="none">请选择设备</SelectItem>
              {devices.map((d) => (
                <SelectItem key={d.id} value={d.id}>
                  {d.sn}
                  {d.name && d.name !== d.sn ? ` (${d.name})` : ''}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-64 pl-9"
              placeholder="按路径 / 值过滤"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <div className="ml-auto">
            <Button
              variant="outline"
              size="sm"
              disabled={!deviceId}
              onClick={() => void paramsQuery.refetch()}
            >
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
            {!deviceId ? (
              <EmptyRow colSpan={cols.length}>请先选择设备以加载参数</EmptyRow>
            ) : paramsQuery.isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : paramsQuery.isError ? (
              <ErrorRow colSpan={cols.length} error={paramsQuery.error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>
                {keyword ? '没有匹配的参数' : '该设备暂无参数'}
              </EmptyRow>
            ) : (
              rows.map((p) => (
                <TableRow key={p.id}>
                  <TableCell className="font-mono text-xs">{p.parameterPath}</TableCell>
                  <TableCell className="max-w-xs truncate text-sm">
                    {p.parameterValue || '—'}
                  </TableCell>
                  <TableCell>
                    <Badge variant={TYPE_VARIANT[p.parameterType] ?? 'muted'}>
                      {p.parameterType || '—'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    {p.writable ? (
                      <Badge variant="success">可写</Badge>
                    ) : (
                      <Badge variant="muted">只读</Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(p.lastUpdatedAt)}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>
      {deviceId && rows.length > 0 && (
        <div className="mt-3 text-sm text-muted-foreground">
          共 {rows.length} 条参数
        </div>
      )}
    </PageShell>
  )
}
