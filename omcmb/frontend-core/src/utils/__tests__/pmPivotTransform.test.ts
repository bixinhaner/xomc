import { describe, expect, it } from 'vitest';
import { formatPivotNumber, pivotLongToWide, parseObjectLdn } from '../pmPivotTransform';
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
    expect(parseObjectLdn(null)).toEqual({ tech: '' });
    expect(parseObjectLdn(undefined)).toEqual({ tech: '' });
    expect(parseObjectLdn('')).toEqual({ tech: '' });
  });

  it('仅 Cellid', () => {
    expect(parseObjectLdn('Cellid=111172245')).toEqual({ tech: 'lte', cellId: '111172245' });
  });

  it('Cellid + PLMN', () => {
    expect(parseObjectLdn('Cellid=111172245,PLMN=46068')).toEqual({
      tech: 'lte',
      cellId: '111172245',
      plmn: '46068',
    });
  });

  it('顺序无关', () => {
    expect(parseObjectLdn('PLMN=00101,Cellid=222')).toEqual({
      tech: 'lte',
      cellId: '222',
      plmn: '00101',
    });
  });

  it('大小写不敏感', () => {
    expect(parseObjectLdn('CELLID=1,plmn=2')).toEqual({ tech: 'lte', cellId: '1', plmn: '2' });
  });

  it('未知键名忽略', () => {
    expect(parseObjectLdn('Cellid=1,Foo=bar')).toEqual({ tech: 'lte', cellId: '1' });
  });
});

describe('pivotLongToWide', () => {
  it('空输入', () => {
    const r = pivotLongToWide([]);
    expect(r.columns).toEqual([]);
    expect(r.rows).toEqual([]);
  });

  it('单设备单 LDN — 1 行 1 列', () => {
    const result = pivotLongToWide([
      row({ time: '2026-05-26T10:00:00Z', metricValue: 5, objectLdn: 'Cellid=111' }),
    ]);
    expect(result.columns).toHaveLength(1);
    expect(result.columns[0].key).toBe('PHY.NbrCqi6');
    expect(result.rows).toHaveLength(1);
    expect(result.rows[0]).toMatchObject({
      time: '2026-05-26T10:00:00Z',
      deviceSn: 'DEV-A',
      objectLdn: 'Cellid=111',
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
    expect(r.rows.map((x) => x.deviceSn).sort()).toEqual(['A', 'B']);
    expect(r.columns).toHaveLength(1); // 单指标
  });

  it('多 LDN → 行按 (time, sn, ldn) 拆开', () => {
    const r = pivotLongToWide([
      row({ objectLdn: 'Cellid=1', metricValue: 1 }),
      row({ objectLdn: 'Cellid=2', metricValue: 2 }),
    ]);
    expect(r.rows).toHaveLength(2);
    expect(r.rows.map((x) => x.objectLdn).sort()).toEqual(['Cellid=1', 'Cellid=2']);
  });

  it('PLMN 解析', () => {
    const r = pivotLongToWide([
      row({ objectLdn: 'Cellid=111,PLMN=46068', metricValue: 1 }),
    ]);
    expect(r.rows[0].objectLdn).toBe('Cellid=111,PLMN=46068');
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

  it('入库缺值行 metricValue=null 时保留行和设备维度，单元格显示占位符', () => {
    const r = pivotLongToWide([
      row({ metricPath: 'M1', metricValue: null, time: '2026-05-26T10:00:00Z', deviceSn: 'DEV-NULL' }),
    ]);
    expect(r.rows).toHaveLength(1);
    expect(r.rows[0].deviceSn).toBe('DEV-NULL');
    expect(r.rows[0].cells['M1']).toBeNull();
    expect(formatPivotNumber(r.rows[0].cells['M1'])).toBe('-');
  });

  it('后端返回已选 object 的 filled 空值骨架时保留 object 行并显示占位符', () => {
    const r = pivotLongToWide([
      row({
        metricPath: 'K900010052',
        displayName: '同频切换成功率-切出',
        metricValue: 12.3,
        objectLdn: 'Cellid=1,PLMN=46000',
      }),
      row({
        metricPath: 'K900010052',
        displayName: '同频切换成功率-切出',
        metricValue: null,
        filled: true,
        objectLdn: 'Cellid=2,PLMN=46000',
      }),
    ]);

    expect(r.rows).toHaveLength(2);
    const missing = r.rows.find((x) => x.objectLdn === 'Cellid=2,PLMN=46000')!;
    expect(missing.cells['K900010052']).toBeNull();
    expect(formatPivotNumber(missing.cells['K900010052'])).toBe('-');
  });

  it('非有限数按缺值处理，避免表格/导出出现 NaN', () => {
    const r = pivotLongToWide([
      row({ metricPath: 'M1', metricValue: Number.NaN }),
      row({ metricPath: 'M2', metricValue: Number.POSITIVE_INFINITY }),
    ]);
    expect(r.rows[0].cells['M1']).toBeNull();
    expect(r.rows[0].cells['M2']).toBeNull();
    expect(formatPivotNumber(Number.NaN)).toBe('-');
  });

  it('指标查询表格数值固定保留两位小数', () => {
    expect(formatPivotNumber(12.345678901234)).toBe('12.35');
    expect(formatPivotNumber(12)).toBe('12.00');
    expect(formatPivotNumber(null)).toBe('-');
    expect(formatPivotNumber(undefined)).toBe('-');
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
