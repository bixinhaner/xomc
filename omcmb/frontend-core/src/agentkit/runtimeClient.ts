import {
  isAgentStreamEvent,
  type AgentError,
  type AgentRuntimeRequest,
  type AgentStreamEvent,
} from './protocol';

export interface AgentStreamHandlers {
  onEvent?: (event: AgentStreamEvent) => void;
  onStart?: (event: Extract<AgentStreamEvent, { type: 'start' }>) => void;
  onDelta?: (event: Extract<AgentStreamEvent, { type: 'delta' }>) => void;
  onThought?: (event: Extract<AgentStreamEvent, { type: 'thought' }>) => void;
  onToolCall?: (
    event: Extract<AgentStreamEvent, { type: 'tool_call' }>
  ) => void;
  onActionPreview?: (
    event: Extract<AgentStreamEvent, { type: 'action_preview' }>
  ) => void;
  onToolResult?: (
    event: Extract<AgentStreamEvent, { type: 'tool_result' }>
  ) => void;
  onProcess?: (event: Extract<AgentStreamEvent, { type: 'process' }>) => void;
  onDone?: (event: Extract<AgentStreamEvent, { type: 'done' }>) => void;
  onError?: (error: AgentError) => void;
}

export interface AgentRuntimeClientOptions {
  endpoint: string;
  getAuthHeaders?: () => Promise<Record<string, string>> | Record<string, string>;
  fetchImpl?: typeof fetch;
}

export interface AgentRuntimeClient {
  stream(
    request: AgentRuntimeRequest,
    handlers: AgentStreamHandlers,
    signal?: AbortSignal
  ): Promise<void>;
}

export class AgentRuntimeError extends Error implements AgentError {
  code: string;
  retryable?: boolean;
  requestId?: string;
  details?: unknown;

  constructor(error: AgentError) {
    super(error.message);
    this.name = 'AgentRuntimeError';
    this.code = error.code;
    this.retryable = error.retryable;
    this.requestId = error.requestId;
    this.details = error.details;
  }
}

function toRuntimeError(error: AgentError): AgentRuntimeError {
  return error instanceof AgentRuntimeError
    ? error
    : new AgentRuntimeError(error);
}

function emitError(handlers: AgentStreamHandlers, error: AgentError): never {
  handlers.onError?.(error);
  handlers.onEvent?.({ type: 'error', error });
  throw toRuntimeError(error);
}

function emitEvent(handlers: AgentStreamHandlers, event: AgentStreamEvent) {
  handlers.onEvent?.(event);

  switch (event.type) {
    case 'start':
      handlers.onStart?.(event);
      break;
    case 'delta':
      handlers.onDelta?.(event);
      break;
    case 'thought':
      handlers.onThought?.(event);
      break;
    case 'tool_call':
      handlers.onToolCall?.(event);
      break;
    case 'action_preview':
      handlers.onActionPreview?.(event);
      break;
    case 'tool_result':
      handlers.onToolResult?.(event);
      break;
    case 'process':
      handlers.onProcess?.(event);
      break;
    case 'done':
      handlers.onDone?.(event);
      break;
    case 'error':
      handlers.onError?.(event.error);
      break;
  }
}

function parseRetryable(status: number): boolean {
  return status === 408 || status === 429 || status >= 500;
}

async function readErrorBody(response: Response): Promise<unknown> {
  const contentType = response.headers.get('content-type') || '';
  try {
    if (contentType.includes('application/json')) {
      return await response.json();
    }
    return await response.text();
  } catch {
    return undefined;
  }
}

function splitNextFrame(buffer: string): [string | undefined, string] {
  const lfIndex = buffer.indexOf('\n\n');
  const crlfIndex = buffer.indexOf('\r\n\r\n');

  if (lfIndex === -1 && crlfIndex === -1) return [undefined, buffer];
  if (crlfIndex !== -1 && (lfIndex === -1 || crlfIndex < lfIndex)) {
    return [buffer.slice(0, crlfIndex), buffer.slice(crlfIndex + 4)];
  }
  return [buffer.slice(0, lfIndex), buffer.slice(lfIndex + 2)];
}

function parseFrame(frame: string): { eventName?: string; data?: string } {
  const lines = frame.split(/\r?\n/);
  const dataLines: string[] = [];
  let eventName: string | undefined;

  for (const line of lines) {
    if (line.startsWith('event:')) {
      const nextEventName = line.slice(6).trim();
      if (nextEventName) eventName = nextEventName;
    }
    if (line.startsWith('data:')) {
      dataLines.push(line.slice(5).trimStart());
    }
  }

  return {
    eventName,
    data: dataLines.length > 0 ? dataLines.join('\n') : undefined,
  };
}

function normalizeEventPayload(eventName: string | undefined, parsed: unknown): unknown {
  if (isAgentStreamEvent(parsed)) return parsed;
  if (!eventName || eventName === 'message' || eventName === 'agent') return parsed;
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return parsed;

  const record = parsed as Record<string, unknown>;
  if (eventName === 'error') {
    return {
      type: 'error',
      error: {
        code: typeof record.code === 'string' ? record.code : 'AGENT_STREAM_ERROR',
        message: typeof record.message === 'string'
          ? record.message
          : typeof record.detail === 'string'
            ? record.detail
            : 'Agent stream failed.',
        retryable: typeof record.retryable === 'boolean' ? record.retryable : true,
      },
    };
  }

  return { type: eventName, ...record };
}

function parseEventPayload(payload: string, eventName?: string): AgentStreamEvent {
  let parsed: unknown;
  try {
    parsed = JSON.parse(payload);
  } catch {
    throw new AgentRuntimeError({
      code: 'INVALID_STREAM_EVENT',
      message: 'Agent stream event is not valid JSON.',
      retryable: false,
      details: payload,
    });
  }

  const normalized = normalizeEventPayload(eventName, parsed);
  if (!isAgentStreamEvent(normalized)) {
    throw new AgentRuntimeError({
      code: 'INVALID_STREAM_EVENT',
      message: 'Agent stream event has an invalid shape.',
      retryable: false,
      details: normalized,
    });
  }

  return normalized;
}

function shouldIgnoreFrame(eventName: string | undefined): boolean {
  return eventName === 'ping' || eventName === 'heartbeat';
}

async function consumeStream(
  response: Response,
  handlers: AgentStreamHandlers,
  signal?: AbortSignal
) {
  if (!response.body) {
    emitError(handlers, {
      code: 'EMPTY_STREAM',
      message: 'Agent runtime returned an empty stream.',
      retryable: true,
    });
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  try {
    for (;;) {
      if (signal?.aborted) {
        emitError(handlers, {
          code: 'REQUEST_ABORTED',
          message: 'Agent request was aborted.',
          retryable: true,
        });
      }

      const { value, done } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      for (;;) {
        const [frame, rest] = splitNextFrame(buffer);
        if (frame === undefined) {
          buffer = rest;
          break;
        }
        buffer = rest;

        const { eventName, data: payload } = parseFrame(frame);
        if (shouldIgnoreFrame(eventName) || !payload || payload === '[DONE]') continue;

        try {
          emitEvent(handlers, parseEventPayload(payload, eventName));
        } catch (error) {
          if (error instanceof AgentRuntimeError) {
            emitError(handlers, error);
          }
          throw error;
        }
      }
    }

    buffer += decoder.decode();
    const trailing = parseFrame(buffer);
    if (!shouldIgnoreFrame(trailing.eventName) && trailing.data && trailing.data !== '[DONE]') {
      try {
        emitEvent(handlers, parseEventPayload(trailing.data, trailing.eventName));
      } catch (error) {
        if (error instanceof AgentRuntimeError) {
          emitError(handlers, error);
        }
        throw error;
      }
    }
  } finally {
    reader.releaseLock();
  }
}

export function createAgentRuntimeClient(
  options: AgentRuntimeClientOptions
): AgentRuntimeClient {
  const fetchImpl = options.fetchImpl ?? fetch;

  return {
    async stream(request, handlers, signal) {
      let response: Response;

      try {
        const authHeaders = options.getAuthHeaders ? await options.getAuthHeaders() : {};
        response = await fetchImpl(options.endpoint, {
          method: 'POST',
          headers: {
            ...authHeaders,
            'Content-Type': 'application/json',
            Accept: 'text/event-stream',
          },
          body: JSON.stringify(request),
          signal,
        });
      } catch (error) {
        if (signal?.aborted) {
          emitError(handlers, {
            code: 'REQUEST_ABORTED',
            message: 'Agent request was aborted.',
            retryable: true,
          });
        }
        emitError(handlers, {
          code: 'NETWORK_ERROR',
          message: 'Agent runtime is unreachable.',
          retryable: true,
          details: error,
        });
      }

      if (!response.ok) {
        emitError(handlers, {
          code: `HTTP_${response.status}`,
          message: `Agent runtime request failed with HTTP ${response.status}.`,
          retryable: parseRetryable(response.status),
          details: await readErrorBody(response),
        });
      }

      await consumeStream(response, handlers, signal);
    },
  };
}
