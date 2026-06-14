import { useState } from 'react'
import { CheckCircle2, Loader2, RefreshCw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { PageShell } from '@/components/layout/PageShell'

import { useReloadDictLoader } from '@core/hooks/api/useDictLoader'
import {
  KNOWN_DICT_LOADERS,
  type DictLoaderName,
  type DictLoaderReloadResult,
} from '@core/services/api/dictLoaderApi'

// ============================================================
// 系统管理 / 字典 Loader 重载 — 对齐 v1 webcode/src/pages/system/DictLoader
// 真实数据 useReloadDictLoader（dictLoaderApi.reload）。手编字典 XML 后热重载。
// ============================================================

function ResultStat({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div>
      <div className="text-[11px] uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className="mt-0.5 text-sm tabular-nums">{value}</div>
    </div>
  )
}

export default function DictLoader() {
  const reloadMutation = useReloadDictLoader()
  const [lastResult, setLastResult] = useState<
    Record<string, DictLoaderReloadResult | undefined>
  >({})
  const [lastError, setLastError] = useState<Record<string, string | undefined>>({})

  const handleReload = (name: DictLoaderName) => {
    reloadMutation.mutate(name, {
      onSuccess: (result) => {
        setLastResult((prev) => ({ ...prev, [name]: result }))
        setLastError((prev) => ({ ...prev, [name]: undefined }))
      },
      onError: (err) => {
        setLastError((prev) => ({
          ...prev,
          [name]: err instanceof Error ? err.message : '未知错误',
        }))
      },
    })
  }

  return (
    <PageShell
      title="字典 Loader 重载"
      description="手工编辑字典 XML（如 paramModel）后，点击对应按钮触发后端重读 + 自动清缓存"
    >
      <div className="space-y-3">
        {KNOWN_DICT_LOADERS.map((loader) => {
          const result = lastResult[loader.name]
          const errMsg = lastError[loader.name]
          const isLoading =
            reloadMutation.isPending && reloadMutation.variables === loader.name
          return (
            <div
              key={loader.name}
              className="rounded-lg border bg-card px-4 py-3"
            >
              <div className="flex items-start justify-between gap-4">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">
                      {loader.labelI18n['zh-CN']}
                    </span>
                    <Badge variant="muted">{loader.name}</Badge>
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {loader.descriptionI18n['zh-CN']}
                  </p>
                </div>
                <Button
                  size="sm"
                  disabled={isLoading}
                  onClick={() => handleReload(loader.name)}
                >
                  {isLoading ? (
                    <Loader2 className="size-4 animate-spin" />
                  ) : (
                    <RefreshCw className="size-4" />
                  )}
                  重新加载
                </Button>
              </div>

              {errMsg ? (
                <div className="mt-3 rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-xs text-destructive">
                  重载失败：{errMsg}
                </div>
              ) : result ? (
                <div className="mt-3 grid grid-cols-2 gap-3 border-t pt-3 sm:grid-cols-4">
                  <ResultStat
                    label="文件"
                    value={`${result.files_loaded} 载入 / ${result.files_skipped} 跳过`}
                  />
                  <ResultStat label="行" value={result.rows_affected} />
                  <ResultStat label="耗时" value={`${result.elapsed_ms} ms`} />
                  {result.cache_keys_cleared !== undefined ? (
                    <ResultStat
                      label="缓存"
                      value={`清 ${result.cache_keys_cleared} 键 · v${result.cache_version_after ?? '?'}`}
                    />
                  ) : (
                    <ResultStat
                      label="状态"
                      value={
                        result.errors && result.errors.length > 0 ? (
                          <span className="text-destructive">
                            {result.errors.join('; ')}
                          </span>
                        ) : (
                          <span className="inline-flex items-center gap-1 text-emerald-600 dark:text-emerald-400">
                            <CheckCircle2 className="size-3.5" /> OK
                          </span>
                        )
                      }
                    />
                  )}
                </div>
              ) : null}
            </div>
          )
        })}
      </div>
    </PageShell>
  )
}
