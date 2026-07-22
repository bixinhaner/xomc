import { describe, expect, it } from 'vitest';

import type { QueryTemplate } from '@core/types/pmQuery';
import { buildTemplateMetricExportFilename, buildTemplateMetricExportText } from './templateMetricExport';

const template: QueryTemplate = {
  id: 'tpl-1',
  name: 'eNB 基础/KPI 模板',
  visibility: 'private',
  creatorId: 'user-1',
  description: '',
  payload: {
    deviceSns: ['SN-1'],
    metricPaths: ['K0001', 'C0002'],
    granularity: 'hourly',
    timeRangePreset: 'last_1h',
    deviceType: 'ENB',
  },
  createdAt: '2026-07-20T00:00:00Z',
  updatedAt: '2026-07-20T00:00:00Z',
};

describe('template metric export', () => {
  it('exports metric IDs as one ID per line', () => {
    expect(buildTemplateMetricExportText(template)).toBe('K0001\nC0002');
  });

  it('uses a filesystem-safe file name', () => {
    expect(buildTemplateMetricExportFilename(template)).toBe('eNB_基础_KPI_模板_metrics.txt');
  });
});
