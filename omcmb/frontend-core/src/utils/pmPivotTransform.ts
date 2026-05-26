/**
 * T-0174 PM long → wide 透视表转换。
 *
 * 输入：AggregatedRow[]（长表，每行 = 一个 时间×设备×指标×LDN 桶）。
 * 输出：列定义（动态指标） + 行数据（时间维度）。
 *
 * 列 key 设计：根据是否多设备 / 多 LDN 决定后缀
 *   - 单设备单 LDN：`metricPath` 直接作 key
 *   - 多设备：`metricPath||deviceSn` 加 [SN] 后缀
 *   - 多 LDN：进一步加 [LDN] 后缀
 */

import type { AggregatedRow } from '../types/pmDashboard';

export interface PivotColumn {
  key: string;
  title: string;
  tooltipFull: string;
  metricPath: string;
  deviceSn?: string;
  objectLdn?: string;
}

export interface PivotRow {
  time: string;
  cells: Record<string, number | null>;
}

export interface PivotResult {
  columns: PivotColumn[];
  rows: PivotRow[];
}

function cellKey(metricPath: string, deviceSn: string | undefined, objectLdn: string | null | undefined, multiDevice: boolean, multiLdn: boolean): string {
  const parts = [metricPath];
  if (multiDevice && deviceSn) parts.push(deviceSn);
  if (multiLdn && objectLdn) parts.push(objectLdn);
  return parts.join('||');
}

function cellTitle(metricPath: string, deviceSn: string | undefined, objectLdn: string | null | undefined, multiDevice: boolean, multiLdn: boolean): string {
  if (!multiDevice && !multiLdn) return metricPath;
  const suffix: string[] = [];
  if (multiDevice && deviceSn) suffix.push(deviceSn);
  if (multiLdn && objectLdn) suffix.push(objectLdn);
  return `${metricPath} [${suffix.join(' / ')}]`;
}

/**
 * 把 long format AggregatedRow[] 转成 wide format（行=时间，列=指标）。
 *
 * 算法：
 *   1. 遍历所有 row，提取 (metricPath, deviceSn, objectLdn) 唯一组合 → 列集合
 *   2. 提取所有时间桶 → 行集合（按时间升序）
 *   3. 对每个 (row, col) 填入 metricValue；缺采 = null
 */
export function pivotLongToWide(rows: AggregatedRow[]): PivotResult {
  if (!rows || rows.length === 0) {
    return { columns: [], rows: [] };
  }

  // 决定是否多设备 / 多 LDN
  const deviceSet = new Set<string>();
  const ldnSet = new Set<string>();
  rows.forEach((r) => {
    if (r.deviceSn) deviceSet.add(r.deviceSn);
    if (r.objectLdn) ldnSet.add(r.objectLdn);
  });
  const multiDevice = deviceSet.size > 1;
  const multiLdn = ldnSet.size > 1;

  // 收集列与时间
  const columnMap = new Map<string, PivotColumn>();
  const timeSet = new Set<string>();
  rows.forEach((r) => {
    const key = cellKey(r.metricPath, r.deviceSn, r.objectLdn, multiDevice, multiLdn);
    if (!columnMap.has(key)) {
      columnMap.set(key, {
        key,
        title: cellTitle(r.metricPath, r.deviceSn, r.objectLdn, multiDevice, multiLdn),
        tooltipFull: [r.metricPath, r.deviceSn, r.objectLdn].filter(Boolean).join(' / '),
        metricPath: r.metricPath,
        deviceSn: r.deviceSn,
        objectLdn: r.objectLdn ?? undefined,
      });
    }
    timeSet.add(r.time);
  });

  // 时间桶按升序
  const times = Array.from(timeSet).sort();
  const columns = Array.from(columnMap.values()).sort((a, b) => a.title.localeCompare(b.title));

  // 时间 → cells 映射
  const rowMap = new Map<string, Record<string, number | null>>();
  times.forEach((t) => rowMap.set(t, {}));
  rows.forEach((r) => {
    const key = cellKey(r.metricPath, r.deviceSn, r.objectLdn, multiDevice, multiLdn);
    const cells = rowMap.get(r.time)!;
    cells[key] = r.metricValue;
  });

  // 填充缺采 cell = null（保证表格列对齐）
  const outRows: PivotRow[] = times.map((t) => {
    const cells: Record<string, number | null> = {};
    const got = rowMap.get(t) ?? {};
    columns.forEach((c) => {
      cells[c.key] = c.key in got ? got[c.key] : null;
    });
    return { time: t, cells };
  });

  return { columns, rows: outRows };
}
