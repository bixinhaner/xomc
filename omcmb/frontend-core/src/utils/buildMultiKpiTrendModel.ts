/**
 * #200 MultiKPITrendChart 数据装配纯函数。
 *
 * 旧逻辑（MultiKPITrendChart.tsx）：xData 只取「第一个有数据的 KPI」的时间点，
 * 其余 KPI 按数组下标 zip。多指标时间点稀疏/错位时会把不同时刻的值塞进同一列，
 * 曲线时间错位、相邻点连不成线、整图贴底平线。
 *
 * 本函数把所有 KPI 的时间点取并集（去重 + 升序）作为统一时间轴，每个 KPI 按
 * 时间值对齐到并集（缺失填 null），保证各 series 值数组长度严格 == xData 长度。
 * 下游 LineChart 配合 connectNulls=true 即可跨缺采点续连。
 *
 * 纯函数、无副作用、与 React/i18n 解耦（label 由调用方先翻译后传入），便于单测。
 */

import { buildUnionTimeAxis, type TimePoint } from './buildUnionTimeAxis';

export type MultiKpiTimeRange = 'yesterday' | 'last_week';

/** 单个 KPI 的展示配置 + 已翻译好的名称 */
export interface MultiKpiSeriesInput {
  /** 已翻译的 series 名称（调用方先做 i18n） */
  name: string;
  /** 该 KPI 的原始时间序列点（time 为 RFC3339，可稀疏/错位） */
  points: TimePoint[];
  color?: string;
}

/** 给 LineChart 用的 series（值数组已对齐到并集时间轴） */
export interface MultiKpiSeriesModel {
  name: string;
  data: Array<number | null>;
  color?: string;
}

export interface MultiKpiTrendModel {
  /** 展示用 x 轴标签（按 timeRange 格式化：last_week→MM/DD，否则→HH:mm） */
  xData: string[];
  /** 完整时间戳（原始 RFC3339 并集），供 tooltip 显示 */
  xDataFull: string[];
  /** 各 series 已对齐到并集时间轴的值数组 */
  series: MultiKpiSeriesModel[];
}

/** 按时间范围格式化并集时间点为展示标签 */
function formatAxisLabel(isoTime: string, timeRange: MultiKpiTimeRange): string {
  const date = new Date(isoTime);
  if (timeRange === 'last_week') {
    // 上周对比显示日期 (MM/DD)
    return `${(date.getMonth() + 1).toString().padStart(2, '0')}/${date
      .getDate()
      .toString()
      .padStart(2, '0')}`;
  }
  // 昨日对比显示时间 (HH:mm)
  return `${date.getHours().toString().padStart(2, '0')}:${date
    .getMinutes()
    .toString()
    .padStart(2, '0')}`;
}

/**
 * 把多 KPI 稀疏/错位时间序列装配为统一并集时间轴 + 对齐 series。
 *
 * @param inputs 多个 KPI（name 已翻译，points 原始时间序列）
 * @param timeRange 决定 x 轴标签格式（HH:mm / MM/DD）
 */
export function buildMultiKpiTrendModel(
  inputs: MultiKpiSeriesInput[],
  timeRange: MultiKpiTimeRange = 'yesterday',
): MultiKpiTrendModel {
  if (!inputs || inputs.length === 0) {
    return { xData: [], xDataFull: [], series: [] };
  }

  const { xData: unionTimes, values } = buildUnionTimeAxis(
    inputs.map((kpi) => kpi.points ?? []),
  );

  return {
    xData: unionTimes.map((t) => formatAxisLabel(t, timeRange)),
    xDataFull: unionTimes,
    series: inputs.map((kpi, i) => ({
      name: kpi.name,
      data: values[i],
      color: kpi.color,
    })),
  };
}
