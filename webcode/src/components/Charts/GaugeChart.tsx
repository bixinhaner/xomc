import React, { useMemo } from 'react';
import ReactECharts from 'echarts-for-react';
import type { EChartsOption } from 'echarts';
import { getTooltipStyle } from './chartTheme';
import { useIsDark } from '@/hooks/useThemeToken';

export interface GaugeChartProps {
  title?: string;
  value: number;
  max?: number;
  height?: number | string;
  unit?: string;
  thresholds?: {
    value: number;
    color: string;
  }[];
  name?: string;
}

const DEFAULT_THRESHOLDS = [
  { value: 60, color: '#52C41A' },
  { value: 80, color: '#FAAD14' },
  { value: 100, color: '#F5222D' },
];

const GaugeChart: React.FC<GaugeChartProps> = ({
  title,
  value,
  max = 100,
  height = 280,
  unit = '%',
  thresholds = DEFAULT_THRESHOLDS,
  name,
}) => {
  const isDark = useIsDark();

  const option = useMemo((): EChartsOption => {
    const pct = Math.min((value / max) * 100, 100);
    const axisLineColors: [number, string][] = thresholds.map((t) => [
      t.value / 100,
      t.color,
    ]);

    const displayColor =
      thresholds.find((t) => pct <= t.value)?.color ?? thresholds[thresholds.length - 1].color;

    const titleColor = isDark ? '#E0E0E0' : '#262626';
    const secondaryText = isDark ? '#909090' : '#8c8c8c';
    const tickColor = isDark ? '#303030' : '#fff';

    return {
      title: title
        ? {
            text: title,
            textStyle: { fontSize: 14, fontWeight: 600, color: titleColor },
            left: 'center',
            top: 4,
          }
        : undefined,
      tooltip: {
        ...getTooltipStyle(isDark),
        formatter: `{b}<br/>{c}${unit}`,
      },
      series: [
        {
          type: 'gauge',
          name: name ?? title ?? '指标',
          startAngle: 225,
          endAngle: -45,
          min: 0,
          max: max,
          radius: '85%',
          center: ['50%', '55%'],
          axisLine: {
            lineStyle: {
              width: 18,
              color: axisLineColors,
            },
          },
          pointer: {
            length: '65%',
            width: 6,
            itemStyle: {
              color: displayColor,
            },
          },
          axisTick: {
            distance: -18,
            length: 6,
            lineStyle: { color: tickColor, width: 2 },
          },
          splitLine: {
            distance: -24,
            length: 12,
            lineStyle: { color: tickColor, width: 2 },
          },
          axisLabel: {
            color: secondaryText,
            fontSize: 11,
            distance: -36,
          },
          anchor: {
            show: true,
            size: 12,
            itemStyle: { color: displayColor },
          },
          detail: {
            valueAnimation: true,
            fontSize: 28,
            fontWeight: 700,
            color: displayColor,
            formatter: `{value}${unit}`,
            offsetCenter: [0, '30%'],
          },
          data: [{ value, name: name ?? '' }],
        },
      ],
    };
  }, [title, value, max, unit, thresholds, name, isDark]);

  return (
    <ReactECharts
      option={option}
      style={{ height, width: '100%' }}
      opts={{ renderer: 'canvas' }}
    />
  );
};

export default GaugeChart;
