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
 * 把一组指标的 current/compare 序列折算成 Day 模式（24 整点）的多条 line series。
 *
 * 选 1 个 → 输出 [today, yesterday]（昨日虚线，灰色对比）。
 * 选 ≥ 2 → 输出 N 条 today，按 palette 循环上色；不画对比线避免视觉爆炸（决策 D2）。
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
): { series: LineSeries[] } {
  if (metricKeys.length === 0) return { series: [] };

  const showCompare = metricKeys.length === 1;
  const out: LineSeries[] = [];

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

    // 选 1 个时图例文案保留原"今日 / 昨日"两词；选 ≥ 2 时图例直接用指标名。
    const todayName = showCompare ? todayLabel : meta.name;
    out.push({ name: todayName, data: todayValues, color });

    if (showCompare) {
      out.push({ name: compareLabel, data: yesterdayValues, color: KPI_COMPARE_LINE_COLOR, dashed: true });
    }
  });

  return { series: out };
}
