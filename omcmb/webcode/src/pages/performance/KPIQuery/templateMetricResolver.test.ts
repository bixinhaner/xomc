import { describe, expect, it, vi } from 'vitest';

import { resolveTemplateMetricPaths } from './templateMetricResolver';

describe('resolveTemplateMetricPaths', () => {
  it('resolves a normal saved KPI ID to its friendly label', async () => {
    const list = vi.fn().mockResolvedValue({
      items: [{ id: 'K900010043', enName: 'Cell Availability Rate', cnName: '小区可用率', isCounter: false }],
      total: 1,
    });

    const result = await resolveTemplateMetricPaths('ENB', ['K900010043'], list);

    expect(result.paths).toEqual(['K900010043']);
    expect(result.labels).toEqual({ K900010043: '小区可用率' });
  });
});
