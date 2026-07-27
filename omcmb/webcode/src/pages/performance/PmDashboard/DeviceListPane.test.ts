import { afterEach, describe, expect, it, vi } from 'vitest';
import dayjs from 'dayjs';
import { PM_QUERY_SELECTION_LIMIT } from '@/constants/pmQueryLimits';
import type { DashboardExportSelection } from '@core/utils/kpiExportParams';
import {
  buildDeviceViewExportInput,
  isDeviceViewDeviceSelectionOverLimit,
  isDeviceViewMetricSelectionOverLimit,
} from './DeviceListPane';
import {
  actualRangeFromMeta,
  buildDeviceViewRequestTimeWindow,
  defaultRangeForGranularity,
  toDeviceViewRequestRFC3339,
} from './deviceListPaneTimeUtils';

const selection: DashboardExportSelection = {
  technology: 'lte',
  deviceSns: ['ISSUE95-LTE-01'],
  metricPaths: ['K001'],
  granularity: '15min',
  startTime: '2026-07-16T07:00:00.000Z',
  endTime: '2026-07-16T08:00:00.000Z',
};

describe('buildDeviceViewExportInput', () => {
  it('设备性能查看导出使用独立 sourceType 和中文任务名前缀', () => {
    const input = buildDeviceViewExportInput(
      selection,
      'KPI导出',
      '设备性能查看',
      new Date(2026, 6, 16, 18, 40, 5),
    );

    expect(input.sourceType).toBe('device_view');
    expect(input.taskName).toBe('KPI导出_设备性能查看_20260716_184005');
    expect(input.taskName).not.toContain('仪表盘');
  });

  it('英文界面下使用英文设备性能查看任务名前缀', () => {
    const input = buildDeviceViewExportInput(
      selection,
      'KPI Export',
      'Device Performance View',
      new Date(2026, 6, 16, 18, 40, 5),
    );

    expect(input.sourceType).toBe('device_view');
    expect(input.taskName).toBe('KPI_Export_Device_Performance_View_20260716_184005');
  });
});

describe('isDeviceViewMetricSelectionOverLimit', () => {
  it('设备性能查看沿用 PM 查询设备数量上限', () => {
    expect(isDeviceViewDeviceSelectionOverLimit(
      Array.from({ length: PM_QUERY_SELECTION_LIMIT }, (_, index) => `SN-${index + 1}`),
    )).toBe(false);
    expect(isDeviceViewDeviceSelectionOverLimit(
      Array.from({ length: PM_QUERY_SELECTION_LIMIT + 1 }, (_, index) => `SN-${index + 1}`),
    )).toBe(true);
  });

  it('设备性能查看沿用 PM 查询指标数量上限', () => {
    expect(isDeviceViewMetricSelectionOverLimit(
      Array.from({ length: PM_QUERY_SELECTION_LIMIT }, (_, index) => `K${index + 1}`),
    )).toBe(false);
    expect(isDeviceViewMetricSelectionOverLimit(
      Array.from({ length: PM_QUERY_SELECTION_LIMIT + 1 }, (_, index) => `K${index + 1}`),
    )).toBe(true);
  });
});

describe('defaultRangeForGranularity', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it('系统时区为 UTC 时默认结束时间使用 UTC 当前钟面', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-24T02:41:30.456Z'));

    const [, end] = defaultRangeForGranularity('15min', 'UTC');

    expect(end.format('YYYY-MM-DD HH:mm:ss.SSS')).toBe('2026-07-24 02:41:30.000');
  });

  it('非法系统时区按 UTC 回退，不使用浏览器本地偏移', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-24T02:41:30.456Z'));

    const [, end] = defaultRangeForGranularity('15min', 'Bad/Zone');

    expect(end.format('YYYY-MM-DD HH:mm:ss.SSS')).toBe('2026-07-24 02:41:30.000');
  });

  it('按粒度生成默认窗口长度', () => {
    const now = dayjs('2026-07-24 02:41:30.456');

    expect(defaultRangeForGranularity('15min', 'UTC', now).map((d) => d.format('YYYY-MM-DD HH:mm:ss')))
      .toEqual(['2026-07-23 02:41:30', '2026-07-24 02:41:30']);
    expect(defaultRangeForGranularity('hourly', 'UTC', now).map((d) => d.format('YYYY-MM-DD HH:mm:ss')))
      .toEqual(['2026-07-23 02:41:30', '2026-07-24 02:41:30']);
    expect(defaultRangeForGranularity('daily', 'UTC', now).map((d) => d.format('YYYY-MM-DD HH:mm:ss')))
      .toEqual(['2026-07-17 02:41:30', '2026-07-24 02:41:30']);
    expect(defaultRangeForGranularity('weekly', 'UTC', now).map((d) => d.format('YYYY-MM-DD HH:mm:ss')))
      .toEqual(['2026-06-24 02:41:30', '2026-07-24 02:41:30']);
    expect(defaultRangeForGranularity('monthly', 'UTC', now).map((d) => d.format('YYYY-MM-DD HH:mm:ss')))
      .toEqual(['2026-01-24 02:41:30', '2026-07-24 02:41:30']);
  });
});

describe('toDeviceViewRequestRFC3339', () => {
  it('设备性能查看提交给后端的时间按系统时区附加偏移并去掉隐藏毫秒', () => {
    expect(toDeviceViewRequestRFC3339(dayjs('2026-07-21 00:00:00.300'), 'UTC'))
      .toBe('2026-07-21T00:00:00Z');
    expect(toDeviceViewRequestRFC3339(dayjs('2026-07-21 01:00:00.300'), 'Asia/Shanghai'))
      .toBe('2026-07-21T01:00:00+08:00');
  });

  it('非法系统时区提交时间按 UTC 钟面兜底', () => {
    expect(toDeviceViewRequestRFC3339(dayjs('2026-07-21 01:00:00.300'), 'Bad/Zone'))
      .toBe('2026-07-21T01:00:00Z');
  });

  it('上一周期窗口与当前窗口使用同一系统时区提交口径', () => {
    const window = buildDeviceViewRequestTimeWindow(
      [dayjs('2026-07-24 00:00:00.300'), dayjs('2026-07-24 03:00:00.300')],
      'UTC',
    );

    expect(window).toEqual({
      startTime: '2026-07-24T00:00:00Z',
      endTime: '2026-07-24T03:00:00Z',
      prevStartTime: '2026-07-23T21:00:00Z',
      prevEndTime: '2026-07-24T00:00:00Z',
    });
  });

  it('非 UTC 系统时区下查询和上一周期窗口使用相同 RFC3339 偏移', () => {
    const window = buildDeviceViewRequestTimeWindow(
      [dayjs('2026-07-24 10:00:00.300'), dayjs('2026-07-24 13:00:00.300')],
      'Asia/Shanghai',
    );

    expect(window).toEqual({
      startTime: '2026-07-24T10:00:00+08:00',
      endTime: '2026-07-24T13:00:00+08:00',
      prevStartTime: '2026-07-24T07:00:00+08:00',
      prevEndTime: '2026-07-24T10:00:00+08:00',
    });
  });

  it('接口 actual range 保留系统钟面后再用于上一周期计算', () => {
    const actualRange = actualRangeFromMeta({
      actualStartTime: '2026-07-24T02:00:00Z',
      actualEndTime: '2026-07-24T05:00:00Z',
    });

    expect(actualRange?.map((d) => d.format('YYYY-MM-DD HH:mm:ss'))).toEqual([
      '2026-07-24 02:00:00',
      '2026-07-24 05:00:00',
    ]);
    expect(buildDeviceViewRequestTimeWindow(actualRange!, 'UTC')).toEqual({
      startTime: '2026-07-24T02:00:00Z',
      endTime: '2026-07-24T05:00:00Z',
      prevStartTime: '2026-07-23T23:00:00Z',
      prevEndTime: '2026-07-24T02:00:00Z',
    });
  });

  it('接口 actual range 带非 UTC 偏移时保留后端系统钟面', () => {
    const actualRange = actualRangeFromMeta({
      actualStartTime: '2026-07-24T10:00:00+08:00',
      actualEndTime: '2026-07-24T13:00:00+08:00',
    });

    expect(actualRange?.map((d) => d.format('YYYY-MM-DD HH:mm:ss'))).toEqual([
      '2026-07-24 10:00:00',
      '2026-07-24 13:00:00',
    ]);
    expect(buildDeviceViewRequestTimeWindow(actualRange!, 'Asia/Shanghai')).toEqual({
      startTime: '2026-07-24T10:00:00+08:00',
      endTime: '2026-07-24T13:00:00+08:00',
      prevStartTime: '2026-07-24T07:00:00+08:00',
      prevEndTime: '2026-07-24T10:00:00+08:00',
    });
  });
});
