import { describe, expect, it } from 'vitest';
import { performanceKpiQueryKeys } from '@core/hooks/api/usePerformance';

describe('performanceKpiQueryKeys locale isolation', () => {
  it('KPI list, catalog and candidate keys change with locale', () => {
    const params = { page: 1, pageSize: 20, keyword: 'rate' };

    expect(performanceKpiQueryKeys.list(params, 'zh-CN')).not.toEqual(
      performanceKpiQueryKeys.list(params, 'en-US'),
    );
    expect(performanceKpiQueryKeys.all('zh-CN')).not.toEqual(
      performanceKpiQueryKeys.all('en-US'),
    );
    expect(performanceKpiQueryKeys.candidates('GSM', true, 'zh-CN')).not.toEqual(
      performanceKpiQueryKeys.candidates('GSM', true, 'en-US'),
    );
  });
});
