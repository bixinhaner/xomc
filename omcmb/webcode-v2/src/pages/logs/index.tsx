import { useMemo, useState } from 'react'
import { ArrowDownToLine, ArrowUpToLine, RefreshCcw, Search } from 'lucide-react'

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

import { useNEMessageLogs, useOperationLogs, useSystemLogs } from '@core/hooks/api/useLogs'
import type { SystemLog, NEMessageLog } from '@core/mock/data/logs'
import type { OperationLog, OperationResult, OperationType } from '@core/types/system'

import { LogDetailModal } from './LogDetailModal'

// ---------------------------------------------------------------------------
// 标签页：系统日志 / 操作审计 / 网元消息（对齐 v1 log 模块三大子页）
// ---------------------------------------------------------------------------

type TabKey = 'system' | 'operation' | 'ne'

const TABS: { key: TabKey; label: string }[] = [
  { key: 'system', label: '系统日志' },
  { key: 'operation', label: '操作审计' },
  { key: 'ne', label: '网元消息' },
]

const PAGE_SIZE = 20

// ---- 系统日志级别样式 ----
type LogLevel = SystemLog['level']

const LEVEL_VARIANT: Record<LogLevel, 'default' | 'warning' | 'destructive' | 'muted'> = {
  INFO: 'default',
  WARN: 'warning',
  ERROR: 'destructive',
  DEBUG: 'muted',
}

// ---- 操作类型 / 结果样式 ----
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

// ---- 网元消息类型样式 ----
const MSG_TYPE_LABEL: Record<NEMessageLog['messageType'], string> = {
  notification: '通知',
  alarm: '告警上报',
  heartbeat: '心跳',
  config_response: '配置响应',
  perf_data: '性能上报',
}

const MSG_TYPE_VARIANT: Record<
  NEMessageLog['messageType'],
  'default' | 'destructive' | 'warning' | 'muted' | 'success'
> = {
  notification: 'default',
  alarm: 'destructive',
  heartbeat: 'muted',
  config_response: 'success',
  perf_data: 'warning',
}

function Stat({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: number | string
  tone?: 'default' | 'emerald' | 'amber' | 'rose' | 'muted'
}) {
  const toneClass = {
    default: 'text-foreground',
    emerald: 'text-emerald-600 dark:text-emerald-400',
    amber: 'text-amber-600 dark:text-amber-400',
    rose: 'text-rose-600 dark:text-rose-400',
    muted: 'text-muted-foreground',
  }[tone]
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>{value}</div>
    </div>
  )
}

// ===========================================================================
// 系统日志页签
// ===========================================================================

function SystemLogsTab() {
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
    <>
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="本页总数" value={rows.length} />
        <Stat label="ERROR" value={counts.ERROR} tone="rose" />
        <Stat label="WARN" value={counts.WARN} tone="amber" />
        <Stat label="INFO" value={counts.INFO} tone="emerald" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-72 pl-9"
            placeholder="消息关键字"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <Input
          className="w-44"
          placeholder="来源 (如 OMC-Core)"
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
            <SelectItem value="INFO">INFO</SelectItem>
            <SelectItem value="WARN">WARN</SelectItem>
            <SelectItem value="ERROR">ERROR</SelectItem>
            <SelectItem value="DEBUG">DEBUG</SelectItem>
          </SelectContent>
        </Select>
        <div className="ml-auto flex items-center gap-2">
          {isFetching && <span className="text-xs text-muted-foreground">刷新中…</span>}
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
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

      <LogDetailModal
        open={detail !== null}
        title="系统日志详情"
        onClose={() => setDetail(null)}
      >
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
    </>
  )
}

// ===========================================================================
// 操作审计页签
// ===========================================================================

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

function OperationLogsTab() {
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
    <>
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-3">
        <Stat label="本页总数" value={rows.length} />
        <Stat label="成功" value={counts.ok} tone="emerald" />
        <Stat label="失败 / 部分" value={counts.fail} tone="rose" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-72 pl-9"
            placeholder="操作员 / 内容关键字"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value)
              setPage(1)
            }}
          />
        </div>
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
            {OP_TYPE_OPTIONS.map((t) => (
              <SelectItem key={t} value={t}>
                {OP_TYPE_LABEL[t]}
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
        <div className="ml-auto flex items-center gap-2">
          {isFetching && <span className="text-xs text-muted-foreground">刷新中…</span>}
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
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
              rows.map((l: OperationLog) => (
                <TableRow key={l.id}>
                  <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                    {formatTime(l.operationTime)}
                  </TableCell>
                  <TableCell className="font-medium">{l.operator || '—'}</TableCell>
                  <TableCell className="font-mono text-xs">{l.clientIp || '—'}</TableCell>
                  <TableCell>{l.module || '—'}</TableCell>
                  <TableCell>
                    <Badge variant="muted">{OP_TYPE_LABEL[l.operationType] ?? l.operationType}</Badge>
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
    </>
  )
}

// ===========================================================================
// 网元消息页签
// ===========================================================================

const MSG_TYPE_OPTIONS: NEMessageLog['messageType'][] = [
  'notification',
  'alarm',
  'heartbeat',
  'config_response',
  'perf_data',
]

function DirectionBadge({ direction }: { direction: NEMessageLog['direction'] }) {
  const isNorth = direction === 'northbound'
  return (
    <Badge variant={isNorth ? 'default' : 'muted'}>
      {isNorth ? <ArrowUpToLine /> : <ArrowDownToLine />}
      {isNorth ? '北向' : '南向'}
    </Badge>
  )
}

function NEMessageLogsTab() {
  const [page, setPage] = useState(1)
  const [deviceSn, setDeviceSn] = useState('')
  const [messageType, setMessageType] = useState<NEMessageLog['messageType'] | ''>('')
  const [detail, setDetail] = useState<NEMessageLog | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
      ...(messageType ? { messageType } : {}),
    }),
    [page, deviceSn, messageType]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useNEMessageLogs(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const counts = useMemo(() => {
    let ok = 0
    let fail = 0
    for (const r of rows) {
      if (r.success) ok += 1
      else fail += 1
    }
    return { ok, fail }
  }, [rows])

  const cols = ['时间', '设备', '消息类型', '方向', '协议', '内容', '结果', '']

  return (
    <>
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-3">
        <Stat label="本页总数" value={rows.length} />
        <Stat label="成功" value={counts.ok} tone="emerald" />
        <Stat label="失败" value={counts.fail} tone="rose" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-64 pl-9"
            placeholder="设备 SN"
            value={deviceSn}
            onChange={(e) => {
              setDeviceSn(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <Select
          value={messageType || 'all'}
          onValueChange={(v) => {
            setMessageType(v === 'all' ? '' : (v as NEMessageLog['messageType']))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-44">
            <SelectValue placeholder="消息类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部类型</SelectItem>
            {MSG_TYPE_OPTIONS.map((t) => (
              <SelectItem key={t} value={t}>
                {MSG_TYPE_LABEL[t]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <div className="ml-auto flex items-center gap-2">
          {isFetching && <span className="text-xs text-muted-foreground">刷新中…</span>}
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
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
              <EmptyRow colSpan={cols.length}>暂无网元消息</EmptyRow>
            ) : (
              rows.map((l) => (
                <TableRow key={l.id}>
                  <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                    {formatTime(l.timestamp)}
                  </TableCell>
                  <TableCell>
                    <div className="font-medium">{l.deviceName || l.deviceSn}</div>
                    <div className="font-mono text-xs text-muted-foreground">{l.deviceSn}</div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={MSG_TYPE_VARIANT[l.messageType] ?? 'default'}>
                      {MSG_TYPE_LABEL[l.messageType] ?? l.messageType}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <DirectionBadge direction={l.direction} />
                  </TableCell>
                  <TableCell className="font-mono text-xs">{l.protocol}</TableCell>
                  <TableCell className="max-w-[280px] truncate font-mono text-xs text-muted-foreground">
                    {l.content}
                  </TableCell>
                  <TableCell>
                    <Badge variant={l.success ? 'success' : 'destructive'}>
                      {l.success ? '成功' : '失败'}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <Button variant="ghost" size="sm" onClick={() => setDetail(l)}>
                      详情
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />

      <LogDetailModal open={detail !== null} title="网元消息详情" onClose={() => setDetail(null)}>
        {detail && (
          <div className="space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant={MSG_TYPE_VARIANT[detail.messageType] ?? 'default'}>
                {MSG_TYPE_LABEL[detail.messageType] ?? detail.messageType}
              </Badge>
              <DirectionBadge direction={detail.direction} />
              <span className="font-mono text-xs text-muted-foreground">{detail.protocol}</span>
              <span className="ml-auto text-xs text-muted-foreground">
                {formatTime(detail.timestamp)}
              </span>
            </div>
            <div className="text-xs text-muted-foreground">
              设备：<span className="font-mono">{detail.deviceSn}</span>
              {detail.deviceName ? ` · ${detail.deviceName}` : ''}
            </div>
            <pre className="whitespace-pre-wrap break-all rounded-md bg-muted p-3 font-mono text-xs">
              {detail.content}
            </pre>
          </div>
        )}
      </LogDetailModal>
    </>
  )
}

// ===========================================================================
// 页面外壳：标签页切换
// ===========================================================================

export function LogsPage() {
  const [tab, setTab] = useState<TabKey>('system')

  return (
    <PageShell
      title="日志中心"
      description="系统运行日志 · 操作审计流水 · 网元消息跟踪"
      toolbar={
        <div className="flex items-center gap-1 rounded-lg border bg-card p-1">
          {TABS.map((t) => (
            <Button
              key={t.key}
              variant={tab === t.key ? 'default' : 'ghost'}
              size="sm"
              onClick={() => setTab(t.key)}
            >
              {t.label}
            </Button>
          ))}
        </div>
      }
    >
      {tab === 'system' && <SystemLogsTab />}
      {tab === 'operation' && <OperationLogsTab />}
      {tab === 'ne' && <NEMessageLogsTab />}
    </PageShell>
  )
}
