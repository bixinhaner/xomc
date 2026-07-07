import { useEffect, useMemo, useState } from 'react'
import { CheckCircle2, Copy, Loader2, PlugZap, RotateCcw, Save } from 'lucide-react'

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
import { PageShell } from '@/components/layout/PageShell'

import {
  useSysConfigsByCategory,
  useBatchUpdateSysConfigs,
} from '@core/hooks/api/useSystem'
import {
  useAdminAgentConfig,
  useSaveAdminAgentConfig,
  useSyncAdminAgentConfig,
  useTestAdminAgentConfig,
} from '@core/hooks/api/useAgentConfig'
import type { AgentAdminConfigUpdate } from '@core/types/agentConfig'
import type {
  BatchUpdateSysConfigItem,
  SysConfigItem,
  SysConfigValueType,
} from '@core/types/system'
import { getSysConfigEnum } from '@core/config/sysConfigEnums'
import { useT } from '@/hooks/useT'

// ============================================================
// 系统管理 / 系统配置 — 对齐 v1 webcode/src/pages/system/SystemConfig
// 真实数据 useSysConfigsByCategory（adminApi.getSysConfigsByCategory，按分类拉 KV）
//   + useBatchUpdateSysConfigs（POST /admin/sysConfig/batch）。
// 后端 7 个分类：basic / security / device / notify / storage / omc / northbound。
// notify tab 已隐藏（#781）：邮件/短信后端未真实打通前不展示。
// v1 的强类型表单 + 保留策略页签未逐项复刻；这里用通用 KV 编辑器覆盖全部分类，
// bool/int/float/string/json 原样回写（value_type 透传）。详见返回 notes。
// ============================================================

const CATEGORIES: { key: string; label: string }[] = [
  { key: 'basic', label: '基础' },
  { key: 'security', label: '安全' },
  { key: 'device', label: '设备' },
  { key: 'storage', label: '存储' },
  { key: 'agent', label: 'Agent' },
  // northbound 已隐藏（#820）：北向功能未完成，待完成后恢复
]

// 已知 sys_configs key 占位（issue #548 切片 3）。
// 后端 admin.SysConfigService.RegisterValidator 注册的 key 在 DB 没记录时，
// 通用 KV 编辑器会显示"暂无配置项"——运维找不到入口去设置。把这类 key 写进
// 占位列表，DB 没有就显示一个空 input 行让用户填，保存后变成真 DB 行。
const KNOWN_KEYS_BY_CATEGORY: Record<
  string,
  Array<{ key: string; valueType: SysConfigValueType; description?: string }>
> = {
  storage: [
    {
      key: 'minio_public_endpoint',
      valueType: 'string',
      description:
        'MinIO 对外可达 endpoint（浏览器/外部 SDK 用，host[:port]）。建议在此填写运维可达地址；留空仅演示 / 开发环境使用，会回退到启动配置中的内部 host',
    },
  ],
  'stationlog.retention': [
    {
      key: 'max_file_count_per_device',
      valueType: 'int',
      description: '每设备故障日志文件数配额，0=禁用',
    },
  ],
}

// 后端待实现、暂不展示的 key（#801 磁盘告警阈值后端未实现，隐藏整卡）。
const HIDDEN_KEYS_BY_CATEGORY: Record<string, string[]> = {
  // #780: eNB 位置移动检测后端逻辑未实现，本次仅隐藏页面入口。
  device: ['locationDetection', 'latitudeToleranceRange'],
  storage: [
    'varDiskAlarmThresHold',
    'homeDiskAlarmThresHold',
    'usrDiskAlarmThresHold',
    'rootDiskAlarmThresHold',
  ],
}

// mergeKnownKeys：DB 拉到的 items + 已知 key 占位行（未出现的）合并。
function mergeKnownKeys(category: string, dbItems: SysConfigItem[]): SysConfigItem[] {
  const known = KNOWN_KEYS_BY_CATEGORY[category]
  if (!known || known.length === 0) return dbItems
  const dbKeys = new Set(dbItems.map((it) => it.key))
  const placeholders: SysConfigItem[] = known
    .filter((k) => !dbKeys.has(k.key))
    .map((k) => ({
      id: '',
      category,
      key: k.key,
      value: '',
      valueType: k.valueType,
      description: k.description,
    }))
  return [...dbItems, ...placeholders]
}

// device 分类按 enb*/cpe* 前缀分组，与 v1「基站类 / CPE 类」结构化表单等深（issue #357）。
// 其余分类不分组（单组），保持通用 KV 编辑器形态。
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

function CategoryEditor({ category }: { category: string }) {
  const t = useT()
  const { data, isLoading, isError, error, refetch, isFetching } =
    useSysConfigsByCategory(category)
  const batchUpdate = useBatchUpdateSysConfigs()

  const pendingKeys = useMemo(
    () => new Set(HIDDEN_KEYS_BY_CATEGORY[category] ?? []),
    [category],
  )

  const items = useMemo<SysConfigItem[]>(
    () => mergeKnownKeys(category, data ?? []).filter((it) => !pendingKeys.has(it.key)),
    [data, category],
  )
  const groups = useMemo(() => groupDeviceItems(category, items), [category, items])

  // 本地编辑态：key -> value（字符串，与后端 sys_configs.value TEXT 列一致）
  const [edits, setEdits] = useState<Record<string, string>>({})

  // 数据到位/切分类后用服务端值重置本地编辑态。
  useEffect(() => {
    const next: Record<string, string> = {}
    for (const it of items) next[it.key] = it.value ?? ''
    setEdits(next)
  }, [items])

  const dirty = useMemo(
    () => items.some((it) => (edits[it.key] ?? '') !== (it.value ?? '')),
    [items, edits]
  )

  function handleSave() {
    const payloadItems: BatchUpdateSysConfigItem[] = items
      .filter((it) => (edits[it.key] ?? '') !== (it.value ?? ''))
      .map((it) => ({
        key: it.key,
        value: edits[it.key] ?? '',
        value_type: it.valueType ?? 'string',
      }))
    if (payloadItems.length === 0) return
    batchUpdate.mutate(
      { category, items: payloadItems },
      { onSuccess: () => void refetch() }
    )
  }

  function handleReset() {
    const next: Record<string, string> = {}
    for (const it of items) next[it.key] = it.value ?? ''
    setEdits(next)
  }

  if (isLoading) {
    return (
      <div className="flex h-40 items-center justify-center text-muted-foreground">
        <Loader2 className="size-5 animate-spin" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
        加载失败：{error instanceof Error ? error.message : '未知错误'}
      </div>
    )
  }

  if (items.length === 0) {
    return (
      <div className="rounded-lg border bg-card px-4 py-8 text-center text-sm text-muted-foreground">
        该分类暂无配置项
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {batchUpdate.isSuccess ? (
        <div className="rounded-md border border-emerald-500/30 bg-emerald-500/5 px-4 py-2 text-sm text-emerald-600 dark:text-emerald-400">
          已保存
        </div>
      ) : null}
      {batchUpdate.isError ? (
        <div className="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-2 text-sm text-destructive">
          保存失败：
          {batchUpdate.error instanceof Error
            ? batchUpdate.error.message
            : '未知错误'}
        </div>
      ) : null}

      {groups.map((group, gi) => (
        <div key={group.titleKey ?? `grp-${gi}`} className="space-y-1.5">
          {group.titleKey ? (
            <div className="px-1 text-sm font-semibold text-foreground">{t(group.titleKey)}</div>
          ) : null}
          <div className="overflow-hidden rounded-lg border bg-card divide-y">
            {group.items.map((it) => {
              const isBool = it.valueType === 'bool'
              const enumSpec = getSysConfigEnum(category, it.key)
              const cur = edits[it.key] ?? ''
              return (
                <div
                  key={it.id || it.key}
                  className="grid gap-2 px-4 py-3 md:grid-cols-[280px_1fr] md:items-center"
                >
                  <div>
                    <Label className="font-mono text-xs">{it.key}</Label>
                    <div className="mt-0.5 flex items-center gap-2">
                      {it.valueType ? (
                        <Badge variant="muted">{it.valueType}</Badge>
                      ) : null}
                      {it.description ? (
                        <span className="text-xs text-muted-foreground">
                          {it.description}
                        </span>
                      ) : null}
                    </div>
                  </div>
                  {enumSpec ? (
                    <Select
                      value={cur}
                      onValueChange={(v) =>
                        setEdits((prev) => ({ ...prev, [it.key]: v }))
                      }
                    >
                      <SelectTrigger className="w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {enumSpec.options.map((opt) => (
                          <SelectItem key={opt.value} value={opt.value}>
                            {t(opt.labelKey)}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  ) : isBool ? (
                    <div className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        className="size-4 cursor-pointer accent-primary disabled:cursor-not-allowed disabled:opacity-50"
                        checked={cur === 'true' || cur === '1'}
                        disabled={false}
                        onChange={(e) =>
                          setEdits((prev) => ({
                            ...prev,
                            [it.key]: e.target.checked ? 'true' : 'false',
                          }))
                        }
                      />
                      <span className="text-xs text-muted-foreground">
                        {cur === 'true' || cur === '1' ? '开启' : '关闭'}
                      </span>
                    </div>
                  ) : (
                    <Input
                      value={cur}
                      onChange={(e) =>
                        setEdits((prev) => ({ ...prev, [it.key]: e.target.value }))
                      }
                    />
                  )}
                </div>
              )
            })}
          </div>
        </div>
      ))}

      <div className="flex items-center gap-2">
        <Button
          size="sm"
          disabled={!dirty || batchUpdate.isPending}
          onClick={handleSave}
        >
          {batchUpdate.isPending ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <Save className="size-4" />
          )}
          保存
        </Button>
        <Button
          variant="outline"
          size="sm"
          disabled={!dirty || batchUpdate.isPending}
          onClick={handleReset}
        >
          <RotateCcw className="size-4" /> 撤销修改
        </Button>
        {isFetching ? (
          <span className="flex items-center gap-1 text-xs text-muted-foreground">
            <Loader2 className="size-3.5 animate-spin" /> 刷新中
          </span>
        ) : null}
      </div>
    </div>
  )
}

type AgentDraft = {
  enabled: boolean
  agentStudioBaseUrl: string
  agentStudioServiceToken: string
  omcPublicBaseUrl: string
  connectorSlug: string
}

function agentDraftFromConfig(config: ReturnType<typeof useAdminAgentConfig>['data']): AgentDraft {
  return {
    enabled: Boolean(config?.enabled),
    agentStudioBaseUrl: config?.agentStudioBaseUrl ?? '',
    agentStudioServiceToken: '',
    omcPublicBaseUrl: config?.omcPublicBaseUrl ?? '',
    connectorSlug: config?.connectorSlug ?? '',
  }
}

function statusVariant(status: string): 'success' | 'destructive' | 'warning' | 'muted' {
  if (status === 'connected') return 'success'
  if (status === 'error') return 'destructive'
  if (status === 'disabled') return 'muted'
  return 'warning'
}

function AgentConfigEditor() {
  const t = useT()
  const configQ = useAdminAgentConfig()
  const saveM = useSaveAdminAgentConfig()
  const testM = useTestAdminAgentConfig()
  const syncM = useSyncAdminAgentConfig()
  const [draft, setDraft] = useState<AgentDraft>(() => agentDraftFromConfig(undefined))

  useEffect(() => {
    setDraft(agentDraftFromConfig(configQ.data))
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
      omcPublicBaseUrl: draft.omcPublicBaseUrl.trim(),
      connectorSlug: draft.connectorSlug.trim() || undefined,
    }
  }

  async function copyText(value: string) {
    if (!value) return
    await navigator.clipboard?.writeText(value)
  }

  const busy = saveM.isPending || testM.isPending || syncM.isPending
  const mutationError = saveM.error || testM.error || syncM.error
  const mutationErrorText = mutationError instanceof Error ? mutationError.message : ''

  if (configQ.isLoading) {
    return (
      <div className="flex h-40 items-center justify-center text-muted-foreground">
        <Loader2 className="size-5 animate-spin" />
      </div>
    )
  }

  if (configQ.isError) {
    return (
      <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
        加载失败：{configQ.error instanceof Error ? configQ.error.message : '未知错误'}
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {configQ.data?.lastError ? (
        <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          {t('system.agent.lastError')}：{configQ.data.lastError}
        </div>
      ) : null}
      {mutationErrorText ? (
        <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          {mutationErrorText}
        </div>
      ) : null}

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="flex items-center justify-between border-b px-4 py-3">
          <div className="flex items-center gap-2 text-sm font-semibold">
            <PlugZap className="size-4 text-primary" />
            {t('system.agent.section.connection')}
          </div>
          <Badge variant={statusVariant(configQ.data?.status ?? 'not_configured')}>
            {configQ.data?.status ?? 'not_configured'}
          </Badge>
        </div>

        <div className="divide-y">
          <div className="grid gap-2 px-4 py-3 md:grid-cols-[240px_1fr] md:items-center">
            <Label>{t('system.agent.enabled')}</Label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                className="size-4 accent-primary"
                checked={draft.enabled}
                onChange={(event) => patch({ enabled: event.target.checked })}
              />
              {draft.enabled ? '开启' : '关闭'}
            </label>
          </div>
          <AgentInputRow
            label={t('system.agent.agentStudioBaseUrl')}
            value={draft.agentStudioBaseUrl}
            onChange={(value) => patch({ agentStudioBaseUrl: value })}
            placeholder="https://agent.example.com"
          />
          <div className="grid gap-2 px-4 py-3 md:grid-cols-[240px_1fr] md:items-center">
            <div>
              <Label>{t('system.agent.serviceToken')}</Label>
              <div className="mt-1">
                <Badge variant={configQ.data?.serviceTokenConfigured ? 'success' : 'warning'}>
                  {configQ.data?.serviceTokenConfigured
                    ? t('system.agent.serviceTokenSet')
                    : t('system.agent.serviceTokenUnset')}
                </Badge>
              </div>
            </div>
            <Input
              type="password"
              value={draft.agentStudioServiceToken}
              onChange={(event) => patch({ agentStudioServiceToken: event.target.value })}
              placeholder={t('system.agent.serviceTokenPlaceholder')}
              autoComplete="new-password"
            />
          </div>
          <AgentInputRow
            label={t('system.agent.omcPublicBaseUrl')}
            value={draft.omcPublicBaseUrl}
            onChange={(value) => patch({ omcPublicBaseUrl: value })}
            placeholder="https://ops.example.com"
          />
          <AgentInputRow
            label={t('system.agent.connectorSlug')}
            value={draft.connectorSlug ?? ''}
            onChange={(value) => patch({ connectorSlug: value })}
            placeholder="external-agent-..."
          />
          <ReadonlyRow
            label={t('system.agent.connectorId')}
            value={configQ.data?.connectorId || t('system.agent.emptyValue')}
            onCopy={() => copyText(configQ.data?.connectorId ?? '')}
          />
          <ReadonlyRow
            label={t('system.agent.runtimeStreamUrl')}
            value={configQ.data?.runtimeStreamUrl || t('system.agent.emptyValue')}
            onCopy={() => copyText(configQ.data?.runtimeStreamUrl ?? '')}
          />
          <div className="grid gap-2 px-4 py-3 md:grid-cols-[240px_1fr] md:items-center">
            <Label>{t('system.agent.lastValidatedAt')}</Label>
            <span className="text-sm text-muted-foreground">
              {configQ.data?.lastValidatedAt || '—'}
            </span>
          </div>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Button variant="outline" size="sm" disabled={busy} onClick={() => setDraft(agentDraftFromConfig(configQ.data))}>
          <RotateCcw className="size-4" /> {t('system.agent.reset')}
        </Button>
        <Button variant="outline" size="sm" disabled={busy} onClick={() => testM.mutate(payload())}>
          {testM.isPending ? <Loader2 className="size-4 animate-spin" /> : <PlugZap className="size-4" />}
          {t('system.agent.test')}
        </Button>
        <Button variant="outline" size="sm" disabled={busy} onClick={() => saveM.mutate(payload())}>
          {saveM.isPending ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
          {t('system.agent.save')}
        </Button>
        <Button size="sm" disabled={busy} onClick={() => syncM.mutate(payload())}>
          {syncM.isPending ? <Loader2 className="size-4 animate-spin" /> : <CheckCircle2 className="size-4" />}
          {t('system.agent.sync')}
        </Button>
        {testM.isSuccess ? <Badge variant="success">{t('system.agent.testSuccess')}</Badge> : null}
        {saveM.isSuccess ? <Badge variant="success">{t('system.agent.saveSuccess')}</Badge> : null}
        {syncM.isSuccess ? <Badge variant="success">{t('system.agent.syncSuccess')}</Badge> : null}
      </div>
    </div>
  )
}

function AgentInputRow(props: {
  label: string
  value: string
  placeholder?: string
  onChange(value: string): void
}) {
  return (
    <div className="grid gap-2 px-4 py-3 md:grid-cols-[240px_1fr] md:items-center">
      <Label>{props.label}</Label>
      <Input
        value={props.value}
        placeholder={props.placeholder}
        onChange={(event) => props.onChange(event.target.value)}
      />
    </div>
  )
}

function ReadonlyRow(props: { label: string; value: string; onCopy(): void }) {
  return (
    <div className="grid gap-2 px-4 py-3 md:grid-cols-[240px_1fr] md:items-center">
      <Label>{props.label}</Label>
      <div className="flex min-w-0 items-center gap-2">
        <Input value={props.value} readOnly className="font-mono text-xs" />
        <Button type="button" variant="outline" size="icon" onClick={props.onCopy}>
          <Copy className="size-4" />
        </Button>
      </div>
    </div>
  )
}

export default function SystemConfig() {
  const [category, setCategory] = useState<string>(CATEGORIES[0].key)

  return (
    <PageShell
      title="系统配置"
      description="sys_configs 键值配置 · 分类编辑可保存生效"
      toolbar={
        <div className="flex flex-wrap items-center gap-1.5">
          {CATEGORIES.map((c) => (
            <Button
              key={c.key}
              size="sm"
              variant={category === c.key ? 'default' : 'outline'}
              onClick={() => setCategory(c.key)}
            >
              {c.label}
            </Button>
          ))}
        </div>
      }
    >
      {category === 'agent' ? <AgentConfigEditor /> : <CategoryEditor key={category} category={category} />}
    </PageShell>
  )
}
