import type { IndicatorInfo, IndicatorGroup, PlatformFormula, IndicatorUnit } from '../../types/indicatorLibrary';

export const mockIndicators: Record<string, IndicatorInfo[]> = {
  ENB: [
    { id: 'ENB-1001', name: 'RRC建立成功率', cnName: 'RRC建立成功率', enName: 'RRC Setup Success Rate', deviceType: 'ENB', counterType: 'rate', indicatorLevel: 'cell', unit: '%', isCounter: false, isEnabled: true, operatorCode: 'cmcc', groupId: 'g-enb-1', groupName: '接入类' },
    { id: 'ENB-1002', name: 'ERAB建立成功率', cnName: 'ERAB建立成功率', enName: 'ERAB Setup Success Rate', deviceType: 'ENB', counterType: 'rate', indicatorLevel: 'cell', unit: '%', isCounter: false, isEnabled: true, operatorCode: 'cmcc', groupId: 'g-enb-1', groupName: '接入类' },
  ],
  GSM: [
    { id: 'GSM-2001', name: '掉话率', cnName: '掉话率', enName: 'Call Drop Rate', deviceType: 'GSM', counterType: 'rate', unit: '%', isCounter: false, isEnabled: true, operatorCode: 'cmcc', groupId: 'g-gsm-1', groupName: '保持类' },
  ],
  GNB: [
    { id: 'GNB-3001', name: 'NR-RRC建立成功率', cnName: 'NR-RRC建立成功率', enName: 'NR RRC Setup Success Rate', deviceType: 'GNB', counterType: 'rate', unit: '%', isCounter: false, isEnabled: true, operatorCode: 'cmcc', groupId: 'g-gnb-1', groupName: '接入类' },
  ],
};

export const mockGroups: Record<string, IndicatorGroup[]> = {
  ENB: [
    { id: 'g-enb-1', name: '接入类', deviceType: 'ENB', operatorCode: 'cmcc' },
    { id: 'g-enb-2', name: '保持类', deviceType: 'ENB', operatorCode: 'cmcc' },
  ],
  GSM: [{ id: 'g-gsm-1', name: '保持类', deviceType: 'GSM', operatorCode: 'cmcc' }],
  GNB: [{ id: 'g-gnb-1', name: '接入类', deviceType: 'GNB', operatorCode: 'cmcc' }],
};

export const mockFormulas: Record<string, PlatformFormula[]> = {
  'ENB-1001': [
    { platformName: 'enb-default', indicatorId: 'ENB-1001', formula: 'C1 / (C1 + C2) * 100' },
    { platformName: 'enb-comba', indicatorId: 'ENB-1001', formula: 'CombaC1 / (CombaC1 + CombaC2) * 100' },
  ],
};

export const mockEnabledIndicators: Record<string, Record<string, string[]>> = {
  ENB: {
    cmcc: ['ENB-1001', 'ENB-1002'],
    default: ['ENB-1001', 'ENB-1002'],
  },
  GSM: { cmcc: ['GSM-2001'] },
  GNB: { cmcc: ['GNB-3001'] },
};

export const mockUnits: IndicatorUnit[] = [
  { id: 'percent', enName: '%', cnName: '百分比' },
  { id: 'count', enName: 'Count', cnName: '次数' },
  { id: 'mbps', enName: 'Mbps', cnName: '兆比特每秒' },
];
