import { useMemo, useState } from 'react'
import { Loader2, RefreshCcw, Search, RotateCw } from 'lucide-react'

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

import { useDeviceList, useSyncDeviceParams } from '@core/hooks/api/useDevices'
import { useDeviceParameters } from '@core/hooks/api/useDeviceParameters'
import type { DeviceParameter } from '@core/types/deviceParameter'

// ============================================================
// 参数同步 — 对照 v1 config/ParamSync。
// 真实数据：选网元(设备) → 拉当前参数快照；触发「同步」走 Path B
// （GetParameterValues 回读真机）刷新参数缓存。
// ============================================================

export default function ParamSync() {
  const [deviceId, setDeviceId] = useState('')
  const [keyword, setKeyword] = useState('')

  const deviceQuery = useDeviceList({ page: 1, pageSize: 200 })
  const devices = deviceQuery.data?.items ?? []

  const paramsQuery = useDeviceParameters(deviceId)
  const syncParams = useSyncDeviceParams()
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

  const handleSync = () => {
    if (!deviceId) return
    syncParams.mutate(
      { deviceId, force: true },
      { onSuccess: () => void paramsQuery.refetch() }
    )
  }

  const cols = ['参数路径', '当前值', '类型', '可写', '最近更新']

  return (
    <PageShell
      title="参数同步"
      description="按网元回读运行期参数（Path B / GetParameterValues）"
      isFetching={paramsQuery.isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Select value={deviceId || 'none'} onValueChange={(v) => setDeviceId(v === 'none' ? '' : v)}>
            <SelectTrigger className="w-72">
              <SelectValue placeholder="选择网元" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="none">请选择网元</SelectItem>
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
          <div className="ml-auto flex items-center gap-2">
            <Button
              size="sm"
              disabled={!deviceId || syncParams.isPending}
              onClick={handleSync}
            >
              {syncParams.isPending ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <RotateCw className="size-4" />
              )}
              同步参数
            </Button>
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
      {syncParams.isError && (
        <div className="mb-3 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-2 text-sm text-destructive">
          同步触发失败：
          {syncParams.error instanceof Error ? syncParams.error.message : '未知错误'}
        </div>
      )}
      {syncParams.isSuccess && (
        <div className="mb-3 rounded-md border border-emerald-500/30 bg-emerald-500/5 px-4 py-2 text-sm text-emerald-600 dark:text-emerald-400">
          已触发参数回读，稍后刷新可见最新值。
        </div>
      )}

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
              <EmptyRow colSpan={cols.length}>请先选择网元以加载参数</EmptyRow>
            ) : paramsQuery.isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : paramsQuery.isError ? (
              <ErrorRow colSpan={cols.length} error={paramsQuery.error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>
                {keyword ? '没有匹配的参数' : '该网元暂无参数'}
              </EmptyRow>
            ) : (
              rows.map((p) => (
                <TableRow key={p.id}>
                  <TableCell className="font-mono text-xs">{p.parameterPath}</TableCell>
                  <TableCell className="max-w-xs truncate text-sm">
                    {p.parameterValue || '—'}
                  </TableCell>
                  <TableCell>
                    <Badge variant="muted">{p.parameterType || '—'}</Badge>
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
        <div className="mt-3 text-sm text-muted-foreground">共 {rows.length} 条参数</div>
      )}
    </PageShell>
  )
}
