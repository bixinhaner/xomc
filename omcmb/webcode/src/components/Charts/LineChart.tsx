/* eslint-disable react-refresh/only-export-components */
import React, { useMemo } from 'react';
import ReactECharts from 'echarts-for-react';
import type { EChartsOption } from 'echarts';
import { getBaseOption, getChartPalette } from './chartTheme';
import { useIsDark } from '@/hooks/useThemeToken';
import { useAppStore } from '@core/store/appStore';
import { formatPmMetricDisplayValue } from '@core/utils/pmMetricValue';

export interface LineSeries {
  name: string;
  data: (number | null)[];
  color?: string;
  /** 为 true 时该线强制画虚线（lineStyle.type='dashed'），用于「上一周期」对比线。默认按索引走原样式。 */
  dashed?: boolean;
  /**
   * 为 false 时该 series 不进 legend（但照画、在 tooltip 里仍可见）。
   * 默认 true。应用场景：上一周期虚线、阈值辅助线等“带、位但不需要 legend 项」的线。
   */
  showInLegend?: boolean;
  /**
   * 单位，如 "Mbps", "%" 等，渲染 tooltip 时会附加到数值后。
   */
  unit?: string;
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
  /** 每个 xDataFull 对应时间桶的结束时间；存在时 tooltip 显示开始/结束两行。 */
  xDataEndFull?: string[];
  formatTooltipStart?: (time: string) => string;
  formatTooltipEnd?: (time: string) => string;
  series: LineSeries[];
  height?: number | string;
  areaFill?: boolean;
  smooth?: boolean;
  yAxisName?: string;
  unit?: string; // 单位，如 "Mbps", "%" 等
  showLegend?: boolean;
  /**
   * 周期对比 tooltip 补充行：与 xData 同长，每项为「上一周期真实起~止」文案。
   * 存在且当前桶项非空时，在 tooltip 当前时间行下方补一行；缺项不显示。
   * 不传 = 原行为不变（向后兼容）。
   */
  compareLabels?: (string | undefined)[];
  formatCompareLabel?: (label: string) => string;
  /**
   * 阈值线配置（如PRB利用率告警线）
   */
  thresholdLines?: ThresholdLine[];
  /**
   * tooltip 数值是否显示为整数（不保留小数）
   * 适用于告警数量、设备数量等整数指标的图表
   */
  integerValues?: boolean;
  /**
   * PM/KPI 图表固定两位小数展示；用于 KPI 趋势图的 y 轴刻度。
   */
  pmMetricValueFormat?: boolean;
  /**
   * #200 是否让曲线跨 null 续连（connectNulls）。
   * 默认 false（保持原行为：遇 null 即断线，不掩盖真实数据空洞）。
   * 仅 KPI 趋势这类「多设备并集时间轴稀疏导致大量 null」的场景显式传 true，
   * 让稀疏打点连成可读曲线；普通图表不要开，避免把真缺采样连成假线。
   */
  connectNulls?: boolean;
}

/** buildLineChartOption 入参：把组件渲染态（主题/调色板）与纯数据一起喂进来 */
export interface BuildLineChartOptionParams {
  title?: string;
  xData: string[];
  xDataFull?: string[];
  xDataEndFull?: string[];
  formatTooltipStart?: (time: string) => string;
  formatTooltipEnd?: (time: string) => string;
  series: LineSeries[];
  areaFill?: boolean;
  smooth?: boolean;
  yAxisName?: string;
  unit?: string;
  showLegend?: boolean;
  compareLabels?: (string | undefined)[];
  formatCompareLabel?: (label: string) => string;
  thresholdLines?: ThresholdLine[];
  integerValues?: boolean;
  pmMetricValueFormat?: boolean;
  connectNulls?: boolean;
  isDark: boolean;
  appTheme: Parameters<typeof getBaseOption>[1];
  palette: string[];
}

/**
 * 纯函数：把数据 + 主题装配成 ECharts option。
 * 从 LineChart 的 useMemo 抽出，便于单测断言（如 #200 的 series[].connectNulls 与 xAxis.data 长度）。
 */
export function buildLineChartOption({
  title,
  xData,
  xDataFull,
  xDataEndFull,
  formatTooltipStart,
  formatTooltipEnd,
  series,
  areaFill = false,
  smooth = true,
  yAxisName,
  unit,
  showLegend = true,
  compareLabels,
  formatCompareLabel,
  thresholdLines,
  integerValues = false,
  pmMetricValueFormat = false,
  connectNulls = false,
  isDark,
  appTheme,
  palette,
}: BuildLineChartOptionParams): EChartsOption {
  const base = getBaseOption(isDark, appTheme);
  const baseYAxis = base.yAxis as { axisLabel?: object };
  const secondaryText = isDark ? '#8B949E' : '#8c8c8c';
  const titleColor = isDark ? '#E6EDF3' : '#262626';

  // 使用工具函数计算智能刻度
  const seriesData = series.map((s) => s.data.filter((v): v is number => v !== null));
  // 找出最大值
  const maxVal = Math.max(...(seriesData.flat().length ? seriesData.flat() : [0]));
  const isPercentage = unit === '%';
  // 对于百分比数据，当最大值小于10%时，使用动态计算的Y轴最大值，否则固定为100%
  let yMax: number | undefined = isPercentage && maxVal > 10 ? 100 : undefined;
  
  // 极小值或全0情况（如最大值不足1），使用 1 作为 max，并结合 minInterval 使得刻度仅为 0 和 1
  if (maxVal === 0) {
    yMax = 1;
  }

  // Legend always at top with scroll enabled for multiple rows
  const getLegendConfig = () => {
    if (!showLegend) return { show: false };

    const baseLegend = base.legend as object;

    // 只列出 showInLegend 非 false 的 series；全都不隐藏时不传 data（保持 echart 默认能收集全量 series）。
    const visibleNames = series.filter((s) => s.showInLegend !== false).map((s) => s.name);
    const hasHidden = visibleNames.length !== series.length;

    return {
      ...baseLegend,
      top: title ? 28 : 8,
      type: 'scroll' as const,
      pageIconSize: 10,
      pageTextStyle: { fontSize: 10 },
      // Allow multiple rows with scroll
      pageButtonItemGap: 2,
      pageButtonGap: 4,
      ...(hasHidden ? { data: visibleNames } : {}),
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
      // 智能避让：提示框放到光标对角，避免盖住曲线/图例/其它设备数据（issue #202）。
      position: (
        point: [number, number],
        _params: unknown,
        _dom: unknown,
        _rect: unknown,
        size: { contentSize: [number, number]; viewSize: [number, number] },
      ) => computeTooltipPosition(point, size),
      formatter: (params: unknown) => {
        const items = params as Array<{ marker: string; seriesName: string; value: unknown; axisValue: string; dataIndex: number; seriesIndex: number }>;
        if (!Array.isArray(items) || items.length === 0) return '';

        const formatTooltipValue = (val: unknown): string => {
          if (integerValues) {
            return typeof val === 'number' && Number.isFinite(val) ? val.toFixed(0) : '-';
          }
          if (pmMetricValueFormat) return formatPmMetricDisplayValue(val);
          if (val === null || val === undefined || Number.isNaN(val as number)) {
            return '-';
          }
          return (val as number).toFixed(2);
        };

        const lines = items.map(item => {
          const s = series[item.seriesIndex];
          const itemUnit = s?.unit ?? unit;
          const displayUnit = (itemUnit && itemUnit !== '%') ? `${itemUnit}` : '';
          const unitSuffix = displayUnit ? ` ${displayUnit}` : '';

          const formattedVal = formatTooltipValue(item.value);
          const displayValue = formattedVal === '-' ? '-' : `${formattedVal}${unitSuffix}`;
          return `${item.marker} ${item.seriesName}: <strong>${displayValue}</strong>`;
        });

        // 周期对比：当前时间行下方补一行（缺项不显示），业务前缀由调用方提供。
        const idx = items[0].dataIndex;
        const compareLabel = compareLabels?.[idx];
        const headerExtra = compareLabel
          ? `<div style="font-size: 11px; color: #8c8c8c; margin-bottom: 4px;">${formatCompareLabel?.(compareLabel) ?? compareLabel}</div>`
          : '';

        // 使用 xDataFull 显示完整时间戳，否则使用 axisValue。
        const displayTime = xDataFull?.[idx] ?? items[0].axisValue;
        const displayEndTime = xDataEndFull?.[idx];
        const timeHeader = displayEndTime && formatTooltipStart && formatTooltipEnd
          ? `<div style="font-weight: 600; margin-bottom: 4px;">
              <div>${formatTooltipStart(displayTime)}</div>
              <div>${formatTooltipEnd(displayEndTime)}</div>
            </div>`
          : `<div style="font-weight: 600; margin-bottom: 4px;">${displayTime}</div>`;

        return `<div style="max-height: 200px; overflow-y: auto;">
          ${timeHeader}
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
      max: yMax,
      minInterval: maxVal === 0 || integerValues ? 1 : undefined,
      ...(pmMetricValueFormat
        ? {
            axisLabel: {
              ...(baseYAxis.axisLabel ?? {}),
              formatter: (value: number) => formatPmMetricDisplayValue(value),
            },
          }
        : {}),
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
        // #200：稀疏并集时间轴下跨 null 续连（仅 connectNulls=true 时），避免曲线断成孤点。
        connectNulls,
        symbol: symbolShape,
        // issue #514：点标记始终可见。原 showSymbol:false / symbolSize:0 会让单个或
        // 极稀疏的孤立点（左右相邻槽位皆空、连不成线段）在画布上彻底隐身，被误判为"暂无数据"。
        // 改为始终画点（不引入 #437 的稀疏阴影/虚线，也不改 #429 的 connectNulls 断档语义）。
        symbolSize: 4,
        showSymbol: true,
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
}

// Line styles for differentiating multiple series
const LINE_STYLES: Array<'solid' | 'dashed' | 'dotted'> = ['solid', 'dashed', 'dotted'];
const SYMBOL_SHAPES: Array<'circle' | 'triangle' | 'diamond' | 'rect' | 'roundRect'> = ['circle', 'triangle', 'diamond', 'rect', 'roundRect'];

/**
 * tooltip 智能定位：把提示框放到鼠标所在象限的「对角」，避免盖住悬浮的数据点/曲线/图例。
 * 多设备图提示框可能很高（多行 + 内部滚动），固定跟随光标会遮挡其它设备曲线，故按光标位置左右/上下避让。
 * point = 鼠标坐标 [x, y]；size.viewSize = 图表容器尺寸；size.contentSize = 提示框尺寸。
 * 返回的坐标已自行 clamp 在容器内，配合 confine 不会溢出。
 */
export function computeTooltipPosition(
  point: [number, number],
  size: { contentSize: [number, number]; viewSize: [number, number] },
): [number, number] {
  const [pointerX, pointerY] = point;
  const [boxW, boxH] = size.contentSize;
  const [viewW, viewH] = size.viewSize;
  const margin = 12;

  // 水平：光标在左半区 → 提示框靠右；在右半区 → 靠左。让框始终落在光标对侧，不压住光标处的曲线。
  let x = pointerX < viewW / 2 ? viewW - boxW - margin : margin;
  // 垂直：光标在上半区 → 提示框沉到下方；在下半区 → 浮到上方。上方留出图例空间。
  let y = pointerY < viewH / 2 ? viewH - boxH - margin : margin;

  // clamp 进容器，避免负值或越界（小图表时 boxH 可能大于可用空间，优先顶对齐）。
  x = Math.max(margin, Math.min(x, Math.max(margin, viewW - boxW - margin)));
  y = Math.max(margin, Math.min(y, Math.max(margin, viewH - boxH - margin)));

  return [x, y];
}

const LineChart: React.FC<LineChartProps> = ({
  title,
  xData,
  xDataFull,
  xDataEndFull,
  formatTooltipStart,
  formatTooltipEnd,
  series,
  height = 280,
  areaFill = false,
  smooth = true,
  yAxisName,
  unit,
  showLegend = true,
  compareLabels,
  formatCompareLabel,
  thresholdLines,
  integerValues = false,
  pmMetricValueFormat = false,
  connectNulls = false,
}) => {
  const isDark = useIsDark();
  const appTheme = useAppStore((s) => s.theme);
  const palette = getChartPalette(appTheme);

  const option = useMemo(
    (): EChartsOption =>
      buildLineChartOption({
        title,
        xData,
        xDataFull,
        xDataEndFull,
        formatTooltipStart,
        formatTooltipEnd,
        series,
        areaFill,
        smooth,
        yAxisName,
        unit,
        showLegend,
        compareLabels,
        formatCompareLabel,
        thresholdLines,
        integerValues,
        pmMetricValueFormat,
        connectNulls,
        isDark,
        appTheme,
        palette,
      }),
    [
      title,
      xData,
      xDataFull,
      xDataEndFull,
      formatTooltipStart,
      formatTooltipEnd,
      series,
      areaFill,
      smooth,
      yAxisName,
      unit,
      showLegend,
      compareLabels,
      formatCompareLabel,
      thresholdLines,
      integerValues,
      pmMetricValueFormat,
      connectNulls,
      isDark,
      appTheme,
      palette,
    ],
  );

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
