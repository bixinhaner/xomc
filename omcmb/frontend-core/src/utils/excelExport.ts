/**
 * 通用 Excel 导出工具（xlsx 0.18.x）。
 *
 * 用途：G6-Gap-10 panel 数据导出 + G7-Gap-4 adhoc 结果导出。
 *
 * PDF 导出策略：依赖浏览器原生 window.print()（带 @media print CSS）；
 * 此处不引入 pdfmake/jspdf 避免重 dep。printElement 帮助把指定 DOM 节点
 * 临时放进 print iframe 触发打印对话框。
 */

import * as XLSX from 'xlsx';

export interface SheetInput {
  name: string;
  // 行对象数组（key = 列标题，value = 单元格内容）；null / undefined 自动空
  rows: Array<Record<string, string | number | null | undefined>>;
}

/**
 * 导出多 sheet 的 .xlsx 文件。
 *
 * @param filename 用户下载文件名（不带扩展名；自动加 .xlsx）
 * @param sheets 多个 sheet 输入
 */
export function exportWorkbook(filename: string, sheets: SheetInput[]) {
  const wb = XLSX.utils.book_new();
  sheets.forEach((s) => {
    if (s.rows.length === 0) {
      // 空 sheet 仍创建（用户期望看到对应 sheet 即使是空的）
      const ws = XLSX.utils.aoa_to_sheet([['(no data)']]);
      XLSX.utils.book_append_sheet(wb, ws, sanitizeSheetName(s.name));
      return;
    }
    const ws = XLSX.utils.json_to_sheet(s.rows);
    XLSX.utils.book_append_sheet(wb, ws, sanitizeSheetName(s.name));
  });
  XLSX.writeFile(wb, `${filename}.xlsx`);
}

/**
 * Excel sheet 名禁止 \\ / ? * [ ] : 且长度 <= 31。
 */
function sanitizeSheetName(name: string): string {
  return name.replace(/[\\/?*[\]:]/g, '_').slice(0, 31) || 'Sheet';
}

/**
 * 调浏览器原生打印对话框，用户可保存为 PDF。
 *
 * 该函数把整个文档触发 print；调用前通常需要先短暂展开折叠区域 / 切换为打印友好布局。
 */
export function printAsPDF(documentTitle?: string) {
  const prevTitle = document.title;
  if (documentTitle) document.title = documentTitle;
  window.print();
  if (documentTitle) {
    // 异步恢复，避免打印对话框关闭时已经过去
    setTimeout(() => {
      document.title = prevTitle;
    }, 500);
  }
}
