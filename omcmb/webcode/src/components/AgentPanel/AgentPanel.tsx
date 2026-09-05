import { useEffect, useMemo, useRef, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { Modal } from 'antd';
import {
  ArrowDownOutlined,
  BellOutlined,
  ArrowsAltOutlined,
  CheckOutlined,
  CloseOutlined,
  CopyOutlined,
  FieldTimeOutlined,
  FileExcelOutlined,
  FileImageOutlined,
  FileOutlined,
  FilePdfOutlined,
  FileTextOutlined,
  DeleteOutlined,
  DownloadOutlined,
  EyeOutlined,
  InfoCircleOutlined,
  PlusOutlined,
  PaperClipOutlined,
  QuestionCircleOutlined,
  RobotOutlined,
  SendOutlined,
  StopOutlined,
  ShrinkOutlined,
  ThunderboltOutlined,
  ToolOutlined,
  WifiOutlined,
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
  type AgentAttachmentRef,
  type AgentArtifactRef,
} from '@core/agentkit';
import { agentApi } from '@core/services/api/agentApi';
import { useAgentAutoScroll } from '@core/hooks/useAgentAutoScroll';
import { useAgentPanelController } from '@core/hooks/useAgentPanelController';
import { useAgentPanelLayout } from '@core/hooks/useAgentPanelLayout';
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
  onPreviewArtifact,
  onDownloadArtifact,
}: {
  message: AgentPanelMessage;
  copiedId: string | null;
  onCopy: (value: string) => void;
  onPreviewArtifact: (artifact: AgentArtifactRef) => void;
  onDownloadArtifact: (artifact: AgentArtifactRef) => void;
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
      ) : message.status === 'cancelled' ? (
        <span className={styles.pendingAnswer}>{t('agent.stopped')}</span>
      ) : null}
      <ArtifactRows
        artifacts={message.artifacts}
        onPreview={onPreviewArtifact}
        onDownload={onDownloadArtifact}
      />
      <ProcessTrace entries={message.process} copiedId={copiedId} onCopy={onCopy} />
    </div>
  );
}

function formatFileSize(size: number | null | undefined): string {
  if (!size || size < 1024) return `${size ?? 0} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function FileTypeIcon({ filename, mimeType }: { filename: string; mimeType?: string | null }) {
  const value = `${mimeType ?? ''} ${filename}`.toLowerCase();
  if (value.includes('pdf')) return <FilePdfOutlined />;
  if (value.includes('image')) return <FileImageOutlined />;
  if (value.includes('spreadsheet') || /\.(xlsx?|csv)$/.test(value)) return <FileExcelOutlined />;
  if (value.includes('text') || /\.(md|txt|json|ya?ml|log)$/.test(value)) return <FileTextOutlined />;
  return <FileOutlined />;
}

function AttachmentRows({
  attachments,
  onRemove,
}: {
  attachments: AgentAttachmentRef[] | undefined;
  onRemove?: (attachmentId: string) => void;
}) {
  if (!attachments?.length) return null;
  return (
    <div className={styles.fileRows}>
      {attachments.map((file) => (
        <div key={file.attachmentId} className={styles.fileRow}>
          <span className={styles.fileIcon}><FileTypeIcon filename={file.filename} mimeType={file.mimeType} /></span>
          <span className={styles.fileMeta}>
            <strong title={file.filename}>{file.filename}</strong>
            <small>{formatFileSize(file.sizeBytes)}</small>
          </span>
          {onRemove && (
            <button type="button" className={styles.fileAction} onClick={() => onRemove(file.attachmentId)} aria-label={file.filename}>
              <DeleteOutlined />
            </button>
          )}
        </div>
      ))}
    </div>
  );
}

function ArtifactRows({
  artifacts,
  onPreview,
  onDownload,
}: {
  artifacts: AgentArtifactRef[] | undefined;
  onPreview: (artifact: AgentArtifactRef) => void;
  onDownload: (artifact: AgentArtifactRef) => void;
}) {
  const t = useT();
  if (!artifacts?.length) return null;
  return (
    <div className={styles.artifacts}>
      <div className={styles.artifactHeading}>{t('agent.generatedFiles')}</div>
      {artifacts.map((artifact) => (
        <div key={artifact.artifactId} className={styles.fileRow}>
          <span className={styles.fileIcon}><FileTypeIcon filename={artifact.filename} mimeType={artifact.mimeType} /></span>
          <span className={styles.fileMeta}>
            <strong title={artifact.filename}>{artifact.filename}</strong>
            <small>{formatFileSize(artifact.sizeBytes)}</small>
          </span>
          <span className={styles.fileActions}>
            <button type="button" className={styles.fileAction} onClick={() => onPreview(artifact)} title={t('agent.previewFile')}>
              <EyeOutlined />
            </button>
            <button type="button" className={styles.fileAction} onClick={() => onDownload(artifact)} title={t('agent.downloadFile')}>
              <DownloadOutlined />
            </button>
          </span>
        </div>
      ))}
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

function EmptyPromptState({ onPrompt }: { onPrompt: (prompt: string) => void }) {
  const t = useT();
  const prompts = [
    { text: t('agent.suggestionNetworkHealth'), icon: <QuestionCircleOutlined /> },
    { text: t('agent.suggestionOnlineDevices'), icon: <WifiOutlined /> },
    { text: t('agent.suggestionAlarms'), icon: <BellOutlined /> },
    { text: t('agent.suggestionConfigChange'), icon: <ToolOutlined /> },
  ];

  return (
    <div className={styles.empty}>
      <div className={styles.emptyBot} aria-hidden="true">
        <span className={styles.emptyBotOrbit} />
        <span className={styles.emptyBotHead}>
          <span className={styles.emptyBotFace}>
            <span />
            <span />
          </span>
        </span>
        <span className={styles.emptyBotDotOne} />
        <span className={styles.emptyBotDotTwo} />
        <span className={styles.emptyBotDotThree} />
      </div>
      <strong>{t('agent.emptyTitle')}</strong>
      <span className={styles.emptyHint}>{t('agent.emptyHint')}</span>
      <div className={styles.emptyPrompts}>
        {prompts.map((prompt) => (
          <button key={prompt.text} type="button" onClick={() => onPrompt(prompt.text)}>
            {prompt.icon}
            <span className={styles.promptText}>{prompt.text}</span>
          </button>
        ))}
      </div>
    </div>
  );
}

export function AgentPanel({ open, onClose }: AgentPanelProps) {
  const t = useT();
  const location = useLocation();
  const [input, setInput] = useState('');
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [diagnosticOpen, setDiagnosticOpen] = useState(false);
  const [newConversationOpen, setNewConversationOpen] = useState(false);
  const [newConversationLoading, setNewConversationLoading] = useState(false);
  const [launchFindingId, setLaunchFindingId] = useState<string | undefined>();
  const fileInputRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    const receiveFinding = (event: Event) => {
      const detail = (event as CustomEvent<{ context?: { agentFindingId?: string }; message?: string }>).detail;
      if (!detail?.context?.agentFindingId) return;
      setLaunchFindingId(detail.context.agentFindingId);
      if (detail.message) setInput(detail.message);
    };
    window.addEventListener('xomc:open-agent-finding', receiveFinding);
    return () => window.removeEventListener('xomc:open-agent-finding', receiveFinding);
  }, []);
  const context = useMemo(
    () => ({
      path: location.pathname,
      query: Object.fromEntries(new URLSearchParams(location.search).entries()),
      ...(launchFindingId ? { extra: { agentFindingId: launchFindingId } } : {}),
    }),
    [launchFindingId, location.pathname, location.search]
  );
  const controller = useAgentPanelController({ context, active: open });
  const errorMessage = controller.error
    ? ({
        ATTACHMENT_EMPTY: t('agent.attachmentEmpty'),
        ATTACHMENT_TOO_LARGE: t('agent.attachmentTooLarge'),
        ATTACHMENT_LIMIT_EXCEEDED: t('agent.attachmentLimit'),
      } as Record<string, string>)[controller.error.code] ?? controller.error.message
    : '';
  const activeAssistant = latestAssistantMessage(controller.messages);
  const nowTick = useStreamingClock(controller.isStreaming);
  const panelLayout = useAgentPanelLayout();
  const {
    scrollContainerRef,
    scrollContentRef,
    showJumpToLatest,
    handleScroll,
    scrollToLatest,
  } = useAgentAutoScroll({ active: open });

  const copyText = async (value: string) => {
    try {
      await navigator.clipboard.writeText(value);
      setCopiedId(value);
      window.setTimeout(() => setCopiedId((current) => (current === value ? null : current)), 1200);
    } catch {
      setCopiedId(null);
    }
  };

  const previewArtifact = async (artifact: AgentArtifactRef) => {
    const preview = window.open('', '_blank');
    try {
      const blob = await agentApi.getArtifactContent(artifact, 'inline');
      const objectUrl = URL.createObjectURL(blob);
      if (preview) preview.location.href = objectUrl;
      else window.open(objectUrl, '_blank');
      window.setTimeout(() => URL.revokeObjectURL(objectUrl), 60000);
    } catch {
      preview?.close();
    }
  };

  const downloadArtifact = async (artifact: AgentArtifactRef) => {
    const blob = await agentApi.getArtifactContent(artifact, 'attachment');
    const objectUrl = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = objectUrl;
    link.download = artifact.filename;
    link.click();
    URL.revokeObjectURL(objectUrl);
  };

  if (!open) return null;

  const handlePrompt = async (prompt: string) => {
    if (!controller.enabled || controller.isStreaming) return;
    const request = controller.sendMessage(prompt);
    scrollToLatest('auto');
    await request;
  };

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!controller.enabled || controller.isStreaming) return;
    const text = input.trim();
    if (!text) return;
    setInput('');
    const request = controller.sendMessage(text);
    scrollToLatest('auto');
    await request;
  };

  const handleComposerKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key !== 'Enter' || event.shiftKey || event.nativeEvent.isComposing) return;
    event.preventDefault();
    if (controller.isStreaming) return;
    event.currentTarget.form?.requestSubmit();
  };

  const startNewConversation = async () => {
    setNewConversationLoading(true);
    try {
      await controller.clear();
      setNewConversationOpen(false);
    } finally {
      setNewConversationLoading(false);
    }
  };

  return (
    <aside
      className={`${styles.panel} ${panelLayout.isResizing ? styles.panelResizing : ''}`}
      style={panelLayout.panelStyle}
      aria-label={t('agent.title')}
    >
      <div
        className={styles.resizeHandle}
        aria-label={t('agent.resizePanel')}
        title={t('agent.resizePanel')}
        {...panelLayout.resizeHandleProps}
      >
        <span className={styles.resizeGrip} aria-hidden="true" />
      </div>
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
          onClick={panelLayout.toggleExpanded}
          aria-label={panelLayout.isExpanded ? t('agent.restorePanel') : t('agent.expandPanel')}
          title={panelLayout.isExpanded ? t('agent.restorePanel') : t('agent.expandPanel')}
        >
          {panelLayout.isExpanded ? <ShrinkOutlined /> : <ArrowsAltOutlined />}
        </button>
        <button
          type="button"
          className={styles.iconBtn}
          onClick={() => setNewConversationOpen(true)}
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

      <div className={styles.bodyShell}>
        <div ref={scrollContainerRef} className={styles.body} onScroll={handleScroll}>
          <div ref={scrollContentRef} className={styles.messageList}>
            {!controller.enabled && (
              <div className={styles.disabledEmpty}>
                <strong>{t('agent.disabledTitle')}</strong>
                <span>{t('agent.disabledHint')}</span>
              </div>
            )}
            {controller.enabled && controller.messages.length === 0 && controller.activities.length === 0 && (
              <EmptyPromptState onPrompt={handlePrompt} />
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
                    <AssistantContent
                      message={message}
                      copiedId={copiedId}
                      onCopy={copyText}
                      onPreviewArtifact={previewArtifact}
                      onDownloadArtifact={downloadArtifact}
                    />
                  ) : (
                    <div className={styles.userContent}>
                      <span>{message.text || (message.status === 'streaming' ? t('agent.streaming') : '')}</span>
                      <AttachmentRows attachments={message.attachments} />
                    </div>
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
                {t('agent.errorPrefix')}: {errorMessage}
              </div>
            )}
          </div>
        </div>
        {showJumpToLatest && (
          <button
            type="button"
            className={styles.jumpToLatest}
            onClick={() => scrollToLatest()}
            aria-label={t('agent.jumpToLatest')}
            title={t('agent.jumpToLatest')}
          >
            <ArrowDownOutlined />
          </button>
        )}
      </div>

      <form className={styles.composer} onSubmit={submit}>
        <AttachmentRows attachments={controller.attachments} onRemove={controller.removeAttachment} />
        <div className={styles.composerRow}>
          <input
            ref={fileInputRef}
            className={styles.hiddenFileInput}
            type="file"
            multiple
            onChange={(event) => {
              void controller.uploadAttachments(Array.from(event.target.files ?? []));
              event.currentTarget.value = '';
            }}
          />
          <button
            type="button"
            className={styles.attachBtn}
            onClick={() => fileInputRef.current?.click()}
            disabled={!controller.enabled || controller.isStreaming || controller.isUploading || controller.attachments.length >= 10}
            aria-label={t('agent.attachFile')}
            title={t('agent.attachFile')}
          >
            <PaperClipOutlined />
          </button>
          <textarea
            rows={1}
            value={input}
            onChange={(event) => setInput(event.target.value)}
            onKeyDown={handleComposerKeyDown}
            placeholder={controller.isUploading ? t('agent.uploadingFile') : t('agent.placeholder')}
            disabled={!controller.enabled}
          />
        </div>
        <button
          type={controller.isStreaming ? 'button' : 'submit'}
          className={`${styles.sendBtn} ${controller.isStreaming ? styles.stopBtn : ''}`}
          onClick={controller.isStreaming ? () => void controller.stop() : undefined}
          disabled={!controller.enabled || controller.isUploading || (!controller.isStreaming && input.trim() === '')}
          aria-label={controller.isStreaming ? t('agent.stop') : t('agent.send')}
        >
          {controller.isStreaming ? <StopOutlined /> : <SendOutlined />}
        </button>
      </form>
      <Modal
        open={newConversationOpen}
        title={t('agent.newConversationConfirmTitle')}
        okText={t('agent.newConversation')}
        cancelText={t('agent.cancel')}
        confirmLoading={newConversationLoading}
        centered
        width={420}
        onOk={() => void startNewConversation()}
        onCancel={() => setNewConversationOpen(false)}
      >
        <p>{t('agent.newConversationConfirmDescription')}</p>
      </Modal>
    </aside>
  );
}
