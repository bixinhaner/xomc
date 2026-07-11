/**
 * issue #429 PM 趋势图断档断开线 —— 共享"规整时间网格"工具。
 *
 * 按“查询窗口起止 + 粒度”在窗口内铺满每个整点槽位（含没数的占位），供
 * 皮肤把稀疏的有数桶对齐到规整网格、空槽置 null、connectNulls=false 断开连线。
 *
 * 必须按"查询窗口起止"铺（而非数据 min/max），否则窗口尾部连续断档铺不出占位槽。
 *
 * v1（webcode）已有自成体系的 extendChartsAxis（耦合 MetricChart 结构 + 星期/小时筛选），
 * 本工具不替代它，仅服务无筛选感知的查询场景。
 */

/** 支持的固定步长粒度（与后端 metrics.Granularity 一致的子集）。 */
export type GridGranularity = '15min' | 'hourly' | 'daily';

const GRID_STEP_MS: Record<GridGranularity, number> = {
  '15min': 15 * 60 * 1000, // 900000
  hourly: 60 * 60 * 1000, // 3600000
  daily: 24 * 60 * 60 * 1000, // 86400000
};

/**
 * 按粒度返回步长（毫秒）。未知粒度返回 undefined（调用方应回退为仅原始桶）。
 */
export function gridStepMs(granularity: string): number | undefined {
  return GRID_STEP_MS[granularity as GridGranularity];
}

/**
 * 生成窗口内规整时间槽位（毫秒数组，升序）。
 *
 * 槽位从 startMs 起、按 step 递增，直到 > endMs 为止。半开区间 [startMs, endMs]：
 * 含起点、最后一个 <= endMs 的槽位也含。窗口非法（start>=end / step 缺失 / 非数）→ 返回 []。
 *
 * @param startMs 查询窗口起（毫秒）
 * @param endMs   查询窗口止（毫秒）
 * @param granularity 粒度（15min/hourly/daily）
 */
export function buildRegularTimeGrid(
  startMs: number,
  endMs: number,
  granularity: string,
): number[] {
  const step = gridStepMs(granularity);
  if (step === undefined) return [];
  if (!Number.isFinite(startMs) || !Number.isFinite(endMs)) return [];
  if (endMs < startMs) return [];
  const slots: number[] = [];
  for (let ms = startMs; ms <= endMs; ms += step) {
    slots.push(ms);
  }
  return slots;
}

/**
 * 把稀疏的"时间→值"点按规整网格对齐成 (number|null)[]。
 *
 * 网格按 buildRegularTimeGrid(startMs,endMs,granularity) 铺；每个槽位取离它最近的、
 * 落在 [slot, slot+step) 半开窗内的有数点，无则置 null（缺采 = 断档）。
 *
 * 桶时间用其毫秒值归桶：对每个有数点，算它落到哪个槽位（floor 对齐到 step 网格相位）。
 * 网格相位以 startMs 为锚，确保后端已对齐到整点窗口的桶能命中槽位。
 *
 * @param points 稀疏点（timeMs 毫秒 + value 数值，value 可为 null 表示该桶占位/缺采）
 * @param startMs 窗口起（毫秒）
 * @param endMs 窗口止（毫秒）
 * @param granularity 粒度
 * @returns { grid: 槽位毫秒数组; values: 与 grid 等长的 (number|null)[] }
 */
export function alignPointsToGrid(
  points: { timeMs: number; value: number | null }[],
  startMs: number,
  endMs: number,
  granularity: string,
): { grid: number[]; values: (number | null)[] } {
  const grid = buildRegularTimeGrid(startMs, endMs, granularity);
  const step = gridStepMs(granularity);
  const values: (number | null)[] = new Array(grid.length).fill(null);
  if (grid.length === 0 || step === undefined) return { grid, values };
  // 槽位毫秒 → 槽位索引，便于 O(1) 命中。
  const slotIndex = new Map<number, number>();
  grid.forEach((ms, i) => slotIndex.set(ms, i));
  for (const p of points) {
    if (p.value === null || p.value === undefined) continue;
    if (!Number.isFinite(p.timeMs)) continue;
    if (p.timeMs < startMs || p.timeMs > endMs) continue;
    // 把点对齐到所属槽位（以 startMs 为相位锚，floor 到 step 网格）。
    const k = Math.floor((p.timeMs - startMs) / step);
    const slotMs = startMs + k * step;
    const idx = slotIndex.get(slotMs);
    if (idx !== undefined) values[idx] = p.value;
  }
  return { grid, values };
}
