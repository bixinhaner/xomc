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
import { formatPmMetricDisplayValue, normalizePmMetricValue } from './pmMetricValue';

// ─── object_ldn 解析（对齐后端 metrics/object_ldn.go） ───

export type ObjectLdnTech = '' | 'lte' | 'nr' | 'gsm';

/**
 * object_ldn 串拆出的全部维度键，与后端 ObjectLDNFields 一一对应。
 * 按制式补缺段留 undefined（不赋空串），用 baseCellId / basePlmn 取跨制式统一值。
 */
export interface ObjectLdnFields {
  tech: ObjectLdnTech;
  // 4G / LTE
  cellId?: string;
  plmn?: string;
  // 5G / NR
  gnbId?: string;
  nrCgi?: string;
  cuid?: string;
  duid?: string;
  plmnId?: string;
  nssai?: string;
  sliceGroup?: string;
  // GSM
  uid?: string;
}

// ─── Pivot 表类型 ───

export interface PivotColumn {
  key: string;        // = metricPath（稳定标识：KPI 为 K 编号，counter 为点分名）；用于单元格取值 + 列宽持久化
  title: string;      // = displayName 友好名（KPI 友好名 / PLMN 级带标记），回退 metricPath
}

export interface PivotRow {
  key: string;             // (time + sn + ldn) 唯一行标识
  time: string;
  startTime?: string;      // 桶开始时间（RFC3339）
  endTime?: string;        // 桶结束时间（RFC3339）
  deviceSn?: string;
  objectLdn?: string;      // 原始 object_ldn（直接展示，不格式化）
  cells: Record<string, number | null>;  // metricPath → value
}

export interface PivotResult {
  columns: PivotColumn[];
  rows: PivotRow[];
}

/**
 * 解析 object_ldn 字符串 → ObjectLdnFields（对齐后端 ParseObjectLDN）。
 *
 * 支持三种制式（大小写不敏感，逗号分隔 Key=Value）：
 *   4G / LTE: "Cellid=N" | "Cellid=N,PLMN=M"
 *   5G / NR:  "Type=Cell,Mode=SA,gNBID=N,NrCGI=N,CUID=N" | ...PLMNID=M | ...DUID=N | ...NSSAI=... | ...SCLICEGROUP=...
 *   GSM:      "Uid=N-N"
 *   空/null/undefined → { tech: '' }
 */
export function parseObjectLdn(ldn: string | null | undefined): ObjectLdnFields {
  if (!ldn) return { tech: '' };
  const kv: Record<string, string> = {};
  ldn.split(',').forEach((part) => {
    const idx = part.indexOf('=');
    if (idx < 0) return;
    const key = part.slice(0, idx).trim().toLowerCase();
    const value = part.slice(idx + 1).trim();
    if (value) kv[key] = value;
  });

  const out: ObjectLdnFields = { tech: '' };
  // 4G
  if (kv['cellid']) out.cellId = kv['cellid'];
  if (kv['plmn']) out.plmn = kv['plmn'];
  // 5G
  if (kv['gnbid']) out.gnbId = kv['gnbid'];
  if (kv['nrcgi']) out.nrCgi = kv['nrcgi'];
  if (kv['cuid']) out.cuid = kv['cuid'];
  if (kv['duid']) out.duid = kv['duid'];
  if (kv['plmnid']) out.plmnId = kv['plmnid'];
  if (kv['nssai']) out.nssai = kv['nssai'];
  if (kv['sclicegroup']) out.sliceGroup = kv['sclicegroup'];
  // GSM
  if (kv['uid']) out.uid = kv['uid'];

  // 制式判定（优先级与后端一致：gNBID/NrCGI → NR；Uid → GSM；Cellid → LTE）
  if (out.gnbId || out.nrCgi) {
    out.tech = 'nr';
    delete out.cellId;
    delete out.plmn;
  } else if (out.uid) {
    out.tech = 'gsm';
    delete out.cellId;
    delete out.plmn;
  } else if (out.cellId) {
    out.tech = 'lte';
  }

  return out;
}

/**
 * 按制式返回基础小区标识（对齐后端 BaseCellID()）。
 *   4G → cellId, 5G → nrCgi, GSM → uid
 */
export function baseCellId(fields: ObjectLdnFields): string | undefined {
  switch (fields.tech) {
    case 'lte': return fields.cellId;
    case 'nr':  return fields.nrCgi;
    case 'gsm': return fields.uid;
    default:    return undefined;
  }
}

/**
 * 按制式返回 PLMN（对齐后端字段名差异：4G PLMN / 5G PLMNID）。
 */
export function basePlmn(fields: ObjectLdnFields): string | undefined {
  switch (fields.tech) {
    case 'lte': return fields.plmn;
    case 'nr':  return fields.plmnId;
    default:    return undefined;
  }
}

/**
 * 输出人类可读测量对象摘要。
 *   5G: "Cell(NrCGI=801)" / "Cell(NrCGI=801) PLMN=46001" / "DU(DUID=1)"
 *   GSM: "GSM(Uid=1-2)"
 *   4G: "Cell(111172245)" / "Cell(111172245) PLMN=46068"
 *   未知: 原串
 */
export function formatObjectLdn(ldn: string | null | undefined): string {
  if (!ldn) return '';
  const f = parseObjectLdn(ldn);
  switch (f.tech) {
    case 'nr': {
      const typeMatch = ldn.match(/\bType=([^,]+)/i);
      const type = typeMatch?.[1] ?? 'NR';
      let label = type;
      if (f.nrCgi) label += `(NrCGI=${f.nrCgi})`;
      else if (f.duid) label += `(DUID=${f.duid})`;
      else if (f.gnbId) label += `(gNBID=${f.gnbId})`;
      if (f.plmnId) label += ` PLMN=${f.plmnId}`;
      return label;
    }
    case 'gsm':
      return `GSM(Uid=${f.uid})`;
    case 'lte': {
      let label = f.cellId ? `Cell(${f.cellId})` : 'Cell';
      if (f.plmn) label += ` PLMN=${f.plmn}`;
      return label;
    }
    default:
      return ldn;
  }
}

/**
 * 透视表数值显示格式：有效数字固定保留两位，缺采显示 "-"。
 * 指标查询页表格与仪表盘普通 Panel 导出共用，保证两处数值格式一致。
 */
export function formatPivotNumber(v: number | null | undefined): string {
  return formatPmMetricDisplayValue(v);
}

/**
 * 把 long format AggregatedRow[] 转成 wide format（行=时间×设备×LDN，列=N 指标）。
 */
export function pivotLongToWide(rows: AggregatedRow[]): PivotResult {
  if (!rows || rows.length === 0) {
    return { columns: [], rows: [] };
  }

  // 收集指标列（按 metricPath 唯一）
  const columnMap = new Map<string, PivotColumn>();
  rows.forEach((r) => {
    if (!columnMap.has(r.metricPath)) {
      columnMap.set(r.metricPath, {
        key: r.metricPath,
        title: r.displayName || r.metricPath,
      });
    } else if (r.displayName) {
      // 占位行可能无 displayName 先建了列；真实行带 displayName 时补上友好名
      const col = columnMap.get(r.metricPath)!;
      if (col.title === col.key) col.title = r.displayName;
    }
  });
  const columns = Array.from(columnMap.values()).sort((a, b) => a.key.localeCompare(b.key));

  // 收集行（按 time + sn + ldn 唯一）
  const rowMap = new Map<string, PivotRow>();
  rows.forEach((r) => {
    const sn = r.deviceSn ?? '';
    const ldn = r.objectLdn ?? '';
    const key = `${r.time}||${sn}||${ldn}`;
    let row = rowMap.get(key);
    if (!row) {
      row = {
        key,
        time: r.time,
        startTime: r.startTime,
        endTime: r.endTime,
        deviceSn: r.deviceSn,
        objectLdn: ldn || undefined,
        cells: {},
      };
      rowMap.set(key, row);
    }
    row.cells[r.metricPath] = normalizePmMetricValue(r.metricValue);
  });

  // 填补缺采单元格 = null
  const outRows = Array.from(rowMap.values()).map((row) => {
    columns.forEach((c) => {
      if (!(c.key in row.cells)) row.cells[c.key] = null;
    });
    return row;
  });

  // 排序：时间倒序（最新在前） → SN 升序 → LDN 升序
  outRows.sort((a, b) => {
    if (a.time !== b.time) return b.time.localeCompare(a.time);
    const sa = a.deviceSn ?? '';
    const sb = b.deviceSn ?? '';
    if (sa !== sb) return sa.localeCompare(sb);
    return (a.objectLdn ?? '').localeCompare(b.objectLdn ?? '');
  });

  return {
    columns,
    rows: outRows,
  };
}
