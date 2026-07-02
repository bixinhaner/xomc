import { useState } from 'react'
import { RefreshCcw, SlidersHorizontal } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { useSysConfigsByCategory } from '@core/hooks/api/useSystem'
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

// 后端 7 个 category（参 frontend-core/src/types/system.ts SysConfigItem 注释）。
// notify tab 已隐藏（#781）：邮件/短信后端未真实打通前不展示。
const CONFIG_CATEGORIES: { key: string; label: string }[] = [
  { key: 'basic', label: '基础' },
  { key: 'security', label: '安全' },
  { key: 'device', label: '设备' },
  { key: 'storage', label: '存储' },
  { key: 'omc', label: 'OMC' },
  { key: 'northbound', label: '北向' },
]

export default function SystemConfig() {
  const t = useT()
  const [category, setCategory] = useState<string>('basic')
  const { data, isLoading, isError, error, isFetching, refetch } = useSysConfigsByCategory(category)
  const items = data ?? []
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
      </div>
    </PageShell>
  )
}
