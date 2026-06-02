import { describe, it, expect } from 'vitest';
import type { AggregatedRow } from '@core/types/pmDashboard';
import { buildKpiChartData, buildKpiCharts } from './kpiSeries';

/** 构造一行聚合数据的小工厂（只填测试关心的字段，其余给默认值）。 */
function row(partial: Partial<AggregatedRow> & { metricPath: string; time: string }): AggregatedRow {
  return {
    deviceSn: 'SN1',
    metricType: 'kpi',
    metricValue: null,
    statisType: 'avg',
    granularity: 'hourly',
    startTime: partial.time,
    endTime: partial.time,
    ingestTime: partial.time,
    objectLdn: null,
    extra: {},
    ...partial,
  };
}

describe('buildKpiChartData', () => {
  it('正常分组：按时间升序排列，X 轴去重，值对齐', () => {
    const rows: AggregatedRow[] = [
      row({ metricPath: 'K1', time: '2026-06-02T02:00:00Z', metricValue: 30 }),
      row({ metricPath: 'K1', time: '2026-06-02T00:00:00Z', metricValue: 10 }),
      row({ metricPath: 'K1', time: '2026-06-02T01:00:00Z', metricValue: 20 }),
      // 干扰：另一个指标，不应进入 K1 的结果
      row({ metricPath: 'K2', time: '2026-06-02T00:00:00Z', metricValue: 99 }),
    ];
    const out = buildKpiChartData(rows, 'K1', null, 'fallback');
    expect(out.xData).toEqual([
      '2026-06-02T00:00:00Z',
      '2026-06-02T01:00:00Z',
      '2026-06-02T02:00:00Z',
    ]);
    expect(out.values).toEqual([10, 20, 30]);
    expect(out.isEmpty).toBe(false);
  });

  it('displayName 优先，缺失时回退配置兜底名', () => {
    const withName = buildKpiChartData(
      [row({ metricPath: 'K1', time: 't1', metricValue: 1, displayName: 'RRC建立成功率' })],
      'K1',
      null,
      'fallback',
    );
    expect(withName.displayName).toBe('RRC建立成功率');

    const noName = buildKpiChartData(
      [row({ metricPath: 'K1', time: 't1', metricValue: 1 })],
      'K1',
      null,
      'fallback',
    );
    expect(noName.displayName).toBe('fallback');
  });

  it('设备级过滤：只取 objectLdn 为空的行，忽略小区/PLMN 行', () => {
    const rows: AggregatedRow[] = [
      row({ metricPath: 'K1', time: 't1', metricValue: 5, objectLdn: null }),
      row({ metricPath: 'K1', time: 't1', metricValue: 88, objectLdn: 'Cellid=111,PLMN=46068' }),
    ];
    const out = buildKpiChartData(rows, 'K1', null, 'f');
    expect(out.xData).toEqual(['t1']);
    expect(out.values).toEqual([5]);
  });

  it('指定 objectLdn 过滤：只画该对象曲线', () => {
    const rows: AggregatedRow[] = [
      row({ metricPath: 'K1', time: 't1', metricValue: 5, objectLdn: null }),
      row({ metricPath: 'K1', time: 't1', metricValue: 88, objectLdn: 'Cellid=111,PLMN=46068' }),
      row({ metricPath: 'K1', time: 't2', metricValue: 77, objectLdn: 'Cellid=111,PLMN=46068' }),
    ];
    const out = buildKpiChartData(rows, 'K1', 'Cellid=111,PLMN=46068', 'f');
    expect(out.xData).toEqual(['t1', 't2']);
    expect(out.values).toEqual([88, 77]);
  });

  it('null 占位：缺采桶为 null，不画点；只要有一个非空则 isEmpty=false', () => {
    const rows: AggregatedRow[] = [
      row({ metricPath: 'K1', time: 't1', metricValue: null, filled: true }),
      row({ metricPath: 'K1', time: 't2', metricValue: 42 }),
      row({ metricPath: 'K1', time: 't3', metricValue: null, filled: true }),
    ];
    const out = buildKpiChartData(rows, 'K1', null, 'f');
    expect(out.xData).toEqual(['t1', 't2', 't3']);
    expect(out.values).toEqual([null, 42, null]);
    expect(out.isEmpty).toBe(false);
  });

  it('空输入：返回空轴空值，isEmpty=true', () => {
    const out = buildKpiChartData([], 'K1', null, 'f');
    expect(out.xData).toEqual([]);
    expect(out.values).toEqual([]);
    expect(out.isEmpty).toBe(true);
    expect(out.displayName).toBe('f');
  });

  it('全 null 占位：有时间桶但全无值 → isEmpty=true', () => {
    const rows: AggregatedRow[] = [
      row({ metricPath: 'K1', time: 't1', metricValue: null, filled: true }),
      row({ metricPath: 'K1', time: 't2', metricValue: null, filled: true }),
    ];
    const out = buildKpiChartData(rows, 'K1', null, 'f');
    expect(out.isEmpty).toBe(true);
    expect(out.values).toEqual([null, null]);
  });
});

describe('buildKpiCharts', () => {
  it('按配置顺序产出多张图，未命中的指标给空图', () => {
    const rows: AggregatedRow[] = [
      row({ metricPath: 'K2', time: 't1', metricValue: 20 }),
      row({ metricPath: 'K1', time: 't1', metricValue: 10 }),
    ];
    const configs = [
      { key: 'K1', label: '指标1' },
      { key: 'K2', label: '指标2' },
      { key: 'K3', label: '指标3' },
    ];
    const charts = buildKpiCharts(rows, configs, null);
    expect(charts.map((c) => c.metricPath)).toEqual(['K1', 'K2', 'K3']);
    expect(charts[0].values).toEqual([10]);
    expect(charts[1].values).toEqual([20]);
    expect(charts[2].isEmpty).toBe(true);
    expect(charts[2].displayName).toBe('指标3');
  });

  it('空输入：每个配置都是空图', () => {
    const charts = buildKpiCharts([], [{ key: 'K1', label: 'a' }], null);
    expect(charts).toHaveLength(1);
    expect(charts[0].isEmpty).toBe(true);
  });
});
