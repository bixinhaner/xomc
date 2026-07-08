import { useMemo, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { Bot, CheckCircle2, Plus, Send, ShieldCheck, Sparkles, User, X } from 'lucide-react'

import {
  AgentMarkdown,
  extractAgentRows,
  formatAgentDetail,
  formatAgentValue,
  type AgentPanelActivity,
  type AgentPanelMessage,
  type AgentProcessEntry,
  type AgentThoughtEntry,
} from '@core/agentkit'
import { useAgentPanelController } from '@core/hooks/useAgentPanelController'
import { useT } from '@/hooks/useT'
import { cn } from '@/lib/utils'

interface AgentPanelProps {
  open: boolean
  onClose: () => void
}

const markdownClassName = [
  'overflow-x-hidden break-words',
  '[&_h1]:mb-2 [&_h1]:mt-3 [&_h1]:text-lg [&_h1]:font-semibold [&_h1]:leading-tight',
  '[&_h2]:mb-2 [&_h2]:mt-3 [&_h2]:text-base [&_h2]:font-semibold [&_h2]:leading-tight',
  '[&_h3]:mb-2 [&_h3]:mt-3 [&_h3]:text-sm [&_h3]:font-semibold [&_h3]:leading-tight',
  '[&_p]:mb-2 [&_p:last-child]:mb-0',
  '[&_ul]:mb-2 [&_ul]:list-disc [&_ul]:pl-5',
  '[&_ol]:mb-2 [&_ol]:list-decimal [&_ol]:pl-5',
  '[&_li]:pl-0.5',
  '[&_li+li]:mt-1',
  '[&_blockquote]:mb-2 [&_blockquote]:border-l-2 [&_blockquote]:border-blue-200 [&_blockquote]:pl-3 [&_blockquote]:text-muted-foreground',
  '[&_a]:text-primary [&_a]:underline [&_a]:underline-offset-2',
  '[&_strong]:font-semibold [&_strong]:text-foreground',
  '[&_code]:rounded [&_code]:bg-muted [&_code]:px-1 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-[0.92em]',
  '[&_pre]:mb-2 [&_pre]:max-w-full [&_pre]:overflow-auto [&_pre]:rounded-md [&_pre]:bg-slate-950 [&_pre]:p-3 [&_pre]:text-slate-100',
  '[&_pre_code]:bg-transparent [&_pre_code]:p-0',
  '[&_.agent-render-table-scroll]:mb-2 [&_.agent-render-table-scroll]:max-w-full [&_.agent-render-table-scroll]:overflow-auto',
  '[&_table]:mb-2 [&_table]:w-full [&_table]:min-w-max [&_table]:border-collapse [&_table]:text-xs',
  '[&_th]:border [&_th]:border-border [&_th]:bg-muted/60 [&_th]:px-2 [&_th]:py-1.5 [&_th]:text-left',
  '[&_td]:border [&_td]:border-border [&_td]:px-2 [&_td]:py-1.5 [&_td]:align-top',
  '[&_.agent-render-image-card]:my-1 [&_.agent-render-image-card]:inline-block [&_.agent-render-image-card]:max-w-full',
  '[&_.agent-render-image]:max-h-80 [&_.agent-render-image]:max-w-full [&_.agent-render-image]:rounded-md [&_.agent-render-image]:border [&_.agent-render-image]:border-border [&_.agent-render-image]:object-contain',
  '[&_.agent-render-image-missing]:inline-block [&_.agent-render-image-missing]:rounded-md [&_.agent-render-image-missing]:bg-muted [&_.agent-render-image-missing]:px-2 [&_.agent-render-image-missing]:py-1 [&_.agent-render-image-missing]:text-xs [&_.agent-render-image-missing]:text-muted-foreground',
].join(' ')

function ThoughtBlock({
  thoughts,
  running,
}: {
  thoughts: AgentThoughtEntry[] | undefined
  running: boolean
}) {
  const t = useT()
  if (!thoughts?.length) return null
  const isStreaming = running || thoughts.some((thought) => thought.status === 'streaming')
  const lineCount = thoughts.reduce((total, thought) => total + Math.max(thought.lines.length, 1), 0)

  return (
    <details
      className={cn(
        'relative overflow-hidden rounded-lg border border-blue-100 bg-blue-50/70 p-3 text-xs text-muted-foreground',
        isStreaming && 'border-blue-200 bg-blue-50 shadow-[0_10px_24px_rgba(37,99,235,0.08)]'
      )}
      open={isStreaming}
    >
      <summary className="flex cursor-pointer list-none items-center gap-2 font-medium text-blue-800 [&::-webkit-details-marker]:hidden">
        <span
          className={cn(
            'size-1.5 rounded-full bg-blue-300',
            isStreaming && 'animate-pulse bg-blue-500 motion-reduce:animate-none'
          )}
          aria-hidden="true"
        />
        <span>{isStreaming ? t('agent.thinking') : t('agent.thoughtDone')}</span>
        {!isStreaming && <span className="rounded-full bg-white px-1.5 py-0.5 text-[11px] text-muted-foreground">{lineCount}</span>}
      </summary>
      {isStreaming && (
        <div className="mt-2 h-px overflow-hidden bg-blue-200/70" aria-hidden="true">
          <div className="h-full w-full animate-pulse bg-blue-500/70 motion-reduce:animate-none" />
        </div>
      )}
      <div className="mt-2 space-y-2 border-t border-blue-100 pt-2">
        {thoughts.map((thought) => (
          <div key={thought.id} className="space-y-1">
            {(thought.lines.length ? thought.lines : [thought.text]).map((line, index) => (
              <p key={`${thought.id}-${index}`} className="m-0 whitespace-pre-wrap leading-5">
                {line}
              </p>
            ))}
          </div>
        ))}
      </div>
    </details>
  )
}

function ProcessTrace({ entries }: { entries: AgentProcessEntry[] | undefined }) {
  const t = useT()
  if (!entries?.length) return null

  return (
    <details className="overflow-hidden rounded-lg border border-border" open={entries.some((entry) => entry.kind === 'error')}>
      <summary className="flex cursor-pointer list-none items-center justify-between bg-muted/50 px-3 py-2 text-xs font-medium text-muted-foreground [&::-webkit-details-marker]:hidden">
        <span>{t('agent.processTrace')}</span>
        <span className="rounded-full bg-background px-2 py-0.5">{entries.length}</span>
      </summary>
      <div className="space-y-2 p-3">
        {entries.map((entry, index) => {
          const detail = formatAgentDetail(entry.detail)
          const isActive = index === entries.length - 1 && entry.kind !== 'error'
          return (
            <div key={entry.id} className="grid grid-cols-[auto_minmax(0,1fr)] gap-2">
              <div className="flex items-start gap-2">
                <span
                  className={cn(
                    'mt-1.5 size-1.5 rounded-full bg-blue-200',
                    isActive && 'animate-pulse bg-blue-500 motion-reduce:animate-none',
                    entry.kind === 'error' && 'bg-destructive'
                  )}
                  aria-hidden="true"
                />
                <span
                  className={cn(
                    'self-start rounded-full bg-blue-50 px-2 py-0.5 text-[11px] font-semibold text-blue-700',
                    isActive && 'ring-1 ring-blue-200',
                    entry.kind === 'error' && 'bg-destructive/10 text-destructive'
                  )}
                >
                  {entry.kind}
                </span>
              </div>
              <div className="min-w-0">
                <div className="truncate text-xs font-medium text-foreground">{entry.title}</div>
                {detail && (
                  <pre className="mt-1 max-h-40 overflow-auto rounded-md bg-muted/60 p-2 text-[11px] leading-5 text-muted-foreground">
                    {detail}
                  </pre>
                )}
              </div>
            </div>
          )
        })}
      </div>
    </details>
  )
}

function AssistantContent({ message }: { message: AgentPanelMessage }) {
  const t = useT()
  const running = message.status === 'streaming'
  return (
    <div className="flex min-w-0 flex-col gap-3">
      <ThoughtBlock thoughts={message.thoughts} running={running} />
      {message.text ? (
        <div className={cn(running && 'after:ml-1 after:inline-block after:h-3.5 after:w-1.5 after:rounded-sm after:bg-blue-500 after:align-[-2px] after:content-[""] after:animate-pulse motion-reduce:after:animate-none')}>
          <AgentMarkdown className={markdownClassName} content={message.text} />
        </div>
      ) : running ? (
        <span className="inline-flex items-center gap-2 text-muted-foreground">
          {t('agent.streaming')}
          <span className="h-3.5 w-1.5 animate-pulse rounded-sm bg-blue-500 motion-reduce:animate-none" aria-hidden="true" />
        </span>
      ) : null}
      <ProcessTrace entries={message.process} />
    </div>
  )
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
  const isActive = activity.status === 'calling' || activity.status === 'running' || isPending

  return (
    <section
      className={cn(
        'relative overflow-hidden rounded-lg border border-border bg-card',
        isActive && 'border-blue-200 shadow-[0_12px_28px_rgba(37,99,235,0.08)]'
      )}
    >
      {isActive && <div className="absolute inset-y-0 left-0 w-0.5 animate-pulse bg-blue-500 motion-reduce:animate-none" aria-hidden="true" />}
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
  const controller = useAgentPanelController({ context, active: open })

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
            'inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium',
            controller.enabled ? 'bg-emerald-50 text-emerald-700' : 'bg-amber-50 text-amber-700'
          )}
        >
          {controller.enabled && (
            <span className="relative size-1.5 rounded-full bg-emerald-500 shadow-[0_0_0_3px_rgba(16,185,129,0.14)] after:absolute after:inset-[-4px] after:rounded-full after:border after:border-emerald-400/50 after:content-[''] after:animate-ping motion-reduce:after:animate-none" aria-hidden="true" />
          )}
          {controller.enabled ? t('agent.connected') : t('agent.disconnected')}
        </span>
        <button
          type="button"
          className="grid size-8 place-items-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-45"
          onClick={controller.clear}
          disabled={controller.isStreaming}
          aria-label={t('agent.newConversation')}
          title={t('agent.newConversation')}
        >
          <Plus className="size-4" />
        </button>
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
            className={cn('flex items-start gap-3', message.role === 'user' && 'flex-row-reverse')}
          >
            <div
              className={cn(
                'grid size-8 shrink-0 place-items-center rounded-full leading-none [&_svg]:block',
                message.role === 'user' ? 'bg-primary text-primary-foreground' : 'bg-muted text-primary',
                message.role === 'assistant' &&
                  message.status === 'streaming' &&
                  'animate-pulse ring-2 ring-blue-200 ring-offset-2 ring-offset-background motion-reduce:animate-none'
              )}
            >
              {message.role === 'user' ? <User className="size-4" /> : <Bot className="size-4" />}
            </div>
            <div
              className={cn(
                'max-w-[292px] rounded-lg border px-3 py-2 text-sm leading-6',
                message.role === 'user'
                  ? 'whitespace-pre-wrap border-primary/20 bg-primary/10'
                  : 'border-border bg-card'
              )}
            >
              {message.role === 'assistant' ? (
                <AssistantContent message={message} />
              ) : (
                message.text || (message.status === 'streaming' ? t('agent.streaming') : '')
              )}
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
