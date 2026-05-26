/**
 * T-0174 PM long → wide 透视表转换。
 *
 * 输入：AggregatedRow[]（长表，每行 = 一个 时间×设备×LDN×指标 桶）。
 * 输出：行 = (time + device_sn + cellid + plmn) 四元组（PM 数据天然唯一标识），列 = N 个指标。
 *
 * object_ldn 字段格式约定（来自 pm_metrics 表）：
 *   - "Cellid=111172245"
 *   - "Cellid=111172245,PLMN=46068"
 *   - 空字符串 / null（站级指标）
 *
 * 后端 uq_pm_metrics_natural 唯一索引保证：(device_oui, device_sn, metric_path, granularity, end_time, time, object_ldn) 唯一。
 * 所以行键 (time + device_sn + object_ldn) 不冲突。
 */

import type { AggregatedRow } from '../types/pmDashboard';

export interface PivotColumn {
  key: string;        // = metricPath
  title: string;      // = metricPath
  metricPath: string;
}

export interface PivotRow {
  key: string;             // (time + sn + ldn) 唯一行标识
  time: string;
  deviceSn?: string;
  cellId?: string;         // 从 object_ldn 解析
  plmn?: string;           // 从 object_ldn 解析
  objectLdn?: string;      // 原始 LDN（备用 tooltip）
  cells: Record<string, number | null>;  // metricPath → value
}

export interface PivotResult {
  columns: PivotColumn[];
  rows: PivotRow[];
  // 多设备 / 多 LDN 标记，组件用来决定是否折叠固定列宽
  hasMultiDevice: boolean;
  hasMultiLdn: boolean;
  hasPlmn: boolean;
}

/**
 * 解析 object_ldn 字符串 → { cellId, plmn }。
 *
 * 支持格式：
 *   "Cellid=111172245"                  → { cellId: "111172245" }
 *   "Cellid=111172245,PLMN=46068"       → { cellId: "111172245", plmn: "46068" }
 *   "PLMN=46068,Cellid=111172245"       → { cellId: "111172245", plmn: "46068" } (顺序无关)
 *   ""  / null / undefined              → {}
 *
 * 大小写不敏感（cellid / Cellid / CELLID / plmn / PLMN）。
 */
export function parseObjectLdn(ldn: string | null | undefined): { cellId?: string; plmn?: string } {
  if (!ldn) return {};
  const out: { cellId?: string; plmn?: string } = {};
  ldn.split(',').forEach((kv) => {
    const idx = kv.indexOf('=');
    if (idx < 0) return;
    const key = kv.slice(0, idx).trim().toLowerCase();
    const value = kv.slice(idx + 1).trim();
    if (!value) return;
    if (key === 'cellid' || key === 'cell_id' || key === 'cell') {
      out.cellId = value;
    } else if (key === 'plmn' || key === 'plmnid' || key === 'plmn_id') {
      out.plmn = value;
    }
  });
  return out;
}

/**
 * 把 long format AggregatedRow[] 转成 wide format（行=时间×设备×LDN，列=N 指标）。
 */
export function pivotLongToWide(rows: AggregatedRow[]): PivotResult {
  if (!rows || rows.length === 0) {
    return { columns: [], rows: [], hasMultiDevice: false, hasMultiLdn: false, hasPlmn: false };
  }

  const deviceSet = new Set<string>();
  const ldnSet = new Set<string>();
  let hasPlmn = false;
  rows.forEach((r) => {
    if (r.deviceSn) deviceSet.add(r.deviceSn);
    if (r.objectLdn) ldnSet.add(r.objectLdn);
    const { plmn } = parseObjectLdn(r.objectLdn);
    if (plmn) hasPlmn = true;
  });

  // 收集指标列（按 metricPath 唯一）
  const columnMap = new Map<string, PivotColumn>();
  rows.forEach((r) => {
    if (!columnMap.has(r.metricPath)) {
      columnMap.set(r.metricPath, {
        key: r.metricPath,
        title: r.metricPath,
        metricPath: r.metricPath,
      });
    }
  });
  const columns = Array.from(columnMap.values()).sort((a, b) => a.metricPath.localeCompare(b.metricPath));

  // 收集行（按 time + sn + ldn 唯一）
  const rowMap = new Map<string, PivotRow>();
  rows.forEach((r) => {
    const sn = r.deviceSn ?? '';
    const ldn = r.objectLdn ?? '';
    const key = `${r.time}||${sn}||${ldn}`;
    let row = rowMap.get(key);
    if (!row) {
      const parsed = parseObjectLdn(ldn);
      row = {
        key,
        time: r.time,
        deviceSn: r.deviceSn,
        cellId: parsed.cellId,
        plmn: parsed.plmn,
        objectLdn: ldn || undefined,
        cells: {},
      };
      rowMap.set(key, row);
    }
    row.cells[r.metricPath] = r.metricValue;
  });

  // 填补缺采单元格 = null
  const outRows = Array.from(rowMap.values()).map((row) => {
    columns.forEach((c) => {
      if (!(c.key in row.cells)) row.cells[c.key] = null;
    });
    return row;
  });

  // 排序：时间升序 → SN → LDN
  outRows.sort((a, b) => {
    if (a.time !== b.time) return a.time.localeCompare(b.time);
    const sa = a.deviceSn ?? '';
    const sb = b.deviceSn ?? '';
    if (sa !== sb) return sa.localeCompare(sb);
    return (a.objectLdn ?? '').localeCompare(b.objectLdn ?? '');
  });

  return {
    columns,
    rows: outRows,
    hasMultiDevice: deviceSet.size > 1,
    hasMultiLdn: ldnSet.size > 1,
    hasPlmn,
  };
}
