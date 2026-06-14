import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Search } from 'lucide-react'

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

import { useAlarmDefinitionList } from '@core/hooks/api/useAlarmDefinitions'
import type { AlarmDefinition, AlarmDefinitionFilter } from '@core/types/alarmDefinition'

// ============================================================
// 告警定义详情 — 对照 v1 alarm-library 二级 AlarmDefinitionTable。
// 路由 /product/alarm-library/:neType,按 ne_type 服务端分页拉取。
// 严重级别下拉从独立全量查询(不带 severityCode)派生,避免选中后收缩。
// ============================================================

const PAGE_SIZE = 20

const SEVERITY_VARIANT: Record<number, 'destructive' | 'warning' | 'default' | 'muted'> = {
  1: 'destructive',
  31001: 'destructive',
  2: 'destructive',
  31002: 'destructive',
  3: 'warning',
  31003: 'warning',
  4: 'default',
  31004: 'default',
}

export function AlarmDefinitionsPage() {
  const navigate = useNavigate()
  const { neType: rawNeType } = useParams<{ neType: string }>()
  const neType = rawNeType ? decodeURIComponent(rawNeType) : undefined

  const [keyword, setKeyword] = useState('')
  const [severityCode, setSeverityCode] = useState<string>('all')
  const [page, setPage] = useState(1)

  const listFilter = useMemo<AlarmDefinitionFilter>(
    () => ({
      neType,
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(severityCode !== 'all' ? { severityCode: Number(severityCode) } : {}),
    }),
    [neType, page, keyword, severityCode]
  )
  const { data, isLoading, isError, error, isFetching } = useAlarmDefinitionList(
    neType ? listFilter : { page: 1, pageSize: 1 }
  )

  // 严重级别选项数据源:独立查询,只按 neType 取全量,不带 severityCode/keyword 过滤。
  const severitySourceFilter = useMemo<AlarmDefinitionFilter>(
    () => ({ neType, page: 1, pageSize: 200 }),
    [neType]
  )
  const severitySourceQuery = useAlarmDefinitionList(
    neType ? severitySourceFilter : { page: 1, pageSize: 1 }
  )
  const severityOptions = useMemo(() => {
    const byCode = new Map<number, string>()
    for (const row of severitySourceQuery.data?.items ?? []) {
      if (!byCode.has(row.severityCode)) byCode.set(row.severityCode, row.severityName ?? '')
    }
    return Array.from(byCode.entries())
      .map(([code, label]) => ({ code, label: label ? `${code} - ${label}` : String(code) }))
      .sort((a, b) => a.code - b.code)
  }, [severitySourceQuery.data])

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const colCount = 7

  return (
    <PageShell
      title={`告警定义 · ${neType ?? ''}`}
      description={`共 ${total} 条告警定义`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => navigate('/product/alarm-library')}>
            <ArrowLeft className="size-4" /> 返回
          </Button>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-80 pl-9"
              placeholder="搜索标识 / 名称"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <Select
            value={severityCode}
            onValueChange={(v) => {
              setSeverityCode(v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-48">
              <SelectValue placeholder="严重级别" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部级别</SelectItem>
              {severityOptions.map((o) => (
                <SelectItem key={o.code} value={String(o.code)}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      }
    >
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-36">标识</TableHead>
              <TableHead>中文名</TableHead>
              <TableHead>英文名</TableHead>
              <TableHead>描述</TableHead>
              <TableHead className="w-40">严重级别</TableHead>
              <TableHead className="w-28">事件类型</TableHead>
              <TableHead className="w-24">界面可见</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={colCount}>
                {keyword.trim() || severityCode !== 'all'
                  ? '没有匹配的告警定义'
                  : '该网元类型暂无告警定义'}
              </EmptyRow>
            ) : (
              rows.map((row) => <DefRow key={row.id} row={row} />)
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </PageShell>
  )
}

function DefRow({ row }: { row: AlarmDefinition }) {
  return (
    <TableRow>
      <TableCell>
        <Badge variant="outline" className="font-mono text-[11px]">
          {row.identifier}
        </Badge>
      </TableCell>
      <TableCell className="max-w-[200px] truncate text-sm" title={row.cnName}>
        {row.cnName || '—'}
      </TableCell>
      <TableCell className="max-w-[220px] truncate text-sm text-muted-foreground" title={row.enName}>
        {row.enName || '—'}
      </TableCell>
      <TableCell className="max-w-[220px] truncate text-sm text-muted-foreground" title={row.description}>
        {row.description || '—'}
      </TableCell>
      <TableCell>
        <Badge variant={SEVERITY_VARIANT[row.severityCode] ?? 'muted'}>
          {row.severityCode} - {row.severityName}
        </Badge>
      </TableCell>
      <TableCell className="text-xs text-muted-foreground">{row.eventType ?? '—'}</TableCell>
      <TableCell>
        {row.isShow ? (
          <Badge variant="success">是</Badge>
        ) : (
          <Badge variant="muted">否</Badge>
        )}
      </TableCell>
    </TableRow>
  )
}

export default AlarmDefinitionsPage
