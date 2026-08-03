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
import { isFinitePmMetricValue } from '@core/utils/pmMetricValue';
import type { Locale } from '@core/types/common';

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
  unit?: string;
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

export type MetricDisplayNameMap = ReadonlyMap<string, string>;

export interface MetricSeriesIdentity {
  key: string;
  name: string;
}

export interface TrustedSeriesSources {
  deviceSns?: string[];
  objectLdns?: string[];
  filterOptions?: Array<{ value: string; label: string }>;
  selectedKeys?: string[];
}

export interface SeriesLabelPrefixes {
  network: string;
  band: string;
  deviceGroup: string;
  product: string;
  aggregateGroup: string;
}

export function formatMetricChartDisplayName(
  metricPath: string,
  displayName?: string,
): string {
  const name = displayName?.trim();
  if (!name || name === metricPath) return metricPath;
  return `${name}（${metricPath}）`;
}

function metricChartDisplayNameOf(
  metricPath: string,
  displayName: string | undefined,
  metricDisplayNames?: MetricDisplayNameMap,
): string {
  const name = displayName?.trim();
  if (name && name !== metricPath) return formatMetricChartDisplayName(metricPath, name);
  return formatMetricChartDisplayName(metricPath, metricDisplayNames?.get(metricPath));
}

function trustedSeriesLabelOf(
  key: string,
  dimension: AdhocDimension,
  labels: SeriesLabelPrefixes,
  name?: string,
): string {
  switch (dimension) {
    case 'network':
      return labels.network;
    case 'device':
      return key;
    case 'band': {
      const value = key.startsWith('Band=') ? key.slice('Band='.length) : key;
      return `${labels.band} ${value}`;
    }
    case 'device_group': {
      if (name) return `${labels.deviceGroup} ${name}`;
      const value = key.startsWith('DeviceGroup=')
        ? key.slice('DeviceGroup='.length)
        : key;
      return `${labels.deviceGroup} ${value.slice(0, 8)}`;
    }
    case 'product':
      if (name) {
        return name.startsWith(`${labels.product} `) ? name : `${labels.product} ${name}`;
      }
      return `${labels.product} ${key.slice(0, 8)}`;
    case 'aggregate_group':
      return `${labels.aggregateGroup} ${key}`;
    default:
      return key;
  }
}

/**
 * #198：为“完全无结果”的图卡派生系列身份。
 *
 * 仅使用任务配置或 filter-options 这类可信骨架；无法确定身份时返回空数组，
 * 禁止用 __unknown__ 之类的假系列冒充真实对象。
 */
export function buildTrustedSeriesIdentities(
  dimension: AdhocDimension,
  sources: TrustedSeriesSources,
  labels: SeriesLabelPrefixes,
): MetricSeriesIdentity[] {
  if (dimension === 'network') {
    return [{
      key: '__network__',
      name: trustedSeriesLabelOf('__network__', dimension, labels),
    }];
  }

  let entries: Array<{ key: string; label?: string }>;
  if (dimension === 'device') {
    entries = (sources.deviceSns ?? []).map((key) => ({ key }));
  } else if (dimension === 'aggregate_group') {
    entries = (sources.objectLdns ?? []).map((key) => ({ key }));
  } else {
    const selected = new Set(sources.selectedKeys ?? []);
    entries = (sources.filterOptions ?? [])
      .filter((option) => selected.size === 0 || selected.has(option.value))
      .map((option) => ({ key: option.value, label: option.label }));
  }

  const seen = new Set<string>();
  return entries.flatMap(({ key, label }) => {
    if (!key || seen.has(key)) return [];
    seen.add(key);
    return [{
      key,
      name: trustedSeriesLabelOf(key, dimension, labels, label),
    }];
  });
}

/**
 * #198：任务配置是图卡全集的可信骨架，结果行只负责提供真实点。
 *
 * 某个 metric_path 在当前筛选/粒度完全没有结果行时，仍按任务配置生成空图卡；
 * series 只保留身份且 values 为空，后续扩轴统一补 '-'，这里绝不制造 0 或结果行。
 */
export function ensureConfiguredMetricCharts(
  charts: MetricChart[],
  metricPaths: string[] | undefined,
  seriesIdentities: MetricSeriesIdentity[],
  metricDisplayNames?: MetricDisplayNameMap,
): MetricChart[] {
  if (!metricPaths || metricPaths.length === 0) return charts;
  const byMetric = new Map(charts.map((chart) => [chart.metricPath, chart]));
  const seen = new Set<string>();
  const out: MetricChart[] = [];
  metricPaths.forEach((metricPath) => {
    if (seen.has(metricPath)) return;
    seen.add(metricPath);
    const chart = byMetric.get(metricPath);
    if (chart) {
      out.push(chart);
      return;
    }
    out.push({
      metricPath,
      displayName: formatMetricChartDisplayName(metricPath, metricDisplayNames?.get(metricPath)),
      buckets: [],
      bucketEnds: [],
      series: seriesIdentities.map(({ key, name }) => ({ key, name, values: [] })),
    });
  });
  return out;
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
export function seriesLabelOf(
  key: string,
  dimension: AdhocDimension,
  name?: string,
  locale: Locale = 'zh-CN',
): string {
  const isEnglish = locale === 'en-US';
  switch (dimension) {
    case 'network':
      return isEnglish ? 'Network' : '全网';
    case 'device':
      return key;
    case 'band': {
      const v = key.startsWith('Band=') ? key.slice('Band='.length) : key;
      return `${isEnglish ? 'Band' : '频段'} ${v}`;
    }
    case 'device_group': {
      const prefix = isEnglish ? 'Device Group' : '设备组';
      if (name) return `${prefix} ${name}`;
      const v = key.startsWith('DeviceGroup=') ? key.slice('DeviceGroup='.length) : key;
      return `${prefix} ${v.slice(0, 8)}`;
    }
    case 'product':
      return name ? `${isEnglish ? 'Product' : '产品'} ${name}` : `${isEnglish ? 'Product' : '产品'} ${key.slice(0, 8)}`;
    case 'aggregate_group':
      return `${isEnglish ? 'Aggregate Group' : '聚合组'} ${key}`;
    default:
      return key;
  }
}

/**
 * T-0194：按任务「已选指标清单」过滤图表（统一对内置 + 自建生效）。
 *   - metricPaths 非空：只保留 metricPath ∈ 清单 的图（让"指标数 X"与出图数一致）。
 *   - metricPaths 为空：不过滤（兜底全画，避免空任务画不出）。
 * 纯函数，便于单测；不改图内系列，只挑图。
 */
export function filterChartsByMetricPaths(
  charts: MetricChart[],
  metricPaths: string[] | undefined,
): MetricChart[] {
  if (!metricPaths || metricPaths.length === 0) return charts;
  const byMetric = new Map(charts.map((c) => [c.metricPath, c]));
  const out: MetricChart[] = [];
  const seen = new Set<string>();
  metricPaths.forEach((metricPath) => {
    if (seen.has(metricPath)) return;
    seen.add(metricPath);
    const chart = byMetric.get(metricPath);
    if (chart) out.push(chart);
  });
  return out;
}

/**
 * #192：按任务 metric_paths 过滤原始结果行。必须在 buildMetricCharts 前执行，
 * 防止配置外指标参与 device_group/product 等维度的固定 legend 全集。
 */
export function filterRowsByMetricPaths(
  rows: AdhocResultRow[],
  metricPaths: string[] | undefined,
): AdhocResultRow[] {
  if (!metricPaths || metricPaths.length === 0) return rows;
  const set = new Set(metricPaths);
  return rows.filter((row) => set.has(row.metricPath));
}

function isFormalResultRow(row: AdhocResultRow): boolean {
  return row.partial !== true && row.periodComplete !== false;
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
  locale: Locale = 'zh-CN',
  metricDisplayNames?: MetricDisplayNameMap,
): MetricChart[] {
  const filtered = rows.filter((r) => r.granularity === granularity && isFormalResultRow(r));
  if (filtered.length === 0) return [];

  // 按 metricPath 分图，保留出现顺序。
  const metricOrder: string[] = [];
  // metricPath → { displayName, bucketSet, series: key → (bucket → value), seriesOrder, seriesName }
  const byMetric = new Map<
    string,
    {
      displayName: string;
      unit?: string;
      unitConflict: boolean;
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
        displayName: metricChartDisplayNameOf(r.metricPath, r.displayName, metricDisplayNames),
        unit: undefined,
        unitConflict: false,
        buckets: new Set(),
        ends: new Map(),
        seriesOrder: [],
        seriesName: new Map(),
        points: new Map(),
      };
      byMetric.set(r.metricPath, m);
      metricOrder.push(r.metricPath);
    }
    const unit = r.unit?.trim();
    if (unit) {
      if (!m.unit && !m.unitConflict) {
        m.unit = unit;
      } else if (m.unit !== unit) {
        m.unit = undefined;
        m.unitConflict = true;
      }
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
      m.seriesName.set(key, seriesLabelOf(key, dimension, readableName, locale));
    }
    // 同 (系列, 桶) 多行取后到值（正常一行一值）。
    if (isFinitePmMetricValue(r.metricValue)) {
      pts.set(r.startTime, r.metricValue);
    }
  });

  // #194：legend 固定全集——某指标公式分母 counter 缺失 / 为 0 时该 (组×桶) 整条被后端跳过
  // （不产假 0，逐点语义正确），但前端若仅按本指标返回行的 distinct key 渲染 legend，会让那条线
  // 整组消失，造成「上行流量两组、E-RAB掉线率只剩一组」的错觉。
  // 治法：对「实体身份即 legend」的维度（device_group/product/band/aggregate_group），把任务本批
  // 结果里所有指标出现过的系列键并成全集，每张图都按全集铺 legend；本指标缺的桶留 '-'（断点，
  // connectNulls=false 自然断开），绝不补 0 假点。network（单线）/device（缺设备=该设备真无数据，
  // legend 缺失有意义）不套全集，保持原行为。
  const fixedLegendDims: AdhocDimension[] = [
    'device_group',
    'product',
    'band',
    'aggregate_group',
  ];
  const useFixedLegend = fixedLegendDims.includes(dimension);

  // 全局系列全集（跨指标，按首次出现序），key → 展示名。
  const globalOrder: string[] = [];
  const globalName = new Map<string, string>();
  if (useFixedLegend) {
    metricOrder.forEach((metricPath) => {
      const m = byMetric.get(metricPath)!;
      m.seriesOrder.forEach((key) => {
        if (!globalName.has(key)) {
          globalOrder.push(key);
          globalName.set(key, m.seriesName.get(key) ?? key);
        }
      });
    });
  }

  return metricOrder.map((metricPath) => {
    const m = byMetric.get(metricPath)!;
    const buckets = Array.from(m.buckets).sort();
    const bucketEnds = buckets.map((b) => m.ends.get(b) ?? '');
    // 本图系列键 = 全集（固定 legend 维度）或本指标自身键（其它维度）。
    const seriesKeys = useFixedLegend ? globalOrder : m.seriesOrder;
    const series: MetricSeries[] = seriesKeys.map((key) => {
      const pts = m.points.get(key);
      const values: MetricSeriesValue[] = buckets.map((b) => {
        const v = pts?.get(b);
        return v === undefined ? '-' : v;
      });
      // 名字优先取本指标解析名，缺则取全局解析名（该组在别的指标里出现过），再回退 key。
      const name = m.seriesName.get(key) ?? globalName.get(key) ?? key;
      return { key, name, values };
    });
    return { metricPath, displayName: m.displayName, unit: m.unit, buckets, bucketEnds, series };
  });
}
