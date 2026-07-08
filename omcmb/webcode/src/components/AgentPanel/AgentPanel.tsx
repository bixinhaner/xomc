import { useMemo, useState } from 'react';
import { useLocation } from 'react-router-dom';
import {
  CloseOutlined,
  PlusOutlined,
  RobotOutlined,
  SendOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import {
  AgentMarkdown,
  extractAgentRows,
  formatAgentDetail,
  formatAgentValue,
  type AgentPanelActivity,
  type AgentPanelMessage,
  type AgentProcessEntry,
  type AgentThoughtEntry,
} from '@core/agentkit';
import { useAgentPanelController } from '@core/hooks/useAgentPanelController';
import { useT } from '@/hooks/useT';
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

function ProcessTrace({ entries }: { entries: AgentProcessEntry[] | undefined }) {
  const t = useT();
  if (!entries?.length) return null;

  return (
    <details className={styles.process} open={entries.some((entry) => entry.kind === 'error')}>
      <summary className={styles.processSummary}>
        <span>{t('agent.processTrace')}</span>
        <span className={styles.processCount}>{entries.length}</span>
      </summary>
      <div className={styles.processList}>
        {entries.map((entry, index) => {
          const detail = formatAgentDetail(entry.detail);
          const isActive = index === entries.length - 1 && entry.kind !== 'error';
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
                <div className={styles.processTitle}>{entry.title}</div>
                {detail && <pre className={styles.processDetail}>{detail}</pre>}
              </div>
            </div>
          );
        })}
      </div>
    </details>
  );
}

function AssistantContent({ message }: { message: AgentPanelMessage }) {
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
      <ProcessTrace entries={message.process} />
    </div>
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function restInput(input: unknown) {
  if (!isRecord(input)) return null;
  const method = typeof input.method === 'string' ? input.method.toUpperCase() : '';
  const path = typeof input.path === 'string' ? input.path : '';
  if (!method || !path) return null;
  return {
    method,
    path,
    operationId: typeof input.operationId === 'string' ? input.operationId : '',
    query: isRecord(input.query) ? input.query : {},
    body: input.body,
    reason: typeof input.reason === 'string' ? input.reason : '',
  };
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

export function AgentPanel({ open, onClose }: AgentPanelProps) {
  const t = useT();
  const location = useLocation();
  const [input, setInput] = useState('');
  const context = useMemo(
    () => ({
      path: location.pathname,
      query: Object.fromEntries(new URLSearchParams(location.search).entries()),
    }),
    [location.pathname, location.search]
  );
  const controller = useAgentPanelController({ context, active: open });

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
