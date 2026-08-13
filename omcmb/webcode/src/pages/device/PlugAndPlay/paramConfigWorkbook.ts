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
  workbookMappings?: ParamConfigWorkbookMapping[];
}

export interface ParamConfigWorkbookMapping {
  displayName: string;
  sheet: string;
  header: string;
  trPath: string;
  source: 'system' | 'custom';
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
  deviceType?: ParamConfigDeviceType;
  productClass?: string;
  quickSettingsGroups?: readonly QuickSettingsGroup[];
  paramMappings?: readonly ParamMapping[];
  quickSettingFields?: readonly (GnbQuickSettingField & { condition?: string })[];
}

export const PARAM_MAPPING_SHEET = '参数映射';
export const PARAM_TEMPLATE_EXAMPLE_SERIAL = 'EXAMPLE-SN-001';
const PARAM_MAPPING_HEADERS = ['页面显示名称', '数据工作表', '参数列名', 'TRPath', '来源', '说明'] as const;

interface DynamicTemplateField {
  sheet: string;
  header: string;
  displayName: string;
  trPath: string;
  defaultValue: string;
}

function templateSheetForGroup(deviceType: ParamConfigDeviceType, group: QuickSettingsGroup): string {
  if (deviceType === 'GSM') return 'GSM';
  const key = `${group.id} ${group.objectPath ?? ''} ${group.params.map((param) => param.standardPath ?? '').join(' ')}`.toLowerCase();
  if (deviceType === 'gNB') {
    if (key.includes('1588') || key.includes('ptp') || key.includes('sync')) return '1588_CONFIGURATION';
    if (key.includes('cellconfig') || /(?:^|[.\s_-])cell(?:$|[.\s_-])/.test(key)) return 'CELL';
    if (key.includes('ipsec')) return 'IPSEC';
    if (key.includes('interface') || key.includes('network') || key.includes('wan') || key.includes('lan')) return 'INTERFACE';
    if (key.includes('plmn') || key.includes('core') || key.includes('amf') || key.includes('ngu')) return 'PLMN';
    if (key.includes('device') || key.includes('time') || key.includes('ntp') || key.includes('management') || key.includes('sync')) return 'DEVICE';
    return 'CELL';
  }
  if (key.includes('cellconfig') || /(?:^|[.\s_-])cell(?:$|[.\s_-])/.test(key)) return 'CELL';
  if (key.includes('ipsec')) return 'NETWORK_IPSEC';
  if (key.includes('1588') || key.includes('ptp') || key.includes('sync')) return '1588_CONFIGURATION';
  if (key.includes('network') || key.includes('wan') || key.includes('lan') || key.includes('ntp') || key.includes('time')) return 'NETWORK';
  if (key.includes('plmn') || key.includes('mme') || key.includes('enable')) return 'NETWORK_ENABLE';
  return 'CELL';
}

function templateSheetForParam(
  deviceType: ParamConfigDeviceType,
  group: QuickSettingsGroup,
  param: QuickSettingsParam,
  trPath: string,
): string {
  const identity = `${param.name} ${param.titleEn} ${param.leaf ?? ''} ${trPath}`
    .toLowerCase().replace(/[\s_-]+/g, '');
  if (/(?:1588|ptp|sync)/.test(identity)) return '1588_CONFIGURATION';
  const groupIdentity = `${group.id} ${group.titleEn} ${group.objectPath ?? ''}`.toLowerCase();
  if (/(?:1588|ptp|sync)/.test(groupIdentity)) return '1588_CONFIGURATION';
  if (deviceType !== 'gNB') {
    return templateSheetForGroup(deviceType, { ...group, params: [param] });
  }
  if (/(?:gnbname|gnbidlength|gnblength|gnblenth|gnbid)/.test(identity)) return 'DEVICE';
  if (identity.includes('plmn')) return 'PLMN';
  return templateSheetForGroup(deviceType, { ...group, params: [param] });
}

function templatePathVariants(template: string, multiInstance: boolean): Array<{ path: string; suffix: string }> {
  const placeholders = template.match(/\{[ij]\}/g)?.length ?? 0;
  if (!multiInstance || placeholders === 0) {
    return [{ path: template.replace(/\{[ij]\}/g, '1'), suffix: '' }];
  }
  const combinations = Array.from({ length: 2 ** placeholders }, (_, value) => (
    Array.from({ length: placeholders }, (_unused, index) => ((value >> (placeholders - index - 1)) & 1) + 1)
  ));
  return combinations.map((instances) => {
    let index = 0;
    return {
      path: template.replace(/\{[ij]\}/g, () => String(instances[index++])),
      suffix: ` [${instances.join('.')}]`,
    };
  });
}

function isCellLevelGroup(group: QuickSettingsGroup): boolean {
  const identity = `${group.id} ${group.titleEn} ${group.objectPath ?? ''}`.toLowerCase();
  return identity.includes('cellconfig')
    || /(?:^|[.\s_-])cell(?:$|[.\s_-])/.test(identity);
}

function lockProductBaseInstances(
  template: string,
  metadata: ParamConfigWorkbookMetadata,
): string {
  if (metadata.deviceType !== 'gNB' || !/BNQ/i.test(metadata.productClass ?? '')) return template;
  return template
    .replace(/(FAPService)\.\{[ij]\}/gi, '$1.1')
    .replace(/(CellConfig(?:\.[^.]+)*?)\.\{[ij]\}/gi, '$1.1');
}

function excludeNetworkTemplateParam(
  group: QuickSettingsGroup,
  param: QuickSettingsParam,
  trPath: string,
): boolean {
  const groupIdentity = `${group.id} ${group.titleEn} ${group.objectPath ?? ''}`.toLowerCase();
  if (!/(?:network|interface|wan|lan)/i.test(groupIdentity)) return false;
  const paramIdentity = `${param.name} ${param.titleEn} ${param.leaf ?? ''} ${trPath}`
    .toLowerCase().replace(/[\s_-]+/g, '');
  return paramIdentity.includes('vlan') || paramIdentity.includes('ipv6');
}

function excludeEnbNeighborTemplateParam(
  metadata: ParamConfigWorkbookMetadata,
  group: QuickSettingsGroup,
  param: QuickSettingsParam,
  trPath: string,
): boolean {
  if (metadata.deviceType !== 'eNB') return false;
  const identity = `${group.id} ${group.titleZh} ${group.titleEn} ${group.objectPath ?? ''} ${param.name} ${param.titleZh} ${param.titleEn} ${param.leaf ?? ''} ${trPath}`
    .toLowerCase();
  return /neighbor|neighbour|邻区/.test(identity);
}

function exampleValueForParam(
  param: QuickSettingsParam,
  metadata: ParamConfigWorkbookMetadata,
): string {
  if (param.defaultValue !== undefined && param.defaultValue !== '') return String(param.defaultValue);
  const path = param.standardPath ?? param.leaf ?? param.name;
  const mappings = metadata.paramMappings ?? [];
  const normalizedPath = normalizePath(path);
  const modelMapping = mappings.find((item) => (
    normalizePath(item.standardPath) === normalizedPath
    || normalizePath(item.privatePath) === normalizedPath
  )) ?? mappings.find((item) => {
    return [item.standardPath, item.privatePath].flatMap(metadataKeys).some(
      (key) => metadataKeys(path).includes(key),
    );
  });
  const configuredField = (metadata.quickSettingFields ?? []).find(
    (field) => canonicalHeader(field.id) === canonicalHeader(param.name),
  );
  const constraint = quickSettingConstraint(param, modelMapping, configuredField);
  if (constraint.options?.length) return String(constraint.options[0]);
  if (constraint.type === 'bool') return 'false';
  if (constraint.type === 'int') return String(constraint.minValue ?? 0);
  return '';
}

function dynamicTemplateFields(metadata?: ParamConfigWorkbookMetadata): DynamicTemplateField[] {
  if (!metadata?.deviceType) return [];
  const fixedFamilySeen = new Map<string, number>();
  return (metadata.quickSettingsGroups ?? []).flatMap((group) => {
    if (/(?:dscp|static-route)/i.test(group.id)) return [];
    const cellLevel = isCellLevelGroup(group);
    const ipsecTunnel = /ipsec/i.test(`${group.id} ${group.titleEn} ${group.objectPath ?? ''}`);
    const fixedMatch = cellLevel ? null : group.id.match(/^(.*)-(\d+)$/);
    if (fixedMatch) {
      const count = (fixedFamilySeen.get(fixedMatch[1]) ?? 0) + 1;
      fixedFamilySeen.set(fixedMatch[1], count);
      if (count > (ipsecTunnel ? 1 : 2)) return [];
    }
    const fixedInstance = fixedMatch ? fixedFamilySeen.get(fixedMatch[1]) : undefined;
    const fields = group.params.flatMap((param) => {
      if (param.readonly) return [];
      const rawTemplate = param.standardPath || (group.objectPath && param.leaf ? `${group.objectPath}${param.leaf}` : '');
      if (!rawTemplate) return [];
      const template = lockProductBaseInstances(rawTemplate, metadata);
      if (excludeNetworkTemplateParam(group, param, template)) return [];
      if (excludeEnbNeighborTemplateParam(metadata, group, param, template)) return [];
      const baseLabel = param.titleEn || param.name;
      return templatePathVariants(template, group.multiInstance && !cellLevel && !ipsecTunnel).map((variant) => ({
          sheet: templateSheetForParam(metadata.deviceType!, group, param, variant.path),
          header: `${baseLabel}${variant.suffix || (fixedInstance ? ` [${fixedInstance}]` : '')}`,
          displayName: baseLabel,
          trPath: variant.path,
          defaultValue: exampleValueForParam(param, metadata),
        }));
    });
    if (group.multiInstance && !cellLevel && !ipsecTunnel && !fixedMatch) {
      const instanceOrder = (header: string): string => header.match(/\s\[([\d.]+)\]$/)?.[1] ?? '';
      fields.sort((left, right) => instanceOrder(left.header).localeCompare(
        instanceOrder(right.header),
        undefined,
        { numeric: true },
      ));
    }
    return fields;
  });
}

function uniqueDynamicTemplateFields(metadata?: ParamConfigWorkbookMetadata): DynamicTemplateField[] {
  const occurrences = new Map<string, number>();
  return dynamicTemplateFields(metadata).map((field) => {
    const key = `${canonicalHeader(field.sheet)}.${canonicalHeader(field.header)}`;
    const occurrence = (occurrences.get(key) ?? 0) + 1;
    occurrences.set(key, occurrence);
    return occurrence === 1 ? field : { ...field, header: `${field.header} [${occurrence}]` };
  });
}

function appendParameterMappingSheet(
  workbook: XLSX.WorkBook,
  metadata?: ParamConfigWorkbookMetadata,
): void {
  if (!metadata?.deviceType || workbook.Sheets[PARAM_MAPPING_SHEET]) return;
  const templateSheets = getParamConfigTemplateSheets(metadata.deviceType) ?? {};
  const locations = Object.entries(templateSheets).flatMap(([sheet, headers]) => (
    headers.map((header) => ({ sheet, header, key: canonicalHeader(header) }))
  ));
  const seen = new Set<string>();
  const dynamicFields = uniqueDynamicTemplateFields(metadata);
  const rows: string[][] = dynamicFields.length > 0 ? dynamicFields.map((field) => [
    field.displayName, field.sheet, field.header, field.trPath, '系统预置', '当前产品即插即用页面字段映射',
  ]) : (metadata.quickSettingsGroups ?? [])
    .filter((group) => !/(?:dscp|static-route)/i.test(group.id))
    .flatMap((group) => group.params)
    .filter((param) => !param.readonly)
    .flatMap((param) => {
      const trPath = param.standardPath?.trim();
      if (!trPath || seen.has(trPath)) return [];
      seen.add(trPath);
      const candidates = [param.name, param.leaf, param.titleZh, param.titleEn, trPath.split('.').pop() ?? '']
        .map(canonicalHeader);
      const location = locations.find((item) => candidates.includes(item.key));
      return [[
        param.titleZh || param.titleEn || param.name,
        location?.sheet ?? '',
        location?.header ?? '',
        trPath,
        '系统预置',
        location ? '模板字段映射' : '页面参数；如需通过文件导入，请填写数据工作表和参数列名',
      ]];
    });
  if (dynamicFields.length === 0) for (const field of metadata.quickSettingFields ?? []) {
    const header = Array.isArray(field.name)
      ? String(field.name[field.name.length - 1] ?? '')
      : String(field.name ?? '');
    const keys = [field.id, header].map(canonicalHeader);
    const mapping = (metadata.paramMappings ?? []).find((item) => (
      [item.standardPath, item.privatePath].flatMap(metadataKeys).some((key) => keys.includes(key))
    ));
    const trPath = mapping?.standardPath || mapping?.privatePath;
    if (!trPath || seen.has(trPath)) continue;
    seen.add(trPath);
    const location = locations.find((item) => item.key === canonicalHeader(header));
    rows.push([
      header || field.id,
      location?.sheet ?? '',
      location?.header ?? header,
      trPath,
      '系统预置',
      location ? '页面字段映射' : '页面参数；如需通过文件导入，请填写数据工作表',
    ]);
  }
  rows.push(['', '', '', '', '用户自定义', '在数据工作表增加参数列，并在本行填写列名与产品参数模型中的具体 TRPath']);
  const worksheet = XLSX.utils.aoa_to_sheet([[...PARAM_MAPPING_HEADERS], ...rows]);
  worksheet['!cols'] = [{ wch: 24 }, { wch: 18 }, { wch: 28 }, { wch: 78 }, { wch: 14 }, { wch: 48 }];
  worksheet['!autofilter'] = { ref: `A1:F${rows.length + 1}` };
  XLSX.utils.book_append_sheet(workbook, worksheet, PARAM_MAPPING_SHEET);
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
    minValue: modelConstraint?.minValue ?? param.minValue,
    maxValue: modelConstraint?.maxValue ?? param.maxValue,
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
  const key = canonicalHeader(header.replace(/\s*\[[\d.]+\]\s*$/, ''));
  const configuredField = metadata?.quickSettingFields?.find((field) => {
    const name = Array.isArray(field.name) ? String(field.name[field.name.length - 1] ?? '') : String(field.name);
    return [field.id, name].flatMap(metadataKeys).includes(key);
  });
  for (const group of metadata?.quickSettingsGroups ?? []) {
    for (const param of group.params) {
      const keys = [param.name, param.leaf, param.standardPath, param.titleEn, param.titleZh]
        .filter(Boolean) as string[];
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

function appendPreservedParameterMappingSheet(
  workbook: XLSX.WorkBook,
  configs: readonly ParamConfigSpreadsheetRow[],
): void {
  if (workbook.Sheets[PARAM_MAPPING_SHEET]
    || !configs.some((config) => config.workbookMappings !== undefined)) return;
  const seen = new Set<string>();
  const mappings = configs.flatMap((config) => config.workbookMappings ?? []).filter((mapping) => {
    const key = `${canonicalHeader(mapping.sheet)}.${canonicalHeader(mapping.header)}.${mapping.trPath.trim().toLowerCase()}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
  const rows = mappings.map((mapping) => [
    mapping.displayName,
    mapping.sheet,
    mapping.header,
    mapping.trPath,
    mapping.source === 'custom' ? '用户自定义' : '系统预置',
    '保留自导入文件的参数映射',
  ]);
  const worksheet = XLSX.utils.aoa_to_sheet([[...PARAM_MAPPING_HEADERS], ...rows]);
  worksheet['!cols'] = [{ wch: 24 }, { wch: 18 }, { wch: 28 }, { wch: 78 }, { wch: 14 }, { wch: 48 }];
  worksheet['!autofilter'] = { ref: `A1:F${rows.length + 1}` };
  XLSX.utils.book_append_sheet(workbook, worksheet, PARAM_MAPPING_SHEET);
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
    appendPreservedParameterMappingSheet(workbook, configs);
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
  appendPreservedParameterMappingSheet(workbook, configs);
  return workbook;
}

export function createParamConfigTemplateWorkbook(
  deviceType: ParamConfigDeviceType,
  metadata?: ParamConfigWorkbookMetadata,
): XLSX.WorkBook {
  const sheets = getParamConfigTemplateSheets(deviceType);
  if (!sheets) throw new ParamConfigWorkbookError('empty_workbook');
  const dynamicFields = uniqueDynamicTemplateFields({ ...metadata, deviceType });
  if (dynamicFields.length > 0) {
    const workbook = XLSX.utils.book_new();
    const sheetNames = [...Object.keys(sheets)];
    if (dynamicFields.some((field) => field.sheet === '1588_CONFIGURATION')
      && !sheetNames.includes('1588_CONFIGURATION')) {
      const ipsecIndex = sheetNames.indexOf('IPSEC');
      sheetNames.splice(ipsecIndex >= 0 ? ipsecIndex : sheetNames.length, 0, '1588_CONFIGURATION');
    }
    for (const sheetName of sheetNames) {
      const fields = dynamicFields.filter((field) => field.sheet === sheetName);
      const headers = ['Serial Number', ...fields.map((field) => field.header)];
      const worksheet = XLSX.utils.aoa_to_sheet([
        headers,
        [PARAM_TEMPLATE_EXAMPLE_SERIAL, ...fields.map((field) => field.defaultValue)],
      ]);
      worksheet['!cols'] = headers.map((header) => ({ wch: Math.min(Math.max(header.length + 2, 16), 38) }));
      worksheet['!autofilter'] = { ref: `A1:${XLSX.utils.encode_col(headers.length - 1)}2` };
      XLSX.utils.book_append_sheet(workbook, worksheet, sheetName.slice(0, 31));
    }
    return workbook;
  }
  const defaults = getParamConfigTemplateDefaults(deviceType);
  const workbook = XLSX.utils.book_new();
  for (const [sheetName, headers] of Object.entries(sheets)) {
    const defaultRow = defaults[sheetName] ?? {};
    const worksheet = XLSX.utils.aoa_to_sheet([
      [...headers],
      headers.map((header) => (
        canonicalHeader(header) === 'SERIALNUMBER'
          ? PARAM_TEMPLATE_EXAMPLE_SERIAL
          : defaultRow[header] ?? ''
      )),
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
  appendParameterMappingSheet(workbook, metadata);
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
  const workbookMappings = parseWorkbookMappings(workbook);
  if (!workbook.Sheets[PARAM_MAPPING_SHEET]) {
    throw new ParamConfigWorkbookError('missing_columns', undefined, PARAM_MAPPING_SHEET);
  }
  if (workbookMappings.length === 0) {
    return parseGeneratedUnmappedTemplateWorkbook(workbook, deviceType, importedAt);
  }
  return parseMappedTemplateWorkbook(
    workbook,
    workbookMappings,
    deviceType,
    importedAt,
    metadata,
  );
}

function parseGeneratedUnmappedTemplateWorkbook(
  workbook: XLSX.WorkBook,
  deviceType: ParamConfigDeviceType,
  importedAt: string,
): ParamConfigSpreadsheetRow[] {
  const rowsBySerial = new Map<string, ParamConfigSpreadsheetRow>();
  for (const sheetName of workbook.SheetNames) {
    if (sheetName === PARAM_MAPPING_SHEET || sheetName === '__XOMC_OPTIONS') continue;
    const worksheet = workbook.Sheets[sheetName];
    if (!worksheet) continue;
    const matrix = XLSX.utils.sheet_to_json<unknown[]>(worksheet, { header: 1, defval: '', raw: true });
    if (matrix.length === 0) continue;
    const headers = (matrix[0] ?? []).map(normalizeHeader);
    const serialIndex = headers.findIndex((header) => canonicalHeader(header) === 'SERIALNUMBER');
    if (serialIndex < 0) throw new ParamConfigWorkbookError('missing_columns', 1, `${sheetName}.Serial Number`);

    matrix.slice(1).forEach((values, offset) => {
      const sheetRow = offset + 2;
      const serialNumber = String(values[serialIndex] ?? '').trim();
      if (!serialNumber) {
        if (values.every((value) => String(value ?? '').trim() === '')) return;
        throw new ParamConfigWorkbookError('invalid_row', sheetRow, 'serialNumber');
      }
      const parameters = Object.fromEntries(headers.flatMap((header, index) => (
        header ? [[header, values[index] ?? '']] : []
      )));
      const previous = rowsBySerial.get(serialNumber);
      rowsBySerial.set(serialNumber, {
        ...(previous ?? { deviceType, serialNumber, updatedBy: 'import', updatedAt: importedAt }),
        sheetParameters: {
          ...(previous?.sheetParameters ?? {}),
          [sheetName]: [...(previous?.sheetParameters?.[sheetName] ?? []), parameters],
        },
        workbookMappings: [],
      });
    });
  }
  if (rowsBySerial.size === 0) throw new ParamConfigWorkbookError('template_no_data');
  return Array.from(rowsBySerial.values());
}

function parseWorkbookMappings(workbook: XLSX.WorkBook): ParamConfigWorkbookMapping[] {
  const worksheet = workbook.Sheets[PARAM_MAPPING_SHEET];
  if (!worksheet) return [];
  return XLSX.utils.sheet_to_json<Record<string, unknown>>(worksheet, { defval: '', raw: true })
    .flatMap((row) => {
      const displayName = String(row['页面显示名称'] ?? '').trim();
      const sheet = String(row['数据工作表'] ?? '').trim();
      const header = String(row['参数列名'] ?? '').trim();
      const trPath = String(row.TRPath ?? '').trim();
      const system = String(row['来源'] ?? '').includes('系统');
      if (!header && !trPath) return [];
      if (system && (!sheet || !header)) return [];
      if (!sheet || !header || !trPath) throw new ParamConfigWorkbookError('invalid_row', undefined, PARAM_MAPPING_SHEET);
      return [{
        displayName: displayName || header,
        sheet,
        header,
        trPath,
        source: system ? 'system' as const : 'custom' as const,
      }];
    });
}

function constraintForMappedPath(
  mapping: ParamConfigWorkbookMapping,
  metadata?: ParamConfigWorkbookMetadata,
): ParameterConstraint | undefined {
  const normalizedPath = normalizePath(mapping.trPath);
  for (const group of metadata?.quickSettingsGroups ?? []) {
    for (const param of group.params) {
      const template = param.standardPath
        || (group.objectPath && param.leaf ? `${group.objectPath}${param.leaf}` : '');
      if (!template || normalizePath(template) !== normalizedPath) continue;
      const modelMapping = metadata?.paramMappings?.find((item) => (
        normalizePath(item.standardPath) === normalizedPath
        || normalizePath(item.privatePath) === normalizedPath
      ));
      const configuredField = metadata?.quickSettingFields?.find((field) => (
        canonicalHeader(field.id) === canonicalHeader(param.name)
      ));
      return quickSettingConstraint(param, modelMapping, configuredField);
    }
  }
  const modelMapping = metadata?.paramMappings?.find((item) => (
    normalizePath(item.standardPath) === normalizedPath
    || normalizePath(item.privatePath) === normalizedPath
  ));
  return modelMapping ? mappingConstraint(modelMapping) : undefined;
}

function validateMappedValue(
  value: unknown,
  mapping: ParamConfigWorkbookMapping,
  sheetRow: number,
  metadata?: ParamConfigWorkbookMetadata,
): void {
  const text = String(value ?? '').trim();
  if (!text) return;
  const constraint = constraintForMappedPath(mapping, metadata);
  if (!constraint) return;
  if (constraint.options?.length && !constraint.options.map(String).includes(text)) {
    throw new ParamConfigWorkbookError('invalid_row', sheetRow, mapping.header);
  }
  // Quick-setting enums are authoritative even when the parameter model exposes
  // their wire values as integers (for example LTE n25 and GSM penalty value 0).
  if (constraint.options?.length) return;
  if (constraint.type === 'int') {
    if (!/^-?\d+$/.test(text)) {
      throw new ParamConfigWorkbookError('invalid_row', sheetRow, mapping.header);
    }
    const numeric = Number(text);
    if ((constraint.minValue !== undefined && numeric < constraint.minValue)
      || (constraint.maxValue !== undefined && numeric > constraint.maxValue)) {
      throw new ParamConfigWorkbookError('invalid_row', sheetRow, mapping.header);
    }
  }
  if (constraint.type === 'string'
    && ((constraint.minValue !== undefined && text.length < constraint.minValue)
      || (constraint.maxValue !== undefined && text.length > constraint.maxValue))) {
    throw new ParamConfigWorkbookError('invalid_row', sheetRow, mapping.header);
  }
  const pattern = constraint.validationPattern ? modelPattern(constraint.validationPattern) : undefined;
  if (pattern && !pattern.test(text)) {
    throw new ParamConfigWorkbookError('invalid_row', sheetRow, mapping.header);
  }
}

function parseMappedTemplateWorkbook(
  workbook: XLSX.WorkBook,
  mappings: ParamConfigWorkbookMapping[],
  deviceType: ParamConfigDeviceType,
  importedAt: string,
  metadata?: ParamConfigWorkbookMetadata,
): ParamConfigSpreadsheetRow[] {
  const mappingByColumn = new Map<string, ParamConfigWorkbookMapping>();
  for (const mapping of mappings) {
    const key = `${canonicalHeader(mapping.sheet)}.${canonicalHeader(mapping.header)}`;
    if (mappingByColumn.has(key)) {
      throw new ParamConfigWorkbookError('invalid_row', undefined, PARAM_MAPPING_SHEET);
    }
    mappingByColumn.set(key, mapping);
  }

  const rowsBySerial = new Map<string, ParamConfigSpreadsheetRow>();
  for (const sheetName of workbook.SheetNames) {
    if (sheetName === PARAM_MAPPING_SHEET || sheetName === '__XOMC_OPTIONS') continue;
    const worksheet = workbook.Sheets[sheetName];
    if (!worksheet) continue;
    const matrix = XLSX.utils.sheet_to_json<unknown[]>(worksheet, { header: 1, defval: '', raw: true });
    if (matrix.length === 0) continue;
    const headers = (matrix[0] ?? []).map(normalizeHeader);
    const serialIndex = headers.findIndex((header) => canonicalHeader(header) === 'SERIALNUMBER');
    if (serialIndex < 0) throw new ParamConfigWorkbookError('missing_columns', 1, `${sheetName}.Serial Number`);
    const columnMappings = headers.map((header) => {
      if (!header || canonicalHeader(header) === 'SERIALNUMBER') return undefined;
      return mappingByColumn.get(`${canonicalHeader(sheetName)}.${canonicalHeader(header)}`);
    });
    headers.forEach((header, index) => {
      if (!header || canonicalHeader(header) === 'SERIALNUMBER' || columnMappings[index]) return;
      const hasValue = matrix.slice(1).some((values) => String(values[index] ?? '').trim() !== '');
      if (hasValue) throw new ParamConfigWorkbookError('missing_columns', 1, `${sheetName}.${header}`);
    });

    matrix.slice(1).forEach((values, offset) => {
      const sheetRow = offset + 2;
      const serialNumber = String(values[serialIndex] ?? '').trim();
      if (!serialNumber) {
        if (values.every((value) => String(value ?? '').trim() === '')) return;
        throw new ParamConfigWorkbookError('invalid_row', sheetRow, 'serialNumber');
      }
      const parameters = Object.fromEntries(headers.flatMap((header, index) => {
        if (!header) return [];
        const mapping = columnMappings[index];
        if (canonicalHeader(header) !== 'SERIALNUMBER' && !mapping) return [];
        if (mapping) validateMappedValue(values[index], mapping, sheetRow, metadata);
        return [[header, values[index] ?? '']];
      }));
      const previous = rowsBySerial.get(serialNumber);
      rowsBySerial.set(serialNumber, {
        ...(previous ?? { deviceType, serialNumber, updatedBy: 'import', updatedAt: importedAt }),
        sheetParameters: {
          ...(previous?.sheetParameters ?? {}),
          [sheetName]: [...(previous?.sheetParameters?.[sheetName] ?? []), parameters],
        },
        workbookMappings: mappings,
      });
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
