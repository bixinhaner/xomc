import { useMemo, useState } from 'react'
import { RefreshCcw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
import { cn } from '@/lib/utils'

import { useDeviceList } from '@core/hooks/api/useDevices'
import { useDeviceParameters } from '@core/hooks/api/useDeviceParameters'
import type { DeviceParameter } from '@core/types/deviceParameter'

// ============================================================
// 按参数类批量配置 — 对照 v1 config/BatchParamClass。
// 真实数据：选设备 → 拉参数；按 TR-069 路径首段聚类为「参数类」，
// 左侧类目树（真实派生）→ 右侧该类参数清单 + 多选标记。
// ============================================================

/** 取参数路径的首个对象段作为「类」分组键。 */
function classKey(path: string): string {
  const trimmed = path.replace(/^\.+/, '')
  const seg = trimmed.split('.').filter(Boolean)
  // 形如 Device.Cellular.X.... → 取前两段更可读；否则取首段
  if (seg.length >= 2) return `${seg[0]}.${seg[1]}`
  return seg[0] ?? path
}

export default function BatchParamClass() {
  const [deviceId, setDeviceId] = useState('')
  const [activeClass, setActiveClass] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())

  const deviceQuery = useDeviceList({ page: 1, pageSize: 200 })
  const devices = deviceQuery.data?.items ?? []

  const paramsQuery = useDeviceParameters(deviceId)
  const allRows: DeviceParameter[] = paramsQuery.data?.items ?? []

  const classes = useMemo(() => {
    const counts = new Map<string, number>()
    for (const p of allRows) {
      const k = classKey(p.parameterPath)
      counts.set(k, (counts.get(k) ?? 0) + 1)
    }
    return Array.from(counts.entries())
      .map(([key, count]) => ({ key, count }))
      .sort((a, b) => a.key.localeCompare(b.key))
  }, [allRows])

  const rows = useMemo(
    () => (activeClass ? allRows.filter((p) => classKey(p.parameterPath) === activeClass) : allRows),
    [allRows, activeClass]
  )

  const toggle = (path: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(path)) next.delete(path)
      else next.add(path)
      return next
    })
  }

  const cols = ['选择', '参数路径', '当前值', '类型', '可写', '最近更新']

  return (
    <PageShell
      title="按参数类批量配置"
      description="按 TR-069 参数类聚合 — 选设备查看类目并批量勾选参数"
      isFetching={paramsQuery.isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Select
            value={deviceId || 'none'}
            onValueChange={(v) => {
              setDeviceId(v === 'none' ? '' : v)
              setActiveClass('')
              setSelected(new Set())
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
      {selected.size > 0 && (
        <div className="mb-3 flex items-center gap-2 rounded-lg border border-primary/30 bg-primary/5 px-4 py-2 text-sm">
          <span className="font-medium">已勾选 {selected.size} 条参数</span>
          <Button
            variant="ghost"
            size="sm"
            className="ml-auto"
            onClick={() => setSelected(new Set())}
          >
            清空选择
          </Button>
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 md:grid-cols-[240px_1fr]">
        {/* 左侧类目树 */}
        <div className="rounded-lg border bg-card">
          <div className="border-b px-3 py-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
            参数类
          </div>
          <div className="max-h-[60vh] overflow-auto p-1">
            {!deviceId ? (
              <div className="p-3 text-sm text-muted-foreground">请先选择设备</div>
            ) : classes.length === 0 ? (
              <div className="p-3 text-sm text-muted-foreground">暂无参数类</div>
            ) : (
              <>
                <button
                  type="button"
                  onClick={() => setActiveClass('')}
                  className={cn(
                    'flex w-full items-center justify-between rounded px-2 py-1.5 text-left text-sm hover:bg-muted/50',
                    activeClass === '' && 'bg-primary/5 font-medium text-primary'
                  )}
                >
                  全部
                  <span className="text-xs text-muted-foreground">{allRows.length}</span>
                </button>
                {classes.map((c) => (
                  <button
                    key={c.key}
                    type="button"
                    onClick={() => setActiveClass(c.key)}
                    className={cn(
                      'flex w-full items-center justify-between rounded px-2 py-1.5 text-left text-sm hover:bg-muted/50',
                      activeClass === c.key && 'bg-primary/5 font-medium text-primary'
                    )}
                  >
                    <span className="truncate font-mono text-xs">{c.key}</span>
                    <span className="ml-2 shrink-0 text-xs text-muted-foreground">{c.count}</span>
                  </button>
                ))}
              </>
            )}
          </div>
        </div>

        {/* 右侧参数表 */}
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
                <EmptyRow colSpan={cols.length}>该类暂无参数</EmptyRow>
              ) : (
                rows.map((p) => (
                  <TableRow key={p.id} data-state={selected.has(p.parameterPath) ? 'selected' : undefined}>
                    <TableCell>
                      <input
                        type="checkbox"
                        aria-label={`选择 ${p.parameterPath}`}
                        className="size-4 cursor-pointer accent-primary"
                        checked={selected.has(p.parameterPath)}
                        onChange={() => toggle(p.parameterPath)}
                      />
                    </TableCell>
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
      </div>
    </PageShell>
  )
}
