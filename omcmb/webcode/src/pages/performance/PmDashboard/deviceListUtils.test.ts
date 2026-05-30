import { describe, it, expect } from 'vitest';
import type { AggregatedRow } from '@core/types/pmDashboard';
import { buildDeviceMetricCharts } from './deviceListUtils';

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
    endTime: '2026-05-30T00:15:00Z',
    ingestTime: '2026-05-30T00:16:00Z',
    objectLdn: null,
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

  it('空输入 → 空数组', () => {
    expect(buildDeviceMetricCharts([], '15min')).toEqual([]);
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
