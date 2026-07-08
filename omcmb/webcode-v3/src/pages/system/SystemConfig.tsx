import { useEffect, useState } from 'react'
import { CheckCircle2, Copy, PlugZap, RefreshCcw, Save, SlidersHorizontal } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { useSysConfigsByCategory } from '@core/hooks/api/useSystem'
import {
  useAdminAgentConfig,
  useSaveAdminAgentConfig,
  useSyncAdminAgentConfig,
  useTestAdminAgentConfig,
} from '@core/hooks/api/useAgentConfig'
import type { AgentAdminConfigUpdate } from '@core/types/agentConfig'
import type { SysConfigItem } from '@core/types/system'
import { getSysConfigEnum } from '@core/config/sysConfigEnums'

import { StateBlock, MiniStat, RowHeader } from './_shared'
import { useT } from '@/hooks/useT'

// device 分类按 enb*/cpe* 前缀分组，与 v1「基站类 / CPE 类」结构化表单等深（issue #357）。
function groupDeviceItems(
  category: string,
  items: SysConfigItem[],
): { titleKey: string | null; items: SysConfigItem[] }[] {
  if (category !== 'device') return [{ titleKey: null, items }]
  const base: SysConfigItem[] = []
  const cpe: SysConfigItem[] = []
  const other: SysConfigItem[] = []
  for (const it of items) {
    if (it.key.startsWith('enb')) base.push(it)
    else if (it.key.startsWith('cpe')) cpe.push(it)
    else other.push(it)
  }
  const groups: { titleKey: string | null; items: SysConfigItem[] }[] = []
  if (base.length) groups.push({ titleKey: 'system.device.informGroup.baseStation', items: base })
  if (cpe.length) groups.push({ titleKey: 'system.device.informGroup.cpe', items: cpe })
  if (other.length) groups.push({ titleKey: null, items: other })
  return groups
}

// 后端待实现、暂不展示的 key（#801 磁盘告警阈值后端未实现）。
const HIDDEN_KEYS = new Set([
  // #780: eNB 位置移动检测后端逻辑未实现，本次仅隐藏页面入口。
  'locationDetection',
  'latitudeToleranceRange',
  'varDiskAlarmThresHold',
  'homeDiskAlarmThresHold',
  'usrDiskAlarmThresHold',
  'rootDiskAlarmThresHold',
])

// 后端 7 个 category（参 frontend-core/src/types/system.ts SysConfigItem 注释）。
// notify tab 已隐藏（#781）：邮件/短信后端未真实打通前不展示。
const CONFIG_CATEGORIES: { key: string; label: string }[] = [
  { key: 'basic', label: '基础' },
  { key: 'security', label: '安全' },
  { key: 'device', label: '设备' },
  { key: 'storage', label: '存储' },
  { key: 'agent', label: 'Agent' },
  // northbound 已隐藏（#820）：北向功能未完成，待完成后恢复
]

type AgentDraft = {
  enabled: boolean
  agentStudioBaseUrl: string
  agentStudioServiceToken: string
  connectorSlug: string
  allowedMethods: string[]
  blockedPathPrefixes: string[]
  toolTimeoutSeconds: number
  maxResponseBytes: number
}

const METHOD_OPTIONS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE']
const READ_ONLY_METHODS = ['GET']
const WRITE_METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE']

function draftFromConfig(config: ReturnType<typeof useAdminAgentConfig>['data']): AgentDraft {
  return {
    enabled: Boolean(config?.enabled),
    agentStudioBaseUrl: config?.agentStudioBaseUrl ?? '',
    agentStudioServiceToken: '',
    connectorSlug: config?.connectorSlug ?? '',
    allowedMethods: config?.policy.allowedMethods?.length ? config.policy.allowedMethods : READ_ONLY_METHODS,
    blockedPathPrefixes: config?.policy.blockedPathPrefixes ?? [],
    toolTimeoutSeconds: config?.policy.toolTimeoutSeconds ?? 30,
    maxResponseBytes: config?.policy.maxResponseBytes ?? 262144,
  }
}

function AgentConfigPanel() {
  const t = useT()
  const configQ = useAdminAgentConfig()
  const saveM = useSaveAdminAgentConfig()
  const testM = useTestAdminAgentConfig()
  const syncM = useSyncAdminAgentConfig()
  const [draft, setDraft] = useState<AgentDraft>(() => draftFromConfig(undefined))

  useEffect(() => {
    setDraft(draftFromConfig(configQ.data))
  }, [configQ.data])

  function patch(next: Partial<AgentDraft>) {
    setDraft((prev) => ({ ...prev, ...next }))
  }

  function payload(): AgentAdminConfigUpdate {
    const token = draft.agentStudioServiceToken.trim()
    return {
      enabled: draft.enabled,
      agentStudioBaseUrl: draft.agentStudioBaseUrl.trim(),
      agentStudioServiceToken: token || undefined,
      connectorSlug: draft.connectorSlug.trim() || undefined,
      allowedMethods: draft.allowedMethods.length ? draft.allowedMethods : READ_ONLY_METHODS,
      blockedPathPrefixes: draft.blockedPathPrefixes,
      toolTimeoutSeconds: draft.toolTimeoutSeconds,
      maxResponseBytes: draft.maxResponseBytes,
    }
  }

  async function copyText(value: string) {
    if (!value) return
    await navigator.clipboard?.writeText(value)
  }

  const busy = saveM.isPending || testM.isPending || syncM.isPending
  const mutationError = saveM.error || testM.error || syncM.error
  const mutationErrorText = mutationError instanceof Error ? mutationError.message : ''
  const executionMode = draft.allowedMethods.some((method) => method !== 'GET') ? 'write' : 'read'

  return (
    <div className="glass-strong relative flex-1 min-h-0 overflow-hidden rounded-sm">
      <div className="scanline" />
      <div className="relative h-full overflow-auto p-3">
        <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-4">
          <MiniStat label="LINK" value={configQ.data?.status ?? 'not_configured'} color={configQ.data?.status === 'connected' ? '#00ff88' : '#ffcc66'} icon={<PlugZap className="size-3.5" />} />
          <MiniStat label="TOKEN" value={configQ.data?.serviceTokenConfigured ? 'SET' : 'MISSING'} color={configQ.data?.serviceTokenConfigured ? '#00ff88' : '#ff4d8d'} />
          <MiniStat label="CONNECTOR" value={configQ.data?.connectorId ? 'READY' : 'EMPTY'} color="#00f0ff" />
          <MiniStat label="MODE" value={draft.enabled ? 'ENABLED' : 'DISABLED'} color="#5b9eff" />
        </div>

        {configQ.isLoading ? (
          <div className="flex h-40 items-center justify-center font-mono text-xs text-cyan-300/70">
            LOADING AGENT CONFIG
          </div>
        ) : null}
        {configQ.isError ? (
          <div className="border border-rose-400/35 bg-rose-500/10 p-3 font-mono text-xs text-rose-100">
            {configQ.error instanceof Error ? configQ.error.message : 'LOAD FAILED'}
          </div>
        ) : null}
        {!configQ.isLoading && !configQ.isError ? (
          <div className="space-y-1.5">
            <RowHeader cols="1.2fr_2.4fr_1fr">
              <span>参数 · KEY</span>
              <span>值 · VALUE</span>
              <span>状态</span>
            </RowHeader>
            <AgentToggleRow
              label={t('system.agent.enabled')}
              checked={draft.enabled}
              onChange={(checked) => patch({ enabled: checked })}
            />
            <AgentEditRow
              label={t('system.agent.agentStudioBaseUrl')}
              value={draft.agentStudioBaseUrl}
              onChange={(value) => patch({ agentStudioBaseUrl: value })}
              placeholder="https://agent.example.com"
            />
            <AgentSecretRow
              label={t('system.agent.serviceToken')}
              value={draft.agentStudioServiceToken}
              configured={Boolean(configQ.data?.serviceTokenConfigured)}
              onChange={(value) => patch({ agentStudioServiceToken: value })}
              placeholder={t('system.agent.serviceTokenPlaceholder')}
            />
            <AgentEditRow
              label={t('system.agent.connectorSlug')}
              value={draft.connectorSlug ?? ''}
              onChange={(value) => patch({ connectorSlug: value })}
              placeholder="external-agent-..."
            />
            <AgentModeRow
              label={t('system.agent.executionMode')}
              value={executionMode}
              readLabel={t('system.agent.readOnlyMode')}
              writeLabel={t('system.agent.writeMode')}
              onChange={(value) => patch({ allowedMethods: value === 'write' ? WRITE_METHODS : READ_ONLY_METHODS })}
            />
            <AgentMethodsRow
              label={t('system.agent.allowedMethods')}
              value={draft.allowedMethods}
              onChange={(value) => patch({ allowedMethods: value.length ? value : READ_ONLY_METHODS })}
            />
            <AgentNumberRow
              label={t('system.agent.maxResponseBytes')}
              value={draft.maxResponseBytes}
              min={4096}
              max={4194304}
              step={4096}
              onChange={(value) => patch({ maxResponseBytes: value })}
            />
            <AgentNumberRow
              label={t('system.agent.toolTimeoutSeconds')}
              value={draft.toolTimeoutSeconds}
              min={1}
              max={300}
              step={1}
              onChange={(value) => patch({ toolTimeoutSeconds: value })}
            />
            <AgentTextAreaRow
              label={t('system.agent.blockedPathPrefixes')}
              value={draft.blockedPathPrefixes.join('\n')}
              onChange={(value) => patch({ blockedPathPrefixes: value.split(/[\n,]+/).map((item) => item.trim()).filter(Boolean) })}
              placeholder="/api/v1/auth/*"
            />
            <AgentReadRow label={t('system.agent.connectorId')} value={configQ.data?.connectorId || t('system.agent.emptyValue')} onCopy={() => copyText(configQ.data?.connectorId ?? '')} />
            <AgentReadRow label={t('system.agent.runtimeStreamUrl')} value={configQ.data?.runtimeStreamUrl || t('system.agent.emptyValue')} onCopy={() => copyText(configQ.data?.runtimeStreamUrl ?? '')} />
            <div className="fleet-row grid grid-cols-[1.2fr_2.4fr_1fr] items-center gap-3 rounded-sm px-3 py-2.5" style={{ ['--row-color' as never]: '#00f0ff' }}>
              <div className="font-mono text-xs text-cyan-100">{t('system.agent.lastValidatedAt')}</div>
              <div className="min-w-0 truncate font-mono text-xs text-cyan-200">{configQ.data?.lastValidatedAt || '—'}</div>
              <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/70">{configQ.data?.status ?? '—'}</div>
            </div>
            {configQ.data?.lastError || mutationErrorText ? (
              <div className="border border-rose-400/35 bg-rose-500/10 p-3 font-mono text-xs text-rose-100">
                {configQ.data?.lastError || mutationErrorText}
              </div>
            ) : null}
            <div className="flex flex-wrap items-center gap-2 pt-2">
              <NeonButton icon={<RefreshCcw />} disabled={busy} onClick={() => setDraft(draftFromConfig(configQ.data))}>
                RESET
              </NeonButton>
              <NeonButton icon={<PlugZap />} disabled={busy} onClick={() => testM.mutate(payload())}>
                TEST LINK
              </NeonButton>
              <NeonButton icon={<Save />} disabled={busy} onClick={() => saveM.mutate(payload())}>
                SAVE
              </NeonButton>
              <NeonButton icon={<CheckCircle2 />} disabled={busy} onClick={() => syncM.mutate(payload())}>
                SAVE + SYNC
              </NeonButton>
              {testM.isSuccess || saveM.isSuccess || syncM.isSuccess ? (
                <span className="chip text-[#00ff88]">
                  {syncM.isSuccess ? t('system.agent.syncSuccess') : saveM.isSuccess ? t('system.agent.saveSuccess') : t('system.agent.testSuccess')}
                </span>
              ) : null}
            </div>
          </div>
        ) : null}
      </div>
    </div>
  )
}

function AgentEditRow(props: {
  label: string
  value: string
  placeholder?: string
  onChange(value: string): void
}) {
  return (
    <div className="fleet-row grid grid-cols-[1.2fr_2.4fr_1fr] items-center gap-3 rounded-sm px-3 py-2.5" style={{ ['--row-color' as never]: '#00f0ff' }}>
      <div className="font-mono text-xs text-cyan-100">{props.label}</div>
      <input
        value={props.value}
        placeholder={props.placeholder}
        onChange={(event) => props.onChange(event.target.value)}
        className="h-8 min-w-0 border border-cyan-400/25 bg-cyan-500/[0.04] px-2 font-mono text-xs text-cyan-100 outline-none placeholder:text-cyan-300/35 focus:border-cyan-300/70"
      />
      <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/70">EDIT</div>
    </div>
  )
}

function AgentModeRow(props: {
  label: string
  value: 'read' | 'write'
  readLabel: string
  writeLabel: string
  onChange(value: 'read' | 'write'): void
}) {
  return (
    <div className="fleet-row grid grid-cols-[1.2fr_2.4fr_1fr] items-center gap-3 rounded-sm px-3 py-2.5" style={{ ['--row-color' as never]: '#00f0ff' }}>
      <div className="font-mono text-xs text-cyan-100">{props.label}</div>
      <div className="flex flex-wrap gap-2">
        {([
          ['read', props.readLabel],
          ['write', props.writeLabel],
        ] as const).map(([value, label]) => (
          <button
            key={value}
            type="button"
            onClick={() => props.onChange(value)}
            className={`border px-3 py-1.5 font-mono text-[10px] uppercase tracking-[0.16em] ${
              props.value === value
                ? 'border-cyan-200 bg-cyan-300/15 text-cyan-100 shadow-[0_0_12px_rgba(0,240,255,0.18)]'
                : 'border-cyan-400/25 text-cyan-300/60 hover:bg-cyan-400/10'
            }`}
          >
            {label}
          </button>
        ))}
      </div>
      <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/70">MODE</div>
    </div>
  )
}

function AgentMethodsRow(props: { label: string; value: string[]; onChange(value: string[]): void }) {
  return (
    <div className="fleet-row grid grid-cols-[1.2fr_2.4fr_1fr] items-center gap-3 rounded-sm px-3 py-2.5" style={{ ['--row-color' as never]: '#00f0ff' }}>
      <div className="font-mono text-xs text-cyan-100">{props.label}</div>
      <div className="flex flex-wrap gap-3">
        {METHOD_OPTIONS.map((method) => (
          <label key={method} className="flex items-center gap-2 font-mono text-xs text-cyan-100">
            <input
              type="checkbox"
              className="size-4 accent-cyan-300"
              checked={props.value.includes(method)}
              onChange={(event) => {
                const next = event.target.checked
                  ? [...props.value, method]
                  : props.value.filter((item) => item !== method)
                props.onChange(METHOD_OPTIONS.filter((item) => next.includes(item)))
              }}
            />
            {method}
          </label>
        ))}
      </div>
      <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/70">ALLOW</div>
    </div>
  )
}

function AgentNumberRow(props: {
  label: string
  value: number
  min: number
  max: number
  step: number
  onChange(value: number): void
}) {
  return (
    <div className="fleet-row grid grid-cols-[1.2fr_2.4fr_1fr] items-center gap-3 rounded-sm px-3 py-2.5" style={{ ['--row-color' as never]: '#00f0ff' }}>
      <div className="font-mono text-xs text-cyan-100">{props.label}</div>
      <input
        type="number"
        value={props.value}
        min={props.min}
        max={props.max}
        step={props.step}
        onChange={(event) => props.onChange(Math.min(props.max, Math.max(props.min, Number(event.target.value) || props.min)))}
        className="h-8 min-w-0 border border-cyan-400/25 bg-cyan-500/[0.04] px-2 font-mono text-xs text-cyan-100 outline-none placeholder:text-cyan-300/35 focus:border-cyan-300/70"
      />
      <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/70">LIMIT</div>
    </div>
  )
}

function AgentTextAreaRow(props: {
  label: string
  value: string
  placeholder?: string
  onChange(value: string): void
}) {
  return (
    <div className="fleet-row grid grid-cols-[1.2fr_2.4fr_1fr] items-start gap-3 rounded-sm px-3 py-2.5" style={{ ['--row-color' as never]: '#00f0ff' }}>
      <div className="font-mono text-xs text-cyan-100">{props.label}</div>
      <textarea
        value={props.value}
        placeholder={props.placeholder}
        onChange={(event) => props.onChange(event.target.value)}
        className="min-h-20 min-w-0 border border-cyan-400/25 bg-cyan-500/[0.04] px-2 py-2 font-mono text-xs text-cyan-100 outline-none placeholder:text-cyan-300/35 focus:border-cyan-300/70"
      />
      <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/70">BLOCK</div>
    </div>
  )
}

function AgentSecretRow(props: {
  label: string
  value: string
  configured: boolean
  placeholder?: string
  onChange(value: string): void
}) {
  return (
    <div className="fleet-row grid grid-cols-[1.2fr_2.4fr_1fr] items-center gap-3 rounded-sm px-3 py-2.5" style={{ ['--row-color' as never]: '#00f0ff' }}>
      <div className="font-mono text-xs text-cyan-100">{props.label}</div>
      <input
        type="password"
        value={props.value}
        placeholder={props.placeholder}
        onChange={(event) => props.onChange(event.target.value)}
        className="h-8 min-w-0 border border-cyan-400/25 bg-cyan-500/[0.04] px-2 font-mono text-xs text-cyan-100 outline-none placeholder:text-cyan-300/35 focus:border-cyan-300/70"
      />
      <div className={`font-mono text-[10px] uppercase tracking-[0.18em] ${props.configured ? 'text-emerald-300' : 'text-amber-300'}`}>
        {props.configured ? 'SET' : 'MISSING'}
      </div>
    </div>
  )
}

function AgentToggleRow(props: { label: string; checked: boolean; onChange(value: boolean): void }) {
  return (
    <div className="fleet-row grid grid-cols-[1.2fr_2.4fr_1fr] items-center gap-3 rounded-sm px-3 py-2.5" style={{ ['--row-color' as never]: '#00f0ff' }}>
      <div className="font-mono text-xs text-cyan-100">{props.label}</div>
      <label className="flex items-center gap-2 font-mono text-xs text-cyan-100">
        <input
          type="checkbox"
          className="size-4 accent-cyan-300"
          checked={props.checked}
          onChange={(event) => props.onChange(event.target.checked)}
        />
        {props.checked ? 'ENABLED' : 'DISABLED'}
      </label>
      <div className={`font-mono text-[10px] uppercase tracking-[0.18em] ${props.checked ? 'text-emerald-300' : 'text-cyan-300/60'}`}>
        MODE
      </div>
    </div>
  )
}

function AgentReadRow(props: { label: string; value: string; onCopy(): void }) {
  return (
    <div className="fleet-row grid grid-cols-[1.2fr_2.4fr_1fr] items-center gap-3 rounded-sm px-3 py-2.5" style={{ ['--row-color' as never]: '#00f0ff' }}>
      <div className="font-mono text-xs text-cyan-100">{props.label}</div>
      <div className="min-w-0 truncate font-mono text-xs text-cyan-200">{props.value}</div>
      <button
        type="button"
        onClick={props.onCopy}
        className="inline-flex h-7 items-center justify-center gap-1 border border-cyan-400/30 px-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-200 hover:bg-cyan-400/10"
      >
        <Copy className="size-3.5" /> COPY
      </button>
    </div>
  )
}

export default function SystemConfig() {
  const t = useT()
  const [category, setCategory] = useState<string>('basic')
  const { data, isLoading, isError, error, isFetching, refetch } = useSysConfigsByCategory(category, category !== 'agent')
  const items = (data ?? []).filter((it) => !HIDDEN_KEYS.has(it.key))
  const groups = groupDeviceItems(category, items)

  return (
    <PageShell
      code="F06"
      title="CONFIG · 系统参数"
      subtitle="SYS CONFIG · KV STORE"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          {CONFIG_CATEGORIES.map((c) => (
            <button
              key={c.key}
              type="button"
              onClick={() => setCategory(c.key)}
              className={`chip transition-all ${
                category === c.key
                  ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                  : 'text-cyan-300/55 hover:text-cyan-200'
              }`}
            >
              {c.label}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="flex h-full flex-col gap-3">
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <MiniStat
            label="当前分类"
            value={CONFIG_CATEGORIES.find((c) => c.key === category)?.label ?? category}
            color="#00f0ff"
            icon={<SlidersHorizontal className="size-3.5" />}
          />
          <MiniStat label="配置项数" value={items.length} color="#00ff88" />
          <MiniStat label="公开项" value={items.filter((i) => i.isPublic).length} color="#a855f7" />
          <MiniStat label="分类总数" value={CONFIG_CATEGORIES.length} color="#5b9eff" />
        </div>

        {category === 'agent' ? (
          <AgentConfigPanel />
        ) : (
        <div className="glass-strong relative flex-1 min-h-0 overflow-hidden rounded-sm">
          <div className="scanline" />
          <div className="relative h-full overflow-auto p-3">
            <StateBlock
              isLoading={isLoading}
              isError={isError}
              error={error}
              isEmpty={items.length === 0}
              emptyLabel="NO CONFIG ITEMS · 该分类无配置"
            >
              <div className="space-y-1.5">
                <RowHeader cols="1.6fr_1.6fr_0.8fr_2fr">
                  <span>键 · KEY</span>
                  <span>值 · VALUE</span>
                  <span>类型</span>
                  <span>说明</span>
                </RowHeader>
                {groups.map((group, gi) => (
                  <div key={group.titleKey ?? `grp-${gi}`} className="space-y-1.5">
                    {group.titleKey ? (
                      <div className="px-1 pt-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/70">
                        ── {t(group.titleKey)} ──
                      </div>
                    ) : null}
                    {group.items.map((cfg: SysConfigItem) => (
                      <div
                        key={cfg.id || `${cfg.category}.${cfg.key}`}
                        className="fleet-row grid grid-cols-[1.6fr_1.6fr_0.8fr_2fr] items-center gap-3 rounded-sm px-3 py-2.5"
                        style={{ ['--row-color' as never]: '#00f0ff' }}
                      >
                        <div className="min-w-0">
                          <div className="truncate font-mono text-xs text-cyan-100">{cfg.key}</div>
                          {cfg.isPublic ? <span className="chip text-[#00ff88]">PUBLIC</span> : null}
                        </div>
                        <div className="min-w-0 truncate font-mono text-xs text-cyan-200">
                          {(() => {
                            // 枚举型配置项：只读展示时把存储值映射为友好文案（如
                            // auto_lmt_to_omc → 自动修改：LMT 名称覆盖网管），与 v1/v2 一致。
                            const enumSpec = getSysConfigEnum(cfg.category, cfg.key)
                            const opt = enumSpec?.options.find((o) => o.value === cfg.value)
                            if (opt) return t(opt.labelKey)
                            return cfg.value || '—'
                          })()}
                        </div>
                        <div className="font-mono text-[10px] uppercase tracking-[0.12em] text-cyan-300/65">
                          {cfg.valueType || 'string'}
                        </div>
                        <div className="min-w-0 truncate text-xs text-cyan-100/75">
                          {cfg.description || '—'}
                        </div>
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            </StateBlock>
          </div>
        </div>
        )}
      </div>
    </PageShell>
  )
}
