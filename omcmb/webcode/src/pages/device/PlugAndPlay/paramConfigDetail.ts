import {
  getParamConfigTemplateDefaults,
  getParamConfigTemplateSheets,
} from './paramConfigTemplate';

export type ImportedSheetParameters = Record<string, Record<string, unknown>[]>;

interface ParamConfigDetailSource {
  deviceType?: string;
  serialNumber?: unknown;
  sheetParameters?: ImportedSheetParameters;
  ipsecEnable?: unknown;
  IPSEC_ENABLE?: unknown;
  cellName?: unknown;
  bandsSupport?: unknown;
  frequency?: unknown;
  bandWidth?: unknown;
  subframeAssignment?: unknown;
  plmnConfigList?: Array<{ plmnId?: unknown; primary?: unknown }>;
}

interface SheetFieldMapping {
  field: string;
  sheet: string;
  header: string;
}

const ENB_SHEET_FIELD_MAPPINGS: SheetFieldMapping[] = [
  { field: 'cellName', sheet: 'CELL', header: 'CELL_NAME' },
  { field: 'cellIdentity', sheet: 'CELL', header: '*ECI' },
  { field: 'bandsSupport', sheet: 'CELL', header: '*BAND' },
  { field: 'frequency', sheet: 'CELL', header: '*EARFCN_DL' },
  { field: 'bandWidth', sheet: 'CELL', header: '*BANDWIDTH_DL' },
  { field: 'phycellid', sheet: 'CELL', header: '*PCI' },
  { field: 'specialSubframePatterns', sheet: 'CELL', header: 'SPECIAL_SUBFRAME_PATTERNS' },
  { field: 'subframeAssignment', sheet: 'CELL', header: 'SUBFRAME_ASSIGNMENT' },
  { field: 'rootSequenceIndex', sheet: 'CELL', header: '*ROOT_SEQUENCE_INDEX' },
  { field: 'tac', sheet: 'CELL', header: '*TAC' },
  { field: 'plmnId', sheet: 'NETWORK_ENABLE', header: '*PLMN' },
  { field: 'ipsecEnable', sheet: 'NETWORK_ENABLE', header: 'IPSEC_ENABLE' },
  { field: 'halobEnable', sheet: 'NETWORK_ENABLE', header: 'HALOB_ENABLE' },
  { field: 'totalTxPower', sheet: 'CELL', header: 'MaxTxPower' },
  { field: 'serviceIp', sheet: 'NETWORK', header: 'WAN IP' },
  { field: 'mgmtIp', sheet: 'NETWORK', header: 'OMC IP' },
  { field: 'ntpSync', sheet: 'NETWORK', header: 'NTP Enable' },
];

const GNB_SHEET_FIELD_MAPPINGS: SheetFieldMapping[] = [
  { field: 'gnbName', sheet: 'CELL', header: 'gNB Name' },
  { field: 'gnbId', sheet: 'CELL', header: '*gNB ID' },
  { field: 'gnbIdLength', sheet: 'CELL', header: '*gNB Lenth' },
  { field: 'pci', sheet: 'CELL', header: '*PCI' },
  { field: 'ssbFrequency', sheet: 'CELL', header: 'SSB Frequency' },
  { field: 'freqBandIndicator', sheet: 'CELL', header: 'Freq BandIndicator' },
  { field: 'nrarfcnndl', sheet: 'CELL', header: 'NRARFCNDL' },
  { field: 'nrarfcnul', sheet: 'CELL', header: 'NRARFCNUL' },
  { field: 'dlbandwidth', sheet: 'CELL', header: 'DLBandwidth' },
  { field: 'duplexMode', sheet: 'CELL', header: 'Duplex Mode' },
  { field: 'nci', sheet: 'PLMN', header: '*NCI' },
  { field: 'tac', sheet: 'PLMN', header: '*TAC' },
  { field: 'ranac', sheet: 'PLMN', header: '*RANAC' },
  { field: 'plmnId', sheet: 'PLMN', header: '*PLMN ID' },
  { field: 'ntpSync', sheet: 'DEVICE', header: 'NTP Enable' },
  { field: 'serviceIp', sheet: 'INTERFACE', header: 'IP Address' },
  { field: 'serviceMask', sheet: 'INTERFACE', header: 'Subnet Mask' },
  { field: 'serviceGateway', sheet: 'INTERFACE', header: 'Gateway' },
  { field: 'serviceVlan', sheet: 'INTERFACE', header: 'Vlan ID' },
  { field: 'omIp', sheet: 'INTERFACE', header: 'OMC IP' },
  { field: 'totalTxPower', sheet: 'CELL', header: 'PowerModify' },
  { field: 'offsetToPointA', sheet: 'CELL', header: 'OffsetToPointA' },
  { field: 'kssb', sheet: 'CELL', header: 'SsbSubcarrierOffset' },
];

const ENB_IPSEC_FIELD_MAPPINGS = [
  ['TUNNEL_INDEX', '*TUNNEL_INDEX'],
  ['TUNNEL_ENABLE', '*TUNNEL_ENABLE'],
  ['TUNNEL_GATEWAY', '*TUNNEL_GATEWAY'],
  ...[
    'LEFT_IDENTIFIER', 'RIGHT_IDENTIFIER', 'LEFT_AUTH', 'RIGHT_AUTH',
    'LEFT_CERT', 'SECRET_KEY', 'LEFTSOURCEIP', 'IKE_ENCRYPTION',
    'ESP_ENCRYPTION', 'IKE_DH_GROUP', 'ESP_DH_GROUP', 'IKE_AUTHENTICATION',
    'ESP_AUTHENTICATION', 'KEYLIFE', 'IKELIFETIME', 'REKEYMARGIN',
    'DPDACTION', 'DPDDELAY', 'RIGHT_SECRET_KEY', 'LEFT_SUBNET', 'RIGHT_SUBNET',
    'FRAGMENTATION', 'LEFT_INTERFACE', 'FORCEENCAPS',
  ].map((field) => [field, field]),
] as const;

function firstRow(
  sheets: ImportedSheetParameters | undefined,
  sheetName: string,
): Record<string, unknown> {
  return sheets?.[sheetName]?.[0] ?? {};
}

function value(row: Record<string, unknown>, ...keys: string[]): unknown {
  for (const key of keys) {
    const candidate = row[key];
    if (candidate !== undefined && candidate !== null && String(candidate).trim() !== '') {
      return candidate;
    }
  }
  return undefined;
}

function stringValue(input: unknown): string | undefined {
  if (input === undefined || input === null || String(input).trim() === '') return undefined;
  return String(input).trim();
}

function booleanValue(input: unknown): boolean | undefined {
  if (typeof input === 'boolean') return input;
  const normalized = String(input ?? '').trim().toLowerCase();
  if (['1', 'true', 'yes', 'on'].includes(normalized)) return true;
  if (['0', 'false', 'no', 'off'].includes(normalized)) return false;
  return undefined;
}

function compact(values: Record<string, unknown>): Record<string, unknown> {
  return Object.fromEntries(
    Object.entries(values).filter(([, item]) => item !== undefined),
  );
}

function cloneSheets(sheets: ImportedSheetParameters): ImportedSheetParameters {
  return Object.fromEntries(Object.entries(sheets).map(([sheetName, rows]) => [
    sheetName,
    rows.map((row) => ({ ...row })),
  ]));
}

function serialHeader(headers: readonly string[]): string | undefined {
  return headers.find((header) => (
    header.replace(/^\*/, '').replace(/[\s_]+/g, '').toLowerCase() === 'serialnumber'
  ));
}

export function withTemplateSheetParameters<T extends ParamConfigDetailSource>(
  config: T,
): T {
  const templateSheets = getParamConfigTemplateSheets(
    config.deviceType as 'eNB' | 'gNB' | 'GSM' | undefined,
  );
  if (!templateSheets) return config;

  const existing = config.sheetParameters ?? {};
  const defaults = getParamConfigTemplateDefaults(
    config.deviceType as 'eNB' | 'gNB' | 'GSM' | undefined,
  );
  const sheetParameters: ImportedSheetParameters = {};
  for (const [sheetName, headers] of Object.entries(templateSheets)) {
    const rows = existing[sheetName]?.length ? existing[sheetName] : [{}];
    const snHeader = serialHeader(headers);
    sheetParameters[sheetName] = rows.map((row) => {
      const hydrated = {
        ...Object.fromEntries(headers.map((header) => [header, ''])),
        ...(defaults[sheetName] ?? {}),
        ...row,
      };
      if (snHeader && String(hydrated[snHeader] ?? '').trim() === '') {
        hydrated[snHeader] = config.serialNumber ?? '';
      }
      return hydrated;
    });
  }
  for (const [sheetName, rows] of Object.entries(existing)) {
    if (!sheetParameters[sheetName]) sheetParameters[sheetName] = rows;
  }
  return { ...config, sheetParameters };
}

function valuesEqual(left: unknown, right: unknown): boolean {
  return JSON.stringify(left) === JSON.stringify(right);
}

function setFirstSheetValue(
  sheets: ImportedSheetParameters,
  sheetName: string,
  header: string,
  valueToSet: unknown,
): void {
  const row = sheets[sheetName]?.[0];
  if (!row) return;
  row[header] = valueToSet ?? '';
}

function replaceSheetRows(
  sheets: ImportedSheetParameters,
  sheetName: string,
  rows: Array<Record<string, unknown>> | undefined,
  serialNumber: unknown,
): void {
  if (!rows) return;
  const templateHeaders = getParamConfigTemplateSheets(
    sheetName === 'NETWORK_IPSEC' ? 'eNB' : 'gNB',
  )?.[sheetName] ?? [];
  const snHeader = serialHeader(templateHeaders);
  sheets[sheetName] = rows.map((row, index) => ({
    ...Object.fromEntries(templateHeaders.map((header) => [header, ''])),
    ...(snHeader ? { [snHeader]: sheets[sheetName]?.[index]?.[snHeader] ?? serialNumber ?? '' } : {}),
    ...row,
  }));
}

function ensureSheetRow(
  sheets: ImportedSheetParameters,
  deviceType: 'eNB' | 'gNB',
  sheetName: string,
  index: number,
  serialNumber: unknown,
): Record<string, unknown> {
  const headers = getParamConfigTemplateSheets(deviceType)?.[sheetName] ?? [];
  sheets[sheetName] ??= [];
  while (sheets[sheetName].length <= index) {
    const snHeader = serialHeader(headers);
    sheets[sheetName].push({
      ...Object.fromEntries(headers.map((header) => [header, ''])),
      ...(snHeader ? { [snHeader]: serialNumber ?? '' } : {}),
    });
  }
  return sheets[sheetName][index];
}

function ipsecHeader(field: string, deviceType: 'eNB' | 'gNB'): string {
  if (deviceType === 'gNB') return field;
  return ENB_IPSEC_FIELD_MAPPINGS.find(([formField]) => formField === field)?.[1] ?? field;
}

// Imported worksheet values are the canonical source used by the backend
// compiler. Keep the structured editor and the full worksheet editor in sync
// so either editing route produces the same XML input.
export function mergeParamConfigFormValues<T extends ParamConfigDetailSource>(
  current: T,
  submitted: Record<string, unknown>,
): T & Record<string, unknown> {
  if (!current.sheetParameters || !submitted.sheetParameters) {
    return { ...current, ...submitted } as T & Record<string, unknown>;
  }

  const originalForm = toParamConfigFormValues(current);
  const sheets = cloneSheets(submitted.sheetParameters as ImportedSheetParameters);
  const mappings = current.deviceType === 'gNB'
    ? GNB_SHEET_FIELD_MAPPINGS
    : current.deviceType === 'eNB'
      ? ENB_SHEET_FIELD_MAPPINGS
      : [];

  for (const mapping of mappings) {
    if (!valuesEqual(submitted[mapping.field], originalForm[mapping.field])) {
      setFirstSheetValue(sheets, mapping.sheet, mapping.header, submitted[mapping.field]);
    }
  }

  if (current.deviceType === 'eNB'
    && !valuesEqual(submitted.mmeList, originalForm.mmeList)) {
    const mmeIp = (submitted.mmeList as Array<Record<string, unknown>> | undefined)?.[0]?.mmeIp;
    setFirstSheetValue(sheets, 'NETWORK_ENABLE', 'MME_IP', mmeIp);
  }

  if (current.deviceType === 'eNB'
    && !valuesEqual(submitted.plmnConfigList, originalForm.plmnConfigList)) {
    const firstPlmn = (submitted.plmnConfigList as Array<Record<string, unknown>> | undefined)?.[0]?.plmnId;
    setFirstSheetValue(sheets, 'NETWORK_ENABLE', '*PLMN', firstPlmn);
  }

  if (current.deviceType === 'gNB'
    && !valuesEqual(submitted.ipsecList, originalForm.ipsecList)) {
    const list = submitted.ipsecList as Array<Record<string, unknown>> | undefined;
    replaceSheetRows(sheets, 'IPSEC', list?.map((item) => (
      Object.fromEntries(ENB_IPSEC_FIELD_MAPPINGS
        .filter(([field]) => field !== 'TUNNEL_INDEX')
        .map(([field]) => [ipsecHeader(field, 'gNB'), item[field] ?? '']))
    )), current.serialNumber);
  }

  if (current.deviceType === 'gNB'
    && !valuesEqual(submitted.plmnConfigList, originalForm.plmnConfigList)) {
    const list = submitted.plmnConfigList as Array<Record<string, unknown>> | undefined;
    list?.forEach((item, index) => {
      const row = ensureSheetRow(sheets, 'gNB', 'PLMN', index, current.serialNumber);
      row['*PLMN ID'] = item.plmnId ?? '';
      row['*PRIMARY'] = item.primary ?? '';
    });
  }

  if (current.deviceType === 'gNB'
    && !valuesEqual(submitted.amfList, originalForm.amfList)) {
    const amfIp = (submitted.amfList as Array<Record<string, unknown>> | undefined)?.[0]?.amfIp;
    setFirstSheetValue(sheets, 'PLMN', 'AMF IP:DEFAULT', amfIp);
  }


  if (current.deviceType === 'gNB'
    && !valuesEqual(submitted.sliceConfigList, originalForm.sliceConfigList)) {
    const list = submitted.sliceConfigList as Array<Record<string, unknown>> | undefined;
    list?.forEach((item, index) => {
      const row = ensureSheetRow(sheets, 'gNB', 'PLMN', index, current.serialNumber);
      row.SD = item.sd ?? '';
      row['SD Value'] = item.sdValue ?? '';
    });
  }

  if (current.deviceType === 'eNB'
    && !valuesEqual(submitted.ipsecList, originalForm.ipsecList)) {
    const list = submitted.ipsecList as Array<Record<string, unknown>> | undefined;
    replaceSheetRows(sheets, 'NETWORK_IPSEC', list?.map((item) => (
      Object.fromEntries(ENB_IPSEC_FIELD_MAPPINGS.map(([field, header]) => (
        [header, item[field] ?? '']
      )))
    )), current.serialNumber);
  }

  const refreshed = toParamConfigFormValues({
    deviceType: current.deviceType,
    sheetParameters: sheets,
    plmnConfigList: current.deviceType === 'eNB'
      ? submitted.plmnConfigList as Array<{ plmnId?: unknown; primary?: unknown }> | undefined
      : undefined,
  });
  return {
    ...current,
    ...submitted,
    ...refreshed,
    ...(current.deviceType === 'gNB'
      ? { IPSEC_ENABLE: submitted.IPSEC_ENABLE }
      : {}),
    sheetParameters: sheets,
  } as T & Record<string, unknown>;
}

export function toParamConfigFormValues(
  config: ParamConfigDetailSource,
): Record<string, unknown> {
  const sheets = config.sheetParameters;
  const cell = firstRow(sheets, 'CELL');
  const plmnRows = sheets?.PLMN ?? [];
  const firstPlmn = plmnRows[0] ?? {};

  if (config.deviceType === 'gNB') {
    const dlBandwidth = stringValue(value(cell, 'DLBandwidth'));
    const ipsecRows = sheets?.IPSEC ?? [];
    return {
      ...config,
      ...compact({
        IPSEC_ENABLE: stringValue(config.IPSEC_ENABLE ?? config.ipsecEnable) || '0',
        gnbName: value(cell, 'gNB Name') ?? config.cellName,
        gnbId: value(cell, '*gNB ID', 'gNB ID'),
        gnbIdLength: value(cell, '*gNB Lenth', '*gNB Length', 'gNB ID Length'),
        pci: value(cell, '*PCI', 'PCI'),
        ssbFrequency: value(cell, 'SSB Frequency'),
        freqBandIndicator: value(cell, 'Freq BandIndicator'),
        nrarfcnndl: value(cell, 'NRARFCNDL'),
        nrarfcnul: value(cell, 'NRARFCNUL'),
        dlbandwidth: dlBandwidth?.replace(/\s*MHz$/i, ''),
        duplexMode: value(cell, 'Duplex Mode'),
        nci: value(firstPlmn, '*NCI', 'NCI'),
        tac: value(firstPlmn, '*TAC', 'TAC'),
        ranac: value(firstPlmn, '*RANAC', 'RANAC'),
        plmnId: value(firstPlmn, '*PLMN ID', 'PLMN ID'),
        ntpSync: stringValue(value(firstRow(sheets, 'DEVICE'), 'NTP Enable')),
        serviceIp: value(firstRow(sheets, 'INTERFACE'), 'IP Address'),
        serviceMask: value(firstRow(sheets, 'INTERFACE'), 'Subnet Mask'),
        serviceGateway: value(firstRow(sheets, 'INTERFACE'), 'Gateway'),
        serviceVlan: value(firstRow(sheets, 'INTERFACE'), 'Vlan ID'),
        omIp: value(firstRow(sheets, 'INTERFACE'), 'OMC IP'),
        totalTxPower: value(cell, 'PowerModify'),
        offsetToPointA: value(cell, 'OffsetToPointA'),
        kssb: value(cell, 'SsbSubcarrierOffset'),
        amfList: value(firstPlmn, 'AMF IP:DEFAULT') !== undefined
          ? [{ amfIp: value(firstPlmn, 'AMF IP:DEFAULT'), amfPort: '' }]
          : undefined,
        plmnConfigList: plmnRows.length > 0
          ? plmnRows.map((row) => compact({
            plmnId: value(row, '*PLMN ID', 'PLMN ID'),
            primary: stringValue(value(row, '*PRIMARY', 'PRIMARY')),
          }))
          : undefined,
        sliceConfigList: plmnRows.length > 0
          ? plmnRows.map((row) => compact({
            sd: stringValue(value(row, 'SD')),
            sdValue: value(row, 'SD Value'),
          }))
          : undefined,
        ipsecList: ipsecRows.length > 0
          ? ipsecRows.map((row, index) => compact({
            key: String(index + 1),
            ...Object.fromEntries(ENB_IPSEC_FIELD_MAPPINGS
              .filter(([field]) => field !== 'TUNNEL_INDEX')
              .map(([field]) => [
                field,
                value(row, ipsecHeader(field, 'gNB')),
              ])),
          }))
          : undefined,
      }),
    };
  }

  if (config.deviceType === 'eNB') {
    const network = firstRow(sheets, 'NETWORK_ENABLE');
    const mmeIp = value(network, 'MME_IP', 'MME IP');
    const ipsecRows = sheets?.NETWORK_IPSEC ?? [];
    return {
      ...config,
      ...compact({
        cellName: value(cell, 'CELL_NAME', 'Cell Name') ?? config.cellName,
        cellIdentity: value(cell, '*ECI', 'ECI'),
        bandsSupport: value(cell, '*BAND', 'BAND') ?? config.bandsSupport,
        frequency: value(cell, '*EARFCN_DL', 'EARFCN_DL') ?? config.frequency,
        bandWidth: value(cell, '*BANDWIDTH_DL', 'BANDWIDTH_DL') ?? config.bandWidth,
        phycellid: value(cell, '*PCI', 'PCI'),
        specialSubframePatterns: stringValue(
          value(cell, 'SPECIAL_SUBFRAME_PATTERNS'),
        ),
        subframeAssignment: stringValue(
          value(cell, 'SUBFRAME_ASSIGNMENT'),
        ) ?? stringValue(config.subframeAssignment),
        rootSequenceIndex: value(cell, '*ROOT_SEQUENCE_INDEX', 'ROOT_SEQUENCE_INDEX'),
        tac: value(cell, '*TAC', 'TAC'),
        plmnId: value(network, '*PLMN', 'PLMN'),
        plmnConfigList: config.plmnConfigList?.length
          ? config.plmnConfigList
          : (value(network, '*PLMN', 'PLMN') !== undefined
            ? [{ plmnId: value(network, '*PLMN', 'PLMN') }]
            : undefined),
        ntpSync: stringValue(value(firstRow(sheets, 'NETWORK'), 'NTP Enable')),
        mmeList: mmeIp !== undefined ? [{ mmeIp }] : undefined,
        ipsecEnable: stringValue(value(network, 'IPSEC_ENABLE')) || '0',
        halobEnable: stringValue(value(network, 'HALOB_ENABLE')),
        totalTxPower: value(cell, 'MaxTxPower'),
        serviceIp: value(firstRow(sheets, 'NETWORK'), 'WAN IP'),
        mgmtIp: value(firstRow(sheets, 'NETWORK'), 'OMC IP'),
        ipsecList: ipsecRows.length > 0
          ? ipsecRows.map((row, index) => compact({
            key: String(index + 1),
            ...Object.fromEntries(ENB_IPSEC_FIELD_MAPPINGS.map(([field, header]) => [
              field,
              field === 'TUNNEL_ENABLE'
                ? booleanValue(value(row, header))
                : value(row, header),
            ])),
          }))
          : undefined,
      }),
    };
  }

  return { ...config };
}
