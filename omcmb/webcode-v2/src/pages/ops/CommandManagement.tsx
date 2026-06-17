import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Search, Eye } from 'lucide-react'

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
  formatTime,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useOpsCommandRecords } from '@core/hooks/api/useOpsTools'
import type { OpsCommandRecord } from '@core/mock/data/opsTools'

// ============================================================
// 命令执行记录 — 对齐 v1 webcode/src/pages/ops/CommandManagement
// 列表 + 筛选；点命令进 /ops/commands/:id 详情看完整输出
// ============================================================

type ResultFilter = '' | 'success' | 'failed'

function formatDuration(ms: number): string {
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)} s`
  return `${ms} ms`
}

function durationTone(ms: number): string {
  if (ms > 5000) return 'text-destructive'
  if (ms > 2000) return 'text-amber-600 dark:text-amber-400'
  return 'text-emerald-600 dark:text-emerald-400'
}

export default function CommandManagement() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [deviceSn, setDeviceSn] = useState('')
  const [operator, setOperator] = useState('')
  const [result, setResult] = useState<ResultFilter>('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
      ...(operator.trim() ? { operator: operator.trim() } : {}),
      ...(result === 'success' ? { success: true } : {}),
      ...(result === 'failed' ? { success: false } : {}),
    }),
    [page, pageSize, deviceSn, operator, result]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useOpsCommandRecords(params)

  const rawRows: OpsCommandRecord[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  // keyword 后端不支持，沿用 v1 在当前页结果上做 client-side 模糊匹配
  const rows = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return rawRows
    return rawRows.filter(
      (r) =>
        r.commandText.toLowerCase().includes(kw) || r.deviceSn.toLowerCase().includes(kw)
    )
  }, [rawRows, keyword])

  const cols = ['执行时间', '命令', '设备名称', '设备 SN', '操作人', '耗时', '结果', '输出摘要', '操作']

  return (
    <PageShell title="命令执行记录" description="设备 MML / 远程命令的执行历史与输出审计">
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-56 pl-9"
            placeholder="命令 / 设备 SN 模糊匹配"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
        <Input
          className="w-44"
          placeholder="设备 SN"
          value={deviceSn}
          onChange={(e) => {
            setDeviceSn(e.target.value)
            setPage(1)
          }}
        />
        <Input
          className="w-36"
          placeholder="操作人"
          value={operator}
          onChange={(e) => {
            setOperator(e.target.value)
            setPage(1)
          }}
        />
        <Select
          value={result || 'all'}
          onValueChange={(v) => {
            setResult(v === 'all' ? '' : (v as ResultFilter))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-32">
            <SelectValue placeholder="结果" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部结果</SelectItem>
            <SelectItem value="success">成功</SelectItem>
            <SelectItem value="failed">失败</SelectItem>
          </SelectContent>
        </Select>
        <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
          <RefreshCcw className={cn(isFetching && 'animate-spin')} /> 刷新
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
              <EmptyRow colSpan={cols.length}>暂无命令记录</EmptyRow>
            ) : (
              rows.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(r.executeTime)}
                  </TableCell>
                  <TableCell>
                    <button
                      type="button"
                      className="rounded bg-muted/60 px-1.5 py-0.5 text-left font-mono text-xs hover:underline"
                      onClick={() => navigate(`/ops/commands`)}
                    >
                      {r.commandText}
                    </button>
                  </TableCell>
                  <TableCell className="text-xs">{r.deviceName || '—'}</TableCell>
                  <TableCell className="font-mono text-xs">{r.deviceSn}</TableCell>
                  <TableCell className="text-xs">{r.operator}</TableCell>
                  <TableCell className={cn('font-mono text-xs', durationTone(r.duration))}>
                    {formatDuration(r.duration)}
                  </TableCell>
                  <TableCell>
                    <Badge variant={r.success ? 'success' : 'destructive'}>
                      {r.success ? '成功' : '失败'}
                    </Badge>
                  </TableCell>
                  <TableCell className="max-w-[260px] truncate text-xs text-muted-foreground">
                    {!r.success && r.errorMessage ? (
                      <span className="text-destructive">{r.errorMessage}</span>
                    ) : (
                      (r.output || '').split('\n')[0] || '—'
                    )}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => navigate(`/ops/commands`)}
                      title="详情"
                    >
                      <Eye className="size-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={setPage} />
    </PageShell>
  )
}
