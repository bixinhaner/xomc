import { describe, expect, it } from 'vitest';
import {
  extractAgentRows,
  formatAgentValue,
  parseApprovedAction,
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
});

