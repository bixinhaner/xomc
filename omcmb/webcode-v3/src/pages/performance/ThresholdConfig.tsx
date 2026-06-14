import { useMemo, useState } from 'react'
import { AlertTriangle, Loader2, Plus, RefreshCcw, Trash2, X } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { cn } from '@/lib/utils'
import {
  useThresholds,
  useCreateThreshold,
  useUpdateThreshold,
  useDeleteThresholds,
} from '@core/hooks/api/usePerformance'
import type { PerformanceThreshold } from '@core/types/performance'

/**
 * F03 · KPI 门限配置（performance/threshold）
 * 走真实 useThresholds 列表 + create/update/delete 三套 mutation。
 * 列表三态完整；新增/编辑用 HUD 弹层；启用开关 + 删除即调真实接口。
 */

const OPERATORS: { value: PerformanceThreshold['operator']; label: string }[] = [
  { value: 'gt', label: '>' },
  { value: 'gte', label: '≥' },
  { value: 'lt', label: '<' },
  { value: 'lte', label: '≤' },
  { value: 'eq', label: '=' },
  { value: 'ne', label: '≠' },
]

const PAGE_SIZE = 20

interface FormState {
  thresholdName: string
  kpiCode: string
  kpiName: string
  operator: PerformanceThreshold['operator']
  warningValue: string
  criticalValue: string
  unit: string
  enabled: boolean
}

const EMPTY_FORM: FormState = {
  thresholdName: '',
  kpiCode: '',
  kpiName: '',
  operator: 'lt',
  warningValue: '',
  criticalValue: '',
  unit: '%',
  enabled: true,
}

export default function ThresholdConfig() {
  const [page, setPage] = useState(1)
  const [editing, setEditing] = useState<PerformanceThreshold | null>(null)
  const [modalOpen, setModalOpen] = useState(false)
  const [form, setForm] = useState<FormState>(EMPTY_FORM)

  const { data, isLoading, isError, error, isFetching, refetch } = useThresholds({ page, pageSize: PAGE_SIZE })
  const createMut = useCreateThreshold()
  const updateMut = useUpdateThreshold()
  const deleteMut = useDeleteThresholds()

  const rows = data?.items ?? []
  const total = data?.total ?? rows.length
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const enabledCount = useMemo(() => rows.filter((r) => r.enabled).length, [rows])

  const openCreate = () => {
    setEditing(null)
    setForm(EMPTY_FORM)
    setModalOpen(true)
  }

  const openEdit = (r: PerformanceThreshold) => {
    setEditing(r)
    setForm({
      thresholdName: r.thresholdName,
      kpiCode: r.kpiCode,
      kpiName: r.kpiName,
      operator: r.operator,
      warningValue: String(r.warningValue ?? ''),
      criticalValue: String(r.criticalValue ?? ''),
      unit: r.unit,
      enabled: r.enabled,
    })
    setModalOpen(true)
  }

  const canSave = form.thresholdName.trim() && form.kpiCode.trim()

  const submit = () => {
    if (!canSave) return
    const payload = {
      thresholdName: form.thresholdName.trim(),
      kpiCode: form.kpiCode.trim(),
      kpiName: form.kpiName.trim() || form.kpiCode.trim(),
      operator: form.operator,
      warningValue: Number(form.warningValue) || 0,
      criticalValue: Number(form.criticalValue) || 0,
      unit: form.unit,
      enabled: form.enabled,
      deviceGroups: editing?.deviceGroups ?? [],
    }
    if (editing) {
      updateMut.mutate(
        { id: editing.id, data: payload },
        { onSuccess: () => setModalOpen(false) },
      )
    } else {
      createMut.mutate(payload, { onSuccess: () => setModalOpen(false) })
    }
  }

  const toggleEnabled = (r: PerformanceThreshold) => {
    updateMut.mutate({ id: r.id, data: { enabled: !r.enabled } })
  }

  const remove = (r: PerformanceThreshold) => {
    deleteMut.mutate([r.id])
  }

  const saving = createMut.isPending || updateMut.isPending

  return (
    <PageShell
      code="F03"
      title="THRESHOLD CONFIG · 门限配置"
      subtitle="KPI ALARM THRESHOLD · WARN / CRIT GATE"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <NeonButton icon={<Plus />} onClick={openCreate}>
            新增门限
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-3">
        <Stat label="THRESHOLDS · 门限总数" value={String(total)} color="#00f0ff" />
        <Stat label="ENABLED · 启用中" value={String(enabledCount)} color="#00ff88" />
        <Stat label="DISABLED · 停用" value={String(rows.length - enabledCount)} color="#ffaa00" />
      </div>

      <GlassPanel strong title="THRESHOLD RULES · 门限规则" meta={`${total} RULES`}>
        <div className="p-3">
          <div className="grid grid-cols-[2fr_1.2fr_70px_1fr_1fr_90px_120px] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            <span>NAME · 名称</span>
            <span>KPI · 编码</span>
            <span>OP</span>
            <span>WARN · 预警</span>
            <span>CRIT · 严重</span>
            <span>STATUS</span>
            <span className="text-right">ACTIONS</span>
          </div>

          <div className="mt-1.5 space-y-1">
            {isLoading ? (
              <Loading text="SYNCING THRESHOLDS…" />
            ) : isError ? (
              <ErrorBox text={`LOAD FAILED · ${error instanceof Error ? error.message : '未知错误'}`} />
            ) : rows.length === 0 ? (
              <EmptyBox text="NO THRESHOLD · 暂无门限" />
            ) : (
              rows.map((r) => (
                <div
                  key={r.id}
                  className="grid grid-cols-[2fr_1.2fr_70px_1fr_1fr_90px_120px] items-center gap-3 rounded-sm px-3 py-2 transition-colors hover:bg-cyan-500/5"
                >
                  <span className="truncate font-display text-sm font-bold text-cyan-100" title={r.thresholdName}>
                    {r.thresholdName}
                  </span>
                  <span className="truncate font-mono text-[11px] text-cyan-300/75" title={r.kpiName || r.kpiCode}>
                    {r.kpiCode}
                  </span>
                  <span className="font-mono text-[12px] text-cyan-200">
                    {OPERATORS.find((o) => o.value === r.operator)?.label ?? r.operator}
                  </span>
                  <span className="font-mono text-[12px] text-[#ffd400]">
                    {r.warningValue} {r.unit}
                  </span>
                  <span className="font-mono text-[12px] text-[#ff2d6f]">
                    {r.criticalValue} {r.unit}
                  </span>
                  <button type="button" onClick={() => toggleEnabled(r)} className="w-fit" title="切换启用">
                    <StatusBadge status={r.enabled ? 'ok' : 'off'} label={r.enabled ? '启用' : '停用'} />
                  </button>
                  <div className="flex justify-end gap-1.5">
                    <NeonButton onClick={() => openEdit(r)}>编辑</NeonButton>
                    <NeonButton tone="danger" icon={<Trash2 />} disabled={deleteMut.isPending} onClick={() => remove(r)}>
                      DEL
                    </NeonButton>
                  </div>
                </div>
              ))
            )}
          </div>

          <div className="mt-3 flex items-center justify-between">
            <span className="font-mono text-[11px] text-cyan-300/55">
              PAGE {page} / {totalPages} · TOTAL {total}
            </span>
            <div className="flex gap-2">
              <NeonButton onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
                ◂ PREV
              </NeonButton>
              <NeonButton onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages}>
                NEXT ▸
              </NeonButton>
            </div>
          </div>
        </div>
      </GlassPanel>

      {modalOpen ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm">
          <GlassPanel
            strong
            className="w-full max-w-lg"
            title={editing ? 'EDIT THRESHOLD · 编辑门限' : 'NEW THRESHOLD · 新增门限'}
            meta={<AlertTriangle className="size-3.5 text-[#ffaa00]" />}
          >
            <button
              type="button"
              onClick={() => setModalOpen(false)}
              className="absolute right-3 top-2.5 text-cyan-300/55 hover:text-cyan-100"
              aria-label="close"
            >
              <X className="size-4" />
            </button>
            <div className="space-y-3 p-4">
              <Field label="门限名称 · NAME">
                <input
                  className="neon-input w-full"
                  value={form.thresholdName}
                  onChange={(e) => setForm((f) => ({ ...f, thresholdName: e.target.value }))}
                  placeholder="如：RRC 成功率低告警"
                />
              </Field>
              <div className="grid grid-cols-2 gap-3">
                <Field label="KPI 编码 · CODE">
                  <input
                    className="neon-input w-full"
                    value={form.kpiCode}
                    onChange={(e) => setForm((f) => ({ ...f, kpiCode: e.target.value }))}
                    placeholder="RRC_SR"
                  />
                </Field>
                <Field label="KPI 名称">
                  <input
                    className="neon-input w-full"
                    value={form.kpiName}
                    onChange={(e) => setForm((f) => ({ ...f, kpiName: e.target.value }))}
                    placeholder="RRC 建立成功率"
                  />
                </Field>
              </div>
              <div className="grid grid-cols-3 gap-3">
                <Field label="比较符 · OP">
                  <div className="flex flex-wrap gap-1">
                    {OPERATORS.map((o) => (
                      <button
                        key={o.value}
                        type="button"
                        onClick={() => setForm((f) => ({ ...f, operator: o.value }))}
                        className={cn(
                          'chip transition-all',
                          form.operator === o.value
                            ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                            : 'text-cyan-300/55 opacity-70 hover:opacity-100',
                        )}
                      >
                        {o.label}
                      </button>
                    ))}
                  </div>
                </Field>
                <Field label="单位 · UNIT">
                  <input
                    className="neon-input w-full"
                    value={form.unit}
                    onChange={(e) => setForm((f) => ({ ...f, unit: e.target.value }))}
                    placeholder="%"
                  />
                </Field>
                <Field label="启用 · ENABLED">
                  <button
                    type="button"
                    onClick={() => setForm((f) => ({ ...f, enabled: !f.enabled }))}
                    className="w-fit"
                  >
                    <StatusBadge status={form.enabled ? 'ok' : 'off'} label={form.enabled ? '启用' : '停用'} />
                  </button>
                </Field>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <Field label="预警值 · WARN">
                  <input
                    className="neon-input w-full"
                    type="number"
                    value={form.warningValue}
                    onChange={(e) => setForm((f) => ({ ...f, warningValue: e.target.value }))}
                    placeholder="97"
                  />
                </Field>
                <Field label="严重值 · CRIT">
                  <input
                    className="neon-input w-full"
                    type="number"
                    value={form.criticalValue}
                    onChange={(e) => setForm((f) => ({ ...f, criticalValue: e.target.value }))}
                    placeholder="95"
                  />
                </Field>
              </div>
              <div className="flex justify-end gap-2 pt-1">
                <NeonButton onClick={() => setModalOpen(false)}>取消</NeonButton>
                <NeonButton icon={saving ? <Loader2 className="animate-spin" /> : <Plus />} disabled={!canSave || saving} onClick={submit}>
                  {editing ? '保存' : '创建'}
                </NeonButton>
              </div>
            </div>
          </GlassPanel>
        </div>
      ) : null}
    </PageShell>
  )
}

/* ───────── 复用小件 ───────── */

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="mb-1 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">{label}</div>
      {children}
    </div>
  )
}

function Stat({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <div className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3" style={{ borderLeftColor: color }}>
      <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/60">{label}</div>
      <div className="font-display text-2xl font-bold leading-tight" style={{ color, textShadow: `0 0 8px ${color}` }}>
        {value}
      </div>
    </div>
  )
}

function Loading({ text }: { text: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
      <Loader2 className="size-5 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">{text}</span>
    </div>
  )
}

function ErrorBox({ text }: { text: string }) {
  return (
    <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">{text}</div>
  )
}

function EmptyBox({ text }: { text: string }) {
  return (
    <div className="flex items-center justify-center py-12 font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/40">
      {text}
    </div>
  )
}
