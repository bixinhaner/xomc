import * as XLSX from 'xlsx';
import ExcelJS from 'exceljs';
import {
  getParamConfigTemplateDefaults,
  getParamConfigTemplateSheets,
} from './paramConfigTemplate';
import { sanitizeRetiredParamConfigFields } from './retiredParamConfigFields';
import type { ParamMapping } from '@core/types/paramModel';
import type { QuickSettingsGroup, QuickSettingsParam } from '@core/types/quicksettings';
import {
  NR_CARRIER_BANDWIDTH_OPTIONS_BY_SCS,
  type GnbQuickSettingField,
} from './gnbQuickSettingsFields';

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
  readonly field?: keyof ParamConfigSpreadsheetRow | string;

  constructor(
    code: ParamConfigWorkbookErrorCode,
    row?: number,
    field?: keyof ParamConfigSpreadsheetRow | string,
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

const BOOLEAN_HEADER_PATTERN = /(?:ENABLE|ENABLED|PRIMARY|FRAGMENTATION|FORCEENCAPS|IS\s*DEFAULT)/i;
const INTEGER_HEADER_PATTERN = /(?:NUMBER|INDEX|BAND|FREQ|ARFCN|PCI|ECI|TAC|RANAC|NCI|(?:GNB|PLMN|VLAN|UNIT)[ _]?ID|INTERVAL|LENGTH|LENTH|POWER|OFFSET|SLOT|SYMBOL|SPACING|VLAN|DOMAIN|ASYMMETRY|TIMEOUT|KEYLIFE|LIFETIME|DELAY|MARGIN|支持频段|频点|子帧配比)/i;

interface ParameterConstraint {
  type: 'bool' | 'int' | 'string' | 'enum';
  range: string;
  options?: readonly string[];
  condition?: string;
  dependentOptions?: Readonly<Record<string, readonly string[]>>;
  dependsOnHeader?: string;
  minValue?: number;
  maxValue?: number;
  validationPattern?: string;
}

export interface ParamConfigWorkbookMetadata {
  quickSettingsGroups?: readonly QuickSettingsGroup[];
  paramMappings?: readonly ParamMapping[];
  quickSettingFields?: readonly (GnbQuickSettingField & { condition?: string })[];
}

const ENUM_OPTIONS: Readonly<Record<string, readonly string[]>> = {
  BANDWIDTH: ['5MHz', '6MHz', '10MHz', '15MHz', '20MHz', '25MHz', '50MHz', '75MHz', '100MHz'],
  BANDWIDTHDL: ['5MHz', '6MHz', '10MHz', '15MHz', '20MHz'],
  DLBANDWIDTH: ['5MHz', '10MHz', '15MHz', '20MHz', '25MHz', '30MHz', '40MHz', '50MHz', '60MHz', '70MHz', '80MHz', '90MHz', '100MHz'],
  ULBANDWIDTH: ['5MHz', '10MHz', '15MHz', '20MHz', '25MHz', '30MHz', '40MHz', '50MHz', '60MHz', '70MHz', '80MHz', '90MHz', '100MHz'],
  DUPLEXMODE: ['FDD', 'TDD'],
  SUBFRAMEASSIGNMENT: ['0', '1', '2', '6'],
  SYNCHRONIZATION: ['GNSS', 'PTP'],
  ADDRESSTYPE: ['DHCP', 'Static', 'DHCPv6', 'Staticv6'],
};

function metadataKeys(value: string): string[] {
  const normalized = canonicalHeader(value);
  const leaf = value.split('.').filter(Boolean).pop() ?? value;
  return Array.from(new Set([normalized, canonicalHeader(leaf)]));
}

function normalizePath(value: string): string {
  return value.trim().replace(/\.\d+(?=\.|$)/g, '.{i}').toLowerCase();
}

function quickSettingConstraint(
  param: QuickSettingsParam,
  mapping: ParamMapping | undefined,
  field: (GnbQuickSettingField & { condition?: string }) | undefined,
): ParameterConstraint {
  const modelConstraint = mapping ? mappingConstraint(mapping) : undefined;
  const options = field?.options?.map((option) => option.value)
    ?? param.enumOptions?.map((option) => option.value)
    ?? param.checkboxOptions
    ?? modelConstraint?.options;
  const normalizedType = String(param.type ?? '').toLowerCase();
  const type = modelConstraint?.type
    ?? (normalizedType === 'boolean' || normalizedType === 'bool' ? 'bool'
      : normalizedType.includes('int') ? 'int' : options?.length ? 'enum' : 'string');
  const range = field?.range
    || param.hint?.trim()
    || (param.minValue !== undefined || param.maxValue !== undefined
      ? `${param.minValue ?? '-∞'} ~ ${param.maxValue ?? '+∞'}`
      : options?.join('、'))
    || modelConstraint?.range
    || (type === 'bool' ? 'true、false' : type === 'int' ? '整数' : '字符串');
  return {
    type,
    range,
    condition: field?.condition,
    options: options?.length ? options : type === 'bool' ? ['true', 'false'] : undefined,
    dependentOptions: field?.control === 'dl-bandwidth' || field?.control === 'ul-bandwidth'
      ? Object.fromEntries(Object.entries(NR_CARRIER_BANDWIDTH_OPTIONS_BY_SCS).map(([scs, entries]) => (
        [scs, entries.map((entry) => entry.value)]
      )))
      : undefined,
    dependsOnHeader: field?.control === 'dl-bandwidth'
      ? 'SubcarrierSpacing(DL)'
      : field?.control === 'ul-bandwidth' ? 'SubcarrierSpacing(UL)' : undefined,
    minValue: modelConstraint?.minValue,
    maxValue: modelConstraint?.maxValue,
    validationPattern: modelConstraint?.validationPattern,
  };
}

function mappingConstraint(mapping: ParamMapping): ParameterConstraint {
  const normalizedType = mapping.dataType.toLowerCase().replace(/[\s_-]+/g, '');
  const type = normalizedType === 'boolean' || normalizedType === 'bool'
    ? 'bool' : normalizedType.includes('int') ? 'int' : 'string';
  const range = mapping.minValue !== undefined || mapping.maxValue !== undefined
    ? `${mapping.minValue ?? '-∞'} ~ ${mapping.maxValue ?? '+∞'}`
    : type === 'bool' ? 'true、false' : type === 'int' ? '整数' : '字符串';
  const enumOptions = mapping.enumValues?.split(',').map((value) => value.trim()).filter(Boolean);
  const min = Number(mapping.minValue);
  const max = Number(mapping.maxValue);
  const integerOptions = type === 'int' && Number.isInteger(min) && Number.isInteger(max)
    && max >= min && max - min <= 32
    ? Array.from({ length: max - min + 1 }, (_, index) => String(min + index))
    : undefined;
  return {
    type,
    range,
    options: enumOptions?.length ? enumOptions : type === 'bool' ? ['true', 'false'] : integerOptions,
    minValue: Number.isFinite(min) ? min : undefined,
    maxValue: Number.isFinite(max) ? max : undefined,
    validationPattern: mapping.validationPattern,
  };
}

function modelPattern(pattern: string): RegExp | undefined {
  try {
    const literal = pattern.match(/^\/(.*)\/([dgimsuvy]*)$/);
    return literal ? new RegExp(literal[1], literal[2]) : new RegExp(pattern);
  } catch {
    return undefined;
  }
}

function metadataConstraint(
  header: string,
  metadata?: ParamConfigWorkbookMetadata,
): ParameterConstraint | undefined {
  const key = canonicalHeader(header);
  const configuredField = metadata?.quickSettingFields?.find((field) => {
    const name = Array.isArray(field.name) ? String(field.name[field.name.length - 1] ?? '') : String(field.name);
    return [field.id, name].flatMap(metadataKeys).includes(key);
  });
  for (const group of metadata?.quickSettingsGroups ?? []) {
    for (const param of group.params) {
      const keys = [param.name, param.leaf, param.standardPath].filter(Boolean) as string[];
      if (!keys.flatMap(metadataKeys).includes(key)
        && (!configuredField || canonicalHeader(param.name) !== canonicalHeader(configuredField.id))) continue;
      const trPath = param.standardPath
        || (group.objectPath && param.leaf ? `${group.objectPath}${param.leaf}` : '');
      const mapping = metadata?.paramMappings?.find((item) => (
        normalizePath(item.standardPath) === normalizePath(trPath)
        || normalizePath(item.privatePath) === normalizePath(trPath)
      ));
      return quickSettingConstraint(param, mapping, configuredField);
    }
  }
  for (const mapping of metadata?.paramMappings ?? []) {
    if ([mapping.standardPath, mapping.privatePath].flatMap(metadataKeys).includes(key)) {
      return mappingConstraint(mapping);
    }
  }
  return undefined;
}

function parameterConstraint(header: string, metadata?: ParamConfigWorkbookMetadata): ParameterConstraint {
  const configured = metadataConstraint(header, metadata);
  if (configured) return configured;
  const canonical = canonicalHeader(header);
  const enumOptions = ENUM_OPTIONS[canonical] ?? ENUM_OPTIONS[normalizeHeader(header).toUpperCase()];
  if (enumOptions) return { type: 'enum', range: enumOptions.join('、'), options: enumOptions };
  if (canonical === 'SERIALNUMBER') return { type: 'string', range: '长度 1-64' };
  if (BOOLEAN_HEADER_PATTERN.test(header)) {
    return { type: 'bool', range: 'true、false', options: ['true', 'false'] };
  }
  if (/(?:PCI)/i.test(header)) return { type: 'int', range: '0-1007' };
  if (/(?:ARFCN|FREQUENCY|频点)/i.test(header)) return { type: 'int', range: '0-3279165' };
  if (/(?:TAC)/i.test(header)) return { type: 'int', range: '0-16777215' };
  if (/(?:RANAC)/i.test(header)) return { type: 'int', range: '0-255' };
  if (/(?:NCI)/i.test(header)) return { type: 'int', range: '0-68719476735' };
  if (/VLAN[ _]?ID/i.test(header)) return { type: 'int', range: '2-4094' };
  if (/PREFIX[ _]?LENGTH/i.test(header)) return { type: 'int', range: '0-128' };
  if (INTEGER_HEADER_PATTERN.test(header)) return { type: 'int', range: '整数' };
  return { type: 'string', range: '字符串' };
}

function parameterNote(header: string, metadata?: ParamConfigWorkbookMetadata): string {
  const constraint = parameterConstraint(header, metadata);
  const type = constraint.type === 'enum' ? '枚举' : constraint.type;
  return [
    `参数类型：${type}`,
    `取值范围：${constraint.range}`,
    constraint.options?.length ? `可选值：${constraint.options.join('、')}` : '',
    constraint.condition ? `显示条件：${constraint.condition}` : '',
  ].filter(Boolean).join('\n');
}

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
      const headers = Array.from(new Set(rows.flatMap((row) => Object.keys(row))));
      const worksheet = XLSX.utils.aoa_to_sheet([
        headers,
        ...rows.map((row) => headers.map((header) => row[header] ?? '')),
      ]);
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
  const worksheet = XLSX.utils.aoa_to_sheet([
    headers,
    ...data.map((row) => headers.map((header) => row[header as keyof typeof row] ?? '')),
  ]);
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

export async function enrichParamConfigWorkbook(
  workbook: XLSX.WorkBook,
  metadata?: ParamConfigWorkbookMetadata,
): Promise<ExcelJS.Workbook> {
  const source = XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer;
  const enriched = new ExcelJS.Workbook();
  await enriched.xlsx.load(source);

  const dataWorksheets = [...enriched.worksheets];
  let optionSheet: ExcelJS.Worksheet | undefined;
  let optionColumn = 0;
  const namedOptionCache = new Map<string, string>();
  const namedOptions = (options: readonly string[], prefix = 'XOMC_OPTIONS'): string => {
    const cacheKey = `${prefix}:${options.join('\u0000')}`;
    const cached = namedOptionCache.get(cacheKey);
    if (cached) return cached;
    optionSheet ??= enriched.addWorksheet('__XOMC_OPTIONS', { state: 'veryHidden' });
    optionColumn += 1;
    options.forEach((option, index) => {
      optionSheet!.getCell(index + 1, optionColumn).value = option;
    });
    const name = prefix === 'XOMC_OPTIONS' ? `${prefix}_${optionColumn}` : prefix;
    const column = optionSheet.getColumn(optionColumn).letter;
    enriched.definedNames.add(`'${optionSheet.name}'!$${column}$1:$${column}$${options.length}`, name);
    namedOptionCache.set(cacheKey, name);
    return name;
  };

  dataWorksheets.forEach((worksheet) => {
    worksheet.getRow(1).eachCell((headerCell, columnNumber) => {
      const header = String(headerCell.value ?? '');
      const constraint = parameterConstraint(header, metadata);
      headerCell.note = parameterNote(header, metadata);
      if (!constraint.options) return;
      let formula = `"${constraint.options.join(',')}"`;
      if (constraint.dependentOptions && constraint.dependsOnHeader) {
        Object.entries(constraint.dependentOptions).forEach(([key, options]) => {
          namedOptions(options, `XOMC_BW_${key}`);
        });
        const headerValues = worksheet.getRow(1).values;
        const dependencyColumn = Array.isArray(headerValues) ? headerValues.findIndex(
          (value) => canonicalHeader(value) === canonicalHeader(constraint.dependsOnHeader),
        ) : -1;
        if (dependencyColumn > 0) {
          const dependencyLetter = worksheet.getColumn(dependencyColumn).letter;
          formula = `INDIRECT("XOMC_BW_"&${dependencyLetter}2)`;
        }
      } else if (formula.length > 255) {
        formula = namedOptions(constraint.options);
      }
      for (let rowNumber = 2; rowNumber <= 1000; rowNumber += 1) {
        const rowFormula = constraint.dependentOptions
          ? formula.replace(/2\)$/, `${rowNumber})`)
          : formula;
        worksheet.getCell(rowNumber, columnNumber).dataValidation = {
          type: 'list',
          allowBlank: true,
          formulae: [rowFormula],
          showErrorMessage: true,
          errorTitle: '数据类型错误',
          error: '请从下拉列表选择有效值',
        };
      }
    });
  });
  return enriched;
}

/** Writes a workbook with hover notes and native dropdowns for bool/enum fields. */
export async function writeParamConfigWorkbookFile(
  workbook: XLSX.WorkBook,
  fileName: string,
  metadata?: ParamConfigWorkbookMetadata,
): Promise<void> {
  const enriched = await enrichParamConfigWorkbook(workbook, metadata);

  const bytes = await enriched.xlsx.writeBuffer();
  const blob = new Blob([bytes], {
    type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = fileName;
  anchor.click();
  URL.revokeObjectURL(url);
}

export function parseParamConfigWorkbook(
  bytes: ArrayBuffer,
  deviceType: ParamConfigDeviceType,
  importedAt: string,
  metadata?: ParamConfigWorkbookMetadata,
): ParamConfigSpreadsheetRow[] {
  const workbook = XLSX.read(bytes, { type: 'array' });
  const templateRows = parseDefaultTemplateWorkbook(workbook, deviceType, importedAt, metadata);
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
  const dataRows = rows
    .map((row, index) => ({ row, sheetRow: index + 2 }))
    .filter(({ row }) => !String(findValue(row, COLUMNS[0]) ?? '').startsWith('string（'));
  if (dataRows.length === 0) throw new ParamConfigWorkbookError('empty_workbook');

  return dataRows.map(({ row, sheetRow }) => {
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
  metadata?: ParamConfigWorkbookMetadata,
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
      if (!serialNumber || /^length\s*:/i.test(serialNumber) || serialNumber.startsWith('string（')) return;

      const sheetRow = headerRowIndex + rowOffset + 2;
      const rawParameters = sanitizeRetiredParamConfigFields({
        [sheetName]: [Object.fromEntries(headers.flatMap((header, index) => (
          header ? [[header, row[index] ?? '']] : []
        )))],
      })[sheetName][0];
      validateTemplateRow(headers, rawParameters, sheetRow, metadata);
      if (deviceType === 'gNB' && sheetName === 'INTERFACE') {
        validateGnbNetworkRow(rawParameters, sheetRow);
      }
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

function validateTemplateRow(
  headers: readonly string[],
  row: Record<string, unknown>,
  sheetRow: number,
  metadata?: ParamConfigWorkbookMetadata,
): void {
  for (const header of headers) {
    if (!header || canonicalHeader(header) === 'SERIALNUMBER') continue;
    const value = String(row[header] ?? '').trim();
    if (header.trim().startsWith('*') && !value) {
      throw new ParamConfigWorkbookError('invalid_row', sheetRow, header);
    }
    if (!value) continue;

    const constraint = metadataConstraint(header, metadata);
    if (!constraint) continue;
    if (constraint.options?.length && !constraint.options.map(String).includes(value)) {
      throw new ParamConfigWorkbookError('invalid_row', sheetRow, header);
    }
    if (constraint.type === 'int') {
      if (!/^-?\d+$/.test(value)) {
        throw new ParamConfigWorkbookError('invalid_row', sheetRow, header);
      }
      const numeric = Number(value);
      if ((constraint.minValue !== undefined && numeric < constraint.minValue)
        || (constraint.maxValue !== undefined && numeric > constraint.maxValue)) {
        throw new ParamConfigWorkbookError('invalid_row', sheetRow, header);
      }
    }
    if (constraint.type === 'string'
      && ((constraint.minValue !== undefined && value.length < constraint.minValue)
        || (constraint.maxValue !== undefined && value.length > constraint.maxValue))) {
      throw new ParamConfigWorkbookError('invalid_row', sheetRow, header);
    }
    const pattern = constraint.validationPattern ? modelPattern(constraint.validationPattern) : undefined;
    if (pattern && !pattern.test(value)) {
      throw new ParamConfigWorkbookError('invalid_row', sheetRow, header);
    }
    if (constraint.dependentOptions && constraint.dependsOnHeader) {
      const dependency = String(row[constraint.dependsOnHeader] ?? '').trim();
      const allowed = constraint.dependentOptions[dependency];
      if (!allowed?.includes(value)) {
        throw new ParamConfigWorkbookError('invalid_row', sheetRow, header);
      }
    }
  }
}

const GNB_ADDRESS_TYPES = new Set(['DHCP', 'Static', 'DHCPv6', 'Staticv6']);

function validateGnbNetworkRow(row: Record<string, unknown>, sheetRow: number): void {
  const addressType = String(row['Address Type'] ?? '').trim();
  if (!GNB_ADDRESS_TYPES.has(addressType)) {
    throw new ParamConfigWorkbookError('invalid_row', sheetRow, 'Address Type');
  }

  const requireValue = (header: string) => {
    if (!String(row[header] ?? '').trim()) {
      throw new ParamConfigWorkbookError('invalid_row', sheetRow, header);
    }
  };
  if (addressType === 'Static' || addressType === 'Staticv6') {
    requireValue('IP Address');
    requireValue('Gateway');
  }
  if (addressType === 'Static') {
    requireValue('Subnet Mask');
  }
  if (addressType === 'Staticv6') {
    const prefix = Number(String(row['Prefix Length'] ?? '').trim());
    if (!Number.isInteger(prefix) || prefix < 0 || prefix > 128) {
      throw new ParamConfigWorkbookError('invalid_row', sheetRow, 'Prefix Length');
    }
  }
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
