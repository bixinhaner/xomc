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
  /** 与 buckets 一一对应的桶结束时间（endTime），供 tooltip 显示「开始~结束」时间段 */
  bucketEnds: string[];
  series: MetricSeries[];
  /** T-0189 周期对比：上一周期系列（已按整数粒度步长对齐到当前轴），ChartCard 渲染为虚线。 */
  compareSeries?: MetricSeries[];
  /**
   * T-0194 周期对比 tooltip：每个当前轴索引对应的上一周期桶起止时间（与 buckets 索引对齐，
   * 无对应点处留空串）。供 ChartCard tooltip 补显上一周期真实「开始~结束」时间段。
   */
  compareBuckets?: string[];
  compareBucketEnds?: string[];
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
 *   device_group     → '设备组 <组名>'（后端 JOIN 解析名优先；缺失回退 uuid 前 8，剥 `DeviceGroup=` 前缀）
 *   product          → '产品 <产品名>'（后端 JOIN 解析名优先；缺失回退 id 前 8）
 *   aggregate_group  → '聚合组' + LDN（小区级，带原 LDN 便于区分多小区）
 *
 * PM-线名解析：name 为后端读时 JOIN 解析出的可读名（product 维度=产品名、device_group 维度=组名）。
 * 命中（非空）时显示可读名；缺失（脏数据/已删/NULL）回退现状的 id 前 8 位，保证不空白。
 */
export function seriesLabelOf(key: string, dimension: AdhocDimension, name?: string): string {
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
      if (name) return `设备组 ${name}`;
      const v = key.startsWith('DeviceGroup=') ? key.slice('DeviceGroup='.length) : key;
      return `设备组 ${v.slice(0, 8)}`;
    }
    case 'product':
      return name ? `产品 ${name}` : `产品 ${key.slice(0, 8)}`;
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
      // startTime → endTime（同桶各行 endTime 相同，后到覆盖）
      ends: Map<string, string>;
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
        ends: new Map(),
        seriesOrder: [],
        seriesName: new Map(),
        points: new Map(),
      };
      byMetric.set(r.metricPath, m);
      metricOrder.push(r.metricPath);
    }
    m.buckets.add(r.startTime);
    m.ends.set(r.startTime, r.endTime);
    const key = seriesKeyOf(r, dimension);
    let pts = m.points.get(key);
    if (!pts) {
      pts = new Map();
      m.points.set(key, pts);
      m.seriesOrder.push(key);
      // PM-线名解析：把该行后端 JOIN 解析出的可读名按维度取出传给标签函数（缺失则 undefined 回退 id 前 8）。
      const readableName =
        dimension === 'product'
          ? r.productName
          : dimension === 'device_group'
            ? r.deviceGroupName
            : undefined;
      m.seriesName.set(key, seriesLabelOf(key, dimension, readableName));
    }
    // 同 (系列, 桶) 多行取后到值（正常一行一值）。
    pts.set(r.startTime, r.metricValue);
  });

  return metricOrder.map((metricPath) => {
    const m = byMetric.get(metricPath)!;
    const buckets = Array.from(m.buckets).sort();
    const bucketEnds = buckets.map((b) => m.ends.get(b) ?? '');
    const series: MetricSeries[] = m.seriesOrder.map((key) => {
      const pts = m.points.get(key)!;
      const values: MetricSeriesValue[] = buckets.map((b) => {
        const v = pts.get(b);
        return v === undefined ? '-' : v;
      });
      return { key, name: m.seriesName.get(key) ?? key, values };
    });
    return { metricPath, displayName: m.displayName, buckets, bucketEnds, series };
  });
}
