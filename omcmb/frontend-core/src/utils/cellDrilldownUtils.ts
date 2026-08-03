/**
 * T-0193 下钻选择器纯逻辑（与 React 解耦，供页面复用）。
 */

import { hasObjectLdnField, parseObjectLdn } from '../types/pmObject';
import type { MetricObject } from '../types/pmObject';

/** 每设备选中的 object_ldn 子集；某设备缺席 = 该设备全选（不过滤）。 */
export type CellSelection = Record<string, string[]>;

/**
 * #241：5G 建议默认只勾两类 object_ldn。
 * - 设备级：Type=gNB
 * - 小区 + PLMN 级：Type=Cell 且存在 PLMNID 字段
 *
 * 只判断字段名和 Type 类别，不绑定 Mode/gNBID/NrCGI/PLMNID 的具体值。
 */
export function isRecommendedNrObjectLdn(objectLdn: string | null | undefined): boolean {
  const fields = parseObjectLdn(objectLdn);
  const objectType = fields.objectType?.toLowerCase();
  if (objectType === 'gnb') return true;
  return objectType === 'cell' && hasObjectLdnField(objectLdn, 'PLMNID');
}

/** 未手动选择时的推荐勾选值：5G 有推荐子集则用子集；没有推荐项则全选不过滤。 */
export function getNrRecommendedDefaultSelectedObjectLdns(objects: MetricObject[]): string[] {
  const all = objects.map((o) => o.objectLdn);
  const recommendedNr = all.filter(isRecommendedNrObjectLdn);
  return recommendedNr.length > 0 ? recommendedNr : all;
}

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

/**
 * #241 专用：在 5G 建议模板默认态下，未手动选择的设备使用推荐子集；
 * 已手动选择的设备仍尊重用户选择。用于“设备性能查看”和“指标查询”，不改变其它入口旧语义。
 */
export function getEffectiveLdnsWithNrRecommendedDefault(
  value: CellSelection,
  objectsByDevice: Record<string, MetricObject[]>,
): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  Object.entries(objectsByDevice).forEach(([sn, objs]) => {
    const all = objs.map((o) => o.objectLdn);
    const sel = value[sn] ?? getNrRecommendedDefaultSelectedObjectLdns(objs);
    if (sel.length === 0 || sel.length >= all.length) return; // 全选不过滤
    sel.forEach((ldn) => {
      if (!seen.has(ldn)) {
        seen.add(ldn);
        out.push(ldn);
      }
    });
  });
  return out;
}
