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
  showLegend?: boolean;
}

// Line styles for differentiating multiple series
const LINE_STYLES: Array<'solid' | 'dashed' | 'dotted'> = ['solid', 'dashed', 'dotted'];
const SYMBOL_SHAPES: Array<'circle' | 'triangle' | 'diamond' | 'rect' | 'roundRect'> = ['circle', 'triangle', 'diamond', 'rect', 'roundRect'];

const LineChart: React.FC<LineChartProps> = ({
  title,
  xData,
  series,
  height = 280,
  areaFill = false,
  smooth = true,
  yAxisName,
  showLegend = true,
}) => {
  const isDark = useIsDark();
  const appTheme = useAppStore((s) => s.theme);
  const palette = getChartPalette(appTheme);

  const option = useMemo((): EChartsOption => {
    const base = getBaseOption(isDark, appTheme);
    const secondaryText = isDark ? '#8B949E' : '#8c8c8c';
    const titleColor = isDark ? '#E6EDF3' : '#262626';

    // Legend always at top with scroll enabled for multiple rows
    const getLegendConfig = () => {
      if (!showLegend) return { show: false };

      const baseLegend = base.legend as object;

      return {
        ...baseLegend,
        top: title ? 28 : 8,
        type: 'scroll' as const,
        pageIconSize: 10,
        pageTextStyle: { fontSize: 10 },
        // Allow multiple rows with scroll
        pageButtonItemGap: 2,
        pageButtonGap: 4,
      };
    };

    // Grid with fixed top to leave room for legend (supports ~2 rows of legend items)
    const getGridConfig = () => {
      const baseGrid = base.grid as object;

      return {
        ...baseGrid,
        top: title ? 80 : 64, // Fixed top to accommodate 2 rows of legend
        bottom: 24,
        left: 16,
        right: 16,
      };
    };

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
        confine: true,
        appendToBody: true,
        className: 'chart-tooltip',
        formatter: (params: unknown) => {
          const items = params as Array<{ marker: string; seriesName: string; value: unknown; axisValue: string }>;
          if (!Array.isArray(items) || items.length === 0) return '';
          const lines = items.map(item =>
            `${item.marker} ${item.seriesName}: <strong>${item.value}</strong>`
          );
          return `<div style="max-height: 200px; overflow-y: auto;">
            <div style="font-weight: 600; margin-bottom: 4px;">${items[0].axisValue}</div>
            ${lines.join('<br/>')}
          </div>`;
        },
      },
      legend: getLegendConfig(),
      grid: getGridConfig(),
      xAxis: {
        ...(base.xAxis as object),
        type: 'category',
        data: xData,
        boundaryGap: false,
        axisLabel: {
          rotate: xData.length > 10 ? 45 : 0,
          fontSize: 10,
          color: secondaryText,
        },
      },
      yAxis: {
        ...(base.yAxis as object),
        type: 'value',
        name: yAxisName,
        nameTextStyle: { color: secondaryText, fontSize: 12 },
        splitLine: {
          lineStyle: {
            type: 'dashed',
            opacity: 0.5,
          },
        },
      },
      series: series.map((s, i) => {
        const color = s.color ?? palette[i % palette.length];
        const lineStyleType = LINE_STYLES[Math.floor(i / SYMBOL_SHAPES.length) % LINE_STYLES.length];
        const symbolShape = SYMBOL_SHAPES[i % SYMBOL_SHAPES.length];

        return {
          name: s.name,
          type: 'line',
          data: s.data,
          smooth,
          symbol: symbolShape,
          symbolSize: 0,
          showSymbol: false,
          itemStyle: {
            color,
          },
          lineStyle: {
            width: 2,
            type: lineStyleType,
            color,
          },
          emphasis: {
            focus: 'series',
            lineStyle: { width: 3 },
            itemStyle: {
              shadowBlur: 8,
              shadowColor: isDark ? 'rgba(0,0,0,0.3)' : 'rgba(0,0,0,0.15)',
            },
          },
          areaStyle: areaFill
            ? {
                opacity: 0.1,
                color: {
                  type: 'linear',
                  x: 0,
                  y: 0,
                  x2: 0,
                  y2: 1,
                  colorStops: [
                    { offset: 0, color },
                    { offset: 1, color: 'rgba(255,255,255,0)' },
                  ],
                },
              }
            : undefined,
          animationDuration: 500,
          animationEasing: 'cubicOut',
        };
      }),
    };
  }, [title, xData, series, areaFill, smooth, yAxisName, isDark, appTheme, palette, showLegend]);

  return (
    <ReactECharts
      option={option}
      style={{ height, width: '100%' }}
      opts={{ renderer: 'canvas' }}
      notMerge={true}
    />
  );
};

export default LineChart;
