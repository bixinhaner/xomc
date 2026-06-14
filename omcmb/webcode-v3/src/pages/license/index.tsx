import { useMemo, useState } from 'react'
import {
  KeyRound,
  ShieldCheck,
  CalendarClock,
  Cpu,
  RefreshCcw,
  CloudUpload,
  History,
  Loader2,
  AlertTriangle,
  Fingerprint,
  Boxes,
  Layers,
  Clock,
  Ticket,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { formatTime } from '@/lib/format'
import { useSystemLicense, useSystemLicenseHistory } from '@core/hooks/api/useSystemLicense'
import type {
  SystemLicense,
  SystemLicenseSignatureStatus,
} from '@core/services/api/systemLicenseApi'
import {
  SystemLicenseErrorCodes,
  extractLicenseErrorCode,
} from '@core/services/api/systemLicenseApi'

import { FeatureTree } from './FeatureTree'
import { UpdateLicenseModal } from './UpdateLicenseModal'
import { DeviceLicensePanel } from './DeviceLicensePanel'
import { Modal } from './Modal'

// ─────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────

function daysRemaining(expiryISO: string | null): number | null {
  if (!expiryISO) return null
  const ms = new Date(expiryISO).getTime() - Date.now()
  return Math.floor(ms / (1000 * 60 * 60 * 24))
}

const SIG_META: Record<
  SystemLicenseSignatureStatus,
  { label: string; color: string; status: string }
> = {
  verified: { label: '已验签', color: '#00ff88', status: 'ok' },
  invalid: { label: '签名无效', color: '#ff2d6f', status: 'critical' },
  unverified: { label: '未验签', color: '#ffaa00', status: 'warning' },
}

function countFeatures(featureList: Record<string, unknown>): number {
  return Object.keys(featureList ?? {}).length
}

function devicesTotal(devicesSupport: Record<string, number>): number {
  return Object.values(devicesSupport ?? {}).reduce((a, b) => a + (Number(b) || 0), 0)
}

// ─────────────────────────────────────────────────────────────────────────
// page
// ─────────────────────────────────────────────────────────────────────────

export function LicensePage() {
  const [updateOpen, setUpdateOpen] = useState(false)
  const [historyOpen, setHistoryOpen] = useState(false)
  const [notice, setNotice] = useState<{ msg: string; tone: 'ok' | 'err' } | null>(null)

  const { data: lic, isLoading, isFetching, error, refetch } = useSystemLicense()

  const errorCode = error ? extractLicenseErrorCode(error) : 0
  const isNotConfigured = errorCode === SystemLicenseErrorCodes.NotConfigured
  const realError = error && !isNotConfigured ? error : null

  const pushNotice = (msg: string, tone: 'ok' | 'err' = 'ok') => setNotice({ msg, tone })

  const toolbar = (
    <>
      <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
        REFRESH
      </NeonButton>
      <NeonButton icon={<History />} onClick={() => setHistoryOpen(true)}>
        HISTORY
      </NeonButton>
      <NeonButton icon={<CloudUpload />} onClick={() => setUpdateOpen(true)}>
        UPDATE
      </NeonButton>
    </>
  )

  return (
    <PageShell
      code="F06"
      title="LICENSE · 印玺管理"
      subtitle="SYSTEM AUTHORIZATION · CAPACITY · FEATURES · DEVICE LICENSES"
      isFetching={isFetching}
      bare
      toolbar={toolbar}
    >
      {/* 操作回执 banner */}
      {notice ? (
        <div
          className="mb-3 flex items-center justify-between rounded-sm border px-4 py-2.5"
          style={{
            borderColor: notice.tone === 'ok' ? '#00ff8855' : '#ff2d6f55',
            background: notice.tone === 'ok' ? '#00ff8810' : '#ff2d6f10',
          }}
        >
          <span
            className="font-mono text-xs"
            style={{ color: notice.tone === 'ok' ? '#00ff88' : '#ff2d6f' }}
          >
            {notice.tone === 'ok' ? '✓ ' : '✕ '}
            {notice.msg}
          </span>
          <button
            type="button"
            onClick={() => setNotice(null)}
            className="font-mono text-[11px] text-cyan-300/55 hover:text-cyan-200"
          >
            DISMISS
          </button>
        </div>
      ) : null}

      <div className="space-y-3">
        {/* ── 系统 license 概览 ────────────────────────────────────── */}
        {isLoading ? (
          <GlassPanel title="SYSTEM LICENSE · 系统授权" className="min-h-0">
            <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
              <Loader2 className="size-4 animate-spin" />
              <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
            </div>
          </GlassPanel>
        ) : realError ? (
          <GlassPanel title="SYSTEM LICENSE · 系统授权" className="min-h-0">
            <div className="m-4 flex items-start gap-2 rounded-sm border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
              <AlertTriangle className="mt-0.5 size-4 shrink-0" />
              <div>
                FAILURE · {realError instanceof Error ? realError.message : '未知错误'}
                <div className="mt-3">
                  <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
                    RETRY
                  </NeonButton>
                </div>
              </div>
            </div>
          </GlassPanel>
        ) : isNotConfigured || !lic ? (
          <NotConfigured onUpload={() => setUpdateOpen(true)} />
        ) : (
          <SystemLicenseOverview lic={lic} />
        )}

        {/* ── 设备 license 文件库 ───────────────────────────────────── */}
        <DeviceLicensePanel onNotice={pushNotice} />
      </div>

      {/* 弹窗 */}
      <UpdateLicenseModal
        open={updateOpen}
        onClose={() => setUpdateOpen(false)}
        onDone={(msg) => pushNotice(msg, 'ok')}
      />
      <HistoryModal open={historyOpen} onClose={() => setHistoryOpen(false)} />
    </PageShell>
  )
}

// ─────────────────────────────────────────────────────────────────────────
// 空态
// ─────────────────────────────────────────────────────────────────────────

function NotConfigured({ onUpload }: { onUpload: () => void }) {
  return (
    <GlassPanel title="SYSTEM LICENSE · 系统授权" className="min-h-0">
      <div className="flex flex-col items-center justify-center gap-3 py-16">
        <KeyRound className="size-12 text-cyan-300/30" />
        <div className="font-display text-lg text-cyan-100">尚未配置系统授权</div>
        <div className="max-w-md text-center font-mono text-xs text-cyan-300/55">
          上传由签发方提供的 license 文件以解锁容量与功能特性。后端将进行严格签名校验。
        </div>
        <NeonButton icon={<CloudUpload />} onClick={onUpload}>
          上传 LICENSE
        </NeonButton>
      </div>
    </GlassPanel>
  )
}

// ─────────────────────────────────────────────────────────────────────────
// 系统 license 概览（基本信息 + 容量仪表 + 设备支持 + 功能列表）
// ─────────────────────────────────────────────────────────────────────────

function SystemLicenseOverview({ lic }: { lic: SystemLicense }) {
  const remain = daysRemaining(lic.expiryDate)
  const sig = SIG_META[lic.signatureStatus]
  const devTotal = devicesTotal(lic.devicesSupport)
  const featCount = countFeatures(lic.featureList)

  // 到期进度环：默认授权窗 365 天，剩余比例（永久=100%）
  const expiryPct = useMemo(() => {
    if (lic.expiryDate == null) return 100
    if (remain == null) return 0
    if (remain <= 0) return 0
    return Math.min(100, (remain / 365) * 100)
  }, [lic.expiryDate, remain])

  const expiryColor =
    lic.expiryDate == null
      ? '#00f0ff'
      : remain == null || remain < 0
        ? '#ff2d6f'
        : remain <= 30
          ? '#ff7a1a'
          : remain <= 90
            ? '#ffaa00'
            : '#00ff88'

  const expiryText =
    lic.expiryDate == null
      ? '永久授权 · PERPETUAL'
      : remain != null && remain < 0
        ? `已过期 ${Math.abs(remain)} 天`
        : `剩余 ${remain ?? 0} 天 · 至 ${formatTime(lic.expiryDate)}`

  return (
    <>
      {/* 顶部四大数 */}
      <div className="grid grid-cols-4 gap-3">
        <StatCard
          icon={<Ticket className="size-4" />}
          label="授权类型 · TYPE"
          value={lic.licenseType}
          color="#00f0ff"
        />
        <StatCard
          icon={<CalendarClock className="size-4" />}
          label="到期 · EXPIRY"
          value={lic.expiryDate == null ? '永久' : remain != null && remain < 0 ? '已过期' : `${remain ?? 0}d`}
          color={expiryColor}
        />
        <StatCard
          icon={<Cpu className="size-4" />}
          label="设备容量 · CAPACITY"
          value={devTotal.toLocaleString()}
          color="#a855f7"
        />
        <StatCard
          icon={<Layers className="size-4" />}
          label="功能模块 · FEATURES"
          value={String(featCount)}
          color="#00ff88"
        />
      </div>

      {/* 基本信息 + 容量环 */}
      <div className="grid grid-cols-12 gap-3">
        <GlassPanel
          title="BASIC INFO · 基本信息"
          meta={<StatusDot color={sig.color} label={sig.label} />}
          className="col-span-8 min-h-0"
        >
          <div className="grid grid-cols-2 gap-px overflow-hidden bg-cyan-500/10">
            <InfoCell icon={<KeyRound className="size-3.5" />} label="LICENSE ID" value={lic.licenseId} mono />
            <InfoCell icon={<Ticket className="size-3.5" />} label="TYPE" value={lic.licenseType} />
            <InfoCell
              icon={<CalendarClock className="size-3.5" />}
              label="EXPIRY"
              value={expiryText}
            />
            <InfoCell
              icon={<ShieldCheck className="size-3.5" />}
              label="SIGNATURE"
              value={sig.label}
              valueColor={sig.color}
            />
            <InfoCell icon={<Boxes className="size-3.5" />} label="ISSUER" value={lic.issuer ?? '—'} />
            <InfoCell icon={<Boxes className="size-3.5" />} label="LICENSEE" value={lic.licensee ?? '—'} />
            <InfoCell icon={<Clock className="size-3.5" />} label="ISSUED AT" value={formatTime(lic.issuedAt)} mono />
            <InfoCell
              icon={<CloudUpload className="size-3.5" />}
              label="UPLOADED AT"
              value={formatTime(lic.uploadedAt)}
              mono
            />
            <InfoCell
              icon={<Fingerprint className="size-3.5" />}
              label="SIGNATURE KEY"
              value={lic.signatureKeyId ? `${lic.signatureKeyId.slice(0, 16)}…` : '—'}
              mono
            />
            <InfoCell
              icon={<KeyRound className="size-3.5" />}
              label="LICENSE PK"
              value={`${lic.id.slice(0, 16)}…`}
              mono
            />
          </div>
        </GlassPanel>

        {/* 到期 / 容量环 */}
        <GlassPanel title="VALIDITY · 有效期" className="col-span-4 min-h-0">
          <div className="flex flex-col items-center justify-center gap-3 py-5">
            <RadialGauge
              value={expiryPct}
              label={lic.expiryDate == null ? 'PERPETUAL' : 'REMAINING'}
              size={140}
              color={expiryColor}
              unit={lic.expiryDate == null ? '∞' : '%'}
            />
            <div className="text-center">
              <div className="font-mono text-xs" style={{ color: expiryColor }}>
                {expiryText}
              </div>
              {lic.expiryDate != null && remain != null && remain >= 0 && remain <= 90 ? (
                <div className="mt-1 inline-flex items-center gap-1 font-mono text-[10px] uppercase tracking-[0.18em] text-amber-300/85">
                  <AlertTriangle className="size-3" />
                  续期预警 · RENEWAL ALERT
                </div>
              ) : null}
            </div>
          </div>
        </GlassPanel>
      </div>

      {/* 设备支持 */}
      <GlassPanel title="DEVICES SUPPORT · 设备类型配额" meta={`TOTAL ${devTotal.toLocaleString()}`}>
        <DevicesSupport devicesSupport={lic.devicesSupport} />
      </GlassPanel>

      {/* 功能列表 */}
      <GlassPanel title="FEATURE MATRIX · 功能授权矩阵" meta={`${featCount} MODULES`}>
        <FeatureTree featureList={lic.featureList} />
      </GlassPanel>
    </>
  )
}

function DevicesSupport({ devicesSupport }: { devicesSupport: SystemLicense['devicesSupport'] }) {
  const entries = useMemo(
    () => Object.entries(devicesSupport ?? {}).sort(([a], [b]) => a.localeCompare(b)),
    [devicesSupport],
  )
  if (entries.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-10">
        <Cpu className="size-8 text-cyan-300/30" />
        <div className="font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/50">
          NO DEVICE QUOTA · 无设备配额
        </div>
      </div>
    )
  }
  return (
    <div className="grid grid-cols-2 gap-3 p-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6">
      {entries.map(([deviceType, capacity]) => (
        <div
          key={deviceType}
          className="glass relative overflow-hidden rounded-sm border-l-2 border-l-cyan-400/60 px-3 py-2.5"
        >
          <div className="truncate font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/65">
            {deviceType}
          </div>
          <div
            className="font-display text-2xl font-bold leading-tight text-cyan-100"
            style={{ textShadow: '0 0 8px rgba(0,240,255,0.4)' }}
          >
            {Number(capacity).toLocaleString()}
          </div>
        </div>
      ))}
    </div>
  )
}

// ─────────────────────────────────────────────────────────────────────────
// 历史弹窗
// ─────────────────────────────────────────────────────────────────────────

function HistoryModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [page, setPage] = useState(1)
  const pageSize = 10
  const params = useMemo(() => ({ page, pageSize }), [page])
  // open 时才查（hook 一直在，但翻页参数受控；关闭后仍缓存无副作用）
  const { data, isLoading, isError, error, isFetching } = useSystemLicenseHistory(params)

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <Modal
      open={open}
      title="LICENSE HISTORY · 授权历史"
      subtitle={isFetching ? 'SYNC… · ORDERED BY REPLACED_AT DESC' : 'ORDERED BY REPLACED_AT DESC'}
      onClose={onClose}
      width={820}
      footer={<NeonButton onClick={onClose}>关闭</NeonButton>}
    >
      {isLoading ? (
        <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
        </div>
      ) : isError ? (
        <div className="flex items-start gap-2 rounded-sm border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
          <AlertTriangle className="mt-0.5 size-4 shrink-0" />
          FAILURE · {error instanceof Error ? error.message : '未知错误'}
        </div>
      ) : rows.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-2 py-12">
          <History className="size-8 text-cyan-300/30" />
          <div className="font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/50">
            NO HISTORY · 无历史记录
          </div>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-[1.4fr_0.9fr_0.9fr_1fr_1fr] gap-3 border-b border-cyan-500/15 px-2 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            <span>LICENSE ID</span>
            <span>TYPE</span>
            <span>SIGNATURE</span>
            <span>UPLOADED</span>
            <span>REPLACED</span>
          </div>
          {rows.map((r) => {
            const sig = SIG_META[r.signatureStatus]
            return (
              <div
                key={r.id}
                className="grid grid-cols-[1.4fr_0.9fr_0.9fr_1fr_1fr] items-center gap-3 border-b border-cyan-500/8 px-2 py-2.5"
              >
                <span className="truncate font-mono text-xs text-cyan-100">{r.licenseId}</span>
                <span className="font-mono text-[11px] text-cyan-300/75">{r.licenseType}</span>
                <span className="font-mono text-[11px]" style={{ color: sig.color }}>
                  {sig.label}
                </span>
                <span className="font-mono text-[11px] text-cyan-300/70">
                  {formatTime(r.uploadedAt)}
                </span>
                <span className="font-mono text-[11px] text-cyan-300/70">
                  {formatTime(r.replacedAt)}
                </span>
              </div>
            )
          })}
          <div className="mt-3 flex items-center justify-between">
            <span className="font-mono text-[11px] text-cyan-300/55">
              PAGE {page} / {totalPages} · TOTAL {total}
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
        </>
      )}
    </Modal>
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
        <div className="flex flex-1 flex-col">
          <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
            {label}
          </div>
          <div
            className="truncate font-display text-2xl font-bold leading-tight text-glow"
            style={{ color }}
          >
            {value}
          </div>
        </div>
      </div>
    </div>
  )
}

function InfoCell({
  icon,
  label,
  value,
  mono,
  valueColor,
}: {
  icon: React.ReactNode
  label: string
  value: string
  mono?: boolean
  valueColor?: string
}) {
  return (
    <div className="bg-[#070b18] px-3.5 py-2.5">
      <div className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
        <span className="text-cyan-300/50">{icon}</span>
        {label}
      </div>
      <div
        className={`mt-0.5 truncate text-sm ${mono ? 'font-mono text-xs' : ''}`}
        style={{ color: valueColor ?? '#cfe7ff' }}
        title={value}
      >
        {value}
      </div>
    </div>
  )
}

function StatusDot({ color, label }: { color: string; label: string }) {
  return (
    <span className="inline-flex items-center gap-1.5 font-mono text-[10px]" style={{ color }}>
      <span
        className="size-1.5 rounded-full"
        style={{ background: color, boxShadow: `0 0 8px ${color}` }}
      />
      {label}
    </span>
  )
}
