import { describe, expect, it, vi } from 'vitest';
import {
  AgentRuntimeError,
  createAgentRuntimeClient,
  type AgentStreamEvent,
} from './runtimeClient';
import type { AgentRuntimeRequest } from './protocol';

const request: AgentRuntimeRequest = {
  message: 'show active alarms',
  locale: 'zh-CN',
  timezone: 'Asia/Shanghai',
  context: { path: '/devices' },
};

function streamResponse(chunks: string[], init?: ResponseInit): Response {
  const encoder = new TextEncoder();
  return new Response(
    new ReadableStream({
      start(controller) {
        for (const chunk of chunks) {
          controller.enqueue(encoder.encode(chunk));
        }
        controller.close();
      },
    }),
    {
      status: 200,
      headers: { 'Content-Type': 'text/event-stream' },
      ...init,
    }
  );
}

describe('createAgentRuntimeClient', () => {
  it('streams validated SSE events in order', async () => {
    const events: AgentStreamEvent[] = [];
    const fetchImpl = vi.fn().mockResolvedValue(
      streamResponse([
        'data: {"type":"start","runId":"run-1","conversationId":"conv-1"}\n\n',
        'data: {"type":"delta","text":"hello"}\n',
        '\n',
        'data: {"type":"done"}\n\n',
      ])
    ) as unknown as typeof fetch;

    const client = createAgentRuntimeClient({
      endpoint: '/runtime',
      getAuthHeaders: async () => ({ Authorization: 'Bearer web-token' }),
      fetchImpl,
    });

    await client.stream(request, { onEvent: (event) => events.push(event) });

    expect(events.map((event) => event.type)).toEqual([
      'start',
      'delta',
      'done',
    ]);
    expect(fetchImpl).toHaveBeenCalledWith(
      '/runtime',
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({
          Authorization: 'Bearer web-token',
          Accept: 'text/event-stream',
        }),
      })
    );
  });

  it('ignores named heartbeat frames from SSE streams', async () => {
    const events: AgentStreamEvent[] = [];
    const fetchImpl = vi.fn().mockResolvedValue(
      streamResponse([
        'event: agent\n',
        'data: {"type":"start","runId":"run-1","conversationId":"conv-1"}\n\n',
        'event: ping\n',
        'data: {"now":"2026-07-08T00:00:00.000Z"}\n\n',
        'event: agent\n',
        'data: {"type":"delta","text":"hello"}\n\n',
        'event: agent\n',
        'data: {"type":"done"}\n\n',
      ])
    ) as unknown as typeof fetch;

    const client = createAgentRuntimeClient({
      endpoint: '/runtime',
      fetchImpl,
    });

    await client.stream(request, { onEvent: (event) => events.push(event) });

    expect(events.map((event) => event.type)).toEqual([
      'start',
      'delta',
      'done',
    ]);
  });

  it('maps named SSE events to runtime event payloads', async () => {
    const events: AgentStreamEvent[] = [];
    const fetchImpl = vi.fn().mockResolvedValue(
      streamResponse([
        'event: start\n',
        'data: {"runId":"run-1","conversationId":"conv-1"}\n\n',
        'event: thought\n',
        'data: {"id":"thought-1","text":"thinking","append":true,"status":"streaming"}\n\n',
        'event: process\n',
        'data: {"id":"process-1","kind":"process","title":"Workspace operation completed","detail":{"total":1}}\n\n',
        'event: delta\n',
        'data: {"text":"hello"}\n\n',
        'event: done\n',
        'data: {}\n\n',
      ])
    ) as unknown as typeof fetch;

    const client = createAgentRuntimeClient({
      endpoint: '/runtime',
      fetchImpl,
    });

    await client.stream(request, { onEvent: (event) => events.push(event) });

    expect(events).toEqual([
      { type: 'start', runId: 'run-1', conversationId: 'conv-1' },
      {
        type: 'thought',
        id: 'thought-1',
        text: 'thinking',
        append: true,
        status: 'streaming',
      },
      {
        type: 'process',
        id: 'process-1',
        kind: 'process',
        title: 'Workspace operation completed',
        detail: { total: 1 },
      },
      { type: 'delta', text: 'hello' },
      { type: 'done' },
    ]);
  });

  it('rejects malformed SSE payloads', async () => {
    const onError = vi.fn();
    const client = createAgentRuntimeClient({
      endpoint: '/runtime',
      getAuthHeaders: async () => ({ Authorization: 'Bearer web-token' }),
      fetchImpl: vi
        .fn()
        .mockResolvedValue(streamResponse(['data: {"type":"delta"}\n\n'])) as unknown as typeof fetch,
    });

    await expect(client.stream(request, { onError })).rejects.toMatchObject({
      code: 'INVALID_STREAM_EVENT',
    });
    expect(onError).toHaveBeenCalledWith(
      expect.objectContaining({ code: 'INVALID_STREAM_EVENT' })
    );
  });

  it('converts HTTP errors to AgentRuntimeError', async () => {
    const onError = vi.fn();
    const client = createAgentRuntimeClient({
      endpoint: '/runtime',
      getAuthHeaders: async () => ({ Authorization: 'Bearer web-token' }),
      fetchImpl: vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: 'bad gateway' }), {
          status: 502,
          headers: { 'Content-Type': 'application/json' },
        })
      ) as unknown as typeof fetch,
    });

    await expect(client.stream(request, { onError })).rejects.toMatchObject({
      code: 'HTTP_502',
      retryable: true,
    });
    expect(onError).toHaveBeenCalledWith(
      expect.objectContaining({ code: 'HTTP_502' })
    );
  });

  it('supports aborting a runtime request', async () => {
    const controller = new AbortController();
    const fetchImpl = vi.fn((_url, init) => {
      const signal = init?.signal as AbortSignal;
      return new Promise<Response>((_resolve, reject) => {
        if (signal.aborted) {
          reject(new DOMException('Aborted', 'AbortError'));
          return;
        }
        signal.addEventListener('abort', () => {
          reject(new DOMException('Aborted', 'AbortError'));
        });
      });
    }) as unknown as typeof fetch;

    const client = createAgentRuntimeClient({
      endpoint: '/runtime',
      getAuthHeaders: async () => ({ Authorization: 'Bearer web-token' }),
      fetchImpl,
    });

    const promise = client.stream(request, {}, controller.signal);
    controller.abort();

    await expect(promise).rejects.toMatchObject({ code: 'REQUEST_ABORTED' });
  });
});
