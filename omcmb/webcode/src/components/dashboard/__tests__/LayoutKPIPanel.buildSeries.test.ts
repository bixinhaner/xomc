/**
 * buildSeries 单测（Issue B：首页 KPI 折线图多指标对比）。
 *
 * 关注点：
 *  - 选 0 个：返回空 series，调用方负责显示占位。
 *  - 选 1 个：主线用指标名，对比线用 compareLabel（"昨日"/"上周"）灰虚线。
 *  - 选 ≥ 2 个：只输出 N 条指标名，按 palette 循环上色，不画对比线（决策 D2）。
 *  - 颜色按 idx 循环，超出 palette 长度也能继续。
 *  - 数据按 conversion 系数缩放；缺失指标的曲线全为 null 但 series 仍占位。
 */

import { describe, it, expect } from 'vitest';
import {
  buildSeries,
  KPI_METRIC_PALETTE,
  KPI_WEEK_LINE_COLOR,
  shouldShowKPIChartLegend,
} from '../LayoutKPIPanel.helpers';
import type { MultiTrendComparisonData } from '../../../../../frontend-core/src/types/dashboard';
import type { ResolvedMetricMeta } from '../useMetricMetadata';

type SeriesValue = number | null;

const X_DATA = Array.from({ length: 24 }, (_, h) => `${h.toString().padStart(2, '0')}:00`);

// Helper：合成 24 整点的 `current` / `compare` 点列。
function makeSeries(values: number[]) {
  return X_DATA.map((_, idx) => ({
    time: `2026-06-24T${String(idx).padStart(2, '0')}:00:00+08:00`,
    value: values[idx] ?? 0,
  }));
}

const trendData: MultiTrendComparisonData = {
  K900010015: {
    current: makeSeries(Array.from({ length: 24 }, (_, i) => i + 1)),
    compare: makeSeries(Array.from({ length: 24 }, (_, i) => (i + 1) * 2)),
    metadata: { kpi_name: 'K900010015', compare_type: 'yesterday' },
  },
  K900010016: {
    current: makeSeries(Array.from({ length: 24 }, (_, i) => i * 10)),
    compare: makeSeries(Array.from({ length: 24 }, (_, i) => i * 5)),
    metadata: { kpi_name: 'K900010016', compare_type: 'yesterday' },
  },
};

// 简化 resolveMeta：给指标一个稳定 name + conversion=1。
const fakeResolveMeta = (key: string): ResolvedMetricMeta => ({
  name: `name:${key}`,
  unit: 'unit.gb',
  conversion: 1,
});

describe('buildSeries', () => {
  it('选 0 个：返回空 series', () => {
    const { series } = buildSeries([], trendData, X_DATA, '今日', '昨日', fakeResolveMeta);
    expect(series).toHaveLength(0);
  });

  it('选 1 个：输出 今日 + 昨日 两条线，昨日虚线灰色', () => {
    const { series } = buildSeries(['K900010015'], trendData, X_DATA, '今日', '昨日', fakeResolveMeta);
    expect(series).toHaveLength(2);
    // 单选时主线用 todayLabel（今日）。
    expect(series[0].name).toBe('今日');
    expect(series[0].dashed).toBeFalsy();
    expect(series[0].color).toBe(KPI_METRIC_PALETTE[0]);
    expect(series[1].name).toBe('昨日');
    expect(series[1].dashed).toBe(true);
    // 昨日色不在 palette 内（避免被误认为某指标）。
    expect(KPI_METRIC_PALETTE).not.toContain(series[1].color);
    // 数据非空。
    expect(series[0].data.filter((v: SeriesValue) => v !== null)).toHaveLength(24);
    expect(series[1].data.filter((v: SeriesValue) => v !== null)).toHaveLength(24);
  });

  it('选 2 个：只输出 today N 条，颜色按 palette 循环', () => {
    const { series } = buildSeries(
      ['K900010015', 'K900010016'],
      trendData,
      X_DATA,
      '今日',
      '昨日',
      fakeResolveMeta,
    );
    expect(series).toHaveLength(2);
    // 图例直接用指标名。
    expect(series[0].name).toBe('name:K900010015');
    expect(series[1].name).toBe('name:K900010016');
    expect(series[0].color).toBe(KPI_METRIC_PALETTE[0]);
    expect(series[1].color).toBe(KPI_METRIC_PALETTE[1]);
    // 不画对比线。
    expect(series.every((s) => !s.dashed)).toBe(true);
  });

  it('选 N 个超过 palette 长度时颜色循环', () => {
    const manyKeys = Array.from({ length: KPI_METRIC_PALETTE.length + 2 }, (_, i) => `K${i}`);
    const { series } = buildSeries(manyKeys, trendData, X_DATA, '今日', '昨日', fakeResolveMeta);
    expect(series).toHaveLength(manyKeys.length);
    // 第 N+1 条应回卷到 palette[0]。
    expect(series[KPI_METRIC_PALETTE.length].color).toBe(KPI_METRIC_PALETTE[0]);
    expect(series[KPI_METRIC_PALETTE.length + 1].color).toBe(KPI_METRIC_PALETTE[1]);
  });

  it('指标在 trendData 中缺失：series 占位、数据全为 null 不抛错', () => {
    const { series } = buildSeries(['K_MISSING'], trendData, X_DATA, '今日', '昨日', fakeResolveMeta);
    expect(series).toHaveLength(2);
    expect(series[0].data.every((v: SeriesValue) => v === null)).toBe(true);
    expect(series[1].data.every((v: SeriesValue) => v === null)).toBe(true);
  });

  it('应用 conversion 缩放：value * conversion 写入 data', () => {
    const scale = (key: string): ResolvedMetricMeta => ({
      name: `name:${key}`,
      unit: 'unit.gb',
      conversion: 1000,
    });
    const { series } = buildSeries(['K900010015'], trendData, X_DATA, '今日', '昨日', scale);
    // 第 1 个数据点原值 1，conversion=1000 → 1000。
    expect(series[0].data[0]).toBe(1000);
  });

  it('按后端时区钟面映射 hourly 桶，不受浏览器时区影响', () => {
    const zoned: MultiTrendComparisonData = {
      K900010015: {
        current: [{ time: '2026-07-13T08:00:00+09:00', value: 8 }],
        compare: [{ time: '2026-07-12T08:00:00+09:00', value: 7 }],
        metadata: { kpi_name: 'K900010015', compare_type: 'yesterday' },
      },
    };
    const { series } = buildSeries(
      ['K900010015'], zoned, X_DATA, '今日', '昨日', fakeResolveMeta,
    );
    expect(series[0].data[8]).toBe(8);
    expect(series[1].data[8]).toBe(7);
  });

  it('trendData 为 undefined 时：返回空数据但不报错', () => {
    const { series } = buildSeries(['K900010015'], undefined, X_DATA, '今日', '昨日', fakeResolveMeta);
    expect(series).toHaveLength(2);
    expect(series[0].data.every((v: SeriesValue) => v === null)).toBe(true);
    expect(series[1].data.every((v: SeriesValue) => v === null)).toBe(true);
  });
});

// --- rolling week 模式 ---

const WEEK_DATES = [
  '2026-07-06', '2026-07-07', '2026-07-08', '2026-07-09',
  '2026-07-10', '2026-07-11', '2026-07-12', '2026-07-13',
];

const weekTrendData: MultiTrendComparisonData = {
  K900010015: {
    current: [
      { time: '2026-07-06T00:00:00+08:00', value: 3.23 },
      { time: '2026-07-08T00:00:00+08:00', value: 4.56 },
      // 与 PM 相同：后端统一出口返回系统业务时区钟面与偏移。
      { time: '2026-07-12T00:00:00+08:00', value: 7.12 },
    ],
    compare: [],
    metadata: { kpi_name: 'K900010015', compare_type: 'last_week' },
  },
};

describe('buildSeries — rolling week 模式', () => {
  it('固定七个完整日加今天日期轴，今天没有 daily 数据时保留空槽', () => {
    const { series, weekXData, weekXDataFull, weekXDataEndFull } = buildSeries(
      ['K900010015'], weekTrendData, X_DATA, '周', 'unused', fakeResolveMeta,
      undefined, 'last_week', WEEK_DATES,
    );
    expect(weekXDataFull).toEqual(WEEK_DATES);
    expect(weekXDataEndFull).toEqual([
      '2026-07-07', '2026-07-08', '2026-07-09', '2026-07-10',
      '2026-07-11', '2026-07-12', '2026-07-13', '2026-07-14',
    ]);
    expect(weekXData).toEqual(['07/06', '07/07', '07/08', '07/09', '07/10', '07/11', '07/12', '07/13']);
    expect(series).toHaveLength(1);
    expect(series[0].name).toBe('name:K900010015');
    expect(series[0].color).toBe(KPI_WEEK_LINE_COLOR);
    expect(series[0].dashed).toBeFalsy();
    expect(series[0].data).toEqual([3.23, null, 4.56, null, null, null, 7.12, null]);
  });

  it('无数据时仍保留七个完整日加今天的日期轴和单一周线', () => {
    const { series, weekXDataFull } = buildSeries(
      ['K900010015'], undefined, X_DATA, '周', 'unused', fakeResolveMeta,
      undefined, 'last_week', WEEK_DATES,
    );
    expect(weekXDataFull).toEqual(WEEK_DATES);
    expect(series).toHaveLength(1);
    expect(series[0].data).toEqual(Array(8).fill(null));
  });

  it('双 KPI 分别保留 daily 点，缺失日期和今天保持空槽', () => {
    const dualTrendData: MultiTrendComparisonData = {
      K900010014: {
        current: [
          { time: '2026-07-06T00:00:00+08:00', value: 61 },
          { time: '2026-07-12T00:00:00+08:00', value: 67 },
        ],
        compare: [],
        metadata: { kpi_name: 'K900010014', compare_type: 'last_week' },
      },
      K900010013: {
        current: [
          { time: '2026-07-07T00:00:00+08:00', value: 31 },
          { time: '2026-07-11T00:00:00+08:00', value: 35 },
        ],
        compare: [],
        metadata: { kpi_name: 'K900010013', compare_type: 'last_week' },
      },
    };

    const { series } = buildSeries(
      ['K900010014', 'K900010013'], dualTrendData, X_DATA, '周', 'unused', fakeResolveMeta,
      undefined, 'last_week', WEEK_DATES,
    );

    expect(series).toHaveLength(2);
    expect(series[0].name).toBe('name:K900010014');
    expect(series[0].data).toEqual([61, null, null, null, null, null, 67, null]);
    expect(series[1].name).toBe('name:K900010013');
    expect(series[1].data).toEqual([null, 31, null, null, null, 35, null, null]);
    expect(series[0].color).not.toBe(series[1].color);
    expect(shouldShowKPIChartLegend('last_week', 2)).toBe(true);
  });

  it('单指标周模式隐藏冗余图例，多指标及天模式保留图例', () => {
    expect(shouldShowKPIChartLegend('last_week', 1)).toBe(false);
    expect(shouldShowKPIChartLegend('last_week', 2)).toBe(true);
    expect(shouldShowKPIChartLegend('yesterday', 1)).toBe(true);
  });
});

describe('buildSeries — direct rollup windows', () => {
  it('hourly 将 UTC 点位映射到固定的最近 24 个完整小时桶', () => {
    const hourlyTrendData: MultiTrendComparisonData = {
      K900010015: {
        current: [
          { time: '2026-07-28T04:00:00Z', value: 61 },
          { time: '2026-07-28T09:00:00Z', value: 66 },
        ],
        compare: [],
        metadata: { kpi_name: 'K900010015', compare_type: 'last_week' },
      },
    };
    const bucketKeys = [
      '2026-07-27T16', '2026-07-27T17', '2026-07-27T18', '2026-07-27T19',
      '2026-07-27T20', '2026-07-27T21', '2026-07-27T22', '2026-07-27T23',
      '2026-07-28T00', '2026-07-28T01', '2026-07-28T02', '2026-07-28T03',
      '2026-07-28T04', '2026-07-28T05', '2026-07-28T06', '2026-07-28T07',
      '2026-07-28T08', '2026-07-28T09', '2026-07-28T10', '2026-07-28T11',
      '2026-07-28T12', '2026-07-28T13', '2026-07-28T14', '2026-07-28T15',
    ];

    const direct = buildSeries(
      ['K900010015'],
      hourlyTrendData,
      [],
      'unused',
      'unused',
      fakeResolveMeta,
      undefined,
      'hourly',
      bucketKeys,
    );

    expect(direct.series[0].data[12]).toBe(61);
    expect(direct.series[0].data[17]).toBe(66);
    expect(direct.weekXDataEndFull?.[12]).toBe('2026-07-28T05');
    expect(direct.weekXDataEndFull?.[17]).toBe('2026-07-28T10');
  });

  it('小时、天、周都只展示后端对应粒度返回的当前序列', () => {
    const bucketKeys = ['2026-07-06', '2026-07-13'];
    const direct = buildSeries(
      ['K900010015'],
      weekTrendData,
      [],
      'unused',
      'unused',
      fakeResolveMeta,
      undefined,
      'weekly',
      bucketKeys,
    );

    expect(direct.series).toHaveLength(1);
    expect(direct.series[0].data).toEqual([3.23, null]);
    expect(direct.weekXDataFull).toEqual(bucketKeys);
    expect(direct.weekXDataEndFull).toEqual(['2026-07-13', '2026-07-20']);
    expect(shouldShowKPIChartLegend('weekly', 1)).toBe(false);
  });
});
