import { describe, it, expect } from 'vitest';
import type { AggregatedRow } from '@core/types/pmDashboard';
import { buildDeviceMetricCharts, filterRowsByObjectLdns } from './deviceListUtils';

// 构造聚合行的小工厂——只填测试关心的字段，其余给确定默认值。
function row(p: Partial<AggregatedRow>): AggregatedRow {
  return {
    deviceSn: p.deviceSn ?? 'SN-A',
    metricPath: p.metricPath ?? 'K900010015',
    displayName: p.displayName,
    metricType: 'kpi',
    metricValue: p.metricValue === undefined ? 1 : p.metricValue,
    statisType: 'avg',
    granularity: p.granularity ?? '15min',
    time: p.startTime ?? '2026-05-30T00:00:00Z',
    startTime: p.startTime ?? '2026-05-30T00:00:00Z',
    endTime: p.endTime ?? '2026-05-30T00:15:00Z',
    ingestTime: '2026-05-30T00:16:00Z',
    objectLdn: p.objectLdn ?? null,
    filled: p.filled,
  };
}

describe('buildDeviceMetricCharts — 设备级转置', () => {
  it('多设备 → 每指标一图、每图每设备一条线（系列名=SN）', () => {
    const rows: AggregatedRow[] = [
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: 'T1', metricValue: 10 }),
      row({ deviceSn: 'SN-B', metricPath: 'M1', startTime: 'T1', metricValue: 20 }),
      row({ deviceSn: 'SN-C', metricPath: 'M1', startTime: 'T1', metricValue: 30 }),
    ];
    const charts = buildDeviceMetricCharts(rows, '15min');
    expect(charts).toHaveLength(1);
    expect(charts[0].metricPath).toBe('M1');
    expect(charts[0].series).toHaveLength(3);
    expect(charts[0].series.map((s) => s.name)).toEqual(['SN-A', 'SN-B', 'SN-C']);
    expect(charts[0].series.map((s) => s.key)).toEqual(['SN-A', 'SN-B', 'SN-C']);
  });

  it('多指标 → 每指标一张图', () => {
    const rows: AggregatedRow[] = [
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: 'T1' }),
      row({ deviceSn: 'SN-A', metricPath: 'M2', startTime: 'T1' }),
    ];
    const charts = buildDeviceMetricCharts(rows, '15min');
    expect(charts.map((c) => c.metricPath)).toEqual(['M1', 'M2']);
  });

  it('缺桶 → 补 "-" 断线（设备 B 缺 T2）', () => {
    const rows: AggregatedRow[] = [
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: 'T1', metricValue: 1 }),
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: 'T2', metricValue: 2 }),
      row({ deviceSn: 'SN-B', metricPath: 'M1', startTime: 'T1', metricValue: 9 }),
      // SN-B 无 T2
    ];
    const charts = buildDeviceMetricCharts(rows, '15min');
    expect(charts[0].buckets).toEqual(['T1', 'T2']);
    const seriesB = charts[0].series.find((s) => s.key === 'SN-B')!;
    expect(seriesB.values).toEqual([9, '-']);
    const seriesA = charts[0].series.find((s) => s.key === 'SN-A')!;
    expect(seriesA.values).toEqual([1, 2]);
  });

  it('null 值 → 当缺桶补 "-" 断线（fill_empty 占位行）', () => {
    const rows: AggregatedRow[] = [
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: 'T1', metricValue: 5 }),
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: 'T2', metricValue: null, filled: true }),
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: 'T3', metricValue: 7 }),
    ];
    const charts = buildDeviceMetricCharts(rows, '15min');
    expect(charts[0].buckets).toEqual(['T1', 'T2', 'T3']);
    expect(charts[0].series[0].values).toEqual([5, '-', 7]);
  });

  it('真实站级 NULL 行不是 fill_empty，占位为设备线断点而不是整条记录消失', () => {
    const rows: AggregatedRow[] = [
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: 'T1', metricValue: null, filled: false, objectLdn: null }),
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: 'T2', metricValue: 7, objectLdn: null }),
    ];
    const charts = buildDeviceMetricCharts(rows, '15min');
    expect(charts[0].series).toHaveLength(1);
    expect(charts[0].series[0].key).toBe('SN-A');
    expect(charts[0].series[0].values).toEqual(['-', 7]);
  });

  it('NaN/Infinity 值按缺值断点处理', () => {
    const rows: AggregatedRow[] = [
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: 'T1', metricValue: Number.NaN }),
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: 'T2', metricValue: Number.POSITIVE_INFINITY }),
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: 'T3', metricValue: 9 }),
    ];
    const charts = buildDeviceMetricCharts(rows, '15min');
    expect(charts[0].series[0].values).toEqual(['-', '-', 9]);
  });

  it('空输入 → 空数组', () => {
    expect(buildDeviceMetricCharts([], '15min')).toEqual([]);
  });

  it('bucketEnds 与 buckets 一一对应记录每桶结束时间（含占位桶）', () => {
    const rows: AggregatedRow[] = [
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: 'T1', endTime: 'T1-end', metricValue: 1, objectLdn: 'Cellid=1' }),
      // 占位行：无小区 + null 值，提前 return，但桶 T2 仍须有结束时间
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: 'T2', endTime: 'T2-end', metricValue: null, filled: true, objectLdn: null }),
    ];
    const charts = buildDeviceMetricCharts(rows, '15min');
    expect(charts[0].buckets).toEqual(['T1', 'T2']);
    expect(charts[0].bucketEnds).toEqual(['T1-end', 'T2-end']);
  });

  it('粒度过滤 → 只取选中粒度的行', () => {
    const rows: AggregatedRow[] = [
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: 'T1', granularity: '15min', metricValue: 1 }),
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: 'T1', granularity: 'hourly', metricValue: 99 }),
    ];
    const charts = buildDeviceMetricCharts(rows, '15min');
    expect(charts).toHaveLength(1);
    expect(charts[0].series[0].values).toEqual([1]);
    // 选 hourly 则只取那行
    const hourly = buildDeviceMetricCharts(rows, 'hourly');
    expect(hourly[0].series[0].values).toEqual([99]);
  });

  it('桶按 startTime 升序对齐（乱序输入也排序）', () => {
    const rows: AggregatedRow[] = [
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: '2026-05-30T02:00:00Z', metricValue: 2 }),
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: '2026-05-30T00:00:00Z', metricValue: 0 }),
      row({ deviceSn: 'SN-A', metricPath: 'M1', startTime: '2026-05-30T01:00:00Z', metricValue: 1 }),
    ];
    const charts = buildDeviceMetricCharts(rows, '15min');
    expect(charts[0].buckets).toEqual([
      '2026-05-30T00:00:00Z',
      '2026-05-30T01:00:00Z',
      '2026-05-30T02:00:00Z',
    ]);
    expect(charts[0].series[0].values).toEqual([0, 1, 2]);
  });

  it('displayName 取行的 displayName，缺则回退 metricPath', () => {
    const rows: AggregatedRow[] = [
      row({ metricPath: 'K900010015', displayName: '上行流量', startTime: 'T1' }),
      row({ metricPath: 'C000060216', startTime: 'T1' }),
    ];
    const charts = buildDeviceMetricCharts(rows, '15min');
    expect(charts.find((c) => c.metricPath === 'K900010015')!.displayName).toBe('上行流量');
    expect(charts.find((c) => c.metricPath === 'C000060216')!.displayName).toBe('C000060216');
  });
});

describe('buildDeviceMetricCharts — T-0193 按设备+小区/PLMN 分线', () => {
  const LDN1 = 'Cellid=111172245,PLMN=46068';
  const LDN2 = 'Cellid=111172246,PLMN=46068';

  it('单设备多小区 → 每小区一条线，系列键含 objectLdn、名带友好名', () => {
    const rows: AggregatedRow[] = [
      row({ deviceSn: '1202000240194DP0015', metricPath: 'M1', startTime: 'T1', metricValue: 1, objectLdn: LDN1 }),
      row({ deviceSn: '1202000240194DP0015', metricPath: 'M1', startTime: 'T1', metricValue: 2, objectLdn: LDN2 }),
    ];
    const charts = buildDeviceMetricCharts(rows, '15min');
    expect(charts[0].series).toHaveLength(2);
    expect(charts[0].series.map((s) => s.key)).toEqual([
      `1202000240194DP0015|${LDN1}`,
      `1202000240194DP0015|${LDN2}`,
    ]);
    // 友好名：设备尾号（末6位）· 小区 · PLMN
    expect(charts[0].series[0].name).toBe('DP0015 · 小区111172245 · PLMN46068');
    expect(charts[0].series[1].name).toBe('DP0015 · 小区111172246 · PLMN46068');
  });

  it('5G CU/DU 同 NrCGI 同 PLMN → 系列名保留完整原始 objectLdn，避免 CUID/DUID 被压成重复名', () => {
    const cuLdn = 'Type=Cell,Mode=SA,gNBID=25,NrCGI=401,CUID=1,PLMNID=00101';
    const duLdn = 'Type=Cell,Mode=SA,gNBID=25,NrCGI=401,DUID=1,PLMNID=00101';
    const rows: AggregatedRow[] = [
      row({ deviceSn: '1202000240194DP0015', metricPath: 'M1', startTime: 'T1', metricValue: 1, objectLdn: cuLdn }),
      row({ deviceSn: '1202000240194DP0015', metricPath: 'M1', startTime: 'T1', metricValue: 2, objectLdn: duLdn }),
    ];

    const charts = buildDeviceMetricCharts(rows, '15min');

    expect(charts[0].series).toHaveLength(2);
    expect(charts[0].series[0].name).toBe(`DP0015 · ${cuLdn}`);
    expect(charts[0].series[1].name).toBe(`DP0015 · ${duLdn}`);
    expect(charts[0].series[0].name).not.toBe(charts[0].series[1].name);
    expect(charts[0].series[0].name).toContain('CUID=1');
    expect(charts[0].series[1].name).toContain('DUID=1');
  });

  it('多设备 × 多小区 → 设备×小区笛卡尔分线（4 条）', () => {
    const rows: AggregatedRow[] = [
      row({ deviceSn: 'SN-AAAAAA', metricPath: 'M1', startTime: 'T1', metricValue: 1, objectLdn: LDN1 }),
      row({ deviceSn: 'SN-AAAAAA', metricPath: 'M1', startTime: 'T1', metricValue: 2, objectLdn: LDN2 }),
      row({ deviceSn: 'SN-BBBBBB', metricPath: 'M1', startTime: 'T1', metricValue: 3, objectLdn: LDN1 }),
      row({ deviceSn: 'SN-BBBBBB', metricPath: 'M1', startTime: 'T1', metricValue: 4, objectLdn: LDN2 }),
    ];
    const charts = buildDeviceMetricCharts(rows, '15min');
    expect(charts[0].series).toHaveLength(4);
    expect(new Set(charts[0].series.map((s) => s.key))).toEqual(
      new Set([
        `SN-AAAAAA|${LDN1}`,
        `SN-AAAAAA|${LDN2}`,
        `SN-BBBBBB|${LDN1}`,
        `SN-BBBBBB|${LDN2}`,
      ]),
    );
  });

  it('缺 object_ldn 行 → 兜底退化为按设备单线（键仅 SN、名仅尾号）', () => {
    const rows: AggregatedRow[] = [
      row({ deviceSn: 'SN-AAAAAA', metricPath: 'M1', startTime: 'T1', metricValue: 1, objectLdn: null }),
      row({ deviceSn: 'SN-AAAAAA', metricPath: 'M1', startTime: 'T2', metricValue: 2, objectLdn: null }),
    ];
    const charts = buildDeviceMetricCharts(rows, '15min');
    expect(charts[0].series).toHaveLength(1);
    expect(charts[0].series[0].key).toBe('SN-AAAAAA');
    expect(charts[0].series[0].name).toBe('AAAAAA'); // 末6位
    expect(charts[0].series[0].values).toEqual([1, 2]);
  });

  it('同设备小区行 + 缺 object_ldn 行混合 → 小区行分线、缺行单独兜底线', () => {
    const rows: AggregatedRow[] = [
      row({ deviceSn: 'SN-AAAAAA', metricPath: 'M1', startTime: 'T1', metricValue: 1, objectLdn: LDN1 }),
      row({ deviceSn: 'SN-AAAAAA', metricPath: 'M1', startTime: 'T1', metricValue: 9, objectLdn: null }),
    ];
    const charts = buildDeviceMetricCharts(rows, '15min');
    expect(charts[0].series.map((s) => s.key)).toEqual([`SN-AAAAAA|${LDN1}`, 'SN-AAAAAA']);
  });

  it('纯 fill_empty 占位行（无小区 + null 值）→ 只对齐桶轴，不聚成「仅设备」空线', () => {
    const rows: AggregatedRow[] = [
      // 真实小区行只在 T1 有值；T2 只有无小区归属的 fill_empty 占位（null 值）
      row({ deviceSn: 'SN-AAAAAA', metricPath: 'M1', startTime: 'T1', metricValue: 1, objectLdn: LDN1 }),
      row({ deviceSn: 'SN-AAAAAA', metricPath: 'M1', startTime: 'T2', metricValue: null, filled: true, objectLdn: null }),
    ];
    const charts = buildDeviceMetricCharts(rows, '15min');
    // 只剩 1 条小区线（占位行不单独成线），但 T2 桶仍进桶集合用于对齐
    expect(charts[0].series).toHaveLength(1);
    expect(charts[0].series[0].key).toBe(`SN-AAAAAA|${LDN1}`);
    expect(charts[0].buckets).toEqual(['T1', 'T2']);
    expect(charts[0].series[0].values).toEqual([1, '-']);
  });
});

describe('filterRowsByObjectLdns — T-0193 即席小区过滤', () => {
  const LDN1 = 'Cellid=111172245,PLMN=46068';
  const LDN2 = 'Cellid=111172246,PLMN=46068';

  it('空白名单 → 不过滤，原样返回', () => {
    const rows: AggregatedRow[] = [
      row({ startTime: 'T1', objectLdn: LDN1 }),
      row({ startTime: 'T1', objectLdn: LDN2 }),
    ];
    expect(filterRowsByObjectLdns(rows, [])).toHaveLength(2);
  });

  it('非空白名单 → 只保留命中行', () => {
    const rows: AggregatedRow[] = [
      row({ startTime: 'T1', metricValue: 1, objectLdn: LDN1 }),
      row({ startTime: 'T1', metricValue: 2, objectLdn: LDN2 }),
    ];
    const out = filterRowsByObjectLdns(rows, [LDN1]);
    expect(out).toHaveLength(1);
    expect(out[0].objectLdn).toBe(LDN1);
  });

  it('过滤模式下无 object_ldn 的行被丢弃', () => {
    const rows: AggregatedRow[] = [
      row({ startTime: 'T1', objectLdn: LDN1 }),
      row({ startTime: 'T1', objectLdn: null }),
    ];
    const out = filterRowsByObjectLdns(rows, [LDN1]);
    expect(out).toHaveLength(1);
    expect(out[0].objectLdn).toBe(LDN1);
  });
});
