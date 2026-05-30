/**
 * T-0189 仪表盘三级筛选 + 周期对比 — 纯函数（与 React 解耦，便于单测）。
 *
 * 两页签（任务仪表盘 / 设备列表）共用：
 *   - filterRowsByWeekdayHour：星期 + 小时段在已取行里按本地时区筛命中点（不重聚合）。
 *   - previousWindow：当前大时间段 [start,end] → 上一周期 [start-L, start]（L=窗口长）。
 *   - buildCompareSeries：把上一周期系列按 +L 偏移对齐到当前轴，供 ChartCard 画虚线。
 *
 * 设计语义见 docs/design/pm-metric-aggregation-dashboard-redesign-20260529.md §3.2 / §3.4。
 */

import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { MetricChart, MetricSeries, MetricSeriesValue } from './taskDashboardUtils';

/** 全选用的常量集合（默认行为）。 */
export const ALL_WEEKDAYS: number[] = [0, 1, 2, 3, 4, 5, 6];
export const ALL_HOURS: number[] = Array.from({ length: 24 }, (_, i) => i);

/** 带 startTime 的行（AdhocResultRow / AggregatedRow 都满足）。 */
interface RowWithStartTime {
  startTime: string;
}

/**
 * 星期 + 小时段过滤：保留某行当且仅当 startTime 的本地星期 ∈ weekdays 且本地小时 ∈ hours。
 * 全选（两集合都覆盖全集）= 不过滤，直接返回原数组（默认行为不变，不破坏已验收出图）。
 * 星期口径 0=周日..6=周六（dayjs().day()），小时 0..23（dayjs().hour()），均本地时区
 * （与 ChartCard 横轴 dayjs(b).format('MM-DD HH:mm') 显示口径一致）。
 * 非法 startTime 行直接丢弃（不抛错）。
 */
export function filterRowsByWeekdayHour<T extends RowWithStartTime>(
  rows: T[],
  weekdays: Set<number>,
  hours: Set<number>,
): T[] {
  const allWeekdays = weekdays.size >= 7;
  const allHours = hours.size >= 24;
  if (allWeekdays && allHours) return rows;
  return rows.filter((r) => {
    const d = dayjs(r.startTime);
    if (!d.isValid()) return false;
    if (!allWeekdays && !weekdays.has(d.day())) return false;
    if (!allHours && !hours.has(d.hour())) return false;
    return true;
  });
}

/**
 * 上一周期窗口：当前 [start,end]，长度 L=end-start，上一周期=[start-L, start]。
 * 长度守恒（上一周期窗口长 === 当前窗口长）。
 */
export function previousWindow(range: [Dayjs, Dayjs]): [Dayjs, Dayjs] {
  const [start, end] = range;
  const lengthMs = end.valueOf() - start.valueOf();
  return [start.subtract(lengthMs, 'millisecond'), start];
}

/**
 * 对比序列对齐：把上一周期系列（在 prevBuckets 上的值）按 +offsetMs 偏移落到当前轴 currentBuckets。
 * 对齐规则：上一周期桶 T_prev 的值落在当前轴位置 T_prev + offsetMs（offsetMs 通常 = L = 窗口长）。
 *   即当前桶 Tc 的对比值 = 上一周期在 Tc - offsetMs 的值。
 * 当前桶在上一周期无对应点 → '-'（断线，不连）。
 * 空数据（currentBuckets 空 / prevSeries 空）不抛错，返回对应空结果。
 *
 * @param currentBuckets 当前周期横轴桶（startTime 升序字符串）
 * @param prevSeries     上一周期系列（同 buildMetricCharts/buildDeviceMetricCharts 转置产出）
 * @param prevBuckets    上一周期横轴桶（与 prevSeries.values 一一对齐）
 * @param offsetMs       偏移毫秒（= 当前窗口长 L）
 */
export function buildCompareSeries(
  currentBuckets: string[],
  prevSeries: MetricSeries[],
  prevBuckets: string[],
  offsetMs: number,
): MetricSeries[] {
  // 当前桶 -> 索引，用「Tc - offsetMs」反查上一周期值。
  // 上一周期桶值映射：shiftedTimestamp(ms) -> value，按系列建。
  return prevSeries.map((s) => {
    // shiftedMs -> value（上一周期 T_prev 偏移 +offsetMs 后的时间戳）
    const shifted = new Map<number, MetricSeriesValue>();
    prevBuckets.forEach((b, i) => {
      const ts = dayjs(b);
      if (!ts.isValid()) return;
      const v = s.values[i];
      if (v === undefined) return;
      shifted.set(ts.valueOf() + offsetMs, v);
    });
    const values: MetricSeriesValue[] = currentBuckets.map((cb) => {
      const ct = dayjs(cb);
      if (!ct.isValid()) return '-';
      const v = shifted.get(ct.valueOf());
      return v === undefined ? '-' : v;
    });
    return { key: `${s.key}__prev`, name: `${s.name}（上一周期）`, values };
  });
}

/**
 * 把上一周期图集挂到当前图集：按 metricPath 匹配图，按 +offsetMs 对齐上一周期系列。
 * 当前图在上一周期无对应（同 metricPath）→ 不挂 compareSeries（图照常只画当前实线）。
 * 空集合不抛错。返回新数组（不就地改原图）。
 *
 * @param currentCharts 当前周期图集（buildMetricCharts/buildDeviceMetricCharts 产出）
 * @param prevCharts    上一周期图集（同口径，已套同样星期/小时段过滤后转置）
 * @param offsetMs      偏移毫秒（= 当前窗口长 L）
 */
export function attachCompareSeries(
  currentCharts: MetricChart[],
  prevCharts: MetricChart[],
  offsetMs: number,
): MetricChart[] {
  const prevByMetric = new Map<string, MetricChart>();
  prevCharts.forEach((p) => prevByMetric.set(p.metricPath, p));
  return currentCharts.map((c) => {
    const p = prevByMetric.get(c.metricPath);
    if (!p) return c;
    return {
      ...c,
      compareSeries: buildCompareSeries(c.buckets, p.series, p.buckets, offsetMs),
    };
  });
}
