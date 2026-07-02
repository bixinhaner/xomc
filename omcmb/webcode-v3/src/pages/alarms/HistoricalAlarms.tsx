import { useCallback, useMemo, useState } from 'react'
import {
  Search,
  RefreshCcw,
  Loader2,
  History,
  Check,
  RotateCcw,
  Trash2,
  Download,
  ChevronDown,
  SlidersHorizontal,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { SeverityBar } from '@/components/viz/SeverityBar'
import { formatTime } from '@/lib/format'
import {
  useHistoricalAlarms,
  useHistoryAlarmCount,
  useAcknowledgeHistoryAlarms,
  useUnacknowledgeHistoryAlarms,
  useDeleteHistoryAlarms,
} from '@core/hooks/api/useAlarms'
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

// 历史告警状态：仅已清除两态（对照 v1 HistoricalAlarms dealState 选项）
const DEAL_LABEL: Record<DealState, string> = {
  '0': '未确认未清除',
  '1': '已确认未清除',
  '2': '未确认已清除',
  '3': '已确认已清除',
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

interface ModalState {
  kind: 'ack' | null
  ids: string[]
}

function escapeCsvCell(value: unknown): string {
  const normalized = value == null ? '' : String(value)
  return `"${normalized.replace(/"/g, '""')}"`
}

function triggerCsvDownload(content: string, filename: string) {
  const blob = new Blob(['﻿' + content], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}

const CSV_FIELDS: { label: string; get: (a: Alarm) => unknown }[] = [
  { label: '设备SN', get: (a) => a.deviceSn },
  { label: '网元类型', get: (a) => neTypeLabel(a.neType) },
  { label: '严重程度', get: (a) => SEV_LABEL[a.severity] ?? a.severity },
  { label: '告警标识', get: (a) => a.alarmIdentifier },
  { label: '可能原因', get: (a) => a.alarmName },
  { label: '事件类型', get: (a) => EVENT_LABEL[a.eventType] ?? a.eventType },
  { label: '告警状态', get: (a) => DEAL_LABEL[a.dealState] ?? '' },
  { label: '故障时间', get: (a) => formatTime(a.eventTime) },
  { label: '更新时间', get: (a) => formatTime(a.updTime) },
  { label: '清除时间', get: (a) => formatTime(a.clearTime) },
  { label: '告警次数', get: (a) => a.alarmCount },
  { label: '确认备注', get: (a) => a.dealMemo },
]

export default function HistoricalAlarms() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  // 高级筛选（对照 v1 FilterBar）
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [dealState, setDealState] = useState<DealState | ''>('')
  const [eventType, setEventType] = useState<EventType | ''>('')
  const [neType, setNeType] = useState('')
  const [deviceSn, setDeviceSn] = useState('')
  const [alarmIdentifier, setAlarmIdentifier] = useState('')

  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [detail, setDetail] = useState<Alarm | null>(null)
  const [modal, setModal] = useState<ModalState>({ kind: null, ids: [] })
  const [submitting, setSubmitting] = useState(false)
  const [opError, setOpError] = useState<string | null>(null)
  const [exporting, setExporting] = useState(false)

  const params = useMemo<AlarmFilter & { page: number; pageSize: number }>(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(dealState ? { dealState } : {}),
      ...(eventType ? { eventType } : {}),
      ...(neType ? { neType } : {}),
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
      ...(alarmIdentifier.trim() ? { alarmIdentifier: alarmIdentifier.trim() } : {}),
    }),
    [page, keyword, dealState, eventType, neType, deviceSn, alarmIdentifier]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useHistoricalAlarms(params)
  const { data: histCnt } = useHistoryAlarmCount()
  const rows = useMemo(() => data?.items ?? [], [data])
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const ackHist = useAcknowledgeHistoryAlarms()
  const unackHist = useUnacknowledgeHistoryAlarms()
  const delHist = useDeleteHistoryAlarms()

  const resetFilters = useCallback(() => {
    setKeyword('')
    setDealState('')
    setEventType('')
    setNeType('')
    setDeviceSn('')
    setAlarmIdentifier('')
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
      if (rows.length > 0 && rows.every((r) => prev.has(r.id))) return new Set()
      return new Set(rows.map((r) => r.id))
    })
  }, [rows])

  const selectedIds = useMemo(() => Array.from(selected), [selected])
  const allChecked = rows.length > 0 && rows.every((r) => selected.has(r.id))

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

  const confirmAck = useCallback(
    async (note: string) => {
      const ids = modal.ids
      if (ids.length === 0) return
      await runOp(() => ackHist.mutateAsync({ ids, note }))
    },
    [modal.ids, ackHist, runOp]
  )

  const onUnack = useCallback(
    (ids: string[]) => {
      if (ids.length === 0) return
      void runOp(() => unackHist.mutateAsync(ids))
    },
    [unackHist, runOp]
  )
  const onDelete = useCallback(
    (ids: string[]) => {
      if (ids.length === 0) return
      void runOp(() => delHist.mutateAsync(ids))
    },
    [delHist, runOp]
  )

  // 导出当前筛选条件全量（对照 v1 分页拉全量导出）
  const handleExport = useCallback(async () => {
    setExporting(true)
    setOpError(null)
    try {
      const exported: Alarm[] = rows.slice()
      const headers = CSV_FIELDS.map((f) => f.label)
      const body = exported.map((a) => CSV_FIELDS.map((f) => f.get(a)))
      const csv = [headers, ...body]
        .map((row) => row.map((cell) => escapeCsvCell(cell)).join(','))
        .join('\n')
      const datePart = new Date().toISOString().slice(0, 10)
      triggerCsvDownload(csv, `historical-alarms-${datePart}.csv`)
    } catch (e) {
      setOpError(e instanceof Error ? e.message : '导出失败')
    } finally {
      setExporting(false)
    }
  }, [rows])

  return (
    <PageShell
      code="F04"
      title="HISTORICAL ALARMS · 历史告警"
      subtitle="CLEARED ALARM ARCHIVE · ACK / UNACK / PURGE"
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
          <NeonButton
            icon={<SlidersHorizontal />}
            onClick={() => setShowAdvanced((v) => !v)}
            className={showAdvanced ? 'shadow-[0_0_10px_currentColor]' : undefined}
          >
            FILTER
          </NeonButton>
          <NeonButton icon={<Download />} onClick={() => void handleExport()} disabled={exporting || rows.length === 0}>
            {exporting ? '导出中…' : 'EXPORT'}
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {/* 高级筛选行（对照 v1 FilterBar） */}
      {showAdvanced && (
        <div className="mb-3 grid grid-cols-2 gap-3 rounded-sm border border-cyan-500/15 bg-cyan-500/[0.03] p-3 md:grid-cols-3 lg:grid-cols-5">
          <FilterInput label="设备 SN" value={deviceSn} onChange={setDeviceSn} placeholder="精确/模糊 SN" />
          <FilterInput label="告警标识" value={alarmIdentifier} onChange={setAlarmIdentifier} placeholder="告警标识" />
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
              { v: '2', t: '未确认已清除' },
              { v: '3', t: '已确认已清除' },
            ]}
          />
          <div className="flex items-end">
            <NeonButton
              icon={<Search />}
              onClick={() => setPage(1)}
            >
              SEARCH
            </NeonButton>
          </div>
          <div className="flex items-end">
            <NeonButton icon={<RotateCcw />} onClick={resetFilters}>
              RESET
            </NeonButton>
          </div>
        </div>
      )}

      {/* 历史告警计数条 */}
      <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-4 lg:grid-cols-5">
        <SevStat label="TOTAL · 总计" color="#00f0ff" value={histCnt?.total_active ?? total} />
        <SevStat label="CRITICAL · 紧急" color={SEV_COLOR.critical} value={histCnt?.critical ?? 0} />
        <SevStat label="MAJOR · 重要" color={SEV_COLOR.major} value={histCnt?.major ?? 0} />
        <SevStat label="MINOR · 次要" color={SEV_COLOR.minor} value={histCnt?.minor ?? 0} />
        <SevStat label="WARNING · 警告" color={SEV_COLOR.warning} value={histCnt?.warning ?? 0} />
      </div>

      {/* 批量操作条 */}
      {selectedIds.length > 0 && (
        <div className="mb-2 flex flex-wrap items-center gap-2 rounded-sm border border-cyan-500/25 bg-cyan-500/[0.05] px-3 py-2">
          <span className="font-mono text-[11px] text-cyan-200">已选 {selectedIds.length} 条</span>
          <NeonButton icon={<Check />} onClick={() => setModal({ kind: 'ack', ids: selectedIds })}>
            确认
          </NeonButton>
          <NeonButton icon={<RotateCcw />} onClick={() => onUnack(selectedIds)}>
            取消确认
          </NeonButton>
          <NeonButton tone="danger" icon={<Trash2 />} onClick={() => onDelete(selectedIds)}>
            删除
          </NeonButton>
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
        <div className="mb-1 grid grid-cols-[24px_24px_1.7fr_2fr_0.9fr_1fr_140px_72px] items-center gap-3 px-3 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
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
          <span>CLEARED</span>
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
            <History className="size-10 text-cyan-300/40" />
            <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
              NO RECORDS · 无历史告警
            </div>
          </div>
        ) : (
          rows.map((a) => {
            const checked = selected.has(a.id)
            const confirmed = a.dealState === '1' || a.dealState === '3'
            return (
              <div
                key={a.id}
                className="fleet-row grid grid-cols-[24px_24px_1.7fr_2fr_0.9fr_1fr_140px_72px] items-center gap-3 rounded-sm px-3 py-2.5"
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
                <button type="button" className="min-w-0 text-left" onClick={() => setDetail(a)}>
                  <div className="truncate font-display text-sm font-bold text-cyan-100">
                    {a.alarmName || a.alarmIdentifier}
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
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">×{a.alarmCount ?? 1}</div>
                </div>
                <div className="font-mono text-[11px] text-cyan-300/75">{formatTime(a.clearTime)}</div>
                <div className="text-right">
                  <span className="chip" style={{ color: confirmed ? '#00ff88' : SEV_COLOR[a.severity] }}>
                    {DEAL_LABEL[a.dealState] ?? '—'}
                  </span>
                </div>
                <div className="flex items-center justify-end gap-1.5">
                  <RowAction
                    title="详情"
                    onClick={() => setDetail(a)}
                    icon={<ChevronDown className="size-3.5 -rotate-90" />}
                  />
                  {confirmed ? (
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
                  )}
                  <RowAction
                    title="删除"
                    danger
                    onClick={() => onDelete([a.id])}
                    icon={<Trash2 className="size-3.5" />}
                  />
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

      {/* 详情抽屉 */}
      <AlarmDetailPanel alarm={detail} open={detail !== null} onClose={() => setDetail(null)} />

      {/* 确认备注弹层 */}
      <ConfirmNoteModal
        open={modal.kind !== null}
        title="确认告警"
        message={`将确认 ${modal.ids.length} 条历史告警，可填写确认备注。`}
        confirmText="确认"
        loading={submitting}
        onConfirm={(note) => void confirmAck(note)}
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

function FilterInput({
  label,
  value,
  onChange,
  placeholder,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  placeholder?: string
}) {
  return (
    <label className="flex flex-col gap-1">
      <span className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">{label}</span>
      <input
        className="neon-input"
        value={value}
        placeholder={placeholder}
        onChange={(e) => onChange(e.target.value)}
      />
    </label>
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
      <span className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">{label}</span>
      <select className="neon-input" value={value} onChange={(e) => onChange(e.target.value)}>
        {options.map((o) => (
          <option key={o.v || 'all'} value={o.v} className="bg-[#03050d]">
            {o.t}
          </option>
        ))}
      </select>
    </label>
  )
}

function SevStat({ label, color, value }: { label: string; color: string; value: number }) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: color }}
    >
      <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/65">{label}</div>
      <div
        className="font-display text-2xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}
