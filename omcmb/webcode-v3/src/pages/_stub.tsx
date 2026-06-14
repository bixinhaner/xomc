import type { ReactNode } from 'react'
import { Construction, Sparkles } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'

export interface StubPageProps {
  code: string
  title: string
  subtitle?: string
  /** 计划要做的事 — 用 4-6 句话描述将来这页要呈现什么 */
  plan: ReactNode[]
  /** 一些可视化的占位区块（grid item） */
  blocks?: { label: string; value: string; tone?: 'cyan' | 'magenta' | 'amber' | 'lime' | 'violet' }[]
}

const TONE_COLOR: Record<string, string> = {
  cyan: '#00f0ff',
  magenta: '#ff00aa',
  amber: '#ffaa00',
  lime: '#00ff88',
  violet: '#a855f7',
}

export function StubPage({ code, title, subtitle, plan, blocks }: StubPageProps) {
  return (
    <PageShell code={code} title={title} subtitle={subtitle ?? 'MODULE READY · 视觉框架已就位'}>
      <div className="grid h-full grid-cols-12 gap-3">
        {/* 左：欢迎 + 计划 */}
        <div className="col-span-12 lg:col-span-7">
          <GlassPanel
            title="OPERATIONAL BRIEFING"
            meta="MODULE PLAN"
            className="h-full"
          >
            <div className="p-5">
              <div className="mb-3 flex items-center gap-2">
                <Sparkles className="size-4 text-cyan-300" />
                <span className="font-display text-sm font-bold tracking-[0.2em] text-cyan-100">
                  MODULE INITIALIZED
                </span>
              </div>
              <p className="mb-4 font-mono text-[12px] leading-relaxed text-cyan-200/85">
                STARFORGE 已在 v3 皮肤里为本模块准备好沉浸式 HUD 容器。
                数据层（@core hooks/services/store）已可直接调用——只需把视图替换为下方 ROADMAP 的实现即可。
              </p>
              <ul className="space-y-2 font-mono text-[12px] text-cyan-100/85">
                {plan.map((p, i) => (
                  <li key={i} className="flex gap-2">
                    <span className="shrink-0 text-cyan-300/65">
                      {String(i + 1).padStart(2, '0')} ⟶
                    </span>
                    <span>{p}</span>
                  </li>
                ))}
              </ul>
            </div>
          </GlassPanel>
        </div>

        {/* 右：占位指标 */}
        <div className="col-span-12 lg:col-span-5">
          <GlassPanel title="MODULE METRICS" meta="PLACEHOLDER" className="h-full">
            <div className="grid grid-cols-2 gap-3 p-4">
              {(blocks ?? defaultBlocks()).map((b, i) => {
                const c = TONE_COLOR[b.tone ?? 'cyan']
                return (
                  <div
                    key={i}
                    className="border border-cyan-500/15 bg-[#03050d]/40 px-3 py-3"
                  >
                    <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/60">
                      {b.label}
                    </div>
                    <div
                      className="mt-1 font-display text-2xl font-bold leading-none"
                      style={{ color: c, textShadow: `0 0 8px ${c}` }}
                    >
                      {b.value}
                    </div>
                  </div>
                )
              })}
              <div className="col-span-2 mt-2 flex items-center gap-2 text-[10px] text-cyan-300/55">
                <Construction className="size-3.5" />
                <span className="uppercase tracking-[0.2em]">UNDER ASSEMBLY</span>
              </div>
            </div>
          </GlassPanel>
        </div>
      </div>
    </PageShell>
  )
}

function defaultBlocks(): NonNullable<StubPageProps['blocks']> {
  return [
    { label: 'STATUS', value: 'READY', tone: 'lime' },
    { label: 'ENDPOINTS', value: '— / —', tone: 'cyan' },
    { label: 'SYNCED', value: '0', tone: 'amber' },
    { label: 'TODO', value: '∞', tone: 'magenta' },
  ]
}
