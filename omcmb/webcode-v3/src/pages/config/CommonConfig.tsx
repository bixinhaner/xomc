import { useEffect, useState } from 'react'
import { RefreshCcw, Save, Loader2, SlidersHorizontal, RotateCcw, CheckCircle2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useSysConfigsByCategory, useBatchUpdateSysConfigs } from '@core/hooks/api/useSystem'
import type { SysConfigItem } from '@core/types/system'
import { StatCard, HudLoading, HudError, HudEmpty } from './_hud'

// system/config 各 category 与 SystemConfig 页 settingsTabs 对齐。
const CATEGORIES: { key: string; label: string }[] = [
  { key: 'device', label: '设备 · DEVICE' },
  { key: 'omc', label: 'OMC · 核心' },
  { key: 'notify', label: '通知 · NOTIFY' },
  { key: 'storage', label: '存储 · STORAGE' },
  { key: 'security', label: '安全 · SECURITY' },
  { key: 'acs_transfer', label: 'ACS 传输 · TRANSFER' },
]

// ===========================================================================
// CONFIG · 公共配置
// 真实 sys_configs（useSysConfigsByCategory）按 category 编辑全局公共参数，
// useBatchUpdateSysConfigs 批量回写。
// ===========================================================================
export default function CommonConfig() {
  const [category, setCategory] = useState(CATEGORIES[0].key)
  const configsQ = useSysConfigsByCategory(category)
  const batchUpdate = useBatchUpdateSysConfigs()

  const configs: SysConfigItem[] = configsQ.data ?? []
  const [draft, setDraft] = useState<Record<string, string>>({})
  const [saved, setSaved] = useState(false)
  const [err, setErr] = useState('')

  // 拉到新数据时重置草稿
  useEffect(() => {
    const next: Record<string, string> = {}
    for (const c of configs) next[c.key] = c.value
    setDraft(next)
    setSaved(false)
    setErr('')
  }, [configs])

  const dirtyKeys = configs.filter((c) => draft[c.key] !== undefined && draft[c.key] !== c.value)

  const submit = () => {
    setErr('')
    setSaved(false)
    const items = dirtyKeys.map((c) => ({ key: c.key, value: draft[c.key] }))
    if (items.length === 0) return
    batchUpdate.mutate(
      { category, items },
      {
        onSuccess: () => setSaved(true),
        onError: (e: Error) => setErr(e.message || '保存失败'),
      },
    )
  }

  const resetDraft = () => {
    const next: Record<string, string> = {}
    for (const c of configs) next[c.key] = c.value
    setDraft(next)
    setSaved(false)
  }

  return (
    <PageShell
      code="F02"
      title="COMMON CONFIG · 公共配置"
      subtitle="SYSTEM KEY-VALUE SETTINGS"
      isFetching={configsQ.isFetching}
      bare
      toolbar={
        <>
          {dirtyKeys.length > 0 ? (
            <NeonButton icon={<RotateCcw />} onClick={resetDraft}>
              撤销改动
            </NeonButton>
          ) : null}
          <NeonButton
            icon={batchUpdate.isPending ? <Loader2 className="animate-spin" /> : <Save />}
            onClick={submit}
            disabled={dirtyKeys.length === 0 || batchUpdate.isPending}
          >
            保存 {dirtyKeys.length > 0 ? `(${dirtyKeys.length})` : ''}
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => void configsQ.refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-3 gap-3">
        <StatCard label="KEYS · 配置项" value={configs.length} color="#00f0ff" />
        <StatCard label="DIRTY · 待保存" value={dirtyKeys.length} color={dirtyKeys.length > 0 ? '#ffaa00' : '#00ff88'} />
        <StatCard label="CATEGORY · 分类" value={category} color="#a855f7" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        {CATEGORIES.map((c) => (
          <button
            key={c.key}
            type="button"
            onClick={() => setCategory(c.key)}
            className={`chip ${category === c.key ? 'text-[#00f0ff] shadow-[0_0_10px_currentColor]' : 'text-[#6b86b6]'}`}
          >
            <SlidersHorizontal className="size-3" />
            {c.label}
          </button>
        ))}
      </div>

      {saved ? (
        <div className="mb-3 flex items-center gap-2 border border-emerald-500/40 bg-emerald-500/5 px-3 py-2 font-mono text-xs text-emerald-300">
          <CheckCircle2 className="size-4" />
          配置已保存并生效
        </div>
      ) : null}
      {err ? (
        <div className="mb-3 border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">{err}</div>
      ) : null}

      <GlassPanel title={`SETTINGS · ${category}`} meta={`${configs.length} KEYS`} className="min-h-0">
        {configsQ.isLoading ? (
          <HudLoading />
        ) : configsQ.isError ? (
          <HudError error={configsQ.error} />
        ) : configs.length === 0 ? (
          <HudEmpty icon={SlidersHorizontal} text="该分类暂无配置项" />
        ) : (
          <div className="max-h-[56vh] space-y-1.5 overflow-auto p-3">
            {configs.map((c) => {
              const isDirty = draft[c.key] !== undefined && draft[c.key] !== c.value
              return (
                <div
                  key={c.id || c.key}
                  className="grid grid-cols-[1.6fr_1.4fr_70px] items-center gap-3 rounded-sm border px-3 py-2.5"
                  style={{ borderColor: isDirty ? 'rgba(255,170,0,0.4)' : 'rgba(0,240,255,0.1)' }}
                >
                  <div className="min-w-0">
                    <code className="block truncate font-mono text-xs text-cyan-100/90">{c.key}</code>
                    <div className="truncate text-[10px] text-cyan-300/55">{c.description || '—'}</div>
                  </div>
                  <div>
                    {c.valueType === 'bool' ? (
                      <select
                        className="neon-input w-full"
                        value={draft[c.key] ?? c.value}
                        onChange={(e) => setDraft((d) => ({ ...d, [c.key]: e.target.value }))}
                      >
                        <option value="true">true</option>
                        <option value="false">false</option>
                      </select>
                    ) : (
                      <input
                        className="neon-input w-full"
                        type={c.valueType === 'int' || c.valueType === 'float' ? 'number' : 'text'}
                        value={draft[c.key] ?? c.value}
                        onChange={(e) => setDraft((d) => ({ ...d, [c.key]: e.target.value }))}
                      />
                    )}
                  </div>
                  <div className="flex justify-end">
                    {isDirty ? (
                      <StatusBadge status="warning" label="改动" />
                    ) : (
                      <span className="chip text-[#5b9eff]">{c.valueType ?? 'string'}</span>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </GlassPanel>
    </PageShell>
  )
}
