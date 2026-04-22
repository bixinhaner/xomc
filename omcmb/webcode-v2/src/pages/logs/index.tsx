import { useMemo, useState } from 'react'
import { Search, RefreshCcw } from 'lucide-react'

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

import { useSystemLogs } from '@core/hooks/api/useLogs'

type LogLevel = 'INFO' | 'WARN' | 'ERROR' | 'DEBUG'

const LEVEL_VARIANT: Record<LogLevel, 'default' | 'warning' | 'destructive' | 'muted'> = {
  INFO: 'default',
  WARN: 'warning',
  ERROR: 'destructive',
  DEBUG: 'muted',
}

export function LogsPage() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(30)
  const [level, setLevel] = useState<LogLevel | ''>('')
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(level ? { level } : {}),
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, pageSize, level, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useSystemLogs(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['级别', '来源', '消息', '时间']

  return (
    <PageShell
      title="系统日志"
      description="运行时日志 · 按级别/来源/关键字过滤"
      isFetching={isFetching}
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-80 pl-9"
              placeholder="消息关键字"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <Select
            value={level || 'all'}
            onValueChange={(v) => {
              setLevel(v === 'all' ? '' : (v as LogLevel))
              setPage(1)
            }}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="级别" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部级别</SelectItem>
              <SelectItem value="INFO">INFO</SelectItem>
              <SelectItem value="WARN">WARN</SelectItem>
              <SelectItem value="ERROR">ERROR</SelectItem>
              <SelectItem value="DEBUG">DEBUG</SelectItem>
            </SelectContent>
          </Select>
          <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
          </Button>
        </>
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
              <EmptyRow colSpan={cols.length}>暂无日志</EmptyRow>
            ) : (
              rows.map((l) => (
                <TableRow key={l.id}>
                  <TableCell>
                    <Badge variant={LEVEL_VARIANT[l.level]}>{l.level}</Badge>
                  </TableCell>
                  <TableCell className="font-mono text-xs">{l.source}</TableCell>
                  <TableCell className="max-w-[600px] truncate">{l.message}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(l.timestamp)}
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
