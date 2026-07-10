import { useMemo, useState } from 'react'
import { Loader2, RotateCcw, Search, Trash2, X } from 'lucide-react'

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
import { cn } from '@/lib/utils'

import {
  useRecycleBinList,
  useRestoreDevices,
  usePermanentDeleteDevices,
} from '@core/hooks/api/useDevices'
import type { RecycleBinFilter } from '@core/hooks/api/useDevices'
import type { Device } from '@core/types/device'
import { formatRecycleOperator } from '@core/utils/recycleBin'

// ============================================================
// 设备回收站 — 已删除设备的恢复 / 永久删除
// 对照 v1 webcode/src/pages/device/RecycleBin（real useRecycleBinList 等）。
// 导入(Excel) 未覆盖（v2 无文件导入组件）。
// ============================================================

const PAGE_SIZE = 20

const NETWORK_VARIANT: Record<string, 'default' | 'success' | 'warning'> = {
  eNB: 'default',
  gNB: 'success',
  GSM: 'warning',
}

function calcOfflineDays(lastOnlineTime: string, deletedAt: string): number {
  const ref = deletedAt || lastOnlineTime
  if (!ref) return 0
  const t = new Date(ref).getTime()
  if (Number.isNaN(t)) return 0
  return Math.max(0, Math.floor((Date.now() - t) / 86400000))
}

export default function RecycleBin() {
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  // #378: 恢复部分成功（含 SN 冲突跳过）时的内联提示。
  const [restoreNotice, setRestoreNotice] = useState<string | null>(null)

  const params = useMemo<RecycleBinFilter>(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(search.trim() ? { search: search.trim() } : {}),
    }),
    [page, search]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useRecycleBinList(params)
  const restore = useRestoreDevices()
  const permanentDelete = usePermanentDeleteDevices()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const pageIds = useMemo(() => rows.map((d: Device) => d.id), [rows])
  const allSelected =
    pageIds.length > 0 && pageIds.every((id) => selectedIds.has(id))
  const someSelected = pageIds.some((id) => selectedIds.has(id))
  const selectedCount = selectedIds.size

  function toggleOne(id: string) {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function toggleAll() {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (allSelected) pageIds.forEach((id) => next.delete(id))
      else pageIds.forEach((id) => next.add(id))
      return next
    })
  }

  function clearSelection() {
    setSelectedIds(new Set())
  }

  function handleRestore() {
    if (selectedCount === 0) return
    setRestoreNotice(null)
    // #378: 恢复可部分成功——SN 冲突设备被后端跳过并回传 conflicts。
    restore.mutate(Array.from(selectedIds), {
      onSuccess: (res) => {
        clearSelection()
        if (res.skipped > 0) {
          const sns = res.conflicts.map((c) => c.serialNumber).join('、')
          setRestoreNotice(
            `已恢复 ${res.restored} 台，${res.skipped} 台因序列号已存在活跃设备被跳过：${sns}`
          )
        }
      },
    })
  }

  function handleDelete() {
    if (selectedCount === 0) return
    if (
      !window.confirm(
        `确认永久删除选中的 ${selectedCount} 台设备？此操作不可恢复。`
      )
    )
      return
    permanentDelete.mutate(Array.from(selectedIds), {
      onSuccess: clearSelection,
    })
  }

  const colCount = 9

  return (
    <PageShell
      title="设备回收站"
      description={`共 ${total} 台已删除设备 · 可恢复或永久删除`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-72 pl-9"
              placeholder="搜索 SN / 名称 / 分组"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value)
                setPage(1)
              }}
            />
          </div>
          {search.trim() && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setSearch('')
                setPage(1)
              }}
            >
              <X className="size-4" /> 重置
            </Button>
          )}
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RotateCcw className="size-4" /> 刷新
            </Button>
          </div>
        </div>
      }
    >
      {selectedCount > 0 && (
        <div className="mb-3 flex flex-wrap items-center gap-2 rounded-lg border border-primary/30 bg-primary/5 px-4 py-2.5 text-sm">
          <span className="font-medium">已选 {selectedCount} 台</span>
          <div className="ml-auto flex flex-wrap items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={restore.isPending}
              onClick={handleRestore}
            >
              {restore.isPending ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <RotateCcw className="size-4" />
              )}
              恢复
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={permanentDelete.isPending}
              onClick={handleDelete}
              className="text-destructive hover:text-destructive"
            >
              {permanentDelete.isPending ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <Trash2 className="size-4" />
              )}
              永久删除
            </Button>
            <Button variant="ghost" size="sm" onClick={clearSelection}>
              <X className="size-4" /> 取消选择
            </Button>
          </div>
        </div>
      )}

      {(restore.isError || permanentDelete.isError) && (
        <div className="mb-3 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-2 text-sm text-destructive">
          操作失败：
          {(restore.error ?? permanentDelete.error) instanceof Error
            ? (restore.error ?? permanentDelete.error)!.message
            : '未知错误'}
        </div>
      )}

      {restoreNotice && (
        // #378: 恢复部分成功（SN 冲突跳过）的内联告警提示（amber，与 warning Badge 同色系）。
        <div className="mb-3 rounded-md border border-amber-500/30 bg-amber-500/5 px-4 py-2 text-sm text-amber-600 dark:text-amber-400">
          {restoreNotice}
        </div>
      )}

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-10">
                <input
                  type="checkbox"
                  aria-label="全选本页"
                  className="size-4 cursor-pointer accent-primary"
                  checked={allSelected}
                  ref={(el) => {
                    if (el) el.indeterminate = !allSelected && someSelected
                  }}
                  onChange={toggleAll}
                />
              </TableHead>
              <TableHead>SN</TableHead>
              <TableHead>制式</TableHead>
              <TableHead>主机名</TableHead>
              <TableHead>MAC</TableHead>
              <TableHead>离线天数</TableHead>
              <TableHead>分组</TableHead>
              <TableHead>账户</TableHead>
              <TableHead>删除时间</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={colCount}>
                {search.trim() ? '没有匹配的设备' : '回收站为空'}
              </EmptyRow>
            ) : (
              rows.map((d: Device) => (
                <TableRow
                  key={d.id}
                  data-state={selectedIds.has(d.id) ? 'selected' : undefined}
                >
                  <TableCell>
                    <input
                      type="checkbox"
                      aria-label={`选择 ${d.sn}`}
                      className="size-4 cursor-pointer accent-primary"
                      checked={selectedIds.has(d.id)}
                      onChange={() => toggleOne(d.id)}
                    />
                  </TableCell>
                  <TableCell>
                    <span className="font-mono text-xs">{d.sn || '—'}</span>
                  </TableCell>
                  <TableCell>
                    {d.networkType ? (
                      <Badge
                        variant={NETWORK_VARIANT[d.networkType] ?? 'muted'}
                      >
                        {d.networkType}
                      </Badge>
                    ) : (
                      <span className="text-xs text-muted-foreground">—</span>
                    )}
                  </TableCell>
                  <TableCell className="text-sm">
                    {d.hostName || d.deviceName || d.name || '—'}
                  </TableCell>
                  <TableCell>
                    <span className="font-mono text-xs text-muted-foreground">
                      {d.macAddress || '—'}
                    </span>
                  </TableCell>
                  <TableCell
                    className={cn(
                      'text-sm tabular-nums',
                      calcOfflineDays(d.lastOnlineTime, d.deletedAt ?? '') > 30 &&
                        'text-amber-600 dark:text-amber-400'
                    )}
                  >
                    {calcOfflineDays(d.lastOnlineTime, d.deletedAt ?? '')} 天
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {d.groupName || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatRecycleOperator(d.deletedBy)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(d.deletedAt)}
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
