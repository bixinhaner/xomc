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
import type { AdhocDimension } from '@core/types/pmAdhoc';
import type { MetricChart, MetricSeries, MetricSeriesValue } from './taskDashboardUtils';

/** 全选用的常量集合（默认行为）。 */
export const ALL_WEEKDAYS: number[] = [0, 1, 2, 3, 4, 5, 6];
export const ALL_HOURS: number[] = Array.from({ length: 24 }, (_, i) => i);

/**
 * PM-DASH-DIMFILTER 频段可读化：把后端给的频段分组键原值（如 'Band=42'）去掉 'Band=' 前缀，
 * 返回纯频段号（'42'）供 i18n 模板 '频段 {n}' 渲染。
 * 不含 'Band=' 前缀（脏数据/非频段值）时原样返回，保证不崩、label 非空。
 * MVP 不引入频段→频率/中文名映射表（YAGNI）。
 */
export function parseBandNumber(value: string): string {
  return value.startsWith('Band=') ? value.slice('Band='.length) : value;
}

/**
 * PM-DASH-DIMFILTER 维度→筛选框元数据：决定该聚合维度是否渲染子集筛选框、用哪个标题/占位符语料键。
 *   - product / device_group / band → 渲染对应框。
 *   - device / aggregate_group / network → 返回 null（不渲染该框，无子集可筛）。
 */
export function dimensionFilterMeta(
  dimension: AdhocDimension | undefined,
): { titleId: string; placeholderId: string } | null {
  switch (dimension) {
    case 'product':
      return {
        titleId: 'perf.dashboard.filterProduct',
        placeholderId: 'perf.dashboard.allProducts',
      };
    case 'device_group':
      return {
        titleId: 'perf.dashboard.filterDeviceGroup',
        placeholderId: 'perf.dashboard.allDeviceGroups',
      };
    case 'band':
      return {
        titleId: 'perf.dashboard.filterBand',
        placeholderId: 'perf.dashboard.allBands',
      };
    default:
      // device / aggregate_group / network / undefined → 不渲染。
      return null;
  }
}

/**
 * PM-DASH-DIMFILTER 维度子集选中值 → results 查询入参映射。
 *   - product 维度选中值是 product_id → productIds。
 *   - device_group / band 维度选中值是 object_ldn 原值（DeviceGroup=<uuid> / Band=<值>）→ objectLdns。
 *   - 空选中 = 不过滤 = 两者皆 undefined（无回归）。
 */
export function dimSelectionToParams(
  dimension: AdhocDimension | undefined,
  selected: string[],
): { productIds?: string[]; objectLdns?: string[] } {
  if (selected.length === 0) return {};
  if (dimension === 'product') return { productIds: selected };
  if (dimension === 'device_group' || dimension === 'band') return { objectLdns: selected };
  return {};
}

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
 * 固定步长粒度 → 单桶毫秒步长。month 不是固定毫秒（自然月天数不等），返回 null 走日历平移。
 * 口径与仪表盘粒度词表一致（'15min' | 'hourly' | 'daily' | 'weekly' | 'monthly'）。
 */
const GRAN_STEP_MS: Record<string, number> = {
  '15min': 15 * 60 * 1000, // 900000
  hourly: 60 * 60 * 1000, // 3600000
  daily: 24 * 60 * 60 * 1000, // 86400000
  weekly: 7 * 24 * 60 * 60 * 1000, // 604800000
};
const APPROX_MONTH_MS = 30 * 24 * 60 * 60 * 1000;

/** 该粒度的固定步长（毫秒）；month/未知粒度返回 undefined。 */
export function granularityStepMs(granularity: string): number | undefined {
  return GRAN_STEP_MS[granularity];
}

/**
 * 把原始平移量 offsetMs（= 窗口长 L，可能带亚秒/分钟零头）吸附为整数个粒度步长，
 * 使整点对齐的上一周期桶精确落到整点对齐的当前桶。
 * 固定步长粒度：snapped = round(offsetMs/step)*step。
 * month/未知粒度：返回 undefined（buildCompareSeries 走日历月平移分支）。
 */
function snapOffsetMs(offsetMs: number, granularity: string): number | undefined {
  const step = GRAN_STEP_MS[granularity];
  if (step === undefined) return undefined;
  return Math.round(offsetMs / step) * step;
}

/**
 * 对比序列对齐：把上一周期系列（在 prevBuckets 上的值）按整数个粒度步长平移落到当前轴 currentBuckets。
 *
 * 根因修复（T-0194）：前后两周期的桶都是后端按粒度整点/整日对齐的；原实现直接用带零头的
 * offsetMs(=end-start) 平移后做精确毫秒等值匹配，整点桶 + 带零头偏移落到非整点毫秒，与当前
 * 整点轴永远 miss → 整条上一周期虚线全变 '-'。现把平移量吸附到整数个粒度步长后再对齐。
 *   - 固定步长粒度（15min/hourly/daily/weekly）：平移量吸附为 round(offsetMs/step)*step。
 *   - month（monthly）：按整数日历月平移 dayjs(prevBucket).add(n,'month')，n=round(offsetMs/≈30d)，
 *     避免按固定毫秒漂移出月界。
 * 对齐后：当前桶 Tc 的对比值 = 上一周期在「Tc 对应的上一周期桶」处的值；无对应点 → '-'（断线，不连，不插值）。
 * 空数据（currentBuckets 空 / prevSeries 空）不抛错，返回对应空结果。
 *
 * 同时产出每个当前轴索引对应的上一周期桶起止时间（compareBuckets/compareBucketEnds，与 currentBuckets
 * 索引对齐，无对应点处留空串），供 ChartCard tooltip 显示上一周期真实时间戳。
 *
 * @param currentBuckets   当前周期横轴桶（startTime 升序字符串）
 * @param prevSeries       上一周期系列（同 buildMetricCharts/buildDeviceMetricCharts 转置产出）
 * @param prevBuckets      上一周期横轴桶（与 prevSeries.values 一一对齐）
 * @param prevBucketEnds   上一周期桶结束时间（与 prevBuckets 索引对齐，可选；缺则 compareBucketEnds 留空串）
 * @param offsetMs         原始平移毫秒（= 当前窗口长 L，可能带零头）
 * @param granularity      粒度（决定吸附步长 / 日历月平移）
 */
export function buildCompareSeries(
  currentBuckets: string[],
  prevSeries: MetricSeries[],
  prevBuckets: string[],
  offsetMs: number,
  granularity: string,
): { series: MetricSeries[]; compareBuckets: string[]; compareBucketEnds: string[] } {
  return {
    series: buildCompareSeriesValues(currentBuckets, prevSeries, prevBuckets, offsetMs, granularity),
    ...buildCompareBuckets(currentBuckets, prevBuckets, [], offsetMs, granularity),
  };
}

/**
 * 把上一周期桶时间戳按整数粒度步长平移到当前轴时间戳（毫秒）。
 * 固定步长粒度：T_prev + snappedOffsetMs。month：dayjs(T_prev).add(n,'month')（整数日历月）。
 * tz 安全：均为在整点对齐的 T_prev 上加整数步长 / 整数月，保持整点对齐（业务时区无 DST）。
 */
function shiftPrevToCurrentMs(prevMs: number, snapped: number | undefined, offsetMs: number): number {
  if (snapped !== undefined) return prevMs + snapped;
  // month 分支：按整数日历月平移。
  const n = Math.round(offsetMs / APPROX_MONTH_MS);
  return dayjs(prevMs).add(n, 'month').valueOf();
}

/** 仅产出对比系列值（内部复用，不含 compareBuckets）。 */
function buildCompareSeriesValues(
  currentBuckets: string[],
  prevSeries: MetricSeries[],
  prevBuckets: string[],
  offsetMs: number,
  granularity: string,
): MetricSeries[] {
  const snapped = snapOffsetMs(offsetMs, granularity);
  return prevSeries.map((s) => {
    // 平移后时间戳(ms) -> value
    const shifted = new Map<number, MetricSeriesValue>();
    prevBuckets.forEach((b, i) => {
      const ts = dayjs(b);
      if (!ts.isValid()) return;
      const v = s.values[i];
      if (v === undefined) return;
      shifted.set(shiftPrevToCurrentMs(ts.valueOf(), snapped, offsetMs), v);
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
 * 产出每个当前轴索引对应的上一周期桶起止时间（与 currentBuckets 索引对齐，无对应点处留空串）。
 * 平移口径与 buildCompareSeriesValues 完全一致（同 snapped / 同日历月平移），保证 tooltip 与曲线同源。
 */
function buildCompareBuckets(
  currentBuckets: string[],
  prevBuckets: string[],
  prevBucketEnds: string[],
  offsetMs: number,
  granularity: string,
): { compareBuckets: string[]; compareBucketEnds: string[] } {
  const snapped = snapOffsetMs(offsetMs, granularity);
  // 当前轴时间戳(ms) -> { start, end }（上一周期原始起止时间字符串）
  const byCurrentMs = new Map<number, { start: string; end: string }>();
  prevBuckets.forEach((b, i) => {
    const ts = dayjs(b);
    if (!ts.isValid()) return;
    byCurrentMs.set(shiftPrevToCurrentMs(ts.valueOf(), snapped, offsetMs), {
      start: b,
      end: prevBucketEnds[i] ?? '',
    });
  });
  const compareBuckets: string[] = [];
  const compareBucketEnds: string[] = [];
  currentBuckets.forEach((cb) => {
    const ct = dayjs(cb);
    const hit = ct.isValid() ? byCurrentMs.get(ct.valueOf()) : undefined;
    compareBuckets.push(hit?.start ?? '');
    compareBucketEnds.push(hit?.end ?? '');
  });
  return { compareBuckets, compareBucketEnds };
}

/**
 * 把上一周期图集挂到当前图集：按 metricPath 匹配图，按整数粒度步长对齐上一周期系列 + 产出上周期桶起止时间。
 * 当前图在上一周期无对应（同 metricPath）→ 不挂 compareSeries（图照常只画当前实线）。
 * 空集合不抛错。返回新数组（不就地改原图）。
 *
 * @param currentCharts 当前周期图集（buildMetricCharts/buildDeviceMetricCharts 产出）
 * @param prevCharts    上一周期图集（同口径，已套同样星期/小时段过滤后转置）
 * @param offsetMs      原始平移毫秒（= 当前窗口长 L，可能带零头）
 * @param granularity   粒度（决定吸附步长 / 日历月平移）— 必传，漏传将退化回原毫秒匹配（bug）
 */
export function attachCompareSeries(
  currentCharts: MetricChart[],
  prevCharts: MetricChart[],
  offsetMs: number,
  granularity: string,
): MetricChart[] {
  const prevByMetric = new Map<string, MetricChart>();
  prevCharts.forEach((p) => prevByMetric.set(p.metricPath, p));
  return currentCharts.map((c) => {
    const p = prevByMetric.get(c.metricPath);
    if (!p) return c;
    const series = buildCompareSeriesValues(c.buckets, p.series, p.buckets, offsetMs, granularity);
    const { compareBuckets, compareBucketEnds } = buildCompareBuckets(
      c.buckets,
      p.buckets,
      p.bucketEnds ?? [],
      offsetMs,
      granularity,
    );
    return { ...c, compareSeries: series, compareBuckets, compareBucketEnds };
  });
}
