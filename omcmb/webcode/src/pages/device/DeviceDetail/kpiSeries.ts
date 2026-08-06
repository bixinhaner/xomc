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
 *   - 缺采/占位：某时间桶无该指标数据 → 该点 null；页面与首页一致跨空桶连接相邻有效点。
 *   - displayName 优先作为线名/标题（后端回填友好名），缺时回退配置中文名。
 */

import type { AggregatedRow } from '@core/types/pmDashboard';
import { normalizePmMetricValue } from '@core/utils/pmMetricValue';

/** 设备级筛选哨兵：objectLdn 为该值或 null/undefined 时只取设备级行。 */
export const DEVICE_LEVEL = null;

/** 设备级行的空 ldn 哨兵序列名（多对象 series 中区别「设备级」与「有 ldn 对象」，留空串为 machine-readable 哨兵。 */
export const DEVICE_LEVEL_LDN = '';

/** 多对象：一个 metricPath 下按 objectLdn 拆出的一条 series。 */
export interface KpiObjectSeries {
  /** objectLdn 原字符串；'' = 设备级行（metricObjects 不包含空 ldn，UI 会手动附加）。 */
  objectLdn: string;
  /** 显示名：优先该对象首行 displayName，其次 objectLdn 原串，均缺时回退 fallback。 */
  name: string;
  /** Y 轴：与所在图的 xData 对齐，未命中为 null。 */
  values: (number | null)[];
}

/** 一个 K 编号对应的图表数据（多对象时多条 series，单对象时 series 长度为 1）。 */
export interface KpiChartData {
  /** K 编号（metricPath），与配置 key 对应。 */
  metricPath: string;
  /** X 轴：时间桶开始时间（ISO 字符串），升序；多对象时为所有对象的时间桶并集。 */
  xData: string[];
  /** 每个时间桶的结束时间（与 xData 索引对齐，ISO 字符串），用于 tooltip 显示真实起止。 */
  xEnds: string[];
  /**
   * 多对象 series。单对象/设备级时长度 = 1。
   * 不在纯函数侧过滤「全 null 对象」——调用方（UI）按 series.values.every(v => v == null) 决定是否不出线。
   */
  series: KpiObjectSeries[];
  /** 向后兼容的首条线值（= series[0]?.values ∥ 空数组），仅为保留老调用方/测试断言可用。 */
  values: (number | null)[];
  /** 图标题：优先首行有值的 displayName，缺时回退 fallbackLabel。 */
  displayName: string;
  /** 是否完全无有效数据点（所有 series 全 null / 空）—— 调用方据此显示「暂无数据」。 */
  isEmpty: boolean;
  /** 是否存在 PM 结果行；全缺值时该字段为 true，但 isEmpty 仍为 true。 */
  hasSamples: boolean;
}

/** 入参归一：将 objectLdn 参数归一为「对象 ldn 集合」。 null/undefined/'' 与空数组均作「设备级单对象」老语义。 */
function normalizeObjectLdns(input: string | string[] | null | undefined): string[] {
  if (input == null) return [DEVICE_LEVEL_LDN];
  if (typeof input === 'string') return [input];
  if (input.length === 0) return [DEVICE_LEVEL_LDN];
  return input;
}

/** 判断一行是否命中目标 objectLdn 集合。集合含 '' 时设备级行（rowLdn 为空）命中。 */
function matchObject(row: AggregatedRow, objectLdns: string[]): boolean {
  const rowLdn = row.objectLdn ?? '';
  return objectLdns.includes(rowLdn);
}

/**
 * 把聚合行映射为某一个 K 编号的图表数据。
 *
 * @param rows         聚合接口返回的全部行（可含多指标 / 多对象 / 多时间桶）。
 * @param metricPath   目标 K 编号。
 * @param objectLdn    下钻对象；string/null/undefined/'' = 单对象老语义；string[] = 多对象（每对象一条 series）。
 * @param fallbackLabel displayName 缺失时的兜底名（配置里的中文名）。
 */
export function buildKpiChartData(
  rows: AggregatedRow[],
  metricPath: string,
  objectLdn: string | string[] | null | undefined,
  fallbackLabel: string,
): KpiChartData {
  const objectLdns = normalizeObjectLdns(objectLdn);

  // 一次扫描：筛 metric + 命中 ldn 集合的行。
  const matched = rows.filter(
    (r) => r.metricPath === metricPath && matchObject(r, objectLdns),
  );

  // X 轴 = 所有 ldn 的时间桶并集（升序去重）；同桶 endTime 以首条出现的为准。
  const endsByTime = new Map<string, string>();
  for (const r of matched) {
    if (!endsByTime.has(r.time)) endsByTime.set(r.time, r.endTime ?? '');
  }
  const xData = Array.from(endsByTime.keys()).sort((a, b) => a.localeCompare(b));
  const xEnds = xData.map((t) => endsByTime.get(t) ?? '');

  // 按对象拆出多条 series，与 xData 对齐；缺对应桶为 null。
  const series: KpiObjectSeries[] = objectLdns.map((ldn) => {
    const objRows = matched.filter((r) => (r.objectLdn ?? '') === ldn);
    const byTime = new Map<string, AggregatedRow>();
    for (const r of objRows) byTime.set(r.time, r); // 同桶重复以最后一条为准。
    const values = xData.map((t) => {
      const v = byTime.get(t)?.metricValue;
      return normalizePmMetricValue(v);
    });
    // legend / series.name 拍板决定：指名重现原始完整 LDN，便于现场对照；
    // 只有设备级行（ldn=''）才回退 fallback（空串不能当 legend 名）。
    // 留意：故意不走 row.displayName，避免后端友好名（如「下行用户平均速率」）顶掉 LDN 区分力。
    const name = ldn !== DEVICE_LEVEL_LDN ? ldn : fallbackLabel;
    return { objectLdn: ldn, name, values };
  });

  // 图标题友好名：仅从首个有 displayName 的行取，缺时回退 fallback。
  const displayName = matched.find((r) => r.displayName)?.displayName || fallbackLabel;
  const hasSamples = matched.length > 0;
  const isEmpty = series.every((s) => s.values.every((v) => v == null));
  // 向后兼容首条 series 值作为 chart.values。
  const values = series[0]?.values ?? [];

  return { metricPath, xData, xEnds, series, values, displayName, isEmpty, hasSamples };
}

/**
 * 按配置顺序，把聚合行映射为多张图（每个 K 编号一张）。
 *
 * @param rows    聚合接口返回的全部行。
 * @param configs 精选 KPI 配置（决定图的数量与顺序、兜底名）。
 * @param objectLdn 下钻对象；string/null/'' = 单对象老语义；string[] = 多对象。
 */
export function buildKpiCharts(
  rows: AggregatedRow[],
  configs: Array<{ key: string; label: string }>,
  objectLdn: string | string[] | null | undefined,
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

/** 一条对象的上周期对齐结果。 */
export interface KpiCompareSeries {
  /** 对应 currentChart.series[i].objectLdn。 */
  objectLdn: string;
  /** 与当前图 xData 索引对齐的上周期值；无对应桶处为 null。 */
  values: (number | null)[];
  /** 与当前图 xData 索引对齐的上周期桶真实开始时间（ISO）；无对应处为空串。 */
  compareBuckets: string[];
  /** 与当前图 xData 索引对齐的上周期桶真实结束时间（ISO）；无对应处为空串。 */
  compareBucketEnds: string[];
}

/** 一张图的上一周期对齐结果：多对象时多条 series；向后兼容首条别名字段。 */
export interface KpiCompareData {
  /** 每对象一条对齐结果，与 currentChart.series 索引对齐。 */
  series: KpiCompareSeries[];
  /** 向后兼容：首条 series 的值（单对象场景不变）。 */
  values: (number | null)[];
  /** 向后兼容：首条 series 的开始时间。 */
  compareBuckets: string[];
  /** 向后兼容：首条 series 的结束时间。 */
  compareBucketEnds: string[];
  /** 上周期是否完全无有效数据点（所有 series 全 null / 无对应桶）—— 调用方据此决定是否挂虚线。 */
  isEmpty: boolean;
}

/**
 * 把上一周期某 K 编号的图数据按整数粒度步长平移、对齐到当前周期 X 轴。
 * 多对象场景下按 currentChart.series 的 objectLdn 逐条靠同名 ldn 查 prevChart 同一对象的上周期值，
 * 避免「对象 A 的虚线拿了对象 B 的上周期值」二维错位。
 *
 * @param currentChart  当前周期该 K 编号的图（提供对齐目标 X 轴 + series 的 ldn 顺序）。
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

  // 预构：上周期按 ldn -> Map<平移后毫秒, {value,start,end}>，只需扫一遍 prevChart。
  const prevByLdn = new Map<
    string,
    Map<number, { value: number | null; start: string; end: string }>
  >();
  const prevSeriesArr = prevChart.series ?? [];
  prevSeriesArr.forEach((ps) => {
    const shifted = new Map<number, { value: number | null; start: string; end: string }>();
    prevChart.xData.forEach((b, i) => {
      const ms = Date.parse(b);
      if (Number.isNaN(ms)) return;
      shifted.set(ms + snapped, {
        value: ps.values[i] ?? null,
        start: b,
        end: prevChart.xEnds[i] ?? '',
      });
    });
    prevByLdn.set(ps.objectLdn, shifted);
  });

  // 当前图 × series 逐对象对齐。老调用方或测试 helper 可能在 currentChart 上未填 series，
  // 退化为「单设备级虚拟 series（ldn=''）」，保证 buildKpiCompareData 输出同形状。
  const currentSeriesArr =
    currentChart.series && currentChart.series.length > 0
      ? currentChart.series
      : [{ objectLdn: DEVICE_LEVEL_LDN, name: '', values: currentChart.values }];
  const series: KpiCompareSeries[] = currentSeriesArr.map((cs) => {
    const shifted = prevByLdn.get(cs.objectLdn);
    const values: (number | null)[] = [];
    const compareBuckets: string[] = [];
    const compareBucketEnds: string[] = [];
    currentChart.xData.forEach((cb) => {
      const ms = Date.parse(cb);
      const hit = !shifted || Number.isNaN(ms) ? undefined : shifted.get(ms);
      values.push(hit?.value ?? null);
      compareBuckets.push(hit?.start ?? '');
      compareBucketEnds.push(hit?.end ?? '');
    });
    return { objectLdn: cs.objectLdn, values, compareBuckets, compareBucketEnds };
  });

  const isEmpty = series.every((s) => s.values.every((v) => v == null));
  const head = series[0];
  return {
    series,
    values: head?.values ?? [],
    compareBuckets: head?.compareBuckets ?? [],
    compareBucketEnds: head?.compareBucketEnds ?? [],
    isEmpty,
  };
}
