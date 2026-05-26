/**
 * PmDashboard 模块工具函数。
 *
 * - csv ↔ array：兼容旧表单"逗号分隔字符串"字段与新 picker 的 string[] 双向转换
 * - techToDeviceType：把 dashboard.technology (lte/nr/gsm) 映射到 IndicatorLibrary 的 DeviceType (ENB/GNB/GSM)
 * - describeCompareMode：按 windowOffset 动态描述对比含义（Panel 标题旁的问号 tooltip 用）
 */

import type { CompareMode, Technology } from '@core/types/pmDashboard';
import type { DeviceType } from '@core/types/indicatorLibrary';

export const csvToArray = (csv: string | undefined | null): string[] =>
  (csv ?? '')
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean);

export const arrayToCsv = (arr: string[] | undefined | null): string =>
  (arr ?? []).join(', ');

export const techToDeviceType = (tech: Technology | undefined): DeviceType => {
  switch (tech) {
    case 'nr':
      return 'GNB';
    case 'gsm':
      return 'GSM';
    case 'lte':
    default:
      return 'ENB';
  }
};

/**
 * 按 compareMode + windowOffset 动态生成对比含义说明文案。
 * - 'none' / undefined → null（panel 标题不显示问号）
 * - 'previous_window' + 已知 offset → "对比昨天同时段" 等
 * - 'previous_window' + 未知 offset → 兜底文案
 * - 'same_window_other_devices' → "已废弃"提示（兼容旧数据）
 */
export function describeCompareMode(
  compareMode: CompareMode | undefined | null,
  windowOffset: string | undefined | null,
): string | null {
  if (!compareMode) return null;
  if (compareMode === 'same_window_other_devices') {
    return '【已废弃】此对比模式已下线，请重新配置 Panel';
  }
  const offsetMap: Record<string, string> = {
    '-1h': '对比前 1 小时',
    '-24h': '对比昨天同时段',
    '-7d': '对比上周同期',
    '-30d': '对比上月同期',
  };
  return offsetMap[windowOffset ?? ''] ?? `对比上一周期（偏移 ${windowOffset ?? '未知'}）`;
}
