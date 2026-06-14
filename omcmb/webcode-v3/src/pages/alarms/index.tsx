import { useCallback, useMemo, useState } from 'react'
import {
  Search,
  RefreshCcw,
  Loader2,
  BellRing,
  SlidersHorizontal,
  Check,
  Eraser,
  Eye,
  RotateCcw,
  Trash2,
  ChevronDown,
  Activity,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { SeverityBar } from '@/components/viz/SeverityBar'
import { Sparkline } from '@/components/viz/Sparkline'
import { formatTime } from '@/lib/format'
import {
  useAlarmList,
  useAlarmCount,
  useHistoryAlarmCount,
  useAcknowledgeAlarms,
  useUnacknowledgeAlarms,
  useClearAlarms,
  useMarkAlarmRead,
  useAcknowledgeHistoryAlarms,
  useUnacknowledgeHistoryAlarms,
  useDeleteHistoryAlarms,
} from '@core/hooks/api/useAlarms'
import { useAlarmTrend, useTopAlarmDevices } from '@core/hooks/api/useDashboard'
import type {
  Alarm,
  AlarmFilter,
  DealState,
  EventType,
} from '@core/types/alarm'
import type { AlarmSeverity } from '@core/types/common'
import { AlarmDetailPanel, neTypeLabel } from './AlarmDetailPanel'
import { ConfirmNoteModal } from './ConfirmNoteModal'

const SEV_LABEL: Record<AlarmSeverity, string> = {
  critical: '紧急',
  major: '重要',
  minor: '次要',
  warning: '警告',
}
const SEV_COLOR: Record<AlarmSeverity, string> = {
  critical: '#ff2d6f',
  major: '#ff7a1a',
  minor: '#ffd400',
  warning: '#5b9eff',
}

const EVENT_LABEL: Record<EventType, string> = {
  communication: '通信',
  qualityOfService: '服务质量',
  processingError: '处理错误',
  device: '设备',
  environment: '环境',
  performance: '性能',
}

const DEAL_LABEL: Record<DealState, string> = {
  '0': '未确认',
  '1': '已确认',
  '2': '已清除',
  '3': '已确认/清除',
}

const NE_TYPE_OPTIONS = ['eNB', 'gNB', 'GSM']
const EVENT_TYPE_OPTIONS: EventType[] = [
  'communication',
  'qualityOfService',
  'processingError',
  'device',
  'environment',
]
const PAGE_SIZE = 30

type Mode = 'active' | 'history'

interface ModalState {
  kind: 'ack' | 'clear' | null
  ids: string[]
}

export function AlarmsPage() {
  const [mode, setMode] = useState<Mode>('active')
  const [page, setPage] = useState(1)
  const [severity, setSeverity] = useState<AlarmSeverity | ''>('')
  const [keyword, setKeyword] = useState('')
  // 高级筛选（对照 v1 FilterBar）
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [dealState, setDealState] = useState<DealState | ''>('')
  const [eventType, setEventType] = useState<EventType | ''>('')
  const [neType, setNeType] = useState('')
  const [unread, setUnread] = useState<'' | '0' | '1'>('')
  const [isUnknown, setIsUnknown] = useState<'' | 'true' | 'false'>('')

  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [detail, setDetail] = useState<Alarm | null>(null)
  const [modal, setModal] = useState<ModalState>({ kind: null, ids: [] })
  const [submitting, setSubmitting] = useState(false)
  const [opError, setOpError] = useState<string | null>(null)

  // ---- 统计（全网计数，与列表分页解耦） ---------------------------------
  const { data: activeCnt } = useAlarmCount()
  const { data: histCnt } = useHistoryAlarmCount()
  const cnt = mode === 'active' ? activeCnt : histCnt

  // ---- 趋势 / Top 设备（概览深度，真实后端数据） ------------------------
  const { data: trend, isLoading: trendLoading } = useAlarmTrend(7)
  const { data: topDevices, isLoading: topLoading } = useTopAlarmDevices()

  // ---- 列表 -------------------------------------------------------------
  const params = useMemo<AlarmFilter & { page: number; pageSize: number; isActive?: boolean }>(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      isActive: mode === 'active',
      ...(severity ? { severity } : {}),
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(dealState ? { dealState } : {}),
      ...(eventType ? { eventType } : {}),
      ...(neType ? { neType } : {}),
      ...(unread ? { unread } : {}),
      ...(isUnknown ? { isUnknown } : {}),
    }),
    [page, mode, severity, keyword, dealState, eventType, neType, unread, isUnknown]
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useAlarmList(params)
  const rows = useMemo(() => data?.items ?? [], [data])
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  // ---- 变更 hooks -------------------------------------------------------
  const ackActive = useAcknowledgeAlarms()
  const unackActive = useUnacknowledgeAlarms()
  const clearActive = useClearAlarms()
  const markRead = useMarkAlarmRead()
  const ackHist = useAcknowledgeHistoryAlarms()
  const unackHist = useUnacknowledgeHistoryAlarms()
  const delHist = useDeleteHistoryAlarms()

  const resetFilters = useCallback(() => {
    setSeverity('')
    setKeyword('')
    setDealState('')
    setEventType('')
    setNeType('')
    setUnread('')
    setIsUnknown('')
    setPage(1)
  }, [])

  const toggleSelect = useCallback((id: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }, [])
  const toggleSelectAll = useCallback(() => {
    setSelected((prev) => {
      if (rows.every((r) => prev.has(r.id)) && rows.length > 0) return new Set()
      return new Set(rows.map((r) => r.id))
    })
  }, [rows])

  const selectedIds = useMemo(() => Array.from(selected), [selected])

  // ---- 操作 -------------------------------------------------------------
  const runOp = useCallback(
    async (fn: () => Promise<unknown>) => {
      setSubmitting(true)
      setOpError(null)
      try {
        await fn()
        setSelected(new Set())
        setModal({ kind: null, ids: [] })
        await refetch()
      } catch (e) {
        setOpError(e instanceof Error ? e.message : '操作失败')
      } finally {
        setSubmitting(false)
      }
    },
    [refetch]
  )

  const confirmModal = useCallback(
    async (note: string) => {
      const ids = modal.ids
      if (ids.length === 0) return
      if (modal.kind === 'ack') {
        await runOp(() =>
          mode === 'active'
            ? ackActive.mutateAsync({ ids, note })
            : ackHist.mutateAsync({ ids, note })
        )
      } else if (modal.kind === 'clear') {
        await runOp(() => clearActive.mutateAsync({ ids, note }))
      }
    },
    [modal, mode, ackActive, ackHist, clearActive, runOp]
  )

  const onUnack = useCallback(
    (ids: string[]) => {
      if (ids.length === 0) return
      void runOp(() =>
        mode === 'active'
          ? unackActive.mutateAsync(ids)
          : unackHist.mutateAsync(ids)
      )
    },
    [mode, unackActive, unackHist, runOp]
  )
  const onMarkRead = useCallback(
    (ids: string[]) => {
      if (ids.length === 0) return
      void runOp(() => Promise.all(ids.map((id) => markRead.mutateAsync(id))))
    },
    [markRead, runOp]
  )
  const onDeleteHist = useCallback(
    (ids: string[]) => {
      if (ids.length === 0) return
      void runOp(() => delHist.mutateAsync(ids))
    },
    [delHist, runOp]
  )

  const switchMode = useCallback((m: Mode) => {
    setMode(m)
    setPage(1)
    setSelected(new Set())
  }, [])

  const allChecked = rows.length > 0 && rows.every((r) => selected.has(r.id))

  return (
    <PageShell
      code="F04"
      title="ALARM ARRAY · 警报阵列"
      subtitle={
        mode === 'active'
          ? 'ACTIVE INCIDENT FEED · 30s AUTO-REFRESH'
          : 'HISTORICAL ARCHIVE · CLEARED ALARMS'
      }
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-72 pl-9"
              placeholder="告警名 / 设备 SN / 标识"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          {(['', 'critical', 'major', 'minor', 'warning'] as const).map((s) => (
            <button
              key={s || 'all'}
              type="button"
              onClick={() => {
                setSeverity(s)
                setPage(1)
              }}
              className={`chip transition-all ${
                severity === s ? 'shadow-[0_0_10px_currentColor]' : 'opacity-60 hover:opacity-100'
              }`}
              style={{ color: s ? SEV_COLOR[s] : '#00f0ff' }}
            >
              {s ? SEV_LABEL[s as AlarmSeverity] : 'ALL'}
            </button>
          ))}
          <NeonButton
            icon={<SlidersHorizontal />}
            onClick={() => setShowAdvanced((v) => !v)}
            className={showAdvanced ? 'shadow-[0_0_10px_currentColor]' : undefined}
          >
            FILTER
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {/* 模式切换 */}
      <div className="mb-3 flex items-center gap-2">
        {(
          [
            { k: 'active', label: 'ACTIVE · 当前告警' },
            { k: 'history', label: 'HISTORY · 历史告警' },
          ] as const
        ).map((m) => (
          <button
            key={m.k}
            type="button"
            onClick={() => switchMode(m.k)}
            className={`chip transition-all ${
              mode === m.k
                ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                : 'text-cyan-300/45 hover:text-cyan-300/80'
            }`}
          >
            {m.label}
          </button>
        ))}
      </div>

      {/* 高级筛选行（对照 v1 FilterBar） */}
      {showAdvanced && (
        <div className="mb-3 grid grid-cols-2 gap-3 rounded-sm border border-cyan-500/15 bg-cyan-500/[0.03] p-3 md:grid-cols-3 lg:grid-cols-5">
          <FilterSelect
            label="网元类型"
            value={neType}
            onChange={setNeType}
            options={[
              { v: '', t: '全部' },
              ...NE_TYPE_OPTIONS.map((n) => ({ v: n, t: neTypeLabel(n) })),
            ]}
          />
          <FilterSelect
            label="事件类型"
            value={eventType}
            onChange={(v) => setEventType(v as EventType | '')}
            options={[
              { v: '', t: '全部' },
              ...EVENT_TYPE_OPTIONS.map((e) => ({ v: e, t: EVENT_LABEL[e] })),
            ]}
          />
          <FilterSelect
            label="告警状态"
            value={dealState}
            onChange={(v) => setDealState(v as DealState | '')}
            options={[
              { v: '', t: '全部' },
              { v: '0', t: '未确认未清除' },
              { v: '1', t: '已确认未清除' },
            ]}
          />
          <FilterSelect
            label="阅读状态"
            value={unread}
            onChange={(v) => setUnread(v as '' | '0' | '1')}
            options={[
              { v: '', t: '全部' },
              { v: '0', t: '已读' },
              { v: '1', t: '未读' },
            ]}
          />
          <FilterSelect
            label="识别状态"
            value={isUnknown}
            onChange={(v) => setIsUnknown(v as '' | 'true' | 'false')}
            options={[
              { v: '', t: '全部' },
              { v: 'true', t: '未识别' },
              { v: 'false', t: '已识别' },
            ]}
          />
          <div className="flex items-end">
            <NeonButton icon={<RotateCcw />} onClick={resetFilters}>
              RESET
            </NeonButton>
          </div>
        </div>
      )}

      {/* 统计 + 概览 */}
      <div className="mb-3 grid grid-cols-12 gap-3">
        <div className="col-span-12 grid grid-cols-3 gap-3 lg:col-span-7 xl:grid-cols-6">
          <SevStat label="CRITICAL" color={SEV_COLOR.critical} value={cnt?.critical ?? 0} />
          <SevStat label="MAJOR" color={SEV_COLOR.major} value={cnt?.major ?? 0} />
          <SevStat label="MINOR" color={SEV_COLOR.minor} value={cnt?.minor ?? 0} />
          <SevStat label="WARNING" color={SEV_COLOR.warning} value={cnt?.warning ?? 0} />
          <SevStat label="UNACKED" color="#00f0ff" value={cnt?.unacknowledged ?? 0} />
          <SevStat label="UNREAD" color="#a855f7" value={cnt?.unread ?? 0} />
        </div>

        {/* 7 日趋势 */}
        <GlassPanel
          title="7-DAY TREND · 趋势"
          meta="LIVE"
          className="col-span-12 lg:col-span-5"
        >
          <AlarmTrendStrip trend={trend} loading={trendLoading} />
        </GlassPanel>
      </div>

      {/* 批量操作条 */}
      {selectedIds.length > 0 && (
        <div className="mb-2 flex flex-wrap items-center gap-2 rounded-sm border border-cyan-500/25 bg-cyan-500/[0.05] px-3 py-2">
          <span className="font-mono text-[11px] text-cyan-200">
            已选 {selectedIds.length} 条
          </span>
          {mode === 'active' ? (
            <>
              <NeonButton icon={<Check />} onClick={() => setModal({ kind: 'ack', ids: selectedIds })}>
                确认
              </NeonButton>
              <NeonButton icon={<RotateCcw />} onClick={() => onUnack(selectedIds)}>
                取消确认
              </NeonButton>
              <NeonButton tone="danger" icon={<Eraser />} onClick={() => setModal({ kind: 'clear', ids: selectedIds })}>
                清除
              </NeonButton>
              <NeonButton icon={<Eye />} onClick={() => onMarkRead(selectedIds)}>
                标记已读
              </NeonButton>
            </>
          ) : (
            <>
              <NeonButton icon={<Check />} onClick={() => setModal({ kind: 'ack', ids: selectedIds })}>
                确认
              </NeonButton>
              <NeonButton icon={<RotateCcw />} onClick={() => onUnack(selectedIds)}>
                取消确认
              </NeonButton>
              <NeonButton tone="danger" icon={<Trash2 />} onClick={() => onDeleteHist(selectedIds)}>
                删除
              </NeonButton>
            </>
          )}
          <NeonButton onClick={() => setSelected(new Set())}>清空选择</NeonButton>
          {submitting && <Loader2 className="size-4 animate-spin text-cyan-300/70" />}
        </div>
      )}

      {opError && (
        <div className="mb-2 border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">
          OP FAILED · {opError}
        </div>
      )}

      {/* 列表头 */}
      {rows.length > 0 && (
        <div className="mb-1 grid grid-cols-[28px_28px_2fr_1.4fr_1fr_1fr_140px_88px] items-center gap-3 px-3 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
          <button
            type="button"
            onClick={toggleSelectAll}
            className={`flex size-4 items-center justify-center rounded-sm border ${
              allChecked ? 'border-cyan-400 bg-cyan-400/20' : 'border-cyan-500/40'
            }`}
            aria-label="select all"
          >
            {allChecked && <Check className="size-3 text-cyan-200" />}
          </button>
          <span />
          <span>ALARM · CAUSE</span>
          <span>DEVICE</span>
          <span>NE / EVENT</span>
          <span>TIME</span>
          <span className="text-right">STATE</span>
          <span className="text-right">OPS</span>
        </div>
      )}

      {/* 列表主体 */}
      <div className="space-y-1.5">
        {isLoading ? (
          <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
            <Loader2 className="size-4 animate-spin" />
            <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
          </div>
        ) : isError ? (
          <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
            FAILURE · {error instanceof Error ? error.message : '未知错误'}
          </div>
        ) : rows.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-3 py-16">
            <BellRing className="size-10 text-emerald-400/60" />
            <div className="font-mono text-xs uppercase tracking-[0.2em] text-emerald-300/70">
              {mode === 'active' ? 'ALL CLEAR · 无活动告警' : 'NO RECORDS · 无历史告警'}
            </div>
          </div>
        ) : (
          rows.map((a) => {
            const checked = selected.has(a.id)
            const confirmed = a.dealState === '1' || a.dealState === '3'
            return (
              <div
                key={a.id}
                className="fleet-row grid grid-cols-[28px_28px_2fr_1.4fr_1fr_1fr_140px_88px] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: SEV_COLOR[a.severity] }}
              >
                <button
                  type="button"
                  onClick={() => toggleSelect(a.id)}
                  className={`flex size-4 items-center justify-center rounded-sm border ${
                    checked ? 'border-cyan-400 bg-cyan-400/20' : 'border-cyan-500/40'
                  }`}
                  aria-label="select row"
                >
                  {checked && <Check className="size-3 text-cyan-200" />}
                </button>
                <SeverityBar severity={a.severity} />
                <button
                  type="button"
                  className="min-w-0 text-left"
                  onClick={() => setDetail(a)}
                >
                  <div className="flex items-center gap-1.5 font-display text-sm font-bold text-cyan-100">
                    {a.unread === '1' && mode === 'active' && (
                      <span className="size-1.5 shrink-0 rounded-full bg-rose-400 shadow-[0_0_6px_currentColor]" />
                    )}
                    <span className="truncate">{a.alarmName || a.alarmIdentifier}</span>
                  </div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">
                    CODE {a.alarmIdentifier} · {EVENT_LABEL[a.eventType] || a.eventType}
                  </div>
                </button>
                <div className="min-w-0">
                  <div className="truncate text-xs text-cyan-100/85">{a.deviceName || '—'}</div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">{a.deviceSn}</div>
                </div>
                <div className="min-w-0">
                  <div className="truncate text-xs text-cyan-100/85">{neTypeLabel(a.neType)}</div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">
                    ×{a.alarmCount ?? 1}
                  </div>
                </div>
                <div className="font-mono text-[11px] text-cyan-300/75">{formatTime(a.eventTime)}</div>
                <div className="text-right">
                  <span
                    className="chip"
                    style={{ color: a.dealState === '0' ? SEV_COLOR[a.severity] : '#00ff88' }}
                  >
                    {DEAL_LABEL[a.dealState] ?? '处理中'}
                  </span>
                </div>
                <div className="flex items-center justify-end gap-1.5">
                  <RowAction
                    title="详情"
                    onClick={() => setDetail(a)}
                    icon={<ChevronDown className="size-3.5 -rotate-90" />}
                  />
                  {mode === 'active' && (
                    confirmed ? (
                      <RowAction
                        title="取消确认"
                        onClick={() => onUnack([a.id])}
                        icon={<RotateCcw className="size-3.5" />}
                      />
                    ) : (
                      <RowAction
                        title="确认"
                        onClick={() => setModal({ kind: 'ack', ids: [a.id] })}
                        icon={<Check className="size-3.5" />}
                      />
                    )
                  )}
                  {mode === 'active' && (
                    <RowAction
                      title="清除"
                      danger
                      onClick={() => setModal({ kind: 'clear', ids: [a.id] })}
                      icon={<Eraser className="size-3.5" />}
                    />
                  )}
                </div>
              </div>
            )
          })
        )}
      </div>

      {/* 分页 */}
      <div className="mt-4 flex items-center justify-between">
        <span className="font-mono text-[11px] text-cyan-300/55">
          PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
        </span>
        <div className="flex gap-2">
          <NeonButton onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
            ◂ PREV
          </NeonButton>
          <NeonButton
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages}
          >
            NEXT ▸
          </NeonButton>
        </div>
      </div>

      {/* Top 故障设备（概览深度） */}
      <div className="mt-3">
        <GlassPanel title="TOP ALARMED · 故障设备 TOP" meta="LIVE">
          <TopDevices devices={topDevices} loading={topLoading} />
        </GlassPanel>
      </div>

      {/* 详情抽屉 */}
      <AlarmDetailPanel alarm={detail} open={detail !== null} onClose={() => setDetail(null)} />

      {/* 确认 / 清除 备注弹层 */}
      <ConfirmNoteModal
        open={modal.kind !== null}
        title={modal.kind === 'clear' ? '清除告警' : '确认告警'}
        message={
          modal.kind === 'clear'
            ? `将清除 ${modal.ids.length} 条告警，可填写清除备注。`
            : `将确认 ${modal.ids.length} 条告警，可填写确认备注。`
        }
        confirmText={modal.kind === 'clear' ? '清除' : '确认'}
        danger={modal.kind === 'clear'}
        loading={submitting}
        onConfirm={(note) => void confirmModal(note)}
        onCancel={() => setModal({ kind: null, ids: [] })}
      />
    </PageShell>
  )
}

function RowAction({
  title,
  icon,
  danger,
  onClick,
}: {
  title: string
  icon: React.ReactNode
  danger?: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      title={title}
      onClick={onClick}
      className={`flex size-6 items-center justify-center rounded-sm border transition-colors ${
        danger
          ? 'border-rose-500/30 text-rose-300/80 hover:border-rose-400/70 hover:text-rose-200'
          : 'border-cyan-500/25 text-cyan-300/75 hover:border-cyan-400/60 hover:text-cyan-100'
      }`}
    >
      {icon}
    </button>
  )
}

function FilterSelect({
  label,
  value,
  onChange,
  options,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  options: { v: string; t: string }[]
}) {
  return (
    <label className="flex flex-col gap-1">
      <span className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">
        {label}
      </span>
      <select
        className="neon-input"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      >
        {options.map((o) => (
          <option key={o.v || 'all'} value={o.v} className="bg-[#03050d]">
            {o.t}
          </option>
        ))}
      </select>
    </label>
  )
}

function SevStat({
  label,
  color,
  value,
}: {
  label: string
  color: string
  value: number
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: color }}
    >
      <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/65">
        {label}
      </div>
      <div
        className="font-display text-2xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}

function AlarmTrendStrip({
  trend,
  loading,
}: {
  trend: { date: string; critical: number; major: number; minor: number; warning: number }[] | undefined
  loading: boolean
}) {
  if (loading) {
    return (
      <div className="flex items-center justify-center gap-2 px-4 py-8 text-cyan-300/55">
        <Loader2 className="size-4 animate-spin" />
        <span className="font-mono text-[11px] uppercase tracking-[0.2em]">LOADING…</span>
      </div>
    )
  }
  if (!trend || trend.length === 0) {
    return (
      <div className="flex items-center justify-center gap-2 px-4 py-8 text-cyan-300/45">
        <Activity className="size-4" />
        <span className="font-mono text-[11px] uppercase tracking-[0.2em]">NO TREND DATA</span>
      </div>
    )
  }
  const series: { key: 'critical' | 'major' | 'minor' | 'warning'; label: string }[] = [
    { key: 'critical', label: 'CRIT' },
    { key: 'major', label: 'MAJ' },
    { key: 'minor', label: 'MIN' },
    { key: 'warning', label: 'WRN' },
  ]
  return (
    <div className="space-y-1.5 p-3">
      {series.map((s) => {
        const points = trend.map((d) => d[s.key])
        const sum = points.reduce((a, b) => a + b, 0)
        return (
          <div key={s.key} className="flex items-center gap-3">
            <span
              className="w-8 shrink-0 font-mono text-[10px] font-bold tracking-[0.16em]"
              style={{ color: SEV_COLOR[s.key], textShadow: `0 0 4px ${SEV_COLOR[s.key]}` }}
            >
              {s.label}
            </span>
            <div className="flex-1">
              <Sparkline data={points} color={SEV_COLOR[s.key]} width={220} height={26} />
            </div>
            <span
              className="w-10 shrink-0 text-right font-display text-xs font-bold"
              style={{ color: SEV_COLOR[s.key] }}
            >
              {sum}
            </span>
          </div>
        )
      })}
    </div>
  )
}

function TopDevices({
  devices,
  loading,
}: {
  devices: { deviceSN: string; technology: string; deviceName: string; alarmCount: number; severity: string }[] | undefined
  loading: boolean
}) {
  if (loading) {
    return (
      <div className="flex items-center justify-center gap-2 px-4 py-6 text-cyan-300/55">
        <Loader2 className="size-4 animate-spin" />
        <span className="font-mono text-[11px] uppercase tracking-[0.2em]">LOADING…</span>
      </div>
    )
  }
  if (!devices || devices.length === 0) {
    return (
      <div className="px-4 py-6 text-center font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/45">
        NO DATA · 暂无故障设备
      </div>
    )
  }
  return (
    <div className="divide-y divide-cyan-500/8">
      {devices.slice(0, 8).map((d) => {
        const color =
          (d.severity in SEV_COLOR ? SEV_COLOR[d.severity as AlarmSeverity] : '#5b9eff')
        return (
          <div
            key={d.deviceSN}
            className="flex items-center gap-2 px-3 py-2 hover:bg-cyan-500/5"
          >
            <span
              className="size-2 rounded-full"
              style={{ background: color, boxShadow: `0 0 8px ${color}` }}
            />
            <div className="min-w-0 flex-1">
              <div className="truncate font-mono text-xs text-cyan-100">
                {d.deviceName || d.deviceSN}
              </div>
              <div className="truncate text-[10px] uppercase tracking-[0.14em] text-cyan-300/55">
                {neTypeLabel(d.technology)} · {d.deviceSN}
              </div>
            </div>
            <div
              className="font-display text-sm font-bold"
              style={{ color, textShadow: `0 0 6px ${color}` }}
            >
              {d.alarmCount}
            </div>
          </div>
        )
      })}
    </div>
  )
}
