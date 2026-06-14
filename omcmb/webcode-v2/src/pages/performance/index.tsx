import { useMemo, useState } from 'react'
import {
  Activity,
  AlertTriangle,
  Gauge,
  ListChecks,
  RefreshCcw,
  Search,
  Trash2,
} from 'lucide-react'

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
  formatTime,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useKPIList,
  useThresholds,
  useCounters,
  usePerformanceTasks,
  useUpdateThreshold,
  useDeleteThresholds,
} from '@core/hooks/api/usePerformance'
import type { PageRequest } from '@core/types/pagination'

// ============================================================
// 性能管理 (PM/KPI) — 多页签业务页
//   KPI 定义库 / 阈值配置 / 计数器 / 采集任务
// 与 v1 (webcode) 子菜单（PmDashboard 任务仪表盘 / KPIQuery 指标查询 / ThresholdConfig
// / PerformanceTaskConfig / PmAdhoc 向导）的差距见文件末注释 & notes。
// ============================================================

type TabKey = 'kpis' | 'thresholds' | 'counters' | 'tasks'

const TABS: { key: TabKey; label: string; icon: typeof Gauge }[] = [
  { key: 'kpis', label: 'KPI 定义库', icon: Gauge },
  { key: 'thresholds', label: '阈值配置', icon: AlertTriangle },
  { key: 'counters', label: '计数器记录', icon: Activity },
  { key: 'tasks', label: '采集任务', icon: ListChecks },
]

const PAGE_SIZE = 20

// ---- 小统计卡（与 devices 页同款，本目录内自带，不动共享组件） ----
function Stat({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: number | string
  tone?: 'default' | 'emerald' | 'amber' | 'muted'
}) {
  const toneClass = {
    default: 'text-foreground',
    emerald: 'text-emerald-600 dark:text-emerald-400',
    amber: 'text-amber-600 dark:text-amber-400',
    muted: 'text-muted-foreground',
  }[tone]
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>{value}</div>
    </div>
  )
}

// ============================================================
// 主组件
// ============================================================
export function PerformancePage() {
  const [tab, setTab] = useState<TabKey>('kpis')

  return (
    <PageShell
      title="性能管理"
      description="KPI 定义库 · 阈值配置 · 计数器记录 · 采集任务"
      toolbar={
        <div className="flex flex-wrap items-center gap-1.5">
          {TABS.map((t) => {
            const Icon = t.icon
            const active = t.key === tab
            return (
              <Button
                key={t.key}
                size="sm"
                variant={active ? 'default' : 'outline'}
                onClick={() => setTab(t.key)}
              >
                <Icon className="size-4" />
                {t.label}
              </Button>
            )
          })}
        </div>
      }
    >
      {tab === 'kpis' && <KpiTab />}
      {tab === 'thresholds' && <ThresholdTab />}
      {tab === 'counters' && <CounterTab />}
      {tab === 'tasks' && <TaskTab />}
    </PageShell>
  )
}

// ============================================================
// 页签 1：KPI 定义库
// ============================================================
function KpiTab() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')

  const params = useMemo<{ keyword?: string } & PageRequest>(
    () => ({ page, pageSize: PAGE_SIZE, ...(keyword.trim() ? { keyword: keyword.trim() } : {}) }),
    [page, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useKPIList(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const categoryCount = useMemo(() => new Set(rows.map((r) => r.category)).size, [rows])

  const cols = ['编码', '名称', '类别', '单位', '描述']

  return (
    <div>
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="本页 KPI" value={rows.length} />
        <Stat label="KPI 总数" value={total} tone="emerald" />
        <Stat label="本页类别" value={categoryCount} tone="amber" />
        <Stat label="当前页" value={`${page}/${totalPages}`} tone="muted" />
      </div>

      <Toolbar
        keyword={keyword}
        onKeyword={(v) => {
          setKeyword(v)
          setPage(1)
        }}
        placeholder="KPI 名称 / 编码"
        isFetching={isFetching}
        onRefresh={() => void refetch()}
      />

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
              <EmptyRow colSpan={cols.length}>暂无 KPI 定义</EmptyRow>
            ) : (
              rows.map((k) => (
                <TableRow key={k.id}>
                  <TableCell className="font-mono text-xs">{k.kpiCode}</TableCell>
                  <TableCell className="font-medium">{k.kpiName}</TableCell>
                  <TableCell>
                    {k.category ? <Badge variant="outline">{k.category}</Badge> : '—'}
                  </TableCell>
                  <TableCell className="text-xs">{k.unit || '—'}</TableCell>
                  <TableCell className="max-w-[420px] truncate text-xs text-muted-foreground">
                    {k.description || '—'}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </div>
  )
}

// ============================================================
// 页签 2：阈值配置（含 启用/停用 切换 + 删除 操作）
// ============================================================
const OPERATOR_LABEL: Record<string, string> = {
  gt: '>',
  lt: '<',
  gte: '≥',
  lte: '≤',
  eq: '=',
  ne: '≠',
}

function ThresholdTab() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [pendingId, setPendingId] = useState<string | null>(null)

  const params = useMemo<PageRequest>(() => ({ page, pageSize: PAGE_SIZE }), [page])
  const { data, isLoading, isError, error, isFetching, refetch } = useThresholds(params)
  const updateThreshold = useUpdateThreshold()
  const deleteThresholds = useDeleteThresholds()

  const allRows = data?.items ?? []
  const rows = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return allRows
    return allRows.filter(
      (r) =>
        r.thresholdName.toLowerCase().includes(kw) || r.kpiCode.toLowerCase().includes(kw)
    )
  }, [allRows, keyword])

  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const enabledCount = allRows.filter((r) => r.enabled).length

  const handleToggle = (id: string, next: boolean) => {
    setPendingId(id)
    updateThreshold.mutate(
      { id, data: { enabled: next } },
      { onSettled: () => setPendingId(null) }
    )
  }

  const handleDelete = (id: string) => {
    setPendingId(id)
    deleteThresholds.mutate([id], { onSettled: () => setPendingId(null) })
  }

  const cols = ['名称', 'KPI 编码', '比较', '告警阈值', '严重阈值', '状态', '更新时间', '操作']

  return (
    <div>
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="阈值规则" value={total} />
        <Stat label="已启用" value={enabledCount} tone="emerald" />
        <Stat label="已停用" value={Math.max(0, allRows.length - enabledCount)} tone="muted" />
        <Stat label="当前页" value={`${page}/${totalPages}`} tone="muted" />
      </div>

      <Toolbar
        keyword={keyword}
        onKeyword={setKeyword}
        placeholder="阈值名称 / KPI 编码"
        isFetching={isFetching || updateThreshold.isPending || deleteThresholds.isPending}
        onRefresh={() => void refetch()}
      />

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
              <EmptyRow colSpan={cols.length}>暂无阈值规则</EmptyRow>
            ) : (
              rows.map((r) => {
                const busy = pendingId === r.id
                return (
                  <TableRow key={r.id} className={cn(busy && 'opacity-50')}>
                    <TableCell className="font-medium">{r.thresholdName}</TableCell>
                    <TableCell className="font-mono text-xs">{r.kpiCode}</TableCell>
                    <TableCell className="font-mono text-xs">
                      {OPERATOR_LABEL[r.operator] ?? r.operator}
                    </TableCell>
                    <TableCell className="tabular-nums">
                      {r.warningValue}
                      {r.unit}
                    </TableCell>
                    <TableCell className="tabular-nums">
                      {r.criticalValue}
                      {r.unit}
                    </TableCell>
                    <TableCell>
                      <Badge variant={r.enabled ? 'success' : 'muted'}>
                        {r.enabled ? '启用' : '停用'}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(r.updateTime)}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1.5">
                        <Button
                          size="sm"
                          variant="outline"
                          disabled={busy}
                          onClick={() => handleToggle(r.id, !r.enabled)}
                        >
                          {r.enabled ? '停用' : '启用'}
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          disabled={busy}
                          className="text-destructive hover:text-destructive"
                          onClick={() => handleDelete(r.id)}
                          aria-label="删除阈值"
                        >
                          <Trash2 className="size-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </div>
  )
}

// ============================================================
// 页签 3：计数器记录（PM 原始计数器）
// ============================================================
function CounterTab() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')

  const params = useMemo<PageRequest>(() => ({ page, pageSize: PAGE_SIZE }), [page])
  const { data, isLoading, isError, error, isFetching, refetch } = useCounters(params)

  const allRows = data?.items ?? []
  const rows = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return allRows
    return allRows.filter(
      (r) =>
        r.kpiCode.toLowerCase().includes(kw) ||
        (r.deviceSn ?? '').toLowerCase().includes(kw)
    )
  }, [allRows, keyword])

  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['计数器', '设备', '取值', '粒度', '采集时间']

  return (
    <div>
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="本页记录" value={rows.length} />
        <Stat label="记录总数" value={total} tone="emerald" />
        <Stat label="当前页" value={`${page}/${totalPages}`} tone="muted" />
        <Stat label="每页" value={PAGE_SIZE} tone="muted" />
      </div>

      <Toolbar
        keyword={keyword}
        onKeyword={setKeyword}
        placeholder="计数器名 / 设备 SN（本页过滤）"
        isFetching={isFetching}
        onRefresh={() => void refetch()}
      />

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
              <EmptyRow colSpan={cols.length}>暂无计数器记录</EmptyRow>
            ) : (
              rows.map((m) => (
                <TableRow key={m.id}>
                  <TableCell className="font-mono text-xs">{m.kpiCode || '—'}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {m.deviceName || m.deviceSn || '—'}
                  </TableCell>
                  <TableCell className="tabular-nums">
                    {m.value}
                    {m.unit ? <span className="ml-0.5 text-muted-foreground">{m.unit}</span> : null}
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{m.granularity}</Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(m.timestamp)}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </div>
  )
}

// ============================================================
// 页签 4：采集任务
// ============================================================
const TASK_STATUS_META: Record<
  string,
  { label: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'muted' }
> = {
  pending: { label: '等待中', variant: 'muted' },
  running: { label: '执行中', variant: 'warning' },
  success: { label: '成功', variant: 'success' },
  failed: { label: '失败', variant: 'destructive' },
  cancelled: { label: '已取消', variant: 'muted' },
}

type TaskStatusFilter = '' | 'pending' | 'running' | 'success' | 'failed' | 'cancelled'

function TaskTab() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState<TaskStatusFilter>('')

  const params = useMemo<PageRequest>(() => ({ page, pageSize: PAGE_SIZE }), [page])
  const { data, isLoading, isError, error, isFetching, refetch } = usePerformanceTasks(params)

  const allRows = data?.items ?? []
  const rows = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    return allRows.filter((r) => {
      if (status && r.status !== status) return false
      if (kw && !r.taskName.toLowerCase().includes(kw)) return false
      return true
    })
  }, [allRows, keyword, status])

  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const runningCount = allRows.filter((r) => r.status === 'running').length
  const failedCount = allRows.filter((r) => r.status === 'failed').length

  const cols = ['任务名称', '类型', '设备数', '指标数', '粒度', '状态', '进度', '创建时间']

  return (
    <div>
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="任务总数" value={total} />
        <Stat label="执行中" value={runningCount} tone="amber" />
        <Stat label="失败" value={failedCount} tone={failedCount > 0 ? 'amber' : 'muted'} />
        <Stat label="当前页" value={`${page}/${totalPages}`} tone="muted" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-72 pl-9"
            placeholder="任务名称（本页过滤）"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
        <Select
          value={status || 'all'}
          onValueChange={(v) => setStatus(v === 'all' ? '' : (v as TaskStatusFilter))}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="pending">等待中</SelectItem>
            <SelectItem value="running">执行中</SelectItem>
            <SelectItem value="success">成功</SelectItem>
            <SelectItem value="failed">失败</SelectItem>
            <SelectItem value="cancelled">已取消</SelectItem>
          </SelectContent>
        </Select>
        <Button
          variant="outline"
          size="sm"
          className="ml-auto"
          disabled={isFetching}
          onClick={() => void refetch()}
        >
          <RefreshCcw className="size-4" /> 刷新
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
              <EmptyRow colSpan={cols.length}>暂无采集任务</EmptyRow>
            ) : (
              rows.map((t) => {
                const meta = TASK_STATUS_META[t.status] ?? {
                  label: t.status,
                  variant: 'outline' as const,
                }
                return (
                  <TableRow key={t.id}>
                    <TableCell className="font-medium">{t.taskName}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{t.taskType}</Badge>
                    </TableCell>
                    <TableCell className="tabular-nums">{t.deviceSns.length}</TableCell>
                    <TableCell className="tabular-nums">{t.kpiCodes.length}</TableCell>
                    <TableCell className="text-xs">{t.granularity || '—'}</TableCell>
                    <TableCell>
                      <Badge variant={meta.variant}>{meta.label}</Badge>
                    </TableCell>
                    <TableCell className="w-40">
                      <div className="flex items-center gap-2">
                        <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
                          <div
                            className="h-full rounded-full bg-primary transition-all"
                            style={{ width: `${Math.min(100, Math.max(0, t.progress))}%` }}
                          />
                        </div>
                        <span className="w-9 text-right text-xs tabular-nums text-muted-foreground">
                          {Math.round(t.progress)}%
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(t.createdAt)}
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </div>
  )
}

// ============================================================
// 复用工具条（搜索框 + 刷新）
// ============================================================
function Toolbar({
  keyword,
  onKeyword,
  placeholder,
  isFetching,
  onRefresh,
}: {
  keyword: string
  onKeyword: (v: string) => void
  placeholder: string
  isFetching?: boolean
  onRefresh: () => void
}) {
  return (
    <div className="mb-3 flex flex-wrap items-center gap-2">
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          className="w-72 pl-9"
          placeholder={placeholder}
          value={keyword}
          onChange={(e) => onKeyword(e.target.value)}
        />
      </div>
      <Button
        variant="outline"
        size="sm"
        className="ml-auto"
        disabled={isFetching}
        onClick={onRefresh}
      >
        <RefreshCcw className="size-4" /> 刷新
      </Button>
    </div>
  )
}
