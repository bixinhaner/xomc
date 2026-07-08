import { useMemo, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { Bot, Plus, Send, ShieldCheck, Sparkles, User, X } from 'lucide-react'

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

interface AgentConsoleProps {
  open: boolean
  onClose: () => void
}

const markdownClassName = [
  'break-words font-mono',
  '[&_h1]:mb-2 [&_h1]:mt-3 [&_h1]:text-sm [&_h1]:font-semibold [&_h1]:leading-tight [&_h1]:text-cyan-50',
  '[&_h2]:mb-2 [&_h2]:mt-3 [&_h2]:text-xs [&_h2]:font-semibold [&_h2]:leading-tight [&_h2]:text-cyan-50',
  '[&_h3]:mb-2 [&_h3]:mt-3 [&_h3]:text-xs [&_h3]:font-semibold [&_h3]:leading-tight [&_h3]:text-cyan-100',
  '[&_p]:mb-2 [&_p:last-child]:mb-0',
  '[&_ul]:mb-2 [&_ul]:list-disc [&_ul]:pl-5',
  '[&_ol]:mb-2 [&_ol]:list-decimal [&_ol]:pl-5',
  '[&_li]:pl-0.5',
  '[&_li+li]:mt-1',
  '[&_blockquote]:mb-2 [&_blockquote]:border-l-2 [&_blockquote]:border-cyan-400/35 [&_blockquote]:pl-3 [&_blockquote]:text-cyan-100/60',
  '[&_a]:text-cyan-200 [&_a]:underline [&_a]:underline-offset-2',
  '[&_strong]:font-semibold [&_strong]:text-cyan-50',
  '[&_code]:border [&_code]:border-cyan-400/20 [&_code]:bg-cyan-400/10 [&_code]:px-1 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-cyan-50',
  '[&_pre]:mb-2 [&_pre]:max-w-full [&_pre]:overflow-auto [&_pre]:border [&_pre]:border-cyan-400/20 [&_pre]:bg-black/45 [&_pre]:p-3 [&_pre]:text-cyan-50',
  '[&_pre_code]:border-0 [&_pre_code]:bg-transparent [&_pre_code]:p-0',
  '[&_.agent-render-table-scroll]:mb-2 [&_.agent-render-table-scroll]:max-w-full [&_.agent-render-table-scroll]:overflow-auto',
  '[&_table]:mb-2 [&_table]:w-full [&_table]:min-w-max [&_table]:border-collapse [&_table]:text-[11px]',
  '[&_th]:border [&_th]:border-cyan-400/25 [&_th]:bg-cyan-400/10 [&_th]:px-2 [&_th]:py-1.5 [&_th]:text-left [&_th]:text-cyan-100',
  '[&_td]:border [&_td]:border-cyan-400/20 [&_td]:px-2 [&_td]:py-1.5 [&_td]:align-top',
  '[&_.agent-render-image-card]:my-1 [&_.agent-render-image-card]:inline-block [&_.agent-render-image-card]:max-w-full',
  '[&_.agent-render-image]:max-h-80 [&_.agent-render-image]:max-w-full [&_.agent-render-image]:border [&_.agent-render-image]:border-cyan-400/25 [&_.agent-render-image]:object-contain',
  '[&_.agent-render-image-missing]:inline-block [&_.agent-render-image-missing]:border [&_.agent-render-image-missing]:border-cyan-400/20 [&_.agent-render-image-missing]:bg-cyan-400/10 [&_.agent-render-image-missing]:px-2 [&_.agent-render-image-missing]:py-1 [&_.agent-render-image-missing]:text-[11px] [&_.agent-render-image-missing]:text-cyan-100/65',
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
        'relative overflow-hidden border border-cyan-400/25 bg-cyan-500/[0.04] p-3 text-cyan-100/70',
        isStreaming && 'border-cyan-300/45 bg-cyan-300/[0.08] shadow-[0_0_18px_rgba(0,240,255,0.12)]'
      )}
      open={isStreaming}
    >
      <summary className="flex cursor-pointer list-none items-center gap-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-200 [&::-webkit-details-marker]:hidden">
        <span
          className={cn(
            'size-1.5 rounded-full bg-cyan-500/60',
            isStreaming && 'animate-pulse bg-cyan-200 shadow-[0_0_10px_rgba(103,232,249,0.65)] motion-reduce:animate-none'
          )}
          aria-hidden="true"
        />
        <span>{isStreaming ? t('agent.thinking') : t('agent.thoughtDone')}</span>
        {!isStreaming && <span className="border border-cyan-400/30 px-1.5 py-0.5 text-cyan-300/75">{lineCount}</span>}
      </summary>
      {isStreaming && (
        <div className="mt-2 h-px bg-cyan-400/20" aria-hidden="true">
          <div className="h-full w-full animate-pulse bg-cyan-200/80 shadow-[0_0_12px_rgba(103,232,249,0.45)] motion-reduce:animate-none" />
        </div>
      )}
      <div className="mt-2 space-y-2 border-t border-cyan-400/20 pt-2">
        {thoughts.map((thought) => (
          <div key={thought.id} className="space-y-1">
            {(thought.lines.length ? thought.lines : [thought.text]).map((line, index) => (
              <p key={`${thought.id}-${index}`} className="m-0 whitespace-pre-wrap font-mono text-[11px] leading-5">
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
    <details className="border border-cyan-400/25" open={entries.some((entry) => entry.kind === 'error')}>
      <summary className="flex cursor-pointer list-none items-center justify-between bg-cyan-400/[0.05] px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300 [&::-webkit-details-marker]:hidden">
        <span>{t('agent.processTrace')}</span>
        <span className="border border-cyan-400/30 px-2 py-0.5">{entries.length}</span>
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
                    'mt-1.5 size-1.5 rounded-full bg-cyan-500/50',
                    isActive && 'animate-pulse bg-cyan-200 shadow-[0_0_10px_rgba(103,232,249,0.65)] motion-reduce:animate-none',
                    entry.kind === 'error' && 'bg-rose-300 shadow-none'
                  )}
                  aria-hidden="true"
                />
                <span
                  className={cn(
                    'self-start border border-cyan-400/30 bg-cyan-400/10 px-2 py-0.5 font-mono text-[10px] uppercase tracking-[0.14em] text-cyan-200',
                    isActive && 'border-cyan-200/60 bg-cyan-300/15',
                    entry.kind === 'error' && 'border-rose-400/40 bg-rose-400/10 text-rose-200'
                  )}
                >
                  {entry.kind}
                </span>
              </div>
              <div className="min-w-0">
                <div className="truncate font-mono text-[11px] text-cyan-100">{entry.title}</div>
                {detail && (
                  <pre className="mt-1 max-h-40 overflow-auto border border-cyan-400/15 bg-black/25 p-2 font-mono text-[11px] leading-5 text-cyan-100/65">
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
        <div className={cn(running && 'after:ml-1 after:inline-block after:h-3.5 after:w-1.5 after:bg-cyan-200 after:align-[-2px] after:shadow-[0_0_8px_rgba(103,232,249,0.6)] after:content-[""] after:animate-pulse motion-reduce:after:animate-none')}>
          <AgentMarkdown className={markdownClassName} content={message.text} />
        </div>
      ) : running ? (
        <span className="inline-flex items-center gap-2 text-cyan-300/65">
          {t('agent.streaming')}
          <span className="h-3.5 w-1.5 animate-pulse bg-cyan-200 shadow-[0_0_8px_rgba(103,232,249,0.6)] motion-reduce:animate-none" aria-hidden="true" />
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
        'relative overflow-hidden border border-cyan-400/35 bg-cyan-500/[0.04] shadow-[0_0_18px_rgba(0,240,255,0.12)]',
        isActive && 'border-cyan-200/60 shadow-[0_0_24px_rgba(0,240,255,0.18)]'
      )}
    >
      {isActive && <div className="absolute inset-y-0 left-0 w-0.5 animate-pulse bg-cyan-200 shadow-[0_0_14px_rgba(103,232,249,0.6)] motion-reduce:animate-none" aria-hidden="true" />}
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
            'inline-flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.18em]',
            controller.enabled ? 'text-emerald-300' : 'text-amber-300'
          )}
        >
          {controller.enabled && (
            <span className="relative size-1.5 rounded-full bg-emerald-300 shadow-[0_0_10px_rgba(110,231,183,0.65)] after:absolute after:inset-[-4px] after:rounded-full after:border after:border-emerald-300/50 after:content-[''] after:animate-ping motion-reduce:after:animate-none" aria-hidden="true" />
          )}
          {controller.enabled ? t('agent.connected') : t('agent.disconnected')}
        </span>
        <button
          type="button"
          className="grid size-7 place-items-center text-cyan-300/70 hover:bg-cyan-400/10 hover:text-cyan-100 disabled:cursor-not-allowed disabled:opacity-40"
          onClick={controller.clear}
          disabled={controller.isStreaming}
          aria-label={t('agent.newConversation')}
          title={t('agent.newConversation')}
        >
          <Plus className="size-4" />
        </button>
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
            className={cn('flex items-start gap-2', message.role === 'user' && 'flex-row-reverse')}
          >
            <div
              className={cn(
                'grid size-8 shrink-0 place-items-center border leading-none [&_svg]:block',
                message.role === 'user'
                  ? 'border-cyan-300/50 bg-cyan-300/15 text-cyan-100'
                  : 'border-cyan-500/30 bg-cyan-500/10 text-cyan-300',
                message.role === 'assistant' &&
                  message.status === 'streaming' &&
                  'animate-pulse border-cyan-200/60 shadow-[0_0_14px_rgba(103,232,249,0.22)] motion-reduce:animate-none'
              )}
            >
              {message.role === 'user' ? <User className="size-4" /> : <Bot className="size-4" />}
            </div>
            <div
              className={cn(
                'max-w-[292px] border px-3 py-2 font-mono text-xs leading-5',
                message.role === 'user'
                  ? 'whitespace-pre-wrap border-cyan-300/40 bg-cyan-300/10 text-cyan-50'
                  : 'border-cyan-500/25 bg-cyan-500/[0.04] text-cyan-100/80'
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
