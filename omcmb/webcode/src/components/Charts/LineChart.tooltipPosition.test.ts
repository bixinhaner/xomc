import { describe, it, expect } from 'vitest';
import { buildLineChartOption, computeTooltipPosition } from './LineChart';

/**
 * issue #202：多设备性能图提示框遮挡曲线/图例/其它设备数据。
 * 提示框应放到光标所在象限的「对角」，并始终 clamp 在容器内。
 */
describe('computeTooltipPosition（提示框避让）', () => {
  const viewSize: [number, number] = [800, 400];
  const boxSize: [number, number] = [200, 160];
  const margin = 12;

  const size = (content: [number, number] = boxSize, view: [number, number] = viewSize) => ({
    contentSize: content,
    viewSize: view,
  });

  it('光标在左上 → 提示框落到右下（对角避让）', () => {
    const [x, y] = computeTooltipPosition([100, 50], size());
    // 右侧：viewW - boxW - margin
    expect(x).toBe(800 - 200 - margin);
    // 下方：viewH - boxH - margin
    expect(y).toBe(400 - 160 - margin);
  });

  it('光标在右下 → 提示框落到左上（对角避让）', () => {
    const [x, y] = computeTooltipPosition([700, 350], size());
    expect(x).toBe(margin);
    expect(y).toBe(margin);
  });

  it('光标在右上 → 提示框落到左下', () => {
    const [x, y] = computeTooltipPosition([700, 50], size());
    expect(x).toBe(margin);
    expect(y).toBe(400 - 160 - margin);
  });

  it('返回坐标恒在容器内（不越界 / 不为负）', () => {
    const [x, y] = computeTooltipPosition([700, 350], size());
    expect(x).toBeGreaterThanOrEqual(0);
    expect(y).toBeGreaterThanOrEqual(0);
    expect(x + boxSize[0]).toBeLessThanOrEqual(viewSize[0]);
    expect(y + boxSize[1]).toBeLessThanOrEqual(viewSize[1]);
  });

  it('提示框比容器还高时退化为顶对齐（不产生负坐标）', () => {
    // boxH(500) > viewH(400)：垂直无处避让，clamp 应回到 margin（顶对齐）而非负值。
    const [x, y] = computeTooltipPosition([100, 50], size([200, 500]));
    expect(y).toBe(margin);
    expect(x).toBeGreaterThanOrEqual(0);
  });
});

describe('buildLineChartOption PM metric value formatting', () => {
  const baseParams = {
    title: '',
    xData: ['T1'],
    series: [{ name: 'KPI', data: [12.345678901234] }],
    isDark: false,
    appTheme: 'classic' as const,
    palette: ['#1677ff'],
  };

  it('tooltip 数值固定保留两位并保留原有单位拼接规则', () => {
    const option = buildLineChartOption({
      ...baseParams,
      unit: 'Mbps',
      pmMetricValueFormat: true,
    });
    const formatter = (option.tooltip as {
      formatter: (params: Array<{ marker: string; seriesName: string; value: unknown; axisValue: string; dataIndex: number; seriesIndex: number }>) => string;
    }).formatter;

    expect(formatter([{ marker: '', seriesName: 'KPI', value: 12.345678901234, axisValue: 'T1', dataIndex: 0, seriesIndex: 0 }]))
      .toContain('12.35 Mbps');
    expect(formatter([{ marker: '', seriesName: 'KPI', value: null, axisValue: 'T1', dataIndex: 0, seriesIndex: 0 }]))
      .toContain('<strong>-</strong>');
  });

  it('PM y 轴刻度固定两位，整数图表仍按整数 tooltip 显示', () => {
    const option = buildLineChartOption({
      ...baseParams,
      pmMetricValueFormat: true,
    });
    const yAxis = option.yAxis as { axisLabel: { formatter: (value: number) => string } };
    expect(yAxis.axisLabel.formatter(12.345678901234)).toBe('12.35');
    expect(yAxis.axisLabel.formatter(12)).toBe('12.00');

    const integerOption = buildLineChartOption({
      ...baseParams,
      unit: 'count',
      integerValues: true,
    });
    const formatter = (integerOption.tooltip as {
      formatter: (params: Array<{ marker: string; seriesName: string; value: unknown; axisValue: string; dataIndex: number; seriesIndex: number }>) => string;
    }).formatter;
    expect(formatter([{ marker: '', seriesName: 'Count', value: 12.34, axisValue: 'T1', dataIndex: 0, seriesIndex: 0 }]))
      .toContain('12 count');
  });

  it('默认不开启 PM y 轴两位小数格式，避免影响通用折线图', () => {
    const option = buildLineChartOption(baseParams);
    const yAxis = option.yAxis as { axisLabel?: { formatter?: (value: number) => string } };

    expect(yAxis.axisLabel?.formatter).toBeUndefined();
  });
});
