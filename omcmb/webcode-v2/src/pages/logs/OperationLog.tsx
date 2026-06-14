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
  Pagination,
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'

import { useOperationLogs } from '@core/hooks/api/useLogs'
import type { OperationLog as OperationLogItem, OperationResult, OperationType } from '@core/types/system'

import { LogNavTabs, PAGE_SIZE, SearchInput, Stat } from './_shared'

// ===========================================================================
// 操作审计（对齐 v1 log/operation）— 操作员流水，按操作类型 / 结果 / 关键字筛选
// ===========================================================================

const OP_TYPE_LABEL: Record<OperationType, string> = {
  create: '新建',
  update: '修改',
  delete: '删除',
  query: '查询',
  export: '导出',
  import: '导入',
  login: '登录',
  logout: '登出',
  execute: '执行',
  deploy: '下发',
  approve: '审批',
}

const OP_TYPE_OPTIONS: OperationType[] = [
  'login',
  'logout',
  'query',
  'create',
  'update',
  'delete',
  'execute',
  'deploy',
  'export',
  'import',
  'approve',
]

const OP_RESULT_VARIANT: Record<OperationResult, 'success' | 'destructive' | 'warning'> = {
  success: 'success',
  failure: 'destructive',
  partial: 'warning',
}

const OP_RESULT_LABEL: Record<OperationResult, string> = {
  success: '成功',
  failure: '失败',
  partial: '部分成功',
}

export default function OperationLog() {
  const [page, setPage] = useState(1)
  const [operationType, setOperationType] = useState<OperationType | ''>('')
  const [result, setResult] = useState<OperationResult | ''>('')
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(operationType ? { operationType } : {}),
      ...(result ? { result } : {}),
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, operationType, result, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useOperationLogs(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const counts = useMemo(() => {
    let ok = 0
    let fail = 0
    for (const r of rows) {
      if (r.result === 'success') ok += 1
      else fail += 1
    }
    return { ok, fail }
  }, [rows])

  const cols = ['时间', '操作员', '客户端 IP', '模块', '操作类型', '对象', '内容', '结果']

  return (
    <PageShell
      title="操作审计"
      description="操作员审计流水 · 按操作类型 / 结果 / 关键字检索"
      isFetching={isFetching}
      toolbar={<LogNavTabs />}
    >
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-3">
        <Stat label="本页总数" value={rows.length} />
        <Stat label="成功" value={counts.ok} tone="emerald" />
        <Stat label="失败 / 部分" value={counts.fail} tone="rose" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <SearchInput
          className="w-72"
          placeholder="操作员 / 内容关键字"
          value={keyword}
          onChange={(v) => {
            setKeyword(v)
            setPage(1)
          }}
        />
        <Select
          value={operationType || 'all'}
          onValueChange={(v) => {
            setOperationType(v === 'all' ? '' : (v as OperationType))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder="操作类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部类型</SelectItem>
            {OP_TYPE_OPTIONS.map((tp) => (
              <SelectItem key={tp} value={tp}>
                {OP_TYPE_LABEL[tp]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={result || 'all'}
          onValueChange={(v) => {
            setResult(v === 'all' ? '' : (v as OperationResult))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="结果" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部结果</SelectItem>
            <SelectItem value="success">成功</SelectItem>
            <SelectItem value="failure">失败</SelectItem>
            <SelectItem value="partial">部分成功</SelectItem>
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
              <EmptyRow colSpan={cols.length}>暂无操作日志</EmptyRow>
            ) : (
              rows.map((l: OperationLogItem) => (
                <TableRow key={l.id}>
                  <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                    {formatTime(l.operationTime)}
                  </TableCell>
                  <TableCell className="font-medium">{l.operator || '—'}</TableCell>
                  <TableCell className="font-mono text-xs">{l.clientIp || '—'}</TableCell>
                  <TableCell>{l.module || '—'}</TableCell>
                  <TableCell>
                    <Badge variant="muted">
                      {OP_TYPE_LABEL[l.operationType] ?? l.operationType}
                    </Badge>
                  </TableCell>
                  <TableCell className="max-w-[160px] truncate text-xs">{l.target || '—'}</TableCell>
                  <TableCell className="max-w-[260px] truncate text-xs text-muted-foreground">
                    {l.content || l.message || '—'}
                  </TableCell>
                  <TableCell>
                    <Badge variant={OP_RESULT_VARIANT[l.result] ?? 'muted'}>
                      {OP_RESULT_LABEL[l.result] ?? l.result}
                    </Badge>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </PageShell>
  )
}
