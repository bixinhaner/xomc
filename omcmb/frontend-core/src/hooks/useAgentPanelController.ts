import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  type AgentApprovedAction,
  type AgentAttachmentRef,
  type AgentError,
  type AgentPageContext,
  type AgentRuntimeMode,
  type AgentStreamEvent,
  parseApprovedAction,
  type AgentPanelActivity,
  type AgentPanelMessage,
  type AgentPendingAction,
  completeAgentThoughts,
  mergeAgentThought,
  upsertAgentProcess,
} from '../agentkit';
import { agentApi } from '../services/api/agentApi';
import { useAppStore } from '../store/appStore';
import { useUserStore } from '../store/userStore';
import { useAgentConversation, useStartAgentConversation } from './api/useAgentConfig';
import { useAgentRuntimeClient } from './useAgentRuntimeClient';

export interface UseAgentPanelControllerOptions {
  context: AgentPageContext;
  active?: boolean;
}

export interface UseAgentPanelControllerResult {
  enabled: boolean;
  messages: AgentPanelMessage[];
  activities: AgentPanelActivity[];
  pendingAction: AgentPendingAction | null;
  isStreaming: boolean;
  isUploading: boolean;
  attachments: AgentAttachmentRef[];
  error: AgentError | null;
  sendMessage: (message: string) => Promise<void>;
  executePendingAction: () => Promise<void>;
  cancelPendingAction: () => void;
  uploadAttachments: (files: File[]) => Promise<void>;
  removeAttachment: (attachmentId: string) => Promise<void>;
  stop: () => Promise<void>;
  clear: () => Promise<void>;
}

let nextAgentPanelId = 0;
const AGENT_CONVERSATION_STORAGE_PREFIX = 'omc-agent-conversation';
const AGENT_CONVERSATION_STATE_VERSION = 1;
const MAX_PERSISTED_MESSAGES = 40;
const MAX_PERSISTED_ACTIVITIES = 30;
const MAX_ATTACHMENT_COUNT = 10;
const MAX_ATTACHMENT_BYTES = 25 * 1024 * 1024;

interface PersistedAgentPanelState {
  version: typeof AGENT_CONVERSATION_STATE_VERSION;
  conversationId?: string;
  messages: AgentPanelMessage[];
  activities: AgentPanelActivity[];
  pendingAction: AgentPendingAction | null;
  updatedAt: number;
}

function createId(prefix: string): string {
  nextAgentPanelId += 1;
  return `${prefix}-${Date.now()}-${nextAgentPanelId}`;
}

function runtimeError(error: unknown): AgentError {
  if (
    error &&
    typeof error === 'object' &&
    'code' in error &&
    'message' in error &&
    typeof (error as { code?: unknown }).code === 'string' &&
    typeof (error as { message?: unknown }).message === 'string'
  ) {
    return error as AgentError;
  }
  return {
    code: 'AGENT_STREAM_FAILED',
    message: error instanceof Error ? error.message : 'Agent request failed.',
    retryable: true,
  };
}

function now() {
  return Date.now();
}

function buildConversationStorageKey(input: {
  connectorId?: string;
  userId?: string;
  conversationId?: string;
}): string | undefined {
  if (!input.connectorId || !input.userId || !input.conversationId) return undefined;
  return `${AGENT_CONVERSATION_STORAGE_PREFIX}:${input.connectorId}:${input.userId}:${input.conversationId}`;
}

function normalizeMessagesForStorage(messages: AgentPanelMessage[]): AgentPanelMessage[] {
  return messages.slice(-MAX_PERSISTED_MESSAGES).map((message) => ({
    ...message,
    status: message.status === 'streaming' ? 'error' : message.status,
    thoughts: completeAgentThoughts(message.thoughts),
  }));
}

function normalizeActivitiesForStorage(activities: AgentPanelActivity[]): AgentPanelActivity[] {
  return activities.slice(-MAX_PERSISTED_ACTIVITIES).map((activity) => ({
    ...activity,
    status:
      activity.status === 'calling' || activity.status === 'running'
        ? 'cancelled'
        : activity.status,
  }));
}

function readConversationState(storageKey: string | undefined): PersistedAgentPanelState | undefined {
  if (!storageKey || typeof window === 'undefined') return undefined;
  try {
    const raw = window.localStorage.getItem(storageKey);
    if (!raw) return undefined;
    if (!raw.trim().startsWith('{')) {
      return {
        version: AGENT_CONVERSATION_STATE_VERSION,
        conversationId: raw,
        messages: [],
        activities: [],
        pendingAction: null,
        updatedAt: 0,
      };
    }
    const parsed = JSON.parse(raw) as Partial<PersistedAgentPanelState>;
    if (parsed.version !== AGENT_CONVERSATION_STATE_VERSION) return undefined;
    return {
      version: AGENT_CONVERSATION_STATE_VERSION,
      conversationId: typeof parsed.conversationId === 'string' ? parsed.conversationId : undefined,
      messages: Array.isArray(parsed.messages) ? normalizeMessagesForStorage(parsed.messages) : [],
      activities: Array.isArray(parsed.activities) ? normalizeActivitiesForStorage(parsed.activities) : [],
      pendingAction: parsed.pendingAction ?? null,
      updatedAt: typeof parsed.updatedAt === 'number' ? parsed.updatedAt : 0,
    };
  } catch {
    return undefined;
  }
}

function writeConversationState(storageKey: string | undefined, state: PersistedAgentPanelState | undefined) {
  if (!storageKey || typeof window === 'undefined') return;
  try {
    if (state && (state.messages.length || state.activities.length || state.pendingAction)) {
      window.localStorage.setItem(storageKey, JSON.stringify(state));
    } else {
      window.localStorage.removeItem(storageKey);
    }
  } catch {
    // Ignore storage quota/privacy-mode failures; in-memory state still works.
  }
}

function upsertActivity(
  activities: AgentPanelActivity[],
  next: AgentPanelActivity
): AgentPanelActivity[] {
  const index = activities.findIndex((activity) => activity.callId === next.callId);
  if (index === -1) return [...activities, next];
  const copy = activities.slice();
  copy[index] = { ...copy[index], ...next };
  return copy;
}

function processId(prefix: string, sourceId?: string): string {
  return sourceId ? `${prefix}-${sourceId}` : createId(prefix);
}

export function useAgentPanelController(
  options: UseAgentPanelControllerOptions
): UseAgentPanelControllerResult {
  const locale = useAppStore((s) => s.locale);
  const systemTimezone = useAppStore((s) => s.systemTimezone);
  const currentUserId = useUserStore((s) => s.currentUser?.id);
  const { enabled, client, config } = useAgentRuntimeClient(undefined, {
    queryEnabled: options.active ?? true,
  });
  const {
    data: conversationData,
    refetch: refetchConversation,
  } = useAgentConversation(
    config.connectorId,
    currentUserId,
    Boolean(options.active ?? true) && enabled
  );
  const startConversation = useStartAgentConversation(config.connectorId, currentUserId);
  const [messages, setMessages] = useState<AgentPanelMessage[]>([]);
  const [activities, setActivities] = useState<AgentPanelActivity[]>([]);
  const [pendingAction, setPendingAction] = useState<AgentPendingAction | null>(null);
  const [conversationId, setConversationId] = useState<string | undefined>();
  const [isStreaming, setIsStreaming] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [attachments, setAttachments] = useState<AgentAttachmentRef[]>([]);
  const [error, setError] = useState<AgentError | null>(null);
  const [storageReady, setStorageReady] = useState(false);
  const actionRequestsRef = useRef(new Map<string, AgentApprovedAction>());
  const abortRef = useRef<AbortController | null>(null);
  const activeRunIdRef = useRef<string | undefined>(undefined);
  const historyKeyRef = useRef<string | undefined>(undefined);
  const skipNextPersistRef = useRef(false);
  const conversationRequestRef = useRef<Promise<string | undefined> | null>(null);

  const timezone = systemTimezone || 'UTC';
  const context = useMemo(() => options.context, [options.context]);
  const conversationStorageKey = useMemo(
    () =>
      buildConversationStorageKey({
        connectorId: config.connectorId,
        userId: currentUserId,
        conversationId,
      }),
    [config.connectorId, conversationId, currentUserId]
  );

  useEffect(() => {
    if (!conversationData?.conversationId) return;
    setConversationId(conversationData.conversationId);
  }, [conversationData?.conversationId]);

  useEffect(() => {
    if (!enabled || !config.connectorId || !currentUserId) {
      setConversationId(undefined);
    }
  }, [config.connectorId, currentUserId, enabled]);

  useEffect(() => {
    setStorageReady(false);
    const restored = readConversationState(conversationStorageKey);
    skipNextPersistRef.current = true;
    setMessages(restored?.messages ?? []);
    setActivities(restored?.activities ?? []);
    setPendingAction(restored?.pendingAction ?? null);
    setError(null);
    setIsStreaming(false);
    setAttachments([]);
    activeRunIdRef.current = undefined;
    actionRequestsRef.current.clear();
    setStorageReady(true);
  }, [conversationStorageKey]);

  useEffect(() => {
    if (!options.active || !enabled || !conversationId || isStreaming) return;
    const historyKey = `${config.connectorId}:${currentUserId}:${conversationId}`;
    if (historyKeyRef.current === historyKey) return;
    historyKeyRef.current = historyKey;
    void agentApi.getConversationMessages()
      .then((history) => {
        if (history.conversationId !== conversationId || history.messages.length === 0) return;
        setMessages(history.messages.map((message) => ({
          ...message,
          createdAt:
            typeof message.createdAt === 'number'
              ? message.createdAt
              : new Date(message.createdAt).getTime(),
          status: message.status === 'error' ? 'error' : 'done',
        })));
      })
      .catch(() => {
        historyKeyRef.current = undefined;
      });
  }, [config.connectorId, conversationId, currentUserId, enabled, isStreaming, options.active]);

  const updateConversationId = useCallback(
    (nextConversationId: string | undefined) => {
      setConversationId(nextConversationId);
    },
    []
  );

  const ensureConversationId = useCallback(async (): Promise<string | undefined> => {
    if (conversationId) return conversationId;
    if (!enabled || !config.connectorId || !currentUserId) return undefined;
    if (!conversationRequestRef.current) {
      conversationRequestRef.current = refetchConversation()
        .then((result) => {
          const nextConversationId = result.data?.conversationId;
          if (nextConversationId) {
            updateConversationId(nextConversationId);
          }
          return nextConversationId;
        })
        .finally(() => {
          conversationRequestRef.current = null;
        });
    }
    return conversationRequestRef.current;
  }, [
    config.connectorId,
    conversationId,
    currentUserId,
    enabled,
    refetchConversation,
    updateConversationId,
  ]);

  useEffect(() => {
    if (!storageReady) return;
    if (skipNextPersistRef.current) {
      skipNextPersistRef.current = false;
      return;
    }
    writeConversationState(conversationStorageKey, {
      version: AGENT_CONVERSATION_STATE_VERSION,
      conversationId,
      messages: normalizeMessagesForStorage(messages),
      activities: normalizeActivitiesForStorage(activities),
      pendingAction,
      updatedAt: Date.now(),
    });
  }, [activities, conversationId, conversationStorageKey, messages, pendingAction, storageReady]);

  const appendAssistantDelta = useCallback((messageId: string, text: string) => {
    setMessages((current) =>
      current.map((message) =>
        message.id === messageId
          ? {
              ...message,
              text: `${message.text}${text}`,
              status: 'streaming',
              thoughts: completeAgentThoughts(message.thoughts),
            }
          : message
      )
    );
  }, []);

  const finishAssistant = useCallback((messageId: string, status: AgentPanelMessage['status']) => {
    setMessages((current) =>
      current.map((message) =>
        message.id === messageId
          ? { ...message, status, thoughts: completeAgentThoughts(message.thoughts) }
          : message
      )
    );
  }, []);

  const appendAssistantThought = useCallback(
    (messageId: string, event: Extract<AgentStreamEvent, { type: 'thought' }>) => {
      setMessages((current) =>
        current.map((message) =>
          message.id === messageId
            ? {
                ...message,
                thoughts: mergeAgentThought(message.thoughts, {
                  id: event.id || createId('agent-thought'),
                  text: event.text,
                  append: event.append,
                  status: event.status,
                  source: 'thought',
                  at:
                    event.at ||
                    (typeof event.lastEventAt === 'number'
                      ? new Date(event.lastEventAt).toISOString()
                      : undefined),
                }),
              }
            : message
        )
      );
    },
    []
  );

  const appendAssistantProcess = useCallback(
    (messageId: string, event: Extract<AgentStreamEvent, { type: 'process' }>) => {
      setMessages((current) =>
        current.map((message) =>
          message.id === messageId
            ? {
                ...message,
                process: upsertAgentProcess(message.process, {
                  id: event.id || createId('agent-process'),
                  kind: event.kind,
                  title: event.title,
                  detail: event.detail,
                  at: event.at,
                }),
              }
            : message
        )
      );
    },
    []
  );

  const stream = useCallback(
    async (input: {
      message: string;
      mode: AgentRuntimeMode;
      approvedAction?: AgentApprovedAction;
      addUserMessage: boolean;
      attachments?: AgentAttachmentRef[];
    }) => {
      if (!client) {
        setError({
          code: 'AGENT_RUNTIME_DISABLED',
          message: 'Agent runtime is not configured.',
          retryable: false,
        });
        return;
      }

      abortRef.current?.abort();
      const abortController = new AbortController();
      abortRef.current = abortController;
      const assistantId = createId('agent-assistant');
      const startedAt = now();
      setError(null);
      setPendingAction(null);
      setIsStreaming(true);
      if (input.addUserMessage) {
        setMessages((current) => [
          ...current,
          {
            id: createId('agent-user'),
            role: 'user',
            text: input.message,
            status: 'done',
            createdAt: startedAt,
            attachments: input.attachments,
          },
          {
            id: assistantId,
            role: 'assistant',
            text: '',
            status: 'streaming',
            createdAt: startedAt,
          },
        ]);
      } else {
        setMessages((current) => [
          ...current,
          {
            id: assistantId,
            role: 'assistant',
            text: '',
            status: 'streaming',
            createdAt: startedAt,
          },
        ]);
      }

      const handleEvent = (event: AgentStreamEvent) => {
        switch (event.type) {
          case 'start':
            activeRunIdRef.current = event.runId;
            updateConversationId(event.conversationId);
            setMessages((current) =>
              current.map((message) =>
                message.id === assistantId
                  ? {
                      ...message,
                      runId: event.runId,
                      conversationId: event.conversationId,
                    }
                  : message
              )
            );
            break;
          case 'tool_request':
            setActivities((current) =>
              upsertActivity(current, {
                callId: event.toolCallId,
                toolName: event.tool,
                title: event.title,
                status: 'calling',
                input: event.input,
                updatedAt: now(),
              })
            );
            appendAssistantProcess(assistantId, {
              type: 'process',
              id: processId('tool-request', event.toolCallId),
              kind: 'tool_call',
              title: event.title,
              detail: event.input,
            });
            break;
          case 'thought':
            appendAssistantThought(assistantId, event);
            break;
          case 'delta':
            appendAssistantDelta(assistantId, event.text);
            break;
          case 'tool_call': {
            const request = parseApprovedAction(event.input) ?? undefined;
            if (request) actionRequestsRef.current.set(event.callId, request);
            setActivities((current) =>
              upsertActivity(current, {
                callId: event.callId,
                toolName: event.toolName,
                title: event.title,
                status: 'calling',
                input: event.input,
                request,
                updatedAt: now(),
              })
            );
            appendAssistantProcess(assistantId, {
              type: 'process',
              id: processId('tool-call', event.callId),
              kind: 'tool_call',
              title: event.title,
              detail: event.input,
            });
            break;
          }
          case 'action_preview': {
            const request = actionRequestsRef.current.get(event.callId);
            setActivities((current) =>
              upsertActivity(current, {
                callId: event.callId,
                toolName: current.find((activity) => activity.callId === event.callId)?.toolName ?? 'actions.execute',
                title: event.title,
                status: 'preview',
                request,
                risk: event.risk,
                summary: event.summary,
                preview: event.preview,
                updatedAt: now(),
              })
            );
            if (request) {
              setPendingAction({
                callId: event.callId,
                message: input.message,
                title: event.title,
                summary: event.summary,
                risk: event.risk,
                request,
                preview: event.preview,
              });
            }
            appendAssistantProcess(assistantId, {
              type: 'process',
              id: processId('action-preview', event.callId),
              kind: 'action_preview',
              title: event.title,
              detail: event.preview,
            });
            break;
          }
          case 'tool_result':
            setActivities((current) =>
              upsertActivity(current, {
                callId: event.callId,
                toolName: current.find((activity) => activity.callId === event.callId)?.toolName ?? 'actions.execute',
                title: current.find((activity) => activity.callId === event.callId)?.title ?? event.callId,
                status: event.status === 'ok' ? 'ok' : 'error',
                output: event.output,
                error: event.error,
                updatedAt: now(),
              })
            );
            setPendingAction((current) => (current?.callId === event.callId ? null : current));
            appendAssistantProcess(assistantId, {
              type: 'process',
              id: processId('tool-result', event.callId),
              kind: event.status === 'ok' ? 'tool_result' : 'error',
              title: event.callId,
              detail: event.status === 'ok' ? event.output : event.error,
            });
            break;
          case 'process':
            appendAssistantProcess(assistantId, event);
            break;
          case 'artifact':
            setMessages((current) =>
              current.map((message) =>
                message.id === assistantId
                  ? { ...message, artifacts: event.files }
                  : message
              )
            );
            break;
          case 'ui_intent':
            setMessages((current) =>
              current.map((message) =>
                message.id === assistantId
                  ? { ...message, uiIntents: [...(message.uiIntents ?? []), event.intent] }
                  : message
              )
            );
            break;
          case 'error':
            setError(event.error);
            finishAssistant(assistantId, 'error');
            break;
          case 'done':
            setMessages((current) =>
              current.map((message) =>
                message.id === assistantId
                  ? { ...message, status: 'done', durationMs: event.durationMs, thoughts: completeAgentThoughts(message.thoughts) }
                  : message
              )
            );
            break;
        }
      };

      try {
        const activeConversationId = await ensureConversationId();
        await client.stream(
          {
            message: input.message,
            conversationId: activeConversationId,
            mode: input.mode,
            approvedAction: input.approvedAction,
            attachments: input.attachments?.map(({ attachmentId, filename }) => ({ attachmentId, filename })),
            locale,
            timezone,
            context,
          },
          { onEvent: handleEvent },
          abortController.signal
        );
        finishAssistant(assistantId, 'done');
      } catch (err) {
        if (!abortController.signal.aborted) {
          const agentError = runtimeError(err);
          setError(agentError);
          finishAssistant(assistantId, 'error');
        }
      } finally {
        if (abortRef.current === abortController) abortRef.current = null;
        activeRunIdRef.current = undefined;
        setIsStreaming(false);
      }
    },
    [
      appendAssistantDelta,
      appendAssistantProcess,
      appendAssistantThought,
      client,
      context,
      ensureConversationId,
      finishAssistant,
      locale,
      timezone,
      updateConversationId,
    ]
  );

  const sendMessage = useCallback(
    async (message: string) => {
      const trimmed = message.trim();
      if (!trimmed || isStreaming) return;
      const turnAttachments = attachments;
      setAttachments([]);
      await stream({ message: trimmed, mode: 'preview', addUserMessage: true, attachments: turnAttachments });
    },
    [attachments, isStreaming, stream]
  );

  const uploadAttachments = useCallback(async (files: File[]) => {
    if (!files.length || isUploading) return;
    const available = Math.max(0, MAX_ATTACHMENT_COUNT - attachments.length);
    if (files.length > available) {
      setError({ code: 'ATTACHMENT_LIMIT_EXCEEDED', message: 'Attachment limit exceeded.' });
      return;
    }
    if (files.some((file) => file.size === 0)) {
      setError({ code: 'ATTACHMENT_EMPTY', message: 'Empty files cannot be uploaded.' });
      return;
    }
    if (files.some((file) => file.size > MAX_ATTACHMENT_BYTES)) {
      setError({ code: 'ATTACHMENT_TOO_LARGE', message: 'Attachment is too large.' });
      return;
    }
    setIsUploading(true);
    setError(null);
    try {
      await ensureConversationId();
      const uploaded: AgentAttachmentRef[] = [];
      for (const file of files) {
        uploaded.push(await agentApi.uploadAttachment(file));
      }
      setAttachments((current) => [...current, ...uploaded].slice(0, MAX_ATTACHMENT_COUNT));
    } catch (err) {
      setError(runtimeError(err));
    } finally {
      setIsUploading(false);
    }
  }, [attachments.length, ensureConversationId, isUploading]);

  const removeAttachment = useCallback(async (attachmentId: string) => {
    setAttachments((current) => current.filter((item) => item.attachmentId !== attachmentId));
    try {
      await agentApi.removeAttachment(attachmentId);
    } catch (err) {
      setError(runtimeError(err));
    }
  }, []);

  const stop = useCallback(async () => {
    const runId = activeRunIdRef.current;
    if (runId) {
      await agentApi.cancelRun(runId).catch(() => false);
    }
    abortRef.current?.abort();
    abortRef.current = null;
    activeRunIdRef.current = undefined;
    setMessages((current) => current.map((message) =>
      message.status === 'streaming'
        ? { ...message, status: 'cancelled', thoughts: completeAgentThoughts(message.thoughts) }
        : message
    ));
    setIsStreaming(false);
  }, []);

  const executePendingAction = useCallback(async () => {
    if (!pendingAction || isStreaming) return;
    setActivities((current) =>
      current.map((activity) =>
        activity.callId === pendingAction.callId
          ? { ...activity, status: 'running', updatedAt: now() }
          : activity
      )
    );
    await stream({
      message: pendingAction.message,
      mode: 'execute',
      approvedAction: { ...pendingAction.request, dryRun: false },
      addUserMessage: false,
    });
  }, [isStreaming, pendingAction, stream]);

  const cancelPendingAction = useCallback(() => {
    if (!pendingAction) return;
    setActivities((current) =>
      current.map((activity) =>
        activity.callId === pendingAction.callId
          ? { ...activity, status: 'cancelled', updatedAt: now() }
          : activity
      )
    );
    setPendingAction(null);
  }, [pendingAction]);

  const clear = useCallback(async () => {
    abortRef.current?.abort();
    abortRef.current = null;
    actionRequestsRef.current.clear();
    setIsStreaming(false);
    setError(null);
    await Promise.allSettled(attachments.map((item) => agentApi.removeAttachment(item.attachmentId)));
    setAttachments([]);
    if (!enabled || !config.connectorId || !currentUserId) {
      writeConversationState(conversationStorageKey, undefined);
      skipNextPersistRef.current = true;
      setMessages([]);
      setActivities([]);
      setPendingAction(null);
      updateConversationId(undefined);
      return;
    }
    try {
      const next = await startConversation.mutateAsync();
      writeConversationState(conversationStorageKey, undefined);
      skipNextPersistRef.current = true;
      setMessages([]);
      setActivities([]);
      setPendingAction(null);
      updateConversationId(next.conversationId);
    } catch (err) {
      setError(runtimeError(err));
    }
  }, [
    config.connectorId,
    attachments,
    conversationStorageKey,
    currentUserId,
    enabled,
    startConversation,
    updateConversationId,
  ]);

  return {
    enabled,
    messages,
    activities,
    pendingAction,
    isStreaming,
    isUploading,
    attachments,
    error,
    sendMessage,
    executePendingAction,
    cancelPendingAction,
    uploadAttachments,
    removeAttachment,
    stop,
    clear,
  };
}
