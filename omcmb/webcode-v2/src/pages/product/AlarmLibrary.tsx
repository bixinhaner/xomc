import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Search, X } from 'lucide-react'

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
  TableCard,
} from '@/components/layout/PageShell'

import { useAlarmNeTypeStats } from '@core/hooks/api/useAlarmDefinitions'
import type { AlarmNeTypeStat } from '@core/types/alarmDefinition'

// ============================================================
// 告警库 — 对照 v1 webcode/src/pages/product/alarm-library 一级 NeTypes 聚合。
// 按 (ne_type, loaded_from) 聚合;点网元类型进该类型的告警定义详情。
// 后端 listNeTypes 不带过滤,数据量小,前端按 neType 客户端模糊过滤。
// ============================================================

function basename(path: string): string {
  if (!path) return ''
  return path.split('/').pop() || path
}

export function AlarmLibraryPage() {
  const navigate = useNavigate()
  const [keyword, setKeyword] = useState('')

  const { data, isLoading, isError, error, isFetching, refetch } = useAlarmNeTypeStats()

  const items = useMemo<AlarmNeTypeStat[]>(() => {
    const all = data?.items ?? []
    const k = keyword.trim().toLowerCase()
    if (!k) return all
    return all.filter((row) => row.neType.toLowerCase().includes(k))
  }, [data, keyword])

  const colCount = 7

  return (
    <PageShell
      title="告警库"
      description={`共 ${items.length} 个网元类型`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-80 pl-9"
              placeholder="搜索网元类型"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          {keyword.trim() && (
            <Button variant="ghost" size="sm" onClick={() => setKeyword('')}>
              <X className="size-4" /> 重置
            </Button>
          )}
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => void refetch()}>
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
              <TableHead className="w-40">网元类型</TableHead>
              <TableHead>加载源</TableHead>
              <TableHead className="w-24 text-right">总数</TableHead>
              <TableHead className="w-20 text-right">紧急</TableHead>
              <TableHead className="w-20 text-right">重要</TableHead>
              <TableHead className="w-20 text-right">次要</TableHead>
              <TableHead className="w-20 text-right">警告</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={error} />
            ) : items.length === 0 ? (
              <EmptyRow colSpan={colCount}>
                {keyword.trim() ? '没有匹配的网元类型' : '暂无告警定义'}
              </EmptyRow>
            ) : (
              items.map((row) => (
                <NeTypeRow key={`${row.neType}__${row.loadedFrom}`} row={row} navigate={navigate} />
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>
    </PageShell>
  )
}

function NeTypeRow({
  row,
  navigate,
}: {
  row: AlarmNeTypeStat
  navigate: ReturnType<typeof useNavigate>
}) {
  return (
    <TableRow>
      <TableCell>
        <button
          type="button"
          className="font-medium text-primary hover:underline"
          onClick={() => navigate(`/product/alarm-library`)}
        >
          {row.neType}
        </button>
      </TableCell>
      <TableCell className="font-mono text-xs text-muted-foreground" title={row.loadedFrom}>
        {row.loadedFrom ? basename(row.loadedFrom) : <Badge variant="muted">手工新增</Badge>}
      </TableCell>
      <TableCell className="text-right tabular-nums">{row.total}</TableCell>
      <TableCell className="text-right tabular-nums">
        {row.criticalCnt > 0 ? <Badge variant="destructive">{row.criticalCnt}</Badge> : '—'}
      </TableCell>
      <TableCell className="text-right tabular-nums">
        {row.majorCnt > 0 ? <Badge variant="destructive">{row.majorCnt}</Badge> : '—'}
      </TableCell>
      <TableCell className="text-right tabular-nums">
        {row.minorCnt > 0 ? <Badge variant="warning">{row.minorCnt}</Badge> : '—'}
      </TableCell>
      <TableCell className="text-right tabular-nums">
        {row.warningCnt > 0 ? <Badge variant="warning">{row.warningCnt}</Badge> : '—'}
      </TableCell>
    </TableRow>
  )
}

export default AlarmLibraryPage
