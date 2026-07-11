import { describe, expect, it } from 'vitest';
import {
  extractAgentRestInput,
  extractAgentRows,
  formatAgentValue,
  mergeAgentThought,
  parseApprovedAction,
  summarizeAgentProcess,
  completeAgentThoughts,
  upsertAgentProcess,
} from './panel';

describe('agent panel helpers', () => {
  it('parses approved action requests from tool call input', () => {
    expect(
      parseApprovedAction({
        actionId: 'system.health',
        input: { limit: 5 },
        dryRun: true,
      })
    ).toEqual({
      actionId: 'system.health',
      input: { limit: 5 },
      dryRun: true,
    });

    expect(parseApprovedAction({ actionId: '' })).toBeNull();
    expect(parseApprovedAction({ input: {} })).toBeNull();
  });

  it('formats compact display values', () => {
    expect(formatAgentValue(undefined)).toBe('-');
    expect(formatAgentValue(['a', 'b'])).toBe('2');
    expect(formatAgentValue({ status: 'ok' })).toBe('{"status":"ok"}');
  });

  it('extracts rows from nested action results and statistics first', () => {
    expect(
      extractAgentRows({
        actionId: 'alarm.active_summary',
        status: 'ok',
        result: {
          statistics: {
            total_active: 158,
            critical: 18,
          },
          items: [{ id: 1 }, { id: 2 }],
        },
      })
    ).toEqual([
      { key: 'total_active', value: '158' },
      { key: 'critical', value: '18' },
      { key: 'items', value: '2' },
    ]);
  });

  it('merges streaming thought chunks and marks them completed', () => {
    const streaming = mergeAgentThought([], {
      id: 'thought-1',
      text: 'Checking',
      append: true,
      source: 'thought',
    });
    const merged = mergeAgentThought(streaming, {
      id: 'thought-1',
      text: ' actions',
      append: true,
      source: 'thought',
    });

    expect(merged).toEqual([
      {
        id: 'thought-1',
        text: 'Checking actions',
        lines: ['Checking actions'],
        status: 'streaming',
        source: 'thought',
        at: undefined,
      },
    ]);
    expect(completeAgentThoughts(merged)?.[0]?.status).toBe('completed');
  });

  it('upserts process entries by id', () => {
    const entries = upsertAgentProcess([], {
      id: 'process-1',
      kind: 'process',
      title: 'Started',
    });

    expect(
      upsertAgentProcess(entries, {
        id: 'process-1',
        kind: 'process',
        title: 'Completed',
        detail: { ok: true },
      })
    ).toEqual([
      {
        id: 'process-1',
        kind: 'process',
        title: 'Completed',
        detail: { ok: true },
      },
    ]);
  });

  it('extracts REST input and summarizes process records', () => {
    expect(
      extractAgentRestInput({
        method: 'get',
        path: '/api/v1/alarms/statistics',
        operationId: 'get.alarms.statistics',
        query: { page: 1 },
      })
    ).toEqual({
      method: 'GET',
      path: '/api/v1/alarms/statistics',
      operationId: 'get.alarms.statistics',
      query: { page: 1 },
      body: undefined,
      reason: '',
    });

    expect(
      summarizeAgentProcess([
        {
          id: 'p1',
          kind: 'tool_call',
          title: 'GET /api/v1/agent/catalog',
          detail: { method: 'GET', path: '/api/v1/agent/catalog' },
        },
        {
          id: 'p2',
          kind: 'tool_call',
          title: 'GET /api/v1/alarms/statistics',
          detail: { method: 'GET', path: '/api/v1/alarms/statistics' },
        },
        {
          id: 'p3',
          kind: 'error',
          title: 'failed',
        },
      ])
    ).toEqual({
      total: 3,
      searches: 1,
      calls: 1,
      readOnly: 2,
      writes: 0,
      errors: 1,
    });
  });
});
