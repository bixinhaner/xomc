/**
 * LayoutKPIPanel 的纯函数 helpers（Issue B：首页 KPI 折线图多指标对比）。
 *
 * 抽离原因：
 *  - 让 `LayoutKPIPanel.tsx` 文件只 export 组件，避免 fast-refresh 失效告警。
 *  - 把 buildSeries 这种纯函数和 palette 常量集中，方便单测和后续替换调色板。
 */

import type { MultiTrendComparisonData } from '@core/types/dashboard';
import type { LineSeries } from '@/components/Charts/LineChart';
import type { ResolvedMetricMeta } from './useMetricMetadata';

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

/**
 * 把时间序列按本地日期聚合（均值），返回 YYYY-MM-DD → 均值 的映射。
 *
 * 同一天内有多个小时点时取均值，适用于利用率/速率等比值指标；流量体积类指标
 * 理论上应用求和，但在 Dashboard 周趋势对比场景以均值呈现日均水位更直观，
 * 后续可按 MetricMeta.isCounter 区分（预留扩展点）。
 */
function aggregateToDays(
  points: ReadonlyArray<{ time: string; value: number }> | undefined,
  conv: number,
): Map<string, number> {
  const dayMap = new Map<string, { sum: number; count: number }>();
  (points ?? []).forEach((point) => {
    const d = new Date(point.time);
    const dateKey = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
    const prev = dayMap.get(dateKey) ?? { sum: 0, count: 0 };
    dayMap.set(dateKey, { sum: prev.sum + point.value * conv, count: prev.count + 1 });
  });
  const result = new Map<string, number>();
  dayMap.forEach((agg, date) => {
    result.set(date, agg.count > 0 ? agg.sum / agg.count : 0);
  });
  return result;
}

/**
 * 把一组指标的 current/compare 序列折算成多条 line series。
 *
 * yesterday 模式（默认）：映射到 24 整点时槽。
 * last_week 模式：按天聚合（均值），同时返回 weekXData/weekXDataFull 供调用方
 *   替换 x 轴（MM/DD 标签），保证时间轴语义正确。
 *
 * 主线图例：选 1 个时展示时间窗口（todayLabel，即"今日"或"本周"）；选 ≥ 2 时展示指标名。
 * 对比线仅用 compareLabel（"昨日"/"上周"）标注对比期，语义清晰不冗余。
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
  compareWindow: 'yesterday' | 'last_week' = 'yesterday',
): { series: LineSeries[]; weekXData?: string[]; weekXDataFull?: string[] } {
  if (metricKeys.length === 0) return { series: [] };

  const showCompare = metricKeys.length === 1;
  const out: LineSeries[] = [];

  if (compareWindow === 'last_week') {
    // --- 上周对比：按天聚合，x 轴为 MM/DD 日期标签 ---
    const allDatesSet = new Set<string>();
    const metricAggregations: Array<{
      currentDays: Map<string, number>;
      compareDays: Map<string, number>;
      meta: ResolvedMetricMeta;
      color: string;
    }> = [];

    metricKeys.forEach((metricKey, idx) => {
      const comparison = trendData?.[metricKey];
      const meta = resolveMeta(metricKey);
      const color = palette[idx % palette.length] ?? KPI_METRIC_PALETTE[0];
      const conv = meta.conversion || 1;

      const currentDays = aggregateToDays(comparison?.current, conv);
      const compareDays = aggregateToDays(comparison?.compare, conv);

      currentDays.forEach((_, date) => allDatesSet.add(date));
      compareDays.forEach((_, date) => allDatesSet.add(date));

      metricAggregations.push({ currentDays, compareDays, meta, color });
    });

    const sortedDates = Array.from(allDatesSet).sort();
    const weekXDataFull = sortedDates;
    const weekXData = sortedDates.map((d) => {
      const [, mm, dd] = d.split('-');
      return `${mm}/${dd}`;
    });

    metricAggregations.forEach(({ currentDays, compareDays, meta, color }) => {
      const currentValues: Array<number | null> = sortedDates.map((d) =>
        currentDays.has(d) ? currentDays.get(d)! : null,
      );
      const todayName = showCompare ? todayLabel : meta.name;
      out.push({ name: todayName, data: currentValues, color, unit: meta.unit });

      if (showCompare) {
        const compareValues: Array<number | null> = sortedDates.map((d) =>
          compareDays.has(d) ? compareDays.get(d)! : null,
        );
        out.push({ name: compareLabel, data: compareValues, color: KPI_COMPARE_LINE_COLOR, dashed: true, unit: meta.unit });
      }
    });

    return { series: out, weekXData, weekXDataFull };
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
        const hour = new Date(point.time).getHours();
        if (hour >= 0 && hour <= 23) {
          todayValues[hour] = point.value * conv;
        }
      });
      if (showCompare) {
        comparison.compare.forEach((point) => {
          const hour = new Date(point.time).getHours();
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
