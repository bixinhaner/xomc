import type { AlarmEmailSubscription } from '@core/services/api/alarmApi';

export const ALARM_EMAIL_EVENT_TYPES = [
  { value: '30000', labelKey: 'alarm.eventType.30000' },
  { value: '30001', labelKey: 'alarm.eventType.30001' },
  { value: '30002', labelKey: 'alarm.eventType.30002' },
  { value: '30003', labelKey: 'alarm.eventType.30003' },
  { value: '30004', labelKey: 'alarm.eventType.30004' },
  { value: '30006', labelKey: 'alarm.eventType.30006' },
] as const;

/**
 * 告警来源由南向告警写入设备 product_class，不是设备制式枚举。
 * 选项以系统产品类型接口为主，并入已有订阅值，避免编辑旧订阅时丢值。
 */
export function buildAlarmSourceOptions(
  productClasses: string[] = [],
  subscriptions: AlarmEmailSubscription[] = [],
): Array<{ value: string; label: string }> {
  const values = new Set<string>();
  const add = (value: string) => {
    const normalized = value.trim();
    if (normalized) values.add(normalized);
  };

  productClasses.forEach(add);
  subscriptions.forEach((subscription) => subscription.alarm_sources.forEach(add));

  return Array.from(values, (value) => ({ value, label: value }));
}
