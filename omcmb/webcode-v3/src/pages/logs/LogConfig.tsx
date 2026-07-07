import { useEffect, useMemo, useState } from 'react'
import { Check, RotateCcw, Save, SlidersHorizontal } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import {
  useBatchUpdateSysConfigs,
  useSysConfigsByCategory,
} from '@core/hooks/api/useSystem'
import type { SysConfigItem, SysConfigValueType } from '@core/types/system'

import { FeedError, FeedLoading } from './_shared'

// ──────────────────────────────────────────────────────────────────────────
// /logs/config —— 日志保留与轮转配置（对照 v1 webcode log/config · LogConfig）
// 真实端点：useSysConfigsByCategory('log.retention' | 'log.rotation')
//          useBatchUpdateSysConfigs（一次保存 = 该分类一次 batch upsert，后端热加载）
// 字段与后端 sys_configs 键 + value_type 严格对齐（seed 000005）。
// HUD 形态：每分类一张玻璃卡片 + 受控数值/开关 + 保存。
// ──────────────────────────────────────────────────────────────────────────

type FieldType = 'int' | 'bool'

interface FieldSpec {
  key: string
  label: string
  type: FieldType
  min?: number
  max?: number
  suffix?: string
}

interface CardSpec {
  category: string
  title: string
  desc: string
  accent: string
  fields: FieldSpec[]
}

const DAYS: Omit<FieldSpec, 'key' | 'label'> = { type: 'int', min: 1, max: 3650, suffix: '天' }

const CARDS: CardSpec[] = [
  {
    category: 'log.retention',
    title: 'RETENTION · 审计 / 业务日志保留',
    desc: 'worker 每日 05:00 cron 批量删过期行；按表分别设置保留天数。',
    accent: '#00f0ff',
    fields: [
      { key: 'enabled', label: '启用保留清理', type: 'bool' },
      { key: 'audit_days', label: '审计日志', ...DAYS },
      { key: 'ops_audit_days', label: '运维审计', ...DAYS },
      { key: 'login_days', label: '登录日志', ...DAYS },
      { key: 'oper_days', label: '操作日志', ...DAYS },
      { key: 'task_days', label: '任务日志', ...DAYS },
      { key: 'system_days', label: '系统日志', ...DAYS },
      { key: 'ne_message_days', label: '网元报文', ...DAYS },
      { key: 'event_days', label: '设备事件', ...DAYS },
    ],
  },
  {
    category: 'log.rotation',
    title: 'ROTATION · 运行期日志文件轮转',
    desc: 'app / acs / worker logger override watcher 读取，≤1 分钟生效。',
    accent: '#a855f7',
    fields: [
      { key: 'max_size_mb', label: '单文件上限', type: 'int', min: 1, max: 10240, suffix: 'MB' },
      { key: 'rotate_interval_minutes', label: '轮转间隔', type: 'int', min: 1, max: 1440, suffix: '分钟' },
      { key: 'max_age_days', label: '最大保留', type: 'int', min: 1, max: 3650, suffix: '天' },
      { key: 'keep_files', label: '保留份数', type: 'int', min: 1, max: 1000, suffix: '份' },
    ],
  },
]

function decode(raw: string, type: FieldType): number | boolean {
  if (type === 'bool') return raw === 'true' || raw === '1'
  const n = parseInt(raw, 10)
  return Number.isFinite(n) ? n : 0
}

function encode(v: number | boolean, type: FieldType): { value: string; value_type: SysConfigValueType } {
  if (type === 'bool') return { value: v ? 'true' : 'false', value_type: 'bool' }
  return { value: String(v ?? 0), value_type: 'int' }
}

export default function LogConfigPage() {
  return (
    <PageShell
      code="F06"
      title="LOG CONFIG · 日志配置"
      subtitle="RETENTION & ROTATION"
    >
      <div className="flex flex-col gap-4">
        {CARDS.map((spec) => (
          <ConfigCard key={spec.category} spec={spec} />
        ))}
      </div>
    </PageShell>
  )
}

function ConfigCard({ spec }: { spec: CardSpec }) {
  const { data, isLoading, isError, error, isFetching } = useSysConfigsByCategory(spec.category)
  const batch = useBatchUpdateSysConfigs()

  const initial = useMemo<Record<string, number | boolean>>(() => {
    const out: Record<string, number | boolean> = {}
    for (const f of spec.fields) out[f.key] = f.type === 'bool' ? false : 0
    ;(data as SysConfigItem[] | undefined)?.forEach((cfg) => {
      const f = spec.fields.find((x) => x.key === cfg.key)
      if (f) out[cfg.key] = decode(cfg.value, f.type)
    })
    return out
  }, [data, spec.fields])

  const [values, setValues] = useState<Record<string, number | boolean>>(initial)
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    setValues(initial)
    setSaved(false)
  }, [initial])

  const dirty = useMemo(
    () => spec.fields.some((f) => values[f.key] !== initial[f.key]),
    [values, initial, spec.fields],
  )

  const handleSave = () => {
    batch.mutate(
      {
        category: spec.category,
        items: spec.fields.map((f) => {
          const enc = encode(values[f.key], f.type)
          return { key: f.key, value: enc.value, value_type: enc.value_type }
        }),
      },
      {
        onSuccess: () => {
          setSaved(true)
          window.setTimeout(() => setSaved(false), 2500)
        },
      },
    )
  }

  const accent = spec.accent

  return (
    <GlassPanel
      strong
      title={spec.title}
      meta={
        isFetching ? (
          <span className="text-cyan-300/50">SYNC…</span>
        ) : (
          <span className="text-cyan-300/40">{spec.category}</span>
        )
      }
    >
      <div className="p-4">
        <div className="mb-4 font-mono text-[11px] text-cyan-300/55">{spec.desc}</div>

        {isLoading ? (
          <FeedLoading />
        ) : isError ? (
          <FeedError error={error} />
        ) : (
          <>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {spec.fields.map((f) =>
                f.type === 'bool' ? (
                  <div
                    key={f.key}
                    className="flex items-center justify-between rounded-sm border-l-2 bg-black/20 px-3 py-2.5"
                    style={{ borderLeftColor: accent }}
                  >
                    <span className="font-mono text-[11px] uppercase tracking-[0.12em] text-cyan-300/70">
                      {f.label}
                    </span>
                    <button
                      type="button"
                      role="switch"
                      aria-checked={Boolean(values[f.key])}
                      onClick={() => setValues((p) => ({ ...p, [f.key]: !p[f.key] }))}
                      className="relative h-5 w-10 rounded-full transition-colors"
                      style={{
                        background: values[f.key] ? accent : '#2a3550',
                        boxShadow: values[f.key] ? `0 0 8px ${accent}` : 'none',
                      }}
                    >
                      <span
                        className="absolute top-0.5 size-4 rounded-full bg-white transition-all"
                        style={{ left: values[f.key] ? '1.375rem' : '0.125rem' }}
                      />
                    </button>
                  </div>
                ) : (
                  <label
                    key={f.key}
                    className="flex items-center justify-between rounded-sm border-l-2 bg-black/20 px-3 py-2"
                    style={{ borderLeftColor: accent }}
                  >
                    <span className="font-mono text-[11px] uppercase tracking-[0.12em] text-cyan-300/70">
                      {f.label}
                    </span>
                    <span className="flex items-center gap-1.5">
                      <input
                        type="number"
                        className="neon-input w-20 text-right tabular-nums"
                        value={Number(values[f.key])}
                        min={f.min}
                        max={f.max}
                        onChange={(e) => {
                          const raw = e.target.value
                          let n = raw === '' ? (f.min ?? 0) : parseInt(raw, 10)
                          if (!Number.isFinite(n)) n = f.min ?? 0
                          if (f.min !== undefined) n = Math.max(f.min, n)
                          if (f.max !== undefined) n = Math.min(f.max, n)
                          setValues((p) => ({ ...p, [f.key]: n }))
                        }}
                      />
                      {f.suffix && (
                        <span className="font-mono text-[10px] text-cyan-300/45">{f.suffix}</span>
                      )}
                    </span>
                  </label>
                ),
              )}
            </div>

            <div className="mt-4 flex items-center justify-end gap-2">
              {batch.isError && (
                <span className="font-mono text-[11px] text-rose-300">
                  保存失败 · {batch.error instanceof Error ? batch.error.message : '未知错误'}
                </span>
              )}
              {saved && (
                <span className="flex items-center gap-1 font-mono text-[11px] text-emerald-300">
                  <Check className="size-3.5" /> 已保存并热加载
                </span>
              )}
              <NeonButton
                icon={<RotateCcw />}
                onClick={() => setValues(initial)}
                disabled={!dirty || batch.isPending}
              >
                重置
              </NeonButton>
              <NeonButton
                icon={batch.isPending ? <SlidersHorizontal className="animate-pulse" /> : <Save />}
                onClick={handleSave}
                disabled={!dirty || batch.isPending}
              >
                {batch.isPending ? '保存中…' : '保存'}
              </NeonButton>
            </div>
          </>
        )}
      </div>
    </GlassPanel>
  )
}
