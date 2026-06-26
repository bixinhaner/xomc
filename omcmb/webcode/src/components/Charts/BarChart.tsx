import React, { useMemo, useRef, useState, useLayoutEffect } from 'react';
import ReactECharts from 'echarts-for-react';
import type { EChartsOption } from 'echarts';
import type { CallbackDataParams } from 'echarts/types/dist/shared';
import { getBaseOption, getChartPalette } from './chartTheme';
import { useIsDark } from '@/hooks/useThemeToken';
import { useAppStore } from '@core/store/appStore';

export interface BarSeries {
  name: string;
  data: ((number | null) | { value: number; name: string })[];
  color?: string;
  stack?: string;
  /** 单独设置该系列的圆角，覆盖全局 borderRadius */
  borderRadius?: number | [number, number, number, number];
}

export interface BarChartProps {
  title?: string;
  xData: string[];
  series: BarSeries[];
  height?: number | string;
  horizontal?: boolean;
  yAxisName?: string;
  barWidth?: number | string;
  /** 圆角大小，默认 6，设为 0 则无圆角 */
  borderRadius?: number;
  /** 是否显示图例，默认 true */
  showLegend?: boolean;
  /** 自定义 tooltip formatter，兼容 ECharts 原生类型 */
  tooltipFormatter?: (params: CallbackDataParams | CallbackDataParams[]) => string;
  /** 点击事件回调 */
  onClick?: (index: number, name: string) => void;
  /** 最小柱高，处理数值极小时渲染展示点，只针对需要的图表显式传入即可 */
  minBarHeight?: number;
}

const BarChart: React.FC<BarChartProps> = ({
  title,
  xData,
  series,
  height = 280,
  horizontal = false,
  yAxisName,
  barWidth,
  borderRadius = 6,
  showLegend = true,
  tooltipFormatter,
  onClick,
  minBarHeight,
}) => {
  const isDark = useIsDark();
  const appTheme = useAppStore((s) => s.theme);
  const palette = getChartPalette(appTheme);
  const containerRef = useRef<HTMLDivElement>(null);
  const [computedHeight, setComputedHeight] = useState<number>(280);

  // 当 height 为 "100%" 时，计算实际像素高度
  useLayoutEffect(() => {
    if (height !== '100%' || !containerRef.current) {
      setComputedHeight(typeof height === 'number' ? height : 280);
      return;
    }

    const updateHeight = () => {
      if (containerRef.current) {
        const h = containerRef.current.clientHeight;
        if (h > 0) {
          setComputedHeight(h);
        }
      }
    };

    updateHeight();

    const resizeObserver = new ResizeObserver(() => {
      updateHeight();
    });

    resizeObserver.observe(containerRef.current);
    return () => resizeObserver.disconnect();
  }, [height]);

  const option = useMemo((): EChartsOption => {
    const base = getBaseOption(isDark, appTheme);
    const secondaryText = isDark ? '#8B949E' : '#8c8c8c';
    const titleColor = isDark ? '#E6EDF3' : '#262626';

    // 当 X 轴数据项超过 6 个时，启用自动间隔显示标签，防止重叠
    const shouldOptimizeLabels = !horizontal && xData.length > 6;
    const categoryAxis = {
      ...(base.xAxis as object),
      type: 'category',
      data: xData,
      axisLabel: {
        color: isDark ? '#C9D1D9' : '#595959',
        fontSize: 12,
        // 自动间隔显示标签，防止重叠
        interval: shouldOptimizeLabels ? 'auto' : 0,
        // 当标签非常多时，可选择旋转标签（可选，当前未启用）
        // rotate: shouldOptimizeLabels && xData.length > 10 ? 30 : 0,
      },
    };

    const valueAxis = {
      ...(base.yAxis as object),
      type: 'value',
      name: yAxisName,
      nameTextStyle: { color: secondaryText, fontSize: 12 },
      // 确保数值轴刻度为整数（适用于告警数量等离散值）
      minInterval: 1,
      axisLabel: {
        formatter: (value: number) => Number.isInteger(value) ? value.toString() : '',
      },
    };

    return {
      ...base,
      tooltip: tooltipFormatter
        ? {
            ...base.tooltip,
            formatter: tooltipFormatter,
          }
        : base.tooltip,
      title: title
        ? {
            text: title,
            textStyle: { fontSize: 14, fontWeight: 600, color: titleColor },
            left: 0,
            top: 4,
          }
        : undefined,
      legend: showLegend ? {
        ...(base.legend as object),
        top: title ? 28 : 8,
      } : undefined,
      grid: {
        ...(base.grid as object),
        top: title ? 56 : 40,
      },
      xAxis: (horizontal ? valueAxis : categoryAxis) as EChartsOption['xAxis'],
      yAxis: (horizontal ? categoryAxis : valueAxis) as EChartsOption['yAxis'],
      series: series.map((s, i) => {
        // 计算该系列最大值，用于后续判断是否为极小值
        const maxVal = Math.max(
          1, // 避免为0
          ...s.data.map(item => {
            if (typeof item === 'object' && item !== null) return item.value || 0;
            return (item as number) || 0;
          })
        );

        // 计算该系列的圆角：优先使用系列自身的 borderRadius，否则使用全局 borderRadius
        let seriesBorderRadius: number | [number, number, number, number];
        if (s.borderRadius !== undefined) {
          seriesBorderRadius = typeof s.borderRadius === 'number'
            ? (horizontal ? [0, s.borderRadius, s.borderRadius, 0] : [s.borderRadius, s.borderRadius, 0, 0])
            : s.borderRadius;
        } else {
          seriesBorderRadius = horizontal ? [0, borderRadius, borderRadius, 0] : [borderRadius, borderRadius, 0, 0];
        }

        return {
          name: s.name,
          type: 'bar',
          data: minBarHeight !== undefined ? s.data.map(item => {
            const val = typeof item === 'object' && item !== null ? item.value : (item || 0);
            const baseObj = typeof item === 'object' && item !== null ? item : { value: val };
            
            // 对于完全为 0 的柱子，将其变为完全透明，使得 minBarHeight 不会让其出现
            if (val === 0) {
              return {
                ...baseObj,
                itemStyle: { ...(baseObj as any).itemStyle, opacity: 0 },
              };
            }

            // 对于极小值（比如高度不足最大值的 2%），为了避免阴影和圆角导致它“悬浮”
            // 我们强制去掉圆角和底部的阴影，让它老老实实贴在 X 轴上
            if (val / maxVal <= 0.02) {
              return {
                ...baseObj,
                itemStyle: { 
                  ...(baseObj as any).itemStyle, 
                  borderRadius: 0, 
                  shadowBlur: 0, 
                  shadowOffsetY: 0, 
                  shadowColor: 'transparent' 
                },
              };
            }

            return item;
          }) : s.data,
          stack: s.stack,
          barWidth: barWidth ?? 'auto',
          barMinHeight: minBarHeight,
          itemStyle: {
            color: s.color ?? palette[i % palette.length],
            borderRadius: seriesBorderRadius,
            shadowBlur: isDark ? 10 : 6,
            shadowOffsetY: isDark ? 4 : 2,
            shadowColor: isDark ? 'rgba(0,0,0,0.25)' : 'rgba(0,0,0,0.08)',
          },
          emphasis: {
            itemStyle: {
              shadowBlur: 18,
              shadowOffsetY: 8,
              shadowColor: isDark ? 'rgba(0,0,0,0.35)' : 'rgba(0,0,0,0.15)',
            },
          },
        };
      }),
    };
  }, [title, xData, series, horizontal, yAxisName, barWidth, borderRadius, isDark, appTheme, palette, tooltipFormatter, minBarHeight]);

  return (
    <div ref={containerRef} style={{ height: typeof height === 'string' ? height : undefined, width: '100%', minHeight: 0 }}>
      <ReactECharts
        option={option}
        style={{ height: computedHeight, width: '100%' }}
        opts={{ renderer: 'canvas' }}
        onEvents={onClick ? {
          click: (params: any) => {
            onClick(params.dataIndex, params.name);
          }
        } : undefined}
      />
    </div>
  );
};

export default BarChart;
