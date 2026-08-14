export interface ParamConfigSheetFieldMapping {
  field: string;
  sheet: string;
  header: string;
}

export const ENB_SHEET_FIELD_MAPPINGS: readonly ParamConfigSheetFieldMapping[] = [
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
  { field: 'ntpSync', sheet: 'NETWORK', header: 'NTP Enable' },
  { field: 'tfcsManagerPrimsrc', sheet: '1588_CONFIGURATION', header: '*SYNCHRONIZATION_MODE' },
];

export const GNB_SHEET_FIELD_MAPPINGS: readonly ParamConfigSheetFieldMapping[] = [
  { field: 'gnbName', sheet: 'CELL', header: 'gNB Name' },
  { field: 'gnbId', sheet: 'CELL', header: '*gNB ID' },
  { field: 'gnbIdLength', sheet: 'CELL', header: '*gNB Lenth' },
  { field: 'pci', sheet: 'CELL', header: '*PCI' },
  { field: 'ssbFrequency', sheet: 'CELL', header: 'SSB Frequency' },
  { field: 'freqBandIndicator', sheet: 'CELL', header: 'Freq BandIndicator' },
  { field: 'nrarfcnndl', sheet: 'CELL', header: 'NRARFCNDL' },
  { field: 'nrarfcnul', sheet: 'CELL', header: 'NRARFCNUL' },
  { field: 'dlbandwidth', sheet: 'CELL', header: 'DL Carrier Bandwidth' },
  { field: 'ulbandwidth', sheet: 'CELL', header: 'UL Carrier Bandwidth' },
  { field: 'nci', sheet: 'PLMN', header: '*NCI' },
  { field: 'tac', sheet: 'PLMN', header: '*TAC' },
  { field: 'ranac', sheet: 'PLMN', header: '*RANAC' },
  { field: 'plmnId', sheet: 'PLMN', header: '*PLMN ID' },
  { field: 'ntpSync', sheet: 'DEVICE', header: 'NTP Enable' },
  { field: 'totalTxPower', sheet: 'CELL', header: 'PowerModify' },
  { field: 'offsetToPointA', sheet: 'CELL', header: 'OffsetToPointA' },
  { field: 'kssb', sheet: 'CELL', header: 'SsbSubcarrierOffset' },
];

export const PARAM_CONFIG_FIELD_MAPPING_MATRIX = {
  eNB: ENB_SHEET_FIELD_MAPPINGS,
  gNB: GNB_SHEET_FIELD_MAPPINGS,
  GSM: [],
} as const;
