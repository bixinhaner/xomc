import { describe, expect, it } from 'vitest';
import {
  isAgentActionDescriptor,
  isAgentError,
  isAgentStreamEvent,
} from './protocol';

describe('agentkit protocol guards', () => {
  it('accepts all supported stream event variants', () => {
    const events = [
      { type: 'start', runId: 'run-1', conversationId: 'conv-1' },
      { type: 'delta', text: 'hello' },
      {
        type: 'tool_call',
        callId: 'call-1',
        toolName: 'actions.search',
        title: 'Search',
        input: { query: 'abc' },
      },
      {
        type: 'action_preview',
        callId: 'call-1',
        title: 'Preview',
        summary: 'Read device rows',
        risk: 'read',
        preview: { rows: [] },
      },
      {
        type: 'tool_result',
        callId: 'call-1',
        status: 'ok',
        output: { rows: [] },
      },
      { type: 'done', usage: { inputTokens: 1, outputTokens: 2 } },
      {
        type: 'error',
        error: { code: 'UPSTREAM_ERROR', message: 'failed' },
      },
    ];

    expect(events.every(isAgentStreamEvent)).toBe(true);
  });

  it('rejects malformed stream events', () => {
    expect(isAgentStreamEvent({ type: 'delta' })).toBe(false);
    expect(
      isAgentStreamEvent({
        type: 'action_preview',
        callId: 'call-1',
        title: 'Preview',
        summary: 'Summary',
        risk: 'write',
        preview: {},
      })
    ).toBe(false);
    expect(
      isAgentStreamEvent({
        type: 'tool_result',
        callId: 'call-1',
        status: 'error',
        error: { message: 'missing code' },
      })
    ).toBe(false);
    expect(isAgentStreamEvent({ type: 'unknown' })).toBe(false);
  });

  it('validates errors and action descriptors', () => {
    expect(
      isAgentError({
        code: 'FORBIDDEN',
        message: 'Forbidden',
        retryable: false,
      })
    ).toBe(true);

    expect(
      isAgentActionDescriptor({
        id: 'device.search',
        title: 'Search devices',
        description: 'Search visible devices',
        inputSchema: { type: 'object' },
        risk: 'read',
        scopes: ['device:read'],
      })
    ).toBe(true);
  });
});

