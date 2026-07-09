// 设备 CSV schema：导入模板 / 列表导出 / 导入解析 三处共用一份列定义。
//
// 设计目标：
//   - 表头统一用中文（SN 列与「设备列表/设备分组」页一致用 'SN'），便于运营在 Excel 看；
//   - 导入字段（snake_case）由 zh → snake 别名表 [[HEADER_ALIAS]] 给出，parseCsv 把中文
//     表头和老的 snake_case 表头都归一到 snake_case，向后兼容；
//   - 导入模板列 = 仅 SN（必填）+ 设备名称 + 备注（产品要求：导入只需这三项）；
//   - 导出列 = /device/group 页面实际展示的列（SN / 设备名称 / 连接状态 / MAC地址 /
//     设备分组 / 归属来源 / 备注），顺序与页面一致。

import type { Device } from '@core/types/device';

export interface ExportCtx {
  t: (id: string, values?: Record<string, string | number>) => string;
}

// ─── 1) 导入字段元数据：模板使用 + 解析器使用 ─────────────────────────────────

export interface ImportColumn {
  /** 中文表头（解析器 HEADER_ALIAS 用） */
  zh: string;
  /** i18n key，模板/导出表头随当前语言渲染 */
  i18nKey: string;
  /** snake_case 字段名（与后端 CreateDeviceRequest 对齐） */
  snake: string;
  /** 是否必填 */
  required: boolean;
  /** 模板第 i 行示例（i=0/1/2），默认中文 */
  example: readonly string[];
  /** 可选 i18n key 列表，若提供则优先用 t(key) 渲染示例（英文等多语言场景）；为空则降级用 example */
  exampleI18nKeys?: readonly string[];
}

// 产品要求：导入只保留 SN（必填）+ 设备名称 + 备注三列。
// 表头 SN 与「设备列表/设备分组」页一致。serial_number 仍是后端字段名。
export const IMPORT_COLUMNS: readonly ImportColumn[] = [
  { zh: 'SN', i18nKey: 'device.csv.column.sn', snake: 'serial_number', required: true,
    example: ['120288069823C4B0060', '1202000690241FB0010', '120200087125BJB0002'] },
  { zh: '设备名称', i18nKey: 'device.csv.column.deviceName', snake: 'device_name', required: false,
    example: ['北京海淀中关村站', '北京朝阳CBD站', '上海浦东陆家嘴站'],
    exampleI18nKeys: [
      'device.csv.example.stationA',
      'device.csv.example.stationB',
      'device.csv.example.stationC',
    ] },
  { zh: '备注', i18nKey: 'device.csv.column.remark', snake: 'remark', required: false,
    example: ['一期', '二期', ''],
    exampleI18nKeys: [
      'device.csv.example.remarkA',
      'device.csv.example.remarkB',
      '',
    ] },
] as const;

/** 导入端：把"中文表头"或"老 snake_case 表头"统一映射到 snake_case。 */
export const HEADER_ALIAS: Readonly<Record<string, string>> = (() => {
  const m: Record<string, string> = {};
  for (const c of IMPORT_COLUMNS) {
    m[c.zh] = c.snake;
    m[c.snake] = c.snake; // self-alias，向后兼容旧 snake_case 模板
  }
  return m;
})();

export const REQUIRED_IMPORT_SNAKE: readonly string[] = IMPORT_COLUMNS
  .filter((c) => c.required)
  .map((c) => c.snake);

export const KNOWN_IMPORT_SNAKE: ReadonlySet<string> = new Set(IMPORT_COLUMNS.map((c) => c.snake));

// ─── 2) 导出列：设备列表展示字段 + OUI/运营商/制式（便于 roundtrip） ─────────

export interface ExportColumn {
  /** 中文表头（解析器向后兼容用） */
  zh: string;
  /** i18n key，导出表头随当前语言渲染 */
  i18nKey: string;
  /** 取值器 */
  getValue: (d: Device, ctx: ExportCtx) => string | number;
}

// 导出列 = /device/group 页面实际展示列，顺序与页面一致：
//   SN / 设备名称 / 连接状态 / MAC地址 / 设备分组 / 归属来源 / 备注
export const EXPORT_COLUMNS: readonly ExportColumn[] = [
  { zh: 'SN', i18nKey: 'device.csv.column.sn',
    getValue: (d) => d.sn ?? '' },
  { zh: '设备名称', i18nKey: 'device.csv.column.deviceName',
    getValue: (d) => d.name ?? '' },
  { zh: '连接状态', i18nKey: 'device.csv.column.connStatus',
    getValue: (d, { t }) => d.connStatus === 'online' ? t('status.online') : t('status.offline') },
  { zh: 'MAC地址', i18nKey: 'device.csv.column.macAddress',
    getValue: (d) => d.macAddress ?? '' },
  { zh: '设备分组', i18nKey: 'device.csv.column.deviceGrouping',
    getValue: (d) => d.groupName ?? '' },
  { zh: '归属来源', i18nKey: 'device.csv.column.sourceType',
    getValue: (d, { t }) => {
      const src = d.sourceType;
      return src ? t(`device.sourceType.${src}`) : '';
    } },
  { zh: '备注', i18nKey: 'device.csv.column.remark',
    getValue: (d) => d.remark ?? '' },
] as const;
