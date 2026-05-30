/**
 * T-0188 设备列表页签出图 — 纯函数（聚合直查行转置），与 React 解耦便于单测。
 *
 * 转置语义：pm_metrics 直查行（AggregatedRow，long 格式，每行一指标一桶一设备）
 *   → 按指标分图、图内按 deviceSn 分线（每设备一条线）。
 *   - 过滤：只取选中 granularity 的行。
 *   - 分图：按 metricPath。
 *   - 分线：系列键恒 = deviceSn、系列名恒 = deviceSn（缺 SN 回退固定键，不丢线、不抛错）。
 *   - 对齐：每条线 values 对齐该图全桶集合（startTime 升序去重），缺桶补 '-'（断线，不连）。
 *   - null 值：metricValue==null（fill_empty 占位行 / 缺采）当缺桶 '-'（不画点、断线）。
 *
 * 复用 taskDashboardUtils 的 MetricChart / MetricSeries 类型（同 shape，下游 ChartCard 共享）。
 * 与 buildMetricCharts 差异：吃 AggregatedRow（非 AdhocResultRow），系列键恒 = deviceSn。
 */

import type { AggregatedRow } from '@core/types/pmDashboard';
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
      seriesOrder: string[];
      points: Map<string, Map<string, number>>;
    }
  >();

  filtered.forEach((r) => {
    let m = byMetric.get(r.metricPath);
    if (!m) {
      m = {
        displayName: r.displayName || r.metricPath,
        buckets: new Set(),
        seriesOrder: [],
        points: new Map(),
      };
      byMetric.set(r.metricPath, m);
      metricOrder.push(r.metricPath);
    }
    m.buckets.add(r.startTime);
    const key = r.deviceSn || UNKNOWN_SN;
    let pts = m.points.get(key);
    if (!pts) {
      pts = new Map();
      m.points.set(key, pts);
      m.seriesOrder.push(key);
    }
    // null（fill_empty / 缺采）不入点表 → 对齐时该桶补 '-' 断线；同 (设备, 桶) 多行取后到值。
    if (r.metricValue !== null) {
      pts.set(r.startTime, r.metricValue);
    }
  });

  return metricOrder.map((metricPath) => {
    const m = byMetric.get(metricPath)!;
    const buckets = Array.from(m.buckets).sort();
    const series: MetricSeries[] = m.seriesOrder.map((sn) => {
      const pts = m.points.get(sn)!;
      const values: MetricSeriesValue[] = buckets.map((b) => {
        const v = pts.get(b);
        return v === undefined ? '-' : v;
      });
      // 系列键恒 = deviceSn，系列名恒 = deviceSn。
      return { key: sn, name: sn, values };
    });
    return { metricPath, displayName: m.displayName, buckets, series };
  });
}
