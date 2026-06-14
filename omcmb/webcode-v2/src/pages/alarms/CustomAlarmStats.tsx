import { useCallback, useEffect, useMemo, useState } from 'react'
import { Plus, RefreshCcw, Search, Trash2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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

import { useCurrentAlarms, useHistoricalAlarms } from '@core/hooks/api/useAlarms'
import type { Alarm, AlarmFilter } from '@core/types/alarm'
import type { AlarmSeverity } from '@core/types/common'

import { AlarmDetailDrawer } from './AlarmDetailDrawer'

const STORAGE_KEY = 'v2-custom-alarm-views'

interface CustomView {
  id: string
  name: string
  scope: 'active' | 'history'
  keyword?: string
  severity?: AlarmSeverity | ''
  neType?: string
}

const DEFAULT_VIEWS: CustomView[] = [
  { id: 'all-active', name: '全部活动告警', scope: 'active' },
  { id: 'critical-active', name: '紧急告警', scope: 'active', severity: 'critical' },
  { id: 'all-history', name: '全部历史告警', scope: 'history' },
]

const SEVERITY_META: Record<
  AlarmSeverity,
  { label: string; variant: 'destructive' | 'warning' | 'default'; text: string }
> = {
  critical: { label: '紧急', variant: 'destructive', text: 'text-red-600 dark:text-red-400' },
  major: { label: '重要', variant: 'destructive', text: 'text-orange-600 dark:text-orange-400' },
  minor: { label: '次要', variant: 'warning', text: 'text-yellow-600 dark:text-yellow-400' },
  warning: { label: '警告', variant: 'warning', text: 'text-sky-600 dark:text-sky-400' },
}

const NE_TYPE_OPTIONS = ['ENB', 'GNB', 'CPE', 'UPS', 'WCG', 'GSM']

function loadViews(): CustomView[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw) as CustomView[]
      if (Array.isArray(parsed) && parsed.length > 0) return parsed
    }
  } catch {
    /* ignore */
  }
  return DEFAULT_VIEWS
}

function Stat({
  label,
  value,
  tone,
}: {
  label: string
  value: number
  tone?: AlarmSeverity
}) {
  return (
    <div className="rounded-lg border bg-card px-3 py-2">
      <div className="text-[11px] uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div
        className={cn(
          'mt-0.5 text-xl font-semibold tabular-nums',
          tone ? SEVERITY_META[tone].text : 'text-foreground'
        )}
      >
        {value}
      </div>
    </div>
  )
}

export default function CustomAlarmStats() {
  const [views, setViews] = useState<CustomView[]>(loadViews)
  const [activeId, setActiveId] = useState<string>(() => loadViews()[0]?.id ?? '')
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [detailAlarm, setDetailAlarm] = useState<Alarm | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)

  // 新建视图弹窗
  const [creating, setCreating] = useState(false)
  const [draftName, setDraftName] = useState('')
  const [draftScope, setDraftScope] = useState<'active' | 'history'>('active')
  const [draftSeverity, setDraftSeverity] = useState<AlarmSeverity | ''>('')
  const [draftNeType, setDraftNeType] = useState('')
  const [draftKeyword, setDraftKeyword] = useState('')

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(views))
    } catch {
      /* ignore */
    }
  }, [views])

  const activeView = useMemo(
    () => views.find((v) => v.id === activeId) ?? views[0] ?? null,
    [views, activeId]
  )

  const filter = useMemo<AlarmFilter & { page: number; pageSize: number }>(() => {
    const f: AlarmFilter & { page: number; pageSize: number } = { page, pageSize }
    if (activeView?.keyword) f.keyword = activeView.keyword
    if (activeView?.severity) f.severity = activeView.severity
    if (activeView?.neType) f.neType = activeView.neType
    return f
  }, [activeView, page, pageSize])

  const isActiveScope = activeView?.scope !== 'history'

  const activeQuery = useCurrentAlarms(filter, {
    refetchIntervalMs: false,
  })
  const historyQuery = useHistoricalAlarms(filter, { refetchIntervalMs: false })
  const query = isActiveScope ? activeQuery : historyQuery

  const { data, isLoading, isError, error, isFetching, refetch } = query
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  // 从当前页结果计算分布（自定义统计口径）
  const stats = useMemo(() => {
    const count = (sev: AlarmSeverity) => rows.filter((r) => r.severity === sev).length
    return {
      total,
      critical: count('critical'),
      major: count('major'),
      minor: count('minor'),
      warning: count('warning'),
      unacked: rows.filter((r) => r.dealState === '0' || r.dealState === '2').length,
    }
  }, [rows, total])

  const selectView = useCallback((id: string) => {
    setActiveId(id)
    setPage(1)
  }, [])

  const deleteView = useCallback(
    (id: string) => {
      setViews((prev) => {
        const next = prev.filter((v) => v.id !== id)
        if (next.length === 0) return prev // 至少保留一个
        if (id === activeId) {
          setActiveId(next[0].id)
          setPage(1)
        }
        return next
      })
    },
    [activeId]
  )

  const resetDraft = useCallback(() => {
    setDraftName('')
    setDraftScope('active')
    setDraftSeverity('')
    setDraftNeType('')
    setDraftKeyword('')
  }, [])

  const createView = useCallback(() => {
    const name = draftName.trim()
    if (!name) return
    const id = `view-${Date.now()}`
    const view: CustomView = {
      id,
      name,
      scope: draftScope,
      severity: draftSeverity || undefined,
      neType: draftNeType || undefined,
      keyword: draftKeyword.trim() || undefined,
    }
    setViews((prev) => [...prev, view])
    setActiveId(id)
    setPage(1)
    setCreating(false)
    resetDraft()
  }, [draftName, draftScope, draftSeverity, draftNeType, draftKeyword, resetDraft])

  const openDetail = useCallback((a: Alarm) => {
    setDetailAlarm(a)
    setDetailOpen(true)
  }, [])

  const cols = ['SN', '严重度', '告警标识', '可能原因', '网元定位', '故障时间']

  return (
    <PageShell
      title="自定义统计"
      description="按自定义视图聚合告警 · 视图保存在本地"
      isFetching={isFetching}
      toolbar={
        <>
          <Button
            variant="outline"
            size="sm"
            onClick={() => refetch()}
          >
            <RefreshCcw /> 刷新
          </Button>
          <Button
            size="sm"
            className="ml-auto"
            onClick={() => {
              resetDraft()
              setCreating(true)
            }}
          >
            <Plus /> 新建视图
          </Button>
        </>
      }
    >
      <div className="flex gap-4">
        {/* 左侧视图列表 */}
        <div className="w-56 shrink-0">
          <div className="overflow-hidden rounded-lg border bg-card">
            {views.map((v) => (
              <div
                key={v.id}
                className={cn(
                  'flex items-center justify-between border-b px-3 py-2 text-sm last:border-b-0',
                  v.id === activeView?.id ? 'bg-primary/10' : 'hover:bg-muted/50'
                )}
              >
                <button
                  type="button"
                  className="flex-1 truncate text-left"
                  onClick={() => selectView(v.id)}
                  title={v.name}
                >
                  {v.name}
                </button>
                {views.length > 1 ? (
                  <button
                    type="button"
                    className="ml-2 text-muted-foreground hover:text-destructive"
                    onClick={() => deleteView(v.id)}
                    aria-label="删除视图"
                  >
                    <Trash2 className="size-3.5" />
                  </button>
                ) : null}
              </div>
            ))}
          </div>
        </div>

        {/* 右侧内容 */}
        <div className="min-w-0 flex-1">
          <div className="mb-3 flex items-center gap-2">
            <span className="text-base font-semibold">{activeView?.name}</span>
            <Badge variant="muted">
              {isActiveScope ? '活动告警' : '历史告警'}
            </Badge>
          </div>

          <div className="mb-4 grid grid-cols-3 gap-3 md:grid-cols-6">
            <Stat label="总计" value={stats.total} />
            <Stat label="紧急" value={stats.critical} tone="critical" />
            <Stat label="重要" value={stats.major} tone="major" />
            <Stat label="次要" value={stats.minor} tone="minor" />
            <Stat label="警告" value={stats.warning} tone="warning" />
            <Stat label="未确认" value={stats.unacked} />
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
                  <EmptyRow colSpan={cols.length}>该视图下暂无告警</EmptyRow>
                ) : (
                  rows.map((a) => (
                    <TableRow key={a.id} className="align-top">
                      <TableCell>
                        <button
                          type="button"
                          className="font-mono text-xs hover:underline"
                          onClick={() => openDetail(a)}
                        >
                          {a.deviceSn}
                        </button>
                      </TableCell>
                      <TableCell>
                        <Badge variant={SEVERITY_META[a.severity].variant}>
                          {SEVERITY_META[a.severity].label}
                        </Badge>
                      </TableCell>
                      <TableCell className="font-mono text-xs">
                        {a.alarmIdentifier || '—'}
                      </TableCell>
                      <TableCell className="max-w-[220px]">
                        <div className="truncate" title={a.alarmName}>
                          {a.alarmName || '—'}
                        </div>
                      </TableCell>
                      <TableCell className="max-w-[200px] text-xs">
                        <div className="truncate" title={a.equipInfo}>
                          {a.equipInfo || '—'}
                        </div>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {formatTime(a.eventTime)}
                      </TableCell>
                    </TableRow>
                  ))
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
        </div>
      </div>

      <AlarmDetailDrawer
        alarm={detailAlarm}
        open={detailOpen}
        onClose={() => {
          setDetailOpen(false)
          setDetailAlarm(null)
        }}
      />

      {/* 新建视图弹窗 */}
      {creating ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="absolute inset-0 bg-black/40"
            onClick={() => setCreating(false)}
            aria-hidden
          />
          <div className="relative w-full max-w-md rounded-lg border bg-background p-5 shadow-xl">
            <h2 className="text-base font-semibold">新建自定义视图</h2>
            <div className="mt-4 space-y-4">
              <div className="space-y-1.5">
                <Label htmlFor="view-name">视图名称</Label>
                <Input
                  id="view-name"
                  value={draftName}
                  placeholder="输入视图名称"
                  maxLength={50}
                  onChange={(e) => setDraftName(e.target.value)}
                  autoFocus
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="view-scope">告警范围</Label>
                <Select
                  value={draftScope}
                  onValueChange={(v) => setDraftScope(v as 'active' | 'history')}
                >
                  <SelectTrigger id="view-scope">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="active">活动告警</SelectItem>
                    <SelectItem value="history">历史告警</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="view-severity">严重度</Label>
                <Select
                  value={draftSeverity || 'all'}
                  onValueChange={(v) =>
                    setDraftSeverity(v === 'all' ? '' : (v as AlarmSeverity))
                  }
                >
                  <SelectTrigger id="view-severity">
                    <SelectValue placeholder="全部" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">全部严重度</SelectItem>
                    <SelectItem value="critical">紧急</SelectItem>
                    <SelectItem value="major">重要</SelectItem>
                    <SelectItem value="minor">次要</SelectItem>
                    <SelectItem value="warning">警告</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="view-ne">告警源</Label>
                <Select
                  value={draftNeType || 'all'}
                  onValueChange={(v) => setDraftNeType(v === 'all' ? '' : v)}
                >
                  <SelectTrigger id="view-ne">
                    <SelectValue placeholder="全部" />
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
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="view-keyword">关键字</Label>
                <div className="relative">
                  <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    id="view-keyword"
                    className="pl-9"
                    value={draftKeyword}
                    placeholder="SN / 告警标识 / 关键字"
                    onChange={(e) => setDraftKeyword(e.target.value)}
                  />
                </div>
              </div>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setCreating(false)}
              >
                取消
              </Button>
              <Button
                size="sm"
                onClick={createView}
                disabled={!draftName.trim()}
              >
                创建
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </PageShell>
  )
}
