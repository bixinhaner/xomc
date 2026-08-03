import { describe, expect, it } from 'vitest';
import { pivotRowsToSheet } from '../pmPivotExport';
import type { AggregatedRow } from '../../types/pmDashboard';

function row(partial: Partial<AggregatedRow>): AggregatedRow {
  return {
    deviceOui: 'OUI',
    deviceSn: 'DEV-A',
    deviceGroupId: undefined,
    metricPath: 'K0001',
    metricType: 'kpi',
    metricValue: 0,
    statisType: 'avg',
    granularity: 'hourly',
    time: '2026-05-26T10:00:00Z',
    startTime: '2026-05-26T10:00:00Z',
    endTime: '2026-05-26T11:00:00Z',
    ingestTime: '2026-05-26T10:00:00Z',
    objectLdn: null,
    extra: {},
    ...partial,
  };
}

describe('pivotRowsToSheet', () => {
  it('exports PM metric values with fixed two decimals and empty placeholders', () => {
    const sheet = pivotRowsToSheet([
      row({
        metricPath: 'K0001',
        displayName: '接入成功率',
        metricValue: 12.345678901234,
      }),
      row({
        metricPath: 'K0002',
        displayName: '整数 KPI',
        metricValue: 12,
      }),
      row({
        metricPath: 'K0003',
        displayName: '空 KPI',
        metricValue: null,
      }),
    ]);

    expect(sheet).toHaveLength(1);
    expect(sheet[0]['接入成功率']).toBe('12.35');
    expect(sheet[0]['整数 KPI']).toBe('12.00');
    expect(sheet[0]['空 KPI']).toBe('-');
  });
});
