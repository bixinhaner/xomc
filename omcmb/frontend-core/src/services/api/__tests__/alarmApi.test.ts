import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getMock, postMock } = vi.hoisted(() => ({ getMock: vi.fn(), postMock: vi.fn() }));

vi.mock('../../http', () => ({
  default: { get: getMock, post: postMock, put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
}));

import { alarmApi } from '../alarmApi';

beforeEach(() => {
  getMock.mockReset();
  getMock.mockResolvedValue({
    data: { items: [], total: 0, page: 1, page_size: 20, total_pages: 0 },
  });
  postMock.mockReset();
});

describe('alarmApi email notification rule', () => {
  it('persists and maps email recipients', async () => {
    postMock.mockResolvedValue({
      data: {
        id: 'rule-email',
        name: 'NOC email',
        filter_type: 'alarm_identifier',
        alarm_sources: [],
        alarm_identifiers: ['11185'],
        device_ids: [],
        device_group_ids: [],
        action: 'notify_email',
        acknowledge_desc: '',
        email_recipients: ['noc@example.com'],
        effective_start: '2026-08-12T01:00:00Z',
        effective_end: '2026-08-12T02:00:00Z',
        priority: 0,
        enabled: true,
        created_at: '2026-08-12T00:00:00Z',
        updated_at: '2026-08-12T00:00:00Z',
      },
    });

    const result = await alarmApi.createRule({
      ruleName: 'NOC email',
      ruleType: 'notify_email',
      severity: 'warning',
      enabled: true,
      conditions: [{ field: 'alarm_identifier', operator: 'contains', value: ['11185'] }],
      actions: [{ type: 'email', target: 'noc@example.com' }],
      emailRecipients: ['noc@example.com'],
      effectiveStart: '2026-08-12T01:00:00Z',
      effectiveEnd: '2026-08-12T02:00:00Z',
    });

    expect(postMock).toHaveBeenCalledWith('/alarms/alarm-filters', expect.objectContaining({
      action: 'notify_email',
      email_recipients: ['noc@example.com'],
      effective_start: '2026-08-12T01:00:00Z',
      effective_end: '2026-08-12T02:00:00Z',
    }));
    expect(result.emailRecipients).toEqual(['noc@example.com']);
    expect(result.effectiveStart).toBe('2026-08-12T01:00:00Z');
    expect(result.effectiveEnd).toBe('2026-08-12T02:00:00Z');
    expect(result.actions).toContainEqual({ type: 'email', target: 'noc@example.com' });
  });
});

describe('alarmApi.getCurrentAlarms', () => {
  it('serializes multiple severities as CSV for backend IN filtering', async () => {
    await alarmApi.getCurrentAlarms({
      severity: ['critical', 'minor'],
      page: 1,
      pageSize: 20,
    });

    const [, options] = getMock.mock.calls[0];
    expect(options.params.severity).toBe('1,3');
  });
});
