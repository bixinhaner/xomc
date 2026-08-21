import { getParamConfigTemplateSheets } from './paramConfigTemplate';
import { primaryInstanceHeader, type ParamConfigDeviceType } from './paramConfigWorkbook';

const COMMON_EXCLUDED_SHEET_FIELDS: Record<string, readonly string[]> = {
  DEVICE: ['Time Zone Term'],
  INTERFACE: ['Interface Name', 'Address Type', 'Prefix Length', 'Bear Type', 'Vlan Name', 'OMC IP'],
  IPSEC: ['FORCEENCAPS'],
};

interface AllocationDefaults {
  start: number;
  end: number;
  step: number;
  reserved: unknown[];
}

const DEFAULT_GNB_ID_ALLOCATION: AllocationDefaults = { start: 1, end: 16_777_215, step: 1, reserved: [] };
const DEFAULT_PCI_ALLOCATION: AllocationDefaults = { start: 0, end: 1007, step: 1, reserved: [] };

function objectValue(input: unknown): Record<string, unknown> {
  return input && typeof input === 'object' && !Array.isArray(input)
    ? input as Record<string, unknown>
    : {};
}

function mergeAllocationDefaults(
  defaults: AllocationDefaults,
  saved: unknown,
): AllocationDefaults {
  const value = objectValue(saved);
  return {
    ...defaults,
    ...value,
    reserved: Array.isArray(value.reserved) ? value.reserved : defaults.reserved,
  };
}

export function sanitizeCommonParamConfig<T extends Record<string, unknown>>(config: T): T {
  const sheetParameters = config.sheetParameters;
  if (!sheetParameters || typeof sheetParameters !== 'object' || Array.isArray(sheetParameters)) {
    return config;
  }

  const sanitizedSheets = { ...sheetParameters } as Record<string, unknown>;
  for (const [sheet, excludedFields] of Object.entries(COMMON_EXCLUDED_SHEET_FIELDS)) {
    const rows = sanitizedSheets[sheet];
    if (!Array.isArray(rows)) continue;
    sanitizedSheets[sheet] = rows.map((row) => {
      if (!row || typeof row !== 'object' || Array.isArray(row)) return row;
      const sanitizedRow = { ...row } as Record<string, unknown>;
      excludedFields.forEach((field) => delete sanitizedRow[field]);
      return sanitizedRow;
    });
  }

  return { ...config, sheetParameters: sanitizedSheets };
}

export function withInitialCommonRadioInstance<T extends Record<string, unknown>>(
  config: T,
  deviceType: ParamConfigDeviceType,
  productClass?: string,
): T & { sheetParameters: Record<string, unknown> } {
  const sheetName = deviceType === 'GSM' ? 'GSM' : 'CELL';
  const header = primaryInstanceHeader(deviceType, sheetName, productClass);
  if (!header) return config as T & { sheetParameters: Record<string, unknown> };

  const sheets = config.sheetParameters && typeof config.sheetParameters === 'object'
    && !Array.isArray(config.sheetParameters)
    ? config.sheetParameters as Record<string, unknown>
    : {};
  if (Array.isArray(sheets[sheetName])) {
    return { ...config, sheetParameters: sheets };
  }

  const templateHeaders = getParamConfigTemplateSheets(deviceType)?.[sheetName] ?? [];
  const row = Object.fromEntries(templateHeaders.map((field) => [field, '']));
  return {
    ...config,
    sheetParameters: {
      ...sheets,
      [sheetName]: [{ ...row, [header]: 1 }],
    },
  };
}

export function withInitialCommonParamConfig<T extends Record<string, unknown>>(
  config: T,
  deviceType: ParamConfigDeviceType,
  productClass?: string,
): T & { sheetParameters: Record<string, unknown> } {
  const initialized: Record<string, unknown> = { ...config, deviceType };
  if (deviceType === 'gNB') {
    initialized.gnbIdAllocation = mergeAllocationDefaults(
      DEFAULT_GNB_ID_ALLOCATION,
      initialized.gnbIdAllocation,
    );
    initialized.pciAllocation = mergeAllocationDefaults(
      DEFAULT_PCI_ALLOCATION,
      initialized.pciAllocation,
    );
    initialized.gnbIdLength = initialized.gnbIdLength ?? 24;
  }
  return withInitialCommonRadioInstance(initialized as T, deviceType, productClass);
}
