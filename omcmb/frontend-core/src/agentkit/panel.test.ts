import { describe, expect, it } from 'vitest';
import {
  extractAgentRows,
  formatAgentValue,
  mergeAgentThought,
  parseApprovedAction,
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
});
