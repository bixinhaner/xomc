import { useEffect, useMemo, useState } from 'react';
import { useLocation } from 'react-router-dom';
import {
  CheckOutlined,
  CloseOutlined,
  CopyOutlined,
  FieldTimeOutlined,
  InfoCircleOutlined,
  PlusOutlined,
  RobotOutlined,
  SendOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import {
  AgentMarkdown,
  extractAgentRows,
  extractAgentRestInput,
  formatAgentDetail,
  formatAgentValue,
  summarizeAgentProcess,
  type AgentPanelActivity,
  type AgentPanelMessage,
  type AgentProcessEntry,
  type AgentThoughtEntry,
} from '@core/agentkit';
import { useAgentPanelController } from '@core/hooks/useAgentPanelController';
import { useT, type TranslateFn } from '@/hooks/useT';
import styles from './AgentPanel.module.css';

interface AgentPanelProps {
  open: boolean;
  onClose: () => void;
}

function ThoughtBlock({
  thoughts,
  running,
}: {
  thoughts: AgentThoughtEntry[] | undefined;
  running: boolean;
}) {
  const t = useT();
  if (!thoughts?.length) return null;
  const isStreaming = running || thoughts.some((thought) => thought.status === 'streaming');
  const lineCount = thoughts.reduce((total, thought) => total + Math.max(thought.lines.length, 1), 0);

  return (
    <details className={`${styles.thought} ${isStreaming ? styles.thoughtActive : ''}`} open={isStreaming}>
      <summary className={styles.thoughtSummary}>
        <span className={styles.statusPulse} aria-hidden="true" />
        <span>{isStreaming ? t('agent.thinking') : t('agent.thoughtDone')}</span>
        {!isStreaming && <span className={styles.thoughtCount}>{lineCount}</span>}
      </summary>
      {isStreaming && <div className={styles.thoughtActivityLine} aria-hidden="true" />}
      <div className={styles.thoughtList}>
        {thoughts.map((thought) => (
          <div key={thought.id} className={styles.thoughtEntry}>
            {(thought.lines.length ? thought.lines : [thought.text]).map((line, index) => (
              <p key={`${thought.id}-${index}`}>{line}</p>
            ))}
          </div>
        ))}
      </div>
    </details>
  );
}

function shortId(value: string | undefined): string {
  if (!value) return '';
  return value.length > 14 ? `${value.slice(0, 8)}...${value.slice(-4)}` : value;
}

function formatElapsed(ms: number, t: TranslateFn): string {
  const seconds = Math.max(0, Math.floor(ms / 1000));
  if (seconds < 60) return t('agent.elapsedSeconds', { seconds });
  const minutes = Math.floor(seconds / 60);
  return t('agent.elapsedMinutes', { minutes, seconds: seconds % 60 });
}

function useStreamingClock(enabled: boolean) {
  const [tick, setTick] = useState(() => Date.now());
  useEffect(() => {
    if (!enabled) return undefined;
    const timer = window.setInterval(() => setTick(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [enabled]);
  return tick;
}

function latestAssistantMessage(messages: AgentPanelMessage[]): AgentPanelMessage | undefined {
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const message = messages[index];
    if (message?.role === 'assistant') return message;
  }
  return undefined;
}

function streamingStatusKey(message: AgentPanelMessage | undefined): string {
  if (!message || message.status !== 'streaming') return '';
  const lastProcess = message.process?.[message.process.length - 1];
  const rest = extractAgentRestInput(lastProcess?.detail);
  if (lastProcess?.kind === 'tool_call' && rest) {
    if (rest.path === '/api/v1/agent/catalog' || rest.path === '/api/v1/agent/catalog/describe') {
      return 'agent.statusSearching';
    }
    return 'agent.statusCalling';
  }
  if (lastProcess?.kind === 'process') {
    const title = lastProcess.title.toLowerCase();
    if (title.includes('running')) return 'agent.statusCalling';
    if (title.includes('complete')) return 'agent.statusComposing';
    return 'agent.statusThinking';
  }
  if (lastProcess?.kind === 'tool_result' || lastProcess?.kind === 'done') {
    return 'agent.statusComposing';
  }
  if (message.text) return 'agent.statusComposing';
  if (message.thoughts?.length) return 'agent.statusThinking';
  return 'agent.statusConnecting';
}

function RuntimeStatusBar({
  message,
  nowTick,
  copiedId,
  onCopy,
  diagnosticOpen,
  onToggleDiagnostic,
  onCloseDiagnostic,
}: {
  message: AgentPanelMessage | undefined;
  nowTick: number;
  copiedId: string | null;
  onCopy: (value: string) => void;
  diagnosticOpen: boolean;
  onToggleDiagnostic: () => void;
  onCloseDiagnostic: () => void;
}) {
  const t = useT();
  if (!message || message.status !== 'streaming') return null;
  const statusKey = streamingStatusKey(message);
  const hasDiagnostics = Boolean(message.runId || message.conversationId);
  const diagnostics = JSON.stringify(
    {
      runId: message.runId,
      conversationId: message.conversationId,
      status: message.status,
      createdAt: message.createdAt,
      processCount: message.process?.length ?? 0,
    },
    null,
    2
  );

  return (
    <div className={styles.runtimeStatus}>
      <span className={styles.runtimePulse} aria-hidden="true" />
      <span className={styles.runtimeText}>
        <FieldTimeOutlined />
        {t(statusKey || 'agent.statusConnecting')} · {formatElapsed(nowTick - message.createdAt, t)}
      </span>
      {hasDiagnostics && (
        <>
          <button
            type="button"
            className={styles.diagnosticBtn}
            onClick={onToggleDiagnostic}
            aria-label={t('agent.diagnostics')}
            title={t('agent.diagnostics')}
          >
            <InfoCircleOutlined />
          </button>
          {diagnosticOpen && (
            <>
              <button
                type="button"
                className={styles.diagnosticBackdrop}
                aria-label={t('agent.close')}
                onClick={onCloseDiagnostic}
              />
              <div className={styles.diagnosticMenu}>
                <div className={styles.diagnosticHeader}>{t('agent.diagnostics')}</div>
                {message.runId && (
                  <div className={styles.diagnosticRow}>
                    <span>{t('agent.runId')}</span>
                    <code title={message.runId}>{shortId(message.runId)}</code>
                    <button
                      type="button"
                      className={styles.diagnosticCopy}
                      onClick={() => onCopy(message.runId as string)}
                      aria-label={t('agent.copy')}
                    >
                      {copiedId === message.runId ? <CheckOutlined /> : <CopyOutlined />}
                    </button>
                  </div>
                )}
                {message.conversationId && (
                  <div className={styles.diagnosticRow}>
                    <span>{t('agent.conversationId')}</span>
                    <code title={message.conversationId}>{shortId(message.conversationId)}</code>
                    <button
                      type="button"
                      className={styles.diagnosticCopy}
                      onClick={() => onCopy(message.conversationId as string)}
                      aria-label={t('agent.copy')}
                    >
                      {copiedId === message.conversationId ? <CheckOutlined /> : <CopyOutlined />}
                    </button>
                  </div>
                )}
                <button type="button" className={styles.diagnosticPrimary} onClick={() => onCopy(diagnostics)}>
                  {t('agent.copyDiagnostics')}
                </button>
              </div>
            </>
          )}
        </>
      )}
    </div>
  );
}

function ProcessTrace({
  entries,
  copiedId,
  onCopy,
}: {
  entries: AgentProcessEntry[] | undefined;
  copiedId: string | null;
  onCopy: (value: string) => void;
}) {
  const t = useT();
  if (!entries?.length) return null;
  const summary = summarizeAgentProcess(entries);
  const hasError = entries.some((entry) => entry.kind === 'error');

  return (
    <details className={styles.process} open={hasError}>
      <summary className={styles.processSummary}>
        <span className={styles.processSummaryTitle}>{t('agent.processTrace')}</span>
        <span className={styles.processCount}>{entries.length}</span>
        <span className={styles.processChips}>
          <span>{t('agent.processSearches')} {summary.searches}</span>
          <span>{t('agent.processCalls')} {summary.calls}</span>
          <span>{t('agent.processReadOnly')} {summary.readOnly}</span>
          {summary.errors > 0 && <span className={styles.processErrorChip}>{t('agent.processErrors')} {summary.errors}</span>}
        </span>
      </summary>
      <div className={styles.processList}>
        {entries.map((entry, index) => {
          const detail = formatAgentDetail(entry.detail);
          const isActive = index === entries.length - 1 && entry.kind !== 'error';
          const rest = extractAgentRestInput(entry.detail);
          const copyValue = rest ? `${rest.method} ${rest.path}` : entry.id;
          return (
            <div
              key={entry.id}
              className={`${styles.processItem} ${isActive ? styles.processItemActive : ''} ${
                entry.kind === 'error' ? styles.processItemError : ''
              }`}
            >
              <span className={styles.processDot} aria-hidden="true" />
              <span className={`${styles.processKind} ${entry.kind === 'error' ? styles.processKindError : ''}`}>
                {entry.kind}
              </span>
              <div className={styles.processBody}>
                <div className={styles.processTitleRow}>
                  <div className={styles.processTitle}>
                    {rest ? (
                      <>
                        <span className={`${styles.methodPill} ${rest.method === 'GET' ? styles.methodGet : styles.methodWrite}`}>
                          {rest.method}
                        </span>
                        <span className={styles.processPath}>{rest.path}</span>
                      </>
                    ) : (
                      entry.title
                    )}
                  </div>
                  <button
                    type="button"
                    className={styles.inlineCopy}
                    onClick={() => onCopy(copyValue)}
                    title={copyValue}
                    aria-label={t('common.copy')}
                  >
                    {copiedId === copyValue ? <CheckOutlined /> : <CopyOutlined />}
                  </button>
                </div>
                {rest?.operationId && <div className={styles.processSubtle}>operationId: {rest.operationId}</div>}
                {detail && <pre className={styles.processDetail}>{detail}</pre>}
              </div>
            </div>
          );
        })}
      </div>
    </details>
  );
}

function AssistantContent({
  message,
  copiedId,
  onCopy,
}: {
  message: AgentPanelMessage;
  copiedId: string | null;
  onCopy: (value: string) => void;
}) {
  const t = useT();
  const running = message.status === 'streaming';
  return (
    <div className={styles.assistantContent}>
      <ThoughtBlock thoughts={message.thoughts} running={running} />
      {message.text ? (
        <div className={running ? styles.streamingAnswer : undefined}>
          <AgentMarkdown className={styles.markdown} content={message.text} />
        </div>
      ) : running ? (
        <span className={styles.pendingAnswer}>
          {t('agent.streaming')}
          <span className={styles.answerCaret} aria-hidden="true" />
        </span>
      ) : null}
      <ProcessTrace entries={message.process} copiedId={copiedId} onCopy={onCopy} />
    </div>
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function restInput(input: unknown) {
  return extractAgentRestInput(input);
}

function ActivityCard({
  activity,
  isPending,
  isStreaming,
  onExecute,
  onCancel,
}: {
  activity: AgentPanelActivity;
  isPending: boolean;
  isStreaming: boolean;
  onExecute: () => void;
  onCancel: () => void;
}) {
  const t = useT();
  const rawInput = activity.request?.input ?? activity.input ?? {};
  const rest = restInput(rawInput);
  const inputRows = Object.entries(rest?.query ?? (isRecord(rawInput) ? rawInput : {}));
  const resultRows = extractAgentRows(activity.output ?? activity.preview, 6);
  const isError = activity.status === 'error';
  const isActive = activity.status === 'calling' || activity.status === 'running' || isPending;
  const isRead = rest?.method === 'GET' || activity.risk === 'read';

  return (
    <section className={`${styles.activity} ${isActive ? styles.activityActive : ''}`}>
      <div className={styles.activityHeader}>
        <span className={styles.activityTitle}>
          <ThunderboltOutlined /> {rest ? rest.path : activity.status === 'preview' ? t('agent.actionPreview') : t('agent.callingTool')}
        </span>
        <span className={isRead ? styles.readOnly : styles.writeMode}>
          {isRead ? t('agent.readOnly') : t('agent.writeMode')}
        </span>
      </div>
      {rest && (
        <div className={styles.restMeta}>
          <span className={`${styles.methodPill} ${rest.method === 'GET' ? styles.methodGet : styles.methodWrite}`}>
            {rest.method}
          </span>
          <span>{t('agent.operationId')}: {rest.operationId || '-'}</span>
        </div>
      )}
      <div className={styles.metaRow}>
        <span>{t('agent.tool')}</span>
        <strong>{activity.request?.actionId || activity.title}</strong>
      </div>
      {rest?.reason && <p className={styles.summary}>{rest.reason}</p>}
      {activity.summary && <p className={styles.summary}>{activity.summary}</p>}
      {inputRows.length > 0 && (
        <div className={styles.tableBlock}>
          <div className={styles.tableTitle}>{t('agent.parameters')}</div>
          <table>
            <tbody>
              {inputRows.map(([key, value]) => (
                <tr key={key}>
                  <td>{key}</td>
                  <td>{formatAgentValue(value)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <div className={styles.policy}>{isRead ? t('agent.readOnlyPolicy') : t('agent.writePolicy')}</div>
      {isPending && (
        <div className={styles.actions}>
          <button type="button" className={styles.primaryBtn} disabled={isStreaming} onClick={onExecute}>
            {t('agent.execute')}
          </button>
          <button type="button" className={styles.secondaryBtn} disabled={isStreaming} onClick={onCancel}>
            {t('agent.cancel')}
          </button>
        </div>
      )}
      {!isPending && activity.status !== 'calling' && (
        <div className={styles.tableBlock}>
          <div className={styles.tableTitle}>
            {isError ? t('agent.failed') : t('agent.previewResult')}
          </div>
          <table>
            <tbody>
              {isError ? (
                <tr>
                  <td>{activity.error?.code ?? 'error'}</td>
                  <td>{activity.error?.message ?? t('agent.failed')}</td>
                </tr>
              ) : resultRows.length > 0 ? (
                resultRows.map((row) => (
                  <tr key={row.key}>
                    <td>{row.key}</td>
                    <td>{row.value}</td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td colSpan={2}>{t('agent.noRows')}</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function shouldRenderActivity(activity: AgentPanelActivity, pendingCallId: string | undefined, isStreaming: boolean): boolean {
  if (activity.callId === pendingCallId) return true;
  if (activity.status === 'error') return true;
  if (!isStreaming) return false;
  if (activity.risk === 'read') return false;
  const rest = extractAgentRestInput(activity.request?.input ?? activity.input);
  if (rest?.method === 'GET') return false;
  return activity.status === 'preview' || activity.status === 'calling' || activity.status === 'running';
}

export function AgentPanel({ open, onClose }: AgentPanelProps) {
  const t = useT();
  const location = useLocation();
  const [input, setInput] = useState('');
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [diagnosticOpen, setDiagnosticOpen] = useState(false);
  const context = useMemo(
    () => ({
      path: location.pathname,
      query: Object.fromEntries(new URLSearchParams(location.search).entries()),
    }),
    [location.pathname, location.search]
  );
  const controller = useAgentPanelController({ context, active: open });
  const activeAssistant = latestAssistantMessage(controller.messages);
  const nowTick = useStreamingClock(controller.isStreaming);

  const copyText = async (value: string) => {
    try {
      await navigator.clipboard.writeText(value);
      setCopiedId(value);
      window.setTimeout(() => setCopiedId((current) => (current === value ? null : current)), 1200);
    } catch {
      setCopiedId(null);
    }
  };

  if (!open) return null;

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    const text = input.trim();
    if (!text) return;
    setInput('');
    await controller.sendMessage(text);
  };

  return (
    <aside className={styles.panel} aria-label={t('agent.title')}>
      <header className={styles.panelHeader}>
        <div className={styles.heading}>
          <RobotOutlined />
          <span>{t('agent.title')}</span>
        </div>
        <span className={controller.enabled ? styles.connected : styles.disconnected}>
          {controller.enabled && <span className={styles.liveDot} aria-hidden="true" />}
          {controller.enabled ? t('agent.connected') : t('agent.disconnected')}
        </span>
        <button
          type="button"
          className={styles.iconBtn}
          onClick={controller.clear}
          disabled={controller.isStreaming}
          aria-label={t('agent.newConversation')}
          title={t('agent.newConversation')}
        >
          <PlusOutlined />
        </button>
        <button type="button" className={styles.iconBtn} onClick={onClose} aria-label={t('agent.close')}>
          <CloseOutlined />
        </button>
      </header>
      <RuntimeStatusBar
        message={activeAssistant}
        nowTick={nowTick}
        copiedId={copiedId}
        onCopy={copyText}
        diagnosticOpen={diagnosticOpen}
        onToggleDiagnostic={() => setDiagnosticOpen((current) => !current)}
        onCloseDiagnostic={() => setDiagnosticOpen(false)}
      />

      <div className={styles.body}>
        {!controller.enabled && (
          <div className={styles.empty}>
            <strong>{t('agent.disabledTitle')}</strong>
            <span>{t('agent.disabledHint')}</span>
          </div>
        )}
        {controller.enabled && controller.messages.length === 0 && controller.activities.length === 0 && (
          <div className={styles.empty}>
            <strong>{t('agent.emptyTitle')}</strong>
            <span>{t('agent.emptyHint')}</span>
          </div>
        )}
        {controller.messages.map((message) => (
          <div key={message.id} className={message.role === 'user' ? styles.userMessage : styles.assistantMessage}>
            <div
              className={`${styles.avatar} ${
                message.role === 'assistant' && message.status === 'streaming' ? styles.avatarActive : ''
              }`}
            >
              {message.role === 'user' ? t('agent.userShort') : <RobotOutlined />}
            </div>
            <div className={styles.bubble}>
              {message.role === 'assistant' ? (
                <AssistantContent message={message} copiedId={copiedId} onCopy={copyText} />
              ) : (
                message.text || (message.status === 'streaming' ? t('agent.streaming') : '')
              )}
            </div>
          </div>
        ))}
        {controller.activities
          .filter((activity) => shouldRenderActivity(activity, controller.pendingAction?.callId, controller.isStreaming))
          .map((activity) => (
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
          <div className={styles.error}>
            {t('agent.errorPrefix')}: {controller.error.message}
          </div>
        )}
      </div>

      <form className={styles.composer} onSubmit={submit}>
        <input
          value={input}
          onChange={(event) => setInput(event.target.value)}
          placeholder={t('agent.placeholder')}
          disabled={!controller.enabled || controller.isStreaming}
        />
        <button
          type="submit"
          className={styles.sendBtn}
          disabled={!controller.enabled || controller.isStreaming || input.trim() === ''}
          aria-label={t('agent.send')}
        >
          <SendOutlined />
        </button>
      </form>
    </aside>
  );
}
