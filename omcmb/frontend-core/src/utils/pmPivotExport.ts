/**
 * PM 透视数据导出：AggregatedRow[] → Excel 行对象数组。
 *
 * 列集合 / 列名 / 时间&数值格式与「指标查询页」透视表 (KPIQuery/PivotTable) 完全一致：
 *   固定维度列：开始时间 / 结束时间 / 设备 SN / 测量对象（原始 object_ldn）
 *   动态指标列：每个 metricPath 一列（列名 = metricPath）
 * 仪表盘普通 Panel 导出复用此函数，保证导出表与指标查询页同款。
 */

import dayjs from 'dayjs';
import type { AggregatedRow } from '../types/pmDashboard';
import { pivotLongToWide, formatPivotNumber } from './pmPivotTransform';

// 固定维度列标题（与 PivotTable 固定列保持一致）
export const PIVOT_FIXED_HEADERS = {
  startTime: '开始时间',
  endTime: '结束时间',
  deviceSn: '设备 SN',
  measObject: '测量对象',
} as const;

const TIME_FMT = 'YYYY-MM-DD HH:mm';

/**
 * 把 long format 聚合行透视成导出行（行=时间×设备×LDN，列=固定维度 + N 指标）。
 */
export function pivotRowsToSheet(rows: AggregatedRow[]): Array<Record<string, string | number | null>> {
  const { columns, rows: pivotRows } = pivotLongToWide(rows);
  return pivotRows.map((r) => {
    const out: Record<string, string | number | null> = {
      [PIVOT_FIXED_HEADERS.startTime]: dayjs(r.startTime ?? r.time).format(TIME_FMT),
      [PIVOT_FIXED_HEADERS.endTime]: r.endTime ? dayjs(r.endTime).format(TIME_FMT) : '-',
      [PIVOT_FIXED_HEADERS.deviceSn]: r.deviceSn ?? '-',
      [PIVOT_FIXED_HEADERS.measObject]: r.objectLdn ?? '-',
    };
    columns.forEach((c) => {
      out[c.title] = formatPivotNumber(r.cells[c.key]);
    });
    return out;
  });
}
