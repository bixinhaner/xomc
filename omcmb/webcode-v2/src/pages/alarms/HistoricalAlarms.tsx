import { useCallback, useMemo, useState } from 'react'
import {
  Check,
  MinusCircle,
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
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useAcknowledgeHistoryAlarms,
  useDeleteHistoryAlarms,
  useHistoricalAlarms,
  useHistoryAlarmCount,
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

// 历史告警只会落在「已清除」两态（未确认已清除 / 已确认已清除），但仍兼容全部四态展示
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

const NE_TYPE_OPTIONS = ['ENB', 'GNB', 'CPE', 'UPS', 'WCG', 'GSM']

// 历史告警快捷筛选：全部 / 严重度 / 已清除态
type QuickFilterKey =
  | 'all'
  | 'critical'
  | 'major'
  | 'minor'
  | 'warning'
  | 'cleared'
  | 'confirmed'

const QUICK_FILTERS: { key: QuickFilterKey; label: string }[] = [
  { key: 'all', label: '全部' },
  { key: 'critical', label: '紧急' },
  { key: 'major', label: '重要' },
  { key: 'minor', label: '次要' },
  { key: 'warning', label: '警告' },
  { key: 'cleared', label: '已清除' },
  { key: 'confirmed', label: '已确认' },
]

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
  tone?: 'default' | 'critical' | 'major' | 'minor' | 'warning'
}) {
  const toneClass = {
    default: 'text-foreground',
    critical: 'text-destructive',
    major: 'text-orange-600 dark:text-orange-400',
    minor: 'text-yellow-600 dark:text-yellow-400',
    warning: 'text-sky-600 dark:text-sky-400',
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

export default function HistoricalAlarms() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)

  // 筛选
  const [keyword, setKeyword] = useState('')
  const [severity, setSeverity] = useState<AlarmSeverity | ''>('')
  const [eventType, setEventType] = useState<EventType | ''>('')
  const [dealState, setDealState] = useState<'' | '2' | '3'>('')
  const [neType, setNeType] = useState('')
  const [quick, setQuick] = useState<QuickFilterKey>('all')

  // 选择 & 弹窗
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [detailAlarm, setDetailAlarm] = useState<Alarm | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)
  const [ackOpen, setAckOpen] = useState(false)
  const [ackIds, setAckIds] = useState<string[]>([])

  // 组装查询参数
  const params = useMemo<AlarmFilter & { page: number; pageSize: number }>(() => {
    const f: AlarmFilter & { page: number; pageSize: number } = { page, pageSize }
    if (keyword.trim()) f.keyword = keyword.trim()
    if (severity) f.severity = severity
    if (eventType) f.eventType = eventType
    if (neType) f.neType = neType
    if (quick === 'cleared') f.dealState = ['2', '3']
    else if (quick === 'confirmed') f.dealState = ['1', '3']
    else if (dealState) f.dealState = dealState
    return f
  }, [page, pageSize, keyword, severity, eventType, neType, dealState, quick])

  const { data, isLoading, isError, error, isFetching, refetch } =
    useHistoricalAlarms(params, { refetchIntervalMs: false })

  const rawRows = useMemo(() => data?.items ?? [], [data])
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  // 未读告警排在最前
  const rows = useMemo(
    () =>
      [...rawRows].sort((a, b) => {
        if (a.unread === '1' && b.unread !== '1') return -1
        if (a.unread !== '1' && b.unread === '1') return 1
        return 0
      }),
    [rawRows]
  )

  // 统计来自后端 API
  const { data: alarmCount } = useHistoryAlarmCount()
  const stats = useMemo(
    () => ({
      total: alarmCount?.total_active ?? total,
      critical: alarmCount?.critical ?? 0,
      major: alarmCount?.major ?? 0,
      minor: alarmCount?.minor ?? 0,
      warning: alarmCount?.warning ?? 0,
    }),
    [alarmCount, total]
  )

  // ----------------------- mutations -----------------------
  const ackHistory = useAcknowledgeHistoryAlarms()
  const unackHistory = useUnacknowledgeHistoryAlarms()
  const deleteHistory = useDeleteHistoryAlarms()
  const ackBusy = ackHistory.isPending

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

  const applyQuick = useCallback((key: QuickFilterKey) => {
    setQuick(key)
    setPage(1)
    setSeverity('')
    setDealState('')
    if (key === 'critical' || key === 'major' || key === 'minor' || key === 'warning') {
      setSeverity(key)
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
        await ackHistory.mutateAsync({ ids: ackIds, note })
        resetSelection()
        setAckOpen(false)
        void refetch()
      } catch {
        setAckOpen(false)
      }
    },
    [ackHistory, ackIds, resetSelection, refetch]
  )

  const doUnack = useCallback(
    async (ids: string[]) => {
      if (ids.length === 0) return
      try {
        await unackHistory.mutateAsync(ids)
        resetSelection()
        void refetch()
      } catch {
        /* hook 内部已 invalidate；失败静默 */
      }
    },
    [unackHistory, resetSelection, refetch]
  )

  const doDelete = useCallback(
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
    '清除时间',
    '次数',
    '操作',
  ]

  return (
    <PageShell
      title="历史告警"
      description="历史告警查询 · 确认 / 反确认 / 删除"
      isFetching={isFetching}
      toolbar={
        <>
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

          <Select
            value={dealState || 'all'}
            onValueChange={(v) => {
              setDealState(v === 'all' ? '' : (v as '2' | '3'))
              setQuick('all')
              setPage(1)
            }}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="处理状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              <SelectItem value="2">未确认已清除</SelectItem>
              <SelectItem value="3">已确认已清除</SelectItem>
            </SelectContent>
          </Select>

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
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-5">
        <Stat label="总计" value={stats.total} />
        <Stat label="紧急" value={stats.critical} tone="critical" />
        <Stat label="重要" value={stats.major} tone="major" />
        <Stat label="次要" value={stats.minor} tone="minor" />
        <Stat label="警告" value={stats.warning} tone="warning" />
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
            disabled={selectedIds.length === 0 || unackHistory.isPending}
            onClick={() => doUnack(selectedIds)}
          >
            <MinusCircle /> 反确认
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={selectedIds.length === 0 || deleteHistory.isPending}
            onClick={() => doDelete(selectedIds)}
          >
            <Trash2 /> 删除
          </Button>
        </div>
      </div>

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c, i) =>
                i === 0 ? (
                  <TableHead key="select" className="w-8">
                    <input
                      type="checkbox"
                      className="size-4 cursor-pointer"
                      checked={allChecked}
                      onChange={toggleAll}
                      aria-label="全选"
                    />
                  </TableHead>
                ) : c === 'SN' ? (
                  <TableHead key={c} className="min-w-56">
                    {c}
                  </TableHead>
                ) : c === '操作' ? (
                  <TableHead key={c} className="w-40">
                    {c}
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
              <EmptyRow colSpan={cols.length}>暂无历史告警</EmptyRow>
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
                    <TableCell className="min-w-56">
                      <button
                        type="button"
                        className="flex max-w-56 items-center gap-1.5 font-mono text-xs hover:underline"
                        onClick={() => openDetail(a)}
                      >
                        {a.unread === '1' ? (
                          <span className="inline-block size-1.5 rounded-full bg-destructive" />
                        ) : null}
                        <span className="truncate" title={a.deviceSn}>{a.deviceSn}</span>
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
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(a.clearTime)}
                    </TableCell>
                    <TableCell className="tabular-nums">{a.alarmCount}</TableCell>
                    <TableCell className="w-40">
                      <div className="flex items-center gap-0.5">
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
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 px-2 text-xs text-destructive"
                          onClick={() => doDelete([a.id])}
                        >
                          删除
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
    </PageShell>
  )
}
