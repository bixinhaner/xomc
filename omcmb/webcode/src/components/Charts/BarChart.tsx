import React, { useMemo, useRef, useState, useLayoutEffect } from 'react';
import ReactECharts from 'echarts-for-react';
import type { EChartsOption } from 'echarts';
import { getBaseOption, getChartPalette } from './chartTheme';
import { useIsDark } from '@/hooks/useThemeToken';
import { useAppStore } from '@core/store/appStore';

export interface BarSeries {
  name: string;
  data: (number | null)[];
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

    const categoryAxis = {
      ...(base.xAxis as object),
      type: 'category',
      data: xData,
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
      },
      xAxis: (horizontal ? valueAxis : categoryAxis) as EChartsOption['xAxis'],
      yAxis: (horizontal ? categoryAxis : valueAxis) as EChartsOption['yAxis'],
      series: series.map((s, i) => {
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
          data: s.data,
          stack: s.stack,
          barWidth: barWidth ?? 'auto',
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
  }, [title, xData, series, horizontal, yAxisName, barWidth, borderRadius, isDark, appTheme, palette]);

  return (
    <div ref={containerRef} style={{ height: typeof height === 'string' ? height : undefined, width: '100%', minHeight: 0 }}>
      <ReactECharts
        option={option}
        style={{ height: computedHeight, width: '100%' }}
        opts={{ renderer: 'canvas' }}
      />
    </div>
  );
};

export default BarChart;
