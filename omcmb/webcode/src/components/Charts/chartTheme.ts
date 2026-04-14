import type { EChartsOption } from 'echarts';
import type { Theme } from '@/types/common';

const CHART_PALETTES: Record<Theme, string[]> = {
  classic: ['#1677FF', '#52C41A', '#FA8C16', '#F5222D', '#722ED1', '#13C2C2', '#EB2F96', '#FAAD14'],
  tech:    ['#0071e3', '#2997ff', '#FF6B6B', '#FFD93D', '#6C5CE7', '#A29BFE', '#FD79A8', '#FDCB6E'],
  fresh:   ['#6366F1', '#22C55E', '#F59E0B', '#EF4444', '#8B5CF6', '#06B6D4', '#EC4899', '#F97316'],
  cyberpunk: ['#00F5FF', '#BF00FF', '#00FF88', '#FF3366', '#FFDD00', '#7B68EE', '#FF6B9D', '#00CED1'],
  minions: ['#FFD93D', '#4169E1', '#FF6B35', '#54A0FF', '#5ED3A3', '#FF9F43', '#EE5A6F', '#A29BFE'],
  tiffany: ['#81D8D0', '#B76E79', '#0ABAB5', '#D4A574', '#7FB3D5', '#F5B7B1', '#82E0AA', '#BB8FCE'],
  rmb: ['#E60012', '#D4AF37', '#8B0000', '#FFD700', '#B8860B', '#DC143C', '#FFA500', '#CD853F'],
};

export function getChartPalette(theme: Theme): string[] {
  return CHART_PALETTES[theme] ?? CHART_PALETTES.classic;
}

export const CHART_COLOR_PALETTE = CHART_PALETTES.classic;

export function getTooltipStyle(isDark: boolean) {
  return {
    backgroundColor: isDark ? 'rgba(22,27,34,0.96)' : 'rgba(255,255,255,0.96)',
    borderColor: isDark ? '#30363D' : '#e8e8e8',
    borderWidth: 1,
    textStyle: {
      color: isDark ? '#E6EDF3' : '#262626',
      fontSize: 13,
    },
    extraCssText: isDark
      ? 'box-shadow: 0 3px 5px 30px rgba(0,0,0,0.22); border-radius: 12px;'
      : 'box-shadow: 0 8px 24px rgba(0,0,0,0.12), 0 2px 8px rgba(0,0,0,0.06); border-radius: 8px; backdrop-filter: blur(8px);',
  };
}

/** Enhanced series item style with 3D depth shadows */
export function get3DItemStyle(color: string, isDark: boolean) {
  return {
    color,
    shadowBlur: isDark ? 12 : 8,
    shadowOffsetY: isDark ? 6 : 4,
    shadowColor: isDark ? 'rgba(0,0,0,0.3)' : 'rgba(0,0,0,0.1)',
    borderRadius: [6, 6, 0, 0],
  };
}

/** Enhanced emphasis style for hover interactions */
export function get3DEmphasisStyle() {
  return {
    itemStyle: {
      shadowBlur: 20,
      shadowOffsetY: 8,
      shadowColor: 'rgba(0,0,0,0.2)',
    },
    scale: true,
    scaleSize: 4,
  };
}

export function getBaseOption(isDark = false, theme: Theme = 'classic'): Partial<EChartsOption> {
  const palette = getChartPalette(theme);
  const primaryColor = palette[0];

  const textColor = isDark ? '#C9D1D9' : '#595959';
  const secondaryText = isDark ? '#8B949E' : '#8c8c8c';
  const axisLineColor = isDark ? '#30363D' : '#d9d9d9';
  const splitLineColor = isDark ? '#21262D' : '#f0f0f0';

  return {
    color: palette,
    tooltip: {
      trigger: 'axis' as const,
      ...getTooltipStyle(isDark),
      axisPointer: {
        type: 'cross' as const,
        lineStyle: {
          color: primaryColor,
          width: 1,
          type: 'dashed' as const,
        },
      },
    },
    legend: {
      show: true,
      textStyle: {
        color: textColor,
        fontSize: 12,
      },
      itemWidth: 12,
      itemHeight: 12,
    },
    grid: {
      top: 48,
      right: 16,
      bottom: 40,
      left: 16,
      containLabel: true,
    },
    textStyle: {
      fontFamily:
        "-apple-system, BlinkMacSystemFont, 'PingFang SC', 'Microsoft YaHei', sans-serif",
      color: textColor,
    },
    xAxis: {
      axisLine: {
        lineStyle: { color: axisLineColor },
      },
      axisTick: {
        lineStyle: { color: axisLineColor },
      },
      axisLabel: {
        color: secondaryText,
        fontSize: 12,
      },
      splitLine: {
        show: false,
      },
    },
    yAxis: {
      axisLine: {
        show: false,
      },
      axisTick: {
        show: false,
      },
      axisLabel: {
        color: secondaryText,
        fontSize: 12,
      },
      splitLine: {
        lineStyle: {
          color: splitLineColor,
          type: 'dashed' as const,
        },
      },
    },
  };
}
