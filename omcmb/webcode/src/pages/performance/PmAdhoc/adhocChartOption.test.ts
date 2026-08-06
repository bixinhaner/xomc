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
    expect(option.series.every((item) => item.connectNulls === true)).toBe(true);
  });

  it('tooltip 和 y 轴数值固定保留两位，缺值显示占位符', () => {
    const option = buildAdhocChartOption(
      [
        { name: 'long-decimal', buckets: ['T1'], values: [12.345678901234] },
        { name: 'empty', buckets: ['T1'], values: ['-'] },
      ],
      ['T1'],
    );

    const tooltip = option.tooltip as {
      formatter: (params: Array<{ axisValue: string; marker: string; seriesName: string; value: unknown }>) => string;
    };
    expect(
      tooltip.formatter([
        { axisValue: 'T1', marker: '', seriesName: 'long-decimal', value: 12.345678901234 },
        { axisValue: 'T1', marker: '', seriesName: 'empty', value: '-' },
      ]),
    ).toContain('12.35');
    expect(
      tooltip.formatter([{ axisValue: 'T1', marker: '', seriesName: 'integer', value: 12 }]),
    ).toContain('12.00');
    expect(
      tooltip.formatter([{ axisValue: 'T1', marker: '', seriesName: 'empty', value: '-' }]),
    ).toContain('<strong>-</strong>');

    const yAxis = option.yAxis as { axisLabel: { formatter: (value: number) => string } };
    expect(yAxis.axisLabel.formatter(12.345678901234)).toBe('12.35');
    expect(yAxis.axisLabel.formatter(12)).toBe('12.00');
  });
});
