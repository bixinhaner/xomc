/**
 * T-0188 设备列表页签出图 — 纯函数（聚合直查行转置），与 React 解耦便于单测。
 *
 * 转置语义：pm_metrics 直查行（AggregatedRow，long 格式，每行一指标一桶一设备一小区）
 *   → 按指标分图、图内按「设备 + 小区/PLMN」分线（T-0193：每设备每小区一条线）。
 *   - 过滤：只取选中 granularity 的行。
 *   - 分图：按 metricPath。
 *   - 分线（T-0193）：系列键 = `deviceSn|objectLdn` 组合键；系列名 = 友好名「设备尾号 · 小区X · PLMNY」。
 *     缺 object_ldn 的行兜底退化为按设备单线（键仅 deviceSn，名仅尾号），不丢线、不抛错。
 *   - 对齐：每条线 values 对齐该图全桶集合（startTime 升序去重），缺桶补 '-'（断线，不连）。
 *   - null 值：metricValue==null（fill_empty 占位行 / 缺采）当缺桶 '-'（不画点、断线）。
 *
 * 复用 taskDashboardUtils 的 MetricChart / MetricSeries 类型（同 shape，下游 ChartCard 共享）。
 * 与 buildMetricCharts 差异：吃 AggregatedRow（非 AdhocResultRow），系列键 = 设备+小区组合键。
 */

import type { AggregatedRow } from '@core/types/pmDashboard';
import { buildDeviceSeriesName, deviceSnTail } from '@core/types/pmObject';
import type { MetricChart, MetricSeries, MetricSeriesValue } from './taskDashboardUtils';

export type { MetricChart, MetricSeries, MetricSeriesValue } from './taskDashboardUtils';

const UNKNOWN_SN = '__unknown__';

/**
 * 转置主函数：聚合直查行 → 每指标一张图（图内按 deviceSn 分多条线，缺桶/null 补 '-'）。
 * @param rows        多设备合并的聚合行（可能混多粒度，本函数内按 granularity 过滤）
 * @param granularity 选中粒度（只取该粒度的行）
 */
export function buildDeviceMetricCharts(
  rows: AggregatedRow[],
  granularity: string,
): MetricChart[] {
  const filtered = rows.filter((r) => r.granularity === granularity);
  if (filtered.length === 0) return [];

  // 按 metricPath 分图，保留出现顺序。
  const metricOrder: string[] = [];
  const byMetric = new Map<
    string,
    {
      displayName: string;
      buckets: Set<string>;
      // startTime → endTime（同桶各行 endTime 相同，后到覆盖）；含占位桶以保证桶轴每点都有结束时间
      ends: Map<string, string>;
      seriesOrder: string[];
      // seriesKey → { name, points }；name 是友好名（设备尾号 · 小区 · PLMN）。
      series: Map<string, { name: string; points: Map<string, number> }>;
    }
  >();

  filtered.forEach((r) => {
    let m = byMetric.get(r.metricPath);
    if (!m) {
      m = {
        displayName: r.displayName || r.metricPath,
        buckets: new Set(),
        ends: new Map(),
        seriesOrder: [],
        series: new Map(),
      };
      byMetric.set(r.metricPath, m);
      metricOrder.push(r.metricPath);
    }
    m.buckets.add(r.startTime);
    m.ends.set(r.startTime, r.endTime);
    // T-0193 分线键：设备 + 小区/PLMN 组合。缺 object_ldn → 兜底退化为按设备单线。
    const sn = r.deviceSn || UNKNOWN_SN;
    const ldn = r.objectLdn ?? '';
    // 纯 fill_empty 占位行（无小区归属 + 无值）只用于对齐桶轴，不单独成线——
    // 否则按小区分线时这些无小区占位行会聚成一条空的「仅设备」兜底线（T-0193）。
    if (!ldn && r.metricValue === null) return;
    const key = ldn ? `${sn}|${ldn}` : sn;
    let s = m.series.get(key);
    if (!s) {
      // 系列名：有小区 → 「尾号 · 小区X · PLMNY」；无小区 → 仅尾号（兜底单线）。
      const name = ldn ? buildDeviceSeriesName(sn, ldn) : deviceSnTail(sn) || sn;
      s = { name, points: new Map() };
      m.series.set(key, s);
      m.seriesOrder.push(key);
    }
    // null（fill_empty / 缺采）不入点表 → 对齐时该桶补 '-' 断线；同 (系列, 桶) 多行取后到值。
    if (r.metricValue !== null) {
      s.points.set(r.startTime, r.metricValue);
    }
  });

  return metricOrder.map((metricPath) => {
    const m = byMetric.get(metricPath)!;
    const buckets = Array.from(m.buckets).sort();
    const bucketEnds = buckets.map((b) => m.ends.get(b) ?? '');
    const series: MetricSeries[] = m.seriesOrder.map((key) => {
      const s = m.series.get(key)!;
      const values: MetricSeriesValue[] = buckets.map((b) => {
        const v = s.points.get(b);
        return v === undefined ? '-' : v;
      });
      return { key, name: s.name, values };
    });
    return { metricPath, displayName: m.displayName, buckets, bucketEnds, series };
  });
}

/**
 * T-0193 即席小区/PLMN 过滤（纯前端，不落库）：在已取的聚合行里只保留命中白名单的行。
 *   - allowedLdns 为空 → 不过滤，原样返回（向后兼容「不下钻」）。
 *   - 非空 → 仅保留 objectLdn ∈ allowedLdns 的行；无 objectLdn 的行在过滤模式下丢弃
 *     （用户既然挑了具体小区，没有小区归属的行不应混入）。
 * 与设备列表页签的下钻勾选配合：勾选子集时生效，全选时上层传空数组。
 */
export function filterRowsByObjectLdns(
  rows: AggregatedRow[],
  allowedLdns: string[],
): AggregatedRow[] {
  if (allowedLdns.length === 0) return rows;
  const allow = new Set(allowedLdns);
  return rows.filter((r) => (r.objectLdn ? allow.has(r.objectLdn) : false));
}
