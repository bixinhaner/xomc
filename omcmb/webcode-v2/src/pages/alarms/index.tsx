import { useCallback, useMemo, useState } from 'react'
import {
  Check,
  Eraser,
  Eye,
  MinusCircle,
  RefreshCcw,
  Search,
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
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useAcknowledgeAlarms,
  useAcknowledgeHistoryAlarms,
  useAlarmCount,
  useClearAlarms,
  useCurrentAlarms,
  useDeleteHistoryAlarms,
  useHistoricalAlarms,
  useMarkAlarmRead,
  useUnacknowledgeAlarms,
  useUnacknowledgeHistoryAlarms,
} from '@core/hooks/api/useAlarms'
import type {
  Alarm,
  AlarmFilter,
  DealState,
  EventType,
} from '@core/types/alarm'
import type { AlarmSeverity } from '@core/types/common'

import { AlarmDetailDrawer } from './AlarmDetailDrawer'
import { NotePromptDialog } from './NotePromptDialog'

// ---------------------------------------------------------------------------
// 展示常量
// ---------------------------------------------------------------------------

const SEVERITY_LABEL: Record<AlarmSeverity, string> = {
  critical: '紧急',
  major: '重要',
  minor: '次要',
  warning: '警告',
}
const SEVERITY_VARIANT: Record<
  AlarmSeverity,
  'destructive' | 'warning' | 'default' | 'muted'
> = {
  critical: 'destructive',
  major: 'destructive',
  minor: 'warning',
  warning: 'warning',
}

const DEAL_STATE_LABEL: Record<DealState, string> = {
  '0': '未确认未清除',
  '1': '已确认未清除',
  '2': '未确认已清除',
  '3': '已确认已清除',
}
const DEAL_STATE_CLASS: Record<DealState, string> = {
  '0': 'text-destructive',
  '1': 'text-amber-600 dark:text-amber-400',
  '2': 'text-sky-600 dark:text-sky-400',
  '3': 'text-emerald-600 dark:text-emerald-400',
}

const EVENT_TYPE_LABEL: Record<EventType, string> = {
  communication: '通信告警',
  qualityOfService: '服务质量告警',
  processingError: '处理出错告警',
  device: '设备告警',
  environment: '环境告警',
  performance: '业务质量告警',
}

// 告警源（网元类型）下拉
const NE_TYPE_OPTIONS = ['ENB', 'GNB', 'CPE', 'UPS', 'WCG', 'GSM']

// 快捷筛选 pill
type QuickFilterKey =
  | 'all'
  | 'critical'
  | 'major'
  | 'minor'
  | 'warning'
  | 'unacked'
  | 'unread'

const QUICK_FILTERS: { key: QuickFilterKey; label: string }[] = [
  { key: 'all', label: '全部' },
  { key: 'critical', label: '紧急' },
  { key: 'major', label: '重要' },
  { key: 'minor', label: '次要' },
  { key: 'warning', label: '警告' },
  { key: 'unacked', label: '未确认' },
  { key: 'unread', label: '未读' },
]

type AlarmTab = 'active' | 'history'

// ---------------------------------------------------------------------------
// 子组件
// ---------------------------------------------------------------------------

function Stat({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: number
  tone?: 'default' | 'critical' | 'major' | 'minor' | 'warning' | 'amber'
}) {
  const toneClass = {
    default: 'text-foreground',
    critical: 'text-destructive',
    major: 'text-orange-600 dark:text-orange-400',
    minor: 'text-yellow-600 dark:text-yellow-400',
    warning: 'text-sky-600 dark:text-sky-400',
    amber: 'text-amber-600 dark:text-amber-400',
  }[tone]

  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>
        {value}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// 主页面
// ---------------------------------------------------------------------------

export function AlarmsPage() {
  const [tab, setTab] = useState<AlarmTab>('active')
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)

  // 筛选
  const [keyword, setKeyword] = useState('')
  const [severity, setSeverity] = useState<AlarmSeverity | ''>('')
  const [eventType, setEventType] = useState<EventType | ''>('')
  const [dealState, setDealState] = useState<'' | '0' | '1'>('')
  const [neType, setNeType] = useState('')
  const [unread, setUnread] = useState<'' | '0' | '1'>('')
  const [quick, setQuick] = useState<QuickFilterKey>('all')

  // 选择 & 弹窗
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [detailAlarm, setDetailAlarm] = useState<Alarm | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)

  const [ackOpen, setAckOpen] = useState(false)
  const [ackIds, setAckIds] = useState<string[]>([])
  const [clearOpen, setClearOpen] = useState(false)
  const [clearIds, setClearIds] = useState<string[]>([])

  const isActive = tab === 'active'

  // 组装查询参数
  const params = useMemo<AlarmFilter & { page: number; pageSize: number }>(() => {
    const f: AlarmFilter & { page: number; pageSize: number } = { page, pageSize }
    if (keyword.trim()) f.keyword = keyword.trim()
    if (severity) f.severity = severity
    if (eventType) f.eventType = eventType
    if (dealState) f.dealState = dealState
    if (neType) f.neType = neType
    if (unread) f.unread = unread
    return f
  }, [page, pageSize, keyword, severity, eventType, dealState, neType, unread])

  // 活动 / 历史 两条查询，按 tab 启用对应轮询
  const activeQuery = useCurrentAlarms(params, {
    refetchIntervalMs: isActive ? 30000 : false,
  })
  const historyQuery = useHistoricalAlarms(params, {
    refetchIntervalMs: false,
  })
  const query = isActive ? activeQuery : historyQuery

  const { data, isLoading, isError, error, isFetching, refetch } = query
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  // 统计（活动告警来自概览接口；历史 tab 退化为列表内分布）
  const { data: alarmCount } = useAlarmCount()
  const stats = useMemo(() => {
    if (isActive) {
      return {
        total: alarmCount?.total_active ?? total,
        critical: alarmCount?.critical ?? 0,
        major: alarmCount?.major ?? 0,
        minor: alarmCount?.minor ?? 0,
        warning: alarmCount?.warning ?? 0,
        unacked: alarmCount?.unacknowledged ?? 0,
        unread: alarmCount?.unread ?? 0,
      }
    }
    // 历史 tab：用当前页 rows 的分布兜底
    const count = (sev: AlarmSeverity) =>
      rows.filter((r) => r.severity === sev).length
    return {
      total,
      critical: count('critical'),
      major: count('major'),
      minor: count('minor'),
      warning: count('warning'),
      unacked: rows.filter((r) => r.dealState === '0' || r.dealState === '2')
        .length,
      unread: rows.filter((r) => r.unread === '1').length,
    }
  }, [isActive, alarmCount, total, rows])

  // ----------------------- mutations -----------------------
  const ackActive = useAcknowledgeAlarms()
  const unackActive = useUnacknowledgeAlarms()
  const clearActive = useClearAlarms()
  const markRead = useMarkAlarmRead()
  const ackHistory = useAcknowledgeHistoryAlarms()
  const unackHistory = useUnacknowledgeHistoryAlarms()
  const deleteHistory = useDeleteHistoryAlarms()

  const ackBusy = ackActive.isPending || ackHistory.isPending
  const clearBusy = clearActive.isPending

  // ----------------------- 交互 -----------------------
  const resetSelection = useCallback(() => setSelected(new Set()), [])

  const toggleRow = useCallback((id: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }, [])

  const toggleAll = useCallback(() => {
    setSelected((prev) => {
      if (prev.size === rows.length && rows.length > 0) return new Set()
      return new Set(rows.map((r) => r.id))
    })
  }, [rows])

  const switchTab = useCallback((next: AlarmTab) => {
    setTab(next)
    setPage(1)
    setSelected(new Set())
    setQuick('all')
  }, [])

  const applyQuick = useCallback((key: QuickFilterKey) => {
    setQuick(key)
    setPage(1)
    // 先清掉受快捷筛选影响的维度
    setSeverity('')
    setUnread('')
    if (key === 'critical' || key === 'major' || key === 'minor' || key === 'warning') {
      setSeverity(key)
    } else if (key === 'unacked') {
      setDealState('0')
    } else if (key === 'unread') {
      setUnread('1')
    } else {
      setDealState('')
    }
  }, [])

  const openAck = useCallback((ids: string[]) => {
    if (ids.length === 0) return
    setAckIds(ids)
    setAckOpen(true)
  }, [])

  const confirmAck = useCallback(
    async (note: string) => {
      try {
        if (isActive) {
          await ackActive.mutateAsync({ ids: ackIds, note })
        } else {
          await ackHistory.mutateAsync({ ids: ackIds, note })
        }
        resetSelection()
        setAckOpen(false)
        void refetch()
      } catch {
        setAckOpen(false)
      }
    },
    [isActive, ackActive, ackHistory, ackIds, resetSelection, refetch]
  )

  const doUnack = useCallback(
    async (ids: string[]) => {
      if (ids.length === 0) return
      try {
        if (isActive) await unackActive.mutateAsync(ids)
        else await unackHistory.mutateAsync(ids)
        resetSelection()
        void refetch()
      } catch {
        /* hook 内部已 invalidate；失败静默，列表保持原状 */
      }
    },
    [isActive, unackActive, unackHistory, resetSelection, refetch]
  )

  const openClear = useCallback((ids: string[]) => {
    if (ids.length === 0) return
    setClearIds(ids)
    setClearOpen(true)
  }, [])

  const confirmClear = useCallback(
    async (note: string) => {
      try {
        await clearActive.mutateAsync({ ids: clearIds, note })
        resetSelection()
        setClearOpen(false)
        void refetch()
      } catch {
        setClearOpen(false)
      }
    },
    [clearActive, clearIds, resetSelection, refetch]
  )

  const doMarkRead = useCallback(
    async (ids: string[]) => {
      if (ids.length === 0) return
      try {
        await Promise.all(ids.map((id) => markRead.mutateAsync(id)))
        resetSelection()
        void refetch()
      } catch {
        /* 静默 */
      }
    },
    [markRead, resetSelection, refetch]
  )

  const doDeleteHistory = useCallback(
    async (ids: string[]) => {
      if (ids.length === 0) return
      try {
        await deleteHistory.mutateAsync(ids)
        resetSelection()
        void refetch()
      } catch {
        /* 静默 */
      }
    },
    [deleteHistory, resetSelection, refetch]
  )

  const openDetail = useCallback((a: Alarm) => {
    setDetailAlarm(a)
    setDetailOpen(true)
  }, [])

  const selectedIds = useMemo(() => [...selected], [selected])
  const allChecked = rows.length > 0 && selected.size === rows.length

  const cols = [
    '',
    'SN',
    '告警源',
    '严重度',
    '告警标识',
    '可能原因',
    '事件类型',
    '处理状态',
    '故障时间',
    '次数',
    '操作',
  ]

  return (
    <PageShell
      title="告警管理"
      description={
        isActive
          ? '当前活动告警 · 每 30 秒自动刷新'
          : '历史告警查询'
      }
      isFetching={isFetching}
      toolbar={
        <>
          {/* Tab 切换 */}
          <div className="inline-flex rounded-md border p-0.5">
            <button
              type="button"
              className={cn(
                'rounded px-3 py-1 text-sm transition-colors',
                isActive
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:text-foreground'
              )}
              onClick={() => switchTab('active')}
            >
              当前告警
            </button>
            <button
              type="button"
              className={cn(
                'rounded px-3 py-1 text-sm transition-colors',
                !isActive
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:text-foreground'
              )}
              onClick={() => switchTab('history')}
            >
              历史告警
            </button>
          </div>

          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-64 pl-9"
              placeholder="设备 SN / 告警标识 / 关键字"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>

          <Select
            value={severity || 'all'}
            onValueChange={(v) => {
              setSeverity(v === 'all' ? '' : (v as AlarmSeverity))
              setQuick('all')
              setPage(1)
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="严重度" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部严重度</SelectItem>
              <SelectItem value="critical">紧急</SelectItem>
              <SelectItem value="major">重要</SelectItem>
              <SelectItem value="minor">次要</SelectItem>
              <SelectItem value="warning">警告</SelectItem>
            </SelectContent>
          </Select>

          <Select
            value={eventType || 'all'}
            onValueChange={(v) => {
              setEventType(v === 'all' ? '' : (v as EventType))
              setPage(1)
            }}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="事件类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部事件类型</SelectItem>
              <SelectItem value="communication">通信告警</SelectItem>
              <SelectItem value="qualityOfService">服务质量告警</SelectItem>
              <SelectItem value="processingError">处理出错告警</SelectItem>
              <SelectItem value="device">设备告警</SelectItem>
              <SelectItem value="environment">环境告警</SelectItem>
            </SelectContent>
          </Select>

          <Select
            value={neType || 'all'}
            onValueChange={(v) => {
              setNeType(v === 'all' ? '' : v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-28">
              <SelectValue placeholder="告警源" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部告警源</SelectItem>
              {NE_TYPE_OPTIONS.map((o) => (
                <SelectItem key={o} value={o}>
                  {o}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {isActive ? (
            <Select
              value={dealState || 'all'}
              onValueChange={(v) => {
                setDealState(v === 'all' ? '' : (v as '0' | '1'))
                setQuick('all')
                setPage(1)
              }}
            >
              <SelectTrigger className="w-36">
                <SelectValue placeholder="处理状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部状态</SelectItem>
                <SelectItem value="0">未确认未清除</SelectItem>
                <SelectItem value="1">已确认未清除</SelectItem>
              </SelectContent>
            </Select>
          ) : null}

          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={() => refetch()}
          >
            <RefreshCcw /> 刷新
          </Button>
        </>
      }
    >
      {/* 统计卡片 */}
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4 lg:grid-cols-7">
        <Stat label="总计" value={stats.total} />
        <Stat label="紧急" value={stats.critical} tone="critical" />
        <Stat label="重要" value={stats.major} tone="major" />
        <Stat label="次要" value={stats.minor} tone="minor" />
        <Stat label="警告" value={stats.warning} tone="warning" />
        <Stat label="未确认" value={stats.unacked} tone="amber" />
        <Stat label="未读" value={stats.unread} tone="amber" />
      </div>

      {/* 快捷筛选 + 批量操作 */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="flex flex-wrap items-center gap-1">
          {QUICK_FILTERS.map((q) => (
            <button
              key={q.key}
              type="button"
              onClick={() => applyQuick(q.key)}
              className={cn(
                'rounded-full border px-3 py-1 text-xs transition-colors',
                quick === q.key
                  ? 'border-primary bg-primary/10 text-primary'
                  : 'border-transparent bg-muted text-muted-foreground hover:text-foreground'
              )}
            >
              {q.label}
            </button>
          ))}
        </div>

        <div className="ml-auto flex items-center gap-2">
          {selectedIds.length > 0 ? (
            <span className="text-xs text-muted-foreground">
              已选 {selectedIds.length} 项
            </span>
          ) : null}
          {isActive ? (
            <>
              <Button
                variant="outline"
                size="sm"
                disabled={selectedIds.length === 0 || ackBusy}
                onClick={() => openAck(selectedIds)}
              >
                <Check /> 确认
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={selectedIds.length === 0}
                onClick={() => doUnack(selectedIds)}
              >
                <MinusCircle /> 反确认
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={selectedIds.length === 0 || clearBusy}
                onClick={() => openClear(selectedIds)}
              >
                <Eraser /> 清除
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={selectedIds.length === 0}
                onClick={() => doMarkRead(selectedIds)}
              >
                <Eye /> 标记已读
              </Button>
            </>
          ) : (
            <>
              <Button
                variant="outline"
                size="sm"
                disabled={selectedIds.length === 0 || ackBusy}
                onClick={() => openAck(selectedIds)}
              >
                <Check /> 确认
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={selectedIds.length === 0}
                onClick={() => doUnack(selectedIds)}
              >
                <MinusCircle /> 反确认
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={selectedIds.length === 0 || deleteHistory.isPending}
                onClick={() => doDeleteHistory(selectedIds)}
              >
                <Eraser /> 删除
              </Button>
            </>
          )}
        </div>
      </div>

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c, i) =>
                i === 0 ? (
                  <TableHead key="select" className="w-10">
                    <input
                      type="checkbox"
                      className="size-4 cursor-pointer"
                      checked={allChecked}
                      onChange={toggleAll}
                      aria-label="全选"
                    />
                  </TableHead>
                ) : (
                  <TableHead key={c}>{c}</TableHead>
                )
              )}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>
                {isActive ? '暂无活动告警' : '暂无历史告警'}
              </EmptyRow>
            ) : (
              rows.map((a) => {
                const confirmed = a.dealState === '1' || a.dealState === '3'
                return (
                  <TableRow key={a.id} className="align-top">
                    <TableCell>
                      <input
                        type="checkbox"
                        className="size-4 cursor-pointer"
                        checked={selected.has(a.id)}
                        onChange={() => toggleRow(a.id)}
                        aria-label="选择行"
                      />
                    </TableCell>
                    <TableCell>
                      <button
                        type="button"
                        className="flex items-center gap-1.5 font-mono text-xs hover:underline"
                        onClick={() => openDetail(a)}
                      >
                        {a.unread === '1' ? (
                          <span className="inline-block size-1.5 rounded-full bg-destructive" />
                        ) : null}
                        {a.deviceSn}
                      </button>
                      {a.deviceName && a.deviceName !== a.deviceSn ? (
                        <div className="text-xs text-muted-foreground">
                          {a.deviceName}
                        </div>
                      ) : null}
                    </TableCell>
                    <TableCell className="text-xs">{a.neType || '—'}</TableCell>
                    <TableCell>
                      <Badge variant={SEVERITY_VARIANT[a.severity]}>
                        {SEVERITY_LABEL[a.severity]}
                      </Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {a.alarmIdentifier || '—'}
                    </TableCell>
                    <TableCell className="max-w-[200px]">
                      <div className="truncate" title={a.alarmName}>
                        {a.alarmName || '—'}
                      </div>
                    </TableCell>
                    <TableCell className="text-xs">
                      {EVENT_TYPE_LABEL[a.eventType]}
                    </TableCell>
                    <TableCell>
                      <span className={cn('text-xs', DEAL_STATE_CLASS[a.dealState])}>
                        {DEAL_STATE_LABEL[a.dealState]}
                      </span>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(a.eventTime)}
                    </TableCell>
                    <TableCell className="tabular-nums">{a.alarmCount}</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 px-2 text-xs"
                          onClick={() => openDetail(a)}
                        >
                          详情
                        </Button>
                        {confirmed ? (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-7 px-2 text-xs"
                            onClick={() => doUnack([a.id])}
                          >
                            反确认
                          </Button>
                        ) : (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-7 px-2 text-xs"
                            onClick={() => openAck([a.id])}
                          >
                            确认
                          </Button>
                        )}
                        {isActive ? (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-7 px-2 text-xs text-destructive"
                            onClick={() => openClear([a.id])}
                          >
                            清除
                          </Button>
                        ) : (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-7 px-2 text-xs text-destructive"
                            onClick={() => doDeleteHistory([a.id])}
                          >
                            删除
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={pageSize}
        onChange={setPage}
      />

      <AlarmDetailDrawer
        alarm={detailAlarm}
        open={detailOpen}
        onClose={() => {
          setDetailOpen(false)
          setDetailAlarm(null)
        }}
      />

      <NotePromptDialog
        open={ackOpen}
        title="确认告警"
        description={`将确认 ${ackIds.length} 条告警`}
        confirmText="确认"
        loading={ackBusy}
        onConfirm={confirmAck}
        onCancel={() => setAckOpen(false)}
      />

      <NotePromptDialog
        open={clearOpen}
        title="清除告警"
        description={`将清除 ${clearIds.length} 条告警`}
        confirmText="清除"
        loading={clearBusy}
        onConfirm={confirmClear}
        onCancel={() => setClearOpen(false)}
      />
    </PageShell>
  )
}
