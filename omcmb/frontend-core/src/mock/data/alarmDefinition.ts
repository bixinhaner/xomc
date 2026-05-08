import type { AlarmDefinition, AlarmSeverityLevel, UnknownAlarmStat } from '../../types/alarmDefinition';

export const mockAlarmSeverityLevels: AlarmSeverityLevel[] = [
  { id: 'sev-1', code: 1, cnName: '严重', enName: 'Critical', colorHex: '#FF4D4F' },
  { id: 'sev-2', code: 2, cnName: '主要', enName: 'Major', colorHex: '#FA8C16' },
  { id: 'sev-3', code: 3, cnName: '次要', enName: 'Minor', colorHex: '#FAAD14' },
  { id: 'sev-4', code: 4, cnName: '警告', enName: 'Warning', colorHex: '#1890FF' },
];

export const mockAlarmDefinitions: AlarmDefinition[] = [
  {
    id: 'ad-1001',
    identifier: '101001',
    neType: 'eNodeB',
    cnName: 'CPU 利用率过高',
    enName: 'CPU Usage Excessive',
    severityCode: 2,
    severityName: 'Major',
    eventType: 'qualityOfService',
    cnProbableCause: 'CPU 占用持续高于 90%',
    enProbableCause: 'CPU usage above 90% sustained',
    cnSuggestion: '检查业务负荷与异常进程',
    enSuggestion: 'Check workload and abnormal processes',
    isShow: true,
    isUnknown: false,
  },
  {
    id: 'ad-1002',
    identifier: '101002',
    neType: 'eNodeB',
    cnName: '小区不可用',
    enName: 'Cell Unavailable',
    severityCode: 1,
    severityName: 'Critical',
    eventType: 'communication',
    cnProbableCause: '射频链路故障',
    enProbableCause: 'RF link failure',
    cnSuggestion: '检查射频前端与同步状态',
    enSuggestion: 'Inspect RF frontend and sync status',
    isShow: true,
    isUnknown: false,
  },
  {
    id: 'ad-1003',
    identifier: 'UNKNOWN-XYZ',
    neType: 'gNodeB',
    cnName: '未识别 Fallback 告警',
    enName: 'Unknown Fallback Alarm',
    severityCode: 4,
    severityName: 'Warning',
    eventType: 'other',
    isShow: true,
    isUnknown: true,
  },
];

export const mockUnknownStats: UnknownAlarmStat[] = [
  {
    productId: 'p-001',
    productName: 'PicoCell-LTE-V2',
    identifier: 'UNKNOWN-XYZ',
    count: 3,
    lastSeenAt: '2026-05-08T01:30:00Z',
  },
];
