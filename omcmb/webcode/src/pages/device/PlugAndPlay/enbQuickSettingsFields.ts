import type { GnbQuickSettingField, GnbQuickSettingGroup } from './gnbQuickSettingsFields';

const sheetField = (sheet: string, header: string) => (
  ['sheetParameters', sheet, 0, header]
);

const technicalOptions = (values: string[]) => values.map((value) => ({ value, label: value }));

const onOffOptions = [
  { value: '1', labelKey: 'common.on' },
  { value: '0', labelKey: 'common.off' },
];

const ipsecFields: GnbQuickSettingField[] = [
  ['TUNNEL_ENABLE', 'TUNNEL_ENABLE', 'provision.lteQuick.tunnelEnable', 'switch'],
  ['TUNNEL_GATEWAY', 'TUNNEL_GATEWAY', 'provision.lteQuick.gatewayAddress'],
  ['LEFT_AUTH', 'LEFT_AUTH', 'provision.lteQuick.leftAuth'],
  ['RIGHT_AUTH', 'RIGHT_AUTH', 'provision.lteQuick.rightAuth'],
  ['LEFT_IDENTIFIER', 'LEFT_IDENTIFIER', 'provision.lteQuick.leftId'],
  ['RIGHT_IDENTIFIER', 'RIGHT_IDENTIFIER', 'provision.lteQuick.rightId'],
  ['LEFTSOURCEIP', 'LEFTSOURCEIP', 'provision.lteQuick.leftSourceIp'],
  ['LEFT_SUBNET', 'LEFT_SUBNET', 'provision.lteQuick.leftSubnet'],
  ['RIGHT_SUBNET', 'RIGHT_SUBNET', 'provision.lteQuick.rightSubnet'],
  ['FRAGMENTATION', 'FRAGMENTATION', 'provision.lteQuick.fragmentation'],
  ['IKE_ENCRYPTION', 'IKE_ENCRYPTION', 'provision.lteQuick.ikeEncryption'],
  ['IKE_DH_GROUP', 'IKE_DH_GROUP', 'provision.lteQuick.ikeDhGroup'],
  ['IKE_AUTHENTICATION', 'IKE_AUTHENTICATION', 'provision.lteQuick.ikeAuthentication'],
  ['ESP_ENCRYPTION', 'ESP_ENCRYPTION', 'provision.lteQuick.espEncryption'],
  ['ESP_DH_GROUP', 'ESP_DH_GROUP', 'provision.lteQuick.espDhGroup'],
  ['ESP_AUTHENTICATION', 'ESP_AUTHENTICATION', 'provision.lteQuick.espAuthentication'],
  ['KEYLIFE', 'KEYLIFE', 'provision.lteQuick.keyLife'],
  ['IKELIFETIME', 'IKELIFETIME', 'provision.lteQuick.ikeLifetime'],
  ['REKEYMARGIN', 'REKEYMARGIN', 'provision.lteQuick.rekeyMargin'],
  ['DPDACTION', 'DPDACTION', 'provision.lteQuick.dpdAction'],
  ['DPDDELAY', 'DPDDELAY', 'provision.lteQuick.dpdDelay'],
].map(([id, name, labelKey, control]) => ({
  id,
  name,
  labelKey,
  control: control as GnbQuickSettingField['control'],
}));

export const ENB_QUICK_SETTING_GROUPS: GnbQuickSettingGroup[] = [
  {
    id: 'enb-cell',
    titleKey: 'provision.lteQuick.cellParameters',
    fields: [
      { id: 'ECI', name: 'cellIdentity', labelKey: 'provision.lteQuick.eci', range: '0 ~ 268435455' },
      { id: 'CellName', name: 'cellName', labelKey: 'provision.lteQuick.cellName' },
      { id: 'TAC', name: 'tac', labelKey: 'provision.lteQuick.tac', range: '0 ~ 65535' },
      { id: 'PCI', name: 'phycellid', labelKey: 'provision.lteQuick.pci', range: '0 ~ 503' },
      { id: 'MaxTxPower', name: 'totalTxPower', labelKey: 'provision.lteQuick.maxTxPower' },
      { id: 'BandSupport', name: 'BandSupport', labelKey: 'provision.lteQuick.bandSupport', control: 'readonly', range: '1 ~ 62' },
      { id: 'BandIndicator', name: 'bandsSupport', labelKey: 'provision.lteQuick.bandIndicator', range: '1 ~ 62' },
      { id: 'DLEarfcn', name: 'frequency', labelKey: 'provision.lteQuick.earfcn', range: '1 ~ 65535' },
      { id: 'DLBandWidth', name: 'bandWidth', labelKey: 'provision.lteQuick.bandwidth', control: 'select', options: [
        { value: 'n25', label: '5MHz' },
        { value: 'n50', label: '10MHz' },
        { value: 'n75', label: '15MHz' },
        { value: 'n100', label: '20MHz' },
      ] },
      { id: 'SubFrameAssignment', name: 'subframeAssignment', labelKey: 'provision.lteQuick.subframeAssignment', control: 'select', options: [
        { value: '0', label: '0 (DL:UL = 1:3)' },
        { value: '1', label: '1 (DL:UL = 2:2)' },
        { value: '2', label: '2 (DL:UL = 3:1)' },
        { value: '6', label: '6 (DL:UL = 3:5)' },
      ] },
      { id: 'SpecialSubFramePatterns', name: 'specialSubframePatterns', labelKey: 'provision.lteQuick.specialSubframePatterns', control: 'select', options: technicalOptions(['5', '7']) },
      { id: 'RootSequenceIndex', name: 'rootSequenceIndex', labelKey: 'provision.lteQuick.rootSequenceIndex', range: '0 ~ 837' },
      { id: 'CellType', name: 'CellType', labelKey: 'provision.lteQuick.cellType', control: 'select', options: [
        { value: '0', labelKey: 'provision.lteQuick.macroCell' },
        { value: '1', labelKey: 'provision.lteQuick.homeCell' },
      ] },
      { id: 'PowerClass', name: 'PowerClass', labelKey: 'provision.lteQuick.powerClass', range: '0 ~ 46' },
    ],
  },
  {
    id: 'enb-plmn',
    titleKey: 'provision.lteQuick.servingPlmnList',
    fields: [{ id: 'PLMNID', name: 'plmnId', labelKey: 'provision.lteQuick.plmn' }],
  },
  {
    id: 'enb-mme',
    titleKey: 'provision.lteQuick.mmeList',
    fields: [],
  },
  {
    id: 'device-time',
    titleKey: 'provision.lteQuick.timeAndSync',
    fields: [
      { id: 'Enable', name: 'ntpSync', labelKey: 'provision.lteQuick.ntpEnable', control: 'select', options: [
        { value: '1', labelKey: 'common.enable' },
        { value: '0', labelKey: 'common.disable' },
      ] },
      { id: 'NTPServer1', name: sheetField('NETWORK', 'NTP Server1'), labelKey: 'provision.lteQuick.ntpServer', labelValues: { index: 1 }, range: '0 ~ 256' },
      ...[2, 3, 4, 5].map((index): GnbQuickSettingField => ({
        id: `NTPServer${index}`, name: `NTPServer${index}`,
        labelKey: 'provision.lteQuick.ntpServer', labelValues: { index }, range: '0 ~ 256',
      })),
      { id: 'LocalTimeZoneName', name: sheetField('NETWORK', 'Local Time Zone'), labelKey: 'provision.lteQuick.localTimeZone', control: 'timezone' },
    ],
  },
  {
    id: 'device-ipsec-control',
    titleKey: 'provision.lteQuick.ipsecConfig',
    fields: [{ id: 'IPSEC_ENABLE', name: 'ipsecEnable', labelKey: 'provision.lteQuick.ipsecEnable', control: 'select', options: onOffOptions }],
  },
  {
    id: 'device-ipsec',
    titleKey: 'provision.lteQuick.ipsecParameters',
    fields: ipsecFields,
    multiInstance: true,
  },
];

export const ENB_TEMPLATE_EXTRA_FIELDS: GnbQuickSettingField[] = [
  { id: 'CELL_NUMBER', name: sheetField('CELL', '*CELL_NUMBER'), labelKey: 'provision.lteQuick.cellNumber' },
  { id: 'HALOB_ENABLE', name: 'halobEnable', labelKey: 'provision.lteQuick.halobEnable', control: 'select', options: onOffOptions },
  { id: 'WAN_IP', name: 'serviceIp', labelKey: 'provision.lteQuick.serviceIp' },
  { id: 'OMC_IP', name: 'mgmtIp', labelKey: 'provision.lteQuick.omcIp' },
];

export const ENB_1588_TEMPLATE_FIELDS: GnbQuickSettingField[] = [
  {
    id: 'PpsTimeMode', name: 'PpsTimeMode', labelKey: 'provision.nrQuick.mode',
    control: 'select', options: technicalOptions([
      'FREE_OSCILLATION', 'GPS_PPS', 'LOCAL_CLOCK_HOLDOVER_GPS_PPS',
      'OCXO_PPS', '1588_PPS', 'GPS_AND_PTP',
    ]),
  },
  {
    id: 'SyncSource', name: 'SyncSource', labelKey: 'provision.nrQuick.syncSource',
    control: 'multi-select', options: technicalOptions(['GPS', 'GLONASS', 'BEIDOU', 'GALILEO', 'QZSS']),
  },
  {
    id: 'PTPProfile', name: 'PTPProfile', labelKey: 'provision.nrQuick.profileType',
    control: 'select', options: technicalOptions(['1588v2']),
  },
  { id: 'PTPDomain', name: 'PTPDomain', labelKey: 'provision.nrQuick.domainNumber' },
  {
    id: 'PTPTransmode', name: 'PTPTransmode', labelKey: 'provision.nrQuick.transmissionMode',
    control: 'select', options: technicalOptions(['L2', 'L3']),
  },
  {
    id: 'PTPInterface', name: 'PTPInterface', labelKey: 'provision.nrQuick.interface',
    control: 'select', options: technicalOptions(['eth-WAN', 'eth0', 'eth1']),
  },
  {
    id: 'PTPUnicastMode', name: 'PTPUnicastMode', labelKey: 'provision.nrQuick.unicastEnable',
    control: 'select', options: [
      { value: '0', labelKey: 'provision.nrQuick.multicast' },
      { value: '1', labelKey: 'provision.nrQuick.unicast' },
    ],
  },
  {
    id: 'PTPSyncInterval', name: 'PTPSyncInterval', labelKey: 'provision.nrQuick.syncInterval',
    control: 'select', options: technicalOptions(['-7', '-6', '-5', '-4', '-3', '-2', '-1', '0']),
  },
  {
    id: 'PTPDelayInterval', name: 'PTPDelayInterval', labelKey: 'provision.nrQuick.delayInterval',
    control: 'select', options: technicalOptions(['-7', '-6', '-5', '-4', '-3', '-2', '-1', '0']),
  },
];

export const ENB_IPSEC_TEMPLATE_EXTRA_FIELDS: GnbQuickSettingField[] = [
  ['TUNNEL_INDEX', 'provision.lteQuick.tunnelIndex'],
  ['LEFT_CERT', 'provision.lteQuick.leftCert'],
  ['SECRET_KEY', 'provision.lteQuick.secretKey'],
  ['RIGHT_SECRET_KEY', 'provision.lteQuick.rightSecretKey'],
  ['LEFT_INTERFACE', 'provision.lteQuick.leftInterface'],
  ['FORCEENCAPS', 'provision.lteQuick.forceEncapsulation'],
].map(([id, labelKey]) => ({ id, name: id, labelKey }));
