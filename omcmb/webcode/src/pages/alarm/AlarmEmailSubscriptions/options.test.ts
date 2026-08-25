import { describe, expect, it } from 'vitest';
import type { AlarmEmailSubscription } from '@core/services/api/alarmApi';
import { ALARM_EMAIL_EVENT_TYPES, buildAlarmSourceOptions } from './options';

function subscription(alarmSources: string[]): AlarmEmailSubscription {
  return {
    id: 'subscription-1',
    name: 'test',
    description: '',
    enabled: true,
    interval_minutes: 10,
    tolerance_minutes: 0,
    recipients: [],
    include_default_recipients: true,
    alarm_identifiers: [],
    severities: [],
    alarm_sources: alarmSources,
    event_types: [],
    device_ids: [],
    device_group_ids: [],
  };
}

describe('AlarmEmailSubscriptions options', () => {
  it('使用项目标准事件类型枚举', () => {
    expect(ALARM_EMAIL_EVENT_TYPES.map((item) => item.value)).toEqual([
      '30000',
      '30001',
      '30002',
      '30003',
      '30004',
      '30006',
    ]);
  });

  it('告警来源合并系统产品类型与历史订阅值并去重', () => {
    expect(buildAlarmSourceOptions(
      ['LTE-Pico', 'GNB-Indoor', ''],
      [subscription(['legacy-source', ' LTE-Pico '])],
    )).toEqual([
      { value: 'LTE-Pico', label: 'LTE-Pico' },
      { value: 'GNB-Indoor', label: 'GNB-Indoor' },
      { value: 'legacy-source', label: 'legacy-source' },
    ]);
  });
});
