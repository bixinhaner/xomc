import { useEffect, useMemo, useState } from 'react'
import { useLocation, useNavigate, useParams } from 'react-router-dom'
import {
  ArrowLeft,
  Save,
  Eye,
  Pencil,
  Loader2,
  Info,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatTime } from '@/lib/format'
import { useProvisioningTask } from '@core/hooks/api/useProvisioning'
import { useProductClasses } from '@core/hooks/api/useDevices'
import {
  provisioningStatusCode,
  type ProvisioningTaskStatusCode,
} from '@core/services/api/provisionApi'
import { KV, StateGate, StatCard } from './_shared'

type Mode = 'add' | 'edit' | 'view'

const STATUS_META: Record<ProvisioningTaskStatusCode, { label: string; color: string }> = {
  '0': { label: 'SUCCESS · 成功', color: '#00ff88' },
  '1': { label: 'FAILED · 失败', color: '#ff2d6f' },
  '2': { label: 'RUNNING · 执行中', color: '#00f0ff' },
  '3': { label: 'PENDING · 未执行', color: '#525a78' },
  '4': { label: 'SKIPPED · 跳过', color: '#a855f7' },
}

// 前端策略草稿（后端无策略资源——与 v1 一致，仅本地配置）。
interface PolicyDraft {
  name: string
  productClass: string
  enabled: boolean
  autoCommission: boolean
  remark: string
}

const EMPTY_DRAFT: PolicyDraft = {
  name: '',
  productClass: '',
  enabled: true,
  autoCommission: true,
  remark: '',
}

export default function FleetPlugAndPlayPolicy() {
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const { id } = useParams<{ id: string }>()

  const mode: Mode = pathname.endsWith('/add')
    ? 'add'
    : pathname.includes('/view/')
      ? 'view'
      : 'edit'

  // edit/view 用 id 加载真实开通任务作为策略上下文锚点。
  const { data: task, isLoading, isError, error } = useProvisioningTask(id ?? '')
  const { data: productClasses } = useProductClasses()

  const [draft, setDraft] = useState<PolicyDraft>(EMPTY_DRAFT)
  const [saved, setSaved] = useState(false)

  // 进入 edit 时用真实任务派生默认值回填草稿（仅一次）。
  useEffect(() => {
    if (mode !== 'add' && task) {
      setDraft((prev) => ({
        ...prev,
        name: prev.name || `策略 · ${task.id.slice(0, 8)}`,
        productClass: prev.productClass || '',
      }))
    }
  }, [mode, task])

  const set = <K extends keyof PolicyDraft>(k: K, v: PolicyDraft[K]) =>
    setDraft((p) => ({ ...p, [k]: v }))

  const titleMode = mode === 'add' ? 'NEW' : mode === 'view' ? 'VIEW' : 'EDIT'
  const code = task ? provisioningStatusCode(task.status) : '3'
  const meta = STATUS_META[code]

  const readonly = mode === 'view'
  const canSave = draft.name.trim().length > 0 && !readonly

  return (
    <PageShell
      code="F09"
      title={`POLICY · ${titleMode}`}
      subtitle="AUTO-PROVISION POLICY · LOCAL CONFIG DRAFT"
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/device/plug-and-play')}>
            BACK
          </NeonButton>
          {mode === 'view' && id ? (
            <NeonButton icon={<Pencil />} onClick={() => navigate(`/device/plug-and-play/edit/${id}`)}>
              EDIT
            </NeonButton>
          ) : null}
          {mode === 'edit' && id ? (
            <NeonButton icon={<Eye />} onClick={() => navigate(`/device/plug-and-play/view/${id}`)}>
              VIEW
            </NeonButton>
          ) : null}
        </>
      }
    >
      <div className="mx-auto max-w-3xl space-y-4">
        <div className="flex items-start gap-2 rounded-sm border border-cyan-500/15 bg-cyan-500/[0.03] px-3 py-2 font-mono text-[10px] text-cyan-300/55">
          <Info className="mt-0.5 size-3.5 shrink-0 text-cyan-400/60" />
          策略为前端配置草稿（F09 后端仅暴露开通任务，无策略资源）；保存仅作用于本地编辑态。
        </div>

        {/* add 模式无 id：直接编辑空草稿；edit/view 先按 id 加载真实任务上下文 */}
        {mode === 'add' ? (
          <PolicyForm
            draft={draft}
            set={set}
            productClasses={productClasses ?? []}
            readonly={false}
          />
        ) : (
          <StateGate
            isLoading={isLoading}
            isError={isError}
            error={error}
            isEmpty={!task}
            loadingLabel="LOADING TASK CONTEXT…"
            emptyLabel={`NO TASK FOR ID ${id ?? ''}`}
          >
            {task ? (
              <>
                <GlassPanel title="TASK CONTEXT · 关联任务" meta={
                  <span className="chip" style={{ color: meta.color }}>{meta.label}</span>
                }>
                  <div className="grid grid-cols-2 gap-3 p-3 md:grid-cols-4">
                    <StatCard label="STEP" value={`${task.currentStep}/${task.totalSteps || '—'}`} color="#00f0ff" />
                    <StatCard label="RETRY" value={`${task.retryCount}/${task.maxRetries}`} color="#a855f7" />
                    <StatCard label="STARTED" value={<span className="text-sm">{formatTime(task.startedAt)}</span>} color="#00ff88" />
                    <StatCard label="COMPLETED" value={<span className="text-sm">{formatTime(task.completedAt)}</span>} color="#ffaa00" />
                  </div>
                  <KV label="DEVICE ID">{task.deviceId}</KV>
                  <KV label="TEMPLATE ID">{task.templateId ?? '—'}</KV>
                  <KV label="ERROR">{task.errorMessage || '—'}</KV>
                </GlassPanel>
                <PolicyForm
                  draft={draft}
                  set={set}
                  productClasses={productClasses ?? []}
                  readonly={readonly}
                />
              </>
            ) : null}
          </StateGate>
        )}

        {!readonly && (
          <div className="flex items-center gap-3">
            <NeonButton
              icon={<Save />}
              disabled={!canSave}
              onClick={() => {
                setSaved(true)
                window.setTimeout(() => navigate('/device/plug-and-play'), 600)
              }}
            >
              {mode === 'add' ? 'CREATE POLICY' : 'SAVE POLICY'}
            </NeonButton>
            {saved && (
              <span className="flex items-center gap-1.5 font-mono text-[11px] text-emerald-300">
                <Loader2 className="size-3.5 animate-spin" /> 已保存草稿，返回…
              </span>
            )}
            {!canSave && <span className="font-mono text-[10px] text-amber-300/70">需填写策略名称</span>}
          </div>
        )}
      </div>
    </PageShell>
  )
}

function PolicyForm({
  draft,
  set,
  productClasses,
  readonly,
}: {
  draft: PolicyDraft
  set: <K extends keyof PolicyDraft>(k: K, v: PolicyDraft[K]) => void
  productClasses: string[]
  readonly: boolean
}) {
  const pcOptions = useMemo(() => productClasses, [productClasses])
  return (
    <GlassPanel strong title="POLICY · 策略配置">
      <div className="grid grid-cols-1 gap-4 p-4 md:grid-cols-2">
        <Field label="POLICY NAME *">
          <input
            className="neon-input w-full"
            value={draft.name}
            disabled={readonly}
            onChange={(e) => set('name', e.target.value)}
            placeholder="如 CMCC-LTE 默认策略"
          />
        </Field>
        <Field label="PRODUCT CLASS">
          <select
            className="neon-input w-full"
            value={draft.productClass}
            disabled={readonly}
            onChange={(e) => set('productClass', e.target.value)}
          >
            <option value="" className="bg-[#03050d]">全部型号</option>
            {pcOptions.map((pc) => (
              <option key={pc} value={pc} className="bg-[#03050d]">
                {pc}
              </option>
            ))}
          </select>
        </Field>
        <Toggle
          label="ENABLED · 启用"
          checked={draft.enabled}
          disabled={readonly}
          onChange={(v) => set('enabled', v)}
        />
        <Toggle
          label="AUTO COMMISSION · 自动调测"
          checked={draft.autoCommission}
          disabled={readonly}
          onChange={(v) => set('autoCommission', v)}
        />
        <Field label="REMARK · 备注" full>
          <textarea
            className="neon-input h-20 w-full resize-none"
            value={draft.remark}
            disabled={readonly}
            onChange={(e) => set('remark', e.target.value)}
          />
        </Field>
      </div>
    </GlassPanel>
  )
}

function Field({
  label,
  full,
  children,
}: {
  label: string
  full?: boolean
  children: React.ReactNode
}) {
  return (
    <label className={`flex flex-col gap-1.5 ${full ? 'md:col-span-2' : ''}`}>
      <span className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/60">
        {label}
      </span>
      {children}
    </label>
  )
}

function Toggle({
  label,
  checked,
  disabled,
  onChange,
}: {
  label: string
  checked: boolean
  disabled?: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <div className="flex items-center justify-between rounded-sm border border-cyan-500/15 px-3 py-2.5">
      <span className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/60">
        {label}
      </span>
      <button
        type="button"
        disabled={disabled}
        onClick={() => onChange(!checked)}
        className={`relative h-5 w-10 rounded-full transition-colors ${
          checked ? 'bg-cyan-400/40' : 'bg-cyan-500/10'
        } ${disabled ? 'opacity-50' : ''}`}
        aria-pressed={checked}
      >
        <span
          className="absolute top-0.5 size-4 rounded-full transition-all"
          style={{
            left: checked ? '1.375rem' : '0.125rem',
            background: checked ? '#00ff88' : '#525a78',
            boxShadow: checked ? '0 0 8px #00ff88' : 'none',
          }}
        />
      </button>
    </div>
  )
}
