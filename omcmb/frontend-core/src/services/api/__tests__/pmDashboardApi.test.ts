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
      technologies: ['lte'],
      startTime: '2026-07-16T00:00:00Z',
      endTime: '2026-07-16T03:00:00Z',
      limit: 50,
      offset: 100,
      pageBy: 'pivot_row',
      countMode: 'n_plus_one',
    });

    const [url, opts] = getMock.mock.calls[0];
    expect(url).toBe('/pm/metrics/aggregated');
    expect(opts.params).toMatchObject({
      granularity: '15min',
      device_sns: 'SN-1,SN-2',
      metric_paths: 'K1,K2',
      technologies: 'lte',
      limit: 50,
      offset: 100,
      page_by: 'pivot_row',
      count_mode: 'n_plus_one',
    });
    expect(opts.timeout).toBe(60_000);
    expect(out.total).toBe(123);
    expect(out.truncated).toBe(false);
  });

  it('读取后端 truncated 标记，不依赖精确 total 判断', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [{ metric_path: 'K1', metric_type: 'kpi', metric_value: 1, granularity: '15min', time: '2026-07-16T00:00:00Z', start_time: '2026-07-16T00:00:00Z', end_time: '2026-07-16T00:15:00Z', ingest_time: '2026-07-16T00:16:00Z' }],
        total: 1,
        truncated: true,
      },
    });

    const out = await pmDashboardApi.queryAggregated({
      granularity: '15min',
      technology: 'lte',
      metricPaths: ['K1'],
      limit: 1,
    });

    expect(getMock.mock.calls[0][1].params.technologies).toBe('lte');
    expect(out.total).toBe(1);
    expect(out.truncated).toBe(true);
  });
});
