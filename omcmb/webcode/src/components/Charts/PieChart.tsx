import React, { useMemo } from 'react';
import ReactECharts from 'echarts-for-react';
import type { EChartsOption } from 'echarts';
import { getBaseOption, getTooltipStyle, getChartPalette } from './chartTheme';
import { useIsDark } from '@/hooks/useThemeToken';
import { useAppStore } from '@core/store/appStore';

export interface PieDataItem {
  name: string;
  value: number;
  color?: string;
}

export interface PieChartProps {
  title?: string;
  data: PieDataItem[];
  height?: number | string;
  donut?: boolean;
  showLegend?: boolean;
  centerText?: string;
  onClick?: (data: PieDataItem) => void;
}

const PieChart: React.FC<PieChartProps> = ({
  title,
  data,
  height = 280,
  donut = false,
  showLegend = true,
  centerText,
  onClick,
}) => {
  const isDark = useIsDark();
  const appTheme = useAppStore((s) => s.theme);
  const palette = getChartPalette(appTheme);

  const option = useMemo((): EChartsOption => {
    const base = getBaseOption(isDark, appTheme);
    const titleColor = isDark ? '#E6EDF3' : '#262626';

    const formattedData = data.map((item, i) => ({
      name: item.name,
      value: item.value,
      itemStyle: {
        color: item.color ?? palette[i % palette.length],
      },
    }));

    const innerRadius = donut ? '50%' : '0%';
    const outerRadius = '80%';  // 增大饼图尺寸

    return {
      ...base,
      color: palette,
      title: title
        ? {
            text: title,
            textStyle: { fontSize: 14, fontWeight: 600, color: titleColor },
            left: 0,
            top: 4,
          }
        : undefined,
      tooltip: {
        trigger: 'item',
        ...getTooltipStyle(isDark),
        formatter: '{b}: {c} ({d}%)',
      },
      legend: showLegend
        ? {
            ...(base.legend as object),
            orient: 'vertical',
            right: 8,
            top: 'middle',
          }
        : { show: false },
      graphic: (donut && centerText
        ? [
            {
              type: 'text',
              left: 'center',
              top: 'center',
              style: {
                text: centerText,
                fill: titleColor,
                fontSize: 16,
                fontWeight: 600,
                textAlign: 'center',
              },
            },
          ]
        : undefined) as EChartsOption['graphic'],
      series: [
        {
          type: 'pie',
          radius: [innerRadius, outerRadius],
          center: showLegend ? ['40%', '50%'] : ['50%', '50%'],
          data: formattedData,
          // 环形图模式下完全不显示标签和引导线
          label: {
            show: false,
          },
          labelLine: {
            show: false,
          },
          itemStyle: {
            shadowBlur: isDark ? 12 : 8,
            shadowOffsetX: 0,
            shadowOffsetY: isDark ? 4 : 2,
            shadowColor: isDark ? 'rgba(0,0,0,0.3)' : 'rgba(0,0,0,0.1)',
            borderRadius: 4,
            borderWidth: 2,
            borderColor: isDark ? 'rgba(0,0,0,0.15)' : 'rgba(255,255,255,0.8)',
          },
          emphasis: {
            scaleSize: 8,
            itemStyle: {
              shadowBlur: 24,
              shadowOffsetX: 0,
              shadowOffsetY: 8,
              shadowColor: isDark ? 'rgba(0,0,0,0.4)' : 'rgba(0,0,0,0.2)',
            },
          },
        },
      ],
    };
  }, [title, data, donut, showLegend, centerText, isDark, appTheme, palette]);

  return (
    <ReactECharts
      option={option}
      style={{ height, width: '100%' }}
      opts={{ renderer: 'canvas' }}
      onEvents={onClick ? { click: (params: any) => {
        const item = data[params.dataIndex];
        if (item) onClick(item);
      } } : undefined}
    />
  );
};

export default PieChart;
