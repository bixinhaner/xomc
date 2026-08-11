import * as XLSX from 'xlsx';
import {
  getParamConfigTemplateDefaults,
  getParamConfigTemplateSheets,
} from './paramConfigTemplate';
import { sanitizeRetiredParamConfigFields } from './retiredParamConfigFields';

export type ParamConfigDeviceType = 'eNB' | 'gNB' | 'GSM';

export interface ParamConfigSpreadsheetRow {
  deviceType?: ParamConfigDeviceType;
  serialNumber: string;
  cellName?: string;
  bandsSupport?: number;
  bandWidth?: string;
  frequency?: number;
  subframeAssignment?: number;
  updatedBy?: string;
  updatedAt?: string;
  sheetParameters?: Record<string, Record<string, unknown>[]>;
}

export type ParamConfigWorkbookErrorCode =
  | 'empty_workbook'
  | 'template_no_data'
  | 'missing_columns'
  | 'invalid_row';

export type ParamConfigImportMode = 'append' | 'replace';

export class ParamConfigWorkbookError extends Error {
  readonly code: ParamConfigWorkbookErrorCode;
  readonly row?: number;
  readonly field?: keyof ParamConfigSpreadsheetRow;

  constructor(
    code: ParamConfigWorkbookErrorCode,
    row?: number,
    field?: keyof ParamConfigSpreadsheetRow,
  ) {
    super(code);
    this.name = 'ParamConfigWorkbookError';
    this.code = code;
    this.row = row;
    this.field = field;
  }
}

type ColumnKey = keyof ParamConfigSpreadsheetRow;

interface Column {
  key: ColumnKey;
  header: string;
  aliases: readonly string[];
  width: number;
  required?: boolean;
}

const COLUMNS: readonly Column[] = [
  { key: 'serialNumber', header: '基站编码', aliases: ['基站编码', 'serialNumber', 'serial_number'], width: 22, required: true },
  { key: 'cellName', header: '基站名称', aliases: ['基站名称', 'cellName', 'cell_name'], width: 20 },
  { key: 'bandsSupport', header: '支持频段', aliases: ['支持频段', 'bandsSupport', 'bands_support'], width: 14, required: true },
  { key: 'bandWidth', header: '带宽', aliases: ['带宽', 'bandWidth', 'band_width'], width: 14, required: true },
  { key: 'frequency', header: '频点', aliases: ['频点', '频率', 'frequency'], width: 16, required: true },
  { key: 'subframeAssignment', header: '子帧配比', aliases: ['子帧配比', 'subframeAssignment', 'subframe_assignment'], width: 16, required: true },
  { key: 'updatedBy', header: '更新人', aliases: ['更新人', 'updatedBy', 'updated_by'], width: 14 },
  { key: 'updatedAt', header: '更新时间', aliases: ['更新时间', 'updatedAt', 'updated_at'], width: 22 },
] as const;

function normalizeHeader(value: unknown): string {
  return String(value ?? '').trim();
}

function canonicalHeader(value: unknown): string {
  return normalizeHeader(value)
    .replace(/^\*/, '')
    .replace(/[\s_]+/g, '')
    .toUpperCase();
}

function findValue(row: Record<string, unknown>, column: Column): unknown {
  for (const alias of column.aliases) {
    if (Object.prototype.hasOwnProperty.call(row, alias)) return row[alias];
  }
  return undefined;
}

function requiredText(value: unknown, row: number, field: ColumnKey): string {
  const text = String(value ?? '').trim();
  if (!text) throw new ParamConfigWorkbookError('invalid_row', row, field);
  return text;
}

function optionalText(value: unknown): string | undefined {
  const text = String(value ?? '').trim();
  return text || undefined;
}

function integerInRange(
  value: unknown,
  row: number,
  field: ColumnKey,
  min: number,
  max: number,
): number {
  const parsed = typeof value === 'number' ? value : Number(String(value ?? '').trim());
  if (!Number.isInteger(parsed) || parsed < min || parsed > max) {
    throw new ParamConfigWorkbookError('invalid_row', row, field);
  }
  return parsed;
}

function optionalIntegerInRange(
  value: unknown,
  row: number,
  field: ColumnKey,
  min: number,
  max: number,
): number | undefined {
  if (String(value ?? '').trim() === '') return undefined;
  return integerInRange(value, row, field, min, max);
}

function normalizeBandwidth(value: unknown, row: number): string {
  const raw = requiredText(value, row, 'bandWidth');
  const match = raw.match(/^(\d+)(?:\s*MHz)?$/i);
  const supported = new Set([5, 6, 10, 15, 20, 25, 50, 75, 100]);
  const bandwidth = match ? Number(match[1]) : Number.NaN;
  if (!supported.has(bandwidth)) {
    throw new ParamConfigWorkbookError('invalid_row', row, 'bandWidth');
  }
  return `${bandwidth}MHz`;
}

export function createParamConfigWorkbook(
  configs: readonly ParamConfigSpreadsheetRow[],
): XLSX.WorkBook {
  const sourceSheetNames = Array.from(new Set(
    configs.flatMap((config) => Object.keys(config.sheetParameters ?? {})),
  ));
  if (sourceSheetNames.length > 0) {
    const workbook = XLSX.utils.book_new();
    for (const sheetName of sourceSheetNames) {
      const rows = configs.flatMap((config) => (
        sanitizeRetiredParamConfigFields(config.sheetParameters ?? {})[sheetName] ?? []
      ));
      if (rows.length === 0) continue;
      const worksheet = XLSX.utils.json_to_sheet(rows);
      XLSX.utils.book_append_sheet(workbook, worksheet, sheetName.slice(0, 31));
    }
    return workbook;
  }

  const headers = COLUMNS.map((column) => column.header);
  const data = configs.map((config) => ({
    基站编码: config.serialNumber,
    基站名称: config.cellName ?? '',
    支持频段: config.bandsSupport ?? '',
    带宽: config.bandWidth ?? '',
    频点: config.frequency ?? '',
    子帧配比: config.subframeAssignment ?? '',
    更新人: config.updatedBy ?? '',
    更新时间: config.updatedAt ?? '',
  }));
  const worksheet = data.length > 0
    ? XLSX.utils.json_to_sheet(data, { header: headers })
    : XLSX.utils.aoa_to_sheet([headers]);
  worksheet['!cols'] = COLUMNS.map((column) => ({ wch: column.width }));
  worksheet['!autofilter'] = { ref: `A1:H${Math.max(data.length + 1, 1)}` };

  const workbook = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(workbook, worksheet, '参数配置');
  return workbook;
}

export function createParamConfigTemplateWorkbook(
  deviceType: ParamConfigDeviceType,
): XLSX.WorkBook {
  const sheets = getParamConfigTemplateSheets(deviceType);
  if (!sheets) throw new ParamConfigWorkbookError('empty_workbook');
  const defaults = getParamConfigTemplateDefaults(deviceType);
  const workbook = XLSX.utils.book_new();
  for (const [sheetName, headers] of Object.entries(sheets)) {
    const defaultRow = defaults[sheetName] ?? {};
    const worksheet = XLSX.utils.aoa_to_sheet([
      [...headers],
      headers.map((header) => defaultRow[header] ?? ''),
    ]);
    worksheet['!cols'] = headers.map((header) => ({
      wch: Math.min(Math.max(header.length + 2, 14), 34),
    }));
    worksheet['!autofilter'] = { ref: `A1:${XLSX.utils.encode_col(headers.length - 1)}2` };
    XLSX.utils.book_append_sheet(workbook, worksheet, sheetName.slice(0, 31));
  }
  return workbook;
}

export function parseParamConfigWorkbook(
  bytes: ArrayBuffer,
  deviceType: ParamConfigDeviceType,
  importedAt: string,
): ParamConfigSpreadsheetRow[] {
  const workbook = XLSX.read(bytes, { type: 'array' });
  const templateRows = parseDefaultTemplateWorkbook(workbook, deviceType, importedAt);
  if (templateRows) return templateRows;

  const firstSheetName = workbook.SheetNames[0];
  if (!firstSheetName) throw new ParamConfigWorkbookError('empty_workbook');
  const worksheet = workbook.Sheets[firstSheetName];
  const matrix = XLSX.utils.sheet_to_json<unknown[]>(worksheet, { header: 1, defval: '' });
  const headers = (matrix[0] ?? []).map(normalizeHeader);
  if (headers.length === 0) throw new ParamConfigWorkbookError('empty_workbook');

  const missingRequiredColumn = COLUMNS.some(
    (column) => column.required && !column.aliases.some((alias) => headers.includes(alias)),
  );
  if (missingRequiredColumn) throw new ParamConfigWorkbookError('missing_columns');

  const rows = XLSX.utils.sheet_to_json<Record<string, unknown>>(worksheet, {
    defval: '',
    raw: true,
  });
  if (rows.length === 0) throw new ParamConfigWorkbookError('empty_workbook');

  return rows.map((row, index) => {
    const sheetRow = index + 2;
    const serialNumber = requiredText(findValue(row, COLUMNS[0]), sheetRow, 'serialNumber');
    const bandsSupport = integerInRange(findValue(row, COLUMNS[2]), sheetRow, 'bandsSupport', 1, 85);
    const bandWidth = normalizeBandwidth(findValue(row, COLUMNS[3]), sheetRow);
    const frequency = integerInRange(findValue(row, COLUMNS[4]), sheetRow, 'frequency', 0, 3279165);
    const subframeAssignment = integerInRange(
      findValue(row, COLUMNS[5]),
      sheetRow,
      'subframeAssignment',
      0,
      6,
    );
    if (![0, 1, 2, 6].includes(subframeAssignment)) {
      throw new ParamConfigWorkbookError('invalid_row', sheetRow, 'subframeAssignment');
    }

    return {
      deviceType,
      serialNumber,
      cellName: optionalText(findValue(row, COLUMNS[1])),
      bandsSupport,
      bandWidth,
      frequency,
      subframeAssignment,
      updatedBy: optionalText(findValue(row, COLUMNS[6])) ?? 'import',
      updatedAt: optionalText(findValue(row, COLUMNS[7])) ?? importedAt,
    };
  });
}

function parseDefaultTemplateWorkbook(
  workbook: XLSX.WorkBook,
  deviceType: ParamConfigDeviceType,
  importedAt: string,
): ParamConfigSpreadsheetRow[] | undefined {
  const hasSerialNumberSheet = workbook.SheetNames.some((sheetName) => {
    const worksheet = workbook.Sheets[sheetName];
    if (!worksheet) return false;
    const matrix = XLSX.utils.sheet_to_json<unknown[]>(worksheet, {
      header: 1,
      defval: '',
      raw: true,
    });
    return matrix.slice(0, 5).some(
      (row) => row.some((cell) => canonicalHeader(cell) === 'SERIALNUMBER'),
    );
  });
  if (!hasSerialNumberSheet) return undefined;

  const rowsBySerial = new Map<string, ParamConfigSpreadsheetRow>();

  for (const sheetName of workbook.SheetNames) {
    const worksheet = workbook.Sheets[sheetName];
    if (!worksheet) continue;
    const matrix = XLSX.utils.sheet_to_json<unknown[]>(worksheet, {
      header: 1,
      defval: '',
      raw: true,
    });
    const headerRowIndex = matrix
      .slice(0, 5)
      .findIndex((row) => row.some((cell) => canonicalHeader(cell) === 'SERIALNUMBER'));
    if (headerRowIndex < 0) continue;

    const headers = (matrix[headerRowIndex] ?? []).map(normalizeHeader);
    const serialIndex = headers.findIndex((header) => canonicalHeader(header) === 'SERIALNUMBER');
    const indexOf = (header: string) => headers.indexOf(header);
    const is4GCell = sheetName === 'CELL' && headers.includes('*SERIAL_NUMBER');
    const is5GCell = sheetName === 'CELL'
      && headers.includes('*Serial Number')
      && headers.includes('gNB Name');

    matrix.slice(headerRowIndex + 1).forEach((row, rowOffset) => {
      const serialNumber = String(row[serialIndex] ?? '').trim();
      if (!serialNumber || /^length\s*:/i.test(serialNumber)) return;

      const sheetRow = headerRowIndex + rowOffset + 2;
      const rawParameters = sanitizeRetiredParamConfigFields({
        [sheetName]: [Object.fromEntries(headers.flatMap((header, index) => (
          header ? [[header, row[index] ?? '']] : []
        )))],
      })[sheetName][0];
      const previous = rowsBySerial.get(serialNumber);
      const sheetParameters = {
        ...(previous?.sheetParameters ?? {}),
        [sheetName]: [
          ...(previous?.sheetParameters?.[sheetName] ?? []),
          rawParameters,
        ],
      };
      let next: ParamConfigSpreadsheetRow = {
        ...(previous ?? {
          deviceType,
          serialNumber,
          updatedBy: 'import',
          updatedAt: importedAt,
        }),
        sheetParameters,
      };

      if (is4GCell) {
        next = {
          ...next,
          cellName: optionalText(row[indexOf('CELL_NAME')]) ?? next.cellName,
          bandsSupport: optionalIntegerInRange(
            row[indexOf('*BAND')],
            sheetRow,
            'bandsSupport',
            1,
            85,
          ) ?? next.bandsSupport,
          bandWidth: optionalText(row[indexOf('*BANDWIDTH_DL')]) ?? next.bandWidth,
          frequency: optionalIntegerInRange(
            row[indexOf('*EARFCN_DL')],
            sheetRow,
            'frequency',
            0,
            3279165,
          ) ?? next.frequency,
          subframeAssignment: optionalIntegerInRange(
            row[indexOf('SUBFRAME_ASSIGNMENT')],
            sheetRow,
            'subframeAssignment',
            0,
            6,
          ) ?? next.subframeAssignment,
        };
      } else if (is5GCell) {
        next = {
          ...next,
          cellName: optionalText(row[indexOf('gNB Name')]) ?? next.cellName,
          bandsSupport: optionalIntegerInRange(
            row[indexOf('Freq BandIndicator')],
            sheetRow,
            'bandsSupport',
            1,
            1024,
          ) ?? next.bandsSupport,
          bandWidth: optionalText(row[indexOf('DLBandwidth')]) ?? next.bandWidth,
          frequency: optionalIntegerInRange(
            row[indexOf('NRARFCNDL')],
            sheetRow,
            'frequency',
            0,
            3279165,
          ) ?? next.frequency,
        };
      }

      rowsBySerial.set(serialNumber, next);
    });
  }

  if (rowsBySerial.size === 0) throw new ParamConfigWorkbookError('template_no_data');
  return Array.from(rowsBySerial.values());
}

export function mergeImportedParamConfigs<T extends { serialNumber: string }>(
  existing: readonly T[],
  imported: readonly T[],
  mode: ParamConfigImportMode,
): T[] {
  if (mode === 'replace') return [...imported];

  const merged = new Map(existing.map((item) => [item.serialNumber, item]));
  imported.forEach((item) => merged.set(item.serialNumber, item));
  return [...merged.values()];
}
