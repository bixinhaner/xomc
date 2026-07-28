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
import utc from 'dayjs/plugin/utc';
import timezone from 'dayjs/plugin/timezone';

dayjs.extend(utc);
dayjs.extend(timezone);
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
 * （与 ChartCard 横轴 MM-DD HH:mm 显示口径一致）。
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

/**
 * T-AXISFILL 横轴铺满后处理入参：
 *   - rangeStartMs / rangeEndMs：所选 SQL 时间窗口（毫秒，[start,end)），界定生成刻度铺到多远。
 *   - weekdays / hours：与 filterRowsByWeekdayHour 同口径的星期/小时筛选集合。
 *   - granularity：粒度（决定步长 / 月日历平移 / 未知退化）。
 */
export interface AxisFillOptions {
  rangeStartMs: number;
  rangeEndMs: number;
  weekdays: Set<number>;
  hours: Set<number>;
  granularity: string;
  /** OMC 系统时区；星期/小时裁剪与空轴钟面都不得依赖浏览器本地时区。 */
  systemTimezone?: string | null;
}

/**
 * 星期 + 小时段裁刻度：与 filterRowsByWeekdayHour 完全同口径（全选短路、否则本地时区 day()/hour()）。
 * 全选（两集合都覆盖全集）→ 直接通过。无效毫秒 → 不通过（跳过）。
 */
function inSystemTimezone(ms: number, systemTimezone?: string | null): Dayjs {
  try {
    return dayjs(ms).tz(systemTimezone?.trim() || 'UTC');
  } catch {
    return dayjs(ms).utc();
  }
}

/**
 * 在 OMC 系统时区的墙上日历中步进，并重新按同一 IANA zone 解析目标钟面。
 * dayjs.tz(...).add(...) 会保留旧 offset；跨 DST 后必须重解析，否则 00:00 会漂移。
 */
function addSystemCalendar(
  ms: number,
  amount: number,
  unit: 'day' | 'month',
  systemTimezone?: string | null,
): number {
  const zone = systemTimezone?.trim() || 'UTC';
  try {
    const shiftedWall = dayjs(ms).tz(zone).add(amount, unit).format('YYYY-MM-DDTHH:mm:ss.SSS');
    return dayjs.tz(shiftedWall, zone).valueOf();
  } catch {
    const shiftedWall = dayjs(ms).utc().add(amount, unit).format('YYYY-MM-DDTHH:mm:ss.SSS');
    return dayjs.tz(shiftedWall, 'UTC').valueOf();
  }
}

function passesWeekdayHour(
  ms: number,
  weekdays: Set<number>,
  hours: Set<number>,
  systemTimezone?: string | null,
): boolean {
  const allWeekdays = weekdays.size >= 7;
  const allHours = hours.size >= 24;
  if (allWeekdays && allHours) return true;
  const d = inSystemTimezone(ms, systemTimezone);
  if (!d.isValid()) return false;
  if (!allWeekdays && !weekdays.has(d.day())) return false;
  if (!allHours && !hours.has(d.hour())) return false;
  return true;
}

/**
 * T-AXISFILL 单图横轴铺满（纯后处理，不动转置）：把横轴从"有数据才上轴"扩成
 * "按范围+粒度连续推出全部应有刻度、套星期/小时筛选裁、并集真实桶"，空刻度补 '-'。
 *
 * 算法（spec §4.2 / 账本设计决策 8 步）：
 *   1. 空图（buckets 为空）→ 以查询窗口起点为相位锚点生成轴，让配置内无结果指标仍有空图。
 *   2. 建真实桶毫秒 → 原索引映射（无效串跳过）。
 *   3. 全无效串 → 原样返回（防 Math.min 空集得 Infinity）。
 *   4. 相位锚点 = 全部真实桶 ms 的最小值（最早真实桶），生成刻度天然同相位。
 *   5. 按 [start,end) 连续生成刻度（已套范围 + 星期/小时裁）：
 *      - 固定步长粒度：先算覆盖 rangeStart 的起始 k（可负），逐步加 step 到超 rangeEnd。
 *      - monthly：以锚点为基准 dayjs(anchorMs).add(k,'month') 双向平移覆盖范围。
 *      - 未知粒度：不生成（退化为仅真实桶，安全兜底）。
 *   6. 并集兜底：最终轴 ms = 生成(已裁、end-exclusive) ∪ 全部真实桶 ms，
 *      升序去重（历史真实桶即使异常落在 end 也不静默丢弃）。
 *   7. 按新轴 ms 重对齐 buckets/bucketEnds/各线 values：命中真实→原桶串/原 end/原值；
 *      空刻度→桶串=dayjs(ms).toISOString()、end=下一刻度ms 或 ms+step（monthly add(1,'month')）的 ISO、各线值 '-'。
 *   8. 返回 {...chart, buckets, bucketEnds, series}（不动 compare 字段，pipeline 保证扩轴在 attach 之前）。
 */
export function extendChartAxis(chart: MetricChart, opts: AxisFillOptions): MetricChart {
  const {
    rangeStartMs,
    rangeEndMs,
    weekdays,
    hours,
    granularity,
    systemTimezone,
  } = opts;

  // 1-2. 真实桶 ms → 原索引；完全无桶时后续以查询窗口起点为可信相位。
  const realMsToIdx = new Map<number, number>();
  chart.buckets.forEach((b, i) => {
    const ms = dayjs(b).valueOf();
    if (Number.isNaN(ms)) return;
    if (!realMsToIdx.has(ms)) realMsToIdx.set(ms, i);
  });

  // 3. 有桶但全部是无效时间串时原样返回，避免用窗口轴掩盖坏数据。
  if (chart.buckets.length > 0 && realMsToIdx.size === 0) return chart;

  // 4. 相位锚点 = 最早真实桶；完全无结果时使用查询窗口起点。
  const anchorMs = realMsToIdx.size > 0
    ? Math.min(...realMsToIdx.keys())
    : rangeStartMs;

  // 15min/hourly 是绝对时长；daily/weekly/monthly 必须按系统时区墙上日历推进。
  const fixedStep =
    granularity === '15min' || granularity === 'hourly'
      ? granularityStepMs(granularity)
      : undefined;
  const calendarStep =
    granularity === 'daily'
      ? { amount: 1, unit: 'day' as const }
      : granularity === 'weekly'
        ? { amount: 7, unit: 'day' as const }
        : granularity === 'monthly'
          ? { amount: 1, unit: 'month' as const }
          : undefined;

  // 5. 连续生成刻度（已套范围 + 星期/小时裁）。
  const generated = new Set<number>();
  const generatedNext = new Map<number, number>();
  const addIfPasses = (ms: number) => {
    // 与 results SQL 保持 [start,end)：end 只表示窗口边界，不是一个可出图桶。
    if (ms < rangeStartMs || ms >= rangeEndMs) return;
    if (!passesWeekdayHour(ms, weekdays, hours, systemTimezone)) return;
    generated.add(ms);
  };
  if (fixedStep !== undefined) {
    // 固定步长粒度：先算覆盖 rangeStart 的起始 k（可负）。
    const kStart = Math.floor((rangeStartMs - anchorMs) / fixedStep);
    for (let ms = anchorMs + kStart * fixedStep; ms < rangeEndMs; ms += fixedStep) {
      addIfPasses(ms);
      if (generated.has(ms)) generatedNext.set(ms, ms + fixedStep);
    }
  } else if (calendarStep) {
    let index = 0;
    let cursor = anchorMs;
    while (cursor > rangeStartMs) {
      index -= 1;
      const previous = addSystemCalendar(
        anchorMs,
        index * calendarStep.amount,
        calendarStep.unit,
        systemTimezone,
      );
      if (previous >= cursor) break;
      cursor = previous;
    }
    while (cursor <= rangeStartMs) {
      const next = addSystemCalendar(
        anchorMs,
        (index + 1) * calendarStep.amount,
        calendarStep.unit,
        systemTimezone,
      );
      if (next > rangeStartMs || next <= cursor) break;
      index += 1;
      cursor = next;
    }
    while (cursor < rangeEndMs) {
      addIfPasses(cursor);
      index += 1;
      const next = addSystemCalendar(
        anchorMs,
        index * calendarStep.amount,
        calendarStep.unit,
        systemTimezone,
      );
      if (next <= cursor) break;
      if (generated.has(cursor)) generatedNext.set(cursor, next);
      cursor = next;
    }
  }
  // 未知粒度：不生成，退化为仅真实桶。

  // 6. 并集兜底：生成(已裁、end-exclusive) ∪ 全部真实桶，升序去重。
  // 真实桶来自后端，哪怕是历史异常边界值也不静默丢弃；[start,end) 只约束前端生成的空桶。
  const axisMs = Array.from(new Set<number>([...generated, ...realMsToIdx.keys()])).sort(
    (a, b) => a - b,
  );

  // 6.5 从第一个有效真实桶字符串提取 UTC offset 分钟数，用于空刻度保持同时区。
  const firstRealBucket = chart.buckets.find((b) => dayjs(b).isValid());
  let fillOffsetMin = 0;
  if (firstRealBucket) {
    const tzMatch = firstRealBucket.match(/([+-])(\d{2}):(\d{2})$/);
    if (tzMatch) {
      const sign = tzMatch[1] === '+' ? 1 : -1;
      fillOffsetMin = sign * (parseInt(tzMatch[2], 10) * 60 + parseInt(tzMatch[3], 10));
    }
    // 'Z' 后缀 → offset 0（默认值已满足）
  }
  const formatGeneratedTime = (ms: number): string => {
    if (systemTimezone?.trim()) {
      return inSystemTimezone(ms, systemTimezone).format('YYYY-MM-DDTHH:mm:ssZ');
    }
    return dayjs(ms).utcOffset(fillOffsetMin).format('YYYY-MM-DDTHH:mm:ssZ');
  };

  // 7. 按新轴 ms 重对齐。
  const nextStepEndMs = (ms: number, idx: number): number => {
    // synthetic 桶的结束必须是自身粒度边界；星期/小时裁剪后的下一个可见桶可能隔数日。
    const generatedBucketEnd = generatedNext.get(ms);
    if (generatedBucketEnd !== undefined) return generatedBucketEnd;
    // 非 synthetic（真实桶）沿用原有相邻轴/粒度 fallback 语义。
    const next = axisMs[idx + 1];
    if (next !== undefined) return next;
    if (fixedStep !== undefined) return ms + fixedStep;
    if (calendarStep) {
      return addSystemCalendar(
        ms,
        calendarStep.amount,
        calendarStep.unit,
        systemTimezone,
      );
    }
    return ms + APPROX_MONTH_MS;
  };
  const buckets: string[] = [];
  const bucketEnds: string[] = [];
  axisMs.forEach((ms, idx) => {
    const realIdx = realMsToIdx.get(ms);
    if (realIdx !== undefined) {
      buckets.push(chart.buckets[realIdx]);
      bucketEnds.push(chart.bucketEnds[realIdx] ?? '');
    } else {
      buckets.push(formatGeneratedTime(ms));
      bucketEnds.push(formatGeneratedTime(nextStepEndMs(ms, idx)));
    }
  });
  const series: MetricSeries[] = chart.series.map((s) => ({
    ...s,
    values: axisMs.map((ms) => {
      const realIdx = realMsToIdx.get(ms);
      return realIdx !== undefined ? (s.values[realIdx] ?? '-') : '-';
    }),
  }));

  // 8. 返回（不动 compare 字段）。
  return { ...chart, buckets, bucketEnds, series };
}

/** T-AXISFILL 批量横轴铺满：对每张图调 extendChartAxis。 */
export function extendChartsAxis(charts: MetricChart[], opts: AxisFillOptions): MetricChart[] {
  return charts.map((c) => extendChartAxis(c, opts));
}
