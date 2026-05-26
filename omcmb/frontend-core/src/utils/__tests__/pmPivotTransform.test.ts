import { describe, expect, it } from 'vitest';
import { pivotLongToWide, parseObjectLdn } from '../pmPivotTransform';
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

describe('parseObjectLdn', () => {
  it('空输入', () => {
    expect(parseObjectLdn(null)).toEqual({});
    expect(parseObjectLdn(undefined)).toEqual({});
    expect(parseObjectLdn('')).toEqual({});
  });

  it('仅 Cellid', () => {
    expect(parseObjectLdn('Cellid=111172245')).toEqual({ cellId: '111172245' });
  });

  it('Cellid + PLMN', () => {
    expect(parseObjectLdn('Cellid=111172245,PLMN=46068')).toEqual({
      cellId: '111172245',
      plmn: '46068',
    });
  });

  it('顺序无关', () => {
    expect(parseObjectLdn('PLMN=00101,Cellid=222')).toEqual({
      cellId: '222',
      plmn: '00101',
    });
  });

  it('大小写不敏感', () => {
    expect(parseObjectLdn('CELLID=1,plmn=2')).toEqual({ cellId: '1', plmn: '2' });
  });

  it('未知键名忽略', () => {
    expect(parseObjectLdn('Cellid=1,Foo=bar')).toEqual({ cellId: '1' });
  });
});

describe('pivotLongToWide', () => {
  it('空输入', () => {
    const r = pivotLongToWide([]);
    expect(r.columns).toEqual([]);
    expect(r.rows).toEqual([]);
    expect(r.hasMultiDevice).toBe(false);
    expect(r.hasMultiLdn).toBe(false);
    expect(r.hasPlmn).toBe(false);
  });

  it('单设备单 LDN — 1 行 1 列', () => {
    const result = pivotLongToWide([
      row({ time: '2026-05-26T10:00:00Z', metricValue: 5, objectLdn: 'Cellid=111' }),
    ]);
    expect(result.columns).toHaveLength(1);
    expect(result.columns[0].metricPath).toBe('PHY.NbrCqi6');
    expect(result.rows).toHaveLength(1);
    expect(result.rows[0]).toMatchObject({
      time: '2026-05-26T10:00:00Z',
      deviceSn: 'DEV-A',
      cellId: '111',
      plmn: undefined,
    });
    expect(result.rows[0].cells['PHY.NbrCqi6']).toBe(5);
  });

  it('单设备 + 2 指标 + 4 时间桶 → 4 行 × 2 列（倒序：最新在前）', () => {
    const inputs: AggregatedRow[] = [];
    ['10:00', '10:15', '10:30', '10:45'].forEach((m, i) => {
      ['M1', 'M2'].forEach((path) => {
        inputs.push(row({ time: `2026-05-26T${m}:00Z`, metricPath: path, metricValue: i + 1 }));
      });
    });
    const r = pivotLongToWide(inputs);
    expect(r.columns).toHaveLength(2);
    expect(r.rows).toHaveLength(4);
    expect(r.hasMultiDevice).toBe(false);
    // 倒序：最新时间 10:45 (i=3, value=4) 在第一行
    expect(r.rows[0].cells['M1']).toBe(4);
    expect(r.rows[0].cells['M2']).toBe(4);
    expect(r.rows[3].cells['M1']).toBe(1);
  });

  it('多设备 → 行按 (time, sn) 拆开', () => {
    const r = pivotLongToWide([
      row({ deviceSn: 'A', metricValue: 1, time: '2026-05-26T10:00:00Z' }),
      row({ deviceSn: 'B', metricValue: 2, time: '2026-05-26T10:00:00Z' }),
    ]);
    expect(r.rows).toHaveLength(2);
    expect(r.hasMultiDevice).toBe(true);
    expect(r.rows.map((x) => x.deviceSn).sort()).toEqual(['A', 'B']);
    expect(r.columns).toHaveLength(1); // 单指标
  });

  it('多 LDN → 行按 (time, sn, ldn) 拆开', () => {
    const r = pivotLongToWide([
      row({ objectLdn: 'Cellid=1', metricValue: 1 }),
      row({ objectLdn: 'Cellid=2', metricValue: 2 }),
    ]);
    expect(r.rows).toHaveLength(2);
    expect(r.hasMultiLdn).toBe(true);
    expect(r.rows.map((x) => x.cellId).sort()).toEqual(['1', '2']);
  });

  it('PLMN 解析', () => {
    const r = pivotLongToWide([
      row({ objectLdn: 'Cellid=111,PLMN=46068', metricValue: 1 }),
    ]);
    expect(r.hasPlmn).toBe(true);
    expect(r.rows[0].cellId).toBe('111');
    expect(r.rows[0].plmn).toBe('46068');
  });

  it('缺采桶 cell = null', () => {
    const r = pivotLongToWide([
      row({ metricPath: 'M1', metricValue: 1, time: '2026-05-26T10:00:00Z' }),
      row({ metricPath: 'M2', metricValue: 2, time: '2026-05-26T10:15:00Z' }),
    ]);
    expect(r.rows).toHaveLength(2);
    const t1 = r.rows.find((x) => x.time === '2026-05-26T10:00:00Z')!;
    const t2 = r.rows.find((x) => x.time === '2026-05-26T10:15:00Z')!;
    expect(t1.cells['M1']).toBe(1);
    expect(t1.cells['M2']).toBeNull();
    expect(t2.cells['M1']).toBeNull();
    expect(t2.cells['M2']).toBe(2);
  });

  it('行按时间倒序 → SN 升序 → LDN 升序（最新数据在前）', () => {
    const r = pivotLongToWide([
      row({ deviceSn: 'B', time: '2026-05-26T10:30:00Z' }),
      row({ deviceSn: 'A', time: '2026-05-26T10:00:00Z' }),
      row({ deviceSn: 'A', time: '2026-05-26T10:30:00Z' }),
      row({ deviceSn: 'B', time: '2026-05-26T10:00:00Z' }),
    ]);
    expect(r.rows.map((x) => `${x.time}|${x.deviceSn}`)).toEqual([
      '2026-05-26T10:30:00Z|A',
      '2026-05-26T10:30:00Z|B',
      '2026-05-26T10:00:00Z|A',
      '2026-05-26T10:00:00Z|B',
    ]);
  });

  it('PivotRow 携带 startTime / endTime', () => {
    const r = pivotLongToWide([
      row({
        time: '2026-05-26T10:00:00Z',
        startTime: '2026-05-26T10:00:00Z',
        endTime: '2026-05-26T10:15:00Z',
      }),
    ]);
    expect(r.rows[0].startTime).toBe('2026-05-26T10:00:00Z');
    expect(r.rows[0].endTime).toBe('2026-05-26T10:15:00Z');
  });
});
