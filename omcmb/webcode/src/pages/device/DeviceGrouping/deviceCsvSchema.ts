// 设备 CSV schema：导入模板 / 列表导出 / 导入解析 三处共用一份列定义。
//
// 设计目标（用户反馈：批量导入和导出模板格式不一样）：
//   - 表头统一用中文，便于运营人员在 Excel 看；
//   - 导入需要的字段（snake_case）由 zh → snake 别名表 [[HEADER_ALIAS]] 给出，
//     parseCsv 把中文表头和老的 snake_case 表头都归一到 snake_case，向后兼容；
//   - 导入模板列 = 导入必填 + 可选；导出列 = 设备列表展示列 + OUI/运营商/制式
//     （导出附带这三列让用户改完可直接 re-import 一遍完成 roundtrip）。

import type { Device, EngStatus } from '@core/types/device';

export interface ExportCtx {
  t: (id: string, values?: Record<string, string | number>) => string;
}

const ENG_STATUS_LABEL_KEYS: Record<EngStatus, string> = {
  commissioned: 'device.engStatus.commissioned',
  uncommissioned: 'device.engStatus.uncommissioned',
  decommissioned: 'device.engStatus.decommissioned',
};

function calcOfflineDays(lastOnlineTime: string | undefined): number {
  if (!lastOnlineTime) return 0;
  const lastOnline = new Date(lastOnlineTime);
  if (Number.isNaN(lastOnline.getTime())) return 0;
  const diff = Date.now() - lastOnline.getTime();
  return Math.max(0, Math.floor(diff / (1000 * 60 * 60 * 24)));
}

// ─── 1) 导入字段元数据：模板使用 + 解析器使用 ─────────────────────────────────

export interface ImportColumn {
  /** 中文表头 */
  zh: string;
  /** snake_case 字段名（与后端 CreateDeviceRequest 对齐） */
  snake: string;
  /** 是否必填 */
  required: boolean;
  /** 模板第 i 行示例（i=0/1/2） */
  example: readonly string[];
}

export const IMPORT_COLUMNS: readonly ImportColumn[] = [
  { zh: '基站编码', snake: 'serial_number', required: true,
    example: ['BAI-LTE-202604001', 'BAI-LTE-202604002', 'BAI-LTE-202604003'] },
  { zh: 'OUI', snake: 'oui', required: true,
    example: ['001E91', '001E91', '001E91'] },
  { zh: '运营商', snake: 'carrier', required: true,
    example: ['cmcc', 'cmcc', 'ctcc'] },
  { zh: '制式', snake: 'technology', required: true,
    example: ['lte', 'lte', 'nr'] },
  { zh: '产品型号', snake: 'product_class', required: false,
    example: ['X100W', 'X100W', 'X200W'] },
  { zh: '基站名称', snake: 'device_name', required: false,
    example: ['北京海淀中关村站', '北京朝阳CBD站', '上海浦东陆家嘴站'] },
  { zh: '站点ID', snake: 'site_id', required: false,
    example: ['SITE-BJ-001', 'SITE-BJ-002', 'SITE-SH-001'] },
  { zh: 'IP地址', snake: 'ip_address', required: false,
    example: ['', '', ''] },
  { zh: '经度', snake: 'longitude', required: false,
    example: ['116.310316', '116.460886', '121.504602'] },
  { zh: '纬度', snake: 'latitude', required: false,
    example: ['39.992177', '39.914585', '31.238068'] },
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
  /** 中文表头（与 IMPORT_COLUMNS 共享字段时同名） */
  zh: string;
  /** 取值器 */
  getValue: (d: Device, ctx: ExportCtx) => string | number;
}

export const EXPORT_COLUMNS: readonly ExportColumn[] = [
  { zh: '连接状态',
    getValue: (d, { t }) => d.connStatus === 'online' ? t('status.online') : t('status.offline') },
  { zh: '安装状态',
    getValue: (d, { t }) => {
      const key = ENG_STATUS_LABEL_KEYS[d.engStatus as EngStatus];
      return key ? t(key) : String(d.engStatus ?? '');
    } },
  // 共享字段（导入也需要）：基站编码 / OUI / 运营商 / 制式
  { zh: '基站编码',
    getValue: (d) => d.sn ?? '' },
  { zh: 'OUI',
    getValue: (d) => d.oui ?? '' },
  { zh: '运营商',
    getValue: (d) => d.carrier ?? '' },
  { zh: '制式',
    getValue: (d) => d.networkType ?? '' },
  { zh: '基站名称',
    getValue: (d) => d.name ?? '' },
  { zh: 'MAC地址',
    getValue: (d) => d.macAddress ?? '' },
  { zh: '设备分组',
    getValue: (d) => d.groupName ?? '' },
  { zh: '归属来源',
    getValue: (d, { t }) => {
      const src = d.sourceType ?? 'manual';
      return src === 'rule' ? t('device.sourceType.rule') : t('device.sourceType.manual');
    } },
  { zh: '经度',
    getValue: (d) => d.longitude ?? '' },
  { zh: '纬度',
    getValue: (d) => d.latitude ?? '' },
  { zh: '高度',
    getValue: (d) => d.gpsHeight ?? '' },
  { zh: '离线天数',
    getValue: (d) => d.connStatus === 'online' ? '-' : calcOfflineDays(d.lastOnlineTime) },
  { zh: '备注',
    getValue: (d) => d.remark ?? '' },
] as const;
