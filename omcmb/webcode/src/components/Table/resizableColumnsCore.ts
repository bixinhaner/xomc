/**
 * 「可拖拽列宽」的纯逻辑核心（#214）—— 不依赖 React / react-resizable，便于单测。
 * React 组件 + hook 在同目录 resizableColumns.tsx，复用本文件。
 */

export const MIN_RESIZE_WIDTH = 60;
export const RESIZABLE_TABLE_CLASS = 'omc-resizable-table';

export type ColWidthMap = Record<string, number>;

/**
 * 把某列宽度设为 width（夹到最小宽、四舍五入），返回新 map（不可变）。
 */
export function applyColumnResize(
  prev: ColWidthMap,
  key: string,
  width: number,
  min: number = MIN_RESIZE_WIDTH,
): ColWidthMap {
  return { ...prev, [key]: Math.max(min, Math.round(width)) };
}

export function loadWidths(storageKey?: string): ColWidthMap {
  if (!storageKey) return {};
  try {
    const raw = localStorage.getItem(storageKey);
    if (!raw) return {};
    const parsed = JSON.parse(raw) as unknown;
    if (parsed && typeof parsed === 'object') return parsed as ColWidthMap;
  } catch {
    // localStorage 不可用 / 内容损坏 → 退回默认列宽
  }
  return {};
}

export function saveWidths(storageKey: string | undefined, widths: ColWidthMap): void {
  if (!storageKey) return;
  try {
    localStorage.setItem(storageKey, JSON.stringify(widths));
  } catch {
    // 持久化失败不影响本次拖拽生效（仅丢失下次保留）
  }
}
