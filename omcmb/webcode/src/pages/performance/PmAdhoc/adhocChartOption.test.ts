import { describe, expect, it } from 'vitest';
import { buildAdhocChartOption, type AdhocMetricSeries } from './adhocChartOption';

describe('buildAdhocChartOption', () => {
  it('多指标时使用滚动图例并保留全部系列（#35）', () => {
    const buckets = ['2026-07-10T00:00:00+08:00', '2026-07-11T00:00:00+08:00'];
    const series: AdhocMetricSeries[] = Array.from({ length: 20 }, (_, i) => ({
      name: `metric-${i + 1}`,
      buckets,
      values: [i + 1, i + 2],
    }));

    const option = buildAdhocChartOption(series, buckets);

    expect(option.legend).toMatchObject({ type: 'scroll', left: 8, right: 8 });
    expect(option.grid).toMatchObject({ top: 58 });
    expect(option.series).toHaveLength(20);
    expect(option.series.map((item) => item.name)).toEqual(series.map((item) => item.name));
  });
});
