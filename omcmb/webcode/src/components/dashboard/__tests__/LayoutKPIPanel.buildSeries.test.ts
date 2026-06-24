/**
 * buildSeries 单测（Issue B：首页 KPI 折线图多指标对比）。
 *
 * 关注点：
 *  - 选 0 个：返回空 series，调用方负责显示占位。
 *  - 选 1 个：输出 today + yesterday 两条线（昨日虚线），与旧单选行为兼容。
 *  - 选 ≥ 2 个：只输出 N 条 today，按 palette 循环上色，不画对比线（决策 D2）。
 *  - 颜色按 idx 循环，超出 palette 长度也能继续。
 *  - 数据按 conversion 系数缩放；缺失指标的曲线全为 null 但 series 仍占位。
 */

import { describe, it, expect } from 'vitest';
import { buildSeries, KPI_METRIC_PALETTE } from '../LayoutKPIPanel.helpers';
import type { MultiTrendComparisonData } from '../../../../../frontend-core/src/types/dashboard';
import type { ResolvedMetricMeta } from '../useMetricMetadata';

type SeriesValue = number | null;

const X_DATA = Array.from({ length: 24 }, (_, h) => `${h.toString().padStart(2, '0')}:00`);

// Helper：合成 24 整点的 `current` / `compare` 点列。
function makeSeries(values: number[]) {
  return X_DATA.map((_, idx) => ({
    time: new Date(2026, 5, 24, idx, 0, 0).toISOString(),
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

  it('选 1 个：输出 today + yesterday 两条线，昨日虚线灰色', () => {
    const { series } = buildSeries(['K900010015'], trendData, X_DATA, '今日', '昨日', fakeResolveMeta);
    expect(series).toHaveLength(2);
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
    // 多选时图例直接用指标名（不是"今日"）。
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

  it('trendData 为 undefined 时：返回空数据但不报错', () => {
    const { series } = buildSeries(['K900010015'], undefined, X_DATA, '今日', '昨日', fakeResolveMeta);
    expect(series).toHaveLength(2);
    expect(series[0].data.every((v: SeriesValue) => v === null)).toBe(true);
    expect(series[1].data.every((v: SeriesValue) => v === null)).toBe(true);
  });
});
