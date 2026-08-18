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
  // #268 e2e:手工新增定义(loadedFrom 为空) — 一级表出现独立"手工新增"行,
  // 镜像测试环境 ENB 手工新增告警的场景。
  {
    id: 'ad-1004',
    identifier: '101099',
    neType: 'eNodeB',
    cnName: '手工新增测试告警',
    enName: 'Manual Added Test Alarm',
    severityCode: 1,
    severityName: 'Critical',
    eventType: 'device',
    cnProbableCause: '手工新增',
    enProbableCause: 'manual added',
    isShow: true,
    isUnknown: false,
    loadedFrom: '',
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
