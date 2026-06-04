import React, { useMemo } from 'react';
import ReactECharts from 'echarts-for-react';
import type { EChartsOption } from 'echarts';
import { getBaseOption, getChartPalette } from './chartTheme';
import { useIsDark } from '@/hooks/useThemeToken';
import { useAppStore } from '@core/store/appStore';
import { calculateSmartTicks } from '@/utils/chartUtils';

export interface LineSeries {
  name: string;
  data: (number | null)[];
  color?: string;
  /** 为 true 时该线强制画虚线（lineStyle.type='dashed'），用于「上一周期」对比线。默认按索引走原样式。 */
  dashed?: boolean;
}

/**
 * 阈值线配置
 */
export interface ThresholdLine {
  value: number;
  label: string;
  color?: string;
  lineType?: 'solid' | 'dashed' | 'dotted';
}

export interface LineChartProps {
  title?: string;
  xData: string[];
  /** 完整时间戳数组（用于 tooltip 显示），格式：年-月-日 小时:分钟:秒 */
  xDataFull?: string[];
  series: LineSeries[];
  height?: number | string;
  areaFill?: boolean;
  smooth?: boolean;
  yAxisName?: string;
  unit?: string; // 单位，如 "Mbps", "%" 等
  showLegend?: boolean;
  /**
   * 周期对比 tooltip 补充行：与 xData 同长，每项为「上一周期真实起~止」文案。
   * 存在且当前桶项非空时，在 tooltip 当前时间行下方补一行「上一周期 …」；缺项不显示。
   * 不传 = 原行为不变（向后兼容）。
   */
  compareLabels?: (string | undefined)[];
  /**
   * 阈值线配置（如PRB利用率告警线）
   */
  thresholdLines?: ThresholdLine[];
}

// Line styles for differentiating multiple series
const LINE_STYLES: Array<'solid' | 'dashed' | 'dotted'> = ['solid', 'dashed', 'dotted'];
const SYMBOL_SHAPES: Array<'circle' | 'triangle' | 'diamond' | 'rect' | 'roundRect'> = ['circle', 'triangle', 'diamond', 'rect', 'roundRect'];

const LineChart: React.FC<LineChartProps> = ({
  title,
  xData,
  xDataFull,
  series,
  height = 280,
  areaFill = false,
  smooth = true,
  yAxisName,
  unit,
  showLegend = true,
  compareLabels,
  thresholdLines,
}) => {
  const isDark = useIsDark();
  const appTheme = useAppStore((s) => s.theme);
  const palette = getChartPalette(appTheme);

  const option = useMemo((): EChartsOption => {
    const base = getBaseOption(isDark, appTheme);
    const secondaryText = isDark ? '#8B949E' : '#8c8c8c';
    const titleColor = isDark ? '#E6EDF3' : '#262626';

    // 使用工具函数计算智能刻度
    const seriesData = series.map((s) => s.data.filter((v): v is number => v !== null));
    // 对于百分比数据，根据数据范围动态调整Y轴
    const { interval, max: calculatedMax } = calculateSmartTicks(seriesData);
    const isPercentage = unit === '%';
    // 对于百分比数据，当最大值小于10%时，使用动态计算的Y轴最大值，否则固定为100%
    const yMax = isPercentage && calculatedMax > 10 ? 100 : calculatedMax;

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
          const items = params as Array<{ marker: string; seriesName: string; value: unknown; axisValue: string; dataIndex: number }>;
          if (!Array.isArray(items) || items.length === 0) return '';

          // 数值格式化函数：处理null/undefined，显示"-"，否则保留2位小数
          const formatTooltipValue = (val: unknown): string => {
            if (val === null || val === undefined || Number.isNaN(val as number)) {
              return '-';
            }
            return (val as number).toFixed(2);
          };

          // 附加单位到数值后（如果单位不为空且不是"%"）
          const displayUnit = (unit && unit !== '%') ? `${unit}` : '';
          const unitSuffix = displayUnit ? ` ${displayUnit}` : '';

          const lines = items.map(item => {
            const formattedVal = formatTooltipValue(item.value);
            // 如果是空值显示"-"，则不附加单位
            const displayValue = formattedVal === '-' ? '-' : `${formattedVal}${unitSuffix}`;
            return `${item.marker} ${item.seriesName}: <strong>${displayValue}</strong>`;
          });

          // 周期对比：当前时间行下方补一行「上一周期 …」（缺项不显示）。
          const idx = items[0].dataIndex;
          const compareLabel = compareLabels?.[idx];
          const headerExtra = compareLabel
            ? `<div style="font-size: 11px; color: #8c8c8c; margin-bottom: 4px;">上一周期 ${compareLabel}</div>`
            : '';

          // 使用 xDataFull 显示完整时间戳，否则使用 axisValue
          const displayTime = xDataFull?.[idx] ?? items[0].axisValue;

          return `<div style="max-height: 200px; overflow-y: auto;">
            <div style="font-weight: 600; margin-bottom: 4px;">${displayTime}</div>
            ${headerExtra}
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
        min: 0,
        interval: interval,
        max: yMax,
        axisLabel: {
          formatter: (value: number) => Number.isInteger(value) ? value.toString() : '',
        },
        splitLine: {
          lineStyle: {
            type: 'dashed',
            opacity: 0.5,
          },
        },
      },
      series: series.map((s, i) => {
        const color = s.color ?? palette[i % palette.length];
        // dashed 显式优先：上一周期对比线强制虚线；否则按索引走原样式（向后兼容）。
        const lineStyleType = s.dashed
          ? 'dashed'
          : LINE_STYLES[Math.floor(i / SYMBOL_SHAPES.length) % LINE_STYLES.length];
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
          areaStyle: areaFill && !s.dashed
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
          markLine: thresholdLines && thresholdLines.length > 0
            ? {
                silent: true,
                symbol: 'none',
                data: thresholdLines.map((threshold) => ({
                  yAxis: threshold.value,
                  label: {
                    show: true,
                    position: 'end',
                    formatter: threshold.label,
                    fontSize: 10,
                    color: isDark ? '#8B949E' : '#8c8c8c',
                  },
                  lineStyle: {
                    type: threshold.lineType || 'dashed',
                    color: threshold.color || (isDark ? '#8B949E' : '#8c8c8c'),
                    width: 1,
                  },
                })),
              }
            : undefined,
          animationDuration: 500,
          animationEasing: 'cubicOut',
        };
      }),
    };
  }, [title, xData, xDataFull, series, areaFill, smooth, yAxisName, unit, isDark, appTheme, palette, showLegend, compareLabels, thresholdLines]);

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
