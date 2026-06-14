import { useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  History as HistoryIcon,
  KeyRound,
  Loader2,
  AlertTriangle,
  RefreshCcw,
  ArrowLeft,
  Filter,
  Layers,
  Cpu,
  Archive,
  ShieldCheck,
  Clock,
  CalendarClock,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatTime } from '@/lib/format'
import { useSystemLicenseHistory } from '@core/hooks/api/useSystemLicense'
import type {
  SystemLicenseHistory,
  SystemLicenseSignatureStatus,
} from '@core/services/api/systemLicenseApi'

// ─────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────

const SIG_META: Record<
  SystemLicenseSignatureStatus,
  { label: string; color: string }
> = {
  verified: { label: '已验签', color: '#00ff88' },
  invalid: { label: '签名无效', color: '#ff2d6f' },
  unverified: { label: '未验签', color: '#ffaa00' },
}

const PAGE_SIZE = 20

function devicesTotal(devicesSupport: Record<string, number>): number {
  return Object.values(devicesSupport ?? {}).reduce((a, b) => a + (Number(b) || 0), 0)
}

function countFeatures(featureList: Record<string, unknown>): number {
  return Object.keys(featureList ?? {}).length
}

// ─────────────────────────────────────────────────────────────────────────
// page · /license/history
// ─────────────────────────────────────────────────────────────────────────

export default function LicenseHistoryPage() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const [page, setPage] = useState(1)

  // URL ?licenseId=… 用于「按某条 license 归档钻取」。空 = 全部。
  const licenseFilter = searchParams.get('licenseId') ?? ''

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(licenseFilter ? { licenseId: licenseFilter } : {}),
    }),
    [page, licenseFilter],
  )

  const { data, isLoading, isFetching, isError, error, refetch } =
    useSystemLicenseHistory(params)

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const clearFilter = () => {
    setPage(1)
    const next = new URLSearchParams(searchParams)
    next.delete('licenseId')
    setSearchParams(next, { replace: true })
  }

  const applyFilter = (licenseId: string) => {
    setPage(1)
    const next = new URLSearchParams(searchParams)
    next.set('licenseId', licenseId)
    setSearchParams(next, { replace: true })
  }

  const toolbar = (
    <>
      <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/license')}>
        BACK
      </NeonButton>
      <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
        REFRESH
      </NeonButton>
    </>
  )

  return (
    <PageShell
      code="F06"
      title="LICENSE HISTORY · 授权历史"
      subtitle="ARCHIVED SYSTEM LICENSES · ORDERED BY REPLACED_AT DESC"
      isFetching={isFetching}
      bare
      toolbar={toolbar}
    >
      <div className="space-y-3">
        {/* 过滤回执条 */}
        {licenseFilter ? (
          <div
            className="flex items-center justify-between rounded-sm border px-4 py-2.5"
            style={{ borderColor: '#00f0ff44', background: '#00f0ff0d' }}
          >
            <span className="flex items-center gap-2 font-mono text-xs text-cyan-200">
              <Filter className="size-3.5 text-cyan-300/70" />
              已按 LICENSE ID 过滤 ·{' '}
              <span className="text-cyan-100">{licenseFilter}</span>
            </span>
            <button
              type="button"
              onClick={clearFilter}
              className="font-mono text-[11px] uppercase tracking-[0.18em] text-cyan-300/55 transition-colors hover:text-cyan-200"
            >
              CLEAR ✕
            </button>
          </div>
        ) : null}

        {/* 顶部统计 */}
        <SummaryStrip total={total} rows={rows} filtered={Boolean(licenseFilter)} />

        {/* 历史列表 */}
        <GlassPanel
          title="ARCHIVE LEDGER · 归档台账"
          meta={`TOTAL ${total.toLocaleString()}`}
        >
          {isLoading ? (
            <LoadingState />
          ) : isError ? (
            <ErrorState
              message={error instanceof Error ? error.message : '未知错误'}
              onRetry={() => refetch()}
            />
          ) : rows.length === 0 ? (
            <EmptyState filtered={Boolean(licenseFilter)} onClear={clearFilter} />
          ) : (
            <HistoryTable
              rows={rows}
              activeId={licenseFilter}
              onFilter={applyFilter}
              page={page}
              totalPages={totalPages}
              total={total}
              onPrev={() => setPage((p) => Math.max(1, p - 1))}
              onNext={() => setPage((p) => Math.min(totalPages, p + 1))}
            />
          )}
        </GlassPanel>
      </div>
    </PageShell>
  )
}

// ─────────────────────────────────────────────────────────────────────────
// 顶部统计带
// ─────────────────────────────────────────────────────────────────────────

function SummaryStrip({
  total,
  rows,
  filtered,
}: {
  total: number
  rows: SystemLicenseHistory[]
  filtered: boolean
}) {
  const verified = rows.filter((r) => r.signatureStatus === 'verified').length
  const latestReplaced = rows.length > 0 ? rows[0].replacedAt : null

  return (
    <div className="grid grid-cols-4 gap-3">
      <StatCard
        icon={<Archive className="size-4" />}
        label={filtered ? '匹配归档 · MATCHED' : '归档总数 · TOTAL'}
        value={total.toLocaleString()}
        color="#00f0ff"
      />
      <StatCard
        icon={<ShieldCheck className="size-4" />}
        label="本页已验签 · VERIFIED"
        value={`${verified}/${rows.length}`}
        color="#00ff88"
      />
      <StatCard
        icon={<Layers className="size-4" />}
        label="本页记录 · ON PAGE"
        value={String(rows.length)}
        color="#a855f7"
      />
      <StatCard
        icon={<Clock className="size-4" />}
        label="最近替换 · LAST REPLACED"
        value={latestReplaced ? formatTime(latestReplaced) : '—'}
        color="#ffaa00"
      />
    </div>
  )
}

// ─────────────────────────────────────────────────────────────────────────
// 表格
// ─────────────────────────────────────────────────────────────────────────

const GRID = 'grid grid-cols-[1.5fr_0.8fr_0.9fr_0.7fr_0.7fr_1.1fr_1.1fr] gap-3'

function HistoryTable({
  rows,
  activeId,
  onFilter,
  page,
  totalPages,
  total,
  onPrev,
  onNext,
}: {
  rows: SystemLicenseHistory[]
  activeId: string
  onFilter: (id: string) => void
  page: number
  totalPages: number
  total: number
  onPrev: () => void
  onNext: () => void
}) {
  return (
    <div className="p-3">
      {/* 表头 */}
      <div
        className={`${GRID} border-b border-cyan-500/15 px-2 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55`}
      >
        <span>LICENSE ID</span>
        <span>TYPE</span>
        <span>SIGNATURE</span>
        <span>DEVICES</span>
        <span>FEATURES</span>
        <span>UPLOADED</span>
        <span>REPLACED</span>
      </div>

      {/* 行 */}
      {rows.map((r) => {
        const sig = SIG_META[r.signatureStatus]
        const active = activeId === r.licenseId
        return (
          <button
            key={r.id}
            type="button"
            onClick={() => onFilter(r.licenseId)}
            title="按此 LICENSE ID 过滤归档"
            className={`${GRID} w-full items-center border-b border-cyan-500/8 px-2 py-2.5 text-left transition-colors hover:bg-cyan-500/5 ${
              active ? 'bg-cyan-500/8' : ''
            }`}
          >
            <span className="flex min-w-0 items-center gap-1.5">
              <KeyRound className="size-3 shrink-0 text-cyan-300/50" />
              <span className="truncate font-mono text-xs text-cyan-100">
                {r.licenseId}
              </span>
            </span>
            <span className="font-mono text-[11px] text-cyan-300/75">
              {r.licenseType}
            </span>
            <span
              className="inline-flex items-center gap-1.5 font-mono text-[11px]"
              style={{ color: sig.color }}
            >
              <span
                className="size-1.5 rounded-full"
                style={{ background: sig.color, boxShadow: `0 0 8px ${sig.color}` }}
              />
              {sig.label}
            </span>
            <span className="flex items-center gap-1 font-mono text-[11px] text-cyan-300/70">
              <Cpu className="size-3 text-cyan-300/40" />
              {devicesTotal(r.devicesSupport).toLocaleString()}
            </span>
            <span className="flex items-center gap-1 font-mono text-[11px] text-cyan-300/70">
              <Layers className="size-3 text-cyan-300/40" />
              {countFeatures(r.featureList)}
            </span>
            <span className="flex items-center gap-1 font-mono text-[11px] text-cyan-300/70">
              <CalendarClock className="size-3 text-cyan-300/40" />
              {formatTime(r.uploadedAt)}
            </span>
            <span className="flex items-center gap-1 font-mono text-[11px] text-cyan-300/70">
              <Clock className="size-3 text-cyan-300/40" />
              {formatTime(r.replacedAt)}
            </span>
          </button>
        )
      })}

      {/* 分页 */}
      <div className="mt-3 flex items-center justify-between">
        <span className="font-mono text-[11px] text-cyan-300/55">
          PAGE {page} / {totalPages} · TOTAL {total.toLocaleString()}
        </span>
        <div className="flex gap-2">
          <NeonButton onClick={onPrev} disabled={page <= 1}>
            ◂ PREV
          </NeonButton>
          <NeonButton onClick={onNext} disabled={page >= totalPages}>
            NEXT ▸
          </NeonButton>
        </div>
      </div>
    </div>
  )
}

// ─────────────────────────────────────────────────────────────────────────
// 三态
// ─────────────────────────────────────────────────────────────────────────

function LoadingState() {
  return (
    <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
      <Loader2 className="size-4 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
    </div>
  )
}

function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="m-4 flex items-start gap-2 rounded-sm border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
      <AlertTriangle className="mt-0.5 size-4 shrink-0" />
      <div>
        FAILURE · {message}
        <div className="mt-3">
          <NeonButton icon={<RefreshCcw />} onClick={onRetry}>
            RETRY
          </NeonButton>
        </div>
      </div>
    </div>
  )
}

function EmptyState({ filtered, onClear }: { filtered: boolean; onClear: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16">
      <HistoryIcon className="size-10 text-cyan-300/30" />
      <div className="font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/50">
        {filtered ? 'NO MATCH · 无匹配归档' : 'NO HISTORY · 无历史记录'}
      </div>
      {filtered ? (
        <NeonButton icon={<Filter />} onClick={onClear}>
          清除过滤
        </NeonButton>
      ) : (
        <div className="max-w-md text-center font-mono text-xs text-cyan-300/45">
          每次上传新 license 覆盖当前授权时，被替换的旧 license 会归档到这里。
        </div>
      )}
    </div>
  )
}

// ─────────────────────────────────────────────────────────────────────────
// small bits
// ─────────────────────────────────────────────────────────────────────────

function StatCard({
  icon,
  label,
  value,
  color,
}: {
  icon: React.ReactNode
  label: string
  value: string
  color: string
}) {
  return (
    <div className="glass relative overflow-hidden rounded-sm">
      <div className="scanline" />
      <div className="relative flex items-stretch gap-3 p-4">
        <div className="flex items-center justify-center border-r border-cyan-500/15 pr-3">
          <span style={{ color }}>{icon}</span>
        </div>
        <div className="flex min-w-0 flex-1 flex-col">
          <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
            {label}
          </div>
          <div
            className="truncate font-display text-xl font-bold leading-tight text-glow"
            style={{ color }}
            title={value}
          >
            {value}
          </div>
        </div>
      </div>
    </div>
  )
}
