import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  type AgentApprovedAction,
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
import { useAppStore } from '../store/appStore';
import { useUserStore } from '../store/userStore';
import { useAgentRuntimeClient } from './useAgentRuntimeClient';

export interface UseAgentPanelControllerOptions {
  context: AgentPageContext;
}

export interface UseAgentPanelControllerResult {
  enabled: boolean;
  messages: AgentPanelMessage[];
  activities: AgentPanelActivity[];
  pendingAction: AgentPendingAction | null;
  isStreaming: boolean;
  error: AgentError | null;
  sendMessage: (message: string) => Promise<void>;
  executePendingAction: () => Promise<void>;
  cancelPendingAction: () => void;
  clear: () => void;
}

let nextAgentPanelId = 0;
const AGENT_CONVERSATION_STORAGE_PREFIX = 'omc-agent-conversation';

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
}): string | undefined {
  if (!input.connectorId || !input.userId) return undefined;
  return `${AGENT_CONVERSATION_STORAGE_PREFIX}:${input.connectorId}:${input.userId}`;
}

function readConversationId(storageKey: string | undefined): string | undefined {
  if (!storageKey || typeof window === 'undefined') return undefined;
  try {
    return window.localStorage.getItem(storageKey) || undefined;
  } catch {
    return undefined;
  }
}

function writeConversationId(storageKey: string | undefined, conversationId: string | undefined) {
  if (!storageKey || typeof window === 'undefined') return;
  try {
    if (conversationId) {
      window.localStorage.setItem(storageKey, conversationId);
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
  const { enabled, client, config } = useAgentRuntimeClient();
  const [messages, setMessages] = useState<AgentPanelMessage[]>([]);
  const [activities, setActivities] = useState<AgentPanelActivity[]>([]);
  const [pendingAction, setPendingAction] = useState<AgentPendingAction | null>(null);
  const [conversationId, setConversationId] = useState<string | undefined>();
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState<AgentError | null>(null);
  const actionRequestsRef = useRef(new Map<string, AgentApprovedAction>());
  const abortRef = useRef<AbortController | null>(null);

  const timezone = systemTimezone || 'UTC';
  const context = useMemo(() => options.context, [options.context]);
  const conversationStorageKey = useMemo(
    () => buildConversationStorageKey({ connectorId: config.connectorId, userId: currentUserId }),
    [config.connectorId, currentUserId]
  );

  useEffect(() => {
    setConversationId(readConversationId(conversationStorageKey));
  }, [conversationStorageKey]);

  const updateConversationId = useCallback(
    (nextConversationId: string | undefined) => {
      setConversationId(nextConversationId);
      writeConversationId(conversationStorageKey, nextConversationId);
    },
    [conversationStorageKey]
  );

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
            updateConversationId(event.conversationId);
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
          case 'error':
            setError(event.error);
            finishAssistant(assistantId, 'error');
            break;
          case 'done':
            finishAssistant(assistantId, 'done');
            break;
        }
      };

      try {
        await client.stream(
          {
            message: input.message,
            conversationId,
            mode: input.mode,
            approvedAction: input.approvedAction,
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
        setIsStreaming(false);
      }
    },
    [
      appendAssistantDelta,
      appendAssistantProcess,
      appendAssistantThought,
      client,
      context,
      conversationId,
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
      await stream({ message: trimmed, mode: 'preview', addUserMessage: true });
    },
    [isStreaming, stream]
  );

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

  const clear = useCallback(() => {
    abortRef.current?.abort();
    abortRef.current = null;
    actionRequestsRef.current.clear();
    setMessages([]);
    setActivities([]);
    setPendingAction(null);
    updateConversationId(undefined);
    setIsStreaming(false);
    setError(null);
  }, [updateConversationId]);

  return {
    enabled,
    messages,
    activities,
    pendingAction,
    isStreaming,
    error,
    sendMessage,
    executePendingAction,
    cancelPendingAction,
    clear,
  };
}
