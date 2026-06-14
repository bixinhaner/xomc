import { useMemo, useState } from 'react'
import { RefreshCcw } from 'lucide-react'

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
import type { SystemLog } from '@core/mock/data/logs'

import { LogDetailModal } from './LogDetailModal'
import { LogNavTabs, PAGE_SIZE, SearchInput, Stat } from './_shared'

// ===========================================================================
// 系统日志（对齐 v1 log/system）— 系统运行日志，按级别/来源/关键字筛选，ERROR 可看详情
// ===========================================================================

type LogLevel = SystemLog['level']

const LEVEL_VARIANT: Record<LogLevel, 'default' | 'warning' | 'destructive' | 'muted'> = {
  INFO: 'default',
  WARN: 'warning',
  ERROR: 'destructive',
  DEBUG: 'muted',
}

const LEVELS: LogLevel[] = ['INFO', 'WARN', 'ERROR', 'DEBUG']

export default function SystemLog() {
  const [page, setPage] = useState(1)
  const [level, setLevel] = useState<LogLevel | ''>('')
  const [source, setSource] = useState('')
  const [keyword, setKeyword] = useState('')
  const [detail, setDetail] = useState<SystemLog | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(level ? { level } : {}),
      ...(source.trim() ? { source: source.trim() } : {}),
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, level, source, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useSystemLogs(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const counts = useMemo(() => {
    const acc: Record<LogLevel, number> = { INFO: 0, WARN: 0, ERROR: 0, DEBUG: 0 }
    for (const r of rows) acc[r.level] = (acc[r.level] ?? 0) + 1
    return acc
  }, [rows])

  const cols = ['级别', '来源', '消息', '时间', '']

  return (
    <PageShell
      title="系统日志"
      description="系统运行日志 · 按级别 / 来源 / 关键字检索"
      isFetching={isFetching}
      toolbar={<LogNavTabs />}
    >
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="本页总数" value={rows.length} />
        <Stat label="ERROR" value={counts.ERROR} tone="rose" />
        <Stat label="WARN" value={counts.WARN} tone="amber" />
        <Stat label="INFO" value={counts.INFO} tone="emerald" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <SearchInput
          className="w-72"
          placeholder="消息关键字"
          value={keyword}
          onChange={(v) => {
            setKeyword(v)
            setPage(1)
          }}
        />
        <Input
          className="w-44"
          placeholder="来源 (如 device)"
          value={source}
          onChange={(e) => {
            setSource(e.target.value)
            setPage(1)
          }}
        />
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
            {LEVELS.map((l) => (
              <SelectItem key={l} value={l}>
                {l}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <div className="ml-auto">
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw className="size-4" /> 刷新
          </Button>
        </div>
      </div>

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c, i) => (
                <TableHead key={c || `c-${i}`}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无系统日志</EmptyRow>
            ) : (
              rows.map((l) => {
                const hasDetail = l.level === 'ERROR' || Boolean(l.details)
                return (
                  <TableRow key={l.id}>
                    <TableCell>
                      <Badge variant={LEVEL_VARIANT[l.level]}>{l.level}</Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs">{l.source}</TableCell>
                    <TableCell className="max-w-[560px] truncate font-mono text-xs">
                      {l.message}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(l.timestamp)}
                    </TableCell>
                    <TableCell className="text-right">
                      {hasDetail ? (
                        <Button variant="ghost" size="sm" onClick={() => setDetail(l)}>
                          详情
                        </Button>
                      ) : null}
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />

      <LogDetailModal open={detail !== null} title="系统日志详情" onClose={() => setDetail(null)}>
        {detail && (
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <Badge variant={LEVEL_VARIANT[detail.level]}>{detail.level}</Badge>
              <span className="font-mono text-xs text-muted-foreground">{detail.source}</span>
              <span className="ml-auto text-xs text-muted-foreground">
                {formatTime(detail.timestamp)}
              </span>
            </div>
            <pre className="whitespace-pre-wrap break-all rounded-md bg-muted p-3 font-mono text-xs">
              {detail.message}
              {detail.details ? `\n\n${detail.details}` : ''}
            </pre>
          </div>
        )}
      </LogDetailModal>
    </PageShell>
  )
}
