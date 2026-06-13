/**
 * #200 多设备/多指标 KPI 时间轴并集对齐。
 *
 * 背景：MultiKPITrendChart 旧逻辑只取「第一个有数据的 series」的时间点当 xData，
 * 其余 series 按数组下标 zip。多设备/多指标时间点稀疏且错位时，朴素 zip 会把不同
 * 时刻的值塞进同一列 → 曲线时间错位、相邻点连不成线。
 *
 * 本函数把所有 series 的时间点取并集（去重 + 升序）作为统一 xAxis，再把每个 series
 * 按时间值对齐：命中并集某时刻则取其值，缺失则填 null。保证每个 series 的值数组长度
 * 严格等于 xData 长度，下游 ECharts 配合 connectNulls 可跨缺采点续连。
 *
 * 纯函数、无副作用，便于单测。
 */

export interface TimePoint {
  time: string;
  value: number;
}

export interface UnionTimeAxis {
  /** 全部 series 时间点的并集（去重 + 升序），作为 ECharts xAxis.data */
  xData: string[];
  /**
   * 与入参 seriesList 一一对应的对齐后值数组。
   * values[i].length === xData.length，缺失时刻填 null。
   */
  values: Array<Array<number | null>>;
}

/**
 * 把多个稀疏/错位的时间序列对齐到统一时间轴并集。
 *
 * @param seriesList 多个 series，每个为 { time, value }[]（时间无需有序，可稀疏）
 * @returns { xData, values } —— xData 为并集升序时间点；values 与 seriesList 一一对应
 */
export function buildUnionTimeAxis(seriesList: TimePoint[][]): UnionTimeAxis {
  if (!seriesList || seriesList.length === 0) {
    return { xData: [], values: [] };
  }

  // 1) 收集所有时间点并集（去重）。同时为每个 series 建 time → value 映射，
  //    同一 series 内同时间点重复时后者覆盖前者。
  const allTimes = new Set<string>();
  const perSeriesMap: Array<Map<string, number>> = seriesList.map((series) => {
    const map = new Map<string, number>();
    (series ?? []).forEach((p) => {
      allTimes.add(p.time);
      map.set(p.time, p.value);
    });
    return map;
  });

  // 2) 并集升序（字符串字典序；ISO/RFC3339 时间字典序即时间序）。
  const xData = Array.from(allTimes).sort((a, b) => (a < b ? -1 : a > b ? 1 : 0));

  // 3) 每个 series 按并集对齐：命中取值，缺失填 null。长度恒 == xData.length。
  const values = perSeriesMap.map((map) =>
    xData.map((t) => (map.has(t) ? map.get(t)! : null)),
  );

  return { xData, values };
}
