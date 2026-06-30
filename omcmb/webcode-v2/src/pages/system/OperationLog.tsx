import { useMemo, useState } from 'react'
import { RefreshCcw, Search, X } from 'lucide-react'

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

import { useOperationLogs } from '@core/hooks/api/useLogs'
import type { OperationResult, OperationType } from '@core/types/system'

// ============================================================
// 系统管理 / 操作日志 — 对齐 v1 webcode/src/pages/log/OperationLog
// 真实数据 useOperationLogs（adminApi.getOperationLogs → /admin/audit-logs）
// ============================================================

const PAGE_SIZE = 20
type ResultFilter = 'all' | OperationResult

const RESULT_META: Record<
  OperationResult,
  { label: string; variant: 'success' | 'destructive' | 'warning' }
> = {
  success: { label: '成功', variant: 'success' },
  failure: { label: '失败', variant: 'destructive' },
  partial: { label: '部分成功', variant: 'warning' },
}

const OP_TYPE_LABEL: Record<string, string> = {
  create: '创建',
  update: '更新',
  delete: '删除',
  query: '查询',
  export: '导出',
  import: '导入',
  login: '登录',
  logout: '登出',
  execute: '执行',
  deploy: '部署',
  approve: '审批',
}

export default function OperationLog() {
  const [page, setPage] = useState(1)
  const [operator, setOperator] = useState('')
  const [keyword, setKeyword] = useState('')
  const [result, setResult] = useState<ResultFilter>('all')
  const [selectedLogId, setSelectedLogId] = useState<string | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(operator.trim() ? { operator: operator.trim() } : {}),
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(result !== 'all' ? { result } : {}),
    }),
    [page, operator, keyword, result]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useOperationLogs(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const selectedLog = rows.find((log) => log.id === selectedLogId) ?? null

  const hasFilter =
    Boolean(operator.trim()) || Boolean(keyword.trim()) || result !== 'all'

  function resetFilters() {
    setOperator('')
    setKeyword('')
    setResult('all')
    setPage(1)
  }

  const cols = ['时间', '操作人', '客户端 IP', '模块', '操作', '对象', '结果', '说明']

  return (
    <PageShell
      title="审计日志"
      description={`操作日志 / 安全日志 / 系统日志共用同一数据源 · 共 ${total} 条`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-56 pl-9"
              placeholder="关键字（对象 / 内容）"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <Input
            className="w-44"
            placeholder="操作人"
            value={operator}
            onChange={(e) => {
              setOperator(e.target.value)
              setPage(1)
            }}
          />
          <Select
            value={result}
            onValueChange={(v) => {
              setResult(v as ResultFilter)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="结果" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部结果</SelectItem>
              <SelectItem value="success">成功</SelectItem>
              <SelectItem value="failure">失败</SelectItem>
              <SelectItem value="partial">部分成功</SelectItem>
            </SelectContent>
          </Select>
          {hasFilter && (
            <Button variant="ghost" size="sm" onClick={resetFilters}>
              <X className="size-4" /> 重置
            </Button>
          )}
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
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
              <EmptyRow colSpan={cols.length}>
                {hasFilter ? '没有匹配的日志' : '暂无操作日志'}
              </EmptyRow>
            ) : (
              rows.map((log) => {
                const rm = RESULT_META[log.result] ?? RESULT_META.failure
                const opType = log.operationType as OperationType
                return (
                    <TableRow
                      key={log.id}
                      className="cursor-pointer"
                      onClick={() => setSelectedLogId(log.id)}
                    >
                    <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                      {formatTime(log.operationTime)}
                    </TableCell>
                    <TableCell className="text-sm font-medium">
                      {log.operator || '—'}
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      {log.clientIp || '—'}
                    </TableCell>
                    <TableCell className="text-xs">
                      {log.module || log.logName || '—'}
                    </TableCell>
                    <TableCell className="text-xs">
                      <Badge variant="outline">
                        {OP_TYPE_LABEL[opType] ?? opType ?? '—'}
                      </Badge>
                    </TableCell>
                    <TableCell
                      className="max-w-[180px] truncate text-xs text-muted-foreground"
                      title={log.target}
                    >
                      {log.target || '—'}
                    </TableCell>
                    <TableCell>
                      <Badge variant={rm.variant}>{rm.label}</Badge>
                    </TableCell>
                    <TableCell
                      className="max-w-[220px] truncate text-xs text-muted-foreground"
                      title={log.message || log.content || log.detail}
                    >
                      {log.message || log.content || log.detail || '—'}
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <div className="glass-strong rounded-md border border-border/40 p-4">
        <div className="mb-3 flex items-center justify-between gap-3">
          <div>
            <div className="text-sm font-semibold text-foreground">详情</div>
            <div className="text-xs text-muted-foreground">
              点击上方任一记录查看完整字段；若后端未返回 details，则展示摘要信息。
            </div>
          </div>
          {selectedLog ? (
            <button
              type="button"
              className="rounded-md border border-border px-3 py-1 text-xs text-muted-foreground transition-colors hover:bg-muted"
              onClick={() => setSelectedLogId(null)}
            >
              关闭
            </button>
          ) : null}
        </div>

        {selectedLog ? (
          <div className="grid gap-3 md:grid-cols-2">
            <div className="space-y-2 text-sm">
              <div><span className="text-muted-foreground">操作人：</span>{selectedLog.operator || '—'}</div>
              <div><span className="text-muted-foreground">IP：</span><span className="font-mono">{selectedLog.clientIp || '—'}</span></div>
              <div><span className="text-muted-foreground">模块：</span>{selectedLog.module || '—'}</div>
              <div><span className="text-muted-foreground">类型：</span>{selectedLog.operationType || '—'}</div>
              <div><span className="text-muted-foreground">目标：</span>{selectedLog.target || '—'}</div>
              <div><span className="text-muted-foreground">结果：</span>{selectedLog.result || '—'}</div>
              <div><span className="text-muted-foreground">时间：</span>{formatTime(selectedLog.operationTime)}</div>
            </div>
            <div className="space-y-2 text-sm">
              <div className="text-muted-foreground">内容</div>
              <pre className="max-h-60 overflow-auto rounded-md bg-muted/40 p-3 text-xs leading-5 text-foreground whitespace-pre-wrap break-all">
                {selectedLog.content || selectedLog.detail || selectedLog.message || '暂无详情内容'}
              </pre>
            </div>
          </div>
        ) : (
          <div className="text-sm text-muted-foreground">当前没有选中记录。</div>
        )}
      </div>

      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={PAGE_SIZE}
        onChange={setPage}
      />
    </PageShell>
  )
}
