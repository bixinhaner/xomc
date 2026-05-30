/**
 * T-0187 任务仪表盘自动出图 — 纯函数（转置 + 系列键派生），与 React 解耦便于单测。
 *
 * 转置语义：聚合结果行（long 格式，每行一指标一桶一系列）→ 按指标分图、图内按系列键分线。
 *   - 过滤：只取选中 granularity 的行。
 *   - 分图：按 metricPath。
 *   - 分线：按 seriesKeyOf(row, dimension) 派生的系列键（network=单线/全网，device=SN，band/group/product 等多线）。
 *   - 对齐：每条线的 values 对齐该图全桶集合（startTime 升序去重），缺桶补 '-'（断线，不连）。
 *
 * 系列键来源（逐维核实自后端 aggregator/executor，见账本回合 0）：
 *   network          → 单条线（全网汇总，无实体键）
 *   device           → deviceSn（真实 SN）
 *   band             → objectLdn = `Band=<值>`
 *   device_group     → objectLdn = `DeviceGroup=<uuid>`
 *   product          → productId（UUID 列）
 *   aggregate_group  → objectLdn（小区级 LDN）
 */

import type { AdhocResultRow, AdhocDimension } from '@core/types/pmAdhoc';

export type MetricSeriesValue = number | '-';

export interface MetricSeries {
  /** 系列键（图内唯一区分多条线） */
  key: string;
  /** 系列展示名（图例 + tooltip） */
  name: string;
  /** 对齐到该图全桶集合的值序列（缺桶 '-'） */
  values: MetricSeriesValue[];
}

export interface MetricChart {
  metricPath: string;
  displayName: string;
  /** 该图横轴桶（startTime 升序去重） */
  buckets: string[];
  series: MetricSeries[];
}

/**
 * 派生系列键：图内据此分多条线。维度决定取哪个字段。
 * network 维度恒为单线，返回固定键 '__network__'。
 * 其余维度缺失对应字段时回退 deviceSn（再回退固定键），保证不抛错、不丢线。
 */
export function seriesKeyOf(row: AdhocResultRow, dimension: AdhocDimension): string {
  switch (dimension) {
    case 'network':
      return '__network__';
    case 'device':
      return row.deviceSn || '__unknown__';
    case 'product':
      return row.productId || row.deviceSn || '__unknown__';
    case 'band':
    case 'device_group':
    case 'aggregate_group':
      return row.objectLdn || row.deviceSn || '__unknown__';
    default:
      return row.deviceSn || '__unknown__';
  }
}

/**
 * 系列展示名：把系列键翻成人能读的标签。
 *   network          → '全网'
 *   device           → SN 原样
 *   band             → '频段 X'（剥 `Band=` 前缀）
 *   device_group     → '设备组 <uuid前8>'（剥 `DeviceGroup=` 前缀）
 *   product          → '产品 <id前8>'
 *   aggregate_group  → '聚合组' + LDN（小区级，带原 LDN 便于区分多小区）
 */
export function seriesLabelOf(key: string, dimension: AdhocDimension): string {
  switch (dimension) {
    case 'network':
      return '全网';
    case 'device':
      return key;
    case 'band': {
      const v = key.startsWith('Band=') ? key.slice('Band='.length) : key;
      return `频段 ${v}`;
    }
    case 'device_group': {
      const v = key.startsWith('DeviceGroup=') ? key.slice('DeviceGroup='.length) : key;
      return `设备组 ${v.slice(0, 8)}`;
    }
    case 'product':
      return `产品 ${key.slice(0, 8)}`;
    case 'aggregate_group':
      return `聚合组 ${key}`;
    default:
      return key;
  }
}

/**
 * 转置主函数：结果行 → 每指标一张图（图内按系列键分多条线，缺桶补 '-'）。
 * @param rows        任务结果行（多粒度混在一起，本函数内按 granularity 过滤）
 * @param dimension   任务聚合维度（决定系列键派生）
 * @param granularity 选中粒度（只取该粒度的行）
 */
export function buildMetricCharts(
  rows: AdhocResultRow[],
  dimension: AdhocDimension,
  granularity: string,
): MetricChart[] {
  const filtered = rows.filter((r) => r.granularity === granularity);
  if (filtered.length === 0) return [];

  // 按 metricPath 分图，保留出现顺序。
  const metricOrder: string[] = [];
  // metricPath → { displayName, bucketSet, series: key → (bucket → value), seriesOrder, seriesName }
  const byMetric = new Map<
    string,
    {
      displayName: string;
      buckets: Set<string>;
      seriesOrder: string[];
      seriesName: Map<string, string>;
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
        seriesName: new Map(),
        points: new Map(),
      };
      byMetric.set(r.metricPath, m);
      metricOrder.push(r.metricPath);
    }
    m.buckets.add(r.startTime);
    const key = seriesKeyOf(r, dimension);
    let pts = m.points.get(key);
    if (!pts) {
      pts = new Map();
      m.points.set(key, pts);
      m.seriesOrder.push(key);
      m.seriesName.set(key, seriesLabelOf(key, dimension));
    }
    // 同 (系列, 桶) 多行取后到值（正常一行一值）。
    pts.set(r.startTime, r.metricValue);
  });

  return metricOrder.map((metricPath) => {
    const m = byMetric.get(metricPath)!;
    const buckets = Array.from(m.buckets).sort();
    const series: MetricSeries[] = m.seriesOrder.map((key) => {
      const pts = m.points.get(key)!;
      const values: MetricSeriesValue[] = buckets.map((b) => {
        const v = pts.get(b);
        return v === undefined ? '-' : v;
      });
      return { key, name: m.seriesName.get(key) ?? key, values };
    });
    return { metricPath, displayName: m.displayName, buckets, series };
  });
}
