import { formatPmMetricDisplayValue } from '@core/utils/pmMetricValue';

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
  const tooltipFormatter = (params: unknown) => {
    const items = Array.isArray(params) ? params : [params];
    if (items.length === 0) return '';
    const first = items[0] as { axisValue?: string };
    const lines = items.map((item) => {
      const row = item as { marker?: string; seriesName?: string; value?: unknown };
      return `${row.marker ?? ''} ${row.seriesName ?? ''}: <strong>${formatPmMetricDisplayValue(row.value)}</strong>`;
    });
    return [`<div>${first.axisValue ?? ''}</div>`, ...lines].join('<br/>');
  };

  return {
    grid: { left: 50, right: 16, top: 58, bottom: 40 },
    xAxis: {
      type: 'category' as const,
      data: buckets,
      name: 'start_time',
      nameLocation: 'middle' as const,
      nameGap: 24,
    },
    yAxis: {
      type: 'value' as const,
      axisLabel: {
        formatter: (value: number) => formatPmMetricDisplayValue(value),
      },
    },
    series: series.map((s) => ({
      name: s.name,
      type: 'line' as const,
      smooth: true,
      showSymbol: true,
      symbolSize: 4,
      data: s.values,
      // 与首页 KPI 趋势一致：缺失值不补 0，但连接相邻有效点。
      connectNulls: true,
    })),
    tooltip: {
      trigger: 'axis' as const,
      formatter: tooltipFormatter,
    },
    legend: { type: 'scroll' as const, top: 0, left: 8, right: 8 },
  };
}
