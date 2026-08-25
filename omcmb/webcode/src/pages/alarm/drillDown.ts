import dayjs, { type Dayjs } from 'dayjs';
import type { AlarmFilter } from '@core/types/alarm';
import type { AlarmSeverity } from '@core/types/common';

/**
 * #236 告警统计页 Top/分布钻取 → 当前告警列表的「建链 / 解析」纯逻辑。
 * 抽出便于单测，组件复用，保证被测即线上逻辑。
 * 注意字段名差异：URL 用 deviceSN / startTime / endTime，列表/后端用 deviceSn / timeRange。
 */

export const VALID_SEVERITIES: AlarmSeverity[] = ['critical', 'major', 'minor', 'warning'];
const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

export type DrillDownTimeRange = '7days' | '30days' | 'custom';

export interface BuildDrillDownInput {
  severity?: string;
  deviceSN?: string;
  timeRange: DrillDownTimeRange;
  customStartDate?: Dayjs | null;
  customEndDate?: Dayjs | null;
  /** 注入「现在」便于测试；不传则取 dayjs()。 */
  now?: Dayjs;
}

export function parseAlarmId(search: URLSearchParams): string | undefined {
  const alarmId = search.get('alarmId')?.trim();
  return alarmId && UUID_PATTERN.test(alarmId) ? alarmId : undefined;
}

export function withoutAlarmId(search: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(search);
  next.delete('alarmId');
  return next;
}

/** 统计页把当前筛选（级别/设备/时间范围）序列化成钻取 URL 的查询参数。 */
export function buildDrillDownSearch(input: BuildDrillDownInput): URLSearchParams {
  const params = new URLSearchParams();
  if (input.severity) params.set('severity', input.severity);
  if (input.deviceSN) params.set('deviceSN', input.deviceSN);

  let start: Dayjs | null = null;
  let end: Dayjs | null = null;
  if (input.timeRange === 'custom' && input.customStartDate && input.customEndDate) {
    start = input.customStartDate;
    end = input.customEndDate;
  } else if (input.timeRange === '7days') {
    end = input.now ?? dayjs();
    start = end.subtract(6, 'day').startOf('day');
  } else if (input.timeRange === '30days') {
    end = input.now ?? dayjs();
    start = end.subtract(29, 'day').startOf('day');
  }
  if (start && end) {
    params.set('startTime', start.toISOString());
    params.set('endTime', end.toISOString());
  }
  return params;
}

/** 当前告警列表挂载时从钻取 URL 解析出列表筛选条件与表单初始值。 */
export function parseDrillDownParams(search: URLSearchParams): {
  filter: AlarmFilter;
  formValues: Record<string, unknown>;
} {
  const filter: AlarmFilter = {};
  const formValues: Record<string, unknown> = {};

  const deviceSN = search.get('deviceSN') ?? search.get('deviceSn');
  if (deviceSN) {
    filter.deviceSn = deviceSN;
    formValues.deviceSn = deviceSN;
  }

  const severity = search.get('severity');
  if (severity && VALID_SEVERITIES.includes(severity as AlarmSeverity)) {
    filter.severity = [severity as AlarmSeverity];
    formValues.severity = [severity];
  }

  const eventType = search.get('eventType');
  if (eventType) {
    filter.eventType = eventType as AlarmFilter['eventType'];
    formValues.eventType = eventType;
  }

  const startTime = search.get('startTime');
  const endTime = search.get('endTime');
  if (startTime && endTime) {
    filter.timeRange = [startTime, endTime];
    formValues.timeRange = [startTime, endTime];
  }

  return { filter, formValues };
}
