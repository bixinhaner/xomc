import { useEffect, useMemo, useState } from 'react'
import { Loader2, RotateCcw, Save } from 'lucide-react'

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

// ============================================================
// 系统管理 / 界面定制 — 对齐 v1 webcode/src/pages/system/UICustomization
// 真实数据 useSysConfigsByCategory('ui_custom') + useBatchUpdateSysConfigs。
// PRD docs/prd/system/ui-customization.md §5：GET /admin/sysConfig?category=ui_custom，
// POST /admin/sysConfig/batch 一次写所有项。图片资产上传（useUploadUIAsset）未复刻，
// 这里覆盖系统名 / 标语 / 主题色等文本项的查改存。详见返回 notes。
// ============================================================

// ui_custom 已知键的中文标签（缺失键回退原 key 展示）。
const FIELD_LABELS: Record<string, string> = {
  system_name: '系统名称',
  systemName: '系统名称',
  login_title: '登录页标题',
  loginTitle: '登录页标题',
  login_subtitle: '登录页副标题',
  loginSubtitle: '登录页副标题',
  copyright: '版权信息',
  primary_color: '主题色',
  primaryColor: '主题色',
  logo_text: 'Logo 文字',
  logoText: 'Logo 文字',
  footer_text: '页脚文字',
  footerText: '页脚文字',
}

export default function UICustomization() {
  const { data, isLoading, isError, error, refetch, isFetching } =
    useSysConfigsByCategory('ui_custom')
  const batchUpdate = useBatchUpdateSysConfigs()

  const items = useMemo<SysConfigItem[]>(() => data ?? [], [data])

  const [edits, setEdits] = useState<Record<string, string>>({})

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
      { category: 'ui_custom', items: payloadItems },
      { onSuccess: () => void refetch() }
    )
  }

  function handleReset() {
    const next: Record<string, string> = {}
    for (const it of items) next[it.key] = it.value ?? ''
    setEdits(next)
  }

  return (
    <PageShell
      title="界面定制"
      description="系统名称 · 登录页 · 主题色等界面文案配置"
      isFetching={isFetching}
    >
      {isLoading ? (
        <div className="flex h-40 items-center justify-center text-muted-foreground">
          <Loader2 className="size-5 animate-spin" />
        </div>
      ) : isError ? (
        <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </div>
      ) : items.length === 0 ? (
        <div className="rounded-lg border bg-card px-4 py-8 text-center text-sm text-muted-foreground">
          暂无界面定制配置项
        </div>
      ) : (
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

          <div className="max-w-2xl space-y-4 rounded-lg border bg-card p-5">
            {items.map((it) => {
              const isColor =
                it.key.toLowerCase().includes('color') &&
                /^#?[0-9a-fA-F]{0,8}$/.test(edits[it.key] ?? '')
              const cur = edits[it.key] ?? ''
              return (
                <div key={it.id || it.key} className="space-y-1.5">
                  <Label className="text-sm">
                    {FIELD_LABELS[it.key] ?? it.key}
                  </Label>
                  <div className="flex items-center gap-2">
                    {isColor ? (
                      <input
                        type="color"
                        className="size-9 cursor-pointer rounded border bg-transparent"
                        value={cur && cur.startsWith('#') ? cur : '#000000'}
                        onChange={(e) =>
                          setEdits((prev) => ({
                            ...prev,
                            [it.key]: e.target.value,
                          }))
                        }
                      />
                    ) : null}
                    <Input
                      value={cur}
                      onChange={(e) =>
                        setEdits((prev) => ({
                          ...prev,
                          [it.key]: e.target.value,
                        }))
                      }
                    />
                  </div>
                  {it.description ? (
                    <p className="text-xs text-muted-foreground">
                      {it.description}
                    </p>
                  ) : null}
                </div>
              )
            })}
          </div>

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
          </div>
        </div>
      )}
    </PageShell>
  )
}
