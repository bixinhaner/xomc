export interface AdhocMetricSeries {
  name: string;
  buckets: string[];
  values: Array<number | '-'>;
}

/**
 * 自定义聚合结果页图表配置。
 *
 * #35：指标较多时普通 legend 会换行并覆盖绘图区。改用 ECharts scroll legend，
 * 左右保留翻页控制空间，并把 grid 顶部下移，确保全部指标仍在同一图表可切换查看。
 */
export function buildAdhocChartOption(series: AdhocMetricSeries[], buckets: string[]) {
  return {
    grid: { left: 50, right: 16, top: 58, bottom: 40 },
    xAxis: {
      type: 'category' as const,
      data: buckets,
      name: 'start_time',
      nameLocation: 'middle' as const,
      nameGap: 24,
    },
    yAxis: { type: 'value' as const },
    series: series.map((s) => ({
      name: s.name,
      type: 'line' as const,
      smooth: true,
      showSymbol: true,
      symbolSize: 4,
      data: s.values,
      connectNulls: false,
    })),
    tooltip: { trigger: 'axis' as const },
    legend: { type: 'scroll' as const, top: 0, left: 8, right: 8 },
  };
}
