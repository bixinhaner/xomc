import type { NamePath } from 'antd/es/form/interface';

export type GnbQuickSettingControl =
  | 'input'
  | 'select'
  | 'multi-select'
  | 'switch'
  | 'readonly'
  | 'timezone'
  | 'dl-bandwidth'
  | 'ul-bandwidth';

export interface GnbQuickSettingOption {
  value: string;
  label?: string;
  labelKey?: string;
}

export interface GnbQuickSettingField {
  id: string;
  name: NamePath;
  labelKey: string;
  labelValues?: Record<string, string | number>;
  control?: GnbQuickSettingControl;
  options?: GnbQuickSettingOption[];
  range?: string;
}

export interface GnbQuickSettingGroup {
  id: string;
  titleKey: string;
  fields: GnbQuickSettingField[];
  multiInstance?: boolean;
}

const sheetField = (sheet: string, header: string): NamePath => (
  ['sheetParameters', sheet, 0, header]
);

const technicalOptions = (values: string[]): GnbQuickSettingOption[] => (
  values.map((value) => ({ value, label: value }))
);

const onOffOptions: GnbQuickSettingOption[] = [
  { value: '1', labelKey: 'common.on' },
  { value: '0', labelKey: 'common.off' },
];

const scsOptions: GnbQuickSettingOption[] = [
  { value: '0', label: '15kHz' },
  { value: '1', label: '30kHz' },
  { value: '2', label: '60kHz' },
];

const ipsecFields: GnbQuickSettingField[] = [
  ['TUNNEL_ENABLE', 'TUNNEL_ENABLE', 'provision.nrQuick.tunnelEnable'],
  ['TUNNEL_GATEWAY', 'TUNNEL_GATEWAY', 'provision.nrQuick.gatewayAddress'],
  ['TUNNEL_LEFT_AUTH', 'LEFT_AUTH', 'provision.nrQuick.leftAuth'],
  ['TUNNEL_RIGHT_AUTH', 'RIGHT_AUTH', 'provision.nrQuick.rightAuth'],
  ['LEFT_IDENTIFIER', 'LEFT_IDENTIFIER', 'provision.nrQuick.leftId'],
  ['RIGHT_IDENTIFIER', 'RIGHT_IDENTIFIER', 'provision.nrQuick.rightId'],
  ['LEFTSOURCEIP', 'LEFTSOURCEIP', 'provision.nrQuick.leftSourceIp'],
  ['LEFTSUBNET', 'LEFT_SUBNET', 'provision.nrQuick.leftSubnet'],
  ['RIGHT_SUBNET', 'RIGHT_SUBNET', 'provision.nrQuick.rightSubnet'],
  ['TUNNEL_FRAGMENTATION', 'FRAGMENTATION', 'provision.nrQuick.fragmentation'],
  ['IKE_ENCRYPTION', 'IKE_ENCRYPTION', 'provision.nrQuick.ikeEncryption'],
  ['IKE_DH_GROUP', 'IKE_DH_GROUP', 'provision.nrQuick.ikeDhGroup'],
  ['IKE_AUTHENTICATION', 'IKE_AUTHENTICATION', 'provision.nrQuick.ikeAuthentication'],
  ['ESP_ENCRYPTION', 'ESP_ENCRYPTION', 'provision.nrQuick.espEncryption'],
  ['ESP_DH_GROUP', 'ESP_DH_GROUP', 'provision.nrQuick.espDhGroup'],
  ['ESP_AUTHENTICATION', 'ESP_AUTHENTICATION', 'provision.nrQuick.espAuthentication'],
  ['KEYLIFE', 'KEYLIFE', 'provision.nrQuick.keyLife'],
  ['IKELIFETIME', 'IKELIFETIME', 'provision.nrQuick.ikeLifetime'],
  ['REKEYMARGIN', 'REKEYMARGIN', 'provision.nrQuick.rekeyMargin'],
  ['DPDACTION', 'DPDACTION', 'provision.nrQuick.dpdAction'],
  ['DPDDELAY', 'DPDDELAY', 'provision.nrQuick.dpdDelay'],
  ['LEFT_INTERFACE', 'LEFT_INTERFACE', 'provision.nrQuick.leftInterface'],
].map(([id, name, labelKey]) => ({ id, name, labelKey }));

export const NR_CARRIER_BANDWIDTH_OPTIONS_BY_SCS: Record<string, GnbQuickSettingOption[]> = {
  '0': technicalOptions(['25', '52', '79', '106', '133', '160', '216', '270']).map((option, index) => ({
    ...option,
    label: ['5MHz(25RB)', '10MHz(52RB)', '15MHz(79RB)', '20MHz(106RB)', '25MHz(133RB)', '30MHz(160RB)', '40MHz(216RB)', '50MHz(270RB)'][index],
  })),
  '1': technicalOptions(['11', '24', '38', '51', '65', '78', '106', '133', '162', '189', '217', '245', '273']).map((option, index) => ({
    ...option,
    label: ['5MHz(11RB)', '10MHz(24RB)', '15MHz(38RB)', '20MHz(51RB)', '25MHz(65RB)', '30MHz(78RB)', '40MHz(106RB)', '50MHz(133RB)', '60MHz(162RB)', '70MHz(189RB)', '80MHz(217RB)', '90MHz(245RB)', '100MHz(273RB)'][index],
  })),
  '2': technicalOptions(['11', '18', '24', '31', '38', '51', '65', '79', '93', '107', '121', '135']).map((option, index) => ({
    ...option,
    label: ['10MHz(11RB)', '15MHz(18RB)', '20MHz(24RB)', '25MHz(31RB)', '30MHz(38RB)', '40MHz(51RB)', '50MHz(65RB)', '60MHz(79RB)', '70MHz(93RB)', '80MHz(107RB)', '90MHz(121RB)', '100MHz(135RB)'][index],
  })),
};

export const GNB_QUICK_SETTING_GROUPS: GnbQuickSettingGroup[] = [
  {
    id: 'device-time',
    titleKey: 'provision.ntpConfig',
    fields: [
      { id: 'Enable', name: 'ntpSync', labelKey: 'provision.nrQuick.ntpMode', control: 'select', options: [
        { value: '1', labelKey: 'device.cell.ntpClient' },
        { value: '0', labelKey: 'device.cell.ntpServer' },
      ] },
      { id: 'LocalTimeZoneName', name: sheetField('DEVICE', 'Local Time Zone'), labelKey: 'provision.nrQuick.localTimeZone', control: 'timezone' },
      ...[1, 2, 3, 4, 5].map((index): GnbQuickSettingField => ({
        id: `NTPServer${index}`,
        name: sheetField('DEVICE', `NTP Server${index}`),
        labelKey: 'provision.nrQuick.ntpServer',
        labelValues: { index },
        range: '0 ~ 256',
      })),
    ],
  },
  {
    id: 'gnb-management',
    titleKey: 'provision.omcConfig',
    fields: [
      { id: 'ManagementServerURL', name: sheetField('DEVICE', 'URL'), labelKey: 'provision.nrQuick.managementServerUrl', range: '1 ~ 256' },
      { id: 'PeriodicInformEnable', name: sheetField('DEVICE', 'Periodic Inform Enable'), labelKey: 'provision.nrQuick.periodicInformEnable', control: 'select', options: onOffOptions },
      { id: 'PeriodicInformTime', name: sheetField('DEVICE', 'Periodic Inform Time'), labelKey: 'provision.nrQuick.periodicInformTime' },
      { id: 'PeriodicInformInterval', name: sheetField('DEVICE', 'Periodic Inform Interval'), labelKey: 'provision.nrQuick.periodicInformInterval' },
    ],
  },
  {
    id: 'gnb-sync-source',
    titleKey: 'provision.syncSourceConfig',
    fields: [
      { id: 'PpsTimeMode', name: sheetField('DEVICE', 'PpsTimeMode'), labelKey: 'provision.nrQuick.mode', control: 'select', options: technicalOptions([
        'FREE_OSCILLATION', 'GPS_PPS', 'LOCAL_CLOCK_HOLDOVER_GPS_PPS', 'OCXO_PPS', '1588_PPS', 'GPS_AND_PTP',
      ]) },
      { id: 'SyncSource', name: 'SyncSource', labelKey: 'provision.nrQuick.syncSource', control: 'multi-select', options: technicalOptions(['GPS', 'GLONASS', 'BEIDOU', 'GALILEO', 'QZSS']) },
      { id: 'ForcedSync', name: 'ForcedSync', labelKey: 'provision.nrQuick.forcedSync', control: 'switch', options: onOffOptions },
      { id: 'PTPProfile', name: 'PTPProfile', labelKey: 'provision.nrQuick.profileType', control: 'select', options: technicalOptions(['1588v2']) },
      { id: 'PTPDomain', name: 'PTPDomain', labelKey: 'provision.nrQuick.domainNumber' },
      { id: 'PTPTransmode', name: 'PTPTransmode', labelKey: 'provision.nrQuick.transmissionMode', control: 'select', options: technicalOptions(['L2', 'L3']) },
      { id: 'PTPInterface', name: 'PTPInterface', labelKey: 'provision.nrQuick.interface', control: 'select', options: technicalOptions(['eth-WAN', 'eth0', 'eth1']) },
      { id: 'PTPUnicastMode', name: 'PTPUnicastMode', labelKey: 'provision.nrQuick.unicastEnable', control: 'select', options: [
        { value: '0', labelKey: 'provision.nrQuick.multicast' },
        { value: '1', labelKey: 'provision.nrQuick.unicast' },
      ] },
      { id: 'PTPSyncInterval', name: 'PTPSyncInterval', labelKey: 'provision.nrQuick.syncInterval', control: 'select', options: technicalOptions(['-7', '-6', '-5', '-4', '-3', '-2', '-1', '0']) },
      { id: 'PTPDelayInterval', name: 'PTPDelayInterval', labelKey: 'provision.nrQuick.delayInterval', control: 'select', options: technicalOptions(['-7', '-6', '-5', '-4', '-3', '-2', '-1', '0']) },
    ],
  },
  {
    id: 'device-ipsec-control',
    titleKey: 'provision.nrQuick.ipsecConfigTitle',
    fields: [
      { id: 'IPSEC_ENABLE', name: 'IPSEC_ENABLE', labelKey: 'provision.nrQuick.ipsecSwitch', control: 'select', options: onOffOptions },
    ],
  },
  {
    id: 'gnb-ipsec',
    titleKey: 'provision.ipsecParameterConfig',
    fields: ipsecFields,
    multiInstance: true,
  },
  {
    id: 'gnb-cell',
    titleKey: 'provision.nrCellParameters',
    fields: [
      { id: 'Band', name: 'freqBandIndicator', labelKey: 'provision.nrQuick.band' },
      { id: 'SsbFrequency', name: 'ssbFrequency', labelKey: 'provision.nrQuick.ssbFrequency', range: '0 ~ 3279165' },
      { id: 'DLSubCarrierSpacing', name: sheetField('CELL', 'SubcarrierSpacing(DL)'), labelKey: 'provision.nrQuick.dlSubcarrierSpacing', control: 'select', options: scsOptions },
      { id: 'ULSubCarrierSpacing', name: sheetField('CELL', 'SubcarrierSpacing(UL)'), labelKey: 'provision.nrQuick.ulSubcarrierSpacing', control: 'select', options: scsOptions },
      { id: 'DLCarrierBandWidth', name: 'dlbandwidth', labelKey: 'provision.nrQuick.dlCarrierBandwidth', control: 'dl-bandwidth' },
      { id: 'ULCarrierBandWidth', name: 'ulbandwidth', labelKey: 'provision.nrQuick.ulCarrierBandwidth', control: 'ul-bandwidth' },
      { id: 'NRARFCNDL', name: 'nrarfcnndl', labelKey: 'provision.nrQuick.nrDlArfcn', range: '0 ~ 3279165' },
      { id: 'NRARFCNUL', name: 'nrarfcnul', labelKey: 'provision.nrQuick.nrUlArfcn', range: '0 ~ 3279165' },
      { id: 'NumOfRxAntenna', name: sheetField('CELL', 'ULAntNum'), labelKey: 'provision.nrQuick.rxAntennaCount', range: '1 ~ 4' },
      { id: 'NumOfTxAntenna', name: sheetField('CELL', 'DLAntNum'), labelKey: 'provision.nrQuick.txAntennaCount', range: '1 ~ 4' },
      { id: 'OffsetToPointA', name: 'offsetToPointA', labelKey: 'provision.nrQuick.offsetToPointA', range: '0 ~ 2199' },
      { id: 'PCI', name: 'pci', labelKey: 'provision.nrQuick.pci', range: '0 ~ 1007' },
      { id: 'PowerModify', name: 'totalTxPower', labelKey: 'provision.nrQuick.powerLevel', range: '0 ~ 43' },
      { id: 'RFEnable', name: 'RFEnable', labelKey: 'provision.nrQuick.rfEnable', control: 'select', options: onOffOptions },
      { id: 'SsbSubcarrierOffset', name: 'kssb', labelKey: 'provision.nrQuick.ssbSubcarrierOffset', range: '0 ~ 31' },
    ],
  },
  {
    id: 'gnb-core',
    titleKey: 'provision.nrCoreParameters',
    fields: [
      { id: 'PLMNID', name: 'plmnId', labelKey: 'provision.nrQuick.plmn' },
      { id: 'TAC', name: 'tac', labelKey: 'provision.nrQuick.tac' },
      { id: 'AmfIP1', name: ['amfList', 0, 'amfIp'], labelKey: 'provision.nrQuick.amfIp' },
      { id: 'gNBName', name: 'gnbName', labelKey: 'provision.nrQuick.gnbName' },
      { id: 'NguBindInterface', name: sheetField('PLMN', 'NguBindInterface'), labelKey: 'provision.nrQuick.nguLocalAddress' },
      { id: 'gNBId', name: 'gnbId', labelKey: 'provision.nrQuick.gnbId' },
      { id: 'gNBIdLength', name: 'gnbIdLength', labelKey: 'provision.nrQuick.gnbIdLength' },
      { id: 'NrcellIdentity', name: 'nci', labelKey: 'provision.nrQuick.nci', range: '0 ~ 68719476735' },
    ],
  },
  {
    id: 'gnb-tdd',
    titleKey: 'provision.nrTddConfig',
    fields: [
      { id: 'DlULTransmissionPeriodicity', name: sheetField('CELL', 'DL ULTransmissionPeriodicity1'), labelKey: 'provision.nrQuick.patternPeriod', labelValues: { pattern: 1 }, range: '0 ~ 7' },
      { id: 'NrofDownlinkSlots', name: sheetField('CELL', 'Nrof DownlinkSlots1'), labelKey: 'provision.nrQuick.patternDlSlots', labelValues: { pattern: 1 }, range: '0 ~ 320' },
      { id: 'NrofDownlinkSymbols', name: sheetField('CELL', 'Nrof DownlinkSymbols1'), labelKey: 'provision.nrQuick.patternDlSymbols', labelValues: { pattern: 1 }, range: '0 ~ 13' },
      { id: 'NrofUplinkSlots', name: sheetField('CELL', 'Nrof  UplinkSlots1'), labelKey: 'provision.nrQuick.patternUlSlots', labelValues: { pattern: 1 }, range: '0 ~ 320' },
      { id: 'NrofUplinkSymbols', name: sheetField('CELL', 'Nrof  UplinkSymbols1'), labelKey: 'provision.nrQuick.patternUlSymbols', labelValues: { pattern: 1 }, range: '0 ~ 13' },
      { id: 'Pat2DlULTransmissionPeriodicity', name: sheetField('CELL', 'DL ULTransmissionPeriodicity2'), labelKey: 'provision.nrQuick.patternPeriod', labelValues: { pattern: 2 }, range: '0 ~ 7' },
      { id: 'Pat2NrofDownlinkSlots', name: sheetField('CELL', 'Nrof  DownlinkSlots2'), labelKey: 'provision.nrQuick.patternDlSlots', labelValues: { pattern: 2 }, range: '0 ~ 320' },
      { id: 'Pat2NrofDownlinkSymbols', name: sheetField('CELL', 'Nrof  DownlinkSymbols2'), labelKey: 'provision.nrQuick.patternDlSymbols', labelValues: { pattern: 2 }, range: '0 ~ 13' },
      { id: 'Pat2NrofUplinkSlots', name: sheetField('CELL', 'Nrof  UplinkSlots2'), labelKey: 'provision.nrQuick.patternUlSlots', labelValues: { pattern: 2 }, range: '0 ~ 320' },
      { id: 'Pat2NrofUplinkSymbols', name: sheetField('CELL', 'Nrof  UplinkSymbols2'), labelKey: 'provision.nrQuick.patternUlSymbols', labelValues: { pattern: 2 }, range: '0 ~ 13' },
    ],
  },
];

export const GNB_TEMPLATE_EXTRA_FIELDS: GnbQuickSettingField[] = [
  { id: 'RANAC', name: 'ranac', labelKey: 'provision.nrQuick.ranac', range: '0 ~ 255' },
  { id: 'PrachRootSequenceIndex', name: sheetField('CELL', 'Prach RootSequenceIndex'), labelKey: 'provision.nrQuick.prachRootSequenceIndex' },
  { id: 'PrachRootSequenceValue', name: sheetField('CELL', 'Prach RootSequenceValue'), labelKey: 'provision.nrQuick.prachRootSequenceValue' },
  { id: 'FORCEENCAPS', name: sheetField('IPSEC', 'FORCEENCAPS'), labelKey: 'provision.nrQuick.forceEncapsulation' },
];

export const GNB_COMMON_EXCLUDED_EXTRA_FIELD_IDS = [
  'FORCEENCAPS',
] as const;
