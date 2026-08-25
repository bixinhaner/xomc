import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
}));

vi.mock('../../http', () => ({
  default: { get: getMock },
}));

import { pmObjectsApi } from '../pmObjectsApi';

beforeEach(() => {
  getMock.mockReset();
});

describe('pmObjectsApi.listMetricObjects', () => {
  it('passes device, technology and optional time range to object discovery API', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [{ object_ldn: 'Cellid=66', cell_id: '66', plmn: '' }],
        total: 1,
      },
    });

    const out = await pmObjectsApi.listMetricObjects(
      ['SN-1'],
      'lte',
      {
        startTime: '2026-08-22T11:51:54Z',
        endTime: '2026-08-23T11:51:54Z',
      },
    );

    const [url, opts] = getMock.mock.calls[0];
    expect(url).toBe('/pm/metrics/objects');
    expect(opts.params).toMatchObject({
      device_sns: 'SN-1',
      technology: 'lte',
      start_time: '2026-08-22T11:51:54Z',
      end_time: '2026-08-23T11:51:54Z',
    });
    expect(out).toEqual([{ objectLdn: 'Cellid=66', cellId: '66', plmn: undefined }]);
  });
});
