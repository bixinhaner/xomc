import type { Device, DeviceGroup } from '@core/types/device';
import type { Locale } from '@core/utils/i18nText';
import { withDeviceGroupDisplayName } from '@core/utils/deviceGroupDisplay';
import { EXPORT_COLUMNS } from './deviceCsvSchema';

export type DeviceCsvT = (id: string, values?: Record<string, string | number>) => string;

export function normalizeDevicesForExport(
  devices: Device[],
  groups: DeviceGroup[],
  locale: Locale,
): Device[] {
  return withDeviceGroupDisplayName(devices, groups, locale);
}

/** 把一个 CSV 字段值转字符串并按需要加引号转义（含逗号 / 换行 / 引号）。 */
export function csvField(value: unknown): string {
  if (value === null || value === undefined) return '';
  const s = typeof value === 'string' ? value : String(value);
  // 包含逗号 / 双引号 / 换行符 → 用双引号包起来，并把双引号 → 双双引号。
  if (s.includes(',') || s.includes('"') || s.includes('\n') || s.includes('\r')) {
    return `"${s.replace(/"/g, '""')}"`;
  }
  return s;
}

/**
 * 按 deviceCsvSchema.EXPORT_COLUMNS 顺序拼 CSV。
 *
 * 列覆盖设备分组页表格导出字段：
 *   SN / 设备名称 / 连接状态 / MAC地址 / 设备分组 / 归属来源 / 备注
 *
 * 表头、状态、枚举字段均走当前 locale 的 i18n 文案。
 */
export function buildCsvForDeviceList(
  devices: Device[],
  t: DeviceCsvT,
): string {
  const headers = EXPORT_COLUMNS.map((c) => csvField(t(c.i18nKey))).join(',');
  const rows = devices.map((d) =>
    EXPORT_COLUMNS.map((c) => csvField(c.getValue(d, { t }))).join(','),
  );
  // UTF-8 BOM 前缀让 Excel 正确识别编码。
  return '\uFEFF' + headers + '\n' + rows.join('\n') + '\n';
}
