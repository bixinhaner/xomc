import { useMemo, useState } from 'react'
import { Loader2, Pencil, RefreshCcw, Search, X } from 'lucide-react'

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
import {
  useDeviceParameters,
  useUpdateParameters,
} from '@core/hooks/api/useDeviceParameters'
import type { DeviceParameter } from '@core/types/deviceParameter'

// ============================================================
// 在线参数配置 — 对照 v1 config/LiveParamConfig。
// 真实数据：选设备 → 拉运行期参数；可写参数行内编辑（SetParameterValues / Path A）。
// ============================================================

export default function LiveParamConfig() {
  const [deviceId, setDeviceId] = useState('')
  const [keyword, setKeyword] = useState('')
  const [editingPath, setEditingPath] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')

  const deviceQuery = useDeviceList({ page: 1, pageSize: 200 })
  const devices = deviceQuery.data?.items ?? []

  const paramsQuery = useDeviceParameters(deviceId)
  const updateParams = useUpdateParameters()
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

  const startEdit = (p: DeviceParameter) => {
    setEditingPath(p.parameterPath)
    setEditValue(p.parameterValue)
  }

  const cancelEdit = () => {
    setEditingPath(null)
    setEditValue('')
  }

  const saveEdit = (p: DeviceParameter) => {
    if (!deviceId) return
    updateParams.mutate(
      {
        deviceId,
        parameters: [
          {
            parameterPath: p.parameterPath,
            parameterValue: editValue,
            parameterType: p.parameterType,
          },
        ],
      },
      {
        onSuccess: () => cancelEdit(),
      }
    )
  }

  const cols = ['参数路径', '当前值', '类型', '可写', '最近更新', '操作']

  return (
    <PageShell
      title="在线参数配置"
      description="按设备查看运行期参数 — 可写参数支持行内修改下发"
      isFetching={paramsQuery.isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Select
            value={deviceId || 'none'}
            onValueChange={(v) => {
              setDeviceId(v === 'none' ? '' : v)
              cancelEdit()
            }}
          >
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
      {updateParams.isError && (
        <div className="mb-3 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-2 text-sm text-destructive">
          参数下发失败：
          {updateParams.error instanceof Error ? updateParams.error.message : '未知错误'}
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
              rows.map((p) => {
                const editing = editingPath === p.parameterPath
                return (
                  <TableRow key={p.id}>
                    <TableCell className="font-mono text-xs">{p.parameterPath}</TableCell>
                    <TableCell className="max-w-xs">
                      {editing ? (
                        <Input
                          className="h-8 w-44"
                          value={editValue}
                          onChange={(e) => setEditValue(e.target.value)}
                        />
                      ) : (
                        <span className="truncate text-sm">{p.parameterValue || '—'}</span>
                      )}
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
                    <TableCell>
                      {editing ? (
                        <div className="flex items-center gap-1">
                          <Button
                            size="sm"
                            disabled={updateParams.isPending}
                            onClick={() => saveEdit(p)}
                          >
                            {updateParams.isPending ? (
                              <Loader2 className="size-3.5 animate-spin" />
                            ) : null}
                            保存
                          </Button>
                          <Button variant="ghost" size="sm" onClick={cancelEdit}>
                            <X className="size-3.5" />
                          </Button>
                        </div>
                      ) : (
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={!p.writable || editingPath !== null}
                          onClick={() => startEdit(p)}
                        >
                          <Pencil className="size-3.5" /> 编辑
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                )
              })
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
