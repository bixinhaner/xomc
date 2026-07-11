/**
 * T-0193 下钻选择器纯逻辑（与 React 解耦，供页面复用）。
 */

import type { MetricObject } from '../types/pmObject';

/** 每设备选中的 object_ldn 子集；某设备缺席 = 该设备全选（不过滤）。 */
export type CellSelection = Record<string, string[]>;

/**
 * 把「按设备的小区清单」+「当前选择」汇成最终生效白名单（拍平、去重）。
 * - 某设备缺席 / 选中数==该设备全部小区数 / 空选 → 视为全选，不贡献过滤项。
 * - 子集设备 → 贡献其选中的 object_ldn。
 * - 全部设备都全选 → 返回空数组（= 不过滤，向后兼容）。
 */
export function getEffectiveLdns(
  value: CellSelection,
  objectsByDevice: Record<string, MetricObject[]>,
): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  Object.entries(objectsByDevice).forEach(([sn, objs]) => {
    const all = objs.map((o) => o.objectLdn);
    const sel = value[sn];
    if (!sel || sel.length === 0 || sel.length >= all.length) return; // 全选不过滤
    sel.forEach((ldn) => {
      if (!seen.has(ldn)) {
        seen.add(ldn);
        out.push(ldn);
      }
    });
  });
  return out;
}
