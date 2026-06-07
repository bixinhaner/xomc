// MML 控制台 V2 —— 结果导出工具（mock 阶段：纯前端 Blob 下载）。
//
// 设计 §3.6 的导出端点（POST /mml/tasks/:id/export）尚未就绪，当前直接由前端把
// 结果表格序列化为 CSV / JSON 下载。XLSX 暂以 CSV 兜底（待后端接入再走真实 xlsx 流）。

import type { ResultColumn, ResultRow, ExportFormat } from './types';
import { STATUS_META } from './constants';

function triggerDownload(content: string, filename: string, mime: string): void {
  const blob = new Blob([content], { type: mime });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

/**
 * 从 URL（如 MinIO 预签名）拉取内容并以 Blob 方式触发浏览器下载。
 * 直接 window.open 预签名 URL 时，因对象无 `Content-Disposition: attachment` 头，浏览器会
 * 打开空白页/内联预览而非下载；这里 fetch→blob→`<a download>` 强制以指定文件名下载。
 */
export async function downloadFromUrl(url: string, filename: string): Promise<void> {
  const resp = await fetch(url);
  if (!resp.ok) throw new Error(`下载失败：HTTP ${resp.status}`);
  const blob = await resp.blob();
  const objUrl = URL.createObjectURL(blob);
  try {
    const a = document.createElement('a');
    a.href = objUrl;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  } finally {
    URL.revokeObjectURL(objUrl);
  }
}

function csvCell(v: string): string {
  if (/[",\n]/.test(v)) return `"${v.replace(/"/g, '""')}"`;
  return v;
}

function toCsv(columns: ResultColumn[], rows: ResultRow[]): string {
  const header = ['序号', '设备SN', '状态', ...columns.map((c) => c.label), '故障码'];
  const lines = rows.map((r, i) =>
    [
      String(i + 1),
      r.deviceSn,
      STATUS_META[r.status].text,
      ...columns.map((c) => r.cells[c.path] ?? ''),
      r.faultCode ?? '',
    ]
      .map(csvCell)
      .join(','),
  );
  return [header.map(csvCell).join(','), ...lines].join('\n');
}

function toJson(columns: ResultColumn[], rows: ResultRow[]): string {
  const data = rows.map((r) => ({
    deviceSn: r.deviceSn,
    status: r.status,
    elapsedMs: r.elapsedMs,
    faultCode: r.faultCode ?? null,
    values: columns.reduce<Record<string, string>>((acc, c) => {
      acc[c.label] = r.cells[c.path] ?? '';
      return acc;
    }, {}),
  }));
  return JSON.stringify(data, null, 2);
}

/** 导出全部设备结果。 */
export function exportAll(
  format: ExportFormat,
  columns: ResultColumn[],
  rows: ResultRow[],
  commandCode: string,
): void {
  const base = `mml-result-${commandCode.replace(/\s+/g, '_')}`;
  if (format === 'json') {
    triggerDownload(toJson(columns, rows), `${base}.json`, 'application/json');
  } else {
    // csv 与 xlsx（mock 兜底）均输出 CSV，xlsx 用 BOM 让 Excel 正确识别中文
    const csv = (format === 'xlsx' ? '﻿' : '') + toCsv(columns, rows);
    triggerDownload(csv, `${base}.${format === 'xlsx' ? 'csv' : 'csv'}`, 'text/csv;charset=utf-8');
  }
}

/** 导出单设备结果（CSV）。 */
export function exportOne(columns: ResultColumn[], row: ResultRow): void {
  const csv = '﻿' + toCsv(columns, [row]);
  triggerDownload(csv, `mml-result-${row.deviceSn}.csv`, 'text/csv;charset=utf-8');
}
