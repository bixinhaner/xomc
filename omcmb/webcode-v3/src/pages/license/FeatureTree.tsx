import { useMemo } from 'react'
import { Layers } from 'lucide-react'

import type { FeatureList } from '@core/services/api/systemLicenseApi'

/**
 * FeatureTree — 渲染 system_license.feature_list 三级嵌套结构（HUD 暗色版）。
 *
 * 顶层 key = 一级模块；值可以是：
 *   - string "All"        — 整个模块全部解锁
 *   - string[] / array    — 该模块部分功能
 *   - object              — 嵌套：子模块 → 功能项数组
 */

function FeatureChip({ text, tone = 'cyan' }: { text: string; tone?: 'cyan' | 'lime' }) {
  const color = tone === 'lime' ? '#00ff88' : '#00f0ff'
  return (
    <span
      className="inline-flex items-center rounded-sm border px-2 py-0.5 font-mono text-[11px]"
      style={{
        color,
        borderColor: `${color}55`,
        background: `${color}12`,
      }}
    >
      {text}
    </span>
  )
}

function FeatureRow({ module, value }: { module: string; value: unknown }) {
  let content: React.ReactNode

  if (typeof value === 'string') {
    // 一级 "All" — 整个模块全开
    content = <FeatureChip text={value === 'All' ? 'ALL · 全部解锁' : value} tone="lime" />
  } else if (Array.isArray(value)) {
    // 一级数组 — 多个功能名
    content =
      value.length === 0 ? (
        <span className="font-mono text-xs text-cyan-300/45">—</span>
      ) : (
        <div className="flex flex-wrap gap-1.5">
          {value.map((item, idx) => (
            <FeatureChip key={`${String(item)}-${idx}`} text={String(item)} />
          ))}
        </div>
      )
  } else if (value && typeof value === 'object') {
    // 二级嵌套：object { subModule: string[] | string }
    const subEntries = Object.entries(value as Record<string, unknown>).sort(([a], [b]) =>
      a.localeCompare(b),
    )
    content = (
      <div className="space-y-2">
        {subEntries.map(([sub, items]) => (
          <div key={sub} className="flex flex-wrap items-center gap-2">
            <span className="font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-cyan-200/80">
              {sub}
            </span>
            {Array.isArray(items) ? (
              items.length === 0 ? (
                <span className="font-mono text-xs text-cyan-300/45">—</span>
              ) : (
                <div className="flex flex-wrap gap-1.5">
                  {items.map((item, idx) => (
                    <FeatureChip key={`${String(item)}-${idx}`} text={String(item)} />
                  ))}
                </div>
              )
            ) : typeof items === 'string' ? (
              <FeatureChip text={items === 'All' ? 'ALL' : items} tone="lime" />
            ) : (
              <span className="font-mono text-xs text-cyan-300/45">{JSON.stringify(items)}</span>
            )}
          </div>
        ))}
      </div>
    )
  } else {
    content = <span className="font-mono text-xs text-cyan-300/45">{JSON.stringify(value)}</span>
  }

  return (
    <div className="grid grid-cols-[180px_1fr] items-start gap-4 border-b border-cyan-500/8 px-3 py-3 last:border-b-0">
      <div className="flex items-center gap-2 font-display text-sm font-bold text-cyan-100">
        <Layers className="size-3.5 text-cyan-300/60" />
        {module}
      </div>
      <div>{content}</div>
    </div>
  )
}

export function FeatureTree({ featureList }: { featureList: FeatureList }) {
  const entries = useMemo(
    () => Object.entries(featureList ?? {}).sort(([a], [b]) => a.localeCompare(b)),
    [featureList],
  )

  if (entries.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-12">
        <Layers className="size-8 text-cyan-300/30" />
        <div className="font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/50">
          NO FEATURES · 无功能授权
        </div>
      </div>
    )
  }

  return (
    <div>
      {entries.map(([module, value]) => (
        <FeatureRow key={module} module={module} value={value} />
      ))}
    </div>
  )
}
