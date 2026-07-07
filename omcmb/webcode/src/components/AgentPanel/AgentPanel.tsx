import { useMemo, useState } from 'react';
import { useLocation } from 'react-router-dom';
import {
  CloseOutlined,
  RobotOutlined,
  SendOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import {
  extractAgentRows,
  formatAgentValue,
  type AgentPanelActivity,
} from '@core/agentkit';
import { useAgentPanelController } from '@core/hooks/useAgentPanelController';
import { useT } from '@/hooks/useT';
import styles from './AgentPanel.module.css';

interface AgentPanelProps {
  open: boolean;
  onClose: () => void;
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
  const input = activity.request?.input ?? {};
  const inputRows = Object.entries(input);
  const resultRows = extractAgentRows(activity.output ?? activity.preview, 6);
  const isError = activity.status === 'error';

  return (
    <section className={styles.activity}>
      <div className={styles.activityHeader}>
        <span className={styles.activityTitle}>
          <ThunderboltOutlined /> {activity.status === 'preview' ? t('agent.actionPreview') : t('agent.callingTool')}
        </span>
        <span className={styles.readOnly}>{t('agent.readOnly')}</span>
      </div>
      <div className={styles.metaRow}>
        <span>{t('agent.tool')}</span>
        <strong>{activity.request?.actionId || activity.title}</strong>
      </div>
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
      <div className={styles.policy}>{t('agent.readOnlyPolicy')}</div>
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
  const controller = useAgentPanelController({ context });

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
          {controller.enabled ? t('agent.connected') : t('agent.disconnected')}
        </span>
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
            <div className={styles.avatar}>{message.role === 'user' ? t('agent.userShort') : <RobotOutlined />}</div>
            <div className={styles.bubble}>
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
