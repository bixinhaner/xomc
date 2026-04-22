import React, { useMemo } from 'react';
import ReactECharts from 'echarts-for-react';
import type { EChartsOption } from 'echarts';
import { getBaseOption, getChartPalette } from './chartTheme';
import { useIsDark } from '@/hooks/useThemeToken';
import { useAppStore } from '@core/store/appStore';

export interface CombinedBarSeries {
  name: string;
  data: (number | null)[];
  color?: string;
  stack?: string;
  yAxisIndex?: 0 | 1;
}

export interface CombinedLineSeries {
  name: string;
  data: (number | null)[];
  color?: string;
  yAxisIndex?: 0 | 1;
  smooth?: boolean;
}

export interface CombinedChartProps {
  title?: string;
  xData: string[];
  barSeries: CombinedBarSeries[];
  lineSeries: CombinedLineSeries[];
  height?: number | string;
  yAxisNames?: [string?, string?];
}

const CombinedChart: React.FC<CombinedChartProps> = ({
  title,
  xData,
  barSeries,
  lineSeries,
  height = 320,
  yAxisNames = [],
}) => {
  const isDark = useIsDark();
  const appTheme = useAppStore((s) => s.theme);
  const palette = getChartPalette(appTheme);

  const option = useMemo((): EChartsOption => {
    const base = getBaseOption(isDark, appTheme);
    const secondaryText = isDark ? '#8B949E' : '#8c8c8c';
    const titleColor = isDark ? '#E6EDF3' : '#262626';
    const allSeries = barSeries.length + lineSeries.length;
    let colorIndex = 0;
    const nextColor = () => palette[colorIndex++ % palette.length];

    const seriesData = [
      ...barSeries.map((s) => ({
        name: s.name,
        type: 'bar' as const,
        data: s.data,
        stack: s.stack,
        yAxisIndex: s.yAxisIndex ?? 0,
        barWidth: allSeries > 3 ? undefined : '30%',
        itemStyle: {
          color: s.color ?? nextColor(),
          borderRadius: [3, 3, 0, 0],
        },
      })),
      ...lineSeries.map((s) => ({
        name: s.name,
        type: 'line' as const,
        data: s.data,
        yAxisIndex: s.yAxisIndex ?? 1,
        smooth: s.smooth ?? true,
        symbol: 'circle',
        symbolSize: 4,
        lineStyle: { width: 2, color: s.color ?? nextColor() },
        itemStyle: { color: s.color ?? palette[colorIndex % palette.length] },
      })),
    ];

    return {
      ...base,
      title: title
        ? {
            text: title,
            textStyle: { fontSize: 14, fontWeight: 600, color: titleColor },
            left: 0,
            top: 4,
          }
        : undefined,
      legend: {
        ...(base.legend as object),
        top: title ? 28 : 8,
      },
      grid: {
        ...(base.grid as object),
        top: title ? 56 : 40,
        right: 60,
      },
      xAxis: {
        ...(base.xAxis as object),
        type: 'category',
        data: xData,
      },
      yAxis: [
        {
          ...(base.yAxis as object),
          type: 'value',
          name: yAxisNames[0],
          nameTextStyle: { color: secondaryText, fontSize: 12 },
          position: 'left',
        },
        {
          ...(base.yAxis as object),
          type: 'value',
          name: yAxisNames[1],
          nameTextStyle: { color: secondaryText, fontSize: 12 },
          position: 'right',
          splitLine: { show: false },
        },
      ],
      series: seriesData,
    };
  }, [title, xData, barSeries, lineSeries, yAxisNames, isDark, appTheme, palette]);

  return (
    <ReactECharts
      option={option}
      style={{ height, width: '100%' }}
      opts={{ renderer: 'canvas' }}
    />
  );
};

export default CombinedChart;
