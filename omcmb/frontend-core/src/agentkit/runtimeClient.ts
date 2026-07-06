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
  onToolCall?: (
    event: Extract<AgentStreamEvent, { type: 'tool_call' }>
  ) => void;
  onActionPreview?: (
    event: Extract<AgentStreamEvent, { type: 'action_preview' }>
  ) => void;
  onToolResult?: (
    event: Extract<AgentStreamEvent, { type: 'tool_result' }>
  ) => void;
  onDone?: (event: Extract<AgentStreamEvent, { type: 'done' }>) => void;
  onError?: (error: AgentError) => void;
}

export interface AgentRuntimeClientOptions {
  endpoint: string;
  getDelegationToken: () => Promise<string>;
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
    case 'tool_call':
      handlers.onToolCall?.(event);
      break;
    case 'action_preview':
      handlers.onActionPreview?.(event);
      break;
    case 'tool_result':
      handlers.onToolResult?.(event);
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

function parseFrameData(frame: string): string | undefined {
  const lines = frame.split(/\r?\n/);
  const dataLines: string[] = [];

  for (const line of lines) {
    if (line.startsWith('data:')) {
      dataLines.push(line.slice(5).trimStart());
    }
  }

  return dataLines.length > 0 ? dataLines.join('\n') : undefined;
}

function parseEventPayload(payload: string): AgentStreamEvent {
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

  if (!isAgentStreamEvent(parsed)) {
    throw new AgentRuntimeError({
      code: 'INVALID_STREAM_EVENT',
      message: 'Agent stream event has an invalid shape.',
      retryable: false,
      details: parsed,
    });
  }

  return parsed;
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

        const payload = parseFrameData(frame);
        if (!payload || payload === '[DONE]') continue;

        try {
          emitEvent(handlers, parseEventPayload(payload));
        } catch (error) {
          if (error instanceof AgentRuntimeError) {
            emitError(handlers, error);
          }
          throw error;
        }
      }
    }

    buffer += decoder.decode();
    const trailing = parseFrameData(buffer);
    if (trailing && trailing !== '[DONE]') {
      try {
        emitEvent(handlers, parseEventPayload(trailing));
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
        const token = await options.getDelegationToken();
        response = await fetchImpl(options.endpoint, {
          method: 'POST',
          headers: {
            Authorization: `Bearer ${token}`,
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
