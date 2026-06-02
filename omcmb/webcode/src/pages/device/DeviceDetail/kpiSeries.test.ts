import { describe, it, expect } from 'vitest';
import type { AggregatedRow } from '@core/types/pmDashboard';
import {
  buildKpiChartData,
  buildKpiCharts,
  buildKpiCompareData,
  type KpiChartData,
} from './kpiSeries';

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

describe('buildKpiCompareData（周期对比对齐，T-0194 回归）', () => {
  /** 构造一张图（仅填对齐相关字段）。 */
  function chart(
    xData: string[],
    values: (number | null)[],
    xEnds?: string[],
  ): KpiChartData {
    return {
      metricPath: 'K1',
      xData,
      xEnds: xEnds ?? xData,
      values,
      displayName: 'd',
      isEmpty: values.every((v) => v == null),
    };
  }

  const HOUR = 3_600_000;
  const DAY = 86_400_000;

  it('hourly：偏移精确整数步长，上周期值平移落到当前整点桶，逐点对齐', () => {
    const current = chart(
      ['2026-06-02T00:00:00.000Z', '2026-06-02T01:00:00.000Z', '2026-06-02T02:00:00.000Z'],
      [10, 20, 30],
    );
    const prev = chart(
      ['2026-06-01T00:00:00.000Z', '2026-06-01T01:00:00.000Z', '2026-06-01T02:00:00.000Z'],
      [1, 2, 3],
    );
    // 当前窗口长 = 24h（无零头）。
    const out = buildKpiCompareData(current, prev, 24 * HOUR, 'hourly');
    expect(out.values).toEqual([1, 2, 3]);
    expect(out.isEmpty).toBe(false);
    expect(out.compareBuckets).toEqual([
      '2026-06-01T00:00:00.000Z',
      '2026-06-01T01:00:00.000Z',
      '2026-06-01T02:00:00.000Z',
    ]);
  });

  it('关键回归：偏移带毫秒/秒零头时，吸附到整数步长仍精确对齐（虚线非全空）', () => {
    const current = chart(
      ['2026-06-02T00:00:00.000Z', '2026-06-02T01:00:00.000Z'],
      [10, 20],
    );
    const prev = chart(
      ['2026-06-01T00:00:00.000Z', '2026-06-01T01:00:00.000Z'],
      [1, 2],
    );
    // 窗口长带 37.123 秒零头：24h + 37123ms。不吸附会整条 miss → 全 null。
    const out = buildKpiCompareData(current, prev, 24 * HOUR + 37_123, 'hourly');
    expect(out.values).toEqual([1, 2]);
    expect(out.isEmpty).toBe(false);
    // 证伪：若不吸附（直接用带零头偏移）则会全空 —— 这里断言非全空即守住回归。
    expect(out.values.some((v) => v != null)).toBe(true);
  });

  it('daily：偏移带零头按天步长吸附，逐日对齐', () => {
    const current = chart(
      ['2026-05-27T00:00:00.000Z', '2026-05-28T00:00:00.000Z'],
      [5, 6],
    );
    const prev = chart(
      ['2026-05-20T00:00:00.000Z', '2026-05-21T00:00:00.000Z'],
      [50, 60],
    );
    // 7 天窗口带 500ms 零头。
    const out = buildKpiCompareData(current, prev, 7 * DAY + 500, 'daily');
    expect(out.values).toEqual([50, 60]);
  });

  it('上周期无对应桶：该位置留空（null），起止文案空串', () => {
    const current = chart(
      ['2026-06-02T00:00:00.000Z', '2026-06-02T01:00:00.000Z', '2026-06-02T02:00:00.000Z'],
      [10, 20, 30],
    );
    // 上周期只有第 1、3 桶有数据，缺第 2 桶。
    const prev = chart(
      ['2026-06-01T00:00:00.000Z', '2026-06-01T02:00:00.000Z'],
      [1, 3],
    );
    const out = buildKpiCompareData(current, prev, 24 * HOUR, 'hourly');
    expect(out.values).toEqual([1, null, 3]);
    expect(out.compareBuckets[1]).toBe('');
  });

  it('上周期整体无数据：只产空对齐结果，isEmpty=true（调用方据此不挂虚线）', () => {
    const current = chart(
      ['2026-06-02T00:00:00.000Z', '2026-06-02T01:00:00.000Z'],
      [10, 20],
    );
    const prev = chart([], []);
    const out = buildKpiCompareData(current, prev, 24 * HOUR, 'hourly');
    expect(out.values).toEqual([null, null]);
    expect(out.isEmpty).toBe(true);
  });

  it('上周期值为 null（缺采占位）平移后仍为 null，不误判为有数据', () => {
    const current = chart(
      ['2026-06-02T00:00:00.000Z', '2026-06-02T01:00:00.000Z'],
      [10, 20],
    );
    const prev = chart(
      ['2026-06-01T00:00:00.000Z', '2026-06-01T01:00:00.000Z'],
      [null, 9],
    );
    const out = buildKpiCompareData(current, prev, 24 * HOUR, 'hourly');
    expect(out.values).toEqual([null, 9]);
    expect(out.isEmpty).toBe(false);
  });

  it('携带 xEnds：compareBucketEnds 取上周期桶真实结束时间', () => {
    const current = chart(
      ['2026-06-02T00:00:00.000Z'],
      [10],
      ['2026-06-02T01:00:00.000Z'],
    );
    const prev = chart(
      ['2026-06-01T00:00:00.000Z'],
      [1],
      ['2026-06-01T01:00:00.000Z'],
    );
    const out = buildKpiCompareData(current, prev, 24 * HOUR, 'hourly');
    expect(out.compareBuckets).toEqual(['2026-06-01T00:00:00.000Z']);
    expect(out.compareBucketEnds).toEqual(['2026-06-01T01:00:00.000Z']);
  });
});
