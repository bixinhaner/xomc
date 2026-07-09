import * as XLSX from 'xlsx';
import {
  VIS_STORAGE_PREFIX,
  ORDER_STORAGE_PREFIX,
} from '@/components/DataTable/ColumnVisibility';
export {
  EXPORT_CONCURRENCY,
  EXPORT_PAGE_SIZE,
  EXPORT_ROW_CAP,
  fetchAllPaged,
} from '../exportPaging';
export type { PageResult } from '../exportPaging';

// 设备列表导出工具。
// 设计要点(用户决策 2026-06-02)：
//   1. 导出内容 = 当前筛选条件命中的全部数据(跨分页),由调用方拉全量后传入;
//   2. 导出列与列顺序 = 用户"列设置"(ColumnVisibility)中勾选 + 拖拽后的结果,
//      与表格当前展示完全一致 —— 通过读取同一份 localStorage 状态实现。
// 这里只负责"列解析 + 文件生成",不关心数据来源与单元格取值。

/** 不可导出的"纯动作"列(无业务文本值)。 */
const NON_EXPORTABLE_KEYS = new Set<string>(['actions', 'latestLog']);

/** 列定义的最小可导出形态(从 DataTableColumn 收窄)。 */
export interface ExportColumnInput {
  key: string;
  title?: unknown;
  dataIndex?: string;
  hidden?: boolean;
}

/** 解析后用于导出的列(标题已确保为字符串)。 */
export interface ResolvedExportColumn {
  key: string;
  title: string;
  dataIndex?: string;
}

function readJsonArray(storageKey: string): string[] | null {
  try {
    const raw = localStorage.getItem(storageKey);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as unknown;
    return Array.isArray(parsed) ? (parsed as string[]) : null;
  } catch {
    return null;
  }
}

/**
 * 依据"列设置"(localStorage)解析出当前应导出的列：
 *  - 顺序：用户拖拽后的顺序(omc_col_order_<tableId>)，缺省按列定义顺序；
 *  - 可见性：未隐藏的列(omc_col_vis_<tableId> 记录隐藏 key)，缺省取列定义 hidden；
 *  - 排除纯动作列(actions / latestLog)。
 * 与 ColumnVisibility 组件读取的是同一份状态，确保"导出 == 列表展示"。
 */
export function resolveVisibleExportColumns(
  columns: ReadonlyArray<ExportColumnInput>,
  tableId: string,
): ResolvedExportColumn[] {
  const byKey = new Map(columns.map((c) => [c.key, c]));

  const storedHidden = readJsonArray(`${VIS_STORAGE_PREFIX}${tableId}`);
  const hiddenSet = new Set<string>(
    storedHidden ?? columns.filter((c) => c.hidden).map((c) => c.key),
  );

  const storedOrder = readJsonArray(`${ORDER_STORAGE_PREFIX}${tableId}`);
  const defOrder = columns.map((c) => c.key);
  // 以存储顺序为准，补齐存储里缺失的新列(追加到末尾)，与 ColumnVisibility 同逻辑。
  const order =
    storedOrder && storedOrder.length > 0
      ? [...storedOrder, ...defOrder.filter((k) => !storedOrder.includes(k))]
      : defOrder;

  const resolved: ResolvedExportColumn[] = [];
  for (const key of order) {
    if (NON_EXPORTABLE_KEYS.has(key)) continue;
    if (hiddenSet.has(key)) continue;
    const col = byKey.get(key);
    if (!col) continue;
    const title = typeof col.title === 'string' && col.title ? col.title : col.key;
    resolved.push({ key: col.key, title, dataIndex: col.dataIndex });
  }
  return resolved;
}

// ─── 文件生成 ────────────────────────────────────────────────────────────

function escapeCsvCell(value: string): string {
  return `"${value.replace(/"/g, '""')}"`;
}

/** headers + 二维行数据 → CSV 文本(CRLF 行尾，Excel 友好)。 */
export function buildCsvContent(headers: string[], rows: string[][]): string {
  const lines = [headers, ...rows].map((cells) => cells.map(escapeCsvCell).join(','));
  return lines.join('\r\n');
}

/** 触发 CSV 下载(带 BOM 以保证 Excel 正确识别 UTF-8 中文)。 */
export function triggerCsvDownload(content: string, filename: string): void {
  const blob = new Blob(['﻿' + content], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}

// 中文按 2 字宽估算列宽，与 AbnormalReboot 导出口径一致。
function widthOfStr(s: string): number {
  let w = 0;
  for (const ch of s) w += /[一-鿿＀-￯]/.test(ch) ? 2 : 1;
  return w;
}

/** headers + 二维行数据 → XLSX 文件并触发下载。 */
export function triggerXlsxDownload(
  headers: string[],
  rows: string[][],
  filename: string,
  sheetName: string,
): void {
  const ws = XLSX.utils.aoa_to_sheet([headers, ...rows]);
  ws['!cols'] = headers.map((h, colIdx) => {
    let max = widthOfStr(h);
    for (const row of rows) {
      const cell = row[colIdx] ?? '';
      const w = widthOfStr(cell);
      if (w > max) max = w;
    }
    return { wch: Math.min(Math.max(max + 2, 10), 50) };
  });
  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, sheetName);
  XLSX.writeFile(wb, filename);
}

/** 导出文件名时间戳：YYYYMMDD-HHmmss。 */
export function exportTimestamp(): string {
  const d = new Date();
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}${p(d.getMonth() + 1)}${p(d.getDate())}-${p(d.getHours())}${p(d.getMinutes())}${p(d.getSeconds())}`;
}
