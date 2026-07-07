import { useMemo, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { Bot, CheckCircle2, Send, ShieldCheck, Sparkles, User, X } from 'lucide-react'

import {
  extractAgentRows,
  formatAgentValue,
  type AgentPanelActivity,
} from '@core/agentkit'
import { useAgentPanelController } from '@core/hooks/useAgentPanelController'
import { useT } from '@/hooks/useT'
import { cn } from '@/lib/utils'

interface AgentPanelProps {
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
    <section className="rounded-lg border border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <div className="flex items-center gap-2 text-sm font-semibold">
          <Sparkles className="size-4 text-primary" />
          {activity.status === 'preview' ? t('agent.actionPreview') : t('agent.callingTool')}
        </div>
        <span className="rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700">
          {t('agent.readOnly')}
        </span>
      </div>
      <div className="space-y-3 p-4 text-sm">
        <div className="flex items-center justify-between gap-3">
          <span className="text-muted-foreground">{t('agent.tool')}</span>
          <strong className="truncate text-foreground">{activity.request?.actionId || activity.title}</strong>
        </div>
        {activity.summary && <p className="text-muted-foreground">{activity.summary}</p>}
        {inputRows.length > 0 && (
          <div>
            <div className="mb-2 text-xs font-medium text-muted-foreground">{t('agent.parameters')}</div>
            <div className="overflow-hidden rounded-md border border-border">
              {inputRows.map(([key, value]) => (
                <div key={key} className="grid grid-cols-[42%_1fr] border-b border-border last:border-b-0">
                  <div className="bg-muted/40 px-3 py-2 text-xs text-muted-foreground">{key}</div>
                  <div className="break-words px-3 py-2 text-xs">{formatAgentValue(value)}</div>
                </div>
              ))}
            </div>
          </div>
        )}
        <div className="flex items-center gap-2 rounded-md border border-emerald-100 bg-emerald-50 px-3 py-2 text-xs text-emerald-700">
          <ShieldCheck className="size-4" />
          {t('agent.readOnlyPolicy')}
        </div>
        {isPending && (
          <div className="grid grid-cols-2 gap-2">
            <button
              type="button"
              className="inline-flex h-9 items-center justify-center rounded-md bg-primary text-sm font-medium text-primary-foreground disabled:opacity-50"
              disabled={isStreaming}
              onClick={onExecute}
            >
              {t('agent.execute')}
            </button>
            <button
              type="button"
              className="inline-flex h-9 items-center justify-center rounded-md border border-border bg-background text-sm font-medium disabled:opacity-50"
              disabled={isStreaming}
              onClick={onCancel}
            >
              {t('agent.cancel')}
            </button>
          </div>
        )}
        {!isPending && activity.status !== 'calling' && (
          <div>
            <div className="mb-2 text-xs font-medium text-muted-foreground">
              {isError ? t('agent.failed') : t('agent.previewResult')}
            </div>
            <div className="overflow-hidden rounded-md border border-border">
              {isError ? (
                <div className="grid grid-cols-[42%_1fr]">
                  <div className="bg-muted/40 px-3 py-2 text-xs text-muted-foreground">{activity.error?.code ?? 'error'}</div>
                  <div className="break-words px-3 py-2 text-xs">{activity.error?.message ?? t('agent.failed')}</div>
                </div>
              ) : rows.length > 0 ? (
                rows.map((row) => (
                  <div key={row.key} className="grid grid-cols-[42%_1fr] border-b border-border last:border-b-0">
                    <div className="bg-muted/40 px-3 py-2 text-xs text-muted-foreground">{row.key}</div>
                    <div className="break-words px-3 py-2 text-xs">{row.value}</div>
                  </div>
                ))
              ) : (
                <div className="px-3 py-2 text-xs text-muted-foreground">{t('agent.noRows')}</div>
              )}
            </div>
          </div>
        )}
      </div>
    </section>
  )
}

export function AgentPanel({ open, onClose }: AgentPanelProps) {
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
    <aside className="fixed bottom-0 right-0 top-14 z-50 flex w-[min(400px,100vw)] flex-col border-l border-border bg-background shadow-2xl">
      <header className="flex h-16 shrink-0 items-center gap-3 border-b border-border px-5">
        <div className="flex min-w-0 flex-1 items-center gap-2 text-base font-semibold">
          <Bot className="size-5 text-primary" />
          <span>{t('agent.title')}</span>
        </div>
        <span
          className={cn(
            'rounded-full px-2 py-0.5 text-xs font-medium',
            controller.enabled ? 'bg-emerald-50 text-emerald-700' : 'bg-amber-50 text-amber-700'
          )}
        >
          {controller.enabled ? t('agent.connected') : t('agent.disconnected')}
        </span>
        <button
          type="button"
          className="grid size-8 place-items-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground"
          onClick={onClose}
          aria-label={t('agent.close')}
        >
          <X className="size-4" />
        </button>
      </header>

      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto p-4">
        {!controller.enabled && (
          <div className="rounded-lg border border-dashed border-border p-4 text-sm text-muted-foreground">
            <div className="mb-1 font-medium text-foreground">{t('agent.disabledTitle')}</div>
            {t('agent.disabledHint')}
          </div>
        )}
        {controller.enabled && controller.messages.length === 0 && controller.activities.length === 0 && (
          <div className="rounded-lg border border-dashed border-border p-4 text-sm text-muted-foreground">
            <div className="mb-1 font-medium text-foreground">{t('agent.emptyTitle')}</div>
            {t('agent.emptyHint')}
          </div>
        )}
        {controller.messages.map((message) => (
          <div
            key={message.id}
            className={cn('flex gap-3', message.role === 'user' && 'flex-row-reverse')}
          >
            <div
              className={cn(
                'grid size-8 shrink-0 place-items-center rounded-full',
                message.role === 'user' ? 'bg-primary text-primary-foreground' : 'bg-muted text-primary'
              )}
            >
              {message.role === 'user' ? <User className="size-4" /> : <Bot className="size-4" />}
            </div>
            <div
              className={cn(
                'max-w-[292px] whitespace-pre-wrap rounded-lg border px-3 py-2 text-sm leading-6',
                message.role === 'user'
                  ? 'border-primary/20 bg-primary/10'
                  : 'border-border bg-card'
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
          <div className="rounded-lg border border-destructive/20 bg-destructive/5 p-3 text-sm text-destructive">
            {t('agent.errorPrefix')}: {controller.error.message}
          </div>
        )}
      </div>

      <form className="grid shrink-0 grid-cols-[1fr_44px] gap-2 border-t border-border p-4" onSubmit={submit}>
        <input
          className="h-11 min-w-0 rounded-md border border-input bg-background px-3 text-sm outline-none focus:border-primary"
          value={input}
          onChange={(event) => setInput(event.target.value)}
          placeholder={t('agent.placeholder')}
          disabled={!controller.enabled || controller.isStreaming}
        />
        <button
          type="submit"
          className="grid h-11 place-items-center rounded-md bg-primary text-primary-foreground disabled:opacity-50"
          disabled={!controller.enabled || controller.isStreaming || input.trim() === ''}
          aria-label={t('agent.send')}
        >
          {controller.isStreaming ? <CheckCircle2 className="size-4" /> : <Send className="size-4" />}
        </button>
      </form>
    </aside>
  )
}

