import React, { useMemo } from 'react';
import ReactECharts from 'echarts-for-react';
import type { EChartsOption } from 'echarts';
import { getBaseOption, getChartPalette } from './chartTheme';
import { useIsDark } from '@/hooks/useThemeToken';
import { useAppStore } from '@/store/appStore';

export interface LineSeries {
  name: string;
  data: (number | null)[];
  color?: string;
}

export interface LineChartProps {
  title?: string;
  xData: string[];
  series: LineSeries[];
  height?: number | string;
  areaFill?: boolean;
  smooth?: boolean;
  yAxisName?: string;
}

const LineChart: React.FC<LineChartProps> = ({
  title,
  xData,
  series,
  height = 280,
  areaFill = false,
  smooth = true,
  yAxisName,
}) => {
  const isDark = useIsDark();
  const appTheme = useAppStore((s) => s.theme);
  const palette = getChartPalette(appTheme);

  const option = useMemo((): EChartsOption => {
    const base = getBaseOption(isDark, appTheme);
    const secondaryText = isDark ? '#8B949E' : '#8c8c8c';
    const titleColor = isDark ? '#E6EDF3' : '#262626';

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
      tooltip: {
        ...(base.tooltip as object),
        trigger: 'axis',
      },
      legend: {
        ...(base.legend as object),
        top: title ? 28 : 8,
      },
      grid: {
        ...(base.grid as object),
        top: title ? 56 : 40,
      },
      xAxis: {
        ...(base.xAxis as object),
        type: 'category',
        data: xData,
        boundaryGap: false,
      },
      yAxis: {
        ...(base.yAxis as object),
        type: 'value',
        name: yAxisName,
        nameTextStyle: { color: secondaryText, fontSize: 12 },
      },
      series: series.map((s, i) => ({
        name: s.name,
        type: 'line',
        data: s.data,
        smooth,
        symbol: 'circle',
        symbolSize: 4,
        itemStyle: {
          color: s.color ?? palette[i % palette.length],
        },
        lineStyle: {
          width: 2.5,
          color: s.color ?? palette[i % palette.length],
          shadowBlur: isDark ? 8 : 4,
          shadowOffsetY: isDark ? 4 : 2,
          shadowColor: isDark
            ? `${s.color ?? palette[i % palette.length]}40`
            : `${s.color ?? palette[i % palette.length]}20`,
        },
        emphasis: {
          lineStyle: { width: 3 },
          itemStyle: {
            shadowBlur: 12,
            shadowColor: isDark ? 'rgba(0,0,0,0.3)' : 'rgba(0,0,0,0.15)',
          },
        },
        areaStyle: areaFill
          ? {
              opacity: 0.15,
              color: {
                type: 'linear',
                x: 0,
                y: 0,
                x2: 0,
                y2: 1,
                colorStops: [
                  {
                    offset: 0,
                    color: s.color ?? palette[i % palette.length],
                  },
                  { offset: 1, color: 'rgba(255,255,255,0)' },
                ],
              },
            }
          : undefined,
      })),
    };
  }, [title, xData, series, areaFill, smooth, yAxisName, isDark, appTheme, palette]);

  return (
    <ReactECharts
      option={option}
      style={{ height, width: '100%' }}
      opts={{ renderer: 'canvas' }}
    />
  );
};

export default LineChart;
