import {
  getParamConfigTemplateDefaults,
  getParamConfigTemplateSheets,
} from './paramConfigTemplate';
import {
  ENB_SHEET_FIELD_MAPPINGS,
  GNB_SHEET_FIELD_MAPPINGS,
} from './paramConfigFieldMappings';
import { GNB_NETWORK_CONFIG_FIELDS } from './gnbNetworkConfigFields';
import { sanitizeRetiredParamConfigFields } from './retiredParamConfigFields';

export type ImportedSheetParameters = Record<string, Record<string, unknown>[]>;

export interface ParamConfigDetailSource {
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
  serviceIp?: unknown;
  serviceMask?: unknown;
  serviceGateway?: unknown;
  serviceVlan?: unknown;
  tfcsManagerPrimsrc?: unknown;
  addressType?: unknown;
  bearType?: unknown;
  prefixLength?: unknown;
  vlanName?: unknown;
  workbookMappings?: Array<{ sheet: string; header: string; trPath: string }>;
  customParams?: Array<{ name?: unknown; value?: unknown; trPath?: unknown }>;
}

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

function canonicalHeader(input: string): string {
  return input.replace(/^\*/, '').replace(/[\s_]+/g, '').toUpperCase();
}

interface FieldMappingTarget {
  leaves: readonly string[];
  headers: readonly string[];
}

interface DirectSheetMappingTarget extends FieldMappingTarget {
  header: string;
  sheetByDeviceType: Partial<Record<'eNB' | 'gNB' | 'GSM', string>>;
}

const FIELD_MAPPING_TARGETS: Record<string, FieldMappingTarget> = {
  cellName: { leaves: ['CELLNAME'], headers: ['CELL_NAME', 'Cell Name'] },
  cellIdentity: { leaves: ['ECI'], headers: ['*ECI', 'ECI'] },
  gnbName: { leaves: ['GNBNAME'], headers: ['gNB Name'] },
  gnbId: { leaves: ['GNBID'], headers: ['*gNB ID', 'gNB ID'] },
  gnbIdLength: { leaves: ['GNBIDLENGTH', 'GNBIDLENTH'], headers: ['*gNB Lenth', '*gNB Length', 'gNB ID Length'] },
  pci: { leaves: ['PCI', 'PHYCELLID', 'PHYSICALCELLID'], headers: ['*PCI', 'PCI'] },
  phycellid: { leaves: ['PCI', 'PHYCELLID', 'PHYSICALCELLID'], headers: ['*PCI', 'PCI'] },
  bandsSupport: { leaves: ['BAND', 'BANDINDICATOR', 'FREQBANDINDICATOR', 'FREQBANDINDICATORNR'], headers: ['*BAND', 'BAND', 'Band', 'Freq BandIndicator'] },
  freqBandIndicator: { leaves: ['BAND', 'BANDINDICATOR', 'FREQBANDINDICATOR', 'FREQBANDINDICATORNR'], headers: ['Freq BandIndicator', 'FreqBandIndicatorNR', 'Band'] },
  RFEnable: { leaves: ['RFENABLE', 'ENABLE'], headers: ['RF Enable', 'RFEnable'] },
  frequency: { leaves: ['EARFCNDL', 'DLEARFCN'], headers: ['*EARFCN_DL', 'EARFCN_DL'] },
  bandWidth: { leaves: ['BANDWIDTHDL', 'DLBANDWIDTH'], headers: ['*BANDWIDTH_DL', 'BANDWIDTH_DL', 'DLBandWidth'] },
  ssbFrequency: { leaves: ['SSBFREQUENCY'], headers: ['SSB Frequency'] },
  nrarfcnndl: { leaves: ['NRARFCNDL'], headers: ['NRARFCNDL'] },
  nrarfcnul: { leaves: ['NRARFCNUL'], headers: ['NRARFCNUL'] },
  dlbandwidth: { leaves: ['DLBANDWIDTH', 'DLCARRIERBANDWIDTH'], headers: ['DL Carrier Bandwidth', 'DLBandwidth'] },
  ulbandwidth: { leaves: ['ULBANDWIDTH', 'ULCARRIERBANDWIDTH'], headers: ['UL Carrier Bandwidth', 'ULBandwidth'] },
  ntpSync: { leaves: ['NTPENABLE', 'NTPMODE'], headers: ['NTP Enable', 'NTP Mode'] },
  specialSubframePatterns: { leaves: ['SPECIALSUBFRAMEPATTERNS'], headers: ['SPECIAL_SUBFRAME_PATTERNS'] },
  subframeAssignment: { leaves: ['SUBFRAMEASSIGNMENT'], headers: ['SUBFRAME_ASSIGNMENT'] },
  rootSequenceIndex: { leaves: ['ROOTSEQUENCEINDEX'], headers: ['*ROOT_SEQUENCE_INDEX', 'ROOT_SEQUENCE_INDEX'] },
  tac: { leaves: ['TAC'], headers: ['*TAC', 'TAC'] },
  ranac: { leaves: ['RANAC'], headers: ['*RANAC', 'RANAC'] },
  nci: { leaves: ['NCI'], headers: ['*NCI', 'NCI'] },
  plmnId: { leaves: ['PLMN', 'PLMNID'], headers: ['*PLMN', 'PLMN', '*PLMN ID', 'PLMN ID'] },
  SyncSource: { leaves: ['SYNCSOURCE'], headers: ['SyncSource', 'Sync Source'] },
  ForcedSync: { leaves: ['FORCEDSYNC'], headers: ['ForcedSync', 'Forced Sync'] },
  PTPProfile: { leaves: ['PTPPROFILE', 'PROFILE'], headers: ['PTPProfile', 'Profile'] },
  PTPDomain: { leaves: ['PTPDOMAIN', 'DOMAIN'], headers: ['PTPDomain', 'Domain'] },
  PTPTransmode: { leaves: ['PTPTRANSMODE', 'TRANSMISSIONMODE'], headers: ['PTPTransmode', 'Transmission Mode'] },
  PTPInterface: { leaves: ['PTPINTERFACE', 'INTERFACE'], headers: ['PTPInterface', 'Interface'] },
  PTPUnicastMode: { leaves: ['PTPUNICASTMODE', 'UNICASTMODE'], headers: ['PTPUnicastMode', 'Unicast Mode'] },
  PTPSyncInterval: { leaves: ['PTPSYNCINTERVAL', 'SYNCINTERVAL'], headers: ['PTPSyncInterval', 'Sync Interval'] },
  PTPDelayInterval: { leaves: ['PTPDELAYINTERVAL', 'DELAYINTERVAL'], headers: ['PTPDelayInterval', 'Delay Interval'] },
  ipsecEnable: { leaves: ['IPSECENABLE'], headers: ['IPSEC_ENABLE', 'IPSec Enable'] },
  IPSEC_ENABLE: { leaves: ['IPSECENABLE'], headers: ['IPSEC_ENABLE', 'IPSec Enable'] },
  halobEnable: { leaves: ['HALOBENABLE'], headers: ['HALOB_ENABLE'] },
  totalTxPower: { leaves: ['MAXTXPOWER', 'POWERMODIFY', 'POWERLEVEL'], headers: ['MaxTxPower', 'PowerModify', 'Power Level'] },
  serviceIp: { leaves: ['WANIP', 'IPADDRESS'], headers: ['WAN IP', 'IP Address'] },
  offsetToPointA: { leaves: ['OFFSETTOPOINTA'], headers: ['OffsetToPointA', 'Offset To Point A'] },
  kssb: { leaves: ['SSBSUBCARRIEROFFSET'], headers: ['SsbSubcarrierOffset', 'SSB Subcarrier Offset'] },
};

const DIRECT_SHEET_MAPPING_TARGETS: readonly DirectSheetMappingTarget[] = [
  { leaves: ['NTPSERVER1'], headers: ['NTP Server1', 'NTP Server 1', 'NTP服务器1', 'NTP 服务器 1'], header: 'NTP Server1', sheetByDeviceType: { gNB: 'DEVICE', eNB: 'NETWORK' } },
  { leaves: ['NTPSERVER2'], headers: ['NTP Server2', 'NTP Server 2', 'NTP服务器2', 'NTP 服务器 2'], header: 'NTP Server2', sheetByDeviceType: { gNB: 'DEVICE' } },
  { leaves: ['NTPSERVER3'], headers: ['NTP Server3', 'NTP Server 3', 'NTP服务器3', 'NTP 服务器 3'], header: 'NTP Server3', sheetByDeviceType: { gNB: 'DEVICE' } },
  { leaves: ['NTPSERVER4'], headers: ['NTP Server4', 'NTP Server 4', 'NTP服务器4', 'NTP 服务器 4'], header: 'NTP Server4', sheetByDeviceType: { gNB: 'DEVICE' } },
  { leaves: ['NTPSERVER5'], headers: ['NTP Server5', 'NTP Server 5', 'NTP服务器5', 'NTP 服务器 5'], header: 'NTP Server5', sheetByDeviceType: { gNB: 'DEVICE' } },
  { leaves: ['LOCALTIMEZONENAME', 'LOCALTIMEZONE'], headers: ['Local Time Zone', '本地时区'], header: 'Local Time Zone', sheetByDeviceType: { gNB: 'DEVICE', eNB: 'NETWORK' } },
  { leaves: ['MANAGEMENTSERVERURL', 'URL'], headers: ['URL', 'Management Server URL', '网管服务器 URL'], header: 'URL', sheetByDeviceType: { gNB: 'DEVICE' } },
  { leaves: ['PERIODICINFORMENABLE'], headers: ['Periodic Inform Enable', '周期上报开关'], header: 'Periodic Inform Enable', sheetByDeviceType: { gNB: 'DEVICE' } },
  { leaves: ['PERIODICINFORMTIME'], headers: ['Periodic Inform Time', '周期上报时间'], header: 'Periodic Inform Time', sheetByDeviceType: { gNB: 'DEVICE' } },
  { leaves: ['PERIODICINFORMINTERVAL'], headers: ['Periodic Inform Interval', '周期上报间隔'], header: 'Periodic Inform Interval', sheetByDeviceType: { gNB: 'DEVICE' } },
  { leaves: ['PPSTIMEMODE'], headers: ['PpsTimeMode', 'PPS Time Mode'], header: 'PpsTimeMode', sheetByDeviceType: { gNB: 'DEVICE' } },
  { leaves: ['BAND', 'BANDINDICATOR', 'FREQBANDINDICATOR', 'FREQBANDINDICATORNR'], headers: ['Freq BandIndicator', 'FreqBandIndicatorNR', 'Band'], header: 'Freq BandIndicator', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: ['SSBFREQUENCY'], headers: ['SSB Frequency', 'SSBFrequency'], header: 'SSB Frequency', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: ['NRARFCNDL'], headers: ['NRARFCNDL'], header: 'NRARFCNDL', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: ['NRARFCNUL'], headers: ['NRARFCNUL'], header: 'NRARFCNUL', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: ['DLBANDWIDTH', 'DLCARRIERBANDWIDTH'], headers: ['DLBandwidth', 'DL Carrier Bandwidth'], header: 'DLBandwidth', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: ['ULBANDWIDTH', 'ULCARRIERBANDWIDTH'], headers: ['ULBandwidth', 'UL Carrier Bandwidth'], header: 'ULBandwidth', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: ['DLSUBCARRIERSPACING'], headers: ['SubcarrierSpacing(DL)', 'DL SubCarrier Spacing', 'DL Subcarrier Spacing', 'DLSubCarrierSpacing'], header: 'SubcarrierSpacing(DL)', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: ['ULSUBCARRIERSPACING'], headers: ['SubcarrierSpacing(UL)', 'UL SubCarrier Spacing', 'UL Subcarrier Spacing', 'ULSubCarrierSpacing'], header: 'SubcarrierSpacing(UL)', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: ['NUMOFTXANTENNA', 'DLANTNUM'], headers: ['DLAntNum', 'Num Of Tx Antenna', 'NumOfTxAntenna', 'Tx Antenna Count'], header: 'DLAntNum', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: ['NUMOFRXANTENNA', 'ULANTNUM'], headers: ['ULAntNum', 'Num Of Rx Antenna', 'NumOfRxAntenna', 'Rx Antenna Count'], header: 'ULAntNum', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: [], headers: ['DL ULTransmissionPeriodicity1', 'Pattern1 Periodicity', 'Pattern1 DL ULTransmissionPeriodicity'], header: 'DL ULTransmissionPeriodicity1', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: [], headers: ['Nrof DownlinkSlots1', 'Pattern1 DL Slots', 'Pattern1 Downlink Slots'], header: 'Nrof DownlinkSlots1', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: [], headers: ['Nrof DownlinkSymbols1', 'Pattern1 DL Symbols', 'Pattern1 Downlink Symbols'], header: 'Nrof DownlinkSymbols1', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: [], headers: ['Nrof  UplinkSlots1', 'Nrof UplinkSlots1', 'Pattern1 UL Slots', 'Pattern1 Uplink Slots'], header: 'Nrof  UplinkSlots1', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: [], headers: ['Nrof  UplinkSymbols1', 'Nrof UplinkSymbols1', 'Pattern1 UL Symbols', 'Pattern1 Uplink Symbols'], header: 'Nrof  UplinkSymbols1', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: [], headers: ['DL ULTransmissionPeriodicity2', 'Pattern2 Periodicity', 'Pattern2 DL ULTransmissionPeriodicity'], header: 'DL ULTransmissionPeriodicity2', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: [], headers: ['Nrof  DownlinkSlots2', 'Nrof DownlinkSlots2', 'Pattern2 DL Slots', 'Pattern2 Downlink Slots'], header: 'Nrof  DownlinkSlots2', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: [], headers: ['Nrof  DownlinkSymbols2', 'Nrof DownlinkSymbols2', 'Pattern2 DL Symbols', 'Pattern2 Downlink Symbols'], header: 'Nrof  DownlinkSymbols2', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: [], headers: ['Nrof  UplinkSlots2', 'Nrof UplinkSlots2', 'Pattern2 UL Slots', 'Pattern2 Uplink Slots'], header: 'Nrof  UplinkSlots2', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: [], headers: ['Nrof  UplinkSymbols2', 'Nrof UplinkSymbols2', 'Pattern2 UL Symbols', 'Pattern2 Uplink Symbols'], header: 'Nrof  UplinkSymbols2', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: ['PRACHROOTSEQUENCEINDEX'], headers: ['Prach RootSequenceIndex', 'Prach Root Sequence Index'], header: 'Prach RootSequenceIndex', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: ['PRACHROOTSEQUENCEVALUE'], headers: ['Prach RootSequenceValue', 'Prach Root Sequence Value'], header: 'Prach RootSequenceValue', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: ['RFENABLE', 'ENABLE'], headers: ['RFEnable', 'RF Enable'], header: 'RFEnable', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: ['POWERMODIFY', 'POWERLEVEL'], headers: ['PowerModify', 'Power Level'], header: 'PowerModify', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: ['OFFSETTOPOINTA'], headers: ['OffsetToPointA', 'Offset To Point A'], header: 'OffsetToPointA', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: ['PCI', 'PHYCELLID', 'PHYSICALCELLID'], headers: ['*PCI', 'PCI'], header: '*PCI', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: ['SSBSUBCARRIEROFFSET'], headers: ['SsbSubcarrierOffset', 'SSB Subcarrier Offset'], header: 'SsbSubcarrierOffset', sheetByDeviceType: { gNB: 'CELL' } },
  { leaves: ['NGUBINDINTERFACE'], headers: ['NguBindInterface', 'NGU Local Address'], header: 'NguBindInterface', sheetByDeviceType: { gNB: 'PLMN' } },
  { leaves: ['TUNNELENABLE'], headers: ['TUNNEL_ENABLE', 'Tunnel Enable'], header: 'TUNNEL_ENABLE', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['TUNNELGATEWAY', 'GATEWAY'], headers: ['TUNNEL_GATEWAY', 'Gateway'], header: 'TUNNEL_GATEWAY', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['LEFTAUTH'], headers: ['LEFT_AUTH', 'Left Auth'], header: 'LEFT_AUTH', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['RIGHTAUTH'], headers: ['RIGHT_AUTH', 'Right Auth'], header: 'RIGHT_AUTH', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['LEFTIDENTIFIER'], headers: ['LEFT_IDENTIFIER', 'Left ID'], header: 'LEFT_IDENTIFIER', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['RIGHTIDENTIFIER'], headers: ['RIGHT_IDENTIFIER', 'Right ID'], header: 'RIGHT_IDENTIFIER', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['LEFTSOURCEIP'], headers: ['LEFTSOURCEIP', 'Left Source IP'], header: 'LEFTSOURCEIP', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['LEFTSUBNET'], headers: ['LEFT_SUBNET', 'Left Subnet'], header: 'LEFT_SUBNET', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['RIGHTSUBNET'], headers: ['RIGHT_SUBNET', 'Right Subnet'], header: 'RIGHT_SUBNET', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['FRAGMENTATION'], headers: ['FRAGMENTATION', 'Fragmentation'], header: 'FRAGMENTATION', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['IKEENCRYPTION'], headers: ['IKE_ENCRYPTION', 'IKE Encryption'], header: 'IKE_ENCRYPTION', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['IKEDHGROUP'], headers: ['IKE_DH_GROUP', 'IKE DH Group'], header: 'IKE_DH_GROUP', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['IKEAUTHENTICATION'], headers: ['IKE_AUTHENTICATION', 'IKE Authentication'], header: 'IKE_AUTHENTICATION', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['ESPENCRYPTION'], headers: ['ESP_ENCRYPTION', 'ESP Encryption'], header: 'ESP_ENCRYPTION', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['ESPDHGROUP'], headers: ['ESP_DH_GROUP', 'ESP DH Group'], header: 'ESP_DH_GROUP', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['ESPAUTHENTICATION'], headers: ['ESP_AUTHENTICATION', 'ESP Authentication'], header: 'ESP_AUTHENTICATION', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['KEYLIFE'], headers: ['KEYLIFE', 'Key Life'], header: 'KEYLIFE', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['IKELIFETIME'], headers: ['IKELIFETIME', 'IKE Lifetime'], header: 'IKELIFETIME', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['REKEYMARGIN'], headers: ['REKEYMARGIN', 'Rekey Margin'], header: 'REKEYMARGIN', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['DPDACTION'], headers: ['DPDACTION', 'DPD Action'], header: 'DPDACTION', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['DPDDELAY'], headers: ['DPDDELAY', 'DPD Delay'], header: 'DPDDELAY', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['LEFTINTERFACE'], headers: ['LEFT_INTERFACE', 'Left Interface'], header: 'LEFT_INTERFACE', sheetByDeviceType: { gNB: 'IPSEC' } },
  { leaves: ['FORCEENCAPS'], headers: ['FORCEENCAPS', 'Force Encapsulation'], header: 'FORCEENCAPS', sheetByDeviceType: { gNB: 'IPSEC' } },
];

function mappingMatchesTarget(
  mapping: { header: string; displayName?: string; trPath: string },
  target: FieldMappingTarget,
): boolean {
  const leaf = mapping.trPath.split('.').filter(Boolean).at(-1) ?? '';
  const leaves = new Set(target.leaves.map(canonicalHeader));
  const headers = new Set(target.headers.map(canonicalHeader));
  return leaves.has(canonicalHeader(leaf))
    || headers.has(canonicalHeader(mapping.header))
    || headers.has(canonicalHeader(mapping.displayName ?? ''));
}

function mappedWorkbookValue(
  config: ParamConfigDetailSource,
  field: string,
): unknown {
  const target = FIELD_MAPPING_TARGETS[field];
  if (!target) return undefined;
  for (const mapping of config.workbookMappings ?? []) {
    if (!mappingMatchesTarget(mapping, target)) continue;
    const mapped = config.sheetParameters?.[mapping.sheet]?.[0]?.[mapping.header];
    if (mapped !== undefined && mapped !== null && String(mapped).trim() !== '') return mapped;
  }
  return undefined;
}

function mappedWorkbookValueForTarget(
  config: ParamConfigDetailSource,
  target: FieldMappingTarget,
): unknown {
  for (const mapping of config.workbookMappings ?? []) {
    if (!mappingMatchesTarget(mapping, target)) continue;
    const mapped = config.sheetParameters?.[mapping.sheet]?.[0]?.[mapping.header];
    if (mapped !== undefined && mapped !== null && String(mapped).trim() !== '') return mapped;
  }
  return undefined;
}

function rowValueByHeaders(
  row: Record<string, unknown>,
  headers: readonly string[],
): unknown {
  const expected = new Set(headers.map(canonicalHeader));
  for (const [header, candidate] of Object.entries(row)) {
    if (!expected.has(canonicalHeader(header))) continue;
    if (candidate !== undefined && candidate !== null && String(candidate).trim() !== '') {
      return candidate;
    }
  }
  return undefined;
}

function withMappedDirectSheetValues(
  config: ParamConfigDetailSource,
): ImportedSheetParameters | undefined {
  if (!config.sheetParameters) return undefined;
  const deviceType = config.deviceType as 'eNB' | 'gNB' | 'GSM' | undefined;
  if (!deviceType) return config.sheetParameters;
  const sheets = cloneSheets(config.sheetParameters);
  for (const target of DIRECT_SHEET_MAPPING_TARGETS) {
    const sheetName = target.sheetByDeviceType[deviceType];
    if (!sheetName) continue;
    const sameSheetMappings = (config.workbookMappings ?? []).filter((mapping) => (
      canonicalHeader(mapping.sheet) === canonicalHeader(sheetName)
        && mappingMatchesTarget(mapping, target)
    ));
    for (const row of sheets[sheetName] ?? []) {
      const mappedInRow = sameSheetMappings
        .map((mapping) => value(row, mapping.header))
        .find((candidate) => candidate !== undefined);
      const direct = mappedInRow ?? rowValueByHeaders(row, target.headers);
      if (direct !== undefined && value(row, target.header) === undefined) {
        row[target.header] = direct;
      }
    }
    const mapped = mappedWorkbookValueForTarget(config, target);
    if (mapped === undefined) continue;
    sheets[sheetName] ??= [{}];
    sheets[sheetName][0] ??= {};
    if (value(sheets[sheetName][0], target.header) === undefined) {
      sheets[sheetName][0][target.header] = mapped;
    }
  }
  return sheets;
}

function sheetValue(
  sheets: ImportedSheetParameters | undefined,
  ...headers: string[]
): unknown {
  const expected = new Set(headers.map(canonicalHeader));
  for (const rows of Object.values(sheets ?? {})) {
    for (const row of rows) {
      for (const [header, candidate] of Object.entries(row)) {
        if (!expected.has(canonicalHeader(header))) continue;
        if (candidate !== undefined && candidate !== null && String(candidate).trim() !== '') {
          return candidate;
        }
      }
    }
  }
  return undefined;
}

function stringValue(input: unknown): string | undefined {
  if (input === undefined || input === null || String(input).trim() === '') return undefined;
  return String(input).trim();
}

function binaryStringValue(input: unknown): string | undefined {
  const parsed = booleanValue(input);
  if (parsed !== undefined) return parsed ? '1' : '0';
  return stringValue(input);
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

function hasMeaningfulSheetValues(row: Record<string, unknown>): boolean {
  return Object.entries(row).some(([header, item]) => {
    if (header.replace(/^\*/, '').replace(/[\s_]+/g, '').toLowerCase() === 'serialnumber') {
      return false;
    }
    return item !== undefined && item !== null && String(item).trim() !== '';
  });
}

function hasMeaningfulNetworkValues(row: Record<string, unknown>): boolean {
  return GNB_NETWORK_CONFIG_FIELDS.some(({ header }) => (
    row[header] !== undefined && row[header] !== null && String(row[header]).trim() !== ''
  ));
}

function cloneSheets(sheets: ImportedSheetParameters): ImportedSheetParameters {
  return Object.fromEntries(Object.entries(sheets).map(([sheetName, rows]) => [
    sheetName,
    rows.map((row) => ({ ...row })),
  ]));
}

function mergeExistingSheetCells(
  current: ImportedSheetParameters,
  submitted: ImportedSheetParameters,
  replacedSheets: ReadonlySet<string> = new Set(),
): ImportedSheetParameters {
  const merged = cloneSheets(current);
  for (const [sheetName, submittedRows] of Object.entries(submitted)) {
    if (replacedSheets.has(sheetName)) {
      const currentRows = current[sheetName] ?? [];
      const controlHeaders = ['Cell Index', '*CELL_NUMBER', 'BTS Index'];
      merged[sheetName] = submittedRows.map((submittedRow, rowIndex) => {
        const controlHeader = controlHeaders.find((header) => submittedRow[header] !== undefined);
        const currentRow = controlHeader
          ? currentRows.find((row) => Number(row[controlHeader]) === Number(submittedRow[controlHeader]))
          : currentRows[rowIndex];
        if (!currentRow) return { ...submittedRow };
        const next = { ...currentRow };
        for (const [header, submittedValue] of Object.entries(submittedRow)) {
          if (Object.prototype.hasOwnProperty.call(currentRow, header) || controlHeaders.includes(header)) {
            next[header] = submittedValue;
          }
        }
        return next;
      });
      continue;
    }
    const currentRows = merged[sheetName];
    if (!currentRows) continue;
    submittedRows.forEach((submittedRow, rowIndex) => {
      const currentRow = currentRows[rowIndex];
      if (!currentRow) return;
      for (const [header, submittedValue] of Object.entries(submittedRow)) {
        if (Object.prototype.hasOwnProperty.call(currentRow, header)) {
          currentRow[header] = submittedValue;
        }
      }
    });
  }
  return merged;
}

function serialHeader(headers: readonly string[]): string | undefined {
  return headers.find((header) => (
    header.replace(/^\*/, '').replace(/[\s_]+/g, '').toLowerCase() === 'serialnumber'
  ));
}

function mappedParameterValues(config: ParamConfigDetailSource): Record<string, unknown> {
  const values: Record<string, unknown> = {};
  for (const mapping of config.workbookMappings ?? []) {
    const mappedValue = config.sheetParameters?.[mapping.sheet]?.[0]?.[mapping.header];
    if (mappedValue !== undefined) values[mapping.trPath] = mappedValue;
  }
  if (config.deviceType === 'gNB') {
    for (const [index, row] of (config.sheetParameters?.INTERFACE ?? []).entries()) {
      const interfaceName = value(row, 'Interface Name');
      const path = networkInterfaceNamePath(index);
      if (interfaceName !== undefined && values[path] === undefined) values[path] = interfaceName;
    }
  }
  for (const item of config.customParams ?? []) {
    const path = stringValue(item.trPath);
    if (!path || !isNetworkInterfacePath(path) || values[path] !== undefined) continue;
    const customValue = stringValue(item.value);
    if (customValue !== undefined) values[path] = customValue;
  }
  return values;
}

function isNetworkInterfacePath(path: string): boolean {
  return /^Device\.(?:Ethernet|IP)\.Interface\./.test(path);
}

function networkInterfaceNamePath(rowIndex: number): string {
  return `Device.Ethernet.Interface.${rowIndex + 1}.Name`;
}

function networkInterfaceSheetTarget(path: string): { rowIndex: number; header: string } | undefined {
  const match = /^Device\.(?:Ethernet|IP)\.Interface\.(\d+)\.Name$/.exec(path);
  if (!match) return undefined;
  return { rowIndex: Number(match[1]) - 1, header: 'Interface Name' };
}

function mergeNetworkParametersIntoInterfaceSheet(
  current: ParamConfigDetailSource,
  sheets: ImportedSheetParameters,
  networkParameterValues: Record<string, unknown> | undefined,
): void {
  if (current.deviceType !== 'gNB' || !networkParameterValues) return;
  for (const [rawPath, rawValue] of Object.entries(networkParameterValues)) {
    const path = stringValue(rawPath);
    if (!path) continue;
    const target = networkInterfaceSheetTarget(path);
    if (!target) continue;
    const row = ensureSheetRow(sheets, 'gNB', 'INTERFACE', target.rowIndex, current.serialNumber);
    if (!setExistingRowValue(row, [target.header], rawValue)) {
      row[target.header] = rawValue ?? '';
    }
  }
}

function mergeSubmittedInterfaceNamesIntoInterfaceSheet(
  current: ParamConfigDetailSource,
  sheets: ImportedSheetParameters,
  submittedSheets: ImportedSheetParameters | undefined,
): void {
  if (current.deviceType !== 'gNB') return;
  const submittedRows = submittedSheets?.INTERFACE ?? [];
  submittedRows.forEach((submittedRow, rowIndex) => {
    const interfaceName = value(submittedRow, 'Interface Name');
    if (interfaceName === undefined) return;
    const row = ensureSheetRow(sheets, 'gNB', 'INTERFACE', rowIndex, current.serialNumber);
    if (!setExistingRowValue(row, ['Interface Name'], interfaceName)) {
      row['Interface Name'] = interfaceName ?? '';
    }
  });
}

function mergeUnmappedNetworkParametersIntoCustomParams(
  current: ParamConfigDetailSource,
  submittedCustomParams: unknown,
  networkParameterValues: Record<string, unknown> | undefined,
): ParamConfigDetailSource['customParams'] | undefined {
  const baseCustomParams = Array.isArray(submittedCustomParams)
    ? submittedCustomParams as ParamConfigDetailSource['customParams']
    : current.customParams;
  if (!networkParameterValues) return baseCustomParams;
  const mappedPaths = new Set((current.workbookMappings ?? []).map((mapping) => mapping.trPath));
  const customByPath = new Map<string, { name?: unknown; value?: unknown; trPath?: unknown }>();
  for (const item of baseCustomParams ?? []) {
    const path = stringValue(item.trPath);
    if (!path || mappedPaths.has(path) || networkInterfaceSheetTarget(path)) continue;
    customByPath.set(path, item);
  }
  for (const [rawPath, rawValue] of Object.entries(networkParameterValues)) {
    const path = stringValue(rawPath);
    if (!path || mappedPaths.has(path) || !isNetworkInterfacePath(path) || networkInterfaceSheetTarget(path)) continue;
    const valueToSave = stringValue(rawValue);
    if (valueToSave === undefined) {
      customByPath.delete(path);
    } else {
      customByPath.set(path, { ...(customByPath.get(path) ?? {}), trPath: path, value: valueToSave });
    }
  }
  const customParams = Array.from(customByPath.values());
  return customParams.length > 0 ? customParams : undefined;
}

function mappedPathValue(
  config: ParamConfigDetailSource,
  leafNames: readonly string[],
): unknown {
  const canonical = (input: string): string => input.replace(/[^a-z0-9]/gi, '').toUpperCase();
  const expected = new Set(leafNames.map(canonical));
  for (const mapping of config.workbookMappings ?? []) {
    const leaf = mapping.trPath.split('.').filter(Boolean).at(-1);
    if (!leaf || !expected.has(canonical(leaf))) continue;
    const mapped = config.sheetParameters?.[mapping.sheet]?.[0]?.[mapping.header];
    if (mapped !== undefined && mapped !== null && String(mapped).trim() !== '') return mapped;
  }
  return undefined;
}

export function withTemplateSheetParameters<T extends ParamConfigDetailSource>(
  config: T,
): T {
  const templateSheets = getParamConfigTemplateSheets(
    config.deviceType as 'eNB' | 'gNB' | 'GSM' | undefined,
  );
  if (!templateSheets) return config;

  const existing = sanitizeRetiredParamConfigFields(config.sheetParameters ?? {}, config.deviceType);
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
): boolean {
  return setSheetRowValue(sheets, sheetName, 0, header, valueToSet);
}

function setSheetRowValue(
  sheets: ImportedSheetParameters,
  sheetName: string,
  rowIndex: number,
  header: string,
  valueToSet: unknown,
  options: { addMissing?: boolean } = {},
): boolean {
  const row = sheets[sheetName]?.[rowIndex];
  if (!row) return false;
  const aliasGroups = [
    ['DL Carrier Bandwidth', 'DLBandwidth'],
    ['UL Carrier Bandwidth', 'ULBandwidth'],
    ['Power Level', 'PowerModify'],
    ['Offset To Point A', 'OffsetToPointA'],
    ['SSB Subcarrier Offset', 'SsbSubcarrierOffset'],
    ['AMF IP:DEFAULT', 'AMF IP'],
  ];
  const requestedAliases = aliasGroups.find((aliases) => (
    aliases.some((alias) => canonicalHeader(alias) === canonicalHeader(header))
  )) ?? [header];
  const actualHeader = Object.keys(row).find((candidate) => (
    requestedAliases.some((alias) => canonicalHeader(candidate) === canonicalHeader(alias))
  ));
  if (!actualHeader) {
    if (!options.addMissing || !header) return false;
    row[header] = valueToSet ?? '';
    return true;
  }
  row[actualHeader] = valueToSet ?? '';
  return true;
}

function setExistingSheetValue(
  sheets: ImportedSheetParameters,
  headers: readonly string[],
  valueToSet: unknown,
): boolean {
  const expected = new Set(headers.map(canonicalHeader));
  for (const rows of Object.values(sheets)) {
    for (const row of rows) {
      const actualHeader = Object.keys(row).find((header) => expected.has(canonicalHeader(header)));
      if (actualHeader) {
        row[actualHeader] = valueToSet ?? '';
        return true;
      }
    }
  }
  return false;
}

function setExistingRowValue(
  row: Record<string, unknown> | undefined,
  headers: readonly string[],
  valueToSet: unknown,
): boolean {
  if (!row) return false;
  const expected = new Set(headers.map(canonicalHeader));
  const actualHeader = Object.keys(row).find((header) => expected.has(canonicalHeader(header)));
  if (!actualHeader) return false;
  row[actualHeader] = valueToSet ?? '';
  return true;
}

function setMappedWorkbookValue(
  current: ParamConfigDetailSource,
  sheets: ImportedSheetParameters,
  field: string,
  valueToSet: unknown,
): boolean {
  const target = FIELD_MAPPING_TARGETS[field];
  if (!target) return false;
  const mapping = (current.workbookMappings ?? []).find((item) => mappingMatchesTarget(item, target));
  if (!mapping) return false;
  return setSheetRowValue(sheets, mapping.sheet, 0, mapping.header, valueToSet, { addMissing: true });
}

function setMappedWorkbookValueForTargetAtRow(
  current: ParamConfigDetailSource,
  sheets: ImportedSheetParameters,
  target: FieldMappingTarget,
  rowIndex: number,
  valueToSet: unknown,
): boolean {
  const mapping = (current.workbookMappings ?? []).find((item) => mappingMatchesTarget(item, target));
  if (!mapping) return false;
  return setSheetRowValue(sheets, mapping.sheet, rowIndex, mapping.header, valueToSet, { addMissing: true });
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
  // validateFields() only returns mounted Form.Item values. Imported workbooks
  // contain product-specific sheets and columns that are intentionally not all
  // rendered by the editor, so the imported structure must remain canonical.
  // Only overlay cells that already exist in that structure; structured editors
  // below handle intentional row additions/replacements.
  const primarySheet = current.deviceType === 'GSM' ? 'GSM' : 'CELL';
  const sheets = sanitizeRetiredParamConfigFields(mergeExistingSheetCells(
    current.sheetParameters,
    submitted.sheetParameters as ImportedSheetParameters,
    new Set([primarySheet]),
  ), current.deviceType);
  mergeSubmittedInterfaceNamesIntoInterfaceSheet(
    current,
    sheets,
    submitted.sheetParameters as ImportedSheetParameters | undefined,
  );
  const networkParameterValues = submitted.networkParameterValues as Record<string, unknown> | undefined;
  if (networkParameterValues) {
    for (const mapping of current.workbookMappings ?? []) {
      if (Object.prototype.hasOwnProperty.call(networkParameterValues, mapping.trPath)) {
        setFirstSheetValue(
          sheets,
          mapping.sheet,
          mapping.header,
          networkParameterValues[mapping.trPath],
        );
      }
    }
  }
  mergeNetworkParametersIntoInterfaceSheet(current, sheets, networkParameterValues);
  const mappings = current.deviceType === 'gNB'
    ? GNB_SHEET_FIELD_MAPPINGS
    : current.deviceType === 'eNB'
      ? ENB_SHEET_FIELD_MAPPINGS
      : [];

  for (const mapping of mappings) {
    if (!valuesEqual(submitted[mapping.field], originalForm[mapping.field])) {
      const aliases = [
        mapping.header,
        ...(FIELD_MAPPING_TARGETS[mapping.field]?.headers ?? []),
      ];
      if (!setMappedWorkbookValue(current, sheets, mapping.field, submitted[mapping.field])
        && !setFirstSheetValue(sheets, mapping.sheet, mapping.header, submitted[mapping.field])) {
        setExistingSheetValue(sheets, aliases, submitted[mapping.field]);
      }
    }
  }

  for (const target of DIRECT_SHEET_MAPPING_TARGETS) {
    const sheetName = target.sheetByDeviceType[current.deviceType as 'eNB' | 'gNB' | 'GSM'];
    if (!sheetName) continue;
    const submittedRows = (submitted.sheetParameters as ImportedSheetParameters | undefined)?.[sheetName] ?? [];
    const originalRows = (originalForm.sheetParameters as ImportedSheetParameters | undefined)?.[sheetName] ?? [];
    submittedRows.forEach((submittedRow, rowIndex) => {
      const nextValue = value(submittedRow, target.header);
      if (nextValue === undefined) return;
      const originalValue = value(originalRows[rowIndex] ?? {}, target.header);
      if (!valuesEqual(nextValue, originalValue)) {
        if (!setMappedWorkbookValueForTargetAtRow(current, sheets, target, rowIndex, nextValue)) {
          setExistingRowValue(sheets[sheetName]?.[rowIndex], target.headers, nextValue);
        }
      }
    });
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

  if (current.deviceType === 'gNB'
    && !valuesEqual(submitted.networkConfigList, originalForm.networkConfigList)) {
    const list = submitted.networkConfigList as Array<Record<string, unknown>> | undefined;
    replaceSheetRows(sheets, 'INTERFACE', list?.map((item) => (
      Object.fromEntries(GNB_NETWORK_CONFIG_FIELDS.map(({ header }) => [header, item[header] ?? '']))
    )), current.serialNumber);
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
  const result = {
    ...current,
    ...submitted,
    ...refreshed,
    ...(current.deviceType === 'gNB'
      ? { IPSEC_ENABLE: submitted.IPSEC_ENABLE }
      : {}),
    sheetParameters: sheets,
    customParams: mergeUnmappedNetworkParametersIntoCustomParams(
      current,
      submitted.customParams,
      networkParameterValues,
    ),
  } as T & Record<string, unknown>;
  // networkParameterValues is transient form state reconstructed from the
  // workbook mappings and customParams. Persisting it on a device override
  // makes common-policy materialization convert the same public parameters,
  // producing a second stale value on subsequent edits and executions.
  delete result.networkParameterValues;
  return result;
}

export function toParamConfigFormValues(
  config: ParamConfigDetailSource,
): Record<string, unknown> {
  const sheets = withMappedDirectSheetValues(config);
  const networkParameterValues = mappedParameterValues(config);
  const cell = firstRow(sheets, 'CELL');
  const plmnRows = sheets?.PLMN ?? [];
  const firstPlmn = plmnRows[0] ?? {};

  if (config.deviceType === 'gNB') {
    const device = firstRow(sheets, 'DEVICE');
    const sync = firstRow(sheets, '1588_CONFIGURATION');
    const dlBandwidth = stringValue(mappedWorkbookValue(config, 'dlbandwidth')
      ?? value(cell, 'DL Carrier Bandwidth', 'DLBandwidth')
      ?? sheetValue(sheets, 'DL Carrier Bandwidth', 'DLBandwidth'));
    const ulBandwidth = stringValue(mappedWorkbookValue(config, 'ulbandwidth')
      ?? value(cell, 'UL Carrier Bandwidth', 'ULBandwidth')
      ?? sheetValue(sheets, 'UL Carrier Bandwidth', 'ULBandwidth'));
    const ipsecRows = sheets?.IPSEC ?? [];
    const populatedIpsecRows = ipsecRows.filter(hasMeaningfulSheetValues);
    const configuredIpsecEnable = binaryStringValue(config.IPSEC_ENABLE ?? config.ipsecEnable);
    const networkRows = sheets?.INTERFACE ?? [];
    const populatedNetworkRows = networkRows.filter(hasMeaningfulNetworkValues);
    const legacyNetworkConfig = compact({
      'Address Type': config.addressType,
      'IP Address': config.serviceIp,
      'Subnet Mask': config.serviceMask,
      'Prefix Length': config.prefixLength,
      'Gateway': config.serviceGateway,
      'Bear Type': config.bearType,
      'Vlan Name': config.vlanName,
      'Vlan ID': config.serviceVlan,
    });
    return {
      ...config,
      sheetParameters: sheets,
      ...(Object.keys(networkParameterValues).length > 0 ? { networkParameterValues } : {}),
      ...compact({
        IPSEC_ENABLE: configuredIpsecEnable || (populatedIpsecRows.length > 0 ? '1' : '0'),
        gnbName: mappedWorkbookValue(config, 'gnbName')
          ?? value(cell, 'gNB Name')
          ?? value(device, 'gNB Name')
          ?? mappedPathValue(config, ['GNBName'])
          ?? config.cellName,
        gnbId: mappedWorkbookValue(config, 'gnbId')
          ?? value(cell, '*gNB ID', 'gNB ID')
          ?? value(device, '*gNB ID', 'gNB ID')
          ?? mappedPathValue(config, ['GNBID']),
        gnbIdLength: mappedWorkbookValue(config, 'gnbIdLength')
          ?? value(cell, '*gNB Lenth', '*gNB Length', 'gNB ID Length')
          ?? value(device, '*gNB Lenth', '*gNB Length', 'gNB ID Length')
          ?? mappedPathValue(config, ['GNBIDLength', 'GNBIDLenth']),
        pci: mappedWorkbookValue(config, 'pci')
          ?? value(cell, '*PCI', 'PCI')
          ?? sheetValue(sheets, '*PCI', 'PCI')
          ?? mappedPathValue(config, ['PCI', 'PhyCellID', 'PhysicalCellID']),
        ssbFrequency: mappedWorkbookValue(config, 'ssbFrequency')
          ?? value(cell, 'SSB Frequency')
          ?? sheetValue(sheets, 'SSB Frequency')
          ?? mappedPathValue(config, ['SSBFrequency']),
        freqBandIndicator: mappedWorkbookValue(config, 'freqBandIndicator')
          ?? value(cell, 'Freq BandIndicator', 'Band')
          ?? sheetValue(sheets, 'Freq BandIndicator', 'FreqBandIndicatorNR', 'Band')
          ?? mappedPathValue(config, ['FreqBandIndicator', 'FreqBandIndicatorNR', 'Band']),
        nrarfcnndl: mappedWorkbookValue(config, 'nrarfcnndl')
          ?? value(cell, 'NRARFCNDL')
          ?? sheetValue(sheets, 'NRARFCNDL'),
        nrarfcnul: mappedWorkbookValue(config, 'nrarfcnul')
          ?? value(cell, 'NRARFCNUL')
          ?? sheetValue(sheets, 'NRARFCNUL'),
        dlbandwidth: dlBandwidth?.replace(/\s*MHz$/i, ''),
        ulbandwidth: ulBandwidth?.replace(/\s*MHz$/i, ''),
        nci: mappedWorkbookValue(config, 'nci')
          ?? value(firstPlmn, '*NCI', 'NCI'),
        tac: mappedWorkbookValue(config, 'tac')
          ?? value(firstPlmn, '*TAC', 'TAC'),
        ranac: mappedWorkbookValue(config, 'ranac')
          ?? value(firstPlmn, '*RANAC', 'RANAC'),
        plmnId: mappedWorkbookValue(config, 'plmnId')
          ?? value(firstPlmn, '*PLMN ID', 'PLMN ID'),
        ntpSync: binaryStringValue(mappedWorkbookValue(config, 'ntpSync')
          ?? value(device, 'NTP Enable', 'NTP Mode')),
        SyncSource: mappedWorkbookValue(config, 'SyncSource')
          ?? value(sync, 'SyncSource', 'Sync Source'),
        ForcedSync: mappedWorkbookValue(config, 'ForcedSync')
          ?? value(sync, 'ForcedSync', 'Forced Sync'),
        PTPProfile: mappedWorkbookValue(config, 'PTPProfile')
          ?? value(sync, 'PTPProfile', 'Profile'),
        PTPDomain: mappedWorkbookValue(config, 'PTPDomain')
          ?? value(sync, 'PTPDomain', 'Domain'),
        PTPTransmode: mappedWorkbookValue(config, 'PTPTransmode')
          ?? value(sync, 'PTPTransmode', 'Transmission Mode'),
        PTPInterface: mappedWorkbookValue(config, 'PTPInterface')
          ?? value(sync, 'PTPInterface', 'Interface'),
        PTPUnicastMode: mappedWorkbookValue(config, 'PTPUnicastMode')
          ?? value(sync, 'PTPUnicastMode', 'Unicast Mode'),
        PTPSyncInterval: mappedWorkbookValue(config, 'PTPSyncInterval')
          ?? value(sync, 'PTPSyncInterval', 'Sync Interval'),
        PTPDelayInterval: mappedWorkbookValue(config, 'PTPDelayInterval')
          ?? value(sync, 'PTPDelayInterval', 'Delay Interval'),
        networkConfigList: populatedNetworkRows.length > 0
          ? populatedNetworkRows.map((row) => compact(Object.fromEntries(
            GNB_NETWORK_CONFIG_FIELDS.map(({ header }) => [header, value(row, header)]),
          )))
          : (Object.keys(legacyNetworkConfig).length > 0 ? [legacyNetworkConfig] : undefined),
        serviceIp: value(firstRow(sheets, 'INTERFACE'), 'IP Address'),
        serviceMask: value(firstRow(sheets, 'INTERFACE'), 'Subnet Mask'),
        serviceGateway: value(firstRow(sheets, 'INTERFACE'), 'Gateway'),
        serviceVlan: value(firstRow(sheets, 'INTERFACE'), 'Vlan ID'),
        totalTxPower: mappedWorkbookValue(config, 'totalTxPower')
          ?? value(cell, 'PowerModify')
          ?? sheetValue(sheets, 'PowerModify', 'Power Level'),
        offsetToPointA: mappedWorkbookValue(config, 'offsetToPointA')
          ?? value(cell, 'OffsetToPointA')
          ?? sheetValue(sheets, 'OffsetToPointA', 'Offset To Point A'),
        kssb: mappedWorkbookValue(config, 'kssb')
          ?? value(cell, 'SsbSubcarrierOffset')
          ?? sheetValue(sheets, 'SsbSubcarrierOffset', 'SSB Subcarrier Offset'),
        RFEnable: mappedWorkbookValue(config, 'RFEnable')
          ?? value(cell, 'RFEnable', 'RF Enable'),
        amfList: value(firstPlmn, 'AMF IP:DEFAULT', 'AMF IP') !== undefined
          ? [{ amfIp: value(firstPlmn, 'AMF IP:DEFAULT', 'AMF IP'), amfPort: '' }]
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
        ipsecList: populatedIpsecRows.length > 0
          ? populatedIpsecRows.map((row, index) => compact({
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
      sheetParameters: sheets,
      ...(Object.keys(networkParameterValues).length > 0 ? { networkParameterValues } : {}),
      ...compact({
        cellName: mappedWorkbookValue(config, 'cellName')
          ?? value(cell, 'CELL_NAME', 'Cell Name')
          ?? config.cellName,
        cellIdentity: mappedWorkbookValue(config, 'cellIdentity')
          ?? value(cell, '*ECI', 'ECI'),
        bandsSupport: mappedWorkbookValue(config, 'bandsSupport')
          ?? value(cell, '*BAND', 'BAND')
          ?? config.bandsSupport,
        frequency: mappedWorkbookValue(config, 'frequency')
          ?? value(cell, '*EARFCN_DL', 'EARFCN_DL')
          ?? config.frequency,
        bandWidth: mappedWorkbookValue(config, 'bandWidth')
          ?? value(cell, '*BANDWIDTH_DL', 'BANDWIDTH_DL')
          ?? config.bandWidth,
        phycellid: mappedWorkbookValue(config, 'phycellid')
          ?? value(cell, '*PCI', 'PCI'),
        specialSubframePatterns: stringValue(
          mappedWorkbookValue(config, 'specialSubframePatterns')
          ?? value(cell, 'SPECIAL_SUBFRAME_PATTERNS'),
        ),
        subframeAssignment: stringValue(
          mappedWorkbookValue(config, 'subframeAssignment')
          ?? value(cell, 'SUBFRAME_ASSIGNMENT'),
        ) ?? stringValue(config.subframeAssignment),
        rootSequenceIndex: mappedWorkbookValue(config, 'rootSequenceIndex')
          ?? value(cell, '*ROOT_SEQUENCE_INDEX', 'ROOT_SEQUENCE_INDEX'),
        tac: mappedWorkbookValue(config, 'tac')
          ?? value(cell, '*TAC', 'TAC'),
        plmnId: mappedWorkbookValue(config, 'plmnId')
          ?? value(network, '*PLMN', 'PLMN'),
        plmnConfigList: config.plmnConfigList?.length
          ? config.plmnConfigList
          : (value(network, '*PLMN', 'PLMN') !== undefined
            ? [{ plmnId: value(network, '*PLMN', 'PLMN') }]
            : undefined),
        ntpSync: stringValue(mappedWorkbookValue(config, 'ntpSync')
          ?? value(firstRow(sheets, 'NETWORK'), 'NTP Enable', 'NTP Mode')),
        mmeList: mmeIp !== undefined ? [{ mmeIp }] : undefined,
        ipsecEnable: stringValue(mappedWorkbookValue(config, 'ipsecEnable')
          ?? value(network, 'IPSEC_ENABLE')) || '0',
        halobEnable: stringValue(mappedWorkbookValue(config, 'halobEnable')
          ?? value(network, 'HALOB_ENABLE')),
        totalTxPower: mappedWorkbookValue(config, 'totalTxPower')
          ?? value(cell, 'X_COM_MaxTxPowerExpanded', 'ReferenceSignalPower', 'PowerClass', 'Transmit Power', 'Tx Power')
          ?? value(cell, 'MaxTxPower'),
        serviceIp: mappedWorkbookValue(config, 'serviceIp')
          ?? value(firstRow(sheets, 'NETWORK'), 'WAN IP'),
        tfcsManagerPrimsrc: value(
          firstRow(sheets, '1588_CONFIGURATION'),
          '*SYNCHRONIZATION_MODE',
        ) ?? config.tfcsManagerPrimsrc,
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

  return {
    ...config,
    sheetParameters: sheets,
    ...(Object.keys(networkParameterValues).length > 0 ? { networkParameterValues } : {}),
  };
}

export function materializeParamConfigDisplayValues<T extends ParamConfigDetailSource>(
  config: T,
): T & Record<string, unknown> {
  const formValues = toParamConfigFormValues(config);
  const cellName = stringValue(
    stringValue(formValues.cellName)
      ?? stringValue(formValues.gnbName)
      ?? stringValue(config.cellName),
  );
  return {
    ...config,
    ...formValues,
    ...(cellName !== undefined ? { cellName } : {}),
  } as T & Record<string, unknown>;
}
