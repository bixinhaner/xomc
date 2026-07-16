import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
}));

vi.mock('../../http', () => ({
  default: { get: getMock },
}));

import { pmDashboardApi } from '../pmDashboardApi';

beforeEach(() => {
  getMock.mockReset();
});

describe('pmDashboardApi.queryAggregated', () => {
  it('按透视表行分页时发送 page_by、limit/offset 和多设备过滤', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [],
        total: 123,
        requested_start_time: '2026-07-16T00:00:00Z',
        requested_end_time: '2026-07-16T03:00:00Z',
      },
    });

    const out = await pmDashboardApi.queryAggregated({
      granularity: '15min',
      deviceSns: ['SN-1', 'SN-2'],
      metricPaths: ['K1', 'K2'],
      startTime: '2026-07-16T00:00:00Z',
      endTime: '2026-07-16T03:00:00Z',
      limit: 50,
      offset: 100,
      pageBy: 'pivot_row',
    });

    const [url, opts] = getMock.mock.calls[0];
    expect(url).toBe('/pm/metrics/aggregated');
    expect(opts.params).toMatchObject({
      granularity: '15min',
      device_sns: 'SN-1,SN-2',
      metric_paths: 'K1,K2',
      limit: 50,
      offset: 100,
      page_by: 'pivot_row',
    });
    expect(out.total).toBe(123);
  });
});
