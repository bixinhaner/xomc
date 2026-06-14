import { useState } from 'react'
import { RotateCw, CheckCircle2, AlertTriangle, Loader2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { useReloadDictLoader } from '@core/hooks/api/useDictLoader'
import {
  KNOWN_DICT_LOADERS,
  type DictLoaderName,
  type DictLoaderReloadResult,
} from '@core/services/api/dictLoaderApi'

import { MiniStat } from './_shared'

type ResultMap = Partial<Record<DictLoaderName, DictLoaderReloadResult>>
type ErrorMap = Partial<Record<DictLoaderName, string>>

export default function DictLoader() {
  const reload = useReloadDictLoader()
  const [results, setResults] = useState<ResultMap>({})
  const [errors, setErrors] = useState<ErrorMap>({})
  const [active, setActive] = useState<DictLoaderName | null>(null)

  const handleReload = (name: DictLoaderName) => {
    setActive(name)
    setErrors((e) => ({ ...e, [name]: undefined }))
    reload
      .mutateAsync(name)
      .then((res) => {
        setResults((r) => ({ ...r, [name]: res }))
      })
      .catch((err: unknown) => {
        setErrors((e) => ({
          ...e,
          [name]: err instanceof Error ? err.message : '重载失败',
        }))
      })
      .finally(() => setActive(null))
  }

  const reloadedCount = Object.keys(results).length

  return (
    <PageShell
      code="F06"
      title="DICT LOADER · 字典热重载"
      subtitle="HOT RELOAD · DICTLOAD"
    >
      <div className="space-y-5">
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <MiniStat label="可重载 Loader" value={KNOWN_DICT_LOADERS.length} color="#00f0ff" icon={<RotateCw className="size-3.5" />} />
          <MiniStat label="本次已重载" value={reloadedCount} color="#00ff88" />
          <MiniStat
            label="失败"
            value={Object.values(errors).filter(Boolean).length}
            color={Object.values(errors).some(Boolean) ? '#ff2d6f' : '#525a78'}
          />
          <MiniStat label="状态" value={active ? 'RUNNING' : 'IDLE'} color="#a855f7" />
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          {KNOWN_DICT_LOADERS.map((loader) => {
            const res = results[loader.name]
            const err = errors[loader.name]
            const running = active === loader.name && reload.isPending
            return (
              <GlassPanel
                key={loader.name}
                strong
                title={loader.labelI18n['zh-CN']}
                meta={loader.name}
                className="p-4"
              >
                <div className="space-y-3">
                  <p className="text-xs leading-relaxed text-cyan-100/70">
                    {loader.descriptionI18n['zh-CN']}
                  </p>

                  <NeonButton
                    icon={running ? <Loader2 className="animate-spin" /> : <RotateCw />}
                    onClick={() => handleReload(loader.name)}
                    disabled={running}
                  >
                    {running ? '重载中…' : '热重载'}
                  </NeonButton>

                  {err ? (
                    <div className="flex items-start gap-2 border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-[11px] text-rose-300">
                      <AlertTriangle className="mt-0.5 size-3.5 shrink-0" />
                      {err}
                    </div>
                  ) : null}

                  {res ? (
                    <div className="space-y-1.5 border border-emerald-500/25 bg-emerald-500/5 px-3 py-2.5 font-mono text-[11px] text-cyan-100/85">
                      <div className="flex items-center gap-1.5 text-emerald-300">
                        <CheckCircle2 className="size-3.5" />
                        重载成功 · {res.elapsed_ms} ms
                      </div>
                      <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-cyan-300/75">
                        <span>影响行数: {res.rows_affected}</span>
                        <span>加载文件: {res.files_loaded}</span>
                        <span>跳过文件: {res.files_skipped}</span>
                        {res.cache_keys_cleared !== undefined ? (
                          <span>清缓存键: {res.cache_keys_cleared}</span>
                        ) : null}
                        {res.cache_version_after !== undefined ? (
                          <span>缓存版本: {res.cache_version_after}</span>
                        ) : null}
                      </div>
                      {res.errors && res.errors.length > 0 ? (
                        <div className="text-rose-300">
                          告警: {res.errors.join('；')}
                        </div>
                      ) : null}
                    </div>
                  ) : null}
                </div>
              </GlassPanel>
            )
          })}
        </div>
      </div>
    </PageShell>
  )
}
