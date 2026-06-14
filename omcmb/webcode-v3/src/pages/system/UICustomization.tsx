import { useMemo } from 'react'
import { RefreshCcw, ImageIcon, Palette } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { useSysConfigsByCategory } from '@core/hooks/api/useSystem'

import { StateBlock, MiniStat, FieldRow } from './_shared'

// 与后端 sys_configs(category='ui_custom') key 对齐
// （参 webcode/src/pages/system/SystemConfig/uiCustomConstants.ts）。
const UI_CUSTOM_FIELDS: { key: string; label: string; sub: string }[] = [
  { key: 'ui_login_background', label: '登录页背景', sub: 'LOGIN BACKGROUND' },
  { key: 'ui_menu_logo_up', label: '收起态 Logo', sub: 'MENU LOGO · COLLAPSED' },
  { key: 'ui_menu_logo_down', label: '展开态 Logo', sub: 'MENU LOGO · EXPANDED' },
]

export default function UICustomization() {
  const { data, isLoading, isError, error, isFetching, refetch } =
    useSysConfigsByCategory('ui_custom')
  const items = data ?? []

  const valueMap = useMemo(() => {
    const m: Record<string, string> = {}
    for (const c of items) m[c.key] = c.value ?? ''
    return m
  }, [items])

  return (
    <PageShell
      code="F06"
      title="UI CUSTOM · 界面定制"
      subtitle="BRANDING ASSETS"
      isFetching={isFetching}
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      <StateBlock
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={false}
        emptyLabel="NO UI ASSETS"
      >
        <div className="space-y-5">
          <div className="grid grid-cols-2 gap-3 md:grid-cols-3">
            <MiniStat
              label="已配置资产"
              value={UI_CUSTOM_FIELDS.filter((f) => valueMap[f.key]).length}
              color="#00f0ff"
              icon={<Palette className="size-3.5" />}
            />
            <MiniStat label="资产槽位" value={UI_CUSTOM_FIELDS.length} color="#a855f7" />
            <MiniStat label="配置项总数" value={items.length} color="#5b9eff" />
          </div>

          <div className="grid gap-4 md:grid-cols-3">
            {UI_CUSTOM_FIELDS.map((f) => {
              const url = valueMap[f.key]
              return (
                <div key={f.key} className="glass rounded-sm p-3">
                  <div className="mb-2 flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/60">
                    <ImageIcon className="size-3.5 text-cyan-300" />
                    {f.label} · {f.sub}
                  </div>
                  <div className="flex h-32 items-center justify-center overflow-hidden rounded-sm border border-cyan-500/15 bg-[#05080f]">
                    {url ? (
                      <img
                        src={url}
                        alt={f.label}
                        className="max-h-full max-w-full object-contain"
                      />
                    ) : (
                      <span className="font-mono text-[11px] text-cyan-300/40">未配置 · DEFAULT</span>
                    )}
                  </div>
                  <div className="mt-2 truncate font-mono text-[10px] text-cyan-300/55" title={url}>
                    {url || '使用内置默认资产'}
                  </div>
                </div>
              )
            })}
          </div>

          <div className="glass rounded-sm p-3">
            <div className="mb-2 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/60">
              原始配置项 · RAW (category=ui_custom)
            </div>
            {items.length > 0 ? (
              items.map((c) => (
                <FieldRow key={c.id || c.key} label={c.key}>
                  <span className="font-mono text-xs text-cyan-200">{c.value || '—'}</span>
                </FieldRow>
              ))
            ) : (
              <div className="font-mono text-xs text-cyan-300/45">该分类暂无配置项，沿用内置默认。</div>
            )}
          </div>
        </div>
      </StateBlock>
    </PageShell>
  )
}
