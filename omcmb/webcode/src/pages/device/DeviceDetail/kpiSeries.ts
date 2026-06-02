/**
 * T-DEV-KPI 设备详情 KPI 面板「聚合行 → 每指标一条 series」纯映射函数。
 *
 * 输入是聚合接口（/pm/metrics/aggregated）返回的扁平 AggregatedRow[]，
 * 输出按「每个 K 编号一张图、一条曲线」组织：X 轴是时间桶（升序去重），
 * Y 轴是该指标在各时间桶的值（null 占位不画点）。
 *
 * 设计要点（纯逻辑，不依赖 React / 网络，便于 vitest 单测）：
 *   - 按 metricPath（K 编号）分组，配置里给定的指标顺序决定图的顺序。
 *   - objectLdn 过滤：传 null/undefined = 设备级（只取 objectLdn 为空的行）；
 *     传具体 objectLdn = 只取该对象的行。后端不按 objectLdn 过滤，全在前端侧筛。
 *   - 时间轴：取该指标筛选后所有行的 time 升序去重作为 X 轴；同一时间桶
 *     若有多行（理论上过滤后唯一）取最后一个。
 *   - 缺采/占位：某时间桶无该指标数据 → 该点 null（LineChart 不画点，曲线断开）。
 *   - displayName 优先作为线名/标题（后端回填友好名），缺时回退配置中文名。
 */

import type { AggregatedRow } from '@core/types/pmDashboard';

/** 设备级筛选哨兵：objectLdn 为该值或 null/undefined 时只取设备级行。 */
export const DEVICE_LEVEL = null;

/** 一个 K 编号对应的图表数据（一条曲线 + X 轴）。 */
export interface KpiChartData {
  /** K 编号（metricPath），与配置 key 对应。 */
  metricPath: string;
  /** X 轴：时间桶开始时间（ISO 字符串），升序。 */
  xData: string[];
  /** 每个时间桶的结束时间（与 xData 索引对齐，ISO 字符串），用于 tooltip 显示真实起止。 */
  xEnds: string[];
  /** Y 轴：与 xData 对齐，缺采为 null。 */
  values: (number | null)[];
  /** 线名/标题：优先 displayName，缺时回退 fallbackLabel。 */
  displayName: string;
  /** 是否完全无有效数据点（全 null / 空）—— 调用方据此显示「暂无数据」。 */
  isEmpty: boolean;
}

/** 判断一行是否命中目标 objectLdn 过滤条件。 */
function matchObject(row: AggregatedRow, objectLdn: string | null | undefined): boolean {
  const rowLdn = row.objectLdn ?? null;
  if (objectLdn == null || objectLdn === '') {
    // 设备级：只取没有 objectLdn 的行。
    return rowLdn == null || rowLdn === '';
  }
  return rowLdn === objectLdn;
}

/**
 * 把聚合行映射为某一个 K 编号的图表数据。
 *
 * @param rows         聚合接口返回的全部行（可含多指标 / 多对象 / 多时间桶）。
 * @param metricPath   目标 K 编号。
 * @param objectLdn    下钻对象；null/undefined/'' = 设备级。
 * @param fallbackLabel displayName 缺失时的兜底名（配置里的中文名）。
 */
export function buildKpiChartData(
  rows: AggregatedRow[],
  metricPath: string,
  objectLdn: string | null | undefined,
  fallbackLabel: string,
): KpiChartData {
  const matched = rows.filter(
    (r) => r.metricPath === metricPath && matchObject(r, objectLdn),
  );

  // 时间桶升序去重；同桶取最后一条（保留 displayName 与值）。
  const byTime = new Map<string, AggregatedRow>();
  for (const r of matched) {
    byTime.set(r.time, r);
  }
  const xData = Array.from(byTime.keys()).sort((a, b) => a.localeCompare(b));
  const xEnds = xData.map((t) => byTime.get(t)?.endTime ?? '');
  const values = xData.map((t) => {
    const v = byTime.get(t)?.metricValue;
    return v == null ? null : v;
  });

  // 友好名：取第一条有 displayName 的行；缺则回退配置名。
  const displayName = matched.find((r) => r.displayName)?.displayName || fallbackLabel;
  const isEmpty = values.every((v) => v == null);

  return { metricPath, xData, xEnds, values, displayName, isEmpty };
}

/**
 * 按配置顺序，把聚合行映射为多张图（每个 K 编号一张）。
 *
 * @param rows    聚合接口返回的全部行。
 * @param configs 精选 KPI 配置（决定图的数量与顺序、兜底名）。
 * @param objectLdn 下钻对象；null/undefined/'' = 设备级。
 */
export function buildKpiCharts(
  rows: AggregatedRow[],
  configs: Array<{ key: string; label: string }>,
  objectLdn: string | null | undefined,
): KpiChartData[] {
  return configs.map((c) => buildKpiChartData(rows, c.key, objectLdn, c.label));
}

// ─── 周期对比（上一周期虚线对齐）────────────────────────────────────────────
//
// 忠实复刻仪表盘 dashboardFilterUtils.ts 的吸附对齐核心逻辑（来源：buildCompareSeries /
// snapOffsetMs / shiftPrevToCurrentMs / buildCompareBuckets）。仪表盘那套作用于多线
// MetricSeries[]，本面板每张图只有单条 KpiChartData，数据形态不同，故在此复刻而非直接复用，
// 并补单测覆盖「偏移带零头仍精确对齐（虚线非全空）」关键回归（T-0194 踩过的坑）。

/** 固定步长粒度 → 单桶毫秒步长（与 dashboardFilterUtils 口径一致）。 */
const GRAN_STEP_MS: Record<string, number> = {
  '15min': 15 * 60 * 1000,
  hourly: 60 * 60 * 1000,
  daily: 24 * 60 * 60 * 1000,
  weekly: 7 * 24 * 60 * 60 * 1000,
};

/**
 * 把原始平移量 offsetMs（= 当前窗口长 L，可能带亚秒/分钟零头）吸附为整数个粒度步长。
 * 固定步长粒度：snapped = round(offsetMs/step)*step。未知粒度返回 undefined（不吸附，退化为原偏移）。
 * 根因（T-0194）：前后两周期桶都是后端按粒度整点对齐的；带零头偏移平移后落到非整点毫秒，
 * 与当前整点轴永远 miss → 整条虚线全空。吸附到整数步长后再对齐即可精确命中。
 */
function snapOffsetMs(offsetMs: number, granularity: string): number {
  const step = GRAN_STEP_MS[granularity];
  if (step === undefined) return offsetMs;
  return Math.round(offsetMs / step) * step;
}

/** 一张图的上一周期对齐结果：对齐到当前轴的值 + 上周期真实起止（tooltip 用）。 */
export interface KpiCompareData {
  /** 与当前图 xData 索引对齐的上周期值；无对应桶处为 null。 */
  values: (number | null)[];
  /** 与当前图 xData 索引对齐的上周期桶真实开始时间（ISO）；无对应处为空串。 */
  compareBuckets: string[];
  /** 与当前图 xData 索引对齐的上周期桶真实结束时间（ISO）；无对应处为空串。 */
  compareBucketEnds: string[];
  /** 上周期是否完全无有效数据点（全 null / 无对应桶）—— 调用方据此决定是否挂虚线。 */
  isEmpty: boolean;
}

/**
 * 把上一周期某 K 编号的图数据按整数粒度步长平移、对齐到当前周期 X 轴。
 *
 * @param currentChart  当前周期该 K 编号的图（提供对齐目标 X 轴 xData）。
 * @param prevChart     上一周期该 K 编号的图（同 buildKpiChartData 产出，xData 为上周期时间桶）。
 * @param offsetMs      原始平移毫秒（= 当前窗口长 L = end-start，可能带零头）。
 * @param granularity   粒度（决定吸附步长）。
 */
export function buildKpiCompareData(
  currentChart: KpiChartData,
  prevChart: KpiChartData,
  offsetMs: number,
  granularity: string,
): KpiCompareData {
  const snapped = snapOffsetMs(offsetMs, granularity);

  // 上周期桶（平移后毫秒）-> { value, start, end }。
  const shifted = new Map<number, { value: number | null; start: string; end: string }>();
  prevChart.xData.forEach((b, i) => {
    const ms = Date.parse(b);
    if (Number.isNaN(ms)) return;
    shifted.set(ms + snapped, {
      value: prevChart.values[i] ?? null,
      start: b,
      end: prevChart.xEnds[i] ?? '',
    });
  });

  const values: (number | null)[] = [];
  const compareBuckets: string[] = [];
  const compareBucketEnds: string[] = [];
  currentChart.xData.forEach((cb) => {
    const ms = Date.parse(cb);
    const hit = Number.isNaN(ms) ? undefined : shifted.get(ms);
    values.push(hit?.value ?? null);
    compareBuckets.push(hit?.start ?? '');
    compareBucketEnds.push(hit?.end ?? '');
  });

  const isEmpty = values.every((v) => v == null);
  return { values, compareBuckets, compareBucketEnds, isEmpty };
}
