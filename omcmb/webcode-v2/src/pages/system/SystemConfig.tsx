import { useEffect, useMemo, useState } from 'react'
import { Loader2, RotateCcw, Save } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { PageShell } from '@/components/layout/PageShell'

import {
  useSysConfigsByCategory,
  useBatchUpdateSysConfigs,
} from '@core/hooks/api/useSystem'
import type {
  BatchUpdateSysConfigItem,
  SysConfigItem,
} from '@core/types/system'
import { useT } from '@/hooks/useT'

// ============================================================
// 系统管理 / 系统配置 — 对齐 v1 webcode/src/pages/system/SystemConfig
// 真实数据 useSysConfigsByCategory（adminApi.getSysConfigsByCategory，按分类拉 KV）
//   + useBatchUpdateSysConfigs（POST /admin/sysConfig/batch）。
// 后端 7 个分类：basic / security / device / notify / storage / omc / northbound。
// v1 的强类型表单 + 保留策略页签未逐项复刻；这里用通用 KV 编辑器覆盖全部分类，
// bool/int/float/string/json 原样回写（value_type 透传）。详见返回 notes。
// ============================================================

const CATEGORIES: { key: string; label: string }[] = [
  { key: 'basic', label: '基础' },
  { key: 'security', label: '安全' },
  { key: 'device', label: '设备' },
  { key: 'notify', label: '通知' },
  { key: 'storage', label: '存储' },
  { key: 'omc', label: 'OMC' },
  { key: 'northbound', label: '北向' },
]

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

  const items = useMemo<SysConfigItem[]>(() => data ?? [], [data])
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
                  {isBool ? (
                    <div className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        className="size-4 cursor-pointer accent-primary"
                        checked={cur === 'true' || cur === '1'}
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
      <CategoryEditor key={category} category={category} />
    </PageShell>
  )
}
