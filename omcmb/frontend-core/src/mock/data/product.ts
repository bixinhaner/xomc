import type { Product, ProductPattern, OrphanDevice, MatchOrderRow } from '../../types/product';

export const mockProducts: Product[] = [
  {
    id: 'p-001',
    name: 'PicoCell-LTE-V2',
    vendor: 'Comba',
    tech: 'lte',
    radioModes: 'fdd-tdd',
    description: 'LTE 皮基站演示产品',
    paramModelId: 'pm-001',
    indicatorDeviceType: 'ENB',
    indicatorPlatform: 'enb-default',
    alarmNeType: 'eNodeB',
    enableFiletype11: true,
    deviceAttrsOverride: { data_type: false, access: true, min_value: false, max_value: false, change_applies: true },
    enableUnknownAlarm: true,
    deviceCount: 12,
    isBuiltin: true,
    patterns: ['PicoCell-LTE-.+', 'CombaPico.*'],
  },
  {
    id: 'p-002',
    name: 'MicroCell-NR-Standard',
    vendor: 'Baicells',
    tech: 'nr',
    radioModes: 'tdd',
    description: '5G NR 微基站标准型号',
    paramModelId: 'pm-002',
    indicatorDeviceType: 'GNB',
    indicatorPlatform: 'gnb-default',
    alarmNeType: 'gNodeB',
    enableFiletype11: false,
    deviceAttrsOverride: { data_type: false, access: false, min_value: false, max_value: false, change_applies: false },
    enableUnknownAlarm: false,
    deviceCount: 5,
    isBuiltin: true,
    patterns: ['MicroCell-NR-.+'],
  },
  {
    id: 'p-003',
    name: 'UPS',
    vendor: 'Baicells',
    tech: 'ups',
    radioModes: '',
    description: 'UPS 演示产品',
    paramModelId: 'pm-ups',
    indicatorDeviceType: 'UPS',
    indicatorPlatform: 'ups-default',
    alarmNeType: 'UPS',
    enableFiletype11: false,
    deviceAttrsOverride: { data_type: true, access: true, min_value: false, max_value: false, change_applies: false },
    enableUnknownAlarm: true,
    deviceCount: 0,
    isBuiltin: true,
    patterns: ['UPS.*'],
  },
];

export const mockPatterns: Record<string, ProductPattern[]> = {
  'p-001': [
    { id: 'pat-001', productId: 'p-001', productClass: 'PicoCell-LTE-.+', sortOrder: 1, isActive: true, source: 'builtin', deletable: false },
    { id: 'pat-002', productId: 'p-001', productClass: 'CombaPico.*', sortOrder: 1000001, isActive: true, source: 'custom', deletable: true },
  ],
  'p-002': [
    { id: 'pat-003', productId: 'p-002', productClass: 'MicroCell-NR-.+', sortOrder: 1, isActive: true, source: 'builtin', deletable: false },
  ],
  'p-003': [
    { id: 'pat-ups-001', productId: 'p-003', productClass: 'UPS.*', sortOrder: 1, isActive: true, source: 'builtin', deletable: false },
  ],
};

export const mockMatchOrder: MatchOrderRow[] = [
  { patternId: 'pat-001', productId: 'p-001', productName: 'PicoCell-LTE-V2', productClass: 'PicoCell-LTE-.+', sortOrder: 1, isActive: true, source: 'builtin' },
  { patternId: 'pat-002', productId: 'p-001', productName: 'PicoCell-LTE-V2', productClass: 'CombaPico.*', sortOrder: 1000001, isActive: true, source: 'custom' },
  { patternId: 'pat-003', productId: 'p-002', productName: 'MicroCell-NR-Standard', productClass: 'MicroCell-NR-.+', sortOrder: 3, isActive: true, source: 'builtin' },
  { patternId: 'pat-ups-001', productId: 'p-003', productName: 'UPS', productClass: 'UPS.*', sortOrder: 4, isActive: true, source: 'builtin' },
];

export const mockOrphans: OrphanDevice[] = [
  {
    id: 'd-orph-1',
    serialNumber: 'SN-ORPH-1001',
    oui: '00ABCD',
    productClass: 'UnknownProduct-X',
    carrier: 'cmcc',
    manufacturer: 'UnknownVendor',
    lastInformAt: '2026-05-08T03:00:00Z',
  },
];
