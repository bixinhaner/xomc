import { useCallback, useMemo, useState } from 'react'
import {
  AlertTriangle,
  Loader2,
  Power,
  RefreshCcw,
  Search,
  ShieldCheck,
  Trash2,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import {
  useAlarmRules,
  useUpdateAlarmRule,
  useDeleteAlarmRules,
} from '@core/hooks/api/useAlarms'
import type { AlarmRule } from '@core/types/alarm'

const PAGE_SIZE = 20

const RULE_TYPE_LABEL: Record<string, { label: string; color: string }> = {
  default: { label: '默认', color: '#5b9eff' },
  ignore: { label: '禁止上报', color: '#ff2d6f' },
  auto_acknowledge: { label: '自动确认', color: '#00ff88' },
  auto_clear: { label: '自动清除', color: '#ff7a1a' },
}

const ENABLED_OPTIONS: { v: string; t: string }[] = [
  { v: '', t: '全部' },
  { v: 'true', t: '已启用' },
  { v: 'false', t: '已禁用' },
]

export default function AlarmRules() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [enabled, setEnabled] = useState('')
  const [busyId, setBusyId] = useState<string | null>(null)
  const [opError, setOpError] = useState<string | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(enabled ? { enabled } : {}),
    }),
    [page, keyword, enabled]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useAlarmRules(params)
  const updateRule = useUpdateAlarmRule()
  const deleteRules = useDeleteAlarmRules()

  const rules = useMemo(() => data?.items ?? [], [data])
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const enabledCount = useMemo(() => rules.filter((r) => r.enabled).length, [rules])

  const onToggle = useCallback(
    async (rule: AlarmRule) => {
      setBusyId(rule.id)
      setOpError(null)
      try {
        await updateRule.mutateAsync({ id: rule.id, data: { enabled: !rule.enabled } })
        await refetch()
      } catch (e) {
        setOpError(e instanceof Error ? e.message : '切换状态失败')
      } finally {
        setBusyId(null)
      }
    },
    [updateRule, refetch]
  )

  const onDelete = useCallback(
    async (rule: AlarmRule) => {
      if (rule.isDefault) {
        setOpError('默认规则不可删除')
        return
      }
      if (rule.enabled) {
        setOpError('请先禁用规则再删除')
        return
      }
      setBusyId(rule.id)
      setOpError(null)
      try {
        await deleteRules.mutateAsync([rule.id])
        await refetch()
      } catch (e) {
        setOpError(e instanceof Error ? e.message : '删除失败')
      } finally {
        setBusyId(null)
      }
    },
    [deleteRules, refetch]
  )

  return (
    <PageShell
      code="F04"
      title="ALARM FILTER RULES · 告警规则"
      subtitle="SUPPRESS · AUTO-ACK · AUTO-CLEAR POLICIES"
      isFetching={isFetching}
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="规则名称"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          {ENABLED_OPTIONS.map((o) => (
            <button
              key={o.v || 'all'}
              type="button"
              onClick={() => {
                setEnabled(o.v)
                setPage(1)
              }}
              className={`chip transition-all ${
                enabled === o.v
                  ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                  : 'text-cyan-300/45 hover:text-cyan-300/80'
              }`}
            >
              {o.t}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-3 gap-3">
        <Stat label="RULES · 规则总数" color="#00f0ff" value={total} />
        <Stat label="ENABLED · 本页启用" color="#00ff88" value={enabledCount} />
        <Stat label="DISABLED · 本页禁用" color="#525a78" value={rules.length - enabledCount} />
      </div>

      {opError && (
        <div className="mb-2 border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">
          OP FAILED · {opError}
        </div>
      )}

      <GlassPanel title="POLICY MATRIX · 规则列表" meta={`PAGE ${page}/${totalPages}`}>
        {/* 表头 */}
        {rules.length > 0 && (
          <div className="grid grid-cols-[2fr_1fr_1fr_1.2fr_120px] items-center gap-3 border-b border-cyan-500/10 px-3.5 py-2 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
            <span>RULE · 名称</span>
            <span>ACTION · 动作</span>
            <span>STATE · 状态</span>
            <span>UPDATED · 更新时间</span>
            <span className="text-right">OPS · 操作</span>
          </div>
        )}

        <div>
          {isLoading ? (
            <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
              <Loader2 className="size-4 animate-spin" />
              <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
            </div>
          ) : isError ? (
            <div className="flex flex-col items-center gap-3 px-4 py-10">
              <AlertTriangle className="size-7 text-rose-400/70" />
              <div className="font-mono text-sm text-rose-300">
                FAILURE · {error instanceof Error ? error.message : '未知错误'}
              </div>
              <NeonButton tone="danger" icon={<RefreshCcw />} onClick={() => refetch()}>
                RETRY
              </NeonButton>
            </div>
          ) : rules.length === 0 ? (
            <div className="flex flex-col items-center justify-center gap-3 py-14">
              <ShieldCheck className="size-9 text-cyan-300/40" />
              <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/45">
                NO RULES · 暂无规则
              </div>
            </div>
          ) : (
            rules.map((r) => {
              const typeCfg = RULE_TYPE_LABEL[r.ruleType] ?? {
                label: r.ruleType || '—',
                color: '#5b9eff',
              }
              const busy = busyId === r.id
              return (
                <div
                  key={r.id}
                  className="grid grid-cols-[2fr_1fr_1fr_1.2fr_120px] items-center gap-3 border-b border-cyan-500/8 px-3.5 py-2.5 last:border-b-0 hover:bg-cyan-500/5"
                >
                  <div className="min-w-0 text-left">
                    <div className="flex items-center gap-1.5">
                      {r.isDefault && (
                        <span className="chip text-[#5b9eff]">默认</span>
                      )}
                      <span className="truncate font-display text-sm font-bold text-cyan-100">
                        {r.ruleName}
                      </span>
                    </div>
                    <div className="truncate font-mono text-[10px] text-cyan-300/50">
                      {r.conditions.length} 条件 · {r.actions.length} 动作
                      {r.userCode ? ` · ${r.userCode}` : ''}
                    </div>
                  </div>
                  <span
                    className="chip"
                    style={{ color: typeCfg.color }}
                  >
                    {typeCfg.label}
                  </span>
                  <span>
                    <StatusBadge
                      status={r.enabled ? 'active' : 'inactive'}
                      label={r.enabled ? '启用' : '禁用'}
                    />
                  </span>
                  <span className="font-mono text-[11px] text-cyan-300/70">
                    {formatTime(r.updateTime)}
                  </span>
                  <div className="flex items-center justify-end gap-1.5">
                    {busy ? (
                      <Loader2 className="size-4 animate-spin text-cyan-300/70" />
                    ) : (
                      <>
                        <RowAction
                          title={r.enabled ? '禁用' : '启用'}
                          onClick={() => void onToggle(r)}
                          icon={<Power className="size-3.5" />}
                          active={r.enabled}
                        />
                        <RowAction
                          title="删除"
                          danger
                          disabled={r.isDefault || r.enabled}
                          onClick={() => void onDelete(r)}
                          icon={<Trash2 className="size-3.5" />}
                        />
                      </>
                    )}
                  </div>
                </div>
              )
            })
          )}
        </div>
      </GlassPanel>

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
    </PageShell>
  )
}

function Stat({ label, color, value }: { label: string; color: string; value: number }) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: color }}
    >
      <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/65">
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

function RowAction({
  title,
  icon,
  danger,
  active,
  disabled,
  onClick,
}: {
  title: string
  icon: React.ReactNode
  danger?: boolean
  active?: boolean
  disabled?: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      title={title}
      onClick={onClick}
      disabled={disabled}
      className={`flex size-6 items-center justify-center rounded-sm border transition-colors ${
        disabled
          ? 'cursor-not-allowed border-cyan-500/10 text-cyan-300/25'
          : danger
            ? 'border-rose-500/30 text-rose-300/80 hover:border-rose-400/70 hover:text-rose-200'
            : active
              ? 'border-emerald-500/40 text-emerald-300/90 hover:border-emerald-400/70'
              : 'border-cyan-500/25 text-cyan-300/75 hover:border-cyan-400/60 hover:text-cyan-100'
      }`}
    >
      {icon}
    </button>
  )
}
