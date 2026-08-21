export type ParamConfigSource = 'import' | 'batch_plan' | 'manual';
export type ParamConfigValidationStatus = 'valid' | 'conflict' | 'incomplete';

export interface ParamConfigInsightSource {
  serialNumber: string;
  deviceType?: string;
  updatedBy?: string;
  gnbId?: unknown;
  pci?: unknown;
  bandsSupport?: unknown;
  freqBandIndicator?: unknown;
  bandWidth?: unknown;
  dlbandwidth?: unknown;
  frequency?: unknown;
  nrarfcnndl?: unknown;
  nrarfcndl?: unknown;
  ssbFrequency?: unknown;
  tac?: unknown;
  sheetParameters?: Record<string, Record<string, unknown>[]>;
  workbookMappings?: Array<{
    sheet: string;
    header: string;
    trPath: string;
  }>;
}

export interface ParamConfigInsight {
  source: ParamConfigSource;
  validationStatus: ParamConfigValidationStatus;
  gnbId?: string;
  pci?: string;
  band?: string;
  bandwidth?: string;
  frequency?: string;
  ssbFrequency?: string;
  tac?: string;
}

function text(value: unknown): string | undefined {
  if (value === undefined || value === null || String(value).trim() === '') return undefined;
  return String(value).trim();
}

function firstSheetValue(
  source: ParamConfigInsightSource,
  sheet: string,
  ...headers: string[]
): string | undefined {
  const row = source.sheetParameters?.[sheet]?.[0];
  if (!row) return undefined;
  for (const header of headers) {
    const value = text(row[header]);
    if (value !== undefined) return value;
  }
  return undefined;
}

function mappedPathValue(
  source: ParamConfigInsightSource,
  leafNames: readonly string[],
): string | undefined {
  const expected = new Set(leafNames.map((name) => name.replace(/[^a-z0-9]/gi, '').toUpperCase()));
  for (const mapping of source.workbookMappings ?? []) {
    if (/\.NeighborList\./i.test(mapping.trPath)) continue;
    const leaf = mapping.trPath.split('.').filter(Boolean).at(-1)?.replace(/[^a-z0-9]/gi, '').toUpperCase();
    if (!leaf || !expected.has(leaf)) continue;
    for (const row of source.sheetParameters?.[mapping.sheet] ?? []) {
      const value = text(row[mapping.header]);
      if (value !== undefined) return value;
    }
  }
  return undefined;
}

export function paramConfigSource(source: ParamConfigInsightSource): ParamConfigSource {
  const updatedBy = source.updatedBy?.trim().toLowerCase();
  if (updatedBy === 'batch-plan' || updatedBy === 'batch_plan') return 'batch_plan';
  if (updatedBy === 'import') return 'import';
  return 'manual';
}

export function buildParamConfigInsights(
  configs: readonly ParamConfigInsightSource[],
): Map<string, ParamConfigInsight> {
  const gnbIdCounts = new Map<string, number>();
  const pciCounts = new Map<string, number>();
  const extracted = configs.map((config) => {
    const gnbId = text(config.gnbId)
      ?? mappedPathValue(config, ['GNBID'])
      ?? firstSheetValue(config, 'CELL', '*gNB ID', 'gNB ID')
      ?? firstSheetValue(config, 'DEVICE', '*gNB ID', 'gNB ID');
    const pci = text(config.pci)
      ?? mappedPathValue(config, ['PCI', 'PhyCellID', 'PhysicalCellID'])
      ?? firstSheetValue(config, 'CELL', '*PCI', 'PCI');
    if (config.deviceType === 'gNB' && gnbId) gnbIdCounts.set(gnbId, (gnbIdCounts.get(gnbId) ?? 0) + 1);
    if (config.deviceType === 'gNB' && pci) pciCounts.set(pci, (pciCounts.get(pci) ?? 0) + 1);
    return { config, gnbId, pci };
  });

  return new Map(extracted.map(({ config, gnbId, pci }) => {
    const incomplete = config.deviceType === 'gNB' && (!gnbId || !pci);
    const conflict = config.deviceType === 'gNB'
      && ((gnbIdCounts.get(gnbId ?? '') ?? 0) > 1 || (pciCounts.get(pci ?? '') ?? 0) > 1);
    const insight: ParamConfigInsight = {
      source: paramConfigSource(config),
      validationStatus: conflict ? 'conflict' : incomplete ? 'incomplete' : 'valid',
      gnbId,
      pci,
      band: text(config.freqBandIndicator ?? config.bandsSupport)
        ?? mappedPathValue(config, ['FreqBandIndicator', 'FreqBandIndicatorNR', 'Band'])
        ?? firstSheetValue(config, 'CELL', 'Freq BandIndicator', '*BAND', 'Band'),
      bandwidth: text(config.dlbandwidth ?? config.bandWidth)
        ?? mappedPathValue(config, ['DLBandwidth', 'CarrierBandwidth', 'Bandwidth'])
        ?? firstSheetValue(config, 'CELL', 'DLBandwidth', '*BANDWIDTH_DL', 'DL Carrier Bandwidth'),
      frequency: text(config.nrarfcnndl ?? config.nrarfcndl ?? config.frequency)
        ?? mappedPathValue(config, ['NRARFCNDL', 'EARFCNDL'])
        ?? firstSheetValue(config, 'CELL', 'NRARFCNDL', '*EARFCN_DL'),
      ssbFrequency: text(config.ssbFrequency)
        ?? mappedPathValue(config, ['SSBFrequency'])
        ?? firstSheetValue(config, 'CELL', 'SSB Frequency'),
      tac: text(config.tac)
        ?? mappedPathValue(config, ['TAC'])
        ?? firstSheetValue(config, 'PLMN', '*TAC'),
    };
    return [config.serialNumber, insight];
  }));
}

export interface ParamConfigImportPreviewItem {
  serialNumber: string;
  action: 'add' | 'update' | 'duplicate';
}

export function buildParamConfigImportPreview(
  existing: readonly { serialNumber: string }[],
  incoming: readonly { serialNumber: string }[],
): ParamConfigImportPreviewItem[] {
  const existingSerials = new Set(existing.map((item) => item.serialNumber.trim()));
  const seen = new Set<string>();
  return incoming.map((item) => {
    const serialNumber = item.serialNumber.trim();
    const action = seen.has(serialNumber)
      ? 'duplicate'
      : existingSerials.has(serialNumber) ? 'update' : 'add';
    seen.add(serialNumber);
    return { serialNumber, action };
  });
}
