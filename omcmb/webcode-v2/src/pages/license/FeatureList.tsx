/**
 * FeatureList (webcode-v2) — 渲染 system_license.feature_list 三级嵌套结构。
 *
 * 与主皮肤 webcode/src/pages/SystemLicense/FeatureListView.tsx 业务语义对齐，
 * 但 UI 走 shadcn/Tailwind（无 antd）。
 *
 * 顶层 key = 一级模块；值可以是：
 *   - string "All"   — 整个模块全部解锁
 *   - string[]       — 该模块部分功能
 *   - object         — 嵌套：子模块 → 功能项数组 / "All"
 */
import { Badge } from '@/components/ui/badge'

import type { FeatureList as FeatureListData } from '@core/services/api/systemLicenseApi'

interface FeatureListProps {
  featureList: FeatureListData
}

export function FeatureList({ featureList }: FeatureListProps) {
  const entries = Object.entries(featureList ?? {}).sort(([a], [b]) =>
    a.localeCompare(b),
  )

  if (entries.length === 0) {
    return (
      <div className="py-8 text-center text-sm text-muted-foreground">
        暂无特性条目
      </div>
    )
  }

  return (
    <div className="divide-y">
      {entries.map(([module, value]) => (
        <FeatureRow key={module} module={module} value={value} />
      ))}
    </div>
  )
}

function FeatureRow({ module, value }: { module: string; value: unknown }) {
  return (
    <div className="flex gap-4 py-3">
      <div className="w-44 shrink-0 pr-2 text-sm font-semibold">{module}</div>
      <div className="flex-1">
        <FeatureValue value={value} />
      </div>
    </div>
  )
}

function FeatureValue({ value }: { value: unknown }) {
  // 一级 "All" / 单字符串 — 整个模块全开
  if (typeof value === 'string') {
    return (
      <Badge variant="success">{value === 'All' ? '全部' : value}</Badge>
    )
  }

  // 一级数组 — 多个功能名
  if (Array.isArray(value)) {
    if (value.length === 0) {
      return <span className="text-sm text-muted-foreground">—</span>
    }
    return (
      <div className="flex flex-wrap gap-1.5">
        {value.map((item, idx) => (
          <Badge key={`${String(item)}-${idx}`} variant="outline">
            {String(item)}
          </Badge>
        ))}
      </div>
    )
  }

  // 二级嵌套：object { subModule: string[] | "All" }
  if (value && typeof value === 'object') {
    const subEntries = Object.entries(value as Record<string, unknown>).sort(
      ([a], [b]) => a.localeCompare(b),
    )
    return (
      <div className="space-y-2">
        {subEntries.map(([sub, items]) => (
          <div key={sub} className="flex flex-wrap items-center gap-1.5">
            <span className="text-sm font-medium">{sub}</span>
            {Array.isArray(items) ? (
              <>
                <span className="text-xs text-muted-foreground">
                  ({items.length})
                </span>
                {items.length === 0 ? (
                  <span className="text-sm text-muted-foreground">—</span>
                ) : (
                  items.map((item, idx) => (
                    <Badge key={`${String(item)}-${idx}`} variant="outline">
                      {String(item)}
                    </Badge>
                  ))
                )}
              </>
            ) : typeof items === 'string' ? (
              <Badge variant="success">{items === 'All' ? '全部' : items}</Badge>
            ) : (
              <span className="text-xs text-muted-foreground">
                {JSON.stringify(items)}
              </span>
            )}
          </div>
        ))}
      </div>
    )
  }

  // 兜底
  return (
    <span className="text-xs text-muted-foreground">{JSON.stringify(value)}</span>
  )
}
