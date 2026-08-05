import { getParamConfigTemplateSheets } from './paramConfigTemplate';
import type { ParamConfigDeviceType } from './paramConfigWorkbook';

export interface TemplateFieldRef {
  sheet: string;
  header: string;
}

const fields = (sheet: string, headers: readonly string[]): TemplateFieldRef[] => (
  headers.map((header) => ({ sheet, header }))
);

export const ENB_GROUPED_TEMPLATE_FIELDS = {
  quickTime: fields('1588_CONFIGURATION', ['*SYNCHRONIZATION_MODE'])
    .concat(fields('NETWORK', ['NTP Server1', 'Local Time Zone'])),
  other: fields('CELL', ['*CELL_NUMBER']).concat(fields('1588_CONFIGURATION', [
    '*1588_ENABLE', '*SYNC_MODE', '*MODE_SWITCH',
    '*DOMAIN', '*SYNC_INTERVAL', '*DELAY_INTERVAL', '*ASYMMETRY',
    '*STARTUP_TIME', 'UNICAST_SERVER_IP_ADDRESS',
  ])),
} as const;

export const GNB_GROUPED_TEMPLATE_FIELDS = {
  quickCell: fields('CELL', [
    'SubcarrierSpacing(DL)', 'SubcarrierSpacing(UL)', 'ULBandwidth',
    'DLAntNum', 'ULAntNum',
  ]),
  quickTime: fields('DEVICE', [
    'Local Time Zone', 'NTP Server1', 'NTP Server2', 'NTP Server3',
    'NTP Server4', 'NTP Server5',
  ]),
  quickSyncSource: fields('DEVICE', ['PpsTimeMode']),
  quickCore: fields('PLMN', ['NguBindInterface']),
  tdd: fields('CELL', [
    'DL ULTransmissionPeriodicity1', 'Nrof DownlinkSlots1',
    'Nrof DownlinkSymbols1', 'Nrof  UplinkSlots1', 'Nrof  UplinkSymbols1',
    'DL ULTransmissionPeriodicity2', 'Nrof  DownlinkSlots2',
    'Nrof  DownlinkSymbols2', 'Nrof  UplinkSlots2', 'Nrof  UplinkSymbols2',
  ]),
  quickIpsec: fields('IPSEC', [
    'TUNNEL_ENABLE', 'TUNNEL_GATEWAY', 'LEFT_AUTH', 'RIGHT_AUTH',
    'LEFT_IDENTIFIER', 'RIGHT_IDENTIFIER', 'LEFTSOURCEIP', 'LEFT_SUBNET',
    'RIGHT_SUBNET', 'FRAGMENTATION', 'IKE_ENCRYPTION', 'IKE_DH_GROUP',
    'IKE_AUTHENTICATION', 'ESP_ENCRYPTION', 'ESP_DH_GROUP',
    'ESP_AUTHENTICATION', 'KEYLIFE', 'IKELIFETIME', 'REKEYMARGIN',
    'DPDACTION', 'DPDDELAY', 'LEFT_INTERFACE',
  ]),
  other: fields('CELL', [
    'Prach RootSequenceIndex', 'Prach RootSequenceValue',
  ]).concat(fields('DEVICE', [
    'Periodic Inform Enable', 'Periodic Inform Time', 'Periodic Inform Interval',
    'Time Zone Term',
  ])).concat(fields('INTERFACE', [
    'Interface Name', 'Address Type', 'Prefix Length', 'Bear Type', 'Vlan Name',
  ])).concat(fields('IPSEC', ['FORCEENCAPS'])),
} as const;

export const GSM_GROUPED_TEMPLATE_FIELDS = {
  quickAbis: fields('GSM', ['IPA', 'Unit ID', 'Bind IP', 'Remote IP']),
  other: fields('GSM', ['WAN IP', 'Synchronization', 'OMC']),
} as const;

const ENB_STRUCTURED_FIELDS = [
  ...fields('CELL', [
    'CELL_NAME', '*ECI', '*BAND', '*EARFCN_DL', '*BANDWIDTH_DL', '*PCI',
    'SPECIAL_SUBFRAME_PATTERNS', 'SUBFRAME_ASSIGNMENT', '*ROOT_SEQUENCE_INDEX', '*TAC',
    'MaxTxPower',
  ]),
  ...fields('NETWORK_ENABLE', ['*PLMN', 'MME_IP', 'IPSEC_ENABLE', 'HALOB_ENABLE']),
  ...fields('NETWORK_IPSEC', [
    '*TUNNEL_INDEX', '*TUNNEL_ENABLE', '*TUNNEL_GATEWAY', 'LEFT_IDENTIFIER',
    'RIGHT_IDENTIFIER', 'LEFT_AUTH', 'RIGHT_AUTH', 'LEFT_CERT', 'SECRET_KEY',
    'LEFTSOURCEIP', 'IKE_ENCRYPTION', 'ESP_ENCRYPTION', 'IKE_DH_GROUP',
    'ESP_DH_GROUP', 'IKE_AUTHENTICATION', 'ESP_AUTHENTICATION', 'KEYLIFE',
    'IKELIFETIME', 'REKEYMARGIN', 'DPDACTION', 'DPDDELAY', 'RIGHT_SECRET_KEY',
    'LEFT_SUBNET', 'RIGHT_SUBNET', 'FRAGMENTATION', 'LEFT_INTERFACE', 'FORCEENCAPS',
  ]),
  ...fields('NETWORK', ['WAN IP', 'OMC IP', 'NTP Enable']),
];

const GNB_STRUCTURED_FIELDS = [
  ...fields('CELL', [
    'gNB Name', '*gNB ID', '*gNB Lenth', '*PCI', 'SSB Frequency',
    'Freq BandIndicator', 'NRARFCNDL', 'NRARFCNUL', 'DLBandwidth', 'Duplex Mode',
    'PowerModify', 'OffsetToPointA', 'SsbSubcarrierOffset',
  ]),
  ...fields('DEVICE', ['NTP Enable']),
  ...fields('PLMN', ['*NCI', '*TAC', '*RANAC', '*PLMN ID', '*PRIMARY', 'SD', 'SD Value']),
  ...fields('PLMN', ['AMF IP:DEFAULT']),
  ...fields('INTERFACE', ['IP Address', 'Subnet Mask', 'Gateway', 'Vlan ID', 'OMC IP']),
];

function key(field: TemplateFieldRef): string {
  return `${field.sheet}.${field.header}`;
}

function isSerialHeader(header: string): boolean {
  return header.replace(/^\*/, '').replace(/[\s_]+/g, '').toLowerCase() === 'serialnumber';
}

export function getTemplateFieldCoverage(deviceType: ParamConfigDeviceType): {
  expected: string[];
  covered: string[];
} {
  const sheets = getParamConfigTemplateSheets(deviceType) ?? {};
  const expected = Object.entries(sheets).flatMap(([sheet, headers]) => (
    headers.filter((header) => !isSerialHeader(header)).map((header) => key({ sheet, header }))
  ));
  const grouped = deviceType === 'eNB'
    ? Object.values(ENB_GROUPED_TEMPLATE_FIELDS).flat()
    : deviceType === 'gNB'
      ? Object.values(GNB_GROUPED_TEMPLATE_FIELDS).flat()
      : Object.values(GSM_GROUPED_TEMPLATE_FIELDS).flat();
  const structured = deviceType === 'eNB'
    ? ENB_STRUCTURED_FIELDS
    : deviceType === 'gNB' ? GNB_STRUCTURED_FIELDS : [];
  return { expected, covered: [...structured, ...grouped].map(key) };
}
