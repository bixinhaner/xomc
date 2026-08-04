/**
 * LayoutKPIPanel 的纯函数 helpers（Issue B：首页 KPI 折线图多指标对比）。
 *
 * 抽离原因：
 *  - 让 `LayoutKPIPanel.tsx` 文件只 export 组件，避免 fast-refresh 失效告警。
 *  - 把 buildSeries 这种纯函数和 palette 常量集中，方便单测和后续替换调色板。
 */

import type { MultiTrendComparisonData } from '@core/types/dashboard';
import type { DashboardKPIGranularity } from '@core/types/dashboard';
import type { LineSeries } from '@/components/Charts/LineChart';
import type { ResolvedMetricMeta } from './useMetricMetadata';
import { formatSystemDate, formatSystemTimeOnly } from '@core/utils/systemTime';

// 多指标对比调色板（决策 D4：按 idx 循环；与 PM 模块色板对齐感）。
export const KPI_METRIC_PALETTE: readonly string[] = [
  '#1677FF', // 蓝
  '#52C41A', // 绿
  '#FA8C16', // 橙
  '#722ED1', // 紫
  '#13C2C2', // 青
  '#EB2F96', // 玫红
  '#F5222D', // 红
  '#A0D911', // 黄绿
];

// 选 1 个时的"昨日对比线"颜色（与今日主色区分，灰色虚线）。
export const KPI_COMPARE_LINE_COLOR = '#999999';
export const KPI_WEEK_LINE_COLOR = '#1677FF';

export type KPIChartWindow = DashboardKPIGranularity | 'yesterday' | 'last_week';

export function shouldShowKPIChartLegend(
  compareWindow: KPIChartWindow,
  metricCount: number,
): boolean {
  return !['hourly', 'daily', 'weekly', 'last_week'].includes(compareWindow)
    || metricCount > 1;
}

function toWeekDateKey(time: string): string {
  return formatSystemDate(time, { placeholder: time.slice(0, 10) });
}

function toDirectBucketKey(time: string, granularity: DashboardKPIGranularity): string {
  const date = toWeekDateKey(time);
  if (granularity !== 'hourly') return date;
  return `${date}T${formatSystemTimeOnly(time).slice(0, 2)}`;
}

function pad2(value: number): string {
  return String(value).padStart(2, '0');
}

function addDaysToDateKey(dateKey: string, days: number): string {
  const parts = dateKey.split('-').map((value) => Number.parseInt(value, 10));
  if (parts.length !== 3 || parts.some((value) => Number.isNaN(value))) {
    return dateKey;
  }
  const [year, month, day] = parts;
  const next = new Date(Date.UTC(year, month - 1, day + days));
  return `${next.getUTCFullYear()}-${pad2(next.getUTCMonth() + 1)}-${pad2(next.getUTCDate())}`;
}

function addHoursToHourKey(hourKey: string, hours: number): string {
  const match = hourKey.match(/^(\d{4})-(\d{2})-(\d{2})T(\d{2})$/);
  if (!match) return hourKey;
  const [, year, month, day, hour] = match;
  const next = new Date(Date.UTC(
    Number.parseInt(year, 10),
    Number.parseInt(month, 10) - 1,
    Number.parseInt(day, 10),
    Number.parseInt(hour, 10) + hours,
  ));
  return `${next.getUTCFullYear()}-${pad2(next.getUTCMonth() + 1)}-${pad2(next.getUTCDate())}T${pad2(next.getUTCHours())}`;
}

function bucketEndKey(bucketKey: string, granularity: KPIChartWindow): string {
  if (granularity === 'hourly') return addHoursToHourKey(bucketKey, 1);
  if (granularity === 'weekly') return addDaysToDateKey(bucketKey, 7);
  return addDaysToDateKey(bucketKey, 1);
}

/**
 * 把一组指标的 current/compare 序列折算成多条 line series。
 *
 * yesterday 模式（默认）：映射到 24 整点时槽。
 * last_week 模式：按天聚合（均值），同时返回 weekXData/weekXDataFull 供调用方
 *   替换 x 轴（MM/DD 标签），保证时间轴语义正确。
 *
 * 天模式：单指标输出今日/昨日两线，多指标只输出各指标今日线。
 * 周模式：直接把后端 daily 点映射到固定七日轴，只输出一组周趋势，不再聚合 hourly
 * 或生成 compare 线。
 *
 * 选 1 个 → 输出 [todayLabel, compareLabel]（对比虚线，灰色）。
 * 选 ≥ 2 → 输出 N 条指标名，按 palette 循环上色；不画对比线（决策 D2）。
 * 选 0 个 → 空数组（调用方应显示占位）。
 */
export function buildSeries(
  metricKeys: string[],
  trendData: MultiTrendComparisonData | undefined,
  xData: string[],
  todayLabel: string,
  compareLabel: string,
  resolveMeta: (key: string) => ResolvedMetricMeta,
  palette: readonly string[] = KPI_METRIC_PALETTE,
  compareWindow: KPIChartWindow = 'yesterday',
  weekDateKeys: string[] = [],
): { series: LineSeries[]; weekXData?: string[]; weekXDataFull?: string[]; weekXDataEndFull?: string[] } {
  if (metricKeys.length === 0) return { series: [] };

  const showCompare = metricKeys.length === 1;
  const out: LineSeries[] = [];

  if (compareWindow === 'hourly' || compareWindow === 'daily' || compareWindow === 'weekly') {
    const axisLabels = weekDateKeys.map((key) => (
      compareWindow === 'hourly'
        ? `${key.slice(5, 10).replace('-', '/')} ${key.slice(11)}:00`
        : key.slice(5).replace('-', '/')
    ));
    metricKeys.forEach((metricKey, idx) => {
      const comparison = trendData?.[metricKey];
      const meta = resolveMeta(metricKey);
      const conv = meta.conversion || 1;
      const byBucket = new Map(
        (comparison?.current ?? []).map((point) => [
          toDirectBucketKey(point.time, compareWindow),
          point.value * conv,
        ]),
      );
      out.push({
        name: meta.name,
        data: weekDateKeys.map((key) => byBucket.get(key) ?? null),
        color: showCompare
          ? KPI_WEEK_LINE_COLOR
          : palette[idx % palette.length] ?? KPI_METRIC_PALETTE[0],
        unit: meta.unit,
      });
    });
    return {
      series: out,
      weekXData: axisLabels,
      weekXDataFull: weekDateKeys,
      weekXDataEndFull: weekDateKeys.map((key) => bucketEndKey(key, compareWindow)),
    };
  }

  if (compareWindow === 'last_week') {
    const weekXDataFull = weekDateKeys;
    const weekXDataEndFull = weekDateKeys.map((key) => bucketEndKey(key, compareWindow));
    const weekXData = weekDateKeys.map((d) => d.slice(5).replace('-', '/'));
    metricKeys.forEach((metricKey, idx) => {
      const comparison = trendData?.[metricKey];
      const meta = resolveMeta(metricKey);
      const conv = meta.conversion || 1;
      const byDate = new Map(
        (comparison?.current ?? []).map((point) => [
          toWeekDateKey(point.time),
          point.value * conv,
        ]),
      );
      const currentValues = weekDateKeys.map((date) => byDate.get(date) ?? null);
      const todayName = meta.name;
      const color = showCompare
        ? KPI_WEEK_LINE_COLOR
        : palette[idx % palette.length] ?? KPI_METRIC_PALETTE[0];
      out.push({ name: todayName, data: currentValues, color, unit: meta.unit });
    });

    return { series: out, weekXData, weekXDataFull, weekXDataEndFull };
  }

  // --- 昨日对比：按小时映射到 24 时槽（原有逻辑）---
  metricKeys.forEach((metricKey, idx) => {
    const comparison = trendData?.[metricKey];
    const meta = resolveMeta(metricKey);
    const color = palette[idx % palette.length] ?? KPI_METRIC_PALETTE[0];

    const todayValues = new Array<number | null>(xData.length).fill(null);
    const yesterdayValues = new Array<number | null>(xData.length).fill(null);

    const conv = meta.conversion || 1;

    if (comparison) {
      comparison.current.forEach((point) => {
        const hour = Number(formatSystemTimeOnly(point.time).slice(0, 2));
        if (hour >= 0 && hour <= 23) {
          todayValues[hour] = point.value * conv;
        }
      });
      if (showCompare) {
        comparison.compare.forEach((point) => {
          const hour = Number(formatSystemTimeOnly(point.time).slice(0, 2));
          if (hour >= 0 && hour <= 23) {
            yesterdayValues[hour] = point.value * conv;
          }
        });
      }
    }

    // 选 1 个时使用传入的 todayLabel（"今日"或"本周"）；多选时使用指标名（决策 D2）。
    const todayName = showCompare ? todayLabel : meta.name;
    out.push({ name: todayName, data: todayValues, color, unit: meta.unit });

    if (showCompare) {
      out.push({ name: compareLabel, data: yesterdayValues, color: KPI_COMPARE_LINE_COLOR, dashed: true, unit: meta.unit });
    }
  });

  return { series: out };
}
