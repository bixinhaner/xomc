import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getMock } = vi.hoisted(() => ({ getMock: vi.fn() }));

vi.mock('../../http', () => ({
  default: { get: getMock, post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
}));

import { alarmApi } from '../alarmApi';

beforeEach(() => {
  getMock.mockReset();
  getMock.mockResolvedValue({
    data: { items: [], total: 0, page: 1, page_size: 20, total_pages: 0 },
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