import { describe, expect, it } from 'vitest';
import { pivotLongToWide } from '../pmPivotTransform';
import type { AggregatedRow } from '../../types/pmDashboard';

function row(partial: Partial<AggregatedRow>): AggregatedRow {
  return {
    deviceOui: 'OUI',
    deviceSn: 'DEV-A',
    deviceGroupId: undefined,
    metricPath: 'PHY.NbrCqi6',
    metricType: 'counter',
    metricValue: 0,
    statisType: 'sum',
    granularity: '15min',
    time: '2026-05-26T10:00:00Z',
    startTime: '2026-05-26T10:00:00Z',
    endTime: '2026-05-26T10:15:00Z',
    ingestTime: '2026-05-26T10:00:00Z',
    objectLdn: null,
    extra: {},
    ...partial,
  };
}

describe('pivotLongToWide', () => {
  it('空输入 → 空结果', () => {
    expect(pivotLongToWide([])).toEqual({ columns: [], rows: [] });
  });

  it('单设备单指标 → 单列', () => {
    const result = pivotLongToWide([
      row({ time: '2026-05-26T10:00:00Z', metricValue: 1 }),
      row({ time: '2026-05-26T10:15:00Z', metricValue: 2 }),
    ]);
    expect(result.columns).toHaveLength(1);
    expect(result.columns[0].key).toBe('PHY.NbrCqi6');
    expect(result.columns[0].title).toBe('PHY.NbrCqi6');
    expect(result.rows).toHaveLength(2);
    expect(result.rows[0].cells['PHY.NbrCqi6']).toBe(1);
    expect(result.rows[1].cells['PHY.NbrCqi6']).toBe(2);
  });

  it('多设备 → 列加 [SN] 后缀', () => {
    const result = pivotLongToWide([
      row({ deviceSn: 'DEV-A', metricValue: 1 }),
      row({ deviceSn: 'DEV-B', metricValue: 2 }),
    ]);
    expect(result.columns).toHaveLength(2);
    expect(result.columns.map((c) => c.title).sort()).toEqual([
      'PHY.NbrCqi6 [DEV-A]',
      'PHY.NbrCqi6 [DEV-B]',
    ]);
  });

  it('多 LDN → 列加 [LDN] 后缀', () => {
    const result = pivotLongToWide([
      row({ objectLdn: 'LDN-1', metricValue: 1 }),
      row({ objectLdn: 'LDN-2', metricValue: 2 }),
    ]);
    expect(result.columns).toHaveLength(2);
    expect(result.columns.map((c) => c.title).sort()).toEqual([
      'PHY.NbrCqi6 [LDN-1]',
      'PHY.NbrCqi6 [LDN-2]',
    ]);
  });

  it('多设备 + 多 LDN → 列加 [SN / LDN] 后缀', () => {
    const result = pivotLongToWide([
      row({ deviceSn: 'A', objectLdn: 'L1' }),
      row({ deviceSn: 'A', objectLdn: 'L2' }),
      row({ deviceSn: 'B', objectLdn: 'L1' }),
    ]);
    expect(result.columns).toHaveLength(3);
    expect(result.columns.map((c) => c.title).sort()).toEqual([
      'PHY.NbrCqi6 [A / L1]',
      'PHY.NbrCqi6 [A / L2]',
      'PHY.NbrCqi6 [B / L1]',
    ]);
  });

  it('缺采桶 cell = null', () => {
    const result = pivotLongToWide([
      row({ deviceSn: 'A', time: '2026-05-26T10:00:00Z', metricValue: 1 }),
      row({ deviceSn: 'B', time: '2026-05-26T10:15:00Z', metricValue: 2 }),
    ]);
    expect(result.rows).toHaveLength(2);
    // A 在 10:15 没数据 → null；B 在 10:00 没数据 → null
    const row0 = result.rows.find((r) => r.time === '2026-05-26T10:00:00Z')!;
    const row1 = result.rows.find((r) => r.time === '2026-05-26T10:15:00Z')!;
    expect(row0.cells['PHY.NbrCqi6||A']).toBe(1);
    expect(row0.cells['PHY.NbrCqi6||B']).toBeNull();
    expect(row1.cells['PHY.NbrCqi6||A']).toBeNull();
    expect(row1.cells['PHY.NbrCqi6||B']).toBe(2);
  });

  it('行按时间升序', () => {
    const result = pivotLongToWide([
      row({ time: '2026-05-26T10:30:00Z' }),
      row({ time: '2026-05-26T10:00:00Z' }),
      row({ time: '2026-05-26T10:15:00Z' }),
    ]);
    expect(result.rows.map((r) => r.time)).toEqual([
      '2026-05-26T10:00:00Z',
      '2026-05-26T10:15:00Z',
      '2026-05-26T10:30:00Z',
    ]);
  });

  it('多指标 → 多列', () => {
    const result = pivotLongToWide([
      row({ metricPath: 'PHY.NbrCqi6', metricValue: 1 }),
      row({ metricPath: 'PHY.NbrCqi7', metricValue: 2 }),
    ]);
    expect(result.columns).toHaveLength(2);
    expect(result.rows[0].cells['PHY.NbrCqi6']).toBe(1);
    expect(result.rows[0].cells['PHY.NbrCqi7']).toBe(2);
  });
});
