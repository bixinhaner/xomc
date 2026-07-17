import { afterEach, describe, expect, it, vi } from 'vitest';
import { useAppStore } from '@core/store/appStore';

import { resolveTemplateMetricPaths } from './templateMetricResolver';

afterEach(() => {
  useAppStore.getState().setLocale('zh-CN');
});

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

  it('uses the English metric name when the current locale is English', async () => {
    useAppStore.getState().setLocale('en-US');
    const list = vi.fn().mockResolvedValue({
      items: [{ id: 'K900010043', enName: 'Cell Availability Rate', cnName: '小区可用率', isCounter: false }],
      total: 1,
    });

    const result = await resolveTemplateMetricPaths('ENB', ['K900010043'], list);

    expect(result.labels).toEqual({ K900010043: 'Cell Availability Rate' });
  });

  it('resolves a saved D-series indicator ID to its friendly label', async () => {
    const list = vi.fn().mockResolvedValue({
      items: [{ id: 'D000000001', enName: 'X2 Handover Success Rate In', cnName: 'eNB间X2切换成功率-切入', isCounter: false }],
      total: 1,
    });

    const result = await resolveTemplateMetricPaths('ENB', ['D000000001'], list);

    expect(result.paths).toEqual(['D000000001']);
    expect(result.labels).toEqual({ D000000001: 'eNB间X2切换成功率-切入' });
  });

  it('uses the English metric name for D-series indicator IDs when the current locale is English', async () => {
    useAppStore.getState().setLocale('en-US');
    const list = vi.fn().mockResolvedValue({
      items: [{ id: 'D000000001', enName: 'X2 Handover Success Rate In', cnName: 'eNB间X2切换成功率-切入', isCounter: false }],
      total: 1,
    });

    const result = await resolveTemplateMetricPaths('ENB', ['D000000001'], list);

    expect(result.labels).toEqual({ D000000001: 'X2 Handover Success Rate In' });
  });

  it('resolves saved metric IDs by exact ID match without depending on the ID prefix', async () => {
    const list = vi.fn().mockResolvedValue({
      items: [{ id: 'X-FUTURE-001', enName: 'Future Metric', cnName: '未来指标', isCounter: false }],
      total: 1,
    });

    const result = await resolveTemplateMetricPaths('ENB', ['X-FUTURE-001'], list);

    expect(result.paths).toEqual(['X-FUTURE-001']);
    expect(result.labels).toEqual({ 'X-FUTURE-001': '未来指标' });
  });

  it('uses the metric ID instead of Chinese name when English name is missing in English locale', async () => {
    useAppStore.getState().setLocale('en-US');
    const list = vi.fn().mockResolvedValue({
      items: [{ id: 'K900010043', enName: '', cnName: '小区可用率', isCounter: false }],
      total: 1,
    });

    const result = await resolveTemplateMetricPaths('ENB', ['K900010043'], list);

    expect(result.labels).toEqual({ K900010043: 'K900010043' });
  });

  it('maps a legacy Chinese KPI name to an English label and keeps the original key safe for detail rendering', async () => {
    useAppStore.getState().setLocale('en-US');
    const list = vi.fn().mockResolvedValue({
      items: [{ id: 'K900010043', enName: 'Cell Availability Rate', cnName: '小区可用率', isCounter: false }],
      total: 1,
    });

    const result = await resolveTemplateMetricPaths('ENB', ['小区可用率'], list);

    expect(result.paths).toEqual(['K900010043']);
    expect(result.labels).toEqual({
      K900010043: 'Cell Availability Rate',
      小区可用率: 'Cell Availability Rate',
    });
  });
});
