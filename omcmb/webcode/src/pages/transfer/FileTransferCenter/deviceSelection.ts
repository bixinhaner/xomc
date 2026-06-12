import type { UnifiedFileTransferDeviceItem } from '@core/types/unifiedFileTransfer';

/**
 * #215 设备选择的纯逻辑辅助：抽出便于单测，且让真分页下的「跨页已选」与
 * 「固件升级选机不串机型」两件事可被回归守护。组件内的 effect/handler 复用这些
 * 函数，确保被测逻辑即线上逻辑。
 */

export type SelectedDeviceMap = Record<string, UnifiedFileTransferDeviceItem>;

/**
 * 把当前页拉到的候选设备并入 selectedDeviceMap（只补设备对象，不动已选 id 列表）。
 * 引用语义与原 inline 实现一致：无变化时原样返回 prev，避免无谓重渲染。
 */
export function mergeCandidatesIntoMap(
  prev: SelectedDeviceMap,
  candidates: readonly UnifiedFileTransferDeviceItem[],
): SelectedDeviceMap {
  if (candidates.length === 0) {
    return prev;
  }
  let changed = false;
  const next = { ...prev };
  for (const item of candidates) {
    if (next[item.id] !== item) {
      next[item.id] = item;
      changed = true;
    }
  }
  return changed ? next : prev;
}

/** 从 selectedDeviceMap 删一台设备；不存在时原样返回 prev（保持引用稳定）。 */
export function removeFromMap(prev: SelectedDeviceMap, deviceId: string): SelectedDeviceMap {
  if (!(deviceId in prev)) {
    return prev;
  }
  const next = { ...prev };
  delete next[deviceId];
  return next;
}

/**
 * #215 回归守护：固件升级类任务是「按机型（productType）」的，换了 productClass
 * 筛选后，旧的不同机型已选设备必须剔除，否则升级会下发到不兼容机型。
 *
 * 关键点：以 selectedDeviceMap 里记录的设备对象判定，而不是「当前页候选集」——
 * 这样翻页不会误删其它页的合法已选（修复前的旧实现正是按当前页候选裁剪才丢选）。
 * 对 map 里查不到对象、或 productType 为空的 id，无法证明其不兼容，保守保留。
 */
export function pruneSelectionForProductClass(
  ids: readonly string[],
  deviceMap: SelectedDeviceMap,
  productClass: string | undefined | null,
): { kept: string[]; droppedCount: number } {
  if (!productClass) {
    return { kept: [...ids], droppedCount: 0 };
  }
  const kept = ids.filter((id) => {
    const dev = deviceMap[id];
    if (!dev || !dev.productType) {
      return true; // 未知机型：保守保留
    }
    return dev.productType === productClass;
  });
  return { kept, droppedCount: ids.length - kept.length };
}
