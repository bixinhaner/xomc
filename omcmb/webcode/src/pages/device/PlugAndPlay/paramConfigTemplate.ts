import type { ParamConfigDeviceType } from './paramConfigWorkbook';

export interface ParamConfigTemplate {
  url: string;
  fileName: string;
}

const PARAM_CONFIG_TEMPLATES: Partial<Record<ParamConfigDeviceType, ParamConfigTemplate>> = {
  eNB: {
    url: '/templates/selfConfiguration_qa.xlsx',
    fileName: 'selfConfiguration_qa.xlsx',
  },
  gNB: {
    url: '/templates/5G_Autonomous_Self-Deployment_Template.xlsx',
    fileName: '5G_Autonomous_Self-Deployment_Template.xlsx',
  },
  GSM: {
    url: '/templates/GSM_Autonomous_Self-Deployment_Template.xlsx',
    fileName: 'GSM_Autonomous_Self-Deployment_Template.xlsx',
  },
};

export type ParamConfigTemplateSheets = Record<string, readonly string[]>;

const PARAM_CONFIG_TEMPLATE_SHEETS: Partial<
  Record<ParamConfigDeviceType, ParamConfigTemplateSheets>
> = {
  eNB: {
    CELL: [
      '*SERIAL_NUMBER', '*CELL_NUMBER', 'CELL_NAME', '*ECI', '*BAND',
      '*EARFCN_DL', '*BANDWIDTH_DL', '*PCI', 'SPECIAL_SUBFRAME_PATTERNS',
      'SUBFRAME_ASSIGNMENT', '*ROOT_SEQUENCE_INDEX', '*TAC',
      'X_COM_MaxTxPowerExpanded', 'MaxTxPower',
    ],
    NETWORK_ENABLE: [
      '*SERIAL_NUMBER', '*PLMN', 'MME_IP', 'IPSEC_ENABLE', 'HALOB_ENABLE',
    ],
    NETWORK_IPSEC: [
      '*SERIAL_NUMBER', '*TUNNEL_INDEX', '*TUNNEL_ENABLE', '*TUNNEL_GATEWAY',
      'LEFT_IDENTIFIER', 'RIGHT_IDENTIFIER', 'LEFT_AUTH', 'RIGHT_AUTH',
      'LEFT_CERT', 'SECRET_KEY', 'LEFTSOURCEIP', 'IKE_ENCRYPTION',
      'ESP_ENCRYPTION', 'IKE_DH_GROUP', 'ESP_DH_GROUP', 'IKE_AUTHENTICATION',
      'ESP_AUTHENTICATION', 'KEYLIFE', 'IKELIFETIME', 'REKEYMARGIN',
      'DPDACTION', 'DPDDELAY', 'RIGHT_SECRET_KEY', 'LEFT_SUBNET',
      'RIGHT_SUBNET', 'FRAGMENTATION', 'LEFT_INTERFACE', 'FORCEENCAPS',
    ],
    '1588_CONFIGURATION': [
      '*SERIAL_NUMBER', '*1588_ENABLE', '*SYNCHRONIZATION_MODE', '*SYNC_MODE',
      '*MODE_SWITCH', '*DOMAIN', '*SYNC_INTERVAL', '*DELAY_INTERVAL',
      '*ASYMMETRY', '*STARTUP_TIME', 'UNICAST_SERVER_IP_ADDRESS',
    ],
    NETWORK: ['*SERIAL_NUMBER', 'WAN IP', 'NTP Enable', 'NTP Server1', 'Local Time Zone'],
  },
  gNB: {
    DEVICE: [
      'Serial Number', 'URL', 'Periodic Inform Enable', 'Periodic Inform Time',
      'Periodic Inform Interval', 'NTP Enable', 'NTP Server1', 'NTP Server2',
      'NTP Server3', 'NTP Server4', 'NTP Server5', 'Local Time Zone',
      'PpsTimeMode',
    ],
    CELL: [
      '*Serial Number', 'gNB Name', '*gNB ID', '*gNB Lenth', '*PCI',
      'SSB Frequency', 'Freq BandIndicator', 'NRARFCNDL', 'NRARFCNUL',
      'DLBandwidth', 'ULBandwidth', 'DLAntNum', 'ULAntNum',
      'DL ULTransmissionPeriodicity1', 'Nrof DownlinkSlots1',
      'Nrof DownlinkSymbols1', 'Nrof  UplinkSlots1', 'Nrof  UplinkSymbols1',
      'DL ULTransmissionPeriodicity2', 'Nrof  DownlinkSlots2',
      'Nrof  DownlinkSymbols2', 'Nrof  UplinkSlots2', 'Nrof  UplinkSymbols2',
      'Prach RootSequenceIndex', 'Prach RootSequenceValue',
      'SubcarrierSpacing(UL)', 'SubcarrierSpacing(DL)', 'PowerModify',
      'OffsetToPointA', 'SsbSubcarrierOffset',
    ],
    PLMN: [
      'Serial Number', '*NCI', '*TAC', '*RANAC', '*PLMN ID', '*PRIMARY',
      'SD', 'SD Value', 'AMF IP:DEFAULT', 'NguBindInterface',
    ],
    INTERFACE: [
      'Serial Number', 'Interface Name', 'Address Type', 'IP Address',
      'Subnet Mask', 'Prefix Length', 'Gateway', 'Bear Type', 'Vlan Name',
      'Vlan ID',
    ],
    IPSEC: [
      'Serial Number', 'TUNNEL_ENABLE', 'TUNNEL_GATEWAY', 'LEFT_AUTH',
      'RIGHT_AUTH', 'RIGHT_SUBNET', 'LEFT_IDENTIFIER', 'RIGHT_IDENTIFIER',
      'LEFTSOURCEIP', 'LEFT_SUBNET', 'FRAGMENTATION', 'IKE_ENCRYPTION',
      'IKE_DH_GROUP', 'IKE_AUTHENTICATION', 'ESP_ENCRYPTION', 'ESP_DH_GROUP',
      'ESP_AUTHENTICATION', 'KEYLIFE', 'IKELIFETIME', 'REKEYMARGIN',
      'DPDACTION', 'DPDDELAY', 'LEFT_INTERFACE', 'FORCEENCAPS',
    ],
  },
  GSM: {
    GSM: [
      'Serial Number', 'IPA', 'Unit ID', 'Remote IP', 'Bind IP', 'WAN IP',
      'Synchronization', 'OMC',
    ],
  },
};

const PARAM_CONFIG_TEMPLATE_DEFAULTS: Partial<
  Record<ParamConfigDeviceType, Record<string, Record<string, unknown>>>
> = {
  eNB: {
    NETWORK_ENABLE: { IPSEC_ENABLE: '0' },
    NETWORK: { 'NTP Enable': '0' },
  },
  GSM: {
    GSM: { IPA: '6969', Synchronization: 'GNSS' },
  },
};

export function toParamConfigDeviceType(
  technology: string | undefined,
  paramModelName?: string,
): ParamConfigDeviceType | undefined {
  // BM is catalogued as a 4G/eNB product for inventory and KPI purposes, but
  // its plug-and-play radio instances are GSM GsmBTSCellDT objects.
  if (paramModelName?.trim().toUpperCase() === 'BM') return 'GSM';
  const normalized = technology?.trim().toLowerCase();
  if (normalized === 'lte' || normalized === 'enb') return 'eNB';
  if (normalized === 'nr' || normalized === 'gnb') return 'gNB';
  if (normalized === 'gsm') return 'GSM';
  return undefined;
}

export function getParamConfigTemplate(
  deviceType: ParamConfigDeviceType | undefined,
): ParamConfigTemplate | undefined {
  return deviceType ? PARAM_CONFIG_TEMPLATES[deviceType] : undefined;
}

export function getParamConfigTemplateSheets(
  deviceType: ParamConfigDeviceType | undefined,
): ParamConfigTemplateSheets | undefined {
  return deviceType ? PARAM_CONFIG_TEMPLATE_SHEETS[deviceType] : undefined;
}

export function getParamConfigTemplateDefaults(
  deviceType: ParamConfigDeviceType | undefined,
): Record<string, Record<string, unknown>> {
  return deviceType ? PARAM_CONFIG_TEMPLATE_DEFAULTS[deviceType] ?? {} : {};
}
