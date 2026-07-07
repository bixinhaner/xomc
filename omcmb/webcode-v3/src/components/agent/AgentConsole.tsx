import { useMemo, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { Bot, Send, ShieldCheck, Sparkles, User, X } from 'lucide-react'

import {
  extractAgentRows,
  formatAgentValue,
  type AgentPanelActivity,
} from '@core/agentkit'
import { useAgentPanelController } from '@core/hooks/useAgentPanelController'
import { useT } from '@/hooks/useT'
import { cn } from '@/lib/utils'

interface AgentConsoleProps {
  open: boolean
  onClose: () => void
}

function ActivityCard({
  activity,
  isPending,
  isStreaming,
  onExecute,
  onCancel,
}: {
  activity: AgentPanelActivity
  isPending: boolean
  isStreaming: boolean
  onExecute: () => void
  onCancel: () => void
}) {
  const t = useT()
  const inputRows = Object.entries(activity.request?.input ?? {})
  const rows = extractAgentRows(activity.output ?? activity.preview, 6)
  const isError = activity.status === 'error'

  return (
    <section className="border border-cyan-400/35 bg-cyan-500/[0.04] shadow-[0_0_18px_rgba(0,240,255,0.12)]">
      <div className="flex items-center justify-between border-b border-cyan-400/25 px-3 py-2">
        <div className="flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.18em] text-cyan-200">
          <Sparkles className="size-3.5" />
          {activity.status === 'preview' ? t('agent.actionPreview') : t('agent.callingTool')}
        </div>
        <span className="border border-amber-300/60 px-2 py-0.5 font-mono text-[10px] uppercase tracking-[0.18em] text-amber-200">
          {t('agent.readOnly')}
        </span>
      </div>
      <div className="space-y-3 p-3 font-mono text-xs text-cyan-100/80">
        <div className="grid grid-cols-[86px_1fr] gap-3">
          <span className="text-cyan-300/60">{t('agent.tool')}</span>
          <strong className="truncate text-cyan-100">{activity.request?.actionId || activity.title}</strong>
        </div>
        {activity.summary && <p className="leading-5 text-cyan-100/70">{activity.summary}</p>}
        {inputRows.length > 0 && (
          <div className="border border-cyan-400/20">
            {inputRows.map(([key, value]) => (
              <div key={key} className="grid grid-cols-[42%_1fr] border-b border-cyan-400/15 last:border-b-0">
                <div className="bg-cyan-400/[0.04] px-3 py-2 text-cyan-300">{key}</div>
                <div className="break-words px-3 py-2 text-cyan-100">{formatAgentValue(value)}</div>
              </div>
            ))}
          </div>
        )}
        <div className="flex items-center gap-2 text-emerald-300">
          <ShieldCheck className="size-4" />
          {t('agent.readOnlyPolicy')}
        </div>
        {isPending && (
          <div className="grid grid-cols-2 gap-2">
            <button
              type="button"
              className="h-9 border border-cyan-300/70 bg-cyan-300/15 font-mono text-xs uppercase tracking-[0.18em] text-cyan-100 shadow-[0_0_14px_rgba(0,240,255,0.2)] disabled:opacity-45"
              disabled={isStreaming}
              onClick={onExecute}
            >
              {t('agent.execute')}
            </button>
            <button
              type="button"
              className="h-9 border border-cyan-500/30 bg-cyan-500/[0.04] font-mono text-xs uppercase tracking-[0.18em] text-cyan-200/75 disabled:opacity-45"
              disabled={isStreaming}
              onClick={onCancel}
            >
              {t('agent.cancel')}
            </button>
          </div>
        )}
        {!isPending && activity.status !== 'calling' && (
          <div>
            <div className="mb-2 text-[10px] uppercase tracking-[0.18em] text-cyan-300/70">
              {isError ? t('agent.failed') : t('agent.previewResult')}
            </div>
            <div className="border border-cyan-400/20">
              {isError ? (
                <div className="grid grid-cols-[42%_1fr]">
                  <div className="bg-rose-400/[0.08] px-3 py-2 text-rose-300">{activity.error?.code ?? 'error'}</div>
                  <div className="break-words px-3 py-2 text-rose-100">{activity.error?.message ?? t('agent.failed')}</div>
                </div>
              ) : rows.length > 0 ? (
                rows.map((row) => (
                  <div key={row.key} className="grid grid-cols-[42%_1fr] border-b border-cyan-400/15 last:border-b-0">
                    <div className="bg-cyan-400/[0.04] px-3 py-2 text-cyan-300">{row.key}</div>
                    <div className="break-words px-3 py-2 text-cyan-100">{row.value}</div>
                  </div>
                ))
              ) : (
                <div className="px-3 py-2 text-cyan-300/65">{t('agent.noRows')}</div>
              )}
            </div>
          </div>
        )}
      </div>
    </section>
  )
}

export function AgentConsole({ open, onClose }: AgentConsoleProps) {
  const t = useT()
  const location = useLocation()
  const [input, setInput] = useState('')
  const context = useMemo(
    () => ({
      path: location.pathname,
      query: Object.fromEntries(new URLSearchParams(location.search).entries()),
    }),
    [location.pathname, location.search]
  )
  const controller = useAgentPanelController({ context })

  if (!open) return null

  const submit = async (event: React.FormEvent) => {
    event.preventDefault()
    const text = input.trim()
    if (!text) return
    setInput('')
    await controller.sendMessage(text)
  }

  return (
    <aside className="fixed bottom-24 right-4 top-[76px] z-40 flex w-[min(400px,calc(100vw-32px))] flex-col border border-cyan-400/55 bg-[#061225] shadow-[0_0_28px_rgba(0,240,255,0.24)]">
      <div className="pointer-events-none absolute inset-0 scanline" />
      <header className="relative flex h-12 shrink-0 items-center gap-3 border-b border-cyan-400/30 px-3">
        <div className="flex min-w-0 flex-1 items-center gap-2 font-mono text-sm uppercase tracking-[0.2em] text-cyan-100">
          <Bot className="size-4 text-cyan-300" />
          <span>{t('agent.title')}</span>
        </div>
        <span
          className={cn(
            'font-mono text-[10px] uppercase tracking-[0.18em]',
            controller.enabled ? 'text-emerald-300' : 'text-amber-300'
          )}
        >
          {controller.enabled ? t('agent.connected') : t('agent.disconnected')}
        </span>
        <button
          type="button"
          className="grid size-7 place-items-center text-cyan-300/70 hover:bg-cyan-400/10 hover:text-cyan-100"
          onClick={onClose}
          aria-label={t('agent.close')}
        >
          <X className="size-4" />
        </button>
      </header>

      <div className="relative min-h-0 flex-1 space-y-3 overflow-y-auto p-3">
        {!controller.enabled && (
          <div className="border border-cyan-400/25 p-3 font-mono text-xs leading-5 text-cyan-100/70">
            <div className="mb-1 text-cyan-100">{t('agent.disabledTitle')}</div>
            {t('agent.disabledHint')}
          </div>
        )}
        {controller.enabled && controller.messages.length === 0 && controller.activities.length === 0 && (
          <div className="border border-cyan-400/25 p-3 font-mono text-xs leading-5 text-cyan-100/70">
            <div className="mb-1 text-cyan-100">{t('agent.emptyTitle')}</div>
            {t('agent.emptyHint')}
          </div>
        )}
        {controller.messages.map((message) => (
          <div
            key={message.id}
            className={cn('flex gap-2', message.role === 'user' && 'flex-row-reverse')}
          >
            <div
              className={cn(
                'grid size-8 shrink-0 place-items-center border',
                message.role === 'user'
                  ? 'border-cyan-300/50 bg-cyan-300/15 text-cyan-100'
                  : 'border-cyan-500/30 bg-cyan-500/10 text-cyan-300'
              )}
            >
              {message.role === 'user' ? <User className="size-4" /> : <Bot className="size-4" />}
            </div>
            <div
              className={cn(
                'max-w-[292px] whitespace-pre-wrap border px-3 py-2 font-mono text-xs leading-5',
                message.role === 'user'
                  ? 'border-cyan-300/40 bg-cyan-300/10 text-cyan-50'
                  : 'border-cyan-500/25 bg-cyan-500/[0.04] text-cyan-100/80'
              )}
            >
              {message.text || (message.status === 'streaming' ? t('agent.streaming') : '')}
            </div>
          </div>
        ))}
        {controller.activities.map((activity) => (
          <ActivityCard
            key={activity.callId}
            activity={activity}
            isPending={controller.pendingAction?.callId === activity.callId}
            isStreaming={controller.isStreaming}
            onExecute={controller.executePendingAction}
            onCancel={controller.cancelPendingAction}
          />
        ))}
        {controller.error && (
          <div className="border border-rose-400/30 bg-rose-500/10 p-3 font-mono text-xs text-rose-100">
            {t('agent.errorPrefix')}: {controller.error.message}
          </div>
        )}
      </div>

      <form className="relative grid shrink-0 grid-cols-[1fr_44px] gap-2 border-t border-cyan-400/25 p-3" onSubmit={submit}>
        <input
          className="h-11 min-w-0 border border-cyan-400/30 bg-cyan-500/[0.04] px-3 font-mono text-xs text-cyan-100 outline-none placeholder:text-cyan-300/45 focus:border-cyan-300/70"
          value={input}
          onChange={(event) => setInput(event.target.value)}
          placeholder={t('agent.placeholder')}
          disabled={!controller.enabled || controller.isStreaming}
        />
        <button
          type="submit"
          className="grid h-11 place-items-center border border-cyan-300/60 bg-cyan-300/15 text-cyan-100 shadow-[0_0_14px_rgba(0,240,255,0.2)] disabled:opacity-45"
          disabled={!controller.enabled || controller.isStreaming || input.trim() === ''}
          aria-label={t('agent.send')}
        >
          <Send className="size-4" />
        </button>
      </form>
    </aside>
  )
}
