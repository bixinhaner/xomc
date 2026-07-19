import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
}));

vi.mock('../../http', () => ({
  default: { get: getMock, post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}));

import { pmApi } from '../pmApi';

beforeEach(() => {
  getMock.mockReset();
});

describe('pmApi.getIndicatorCandidates', () => {
  it('透传后端 indicator_level，供聚合任务指标列表显示级别', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [
          {
            id: 'K3001',
            name: 'gsm_call_sr',
            display_name: 'GSM呼叫成功率',
            is_counter: '0',
            indicator_level: 'both',
          },
        ],
        total: 1,
      },
    });

    const out = await pmApi.getIndicatorCandidates('GSM', { includeCounters: true });

    expect(getMock).toHaveBeenCalledWith('/pm/kpi/definitions', {
      params: { device_type: 'GSM', include_counters: true },
    });
    expect(out).toEqual([
      expect.objectContaining({
        id: 'K3001',
        cnName: 'GSM呼叫成功率',
        indicatorLevel: 'both',
      }),
    ]);
  });
});
